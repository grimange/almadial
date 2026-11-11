//go:build linux

package diagnostics

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/provision"
)

const fakeSecret = "FAKE_PASSWORD_SIP_AUTH_COOKIE_CALLER_DATA"

type fakeRunner struct {
	mutate func(execution.Command, *execution.Result)
	calls  []execution.Command
}

func (f *fakeRunner) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls = append(f.calls, c)
	s := ""
	args := strings.Join(c.Args, " ")
	switch filepath.Base(c.Executable) {
	case "systemctl":
		s = "active"
		if strings.Contains(args, "updater") || strings.Contains(args, "listener") {
			s = "inactive"
			if strings.Contains(args, "UnitFileState") {
				s = "disabled"
			}
		}
	case "uname":
		s = "6.12.0-fixture"
	case "timedatectl":
		s = "yes"
	case "getenforce":
		s = "Enforcing"
	case "rpm":
		s = strings.Repeat("fixture\n", 731)
	case "asterisk":
		switch args {
		case "-rx core show version":
			s = "Asterisk 22.12.0"
		case "-rx module show":
			for _, m := range provision.AsteriskModules() {
				s += m + ".so fixture 1 Running core\n"
			}
		case "-rx pjsip show endpoint alm_s10_9061":
			s = "Endpoint:  alm_s10_9061 fixture"
		case "-rx pjsip show contacts":
			s = "No objects found."
		}
	case "mariadb":
		s = "10.11.18-MariaDB\n1748"
	case "getent":
		s = "127.0.0.1 STREAM localhost"
	case "ss":
		for _, p := range []string{"80", "443", "8088", "8089", "3306", "5038", "4577"} {
			s += "LISTEN 0 100 127.0.0.1:" + p + " 0.0.0.0:*\n"
		}
	case "firewall-cmd":
		s = "almadial-s09\n  priority: -20000\n  target: DROP\n  ingress-zones: ANY\n  egress-zones: HOST\n  services:\n  ports: 22/tcp\n  protocols: icmp ipv6-icmp\n  masquerade: no\n  forward-ports:\n  source-ports:\n  icmp-blocks:\n  rich rules:"
		if !strings.Contains(args, "--permanent") {
			s = strings.Replace(s, "almadial-s09", "almadial-s09 (active)", 1)
		}
	case "ps":
		s = strings.Repeat("almadial-s06-fastagi.service asterisk /usr/bin/perl /usr/share/astguiclient/almadial-s06-FastAGI_log.pl\n", 6)
	case "curl":
		s = "401"
		if strings.Contains(args, ":8089") {
			s = "426"
		}
	case "stat":
		s = "2097152 4096"
	}
	r := execution.Result{Started: true, ExitCode: 0, Stdout: s}
	if f.mutate != nil {
		f.mutate(c, &r)
	}
	if r.ExitCode != 0 {
		return r, errors.New(fakeSecret)
	}
	return r, nil
}
func fixture(f *fakeRunner) Collector {
	return Collector{Runner: f, Now: func() time.Time { return time.Unix(0, 0) }, ReadFile: func(p string, _ int) ([]byte, error) {
		switch p {
		case "/usr/lib/os-release":
			return []byte("ID=\"almalinux\"\nVERSION_ID=\"10.2\"\n"), nil
		case "/etc/cron.d/almadial-s06":
			return []byte(provision.RuntimeCron()), nil
		}
		return nil, os.ErrNotExist
	}}
}
func check(t *testing.T, r Report, id string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("missing check", id)
	return Check{}
}
func TestHealthyDeterministicReadOnly(t *testing.T) {
	f := &fakeRunner{}
	c := fixture(f)
	r := c.Collect(context.Background(), Options{CAFile: "/approved/trust.pem"})
	if r.ExitCode() != 0 {
		t.Fatalf("%+v", r.Checks)
	}
	again := c.Collect(context.Background(), Options{CAFile: "/approved/trust.pem"})
	if !reflect.DeepEqual(r, again) {
		t.Fatal("nondeterministic report")
	}
	if check(t, r, "pjsip.registration").State != "OK" || check(t, r, "runtime.updater").State != "OK" {
		t.Fatal("idle/deferred treated as failure")
	}
	// Inspect every executable/argv, not just absence of one forbidden keyword.
	for _, cmd := range f.calls {
		if cmd.Timeout > 4*time.Second || cmd.StdoutLimit > 256<<10 {
			t.Fatal("unbounded command")
		}
		a := strings.Join(cmd.Args, " ")
		switch filepath.Base(cmd.Executable) {
		case "systemctl":
			if cmd.Args[0] != "show" {
				t.Fatal("service mutation")
			}
		case "asterisk":
			if len(cmd.Args) != 2 || cmd.Args[0] != "-rx" || !strings.Contains(cmd.Args[1], "show") {
				t.Fatal("asterisk mutation")
			}
		case "firewall-cmd":
			if !strings.Contains(a, "--info-policy=") {
				t.Fatal("firewall mutation")
			}
		case "mariadb":
			if !strings.HasSuffix(a, "SELECT VERSION(); SELECT db_schema_version FROM asterisk.system_settings;") {
				t.Fatal("SQL mutation")
			}
		case "curl":
			if strings.Contains(a, "--insecure") || strings.Contains(a, "POST") {
				t.Fatal("unsafe HTTP")
			}
		case "uname", "timedatectl", "getenforce", "journalctl", "rpm", "getent", "ss", "ps", "test", "openssl", "stat", "runuser":
		default:
			t.Fatal("unexpected executable", cmd.Executable)
		}
	}
}
func TestFaultsStableActionableAndSanitized(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		mutate   func(execution.Command, *execution.Result)
	}{
		{"stopped service", "service.asterisk", func(c execution.Command, r *execution.Result) {
			if strings.Join(c.Args, " ") == "show --property=ActiveState --value asterisk.service" {
				r.Stdout = "inactive"
			}
		}},
		{"missing module", "asterisk.modules", func(c execution.Command, r *execution.Result) {
			if strings.Contains(strings.Join(c.Args, " "), "module show") {
				r.Stdout = ""
			}
		}},
		{"missing endpoint", "pjsip.endpoint", func(c execution.Command, r *execution.Result) {
			if strings.Contains(strings.Join(c.Args, " "), "show endpoint") {
				r.Stdout = "No objects found."
			}
		}},
		{"missing command", "database.schema", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "mariadb") {
				r.Started = false
				r.ExitCode = -1
				r.Stderr = fakeSecret
			}
		}},
		{"timeout", "tls.https", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "curl") {
				r.ExitCode = 124
				r.Stderr = fakeSecret
			}
		}},
		{"expired certificate", "tls.expiration", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "openssl") {
				r.ExitCode = 1
			}
		}},
		{"wrong workers", "runtime.fastagi", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "/ps") {
				r.Stdout = ""
			}
		}},
		{"external bind", "listener.443", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "/ss") {
				r.Stdout = strings.ReplaceAll(r.Stdout, "127.0.0.1:443", "0.0.0.0:443")
			}
		}},
		{"unsafe policy", "firewall.policy", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "firewall-cmd") {
				r.Stdout = strings.ReplaceAll(r.Stdout, "services:", "services: https")
			}
		}},
		{"rich rule continuation", "firewall.policy", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "firewall-cmd") {
				r.Stdout += "\n    rule family=ipv4 accept"
			}
		}},
		{"low disk", "recording.space", func(c execution.Command, r *execution.Result) {
			if strings.HasSuffix(c.Executable, "stat") {
				r.Stdout = "1 4096"
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRunner{mutate: tc.mutate}
			r := fixture(f).Collect(context.Background(), Options{CAFile: "/ca"})
			x := check(t, r, tc.id)
			if x.State == "OK" || x.Guidance == "" || r.ExitCode() != 10 {
				t.Fatalf("%+v", x)
			}
			b, _ := json.Marshal(r)
			if bytes.Contains(b, []byte(fakeSecret)) {
				t.Fatal("secret leaked")
			}
		})
	}
}
func TestNoRawInputsInBundle(t *testing.T) {
	f := &fakeRunner{mutate: func(c execution.Command, r *execution.Result) { r.Stdout += "\n" + fakeSecret; r.Stderr = fakeSecret }}
	r := fixture(f).Collect(context.Background(), Options{CAFile: "/ca", Journal: "/fake", Manifest: "/fake"})
	b, e := Bundle(r)
	if e != nil {
		t.Fatal(e)
	}
	members := unpack(t, b)
	for n, b := range members {
		if bytes.Contains(b, []byte(fakeSecret)) {
			t.Fatal("raw secret in", n)
		}
	}
}
func unpack(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	gz, e := gzip.NewReader(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	m := map[string][]byte{}
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		if h.Mode != 0600 || h.Typeflag != tar.TypeReg {
			t.Fatal("unsafe member")
		}
		m[h.Name], e = io.ReadAll(tr)
		if e != nil {
			t.Fatal(e)
		}
	}
	return m
}
func TestBundleIntegrityLimitsAndDestinations(t *testing.T) {
	r := fixture(&fakeRunner{}).Collect(context.Background(), Options{CAFile: "/ca"})
	b, e := Bundle(r)
	if e != nil {
		t.Fatal(e)
	}
	b2, _ := Bundle(r)
	if !bytes.Equal(b, b2) {
		t.Fatal("archive nondeterminism")
	}
	members := unpack(t, b)
	if len(members) != 4 {
		t.Fatal("unexpected inventory")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(members["SHA256SUMS"])), "\n") {
		f := strings.Fields(line)
		s := sha256.Sum256(members[f[1]])
		if f[0] != hex.EncodeToString(s[:]) {
			t.Fatal("checksum mismatch")
		}
	}
	var manifest struct {
		Members []Member `json:"members"`
	}
	if json.Unmarshal(members["manifest.json"], &manifest) != nil || len(manifest.Members) != 2 {
		t.Fatal("manifest")
	}
	for _, m := range manifest.Members {
		h := sha256.Sum256(members[m.Name])
		if m.Bytes != len(members[m.Name]) || m.SHA256 != hex.EncodeToString(h[:]) {
			t.Fatal("member identity")
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.tar.gz")
	if WriteBundle(path, b) != nil {
		t.Fatal("write")
	}
	s, _ := os.Stat(path)
	if s.Mode().Perm() != 0600 {
		t.Fatal("permissions")
	}
	if WriteBundle(path, b) == nil {
		t.Fatal("overwrite")
	}
	os.Symlink(dir, filepath.Join(dir, "link"))
	os.Symlink(path, filepath.Join(dir, "target"))
	for _, p := range []string{filepath.Join(dir, "link/new"), filepath.Join(dir, "target"), "relative", dir + "/../bad", filepath.Join(dir, "missing/new")} {
		if WriteBundle(p, b) == nil {
			t.Fatal("unsafe destination", p)
		}
	}
	r.Checks[0].Observation = strings.Repeat("x", MaxBundleBytes)
	if _, e := Bundle(r); e == nil {
		t.Fatal("uncompressed limit")
	}
}
func TestReadBoundedRejectsSymlinkSpecialAndOversized(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "file")
	os.WriteFile(p, []byte(fakeSecret), 0600)
	os.Symlink(p, filepath.Join(d, "link"))
	for _, p := range []string{p, filepath.Join(d, "link"), d, "/dev/null"} {
		if _, e := readBounded(p, 2); e == nil {
			t.Fatal("unsafe read", p)
		}
	}
}
func TestAVCNoMatchIsNotCommandFailure(t *testing.T) {
	f := &fakeRunner{mutate: func(c execution.Command, r *execution.Result) {
		if strings.HasSuffix(c.Executable, "journalctl") {
			r.ExitCode = 1
		}
	}}
	r := fixture(f).Collect(context.Background(), Options{})
	if check(t, r, "security.avc").State != "OK" {
		t.Fatal("no-match exit status misclassified")
	}
}
