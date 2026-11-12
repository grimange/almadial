import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('recording_backup', Path(__file__).parents[1] / 'recording_backup.py')
b = importlib.util.module_from_spec(spec)
spec.loader.exec_module(b)


class RecordingBackupTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / 'source').mkdir()
        (self.root / 'source' / 'tone.wav').write_bytes(b'fake-audio-test-only')
        (self.root / 'source' / 'tone.wav').chmod(0o640)
        self.archive = self.root / 'backup.tar'
        b.create(self.root / 'source', ['tone.wav'], self.archive)

    def rewrite(self, transform):
        with tarfile.open(self.archive) as t:
            entries = [(m, t.extractfile(m).read()) for m in t.getmembers()]
        with tarfile.open(self.archive, 'w') as t:
            for m, data in transform(entries):
                m.size = len(data)
                t.addfile(m, io.BytesIO(data))

    def test_roundtrip_metadata_and_repeat_rejected(self):
        dest = self.root / 'restored'
        b.restore(self.archive, dest)
        self.assertEqual((dest / 'tone.wav').read_bytes(), b'fake-audio-test-only')
        self.assertEqual((dest / 'tone.wav').stat().st_mode & 0o777, 0o640)
        self.assertEqual(self.archive.stat().st_mode & 0o777, 0o600)
        with self.assertRaises(FileExistsError):
            b.restore(self.archive, dest)

    def test_corrupt_rejected_before_destination(self):
        self.rewrite(lambda entries: [(m, b'X' * len(data) if m.name.startswith('data/') else data)
                                      for m, data in entries])
        with self.assertRaisesRegex(ValueError, 'checksum'):
            b.restore(self.archive, self.root / 'out')
        self.assertFalse((self.root / 'out').exists())

    def test_missing_member_rejected(self):
        self.rewrite(lambda entries: entries[:1])
        with self.assertRaisesRegex(ValueError, 'missing'):
            b.verify(self.archive)

    def test_unsafe_path_rejected(self):
        def change(entries):
            entries[1][0].name = '../escape'
            return entries
        self.rewrite(change)
        with self.assertRaisesRegex(ValueError, 'unsafe'):
            b.verify(self.archive)

    def test_symlink_member_rejected(self):
        def change(entries):
            m, _ = entries[1]
            m.type, m.linkname = tarfile.SYMTYPE, '/etc/passwd'
            return entries[:1] + [(m, b'')]
        self.rewrite(change)
        with self.assertRaisesRegex(ValueError, 'unsafe'):
            b.verify(self.archive)

    def test_source_and_destination_symlinks_rejected(self):
        (self.root / 'link').symlink_to(self.root / 'source', target_is_directory=True)
        with self.assertRaisesRegex(ValueError, 'symlink'):
            b.create(self.root / 'link', ['tone.wav'], self.root / 'other.tar')
        with self.assertRaisesRegex(ValueError, 'symlink'):
            b.restore(self.archive, self.root / 'link' / 'out')

    def test_duplicate_member_rejected(self):
        self.rewrite(lambda entries: entries + [entries[1]])
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            b.verify(self.archive)

    def test_existing_archive_never_overwritten(self):
        before = self.archive.read_bytes()
        with self.assertRaises(FileExistsError):
            b.create(self.root / 'source', ['tone.wav'], self.archive)
        self.assertEqual(before, self.archive.read_bytes())
