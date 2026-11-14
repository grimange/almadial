"""Remote S06 ownership handoff after verified S01-S10 provisioning.

Native upstream workers retain dialing/queue ownership. This adapter only
publishes pinned derivations, units and maintenance schedules. The ordinary
application database is mutable; immutable source/configuration is still checked.
"""
import gzip
import pwd
import socket
import time

NR_HOME = Path('/var/lib/almadial/native-runtime')
NR_SECRET = Path('/etc/astguiclient-native-ami.secret')
NR_CRON = Path('/etc/cron.d/almadial-native')
NR_UNITS = {'updater':'AST_update.pl', 'listener':'AST_manager_listen.pl',
            'sender':'AST_manager_send.pl', 'autodial':'AST_VDauto_dial.pl'}
NR_MAINTENANCE = [
 ('*/3 * * * *','AST_CRON_audio_1_move_VDonly.pl','--HTTPS'),
 ('* * * * *','AST_manager_kill_hung_congested.pl',''),
 ('* * * * *','AST_vm_update.pl',''),
 ('* * * * *','AST_conf_update.pl',''),
 ('11 * * * *','AST_flush_DBqueue.pl','-q'),
 ('33 * * * *','AST_cleanup_agent_log.pl',''),
 ('50 0 * * *','AST_cleanup_agent_log.pl','--last-24hours'),
 ('25 0 * * *','AST_DB_dead_cb_purge.pl','--purge-non-cb --quiet'),
 ('* * * * *','AST_VDhopper.pl','-q'),
 ('1 1,7 * * *','ADMIN_adjust_GMTnow_on_leads.pl','--list-settings'),
 ('2 1 * * *','AST_reset_mysql_vars.pl',''),
 ('3 1 * * *','AST_DB_optimize.pl',''),
 ('2 0 * * 0','AST_agent_week.pl','')]
# Asterisk mixes the two directions natively. The upstream VD-only archiver
# retains lossless WAV; optional lossy compression/FTP is deliberately disabled.
# Distribution/FTP, QueueMetrics, inbound email and destructive recording/log
# retention are optional upstream examples, not prerequisites for manual dialing.
# Upstream keepalive retains maintenance/configuration generation. Its X option
# disables daemon supervision, which is owned exclusively by systemd.

def native_cron():
    header = '# AlmaDial remote native VICIdial maintenance; systemd owns long-running workers.\nSHELL=/bin/bash\nPATH=/usr/sbin:/usr/bin:/sbin:/bin\nCRON_TZ=UTC\nMAILTO=""\n'
    return (header + '* * * * * root /usr/bin/systemctl start almadial-native-maintenance.service\n' + ''.join(schedule+' asterisk /usr/bin/flock -n /run/asterisk/almadial-'+name+'.lock /usr/bin/perl /usr/share/astguiclient/'+name+(' '+args if args else '')+'\n' for schedule,name,args in NR_MAINTENANCE)).encode()

def native_unit(name, script):
    return ('''# AlmaDial native VICIdial runtime; no competing ADMIN_keepalive.
[Unit]
Description=VICIdial native '''+name+'''
Requires=mariadb.service asterisk.service almadial-s06-fastagi.service
After=mariadb.service asterisk.service almadial-s06-fastagi.service
StartLimitIntervalSec=60
StartLimitBurst=5
[Service]
Type=simple
User=asterisk
Group=asterisk
WorkingDirectory=/usr/share/astguiclient
Environment=PATH=/usr/sbin:/usr/bin:/sbin:/bin
ExecStartPre=/usr/bin/perl /usr/share/astguiclient/almadial-s06-ready.pl
ExecStart=/usr/bin/perl /usr/share/astguiclient/'''+script+'''
Restart=always
RestartSec=5
KillMode=control-group
TimeoutStopSec=30
NoNewPrivileges=true
PrivateTmp=true
UMask=0027
[Install]
WantedBy=multi-user.target
''').encode()

def nr_protected(path, mode=0o600, group=0):
    return read(path,mode,group)

def nr_source(p, name):
    compressed=base64.b64decode(p['Sources'][name],validate=True)
    with gzip.GzipFile(fileobj=__import__('io').BytesIO(compressed)) as source:
        raw=source.read((4<<20)+1)
    if len(raw)>4<<20: raise ValueError('native pinned input exceeds source bound')
    if sha(raw)!=p['SourcePins'][name]: raise ValueError('native pinned source mismatch: '+name)
    return raw

