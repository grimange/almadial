//go:build linux

package provision

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed asterisk_files.py
var asteriskAdapter string

type AsteriskNative struct {
	Database DatabaseNative
	Source   AsteriskSource
}

func asteriskCommand(exe string, args ...string) execution.Command {
	return execution.Command{Executable: exe, Args: args, Timeout: 5 * time.Minute, StdoutLimit: 4 << 20, StderrLimit: 1 << 20}
}
func asteriskActionCommand(id string) execution.Command {
	c := asteriskCommand("/proc/self/exe", "__asterisk", id)
	c.Timeout = 2 * time.Hour
	return c
}
func (n AsteriskNative) run(ctx context.Context, c execution.Command) (execution.Result, error) {
	r, e := n.Database.Native.Runner.Run(ctx, c)
	if e != nil {
		return r, e
	}
	if !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return r, errAsterisk
	}
	return r, nil
}
func (n AsteriskNative) toolchain(ctx context.Context) (string, error) {
	all := []string{}
	for _, exe := range []string{"/usr/bin/gcc", "/usr/bin/g++", "/usr/bin/make"} {
		r, e := n.run(ctx, asteriskCommand(exe, "--version"))
		if e != nil {
			return "", e
		}
		all = append(all, r.Stdout)
	}
	return hash(all), nil
}
func (n AsteriskNative) binding(toolchain string) string {
	return hash(struct{ Source, Policy, Toolchain, Architecture, Prefix string }{n.Source.Digest(), AsteriskBuildPolicyDigest(), toolchain, "x86_64", "/usr"})
}
func (n AsteriskNative) fileCommand(mode, tc, service string) execution.Command {
	p := map[string]any{"modules": AsteriskModules(), "files": asteriskFiles(), "unit": asteriskUnit, "binding": n.binding(tc), "archive": asteriskArchiveSHA, "pj": pjArchiveSHA, "service": service}
	return asteriskCommand("/usr/bin/python3", "-B", "-c", asteriskAdapter, mode, jsonString(p))
}
func (n AsteriskNative) service(ctx context.Context) (map[string]string, error) {
	r, e := n.run(ctx, asteriskCommand("/usr/bin/systemctl", "show", "asterisk.service", "--property=LoadState,ActiveState,UnitFileState,MainPID,InvocationID,User,Group,FragmentPath,DropInPaths"))
	if e != nil {
		return nil, e
	}
	m := map[string]string{}
	for _, l := range strings.Split(r.Stdout, "\n") {
		a, b, ok := strings.Cut(l, "=")
		if ok {
			m[a] = b
		}
	}
	return m, nil
}
func (n AsteriskNative) cli(ctx context.Context, cmd string) (string, error) {
	r, e := n.run(ctx, asteriskCommand("/usr/sbin/asterisk", "-rx", cmd))
	if e != nil {
		return "", e
	}
	if strings.Contains(r.Stdout, "No such command") || strings.Contains(r.Stdout, "not registered") {
		return "", errAsterisk
	}
	return r.Stdout, nil
}
func (n AsteriskNative) InspectAsterisk(ctx context.Context) (AsteriskSnapshot, error) {
	s := AsteriskSnapshot{Source: n.Source.Digest()}
	if s.Source == "" {
		return s, errAsterisk
	}
	db, e := n.Database.InspectDatabase(ctx)
	if e != nil || !databaseSatisfied(db) {
		return s, errAsterisk
	}
	s.Host, s.Foundation, s.Database = db.Host, db.Foundation, db.Digest()
	tc, e := n.toolchain(ctx)
	if e != nil {
		return s, e
	}
	s.Toolchain = tc
	svc, e := n.service(ctx)
	if e != nil {
		return s, e
	}
	s.Active = svc["ActiveState"] == "active"
	s.Enabled = svc["UnitFileState"] == "enabled"
	if s.Active {
		s.ServiceID = svc["InvocationID"]
	}
	r, e := n.run(ctx, n.fileCommand("inspect", tc, s.ServiceID))
	if e != nil {
		return s, e
	}
	var f struct {
		Build, Install, Configuration, Unit string
		RestartVerified                     bool
		Modules                             []AsteriskModule
	}
	if decode(r.Stdout, &f) != nil {
		return s, errAsterisk
	}
	s.Build, s.Install, s.Configuration, s.Unit, s.RestartVerified, s.Modules = f.Build, f.Install, f.Configuration, f.Unit, f.RestartVerified, f.Modules
	s.EnvironmentOK = true
	// A proven incomplete runtime cannot be healthy. Ignore systemd restart-loop
	// invocation churn only for these exact receipt-backed repair states.
	if s.Install == "resources-missing" || s.Install == "libraries-missing" {
		s.Active, s.RuntimeOK, s.RestartVerified, s.ServiceID = false, false, false, ""
		return s, nil
	}
	if !s.Active {
		return s, nil
	}
	if s.Install == "absent" || s.Configuration == "absent" || s.Unit == "absent" {
		return s, errAsterisk
	}
	s.RuntimeOK = n.runtime(ctx, svc, &s) == nil
	return s, nil
}
func (n AsteriskNative) runtime(ctx context.Context, svc map[string]string, s *AsteriskSnapshot) error {
	if svc["User"] != "asterisk" || svc["Group"] != "asterisk" || svc["FragmentPath"] != "/etc/systemd/system/asterisk.service" || svc["DropInPaths"] != "" || !shaInvocation(svc["InvocationID"]) {
		return errAsterisk
	}
	pid, e := strconv.Atoi(svc["MainPID"])
	if e != nil || pid < 2 {
		return errAsterisk
	}
	u, e := user.Lookup("asterisk")
	if e != nil {
		return e
	}
	g, e := user.LookupGroup("asterisk")
	if e != nil {
		return e
	}
	status, e := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if e != nil {
		return e
	}
	for _, pair := range [][2]string{{"Uid:", u.Uid}, {"Gid:", g.Gid}} {
		found := false
		for _, l := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(l, pair[0]) {
				v := strings.Fields(l)
				if len(v) != 5 {
					return errAsterisk
				}
				for _, x := range v[1:] {
					if x != pair[1] {
						return errAsterisk
					}
				}
				found = true
			}
		}
		if !found {
			return errAsterisk
		}
	}
	exe, e := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if e != nil || exe != "/usr/sbin/asterisk" {
		return errAsterisk
	}
	// Independent duplicate-process check includes non-systemd instances.
	entries, e := os.ReadDir("/proc")
	if e != nil {
		return e
	}
	count := 0
	for _, d := range entries {
		if _, e := strconv.Atoi(d.Name()); e != nil {
			continue
		}
		x, e := os.Readlink("/proc/" + d.Name() + "/exe")
		if e == nil && x == "/usr/sbin/asterisk" {
			count++
		}
	}
	if count != 1 {
		return errAsterisk
	}
	enforce, e := n.run(ctx, asteriskCommand("/usr/sbin/getenforce"))
	if e != nil || strings.TrimSpace(enforce.Stdout) != "Enforcing" {
		return errAsterisk
	}
	domain, e := os.ReadFile(fmt.Sprintf("/proc/%d/attr/current", pid))
	if e != nil || !strings.Contains(string(domain), ":system_r:") {
		return errAsterisk
	}
	paths := []string{"/var/lib/asterisk/documentation", "/var/lib/asterisk/documentation/core-en_US.xml", "/var/lib/asterisk/documentation/appdocsxml.xslt", "/var/lib/asterisk/documentation/appdocsxml.dtd", "/usr/sbin/asterisk", "/usr/lib/libasteriskssl.so.1", "/usr/lib/libasteriskpj.so.2", "/etc/systemd/system/asterisk.service"}
	for _, m := range AsteriskModules() {
		paths = append(paths, "/usr/lib/asterisk/modules/"+m+".so")
	}
	for name := range asteriskFiles() {
		paths = append(paths, "/etc/asterisk/"+name+".conf", "/etc/asterisk/almadial-s04-"+name+".conf")
	}
	for _, p := range paths {
		if _, e := n.run(ctx, asteriskCommand("/usr/sbin/matchpathcon", "-V", p)); e != nil {
			return errAsterisk
		}
	}
	version, e := n.cli(ctx, "core show version")
	if e != nil || !strings.HasPrefix(version, "Asterisk "+AsteriskVersion+" ") {
		return errAsterisk
	}
	for j, m := range s.Modules {
		v, e := n.cli(ctx, "module show like "+m.Name+".so")
		if e != nil {
			return e
		}
		for _, line := range strings.Split(v, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 1 && fields[0] == m.Name+".so" && strings.Contains(line, "Running") {
				s.Modules[j].Loaded = true
			}
		}
		if !s.Modules[j].Loaded {
			return errAsterisk
		}
	}
	capabilities := map[string]string{"chan_pjsip": "pjsip show transports", "app_confbridge": "core show application ConfBridge", "res_agi": "core show application AGI", "app_mixmonitor": "core show application MixMonitor"}
	for j, m := range s.Modules {
		if cmd, ok := capabilities[m.Name]; ok {
			v, e := n.cli(ctx, cmd)
			if e != nil || strings.Contains(v, "not registered") || strings.Contains(v, "not available") || strings.TrimSpace(v) == "" {
				return errAsterisk
			}
			if m.Name == "chan_pjsip" && !strings.Contains(v, "No objects found.") {
				return errAsterisk
			}
			s.Modules[j].Capability = cmd
		}
		s.Modules[j].Pass = m.Source && m.Built && m.Installed && s.Modules[j].Loaded
	}
	ami, e := n.cli(ctx, "manager show settings")
	if e != nil || !strings.Contains(ami, "127.0.0.1:5038") {
		return errAsterisk
	}
	users, e := n.cli(ctx, "manager show users")
	if e != nil || strings.TrimSpace(users) != "There are no manager users." {
		return errAsterisk
	}
	http, e := n.cli(ctx, "http show status")
	if e != nil || !strings.Contains(http, "127.0.0.1:8088") {
		return errAsterisk
	}
	sock, e := n.run(ctx, asteriskCommand("/usr/sbin/ss", "-H", "-lntup"))
	if e != nil || verifyAsteriskListeners(sock.Stdout, pid) != nil {
		return errAsterisk
	}
	return nil
}
func shaInvocation(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
func verifyAsteriskListeners(raw string, pid int) error {
	seen := map[string]bool{}
	resolver := map[string]bool{}
	for _, l := range strings.Split(raw, "\n") {
		f := strings.Fields(l)
		if len(f) < 6 {
			continue
		}
		addr := f[4]
		owned := strings.Contains(l, fmt.Sprintf("pid=%d,", pid))
		management := strings.HasSuffix(addr, ":5038") || strings.HasSuffix(addr, ":8088")
		if !owned && !management {
			continue
		}
		// PJPROJECT 2.17 resolver.c opens one unconnected ephemeral UDP socket
		// per IP family. They are independent of SIP transports (verified empty
		// by CLI and closed configuration). No other Asterisk UDP bind is allowed.
		if owned && f[0] == "udp" && f[1] == "UNCONN" {
			host, port, ok := strings.Cut(addr, ":")
			if strings.HasPrefix(addr, "[::]:") {
				host = "[::]"
				port = strings.TrimPrefix(addr, "[::]:")
				ok = true
			}
			number, e := strconv.Atoi(port)
			if !ok || e != nil || number < 32768 || number > 65535 || (host != "0.0.0.0" && host != "[::]") || resolver[host] {
				return errAsterisk
			}
			resolver[host] = true
			continue
		}
		if !owned || f[0] != "tcp" || (addr != "127.0.0.1:5038" && addr != "127.0.0.1:8088") {
			return errAsterisk
		}
		seen[addr] = true
	}
	if !seen["127.0.0.1:5038"] || !seen["127.0.0.1:8088"] {
		return errAsterisk
	}
	return nil
}
func (n AsteriskNative) Runner(s AsteriskStage) execution.CommandRunner {
	a := map[string]string{}
	for _, v := range s.Graph().Actions() {
		a[jsonString(v.Command)] = v.ID
	}
	return asteriskRunner{n, a, s.doc().Source, s.doc().Before.Toolchain}
}

type asteriskRunner struct {
	native            AsteriskNative
	actions           map[string]string
	source, toolchain string
}

func (r asteriskRunner) Validate(c execution.Command) error {
	if _, ok := r.actions[jsonString(c)]; !ok || r.source != r.native.Source.Digest() {
		return errAsterisk
	}
	return r.native.Database.Native.Runner.Validate(c)
}
func (r asteriskRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if e := r.Validate(c); e != nil {
		return execution.Result{}, e
	}
	n := r.native
	id := r.actions[jsonString(c)]
	var last execution.Result
	run := func(c execution.Command) error { var e error; last, e = n.run(ctx, c); return e }
	files := func(mode string) error { return run(n.fileCommand(mode, r.toolchain, "")) }
	var e error
	switch id {
	case "asterisk.00.build":
		e = r.build(ctx)
	case "asterisk.01.install":
		e = run(buildCommand("/usr/bin/make", "-C", asteriskTree, "install", "DESTDIR="+asteriskWork+"/dest", "DOWNLOAD_TO_STDOUT=/usr/bin/false"))
		if e == nil {
			e = files("install")
		}
		if e == nil {
			e = r.labels(ctx)
		}
		if e == nil {
			e = run(asteriskCommand("/usr/sbin/ldconfig"))
		}
	case "asterisk.01.libraries", "asterisk.01.resources":
		e = run(asteriskCommand("/usr/bin/systemctl", "stop", "asterisk.service"))
		if e == nil {
			mode := "libraries"
			if id == "asterisk.01.resources" {
				mode = "resources"
			}
			e = files(mode)
		}
		if e == nil {
			e = r.labels(ctx)
		}
		if e == nil {
			e = run(asteriskCommand("/usr/sbin/ldconfig"))
		}
	case "asterisk.02.config":
		e = files("config")
		if e == nil {
			e = r.labels(ctx)
		}
	case "asterisk.03.unit":
		e = files("unit")
		if e == nil {
			e = r.labels(ctx)
		}
		if e == nil {
			e = run(asteriskCommand("/usr/bin/systemctl", "daemon-reload"))
		}
	case "asterisk.04.service":
		e = run(asteriskCommand("/usr/bin/systemctl", "enable", "--now", "asterisk.service"))
		if e == nil {
			e = n.wait(ctx)
		}
	case "asterisk.05.restart":
		e = run(asteriskCommand("/usr/bin/systemctl", "stop", "asterisk.service"))
		if e == nil {
			svc, x := n.service(ctx)
			if x != nil || svc["MainPID"] != "0" || svc["ActiveState"] != "inactive" {
				e = errAsterisk
			}
		}
		if e == nil {
			e = run(asteriskCommand("/usr/bin/systemctl", "start", "asterisk.service"))
		}
		if e == nil {
			e = n.wait(ctx)
		}
		if e == nil {
			svc, x := n.service(ctx)
			if x != nil {
				e = x
			} else {
				e = run(n.fileCommand("restart", r.toolchain, svc["InvocationID"]))
			}
		}
	default:
		e = errAsterisk
	}
	if e != nil {
		if last.Started && last.ExitCode != 0 {
			return last, e
		}
		return execution.Result{Started: true, ExitCode: 1}, e
	}
	return execution.Result{Started: true, ExitCode: 0, Stdout: "S04 bounded action completed; independent postconditions required"}, nil
}
func (n AsteriskNative) wait(ctx context.Context) error {
	for i := 0; i < 30; i++ {
		s, e := n.InspectAsterisk(ctx)
		if e == nil && s.RuntimeOK {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return errAsterisk
}
func buildCommand(exe string, args ...string) execution.Command {
	a := []string{"--chdir=" + asteriskTree, "PATH=/usr/bin:/usr/sbin:/bin", "HOME=" + asteriskWork, "LC_ALL=C", exe}
	a = append(a, args...)
	c := asteriskCommand("/usr/bin/env", a...)
	c.Timeout = 90 * time.Minute
	c.StdoutLimit = 16 << 20
	c.StderrLimit = 16 << 20
	return c
}
func (r asteriskRunner) build(ctx context.Context) error {
	n := r.native
	if _, e := n.run(ctx, n.fileCommand("prepare", r.toolchain, "")); e != nil {
		return e
	}
	for path, b := range map[string][]byte{asteriskWork + "/asterisk.tar.gz": n.Source.archive, asteriskWork + "/cache/pjproject-2.17.tar.bz2": n.Source.pj} {
		if e := exclusiveFile(path, b); e != nil {
			return e
		}
	}
	if _, e := n.run(ctx, n.fileCommand("extract", r.toolchain, "")); e != nil {
		return e
	}
	cmds := []execution.Command{
		buildCommand(asteriskTree+"/configure", "--with-pjproject-bundled", "--with-externals-cache="+asteriskWork+"/cache", "--without-dahdi"),
		buildCommand("/usr/bin/make", "menuselect.makeopts", "DOWNLOAD_TO_STDOUT=/usr/bin/false"),
		buildCommand(asteriskTree+"/menuselect/menuselect", "--disable-category", "MENUSELECT_CORE_SOUNDS", "--disable-category", "MENUSELECT_EXTRA_SOUNDS", "--disable-category", "MENUSELECT_MOH", "--disable", "BUILD_NATIVE", "--enable", "chan_pjsip", "--enable", "res_pjsip", "--enable", "res_pjsip_transport_websocket", "--enable", "res_http_websocket", "--enable", "res_srtp", "--enable", "res_crypto", "--enable", "app_confbridge", "--enable", "app_echo", "--enable", "res_pjsip_config_wizard", "menuselect.makeopts"),
		buildCommand("/usr/bin/make", "-j4", "DOWNLOAD_TO_STDOUT=/usr/bin/false"),
	}
	for j, c := range cmds {
		v, e := n.run(ctx, c)
		log := jsonString(struct {
			Command execution.Command
			Result  execution.Result
		}{c, v})
		if ee := exclusiveFile(fmt.Sprintf("%s/build-%02d.json", asteriskWork, j), []byte(log)); ee != nil {
			return ee
		}
		if e != nil {
			return e
		}
	}
	_, e := n.run(ctx, n.fileCommand("seal-build", r.toolchain, ""))
	return e
}
func exclusiveFile(path string, b []byte) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func (r asteriskRunner) labels(ctx context.Context) error {
	// Only exact newly created S04 resources receive vendor default contexts.
	paths := []string{"/var/lib/asterisk/documentation", "/var/lib/asterisk/documentation/core-en_US.xml", "/var/lib/asterisk/documentation/appdocsxml.xslt", "/var/lib/asterisk/documentation/appdocsxml.dtd", "/usr/sbin/asterisk", "/usr/lib/libasteriskssl.so.1", "/usr/lib/libasteriskpj.so.2", "/etc/systemd/system/asterisk.service", "/usr/lib/asterisk", "/usr/lib/asterisk/modules"}
	for _, m := range AsteriskModules() {
		paths = append(paths, "/usr/lib/asterisk/modules/"+m+".so")
	}
	for k := range asteriskFiles() {
		paths = append(paths, "/etc/asterisk/"+k+".conf", "/etc/asterisk/almadial-s04-"+k+".conf")
	}
	for _, p := range paths {
		if _, e := os.Lstat(p); os.IsNotExist(e) {
			continue
		}
		if _, e := r.native.run(ctx, asteriskCommand("/usr/sbin/restorecon", p)); e != nil {
			return e
		}
	}
	return nil
}
