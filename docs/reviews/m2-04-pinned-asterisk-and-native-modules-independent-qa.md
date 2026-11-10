# ALM-19 / M2-04 Independent Asterisk 22.12.0 Provisioning and Native Module QA Report

- **Task ID:** `nwk5zbmgwuuhsnlsh92sxl1v` (ALM-19 / M2-04)
- **Task Title:** Provision pinned Asterisk and verify native modules
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-10T09:12:00+08:00 (2026-11-10T01:12:00Z)
- **Repository Baseline Commit:** `7b3aa6e7349c8a32658c354d5cc86953e2ec16ec` (Accepted ALM-18 commit)
- **Developer Baseline HEAD:** `631cbe61b2ac6f4d7fa5665438c87fce2318dbd6` (includes intervening operator preview logout fix)
- **Current Working HEAD:** `631cbe61b2ac6f4d7fa5665438c87fce2318dbd6`
- **Final Verdict:** `ALMADIAL_M2_04_ACCEPTED`

---

## 1. Governance and Independent Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-19 (`nwk5zbmgwuuhsnlsh92sxl1v`), number 19 in Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner. No task reassignment or self-assignment occurred.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`).
- **Task Status Prior to Review:** `in-review` (QA Review, column `in73pv6vxczf7usmb8c03msz`).
- **Implementation Evidence Comment:** Verified comment `b951uhd5rwbzy12jkoo4ja2v` posted by Codex on 2026-11-09T23:09:41.440Z matches recorded implementation details, file counts, and digests.
- **Dependency Graph:**
  - Relation ALM-17 $\rightarrow$ ALM-19 (`s8822axs1a6cmyxg9fa698eg`): Verified intact (`blocks`), source ALM-17 (`ou5231ijs6ilp28howhd6xs1`) status `done` (Accepted).
  - Relation ALM-19 $\rightarrow$ ALM-20 / M2-05 (`w2s0nv3r754tafaacmf18zd2`): Verified intact (`blocks`), target ALM-20 (`jj53k7whedrsmfdamutw1wgz`) status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Authoritative Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Scope:** ALM-19 implements Stage S04 (`internal/provision/asterisk.go`, `asterisk_source.go`, `asterisk_native_linux.go`, `asterisk_stage_linux.go`, `asterisk_files.py`, and `cmd/almadial-asterisk`), responsible for compiling and installing pinned Asterisk 22.12.0 with bundled PJPROJECT 2.17, configuring owned systemd and configuration includes, starting `asterisk.service` under `asterisk:asterisk`, verifying the exact 40-module runtime closure, and enforcing local-only management access.
- **Separation of Concerns:** ALM-19 does not perform live SIP phone calls, deploy VICIphone, import VICIdial Perl cron jobs, or expose public SIP/WSS transports.
- **Candidate Manifest Authorization:** `manifests/stack.lock.yaml` remains strictly `candidate`. `InstallationAuthorized()` continues to deny production installation. Laboratory provisioning in `cmd/almadial-asterisk` requires explicit, cryptographically bound `AsteriskLabAuthorization` with scope `ALM-19/disposable-lab`.

---

## 2. Protected Evidence Ledger Verification

