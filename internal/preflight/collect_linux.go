package preflight

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Operation is a closed read-only command set, not an arbitrary command runner.
type Operation string

const (
	Sockets  Operation = "sockets"
	Packages Operation = "packages"
	Services Operation = "services"
	Firewall Operation = "firewall"
	Clock    Operation = "clock"
	Network  Operation = "network"
)

func command(op Operation) (string, []string, error) {
	switch op {
	case Network:
		return "/usr/sbin/ip", []string{"-j", "address", "show"}, nil
	case Sockets:
		return "/usr/sbin/ss", []string{"-H", "-lntu", "-n"}, nil
	case Packages:
		return "/usr/bin/rpm", []string{"-qa", "--qf", "%{NAME}\n"}, nil
	case Services:
		return "/usr/bin/systemctl", []string{"list-units", "--all", "--type=service", "--no-legend", "--no-pager", "--plain"}, nil
	case Firewall:
		return "/usr/sbin/nft", []string{"--json", "--stateless", "list", "ruleset"}, nil
	case Clock:
		return "/usr/bin/timedatectl", []string{"show", "--property=NTPSynchronized", "--value"}, nil
	default:
		return "", nil, errors.New("unsupported read-only probe")
	}
}

// Source supports only reads. Fakes can record every operation; production has
// no write/install/service-management API. Implementations must honor context.
type Source interface {
	Read(string) ([]byte, error)
	Architecture() (string, error)
	Processes() ([]string, error)
	Disk(string) (Disk, error)
	Exists(string) (bool, error)
	Run(context.Context, Operation) ([]byte, error)
}
type Linux struct{}

func (Linux) Architecture() (string, error) {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return "", err
	}
	var b []byte
	for _, c := range u.Machine {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	if string(b) == "x86_64" {
		return "amd64", nil
	}
	return string(b), nil
}
func (l Linux) Processes() ([]string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if _, err := strconv.ParseUint(entry.Name(), 10, 32); err != nil {
			continue
		}
		b, e := l.Read("/proc/" + entry.Name() + "/comm")
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return nil, errors.New("process inspection incomplete")
		}
		names = append(names, strings.TrimSpace(string(b)))
	}
	return names, nil
}

func (Linux) Read(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	if len(b) > 4*1024*1024 {
		return nil, errors.New("probe output exceeds limit")
	}
	return b, err
}
func (Linux) Disk(path string) (Disk, error) {
	var st syscall.Statfs_t
	err := syscall.Statfs(path, &st)
	return Disk{path, st.Blocks * uint64(st.Bsize), st.Bavail * uint64(st.Bsize)}, err
}
func (Linux) Exists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// CommandError records process status without retaining output or host paths.
type CommandError struct{ ExitCode int }

func (e *CommandError) Error() string { return "read-only command unavailable or failed" }

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4*1024*1024 {
		return 0, errors.New("probe output exceeds limit")
	}
	return b.Buffer.Write(p)
}
func (Linux) Run(ctx context.Context, op Operation) ([]byte, error) {
	path, args, err := command(op)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin", "LANG=C", "LC_ALL=C", "SYSTEMD_PAGER=cat"}
	var b boundedOutput
	cmd.Stdout = &b
	cmd.Stderr = io.Discard
	err = cmd.Run()
	if err != nil {
		// Minimal cloud images may omit nft. A modular, currently unloaded
		// nf_tables implementation cannot contain an nftables ruleset. Never
		// substitute this proof for a permission error or a loaded/built-in
		// implementation; those still require the native ruleset inspection.
		if op == Firewall && os.IsNotExist(err) && unloadedNFTables(Linux{}) {
			return []byte(`{"nftables":[]}`), nil
		}
		code := -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		}
		return nil, &CommandError{code}
	}
	return b.Bytes(), nil
}

