"""Synthetic overlay contract tests; these never assert live browser acceptance."""
import importlib.util
import json
from pathlib import Path
import tempfile
import subprocess
import sqlite3
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location(
    'm007_assets', Path(__file__).resolve().parents[1] / 'lab/m0_07_assets.py')
assets = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(assets)

PHONE = b'<?php $debug_access_log = true; ?>'
JS = b'''logBuiltinEnabled: true; logConfiguration: true; logLevel: "debug"; traceSip: true;
function registerButton( ) {
	debugOut( 'Register Button Pressed' );

	// Send REGISTER
	myRegisterer.register()
        .then(handleRegRegistering).catch(handleRegFailed);
    return true;
}
function handleRegRegistered() {
    uiElements.reg_status.value = incall ? lang.incall + ' - ' + caller : 'Registered';
}
'''
SIP = b'// Synthetic dependency, not upstream SIP.js\n'
AGENT = b'''<?php
$WebPhonEurl = "$webphone_url?phone_login=$b64_phone_login&phone_login=$b64_phone_login&phone_pass=$b64_phone_pass&server_ip=$b64_server_ip&callerid=$b64_callerid&protocol=$b64_protocol&codecs=$b64_codecs&options=$b64_options&system_key=$b64_system_key";
echo "<iframe src=\\"$WebPhonEurl\\"></iframe>";
echo "<iframe src=\\"$WebPhonEurl\\"></iframe>";
$no_delete_sessions\t\t= '1';
$user_abb = preg_replace("/^\\./i","",$user_abb);
values('$VD_login','$server_ip','$session_id','$SIP_user','PAUSED','','$VD_campaign');
values('$VD_login','$server_ip','$session_id','$SIP_user','PAUSED','','$VD_campaign');
document.getElementById('InternalChatIFrame').src='./agc_agent_manager_chat_interface.php?user='+user+'&pass='+pass;
document.getElementById('InternalChatIFrame').src='./agc_agent_manager_chat_interface.php?lead_id=' + document.vicidial_form.lead_id.value + '&list_id=' + form_list_id + '&user=' + user + '&pass=' + pass;
document.getElementById('vcEmailIFrame').src='./vdc_email_display.php?user=' + user + '&pass=' + pass;
document.getElementById('CustomerChatIFrame').src='./vdc_chat_display.php?user=' + user + '&pass=' + pass;
\tfunction InternalChatContentsLoad(ICHrefresh) {}
?>'''
DB = b'''<?php
$link=mysqli_connect("$VARDB_server", "$VARDB_user", "$VARDB_pass", "$VARDB_database", $VARDB_port);
?>'''


class AssetOverlayTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / 'source'
        (self.source / 'js').mkdir(parents=True)
        self.data = {'viciphone.php': PHONE, 'js/vici_phone.js': JS, 'js/sip.js': SIP}
        for name, data in self.data.items():
            (self.source / name).write_bytes(data)
        (self.source / 'COPYING').write_text('Synthetic license notice\n')
        self.license = self.root / 'LICENSE'
        self.license.write_text('Synthetic repository-root license\n')
        self.agent, self.db = self.root / 'agent.php', self.root / 'db.php'
        self.agent.write_bytes(AGENT)
        self.db.write_bytes(DB)
        self.output = self.root / 'output'
        for attribute, value in (
                ('PINS', {name: assets.digest(data) for name, data in self.data.items()}),
                ('AGENT_PIN', assets.digest(AGENT)), ('DBCONNECT_PIN', assets.digest(DB)),
                ('LICENSE_PIN', assets.digest(self.license.read_bytes()))):
            mock = patch.object(assets, attribute, value)
            mock.start()
            self.addCleanup(mock.stop)

    def prepare(self, output=None):
        return assets.prepare(self.source, output or self.output, self.agent, self.db)

    def test_source_immutable_and_dependency_notice_preserved(self):
        self.prepare()
        for name, data in self.data.items():
            self.assertEqual((self.source / name).read_bytes(), data)
        self.assertEqual(self.agent.read_bytes(), AGENT)
        self.assertEqual(self.db.read_bytes(), DB)
        self.assertEqual((self.output / 'viciphone/COPYING').read_bytes(),
                         (self.source / 'COPYING').read_bytes())
        self.assertEqual((self.output / 'viciphone/js/sip.js').read_bytes(), SIP)
        self.assertEqual((self.output / 'viciphone/LICENSE').read_bytes(), self.license.read_bytes())

    def test_phone_debug_disabled(self):
        self.prepare()
        php = (self.output / 'viciphone/viciphone.php').read_text()
        js = (self.output / 'viciphone/js/vici_phone.js').read_text()
        self.assertIn('$debug_access_log = false;', php)
        for marker in ('logBuiltinEnabled: false', 'logConfiguration: false',
                       'logLevel: "error"', 'traceSip: false'):
            self.assertIn(marker, js)

    def test_agent_post_replaces_both_frames_and_escapes_fields(self):
        result = assets.post_iframe_overlay(AGENT).decode()
        self.assertNotIn('$webphone_url?phone_login=', result)
        self.assertIn('$WebPhonEurl = $webphone_url;', result)
        self.assertEqual(result.count('srcdoc=\\"$WebPhonEpost\\"'), 2)
        self.assertNotIn('src=\\"$WebPhonEurl\\"', result)
        self.assertIn("htmlspecialchars($webphone_url, ENT_QUOTES, 'UTF-8')", result)
        self.assertIn("htmlspecialchars($alm7_value, ENT_QUOTES, 'UTF-8')", result)
        self.assertIn("htmlspecialchars($alm7_form, ENT_QUOTES, 'UTF-8')", result)
        self.assertIn('<form method="post"', result)

    def test_connection_preserves_strict_mode_byte_for_byte(self):
        self.assertEqual(assets.connection_overlay(DB), DB)

    def test_live_agent_lead_is_zero_and_conference_is_released(self):
        result = assets.agent_lifecycle_overlay(AGENT).decode()
        self.assertEqual(result.count("'PAUSED',0,'$VD_campaign'"), 2)
        self.assertIn("$no_delete_sessions\t\t= '0';", result)
        self.assertNotIn('sql_mode', result)

    def test_manual_recovery_uses_transport_then_register_and_handles_same_state(self):
        result = assets.phone_overlay('js/vici_phone.js', JS).decode()
        functions = result[result.index('function registerButton'):]
        script = '''
const SIP={TransportState:{Disconnected:'Disconnected'}};
const uiElements={reg_status:{value:''}}, lang={incall:'Incall'};
let incall=false, caller='9600000', events=[];
function debugOut(){} function handleRegRegistering(){events.push('request');}
function handleRegFailed(){events.push('failed');}
const myUserAgent={transport:{state:'Disconnected'}, reconnect:async()=>{events.push('reconnect');}};
const myRegisterer={register:async(options)=>{events.push('register');
    setTimeout(()=>options.requestDelegate.onAccept(),0);}};
''' + functions + '''
(async()=>{
 registerButton(); await new Promise(r=>setTimeout(r,10));
 if(events.join(',')!=='reconnect,register,request'||uiElements.reg_status.value!=='Registered')throw Error('idle recovery');
 events=[];incall=true;myUserAgent.transport.state='Connected';
 registerButton();await new Promise(r=>setTimeout(r,10));
 if(events.join(',')!=='register,request'||uiElements.reg_status.value!=='Incall - 9600000')throw Error('active refresh');
 events=[];myUserAgent.transport.state='Disconnected';myUserAgent.reconnect=async()=>{throw Error('offline')};
 registerButton();await new Promise(r=>setTimeout(r,10));
 if(events.join(',')!=='failed')throw Error('failed reconnect must not REGISTER');
})().catch(e=>{console.error(e);process.exitCode=1});
'''
        subprocess.run(['node', '-e', script], check=True, capture_output=True)

    def test_agent_call_id_fits_queue_and_panel_posts_preserve_duplicate_fields(self):
        result = assets.agent_lifecycle_overlay(AGENT).decode()
        statement = next(line for line in result.splitlines() if '$user_abb = substr' in line)
        subprocess.run(['php', '-r', '$user_abb="long-agent-name";' + statement +
                        'if($user_abb!=="name"||strlen("LPvdcW"."1793980374".$user_abb)!==20)exit(1);'],
                       check=True, capture_output=True)
        helper = result[result.index('    function alm7_agent_panel_post'):result.index('    function alm7_internal_chat_post')]
        script = '''let sent;const document={
 createElement:tag=>({children:[],appendChild(x){this.children.push(x)},submit(){sent=this},remove(){}}),
 body:{appendChild(){}}};
''' + helper + '''
alm7_agent_panel_post('CustomerChatIFrame','./vdc_chat_display.php','user=fixture&pass=example%26value&session_id=&session_id=123');
if(sent.method!=='POST'||sent.action.includes('?')||sent.target!=='CustomerChatIFrame')throw Error('navigation');
if(JSON.stringify(sent.children.map(x=>[x.name,x.value]))!==JSON.stringify([['user','fixture'],['pass','example&value'],['session_id',''],['session_id','123']]))throw Error('fields');
'''
        subprocess.run(['node', '-e', script], check=True, capture_output=True)

    def session_source(self, name):
        source = "SELECT count(*) from web_client_sessions where session_name='$session_name' and server_ip='$server_ip';"
        if name == 'vdc_db_query.php':
            source += "\ndate_of_birth='\" . mysqli_real_escape_string($link, $date_of_birth) . \"'"
            source += "\nif ($ACTION == 'manDiaLlogCaLL')\n{\n// start and end share this operation\n}"
        if name == 'conf_exten_check.php':
            source += "\nvisibility='$visibility_type',agent_log_id='$agent_log_id';"
        count = {'vdc_db_query.php': 19, 'manager_send.php': 22, 'conf_exten_check.php': 0}[name]
        return (source + "\nINSERT INTO vicidial_manager values('',''," * count).encode()

    def test_logout_session_authorization_and_fresh_login(self):
        source = self.session_source('conf_exten_check.php')
        with patch.dict(assets.SESSION_PINS, {'conf_exten_check.php': assets.digest(source)}):
            query = assets.session_overlay('conf_exten_check.php', source).decode().splitlines()[0]
        query = query.replace("'$session_name'", ':session').replace("'$server_ip'", ':server').replace("'$user'", ':user')
        db = sqlite3.connect(':memory:')
        self.addCleanup(db.close)
        db.executescript('''CREATE TABLE web_client_sessions(session_name,server_ip);
CREATE TABLE vicidial_session_data(session_name,server_ip,user,conf_exten);
CREATE TABLE vicidial_live_agents(user,server_ip,conf_exten);
INSERT INTO web_client_sessions VALUES('old','loopback'),('active','loopback');
INSERT INTO vicidial_session_data VALUES('active','loopback','fixture','9600000');
INSERT INTO vicidial_live_agents VALUES('fixture','loopback','9600000');''')
        def authorized(session, user='fixture'):
            return db.execute(query, {'session': session, 'server': 'loopback', 'user': user}).fetchone()[0]
        self.assertEqual(authorized('active'), 1)
        self.assertEqual(authorized('old'), 0)
        self.assertEqual(authorized('active', 'other'), 0)
        db.execute('DELETE FROM vicidial_live_agents')
        self.assertEqual(authorized('active'), 0)
        db.execute("INSERT INTO vicidial_live_agents VALUES('fixture','loopback','9600001')")
        self.assertEqual(authorized('active'), 0)
        db.execute("INSERT INTO web_client_sessions VALUES('fresh','loopback')")
        db.execute("INSERT INTO vicidial_session_data VALUES('fresh','loopback','fixture','9600001')")
        self.assertEqual(authorized('fresh'), 1)

    def test_strict_sql_typed_values_and_operation_scope(self):
        for name in assets.SESSION_PINS:
            source = self.session_source(name)
            with patch.dict(assets.SESSION_PINS, {name: assets.digest(source)}):
                result = assets.session_overlay(name, source).decode()
            self.assertNotIn('sql_mode', result)
            self.assertNotIn("INSERT INTO vicidial_manager values('','',", result)
            if name == 'vdc_db_query.php':
                operation = result[result.index("if ($ACTION == 'manDiaLlogCaLL')"):]
                self.assertIn("if ($alt_dial === 'MANUAL_DIALNOW') {$alt_dial = 'MANUAL';}", operation)
                self.assertIn("date_of_birth=NULLIF('", result)
                self.assertIn(",'')", result)
            if name == 'conf_exten_check.php':
                self.assertIn('ON DUPLICATE KEY UPDATE event_end_epoch=VALUES(event_end_epoch)', result)
                self.assertNotIn('INSERT IGNORE', result)

    def test_listener_handles_modern_event_fields_without_changing_other_versions(self):
        block = '@command_line=split(/\\n/, $input_lines[$ILcount]);'
        source = ("'/etc/astguiclient.conf'\n" + (block + '\n') * 4).encode()
        with patch.dict(assets.CALL_RUNTIME_PINS, {'AST_manager_listen.pl': assets.digest(source)}):
            result = assets.call_runtime_overlay('AST_manager_listen.pl', source).decode()
        fragment = result.splitlines()[1:]
        fragment = '\n'.join(fragment)
        script = '''$asterisk_version='22.12.0';$ILcount=0;
$input_lines[0]="Event: Hangup\\nPrivilege: call,all\\nChannel: PJSIP/9052-01\\nChannelState: 6\\nCallerIDNum: fixture\\nUniqueid: 123.4\\nCause: 16\\nCause-txt: Normal Clearing\\n";
''' + fragment + '''
die 'channel lost' unless $command_line[2] eq 'Channel: PJSIP/9052-01';
die 'uniqueid position' unless $command_line[3] eq 'Uniqueid: 123.4';
'''
        subprocess.run(['perl', '-e', script], check=True, capture_output=True)

    def test_runtime_sources_reject_unpinned_inputs(self):
        for name in assets.CALL_RUNTIME_PINS:
            with self.assertRaisesRegex(ValueError, 'source mismatch'):
                assets.call_runtime_overlay(name, b'changed')
        for name in assets.MANAGER_PINS:
            with self.assertRaisesRegex(ValueError, 'source mismatch'):
                assets.manager_overlay(name, b'changed')

    def test_updater_asterisk22_framing_preserves_real_pjsip_channel(self):
        source = ("'/etc/astguiclient.conf'\n"
                  'if ($AST_ver =~ /^12\\./i) {$show_channels_format = 4;}\n'
                  '$show_channels_format = 1;\n' +
                  "Prompt => '/--END COMMAND-.*/'\n" * 3 +
                  '\t\t##### TEST CHANNELS ZAP/IAX2/Local TO SEE IF THERE WAS A LARGE HICCUP IN OUTPUT FROM PREVIOUS OUTPUT\n' +
                  "where protocol='SIP' and server_ip='$server_ip'\n" * 2 +
                  '/^Zap|^IAX2|^SIP|^Local|^DAHDI/\n' * 2 +
                  '/^SIP|^Zap|^IAX2|^DAHDI/\n' + '/^SIP/\n' * 2 +
                  's/^SIP\\///gi\n' +
                  '$stmtB = "SELECT count(*) FROM $live_channels where server_ip=\'$server_ip\' and channel=\'$PQchannel\' and ( (extension=\'$PQextension\') or (extension LIKE \\"%.agi\\") );";\n').encode()
        with patch.object(assets, 'UPDATER_PIN', assets.digest(source)):
            derived = assets.updater_overlay(source).decode()
        self.assertEqual(derived.count("protocol IN ('SIP','PJSIP')"), 2)
        fragment = derived[derived.index('        if (($AST_ver'):derived.index('\t\t##### TEST CHANNELS')]
        script = '''$AST_ver='22.12.0';
@list_channels=("Response: Success\\n","Message: Command output follows\\n","Output: PJSIP/9051-0001!context!9600000!1!Up!ConfBridge!9600000\\n","Output: 1 active channel\\n");
''' + fragment + '''
die 'missing framing' unless $list_channels[0] eq "Response: Follows\\n";
die 'channel identity lost' unless $list_channels[2] =~ /^PJSIP\\/9051-0001!/;
die 'missing terminator' unless $list_channels[-1] eq "--END COMMAND--\\n";
$AST_ver='18.0'; @list_channels=("Message: Command output follows\\n","Output: untouched\\n");
''' + fragment + '''
die 'other version modified' unless $list_channels[1] eq "Output: untouched\\n";
'''
        subprocess.run(['perl', '-e', script], check=True, capture_output=True)
        park = derived[derived.index('$stmtB ='):]
        script = '''$server_ip='127.0.0.1'; $live_channels='live_channels';
$live_sip_channels='live_sip_channels'; $PQextension='park';
$AST_ver='22.12.0'; $PQchannel='PJSIP/9052-test';
''' + park + '''
die 'PJSIP park not checked in client table' unless $stmtB =~ /FROM live_sip_channels/;
die 'must check actual park AGI' unless $stmtB =~ /channel_data='park_CID.agi'/;
$AST_ver='18.0';
''' + park + '''
die 'other version changed' unless $stmtB =~ /FROM live_channels/;
$AST_ver='22.12.0'; $PQchannel='Local/test';
''' + park + '''
die 'other channel changed' unless $stmtB =~ /FROM live_channels/;
'''
        subprocess.run(['perl', '-e', script], check=True, capture_output=True)

    def test_calendar_scroll_without_instances_and_existing_instances(self):
        source = b'function f_tcal_hideAll () {\r\nfor(var i=0;i<window.A_TCALSIDX.length;i++) window.A_TCALSIDX[i].f_hide();\r\n}'
        with patch.object(assets, 'CALENDAR_PIN', assets.digest(source)):
            result = assets.calendar_overlay(source)
        script = ('const window={};' + result.decode() + ';f_tcal_hideAll();'
                  'let count=0;window.A_TCALSIDX=[{f_hide(){count++}},{f_hide(){count++}}];'
                  'f_tcal_hideAll();if(count!==2)throw Error("calendar hide broken");')
        subprocess.run(['node', '-e', script], check=True, capture_output=True)
        self.assertEqual(result.count(b'\r\n'), source.count(b'\r\n') + 1)

    def test_lifecycle_source_pin_rejection(self):
        for function in [assets.calendar_overlay, assets.updater_overlay]:
            with self.assertRaisesRegex(ValueError, 'source mismatch'):
                function(b'unreviewed source')

    def test_unexpected_live_agent_insert_count_rejected(self):
        source = AGENT.replace(b"'PAUSED','','", b"'READY','','", 1)
        with patch.object(assets, 'AGENT_PIN', assets.digest(source)):
            with self.assertRaisesRegex(ValueError, 'two reviewed live-agent'):
                assets.agent_lifecycle_overlay(source)

    def test_each_changed_pin_fails_before_output_creation(self):
        for file in [self.source / name for name in self.data] + [self.agent, self.db, self.license]:
            with self.subTest(file=file.name):
                original = file.read_bytes()
                file.write_bytes(original + b'changed')
                with self.assertRaisesRegex(ValueError, 'mismatch'):
                    self.prepare()
                self.assertFalse(self.output.exists())
                file.write_bytes(original)

    def test_marker_absent_or_ambiguous_fails(self):
        for data in (b'no marker', PHONE + PHONE):
            with self.subTest(data=data):
                with patch.object(assets, 'PINS', {'viciphone.php': assets.digest(data)}):
                    with self.assertRaisesRegex(ValueError, 'exactly one'):
                        assets.phone_overlay('viciphone.php', data)

    def test_unexpected_frame_count_rejected(self):
        data = AGENT.replace(b'src=\\"$WebPhonEurl\\"', b'src="other"', 1)
        with patch.object(assets, 'AGENT_PIN', assets.digest(data)):
            with self.assertRaisesRegex(ValueError, 'two reviewed'):
                assets.post_iframe_overlay(data)

    def test_repeated_overlay_is_identical_and_offline_only(self):
        first = self.prepare()
        second_path = self.root / 'second'
        self.assertEqual(first, self.prepare(second_path))
        self.assertEqual((self.output / 'asset-manifest.json').read_bytes(),
                         (second_path / 'asset-manifest.json').read_bytes())
        manifest = json.loads((self.output / 'asset-manifest.json').read_text())
        self.assertEqual(manifest['live_runtime'], 'NOT_RUN')
        for name, expected in manifest['files'].items():
            self.assertEqual(assets.digest((self.output / name).read_bytes()), expected)
        self.assertEqual(self.output.stat().st_mode & 0o777, 0o700)

    def test_existing_destination_preserved(self):
        self.output.mkdir()
        sentinel = self.output / 'operator-data'
        sentinel.write_bytes(b'keep')
        with self.assertRaises(FileExistsError):
            self.prepare()
        self.assertEqual(sentinel.read_bytes(), b'keep')

    def test_source_and_output_symlinks_rejected(self):
        link = self.source / 'linked'
        link.symlink_to(self.agent)
        with self.assertRaisesRegex(ValueError, 'symlink'):
            self.prepare()
        link.unlink()
        self.output.symlink_to(self.root / 'nonexistent')
        with self.assertRaises(FileExistsError):
            self.prepare()
        self.assertFalse((self.root / 'nonexistent').exists())

    def test_agent_symlink_rejected(self):
        link = self.root / 'linked-agent'
        link.symlink_to(self.agent)
        with self.assertRaisesRegex(ValueError, 'symlink'):
            assets.prepare(self.source, self.output, link)
        self.assertFalse(self.output.exists())


if __name__ == '__main__':
    unittest.main()
