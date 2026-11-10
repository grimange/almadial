# AlmaDial — ALM-23 / M3-01: Independent Apache/PHP and VICIdial Application QA Report

## Acceptance Status and Governance

- **Task ID:** `ed6ilp2yw5ldla4ovm44i3vc`
- **Canonical Work Package:** ALM-23 / M3-01
- **Task Title:** Provision Apache/PHP and verify application flows
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Starting Git Baseline:** `512cf11705255da556d977328774f64ce39a6947`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `alm21-pre-runtime-20261110` (preserved)
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_01_WEB_RUNTIME_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S07 provisions an owned Apache 2.4 / PHP-FPM 8.3 web delivery runtime on AlmaLinux 10 for native VICIdial r4037. Independent QA was conducted across source code, unit and integration tests, live laboratory VM evaluation, browser-driven end-to-end administrative and agent workflows via Playwright Chromium 157.0.8103.38, failure-recovery reconciliation, and bit-for-bit baseline preservation.

All canonical acceptance criteria for ALM-23 are satisfied:
1. **Apache/PHP Delivery:** Apache 2.4.63 and PHP-FPM 8.3.36 are active and healthy under systemd supervision, using Unix-domain socket communication (`/run/php-fpm/www.sock`).
2. **S07 Configuration Ownership:** S07 strictly owns `/etc/httpd/almadial-s07.conf` and `/etc/httpd/conf.modules.d/99-almadial-s07.conf` (`root:root 0644, httpd_config_t`). Foreign Apache and PHP configuration files are hashed and preserved.
3. **Guarded First-Login Bootstrap:** Pinned VICIdial's destructive `admin.php` setup form (which rotates the administrator password and rewrites all phone credentials) is guarded. S07 verifies the existing 48-hex protected administrator credential from `/etc/almadial/secrets/admin` and transitions only `system_settings.first_login_trigger` (`Y` → `N`) under table lock, without modifying any phone, credential, or schema element. Downstream normalization enables clean S05 receipt comparison.
4. **Browser-Driven Administrative Workflow:** Successful anonymous 401 challenge, authenticated HTTP Basic login to `admin.php`, navigation across Users, Campaigns, Lists, and Scripts, creation and modification of disposable level-1 test user `ALM23QA`, fresh application listing verification, and MariaDB database verification.
5. **Agent-Facing & Timeclock Workflows:** `agc/vicidial.php` loads correctly alongside all 7 required static/application assets (CSS, JS, images). `options.php` access is denied (HTTP 403). `agc/timeclock.php` correctly rejects invalid credentials, authenticates `ALM23QA`, records clock-in (`LOGIN`), respects the native 30-second cooldown, records clock-out (`LOGOUT`), and maintains zero live agents.
6. **Recovery & Clean NO_OP:** S07 plans return an empty graph (`[]`) on a satisfied host, generating zero runner traces (`no_op: true`). Synthetic copied-journal fixtures reconcile all three S07 actions (`web.00.files`, `web.01.services`, `web.02.verify`) to `succeeded` without re-executing mutations.
7. **Baseline Preservation:** 100% bit-for-bit preservation of the accepted M2 backend: 725 RPMs (`bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`), schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 40 Asterisk modules, FastAGI (1+5 processes), and SELinux Enforcing with zero AVC denials.

---

## 2. Independent Acceptance Matrix

