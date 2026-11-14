package artifacts

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/grimange/almadial/internal/releaseassets"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireInstallRequiresPinnedManifest(t *testing.T) {
	if _, err := AcquireInstall(context.Background(), Manifest{}, privateDir(t), &bytes.Buffer{}); err == nil {
		t.Fatal("uninitialized manifest accepted")
	}
}

func TestOriginPolicy(t *testing.T) {
	m, err := Load(bytes.NewReader(manifestFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	policy := func(digest string, stores []string) []byte {
		b, _ := json.Marshal(map[string]any{"schema": 1, "manifest_sha256": digest, "private_content_addressed_stores": stores})
		return b
	}
	configured, err := WithAcquisitionOrigins(m, bytes.NewReader(policy(m.Digest(), []string{"https://approved.invalid/sources"})))
	if err != nil || configured.Digest() != m.Digest() {
		t.Fatal("source identity changed", err)
	}
	for _, name := range installComponents {
		before, _ := m.Artifact(name)
		after, _ := configured.Artifact(name)
		if before.sha256 != after.sha256 || before.version != after.version || before.tree != after.tree || before.Source() != after.Source() || len(before.origins) != 0 || len(after.origins) != 1 || !strings.Contains(after.origins[0], "/sha256/"+before.sha256+"/"+name+".artifact") {
			t.Fatal("origin and source conflated", name)
		}
	}
	for _, b := range [][]byte{
		policy(strings.Repeat("0", 64), []string{"https://approved.invalid/sources"}),
		policy(m.Digest(), []string{"http://approved.invalid/sources"}),
		policy(m.Digest(), []string{"https://user:secret@approved.invalid/sources"}),
		policy(m.Digest(), []string{"https://approved.invalid/sources?token=secret"}),
		policy(m.Digest(), []string{"https://approved.invalid/a/../sources"}),
		policy(m.Digest(), []string{"https://approved.invalid/%2e%2e"}),
		policy(m.Digest(), nil),
		append(policy(m.Digest(), []string{"https://approved.invalid/sources"}), []byte(" {}")...),
	} {
		if _, e := WithAcquisitionOrigins(m, bytes.NewReader(b)); e == nil {
			t.Fatal("invalid policy accepted")
		}
	}
}

func TestDurableAcquisition(t *testing.T) {
	good := tgz(t, entry{"r4037/file", "accepted source", 0})
	wrong := tgz(t, entry{"r4038/file", "wrong revision", 0})
	for _, scenario := range []string{"nightly404", "corrupt-cache", "wrong-revision", "offline-cache", "origins-exhausted", "untrusted-store"} {
		t.Run(scenario, func(t *testing.T) {
			upstreamHits, storeHits := 0, 0
			server, roots := tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/nightly" {
					upstreamHits++
					http.NotFound(w, r)
					return
				}
				storeHits++
				if scenario == "origins-exhausted" {
					http.Error(w, "unavailable", 503)
					return
				}
				if scenario == "wrong-revision" {
					w.Write(wrong)
					return
				}
				w.Write(good)
			}))
			spec := fixtureSpec(good)
			spec.source = server.URL + "/nightly"
			spec.origins = []string{server.URL + "/store"}
			dir := privateDir(t)
			final := filepath.Join(dir, "fixture.artifact")
			if scenario == "corrupt-cache" || scenario == "offline-cache" {
				data := good
				if scenario == "corrupt-cache" {
					data = []byte("corrupt")
				}
				if e := os.WriteFile(final, data, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if scenario == "offline-cache" {
				server.Close()
			}
			if scenario == "untrusted-store" {
				roots = nil
			}
			result, err := Fetch(context.Background(), spec, FetchOptions{dir, roots})
			fails := scenario == "wrong-revision" || scenario == "origins-exhausted" || scenario == "untrusted-store"
			if fails {
				if err == nil || result.Identity == Verified || !strings.Contains(err.Error(), "all approved acquisition origins exhausted") {
					t.Fatal(result, err)
				}
				if _, e := os.Stat(final); !os.IsNotExist(e) {
					t.Fatal("unverified cache published", e)
				}
				if scenario == "wrong-revision" && !strings.Contains(err.Error(), "SHA-256 mismatch") {
					t.Fatal(err)
				}
			} else {
				if err != nil || result.Identity != Verified || result.ActualSHA256 != spec.sha256 {
					t.Fatal(result, err)
				}
				if scenario == "offline-cache" && (upstreamHits != 0 || storeHits != 0 || result.Transport != NotVerified) {
					t.Fatal("offline cache used network")
				}
				if scenario != "offline-cache" && (storeHits != 1 || upstreamHits != 0) {
					t.Fatal("durable origin priority", storeHits, upstreamHits)
				}
			}
			if scenario == "corrupt-cache" {
				files, e := filepath.Glob(final + ".rejected-*")
				if e != nil || len(files) != 1 {
					t.Fatal("rejected bytes not preserved", e)
				}
				b, e := os.ReadFile(files[0])
				if e != nil || string(b) != "corrupt" {
					t.Fatal(e)
				}
			}
		})
	}
	// Explicit primary 404 -> approved second origin, unchanged content identity.
	hits := []string{}
	server, roots := tlsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		if r.URL.Path == "/nightly" {
			http.NotFound(w, r)
		} else {
			w.Write(good)
		}
	}))
	s := fixtureSpec(good)
	s.source = server.URL + "/nightly"
	s.origins = []string{server.URL + "/nightly", server.URL + "/retained"}
	r, e := Fetch(context.Background(), s, FetchOptions{privateDir(t), roots})
	if e != nil || r.ActualSHA256 != s.sha256 || strings.Join(hits, ",") != "/nightly,/retained" {
		t.Fatal("404 fallback", r, e, hits)
	}
}

func TestConfiguredStorePreservesIdentity(t *testing.T) {
	m, err := Load(bytes.NewReader(releaseassets.Manifest))
	if err != nil {
		t.Fatal(err)
	}
	for _, store := range []string{"", "https://first.example.test/sources", "https://second.example.test/sources"} {
		policy, err := AcquisitionPolicy(m, store)
		if err != nil {
			t.Fatal(err)
		}
		configured, err := WithAcquisitionOrigins(m, bytes.NewReader(policy))
		if err != nil || configured.Digest() != m.Digest() {
			t.Fatal("location changed manifest", err)
		}
		for _, name := range installComponents {
			original, _ := m.artifact(name)
			candidate, _ := configured.artifact(name)
			if original.sha256 != candidate.sha256 || original.version != candidate.version || original.tree != candidate.tree {
				t.Fatal("identity changed", name)
			}
			if store == "" {
				if len(configured.origins[name]) != 0 {
					t.Fatal("upstream-only policy injected a mirror", name)
				}
			} else if len(configured.origins[name]) != 1 || !strings.Contains(configured.origins[name][0], original.sha256) {
				t.Fatal("mirror does not retain content identity", name)
			}
		}
	}
	for _, bad := range []string{"http://example.test/sources", "https://user:secret@example.test/sources"} {
		if _, err := AcquisitionPolicy(m, bad); err == nil {
			t.Fatal("unsafe store accepted")
		}
	}
}