def nr_derivations(p):
    outputs={}
    def add(name,raw):
        path=Path('/var/www/html')/name[4:] if name.startswith('www/') else Path('/usr/share/astguiclient')/name[4:] if name.startswith('bin/') else Path('/var/lib/asterisk/agi-bin')/name[4:]
        outputs[str(path)]=raw
    for name in ['www/agc/vicidial.php','www/agc/vdc_db_query.php','www/agc/conf_exten_check.php','www/agc/manager_send.php','www/agc/calendar_db.js']:
        raw=nr_source(p,name)
        if name.endswith('/vicidial.php'): raw=agent_lifecycle_overlay(raw)
        elif name.endswith('/calendar_db.js'): raw=calendar_overlay(raw)
        else: raw=session_overlay(Path(name).name,raw)
        add(name,raw)
    for script in set(NR_UNITS.values())|{x[1] for x in NR_MAINTENANCE}|{'AST_send_action_child.pl','AST_CRON_audio_1_move_mix.pl','AST_CRON_audio_2_compress.pl'}:
        name='bin/'+script;raw=nr_source(p,name)
        if script=='AST_update.pl': raw=updater_overlay(raw)
        elif script=='AST_manager_listen.pl': raw=call_runtime_overlay(script,raw)
        elif script in MANAGER_PINS: raw=manager_overlay(script,raw)
        if script in ['AST_VDauto_dial.pl','AST_manager_send.pl']:
            marker=b'/bin/ps -o "%p %a" --no-headers -A'
            if raw.count(marker)!=1: raise ValueError('native procps source marker')
            raw=raw.replace(marker,b'/bin/ps -o pid,args -ww --no-headers -A')
        raw=raw.replace(b'/etc/astguiclient-m007-updater.conf',b'/etc/astguiclient-asterisk.conf').replace(b'/etc/astguiclient.conf',b'/etc/astguiclient-asterisk.conf').replace(b'/etc/almadial-m007-ami.secret',bytes(str(NR_SECRET),'ascii'))
        add(name,raw)
    raw=nr_source(p,'bin/ADMIN_keepalive_ALL.pl')
    text=raw.decode()
    text=replace_once(text,"'/etc/astguiclient.conf'","'/etc/astguiclient-asterisk.conf'")
    text=replace_once(text,'# Customized Variables',"# Only native maintenance; systemd owns daemon supervision.\n$VARactive_keepalives='X';\numask 0027;\n# Customized Variables")
    # S10 owns its explicit endpoint; avoid a second wizard endpoint when the
    # native generator rebuilds ordinary administrator-created phones.
    marker="and protocol='PJSIP' and active='Y' order by extension;"
    text=replace_once(text,marker,"and protocol='PJSIP' and active='Y' and extension!='alm_s10_9061' order by extension;")
    # Same reviewed GSM priority correction as the accepted S05 generator.
    a, marker, b=text.partition('$Vext .= "exten => 8310,1,Answer\\n";')
    if not marker or b.count('exten => 8309,2,MixMonitor(')!=1: raise ValueError('native GSM generation marker')
    text=a+marker+b.replace('exten => 8309,2,MixMonitor(','exten => 8310,2,MixMonitor(',1)
    # Native mixed WAV is the input expected by upstream VD-only archival.
    # Avoid scheduling a mixer/compressor whose optional codecs are not installed.
    mix=r'MixMonitor(,r($PATHmonitor/\${CALLERID(name)}-in.wav)t($PATHmonitor/\${CALLERID(name)}-out.wav))'
    if text.count(mix)!=2: raise ValueError('native mixed recording markers')
    text=text.replace(mix,r'MixMonitor($PATHmonitor/\${CALLERID(name)}-in.wav)')
    reloads={'extensions reload':'dialplan reload','dialplan reload':'dialplan reload','sip reload':None,'iax2 reload':None,'reload app_voicemail.so':'module reload app_voicemail.so','module reload app_voicemail.so':'module reload app_voicemail.so','moh reload':'moh reload','reload app_meetme.so':None,'module reload app_meetme.so':None,'pjsip reload':'module reload res_pjsip.so','reload app_confbridge.so':'module reload app_confbridge.so','/usr/sbin/asterisk -vvvvgcT':'core waitfullybooted'}
    pattern=r"`screen -XS asterisk eval 'stuff \"([^\"]+)\\015\"'`"
    found=re.findall(pattern,text)
    if len(found)!=16 or any(x not in reloads for x in found): raise ValueError('native maintenance reload markers')
    def native_reload(match):
        command=reloads[match[1]]
        if command is None: return '`/usr/bin/true`'
        commands=[command]
        if match[1]=='pjsip reload': commands.append('module reload res_pjsip_config_wizard.so')
        return '`'+'; '.join('/usr/sbin/asterisk -rx "'+x+'"' for x in commands)+'`'
    text=re.sub(pattern,native_reload,text)
    outputs['/usr/share/astguiclient/ADMIN_keepalive_ALL.pl']=text.encode()
    # Pinned source modules are already installed by S05. Make the AGI lookup
    # explicit instead of relying on the interpreter's compiled @INC directory.
    raw=nr_source(p,'agi/park_CID.agi')
    raw=call_runtime_overlay('park_CID.agi',raw).replace(b'/etc/astguiclient-m007-updater.conf',b'/etc/astguiclient-asterisk.conf').replace(b'/opt/almadial-m007',b'/usr/local/share/perl5/5.40')
    add('agi/park_CID.agi',raw)
    return outputs

