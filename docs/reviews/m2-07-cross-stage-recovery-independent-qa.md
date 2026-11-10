# ALM-22 / M2-07 Independent Cross-Stage Recovery QA Report

- **Task ID:** `g1jf2c808sylx8twxhzas9oc` (ALM-22 / M2-07)
- **Task Title:** Validate safe recovery across native backend stages
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Milestone:** M2 — Native Backend Installation
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — *strictly preserved*
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T23:25:00+08:00 (2026-11-10T15:25:00Z)
- **Repository Baseline Commit:** `86685e7a594ee2863cbd0357dd948803651e0b6e` (Accepted ALM-21 commit)
- **Current HEAD:** `86685e7a594ee2863cbd0357dd948803651e0b6e`
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-07/20261110T135237Z-alm22/` (Master ledger SHA-256: `ba50b10cc59f24bcacff85685cc8acb7b499b886eb02800401b0faac9ba222bf`, 169 entries verified)
- **Independent QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-07/qa-20261110T152351Z-alm22/` (Master ledger SHA-256: `7eaa05966378c6ae6c7ae835655cd86a58ffa911ecb4b0f8fdb66194f7db69cb`, 9 entries verified)
- **Final Verdict:** `ALMADIAL_M2_07_ACCEPTED`

---

## 1. Executive Summary & Independent QA Matrix

ALM-22 validates the integration, recovery, idempotency, and dependency contracts across all six Accepted backend installation stages (S01 through S06). Independent review confirms that the existing stage implementations enforce consistent dependency ordering, prevent downstream readiness on incomplete prerequisites, reconcile interrupted complete states without action replay, reject stale or conflicting identities, and maintain genuine NO_OP behavior on the installed laboratory host while strictly preserving historical baselines.

### 1.1 Decision Matrix

| Criterion | Result | Evidence Reference |
|---|---|---|
| **S01–S06 dependencies** | **PASS** | `cross-stage-dependencies-matrix.json`, `TestBackendCompleteChain` |
| **Complete-state reconciliation** | **PASS** | `copied-journal-and-identity-recovery.json`, `TestBackendCopiedJournalRecovery/interrupted` |
| **Incomplete upstream blocking** | **PASS** | `TestBackendUpstreamBlocksDependents` (subtests S01–S05) |
| **Stale identity rejection** | **PASS** | `TestBackendMaterialIdentityPropagates`, `TestBackendCopiedJournalRecovery` |
| **Lock contention and scope** | **PASS** | `lock-contention-and-scope.json`, `TestBackendCrossStageExecutionLock` |
| **Actual-host NO_OP** | **PASS** | `actual-host-noop-plan.json` (Stage SHA-256 `142ce473799409602dbb3008e910f0920e9b6815a217fbb6a8f3fb6fbb4ea768`, 0 actions) |
| **Previous baseline preservation** | **PASS** | `host-baseline-preservation.json` (725 RPMs, schema 1748, catalog SHA, 2 seed phones SHA, SELinux Enforcing, 0 AVCs) |
| **Regression tests** | **PASS** | `regressions-summary.json` (Go 205 tests/1,337 events, race clean, Python 181 tests, 11 PTY journeys) |
| **Evidence integrity** | **PASS** | `developer-ledger-verification.txt` (169 entries verified against ledger `ba50b10c...`) |

---

## 2. Governance and Task Verification

### 2.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-22 (`g1jf2c808sylx8twxhzas9oc`), number 22 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No reassignment occurred.
- **Reviewer Identity:** Verified via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Initial Task Status:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Developer Evidence Comment:** Verified comment `rkpr4rpt9n8fbr598sw6k8lg` posted by Codex on 2026-11-10T14:04:12.806Z. All recorded test counts, digests, and observations matched independent findings.
- **Upstream & Downstream Dependencies:**
  - Upstream ALM-16 through ALM-21: Confirmed all six tasks are in `done` (Accepted) status with Codex ownership intact.
  - Relation ALM-21 $\rightarrow$ ALM-22 (`qcq551kz5z3jidx7jo99lp94`): Confirmed intact (`blocks`).
  - Relations ALM-22 $\rightarrow$ ALM-23 (`fbb4xu4j12bane7meg3kv2rk`) and ALM-22 $\rightarrow$ ALM-25 (`l3nwww6nobaxo7fnsp8ml6gq`): Confirmed intact (`blocks`); downstream tasks ALM-23 and ALM-25 remain `to-do` and unstarted (`userId: null`).

