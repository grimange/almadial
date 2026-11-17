package provision

import (
	"context"
	"github.com/grimange/almadial/internal/execution"
	"strings"
	"testing"
)

func releaseFixture() (Snapshot, PackageSelection) {
	s := fixture()
	s.Release = "10.3"
	s.Installed = nil
	s.KeyHashes = []string{almaKeySHA}
	tx := Transaction{BeforeSHA256: s.Digest(), Install: currentPackagePolicy().Closure, Remove: []Package{}}
	return s, PackageSelection{PackagePolicyDigest("10.3"), tx}
}
func TestReleasePackageSelection(t *testing.T) {
	s, p := releaseFixture()
	if err := p.validate(s); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*Snapshot, *PackageSelection)
	}{
		{"wrong minor", func(s *Snapshot, p *PackageSelection) { s.Release = "10.4" }},
		{"historical host", func(s *Snapshot, p *PackageSelection) { s.Release = "10.2" }},
		{"different policy", func(s *Snapshot, p *PackageSelection) { p.PolicySHA256 = PolicyDigest() }},
		{"wrong host", func(s *Snapshot, p *PackageSelection) { s.Host = strings.Repeat("b", 64) }},
		{"missing required RPM", func(s *Snapshot, p *PackageSelection) {
			for i, c := range p.Transaction.Install {
				if c.Name == "gcc" {
					p.Transaction.Install = append(p.Transaction.Install[:i], p.Transaction.Install[i+1:]...)
					break
				}
			}
		}},
		{"repository version drift", func(s *Snapshot, p *PackageSelection) { p.Transaction.Install[0].EVR = "0:99-1.el10_3" }},
		{"untrusted repository", func(s *Snapshot, p *PackageSelection) { p.Transaction.Install[0].Repository = "untrusted" }},
		{"changed payload", func(s *Snapshot, p *PackageSelection) { p.Transaction.Install[0].SHA256 = strings.Repeat("f", 64) }},
		{"incompatible dependency replacement", func(s *Snapshot, p *PackageSelection) {
			p.Transaction.Remove = []Package{{Name: "glibc", EVR: "0:1-1.el10", Arch: "x86_64", Repository: "baseos"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, p := releaseFixture()
			p.Transaction.Install = append([]Candidate(nil), p.Transaction.Install...)
			tc.change(&s, &p)
			if p.validate(s) == nil {
				t.Fatal("unsafe selection accepted")
			}
		})
	}
	if PackagePolicyDigest("10.2") != PolicyDigest() || PackagePolicyDigest("10.3") == PolicyDigest() || PackagePolicyDigest("10.4") != "" {
		t.Fatal("release identity")
	}
}

type unavailablePackageRunner struct{ calls int }

func (r *unavailablePackageRunner) Validate(execution.Command) error { return nil }
func (r *unavailablePackageRunner) Run(context.Context, execution.Command) (execution.Result, error) {
	r.calls++
	return execution.Result{Started: true, ExitCode: 1, Stderr: "repository unavailable"}, nil
}
func TestPreapprovalUnavailableRepository(t *testing.T) {
	s, _ := releaseFixture()
	runner := &unavailablePackageRunner{}
	if _, err := (Native{runner}).ResolveInstallPackages(context.Background(), s); err == nil || runner.calls != 1 {
		t.Fatal("unavailable repository substituted or proceeded")
	}
}
func TestFrozenHistoricalAdapter(t *testing.T) {
	if rawSHA([]byte(nativeAdapter)) != "c0d524d3e7ebad7fa1c1e6f9d08756612fcca3563ce8683f25f4d6076e4f2335" {
		t.Fatal("historical graph adapter bytes changed")
	}
	if nativeCommand("resolve", fixture(), nil).Args[2] != nativeAdapter {
		t.Fatal("legacy command changed")
	}
	s, _ := releaseFixture()
	if nativeCommand("resolve", s, nil).Args[2] != releaseNativeAdapter {
		t.Fatal("release adapter not selected")
	}
}
