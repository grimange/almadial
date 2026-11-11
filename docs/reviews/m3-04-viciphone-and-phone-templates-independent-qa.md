# AlmaDial — ALM-26 / M3-04: Independent VICIphone and PJSIP Registration QA Report

## Acceptance Status and Governance

- **Task ID:** `wcip8y6y5jezllppbokixauc`
- **Canonical Work Package:** ALM-26 / M3-04
- **Task Title:** Integrate pinned VICIphone and generated phone templates
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Starting Git Baseline:** `00c8e1b24f39d58953f20ab3737a421fff98d816`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (`10.160.90.233`, Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `almadial-alm16-rpm.alm25-pre-firewall-20261111` (preserved)
- **Developer Evidence Comment:** `qwpn4mjxq8o7q0d44xbo9n5d`
- **Developer Verdict:**
  ```text
  ALMADIAL_M3_04_VICIPHONE_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_04_VICIPHONE_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S10 implements self-hosted VICIphone asset deployment, deterministic VICIdial/Asterisk PJSIP phone provisioning (`alm_s10_9061`), protected credential storage, and WebRTC signaling integration over trusted WSS. Independent QA was conducted across Go source code, focused Python and Go test suites, live laboratory VM evaluation on `almadial-alm16-rpm`, real browser execution using Chromium 157 driven through Playwright MCP, independent Asterisk PJSIP AOR/contact verification, recovery reconciliation, and comprehensive S01–S09 baseline preservation.

All canonical acceptance criteria for ALM-26 are satisfied:
1. **Pinned VICIphone & SIP.js Assets:** Deployed assets at `/var/www/html/almadial-phone` correspond bit-for-bit with pinned VICIphone revision `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` (source archive SHA-256 `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8`) and SIP.js version `0.20.1` (SHA-256 `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b`). Upstream AGPL notices and bundled licenses are fully retained.
2. **Self-Hosted Resources:** 47 deployed files under `/var/www/html/almadial-phone` require zero external CDN or external cloud dependencies.
3. **Deterministic Provisioning & Settings Container:** One dedicated synthetic phone `alm_s10_9061` and settings container `ALMADIAL_S10` are provisioned in VICIdial with persistent protected random credential (root-only `0600` `/etc/almadial/secrets/phone-s10`). Generated PJSIP endpoint, auth (`almadial-s10-auth`), transport (`almadial-s10-wss`), and AOR (`alm_s10_9061`) in `/etc/asterisk/almadial-s10-phone.conf` (`0640 root:asterisk`) match VICIdial identities exactly.
4. **Baseline Phone Preservation:** Pre-existing phones `callin` and `gs102` are preserved across all 107 database columns with zero credential drift or schema mutations.
5. **Trusted Browser Registration:** Real Chromium browser accessed `https://localhost:8443/almadial-phone/` and connected to `wss://localhost:8089/ws` through authorized local forwarding with full certificate validation enabled against the S08 private lab CA (zero TLS validation bypass).
6. **Authentication & Contact Verification:**
   - Negative authentication with invalid credentials resulted in `Unregistered` status and zero Asterisk contacts.
   - Positive authentication with valid synthetic credentials resulted in successful SIP REGISTER challenge/response (`SIP/2.0 200 OK`), browser UI `Registered`, and exactly one matching Asterisk PJSIP contact: `alm_s10_9061/sip:rgc3hus8@127.0.0.1:34386;transport=ws;x-ast-orig-host=192.0.2.58:0`.
   - Explicit unregister removed the Asterisk contact (`0` contacts). Reconnect via visible Register button re-established `Registered` status with exactly one contact.
   - Sustained registration evidence across 610 seconds from developer qualification is verified.
7. **Installer Recovery & Clean NO_OP:** S10 plan emits empty action graph (`[]`) on a satisfied host with `no_op: true` and zero runner traces. Copied-journal reconciliation succeeds without replaying phone provisioning or Asterisk module reloads.
8. **S01–S09 Baseline Preservation:** 731 RPMs, MariaDB schema version 1748, Asterisk 22.12.0, VICIdial r4037, Apache/PHP, S08 certificate generation (`30ee4d41...`), S09 firewalld policy (`almadial-s09`), FastAGI parent and five workers, disabled updater/listener, zero active cron jobs, and SELinux Enforcing.
9. **Signaling vs Media Boundary:** Working SIP over WSS registration is proven. Media, RTP/RTCP, ICE audio streaming, two-way agent audio, and live conference calling remain strictly deferred to ALM-27 (M3-05).

---

## 2. Independent Acceptance Matrix

| Criterion | Result | Evidence |
| :--- | :---: | :--- |
| **Pinned VICIphone identity** | **PASS** | Deployed revision `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`, SIP.js `0.20.1` (`9eb6524a...`), upstream LICENSE (`e7ae8b5d...`); [phone-pjsip-inspection.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/phone-pjsip-inspection.json). |
| **Self-hosted assets** | **PASS** | 47 files under `/var/www/html/almadial-phone`; zero external CDN requests; HTTP 200 on all local scripts/styles; [browser-registration-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/browser-registration-independent-qa.json). |
| **Generated phone/PJSIP objects** | **PASS** | MariaDB phone `alm_s10_9061`, container `ALMADIAL_S10`; Asterisk endpoint `alm_s10_9061`, auth `almadial-s10-auth`, AOR `alm_s10_9061`, transport `almadial-s10-wss`; [phone-pjsip-inspection.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/phone-pjsip-inspection.json). |
| **Existing phones preserved** | **PASS** | `callin` and `gs102` all 107 columns intact; total phone count in `asterisk.phones` is exactly 3; credentials unchanged; [baseline-preservation-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/baseline-preservation-qa.json). |
| **Trusted browser HTTPS/WSS** | **PASS** | Real Chromium 157 via Playwright MCP; S08 lab CA in NSS database; strict TLS validation enabled (`ignoreHTTPSErrors: false`); loopback forwarding to 8443 (Apache) and 8089 (Asterisk WSS); [browser-registration-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/browser-registration-independent-qa.json). |
| **SIP authentication/registration** | **PASS** | Invalid credentials -> UI `Unregistered`, 0 Asterisk contacts. Valid credentials -> authenticated `REGISTER` -> `SIP/2.0 200 OK`, UI `Registered`, 1 Asterisk contact; [browser-registration-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/browser-registration-independent-qa.json). |
| **Asterisk contact verification** | **PASS** | `pjsip show aor alm_s10_9061` reports contact `alm_s10_9061/sip:rgc3hus8@...`; endpoint state changes to `Not in use`; [browser-registration-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/browser-registration-independent-qa.json). |
| **Unregister/reconnect** | **PASS** | Click `#unregister` -> UI `Unregistered`, 0 Asterisk contacts. Click `#register` -> UI `Registered`, exactly 1 Asterisk contact restored; clean final unregister verified; [browser-registration-independent-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/browser-registration-independent-qa.json). |
| **Registration refresh evidence** | **PASS** | Sustained registration verified across 610-second window in developer ledger (`browser-registration-renewed.json`); contact count remained 1 with subsequent re-REGISTER; [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/developer-evidence-verification.json). |
| **S10 recovery** | **PASS** | Copied interrupted journal fixture reconciles `phone.00.activate` to `succeeded` with `no_op: true` and `traces: []`; canonical journal untouched; [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/developer-evidence-verification.json). |
| **Fresh NO_OP** | **PASS** | `almadial-phone plan` returns empty action graph `[]`; `almadial-phone lab-run` reports `no_op: true` with zero runner traces; [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/developer-evidence-verification.json). |
| **S01–S09 preservation** | **PASS** | 731 RPMs, MariaDB schema 1748 (345 tables), Asterisk 22.12.0, VICIdial r4037, FastAGI (parent + 5 workers), S08 cert `30ee4d41...`, S09 firewalld active, SELinux Enforcing; [baseline-preservation-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/baseline-preservation-qa.json). |
| **Regression** | **PASS** | Go 1.27.3 vet/fmt/test/race passed (all packages pass, 0 races); Python passed (210 tests, including `test_s10_phone.py`); 17 CLI smoke tests; 11 terminal journeys; 11 Linux amd64 reproducible builds; `git diff --check` passed; [regression-summary-qa.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/regression-summary-qa.json). |
| **Evidence integrity** | **PASS** | Developer primary ledger (139 files, SHA-256 `545dad9dc347e3b183c695bd2a7ad24006fab060139146c1a088d824827d1cbb`) and handoff ledger (SHA-256 `9c2f0b6f86630c77fca396da8f53e5d1f69976ae8f5a075bc06d9181aec0730c`) verified 100% OK; independent QA ledger sealed (`c371c31e...`); [developer-evidence-verification.json](file:///home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/developer-evidence-verification.json). |

---

## 3. Independent Browser Registration Journey

### 3.1 Network & Forwarding Configuration
- Forwarding: Host loopback `127.0.0.1:8443` -> VM loopback `127.0.0.1:443` (Apache HTTPS).
- Forwarding: Host loopback `127.0.0.1:8089` -> VM loopback `127.0.0.1:8089` (Asterisk WSS).
- S09 Firewall: Remains Enforcing with `DROP` policy on external interfaces; no public port opening.
- Certificate Authority: S08 private lab CA installed into user NSS database (`$HOME/.pki/nssdb`); normal certificate validation strictly enforced.

### 3.2 Negative Authentication Test
- Action: Open `https://localhost:8443/almadial-phone/`, submit phone `alm_s10_9061` with invalid password `InvalidSyntheticFixture`, click `Connect phone`.
- Browser UI Observation: Status remains `Unregistered`.
- Asterisk PJSIP AOR Observation:
  ```text
  pjsip show aor alm_s10_9061 -> Contact: (empty)
  pjsip show contacts -> No objects found.
  ```
- Verdict: PASS.

### 3.3 Positive Authentication & Registration Test
- Action: Open `https://localhost:8443/almadial-phone/`, submit valid synthetic secret from `/etc/almadial/secrets/phone-s10`, click `Connect phone`.
- Browser UI Observation: Status transitioned to `Registered`.
- Asterisk PJSIP AOR Observation:
  ```text
        Aor:  alm_s10_9061                                         1
      Contact:  alm_s10_9061/sip:rgc3hus8@127.0.0.1:34386;tr 550c27eb1d NonQual nan
  ```
- Endpoint Status: `Endpoint: alm_s10_9061 Not in use 0 of inf`.
- Verdict: PASS.

### 3.4 Unregister & Reconnect Cycle
- Action 1: Click `#unregister` button.
  - Browser UI: Status changed to `Unregistered`.
  - Asterisk CLI: Contact removed immediately (`Contact: (empty)`, `0 contacts`).
- Action 2: Click `#register` button.
  - Browser UI: Status restored to `Registered`.
  - Asterisk CLI: Contact restored (`1 contact`).
- Action 3: Final unregister and browser session close.
  - Status clean, zero lingering contacts on Asterisk.
- Verdict: PASS.

---

## 4. Signaling vs Media Acceptance Boundary

This verification is strictly bounded to WebRTC **SIP signaling over WSS** and phone configuration templates.
- **Accepted in ALM-26:**
  - Pinned VICIphone deployment and notices.
  - VICIdial phone template and settings container creation.
  - Asterisk PJSIP endpoint, AOR, auth, and WSS transport configuration.
  - Browser TLS/WSS connectivity and SIP REGISTER negotiation.
  - Asterisk registered contact tracking and unregister cleanup.
- **Explicitly Deferred to ALM-27 (M3-05):**
  - Full WebRTC media path (RTP/RTCP audio packets).
  - ICE candidate gathering and STUN/TURN traversal.
  - DTLS-SRTP key exchange and bidirectional audio verification.
  - VICIdial ConfBridge audio meetme conferencing.
  - Call recording creation and playback verification.
  - Live agent campaign login and dialing workflows.

---

## 5. Evidence Artifacts

Independent QA evidence directory:
`/home/ra/almadial-lab/evidence/m3-04/qa-20261112T002025Z-alm26/`

SHA-256 Ledger (`SHA256SUMS` SHA-256: `c371c31e0b1e9a63ec1e9145ea04ca0d1f20ffe9f3375075b13a87162e310e47`):
- `browser-registration-independent-qa.json`: `a4f8992adca535352c7c59806e23bb4628f4bcbb20cf014b22c7eb163ff4f5ad`
- `phone-pjsip-inspection.json`: `d475cf260fc5fcf86c221ba89f2d5baaebe37018ae874d12eb0f17b3a9ce0204`
- `baseline-preservation-qa.json`: `6ba433434aa8f9441113b2e5cc3b060ea0b04c64366ebce9cfc89283f585cae9`
- `regression-summary-qa.json`: `ee44b8ee52bb07d722d5a3ef2c116c905545db2ae4bc9ef75df42b08a38b1f75`
- `developer-evidence-verification.json`: `60f0896eb9c5a7749eb72449c235cb511993930b2c1f96e4e5bc531d0fc7a20c`

---

## 6. Final Recommendation

All canonical acceptance criteria for ALM-26 (M3-04) are satisfied. No defects were encountered during independent QA.
Final verdict: **`ALMADIAL_M3_04_VICIPHONE_ACCEPTED`**.
Proceed to Git acceptance commit and Kaneo status transition to **Accepted** (`done`), preserving Codex ownership.
