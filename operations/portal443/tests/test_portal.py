"""Mocked accounts/firewall; HTTP tests run the real Python handler on loopback."""
import contextlib
import http.client
import io
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import threading
import time
import types
import unittest
from unittest.mock import patch
from urllib.parse import urlencode

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
try:
    import pymysql
except ImportError:
    # Local mock suite doesn't need a DB driver; production reuses installed PyMySQL.
    sys.modules['pymysql'] = types.SimpleNamespace()
import bcrypt
import vicidial_auth as auth
import portal
import apply_requests as worker
from phone_policy import monitor_allowed
from common import public_ip, atomic_json, grant_key


def row(**values):
    return dict(user='mockagent', active='Y', user_level=1, api_only_user='0',
                force_change_password='N', **values)


class FakeDB:
    def __init__(self, user, settings):
        self.results = iter((user, settings))

    def __enter__(self):
        return self

    def __exit__(self, *args):
        pass

    def cursor(self):
        return self

    def execute(self, sql, params=None):
        self.result = next(self.results)

    def fetchone(self):
        return self.result

    def close(self):
        pass


class Authentication(unittest.TestCase):
    def test_policy(self):
        base = row()
        for changes, expected in (({}, 'agent'), ({'user_level': 8}, 'web'),
                                  ({'user_level': 9}, 'web'), ({'user_level': 7}, 'agent'),
                                  ({'active': 'N'}, None), ({'api_only_user': '1'}, None),
                                  ({'force_change_password': 'Y'}, None), ({'user_level': 0}, None)):
            with self.subTest(changes=changes):
                self.assertEqual(auth._role_from_row({**base, **changes}), expected)

    def test_plaintext_valid_invalid_disabled(self):
        user = {**row(), 'pass': 'fake-password-only'}
        settings = {'pass_hash_enabled': '0'}
        for password, active, expected in (('fake-password-only', 'Y', 'agent'),
                                            ('incorrect', 'Y', None), ('fake-password-only', 'N', None)):
            with patch.object(auth, '_connect', return_value=FakeDB({**user, 'active': active}, settings)):
                self.assertEqual(auth.authenticate('mockagent', password), expected)
        for level, active, expected in ((8, 'Y', 'web'), (9, 'Y', 'web'), (9, 'N', None)):
            with patch.object(auth, '_connect', return_value=FakeDB({**user, 'user_level': level, 'active': active}, settings)):
                self.assertEqual(auth.authenticate('mockagent', 'fake-password-only'), expected)

    def test_bcrypt_legacy_suffix(self):
        salt = bcrypt.gensalt(rounds=4, prefix=b'2a')
        digest = bcrypt.hashpw(b'fake-password-only', salt).decode()
        user = {**row(), 'pass_hash': digest[29:60]}
        settings = {'pass_hash_enabled': '1', 'pass_key': salt.decode()[7:29], 'pass_cost': 4}
        for password, expected in (('fake-password-only', 'agent'), ('incorrect', None)):
            with patch.object(auth, '_connect', return_value=FakeDB(user, settings)):
                self.assertEqual(auth.authenticate('mockagent', password), expected)


