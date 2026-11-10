//go:build linux

package provision

import (
	"context"
	"encoding/json"

	"github.com/grimange/almadial/internal/execution"
)

func vicidialAsteriskInspection(n AsteriskNative, toolchain, service string, wrapped bool) (execution.Command, error) {
	c := n.fileCommand("inspect", toolchain, service)
	if len(c.Args) != 5 {
		return execution.Command{}, errVICIdial
	}
	var p map[string]any
	if json.Unmarshal([]byte(c.Args[4]), &p) != nil {
		return execution.Command{}, errVICIdial
	}
	if wrapped {
		p["s05_handoff"] = true
	}
	c.Args[4] = jsonString(p)
	return c, nil
}

// The S04 parent identity remains immutable. This inspector freshly checks the
// same executable, libraries, forty modules, fragments, unit and live process.
// Only the five closed S05 include transitions and one protected local AMI
// account are substituted for S04's initial configuration/account expectations.
// Generation/publication is verified separately, never inferred from this hash.
func inspectVICIdialAsterisk(ctx context.Context, n AsteriskNative, accepted AsteriskSnapshot, databaseAccepted DatabaseSnapshot, wrapped bool, ami string) (string, error) {
	if ctx == nil || !asteriskSatisfied(accepted) || n.Source.Digest() != accepted.Source || (wrapped && !vicidialHex(ami, 20)) {
		return "", errVICIdial
	}
	db, e := inspectVICIdialDatabase(ctx, n.Database, databaseAccepted)
	if e != nil || !databaseSatisfied(db) {
		return "", errVICIdial
	}
	s := AsteriskSnapshot{Host: db.Host, Foundation: db.Foundation, Database: db.Digest(), Source: n.Source.Digest(), EnvironmentOK: true}
	s.Toolchain, e = n.toolchain(ctx)
	if e != nil {
		return "", errVICIdial
	}
	svc, e := n.service(ctx)
	if e != nil {
		return "", errVICIdial
	}
	s.Active, s.Enabled = svc["ActiveState"] == "active", svc["UnitFileState"] == "enabled"
	if s.Active {
		s.ServiceID = svc["InvocationID"]
	}
	// The accepted S04 restart receipt proves its recorded qualification, not
	// that systemd will reuse an InvocationID after the stopped-VM snapshot.
	// Validate that historical receipt against the immutable accepted invocation;
	// independently verify the current unit, PID, identity and modules below.
	c, e := vicidialAsteriskInspection(n, s.Toolchain, accepted.ServiceID, wrapped)
	if e != nil {
		return "", errVICIdial
	}
	r, e := n.run(ctx, c)
	if e != nil {
		return "", errVICIdial
	}
	var f struct {
		Build, Install, Configuration, Unit string
		RestartVerified                     bool
		Modules                             []AsteriskModule
	}
	if decode(r.Stdout, &f) != nil || !shaPattern.MatchString(f.Configuration) {
		return "", errVICIdial
	}
	s.Build, s.Install, s.Unit, s.RestartVerified, s.Modules = f.Build, f.Install, f.Unit, f.RestartVerified, f.Modules
	if !wrapped && f.Configuration != accepted.Configuration {
		return "", errVICIdial
	}
	// The fixed filesystem adapter already checked every wrapper against the
	// selected exact variant and every S04 fragment against its original bytes.
	// Normalize that declared ownership transition for parent identity matching.
	s.Configuration = accepted.Configuration
	var management func(context.Context) error
	if wrapped {
		management = func(ctx context.Context) error {
			users, e := n.cli(ctx, "manager show users")
			if e != nil || verifyVICIdialManagerUsers(users) != nil {
				return errVICIdial
			}
			return verifyVICIdialAMI(ctx, ami)
		}
	}
	s.RuntimeOK = n.runtimeWithManagement(ctx, svc, &s, management) == nil
	if !s.RuntimeOK || !s.RestartVerified {
		return "", errVICIdial
	}
	// Invocation changes are expected after a legitimate service/VM restart.
	// They may not mask any other structural or runtime parent difference.
	s.ServiceID = accepted.ServiceID
	if !asteriskSatisfied(s) || s.Digest() != accepted.Digest() {
		return "", errVICIdial
	}
	return accepted.Digest(), nil
}

// A snapshot stop/start changes MariaDB's invocation while preserving its
// independently reverified catalog, credentials, grants and local runtime.
// Normalize only that ephemeral field to the accepted parent's identity.
func inspectVICIdialDatabase(ctx context.Context, n DatabaseNative, accepted DatabaseSnapshot) (DatabaseSnapshot, error) {
	fresh, e := n.InspectDatabase(ctx)
	if e != nil || VerifyDatabaseForApplication(fresh, n.Source) != nil {
		return DatabaseSnapshot{}, errVICIdial
	}
	return normalizeVICIdialDatabase(fresh, accepted)
}
func normalizeVICIdialDatabase(fresh, accepted DatabaseSnapshot) (DatabaseSnapshot, error) {
	if !databaseSatisfied(fresh) || !databaseSatisfied(accepted) || !shaInvocation(fresh.ServiceID) || !shaInvocation(accepted.ServiceID) {
		return DatabaseSnapshot{}, errVICIdial
	}
	fresh.ServiceID = accepted.ServiceID
	if fresh.Digest() != accepted.Digest() {
		return DatabaseSnapshot{}, errVICIdial
	}
	return fresh, nil
}