func unloadedNFTables(src Source) bool {
	loaded, err := src.Exists("/sys/module/nf_tables")
	if err != nil || loaded {
		return false
	}
	kernel, err := src.Read("/proc/sys/kernel/osrelease")
	name := strings.TrimSpace(string(kernel))
	if err != nil || name == "" || strings.ContainsAny(name, "/\\\x00\n\r") {
		return false
	}
	config, err := src.Read("/boot/config-" + name)
	if err != nil || !strings.Contains("\n"+string(config), "\nCONFIG_NF_TABLES=m\n") {
		return false
	}
	loaded, err = src.Exists("/sys/module/nf_tables")
	return err == nil && !loaded
}
func Collect(ctx context.Context, src Source) Snapshot {
	s := Snapshot{Known: make(map[string]bool)}
	architecture, archErr := src.Architecture()
	s.Architecture = architecture
	s.Known["architecture"] = archErr == nil && architecture != ""
	run := func(op Operation) ([]byte, error) {
		b, e := src.Run(ctx, op)
		if e != nil {
			code := -1
			var ce *CommandError
			if errors.As(e, &ce) {
				code = ce.ExitCode
			}
			s.Failures = append(s.Failures, ProbeFailure{string(op), code})
		}
		return b, e
	}
	read := func(path string) string {
		b, err := src.Read(path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	s.HostID = read("/etc/machine-id")
	_, hostErr := hex.DecodeString(s.HostID)
	s.Known["identity"] = len(s.HostID) == 32 && hostErr == nil
	osrelease := read("/etc/os-release")
	for _, line := range strings.Split(osrelease, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, "\"")
		switch k {
		case "ID":
			s.OS = v
		case "VERSION_ID":
			s.Release = v
		}
	}
	bnet, enet := run(Network)
	if enet == nil {
		var devices []struct {
			AddressInfo []struct {
				Local string `json:"local"`
			} `json:"addr_info"`
		}
		if json.Unmarshal(bnet, &devices) == nil && len(devices) > 0 {
			s.Known["network"] = true
			for _, d := range devices {
				for _, i := range d.AddressInfo {
					a, e := netip.ParseAddr(i.Local)
					if e != nil {
						s.Known["network"] = false
					} else {
						s.Addresses = append(s.Addresses, a)
					}
				}
			}
		}
	}
	s.DNSConfigured = strings.Contains(read("/etc/resolv.conf"), "nameserver ")
	s.Kernel = read("/proc/sys/kernel/osrelease")
	s.Known["os"] = s.OS != "" && s.Release != "" && s.Kernel != ""
	cpu := read("/proc/cpuinfo")
	var shared map[string]bool
	flagSections := 0
	for _, line := range strings.Split(cpu, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "processor":
			s.CPUs++
		case "flags":
			flagSections++
			flags := map[string]bool{}
			for _, f := range strings.Fields(v) {
				flags[f] = true
			}
			if shared == nil {
				shared = flags
			} else {
				for f := range shared {
					if !flags[f] {
						delete(shared, f)
					}
				}
			}
		}
	}
	for f := range shared {
		s.CPUFlags = append(s.CPUFlags, f)
	}
	s.Known["cpu"] = archErr == nil && s.CPUs > 0 && len(shared) > 0 && flagSections == s.CPUs
	mem := read("/proc/meminfo")
	memFields := map[string]bool{}
	for _, line := range strings.Split(mem, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[2] != "kB" {
			continue
		}
		n, err := strconv.ParseUint(fields[1], 10, 54)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			s.TotalMemory = n * 1024
			memFields["total"] = true
		case "MemAvailable:":
			s.AvailableMemory = n * 1024
			memFields["available"] = true
		}
	}
	s.Known["memory"] = s.TotalMemory > 0 && s.AvailableMemory <= s.TotalMemory && memFields["total"] && memFields["available"]
	s.Known["disk"] = true
	for _, path := range []string{"/", "/usr", "/var"} {
		d, err := src.Disk(path)
		if err != nil || d.Total == 0 || d.Available > d.Total {
			s.Known["disk"] = false
		} else {
			s.Disks = append(s.Disks, d)
		}
	}
	b, err := run(Sockets)
	if err == nil {
		s.Listeners, err = parseSockets(string(b))
		s.Known["listeners"] = err == nil
	}
	b, err = run(Packages)
	if err == nil {
		s.Known["packages"] = len(bytes.TrimSpace(b)) > 0
		for _, name := range strings.Fields(string(b)) {
			if packageConflict(name) {
				s.Packages = append(s.Packages, name)
			}
		}
	}
	b, err = run(Services)
	if err == nil {
		s.Known["services"] = true
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if line == "" {
				continue
			}
			f := strings.Fields(line)
			if len(f) < 4 {
				s.Known["services"] = false
				continue
			}
			if f[0] == "nftables.service" || f[0] == "firewalld.service" {
				s.FirewallServices = append(s.FirewallServices, strings.Join(f[:4], " "))
			}
			if relevant(f[0]) {
				s.Services = append(s.Services, strings.Join(f[:4], " "))
			}
		}
	}
	names, procErr := src.Processes()
	s.Known["processes"] = procErr == nil
	for _, name := range names {
		if relevant(name) || name == "mysqld" || name == "mariadbd" || strings.HasPrefix(name, "AST_") || strings.HasPrefix(name, "ADMIN_") {
			s.Processes = append(s.Processes, name)
		}
	}
	s.Known["ownership"] = true
	for _, path := range []string{"/etc/asterisk", "/etc/astguiclient.conf", "/usr/share/astguiclient", "/srv/www/htdocs/vicidial", "/var/lib/mysql", "/var/lib/almadial"} {
		exists, err := src.Exists(path)
		if err != nil {
			s.Known["ownership"] = false
		}
		if exists {
			s.OwnedPaths = append(s.OwnedPaths, path)
		}
	}
	switch read("/sys/fs/selinux/enforce") {
	case "1":
		s.SELinux = "Enforcing"
		s.Known["selinux"] = true
	case "0":
		s.SELinux = "Permissive"
		s.Known["selinux"] = true
	default:
		exists, err := src.Exists("/sys/fs/selinux")
		if err == nil && !exists {
			s.SELinux = "Disabled"
			s.Known["selinux"] = true
		}
	}
	b, err = run(Firewall)
	if err == nil {
		var rules struct {
			Nftables []map[string]any `json:"nftables"`
		}
		if json.Unmarshal(b, &rules) != nil || rules.Nftables == nil {
			s.Known["firewall"] = false
		} else {
			s.Known["firewall"] = true
			material := []map[string]any{}
			for _, entry := range rules.Nftables {
				if len(entry) != 1 {
					s.Known["firewall"] = false
					continue
				}
				if _, metadata := entry["metainfo"]; !metadata {
					material = append(material, entry)
				}
			}
			s.FirewallSHA256 = hash(material)
			if len(material) == 0 {
				s.Firewall = "empty nftables ruleset"
			} else {
				s.Firewall = "nftables rules present"
			}
		}
	}
	b, err = run(Clock)
	if err == nil {
		s.TimeSync = strings.TrimSpace(string(b))
		s.Known["time"] = s.TimeSync == "yes" || s.TimeSync == "no"
	}
	sort.Strings(s.CPUFlags)
	sort.Strings(s.Packages)
	sort.Strings(s.Services)
	sort.Strings(s.Processes)
	sort.Strings(s.OwnedPaths)
	sort.Strings(s.FirewallServices)
	sort.Slice(s.Addresses, func(i, j int) bool { return s.Addresses[i].Less(s.Addresses[j]) })
	sort.Slice(s.Listeners, func(i, j int) bool {
		a, b := s.Listeners[i], s.Listeners[j]
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		if a.Address != b.Address {
			return a.Address.Less(b.Address)
		}
		return a.Port < b.Port
	})
	return s
}
func relevant(name string) bool {
	for _, p := range []string{"asterisk", "mariadb", "mysql", "httpd", "vicidial", "astguiclient"} {
		if name == p || strings.HasPrefix(name, p+"-") || strings.HasPrefix(name, p+".") {
			return true
		}
	}
	return false
}

func packageConflict(name string) bool {
	for _, p := range []string{"asterisk", "mariadb-server", "mysql-server", "httpd", "vicidial", "astguiclient"} {
		if name == p || strings.HasPrefix(name, p+"-") {
			return true
		}
	}
	return false
}