NR_DIALPLAN=b''' ; AlmaDial native single-server agent contexts. No external trunk routes.
[default]
include => vicidial-auto-internal
include => vicidial-auto-phones
include => vicidial-auto-external
include => almadial-native-agents
[almadial-native-agents]
exten => _9600XXX,1,Answer()
 same => n,ConfBridge(${EXTEN},vici_agent_bridge,vici_customer_user)
 same => n,Hangup()
exten => _29600XXX,1,Answer()
 same => n,ConfBridge(${EXTEN:1},vici_agent_bridge,vici_agent_user)
 same => n,Hangup()
exten => _39600XXX,1,Answer()
 same => n,ConfBridge(${EXTEN:1},vici_agent_bridge,vici_admin_user)
 same => n,Hangup()
exten => _59600XXX,1,Answer()
 same => n,ConfBridge(${EXTEN:1},vici_agent_bridge,vici_recording_user)
 same => n,Hangup()
exten => 8301,1,Answer()
 same => n,AGI(park_CID.agi)
 same => n,Hangup()
exten => 8303,1,Answer()
 same => n,AGI(park_CID.agi)
 same => n,Hangup()
exten => h,1,AGI(agi://127.0.0.1:4577/call_log--HVcauses--PRI-----NODEBUG-----${HANGUPCAUSE})
'''

def nr_secret(create=False):
    group=grp.getgrnam('asterisk').gr_gid
    if not NR_SECRET.exists():
        if not create: return None
        atomic(NR_SECRET,(secrets.token_hex(10)+'\n').encode(),0o640,group)
    value=nr_protected(NR_SECRET,0o640,group).decode().strip()
    if not re.fullmatch('[0-9a-f]{20}',value): raise ValueError('native AMI secret metadata or identity')
    # Pinned schema ASTmgrSECRET is varchar(20); 20 hex characters are 80 bits.
    return value[:20]

def nr_desired(p, secret):
    files=nr_derivations(p)
    for key,script in NR_UNITS.items(): files['/etc/systemd/system/almadial-native-'+key+'.service']=native_unit(key,script)
    files[str(NR_CRON)]=native_cron()
    files['/etc/systemd/system/almadial-native-maintenance.service']=b'''[Unit]
Description=VICIdial native maintenance and configuration generation
Requires=mariadb.service asterisk.service
After=mariadb.service asterisk.service
[Service]
Type=oneshot
User=root
Group=asterisk
UMask=0027
ExecStart=/usr/bin/perl /usr/share/astguiclient/ADMIN_keepalive_ALL.pl
TimeoutStartSec=55
'''
    files['/etc/asterisk/almadial-native.conf']=NR_DIALPLAN
    files['/etc/asterisk/pjsip_wizard.conf']=b'#include "pjsip_wizard-vicidial.conf"\n'
    if secret:
        files['/etc/asterisk/almadial-native-manager.conf']=('[almadial-native]\nsecret='+secret+'\ndeny=0.0.0.0/0.0.0.0\npermit=127.0.0.1/255.255.255.255\nread=system,call,agent\nwrite=call,originate,command\n').encode()
    return files

def nr_meta(path):
    st=path.lstat()
    if stat.S_ISLNK(st.st_mode):
        return {'link':os.readlink(path),'uid':st.st_uid,'gid':st.st_gid,'mode':stat.S_IMODE(st.st_mode)}
    if not stat.S_ISREG(st.st_mode) or st.st_nlink!=1 or st.st_size>64<<20: raise ValueError('native baseline file type/size: '+str(path))
    return {'sha256':sha(path.read_bytes()),'uid':st.st_uid,'gid':st.st_gid,'mode':stat.S_IMODE(st.st_mode),'label':os.getxattr(path,'security.selinux').decode().strip('\x00')}

