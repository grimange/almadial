package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

func syntheticHostAndBudget(t *testing.T, bindIP netip.Addr) (preflight.Snapshot, preflight.Requirements) {
	t.Helper()
	s := preflight.Snapshot{
		HostID:          "m1-gate-synthetic-host",
		OS:              "almalinux",
		Release:         "10.2",
		Architecture:    "amd64",
		CPUs:            4,
		CPUFlags:        []string{"sse2"},
		TotalMemory:     8192,
		AvailableMemory: 4096,
		Disks:           []preflight.Disk{{Path: "/", Available: 99999999}},
		Addresses:       []netip.Addr{bindIP},
		SELinux:         "Enforcing",
		TimeSync:        "yes",
		Known:           map[string]bool{},
	}
	for _, k := range preflight.RequiredProbes() {
		s.Known[k] = true
	}
	r := preflight.Requirements{
		Profile:            "SYNTHETIC-M1-GATE",
		MinCPUs:            2,
		CPUFlags:           []string{"sse2"},
		MinTotalMemory:     2048,
		MinAvailableMemory: 1024,
		MinDiskAvailable:   10000,
	}
	return s, r
}

func loadManifest(t *testing.T) artifacts.Manifest {
	t.Helper()
	f, err := os.Open("../../manifests/stack.lock.yaml")
	if err != nil {
		t.Fatalf("cannot open stack manifest: %v", err)
	}
	defer f.Close()
	m, err := artifacts.Load(f)
	if err != nil {
		t.Fatalf("cannot load stack manifest: %v", err)
	}
	return m
}

