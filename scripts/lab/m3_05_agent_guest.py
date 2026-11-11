#!/usr/bin/env python3
"""Temporary ALM-27 agent fixture. Parent must hold M1 lock and durable intent.

Only a new clone database, local Alias, temporary AMI/PJSIP include and bounded
upstream updater/listener are used. No accepted application files are patched.
Secret values are generated on the guest and never written to diagnostic output.
"""
import base64
import grp
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = Path('/opt/almadial-alm27')
DB = 'almadial_alm27'
CONF = Path('/etc/astguiclient-alm27.conf')
AMI = Path('/etc/asterisk/almadial-alm27-manager.conf')
PJSIP = Path('/etc/asterisk/almadial-alm27-phone.conf')
WEB = Path('/etc/httpd/conf.modules.d/99-almadial-alm27.conf')
MACHINE = '0fffa9a4683f35b696f0b6ee7330734b'


def run(argv, data=None, timeout=90):
    p = subprocess.run(argv, input=data, capture_output=True, timeout=timeout)
    if p.returncode:
        # SQL and configuration can contain secrets. Never echo input or stderr.
        detail = ''
        if Path(argv[0]).name == 'mariadb':
            lines = p.stderr.decode(errors='replace').splitlines()
            error = next((x for x in lines if x.startswith('ERROR ')), '')
            detail = ': ' + re.sub(r"'[^']*'", '[redacted]', error)[:220]
        raise RuntimeError(Path(argv[0]).name + ' failed with exit ' + str(p.returncode) + detail)
    return p.stdout.decode().strip()


def sql(query):
    return run(['mariadb', '--no-defaults', '--batch', '--raw', '--skip-column-names'], query.encode())


def cli(command):
    result = run(['asterisk', '-rx', command])
    assert 'No such command' not in result and 'failed.' not in result, result[:1000]
    return result


def write(path, text, mode=0o640, group='asterisk'):
    with path.open('xb') as f:
        os.fchmod(f.fileno(), mode)
        os.fchown(f.fileno(), 0, grp.getgrnam(group).gr_gid)
        f.write(text.encode() if isinstance(text, str) else text)
        f.flush()
        os.fsync(f.fileno())


def replace(path, expected, content):
    assert path.read_bytes() == expected
    st = path.stat()
    fd, name = tempfile.mkstemp(prefix='.alm27-', dir=path.parent)
    try:
        with os.fdopen(fd, 'wb') as f:
            os.fchmod(f.fileno(), st.st_mode & 0o7777)
            os.fchown(f.fileno(), st.st_uid, st.st_gid)
            f.write(content)
            f.flush()
            os.fsync(f.fileno())
        assert path.read_bytes() == expected
        os.replace(name, path)
        d = os.open(path.parent, os.O_DIRECTORY)
        try:
            os.fsync(d)
        finally:
            os.close(d)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def wrapper(path, include, add):
    marker = ('\n#include ' + include.name + '\n').encode()
    assert not path.is_symlink()
    original = ROOT / (path.name + '.original')
    if add:
        write(original, path.read_bytes(), 0o600)
        assert marker not in path.read_bytes()
        replace(path, original.read_bytes(), original.read_bytes() + marker)
    else:
        assert path.read_bytes() == original.read_bytes() + marker
        replace(path, original.read_bytes() + marker, original.read_bytes())
    run(['restorecon', str(path)])


def inspect():
    assert Path('/etc/machine-id').read_text().strip() == MACHINE
    assert run(['getenforce']) == 'Enforcing'
    assert run(['systemctl', 'is-active', 'asterisk', 'mariadb', 'httpd', 'firewalld']).split() == ['active'] * 4
    return {'machine': MACHINE, 'present': ROOT.exists(), 'channels': cli('core show channels concise'),
            'database_present': sql("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='almadial_alm27'") == '1',
            'accepted_phone_digest': hashlib.sha256(sql("SELECT JSON_ARRAY(extension,server_ip,login,pass) FROM asterisk.phones ORDER BY extension").encode()).hexdigest()}


