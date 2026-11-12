# AlmaDial — ALM-31 / M4-03: Independent Security Defaults, Exposure and Credential Leakage QA Report

## Acceptance Status and Governance

- **Task ID:** `mwbiriekq9liuvx3eggyjyko`
- **Canonical Work Package:** ALM-31 / M4-03
- **Task Title:** Audit security defaults, exposure and credential leakage
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted Baseline Commit:** `045534f3f62ac34ace7b136395293ad4c76e9668`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Evaluated Target Host:** `almadial-alm29-clean-d` (`10.160.90.38`, Machine ID `4c7b492940923838a0700adba65ed3eb`, AlmaLinux 10.2, 731 RPMs, snapshot `alm29-pristine`)
- **Authorized Independent Peer:** `almadial-alm29-clean-e` (`10.160.90.239`, Machine ID `ebf868d3a0853b4294cd85e589a12f5c`, snapshot `alm29-pristine`)
- **Developer Evidence Comment:** `g3xtfsapiq7f0w6svhrz8a7n`
- **Developer Evidence Ledger:** `/home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/SHA256SUMS` (`543fc6d40d02cda292d2d7a69deef5bd50152b521920143698d05bc63a2dfef2`, 65 files, 100% verified)
- **Independent QA Ledger:** `/home/ra/almadial-lab/evidence/m4-03/qa-20261113T021500Z-alm31/SHA256SUMS` (`0089787be1e155223913526a529638444ee4a346c084bcc15cbc3704e5e57491`, 11 files, 100% verified)
- **Developer Verdict:**
  ```text
  ALMADIAL_M4_03_SECURITY_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M4_03_SECURITY_ACCEPTED
  ```

---

## 1. Independent QA Acceptance Matrix

