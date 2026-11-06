"""Safety tests; mocked database/native commands are not live acceptance."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import tempfile
import time
import unittest
from unittest.mock import patch
import wave

spec = importlib.util.spec_from_file_location('finalize', Path(__file__).with_name('finalize.py'))
f = importlib.util.module_from_spec(spec)
spec.loader.exec_module(f)


class RecordingTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.raw, self.done, self.state = [self.base / n for n in ('raw', 'done', 'state')]
        for p in (self.raw, self.done, self.state):
            p.mkdir()
        self.row = dict(id=999, name='PR01_SYNTHETIC', start=1, end=2, length=1, location='')
        self.input = self.raw / 'PR01_SYNTHETIC-in.wav'
        self.output = self.raw / 'PR01_SYNTHETIC-out.wav'
        self.final = self.done / 'PR01_SYNTHETIC-all.wav'
        self.wav(self.input, 8000)
        self.wav(self.output, 0)
        self.before = [p.read_bytes() for p in (self.input, self.output)]
        self.db = dict(self.row)
        self.commands = []
        self.addCleanup(patch.stopall)
        patch.object(f, 'command', side_effect=self.command).start()
        patch.object(f, 'rows', side_effect=lambda *a: [dict(self.db)]).start()
        patch.object(f.shutil, 'disk_usage', return_value=shutil._ntuple_diskusage(10**12, 0, 10**12)).start()
        # Exercise real writable descriptors owned by this test process without
        # requiring access to unrelated users' /proc entries on the desktop.
        # The root production wrapper still scans every process, unchanged.
        real_glob = Path.glob
        patch.object(Path, 'glob', lambda path, pattern:
                     iter([Path('/proc') / str(os.getpid())])
                     if path == Path('/proc') and pattern == '[0-9]*'
                     else real_glob(path, pattern)).start()

    def wav(self, path, frames):
        with wave.open(str(path), 'wb') as w:
            w.setparams((1, 2, 8000, 0, 'NONE', 'uncompressed'))
            w.writeframes(b'\x01\x00' * frames)
        os.utime(path, (time.time() - 120, time.time() - 120))

    def command(self, argv, **kwargs):
        self.commands.append(argv)
        if argv[0].endswith('mariadb'):
            return '1\n'
        if argv[0].endswith('perl'):
            stage = Path(next(x.split('=', 1)[1] for x in argv if x.startswith('-PATHmonitor=')))
            shutil.move(stage / self.input.name, self.final)
            (stage / self.output.name).unlink()
            self.db['location'] = f.ORIGIN + self.final.name
        return ''

    def process(self):
        return f.process(self.row, self.state, self.raw, self.done)

    def originals_intact(self):
        self.assertEqual([p.read_bytes() for p in (self.input, self.output)], self.before)

    def test_finalize_header_only_companion_and_native_https(self):
        self.assertEqual(self.process(), 'finalized_private')
        self.originals_intact()
        self.assertEqual(self.final.read_bytes(), self.before[0])
        self.assertEqual(self.final.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.db['location'], f.ORIGIN + self.final.name)
        self.assertIn('--HTTPS', next(a for a in self.commands if a[0].endswith('perl')))
        self.assertEqual(json.loads((self.state / '999.json').read_text())['seconds'], 1)

    def test_repeated_run_skips_native(self):
        self.process()
        self.row = dict(self.db)
        self.assertEqual(self.process(), 'already_finalized')
        self.assertEqual(sum(a[0].endswith('perl') for a in self.commands), 1)

    def test_active_metadata_waits_without_native_or_database_write(self):
        self.row['end'] = 0
        self.assertEqual(self.process(), 'waiting_for_closed_recording')
        self.assertFalse(self.commands)

    def test_recent_end_waits(self):
        self.row['end'] = time.time() - 5
        self.assertEqual(self.process(), 'waiting_for_closed_recording')

    def test_open_writer_is_detected_and_deferred(self):
        with self.input.open('ab'):
            self.assertTrue(f.writer_open([self.input]))
            self.assertEqual(self.process(), 'waiting_for_writer')

    def test_read_only_descriptor_does_not_block(self):
        with self.input.open('rb'):
            self.assertFalse(f.writer_open([self.input]))

    def test_recent_file_is_deferred(self):
        os.utime(self.input, None)
        self.assertEqual(self.process(), 'waiting_for_writer')

    def test_nonempty_output_is_not_discarded(self):
        self.wav(self.output, 100)
        with self.assertRaisesRegex(f.Refused, 'nonempty_output'):
            self.process()
        self.assertFalse(self.final.exists())

    def test_duration_mismatch_is_rejected(self):
        self.row['length'] = 60
        with self.assertRaisesRegex(f.Refused, 'duration_mismatch'):
            self.process()

    def test_empty_input_is_rejected(self):
        self.wav(self.input, 0)
        with self.assertRaisesRegex(f.Refused, 'empty_input'):
            self.process()

    def test_truncated_wav_is_rejected(self):
        data = self.input.read_bytes()
        self.input.write_bytes(data[:-100])
        with self.assertRaisesRegex(f.Refused, 'truncated'):
            f.audio(self.input)

    def test_symlink_is_rejected(self):
        target = self.base / 'target'
        self.input.rename(target)
        self.input.symlink_to(target)
        with self.assertRaisesRegex(f.Refused, 'nonregular'):
            self.process()

    def test_hardlink_is_rejected(self):
        os.link(self.input, self.base / 'other')
        with self.assertRaisesRegex(f.Refused, 'linked'):
            self.process()

    def test_existing_location_is_not_replaced(self):
        self.row['location'] = 'https://example.invalid/other'
        with self.assertRaisesRegex(f.Refused, 'location_conflict'):
            self.process()

    def test_existing_final_collision_is_rejected(self):
        self.final.write_bytes(b'existing')
        with self.assertRaisesRegex(f.Refused, 'final_collision'):
            self.process()

    def test_duplicate_database_filename_is_rejected(self):
        f.command.side_effect = lambda *a, **k: '2\n'
        with self.assertRaisesRegex(f.Refused, 'duplicate_recording'):
            self.process()

    def test_native_nonzero_preserves_originals_and_has_no_receipt(self):
        def fail(argv, **kwargs):
            if argv[0].endswith('perl'):
                raise f.Refused('command_failed')
            return self.command(argv, **kwargs)
        f.command.side_effect = fail
        with self.assertRaisesRegex(f.Refused, 'command_failed'):
            self.process()
        self.originals_intact()
        self.assertFalse((self.state / '999.json').exists())

    def test_false_native_success_without_file_is_rejected(self):
        f.command.side_effect = lambda *a, **k: '1\n'
        with self.assertRaises(FileNotFoundError):
            self.process()
        self.originals_intact()

    def test_database_update_failure_is_not_success(self):
        def fail_db(argv, **kwargs):
            result = self.command(argv, **kwargs)
            if argv[0].endswith('perl'):
                self.db['location'] = ''
            return result
        f.command.side_effect = fail_db
        with self.assertRaisesRegex(f.Refused, 'database_update_not_confirmed'):
            self.process()
        self.originals_intact()
        self.assertFalse((self.state / '999.json').exists())

    def test_metadata_race_prevents_native_run(self):
        f.rows.return_value = []
        f.rows.side_effect = None
        with self.assertRaisesRegex(f.Refused, 'metadata_changed'):
            self.process()
        self.assertFalse(any(a[0].endswith('perl') for a in self.commands))

    def test_capacity_check_preserves_originals(self):
        f.shutil.disk_usage.return_value = shutil._ntuple_diskusage(1, 1, 0)
        with self.assertRaisesRegex(f.Refused, 'capacity'):
            self.process()
        self.originals_intact()

    def test_receipt_conflict_is_rejected(self):
        self.process()
        self.row = dict(self.db)
        (self.state / '999.json').write_text('{}')
        with self.assertRaisesRegex(f.Refused, 'receipt_conflict'):
            self.process()


class QueryTests(unittest.TestCase):
    def test_unsafe_name_rejected(self):
        with patch.object(f, 'command', return_value='1\t../escape\t1\t2\t1\t\n'):
            with self.assertRaisesRegex(f.Refused, 'unsafe_recording_name'):
                f.rows()

    def test_query_is_read_only_bounded_and_excludes_historical(self):
        with patch.object(f, 'command', return_value='') as cmd:
            self.assertEqual(f.rows(boundary=123), [])
        sql = cmd.call_args.args[0][-1]
        self.assertTrue(sql.startswith('SELECT '))
        self.assertIn('start_epoch>=123', sql)
        self.assertIn('LIMIT 20', sql)
        self.assertIn('location IS NULL', sql)


if __name__ == '__main__':
    unittest.main()
