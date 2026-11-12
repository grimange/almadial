"""S10 bounded registration-only assets, one phone and one settings container.

Credentials stay in protected files/stdin. No dialing, firewall or TLS changes.
"""
import base64
import grp
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import sys
import tempfile

IDENTITY='alm_s10_9061'
CONTAINER='ALMADIAL_S10'
ROOT=Path('/var/www/html/almadial-phone')
WORK=Path('/var/lib/almadial/phone-s10')
SECRET=Path('/etc/almadial/secrets/phone-s10')
WRAPPER=Path('/etc/asterisk/pjsip.conf')
BASE=b'#include almadial-s04-pjsip.conf\n#include "pjsip-vicidial.conf"\n'
WRAPPED=BASE+b'#include "almadial-s10-phone.conf"\n'
PJSIP=Path('/etc/asterisk/almadial-s10-phone.conf')
SETTINGS='autoGain:0\\nechoCan:0\\nnoiseSup:0\\ndialRegExten:0\\nprogReg:na'

def sha(data): return hashlib.sha256(data).hexdigest()
def digest(value): return sha(json.dumps(value,sort_keys=True,separators=(',',':')).encode())
def run(argv, data=None):
    r=subprocess.run(argv,input=data,capture_output=True,timeout=45)
    if r.returncode:
        # SQL and PJSIP contents may contain secrets. Report operation only.
        raise ValueError('native operation failed: '+Path(argv[0]).name+' exit '+str(r.returncode))
    return r.stdout.decode().strip()
def sql(query):
    return run(['/usr/bin/mariadb','--defaults-extra-file=/etc/almadial/secrets/database-client.cnf','--batch','--skip-column-names','--raw','asterisk'],query.encode())
def literal(value): return "CONVERT(0x"+str(value).encode().hex()+" USING utf8mb4)" if str(value) else "''"
def safe(path):
    for item in [path]+list(path.parents):
        if item.is_symlink(): raise ValueError('symlink at S10 boundary')
        if item.exists():
            st=item.lstat()
            if st.st_uid!=0 or st.st_mode&0o022: raise ValueError('unsafe S10 ownership')
def read(path, mode, gid=0):
    safe(path)
    st=path.stat()
    if not stat.S_ISREG(st.st_mode) or st.st_nlink!=1 or st.st_gid!=gid or stat.S_IMODE(st.st_mode)!=mode or st.st_size>4<<20:
        raise ValueError('unsafe S10 file metadata')
    return path.read_bytes()
def atomic(path, data, mode, gid=0):
    safe(path)
    fd,name=tempfile.mkstemp(prefix='.almadial-',dir=path.parent)
    try:
        os.fchmod(fd,mode); os.fchown(fd,0,gid)
        with os.fdopen(fd,'wb') as f: f.write(data); f.flush(); os.fsync(f.fileno())
        os.replace(name,path)
        d=os.open(path.parent,os.O_RDONLY|os.O_DIRECTORY)
        try: os.fsync(d)
        finally: os.close(d)
    finally:
        if os.path.exists(name): os.unlink(name)
    run(['/usr/sbin/restorecon',str(path)])
def rows(table,where):
    cols=sql("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='asterisk' AND TABLE_NAME="+literal(table)+" ORDER BY ORDINAL_POSITION").splitlines()
    if not all(re.fullmatch('[a-zA-Z0-9_]+',x) for x in cols): raise ValueError('invalid database columns')
    expressions=','.join('HEX(CAST(`'+x+'` AS BINARY))' for x in cols)
    raw=sql('SELECT JSON_ARRAY('+expressions+') FROM `'+table+'` WHERE '+where)
    data=sorted(raw.splitlines())
    return cols,data

def phone_values(secret):
    return dict(extension=IDENTITY,dialplan_number='9061',voicemail_id='9061',server_ip='127.0.0.1',login=IDENTITY,pass_=secret,
                active='Y',status='ACTIVE',phone_type='ALMADIAL_S10',fullname='AlmaDial synthetic WebRTC',protocol='PJSIP',login_pass=secret,
                ext_context='alm-s10-no-calls',phone_context='alm-s10-no-calls',template_id='--NONE--',
                is_webphone='Y',webphone_settings=CONTAINER,webphone_dialpad='N',webphone_auto_answer='N',webphone_debug='N',codecs_list='ulaw,alaw',codecs_with_template='0')
def desired_values(secret):
    v=phone_values(secret);v['pass']=v.pop('pass_');return v

def config(secret):
    return (''' ; AlmaDial S10: synthetic registration only; no dialplan routes.
[almadial-s10-wss]
type=transport
protocol=wss
bind=127.0.0.1

[alm_s10_9061]
type=endpoint
transport=almadial-s10-wss
context=alm-s10-no-calls
disallow=all
allow=ulaw,alaw
auth=almadial-s10-auth
aors=alm_s10_9061
webrtc=yes
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes

[almadial-s10-auth]
type=auth
auth_type=userpass
username=alm_s10_9061
password='''+secret+'''

[alm_s10_9061]
type=aor
max_contacts=1
remove_existing=yes
minimum_expiration=60
maximum_expiration=600
default_expiration=600
''').lstrip().encode()

