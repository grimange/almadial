#!/usr/bin/python3
"""Root-only, idle-window logical backup. No restore or deletion command."""
import argparse
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import resource
import shutil
import signal
import socket
import sqlite3
import stat
import subprocess
import sys
import tarfile
import tempfile
import time

CONFIG = Path('/etc/alma-vicidial-backup.json')
PAYLOADS = ('database.sql', 'files.tar', 'astdb.sqlite3', 'inventory.json', 'manifest.json')


class BackupError(Exception):
    pass


def utc():
    return dt.datetime.now(dt.timezone.utc).isoformat()


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def write_json(path, value):
    temporary = path.with_suffix('.new')
    with temporary.open('w') as stream:
        os.fchmod(stream.fileno(), 0o600)
        json.dump(value, stream, indent=2)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def private_dir(path):
    if path.is_symlink():
        raise BackupError('symlink_destination')
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    s = path.stat()
    if s.st_uid != 0 or stat.S_IMODE(s.st_mode) != 0o700:
        raise BackupError('destination_not_root_private')


def run(argv, timeout=20, output=None):
    """Do not propagate subprocess output: it may contain sensitive SQL or paths."""
    with subprocess.Popen(argv, stdout=output or subprocess.PIPE,
                          stderr=subprocess.DEVNULL, start_new_session=True) as process:
        try:
            stdout, _ = process.communicate(timeout=timeout)
        except BaseException:
            # Killing the client closes its connection and releases its read lock.
            if process.poll() is None:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass  # Child exited between poll and kill; wait still reaps it.
            process.wait()
            raise
        if process.returncode:
            raise BackupError(Path(argv[0]).name + '_nonzero')
    return stdout.decode() if output is None else ''


def sql(config, query):
    return run(['mariadb', '--no-defaults', '--protocol=socket',
                '--socket=' + config['socket'], '--user=root', '--batch',
                '--skip-column-names', '--execute=' + query])


def idle(config):
    channels = run(['asterisk', '-rx', 'core show channels count'])
    if not re.search(r'^0 active channels$', channels, re.M):
        raise BackupError('active_asterisk_channels')
    counts = sql(config, "SELECT COUNT(*) FROM asterisk.vicidial_live_agents; "
                 "SELECT COUNT(*) FROM asterisk.vicidial_auto_calls; "
                 "SELECT COUNT(*) FROM information_schema.PROCESSLIST "
                 "WHERE ID<>CONNECTION_ID() AND COMMAND NOT IN ('Sleep','Daemon'); "
                 "SELECT COUNT(*) FROM information_schema.INNODB_TRX;")
    if counts.split() != ['0'] * 4:
        raise BackupError('active_agents_calls_or_database_work')


def scan(roots, hashes=False):
    """Do not follow symlinks; reject special files and bound inventory size."""
    result = {}
    for root in roots:
        paths = [Path(root)]
        if paths[0].is_dir() and not paths[0].is_symlink():
            paths += list(paths[0].rglob('*'))
        for path in paths:
            s = path.lstat()
            if not (stat.S_ISREG(s.st_mode) or stat.S_ISDIR(s.st_mode) or stat.S_ISLNK(s.st_mode)):
                raise BackupError('special_file_in_dataset')
            if stat.S_ISREG(s.st_mode):
                result[str(path).lstrip('/')] = dict(size=s.st_size, mtime_ns=s.st_mtime_ns)
                if hashes:
                    result[str(path).lstrip('/')]['sha256'] = digest(path)
            if len(result) > 200000:
                raise BackupError('too_many_dataset_files')
    return result


def check_capacity(config, base, source_bytes, database_bytes):
    complete = list((base / 'complete').iterdir())
    if len(complete) >= config['max_complete_backups']:
        raise BackupError('retention_cap_requires_operator_review')
    pool_bytes = sum(p.stat().st_size for p in base.rglob('*') if p.is_file())
    estimate = source_bytes + database_bytes * 4 + 64 * 1024 * 1024
    fs = os.statvfs(base)
    if estimate > config['max_run_bytes']:
        raise BackupError('dataset_exceeds_run_budget')
    if pool_bytes + estimate > config['max_pool_bytes']:
        raise BackupError('backup_pool_budget_exceeded')
    if fs.f_bavail * fs.f_frsize < estimate + config['reserve_bytes'] or fs.f_favail < 10000:
        raise BackupError('insufficient_free_space_or_inodes')
    return dict(source_bytes=source_bytes, estimated_bytes=estimate, pool_bytes=pool_bytes,
                free_bytes=fs.f_bavail * fs.f_frsize, free_inodes=fs.f_favail)


