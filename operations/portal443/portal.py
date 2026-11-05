#!/usr/bin/env python3
"""Loopback-only form portal. Apache overwrites the source header from its socket."""
from collections import OrderedDict
from datetime import datetime, timezone
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
APP_CSS = Path(__file__).with_name('app.css')
# Reuse the installed asset without redistributing or modifying VICIdial artwork.
LOGO = Path('/var/www/html/vicidial/images/vicidial_admin_web_logo.png')


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
        data = body.encode() if isinstance(body, str) else body
        self.send_response(code)
        for key, value in {'Content-Type': kind, 'Content-Length': str(len(data)),
                           'Cache-Control': 'no-store, private', 'X-Content-Type-Options': 'nosniff',
                           'Referrer-Policy': 'same-origin', 'X-Frame-Options': 'DENY',
                           'Content-Security-Policy': "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'self'; img-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"}.items():
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
        try:
            notice_cookie = SimpleCookie(self.headers.get('Cookie', ''))
            access_notice = '__Secure-AlmaNotice' in notice_cookie and notice_cookie['__Secure-AlmaNotice'].value == 'required'
        except CookieError:
            access_notice = False
        notice = ('<p class="notice" role="status">VICIdial access is missing, expired or revoked. '
                  'Sign in and authorize your current public IP to continue.</p>') if access_notice else ''
        csrf = html.escape(s['csrf'])
        signed_in = 'user' in s
        status = grant_status(s) if signed_in else {'state': 'idle'}
        active = status['state'] == 'active'
        pending = status['state'] == 'pending'
        step = 3 if active else 2 if signed_in else 1

        def form(action, fields, css=''):
            return f'<form class="{css}" method="post" action="/portal/{action}"><input type="hidden" name="csrf" value="{csrf}">{fields}</form>'

        steps = '<ol class="steps" aria-label="Access steps">'
        for number, label in enumerate(('Sign in', 'Authorize IP', 'Continue'), 1):
            current = ' aria-current="step"' if number == step else ''
            steps += f'<li{current}><span>{number}</span>{label}</li>'
        steps += '</ol>'
        content = '<section class="panel" aria-labelledby="panel-title">'
        if 'user' not in s:
            content += '<h2 id="panel-title" class="panel-title">User Login</h2><div class="panel-body"><p class="intro">Sign in with your VICIdial account to authorize access.</p>'
            if message:
                content += f'<p id="login-feedback" class="notice error" role="alert">{html.escape(message)}</p>'
            described = ' aria-describedby="login-feedback"' if message else ''
            content += form('login', '<div class="form-row"><label for="username">Username</label>'
                            f'<input id="username" name="username" required minlength="2" maxlength="64" autocomplete="username" autocapitalize="none" spellcheck="false"{described}></div>'
                            '<div class="form-row"><label for="password">Password</label>'
                            f'<input id="password" type="password" name="password" required maxlength="256" autocomplete="current-password"{described}></div>'
                            '<div class="form-actions"><button class="button primary">Sign in</button></div>', 'login-form')
            content += '<p class="form-note">Use your existing account. Passwords are used only to verify sign-in.</p></div>'
        else:
            role = 'Supervisor' if s['role'] == 'web' else 'Agent'
            destination = DESTINATIONS[s['role']]
            content += '<h2 id="panel-title" class="panel-title">Access Authorization</h2><div class="panel-body">'
            content += f'<dl class="details"><div><dt>VICIdial username</dt><dd>{html.escape(s["user"])}</dd></div><div><dt>Account access type</dt><dd>{role}</dd></div>'
            content += f'<div><dt>{"Authorized public IP" if active else "Public IP"}</dt><dd><code>{self.ip}</code></dd></div></dl>'
            if active:
                content += '<p class="form-note">' + ('Phone access: SIP/RTP enabled for agent or authorized supervisor monitoring.' if status.get('sip_enabled', s['role'] == 'agent') else 'Phone access: web-only; this account has no supervisor monitoring permission.') + '</p>'
            content += f'<section class="authorization {status["state"]}" aria-labelledby="state">'
            if active:
                minutes = status['remaining'] // 60
                expiry = datetime.fromtimestamp(status['expires_at'], timezone.utc)
                content += '<h3 id="state" role="status">Access granted</h3>'
                content += f'<p>Authorization expires in: <strong id="lifetime">{minutes // 60}h {minutes % 60}m</strong></p>'
                content += f'<p class="expiry">Grant expiry: <time datetime="{expiry.isoformat()}">{expiry.strftime("%d %b %Y, %H:%M:%S UTC")}</time></p>'
                content += '<p id="countdown" class="countdown" hidden>Redirecting in: 5 seconds</p><a class="button primary" id="continue" href="/portal/continue">Continue Now</a>'
            elif pending:
                content += '<h3 id="state" role="status">Authorizing your IP...</h3><p>Waiting for access to be confirmed. This page will update automatically.</p><p id="countdown" hidden></p>'
            else:
                content += '<h3 id="state" role="status">' + html.escape(status.get('message', 'Ready to authorize')) + '</h3>'
                content += '<p>Authorize this public IP for up to 12 hours to continue to VICIdial.</p>'
                content += form('grant', '<button class="button primary">Authorize my current IP</button>')
            content += '</section>'
            content += f'<p class="destination">Destination: <code>{destination}</code></p>'
            content += '<div class="session-actions"><a href="/portal/">Refresh authorization status</a>'
            content += form('logout', '<button class="text-button">Revoke my grants on this IP &amp; sign out</button>') + '</div>'
            content += '<p class="form-note">Revocation includes your older browser sessions on this IP. Other users with valid phone grants on the same IP keep access.</p>'
            content += '<noscript><p class="notice">JavaScript is disabled. Refresh this page to confirm authorization. Continue Now appears only after server confirmation.</p></noscript></div>'
            if status['state'] in ('active', 'pending'):
                content += '<script src="/portal/app.js" defer></script>'
        content += '</section>'
        help_title = 'Continue to VICIdial' if signed_in else 'Secure access to your dialer'
        help_body = ('Your account determines the destination. VICIdial will ask you to sign in separately and complete any required two-factor authentication.'
                     if signed_in else 'Sign in, authorize your current public IP, then continue to your VICIdial application.')
        help_body += '</p><p>Access expires automatically. Other users on this public IP share network access; each uses their own VICIdial account.'
        return ('<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">'
                '<title>VICIdial Secure Access</title><link rel="stylesheet" href="/portal/app.css"></head><body>'
                '<a class="skip-link" href="#main">Skip to main content</a>'
                '<header class="masthead"><div class="brand"><img src="/portal/logo.png" width="170" height="45" alt="VICIdial">'
                '<span class="brand-label">Secure Access Portal</span></div><span class="connection">HTTPS connection</span></header>'
                '<div class="title-bar"><div><h1>VICIdial Secure Access</h1><p>Your gateway to the agent and administration interfaces</p></div></div>'
                '<main id="main" class="container">' + notice + steps +
                f'<p class="ip-line">Current public IP: <strong><code>{self.ip}</code></strong></p>'
                '<div class="workspace">' + content + f'<aside class="help-panel"><h2>{help_title}</h2><p>{help_body}</p>'
                '<p class="help-note">Changing networks? Return here to authorize your new public IP.</p></aside></div></main>'
                '<footer>VICIdial Secure Access <span aria-hidden="true">&middot;</span> Application permissions are managed by VICIdial.</footer></body></html>')

    def do_GET(self):
        if not self.checks():
            return
        if self.path == '/':
            self.send(303, location='/portal/')
            return
        if self.path == '/portal/app.js':
            self.send(200, APP_JS.read_text(), 'application/javascript')
            return
        if self.path in ('/portal/app.css', '/portal/logo.png'):
            source, kind = (APP_CSS, 'text/css; charset=utf-8') if self.path.endswith('.css') else (LOGO, 'image/png')
            try:
                self.send(200, source.read_bytes(), kind)
            except OSError:
                self.send(503, 'Portal asset unavailable')
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
        reason = 'form'
        try:
            length = int(self.headers.get('Content-Length', '0'))
            if not 1 <= length <= 4096 or self.headers.get('Content-Type', '').split(';')[0].strip() != 'application/x-www-form-urlencoded':
                raise ValueError('Invalid form')
            fields = parse_qs(self.rfile.read(length).decode(), strict_parsing=True, max_num_fields=4)
            if any(len(v) != 1 for v in fields.values()):
                raise ValueError('Duplicate field')
            fields = {k: v[0] for k, v in fields.items()}
            reason = 'session' if not s else 'csrf'
            if not s or not secrets.compare_digest(fields.get('csrf', ''), s['csrf']):
                raise ValueError('Invalid CSRF')
            reason = 'origin'
            if self.headers.get('Origin', ORIGIN) != ORIGIN:
                raise ValueError('Invalid origin')
        except (ValueError, UnicodeError):
            print('FORM_REJECT reason=' + reason, flush=True)
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
                    try:
                        self.queue(s, 'revoke_user')
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
