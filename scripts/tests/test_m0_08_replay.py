"""Offline stop-boundary tests; these never establish VM reproducibility."""
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'lab'))
import m0_08_replay as replay


class ReplayBoundaries(unittest.TestCase):
    def test_wrong_host_stops_before_input_reads_or_mutations(self):
        with mock.patch.object(sys, 'argv', ['replay', 'packages']), \
             mock.patch.object(replay.os, 'geteuid', return_value=0), \
             mock.patch.object(replay.subprocess, 'check_output', return_value='reference\n'), \
             mock.patch.object(replay, 'check_inputs') as inputs, \
             mock.patch.object(replay, 'packages') as packages:
            with self.assertRaises(SystemExit):
                replay.main()
            inputs.assert_not_called()
            packages.assert_not_called()

    def test_changed_source_stops_before_installation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'source').write_bytes(b'changed bytes')
            (root / 'inputs.json').write_text(json.dumps({'source': hashlib.sha256(b'accepted').hexdigest()}))
            with mock.patch.object(replay, 'INPUT', root):
                with self.assertRaisesRegex(ValueError, 'input mismatch'):
                    replay.check_inputs()

    def test_input_paths_cannot_escape_bundle(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('../outside', '/etc/shadow'):
                (root / 'inputs.json').write_text(json.dumps({name: '0' * 64}))
                with mock.patch.object(replay, 'INPUT', root):
                    with self.assertRaisesRegex(ValueError, 'plain filename'):
                        replay.check_inputs()

    def test_failed_command_is_recorded_and_stops(self):
        with tempfile.TemporaryDirectory() as directory:
            with mock.patch.object(replay, 'ROOT', Path(directory)), \
                 mock.patch.object(replay.subprocess, 'run', return_value=mock.Mock(
                     returncode=1, stdout='', stderr='fixture failure')):
                with self.assertRaisesRegex(RuntimeError, 'failed'):
                    replay.run('failed', ['false'], data='private input')
            record = json.loads((Path(directory) / 'failed.json').read_text())
            self.assertEqual(record['exit_code'], 1)
            self.assertNotIn('private input', json.dumps(record))

    def test_retry_preserves_existing_command_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'previous.json').write_text('preserved')
            with mock.patch.object(replay, 'ROOT', root), \
                 mock.patch.object(replay.subprocess, 'run') as command:
                with self.assertRaises(FileExistsError):
                    replay.run('previous', ['false'])
                command.assert_not_called()
            self.assertEqual((root / 'previous.json').read_text(), 'preserved')
