"""Real loopback HTTP handler; synthetic DB accounts and WAVs only."""
import base64
from concurrent.futures import ThreadPoolExecutor
import http.client
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import threading
import time
import types
import unittest
from unittest.mock import patch
import wave

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
try:
    import pymysql
except ImportError:
    sys.modules['pymysql'] = types.SimpleNamespace()
import bcrypt
import portal
import recording_media as media

NAME = '20261107-024337_SYNTHETIC-all.wav'
PATH = '/vicidial/recordings/' + NAME


class DB:
    def __init__(self, fixture):
        self.f = fixture

    def cursor(self):
        return self

    def __enter__(self):
        return self

    def __exit__(self, *args):
        pass

    def execute(self, sql, params=None):
        assert sql.startswith('SELECT '), 'Media must never write VICIdial state'
        self.f.queries.append((sql, params))
        if 'FROM system_settings' in sql:
            self.result = self.f.settings
        elif 'FROM vicidial_users ' in sql:
            self.result = self.f.user if params[0] == 'synthetic' else None
        elif 'FROM vicidial_user_groups ' in sql:
            self.result = self.f.group
        elif 'FROM recording_log ' in sql:
            self.result = self.f.records if params[0] == NAME[:-8] else []
        else:
            raise AssertionError('Unexpected SELECT')

    def fetchone(self):
        return self.result

    def fetchall(self):
        return self.result

    def close(self):
        pass


