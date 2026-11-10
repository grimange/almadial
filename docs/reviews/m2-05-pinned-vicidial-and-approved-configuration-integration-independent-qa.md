# ALM-20 / M2-05 Independent VICIdial Application Provisioning and Configuration Integration QA Report

- **Task ID:** `jj53k7whedrsmfdamutw1wgz` (ALM-20 / M2-05)
- **Task Title:** Provision pinned VICIdial and approved configuration integration
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Milestone:** M2 — Native Backend Installation
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T19:37:00+08:00 (2026-11-10T11:37:00Z)
- **Repository Baseline Commit:** `34509d29c21e519a0cea9253234d71b7b9e22cee` (Accepted ALM-19 commit)
- **Current HEAD:** `34509d29c21e519a0cea9253234d71b7b9e22cee`
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-05/20261110T025748Z-alm20/` (Master ledger SHA-256: `784555db98ea5e66783f1b78fff4a83be63497e5e52618f68a2c5f549ddf2428`, 220 entries verified)
- **Independent QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-05/qa-20261110T113600Z-alm20/` (Master ledger SHA-256: `ea3564bab92384bf7c8412b7876e39af3531ff8397e92531b857707aad2e5911`, 16 entries verified)
- **Final Verdict:** `ALMADIAL_M2_05_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-20 (`jj53k7whedrsmfdamutw1wgz`), number 20 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the task implementation owner. No reassignment or self-assignment occurred.
- **Reviewer Identity:** Verified via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Initial Task Status:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Developer Evidence Comment:** Verified comment `bg05gj20ml1igbfoaiezowh4` posted by Codex on 2026-11-10T04:47:45.300Z matches recorded implementation details, file counts, and digests.
- **Dependency Graph:**
  - Relation ALM-18 $\rightarrow$ ALM-20 (`ilb64crk31gw45vg3dh2ok9z`): Verified intact (`blocks`), source ALM-18 (`ldl2ak2bk3pcq7bwvk9w54wc`) status `done` (Accepted).
  - Relation ALM-19 $\rightarrow$ ALM-20 (`w2s0nv3r754tafaacmf18zd2`): Verified intact (`blocks`), source ALM-19 (`nwk5zbmgwuuhsnlsh92sxl1v`) status `done` (Accepted).
  - Relation ALM-20 $\rightarrow$ ALM-21 / M2-06 (`aeplc7v5wibehiiu3h7wmh1j`): Verified intact (`blocks`), target ALM-21 (`zoueucy7dz57wpf1jbbgwp8m`) status `to-do` (Backlog, unstarted).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-20 implements Stage S05 (`internal/provision/vicidial*.go`, `vicidial_files.py`, and `cmd/almadial-vicidial`), responsible for deploying the pinned VICIdial r4037 application files, pure-Perl `Net::Telnet` 3.05, establishing owned Apache and Asterisk configuration includes, verifying S03 schema preservation, guarded registration of the local application and AMI probe account, verifying Asterisk 40-module runtime health, Apache/PHP-FPM endpoint responses, Perl/AGI non-root loading, and ensuring clean idempotency (NO_OP) and crash-checkpoint recovery.
- **Separation of Concerns:** ALM-20 does not perform live SIP phone calls, agent logins, outbound dialer campaigns, RTP audio streaming, or public WebRTC exposure. Real endpoint configuration is explicitly assigned to M3.
- **Candidate Manifest Authorization:** `manifests/stack.lock.yaml` remains strictly `candidate`. `InstallationAuthorized()` continues to deny production installation. Laboratory provisioning in `cmd/almadial-vicidial` requires explicit, cryptographically bound `VICIdialLabAuthorization` with scope `ALM-20/disposable-lab`.

---

## 2. Source Provenance and Archive Verification

### 2.1 Pinned VICIdial Source
- **Upstream Project:** VICIdial SVN revision `r4037`, application version `2.14b0.5`.
- **Source Archive:** `vicidial-r4037.zip`
  - SHA-256: `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` (exact match)
  - Canonical Inventory: exactly 2,511 regular files.
  - Tree SHA-256: `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` (exact match)
  - Security Validations: Duplicate name rejection, relative path cleaning, strict size bounds (256 MB per file, 2 GB total), and non-executable extraction checks all verified.

### 2.2 Pinned Pure-Perl Dependency
- **Component:** `Net::Telnet` version 3.05.
- **Source Archive:** `Net-Telnet-3.05.tar.gz`
  - SHA-256: `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` (exact match)
  - Installed directly to `/usr/local/share/perl5/5.40/Net/Telnet.pm` without executing `Makefile.PL`, CPAN, or modifying distribution RPM packages.

---

## 3. Installed Application and Deployment Inventory

### 3.1 Exact 2,153-File Deployment
The loader deterministically selects 2,153 application files from the 2,511 source entries:
- **Web files (1,349):** Destination `/var/www/html/`, group `apache`, mode `0640`, context `httpd_sys_content_t`.
- **Application scripts (135):** Destination `/usr/share/astguiclient/`, group `root`, mode `0755`, context `bin_t`.
- **AGI files (69):** Destination `/var/lib/asterisk/agi-bin/`, group `asterisk`, mode `0750`, context `bin_t`.
- **Sounds (471):** Destination `/var/lib/asterisk/sounds/`, group `asterisk`, mode `0640`, context `var_lib_t`.
- **Documentation, libraries & attribution (129):** Destination `/usr/share/doc/almadial-vicidial/` and `/usr/local/share/perl5/5.40/`.

### 3.2 Excluded Files Policy (358 Files)
Exactly 358 source entries are not deployed into runtime:
- Upstream installation scripts (`install.pl`) not used.
- Upstream SQL schema bootstrap files (`*.sql`) excluded (preserving S03 ownership).
- Unused sample configurations and legacy MeetMe/SIP templates.
- Non-runtime source repository artifacts.

### 3.3 Bounded Procps Script Adjustments
Exactly four Perl scripts receive the previously qualified procps command adjustment (`ps -o pid,args -ww --no-headers -A` replacing the deprecated `-o "%p %a"` format):
1. `bin/ADMIN_keepalive_ALL.pl` (SHA-256 `4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67`)
2. `bin/AST_manager_send.pl` (SHA-256 `83be4e5ca25591adc6991273b84114b53164ef49b7bf04bee9e9b7413b7cb384`)
3. `bin/AST_VDauto_dial.pl` (SHA-256 `cd2115f06d8c7730af73d67d468612c4ac127f815b31689a3112b374a14d9d87`)
4. `bin/AST_VDremote_agents.pl` (SHA-256 `0187ca67f99c0b435280fe76a2cfd2c4816ad7f82411f2112bc988c37ee4a596`)
All four source preimages and replacements were verified byte-for-byte.

---

## 4. Database Preservation and S03/S05 Ownership

### 4.1 Schema and Catalog Verification
- **Database Engine:** MariaDB `10.11.18-MariaDB` (socket-only).
- **Database Name:** `asterisk`
- **Schema Version:** `1748` (`SELECT db_schema_version FROM asterisk.system_settings;`).
- **Catalog Fingerprint:** `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` (matches ALM-18 and ALM-19).
- **Structure:** 345 tables, 1,011 index metadata rows.
- **S03 Ownership:** S05 consumed and verified the database via `VerifyDatabaseForApplication` without reimporting DDL, re-running seeds, altering grants, or rotating credentials.

### 4.2 Legacy Seed-Phone Record Preservation
- Both legacy seed-phone records in `asterisk.phones` are 100% intact across all 107 columns:
  - `callin`: dialplan `8300`, voicemail `8300`, `Dial-in agent phone`, server `10.10.10.15`, protocol `EXTERNAL`.
  - `gs102`: dialplan `102`, voicemail `102`, `Grandstream BT 102`, server `10.10.10.15`, protocol `SIP`.
- **Combined SHA-256:** `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04` (exact match across all 107 columns).
- Remote server reference `10.10.10.15` strictly preserved. No live phone endpoints generated.

---

## 5. Asterisk and AMI Integration

### 5.1 Runtime and Module Preservation
- **Asterisk Version:** Asterisk 22.12.0 (`asterisk -rx "core show version"`).
- **Native Modules:** All 40 required native modules confirmed loaded and running (`Running` status in `module show`).
- **PJSIP Transports:** `asterisk -rx "pjsip show transports"` confirmed `No objects found.`. Zero public SIP listeners.
- **ConfBridge Profile:** `default_bridge` verified active.

### 5.2 AMI Authentication and Least-Privilege Verification
- **Binding:** Strictly loopback `127.0.0.1:5038`.
- **Account:** `almadial-s05` with `read=system` and `write=system`.
- **Credential:** Stored privately at `/etc/almadial/secrets/vicidial-ami` (`0600`, `root:root`).
- **Probes Verified Live on VM:**
  - `Action: Login` $\rightarrow$ `Response: Success` (`Authentication accepted`)
  - `Action: Ping` $\rightarrow$ `Response: Success` (`Pong`)
  - `Action: CoreStatus` $\rightarrow$ `Response: Success`
  - `Action: Command` (`core show version`) $\rightarrow$ `Response: Error` (`Permission denied`)
  - `Action: Logoff` $\rightarrow$ `Response: Goodbye`

---

## 6. Apache/PHP-FPM Integration and Defect Resolutions

### 6.1 HTTP Endpoint Behavior
- **Apache/PHP-FPM:** Operational, listening strictly on `127.0.0.1:80`.
- **Admin Endpoint:** `http://127.0.0.1/vicidial/admin.php` returns HTTP **401** (`Basic realm="CONTACT-CENTER-ADMIN"`).
- **Agent Entry Endpoint:** `http://127.0.0.1/agc/vicidial.php` returns HTTP **200** (VICIdial Agent phone login screen).
- **Protected Options:** `http://127.0.0.1/vicidial/options.php` and `http://127.0.0.1/agc/options.php` return HTTP **403** (Forbidden).