# The operator supplies the synthetic phone password through a normal form.
# Settings are read from VICIdial; no SIP secret is exposed in URLs or logs.
LAUNCHER=r'''<?php
header('Cache-Control: no-store');
header('Referrer-Policy: no-referrer');
require('/var/www/html/agc/dbconnect_mysqli.php');
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
?><!doctype html><title>AlmaDial synthetic phone</title><h1>AlmaDial synthetic phone</h1>
<form method="post"><label>Phone <input name="phone" value="alm_s10_9061" readonly></label>
<label>Password <input type="password" name="password" required autocomplete="off"></label>
<button type="submit">Open phone</button></form><?php exit; }
if (($_POST['phone'] ?? '') !== 'alm_s10_9061' || !preg_match('/\A[0-9a-zA-Z]{1,64}\z/',$_POST['password'] ?? '')) {http_response_code(400); exit('Invalid synthetic phone input');}
$result=mysqli_query($link,"SELECT p.login,p.webphone_settings,c.container_entry FROM phones p JOIN vicidial_settings_containers c ON c.container_id=p.webphone_settings WHERE p.extension='alm_s10_9061' AND c.container_id='ALMADIAL_S10'");
$row=$result ? mysqli_fetch_row($result) : false;
if (!$row) {http_response_code(503); exit('Synthetic phone configuration unavailable');}
$fields=['phone_login'=>$row[0],'phone_pass'=>$_POST['password'],'server_ip'=>'localhost','callerid'=>'9061','protocol'=>'PJSIP','codecs'=>'ulaw,alaw','options'=>'WEBSOCKETURLwss://localhost:8089/ws--SETTINGS'.$row[2],'system_key'=>''];
echo '<!doctype html><title>Opening AlmaDial phone</title><form method="post" action="viciphone.php">';
foreach ($fields as $name=>$value) {echo '<input type="hidden" name="'.htmlspecialchars($name,ENT_QUOTES,'UTF-8').'" value="'.htmlspecialchars(base64_encode($value),ENT_QUOTES,'UTF-8').'">';}
echo '<button type="submit">Connect phone</button></form>';
?>
'''.encode()

def inspect(payload):
    gid=grp.getgrnam('asterisk').gr_gid
    wrapper=read(WRAPPER,0o640,gid)
    if wrapper not in (BASE,WRAPPED): raise ValueError('foreign PJSIP wrapper')
    foreign=digest([rows('phones',"extension<>"+literal(IDENTITY)+" OR extension IS NULL"),rows('vicidial_settings_containers',"container_id<>"+literal(CONTAINER))])
    cols,phones=rows('phones','extension='+literal(IDENTITY))
    _,containers=rows('vicidial_settings_containers','container_id='+literal(CONTAINER))
    secret=None
    if SECRET.exists():
        secret=read(SECRET,0o600).decode().strip()
        if not re.fullmatch('[0-9a-f]{32}',secret): raise ValueError('invalid protected phone credential')
    files={k:base64.b64decode(v,validate=True) for k,v in payload['Files'].items()};files['index.php']=LAUNCHER
    if any('/' in part and ('..' in part.split('/')) for part in files): raise ValueError('unsafe asset path')
    state='absent';manifest={}
    if ROOT.exists():
        safe(ROOT)
        actual=sorted(str(f.relative_to(ROOT)) for f in ROOT.rglob('*') if not f.is_dir())
        if actual!=sorted(files): raise ValueError('foreign phone assets')
        for name,data in files.items():
            if read(ROOT/name,0o644)!=data: raise ValueError('phone asset drift')
            manifest[name]=sha(data)
        state=digest(manifest)
    pjsip=None
    if PJSIP.exists(): pjsip=read(PJSIP,0o640,gid)
    if not phones and not containers and secret is None and state=='absent' and pjsip is None and wrapper==BASE:
        current=digest([foreign,'absent'])
        return dict(Host='',Foundation='',Foreign=foreign,Files='absent',Services='absent',Health='absent',Current=current)
    if not secret or len(phones)!=1 or len(containers)!=1 or state=='absent' or pjsip!=config(secret) or wrapper!=WRAPPED:
        raise ValueError('partial S10 state requires explicit reconciliation')
    row=json.loads(phones[0]);values={k:None if v is None else bytes.fromhex(v).decode() for k,v in zip(cols,row)}
    if any(values[k]!=v for k,v in desired_values(secret).items()): raise ValueError('synthetic phone row mismatch')
    if sql('SELECT container_entry FROM vicidial_settings_containers WHERE container_id='+literal(CONTAINER))!=SETTINGS: raise ValueError('settings container drift')
    receipt=json.loads(read(WORK/'receipt.json',0o600))
    expected=dict(Source=payload['Source'],Phone=digest(phones),Container=digest(containers),Assets=state,PJSIP=sha(pjsip),Credential=sha(secret.encode()))
    if receipt!=expected: raise ValueError('S10 receipt mismatch')
    endpoint=run(['/usr/sbin/asterisk','-rx','pjsip show endpoint '+IDENTITY])
    transport=run(['/usr/sbin/asterisk','-rx','pjsip show transport almadial-s10-wss'])
    healthy=('Endpoint:  '+IDENTITY in endpoint or IDENTITY in endpoint) and 'almadial-s10-wss' in transport and 'wss' in transport and '127.0.0.1' in transport
    if 'Unable to find object' in endpoint or 'Unable to find object' in transport: healthy=False
    current=digest(expected)
    return dict(Host='',Foundation='',Foreign=foreign,Files=digest(expected),Services=digest(['pjsip','loaded']) if healthy else 'partial',Health=digest(['registration-ready',current]) if healthy else 'partial',Current=current)

