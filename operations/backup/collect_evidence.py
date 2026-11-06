#!/usr/bin/python3
"""Read-only sanitized host evidence; run through SSH stdin, never copy archives."""
import hashlib
import json
from pathlib import Path
import socket
import ssl
import subprocess
import time
import urllib.error
import urllib.request


def run(argv):
    try:
        p = subprocess.run(argv, capture_output=True, text=True, timeout=25)
        return dict(rc=p.returncode, stdout=p.stdout[:20000], stderr=p.stderr[:1000])
    except Exception as e:
        return dict(error=type(e).__name__)


result = dict(time_utc=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()), host=socket.gethostname())
commands = {
    'database': ['mariadb', '--batch', '--skip-column-names', '-e',
        "SELECT VERSION(); SELECT TABLE_SCHEMA,COALESCE(ENGINE,'VIEW'),COUNT(*),SUM(DATA_LENGTH+INDEX_LENGTH) "
        "FROM information_schema.TABLES WHERE TABLE_SCHEMA NOT IN ('information_schema','performance_schema','sys') "
        "GROUP BY TABLE_SCHEMA,ENGINE; SHOW VARIABLES WHERE Variable_name IN ('log_bin','socket'); "
        "SELECT COUNT(*) FROM asterisk.vicidial_live_agents; SELECT COUNT(*) FROM asterisk.vicidial_auto_calls;"],
    'channels': ['asterisk', '-rx', 'core show channels count'],
    'apache_syntax': ['httpd', '-t'],
    'listeners': ['ss', '-lntu'],
    'firewall_runtime': ['firewall-cmd', '--zone=public', '--list-all'],
    'firewall_permanent': ['firewall-cmd', '--permanent', '--zone=public', '--list-all'],
    'sip_gate': ['nft', '-j', 'list', 'table', 'inet', 'alma_portal_sip'],
    'disk': ['df', '-hT', '/'],
    'inodes': ['df', '-i', '/'],
    'backup_timer': ['systemctl', 'show', 'alma-vicidial-backup.timer', '-p', 'ActiveState', '-p',
                     'UnitFileState', '-p', 'TimersCalendar', '-p', 'NextElapseUSecRealtime', '-p', 'Persistent'],
    'backup_service': ['systemctl', 'show', 'alma-vicidial-backup.service', '-p', 'Result', '-p', 'ExecMainStatus', '-p', 'ActiveState'],
    'tls_timer': ['systemctl', 'show', 'alma-portal-renew.timer', '-p', 'ActiveState', '-p', 'UnitFileState', '-p', 'Persistent'],
    'worker': ['systemctl', 'show', 'alma-portal-443-apply.service', '-p', 'Result', '-p', 'ExecMainStatus']}
result['commands'] = {k: run(v) for k, v in commands.items()}
result['services'] = {name: run(['systemctl', 'show', name, '-p', 'MainPID', '-p', 'ActiveState', '-p', 'ExecMainStartTimestamp', '-p', 'NRestarts']) for name in
    ['asterisk', 'mariadb', 'httpd', 'php-fpm', 'alma-portal-443', 'firewalld', 'fail2ban', 'sshd', 'crond']}
paths = ['/etc/cron.d/vicidial', '/etc/my.cnf.d/91-vicidial-sql-compat.cnf',
    '/etc/httpd/conf.d/02-alma-vicidial-ssl.conf', '/etc/httpd/conf.d/00-alma-acme.conf',
    '/usr/local/libexec/alma-portal-tls', '/usr/local/sbin/alma-portal-cert-deploy',
    '/etc/systemd/system/alma-portal-renew.service', '/etc/systemd/system/alma-portal-renew.timer',
    '/etc/asterisk/extensions.conf', '/etc/asterisk/sip.conf', '/etc/asterisk/pjsip.conf',
    '/etc/asterisk/rtp.conf', '/opt/alma-portal-443/common.py', '/opt/alma-portal-443/access.lua',
    '/opt/alma-portal-443/apply_requests.py', '/opt/alma-portal-443/sip_guard.py',
    '/usr/share/astguiclient/ADMIN_keepalive_ALL.pl', '/usr/share/astguiclient/AST_manager_send.pl',
    '/usr/share/astguiclient/AST_VDauto_dial.pl', '/usr/share/astguiclient/AST_VDremote_agents.pl',
    '/usr/local/sbin/alma-vicidial-backup', '/etc/alma-vicidial-backup.json',
    '/etc/systemd/system/alma-vicidial-backup.service', '/etc/systemd/system/alma-vicidial-backup.timer']
result['hashes'] = {p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in paths if Path(p).is_file()}
recordings = []
for root in ['/var/spool/asterisk/monitor', '/var/spool/asterisk/monitorDONE']:
    for p in sorted(Path(root).rglob('*')):
        if p.is_file():
            recordings.append((str(p), p.stat().st_size, hashlib.sha256(p.read_bytes()).hexdigest()))
result['recordings'] = dict(files=len(recordings), bytes=sum(r[1] for r in recordings),
    inventory_sha256=hashlib.sha256(json.dumps(recordings).encode()).hexdigest())
base = Path('/var/lib/alma-vicidial-backup')
if base.exists():
    result['health'] = run(['/usr/local/sbin/alma-vicidial-backup', 'health'])
    result['backup_permissions'] = [dict(path=str(p), mode=oct(p.stat().st_mode & 0o777), uid=p.stat().st_uid)
        for p in [base, base / 'complete', *base.glob('complete/*'), *base.glob('complete/*/*')]]
    result['complete_backups'] = []
    for path in sorted((base / 'complete').iterdir()):
        manifest = json.loads((path / 'manifest.json').read_text())
        result['complete_backups'].append(dict(id=path.name, created_at=manifest['created_at'],
            archive_bytes=manifest['archive_bytes'], database=manifest['database'],
            archive_members=manifest['archive_members'], included_roots=manifest['included_roots'],
            recordings=len(manifest['recording_files']), required_members=len(manifest['required_archive_members']),
            checksums_sha256=hashlib.sha256((path / 'SHA256SUMS').read_bytes()).hexdigest(),
            verification=json.loads((path / 'VERIFIED.json').read_text())))
ip = '2.29.153.161'
with socket.create_connection((ip, 443), timeout=8) as raw:
    with ssl.create_default_context().wrap_socket(raw, server_hostname=ip) as tls:
        result['served_tls'] = dict(sha256=hashlib.sha256(tls.getpeercert(True)).hexdigest(), version=tls.version())


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


opener = urllib.request.build_opener(NoRedirect, urllib.request.ProxyHandler({}))
result['http'] = []
for path, accept, ajax in [('/', 'text/html', False), ('/portal/', 'text/html', False),
    ('/agc/vicidial.php', 'text/html', False), ('/vicidial/admin.php', 'text/html', False),
    ('/agc/vicidial.php', 'application/json', True), ('/vicidial/admin.php', 'application/json', True)]:
    headers = {'Accept': accept}
    if ajax:
        headers['X-Requested-With'] = 'XMLHttpRequest'
    try:
        response = opener.open(urllib.request.Request('https://' + ip + path, headers=headers), timeout=8)
    except urllib.error.HTTPError as e:
        response = e
    with response:
        result['http'].append(dict(path=path, ajax=ajax, code=response.code, location=response.headers.get('Location')))
print(json.dumps(result, indent=2))
