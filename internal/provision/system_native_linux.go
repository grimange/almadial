//go:build linux

package provision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func splitOptions(s string) []string { return strings.Split(s, ",") }
func systemCommand(path string, args ...string) execution.Command {
	return execution.Command{Executable: path, Args: args, Timeout: 30 * time.Second, StdoutLimit: 4 << 20, StderrLimit: 16 << 10}
}
func (n Native) systemRead(ctx context.Context, path string, args ...string) (string, error) {
	if ctx == nil || n.Runner == nil {
		return "", errSystem
	}
	r, e := n.Runner.Run(ctx, systemCommand(path, args...))
	if e != nil {
		return "", e
	}
	if !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return "", errSystem
	}
	return strings.TrimSpace(r.Stdout), nil
}

// safeDirectory opens every component relative to its retained parent inode.
// No pathname is followed through a symlink, including intermediate components.
// It neither creates nor changes anything. A missing component is distinct from
// a symlink, regular file, inaccessible path or other ambiguous state.
func safeDirectory(path string) (*os.File, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") {
		return nil, errSystem
	}
	fd, e := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if e != nil {
		return nil, errSystem
	}
	if path != "/" {
		for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
			next, e := syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
			syscall.Close(fd)
			if e != nil {
				if errors.Is(e, syscall.ENOENT) {
					return nil, os.ErrNotExist
				}
				return nil, errSystem
			}
			fd = next
		}
	}
	return os.NewFile(uintptr(fd), "managed-directory"), nil
}
func labelFD(f *os.File) (string, error) {
	b := make([]byte, 4096)
	n, e := syscall.Getxattr("/proc/self/fd/"+strconv.Itoa(int(f.Fd())), "security.selinux", b)
	if e != nil {
		return "", errSystem
	}
	return strings.TrimRight(string(b[:n]), "\x00"), nil
}
func observeDirectory(path string) DirectoryObservation {
	d := DirectoryObservation{Path: path}
	f, e := safeDirectory(path)
	if errors.Is(e, os.ErrNotExist) {
		return d
	}
	if e != nil {
		d.Unsafe = true
		return d
	}
	defer f.Close()
	var st syscall.Stat_t
	if syscall.Fstat(int(f.Fd()), &st) != nil {
		d.Unsafe = true
		return d
	}
	d.Exists = true
	d.UID = st.Uid
	d.GID = st.Gid
	d.Mode = st.Mode & 07777
	d.Device = uint64(st.Dev)
	d.Inode = st.Ino
	d.Label, e = labelFD(f)
	if e != nil {
		d.Unsafe = true
	}
	return d
}
func systemRange() (uint32, uint32, error) {
	b, e := os.ReadFile("/etc/login.defs")
	if e != nil {
		return 0, 0, errSystem
	}
	values := map[string]uint32{}
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(strings.SplitN(l, "#", 2)[0])
		if len(f) == 2 {
			switch f[0] {
			case "UID_MIN", "GID_MIN", "SYS_UID_MAX", "SYS_GID_MAX":
				v, e := strconv.ParseUint(f[1], 10, 32)
				if e != nil {
					return 0, 0, errSystem
				}
				values[f[0]] = uint32(v)
			}
		}
	}
	u, g := values["SYS_UID_MAX"], values["SYS_GID_MAX"]
	// shadow-utils' documented system ranges end immediately before UID_MIN /
	// GID_MIN when explicit SYS_*_MAX values are absent. Never use lab numeric IDs.
	if u == 0 && values["UID_MIN"] > 1 {
		u = values["UID_MIN"] - 1
	}
	if g == 0 && values["GID_MIN"] > 1 {
		g = values["GID_MIN"] - 1
	}
	if u == 0 || g == 0 {
		return 0, 0, errSystem
	}
	return u, g, nil
}
func (n Native) systemNSS(ctx context.Context, s *SystemSnapshot) error {
	r, e := n.systemRead(ctx, "/usr/bin/getent", "passwd")
	if e != nil {
		return e
	}
	for _, l := range strings.Split(r, "\n") {
		f := strings.Split(l, ":")
		if len(f) != 7 {
			return errSystem
		}
		u, e := strconv.ParseUint(f[2], 10, 32)
		if e != nil {
			return errSystem
		}
		g, e := strconv.ParseUint(f[3], 10, 32)
		if e != nil {
			return errSystem
		}
		s.Accounts = append(s.Accounts, AccountObservation{Name: f[0], Home: f[5], Shell: f[6], UID: uint32(u), GID: uint32(g), Groups: []string{}})
	}
	r, e = n.systemRead(ctx, "/usr/bin/getent", "group")
	if e != nil {
		return e
	}
	for _, l := range strings.Split(r, "\n") {
		f := strings.Split(l, ":")
		if len(f) != 4 {
			return errSystem
		}
		g, e := strconv.ParseUint(f[2], 10, 32)
		if e != nil {
			return errSystem
		}
		s.Groups = append(s.Groups, GroupObservation{f[0], uint32(g)})
	}
	for i, a := range s.Accounts {
		for _, p := range ServiceIdentities() {
			if a.Name == p.Name {
				r, e = n.systemRead(ctx, "/usr/bin/id", "-nG", p.Name)
				if e != nil {
					return e
				}
				s.Accounts[i].Groups = strings.Fields(r)
			}
		}
	}
	return nil
}
func (n Native) storage(ctx context.Context, path string) (StorageObservation, error) {
	var s StorageObservation
	for {
		f, e := safeDirectory(path)
		if errors.Is(e, os.ErrNotExist) {
			path = filepath.Dir(path)
			continue
		}
		if e != nil {
			return s, e
		}
		defer f.Close()
		var st syscall.Statfs_t
		if syscall.Fstatfs(int(f.Fd()), &st) != nil || st.Bsize <= 0 {
			return s, errSystem
		}
		s.AvailableBytes = st.Bavail * uint64(st.Bsize)
		s.AvailableInodes = st.Ffree
		break
	}
	r, e := n.systemRead(ctx, "/usr/bin/findmnt", "--json", "--target", path, "--output", "TARGET,SOURCE,FSTYPE,OPTIONS")
	if e != nil {
		return s, e
	}
	var v struct {
		Filesystems []struct{ Target, Source, Fstype, Options string } `json:"filesystems"`
	}
	if decode(r, &v) != nil || len(v.Filesystems) != 1 {
		return s, errSystem
	}
	f := v.Filesystems[0]
	s.Mount = f.Target
	s.Source = f.Source
	s.Filesystem = f.Fstype
	s.Options = f.Options
	return s, nil
}

