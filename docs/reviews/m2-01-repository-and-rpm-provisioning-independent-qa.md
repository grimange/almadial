# ALM-16 / M2-01 Independent RPM Provisioning, Repository Trust, and Recovery QA Report

- **Task ID:** `g3ibghcq6p3y55o1bczk5a7n` (ALM-16 / M2-01)
- **Task Title:** Provision approved repositories and RPM dependencies
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T01:30:00Z
- **Repository Baseline Commit:** `c761930235a7810008dc3fb45d6325796ff5b208` (Milestone M1 Gate Accepted)
- **Starting HEAD:** `d851d86279e2611b3fa70d57834f5cc938d8e200` (Preserved intervening operator commit)
- **Accepted Commit:** `1fa2afc` (recorded below after finalization)
- **Final Verdict:** `ALMADIAL_M2_01_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-16 (`g3ibghcq6p3y55o1bczk5a7n`), number 16 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No task reassignment or self-assignment occurred.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Evidence Comment:** Verified comment `mdo1a70w5aqj0q9rtp5o8b04` posted by Codex on 2026-11-09T17:10:59Z matches recorded implementation details and digests.
- **Dependency Graph:**
  - Upstream ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`, M1-07): Verified status `done` (Accepted).
  - Milestone M1 Integration Gate: Verified Accepted (`c761930235a7810008dc3fb45d6325796ff5b208`).
  - Relation ALM-15 $\rightarrow$ ALM-16 (`vpmvdb97n6ptzwb56k0ll5v9`): Verified intact (`blocks`).
  - Relation ALM-16 $\rightarrow$ ALM-17 (`xbvyqe08plnsvrgn47yy7alg`): Verified intact (`blocks`).
  - Downstream ALM-17 (`ou5231ijs6ilp28howhd6xs1`, M2-02): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-16 implements Stage S01 (`internal/provision` and `cmd/almadial-rpm`), responsible for repository trust, exact RPM dependency resolution, immutable plan generation, and authorized laboratory provisioning.
- **Separation of Concerns:** ALM-16 does not build Asterisk, compile PJPROJECT, install VICIdial application files, deploy VICIphone, initialize MariaDB databases, modify telephony services, or alter firewalld/SELinux policies.
- **Candidate Manifest Authorization:** `manifests/stack.lock.yaml` remains strictly `candidate`. `InstallationAuthorized()` continues to deny production installation. Laboratory provisioning in `cmd/almadial-rpm` requires explicit, cryptographically bound `LabAuthorization` with scope `ALM-16/disposable-lab`, host machine-ID digest, stage digest, parent plan digest, manifest digest, snapshot reference, and operator evidence digest.

---

## 2. Protected Evidence Ledger Verification

### 2.1 Developer Evidence Ledger
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-01/20261109T161558Z/`
- **Master Ledger:** `SHA256SUMS`
  - Computed SHA-256: `67c9095b1accc8d532380cb1b4e64cb54ec06b2073e639b070829d30daf984be` (exact match)
  - File Count: 258 entries
  - Verification: `sha256sum -c SHA256SUMS` executed with 100% OK.
- **Handoff Ledger:** `kaneo/SHA256SUMS`
  - Computed SHA-256: `f18f9456c2d8687390eac735031ad8e31cf80f5c7ccb852387b055f8c0891e19` for `readback.json` (exact match).
- **Task Source Manifest:** `task-source-sha256.json`
  - All 17 task-owned files independently verified byte-for-byte against the working tree.

### 2.2 Independent QA Evidence Ledger
- **QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-01/qa-20261109T172800Z/` (mode `0700`, files `0600`)
- **Master Ledger SHA-256:** `7ec52563943978c365c8a8c87873f82ae64bee787d17e21812080e1d53d7b81b`
- **Artifacts Preserved (17 files):**
  1. `starting-head.txt`
  2. `ancestry-verification.txt`
  3. `developer-ledger-verification.txt`
  4. `developer-kaneo-ledger.txt`
  5. `task-source-sha256.json`
  6. `package-requirements-matrix.json`
  7. `repository-trust-verification.json`
  8. `reviewed-transaction-verification.json`
  9. `vm-inventory-verification.json`
  10. `idempotency-noop-evidence.json`
  11. `interrupted-recovery-verification.json`
  12. `sqlite-read-coordination-audit.json`
  13. `go-test-results.json`
  14. `python-test-results.json`
  15. `cli-smoke-results.json`
  16. `pty-wizard-results.json`
  17. `reproducible-builds.json`

