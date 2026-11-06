"""Deterministic mocked clock/DB/firewall tests. Not browser/SIP acceptance."""
import json
import os
from pathlib import Path
import types
import unittest
from unittest.mock import patch

import test_portal as fixtures
from test_portal import row
import apply_requests as worker
import portal
from common import grant_key
from phone_policy import office_allowed


class OfficeHTTP(fixtures.PortalHTTP):
    def setUp(self):
        super().setUp()
        self.office = patch.object(portal, 'office_supervisor', return_value=True)
        self.office.start()

    def tearDown(self):
        self.office.stop()
        super().tearDown()

    def test_agent_cannot_issue_office_or_choose_ip_duration(self):
        s = self.login()
        self.assertNotIn('Shared Office Access —', self.request()[2])
        self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office'})[0], 403)
        self.assertEqual(list((self.base/'queue').iterdir()), [])
        self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'ip': '1.1.1.1', 'duration': '999999'})
        data = json.loads(next((self.base/'queue').iterdir()).read_text())
        self.assertEqual(data['ip'], '8.8.8.8')
        self.assertNotIn('duration', data)

    def test_supervisor_office_options_owned_view_and_scoped_revokes(self):
        s = self.login('web')
        self.assertIn('Shared Office Access — 24 hours', self.request()[2])
        self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office'})[0], 303)
        self.assertEqual(s['request'], grant_key(s['user'], s['ip'], 'office'))
        self.assertEqual(self.request('/portal/continue')[0], 403)
        self.activate(s, grant_type='office', sip_enabled=True)
        self.assertEqual(self.request('/portal/continue')[1]['Location'], '/vicidial/admin.php')
        # A fresh login can find/revoke its office grant without auto-redirect.
        s = self.login('web')
        body = self.request()[2]
        self.assertIn('Revoke Shared Office Access', body)
        self.assertNotIn('<script src=', body)
        self.assertNotIn('id="continue"', body)
        with patch.object(portal, 'office_supervisor', return_value=False):
            self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office'})[0], 403)
            self.assertEqual(self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office', 'action': 'revoke'})[0], 200)
        self.request('/portal/logout', 'POST', {'csrf': s['csrf']})
        revokes = [json.loads(p.read_text()) for p in (self.base/'queue').iterdir() if json.loads(p.read_text())['kind'] == 'revoke_user']
        self.assertEqual({v['grant_type'] for v in revokes}, {'personal', 'office'})

    def test_reauthorization_waits_for_new_worker_confirmation(self):
        import time
        s = self.login('web')
        self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office'})
        now = int(time.time())
        self.activate(s, state='expired', requested_at=now-86401, expires_at=now-1, grant_type='office')
        self.request('/portal/grant', 'POST', {'csrf': s['csrf'], 'grant_type': 'office'})
        self.assertEqual(json.loads(self.request('/portal/status')[2])['state'], 'pending')
        self.assertEqual(self.request('/portal/continue')[0], 403)
        self.activate(s, grant_type='office')
        self.assertEqual(self.request('/portal/continue')[0], 303)

    def test_owned_grants_hide_other_issuers(self):
        s = self.login('web')
        self.grant(s)
        self.activate(s, user='someoneelse', grant_type='office')
        self.assertFalse(portal.own_grants(s))
        self.assertNotIn('Revoke Shared Office Access', self.request()[2])


class OfficeWorker(unittest.TestCase):
    def setUp(self):
        import tempfile
        self.tmp = tempfile.TemporaryDirectory()
        self.base = Path(self.tmp.name)
        for name in ('web', 'status', 'queue'):
            (self.base/name).mkdir()
        self.patches = [patch.object(worker, 'BASE', self.base), patch.object(worker, 'QUEUE', self.base/'queue'),
            patch.object(worker.pwd, 'getpwnam', return_value=types.SimpleNamespace(pw_uid=os.getuid())),
            patch.object(worker.grp, 'getgrnam', return_value=types.SimpleNamespace(gr_gid=0)),
            patch.object(worker, 'atomic_json', side_effect=lambda p,v,*a,**k: Path(p).write_text(json.dumps(v)))]
        for p in self.patches:
            p.start()
        self.state, self.rules, self.serial = {}, set(), 0
        self.eligible = True
        self.role = 'web'
        self.allowed = True

    def tearDown(self):
        for p in reversed(self.patches):
            p.stop()
        self.tmp.cleanup()

    def enqueue(self, kind='grant', grant_type='personal', at=100, user='supervisor'):
        self.serial += 1
        path = self.base/'queue'/('%032x.request' % self.serial)
        path.write_text(json.dumps(dict(kind=kind, grant_type=grant_type, at=at, user=user, ip='8.8.8.8', role=self.role)))
        path.chmod(0o600)
        return grant_key(user, '8.8.8.8', grant_type)

    def process(self, now, failed=False):
        def command(*args):
            action, value = args[0].split('=', 1)
            if failed:
                return 2
            if action == '--query-rich-rule':
                return 0 if value in self.rules else 1
            if action == '--add-rich-rule':
                self.rules.add(value)
            else:
                self.rules.discard(value)
            return 0
        worker.process(self.state, now, role_check=lambda u: self.role if self.eligible else None,
            phone_check=lambda u: True, office_check=lambda u: self.allowed,
            command=command, gate=lambda sip, grace, now:set(sip) if not failed else set())

    def test_durations_idempotence_coexistence_both_revoke_orders_and_grace(self):
        personal = self.enqueue()
        office = self.enqueue(grant_type='office')
        self.process(100)
        self.assertEqual(self.state['leases'][personal]['expires_at'], 43300)
        self.assertEqual(self.state['leases'][office]['expires_at'], 86500)
        self.assertEqual(self.state['leases'][office]['services'], ['https', 'sip', 'rtp'])
        self.assertEqual(len(self.rules), 2)  # aggregate per IP, never per user
        self.enqueue(grant_type='office', at=200)
        self.process(200)
        self.assertEqual(self.state['leases'][office]['expires_at'], 86500)
        self.enqueue('revoke_user', at=201)
        self.process(201)
        self.assertEqual(self.state['sip_previous'], {'8.8.8.8': 86500})
        self.enqueue(at=202)
        self.process(202)
        self.enqueue('revoke_user', 'office', 203)
        self.process(203)
        self.assertEqual(self.state['sip_previous'], {'8.8.8.8': 43402})
        self.enqueue('revoke_user', at=204)
        self.process(204)
        self.assertEqual(self.rules, set())
        self.assertEqual(self.state['rtp_grace'], {'8.8.8.8': 324})
        self.enqueue('revoke_user', at=205)
        self.process(205)
        self.assertEqual(self.state['rtp_grace'], {'8.8.8.8': 324})

    def test_permissions_recheck_invalid_user_and_worker_failure(self):
        key = self.enqueue(grant_type='office')
        self.allowed = False
        self.process(100)
        self.assertNotIn(key, self.state['leases'])
        self.allowed = True
        self.enqueue(grant_type='office', at=101)
        self.process(101)
        self.assertTrue(self.state['leases'][key]['eligible'])
        self.allowed = False
        self.process(102)
        self.assertFalse(self.state['leases'][key]['eligible'])
        self.assertFalse(self.rules)
        self.eligible = False
        self.enqueue(at=103)
        self.process(103)
        self.assertFalse(worker.aggregate(self.state['leases'], 103)[0])
        self.eligible = self.allowed = True
        self.enqueue(at=104)
        self.process(104, failed=True)
        self.assertFalse(list((self.base/'web').iterdir()))

    def test_expiry_stale_queue_reauthorize_and_typed_tombstones(self):
        key = self.enqueue(grant_type='office')
        self.process(100)
        self.enqueue(grant_type='office', at=86499)
        self.process(86501)
        self.assertEqual(self.state['leases'][key]['expires_at'], 86500)
        self.assertFalse(self.rules)
        self.enqueue(grant_type='office', at=86502)
        self.process(86502)
        self.assertEqual(self.state['leases'][key]['expires_at'], 172902)
        self.enqueue('revoke_user', 'office', 86503)
        self.enqueue(grant_type='office', at=86503)
        self.process(86503)
        self.assertFalse(worker.aggregate(self.state['leases'], 86503)[1])
        self.role = 'agent'
        self.enqueue(grant_type='office', at=86504)
        self.process(86504)
        self.assertFalse(worker.aggregate(self.state['leases'], 86504)[1])

    def test_native_office_permission_policy(self):
        base = {**row(), 'user_level': 9, 'modify_users': '1', 'vdc_agent_api_access': '1', 'api_allowed_functions': ' blind_monitor '}
        for changes, expected in (({}, True), ({'user_level': 8}, True), ({'user_level': 7}, False),
            ({'modify_users': '0'}, False), ({'active': 'N'}, False), ({'api_only_user': '1'}, False),
            ({'vdc_agent_api_access': '0'}, False), ({'api_allowed_functions': ''}, False)):
            self.assertEqual(office_allowed({**base, **changes}), expected)
        self.assertFalse(office_allowed(None))
