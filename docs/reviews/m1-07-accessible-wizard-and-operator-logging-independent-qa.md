# ALM-15 / M1-07 Independent Terminal Wizard, Approval Integrity, and Operator Logging QA Report

- **Task ID:** `r6qm3nzbahxpvcw8ejk71s0q` (ALM-15 / M1-07)
- **Task Title:** Deliver accessible wizard and redacted operator logging
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-09T14:48:00Z
- **Repository Baseline Commit:** `c32a88a24613330f7dd9576deebd00be55f0b463` (Accepted ALM-14)
- **Accepted Commit:** `89fd1d7`
- **Final Verdict:** `ALMADIAL_M1_07_ACCEPTED`

---

## 1. Governance and Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No reassignment or self-assignment occurred.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Evidence Comment:** Verified comment `n53df86duor3011f7ucywpq6` matches recorded implementation details and digests.
- **Dependency Graph:**
  - Upstream ALM-10 (`bcyodkgz47sak45stno5um0b`, M1-02): Verified status `done` (Accepted).
  - Upstream ALM-11 (`xfy4pagbrmppq3yijr7nz7d7`, M1-03): Verified status `done` (Accepted).
  - Upstream ALM-14 (`q2kgwrrzpn4zkzgvj08y6g1h`, M1-06): Verified status `done` (Accepted).
  - Relation ALM-15 $\rightarrow$ ALM-16 (`vpmvdb97n6ptzwb56k0ll5v9`): Verified intact (`blocks`).
  - Downstream ALM-16 (`g3ibghcq6p3y55o1bczk5a7n`, M2-01): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Execution Boundary:** ALM-15 delivers the accessible terminal interface, preflight and immutable plan presentation, explicit approval decisions, deterministic progress reporting, and secret-safe operator logging.
- **Zero Host Mutation / Zero Execution:** The wizard executes zero installation stages, installs zero RPM packages, modifies zero host configuration, creates zero journals, acquires zero execution locks, and initiates zero remote connections. All positive approval and eligible host tests use explicitly synthetic fixtures; the shipped candidate stack manifest remains unauthorized.

---

## 2. Protected Implementation Evidence Verification

