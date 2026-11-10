package provision

import "strings"

const vicidialWork = "/var/lib/almadial/vicidial-s05"
const vicidialGenerated = vicidialWork + "/generated"
const vicidialGeneratorSourceSHA = "4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67"
const vicidialGeneratorBlockSHA = "41c7019e76f76e2aba8cac030b47fc17bce9b8544afcaa9980273491e313569f"
const vicidialGeneratorDerivedSHA = "a830cc4f92febd2ddd044fb04552e096a434b261f0209db7ba33842b3a11e14f"

// VICIdialGenerator records a derivation, not execution or publication success.
// Only the four modern outputs may be included by the native runtime. The
// upstream legacy outputs remain private and cannot authorize a SIP listener.
type VICIdialGenerator struct {
	Source, Block, Derived              string
	PathReplacements, SuppressedReloads int
	Outputs, RuntimeIncludes            []string
}

func vicidialGeneratorOutputs() []string {
	return []string{"extensions-vicidial.conf", "iax-vicidial.conf", "sip-vicidial.conf", "pjsip-vicidial.conf", "pjsip_wizard-vicidial.conf", "voicemail.conf", "musiconhold-vicidial.conf", "meetme-vicidial.conf", "confbridge-vicidial.conf"}
}

// Generator derives only the M0-reviewed generation block and helpers, never
// the daemon/cron paths in ADMIN_keepalive_ALL.pl. No database is contacted.
// A future journaled caller must separately guard registration, generation,
// preimages, publication and actual native reload postconditions.
func (s VICIdialSource) Generator() ([]byte, VICIdialGenerator, error) {
	return deriveVICIdialGenerator(s.files["bin/ADMIN_keepalive_ALL.pl"])
}

func generatorPart(text, first, last string) (string, error) {
	if strings.Count(text, first) != 1 || strings.Count(text, last) != 1 {
		return "", errVICIdial
	}
	a, b := strings.Index(text, first), strings.Index(text, last)
	if b <= a {
		return "", errVICIdial
	}
	return text[a:b], nil
}

func deriveVICIdialGenerator(data []byte) ([]byte, VICIdialGenerator, error) {
	d := VICIdialGenerator{}
	if rawSHA(data) != vicidialGeneratorSourceSHA {
		return nil, d, errVICIdial
	}
	text := string(data)
	block, e := generatorPart(text, "if ( ($active_asterisk_server =~ /Y/) && ($generate_vicidial_conf =~ /Y/) && ($rebuild_conf_files =~ /Y/) )", "#####  END Creation of")
	if e != nil || rawSHA([]byte(block)) != vicidialGeneratorBlockSHA {
		return nil, d, errVICIdial
	}
	settings, e := generatorPart(text, "##### Get the settings from system_settings #####", "if ($VARactive_keepalives =~ /X/)")
	if e != nil || strings.Count(text, "sub leading_zero($)") != 1 {
		return nil, d, errVICIdial
	}
	helpers := text[strings.Index(text, "sub leading_zero($)"):]
	lines := strings.Split(text, "\n")
	if len(lines) < 213 {
		return nil, d, errVICIdial
	}
	reserved := strings.Join(lines[209:213], "\n")
	// Apply precisely the M0 GSM priority correction before generation. Never
	// edit a generated dialplan to mask an unreviewed algorithm change.
	marker := `$Vext .= "exten => 8310,1,Answer\n";`
	if strings.Count(block, marker) != 1 {
		return nil, d, errVICIdial
	}
	a, b, _ := strings.Cut(block, marker)
	const old = "exten => 8309,2,MixMonitor("
	if strings.Count(b, old) != 1 {
		return nil, d, errVICIdial
	}
	block = a + marker + strings.Replace(b, old, "exten => 8310,2,MixMonitor(", 1)
	paths := strings.Count(block, "/etc/asterisk/")
	reloads := strings.Count(block, "`screen -XS asterisk")
	if paths != 94 || reloads != 15 {
		return nil, d, errVICIdial
	}
	block = strings.ReplaceAll(block, "/etc/asterisk/", vicidialGenerated+"/")
	block = strings.ReplaceAll(block, "`screen -XS asterisk", "`almadial-s05-no-reload -XS asterisk")
	// All values are internal constants except the existing protected S03
	// credential read by this process. No secret appears in derived source,
	// command arguments, derivation metadata or the application manifest.
	const prefix = `#!/usr/bin/perl
# AlmaDial S05: M0-approved isolated r4037 generator, never a keepalive daemon.
use DBI;
use File::Compare;
umask 0077;
$ENV{PATH}='/var/lib/almadial/vicidial-s05/tools:/usr/bin:/bin';
open(my $credential, '<', '/etc/almadial/secrets/database') or die "protected S03 credential unavailable\n";
my $database_password = <$credential>;
close($credential);
chomp($database_password);
die "invalid S03 credential\n" unless $database_password =~ /\A[0-9a-f]{48}\z/;
$server_ip='127.0.0.1'; $VARserver_ip=$server_ip;
$PATHmonitor='/var/spool/asterisk/monitor';
$dbhA=DBI->connect('DBI:mysql:database=asterisk;mysql_socket=/var/lib/mysql/mysql.sock','almadial',$database_password,{RaiseError=>1,PrintError=>0});
$database_password='';
$DB=0;$DBX=0;$DBXXX=0;
`
	result := []byte(prefix + reserved + "\n" + settings + "\n" + block + "\n" + helpers)
	if rawSHA(result) != vicidialGeneratorDerivedSHA {
		return nil, VICIdialGenerator{}, errVICIdial
	}
	d = VICIdialGenerator{vicidialGeneratorSourceSHA, vicidialGeneratorBlockSHA, rawSHA(result), paths, reloads, vicidialGeneratorOutputs(), []string{"extensions-vicidial.conf", "pjsip-vicidial.conf", "pjsip_wizard-vicidial.conf", "confbridge-vicidial.conf"}}
	return result, d, nil
}
