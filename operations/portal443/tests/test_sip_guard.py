import json
import sys
import types
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import sip_guard as guard


class SipGuard(unittest.TestCase):
    def test_grace_deadlines_and_shared_ip(self):
        self.assertEqual(guard.grace_state({'8.8.8.8': 900}, {'8.8.8.8': 800}, {}, 100), {})
        grace = guard.grace_state({}, {'8.8.8.8': 900}, {}, 100)
        self.assertEqual(grace, {'8.8.8.8': 220})
        self.assertEqual(guard.grace_state({}, {}, grace, 150), grace)
        self.assertEqual(guard.grace_state({}, {}, grace, 221), {})
        self.assertEqual(guard.grace_state({}, {'8.8.8.8': 90}, {}, 100), {'8.8.8.8': 210})

    def test_freshness_and_absolute_expiry_atomic_confirmation(self):
        calls = []
        def command(*args, script=None):
            calls.append((args, script))
            if args[:3] == ('-j', 'list', 'table'):
                return types.SimpleNamespace(returncode=0, stdout=json.dumps({'nftables': [
                    {'rule': {'chain': chain, 'comment': guard.POLICY}} for chain in ('input', 'output')]}))
            if args[0] == '-j':
                members = ['8.8.8.8', '9.9.9.9'] if args[-1] == 'agents4' else ['8.8.8.8', '9.9.9.9', '1.1.1.1']
                data = {'nftables': [{'set': {'elem': [{'elem': {'val': ip}} for ip in members]}}]}
                return types.SimpleNamespace(returncode=0, stdout=json.dumps(data))
            return types.SimpleNamespace(returncode=0)
        confirmed = guard.refresh({'8.8.8.8': 1000, '9.9.9.9': 110}, {'1.1.1.1': 120}, 100, command)
        self.assertEqual(confirmed, {'8.8.8.8', '9.9.9.9'})
        script = next(s for a, s in calls if a[0] == '-f')
        self.assertIn('8.8.8.8 timeout 35s', script)
        self.assertIn('9.9.9.9 timeout 10s', script)
        self.assertIn('8.8.8.8 timeout 155s', script)
        self.assertIn('1.1.1.1 timeout 20s', script)
        self.assertEqual(sum(1 for a,s in calls if a[0] == '-f'), 1)

    def test_unconfirmed_or_failed_guard_never_succeeds(self):
        with self.assertRaises(RuntimeError):
            guard.refresh({'8.8.8.8': 200}, {}, 100,
                          lambda *a, **k: types.SimpleNamespace(returncode=1))
        def mismatch(*a, **k):
            return types.SimpleNamespace(returncode=0, stdout='{"nftables":[{"set":{}}]}')
        with self.assertRaises(RuntimeError):
            guard.refresh({'8.8.8.8': 200}, {}, 100, mismatch)

    def test_guard_covers_both_directions_before_established_accept(self):
        script = guard.template()
        self.assertEqual(script.count('priority -10'), 2)
        for direction in ('dport', 'sport'):
            for port in ('5060', '5061'):
                self.assertIn(f'udp {direction} {port} counter drop', script)
            self.assertIn(f'udp {direction} 10000-20000 counter drop', script)
        self.assertEqual(script.count('ct state established ct mark & 0x40000000 != 0 return'), 2)
        self.assertNotIn('flush ruleset', script)
        self.assertNotIn(' accept\n', script)

    def test_existing_policy_upgrade_preserves_sets_and_is_atomic(self):
        calls = []
        def command(*args, script=None):
            calls.append((args, script))
            return types.SimpleNamespace(returncode=0, stdout='{"nftables":[]}')
        guard.bootstrap(command)
        self.assertEqual(len(calls), 2)
        script = calls[1][1]
        for chain in ('input', 'output'):
            self.assertIn(f'flush chain inet {guard.TABLE} {chain}', script)
        self.assertNotIn('flush set', script)
        self.assertNotIn('delete table', script)
        self.assertIn('udp dport 5060 counter drop', script)
        self.assertIn('udp sport 5061 counter drop', script)
        # Failure cannot be advertised as a successful policy migration.
        with self.assertRaises(RuntimeError):
            guard.bootstrap(lambda *a, **k: types.SimpleNamespace(
                returncode=1 if a[0] == '-f' else 0, stdout='{"nftables":[]}'))

    def test_current_policy_bootstrap_does_not_duplicate_rules(self):
        calls = []
        def command(*args, **kwargs):
            calls.append(args)
            return types.SimpleNamespace(returncode=0, stdout=json.dumps({'nftables': [
                {'rule': {'chain': chain, 'comment': guard.POLICY}} for chain in ('input', 'output')]}))
        guard.bootstrap(command)
        self.assertEqual(len(calls), 1)


if __name__ == '__main__':
    unittest.main()