### 6.2 First-Request Web-Key Defect Resolution
- **Issue:** In the initial developer run, upstream PHP code populated `system_settings.pass_key` upon receiving the first HTTP request, causing post-install generator input drift.
- **Resolution:** S05's guarded registration action (`vicidial.03.register`) detects blank `pass_key` and initializes it with cryptographically secure random bytes in the database under table write locks before web probes execute. Existing non-blank keys are preserved. Verified on the live VM with zero postcondition drift.

### 6.3 AGI Perl Module-Path Defect Resolution
- **Issue:** Upstream `extras/Asterisk/AGI.pm` was initially placed under the application library directory, preventing unprivileged runtime processes from resolving `Asterisk::AGI`.
- **Resolution:** `extras/Asterisk.pm` and `extras/Asterisk/AGI.pm` are installed to `/usr/local/share/perl5/5.40/`, which is in the standard Perl 5.40 `@INC` search path.
- **Live Verification:** `sudo -u asterisk perl -MAsterisk::AGI -e '...'` and `sudo -u asterisk perl -MNet::Telnet -e '...'` both executed with exit code 0.

---

## 7. Perl/AGI and PHP Runtime Syntax Checks

- **Perl Syntax (`perl -c`):** Exactly 9 core scripts verified clean:
  1. `ADMIN_keepalive_ALL.pl` (syntax OK)
  2. `AST_update.pl` (syntax OK)
  3. `AST_update_AMI2.pl` (syntax OK)
  4. `AST_manager_listen.pl` (syntax OK)
  5. `AST_manager_send.pl` (syntax OK)
  6. `AST_VDauto_dial.pl` (syntax OK)
  7. `AST_VDadapt.pl` (syntax OK)
  8. `AST_conf_update.pl` (syntax OK)
  9. `FastAGI_log.pl` (syntax OK)
