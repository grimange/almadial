#!/usr/bin/env python3
"""Verify and retain exact approved archives in a private content-addressed store.

Offline maintainer tool: never downloads, repins, publishes or grants rights.
Intact archive licenses/notices are retained; private lab use only for ALM-34.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from prepare_compliance_source import archive_files

ROOT = Path(__file__).resolve().parents[2]
NAMES = {'asterisk-22.12.0.tar.gz':'asterisk', 'pjproject-2.17.tar.bz2':'pjproject',
         'Net-Telnet-3.05.tar.gz':'net-telnet', 'vicidial-r4037.zip':'vicidial',
         'viciphone.tar.gz':'viciphone'}

def private_directory(path):
    try:
        path.mkdir(mode=0o700)
    except FileExistsError:
        pass
    st = path.lstat()
    if path.is_symlink() or not path.is_dir() or st.st_mode & 0o077 or st.st_uid != os.geteuid():
        raise ValueError('expected owned private store directory')

def retain(archives, store):
    inventory = json.loads((ROOT/'docs/compliance/m4-04/source-files.json').read_text())
    # Verify every original archive and the full independent file inventory
    # before publishing any new object. No source reconstruction/substitution.
    verified = [(name, *archive_files(archives/name, inventory[name])) for name in NAMES]
    private_directory(store)
    private_directory(store/'sha256')
    rows = []
    for name, raw, files in verified:
        sha = hashlib.sha256(raw).hexdigest()
        directory = store/'sha256'/sha
        private_directory(directory)
        target = directory/(NAMES[name]+'.artifact')
        try:
            fd = os.open(target, os.O_CREAT|os.O_EXCL|os.O_WRONLY, 0o600)
        except FileExistsError:
            st = target.lstat()
            if target.is_symlink() or not target.is_file() or st.st_mode & 0o077 or st.st_uid != os.geteuid() or target.read_bytes() != raw:
                raise ValueError('retained object differs or has unsafe metadata; refusing replacement')
        else:
            with os.fdopen(fd, 'wb') as f:
                f.write(raw)
                f.flush()
                os.fsync(f.fileno())
            target.chmod(0o400)
        rows.append(dict(component=NAMES[name], original_archive=name, sha256=sha,
                         bytes=len(raw), regular_files=len(files), path=str(target),
                         verification='exact archive SHA-256 plus accepted ALM-32 file inventory'))
    return rows

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--archives',type=Path,required=True)
    p.add_argument('--store',type=Path,required=True)
    a=p.parse_args()
    print(json.dumps(retain(a.archives,a.store),indent=2))

if __name__=='__main__':
    main()
