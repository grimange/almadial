"""Exercise the exact deployed projection block; no portal/grant/firewall mutation."""
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

PATCH = Path(__file__).parents[2] / 'operations/webphone/config/portal-phone-projection.patch'
# Added lines are the bounded block inserted into the existing native portal worker.
CODE = '\n'.join(line[1:] for line in PATCH.read_text().splitlines()
                 if line.startswith('+') and not line.startswith('+++'))
CODE = '\n'.join(line[4:] for line in CODE.splitlines())

class Projection(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.base = Path(self.tmp.name)
        self.env = dict(BASE=self.base, os=os, now=100, HEARTBEAT=35, sip={}, confirmed=set())

    def tearDown(self):
        self.tmp.cleanup()

    def publish(self):
        # Only root ownership is mocked. Real mode, atomic replace and contents are exercised.
        with patch.object(os, 'chown'), patch.object(os, 'fchown'):
            exec(compile(CODE, 'deployed-phone-projection', 'exec'), self.env)
        return self.base / 'phone-websocket'

    def test_web_only_or_unconfirmed_phone_has_no_projection(self):
        self.env['sip'] = {'192.0.2.1': 300}
        self.assertEqual(list(self.publish().iterdir()), [])

    def test_confirmed_phone_freshness_and_permissions(self):
        self.env.update(sip={'192.0.2.1': 300}, confirmed={'192.0.2.1'})
        directory = self.publish()
        self.assertEqual((directory/'192.0.2.1').read_text(), '135')
        self.assertEqual(directory.stat().st_mode & 0o777, 0o750)
        self.assertEqual((directory/'192.0.2.1').stat().st_mode & 0o777, 0o640)
        self.assertFalse(list(directory.glob('.*')))

    def test_short_lease_not_extended_and_revocation_removes_file(self):
        self.env.update(sip={'192.0.2.1': 110}, confirmed={'192.0.2.1'})
        directory=self.publish()
        self.assertEqual((directory/'192.0.2.1').read_text(), '110')
        self.env.update(sip={}, confirmed=set(), now=111)
        self.assertEqual(list(self.publish().iterdir()), [])

    def test_failed_reconfirmation_removes_prior_projection(self):
        self.env.update(sip={'192.0.2.1': 300}, confirmed={'192.0.2.1'})
        self.publish()
        self.env['confirmed']=set()
        self.assertEqual(list(self.publish().iterdir()), [])

if __name__ == '__main__':
    unittest.main()