---

## 3. Cross-Stage Dependency and Recovery Verification

### 3.1 S01–S06 Dependency Ordering and Gate Enforcement
Independent execution of `TestBackendCompleteChain` and `TestBackendUpstreamBlocksDependents` verified:
- **Dependency Flow:** S01 (RPMs) $\rightarrow$ S02 (Storage/SELinux) $\rightarrow$ S03 (MariaDB/Schema) $\rightarrow$ S04 (Asterisk) $\rightarrow$ S05 (VICIdial App) $\rightarrow$ S06 (Runtime Services).
- **Postcondition Enforcement:** Downstream preparation (`Prepare`) strictly checks that upstream postconditions are met. Injected defects demonstrate hard blocking:
  - Missing RPM in S01: Blocks S03, S05, and S06.
  - Missing storage directory in S02: Blocks S03, S05, and S06.
  - Incomplete schema / table count reduction in S03: Blocks S05 and S06.
  - Unloaded / failing native module in S04: Blocks S05 and S06.
  - Absent application configuration in S05: Blocks S06.
- In each test case, observation digests are updated to reflect the injected defect; downstream stages reject the incomplete prerequisites directly via postcondition verification rather than merely failing on stale digest hashes.

### 3.2 Complete-State Reconciliation Without Replay
Independent execution of `TestBackendCopiedJournalRecovery/interrupted` verified:
- A fixture journal with a complete state where the final outcome was recorded as `RUNNING` (simulating process termination after effects completed) reconciles cleanly to `SUCCEEDED`.
- Zero runner calls (`calls: 0`), zero traces emitted, and zero mutations to the observed state.
- Existing completed effects are never replayed.

### 3.3 Stale Identity Rejection
Independent execution of `TestBackendMaterialIdentityPropagates` and `TestBackendCopiedJournalRecovery` (subtests `config`, `manifest`, `plan`, `host`, `graph`, `action_metadata`) verified:
- Any divergence in configuration, manifest, plan, target host, execution graph, or action metadata causes immediate failure with `journal.ErrIdentity`.
- The rejected journal file on disk is preserved unchanged without overwrite or corruption.
- Stale authorization tokens cannot execute plans derived from modified inputs.

---

## 4. Installation Lock Scope Assessment

### 4.1 Verification Findings
- **Lock Mechanism:** Exclusive non-blocking flock (`syscall.Flock(apply.lock, syscall.LOCK_EX|syscall.LOCK_NB)`).
- **Contention Rejection:** `TestBackendCrossStageExecutionLock` confirms that when an installation directory lock is held, mutating operations in both S03 (`ExecuteDatabaseLab`) and S06 (`ExecuteRuntimeLab`) fail immediately with `journal.ErrBusy` before any runner calls are initiated.
- **Lock Scope:** The lock file `apply.lock` is scoped to the specific installation state directory passed to the stage CLI (e.g. `/var/lib/almadial/...`).
- **Accepted Design Boundary:** The lock is directory-scoped, not host-global. If an operator were to concurrently target two different state directories on the same host, flock on separate lock files would not prevent concurrent execution. This matches the accepted M1/M2 architectural contract (which specifies a single canonical state directory per installation session) and is classified as an accepted design boundary, not an ALM-22 defect.

---

## 5. Actual Host NO_OP and Baseline Preservation

### 5.1 Actual Host NO_OP Verification
Independent read-only assessment was performed live on authorized VM `almadial-alm16-rpm` (`10.160.90.233`, machine ID `0fffa9a4683f35b696f0b6ee7330734b`):
- `almadial-runtime inspect`: Exit code 0, snapshot fully populated.
- `almadial-runtime plan`: Exit code 0, returned empty graph `[]` (0 actions).
- **Stage SHA-256:** `142ce473799409602dbb3008e910f0920e9b6815a217fbb6a8f3fb6fbb4ea768` (exact match to developer witness).
- Confirms genuine NO_OP: zero package transactions, zero database imports, zero Asterisk builds, zero file modifications, and zero service restarts.