// InspectSystem performs local read-only observations only. It never refreshes
// DNF metadata, creates state directories or changes SELinux mappings.
func (n Native) InspectSystem(ctx context.Context) (SystemSnapshot, error) {
	var s SystemSnapshot
	var e error
	if s.RPM, e = n.Inspect(ctx); e != nil {
		return s, e
	}
	if s.SELinux, e = n.systemRead(ctx, "/usr/sbin/getenforce"); e != nil {
		return s, e
	}
	b, e := os.ReadFile("/etc/selinux/config")
	if e != nil {
		return s, errSystem
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "SELINUXTYPE=") {
			s.Policy = strings.Trim(strings.TrimPrefix(l, "SELINUXTYPE="), "\" ")
		}
	}
	// Bind vendor and local persistent context definitions. matchpathcon below
	// proves the effective mapping and that its expected SELinux type exists.
	var policies []string
	for _, p := range []string{"file_contexts", "file_contexts.local", "file_contexts.subs", "file_contexts.subs_dist"} {
		b, e := os.ReadFile("/etc/selinux/targeted/contexts/files/" + p)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return s, errSystem
		}
		policies = append(policies, string(b))
	}
	s.LocalContextsSHA256 = hash(policies)
	if s.SystemUIDMax, s.SystemGIDMax, e = systemRange(); e != nil {
		return s, e
	}
	if e = n.systemNSS(ctx, &s); e != nil {
		return s, e
	}
	for _, p := range ManagedDirectories() {
		d := observeDirectory(p.Path)
		d.DefaultContext, e = n.systemRead(ctx, "/usr/sbin/matchpathcon", "-n", p.Path)
		if e != nil {
			return s, e
		}
		s.Directories = append(s.Directories, d)
	}
	if s.Recording, e = n.storage(ctx, "/var/spool/asterisk/monitorDONE"); e != nil {
		return s, e
	}
	live, e := n.storage(ctx, "/var/spool/asterisk/monitor")
	if e != nil {
		return s, e
	}
	if live.Mount != s.Recording.Mount || live.Source != s.Recording.Source || live.Filesystem != s.Recording.Filesystem {
		return s, errSystem
	}
	return s, nil
}