def prepare(payload):
    state = inspect()
    assert not state['present'] and not state['database_present'] and not state['channels']
    assert not any(p.exists() for p in [CONF, AMI, PJSIP, WEB])
    ROOT.mkdir(mode=0o755)
    os.chown(ROOT, 0, grp.getgrnam('apache').gr_gid)
    write(ROOT / 'authority.json', json.dumps({'task': 'ALM-27', 'machine': MACHINE, 'parent_intent': payload['intent']}), 0o600)
    archive = Path('/var/lib/almadial/asterisk-s04/asterisk.tar.gz')
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == 'f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2'
    assert 'app_dial.so' not in cli('module show like app_dial')
    dial = Path('/var/lib/almadial/asterisk-s04/asterisk-22.12.0/apps/app_dial.so')
    target = Path('/usr/lib/asterisk/modules/app_dial.so')
    assert not target.exists() and not target.is_symlink()
    write(ROOT / 'dial-module.json', json.dumps({'sha256': hashlib.sha256(dial.read_bytes()).hexdigest()}), 0o600)
    write(target, dial.read_bytes(), 0o644)
    run(['restorecon', str(target)])
    cli('module load app_dial.so')
    assert 'app_dial.so' in cli('module show like app_dial')
    # Clone the existing complete schema/data; never import or change asterisk.
    sql('CREATE DATABASE almadial_alm27;')
    dump = subprocess.run(['mariadb-dump', '--no-defaults', '--single-transaction', '--skip-lock-tables', 'asterisk'], capture_output=True, timeout=120)
    assert dump.returncode == 0
    run(['mariadb', '--no-defaults', DB], dump.stdout, timeout=120)
    credential = {key: secrets.token_hex(10 if key == 'ami' else 16) for key in ['database', 'ami', 'agent', '9061', '9062']}
    write(ROOT / 'credentials.json', json.dumps(credential), 0o600)
    sql("CREATE USER 'alm27'@'localhost' IDENTIFIED BY '" + credential['database'] + "'; GRANT ALL ON almadial_alm27.* TO 'alm27'@'localhost';")
    # The base table is untouched; only the owned clone gets synthetic records.
    sql("USE almadial_alm27; DELETE FROM phones; DELETE FROM vicidial_live_agents; DELETE FROM vicidial_manager; DELETE FROM vicidial_auto_calls; DELETE FROM web_client_sessions; DELETE FROM vicidial_session_data; UPDATE vicidial_confbridges SET extension='';")
    cols = sql("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='asterisk' AND TABLE_NAME='phones' ORDER BY ORDINAL_POSITION").splitlines()
    for identity in ['9061', '9062']:
        values = {k: "`" + k + "`" for k in cols}
        for key, value in dict(extension=identity, login=identity, login_pass=credential[identity], pass_=credential[identity], conf_secret=credential[identity], meetme_enter_login_filename='', dialplan_number=identity, protocol='PJSIP',
                               ext_context='alm27-agent', phone_context='alm27-agent', is_webphone='Y',
                               webphone_dialpad='Y', webphone_auto_answer='Y', webphone_settings='ALMADIAL_S10').items():
            values['pass' if key == 'pass_' else key] = "'" + value + "'"
        sql('INSERT INTO almadial_alm27.phones SELECT ' + ','.join(values[k] for k in cols) + " FROM asterisk.phones WHERE extension='alm_s10_9061';")
    sql("USE almadial_alm27; INSERT INTO vicidial_user_groups(user_group,group_name,allowed_campaigns) VALUES('ALM27','Synthetic acceptance',' ALM27 '); INSERT INTO vicidial_users(user,pass,full_name,user_level,user_group,active,phone_login,phone_pass) VALUES('alm27qa','" + credential['agent'] + "','Synthetic acceptance',1,'ALM27','Y','9061','" + credential['9061'] + "'); INSERT INTO vicidial_campaigns(campaign_id,campaign_name,active,dial_method,auto_dial_level,campaign_allow_inbound,allow_closers,campaign_recording,manual_dial_list_id,dial_prefix,manual_dial_prefix,manual_dial_filter,agent_allow_group_alias,web_form_address) VALUES('ALM27','Synthetic acceptance','Y','MANUAL',0,'N','N','NEVER',999,'','','NONE','N',''); UPDATE system_settings SET webphone_url='https://localhost:8443/alm27-phone/viciphone.php'; UPDATE servers SET ext_context='alm27-agent',web_socket_url='wss://localhost:8089/ws',ASTmgrUSERNAME='alm27',ASTmgrUSERNAMEupdate='alm27',ASTmgrUSERNAMElisten='alm27',ASTmgrUSERNAMEsend='alm27',ASTmgrSECRET='" + credential['ami'] + "',vd_server_logs='N' WHERE server_ip='127.0.0.1';")
    sql("UPDATE almadial_alm27.vicidial_campaigns SET no_hopper_leads_logins='Y',manual_dial_lead_id='Y' WHERE campaign_id='ALM27';")
    sql("USE almadial_alm27; INSERT INTO vicidial_lists(list_id,list_name,campaign_id,active) VALUES(999,'Synthetic acceptance','ALM27','Y') ON DUPLICATE KEY UPDATE campaign_id='ALM27',active='Y'; DELETE FROM vicidial_list; INSERT INTO vicidial_list(lead_id,list_id,status,gmt_offset_now,phone_code,phone_number,first_name,last_name) VALUES(27,999,'NEW',0,'','99062','Synthetic','Acceptance');")
    sql("UPDATE almadial_alm27.vicidial_users SET agentcall_manual='1',vicidial_transfers='1' WHERE user='alm27qa';")
    for name in ['bin', 'logs', 'web']:
        (ROOT / name).mkdir(mode=0o755)
    os.chown(ROOT / 'logs', pwd.getpwnam('asterisk').pw_uid, grp.getgrnam('asterisk').gr_gid)
    shutil.copytree('/var/www/html/agc', ROOT / 'web/agc', symlinks=False)
    shutil.copytree('/var/www/html/almadial-phone', ROOT / 'web/phone', symlinks=False)
    for path in (ROOT / 'web/phone').glob('*.php'):
        data = path.read_bytes()
        path.write_bytes(data.replace(b'/var/www/html/agc/', bytes(str(ROOT / 'web/agc') + '/', 'utf-8')))
    for name, encoded in payload['files'].items():
        assert (name.startswith(('web/agc/', 'bin/')) or name == 'web/phone/index.php') and '..' not in Path(name).parts
        p = ROOT / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_bytes(base64.b64decode(encoded, validate=True))
        os.chmod(p, 0o755 if name.startswith('bin/') else 0o644)
    # copytree preserves root-owned restrictive source modes. Give only the
    # disposable PHP clone its intended read/traverse permissions.
    for path in (ROOT / 'web').rglob('*'):
        os.chown(path, 0, grp.getgrnam('apache').gr_gid)
        os.chmod(path, 0o755 if path.is_dir() else 0o640)
    connect = ROOT / 'web/agc/dbconnect_mysqli.php'
    data = connect.read_bytes()
    assert data.count(b'/etc/astguiclient.conf') == 2
    connect.write_bytes(data.replace(b'/etc/astguiclient.conf', b'/opt/almadial-alm27/web.conf'))
    write(CONF, f'PATHhome => {ROOT}/bin\nPATHlogs => {ROOT}/logs\nPATHagi => /var/lib/asterisk/agi-bin\nPATHweb => {ROOT}/web\nPATHsounds => /var/lib/asterisk/sounds\nPATHmonitor => /var/spool/asterisk/monitor\nPATHDONEmonitor => /var/spool/asterisk/monitorDONE\nVARserver_ip => 127.0.0.1\nVARDB_server => localhost\nVARDB_database => {DB}\nVARDB_user => alm27\nVARDB_pass => {credential["database"]}\nVARDB_port => 3306\nExpectedDBSchema => 1748\n', group='asterisk')
    write(ROOT / 'web.conf', CONF.read_bytes(), group='apache')
    write(ROOT / 'ami.secret', credential['ami'] + '\n')
    write(AMI, '[alm27]\nsecret=' + credential['ami'] + '\ndeny=0.0.0.0/0.0.0.0\npermit=127.0.0.1/255.255.255.255\nread=system,call,agent\nwrite=call,originate,command\n')
    wrapper(Path('/etc/asterisk/manager.conf'), AMI, True)
    config = '; Temporary ALM-27 synthetic phones, loopback WSS only\n'
    for identity in ['9061', '9062']:
        config += f'[{identity}]\ntype=endpoint\ntransport=almadial-s10-wss\ncontext=alm27-agent\ndisallow=all\nallow=ulaw,alaw\nauth=alm27-auth-{identity}\naors={identity}\nwebrtc=yes\ndirect_media=no\nrtp_symmetric=yes\nforce_rport=yes\nrewrite_contact=yes\nmedia_address=10.160.90.233\nbind_rtp_to_media_address=yes\nrtp_port_start=10000\nrtp_port_end=10019\n[alm27-auth-{identity}]\ntype=auth\nauth_type=userpass\nusername={identity}\npassword={credential[identity]}\n[{identity}]\ntype=aor\nmax_contacts=1\nremove_existing=yes\n'
    write(PJSIP, config)
    wrapper(Path('/etc/asterisk/pjsip.conf'), PJSIP, True)
    write(WEB, f'''Listen 127.0.0.1:8443 https
<VirtualHost 127.0.0.1:8443>
ServerName localhost
SSLEngine on
SSLCertificateFile /etc/pki/tls/almadial-s08/current/fullchain.pem
SSLCertificateKeyFile /etc/pki/tls/almadial-s08/current/key.pem
DocumentRoot {ROOT}/web
Alias /alm27-agc {ROOT}/web/agc
Alias /alm27-phone {ROOT}/web/phone
DirectoryIndex index.php
<Directory "{ROOT}/web">
Require local
</Directory>
<FilesMatch "\\.php$">
SetHandler "proxy:unix:/run/php-fpm/www.sock|fcgi://localhost"
</FilesMatch>
</VirtualHost>
''', 0o644, 'root')
    run(['chcon', '-R', '-t', 'httpd_sys_content_t', str(ROOT / 'web')])
    run(['chcon', '-t', 'httpd_sys_content_t', str(ROOT / 'web.conf')])
    run(['restorecon', str(CONF), str(AMI), str(PJSIP), str(WEB)])
    run(['httpd', '-f', '/etc/httpd/conf/almadial-s05.conf', '-t'])
    for command in ['manager reload', 'module reload res_pjsip.so']:
        cli(command)
    run(['systemctl', 'reload', 'httpd'])
    routes = [('9600000', 1, 'Answer()'), ('9600000', 2, 'ConfBridge(9600000,default_bridge,default_user)'), ('29600000', 1, 'Answer()'), ('29600000', 2, 'ConfBridge(9600000,default_bridge,default_user)'), ('29600001', 1, 'Answer()'), ('29600001', 2, 'ConfBridge(9600001,default_bridge,default_user)'), ('9600001', 1, 'Answer()'), ('9600001', 2, 'ConfBridge(9600001,default_bridge,default_user)'), ('99062', 1, 'Answer()'), ('99062', 2, 'Dial(PJSIP/9062,30,oG(alm27-agent^customerjoin^1))'), ('customerjoin', 1, 'ConfBridge(9600000,default_bridge,default_user)'), ('customerjoin', 2, 'Hangup()')]
    # VICIdial's native Local_out_prefix defaults to 9 even for a blank campaign
    # prefix. Admit only this exact synthetic destination, never a dial pattern.
    routes += [('999062', 1, 'Goto(alm27-agent,99062,1)')]
    # Dial G sends caller to priority 1 and callee to priority 2. Both must
    # enter the conference; neither leg may land directly on Hangup.
    routes = [r for r in routes if r[0] != 'customerjoin'] + [
        ('customerjoin', 1, 'Goto(alm27-agent,customerjoin,3)'),
        ('customerjoin', 2, 'Goto(alm27-agent,customerjoin,3)'),
        ('customerjoin', 3, 'ConfBridge(9600000,default_bridge,default_user)'),
        ('customerjoin', 4, 'Hangup()')]
    for extension, priority, application in routes:
        cli('dialplan add extension ' + extension + ',' + str(priority) + ',' + application + ' into alm27-agent')
    for name in ['AST_update.pl', 'AST_manager_listen.pl']:
        run(['perl', '-c', str(ROOT / 'bin' / name)])
        run(['systemd-run', '--unit=alm27-' + name.split('.')[0], '--property=User=asterisk', '--property=RuntimeMaxSec=900', '--property=WorkingDirectory=' + str(ROOT / 'bin'), '/usr/bin/perl', str(ROOT / 'bin' / name)])
    rule = 'rule family="ipv4" source address="10.160.90.1/32" port port="10000-10019" protocol="udp" accept'
    run(['firewall-cmd', '--policy=almadial-s09', '--add-rich-rule=' + rule, '--timeout=600'])
    return {'prepared': True, 'database': DB, 'phones': ['9061', '9062'], 'campaign': 'ALM27', 'bounded_seconds': 900}


