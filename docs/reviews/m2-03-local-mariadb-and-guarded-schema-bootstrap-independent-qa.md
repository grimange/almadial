# ALM-18 / M2-03 Independent MariaDB and VICIdial Schema Bootstrap QA Report

- **Task ID:** `ldl2ak2bk3pcq7bwvk9w54wc` (ALM-18 / M2-03)
- **Task Title:** Provision local MariaDB and guarded schema bootstrap
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T05:46:00+08:00 (2026-11-09T21:46:00Z)
- **Repository Baseline Commit:** `39dcfb4f965c4a9a5cdf351b4290702b974bf4ac` (Accepted ALM-17 commit)
- **Developer Baseline HEAD:** `a8ba1afecc54ead750c6b7c4c992cca832e1d723`
- **Current Working HEAD:** `a8ba1afecc54ead750c6b7c4c992cca832e1d723`
- **Final Verdict:** `ALMADIAL_M2_03_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-18 (`ldl2ak2bk3pcq7bwvk9w54wc`), number 18 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No task reassignment or self-assignment occurred.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Task Status Prior to Review:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Implementation Evidence Comment:** Verified comment `bjpxz8lzoklzb8p6920f7f14` posted by Codex on 2026-11-09T21:29:04Z matches recorded implementation details and digests.
- **Dependency Graph:**
  - Relation ALM-17 $\rightarrow$ ALM-18 (`wr227vtlq2zmy1omyu24e57d`): Verified intact (`blocks`), source ALM-17 (`ou5231ijs6ilp28howhd6xs1`) status `done` (Accepted).
  - Relation ALM-18 $\rightarrow$ ALM-20 / M2-05 (`ilb64crk31gw45vg3dh2ok9z`): Verified intact (`blocks`), target ALM-20 (`jj53k7whedrsmfdamutw1wgz`) status `to-do` (Backlog, unassigned).
  - Relation ALM-17 $\rightarrow$ ALM-19 (`s8822axs1a6cmyxg9fa698eg`): Verified intact (`blocks`), target ALM-19 (`nwk5zbmgwuuhsnlsh92sxl1v`) status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-18 implements Stage S03 (`internal/provision/database.go`, `database_source.go`, `database_native_linux.go`, `database_stage_linux.go`, `database_files.py` and `cmd/almadial-db`), responsible for local MariaDB instance configuration, protected credentials, application database grants, verified initial VICIdial schema bootstrap, and partial-DDL recovery.
- **Separation of Concerns:** ALM-18 does not build Asterisk, compile PJPROJECT, install VICIdial application files, deploy VICIphone, configure Apache/PHP, or start telephony services.
- **Candidate Manifest Authorization:** `manifests/stack.lock.yaml` remains strictly `candidate`. `InstallationAuthorized()` continues to deny production installation. Laboratory provisioning in `cmd/almadial-db` requires explicit, cryptographically bound `DatabaseLabAuthorization` with scope `ALM-18/disposable-lab`.

---

## 2. Protected Evidence Ledger Verification

### 2.1 Developer Evidence Ledger
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-03/20261110T045300-alm18/`
- **Master Ledger:** `SHA256SUMS`
  - Computed SHA-256: `f80b189a06ceff5191c51659d1e988080a1f507885cae5c24ff4c11cfcfec781` (exact match)
  - File Count: 155 entries
  - Verification: `sha256sum -c SHA256SUMS` executed with 100% OK across all 155 files.
- **Handoff Ledger:** `handoff/SHA256SUMS`
  - Computed SHA-256: `917b98827e5c0c4d55e81771a5b578baf9ca4939df60a8978eab1da84c65b02a` for `mcp-receipt.json` (exact match).
  - Verification: `sha256sum -c SHA256SUMS` executed with 100% OK.
- **Task Source Manifest:** `source-sha256.json`
  - All 11 task-owned files independently verified byte-for-byte against the working tree.

