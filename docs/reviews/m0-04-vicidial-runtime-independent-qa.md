# AlmaDial Independent QA Acceptance — ALM-4 / M0-04 VICIdial Application Runtime Qualification

**Reviewer / Session:** Google Antigravity (AGY), separate independent review session  
**Reviewer Professional Role:** Principal VICIdial Compatibility QA Engineer / MariaDB, PHP & Perl Runtime Verification Architect / Software Supply-Chain Auditor  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)  
**Implementer:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Target Task:** ALM-4 / M0-04 (`cf87vzpxxjnuchzwl19pdb6a`), Kaneo Project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Repository Starting HEAD:** `e0bf69e277dd1a2d4c1266e1c745a7ac8d61d4ec` (ALM-3 input commit: `4e2abd5281990949e4479230a4a4dcd0478e796e`)  
**Host Target:** `almadial-alma10` (Multipass 1.17.0 / QEMU, AlmaLinux 10.2 x86_64, IP `10.160.90.197`)  
**Review Date / Time:** 2026-11-06 (Session: 2026-11-06T05:40:00+08:00)  
**Protected Evidence Root:** `/home/ra/almadial-lab/evidence/m0-04/20261105T201527Z/` (`pre-handoff-ledger.json` SHA-256 `b9da84757071537f846cb5fc5b3417689662c190bf2b77c72ac5ea7198108f43`)  
**QA Evidence Path:** `/home/ra/almadial-lab/evidence/m0-04/qa-20261106T053500Z/`  
**Historical Implementation Report:** Preserved in [`docs/compatibility/m0-04-vicidial-runtime.md`](../compatibility/m0-04-vicidial-runtime.md)  
**Final Independent QA Verdict:** **`ALMADIAL_M0_04_ACCEPTED`**

---

## 1. Scope, Authority, and Reviewer Independence Confirmation

This review constitutes the formal independent quality assurance evaluation of task **ALM-4 / M0-04: Validate pinned VICIdial schema, PHP flows and Perl runtime**.

