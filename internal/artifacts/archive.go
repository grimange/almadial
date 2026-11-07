package artifacts

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxExpanded int64 = 2 << 30
const maxEntries = 30000
const maxFile int64 = 256 << 20

// inspect never extracts. All entries, including directories, are checked before
// content is hashed. Only Asterisk's contained links to regular files are allowed
// for inspection; no entry is ever extracted or followed on the filesystem.
func inspect(ctx context.Context, s Spec, f *os.File, size int64) (int, string, error) {
	seen := map[string]bool{}
	records := map[string]string{}
	links := map[string]string{}
	var total int64
	entries := 0
	add := func(name string, mode os.FileMode, n int64, r io.Reader) error {
		entries++
		if entries > maxEntries {
			return errors.New("archive: too many entries")
		}
		name = strings.TrimSuffix(name, "/")
		if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00\r\n") || strings.HasPrefix(name, "/") || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
			return errors.New("archive: unsafe path")
		}
		if seen[name] {
			return errors.New("archive: duplicate entry")
		}
		seen[name] = true
		if mode.IsDir() {
			if n != 0 {
				return errors.New("archive: nonempty directory entry")
			}
			return nil
		}
		if mode&os.ModeSymlink != 0 && s.component == "asterisk" && links[name] != "" {
			return nil
		}
		if !mode.IsRegular() {
			return errors.New("archive: unsupported entry type or link")
		}
		if n < 0 || n > maxFile || total+n > maxExpanded {
			return errors.New("archive: expansion limit")
		}
		total += n
		h := sha256.New()
		got, err := io.Copy(h, io.LimitReader(contextReader{ctx, r}, n+1))
		if err != nil || got != n {
			return errors.New("archive: truncated content")
		}
		relative := name
		if s.root != "" {
			prefix := s.root + "/"
			if !strings.HasPrefix(name, prefix) {
				return errors.New("archive: unexpected source root")
			}
			relative = strings.TrimPrefix(name, prefix)
		}
		records[relative] = hex.EncodeToString(h.Sum(nil))
		return nil
	}
	if s.format == "zip" {
		if err := zipBounds(f, size); err != nil {
			return 0, "", err
		}
		z, err := zip.NewReader(f, size)
		if err != nil {
			return 0, "", errors.New("archive: malformed ZIP")
		}
		for _, e := range z.File {
			if e.UncompressedSize64 > uint64(maxFile) {
				return 0, "", errors.New("archive: expansion limit")
			}
			r, err := e.Open()
			if err != nil {
				return 0, "", errors.New("archive: unreadable ZIP entry")
			}
			err = add(e.Name, e.Mode(), int64(e.UncompressedSize64), r)
			r.Close()
			if err != nil {
				return 0, "", err
			}
		}
	} else {
		var reader io.Reader = f
		if s.format == "tgz" {
			g, err := gzip.NewReader(f)
			if err != nil {
				return 0, "", errors.New("archive: malformed gzip")
			}
			defer g.Close()
			reader = g
		} else if s.format == "tbz" {
			reader = bzip2.NewReader(f)
		} else {
			return 0, "", errors.New("archive: unsupported format")
		}
		// Bound the decompressed stream itself, including headers and padding.
		limited := &io.LimitedReader{R: contextReader{ctx, reader}, N: maxExpanded + 1}
		t := tar.NewReader(limited)
		for {
			e, err := t.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, "", errors.New("archive: malformed tar")
			}
			if e.Typeflag == tar.TypeXGlobalHeader {
				entries++
				if entries > maxEntries {
					return 0, "", errors.New("archive: too many entries")
				}
				if len(e.PAXRecords) != 1 || e.PAXRecords["comment"] == "" {
					return 0, "", errors.New("archive: unsupported global metadata")
				}
				if s.component == "viciphone" && e.PAXRecords["comment"] != "24e98fccfa7b7079fff970e02fca5f002ac1f2cc" {
					return 0, "", errors.New("archive: wrong pinned commit")
				}
				continue
			}
			for k := range e.PAXRecords {
				if strings.HasPrefix(k, "GNU.sparse") {
					return 0, "", errors.New("archive: sparse entries unsupported")
				}
			}
			if e.Typeflag == tar.TypeSymlink && s.component == "asterisk" {
				target := path.Join(path.Dir(e.Name), e.Linkname)
				if e.Linkname == "" || path.IsAbs(e.Linkname) || strings.ContainsAny(e.Linkname, "\\\x00\r\n") || !strings.HasPrefix(target, "asterisk-22.12.0/") {
					return 0, "", errors.New("archive: unsafe symlink target")
				}
				links[e.Name] = target
			} else if e.Typeflag != tar.TypeReg && e.Typeflag != tar.TypeRegA && e.Typeflag != tar.TypeDir {
				return 0, "", errors.New("archive: unsupported entry type or link")
			}
			if err = add(e.Name, e.FileInfo().Mode(), e.Size, t); err != nil {
				return 0, "", err
			}
		}
		// Read through the compressor trailer so truncated gzip/bzip2 and CRC errors
		// cannot hide after tar's end marker.
		if _, err := io.Copy(io.Discard, limited); err != nil {
			return 0, "", errors.New("archive: invalid compressor trailer")
		}
		if limited.N <= 0 {
			return 0, "", errors.New("archive: expansion limit")
		}
	}
	for name, target := range links {
		if _, ok := records[target]; !ok {
			return 0, "", errors.New("archive: symlink must target a regular source file")
		}
		for p := range seen {
			if strings.HasPrefix(p, name+"/") {
				return 0, "", errors.New("archive: entry beneath symlink")
			}
		}
	}
	keys := make([]string, 0, len(records))
	for k := range records {
		keys = append(keys, k)
	}
	// Match sorted(pathlib.Path) in the accepted Python tree verifier exactly:
	// compare path components, not slash-delimited strings or digest records.
	sort.Slice(keys, func(i, j int) bool { return pathLess(keys[i], keys[j]) })
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s  %s\n", records[k], k)
	}
	tree := hex.EncodeToString(h.Sum(nil))
	if len(keys) == 0 {
		return 0, "", errors.New("archive: empty source inventory")
	}
	if s.embeddedPath != "" && (records[s.embeddedPath] != s.embeddedSHA || len(keys) != s.files) {
		return len(keys), tree, errors.New("archive: embedded asset identity mismatch")
	}
	if s.tree != "" && (tree != s.tree || len(keys) != s.files) {
		return len(keys), tree, errors.New("archive: canonical source-tree mismatch")
	}
	return len(keys), tree, nil
}
func pathLess(a, b string) bool {
	x, y := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return len(x) < len(y)
}

