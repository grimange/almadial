# AlmaDial Independent QA Acceptance — ALM-6 / M0-06 ConfBridge, AMI, FastAGI and Recording Runtime

**Reviewer / Session:** Google Antigravity (AGY), separate independent reviewer session  
**Reviewer Professional Role:** Principal Asterisk/WebRTC Telephony QA Engineer, Real-Time Media Verification Architect, and VICIdial Runtime Integration Auditor  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)  
**Implementer:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Target Task:** ALM-6 / M0-06 (`xyu5rkqfhnnjl82bq5ywt9t8`), Kaneo Project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Repository Starting HEAD:** `91616c7792757479947a0d7a80401c5953225e72` (ALM-5 baseline: `e99be474bc24462ae7e24bc0ba71e4920bc9cfb4`)  
**Host Target:** `almadial-alma10` (Multipass 1.17.0 / QEMU, AlmaLinux 10.2 x86_64, IP `10.160.90.197`)  
**Review Date / Time:** 2026-11-06 (Session: 2026-11-06T10:42:00+08:00)  
**Protected Evidence Root:** `/home/ra/almadial-lab/evidence/m0-06/20261106T011827Z/` (`evidence-ledger.json` SHA-256 `07089e48285554765b98dcb658442caa0787d178b26e0a7b08cdf85a33bae544`)  
**QA Evidence Path:** `/home/ra/almadial-lab/evidence/m0-06/qa-20261106T104000Z/` (`qa-evidence-ledger.json` SHA-256 `12f13f84848449f8821eb624a991494a6cc453c8460f1a4b4eba5cef1844d51a`)  
**Historical Implementation Report:** Preserved in [`docs/compatibility/m0-06-telephony-runtime.md`](../compatibility/m0-06-telephony-runtime.md)  
**Final Independent QA Verdict:** **`ALMADIAL_M0_06_ACCEPTED`**

---

## 1. Scope, Authority, and Reviewer Independence Confirmation

This review constitutes the formal independent quality assurance evaluation of task **ALM-6 / M0-06: Verify ConfBridge, AMI, AGI and recording lifecycle**.

