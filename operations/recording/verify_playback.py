#!/usr/bin/env python3
"""Metadata-only deployed verification; never supplies credentials or reads media bodies."""
from pathlib import Path
import subprocess

REMOTE = r'''
import http.client,json,ssl,subprocess,sys,time
from pathlib import Path
name='20261107-024337_7000000001-all.wav'
legacy='/RECORDINGS/'+name
canonical='/vicidial/recordings/'+name
results={}

def request(host,port,path,headers=None,method='HEAD',tls=False):
 c=http.client.HTTPSConnection(host,port,context=ssl.create_default_context(),timeout=8) if tls else http.client.HTTPConnection(host,port,timeout=8)
 c.request(method,path,headers=headers or {})
 r=c.getresponse()
 result=dict(status=r.status,headers={k:v for k,v in r.getheaders() if k.lower() in ('location','www-authenticate','content-type','content-length','cache-control','content-range')})
 # GET probes are limited to public homepage and known denial responses, never audio.
 if method=='GET':
  assert not r.getheader('Content-Type','').startswith('audio/'),'Unexpected media; do not collect body'
  result['body_bytes']=len(r.read(200000))
 c.close();return result

web=Path('/var/lib/alma-portal-443/web')
active=[p.name for p in web.iterdir() if int(p.read_text())>time.time()]
assert active,'Operator must authorize an IP before loopback missing-credential test'
headers={'Host':'2.29.153.161','X-Alma-Source':active[0]}
results['loopback_missing_credentials']=request('127.0.0.1',19091,canonical,headers)
results['loopback_legacy']=request('127.0.0.1',19091,legacy,headers)
assert results['loopback_missing_credentials']['status']==401
assert results['loopback_missing_credentials']['headers']['WWW-Authenticate']=='Basic realm="CONTACT-CENTER-ADMIN"'
assert results['loopback_legacy']['status']==307
results['public_homepage']=request('2.29.153.161',443,'/',method='GET',tls=True)
assert results['public_homepage']['status']==303
assert results['public_homepage']['headers']['Location']=='/portal/'
results['public_portal']=request('2.29.153.161',443,'/portal/',method='GET',tls=True)
assert results['public_portal']['status']==200
for key,path,extra,method in [
 ('unauthorized_canonical',canonical,{},'HEAD'),('unauthorized_legacy',legacy,{},'HEAD'),
 ('spoofed_source',canonical,{'X-Alma-Source':active[0],'X-Forwarded-For':active[0],'X-Real-IP':active[0]},'HEAD'),
 ('directory','/RECORDINGS/',{},'HEAD'),('raw','/RECORDINGS/20261107-024337_7000000001-in.wav',{},'HEAD'),
 ('agent_browser','/agc/vicidial.php',{'Accept':'text/html','Sec-Fetch-Mode':'navigate','Sec-Fetch-Dest':'document'},'GET'),
 ('admin_browser','/vicidial/admin.php',{'Accept':'text/html','Sec-Fetch-Mode':'navigate','Sec-Fetch-Dest':'document'},'GET'),
 ('agent_api','/agc/vicidial.php',{'Accept':'application/json'},'GET')]:
 results[key]=request('2.29.153.161',443,path,extra,method,tls=True)
 assert results[key]['status']==(303 if key.endswith('_browser') else 403),(key,results[key])
results['http_root']=request('2.29.153.161',80,'/')
assert results['http_root']['status']==403
# Verify added SELECTs and file opening as the actual unprivileged service user.
# Do not select password/hash/key values or consume any production audio bytes.
code="""
import json,sys
sys.path.insert(0,'/opt/alma-portal-443')
import vicidial_auth as auth, recording_media as media
conn=auth._connect()
with conn.cursor() as c:
 c.execute('SELECT access_recordings,view_reports,user_group,failed_login_count,last_login_date FROM vicidial_users WHERE user=%s',('6666',));user=c.fetchone();user.pop('last_login_date')
 c.execute('SELECT two_factor_auth_hours,allow_ip_lists,LENGTH(system_ip_blacklist) AS blacklist_length,log_recording_access FROM system_settings LIMIT 1');settings=c.fetchone()
 c.execute('SELECT allowed_campaigns,allowed_reports,admin_viewable_groups FROM vicidial_user_groups WHERE user_group=%s',(user['user_group'],));group=c.fetchone()
 c.execute('SELECT recording_id,filename,end_epoch,length_in_sec,location FROM recording_log WHERE recording_id=2');record=c.fetchone()
stream,size=media.open_recording(record,'20261107-024337_7000000001-all.wav');stream.close();conn.close()
print(json.dumps(dict(user=user,settings=settings,global_group=group,open_verified_inode_size=size)))
"""
p=subprocess.run(['runuser','-u','alma-portal','--','/opt/alma-portal-venv/bin/python','-B','-c',code],capture_output=True,text=True,timeout=10)
if p.returncode:
 print(json.dumps(dict(stage='unprivileged_metadata',stderr=p.stderr)));raise SystemExit(p.returncode)
results['unprivileged_metadata_and_file']=json.loads(p.stdout)
assert results['unprivileged_metadata_and_file']['open_verified_inode_size']==892204
print(json.dumps(results,indent=2))
'''
if __name__ == '__main__':
 p=subprocess.run(['ssh','-T','-p','22222','root@2.29.153.161','/usr/bin/python3','-B','-'],input=REMOTE,text=True,capture_output=True)
 Path(__file__).parent.parent.joinpath('qa/evidence/recording-playback/integration.json').write_text(p.stdout if p.returncode==0 else p.stdout+p.stderr)
 print(p.stdout,p.stderr)
 raise SystemExit(p.returncode)
