"""Deterministic failure-boundary tests; never invoke live Certbot or systemctl."""
import contextlib
import importlib.util
import io
import os
from pathlib import Path
import tempfile
import time
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('tls', Path(__file__).with_name('alma_portal_tls.py'))
tls = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tls)
actual_command = tls.command


class TLSIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.stack = contextlib.ExitStack()
        self.addCleanup(self.stack.close)
        self.stack.enter_context(contextlib.redirect_stdout(io.StringIO()))
        self.stack.enter_context(patch.object(tls, 'STATE', Path(self.temp.name)))
        self.stack.enter_context(patch.object(tls, 'identity', return_value=('2.29.153.161', Path('/expected'))))
        self.stack.enter_context(patch.object(tls, 'disk_fingerprint', return_value='abc'))
        self.cert = self.stack.enter_context(patch.object(tls, 'certificate', return_value={'fingerprint': 'abc'}))
        self.served = self.stack.enter_context(patch.object(tls, 'served', return_value='abc'))
        self.wait = self.stack.enter_context(patch.object(tls, 'wait_served'))
        self.cmd = self.stack.enter_context(patch.object(tls, 'command', return_value=''))

    def receipt(self, outcome='ok'):
        def execute(stage, argv, **kwargs):
            if stage in ('certbot_renew', 'certbot_dry_run', 'deployment_recovery'):
                tls.save_state('deployment.json', {'run_id': kwargs['env']['ALMA_TLS_RUN_ID'],
                                                   'outcome': outcome})
            return ''
        return execute

    def test_noop_does_not_reload_or_invoke_hook(self):
        tls.renew()
        state = tls.read_state('status.json')
        self.assertEqual(state['outcome'], 'ok')
        self.assertFalse(state['deployed'])
        self.assertFalse(state['certificate_changed'])
        self.assertEqual([call.args[0] for call in self.cmd.call_args_list], ['certbot_renew'])

    def test_dry_run_explicitly_exercises_deploy_hook(self):
        self.cmd.side_effect = self.receipt()
        tls.renew(True)
        argv = self.cmd.call_args_list[0].args[1]
        self.assertIn('--dry-run', argv)
        self.assertIn('--run-deploy-hooks', argv)
        self.assertTrue(tls.read_state('status.json')['deployed'])

    def test_dry_run_without_receipt_fails(self):
        with self.assertRaisesRegex(tls.Failure, 'receipt missing'):
            tls.renew(True)
        self.assertEqual(tls.read_state('status.json')['outcome'], 'failed')

    def test_certbot_zero_cannot_hide_failed_hook(self):
        self.cmd.side_effect = self.receipt('failed')
        with self.assertRaisesRegex(tls.Failure, 'despite Certbot'):
            tls.renew()
        self.assertEqual(tls.read_state('status.json')['outcome'], 'failed')

    def test_changed_certificate_requires_deploy_receipt(self):
        self.cert.return_value = {'fingerprint': 'new'}
        self.served.return_value = 'new'
        with self.assertRaisesRegex(tls.Failure, 'receipt missing'):
            tls.renew()

    def test_failed_reload_persists_failure(self):
        def execute(stage, argv, **kwargs):
            if stage == 'apache_reload':
                raise tls.Failure('apache_reload: exit 1')
        self.cmd.side_effect = execute
        with self.assertRaisesRegex(tls.Failure, 'apache_reload'):
            tls.deploy()
        self.assertEqual(tls.read_state('deployment.json')['outcome'], 'failed')
        self.wait.assert_not_called()

    def test_bad_config_never_reloads(self):
        def execute(stage, argv, **kwargs):
            if stage == 'apache_config':
                raise tls.Failure('apache_config: exit 1')
        self.cmd.side_effect = execute
        with self.assertRaises(tls.Failure):
            tls.deploy()
        self.assertEqual([c.args[0] for c in self.cmd.call_args_list], ['apache_config'])

    def test_unexpected_lineage_fails_before_reload(self):
        with patch.dict(os.environ, RENEWED_LINEAGE='/unexpected'):
            with self.assertRaisesRegex(tls.Failure, 'unexpected renewal target'):
                tls.deploy()
        self.cmd.assert_not_called()

    def test_serve_mismatch_makes_deployment_fail(self):
        self.wait.side_effect = tls.Failure('served_certificate: mismatch')
        with self.assertRaises(tls.Failure):
            tls.deploy()
        self.assertEqual(tls.read_state('deployment.json')['outcome'], 'failed')

    def test_previous_failed_deployment_retried_on_noop(self):
        tls.save_state('deployment.json', {'outcome': 'failed', 'run_id': 'older'})
        self.cmd.side_effect = lambda stage, argv, **kwargs: (
            self.receipt()(stage, argv, **kwargs) if stage == 'deployment_recovery' else '')
        tls.renew()
        self.assertTrue(tls.read_state('status.json')['deployed'])
        self.assertEqual(self.cmd.call_args_list[-1].args[0], 'deployment_recovery')

    def test_tls_connection_failure_attempts_deployment_recovery(self):
        self.served.side_effect = OSError('unreachable')
        self.cmd.side_effect = self.receipt()
        tls.renew()
        self.assertTrue(tls.read_state('status.json')['deployed'])

    def test_stale_or_failed_state_fails_health(self):
        for state in [dict(outcome='failed', last_success_at=time.time()),
                      dict(outcome='ok', last_success_at=time.time() - 31 * 3600),
                      dict(outcome='running', last_success_at=time.time(), started_at=time.time() - 721)]:
            with self.subTest(state=state):
                tls.save_state('status.json', state)
                with self.assertRaises(tls.Failure):
                    tls.check()

    def test_atomic_state_is_private(self):
        tls.save_state('status.json', {'outcome': 'ok'})
        tls.save_state('status.json', {'outcome': 'failed'})
        self.assertEqual(tls.read_state('status.json')['outcome'], 'failed')
        self.assertEqual((tls.STATE / 'status.json').stat().st_mode & 0o777, 0o600)
        self.assertEqual(list(tls.STATE.glob('status.json.*')), [])

    def test_nonzero_subprocess_is_not_suppressed(self):
        # Use the actual boundary, with a harmless local Python subprocess.
        with self.assertRaisesRegex(tls.Failure, 'exit 7'):
            actual_command('negative', ['/usr/bin/python3', '-c', 'raise SystemExit(7)'])


if __name__ == '__main__':
    unittest.main()
