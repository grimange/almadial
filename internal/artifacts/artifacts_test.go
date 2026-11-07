package artifacts

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func manifestFixture(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../../manifests/stack.lock.yaml")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestManifest(t *testing.T) {
	b := manifestFixture(t)
	m, e := Load(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	if m.Digest() != digestBytes(b) || m.Status() != "candidate" || m.InstallationAuthorized() {
		t.Fatal("incorrect identity/authority")
	}
	pkgs := m.Packages()
	pkgs["php"] = RPM{}
	if m.Packages()["php"].Repository != "appstream" {
		t.Fatal("mutable package policy")
	}
	for _, name := range []string{"asterisk", "vicidial", "viciphone", "pjproject", "net-telnet"} {
		s, e := m.Artifact(name)
		if e != nil || !s.valid() {
			t.Fatal(name, e)
		}
	}
	if _, e = m.Artifact("latest"); e == nil {
		t.Fatal("unknown component accepted")
	}
	cases := []struct{ name, old, new string }{
		{"schema", "schema_version: 1", "schema_version: 2"},
		{"schema type", "schema_version: 1", "schema_version: \"1\""},
		{"version", "version: 22.12.0", "version: latest"},
		{"revision", "revision: 4037", "revision: 4038"},
		{"unsupported provenance", "provenance_state: official_https_snapshot_byte_matched_to_pinned_export", "provenance_state: unknown-signature"},
		{"missing digest", "  sha256: f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2\n", ""},
		{"invalid digest", "f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2", "secret-SENSITIVE"},
		{"unknown origin", "https://downloads.asterisk.org/", "https://untrusted.invalid/"},
		{"unknown field", "profile: almalinux10-single", "secret-SENSITIVE: nope\nprofile: almalinux10-single"},
		{"duplicate component", "asterisk:\n", "asterisk: {}\nasterisk:\n"},
		{"duplicate leaf", "schema_version: 1", "schema_version: 1\nschema_version: 1"},
		{"alias", "schema_version: 1", "schema_version: &version 1"},
		{"candidate promotion", "status: candidate", "status: accepted"},
		{"package origin", "repository: appstream", "repository: unknown"},
		{"malformed", "schema_version: 1", "schema_version: ["},
		{"wrong structure", "asterisk:\n", "asterisk: []\nwrong:\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, e := Load(strings.NewReader(strings.Replace(string(b), tt.old, tt.new, 1)))
			if e == nil {
				t.Fatal("accepted invalid manifest")
			}
			if strings.Contains(e.Error(), "SENSITIVE") {
				t.Fatal("sensitive input echoed")
			}
		})
	}
	for _, suffix := range []string{"\n---\n{}", "\n---\n", "\npassword: secret-SENSITIVE"} {
		if _, e = Load(bytes.NewReader(append(append([]byte{}, b...), suffix...))); e == nil {
			t.Fatal("extra document/field accepted")
		}
	}
	if _, e = Load(strings.NewReader(strings.Repeat("x", manifestLimit+1))); e == nil {
		t.Fatal("oversize accepted")
	}
	if _, e = (Manifest{}).Artifact("asterisk"); e == nil {
		t.Fatal("zero manifest trusted")
	}
}

type entry struct {
	name, body string
	kind       byte
}