- **PHP Syntax (`php -l`):** All 309 PHP files (307 source + 2 options) verified syntax clean (0 errors).

---

## 8. Security, Service, and Network Isolation

- **SELinux:** Current mode is `Enforcing`, policy `targeted`.
  - Process domains: Asterisk (`unconfined_service_t`), MariaDB (`mysqld_t`), Apache & PHP-FPM (`httpd_t`).
  - Audit log query (`ausearch -m avc -ts boot`) confirmed **0 AVC denials** since boot.
- **Network Isolation:** Only SSH (`0.0.0.0:22`, `[::]:22`) and ephemeral UDP ports are exposed outside localhost. All application listeners (3306, 5038, 8088, 80) strictly bound to `127.0.0.1`.

---

## 9. Execution, Locking, Recovery, and Fresh NO_OP

- **Seven S05 Actions:**
  1. `vicidial.00.files`
  2. `vicidial.01.perl`
  3. `vicidial.02.configuration`
  4. `vicidial.03.registration`
  5. `vicidial.04.generate`
  6. `vicidial.05.web`
  7. `vicidial.06.verify`
- **Locking & Intent:** Stage lock (`apply.lock`) and durable journal intent precede any mutations.
- **Foreign Edit Rejection:** Alteration of owned generated file (`confbridge-vicidial.conf`) was rejected with exit code 1; exact preimage restored without reload.
- **Copied-Journal Recovery:** Reconciled all 7 actions from checkpoint without replaying completed actions.
- **Fresh NO_OP:** Confirmed `no_op: true`, empty action graph, 0 traces, and identical service PIDs (`asterisk` 792, `mariadbd` 886, `httpd` 11829, `php-fpm` 11700) and InvocationIDs.

