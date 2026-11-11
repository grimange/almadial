//go:build linux

// Package diagnostics projects fixed read-only probes into an allowlisted report.
// It has no provisioning, repair, journal-session or service-control entry point.
package diagnostics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/buildinfo"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/provision"
)

type Check struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Observation string `json:"observation"`
	Expected    string `json:"expected"`
	Guidance    string `json:"guidance"`
	Blocking    bool   `json:"blocking"`
}
type Report struct {
	Schema      int     `json:"schema_version"`
	Tool        string  `json:"tool"`
	CollectedAt string  `json:"collected_at"`
	ReadOnly    bool    `json:"read_only"`
	Checks      []Check `json:"checks"`
}

func (r Report) ExitCode() int {
	for _, c := range r.Checks {
		if c.Blocking && c.State != "OK" {
			return 10
		}
	}
	return 0
}

type Options struct{ Manifest, Journal, CAFile string }
type Runner interface {
	Run(context.Context, execution.Command) (execution.Result, error)
}
type Collector struct {
	Runner   Runner
	ReadFile func(string, int) ([]byte, error)
	Now      func() time.Time
}

func New() Collector { r, _ := execution.NewRunner(nil); return Collector{r, readBounded, time.Now} }

// readBounded rejects symlink path components, nonregular files and oversized input. No
// bytes from these files are ever copied to exported diagnostics.
func readBounded(path string, limit int) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("unsafe diagnostic input")
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	parent, e := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	for _, part := range parts[:len(parts)-1] {
		next, err := syscall.Openat(parent, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		syscall.Close(parent)
		if err != nil {
			return nil, err
		}
		parent = next
	}
	fd, e := syscall.Openat(parent, parts[len(parts)-1], syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	syscall.Close(parent)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), "diagnostic input")
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > int64(limit) {
		return nil, fmt.Errorf("unsafe diagnostic input")
	}
	b, e := io.ReadAll(io.LimitReader(f, int64(limit+1)))
	if len(b) > limit {
		return nil, fmt.Errorf("diagnostic input too large")
	}
	return b, e
}
func (c Collector) probe(ctx context.Context, exe string, args ...string) (string, bool) {
	r, e := c.Runner.Run(ctx, execution.Command{Executable: exe, Args: args, Timeout: 4 * time.Second, StdoutLimit: 256 << 10, StderrLimit: 4096})
	return strings.TrimSpace(r.Stdout), e == nil && r.Started && r.ExitCode == 0 && !r.StdoutTruncated && !r.StderrTruncated
}

var version = regexp.MustCompile(`\b[0-9]{1,3}\.[0-9]{1,3}(?:\.[0-9]{1,3})?\b`)

