#!/usr/bin/python3
"""Install into the existing portal only; preserve live carriers.json and services."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys

import carrier_cli as cli

BASE = Path('/opt/alma-portal-443')
CONFIG = BASE / 'carriers.json'
BACKUP_ROOT = Path('/root/qa-backups')
FILES = {'sip_guard.py': BASE / 'sip_guard.py',
         'carrier_cli.py': BASE / 'carrier_cli.py',
         'alma-portal-carriers': Path('/usr/local/sbin/alma-portal-carriers')}
PEEREDGE = '38.147.130.74'
LEGACY = f'rule family="ipv4" source address="{PEEREDGE}/32" port port="5060" protocol="udp" accept'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def contexts():
    subprocess.run(['/usr/sbin/restorecon', *[str(p) for p in FILES.values() if p.exists()]], check=True)


def legacy_set(fw, wanted, permanent):
    present = LEGACY in fw.rich(permanent)
    if present != wanted:
        fw.command('/usr/bin/firewall-cmd', *(('--permanent',) if permanent else ()),
                   '--zone=public', f'--{"add" if wanted else "remove"}-rich-rule={LEGACY}')


def rollback(backup, fw):
    manifest = json.loads((backup / 'manifest.json').read_text())
    if digest(CONFIG) != manifest['config_sha256']:
        raise RuntimeError('Carrier configuration changed since installation; review before rollback')
    failures = []
    # Re-establish pre-existing connectivity before removing new managed rules.
    actions = []
    if manifest['adopt_peeredge']:
        actions += [(f'legacy {permanent}', lambda p=permanent: legacy_set(fw, manifest['legacy'][int(p)], p))
                    for permanent in (True, False)]
    actions += [('permanent rules', lambda: fw.set_rich(set(manifest['permanent']), True)),
                ('runtime rules', lambda: fw.set_rich(set(manifest['runtime']))),
                ('carrier set', lambda: fw.restore_nft(set(manifest['members'])))]
    def restore_files():
        for name, path in FILES.items():
            metadata = manifest['files'][name]
            if metadata is None:
                path.unlink(missing_ok=True)
            else:
                cli.atomic_write(path, (backup / name).read_bytes(), metadata['mode'])
                os.chown(path, metadata['uid'], metadata['gid'])
        contexts()
    actions.append(('files', restore_files))
    for name, action in actions:
        try:
            action()
        except Exception as error:
            failures.append(f'{name}: {error}')
    for name, path in FILES.items():
        old = manifest['files'][name]
        if (path.exists() if old is None else not path.exists() or digest(path) != old['sha256']):
            failures.append(f'{name}: content read-back mismatch')
    expected = (set(manifest['members']), True, set(manifest['runtime']), set(manifest['permanent']))
    if cli.snapshot(fw) != expected:
        failures.append('firewall read-back mismatch')
    if manifest['adopt_peeredge'] and [LEGACY in fw.rich(p) for p in (False, True)] != manifest['legacy']:
        failures.append('legacy rule read-back mismatch')
    if failures:
        raise RuntimeError('ROLLBACK INCOMPLETE: ' + '; '.join(failures))
    print(f'Rollback verified: {backup}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--adopt-peeredge', action='store_true',
                        help='Migrate only the accepted historical PeerEdge rich rule to managed ownership')
    parser.add_argument('--expect-guard-sha256', help='Refuse installation if the deployed guard changed')
    parser.add_argument('--rollback', type=Path, help='Restore this installer backup; refuses changed carrier configuration')
    args = parser.parse_args()
    if os.geteuid() != 0:
        parser.exit(1, 'Root required.\n')
    cli.guard.CARRIER_CONFIG = CONFIG
    fw = cli.Firewall()
    with cli.locked():
        if args.rollback:
            rollback(args.rollback, fw)
            return
        configured, raw, _ = cli.load(CONFIG)
        if args.expect_guard_sha256 and digest(FILES['sip_guard.py']) != args.expect_guard_sha256:
            raise RuntimeError('Deployed guard differs from the inspected baseline')
        before = cli.snapshot(fw)
        if not before[1] or set(configured) != before[0]:
            raise RuntimeError('Existing carrier guard policy/membership must be healthy before installing')
        if args.adopt_peeredge and PEEREDGE not in configured:
            raise RuntimeError('Cannot adopt PeerEdge rule without its configured address')
        backup = BACKUP_ROOT / ('carrier-cli-' + datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
        backup.mkdir(mode=0o700, parents=True)
        manifest = {'config_sha256': digest(CONFIG), 'members': sorted(before[0]),
                    'runtime': sorted(before[2]), 'permanent': sorted(before[3]),
                    'legacy': [LEGACY in fw.rich(p) for p in (False, True)],
                    'adopt_peeredge': args.adopt_peeredge, 'files': {}}
        for name, path in FILES.items():
            if path.is_symlink():
                raise RuntimeError(f'Refusing symlink target: {path}')
            if path.exists():
                info = path.stat()
                if not stat.S_ISREG(info.st_mode) or info.st_uid != cli.CONFIG_OWNER or info.st_mode & 0o022:
                    raise RuntimeError(f'Unsafe existing file: {path}')
                manifest['files'][name] = {'mode': stat.S_IMODE(info.st_mode), 'uid': info.st_uid,
                                           'gid': info.st_gid, 'sha256': digest(path)}
                (backup / name).write_bytes(path.read_bytes())
            else:
                manifest['files'][name] = None
        (backup / 'carriers.json').write_bytes(raw)
        (backup / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
        print(f'Backup: {backup}', flush=True)
        try:
            for name, path in FILES.items():
                cli.atomic_write(path, Path(__file__).with_name(name).read_bytes(),
                                 0o755 if name == 'alma-portal-carriers' else 0o644)
            contexts()
            cli.execute('sync', path=CONFIG, firewall=fw)
            if args.adopt_peeredge:
                legacy_set(fw, False, False)
                legacy_set(fw, False, True)
            if digest(CONFIG) != manifest['config_sha256']:
                raise RuntimeError('Installer unexpectedly changed carrier configuration')
            if cli.execute('validate', path=CONFIG, firewall=fw):
                raise RuntimeError('Post-install verification failed')
            for name, path in FILES.items():
                if digest(path) != digest(Path(__file__).with_name(name)):
                    raise RuntimeError(f'Installed content mismatch: {path}')
                print(f'Installed: {path} sha256={digest(path)}')
        except Exception:
            rollback(backup, fw)
            raise
    print('Installation verified. Live carriers.json preserved; no services restarted.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print(f'Error: {error}', file=sys.stderr)
        sys.exit(1)