---

## 10. Evidence Integrity and Overwrite Incident Investigation

- **Developer Evidence Ledger:** `/home/ra/almadial-lab/evidence/m2-05/20261110T025748Z-alm20/SHA256SUMS` (SHA-256: `784555db98ea5e66783f1b78fff4a83be63497e5e52618f68a2c5f549ddf2428`). 100% of all 220 entries verified OK.
- **Helper Overwrite Incident:**
  - Obsolete `reproduce-baseline.py` overwrote `pre-s05-baseline.json` during the initial web-key failure.
  - Snapshot `alm20-pre-vicidial-20261110` was intact; VM was restored and baseline re-captured into `recovered-pre-s05-baseline.json`.
  - All preservation claims (schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, 725 RPMs, seed phones) were corroborated independently via ALM-18/ALM-19 QA evidence and raw DB queries.
  - Incident classified as **RESOLVED, FULLY TRACEABLE, INDEPENDENTLY CORROBORATED, AND NONBLOCKING**.

---

## 11. Code Quality, Test Suite, and Reproducible Builds

- **Go Code Standards (Go 1.27.3 linux/amd64):**
  - `gofmt -l cmd/ internal/`: Clean (0 unformatted files).
  - `go vet ./...`: Clean (exit code 0).
  - `go mod verify`: Clean (`all modules verified`).
- **Go Test Suite:**
  - Full suite (`go test ./...`): 197 top-level pass, 0 fail; 1,295 passing events.
  - Race detector (`go test -race ./...`): 197 top-level pass, 0 fail; 1,295 passing events; clean race detector.
- **Python Test Suite:**
  - `python3 -m unittest discover -s scripts/tests -v`: 164 tests passed with OK.
  - `python3 scripts/validate_scaffold.py`: PASS.
  - `python3 scripts/verify_candidate_stack.py`: PASS.
  - `python3 scripts/verify_config_generation.py`: PASS.
  - `git diff --check`: Clean (exit code 0).
  - CLI smoke tests: 17 passed.
  - PTY journeys: 11 passed.
- **Reproducible Builds:**
  - Main CLI (`scripts/build_go.py`): `06ebba4b4b4f121784766cdddaa03d9b42d3709af6295fb2688e4797fca7f5d3` (exact match).
  - S05 Qualifier (`cmd/almadial-vicidial`): `1e1b77d959d98ea83f9c0f760073966feaa08facc8796aefc2840c219bf9f053` (exact match, verified on VM).

---

## 12. Final Independent QA Verdict

All canonical acceptance criteria for ALM-20 / M2-05 are independently verified and satisfied.

**Final Verdict:**
`ALMADIAL_M2_05_ACCEPTED`
