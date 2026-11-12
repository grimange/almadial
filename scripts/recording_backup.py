#!/usr/bin/env python3
"""Bounded, explicit-file backup verifier for the manual ALM-33 restore drill.

No database operations, service changes, automatic discovery or live-path restore.
Archive digests prove integrity, not publisher authenticity. Keep archives private.
"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import stat
import tarfile

MAX_BYTES = 1024 * 1024 * 1024
MAX_FILES = 10000


def safe_path(name):
    p = PurePosixPath(name)
    if not name or p.is_absolute() or any(x in ('', '.', '..') for x in name.split('/')):
        raise ValueError('unsafe relative member path')
    return p


def no_links(path):
    path = Path(os.path.abspath(path))
    for p in [*reversed(path.parents), path]:
        if p.is_symlink():
            raise ValueError('symlink path rejected')
    return path


def digest(data):
    return hashlib.sha256(data).hexdigest()


def create(root, paths, output):
    root = no_links(root)
    output = no_links(output)
    if not root.is_dir() or not output.parent.is_dir():
        raise ValueError('source root and output parent must exist')
    records, payload, total = [], [], 0
    if not paths or len(paths) > MAX_FILES or len(set(paths)) != len(paths):
        raise ValueError('empty, duplicate or excessive inventory')
    for name in sorted(paths):
        safe_path(name)
        p = no_links(root / name)
        before = p.stat()
        if not stat.S_ISREG(before.st_mode) or before.st_mode & 0o7000:
            raise ValueError('only ordinary regular files are supported')
        total += before.st_size
        if total > MAX_BYTES:
            raise ValueError('backup size limit exceeded')
        with p.open('rb') as f:
            data = f.read(before.st_size + 1)
        after = p.stat()
        if (before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns) != (
                after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns) or len(data) != before.st_size:
            raise ValueError('source changed while collecting; stop and quiesce')
        records.append(dict(path=name, size=len(data), sha256=digest(data),
                            mode=stat.S_IMODE(before.st_mode), uid=before.st_uid,
                            gid=before.st_gid, mtime_ns=before.st_mtime_ns))
        payload.append(data)
    manifest = dict(format='almadial-manual-backup-v1', files=records)
    encoded = json.dumps(manifest, sort_keys=True, indent=2).encode() + b'\n'
    with output.open('xb') as f:
        os.fchmod(f.fileno(), 0o600)
        with tarfile.open(fileobj=f, mode='w', format=tarfile.USTAR_FORMAT) as archive:
            for name, data in [('manifest.json', encoded)] + [
                    ('data/' + r['path'], b) for r, b in zip(records, payload)]:
                info = tarfile.TarInfo(name)
                info.size, info.mode = len(data), 0o600
                archive.addfile(info, io.BytesIO(data))
    verify(output)
    return manifest


def verify(archive):
    archive = no_links(archive)
    if not archive.is_file() or archive.stat().st_size > MAX_BYTES + 32 * 1024 * 1024:
        raise ValueError('archive size/type invalid')
    with tarfile.open(archive, 'r:') as t:
        members = t.getmembers()
        if not members or len(members) > MAX_FILES + 1:
            raise ValueError('member limit or empty archive')
        names = [m.name for m in members]
        if len(names) != len(set(names)):
            raise ValueError('duplicate archive members')
        total = 0
        for m in members:
            safe_path(m.name)
            total += m.size
            if not m.isfile() or m.size < 0 or total > MAX_BYTES + 8 * 1024 * 1024:
                raise ValueError('unsafe member type or excessive size')
        if 'manifest.json' not in names or t.getmember('manifest.json').size > 8 * 1024 * 1024:
            raise ValueError('missing or excessive manifest')
        manifest = json.load(t.extractfile('manifest.json'))
        if manifest.get('format') != 'almadial-manual-backup-v1':
            raise ValueError('unsupported manifest')
        records = manifest['files']
        if not records or len(records) > MAX_FILES:
            raise ValueError('invalid inventory')
        expected = ['manifest.json']
        for r in records:
            safe_path(r['path'])
            for key in ('size', 'mode', 'uid', 'gid', 'mtime_ns'):
                if type(r[key]) is not int or r[key] < 0:
                    raise ValueError('invalid numeric metadata')
            if r['mode'] > 0o777:
                raise ValueError('unsafe file mode')
            name = 'data/' + r['path']
            expected.append(name)
            if name not in names or t.getmember(name).size != r['size']:
                raise ValueError('missing or size-mismatched required member')
            if digest(t.extractfile(name).read()) != r['sha256']:
                raise ValueError('checksum mismatch')
        if len(expected) != len(set(expected)) or set(expected) != set(names):
            raise ValueError('unexpected or duplicate inventory')
    return manifest


def restore(archive, destination):
    manifest = verify(archive)  # Validate the entire archive before any output.
    destination = no_links(destination)
    if not destination.parent.is_dir():
        raise ValueError('destination parent must already exist')
    # Fresh directory only: a repeat or conflicting destination is never merged.
    destination.mkdir(mode=0o700)
    with tarfile.open(archive, 'r:') as t:
        for r in manifest['files']:
            p = destination / r['path']
            p.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            with p.open('xb') as f:
                os.fchmod(f.fileno(), 0o600)
                f.write(t.extractfile('data/' + r['path']).read())
                os.fchown(f.fileno(), r['uid'], r['gid'])
                os.fchmod(f.fileno(), r['mode'])
            os.utime(p, ns=(r['mtime_ns'], r['mtime_ns']))
            s = p.stat()
            if (digest(p.read_bytes()), stat.S_IMODE(s.st_mode), s.st_uid, s.st_gid,
                    s.st_mtime_ns) != (r['sha256'], r['mode'], r['uid'], r['gid'], r['mtime_ns']):
                raise ValueError('restored content or metadata mismatch; quarantine destination')
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    c = sub.add_parser('create')
    c.add_argument('--root', required=True)
    c.add_argument('--inventory', required=True, help='JSON array of explicit relative regular-file paths')
    c.add_argument('--output', required=True)
    v = sub.add_parser('verify')
    v.add_argument('archive')
    r = sub.add_parser('restore')
    r.add_argument('archive')
    r.add_argument('--destination', required=True, help='new isolated directory; never a live system root')
    args = parser.parse_args()
    try:
        if args.command == 'create':
            result = create(args.root, json.loads(Path(args.inventory).read_text()), args.output)
        elif args.command == 'verify':
            result = verify(args.archive)
        else:
            result = restore(args.archive, args.destination)
        print(json.dumps(dict(result='PASS', files=len(result['files']),
                              bytes=sum(r['size'] for r in result['files']))))
        return 0
    except (OSError, ValueError, KeyError, TypeError, tarfile.TarError) as error:
        # Avoid echoing arbitrary paths, file contents or credential-bearing input.
        print(json.dumps(dict(result='REJECT', reason=type(error).__name__)))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
