package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/grimange/almadial/internal/journal"
)

func TestExplicitKeyPreparation(t *testing.T) {
	h := sha256.Sum256([]byte(epelPublicKey))
	if hex.EncodeToString(h[:]) != epelKeySHA {
		t.Fatal("embedded public key pin")
	}
	p, m := inputs(t)
	before := fixture("autoconf")
	before.KeyHashes = []string{almaKeySHA}
	if before.ValidateTrust() == nil {
		t.Fatal("normal package stage accepted missing key")
	}
	if _, e := PrepareResolution(p, m, before); e == nil {
		t.Fatal("implicit import accepted")
	}
	st, e := PrepareRepositoryTrust(p, m, before)
	if e != nil {
		t.Fatal(e)
	}
	after := canonical(before)
	after.KeyHashes = append(after.KeyHashes, epelKeySHA)
	after.Installed = append(after.Installed, epelKeyRPM)
	f := &fakeNative{s: before, after: after}
	dir := filepath.Join(t.TempDir(), "state")
	f.onRun = func() {
		d, e := journal.Inspect(dir)
		if e != nil || d.Actions[0].State != journal.Running {
			t.Fatal("key import before durable intent", e)
		}
	}
	out, e := ExecuteLab(context.Background(), st, m, auth(st), dir, Native{f}, false, nil)
	if e != nil || out.Journal.Actions[0].State != journal.Succeeded || f.mutations != 1 {
		t.Fatal(out, e)
	}
	if _, e = ExecuteLab(context.Background(), st, m, auth(st), dir, Native{f}, true, nil); e != nil || f.mutations != 1 {
		t.Fatal("completed import replayed", e)
	}
	noop, e := PrepareRepositoryTrust(p, m, after)
	if e != nil {
		t.Fatal(e)
	}
	out, e = ExecuteLab(context.Background(), noop, m, auth(noop), "/must-not-be-created", Native{f}, false, nil)
	if e != nil || !out.NoOp || f.mutations != 1 {
		t.Fatal(out, e)
	}
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"missing key RPM", func(s *Snapshot) { s.Installed = s.Installed[:len(s.Installed)-1] }},
		{"wrong key", func(s *Snapshot) { s.KeyHashes = []string{almaKeySHA, hash("wrong")} }},
		{"unexpected package", func(s *Snapshot) { s.Installed = append(s.Installed, Installed{"extra", "0:1-1.el10", "x86_64", ""}) }},
		{"history changed", func(s *Snapshot) { s.HistorySHA256 = hash("unrelated transaction") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := canonical(after)
			tc.edit(&s)
			if verifyKeyImport(before, s) == nil {
				t.Fatal("incorrect key import accepted")
			}
		})
	}
	collision := canonical(before)
	collision.Installed = append(collision.Installed, epelKeyRPM)
	if _, e := PrepareRepositoryTrust(p, m, collision); e == nil {
		t.Fatal("key identity collision accepted")
	}
}
