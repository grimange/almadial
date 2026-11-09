# ALM-17 / M2-02 Independent Service Identity, Storage, SELinux, and Recovery QA Report

- **Task ID:** `ou5231ijs6ilp28howhd6xs1` (ALM-17 / M2-02)
- **Task Title:** Provision service identities, storage and SELinux labels
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T03:55:00+08:00 (2026-11-09T19:55:00Z)
- **Repository Baseline Commit:** `36c1b6eb8e183e241e1cb58d3e7c0fa95ad73e1a` (Accepted ALM-16 commit)
- **Developer Baseline HEAD:** `ec101ecb61138fe3ccf6d613dac3d20c861b77c2`
- **Starting Working HEAD:** `1996b951f68503fdabef87e0e8ba332075fb30b1` (Preserved intervening operator commits)
- **Final Verdict:** `ALMADIAL_M2_02_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-17 (`ou5231ijs6ilp28howhd6xs1`), number 17 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No task reassignment or self-assignment occurred.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Task Status Prior to Review:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Implementation Evidence Comment:** Verified comment `fcwczwn4hkzjjb8sry263p0n` posted by Codex on 2026-11-09T18:22:18Z matches recorded implementation details and digests.
- **Dependency Graph:**
  - Relation ALM-16 $\rightarrow$ ALM-17 (`xbvyqe08plnsvrgn47yy7alg`): Verified intact (`blocks`), source ALM-16 (`g3ibghcq6p3y55o1bczk5a7n`) status `done` (Accepted).
  - Relation ALM-17 $\rightarrow$ ALM-18 (`wr227vtlq2zmy1omyu24e57d`): Verified intact (`blocks`), target ALM-18 (`ldl2ak2bk3pcq7bwvk9w54wc`) status `to-do` (Backlog, unassigned).
  - Relation ALM-17 $\rightarrow$ ALM-19 (`s8822axs1a6cmyxg9fa698eg`): Verified intact (`blocks`), target ALM-19 (`nwk5zbmgwuuhsnlsh92sxl1v`) status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-17 implements Stage S02 (`internal/provision/system.go`, `system_native_linux.go`, `system_stage_linux.go` and `cmd/almadial-system`), responsible for service identities (`apache`, `mysql`, `asterisk`), 16 managed filesystem directories, SELinux persistent label verification, recording storage budget inspection, exclusive execution locking, and crash recovery.
- **Separation of Concerns:** ALM-17 does not build Asterisk, compile PJPROJECT, install VICIdial application files, deploy VICIphone, initialize MariaDB databases, modify telephony services, or alter firewalld/SELinux policies.
- **Candidate Manifest Authorization:** `manifests/stack.lock.yaml` remains strictly `candidate`. `InstallationAuthorized()` continues to deny production installation. Laboratory provisioning in `cmd/almadial-system` requires explicit, cryptographically bound `SystemLabAuthorization` with scope `ALM-17/disposable-lab`.

---

## 2. Protected Evidence Ledger Verification

### 2.1 Developer Evidence Ledger
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-02/20261109T181303Z/`
- **Master Ledger:** `SHA256SUMS`
  - Computed SHA-256: `27a5f52f787cccced373053b2de113fdc5b6e7b5b230c4d42a7319bf7b64f311` (exact match)
  - File Count: 218 entries
  - Verification: `sha256sum -c SHA256SUMS` executed with 100% OK across all 218 files.
- **Handoff Ledger:** `HANDOFF_SHA256SUMS`
  - Computed SHA-256: `23a3c6c6000750a947af52d39c1e48f6033ec597bcceab11400fe0f0298b312a` (exact match).
  - Verification: `sha256sum -c HANDOFF_SHA256SUMS` executed with 100% OK.
- **Task Source Manifest:** `source-sha256.json`
  - All 7 task-owned files independently verified byte-for-byte against the working tree.

