# ALM-8 Independent QA and M0 Acceptance Report

- **Task ID:** `rwpendvaoqpi9mrzgskej6qb` (ALM-7 / reference) / `ftlf2grvt89s3h4nv1yz4x11` (ALM-8 / target)
- **Task Title:** Repeat the complete baseline and propose stack-lock acceptance
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`)
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-06T22:35:00Z
- **Reference Target VM:** `almadial-alma10` (`10.160.90.197`)
- **Reproduction Target VM:** `almadial-alma10-repeat` (`10.160.90.147`)
- **Final Verdict:** `ALMADIAL_M0_08_ACCEPTED`

---

## 1. Governance and Scope Boundary Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-8 (`ftlf2grvt89s3h4nv1yz4x11`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Comment:** Verified existing implementation evidence comment `nuye65m8rvchsvn8eba8ied0`.
- **Dependency Graph:**
  - Upstream ALM-7 (`rwpendvaoqpi9mrzgskej6qb`): Verified status `done` (Accepted).
  - Relation ALM-7 → ALM-8 (`sq76uz00g3qczpiwnamj1d26`): Verified intact.
  - Relation ALM-8 → ALM-9 (`yuko5stwg99z60pmy48byx7g`): Verified intact.
  - Downstream ALM-9 (`y04mu7040nia07ii8ny8b3yr`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Approved Product Scope Realignment
In accordance with `docs/planning/alm-8-product-scope-realignment.md`:
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Accepted M0 Scope Boundary:** Reproducible AlmaLinux 10 single-server installation and verified local/LAN compatibility.
- **External Network Scope:** Direct-public, 1:1 NAT, and remote-home-agent network topologies are deployment-specific operational concerns. They are explicitly NOT universal single-server installer prerequisites. Their test statuses remain `NOT_RUN` under M0 and are formally preserved under ALM-27 (`ea1axf0373fggib2v5u7ciyv`) and ALM-29 (`xu7wxvch94ii2iw38hpb8d58`).
- **No Scope Expansion:** ALM-8 does not expand into multi-node clustering, public-NAT certification, or remote-agent deployment platforms.

---

## 2. Clean Installation and Environment Provenance

### 2.1 VM Infrastructure
- **Reference Host:** `almadial-alma10` (Multipass / QEMU, IP `10.160.90.197`).
- **Reproduction Host:** `almadial-alma10-repeat` (Multipass / QEMU, IP `10.160.90.147`).
- **Base Image:** Official AlmaLinux 10.2 cloud image (`AlmaLinux-10-GenericCloud-10.2-20261012.0.x86_64.qcow2`, SHA-256 digest prefix `cbee4726fc0f...`).
- **Hardware Profile:** 4 vCPUs, 8 GiB RAM, 80 GiB storage allocation.
- **Clean Snapshot:** `almadial-alma10-repeat.clean-baseline` intact; 0 snapshot restorations performed during independent QA.
- **Kernel & OS:** Linux `6.12.0-211.64.1.el10_2.x86_64`, AlmaLinux release 10.2 (Sea Lion).

---

## 3. Compatibility Corrections and Engine Integration

Independent review audited all code modifications applied to achieve AlmaLinux 10 compatibility:

### 3.1 PJSIP Parked Channel Resolution in `AST_update.pl`
- **Issue:** Asterisk 22 uses PJSIP channel naming (`PJSIP/9051-00000001`). Upstream VICIdial `AST_update.pl` historically checked `live_channels` using SIP-specific patterns, causing premature deletion of parked channels from `parked_channels`.
- **Correction:** `AST_update.pl` was modified to route PJSIP channels to check `live_sip_channels` with `park_CID.agi`, matching Asterisk 22 channel naming semantics.
- **Verification:** Verified during live park/resume cycles on the second VM (`agent5-resume2-queue.json`, `postboot-park-ui-samples.json`). Parked channels remained intact in MariaDB during the park hold period without premature clearing.

### 3.2 MariaDB Strict Mode Visibility in `conf_exten_check.php`
- **Issue:** Under MariaDB 10.11 strict SQL mode (`STRICT_TRANS_TABLES`), repeated fast batch inserts to event tables caused duplicate key collisions (Error 1062).
- **Correction:** `conf_exten_check.php` was amended to use `ON DUPLICATE KEY UPDATE event_end_epoch=VALUES(event_end_epoch);`, ensuring idempotency without globally relaxing MariaDB `sql_mode`.
- **Verification:** Verified that `almadial_m005` operates with `STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION` with zero SQL errors logged.

### 3.3 Perl 5.40 Module Architecture
- **Audit:** AlmaLinux 10 ships Perl 5.40.2. Architecture-specific Perl shared objects (`.so`) must reside under `/usr/local/lib64/perl5/5.40/`, while pure-Perl modules reside under `/usr/local/share/perl5/5.40/`.
- **Verification:** Verified on the second VM via `reference-perl-path-check.json` and live script execution.

---

## 4. Live Runtime Verification and Health Audit

Targeted live inspection was conducted on reproduction VM `almadial-alma10-repeat` (`10.160.90.147`):

### 4.1 Systemd Services Status
| Service | Status | Sub-State | Mode |
|---|---|---|---|
| `asterisk.service` | `active` | `running` | Native systemd service |
| `mariadb.service` | `active` | `running` | Native systemd service |
| `httpd.service` | `active` | `running` | Native systemd service |
| `php-fpm.service` | `active` | `running` | Native systemd service |
| `almadial-m006-fastagi.service` | `active` | `running` | Custom FastAGI manager |

### 4.2 Network Listeners and Security Isolation (`ss -tulpn`)
- **MariaDB (TCP 3306):** 0 listeners. `skip_networking=1` active; communication is restricted entirely to `/var/lib/mysql/mysql.sock`.
- **Asterisk AMI (TCP 5038):** Bound strictly to `127.0.0.1:5038` (loopback only). No external exposure.
- **VICIdial FastAGI (TCP 4577):** Bound strictly to `127.0.0.1:4577` (loopback only). No external exposure.
- **Apache HTTP (TCP 8080):** Bound strictly to `127.0.0.1:8080` (loopback only).
- **Apache HTTPS (TCP 8443):** Bound to `*:8443` with TLS encryption enabled.
- **Asterisk HTTP (TCP 8088):** Bound strictly to `127.0.0.1:8088` (loopback only).
- **Asterisk WSS (TCP 8089):** Bound to `0.0.0.0:8089` with TLS/WSS enabled for WebRTC signaling.
- **SELinux:** Verified mode `Enforcing` (targeted policy).

### 4.3 Asterisk 22.12.0 State
- **Active Channels:** 0 channels, 0 active calls (clean quiescent state).
- **ConfBridge:** 0 active conferences.
- **PJSIP Endpoints:** Configured endpoints 9001, 9051, 9052, 9053 intact.
- **Dialplan:** Context `9001@alm3-echo` present, routing to native `Echo()` application.

### 4.4 MariaDB 10.11.18 State
- **Schema:** `almadial_m005` containing 345 tables.
- **VICIdial Version:** `2.14b0.5`, schema version `1748`.
- **SQL Mode:** `STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.
- **Live Agents:** 0 active agent sessions (`vicidial_live_agents`).
- **Live Channels:** 0 active SIP channels (`live_sip_channels`).
- **Parked Channels:** 0 active parked calls (`parked_channels`).
- **Manager Queue:** 0 pending actions (`vicidial_manager`).
- **Heartbeat:** `server_updater` verified active with recent timestamp updates.

