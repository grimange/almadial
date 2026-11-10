package provision

import (
	"strings"
	"testing"
)

func TestVICIdialOwnedConfiguration(t *testing.T) {
	db, ami := strings.Repeat("a", 48), strings.Repeat("b", 20)
	files, e := vicidialConfiguration(db, ami)
	if e != nil || len(files) != 6 {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f.path] || f.owner != "root" || f.mode != 0640 || len(f.content) == 0 {
			t.Fatal(f.path)
		}
		seen[f.path] = true
		s := string(f.content)
		switch {
		case strings.HasSuffix(f.path, "/options.php"):
			if s != "<?php\n$use_slave_server = 0;\n?>\n" || f.group != "apache" {
				t.Fatal(f.path)
			}
		case f.path == "/etc/asterisk/almadial-s05-manager.conf":
			if strings.Contains(s, db) || !strings.Contains(s, "secret="+ami+"\n") || !strings.Contains(s, "permit=127.0.0.1/255.255.255.255\nread=system\nwrite=system\n") || strings.Contains(s, "command") || strings.Contains(s, "originate") {
				t.Fatal("unexpected management privilege")
			}
		case strings.HasSuffix(f.path, "astguiclient.conf") || strings.HasSuffix(f.path, "astguiclient-asterisk.conf"):
			if !strings.Contains(s, "VARDB_pass => "+db+"\n") || !strings.Contains(s, "VARDB_server => localhost\n") || !strings.Contains(s, "ExpectedDBSchema => 1748\n") || strings.Contains(s, ami) {
				t.Fatal("application connection mismatch")
			}
		}
	}
	for _, pair := range [][2]string{{"", ami}, {db, ""}, {strings.Repeat("a", 47), ami}, {db, ami + "\n"}, {db, "'; DROP TABLE x; --"}, {strings.Repeat("z", 48), ami}} {
		if _, e := vicidialConfiguration(pair[0], pair[1]); e == nil {
			t.Fatal("invalid secret accepted")
		}
	}
}

func TestVICIdialAsteriskHandoff(t *testing.T) {
	m := vicidialAsteriskWrappers()
	if len(m) != 5 {
		t.Fatal(m)
	}
	for p, transition := range m {
		name := strings.TrimSuffix(strings.TrimPrefix(p, "/etc/asterisk/"), ".conf")
		if name != "pjsip_wizard" {
			if transition[0] != "#include almadial-s04-"+name+".conf\n" || !strings.HasPrefix(transition[1], transition[0]) {
				t.Fatal("S04 ownership lost", p)
			}
		} else if transition[0] != "" {
			t.Fatal("foreign wizard preimage allowed")
		}
		for _, forbidden := range []string{"transport=", "type=transport", "bind=", "0.0.0.0", "chan_sip", "#include sip.conf"} {
			if strings.Contains(transition[1], forbidden) {
				t.Fatal("unexpected SIP configuration", p)
			}
		}
	}
}

func TestVICIdialRegistrationIsNotBootstrap(t *testing.T) {
	q, e := vicidialRegistrationSQL(strings.Repeat("a", 20))
	if e != nil || !strings.Contains(q, "WHERE server_id='TESTast' AND server_ip='10.10.10.15'") || !strings.Contains(q, "conf_engine='CONFBRIDGE'") || !strings.Contains(q, "webroot_writable='0'") {
		t.Fatal(e, q)
	}
	for _, forbidden := range []string{"CREATE ", "ALTER ", "DROP ", "DELETE ", "INSERT ", "GRANT ", "phones", "vicidial_users", "db_schema_version"} {
		if strings.Contains(q, forbidden) {
			t.Fatal("S03 ownership crossed", forbidden)
		}
	}
	if _, e := vicidialRegistrationSQL("bad credential"); e == nil {
		t.Fatal("unvalidated SQL value")
	}
}