### 2.2 Independent QA Evidence Ledger
- **QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-03/qa-20261109T214500Z/` (mode `0700`, files `0600`)
- **Master Ledger SHA-256:** `5f53cabf333494c1a06675ad2987a24b160b063f1353a8e46bd5a5f64cc746ce`
- **Artifacts Preserved (16 files):**
  1. `starting-head.txt`
  2. `developer-ledger-verification.txt`
  3. `task-source-sha256.json`
  4. `vm-mariadb-service-and-listener.txt`
  5. `vm-credentials-and-grants.json`
  6. `vm-schema-catalog-verification.txt`
  7. `vm-seed-data-verification.json`
  8. `vm-recovery-and-idempotency.json`
  9. `vm-baseline-preservation.txt`
  10. `go-test-results.json`
  11. `go-race-results.json`
  12. `python-test-results.json`
  13. `cli-smoke-results.json`
  14. `pty-journey-results.json`
  15. `reproducible-builds.json`
  16. `kaneo-baseline.json`

---

## 3. S03 / S05 Schema Ownership Contract

### 3.1 S03 as Sole Authoritative Initial Schema Writer
- **Exclusive Bootstrap Responsibility:** S03 owns initial database creation, local configuration, credentials, user grants, verified initial SQL execution (`extras/MySQL_AST_CREATE_tables.sql` derived + `extras/first_server_install.sql`), and seed data validation.
- **Deterministic SQL Derivation:** Matches the reviewed M0 Python derivation (`scripts/prepare_vicidial_r4037.py`) by removing exactly three legacy global privilege statements from upstream CREATE SQL, avoiding global grant pollution while preserving all upstream tables and indexes.

### 3.2 Non-Mutating S05 Verification Contract
- **Read-Only Verification:** S05 consumes the schema exclusively via `VerifyDatabaseForApplication(s DatabaseSnapshot, source DatabaseSource) error`.
- **Zero Mutating Paths:** `VerifyDatabaseForApplication` is pure and cannot invoke initial bootstrap, drop tables, reimport SQL, reset seed values, or rotate credentials.
- **Incomplete Schema Rejection:** If any schema object, index, seed table, or credential binding is incomplete, `VerifyDatabaseForApplication` returns `errDatabase`, halting S05 before application deployment.

---

## 4. MariaDB Instance and Local Transport Verification

### 4.1 Service and Configuration Inspection
Live VM inspection on `almadial-alm16-rpm` (`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`):

| Property | Value / Status | Verification Method | Finding |
|---|---|---|---|
| Package Baseline | `3:10.11.18-1.el10_2.alma.2.x86_64` | `rpm -q mariadb-server` | PASS |
| Service State | `active (running)`, `enabled` | `systemctl status mariadb` | PASS |
| Config File | `/etc/my.cnf.d/zz-almadial.cnf` | `stat -c '%U:%G %a %C'` (`root:mysql 640 mysqld_etc_t:s0`) | PASS |
| Data Directory | `/var/lib/mysql` | `stat -c '%U:%G %a %C'` (`mysql:mysql 755 mysqld_db_t:s0`) | PASS |
| Unix Socket | `/var/lib/mysql/mysql.sock` | `stat -c '%U:%G %a %C'` (`mysql:mysql 777 mysqld_var_run_t:s0`) | PASS |
| SELinux Status | `Enforcing`, `targeted` | `getenforce`, `sestatus`, `matchpathcon -V` | PASS |

### 4.2 Network Isolation and Transport Boundary
- **Loopback-Only TCP Listener:** `ss -lntp` confirms `127.0.0.1:3306` is the sole TCP listener for `mariadbd`. Zero external IPv4 or IPv6 listening endpoints exist.
- **Local Unix Socket Transport:** Application connectivity uses `/var/lib/mysql/mysql.sock`. Under `skip-name-resolve`, TCP application authentication is intentionally rejected, preventing network-based bypass.

---

## 5. Protected Credentials and Database Grants

### 5.1 Credential Metadata and Isolation
All credentials reside under `/etc/almadial/secrets/` with strict mode `0600` and `root:root` ownership:
- `/etc/almadial/secrets/database`: Unique 48-hex application password.
- `/etc/almadial/secrets/admin`: Unique 48-hex initial VICIdial admin password.
- `/etc/almadial/secrets/database-client.cnf`: Private client option file.
- **Zero Disclosure:** No credentials appear in command-line arguments, process logs, journals, or test outputs. The execution runner redacts registered secrets.

### 5.2 Application Account and Effective Grants
Administrative queries against `mysql.user` and `information_schema.SCHEMA_PRIVILEGES`:
- `almadial@localhost` is the sole application database account.
- **Effective Grants on `asterisk.*`:** `SELECT, INSERT, UPDATE, DELETE, LOCK TABLES`.
- **Zero Global Privileges:** `information_schema.USER_PRIVILEGES` confirms only `USAGE` at global scope. Zero `GRANT OPTION`, zero routine/table/column privilege escalation.

---

## 6. Pinned VICIdial Schema and Catalog Verification

### 6.1 Schema Metadata Audit
Live inspection of the provisioned `asterisk` database:
- **Schema Version:** `1748` (verified via `system_settings.db_schema_version`).
- **Application Version String:** `2.14b0.5`.
- **Total Tables:** `345` tables.
- **Total Index Metadata Rows:** `1,011` index rows in `information_schema.STATISTICS`.
- **Other Schema Objects:** 0 views, 0 stored routines, 0 triggers.
- **Character Set & Collation:** `utf8mb3` / `utf8mb3_unicode_ci`.
- **SQL Mode:** `STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.

