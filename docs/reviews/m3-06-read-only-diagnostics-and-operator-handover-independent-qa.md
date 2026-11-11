# AlmaDial — ALM-28 / M3-06: Independent Diagnostics and Operator Handover QA Report

## Acceptance Status and Governance

- **Task ID:** `hptrjdzgsafy21edd6nqiigc`
- **Canonical Work Package:** ALM-28 / M3-06
- **Task Title:** Deliver read-only diagnostics and operator handover
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted ALM-27 Baseline Commit:** `f71c68c898015d20dfeb9385f2c3903d2e576f9a`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (`10.160.90.233`, Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `almadial-alm16-rpm.alm25-pre-firewall-20261111` (preserved)
- **Developer Evidence Comment:** `oqsz0oy0kic4bh903ndmgydy`
- **Developer Verdict:**
  ```text
  ALMADIAL_M3_06_DIAGNOSTICS_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_06_DIAGNOSTICS_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S12 introduces read-only operational diagnostics (`almadial status`, `almadial doctor`), explicit evidence-export support bundle generation (`almadial evidence export` / `almadial support-bundle`), and practical operator handover documentation for NOC maintenance on native AlmaLinux 10.2.

Independent QA was conducted across the Go codebase, CLI smoke cases, terminal journeys, live evaluation on laboratory VM `almadial-alm16-rpm`, before-and-after managed-state comparison across 27 system dimensions, support-bundle allowlisting and secret redaction audit, archive integrity verification, operator documentation review, and full regression test execution.

All canonical acceptance criteria for ALM-28 are satisfied:
1. **Accurate Operational Health Status:** `almadial status` executes 39 fixed, bounded diagnostic probes covering systemd units, OS identity, kernel, NTP clock synchronization, SELinux enforcing mode, AVC audit logs, RPM package count, Asterisk version/modules, PJSIP configuration/registration, MariaDB schema, loopback TCP listeners, firewalld policy summary, FastAGI process hierarchy, deferred units, cron definitions, application entrypoints, trusted TLS endpoints, certificate expiration, recording filesystem capacity/permissions, manifest identity, and journal state.
2. **Deterministic Warning and Exit Codes:** Exit code `10` is returned for degraded non-blocking findings (38 OK, 1 WARN: `tls.expiration`, 0 FAIL). The warning accurately identifies the accepted 14-day lab certificate expiring on `2026-11-25 00:45:48 UTC` (within 30 days) while confirming active HTTPS and WSS routes remain fully trusted.
3. **Actionable Doctor Guidance Without Auto-Remediation:** `almadial doctor` presents component identity, actual observation, expected condition, and specific local troubleshooting pointers for each probe. Code audit and host execution confirm zero mutating operations: no service restarts, configuration overwrites, database mutations, firewall reloads, certificate renewals, or installer reconciliation.
4. **Sanitized Support Bundle:** `almadial evidence export --output ...` writes an exclusively created, mode-0600 tar.gz archive containing exactly four allowlisted members (`report.json`, `redaction.json`, `manifest.json`, `SHA256SUMS`). Archive size is 3,563 bytes (well within the 1 MiB limit). In-archive checksums verify 100% OK. In-memory and archive text scans confirm zero leakage of credentials from `/etc/almadial/secrets/`, private keys, SIP authorization headers, session cookies, database passwords, raw logs, or audio recordings.
5. **Strict Read-Only Guarantee:** Comparing all 27 managed baseline fields before and after executing `status`, `doctor`, and `evidence export` confirms zero managed-state drift (`managed_state_identical: true`).
6. **NOC Usability & Operator Handover:** `docs/operations/almadial-operator-handover.md` provides clear, concise instructions for daily inspection, application access URLs, troubleshooting tables, recovery boundaries, and support collection without requiring internal knowledge of Go action graphs or journals.

---

## 2. Independent Acceptance Matrix

| Criterion | Result | Evidence |
| :--- | :---: | :--- |
| **Status command** | **PASS** | 39 probes evaluated; 38 OK, 1 WARN, 0 FAIL; exit code 10; [status.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/status.stdout) & [status-text.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/status-text.stdout). |
| **Text/JSON consistency** | **PASS** | Both text and `--json` invocations report identical 39 check IDs, statuses, observations, and exit code 10; [status-text.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/status-text.stdout) and [status.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/status.stdout). |
| **Warning and exit-code semantics** | **PASS** | Exit code 10 represents degraded required check; `tls.expiration` WARN mathematically verified against cert validity (`2026-11-11` to `2026-11-25`, <= 30 days remaining); [status.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/status.stdout). |
| **Doctor functionality** | **PASS** | Doctor outputs observations, expectations, and actionable guidance for all 39 checks with zero auto-remediation; [doctor.stdout](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/doctor.stdout). |
| **Strict read-only execution** | **PASS** | 27 managed baseline fields identical before and after diagnostic execution; zero DB writes, zero service restarts, zero firewall changes; [state-preservation-qa.json](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/state-preservation-qa.json). |
| **Support-bundle allowlist** | **PASS** | Exactly 4 allowlisted archive members (`SHA256SUMS`, `manifest.json`, `redaction.json`, `report.json`); projection model prevents arbitrary file collection; [bundle-verification.json](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/bundle-verification.json). |
| **Redaction and secret exclusion** | **PASS** | 5 secret files in `/etc/almadial/secrets/` scanned and confirmed absent; private key headers, SIP authorization, cookies, and passwords absent; [bundle-verification.json](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/bundle-verification.json). |
| **Bundle integrity/size/permissions** | **PASS** | Archive mode `0600`, size 3,563 bytes (<= 1 MiB limit), uncompressed 13,965 bytes (<= 1 MiB); SHA256SUMS and manifest.json verified 100% OK; [bundle-verification.json](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/bundle-verification.json). |
| **Operator handover** | **PASS** | `docs/operations/almadial-operator-handover.md` verified; actionable commands, exit code contract, bootstrap URLs, troubleshooting matrix, and recovery boundaries documented clearly. |
| **S01–S11 preservation** | **PASS** | 731 RPMs, MariaDB schema 1748, Asterisk 22.12.0, legacy phones `callin`/`gs102` (all 107 columns intact), FastAGI active, SELinux Enforcing, firewalld active; [after-baseline.json](file:///home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/after-baseline.json). |
| **Regression tests** | **PASS** | Go vet/fmt/test/race passed (1,442 events, 0 races); Python passed (221 tests); 23 CLI smoke cases; 11 terminal journeys; reproducible builds verified; git diff clean. |
| **Evidence integrity** | **PASS** | Developer core ledger (89 files, SHA-256 `443eef53a2d6b7774ecbe7020094f5ace0fea2e2313e51e022ce78292f01967d`) 100% verified; independent QA ledger sealed (`13641b68...`). |

---

## 3. Detailed Verification Findings

### 3.1 Status & Doctor Operational Observations
- Total checks evaluated: **39**
  - **38 OK:** Services (`mariadb`, `asterisk`, `httpd`, `php-fpm`, `firewalld`, `crond`, `almadial-s06-fastagi`), Host OS (AlmaLinux 10.2), Kernel (6.12.0), Time sync, SELinux (Enforcing), Kernel AVC denials (none in bounded query), RPM inventory (731 packages), Asterisk version (22.12.0) and modules (40 required modules running), PJSIP endpoint configuration and registration status (0 synthetic contacts idle/logged out is expected), Database schema (1748), DNS localhost resolution, Loopback TCP listeners (80, 443, 8088, 8089, 3306, 5038, 4577), Managed firewall policy, FastAGI process hierarchy (1 parent + 5 workers), Deferred units (`updater`, `listener` inactive/disabled), Owned cron (`almadial-s06` inactive), VICIdial entry point (`admin.php` present/readable), Trusted TLS routes (HTTPS and WSS), Recording space (75,548 MiB available) and directory access permissions, Validated source manifest (`stack.lock.yaml`, CANDIDATE), and Stage journal (0 uncompleted actions).
  - **1 WARN:** `tls.expiration` — Certificate expires within 30 days (`2026-11-25 00:45:48 UTC`).
  - **0 FAIL.**
- Exit code: **10** (indicating degraded non-blocking required check).
- Doctor output: For every probe, outputs component state, observation, expected condition, and local inspection commands (e.g., `openssl x509 -noout -dates`, `systemctl status`, `journalctl`). Zero remediation actions performed.

### 3.2 Support Bundle Creation, Allowlisting, and Redaction
- Invocation: `almadial evidence export --output /tmp/alm28-qa-support.tar.gz ...`
- Exit code: **0**
- Mode and Size: Mode `0600`, compressed size 3,563 bytes, uncompressed size 13,965 bytes (under 1 MiB limit).
- Members: Exactly 4 allowlisted files:
  1. `report.json`: JSON serialization of diagnostic findings.
  2. `redaction.json`: Redaction policy, field bounds, and exclusions.
  3. `manifest.json`: Member metadata, byte lengths, and SHA-256 digests.
  4. `SHA256SUMS`: Cryptographic checksums covering `report.json`, `redaction.json`, and `manifest.json`.
- Secret Redaction Audit:
  - Scanned all 5 secret files in `/etc/almadial/secrets/` against uncompressed archive content; 0 secrets found.
  - Scanned for private keys (`BEGIN PRIVATE KEY`), SIP authorization (`Authorization:`), HTTP cookies (`Cookie:`), SIP URIs (`sip:`), and passwords (`password=`); all forbidden patterns absent.

### 3.3 Strict Read-Only Preservation
All 27 managed baseline fields were captured immediately before and after the diagnostic execution sequence:

| Field | Pre-Check Digest / Value | Post-Check Digest / Value | Status |
| :--- | :--- | :--- | :---: |
| `machine_id` | `0fffa9a4683f35b696f0b6ee7330734b` | `0fffa9a4683f35b696f0b6ee7330734b` | UNCHANGED |
| `selinux` | `Enforcing` | `Enforcing` | UNCHANGED |
| `catalog` | `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` | `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` | UNCHANGED |
| `rpm_count` | `731` | `731` | UNCHANGED |
| `rpm_sha` | `113ac0fae3aabc42859282ccce65a1c882ff34bbc68e0c3ebeb0ff9f2a012a20` | `113ac0fae3aabc42859282ccce65a1c882ff34bbc68e0c3ebeb0ff9f2a012a20` | UNCHANGED |
| `phones_sha` | `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04` | `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04` | UNCHANGED |
| `phone_columns`| 107 columns | 107 columns | UNCHANGED |
| `service_identity`| Invocations & monotonic start timestamps | Identical | UNCHANGED |
| `nft` | Ruleset listing | Identical | UNCHANGED |
| `credentials`| `/etc/almadial/secrets` tree | Identical | UNCHANGED |
| `application_config`| `/etc/asterisk` tree | Identical | UNCHANGED |
| `httpd_config`| `/etc/httpd` tree | Identical | UNCHANGED |
| `php_config` | `/etc/php-fpm.d` tree | Identical | UNCHANGED |
| `cron` | `/etc/cron.d` tree | Identical | UNCHANGED |
| `firewall` | `/etc/firewalld` tree | Identical | UNCHANGED |
| `tls` | `/etc/pki/tls/almadial-s08` tree | Identical | UNCHANGED |
| `journals` | `/var/lib/almadial/**/journal.json` digests | Identical | UNCHANGED |
| `recordings` | `/var/spool/asterisk/monitor` tree | Identical | UNCHANGED |
| `recordings_done`| `/var/spool/asterisk/monitorDONE` tree | Identical | UNCHANGED |
| `repositories`| `/etc/yum.repos.d` tree | Identical | UNCHANGED |
| `application_data`| 7 tables (`vicidial_campaigns`, `vicidial_list`, etc.) | Identical | UNCHANGED |
| `identities` | `asterisk`, `apache` user entries | Identical | UNCHANGED |
| `sockets` | `ss -H -lntup` listeners | Identical | UNCHANGED |
| `asterisk_version`| `Asterisk 22.12.0` | `Asterisk 22.12.0` | UNCHANGED |
| `asterisk_modules`| `module show` list | Identical | UNCHANGED |
| `services` | `asterisk`, `mariadb` active states | Identical | UNCHANGED |
| `host_sha` | `machine_id` digest | Identical | UNCHANGED |

Result: **`managed_state_identical: true`** (27 of 27 fields unchanged).

---

## 4. Evidence Artifacts

Independent QA evidence directory:
`/home/ra/almadial-lab/evidence/m3-06/qa-20261112T060500Z-alm28/`

Sealed SHA-256 Ledger (`SHA256SUMS` SHA-256: `13641b680145cd46a96efb179d763bcaa18574d8f75b25d2b25ef58335be30bc`):
- `status.stdout`: `0c98f80c65637951a8470a7bfe467ca08a287425f16279f170e791e892c90c68`
- `status-text.stdout`: `23ebc3258db9ca8be0aa207c72ec11aaae113a36dbba33f1a0ec947dfebf1aa8`
- `doctor.stdout`: `e0970bc30f9d924dbca48eb6f4e179e830ee307d0d08a513511ebf1d24c0d01d`
- `bundle.stdout`: `cdcfb013c7a36cb9eb070fb1f2c27b0b63390cbf9073e5ffb5109b8bfe64c9ea`
- `bundle-verification.json`: `6ff4d84fbb3b15cb267675f36e4f3a9e22aa744c852467d32c02aa00c25d80b7`
- `before-baseline.json`: `a4fbe28682a0d7d8e29a8f4cfae34aa3d54070aeb0498b0f279fe1e5509df44f`
- `after-baseline.json`: `a4fbe28682a0d7d8e29a8f4cfae34aa3d54070aeb0498b0f279fe1e5509df44f`
- `state-preservation-qa.json`: `f0e21017df86ee8fa023e101f38e3e7f6d895828ecffcf8175b9f71c2617f631`
- `qa-results-summary.json`: `05e32402dd1240c1ceb79e7be9ea0199e847c9431e51ea4cfeb89f929fc5f037`

---

## 5. Final Recommendation

All canonical acceptance criteria for ALM-28 (M3-06) are satisfied. The diagnostic tooling is deterministic, actionable, strictly read-only, and completely safe for NOC operator use.
Final verdict: **`ALMADIAL_M3_06_DIAGNOSTICS_ACCEPTED`**.
Proceed to local Git acceptance commit and Kaneo status transition to **Accepted** (`done`), strictly preserving Codex ownership.
