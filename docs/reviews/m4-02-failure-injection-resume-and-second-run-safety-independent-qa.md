# AlmaDial — ALM-30 / M4-02: Independent Failure Injection, Resume and Second-Run Safety QA Report

## Acceptance Status and Governance

- **Task ID:** `lcy5mrwd89v2hahs1qzn4uow`
- **Canonical Work Package:** ALM-30 / M4-02
- **Task Title:** Qualify failure injection, resume and second-run safety
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted Baseline Commit:** `d2805e4447aad47db91c48d3eb0e5456c22a118f`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Evaluated Target VM:** `almadial-alm29-clean-d` (Machine ID `4c7b492940923838a0700adba65ed3eb`, AlmaLinux 10.2, 731 RPMs, snapshot `alm29-pristine`)
- **Developer Evidence Comment:** `xhsy6riajj3hv0t3g3bu6s5l`
- **Developer Evidence Ledger:** `/home/ra/almadial-lab/evidence/m4-02/20261112T155041Z-alm30/SHA256SUMS` (`0c581d6bbdcbe465a4fe707946407722fef561e942a10b73c7ade5edb1e49c57`, 71 files, 100% verified)
- **Independent QA Ledger:** `/home/ra/almadial-lab/evidence/m4-02/qa-20261113T010500Z-alm30/SHA256SUMS` (`f09f60617babee2190f61991e02a0badaada21947884292613dee060717c5386`, 9 files, 100% verified)
- **Developer Verdict:**
  ```text
  ALMADIAL_M4_02_RECOVERY_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M4_02_RECOVERY_ACCEPTED
  ```

---

## 1. Independent QA Acceptance Matrix

