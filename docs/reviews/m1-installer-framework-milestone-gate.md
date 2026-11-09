# AlmaDial — Milestone M1 (Installer Framework) Independent Integration & Acceptance Gate Report

- **Milestone:** M1 — Installer Framework
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Review Scope:** Work packages ALM-9 through ALM-15 (M1-01 through M1-07)
- **Independent Gate Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Repository Baseline HEAD:** `eaa147f20ae2588338def24f70e4e827791aa1c3`
- **Date:** 2026-11-09T15:08:00Z
- **Final Milestone Verdict:** `ALMADIAL_M1_INTEGRATION_GATE_ACCEPTED`

---

## 1. Review Identity and Governance

### 1.1 Authority and Roles
- **Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Purpose:** M1 delivers the core installer framework: CLI routing, strict typed configuration, read-only host preflight, immutable plan synthesis, manifest/artifact verification, bounded subprocess execution, atomic state journaling, exclusive file locking, interactive terminal wizard, deterministic progress reporting, and secret-safe operator logging.
- **Governance Finding:** Kaneo project `AlmaDial` (`ghritpte6e40o5ayktgz0vh1`) tracks 35 canonical work packages. Milestone M1 comprises tasks ALM-9 through ALM-15. No separate standalone milestone-gate task entity exists in the native Kaneo tracker. The milestone gate is evaluated and finalized through this independent cross-module audit, protected evidence ledger, and documented milestone transition.
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner across all seven M1 tasks. No ownership changes occurred during this gate review.

---

## 2. Accepted Work-Package Inventory

All seven M1 work packages are independently accepted in Kaneo and present in the repository:

| Work Package | Kaneo ID | Status | Assignee | Accepted Commit | QA Report | Evidence Directory |
|---|---|---|---|---|---|---|
| **ALM-9 / M1-01** — Go CLI & Reproducible Build | `y04mu7040nia07ii8ny8b3yr` | `done` | Codex | `cce3d155f31bfcdb5cfc2b1977f6ac5f21320150` | `docs/reviews/m1-01-go-cli-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-01/qa-20261106T225500Z/` |
| **ALM-10 / M1-02** — Typed Configuration & Validation | `bcyodkgz47sak45stno5um0b` | `done` | Codex | `782f5e3bbfacf15c3687f9197da647b0b6b23faa` | `docs/reviews/m1-02-typed-configuration-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-02/qa-20261106T234500Z/` |
| **ALM-11 / M1-03** — Host Preflight & Immutable Plan | `xfy4pagbrmppq3yijr7nz7d7` | `done` | Codex | `baaf4f647b9f5ce5b8cf6b2694a9939ee68d1192` | `docs/reviews/m1-03-preflight-and-immutable-plan-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-03/qa-20261107T004000Z/` |
| **ALM-12 / M1-04** — Manifest & Artifact Verification | `yrjd60b1at9ug4hetrkvz01v` | `done` | Codex | `3b229ee77185789d00f531c74c062d575ff03488` | `docs/reviews/m1-04-manifest-and-artifact-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-04/qa-20261107T021500Z/` |
| **ALM-13 / M1-05** — Bounded Runner & Error Taxonomy | `xrai81mxz5p8r600svlnqgts` | `done` | Codex | `a2de242a0d70596dc65d53da0d61cfaa957e9c56` | `docs/reviews/m1-05-bounded-runner-and-action-graph-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-05/qa-20261107T034500Z/` |
| **ALM-14 / M1-06** — Atomic Journal & Exclusive Lock | `q2kgwrrzpn4zkzgvj08y6g1h` | `done` | Codex | `c32a88a24613330f7dd9576deebd00be55f0b463` | `docs/reviews/m1-06-atomic-journal-and-exclusive-execution-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-06/qa-20261107T072500Z/` |
| **ALM-15 / M1-07** — Accessible Wizard & Logging | `r6qm3nzbahxpvcw8ejk71s0q` | `done` | Codex | `89fd1d78eec07383c29343a2ba0e848287c1bf77` / `eaa147f20ae2588338def24f70e4e827791aa1c3` | `docs/reviews/m1-07-accessible-wizard-and-operator-logging-independent-qa.md` | `/home/ra/almadial-lab/evidence/m1-07/20261109T144500Z-agy/` |

---

## 3. Cross-Module Integration Matrix

All cross-module interfaces were exercised and verified through dedicated integration tests in `internal/cli/m1_gate_integration_test.go`:

| Cross-Module Interface | Integration Contract | Test Method | Status | Evidence Reference |
|---|---|---|---|---|
| **CLI $\rightarrow$ Configuration** | Validates schema version 1, rejects unknown fields, supplies non-sensitive defaults | Unit & CLI tests | **PASS** | `TestWizardNoninteractive`, `TestM1GateIntegration_ConfigToPlan` |
| **Configuration $\rightarrow$ Preflight** | Passes bind/advertise IPs, ports, and credential references into probe evaluation | Functional integration | **PASS** | `TestM1GateIntegration_ConfigToPlan` |
| **Preflight $\rightarrow$ Plan** | Synthesizes immutable plan from host snapshot and explicit reviewed budget | Functional integration | **PASS** | `TestM1GateIntegration_ConfigToPlan` |
| **Artifacts $\rightarrow$ Manifest Identity** | Computes lowercase SHA-256 manifest digest; enforces CANDIDATE unauthorization | Cryptographic check | **PASS** | `TestM1GateIntegration_ManifestToPlanIdentity` |
| **Manifest Identity $\rightarrow$ Plan Freshness** | Plan embeds manifest digest; altered manifest invalidates plan digest | Digest comparison | **PASS** | `TestM1GateIntegration_ManifestToPlanIdentity` |
| **Plan $\rightarrow$ Wizard** | Renders human-readable plan, findings, intended changes, and blockers | PTY & UI test | **PASS** | `TestPresentationAndEquivalentPlans`, `TestM1GateIntegration_ZeroMutationPreparation` |
| **Plan $\rightarrow$ Action Graph Contract** | Plan stages and dependencies map deterministically to Action Graph structures | Model verification | **PASS** | `TestM1GateIntegration_GraphAndJournalContracts` |
| **Action Graph $\rightarrow$ Journal Identity** | Journal `Identity` binds Graph digest together with Host, Config, Manifest, and Plan | Identity hash check | **PASS** | `TestM1GateIntegration_GraphAndJournalContracts` |
| **Runner $\rightarrow$ Progress/Error Reporting** | `execution.Trace` adapts to `operatorui.Event` across all 10 ALM-13 error categories | Trace adapter test | **PASS** | `TestTraceAdapter`, `TestProgressSequenceAndTaxonomy` |
| **Execution Diagnostics $\rightarrow$ Operator Logging** | Redacted stdout/stderr fragments routed to structured JSON logs with 16 KiB / 1 MiB bounds | Logging stream check | **PASS** | `TestM1GateIntegration_LoggingAndRedaction` |
| **Journal $\rightarrow$ Resume/Reconciliation** | Interrupted/drifted actions reconcile to Pending (if idempotent & absent) or block on drift | Replay simulation | **PASS** | `TestM1GateIntegration_GraphAndJournalContracts` |
| **Approval $\rightarrow$ Future Mutation Boundary** | Transient approval decision binds reviewed config and plan; candidate manifest rejected | Validation check | **PASS** | `TestM1GateIntegration_ApprovalBoundaries` |

---

## 4. End-User Preparation Workflow

The end-user preparation workflow operates deterministically through the compiled CLI:
1. **Interactive Terminal Detection:** Requires `ioctl(TCGETS)` on both `stdin` and `stdout`. Nonterminal input or redirected stdout without `--unattended` exits with code 2.
2. **Keyboard-Only Prompts:** Plain text interaction without ANSI escape sequences, color dependencies, or full-screen takeover. Verified readable under narrow 32-column terminal wrapping.
3. **Fixed Prompt Order:** Target IP, advertised IP, optional hostname, SIP port, RTP port range, optional WebRTC, credential references (`database`, `ami`, `sip`, `admin`), and optional WSS/TLS parameters.
4. **IP-Only Installation:** Hostname is optional (`-` clears). WebRTC is optional (`no` omits WSS/TLS prompts and plan stages).
5. **Credential Safety:** Prompts accept source references only (`env:NAME` or `file:/path`), never raw secrets or passwords. Values are not resolved or written to disk during wizard execution.
6. **Preflight & Plan Presentation:** Actual findings (`PASS`, `WARNING`, `FAIL`, `NOT_VERIFIED`) are displayed. Blocking failures provide remediation instructions and report that installation cannot continue.
7. **Approval Refusal:** Candidate manifests and failing host prerequisites reject approval deterministically.
8. **Fail-Closed Handling:** Input overflow (>4,096 bytes), EOF, partial lines, `cancel`, `Ctrl+C` (`\x03`), and output write errors abort cleanly without approval.

---

## 5. Security and Authorization Architecture

- **Manifest Provenance:** The shipped stack manifest (`manifests/stack.lock.yaml`) is strictly verified. Its status is `candidate`, and `InstallationAuthorized()` returns `false`. The CLI cannot approve or authorize candidate manifests for mutation.
- **Identity Binding:** Approval binds effective configuration SHA-256 and immutable plan SHA-256. Any modification to configuration, material host facts, or plan content invalidates prior approval.
- **Operator Consent Boundary:** A SHA-256 digest identifies configuration or plan data; it does not authenticate an operator or establish consent. The future executor must revalidate authorization at the mutation boundary.
- **Exclusive Process Ownership:** `apply.lock` uses Linux `flock(LOCK_EX | LOCK_NB)` on a dedicated file in an owned `0700` directory. Contending processes fail fast with `ErrBusy`. Locks are released cleanly on normal exit, crash, or `SIGKILL`.
- **Diagnostic Logging Security:** Operator logs are written as structured JSON with permissions `0600` inside existing owned `0700` directories. Symlinks and existing files are rejected via component-by-component `openat(O_NOFOLLOW)`. Registered secrets are redacted before reaching any sink, including across split chunk and stream fragments within a combined record.

---

## 6. Crash Recovery and Execution Readiness