### 2.1 Developer Evidence Ledger
- **Developer Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-04/20261110T060000-alm19/`
- **Master Ledger:** `SHA256SUMS.final`
  - Computed SHA-256: `c48e884b546f1929e8c9d9cd891b29f2dfcf68495eac387e85b34e40c55324ad` (exact match)
  - File Count: 260 entries
  - Verification: `sha256sum -c SHA256SUMS.final` executed with 100% OK across all 260 files.
- **Core Ledger:** `SHA256SUMS.core`
  - Computed SHA-256: `7c186571ea42614f8f30d2d56ffcf6d875bd8da21ad7a700c6afbcd03b36cdb0` (exact match)
  - File Count: 253 entries
  - Verification: `sha256sum -c SHA256SUMS.core` executed with 100% OK across all 253 files.

### 2.2 Independent QA Evidence Ledger
- **QA Evidence Directory:** `/home/ra/almadial-lab/evidence/m2-04/qa-20261110T073000-alm19/` (mode `0700`, files `0600`)
- **Master Ledger SHA-256:** `1f0689a08ce617ed8ed46c9670e293eadb31525fb25c47f156f3ca29de0a7897`
- **Artifacts Preserved (15 files):**
  1. `vm-identity.json`: Target instance `almadial-alm16-rpm` (`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`) and snapshot tree.
  2. `asterisk-version.txt`: Live `asterisk -V` and `core show version` output (Asterisk 22.12.0).
  3. `systemd-asterisk.json`: Live systemd properties (`active`, `enabled`, `MainPID=54288`, `User=asterisk`, `Group=asterisk`, `InvocationID=1a82f6bccb5a47ccb5f2d89414ad9902`).
  4. `modules-live-matrix.json`: Live verification of all 40 required native modules running in the active process.
  5. `listeners.txt`: Socket inspection (`ss -lntup`) confirming AMI (`127.0.0.1:5038`), HTTP (`127.0.0.1:8088`), and zero public SIP listeners.
  6. `pjsip-transports.txt`: Live `pjsip show transports` output confirming no configured public SIP transports (`No objects found.`).
  7. `confbridge-live.txt`: Live ConfBridge CLI commands confirming `default_bridge` profile.
  8. `agi-live.txt`: Live Asterisk AGI CLI commands confirming `res_agi` operational readiness.
  9. `selinux-audit.txt`: Live SELinux inspection confirming `Enforcing`, `unconfined_service_t`, and 0 AVC denials.
  10. `installed-files-sha256.txt`: Live VM file verification for all 46 installed Asterisk files matching `installed-inventory.json`.
  11. `baseline-preservation.json`: Unchanged MariaDB 10.11.18 baseline (schema 1748, 345 tables, 1011 indexes), 725 installed RPMs, 9 repos, and intact empty recording directories.
  12. `source-signature-verification.txt`: Archive digests and `gpgv` detached signature verification against key `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`.
  13. `go-test-qa.json`: Full Go 1.27.3 test/race and Python test suite execution results.
  14. `reproducible-build-qa.json`: `almadial-asterisk` build reproduction matching SHA-256 `5d73cad01f3a9c8758ec542692760e4ff8b0c8dd0f68be20448290ba10aa860e`.
  15. `noop-verification.json`: Fresh assessment `almadial-asterisk lab-run` execution resulting in `no_op: true`, zero traces, and unchanged systemd `MainPID`/`InvocationID`.

---

## 3. Source and Cryptographic Provenance Verification

- **Asterisk Source Archive:** `asterisk-22.12.0.tar.gz`
  - SHA-256: `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` (PASS)
- **Detached Signature:** `asterisk-22.12.0.tar.gz.asc`
  - Signed by: Digium / Asterisk Development Team
  - Key Fingerprint: `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`
  - Verification: `gpgv --keyring ./keyring.gpg asterisk-22.12.0.tar.gz.asc asterisk-22.12.0.tar.gz` verified with `Good signature` (PASS).
- **Bundled PJPROJECT Archive:** `pjproject-2.17.tar.bz2`
  - SHA-256: `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` (PASS)
- **Source Tree Verification:** Verified `LoadAsteriskSource` root extraction checks, `.version`, Makefile/configure, bundled PJPROJECT contract, PJSIP, ConfBridge, and `res_agi.c`.

---

## 4. Live Laboratory VM Inspection (`almadial-alm16-rpm`)

- **Host Identity:** `685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`, AlmaLinux 10.2 x86_64.
- **Pre-qualification Snapshot:** `alm19-pre-asterisk-20261110` verified intact. No snapshot restore was executed.
- **Asterisk Binary and Libraries:**
  - `/usr/sbin/asterisk`: mode `0755`, root:root, dynamically links to `libasteriskssl.so.1` and `libasteriskpj.so.2`.
  - `/usr/lib/libasteriskssl.so.1` & `/usr/lib/libasteriskpj.so.2`: mode `0755`, root:root, registered in dynamic linker cache.
- **XML Documentation Resources:**
  - `/var/lib/asterisk/documentation/core-en_US.xml`
  - `/var/lib/asterisk/documentation/appdocsxml.xslt`
  - `/var/lib/asterisk/documentation/appdocsxml.dtd`
  - Mode `0644`, root:root, required for native Stasis and CLI doc initialization.
- **Systemd Service (`asterisk.service`):**
  - Unit path: `/etc/systemd/system/asterisk.service` (mode `0644`, root:root).
  - ExecStart: `/usr/sbin/asterisk -f -C /etc/asterisk/asterisk.conf`.
  - Execution identity: `User=asterisk`, `Group=asterisk`, `UMask=0027`, `RuntimeDirectory=asterisk` mode `0750`.
  - Service status: `active (running)`, `enabled`, `MainPID=54288`.
- **40 Required Native Modules:**
  - Querying `asterisk -rx "module show"` against `lab/module-matrix.json` confirms all 40 modules are actively loaded in memory:
    - PJPROJECT/PJSIP (13): `res_pjproject`, `res_pjsip`, `chan_pjsip`, `res_pjsip_config_wizard`, `res_pjsip_authenticator_digest`, `res_pjsip_endpoint_identifier_user`, `res_pjsip_endpoint_identifier_ip`, `res_pjsip_registrar`, `res_pjsip_session`, `res_pjsip_sdp_rtp`, `res_pjsip_nat`, `res_pjsip_transport_websocket`, `res_pjsip_pubsub`.
    - Sorcery (4): `res_sorcery_config`, `res_sorcery_memory`, `res_sorcery_astdb`, `res_sorcery_memory_cache`.
    - Security & HTTP (3): `res_crypto`, `res_http_websocket`, `res_srtp`.
    - RTP & Timing (2): `res_rtp_asterisk`, `res_timing_timerfd`.
    - Dialplan & Applications (6): `pbx_config`, `app_echo`, `app_confbridge`, `res_speech`, `res_agi`, `app_mixmonitor`.
    - Bridges (4): `bridge_simple`, `bridge_builtin_features`, `bridge_builtin_interval_features`, `bridge_softmix`.
    - Codecs & Formats (5): `codec_ulaw`, `codec_alaw`, `codec_resample`, `format_pcm`, `format_wav`.
    - Dialplan Functions (3): `func_callerid`, `func_volume`, `func_jitterbuffer`.
  - Loaded modules count: exactly 40 required dynamic modules + core builtins. Missing modules: 0.
- **PJSIP Transports and Socket Isolation:**
  - `asterisk -rx "pjsip show transports"` outputs `No objects found.`. Zero public SIP transports exist.
  - Sockets (`ss -lntup`):
    - TCP `127.0.0.1:5038` (AMI) — loopback only, displayconnects=no, no administrative users.
    - TCP `127.0.0.1:8088` (HTTP) — loopback only, tlsenable=no.
    - UDP `0.0.0.0:48542` and `[::]:51016` — verified as ephemeral response wildcard sockets opened by bundled PJPROJECT's internal DNS resolver (`resolver.c`).
- **SELinux Verification:**
  - Mode: `Enforcing`.
  - Process Domain: `system_u:system_r:unconfined_service_t:s0` (distribution vendor default for custom services; no artificial custom policy claimed).
  - File Contexts: `/usr/sbin/asterisk` (`bin_t`), `/etc/asterisk/` (`etc_t`), `/var/lib/asterisk/documentation/` (`var_lib_t`), `/var/spool/asterisk/` (`var_spool_t`).
  - Audit Log: `ausearch -m avc -ts today | grep asterisk` returns zero denials.
- **Configuration File Ownership:**
  - Base configuration files in `/etc/asterisk/` contain only `#include almadial-s04-<name>.conf`.
  - Owned includes (`almadial-s04-*.conf`): mode `0640`, `root:asterisk`.
  - No `make samples` was run; no foreign or operator files clobbered.