### 4.5 VM Reboot Survival
- Codex evidence records two full VM reboots of `almadial-alma10-repeat`.
- Post-reboot health artifacts (`after-full-reboot.json`, `postboot-agent-media-analysis.json`, `reboot2-native-9001-local.wav`) demonstrate that Asterisk, MariaDB, HTTPD, PHP-FPM, and FastAGI recovered automatically with correct state preservation and zero operator intervention required.

### 4.6 Media Re-Analysis
Independent re-analysis was performed using `ffprobe` and `ffmpeg volumedetect` on telephony recordings:
- `repeat-telephony-1-two-party-9051-remote.wav` (PCM 16-bit, 8000 Hz, mono, valid non-silent audio, mean volume -24.1 dB).
- `repeat-telephony-2-two-party-9051-remote.wav` (PCM 16-bit, 8000 Hz, mono, valid non-silent audio, mean volume -23.8 dB).
- `telephony1/recordings/alm6-repeat-telephony-1-1793994536.wav` (PCM 16-bit, 16000 Hz, mono ConfBridge recording).
- `telephony2/recordings/alm6-repeat-telephony-2-1793997243.wav` (PCM 16-bit, 16000 Hz, mono ConfBridge recording).
- `repeat-agent-5-9051-remote.wav` and `repeat-agent-postboot-9051-remote.wav` (VICIphone WebRTC bidirectional audio verified).

---

## 5. Cross-VM Reproducibility Comparison

| Attribute | Reference VM (`almadial-alma10`) | Reproduction VM (`almadial-alma10-repeat`) | Assessment |
|---|---|---|---|
| Installed RPM Count | 744 packages | 744 packages | Identical |
| Kernel Version | `6.12.0-211.64.1.el10_2.x86_64` | `6.12.0-211.64.1.el10_2.x86_64` | Identical |
| Asterisk Version | Asterisk 22.12.0 | Asterisk 22.12.0 | Identical |
| PJPROJECT Version | 2.17 (bundled) | 2.17 (bundled) | Identical |
| MariaDB Version | 10.11.18-MariaDB | 10.11.18-MariaDB | Identical |
| Apache HTTPD | 2.4.63 (AlmaLinux) | 2.4.63 (AlmaLinux) | Identical |
| PHP-FPM Version | 8.3.36 | 8.3.36 | Identical |
| Perl Version | 5.40.2 | 5.40.2 | Identical |
| VICIdial Revision | r4037 | r4037 | Identical |
| VICIdial DB Schema | 1748 (345 tables) | 1748 (345 tables) | Identical |
| VICIphone Asset Stack | 3.0 (SIP.js 0.20.1) | 3.0 (SIP.js 0.20.1) | Identical |
| SELinux Status | Enforcing | Enforcing | Identical |
| Differences Identified | Instance UUID, IP, Hostname, TLS certs, `kernel-headers` (211.73.1 vs 211.74.1) | Expected non-substantive variance |

