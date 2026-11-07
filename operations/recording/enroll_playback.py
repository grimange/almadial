#!/usr/bin/python3
"""Root-only projection of verified native receipts; no audio copy or DB writes.

Grants the existing backend group read access to individual finalized inodes.
Records prior file mode/group for scoped rollback. Raw files are never touched.
"""
import fcntl
import grp
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import wave

STATE = Path('/var/lib/alma-vicidial-recording')
TARGET = Path('/var/lib/alma-recording-playback')
ROOT = Path('/var/spool/asterisk/monitorDONE')
ORIGIN = 'https://2.29.153.161/RECORDINGS/'
ROOT_UID = 0


def fingerprint(s):
    return [s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns, s.st_ctime_ns]


def enroll(record, receipt, gid):
    rid, name, end, location = record
    rid = int(rid)
    if (rid != receipt['recording_id'] or int(end or 0) <= 0
            or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_-]{0,199}', name)):
        raise ValueError('invalid_native_association')
    name += '-all.wav'
    if location != ORIGIN + name:
        raise ValueError('unexpected_native_location')
    output = TARGET / (str(rid) + '.json')
    previous = json.loads(output.read_text()) if output.exists() else None
    directory = os.open(ROOT, os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
    finally:
        os.close(directory)
    with os.fdopen(fd, 'rb') as stream:
        before = os.fstat(stream.fileno())
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or before.st_uid != ROOT_UID:
            raise ValueError('unsafe_final_inode')
        mode = stat.S_IMODE(before.st_mode)
        if mode not in (0o600, 0o640):
            raise ValueError('unexpected_final_permissions')
        digest = hashlib.file_digest(stream, 'sha256').hexdigest()
        if digest != receipt['final_sha256']:
            raise ValueError('final_receipt_hash_mismatch')
        stream.seek(0)
        with wave.open(stream, 'rb') as wav:
            if (wav.getnchannels(), wav.getsampwidth(), wav.getframerate()) != (1, 2, 8000):
                raise ValueError('unsupported_audio')
            if abs(wav.getnframes() / wav.getframerate() - receipt['seconds']) > .02:
                raise ValueError('duration_mismatch')
        if fingerprint(os.fstat(stream.fileno())) != fingerprint(before):
            raise ValueError('final_inode_changed_during_verification')
        # A matching root receipt establishes completed native processing.
        if previous:
            if previous['name'] != name or previous['sha256'] != digest:
                raise ValueError('enrollment_conflict')
            if fingerprint(before) == previous['fingerprint'] and before.st_gid == gid and mode == 0o640:
                return 'already_enrolled'
        original = previous['original'] if previous else dict(gid=before.st_gid, mode=mode)
        # Publish recovery metadata before changing permissions, including on
        # interrupted enrollment. This directory is writable only by root.
        metadata = dict(recording_id=rid, name=name, sha256=digest, original=original,
                        fingerprint=fingerprint(before))
        publish(output, metadata, gid)
        os.fchown(stream.fileno(), ROOT_UID, gid)
        os.fchmod(stream.fileno(), 0o640)
        metadata['fingerprint'] = fingerprint(os.fstat(stream.fileno()))
        publish(output, metadata, gid)
    return 'enrolled'


def publish(path, metadata, gid):
    temp = path.with_suffix('.new')
    fd = os.open(temp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as stream:
        json.dump(metadata, stream, sort_keys=True)
        stream.flush()
        os.fsync(stream.fileno())
        os.fchown(stream.fileno(), ROOT_UID, gid)
        os.fchmod(stream.fileno(), 0o640)
    os.replace(temp, path)


def main():
    if os.geteuid() != 0:
        raise PermissionError('root_required')
    os.umask(0o077)
    gid = grp.getgrnam('alma-portal').gr_gid
    TARGET.mkdir(mode=0o750, exist_ok=True)
    os.chown(TARGET, 0, gid)
    # mkdir's mode is filtered by the private process umask (077).
    # Permit only the existing backend group to traverse verified receipts.
    os.chmod(TARGET, 0o750)
    with (TARGET / '.lock').open('a') as lock, (STATE / 'lock').open('a') as finalizer_lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        fcntl.flock(finalizer_lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        count = 0
        for receipt_path in STATE.glob('[0-9]*.json'):
            if not re.fullmatch(r'[0-9]+\.json', receipt_path.name):
                continue
            receipt = json.loads(receipt_path.read_text())
            rid = int(receipt_path.stem)
            # Only receipts created by the guarded native finalizer qualify;
            # querying one bounded native record is not a second catalog.
            command = ['/usr/bin/mariadb', '--no-defaults', '--batch', '--raw', '--skip-column-names',
                       '-e', 'SELECT recording_id,filename,end_epoch,location FROM asterisk.recording_log '
                       'WHERE recording_id=' + str(rid) + ' LIMIT 2;']
            result = subprocess.run(command, check=True, capture_output=True, text=True, timeout=10)
            rows = result.stdout.splitlines()
            if len(rows) != 1 or len(rows[0].split('\t')) != 4:
                raise ValueError('ambiguous_native_record')
            outcome = enroll(rows[0].split('\t'), receipt, gid)
            count += 1
            print('RECORDING_ENROLL id=' + str(rid) + ' outcome=' + outcome, flush=True)
        print('RECORDING_ENROLL verified=' + str(count), flush=True)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        # Never log native DB stderr, filenames or sensitive data.
        print('RECORDING_ENROLL_FAILED type=' + type(error).__name__, flush=True)
        raise SystemExit(1)
