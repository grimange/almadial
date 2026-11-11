#!/usr/bin/env python3
"""Real Chromium expiry rejection on an isolated rootless loopback TLS fixture.

A valid control and expired leaf share the same trusted temporary CA and SAN.
No accepted server certificate is touched; temporary keys/profiles are removed.
Requires the laboratory's existing Python cryptography, NSS and Playwright tools.
"""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import ssl
import subprocess
import tempfile
import threading

from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import ExtendedKeyUsageOID, NameOID


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'AlmaDial ALM-27 expiry control')

    def log_message(self, *_):
        pass


def experiment():
    with tempfile.TemporaryDirectory(prefix='alm27-expiry-') as temporary:
        root = Path(temporary)
        now = datetime.now(timezone.utc)
        ca_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        name = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, 'ALM-27 temporary expiry fixture')])
        ca = (x509.CertificateBuilder().subject_name(name).issuer_name(name)
              .public_key(ca_key.public_key()).serial_number(x509.random_serial_number())
              .not_valid_before(now - timedelta(days=30)).not_valid_after(now + timedelta(days=1))
              .add_extension(x509.BasicConstraints(ca=True, path_length=0), critical=True)
              .sign(ca_key, hashes.SHA256()))
        ca_path = root / 'ca.pem'
        ca_path.write_bytes(ca.public_bytes(serialization.Encoding.PEM))
        nss = root / 'home/.pki/nssdb'
        nss.mkdir(parents=True)
        for argv in [['certutil', '-N', '--empty-password', '-d', 'sql:' + str(nss)],
                     ['certutil', '-A', '-d', 'sql:' + str(nss), '-n', 'ALM27 expiry fixture',
                      '-t', 'C,,', '-i', str(ca_path)]]:
            subprocess.run(argv, check=True, capture_output=True, timeout=15)
        servers, rows = [], {}
        try:
            for label, start, end in [('valid', now - timedelta(hours=1), now + timedelta(hours=1)),
                                      ('expired', now - timedelta(days=2), now - timedelta(days=1))]:
                key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
                leaf = (x509.CertificateBuilder().subject_name(x509.Name([
                    x509.NameAttribute(NameOID.COMMON_NAME, 'localhost')])).issuer_name(name)
                    .public_key(key.public_key()).serial_number(x509.random_serial_number())
                    .not_valid_before(start).not_valid_after(end)
                    .add_extension(x509.SubjectAlternativeName([x509.DNSName('localhost')]), critical=False)
                    .add_extension(x509.BasicConstraints(ca=False, path_length=None), critical=True)
                    .add_extension(x509.ExtendedKeyUsage([ExtendedKeyUsageOID.SERVER_AUTH]), critical=False)
                    .sign(ca_key, hashes.SHA256()))
                cert_path, key_path = root / (label + '.pem'), root / (label + '.key')
                cert_path.write_bytes(leaf.public_bytes(serialization.Encoding.PEM))
                key_path.write_bytes(key.private_bytes(serialization.Encoding.PEM,
                                                       serialization.PrivateFormat.PKCS8,
                                                       serialization.NoEncryption()))
                os.chmod(key_path, 0o600)
                server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
                context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
                context.load_cert_chain(cert_path, key_path)
                server.socket = context.wrap_socket(server.socket, server_side=True)
                threading.Thread(target=server.serve_forever, daemon=True).start()
                servers.append(server)
                check = subprocess.run(['openssl', 'verify', '-CAfile', str(ca_path),
                                        '-verify_hostname', 'localhost', str(cert_path)],
                                       capture_output=True, text=True, timeout=15)
                rows[label] = {'port': server.server_port, 'openssl_exit': check.returncode,
                               'openssl_output': (check.stdout + check.stderr).replace(str(root), '[fixture]'),
                               'certificate_sha256': hashlib.sha256(cert_path.read_bytes()).hexdigest()}
            code = r'''
const {chromium}=require('/home/ra/.npm/_npx/51691537fc71f2b0/node_modules/playwright');
(async()=>{
const input=JSON.parse(process.argv[1]);
const c=await chromium.launchPersistentContext(input.root+'/profile',{
 executablePath:'/opt/google/chrome/chrome',headless:true,ignoreHTTPSErrors:false,
 env:{...process.env,HOME:input.root+'/home'}});
const p=await c.newPage(),out={browser:c.browser().version(),trust_bypass:false};
try {for(const label of ['valid','expired']) {
 try {const r=await p.goto('https://localhost:'+input[label],{timeout:15000});out[label]={status:r.status(),body:await p.locator('body').innerText()};}
 catch(e){out[label]={error:String(e).split('\n')[0]};}
}} finally{await c.close();} console.log(JSON.stringify(out));
})().catch(e=>{console.error(e.name);process.exitCode=1;});
'''
            operation = subprocess.run(['node', '-e', code, json.dumps({
                'root': str(root), 'valid': rows['valid']['port'], 'expired': rows['expired']['port']})],
                capture_output=True, text=True, timeout=60)
            assert operation.returncode == 0, operation.stderr[-400:]
            browser = json.loads(operation.stdout)
            passed = (rows['valid']['openssl_exit'] == 0 and rows['expired']['openssl_exit'] != 0
                      and 'certificate has expired' in rows['expired']['openssl_output']
                      and browser['valid'].get('status') == 200
                      and 'net::ERR_CERT_DATE_INVALID' in browser['expired'].get('error', ''))
            return {'state': 'PASS' if passed else 'FAIL', 'native': rows, 'browser': browser,
                    'boundary': 'Isolated loopback fixture; accepted S08 certificates unchanged'}
        finally:
            for server in servers:
                server.shutdown()
                server.server_close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--evidence', type=Path, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    assert not args.evidence.exists(), 'evidence output already exists'
    result = experiment()
    with args.evidence.open('x') as output:
        json.dump(result, output, indent=2, sort_keys=True)
    print(json.dumps(result, indent=2))
    raise SystemExit(0 if result['state'] == 'PASS' else 1)