class PortalHTTP(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = portal.Server(('127.0.0.1', 0), portal.Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.base = Path(self.tmp.name)
        for name in ('status', 'web', 'queue'):
            (self.base / name).mkdir()
        self.patches = [patch.object(portal, 'BASE', self.base),
                        patch.object(portal, 'QUEUE', self.base / 'queue')]
        for p in self.patches:
            p.start()
        portal.SESSIONS.clear()
        portal.FAILURES.clear()
        self.cookie = ''

    def tearDown(self):
        for p in self.patches:
            p.stop()
        self.tmp.cleanup()

    def request(self, path='/portal/', method='GET', fields=None, extra=None):
        conn = http.client.HTTPConnection('127.0.0.1', self.server.server_port, timeout=5)
        headers = {'Host': portal.PUBLIC_IP, 'X-Alma-Source': '8.8.8.8'}
        if self.cookie:
            headers['Cookie'] = self.cookie
        if fields is not None:
            headers['Content-Type'] = 'application/x-www-form-urlencoded'
            headers['Origin'] = portal.ORIGIN
        headers.update(extra or {})
        conn.request(method, path, urlencode(fields) if fields else None, headers)
        response = conn.getresponse()
        body = response.read().decode(errors='replace')
        result = (response.status, dict(response.getheaders()), body)
        if response.getheader('Set-Cookie'):
            self.cookie = response.getheader('Set-Cookie').split(';')[0]
        conn.close()
        return result

    def login(self, role='agent'):
        code, headers, body = self.request()
        self.assertEqual(code, 200)
        csrf = re.search('name="csrf" value="([a-f0-9]+)"', body)[1]
        with patch.object(portal, 'authenticate', return_value=role):
            code, headers, _ = self.request('/portal/login', 'POST',
                                            {'csrf': csrf, 'username': 'mocksupervisor' if role == 'web' else 'mockagent', 'password': 'fake-password-only'})
        self.assertEqual(code, 303)
        return next(iter(portal.SESSIONS.values()))

    def grant(self, s):
        self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf']})[0], 303)
        self.assertTrue(any(json.loads(p.read_text())['kind'] == 'grant' for p in (self.base/'queue').iterdir()))

    def activate(self, s, **overrides):
        now = int(time.time())
        status = dict(ip=s['ip'], user=s['user'], role=s['role'], state='active',
                      expires_at=now+43200, checked_until=now+35, requested_at=s.get('queued_at', now))
        status.update(overrides)
        (self.base / 'status' / s['request']).write_text(json.dumps(status))
        (self.base / 'web' / s['ip']).write_text(str(now+35))

    def test_public_session_cookie_and_no_basic_challenge(self):
        code, headers, body = self.request()
        self.assertEqual(code, 200)
        for value in ('Secure', 'HttpOnly', 'SameSite=Strict', 'Path=/portal/'):
            self.assertIn(value, headers['Set-Cookie'])
        self.assertNotIn('WWW-Authenticate', headers)
        # no-referrer can null the Origin on real browser form POSTs.
        self.assertEqual(headers['Referrer-Policy'], 'same-origin')
        self.assertIn('VICIdial Secure Access', body)
        self.assertIn('alt="VICIdial"', body)
        self.assertIn('href="/portal/app.css"', body)
        self.assertIn('for="username"', body)
        self.assertIn('for="password"', body)
        self.assertIn('Skip to main content', body)
        self.assertNotIn('unsafe-inline', headers['Content-Security-Policy'])
        self.assertEqual(self.request('/portal/status')[0], 401)

    def test_fixed_public_assets_and_no_asset_path_expansion(self):
        logo = self.base/'logo.png'
        logo.write_bytes(b'PNG-test-fixture')
        with patch.object(portal, 'LOGO', logo):
            code, headers, body = self.request('/portal/logo.png')
        self.assertEqual((code, headers['Content-Type'], body), (200, 'image/png', 'PNG-test-fixture'))
        code, headers, body = self.request('/portal/app.css')
        self.assertEqual((code, headers['Content-Type']), (200, 'text/css; charset=utf-8'))
        self.assertIn('@media (max-width: 420px)', body)
        self.assertIn(':focus-visible', body)
        self.assertIn('#015b91', body)
        self.assertIn('#d9e6fe', body)
        self.assertNotEqual(self.request('/portal/logo.png/agc/vicidial.php')[0], 200)

    def test_client_fields_cannot_escalate_role_or_supply_destination(self):
        s = self.login('agent')
        code, _, _ = self.request('/portal/grant', 'POST',
                                  {'csrf': s['csrf'], 'role': 'web', 'destination': '/vicidial/admin.php'})
        self.assertEqual(code, 303)
        queued = json.loads(next((self.base/'queue').iterdir()).read_text())
        self.assertEqual(queued['role'], 'agent')
        self.assertNotIn('destination', queued)
        self.activate(s)
        code, headers, body = self.request('/portal/continue')
        self.assertEqual((code, headers['Location']), (303, '/agc/vicidial.php'))

    def test_login_csrf_origin_and_ip_binding(self):
        s = self.login()
        self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': 'wrong'})[0], 403)
        self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf']}, {'Origin': 'https://evil.example'})[0], 403)
        self.assertEqual(self.request('/portal/status', extra={'X-Alma-Source': '1.1.1.1'})[0], 401)

    def test_queue_is_not_active_and_both_fixed_redirects(self):
        for role, destination in portal.DESTINATIONS.items():
            with self.subTest(role=role):
                s = self.login(role)
                self.grant(s)
                self.assertEqual(json.loads(self.request('/portal/status')[2])['state'], 'pending')
                self.assertEqual(self.request('/portal/continue')[0], 403)
                self.assertNotIn('id="continue"', self.request()[2])
                self.activate(s)
                code, headers, _ = self.request('/portal/continue')
                self.assertEqual((code, headers['Location']), (303, destination))
                self.assertIn('Redirecting in: 5 seconds', self.request()[2])
                self.assertIn('Grant expiry:', self.request()[2])
                self.assertIn('hidden>Redirecting in: 5 seconds', self.request()[2])
                self.assertEqual(self.request('/portal/continue?next=https://evil.example')[0], 400)
                self.assertIn('Refresh this page', self.request()[2])

    def test_expired_revoked_failed_stale_worker_do_not_continue(self):
        s = self.login()
        self.grant(s)
        now = int(time.time())
        for overrides in ({'expires_at': now-1}, {'checked_until': now-1},
                          {'state': 'revoked'}, {'state': 'failed'}):
            self.activate(s, **overrides)
            self.assertEqual(self.request('/portal/continue')[0], 403)

    def test_worker_timeout_and_logout(self):
        s = self.login()
        self.grant(s)
        s['queued_at'] -= 121
        self.assertEqual(json.loads(self.request('/portal/status')[2])['state'], 'failed')
        self.assertEqual(self.request('/portal/logout', 'POST', {'csrf': s['csrf']})[0], 303)
        self.assertEqual(self.request('/portal/status')[0], 401)
        self.assertTrue(any(json.loads(p.read_text()).get('kind') == 'revoke_user' for p in (self.base / 'queue').iterdir()))

    def test_rate_limit_and_log_redaction(self):
        _, _, body = self.request()
        csrf = re.search('name="csrf" value="([a-f0-9]+)"', body)[1]
        output = io.StringIO()
        with patch.object(portal, 'authenticate', return_value=None), contextlib.redirect_stdout(output):
            for _ in range(7):
                code, _, body = self.request('/portal/login', 'POST', {'csrf': csrf, 'username': 'mockagent', 'password': 'fake-password-only'})
                self.assertEqual(code, 401)
                self.assertNotIn('fake-password-only', body)
                self.assertIn('role="alert"', body)
            self.assertEqual(self.request('/portal/login', 'POST', {'csrf': csrf, 'username': 'mockagent', 'password': 'fake-password-only'})[0], 429)
        self.assertEqual(output.getvalue().count('AUTH_FAIL ip=8.8.8.8'), 7)
        self.assertNotIn('fake-password-only', output.getvalue())
        self.assertNotIn('mockagent', output.getvalue())


class SupervisorPhonePolicy(unittest.TestCase):
    def test_native_monitor_permissions_and_account_eligibility(self):
        base = {**row(), 'user_level': 9, 'vdc_agent_api_access': '1',
                'api_allowed_functions': ' blind_monitor '}
        for changes, expected in (({}, True), ({'user_level': 8}, True),
                ({'user_level': 7}, False), ({'active': 'N'}, False),
                ({'api_only_user': '1'}, False), ({'force_change_password': 'Y'}, False),
                ({'vdc_agent_api_access': '0'}, False), ({'api_allowed_functions': ' ALL_FUNCTIONS '}, True),
                ({'api_allowed_functions': ' not_blind_monitor '}, False),
                ({'api_allowed_functions': '\tblind_monitor\t'}, False),
                ({'api_allowed_functions': ''}, False)):
            with self.subTest(changes=changes):
                self.assertEqual(monitor_allowed({**base, **changes}), expected)
        self.assertFalse(monitor_allowed(None))


class FirewallLeases(unittest.TestCase):
    def test_account_revoke_defeats_pending_grant_and_preserves_other_users(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            for name in ('web', 'status', 'queue'):
                (base/name).mkdir()
            def enqueue(key, kind, at):
                p = base/'queue'/(key + '.request')
                p.write_text(json.dumps(dict(kind=kind, at=at, ip='8.8.8.8', user='mockagent', role='agent')))
                p.chmod(0o600)
            with patch.object(worker, 'BASE', base), patch.object(worker, 'QUEUE', base/'queue'), \
                 patch.object(worker.pwd, 'getpwnam', return_value=types.SimpleNamespace(pw_uid=os.getuid())), \
                 patch.object(worker.grp, 'getgrnam', return_value=types.SimpleNamespace(gr_gid=0)), \
                 patch.object(worker, 'atomic_json', side_effect=lambda p,v,*a,**k: Path(p).write_text(json.dumps(v))):
                state = {}
                # Lexical processing deliberately sees revoke BEFORE old grant.
                enqueue('a'*32, 'revoke_user', 100)
                enqueue('b'*32, 'grant', 100)
                def process(now):
                    worker.process(state, now, role_check=lambda u:'agent', command=lambda *a:0,
                                   gate=lambda sip, grace, now:set(sip))
                process(100)
                self.assertFalse(worker.aggregate(state['leases'], 100)[1])
                self.assertEqual(state['revoked_before']['mockagent@8.8.8.8:personal'], 100)
                enqueue('c'*32, 'grant', 101)
                process(101)
                self.assertTrue(state['leases'][grant_key('mockagent', '8.8.8.8')]['sip_enabled'])
                state['leases']['e'*32] = dict(ip='8.8.8.8', user='anotheruser', role='agent',
                                                expires_at=500, eligible=True, requested_at=99)
                enqueue('d'*32, 'revoke_user', 102)
                process(102)
                self.assertTrue(state['leases'][grant_key('mockagent', '8.8.8.8')]['revoked'])
                self.assertEqual(state['leases']['e'*32]['expires_at'], 500)
                self.assertEqual(state['sip_previous'], {'8.8.8.8': 500})

    def test_shared_ip_and_roles(self):
        leases = {'a': dict(ip='8.8.8.8', role='agent', expires_at=100, eligible=True),
                  'b': dict(ip='8.8.8.8', role='agent', expires_at=200, eligible=True),
                  'c': dict(ip='8.8.8.8', role='web', expires_at=300, eligible=True)}
        self.assertEqual(worker.aggregate(leases, 50), ({'8.8.8.8': 300}, {'8.8.8.8': 200}))
        leases['b']['expires_at'] = 50
        self.assertEqual(worker.aggregate(leases, 101), ({'8.8.8.8': 300}, {}))
        leases['c']['sip_enabled'] = True
        self.assertEqual(worker.aggregate(leases, 101), ({'8.8.8.8': 300}, {'8.8.8.8': 300}))
        leases['c']['eligible'] = False
        self.assertEqual(worker.aggregate(leases, 101), ({}, {}))

    def test_firewall_confirm_failure_and_revoke(self):
        rules = set()
        calls = []
        def command(*args):
            calls.append(args)
            action, value = args[0].split('=', 1)
            if action == '--query-rich-rule':
                return 0 if value in rules else 1
            if action == '--add-rich-rule':
                rules.add(value)
            else:
                rules.discard(value)
            return 0
        good, state = worker.reconcile({'8.8.8.8': 200}, {}, 100, command)
        self.assertEqual(good, {'8.8.8.8'})
        self.assertEqual(len(rules), 2)
        self.assertTrue(all('--timeout=100' in c for c in calls if c[0].startswith('--add')))
        good, state = worker.reconcile({}, state, 150, command)
        self.assertEqual((good, rules), (set(), set()))
        good, state = worker.reconcile({'8.8.8.8': 200}, {}, 100, lambda *a: 2)
        self.assertEqual(good, set())
        self.assertIn('8.8.8.8', state) # ownership survives partial failure for retry

    def test_worker_rechecks_eligibility_and_publishes_only_after_confirm(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            for name in ('web', 'status', 'queue'):
                (base / name).mkdir()
            with patch.object(worker, 'BASE', base), patch.object(worker, 'QUEUE', base/'queue'), \
                 patch.object(worker.pwd, 'getpwnam', return_value=types.SimpleNamespace(pw_uid=0)), \
                 patch.object(worker.grp, 'getgrnam', return_value=types.SimpleNamespace(gr_gid=0)):
                # Preexisting leases bypass request-owner fixture differences on local non-root runs.
                lease = dict(ip='8.8.8.8', user='mockagent', role='agent', expires_at=1000, eligible=True)
                state = {'leases': {'a'*32: lease}}
                with patch.object(worker, 'atomic_json', side_effect=lambda p,v,*a,**k: Path(p).write_text(json.dumps(v))):
                    worker.process(state, 100, role_check=lambda u: None, command=lambda *a: 1, gate=lambda *a: set())
                    self.assertFalse((base/'web'/'8.8.8.8').exists())
                    self.assertEqual(json.loads((base/'status'/('a'*32)).read_text())['state'], 'failed')
                    worker.process(state, 110, role_check=lambda u: 'agent', command=lambda *a: 2, gate=lambda *a: {'8.8.8.8'})
                    self.assertFalse((base/'web'/'8.8.8.8').exists())
                    def failed_gate(*args):
                        raise RuntimeError('simulated netlink failure')
                    worker.process(state, 115, role_check=lambda u: 'agent', command=lambda *a: 0, gate=failed_gate)
                    self.assertFalse((base/'web'/'8.8.8.8').exists())
                    self.assertEqual(json.loads((base/'status'/('a'*32)).read_text())['state'], 'failed')
                    worker.process(state, 120, role_check=lambda u: 'agent', command=lambda *a: 0, gate=lambda *a: {'8.8.8.8'})
                    self.assertEqual((base/'web'/'8.8.8.8').read_text(), '155')
                    self.assertEqual(json.loads((base/'status'/('a'*32)).read_text())['state'], 'active')
                    lease['expires_at'] = 125
                    worker.process(state, 130, role_check=lambda u: 'agent', command=lambda *a: 1, gate=failed_gate)
                    self.assertEqual(state['rtp_grace'], {'8.8.8.8': 250})
                    self.assertEqual(json.loads((base/'state.json').read_text())['sip_previous'], {})
                    worker.process(state, 300, role_check=lambda u: 'agent', command=lambda *a: 1, gate=failed_gate)
                    self.assertEqual(state['rtp_grace'], {})
                    # A client/stale sip_enabled flag cannot confer phone access.
                    state['leases'] = {'b'*32: dict(ip='8.8.8.8', user='mocksupervisor', role='web',
                                                  expires_at=1000, eligible=True, sip_enabled=True)}
                    gate = lambda sip, grace, now: set(sip)
                    worker.process(state, 350, role_check=lambda u: 'web', command=lambda *a: 0,
                                   gate=gate, phone_check=lambda u: False)
                    self.assertEqual(state['sip_previous'], {})
                    self.assertFalse(state['leases']['b'*32]['sip_enabled'])
                    self.assertTrue((base/'web'/'8.8.8.8').exists())
                    worker.process(state, 360, role_check=lambda u: 'web', command=lambda *a: 0,
                                   gate=gate, phone_check=lambda u: True)
                    self.assertEqual(state['sip_previous'], {'8.8.8.8': 1000})
                    worker.process(state, 370, role_check=lambda u: 'web', command=lambda *a: 0,
                                   gate=gate, phone_check=lambda u: False)
                    self.assertEqual(state['sip_previous'], {})
                    self.assertEqual(state['rtp_grace'], {'8.8.8.8': 490})
                    self.assertTrue((base/'web'/'8.8.8.8').exists())


if __name__ == '__main__':
    unittest.main()
