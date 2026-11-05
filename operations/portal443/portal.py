#!/usr/bin/env python3
"""Loopback-only form portal. Apache overwrites the source header from its socket."""
from collections import OrderedDict
from http.cookies import SimpleCookie, CookieError
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import html
import json
import os
from pathlib import Path
import secrets
import threading
import time
from urllib.parse import parse_qs

from common import BASE, QUEUE, DESTINATIONS, LIFETIME, atomic_json, public_ip
from vicidial_auth import authenticate

PUBLIC_IP = os.environ.get('PORTAL_PUBLIC_IP', '2.29.153.161')
ORIGIN = 'https://' + PUBLIC_IP
COOKIE = '__Secure-AlmaPortal'
SESSIONS = OrderedDict()
FAILURES = OrderedDict()
LOCK = threading.RLock()
AUTH_SLOTS = threading.BoundedSemaphore(4)
APP_JS = Path(__file__).with_name('app.js')


def grant_status(session, now=None):
    now = int(time.time()) if now is None else now
    if 'request' not in session:
        return {'state': 'idle'}
    try:
        status = json.loads((BASE / 'status' / session['request']).read_text())
        if status.get('ip') != session['ip'] or status.get('user') != session['user']:
            raise ValueError('Identity mismatch')
        if status['state'] == 'active':
            try:
                web_expiry = int((BASE / 'web' / session['ip']).read_text().strip())
            except FileNotFoundError:
                web_expiry = 0
            if now >= min(status['expires_at'], status['checked_until'], web_expiry):
                return {'state': 'expired', 'message': 'Access expired or the worker is unavailable. Authorize again.'}
            return {**status, 'server_now': now, 'remaining': status['expires_at'] - now,
                    'destination': DESTINATIONS[status['role']]}
        return {'state': status['state'], 'message': 'Authorization failed or was revoked. Please retry.'}
    except FileNotFoundError:
        if now - session['queued_at'] <= 120:
            return {'state': 'pending', 'message': 'Authorizing your IP...'}
        return {'state': 'failed', 'message': 'The authorization worker did not confirm access. Please retry.'}
    except (OSError, ValueError, KeyError, TypeError):
        return {'state': 'failed', 'message': 'Authorization could not be verified. Please retry.'}


