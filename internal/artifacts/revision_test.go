package artifacts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Opt-in evidence check against an already exported real upstream revision.
// Does not fetch, install, or execute source. Output remains protected evidence.
func TestAcceptedRevisionExportEvidence(t *testing.T) {
	dir := os.Getenv("ALMADIAL_TEST_SVN_EXPORT")
	output := os.Getenv("ALMADIAL_TEST_CANONICAL_OUTPUT")
	if dir == "" || output == "" {
		t.Skip("accepted export evidence not requested")
	}
	m, err := Load(bytes.NewReader(manifestFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	s, err := m.Artifact("vicidial")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := packRevision(context.Background(), s, dir, f); err != nil {
		t.Fatal(err)
	}
	f.Seek(0, 0)
	r, err := verifyInstallFile(context.Background(), s, f)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(r)
	t.Log(string(b))
}

// Opt-in acquisition from public HTTPS upstreams, reusing only the separately
// verified SVN representation. This workstation check is not the external gate.
func TestPublicAcquisitionEvidence(t *testing.T) {
	dir := os.Getenv("ALMADIAL_TEST_PUBLIC_CACHE")
	if dir == "" {
		t.Skip("public acquisition evidence not requested")
	}
	m, err := Load(bytes.NewReader(manifestFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, component := range installComponents {
		s, err := m.Artifact(component)
		if err != nil {
			t.Fatal(err)
		}
		cache := filepath.Join(dir, s.sha256)
		if err := os.MkdirAll(cache, 0700); err != nil {
			t.Fatal(err)
		}
		r, err := Fetch(context.Background(), s, FetchOptions{Directory: cache})
		if err != nil {
			t.Fatal(component, err)
		}
		if component == "asterisk" {
			r, err = VerifyAsteriskSignature(context.Background(), s, filepath.Join(cache, component+".artifact"), "../releaseassets/asterisk.asc", "../releaseassets/asterisk-keyring.gpg", privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
		}
		b, _ := json.Marshal(r)
		t.Log(string(b))
	}
}

func revisionFixture(t *testing.T) (Spec, revisionExporter) {
	t.Helper()
	data := []byte("accepted source\n")
	s := Spec{component: "vicidial", version: "r4037/schema1748", sha256: strings.Repeat("a", 64), format: "zip", root: "2026-11-05", files: 1}
	s.tree = digestBytes([]byte(digestBytes(data) + "  source.pl\n"))
	return s, func(_ context.Context, destination, _ string) error {
		if err := os.Mkdir(destination, 0700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, "source.pl"), data, 0600)
	}
}

func TestRevisionFallbackAndOfflineCache(t *testing.T) {
	s, exporter := revisionFixture(t)
	hits, exports := 0, 0
	server, roots := tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.WriteHeader(404) }))
	s.source = server.URL
	export := func(ctx context.Context, destination, config string) error {
		exports++
		return exporter(ctx, destination, config)
	}
	dir := privateDir(t)
	r, err := fetch(context.Background(), s, FetchOptions{dir, roots}, export)
	if err != nil || r.Identity != Verified || r.ContentPolicy != "svn-revision-tree-sha256" || r.ExpectedSHA256 != "" || r.ReferenceArchiveSHA256 != s.sha256 || r.TreeSHA256 != s.tree || r.Publisher != NotVerified || r.ActualSHA256 == s.sha256 {
		t.Fatal(r, err)
	}
	path := filepath.Join(dir, "vicidial.artifact")
	first, _ := os.ReadFile(path)
	if _, err := VerifyLocal(context.Background(), s, path); err == nil {
		t.Fatal("reconstructed ZIP falsely accepted as original archive")
	}
	if _, err := VerifyInstallInput(context.Background(), s, path); err != nil {
		t.Fatal(err)
	}
	server.Close()
	// Cache verification never needs an origin, SVN binary, or network.
	if _, err := fetch(context.Background(), s, FetchOptions{dir, nil}, func(context.Context, string, string) error { t.Fatal("offline cache exported SVN"); return nil }); err != nil {
		t.Fatal(err)
	}
	if hits != 1 || exports != 1 {
		t.Fatal(hits, exports)
	}
	// Corrupt cache is rejected and quarantined, then the same source is reacquired.
	server, roots = tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	s.source = server.URL
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := fetch(context.Background(), s, FetchOptions{dir, roots}, export); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if !bytes.Equal(first, second) || exports != 2 {
		t.Fatal("revision representation is not deterministic")
	}
	quarantine, _ := filepath.Glob(path + ".rejected-*")
	if len(quarantine) != 1 {
		t.Fatal("corruption was not retained separately")
	}
}

func TestRevisionFailuresNeverPublish(t *testing.T) {
	for _, failure := range []string{"wrong revision", "symlink", "unavailable", "canceled"} {
		t.Run(failure, func(t *testing.T) {
			s, good := revisionFixture(t)
			server, roots := tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
			s.source = server.URL
			dir := privateDir(t)
			export := func(ctx context.Context, destination, config string) error {
				if failure == "unavailable" {
					return errors.New("upstream unavailable")
				}
				if failure == "canceled" {
					<-ctx.Done()
					return ctx.Err()
				}
				if err := good(ctx, destination, config); err != nil {
					return err
				}
				if failure == "symlink" {
					return os.Symlink("/etc/passwd", filepath.Join(destination, "link"))
				}
				return os.WriteFile(filepath.Join(destination, "source.pl"), []byte("wrong source revision"), 0600)
			}
			ctx, cancel := context.WithCancel(context.Background())
			if failure == "canceled" {
				cancel()
			}
			defer cancel()
			r, err := fetch(ctx, s, FetchOptions{dir, roots}, export)
			if err == nil || r.Identity == Verified {
				t.Fatal(r, err)
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 0 {
				t.Fatal("failed prerequisite published partial input", entries)
			}
		})
	}
}