def dump_database(config, workspace):
    argv = ['mariadb-dump', '--no-defaults', '--protocol=socket',
            '--socket=' + config['socket'], '--user=root', '--all-databases',
            '--lock-all-tables', '--quick', '--routines', '--events', '--triggers',
            '--hex-blob', '--default-character-set=utf8mb4']
    start = time.monotonic()
    with (workspace / 'database.sql').open('wb') as stream:
        run(argv, timeout=config['dump_timeout_seconds'], output=stream)
    # Duration includes client startup and lock acquisition: an upper bound on hold time.
    return dict(method='mariadb-dump --all-databases --lock-all-tables',
                timeout_seconds=config['dump_timeout_seconds'],
                lock_hold_upper_bound_seconds=round(time.monotonic() - start, 4),
                completed_at=utc())


def inspect_dump(path):
    tables = set()
    databases = set()
    current = None
    footer = False
    with path.open('rb') as stream:
        for line in stream:
            if line.startswith(b'-- Current Database: `'):
                current = line.split(b'`')[1].decode()
                databases.add(current)
            table = re.match(rb'CREATE TABLE (?:IF NOT EXISTS )?`([^`]+)`', line)
            if table:
                tables.add((current, table.group(1).decode()))
            if line.startswith(b'-- Dump completed on '):
                footer = True
    if not footer or not {'asterisk', 'mysql'}.issubset(databases):
        raise BackupError('incomplete_database_dump')
    if ('mysql', 'global_priv') not in tables:
        raise BackupError('database_account_recovery_missing')
    return tables


def verify(directory):
    manifest = json.loads((directory / 'manifest.json').read_text())
    expected = dict(line.split('  ', 1)[::-1] for line in
                    (directory / 'SHA256SUMS').read_text().splitlines())
    if set(expected) != set(PAYLOADS):
        raise BackupError('unexpected_payload_set')
    for name in PAYLOADS:
        path = directory / name
        if path.is_symlink() or not path.is_file() or digest(path) != expected[name]:
            raise BackupError('payload_checksum_mismatch')
    tables = inspect_dump(directory / 'database.sql')
    inventory = json.loads((directory / 'inventory.json').read_text())
    wanted = {(r[0], r[1]) for r in inventory['tables'] if r[2] != 'VIEW' and r[0] != 'sys'}
    # Native --all-databases omits generated sys diagnostics; recreate with matching RPMs.
    if not wanted.issubset(tables):
        raise BackupError('database_table_definition_missing')
    recording_seen = set()
    member_names = set()
    member_count = 0
    with tarfile.open(directory / 'files.tar', 'r:') as archive:
        for member in archive:
            p = PurePosixPath(member.name)
            if p.is_absolute() or '..' in p.parts:
                raise BackupError('unsafe_archive_member')
            member_count += 1
            member_names.add(member.name.rstrip('/'))
            if member.isfile():
                h = hashlib.sha256()
                with archive.extractfile(member) as content:
                    for block in iter(lambda: content.read(1024 * 1024), b''):
                        h.update(block)
                if member.name in manifest['recording_files']:
                    if h.hexdigest() != manifest['recording_files'][member.name]['sha256']:
                        raise BackupError('recording_content_mismatch')
                    recording_seen.add(member.name)
    if recording_seen != set(manifest['recording_files']):
        raise BackupError('recording_member_missing')
    if not set(manifest['required_archive_members']).issubset(member_names):
        raise BackupError('required_recovery_member_missing')
    with sqlite3.connect('file:' + str(directory / 'astdb.sqlite3') + '?mode=ro', uri=True) as db:
        if db.execute('PRAGMA integrity_check').fetchone()[0] != 'ok':
            raise BackupError('astdb_integrity_failure')
    if member_count != manifest['archive_members']:
        raise BackupError('archive_member_count_mismatch')
    return dict(verified_at=utc(), payloads=len(PAYLOADS), archive_members=member_count,
                database_tables=len(tables), recording_files=len(recording_seen))