| # | Acceptance Criterion | Result | Detailed Evaluation & Evidence |
|---|---|:---:|---|
| 1 | **Canonical ALM-30 Scope** | **PASS** | Bounded strictly to failure injection, resume, concurrency locking, plan approval, and second-run safety qualification within the native single-server installer. No production mutation, cluster expansion, or external scope creep. [implementation-report.md](file:///home/ra/almadial-lab/evidence/m4-02/20261112T155041Z-alm30/implementation-report.md) |
| 2 | **Authoritative Stage Mapping** | **PASS** | Source-verified directly from `internal/provision/install_linux.go` (`installStages`): S01 Packages, S02 Identities/Storage/SELinux, S03 MariaDB, S04 Asterisk, S05 VICIdial, S06 Runtime, S07 Web, S08 TLS, S09 Firewall, S10 VICIphone/PJSIP, S11 Explicit functional qualification harness, S12 Read-only diagnostics. Resolves prior narrative ambiguity without rewriting ALM-29. |
| 3 | **20 Deterministic Fixtures** | **PASS** | 20/20 before/after-effect fixtures in `TestInstallationBoundaryInjection` passed on Go 1.27.3. Completed effects never replayed (`effects == 1`); original approved inputs strictly preserved. |
| 4 | **S01–S09 Safe Rejection** | **PASS (Safely Blocked)** | Uncertain historical completions on copied native journals correctly refused automatic resumption (`ErrReconcile`, persisted state `reconciliation-required`). Rejection preserves existing host state; zero provisioning executor invocations occurred. |
| 5 | **S10 Journal Reconciliation** | **PASS (Reconciliation Succeeded)** | Interrupted S10 copied journal reconciled actual satisfied native postconditions without mutating actions or executor calls. Repeat resume verified safe and idempotent. |
| 6 | **Concurrent Execution Rejection** | **PASS** | Synchronously held `apply.lock` in `TestInstallationLockProcess` and live VM CLI invocations rejected competing `install` and `resume` processes before mutation (exit code 1, stderr `almadial installation: another process owns exclusive execution`). Lock ownership safely acquired after release. |
| 7 | **Approval and Identity Validation** | **PASS** | Missing `--approve-sha256` returned exit code 2; mismatched plan SHA-256 returned exit code 1. Valid approved plan executed cleanly with exit code 0 (`no_op: true`). Host, manifest, plan, and journal identity checks enforced before mutations. |
| 8 | **Repeat Installation NO_OP** | **PASS** | Clean installed system returned empty plan (`actions: []`, `no_op: true`) and zero-mutation repeat execution (exit code 0, `no_op: true`). No package transactions, compiler builds, or service restarts invoked. |
| 9 | **102-Item Preservation Contract** | **PASS** | 102 managed configuration files, journals, receipts, TLS credentials, cron, and audio recordings strictly preserved bit-for-bit, including file permissions (modes), UIDs, and GIDs. 731 packages, effective nftables ruleset, SELinux mode, and systemd service invocation identities remained unchanged. |
| 10 | **Database Audit-Write Attribution** | **PASS** | Controlled inspection across 531 database column/catalog/grant/recording checks confirmed 529 checks strictly identical. Exactly 2 changed fields (`vicidial_users.last_login_date` and `vicidial_users.modify_stamp`) conclusively attributed to upstream VICIdial `functions.php` admin login authentication during S07/S08 health checks, not installer provisioning replay. |
| 11 | **Functional Stack Integrity** | **PASS** | MariaDB (schema 1748), Asterisk 22.6.0, FastAGI (parent + 5 workers), Apache/PHP 8.2, PJSIP, loopback HTTPS/WSS, and S09 firewall isolation remain active and healthy. Read-only diagnostics (`status` and `doctor`) returned exit code 10, correctly reporting the non-blocking 14-day lab certificate expiry warning policy. |
| 12 | **Mandatory Regression Suite** | **PASS** | Full suite passed: 1,456 Go test events (0 failures, 2 deliberate native skips), 0 data races under `go test -race ./...`, 222 Python unit tests, 14 CLI smoke tests, 11 terminal wizard journeys, 12 reproducible binary targets built twice with bit-for-bit hash agreement. Hosted CI formally `NOT_RUN`. |
| 13 | **Evidence Integrity** | **PASS** | All 71 developer evidence files under `/home/ra/almadial-lab/evidence/m4-02/20261112T155041Z-alm30/` independently verified against `SHA256SUMS` (`0c581d6bbdcbe465a4fe707946407722fef561e942a10b73c7ade5edb1e49c57`). Independent QA ledger with 9 verified files recorded under `/home/ra/almadial-lab/evidence/m4-02/qa-20261113T010500Z-alm30/` (`f09f60617babee2190f61991e02a0badaada21947884292613dee060717c5386`). |

---

## 2. Authoritative Stage Crosswalk

The authoritative stage mapping is codified in `internal/provision/install_linux.go` (`installStages`). Each stage establishes an explicit foundation for subsequent actions:

| Stage ID | Responsibility | Mandatory Foundation | Decisive Postcondition Observation |
|---|---|---|---|
| **S01** | Packages and repositories | Approved host, sources, and package plan | Pinned RPM inventory (731 packages) |
| **S02** | Identities, storage, and SELinux | S01 | Asterisk/VICIdial users/groups, directories, SELinux file labels |
| **S03** | MariaDB and VICIdial schema | S02 | MariaDB service, schema 1748, tables, users, and grants |
| **S04** | Asterisk and native modules | S03 and package/build inputs | Asterisk 22.6.0 service, compiled native modules, PJSIP stack |
| **S05** | VICIdial application | S04 and database | Pinned VICIdial code, generated configuration files |
| **S06** | FastAGI and deferred scheduling | S05 | FastAGI parent plus 5 workers active, inactive cron contract |
| **S07** | Apache/PHP application delivery | S06 | Apache/PHP-FPM web delivery, healthy unauthenticated endpoints |
| **S08** | Trusted supplied-certificate TLS | S07 | Trusted HTTPS (`:443`) and WSS (`:8089`) TLS bindings |
| **S09** | Isolated host firewall | S08 | Strict drop-by-default firewall policy, management SSH port |
| **S10** | Pinned VICIphone and synthetic PJSIP | S09 | Deployed VICIphone web client, synthetic PJSIP endpoints |
| **S11** | Synthetic call/media acceptance | S10 | Explicitly invoked qualification harness (not automatic install stage) |
| **S12** | Read-only diagnostics | Provisioned stack | Comprehensive non-mutating system health evaluation (`status`/`doctor`) |

---

## 3. Deterministic Interruption & Failure Injection Fixtures

The deterministic fixture suite in `internal/provision/install_recovery_linux_test.go` exercises the real `driveInstallStage`, exclusive locking, atomic journals, and receipt handlers without executing real OS package transactions or compilers:

- **Fixture Execution:** `go test -v -run '^TestInstallationBoundaryInjection$' ./internal/provision`
- **Result:** 20/20 test cases passed in 0.027s.
- **Before-Effect Boundaries:**
  - **S01 & S03:** Explicitly designated `retry = journal.Never`. When interrupted before execution, resumption returned `journal.ErrReconcile` and persisted an explicit block. Effects remained exactly `0`. Prevents unsafe replay of incomplete DNF transactions or partial database migrations.
  - **S02, S04–S10:** Idempotent recovery. Resumption executed the action, observing postconditions and logging completion with exactly `1` effect.
- **After-Effect Boundaries (S01–S10):**
  - Action executed (`effects == 1`), but journal marked `RUNNING` without a persisted receipt.
  - On resumption, the observer detected the satisfied postcondition, finalized the journal state to `Succeeded`, and skipped duplicate execution.
  - Effects remained strictly `1`.
  - Re-invoking `driveInstallStage` confirmed that existing accepted receipts bypass executor logic entirely.

---

## 4. Native Copied-Journal Safety Qualification

Evaluated on disposable clean installation Host D (`almadial-alm29-clean-d`, Machine ID `4c7b492940923838a0700adba65ed3eb`, pristine snapshot `alm29-pristine`):

- **Targeted Test:** `TestInstallationNativeCopiedRecovery` (gated by `ALM30_NATIVE_OUTPUT`).
- **Safety Boundary:** Operates exclusively on isolated copies under `/var/lib/almadial-alm30/`; canonical `/var/lib/almadial-install` remained untouched.
- **Observer-Only Evaluation:** Uses native component observers (`checks`, `systemChecks`, `databaseChecks`, `asteriskChecks`, `vicidialChecks`, etc.) without exposing an action executor or provisioning runner.
- **Outcomes:**
  - **S01–S09:** Safely refused automatic recovery (`journal.ErrReconcile`, persisted state `reconciliation-required`). The observers correctly identified that historical completion cannot be assumed without explicit review. Crucially, **zero native commands were replayed**, and host state was preserved.
  - **S10:** Reconciled actual satisfied native postconditions, maintained identity, and succeeded on repeated resume without mutations.
- **QA Qualification:** The refusal of S01–S09 to automatically resume uncertain historical state represents **correct, safe design behavior**, satisfying the canonical acceptance contract that "every selected interruption is safe to resume or explicitly blocked."

---

## 5. Exclusive Concurrency Locking and Cryptographic Plan Approval

### 5.1 Concurrency Locking (`apply.lock`)

- **Handshake Verification:** `TestInstallationLockProcess` verified cross-process locking via pipe synchronization without arbitrary timing sleeps. Competing `install` and `resume` requests returned `journal.ErrBusy`.
- **Live VM Qualification:** On Host D, `flock -x` was held on `/var/lib/almadial-install/apply.lock` while the real CLI was invoked:
  - `almadial install`: Rejected with exit code 1 (`almadial installation: another process owns exclusive execution`).
  - `almadial resume`: Rejected with exit code 1 (`almadial installation: another process owns exclusive execution`).
  - Upon lock release, both `install` and `resume` completed with exit code 0 and `no_op: true`.

### 5.2 Plan Approval & Identity Validation

- **Missing Approval:** `almadial install --plan ... --manifest ...` failed with exit code 2 (mandatory flag requirement).
- **Incorrect Approval:** `--approve-sha256 00000000...` failed with exit code 1 before any mutation was attempted.
- **Approved Repeat Execution:** `--approve-sha256 abc65d197c...` succeeded with exit code 0 (`no_op: true`).
- **Identity Guarantees:** Existing tests (`TestInstallationImmutableIdentity`, `TestResumeIdentity`, `TestM1GateIntegration_ApprovalBoundaries`) confirm that modifications to host state, manifest, plan, or journal identity prevent unauthorized execution.

---

## 6. Second-Run Safety & Managed State Preservation

Independent verification of repeat execution on the provisioned system confirmed genuine NO_OP behavior:

- **Plan Assessment:** Fresh plan returned `actions: []` and `no_op: true`.
- **Installer Repeat Run:** `almadial resume` executed with exit code 0 and `no_op: true`.
- **Preserved Managed State (102 Items):**
  - All 102 managed configuration files, journals, receipts, TLS certificates, cron definitions, and monitor audio files retained identical SHA-256 digests.
  - File ownership (UID/GID) and permissions (file modes) remained 100% identical.
- **System Service & OS Invariance:**
  - Package count remained exactly 731 RPMs.
  - Effective `nft -j list ruleset` digest remained identical.
  - SELinux mode remained `Enforcing`.
  - Service invocation IDs, start timestamps, and restart counters for `mariadb`, `asterisk`, `httpd`, `php-fpm`, `firewalld`, and FastAGI daemons remained identical (zero service restarts).

---

## 7. Resolution and Causal Attribution of Database Audit Fields

During repeat execution qualification, an initial whole-database digest comparison reported a discrepancy. Controlled per-column analysis isolated the variance across 531 individual database checks:

- **Strictly Equal Checks:** 529 checks showed identical SHA-256 hashes, including:
  - All `phones` table columns (extensions, logins, passwords, SIP credentials).
  - All `system_settings` records.
  - Schema column definitions (`information_schema.columns`).
  - MariaDB user privileges (`mysql.global_priv`).
  - Call recording logs (`recording_log`).
- **Changed Fields (Exactly 2):**
  1. `vicidial_users.last_login_date`
  2. `vicidial_users.modify_stamp`
- **Causal Attribution:**
  - Upstream VICIdial `functions.php` explicitly updates `last_login_date = NOW()` upon successful administrator HTTP authentication.
  - `modify_stamp` updates automatically via MariaDB timestamp triggers on row update.
  - The S07/S08 health checks included in `almadial resume` execute an authenticated query against the web portal to verify application availability.
  - This activity is normal application runtime auditing. It does **not** represent installer provisioning replay, data overwrite, or configuration corruption.
  - The controlled test conclusively proved zero database writes occur when web health probes are withheld.

---

## 8. Functional Stack Diagnostics

Read-only diagnostics on the installed system confirmed stack operational health:

- **Probes:** 39 total checks executed.
- **Results:** 38 OK, 1 WARN, 0 FAIL.
- **Exit Code:** 10 (matching the canonical warning exit contract).
- **Warning Qualification:** The single warning (`tls.expiration`) was triggered by the test harness's 14-day laboratory TLS certificate, which falls within the diagnostic tool's standard 30-day renewal warning window. This is an expected operational finding, not a system defect.
- **Stack Components Verified:**
  - MariaDB service active, schema version 1748.
  - Asterisk 22.6.0 running with required native modules and PJSIP transport-wss active.
  - FastAGI running with 1 parent and 5 worker processes.
  - Apache/PHP-FPM operational.
  - Host firewall S09 drop-by-default policy enforcing network isolation.

---

## 9. Comprehensive Regression Verification

All regression tests were independently executed in the local test environment using Go 1.27.3:

| Regression Check | Command | Observed Result | Status |
|---|---|---|:---:|
| **Go Vet** | `go vet ./...` | Clean exit 0; zero issues | **PASS** |
| **Go Unit Tests** | `go test -json ./...` | 1,456 passing events across 21 packages (2 skipped) | **PASS** |
| **Go Race Detector** | `go test -race ./...` | Clean exit 0; zero data races | **PASS** |
| **Go Mod Verify** | `go mod verify` | All modules verified clean | **PASS** |
| **Go Mod Tidy Check** | `go mod tidy -diff` | Clean exit 0; zero drift | **PASS** |
| **Go Formatting** | `gofmt -l .` | Zero unformatted files | **PASS** |
| **Scaffold Validation** | `python3 scripts/validate_scaffold.py` | 11 files, 4 skills validated | **PASS** |
| **Python Unit Tests** | `python3 -m unittest discover -s scripts/tests -v` | 222 tests passed in 2.57s | **PASS** |
| **Git Diff Whitespace** | `git diff --check` | Clean exit 0; zero whitespace/conflict errors | **PASS** |
| **CLI Smoke Cases** | Built binary smoke matrix | 14/14 test cases passed | **PASS** |
| **Terminal Journeys** | `scripts/test_terminal_wizard.py` | 11 journeys passed (exit 0) | **PASS** |
| **Reproducible Builds** | 12 plain command targets built twice | 12/12 targets bit-for-bit identical SHA-256 | **PASS** |
| **Hosted CI** | Remote workflow triggers | No remote push authorized | **NOT_RUN** |

---

## 10. Evidence Verification Ledgers

### 10.1 Developer Evidence Ledger

- **Directory:** `/home/ra/almadial-lab/evidence/m4-02/20261112T155041Z-alm30/`
- **Ledger Digest:** `0c581d6bbdcbe465a4fe707946407722fef561e942a10b73c7ade5edb1e49c57` (`SHA256SUMS`)
- **Audit Verification:** All 71 covered evidence files verified with 100% SHA-256 agreement.

### 10.2 Independent QA Evidence Ledger

- **Directory:** `/home/ra/almadial-lab/evidence/m4-02/qa-20261113T010500Z-alm30/`
- **Ledger Digest:** `f09f60617babee2190f61991e02a0badaada21947884292613dee060717c5386` (`SHA256SUMS`)
- **Contained Evidence Files:**
  - `identity.json`
  - `ledger-verification.json`
  - `fixture-verification.json`
  - `copied-journal-evaluation.json`
  - `concurrency-and-approval.json`
  - `second-run-noop-and-preservation.json`
  - `database-audit-write-attribution.json`
  - `regression-summary.json`
  - `qa-results-summary.json`

---

## 11. Final Independent QA Verdict

All canonical acceptance criteria for Work Package ALM-30 / M4-02 have been thoroughly and independently verified. The recovery adapter, durable journals, process locking, and plan approval prevent duplicate mutations and safely refuse uncertain historical completions. Second-run execution produces genuine NO_OP while preserving managed host and database state.

The independent QA verdict is formally declared:

```text
ALMADIAL_M4_02_RECOVERY_ACCEPTED
```
