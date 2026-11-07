"""Individual Basic authentication and bounded, receipt-verified WAV delivery.

Only the existing loopback Handler calls this module, after trusted-source checks.
Apache must retain its network gate. No portal cookie or auth session is used.
"""
import base64
from collections import OrderedDict
import json
import os
from pathlib import Path
import re
import stat
import threading
import time

import vicidial_auth as auth
from common import BASE

ROOT = Path('/var/spool/asterisk/monitorDONE')
ROOT_UID = 0
RECEIPTS = Path('/var/lib/alma-recording-playback')
NAME = r'[A-Za-z0-9][A-Za-z0-9_-]{0,199}-all\.wav'
ROUTE = re.compile(r'^/(RECORDINGS|vicidial/recordings)/(' + NAME + r')$')
ORIGIN = 'https://2.29.153.161'
SLOTS = threading.BoundedSemaphore(4)
LOCK = threading.Lock()
FAILURES = OrderedDict()


class Denied(Exception):
    def __init__(self, code):
        self.code = code


def network_active(ip):
    try:
        return int((BASE / 'web' / ip).read_text()) > time.time()
    except (OSError, ValueError):
        return False


def credentials(headers):
    values = headers.get_all('Authorization', [])
    if len(values) != 1 or len(values[0]) > 1024:
        raise Denied(401)
    try:
        scheme, encoded = values[0].split(' ', 1)
        if scheme.lower() != 'basic':
            raise ValueError()
        user, password = base64.b64decode(encoded, validate=True).decode('utf-8').split(':', 1)
        if not auth.USER_PATTERN.fullmatch(user) or not 1 <= len(password) <= 256:
            raise ValueError()
        # Native user_authorization strips these characters. Reject, never
        # normalize credentials into another password or accept a wider policy.
        if any(c in password for c in "|`&'\"\\; \r\n\x00"):
            raise ValueError()
        return user, password
    except (ValueError, UnicodeError):
        raise Denied(401) from None


def authorize_recording(user, password, name):
    """Fresh, read-only native eligibility/permission/metadata checks each time."""
    conn = auth._connect()
    try:
        with conn.cursor() as cur:
            cur.execute('SELECT pass_hash_enabled,pass_key,pass_cost,two_factor_auth_hours,'
                        'allow_ip_lists,system_ip_blacklist,log_recording_access FROM system_settings LIMIT 1')
            settings = cur.fetchone()
            if (not settings or int(settings['two_factor_auth_hours']) != 0
                    or int(settings['allow_ip_lists']) != 0
                    or settings['system_ip_blacklist'].strip()
                    or int(settings['log_recording_access']) != 0):
                # A password alone cannot demonstrate browser-bound native 2FA.
                # Never silently bypass an unsupported native authentication or
                # redirect/logging policy. Native source remains untouched.
                raise Denied(503)
            cur.execute('SELECT user,active,user_level,api_only_user,force_change_password,'
                        'pass,pass_hash,access_recordings,view_reports,user_group,failed_login_count,'
                        'UNIX_TIMESTAMP(last_login_date) AS login_epoch,UNIX_TIMESTAMP() AS db_now '
                        'FROM vicidial_users WHERE user=%s LIMIT 1', (user,))
            row = cur.fetchone()
            if not row or not auth.verify_password(row, settings, password):
                raise Denied(401)
            if (auth._role_from_row(row) != 'web' or int(row['access_recordings']) < 1
                    or int(row['view_reports']) != 1
                    or (int(row['failed_login_count']) >= 10
                        and (row['login_epoch'] is None or row['login_epoch'] >= row['db_now'] - 900))):
                raise Denied(403)
            cur.execute('SELECT allowed_campaigns,allowed_reports,admin_viewable_groups '
                        'FROM vicidial_user_groups WHERE user_group=%s LIMIT 1', (row['user_group'],))
            group = cur.fetchone()
            if (not group or '-ALL-CAMPAIGNS-' not in group['allowed_campaigns'].split()
                    or group['allowed_reports'].strip() != 'ALL REPORTS'
                    or group['admin_viewable_groups'].strip() != '---ALL---'):
                raise Denied(403)
            cur.execute('SELECT recording_id,filename,end_epoch,length_in_sec,location '
                        'FROM recording_log WHERE filename=%s LIMIT 2', (name[:-8],))
            rows = cur.fetchall()
            if len(rows) != 1:
                raise Denied(404)
            record = rows[0]
            if (int(record['end_epoch'] or 0) <= 0 or int(record['end_epoch']) > time.time()
                    or int(record['length_in_sec'] or 0) <= 0
                    or record['location'] != ORIGIN + '/RECORDINGS/' + name):
                raise Denied(404)
            return record
    finally:
        conn.close()


def fingerprint(s):
    return [s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns, s.st_ctime_ns]


def open_recording(record, name):
    """Read only a root-enrolled, native-associated inode, never a supplied path."""
    try:
        receipt = json.loads((RECEIPTS / (str(int(record['recording_id'])) + '.json')).read_text())
        if receipt['name'] != name or receipt['recording_id'] != int(record['recording_id']):
            raise Denied(404)
        root_fd = os.open(ROOT, os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=root_fd)
        finally:
            os.close(root_fd)
        with os.fdopen(fd, 'rb') as stream:
            s = os.fstat(stream.fileno())
            if (not stat.S_ISREG(s.st_mode) or s.st_nlink != 1 or s.st_uid != ROOT_UID
                    or stat.S_IMODE(s.st_mode) != 0o640
                    or fingerprint(s) != receipt['fingerprint']):
                raise Denied(404)
            # Return ownership of the verified open inode, not a reopened path.
            return os.fdopen(os.dup(stream.fileno()), 'rb'), s.st_size
    except (OSError, ValueError, KeyError, TypeError):
        raise Denied(404) from None