def activate(p):
    before=inspect(p)
    if before['Current']!=p['ExpectedCurrent'] or before['Health'] not in ('absent','partial'): raise ValueError('stale S10 activation')
    if before['Health']=='partial':
        run(['/usr/sbin/asterisk','-rx','module reload res_pjsip.so'])
        out=inspect(p)
        if out['Health']=='partial': raise ValueError('PJSIP registration configuration not loaded')
        return out
    gid=grp.getgrnam('asterisk').gr_gid
    for path in (ROOT,WORK):
        safe(path);path.mkdir(mode=0o755 if path==ROOT else 0o700)
        os.chmod(path,0o755 if path==ROOT else 0o700)
    secret=secrets.token_hex(16)
    fd=os.open(SECRET,os.O_CREAT|os.O_EXCL|os.O_WRONLY|os.O_NOFOLLOW,0o600)
    with os.fdopen(fd,'w') as f: f.write(secret+'\n');f.flush();os.fsync(f.fileno())
    run(['/usr/sbin/restorecon',str(SECRET)])
    files={k:base64.b64decode(v,validate=True) for k,v in p['Files'].items()};files['index.php']=LAUNCHER
    for name,data in sorted(files.items()):
        target=ROOT/name
        target.parent.mkdir(parents=True,exist_ok=True,mode=0o755)
        # All descendants belong to this newly created deployment. mkdir is
        # filtered by umask; Apache still needs traversal under operator 077.
        for parent in [target.parent,*target.parent.parents]:
            if parent==ROOT: break
            os.chmod(parent,0o755)
        atomic(target,data,0o644)
    for target in (ROOT/'index.php',ROOT/'viciphone.php'): run(['/usr/bin/php','-l',str(target)])
    run(['/usr/sbin/restorecon','-R',str(ROOT)])
    vals=desired_values(secret)
    columns=','.join('`'+x+'`' for x in vals)
    query='START TRANSACTION; INSERT INTO phones ('+columns+') VALUES ('+','.join(literal(v) for v in vals.values())+');'
    query+='INSERT INTO vicidial_settings_containers (container_id,container_type,container_entry) VALUES ('+literal(CONTAINER)+",'WEBPHONE_SETTINGS',"+literal(SETTINGS)+'); COMMIT;'
    sql(query)
    atomic(WORK/'pjsip.conf.before',BASE,0o600)
    atomic(PJSIP,config(secret),0o640,gid)
    atomic(WRAPPER,WRAPPED,0o640,gid)
    _,phone=rows('phones','extension='+literal(IDENTITY))
    _,container=rows('vicidial_settings_containers','container_id='+literal(CONTAINER))
    receipt=dict(Source=p['Source'],Phone=digest(phone),Container=digest(container),Assets=digest({k:sha(v) for k,v in files.items()}),PJSIP=sha(config(secret)),Credential=sha(secret.encode()))
    atomic(WORK/'receipt.json',json.dumps(receipt,sort_keys=True).encode(),0o600)
    run(['/usr/sbin/asterisk','-rx','module reload res_pjsip.so'])
    out=inspect(p)
    if out['Health'] in ('absent','partial'): raise ValueError('PJSIP registration configuration not loaded')
    return out

def main():
    try:
        payload=json.load(sys.stdin)
        if payload['Request']!={'Identity':IDENTITY,'Hostname':'localhost','WSSURL':'wss://localhost:8089/ws'}: raise ValueError('unsupported phone topology')
        if run(['/usr/sbin/getenforce'])!='Enforcing': raise ValueError('SELinux not enforcing')
        result=inspect(payload) if sys.argv[1]=='phone-inspect' else activate(payload) if sys.argv[1]=='phone-activate' else None
        if result is None: raise ValueError('unknown S10 operation')
        print(json.dumps(result));return 0
    except (ValueError,OSError,subprocess.SubprocessError) as e:
        print('S10: '+str(e),file=sys.stderr);return 1

if __name__=='__main__': sys.exit(main())
