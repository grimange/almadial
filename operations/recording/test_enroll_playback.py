"""Synthetic enrollment tests; no production DB or media access."""
import hashlib
import json
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch
import wave

import enroll_playback as e


class Enrollment(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.name = '20261107-024337_SYNTHETIC'
        self.file = self.root / (self.name + '-all.wav')
        with wave.open(str(self.file), 'wb') as wav:
            wav.setparams((1, 2, 8000, 0, 'NONE', 'not compressed'))
            wav.writeframes(b'\x01\x02' * 8000)
        self.file.chmod(0o600)
        self.receipt = dict(recording_id=2, seconds=1,
                            final_sha256=hashlib.sha256(self.file.read_bytes()).hexdigest())
        self.row = [2, self.name, 1, e.ORIGIN + self.file.name]
        self.patches = [patch.object(e, 'ROOT', self.root), patch.object(e, 'TARGET', self.root),
                        patch.object(e, 'ROOT_UID', os.getuid())]
        for p in self.patches:
            p.start()

    def tearDown(self):
        for p in self.patches:
            p.stop()
        self.tmp.cleanup()

    def test_idempotent_projection_and_recovery_permissions(self):
        self.assertEqual(e.enroll(self.row, self.receipt, os.getgid()), 'enrolled')
        metadata = json.loads((self.root / '2.json').read_text())
        self.assertEqual(metadata['original'], dict(gid=os.getgid(), mode=0o600))
        self.assertEqual(self.file.stat().st_mode & 0o777, 0o640)
        self.assertEqual(e.enroll(self.row, self.receipt, os.getgid()), 'already_enrolled')
        self.assertEqual(json.loads((self.root / '2.json').read_text()), metadata)
        # Rehearse permission rollback only against this synthetic inode.
        self.file.chmod(metadata['original']['mode'])
        self.assertEqual(self.file.stat().st_mode & 0o777, 0o600)
        self.assertEqual(hashlib.sha256(self.file.read_bytes()).hexdigest(), self.receipt['final_sha256'])

    def test_failure_preserves_audio_and_permissions(self):
        for receipt in ({**self.receipt, 'final_sha256': 'bad'}, {**self.receipt, 'seconds': 9}):
            with self.assertRaises(ValueError):
                e.enroll(self.row, receipt, os.getgid())
            self.assertFalse((self.root / '2.json').exists())
            self.assertEqual(self.file.stat().st_mode & 0o777, 0o600)
            self.assertEqual(hashlib.sha256(self.file.read_bytes()).hexdigest(), self.receipt['final_sha256'])

    def test_rejects_unfinalized_mismapped_symlink_and_hardlink(self):
        for row in ([2, self.name, 0, self.row[3]], [2, '../unsafe', 1, self.row[3]],
                    [2, self.name, 1, 'http://example.invalid/audio']):
            with self.assertRaises(ValueError):
                e.enroll(row, self.receipt, os.getgid())
        target = self.root / 'other'
        self.file.rename(target)
        self.file.symlink_to(target)
        with self.assertRaises(OSError):
            e.enroll(self.row, self.receipt, os.getgid())
        self.file.unlink()
        os.link(target, self.file)
        with self.assertRaises(ValueError):
            e.enroll(self.row, self.receipt, os.getgid())

    def test_receipt_directory_traversal_under_private_umask(self):
        state = self.root / 'private-state'
        state.mkdir()
        target = self.root / 'playback-receipts'
        previous_umask = os.umask(0o077)
        try:
            with patch.object(e, 'STATE', state), patch.object(e, 'TARGET', target), \
                    patch.object(e.os, 'geteuid', return_value=0), patch.object(e.os, 'chown'), \
                    patch.object(e.grp, 'getgrnam', return_value=types.SimpleNamespace(gr_gid=os.getgid())):
                e.main()
            self.assertEqual(target.stat().st_mode & 0o777, 0o750)
        finally:
            os.umask(previous_umask)


if __name__ == '__main__':
    unittest.main()
