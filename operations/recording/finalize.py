#!/usr/bin/python3
"""Private, fail-closed staging for the unmodified native VICIdial VD-only job.

No SQL writes: only the native script updates location. No HTTP media service.
The input is already a conference mix; a nonempty output stream is unsupported.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import tempfile
import time
import wave

CONFIG = Path('/etc/alma-vicidial-recording.json')
STATE = Path('/var/lib/alma-vicidial-recording')
RAW = Path('/var/spool/asterisk/monitor')
DONE = Path('/var/spool/asterisk/monitorDONE')
NATIVE = Path('/usr/share/astguiclient/AST_CRON_audio_1_move_VDonly.pl')
NATIVE_SHA = 'a9837929fcda93d4ca238eb0f06ac5d2fc3198c3029c2e6132f29fe3091c25c0'
ORIGIN = 'https://2.29.153.161/RECORDINGS/'


class Refused(Exception):
    pass


def command(argv, timeout=30):
    try:
        p = subprocess.run(argv, capture_output=True, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise Refused('command_unavailable_or_timed_out') from None
    if p.returncode:
        # Never log arbitrary native diagnostics, paths, SQL or credentials.
        raise Refused('command_failed')
    return p.stdout.decode()


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for block in iter(lambda: f.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def atomic_json(path, data):
    tmp = path.with_suffix('.new')
    with tmp.open('w') as f:
        json.dump(data, f, indent=2)
        f.flush()
        os.fsync(f.fileno())
    tmp.chmod(0o600)
    os.replace(tmp, path)


def rows(recording_id=None, boundary=0):
    where = (f'recording_id={int(recording_id)}' if recording_id else
             f"start_epoch>={int(boundary)} AND (location IS NULL OR location='')")
    sql = ('SELECT recording_id,filename,start_epoch,COALESCE(end_epoch,0),'
           'COALESCE(length_in_sec,0),COALESCE(location,\'\') FROM asterisk.recording_log WHERE '
           + where + ' ORDER BY recording_id LIMIT 20;')
    output = command(['/usr/bin/mariadb', '--no-defaults', '--batch', '--raw',
                      '--skip-column-names', '-e', sql])
    result = []
    for line in output.splitlines():
        fields = line.split('\t')
        if len(fields) != 6:
            raise Refused('invalid_database_result')
        rid, name, start, end, length, location = fields
        if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_-]{0,199}', name):
            raise Refused('unsafe_recording_name')
        result.append(dict(id=int(rid), name=name, start=int(start), end=int(end),
                           length=int(length), location=location))
    return result


def snapshot(path):
    s = path.lstat()
    if not stat.S_ISREG(s.st_mode) or s.st_nlink != 1:
        raise Refused('nonregular_or_linked_input')
    return (s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns, s.st_ctime_ns)


def writer_open(paths):
    """All-process writable-fd check; a disappearing process is harmless."""
    targets = {(p.stat().st_dev, p.stat().st_ino) for p in paths}
    for process in Path('/proc').glob('[0-9]*'):
        try:
            for fd in (process / 'fd').iterdir():
                try:
                    s = fd.stat()
                    if (s.st_dev, s.st_ino) in targets:
                        flags = (process / 'fdinfo' / fd.name).read_text()
                        value = re.search(r'^flags:\s+([0-7]+)$', flags, re.M)
                        if value is None or int(value[1], 8) & os.O_ACCMODE:
                            return True
                except FileNotFoundError:
                    continue
        except FileNotFoundError:
            continue
    return False


def audio(path, allow_empty=False):
    snapshot(path)
    try:
        with wave.open(str(path), 'rb') as w:
            info = dict(channels=w.getnchannels(), rate=w.getframerate(),
                        width=w.getsampwidth(), frames=w.getnframes())
            if (info['channels'], info['rate'], info['width']) != (1, 8000, 2):
                raise Refused('unsupported_conference_wav_format')
            if not allow_empty and not info['frames']:
                raise Refused('empty_input_audio')
            remaining = info['frames'] * 2
            while remaining:
                block = w.readframes(min(65536, remaining // 2))
                if not block:
                    raise Refused('truncated_wav')
                remaining -= len(block)
    except (wave.Error, EOFError):
        raise Refused('invalid_wav') from None
    return info


def eligible(row, now):
    return row['end'] > row['start'] > 0 and row['length'] > 0 and now - row['end'] >= 60


def process(row, state=STATE, raw=RAW, done=DONE):
    if not eligible(row, time.time()):
        return 'waiting_for_closed_recording'
    unique = command(['/usr/bin/mariadb', '--no-defaults', '-N', '-B', '-e',
                      "SELECT COUNT(*) FROM asterisk.recording_log WHERE filename='" + row['name'] + "';"]).strip()
    if unique != '1':
        raise Refused('duplicate_recording_name')
    source = raw / (row['name'] + '-in.wav')
    companion = raw / (row['name'] + '-out.wav')
    final = done / (row['name'] + '-all.wav')
    expected = ORIGIN + final.name
    if row['location'] and row['location'] != expected:
        raise Refused('existing_location_conflict')
    if not source.exists() or not companion.exists():
        raise Refused('raw_pair_missing')
    originals = [source, companion]
    before = [snapshot(p) for p in originals]
    if writer_open(originals) or any(time.time() - p.stat().st_mtime < 60 for p in originals):
        return 'waiting_for_writer'
    info = audio(source)
    out_info = audio(companion, allow_empty=True)
    if out_info['frames']:
        raise Refused('nonempty_output_needs_separate_mixing_review')
    seconds = info['frames'] / info['rate']
    if abs(seconds - row['length']) > max(5, row['length'] * .1):
        raise Refused('duration_mismatch')
    checksums = [digest(p) for p in originals]
    receipt = state / (str(row['id']) + '.json')
    if final.exists():
        snapshot(final)
        if digest(final) != checksums[0]:
            raise Refused('final_collision')
        if row['location'] == expected and receipt.exists():
            old = json.loads(receipt.read_text())
            if old.get('raw_sha256') != checksums or old.get('final_sha256') != checksums[0]:
                raise Refused('receipt_conflict')
            return 'already_finalized'
    free = shutil.disk_usage(done).free
    if free < sum(x[2] for x in before) * 2 + 1024**3:
        raise Refused('insufficient_recording_capacity')
    # Native -s switches initialize existing upstream CLI* override variables.
    # The source and live /etc/astguiclient.conf are never modified.
    with tempfile.TemporaryDirectory(prefix='stage-', dir=state) as tmp:
        stage = Path(tmp)
        for p in originals:
            with p.open('rb') as src, (stage / p.name).open('xb') as dest:
                shutil.copyfileobj(src, dest)
                dest.flush()
                os.fsync(dest.fileno())
            (stage / p.name).chmod(0o600)
        if before != [snapshot(p) for p in originals] or writer_open(originals):
            raise Refused('input_changed_during_staging')
        current = rows(row['id'])
        if current != [row]:
            raise Refused('recording_metadata_changed')
        for p, sha in zip(originals, checksums):
            if digest(stage / p.name) != sha:
                raise Refused('staged_copy_mismatch')
        command(['/usr/bin/perl', '-s', str(NATIVE), '-CLImonitor=1',
                 '-PATHmonitor=' + str(stage), '-CLIDONEmonitor=1',
                 '-PATHDONEmonitor=' + str(done), '--', '--HTTPS', '--quiet'], timeout=90)
    if before != [snapshot(p) for p in originals] or [digest(p) for p in originals] != checksums:
        raise Refused('originals_changed')
    if audio(final) != info or digest(final) != checksums[0]:
        raise Refused('native_output_invalid')
    final.chmod(0o600)
    # Decoder verifies the whole stream; native VDonly moves rather than converts.
    command(['/usr/bin/sox', str(final), '-n', 'stat'])
    actual = rows(row['id'])
    if len(actual) != 1 or actual[0] != dict(row, location=expected):
        raise Refused('native_database_update_not_confirmed')
    with final.open('rb') as f:
        os.fsync(f.fileno())
    atomic_json(receipt, dict(recording_id=row['id'], final_sha256=checksums[0],
                             raw_sha256=checksums, seconds=seconds, at=int(time.time()),
                             publication='disabled_pending_individual_authorization'))
    return 'finalized_private'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--recording-id', type=int, help='Explicit historical recovery, originals retained')
    parser.add_argument('--health', action='store_true')
    args = parser.parse_args()
    os.umask(0o077)
    if os.geteuid() != 0:
        raise Refused('root_required')
    STATE.mkdir(mode=0o700, exist_ok=True)
    state_info = STATE.lstat()
    if not stat.S_ISDIR(state_info.st_mode) or state_info.st_uid != 0 or state_info.st_mode & 0o077:
        raise Refused('private_state_permissions_required')
    if args.health:
        status = json.loads((STATE / 'status.json').read_text())
        status['age_seconds'] = int(time.time()) - status['at']
        print(json.dumps(status, indent=2))
        return 0 if status['outcome'] == 'ok' and status['age_seconds'] < 600 else 1
    with (STATE / 'lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise Refused('processing_already_running') from None
        status = dict(at=int(time.time()), outcome='running', results=[])
        atomic_json(STATE / 'status.json', status)
        try:
            config_info = CONFIG.lstat()
            if not stat.S_ISREG(config_info.st_mode) or config_info.st_uid != 0 or config_info.st_mode & 0o077:
                raise Refused('private_configuration_permissions_required')
            config = json.loads(CONFIG.read_text())
            if digest(NATIVE) != NATIVE_SHA:
                raise Refused('native_source_changed_review_required')
            if not Path('/usr/bin/sox').is_file() or not Path('/usr/bin/soxi').is_file():
                raise Refused('audio_tools_missing')
            settings = command(['/usr/bin/mariadb', '--no-defaults', '-N', '-B', '-e',
                                'SELECT stereo_recording,stereo_parallel_recording FROM asterisk.system_settings;']).strip()
            if settings != '0\t0':
                raise Refused('stereo_mode_requires_separate_review')
            batch = rows(args.recording_id, config['automatic_start_epoch'])
            if args.recording_id and not batch:
                raise Refused('recording_not_found')
            if len({r['name'] for r in batch}) != len(batch):
                raise Refused('duplicate_recording_name')
            for row in batch:
                status['results'].append(dict(id=row['id'], result=process(row)))
            status['outcome'] = 'ok'
        except Exception as e:
            status['outcome'] = 'failed'
            status['reason'] = str(e) if isinstance(e, Refused) else type(e).__name__
            raise
        finally:
            status['at'] = int(time.time())
            atomic_json(STATE / 'status.json', status)
            print(json.dumps(status))
    return 0


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as e:
        print(json.dumps(dict(outcome='failed', reason=str(e) if isinstance(e, Refused) else type(e).__name__)))
        raise SystemExit(1)