---

## 3. Repository Provenance and Trust Contract Verification

### 3.1 Approved Repositories
The repository policy in `internal/provision/policy.go` approves exactly five repositories:

| Repository | Source URL / Mirrorlist | Usage / Policy |
|---|---|---|
| `baseos` | `https://mirrors.almalinux.org/mirrorlist/10/baseos` | Enabled AlmaLinux 10 distribution definition |
| `appstream` | `https://mirrors.almalinux.org/mirrorlist/10/appstream` | Enabled AlmaLinux 10 distribution definition |
| `crb` | `https://mirrors.almalinux.org/mirrorlist/10/crb` | Enabled AlmaLinux 10 distribution definition |
| `extras` | `https://mirrors.almalinux.org/mirrorlist/10/extras` | Enabled AlmaLinux 10 distribution definition |
| `epel-almadial` | `https://dl.fedoraproject.org/pub/epel/10.2/Everything/x86_64/` | Transaction-local; strictly restricted to 6 qualified Perl RPMs |

### 3.2 EPEL Scoping Boundary
- **Transaction-Local Enforcement:** EPEL is configured in-memory during native DNF execution with `includepkgs=perl-Class-Mix,perl-Crypt-Eksblowfish,perl-Devel-CallChecker,perl-DynaLoader-Functions,perl-Params-Classify,perl-Switch`.
- **Zero Persistent Modification:** No EPEL `.repo` file is written to `/etc/yum.repos.d/`. All 9 pre-existing AlmaLinux repository files (`almalinux-*.repo`) preserved their exact SHA-256 hashes before and after provisioning.
- **Zero Foreign Package Leakage:** In DNF transaction 1, exactly the 6 approved Perl packages were installed from `@epel-almadial`. Zero extraneous EPEL packages or transitive dependencies entered the package closure.

### 3.3 Signing Key Identities and Verification
- **AlmaLinux 10 Key:**
  - Key ID: `c2a1e572` (`EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572`)
  - ASCII Armor SHA-256: `de57c3c0a2dcde88ff5b894e74335f55a0ec074815f36d472270d937c16f077b`
  - Verified on 38 AlmaLinux packages in target VMs.
- **Fedora EPEL 10 Key:**
  - Key ID: `e37ed158` (`7D8D15CBFC4E62688591FB2633D98517E37ED158`)
  - ASCII Armor SHA-256: `de390fc168eae5ab2852e9e93d34a0b9ddf05cf9ce90ee28d97de26a4b1f6b93` (`internal/provision/epel.asc`)
  - Verified on all 6 EPEL Perl packages in target VMs.
- **Security Controls:** GPG checking (`gpgcheck=1`) and TLS verification are mandatory. The native adapter checks package signatures via `base.package_signature_check(p)` before transaction commit. Foreign repositories, conflicting definitions, and invalid signatures fail closed.

---

## 4. Package Requirements Verification Matrix

All 44 direct RPM requirements from `internal/provision/packages.json` were queried directly on live disposable VMs `almadial-alm16-rpm` and `almadial-alma10-repeat`.