### 2.1 Core Evidence Ledger
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m1-07/20261107T085319Z-codex/`
- **Core Ledger:** `core/SHA256SUMS`
  - Computed SHA-256: `d4ba28e421b263747a0950f22097d2bf41464c1d2412236f98c0ca3fdef5a09f` (exact match)
  - All 77 indexed artifacts verified with status `OK`.
- **Source Manifest:** `core/source-files.json`
  - Computed SHA-256: `9f39bb7279fbb13c1b4c07d7f9781e77e3f24c9384cf41136eb8f645f8cf004b` (exact match)
  - All 14 task-owned files verified byte-for-byte against the repository working tree.
- **Handoff Receipt:** `handoff/kaneo-readback.json` independently verified.

---

## 3. Terminal Wizard Inspection & PTY Verification (`operatorui/wizard.go`, `cli/wizard_linux.go`)

### 3.1 Interface Usability and Accessibility
- **Terminal Detection:** Enforces Linux terminal detection via `ioctl(TCGETS)` on both `stdin` and `stdout`. Nonterminal input or redirected stdout without `--unattended` fails with code 2.
- **No Advanced TUI / No ANSI Escape Codes:** Standard line-based interaction without ANSI cursor positioning, clearing, or animations. Output is cleanly readable in narrow 32-column SSH terminals.
- **Prompt Sequence:** Ordered deterministically:
  1. Target server bind IP
  2. Advertised IP (required for wildcard bind)
  3. Optional hostname (`-` clears optional text; IP-only supported)
  4. SIP UDP port
  5. RTP first UDP port
  6. RTP last UDP port
  7. Enable optional WebRTC (`yes`/`no`)
  8. Credentials (`database`, `ami`, `sip`, `admin`) — credential references only (`env:NAME` or `file:/path`)
  9. WebRTC parameters (if enabled): WSS TCP port, TLS certificate file reference, TLS private-key file reference
- **Defaults & Correction:** Enter keeps the displayed non-sensitive default from `config.Defaults`. Syntax or validation errors allow immediate in-place correction without losing prior answers.
- **Fail-Closed Boundaries:**
  - Input lines exceeding the 4,096-byte reader bound fail closed.
  - EOF or partial final lines fail closed (`ErrInput`).
  - `cancel`, `Ctrl+C` (`\x03`), or context cancellation immediately abort with `ErrCanceled` (exit code 1).
  - Terminal write failures fail closed (`ErrOutput`).

### 3.2 Executable PTY Journeys
All 11 journeys in `scripts/test_terminal_wizard.py` were reproduced against the compiled executable binary:
1. `loaded-review` (exit 0, PASS): Displays loaded valid configuration and candidate plan review.
2. `narrow-candidate-rejection` (exit 1, PASS): 32-column terminal, candidate manifest correctly rejects approval.
3. `fresh-IP-only` (exit 0, PASS): Fresh interactive session without hostname or WebRTC.
4. `invalid-correction` (exit 0, PASS): Invalid IP corrected in-place; valid plan rendered.
5. `EOF` (exit 1, PASS): Terminal stream close fails closed with `input ended or failed`.
6. `explicit-cancel` (exit 1, PASS): Prompt typing `cancel` aborts without approval.
7. `Ctrl-C` (exit 1, PASS): Interrupt character aborts immediately without approval.
8. `redirected-output` (exit 2, PASS): Non-terminal stdout without `--unattended` rejected.
9. `pipe-rejected` (exit 2, PASS): Pipe input without `--unattended` rejected.
10. `unattended-review` (exit 0, PASS): Noninteractive review reports `approval NOT REQUESTED`.
11. `unattended-missing-approval` (exit 1, PASS): Unattended without approval flags rejected.

---

## 4. Noninteractive & Unattended Workflow

- **Explicit Intent:** Non-terminal execution strictly requires `--unattended --config FILE`. Stdin is never used as configuration.
- **Validation Order:**
  1. Config file parsing and validation via accepted `config.Load` and `config.Validate`.
  2. Manifest loading via `artifacts.Load`.
  3. Preflight requirements bounded parsing (64 KiB, disallowing unknown fields).
  4. Local host preflight collection and plan synthesis.
  5. Unattended approval validation via `config.ValidateUnattended`.
- **Exit Codes:** Preserves accepted CLI convention:
  - `0`: Successful review or valid synthetic approval decision.
  - `1`: Operational failure, cancellation, rejected approval, or logging failure.
  - `2`: Invalid CLI usage, bad arguments, or missing nonterminal configuration.

---

## 5. Plan Review & Explicit Operator Approval Integrity

### 5.1 Three Distinct Conceptual Boundaries
The implementation strictly preserves the separation between:
1. **Reviewed Plan:** The immutable plan output generated from effective configuration, preflight snapshot, requirements, and manifest.
2. **Operator Approval:** The deliberate human affirmative decision bound to exact configuration and plan SHA-256 digests.
3. **Installation Authorization:** The full set of host eligibility, manifest qualification, and security preconditions required before mutation.

### 5.2 Approval Integrity Verification
All 14 mandatory approval test scenarios pass:
- Explicit affirmative input (`yes`, `y`, case-insensitive) records transient approval.
- Negative input (`no`, `n`), empty responses, EOF, `Ctrl+C`, and `cancel` refuse approval.
- Any change to effective configuration, configuration digest, plan content, or plan digest invalidates prior approval.
- Failed mandatory preflight findings or permissive SELinux refuse authorization.
- CANDIDATE manifest always denies installation authorization (`InstallationAuthorized() == false`).
- Zero bypass: The CLI exposes no flag, override, or parameter to force approval of unauthorized or candidate manifests. Positive approval is testable only via internal package test seams using synthetic authorized manifests.
- Transient decision only: No journal file, execution lock, or host state is created upon approval.

### 5.3 Immutable Plan Accessors
The two additions to `internal/plan/plan.go`:
```go
func (p Plan) ConfigSHA256() string   { d, _ := p.doc(); return d.ConfigSHA256 }
func (p Plan) ManifestSHA256() string { d, _ := p.doc(); return d.ManifestSHA256 }
```
were verified to read from a defensive unmarshaled document copy without mutating plan state or affecting plan SHA-256 digest calculations.

---

## 6. Progress Reporting & Error Taxonomy (`operatorui/progress.go`)

- **Trace Adaptation:** `EventFromTrace` cleanly adapts accepted ALM-13 `execution.Trace` objects into deterministic UI progress events.
- **Ten Error Categories:** All ten accepted ALM-13 categories (`contract`, `precondition`, `process_start`, `nonzero_exit`, `timeout`, `canceled`, `output_limit`, `postcondition`, `dependency`, `internal`) map to distinct actionable next-step guidance messages without leaking sensitive diagnostics.
- **Deterministic Rendering:** No clocks, random IDs, or non-deterministic maps affect event output.
- **Event Validation:** Missing action identifiers on action events, contradictory failure/completion events, or invalid event types are rejected before reaching any output sink.

---

## 7. Operator Logging & Private Storage (`operatorlog/`)

### 7.1 Structured JSON & Stream Routing
- Records emit stable JSON lines with `level`, `stage`, `action`, `code`, `message`, and `next`.
- `INFO` records route exclusively to `stdout`.
- `WARNING` and `ERROR` records route exclusively to `stderr`.
- Optional file sink receives identical sanitized records.

### 7.2 Bounds & Sticky Poisoning
- Textual fields bounded to 16 KiB (`MaxMessage = 16384`) before masking.
- Cumulative output bounded to 1 MiB (`MaxOutput = 1048576`) per logger across all sinks.
- Sticky failure: Sinks that fail, short-write, or reach the 1 MiB limit permanently poison the logger (`ErrLimit`, `ErrWrite`), preventing silent partial diagnostic loss.

### 7.3 Restrictive Private Log Files (`file_linux.go`)
- Absolute, clean path required.
- Component-by-component traversal from `/` using `syscall.Openat` with `O_NOFOLLOW | O_DIRECTORY | O_CLOEXEC`.
- Parent directory must exist, be owned by the effective user, and have permissions `0700`.
- New log file created with `O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC` and mode `0600`.
- Symlink components (parent or file) and existing destination files are strictly rejected.
- Verified in 5 targeted smoke tests:
  1. Valid `0700` parent creates `0600` file: PASS.
  2. Existing file overwrite attempt rejected: PASS.
  3. Public `0755` parent directory rejected: PASS.
  4. Symlink directory parent rejected: PASS.
  5. Zero journal, lock, or state files created in the logging directory: PASS.

---

## 8. Registered-Secret & Cross-Stream Redaction

- **Secret Registration:** Up to 64 non-empty patterns, max 4,096 bytes per pattern, max 64 KiB total registration.
- **Masking Algorithm:** Matches tokens against original bytes before control character stripping or formatting.
- **Overlapping & Repeated Patterns:** Handled cleanly with contiguous `[REDACTED]` replacement tokens.
- **Truncation & Boundary Masking:** Partial token suffixes at the 16 KiB message boundary are masked before truncation indicators are appended.
- **Combined-Record Cross-Stream Redaction:** When ordered fragments (e.g. stdout followed by stderr) are passed within a single record via `Redactor.Text(fragments...)`, secrets spanning chunk boundaries are joined and redacted without leaks.
- **Defined Scope Boundary:** Independent log records are not correlated; callers must register all known secrets prior to emitting diagnostics.

---

## 9. Sandbox Test Investigation

- **Developer Finding:** Initial sandboxed Go test runs failed with:
  `panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted` in `internal/artifacts/TestFetch`.
- **Independent Verification:**
  - The failure was reproduced from developer logs and verified to stem strictly from container/sandbox namespace restrictions prohibiting loopback socket binding (`[::1]:0`).
  - The tests use standard library `httptest.NewServer` bound to loopback.
  - Zero external internet access is required.
  - In an authorized test environment with loopback access, the full Go test suite and race detector pass 100% with zero failures.
  - No test code, TLS validation, or production behavior was weakened to bypass sandbox limitations.

---

## 10. Independent Verification & Reproducibility Results

| Suite / Check | Command / Procedure | Scope / Count | Status | Notes |
|---|---|---|---|---|
| Go Formatting | `make fmt-check` | `cmd`, `internal` | **PASS** | Zero formatting diffs |
| Go Vet | `go vet ./...` | All packages | **PASS** | Clean |
| Go Modules | `go mod verify` / `tidy -diff` | Direct / stdlib | **PASS** | Zero new dependencies |
| Git Hygiene | `git diff --check` | 14 task files | **PASS** | No whitespace/conflict markers |
| Go Test Suite | `go test -count=1 -json ./...` | 104 top-level / 530 events | **PASS** | 0 failures, 1 opt-in skip (`TestM0Artifacts`) |
| Go Race Detector | `go test -race -count=1 -json ./...` | 104 top-level / 530 events | **PASS** | 0 failures, 1 opt-in skip (`TestM0Artifacts`) |
| Python Unit Tests | `python3 -m unittest discover` | 126 tests | **PASS** | Ran in 1.844s, 0 failures |
| Scaffold & Stack Checks | `validate_scaffold.py` / `verify_candidate_stack.py` | Scaffold, candidate stack | **PASS** | Clean verification |
| Config Generation | `verify_config_generation.py` | Config templates | **PASS** | Clean verification |
| PTY Journeys | `test_terminal_wizard.py` | 11 executable journeys | **PASS** | Narrow terminal, EOF, Ctrl+C, cancel |
| CLI Smoke Checks | `cli-smoke.json` | 17 CLI invocations | **PASS** | Exit codes 0, 1, 2; no secret leak |
| Private Log Smoke | `private-log-smoke.json` | 5 permission checks | **PASS** | 0700 dir, 0600 file, symlink rejection |
| Reproducible Builds | `scripts/build_go.py` across two dirs | SHA-256 byte comparison | **PASS** | Byte-identical: `3213f3631068449095722c36cab98df990938ee1f06e721679d2c4d0a9077876` |

---

## 11. Protected AGY QA Evidence Directory

- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-07/20261109T144500Z-agy/`
- **Permissions:** Mode `0700` directory, mode `0600` files.
- **Master SHA256SUMS Digest:** `bd3c37a558b1ad0a52b0912e68c9cdbc835bee2d5d04ea2a59275f107e86b221`
- **Key Evidence Artifacts:**
  - `qa-evidence-ledger.json`: Master QA session ledger.
  - `qa-go-and-python-suites.json`: Full Go and Python test run records.
  - `qa-reproducible-builds-and-smokes.json`: Build hashes, CLI smoke, and private log permission results.
  - `qa-terminal-wizard-and-approval-tests.json`: PTY journeys, terminal, approval, progress, and logging results.
  - `qa-historical-ledgers.json`: Verification of developer evidence `SHA256SUMS` and `source-files.json`.
  - `pty-evidence/`: Complete captured transcripts for all 11 terminal/nonterminal journeys.

---

## 12. Final Kaneo Acceptance and Milestone Transition

All canonical ALM-15 acceptance criteria are met without requiring another development cycle.
1. Codex remains preserved as the implementation owner (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`).
2. Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`) submits this independent QA review.
3. Task ALM-15 status transitions to `done` (Accepted).
4. All upstream dependencies (ALM-10, ALM-11, ALM-14) remain `done`.
5. Downstream task ALM-16 remains `to-do` (Backlog).
6. With ALM-9 through ALM-15 now independently accepted, the M1 milestone components are complete, paving the way for the separate M1 milestone integration gate review prior to M2 package provisioning.