func (c Collector) Collect(ctx context.Context, o Options) Report {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	r := Report{Schema: 1, Tool: buildinfo.String(), CollectedAt: c.Now().UTC().Format(time.RFC3339), ReadOnly: true, Checks: []Check{}}
	add := func(id, state, obs, expected, guide string, blocking bool) {
		r.Checks = append(r.Checks, Check{id, state, obs, expected, guide, blocking})
	}
	test := func(id string, available, good bool, obs, expected, guide string) {
		state := "OK"
		if !available {
			state = "UNKNOWN"
			obs = "probe unavailable, denied, timed out or exceeded capture limit"
		} else if !good {
			state = "FAIL"
		}
		add(id, state, obs, expected, guide, true)
	}
	for _, unit := range []string{"mariadb", "asterisk", "httpd", "php-fpm", "firewalld", "crond", "almadial-s06-fastagi"} {
		s, ok := c.probe(ctx, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", unit+".service")
		state := "not active"
		if s == "active" {
			state = "active"
		}
		test("service."+unit, ok, s == "active", state, "active", "Inspect systemctl status and journalctl -u "+unit+"; do not restart through diagnostics.")
	}
	osb, e := c.ReadFile("/usr/lib/os-release", 8192)
	osok := e == nil
	test("host.os", osok, strings.Contains(string(osb), "ID=\"almalinux\"") && strings.Contains(string(osb), "VERSION_ID=\"10.2\""), "OS identity checked against accepted AlmaLinux 10.2", "AlmaLinux 10.2", "Inspect /etc/os-release; installation compatibility is bounded to the accepted stack.")
	s, ok := c.probe(ctx, "/usr/bin/uname", "-r")
	v := version.FindString(s)
	test("host.kernel", ok && v != "", v != "", "kernel version "+v, "readable kernel version", "Inspect uname -r locally.")
	s, ok = c.probe(ctx, "/usr/bin/timedatectl", "show", "--property=NTPSynchronized", "--value")
	test("host.time", ok, s == "yes", "synchronization checked", "NTP synchronized", "Inspect timedatectl and chronyc tracking; clock skew affects certificates.")
	s, ok = c.probe(ctx, "/usr/sbin/getenforce")
	test("security.selinux", ok, s == "Enforcing", "SELinux mode checked", "Enforcing", "Inspect getenforce and AVCs; never disable SELinux to repair a service.")
	avc, avcErr := c.Runner.Run(ctx, execution.Command{Executable: "/usr/bin/journalctl", Args: []string{"--dmesg", "--boot", "--no-pager", "--output=cat", "--grep=avc:.*denied", "--lines=50"}, Timeout: 4 * time.Second, StdoutLimit: 256 << 10, StderrLimit: 4096})
	s, ok = strings.TrimSpace(avc.Stdout), avc.Started && !avc.StdoutTruncated && !avc.StderrTruncated && ((avcErr == nil && avc.ExitCode == 0) || (avc.ExitCode == 1 && avc.Stdout == "" && avc.Stderr == ""))
	state := "OK"
	obs := "no matching kernel AVC denials in bounded boot query"
	if !ok {
		state = "UNKNOWN"
		obs = "kernel AVC query unavailable"
	} else if s != "" && s != "-- No entries --" {
		state = "WARN"
		obs = "kernel AVC denials present; at most 50 selected entries inspected"
	}
	add("security.avc", state, obs, "no relevant denials", "Inspect journalctl -k -b and ausearch locally; no raw log messages are exported. Audit-file-only denials are outside this kernel query.", false)
	s, ok = c.probe(ctx, "/usr/bin/rpm", "-qa", "--qf", "%{NAME}\n")
	n := len(strings.Fields(s))
	test("packages.inventory", ok, n == 731, fmt.Sprintf("%d packages installed", n), "731 accepted packages", "Compare accepted RPM/repository evidence before installing or removing anything.")
	s, ok = c.probe(ctx, "/usr/sbin/asterisk", "-rx", "core show version")
	v = version.FindString(s)
	test("asterisk.version", ok, v == "22.12.0", "Asterisk version "+v, "22.12.0", "Inspect asterisk -rx 'core show version'.")
	s, ok = c.probe(ctx, "/usr/sbin/asterisk", "-rx", "module show")
	missing := 0
	for _, m := range provision.AsteriskModules() {
		found := false
		for _, line := range strings.Split(s, "\n") {
			f := strings.Fields(line)
			if len(f) > 0 && f[0] == m+".so" && strings.Contains(line, "Running") {
				found = true
				break
			}
		}
		if !found {
			missing++
		}
	}
	test("asterisk.modules", ok, missing == 0, fmt.Sprintf("%d required modules missing or not running", missing), "all 40 required native modules running", "Inspect asterisk -rx 'module show'; preserve source/module inventory before escalation.")
	s, ok = c.probe(ctx, "/usr/sbin/asterisk", "-rx", "pjsip show endpoint alm_s10_9061")
	test("pjsip.endpoint", ok, strings.Contains(s, "Endpoint:  alm_s10_9061"), "synthetic endpoint configuration inspected", "alm_s10_9061 configured", "Inspect asterisk -rx 'pjsip show endpoint alm_s10_9061'; do not rotate credentials.")
	s, ok = c.probe(ctx, "/usr/sbin/asterisk", "-rx", "pjsip show contacts")
	contacts := strings.Count(s, "Contact:  alm_s10_9061/")
	state = "OK"
	if !ok {
		state = "UNKNOWN"
	}
	add("pjsip.registration", state, fmt.Sprintf("%d synthetic contacts; absence after browser logout is expected", contacts), "registration optional when no browser session is active", "For an intended active session inspect PJSIP contacts and browser trust/WSS. Registration alone does not prove audio.", false)
	s, ok = c.probe(ctx, "/usr/bin/mariadb", "--no-defaults", "--protocol=SOCKET", "--socket=/var/lib/mysql/mysql.sock", "--user=root", "--batch", "--skip-column-names", "-e", "SELECT VERSION(); SELECT db_schema_version FROM asterisk.system_settings;")
	lines := strings.Split(s, "\n")
	dbok := len(lines) == 2 && version.FindString(lines[0]) == "10.11.18" && lines[1] == "1748"
	test("database.schema", ok, dbok, "database version and schema queried without credentials in argv", "MariaDB 10.11.18; schema 1748", "Run as authorized local root for socket authentication; inspect mariadb service and accepted schema evidence. Never reimport a partial schema.")
	s, ok = c.probe(ctx, "/usr/bin/getent", "ahostsv4", "localhost")
	dnsok := len(strings.Fields(s)) > 0
	for _, l := range strings.Split(s, "\n") {
		f := strings.Fields(l)
		if len(f) > 0 && f[0] != "127.0.0.1" {
			dnsok = false
		}
	}
	test("network.dns", ok, dnsok, "localhost resolution checked", "localhost resolves to loopback", "Inspect getent ahostsv4 localhost and /etc/hosts. External DNS is outside the accepted localhost topology.")
	c.listeners(ctx, test)
	c.firewall(ctx, test)
	s, ok = c.probe(ctx, "/usr/bin/ps", "-eo", "unit=,user=,args=")
	count := 0
	foreign := false
	for _, l := range strings.Split(s, "\n") {
		f := strings.Fields(l)
		if len(f) >= 4 && f[2] == "/usr/bin/perl" && f[3] == "/usr/share/astguiclient/almadial-s06-FastAGI_log.pl" {
			count++
			if f[0] != "almadial-s06-fastagi.service" || f[1] != "asterisk" || len(f) != 4 {
				foreign = true
			}
		}
	}
	test("runtime.fastagi", ok, count == 6 && !foreign, fmt.Sprintf("%d FastAGI processes; identity and unit checked", count), "one parent plus five workers owned by asterisk in accepted unit", "Inspect systemctl status almadial-s06-fastagi and exact process ownership; do not start a competing keepalive.")
	for _, component := range provision.RuntimeInventory() {
		if component.Classification != "REQUIRED_CONFIGURED" {
			continue
		}
		a, aok := c.probe(ctx, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", component.Owner)
		b, bok := c.probe(ctx, "/usr/bin/systemctl", "show", "--property=UnitFileState", "--value", component.Owner)
		test("runtime."+component.ID, aok && bok, a == "inactive" && b == "disabled", "deferred unit activation checked", "disabled and inactive by accepted policy", "Inspect unit status; activation requires a separate reviewed runtime policy.")
	}
	b, e := c.ReadFile("/etc/cron.d/almadial-s06", 16384)
	test("runtime.cron", e == nil, string(b) == provision.RuntimeCron(), "owned cron definition compared to accepted inactive contract", "exact S06 definition; zero active application jobs", "Inspect /etc/cron.d/almadial-s06. Foreign schedules are not copied or changed; a complete cron audit remains an operator check.")
	_, readable := c.probe(ctx, "/usr/bin/test", "-r", "/var/www/html/vicidial/admin.php")
	_, nonempty := c.probe(ctx, "/usr/bin/test", "-s", "/var/www/html/vicidial/admin.php")
	test("vicidial.application", true, readable && nonempty, "administrator entry point readable and nonempty", "accepted application file available", "Inspect accepted S05 deployment hashes and Apache/PHP-FPM logs; diagnostics do not authenticate or exercise sessions.")
	c.tls(ctx, o, test, add)
	s, ok = c.probe(ctx, "/usr/bin/stat", "-f", "--format=%a %S", "/var/spool/asterisk/monitor")
	f := strings.Fields(s)
	var free uint64
	if len(f) == 2 {
		a, _ := strconv.ParseUint(f[0], 10, 64)
		b, _ := strconv.ParseUint(f[1], 10, 64)
		if b > 0 && a <= ^uint64(0)/b {
			free = a * b
		}
	}
	test("recording.space", ok && len(f) == 2, free >= 1<<30, fmt.Sprintf("%d MiB available", free>>20), "at least 1024 MiB recording space", "Inspect df and recording retention locally; no recordings are opened or copied.")
	_, ok = c.probe(ctx, "/usr/sbin/runuser", "--user", "asterisk", "--", "/usr/bin/test", "-r", "/var/spool/asterisk/monitor")
	_, wok := c.probe(ctx, "/usr/sbin/runuser", "--user", "asterisk", "--", "/usr/bin/test", "-w", "/var/spool/asterisk/monitor")
	_, xok := c.probe(ctx, "/usr/sbin/runuser", "--user", "asterisk", "--", "/usr/bin/test", "-x", "/var/spool/asterisk/monitor")
	test("recording.access", true, ok && wok && xok, "non-writing access checks as asterisk", "directory readable, writable and searchable by asterisk", "Inspect namei, modes, free space and SELinux labels; no test file is created. Run as authorized local root.")
	c.metadata(o, add)
	return r
}

type testCheck func(string, bool, bool, string, string, string)
type addCheck func(string, string, string, string, string, bool)

func (c Collector) listeners(ctx context.Context, test testCheck) {
	s, ok := c.probe(ctx, "/usr/sbin/ss", "-H", "-lnt")
	for _, port := range []string{"80", "443", "8088", "8089", "3306", "5038", "4577"} {
		found := false
		unsafe := false
		for _, l := range strings.Split(s, "\n") {
			f := strings.Fields(l)
			if len(f) < 4 {
				continue
			}
			a := f[3]
			if strings.HasSuffix(a, ":"+port) {
				found = true
				if a != "127.0.0.1:"+port {
					unsafe = true
				}
			}
		}
		test("listener."+port, ok, found && !unsafe, "bind scope inspected", "required TCP listener on 127.0.0.1 only", "Inspect ss -lntp locally; a firewall permission does not change service binding.")
	}
}
func (c Collector) firewall(ctx context.Context, test testCheck) {
	a, ok := c.probe(ctx, "/usr/bin/firewall-cmd", "--info-policy=almadial-s09")
	b, bok := c.probe(ctx, "/usr/bin/firewall-cmd", "--permanent", "--info-policy=almadial-s09")
	normalize := func(s string) string {
		return strings.TrimSpace(strings.ReplaceAll(s, "almadial-s09 (active)", "almadial-s09"))
	}
	good := normalize(a) == normalize(b) && strings.Contains(a, "target: DROP") && strings.Contains(a, "priority: -20000") && strings.Contains(a, "ingress-zones: ANY") && strings.Contains(a, "egress-zones: HOST")
	for _, l := range strings.Split(a, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "rule ") {
			good = false // firewalld prints rich rules on continuation lines
		}
		for _, key := range []string{"services:", "forward-ports:", "source-ports:", "icmp-blocks:", "rich rules:"} {
			if strings.HasPrefix(l, key) && l != key {
				good = false
			}
		}
		if strings.HasPrefix(l, "masquerade:") && l != "masquerade: no" {
			good = false
		}
		if strings.HasPrefix(l, "protocols:") && l != "protocols: icmp ipv6-icmp" {
			good = false
		}
		if strings.HasPrefix(l, "ports:") {
			fields := strings.Fields(strings.TrimPrefix(l, "ports:"))
			if len(fields) != 1 || !regexp.MustCompile(`^[0-9]{1,5}/tcp$`).MatchString(fields[0]) {
				good = false
			}
		}
	}
	test("firewall.policy", ok && bok, good, "managed runtime/permanent policy summary compared", "isolated DROP policy, matching permanent configuration, one management TCP port", "Inspect firewall-cmd --info-policy=almadial-s09 and nft list ruleset. This summary does not replace S09 full policy qualification.")
}
func (c Collector) tls(ctx context.Context, o Options, test testCheck, add addCheck) {
	if o.CAFile == "" {
		add("tls.trust", "NOT_CONFIGURED", "explicit trust anchor not supplied", "--ca-file for the intended public/private CA", "Supply the approved CA file; insecure TLS bypasses are never used.", true)
		return
	}
	for _, p := range []struct{ id, url, expected string }{{"https", "https://localhost/vicidial/admin.php", "401"}, {"wss", "https://localhost:8089/ws", "426"}} {
		s, ok := c.probe(ctx, "/usr/bin/curl", "--disable", "--noproxy", "*", "--silent", "--show-error", "--max-time", "3", "--cacert", o.CAFile, "--output", "/dev/null", "--write-out", "%{http_code}", p.url)
		test("tls."+p.id, ok, s == p.expected, "trusted TLS handshake and expected unauthenticated route response checked", "valid localhost certificate chain; HTTP "+p.expected, "Inspect approved CA, certificate expiration, Apache/Asterisk logs and HTTPS/WSS listeners. No authentication, SIP session or media test is performed.")
	}
	// OpenSSL prints only public certificate expiration; key material is never read.
	s, ok := c.probe(ctx, "/usr/bin/openssl", "x509", "-in", "/etc/pki/tls/almadial-s08/current/fullchain.pem", "-noout", "-checkend", "2592000")
	_ = s
	state := "OK"
	obs := "installed certificate valid for more than 30 days"
	if !ok {
		state = "WARN"
		obs = "certificate expires within 30 days or certificate inspection unavailable"
	}
	add("tls.expiration", state, obs, "more than 30 days before expiration", "Inspect openssl x509 -noout -dates against the accepted certificate; renewal is a separate authorized action.", true)
}
func (c Collector) metadata(o Options, add addCheck) {
	if o.Manifest == "" {
		add("installer.manifest", "NOT_CONFIGURED", "no explicit manifest selected", "--manifest to inspect source identity", "Supply the reviewed manifest; candidate status does not authorize production installation.", false)
	} else {
		b, e := c.ReadFile(o.Manifest, 64<<10)
		m, me := artifacts.Load(strings.NewReader(string(b)))
		state := "OK"
		obs := "validated manifest; status " + m.Status() + "; sha256 " + m.Digest()
		if e != nil || me != nil {
			state = "FAIL"
			obs = "manifest unavailable or invalid"
		}
		add("installer.manifest", state, obs, "valid reviewed source manifest; CANDIDATE remains CANDIDATE", "Inspect the selected manifest and accepted provenance; diagnostics never promote it.", true)
	}
	if o.Journal == "" {
		add("installer.journal", "NOT_CONFIGURED", "no explicit journal selected", "--journal for one relevant execution record", "Select the known stage journal; no latest journal is guessed and no lock/session is opened.", false)
	} else {
		b, e := c.ReadFile(o.Journal, journal.MaxBytes)
		d, de := journal.Decode(strings.NewReader(string(b)))
		state := "OK"
		pending := 0
		for _, a := range d.Actions {
			if a.State != journal.Succeeded {
				pending++
			}
		}
		if pending > 0 {
			state = "WARN"
		}
		obs := fmt.Sprintf("%d actions; %d not succeeded", len(d.Actions), pending)
		if e != nil || de != nil {
			state = "FAIL"
			obs = "journal unavailable or invalid"
		}
		if e == nil && de == nil {
			sum := sha256.Sum256(b)
			obs += "; sha256 " + hex.EncodeToString(sum[:])
		}
		add("installer.journal", state, obs, "valid journal; review incomplete actions against fresh host state", "Preserve journal and identity before using the accepted stage recovery workflow; this command does not reconcile or replay actions.", true)
	}
}
