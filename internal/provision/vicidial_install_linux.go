//go:build linux

package provision

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"syscall"
)

// The source archive is the only large staging write. Use the retained verified
// bytes, a private held directory and create-if-absent publication. This helper
// is private to the journaled native files action; no CLI exposes it directly.
func stageVICIdialArchive(source VICIdialSource) error {
	if os.Geteuid() != 0 || source.Digest() == "" || rawSHA(source.archive) != databaseArchiveSHA {
		return errVICIdial
	}
	parent, e := safeDirectory("/var/lib/almadial")
	if e != nil {
		return errVICIdial
	}
	defer parent.Close()
	var st syscall.Stat_t
	if syscall.Fstat(int(parent.Fd()), &st) != nil || st.Uid != 0 || st.Gid != 0 || st.Mode&07777 != 0700 {
		return errVICIdial
	}
	if e = syscall.Mkdirat(int(parent.Fd()), "vicidial-s05", 0700); e != nil && !errors.Is(e, syscall.EEXIST) {
		return errVICIdial
	}
	fd, e := syscall.Openat(int(parent.Fd()), "vicidial-s05", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return errVICIdial
	}
	work := os.NewFile(uintptr(fd), "private S05 staging")
	defer work.Close()
	if syscall.Fstat(fd, &st) != nil || st.Uid != 0 || st.Gid != 0 || st.Mode&07777 != 0700 {
		return errVICIdial
	}
	if e = parent.Sync(); e != nil {
		return e
	}
	root, e := os.OpenRoot("/proc/self/fd/" + strconv.Itoa(fd))
	if e != nil {
		return errVICIdial
	}
	defer root.Close()
	if f, e := root.OpenFile("source.zip", os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0); e == nil {
		defer f.Close()
		s, e := f.Stat()
		if e != nil || !s.Mode().IsRegular() || s.Mode().Perm() != 0600 || s.Size() != int64(len(source.archive)) {
			return errVICIdial
		}
		v, ok := s.Sys().(*syscall.Stat_t)
		if !ok || v.Uid != 0 || v.Gid != 0 || v.Nlink != 1 {
			return errVICIdial
		}
		b, e := io.ReadAll(io.LimitReader(f, int64(len(source.archive))+1))
		if e != nil || !bytes.Equal(b, source.archive) {
			return errVICIdial
		}
		return nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return errVICIdial
	}
	nonce := make([]byte, 16)
	if _, e = rand.Read(nonce); e != nil {
		return e
	}
	tmp := ".source-" + hex.EncodeToString(nonce)
	f, e := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return errVICIdial
	}
	defer root.Remove(tmp)
	_, e = f.Write(source.archive)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if e = root.Link(tmp, "source.zip"); e != nil {
		return errVICIdial
	}
	if e = root.Remove(tmp); e != nil {
		return e
	}
	return work.Sync()
}

func installVICIdialFiles(ctx context.Context, source VICIdialSource, runner vicidialInputRunner) (string, error) {
	if ctx == nil || runner == nil {
		return "", errVICIdial
	}
	// Inspect all destinations before even creating a private staging directory.
	before, e := readVICIdialFiles(ctx, source, runner)
	if e != nil {
		return "", e
	}
	files, e := source.Deployment()
	if e != nil {
		return "", e
	}
	if before.State == "COMPLETE" {
		return hash(files), nil
	}
	if e = stageVICIdialArchive(source); e != nil {
		return "", e
	}
	input, e := json.Marshal(struct{ Files []VICIdialFile }{files})
	if e != nil {
		return "", errVICIdial
	}
	r, e := runner.RunInput(ctx, vicidialFilesCommand("install"), input)
	if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return "", errVICIdial
	}
	after, e := readVICIdialFiles(ctx, source, runner)
	if e != nil || after.State != "COMPLETE" {
		return "", errVICIdial
	}
	return hash(files), nil
}
