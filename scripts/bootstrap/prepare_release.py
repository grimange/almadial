#!/usr/bin/env python3
"""Prepare an installer-only private release directory; never publishes it."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--endpoint', default='', help='optional reviewed default binary base URL; omit for operator-configured delivery')
    p.add_argument('--version', required=True)
    p.add_argument('--expected-sha256', help='approved binary digest; checked before executing the supplied binary')
    p.add_argument('--output', type=Path, required=True)
    a = p.parse_args()
    u = urlsplit(a.endpoint)
    if a.endpoint and (u.scheme != 'https' or not u.hostname or u.username or u.password or u.query or u.fragment or not re.fullmatch(r'https://[A-Za-z0-9][A-Za-z0-9.:/_-]*', a.endpoint)):
        p.error('a reviewed HTTPS endpoint without credentials/query is required')
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9.+_-]{0,80}', a.version):
        p.error('invalid version')
    binary = a.binary.read_bytes()
    if binary[:4] != b'\x7fELF' or binary[18:20] != b'\x3e\x00':
        p.error('expected Linux amd64 ELF')
    sha = hashlib.sha256(binary).hexdigest()
    if a.expected_sha256 and (not re.fullmatch(r'[0-9a-f]{64}', a.expected_sha256) or sha != a.expected_sha256):
        p.error('binary does not match the approved SHA-256')
    identity = subprocess.check_output([str(a.binary.resolve()), '--version'], text=True)
    if identity.splitlines()[0:1] != ['almadial '+a.version]:
        p.error('binary version does not match the requested release')
    source = re.search(r'^commit: ([0-9a-f]{40})$', identity, re.MULTILINE)
    if not source:
        p.error('binary must identify an exact committed source revision')
    a.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    name = 'almadial-linux-amd64'
    (a.output / name).write_bytes(binary)
    script = (ROOT / 'scripts/bootstrap/install.sh.in').read_text()
    for key, val in {'VERSION':a.version, 'BINARY_URL':a.endpoint.rstrip('/')+'/'+name if a.endpoint else '', 'BINARY_SHA256':sha}.items():
        script = script.replace('@'+key+'@', val)
    (a.output / 'install.sh').write_text(script)
    manifest = {'schema':1,'version':a.version,'binary':name,'sha256':sha,
                'target':'linux/amd64','endpoint':a.endpoint,
                'build_identity':identity,
                'source_commit':source.group(1),
                'packaging_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
                'packaging_source_payload_sha256':hashlib.sha256(b''.join(str(f.relative_to(ROOT)).encode()+b'\0'+hashlib.sha256(f.read_bytes()).digest() for f in sorted({*ROOT.glob('cmd/**/*'),*ROOT.glob('internal/**/*'),ROOT/'go.mod',ROOT/'go.sum'}) if f.is_file() and '__pycache__' not in f.parts)).hexdigest(),
                'packaging_diff_sha256':hashlib.sha256(subprocess.check_output(['git','diff','HEAD'],cwd=ROOT)).hexdigest(),
                'distribution':'AlmaDial CLI only; no upstream application archives'}
    (a.output / 'release.json').write_text(json.dumps(manifest,indent=2)+'\n')
    shutil.copyfile(ROOT/'LICENSE', a.output/'LICENSE')
    # Preserve notices for the Go dependency actually linked into this CLI.
    shutil.copyfile(ROOT/'docs/THIRD_PARTY_NOTICES.md',a.output/'THIRD_PARTY_NOTICES.md')
    licenses = a.output/'licenses'
    licenses.mkdir()
    shutil.copytree(ROOT/'third_party/go-yaml', licenses/'go-yaml')
    for directory in sorted((ROOT/'docs/compliance/m4-04/licenses').glob('Go*')):
        shutil.copytree(directory, licenses/directory.name)
    shutil.copyfile(ROOT/'docs/compliance/m4-04/licenses/Apache-2.0.txt',licenses/'Apache-2.0.txt')
    (a.output/'CLI-NOTICES.txt').write_text(
        'AlmaDial-original code: root LICENSE (MIT).\n'
        'Go standard library and toolchain vendor notices: licenses/Go* (BSD and patent grants).\n'
        'go.yaml.in/yaml/v3 v3.0.4: licenses/go-yaml/LICENSE and NOTICE; '
        'licenses/Apache-2.0.txt (MIT and Apache-2.0 portions).\n'
        'THIRD_PARTY_NOTICES.md preserves the broader repository review and its unresolved '
        'application-distribution boundaries; those applications are not in this payload.\n')
    (a.output/'SHA256SUMS').write_text(''.join(hashlib.sha256(f.read_bytes()).hexdigest()+'  '+str(f.relative_to(a.output))+'\n' for f in sorted(a.output.rglob('*')) if f.is_file()))
    print(json.dumps(manifest,indent=2))

if __name__ == '__main__':
    main()
