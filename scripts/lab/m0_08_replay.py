#!/usr/bin/env python3
"""Explicit ALM-8 lab steps, only on almadial-alma10-repeat; not an installer.

Root steps require the recorded clean-baseline snapshot and verified input bundle.
Each step refuses existing outputs. Failures preserve partial state for inspection.
No remote calls, campaign daemons, schema drops or snapshot restoration.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import zipfile
import secrets
import sys

ROOT = Path('/root/almadial-m008')
INPUT = Path('/home/ubuntu/alm8-inputs')


def run(label, argv, data=None, timeout=120):
    path = ROOT / (label + '.json')
    if path.exists():
        raise FileExistsError(path)
    p = subprocess.run(argv, input=data, text=True, capture_output=True, timeout=timeout)
    path.write_text(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                   argv=argv, exit_code=p.returncode,
                                   stdout=p.stdout, stderr=p.stderr), indent=2))
    print(label, p.returncode, flush=True)
    if p.returncode:
        raise RuntimeError(label + ' failed; inspect protected command record')
    return p.stdout


def check_inputs():
    pins = json.loads((INPUT / 'inputs.json').read_text())
    for name, pin in pins.items():
        if Path(name).name != name or name in ('', '.', '..'):
            raise ValueError('input must be a plain filename')
        p = INPUT / name
        if p.is_symlink() or hashlib.sha256(p.read_bytes()).hexdigest() != pin:
            raise ValueError('input mismatch: ' + name)


def packages():
    specs = json.loads((INPUT / 'package-specs.json').read_text())
    run('packages-before', ['rpm', '-qa', '--qf', '%{NAME}|%{EPOCHNUM}|%{VERSION}|%{RELEASE}|%{ARCH}\n'])
    run('install-build-runtime', ['dnf', '-y', '--disablerepo=*',
        '--enablerepo=baseos,appstream,crb,extras', 'install', *specs], timeout=1200)
    run('epel-key-import', ['rpm', '--import', str(INPUT / 'RPM-GPG-KEY-EPEL-10')])
    run('epel-rpm-signatures', ['rpmkeys', '--checksig', str(INPUT / 'perl-Switch.rpm'),
                               str(INPUT / 'perl-Crypt-Eksblowfish.rpm')])
    run('install-perl-epel', ['dnf', '-y', '--disablerepo=*',
        '--enablerepo=baseos,appstream,crb,extras',
        '--repofrompath=epel-alm8,https://dl.fedoraproject.org/pub/epel/10.2/Everything/x86_64/',
        '--setopt=epel-alm8.gpgcheck=1',
        '--setopt=epel-alm8.gpgkey=file://' + str(INPUT / 'RPM-GPG-KEY-EPEL-10'),
        '--setopt=localpkg_gpgcheck=1', 'install', str(INPUT / 'perl-Switch.rpm'),
        str(INPUT / 'perl-Crypt-Eksblowfish.rpm'), 'perl-Class-Mix-0.006-22.el10_0',
        'perl-Devel-CallChecker-0.009-7.el10_0', 'perl-DynaLoader-Functions-0.004-5.el10_0',
        'perl-Params-Classify-0.015-18.el10_0'], timeout=1200)
    run('packages-after', ['rpm', '-qa', '--qf', '%{NAME}|%{EPOCHNUM}|%{VERSION}|%{RELEASE}|%{ARCH}\n'])
    run('repository-policy', ['dnf', 'repolist', '--enabled'])


def sources():
    source = ROOT / 'source'
    source.mkdir(mode=0o700)
    with zipfile.ZipFile(INPUT / 'vicidial-r4037.zip') as z:
        for item in z.infolist():
            rel = Path(*Path(item.filename).parts[1:])
            if str(rel) == '.':
                continue
            if rel.is_absolute() or '..' in rel.parts or (item.external_attr >> 16) & 0o170000 == 0o120000:
                raise ValueError('unsafe source member')
            out = source / rel
            if item.is_dir():
                out.mkdir(parents=True, exist_ok=True)
            else:
                out.parent.mkdir(parents=True, exist_ok=True)
                out.write_bytes(z.read(item))
    records = ''.join(hashlib.sha256(p.read_bytes()).hexdigest() + '  ' +
                      p.relative_to(source).as_posix() + '\n'
                      for p in sorted(source.rglob('*')) if p.is_file())
    if hashlib.sha256(records.encode()).hexdigest() != '04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed':
        raise ValueError('r4037 tree identity mismatch')
    (ROOT / 'source-files.sha256').write_text(records)
    run('schema-perl-derivation', ['python3', str(INPUT / 'prepare_vicidial_r4037.py'),
        '--source-dir', str(source), '--output-dir', str(ROOT / 'derived')])
    run('generator-derivation', ['python3', str(INPUT / 'm0_05_generator.py'),
        '--source', str(source / 'bin/ADMIN_keepalive_ALL.pl'), '--output-dir', str(ROOT / 'generator')])
    # Net::Telnet is pure Perl; no unchecked CPAN installation script is executed.
    with tarfile.open(INPUT / 'Net-Telnet-3.05.tar.gz') as t:
        data = t.extractfile('Net-Telnet-3.05/lib/Net/Telnet.pm').read()
    module = Path('/usr/local/share/perl5/5.40/Net/Telnet.pm')
    if module.exists():
        raise FileExistsError(module)
    module.parent.mkdir(mode=0o755, parents=True, exist_ok=True)
    for parent in [module.parent, module.parent.parent, module.parent.parent.parent]:
        parent.chmod(0o755)
    module.write_bytes(data)
    module.chmod(0o644)
    run('perl-module-closure', ['runuser', '-u', 'nobody', '--', 'perl', '-MDBI', '-MDBD::mysql',
        '-MNet::Telnet', '-MNet::Server', '-MSwitch', '-MCrypt::Eksblowfish::Bcrypt',
        '-MTime::HiRes', '-e', 'print "Perl=$] DBI=$DBI::VERSION DBD=$DBD::mysql::VERSION Telnet=$Net::Telnet::VERSION Server=$Net::Server::VERSION\n"'])


def build():
    base = Path('/home/ubuntu/alm3')
    base.mkdir(mode=0o755)
    cache = base / 'cache'
    cache.mkdir(mode=0o755)
    shutil.copyfile(INPUT / 'pjproject-2.17.tar.bz2', cache / 'pjproject-2.17.tar.bz2')
    with tarfile.open(INPUT / 'asterisk-22.12.0.tar.gz') as t:
        t.extractall(base, filter='data')
    for p in [base, *base.rglob('*')]:
        shutil.chown(p, 'ubuntu', 'ubuntu')
    source = base / 'asterisk-22.12.0'
    # argv and cwd are fixed; no shell, downloads, source patches or make samples.
    def make(label, argv, timeout=120):
        run(label, ['runuser', '-u', 'ubuntu', '--', 'env', '-C', str(source), *argv], timeout=timeout)
    make('asterisk-configure', ['./configure', '--with-pjproject-bundled',
         '--with-externals-cache=' + str(cache), '--without-dahdi'], 900)
    make('asterisk-menuselect-init', ['make', 'menuselect.makeopts'], 300)
    make('asterisk-menuselect', ['menuselect/menuselect',
         '--disable-category', 'MENUSELECT_CORE_SOUNDS',
         '--disable-category', 'MENUSELECT_EXTRA_SOUNDS',
         '--disable-category', 'MENUSELECT_MOH', '--disable', 'BUILD_NATIVE',
         *[x for module in ['chan_pjsip', 'res_pjsip', 'res_pjsip_transport_websocket',
             'res_http_websocket', 'res_srtp', 'res_crypto', 'app_confbridge', 'app_echo',
             'res_pjsip_config_wizard'] for x in ['--enable', module]], 'menuselect.makeopts'])
    make('asterisk-make', ['make', '-j4'], 1800)
    run('asterisk-install', ['make', '-C', str(source), 'install'], timeout=300)
    run('asterisk-binary-version', ['/usr/sbin/asterisk', '-V'])
    run('asterisk-compiler', ['gcc', '--version'])
    run('asterisk-linker', ['ld', '--version'])
    (ROOT / 'asterisk-binary.sha256').write_text(hashlib.sha256(Path('/usr/sbin/asterisk').read_bytes()).hexdigest() + '\n')


def database():
    cfg = Path('/etc/my.cnf.d/almadial-m008.cnf')
    if cfg.exists():
        raise FileExistsError(cfg)
    cfg.write_text('[mariadb]\nskip-networking\n')
    cfg.chmod(0o640)
    shutil.chown(cfg, 'root', 'mysql')
    run('database-start', ['systemctl', 'enable', '--now', 'mariadb'])
    run('database-baseline', ['mariadb', '-N'], 'SELECT VERSION(),@@GLOBAL.sql_mode,@@skip_networking; SHOW DATABASES;')
    run('database-create', ['mariadb'], 'CREATE DATABASE almadial_m005 CHARACTER SET utf8mb3 COLLATE utf8mb3_unicode_ci;')
    run('database-schema-import', ['mariadb', '--show-warnings', 'almadial_m005'],
        (ROOT / 'derived/extras/MySQL_AST_CREATE_tables.sql').read_text(), 180)
    run('database-first-server', ['mariadb', '--show-warnings', 'almadial_m005'],
        (ROOT / 'source/extras/first_server_install.sql').read_text())
    result = run('database-schema-postcondition', ['mariadb', '-N', 'almadial_m005'],
        'SELECT db_schema_version FROM system_settings; SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE(); '
        'SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE(); '
        'SELECT ENGINE,COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() GROUP BY ENGINE;')
    if result.splitlines()[:3] != ['1748', '345', '1011']:
        raise ValueError('schema postcondition mismatch')
    # stdin is deliberately absent from command records; credentials never enter argv.
    dbpw, adminpw = secrets.token_hex(24), secrets.token_hex(16)
    (ROOT / 'credentials.json').write_text(json.dumps({'db_user':'alm008', 'db_password':dbpw,
                                                     'admin_user':'6666', 'admin_password':adminpw}))
    run('database-grants', ['mariadb'], "CREATE USER 'alm008'@'localhost' IDENTIFIED BY '" + dbpw + "'; "
        "GRANT SELECT,INSERT,UPDATE,DELETE,LOCK TABLES ON almadial_m005.* TO 'alm008'@'localhost'; "
        "UPDATE almadial_m005.vicidial_users SET pass='" + adminpw + "' WHERE user='6666';")


def put(path, content, mode=0o640, user='root', group='asterisk'):
    p = Path(path)
    if p.exists():
        raise FileExistsError(p)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)
    p.chmod(mode)
    shutil.chown(p, user, group)


def base():
    run('asterisk-account', ['useradd', '--system', '--home-dir', '/var/lib/asterisk',
                            '--shell', '/sbin/nologin', 'asterisk'])
    for name in ['/var/lib/asterisk', '/var/spool/asterisk', '/var/log/asterisk',
                 '/var/spool/asterisk/monitor']:
        p = Path(name)
        p.mkdir(mode=0o750, parents=True, exist_ok=True)
        # make install may populate these paths; only Asterisk-owned runtime data.
        for f in [p, *p.rglob('*')]:
            shutil.chown(f, 'asterisk', 'asterisk')
    Path('/etc/asterisk').mkdir(exist_ok=True)
    Path('/etc/asterisk').chmod(0o750)
    shutil.chown('/etc/asterisk', 'root', 'asterisk')
    keys = Path('/etc/asterisk/keys')
    keys.mkdir(mode=0o750)
    keys.chmod(0o750)  # Explicitly override the private evidence umask for service traversal.
    shutil.chown(keys, 'root', 'asterisk')
    ca = ROOT / 'ca'
    ca.mkdir(mode=0o700)
    run('tls-ca', ['openssl', 'req', '-x509', '-newkey', 'rsa:3072', '-nodes',
        '-keyout', str(ca / 'ca.key'), '-out', str(ca / 'ca.crt'), '-days', '7',
        '-subj', '/CN=AlmaDial ALM8 isolated repeat laboratory CA'])
    run('tls-server-request', ['openssl', 'req', '-newkey', 'rsa:3072', '-nodes',
        '-keyout', str(keys / 'server.key'), '-out', str(ca / 'server.csr'),
        '-subj', '/CN=almadial-alma10.test'])
    (ca / 'server.ext').write_text('subjectAltName=DNS:almadial-alma10.test\nextendedKeyUsage=serverAuth\nbasicConstraints=CA:FALSE\n')
    run('tls-server-sign', ['openssl', 'x509', '-req', '-in', str(ca / 'server.csr'),
        '-CA', str(ca / 'ca.crt'), '-CAkey', str(ca / 'ca.key'), '-CAcreateserial',
        '-out', str(keys / 'server.crt'), '-days', '7', '-extfile', str(ca / 'server.ext')])
    run('tls-verify', ['openssl', 'verify', '-CAfile', str(ca / 'ca.crt'),
                     '-verify_hostname', 'almadial-alma10.test', str(keys / 'server.crt')])
    for p in keys.iterdir():
        p.chmod(0o640)
        shutil.chown(p, 'root', 'asterisk')
    put('/etc/asterisk/asterisk.conf', '''[directories]
astetcdir => /etc/asterisk
astmoddir => /usr/lib/asterisk/modules
astvarlibdir => /var/lib/asterisk
astdbdir => /var/lib/asterisk
astkeydir => /var/lib/asterisk
astdatadir => /var/lib/asterisk
astagidir => /var/lib/asterisk/agi-bin
astspooldir => /var/spool/asterisk
astrundir => /run/asterisk
astlogdir => /var/log/asterisk
[options]
runuser = asterisk
rungroup = asterisk
''')
    modules = '''res_crypto res_http_websocket res_pjproject res_sorcery_config
res_sorcery_memory res_sorcery_astdb res_sorcery_memory_cache res_pjsip
res_pjsip_config_wizard res_pjsip_authenticator_digest res_pjsip_endpoint_identifier_user
res_pjsip_endpoint_identifier_ip res_pjsip_registrar res_pjsip_session res_pjsip_sdp_rtp
res_pjsip_nat res_pjsip_transport_websocket res_rtp_asterisk res_srtp res_timing_timerfd
res_pjsip_pubsub chan_pjsip pbx_config app_echo app_confbridge bridge_softmix bridge_simple
bridge_builtin_features bridge_builtin_interval_features codec_ulaw codec_alaw codec_resample
format_pcm format_wav func_callerid func_volume func_jitterbuffer func_hangupcause app_exec app_dial res_speech
res_agi app_mixmonitor app_playback func_channel func_logic func_strings res_pjsip_dtmf_info'''.split()
    put('/etc/asterisk/modules.conf', '[modules]\nautoload=no\n' + ''.join('load=' + m + '.so\n' for m in modules))
    put('/etc/asterisk/http.conf', '''[general]
enabled=yes
bindaddr=127.0.0.1
bindport=8088
enablestatic=yes
tlsenable=yes
tlsbindaddr=0.0.0.0:8089
tlscertfile=/etc/asterisk/keys/server.crt
tlsprivatekey=/etc/asterisk/keys/server.key
''')
    put('/etc/asterisk/rtp.conf', '[general]\nrtpstart=10000\nrtpend=10019\nicesupport=yes\nstrictrtp=yes\n')
    native = secrets.token_hex(24)
    (ROOT / 'phone-9001').write_text(native + '\n')
    put('/etc/asterisk/pjsip.conf', '''[transport-wss]
type=transport
protocol=wss
bind=0.0.0.0
[9001]
type=aor
max_contacts=1
remove_existing=yes
[9001]
type=auth
auth_type=userpass
username=9001
password=''' + native + '''
[9001]
type=endpoint
transport=transport-wss
context=alm3-echo
disallow=all
allow=ulaw,alaw
auth=9001
aors=9001
webrtc=yes
dtls_auto_generate_cert=yes
ice_support=yes
use_avpf=yes
rtcp_mux=yes
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes
''')
    put('/etc/asterisk/extensions.conf', '''[globals]
[alm3-echo]
exten => 9001,1,Answer()
 same => n,Echo()
 same => n,Hangup()
''')
    put('/etc/asterisk/confbridge.conf', '[general]\n')
    put('/etc/asterisk/logger.conf', '[general]\n[logfiles]\nconsole => notice,warning,error\nmessages => notice,warning,error,verbose,dtmf\n')
    put('/etc/systemd/system/asterisk.service', '''[Unit]
Description=AlmaDial native Asterisk 22 lab runtime
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
User=asterisk
Group=asterisk
RuntimeDirectory=asterisk
RuntimeDirectoryMode=0750
UMask=0027
ExecStart=/usr/sbin/asterisk -f -C /etc/asterisk/asterisk.conf
ExecStop=/usr/sbin/asterisk -rx 'core stop now'
Restart=on-failure
[Install]
WantedBy=multi-user.target
''', 0o644, group='root')
    run('base-selinux-labels', ['restorecon', '-R', '/etc/asterisk', '/var/lib/asterisk',
                              '/var/spool/asterisk', '/var/log/asterisk', '/etc/systemd/system/asterisk.service'])
    run('base-systemd-reload', ['systemctl', 'daemon-reload'])
    run('base-asterisk-start', ['systemctl', 'enable', '--now', 'asterisk'])


def generator():
    def sql(q):
        return subprocess.run(['mariadb', '-N', 'almadial_m005'], input=q, text=True,
                              capture_output=True, check=True, timeout=30).stdout.strip()
    stage = Path('/root/almadial-m005/generated')
    stage.mkdir(mode=0o700, parents=True)
    (stage / 'voicemail.conf').write_text('[general]\n[default]\n')
    put('/root/almadial-m005/tools/m005-screen', '#!/bin/sh\nprintf "%s\\n" "$*" >> /root/almadial-m005/reload-requests.log\n', 0o700, group='root')
    sql("DELETE FROM phones; DELETE FROM conferences; DELETE FROM vicidial_conferences; "
        "UPDATE server_updater SET server_ip='127.0.0.1'; "
        "UPDATE system_settings SET custom_dialplan_entry='',generate_cross_server_exten='0',default_codecs='ulaw,alaw'; "
        "UPDATE servers SET server_ip='127.0.0.1',active='Y',active_asterisk_server='Y',generate_vicidial_conf='Y',"
        "rebuild_conf_files='Y',asterisk_version='22.12.0',conf_engine='CONFBRIDGE',conf_secret='" + secrets.token_hex(20) + "',custom_dialplan_entry='';")
    runtime = Path('/root/almadial-m006')
    runtime.mkdir(mode=0o700)
    websecrets = {}
    for identity, context, template in [('9051','alm5-local','ALM5-WSS'),
            ('9052','alm5-local','ALM5-WSS'), ('9053','alm6-supervisor','ALM6-MON')]:
        password, webpw = secrets.token_hex(24), secrets.token_hex(20)
        (runtime / ('phone-' + identity)).write_text(password + '\n')
        websecrets[identity] = webpw
        sql("INSERT INTO phones SET extension='" + identity + "',dialplan_number='" + identity + "',voicemail_id='" + identity + "',"
            "login='" + identity + "',pass='" + webpw + "',server_ip='127.0.0.1',protocol='PJSIP',active='Y',"
            "conf_secret='" + password + "',phone_context='" + context + "',ext_context='alm5-local',phone_ring_timeout=20,"
            "codecs_list='ulaw,alaw',template_id='" + template + "',fullname='ALM8 local test " + identity + "',outbound_cid='" + identity + "';")
    (ROOT / 'phone-web-secrets.json').write_text(json.dumps(websecrets))
    for template, context in [('ALM5-WSS','alm5-local'), ('ALM6-MON','alm6-supervisor')]:
        content = 'transport=transport-wss\naor/max_contacts=1\nendpoint/context=' + context + '\nendpoint/webrtc=yes\nendpoint/dtls_auto_generate_cert=yes\nendpoint/direct_media=no\nendpoint/rtp_symmetric=yes\nendpoint/rewrite_contact=yes\nendpoint/disallow=all\nendpoint/allow=ulaw,alaw'
        sql("INSERT INTO vicidial_conf_templates SET template_id='" + template + "',template_name='ALM8 local WSS',template_contents='" + content + "';")
    for name in ['first', 'second']:
        sql("UPDATE servers SET rebuild_conf_files='Y' WHERE server_ip='127.0.0.1';")
        run('generator-' + name, ['perl', str(ROOT / 'generator/generator.pl')])
        (ROOT / ('generated-' + name + '.json')).write_text(json.dumps({p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in stage.iterdir() if p.is_file()}, indent=2))
    if (ROOT / 'generated-first.json').read_bytes() != (ROOT / 'generated-second.json').read_bytes():
        raise ValueError('generation not byte-identical; inspect before publishing')
    base = Path('/etc/asterisk')
    for name in ['pjsip-vicidial.conf','pjsip_wizard-vicidial.conf','extensions-vicidial.conf','confbridge-vicidial.conf']:
        put(base / name, (stage / name).read_text())
    for name, declaration in [('pjsip.conf','pjsip-vicidial.conf'), ('confbridge.conf','confbridge-vicidial.conf'), ('extensions.conf','extensions-vicidial.conf')]:
        p = base / name
        shutil.copy2(p, ROOT / (name + '.before-generated'))
        extra = '\n[globals]\n' if name == 'extensions.conf' else '\n'
        p.write_text(p.read_text() + extra + '#include "' + declaration + '"\n')
    put(base / 'pjsip_wizard.conf', '#include "pjsip_wizard-vicidial.conf"\n')
    with (base / 'extensions.conf').open('a') as f:
        f.write('''
[alm5-local]
include => vicidial-auto-phones
include => alm6-test
exten => 9059,1,Hangup()
[alm6-test]
exten => 9600001,1,Answer()
 same => n,Set(CALLERID(name)=ALM6-${CALLERID(num)})
 same => n,AGI(agi://127.0.0.1:4577/call_log)
 same => n,Goto(vicidial-auto-external,473782179600001,1)
exten => 9600099,1,Answer()
 same => n,AGI(agi://127.0.0.1:4577/call_log)
 same => n,Hangup()
exten => h,1,AGI(agi://127.0.0.1:4577/call_log--HVcauses--PRI-----NODEBUG-----${HANGUPCAUSE}-----${DIALSTATUS}-----${DIALEDTIME}-----${ANSWEREDTIME})
[alm6-supervisor]
exten => 9600001,1,Answer()
 same => n,ConfBridge(9600001,vici_agent_bridge,vici_monitor_user,vici_monitor_menu)
 same => n,Hangup()
[alm6-barge]
exten => 9600001,1,ConfBridge(9600001,vici_agent_bridge,vici_barge_user)
 same => n,Hangup()
[alm6-transfer]
exten => 9600001,1,Goto(vicidial-auto-external,473782179600001,1)
''')
    ami = secrets.token_hex(10)  # r4037 servers.ASTmgrSECRET is varchar(20)
    (runtime / 'ami-secret').write_text(ami + '\n')
    put(base / 'manager.conf', '[general]\nenabled=yes\nwebenabled=no\nport=5038\nbindaddr=127.0.0.1\n[alm6-runtime]\nsecret=' + ami + '\ndeny=0.0.0.0/0.0.0.0\npermit=127.0.0.1/255.255.255.255\nread=system,call,reporting,agent\nwrite=system,call,reporting,agent,originate\n')
    run('generated-selinux', ['restorecon', '-R', str(base)])
    for i, cmd in enumerate(['module reload res_pjsip.so','module reload res_pjsip_config_wizard.so', 'dialplan reload','module reload app_confbridge.so','manager reload','pjsip show endpoints','dialplan show 9600001@alm6-test','confbridge show profile bridge vici_agent_bridge']):
        run('generated-runtime-' + str(i), ['asterisk', '-rx', cmd])


def telephony():
    sys.path.insert(0, str(INPUT))
    import m0_06_fastagi
    app = Path('/opt/almadial-m006')
    app.mkdir(mode=0o755)
    app.chmod(0o755)
    put(app / 'FastAGI_log.pl', m0_06_fastagi.derive((ROOT / 'source/bin/FastAGI_log.pl').read_bytes()).decode(), 0o644)
    logs = Path('/var/log/almadial-m006')
    logs.mkdir(mode=0o750)
    logs.chmod(0o750)
    shutil.chown(logs, 'asterisk', 'asterisk')
    password = secrets.token_hex(24)
    run('fastagi-account', ['mariadb'], "CREATE USER 'alm6'@'localhost' IDENTIFIED BY '" + password + "'; "
        "GRANT SELECT,INSERT,UPDATE ON almadial_m005.* TO 'alm6'@'localhost'; "
        "GRANT DELETE ON almadial_m005.live_inbound TO 'alm6'@'localhost';")
    put('/etc/astguiclient-m006.conf', '''PATHlogs => /var/log/almadial-m006
VARDB_server => localhost
VARDB_database => almadial_m005
VARDB_user => alm6
VARDB_pass => ''' + password + '''
VARDB_port => 3306
VARserver_ip => 127.0.0.1
VARfastagi_log_min_servers => 1
VARfastagi_log_max_servers => 2
VARfastagi_log_min_spare_servers => 1
VARfastagi_log_max_spare_servers => 1
''')
    put('/etc/systemd/system/almadial-m006-fastagi.service', '''[Unit]
Description=AlmaDial bounded r4037 FastAGI lab runtime
After=mariadb.service network.target
Requires=mariadb.service
[Service]
Type=simple
User=asterisk
Group=asterisk
ExecStart=/usr/bin/perl /opt/almadial-m006/FastAGI_log.pl
Restart=on-failure
NoNewPrivileges=yes
PrivateTmp=yes
[Install]
WantedBy=multi-user.target
''', 0o644, group='root')
    # Publish fresh diagnostic assets from verified sources, never old captures.
    with tarfile.open(INPUT / 'viciphone.tar.gz') as t:
        t.extractall(ROOT / 'viciphone-source', filter='data')
    phone = ROOT / 'viciphone-source/ViciPhone-24e98fccfa7b7079fff970e02fca5f002ac1f2cc'
    static = Path('/var/lib/asterisk/static-http')
    static.mkdir(exist_ok=True)
    static.chmod(0o755)
    for name, data in [('alm3.html',(INPUT / 'm0_03_webrtc_client.html').read_bytes()),
                       ('alm6.html',(INPUT / 'm0_06_webrtc_client.html').read_bytes()),
                       ('sip.js',(phone / 'src/js/sip.js').read_bytes())]:
        p = static / name
        if p.exists():
            raise FileExistsError(p)
        p.write_bytes(data)
        p.chmod(0o644)
        shutil.chown(p, 'root', 'asterisk')
    telephony_start()


def telephony_start():
    # Also the explicit resume boundary after a missing-module syntax failure.
    sys.path.insert(0, str(INPUT))
    import m0_07_assets
    for name, pin in m0_07_assets.AGI_MODULE_PINS.items():
        data = (ROOT / 'source/extras' / name).read_bytes()
        if hashlib.sha256(data).hexdigest() != pin:
            raise ValueError('AGI module source mismatch')
        p = Path('/usr/local/share/perl5/5.40') / name
        if p.exists():
            raise FileExistsError(p)
        p.parent.mkdir(parents=True, exist_ok=True)
        p.parent.chmod(0o755)
        p.write_bytes(data)
        p.chmod(0o644)
    run('fastagi-prerequisite-nonroot', ['runuser', '-u', 'asterisk', '--', 'perl', '-MAsterisk::AGI', '-e', 'print "Asterisk::AGI loaded as non-root\n"'])
    run('fastagi-syntax-complete', ['perl', '-c', '/opt/almadial-m006/FastAGI_log.pl'])
    run('fastagi-labels', ['restorecon', '-R', '/opt/almadial-m006', '/var/log/almadial-m006',
                          '/etc/astguiclient-m006.conf', '/var/lib/asterisk/static-http'])
    run('fastagi-systemd', ['systemctl', 'daemon-reload'])
    run('fastagi-start', ['systemctl', 'enable', '--now', 'almadial-m006-fastagi'])
    run('native-listeners', ['ss', '-lntup'])
    run('native-enforcing', ['getenforce'])


def application():
    """Fresh webroot, accepted source derivations, bounded upstream runtime only."""
    import base64
    sys.path.insert(0, str(INPUT))
    import m0_07_assets as assets
    source = ROOT / 'source'
    agc = source / 'www/agc'
    isolated = ROOT / 'dbconnect-isolated.php'
    # read_text intentionally normalizes upstream CRLF, matching the accepted pin.
    isolated.write_text((agc / 'dbconnect_mysqli.php').read_text().replace(
        '/etc/astguiclient.conf', '/etc/astguiclient-m007.conf'))
    phone = next((ROOT / 'viciphone-source').glob('*/src'))
    overlay = ROOT / 'application-overlay'
    manifest = assets.prepare(phone, overlay, agc/'vicidial.php', isolated,
        agc/'calendar_db.js', source/'bin/AST_update.pl', agc, source/'bin', source)
    web = Path('/var/www/html/m007')
    shutil.copytree(source/'www', web)
    shutil.copytree(overlay/'viciphone', web/'viciphone')
    for name in ['vicidial.php', 'dbconnect_mysqli.php', 'calendar_db.js', *assets.SESSION_PINS]:
        shutil.copyfile(overlay/name, web/'agc'/name)
    for sub in ['agc', 'vicidial']:
        (web/sub/'options.php').write_text('<?php\n$use_slave_server = 0;\n?>\n')
    # Upstream admin.php attempts first-run report cleanup and marker creation
    # before checking webroot_writable. Provision the accepted immutable marker.
    (web/'vicidial/old_clear').write_text('AlmaDial immutable webroot prerequisite\n')
    for p in [web, *web.rglob('*')]:
        p.chmod(0o750 if p.is_dir() else 0o640)
        shutil.chown(p, 'root', 'apache')
    app = Path('/opt/almadial-m007')
    app.mkdir(mode=0o755)
    app.chmod(0o755)
    for name in ['AST_update.pl', *assets.MANAGER_PINS, *assets.CALL_RUNTIME_PINS, *assets.AGI_MODULE_PINS]:
        p = app/name
        p.parent.mkdir(exist_ok=True, parents=True)
        p.parent.chmod(0o755)
        shutil.copyfile(overlay/name, p)
        p.chmod(0o755 if p.suffix in ('.pl', '.agi') else 0o644)
        if p.suffix in ('.pl', '.agi'):
            run('application-syntax-' + p.name, ['perl', '-c', str(p)])
    logs = Path('/var/log/almadial-m007')
    logs.mkdir(mode=0o750)
    logs.chmod(0o750)
    shutil.chown(logs, 'asterisk', 'asterisk')
    creds = json.loads((ROOT/'credentials.json').read_text())
    conf = ('PATHhome => /opt/almadial-m007\nPATHlogs => /var/log/almadial-m007\n'
        'PATHagi => /opt/almadial-m007\nPATHweb => /var/www/html/m007\n'
        'PATHsounds => /var/lib/asterisk/sounds\nPATHmonitor => /var/spool/asterisk/monitor\n'
        'VARserver_ip => 127.0.0.1\nVARDB_server => localhost\nVARDB_database => almadial_m005\n'
        'VARDB_user => alm008\nVARDB_pass => ' + creds['db_password'] + '\nVARDB_port => 3306\nExpectedDBSchema => 1748\n')
    for name, group in [('astguiclient.conf','apache'), ('astguiclient-m007.conf','apache'),
                         ('astguiclient-m007-updater.conf','asterisk')]:
        put('/etc/'+name, conf, group=group)
    ami = Path('/root/almadial-m006/ami-secret').read_text().strip()
    put('/etc/almadial-m007-ami.secret', ami+'\n')
    manager = ''.join('['+role+']\nsecret='+ami+'\ndeny=0.0.0.0/0.0.0.0\npermit=127.0.0.1/255.255.255.255\nread='+read+'\nwrite=command\n'
        for role, read in [('alm7-updater','system,call'),('alm7-listener','system,call,dtmf')])
    put('/etc/asterisk/manager-alm7.conf', manager)
    with Path('/etc/asterisk/manager.conf').open('a') as f:
        f.write('\n#include "manager-alm7.conf"\n')
    # Secrets are only SQL stdin; no credentials in command argv or console output.
    agentpw = secrets.token_hex(12)
    webpw = json.loads((ROOT/'phone-web-secrets.json').read_text())['9051']
    run('agent-fixture-seed', ['mariadb', 'almadial_m005'], """