| Criterion | Result | Verified Observations & Evidence |
| :--- | :---: | :--- |
| **Apache/PHP Integration** | **PASS** | Apache 2.4.63 & PHP-FPM 8.3.36 active/enabled; Unix socket `/run/php-fpm/www.sock`; `httpd -t` Syntax OK; `php-fpm -t` test successful; 9 required PHP extensions active (`mysqli`, `mysqlnd`, `mbstring`, `curl`, `gd`, `xml`, etc.); loopback-only listeners (`127.0.0.1:80`). |
| **S05-to-S07 Ownership** | **PASS** | S05 retains VICIdial application file tree and `/etc/astguiclient.conf`; S07 owns `/etc/httpd/almadial-s07.conf` and `/etc/httpd/conf.modules.d/99-almadial-s07.conf`; zero clobbering of vendor or operator configuration. |
| **Administrator Authentication** | **PASS** | Anonymous request to `/vicidial/admin.php` returns HTTP 401 with `WWW-Authenticate: Basic realm="CONTACT-CENTER-ADMIN"`; protected 48-hex credential authenticates successfully (HTTP 200) displaying `ADMINISTRATION: SYSTEM SNAPSHOT STATS` and `Logout (6666)`. |
| **Administrator Navigation & CRUD** | **PASS** | Successfully navigated Users (`ADD=0A`), Campaigns (`ADD=10`), Lists (`ADD=100`), Scripts (`ADD=1000000`) without fatal errors; created disposable user `ALM23QA`, edited full name to `ALM23 Web QA Edited`, verified in fresh Users listing, and confirmed in MariaDB (`user_id=5, level=1, group=ADMIN`). |
| **Guarded First-Login Completion** | **PASS** | Upstream destructive setup handler bypassed; `web_setup()` verifies existing 48-hex secret matches active level-9 user `6666` and updates only `first_login_trigger` `Y` → `N` under table locks; `callin` and `gs102` seed phones remain untouched. |
| **Agent-Facing Pages & Resources** | **PASS** | `/agc/vicidial.php` returns HTTP 200 (`title: Agent web client: Phone Login`); 7 static/application assets return HTTP 200 with appropriate Content-Type; `/agc/options.php` and `/vicidial/options.php` return HTTP 403 Forbidden. |
| **Timeclock Journey** | **PASS** | Invalid credentials rejected with "not active in the system"; valid authentication clocks in (`LOGIN` logged in `vicidial_timeclock_log`); 30s native cooldown respected; clock-out succeeds (`LOGOUT` logged, `login_sec=40`); zero live agents; test account and timeclock events cleaned up while preserving `vicidial_admin_log` audit entries. |
| **S07 Journal Recovery** | **PASS** | `TestWebExecutionAndRecovery` passes interruption after each effect (`web.00.files`, `web.01.services`, `web.02.verify`); synthetic copied-journal fixture reconciles completed host state to `succeeded` without replaying mutation actions (0 traces). |
| **Actual-Host NO_OP** | **PASS** | `almadial-web plan` returns empty graph `[]` (`stage_sha256: 26729790e5b9d6769b61293100d0d9b408be4b132f2b9d06f6c16b20ff32d671`); `lab-run` returns `no_op: true`, 0 runner traces, 0 file rewrites, 0 service reloads, and 0 database changes. |
| **M2 Baseline Preservation** | **PASS** | 725 RPMs (`bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`), schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 40 Asterisk modules, FastAGI (1+5), updater/listener disabled, dialer cron inactive, and SELinux Enforcing (0 AVCs). |
| **Regression Tests** | **PASS** | Go 1.27.3 vet/fmt/test/race passed (208 top-level tests, 1,349 pass events, 0 race defects); Python passed (188 tests, including 7 S07 tests); scaffold validation (11 files, 4 skills), candidate stack, config generation, 17 CLI smoke tests, and 11 PTY wizard journeys passed; 8 Linux amd64 command targets verified byte-identical. |
| **Evidence Integrity** | **PASS** | Developer primary ledger (221 entries, SHA-256 `548b081bda76eadb253ec7f99d6630732bba2b3ebc16e12ce5bdc19bc77cf854`) verified 100% OK; independent QA evidence directory sealed with 6-entry SHA-256 ledger `2795b23c34aea67a03f928f7a3b92c5066a8e0a44901c5375bebe5619e1d2309`. |

---

## 3. Detailed Verification Results

### 3.1 Apache and PHP-FPM Configuration & Service State
- **Apache Service:** `httpd.service` is active (running) and enabled. Main PID was preserved without restart.
- **PHP-FPM Service:** `php-fpm.service` is active (running) and enabled, communicating via `/run/php-fpm/www.sock`.
- **Apache Vhost:** Defined in `/etc/httpd/almadial-s07.conf`, included via `/etc/httpd/conf.modules.d/99-almadial-s07.conf`. Sets `X-AlmaDial-Web "S07"`, enforces `Require local`, disables directory indexing, and denies direct access to `.ht*`, `.user.ini`, and `options.php`.
- **Security & Listeners:** Listening strictly on `127.0.0.1:80` (HTTP). No public listener was created. SELinux remains `Enforcing` with zero AVC denials observed.

