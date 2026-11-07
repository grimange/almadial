//go:build linux

package journal

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

const lockName = "apply.lock"
const journalName = "journal.json"

// PersistenceError explicitly records the rename durability window. When
// Uncertain is true, the new journal may already be visible. Never assume rollback.
// The syscall error is deliberately not exposed: it may include private paths.
type PersistenceError struct {
	Operation string
	Uncertain bool
}

func (e *PersistenceError) Error() string {
	return fmt.Sprintf("journal persistence failed at %s (durability uncertain: %t); close, inspect and reconcile", e.Operation, e.Uncertain)
}

// Session retains an exclusive Linux flock on a separate stable inode. It is not
// an execution approval. No exported writer bypasses ownership. Close after any
// persistence error; no further transitions are allowed in a poisoned session.
type Session struct {
	mu       sync.Mutex
	dir      *os.File
	lock     *os.File
	data     *Document
	poisoned bool
	hook     func(string) error // test-only fault boundary; never supplied by production callers
	write    func(*os.File, []byte) (int, error)
}

// openDirectory walks every path component with O_NOFOLLOW. Only the final leaf
// may be created, and only by Acquire. Parent directories must already exist.
func openDirectory(path string, create bool) (*os.File, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || strings.ContainsAny(path, "\x00\r\n") {
		return nil, ErrUnsafe
	}
	fd, e := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if e != nil {
		return nil, ErrUnsafe
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for n, p := range parts {
		next, e := syscall.Openat(fd, p, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if errors.Is(e, syscall.ENOENT) && create && n == len(parts)-1 {
			if e = syscall.Mkdirat(fd, p, 0700); e != nil && !errors.Is(e, syscall.EEXIST) {
				syscall.Close(fd)
				return nil, ErrUnsafe
			}
			if e = syscall.Fsync(fd); e != nil {
				syscall.Close(fd)
				return nil, ErrUnsafe
			}
			next, e = syscall.Openat(fd, p, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		}
		syscall.Close(fd)
		if e != nil {
			if errors.Is(e, syscall.ENOENT) {
				return nil, os.ErrNotExist
			}
			return nil, ErrUnsafe
		}
		fd = next
	}
	f := os.NewFile(uintptr(fd), "state-directory")
	var st syscall.Stat_t
	if syscall.Fstat(fd, &st) != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&07777 != 0700 {
		f.Close()
		return nil, ErrUnsafe
	}
	return f, nil
}
func regular(dir *os.File, name string, create bool) (*os.File, error) {
	flags := syscall.O_RDONLY
	if create {
		flags = syscall.O_RDWR | syscall.O_CREAT
	}
	fd, e := syscall.Openat(int(dir.Fd()), name, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0600)
	if e != nil {
		if errors.Is(e, syscall.ENOENT) {
			return nil, os.ErrNotExist
		}
		return nil, ErrUnsafe
	}
	f := os.NewFile(uintptr(fd), name)
	var st syscall.Stat_t
	if syscall.Fstat(fd, &st) != nil || st.Mode&syscall.S_IFMT != syscall.S_IFREG || st.Mode&07777 != 0600 || st.Uid != uint32(os.Geteuid()) || st.Nlink > 1 || (st.Nlink == 0 && (create || name != journalName)) {
		f.Close()
		return nil, ErrUnsafe
	}
	// A reader may hold the old journal across an atomic rename. Its now
	// unlinked inode remains a valid complete snapshot; lock inodes must stay
	// linked. Multiple links are forbidden for both files.
	return f, nil
}
func read(dir *os.File) (Document, error) {
	f, e := regular(dir, journalName, false)
	if e != nil {
		return Document{}, e
	}
	defer f.Close()
	return Decode(f)
}

// Inspect opens existing state read-only. A missing directory/journal returns
// os.ErrNotExist; neither a lock, directory nor temporary file is created.
func Inspect(path string) (Document, error) {
	d, e := openDirectory(path, false)
	if e != nil {
		return Document{}, e
	}
	defer d.Close()
	return read(d)
}

// Acquire fails fast on contention. Cooperating callers must retain the returned
// session for the complete apply/resume operation. Never unlink apply.lock.
func Acquire(path string) (*Session, error) {
	d, e := openDirectory(path, true)
	if e != nil {
		return nil, e
	}
	f, e := regular(d, lockName, true)
	if e != nil {
		d.Close()
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		d.Close()
		if errors.Is(e, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, ErrUnsafe
	}
	s := &Session{dir: d, lock: f, write: func(f *os.File, b []byte) (int, error) { return f.Write(b) }}
	if e = s.owned(); e != nil {
		s.Close()
		return nil, e
	}
	// Persist initial lock entry; its inode remains stable for all future updates.
	if e = d.Sync(); e != nil {
		s.Close()
		return nil, ErrUnsafe
	}
	return s, nil
}
func (s *Session) owned() error {
	if s == nil || s.lock == nil || s.dir == nil || s.poisoned {
		return ErrClosed
	}
	f, e := regular(s.dir, lockName, false)
	if e != nil {
		return ErrUnsafe
	}
	defer f.Close()
	a, e := f.Stat()
	if e != nil {
		return ErrUnsafe
	}
	b, e := s.lock.Stat()
	if e != nil || !os.SameFile(a, b) {
		return ErrUnsafe
	}
	return nil
}
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lock == nil {
		return nil
	}
	// close releases flock in the kernel, including on process termination. The
	// descriptor is CLOEXEC; child processes do not inherit execution ownership.
	e := s.lock.Close()
	s.lock = nil
	if d := s.dir.Close(); e == nil {
		e = d
	}
	s.dir = nil
	return e
}
func (s *Session) Snapshot() (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return Document{}, e
	}
	if s.data == nil {
		return Document{}, ErrInvalid
	}
	return clone(*s.data), nil
}
func (s *Session) Initialize(d Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return e
	}
	if s.data != nil {
		return ErrTransition
	}
	if _, e := read(s.dir); !errors.Is(e, os.ErrNotExist) {
		if e != nil {
			return e
		}
		return ErrTransition
	}
	if e := d.Validate(); e != nil {
		return e
	}
	for _, a := range d.Actions {
		if a.State != Pending {
			return ErrTransition
		}
	}
	return s.persist(d)
}
func (s *Session) boundary(name string) error {
	if s.hook != nil {
		return s.hook(name)
	}
	return nil
}
func (s *Session) persist(d Document) (err error) {
	if e := s.owned(); e != nil {
		return e
	}
	b, e := d.JSON()
	if e != nil {
		return e
	}
	var f *os.File
	tmp := ""
	renamed := false
	op := "create"
	defer func() {
		if f != nil {
			_ = f.Close()
		}
		if tmp != "" && !renamed {
			_ = syscall.Unlinkat(int(s.dir.Fd()), tmp)
		}
		if err != nil {
			s.poisoned = true
			err = &PersistenceError{op, renamed}
		}
	}()
	if err = s.boundary(op); err != nil {
		return err
	}
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return err
	}
	tmp = ".journal-" + hex.EncodeToString(token[:]) + ".tmp"
	fd, e := syscall.Openat(int(s.dir.Fd()), tmp, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return e
	}
	f = os.NewFile(uintptr(fd), tmp)
	op = "write"
	if err = s.boundary(op); err != nil {
		return err
	}
	n, e := s.write(f, b)
	if e != nil {
		return e
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	op = "file_sync"
	if err = s.boundary(op); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	op = "close"
	if err = s.boundary(op); err != nil {
		return err
	}
	err = f.Close()
	f = nil
	if err != nil {
		return err
	}
	op = "rename"
	if err = s.boundary(op); err != nil {
		return err
	}
	// Reject corrupt/unsafe replacement targets, including symlinks and hardlinks.
	if _, e = read(s.dir); e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if err = syscall.Renameat(int(s.dir.Fd()), tmp, int(s.dir.Fd()), journalName); err != nil {
		return err
	}
	renamed = true
	op = "directory_sync"
	if err = s.boundary(op); err != nil {
		return err
	}
	if err = s.dir.Sync(); err != nil {
		return err
	}
	op = "committed"
	if err = s.boundary(op); err != nil {
		return err
	}
	c := clone(d)
	s.data = &c
	return nil
}
