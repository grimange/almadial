"""Offline source preparation checks; no host effects or network access."""
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
SPEC = importlib.util.spec_from_file_location('prepare_compliance_source', ROOT / 'scripts/prepare_compliance_source.py')
SOURCE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SOURCE)


class SourceTests(unittest.TestCase):
    def test_coverage_is_current_and_explicit(self):
        data = SOURCE.coverage()
        self.assertEqual(data['total_components_evaluated'], 40)
        self.assertEqual(data['components_with_conflicting_evidence'], 1)
        self.assertGreater(data['components_with_explicit_exceptions'], 0)
        self.assertEqual(SOURCE.encoded(data), (SOURCE.DEFAULT / 'component-coverage.json').read_bytes())
        sip = next(c for c in data['components'] if c['name'] == 'SIP.js')
        self.assertEqual(sip['license'], 'MIT AND WTFPL')
        self.assertTrue(any('2015' in line and 'Cruz' in line for line in sip['copyright_evidence']))

    def test_unsafe_member_and_symlink_rejected(self):
        for name in ('../secret', '/etc/shadow', 'a/../../b'):
            with self.assertRaises(ValueError):
                SOURCE.safe(name)
        with tempfile.TemporaryDirectory() as tmp:
            link = Path(tmp) / 'link'
            link.symlink_to(ROOT / 'LICENSE')
            with self.assertRaises(ValueError):
                SOURCE.regular(link)

    def test_archive_hash_and_file_inventory(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'source.tar.gz'
            with tarfile.open(path, 'w:gz') as t:
                h = tarfile.TarInfo('source/LICENSE')
                h.size = 3
                t.addfile(h, io.BytesIO(b'MIT'))
            spec = dict(sha256=SOURCE.digest(path.read_bytes()), files=[dict(
                path='source/LICENSE', bytes=3, sha256=SOURCE.digest(b'MIT'))])
            self.assertEqual(SOURCE.archive_files(path, spec)[1]['source/LICENSE'], b'MIT')
            spec['sha256'] = '0' * 64
            with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
                SOURCE.archive_files(path, spec)
            spec['sha256'] = SOURCE.digest(path.read_bytes())
            spec['files'][0]['sha256'] = '0' * 64
            with self.assertRaisesRegex(ValueError, 'inventory mismatch'):
                SOURCE.archive_files(path, spec)

    def test_complete_embedded_md5_notice(self):
        notice = (SOURCE.DEFAULT / 'licenses/SIP.js-0.20.1/embedded-notices.txt').read_text()
        self.assertIn('Copyright (C) 2014 Raymond Hill', notice)
        self.assertIn('Copyright (C) 2015 André Cruz', notice)
        self.assertIn('Version 2, December 2004', notice)
        self.assertIn('WITHOUT WARRANTY', notice)


if __name__ == '__main__':
    unittest.main()
