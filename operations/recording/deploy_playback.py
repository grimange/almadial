#!/usr/bin/env python3
"""Explicit PR-01C deployment only; never native source, accounts or firewall writes."""
import base64
import json
from pathlib import Path
import subprocess

HERE = Path(__file__).parent
EVIDENCE = HERE.parent / 'qa/evidence/recording-playback'
baseline = json.loads((EVIDENCE / 'before.json').read_text())
mapping = [
 (HERE.parent/'portal443/portal.py','/opt/alma-portal-443/portal.py',0o644),
 (HERE.parent/'portal443/vicidial_auth.py','/opt/alma-portal-443/vicidial_auth.py',0o644),
 (HERE.parent/'portal443/recording_media.py','/opt/alma-portal-443/recording_media.py',0o644),
 (HERE.parent/'portal443/02-alma-vicidial-ssl.conf','/etc/httpd/conf.d/02-alma-vicidial-ssl.conf',0o644),
 (HERE/'enroll_playback.py','/usr/local/sbin/alma-vicidial-recording-enroll',0o700),
 (HERE/'playback-backend.conf','/etc/systemd/system/alma-portal-443.service.d/recording-playback.conf',0o644),
 (HERE/'playback-enroll.conf','/etc/systemd/system/alma-vicidial-recording.service.d/playback-enroll.conf',0o644)]
payload = {dest:dict(data=base64.b64encode(src.read_bytes()).decode(),mode=mode)
           for src,dest,mode in mapping}