def capture(config, base):
    idle(config)
    required = config['required']
    if any(not Path(p).exists() for p in required):
        raise BackupError('required_dataset_missing')
    roots = required + [p for p in config['optional'] if Path(p).exists()]
    source = scan(roots)
    rows = sql(config, "SELECT TABLE_SCHEMA,TABLE_NAME,COALESCE(ENGINE,'VIEW'),COALESCE(TABLE_ROWS,0) "
               "FROM information_schema.TABLES WHERE TABLE_SCHEMA NOT IN "
               "('information_schema','performance_schema') ORDER BY TABLE_SCHEMA,TABLE_NAME;")
    db_size = int(sql(config, "SELECT COALESCE(SUM(DATA_LENGTH+INDEX_LENGTH),0) "
                     "FROM information_schema.TABLES WHERE TABLE_SCHEMA NOT IN "
                     "('information_schema','performance_schema');").strip())
    capacity = check_capacity(config, base, sum(x['size'] for x in source.values()), db_size)
    recordings_before = scan([p for p in config['recording_roots'] if Path(p).exists()], hashes=True)
    identifier = dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%S') + '-' + os.urandom(4).hex()
    workspace = Path(tempfile.mkdtemp(prefix='.partial-' + identifier + '-', dir=base))
    try:
        inventory = dict(tables=[line.split('\t') for line in rows.splitlines()],
            mariadb=sql(config, 'SELECT VERSION();').strip(),
            database_settings=sql(config, "SHOW VARIABLES WHERE Variable_name IN "
                "('sql_mode','character_set_server','collation_server','log_bin');").splitlines(),
            packages=run(['rpm', '-qa', '--qf', '%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}\n']).splitlines(),
            asterisk=run(['asterisk', '-rx', 'core show version']).strip(),
            kernel=run(['uname', '-r']).strip(),
            excluded=['running channels, sockets, caches, outgoing call queue, general logs',
                      'historical /root dumps and rollback archives (left untouched)',
                      'raw MariaDB datadir (logical backup instead)',
                      'generated sys diagnostics schema (native all-databases exclusion; recreate from matching MariaDB RPMs)',
                      'live AstDB file (online SQLite backup instead)'])
        write_json(workspace / 'inventory.json', inventory)
        idle(config)
        database = dump_database(config, workspace)
        idle(config)
        with sqlite3.connect('file:/var/lib/asterisk/astdb.sqlite3?mode=ro', uri=True) as src:
            with sqlite3.connect(workspace / 'astdb.sqlite3') as dst:
                src.backup(dst)
        run(['tar', '--acls', '--xattrs', '--selinux', '--numeric-owner',
             '--exclude=var/lib/asterisk/astdb.sqlite3*', '--exclude=*/__pycache__',
             '-C', '/', '-cpf', str(workspace / 'files.tar'),
             *[p.lstrip('/') for p in roots]], timeout=600)
        idle(config)
        if recordings_before != scan([p for p in config['recording_roots'] if Path(p).exists()], hashes=True):
            raise BackupError('recordings_changed_during_capture')
        with tarfile.open(workspace / 'files.tar') as archive:
            members = sum(1 for _ in archive)
        manifest = dict(format='alma-vicidial-backup/v1', id=identifier, host=socket.gethostname(),
            created_at=utc(), database=database, included_roots=roots,
            absent_optional=[p for p in config['optional'] if not Path(p).exists()],
            recording_files=recordings_before, capacity=capacity, archive_members=members,
            archive_bytes=(workspace / 'files.tar').stat().st_size,
            required_archive_members=[p.lstrip('/') for p in roots] + [
                'usr/lib64/asterisk/modules/res_monitor.so',
                'usr/lib64/asterisk/modules/chan_sip.so',
                'usr/share/astguiclient/ADMIN_keepalive_ALL.pl',
                'usr/share/astguiclient/AST_manager_send.pl',
                'usr/share/astguiclient/AST_VDauto_dial.pl',
                'usr/share/astguiclient/AST_VDremote_agents.pl',
                'etc/my.cnf.d/91-vicidial-sql-compat.cnf',
                'usr/local/libexec/alma-portal-tls',
                'usr/local/sbin/alma-portal-cert-deploy'],
            recording_consistency='idle checks plus stable per-file hashes before/after capture; not an atomic cross-system snapshot',
            portal_state='for recovery evidence only; do not reactivate archived grants or sessions',
            memory_tables='captured, but runtime state must be invalidated on isolated recovery before application startup')
        write_json(workspace / 'manifest.json', manifest)
        (workspace / 'SHA256SUMS').write_text(''.join(digest(workspace / p) + '  ' + p + '\n' for p in PAYLOADS))
        total = sum(p.stat().st_size for p in workspace.iterdir())
        if total > config['max_run_bytes']:
            raise BackupError('completed_backup_exceeds_run_budget')
        verified = verify(workspace)
        write_json(workspace / 'VERIFIED.json', verified)
        # Flush payloads and directory entries before publishing a complete backup.
        for path in workspace.iterdir():
            with path.open('rb') as stream:
                os.fsync(stream.fileno())
        final = base / 'complete' / identifier
        os.replace(workspace, final)
        for directory in (final, final.parent, base):
            fd = os.open(directory, os.O_RDONLY | os.O_DIRECTORY)
            try:
                os.fsync(fd)
            finally:
                os.close(fd)
        return dict(id=identifier, completed_at=utc(), bytes=total, database=database,
                    integrity=verified, archive_sha256=digest(final / 'files.tar'))
    finally:
        if workspace.exists():
            shutil.rmtree(workspace)


