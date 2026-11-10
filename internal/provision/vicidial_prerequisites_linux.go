//go:build linux

package provision

import (
	"context"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

// These commands only load approved modules and report interpreter identities.
// No application script, CPAN installer, service or scheduled job is executed.
const vicidialPerlProbe = `use JSON::PP; my @names=qw(DBI DBD::mysql Net::Telnet Net::Server Time::HiRes Switch Crypt::Eksblowfish::Bcrypt); my %versions; for my $m (@names) { if (eval "require $m; 1") { no strict 'refs'; $versions{$m}="${$m.'::VERSION'}"; } else { $versions{$m}='absent'; } } print JSON::PP->new->canonical->encode({version=>sprintf('%vd',$^V),modules=>\%versions});`
const vicidialPHPProbe = `$v=[]; foreach (['mysqli','pdo_mysql','mbstring','curl','xml','gd','ldap','session'] as $m) {$v[$m]=extension_loaded($m);} echo json_encode(['version'=>PHP_VERSION,'extensions'=>$v]);`

type VICIdialPrerequisites struct {
	Perl, PHP       string
	NetTelnetAbsent bool
}

// ReadVICIdialPrerequisites checks only the qualified interpreter/module set
// and Enforcing mode. It does not establish application installation readiness.
func ReadVICIdialPrerequisites(ctx context.Context) (VICIdialPrerequisites, error) {
	runner, e := execution.NewRunner(nil)
	if e != nil {
		return VICIdialPrerequisites{}, e
	}
	return inspectVICIdialPrerequisites(ctx, runner)
}

var vicidialPerlVersions = map[string]string{
	"DBI": "1.643", "DBD::mysql": "5.007", "Net::Telnet": "3.05",
	"Net::Server": "2.014", "Time::HiRes": "1.9777", "Switch": "2.17",
	"Crypt::Eksblowfish::Bcrypt": "0.009",
}

func inspectVICIdialPrerequisites(ctx context.Context, runner execution.CommandRunner) (VICIdialPrerequisites, error) {
	var out VICIdialPrerequisites
	if ctx == nil || runner == nil {
		return out, errVICIdial
	}
	run := func(c execution.Command) (string, error) {
		r, e := runner.Run(ctx, c)
		if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
			return "", errVICIdial
		}
		return r.Stdout, nil
	}
	p, e := run(dbCommand("/usr/bin/perl", "-e", vicidialPerlProbe))
	if e != nil {
		return out, e
	}
	var perl struct {
		Version string            `json:"version"`
		Modules map[string]string `json:"modules"`
	}
	if decode(p, &perl) != nil || perl.Version != "5.40.2" || len(perl.Modules) != len(vicidialPerlVersions) {
		return out, errVICIdial
	}
	for module, version := range vicidialPerlVersions {
		if module == "Net::Telnet" && perl.Modules[module] == "absent" {
			out.NetTelnetAbsent = true
			continue
		}
		if perl.Modules[module] != version {
			return out, errVICIdial
		}
	}
	q, e := run(dbCommand("/usr/bin/php", "-r", vicidialPHPProbe))
	if e != nil {
		return out, e
	}
	var php struct {
		Version    string          `json:"version"`
		Extensions map[string]bool `json:"extensions"`
	}
	if decode(q, &php) != nil || php.Version != "8.3.36" || len(php.Extensions) != 8 {
		return out, errVICIdial
	}
	for _, name := range []string{"mysqli", "pdo_mysql", "mbstring", "curl", "xml", "gd", "ldap", "session"} {
		if !php.Extensions[name] {
			return out, errVICIdial
		}
	}
	s, e := run(dbCommand("/usr/sbin/getenforce"))
	if e != nil || strings.TrimSpace(s) != "Enforcing" {
		return out, errVICIdial
	}
	out.Perl, out.PHP = hash(perl), hash(php)
	return out, nil
}
