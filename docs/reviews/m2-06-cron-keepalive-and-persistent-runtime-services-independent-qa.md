# ALM-21 / M2-06 Independent Runtime and Reboot Persistence QA Report

- **Task ID:** `zoueucy7dz57wpf1jbbgwp8m` (ALM-21 / M2-06)
- **Task Title:** Configure cron, keepalive and persistent runtime services
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Milestone:** M2 — Native Backend Installation
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T21:15:00+08:00 (2026-11-10T13:15:00Z)
- **Repository Baseline Commit:** `0f36041d566e25699656160f502f73bafceec664` (Accepted ALM-20 commit)
- **Current HEAD:** `0f36041d566e25699656160f502f73bafceec664`
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-06/20261110T122348Z-alm21/` (Master ledger SHA-256: `dbcdb6a81d525814ef58b53169727b1414cff64490450548a73f22cbbc5a0493`, 448 entries verified)
- **Independent QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-06/qa-20261110T131204Z-alm21/` (Master ledger SHA-256: `73744d60b54eaece6a3a19ae9394d67b7cefe9fe7a4bfe816cc832a5660bbb25`, 10 entries verified)
- **Final Verdict:** `ALMADIAL_M2_06_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-21 (`zoueucy7dz57wpf1jbbgwp8m`), number 21 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No reassignment occurred.
- **Reviewer Identity:** Verified via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Initial Task Status:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Developer Comments:**
  - Scope decision comment `qwmqxczkksmuxlii474t0qsw` confirmed operator authorization of the minimal S06 contract (dialing inactive, FastAGI active, updater/listener configured but inactive, deferred cron examples).
  - Evidence comment `vgm8npg1icygf5dq1r6iyo11` verified; reported file counts, PIDs, boot IDs, and digests matched independent observations.
- **Dependency Graph:**
  - Source relation ALM-20 $\rightarrow$ ALM-21 (`aeplc7v5wibehiiu3h7wmh1j`): Verified intact (`blocks`), source ALM-20 (`jj53k7whedrsmfdamutw1wgz`) status `done` (Accepted).
  - Downstream relation ALM-21 $\rightarrow$ ALM-22 (`qcq551kz5z3jidx7jo99lp94`): Verified intact (`blocks`), target ALM-22 (`g1jf2c808sylx8twxhzas9oc`) status `to-do` (unassigned, unstarted).

### 1.2 Scope Discipline and Authorized Minimal Contract
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-21 implements Stage S06 (`internal/provision/runtime*.go`, `cmd/almadial-runtime/`), responsible for persistent runtime services, systemd supervision of FastAGI, configured-but-inactive manager updater and listener units, commented deferred cron definitions, and reboot persistence.
- **Minimal Contract Enforcement:**
  - Zero active application cron jobs are authorized or created.
  - Full `ADMIN_keepalive_ALL.pl` is deferred; it performs maintenance writes beyond supervision and would compete with systemd FastAGI.
  - `AST_VDhopper.pl` is deferred; campaign dialing is inactive.
  - Telephony dialing workloads, agent logins, outbound calls, RTP media, and WebRTC browser sessions remain excluded and deferred to subsequent milestones.

---

## 2. Developer Evidence Audit

### 2.1 Ledger Verification
- The developer ledger at `/home/ra/almadial-lab/evidence/m2-06/20261110T122348Z-alm21/SHA256SUMS` was independently hashed:
  - Expected SHA-256: `dbcdb6a81d525814ef58b53169727b1414cff64490450548a73f22cbbc5a0493`
  - Computed SHA-256: `dbcdb6a81d525814ef58b53169727b1414cff64490450548a73f22cbbc5a0493` (MATCH)
  - All 448 entries in the ledger were verified against the filesystem with `sha256sum -c`; 100% passed without errors.
- The directory and file permissions meet security isolation standards (`0700` directory, `0600` files).

---

## 3. FastAGI Runtime and MariaDB Socket Transport Verification

### 3.1 Live FastAGI Process Pool
Inspection on authorized VM `almadial-alm16-rpm` (`10.160.90.233`, machine ID `0fffa9a4683f35b696f0b6ee7330734b`):
- **Unit File:** `/etc/systemd/system/almadial-s06-fastagi.service` (mode `0644`, `root:root`).
- **Service State:** `active (running)`, `enabled` (preset: disabled).
- **Process Supervision:**
  - Exactly one parent: Main PID `1328`, PPID `1`, User `asterisk`, Group `asterisk`, Command `/usr/bin/perl /usr/share/astguiclient/almadial-s06-FastAGI_log.pl`.
  - Exactly five native `Net::Server::PreFork` worker children: PIDs `1342`, `1343`, `1344`, `1345`, `1346`, all PPID `1328`, User `asterisk`, Group `asterisk`.
  - CGroup: All 6 processes reside exclusively in `/system.slice/almadial-s06-fastagi.service`.
  - SELinux Domain: `system_u:system_r:unconfined_service_t:s0`.
- **Listening Socket:** Loopback TCP `127.0.0.1:4577`, bound exclusively to FastAGI perl processes. No public interfaces exposed.

### 3.2 MariaDB Socket Readiness Transport Correction
- **Discovered Defect:** The initial S06 implementation attempted TCP `127.0.0.1:3306` readiness, which failed against the accepted socket-local database grant (`almadial@localhost`).
- **Resolution Audit:**
  - The readiness helper `/usr/share/astguiclient/almadial-s06-ready.pl` (mode `0644`, `root:root`) was corrected to use Unix domain socket transport:
    `DBI->connect("DBI:mysql:database=asterisk;mysql_socket=/var/lib/mysql/mysql.sock", $c{VARDB_user}, $c{VARDB_pass}, ...)`
  - Reads credentials from protected `/etc/astguiclient-s06.conf` (`0640`, `root:asterisk`).
  - Executes bounded `SELECT 1` with a 10-second alarm and 5-second connect timeout.
  - Zero database privileges were broadened; MariaDB grant remains `SELECT,INSERT,UPDATE,DELETE,LOCK TABLES ON asterisk.*`.
  - Zero credentials leaked in process argv, unit files, or system logs.

---

## 4. Cron Configuration and Deferred Services

### 4.1 Cron File Integrity
- **Path:** `/etc/cron.d/almadial-s06` (mode `0644`, `root:root`, SELinux `system_cron_spool_t`).
- **Content:** Defines `SHELL=/bin/bash`, `PATH=/usr/sbin:/usr/bin:/sbin:/bin`, `CRON_TZ=UTC`, `MAILTO=""`.
- **Active Jobs:** Exactly **0 active application cron jobs**.
- **Deferred Upstream Examples:** Retained as commented documentation:
  `# * * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl`
  `# * * * * * root /usr/share/astguiclient/AST_VDhopper.pl -q`