// Bound ZIP central-directory allocations before archive/zip reads its inventory.
// Selected sources use ordinary single-disk ZIP; ZIP64/spanned ZIP is unsupported.
func zipBounds(f *os.File, size int64) error {
	n := int64(65557)
	if size < n {
		n = size
	}
	tail := make([]byte, n)
	if _, err := f.ReadAt(tail, size-n); err != nil {
		return errors.New("archive: unreadable ZIP directory")
	}
	for i := len(tail) - 22; i >= 0; i-- {
		if string(tail[i:i+4]) != "PK\x05\x06" {
			continue
		}
		e := tail[i:]
		u16 := func(i int) uint16 { return binary.LittleEndian.Uint16(e[i : i+2]) }
		u32 := func(i int) uint32 { return binary.LittleEndian.Uint32(e[i : i+4]) }
		if int(u16(20))+22 != len(e) {
			continue
		}
		if u16(4) != 0 || u16(6) != 0 || u16(8) != u16(10) || u16(10) == 65535 || int(u16(10)) > maxEntries || u32(12) > 16<<20 || uint64(u32(16))+uint64(u32(12)) > uint64(size-n+int64(i)) {
			return errors.New("archive: unsupported or oversized ZIP directory")
		}
		return nil
	}
	return errors.New("archive: malformed ZIP end record")
}
