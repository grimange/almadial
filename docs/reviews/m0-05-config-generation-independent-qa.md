# AlmaDial Independent QA Acceptance — ALM-5 / M0-05 VICIdial PJSIP & Dialplan Configuration Generation

**Reviewer / Session:** Google Antigravity (AGY), separate independent reviewer session  
**Reviewer Professional Role:** Principal Asterisk/VICIdial Integration QA Engineer, Configuration Integrity Architect, and Linux Telephony Reliability Specialist  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)  
**Implementer:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Target Task:** ALM-5 / M0-05 (`z6mn0h4rh66ip9hh7hh769eu`), Kaneo Project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Repository Starting HEAD:** `5619db80e240a19ab5546af30c3e740128e95c23` (ALM-4 commit: `387f5e3cdd7d482f03bf060b4e84f76641681945`)  
**Host Target:** `almadial-alma10` (Multipass 1.17.0 / QEMU, AlmaLinux 10.2 x86_64, IP `10.160.90.197`)  
**Review Date / Time:** 2026-11-06 (Session: 2026-11-06T07:45:00+08:00)  
**Protected Evidence Root:** `/home/ra/almadial-lab/evidence/m0-05/20261105T222428Z/` (`qa-evidence-ledger.json` SHA-256 `dec1d009dda76337c3e025f6684a9731ac98e23123a9ff73ff535cb3e2fc5579`)  
**QA Evidence Path:** `/home/ra/almadial-lab/evidence/m0-05/qa-20261106T074000Z/`  
**Historical Implementation Reports:** Preserved in [`docs/compatibility/m0-05-config-ownership.md`](../compatibility/m0-05-config-ownership.md), [`docs/compatibility/m0-05-vicidial-config-generation.md`](../compatibility/m0-05-vicidial-config-generation.md), [`docs/architecture/config-ownership.md`](../architecture/config-ownership.md), and [`docs/compatibility/asterisk-vicidial-adapter.md`](../compatibility/asterisk-vicidial-adapter.md)  
**Final Independent QA Verdict:** **`ALMADIAL_M0_05_ACCEPTED`**

---

## 1. Scope, Authority, and Reviewer Independence Confirmation

This review constitutes the formal independent quality assurance evaluation of task **ALM-5 / M0-05: Audit VICIdial PJSIP and dialplan generation ownership**.

