//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

// InstallRequest is the bounded local laboratory product profile. Certificate
// issuance and artifact acquisition remain explicit inputs, not hidden downloads.
// CANDIDATE qualification is never production installation authorization.
type InstallRequest struct {
	Config        config.Config
	Requirements  preflight.Requirements
	Recording     RecordingBudget
	TLS           TLSRequest
	Network       NetworkRequest
	Phone         PhoneRequest
	Sources       InstallSourcePaths
	Authorization InstallAuthorization
}
type InstallSourcePaths struct {
	VICIdial, Perl, Asterisk, Signature, Keyring, Scratch, PJProject, Phone string
}
type InstallAuthorization struct {
	TaskID, Host, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, ManagementRecoveryVerified       bool
}
type InstallSources struct {
	Database    DatabaseSource
	Asterisk    AsteriskSource
	Application VICIdialSource
	Perl        VICIdialPerlSource
	Phone       PhoneSource
}

func LoadInstallSources(ctx context.Context, m artifacts.Manifest, p InstallSourcePaths) (s InstallSources, e error) {
	if s.Database, e = LoadDatabaseSource(ctx, m, p.VICIdial); e != nil {
		return
	}
	if s.Application, e = LoadVICIdialSource(ctx, m, p.VICIdial); e != nil {
		return
	}
	if s.Perl, e = LoadVICIdialPerlSource(ctx, m, p.Perl); e != nil {
		return
	}
	if s.Asterisk, e = LoadAsteriskSource(ctx, m, p.Asterisk, p.Signature, p.Keyring, p.Scratch, p.PJProject); e != nil {
		return
	}
	s.Phone, e = LoadPhoneSource(ctx, m, p.Phone)
	return
}
func (s InstallSources) identities() []string {
	return []string{s.Database.Digest(), s.Asterisk.Digest(), s.Application.Digest(), s.Perl.Digest(), s.Phone.Digest()}
}

type installDocument struct {
	Request     InstallRequest
	Initial     Inputs
	Manifest    string
	Sources     []string
	Certificate TLSCertificate
	Stages      []string
}

// InstallPlan owns its serialized blueprint. Stage action graphs are specialized
// only after fresh mandatory upstream postconditions succeed. This preserves the
// accepted planners' identity rules instead of inventing future host snapshots.
type InstallPlan struct{ data string }

func (p InstallPlan) JSON() []byte { return []byte(p.data) }
func (p InstallPlan) Digest() string {
	if p.data == "" {
		return ""
	}
	return hash(json.RawMessage(p.data))
}
func (p InstallPlan) document() (d installDocument, e error) { e = decode(p.data, &d); return }

var installStages = []string{"S01 packages and repositories", "S02 identities, storage and SELinux", "S03 MariaDB and VICIdial schema", "S04 Asterisk and native modules", "S05 VICIdial application", "S06 FastAGI and deferred scheduling", "S07 Apache/PHP application delivery", "S08 trusted supplied-certificate TLS", "S09 isolated host firewall", "S10 pinned VICIphone and synthetic PJSIP", "S11 synthetic call/media acceptance (qualification)", "S12 read-only diagnostics"}
var errInstall = errors.New("unified installation contract failed; preserve stage journal and use the same approved plan for resume")

// ErrInstallPaused marks a deliberate stop at a verified stage boundary. It
// never interrupts a package transaction, schema import or service action.
var ErrInstallPaused = errors.New("installation paused after verified stage; resume with the same approved plan")

