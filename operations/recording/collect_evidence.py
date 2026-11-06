#!/usr/bin/python3
"""Read-only metadata collector for SSH stdin. No credentials or audio output."""
import hashlib
import json
from pathlib import Path
import socket
import ssl
import subprocess
import time
import urllib.error
import urllib.request
import wave


def run(argv):
    p = subprocess.run(argv, capture_output=True, text=True, timeout=35)
    return dict(rc=p.returncode, stdout=p.stdout[:20000], stderr=p.stderr[:1000])


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


result = dict(host=socket.gethostname(), utc=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()))
paths = ['/etc/cron.d/vicidial', '/etc/httpd/conf.d/02-alma-vicidial-ssl.conf',
         '/etc/httpd/conf.d/00-alma-acme.conf', '/etc/asterisk/extensions.conf',
         '/etc/asterisk/sip.conf', '/etc/asterisk/pjsip.conf', '/etc/asterisk/rtp.conf',
         '/opt/alma-portal-443/access.lua', '/opt/alma-portal-443/common.py',
         '/opt/alma-portal-443/apply_requests.py', '/opt/alma-portal-443/sip_guard.py',
         '/usr/local/sbin/alma-portal-cert-deploy', '/usr/local/libexec/alma-portal-tls',
         '/usr/local/sbin/alma-vicidial-backup', '/etc/alma-vicidial-backup.json',
         '/etc/systemd/system/alma-portal-renew.service', '/etc/systemd/system/alma-portal-renew.timer',
         '/etc/systemd/system/alma-vicidial-backup.service', '/etc/systemd/system/alma-vicidial-backup.timer',
         '/usr/share/astguiclient/AST_CRON_audio_1_move_mix.pl',
         '/usr/share/astguiclient/AST_CRON_audio_1_move_VDonly.pl',
         '/usr/share/astguiclient/AST_CRON_audio_2_compress.pl',
         '/var/www/html/vicidial/recording_log_redirect.php', '/var/www/html/vicidial/user_stats.php',
         '/var/www/html/vicidial/admin_modify_lead.php',
         '/usr/local/sbin/alma-vicidial-recording', '/etc/alma-vicidial-recording.json',
         '/etc/systemd/system/alma-vicidial-recording.service', '/etc/systemd/system/alma-vicidial-recording.timer']
result['hashes'] = {p: sha(Path(p)) for p in paths if Path(p).is_file()}
result['services'] = {n: run(['systemctl', 'show', n, '-p', 'MainPID', '-p', 'ActiveState',
                             '-p', 'ExecMainStartTimestamp', '-p', 'NRestarts']) for n in
    ['asterisk', 'mariadb', 'httpd', 'php-fpm', 'alma-portal-443', 'firewalld', 'fail2ban', 'sshd', 'crond']}
result['timers'] = {n: run(['systemctl', 'show', n, '-p', 'ActiveState', '-p', 'UnitFileState',
                           '-p', 'NextElapseUSecRealtime', '-p', 'NextElapseUSecMonotonic', '-p', 'Persistent']) for n in
    ['alma-portal-renew.timer', 'alma-vicidial-backup.timer', 'alma-vicidial-recording.timer']}
result['channels'] = run(['asterisk', '-rx', 'core show channels count'])
result['dialplan'] = run(['asterisk', '-rx', 'dialplan show 8309@default'])
result['listeners'] = run(['ss', '-lntu'])
result['apache'] = run(['httpd', '-t'])
sql = """SELECT recording_id,filename,start_epoch,end_epoch,length_in_sec,COALESCE(location,'NULL')
FROM asterisk.recording_log WHERE recording_id IN (1,2) ORDER BY recording_id;
SELECT uniqueid,call_date,length_in_sec,status,campaign_id,lead_id FROM asterisk.vicidial_log
WHERE uniqueid='1793990617.45';
SELECT stereo_recording,stereo_parallel_recording,log_recording_access FROM asterisk.system_settings;
SELECT active,user_level,access_recordings,COUNT(*) FROM asterisk.vicidial_users GROUP BY 1,2,3;"""
result['database'] = run(['mariadb', '--no-defaults', '-N', '-B', '-e', sql])
result['cron'] = Path('/etc/cron.d/vicidial').read_text()
result['recordings'] = []
for root in ('/var/spool/asterisk/monitor', '/var/spool/asterisk/monitorDONE'):
    for p in sorted(Path(root).rglob('*.wav')):
        # Known synthetic test recordings only; exclude unrelated calls.
        if p.name.startswith(('20261107-024337_7000000001-', '20261106-024951_7000000001-')):
            s = p.stat()
            with wave.open(str(p), 'rb') as w:
                item = dict(path=str(p), size=s.st_size, sha256=sha(p), mode=oct(s.st_mode & 0o777),
                            channels=w.getnchannels(), rate=w.getframerate(), width=w.getsampwidth(),
                            frames=w.getnframes(), seconds=w.getnframes()/w.getframerate())
            result['recordings'].append(item)