def nr_baseline(p):
    # All immutable application source, installed Asterisk and installer-owned
    # config is frozen only AFTER successful fresh recursive S01-S10 verification.
    # Native generator outputs and bridge/queue/session DB rows are application
    # owned after this explicit handoff, not immutable installation snapshots.
    roots=['/etc/asterisk','/etc/httpd','/etc/php-fpm.d','/etc/my.cnf.d','/etc/firewalld','/etc/pki/tls/almadial-s08','/etc/cron.d','/var/spool/cron','/usr/share/astguiclient','/var/www/html','/usr/lib/asterisk/modules','/usr/local/share/perl5/5.40','/var/lib/asterisk/agi-bin','/var/lib/asterisk/sounds']
    paths=set()
    for root in roots:
        for directory,dirs,files in os.walk(root,followlinks=False):
            paths.update(Path(directory)/name for name in files)
            paths.update(Path(directory)/name for name in dirs if (Path(directory)/name).is_symlink())
    paths.update(Path(x) for x in ['/etc/astguiclient.conf','/etc/astguiclient-asterisk.conf','/etc/astguiclient-s06.conf','/usr/sbin/asterisk','/usr/lib/libasteriskssl.so.1','/usr/lib/libasteriskpj.so.2','/etc/systemd/system/asterisk.service','/etc/cron.d/almadial-s06'])
    paths.update(Path('/etc/systemd/system/almadial-s06-'+x+'.service') for x in ['fastagi','updater','listener'])
    owned=set(nr_desired(p,None))|{'/etc/asterisk/almadial-native-manager.conf','/usr/lib/asterisk/modules/app_dial.so'}
    # Wrapper preimages are retained and exact include transitions checked on
    # every inspection, along with the supplemental module's build provenance.
    return {str(path):nr_meta(path) for path in sorted(paths) if str(path) not in owned and '.alm46-native-before' not in str(path)}

def nr_load_baseline(p):
    path=NR_HOME/'baseline.json'
    if not path.exists(): return None
    value=json.loads(nr_protected(path))
    if value['Binding']!=p['Binding']: raise ValueError('native handoff identity changed; preserve journal')
    return value

def nr_original(path, baseline):
    raw=path.read_bytes()
    key=str(path)
    if key not in baseline: raise ValueError('missing native preimage')
    old=baseline[key]
    if sha(raw)==old.get('sha256'): return raw
    backup=nr_backup(path)
    if not backup.exists(): backup=Path(str(path)+'.alm46-native-before')
    before=nr_protected(backup)
    if sha(before)!=old.get('sha256'): raise ValueError('native preimage checksum conflict: '+key)
    return before

def nr_wrappers(p,baseline,secret):
    group=grp.getgrnam('asterisk').gr_gid
    result={}
    for name,include in [('manager.conf','almadial-native-manager.conf'),('extensions.conf','almadial-native.conf')]:
        path=Path('/etc/asterisk')/name
        before=nr_original(path,baseline)
        result[str(path)]=before+('#include "'+include+'"\n').encode()
    path=Path('/etc/asterisk/modules.conf'); before=nr_original(path,baseline)
    result[str(path)]=before+b'[modules](+)\nload = app_dial.so\n'
    path=Path('/etc/asterisk/almadial-s10-phone.conf');before=nr_original(path,baseline)
    if before.count(b'context=alm-remote-synthetic')!=1: raise ValueError('unexpected pre-runtime phone context')
    result[str(path)]=before.replace(b'context=alm-remote-synthetic',b'context=default')
    return result

def nr_verify_baseline(p,baseline,wrappers):
    # Dynamic upstream generated configuration is checked via syntax, current
    # AMI/dialplan, DB registration and application health rather than frozen DB
    # payloads; every other original file retains its exact metadata and digest.
    generated={'/etc/asterisk/'+n for n in ['extensions-vicidial.conf','pjsip-vicidial.conf','pjsip_wizard-vicidial.conf','confbridge-vicidial.conf','voicemail.conf','sip-vicidial.conf','iax-vicidial.conf','musiconhold-vicidial.conf','meetme-vicidial.conf','BUILDvoicemail-vicidial.conf']}
    transient={'/etc/asterisk/BUILD'+n.split('/')[-1] for n in generated if '/BUILD' not in n}
    current=nr_baseline(p)
    owned=set(nr_desired(p,None))|{'/etc/asterisk/almadial-native-manager.conf','/usr/lib/asterisk/modules/app_dial.so'}
    if set(current)-generated-transient!=set(baseline)-generated-transient-owned: raise ValueError('native immutable file inventory changed')
    for path in generated|transient:
        if path in current:
            m=current[path]
            if 'link' in m or m['uid'] not in [0,pwd.getpwnam('asterisk').pw_uid] or m['gid'] not in [0,grp.getgrnam('asterisk').gr_gid] or m['mode'] not in [0o600,0o640,0o644] or m['label'].split(':')[2]!='etc_t': raise ValueError('native generated metadata drift: '+path)
    for path,expected in baseline.items():
        if path in wrappers or path in generated or path in transient or path in owned: continue
        if current[path]!=expected: raise ValueError('native foundation drift: '+path)