def health(base):
    status_path = base / 'status.json'
    state = json.loads(status_path.read_text()) if status_path.exists() else {}
    success = state.get('last_success', {})
    age = None
    if success:
        age = (dt.datetime.now(dt.timezone.utc) - dt.datetime.fromisoformat(success['completed_at'])).total_seconds()
    fs = os.statvfs(base)
    result = dict(state, newest_complete_age_seconds=age,
                  remaining_bytes=fs.f_bavail * fs.f_frsize, free_inodes=fs.f_favail)
    verified_present = bool(success) and (base / 'complete' / success['id'] / 'VERIFIED.json').is_file()
    result['last_success_verification_present'] = verified_present
    with (base / 'job.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            result['job_lock_held'] = False
        except BlockingIOError:
            result['job_lock_held'] = True
    good = (state.get('outcome') == 'ok' and age is not None and age < 30 * 3600
            and verified_present and fs.f_favail >= 10000)
    return result, 0 if good else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['backup', 'health', 'verify'])
    parser.add_argument('archive', nargs='?', type=Path)
    args = parser.parse_args()
    if os.geteuid() != 0:
        parser.error('root required')
    os.umask(0o077)
    config = json.loads(CONFIG.read_text())
    if socket.gethostname() != config['hostname']:
        raise BackupError('host_identity_mismatch')
    base = Path(config['destination'])
    private_dir(base)
    private_dir(base / 'complete')
    if args.command == 'health':
        value, code = health(base)
        print(json.dumps(value, indent=2))
        return code
    if args.command == 'verify':
        if args.archive is None or args.archive.resolve().parent != (base / 'complete').resolve():
            raise BackupError('choose_complete_backup_in_configured_pool')
        print(json.dumps(verify(args.archive), indent=2))
        return 0
    with (base / 'job.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            print('BACKUP_FAILED execution_lock_busy', file=sys.stderr)
            return 1  # Never overwrite status belonging to a running job.
        state_path = base / 'status.json'
        state = json.loads(state_path.read_text()) if state_path.exists() else {}
        state.update(outcome='running', attempt_at=utc(), pid=os.getpid())
        write_json(state_path, state)
        try:
            # Hard ceiling for any individual file. Preflight also bounds total pool size.
            resource.setrlimit(resource.RLIMIT_FSIZE, (config['max_run_bytes'], config['max_run_bytes']))
            def interrupted(signum, frame):
                raise BackupError('interrupted_' + str(signum))
            signal.signal(signal.SIGTERM, interrupted)
            signal.signal(signal.SIGINT, interrupted)
            signal.signal(signal.SIGXFSZ, interrupted)
            state['last_success'] = capture(config, base)
            state.update(outcome='ok', finished_at=utc())
            write_json(state_path, state)
            print('BACKUP_COMPLETE ' + json.dumps(state['last_success']))
            return 0
        except Exception as error:
            reason = str(error) if isinstance(error, BackupError) else type(error).__name__
            state.update(outcome='failed', finished_at=utc(), last_failure=dict(at=utc(), reason=reason))
            write_json(state_path, state)
            print('BACKUP_FAILED ' + reason, file=sys.stderr)
            return 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except Exception as error:
        # A malformed config or an inaccessible status directory must also fail closed.
        print('BACKUP_FAILED ' + type(error).__name__, file=sys.stderr)
        sys.exit(1)
