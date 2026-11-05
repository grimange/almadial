#!/usr/bin/env python3
"""Root timer: revalidate accounts, reconcile owned firewall leases, publish web TTLs."""
import fcntl
import grp
import ipaddress
import json
import os
import pwd
import re
import stat
import subprocess
import time

from common import BASE, QUEUE, LIFETIME, HEARTBEAT, atomic_json, public_ip
from vicidial_auth import current_role, USER_PATTERN

ID = re.compile(r'^[a-f0-9]{32}$')
PORTS = ('5061', '10000-20000')


def rule(ip, port):
    # Positive priority keeps Fail2Ban rejects ahead of these accepts.
    return f'rule priority="10" family="ipv4" source address="{ip}/32" port port="{port}" protocol="udp" accept'


def firewall(*args):
    r = subprocess.run(['/usr/bin/firewall-cmd', '--zone=public', *args],
                       capture_output=True, text=True, timeout=8, check=False)
    return r.returncode


def aggregate(leases, now):
    web, sip = {}, {}
    for lease in leases.values():
        if lease['expires_at'] <= now or not lease.get('eligible'):
            continue
        ip = lease['ip']
        web[ip] = max(web.get(ip, 0), lease['expires_at'])
        if lease['role'] == 'agent':
            sip[ip] = max(sip.get(ip, 0), lease['expires_at'])
    return web, sip


def reconcile(sip, previous, now, command=firewall):
    """Touch only uniquely marked rules owned by this worker; return confirmed IPs."""
    confirmed, synced = set(), {}
    for ip in set(previous) | set(sip):
        expiry = sip.get(ip, 0)
        try:
            for port in PORTS:
                value = rule(ip, port)
                query = command('--query-rich-rule=' + value)
                if query not in (0, 1):
                    raise RuntimeError('Firewall query unavailable')
                if query == 0 and (not expiry or previous.get(ip) != expiry):
                    if command('--remove-rich-rule=' + value) != 0:
                        raise RuntimeError('Firewall revoke failed')
                    query = 1
                if expiry > now and query == 1:
                    if command('--add-rich-rule=' + value, '--timeout=' + str(expiry - now)) != 0:
                        raise RuntimeError('Firewall grant failed')
                if expiry > now and command('--query-rich-rule=' + value) != 0:
                    raise RuntimeError('Firewall grant not confirmed')
            if expiry > now:
                synced[ip] = expiry
                confirmed.add(ip)
        except (RuntimeError, subprocess.SubprocessError):
            # Keep ownership for retry even if only one of the two rules was added.
            synced[ip] = previous.get(ip, expiry)
            print('FIREWALL_RECONCILE_FAILED', flush=True)
    return confirmed, synced


def process(state, now, role_check=current_role, command=firewall):
    portal_uid = pwd.getpwnam('alma-portal').pw_uid
    portal_gid = grp.getgrnam('alma-portal').gr_gid
    leases = state.setdefault('leases', {})
    for path in sorted(QUEUE.glob('*.request'))[:50]:
        request_id = path.stem
        data = None
        try:
            if not ID.fullmatch(request_id):
                raise ValueError('Invalid ID')
            fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
            with os.fdopen(fd) as stream:
                st = os.fstat(stream.fileno())
                if not stat.S_ISREG(st.st_mode) or st.st_uid != portal_uid or st.st_size > 2048 or st.st_mode & 0o077:
                    raise ValueError('Invalid request file')
                data = json.load(stream)
            ip = public_ip(data['ip'])
            user, role, at = data['user'], data['role'], data['at']
            if not isinstance(user, str) or not USER_PATTERN.fullmatch(user) or role not in ('agent', 'web'):
                raise ValueError('Invalid identity')
            if type(at) is not int or not 0 <= now - at <= 120:
                raise ValueError('Stale request')
            if data['kind'] == 'revoke':
                target = data['target']
                lease = leases.get(target)
                if lease and (lease['user'], lease['ip']) == (user, ip):
                    lease['expires_at'] = now
                    lease['revoked'] = True
            elif data['kind'] == 'grant' and role_check(user) == role:
                leases[request_id] = {'ip': ip, 'user': user, 'role': role,
                                      'expires_at': now + LIFETIME, 'eligible': True}
            else:
                raise ValueError('Account no longer eligible')
        except Exception:
            if ID.fullmatch(request_id) and isinstance(data, dict):
                atomic_json(BASE / 'status' / request_id,
                            {'ip': data.get('ip'), 'user': data.get('user'), 'state': 'failed'}, gid=portal_gid)
            print('REQUEST_REJECTED', flush=True)
        finally:
            path.unlink(missing_ok=True)

    # Account disable/role changes fail closed even for previously granted leases.
    roles = {}
    for lease in leases.values():
        user = lease['user']
        if user not in roles:
            try:
                roles[user] = role_check(user)
            except Exception:
                roles[user] = None
        lease['eligible'] = roles[user] == lease['role']
    web, sip = aggregate(leases, now)
    # Persist ownership BEFORE firewall mutation, so a crash cannot orphan a grant.
    old_synced = {**dict.fromkeys(state.get('owned', []), 0), **state.get('synced', {})}
    state['owned'] = list(set(old_synced) | set(sip))
    atomic_json(BASE / 'state.json', state, 0o600)
    confirmed, synced = reconcile(sip, old_synced, now, command)
    state['synced'] = synced
    state['owned'] = list(synced)
    for ip in list(web):
        # If agent rules fail, don't advertise that IP as active for web either.
        if ip in sip and ip not in confirmed:
            del web[ip]
    for path in (BASE / 'web').iterdir():
        if path.name not in web:
            path.unlink()
    for ip, expiry in web.items():
        # Short freshness cap makes a stopped/crashed worker fail closed at Apache.
        path = BASE / 'web' / ip
        tmp = path.with_name('.' + ip)
        fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o644)
        with os.fdopen(fd, 'w') as stream:
            os.fchmod(stream.fileno(), 0o644)
            stream.write(str(min(expiry, now + HEARTBEAT)))
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(tmp, path)
    for key, lease in list(leases.items()):
        status = 'active'
        if lease.get('revoked'):
            status = 'revoked'
        elif lease['expires_at'] <= now:
            status = 'expired'
        elif not lease['eligible'] or lease['ip'] not in web:
            status = 'failed'
        atomic_json(BASE / 'status' / key, {**lease, 'state': status, 'checked_until': now + HEARTBEAT}, gid=portal_gid)
        if now - lease['expires_at'] > 86400:
            del leases[key]
    for path in (BASE / 'status').iterdir():
        if path.stat().st_mtime < now - 86400 and path.name not in leases:
            path.unlink()
    atomic_json(BASE / 'state.json', state, 0o600)


def main():
    with open(BASE / 'worker.lock', 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        path = BASE / 'state.json'
        state = json.loads(path.read_text()) if path.exists() else {}
        process(state, int(time.time()))


if __name__ == '__main__':
    main()