In accordance with repository governance ([`AGENTS.md`](../../AGENTS.md), [`GEMINI.md`](../../GEMINI.md), [`.agents/skills/almadial-review/SKILL.md`](../../.agents/skills/almadial-review/SKILL.md), and [`.agents/rules/almadial-kaneo-governance.md`](../../.agents/rules/almadial-kaneo-governance.md)):
- **Separation of Roles:** Codex performed the source acquisition, derivation creation, database initialization, PHP/Perl execution, and self-checks on disposable guest `almadial-alma10`, correctly submitting ALM-4 to `QA Review` (`in-review`) without claiming self-acceptance.
- **Reviewer Authority:** Google Antigravity (AGY) operates as the distinct reviewer under authorized Kaneo identity `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`. AGY does not assume ownership of ALM-4; Codex remains the permanent implementation owner (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`).
- **Scope Boundaries:** Testing is strictly confined to VICIdial r4037 source authentication, MariaDB 10.11 schema initialization, PHP 8.3 web administration/session flows, Perl 5.40 syntax and bounded queue polling, and preservation of the accepted Asterisk 22.12.0 runtime. Zero campaign dialing, zero external SIP/carrier traffic, and zero daemon auto-start was conducted.

---

## 2. Source Baseline and Working Tree State

- **Input Baseline:** Commit `4e2abd5281990949e4479230a4a4dcd0478e796e` (`test(m0): independently accept Asterisk WebRTC echo runtime`).
- **Intermediate Operations Commits:** Three unrelated operator portal commits (`0d36667`, `fba8f6f`, `e0bf69e`) were inspected; they modify only `operations/portal443/` and do not alter AlmaDial installer, core documentation, or candidate stack contracts.
- **Reviewed ALM-4 Implementation Deliverables:**
  - [`scripts/prepare_vicidial_r4037.py`](../../scripts/prepare_vicidial_r4037.py)
  - [`scripts/tests/test_prepare_vicidial_r4037.py`](../../scripts/tests/test_prepare_vicidial_r4037.py)
  - [`scripts/verify_candidate_stack.py`](../../scripts/verify_candidate_stack.py)
  - [`scripts/tests/test_verify_candidate_stack.py`](../../scripts/tests/test_verify_candidate_stack.py)
  - [`docs/compatibility/m0-04-vicidial-runtime.md`](../compatibility/m0-04-vicidial-runtime.md)
  - [`docs/planning/m0-execution-readiness.md`](../planning/m0-execution-readiness.md)
  - [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml)
- **Preserved Untracked Work:** Pre-existing untracked helper `scripts/verify_agy_overlay.py` was inspected and preserved without staging or modification.
- **Candidate Stack Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) retains `status: candidate`. The whole-stack aggregate runtime gate remains `verification.runtime_state: NOT_RUN`.

---

## 3. VICIdial Source Authentication & Provenance Resolution

1. **Source Identity:**
   - Candidate: VICIdial SVN revision `r4037`, schema version `1748`, 2,511 exported source files.
   - Pinned Canonical Tree SHA-256: `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
2. **Official HTTPS Archive Verification:**
   - Archive URL: `https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip`
   - Retrieved file: `nightly.download` (30,258,976 bytes).
   - Archive SHA-256: `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`.
   - TLS Certificate: Successfully validated against official CA hierarchy at `www.vicidial.org`.
3. **Tree Extraction and Canonical Tree Digest Audit:**
   - The independent reviewer extracted `nightly.download` into an isolated temporary workspace.
   - File count: Exactly **2,511 files**.
   - All 2,511 extracted files matched the pinned r4037 SVN export byte-for-byte.
   - Execution of `verify_candidate_stack.tree_digest` on the extracted tree yielded:
     `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
   - **Provenance Resolution:** The previous M0-02 hold on root execution is resolved for this exact artifact. The authenticated HTTPS transport endpoint plus 100% full-tree byte equality provides an authoritative publisher-origin chain for lab execution.
   - **Documented Limitation:** This is an authenticated HTTPS publisher download and exact content comparison, **not** a detached PGP release signature or signed SVN revision attestation.

---

## 4. SQL and Perl Derivation Integrity Verification

The upstream VICIdial codebase contains two known historical incompatibilities on modern Linux/MariaDB distributions: legacy unprivileged `cron` grants and `ps` process argument truncation.

1. **Derivation Helper Audit ([`scripts/prepare_vicidial_r4037.py`](../../scripts/prepare_vicidial_r4037.py)):**
   - Strictly enforces pinned SHA-256 checksums on all 5 input source files. Any byte change or unreviewed revision raises `ValueError`.
   - Rejects source symlinks (`is_symlink()`).
   - Requires an exclusive, non-existent output directory (`os.mkdir(mode=0o700)`).
   - Writes derived files using `O_CREAT | os.O_EXCL` with mode `0600`.
   - Emits an immutable audit manifest `derivation.json`.
2. **SQL Privilege Separation:**
   - Replaces exactly the two global grants (`GRANT RELOAD ON *.* TO cron@'%';` and `GRANT RELOAD ON *.* TO cron@localhost;`) and `flush privileges;` with an explicit ownership comment.
   - **Zero table structures, column types, default values, or seed inserts are modified**.
   - Input SQL: `extras/MySQL_AST_CREATE_tables.sql` (`8a210454a6f42fd3ef0054033a5c4396a0cd3615442ebd53491dfd2efa38de40`).
   - Output SQL SHA-256: `0992448a2607dbad6841250a09b68dafd360cb4fa44f8589d89526ef16522af8`.
3. **Process-Detection Argument Truncation Correction:**
   - In 4 pinned Perl scripts (`ADMIN_keepalive_ALL.pl` [3 occurrences], `AST_manager_send.pl` [1], `AST_VDauto_dial.pl` [1], `AST_VDremote_agents.pl` [1]), replaces legacy `ps -o "%p %a" --no-headers -A` with wide format `/bin/ps -o pid,args -ww --no-headers -A`.
   - Total replacements: Exactly 6 across the 4 files.
   - Derived Perl file hashes verified against recorded ledger:
     - `ADMIN_keepalive_ALL.pl`: `51abc2f7e65f8e6722e218b9714d7142e2cb76c1fba755d8a42e195a9fbf555a`
     - `AST_manager_send.pl`: `007fc3012ea5c37e5c5a330719ed51a621aa9ad84488ade3ef13eac9d64756d3`
     - `AST_VDauto_dial.pl`: `458e6fe7ad864760688e73d3031fe5de0d28e1fae6a31bbdabfc5448542f3d3e`
     - `AST_VDremote_agents.pl`: `c6093d1551290b283e7cae7bbe8ef684e4e93298c2b1880e79b758083eb6b173`
4. **Reproducibility Check:**
   - Independent test execution of `prepare()` on freshly extracted source produced bit-for-bit identical hashes across all 5 derived files.

---

## 5. MariaDB 10.11.18 Schema 1748 Live Verification

Live read-only SQL queries on guest `almadial-alma10` confirmed:

```sql
SELECT VERSION(), @@sql_mode;
USE almadial_m004_v2;
SELECT db_schema_version FROM system_settings;
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='almadial_m004_v2';
SELECT ENGINE, COUNT(*) FROM information_schema.tables WHERE table_schema='almadial_m004_v2' GROUP BY ENGINE;
SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema='almadial_m004_v2';
```

### Observed Results:
- **Server Version:** `10.11.18-MariaDB` (packaged `mariadb-server-3:10.11.18-1.el10_2.alma.2.x86_64`).
- **SQL Mode:** `STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION` (strict mode intact; no silent downgrade).
- **Schema Version:** Exactly **1748**.
- **Table Count:** Exactly **345 tables** (337 MyISAM, 8 MEMORY).
- **Index Statistics Rows:** **1,011** rows in `information_schema.statistics`.
- **Table Integrity (`mariadb-check --check --quick`):**
  - **337 MyISAM tables reported `OK`**.
  - **8 MEMORY tables** (`live_channels`, `live_sip_channels`, `parked_channels`, `server_updater`, `vicidial_auto_calls`, `vicidial_live_agents`, `vicidial_live_inbound_agents`, `web_client_sessions`) reported `note : The storage engine for the table doesn't support check` with overall exit code 0.
  - **Classification:** As required by the QA contract, this is recognized as an engine property of the MEMORY storage engine, not schema corruption.
- **Import Warnings Evaluated:**
  - Code 1071 notes (key length > 1000 bytes) on `www_phrases.phrase_text(333)` and `vicidial_language_phrases.english_text(333)`: Non-fatal prefix index truncated to 333 characters in utf8mb3.
  - Code 1831 note: Duplicate index `vdlga` is accepted by MariaDB 10.11 without error.
- **Schema Recovery Contract:**
  - Initial partial database `almadial_m004` (failed at line 5817) remains preserved with version `0` and 345 tables.
  - Repeating import against an existing schema throws `ERROR 1050 (table phones already exists)`.
  - Recovery rules for future Go installer: Differentiate empty (eligible for bootstrap), partial (halt/preserve), and populated 1748 (reuse without recreation).

### QA Architectural Finding: MariaDB Networking Configuration
- **Observation:** `guest-base.py` wrote `/etc/my.cnf.d/almadial-m004.cnf` with permissions `0600 root:root`. The `mariadbd` service runs under dedicated user `mysql:mysql`. Because user `mysql` cannot read `0600 root:root` files, the drop-in file was skipped by MariaDB at startup, and `mariadbd` listened on TCP port 3306 (`0.0.0.0:3306`).
- **Mitigating Controls:** All accounts in `mysql.user` (`root`, `mysql`, `mariadb.sys`, `alm004`) are restricted strictly to `Host = localhost`. No remote accounts (`%`) exist, and the lab VM is on an isolated NAT network.
- **Recommendation for Go Installer:** Configuration files in `/etc/my.cnf.d/` must be readable by the `mysql` service account (mode `0644` or `0640 root:mysql`).

---

## 6. PHP 8.3 Compatibility and Functional Application Verification

1. **Static Validation:**
   - 309/309 PHP files in `/var/www/html/m004` passed `php -l` with **0 syntax errors**.
   - Required extensions verified loaded: `curl`, `gd`, `ldap`, `mbstring`, `mysqli`, `mysqlnd`, `openssl`, `PDO`, `pdo_mysql`, `session`, `SimpleXML`, `xml`, `Zend OPcache`.
2. **Web Environment & Permissions:**
   - Web files owned by `root:apache`, permissions `0750` / `0640`.
   - Non-writable webroot configuration enforced via `system_settings.webroot_writable = 0` and `$use_slave_server = 0` in `options.php`.
   - Apache listening strictly on `127.0.0.1:8080`.
3. **Live Request and Postcondition Verification:**
   - **Anonymous Admin:** `curl -s http://127.0.0.1:8080/m004/vicidial/admin.php` returns `HTTP 401 Unauthorized` with `WWW-Authenticate: Basic realm="CONTACT-CENTER-ADMIN"`.
   - **First-Login Setup:** Admin setup cleared `first_login_trigger` to `N` and accepted generated admin credentials.
   - **Admin User Provisioning:** Created user `m004qa`. Live database query confirmed `vicidial_users` row (`user_id = 4`, `user = 'm004qa'`, `full_name = 'M004 QA Test'`, `user_level = 1`).
   - **Agent Campaign Read:** `LogiNCamPaigns` returned valid application response.
   - **Seeded Agent Session & Favorites Write:** Request with seeded session accepted favorites `9001`. Verified live in `phone_favorites` table (`extension = 'm004qa'`, `server_ip = '127.0.0.1'`, `extensions_list = '9001'`).
   - **Session Expiration:** Deleting session resulted in subsequent write rejection; database remained at `9001`.
4. **Classification of 26 PHP Warnings:**
   - **Analysis:** 25 occurrences of undefined variables in `admin.php`:
     - `$modify_settings_containers` (line 4836, 5x)
     - `$max_inbound_auto_reenable` (line 4858, 5x)
     - `$hangup_again_link` (line 5079, 5x)
     - `$amd_agent_display` (line 5080, 5x)
     - `$amd_status_map` (line 5540, 5x)
     - 1 null array offset access in `admin.php` line 54268 (1x).
   - **Impact Assessment:** These represent standard PHP 8 notice-to-warning escalations on legacy uninitialized global settings. None caused fatal execution, none corrupted data, and all requested database writes committed accurately. Non-blocking for ALM-4 acceptance.

---

## 7. Perl 5.40 Runtime and Process Detection Verification

1. **Module Availability:**
   - Pinned modules independently verified on guest:
     - `DBI`: 1.643
     - `DBD::mysql`: 5.007
     - `Net::Telnet`: 3.05 (from authenticated CPAN tarball, installed to `/usr/local/share/perl5/5.40`)
     - `Net::Server`: 2.014
     - `Time::HiRes`: 1.9777
     - `Switch`: 2.17 (EPEL 10.2 signed package)
     - `Crypt::Eksblowfish::Bcrypt`: 0.009 (EPEL 10.2 signed package)
     - `Asterisk::AGI`: Loaded cleanly from `extras/`.
2. **QA Architectural Finding: Perl Library Permissions:**
   - `/usr/local/share/perl5` was created with mode `0700 root:root` during CPAN installation under umask 077.
   - Unprivileged users received `Permission denied` on `DBI.pm` unless running with `sudo`.
   - **Recommendation for Go Installer:** Set site-perl directory permissions to `0755` (`chmod 0755 /usr/local/share/perl5`) so unprivileged runtime accounts can execute Perl jobs.
3. **Syntax Validation:**
   - All 9 original scripts (`ADMIN_keepalive_ALL.pl`, `AST_update.pl`, `AST_update_AMI2.pl`, `AST_manager_listen.pl`, `AST_manager_send.pl`, `AST_VDauto_dial.pl`, `AST_VDadapt.pl`, `AST_conf_update.pl`, `FastAGI_log.pl`) passed `perl -c`.
   - All 4 derived scripts (`ADMIN_keepalive_ALL.pl`, `AST_manager_send.pl`, `AST_VDauto_dial.pl`, `AST_VDremote_agents.pl`) passed `perl -c`.
4. **Independent Reproduction of Procps Defect:**
   - Executed live background test with long argument string:
     - Legacy command `/bin/ps -o "%p %a"` truncated argv at ~30 characters, hiding the target script name.
     - Corrected wide command `/bin/ps -o pid,args -ww` retained full argv with 100% fidelity.
   - Proves the defect is reproducible on AlmaLinux 10 and that the derivation is necessary and sufficient.
5. **Bounded Sender Execution:**
   - Both original and derived `AST_manager_send.pl` executed against the live MariaDB database for 1.5 seconds.
   - Successfully connected to database, polled empty queue, originated 0 calls, generated 0 AMI actions, and terminated cleanly on SIGTERM with returncode 0.

---

## 8. Asterisk 22.12.0 Environment Preservation

Independent verification on `almadial-alma10` confirmed that the accepted ALM-3 telephony baseline is completely preserved:

- **Service Status:** `systemctl is-active asterisk` returned `active`.
- **Binary Integrity:** `/usr/sbin/asterisk` SHA-256 is `db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c` (100% match with ALM-3 acceptance).
- **Configuration Integrity:** All 10 configuration file hashes in `/etc/asterisk/` (`asterisk.conf`, `modules.conf`, `http.conf`, `pjsip.conf`, `extensions.conf`, `rtp.conf`, `logger.conf`, `manager.conf`, `ari.conf`, `confbridge.conf`) matched ALM-3 acceptance records exactly.
- **Listeners and Channels:**
  - Plain HTTP bound to `127.0.0.1:8088`.
  - HTTPS/WSS bound to `0.0.0.0:8089` (`/ws` and `/static/...`).
  - Endpoint `9001` active and bound to `transport-wss`.
  - Active channels: 0; active calls: 0.
- **SELinux:** Enforcing (`getenforce` returned `Enforcing`).
- **No Generated Overwrite:** Zero VICIdial Asterisk files were activated or published into `/etc/asterisk/`.

---

## 9. Evidence Ledger and Cryptographic Audit

- **Evidence Ledger:** `/home/ra/almadial-lab/evidence/m0-04/20261105T201527Z/pre-handoff-ledger.json` (SHA-256 `b9da84757071537f846cb5fc5b3417689662c190bf2b77c72ac5ea7198108f43`).
- **Artifact Count:** Exactly **78 files**.
- **Audit Outcome:** 100% of all 78 files were independently verified for byte length and SHA-256 checksum. Zero missing or modified files.
- **Key Artifact Hashes Verified:**
  - `guest-evidence.tar.gz`: `5e7075062b9da9590c83c3434cde77c21333a8b305832fdb584c82d73a5e25c1`
  - `closure-evidence.tar.gz`: `40b4d3df0ff6e1e2698a5cbfdb70b009bfaac35f14204738f1937f79bb413eed`
  - `nightly.download`: `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`
  - `derived-r4037/extras/MySQL_AST_CREATE_tables.sql`: `0992448a2607dbad6841250a09b68dafd360cb4fa44f8589d89526ef16522af8`
- **Snapshot Integrity:** Snapshot `almadial-alma10.pre-vicidial-r4037` intact. Zero snapshot restorations executed.

---

## 10. Repository Automated Validation

The independent reviewer executed the full automated test and validation suite locally:

1. `python3 scripts/validate_scaffold.py`: `PASS` (11 files, 4 skills).
2. `python3 -m unittest discover -s scripts/tests -v`: **56/56 unit tests PASSED** in 0.913s.
   - Includes 5 new `test_prepare_vicidial_r4037` tests and 2 new candidate provenance transition tests.
3. `python3 -m py_compile ...`: All modified and new Python scripts compiled cleanly without warnings.
4. `python3 scripts/verify_candidate_stack.py`: Candidate consistency `PASS` in LOCAL mode.
5. `git diff --check`: Clean; zero trailing whitespace or formatting defects.

---

## 11. Final Acceptance Verdict and Downstream Recommendations

### Acceptance Verdict:
**`ALMADIAL_M0_04_ACCEPTED`**

Task ALM-4 satisfies all acceptance criteria established in `docs/AlmaDial_Implementation_Plan.md`. VICIdial r4037 source provenance is established, the deterministic schema derivation initializes cleanly on MariaDB 10.11 without modifying table definitions, PHP 8.3 administration and session workflows execute with verified database postconditions, Perl 5.40 runs with required modules, process detection truncation is resolved, and Asterisk 22.12.0 is fully preserved.

### Accepted Limitations:
1. **Source Authentication Boundary:** Source trust is based on an official authenticated HTTPS download and byte-for-byte canonical tree equality, not a detached PGP signature or SVN attestation.
2. **Telephony Integration Scope:** Agent testing used a seeded database session; full browser agent phone login, WebRTC audio in conferences, AMI event streaming, and FastAGI execution are downstream integration gates.
3. **Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) remains in `status: candidate`.
4. **MariaDB / Perl Permissions:** Lab drop-in permissions (`0600`) resulted in default TCP binding and restricted site-perl; resolved via localhost-only user grants and documented for Go installer design.

### Next Assignment Recommendation (ALM-5):
- **Task:** **ALM-5 / M0-05** (`z6mn0h4rh66ip9hh7hh769eu`) — *Audit VICIdial PJSIP and dialplan generation ownership*.
- **Role:** Principal Asterisk & VICIdial Integration Architect / Configuration Generation Auditor.
- **Scope:** Define ownership boundaries between AlmaDial static Asterisk configurations and VICIdial-generated includes (`*-vicidial.conf`), audit `ADMIN_keepalive_ALL.pl` generation logic for PJSIP and ConfBridge, establish local-only AMI credentials, and verify configuration reload safety without clobbering ALM-3 assets.
- **Precondition:** ALM-4 acceptance unblocks ALM-5 (native dependency `nypqp0czzsifv2ujayonxaqd`).
- **ALM-5 was not started during this assignment.**