def nr_backup(path):
    return NR_HOME/('preimage-'+sha(str(path).encode()))

def nr_atomic(path,raw,mode,gid):
    # Walk no-follow directory descriptors. Only the two accepted Asterisk-owned
    # directories may be non-root; no symlink or world/group writable ancestor.
    fd=os.open('/',os.O_RDONLY|os.O_DIRECTORY)
    try:
        current=Path('/')
        for part in path.parts[1:-1]:
            current=current/part
            child=os.open(part,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=fd)
            os.close(fd);fd=child;st=os.fstat(fd)
            allowed=pwd.getpwnam('asterisk').pw_uid if str(current) in ['/var/lib/asterisk','/var/lib/asterisk/agi-bin'] else 0
            if st.st_uid!=allowed or st.st_mode&0o022: raise ValueError('native unsafe publication directory')
        name='.almadial-native-'+secrets.token_hex(12)
        output=os.open(name,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,mode,dir_fd=fd)
        try:
            os.fchmod(output,mode);os.fchown(output,0,gid)
            with os.fdopen(output,'wb') as stream: stream.write(raw);stream.flush();os.fsync(stream.fileno())
            os.replace(name,path.name,src_dir_fd=fd,dst_dir_fd=fd);os.fsync(fd)
        finally:
            try: os.unlink(name,dir_fd=fd)
            except FileNotFoundError: pass
    finally: os.close(fd)
    run(['/usr/sbin/restorecon',str(path)])

def nr_publish(path,raw,expected=None):
    path=Path(path)
    if path.exists():
        meta=nr_meta(path)
        if 'link' in meta or meta['uid']!=0: raise ValueError('native publication file ownership conflict')
        before=path.read_bytes()
        if before==raw: return
        if expected is None or sha(before)!=expected:
            raise ValueError('native conflicting file: '+str(path))
        backup=nr_backup(path)
        if not backup.exists(): atomic(backup,before,0o600)
        elif nr_protected(backup)!=before: raise ValueError('native backup conflict')
        nr_atomic(path,raw,meta['mode'],meta['gid'])
    else:
        group=grp.getgrnam('asterisk').gr_gid if str(path).startswith('/etc/asterisk/') else 0
        nr_atomic(path,raw,0o640 if group else 0o644,group)

def nr_phone_template(p):
    n=p['Network']
    lines=['transport=almadial-s10-wss','context=default','webrtc=yes','direct_media=no','rtp_symmetric=yes','force_rport=yes','rewrite_contact=yes','media_address='+n['ManagementAddress'],'bind_rtp_to_media_address=yes','rtp_port_start='+str(n['RTPStart']),'rtp_port_end='+str(n['RTPEnd']),'ice_support=yes','dtls_verify=fingerprint','dtls_setup=actpass','dtls_cert_file=/etc/pki/tls/almadial-s08/current/fullchain.pem','dtls_private_key=/etc/pki/tls/almadial-s08/current/key.pem']
    return '\n'.join('endpoint/'+x for x in lines)+'\naor/max_contacts=1\naor/remove_existing=yes\n'