### 2.2 Independent QA Evidence Ledger
- **QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-02/qa-20261109T195500Z/` (mode `0700`, files `0600`)
- **Master Ledger SHA-256:** `7b7f76208408091fab3ecfbae1ce5b0c576851328de77a26c9470e0a79ac35c4`
- **Artifacts Preserved (17 files):**
  1. `starting-head.txt`
  2. `ancestry-verification.txt`
  3. `developer-ledger-verification.txt`
  4. `task-source-sha256.json`
  5. `vm-nss-accounts.json`
  6. `vm-16-directories.json`
  7. `vm-selinux-verification.txt`
  8. `vm-recording-storage.txt`
  9. `vm-baseline-preservation.txt`
  10. `vm-journal-and-recovery.json`
  11. `go-test-results.json`
  12. `go-race-results.json`
  13. `python-test-results.json`
  14. `cli-smoke-results.json`
  15. `pty-journey-results.json`
  16. `reproducible-builds.json`
  17. `kaneo-baseline.json`

---

## 3. Service Identity Verification

### 3.1 Observed Linux NSS Accounts on Authorized Laboratory VM
Live VM inspection on `almadial-alm16-rpm` (`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`):

| Account | UID:GID | Home Directory | Login Shell | Groups | Management Class | Finding |
|---|---|---|---|---|---|---|
| `apache` | `48:48` | `/usr/share/httpd` | `/sbin/nologin` | `48(apache)` | Pre-existing vendor RPM | PASS (Preserved) |
| `mysql` | `27:27` | `/var/lib/mysql` | `/sbin/nologin` | `27(mysql)` | Pre-existing vendor RPM | PASS (Preserved) |
| `asterisk` | `993:993` | `/var/lib/asterisk` | `/sbin/nologin` | `993(asterisk)` | Provisioned system account | PASS (Verified) |

### 3.2 Account Security and Policy Compliance
1. **Vendor Identity Preservation:** Existing RPM accounts for Apache and MariaDB were verified untouched. No modifications to their UIDs, GIDs, homes, shells, or group memberships occurred.
2. **Dynamic System Allocation:** Asterisk system identity was allocated dynamically via shadow-utils (`groupadd --system` and `useradd --system`). The observed numeric UID/GID (`993:993`) is an observation of host allocation, not a hardcoded default.
3. **Shell and Home Discipline:** All service accounts possess `/sbin/nologin` (non-interactive). The Asterisk home directory is `/var/lib/asterisk`, created without auto-skel or password initialization.
4. **Group Isolation:** Each account is restricted strictly to its own primary group. Apache does not belong to the Asterisk group, preventing cross-service privilege leakage.
5. **Deterministic Conflict Rejection:** Unit and live negative tests verify that conflicting UIDs, mismatched primary groups, interactive shells, unexpected home directories, or missing vendor accounts deterministically block execution with structured findings.

---

## 4. Filesystem Hierarchy and Directory Matrix Verification

### 4.1 Matrix of 16 Managed Directories
Live read-only `stat` and `namei` inspection on `almadial-alm16-rpm`:

| # | Path | Owner:Group | Mode | Expected Context | Actual Observed Context | Status |
|---|---|---|---|---|---|---|
| 1 | `/etc/almadial` | `root:root` | `0700` | `etc_t` | `system_u:object_r:etc_t:s0` | PASS |
| 2 | `/etc/almadial/secrets` | `root:root` | `0700` | `etc_t` | `system_u:object_r:etc_t:s0` | PASS |
| 3 | `/var/lib/almadial` | `root:root` | `0700` | `var_lib_t` | `system_u:object_r:var_lib_t:s0` | PASS |
| 4 | `/var/lib/almadial/artifacts` | `root:root` | `0700` | `var_lib_t` | `system_u:object_r:var_lib_t:s0` | PASS |
| 5 | `/var/lib/almadial/backups` | `root:root` | `0700` | `var_lib_t` | `system_u:object_r:var_lib_t:s0` | PASS |
| 6 | `/var/log/almadial` | `root:root` | `0700` | `var_log_t` | `system_u:object_r:var_log_t:s0` | PASS |
| 7 | `/var/log/almadial/evidence` | `root:root` | `0700` | `var_log_t` | `system_u:object_r:var_log_t:s0` | PASS |
| 8 | `/etc/asterisk` | `root:asterisk` | `0750` | `etc_t` | `system_u:object_r:etc_t:s0` | PASS |
| 9 | `/etc/asterisk/keys` | `root:asterisk` | `0750` | `etc_t` | `system_u:object_r:etc_t:s0` | PASS |
| 10 | `/var/lib/asterisk` | `asterisk:asterisk` | `0750` | `var_lib_t` | `system_u:object_r:var_lib_t:s0` | PASS |
| 11 | `/var/lib/asterisk/agi-bin` | `asterisk:asterisk` | `0750` | `bin_t` | `system_u:object_r:bin_t:s0` | PASS |
| 12 | `/var/lib/asterisk/sounds` | `asterisk:asterisk` | `0750` | `var_lib_t` | `system_u:object_r:var_lib_t:s0` | PASS |
| 13 | `/var/log/asterisk` | `asterisk:asterisk` | `0750` | `var_log_t` | `system_u:object_r:var_log_t:s0` | PASS |
| 14 | `/var/spool/asterisk` | `asterisk:asterisk` | `0750` | `var_spool_t` | `system_u:object_r:var_spool_t:s0` | PASS |
| 15 | `/var/spool/asterisk/monitor` | `asterisk:asterisk` | `0750` | `var_spool_t` | `system_u:object_r:var_spool_t:s0` | PASS |
| 16 | `/var/spool/asterisk/monitorDONE` | `asterisk:asterisk` | `0750` | `var_spool_t` | `system_u:object_r:var_spool_t:s0` | PASS |

### 4.2 Path Safety and Symlink Resistance
- **O_NOFOLLOW / O_DIRECTORY:** Native descriptor-relative traversal using `openat` and `mkdirat` ensures symlinks in any path component are strictly rejected.
- **Fixed Child Helper:** Directory operations execute via an internal child helper bound to the parent's file descriptor, held ALM-14 lock, and durable running journal record. Arbitrary destination arguments or direct helper invocations are rejected.
- **Zero Broad Chmod/Chown:** Mutations operate strictly on specific directory inodes. No recursive ownership (`chown -R`) or permission changes (`chmod -R 777`) are executed.

### 4.3 Existing Nonempty Directory Content Preservation
- **Live Sentinel Inode Verification:** On the authorized VM, an operator-placed sentinel file `/var/lib/almadial/backups/alm17-authorized-operator-sentinel` was verified:
  - Inode: `67108992` (unchanged)
  - SHA-256: `5b96dd5b2aa0368173f6088c4c5e7434e0278a83118dba97b8a2eb26b55ff268` (unchanged)
  - Ownership: `root:root` (0:0)
  - Mode: `0600`
- **Result:** Existing unrelated files inside managed directories are strictly preserved without metadata clobbering or data truncation.

---

## 5. SELinux Enforcement and Persistent Label Verification

### 5.1 Host SELinux Mode
- `getenforce`: `Enforcing`
- `sestatus`: Mode from config `enforcing`, loaded policy `targeted`, MLS enabled.
- Zero SELinux mode changes, permissive bypasses, or custom policy module injections occurred.

### 5.2 Context Verification and Matchpathcon Audit
- **Persistent Context Reuse:** All 16 directories reuse standard distribution file-context mappings (`etc_t`, `var_lib_t`, `var_log_t`, `bin_t`, `var_spool_t`). Zero custom `semanage fcontext` rules were added.
- **`matchpathcon -V`:** Executed against all 16 managed directories; all 16 reported `verified.`.
- **`restorecon -n -v`:** Executed against all 16 managed directories; produced 0 proposed modifications (exit code 0), verifying that in-memory labels match persistent disk policies.

### 5.3 Downstream Runtime Boundary Qualification
- *Explicit Limitation Note:* Verification of directory file contexts (`matchpathcon -V`) confirms that filesystem labels match the targeted distribution policy. It does **not** prove that future Asterisk or VICIdial runtime processes will have all required SELinux access vectors. Full AVC audit and service confinement testing belong to downstream installation and runtime qualification stages.

---

## 6. Recording Storage Capacity and Inodes

### 6.1 Recording Storage Inspection
Read-only inspection of `/var/spool/asterisk/monitor` and `/var/spool/asterisk/monitorDONE` via `findmnt` and `df`:

| Parameter | Observed Value | Approved Laboratory Minimum | Qualification Verdict |
|---|---|---|---|
| Target Mount Point | `/` | `/` | PASS |
| Source Device | `/dev/sda4` | `/dev/sda4` | PASS |
| Filesystem Type | `xfs` | `xfs` or `ext4` | PASS |
| Available Capacity | `81,445,748,736` bytes (~75.8 GiB) | `1,073,741,824` bytes (1 GiB) | PASS |
| Available Inodes | `41,269,657` free inodes | `10,000` free inodes | PASS |
| Mount Options | `rw,relatime,seclabel,attr2,inode64,logbufs=8,logbsize=32` | Local, writable, seclabel | PASS |

### 6.2 Laboratory vs Production Policy Boundary
- The verified minimum (1 GiB and 10,000 inodes) is an explicit **laboratory qualification threshold** approved by the operator.
- It is **not** a production recording capacity standard. Production storage budgets remain a mandatory operator-supplied configuration input.

---

## 7. Execution Architecture, Locking, and Recovery

### 7.1 Action Graph Identity and Runner Contract
- Pure assessment produces a deterministic 18-action execution graph (`system.00.group`, `system.01.user`, `system.02.directory.00` through `system.02.directory.15`).
- Mutating actions use direct absolute command execution (no shell interpolation), 30-second deadlines, 4 MiB stdout limits, and 16 KiB stderr limits.
- Actions execute through the accepted ALM-13 runner and ALM-14 journal.

### 7.2 Interrupted-Action Reconciliation
- In the developer laboratory qualification run, process execution was killed via `SIGKILL` immediately after Asterisk group creation (`system.00.group`).
- The journal captured `system.00.group` in state `running` with exit code `-1` and the remaining 17 actions `pending`.
- Upon `lab-resume`, the stage re-observed host state, identified that the Asterisk group already existed and satisfied the postcondition, reconciled the action to `succeeded` without redundant recreation, and completed the remaining 17 actions.
- All 18 final journal records reached `succeeded`.

### 7.3 Idempotent NO_OP Assessment
- Following completion, a fresh inspection and assessment on the live host returned 0 required actions.
- Fresh execution and resume of the completed plan both returned `no_op: true` with zero mutations and zero state transitions.

---

## 8. ALM-16 RPM Baseline and System Preservation

- **RPM Inventory:** Exactly 725 installed packages before and after S02 provisioning. Zero packages installed, removed, or upgraded.
- **Repository Trust:** All 9 AlmaLinux repository files in `/etc/yum.repos.d/` preserved their pristine state. Zero foreign repositories or EPEL repo files written.
- **DNF Transaction History:** Exactly 1 transaction recorded (ALM-16 transaction 1 on 2026-11-10 00:45). Zero DNF transactions executed in ALM-17.
- **Firewall and System Services:** Zero firewalld rule changes, zero network modifications, zero service startup or application daemon activations.

---

## 9. Regression Testing and Verification Suites

### 9.1 Go Verification Suites
Executed using the repository's pinned Go 1.27.3 toolchain (`/tmp/alm9-toolchain/go/bin/go`):
- `go fmt ./...`: Exit 0 (clean, no files modified).
- `make fmt-check`: Exit 0.
- `go vet ./...`: Exit 0 (zero issues).
- `go mod verify`: Exit 0 (`all modules verified`).
- `go test ./...`:
  - Total Test Packages: 13 ok, 1 no test files
  - Distinct Top-Level Passed: 137 (138 total, 1 skipped: `TestM0Artifacts` external fixture)
  - Pass Events: 743
  - Failures: 0
- `go test -race ./...`:
  - Distinct Top-Level Passed: 137
  - Pass Events: 743
  - Race Conditions: 0

### 9.2 Python Verification Suites
- `python3 scripts/validate_scaffold.py`: PASS (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`:
  - Total Tests: 131
  - Passed: 131, Failures: 0, Errors: 0 (Ran in 1.868s, OK).
