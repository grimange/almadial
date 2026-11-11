//go:build linux

package diagnostics

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const MaxBundleBytes = 1 << 20

var ErrDestination = errors.New("bundle destination must be a new absolute file in an existing directory with no symlink components")

type Member struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Bundle constructs a bounded archive from projected diagnostic fields only.
// It never accepts a source directory, raw command output, logs or config files.
func Bundle(r Report) ([]byte, error) {
	report, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return nil, e
	}
	redaction := []byte(`{"schema_version":1,"strategy":"allowlisted projections; raw inputs excluded before serialization","excluded":["raw stdout/stderr","raw logs and AVC messages","configuration contents","environment","private keys","passwords and authorization headers","cookies and tokens","contact URIs and caller data","database rows","recording contents","raw journal actions and identities"],"included":["fixed check identifiers and guidance","enumerated health states","validated versions","counts","public manifest and journal SHA-256"],"limits":{"probe_seconds":4,"overall_seconds":90,"probe_stdout_bytes":262144,"probe_stderr_bytes":4096,"archive_and_uncompressed_bytes":1048576},"warnings":"No raw logs are collected. No production readiness or full stage qualification is implied."}`)
	files := map[string][]byte{"report.json": report, "redaction.json": redaction}
	names := []string{"redaction.json", "report.json"}
	members := []Member{}
	for _, n := range names {
		sum := sha256.Sum256(files[n])
		members = append(members, Member{n, len(files[n]), hex.EncodeToString(sum[:])})
	}
	manifest, e := json.MarshalIndent(struct {
		Schema  int      `json:"schema_version"`
		Members []Member `json:"members"`
	}{1, members}, "", "  ")
	if e != nil {
		return nil, e
	}
	files["manifest.json"] = manifest
	names = append(names, "manifest.json")
	sort.Strings(names)
	var checks strings.Builder
	for _, n := range names {
		sum := sha256.Sum256(files[n])
		fmt.Fprintf(&checks, "%x  %s\n", sum, n)
	}
	files["SHA256SUMS"] = []byte(checks.String())
	names = append(names, "SHA256SUMS")
	sort.Strings(names)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	total := 0
	for _, n := range names {
		b := files[n]
		total += len(b)
		if total > MaxBundleBytes {
			return nil, errors.New("diagnostic bundle exceeds size limit")
		}
		if e = tw.WriteHeader(&tar.Header{Name: n, Size: int64(len(b)), Mode: 0600, Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); e != nil {
			return nil, e
		}
		if _, e = tw.Write(b); e != nil {
			return nil, e
		}
	}
	if e = tw.Close(); e != nil {
		return nil, e
	}
	if e = gz.Close(); e != nil {
		return nil, e
	}
	if buf.Len() > MaxBundleBytes {
		return nil, errors.New("diagnostic archive exceeds size limit")
	}
	return buf.Bytes(), nil
}

// WriteBundle traverses directories by fd with O_NOFOLLOW and creates exactly
// one O_EXCL file. Existing destinations, symlinks and path traversal are rejected.
// No directories, temporary files or existing files are modified.
func WriteBundle(path string, b []byte) error {
	if len(b) > MaxBundleBytes || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") || path == "/" {
		return ErrDestination
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	fd, e := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if e != nil {
		return ErrDestination
	}
	for _, p := range parts[:len(parts)-1] {
		next, err := syscall.Openat(fd, p, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		syscall.Close(fd)
		if err != nil {
			return ErrDestination
		}
		fd = next
	}
	defer syscall.Close(fd)
	name := parts[len(parts)-1]
	out, e := syscall.Openat(fd, name, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return ErrDestination
	}
	f := os.NewFile(uintptr(out), "support bundle")
	// A failed write leaves the newly created partial file for inspection.
	// Never unlink by name: a concurrent directory owner could replace it.
	defer f.Close()
	if _, e = io.Copy(f, bytes.NewReader(b)); e != nil {
		return errors.New("cannot write diagnostic bundle")
	}
	if e = f.Sync(); e != nil {
		return errors.New("cannot sync diagnostic bundle")
	}
	if e = f.Close(); e != nil {
		return errors.New("cannot close diagnostic bundle")
	}
	return nil
}