class Handler(BaseHTTPRequestHandler):
    server_version = 'AlmaAccess/3'
    sys_version = ''

    def log_message(self, fmt, *args):
        # Never log URLs, query strings, headers, form bodies or credentials.
        pass

    def send(self, code, body='', kind='text/html; charset=utf-8', cookie=None, location=None):
        data = body.encode()
        self.send_response(code)
        for key, value in {'Content-Type': kind, 'Content-Length': str(len(data)),
                           'Cache-Control': 'no-store, private', 'X-Content-Type-Options': 'nosniff',
                           'Referrer-Policy': 'no-referrer', 'X-Frame-Options': 'DENY',
                           'Content-Security-Policy': "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"}.items():
            self.send_header(key, value)
        if cookie:
            self.send_header('Set-Cookie', f'{COOKIE}={cookie}; Path=/portal/; HttpOnly; Secure; SameSite=Strict; Max-Age={LIFETIME}')
        if location:
            self.send_header('Location', location)
        self.end_headers()
        self.wfile.write(data)

    def checks(self):
        try:
            if self.client_address[0] != '127.0.0.1':
                raise ValueError('Proxy required')
            if self.headers.get('Host') not in (PUBLIC_IP, PUBLIC_IP + ':443'):
                raise ValueError('Invalid host')
            self.ip = public_ip(self.headers['X-Alma-Source'])
            if '?' in self.path or self.headers.get('Transfer-Encoding'):
                raise ValueError('Invalid request')
            return True
        except (ValueError, TypeError):
            self.send(400, 'Request rejected')
            return False

    def session(self, create=False):
        now = time.time()
        try:
            jar = SimpleCookie(self.headers.get('Cookie', ''))
            token = jar[COOKIE].value if COOKIE in jar else ''
        except CookieError:
            token = ''
        with LOCK:
            for key in list(SESSIONS):
                if SESSIONS[key]['until'] <= now:
                    del SESSIONS[key]
            s = SESSIONS.get(token)
            if s and s['ip'] == self.ip:
                return s, None
            if create:
                if len(SESSIONS) >= 10000:
                    return None, None
                token = secrets.token_hex(32)
                s = {'ip': self.ip, 'csrf': secrets.token_hex(32), 'until': now + 900}
                SESSIONS[token] = s
                return s, token
        return None, None

    def page(self, s, message=''):
        csrf = html.escape(s['csrf'])
        def form(action, fields):
            return f'<form method="post" action="/portal/{action}"><input type="hidden" name="csrf" value="{csrf}">{fields}</form>'
        content = f'<p>Current public IP: <strong>{self.ip}</strong></p><p>{html.escape(message)}</p>'
        if 'user' not in s:
            content += form('login', '<p><label>VICIdial username <input name="username" required maxlength="64" autocomplete="username"></label></p><p><label>Password <input type="password" name="password" required maxlength="256" autocomplete="current-password"></label></p><button>Sign in</button>')
        else:
            status = grant_status(s)
            role = 'Supervisor' if s['role'] == 'web' else 'Agent'
            content += f'<p>Signed in as {html.escape(s["user"])}. Access type: {role}</p>'
            if status['state'] == 'active':
                minutes = status['remaining'] // 60
                content += f'<h2 id="state">Access granted</h2><p>Authorized IP: {self.ip}</p><p>Authorization expires in: <span id="lifetime">{minutes // 60}h {minutes % 60}m</span></p><p id="countdown">Redirecting in: 5 seconds</p><p><a id="continue" href="/portal/continue">Continue Now</a></p>'
            elif status['state'] == 'pending':
                content += '<h2 id="state">Authorizing your IP...</h2><p id="countdown"></p>'
            else:
                content += '<h2 id="state">' + html.escape(status.get('message', 'Authorize access')) + '</h2>'
                content += form('grant', '<button>Authorize my current IP for 12 hours</button>')
            content += '<p><a href="/portal/">Refresh authorization status</a></p><noscript><p>Refresh this page to confirm authorization. Continue Now appears only after server confirmation.</p></noscript>'
            content += form('logout', '<button>Revoke this session’s authorization and sign out</button>')
            if status['state'] in ('active', 'pending'):
                content += '<script src="/portal/app.js" defer></script>'
        return '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>VICIdial Secure Access</title></head><body style="font:17px system-ui;max-width:700px;margin:4rem auto;padding:1rem"><h1>VICIdial Secure Access</h1>' + content + '<p>Network access only. Sign in to VICIdial separately; its existing permissions still apply. Shared public IPs share network reachability.</p></body></html>'

    def do_GET(self):
        if not self.checks():
            return
        if self.path == '/':
            self.send(303, location='/portal/')
            return
        if self.path == '/portal/app.js':
            self.send(200, APP_JS.read_text(), 'application/javascript')
            return
        s, cookie = self.session(create=self.path == '/portal/')
        if s is None:
            self.send(401, 'Session required. Open /portal/.')
            return
        if self.path == '/portal/':
            self.send(200, self.page(s), cookie=cookie)
        elif 'user' not in s:
            self.send(401, 'Sign in required')
        elif self.path == '/portal/status':
            self.send(200, json.dumps(grant_status(s)), 'application/json')
        elif self.path == '/portal/continue':
            status = grant_status(s)
            if status['state'] != 'active':
                self.send(403, 'Access is not active. Return to /portal/.')
            else:
                self.send(303, location=DESTINATIONS[status['role']])
        else:
            self.send(404, 'Not found')

    def do_POST(self):
        if not self.checks():
            return
        s, _ = self.session()
        try:
            length = int(self.headers.get('Content-Length', '0'))
            if not 1 <= length <= 4096 or self.headers.get('Content-Type') != 'application/x-www-form-urlencoded':
                raise ValueError('Invalid form')
            fields = parse_qs(self.rfile.read(length).decode(), strict_parsing=True, max_num_fields=4)
            if any(len(v) != 1 for v in fields.values()):
                raise ValueError('Duplicate field')
            fields = {k: v[0] for k, v in fields.items()}
            if not s or not secrets.compare_digest(fields.get('csrf', ''), s['csrf']):
                raise ValueError('Invalid CSRF')
            if self.headers.get('Origin', ORIGIN) != ORIGIN:
                raise ValueError('Invalid origin')
        except (ValueError, UnicodeError):
            self.send(403, 'Form or CSRF validation failed')
            return
        if self.path == '/portal/login':
            now = time.monotonic()
            with LOCK:
                failures = [t for t in FAILURES.get(self.ip, []) if now - t < 900]
                FAILURES[self.ip] = failures
                if len(FAILURES) > 20000:
                    FAILURES.popitem(last=False)
                if len(failures) >= 7:
                    self.send(429, 'Too many attempts; try later')
                    return
                # Charge every attempt before the slow DB/hash operation, preventing concurrent bursts.
                failures.append(now)
            if not AUTH_SLOTS.acquire(blocking=False):
                self.send(503, 'Busy; try later')
                return
            try:
                role = authenticate(fields.get('username', ''), fields.get('password', ''))
            except Exception:
                self.send(503, 'Authentication service unavailable')
                return
            finally:
                AUTH_SLOTS.release()
            if role not in DESTINATIONS:
                print('AUTH_FAIL ip=' + self.ip, flush=True)
                self.send(401, self.page(s, 'Credentials or account eligibility rejected'))
                return
            with LOCK:
                if now in FAILURES.get(self.ip, []):
                    FAILURES[self.ip].remove(now)
                # Rotate the browser session after login. Never keep a password.
                for key in list(SESSIONS):
                    if SESSIONS[key] is s:
                        del SESSIONS[key]
                token = secrets.token_hex(32)
                s = {'ip': self.ip, 'user': fields['username'], 'role': role,
                     'csrf': secrets.token_hex(32), 'until': time.time() + LIFETIME}
                SESSIONS[token] = s
            self.send(303, cookie=token, location='/portal/')
        elif self.path in ('/portal/grant', '/portal/logout') and 'user' in s:
            with LOCK:
                if self.path == '/portal/logout':
                    if 'request' in s:
                        try:
                            self.queue(s, 'revoke')
                        except OSError:
                            self.send(503, 'Revocation could not be queued. Please retry.')
                            return
                    for key in list(SESSIONS):
                        if SESSIONS[key] is s:
                            del SESSIONS[key]
                    self.send(303, location='/portal/')
                    return
                if grant_status(s)['state'] not in ('active', 'pending'):
                    try:
                        self.queue(s, 'grant')
                    except OSError:
                        self.send(503, 'Authorization could not be queued. Please retry.')
                        return
            self.send(303, location='/portal/')
        else:
            self.send(403, 'Sign in required or invalid action')

    def queue(self, s, kind):
        request_id = secrets.token_hex(16)
        data = {'ip': s['ip'], 'user': s['user'], 'role': s['role'], 'at': int(time.time()), 'kind': kind}
        if kind == 'revoke':
            data['target'] = s['request']
        atomic_json(QUEUE / (request_id + '.request'), data, 0o600)
        if kind == 'grant':
            s['request'] = request_id
            s['queued_at'] = data['at']


class Server(ThreadingHTTPServer):
    daemon_threads = True
    request_queue_size = 32

    def get_request(self):
        sock, address = super().get_request()
        sock.settimeout(8)
        return sock, address


if __name__ == '__main__':
    os.umask(0o027)
    Server(('127.0.0.1', 19091), Handler).serve_forever()
