"""Deterministic synthetic tests; no host or production-data access."""
import importlib.util
import json
from pathlib import Path
import sqlite3
import subprocess
import tarfile
import tempfile
import time
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('backup', Path(__file__).with_name('backup.py'))
b = importlib.util.module_from_spec(spec)
spec.loader.exec_module(b)


class BackupTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        (self.root / 'complete').mkdir()

    def tearDown(self):
        self.temp.cleanup()

    def fixture(self):
        root = self.root / 'complete' / 'fixture'
        root.mkdir()
        (root / 'database.sql').write_text('-- Current Database: `mysql`\nCREATE TABLE `global_priv` (x int);\n'
            '-- Current Database: `asterisk`\nCREATE TABLE `example` (x int);\n-- Dump completed on fixture\n')
        with sqlite3.connect(root / 'astdb.sqlite3') as db:
            db.execute('CREATE TABLE test (value INTEGER)')
        source = self.root / 'recording.wav'
        source.write_bytes(b'synthetic samples')
        with tarfile.open(root / 'files.tar', 'w') as archive:
            archive.add(source, arcname='var/spool/asterisk/monitor/test.wav')
        b.write_json(root / 'manifest.json', dict(archive_members=1,
            required_archive_members=['var/spool/asterisk/monitor/test.wav'],
            recording_files={'var/spool/asterisk/monitor/test.wav': {'sha256': b.digest(source)}}))
        b.write_json(root / 'inventory.json', dict(tables=[['mysql', 'global_priv', 'Aria', '1'],
                                                          ['asterisk', 'example', 'MyISAM', '0']]))
        self.sums(root)
        return root

    def sums(self, root):
        (root / 'SHA256SUMS').write_text(''.join(b.digest(root / p) + '  ' + p + '\n' for p in b.PAYLOADS))

    def test_complete_fixture_verifies(self):
        self.assertEqual(b.verify(self.fixture())['database_tables'], 2)

    def test_native_system_table_syntax_and_generated_sys_exclusion(self):
        root = self.fixture()
        p = root / 'database.sql'
        p.write_text(p.read_text().replace('CREATE TABLE `global_priv`', 'CREATE TABLE IF NOT EXISTS `global_priv`'))
        data = json.loads((root / 'inventory.json').read_text())
        data['tables'].append(['sys', 'sys_config', 'Aria', '6'])
        b.write_json(root / 'inventory.json', data)
        self.sums(root)
        self.assertEqual(b.verify(root)['database_tables'], 2)

    def test_corruption_rejected(self):
        root = self.fixture()
        with (root / 'files.tar').open('ab') as stream:
            stream.write(b'corrupt')
        with self.assertRaisesRegex(b.BackupError, 'checksum'):
            b.verify(root)

    def test_missing_account_recovery_rejected(self):
        root = self.fixture()
        path = root / 'database.sql'
        path.write_text(path.read_text().replace('`global_priv`', '`other`'))
        self.sums(root)
        with self.assertRaisesRegex(b.BackupError, 'account_recovery'):
            b.verify(root)

    def test_missing_table_definition_rejected(self):
        root = self.fixture()
        data = json.loads((root / 'inventory.json').read_text())
        data['tables'].append(['asterisk', 'missing', 'MyISAM', '0'])
        b.write_json(root / 'inventory.json', data)
        self.sums(root)
        with self.assertRaisesRegex(b.BackupError, 'table_definition'):
            b.verify(root)

    def test_incomplete_dump_rejected(self):
        root = self.fixture()
        (root / 'database.sql').write_text('-- incomplete')
        self.sums(root)
        with self.assertRaisesRegex(b.BackupError, 'incomplete'):
            b.verify(root)

    def test_recording_content_verified_against_source(self):
        root = self.fixture()
        data = json.loads((root / 'manifest.json').read_text())
        data['recording_files']['var/spool/asterisk/monitor/test.wav']['sha256'] = '0' * 64
        b.write_json(root / 'manifest.json', data)
        self.sums(root)
        with self.assertRaisesRegex(b.BackupError, 'recording_content'):
            b.verify(root)

    def test_required_custom_file_must_be_present(self):
        root = self.fixture()
        data = json.loads((root / 'manifest.json').read_text())
        data['required_archive_members'].append('missing/custom.so')
        b.write_json(root / 'manifest.json', data)
        self.sums(root)
        with self.assertRaisesRegex(b.BackupError, 'required_recovery'):
            b.verify(root)

    def test_capacity_stops_at_cap_without_deleting(self):
        root = self.fixture()
        before = b.digest(root / 'files.tar')
        with self.assertRaisesRegex(b.BackupError, 'retention_cap'):
            b.check_capacity({'max_complete_backups': 1}, self.root, 0, 0)
        self.assertEqual(before, b.digest(root / 'files.tar'))

    def test_private_destination_rejects_symlink(self):
        link = self.root / 'link'
        link.symlink_to(self.root / 'complete', target_is_directory=True)
        with self.assertRaisesRegex(b.BackupError, 'symlink'):
            b.private_dir(link)

    def test_active_channels_abort_before_sql(self):
        with patch.object(b, 'run', return_value='1 active channels\n'), patch.object(b, 'sql') as sql:
            with self.assertRaisesRegex(b.BackupError, 'active_asterisk'):
                b.idle({})
            sql.assert_not_called()

    def test_active_agents_abort(self):
        with patch.object(b, 'run', return_value='0 active channels\n'), patch.object(b, 'sql', return_value='1\n0\n0\n0\n'):
            with self.assertRaisesRegex(b.BackupError, 'active_agents'):
                b.idle({})

    def test_native_dump_has_global_lock_and_timeout(self):
        with patch.object(b, 'run', return_value='') as run:
            b.dump_database({'socket': '/fake.sock', 'dump_timeout_seconds': 30}, self.root)
            argv = run.call_args.args[0]
            self.assertIn('--lock-all-tables', argv)
            self.assertIn('--all-databases', argv)
            self.assertNotIn('--single-transaction', argv)
            self.assertEqual(run.call_args.kwargs['timeout'], 30)

    def test_subprocess_timeout_is_bounded(self):
        start = time.monotonic()
        with self.assertRaises(subprocess.TimeoutExpired):
            b.run(['/usr/bin/python3', '-c', 'import time; time.sleep(30)'], timeout=0.1)
        self.assertLess(time.monotonic() - start, 2)

    def test_subprocess_error_does_not_echo_secret_output(self):
        with self.assertRaisesRegex(b.BackupError, '^python3_nonzero$'):
            b.run(['/usr/bin/python3', '-c', 'import sys; print("fake-private"); sys.exit(1)'])

    def test_health_preserves_success_and_reports_failure(self):
        success = dict(completed_at=b.utc(), id='old-good')
        b.write_json(self.root / 'status.json', dict(outcome='failed', last_success=success,
                                                   last_failure={'reason': 'fixture'}))
        result, code = b.health(self.root)
        self.assertEqual(code, 1)
        self.assertEqual(result['last_success'], success)

    def test_failed_capture_cleans_only_its_private_workspace(self):
        root = self.fixture()
        good = b.digest(root / 'files.tar')
        config = dict(required=[], optional=[], recording_roots=[])
        with patch.object(b, 'idle'), patch.object(b, 'scan', return_value={}), \
             patch.object(b, 'sql', side_effect=['', '0', '10.11.18', '']), \
             patch.object(b, 'check_capacity', return_value={}), patch.object(b, 'run', return_value='fixture'), \
             patch.object(b, 'dump_database', side_effect=b.BackupError('fixture_dump_failure')):
            with self.assertRaisesRegex(b.BackupError, 'fixture_dump_failure'):
                b.capture(config, self.root)
        self.assertEqual(list(self.root.glob('.partial-*')), [])
        self.assertEqual(b.digest(root / 'files.tar'), good)


if __name__ == '__main__':
    unittest.main()