- `python3 scripts/verify_candidate_stack.py`: PASS (`candidate_consistency: PASS`).
- `python3 scripts/verify_config_generation.py`: PASS (`state: PASS`).
- `git diff --check`: Clean (no whitespace or conflict markers).

### 9.3 CLI Smoke Tests and PTY Journeys
- **CLI Smoke Suite:** 17/17 cases passed. Verified help, version, and rejection of invalid subcommands without credential or secret leakage.
- **Terminal/Nonterminal Wizard Journeys:** 11/11 cases passed via `scripts/test_terminal_wizard.py` across real Linux PTYs (8 interactive cases + 3 non-terminal cases).

### 9.4 Reproducible Builds
Two isolated CGO-disabled Linux amd64 builds from independent build directories yielded identical binary hashes:
- `almadial`: `a87f7ee60fd31c84df175f016713bd7512d4b2dd314b98b108e3040eb341fea0`
- `almadial-rpm`: `945e5cdcfd17b11673a07f3c0a6a164c412e6fd1bd2ddd29f9ff1ac83ed97bf5`
- `almadial-system`: `5cff10332626f0d13621f93386cf07b3b7be5afbadbaadcb9b125f04d278c927`

---

## 10. Independent QA Summary Matrix

| Verification Area | Requirement / Check | Observed Result | Status |
|---|---|---|---|
| Service Identities | Vendor account preservation (`apache`, `mysql`) | Preserved intact (48:48, 27:27, nologin) | PASS |
| Service Identities | Asterisk system account and group | Allocated dynamically (993:993, nologin) | PASS |
| Service Identities | UID/GID conflict rejection & isolation | Rejects conflicts; primary groups only | PASS |
| Filesystem | 16 managed directories existence and stat | Correct ownership and modes (0700/0750) | PASS |
| Filesystem | Symlink resistance & path safety | `O_NOFOLLOW | O_DIRECTORY`, helper locked | PASS |
| Filesystem | Nonempty directory content preservation | Sentinel file retained inode, hash, mode | PASS |
| SELinux | SELinux Enforcing & targeted policy | Mode Enforcing, policy targeted | PASS |
| SELinux | Persistent label verification | `matchpathcon -V` verified 16/16 | PASS |
| SELinux | Restorecon dry-run inspection | `restorecon -n -v` proposed 0 modifications | PASS |
| Storage | Recording storage budget check | 81.4 GB / 41.2M inodes exceeds lab minimum | PASS |
| Execution & Lock | Immutable graph & exclusive flock | 18 actions, linear dependencies, held lock | PASS |
| Recovery | Interrupted action reconciliation | Re-observed host postcondition, completed | PASS |
| Idempotency | Fresh assessment & rerun NO_OP | 0 required actions, no-op execution | PASS |
| Baseline | ALM-16 RPM & repository preservation | 725 RPMs, 9 repos, 0 new DNF transactions | PASS |
| Test Suites | Go test suite & race detector | 137 top-level passed, 743 pass events, 0 race | PASS |
| Test Suites | Python unit & validation suites | 131 tests passed, scaffold & stack valid | PASS |
| CLI & Journeys | CLI smoke & PTY wizard journeys | 17 smoke passed, 11 PTY journeys passed | PASS |
| Toolchain | Reproducible Go builds | 3/3 binaries byte-identical across builds | PASS |
| Evidence | Protected QA evidence ledger | 17 artifacts, SHA-256 sealed | PASS |

---

## 11. Final Verdict

All canonical acceptance criteria for ALM-17 / M2-02 are satisfied. The implementation adheres strictly to the single-server scope, preserves vendor accounts, allocates the Asterisk service identity deterministically, provisions the 16 approved directory paths with exact permissions and persistent SELinux contexts, respects the approved laboratory storage budget, reconciles crashed actions via durable journaling, and preserves the ALM-16 package baseline.

**Verdict:**

`ALMADIAL_M2_02_ACCEPTED`
