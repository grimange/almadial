"""Carrier media lifecycle with isolated files and injectable firewall state only."""
import contextlib
import importlib.util
import io
import json
import multiprocessing
import os
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import carrier_cli as cli
import sip_guard as guard

SIP = '38.147.130.74'
MEDIA = '8.8.8.8'  # Fake backends only: never deployed to the live allowlist.
OTHER = '9.9.9.9'


class Backend(cli.Firewall):
    def __init__(self, path, media=False, state=None):
        super().__init__(media)
        self.path = path
        self.state = state if state is not None else {
            'sip': {SIP}, 'media': set(),
            'runtime': {cli.rule(SIP), 'unrelated agent rule'},
            'permanent': {cli.rule(SIP)}, 'failure': None,
            'agents4': {'1.1.1.1': 10}, 'grace4': {'1.1.1.1': 130},
            'policy': {'rules': [], 'media_set': None}}

    def nft(self):
        return set(self.state['media' if self.media else 'sip']), True

    def rich(self, permanent=False):
        return set(self.state['permanent' if permanent else 'runtime'])

    def command(self, *args):
        if args[-1].startswith(('--add-rich-rule=', '--remove-rich-rule=')):
            target = self.state['permanent' if '--permanent' in args else 'runtime']
            op, value = args[-1].split('=', 1)
            target.add(value) if op.startswith('--add') else target.discard(value)
            if self.state['failure'] == 'rich':
                self.state['failure'] = None
                raise RuntimeError('injected partial rich failure')
            return ''
        raise AssertionError(args)

    def sync_nft(self):
        self.state['media' if self.media else 'sip'] = set(cli.load(self.path, self.media)[0])
        if self.state['failure'] == 'nft':
            self.state['failure'] = None
            raise RuntimeError('injected reply loss after commit')

    def restore_nft(self, values):
        self.state['media' if self.media else 'sip'] = set(values)

    def policy_snapshot(self):
        return self.state['policy']

    def restore_policy(self, value):
        self.state['policy'] = value


def child(path, lock, operation, ip, media):
    cli.CONFIG_OWNER = os.getuid()
    with cli.locked(Path(lock)):
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            cli.execute(operation, ip, Path(path), Backend(Path(path), media), media)