REMOTE = r'''
import base64,datetime,hashlib,json,os,shutil,socket,subprocess,tarfile,tempfile
from pathlib import Path
def run(*args,**kwargs):
 return subprocess.run(args,check=True,capture_output=True,text=True,timeout=60,**kwargs).stdout
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def sql(query):return run('mariadb','--no-defaults','-B','-N',input=query)
assert os.geteuid()==0 and socket.gethostname()=='dialer01'
os.umask(0o077)
for name,want in baseline['files'].items():
 assert sha(Path(name))==want['sha256'],'Concurrent change: '+name
assert sql("SELECT access_recordings FROM asterisk.vicidial_users WHERE user='6666';").strip()=='1'
assert sql('SELECT log_recording_access FROM asterisk.system_settings;').strip()=='0'
assert subprocess.run(['systemctl','is-active','--quiet','alma-vicidial-recording.service']).returncode!=0,'Wait for finalizer'
health=json.loads(run('/usr/local/sbin/alma-vicidial-backup','health'))
assert health['outcome']=='ok' and health['last_success_verification_present'] and health['newest_complete_age_seconds']<86400
assert not Path('/var/lib/alma-recording-playback').exists(),'Already enrolled; inspect before redeploy'
old=[Path(n) for n in payload if Path(n).exists()]
new=[n for n in payload if not Path(n).exists()]
assert set(new)=={n for n in payload if n not in baseline['files']},'Unexpected installed file'
backup=Path('/root/alma-recording-pr01c-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
backup.mkdir(mode=0o700)
manifest={str(p):dict(sha256=sha(p),mode=p.stat().st_mode&0o777,uid=p.stat().st_uid,gid=p.stat().st_gid) for p in old}
run('tar','--acls','--xattrs','--selinux','-C','/','-cpf',str(backup/'originals.tar'),*[str(p).lstrip('/') for p in old])
(backup/'manifest.json').write_text(json.dumps(manifest,indent=2))
with tempfile.TemporaryDirectory(dir=backup) as tmp:
 run('tar','--acls','--xattrs','--selinux','-C',tmp,'-xpf',str(backup/'originals.tar'))
 for name,want in manifest.items():
  p=Path(tmp)/name.lstrip('/');s=p.stat()
  assert sha(p)==want['sha256'] and (s.st_mode&0o777,s.st_uid,s.st_gid)==(want['mode'],want['uid'],want['gid'])
required={
 'system_settings':{'two_factor_auth_hours','allow_ip_lists','system_ip_blacklist','log_recording_access'},
 'vicidial_users':{'access_recordings','view_reports','user_group','failed_login_count','last_login_date'},
 'vicidial_user_groups':{'user_group','allowed_campaigns','allowed_reports','admin_viewable_groups'},
 'recording_log':{'recording_id','filename','end_epoch','length_in_sec','location'}}
rows=sql("SELECT TABLE_NAME,COLUMN_NAME FROM information_schema.COLUMN_PRIVILEGES WHERE GRANTEE=CONCAT(CHAR(39),'alma_portal_auth',CHAR(39),'@',CHAR(39),'localhost',CHAR(39));")
existing={tuple(r.split('\t')) for r in rows.splitlines()}
added={t:sorted(c for c in cols if (t,c) not in existing) for t,cols in required.items()}
added={t:c for t,c in added.items() if c}
revoke='\n'.join('REVOKE SELECT ('+','.join(cols)+') ON asterisk.'+t+" FROM 'alma_portal_auth'@'localhost';" for t,cols in added.items())
(backup/'revoke.sql').write_text(revoke+'\n')
rollback=r"""#!/usr/bin/python3
import hashlib,json,os,subprocess,sys
from pathlib import Path
base=Path(__file__).resolve().parent
def run(*a,**kw):return subprocess.run(a,check=True,timeout=60,**kw)
if '--check' in sys.argv:
 manifest=json.loads((base/'manifest.json').read_text())
 assert manifest and (base/'originals.tar').is_file() and (base/'revoke.sql').is_file()
 print('Rollback metadata and archive present; production unchanged');raise SystemExit(0)
assert os.geteuid()==0
assert subprocess.run(['systemctl','is-active','--quiet','alma-vicidial-recording.service']).returncode!=0,'Wait for finalizer'
# Close the media route before removing its verifier. Restore only PR01C originals.
run('tar','--acls','--xattrs','--selinux','-C','/','-xpf',str(base/'originals.tar'))
run('httpd','-t');run('systemctl','reload','httpd')
for n in NEW_FILES:Path(n).unlink(missing_ok=True)
run('systemctl','daemon-reload');run('systemctl','restart','alma-portal-443')
run('mariadb','--no-defaults',input=(base/'revoke.sql').read_bytes())
for p in Path('/var/lib/alma-recording-playback').glob('[0-9]*.json'):
 m=json.loads(p.read_text());f=Path('/var/spool/asterisk/monitorDONE')/m['name']
 assert f.parent==Path('/var/spool/asterisk/monitorDONE') and not f.is_symlink()
 s=f.stat();assert s.st_uid==0 and s.st_nlink==1
 assert hashlib.sha256(f.read_bytes()).hexdigest()==m['sha256'],'Changed audio; inspect permissions manually'
 os.chown(f,0,m['original']['gid']);os.chmod(f,m['original']['mode'])
print('PR01C rollback complete; raw/final audio, native metadata and recording timer preserved')
""".replace('NEW_FILES',repr(new))
(backup/'rollback.py').write_text(rollback);(backup/'rollback.py').chmod(0o700)
compile(rollback,'rollback','exec')
run(str(backup/'rollback.py'),'--check')
print('ROLLBACK_READY '+str(backup),flush=True)
try:
 for table,cols in added.items():
  sql('GRANT SELECT ('+','.join(cols)+') ON asterisk.'+table+" TO 'alma_portal_auth'@'localhost';")
 # Apache route is last: prior live configuration continues denying recordings.
 apache='/etc/httpd/conf.d/02-alma-vicidial-ssl.conf'
 for name,item in payload.items():
  if name==apache:continue
  p=Path(name);p.parent.mkdir(exist_ok=True)
  temp=p.with_name(p.name+'.pr01c-new');temp.write_bytes(base64.b64decode(item['data']));temp.chmod(item['mode']);os.replace(temp,p)
 for name in ['/opt/alma-portal-443/portal.py','/opt/alma-portal-443/vicidial_auth.py','/opt/alma-portal-443/recording_media.py','/usr/local/sbin/alma-vicidial-recording-enroll']:
  compile(Path(name).read_text(),name,'exec')
 enrollment=run('/usr/local/sbin/alma-vicidial-recording-enroll')
 repeat=run('/usr/local/sbin/alma-vicidial-recording-enroll')
 run('restorecon',*[n for n in payload if n!=apache],'/var/lib/alma-recording-playback')
 run('systemctl','daemon-reload')
 run('systemd-analyze','verify','alma-portal-443.service','alma-vicidial-recording.service')
 run('systemctl','restart','alma-portal-443')
 run('systemctl','is-active','alma-portal-443')
 p=Path(apache);p.write_bytes(base64.b64decode(payload[apache]['data']));p.chmod(payload[apache]['mode'])
 run('restorecon',apache);run('httpd','-t');run('systemctl','reload','httpd')
 print(json.dumps(dict(backup=str(backup),rollback='Verified archive extraction, metadata equality, compile, --check; no live rollback executed',
  enrollment=enrollment,repeat=repeat,added_select_columns=added,
  files={n:sha(Path(n)) for n in payload},apache='Syntax OK; graceful reload',
  service_actions=['daemon-reload','restart alma-portal-443','reload httpd']),indent=2))
except Exception:
 print('PR01C_DEPLOYMENT_FAILED; restoring private route and prior verifier',flush=True)
 run(str(backup/'rollback.py'))
 raise
'''

if __name__ == '__main__':
    code = 'payload='+repr(payload)+'\nbaseline='+repr(baseline)+'\n'+REMOTE
    result = subprocess.run(['ssh','-T','-p','22222','-o','BatchMode=yes','root@2.29.153.161',
                             '/usr/bin/python3','-B','-'],input=code,text=True,capture_output=True)
    (EVIDENCE/'deployment.txt').write_text(result.stdout+result.stderr)
    print(result.stdout,result.stderr)
    raise SystemExit(result.returncode)
