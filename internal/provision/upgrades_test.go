package provision

import (
	"encoding/json"
	"testing"
)

func TestReviewedCoupledUpdates(t *testing.T) {
	s := fixture("autoconf")
	tx := transaction(s)
	for _, u := range Upgrades() {
		s.Installed = append(s.Installed, Installed{u.Before.Name, u.Before.EVR, u.Before.Arch, "dee5c11cc2a1e572"})
		tx.Remove = append(tx.Remove, u.Before)
		tx.Install = append(tx.Install, u.After)
	}
	tx.BeforeSHA256 = s.Digest()
	if err := tx.Validate(s); err != nil {
		t.Fatal(err)
	}
	if err := Verify(s, applied(s, tx), tx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*Transaction)
	}{
		{"unpaired removal", func(v *Transaction) { v.Install = v.Install[:1] }},
		{"unpaired replacement", func(v *Transaction) { v.Remove = v.Remove[1:] }},
		{"duplicate removal", func(v *Transaction) { v.Remove = append(v.Remove, v.Remove[0]) }},
		{"different old version", func(v *Transaction) { v.Remove[0].EVR = "0:2.40.2-17.el10" }},
		{"different new version", func(v *Transaction) { v.Install[1].EVR = "0:2.40.2-19.el10" }},
		{"downgrade", func(v *Transaction) { v.Install[1].EVR = "0:2.40.2-17.el10" }},
		{"different architecture", func(v *Transaction) { v.Install[1].Arch = "noarch" }},
		{"different source", func(v *Transaction) { v.Install[1].Repository = "appstream" }},
		{"different payload", func(v *Transaction) { v.Install[1].SHA256 = hash("other payload") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tx)
			var v Transaction
			_ = json.Unmarshal(b, &v)
			tc.edit(&v)
			if v.Validate(s) == nil {
				t.Fatal("accepted an unreviewed update")
			}
		})
	}
	bad := applied(s, tx)
	bad.Installed = append(bad.Installed, s.Installed[len(s.Installed)-1])
	if Verify(s, bad, tx) == nil {
		t.Fatal("retained old version accepted")
	}
	noCommit := applied(s, tx)
	noCommit.HistorySHA256 = s.HistorySHA256
	if Verify(s, noCommit, tx) == nil {
		t.Fatal("RPM writes without a committed DNF history record accepted")
	}
}