In accordance with repository governance ([`AGENTS.md`](../../AGENTS.md), [`GEMINI.md`](../../GEMINI.md), [`.agents/skills/almadial-review/SKILL.md`](../../.agents/skills/almadial-review/SKILL.md), [`.agents/rules/almadial-kaneo-governance.md`](../../.agents/rules/almadial-kaneo-governance.md), and [`.agents/rules/almadial-evidence-integrity.md`](../../.agents/rules/almadial-evidence-integrity.md)):
- **Separation of Roles:** Codex performed the telephony runtime implementation, source derivation, guest automation, browser WebRTC tests, and developer self-checks on disposable guest `almadial-alma10`, submitting ALM-6 to `QA Review` (`in-review`) without claiming self-acceptance.
- **Reviewer Authority:** Google Antigravity (AGY) operates as the distinct reviewer under authorized Kaneo identity `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`. AGY does not assume ownership of ALM-6; Codex remains the permanent implementation owner (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`).
- **Scope Boundaries:** Testing is strictly confined to VICIdial r4037-generated WebRTC endpoints (9051, 9052, 9053), Asterisk 22.12.0 ConfBridge room admission (9600001), bidirectional media exchange, supervisor listen-only monitoring and authorized barge, local AMI call control, actual VICIdial FastAGI execution and automatic database row closure, MixMonitor recording lifecycle and content correlation, and Asterisk baseline preservation. Zero external carrier trunks, zero campaign autodialing, and zero snapshot restorations were executed.

---

## 2. Source Baseline and Working Tree State

- **Input Baseline:** Commit `e99be474bc24462ae7e24bc0ba71e4920bc9cfb4` (`test(m0): independently accept VICIdial config generation`).
- **Intermediate Operations Commits:** Three operator portal commits (`3cadd31`, `a7a8d96`, `91616c7`) were inspected; they modify only `operations/portal443/` and do not alter AlmaDial installer, core documentation, or candidate stack contracts.
- **Reviewed ALM-6 Implementation Deliverables:**
  - [`docs/compatibility/m0-06-telephony-runtime.md`](../compatibility/m0-06-telephony-runtime.md)
  - [`scripts/lab/m0_06_fastagi.py`](../../scripts/lab/m0_06_fastagi.py)
  - [`scripts/lab/m0_06_media.py`](../../scripts/lab/m0_06_media.py)
  - [`scripts/lab/m0_06_runtime.py`](../../scripts/lab/m0_06_runtime.py)
  - [`scripts/lab/m0_06_browser.js`](../../scripts/lab/m0_06_browser.js)
  - [`scripts/lab/m0_06_webrtc_client.html`](../../scripts/lab/m0_06_webrtc_client.html)
  - [`scripts/tests/test_m0_06_runtime.py`](../../scripts/tests/test_m0_06_runtime.py)
- **Preserved Untracked Work:** Pre-existing untracked helper [`scripts/verify_agy_overlay.py`](../../scripts/verify_agy_overlay.py) and follow-up document `docs/reviews/m0-05-acceptance-readback-followup.md` were inspected and preserved without staging or modification.
- **Candidate Stack Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) retains `status: candidate`.

---

## 3. Evidence Ledger and Cryptographic Audit

- **Evidence Ledger:** `/home/ra/almadial-lab/evidence/m0-06/20261106T011827Z/evidence-ledger.json` (SHA-256 `07089e48285554765b98dcb658442caa0787d178b26e0a7b08cdf85a33bae544`).
- **Artifact Count:** Exactly **294 files**.
- **Audit Outcome:** 100% of all 294 files were independently verified for byte length and SHA-256 checksum against the ledger. Zero missing or modified files.
- **Key Artifact Hashes Verified:**
  - `final-evidence.tar.gz`: `24afd6dc6d8ee84cebb8cf22a03dae2b6493894b5ec68ed2f2e85be394b345fc`
  - `final-export/alm6-final-1793930610.wav`: `462c40b545c29436b6689416d3a91f9dbebdbca7f5ad709b1b11a2bb0a5bbcba`
  - `final-media-analysis.json`: `3b7bcac2eb5c42fedb1e7a37ca08bbb5fa228e7cf7e4e0bb90c3c02b4ff203b1`
  - `final-recording-analysis.json`: `282b6035487f094d08c20d2069ccfd3c7a9a37993010c893239244bffa022890`
- **Snapshot Integrity:** Snapshot `pre-alm6-telephony-runtime` intact on `almadial-alma10`. Zero snapshot restorations were executed.
- **Protected QA Workspace:** Dedicated review evidence recorded under `/home/ra/almadial-lab/evidence/m0-06/qa-20261106T104000Z/` (mode `0700`, ledger SHA-256 `12f13f84848449f8821eb624a991494a6cc453c8460f1a4b4eba5cef1844d51a`).

---

## 4. Secure PJSIP WebRTC Registration & Topology

1. **Independent Browser Endpoints:**
   - 9051: Participant A (Context `alm5-local`, user phone)
   - 9052: Participant B (Context `alm5-local`, user phone)
   - 9053: Supervisor C (Context `alm6-supervisor`, dedicated `ALM6-MON` template)
2. **Security & Transport Attributes:**
   - Certificate-validated HTTPS/WSS: Connected via `https://almadial-alma10.test:8089` using trusted lab root CA; `ignoreHTTPSErrors: false` strictly enforced.
   - SIP Authentication: Verified PJSIP digest challenge and registration against phone credentials (`conf_secret`). Negative authentication test confirmed: invalid SIP password fails registration and is logged by Asterisk.
   - ICE / DTLS-SRTP: Verified `connected` state, media codec negotiated as `PCMU / 8000 Hz`, cipher `SRTP_AES128_CM_HMAC_SHA1_80`.
   - Contact Management: All contacts cleaned up upon teardown (0 remaining contacts).

---

## 5. ConfBridge Admission & Generated Dialplan Routing

1. **Conference Route Traversal:**
   - Dialplan entry: `alm5-local -> alm6-test,9600001 -> Answer() -> FastAGI(call_log) -> Goto(vicidial-auto-external,473782179600001,1)`.
   - Generated ConfBridge destination: Context `vicidial-auto-external`, extension `_473782179600XXX` executing `ConfBridge(${EXTEN:8},vici_agent_bridge,vici_audio_user)`.
   - Verified live in Asterisk: `dialplan show 9600001@alm6-test` and `dialplan show _473782179600XXX@vicidial-auto-external`.
2. **Room Allocation & Participation:**
   - Room `9600001` allocated conditionally in MariaDB `vicidial_confbridges`. First claim updated 1 row; competing claim updated 0 rows. Released to blank at teardown.
   - Both participants 9051 and 9052 admitted simultaneously into bridge `9600001`.

---

## 6. Bidirectional Conference Media Verification

Using the verified raw PCM WAV files and [`scripts/lab/m0_06_media.py`](../../scripts/lab/m0_06_media.py) with inherited ALM-3 thresholds, independent re-analysis established:

1. **Direction A (9051) → B (9052):**
   - Result: **`PASS`**
   - Fixture pattern match: `0.994059` (threshold: ≥0.90)
   - Envelope correlation: `0.994686` (threshold: ≥0.85)
   - Tone agreement: `0.984848` (threshold: ≥0.90)
   - Estimated delay: `40 ms` (max: 2000 ms)
   - Remote RMS: `0.1154` (threshold: ≥0.02)
   - Sampled frames: 602 paired AudioWorklet frames across 12.0 seconds.
2. **Direction B (9052) → A (9051):**
   - Result: **`PASS`**
   - Fixture pattern match: `0.998020` (threshold: ≥0.90)
   - Envelope correlation: `0.997359` (threshold: ≥0.85)
   - Tone agreement: `1.000000` (threshold: ≥0.90)
   - Estimated delay: `60 ms` (max: 2000 ms)
   - Remote RMS: `0.1147` (threshold: ≥0.02)
   - Sampled frames: 602 paired AudioWorklet frames across 12.0 seconds.

---

## 7. Supervisor Monitoring & Authorized Barge Verification

1. **Listen-Only Monitoring:**
   - Supervisor 9053 joined `9600001` via `alm6-supervisor` using generated `vici_monitor_user` and `vici_monitor_menu`.
   - Received remote mixture matched both primary sources across 281 distinguishable frames with agreement `0.950178` (threshold: ≥0.90).
2. **Negative Audio-Contribution Test:**
   - Tested supervisor transmitted signal while in listen-only monitor mode against Participant A's remote capture.
   - Result: **`FAIL`** (as required; positive source test failed with tone agreement `0.375587` and envelope correlation `0.848019`).
   - Confirms supervisor is strictly muted in the bridge and cannot inject unauthorized audio.
3. **Authorized Barge Operation:**
   - Performed authorized AMI `Redirect` transferring 9053 into `alm6-barge` with `vici_barge_user`.
   - Result: **`PASS`**
   - Fixture pattern match: `0.988201`
   - Envelope correlation: `0.970590`
   - Tone agreement: `1.000000`
   - Estimated delay: `60 ms`
   - Confirms supervisor transitions cleanly to an active audio contributor upon authorization.

---

## 8. Local AMI Operations & Security Audit

1. **Binding & Permissions:**
   - Dedicated test account `[alm6-runtime]` in `/etc/asterisk/manager.conf`: `read=system,call,reporting,agent`, `write=system,call,reporting,agent`.
   - Bound strictly to `127.0.0.1:5038`.
2. **Live Functional & Negative Boundary Testing:**
   - **Invalid Login:** Rejected with `Response: Error`, `Message: Authentication failed`.
   - **Valid Login:** Accepted with `Response: Success`, `Message: Authentication accepted`.
   - **Ping / CoreStatus:** Ping returned `Pong`; CoreStatus returned current calls `0`.
   - **ConfbridgeList:** Returned `Response: Error | Message: No active conferences` when idle.
   - **Command Privilege Denial:** Action `Command` (`core show version`) returned `Response: Error | Message: Permission denied`. Confirms boundary protection against arbitrary command execution.

---

## 9. VICIdial FastAGI Execution & Source Derivation

1. **Source Derivation Verification ([`scripts/lab/m0_06_fastagi.py`](../../scripts/lab/m0_06_fastagi.py)):**
   - Pinned input upstream `FastAGI_log.pl` SHA-256: `f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a`.
   - Derived script SHA-256: `9e90d5a32f1794213f1ea36d33facb9503666fa64f78b3142e37c57163d67f43`.
   - Exact substitution: Fixed MariaDB strict-mode INSERT failure where empty strings were inserted into datetime/numeric fields (`call_log`), changing them to `NULL,0,0,0`. Global `sql_mode` remained strict (`STRICT_TRANS_TABLES`).
   - Private configuration redirection: `/etc/astguiclient-m006.conf`, listener bound to `127.0.0.1:4577`, process user/group `asterisk:asterisk`.
   - Guest deployed `/opt/almadial-m006/FastAGI_log.pl` confirmed with SHA-256 `9e90d5...`.
2. **Live FastAGI Daemon & Database Row Closure:**
   - Verified active systemd service `almadial-m006-fastagi.service` running as `asterisk` with listening socket `127.0.0.1:4577`.
   - Live MariaDB queries on `almadial_m005.call_log` confirmed all final test call rows automatically closed with populated `end_time` and `length_in_sec`:
     - `1793930610.8`: `PJSIP/9051-00000005`, length `47 s`
     - `1793930610.9`: `PJSIP/9052-00000006`, length `45 s`
     - `1793930631.13`: `PJSIP/9053-00000007`, length `26 s`

---

## 10. Recording Lifecycle & Content Correlation

1. **Recording File Integrity:**
   - File: `/var/spool/asterisk/monitor/alm6-final-1793930610.wav` (SHA-256 `462c40b545c29436b6689416d3a91f9dbebdbca7f5ad709b1b11a2bb0a5bbcba`).
   - FFprobe analysis: Mono PCM signed 16-bit, `8000 Hz`, duration `46.70 s`, size `747,244 bytes`. Exit code 0.
   - File descriptor cleanly closed upon `ConfbridgeStopRecord`.
2. **Audio Content Analysis:**
   - Analyzed decoded 16 kHz resampled WAV via [`scripts/lab/m0_06_media.py mixed_recording`](../../scripts/lab/m0_06_media.py):
   - Result: **`PASS`**
   - Both source tones agreement: `0.904968` (threshold: ≥0.90) across 463 distinguishable frames.
   - Confirms both participants' injected signals were captured in the recorded conference audio.
3. **Database Correlation:**
   - Verified live in `almadial_m005.recording_log`: Row 4 recorded start `2026-11-06 10:03:31`, end `2026-11-06 10:04:17`, length `47 s`, filename `alm6-final`, location `/var/spool/asterisk/monitor/alm6-final-1793930610.wav`.
   - Lab-controller ownership acknowledged: Recording was controlled via native AMI and correlated by the lab controller, not via the VICIdial agent UI.

---

## 11. Service Restart, Baseline Preservation & Cleanup

1. **Service Restart & Re-testing:**
   - Asterisk 22.12.0 survived controlled service restart cleanly.
   - Repeated full conference lifecycle test passed after restart.
   - Native endpoint 9001 Echo regression passed: envelope correlation `0.999335`, tone agreement `1.0`.
2. **Final System Read-Back & Cleanup:**
   - Active calls: **0**; active channels: **0**; active conferences: **0**; PJSIP contacts: **0**.
   - MariaDB: Socket only, `skip_networking = ON`, 0 TCP listeners (port 3306 disabled).
   - FastAGI: Listening strictly on `127.0.0.1:4577`.
   - AMI: Listening strictly on `127.0.0.1:5038`.
   - SELinux: **Enforcing**.

---

## 12. Repository Automated Validation

The independent reviewer executed the full repository test suite:

1. `python3 scripts/validate_scaffold.py`: **`PASS`** (11 required files, 4 skills).
2. `python3 -m unittest discover -s scripts/tests -v`: **`96/96 unit tests PASSED`** in 1.677s (including all 15 new ALM-6 tests).
3. `python3 scripts/verify_candidate_stack.py`: **`PASS`** in LOCAL mode (`candidate_consistency: PASS`).
4. `python3 scripts/verify_config_generation.py`: **`PASS`** (`REDACTED_FIXTURE_REPLAY`).
5. `python3 -m py_compile ...`: Clean syntax check on all ALM-6 Python scripts.
6. `node --check scripts/lab/m0_06_browser.js`: Clean JavaScript syntax check.
7. `git diff --check`: Clean; zero trailing whitespace or formatting defects.

---

## 13. Final Acceptance Verdict and Downstream Recommendations

### Acceptance Verdict:
**`ALMADIAL_M0_06_ACCEPTED`**

Task ALM-6 satisfies all mandatory acceptance criteria established in `docs/AlmaDial_Implementation_Plan.md`. VICIdial-generated WebRTC endpoints exchange real bidirectional audio through Asterisk 22.12.0 ConfBridge, supervisor listen-only and barge transitions are strictly enforced, local AMI operations and security boundaries are confirmed, VICIdial FastAGI starts and closes calls automatically in MariaDB under strict SQL mode, and conference recordings are verified for audio content and metadata correlation.

### Accepted Limitations:
1. **Agent UI Integration:** Testing used synthetic browser contexts and SIP.js; full VICIphone web interface integration, campaign autodialing, and agent UI recording initiation remain downstream in ALM-7 and M1+.
2. **Physical Hardware:** Real acoustic microphone/speaker tests were not conducted; deterministic fake-audio injection was verified through actual browser WebRTC capture tracks.
3. **Boot Persistence:** Laboratory services require explicit systemd service startup; full automated OS boot sequence persistence is a separate operational milestone.
4. **Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) remains in `status: candidate`.

### Next Assignment Recommendation (ALM-7):
- **Task:** **ALM-7 / M0-07** (`rwpendvaoqpi9mrzgskej6qb`) — *Validate self-hosted VICIphone and browser end-to-end media*.
- **Role:** Principal WebRTC Front-End & Telephony Integration Engineer.
- **Scope:** Self-host the pinned VICIphone web phone application, integrate with VICIdial agent web interface over HTTPS, establish real agent login and phone session attachment, verify bi-directional audio in live conferences, test agent transfer/hangup controls, and validate WebRTC media stability.
- **Precondition:** ALM-6 acceptance unblocks ALM-7 (native dependency `rapwb42p3l0v6r8t9ym7dkz5`).
- **ALM-7 was not started during this assignment.**
