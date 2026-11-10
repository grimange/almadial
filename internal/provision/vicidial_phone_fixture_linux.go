//go:build linux

package provision

import (
	"context"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

// This fixture executes the pinned upstream phone and dialplan generation
// branch in Perl with an in-memory statement object. It cannot connect to a
// database, write a configuration file, reload Asterisk, or create an endpoint.
// It qualifies generation logic, not endpoint registration or calling.
const vicidialPhoneFixtureHarness = `use JSON::PP;
package AlmaDialPhoneFixture;
sub prepare {
    my ($self,$sql)=@_;
    die "unexpected fixture query\n" unless $sql eq $main::fixture_query;
    return bless {}, 'AlmaDialPhoneStatement';
}
package AlmaDialPhoneStatement;
sub execute { 1 }
sub rows { 1 }
sub fetchrow_array {
    return ('almadial_fixture_9005',$main::fixture_dialplan,'9005','fixture-only',
      '--NONE--','','','--NONE--','','9005','AlmaDial isolated fixture',
      'default','30','fixture-only','N','ulaw,alaw','0','central','','N','','');
}
sub finish { 1 }
package main;
$dbhA=bless {}, 'AlmaDialPhoneFixture';
$server_ip='127.0.0.1'; $asterisk_version='22.12.0';
$DB=0; $DBX=0; $DBXXX=0;
`

type VICIdialPhoneFixtureResult struct {
	Source, Branch, Harness string
	Cases                   []VICIdialPhoneFixtureCase
}
type VICIdialPhoneFixtureCase struct {
	Name, Wizard, Dialplan, Voicemail string
}

func vicidialPhoneFixtureParts(s VICIdialSource) (string, string, string, error) {
	data := s.files["bin/ADMIN_keepalive_ALL.pl"]
	if rawSHA(data) != vicidialGeneratorSourceSHA {
		return "", "", "", errVICIdial
	}
	text := string(data)
	branch, e := generatorPart(text, "##### BEGIN Generate the PJSIP phone entries #####", "##### END Generate the PJSIP phone entries #####")
	if e != nil || strings.Count(branch, "$dbhA->prepare(") != 2 {
		return "", "", "", errVICIdial
	}
	_, rest, ok := strings.Cut(branch, "$stmtA = \"")
	if !ok {
		return "", "", "", errVICIdial
	}
	query, _, ok := strings.Cut(rest, "\";")
	if !ok || !strings.HasPrefix(query, "SELECT extension,dialplan_number,") {
		return "", "", "", errVICIdial
	}
	index := strings.Index(text, "sub parse_asterisk_version\n")
	if index < 0 || strings.Count(text, "sub parse_asterisk_version\n") != 1 {
		return "", "", "", errVICIdial
	}
	helper := text[index:]
	return branch, query, helper, nil
}

// QualifyVICIdialPhoneFixture is safe on a developer workstation: only a bounded
// Perl process receives verified code over stdin. No application credentials or
// live database handles enter the fixture.
func QualifyVICIdialPhoneFixture(ctx context.Context, source VICIdialSource) (VICIdialPhoneFixtureResult, error) {
	runner, e := execution.NewRunner(nil)
	if e != nil {
		return VICIdialPhoneFixtureResult{}, e
	}
	return qualifyVICIdialPhoneFixture(ctx, source, runner)
}

func qualifyVICIdialPhoneFixture(ctx context.Context, source VICIdialSource, runner vicidialInputRunner) (VICIdialPhoneFixtureResult, error) {
	branch, query, helper, e := vicidialPhoneFixtureParts(source)
	if e != nil || ctx == nil || runner == nil {
		return VICIdialPhoneFixtureResult{}, errVICIdial
	}
	result := VICIdialPhoneFixtureResult{Source: vicidialGeneratorSourceSHA, Branch: rawSHA([]byte(branch)), Harness: rawSHA([]byte(vicidialPhoneFixtureHarness))}
	for _, name := range []string{"normal", "reserved", "empty"} {
		setup := "$fixture_dialplan='9005'; $reserved_extensions=',8300,8310,';\n"
		if name == "reserved" {
			setup = "$fixture_dialplan='9005'; $reserved_extensions=',9005,';\n"
		} else if name == "empty" {
			setup = "$fixture_dialplan=''; $reserved_extensions=',8300,8310,';\n"
		}
		// The query uses only the fixed server IP. It is compared byte-for-byte
		// by the fake statement; a template or other unexpected query aborts.
		program := vicidialPhoneFixtureHarness + setup + "$fixture_query=\"" + query + "\";\n" + branch + "\nprint JSON::PP->new->canonical->encode({Wizard=>$Ppjsipw,Dialplan=>$Pext,Voicemail=>$vm});\n" + helper
		c := dbCommand("/usr/bin/perl", "-")
		r, e := runner.RunInput(ctx, c, []byte(program))
		var outputs struct{ Wizard, Dialplan, Voicemail string }
		if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated || decode(r.Stdout, &outputs) != nil || verifyVICIdialPhoneFixtureOutput(name, outputs.Wizard, outputs.Dialplan, outputs.Voicemail) != nil {
			return VICIdialPhoneFixtureResult{}, errVICIdial
		}
		result.Cases = append(result.Cases, VICIdialPhoneFixtureCase{name, rawSHA([]byte(outputs.Wizard)), rawSHA([]byte(outputs.Dialplan)), rawSHA([]byte(outputs.Voicemail))})
	}
	return result, nil
}

func verifyVICIdialPhoneFixtureOutput(name, wizard, dialplan, voicemail string) error {
	for _, marker := range []string{"[almadial_fixture_9005]", "type=wizard", "inbound_auth/username=almadial_fixture_9005", "inbound_auth/password=fixture-only", "endpoint/allow=ulaw", "endpoint/allow=alaw"} {
		if !strings.Contains(wizard, marker) {
			return errVICIdial
		}
	}
	if strings.Contains(wizard, "type=transport") || !strings.Contains(voicemail, "9005 => fixture-only,almadial_fixture_9005 Mailbox,") {
		return errVICIdial
	}
	switch name {
	case "normal":
		if !strings.Contains(dialplan, "exten => 9005,1,Dial(${PJSIP_DIAL_CONTACTS(almadial_fixture_9005)},30,)") || !strings.Contains(dialplan, "exten => 9005,3,Hangup()") || strings.Contains(dialplan, "SKIP!") {
			return errVICIdial
		}
	case "reserved", "empty":
		marker := "RESERVED MATCH PJSIP SKIP!"
		if name == "empty" {
			marker = "EMPTY EXTEN PJSIP SKIP!"
		}
		if !strings.Contains(dialplan, marker) || strings.Contains(dialplan, "exten =>") {
			return errVICIdial
		}
	default:
		return errVICIdial
	}
	return nil
}
