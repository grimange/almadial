"""Rootless bootstrap fixtures with real curl/TLS; native qualification is separate."""
from contextlib import contextmanager, ExitStack
import hashlib
import http.server
import importlib.util
import json
import os
from pathlib import Path
import ssl
import subprocess
import tempfile
import tarfile
import threading
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
VERSION = 'alm46-internet-rc1'
DIGEST = '618f4b0441cde7f17d1a35382b581da5669e0efa67b04f28175bc0cb38c8e6b5'


@contextmanager
def https_origin(certificate, key, payload, status=200, redirect=None):
    requests = []

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            requests.append(self.path)
            if redirect and (self.path == '/almadial-linux-amd64' or redirect == 'loop'):
                self.send_response(302)
                location = self.path if redirect == 'loop' else redirect
                self.send_header('Location', location)
                self.send_header('Content-Length', '0')
                self.end_headers()
                return
            self.send_response(status)
            self.send_header('Content-Length', str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def handle(self):
            try:
                super().handle()
            except (ConnectionResetError, BrokenPipeError):
                pass  # Expected when curl rejects this fixture's untrusted CA.

        def log_message(self, *_args):
            pass

    server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(certificate, key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f'https://127.0.0.1:{server.server_port}/almadial-linux-amd64', requests
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


class BootstrapTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tls = tempfile.TemporaryDirectory()
        cls.cert = Path(cls.tls.name)/'cert.pem'
        cls.key = Path(cls.tls.name)/'key.pem'
        subprocess.run([
            'openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
            '-keyout', str(cls.key), '-out', str(cls.cert), '-days', '1',
            '-subj', '/CN=bootstrap-fixture', '-addext',
            'subjectAltName=IP:127.0.0.1',
        ], check=True, capture_output=True)

        cls.other_cert = Path(cls.tls.name)/'other.pem'
        cls.other_key = Path(cls.tls.name)/'other-key.pem'
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
                        '-keyout', str(cls.other_key), '-out', str(cls.other_cert), '-days', '1',
                        '-subj', '/CN=untrusted-CDN', '-addext', 'subjectAltName=IP:127.0.0.1'],
                       check=True, capture_output=True)

    @classmethod
    def tearDownClass(cls):
        cls.tls.cleanup()

    def run_case(self, *, corrupt=False, trusted=True, configured=True,
                 os_id='almalinux', version='10.2', arch='x86_64', status=200,
                 default=False, url_override=None, digest_override=False, launch_failure=False,
                 redirect=None, svn_present=True, package_failure=False):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d)
            dest = p/'target'
            dest.mkdir()
            existing = dest/'almadial'
            existing.write_bytes(b'previous installation\n')
            tools = p/'tools'
            tools.mkdir()
            marker = p/'executed'
            svn = tools/'svn'
            if svn_present:
                svn.write_text('#!/bin/bash\nexit 0\n')
                svn.chmod(0o755)
            binary = b'#!/bin/bash\ntouch "$BOOTSTRAP_TEST_MARKER"\nprintf "fixture version\\n"\n'
            if launch_failure:
                binary += b'exit 42\n'
            osfile = p/'os-release'
            osfile.write_text(f'ID={os_id}\nVERSION_ID={version}\n')
            for name, body in {
                'uname': f'printf "%s\\n" {arch}',
                'dnf': ('exit 1' if package_failure else f'touch {svn}; chmod 755 {svn}'),
                'install': 'cp "${@: -2:1}" "${@: -1}"; chmod 755 "${@: -1}"',
            }.items():
                f = tools/name
                f.write_text('#!/bin/bash\nset -e\n'+body+'\n')
                f.chmod(0o755)
            # Only emulate root, OS identity and installation destination.
            # curl, certificate checks, hashes, execution and replacement are real.
            script = (ROOT/'scripts/bootstrap/install.sh.in').read_text()
            script = script.replace('[[ ${EUID} == 0 ]]', '[[ 0 == 0 ]]')
            script = script.replace('/etc/os-release', str(osfile)).replace('/usr/local/sbin', str(dest))
            script = script.replace('/usr/bin/svn', str(svn))
            script = script.replace('@VERSION@', 'test')
            script = script.replace('@BINARY_SHA256@', hashlib.sha256(binary).hexdigest())
            env = {k: v for k, v in os.environ.items() if not k.startswith('ALMADIAL_')
                   and k not in ('CURL_CA_BUNDLE', 'SSL_CERT_FILE', 'SSL_CERT_DIR')}
            env.update(PATH=str(tools)+':'+os.environ['PATH'], NO_PROXY='127.0.0.1',
                       BOOTSTRAP_TEST_MARKER=str(marker))
            if trusted:
                env['ALMADIAL_CA_FILE'] = str(self.cert)
            if digest_override:
                env['ALMADIAL_BINARY_SHA256'] = hashlib.sha256(b'wrong revision').hexdigest()
            with ExitStack() as stack:
                payload = b'wrong revision' if corrupt else binary
                extra_requests = []
                if redirect in ('alternate', 'untrusted-CDN'):
                    cert, key = (self.other_cert, self.other_key) if redirect == 'untrusted-CDN' else (self.cert, self.key)
                    alternate, extra_requests = stack.enter_context(https_origin(cert, key, payload))
                    redirect = alternate
                url, requests = stack.enter_context(https_origin(self.cert, self.key, payload, status, redirect))
                if configured:
                    env['ALMADIAL_BINARY_URL'] = url if url_override is None else url_override
                script = script.replace('@BINARY_URL@', url if default else '')
                run = subprocess.run(['bash', '-c', script], env=env, capture_output=True, text=True)
                self.assertFalse(any(dest.glob('.almadial-bootstrap.*')))
                return run, marker.exists(), existing.read_bytes(), len(requests)+len(extra_requests), binary

    def test_configured_and_alternate_origins(self):
        # Each run serves the same contract at a newly allocated HTTPS origin.
        for _ in range(2):
            run, executed, installed, requests, binary = self.run_case()
            self.assertEqual(run.returncode, 0, run.stderr)
            self.assertTrue(executed)
            self.assertEqual(installed, binary)
            self.assertEqual(requests, 1)

    def test_reviewed_default_remains_supported(self):
        run, executed, _, _, _ = self.run_case(configured=False, default=True)
        self.assertEqual(run.returncode, 0, run.stderr)
        self.assertTrue(executed)

    def test_https_release_asset_redirects_preserve_identity(self):
        for redirect in ('/asset', 'alternate'):
            with self.subTest(redirect=redirect):
                run, executed, installed, requests, binary = self.run_case(
                    configured=False, default=True, redirect=redirect)
                self.assertEqual(run.returncode, 0, run.stderr)
                self.assertTrue(executed)
                self.assertEqual(installed, binary)
                self.assertEqual(requests, 2)

    def test_redirect_failures_never_execute_or_replace(self):
        for kwargs in ({'redirect': 'http://127.0.0.1:1/asset'},
                       {'redirect': 'loop'},
                       {'redirect': 'alternate', 'corrupt': True},
                       {'redirect': 'untrusted-CDN'}):
            with self.subTest(kwargs=kwargs):
                run, executed, installed, requests, _ = self.run_case(**kwargs)
                self.assertNotEqual(run.returncode, 0)
                self.assertFalse(executed)
                self.assertEqual(installed, b'previous installation\n')
                self.assertLessEqual(requests, 4)

    def test_signed_svn_utility_preparation(self):
        run, executed, _, _, _ = self.run_case(svn_present=False)
        self.assertEqual(run.returncode, 0, run.stderr)
        self.assertTrue(executed)
        run, executed, installed, _, _ = self.run_case(svn_present=False, package_failure=True)
        self.assertNotEqual(run.returncode, 0)
        self.assertFalse(executed)
        self.assertEqual(installed, b'previous installation\n')

    def test_verified_launch_failure_preserves_existing_binary(self):
        run, executed, installed, _, _ = self.run_case(launch_failure=True)
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('verified CLI launch failed', run.stderr)
        self.assertTrue(executed)
        self.assertEqual(installed, b'previous installation\n')

    def test_checksum_failure_never_executes_or_replaces(self):
        run, executed, installed, _, _ = self.run_case(corrupt=True, digest_override=True)
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('checksum mismatch', run.stderr)
        self.assertFalse(executed)
        self.assertEqual(installed, b'previous installation\n')

    def test_untrusted_tls_never_executes_or_replaces(self):
        run, executed, installed, requests, _ = self.run_case(trusted=False)
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('certificate', run.stderr.lower())
        self.assertFalse(executed)
        self.assertEqual(installed, b'previous installation\n')
        self.assertEqual(requests, 0)

    def test_missing_endpoint_is_actionable(self):
        run, executed, installed, requests, _ = self.run_case(configured=False)
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('set ALMADIAL_BINARY_URL', run.stderr)
        self.assertFalse(executed)
        self.assertEqual(installed, b'previous installation\n')
        self.assertEqual(requests, 0)

    def test_unsafe_urls_are_rejected(self):
        for url in ('http://example.invalid/cli', 'https://user:pass@example.invalid/cli',
                    'https://example.invalid/cli?sha=anything', 'https://example.invalid/cli#fragment'):
            with self.subTest(url=url):
                run, executed, installed, requests, _ = self.run_case(url_override=url)
                self.assertNotEqual(run.returncode, 0)
                self.assertFalse(executed)
                self.assertEqual(installed, b'previous installation\n')
                self.assertEqual(requests, 0)

    def test_http_failure_and_redirect_never_install(self):
        for status in (404, 302):
            with self.subTest(status=status):
                run, executed, installed, requests, _ = self.run_case(status=status, corrupt=True)
                self.assertNotEqual(run.returncode, 0)
                self.assertFalse(executed)
                self.assertEqual(installed, b'previous installation\n')
                self.assertEqual(requests, 1)

    def test_unsupported_os_and_architecture(self):
        for kwargs in ({'os_id': 'ubuntu'}, {'version': '10.1'}, {'arch': 'aarch64'}):
            with self.subTest(kwargs=kwargs):
                run, executed, installed, requests, _ = self.run_case(**kwargs)
                self.assertNotEqual(run.returncode, 0)
                self.assertFalse(executed)
                self.assertEqual(installed, b'previous installation\n')
                self.assertEqual(requests, 0)

    def test_operator_script_matches_approved_release(self):
        template = (ROOT/'scripts/bootstrap/install.sh.in').read_text()
        for key, value in {'VERSION': VERSION, 'BINARY_URL': 'https://github.com/grimange/almadial/releases/download/'+VERSION+'/almadial-linux-amd64', 'BINARY_SHA256': DIGEST}.items():
            template = template.replace('@'+key+'@', value)
        self.assertEqual(template, (ROOT/'scripts/bootstrap/install.sh').read_text())
        self.assertNotIn('10.160.90.1', template)

    def test_release_generator_optional_origin_keeps_content_identity(self):
        spec = importlib.util.spec_from_file_location('prepare_release', ROOT/'scripts/bootstrap/prepare_release.py')
        generator = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(generator)
        with tempfile.TemporaryDirectory() as d:
            p = Path(d)
            binary = p/'fixture-cli'
            elf = bytearray(64)
            elf[:4], elf[18:20] = b'\x7fELF', b'\x3e\x00'
            binary.write_bytes(elf)
            expected = hashlib.sha256(elf).hexdigest()
            for name, endpoint in (('portable', ''), ('default', 'https://approved.example.invalid/release')):
                output = p/name
                argv = ['prepare_release.py', '--binary', str(binary), '--version', 'fixture', '--output', str(output)]
                if endpoint:
                    argv += ['--endpoint', endpoint]
                # No synthetic ELF is executed. Only identity collection is mocked.
                with mock.patch('sys.argv', argv), mock.patch.object(generator.subprocess, 'check_output',
                        side_effect=['almadial fixture\ncommit: '+('a'*40)+'\n', 'packaging-checkout\n', b'']), mock.patch('builtins.print'):
                    generator.main()
                release = json.loads((output/'release.json').read_text())
                self.assertEqual(release['sha256'], expected)
                self.assertEqual(release['version'], 'fixture')
                self.assertEqual(release['source_commit'], 'a'*40)
                self.assertEqual(release['packaging_commit'], 'packaging-checkout')
                self.assertEqual(release['endpoint'], endpoint)
                self.assertEqual(release['publication_status'], 'NOT_PUBLISHED')
                with tarfile.open(output/'almadial-cli-notices.tar.gz') as notices:
                    self.assertIn('licenses/go-yaml/LICENSE', notices.getnames())
                    self.assertNotIn('almadial-linux-amd64', notices.getnames())
                rendered = (output/'install.sh').read_text()
                self.assertIn("readonly BINARY_SHA256='"+expected+"'", rendered)
                self.assertIn("readonly DEFAULT_BINARY_URL='"+(endpoint+'/almadial-linux-amd64' if endpoint else '')+"'", rendered)

    def test_github_release_preparation_supplies_versioned_default(self):
        spec = importlib.util.spec_from_file_location('prepare_release', ROOT/'scripts/bootstrap/prepare_release.py')
        generator = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(generator)
        with tempfile.TemporaryDirectory() as d:
            p = Path(d)
            binary = p/'fixture-cli'
            elf = bytearray(64)
            elf[:4], elf[18:20] = b'\x7fELF', b'\x3e\x00'
            binary.write_bytes(elf)
            expected = hashlib.sha256(elf).hexdigest()
            argv = ['prepare_release.py', '--binary', str(binary), '--version', 'fixture-rc1',
                    '--github-repository', 'grimange/almadial', '--expected-sha256', expected,
                    '--output', str(p/'release')]
            with mock.patch('sys.argv', argv), mock.patch.object(generator.subprocess, 'check_output',
                    side_effect=['almadial fixture-rc1\ncommit: '+('a'*40)+'\n', 'packaging-checkout\n', b'']), mock.patch('builtins.print'):
                generator.main()
            release = json.loads((p/'release/release.json').read_text())
            self.assertEqual(release['publication_status'], 'NOT_PUBLISHED')
            self.assertEqual(release['channel'], 'github-release')
            self.assertEqual(release['sha256'], expected)
            self.assertIn("readonly DEFAULT_BINARY_URL='https://github.com/grimange/almadial/releases/download/fixture-rc1/almadial-linux-amd64'",
                          (p/'release/install.sh').read_text())
            # The publication payload is concrete, but no network tool is invoked.
            for extra in ([], ['--expected-sha256', expected, '--github-repository', '../wrong']):
                bad = argv[:argv.index('--expected-sha256')]+['--output', str(p/'bad')]+extra
                with mock.patch('sys.argv', bad), mock.patch('sys.stderr'), self.assertRaises(SystemExit):
                    generator.main()
                self.assertFalse((p/'bad').exists())


    def test_release_generator_rejects_unapproved_identity_before_publication(self):
        spec = importlib.util.spec_from_file_location('prepare_release', ROOT/'scripts/bootstrap/prepare_release.py')
        generator = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(generator)
        with tempfile.TemporaryDirectory() as d:
            p = Path(d)
            binary = p/'fixture-cli'
            elf = bytearray(64)
            elf[:4], elf[18:20] = b'\x7fELF', b'\x3e\x00'
            binary.write_bytes(elf)
            for name, identity, extra in (
                    ('checksum', '', ['--expected-sha256', '0'*64]),
                    ('version', 'almadial wrong\ncommit: '+('a'*40)+'\n', []),
                    ('dirty', 'almadial fixture\ncommit: dirty-build\n', [])):
                output = p/name
                argv = ['prepare_release.py', '--binary', str(binary), '--version', 'fixture', '--output', str(output)]+extra
                with mock.patch('sys.argv', argv), mock.patch.object(generator.subprocess, 'check_output', return_value=identity) as run, mock.patch('sys.stderr'), self.assertRaises(SystemExit):
                    generator.main()
                self.assertFalse(output.exists())
                if name == 'checksum':
                    run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
