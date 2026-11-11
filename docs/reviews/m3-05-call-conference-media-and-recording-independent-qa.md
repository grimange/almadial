# AlmaDial — ALM-27 / M3-05: Independent WebRTC Call, Conference, Media and Recording QA Report

## Acceptance Status and Governance

- **Task ID:** `ea1axf0373fggib2v5u7ciyv`
- **Canonical Work Package:** ALM-27 / M3-05
- **Task Title:** Run integrated call, conference, media and recording acceptance
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted ALM-26 Baseline Commit:** `f96be291725862334037a63bfcd31b74279edc3f`
- **Current Git HEAD:** `d2b6736692745478006effec301c348d19ca6f76` (includes 4 accepted concurrent operator commits)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (`10.160.90.233`, Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `almadial-alm16-rpm.alm25-pre-firewall-20261111` (preserved)
- **Developer Evidence Comment:** `kgs1ylvhmvbv8ixu6ui3uu6z`
- **Developer Verdict:**
  ```text
  ALMADIAL_M3_05_MEDIA_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_05_MEDIA_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S11 delivers the end-to-end integration and verification of WebRTC SIP calling, ConfBridge meetme conferencing, ICE and DTLS-SRTP encryption, bidirectional audio streaming, native Asterisk call recording, FastAGI call logging, and the complete VICIdial agent journey on native AlmaLinux 10.2.

Independent QA was conducted across Go source code, focused Python test suites, live laboratory VM evaluation on `almadial-alm16-rpm`, real browser execution using Chromium driven through Playwright MCP, independent Asterisk channel and ConfBridge state inspection, dual-party audio tone analysis, PCM recording file alignment, database call-log correlation, reboot checkpoint audit, and comprehensive S01–S10 baseline preservation.

All canonical acceptance criteria for ALM-27 are satisfied:
1. **Real Synthetic Call Execution:** Chromium browser with synthetic phone `alm_s10_9061` registered via WSS (`wss://localhost:8089/ws`), negotiated SIP INVITE, answered on channel `PJSIP/alm_s10_9061-00000011` in `Up` state, joined ConfBridge `9600271`, and executed clean BYE hangup with zero lingering channels.
2. **ConfBridge Participation & Cleanup:** Asterisk ConfBridge `9600271` correctly accepted WebRTC channels, bridged audio participants, and released all channels and bridge allocations upon call termination.
3. **ICE & DTLS-SRTP Negotiation:** Authenticated WSS connection successfully completed DTLS handshake with certificate verification, negotiated SRTP encryption keys, and established active local media transport.
4. **Bidirectional Audio Verification:** Dual-browser audio test confirmed distinct synthetic dual-frequency tones transmitted and received in both directions (A->B match `0.992095`, RMS `0.107068`; B->A match `0.992126`, RMS `0.114198`), proving real two-way audio with zero silence or clipping defects.
5. **Call Recording Capture:** Native Asterisk MixMonitor recordings (`alm-s11-1794417268.12.wav`, duration 18.98s; `alm-s11-1794417275.15.wav`, duration 12.62s) verified as valid PCM16 8000Hz mono WAV files.
6. **Recording Content Alignment:** Captured recording files matched browser transmitted tones with 100% agreement across 235 evaluated 20ms frames (`agreement: 1.0`).
7. **FastAGI Call-Log Correlation:** Asterisk `asterisk.call_log` database entries accurately recorded start times, durations (19s and 13s), channel identifiers, and caller IDs correlated with recording files.
8. **Complete Agent-28 Workflow:** Full 11-phase VICIdial agent journey (login -> conference meetme join -> manual dial -> customer answer -> blind transfer -> customer hangup/disposition -> agent hangup -> logout) verified with zero lingering channels, zero locked conferences, and clean queue release.
9. **Installer Recovery & NO_OP:** S11 planner emits clean NO_OP plan on satisfied host; copied journal reconciliation succeeds without re-execution.
10. **Reboot Persistence:** Authorized VM reboot checkpoint verified (Boot ID `6988b3a3-f8ea-4c0f-b5c0-7be4176e7be4`); all 7 telephony and web systemd services restored cleanly to active/running state.
11. **S01–S10 Baseline Preservation:** 731 RPMs, MariaDB schema 1748 across 345 tables, 3 phones in `asterisk.phones` (`callin`, `gs102`, `alm_s10_9061`), FastAGI parent and 5 workers, SELinux Enforcing, and firewalld active.
12. **Signaling vs Media Qualification:** Local/LAN WebRTC media, ConfBridge mixing, and recording are proven. Carrier PSTN routing, public STUN/TURN traversal, and web recording publication UI are strictly out of scope per the 2026-11-07 Product Scope Realignment.

---

## 2. Independent Acceptance Matrix

| # | Criterion | Result | Evidence |
| :-: | :--- | :---: | :--- |
| **1** | **Real synthetic call (INVITE / answer / hangup)** | **PASS** | Phone `alm_s10_9061` dialed `9271`, channel `PJSIP/alm_s10_9061-00000011` reached `Up` state, BYE cleanly terminated call with 0 lingering channels; [call-conference-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/call-conference-independent-qa.json). |
| **2** | **ConfBridge integration & cleanup** | **PASS** | Channel joined ConfBridge `9600271`; `confbridge list 9600271` showed active caller; hangup released caller and bridge; [single-call-result.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/single-call-result.json). |
| **3** | **ICE & DTLS-SRTP negotiation** | **PASS** | DTLS handshake succeeded with S08 lab certificates; SRTP crypto negotiated; active ICE candidate pair established; [call-conference-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/call-conference-independent-qa.json). |
| **4** | **Bidirectional audio verification** | **PASS** | Dual-tone analysis: A->B pattern match `0.992095` (RMS `0.107068`), B->A pattern match `0.992126` (RMS `0.114198`), zero silence, operator confirmed; [media-and-recording-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/media-and-recording-qa.json). |
| **5** | **Call recording capture** | **PASS** | Captured `alm-s11-1794417268.12.wav` (18.98s) and `alm-s11-1794417275.15.wav` (12.62s) in `/var/spool/asterisk/monitor/`; valid PCM16 8000Hz mono WAV; [media-and-recording-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/media-and-recording-qa.json). |
| **6** | **Recording content alignment** | **PASS** | Whole-file tone detection confirmed 100% agreement across 235 frames (`agreement: 1.0`, zero tone inversion); [media-and-recording-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/media-and-recording-qa.json). |
| **7** | **Call-log correlation** | **PASS** | Database entries in `asterisk.call_log` for unique IDs `1794417268.12` (19s) and `1794417275.15` (13s) matched recordings and channels; [media-and-recording-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/media-and-recording-qa.json). |
| **8** | **Agent-28 workflow execution** | **PASS** | 11-phase journey (login -> conf `8600051` -> manual dial `7271` -> answer -> blind transfer `8271` -> customer hangup/disposition -> agent hangup -> logout); [agent-workflow-and-cleanup-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/agent-workflow-and-cleanup-qa.json). |
| **9** | **Agent session cleanup** | **PASS** | Zero active channels (`0 active channels`), zero active calls (`0 active calls`), zero lingering conference callers, clean queue state; [agent-workflow-and-cleanup-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/agent-workflow-and-cleanup-qa.json). |
| **10** | **Installer NO_OP and recovery** | **PASS** | S11 planner outputs `no_op: true`, empty action graph `[]`, 0 runner traces; journal recovery fixture verified; [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/developer-evidence-verification.json). |
| **11** | **Reboot persistence** | **PASS** | Authorized VM reboot checkpoint confirmed (Boot ID `6988b3a3-f8ea-4c0f-b5c0-7be4176e7be4`); all 7 services active (`httpd`, `mariadb`, `asterisk`, `vicidial-fastagi`, `firewalld`, `chronyd`, `sshd`); [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/developer-evidence-verification.json). |
| **12** | **S01–S10 baseline preservation** | **PASS** | 731 RPMs, MariaDB schema version 1748 across 345 tables, 3 phones in `asterisk.phones` (`callin`, `gs102`, `alm_s10_9061`), FastAGI running, SELinux Enforcing, firewalld active; [baseline-preservation-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/baseline-preservation-qa.json). |
| **13** | **Regression suite verification** | **PASS** | Go vet/fmt/test/race passed (1,402 events, 0 race defects); Python passed (221 tests); 17 CLI smoke tests; 11 terminal journeys; 12 reproducible Go builds; git diff clean; [regression-summary-qa.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/regression-summary-qa.json). |
| **14** | **Evidence integrity** | **PASS** | Developer primary ledger (520 files, SHA-256 `1de0981c58005fd4079cb6200890ca66bf44fd314a9062bb0f166a9829a1b47d`) 100% verified; independent QA ledger sealed (10 files, SHA-256 `ca5698fd...`); [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/developer-evidence-verification.json). |

---

## 3. Independent Live Call & ConfBridge Verification

### 3.1 Network & Forwarding Topology
- HTTPS Signaling: Host `127.0.0.1:8443` -> VM `127.0.0.1:443` (Apache).
- WSS Signaling: Host `127.0.0.1:8089` -> VM `127.0.0.1:8089` (Asterisk PJSIP WSS).
- RTP Media: Host `127.0.0.1:10000-10020` -> VM `127.0.0.1:10000-10020` (UDP RTP forwarders).
- Certificate Validation: Strict TLS validation enabled using S08 lab CA in user NSS database.

### 3.2 Real Call Journey Execution
- **Registration:** Synthetic phone `alm_s10_9061` authenticated over WSS -> UI `Registered`, exactly 1 Asterisk contact registered.
- **Dialing:** Dialed extension `9271` via dialplan context `alm-s11-media`.
- **Call Establishment:** Channel `PJSIP/alm_s10_9061-00000011` transitioned to `Up` state.
- **ConfBridge Join:** Channel joined ConfBridge bridge `9600271`.
  - Asterisk CLI verification:
    ```text
    Bridge ID: 9600271
    Type: ConfBridge
    Channel: PJSIP/alm_s10_9061-00000011
    ```
- **Hangup & Release:** SIP BYE sent from client -> Asterisk received hangup -> Channel destroyed -> ConfBridge deleted -> Active channels returned to `0`.

---

## 4. Media & Recording Verification

### 4.1 Bidirectional Audio Analysis
- Party A played synthetic dual-tone pattern (440Hz / 880Hz alternating).
- Party B played synthetic dual-tone pattern (600Hz / 1200Hz alternating).
- Results:
  - Transmission A -> B: Correlation `0.992095`, RMS `0.107068`.
  - Transmission B -> A: Correlation `0.992126`, RMS `0.114198`.
  - Confirmed: Zero packet loss, zero clipping, zero half-duplex or one-way audio anomalies.

### 4.2 Recording Analysis & Call-Log Correlation
- Audio Files:
  - `alm-s11-1794417268.12.wav` (18.98 seconds, 303,724 bytes, RIFF/WAVE PCM16 8000Hz mono).
  - `alm-s11-1794417275.15.wav` (12.62 seconds, 201,964 bytes, RIFF/WAVE PCM16 8000Hz mono).
- Content Agreement:
  - Frame-by-frame tone alignment against browser transmitted audio yielded 100% tone match across 235 evaluated frames (`agreement: 1.0`).
- Database Call Log:
  - `uniqueid = 1794417268.12`: `start_time = 2026-11-11 16:47:48`, `length_in_sec = 19`, `channel = PJSIP/alm_s10_9061-0000000c`.
  - `uniqueid = 1794417275.15`: `start_time = 2026-11-11 16:47:55`, `length_in_sec = 13`, `channel = PJSIP/alm_s10_9061-0000000d`.

---

## 5. Scope Realignment and Product Boundaries

In alignment with the 2026-11-07 Product Scope Realignment:
- **Verified & Accepted in ALM-27:**
  - Single-server Asterisk/PJSIP WebRTC media over local/LAN network.
  - ConfBridge audio mixing for VICIdial meetme conferences.
  - Full bidirectional audio streaming with verified non-silence RMS and tone fidelity.
  - Native Asterisk call recording and FastAGI call-log correlation.
  - Complete VICIdial agent session workflow (login, conference, manual dial, transfer, disposition, logout) and complete channel cleanup.
  - S01–S10 baseline preservation and installer recovery.
- **Explicitly Deferred / Out of Scope (NOT_RUN):**
  - Public STUN/TURN traversal over external NAT networks.
  - External PSTN carrier trunking and live outbound carrier calling.
  - Browser recording playback UI or web publishing portals.
  - Multi-server cluster or remote agent deployments.

---

## 6. Independent QA Evidence Artifacts

Independent QA evidence directory:
`/home/ra/almadial-lab/evidence/m3-05/qa-20261112T041545Z-alm27/`

SHA-256 Ledger (`SHA256SUMS` SHA-256: `ca5698fd8524e26f53fd213b616a18e3004e94f80c2f77d135bbbf92e4d294b0`):
- `agent-workflow-and-cleanup-qa.json`: `ab3b2af73d4cabfd83cf0e573537e57ceba73efe4786026fe02a7d3dff3a34a7`
- `baseline-preservation-qa.json`: `065f4f4493c1949b5943ccdbe6712b1d7f51d744bd142ff8bfd046e36a397d41`
- `call-conference-independent-qa.json`: `55d55e0d05bb985a10b8b936a7bf06b76b966835675c80acbe87791340a7568f`
- `cleanup.json`: `8318b7e854e853acb256c6f7ad90502cbacc538ec728009698a3562995d64654`
- `developer-evidence-verification.json`: `c4c5f4527b37dd9991ad3078235e63588a9d55c4e4075c43757376b7d5ed7d2b`
- `media-and-recording-qa.json`: `aa6b0238f63261d210f03c0471c1c4dac13dadb88502b0204949fa1b943fa451`
- `prepared.json`: `b60261affc7e885545d95756414b8ab7cfccaa3e6743997e06ba34ce2bfbf23a`
- `public-lab-ca.pem`: `89b7d4963102cef60722d55221f7cd949cfc09dacbe291c0500e58602d5f3d71`
- `regression-summary-qa.json`: `0ddd494be7096deb88d4708620da231422da0aa8d12cc48137a614344bffa8af`
- `single-call-result.json`: `2ccad96372c2e7400f6f5c01451d92a11fd3452fe47d9c8d8dc6ad07e5c706c0`

---

## 7. Final Recommendation

All canonical acceptance criteria for ALM-27 (M3-05) are satisfied. All 21 developer evidence checks and independent reproduction tests passed deterministically.
Final verdict: **`ALMADIAL_M3_05_MEDIA_ACCEPTED`**.
Proceed to Git acceptance commit and Kaneo status transition to **Accepted** (`done`), strictly preserving Codex ownership.
