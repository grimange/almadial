//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
)

func vicidialRegisteredFixture(ami string) vicidialRegistration {
	r := vicidialSeedRegistration()
	r.IP, r.Version, r.Conference = "127.0.0.1", "22.12.0", "CONFBRIDGE"
	r.User, r.Update, r.Listen, r.Send = "almadial-s05", "almadial-s05", "almadial-s05", "almadial-s05"
	r.PassKey, r.PassKeyValid = rawSHA([]byte("fixture-existing-key")), 1
	r.Secret, r.Webroot = rawSHA([]byte(ami)), "0"
	return r
}

func TestVICIdialRegistrationPreimages(t *testing.T) {
	ami := strings.Repeat("b", 20)
	seed, complete := vicidialSeedRegistration(), vicidialRegisteredFixture(ami)
	for _, tc := range []struct {
		name   string
		row    vicidialRegistration
		change func(*vicidialRegistration)
		want   string
	}{
		{"accepted seed", seed, nil, "absent"},
		{"complete", complete, nil, "complete"},
		{"blank key needs explicit initialization", complete, func(r *vicidialRegistration) { r.PassKey = rawSHA(nil); r.PassKeyValid = 0 }, "partial"},
		{"invalid nonblank key", complete, func(r *vicidialRegistration) { r.PassKeyValid = 0 }, "error"},
		{"generator owns flag", complete, func(r *vicidialRegistration) { r.Rebuild = "N" }, "complete"},
		{"interrupted after server update", complete, func(r *vicidialRegistration) { r.Webroot = "1" }, "partial"},
		{"extra server", seed, func(r *vicidialRegistration) { r.Servers = 2 }, "error"},
		{"extra settings", seed, func(r *vicidialRegistration) { r.Settings = 2 }, "error"},
		{"wrong server", seed, func(r *vicidialRegistration) { r.ID = "operator" }, "error"},
		{"foreign IP", seed, func(r *vicidialRegistration) { r.IP = "192.0.2.1" }, "error"},
		{"foreign AMI", complete, func(r *vicidialRegistration) { r.Secret = rawSHA([]byte("another credential")) }, "error"},
		{"foreign send identity", complete, func(r *vicidialRegistration) { r.Send = "admin" }, "error"},
		{"disabled generation", complete, func(r *vicidialRegistration) { r.Generate = "N" }, "error"},
		{"wrong version", complete, func(r *vicidialRegistration) { r.Version = "18.0" }, "error"},
		{"old conference engine", complete, func(r *vicidialRegistration) { r.Conference = "MEETME" }, "error"},
		{"unrecognized partial", seed, func(r *vicidialRegistration) { r.Webroot = "0" }, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.row
			if tc.change != nil {
				tc.change(&r)
			}
			state, err := r.state(ami)
			if tc.want == "error" {
				if err == nil {
					t.Fatal("foreign state accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "complete" {
				want, _ := complete.state(ami)
				if !shaPattern.MatchString(state) || state != want {
					t.Fatal("generation invalidated registration")
				}
			} else if state != tc.want {
				t.Fatal(state)
			}
		})
	}
}

type vicidialRegistrationRunner struct {
	t                                    *testing.T
	row                                  vicidialRegistration
	ami                                  string
	commands, writes                     int
	failed, partial, noChange, truncated bool
}

func (r *vicidialRegistrationRunner) RunInput(_ context.Context, c execution.Command, input []byte) (execution.Result, error) {
	r.t.Helper()
	r.commands++
	if c.Executable != "/usr/bin/mariadb" || strings.Contains(strings.Join(c.Args, " "), r.ami) {
		r.t.Fatal("unsafe SQL invocation")
	}
	result := execution.Result{Started: true, StdoutTruncated: r.truncated}
	if string(input) == vicidialRegistrationQuery {
		b, _ := json.Marshal(r.row)
		result.Stdout = string(b)
		return result, nil
	}
	want, err := guardedVICIdialRegistrationSQL(r.row, r.ami)
	if err != nil || string(input) != want {
		r.t.Fatal("unexpected mutation")
	}
	r.writes++
	if !r.noChange {
		r.row = vicidialRegisteredFixture(r.ami)
		if r.partial {
			r.row.Webroot = "1"
		}
	}
	if r.failed {
		result.ExitCode = 1
	}
	return result, nil
}

func TestVICIdialRegistrationGuardAndPostcondition(t *testing.T) {
	ami := strings.Repeat("b", 20)
	for _, tc := range []struct {
		name   string
		change func(*vicidialRegistrationRunner)
		valid  bool
	}{
		{"fresh", func(*vicidialRegistrationRunner) {}, true},
		{"partial completion", func(r *vicidialRegistrationRunner) { r.row = vicidialRegisteredFixture(ami); r.row.Webroot = "1" }, true},
		{"already complete", func(r *vicidialRegistrationRunner) { r.row = vicidialRegisteredFixture(ami) }, true},
		{"failed SQL", func(r *vicidialRegistrationRunner) { r.failed = true }, false},
		{"partial postimage", func(r *vicidialRegistrationRunner) { r.partial = true }, false},
		{"exit zero without updates", func(r *vicidialRegistrationRunner) { r.noChange = true }, false},
		{"truncated observation", func(r *vicidialRegistrationRunner) { r.truncated = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &vicidialRegistrationRunner{t: t, row: vicidialSeedRegistration(), ami: ami}
			tc.change(r)
			state, err := registerVICIdial(context.Background(), r, ami, hash(r.row))
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
			if tc.valid && !shaPattern.MatchString(state) {
				t.Fatal(state)
			}
			if tc.name == "already complete" && r.writes != 0 {
				t.Fatal("unnecessary mutation")
			}
		})
	}
	r := &vicidialRegistrationRunner{t: t, row: vicidialSeedRegistration(), ami: ami}
	if _, err := registerVICIdial(context.Background(), r, ami, hash("stale preimage")); err == nil || r.writes != 0 {
		t.Fatal("stale preimage allowed")
	}
	query, err := guardedVICIdialRegistrationSQL(r.row, ami)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"LOCK TABLES asterisk.servers WRITE, asterisk.system_settings WRITE;", "@almadial_servers=1", "@almadial_settings=1", "SHA2(ASTmgrSECRET,256)", "UNLOCK TABLES;", "RANDOM_BYTES(16)", "SHA2(COALESCE(pass_key,''),256)", "pass_key IS NULL OR pass_key=''"} {
		if !strings.Contains(query, v) {
			t.Fatal("missing guard", v)
		}
	}
	for _, v := range []string{"INSERT ", "DELETE ", "CREATE ", "ALTER ", "DROP ", "GRANT ", "phones", "TRUNCATE ", "db_schema_version"} {
		if strings.Contains(query, v) {
			t.Fatal("bootstrap or unrelated mutation", v)
		}
	}
}

func TestVICIdialPassKeyInitializationPreservesExistingKeys(t *testing.T) {
	ami := strings.Repeat("a", 20)
	for _, existing := range []bool{false, true} {
		before := vicidialSeedRegistration()
		if existing {
			before.PassKey = rawSHA([]byte("existing-private-key"))
			before.PassKeyValid = 1
		}
		q, err := guardedVICIdialRegistrationSQL(before, ami)
		if err != nil {
			t.Fatal(err)
		}
		for _, clause := range []string{"pass_key=IF(pass_key IS NULL OR pass_key='',LOWER(HEX(RANDOM_BYTES(16))),pass_key)", "WHERE SHA2(COALESCE(pass_key,''),256)='" + before.PassKey + "'", "@almadial_settings=1"} {
			if !strings.Contains(q, clause) {
				t.Fatal("missing key preservation/preimage guard")
			}
		}
	}
	before := vicidialRegisteredFixture(ami)
	after := before
	after.PassKey = rawSHA([]byte("foreign-key"))
	a, err := before.state(ami)
	if err != nil {
		t.Fatal(err)
	}
	b, err := after.state(ami)
	if err != nil || a == b {
		t.Fatal("key drift not bound by registration identity")
	}
}
