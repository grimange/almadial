"""Real worker/DB-eligibility/firewalld probe, scoped to the server's own source IP.

No password lookup or user/schema mutation. This does not test password login.
Run as root on the authorized VPS after activation, through SSH only.
"""
import grp
import json
import os
from pathlib import Path
import pwd
import secrets
import subprocess
import sys
import time

sys.path.insert(0, '/opt/alma-portal-443')
from common import BASE, QUEUE, grant_key
from apply_requests import rule, PORTS
from vicidial_auth import _connect
from sip_guard import TABLE
from phone_policy import supervisor_phone

IP = '2.29.153.161'
uid = pwd.getpwnam('alma-portal').pw_uid
gid = grp.getgrnam('alma-portal').gr_gid


def command(*args):
    return subprocess.run(list(args), capture_output=True, timeout=90, check=False)


def queue(data):
    key = secrets.token_hex(16)
    path = QUEUE / (key + '.request')
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as stream:
        os.fchown(stream.fileno(), uid, gid)
        json.dump(data, stream)
    return key


def run_worker():
    assert command('systemctl', 'start', 'alma-portal-443-apply.service').returncode == 0


def guarded():
    result = command('nft', '-j', 'get', 'element', 'inet', TABLE, 'agents4', '{', IP, '}')
    return result.returncode == 0


def status(key):
    return json.loads((BASE / 'status' / key).read_text())


def main():
    c = _connect()
    try:
        with c.cursor() as cursor:
            cursor.execute("SELECT user FROM vicidial_users WHERE active='Y' AND user_level BETWEEN 1 AND 7 AND api_only_user='0' AND force_change_password='N' LIMIT 1")
            user = cursor.fetchone()['user']
            cursor.execute("SELECT user FROM vicidial_users WHERE active='Y' AND user_level > 7 AND api_only_user='0' AND force_change_password='N' LIMIT 1")
            supervisor = cursor.fetchone()['user']
    finally:
        c.close()
    supervisor_has_phone = supervisor_phone(supervisor)
    assert not (BASE / 'web' / IP).exists(), 'Refuse to disturb existing web grant'
    for port in PORTS:
        assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 1
    data = {'ip': IP, 'user': user, 'role': 'agent', 'at': int(time.time()), 'kind': 'grant'}
    queue(data)
    key = grant_key(user, IP)
    grants = [(key, data)]
    try:
        run_worker()
        current = status(key)
        assert current['state'] == 'active'
        assert guarded()
        assert 43100 < current['expires_at'] - time.time() <= 43200
        assert (BASE/'web'/IP).stat().st_uid == 0
        for port in PORTS:
            assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 0
        print('PASS live queue -> account recheck -> timed SIP/RTP rules -> active status/web grant')
        queue({**data, 'at': int(time.time())})
        second = key
        grants.append((second, data))
        web_data = {**data, 'user': supervisor, 'role': 'web', 'at': int(time.time())}
        queue(web_data)
        web_key = grant_key(supervisor, IP)
        grants.append((web_key, web_data))
        run_worker()
        assert status(second)['state'] == status(web_key)['state'] == 'active'
        assert status(second)['expires_at'] == current['expires_at']
        print('PASS repeated personal request is idempotent and retains expiry')
        assert guarded()
        queue({**data, 'kind': 'revoke', 'target': key, 'at': int(time.time())})
        run_worker()
        assert status(key)['state'] == 'revoked'
        assert status(second)['state'] == 'revoked'
        assert status(web_key)['state'] == 'active'
        for port in PORTS:
            assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == (0 if supervisor_has_phone else 1)
        print('PASS shared IP: revoking personal access preserves another issuer supervisor grant')
        revoke_at = int(time.time())
        queue({**data, 'kind': 'revoke_user', 'at': revoke_at})
        run_worker()
        assert status(second)['state'] == 'revoked'
        assert status(web_key)['state'] == 'active'
        assert guarded() == supervisor_has_phone
        assert (BASE/'web'/IP).exists()
        for port in PORTS:
            assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == (0 if supervisor_has_phone else 1)
        print('PASS shared IP: supervisor retains web; SIP matches native monitoring permission: ' + str(supervisor_has_phone))
        queue({**data, 'at': revoke_at})
        stale = key
        run_worker()
        assert status(stale)['state'] in ('failed', 'revoked')
        print('PASS account-scoped revoke clears old sessions; stale queued grant cannot reactivate access')
    finally:
        for target, grant_data in grants:
            queue({**grant_data, 'kind': 'revoke', 'target': target, 'at': int(time.time())})
        run_worker()
        # Fallback cleanup is limited to the two test-specific owned rules.
        for port in PORTS:
            if command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 0:
                command('firewall-cmd', '--zone=public', '--remove-rich-rule=' + rule(IP, port))
    assert all(status(target)['state'] == 'revoked' for target, _ in grants)
    assert not (BASE/'web'/IP).exists()
    assert not guarded()
    for port in PORTS:
        assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 1
    print('PASS live per-lease revoke removes web and owned SIP/RTP rules; fixture cleaned')
    queue({**data, 'user': 'nonexistent-test-account', 'at': int(time.time())})
    key = grant_key('nonexistent-test-account', IP)
    run_worker()
    assert status(key)['state'] == 'failed'
    assert not (BASE/'web'/IP).exists()
    print('PASS live ineligible account request fails closed')


if __name__ == '__main__':
    main()
