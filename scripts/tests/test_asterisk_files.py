"""S04 filesystem fixture tests. All fixtures are disposable, rootless paths."""
import importlib.util
import json
import os
from pathlib import Path
import stat
import tempfile
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2] / 'internal/provision/asterisk_files.py'

class AsteriskFiles(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('asterisk_fixture', SOURCE)
        self.m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.m)
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.m.W = self.root
        self.m.T = self.root / 'source'
        self.m.p = {'binding':'fixture binding','modules':['res_agi']}
        # The adapter's production ownership boundary is always uid 0. Only the
        # fixture substitutes the current account and an isolated parent tree.
        def fixture_safe(path):
            path = Path(path)
            self.assertTrue(path.is_relative_to(self.root))
            for x in [self.root, *reversed(path.relative_to(self.root).parents), path]:
                x = x if x.is_absolute() else self.root/x
                if x.exists() or x.is_symlink():
                    s = x.lstat()
                    if stat.S_ISLNK(s.st_mode) or s.st_uid != os.getuid() or s.st_mode & 0o022:
                        raise ValueError('unsafe fixture path')
            return path
        self.m.safe = fixture_safe
        self.addCleanup(patch.stopall)
        patch.object(self.m.os, 'fchown', lambda *args: None).start()

    def test_create_preserves_existing_and_syncs(self):
        p = self.root/'owned'
        self.m.create(p,b'accepted')
        self.assertEqual(p.read_bytes(),b'accepted')
        self.assertEqual(stat.S_IMODE(p.stat().st_mode),0o600)
        with self.assertRaises(FileExistsError): self.m.create(p,b'clobber')
        self.assertEqual(p.read_bytes(),b'accepted')

    def test_symlink_and_writable_parent_rejected(self):
        target=self.root/'operator';target.write_bytes(b'preserve')
        link=self.root/'link';link.symlink_to(target)
        with self.assertRaises(ValueError): self.m.create(link,b'bad')
        parent=self.root/'writable';parent.mkdir();parent.chmod(0o777)
        with self.assertRaises(ValueError): self.m.create(parent/'bad',b'bad')
        self.assertEqual(target.read_bytes(),b'preserve')

    def test_wrong_mode_and_group_rejected(self):
        p=self.root/'file';p.write_bytes(b'data');p.chmod(0o644)
        with self.assertRaises(ValueError): self.m.read(p,0o600,os.getgid())
        with self.assertRaises(ValueError): self.m.read(p,0o644,os.getgid()+1)

    def test_config_absent_valid_operator_and_partial(self):
        p=self.root/'base';q=self.root/'include'
        self.m.config_files=lambda:({str(p):b'include',str(q):b'owned'},os.getgid())
        self.assertEqual(self.m.configuration(),'absent')
        self.m.create(p,b'include',0o640)
        self.assertEqual(self.m.configuration(),'invalid')
        self.m.create(q,b'owned',0o640)
        self.assertEqual(len(self.m.configuration()),64)
        q.write_bytes(b'operator modification')
        self.assertEqual(self.m.configuration(),'invalid')
        self.assertEqual(q.read_bytes(),b'operator modification')

    def test_partial_and_stale_build_receipt_rejected(self):
        with self.assertRaises(FileNotFoundError): self.m.build()
        p=self.root/'build.json';p.write_text(json.dumps({'binding':'different source'}));p.chmod(0o600)
        # Production receipt checks require group root, fixture uses current gid.
        original=self.m.read
        self.m.read=lambda p,perm=None,gid=0:original(p,perm,os.getgid())
        with self.assertRaisesRegex(ValueError,'stale build'): self.m.build()

    def test_runtime_libraries_and_xml_required_for_complete_inventory(self):
        original=self.m.read
        self.m.read=lambda p,perm=None,gid=0:original(p,perm,os.getgid())
        self.m.build=lambda:'verified fixture'
        paths=[]
        for name in ('asterisk','libasteriskssl.so.1','libasteriskpj.so.2','documentation/core-en_US.xml','documentation/appdocsxml.xslt','documentation/appdocsxml.dtd'):
            src=self.root/'source'/name;src.parent.mkdir(parents=True,exist_ok=True);src.write_bytes(name.encode())
            dst=self.root/'live'/name;dst.parent.mkdir(parents=True,exist_ok=True)
            paths.append((src,dst))
            for parent in (src.parent,dst.parent,self.root/'source',self.root/'live'):parent.chmod(0o755)
        self.m.install_paths=lambda:paths
        self.m.resource_paths=lambda:paths[3:]
        installed={}
        def seal():
            p=self.root/'install.json';p.write_text(json.dumps({'binding':'fixture binding','files':installed}));p.chmod(0o600)
        def put(pair):
            src,dst=pair;dst.write_bytes(src.read_bytes());dst.chmod(self.m.artifact_mode(dst));installed[str(dst)]=self.m.sha(dst.read_bytes())
        put(paths[0]);seal()
        self.assertEqual(self.m.install(),'libraries-missing')
        put(paths[1]);put(paths[2]);seal()
        self.assertEqual(self.m.install(),'resources-missing')
        put(paths[3]);seal()
        with self.assertRaises(FileNotFoundError):self.m.install()
        put(paths[4]);put(paths[5]);seal()
        self.assertEqual(len(self.m.install()),64)
        paths[3][1].write_bytes(b'corrupt XML')
        with self.assertRaisesRegex(ValueError,'artifact drift'):self.m.install()

if __name__ == '__main__': unittest.main()