| # | Acceptance Criterion | Result | Detailed Evaluation & Evidence |
|---|---|:---:|---|
| 1 | **Canonical ALM-31 Scope** | **PASS** | Bounded strictly to security defaults, network exposure, authentication rejection, credential protection, sentinel leakage tests, and support-bundle security in the single-server product. No public carrier, general penetration test, or out-of-scope hardening. [docs/implementation/m4-03-security-defaults-exposure-and-credential-leakage.md](file:///home/ra/Documents/almadial/docs/implementation/m4-03-security-defaults-exposure-and-credential-leakage.md). |
| 2 | **IPv4/Link-Local IPv6 Isolation** | **PASS** | Evaluated from independent peer `almadial-alm29-clean-e`. 20 targeted probes across IPv4 and link-local IPv6 confirmed SSH (22) connected, while all 18 internal application/SIP/infrastructure ports timed out. [peer-reachability.json](file:///home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/peer-reachability.json). |
| 3 | **Internal TCP Service Exposure** | **PASS** | Host inspection via `ss -lntp` confirmed internal services are strictly bound to loopback: MariaDB (`127.0.0.1:3306`), Asterisk AMI (`127.0.0.1:5038`), FastAGI (`127.0.0.1:4577`), Apache (`127.0.0.1:80`, `:443`), Asterisk HTTP/WSS (`127.0.0.1:8088`, `:8089`). [inspect.json](file:///home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/inspect.json). |
| 4 | **Relevant UDP/SIP Policy** | **PASS** | No external SIP UDP/TCP listeners configured on ports 5060/5061. The sole SIP transport is `almadial-s10-wss` bound logically to loopback. S09 nftables policy (`almadial-s09`, priority `-20000`, target `DROP`) enforces complete isolation. |
| 5 | **Unauthorized SIP Registration** | **PASS** | Real WSS SIP registration on `:8089` rejected anonymous access (401), invalid password (401), and unknown identity (401). Zero contacts registered on Asterisk. [sip-tls.json](file:///home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/sip-tls.json). |
| 6 | **Authorized SIP Registration** | **PASS** | Authorized synthetic registration challenged and accepted (401 -> 200 OK); Asterisk contact `alm_s10_9061` verified via `pjsip show contacts`. Explicit unregistration (`Expires: 0`) returned 200 OK with zero residual contacts. |
| 7 | **Sensitive-File Permissions** | **PASS** | Permissions verified: `/etc/almadial/secrets/phone-s10` (0600, parent 0700, root:root), TLS private key (0640, parent 0750, root:asterisk), VICIdial DB config (0640, parent 0750, root:apache), `manager.conf` (0640, parent 0750, root:asterisk), journals (0600, root:root). No POSIX ACL overrides. [permissions-leakage-v2.json](file:///home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/permissions-leakage-v2.json). |
| 8 | **Runtime Identities and SELinux** | **PASS** | Process privileges verified: MariaDB as `mysql`, Asterisk and FastAGI as `asterisk`, Apache/PHP-FPM workers as `apache`. SELinux is `Enforcing`; `matchpathcon -V` verified 5 sensitive runtime paths; boot AVC query found 0 denials. |
| 9 | **Credential Sentinel Leakage** | **PASS** | 6 synthetic secret classes (`database`, `sip`, `ami`, `administrator`, `private_key`, `session`) passed failure capture, secret-bearing argv rejection, operator logs, and diagnostic projection without leakage. VM scan of 160 process argvs and 53 state/log files (3.9 MB) found zero protected secret matches. [internal/cli/security_qualification_test.go](file:///home/ra/Documents/almadial/internal/cli/security_qualification_test.go). |
| 10 | **Support-Bundle Security** | **PASS** | Generated support bundle is 3,516 bytes, mode 0600. Exactly four approved regular members (`SHA256SUMS`, `manifest.json`, `redaction.json`, `report.json`). Zero raw config dumps, private keys, database dumps, audio recordings, or browser sessions included. |
| 11 | **HTTPS/WSS Trust and Identity** | **PASS** | TLS 1.3 verified on ports 443 and 8089 using private lab CA. Negative tests confirmed invalid hostname rejected (code 62) and untrusted CA rejected (code 20). Expected 14-day lab certificate expiry warning produces informational exit code 10. |
| 12 | **Firewall Policy Preservation** | **PASS** | Managed policy `almadial-s09` remains active in firewalld and nftables. Runtime and permanent policies agree; SSH (22/tcp) and ICMP remain the sole external allowances. |
| 13 | **Managed-State Preservation** | **PASS** | 107 managed configuration files, journals, receipts, certificates, cron, and audio recordings strictly preserved bit-for-bit, including file permissions and ownership. 731 packages, effective nftables ruleset, and service invocation identities unchanged. Read-only diagnostics caused zero database mutations. Fresh planning emitted `actions: []`, `no_op: true`. |
| 14 | **Mandatory Regression Suite** | **PASS** | Local execution on Go 1.27.3: 1,464 Go test events (0 failures, 2 deliberate native skips), 0 data races under `-race`, 222 Python unit tests, 14 CLI smoke tests, 11 terminal wizard journeys, 12 reproducible binary targets built twice with bit-for-bit hash agreement. Hosted CI `NOT_RUN`. |
| 15 | **Evidence Integrity** | **PASS** | All 65 developer evidence files verified against `SHA256SUMS` (`543fc6d40d02cda292d2d7a69deef5bd50152b521920143698d05bc63a2dfef2`). Independent QA ledger with 11 verified files recorded under `/home/ra/almadial-lab/evidence/m4-03/qa-20261113T021500Z-alm31/` (`0089787be1e155223913526a529638444ee4a346c084bcc15cbc3704e5e57491`). |

---

## 2. Network Exposure and Listener Isolation

Independent peer validation from `almadial-alm29-clean-e` targeting `almadial-alm29-clean-d` confirmed strict host boundary isolation:
- **Management Connectivity:** Port 22 (SSH) connected successfully across both IPv4 (`10.160.90.38:22`) and link-local IPv6 (`fe80::5054:ff:fe8a:96e5:22`), receiving standard OpenSSH banners (`SSH-2.0-OpenSSH_9.9`).
- **Internal Service Isolation:** 18 targeted TCP connection probes timed out completely:
  - Apache HTTP (`80`) and HTTPS (`443`)
  - Asterisk HTTP (`8088`) and WSS (`8089`)
  - MariaDB (`3306`)
  - Asterisk AMI (`5038`)
  - FastAGI (`4577`)
  - SIP TCP (`5060`) and SIP TLS (`5061`)
- **Host Socket Bindings:** Inspection via `ss -lntp` confirmed that all internal services bind exclusively to `127.0.0.1`.
- **Firewall Enforcement:** Managed firewall policy `almadial-s09` enforces priority `-20000` with target `DROP`. External ingress allowances are strictly restricted to TCP port 22 and ICMP/IPv6-ICMP.

---

## 3. SIP Authentication and Transport Verification

Evaluated over trusted WSS connection on `127.0.0.1:8089`:
- **Anonymous Access:** Unauthenticated `REGISTER` returned HTTP/SIP 401 Unauthorized; zero contacts registered.
- **Wrong Password:** Challenged `REGISTER` with invalid password returned 401; zero contacts registered.
- **Unknown Identity:** Challenged `REGISTER` with unknown identity returned 401; zero contacts registered.
- **Authorized Registration:** Valid credentials successfully completed digest challenge (401 -> 200 OK), registering Asterisk contact `alm_s10_9061`.
- **Clean Unregistration:** `REGISTER` with `Expires: 0` returned 200 OK and cleanly removed the active contact. `pjsip show contacts` confirmed zero residual contacts remained.

---

## 4. Credential Protection, Permissions and Runtime Privileges

- **Filesystem Permissions:**
  - `/etc/almadial/secrets/phone-s10`: mode `0600`, parent directory `0700`, owner `root:root`.
  - `/etc/pki/tls/almadial-s08/current/key.pem`: mode `0640`, parent directory `0750`, owner `root:asterisk`.
  - `/var/www/html/vicidial/dbconnect_mysqli.php`: mode `0640`, parent directory `0750`, owner `root:apache`.
  - `/etc/asterisk/manager.conf`: mode `0640`, parent directory `0750`, owner `root:asterisk`.
  - `/var/lib/almadial-install/`: mode `0700`, stage journals mode `0600`, owner `root:root`.
  - Native extended attribute checks confirmed no POSIX ACL overrides exist.
- **Runtime Service Privileges:**
  - MariaDB runs as unprivileged user `mysql:mysql`.
  - Asterisk and FastAGI run as unprivileged user `asterisk:asterisk`.
  - Apache HTTPD and PHP-FPM worker pools run as unprivileged user `apache:apache`.
- **SELinux Status:** Mode is `Enforcing`. All 5 sensitive and runtime paths verified matching policy via `matchpathcon -V`. Bounded audit query confirmed 0 kernel AVC denials during boot and execution.
- **Default Credentials:** Count-only inspection confirmed zero common default passwords in administrator records, synthetic phone accounts, or legacy phone profiles (`callin`, `gs102`).

---

## 5. Synthetic Credential Leakage Verification

Independent execution of `TestALM31CredentialLeakageBoundaries` in [internal/cli/security_qualification_test.go](file:///home/ra/Documents/almadial/internal/cli/security_qualification_test.go) verified that sensitive values are protected across all output surfaces:
- **Sentinel Classes Tested:** `database`, `sip`, `ami`, `administrator`, `private_key`, `session`.
- **Command Output Redaction:** Process failure output containing secrets is strictly redacted to `[REDACTED]`, preserving exit status 7 while withholding raw text.
- **Process Argument Protection:** Passing secret values in command-line arguments is safely rejected before process startup.
- **Operator Logging:** `operatorlog.Redactor` scrubs secrets from stdout, stderr, and file log streams.
- **State Serialization:** Invalid journal inputs containing secrets fail with `journal.ErrInvalid` without echoing secret bytes.
- **Live VM Scans:** Bounded scans of 160 process argument sets, 53 state/log files (3.9 MB), and service log buffers found zero matches against protected scalar values and zero private key markers.

---

## 6. Support-Bundle Security and Redaction

The generated diagnostic export archive (`/tmp/alm31-support-20261112T173255Z-alm31.tar.gz`) adheres to the strict security specification:
- **Archive Properties:** Size 3,516 bytes, mode `0600`, SHA-256 `accf73e377d4e52b1afdb9ef1b26f52dcf5dde6809a8b8b61e650433149c01c5`.
- **Allowlisted Members:** Exactly four regular files with mode `0600`:
  1. `SHA256SUMS`
  2. `manifest.json`
  3. `redaction.json`
  4. `report.json`
- **Redaction Strategy:** Pure allowlist projection. Excludes raw stdout/stderr, configuration dumps, private keys, passwords, database rows, audio recordings, and session cookies. No sensitive credentials or private key markers were present.

---

## 7. TLS and Firewall Defaults

- **TLS Handshake & Hostname Validation:** HTTPS (`:443`) and WSS (`:8089`) complete TLS 1.3 handshakes when validated against the private laboratory CA. Negative tests confirmed that hostname mismatches fail with OpenSSL verification code 62, and untrusted CAs fail with verification code 20.
- **Diagnostic Warning Contract:** The 14-day lab certificate expiry generates an expected `WARN` (exit code 10) in `status` and `doctor`, which is the intended informational behavior for short-lived certificates.
- **Firewall Persistence:** S09 policy remains active with zero drift between runtime and permanent rule sets. External network access is strictly confined to SSH management.

---

## 8. Managed-State Preservation

- **File Integrity (107 Items):** All 107 managed configuration files, journals, receipts, TLS certificates, cron definitions, and monitor audio files retained identical bytes, permissions, and ownership.
- **System Invariance:** Package inventory (731 RPMs), effective nftables ruleset digest, SELinux mode, and systemd service invocation identities remained identical.
- **Database Non-Mutation:** Read-only diagnostics executed with zero database writes. Installer planning returned `actions: []`, `no_op: true`. The normal authenticated web health probe updated only the expected `vicidial_users.last_login_date` and `modify_stamp` audit fields (529 of 531 checks identical), as previously established in ALM-30.

---

## 9. Comprehensive Regression Verification

All tests were independently executed locally using Go 1.27.3:

| Regression Check | Command | Observed Result | Status |
|---|---|---|:---:|
| **Go Vet** | `go vet ./...` | Clean exit 0; zero issues | **PASS** |
| **Go Unit Tests** | `go test -json ./...` | 1,464 passing events across 21 packages (2 skipped) | **PASS** |
| **Go Race Detector** | `go test -race ./...` | Clean exit 0; zero data races | **PASS** |
| **Go Mod Verify** | `go mod verify` | All modules verified clean | **PASS** |
| **Go Mod Tidy Check** | `go mod tidy -diff` | Clean exit 0; zero drift | **PASS** |
| **Go Formatting** | `gofmt -l .` | Zero unformatted files | **PASS** |
| **Scaffold Validation** | `python3 scripts/validate_scaffold.py` | 11 files, 4 skills validated | **PASS** |
| **Python Unit Tests** | `python3 -m unittest discover -s scripts/tests -v` | 222 tests passed in 2.60s | **PASS** |
| **Git Diff Whitespace** | `git diff --check` | Clean exit 0; zero whitespace/conflict errors | **PASS** |
| **CLI Smoke Cases** | Built binary smoke matrix | 14/14 test cases passed | **PASS** |
| **Terminal Journeys** | `scripts/test_terminal_wizard.py` | 11 journeys passed (exit 0) | **PASS** |
| **Reproducible Builds** | 12 plain command targets built twice | 12/12 targets bit-for-bit identical SHA-256 | **PASS** |
| **Hosted CI** | Remote workflow triggers | No remote push authorized | **NOT_RUN** |

---

## 10. Evidence Verification Ledgers

### 10.1 Developer Evidence Ledger

- **Directory:** `/home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/`
- **Ledger Digest:** `543fc6d40d02cda292d2d7a69deef5bd50152b521920143698d05bc63a2dfef2` (`SHA256SUMS`)
- **Audit Verification:** All 65 covered evidence files verified with 100% SHA-256 agreement.

### 10.2 Independent QA Evidence Ledger

- **Directory:** `/home/ra/almadial-lab/evidence/m4-03/qa-20261113T021500Z-alm31/`
- **Ledger Digest:** `0089787be1e155223913526a529638444ee4a346c084bcc15cbc3704e5e57491` (`SHA256SUMS`)
- **Contained Evidence Files:**
  - `identity.json`
  - `ledger-verification.json`
  - `network-exposure.json`
  - `sip-authentication.json`
  - `permissions-and-privileges.json`
  - `credential-leakage.json`
  - `support-bundle.json`
  - `tls-and-firewall.json`
  - `state-preservation.json`
  - `regression-summary.json`
  - `qa-results-summary.json`

---

## 11. Final Independent QA Verdict

All canonical acceptance criteria for Work Package ALM-31 / M4-03 have been thoroughly and independently verified. Network exposure is strictly bounded to SSH management, SIP authentication rejects unauthorized registrations, sensitive files and runtime privileges adhere to the approved security defaults, and diagnostic exports do not leak credentials.

The independent QA verdict is formally declared:

```text
ALMADIAL_M4_03_SECURITY_ACCEPTED
```