func tgz(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var b bytes.Buffer
	g := gzip.NewWriter(&b)
	w := tar.NewWriter(g)
	for _, e := range entries {
		k := e.kind
		if k == 0 {
			k = tar.TypeReg
		}
		size := int64(len(e.body))
		if k != tar.TypeReg {
			size = 0
		}
		h := &tar.Header{Name: e.name, Mode: 0600, Size: size, Typeflag: k}
		if k == tar.TypeSymlink || k == tar.TypeLink {
			h.Linkname = "../../escape"
		}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if size > 0 {
			if _, err := io.WriteString(w, e.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	w.Close()
	g.Close()
	return b.Bytes()
}
func fixtureSpec(b []byte) Spec {
	return Spec{component: "fixture", version: "1", source: "https://approved.invalid/source.tar.gz", sha256: digestBytes(b), format: "tgz"}
}
func writeFixture(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "artifact")
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestLocalAndArchive(t *testing.T) {
	good := tgz(t, entry{"source/a", "first", 0}, entry{"source/b", "second", 0})
	s := fixtureSpec(good)
	p := writeFixture(t, good)
	a, e := VerifyLocal(context.Background(), s, p)
	if e != nil {
		t.Fatal(e)
	}
	b, e := VerifyLocal(context.Background(), s, p)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("not deterministic", e)
	}
	if a.Identity != Verified || a.Publisher != NotVerified || a.Transport != NotVerified || a.InstallationAuthorized || a.Disposition == Verified {
		t.Fatal("identity mistaken for provenance/authority")
	}
	s.tree = a.TreeSHA256
	s.files = 2
	if _, e = VerifyLocal(context.Background(), s, p); e != nil {
		t.Fatal(e)
	}
	s.files = 3
	if _, e = VerifyLocal(context.Background(), s, p); e == nil {
		t.Fatal("incorrect tree count accepted")
	}
	s = fixtureSpec(good)
	s.tree = strings.Repeat("0", 64)
	if _, e = VerifyLocal(context.Background(), s, p); e == nil {
		t.Fatal("incorrect tree digest accepted")
	}
	bads := map[string][]byte{
		"malformed": []byte("not gzip"), "truncated": good[:len(good)-5],
		"traversal": tgz(t, entry{"../escape", "x", 0}), "absolute": tgz(t, entry{"/escape", "x", 0}),
		"duplicate": tgz(t, entry{"x", "a", 0}, entry{"x", "b", 0}),
		"symlink":   tgz(t, entry{"link", "", tar.TypeSymlink}), "hardlink": tgz(t, entry{"link", "", tar.TypeLink}),
		"device": tgz(t, entry{"device", "", tar.TypeChar}), "newline": tgz(t, entry{"a\nb", "x", 0}),
	}
	for name, data := range bads {
		t.Run(name, func(t *testing.T) {
			r, e := VerifyLocal(context.Background(), fixtureSpec(data), writeFixture(t, data))
			if e == nil || r.Identity == Verified {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	if r, e := VerifyLocal(context.Background(), fixtureSpec(good), writeFixture(t, append(good, 'x'))); e == nil || r.Identity != Rejected {
		t.Fatal("bad digest accepted")
	}
	if _, e = VerifyLocal(context.Background(), fixtureSpec(good), filepath.Join(t.TempDir(), "missing")); e == nil {
		t.Fatal("missing accepted")
	}
	if _, e = VerifyLocal(context.Background(), Spec{}, p); e == nil {
		t.Fatal("uninitialized spec accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = VerifyLocal(ctx, fixtureSpec(good), p); e == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestZIP(t *testing.T) {
	for _, name := range []string{"root/a", "../escape", "/escape", "root/a/../b"} {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			w := zip.NewWriter(&b)
			f, _ := w.Create(name)
			f.Write([]byte("a"))
			w.Close()
			s := fixtureSpec(b.Bytes())
			s.format = "zip"
			s.root = "root"
			_, e := VerifyLocal(context.Background(), s, writeFixture(t, b.Bytes()))
			if (e == nil) != (name == "root/a") {
				t.Fatal(e)
			}
		})
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for i := 0; i < 2; i++ {
		f, _ := w.Create("duplicate")
		f.Write([]byte("x"))
	}
	w.Close()
	s := fixtureSpec(b.Bytes())
	s.format = "zip"
	if _, e := VerifyLocal(context.Background(), s, writeFixture(t, b.Bytes())); e == nil {
		t.Fatal("duplicate ZIP accepted")
	}
}
func privateDir(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	if e := os.Chmod(p, 0700); e != nil {
		t.Fatal(e)
	}
	return p
}
func tlsFixture(t *testing.T, handler http.Handler) (*httptest.Server, *x509.CertPool) {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	return server, pool
}
func TestFetch(t *testing.T) {
	data := tgz(t, entry{"source/file", "content", 0})
	hits := 0
	server, roots := tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.Write(data) }))
	s := fixtureSpec(data)
	s.source = server.URL + "/source"
	dir := privateDir(t)
	r, e := Fetch(context.Background(), s, FetchOptions{dir, roots})
	if e != nil || r.Identity != Verified || r.Transport != Verified || r.Publisher != Verified {
		t.Fatal(r, e)
	}
	r, e = Fetch(context.Background(), s, FetchOptions{dir, roots})
	if e != nil || hits != 1 || r.Transport != NotVerified {
		t.Fatal("cache fetch or fabricated transport", r, e)
	}
	if e = os.WriteFile(filepath.Join(dir, "fixture.artifact"), []byte("corruption"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Fetch(context.Background(), s, FetchOptions{dir, roots}); e == nil || hits != 1 {
		t.Fatal("cache corruption accepted/downloaded over")
	}
	if _, e = Fetch(context.Background(), s, FetchOptions{privateDir(t), nil}); e == nil {
		t.Fatal("untrusted TLS accepted")
	}
	// httptest's CA certificate covers example.com and loopback IPs, not localhost.
	s.source = strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/source"
	if _, e = Fetch(context.Background(), s, FetchOptions{privateDir(t), roots}); e == nil {
		t.Fatal("hostname mismatch accepted")
	}
	s.source = "http://example.invalid/secret-SENSITIVE"
	if _, e = Fetch(context.Background(), s, FetchOptions{privateDir(t), roots}); e == nil || strings.Contains(e.Error(), "SENSITIVE") {
		t.Fatal("unsafe URL accepted/leaked")
	}
}
func TestFetchFailures(t *testing.T) {
	data := tgz(t, entry{"a", "b", 0})
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"redirect", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://unapproved.invalid/secret-SENSITIVE", 302)
		}},
		{"oversize", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(maxArchiveBytes+1))
			w.WriteHeader(200)
		}},
		{"interrupted", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(len(data)+10))
			w.Write(data[:5])
		}},
		{"corruption", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("changed")) }},
		{"http error", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server, roots := tlsFixture(t, tt.handler)
			s := fixtureSpec(data)
			s.source = server.URL
			dir := privateDir(t)
			r, e := Fetch(context.Background(), s, FetchOptions{dir, roots})
			if e == nil || r.Disposition == Verified || strings.Contains(e.Error(), "SENSITIVE") {
				t.Fatal("failure accepted/leaked", e)
			}
			files, _ := os.ReadDir(dir)
			if len(files) != 0 {
				t.Fatal("incomplete file retained")
			}
		})
	}
}
func TestSignatureStatus(t *testing.T) {
	good := "[GNUPG:] VALIDSIG " + asteriskFingerprint + " 2026-11-05 1793884505 0 4 0 1 10 00 " + asteriskFingerprint
	if !validSignatureStatus(good, asteriskFingerprint) {
		t.Fatal("valid signature status rejected")
	}
	for _, bad := range []string{strings.ReplaceAll(good, asteriskFingerprint, strings.Repeat("0", 40)), strings.Replace(good, " 10 00 ", " 2 00 ", 1), "[GNUPG:] GOODSIG 123 name"} {
		if validSignatureStatus(bad, asteriskFingerprint) {
			t.Fatal("bad signature status accepted")
		}
	}
	data := tgz(t, entry{"a", "b", 0})
	s := fixtureSpec(data)
	s.component = "asterisk"
	if _, e := VerifyAsteriskSignature(context.Background(), s, writeFixture(t, data), writeFixture(t, []byte("sig")), writeFixture(t, []byte("untrusted key")), privateDir(t)); e == nil {
		t.Fatal("untrusted key accepted")
	}
}