- **User Spools:** Both `crontab -l -u root` and `crontab -l -u asterisk` confirm no user crontabs exist.

### 4.2 Deferred Systemd Units
- **Updater Unit:** `almadial-s06-updater.service` (mode `0644`, `root:root`).
- **Listener Unit:** `almadial-s06-listener.service` (mode `0644`, `root:root`).
- **State:** Both units are `inactive` and `disabled`.
- **Enforcement:** Both units specify `RefuseManualStart=yes`.
- **Live Refusal Check:**
  - `systemctl start almadial-s06-updater.service` $\rightarrow$ exit code 4 (`Failed to start ... Operation refused, unit ... configured to refuse manual start/stop`).
  - `systemctl start almadial-s06-listener.service` $\rightarrow$ exit code 4 (`Failed to start ... Operation refused`).

---

## 5. Reboot Persistence and Service Foundations

### 5.1 Reboot Verification
- **Authorized Single Reboot:** The operator authorized a single controlled VM reboot.
- **Boot ID Progression:**
  - Pre-reboot: `cdd8675e-8099-4f19-acc7-eed24dc1079b`
  - Post-reboot: `9ecd6163-6300-4898-8e93-262019adc5e1`
- **Snapshot Integrity:** Recovery snapshot `alm21-pre-runtime-20261110` is verified intact in Multipass.
- **No Manual Repair:** All services returned to operational state automatically following reboot without manual intervention.

### 5.2 Core Persistent Units
All six core system units are confirmed `active (running)` and `enabled`:
1. `mariadb.service`
2. `asterisk.service`
3. `httpd.service`
4. `php-fpm.service`
5. `crond.service`
6. `almadial-s06-fastagi.service`

### 5.3 Security and System Isolation
- **SELinux Mode:** `Enforcing`.
- **Audit Log Denials:** `ausearch -m avc -ts boot` confirmed zero AVC denials since boot (`<no matches>`).
- **Network Listeners:**
  - MariaDB: `127.0.0.1:3306` (loopback only)
  - Asterisk AMI: `127.0.0.1:5038` (loopback only)
  - Asterisk HTTP: `127.0.0.1:8088` (loopback only)
  - Apache HTTP: `127.0.0.1:80` (loopback only)
  - FastAGI: `127.0.0.1:4577` (loopback only)
  - SSH: `0.0.0.0:22` and `[::]:22` (public)
  - NTP: `127.0.0.1:323` and `[::1]:323` (chronyd)
  - SIP: Zero public SIP listeners (`pjsip show transports` $\rightarrow$ `No objects found.`).
- **Asterisk Channel State:** `core show channels` confirmed 0 active channels, 0 active calls, 0 calls processed.

---

## 6. Baseline Preservation