In accordance with repository governance ([`AGENTS.md`](../../AGENTS.md), [`GEMINI.md`](../../GEMINI.md), [`.agents/skills/almadial-review/SKILL.md`](../../.agents/skills/almadial-review/SKILL.md), [`.agents/rules/almadial-kaneo-governance.md`](../../.agents/rules/almadial-kaneo-governance.md), and [`.agents/rules/almadial-evidence-integrity.md`](../../.agents/rules/almadial-evidence-integrity.md)):
- **Separation of Roles:** Codex performed the generator adaptation, source derivation correction, configuration staging/publishing, Asterisk reloading, WebRTC echo preservation tests, and self-checks on disposable guest `almadial-alma10`, correctly submitting ALM-5 to `QA Review` (`in-review`) without claiming self-acceptance.
- **Reviewer Authority:** Google Antigravity (AGY) operates as the distinct reviewer under authorized Kaneo identity `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`. AGY does not assume ownership of ALM-5; Codex remains the permanent implementation owner (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`).
- **Scope Boundaries:** Testing is strictly confined to VICIdial r4037 configuration generation, PJSIP endpoint 9051 loading, ConfBridge profile and room generation, local-only AMI privileges, MariaDB socket isolation, Perl site-perl traversal permissions, and preservation of the accepted Asterisk 22.12.0 WebRTC echo runtime. Zero campaign autodialing, zero external carrier trunk calls, zero live agent conferences, and zero snapshot restorations were executed.

---

## 2. Source Baseline and Working Tree State

- **Input Baseline:** Commit `387f5e3cdd7d482f03bf060b4e84f76641681945` (`test(m0): independently accept VICIdial application runtime`).
- **Intermediate Operations Commits:** Two unrelated operator portal commits (`1afdd6f`, `5619db8`) were inspected; they modify only `operations/portal443/` and do not alter AlmaDial installer, core documentation, or candidate stack contracts.
- **Reviewed ALM-5 Implementation Deliverables:**
  - [`docs/architecture/config-ownership.md`](../architecture/config-ownership.md)
  - [`docs/compatibility/asterisk-vicidial-adapter.md`](../compatibility/asterisk-vicidial-adapter.md)
  - [`docs/compatibility/m0-05-config-ownership.md`](../compatibility/m0-05-config-ownership.md)
  - [`docs/compatibility/m0-05-vicidial-config-generation.md`](../compatibility/m0-05-vicidial-config-generation.md)
  - [`scripts/lab/m0_05_capture.js`](../../scripts/lab/m0_05_capture.js)
  - [`scripts/lab/m0_05_generator.py`](../../scripts/lab/m0_05_generator.py)
  - [`scripts/tests/fixtures/config_generation/r4037.json`](../../scripts/tests/fixtures/config_generation/r4037.json)
  - [`scripts/tests/test_verify_config_generation.py`](../../scripts/tests/test_verify_config_generation.py)
  - [`scripts/verify_config_generation.py`](../../scripts/verify_config_generation.py)
- **Preserved Untracked Work:** Pre-existing untracked helper `scripts/verify_agy_overlay.py` was inspected and preserved without staging or modification.
- **Candidate Stack Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) retains `status: candidate`.

---

## 3. Evidence Ledger and Cryptographic Audit

- **Evidence Ledger:** `/home/ra/almadial-lab/evidence/m0-05/20261105T222428Z/qa-evidence-ledger.json` (SHA-256 `dec1d009dda76337c3e025f6684a9731ac98e23123a9ff73ff535cb3e2fc5579`).
- **Artifact Count:** Exactly **81 files**.
- **Audit Outcome:** 100% of all 81 files were independently verified for byte length and SHA-256 checksum against the ledger. Zero missing or modified files.
- **Key Artifact Hashes Verified:**
  - `final-generated-private.tar.gz`: `71fcfcd2cf04a9759f00fc0e8cf7db562b2f337f3537e1905a9b5c61afbd458e`
  - `paired-echo-local.wav`: `eb9d4ca97eb6eb75618171d9d713da3d9055ee516ddddcf7a8bf4e8db674d81d`
  - `paired-echo-remote.wav`: `73752e2e7ce379f6e5e8aeb9b5b93d79fe1c6ee377a0b5a1b028dfbe480d1912`
  - `paired-repeat-echo-local.wav`: `eb9d4ca97eb6eb75618171d9d713da3d9055ee516ddddcf7a8bf4e8db674d81d`
  - `paired-repeat-echo-remote.wav`: `e0971bfae94d84fbb30c2c31502fa621ccf401f11a76f2f9f8f41da0fe681464`
- **Snapshot Integrity:** Snapshot `pre-alm5-config-audit` intact on `almadial-alma10`. Zero snapshot restorations were required or executed.
- **Protected QA Workspace:** Dedicated review evidence recorded under `/home/ra/almadial-lab/evidence/m0-05/qa-20261106T074000Z/` (mode `0700`).

---

## 4. VICIdial Source Derivation Integrity (Line 3655 Priority Correction)

In upstream VICIdial r4037, `ADMIN_keepalive_ALL.pl` line 3655 contains a latent dialplan priority defect under Asterisk 21+ execution:

```perl
# Upstream r4037 line 3655 inside the Asterisk 21+ branch for extension 8310:
$Vext .= "exten => 8309,2,MixMonitor(\${VICIDIAL_RECORDING_FILE}.\${VICIDIAL_RECORDING_EXT},b,\${VICIDIAL_MIXMON_COMMAND})\n";
```

1. **Defect Characterization:**
   - Inside the extension `8310` GSM recording block, priority 2 was incorrectly emitted targeting extension `8309`.
   - This produced a duplicate priority 2 for `8309` and left extension `8310` with missing priority 2 in generated dialplan.
2. **Derivation Helper ([`scripts/lab/m0_05_generator.py`](../../scripts/lab/m0_05_generator.py)):**
   - Enforces strict input validation against the ALM-4 derived `ADMIN_keepalive_ALL.pl` (SHA-256 `007fc3012ea5c37e5c5a330719ed51a621aa9ad84488ade3ef13eac9d64756d3`).
   - Substitutes exactly the single target line to emit `exten => 8310,2,MixMonitor...`.
   - Resulting derived SHA-256: `8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda`.
3. **Reproducibility:**
   - Independent re-derivation in Python reproduced bit-for-bit identical output matching the ledger hash.
   - Live inspection of `/etc/asterisk/extensions-vicidial.conf` on `almadial-alma10` confirmed that extension `8310` correctly contains priority 2, with zero duplicate priorities under `8309`.

---

## 5. Configuration Ownership Contract and Generated Outputs

The implementation formally documents and enforces the boundary between AlmaDial-owned base configuration files and VICIdial-generated include files:

1. **Ownership Matrix:**
   - **AlmaDial Owned:** Base files (`asterisk.conf`, `modules.conf`, `http.conf`, `pjsip.conf`, `extensions.conf`, `confbridge.conf`, `manager.conf`, `logger.conf`), TLS certificates, `transport-wss`, WebRTC echo endpoint `9001`, `#include` directives, and local AMI configuration.
   - **VICIdial Generator Owned:** 9 generated files:
     - `extensions-vicidial.conf`
     - `pjsip-vicidial.conf`
     - `pjsip_wizard-vicidial.conf`
     - `confbridge-vicidial.conf`
     - `iax-vicidial.conf`
     - `sip-vicidial.conf`
     - `meetme-vicidial.conf`
     - `musiconhold-vicidial.conf`
     - `voicemail.conf` (documented unsuffixed exception)
2. **Voicemail Ownership Exception:**
   - Upstream VICIdial reads and re-emits `voicemail.conf` directly without supporting a `-vicidial.conf` include structure.
   - This exception is explicitly identified, justified, and audited in [`docs/architecture/config-ownership.md`](../architecture/config-ownership.md) and [`scripts/verify_config_generation.py`](../../scripts/verify_config_generation.py).
3. **File System Permissions:**
   - All published files in `/etc/asterisk/` are owned by `root:asterisk` with mode `0640`.
   - Verified live on `almadial-alma10`.

---

## 6. Asterisk 22.12.0 PJSIP & Dialplan Integration (Endpoint 9051)

1. **PJSIP Endpoint 9051:**
   - Provisioned via `pjsip_wizard-vicidial.conf` using the `user_phone` wizard template.
   - Live Asterisk CLI query `pjsip show endpoint 9051` confirmed:
     - Bound to `@pjsip_wizard: 9051`
     - Context: `alm5-local`
     - Transport: `transport-wss`
     - Codecs: `(ulaw|alaw)`
     - DTLS / WebRTC: Enabled (`dtls_verify = no`, `dtls_cert_file = /etc/asterisk/keys/asterisk.pem`, `webrtc = yes`)
     - Device state: `Unavailable` (no active registered contact, ready for SIP.js/VICIphone registration).
2. **Dialplan Inclusion & Flow:**
   - Context `alm5-local` includes `vicidial-auto-phones`.
   - Extension `9051` correctly executes `${PJSIP_DIAL_CONTACTS(9051)},21,` with fallback to `alm5-local,9059,1` (`Hangup()`).
   - Extension `9001` (`alm3-echo`) remains active and unaffected: `1. Answer, 2. Echo, 3. Hangup`.
3. **Context Isolation:**
   - Legacy `vicidial-auto-external` MeetMe dialplan route is quarantined and unreachable from test extensions.

---

## 7. ConfBridge 300-Room Allocation & Asterisk Profiles

1. **Asterisk Profiles:**
   - Live Asterisk CLI commands `confbridge show profile users` and `confbridge show profile bridges` confirmed the loading of:
     - **9 User Profiles:** `vici_recording_user`, `vici_barge_user`, `vici_agent_user`, `vici_ringall_user`, `vici_monitor_user`, `vici_admin_user`, `vici_dtmf_user`, `vici_audio_user`, `vici_customer_user`.
     - **1 Bridge Profile:** `vici_agent_bridge`.
     - **1 Menu:** `vici_monitor_menu`.
2. **Database Room Allocation:**
   - Direct MariaDB query on `almadial_m005.vicidial_confbridges` confirmed exactly **300 conference rooms** (`conf_exten` from `9600000` to `9600299`) allocated to `server_ip = '127.0.0.1'`.
   - Legacy MeetMe module remains uninstalled; ConfBridge serves all dynamic conference bridging.

---

## 8. MariaDB & Perl Permission Corrections

The two architectural findings identified in the ALM-4 review were verified to be cleanly resolved:

1. **MariaDB Socket Isolation and Permissions:**
   - Configuration file `/etc/my.cnf.d/almadial-m004.cnf` is owned by `root:mysql` with mode `0640` (`-rw-r-----.`).
   - Contains `[mariadb]\nskip-networking`.
   - `ss -tulpn | grep 3306` confirms **0 TCP listeners** (MariaDB is completely isolated from the network).
   - Local socket connectivity verified (`SHOW VARIABLES LIKE 'skip_networking'` returned `ON`).
2. **Perl Site-Perl Traversal Permissions:**
   - `/usr/local/share/perl5` and all subdirectories are mode `0755` (`drwxr-xr-x. root:root`).
   - Library files are mode `0644`.
   - Non-root unprivileged test (UID 1000) confirmed that `DBI`, `DBD::mysql`, `Net::Telnet`, and `Time::HiRes` load cleanly without `Permission denied` errors.

---

## 9. Local AMI Security Audit

1. **Permissions & Binding:**
   - `/etc/asterisk/manager.conf` is owned by `root:asterisk` with mode `0640` (`-rw-r-----.`).
   - `ss -tulpn | grep 5038` confirmed binding strictly to `127.0.0.1:5038`.
2. **Live Socket & Privilege Boundary Testing:**
   - Connected to `127.0.0.1:5038` and received Asterisk AMI banner `Asterisk Call Manager/11.0.0`.
   - **Invalid Login:** Rejected with `Response: Error`, `Message: Authentication failed`.
   - **Valid Login (`alm5-local`):** Accepted with `Response: Success`, `Message: Authentication accepted`.
   - **Action `Ping`:** Returned `Response: Success`, `Ping: Pong`.
   - **Action `CoreStatus`:** Returned `Response: Success`, `CoreCurrentCalls: 0`.
   - **Privilege Boundary Enforcement:** Action `Command` (`core show version`) returned `Response: Error`, `Message: Permission denied`.
   - Confirms that the local AMI account has minimal required privileges and cannot execute arbitrary shell or core Asterisk CLI commands.

---

## 10. WebRTC Echo Audio Quality & Baseline Preservation

Independent re-analysis of the paired WebRTC audio recordings using [`scripts/lab/m0_03_signal.py`](../../scripts/lab/m0_03_signal.py) confirmed:

1. **First Paired Run (`paired-echo`):**
   - Result: **`PASS`**
   - Envelope correlation: `0.9991` (threshold: 0.85)
   - Pattern match: `1.0` (threshold: 0.9)
   - Tone agreement: `1.0` (threshold: 0.9)
   - Estimated latency: `40 ms` (max: 2000 ms)
   - Remote RMS: `0.1133` (threshold: 0.02)
   - Frequencies observed: `[697, 941, 1209, 1633] Hz`
2. **Second Paired Run After Asterisk Reload (`paired-repeat-echo`):**
   - Result: **`PASS`**
   - Envelope correlation: `0.9822` (threshold: 0.85)
   - Pattern match: `0.9955` (threshold: 0.9)
   - Tone agreement: `0.9810` (threshold: 0.9)
   - Estimated latency: `40 ms` (max: 2000 ms)
   - Remote RMS: `0.1133` (threshold: 0.02)
   - Frequencies observed: `[697, 941, 1209, 1633] Hz`
3. **Capture Alignment Analysis:**
   - Evaluated the unaligned run `final-echo` (pattern match `0.8678` due to leading frame drop during independent capture startup).
   - Confirmed Codex's diagnosis that paired simultaneous capture via [`scripts/lab/m0_05_capture.js`](../../scripts/lab/m0_05_capture.js) eliminates startup jitter, yielding bit-aligned 802 frames on both channels.

---

## 11. Repository Automated Validation

The independent reviewer executed the full repository test suite:

1. `python3 scripts/validate_scaffold.py`: **`PASS`** (11 required files, 4 skills).
2. `python3 -m unittest discover -s scripts/tests -v`: **81/81 unit tests PASSED** in 0.952s.
   - Includes 25 new `ConfigurationContractTests` in `test_verify_config_generation.py`.
3. `python3 scripts/verify_candidate_stack.py`: **`PASS`** in LOCAL mode.
4. `python3 scripts/verify_config_generation.py`: **`PASS`** (`REDACTED_FIXTURE_REPLAY`).
5. `git diff --check`: Clean; zero trailing whitespace or format defects.

---

## 12. Final Acceptance Verdict and Downstream Recommendations

### Acceptance Verdict:
**`ALMADIAL_M0_05_ACCEPTED`**

Task ALM-5 satisfies all acceptance criteria established in `docs/AlmaDial_Implementation_Plan.md`. The configuration ownership boundary is cleanly defined, upstream priority defect on line 3655 is corrected via deterministic derivation, PJSIP endpoint 9051 and ConfBridge profiles load into Asterisk 22.12.0 without error, 300 conference rooms are populated, local AMI access is strictly bound to `127.0.0.1` with privilege enforcement, MariaDB and Perl permission defects are resolved, and WebRTC echo returns verified four-tone audio.

### Accepted Limitations:
1. **Endpoint Registration Scope:** Endpoint 9051 was validated through Asterisk wizard parsing, dialplan contact resolution, and codec negotiation; full SIP.js browser registration and agent session attachment is downstream in ALM-6.
2. **MeetMe Dialplan Quarantined:** Legacy MeetMe routing in `vicidial-auto-external` is excluded from test contexts and unmaintained.
3. **Keepalive Daemon Execution:** `ADMIN_keepalive_ALL.pl` was audited and executed boundedly to generate configs; continuous cron/daemon background operation is not yet active.
4. **Manifest Status:** [`manifests/stack.lock.yaml`](../../manifests/stack.lock.yaml) remains in `status: candidate`.

### Next Assignment Recommendation (ALM-6):
- **Task:** **ALM-6 / M0-06** (`xyu5rkqfhnnjl82bq5ywt9t8`) — *Validate VICIdial agent WebRTC phone, audio and conferences*.
- **Role:** Principal WebRTC / Telephony Audio Verification Architect.
- **Scope:** Connect VICIphone/SIP.js via HTTPS/WSS to PJSIP endpoint 9051, join ConfBridge conference room (e.g. 9600001), verify bidirectional browser-to-conference audio streaming, test agent barge/monitor functions, and confirm call recording file creation.
- **Precondition:** ALM-5 acceptance unblocks ALM-6 (native dependency `ypz5z067r26245n5j5980x9d`).
- **ALM-6 was not started during this assignment.**
