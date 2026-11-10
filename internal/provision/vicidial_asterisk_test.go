package provision

import (
	"strings"
	"testing"
)

func TestVICIdialDatabaseRestartNormalization(t *testing.T) {
	i, _, _ := databaseInputs(t, true)
	accepted := i.Snapshot
	accepted.ServiceID = strings.Repeat("a", 32)
	fresh := accepted
	fresh.ServiceID = strings.Repeat("b", 32)
	got, e := normalizeVICIdialDatabase(fresh, accepted)
	if e != nil || got.Digest() != accepted.Digest() {
		t.Fatal("healthy restart not recognized", e)
	}
	for _, change := range []func(*DatabaseSnapshot){
		func(s *DatabaseSnapshot) { s.CatalogSHA = strings.Repeat("0", 64) },
		func(s *DatabaseSnapshot) { s.CredentialID = strings.Repeat("0", 64) },
		func(s *DatabaseSnapshot) { s.ApplicationAuthOK = false },
		func(s *DatabaseSnapshot) { s.ServiceID = "" },
		func(s *DatabaseSnapshot) { s.GrantsOK = false },
		func(s *DatabaseSnapshot) { s.Foundation = strings.Repeat("0", 64) },
		func(s *DatabaseSnapshot) { s.Version = "1749" },
		func(s *DatabaseSnapshot) { s.Active = false },
	} {
		f := fresh
		change(&f)
		if _, e := normalizeVICIdialDatabase(f, accepted); e == nil {
			t.Fatal("unrelated drift accepted")
		}
	}
}