func PrepareInstall(r InstallRequest, initial Inputs, m artifacts.Manifest, sources InstallSources, cert TLSCertificate) (InstallPlan, error) {
	if hash(r.Config) != hash(initial.Config) || hash(r.Requirements) != hash(initial.Requirements) || initial.Transaction != nil || initial.PrepareRepositoryTrust || !networkRequestValid(r.Network) || !phoneRequestValid(r.Phone) || !tlsRequestValid(r.TLS) || !shaPattern.MatchString(cert.LeafSHA256) {
		return InstallPlan{}, errInstall
	}
	if r.Network.Mode == "remote" {
		c := config.Defaults(r.Config)
		if r.TLS.Mode != "remote" || r.TLS.Hostname != c.Network.Hostname || r.Phone.Hostname != c.Network.Hostname || r.Phone.WSSURL != "wss://"+c.Network.Hostname+"/ws" || c.Network.BindAddress.String() != r.Network.ManagementAddress || c.Network.AdvertiseAddress != c.Network.BindAddress || int(r.Network.RTPStart) != *c.RTP.Start || int(r.Network.RTPEnd) != *c.RTP.End {
			return InstallPlan{}, errInstall
		}
	} else if r.TLS.Mode != "provided" || r.TLS.Hostname != "localhost" || r.Phone.Hostname != "localhost" {
		return InstallPlan{}, errInstall
	}
	a := r.Authorization
	if (a.TaskID != "xu7wxvch94ii2iw38hpb8d58" && a.TaskID != "it8aq4utqix0uz8sjgxrtssk" && a.TaskID != "a1doi4qsyn1lspqtydw84smh") || a.Host != initial.Inventory.Host || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged || !a.ManagementRecoveryVerified {
		return InstallPlan{}, errInstall
	}
	if _, e := initial.Prepare(m); e != nil { // Resolution requires trust; the clean host may still need the accepted key-import action.
		trust := initial
		trust.PrepareRepositoryTrust = true
		if _, e = trust.Prepare(m); e != nil {
			return InstallPlan{}, e
		}
	}
	parent, e := plan.New(initial.Config, initial.Preflight, initial.Requirements, m.Digest())
	if e != nil {
		return InstallPlan{}, e
	}
	if !parent.Eligible() {
		return InstallPlan{}, fmt.Errorf("clean-host preflight blocked: %s", preflight.Human(parent.Findings()))
	}
	for _, id := range sources.identities() {
		if !shaPattern.MatchString(id) {
			return InstallPlan{}, errInstall
		}
	}
	return InstallPlan{jsonString(installDocument{r, initial, m.Digest(), sources.identities(), cert, append([]string{}, installStages...)})}, nil
}
func DecodeInstallPlan(raw []byte, m artifacts.Manifest, sources InstallSources) (InstallPlan, error) {
	var d installDocument
	if decode(string(raw), &d) != nil {
		return InstallPlan{}, errInstall
	}
	p, e := PrepareInstall(d.Request, d.Initial, m, sources, d.Certificate)
	if e != nil || string(p.JSON()) != string(raw) {
		return InstallPlan{}, errInstall
	}
	return p, nil
}

// InstallResult deliberately distinguishes provisioned state from real browser,
// media and reboot qualification. Those results cannot be inferred from journals.
type InstallResult struct {
	PlanSHA256            string   `json:"plan_sha256"`
	Completed             []string `json:"completed"`
	NoOp                  bool     `json:"no_op"`
	QualificationRequired []string `json:"qualification_required"`
}
type installRun struct {
	ctx        context.Context
	plan       InstallPlan
	d          installDocument
	m          artifacts.Manifest
	sources    InstallSources
	native     Native
	state      string
	lock       *journal.Session
	log        *operatorlog.Logger
	result     InstallResult
	pauseAfter string
}

func (r *installRun) preserve(name string, v any) error {
	return r.lock.PreserveInput(name+".input.json", []byte(jsonString(v)))
}
func (r *installRun) read(name string, v any) error {
	b, e := journal.InspectInput(r.state, name+".input.json")
	if e != nil {
		return e
	}
	return decode(string(b), v)
}
func (r *installRun) stageState(name string) string { return filepath.Join(r.state, name) }
func (r *installRun) resume(name string) (bool, error) {
	_, e := journal.Inspect(r.stageState(name))
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	return e == nil, e
}

// ExecuteInstall retains the existing flock for the whole workflow. Each stage
// retains its own existing journal, intent, executor and reconciliation policy.
// The immutable inputs are not an alternative installation progress database.
func ExecuteInstall(ctx context.Context, p InstallPlan, m artifacts.Manifest, s InstallSources, n Native, state, approval string, log *operatorlog.Logger) (InstallResult, error) {
	return ExecuteInstallCheckpoint(ctx, p, m, s, n, state, approval, log, "")
}

