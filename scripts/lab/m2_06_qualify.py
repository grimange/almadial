#!/usr/bin/env python3
"""Exact-target ALM-21 qualification driver; no dialing or automatic reboot.

Run only with the task's explicit disposable-VM authority and recovery snapshot.
Inspection/plan/run outputs are private and create-only. The Go executor owns
all S06 application mutations under its exclusive lock and durable journal.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

VM = 'almadial-alm16-rpm'
MACHINE = '0fffa9a4683f35b696f0b6ee7330734b'
GUEST = '/tmp/alm21-qualification/'
COMMON = ['--manifest', '/tmp/alm19-qualification/stack.lock.yaml',
          '--source', '/tmp/alm20-qualification/vicidial-r4037.zip',
          '--perl-source', '/tmp/alm20-qualification/Net-Telnet-3.05.tar.gz',
          '--asterisk-source', '/tmp/alm19-qualification/asterisk-22.12.0.tar.gz',
          '--signature', '/tmp/alm19-qualification/asterisk-22.12.0.tar.gz.asc',
          '--keyring', '/tmp/alm19-qualification/keyring.gpg',
          '--scratch', '/tmp/alm19-qualification/scratch',
          '--pjproject', '/tmp/alm19-qualification/pjproject-2.17.tar.bz2']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['deploy', 'inspect', 'plan', 'lab-run', 'lab-resume'])
    parser.add_argument('label')
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--input')
    parser.add_argument('--plan')
    parser.add_argument('--state')
    parser.add_argument('--binary', type=Path)
    args = parser.parse_args()
    os.umask(0o077)
    e = args.evidence.resolve(strict=True)
    expected = Path.home() / 'almadial-lab/evidence/m2-06'
    if e.parent != expected or e.stat().st_mode & 0o077:
        parser.error('private ALM-21 evidence directory required')
    for value in (args.label, args.input, args.plan, args.state):
        if value is not None and not re.fullmatch(r'[a-z0-9-]{1,80}', value):
            parser.error('invalid evidence label')

    def run(label, argv):
        p = subprocess.run(argv, capture_output=True, timeout=900)
        (e / (label + '.stdout')).open('xb').write(p.stdout)
        (e / (label + '.stderr')).open('xb').write(p.stderr)
        (e / (label + '.exit')).open('x').write(str(p.returncode) + '\n')
        if p.returncode:
            raise RuntimeError(f'{label}: exit {p.returncode}; inspect protected stderr')
        return p.stdout

    def guest(label, code):
        prefix = f'import pathlib; assert pathlib.Path("/etc/machine-id").read_text().strip()=={MACHINE!r}; '
        return run(label, ['multipass', 'exec', VM, '--', 'sudo', 'python3', '-c', prefix + code])

    if args.mode == 'deploy':
        if args.binary is None:
            parser.error('--binary required')
        data = args.binary.read_bytes()
        digest = hashlib.sha256(data).hexdigest()
        run(args.label + '-transfer', ['multipass', 'transfer', str(args.binary), VM + ':/home/ubuntu/alm21-runtime-upload'])
        guest(args.label, f'import os,hashlib,shutil; os.umask(0o077); p=pathlib.Path("/home/ubuntu/alm21-runtime-upload"); assert hashlib.sha256(p.read_bytes()).hexdigest()=={digest!r}; pathlib.Path({GUEST!r}).mkdir(mode=0o700,exist_ok=True); shutil.copyfile(p,{(GUEST+"almadial-runtime")!r}); os.chmod({(GUEST+"almadial-runtime")!r},0o700); print({digest!r})')
        return
    argv = [GUEST + 'almadial-runtime', args.mode, *COMMON]
    if args.mode == 'inspect':
        argv += ['--accepted-vicidial', '/tmp/alm20-qualification/sealed-final-inspect.json']
    else:
        if args.input is None:
            parser.error('--input required')
        argv += ['--input', GUEST + args.input + '.json']
    if args.mode in ('lab-run', 'lab-resume'):
        if args.plan is None or args.state is None:
            parser.error('--plan and --state required')
        plan = json.loads((e / (args.plan + '.json')).read_bytes())
        stage = plan['stage']
        snapshot = (e / 'snapshot-info.stdout').read_bytes()
        info = json.loads(snapshot)['info'][VM]['snapshots']['alm21-pre-runtime-20261110']
        assert 'ALM-21 explicitly authorized' in info['comment']
        authorization = dict(Scope='ALM-21/disposable-lab', Host=stage['Before']['Host'],
                             StageSHA256=plan['stage_sha256'], ParentSHA256=stage['Parent'],
                             ManifestSHA256=stage['Manifest'], PolicySHA256=stage['Policy'],
                             SnapshotReference=hashlib.sha256(snapshot).hexdigest(),
                             OperatorEvidenceSHA256=hashlib.sha256((e / 'authorization.txt').read_bytes()).hexdigest(),
                             CandidateAcknowledged=True, DialingInactive=True)
        raw = json.dumps(authorization).encode()
        (e / (args.label + '-authorization.json')).open('xb').write(raw)
        guest(args.label + '-authorization', f'import os; os.umask(0o077); f=open({(GUEST+args.label+"-authorization.json")!r},"xb"); f.write({raw!r}); f.flush(); os.fsync(f.fileno()); f.close()')
        argv += ['--authorization', GUEST + args.label + '-authorization.json', '--state-dir', '/var/lib/almadial/runtime-s06-' + args.state]
    code = f'import subprocess,os,json; os.umask(0o077); p=subprocess.run({argv!r},capture_output=True,timeout=840); f=open({(GUEST+args.label+".json")!r},"xb"); f.write(p.stdout); f.flush(); os.fsync(f.fileno()); f.close(); print(p.stderr.decode(),end=""); raise SystemExit(p.returncode)'
    guest(args.label + '-execute', code)
    export = '/home/ubuntu/alm21-' + args.label + '.json'
    guest(args.label + '-export', f'import shutil,os,pwd; shutil.copyfile({(GUEST+args.label+".json")!r},{export!r}); u=pwd.getpwnam("ubuntu"); os.chown({export!r},u.pw_uid,u.pw_gid); os.chmod({export!r},0o600)')
    destination = e / (args.label + '.json')
    if destination.exists():
        raise RuntimeError('evidence destination already exists')
    run(args.label + '-read', ['multipass', 'transfer', VM + ':' + export, str(destination)])
    raw = destination.read_bytes()
    guest(args.label + '-remove-export', f'pathlib.Path({export!r}).unlink()')
    result = json.loads(raw)
    print(json.dumps({'mode': args.mode, 'label': args.label, 'no_op': result.get('no_op'), 'actions': len(result.get('traces', result.get('graph', [])))}))


if __name__ == '__main__':
    main()