- **Baseline Preservation:**
  - MariaDB: Version `10.11.18-MariaDB` active, schema `db_schema_version=1748`, 345 tables, 1011 indexes intact.
  - RPM Packages: Exactly 725 installed RPM packages intact (0 added, 0 removed).
  - Repositories: Exactly 9 repository files in `/etc/yum.repos.d/` intact.
  - Storage: `/var/spool/asterisk/monitor` and `/var/spool/asterisk/monitorDONE` empty, mode `0750`, `asterisk:asterisk`.

---

## 5. Recovery, Interruption, and NO_OP Re-qualification

- **Defect Reproduction and Correction:**
  - The developer recorded two initial failure scenarios: omission of versioned runtime libraries (dynamic linker failure) and omission of generated XML documentation (native Stasis startup failure).
  - The repair actions (`asterisk.01.libraries` and `asterisk.01.resources`) repaired the installed runtime closure using receipt-backed reconciliation without recompiling or altering foreign files.
- **Independent NO_OP Assessment:**
  - Executed `almadial-asterisk lab-run` in a fresh mode `0700` state directory on `almadial-alm16-rpm`.
  - Result: `no_op: true`, `traces: []`, `actions: []`.
  - Systemd `MainPID` (54288) and `InvocationID` (`1a82f6bccb5a47ccb5f2d89414ad9902`) remained unchanged.
  - Proves zero unnecessary compilation, zero service restarts, zero file clobbering, and clean idempotency.