// 1. Configuration-to-Plan Integration (Section 5)
func TestM1GateIntegration_ConfigToPlan(t *testing.T) {
	m := loadManifest(t)

	// Scenario A: Basic IPv4 IP-only installation without WebRTC or hostname
	t.Run("Scenario A: Basic IP-only", func(t *testing.T) {
		f, err := os.Open("../config/testdata/basic.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		c, err := config.Load(f)
		if err != nil {
			t.Fatalf("config load failed: %v", err)
		}
		c.WebRTC.Enabled = false
		c.Network.Hostname = ""
		c = config.Defaults(c)

		if err := config.Validate(c); err != nil {
			t.Fatalf("config validation failed: %v", err)
		}
		cd, err := config.Digest(c)
		if err != nil || cd == "" {
			t.Fatalf("config digest failed: %v", err)
		}

		s, r := syntheticHostAndBudget(t, c.Network.BindAddress)
		p, err := plan.New(c, s, r, m.Digest())
		if err != nil {
			t.Fatalf("plan generation failed: %v", err)
		}
		if !p.Eligible() {
			t.Fatal("expected eligible plan for valid synthetic host")
		}

		stages := p.Stages()
		hasWebRTC := false
		for _, st := range stages {
			if strings.Contains(strings.ToLower(st.ID+st.Description), "webrtc") || strings.Contains(strings.ToLower(st.ID+st.Description), "wss") {
				hasWebRTC = true
			}
		}
		if hasWebRTC {
			t.Fatal("basic installation plan should not contain WebRTC stages")
		}
	})

	// Scenario B: WebRTC-enabled installation
	t.Run("Scenario B: WebRTC-enabled", func(t *testing.T) {
		f, err := os.Open("../config/testdata/webrtc.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		c, err := config.Load(f)
		if err != nil {
			t.Fatalf("config load failed: %v", err)
		}
		c = config.Defaults(c)

		if err := config.Validate(c); err != nil {
			t.Fatalf("config validation failed: %v", err)
		}
		s, r := syntheticHostAndBudget(t, c.Network.BindAddress)
		p, err := plan.New(c, s, r, m.Digest())
		if err != nil {
			t.Fatalf("plan generation failed: %v", err)
		}
		if !p.Eligible() {
			t.Fatal("expected eligible plan for WebRTC config")
		}

		stages := p.Stages()
		hasWebRTC := false
		for _, st := range stages {
			if strings.Contains(strings.ToLower(st.ID+st.Description), "webrtc") || strings.Contains(strings.ToLower(st.ID+st.Description), "tls") {
				hasWebRTC = true
			}
		}
		if !hasWebRTC {
			t.Fatal("WebRTC installation plan must contain WebRTC/TLS stages")
		}
	})

	// Scenario C: Invalid configurations rejected before planning
	t.Run("Scenario C: Invalid configurations", func(t *testing.T) {
		f, err := os.Open("../config/testdata/basic.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		base, _ := config.Load(f)

		for _, tc := range []struct {
			name   string
			mutate func(*config.Config)
		}{
			{"invalid port", func(c *config.Config) { p := 0; c.SIP.Port = &p }},
			{"invalid rtp range", func(c *config.Config) { s := 20000; e := 10000; c.RTP.Start = &s; c.RTP.End = &e }},
			{"missing TLS certificate when WebRTC enabled", func(c *config.Config) {
				c.WebRTC.Enabled = true
				c.WebRTC.TLS = &config.TLS{PrivateKeyFile: "/valid/key"}
			}},
			{"malformed credential reference", func(c *config.Config) {
				c.Credentials.Database = config.SecretRef{Type: "invalid-type", Source: "foo"}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := base
				tc.mutate(&c)
				c = config.Defaults(c)
				if err := config.Validate(c); err == nil {
					t.Fatalf("expected config validation failure for %s", tc.name)
				}
			})
		}
	})

	// Scenario D: Host incompatibility blocks plan eligibility
	t.Run("Scenario D: Host incompatibility", func(t *testing.T) {
		f, err := os.Open("../config/testdata/basic.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		c, _ := config.Load(f)
		c = config.Defaults(c)

		s, r := syntheticHostAndBudget(t, c.Network.BindAddress)
		s.OS = "ubuntu" // Incompatible OS
		p, err := plan.New(c, s, r, m.Digest())
		if err != nil {
			t.Fatalf("plan generation error: %v", err)
		}
		if p.Eligible() {
			t.Fatal("expected ineligible plan for non-AlmaLinux OS")
		}

		s, r = syntheticHostAndBudget(t, c.Network.BindAddress)
		s.CPUs = 1 // Inadequate CPU
		r.MinCPUs = 4
		p, err = plan.New(c, s, r, m.Digest())
		if err != nil {
			t.Fatalf("plan generation error: %v", err)
		}
		if p.Eligible() {
			t.Fatal("expected ineligible plan for inadequate CPUs")
		}
	})
}

// 2. Manifest-to-Plan Identity Integration (Section 6)
func TestM1GateIntegration_ManifestToPlanIdentity(t *testing.T) {
	m := loadManifest(t)
	f, _ := os.Open("../config/testdata/basic.json")
	defer f.Close()
	c, _ := config.Load(f)
	c = config.Defaults(c)
	s, r := syntheticHostAndBudget(t, c.Network.BindAddress)

	// 1. Same manifest produces identical plan identity
	p1, err := plan.New(c, s, r, m.Digest())
	if err != nil {
		t.Fatal(err)
	}
	p2, err := plan.New(c, s, r, m.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if p1.Digest() != p2.Digest() {
		t.Fatalf("expected identical plan digests, got %s vs %s", p1.Digest(), p2.Digest())
	}

	// 2. Changed manifest identity alters plan identity
	alteredManifestSHA := strings.Repeat("a", 64)
	pAltered, err := plan.New(c, s, r, alteredManifestSHA)
	if err != nil {
		t.Fatal(err)
	}
	if pAltered.Digest() == p1.Digest() {
		t.Fatal("altered manifest identity must yield distinct plan digest")
	}

	// 3. CANDIDATE manifest is NOT authorized for installation
	if m.Status() != "candidate" {
		t.Fatalf("expected candidate manifest status, got %s", m.Status())
	}
	if m.InstallationAuthorized() {
		t.Fatal("CANDIDATE manifest must NOT be authorized for installation")
	}
}

// 3. Approval and Authorization Boundaries (Section 8)
func TestM1GateIntegration_ApprovalBoundaries(t *testing.T) {
	m := loadManifest(t)
	f, _ := os.Open("../config/testdata/basic.json")
	defer f.Close()
	c, _ := config.Load(f)
	c = config.Defaults(c)
	s, r := syntheticHostAndBudget(t, c.Network.BindAddress)

	prep, err := operatorui.Prepare(c, s, r, m)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}

	cd, _ := config.Digest(c)
	pd := prep.Plan.Digest()

	// 1. Candidate manifest rejects public decision validation
	goodDecision := operatorui.Decision{Approved: true, ConfigSHA256: cd, PlanSHA256: pd}
	if err := goodDecision.Validate(prep); err == nil {
		t.Fatal("Validate must reject candidate manifest without installation authorization")
	}

	// 2. Interactive approval rejects candidate manifest
	var promptBuf bytes.Buffer
	prompts := operatorui.NewPrompts(context.Background(), strings.NewReader("yes\n"), &promptBuf)
	if _, err := prep.Approve(context.Background(), prompts); !errors.Is(err, operatorui.ErrBlocked) {
		t.Fatalf("Approve must fail with ErrBlocked on candidate manifest, got: %v", err)
	}

	// 3. Negative approval responses never approve
	for _, neg := range []string{"no\n", "n\n", "\n", "cancel\n", "\x03"} {
		prompts := operatorui.NewPrompts(context.Background(), strings.NewReader(neg), &promptBuf)
		if _, err := prep.Approve(context.Background(), prompts); err == nil {
			t.Fatalf("expected error on negative approval %q", neg)
		}
	}
}

// 4. Action Graph and Journal Integration Contracts (Section 9)
func TestM1GateIntegration_GraphAndJournalContracts(t *testing.T) {
	cmd := execution.Command{
		Executable:  "/bin/true",
		Args:        []string{"/bin/true"},
		Timeout:     5 * time.Second,
		StdoutLimit: 1024,
		StderrLimit: 1024,
	}
	action1 := execution.Action{
		ID:             "stage1.init",
		StageID:        "stage1",
		Description:    "Initialize stage 1 test resource",
		Dependencies:   []string{},
		Preconditions:  []string{"precondition.met"},
		Command:        cmd,
		Postconditions: []string{"stage1.initialized"},
	}
	action2 := execution.Action{
		ID:             "stage2.apply",
		StageID:        "stage2",
		Description:    "Apply stage 2 test resource",
		Dependencies:   []string{"stage1.init"},
		Preconditions:  []string{"stage1.initialized"},
		Command:        cmd,
		Postconditions: []string{"stage2.applied"},
	}

	graph, err := execution.NewGraph([]execution.Action{action1, action2})
	if err != nil {
		t.Fatalf("NewGraph failed: %v", err)
	}
	graphDigest := graph.Digest()
	if graphDigest == "" {
		t.Fatal("empty graph digest")
	}

	hashA := strings.Repeat("1", 64)
	id := journal.Identity{
		Host:      hashA,
		HostState: hashA,
		Config:    hashA,
		Manifest:  hashA,
		Plan:      hashA,
		Policy:    hashA,
		Graph:     graphDigest,
	}

	tmpDir := t.TempDir()
	os.Chmod(tmpDir, 0700)
	stateDir := filepath.Join(tmpDir, "state")

	// Acquire exclusive execution session
	session, err := journal.Acquire(stateDir)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer session.Close()

	// Contention: second process acquiring same dir fails with ErrBusy
	contender, err := journal.Acquire(stateDir)
	if !errors.Is(err, journal.ErrBusy) {
		if contender != nil {
			contender.Close()
		}
		t.Fatalf("expected ErrBusy on concurrent Acquire, got: %v", err)
	}

	// Initialize journal document
	doc := journal.Document{
		SchemaVersion: 1,
		Identity:      id,
		Actions: []journal.Record{
			{ID: "stage1.init", Stage: "stage1", Dependencies: []string{}, Postconditions: []string{"stage1.initialized"}, Retry: journal.Idempotent, State: journal.Pending, ExitCode: -1},
			{ID: "stage2.apply", Stage: "stage2", Dependencies: []string{"stage1.init"}, Postconditions: []string{"stage2.applied"}, Retry: journal.Idempotent, State: journal.Pending, ExitCode: -1},
		},
	}
	if err := session.Initialize(doc); err != nil {
		t.Fatalf("session.Initialize failed: %v", err)
	}

	evidence1 := sha256Hex("stage1-evidence-ok")
	observer := &mockObserver{
		observations: map[string]journal.Observation{
			"stage1.init":  {State: journal.Satisfied, Evidence: evidence1},
			"stage2.apply": {State: journal.Absent, Evidence: sha256Hex("absent")},
		},
	}

	// Execute action 1: Begin (Running) -> Finish (Succeeded)
	if err := session.Begin("stage1.init"); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	trace1 := execution.Trace{
		StageID:  "stage1",
		ActionID: "stage1.init",
		Outcome:  "succeeded",
		Result:   execution.Result{Started: true, ExitCode: 0},
	}
	if err := session.Finish(context.Background(), trace1, observer); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	// Begin action 2, simulate process crash leaving action 2 in Running
	if err := session.Begin("stage2.apply"); err != nil {
		t.Fatalf("Begin action 2 failed: %v", err)
	}
	session.Close() // Release lock simulate interrupted crash

	// Test Resume & Reconciliation: Acquire again and Resume
	resumeSession, err := journal.Acquire(stateDir)
	if err != nil {
		t.Fatalf("Acquire for resume failed: %v", err)
	}
	defer resumeSession.Close()

	// Expected doc has Pending actions for validation
	expectedDoc := journal.Document{
		SchemaVersion: 1,
		Identity:      id,
		Actions: []journal.Record{
			{ID: "stage1.init", Stage: "stage1", Dependencies: []string{}, Postconditions: []string{"stage1.initialized"}, Retry: journal.Idempotent, State: journal.Pending, ExitCode: -1},
			{ID: "stage2.apply", Stage: "stage2", Dependencies: []string{"stage1.init"}, Postconditions: []string{"stage2.applied"}, Retry: journal.Idempotent, State: journal.Pending, ExitCode: -1},
		},
	}

	if err := resumeSession.Resume(context.Background(), expectedDoc, observer); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	snap, err := resumeSession.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if snap.Actions[0].State != journal.Succeeded {
		t.Fatalf("expected action 1 Succeeded, got %s", snap.Actions[0].State)
	}
	// Action 2 was absent & idempotent, so it reconciled back to Pending
	if snap.Actions[1].State != journal.Pending {
		t.Fatalf("expected action 2 reconciled to Pending, got %s", snap.Actions[1].State)
	}
}

type mockObserver struct {
	observations map[string]journal.Observation
}

func (m *mockObserver) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	if obs, ok := m.observations[r.ID]; ok {
		return obs, nil
	}
	return journal.Observation{State: journal.Unknown}, nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// 5. Zero-Mutation Preparation Boundaries (Section 10)
func TestM1GateIntegration_ZeroMutationPreparation(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chmod(tmpDir, 0700)
	t.Setenv("ALMADIAL_STATE_DIR", filepath.Join(tmpDir, "state"))

	m := loadManifest(t)
	f, _ := os.Open("../config/testdata/basic.json")
	defer f.Close()
	c, _ := config.Load(f)
	c = config.Defaults(c)
	s, r := syntheticHostAndBudget(t, c.Network.BindAddress)

	// Perform all preparation operations
	prep, err := operatorui.Prepare(c, s, r, m)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := prep.Present(&out); err != nil {
		t.Fatal(err)
	}

	// Verify no state directory or state files were created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("preparation created unexpected files: %v", entries)
	}

	// Read-only Inspect on non-existent path creates zero files
	if _, err := journal.Inspect(filepath.Join(tmpDir, "nonexistent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist from Inspect, got: %v", err)
	}
	entries, _ = os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Fatalf("Inspect created unexpected files: %v", entries)
	}
}

// 6. Operator Logging and Redaction Integration (Section 11)
func TestM1GateIntegration_LoggingAndRedaction(t *testing.T) {
	secret := "SECRET-TOP-CONFIDENTIAL-KEY"
	redactor, err := operatorlog.NewRedactor([]string{secret})
	if err != nil {
		t.Fatalf("NewRedactor failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	logger, err := operatorlog.New(&stdout, &stderr, nil, redactor)
	if err != nil {
		t.Fatalf("operatorlog.New failed: %v", err)
	}

	// Test INFO routing to stdout
	recInfo := operatorlog.Record{Level: operatorlog.Info, Stage: "S01", Action: "check", Code: "ok", Message: "Diagnostic message with " + secret}
	if err := logger.Write(recInfo); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatal("INFO record must not write to stderr")
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatal("secret leaked in stdout")
	}
	if !strings.Contains(stdout.String(), "[REDACTED]") {
		t.Fatal("expected [REDACTED] in stdout")
	}

	// Test ERROR routing to stderr
	recErr := operatorlog.Record{Level: operatorlog.Error, Stage: "S02", Action: "install", Code: "failed", Message: "Error containing " + secret}
	if err := logger.Write(recErr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "[REDACTED]") {
		t.Fatal("expected [REDACTED] in stderr")
	}
	if strings.Contains(stderr.String(), secret) {
		t.Fatal("secret leaked in stderr")
	}

	// Test cross-stream fragment redaction within one record
	fragment1 := "Connecting with password: SECRET-TOP-"
	fragment2 := "CONFIDENTIAL-KEY and port 5060"
	recCombined := operatorlog.Record{Level: operatorlog.Info, Stage: "S03", Action: "db", Code: "connect", Message: fragment1}
	stdout.Reset()
	if err := logger.Write(recCombined, fragment2); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatal("split secret leaked across fragments")
	}
	if !strings.Contains(stdout.String(), "[REDACTED]") {
		t.Fatal("split secret was not redacted across fragments")
	}
}
