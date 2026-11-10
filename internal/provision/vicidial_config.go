package provision

import "strings"

// Config contents are private because they contain the existing S03 database
// credential and a dedicated local AMI credential. Public plans bind their
// identity, never serialize these bytes.
type vicidialOwnedFile struct {
	path, owner, group, label string
	mode                      uint32
	content                   []byte
}

func vicidialHex(s string, n int) bool {
	return len(s) == n && strings.Trim(s, "0123456789abcdef") == ""
}

func vicidialConfiguration(database, ami string) ([]vicidialOwnedFile, error) {
	if !vicidialHex(database, 48) || !vicidialHex(ami, 20) {
		return nil, errVICIdial
	}
	app := "PATHhome => /usr/share/astguiclient\nPATHlogs => /var/log/asterisk\nPATHagi => /var/lib/asterisk/agi-bin\nPATHweb => /var/www/html\nPATHsounds => /var/lib/asterisk/sounds\nPATHmonitor => /var/spool/asterisk/monitor\nPATHDONEmonitor => /var/spool/asterisk/monitorDONE\nVARserver_ip => 127.0.0.1\nVARDB_server => localhost\nVARDB_database => asterisk\nVARDB_user => almadial\nVARDB_pass => " + database + "\nVARDB_port => 3306\nExpectedDBSchema => 1748\n"
	files := []vicidialOwnedFile{
		{"/etc/astguiclient.conf", "root", "apache", "etc_t", 0640, []byte(app)},
		{"/etc/astguiclient-asterisk.conf", "root", "asterisk", "etc_t", 0640, []byte(app)},
		{"/etc/asterisk/almadial-s05-manager.conf", "root", "asterisk", "etc_t", 0640, []byte("; S05 local integration probe. S06 owns active runtime roles.\n[almadial-s05]\nsecret=" + ami + "\ndeny=0.0.0.0/0.0.0.0\npermit=127.0.0.1/255.255.255.255\nread=system\nwrite=system\n")},
		{"/var/www/html/agc/options.php", "root", "apache", "httpd_sys_content_t", 0640, []byte("<?php\n$use_slave_server = 0;\n?>\n")},
		{"/var/www/html/vicidial/options.php", "root", "apache", "httpd_sys_content_t", 0640, []byte("<?php\n$use_slave_server = 0;\n?>\n")},
		{"/var/www/html/vicidial/old_clear", "root", "apache", "httpd_sys_content_t", 0640, []byte("AlmaDial immutable webroot prerequisite\n")},
	}
	return files, nil
}

// These wrapper transitions are closed S04 -> S05 handoffs, not append-to-any-
// configuration operations. Publication must first prove the exact predecessor
// or an already accepted S05 receipt; all S04 fragment contents stay intact.
func vicidialAsteriskWrappers() map[string][2]string {
	out := map[string][2]string{}
	for name, includes := range map[string]string{
		"manager":    "#include \"almadial-s05-manager.conf\"\n",
		"extensions": "[globals]\n#include \"extensions-vicidial.conf\"\n",
		"pjsip":      "#include \"pjsip-vicidial.conf\"\n",
		"confbridge": "#include \"confbridge-vicidial.conf\"\n",
	} {
		before := "#include almadial-s04-" + name + ".conf\n"
		out["/etc/asterisk/"+name+".conf"] = [2]string{before, before + includes}
	}
	out["/etc/asterisk/pjsip_wizard.conf"] = [2]string{"", "#include \"pjsip_wizard-vicidial.conf\"\n"}
	return out
}

// Registration is an S05 application-owned update, never an S03 bootstrap.
// No phones, users, campaigns, table definitions or credentials are created.
// The caller must verify the exact accepted TESTast preimage under durable
// intent and separately prove the postimage. MyISAM updates are not atomic.
func vicidialRegistrationSQL(ami string) (string, error) {
	if !vicidialHex(ami, 20) {
		return "", errVICIdial
	}
	return "UPDATE asterisk.servers SET server_ip='127.0.0.1',active='Y',active_asterisk_server='Y',generate_vicidial_conf='Y',rebuild_conf_files='Y',asterisk_version='22.12.0',conf_engine='CONFBRIDGE',ASTmgrUSERNAME='almadial-s05',ASTmgrUSERNAMEupdate='almadial-s05',ASTmgrUSERNAMElisten='almadial-s05',ASTmgrUSERNAMEsend='almadial-s05',ASTmgrSECRET='" + ami + "' WHERE server_id='TESTast' AND server_ip='10.10.10.15';\nUPDATE asterisk.system_settings SET webroot_writable='0';\n", nil
}