def cleanup():
    assert ROOT.exists() and json.loads((ROOT / 'authority.json').read_text())['machine'] == MACHINE
    for name in ['AST_update', 'AST_manager_listen']:
        subprocess.run(['systemctl', 'stop', 'alm27-' + name], capture_output=True, timeout=30)
    for line in cli('core show channels concise').splitlines():
        fields = line.split('!')
        assert len(fields) > 2 and (fields[1] == 'alm27-agent' or fields[0].startswith(('PJSIP/9061-', 'PJSIP/9062-', 'CBAnn/9600000-', 'CBAnn/9600001-'))), 'foreign active call prevents cleanup'
        cli('channel request hangup ' + fields[0])
    cli('dialplan remove context alm27-agent')
    if 'app_dial.so' in cli('module show like app_dial'):
        cli('module unload app_dial.so')
    target = Path('/usr/lib/asterisk/modules/app_dial.so')
    ownership = ROOT / 'dial-module.json'
    if ownership.exists() and target.exists():
        assert hashlib.sha256(target.read_bytes()).hexdigest() == json.loads(ownership.read_text())['sha256']
        target.unlink()
    rule = 'rule family="ipv4" source address="10.160.90.1/32" port port="10000-10019" protocol="udp" accept'
    q = subprocess.run(['firewall-cmd', '--policy=almadial-s09', '--query-rich-rule=' + rule], capture_output=True)
    if q.returncode == 0:
        run(['firewall-cmd', '--policy=almadial-s09', '--remove-rich-rule=' + rule])
    for path, include in [(Path('/etc/asterisk/manager.conf'), AMI), (Path('/etc/asterisk/pjsip.conf'), PJSIP)]:
        original = ROOT / (path.name + '.original')
        if original.exists():
            wrapper(path, include, False)
    for command in ['manager reload', 'module reload res_pjsip.so']:
        cli(command)
    for path in [CONF, AMI, PJSIP, WEB]:
        if path.exists():
            path.unlink()
    run(['httpd', '-f', '/etc/httpd/conf/almadial-s05.conf', '-t'])
    run(['systemctl', 'reload', 'httpd'])
    sql("DROP DATABASE IF EXISTS almadial_alm27; DROP USER IF EXISTS 'alm27'@'localhost';")
    shutil.rmtree(ROOT)
    return inspect()