### 6.2 Deterministic Full Catalog Digest
- The canonical catalog query combines JSON-serialized definitions of all tables, columns, indexes, views, routines, and triggers ordered by binary name.
- **Expected Digest:** `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`.
- **Observed Live Digest:** `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`.
- **Result:** Exact 100% match.

### 6.3 Seed Data Verification
- **Populated Seed Tables:** All 33 designated seed tables were verified nonempty.
- **Required Reference Data:**
  - `vicidial_users WHERE user='6666'`: 1 row (password matches `/etc/almadial/secrets/admin`).
  - `vicidial_user_groups WHERE user_group='ADMIN'`: 1 row.
  - `vicidial_statuses WHERE status IN ('NEW','SALE')`: 2 rows.
  - `vicidial_qc_codes WHERE code IN ('QCPASS','QCFAIL','QCCANCEL')`: 3 rows.
  - `servers WHERE server_id='TESTast'`: 1 row.

---

## 7. Review of Corrected Postcondition Defects

Codex reported two postcondition defects encountered during developer qualification:
1. **Disabled Log Variable Representation:** In MariaDB 10.11, `@@general_log`, `@@slow_query_log`, and `@@log_bin` return the string `OFF` rather than numeric `0`. The initial check expected `0`. Correcting this to `OFF|OFF|OFF` strictly ensures that database logging is disabled to prevent credential leakage in query/binary logs.
2. **Seed User Group Key:** The initial seed check looked for `---ALL---` in `vicidial_user_groups`, whereas the pinned r4037 seed inserts group `ADMIN`. Correcting the check to `ADMIN` strictly matches the authoritative source.

**Independent QA Verdict on Corrections:** Both corrections aligned postcondition assertions with authoritative runtime and source behaviors, did not weaken validation, and verified that incomplete states are rejected.

---

## 8. Recovery, Interrupted DDL, and Idempotency

### 8.1 Interrupted DDL Handling
- Tested in an isolated test database (`almadial_alm18_test`).
- When DDL was interrupted by timeout, the database state was correctly classified as `PARTIAL`.
- `lab-resume` required manual reconciliation and refused to blindly replay DDL against a partial schema.
- The isolated test database was cleaned up under explicit journaled authorization without affecting the primary `asterisk` database.

### 8.2 Missing-Column Rejection
- Dropping a required column altered the catalog fingerprint, transitioning state to `PARTIAL`.
- `plan` and `VerifyDatabaseForApplication` deterministically blocked execution.

### 8.3 Restart Persistence and Fresh NO_OP
- Following MariaDB service restart (with verified invocation ID change), all 345 tables, 1,011 indexes, and 33 seed table counts remained intact.
- A fresh inspection produced zero required actions (`no_op: true`).
- Executing `lab-resume` on the completed journal returned `no_op: true` with zero mutating actions.

---

## 9. Baseline Preservation and Host State

- **RPM Package Inventory:** Exactly 725 installed RPMs before and after ALM-18. Zero packages installed, removed, or updated.
- **Repository Trust:** All 9 AlmaLinux repository files in `/etc/yum.repos.d/` preserved untouched.
- **DNF Transaction History:** Exactly 1 transaction recorded (from ALM-16). Zero new DNF transactions.
- **Filesystem & System Identities:** All 16 directories from ALM-17 and system identities (`apache`, `mysql`, `asterisk`) preserved intact.
- **Storage Reserve:** Available storage on `/dev/sda4` is 81.2 GB and 41.2M inodes, exceeding the approved 1 GiB / 10,000 inodes lab reserve.

---

## 10. Regression Testing and Reproducibility

### 10.1 Go Verification Suites (Go 1.27.3)
- `gofmt -l cmd internal`: Clean (exit code 0).
- `go vet ./...`: Clean (exit code 0).
- `go mod verify`: Clean (`all modules verified`).
- `go test -json ./...`:
  - Top-Level Passed: 145 (146 total, 1 skipped: `TestM0Artifacts`).
  - Pass Events: 804.
  - Failures: 0.
- `go test -race -json ./...`:
  - Top-Level Passed: 145.
  - Pass Events: 804.
  - Race Conditions: 0.