// ApplySystemDirectory is a fixed helper used only by almadial-system's bounded
// child process after durable intent. It exposes no arbitrary path or command.
// It never replaces an existing inode or walks directory contents. Persistent
// vendor mappings are verified before applying their full context via the held
// inode descriptor; a later restorecon/relabel therefore selects the same label.
func ApplySystemDirectory(ctx context.Context, index int, request SystemDirectoryRequest, n Native) error {
	before := request.Before
	ps := ManagedDirectories()
	if ctx == nil || ctx.Err() != nil || index < 0 || index >= len(ps) {
		return errSystem
	}
	p := ps[index]
	if verifySystemHelperIntent(index, request.HostState) != nil {
		return errSystem
	}
	if before.Path != p.Path || before.Unsafe {
		return errSystem
	}
	mode, e := n.systemRead(ctx, "/usr/sbin/getenforce")
	if e != nil || mode != "Enforcing" {
		return errSystem
	}
	label, e := n.systemRead(ctx, "/usr/sbin/matchpathcon", "-n", p.Path)
	if e != nil || label != expectedContext(p) {
		return errSystem
	}
	s := SystemSnapshot{}
	if e = n.systemNSS(ctx, &s); e != nil {
		return e
	}
	a, ok := account(s, p.Owner)
	g, gok := group(s, p.Group)
	if !ok || !gok {
		return errSystem
	}
	parent, e := safeDirectory(filepath.Dir(p.Path))
	if e != nil {
		return e
	}
	defer parent.Close()
	fd, e := syscall.Openat(int(parent.Fd()), filepath.Base(p.Path), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	created := false
	if errors.Is(e, syscall.ENOENT) && !before.Exists {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if syscall.Mkdirat(int(parent.Fd()), filepath.Base(p.Path), 0700) != nil {
			return errSystem
		}
		created = true
		fd, e = syscall.Openat(int(parent.Fd()), filepath.Base(p.Path), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	}
	if e != nil {
		return errSystem
	}
	f := os.NewFile(uintptr(fd), "managed-directory")
	defer f.Close()
	var st syscall.Stat_t
	if syscall.Fstat(fd, &st) != nil {
		return errSystem
	}
	if !created {
		// Only a label repair is authorized for an existing compatible inode.
		if !before.Exists || uint64(st.Dev) != before.Device || st.Ino != before.Inode || st.Uid != a.UID || st.Gid != g.GID || st.Mode&07777 != p.Mode {
			return errSystem
		}
	} else {
		if syscall.Fchown(fd, int(a.UID), int(g.GID)) != nil || syscall.Fchmod(fd, p.Mode) != nil {
			return errSystem
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if syscall.Setxattr("/proc/self/fd/"+strconv.Itoa(fd), "security.selinux", append([]byte(label), 0), 0) != nil {
		return errSystem
	}
	if syscall.Fsync(fd) != nil || syscall.Fsync(int(parent.Fd())) != nil {
		return errSystem
	}
	// Verify the held inode and its current pathname independently. A renamed or
	// substituted destination cannot turn a successful helper exit into success.
	after := observeDirectory(p.Path)
	after.DefaultContext = label
	if !directorySatisfied(s, p, after) || after.Inode != st.Ino || after.Device != uint64(st.Dev) {
		return errSystem
	}
	return nil
}

// A direct invocation of the private helper must not bypass the S02 entry
// gate. The helper requires the same executable as its live parent, a retained
// ALM-14 lock descriptor and a matching durable running record. These checks
// reuse ALM-14's reader and lock; they do not create another lock or journal.
func verifySystemHelperIntent(index int, hostState string) error {
	if !shaPattern.MatchString(hostState) {
		return errSystem
	}
	parent := "/proc/" + strconv.Itoa(os.Getppid())
	self, e := os.Stat("/proc/self/exe")
	if e != nil {
		return errSystem
	}
	other, e := os.Stat(parent + "/exe")
	if e != nil || !os.SameFile(self, other) {
		return errSystem
	}
	fds, e := os.ReadDir(parent + "/fd")
	if e != nil {
		return errSystem
	}
	for _, fd := range fds {
		path, e := os.Readlink(parent + "/fd/" + fd.Name())
		if e != nil || filepath.Base(path) != "apply.lock" {
			continue
		}
		f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if e != nil {
			continue
		}
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
			continue
		}
		f.Close()
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			continue
		}
		d, e := journal.Inspect(filepath.Dir(path))
		if e != nil || d.Identity.HostState != hostState {
			continue
		}
		for _, a := range d.Actions {
			if a.Stage == "S02" && a.ID == fmt.Sprintf("system.02.directory.%02d", index) && a.State == journal.Running {
				return nil
			}
		}
	}
	return errSystem
}
