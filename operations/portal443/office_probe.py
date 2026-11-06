#!/usr/bin/env python3
"""Live DB eligibility + worker/firewalld/nft/HTTPS, server-IP fixture only.

No real passwords, registrations or operator-IP grants are used. Expiry is
accelerated only for the fixture; this is not a 24-hour wall-clock test.
"""
import fcntl
import json
import sys
import time
sys.path.insert(0, '/opt/alma-portal-443')
from common import BASE, grant_key, atomic_json
from phone_policy import office_supervisor
from vicidial_auth import _connect
from worker_probe import IP, PORTS, rule, command, queue, run_worker, guarded, status
from probe import request


def mutate(callback):
    with open(BASE/'worker.lock', 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        state = json.loads((BASE/'state.json').read_text())
        callback(state)
        atomic_json(BASE/'state.json', state, 0o600)


def main():
    c = _connect()
    try:
        with c.cursor() as cursor:
            cursor.execute("SELECT user FROM vicidial_users WHERE active='Y' AND user_level BETWEEN 1 AND 7 AND api_only_user='0' AND force_change_password='N' LIMIT 1")
            agent = cursor.fetchone()['user']
            cursor.execute("SELECT user FROM vicidial_users WHERE active='Y' AND user_level IN (8,9) AND modify_users='1' AND api_only_user='0' AND force_change_password='N'")
            supervisors = cursor.fetchall()
        supervisor = next(v['user'] for v in supervisors if office_supervisor(v['user']))
    finally:
        c.close()
    initial = json.loads((BASE/'state.json').read_text())
    assert not any(v['ip'] == IP and v['expires_at'] > time.time() for v in initial.get('leases', {}).values())
    assert not (BASE/'web'/IP).exists()
    for port in PORTS:
        assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 1
    assert command('systemctl', 'stop', 'alma-portal-443-apply.timer').returncode == 0
    def submit(user, role, kind='grant', grant_type='personal', at=None):
        queue(dict(ip=IP, user=user, role=role, kind=kind, grant_type=grant_type,
                   at=int(time.time()) if at is None else at))
        run_worker()
        return grant_key(user, IP, grant_type)
    office = grant_key(supervisor, IP, 'office')
    personal = grant_key(agent, IP)
    try:
        denied = submit(agent, 'agent', grant_type='office')
        assert status(denied)['state'] == 'failed' and not guarded()
        denied = submit('nonexistent-test-account', 'agent')
        assert status(denied)['state'] == 'failed' and not guarded()
        print('PASS live agent office rejection; nonexistent user rejection')
        submit(supervisor, 'web', grant_type='office')
        current = status(office)
        assert current['state'] == 'active' and current['grant_type'] == 'office'
        assert current['expires_at'] - current['issued_at'] == 86400
        assert current['services'] == ['https', 'sip', 'rtp'] and guarded()
        for port in PORTS:
            assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 0
        # No portal cookie/credentials: IP authorization reaches native logins.
        assert request('/agc/vicidial.php')[0] == 200
        code, headers, _ = request('/vicidial/admin.php')
        assert code == 401 and 'WWW-Authenticate' in headers
        print('PASS office24h; aggregate timed firewall/nft grant; cookie-free peer reaches native logins; native admin authentication still required')
        submit(supervisor, 'web', grant_type='office')
        assert status(office)['expires_at'] == current['expires_at']
        print('PASS repeat office authorization retains expiry')
        submit(agent, 'agent')
        assert status(personal)['expires_at'] - status(personal)['issued_at'] == 43200
        submit(agent, 'agent', 'revoke_user')
        assert status(personal)['state'] == 'revoked' and status(office)['state'] == 'active' and guarded()
        print('PASS personal12h and revoke preserves office phone/web access')
        time.sleep(2)
        submit(agent, 'agent')
        submit(supervisor, 'web', 'revoke_user', 'office')
        assert status(office)['state'] == 'revoked' and status(personal)['state'] == 'active' and guarded()
        print('PASS office revoke preserves independent personal phone/web access')
        revoked_at = int(time.time())
        submit(agent, 'agent', 'revoke_user', at=revoked_at)
        submit(agent, 'agent', at=revoked_at)
        assert status(personal)['state'] in ('revoked', 'failed') and not guarded()
        assert not (BASE/'web'/IP).exists()
        for port in PORTS:
            assert command('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(IP, port)).returncode == 1
        state = json.loads((BASE/'state.json').read_text())
        assert int(time.time()) < state['rtp_grace'][IP] <= int(time.time()) + 120
        print('PASS last revoke closes fresh SIP/web, preserves bounded RTP grace; stale queue cannot reactivate')
        time.sleep(2)
        submit(supervisor, 'web', grant_type='office')
        expiry = int(time.time()) - 1
        mutate(lambda s:s['leases'][office].update(expires_at=expiry))
        submit(supervisor, 'web', grant_type='office', at=expiry-1)
        assert status(office)['state'] == 'expired' and not guarded()
        assert request('/vicidial/admin.php')[0] == 403
        print('PASS accelerated office expiry blocks firewall/nft/web; pre-expiry queue cannot extend')
        submit(supervisor, 'web', grant_type='office')
        assert status(office)['state'] == 'active' and guarded()
        print('PASS fresh authorization after expiry succeeds')
    finally:
        # Scope cleanup to fixture IP only; leave unrelated grants/state intact.
        def clean(state):
            for lease in state.get('leases', {}).values():
                if lease['ip'] == IP:
                    lease.update(expires_at=min(lease['expires_at'], int(time.time())), revoked=True)
        mutate(clean)
        run_worker()
        command('systemctl', 'start', 'alma-portal-443-apply.timer')
    final = json.loads((BASE/'state.json').read_text())
    for key, before in initial.get('leases', {}).items():
        if before['ip'] != IP:
            assert final['leases'][key]['expires_at'] == before['expires_at']
            assert final['leases'][key].get('revoked') == before.get('revoked')
    assert not guarded() and not (BASE/'web'/IP).exists()
    print('PASS fixture cleaned; unrelated personal grants and expiries preserved')


if __name__ == '__main__':
    main()