---

## 6. Security and Persistence Audit

1. **Credentials:** No plain-text passwords or secret keys are checked into Git. Test credentials in lab environments are isolated and documentation-only.
2. **SELinux:** Maintained in `Enforcing` mode throughout installation and runtime without requiring policy disablement.
3. **Firewall / Port Boundaries:** MariaDB, AMI, FastAGI, and local HTTP are strictly loopback/socket-bound. Publicly exposed ports are restricted to TLS-enabled HTTPS (8443) and WSS (8089).
4. **Idempotency & Replay:** Replay build script `m0_08_replay.py` executes reproducibly and verifies all preconditions.

---

## 7. Repository Validation and Automated Checks

All automated verification tools executed locally on repository root:
- `python3 scripts/validate_scaffold.py`: **PASS** (Scaffold structure and rules intact).
- `python3 scripts/verify_candidate_stack.py`: **PASS** (Candidate stack specification conforms to schema).
- `python3 scripts/verify_config_generation.py`: **PASS** (Deterministic configuration generation verified).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.849s).
- `python3 -m unittest scripts/tests/test_m0_07_assets.py -v`: **PASS** (22/22 passed).
- `python3 -m unittest scripts/tests/test_m0_08_replay.py -v`: **PASS** (5/5 passed).
- `python3 -m unittest scripts/tests/test_m0_06_runtime.py -v`: **PASS** (18/18 passed).

---

## 8. Stack-Lock Acceptance Recommendation

- **Current Status:** `candidate` in `manifests/stack.lock.yaml`.
- **Target Target Lock:** `manifests/almalinux10-single.lock.yaml`.
- **Recommendation:** **ACCEPT_FOR_LOCAL_LAN_SINGLE_SERVER**.
- **Justification:** The software stack has demonstrated 100% reproducible deployment across two independent AlmaLinux 10.2 clean VMs, surviving reboots, conference mixing, agent sessions, and database strict-mode requirements.
- **Repository Action:** Per repository safety governance, `manifests/stack.lock.yaml` retains its `candidate` label in source until formal release maintainer promotion.

---

## 9. Evidence Artifacts and Cryptographic Hashes

### 9.1 Historical Second-Installation Ledgers
- **Ledger 1 (Second-Installation Complete Evidence):**
  - Path: `/home/ra/almadial-lab/evidence/m0-08/20261106T192640Z/SHA256SUMS`
  - SHA-256: `ebaa490c6650d672121d20f1dbcfdf86038703435b0f49255326b907310619fc`
  - Verification: 475 entries verified, 475 passed, 0 failed.
- **Ledger 2 (Scope Realignment Documentation):**
  - Path: `/home/ra/almadial-lab/evidence/m0-08/scope-realignment-20261106T213215Z/SHA256SUMS`
  - SHA-256: `be8bd1c882bdc6fc12ea2f5516a429f14c04793607d0216146d2d84f89da049b`
  - Verification: 46 entries verified, 46 passed, 0 failed.

### 9.2 Independent QA Evidence Directory
- **QA Directory:** `/home/ra/almadial-lab/evidence/m0-08/qa-20261106T223000Z/` (Permissions `0700`)
- **Ledger SHA256SUMS:** `163782872a06a929987816132808d02f0beb365ac18dce101d6e0f3cfdbe0a91`
- **Sealed Artifacts:**
  - `qa-historical-ledgers.json`: `2bc0bbc2016021214e119785c0f904dcbc28a2f369bf6d01529bcb84c9c98136`
  - `qa-second-vm-health.json`: `cd01dee0a7f2e6f584315f4552721653078f1410e93b70568305379e6fb72db2`
  - `qa-media-reanalysis.json`: `dd33690cbb864df578e7a93c8bf4a38f3649fd99c64ba8ec53cffb257fd690f1`
  - `qa-repository-validation.json`: `f12b78fbef3ddc8f0fe7f9350472d8762db3aeae977ff4186ef938ff16102e1d`
  - `qa-evidence-ledger.json`: `2d9f9ff3ed5b8dcb9d90bcdd353715712670536808e633fd075ea37c9927c51d`

---

## 10. Final Acceptance Verdict

# `ALMADIAL_M0_08_ACCEPTED`

Task ALM-8 satisfies all authorized acceptance criteria under the approved local/LAN single-server product scope. Milestone M0 compatibility research and baseline verification is formally complete and accepted.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-9 / M1-01** (`y04mu7040nia07ii8ny8b3yr`): *Implement Go Foundation, Host Preflight, and Plan Engine Architecture*.
