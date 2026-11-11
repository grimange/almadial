"""Rootless file-ownership regressions; no Asterisk/firewall commands execute."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'media_native', Path(__file__).parents[1] / 'lab/m3_05_native.py')
native = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(native)


class OwnedWrapperTests(unittest.TestCase):
    def test_replace_preserves_metadata_and_relabels(self):
        with tempfile.TemporaryDirectory() as root:
            wrapper = Path(root, 'pjsip.conf')
            wrapper.write_bytes(b'original')
            wrapper.chmod(0o640)
            metadata = wrapper.stat()
            with mock.patch.object(native, 'WRAPPER', wrapper), mock.patch.object(native, 'run') as command:
                native.replace_wrapper(b'original', b'original\n#include fixture\n')
            self.assertEqual(wrapper.read_bytes(), b'original\n#include fixture\n')
            self.assertEqual(wrapper.stat().st_mode, metadata.st_mode)
            self.assertEqual(wrapper.stat().st_uid, metadata.st_uid)
            self.assertEqual(wrapper.stat().st_gid, metadata.st_gid)
            command.assert_called_once_with(['/usr/sbin/restorecon', str(wrapper)])
            self.assertEqual(list(Path(root).iterdir()), [wrapper])

    def test_foreign_change_is_never_overwritten(self):
        with tempfile.TemporaryDirectory() as root:
            wrapper = Path(root, 'pjsip.conf')
            wrapper.write_bytes(b'operator change')
            with mock.patch.object(native, 'WRAPPER', wrapper), mock.patch.object(native, 'run') as command:
                with self.assertRaisesRegex(AssertionError, 'concurrently'):
                    native.replace_wrapper(b'original', b'replacement')
            self.assertEqual(wrapper.read_bytes(), b'operator change')
            command.assert_not_called()

    def test_symlink_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            target = Path(root, 'operator')
            target.write_bytes(b'operator')
            wrapper = Path(root, 'pjsip.conf')
            wrapper.symlink_to(target)
            with mock.patch.object(native, 'WRAPPER', wrapper), mock.patch.object(native, 'run') as command:
                with self.assertRaisesRegex(AssertionError, 'symlink'):
                    native.replace_wrapper(b'operator', b'replacement')
            self.assertEqual(target.read_bytes(), b'operator')
            command.assert_not_called()