def nr_database(secret,p,mutate=False):
    template=nr_phone_template(p)
    present=sql("SELECT template_contents FROM vicidial_conf_templates WHERE template_id='ALMADIAL_WEBRTC';")
    if present and present!=template.strip(): raise ValueError('native browser phone template ownership conflict')
    if mutate:
        if not present: sql("INSERT INTO vicidial_conf_templates(template_id,template_name,template_contents) VALUES('ALMADIAL_WEBRTC','AlmaDial remote WebRTC',"+literal(template)+");")
        backup=NR_HOME/'database-preimage.json'
        if not backup.exists():
            before=sql("SELECT JSON_OBJECT('server_id',server_id,'ASTmgrUSERNAME',ASTmgrUSERNAME,'ASTmgrUSERNAMEupdate',ASTmgrUSERNAMEupdate,'ASTmgrUSERNAMElisten',ASTmgrUSERNAMElisten,'ASTmgrUSERNAMEsend',ASTmgrUSERNAMEsend,'ASTmgrSECRET',ASTmgrSECRET,'ext_context',ext_context,'web_socket_url',web_socket_url,'vd_server_logs',vd_server_logs) FROM servers WHERE server_id='TESTast' AND server_ip='127.0.0.1'; SELECT JSON_OBJECT('extension',extension,'ext_context',ext_context,'phone_context',phone_context,'webphone_dialpad',webphone_dialpad) FROM phones WHERE extension='alm_s10_9061'; SELECT JSON_OBJECT('webphone_url',webphone_url) FROM system_settings;")
            atomic(backup,json.dumps([json.loads(x) for x in before.splitlines()]).encode(),0o600)
        roles=sql("SELECT CONCAT_WS(',',ASTmgrUSERNAME,ASTmgrUSERNAMEupdate,ASTmgrUSERNAMElisten,ASTmgrUSERNAMEsend) FROM servers WHERE server_id='TESTast' AND server_ip='127.0.0.1';")
        if roles not in [','.join([name]*4) for name in ['almadial-s05','almadial-native']]: raise ValueError('native AMI role ownership conflict')
        if sql("SELECT COUNT(*) FROM phones WHERE extension='alm_s10_9061'; SELECT COUNT(*) FROM system_settings;").splitlines()!=['1','1']: raise ValueError('native database ownership/cardinality conflict')
        sql("UPDATE servers SET ASTmgrUSERNAME='almadial-native',ASTmgrUSERNAMEupdate='almadial-native',ASTmgrUSERNAMElisten='almadial-native',ASTmgrUSERNAMEsend='almadial-native',ASTmgrSECRET="+literal(secret)+",ext_context='default',web_socket_url="+literal(p['WSSURL'])+",vd_server_logs='N',rebuild_conf_files='Y' WHERE server_id='TESTast' AND server_ip='127.0.0.1';")
        sql("UPDATE phones SET ext_context='default',phone_context='default',webphone_dialpad='Y' WHERE extension='alm_s10_9061' AND server_ip='127.0.0.1';")
        sql('UPDATE system_settings SET webphone_url='+literal('https://'+p['Hostname']+'/almadial-phone/viciphone.php')+';')
    result=sql("SELECT IF(COUNT(*)=1,1,0) FROM servers WHERE server_id='TESTast' AND server_ip='127.0.0.1' AND active='Y' AND active_asterisk_server='Y' AND conf_engine='CONFBRIDGE' AND asterisk_version='22.12.0' AND ASTmgrUSERNAME='almadial-native' AND ASTmgrUSERNAMEupdate='almadial-native' AND ASTmgrUSERNAMElisten='almadial-native' AND ASTmgrUSERNAMEsend='almadial-native' AND SHA2(ASTmgrSECRET,256)="+literal(sha(secret.encode()))+"; SELECT db_schema_version FROM system_settings; SELECT COUNT(*) FROM phones WHERE extension='alm_s10_9061' AND ext_context='default' AND phone_context='default' AND protocol='PJSIP';")
    return result.splitlines()==['1','1748','1'] and bool(present)

def nr_ami(secret):
    # Local authenticated protocol probe; neither a CLI exit nor an enabled unit
    # alone proves readiness. Events off avoids consuming listener events.
    with socket.create_connection(('127.0.0.1',5038),timeout=5) as connection:
        connection.settimeout(5);stream=connection.makefile('rb')
        if not stream.readline(4097).startswith(b'Asterisk Call Manager/'): return False
        for action,extra in [('Login','Username: almadial-native\r\nSecret: '+secret+'\r\nEvents: off\r\n'),('Ping',''),('Command','Command: core show channels count\r\n'),('Logoff','')]:
            connection.sendall(('Action: '+action+'\r\n'+extra+'\r\n').encode());data=b''
            for _ in range(128):
                line=stream.readline(4097)
                if not line or len(line)>4096: return False
                data+=line
                if line==b'\r\n': break
            if b'Response: Success' not in data and not (action=='Command' and b'Response: Follows' in data) and not (action=='Logoff' and b'Response: Goodbye' in data): return False
        return True

def nr_health(p,secret):
    if not nr_database(secret,p) or not nr_ami(secret): return False
    # A root AMI probe cannot establish that the native child can read its input.
    try: run(['/usr/sbin/runuser','-u','asterisk','--','/usr/bin/test','-r',str(NR_SECRET)])
    except ValueError: return False
    if run(['/usr/sbin/getenforce'])!='Enforcing': raise ValueError('native runtime requires SELinux Enforcing')
    for module in ['app_dial','app_confbridge','res_agi','app_mixmonitor']:
        if not re.search(r'^'+module+r'\.so\s+.*Running',run(['/usr/sbin/asterisk','-rx','module show like '+module+'.so']),re.M): return False
    if 'almadial-native-agents' not in run(['/usr/sbin/asterisk','-rx','dialplan show default']): return False
    if 'ConfBridge(' not in run(['/usr/sbin/asterisk','-rx','dialplan show 29600000@default']): return False
    if 'MixMonitor(/var/spool/asterisk/monitor/${CALLERID(name)}-in.wav)' not in run(['/usr/sbin/asterisk','-rx','dialplan show 8309@vicidial-auto-internal']): return False
    heartbeat=sql("SELECT IF(COUNT(*)=1,1,0) FROM server_updater WHERE server_ip='127.0.0.1' AND last_update>=DATE_SUB(NOW(),INTERVAL 30 SECOND);")
    return heartbeat=='1'

