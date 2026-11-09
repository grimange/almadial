//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/plan"
)

// Stage is an immutable S01 specialization of the accepted M1 plan/action graph.
// It is not a second installation planner or a general executor.
type Stage struct {
	data  string
	graph execution.Graph
}
type stageData struct {
	Mode        string       `json:"mode"`
	Parent      string       `json:"parent"`
	Config      string       `json:"config"`
	Manifest    string       `json:"manifest"`
	Policy      string       `json:"policy"`
	Before      Snapshot     `json:"before"`
	Transaction *Transaction `json:"transaction,omitempty"`
	Graph       string       `json:"graph"`
}

func prepare(parent plan.Plan, m artifacts.Manifest, s Snapshot, mode string, t *Transaction) (Stage, error) {
	if parent.Digest() == "" || m.Digest() == "" || parent.ManifestSHA256() != m.Digest() || s.validateTrust(mode != "trust") != nil {
		return Stage{}, errPolicy
	}
	if t != nil {
		if t.Validate(s) != nil {
			return Stage{}, errPolicy
		}
		v := normalize(*t)
		t = &v
		mode = "apply"
	}
	actions := []execution.Action{}
	if (mode == "trust" && !contains(s.KeyHashes, epelKeySHA)) || mode == "resolve" || (t != nil && len(t.Install) > 0) {
		actions = append(actions, execution.Action{ID: "rpm." + mode, StageID: "S01", Description: "Qualified RPM " + mode + " using native DNF", Dependencies: []string{}, Preconditions: []string{"rpm.fresh"}, Command: nativeCommand(mode, s, t), Postconditions: []string{"rpm." + mode + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return Stage{}, e
	}
	d := stageData{mode, parent.Digest(), parent.ConfigSHA256(), m.Digest(), PolicyDigest(), canonical(s), t, g.Digest()}
	b, e := json.Marshal(d)
	if e != nil {
		return Stage{}, errPolicy
	}
	return Stage{string(b), g}, nil
}

// PrepareResolution plans the explicit metadata/cache operation, never executes
// it. Strict inspection needs neither metadata refresh nor this operation.
func PrepareResolution(p plan.Plan, m artifacts.Manifest, s Snapshot) (Stage, error) {
	return prepare(p, m, s, "resolve", nil)
}

// PrepareRepositoryTrust plans only the exact M0 EPEL public-key import.
// Repository files are neither created nor rewritten.
func PrepareRepositoryTrust(p plan.Plan, m artifacts.Manifest, s Snapshot) (Stage, error) {
	return prepare(p, m, s, "trust", nil)
}

// PreparePackages accepts a native solver result bound to the exact inventory.
func PreparePackages(p plan.Plan, m artifacts.Manifest, s Snapshot, t Transaction) (Stage, error) {
	return prepare(p, m, s, "apply", &t)
}
func (s Stage) JSON() []byte { return []byte(s.data) }
func (s Stage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s Stage) Graph() execution.Graph { return s.graph }
func (s Stage) doc() stageData         { var d stageData; _ = json.Unmarshal([]byte(s.data), &d); return d }

// LabAuthorization records externally established operator permission. It does
// not discover or grant that permission. Its digest bindings prevent reuse on a
// different host, configuration, manifest, graph or transaction. Normal installer
// approval remains governed by InstallationAuthorized(), which is unchanged.
type LabAuthorization struct {
	Scope                  string `json:"scope"`
	Host                   string `json:"host"`
	StageSHA256            string `json:"stage_sha256"`
	ParentSHA256           string `json:"parent_sha256"`
	ManifestSHA256         string `json:"manifest_sha256"`
	SnapshotReference      string `json:"snapshot_reference"`
	OperatorEvidenceSHA256 string `json:"operator_evidence_sha256"`
	CandidateAcknowledged  bool   `json:"candidate_acknowledged"`
}

func (a LabAuthorization) validate(s Stage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-16/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || d.Policy != PolicyDigest() || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !shaPattern.MatchString(a.SnapshotReference) || !a.CandidateAcknowledged {
		return &execution.Error{Category: execution.Precondition, StageID: "S01", Detail: "explicit disposable laboratory authorization and exact identity bindings required", ExitCode: -1}
	}
	return nil
}

type Outcome struct {
	NoOp        bool              `json:"no_op"`
	Transaction *Transaction      `json:"transaction,omitempty"`
	Traces      []execution.Trace `json:"traces"`
	Journal     *journal.Document `json:"journal,omitempty"`
}

type checks struct {
	native      Native
	d           stageData
	result      execution.Result
	after       Snapshot
	transaction *Transaction
}

func (c *checks) Supports(id string) bool {
	return id == "rpm.fresh" || id == "rpm.resolve.verified" || id == "rpm.apply.verified" || id == "rpm.trust.verified"
}
func (c *checks) Check(ctx context.Context, id string) error {
	s, e := c.native.Inspect(ctx)
	if e != nil {
		return e
	}
	switch id {
	case "rpm.fresh":
		if s.Digest() != c.d.Before.Digest() {
			return errPolicy
		}
	case "rpm.resolve.verified":
		if s.Digest() != c.d.Before.Digest() {
			return errPolicy
		}
		var t Transaction
		if decode(c.result.Stdout, &t) != nil {
			return errPolicy
		}
		t = normalize(t)
		c.transaction = &t
		if t.Validate(s) != nil {
			return errPolicy
		}
	case "rpm.apply.verified":
		if c.d.Transaction == nil || Verify(c.d.Before, s, *c.d.Transaction) != nil {
			return errPolicy
		}
	case "rpm.trust.verified":
		if verifyKeyImport(c.d.Before, s) != nil {
			return errPolicy
		}
	default:
		return errPolicy
	}
	c.after = s
	return nil
}
func (c *checks) Observe(ctx context.Context, _ journal.Record) (journal.Observation, error) {
	// Interrupted resolution has no durable transaction output: never fabricate it.
	if c.d.Mode == "resolve" && c.transaction == nil {
		return journal.Observation{State: journal.Unknown}, nil
	}
	if e := c.Check(ctx, "rpm."+c.d.Mode+".verified"); e != nil {
		return journal.Observation{State: journal.Unknown}, e
	}
	return journal.Observation{State: journal.Satisfied, Evidence: hash(struct{ Snapshot, Transaction string }{c.after.Digest(), hash(c.transaction)})}, nil
}

type capture struct {
	execution.CommandRunner
	c *checks
}

func (r capture) Run(ctx context.Context, cmd execution.Command) (execution.Result, error) {
	v, e := r.CommandRunner.Run(ctx, cmd)
	r.c.result = v
	return v, e
}

func (s Stage) journalDocument(a LabAuthorization) (journal.Document, error) {
	d := s.doc()
	identity := journal.Identity{Host: d.Before.Host, HostState: d.Before.Digest(), Config: d.Config, Manifest: d.Manifest, Plan: d.Parent, Policy: hash(struct{ Policy, Stage, Authorization string }{d.Policy, s.Digest(), hash(a)}), Graph: s.graph.Digest()}
	return journal.New(identity, s.graph, map[string]journal.Retry{s.graph.Actions()[0].ID: journal.Never})
}

// ExecuteLab is the sole mutation entry point for S01 qualification. The one
// fixed action is executed by ALM-13 after ALM-14 ownership and durable intent.
// The caller must supply an original approved Stage on resume; rebuilding from a
// partially changed host would destroy its recovery identity and is rejected.
func ExecuteLab(ctx context.Context, s Stage, m artifacts.Manifest, a LabAuthorization, stateDir string, n Native, resume bool, log *operatorlog.Logger) (Outcome, error) {
	out := Outcome{Traces: []execution.Trace{}}
	if ctx == nil || n.Runner == nil {
		return out, errPolicy
	}
	if e := a.validate(s, m); e != nil {
		return out, e
	}
	d := s.doc()
	fresh, e := n.Inspect(ctx)
	if e != nil {
		return out, e
	}
	trust := fresh
	if resume {
		// An interrupted native transaction is precisely why reconciliation is
		// needed. Let ALM-14 durably record uncertainty; the observer still uses
		// the actual incomplete history and cannot approve or replay it.
		trust.HistoryComplete = true
	}
	if fresh.Host != d.Before.Host || trust.validateTrust(d.Mode != "trust") != nil {
		return out, errPolicy
	}
	if !resume && fresh.Digest() != d.Before.Digest() {
		return out, &execution.Error{Category: execution.Precondition, StageID: "S01", Detail: "RPM plan stale; review fresh inventory and transaction", ExitCode: -1}
	}
	c := &checks{native: n, d: d}
	if len(s.graph.Actions()) == 0 {
		if d.Mode == "trust" {
			if verifyKeyImport(d.Before, fresh) != nil {
				return out, errPolicy
			}
		} else if d.Transaction == nil || Verify(d.Before, fresh, *d.Transaction) != nil {
			return out, errPolicy
		}
		out.NoOp = true
		out.Transaction = d.Transaction
		return out, nil
	}
	session, e := journal.Acquire(stateDir)
	if e != nil {
		return out, e
	}
	defer session.Close()
	id := s.graph.Actions()[0].ID
	doc, e := s.journalDocument(a)
	if e != nil {
		return out, e
	}
	if resume {
		e = session.Resume(ctx, doc, c)
		current, _ := session.Snapshot()
		out.Journal = &current
		if e != nil {
			return out, e
		}
		if len(current.Actions) == 1 && current.Actions[0].State == journal.Succeeded {
			return out, nil
		}
	} else if e = session.Initialize(doc); e != nil {
		return out, e
	}
	// Revalidate again under the retained installer lock. Native DNF also takes
	// its own standard package-manager lock and runs its transaction test.
	if e = c.Check(ctx, "rpm.fresh"); e != nil {
		return out, e
	}
	if e = session.Begin(id); e != nil {
		return out, e
	}
	if log != nil {
		_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S01", ActionID: id})
	}
	out.Traces, e = s.graph.Execute(ctx, capture{n.Runner, c}, c)
	if len(out.Traces) != 1 {
		return out, errors.New("RPM execution uncertain; inspect journal and reconcile")
	}
	t := out.Traces[0]
	// A canceled action can be blocked after durable Begin, before process start.
	// Persist its actual failure category; do not leave an unexplained running intent.
	if t.Outcome == "blocked" {
		t.Outcome = "failed"
	}
	finish := session.Finish(ctx, t, c)
	current, _ := session.Snapshot()
	out.Journal = &current
	out.Transaction = c.transaction
	if d.Transaction != nil {
		out.Transaction = d.Transaction
	}
	if log != nil {
		event, ee := operatorui.EventFromTrace(out.Traces[0])
		if ee == nil {
			_ = operatorui.RenderEvent(log, event)
		}
	}
	if finish != nil {
		return out, finish
	}
	return out, e
}
