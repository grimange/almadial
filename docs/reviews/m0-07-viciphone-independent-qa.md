# AlmaDial Independent QA Acceptance — ALM-7 / M0-07 Validate Self-Hosted VICIphone and Browser End-to-End Media

**Reviewer / Session:** Google Antigravity (AGY), separate independent reviewer session  
**Reviewer Professional Role:** Principal VICIdial/VICIphone Integration QA Engineer, WebRTC Real-Time Media Verification Architect, and Telephony Application Security Specialist  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)  
**Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Target Task:** ALM-7 / M0-07 (`rwpendvaoqpi9mrzgskej6qb`), Kaneo Project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Task Title:** M0-07 — Validate self-hosted VICIphone and browser end-to-end media  
**Repository Starting HEAD:** `6d23453452f00b96ae152f0e15a9a32de075c96f` (accepted ALM-6 baseline: `c3e5593921c55e65667418cd446d2d75ba541083`)  
**Target VM:** `almadial-alma10` (Multipass 1.17.0 / QEMU, AlmaLinux 10.2 x86_64, IP `10.160.90.197`)  
**Review Date / Time:** 2026-11-06 (Session UTC: 2026-11-06T18:55:00Z / Local: 2026-11-07T02:55:00+08:00)  
**Historical Implementation Evidence Roots:**
- Original: `/home/ra/almadial-lab/evidence/m0-07/20261106T072425Z/` (`evidence-ledger.json` SHA-256 `7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`, 255 entries)
- Continuation: `/home/ra/almadial-lab/evidence/m0-07/continuation-20261106T150016Z/` (`evidence-ledger.json` SHA-256 `6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c`, 426 entries)
- Scope Approval: `/home/ra/almadial-lab/evidence/m0-07/scope-approved-20261106T175823Z/` (`evidence-ledger.json` SHA-256 `7b348f5229d7a53c0d0bb48208fb9b5a97c1ef5f787e1de2f9d6189b15c5da8a`, 12 entries)
**Dedicated QA Evidence Path:** `/home/ra/almadial-lab/evidence/m0-07/qa-20261106T185500Z/` (`qa-evidence-ledger.json` SHA-256 `416160a67a07a4ac141e81b50c2bdbdd9da10144dbb5c6b43c670ee0a11b352d`, 7 sealed artifacts)  
**Historical Implementation Report:** Preserved in [`docs/compatibility/m0-07-viciphone-browser-runtime.md`](../compatibility/m0-07-viciphone-browser-runtime.md)  
**Approved Scope Decision Record:** Preserved in [`docs/planning/alm-7-network-scope-proposal.md`](../planning/alm-7-network-scope-proposal.md)  
**Final Independent QA Verdict:** **`ALMADIAL_M0_07_ACCEPTED`**

---

## 1. Scope, Governance, and Reviewer Independence Confirmation

This review constitutes the formal independent quality assurance evaluation of task **ALM-7 / M0-07: Validate self-hosted VICIphone and browser end-to-end media**.

