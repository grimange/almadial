#!/usr/bin/env python3
"""Invalidate the service verifier exposed by a grants diagnostic.

Run only on the authorized host through SSH. No secret-bearing argv or output.
Keeps a private local backup, preserves file metadata, rolls back on failure.
Does not change any VICIdial application user or DB privilege.
"""
import configparser
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys

sys.path.insert(0, '/opt/alma-portal-443')
from vicidial_auth import _connect

CONFIG = Path('/etc/alma-portal/vicidial-db.ini')
BACKUP = Path('/root/alma-portal-revoke-backup/db-service-config.before.ini')
old = CONFIG.read_bytes()
meta = CONFIG.stat()
parser = configparser.ConfigParser(interpolation=None)
parser.read_string(old.decode())
assert parser['db']['user'] == 'alma_portal_auth'
old_secret = parser['db']['password']
new_secret = secrets.token_hex(32)
assert len(re.findall(rb'(?m)^password\s*=.*$', old)) == 1
new = re.sub(rb'(?m)^password\s*=.*$', b'password = ' + new_secret.encode(), old)


def sql(query):
    result = subprocess.run(['mysql', '--batch', '--skip-column-names'], input=query,
                            text=True, capture_output=True, timeout=10)
    if result.returncode:
        raise RuntimeError('Private database operation failed')
    return result.stdout.strip()


def alter(secret):
    # Values never appear in argv, stdout or exceptions. Escape the previous
    # secret as SQL text in case recovery is needed; new value is hex only.
    escaped = secret.replace('\\', '\\\\').replace("'", "''")
    sql("ALTER USER 'alma_portal_auth'@'localhost' IDENTIFIED BY '" + escaped + "';")


def write(value):
    tmp = CONFIG.with_name('.vicidial-db.rotation')
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, meta.st_mode & 0o777)
    with os.fdopen(fd, 'wb') as stream:
        os.fchown(stream.fileno(), meta.st_uid, meta.st_gid)
        stream.write(value)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(tmp, CONFIG)


assert sql('SELECT @@global.general_log') == '0', 'Stop: SQL general logging is enabled'
fd = os.open(BACKUP, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(fd, 'wb') as stream:
    stream.write(old)
subprocess.run(['systemctl', 'stop', 'alma-portal-443-apply.timer', 'alma-portal-443-apply.service'], check=True)
changed = False
try:
    alter(new_secret)
    changed = True
    write(new)
    subprocess.run(['restorecon', str(CONFIG)], capture_output=True, check=True)
    c = _connect()
    with c.cursor() as cursor:
        cursor.execute('SELECT 1')
        assert next(iter(cursor.fetchone().values())) == 1
    c.close()
except Exception:
    if changed:
        alter(old_secret)
        if CONFIG.read_bytes() != old:
            write(old)
    print('SERVICE_DB_ROTATION_FAILED_RESTORED', flush=True)
    raise SystemExit(1)
finally:
    subprocess.run(['systemctl', 'start', 'alma-portal-443-apply.service', 'alma-portal-443-apply.timer'], check=True)
print('SERVICE_DB_SECRET_ROTATED_READONLY_CONNECT_PASS', flush=True)