def state():
    assert inspect()['present']
    rows = sql("SELECT JSON_ARRAY(man_id,status,action,channel,cmd_line_b,cmd_line_c,cmd_line_d,cmd_line_e) FROM almadial_alm27.vicidial_manager ORDER BY man_id").splitlines()
    return {'queue': [json.loads(x) for x in rows], 'channels': cli('core show channels concise'),
            'conference': cli('confbridge list'),
            'agents': sql("SELECT JSON_ARRAY(user,status,conf_exten,lead_id,uniqueid) FROM almadial_alm27.vicidial_live_agents"),
            'heartbeat_age': sql("SELECT TIMESTAMPDIFF(SECOND,last_update,NOW()) FROM almadial_alm27.server_updater WHERE server_ip='127.0.0.1'"),
            'reservations': sql("SELECT COUNT(*) FROM almadial_alm27.vicidial_confbridges WHERE extension!=''"),
            'contacts': cli('pjsip show contacts')}


def dispatch():
    # Dispatch just the first real UI queue request. No automatic campaign worker.
    pending = sql("SELECT JSON_ARRAY(man_id,action,channel,cmd_line_b,cmd_line_c,cmd_line_d,cmd_line_e,cmd_line_f,cmd_line_g,cmd_line_h,cmd_line_i,cmd_line_j,cmd_line_k) FROM almadial_alm27.vicidial_manager WHERE status IN ('NEW','QUEUE') ORDER BY man_id LIMIT 1").splitlines()
    if not pending:
        return {'dispatched': False}
    row = json.loads(pending[0])
    action = row[1]
    assert action in ('Originate', 'Redirect', 'Hangup', 'Command'), 'unapproved queue action'
    fields = row[2:]
    for item in fields:
        if item.startswith('Context:'):
            assert item == 'Context: alm27-agent', 'foreign context'
        if item.startswith('Channel:') or item.startswith('ExtraChannel:'):
            channel = item.split(': ', 1)[1]
            assert channel == 'Local/55559600000@alm27-agent' or channel.startswith(('PJSIP/9061-', 'PJSIP/9062-', 'Local/99062@alm27-agent-', 'Local/9600000@alm27-agent-', 'Local/9600001@alm27-agent-')) or channel in ('PJSIP/9061', 'PJSIP/9062', 'Local/99062@alm27-agent', 'Local/9600000@alm27-agent', 'Local/9600001@alm27-agent'), 'foreign channel'
        if item.startswith('Exten:'):
            assert item in ('Exten: 29600000', 'Exten: 9600000', 'Exten: 9600001', 'Exten: 99062', 'Exten: 999062', 'Exten: 8300'), 'foreign destination'
        if item.startswith('Command:'):
            assert item in ('Command: confbridge kick 9600000 all', 'Command: confbridge kick 9600001 all'), 'foreign command'
    run(['runuser', '-u', 'asterisk', '--', '/usr/bin/perl', str(ROOT / 'bin/AST_manager_send.pl'), '--sendonlyone', '--nocounteroutput'], timeout=25)
    return {'dispatched': True, 'manager_id': row[0], 'action': action}


if __name__ == '__main__':
    mode = sys.argv[1]
    assert mode in ('inspect', 'prepare', 'cleanup', 'state', 'dispatch')
    print(json.dumps(prepare(json.load(sys.stdin)) if mode == 'prepare' else globals()[mode](), sort_keys=True))
