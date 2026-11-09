"""Fixed S03 paths only. No shell, replacement, secret output or SQL execution."""
import hashlib
import json
import os
from pathlib import Path
import pwd
import secrets
import stat
import subprocess
import sys

CONFIG_PATH = '/etc/my.cnf.d/zz-almadial.cnf'
CONFIG = '[mariadb]\nbind-address=127.0.0.1\nport=3306\nskip-name-resolve\nsocket=/var/lib/mysql/mysql.sock\ncharacter-set-server=utf8mb3\ncollation-server=utf8mb3_unicode_ci\nsql-mode=STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION\n'
SECRETS = ('/etc/almadial/secrets/database', '/etc/almadial/secrets/admin')
CLIENT = '/etc/almadial/secrets/database-client.cnf'

def client(value):
    return b'[client]\nuser=almadial\npassword=' + value.strip() + b'\n'

def safe_parent(path):
    for p in reversed(Path(path).parents):
        s = p.lstat()
        if not stat.S_ISDIR(s.st_mode) or s.st_uid != 0 or s.st_mode & 0o022:
            raise ValueError('unsafe parent')

def read(path, mode, gid=0):
    safe_parent(path)
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    except FileNotFoundError:
        return None
    with os.fdopen(fd, 'rb') as f:
        s = os.fstat(f.fileno())
        if not stat.S_ISREG(s.st_mode) or s.st_nlink != 1 or s.st_uid != 0 or s.st_gid != gid or stat.S_IMODE(s.st_mode) != mode or s.st_size > 4096:
            raise ValueError('unsafe file')
        return f.read(4097)

def create(path, value, mode, gid=0):
    safe_parent(path)
    # O_EXCL preserves pre-existing operator files. A crash leaves either the
    # complete verified file or a conflicting partial file requiring review.
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, mode)
    with os.fdopen(fd, 'wb') as f:
        os.fchown(f.fileno(), 0, gid)
        os.fchmod(f.fileno(), mode)
        f.write(value)
        f.flush()
        os.fsync(f.fileno())
    d = os.open(str(Path(path).parent), os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(d)
    finally:
        os.close(d)

def inspect():
    mysql = pwd.getpwnam('mysql')
    d = Path('/var/lib/mysql').lstat()
    if not stat.S_ISDIR(d.st_mode) or d.st_uid != mysql.pw_uid or d.st_gid != mysql.pw_gid or d.st_mode & 0o022:
        raise ValueError('unsafe data directory')
    v = read(CONFIG_PATH, 0o640, mysql.pw_gid)
    cfg = 'absent' if v is None else ('valid' if v == CONFIG.encode() else 'conflict')
    values = [read(p, 0o600) for p in SECRETS]
    valid = all(v is not None and len(v) == 49 and v[-1:] == b'\n' and all(c in b'0123456789abcdef' for c in v[:-1]) for v in values)
    option = read(CLIENT, 0o600)
    valid = valid and option == client(values[0])
    sec = 'valid' if valid else ('absent' if values == [None, None] and option is None else 'conflict')
    digest = hashlib.sha256(b''.join(values)).hexdigest() if valid else ''
    inputs = []
    for p in [Path('/etc/my.cnf')] + sorted(Path('/etc/my.cnf.d').iterdir()):
        if str(p) == CONFIG_PATH:
            continue
        safe_parent(str(p))
        s = p.lstat()
        if not stat.S_ISREG(s.st_mode) or s.st_uid != 0 or s.st_mode & 0o022:
            raise ValueError('unsafe vendor configuration')
        inputs.append((str(p), hashlib.sha256(p.read_bytes()).hexdigest()))
    unit = subprocess.run(['/usr/bin/systemctl', 'cat', 'mariadb'], capture_output=True, check=True).stdout
    inputs.append(('systemd-unit', hashlib.sha256(unit).hexdigest()))
    # Named properties avoid reliance on systemd's property ordering.
    props = dict(line.split('=',1) for line in subprocess.run(['/usr/bin/systemctl', 'show', 'mariadb', '--property=ActiveState,UnitFileState,InvocationID'], capture_output=True, check=True).stdout.decode().splitlines())
    if props['ActiveState'] == 'active':
        socket = Path('/var/lib/mysql/mysql.sock').lstat()
        if not stat.S_ISSOCK(socket.st_mode) or socket.st_uid != mysql.pw_uid or socket.st_gid != mysql.pw_gid or stat.S_IMODE(socket.st_mode) != 0o777:
            raise ValueError('unexpected vendor socket metadata')
    print(json.dumps(dict(Config=cfg, Secrets=sec, CredentialID=digest, ConfigInputs=hashlib.sha256(json.dumps(inputs,separators=(',',':')).encode()).hexdigest(), Initialized=any(Path('/var/lib/mysql').iterdir()), Active=props['ActiveState']=='active', Enabled=props['UnitFileState']=='enabled', ServiceID=props['InvocationID'])))

try:
    if len(sys.argv) != 2:
        raise ValueError('operation')
    if sys.argv[1] == 'inspect':
        inspect()
    elif sys.argv[1] == 'config':
        create(CONFIG_PATH, CONFIG.encode(), 0o640, pwd.getpwnam('mysql').pw_gid)
    elif sys.argv[1] == 'credentials':
        if any(read(p, 0o600) is not None for p in (*SECRETS, CLIENT)):
            raise ValueError('existing credentials')
        for p in SECRETS:
            create(p, (secrets.token_hex(24)+'\n').encode(), 0o600)
        create(CLIENT, client(read(SECRETS[0], 0o600)), 0o600)
    else:
        raise ValueError('operation')
except Exception:
    # Python exception text may contain paths or data. It is not diagnostic output.
    sys.stderr.write('S03 fixed file operation failed; inspect protected state\n')
    sys.exit(1)