def nr_services():
    for name,script in NR_UNITS.items():
        try: raw=run(['/usr/bin/systemctl','show','almadial-native-'+name+'.service','--property=ActiveState,UnitFileState,MainPID,User,Group,DropInPaths,NeedDaemonReload'])
        except ValueError: return False
        values=dict(line.split('=',1) for line in raw.splitlines() if '=' in line)
        if values.get('DropInPaths'): raise ValueError('native runtime unit override conflict')
        if any(values.get(k)!=v for k,v in {'ActiveState':'active','UnitFileState':'enabled','User':'asterisk','Group':'asterisk','NeedDaemonReload':'no'}.items()): return False
        pid=values['MainPID']
        try:
            argv=Path('/proc/'+pid+'/cmdline').read_bytes().rstrip(b'\0').split(b'\0')
            if argv!=[b'/usr/bin/perl',('/usr/share/astguiclient/'+script).encode()]: return False
            if Path('/proc/'+pid+'/cgroup').read_text().strip()!='0::/system.slice/almadial-native-'+name+'.service': return False
            status=Path('/proc/'+pid+'/status').read_text()
            uidline=next(x for x in status.splitlines() if x.startswith('Uid:'))
            if set(uidline.split()[1:])!={str(pwd.getpwnam('asterisk').pw_uid)}: return False
        except (FileNotFoundError,ProcessLookupError): return False
    processes=run(['/usr/bin/ps','-eo','unit=,user=,args=']).splitlines()
    fast=[x.split() for x in processes if x.split()[2:4]==['/usr/bin/perl','/usr/share/astguiclient/almadial-s06-FastAGI_log.pl']]
    if len(fast)!=6 or any(x!=['almadial-s06-fastagi.service','asterisk','/usr/bin/perl','/usr/share/astguiclient/almadial-s06-FastAGI_log.pl'] for x in fast): return False
    if any(x.split()[2:4]==['/usr/bin/perl','/usr/share/astguiclient/ADMIN_keepalive_ALL.pl'] and x.split()[0]!='almadial-native-maintenance.service' for x in processes): raise ValueError('competing native supervisor')
    return run(['/usr/bin/systemctl','is-active','crond'])=='active'

def nr_expected_meta(p,path):
    deployment=p.get('DeploymentMetadata',{}).get(path)
    if deployment:
        return dict(uid=0,gid=grp.getgrnam(deployment['Group']).gr_gid,mode=deployment['Mode'],label=deployment['Label'])
    group=grp.getgrnam('asterisk').gr_gid if path.startswith('/etc/asterisk/') else 0
    label='etc_t' if path.startswith('/etc/asterisk/') else 'systemd_unit_file_t' if path.startswith('/etc/systemd/') else 'system_cron_spool_t' if path==str(NR_CRON) else 'lib_t'
    return dict(uid=0,gid=group,mode=0o640 if group else 0o644,label=label)

def nr_matches_meta(p,path):
    meta=nr_meta(Path(path));want=nr_expected_meta(p,path)
    return all(meta.get(k)==v for k,v in want.items() if k!='label') and meta.get('label','').split(':')[2]==want['label']

def nr_state(p):
    value=nr_load_baseline(p)
    baseline=value['Files'] if value else nr_baseline(p)
    foreign=digest(baseline)
    secret=nr_secret();desired=nr_desired(p,secret)
    wrappers=nr_wrappers(p,baseline,secret)
    nr_verify_baseline(p,baseline,wrappers)
    missing=not value or not secret or not Path('/etc/asterisk/voicemail.conf').exists()
    for path,raw in {**desired,**wrappers}.items():
        if not Path(path).exists() or Path(path).read_bytes()!=raw: missing=True
        elif path in wrappers:
            meta=nr_meta(Path(path));before=baseline[path]
            if any(meta.get(k)!=before.get(k) for k in ['uid','gid','mode']) or meta.get('label','').split(':')[1:]!=before.get('label','').split(':')[1:]: raise ValueError('native wrapper metadata drift: '+path)
        elif not nr_matches_meta(p,path): raise ValueError('native owned metadata drift: '+path)
    module=Path('/usr/lib/asterisk/modules/app_dial.so')
    build=Path('/var/lib/almadial/asterisk-s04/asterisk-22.12.0/apps/app_dial.so')
    module_sha=value['ModuleSHA256'] if value else sha(build.read_bytes())
    if not module.exists() or sha(module.read_bytes())!=module_sha: missing=True
    elif not nr_matches_meta(p,str(module)): raise ValueError('native module metadata drift')
    if secret and not nr_database(secret,p): missing=True
    files=digest({k:sha(v) for k,v in {**desired,**wrappers}.items()}|{'app_dial.so':module_sha}) if not missing else 'absent'
    services=nr_services() if not missing else False
    healthy=nr_health(p,secret) if services else False
    return {'Foreign':foreign,'Files':files,'Services':digest('native-four-workers-and-crond/v1') if services else 'absent','Health':digest('native-AMI-modules-dialplan-database-updater/v1') if healthy else 'absent'}

