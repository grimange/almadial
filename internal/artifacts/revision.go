package artifacts

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

const vicidialSVN = "svn://svn.eflo.net:3690/agc_2-X/trunk"

// The validated manifest supplies the independently accepted full-tree digest.
// This is a second representation of that source, never an archive checksum
// exception for other components, versions, or arbitrary operator inputs.
func revisionSpec(s Spec) bool {
	return s.valid() && s.component == "vicidial" && s.version == "r4037/schema1748" && s.format == "zip" && s.root == "2026-11-05" && len(s.tree) == 64 && s.files > 0
}

func revisionComment(s Spec) string {
	return "AlmaDial canonical SVN r4037; tree-sha256=" + s.tree
}

type revisionExporter func(context.Context, string, string) error

func exportSVN(ctx context.Context, destination, config string) error {
	if _, err := os.Stat("/usr/bin/svn"); err != nil {
		return errors.New("SVN utility missing; install the signed AlmaLinux subversion package, then rerun almadial install")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/svn", "export", "--non-interactive", "--no-auth-cache", "--ignore-externals", "--quiet", "--native-eol", "LF", "--config-dir", config, "-r", "4037", vicidialSVN+"@4037", destination)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	// Do not inherit user SVN configuration, proxy settings or credential helpers.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C", "HOME=" + filepath.Dir(config)}
	if err := cmd.Run(); err != nil {
		return errors.New("authoritative SVN r4037 export failed; check outbound TCP 3690 and upstream availability")
	}
	return nil
}

func fetchRevision(ctx context.Context, s Spec, root *os.Root, final string, export revisionExporter) (Result, error) {
	r := initial(s)
	if !revisionSpec(s) {
		return r, errors.New("revision acquisition is not approved for this source")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return r, err
	}
	name := ".revision-" + hex.EncodeToString(nonce)
	if err := root.Mkdir(name, 0700); err != nil {
		return r, err
	}
	defer root.RemoveAll(name)
	base := filepath.Join(root.Name(), name)
	destination := filepath.Join(base, "tree")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// Bound an untrusted export while it is arriving, as well as before packing.
	done := make(chan error, 1)
	go func() { done <- export(ctx, destination, filepath.Join(base, "config")) }()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				return r, err
			}
			goto exported
		case <-ctx.Done():
			cancel()
			<-done
			return r, ctx.Err()
		case <-ticker.C:
			if _, err := exportFiles(destination); err != nil && !os.IsNotExist(err) {
				cancel()
				<-done
				return r, err
			}
		}
	}
exported:
	f, err := root.OpenFile(name+"/canonical.zip", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return r, err
	}
	defer f.Close()
	if err = packRevision(ctx, s, destination, f); err != nil {
		return r, err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return r, err
	}
	r, err = verifyInstallFile(ctx, s, f)
	if err != nil {
		return r, err
	}
	if err = f.Sync(); err != nil {
		return r, err
	}
	if err = root.Link(name+"/canonical.zip", final); err != nil {
		return r, errors.New("revision cache publication collision; reverify existing artifact")
	}
	return r, nil
}

func exportFiles(directory string) ([]string, error) {
	var files []string
	var total int64
	entries := 0
	err := filepath.WalkDir(directory, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries++
		if entries > maxEntries {
			return errors.New("revision export exceeds entry limit")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxFile || info.Size() < 0 {
			return errors.New("revision export contains unsupported or oversized file")
		}
		total += info.Size()
		if total > maxExpanded {
			return errors.New("revision export exceeds expansion limit")
		}
		files = append(files, name)
		return nil
	})
	sort.Strings(files)
	return files, err
}

func packRevision(ctx context.Context, s Spec, directory string, f *os.File) error {
	files, err := exportFiles(directory)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	if err = w.SetComment(revisionComment(s)); err != nil {
		return err
	}
	for _, name := range files {
		rel, err := filepath.Rel(directory, name)
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: s.root + "/" + filepath.ToSlash(rel), Method: zip.Store}
		h.SetMode(0644)
		h.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		out, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		in, err := openRegular(name, maxFile)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, contextReader{ctx, in})
		in.Close()
		if err != nil {
			return err
		}
		if n, err := f.Seek(0, io.SeekCurrent); err != nil || n > maxArchiveBytes {
			return fmt.Errorf("revision archive exceeds limit")
		}
	}
	return w.Close()
}