func ExecuteInstallCheckpoint(ctx context.Context, p InstallPlan, m artifacts.Manifest, s InstallSources, n Native, state, approval string, log *operatorlog.Logger, pauseAfter string) (InstallResult, error) {
	result := InstallResult{PlanSHA256: p.Digest(), Completed: []string{}, QualificationRequired: []string{"browser registration and bounded call/media/recording", "reboot persistence", "independent QA"}}
	if pauseAfter != "" && pauseAfter != "S03" {
		return result, errInstall
	}
	d, e := p.document()
	if e != nil || approval != p.Digest() || approval == "" {
		return result, errInstall
	}
	validated, e := DecodeInstallPlan(p.JSON(), m, s)
	if e != nil || validated.Digest() != p.Digest() {
		return result, errInstall
	}
	current, e := n.Inspect(ctx)
	if e != nil || current.Host != d.Initial.Inventory.Host {
		return result, errInstall
	}
	// Validate supplied material before any stage can mutate the host, even
	// when TLS itself will be installed only after its web prerequisites.
	certificate, e := (TLSNative{Request: d.Request.TLS}).Material()
	if e != nil || certificate != d.Certificate {
		return result, fmt.Errorf("installation TLS input changed or invalid: %w", errInstall)
	}
	lock, e := journal.Acquire(state)
	if e != nil {
		return result, e
	}
	defer lock.Close()
	r := installRun{ctx: ctx, plan: p, d: d, m: m, sources: s, native: n, state: state, lock: lock, log: log, result: result, pauseAfter: pauseAfter}
	if e = r.preserve("approved-plan", json.RawMessage(p.JSON())); e != nil {
		return result, e
	}
	// A complete receipt is only an input for fresh recursive S01-S10 inspection.
	var phone PhoneInputs
	if e = r.read("S10-accepted", &phone); e == nil {
		if e = r.verifyComplete(phone); e != nil {
			return result, e
		}
		r.result.NoOp = true
		r.result.Completed = append([]string{}, installStages[:10]...)
		return r.result, nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return result, e
	}
	if e = r.packages(); e != nil {
		return r.result, fmt.Errorf("S01: %w", e)
	}
	if e = r.backend(); e != nil {
		return r.result, e
	}
	if e = r.read("S10-accepted", &phone); e != nil {
		return r.result, e
	}
	if e = r.verifyComplete(phone); e != nil {
		return r.result, e
	}
	return r.result, nil
}
func (r *installRun) verifyComplete(i PhoneInputs) error {
	n := r.phoneNative(i)
	fresh, e := n.InspectPhone(r.ctx)
	if e != nil {
		return fmt.Errorf("fresh S01-S10 verification: %w", e)
	}
	i.Snapshot = fresh
	stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database, r.sources.Phone)
	if e != nil || len(stage.Graph().Actions()) != 0 {
		return errors.New("fresh installed state requires attention; no automatic replay of completed stages")
	}
	return nil
}

// AssessInstall reuses the recursive accepted inspectors without acquiring a
// mutation lock or publishing inputs. A receipt alone never proves readiness.
func AssessInstall(ctx context.Context, p InstallPlan, m artifacts.Manifest, s InstallSources, n Native, state string) error {
	validated, e := DecodeInstallPlan(p.JSON(), m, s)
	if e != nil || validated.Digest() != p.Digest() {
		return errInstall
	}
	d, e := p.document()
	if e != nil {
		return e
	}
	fresh, e := n.Inspect(ctx)
	if e != nil || fresh.Host != d.Initial.Inventory.Host {
		return errInstall
	}
	r := installRun{ctx: ctx, plan: p, d: d, m: m, sources: s, native: n, state: state}
	var accepted json.RawMessage
	if e = r.read("approved-plan", &accepted); e != nil || string(accepted) != string(p.JSON()) {
		return errInstall
	}
	var phone PhoneInputs
	if e = r.read("S10-accepted", &phone); e != nil {
		return e
	}
	return r.verifyComplete(phone)
}
func (r *installRun) phoneNative(i PhoneInputs) PhoneNative {
	ast := i.Parent.Parent.Parent.Parent.Parent.Parent
	db := ast.Parent
	app := VICIdialNative{Asterisk: AsteriskNative{Database: DatabaseNative{Native: r.native, Budget: db.Parent.Budget, Source: r.sources.Database}, Source: r.sources.Asterisk}, Accepted: ast.Snapshot, DatabaseAccepted: db.Snapshot, Source: r.sources.Application, Dependency: r.sources.Perl}
	return PhoneNative{Network: NetworkNative{TLS: TLSNative{Web: WebNative{Runtime: RuntimeNative{Application: app}}, Request: i.Parent.Parent.Request, Certificate: i.Parent.Parent.Certificate}, Request: i.Parent.Request, Dependencies: i.Parent.Dependencies}, Request: i.Request, Source: r.sources.Phone}
}