| # | Package Name | Pinned EVR | Arch | Repository Origin | Signer Key ID | Observed Status |
|---|---|---|---|---|---|---|
| 1 | `autoconf` | `0:2.71-13.el10` | `noarch` | `appstream` | `c2a1e572` | **SATISFIED** |
| 2 | `automake` | `0:1.16.5-20.el10` | `noarch` | `appstream` | `c2a1e572` | **SATISFIED** |
| 3 | `bzip2` | `0:1.0.8-25.el10` | `x86_64` | `baseos` | `c2a1e572` | **SATISFIED** |
| 4 | `ed` | `0:1.20-5.el10` | `x86_64` | `baseos` | `c2a1e572` | **SATISFIED** |
| 5 | `gcc` | `0:14.3.1-4.4.el10.alma.2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 6 | `gcc-c++` | `0:14.3.1-4.4.el10.alma.2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 7 | `httpd` | `0:2.4.63-13.el10_2.8` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 8 | `jansson-devel` | `0:2.14-3.el10` | `x86_64` | `crb` | `c2a1e572` | **SATISFIED** |
| 9 | `libedit-devel` | `0:3.1-52.20230828cvs.el10` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 10 | `libsrtp-devel` | `0:2.3.0-16.el10` | `x86_64` | `crb` | `c2a1e572` | **SATISFIED** |
| 11 | `libtool` | `0:2.4.7-13.el10` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 12 | `libuuid-devel` | `0:2.40.2-18.el10_2.20` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 13 | `libxml2-devel` | `0:2.12.5-10.el10_2.4` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 14 | `make` | `1:4.4.1-9.el10` | `x86_64` | `baseos` | `c2a1e572` | **SATISFIED** |
| 15 | `mariadb` | `3:10.11.18-1.el10_2.alma.2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 16 | `mariadb-backup` | `3:10.11.18-1.el10_2.alma.2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 17 | `mariadb-server` | `3:10.11.18-1.el10_2.alma.2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 18 | `mod_ssl` | `1:2.4.63-13.el10_2.8` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 19 | `ncurses-devel` | `0:6.4-15.20240127.el10_1` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 20 | `openssl-devel` | `1:3.5.8-2.el10_2.alma.1` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 21 | `patch` | `0:2.7.6-26.el10` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 22 | `perl` | `4:5.40.2-515.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 23 | `perl-Class-Mix` | `0:0.006-22.el10_0` | `noarch` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 24 | `perl-Crypt-Eksblowfish` | `0:0.009-42.el10_0` | `x86_64` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 25 | `perl-DBD-MySQL` | `0:5.007-4.el10` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 26 | `perl-DBI` | `0:1.643-26.el10_2.7` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 27 | `perl-Devel-CallChecker` | `0:0.009-7.el10_0` | `x86_64` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 28 | `perl-DynaLoader-Functions` | `0:0.004-5.el10_0` | `noarch` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 29 | `perl-Net-Server` | `0:2.014-9.el10` | `noarch` | `appstream` | `c2a1e572` | **SATISFIED** |
| 30 | `perl-Params-Classify` | `0:0.015-18.el10_0` | `x86_64` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 31 | `perl-Switch` | `0:2.17-33.el10_1` | `noarch` | `epel-almadial` | `e37ed158` | **SATISFIED** |
| 32 | `perl-Time-HiRes` | `4:1.9777-511.el10` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 33 | `php-cli` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 34 | `php-common` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 35 | `php-fpm` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 36 | `php-gd` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 37 | `php-ldap` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 38 | `php-mbstring` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 39 | `php-mysqlnd` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 40 | `php-opcache` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 41 | `php-xml` | `0:8.3.36-1.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 42 | `pkgconf` | `0:2.1.0-3.el10` | `x86_64` | `baseos` | `c2a1e572` | **SATISFIED** |
| 43 | `sqlite-devel` | `0:3.46.1-6.el10_2` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |
| 44 | `zlib-ng-compat-devel` | `0:2.2.3-3.el10_1` | `x86_64` | `appstream` | `c2a1e572` | **SATISFIED** |

**Verification Outcome:** 44/44 SATISFIED on `almadial-alm16-rpm` and 44/44 SATISFIED on `almadial-alma10-repeat`.

---

## 5. DNF Transaction and RPM Postconditions Verification

### 5.1 Reviewed Transaction Proposal
- **Proposal SHA-256:** `79eba1f6bc2f0fdd4e9960fb95f22f57e2721b0cb673e9f494dbe6ee7ef0f7f8` (`clean/reviewed-transaction.json`).
- **Installs:** 351 packages (344 pure additions + 7 updated packages).
- **Removals:** 7 old packages.
- **Repositories Involved:** `appstream`, `baseos`, `crb`, `epel-almadial`.

### 5.2 The Seven Approved Coupled BaseOS Updates
The initial additive policy correctly rejected the transaction because DNF required seven coupled updates to satisfy `libuuid-devel`. The operator explicitly reviewed and authorized these exact seven BaseOS updates in `internal/provision/upgrades.json`:

| Package | Replaced Version (`@System`) | New Version (`baseos`) | Payload SHA-256 |
|---|---|---|---|
| `libblkid` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `b472dada4c00867620c685a607ef83bede8be30d0d0bdee78146705ce3da1e3f` |
| `libfdisk` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `dd79991e6155039de9d923c0bc78998436e7c51719909a260b6af1a98a601f7b` |
| `libmount` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `6d687517d4f6dd724e3bedf51b6b81a37390f30a7fbdb4ce8bbc1dcc05116256` |
| `libsmartcols` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `7323f3157a0b7cb5024b56b560abb6db3f502ee63c2f7b29171b9b567e7b7689` |
| `libuuid` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `cbd3bce21a3f15a0dada094fe58ca1b23b5b7eb629273897a6f2d29c22f11b70` |
| `util-linux` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `839bd1debbf135694acc8c0354c0a03f69ba527e8a21b88038a49943f75f36dc` |
| `util-linux-core` | `0:2.40.2-18.el10.x86_64` | `0:2.40.2-18.el10_2.20.x86_64` | `eb516c7e01013603e908f17fac28c523e6daec751cad1e43c8b5f30caafae6e3` |

No arbitrary upgrades, downgrades, cross-architecture substitutions, or uncoupled removals occurred.

### 5.3 Live Laboratory VM Verification
- **VM A — `almadial-alma10-repeat`:**
  - Initial/current RPM count: 744 packages.
  - All 44 direct requirements satisfied.
  - Native transaction resolution returned empty transaction (`no_op: true`).
  - Recorded recovery snapshot: `alm16-recovery-20261110`; historical `clean-baseline` preserved.
- **VM B — `almadial-alm16-rpm`:**
  - Fresh image baseline: 380 RPMs (0/44 satisfied).
  - After EPEL public key import: 381 RPMs (Key ID `e37ed158` added).
  - After provisioned transaction: 725 RPMs (344 new + 7 updated, 44/44 satisfied).
  - Target machine-ID SHA-256: `685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`.
  - DNF transaction 1: 351 altered (344 I, 7 U), Return-Code: Success, committed in 19 seconds.
  - Snapshots: `alm16-baseline-20261110` and `alm16-qualified-20261110`.

---

## 6. Idempotency and Second-Run Verification

### 6.1 Deterministic NO_OP Behavior
- On the provisioned system (`almadial-alm16-rpm`, 725 RPMs), running `lab-run` with `packages-inputs.json` resulted in:
  ```json
  {"no_op": true, "transaction": {"before_sha256": "...", "install": [], "remove": []}, "traces": []}
  ```
- **Exit Code:** 0.
- **Package Inventory:** Exactly 725 RPMs before and after. Zero unnecessary installs, updates, or removals.
- **Repository Hashes:** All 9 `/etc/yum.repos.d/*.repo` files remained byte-identical.
- **Completed Journal Resume:** Running `lab-resume` on the completed journal re-observed the installed state, confirmed postconditions, and exited 0 without transaction replay.

---

## 7. Interrupted-Transaction and Crash Recovery Verification

### 7.1 Recorded Real Interruption Evidence
- **Signal Delivery:** Controller sent SIGTERM during live transaction after observed RPM database writes.
- **Observed State at Interruption:**
  - RPM count at start: 381.
  - RPM count at signal: 386.
  - RPM count at stop: 387.
  - DNF transaction history recorded transaction 1 unfinished: `dt_end=0`, `state=0` (`[[1, 1794243616, 0, 0]]`).
  - Native apply exited with code 1.
- **Durable Journal Outcome:** Action `rpm.apply` recorded state `reconciliation_required` with `exit_code: -1`.
- **Replay Refusal:** Executing `lab-resume` against the interrupted journal:
  - Exited with code 1.
  - Emitted diagnostic: `almadial-rpm: action requires fresh reconciliation or operator intervention`.
  - Refused automatic replay of the non-idempotent package transaction.
  - Preserved RPM count (387) and DNF history without data corruption.
- **Snapshot Recovery:** The disposable VM was restored cleanly to `alm16-qualified-20261110` after evidence capture.

---

## 8. M1 Framework Integration Verification

### 8.1 Action Graph and Bounded Runner
- **Action Sequence:** Fixed actions in S01:
  - `rpm.trust`: Validates and imports pinned EPEL GPG key.
  - `rpm.resolve`: Calls native DNF solver in read-only mode to produce immutable transaction proposal.
  - `rpm.apply`: Verifies host preconditions, takes exclusive locks, re-checks proposal digest against live resolution, commits transaction, and verifies RPM postconditions.
- **Runner Boundaries:** Python invocation uses explicit argv (`/usr/bin/python3`, embedded script, explicit arguments). Execution limits enforced: 20-minute deadline, 4 MiB stdout limit, 64 KiB stderr limit, process group termination on cancellation.

### 8.2 Exclusive Locking and Journaling
- **Advisory File Lock:** Pre-flight lock acquisition on `apply.lock` via Linux `flock(LOCK_EX | LOCK_NB)` in 0700 state directory before any mutation.
- **Atomic Journal Transitions:** Durable state recorded before action execution (`started`), transitioning to `succeeded` only after independent RPM postcondition verification.

---

## 9. Native DNF Read-Only Side Effects Audit

### 9.1 SQLite Coordination Files
- **Files Checked:** 58 relevant filesystem paths.
- **Observed File Modifications During Read-Only Inspection:**
  - `/usr/lib/sysimage/rpm/rpmdb.sqlite-shm`
  - `/var/lib/dnf/history.sqlite-shm`
- **Classification:** `INCIDENTAL_SQLITE_READ_COORDINATION_NONBLOCKING`.
  - These `-shm` shared-memory files are temporary index coordination files updated by the SQLite engine even when opening databases in read-only mode (`file:...immutable=1` or read locks).
  - Zero persistent changes occurred to `/usr/lib/sysimage/rpm/rpmdb.sqlite`, `/var/lib/dnf/history.sqlite`, `/etc/yum.repos.d/`, or the DNF metadata cache.
  - Total installed RPM count remained exactly 725.

---

## 10. Test Suites, Regressions, and Reproducible Builds

### 10.1 Go Test Suites
- `go fmt ./...`: Exit 0 (all files properly formatted).
- `make fmt-check`: Exit 0.
- `go vet ./...`: Exit 0.
- `go mod tidy -diff`: Exit 0.
- `go mod verify`: Exit 0 (`all modules verified`).
- `go test ./...`:
  - Distinct Top-Level Passed: 125
  - Test/Subtest Pass Events: 644
  - Failures: 0
  - Skipped: 1 (`TestM0Artifacts`, optional M0 replay)
- `go test -race ./...`:
  - Distinct Top-Level Passed: 125
  - Test/Subtest Pass Events: 644
  - Race Conditions: 0

### 10.2 Python Verification Suites
- `scripts/validate_scaffold.py`: PASS (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: 131 tests passed in 1.874s (including 5 new native-adapter tests in `test_m2_01_native.py`).
- `scripts/verify_candidate_stack.py`: PASS.
- `scripts/verify_config_generation.py`: PASS.
- `git diff --check`: Exit 0 (whitespace clean).

### 10.3 CLI Smoke and PTY Wizard Tests
- `scripts/test_cli_smoke`: 17/17 cases passed (exit codes 0 and 2 verified).
- `scripts/test_terminal_wizard.py`: 11/11 interactive and unattended journeys passed.

### 10.4 Reproducible Builds
Built from two independent source copies (`a` and `b`) with controlled environment variables and metadata `35b439365729ab2bc0984a148ad6801e14d0258c-alm16-dirty`:

| Binary | Source Copy | Output SHA-256 | Match |
|---|---|---|---|
| `almadial` | `a` | `1eaf7bd52421b2b4928083894be5fc8a79276d0f1307536b46910b10b8606173` | **BYTE-IDENTICAL** |
| `almadial` | `b` | `1eaf7bd52421b2b4928083894be5fc8a79276d0f1307536b46910b10b8606173` | **BYTE-IDENTICAL** |
| `almadial-rpm` | `a` | `d6895beb4b8e7809514baee2ddf372012bb68361db6e3eb8c98c618ae71fe0b1` | **BYTE-IDENTICAL** |
| `almadial-rpm` | `b` | `d6895beb4b8e7809514baee2ddf372012bb68361db6e3eb8c98c618ae71fe0b1` | **BYTE-IDENTICAL** |

Outputs match the developer's reported binary hashes byte-for-byte.

---

## 11. Final Independent QA Verdict

### 11.1 Matrix Summary

| Criterion | Evaluation | Verdict |
|---|---|---|
| Approved Repositories | BaseOS, AppStream, CRB, Extras, and transaction-local EPEL | **PASS** |
| EPEL Restriction | Scoped to exact 6 Perl packages; zero repo file or global leak | **PASS** |
| GPG / TLS Verification | Pinned keys `c2a1e572` and `e37ed158` enforced; TLS verified | **PASS** |
| 44 Package Requirements | 44/44 SATISFIED on live VMs with exact EVRs and architectures | **PASS** |
| Exact Version Constraints | Exact version matching; missing versions fail closed | **PASS** |
| DNF Resolution & Proposal | Proposal SHA-256 `79eba1f6...` matches transaction 1 alters | **PASS** |
| Seven BaseOS Updates | Explicitly reviewed, authorized, and pinned in `upgrades.json` | **PASS** |
| RPM Postconditions | 725 RPMs verified; 44/44 verified with correct RSA headers | **PASS** |
| Idempotency (Second Run) | `no_op: true`, empty transaction, 0 package mutations | **PASS** |
| M1 Framework Integration | Reuses action graph, bounded runner, flock, journal | **PASS** |
| Interrupted Recovery | Stops at 387 RPMs; `reconciliation_required`; replay refused | **PASS** |
| Candidate Authorization | CANDIDATE manifest denied; `LabAuthorization` strictly scoped | **PASS** |
| SQLite Coordination Audit | Incidental `-shm` coordination classified as nonblocking | **PASS** |
| Zero Unrelated Mutation | Zero Asterisk/VICIdial/DB/firewall/SELinux mutations | **PASS** |
| Go Regression & Race | 125 passed, 644 subtest events, 0 race conditions, vet clean | **PASS** |
| Python Regression | 131 passed, scaffold and candidate stack checks pass | **PASS** |
| CLI Smoke & PTY Journeys | 17/17 CLI smoke, 11/11 PTY journeys passed | **PASS** |
| Reproducible Builds | Byte-identical Linux amd64 binaries across two builds | **PASS** |
| Evidence Ledger Integrity | Developer (258 files) and QA (17 files) ledgers sealed | **PASS** |

### 11.2 Acceptance Decision

**Verdict:** `ALMADIAL_M2_01_ACCEPTED`

All canonical acceptance criteria for **ALM-16 / M2-01** are satisfied. The implementation is accepted for local Git finalization and Kaneo status update.