### 10.2 Python Verification Suites
- `python3 scripts/validate_scaffold.py`: PASS (11 files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: 131 tests passed in 1.875s (OK).
- `python3 scripts/verify_candidate_stack.py`: PASS.
- `python3 scripts/verify_config_generation.py`: PASS.
- `git diff --check`: Clean (no whitespace errors or conflict markers).

### 10.3 CLI Smoke and PTY Journeys
- **CLI Smoke Suite:** 17/17 cases passed (0 secret leaks).
- **PTY Wizard Journeys:** 11/11 journeys passed across real Linux PTYs (8 interactive + 3 nonterminal).

### 10.4 Reproducible Builds (Linux amd64)
Two isolated builds from clean directories yielded identical SHA-256 hashes matching developer evidence:
- `almadial`: `95e3045bde6c6d960e7ab091e5ff4162db109912c7e5cd4a57f4a4234a4f5b33`
- `almadial-rpm`: `155117a46878d6af7fbb30ac67d24dc1d820dc875314c5812d6e16bb130d6810`
- `almadial-system`: `7b0155dca7db626d22f85f87fafbc134698e76267fd5802bea00a407318aad05`
- `almadial-db`: `527a47938928aa8fc3ea3190269c676b6b40db72019e08f2a653a1b9145ab3e9`

---

## 11. Independent QA Summary Matrix

| Verification Area | Requirement / Check | Observed Result | Status |
|---|---|---|---|
| Schema Ownership | S03 authoritative initial schema writer | Initial DDL/seeds owned by S03; M0 derivation verified | **PASS** |
| Schema Ownership | Non-mutating S05 verification contract | `VerifyDatabaseForApplication` pure, read-only | **PASS** |
| MariaDB Provisioning | Package baseline & version | MariaDB 10.11.18 (`3:10.11.18-1.el10_2.alma.2.x86_64`) | **PASS** |
| MariaDB Provisioning | Local loopback listener & socket | `127.0.0.1:3306` only; Unix socket `/var/lib/mysql/mysql.sock` | **PASS** |
| MariaDB Provisioning | Configuration file ownership & mode | `/etc/my.cnf.d/zz-almadial.cnf` (`root:mysql 0640 mysqld_etc_t`) | **PASS** |
| Security | Protected credentials storage | `/etc/almadial/secrets/*` (`root:root 0600 etc_t`), 0 leaks | **PASS** |
| Security | Application database grants | `almadial@localhost` (SELECT, INSERT, UPDATE, DELETE, LOCK TABLES) | **PASS** |
| Security | Privilege escalation prevention | Zero global grants, zero GRANT OPTION, zero external hosts | **PASS** |
| Schema Verification | Pinned r4037 source provenance | Verified r4037 archive & exact M0 derived SQL statements | **PASS** |
| Schema Verification | Schema version & table count | Version 1748; 345 tables; 1,011 index metadata rows | **PASS** |
| Schema Verification | Deterministic catalog digest | Full catalog SHA-256 `10f5b889...` verified 100% | **PASS** |
| Schema Verification | Seed table row counts & keys | 33 seed tables nonempty; key seed records & admin pass verified | **PASS** |
| Schema Verification | Character set, collation, SQL mode | `utf8mb3` / `utf8mb3_unicode_ci`; strict SQL mode | **PASS** |
| Recovery | Interrupted DDL handling | PARTIAL classification; automatic replay blocked | **PASS** |
| Recovery | Missing-column rejection | Catalog mismatch rejects schema as PARTIAL | **PASS** |
| Recovery | Complete schema reconciliation | Complete database reconciles without reimporting | **PASS** |
| Idempotency | Service restart persistence | Schema & seed counts survive service restart | **PASS** |
| Idempotency | Fresh assessment & resume NO_OP | 0 required actions, empty graph, no-op execution | **PASS** |
| Baseline | ALM-16/17 baseline preservation | 725 RPMs, 9 repos, 1 DNF tx, 16 directories, SELinux Enforcing | **PASS** |
| Test Suites | Go test suite & race detector | 145 top-level passed, 804 pass events, clean race detector | **PASS** |
| Test Suites | Python unit & validation suites | 131 tests passed; scaffold, stack, config-gen clean | **PASS** |
| CLI & Journeys | CLI smoke & PTY wizard journeys | 17 smoke passed; 11 PTY journeys passed | **PASS** |
| Toolchain | Reproducible Go builds | 4/4 binaries byte-identical across isolated builds | **PASS** |
| Evidence | Protected QA evidence ledger | 16 artifacts, SHA-256 sealed | **PASS** |

---

## 12. Final Verdict

All canonical acceptance criteria for ALM-18 / M2-03 are satisfied. S03 is established as the sole initial schema writer, S05 cannot implicitly repeat bootstrap, MariaDB is strictly confined to local transport, database credentials and application grants are properly isolated, the pinned r4037 schema is completely verified, interrupted DDL is safely handled, restart persistence is established, and repeat execution produces fresh NO_OP.

**Verdict:**

`ALMADIAL_M2_03_ACCEPTED`
