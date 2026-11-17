package provision

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed native_legacy.py
var nativeAdapter string

// The historical adapter bytes remain frozen because command graphs bind them.
//
//go:embed native.py
var releaseNativeAdapter string

// Native's runner is ALM-13's bounded direct-argv runner. It is injectable only
// at this internal package boundary for deterministic tests, never via user argv.
type Native struct{ Runner execution.CommandRunner }

func NewNative(sensitive []string) (Native, error) {
	r, e := execution.NewRunner(sensitive)
	return Native{rpmInputRunner{r}}, e
}

// Keep the approved command/graph identity while transporting large RPM
// requests over the existing bounded stdin channel. Linux limits each argv
// element to 128 KiB, even when the total ARG_MAX budget is larger.
type rpmInputRunner struct{ *execution.Runner }

func (r rpmInputRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if c.Executable == "/usr/bin/python3" && len(c.Args) == 5 && c.Args[0] == "-B" && c.Args[1] == "-c" && (c.Args[2] == nativeAdapter || c.Args[2] == releaseNativeAdapter) && len(c.Args[4]) >= 64<<10 {
		payload := []byte(c.Args[4])
		program := c.Args[2]
		c.Args = append([]string(nil), c.Args[:4]...)
		c.Args[2] = "import sys\nsys.argv.append(sys.stdin.read())\n" + program
		return r.RunInput(ctx, c, payload)
	}
	return r.Runner.Run(ctx, c)
}
func command(mode string, payload any) execution.Command {
	args := []string{"-B", "-c", nativeAdapter, mode}
	if payload != nil {
		b, _ := json.Marshal(payload)
		args = append(args, string(b))
	}
	return execution.Command{Executable: "/usr/bin/python3", Args: args, Timeout: 20 * time.Minute, StdoutLimit: 4 << 20, StderrLimit: 64 << 10}
}
func decode(raw string, v any) error {
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errPolicy
	}
	if _, e := d.Token(); e != io.EOF {
		return errPolicy
	}
	return nil
}
func (n Native) Inspect(ctx context.Context) (Snapshot, error) {
	var s Snapshot
	if ctx == nil || n.Runner == nil {
		return s, errPolicy
	}
	c := command("inspect", nil)
	c.Timeout = 30 * time.Second
	r, e := n.Runner.Run(ctx, c)
	if e != nil {
		return s, e
	}
	if !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return s, errPolicy
	}
	if decode(r.Stdout, &s) != nil {
		return Snapshot{}, errPolicy
	}
	return canonical(s), nil
}

type request struct {
	Mode         string                `json:"mode"`
	Packages     []Package             `json:"packages"`
	Before       Snapshot              `json:"before"`
	BeforeSHA256 string                `json:"before_sha256"`
	Transaction  *Transaction          `json:"transaction,omitempty"`
	PublicKey    string                `json:"public_key,omitempty"`
	Policy       *ReleasePackagePolicy `json:"policy,omitempty"`
}

func nativeCommand(mode string, s Snapshot, t *Transaction) execution.Command {
	packages := packagesFor(s.Release)
	// A reviewed official-repository addition (for example S09 firewalld) is
	// already permitted by Transaction.Validate. Re-solve its exact direct
	// requests as well; otherwise apply silently resolves only the M0 set and
	// rejects the separately reviewed transaction as stale.
	if mode == "apply" && t != nil {
		for _, candidate := range t.Install {
			found := false
			for _, p := range packages {
				if p == candidate.Package {
					found = true
				}
			}
			if !found {
				packages = append(packages, candidate.Package)
			}
		}
	}
	r := request{Mode: mode, Packages: packages, Before: canonical(s), BeforeSHA256: s.Digest(), Transaction: t}
	if mode == "trust" {
		r.PublicKey = epelPublicKey
	}
	c := command(mode, r)
	if s.Release == "10.3" {
		r.Policy = currentPackagePolicy()
		r.PublicKey = epelPublicKey
		c = command(mode, r)
		c.Args[2] = releaseNativeAdapter
	}
	return c
}

// ResolveInstallPackages performs only DNF cache/download/signature operations.
// It never imports a system key, installs an RPM or starts a service.
func (n Native) ResolveInstallPackages(ctx context.Context, s Snapshot) (PackageSelection, error) {
	if s.Release != "10.3" || s.validateTrust(false) != nil {
		return PackageSelection{}, errPolicy
	}
	result, err := n.Runner.Run(ctx, nativeCommand("resolve", s, nil))
	if err != nil {
		return PackageSelection{}, err
	}
	var t Transaction
	if !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated || decode(result.Stdout, &t) != nil {
		return PackageSelection{}, errors.New("RPM pre-approval resolution failed: verify AlmaLinux 10.3 official repositories and outbound HTTPS; exact packages, dependencies, checksums and trusted signatures must resolve before provisioning; no versions were substituted")
	}
	selection := PackageSelection{PackagePolicyDigest(s.Release), t}
	if err := selection.validate(s); err != nil {
		return PackageSelection{}, err
	}
	fresh, err := n.Inspect(ctx)
	if err != nil || fresh.Digest() != s.Digest() {
		return PackageSelection{}, errPolicy
	}
	return selection, nil
}
