"""Rootless behavioral tests of the fixed S05 file adapter."""
import hashlib
import base64
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

MODULE = Path(__file__).resolve().parents[2] / 'internal/provision/vicidial_files.py'
spec = importlib.util.spec_from_file_location('m205_files', MODULE)
adapter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(adapter)


class Files(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.patches = [patch.object(adapter, 'ROOT', self.root),
                        patch.object(adapter, 'ROOT_UID', os.getuid()),
                        patch.object(adapter, 'uid', lambda _: os.getuid()),
                        patch.object(adapter, 'gid', lambda _: os.getgid()),
                        patch.object(adapter, 'label', lambda _: 'httpd_sys_content_t'),
                        patch.object(adapter, 'set_label', lambda *_: None)]
        for p in self.patches:
            p.start()
        for p in ('var/www/html', 'usr/share', 'var/lib/almadial/vicidial-s05'):
            (self.root / p).mkdir(parents=True)
        # The caller's umask must not turn fixture ancestors into unsafe
        # group-writable directories. Production correctly refuses those.
        for p in self.root.rglob('*'):
            if p.is_dir():
                p.chmod(0o755)
        self.raw = b'synthetic application fixture, never executable'
        self.file = {'Source': 'www/agc/fixture.php',
                     'Target': '/var/www/html/agc/fixture.php',
                     'SourceSHA256': adapter.sha(self.raw), 'SHA256': adapter.sha(self.raw),
                     'Owner': 'root', 'Group': 'apache', 'Mode': 0o640,
                     'Label': 'httpd_sys_content_t', 'Writer': 'upstream-r4037'}
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, 'w') as z:
            z.writestr('2026-11-05/' + self.file['Source'], self.raw)
        self.archive = archive.getvalue()
        target = self.root / adapter.ARCHIVE.lstrip('/')
        target.write_bytes(self.archive)
        target.chmod(0o600)
        self.pin = patch.object(adapter, 'ARCHIVE_SHA', hashlib.sha256(self.archive).hexdigest())
        self.pin.start()

    def tearDown(self):
        self.pin.stop()
        for p in reversed(self.patches):
            p.stop()
        self.tmp.cleanup()

    def test_agi_modules_use_exact_versioned_runtime_paths(self):
        for target in adapter.AGI_MODULES:
            self.assertTrue(adapter.target_ok(target))
        self.assertFalse(adapter.target_ok('/usr/local/share/perl5/5.40/Asterisk/Foreign.pm'))
        self.assertIn('/usr/local/share/perl5/5.40/Asterisk', adapter.OWNED_DIRECTORIES)

    def test_runtime_receipt_requires_complete_identity_and_preserves_bytes(self):
        work = self.root / adapter.WORK.lstrip('/')
        work.chmod(0o700)
        components = {name: 'a' * 64 for name in
                      ('Files', 'Perl', 'Configuration', 'Registration', 'Generated', 'Web', 'Inputs')}
        p = {'Receipt': {'Binding': 'b' * 64, 'Components': components}}
        self.assertEqual(adapter.runtime_receipt(p), {'State': 'absent'})
        self.assertFalse((work / 'runtime.json').exists())
        result = adapter.runtime_receipt(p, True)
        inode = (work / 'runtime.json').stat().st_ino
        self.assertEqual(adapter.runtime_receipt(p), result)
        self.assertEqual(adapter.runtime_receipt(p, True), result)
        self.assertEqual((work / 'runtime.json').stat().st_ino, inode)
        components['Inputs'] = 'c' * 64
        with self.assertRaises(ValueError):
            adapter.runtime_receipt(p, True)
        del components['Files']
        with self.assertRaises(ValueError):
            adapter.runtime_receipt(p)

    def test_deferred_jobs_inspect_actual_processes_and_cron(self):
        process = self.root / 'proc/12345'
        process.mkdir(parents=True)
        (process / 'cmdline').write_bytes(b'/usr/sbin/mariadbd\0')
        cron = self.root / 'etc/cron.d'
        cron.mkdir(parents=True)
        for directory in (process, process.parent, cron, cron.parent):
            directory.chmod(0o755)
        job = cron / 'maintenance'
        job.write_bytes(b'# AST_keepalive deferred\n0 0 * * * root /usr/bin/true\n')
        self.assertEqual(adapter.deferred_inspect(), {'Inactive': True})
        for raw in (b'/usr/bin/perl\0/usr/share/astguiclient/AST_update.pl\0',
                    b'/usr/bin/screen\0ADMIN_keepalive_ALL.pl\0'):
            (process / 'cmdline').write_bytes(raw)
            with self.assertRaises(ValueError):
                adapter.deferred_inspect()
        (process / 'cmdline').write_bytes(b'/usr/sbin/asterisk\0')
        job.write_bytes(b'* * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl\n')
        with self.assertRaises(ValueError):
            adapter.deferred_inspect()
        self.assertIn(b'ADMIN_keepalive_ALL', job.read_bytes())
        job.unlink()
        job.symlink_to(self.root / 'outside')
        with self.assertRaises(OSError):
            adapter.deferred_inspect()

    def generation_fixture(self):
        work = self.root / adapter.WORK.lstrip('/')
        work.chmod(0o700)
        generated = work / 'generated'
        generated.mkdir(mode=0o700)
        base = self.root / 'etc/asterisk'
        base.mkdir(parents=True, mode=0o750)
        base.parent.chmod(0o755)
        for name, (before, _) in adapter.WRAPPERS.items():
            if before:
                (base / name).write_bytes(before)
                (base / name).chmod(0o640)
        for name in adapter.MODERN:
            (generated / name).write_bytes(b'; synthetic output, not runtime proof\n[fixture]\n')
            (generated / name).chmod(0o600)
        return work, generated, base, {'Binding': 'a' * 64, 'Inputs': 'b' * 64}

    def test_generation_publication_preimages_repeat_and_foreign_edit(self):
        work, generated, base, request = self.generation_fixture()
        with patch.object(adapter, 'label', lambda _: 'etc_t'):
            self.assertEqual(adapter.generation_state(request), {'State': 'absent', 'Wrapped': False})
            self.assertFalse((work / 'generation.json').exists())
            result = adapter.generation_state(request, True)
            self.assertEqual(len(result['State']), 64)
            before = (base / 'extensions.conf').stat()
            self.assertEqual(adapter.generation_state(request), result)
            self.assertEqual(adapter.generation_state(request, True), result)
            self.assertEqual((base / 'extensions.conf').stat().st_ino, before.st_ino)
            for name, (original, extra) in adapter.WRAPPERS.items():
                self.assertEqual((base / name).read_bytes(), original + extra)
                self.assertEqual((work / ('before-' + name)).read_bytes(), original)
            for name in ('sip-vicidial.conf', 'iax-vicidial.conf', 'voicemail.conf'):
                self.assertFalse((base / name).exists())
            (base / 'pjsip-vicidial.conf').write_bytes(b'; operator edit')
            with self.assertRaises(ValueError):
                adapter.generation_state(request, True)
            self.assertEqual((base / 'pjsip-vicidial.conf').read_bytes(), b'; operator edit')

    def test_generation_conflicts_fail_before_publication(self):
        for defect in ('wrapper', 'destination', 'missing', 'symlink', 'directive', 'transport', 'metadata'):
            with self.subTest(defect=defect), tempfile.TemporaryDirectory() as isolated:
                oldroot = self.root
                self.root = Path(isolated)
                for name in ('var/lib/almadial/vicidial-s05',):
                    (self.root / name).mkdir(parents=True)
                for path in self.root.rglob('*'):
                    if path.is_dir(): path.chmod(0o755)
                try:
                    with patch.object(adapter, 'ROOT', self.root), patch.object(adapter, 'label', lambda _: 'etc_t'):
                        work, generated, base, request = self.generation_fixture()
                        if defect == 'wrapper': (base / 'manager.conf').write_bytes(b'; operator include')
                        if defect == 'destination':
                            (base / adapter.MODERN[0]).write_bytes(b'operator data')
                            (base / adapter.MODERN[0]).chmod(0o640)
                        candidate = generated / adapter.MODERN[1]
                        if defect == 'missing': candidate.unlink()
                        if defect == 'symlink':
                            candidate.unlink(); candidate.symlink_to(generated / adapter.MODERN[0])
                        if defect == 'directive': candidate.write_bytes(b'#exec malicious\n')
                        if defect == 'transport': candidate.write_bytes(b'[transport]\ntype=transport\n')
                        if defect == 'metadata': candidate.chmod(0o644)
                        with self.assertRaises((ValueError, OSError)):
                            adapter.generation_state(request, True)
                        self.assertFalse((work / 'publication-intent.json').exists())
                        self.assertFalse((base / adapter.MODERN[3]).exists())
                finally:
                    self.root = oldroot

    def test_generation_interruption_blocks_automatic_replay(self):
        work, generated, base, request = self.generation_fixture()
        publish = adapter.publish_at
        calls = []
        def interrupted(parent, name, *args, **kwargs):
            if name == adapter.MODERN[1]:
                raise OSError('fixture interruption')
            calls.append(name)
            return publish(parent, name, *args, **kwargs)
        with patch.object(adapter, 'label', lambda _: 'etc_t'):
            with patch.object(adapter, 'publish_at', interrupted):
                with self.assertRaises(OSError): adapter.generation_state(request, True)
            self.assertTrue((work / 'publication-intent.json').is_file())
            self.assertFalse((work / 'generation.json').exists())
            for mutate in (False, True):
                with self.assertRaises(ValueError): adapter.generation_state(request, mutate)
            self.assertEqual((base / adapter.MODERN[0]).read_bytes(), (generated / adapter.MODERN[0]).read_bytes())

    def test_fresh_rerun_and_preserved_data(self):
        data = self.root / 'var/www/html/operator.data'
        data.write_bytes(b'preserve this operator file')
        before = data.stat()
        self.assertEqual(adapter.install([self.file]), {'Installed': 1, 'Preserved': 0})
        path = self.root / self.file['Target'].lstrip('/')
        first = path.stat()
        self.assertEqual(path.read_bytes(), self.raw)
        self.assertEqual(adapter.install([self.file]), {'Installed': 0, 'Preserved': 1})
        self.assertEqual((first.st_ino, first.st_mtime_ns), (path.stat().st_ino, path.stat().st_mtime_ns))
        self.assertEqual((before.st_ino, before.st_mtime_ns), (data.stat().st_ino, data.stat().st_mtime_ns))
        self.assertEqual(data.read_bytes(), b'preserve this operator file')

    def test_partial_missing_file_completion(self):
        adapter.install([self.file])
        second = dict(self.file, Target='/var/www/html/agc/other.php')
        self.assertEqual(adapter.install([self.file, second]), {'Installed': 1, 'Preserved': 1})

    def owned_perl(self):
        raw = b'synthetic dependency bytes, never executable'
        return [dict(Target=p, Owner='root', Group='root', Mode=0o644,
                     Label='usr_t', SHA256=adapter.sha(raw),
                     Content=base64.b64encode(raw).decode())
                for p in adapter.OWNED_PATHS['perl']]

    def test_owned_dependency_fresh_partial_repeat_and_conflict(self):
        (self.root / 'usr/local/share').mkdir(parents=True, mode=0o755)
        (self.root / 'usr/local').chmod(0o755)
        (self.root / 'usr/share/doc').mkdir(mode=0o755)
        files = self.owned_perl()
        with patch.object(adapter, 'label', lambda _: 'usr_t'):
            initial = adapter.owned(files, 'perl', False)
            self.assertEqual(initial['Missing'], list(adapter.OWNED_PATHS['perl']))
            self.assertFalse((self.root / 'usr/local/share/perl5').exists())
            adapter.owned(files, 'perl', True)
            before = (self.root / files[0]['Target'].lstrip('/')).stat()
            self.assertEqual(adapter.owned(files, 'perl', False)['Verified'], 2)
            adapter.owned(files, 'perl', True)
            after = (self.root / files[0]['Target'].lstrip('/')).stat()
            self.assertEqual((before.st_ino, before.st_mtime_ns), (after.st_ino, after.st_mtime_ns))
            notice = self.root / files[1]['Target'].lstrip('/')
            notice.unlink()
            self.assertEqual(adapter.owned(files, 'perl', False)['Verified'], 1)
            adapter.owned(files, 'perl', True)
            notice.write_bytes(b'operator-owned replacement')
            with self.assertRaises(ValueError):
                adapter.owned(files, 'perl', True)
            self.assertEqual(notice.read_bytes(), b'operator-owned replacement')

    def test_owned_component_rejects_foreign_paths_and_metadata_before_write(self):
        for change in ('path', 'mode', 'digest', 'group', 'context', 'duplicate', 'missing'):
            with self.subTest(change=change):
                files = self.owned_perl()
                if change == 'path': files[0]['Target'] = '/etc/passwd'
                if change == 'mode': files[0]['Mode'] = 0o777
                if change == 'digest': files[0]['SHA256'] = '0' * 64
                if change == 'group': files[0]['Group'] = 'apache'
                if change == 'context': files[0]['Label'] = 'unconfined_t'
                if change == 'duplicate': files[1] = files[0]
                if change == 'missing': files.pop()
                with self.assertRaises(ValueError):
                    adapter.owned(files, 'perl', True)
                self.assertFalse((self.root / 'usr/local/share/perl5').exists())

    def ami_create(self, secret):
        output, errors = io.StringIO(), io.StringIO()
        with patch.object(adapter.sys, 'argv', ['adapter', 'ami-create']), \
                patch.object(adapter.sys, 'stdin', io.StringIO(json.dumps({'AMI': secret}))), \
                patch.object(adapter.sys, 'stdout', output), \
                patch.object(adapter.sys, 'stderr', errors):
            code = adapter.main()
        self.assertNotIn(secret, output.getvalue() + errors.getvalue())
        return code, output.getvalue()

    def secret_directory(self):
        parent = self.root / 'etc/almadial/secrets'
        parent.mkdir(parents=True, mode=0o700)
        (self.root / 'etc').chmod(0o755)
        (self.root / 'etc/almadial').chmod(0o700)
        return parent

    def test_ami_secret_creation_preserves_existing_credential(self):
        parent = self.secret_directory()
        secret = 'a' * 20
        code, output = self.ami_create(secret)
        self.assertEqual(code, 0)
        self.assertEqual(json.loads(output), {'Created': True})
        path = parent / 'vicidial-ami'
        before = path.stat()
        self.assertEqual(path.read_bytes(), (secret + '\n').encode())
        self.assertEqual(before.st_mode & 0o777, 0o600)
        self.assertEqual(self.ami_create('b' * 20)[0], 1)
        after = path.stat()
        self.assertEqual((before.st_ino, before.st_mtime_ns), (after.st_ino, after.st_mtime_ns))
        self.assertEqual(path.read_bytes(), (secret + '\n').encode())

    def test_ami_missing_credential_with_configuration_blocks_creation(self):
        parent = self.secret_directory()
        for kind in ('regular', 'symlink'):
            with self.subTest(kind=kind):
                config = self.root / 'etc/astguiclient.conf'
                if kind == 'regular':
                    config.write_bytes(b'existing operator configuration')
                else:
                    config.symlink_to('/does/not/exist')
                self.assertEqual(self.ami_create('a' * 20)[0], 1)
                self.assertFalse((parent / 'vicidial-ami').exists())
                config.unlink()

    def test_ami_invalid_secret_and_unsafe_leaf_are_rejected(self):
        parent = self.secret_directory()
        for secret in ('short', 'a' * 19 + '\n', 'z' * 20):
            self.assertEqual(self.ami_create(secret)[0], 1)
            self.assertFalse((parent / 'vicidial-ami').exists())
        victim = self.root / 'operator-secret'
        victim.write_bytes(b'preserve')
        (parent / 'vicidial-ami').symlink_to(victim)
        self.assertEqual(self.ami_create('a' * 20)[0], 1)
        self.assertEqual(victim.read_bytes(), b'preserve')

    def test_label_failure_does_not_publish_final_inode(self):
        with patch.object(adapter, 'set_label', side_effect=OSError('synthetic denial')):
            with self.assertRaises(OSError):
                adapter.install([self.file])
        target = self.root / self.file['Target'].lstrip('/')
        self.assertFalse(target.exists())
        self.assertFalse(list(target.parent.glob('.almadial-s05-*')))
        self.assertEqual(adapter.install([self.file])['Installed'], 1)

    def test_foreign_edit_blocks_all_writes(self):
        adapter.install([self.file])
        path = self.root / self.file['Target'].lstrip('/')
        path.write_bytes(b'operator content')
        missing = dict(self.file, Target='/var/www/html/agc/not-created.php')
        with self.assertRaises(ValueError):
            adapter.install([missing, self.file])
        self.assertFalse((self.root / missing['Target'].lstrip('/')).exists())
        self.assertEqual(path.read_bytes(), b'operator content')

    def test_wrong_mode_is_preserved(self):
        adapter.install([self.file])
        path = self.root / self.file['Target'].lstrip('/')
        path.chmod(0o666)
        with self.assertRaises(ValueError):
            adapter.install([self.file])
        self.assertEqual(path.stat().st_mode & 0o777, 0o666)

    def test_wrong_owner(self):
        adapter.install([self.file])
        with patch.object(adapter, 'uid', lambda _: os.getuid() + 123):
            with self.assertRaises(ValueError):
                adapter.observation(self.file)

    def test_parent_symlink(self):
        (self.root / 'var/www/html/agc').symlink_to(self.root / 'usr/share', target_is_directory=True)
        with self.assertRaises(OSError):
            adapter.install([self.file])
        self.assertFalse((self.root / 'usr/share/fixture.php').exists())

    def test_leaf_symlink(self):
        (self.root / 'var/www/html/agc').mkdir(mode=0o755)
        victim = self.root / 'victim'
        victim.write_bytes(b'unchanged')
        (self.root / self.file['Target'].lstrip('/')).symlink_to(victim)
        with self.assertRaises(OSError):
            adapter.install([self.file])
        self.assertEqual(victim.read_bytes(), b'unchanged')

    def test_hardlink_and_fifo_rejected(self):
        adapter.install([self.file])
        path = self.root / self.file['Target'].lstrip('/')
        os.link(path, self.root / 'alias')
        with self.assertRaises(ValueError):
            adapter.observation(self.file)
        path.unlink()
        os.mkfifo(path, 0o640)
        with self.assertRaises(ValueError):
            adapter.observation(self.file)

    def test_archive_changed(self):
        (self.root / adapter.ARCHIVE.lstrip('/')).write_bytes(b'corrupt')
        with self.assertRaises(ValueError):
            adapter.install([self.file])
        self.assertFalse((self.root / self.file['Target'].lstrip('/')).exists())

    def test_derived_content_changed(self):
        wrong = dict(self.file, SHA256=adapter.sha(b'unreviewed modification'))
        with self.assertRaises(ValueError):
            adapter.install([wrong])

    def test_source_inventory_changed(self):
        wrong = dict(self.file, SourceSHA256=adapter.sha(b'wrong source'))
        with self.assertRaises(ValueError):
            adapter.install([wrong])

    def test_no_overwrite_race(self):
        adapter.install([self.file])
        with self.assertRaises(FileExistsError):
            adapter.create(self.file, b'forbidden overwrite')
        self.assertEqual((self.root / self.file['Target'].lstrip('/')).read_bytes(), self.raw)
        self.assertFalse(list((self.root / 'var/www/html/agc').glob('.almadial-s05-*')))

    def test_writeable_parent_rejected(self):
        (self.root / 'var/www/html').chmod(0o777)
        with self.assertRaises(ValueError):
            adapter.install([self.file])

    def test_invalid_destinations(self):
        for target in ('/etc/passwd', '/var/www/html/../escape',
                       '/var/www/html//double', '/var/www/html/line\nbreak'):
            with self.subTest(target=target), self.assertRaises(ValueError):
                adapter.observation(dict(self.file, Target=target))


if __name__ == '__main__':
    unittest.main()