### 3.2 Administrator Browser Workflow (Playwright)
- **Tooling:** Playwright Chromium 157.0.8103.38 connected via temporary SSH host-loopback tunnel (`127.0.0.1:18023` -> VM `127.0.0.1:80`).
- **Authentication:** Anonymous GET `/vicidial/admin.php` challenged with HTTP 401. Supplying HTTP Basic authentication (`6666:<48-hex-secret>`) loaded the authenticated snapshot stats dashboard.
- **Navigation:** Verified that clicking or navigating to Users (`ADD=0A`), Campaigns (`ADD=10`), Lists (`ADD=100`), and Scripts (`ADD=1000000`) rendered real application HTML without fatal PHP errors.
- **User CRUD:** Created user `ALM23QA` (level 1, group ADMIN), edited full name to `ALM23 Web QA Edited`, confirmed the row was listed in `admin.php?ADD=0A`, and confirmed record persistence in MariaDB table `vicidial_users`.
- **Screenshot Evidence:** Captured full-page screenshot at `browser/admin-users.png`.

### 3.3 Agent-Facing Web Functionality & Timeclock Journey
- **Agent Entry Point:** `/agc/vicidial.php` rendered `Agent web client: Phone Login` with all 7 application static assets (style.css, custom.css, calendar_db.js, confetti.php, calendar.css, vicidial_admin_web_logo.png, agent_loading_animation.gif) returning HTTP 200.
- **Hardened Access:** Access to `/agc/options.php` and `/vicidial/options.php` returned HTTP 403 Forbidden.
- **Timeclock Testing:**
  - Navigated to `/agc/timeclock.php`.
  - Tested invalid credentials (`ALM23QA` / `wrongpassword1234`): correctly rejected with error message.
  - Tested valid credentials: authenticated, displayed "Click LOGIN below to log-in", clicked `LOGIN`, recorded clock-in.
  - Immediate re-login triggered the expected native 30-second cooldown message.
  - Following the cooldown period, authenticated, displayed "Click LOGOUT below to log-out", clicked `LOGOUT`, recorded clock-out (`login_sec: 40`).
  - Verified MariaDB `vicidial_timeclock_log` recorded exactly one `LOGIN` and one `LOGOUT` event; verified `vicidial_live_agents` remained 0.
- **Screenshot Evidence:** Captured full-page screenshot at `browser/timeclock-out.png`.
- **Cleanup:** Test account `ALM23QA` and associated timeclock records were removed from MariaDB while preserving `vicidial_admin_log` audit entries (`admin_log_id: 4` and `5`). The SSH tunnel was terminated and temporary host keys were removed from the VM.

### 3.4 Defect Resolution & Bounded Scope Analysis
1. **Guarded Setup Handler:** Upstream `admin.php` setup behavior was prevented from executing destructive password/phone updates. S07 updates only `system_settings.first_login_trigger` (`Y` → `N`) under table lock, guarded by the verified presence of user `6666`.
2. **Logout HTML Markup Parsing:** Native probe checks normalized visible text, correctly handling upstream HTML tags separating `Logout` from `(6666)`.
3. **Documented Nonblocking Findings:**
   - Upstream undefined-variable warnings in PHP-FPM error logs (nonfatal, standard VICIdial behavior).
   - Missing `/favicon.ico` returning HTTP 404.
   - Pinned Timeclock password input field `maxlength=20` (disposable 20-hex password utilized for test; protected 48-hex admin password preserved).
   - Empty campaign selector on agent phone login (expected in dialing-inactive baseline).

---

## 4. Regression & Reproducibility Audit

### 4.1 Automated Test Execution