In accordance with repository governance ([`AGENTS.md`](../../AGENTS.md), [`GEMINI.md`](../../GEMINI.md), [`.agents/skills/almadial-review/SKILL.md`](../../.agents/skills/almadial-review/SKILL.md), [`.agents/rules/almadial-kaneo-governance.md`](../../.agents/rules/almadial-kaneo-governance.md), and [`.agents/rules/almadial-evidence-integrity.md`](../../.agents/rules/almadial-evidence-integrity.md)):
- **Separation of Roles:** Codex performed the self-hosted VICIphone integration, source derivations, guest overlay deployment, browser automation, call control and media tests, submitting ALM-7 to `QA Review` (`in-review`) without claiming self-acceptance.
- **Reviewer Authority:** Google Antigravity (AGY) operates as the distinct reviewer under authorized Kaneo identity `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`. AGY does not assume ownership of ALM-7; Codex remains the permanent implementation owner (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`).
- **Scope Boundary and Approval Provenance:**
  - The scope amendment establishing local/LAN compatibility acceptance was independently verified through Kaneo read-back.
  - Operator approval was recorded on 2026-11-06 UTC in Kaneo comment `nuggyhfh2z7ysi1jc3bvls89`. The operator replied verbatim **“yes I approve”** to the exact query: *"Do you approve the ALM-7-only amendment in the proposal, including the ALM-27 external-qualification addendum, while leaving ALM-8 and the aggregate M0 exit criteria unchanged?"*
  - The reviewed proposal SHA-256 (`424370a6868c39cf086f1650ec000b6fc2ba73ceac8c7a82393dfa44968e843d`) and the complete original ALM-7 description before amendment were verified preserved.
  - Downstream obligations are verified in Kaneo:
    - **Direct-public**, **1:1 NAT**, and **remote-home-agent** profiles remain **NOT_RUN** and are deferred to mandatory external qualification under **ALM-27 / M3-05** (`ea1axf0373fggib2v5u7ciyv`), which retains the exact approved addendum.
    - Release repetition remains mandatory under **ALM-29 / M4-01** (`xu7wxvch94ii2iw38hpb8d58`).
    - Aggregate M0 exit criteria (§§9.3 and 14) and ALM-8 complete-baseline repetition remain unchanged. No external profile was marked PASS.

---

## 2. Working Tree Baseline and Delivery Audit

- **Starting HEAD:** `6d23453452f00b96ae152f0e15a9a32de075c96f` (`docs(portal): record verified dual-SIP authorization and revocation`).
- **Accepted ALM-6 Baseline:** Commit `c3e5593921c55e65667418cd446d2d75ba541083` (`test(m0): independently accept VICIdial telephony runtime`).
- **Intervening Operator Commits:** Two operator portal commits (`47efc2f`, `6d23453`) alter only `operations/portal443/` and do not modify AlmaDial installer, core documentation, or candidate stack contracts.
- **Reviewed ALM-7 Deliverables:**
  - [`docs/compatibility/m0-07-viciphone-browser-runtime.md`](../compatibility/m0-07-viciphone-browser-runtime.md)
  - [`docs/planning/alm-7-network-scope-proposal.md`](../planning/alm-7-network-scope-proposal.md)
  - [`scripts/lab/m0_07_assets.py`](../../scripts/lab/m0_07_assets.py)
  - [`scripts/lab/m0_07_browser.js`](../../scripts/lab/m0_07_browser.js)
  - [`scripts/tests/test_m0_07_assets.py`](../../scripts/tests/test_m0_07_assets.py)
  - [`docs/AlmaDial_Implementation_Plan.md`](../AlmaDial_Implementation_Plan.md) (§10.3 and §14 M0-07 cells)
  - [`docs/implementation-plan.md`](../implementation-plan.md) (identical §10.3 and §14 M0-07 cells)
- **Preserved Untracked Work:** Pre-existing untracked helper [`scripts/verify_agy_overlay.py`](../../scripts/verify_agy_overlay.py) and follow-up document `docs/reviews/m0-05-acceptance-readback-followup.md` are preserved without modification.
- **Candidate Stack Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) retains `status: candidate`.

---

## 3. Cryptographic Audit of Protected Evidence

All three historical evidence ledgers were audited independently:
1. **Original Implementation Evidence** (`/home/ra/almadial-lab/evidence/m0-07/20261106T072425Z/`):
   - Ledger SHA-256: `7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`.
   - Entry Count: Exactly 255 files.
   - Outcome: 100% byte length and SHA-256 checksum match. 0 failures.
2. **Continuation Implementation Evidence** (`/home/ra/almadial-lab/evidence/m0-07/continuation-20261106T150016Z/`):
   - Ledger SHA-256: `6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c`.
   - Entry Count: Exactly 426 files.
   - Outcome: 100% byte length and SHA-256 checksum match. 0 failures.
3. **Approved Scope Evidence** (`/home/ra/almadial-lab/evidence/m0-07/scope-approved-20261106T175823Z/`):
   - Ledger SHA-256: `7b348f5229d7a53c0d0bb48208fb9b5a97c1ef5f787e1de2f9d6189b15c5da8a`.
   - Entry Count: Exactly 12 sealed artifacts.
   - Outcome: 100% byte length and SHA-256 checksum match. 0 failures.
4. **Dedicated QA Evidence Path** (`/home/ra/almadial-lab/evidence/m0-07/qa-20261106T185500Z/`):
   - Created with permissions `0700` (files `0600`).
   - Sealed with `qa-evidence-ledger.json` (SHA-256 `416160a67a07a4ac141e81b50c2bdbdd9da10144dbb5c6b43c670ee0a11b352d`, 7 artifacts).

---

## 4. Self-Hosted VICIphone Source and Asset Integrity

1. **Source Provenance:**
   - Upstream VICIphone Version: 3.0
   - Git Commit: `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`
   - Archive SHA-256: `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8`
   - Embedded SIP.js Version: 0.20.1
2. **Deployed Guest Asset Audit:**
   - Verified on `almadial-alma10` against `asset-manifest.json` (`6510b421898c9b0c739e76167c7acbba4edfd245328702b581737221088f6325`).
   - Exactly 59 deployed files audited: 46 VICIphone assets under `/var/www/html/m007/viciphone/`, 6 VICIdial application files under `/var/www/html/m007/agc/`, and 7 Perl/AGI runtime files under `/opt/almadial-m007/`.
   - Result: 100% hash match across all 59 files. Zero discrepancies.
3. **No External CDN Dependencies:**
   - All JavaScript, CSS, images, and audio assets are self-hosted under `/var/www/html/m007/viciphone/`.
   - Zero external runtime script or style requests.
4. **Credential Handling & Security:**
   - Hash-bound overlay replaces credential-bearing iframe URL GET parameters (`$WebPhonEurl`) with secure programmatic POST form submission into `srcdoc`.
   - Internal chat and agent panel navigations are patched to POST.
   - Debug logging disabled in `viciphone.php` (`$debug_access_log = false`) and `js/vici_phone.js` (`logLevel: "error"`, `traceSip: false`).
   - Apache access log format (`m007-access.log`) records only method, path, and status (`%h %t %m %U %>s`); query strings and POST bodies are omitted.
   - Credential audit (`credential-audit-controls.json`): 1,546 network requests, 2 Referer headers, 22 console events, 14 browser history URLs, and 161,642 bytes of Apache access logs inspected with zero credential leaks.

---

## 5. Agent Lifecycle, Heartbeat, and SQL/Calendar Compatibility

1. **Agent Web Authentication & PJSIP Registration:**
   - Real VICIdial agent login (`m007qa`) via HTTPS (`https://almadial-alma10.test:8443/agc/vicidial.php`).
   - Embedded VICIphone loads within the authenticated agent iframe.
   - Endpoint 9051 registers securely to Asterisk via WSS (`wss://almadial-alma10.test:8089/ws`).
   - Phone contact created with contact URI and digest challenge verification.
2. **Heartbeat and AST_update Integration:**
   - Investigated upstream `AST_update.pl` missing state as root cause of conference/time-sync warnings.
   - OS NTP is healthy (`NTPSynchronized=yes`).
   - Bounded derivation updates Asterisk 22 format 4 channel parsing, AMI framing, and PJSIP channel lookups.
   - Updates `server_updater` table with current timestamps (`last_update`, `db_time`), suppressing false `TIME_SYNC` and `SysteMDisablEBoX` agent warnings.
   - Clean bounded startup and shutdown verified.
3. **MariaDB Strict SQL Mode Compatibility:**
   - Strict SQL mode is active globally and per session: `STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.
   - Global strict mode was never relaxed; previous connection-wide workaround was removed.
   - Pinned operation corrections verified:
     - Error 1366 (empty string in integer `lead_id` on login): corrected to integer `0`.
     - Error 1366 (empty string in `vicidial_manager` AUTO_INCREMENT primary key): corrected to `NULL`.
     - Error 1406 (caller ID name exceeding varchar(20)): corrected dot-trimming bug with `substr($user_abb, -4)`.
     - Error 1406 (`MANUAL_DIALNOW` into varchar(6)): translated to `MANUAL`.
     - Error 1292 (empty string in date of birth): corrected to `NULLIF(escaped_value, '')`.
   - Live reproduction via `strict-sql-probe.py` on `almadial-alma10`: verified that uncorrected statements fail with 1366/1406/1292, and corrected statements succeed under strict SQL mode across PHP and Perl DBI.
4. **Calendar Initialization Correction:**
   - Pinned guard added to `f_tcal_hideAll` in `/var/www/html/m007/agc/calendar_db.js:306`.
   - Returns safely when `window.A_TCALSIDX` is not yet instantiated.
   - Prevents `TypeError: Cannot read properties of undefined (reading 'length')` during agent page scrolling.
   - Verified via unit test `test_calendar_scroll_without_instances_and_existing_instances`. Zero browser exceptions in final runs.

---

## 6. Logout Authorization Revocation Audit

1. **Vulnerability Analysis:**
   - Original VICIdial code in three protected handlers (`vdc_db_query.php`, `conf_exten_check.php`, `manager_send.php`) verified only `SELECT count(*) from web_client_sessions where session_name='$session_name' and server_ip='$server_ip'`.
   - Because `web_client_sessions` retains historical session records after logout or abandoned tabs, a stale session identifier could previously authorize subsequent requests.
2. **Corrected Authorization Query:**
   ```sql
   SELECT count(*) FROM web_client_sessions w
   JOIN vicidial_session_data s ON s.session_name=w.session_name AND s.server_ip=w.server_ip
   JOIN vicidial_live_agents a ON a.user=s.user AND a.server_ip=s.server_ip AND a.conf_exten=s.conf_exten
   WHERE w.session_name='$session_name' AND w.server_ip='$server_ip' AND s.user='$user';
   ```
3. **Live Independent Verification:**
   - Tested against the 3 retained historical sessions in `web_client_sessions`.
   - Unpatched query returned `1` (authorizing stale session).
   - Patched query returned `0` (rejection).
   - Live HTTP requests with valid user credentials and stale session name `1793950961_905116334922` returned `Invalid session_name: |1793950961_905116334922|127.0.0.1|` across all three handlers:
     - `vdc_db_query.php`: REJECTED
     - `conf_exten_check.php`: REJECTED
     - `manager_send.php`: REJECTED
   - Confirmed: stale session identifiers cannot authorize protected operations after logout.

---

## 7. Bidirectional VICIphone WebRTC Media Verification

1. **Topology & Conference Configuration:**
   - Endpoint 9051: Authenticated agent embedded VICIphone in ConfBridge 9600000.
   - Endpoint 9052: Customer VICIphone dialed via `99052` and connected to ConfBridge 9600000.
   - ConfBridge: 9600000 (`vici_agent_bridge`, `vici_audio_user`).
   - Transport: WSS (`transport-wss`, port 8089) with TLS 1.3.
   - Codec: PCMU (8000 Hz, 1 channel).
   - Media Security: DTLS-SRTP (`SRTP_AES128_CM_HMAC_SHA1_80`), ICE state `connected`.
2. **Independent Signal Re-Analysis:**
   - Raw AudioWorklet PCM WAV captures analyzed with [`scripts/lab/m0_06_media.py`](../../scripts/lab/m0_06_media.py):
   - **Direction 9051 → 9052:**
     - Pattern agreement: `0.992095` (threshold: ≥0.90) — **PASS**
     - Envelope correlation: `0.998234` (threshold: ≥0.85) — **PASS**
     - Tone agreement: `0.997821` (threshold: ≥0.90) — **PASS**
     - Estimated delay: `60 ms` (max: 2000 ms) — **PASS**
     - Remote RMS: `0.148849` (threshold: ≥0.02) — **PASS**
     - 602 frames evaluated over 12.0 seconds.
   - **Direction 9052 → 9051:**
     - Pattern agreement: `0.992095` (threshold: ≥0.90) — **PASS**
     - Envelope correlation: `0.984232` (threshold: ≥0.85) — **PASS**
     - Tone agreement: `1.000000` (threshold: ≥0.90) — **PASS**
     - Estimated delay: `40 ms` (max: 2000 ms) — **PASS**
     - Remote RMS: `0.152799` (threshold: ≥0.02) — **PASS**
     - 602 frames evaluated over 12.0 seconds.
3. **Attribution & Loopback Check:**
   - Distinct synthetic DTMF tone patterns confirmed in the opposite participant's remote track.
   - Local loopback false-positive check verified.

---

## 8. Application Call Controls Verification

1. **Park / Resume:**
   - Customer channel in 9600000 redirected to extension 8301 running upstream `park_CID.agi`.
   - Asterisk channel enters parked state; `parked_channels` row created; UI reflects `customerparked=1`.
   - Agent remains active in ConfBridge 9600000.
   - Resume redirects customer back into ConfBridge 9600000; `parked_channels` row removed.
   - Zero orphaned parked channels.
2. **Blind Transfer:**
   - Dial Blind Transfer to local conference 9600001 submitted through the agent UI.
   - Customer channel redirected to ConfBridge 9600001.
   - Agent remains in ConfBridge 9600000 with status `PAUSED`.
   - `vicidial_log` records call disposition `XFER`.
   - Empty destination validation rejected by client-side alert.
3. **Hangup & DTMF:**
   - UI Hangup terminates the channel cleanly, dispatches conference cleanup, and closes call logs.
   - SIP INFO DTMF reception baseline retained; Asterisk `res_pjsip_dtmf_info` processes events.
   - Phone-level SIP REFER and SIP hold confirmed **NOT_APPLICABLE** in pinned VICIphone 3.0 UI (mute is distinct).

---

## 9. Unexpected WSS Disconnection and Supported Recovery

1. **Signaling Loss Induction:**
   - Established server-side WSS socket interrupted abruptly using `ss -K` without SIP logout or TLS alteration.
   - Evaluated both idle and during active local conference call.
2. **Observed Behavior:**
   - UI immediately reflects **WS Disconnected**; transport state transitions to `Disconnected`.
   - Active RTP media continues in ConfBridge across the media socket during signaling disruption.
   - Asterisk removes stale PJSIP contact.
3. **Manual Recovery Contract:**
   - Clicking **Register** invokes patched `vici_phone.js` logic calling `myUserAgent.reconnect()`, followed by `myRegisterer.register()`.
   - Transport reconnects to `Connected`; registration succeeds (`Registered`); status reflects call state.
   - Explicit old-call Hangup terminates old dialog (`ice: closed`).
   - New call established to ConfBridge 9600001 (`call: Established`, `ice: connected`).
   - PJSIP contact reconciled: exactly 1 contact per endpoint, 0 duplicate contacts.
   - Transparent continuous same-call recovery is correctly documented as **NOT_APPLICABLE**; the manual recovery contract is satisfied.

---

## 10. Security Boundaries, Baseline Preservation, and Cleanup

1. **Security Posture:**
   - SELinux: **Enforcing** (confirmed live).
   - MariaDB: Socket-only via `/var/lib/mysql/mysql.sock`; `skip_networking = ON`; TCP port 3306 has 0 listeners.
   - Asterisk AMI: Listening strictly on `127.0.0.1:5038`.
   - FastAGI: Listening strictly on `127.0.0.1:4577`.
   - Apache HTTPS: Bound to port 8443 with TLS 1.3 / TLS 1.2, `Referrer-Policy: no-referrer`, and `Cache-Control: no-store`.
   - Asterisk WSS: Bound to port 8089 with trusted TLS.
2. **Negative TLS Validation:**
   - Untrusted CA rejected: `curl` fails with exit code 60 (`unable to get local issuer certificate`).
   - Hostname mismatch rejected: connecting to `wrong-almadial.test` fails with exit code 60 (`no alternative certificate subject name matches target hostname`).
   - Playwright rejects invalid CA with `net::ERR_CERT_AUTHORITY_INVALID` and mismatched hostname with `net::ERR_CERT_COMMON_NAME_INVALID`. No insecure overrides.
3. **Baseline Preservation:**
   - Native endpoint 9001 and echo dialplan `9001@alm3-echo` (`Echo()`) intact.
   - Generated endpoints 9051, 9052, 9053 intact.
   - Asterisk binary and 17 Asterisk `.conf` hashes match accepted baseline.
4. **Final System Read-Back:**
   - Active calls: **0**
   - Active channels: **0**
   - Active conferences: **0**
   - PJSIP contacts: **0**
   - Live agents: **0**
   - Occupied confbridges: **0**
   - Parked channels: **0**
   - Pending manager queue items: **0**
   - Retained web client session rows: 3 historical records (all 3 reject protected requests).

---

## 11. Automated Repository Validation Suite

The independent reviewer executed the full repository test suite:

1. `python3 scripts/validate_scaffold.py`: **`PASS`** (11 required files, 4 skills).
2. `python3 -m unittest discover -s scripts/tests -v`: **`118/118 unit tests PASSED`** in 1.825s (including all 22 ALM-7 tests).
3. `python3 scripts/verify_candidate_stack.py`: **`PASS`** in LOCAL mode (`candidate_consistency: PASS`).
4. `python3 scripts/verify_config_generation.py`: **`PASS`** (`REDACTED_FIXTURE_REPLAY`).
5. `python3 -m py_compile scripts/lab/m0_07_assets.py scripts/tests/test_m0_07_assets.py`: **`PASS`** (clean syntax).
6. `node --check scripts/lab/m0_07_browser.js`: **`PASS`** (clean syntax).
7. `git diff --check`: **`PASS`** (zero trailing whitespace or formatting defects).

---

## 12. Independent QA Compliance Matrix

| Requirement / Component | Verification Method | Observed Result | Classification |
|---|---|---|---|
| Scope Amendment Approval Provenance | Kaneo comment `nuggyhfh2z7ysi1jc3bvls89` read-back | Verbatim "yes I approve", proposal SHA-256 `424370a6...` | **PASS** |
| Downstream External Gate Preservation | Kaneo ALM-27 & ALM-29 task read-back | Retained as Backlog with mandatory external addendum | **PASS** |
| Protected Evidence Integrity | Re-hash 3 historical ledgers (255, 426, 12 files) | 100% byte and hash match | **PASS** |
| Self-Hosted VICIphone 3.0 / SIP.js 0.20.1 | Asset audit against `asset-manifest.json` | 59/59 files verified on guest, no CDN requests | **PASS** |
| Credential Exposure Protections | Audit 1,546 requests, Referers, console, history | 0 password leaks; POST submission enforced | **PASS** |
| Real Agent Web Authentication | Playwright HTTPS agent login | Authenticated session created, embedded phone loaded | **PASS** |
| PJSIP WebRTC Registration | Asterisk CLI & SIP.js state | Endpoint 9051 registered, contact created | **PASS** |
| Upstream Heartbeat Integration | `AST_update.pl` execution & `server_updater` audit | Formats normalized, warnings suppressed | **PASS** |
| MariaDB Strict SQL Compatibility | Live probe on `almadial-alma10` | 1366, 1406, 1292 corrected; strict mode active | **PASS** |
| Calendar Initialization | Unit test & agent scrolling inspection | `window.A_TCALSIDX` guard active, 0 exceptions | **PASS** |
| Logout Authorization Revocation | Live HTTP curl probes against 3 handlers | All 3 return `Invalid session_name` for stale session | **PASS** |
| Two-Way VICIphone Media (9051 ↔ 9052) | Media analysis on AudioWorklet captures | Patterns ≥0.992, correlations ≥0.984, tone ≥0.997 | **PASS** |
| VICIdial Park / Resume Controls | Live call execution & DB/AMI audit | Customer enters `park_CID.agi`, returns cleanly | **PASS** |
| VICIdial Blind Transfer Control | Live call execution & DB/AMI audit | Customer moved to 9600001, agent stays, XFER disposition | **PASS** |
| Call Teardown & Hangup | UI hangup & CLI inspection | Clean channel closure, conference released | **PASS** |
| DTMF Reception | Asterisk CLI inspection | SIP INFO baseline retained | **PASS** |
| Unexpected WSS-Loss Recovery | Socket termination via `ss -K` | WS Disconnected -> Register -> New Call PASS | **PASS** |
| Negative TLS Enforcement | Untrusted CA and hostname mismatch curl | Both rejected with exit code 60 | **PASS** |
| Local/LAN Network Profile | Multipass desktop-to-VM | Validated under approved boundary | **PASS** |
| Direct-Public Network Profile | Examination of task scope | Authorized public infra absent | **NOT_RUN (Deferred to ALM-27)** |
| 1:1 NAT Network Profile | Examination of task scope | Fixed mapping gateway absent | **NOT_RUN (Deferred to ALM-27)** |
| Remote-Home-Agent Network Profile | Examination of task scope | External home router/browser absent | **NOT_RUN (Deferred to ALM-27)** |
| Security & Resource Cleanup | Port scan, SELinux, Asterisk/MariaDB audit | Enforcing, DB socket-only, 0 channels/calls/contacts | **PASS** |

---

## 13. Final Acceptance Verdict and Downstream Recommendations

### Acceptance Verdict:
**`ALMADIAL_M0_07_ACCEPTED`**

Task ALM-7 satisfies all mandatory acceptance criteria under its authorized local/LAN acceptance boundary. Self-hosted VICIphone 3.0 and SIP.js 0.20.1 integrate natively with VICIdial r4037 and Asterisk 22.12.0 on AlmaLinux 10.2. Real agent authentication, embedded phone registration, bidirectional conference media exceeding all objective signal thresholds, upstream heartbeat processing, strict MariaDB SQL compatibility, calendar initialization, post-logout authorization revocation, application park/resume and blind transfer, and predictable manual WSS recovery are independently verified.

The external network profiles (direct-public, 1:1 NAT, and remote-home-agent) remain accurately classified as **NOT_RUN** and are deferred to mandatory downstream qualification under ALM-27 / M3-05 and ALM-29 / M4-01. Aggregate M0 exit criteria remain unchanged.

### Accepted Limitations:
1. **Local/LAN Boundary:** Testing was conducted within the local host/VM network boundary; direct-public, 1:1 NAT, and remote-home-agent traversal remain NOT_RUN pending ALM-27.
2. **Physical Acoustic Audio:** Injected synthetic microphone signals were captured and evaluated through real browser WebRTC tracks; physical acoustic transducer tests were not performed.
3. **Manual WSS Recovery:** Recovery requires user action (clicking Register, hanging up old dialog, placing new call); automatic seamless in-call reconnect is not supported or claimed.
4. **Boot Persistence:** Laboratory services require explicit service startup; automated OS boot orchestration is evaluated in M1+.
5. **Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) remains `status: candidate`.

### Next Assignment Recommendation (ALM-8):
- **Task:** **ALM-8 / M0-08** (`ftlf2grvt89s3h4nv1yz4x11`) — *Repeat the complete baseline and propose stack-lock acceptance*.
- **Role:** Principal Compatibility Baseline Engineer & System Reliability Architect.
- **Worker Configuration:** Codex (Implementation Owner), Gemini Pro-class model, High reasoning effort.
- **Required Tools:** Git, Bash, Multipass CLI, Asterisk 22, MariaDB, PHP 8.3, Perl 5.40, Python 3.12+, OpenSSL.
- **Runtime Scope:** Execute a second clean-VM pass on fresh AlmaLinux 10.2 instance; repeat all M0-01 through M0-07 checks without undocumented manual intervention; generate repeatable evidence; reconcile candidate manifest; propose lock promotion.
- **Precondition:** ALM-7 acceptance unblocks ALM-8 (native dependency `sq76uz00g3qczpiwnamj1d26`).
- **ALM-8 was not started during this assignment.**
