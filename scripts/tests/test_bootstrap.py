"""Rootless bootstrap fixtures with real curl/TLS; native qualification is separate."""
from contextlib import contextmanager
import hashlib
import http.server
import importlib.util
import json
import os
from pathlib import Path
import ssl
import subprocess
import tempfile
import threading
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
VERSION = 'm4-06-acquisition-dev'
DIGEST = '4b484239a3939e9f1f12eb6fda22a0f74a2e3d035023d3b01ec8e9e43aaf78f9'


@contextmanager
def https_origin(certificate, key, payload, status=200):
    requests = []

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            requests.append(self.path)
            self.send_response(status)
            self.send_header('Content-Length', str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def handle(self):
            try:
                super().handle()
            except ConnectionResetError:
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

    @classmethod
    def tearDownClass(cls):
        cls.tls.cleanup()

    def run_case(self, *, corrupt=False, trusted=True, configured=True,
                 os_id='almalinux', version='10.2', arch='x86_64', status=200,
                 default=False, url_override=None, digest_override=False, launch_failure=False):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d)
            dest = p/'target'
            dest.mkdir()
            existing = dest/'almadial'
            existing.write_bytes(b'previous installation\n')
            tools = p/'tools'
            tools.mkdir()
            marker = p/'executed'
            binary = b'#!/bin/bash\ntouch "$BOOTSTRAP_TEST_MARKER"\nprintf "fixture version\\n"\n'
            if launch_failure:
                binary += b'exit 42\n'
            osfile = p/'os-release'
            osfile.write_text(f'ID={os_id}\nVERSION_ID={version}\n')
            for name, body in {
                'uname': f'printf "%s\\n" {arch}',
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
            with https_origin(self.cert, self.key, b'wrong revision' if corrupt else binary, status) as (url, requests):
                if configured:
                    env['ALMADIAL_BINARY_URL'] = url if url_override is None else url_override
                script = script.replace('@BINARY_URL@', url if default else '')
                run = subprocess.run(['bash', '-c', script], env=env, capture_output=True, text=True)
                self.assertFalse(any(dest.glob('.almadial-bootstrap.*')))
                return run, marker.exists(), existing.read_bytes(), len(requests), binary

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
        for key, value in {'VERSION': VERSION, 'BINARY_URL': '', 'BINARY_SHA256': DIGEST}.items():
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
                        side_effect=['fixture version\n', 'packaging-checkout\n', b'']), mock.patch('builtins.print'):
                    generator.main()
                release = json.loads((output/'release.json').read_text())
                self.assertEqual(release['sha256'], expected)
                self.assertEqual(release['version'], 'fixture')
                self.assertEqual(release['endpoint'], endpoint)
                rendered = (output/'install.sh').read_text()
                self.assertIn("readonly BINARY_SHA256='"+expected+"'", rendered)
                self.assertIn("readonly DEFAULT_BINARY_URL='"+(endpoint+'/almadial-linux-amd64' if endpoint else '')+"'", rendered)


if __name__ == '__main__':
    unittest.main()