class Playback(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = portal.Server(('127.0.0.1', 0), portal.Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = self.root = Path(self.tmp.name)
        (root / 'web').mkdir()
        (root / 'web/8.8.8.8').write_text(str(int(time.time()) + 300))
        self.file = root / NAME
        with wave.open(str(self.file), 'wb') as wav:
            wav.setparams((1, 2, 8000, 0, 'NONE', 'not compressed'))
            wav.writeframes(b'\x01\x02' * 24000)
        self.file.chmod(0o640)
        self.receipt = dict(recording_id=2, name=NAME, fingerprint=media.fingerprint(self.file.stat()))
        (root / '2.json').write_text(json.dumps(self.receipt))
        self.user = dict(user='synthetic', active='Y', user_level=9, api_only_user='0',
                         force_change_password='N', **{'pass': 'synthetic-pass'},
                         access_recordings=1, view_reports=1, user_group='SYNTHETIC',
                         failed_login_count=0, login_epoch=0, db_now=int(time.time()))
        self.settings = dict(pass_hash_enabled='0', two_factor_auth_hours=0, allow_ip_lists=0,
                             system_ip_blacklist='', log_recording_access=0)
        self.group = dict(allowed_campaigns=' -ALL-CAMPAIGNS- - -', allowed_reports='ALL REPORTS',
                          admin_viewable_groups=' ---ALL--- ')
        self.records = [dict(recording_id=2, filename=NAME[:-8], end_epoch=1,
                             length_in_sec=3, location=media.ORIGIN + '/RECORDINGS/' + NAME)]
        self.queries = []
        self.patches = [patch.object(media, 'ROOT', root), patch.object(media, 'RECEIPTS', root),
                        patch.object(media, 'ROOT_UID', os.getuid()), patch.object(media, 'BASE', root),
                        patch.object(media.auth, '_connect', side_effect=lambda: DB(self)),
                        patch.dict(os.environ, ALMA_RECORDING_PLAYBACK='1')]
        for p in self.patches:
            p.start()
        media.FAILURES.clear()

    def tearDown(self):
        for p in reversed(self.patches):
            p.stop()
        self.tmp.cleanup()

    def request(self, path=PATH, method='GET', password='synthetic-pass', extra=None, credentials=True):
        headers = {'Host': portal.PUBLIC_IP, 'X-Alma-Source': '8.8.8.8'}
        if credentials:
            headers['Authorization'] = 'Basic ' + base64.b64encode(('synthetic:' + password).encode()).decode()
        headers.update(extra or {})
        conn = http.client.HTTPConnection('127.0.0.1', self.server.server_port, timeout=5)
        conn.request(method, path, headers=headers)
        response = conn.getresponse()
        result = response.status, dict(response.getheaders()), response.read()
        conn.close()
        return result

    def test_get_head_range_and_fixed_307(self):
        size = self.file.stat().st_size
        code, headers, body = self.request()
        self.assertEqual((code, len(body), headers['Content-Type']), (200, size, 'audio/wav'))
        self.assertEqual(headers['Cache-Control'], 'no-store, private')
        self.assertEqual(self.request(method='HEAD')[0::2], (200, b''))
        for value, expected in [('bytes=0-43', (0, 43)), ('bytes=44-', (44, size-1)),
                                ('bytes=-10', (size-10, size-1))]:
            with self.subTest(value=value):
                code, headers, body = self.request(extra={'Range': value})
                self.assertEqual(code, 206)
                a, b = expected
                self.assertEqual(body, self.file.read_bytes()[a:b+1])
                self.assertEqual(headers['Content-Range'], f'bytes {a}-{b}/{size}')
        for method in ('GET', 'HEAD'):
            code, headers, body = self.request('/RECORDINGS/' + NAME, method=method)
            self.assertEqual((code, headers['Location'], body), (307, PATH, b''))

    def test_bad_password_missing_identity_and_no_disclosure(self):
        for kwargs in ({'password': 'incorrect-private'}, {'credentials': False}):
            code, headers, body = self.request(**kwargs)
            self.assertEqual((code, body), (401, b''))
            self.assertEqual(headers['WWW-Authenticate'], 'Basic realm="CONTACT-CENTER-ADMIN"')
        self.assertEqual(self.request(extra={'Authorization': 'Basic ???'})[0], 401)
        self.assertEqual(self.request(extra={'Cookie': '__Secure-AlmaPortal=fake'}, credentials=False)[0], 401)

    def test_current_native_permission_and_account_revocation(self):
        self.assertEqual(self.request()[0], 200)
        for key, value in [('access_recordings', 0), ('active', 'N'), ('user_level', 1),
                           ('api_only_user', '1'), ('force_change_password', 'Y'), ('view_reports', 0),
                           ('failed_login_count', 10)]:
            old = self.user[key]
            self.user[key] = value
            if key == 'failed_login_count':
                self.user['login_epoch'] = int(time.time())
            with self.subTest(key=key):
                self.assertEqual(self.request()[0], 403)
            self.user[key] = old
        self.user['failed_login_count'] = 10
        self.user['login_epoch'] = 1
        media.FAILURES.clear()
        self.assertEqual(self.request()[0], 200)

    def test_unsupported_native_policy_fails_closed(self):
        for key, value in [('two_factor_auth_hours', 1), ('allow_ip_lists', 1),
                           ('system_ip_blacklist', 'configured-list'), ('log_recording_access', 1)]:
            old = self.settings[key]
            self.settings[key] = value
            self.assertEqual(self.request()[0], 503)
            self.settings[key] = old

    def test_bcrypt_native_mode(self):
        salt = bcrypt.gensalt(rounds=4, prefix=b'2a')
        self.settings.update(pass_hash_enabled=1, pass_cost=4, pass_key=salt.decode()[7:29])
        self.user['pass_hash'] = bcrypt.hashpw(b'synthetic-pass', salt).decode()[29:]
        self.assertEqual(self.request()[0], 200)
        self.assertEqual(self.request(password='wrong')[0], 401)

    def test_restricted_group_denied(self):
        for key in self.group:
            old = self.group[key]
            self.group[key] = 'restricted'
            self.assertEqual(self.request()[0], 403)
            self.group[key] = old

    def test_closed_unique_native_association_required(self):
        for records in ([], self.records * 2, [{**self.records[0], 'end_epoch': 0}],
                        [{**self.records[0], 'location': 'http://example.invalid/file.wav'}]):
            old = self.records
            self.records = records
            self.assertEqual(self.request()[0], 404)
            self.records = old

    def test_receipt_and_unchanged_finalized_inode_required(self):
        (self.root / '2.json').unlink()
        self.assertEqual(self.request()[0], 404)
        (self.root / '2.json').write_text(json.dumps(self.receipt))
        with self.file.open('ab') as stream:
            stream.write(b'changed')
        self.assertEqual(self.request()[0], 404)

    def test_symlink_hardlink_and_arbitrary_paths_denied(self):
        target = self.root / 'unrelated.wav'
        self.file.rename(target)
        self.file.symlink_to(target)
        self.assertEqual(self.request()[0], 404)
        self.file.unlink()
        os.link(target, self.file)
        self.assertEqual(self.request()[0], 404)
        for name in ('../secrets', '%2e%2e/secrets', 'raw-in.wav', 'anything.txt'):
            self.assertNotEqual(self.request('/vicidial/recordings/' + name)[0], 200)

    def test_range_errors_methods_and_cross_origin(self):
        for value in ('bytes=9-1', 'bytes=999999-', 'bytes=-0', 'bytes=0-1,4-5', 'bad', 'bytes=-'):
            code, headers, body = self.request(extra={'Range': value})
            self.assertEqual((code, body), (416, b''))
            self.assertEqual(headers['Content-Range'], 'bytes */' + str(self.file.stat().st_size))
        self.assertEqual(self.request(method='POST')[0], 405)
        self.assertEqual(self.request(extra={'Sec-Fetch-Site': 'cross-site'})[0], 403)

    def test_network_and_feature_fail_closed(self):
        (self.root / 'web/8.8.8.8').write_text('1')
        self.assertEqual(self.request()[0], 403)
        self.assertEqual(self.request('/RECORDINGS/' + NAME)[0], 403)
        os.environ['ALMA_RECORDING_PLAYBACK'] = '0'
        self.assertEqual(self.request()[0], 503)

    def test_independent_concurrent_and_failed_auth_rate_limit(self):
        with ThreadPoolExecutor(max_workers=2) as executor:
            results = list(executor.map(lambda _: self.request(extra={'Range': 'bytes=0-99'}), range(6)))
        self.assertEqual([r[0] for r in results], [206] * 6)
        self.assertEqual([self.request(password='incorrect')[0] for _ in range(8)], [401]*7 + [429])

    def test_large_file_reads_are_bounded(self):
        with self.file.open('r+b') as stream:
            stream.truncate(8 * 1024 * 1024)
        self.receipt['fingerprint'] = media.fingerprint(self.file.stat())
        (self.root / '2.json').write_text(json.dumps(self.receipt))
        original = media.open_recording
        reads = []

        class Tracked:
            def __init__(self, stream):
                self.stream = stream
            def __enter__(self):
                return self
            def __exit__(self, *args):
                self.stream.close()
            def seek(self, pos):
                self.stream.seek(pos)
            def read(self, size):
                reads.append(size)
                return self.stream.read(size)

        def tracking(*args):
            stream, size = original(*args)
            return Tracked(stream), size

        with patch.object(media, 'open_recording', side_effect=tracking):
            code, headers, body = self.request()
        self.assertEqual((code, len(body)), (200, 8 * 1024 * 1024))
        self.assertEqual(max(reads), 65536)


if __name__ == '__main__':
    unittest.main()