---

## 6. Code Quality, Test Suite, and Reproducible Builds

- **Go Code Standards (Go 1.27.3 linux/amd64):**
  - `gofmt -l cmd/ internal/`: Clean (0 unformatted files).
  - `go vet ./...`: Clean (exit code 0).
  - `go mod verify`: Clean (`all modules verified`).
- **Go Test Suite:**
  - Full suite (`go test -json ./...`): 156 top-level pass, 0 fail, 1 skip (`TestM0Artifacts` opt-in fixture); 1065 total passing events.
  - Race detector (`go test -race -json ./...`): 156 top-level pass, 0 fail, 1 skip; 1065 total passing events; clean race detector report.
- **Python Test Suite:**
  - `scripts/validate_scaffold.py`: PASS.
  - `scripts/verify_candidate_stack.py`: PASS.
  - `scripts/verify_config_generation.py`: PASS.
  - Unit tests (`python3 -m unittest discover -s scripts/tests -v`): 137 tests executed, 0 failures, 0 errors, status OK.
- **Reproducible Go Builds:**
  - `almadial-asterisk` built with `CGO_ENABLED=0 -mod=readonly -trimpath -buildvcs=false`.
  - SHA-256: `5d73cad01f3a9c8758ec542692760e4ff8b0c8dd0f68be20448290ba10aa860e` independently reproduced and verified byte-for-byte.

---

## 7. QA Acceptance Verdict and Milestone Handoff

### Verdict: `ALMADIAL_M2_04_ACCEPTED`

The implementation of Stage S04 / ALM-19 satisfies all canonical acceptance criteria:
1. Pinned Asterisk 22.12.0 and PJPROJECT 2.17 source archives and GPG signatures are cryptographically verified.
2. Compiled and installed runtime closure provides all 40 required native modules actively loaded in Asterisk.
3. Systemd `asterisk.service` runs reliably as `asterisk:asterisk` with explicit file ownership and permissions.
4. SELinux runs in `Enforcing` mode with 0 AVC denials.
5. Management sockets are strictly restricted to localhost (`127.0.0.1:5038` and `127.0.0.1:8088`), with zero public SIP transports configured.
6. MariaDB database baseline (schema 1748, 345 tables), RPM inventory (725 packages), and storage hierarchies are fully preserved.
7. Fresh NO_OP evaluation demonstrates clean idempotency with zero service restarts and zero state drift.

### Milestone Handoff
- **Kaneo Task Finalization:** Task ALM-19 (`nwk5zbmgwuuhsnlsh92sxl1v`) transitioned to `done` (Accepted), retaining Codex ownership.
- **Next Task:** ALM-20 / M2-05 (`jj53k7whedrsmfdamutw1wgz`) — *Provision pinned VICIdial and approved configuration integration*.