def byte_range(value, size):
    if value is None:
        return 0, size - 1, 200
    match = re.fullmatch(r'bytes=(\d*)-(\d*)', value)
    if not match or not any(match.groups()):
        raise Denied(416)
    a, b = match.groups()
    if not a:
        if int(b) <= 0:
            raise Denied(416)
        start, end = max(0, size - int(b)), size - 1
    else:
        start, end = int(a), min(size - 1, int(b)) if b else size - 1
    if start >= size or start > end:
        raise Denied(416)
    return start, end, 206


def reply(h, code, **headers):
    h._media_started = True
    h.send_response(code)
    for key, value in {'Cache-Control': 'no-store, private', 'Pragma': 'no-cache',
                       'X-Content-Type-Options': 'nosniff', 'Referrer-Policy': 'no-referrer',
                       'Content-Security-Policy': "default-src 'none'; media-src 'self'; frame-ancestors 'self'", **headers}.items():
        h.send_header(key, str(value))
    h.end_headers()


def serve(h):
    """Return True only for the recording namespace (including malformed paths)."""
    if not h.path.startswith(('/RECORDINGS/', '/vicidial/recordings/')):
        return False
    h._media_started = False
    user, rid, code, started = '', 0, 503, time.monotonic()
    acquired = False
    size = None
    try:
        if os.environ.get('ALMA_RECORDING_PLAYBACK') != '1':
            raise Denied(503)
        if not network_active(h.ip):
            raise Denied(403)
        match = ROUTE.fullmatch(h.path)
        if not match:
            raise Denied(404)
        if h.command not in ('GET', 'HEAD'):
            raise Denied(405)
        if h.headers.get('Sec-Fetch-Site') == 'cross-site' or h.headers.get('Origin', ORIGIN) != ORIGIN:
            raise Denied(403)
        namespace, name = match.groups()
        if namespace == 'RECORDINGS':
            code = 307
            reply(h, code, Location='/vicidial/recordings/' + name, **{'Content-Length': '0'})
            return True
        user, password = credentials(h.headers)
        key = (h.ip, user)
        with LOCK:
            failures = [t for t in FAILURES.get(key, []) if started - t < 900]
            FAILURES[key] = failures
            FAILURES.move_to_end(key)
            while len(FAILURES) > 10000:
                FAILURES.popitem(last=False)
            # Per-IP bound also covers attempts rotating usernames.
            ipkey = (h.ip, '')
            ip_failures = [t for t in FAILURES.get(ipkey, []) if started - t < 900]
            FAILURES[ipkey] = ip_failures
            if len(failures) >= 7 or len(ip_failures) >= 30:
                raise Denied(429)
            failures.append(started)
            ip_failures.append(started)
        acquired = SLOTS.acquire(blocking=False)
        if not acquired:
            raise Denied(503)
        record = authorize_recording(user, password, name)
        with LOCK:
            for bucket in (key, ipkey):
                if started in FAILURES.get(bucket, []):
                    FAILURES[bucket].remove(started)
        rid = int(record['recording_id'])
        stream, size = open_recording(record, name)
        with stream:
            if len(h.headers.get_all('Range', [])) > 1:
                raise Denied(416)
            start, end, code = byte_range(h.headers.get('Range') if h.command == 'GET' else None, size)
            headers = {'Content-Type': 'audio/wav', 'Accept-Ranges': 'bytes',
                       'Content-Length': str(end - start + 1),
                       'Content-Disposition': 'inline; filename="recording-' + str(rid) + '.wav"'}
            if code == 206:
                headers['Content-Range'] = f'bytes {start}-{end}/{size}'
            reply(h, code, **headers)
            if h.command == 'GET':
                stream.seek(start)
                remaining = end - start + 1
                while remaining and time.monotonic() - started < 120 and network_active(h.ip):
                    block = stream.read(min(65536, remaining))
                    if not block:
                        break
                    h.wfile.write(block)
                    remaining -= len(block)
                if remaining:
                    h.close_connection = True
                    code = 499
    except Denied as error:
        code = error.code
        headers = {'Content-Length': '0'}
        if code == 401:
            headers['WWW-Authenticate'] = 'Basic realm="CONTACT-CENTER-ADMIN"'
            if user:
                print('AUTH_FAIL ip=' + h.ip, flush=True)
        if code == 405:
            headers['Allow'] = 'GET, HEAD'
        if code == 416 and size is not None:
            headers['Content-Range'] = 'bytes */' + str(size)
        reply(h, code, **headers)
    except (BrokenPipeError, ConnectionError, TimeoutError):
        h.close_connection = True
        code = 499
    except Exception:
        code = 503
        if h._media_started:
            h.close_connection = True
        else:
            reply(h, code, **{'Content-Length': '0'})
    finally:
        if acquired:
            SLOTS.release()
        print(json.dumps(dict(event='recording_access', user=user, recording_id=rid,
                              ip=h.ip, status=code)), flush=True)
    return True
