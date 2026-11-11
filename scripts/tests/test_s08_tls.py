"""Closed S08 activation/reversal fixtures; no root, network or service changes."""
import base64
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock

ROOT = Path(__file__).resolve().parents[2]

class TLSActivation(unittest.TestCase):
    def fixture(self, directory, initial=False):
        ns = {'__name__': 's08_fixture'}
        exec(compile((ROOT/'internal/provision/tls_native.py').read_text(), 'tls_native.py', 'exec'), ns)
        root = Path(directory)
        ns.update(R=root, AP=root/'http.conf', VH=root/'vhost.conf', LD=root/'loader.conf')
        old, new = 'a'*64, 'b'*64
        (root/new).mkdir()
        for name in ('fullchain.pem', 'key.pem', 'trust.pem'):
            (root/new/name).write_bytes(b'fixture')
        if not initial:
            (root/old).mkdir()
            (root/'current').symlink_to(old)
        for key in ('AP', 'VH', 'LD'):
            if key == 'AP' or not initial:
                ns[key].write_bytes(b'original '+key.encode())
        ns['safe'] = lambda p: None
        ns['current'] = lambda: str((root/'current').readlink()) if (root/'current').is_symlink() else None
        ns['contracts'] = lambda p: 0
        ns['read'] = lambda p, *a: p.read_bytes()
        ns['write'] = lambda p, data, *a: p.write_bytes(data)
        ns['inspect'] = lambda p: {'Current': 'expected'}
        ns['ssl'] = Mock()
        payload = dict(ExpectedCurrent='expected', Generation=new, Chain=base64.b64encode(b'fixture').decode(), Key=base64.b64encode(b'fixture').decode(), Trust=base64.b64encode(b'fixture').decode(), HTTP='new AP', Vhost='new VH', Loader='new LD')
        return ns, payload, old, new

    def test_failed_reload_restores_last_working_pointer_and_config(self):
        for initial in (False, True):
            with self.subTest(initial=initial), tempfile.TemporaryDirectory() as d:
                ns, p, old, new = self.fixture(d, initial)
                before = {k: ns[k].read_bytes() if ns[k].exists() else None for k in ('AP','VH','LD')}
                ns['health'] = lambda p: False
                reloads = []
                def reload():
                    reloads.append(ns['current']())
                    if len(reloads) == 1:
                        raise ValueError('controlled failed activation')
                ns['reload_services'] = reload
                with self.assertRaisesRegex(ValueError, 'previous configuration restored'):
                    ns['activate'](p)
                self.assertEqual(reloads, [new, None if initial else old])
                self.assertEqual(ns['current'](), None if initial else old)
                for k, value in before.items():
                    self.assertEqual(ns[k].read_bytes() if ns[k].exists() else None, value)
                self.assertTrue((Path(d)/new/'key.pem').exists())

    def test_success_keeps_previous_generation_and_repeat_does_not_reload(self):
        with tempfile.TemporaryDirectory() as d:
            ns,p,old,new = self.fixture(d)
            ns['health'] = Mock(side_effect=[False,True])
            ns['reload_services'] = Mock()
            ns['activate'](p)
            self.assertEqual(ns['current'](),new)
            self.assertEqual((Path(d)/'previous').readlink(),Path(old))
            ns['reload_services'].assert_called_once()
            ns['reload_services'].reset_mock()
            ns['health'] = lambda p: True
            ns['activate'](p)
            ns['reload_services'].assert_not_called()

    def test_stale_transaction_rejects_before_mutation(self):
        with tempfile.TemporaryDirectory() as d:
            ns,p,old,new = self.fixture(d)
            ns['inspect'] = lambda p: {'Current': 'changed'}
            ns['reload_services'] = Mock()
            with self.assertRaisesRegex(ValueError,'stale TLS transaction'):
                ns['activate'](p)
            self.assertEqual(ns['current'](),old)
            ns['reload_services'].assert_not_called()