Live verification on the VM confirmed 100% bit-for-bit preservation of all accepted historical baselines:
- **Database Engine & Version:** MariaDB `10.11.18-MariaDB`.
- **Database Schema Version:** `1748` (`SELECT db_schema_version FROM asterisk.system_settings;`).
- **Database Catalog SHA-256:** `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` (345 tables, 1,011 indexes, exact match).
- **Legacy Seed-Phone Records:** Both records (`callin` and `gs102`) intact across all 107 columns:
  - Combined SHA-256: `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04` (exact match).
- **Installed RPM Packages:** 725 packages:
  - Inventory SHA-256: `bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44` (exact match).
- **Recording Storage:** `/var/spool/asterisk/monitor` and `/var/spool/asterisk/monitorDONE` preserved with mode `0750` owned by `asterisk:asterisk`.

---

## 7. Stage S06 Architecture and Fresh NO_OP Verification

### 7.1 Stage Execution Model
- S06 defines a structured, three-action execution graph:
  1. `runtime.00.files`: Provisions owned config, cron, scripts, and unit files with atomic create-if-absent semantics.
  2. `runtime.01.services`: Reloads systemd, enables persistent units, starts FastAGI.
  3. `runtime.02.verify`: Inspects process health, cgroups, user identities, sockets, and database readiness.
- Preceded by exclusive `journal.Acquire` locking and durable intent recording.

### 7.2 Fresh NO_OP Assessment
An independent read-only inspection and planning cycle was executed live on the VM:
- `almadial-runtime inspect`: Exited 0, snapshot fully populated.
- `almadial-runtime plan`: Exited 0, returned empty graph `[]` with 0 actions.
- Proves genuine idempotency: no file rewrites, no systemd daemon-reloads, no service restarts, and no PID churn on healthy systems.

---

## 8. Test Regressions and Reproducible Builds

### 8.1 Go 1.27.3 Regressions
- `go vet ./...`: Exited 0 (clean).
- `go test ./...`: 200 top-level tests, 1,318 pass events (exit 0).
- `go test -race ./...`: 100% pass without data races (exit 0).
- `go mod verify`: `all modules verified` (exit 0).
- `go mod tidy -diff`: Clean (exit 0).

### 8.2 Python Regressions
- `python3 scripts/validate_scaffold.py`: PASS (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: 181 tests passed (exit 0).
- `python3 scripts/verify_candidate_stack.py`: PASS.
- `python3 scripts/verify_config_generation.py`: PASS.
- `scripts/test_terminal_wizard.py`: 11 executable terminal/nonterminal journeys PASS.
- `git diff --check`: Clean (no whitespace or formatting errors).

### 8.3 Reproducible Linux amd64 Builds
Independent builds on host verified byte-for-byte reproducibility:
- **Main CLI (`almadial`):**
  - Tool: `scripts/build_go.py`
  - Computed SHA-256: `1a9a15c5a1edb21ad7d36c41c375d90d38650d61cf136b4a2d5e7d10841c3b9a`
  - Re-build SHA-256: `1a9a15c5a1edb21ad7d36c41c375d90d38650d61cf136b4a2d5e7d10841c3b9a` (100% byte-identical)
- **Runtime Qualification CLI (`almadial-runtime`):**
  - Tool: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly -trimpath -buildvcs=false ./cmd/almadial-runtime`
  - Computed SHA-256: `c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a`
  - Re-build SHA-256: `c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a` (100% byte-identical)

---

## 9. Independent Acceptance Verdict

All canonical requirements and acceptance criteria for **ALM-21 / M2-06** have been met:
1. FastAGI runs reliably under native systemd supervision with 1 parent and 5 prefork worker children as `asterisk:asterisk` on loopback `127.0.0.1:4577`.
2. FastAGI MariaDB readiness operates securely via Unix domain socket `/var/lib/mysql/mysql.sock` without TCP broadening or credential leaks.
3. Asterisk updater and manager listener units are configured, disabled, inactive, and enforce `RefuseManualStart=yes`.
4. Cron configuration `/etc/cron.d/almadial-s06` enforces 0 active application jobs with deferred examples retained as documentation.
5. Persistent services survive reboot automatically without manual repair.
6. MariaDB 10.11.18, schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 725 RPMs, storage, and SELinux Enforcing remain 100% intact.
7. Genuine fresh NO_OP behavior verified live.
8. Go and Python regression suites and reproducible builds pass with zero defects.

**Final Verdict:** `ALMADIAL_M2_06_ACCEPTED`

---

## 10. Handoff and Downstream Work Package

- **Task Status:** Update Kaneo task **ALM-21** (`zoueucy7dz57wpf1jbbgwp8m`) to **Accepted** (`done`).
- **Task Ownership:** Retain Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) as the implementation owner.
- **Downstream Task:** **ALM-22 / M2-07** (`g1jf2c808sylx8twxhzas9oc`): "Validate safe recovery across native backend stages" remains in **Backlog** (`to-do`), unassigned, and unstarted.
