import contextlib
import io
import importlib.util
import json
import multiprocessing
import os
from pathlib import Path
import sys
import tempfile
import time
import types
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import carrier_cli as cli
import sip_guard as guard

A = '38.147.130.74'
B = '8.8.8.8'  # Isolated fake backend only; never authorize live.
C = '9.9.9.9'


class FakeFirewall:
    def __init__(self, path):
        self.path = path
        self.members = set(json.loads(path.read_text()))
        self.runtime = {cli.rule(x) for x in self.members}
        self.permanent = set(self.runtime)
        self.unrelated = {'rule priority="10" family="ipv4" source address="8.8.8.8/32" port port="5060" protocol="udp" accept'}
        self.agents = {'1.1.1.1': 12345}
        self.calls = []
        self.failure = None
        self.policy = True

    def policy_snapshot(self):
        return {'rules': [], 'media_set': None}

    def restore_policy(self, saved):
        self.policy = True

    def nft(self):
        return set(self.members), self.policy

    def managed(self, permanent=False):
        return set(self.permanent if permanent else self.runtime)

    def set_rich(self, desired, permanent=False):
        self.calls.append(('rich', permanent, set(desired)))
        if permanent:
            self.permanent = set(desired)
        else:
            self.runtime = set(desired)
        if self.failure == 'rich':
            self.failure = None
            raise RuntimeError('injected failure after runtime commit')

    def sync_nft(self):
        self.calls.append(('nft',))
        self.members = set(json.loads(self.path.read_text()))
        if self.failure == 'nft':
            self.failure = None
            raise RuntimeError('injected lost reply after nft commit')

    def restore_nft(self, members):
        self.members = set(members)


def concurrent_change(config, lock, operation, address, delay=0):
    cli.CONFIG_OWNER = os.getuid()
    with cli.locked(Path(lock)):
        # A worker refresh and CLI operations all serialize on the same inode.
        time.sleep(delay)
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            cli.execute(operation, address, Path(config), FakeFirewall(Path(config)))


