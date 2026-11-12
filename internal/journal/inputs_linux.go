//go:build linux

package journal

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"regexp"
	"syscall"
)

// Installation inputs preserve the exact approved stage before-image needed by
// Resume. They contain no action states: the existing stage journal remains the
// sole execution record. Publication shares its retained execution ownership.
var inputName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,95}\.input\.json$`)

const inputLimit = 32 << 20

func readInput(dir *os.File, name string) ([]byte, error) {
	if !inputName.MatchString(name) {
		return nil, ErrInvalid
	}
	f, e := regular(dir, name, false)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || st.Size() > inputLimit {
		return nil, ErrInvalid
	}
	return io.ReadAll(io.LimitReader(f, inputLimit+1))
}

// InspectInput is strictly read-only, including when the directory is absent.
func InspectInput(path, name string) ([]byte, error) {
	d, e := openDirectory(path, false)
	if e != nil {
		return nil, e
	}
	defer d.Close()
	return readInput(d, name)
}

// PreserveInput atomically publishes immutable input under the existing flock.
// Identical bytes are a NO_OP; conflicting bytes never replace an earlier input.
func (s *Session) PreserveInput(name string, data []byte) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return e
	}
	if !inputName.MatchString(name) || len(data) == 0 || len(data) > inputLimit {
		return ErrInvalid
	}
	old, e := readInput(s.dir, name)
	if e == nil {
		if bytes.Equal(old, data) {
			return nil
		}
		return ErrInvalid
	}
	if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	var token [16]byte
	if _, e = rand.Read(token[:]); e != nil {
		return e
	}
	tmp := ".input-" + hex.EncodeToString(token[:]) + ".tmp"
	fd, e := syscall.Openat(int(s.dir.Fd()), tmp, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return ErrUnsafe
	}
	f := os.NewFile(uintptr(fd), tmp)
	defer func() { f.Close(); _ = syscall.Unlinkat(int(s.dir.Fd()), tmp) }()
	n, e := f.Write(data)
	if e != nil || n != len(data) {
		s.poisoned = true
		return &PersistenceError{Operation: "input-write"}
	}
	if f.Sync() != nil {
		s.poisoned = true
		return &PersistenceError{Operation: "input-sync"}
	}
	if syscall.Renameat(int(s.dir.Fd()), tmp, int(s.dir.Fd()), name) != nil {
		s.poisoned = true
		return &PersistenceError{Operation: "input-publish"}
	}
	if s.dir.Sync() != nil {
		s.poisoned = true
		return &PersistenceError{Operation: "input-directory-sync", Uncertain: true}
	}
	return nil
}