// Opt-in, read-only qualification of preserved real M0 artifacts, separate from
// deterministic synthetic tests. No network, extraction, installation or VMs.
func TestM0Artifacts(t *testing.T) {
	dir := os.Getenv("ALMADIAL_M0_ARTIFACTS_DIR")
	if dir == "" {
		t.Skip("real M0 artifacts not requested")
	}
	m, e := Load(bytes.NewReader(manifestFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	names := []struct{ component, file string }{{"asterisk", "asterisk-22.12.0.tar.gz"}, {"vicidial", "vicidial-r4037.zip"}, {"viciphone", "viciphone.tar.gz"}, {"pjproject", "pjproject-2.17.tar.bz2"}, {"net-telnet", "Net-Telnet-3.05.tar.gz"}}
	for _, n := range names {
		t.Run(n.component, func(t *testing.T) {
			s, e := m.Artifact(n.component)
			if e != nil {
				t.Fatal(e)
			}
			r, e := VerifyLocal(context.Background(), s, filepath.Join(dir, n.file))
			j, _ := json.Marshal(r)
			t.Log(string(j))
			if e != nil {
				t.Fatal(e)
			}
		})
	}
	if key := os.Getenv("ALMADIAL_ASTERISK_KEYRING"); key != "" {
		s, _ := m.Artifact("asterisk")
		r, e := VerifyAsteriskSignature(context.Background(), s, filepath.Join(dir, "asterisk-22.12.0.tar.gz"), os.Getenv("ALMADIAL_ASTERISK_SIGNATURE"), key, privateDir(t))
		j, _ := json.Marshal(r)
		t.Log(string(j))
		if e != nil {
			t.Fatal(e)
		}
		bad := writeFixture(t, []byte("corrupt signature"))
		if _, e = VerifyAsteriskSignature(context.Background(), s, filepath.Join(dir, "asterisk-22.12.0.tar.gz"), bad, key, privateDir(t)); e == nil {
			t.Fatal("corrupt detached signature accepted")
		}
	}
}
