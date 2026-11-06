#!/usr/bin/python3
"""Bounded read-only TLS/regression evidence collector; execute over SSH stdin."""
import hashlib
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import time
import urllib.error
import urllib.request


def run(argv, timeout=20):
    try:
        p = subprocess.run(argv, capture_output=True, text=True, timeout=timeout)
        return dict(rc=p.returncode, stdout=p.stdout[-18000:], stderr=p.stderr[-2000:])
    except Exception as e:
        return dict(error=type(e).__name__)


def file_info(name, content=False):
    p = Path(name)
    if not p.exists():
        return dict(exists=False)
    s = p.stat()
    result = dict(mode=oct(s.st_mode & 0o777), uid=s.st_uid, gid=s.st_gid,
                  size=s.st_size, sha256=hashlib.sha256(p.read_bytes()).hexdigest())
    if content:
        result['content'] = p.read_text()
    return result


ip = '2.29.153.161'
lineage = '/etc/letsencrypt/live/' + ip
result = dict(time_utc=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
              hostname=socket.gethostname())
result['files'] = {name: file_info(name, content) for name, content in [
    ('/usr/local/sbin/alma-portal-cert-deploy', True),
    ('/usr/local/libexec/alma-portal-tls', False),
    ('/etc/systemd/system/alma-portal-renew.service', True),
    ('/etc/systemd/system/alma-portal-renew.timer', True),
    ('/etc/letsencrypt/renewal/' + ip + '.conf', False),
    ('/etc/httpd/conf.d/00-alma-acme.conf', True),
    ('/etc/httpd/conf.d/02-alma-vicidial-ssl.conf', True),
    ('/opt/alma-portal-443/access.lua', False),
    ('/opt/alma-portal-443/portal.py', False),
    ('/opt/alma-portal-443/common.py', False),
    ('/opt/alma-portal-443/apply_requests.py', False),
    ('/opt/alma-portal-443/sip_guard.py', False),
]}
result['certificate'] = run(['openssl', 'x509', '-in', lineage + '/cert.pem',
                            '-noout', '-dates', '-serial', '-fingerprint', '-sha256',
                            '-ext', 'subjectAltName'])
result['certificate_chain'] = run(['openssl', 'verify', '-purpose', 'sslserver',
    '-verify_ip', ip, '-CAfile', '/etc/pki/tls/certs/ca-bundle.crt',
    '-untrusted', lineage + '/chain.pem', lineage + '/cert.pem'])
result['served'] = {}
for host in ['127.0.0.1', ip]:
    try:
        with socket.create_connection((host, 443), timeout=8) as raw:
            with ssl.create_default_context().wrap_socket(raw, server_hostname=ip) as tls:
                result['served'][host] = dict(sha256=hashlib.sha256(tls.getpeercert(True)).hexdigest(),
                                             protocol=tls.version(), verified_ip=ip)
    except Exception as e:
        result['served'][host] = dict(error=type(e).__name__)
commands = {
    'apache_syntax': ['httpd', '-t'],
    'apache_vhosts': ['httpd', '-S'],
    'timer': ['systemctl', 'show', 'alma-portal-renew.timer', '-p', 'ActiveState',
              '-p', 'UnitFileState', '-p', 'NextElapseUSecRealtime', '-p', 'LastTriggerUSec',
              '-p', 'TimersCalendar', '-p', 'Persistent'],
    'renewal_service': ['systemctl', 'show', 'alma-portal-renew.service', '-p', 'Result',
                        '-p', 'ExecMainStatus', '-p', 'ExecStart', '-p', 'TimeoutStartUSec'],
    'listeners': ['ss', '-lntup'],
    'firewall_runtime': ['firewall-cmd', '--zone=public', '--list-all'],
    'firewall_permanent': ['firewall-cmd', '--permanent', '--zone=public', '--list-all'],
    'sip_guard': ['nft', 'list', 'table', 'inet', 'alma_portal_sip'],
    'timers': ['systemctl', 'list-timers', '--all', '--no-pager'],
    'root_cron': ['crontab', '-l'],
    'asterisk_cron': ['crontab', '-u', 'asterisk', '-l'],
    'recent_renewal': ['journalctl', '-u', 'alma-portal-renew.service', '-n', '65', '--no-pager'],
    'asterisk_version': ['asterisk', '-rx', 'core show version'],
}
result['commands'] = {name: run(argv) for name, argv in commands.items()}
result['services'] = {unit: run(['systemctl', 'show', unit, '-p', 'ActiveState',
    '-p', 'SubState', '-p', 'MainPID', '-p', 'ExecMainStartTimestamp', '-p', 'NRestarts']) for unit in [
    'httpd', 'asterisk', 'mariadb', 'alma-portal-443', 'alma-portal-443-apply.timer',
    'alma-portal-sip-guard', 'firewalld', 'fail2ban', 'sshd']}
result['state'] = {n: json.loads((Path('/var/lib/alma-portal-tls') / n).read_text())
                   for n in ['status.json', 'deployment.json']
                   if (Path('/var/lib/alma-portal-tls') / n).exists()}


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


opener = urllib.request.build_opener(NoRedirect, urllib.request.ProxyHandler({}))
result['http'] = []
for scheme, path, accept, ajax in [
    ('https', '/', 'text/html', False),
    ('https', '/portal/', 'text/html', False),
    ('https', '/agc/vicidial.php', 'text/html', False),
    ('https', '/vicidial/admin.php', 'text/html', False),
    ('https', '/agc/vicidial.php', 'application/json', True),
    ('https', '/vicidial/admin.php', 'application/json', True),
    ('https', '/agc/nonexistent-qa-resource.js', '*/*', False),
    ('http', '/', 'text/html', False),
    ('http', '/.well-known/acme-challenge/pr08-read-only-missing', '*/*', False),
]:
    headers = {'Accept': accept}
    if ajax:
        headers['X-Requested-With'] = 'XMLHttpRequest'
    request = urllib.request.Request(scheme + '://' + ip + path, headers=headers)
    try:
        response = opener.open(request, timeout=10)
    except urllib.error.HTTPError as e:
        response = e
    except Exception as e:
        result['http'].append(dict(path=path, scheme=scheme, error=type(e).__name__))
        continue
    with response:
        result['http'].append(dict(scheme=scheme, path=path, accept=accept, ajax=ajax,
                                  code=response.code, location=response.headers.get('Location'),
                                  content_type=response.headers.get('Content-Type')))
print(json.dumps(result, indent=2))