class CarrierCLI(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name) / 'carriers.json'
        self.path.write_text(json.dumps([A]) + '\n')
        self.path.chmod(0o640)
        self.lock = Path(self.tmp.name) / 'worker.lock'
        self.lock.touch()
        self.patch = patch.object(cli, 'CONFIG_OWNER', os.getuid())
        self.patch.start()
        self.addCleanup(self.patch.stop)
        self.fw = FakeFirewall(self.path)

    def call(self, operation, ip=None):
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            return cli.execute(operation, ip, self.path, self.fw)

    def test_add_remove_preserves_config_and_agents(self):
        original_agents = dict(self.fw.agents)
        unrelated = set(self.fw.unrelated)
        self.assertEqual(self.call('add', B), 0)
        self.assertEqual(json.loads(self.path.read_text()), [A, B])
        self.assertEqual(self.fw.members, {A, B})
        self.assertEqual(self.fw.runtime, {cli.rule(A), cli.rule(B)})
        self.assertEqual(self.fw.permanent, self.fw.runtime)
        self.call('remove', B)
        self.assertEqual(json.loads(self.path.read_text()), [A])
        self.assertEqual(self.fw.runtime, {cli.rule(A)})
        self.assertEqual(self.fw.agents, original_agents)
        self.assertEqual(self.fw.unrelated, unrelated)

    def test_noop_and_sync_idempotence(self):
        inode = self.path.stat().st_ino
        for operation, ip in [('add', A), ('sync', None), ('sync', None)]:
            self.assertEqual(self.call(operation, ip), 0)
        self.assertEqual(self.path.stat().st_ino, inode)
        self.assertEqual(len(self.fw.runtime), 1)
        self.fw.calls.clear()
        self.call('remove', B)
        self.assertEqual(self.fw.calls, [])

    def test_invalid_unsuitable_and_duplicate_addresses(self):
        original = self.path.read_bytes()
        for value in ['garbage', '1.2.3', '::1', '0.0.0.0', '127.0.0.1', '224.0.0.1', '255.255.255.255', '10.0.0.1', '203.0.113.10', '240.1.1.1', '100.64.0.1', '169.254.1.1']:
            with self.subTest(value=value), self.assertRaises(ValueError):
                self.call('add', value)
            self.assertEqual(self.path.read_bytes(), original)
        self.path.write_text(json.dumps([A, A]))
        with self.assertRaisesRegex(ValueError, 'Duplicate'):
            self.call('validate')

    def test_invalid_json_schema_permissions_owner_and_symlink(self):
        for contents in ['{', '{}', '[2]', '{"carriers":[],"unrelated":"keep"}']:
            self.path.write_text(contents)
            with self.assertRaises(ValueError):
                self.call('sync')
            self.assertEqual(self.path.read_text(), contents)
        self.path.write_text(json.dumps([A]))
        self.path.chmod(0o666)
        with self.assertRaisesRegex(ValueError, 'root-owned'):
            self.call('sync')
        self.path.chmod(0o640)
        with patch.object(cli, 'CONFIG_OWNER', -1), self.assertRaises(ValueError):
            self.call('validate')
        link = self.path.with_name('link')
        link.symlink_to(self.path)
        with self.assertRaises(OSError):
            cli.load(link)

    def test_atomic_replace_preserves_mode(self):
        inode = self.path.stat().st_ino
        self.call('add', B)
        self.assertNotEqual(self.path.stat().st_ino, inode)
        self.assertEqual(self.path.stat().st_mode & 0o777, 0o640)
        self.assertEqual(list(self.path.parent.glob('.carriers-*')), [])
        current = self.path.read_bytes()
        with patch.object(cli.os, 'replace', side_effect=OSError('injected replace failure')):
            with self.assertRaises(OSError):
                cli.atomic_write(self.path, b'[]', 0o640)
        self.assertEqual(self.path.read_bytes(), current)
        self.assertEqual(list(self.path.parent.glob('.carriers-*')), [])

    def test_readonly_reports_all_drift_without_mutation(self):
        self.fw.members = set()
        self.fw.runtime = set()
        self.fw.permanent.add(cli.rule(B))
        self.fw.policy = False
        original = self.path.read_bytes()
        self.assertEqual(self.call('list'), 1)
        self.assertEqual(self.call('validate'), 1)
        self.assertEqual(self.path.read_bytes(), original)
        self.assertEqual(self.fw.calls, [])

    def test_sync_repairs_only_managed_drift(self):
        self.fw.members = {B}
        self.fw.runtime = {cli.rule(B)}
        self.fw.permanent = set()
        self.call('sync')
        self.assertEqual(self.fw.members, {A})
        self.assertEqual(self.fw.runtime, {cli.rule(A)})
        self.assertEqual(self.fw.permanent, {cli.rule(A)})
        self.assertEqual(len(self.fw.unrelated), 1)

    def test_partial_failure_restores_exact_previous_drift(self):
        for failure in ('rich', 'nft'):
            with self.subTest(failure=failure):
                self.fw.runtime = {cli.rule(B)}
                previous = cli.snapshot(self.fw)
                original = self.path.read_bytes()
                self.fw.failure = failure
                with self.assertRaisesRegex(RuntimeError, 'Previous state restored'):
                    self.call('add', C)
                self.assertEqual(cli.snapshot(self.fw), previous)
                self.assertEqual(self.path.read_bytes(), original)

    def test_rollback_failure_is_explicit(self):
        self.fw.failure = 'nft'
        with patch.object(self.fw, 'restore_nft', side_effect=RuntimeError('injected restore error')):
            with self.assertRaisesRegex(RuntimeError, 'ROLLBACK INCOMPLETE.*nftables'):
                self.call('add', B)

    def test_rule_namespace_and_port_scope(self):
        fw = cli.Firewall()
        unrelated = self.fw.unrelated | {cli.rule(B).replace('5060', '5061'), cli.rule(C).replace('accept', 'drop')}
        current = {cli.rule(A), cli.rule(B)}
        commands = []
        def command(*args):
            commands.append(args)
            if args[-1] == '--list-rich-rules':
                return '\n'.join(current | unrelated)
            if '--remove-rich-rule=' in args[-1]:
                current.remove(args[-1].split('=', 1)[1])
            if '--add-rich-rule=' in args[-1]:
                current.add(args[-1].split('=', 1)[1])
            return ''
        with patch.object(fw, 'command', side_effect=command):
            fw.set_rich({cli.rule(A)}, True)
            fw.set_rich({cli.rule(A)}, True)
        changes = [a for a in commands if a[-1] != '--list-rich-rules']
        self.assertEqual(len(changes), 1)
        self.assertIn('--permanent', changes[0])
        self.assertEqual(changes[0][-1], '--remove-rich-rule=' + cli.rule(B))
        self.assertNotIn('10000', cli.rule(A))
        self.assertNotIn('5061', cli.rule(A))

    def test_nonroot_mutation_denied(self):
        with patch.object(cli.os, 'geteuid', return_value=123):
            with self.assertRaises(SystemExit) as result:
                cli.main(['add', A])
        self.assertEqual(result.exception.code, 1)

    def test_nft_readback_and_restore_are_carrier_only(self):
        fw = cli.Firewall()
        objects = [{'set': {'name': 'carriers4', 'elem': [A]}},
                   *[{'rule': {'chain': chain, 'comment': guard.POLICY,
                               'expr': [{'match': {'right': '@carriers4'}}]}}
                     for chain in ('input', 'output')]]
        with patch.object(fw, 'command', return_value=json.dumps({'nftables': objects})):
            self.assertEqual(fw.nft(), ({A}, True))
        with patch.object(guard, 'nft', return_value=types.SimpleNamespace(returncode=0)) as command:
            fw.restore_nft({A})
        script = command.call_args.kwargs['script']
        self.assertEqual(script, f'flush set inet {guard.TABLE} carriers4\nadd element inet {guard.TABLE} carriers4 {{ {A} }}\n')
        objects[0]['set']['flags'] = ['timeout']
        with patch.object(fw, 'command', return_value=json.dumps({'nftables': objects})):
            with self.assertRaisesRegex(RuntimeError, 'timeout'):
                fw.nft()

    def test_installer_preserves_live_config_and_rollback_restores_files(self):
        spec = importlib.util.spec_from_file_location('install_carriers', Path(cli.__file__).with_name('install-carriers.py'))
        installer = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(installer)
        files = {name: self.path.parent / name for name in installer.FILES}
        original_guard = b'# pre-install guard\n'
        files['sip_guard.py'].write_bytes(original_guard)
        files['sip_guard.py'].chmod(0o644)
        backup_root = self.path.parent / 'backups'
        # FakeFirewall tests the migration ordering and exact legacy adoption.
        legacy = {False: {installer.LEGACY}, True: {installer.LEGACY}}
        self.fw.runtime.clear()
        self.fw.permanent.clear()
        before = cli.snapshot(self.fw)
        def rich(permanent=False):
            return self.fw.managed(permanent) | legacy[permanent]
        def command(*args):
            permanent = '--permanent' in args
            if args[-1].startswith('--remove-rich-rule='):
                legacy[permanent].discard(args[-1].split('=', 1)[1])
            elif args[-1].startswith('--add-rich-rule='):
                legacy[permanent].add(args[-1].split('=', 1)[1])
        original = self.path.read_bytes()
        real_lock = cli.locked
        with patch.object(installer, 'FILES', files), patch.object(installer, 'CONFIG', self.path), \
             patch.object(installer, 'BACKUP_ROOT', backup_root), patch.object(installer, 'contexts'), \
             patch.object(installer, 'check_media_range'), \
             patch.object(installer.cli.guard, 'CARRIER_CONFIG', self.path), \
             patch.object(installer.cli, 'locked', lambda: real_lock(self.lock)), \
             patch.object(installer.cli.os, 'geteuid', return_value=0), \
             patch.object(installer.cli, 'Firewall', return_value=self.fw), \
             patch.object(installer.cli.guard, 'bootstrap'), \
             patch.object(installer.cli, 'execute', wraps=cli.execute) as execute, \
             patch.object(self.fw, 'rich', side_effect=rich, create=True), \
             patch.object(self.fw, 'command', side_effect=command, create=True), \
             patch.object(sys, 'argv', ['install-carriers.py', '--adopt-peeredge']), \
             contextlib.redirect_stdout(io.StringIO()):
            # SIP fake covers the original installer migration; the separate
            # media integration backend is exercised in test_carrier_media.
            real_execute = execute._mock_wraps
            execute.side_effect = lambda *a, **k: 0 if k.get('media') else real_execute(*a, **k)
            installer.main()
            backup = next(backup_root.iterdir())
            self.assertEqual(self.path.read_bytes(), original)
            self.assertEqual(legacy, {False: set(), True: set()})
            self.assertEqual(cli.snapshot(self.fw)[2:], ({cli.rule(A)}, {cli.rule(A)}))
            installer.rollback(backup, self.fw)
            self.assertEqual(files['sip_guard.py'].read_bytes(), original_guard)
            self.assertFalse(files['carrier_cli.py'].exists())
            self.assertFalse(files['alma-portal-carriers'].exists())
            self.assertEqual(cli.snapshot(self.fw), before)
            self.assertEqual(legacy, {False: {installer.LEGACY}, True: {installer.LEGACY}})
            self.path.write_text('[]')
            with self.assertRaisesRegex(RuntimeError, 'changed since installation'):
                installer.rollback(backup, self.fw)

    def test_concurrent_add_add_and_add_remove(self):
        for actions, expected in [([('add', B), ('add', C)], {A, B, C}),
                                  ([('add', B), ('remove', A)], {B})]:
            self.path.write_text(json.dumps([A]))
            processes = [multiprocessing.get_context('spawn').Process(target=concurrent_change,
                         args=(str(self.path), str(self.lock), op, ip, 0.1)) for op, ip in actions]
            for p in processes: p.start()
            for p in processes:
                p.join(5)
                self.assertEqual(p.exitcode, 0)
            self.assertEqual(set(json.loads(self.path.read_text())), expected)

    def test_worker_lock_overlap_and_timeout(self):
        with cli.locked(self.lock):
            child = multiprocessing.get_context('spawn').Process(target=concurrent_change, args=(str(self.path), str(self.lock), 'add', B))
            child.start()
            time.sleep(0.15)
            self.assertEqual(json.loads(self.path.read_text()), [A])
            with self.assertRaises(RuntimeError):
                with cli.locked(self.lock, timeout=0.05):
                    pass
        child.join(5)
        self.assertEqual(child.exitcode, 0)
        self.assertEqual(json.loads(self.path.read_text()), [A, B])


if __name__ == '__main__':
    unittest.main()
