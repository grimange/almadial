#!/usr/bin/env python3
"""Guarded rollback/reapply for the three scoped host configuration edits only.
Run on dialer01 as root, with no active channels. Native fixture rollback is separate.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import socket
import subprocess

ROOT = Path('/root/prd-webphone-00-backup')
FILES = ('/etc/asterisk/http.conf', '/etc/httpd/conf.d/02-alma-vicidial-ssl.conf',
         '/opt/alma-portal-443/apply_requests.py')

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def run(*args):
    return subprocess.run(args, check=True, text=True, capture_output=True).stdout.strip()

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('capture-deployed','check','rollback','restore-deployed'))
    action=parser.parse_args().action
    if socket.gethostname() != 'dialer01':
        raise SystemExit('Wrong host')
    before=json.loads((ROOT/'manifest.json').read_text())
    for name in FILES:
        if sha(ROOT/name.lstrip('/')) != before[name]:
            raise SystemExit('Original backup mismatch: '+name)
    after_path=ROOT/'deployed-manifest.json'
    if action=='capture-deployed':
        if after_path.exists(): raise SystemExit('Deployed snapshot already exists')
        for name in FILES:
            target=ROOT/'deployed'/name.lstrip('/')
            target.parent.mkdir(parents=True,exist_ok=True)
            shutil.copy2(name,target)
        after_path.write_text(json.dumps({n:sha(n) for n in FILES},indent=2))
    after=json.loads(after_path.read_text())
    for name in FILES:
        if sha(ROOT/'deployed'/name.lstrip('/')) != after[name]:
            raise SystemExit('Deployed backup mismatch: '+name)
    if action in ('rollback','restore-deployed'):
        if not run('asterisk','-rx','core show channels count').startswith('0 active channels'):
            raise SystemExit('Active channels: maintenance boundary')
        expected=after if action=='rollback' else before
        source=ROOT if action=='rollback' else ROOT/'deployed'
        if any(sha(n)!=expected[n] for n in FILES):
            raise SystemExit('Configuration drift: review before restoring')
        for name in FILES: shutil.copy2(source/name.lstrip('/'),name)
        run('httpd','-t')
        run('asterisk','-rx','module reload http')
        run('systemctl','reload','httpd')
    print(json.dumps({'action':action,'hashes':{n:sha(n) for n in FILES},
                      'http':run('asterisk','-rx','http show status'),
                      'apache':run('systemctl','is-active','httpd')},indent=2))

if __name__=='__main__':
    main()