def native_main():
    try:
        mode=sys.argv[1:]
        if mode==['native-health']:
            value=json.loads(nr_protected(NR_HOME/'baseline.json'))
            secret=nr_secret()
            if not secret: raise ValueError('native AMI input unavailable')
            print(json.dumps({'services':nr_services(),'ami':nr_ami(secret),'runtime':nr_health(value,secret)}));return
        p=json.load(sys.stdin)
        if mode==['native-inspect']: print(json.dumps(nr_state(p)));return
        if mode==['native-files']:
            state=nr_state(p)
            if state['Foreign']!=p['ExpectedBaseline']: raise ValueError('native baseline changed before publication')
            value=nr_load_baseline(p)
            if value is None:
                NR_HOME.mkdir(mode=0o700,exist_ok=True);safe(NR_HOME)
                value={'Binding':p['Binding'],'Policy':p['Policy'],'Files':nr_baseline(p),'ModuleSHA256':sha(Path('/var/lib/almadial/asterisk-s04/asterisk-22.12.0/apps/app_dial.so').read_bytes()),'Hostname':p['Hostname'],'WSSURL':p['WSSURL'],'Network':p['Network']}
                atomic(NR_HOME/'baseline.json',json.dumps(value,sort_keys=True).encode(),0o600)
            baseline=value['Files'];secret=nr_secret(True)
            desired=nr_desired(p,secret);wrappers=nr_wrappers(p,baseline,secret)
            for path,raw in {**desired,**wrappers}.items():
                expected=p['DeploymentPins'].get(path) or baseline.get(path,{}).get('sha256')
                nr_publish(path,raw,expected)
            module=Path('/usr/lib/asterisk/modules/app_dial.so')
            raw=Path('/var/lib/almadial/asterisk-s04/asterisk-22.12.0/apps/app_dial.so').read_bytes()
            if sha(raw)!=value['ModuleSHA256']: raise ValueError('native retained module build drift')
            nr_publish(module,raw)
            atomic(NR_SECRET,(secret+'\n').encode(),0o640,grp.getgrnam('asterisk').gr_gid)
            nr_database(secret,p,True)
            # Native generation reads its header before generating mailboxes.
            # Seed only a missing file; existing operator/application content wins.
            if not Path('/etc/asterisk/voicemail.conf').exists():
                nr_publish('/etc/asterisk/voicemail.conf',b'[general]\n[default]\n; END OF FILE\n')
            print(json.dumps(nr_state(p)));return
        if mode==['native-services']:
            state=nr_state(p)
            if state['Files']=='absent': raise ValueError('native files incomplete; reconcile first')
            run(['/usr/bin/systemctl','daemon-reload'])
            # Generate application-owned phone/recording contexts before readiness.
            run(['/usr/bin/systemctl','start','almadial-native-maintenance.service'])
            commands=['manager reload','dialplan reload','module reload res_pjsip.so','module reload res_pjsip_config_wizard.so']
            if not re.search(r'^app_dial.so\s+.*Running',run(['/usr/sbin/asterisk','-rx','module show like app_dial.so']),re.M): commands.insert(1,'module load app_dial.so')
            for cmd in commands:
                output=run(['/usr/sbin/asterisk','-rx',cmd])
                if 'Unable to load' in output or 'No such command' in output: raise ValueError('native Asterisk activation failed')
            for name in ['listener','updater','sender','autodial']:
                run(['/usr/bin/systemctl','enable','--now','almadial-native-'+name+'.service'])
            deadline=time.monotonic()+35
            while time.monotonic()<deadline:
                state=nr_state(p)
                if state['Health']!='absent': print(json.dumps(state));return
                time.sleep(1)
            raise ValueError('native runtime readiness failed; inspect owned unit journals and server_updater')
        raise ValueError('unsupported native runtime operation')
    except Exception as error:
        # No SQL, input bytes, AMI data or subprocess output is exported.
        print('native runtime: '+str(error) if isinstance(error,ValueError) else 'native runtime operation failed: '+type(error).__name__+' at '+','.join(str(x.lineno)+':'+x.name for x in __import__('traceback').extract_tb(error.__traceback__)),file=sys.stderr)
        sys.exit(1)

if __name__=='__main__': native_main()
