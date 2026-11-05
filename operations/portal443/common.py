"""Shared file formats; no credentials or browser tokens are persisted here."""
import ipaddress
import json
import os
from pathlib import Path
import secrets

BASE = Path('/var/lib/alma-portal-443')
QUEUE = Path('/var/lib/alma-portal/requests-443')
LIFETIME = 43200
HEARTBEAT = 35
DESTINATIONS = {'agent': '/agc/vicidial.php', 'web': '/vicidial/admin.php'}


def public_ip(value):
    ip = ipaddress.ip_address(value)
    if ip.version != 4 or not ip.is_global:
        raise ValueError('Public IPv4 required')
    return str(ip)


def atomic_json(path, value, mode=0o640, gid=None):
    path = Path(path)
    tmp = path.with_name('.' + secrets.token_hex(16))
    try:
        fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, mode)
        with os.fdopen(fd, 'w') as stream:
            if gid is not None:
                os.fchown(stream.fileno(), 0, gid)
            json.dump(value, stream, separators=(',', ':'))
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(tmp, path)
        fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(fd)
        finally:
            os.close(fd)
    finally:
        tmp.unlink(missing_ok=True)