UPDATE servers SET ASTmgrUSERNAMEupdate='alm7-updater',ASTmgrUSERNAMElisten='alm7-listener',ASTmgrUSERNAMEsend='alm6-runtime',ASTmgrSECRET='"""+ami+"""',web_socket_url='wss://almadial-alma10.test:8089/ws' WHERE server_ip='127.0.0.1';
UPDATE system_settings SET webroot_writable='0',webphone_url='https://almadial-alma10.test:8443/viciphone/viciphone.php';
INSERT INTO vicidial_user_groups (user_group,group_name,allowed_campaigns) VALUES ('M007','ALM8 isolated QA',' M007 - ');
INSERT INTO vicidial_users (user,pass,full_name,user_level,user_group,active,phone_login,phone_pass,agentcall_manual) VALUES ('m007qa','"""+agentpw+"""','ALM8 test agent',1,'M007','Y','9051','"""+webpw+"""','1');
INSERT INTO vicidial_campaigns (campaign_id,campaign_name,active,dial_method,auto_dial_level,campaign_allow_inbound,allow_closers,campaign_recording,manual_dial_list_id,no_hopper_leads_logins,omit_phone_code,dial_prefix) VALUES ('M007','ALM8 isolated local test','Y','MANUAL','0','N','N','NEVER',7007,'Y','Y','X');
UPDATE phones SET is_webphone='Y',webphone_dialpad='Y',webphone_dialbox='Y',webphone_auto_answer='Y',webphone_mute='Y',webphone_volume='Y',webphone_debug='N',webphone_settings='M007' WHERE extension='9051' AND server_ip='127.0.0.1';
INSERT INTO vicidial_settings_containers (container_id,container_notes,container_type,container_entry) VALUES ('M007','ALM8 synthetic audio settings','OTHER','autoGain:0\\nechoCan:0\\nnoiseSup:0\\ndialRegExten:1');
INSERT INTO vicidial_lists SET list_id=7007,list_name='ALM8 local only',campaign_id='M007',active='Y';
INSERT INTO vicidial_list SET list_id=7007,phone_code='1',phone_number='99052',first_name='Synthetic',last_name='ALM8',status='NEW';
""")
    (ROOT/'agent-credentials.json').write_text(json.dumps({'user':'m007qa','pass':agentpw,'phone_login':'9051','phone_pass':webpw,'campaign':'M007'}))
    for identity in ['9051','9052']:
        values = {'phone_login':identity,'phone_pass':Path('/root/almadial-m006/phone-'+identity).read_text().strip(),
                  'server_ip':'almadial-alma10.test','codecs':'ulaw,alaw',
                  'options':'WEBSOCKETURLwss://almadial-alma10.test:8089/ws--SETTINGSautoGain:0\\nechoCan:0\\nnoiseSup:0--AUTOANSWER_Y'}
        (ROOT/('phone-post-'+identity+'.json')).write_text(json.dumps({k:base64.b64encode(v.encode()).decode() for k,v in values.items()}))
    fixtures = '''[alm5-local](+)
include => alm7-agent
[alm7-agent]
exten => 9600000,1,Answer()
 same => n,Goto(vicidial-auto-external,473782179600000,1)
exten => 99052,1,AGI(agi://127.0.0.1:4577/call_log)
 same => n,Dial(PJSIP/9052,30,oG(alm7-agent^customerjoin^1))
 same => n,Hangup()
exten => 8301,1,Answer()
 same => n,AGI(/opt/almadial-m007/park_CID.agi)
 same => n,Hangup()
exten => customerjoin,1,Hangup()
 same => n,Goto(vicidial-auto-external,473782179600000,1)
'''
    put('/etc/asterisk/extensions-alm7.conf', fixtures)
    with Path('/etc/asterisk/extensions.conf').open('a') as f:
        f.write('\n#include "extensions-alm7.conf"\n')
    h = Path('/etc/httpd/conf/httpd.conf')
    old = h.read_text()
    if old.count('Listen 80') != 1:
        raise ValueError('unexpected Apache listener input')
    shutil.copy2(h, ROOT/'httpd.conf.before')
    h.write_text(old.replace('Listen 80','Listen 127.0.0.1:8080'))
    defaultssl = Path('/etc/httpd/conf.d/ssl.conf')
    if defaultssl.exists():
        defaultssl.rename(ROOT/'ssl.conf.before')
    put('/etc/httpd/conf.d/almadial-m008.conf', '''Listen 8443 https
<VirtualHost *:8443>
ServerName almadial-alma10.test
DocumentRoot /var/www/html/m007
SSLEngine on
SSLCertificateFile /etc/asterisk/keys/server.crt
SSLCertificateKeyFile /etc/asterisk/keys/server.key
Header always set Referrer-Policy "no-referrer"
Header always set Cache-Control "no-store"
LogFormat "%a %t %m %U %>s %b" alm8_safe
CustomLog /var/log/httpd/alm8_access.log alm8_safe
<Directory /var/www/html/m007>
Require all granted
AllowOverride None
</Directory>
</VirtualHost>
''', 0o644, group='root')
    for role, script in [('updater','AST_update.pl'),('listener','AST_manager_listen.pl')]:
        put('/etc/systemd/system/almadial-m007-'+role+'.service', '''[Unit]
Description=AlmaDial bounded upstream agent '''+role+'''
After=mariadb.service asterisk.service
Requires=mariadb.service asterisk.service
[Service]
Type=simple
User=asterisk
Group=asterisk
WorkingDirectory=/opt/almadial-m007
ExecStart=/usr/bin/perl /opt/almadial-m007/'''+script+'''
Restart=on-failure
NoNewPrivileges=yes
[Install]
WantedBy=multi-user.target
''', 0o644, group='root')
    run('application-labels', ['restorecon','-R',str(web),str(app),str(logs),'/etc/httpd',
        '/etc/astguiclient.conf','/etc/astguiclient-m007.conf','/etc/astguiclient-m007-updater.conf',
        '/etc/almadial-m007-ami.secret','/etc/asterisk'])
    run('application-php-lint', ['php','-l',str(web/'agc/vicidial.php')])
    run('application-http-config', ['httpd','-t'])
    run('application-systemd', ['systemctl','daemon-reload'])
    for cmd in ['manager reload','dialplan reload']:
        run('application-'+cmd.split()[0]+'-reload',['asterisk','-rx',cmd])
    run('application-start', ['systemctl','enable','--now','php-fpm','httpd',
        'almadial-m007-updater','almadial-m007-listener'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    steps = {'packages': packages, 'sources': sources, 'build': build, 'database': database,
             'base': base, 'generator': generator, 'telephony': telephony,
             'telephony-start': telephony_start, 'application': application}
    parser.add_argument('step', choices=steps)
    a = parser.parse_args()
    if os.geteuid() != 0 or subprocess.check_output(['hostname'], text=True).strip() != 'almadial-alma10-repeat':
        raise SystemExit('Only root on explicitly authorized almadial-alma10-repeat may execute these steps')
    os.umask(0o077)
    ROOT.mkdir(mode=0o700, exist_ok=True)
    check_inputs()
    steps[a.step]()


if __name__ == '__main__':
    main()