func (r *installRun) rpm(name string, i Inputs) (Outcome, error) {
	var saved Inputs
	if e := r.read(name, &saved); e == nil {
		i = saved
	} else if !errors.Is(e, os.ErrNotExist) {
		return Outcome{}, e
	} else if e = r.preserve(name, i); e != nil {
		return Outcome{}, e
	}
	stage, e := i.Prepare(r.m)
	if e != nil {
		return Outcome{}, e
	}
	d := stage.doc()
	a := r.d.Request.Authorization
	auth := LabAuthorization{Scope: "ALM-16/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged}
	resume, e := r.resume(name)
	if e != nil {
		return Outcome{}, e
	}
	return ExecuteLab(r.ctx, stage, r.m, auth, r.stageState(name), r.native, resume, r.log)
}
func (r *installRun) packages() error {
	var accepted Inputs
	if e := r.read("S01-accepted", &accepted); e == nil {
		var apply Inputs
		if e = r.read("S01-apply", &apply); e != nil {
			return e
		}
		fresh, e := r.native.Inspect(r.ctx)
		if e != nil {
			return e
		}
		if apply.Transaction == nil {
			return errInstall
		}
		if e = Verify(apply.Inventory, fresh, *apply.Transaction); e != nil {
			return e
		}
		r.result.Completed = append(r.result.Completed, "S01")
		return nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	i := r.d.Initial
	i.PrepareRepositoryTrust = true
	if _, e := r.rpm("S01-trust", i); e != nil {
		return e
	}
	fresh, e := r.native.Inspect(r.ctx)
	if e != nil {
		return e
	}
	i.Inventory = fresh
	i.PrepareRepositoryTrust = false
	var tx Transaction
	if e = r.read("S01-transaction", &tx); errors.Is(e, os.ErrNotExist) {
		outcome, ee := r.rpm("S01-resolve", i)
		if ee != nil {
			return ee
		}
		if outcome.Transaction == nil {
			return errors.New("S01 resolution output unavailable after interruption; preserve journal and obtain a newly reviewed metadata-only resolution")
		}
		tx = *outcome.Transaction
		if e = r.preserve("S01-transaction", tx); e != nil {
			return e
		}
	} else if e != nil {
		return e
	}
	i.Transaction = &tx
	if _, e = r.rpm("S01-apply", i); e != nil {
		return e
	}
	fresh, e = r.native.Inspect(r.ctx)
	if e != nil || Verify(i.Inventory, fresh, tx) != nil {
		return errInstall
	}
	i.Inventory = fresh
	i.Transaction = nil
	if e = r.preserve("S01-accepted", i); e != nil {
		return e
	}
	r.result.Completed = append(r.result.Completed, "S01")
	return nil
}

// driveInstallStage only retains approved inputs and invokes the existing stage
// executor. It neither interprets action states nor retries effects; Resume is
// entirely owned by that executor and its accepted journal observer.
func driveInstallStage[T any](r *installRun, name string, inspect func() (T, error), execute func(T, bool) error) (out T, err error) {
	if e := r.read(name+"-accepted", &out); e == nil {
		r.result.Completed = append(r.result.Completed, name)
		return out, nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return out, e
	}
	var before T
	e := r.read(name+"-before", &before)
	if errors.Is(e, os.ErrNotExist) {
		if before, e = inspect(); e != nil {
			return out, fmt.Errorf("%s inspect: %w", name, e)
		}
		if e = r.preserve(name+"-before", before); e != nil {
			return out, e
		}
	} else if e != nil {
		return out, e
	}
	resume, e := r.resume(name)
	if e != nil {
		return out, e
	}
	if e = execute(before, resume); e != nil {
		return out, fmt.Errorf("%s execution: %w", name, e)
	}
	if out, e = inspect(); e != nil {
		return out, fmt.Errorf("%s postconditions: %w", name, e)
	}
	if e = r.preserve(name+"-accepted", out); e != nil {
		return out, e
	}
	r.result.Completed = append(r.result.Completed, name)
	if name == r.pauseAfter {
		return out, ErrInstallPaused
	}
	return out, nil
}

func (r *installRun) backend() error {
	var base Inputs
	if e := r.read("S01-accepted", &base); e != nil {
		return e
	}
	a := r.d.Request.Authorization
	system, e := driveInstallStage(r, "S02", func() (SystemInputs, error) {
		s, e := r.native.InspectSystem(r.ctx)
		return SystemInputs{Parent: base, Snapshot: s, Budget: r.d.Request.Recording}, e
	}, func(i SystemInputs, resume bool) error {
		stage, e := i.Prepare(r.m)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := SystemLabAuthorization{Scope: "ALM-17/disposable-lab", Host: d.Before.RPM.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged}
		_, e = ExecuteSystemLab(r.ctx, stage, r.m, auth, r.stageState("S02"), r.native, r.native.Runner, resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	database := DatabaseNative{Native: r.native, Budget: r.d.Request.Recording, Source: r.sources.Database}
	db, e := driveInstallStage(r, "S03", func() (DatabaseInputs, error) {
		s, e := database.InspectDatabase(r.ctx)
		return DatabaseInputs{Parent: system, Snapshot: s, SourceSHA256: r.sources.Database.Digest()}, e
	}, func(i DatabaseInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := DatabaseLabAuthorization{Scope: "ALM-18/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, SourceSHA256: d.Source}
		_, e = ExecuteDatabaseLab(r.ctx, stage, r.m, auth, r.stageState("S03"), database, database.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	asterisk := AsteriskNative{Database: database, Source: r.sources.Asterisk}
	ast, e := driveInstallStage(r, "S04", func() (AsteriskInputs, error) {
		s, e := asterisk.InspectAsterisk(r.ctx)
		return AsteriskInputs{Parent: db, Snapshot: s, SourceSHA256: r.sources.Asterisk.Digest()}, e
	}, func(i AsteriskInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Asterisk)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := AsteriskLabAuthorization{Scope: "ALM-19/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, SourceSHA256: d.Source}
		_, e = ExecuteAsteriskLab(r.ctx, stage, r.m, auth, r.stageState("S04"), asterisk, asterisk.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	application := VICIdialNative{Asterisk: asterisk, Accepted: ast.Snapshot, DatabaseAccepted: db.Snapshot, Source: r.sources.Application, Dependency: r.sources.Perl}
	app, e := driveInstallStage(r, "S05", func() (VICIdialInputs, error) {
		s, e := application.InspectVICIdial(r.ctx)
		return VICIdialInputs{Parent: ast, Snapshot: s, SourceSHA256: r.sources.Application.Digest(), DependencySHA256: r.sources.Perl.Digest()}, e
	}, func(i VICIdialInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := VICIdialLabAuthorization{Scope: "ALM-20/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, SourceSHA256: d.Source, DependencySHA256: d.Dependency}
		_, e = ExecuteVICIdialLab(r.ctx, stage, r.m, auth, r.stageState("S05"), application, application.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	runtime := RuntimeNative{Application: application}
	rt, e := driveInstallStage(r, "S06", func() (RuntimeInputs, error) {
		s, e := runtime.InspectRuntime(r.ctx)
		return RuntimeInputs{Parent: app, Snapshot: s}, e
	}, func(i RuntimeInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := RuntimeLabAuthorization{Scope: "ALM-21/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, PolicySHA256: d.Policy, DialingInactive: true}
		_, e = ExecuteRuntimeLab(r.ctx, stage, r.m, auth, r.stageState("S06"), runtime, runtime.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	web := WebNative{Runtime: runtime}
	wb, e := driveInstallStage(r, "S07", func() (WebInputs, error) {
		s, e := web.InspectWeb(r.ctx)
		return WebInputs{Parent: rt, Snapshot: s}, e
	}, func(i WebInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := WebLabAuthorization{Scope: "ALM-23/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, PolicySHA256: d.Policy, DialingInactive: true}
		_, e = ExecuteWebLab(r.ctx, stage, r.m, auth, r.stageState("S07"), web, web.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	tls := TLSNative{Web: web, Request: r.d.Request.TLS, Certificate: r.d.Certificate}
	ts, e := driveInstallStage(r, "S08", func() (TLSInputs, error) {
		s, e := tls.InspectTLS(r.ctx)
		return TLSInputs{Parent: wb, Snapshot: s, Request: r.d.Request.TLS, Certificate: r.d.Certificate}, e
	}, func(i TLSInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := TLSLabAuthorization{Scope: "ALM-24/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, PolicySHA256: d.Policy, DialingInactive: true}
		_, e = ExecuteTLSLab(r.ctx, stage, r.m, auth, r.stageState("S08"), tls, tls.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	dependencies, e := r.firewallPackages(base)
	if e != nil {
		return e
	}
	network := NetworkNative{TLS: tls, Request: r.d.Request.Network, Dependencies: &dependencies}
	net, e := driveInstallStage(r, "S09", func() (NetworkInputs, error) {
		s, e := network.InspectNetwork(r.ctx)
		return NetworkInputs{Parent: ts, Snapshot: s, Request: r.d.Request.Network, Dependencies: &dependencies}, e
	}, func(i NetworkInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := NetworkLabAuthorization{Scope: "ALM-25/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, PolicySHA256: d.Policy, DialingInactive: true, ManagementRecoveryVerified: a.ManagementRecoveryVerified}
		_, e = ExecuteNetworkLab(r.ctx, stage, r.m, auth, r.stageState("S09"), network, network.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	phone := PhoneNative{Network: network, Request: r.d.Request.Phone, Source: r.sources.Phone}
	ph, e := driveInstallStage(r, "S10", func() (PhoneInputs, error) {
		s, e := phone.InspectPhone(r.ctx)
		return PhoneInputs{Parent: net, Snapshot: s, Request: r.d.Request.Phone, PhoneSourceSHA256: r.sources.Phone.Digest()}, e
	}, func(i PhoneInputs, resume bool) error {
		stage, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database, r.sources.Phone)
		if e != nil {
			return e
		}
		d := stage.doc()
		auth := PhoneLabAuthorization{Scope: "ALM-26/disposable-lab", Host: d.Before.Host, StageSHA256: stage.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SnapshotReference: a.SnapshotReference, OperatorEvidenceSHA256: a.OperatorEvidenceSHA256, CandidateAcknowledged: a.CandidateAcknowledged, PolicySHA256: d.Policy, DialingInactive: true, ManagementRecoveryVerified: a.ManagementRecoveryVerified}
		_, e = ExecutePhoneLab(r.ctx, stage, r.m, auth, r.stageState("S10"), phone, phone.Runner(stage), resume, r.log)
		return e
	})
	if e != nil {
		return e
	}
	_ = ph
	return nil
}

func (r *installRun) firewallPackages(base Inputs) (NetworkDependencies, error) {
	var dependencies NetworkDependencies
	if e := r.read("S09-dependencies", &dependencies); e != nil {
		if !errors.Is(e, os.ErrNotExist) {
			return dependencies, e
		}
		before, e := r.native.Inspect(r.ctx)
		if e != nil || before.Digest() != base.Inventory.Digest() {
			return dependencies, errInstall
		}
		var candidates []Candidate
		if json.Unmarshal([]byte(networkDependencyCandidates), &candidates) != nil {
			return dependencies, errInstall
		}
		dependencies = NetworkDependencies{Before: before, Transaction: normalize(Transaction{BeforeSHA256: before.Digest(), Install: candidates, Remove: []Package{}})}
		if !dependencies.valid() {
			return dependencies, errInstall
		}
		if e = r.preserve("S09-dependencies", dependencies); e != nil {
			return dependencies, e
		}
	}
	fresh, e := r.native.Inspect(r.ctx)
	if e != nil {
		return dependencies, e
	}
	if dependencies.Verify(fresh) == nil {
		return dependencies, nil
	}
	input := base
	input.Inventory = dependencies.Before
	input.Transaction = &dependencies.Transaction
	if _, e = r.rpm("S09-packages", input); e != nil {
		return dependencies, e
	}
	fresh, e = r.native.Inspect(r.ctx)
	if e != nil {
		return dependencies, e
	}
	return dependencies, dependencies.Verify(fresh)
}

// InstallPlanRequest returns only the source locations needed to validate an
// untrusted plan. It grants no execution authority; DecodeInstallPlan must follow.
func InstallPlanRequest(raw []byte) (InstallRequest, error) {
	var d installDocument
	if e := decode(string(raw), &d); e != nil {
		return InstallRequest{}, errInstall
	}
	return d.Request, nil
}
