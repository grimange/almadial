//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
)

type vicidialProbeRunner struct {
	perl, php, mode string
	fail, truncate  bool
	calls           int
}

func (r *vicidialProbeRunner) Validate(execution.Command) error { return nil }
func (r *vicidialProbeRunner) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	r.calls++
	out := ""
	switch c.Executable {
	case "/usr/bin/perl":
		if len(c.Args) != 2 || c.Args[0] != "-e" || c.Args[1] != vicidialPerlProbe {
			panic("unexpected command")
		}
		out = r.perl
	case "/usr/bin/php":
		if len(c.Args) != 2 || c.Args[0] != "-r" || c.Args[1] != vicidialPHPProbe {
			panic("unexpected command")
		}
		out = r.php
	case "/usr/sbin/getenforce":
		if len(c.Args) != 0 {
			panic("unexpected command")
		}
		out = r.mode
	default:
		panic("mutating or unexpected probe")
	}
	result := execution.Result{Started: true, Stdout: out, StdoutTruncated: r.truncate}
	if r.fail {
		result.ExitCode = 1
	}
	return result, nil
}

func vicidialProbes() *vicidialProbeRunner {
	m := map[string]string{}
	for name, v := range vicidialPerlVersions {
		m[name] = v
	}
	b, _ := json.Marshal(map[string]any{"version": "5.40.2", "modules": m})
	p := `{"version":"8.3.36","extensions":{"mysqli":true,"pdo_mysql":true,"mbstring":true,"curl":true,"xml":true,"gd":true,"ldap":true,"session":true}}`
	return &vicidialProbeRunner{perl: string(b), php: p, mode: "Enforcing\n"}
}

func TestVICIdialRuntimePrerequisites(t *testing.T) {
	for _, tc := range []struct {
		name          string
		change        func(*vicidialProbeRunner)
		valid, absent bool
	}{
		{"complete", func(*vicidialProbeRunner) {}, true, false},
		{"reviewed source dependency missing", func(r *vicidialProbeRunner) {
			r.perl = strings.Replace(r.perl, `"Net::Telnet":"3.05"`, `"Net::Telnet":"absent"`, 1)
		}, true, true},
		{"DBI missing", func(r *vicidialProbeRunner) { r.perl = strings.Replace(r.perl, `"DBI":"1.643"`, `"DBI":"absent"`, 1) }, false, false},
		{"wrong Perl", func(r *vicidialProbeRunner) { r.perl = strings.Replace(r.perl, "5.40.2", "5.38.0", 1) }, false, false},
		{"wrong approved dependency", func(r *vicidialProbeRunner) { r.perl = strings.Replace(r.perl, "3.05", "3.06", 1) }, false, false},
		{"missing PHP extension", func(r *vicidialProbeRunner) { r.php = strings.Replace(r.php, `"mysqli":true`, `"mysqli":false`, 1) }, false, false},
		{"PHP module substituted", func(r *vicidialProbeRunner) { r.php = strings.Replace(r.php, `"ldap"`, `"sqlite"`, 1) }, false, false},
		{"PHP version mismatch", func(r *vicidialProbeRunner) { r.php = strings.Replace(r.php, "8.3.36", "8.4.0", 1) }, false, false},
		{"SELinux permissive", func(r *vicidialProbeRunner) { r.mode = "Permissive" }, false, false},
		{"exit failure despite valid stdout", func(r *vicidialProbeRunner) { r.fail = true }, false, false},
		{"truncated observation", func(r *vicidialProbeRunner) { r.truncate = true }, false, false},
		{"unrecognized JSON", func(r *vicidialProbeRunner) {
			r.php = strings.Replace(r.php, `"version":`, `"unexpected":true,"version":`, 1)
		}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := vicidialProbes()
			tc.change(r)
			p, e := inspectVICIdialPrerequisites(context.Background(), r)
			if (e == nil) != tc.valid {
				t.Fatal(e)
			}
			if e == nil && (!shaPattern.MatchString(p.Perl) || !shaPattern.MatchString(p.PHP) || p.NetTelnetAbsent != tc.absent) {
				t.Fatal(p)
			}
		})
	}
}

func TestVICIdialLocalWebContract(t *testing.T) {
	f := vicidialWebConfiguration()
	if len(f) != 2 || f[0].path != "/etc/httpd/conf/almadial-s05.conf" || f[1].path != "/etc/systemd/system/httpd.service.d/almadial-s05.conf" {
		t.Fatal("unexpected service ownership")
	}
	if strings.Count(vicidialHTTPD, "Listen ") != 1 || !strings.Contains(vicidialHTTPD, "Listen 127.0.0.1:80\n") || !strings.Contains(vicidialHTTPD, "proxy:unix:/run/php-fpm/www.sock|fcgi://localhost") || !strings.Contains(vicidialHTTPD, "Require local") {
		t.Fatal("exposure or PHP handling missing")
	}
	for _, v := range []string{"Listen 80\n", "IncludeOptional conf.d", "Require all granted", "Options Indexes", "AllowOverride All"} {
		if strings.Contains(vicidialHTTPD, v) {
			t.Fatal(v)
		}
	}
	if vicidialHTTPDDropIn != "[Service]\nEnvironment=\"OPTIONS=-f /etc/httpd/conf/almadial-s05.conf\"\n" {
		t.Fatal("vendor service replacement")
	}
}
