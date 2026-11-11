"""Focused S09 native safety fixtures; no root or network changes."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock

ROOT = Path(__file__).resolve().parents[2]


class NetworkActivation(unittest.TestCase):
    def fixture(self, directory):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        ns['POLICY'] = Path(directory)/'almadial-s09.xml'
        ns['inspect'] = Mock(side_effect=[dict(Current='expected', Foreign='foreign'), dict(Foreign='foreign', Health='verified')])
        ns['service'] = lambda: ['inactive', 'enabled']
        ns['policy_health'] = lambda p: False
        ns['run'] = Mock(return_value='')
        ns['write_policy'] = lambda text: ns['POLICY'].write_text(text)
        return ns, dict(ExpectedCurrent='expected', XML='approved')

    def test_success_checks_effective_state_before_disarming_recovery(self):
        with tempfile.TemporaryDirectory() as d:
            ns, p = self.fixture(d)
            ns['activate'](p)
            self.assertEqual(ns['POLICY'].read_text(), 'approved')
            commands = [c.args[0] for c in ns['run'].call_args_list]
            self.assertIn('--on-active=120s', commands[0])
            self.assertIn("'enable'", commands[0][-1])
            self.assertEqual(commands[-1], ['/usr/bin/systemctl', 'stop', 'almadial-s09-recovery.timer'])
            self.assertEqual(ns['inspect'].call_count, 2)

    def test_failed_activation_restores_only_owned_policy_and_prior_service_state(self):
        with tempfile.TemporaryDirectory() as d:
            ns, p = self.fixture(d)
            foreign = Path(d)/'operator.xml'
            foreign.write_text('operator data')
            ns['inspect'].side_effect = [dict(Current='expected', Foreign='foreign'), ValueError('failed check')]
            run = ns['run']
            def execute_rollback(argv, optional=False):
                if argv[:4] == ['/usr/bin/python3', '-B', '-c', argv[-1]]:
                    exec(argv[-1], {'__builtins__': __builtins__})
                return ''
            # Execute rollback file operations, while replacing its systemctl subprocess.
            from unittest.mock import patch
            run.side_effect = execute_rollback
            with patch('subprocess.run') as native:
                with self.assertRaisesRegex(ValueError, 'prior managed state restored'):
                    ns['activate'](p)
                self.assertEqual(native.call_args_list[-1].args[0], ['/usr/bin/systemctl', 'enable', 'firewalld'])
            self.assertFalse(ns['POLICY'].exists())
            self.assertEqual(foreign.read_text(), 'operator data')

    def test_stale_state_and_noop_do_not_start_any_native_operation(self):
        with tempfile.TemporaryDirectory() as d:
            ns, p = self.fixture(d)
            p['ExpectedCurrent'] = 'stale'
            with self.assertRaisesRegex(ValueError, 'stale network transaction'):
                ns['activate'](p)
            ns['run'].assert_not_called()
            ns['inspect'] = lambda p: dict(Current='expected')
            p['ExpectedCurrent'] = 'expected'
            ns['policy_health'] = lambda p: True
            ns['activate'](p)
            ns['run'].assert_not_called()

    def test_runtime_only_foreign_state_is_rejected(self):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        ns['fw'] = lambda *args, **kw: 'persistent' if kw.get('permanent') else 'runtime'
        with self.assertRaisesRegex(ValueError, 'foreign runtime-only zone state'):
            ns['foreign_runtime']()

    def test_installed_default_active_header_preserves_real_divergence(self):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        permanent = 'public (default)\n  interfaces: eth0\n  services: ssh\n'
        runtime = permanent.replace('(default)', '(default, active)')
        ns['fw'] = lambda *a, **k: (permanent if k.get('permanent') else runtime) if a == ('--list-all-zones',) else ''
        self.assertEqual(len(ns['foreign_runtime']()), 64)
        runtime = runtime.replace('services: ssh', 'services: ssh https')
        with self.assertRaisesRegex(ValueError, 'foreign runtime-only zone state'):
            ns['foreign_runtime']()
        self.assertNotEqual(ns['normalized']('public (default, disabled)'), ns['normalized']('public (default)'))

    def test_native_failure_identifies_operation_status_and_both_streams(self):
        from subprocess import CompletedProcess
        from unittest.mock import patch
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        argv = ['/usr/bin/firewall-cmd', '--list-all-zones']
        with patch('subprocess.run', return_value=CompletedProcess(argv, 7, 'distinct stdout', 'distinct stderr')):
            with self.assertRaisesRegex(ValueError, 'firewall-cmd --list-all-zones.*exit 7.*distinct stdout.*distinct stderr'):
                ns['run'](argv)

    def test_networkmanager_binding_requires_matching_active_profile(self):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        permanent = 'public (default)\n  interfaces: \n  services: ssh\n'
        runtime = permanent.replace('(default)', '(default, active)').replace('interfaces: ', 'interfaces: eth0')
        def fw(*args, permanent=False):
            if args == ('--list-all-zones',): return permanent_text if permanent else runtime
            return 'public' if args == ('--get-default-zone',) else ''
        permanent_text = permanent
        ns['fw'] = fw
        uuid = '1dd9a779-d327-56e1-8454-c65e2556c12c'
        ns['run'] = Mock(side_effect=[uuid, ''])
        self.assertEqual(len(ns['foreign_runtime']()), 64)
        ns['run'] = Mock(side_effect=[uuid, 'trusted'])
        with self.assertRaisesRegex(ValueError, 'zone differs from NetworkManager'):
            ns['foreign_runtime']()
        ns['run'] = Mock(return_value='--')
        with self.assertRaisesRegex(ValueError, 'lacks active NetworkManager'):
            ns['foreign_runtime']()
        runtime = runtime.replace('services: ssh', 'services: ssh https')
        ns['run'] = Mock(side_effect=[uuid, ''])
        with self.assertRaisesRegex(ValueError, 'foreign runtime-only zone state.*permanent'):
            ns['foreign_runtime']()

    def test_foreign_nft_hook_is_preserved_and_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            ns, p = self.fixture(d)
            ns['ROOT'] = Path(d)
            original = [{'chain': dict(family='inet', table='operator', name='input')}]
            ns['nft'] = lambda: original
            with self.assertRaisesRegex(ValueError, 'foreign nftables state requires review'):
                ns['foreign']()
            self.assertEqual(original[0]['chain']['table'], 'operator')

    def test_vendor_ipv6_policy_can_remain_ahead_of_zones_without_competing(self):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        def fw(*args, permanent=False):
            if args == ('--list-all-zones',): return 'public' if permanent else 'public (active)'
            if args == ('--get-policies',): return 'allow-host-ipv6'
            if args == ('--info-policy=allow-host-ipv6',): return 'allow-host-ipv6' if permanent else 'allow-host-ipv6 (active)'
            if args == ('--policy=allow-host-ipv6', '--get-priority'):
                self.assertTrue(permanent, 'installed CLI rejects runtime priority query')
                return '-15000'
            return ''
        ns['fw'] = fw
        self.assertEqual(len(ns['foreign_runtime']()), 64)
        base = fw
        ns['fw'] = lambda *a, **k: '-21000' if a == ('--policy=allow-host-ipv6', '--get-priority') else base(*a, **k)
        with self.assertRaisesRegex(ValueError, 'competing policy priority'):
            ns['foreign_runtime']()

    def test_policy_health_uses_supported_permanent_only_options(self):
        with tempfile.TemporaryDirectory() as d:
            ns, p = self.fixture(d)
            del ns['policy_health']
            # Restore the native health implementation replaced by fixture().
            original = {'__name__': 's09_health'}
            exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), original)
            original['POLICY'] = ns['POLICY']
            ns['POLICY'].write_text('approved')
            original['service'] = lambda: ['active', 'enabled']
            original['run'] = Mock(return_value='verified label')
            p['Request'] = dict(SSHPort=22)
            values = {'--get-target': 'DROP', '--get-priority': '-20000', '--list-ingress-zones': 'ANY', '--list-egress-zones': 'HOST', '--list-ports': '22/tcp', '--list-protocols': 'icmp ipv6-icmp'}
            def fw(*args, permanent=False):
                if args == ('--get-active-policies',): return 'almadial-s09'
                if args == ('--info-policy=almadial-s09',): return 'almadial-s09' if permanent else 'almadial-s09 (active)'
                if args[-1] in ('--get-priority', '--get-target'):
                    self.assertTrue(permanent, 'installed CLI rejects runtime option')
                return values.get(args[-1], '')
            original['fw'] = fw
            original['nft'] = lambda: [{'rule': dict(chain='filter_IN_policy_almadial-s09', expr=['drop', 'accept', 'tcp', 22])}]
            self.assertTrue(original['policy_health'](p))

    def test_disabled_vendor_policy_retains_semantic_disable_field(self):
        ns = {'__name__': 's09_fixture'}
        exec(compile((ROOT/'internal/provision/network_native.py').read_text(), 'network_native.py', 'exec'), ns)
        permanent = 'gateway-dmz-to-HOST\n  disable: yes\n  target: CONTINUE'
        runtime = permanent.replace('gateway-dmz-to-HOST', 'gateway-dmz-to-HOST (disabled)')
        self.assertEqual(ns['normalized_policy'](runtime), ns['normalized_policy'](permanent))
        self.assertNotEqual(ns['normalized_policy'](runtime.replace('disable: yes', 'disable: no')), ns['normalized_policy'](permanent))