class CarrierMedia(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name) / 'carriers.json'
        self.path.write_text(json.dumps([SIP]) + '\n')
        self.path.chmod(0o640)
        self.lock = self.path.with_name('worker.lock')
        self.lock.touch()
        p = patch.object(cli, 'CONFIG_OWNER', os.getuid())
        p.start()
        self.addCleanup(p.stop)
        self.media = Backend(self.path, True)
        self.sip = Backend(self.path, False, self.media.state)

    def run_cli(self, operation, ip=None, media=True):
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            return cli.execute(operation, ip, self.path, self.media if media else self.sip, media)

    def test_media_lifecycle_legacy_migration_and_sip_backward_compatibility(self):
        untouched = {k: self.media.state[k].copy() for k in ('agents4', 'grace4', 'sip')}
        self.run_cli('add', MEDIA)
        self.assertEqual(json.loads(self.path.read_text()), {'sip': [SIP], 'media': [MEDIA]})
        for permanent in (False, True):
            self.assertIn(cli.rule(MEDIA, True), self.media.rich(permanent))
            self.assertIn(cli.rule(SIP), self.media.rich(permanent))
        for op in ('list', 'validate', 'sync'):
            self.assertEqual(self.run_cli(op, media=False), 0)
        self.run_cli('add', OTHER, media=False)
        self.assertEqual(self.media.nft()[0], {MEDIA})
        self.run_cli('remove', OTHER, media=False)
        self.run_cli('remove', MEDIA)
        self.assertEqual(self.media.nft()[0], set())
        self.assertEqual(self.media.managed(), set())
        self.assertEqual(self.media.managed(True), set())
        self.assertIn('unrelated agent rule', self.media.rich())
        for key, value in untouched.items():
            self.assertEqual(self.media.state[key], value)

    def test_metadata_and_other_collection_preserved(self):
        self.path.write_text(json.dumps({'sip': [SIP], 'media': [], 'note': {'ticket': 46}}))
        self.run_cli('add', MEDIA)
        self.run_cli('add', OTHER, media=False)
        value = json.loads(self.path.read_text())
        self.assertEqual(value['note'], {'ticket': 46})
        self.assertEqual(value['media'], [MEDIA])
        self.assertEqual(value['sip'], [SIP, OTHER])

    def test_invalid_config_and_input_no_mutations(self):
        raw = self.path.read_bytes()
        for value in ('bad', '0.0.0.0', '224.1.2.3', '127.0.0.1', '10.0.0.1',
                      '203.0.113.10', '255.255.255.255', '::1', '100.64.0.1'):
            with self.assertRaises(ValueError):
                self.run_cli('add', value)
            self.assertEqual(self.path.read_bytes(), raw)
        for media in ([MEDIA, MEDIA], [1], 'bad'):
            self.path.write_text(json.dumps({'sip': [SIP], 'media': media}))
            with self.assertRaises(ValueError):
                self.run_cli('validate')

    def test_noop_idempotence_and_drift(self):
        self.run_cli('add', MEDIA)
        inode = self.path.stat().st_ino
        for op, ip in [('add', MEDIA), ('sync', None), ('sync', None), ('remove', OTHER)]:
            self.assertEqual(self.run_cli(op, ip), 0)
        self.assertEqual(self.path.stat().st_ino, inode)
        self.media.state['runtime'].remove(cli.rule(MEDIA, True))
        self.media.state['permanent'].add(cli.rule(OTHER, True))
        self.media.state['media'] = {OTHER}
        before = self.path.read_bytes()
        self.assertEqual(self.run_cli('list'), 1)
        self.assertEqual(self.run_cli('validate'), 1)
        self.assertEqual(self.path.read_bytes(), before)
        self.assertEqual(self.run_cli('sync'), 0)
        self.assertEqual(self.media.nft()[0], {MEDIA})
        self.assertEqual(self.media.managed(True), {cli.rule(MEDIA, True)})

    def test_partial_failure_exact_rollback_both_namespaces(self):
        for failure in ('rich', 'nft'):
            before = (self.path.read_bytes(), cli.snapshot(self.sip), cli.snapshot(self.media))
            self.media.state['failure'] = failure
            with self.assertRaisesRegex(RuntimeError, 'Previous state restored'):
                self.run_cli('add', MEDIA)
            self.assertEqual((self.path.read_bytes(), cli.snapshot(self.sip), cli.snapshot(self.media)), before)
        self.media.state['failure'] = 'nft'
        with patch.object(self.media, 'restore_nft', side_effect=RuntimeError('failed rollback')):
            with self.assertRaisesRegex(RuntimeError, 'ROLLBACK INCOMPLETE'):
                self.run_cli('add', MEDIA)

    def test_generated_rules_media_order_local_ports_and_no_sip_authorization(self):
        for chain, field, addr in [('input', 'dport', 'saddr'), ('output', 'sport', 'daddr')]:
            rules = guard.chain_rules(chain)
            media = next(r for r in rules if '@carrier_media4' in r)
            self.assertEqual(media, f'ip {addr} @carrier_media4 udp {field} 10000-20001 counter return')
            self.assertLess(rules.index(media), next(i for i, r in enumerate(rules) if '10000-20000 counter drop' in r))
            self.assertNotIn('5061', media)
            self.assertNotIn('accept', media)
            # The remote RTP port is deliberately unconstrained.
            self.assertNotIn('sport' if field == 'dport' else 'dport', media)
        self.assertEqual(cli.rule(MEDIA, True), f'rule priority="-74" family="ipv4" source address="{MEDIA}/32" port port="10000-20001" protocol="udp" accept')
        self.assertFalse(cli.OWNED.fullmatch(cli.rule(MEDIA, True)))
        self.assertFalse(cli.MEDIA_OWNED.fullmatch(cli.rule(SIP)))

    def test_media_set_readback_restore_scoped_and_guard_bootstrap(self):
        firewall = cli.Firewall(True)
        objects = [{'set': {'name': 'carrier_media4', 'elem': [MEDIA]}},
                   *[{'rule': {'chain': c, 'comment': guard.POLICY, 'expr': [{'match': {'right': '@carrier_media4'}}]}}
                     for c in ('input', 'output')]]
        with patch.object(firewall, 'command', return_value=json.dumps({'nftables': objects})):
            self.assertEqual(firewall.nft(), ({MEDIA}, True))
        with patch.object(guard, 'nft', return_value=types.SimpleNamespace(returncode=0)) as command:
            firewall.restore_nft({MEDIA})
        script = command.call_args.kwargs['script']
        self.assertIn('carrier_media4', script)
        self.assertNotIn(' agents4', script)
        self.assertNotIn(' carriers4', script)
        with patch.object(guard, 'bootstrap') as bootstrap:
            firewall.sync_nft()
            bootstrap.assert_called_once_with(membership='carrier_media4')

    def test_concurrent_media_add_add_and_sip_add_media_remove(self):
        self.run_cli('add', MEDIA)
        scenarios = [([('add', OTHER, True), ('add', MEDIA, True)], {SIP}, {MEDIA, OTHER}),
                     ([('add', OTHER, False), ('remove', MEDIA, True)], {SIP, OTHER}, {OTHER})]
        for actions, sip, media in scenarios:
            ps = [multiprocessing.get_context('spawn').Process(target=child,
                  args=(str(self.path), str(self.lock), op, ip, scope)) for op, ip, scope in actions]
            with cli.locked(self.lock):
                for p in ps: p.start()
                # A competing worker uses the same lock; explicit timeout is bounded.
                with self.assertRaises(RuntimeError):
                    with cli.locked(self.lock, timeout=0.05): pass
            for p in ps:
                p.join(5)
                self.assertEqual(p.exitcode, 0)
            value = guard.validate_config(json.loads(self.path.read_text()))
            self.assertEqual(set(value['sip']), sip)
            self.assertEqual(set(value['media']), media)

    def test_cli_media_dispatch_and_root_boundary(self):
        for op in ('list', 'add', 'remove', 'validate', 'sync'):
            args = ['media', op] + ([MEDIA] if op in ('add', 'remove') else [])
            with patch.object(cli, 'locked', return_value=contextlib.nullcontext()), \
                 patch.object(cli.os, 'geteuid', return_value=0), patch.object(cli, 'execute', return_value=0) as execute:
                self.assertEqual(cli.main(args), 0)
                self.assertTrue(execute.call_args.kwargs['media'])
                self.assertEqual(execute.call_args.args[0], op)
        with patch.object(cli.os, 'geteuid', return_value=10), self.assertRaises(SystemExit):
            cli.main(['media', 'add', MEDIA])

    def test_guard_media_reload_and_bootstrap_keep_dynamic_sets(self):
        self.path.write_text(json.dumps({'sip': [SIP], 'media': [MEDIA]}))
        objects = [{'rule': {'chain': c, 'comment': guard.POLICY}} for c in ('input', 'output')]
        objects += [{'set': {'name': 'carriers4', 'elem': [OTHER]}},
                    {'set': {'name': 'carrier_media4', 'elem': []}}]
        calls = []
        def command(*args, script=None):
            calls.append((args, script))
            return types.SimpleNamespace(returncode=0, stdout=json.dumps({'nftables': objects}))
        with patch.object(guard, 'CARRIER_CONFIG', self.path):
            guard.bootstrap(command, membership='carrier_media4')
            script = calls[-1][1]
            self.assertIn(MEDIA, script)
            self.assertIn('carrier_media4', script)
            self.assertNotIn(' carriers4', script)  # SIP drift is outside a media mutation.
            self.assertNotIn('agents4', script)
            self.assertNotIn('grace4', script)
            objects[-1]['set']['elem'] = [MEDIA]
            calls.clear()
            guard.bootstrap(command, membership='carrier_media4')
            self.assertEqual(len(calls), 1)
            with patch.object(guard, 'nft', side_effect=command):
                fresh = guard.template()
            self.assertIn('set carrier_media4 { type ipv4_addr; elements = { ' + MEDIA, fresh)

    def test_installer_rejects_mismatched_effective_rtp_range(self):
        spec = importlib.util.spec_from_file_location('installer_range', Path(cli.__file__).with_name('install-carriers.py'))
        installer = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(installer)
        for output in ('Port start: 10000\nPort end: 20000\n',
                       'Port start: 5000\nPort end: 9000\n', 'unknown command'):
            with patch.object(installer.subprocess, 'run', return_value=types.SimpleNamespace(stdout=output)):
                if '10000' in output:
                    installer.check_media_range()
                else:
                    with self.assertRaisesRegex(RuntimeError, 'RTP range'):
                        installer.check_media_range()

    def test_installer_media_rollback_and_live_inventory_preservation(self):
        spec = importlib.util.spec_from_file_location('installer_media', Path(cli.__file__).with_name('install-carriers.py'))
        installer = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(installer)
        files = {name: self.path.parent / name for name in installer.FILES}
        for path in files.values():
            path.write_text('# original\n')
            path.chmod(0o644)
        before = (self.path.read_bytes(), cli.snapshot(self.sip), cli.snapshot(self.media))
        real_lock = cli.locked
        factory = lambda media=False: self.media if media else self.sip
        with patch.object(installer, 'FILES', files), patch.object(installer, 'CONFIG', self.path), \
             patch.object(installer, 'BACKUP_ROOT', self.path.parent / 'backups'), patch.object(installer, 'contexts'), \
             patch.object(installer, 'check_media_range'), \
             patch.object(cli.guard, 'CARRIER_CONFIG', self.path), patch.object(cli.guard, 'bootstrap'), \
             patch.object(cli, 'locked', lambda: real_lock(self.lock)), patch.object(cli, 'Firewall', side_effect=factory), \
             patch.object(cli.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['install-carriers.py']), \
             contextlib.redirect_stdout(io.StringIO()):
            installer.main()
            backup = next((self.path.parent / 'backups').iterdir())
            installer.rollback(backup, self.sip)
        self.assertEqual((self.path.read_bytes(), cli.snapshot(self.sip), cli.snapshot(self.media)), before)
        for path in files.values(): self.assertEqual(path.read_text(), '# original\n')


if __name__ == '__main__':
    unittest.main()