result['firewall'] = {}
for label, argv in [('runtime', ['firewall-cmd', '--zone=public', '--list-all']),
                    ('permanent', ['firewall-cmd', '--permanent', '--zone=public', '--list-all'])]:
    output = run(argv)
    result['firewall'][label] = dict(rc=output['rc'], sha256=hashlib.sha256(output['stdout'].encode()).hexdigest())
gate = json.loads(run(['nft', '-j', 'list', 'table', 'inet', 'alma_portal_sip'])['stdout'])
def clear_counters(obj):
    if isinstance(obj, dict):
        return {k: clear_counters(v) for k,v in obj.items() if k not in ('metainfo', 'packets', 'bytes', 'expires')}
    return [clear_counters(v) for v in obj] if isinstance(obj,list) else obj
result['sip_gate_policy_sha256'] = hashlib.sha256(json.dumps(clear_counters(gate), sort_keys=True).encode()).hexdigest()
result['worker'] = run(['systemctl', 'show', 'alma-portal-443-apply.service', '-p', 'Result', '-p', 'ExecMainStatus'])
result['backup_health'] = run(['/usr/local/sbin/alma-vicidial-backup', 'health'])
result['backup_complete_sets'] = len(list(Path('/var/lib/alma-vicidial-backup/complete').iterdir()))
if Path('/usr/local/sbin/alma-vicidial-recording').exists():
    result['recording_health'] = run(['/usr/local/sbin/alma-vicidial-recording', '--health'])
    result['recording_service'] = run(['systemctl', 'show', 'alma-vicidial-recording.service', '-p', 'Result', '-p', 'ExecMainStatus'])
    result['receipts'] = {p.name: json.loads(p.read_text()) for p in Path('/var/lib/alma-vicidial-recording').glob('[0-9]*.json')}
with socket.create_connection(('2.29.153.161',443),timeout=10) as connection:
    with ssl.create_default_context().wrap_socket(connection,server_hostname='2.29.153.161') as tls:
        result['tls'] = dict(sha256=hashlib.sha256(tls.getpeercert(True)).hexdigest(), version=tls.version())
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args):
        return None
opener=urllib.request.build_opener(NoRedirect,urllib.request.ProxyHandler({}))
result['http'] = []
for scheme,path,accept,ajax in [('https','/','text/html',False), ('https','/portal/','text/html',False),
    ('https','/agc/vicidial.php','text/html',False), ('https','/vicidial/admin.php','text/html',False),
    ('https','/agc/vicidial.php','application/json',True), ('https','/vicidial/admin.php','application/json',True),
    ('https','/RECORDINGS/','text/html',False), ('https','/RECORDINGS/PR01_SYNTHETIC-all.wav','audio/wav',False),
    ('https','/RECORDINGS/20261107-024337_7000000001-all.wav','audio/wav',False),
    ('http','/RECORDINGS/PR01_SYNTHETIC-all.wav','audio/wav',False), ('http','/','text/html',False)]:
    headers={'Accept':accept}
    if ajax:
        headers['X-Requested-With']='XMLHttpRequest'
    method='HEAD' if '/RECORDINGS' in path else 'GET'
    req=urllib.request.Request(scheme+'://2.29.153.161'+path,headers=headers,method=method)
    try:
        response=opener.open(req,timeout=10)
    except urllib.error.HTTPError as e:
        response=e
    with response:
        result['http'].append(dict(scheme=scheme,path=path,method=method,ajax=ajax,code=response.code,
                                   location=response.headers.get('Location')))
print(json.dumps(result,indent=2))