| Test Suite / Tool | Command Line | Result | Pass / Events |
| :--- | :--- | :---: | :--- |
| **Go Test Suite** | `/tmp/alm20-toolchain/go/bin/go test -json ./...` | **PASS** | 208 top-level tests, 1,349 pass events |
| **Go Race Detector** | `/tmp/alm20-toolchain/go/bin/go test -race -json ./...` | **PASS** | 208 top-level tests, 1,349 pass events, 0 race defects |
| **Go Vet & Format** | `go vet ./...` & `make fmt-check` | **PASS** | 0 warnings, clean formatting |
| **Go Module Verify** | `go mod verify` & `go mod tidy -diff` | **PASS** | All modules verified, 0 diffs |
| **Python Unit Tests** | `python3 -m unittest discover -s scripts/tests -v` | **PASS** | 188 tests passed in 2.05s |
| **Focused S07 Tests** | `python3 -m unittest scripts/tests/test_s07_web.py -v` | **PASS** | 7 tests passed |
| **Scaffold Validation** | `python3 scripts/validate_scaffold.py` | **PASS** | 11 files, 4 skills validated |
| **CLI Smoke Tests** | `alm23-smoke.py` suite | **PASS** | 17 of 17 cases passed |
| **PTY Wizard Journeys** | `python3 scripts/test_terminal_wizard.py` | **PASS** | 11 journeys passed |
| **PHP Syntax Linter** | `php -l` on admin, agent, and timeclock PHP files | **PASS** | No syntax errors detected |

### 4.2 Reproducible Build Verification (Linux amd64)

All eight Go binary targets compile deterministically and match the developer-reported digests:

- `almadial` (Main CLI): `9eddabb8123b7060ac5e5783592d9b8c44bb7d2d80d5c8f102a6a5b061508dd7`
- `almadial-web` (Stage S07): `510215e0120c0e98de80d9e1c63493c14e0d986ac1cc52d94d901a4c21e91b68`
- `almadial-runtime`: `6911c2ee08e3b16ee5ac9e63149be94c910bf41ebba470d796bfdeed778eefcb`
- `almadial-rpm`: `421e5ca0c58c526e1bec67206e28106f0b2ad084210c1d3085f70c05be0982f8`
- `almadial-system`: `92a0ceda102be305fdc1bea556bd120be7ee71c593c95ad02fee5c990127b3d5`
- `almadial-db`: `327e2be151c2ee28be6cc1426e7bcfb343572fee1d27908372dc4c4425544969`
- `almadial-asterisk`: `6eb5b4a1e66eca1c7dfc263bc9c7cd8b6f255e03a40aac8acbdb409c4629ab74`
- `almadial-vicidial`: `c684fe6fe841209592391da1162922d0fff8f290c0fe9f62bf6277e8f778849f`

---

## 5. Evidence Provenance & Checksums

- **Developer Evidence Directory:** `~/almadial-lab/evidence/m3-01/20261110T163346Z-alm23/` (0700)
  - Primary Ledger: `SHA256SUMS` (221 entries, SHA-256 `548b081bda76eadb253ec7f99d6630732bba2b3ebc16e12ce5bdc19bc77cf854`, verified 100% OK).
- **Independent QA Evidence Directory:** `~/almadial-lab/evidence/m3-01/qa-20261111T070500Z-alm23/` (0700)
  - Master Ledger: `SHA256SUMS` (SHA-256 `2795b23c34aea67a03f928f7a3b92c5066a8e0a44901c5375bebe5619e1d2309`, 6 files verified OK):
    - `baseline-preservation.json`: `a4f3b2eeea0d01ba4729fcf95dbad9b26bdf3b4fc48792e399ee60d3d0bc489e`
    - `browser-acceptance.json`: `68e82ef6b2f7960350d2bbd71f11a87b7a1215bb41fdbfb9c09c73336423c14a`
    - `regression-summary.json`: `7b233a76384a3ff30218a543e498118d0dd9c6bfda2b36b0fa3129486c90ef43`
    - `reproducible-builds.txt`: `3f03b290d0b04c8fbf4f056972ef7aa7523bb6f78f85f3408df2b982ce15c619`
    - `browser/admin-users.png`: `335e38eb829370ce79ba65cba0ae38d6df7e91d5eb8aa2715112f45ec00c25a0`
    - `browser/timeclock-out.png`: `5985b8fae72be69e6b52c03cf5cc8540c9ae7325bdfa4fa0a4b706c9b3603417`

---

## 6. Final Recommendation

Stage S07 satisfies all requirements for web runtime delivery, application configuration, browser-driven workflows, failure recovery, and backend baseline preservation. The task is fully qualified and recommended for acceptance. Codex ownership is strictly preserved on ALM-23, and downstream task ALM-24 remains unstarted.
