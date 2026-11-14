#!/usr/bin/env python3
"""Reviewed r4037 compatibility derivations reused by the native runtime handoff.

Does not fetch, publish, execute PHP, change services, or assert live acceptance.
Upstream inputs are immutable. Output directory must not exist.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil

PINS = {
    'viciphone.php': 'e37d92df58d3c4f1b55a24a4364f9e4c9653c45820628fb657c42bd443fb6a6c',
    'js/vici_phone.js': 'c7763fd3cc5a56d604c577291e6bbd25afe7f3b1cfa5fadbce914d30cf94779b',
    'js/sip.js': '9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b',
}
AGENT_PIN = 'f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358'
LICENSE_PIN = 'e7ae8b5da17709474ba25316441bc064148841cf101e7eef318c3c3ee5a0827b'
# Isolated ALM-7 dbconnect input, already pointing at the lab-only config file.
# This is a derivation pin, not an assertion that upstream uses non-strict SQL.
DBCONNECT_PIN = '6cce68c7e7ecd919d7906476fb332e8978b7906224ae878b302c17de98d41a15'
CALENDAR_PIN = 'ee38674284705a761ee64a237efd9744095ef0e0d7da993cad3aed7f122ee7e7'
UPDATER_PIN = 'ee0e5a752d47ababb0db4657a0825757b62fd41433bc4a04a4ed7ea968b8ef3b'
SESSION_PINS = {
    'vdc_db_query.php': '5aaeabd1d9b247dc563b9e94bb86d0b3323346b226b52d47a00ca2506897108d',
    'conf_exten_check.php': '1dcff537c57b61e2d8ea1bdc3770463fcc27a554982a39a444cea3fc3e628316',
    'manager_send.php': '8c53bbc25d595b0e0b982f06ad1eaf083ababb13ccb59d3678fa0f8252852a6a',
}
MANAGER_PINS = {
    'AST_manager_send.pl': '83be4e5ca25591adc6991273b84114b53164ef49b7bf04bee9e9b7413b7cb384',
    'AST_send_action_child.pl': '46641e4f67c91122b520a932559e3b4f162609360cd4d15e38d9b7bf8bad15c9',
}
CALL_RUNTIME_PINS = {
    'AST_manager_listen.pl': '4111ce1d9714ee56c0eb243779bad94eb62f6ca8cfe23a3c229f661b08d587dd',
    'park_CID.agi': 'a41e746a0e5f87261f1510751321e7e4131703f2876dfb3351de99c2246e2b20',
}
CALL_RUNTIME_INPUTS = {
    'AST_manager_listen.pl': 'bin/AST_manager_listen.pl',
    'park_CID.agi': 'agi/park_CID.agi',
    'Asterisk.pm': 'extras/Asterisk.pm',
    'Asterisk/AGI.pm': 'extras/Asterisk/AGI.pm',
}
AGI_MODULE_PINS = {
    'Asterisk.pm': '26f266508ef3c9087bb19cad34b51c30a50853b60550e14626d2a7544c865915',
    'Asterisk/AGI.pm': '38f4e84f50e93e53d63b279584f5fca04553528c7660022d4b082aac30314c9a',
}


def call_runtime_overlay(name, data):
    """Bounded upstream listener/park process, with isolated configuration."""
    if r4037_digest(data) != CALL_RUNTIME_PINS[name]:
        raise ValueError('Pinned r4037 call runtime source mismatch: ' + name)
    text = replace_once(data.decode(), "'/etc/astguiclient.conf'", "'/etc/astguiclient-m007-updater.conf'")
    if name == 'park_CID.agi':
        # The lab uses pinned upstream Asterisk AGI modules in its own directory.
        text = replace_once(text, 'use Asterisk::AGI;', "use lib '/opt/almadial-m007';\nuse Asterisk::AGI;")
    else:
        old = '@command_line=split(/\\n/, $input_lines[$ILcount]);'
        if text.count(old) != 4:
            raise ValueError('Unexpected listener event parser count')
        text = text.replace(old, '''if ($asterisk_version =~ /^22\\./ && $input_lines[$ILcount] =~ /^Event: (Newstate|Hangup)\\n/m) {
                            my %alm7_fields = map { split(/: /, $_, 2) } split(/\\n/, $input_lines[$ILcount]);
                            my @alm7_order = $alm7_fields{Event} eq 'Newstate'
                              ? qw(Event Privilege Channel ChannelState ChannelStateDesc CallerIDNum CallerIDName ConnectedLineNum ConnectedLineName Uniqueid)
                              : qw(Event Privilege Channel Uniqueid Cause Cause-txt);
                            $input_lines[$ILcount] = join("\\n", map { $_ . ': ' . ($alm7_fields{$_} // '') } @alm7_order);
                        }
                        ''' + old)
    return text.encode()


def manager_overlay(name, data):
    """Use the upstream single-action sender without an AMI secret in argv/logs."""
    if r4037_digest(data) != MANAGER_PINS[name]:
        raise ValueError('Pinned r4037 manager source mismatch: ' + name)
    text = data.decode()
    if name == 'AST_manager_send.pl':
        text = replace_once(text, "'/etc/astguiclient.conf'", "'/etc/astguiclient-m007-updater.conf'")
        text = replace_once(text, '\t\t\t\t\t$launch .= " --ASTmgrSECRET=" . $servConf->{ASTmgrSECRET};', '')
    else:
        text = replace_once(text, "\t\t'ASTmgrSECRET=s' => \\$ASTmgrSECRET,", '')
        old = '\t\tprint "  ASTmgrSECRET:          $ASTmgrSECRET\\n" if ($ASTmgrSECRET);'
        if text.count(old) != 2:
            raise ValueError('Unexpected child credential debug marker')
        text = text.replace(old, '')
        text = replace_once(text, 'if ($action) {', '''# AlmaDial bounded lab: a protected local file replaces the secret argv option.
open(my $alm7_secret, '<', '/etc/almadial-m007-ami.secret') or die "AMI secret file unavailable";
$ASTmgrSECRET = <$alm7_secret>;
close($alm7_secret);
$ASTmgrSECRET =~ s/[\\r\\n]+$//;
die "AMI secret file empty" unless length($ASTmgrSECRET);

if ($action) {''')
    return text.encode()


def session_overlay(name, data):
    """Retained failed-login rows cannot authorize a current agent operation."""
    if r4037_digest(data) != SESSION_PINS[name]:
        raise ValueError('Pinned r4037 session source mismatch: ' + name)
    old = "SELECT count(*) from web_client_sessions where session_name='$session_name' and server_ip='$server_ip';"
    new = ("SELECT count(*) FROM web_client_sessions w "
           "JOIN vicidial_session_data s ON s.session_name=w.session_name AND s.server_ip=w.server_ip "
           "JOIN vicidial_live_agents a ON a.user=s.user AND a.server_ip=s.server_ip AND a.conf_exten=s.conf_exten "
           "WHERE w.session_name='$session_name' AND w.server_ip='$server_ip' AND s.user='$user';")
    text = replace_once(data.decode(), old, new)
    if name == 'conf_exten_check.php':
        # Concurrent visibility batches can repeat (user, visibility, end epoch).
        # Preserve the first audit observation on that unique-key collision;
        # other strict SQL failures must still fail. No INSERT IGNORE/sql_mode.
        audit = "visibility='$visibility_type',agent_log_id='$agent_log_id';"
        text = replace_once(text, audit,
                            "visibility='$visibility_type',agent_log_id='$agent_log_id' "
                            "ON DUPLICATE KEY UPDATE event_end_epoch=VALUES(event_end_epoch);")
    if name == 'vdc_db_query.php':
        # The UI sends a workflow label; alt_dial is the six-character dial type.
        old_dob = "date_of_birth='\" . mysqli_real_escape_string($link, $date_of_birth) . \"'"
        text = replace_once(text, old_dob, "date_of_birth=NULLIF('\" . mysqli_real_escape_string($link, $date_of_birth) . \"','')")
        marker = "if ($ACTION == 'manDiaLlogCaLL')\n{"
        text = replace_once(text, marker, marker + "\n\tif ($alt_dial === 'MANUAL_DIALNOW') {$alt_dial = 'MANUAL';}")
    # Strict MariaDB rejects '' for the queue's auto-increment primary key.
    # NULL requests allocation regardless of NO_AUTO_VALUE_ON_ZERO.
    counts = {'vdc_db_query.php': 19, 'manager_send.php': 22, 'conf_exten_check.php': 0}
    old = "INSERT INTO vicidial_manager values('','',"
    if text.count(old) != counts[name]:
        raise ValueError('Unexpected manager queue insert count: ' + name)
    return text.replace(old, "INSERT INTO vicidial_manager values(NULL,'',").encode()


def r4037_digest(data):
    return hashlib.sha256(data).hexdigest()


def replace_once(text, old, new):
    if text.count(old) != 1:
        raise ValueError('Expected exactly one reviewed source marker')
    return text.replace(old, new, 1)


def phone_overlay(name, data):
    if r4037_digest(data) != PINS[name]:
        raise ValueError('Pinned VICIphone source mismatch: ' + name)
    text = data.decode()
    if name == 'viciphone.php':
        text = replace_once(text, '$debug_access_log = true;', '$debug_access_log = false;')
    elif name == 'js/vici_phone.js':
        for old, new in [('logBuiltinEnabled: true', 'logBuiltinEnabled: false'),
                         ('logConfiguration: true', 'logConfiguration: false'),
                         ('logLevel: "debug"', 'logLevel: "error"'),
                         ('traceSip: true', 'traceSip: false')]:
            text = replace_once(text, old, new)
        # The existing user-facing Register control owns manual recovery.
        text = replace_once(text, "function registerButton( ) {\n\tdebugOut( 'Register Button Pressed' );\n\n\t// Send REGISTER\n\tmyRegisterer.register()",
                            "function registerButton( ) {\n\tdebugOut( 'Register Button Pressed' );\n\n\t// SIP.js requires a connected transport before REGISTER.\n\tconst ready = myUserAgent.transport.state === SIP.TransportState.Disconnected\n\t\t? myUserAgent.reconnect() : Promise.resolve();\n\tready.then(() => myRegisterer.register({requestDelegate: {onAccept: handleRegRegistered}}))")
        # A refresh can succeed while RegistererState remains Registered.
        text = replace_once(text, 'function handleRegRegistered() {',
                            "function handleRegRegistered() {\n\tif (incall) uiElements.reg_status.value = lang.incall + ' - ' + caller;")
    return text.encode()


def post_iframe_overlay(data):
    """Keep SIP credentials out of URLs/session URL rows, using upstream POST input."""
    if r4037_digest(data) != AGENT_PIN:
        raise ValueError('Pinned r4037 agent source mismatch')
    text = data.decode()
    old = '$WebPhonEurl = "$webphone_url?phone_login=$b64_phone_login&phone_login=$b64_phone_login&phone_pass=$b64_phone_pass&server_ip=$b64_server_ip&callerid=$b64_callerid&protocol=$b64_protocol&codecs=$b64_codecs&options=$b64_options&system_key=$b64_system_key";'
    new = '''$WebPhonEurl = $webphone_url;
                // AlmaDial r4037 lab overlay: upstream VICIphone accepts POST.
                $alm7_fields = array('phone_login'=>$b64_phone_login, 'phone_pass'=>$b64_phone_pass,
                    'server_ip'=>$b64_server_ip, 'callerid'=>$b64_callerid, 'protocol'=>$b64_protocol,
                    'codecs'=>$b64_codecs, 'options'=>$b64_options, 'system_key'=>$b64_system_key);
                $alm7_form = '<form method="post" action="'.htmlspecialchars($webphone_url, ENT_QUOTES, 'UTF-8').'">';
                foreach ($alm7_fields as $alm7_name=>$alm7_value) {
                    $alm7_form .= '<input type="hidden" name="'.$alm7_name.'" value="'.htmlspecialchars($alm7_value, ENT_QUOTES, 'UTF-8').'">';
                }
                $alm7_form .= '</form><script>document.forms[0].submit();</script>';
                $WebPhonEpost = htmlspecialchars($alm7_form, ENT_QUOTES, 'UTF-8');'''
    text = replace_once(text, old, new)
    marker = 'src=\\"$WebPhonEurl\\"'
    if text.count(marker) != 2:
        raise ValueError('Expected two reviewed embedded phone frames')
    return text.replace(marker, 'srcdoc=\\"$WebPhonEpost\\"').encode()


def connection_overlay(data):
    """Preserve strict SQL on the isolated connection; fix typed values at source."""
    if r4037_digest(data) != DBCONNECT_PIN:
        raise ValueError('Pinned isolated dbconnect source mismatch')
    return data


def agent_lifecycle_overlay(data):
    """Derive the reviewed r4037 login and conference-release settings only."""
    text = post_iframe_overlay(data).decode()
    old = "values('$VD_login','$server_ip','$session_id','$SIP_user','PAUSED','','$VD_campaign'"
    if text.count(old) != 2:
        raise ValueError('Expected two reviewed live-agent inserts')
    text = text.replace(old, old.replace("'PAUSED','','", "'PAUSED',0,'"))
    text = replace_once(text, "$no_delete_sessions\t\t= '1';", "$no_delete_sessions\t\t= '0';")
    # hideDiv('InternalChatPanel') refreshes this frame even with chats disabled.
    # The upstream endpoint already accepts POST; keep its credentials out of URLs.
    text = replace_once(text, "document.getElementById('InternalChatIFrame').src='./agc_agent_manager_chat_interface.php?user='+user+'&pass='+pass;",
                        "alm7_internal_chat_post({user:user,pass:pass});")
    text = replace_once(text, "document.getElementById('InternalChatIFrame').src='./agc_agent_manager_chat_interface.php?lead_id=' + document.vicidial_form.lead_id.value + '&list_id=' + form_list_id + '&user=' + user + '&pass=' + pass;",
                        "alm7_internal_chat_post({user:user,pass:pass,lead_id:document.vicidial_form.lead_id.value,list_id:form_list_id});")
    marker = '\tfunction InternalChatContentsLoad(ICHrefresh)'
    text = replace_once(text, marker, '''    function alm7_internal_chat_post(fields) {
        var form=document.createElement('form');
        form.method='POST'; form.action='./agc_agent_manager_chat_interface.php';
        form.target='InternalChatIFrame';
        Object.keys(fields).forEach(function(name) {
            var input=document.createElement('input'); input.type='hidden';
            input.name=name; input.value=fields[name]; form.appendChild(input);
        });
        document.body.appendChild(form); form.submit(); form.remove();
    }
''' + marker)
    # Disposition/logout refresh these panels even when email/chat are disabled.
    for frame, endpoint in [('vcEmailIFrame', 'vdc_email_display.php'),
                            ('CustomerChatIFrame', 'vdc_chat_display.php')]:
        prefix = "document.getElementById('" + frame + "').src='./" + endpoint + "?"
        lines = [line for line in text.splitlines() if prefix in line]
        if len(lines) != 1 or not lines[0].endswith(';'):
            raise ValueError('Unexpected agent panel navigation: ' + frame)
        old = lines[0]
        query_expression = old.split(prefix, 1)[1][:-1]
        text = replace_once(text, old, "\t\talm7_agent_panel_post('" + frame + "','./" + endpoint + "','" + query_expression + ');')
    text = replace_once(text, '    function alm7_internal_chat_post(fields) {', '''    function alm7_agent_panel_post(frame, endpoint, query) {
        var form=document.createElement('form');
        form.method='POST'; form.action=endpoint; form.target=frame;
        new URLSearchParams(query).forEach(function(value,name) {
            var input=document.createElement('input'); input.type='hidden';
            input.name=name; input.value=value; form.appendChild(input);
        });
        document.body.appendChild(form); form.submit(); form.remove();
    }
    function alm7_internal_chat_post(fields) {''')
    # r4037 accidentally matches a literal dot while trying to trim the user ID.
    text = replace_once(text, '$user_abb = preg_replace("/^\\./i","",$user_abb);',
                        '$user_abb = substr($user_abb, -4);')
    return text.encode()


def calendar_overlay(data):
    """An agent may scroll before any tcal instance creates the registry."""
    if r4037_digest(data) != CALENDAR_PIN:
        raise ValueError('Pinned r4037 calendar source mismatch')
    return replace_once(data.decode(), 'function f_tcal_hideAll () {',
                        'function f_tcal_hideAll () {\r\n'
                        '\tif (!window.A_TCALSIDX) return; // No calendars instantiated yet.').encode()


def updater_overlay(data):
    """Upstream AST_update, isolated config and Asterisk 22/PJSIP parsing.

    This does not start a process or dispatch the vicidial_manager queue.
    """
    if r4037_digest(data) != UPDATER_PIN:
        raise ValueError('Pinned r4037 updater source mismatch')
    text = replace_once(data.decode(), "'/etc/astguiclient.conf'", "'/etc/astguiclient-m007-updater.conf'")
    marker = 'if ($AST_ver =~ /^12\\./i) {$show_channels_format = 4;}'
    text = replace_once(text, marker, marker + '\nif ($AST_ver =~ /^22\\./i) {$show_channels_format = 4;}')
    marker = '$show_channels_format = 1;'
    text = replace_once(text, marker, marker + "\n$alm7_command_prompt = ($AST_ver =~ /^22\\./i) ? '/\\r?\\n\\r?\\n/' : '/--END COMMAND-.*/';")
    old = "Prompt => '/--END COMMAND-.*/'"
    if text.count(old) != 3:
        raise ValueError('Unexpected updater AMI command marker')
    text = text.replace(old, 'Prompt => $alm7_command_prompt')
    marker = '\t\t##### TEST CHANNELS ZAP/IAX2/Local TO SEE IF THERE WAS A LARGE HICCUP IN OUTPUT FROM PREVIOUS OUTPUT'
    text = replace_once(text, marker, '''\t\t# Asterisk 22 AMI Command returns a framed Success with Output fields.
        # Adapt only a successful command response to the upstream parser shape.
        if (($AST_ver =~ /^22\\./i) && (grep { /^Message: Command output follows/ } @list_channels)) {
            @list_channels = map { s/^Output: ?//; $_ } grep { /^Output: .+\\S/ } @list_channels;
            unshift @list_channels, "Response: Follows\\n", "Privilege: Command\\n";
            push @list_channels, "--END COMMAND--\\n";
        }
''' + marker)
    old = "where protocol='SIP' and server_ip='$server_ip'"
    if text.count(old) != 2:
        raise ValueError('Unexpected updater phone lookup marker')
    text = text.replace(old, "where protocol IN ('SIP','PJSIP') and server_ip='$server_ip'")
    # Keep actual PJSIP channel names for subsequent AMI commands.
    for old, new, count in [
            ('/^Zap|^IAX2|^SIP|^Local|^DAHDI/', '/^Zap|^IAX2|^SIP|^PJSIP|^Local|^DAHDI/', 2),
            ('/^SIP|^Zap|^IAX2|^DAHDI/', '/^SIP|^PJSIP|^Zap|^IAX2|^DAHDI/', 1),
            ('/^SIP/', '/^(?:SIP|PJSIP)/', 2),
            ('s/^SIP\\///gi', 's/^(?:SIP|PJSIP)\\///gi', 1)]:
        if text.count(old) != count:
            raise ValueError('Unexpected updater channel parser marker')
        text = text.replace(old, new)
    # AST_update stores PJSIP clients in live_sip_channels. The legacy park
    # validator only checks live_channels and deletes a still-parked client
    # after six misses. Match the actual upstream park AGI data, not merely
    # any live client channel (which could already have resumed).
    marker = '$stmtB = "SELECT count(*) FROM $live_channels where server_ip=\'$server_ip\' and channel=\'$PQchannel\' and ( (extension=\'$PQextension\') or (extension LIKE \\"%.agi\\") );";'
    text = replace_once(text, marker, marker + '''
                if (($AST_ver =~ /^22\\./i) && ($PQchannel =~ /^PJSIP\\//)) {
                    $stmtB = "SELECT count(*) FROM $live_sip_channels where server_ip='$server_ip' and channel='$PQchannel' and (channel_data='park_CID.agi' or channel_data LIKE '%/park_CID.agi');";
                }''')
    return text.encode()