- **Action Graph Compatibility:** The accepted `execution.Action` model defines explicit preconditions, command parameters, timeout limits, output buffers, and postconditions.
- **Atomic Persistence Pipeline:** Journal updates follow a 7-stage commit sequence: create `.journal-<token>.tmp` (0600, O_EXCL, O_NOFOLLOW), write payload, check short-write, `f.Sync()`, close, `renameat`, and directory `fsync`.
- **Durable Intent:** `Session.Begin(id)` commits `Running` state durably before any mutating subprocess begins.
- **Postcondition Verification:** `Session.Finish` requires an `execution.Trace` and re-checks postconditions via trusted observers before transitioning an action to `Succeeded`. Exit code 0 is necessary but never sufficient.
- **Resume Reconciliation:** Interrupted runs re-inspect all actions. Only idempotent actions confirmed to be in a safe pre-action `Absent` state return to `Pending`. Succeeded-action drift or non-idempotent actions cascade to `ReconciliationRequired`, blocking automatic execution replay.

---

## 7. Deferred Executor Responsibilities

M1 establishes the framework and safety boundaries; it does **not** deliver the complete installation executor. The future executor (to be constructed in subsequent milestones) must enforce:
1. Re-validating approved manifest status (once promoted from CANDIDATE).
2. Verifying selected source artifacts and cryptographic signatures.
3. Validating effective configuration and resolving credential references at runtime.
4. Re-collecting and checking host preflight observations on the target server.
5. Re-validating immutable plan freshness against the live host.
6. Verifying operator consent at the mutation boundary.
7. Acquiring exclusive execution ownership (`apply.lock`).
8. Persisting durable action intent (`Session.Begin`).
9. Executing bounded subprocesses through `execution.Runner`.
10. Validating postconditions via stage observers before marking `Succeeded`.
11. Reconciling interrupted or drifted actions before resuming.

---

## 8. Regression Suite and Reproducibility

- **Go Test Suite:** 110 top-level passed, 544 pass events, 0 failures, 1 opt-in skip (`TestM0Artifacts`).
- **Go Race Detector:** 110 top-level passed, 544 pass events, clean execution.
- **Go Vet & Formatting:** `make fmt-check`, `go vet ./...`, `go mod verify`, `git diff --check` all **PASS**.
- **Python Suites:** 126 unit tests passed in 1.868s; scaffold, candidate stack, and config generation checks all **PASS**.
- **CLI Smoke Tests:** 17/17 cases passed (exit codes 0, 1, 2) without secret leaks.
- **PTY Journeys:** 11/11 executable terminal and nonterminal journeys passed.
- **Reproducible Linux amd64 Builds:** Two independent builds from separate source trees yielded byte-identical ELF binaries:
  - **SHA-256:** `892d00facc9b8406694a85cb774f15327c43bea01c91613ed959cf3a54e6835a` (8,299,244 bytes).

---

## 9. Protected Milestone Evidence

All audit artifacts and test logs are sealed in the dedicated milestone evidence directory:
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-gate/20261109T150200Z-agy/`
- **Permissions:** Mode `0700` directory, mode `0600` files.
- **Master SHA256SUMS Digest:** `ba509e0bc303a5e2d323c93d6a52f89d4277695c05c708fd61b2a7b92874304e`
- **Contained Files:**
  - `m1-gate-evidence-ledger.json`: Master milestone gate summary.
  - `m1-gate-qa-provenance.json`: Verified inventory of ALM-9 through ALM-15 tasks, commits, and reports.
  - `m1-gate-integration-matrix.json`: Matrix of the 12 cross-module interface contracts and verification results.
  - `m1-gate-test-inventory.json`: Comprehensive Go, Python, PTY, and CLI test execution counts.
  - `go-test.json`: Full JSON event output of repository Go tests (110 top-level / 544 pass events).
  - `go-race.json`: Full JSON event output under the Go race detector.
  - `cli-smoke.json`: Results of 17 CLI smoke tests.
  - `pty-evidence/`: Output transcripts for all 11 terminal/nonterminal journeys.
  - `reproducible-builds.json`: Metadata and SHA-256 digests for reproducible builds.

---

## 10. Gate Verdict and Handoff Recommendation

### Final Verdict: `ALMADIAL_M1_INTEGRATION_GATE_ACCEPTED`

The seven accepted M1 components form a coherent, deterministic, and verifiable Go installer framework. All canonical milestone integration criteria have been met.

### Next Recommended Assignment: Milestone M2 Kickoff
- **Target Task:** ALM-16 / M2-01 — *Provision approved repositories and RPM dependencies* (`g3ibghcq6p3y55o1bczk5a7n`).
- **Current Status:** `to-do` (Backlog, unassigned).
- **Recommended AI Worker:** Codex (implementation owner).
- **Recommended Model:** Pinned Codex implementation configuration.
- **Recommended Role:** Systems & Packaging Go Engineer.
- **Scope & Boundaries:** Provision verified repository configuration files and dependencies for AlmaLinux 10 on a disposable lab VM; do not touch production systems; adhere strictly to the accepted M1 runner, locking, and journal contracts.
