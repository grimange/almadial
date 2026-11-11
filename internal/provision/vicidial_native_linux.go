//go:build linux

package provision

import (
	"context"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

// VICIdialNative consumes the accepted S04 parent. It never invokes S03's
// mutating executor; all database foundation checks are read-only.
type VICIdialNative struct {
	Asterisk         AsteriskNative
	Accepted         AsteriskSnapshot
	DatabaseAccepted DatabaseSnapshot
	Source           VICIdialSource
	Dependency       VICIdialPerlSource
	PhoneHandoff     bool // S10 alone verifies its separately owned synthetic row.
	WebHandoff       bool // S07 owns first_login_trigger; all other S05 inputs remain exact.
}

func (n VICIdialNative) input() (vicidialInputRunner, error) {
	r, ok := n.Asterisk.Database.Native.Runner.(vicidialInputRunner)
	if !ok || r == nil {
		return nil, errVICIdial
	}
	return r, nil
}
func (n VICIdialNative) binding() string {
	return hash(struct{ Parent, Source, Dependency, Policy string }{n.Accepted.Digest(), n.Source.Digest(), n.Dependency.Digest(), VICIdialPolicyDigest()})
}

type VICIdialCheckError struct{ Gate string }

func (e *VICIdialCheckError) Error() string { return "S05 native verification failed: " + e.Gate }

func (n VICIdialNative) InspectVICIdial(ctx context.Context) (s VICIdialSnapshot, err error) {
	return n.inspectVICIdial(ctx, false)
}

// S06 validates its closed process/schedule inventory before using this
// internal handoff. The public S05 inspector retains its inactive-job gate.
func (n VICIdialNative) inspectVICIdial(ctx context.Context, runtimeHandoff bool) (s VICIdialSnapshot, err error) {
	gate := "verified inputs"
	defer func() {
		if err != nil {
			err = &VICIdialCheckError{Gate: gate}
		}
	}()
	runner, e := n.input()
	if e != nil || ctx == nil || n.Source.Digest() == "" || n.Dependency.Digest() == "" || !asteriskSatisfied(n.Accepted) {
		return s, errVICIdial
	}
	gate = "database foundation"
	db, e := inspectVICIdialDatabase(ctx, n.Asterisk.Database, n.DatabaseAccepted)
	if e != nil || VerifyDatabaseForApplication(db, n.Asterisk.Database.Source) != nil {
		return s, errVICIdial
	}
	s.Host, s.Foundation, s.Database = db.Host, db.Foundation, db.Digest()
	if s.Host != n.Accepted.Host || s.Foundation != n.Accepted.Foundation || s.Database != n.Accepted.Database {
		return s, errVICIdial
	}
	s.Source, s.Dependency = n.Source.Digest(), n.Dependency.Digest()
	gate = "generator inputs"
	s.GeneratorInputs, _, e = inspectVICIdialGeneratorInputsPhone(ctx, runner, n.WebHandoff, n.PhoneHandoff)
	if e != nil {
		return s, e
	}
	gate = "application files"
	assessment, e := readVICIdialFiles(ctx, n.Source, runner)
	if e != nil {
		return s, e
	}
	switch assessment.State {
	case "ABSENT":
		s.Files = "absent"
	case "PARTIAL":
		s.Files = "partial"
	case "COMPLETE":
		files, _ := n.Source.Deployment()
		s.Files = hash(files)
	default:
		return s, errVICIdial
	}
	gate = "Perl files"
	perl, e := n.Dependency.deployment()
	if e != nil {
		return s, e
	}
	s.Perl, e = vicidialOwned(ctx, runner, "perl", perl, false)
	if e != nil {
		return s, e
	}
	gate = "runtime prerequisites"
	prerequisites, e := inspectVICIdialPrerequisites(ctx, n.Asterisk.Database.Native.Runner)
	if e != nil || shaPattern.MatchString(s.Perl) && prerequisites.NetTelnetAbsent {
		return s, errVICIdial
	}
	gate = "deferred jobs"
	if !runtimeHandoff && verifyVICIdialDeferred(ctx, runner) != nil {
		return s, errVICIdial
	}
	gate = "protected configuration"
	secrets, e := databaseSecrets()
	if e != nil || len(secrets) != 2 {
		return s, errVICIdial
	}
	ami, e := vicidialAMISecret()
	if e != nil {
		return s, e
	}
	// Public placeholders are used only to establish that every config file is
	// absent. Any populated file without its original protected secret conflicts.
	key := ami
	if key == "" {
		key = strings.Repeat("0", 20)
	}
	files, e := vicidialConfiguration(secrets[0], key)
	if e != nil {
		return s, e
	}
	s.Configuration, e = vicidialOwned(ctx, runner, "configuration", files, false)
	if e != nil || ami == "" && s.Configuration != "absent" {
		return s, errVICIdial
	}
	gate = "application registration"
	_, s.Registration, e = inspectVICIdialRegistration(ctx, runner, key)
	if e != nil {
		return s, e
	}
	gate = "generated configuration"
	var wrapped bool
	s.Generated, wrapped, e = inspectVICIdialGenerationPhone(ctx, runner, n.binding(), s.GeneratorInputs, n.PhoneHandoff)
	if e != nil {
		return s, e
	}
	gate = "Asterisk handoff"
	s.Asterisk, e = inspectVICIdialAsterisk(ctx, n.Asterisk, n.Accepted, n.DatabaseAccepted, wrapped, ami)
	if e != nil {
		return s, e
	}
	gate = "bridge slots"
	if inspectVICIdialBridges(ctx, runner, !wrapped) != nil {
		return s, errVICIdial
	}
	gate = "Apache PHP integration"
	s.Web, e = n.inspectWeb(ctx, runner)
	if e != nil {
		return s, e
	}
	gate = "runtime receipt"
	s.Runtime = "absent"
	if vicidialComponentsComplete(s) {
		s.Runtime, e = n.runtimeReceipt(ctx, runner, s, false)
		if e != nil {
			return s, e
		}
	}
	if !vicidialReady(s) {
		return s, errVICIdial
	}
	return s, nil
}
func vicidialComponentsComplete(s VICIdialSnapshot) bool {
	for _, state := range []string{s.Files, s.Perl, s.Configuration, s.Registration, s.Generated, s.Web} {
		if !shaPattern.MatchString(state) {
			return false
		}
	}
	return true
}
func (n VICIdialNative) runtimeReceipt(ctx context.Context, r vicidialInputRunner, s VICIdialSnapshot, mutate bool) (string, error) {
	if !vicidialComponentsComplete(s) {
		return "", errVICIdial
	}
	receipt := struct {
		Binding    string
		Components map[string]string
	}{n.binding(), map[string]string{"Files": s.Files, "Perl": s.Perl, "Configuration": s.Configuration, "Registration": s.Registration, "Generated": s.Generated, "Web": s.Web, "Inputs": s.GeneratorInputs}}
	mode := "runtime-inspect"
	if mutate {
		mode = "runtime-seal"
	}
	var result struct{ State string }
	if vicidialAdapter(ctx, r, mode, struct{ Receipt any }{receipt}, &result) != nil || result.State != "absent" && !shaPattern.MatchString(result.State) || mutate && !shaPattern.MatchString(result.State) {
		return "", errVICIdial
	}
	return result.State, nil
}
func (n VICIdialNative) inspectWeb(ctx context.Context, runner vicidialInputRunner) (string, error) {
	state, e := vicidialOwned(ctx, runner, "web", vicidialWebConfiguration(), false)
	if e != nil {
		return "", e
	}
	if !shaPattern.MatchString(state) {
		// An already active web server without the complete owned selection must
		// never be silently adopted or restarted by application-file provisioning.
		r, e := n.Asterisk.run(ctx, dbCommand("/usr/bin/systemctl", "show", "httpd.service", "--property=ActiveState"))
		if e != nil || strings.TrimSpace(r.Stdout) != "ActiveState=inactive" {
			return "", errVICIdial
		}
		return state, nil
	}
	service, e := inspectVICIdialWebServiceTLS(ctx, n.Asterisk.Database.Native.Runner, n.Asterisk.TLSHandoff)
	if e != nil {
		return "partial", nil
	}
	if inspectVICIdialHTTP(ctx, vicidialWebClient()) != nil {
		return "", errVICIdial
	}
	return hash(struct{ Configuration, Service string }{state, service}), nil
}

// The dispatcher accepts only commands from this immutable graph. It is used
// inside ALM-13/14 after exclusive ownership and durable action intent.
type vicidialNativeRunner struct {
	native VICIdialNative
	stage  VICIdialStage
}

func (n VICIdialNative) Runner(stage VICIdialStage) execution.CommandRunner {
	return vicidialNativeRunner{n, stage}
}
func (r vicidialNativeRunner) Validate(c execution.Command) error {
	if r.stage.doc().Source != r.native.Source.Digest() || r.stage.doc().Dependency != r.native.Dependency.Digest() || r.stage.doc().Policy != VICIdialPolicyDigest() {
		return errVICIdial
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(a.Command) == jsonString(c) {
			return nil
		}
	}
	return errVICIdial
}
func (r vicidialNativeRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 {
		return execution.Result{}, errVICIdial
	}
	runner, e := r.native.input()
	if e != nil {
		return execution.Result{}, e
	}
	n := r.native
	switch c.Args[1] {
	case "vicidial.00.files":
		_, e = installVICIdialFiles(ctx, n.Source, runner)
	case "vicidial.01.perl":
		var files []vicidialOwnedFile
		files, e = n.Dependency.deployment()
		if e == nil {
			_, e = vicidialOwned(ctx, runner, "perl", files, true)
		}
	case "vicidial.02.configuration":
		_, e = provisionVICIdialConfiguration(ctx, runner)
	case "vicidial.03.registration":
		var ami string
		ami, e = vicidialAMISecret()
		if e == nil && !vicidialHex(ami, 20) {
			e = errVICIdial
		}
		if e == nil {
			var row vicidialRegistration
			row, _, e = inspectVICIdialRegistration(ctx, runner, ami)
			if e == nil {
				e = preserveVICIdialRegistration(ctx, runner, row)
			}
			if e == nil {
				_, e = registerVICIdial(ctx, runner, ami, hash(row))
			}
		}
	case "vicidial.04.generate":
		var ami string
		ami, e = vicidialAMISecret()
		if e == nil {
			_, e = generateVICIdial(ctx, runner, n.Asterisk.Database.Native.Runner, n.Source, n.binding(), r.stage.doc().Before.GeneratorInputs, ami)
		}
		if e == nil {
			e = reloadVICIdialAsterisk(ctx, n.Asterisk)
		}
	case "vicidial.05.web":
		e = n.activateWeb(ctx, runner)
	case "vicidial.06.verify":
		e = n.verifyRuntime(ctx, runner)
	default:
		e = errVICIdial
	}
	result := execution.Result{Started: true}
	if e != nil {
		result.ExitCode = 1
	}
	return result, e
}

func (n VICIdialNative) activateWeb(ctx context.Context, runner vicidialInputRunner) error {
	if _, e := n.inspectWeb(ctx, runner); e != nil {
		return e
	}
	if _, e := vicidialOwned(ctx, runner, "web", vicidialWebConfiguration(), true); e != nil {
		return e
	}
	run := func(args ...string) error { _, e := n.Asterisk.run(ctx, dbCommand(args[0], args[1:]...)); return e }
	if run("/usr/sbin/httpd", "-t", "-f", "/etc/httpd/conf/almadial-s05.conf") != nil || run("/usr/bin/systemctl", "daemon-reload") != nil {
		return errVICIdial
	}
	selection, e := n.Asterisk.run(ctx, dbCommand("/usr/bin/systemctl", "show", "httpd.service", "--property=LoadState,ActiveState,MainPID,FragmentPath,DropInPaths,Environment"))
	if e != nil {
		return errVICIdial
	}
	if _, e = verifyVICIdialWebUnit(selection.Stdout, false); e != nil {
		return e
	}
	if run("/usr/bin/systemctl", "start", "php-fpm.service", "httpd.service") != nil {
		return errVICIdial
	}
	if state, e := n.inspectWeb(ctx, runner); e != nil || !shaPattern.MatchString(state) {
		return errVICIdial
	}
	// Initial activation qualifies persistence once. A fresh satisfied assessment
	// omits this graph action and therefore cannot restart either service.
	if run("/usr/bin/systemctl", "restart", "httpd.service") != nil {
		return errVICIdial
	}
	if state, e := n.inspectWeb(ctx, runner); e != nil || !shaPattern.MatchString(state) {
		return errVICIdial
	}
	return nil
}
func vicidialSyntaxScripts() []string {
	return []string{"ADMIN_keepalive_ALL.pl", "AST_update.pl", "AST_update_AMI2.pl", "AST_manager_listen.pl", "AST_manager_send.pl", "AST_VDauto_dial.pl", "AST_VDadapt.pl", "AST_conf_update.pl", "FastAGI_log.pl"}
}
func (n VICIdialNative) verifyRuntime(ctx context.Context, runner vicidialInputRunner) error {
	// A zero-local-phone deployment is valid, but it cannot qualify the phone
	// generation branch. Exercise that branch twice with synthetic in-memory
	// rows, without editing the seed phones or loading fixture endpoints.
	first, e := qualifyVICIdialPhoneFixture(ctx, n.Source, runner)
	if e != nil {
		return e
	}
	second, e := qualifyVICIdialPhoneFixture(ctx, n.Source, runner)
	if e != nil || hash(first) != hash(second) {
		return errVICIdial
	}
	if reloadVICIdialAsterisk(ctx, n.Asterisk) != nil {
		return errVICIdial
	}
	// Match M0's non-root prerequisite check without executing an AGI request.
	if _, e := n.Asterisk.run(ctx, dbCommand("/usr/sbin/runuser", "-u", "asterisk", "--", "/usr/bin/perl", "-MAsterisk::AGI", "-e", "print qq(Asterisk::AGI loaded as non-root\\n)")); e != nil {
		return errVICIdial
	}
	// Parse only verified installed sources. Perl -c evaluates approved module
	// imports but never enters the upstream keepalive or dialing main loops.
	for _, name := range vicidialSyntaxScripts() {
		if _, e := n.Asterisk.run(ctx, dbCommand("/usr/bin/perl", "-c", vicidialHome+"/"+name)); e != nil {
			return errVICIdial
		}
	}
	count := 0
	for _, file := range n.Source.Inventory() {
		if strings.HasPrefix(file.Path, "www/") && strings.HasSuffix(file.Path, ".php") {
			if _, e := n.Asterisk.run(ctx, dbCommand("/usr/bin/php", "-l", vicidialWeb+strings.TrimPrefix(file.Path, "www"))); e != nil {
				return errVICIdial
			}
			count++
		}
	}
	if count != 307 {
		return errVICIdial
	}
	if e := verifyVICIdialDeferred(ctx, runner); e != nil {
		return e
	}
	fresh, e := n.InspectVICIdial(ctx)
	if e != nil || !vicidialComponentsComplete(fresh) {
		return errVICIdial
	}
	_, e = n.runtimeReceipt(ctx, runner, fresh, true)
	return e
}
func verifyVICIdialDeferred(ctx context.Context, runner vicidialInputRunner) error {
	var result struct{ Inactive bool }
	if vicidialAdapter(ctx, runner, "deferred-inspect", struct{}{}, &result) != nil || !result.Inactive {
		return errVICIdial
	}
	return nil
}
