#!/usr/bin/env python3
"""Bounded sanitized PR-01C inspection over noninteractive SSH."""
from pathlib import Path
import subprocess
import sys

REMOTE = r'''
import datetime,hashlib,json,socket,subprocess
from pathlib import Path
def run(*args):
 p=subprocess.run(args,capture_output=True,text=True,timeout=30)
 return dict(code=p.returncode,stdout=p.stdout.strip(),stderr=p.stderr.strip())
paths=['/opt/alma-portal-443/portal.py','/opt/alma-portal-443/vicidial_auth.py',
 '/opt/alma-portal-443/access.lua','/etc/httpd/conf.d/02-alma-vicidial-ssl.conf',
 '/usr/local/sbin/alma-vicidial-recording','/etc/alma-vicidial-recording.json',
 '/etc/systemd/system/alma-vicidial-recording.timer','/etc/cron.d/vicidial',
 '/usr/local/sbin/alma-portal-cert-deploy']
paths += ['/var/www/html/vicidial/'+n for n in ['admin.php','functions.php','user_stats.php',
 'admin_modify_lead.php','recording_log_redirect.php']]
files={}
for name in paths:
 p=Path(name)
 if p.exists():
  s=p.stat();files[name]=dict(sha256=hashlib.sha256(p.read_bytes()).hexdigest(),mode=s.st_mode&0o777,uid=s.st_uid,gid=s.st_gid)
services={n:run('systemctl','show',n,'-p','ActiveState','-p','SubState','-p','MainPID','-p','ExecMainStartTimestamp','-p','UnitFileState')
 for n in ['httpd','asterisk','mariadb','php-fpm','alma-portal-443','alma-portal-443-apply.timer',
 'alma-portal-sip-guard','alma-vicidial-recording.timer','alma-vicidial-backup.timer','alma-portal-renew.timer','fail2ban']}
sql="""SELECT user,active,user_level,access_recordings,view_reports,user_group FROM asterisk.vicidial_users WHERE user='6666';
SELECT log_recording_access,two_factor_auth_hours,allow_ip_lists,LENGTH(system_ip_blacklist) FROM asterisk.system_settings;
SELECT recording_id,filename,end_epoch,length_in_sec,location FROM asterisk.recording_log WHERE recording_id IN(1,2);
SELECT TABLE_NAME,COLUMN_NAME,PRIVILEGE_TYPE FROM information_schema.COLUMN_PRIVILEGES WHERE GRANTEE=CONCAT(CHAR(39),'alma_portal_auth',CHAR(39),'@',CHAR(39),'localhost',CHAR(39)) ORDER BY TABLE_NAME,COLUMN_NAME;
"""
result=dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(),host=socket.gethostname(),files=files,services=services,
 database=run('mariadb','--no-defaults','-B','-e',sql),
 backup=run('/usr/local/sbin/alma-vicidial-backup','health'),
 firewalld_permanent=run('firewall-cmd','--permanent','--zone=public','--list-all'),
 firewalld_runtime=run('firewall-cmd','--zone=public','--list-all'),
 nft_rules=run('nft','list','table','inet','alma_portal_sip'),
 listeners=run('ss','-lntup'),apache=run('httpd','-t'),
 calls=run('asterisk','-rx','core show channels count'))
final=Path('/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav')
s=final.stat()
result['final_recording']=dict(size=s.st_size,sha256=hashlib.sha256(final.read_bytes()).hexdigest(),mode=s.st_mode&0o777,uid=s.st_uid,gid=s.st_gid)
print(json.dumps(result,indent=2))
'''

if __name__ == '__main__':
    result = subprocess.run(['ssh','-T','-p','22222','-o','BatchMode=yes',
                             'root@2.29.153.161','/usr/bin/python3','-B','-'],
                            input=REMOTE,text=True,capture_output=True,check=True)
    Path(sys.argv[1]).write_text(result.stdout)
    print('Saved sanitized inspection:',sys.argv[1])
