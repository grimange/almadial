package provision

import (
	"strings"
	"testing"
)

func TestVICIdialPhoneFixtureOutputs(t *testing.T) {
	wizard := "[almadial_fixture_9005]\ntype=wizard\ninbound_auth/username=almadial_fixture_9005\ninbound_auth/password=fixture-only\nendpoint/allow=ulaw\nendpoint/allow=alaw\n"
	normal := "exten => 9005,1,Dial(${PJSIP_DIAL_CONTACTS(almadial_fixture_9005)},30,)\nexten => 9005,3,Hangup()\n"
	vm := "9005 => fixture-only,almadial_fixture_9005 Mailbox,"
	for _, tc := range []struct {
		name, w, d, v string
		ok            bool
	}{
		{"normal", wizard, normal, vm, true},
		{"reserved", wizard, "; RESERVED MATCH PJSIP SKIP!", vm, true},
		{"empty", wizard, "; EMPTY EXTEN PJSIP SKIP!", vm, true},
		{"normal", wizard, "", vm, false},
		{"normal", strings.ReplaceAll(wizard, "type=wizard", ""), normal, vm, false},
		{"normal", wizard + "type=transport", normal, vm, false},
		{"normal", wizard, strings.ReplaceAll(normal, "PJSIP_DIAL_CONTACTS", "SIP"), vm, false},
		{"normal", wizard, normal, "", false},
		{"reserved", wizard, "; RESERVED MATCH PJSIP SKIP!\n" + normal, vm, false},
		{"empty", wizard, "; RESERVED MATCH PJSIP SKIP!", vm, false},
		{"unknown", wizard, normal, vm, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (verifyVICIdialPhoneFixtureOutput(tc.name, tc.w, tc.d, tc.v) == nil) != tc.ok {
				t.Fatal("incorrect generation classification")
			}
		})
	}
}
func TestVICIdialPhoneFixtureRejectsUnverifiedSource(t *testing.T) {
	if _, _, _, e := vicidialPhoneFixtureParts(vicidialFixture()); e == nil {
		t.Fatal("unverified source admitted")
	}
	for _, forbidden := range []string{"use DBI", "DBI->connect", "system(", "open(", "`", "/etc/", "database_password"} {
		if strings.Contains(vicidialPhoneFixtureHarness, forbidden) {
			t.Fatal("effectful fixture harness", forbidden)
		}
	}
}