### 5.2 Baseline Preservation (100% Bit-for-Bit Match)
All previously accepted baselines were confirmed intact live on the VM:
- **RPM Inventory:** 725 packages, SHA-256 `bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`.
- **Database Engine & Version:** MariaDB `10.11.18-MariaDB`.
- **Schema Version:** `1748` (`system_settings.db_schema_version`).
- **Database Catalog SHA-256:** `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` (345 tables, 1,011 indexes).
- **Legacy Seed Phones SHA-256:** `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04` (`callin` and `gs102` across all 107 columns).
- **Storage Directories:** `/var/spool/asterisk/monitor` and `/var/spool/asterisk/monitorDONE` (both mode `0750`, owned by `asterisk:asterisk`).
- **SELinux:** `Enforcing`. Audit log inspection since boot confirmed 0 AVC denials (`ausearch -m avc -ts boot` returned exit 1, no matches).
- **Services:** `mariadb`, `asterisk`, `httpd`, `php-fpm`, `crond`, and `almadial-s06-fastagi` are all `active` and `enabled`. FastAGI process pool contains exactly 1 parent + 5 worker children under `asterisk:asterisk`. Deferred updater and listener units remain inactive and disabled.
- **Telephony State:** 0 active channels, 0 active calls, 0 calls processed, and 0 public SIP listeners.

---

## 6. Test Regressions and Reproducible Builds

### 6.1 Go 1.27.3 Regressions
- `go test -count=1 ./internal/provision -run '^TestBackend'`: 5 top-level tests, 19 passing events (PASS).
- `go test ./...`: 205 top-level tests, 1,337 passing events (PASS).
- `go test -race ./...`: 205 top-level tests, 1,337 passing events, zero data races (PASS).
- `go vet ./...`: Clean (exit 0).
- `go fmt ./...` and `make fmt-check`: Clean (exit 0).
- `go mod verify`: Clean (`all modules verified`).
- `go mod tidy -diff`: Clean (exit 0).

### 6.2 Python Regressions
- `python3 scripts/validate_scaffold.py`: PASS (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: 181 tests passed (exit 0).
- `python3 scripts/verify_candidate_stack.py`: PASS.
- `python3 scripts/verify_config_generation.py`: PASS.
- `scripts/test_terminal_wizard.py`: 11 executable terminal/nonterminal journeys PASS.
- `git diff --check`: Clean (no whitespace or formatting errors).

### 6.3 Reproducible Linux amd64 Builds
Independent builds verified byte-for-byte reproducibility across all seven command targets:
- `cmd/almadial`: SHA-256 `41f842fc3bae3cf2a2ca943929384c71af9d5190b4a3bc296361480bd56ccea6` (byte-identical).
- `cmd/almadial-runtime`: SHA-256 `c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a` (byte-identical).
- `cmd/almadial-rpm`, `cmd/almadial-system`, `cmd/almadial-db`, `cmd/almadial-asterisk`, `cmd/almadial-vicidial`: all byte-identical across isolated build roots.

---

## 7. Accepted Limitations and NOT_RUN Items

1. **Top-Level Orchestrator:** S01–S06 integration is provided via stage qualification CLIs and nested input structures. A unified, top-level end-to-end installer command is deferred to future milestone planning.
2. **Lock Scope:** The execution lock is directory-scoped to the designated state directory rather than host-global. This satisfies the canonical requirement for single-installation safety.
3. **Nondestructive Evaluation:** Actual VM verification was conducted via read-only inspection and pure planning (NO_OP). Destructive failure injection is verified through unit tests and historical stage interruption records, preserving the lab VM.
4. **NOT_RUN:** Hosted CI, outbound telephony calling, live agent WebRTC media sessions, and manifest promotion to production remain NOT_RUN and deferred.

---

## 8. Final Independent Verdict

All canonical requirements and acceptance criteria for **ALM-22 / M2-07** have been satisfied:
- S01 through S06 dependency ordering is verified and enforced.
- Complete-state reconciliation avoids action replay.
- Incomplete upstream prerequisites block dependent stage readiness.
- Stale identities and conflicting execution ownership are rejected.
- Lock scope is verified and classified.
- Actual host satisfies all backend stages with a deterministic NO_OP plan.
- All historical baselines remain 100% intact.
- Full regression suites and reproducible builds pass with zero defects.

**Final Verdict:** `ALMADIAL_M2_07_ACCEPTED`

---

## 9. Handoff & Milestone Status

- **Task Status:** Update Kaneo task **ALM-22** (`g1jf2c808sylx8twxhzas9oc`) to **Accepted** (`done`, column `xs5usmxtzc3m5ylx0bnborhn`).
- **Task Ownership:** Retain Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) as the implementation owner.
- **Milestone Status:** All seven work packages in Milestone M2 (ALM-16 through ALM-22) are now Accepted!
- **Next Assignment:** Formal M2 Milestone Gate Closure Review before initiating Milestone M3.
