package provision

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/grimange/almadial/internal/journal"
)

func TestInterruptedProvisioning(t *testing.T) {
	p, m := inputs(t)
	before := fixture("autoconf", "gcc")
	tx := transaction(before)
	for _, tc := range []struct {
		name    string
		resolve bool
		actual  Snapshot
		want    journal.State
	}{
		{"completed RPM transaction independently reconciles", false, applied(before, tx), journal.Succeeded},
		{"nothing installed never automatically replays", false, before, journal.ReconciliationRequired},
		{"partial RPM transaction never automatically replays", false, applied(before, Transaction{Install: tx.Install[:1]}), journal.ReconciliationRequired},
		{"lost solver result requires a new reviewed operation", true, before, journal.ReconciliationRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, e := PreparePackages(p, m, before, tx)
			if tc.resolve {
				st, e = PrepareResolution(p, m, before)
			}
			if e != nil {
				t.Fatal(e)
			}
			a := auth(st)
			dir := filepath.Join(t.TempDir(), "state")
			doc, e := st.journalDocument(a)
			if e != nil {
				t.Fatal(e)
			}
			s, e := journal.Acquire(dir)
			if e != nil {
				t.Fatal(e)
			}
			if e = s.Initialize(doc); e != nil {
				t.Fatal(e)
			}
			if e = s.Begin(st.Graph().Actions()[0].ID); e != nil {
				t.Fatal(e)
			}
			if e = s.Close(); e != nil {
				t.Fatal(e)
			}
			// Simulate a dead owner after durable intent, with independent actual
			// RPM observations. The new process must never replay the transaction.
			f := &fakeNative{s: tc.actual, tx: tx}
			out, e := ExecuteLab(context.Background(), st, m, a, dir, Native{f}, true, nil)
			if tc.want == journal.Succeeded && e != nil {
				t.Fatal(e)
			}
			if tc.want == journal.ReconciliationRequired && !errors.Is(e, journal.ErrReconcile) {
				t.Fatal(e)
			}
			if out.Journal == nil || out.Journal.Actions[0].State != tc.want || f.mutations != 0 {
				t.Fatal(out, f.mutations)
			}
			// Repeating reconciliation cannot erase uncertainty or replay it.
			_, e = ExecuteLab(context.Background(), st, m, a, dir, Native{f}, true, nil)
			if tc.want == journal.ReconciliationRequired && !errors.Is(e, journal.ErrReconcile) {
				t.Fatal(e)
			}
			if f.mutations != 0 {
				t.Fatal("unsafe replay")
			}
		})
	}
}

func TestResumeIdentityAndUnknownHistory(t *testing.T) {
	p, m := inputs(t)
	s := fixture("autoconf")
	tx := transaction(s)
	st, _ := PreparePackages(p, m, s, tx)
	a := auth(st)
	dir := filepath.Join(t.TempDir(), "state")
	f := &fakeNative{s: s, tx: tx, after: applied(s, tx)}
	if _, e := ExecuteLab(context.Background(), st, m, a, dir, Native{f}, false, nil); e != nil {
		t.Fatal(e)
	}
	// A different valid permission reference still changes the journal identity.
	a.OperatorEvidenceSHA256 = hash("different permission")
	if _, e := ExecuteLab(context.Background(), st, m, a, dir, Native{f}, true, nil); !errors.Is(e, journal.ErrIdentity) {
		t.Fatal(e)
	}
	f.s.HistoryComplete = false
	out, e := ExecuteLab(context.Background(), st, m, auth(st), dir, Native{f}, true, nil)
	if !errors.Is(e, journal.ErrReconcile) || out.Journal == nil || out.Journal.Actions[0].State != journal.ReconciliationRequired {
		t.Fatal("unknown DNF history was not durably reconciled", out, e)
	}
	if f.mutations != 1 {
		t.Fatal("resume mutated RPM state")
	}
}
