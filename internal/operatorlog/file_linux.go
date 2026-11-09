package operatorlog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// OpenFile creates a NEW 0600 log in an existing operator-owned private directory.
// Every path component is opened without following symlinks. No directories,
// journal or locks are created. Existing files are never replaced or appended.
// Retention is one bounded run per explicit destination; no automatic rotation
// or deletion of operator files is performed.
func OpenFile(path string) (*os.File, error) {
	bad := errors.New("operator log: require a new absolute file in an owned 0700 directory without symlinks")
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return nil, bad
	}
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, bad
	}
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Dir(path), "/"), "/") {
		if part == "" {
			continue
		}
		next, e := syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		syscall.Close(fd)
		if e != nil {
			return nil, bad
		}
		fd = next
	}
	defer syscall.Close(fd)
	var st syscall.Stat_t
	if syscall.Fstat(fd, &st) != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&0777 != 0700 {
		return nil, bad
	}
	logfd, err := syscall.Openat(fd, filepath.Base(path), syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, bad
	}
	return os.NewFile(uintptr(logfd), "operator-log"), nil
}
