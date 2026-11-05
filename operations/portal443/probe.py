#!/usr/bin/env python3
"""On-host HTTPS integration checks. No credentials, DB changes or SIP rules.

Temporary web-only fixture for the server's own public IP; never the desktop IP.
The probe does not establish independent internet/client or real-user acceptance.
"""
import http.client
import os
from pathlib import Path
import socket
import ssl
import time

IP = '2.29.153.161'
CONTEXT = ssl.create_default_context()


def request(path, port=443, headers=None, method='GET'):
    cls = http.client.HTTPSConnection if port == 443 else http.client.HTTPConnection
    kwargs = {'timeout': 8}
    if port == 443:
        kwargs['context'] = CONTEXT
    connection = cls(IP, port, **kwargs)
    # Force the trusted source to the server's real public address, not loopback.
    connection.source_address = (IP, 0)
    connection.request(method, path, headers=headers or {})
    response = connection.getresponse()
    body = response.read()
    result = response.status, dict(response.getheaders()), body
    connection.close()
    return result


def main():
    paths = ['/agc/', '/agc/vicidial.php', '/vicidial/', '/vicidial/admin.php',
             '/agc/vdc_db_query.php', '/vicidial/non_agent_api.php', '/recordings/',
             '/RECORDINGS/', '/robots.txt', '/icons/', '/cgi-bin/', '/.noindex.html',
             '/server-status', '/%61gc/vicidial.php', '//agc/vicidial.php',
             '/portal/../agc/vicidial.php', '/portal/%2e%2e/agc/vicidial.php',
             '/agc%2fvicidial.php', '/portal/app.js/agc/vicidial.php',
             '/portal/continue/../agc/vicidial.php', '/agc/vicidial.php/portal/',
             '/%252e%252e/agc/vicidial.php']
    grant = Path('/var/lib/alma-portal-443/web') / IP
    # Refuse to disturb any real grant for this IP.
    if grant.exists():
        raise SystemExit('Refusing to overwrite an existing grant fixture')
    assert request('/')[0] == 303, 'root portal route'
    status, headers, body = request('/portal/')
    assert status == 200 and b'VICIdial Secure Access' in body
    assert 'WWW-Authenticate' not in headers
    assert request('/portal/status')[0] == 401
    assert request('/portal/continue')[0] == 401
    print('PASS TLS trust/IP SAN; root portal; private status and continue')
    for path in paths:
        for spoof in ({}, {'X-Forwarded-For': '158.62.74.253', 'X-Real-IP': '158.62.74.253', 'X-Alma-Source': '158.62.74.253'}):
            code = request(path, headers=spoof)[0]
            assert code in (400, 403, 404), (path, code)
    for host in ('localhost', 'untrusted.invalid', IP + ':9091'):
        assert request('/agc/vicidial.php', headers={'Host': host})[0] == 403
    print('PASS default-deny paths, encoded paths, aliases, PHP, host and header spoofing')
    for path in ('/', '/agc/vicidial.php', '/vicidial/admin.php', '/icons/', '/.noindex.html', '/cgi-bin/', '/server-status'):
        assert request(path, 80)[0] == 403, ('HTTP', path)
    challenge = Path('/var/www/alma-acme/.well-known/acme-challenge/alma-portal443-probe')
    assert not challenge.exists()
    try:
        challenge.write_text('acme-route-only')
        os.chmod(challenge, 0o644)
        code, _, body = request('/.well-known/acme-challenge/' + challenge.name, 80)
        assert code == 200 and body == b'acme-route-only'
    finally:
        challenge.unlink(missing_ok=True)
    print('PASS HTTP deny; real ACME webroot challenge file')
    try:
        grant.write_text(str(int(time.time()) + 20))
        os.chmod(grant, 0o644)
        # Native login pages are evidence of boundary crossing, not logged-in users.
        for path in ('/agc/vicidial.php', '/vicidial/admin.php', '/robots.txt'):
            code = request(path)[0]
            assert code in (200, 401), ('active web fixture', path, code)
        print('PASS temporary server-IP web fixture reaches native login pages')
        grant.write_text(str(int(time.time()) - 1))
        assert request('/agc/vicidial.php')[0] == 403
        print('PASS expired web grant denied')
    finally:
        grant.unlink(missing_ok=True)
    assert request('/vicidial/admin.php')[0] == 403
    print('PASS revoked web grant denied; fixture removed')


if __name__ == '__main__':
    main()
