# ALM-7 local compatibility scope — approved operator decision

Prepared by Codex, 2026-11-06 UTC (2026-11-07 Asia/Manila).
**APPROVED — ALM-7-only amendment; aggregate M0 and ALM-8 gates unchanged.**

Approval source: the operator's reply in the current Codex conversation,
verbatim **“yes I approve”**, directly answering:

> Do you approve the ALM-7-only amendment in the proposal, including the ALM-27 external-qualification addendum, while leaving ALM-8 and the aggregate M0 exit criteria unchanged?

Approval was recorded on 2026-11-06 UTC in Kaneo ALM-7 comment
`nuggyhfh2z7ysi1jc3bvls89`; that comment also preserves the entire original
ALM-7 description before replacement. Its content was read back exactly through
MCP. The reviewed draft SHA-256 was
`424370a6868c39cf086f1650ec000b6fc2ba73ceac8c7a82393dfa44968e843d`;
its unchanged copy remains in protected preparation evidence
`~/almadial-lab/evidence/m0-07/scope-disposition-20261106T175455Z/`.

The four approved ALM-7 sections and ALM-27 addendum below have been applied and
read back exactly. Both plan paths and the existing runtime report reflect the
same boundary. External profiles remain NOT_RUN. ALM-27 remains Backlog and
unassigned; ALM-29 is unchanged. Independent AGY review remains mandatory.
QA Review handoff status and final evidence are recorded in the existing runtime
report. No aggregate M0 exit waiver, ALM-8 start, lock promotion, infrastructure
change or production-readiness claim is authorized.

The remaining text preserves the proposal and audit **as reviewed before
approval**. References below to an absent decision, unchanged files or the
preparation verdict describe that historical preparation, not the applied state.

Task: ALM-7 / M0-07, `rwpendvaoqpi9mrzgskej6qb`, AlmaDial project
`ghritpte6e40o5ayktgz0vh1`. Implementation owner remains Codex.
This is a governance proposal, not a second runtime report. The implementation
record remains [the existing runtime report](../compatibility/m0-07-viciphone-browser-runtime.md).

## Decision requested and authorization audit

Approve or reject the exact ALM-7 amendment below: qualify the pinned local/LAN
VICIphone integration independently, retain direct-public, 1:1 NAT and
remote-home-agent tests as NOT_RUN, and retain all three as mandatory external
deployment qualification under the existing ALM-27 / M3-05 task, with independent
release repetition under ALM-29 / M4-01. No task is created or assigned by this
proposal. No external testing infrastructure or exposure is authorized by it.

No explicit operator approval of this amendment was found in the retrieved
ALM-7 description, its two comments, six activity entries or custom fields; the
descriptions/comments of the candidate downstream tasks; or the project plans,
runtime report and governance documents searched for an ALM-7 scope disposition.
The current assignment explicitly authorizes preparation, not approval. An AI
recommendation, successful LAN tests and this draft do not satisfy the gate.

Existing continuation comment `rlg17bykh6nt8ya5rxn4b88y` was retrieved through
`kaneo-local` and compared with its protected read-back. It explicitly records
BLOCKED pending external qualification or a scope amendment. It is implementation
evidence, not operator authorization. Original comment
`uq74ayopwguhauf8hdpf4wl8` is also retained.

Until an explicit decision is recorded: preserve the original criteria,
ALM-7 In Progress/Codex, ALM-8 Backlog/unassigned, and all dependencies. Do not
move ALM-7 to QA Review or claim acceptance.

## Existing acceptance contract

The current task's scope says:

> Pin and self-host reviewed VICIphone and JS dependencies; test selected direct-WSS or 443-proxy topology against the chosen VICIdial settings-container and phone templates. Cover LAN, direct-public, 1:1 NAT and remote-home-agent P0 paths, reconnect and certificate negatives.

Its acceptance criteria say:

> All §10.3 P0 cases pass with actual browser audio; browser-only credentials do not leak into exported logs/URLs; no signaling-only PASS; restricted NAT/IPv6 limitations explicitly stated; optional TURN not silently added.
>
> All mandatory cases must have attributable evidence. A missing or failed mandatory case remains BLOCKED/NOT_RUN/FAIL and prevents acceptance. Deliverables must be reproducible from the identified baseline; report limitations and the next bounded task.

Consequently, the three unexecuted external profiles currently prevent ALM-7
acceptance. No existing FAIL has been relabeled as an environmental limitation.
Confirmed lifecycle defects were corrected and retested in the recorded
continuation. The missing external layouts were never executed.

## Requirement reconciliation

Evidence abbreviations: **O** = original protected evidence
`~/almadial-lab/evidence/m0-07/20261106T072425Z/`; **C** = continuation
`~/almadial-lab/evidence/m0-07/continuation-20261106T150016Z/`.
PASS below means recorded implementation runtime evidence supports the bounded
claim. It does not mean independent acceptance. The current acceptance contract
still applies until amended. Sensitive artifacts stay outside Git.

| Original requirement / necessary local workflow | Implementation result and attributable evidence | Current acceptance disposition | Proposed boundary / remaining issue |
|---|---|---|---|
| Exact self-hosted VICIphone and JS; no opaque CDN dependency | PASS: VICIphone 3.0 `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`, embedded SIP.js 0.20.1; C `final-overlay/asset-manifest.json`, source provenance | Local evidence complete; AGY pending | Local; 46 phone assets, 59 total deployed files verified |
| Real agent authentication and correct phone/session association | PASS: real VICIdial login, embedded 9051 and separate 9052; C `controls-final-browser.json`, `controls-final-heartbeat-window.json` and associated live-state records | Local evidence complete; AGY pending | Local; no seeded login substituted for UI authentication |
| Trusted HTTPS and WSS separately; generated PJSIP registration/contact lifetime | PASS: actual browser registration and ICE/DTLS-SRTP; C final browser/state records and recovery contact observations | Local evidence complete; AGY pending | Local direct WSS topology; no claim for untested 443 proxy |
| Agent initialization, heartbeat, conference and time reporting | PASS after bounded upstream AST_update/listener/sender corrections; C `heartbeat-live.json`, `updater-fixed-live.json`, `controls-final-heartbeat-window.json` | Local evidence complete; AGY pending | Local; upstream processes must run before replay; stale heartbeat correctly blocks controls; boot automation is not qualified |
| PHP/MariaDB application semantics | PASS: live strict-mode failures 1366/1406/1292 reproduced and corrected at pinned operations; C `strict-sql-live-probe.json` | Local evidence complete; AGY pending | Local; global/PHP/Perl strict modes preserved; previous connection-wide workaround removed |
| Functional agent JavaScript/calendar initialization | PASS: absent calendar registry guard, existing calendar behavior retained; final browser errors empty and offline regression | Local evidence complete; AGY pending | Local; no global exception suppression |
| Actual two-way browser audio; no signaling-only PASS | PASS: C `controls-final-browser.json` plus `controls-final-media-analysis.json`; patterns 0.99209 both ways, correlation 0.99823/0.98423, tone 0.99782/1.00000, delay 60/40 ms | Local evidence complete; AGY pending | Local; distinct injected microphone signals traverse real browser media; physical microphone/speaker NOT_RUN |
| Conference join/leave and agent/customer association | PASS: same selected PJSIP customer in conference 9600000; C final browser/CLI/heartbeat records | Local evidence complete; AGY pending | Local; corrected fixture puts the customer in the conference the UI actually inspects |
| Supported park/hold control | PASS for actual VICIdial park/resume: same customer enters park AGI then returns to 9600000; C `controls-final-park-state.json`, `controls-final-resume-state.json` and browser states | Local evidence complete; AGY pending | Local; phone SIP hold NOT_APPLICABLE to this UI; mute is separate |
| Supported transfer control and failure handling | PASS: actual UI blind transfer to local 9600001, agent remains 9600000, disposition XFER; empty destination rejected; C `controls-final-transfer-state.json`, `controls-final-disposition.json`, final browser record | Local evidence complete; AGY pending | Local; standalone AMI Redirect is not UI proof; phone SIP REFER NOT_APPLICABLE |
| Hangup and supported DTMF | PASS: UI hangup/CLI cleanup; O `dtmf-info-module-receipt.json`, `dtmf-new-call-receipt.json`, `viciphone-dtmf-receipt.json` establish SIP INFO reception | Baseline retained; AGY pending | Local; final keypad click is not claimed as a new DTMF reception measurement |
| Logout, authorization revocation and fresh login | PASS: C `controls-final-revocation.json`, `revocation-all-endpoints-final.json`; all three protected handlers reject old session names; cookie list empty, fresh login works | Local evidence complete; AGY pending | Local; retained abandoned database rows are history, not evidence of authorization; original count-only defect was corrected |
| Unexpected idle and active WSS loss, reconnect | PASS for visible WS Disconnected, manual Register recovery, explicit old-call hangup and successful new call; C `recovery-newcall-final-browser.json` plus paired CLI states | Local evidence complete for documented manual contract; AGY pending | Local; existing media may continue on signaling loss; automatic reconnect and seamless original-call continuity are not claimed |
| No stranded bridge or duplicate registration after recovery/refresh | PASS for recorded recovery and fresh-login paths; C recovery states and `final-lifecycle-preservation.json`; O recovery browser/live records | Local evidence complete; AGY pending | Local; one contact per endpoint after recovery, final resource counts zero |
| Recording observations | PASS for controlled AMI conference recording; O `recording-content-analysis.json`, `ffprobe-viciphone-recording.json`: 17.18 s decodable audio with both patterns | Baseline retained; AGY pending | Local baseline only; no agent recording-UI/metadata claim or new recording replay |
| Invalid/expired/mismatched certificates and invalid credentials rejected | PASS: O `negative-results.json`, `negative-hostname-results.json` and corresponding traces | Baseline retained; AGY pending | Local negatives; TLS validation retained |
| Browser credentials protected in exported logs/URLs | PASS for final audited surfaces: C `credential-audit-controls.json`, 1,546 requests, 2 Referers, 22 console events, 14 history URLs, 161,642 access-log bytes, zero tested leaks | Local evidence complete; AGY pending | Local; escaped POST/debug suppression retained; private DOM, POST and SIP traces remain sensitive |
| LAN §10.3 P0 media | PASS: actual two-client final conference replay and unchanged thresholds | Local evidence complete; AGY pending | Local/LAN only; Multipass NAT does not establish public NAT traversal |
| Direct-public §10.3 P0 registration/dial/bridge/recording | NOT_RUN: no authorized public interface/address, routing, DNS/TLS or external browser supplied | BLOCKED under current ALM-7 criteria | External; proposed deferral only, never PASS |
| 1:1 NAT §10.3 P0 media/SIP routing | NOT_RUN: no authorized controllable fixed mapping/gateway/ingress supplied | BLOCKED under current ALM-7 criteria | External; proposed deferral only, never PASS |
| Remote-home-agent §10.3 P0 WSS and two-way media | NOT_RUN: no authorized external home browser/router and qualified public server path supplied | BLOCKED under current ALM-7 criteria | External; proposed deferral only, never PASS |
| Restricted NAT / IPv6 limitations; optional TURN | NOT_RUN for runtime qualification; limitations explicitly recorded, no TURN added | Documentation requirement met; no runtime support claim | P1 limitations remain; no inferred traversal, IPv6 or TURN support |
| Preserve accepted telephony, security and complete cleanup | PASS in C `final-lifecycle-preservation.json`: Asterisk 22.12.0, 9001/9051/9052, fifteen preserved hashes, SELinux Enforcing, socket-only DB, loopback AMI/FastAGI; zero active call/session resources | Local evidence complete; AGY pending | Local; bounded test processes stopped, snapshot verified and never restored |
| Independent reviewer and explicit acceptance | NOT_RUN for ALM-7; Codex is implementer | Still mandatory | Separate future AGY assignment; no self-acceptance |

Signal thresholds were not reduced: pattern ≥0.90, correlation ≥0.85, tone ≥0.90,
RMS ≥0.02, delay <2000 ms. The media analyzer alone labels runtime state NOT_RUN;
the runtime claim comes from its pairing with actual browser and Asterisk records.
Manual bounded updater/sender startup is an explicit replay prerequisite, not
evidence of unattended boot/service readiness. Existing failed attempts and
superseded derivations remain in the historical evidence.

## Exact proposed ALM-7 changes — apply only after approval

Preserve the complete original description in task history before amendment.
Keep identity, implementation owner, dependencies, exclusions and independent
review requirements. Replace only these four task sections as follows.

**Scope and required changes — replacement substantive paragraph:**

> ALM-7 establishes local/LAN compatibility of pinned self-hosted VICIphone 3.0 and SIP.js 0.20.1 with VICIdial r4037 and Asterisk 22.12.0 on the authorized AlmaLinux 10 lab target. Qualify the selected trusted HTTPS/direct-WSS topology, generated phone association, real agent and browser media lifecycle, supported application call controls and documented recovery. Direct-public, 1:1 NAT and remote-home-agent profiles remain NOT_RUN and are deferred from this task's acceptance to mandatory external qualification under ALM-27 / M3-05 and release repetition under ALM-29 / M4-01. Local acceptance does not qualify remote/public deployment or production readiness. Existing remote-target authorization, snapshot and safety restrictions remain applicable.

**Deliverables — replacement:**

> Exact phone/browser/JS/source identities and deterministic derivation hashes; selected WSS topology decision; attributable local browser/SIP/ICE/DTLS/SRTP and two-way audio evidence; heartbeat, supported call-control, logout-revocation and cleanup records; controlled recording observations with their actual scope; documented manual WSS recovery; explicit NOT_RUN external profiles, prerequisites and downstream mandatory gate. Preserve the original implementation evidence and its limitations.

**Verification procedure — replacement substantive paragraph:**

> Independently reproduce real VICIdial agent login and embedded phone association, certificate-validated HTTPS and WSS separately, secure registration/contact uniqueness, two-way conference media at unchanged thresholds, conference join/leave, supported park/resume and local transfer, hangup and supported DTMF, heartbeat initialization, strict SQL compatibility, calendar initialization, browser refresh/fresh login, idle and active unexpected WSS loss with the documented manual Register/old-call termination/new-call sequence, recording observations, invalid credentials/certificate rejection, logout authorization revocation and complete call/session cleanup. Audit tested credential URL/log surfaces and verify preservation of accepted telephony/security baselines. Retain the existing repository-check paragraph. No simulated check replaces live evidence.

**Acceptance criteria — replacement:**

> ALM-7 acceptance is limited to local/LAN compatibility of the identified pinned stack. Require attributable real-execution evidence and independent AGY verification of: (1) real VICIdial agent authentication; (2) correct phone/session association; (3) secure generated PJSIP registration; (4) bidirectional WebRTC conference media; (5) supported agent call controls; (6) agent heartbeat and initialization; (7) logout authorization revocation; (8) predictable documented WSS recovery; (9) certificate and credential enforcement; (10) deterministic asset/source compatibility; (11) preservation of the accepted telephony runtime; (12) complete local call/session cleanup; and (13) separate independent AGY QA and explicit maintainer acceptance. Preserve controlled recording observations, supported DTMF evidence and documented limitations. No signaling-only PASS, secret leakage or weakened TLS/SQL/SELinux protections is acceptable. Missing or failed mandatory local cases prevent acceptance.
>
> Direct-public, 1:1 NAT and remote-home-agent profiles are NOT_RUN and deferred, not passed. They remain mandatory deployment/release qualifications under ALM-27 / M3-05 and ALM-29 / M4-01 before remote-agent or production-readiness claims. Restricted NAT, IPv6, physical-device and automatic same-call continuity limitations remain explicit; TURN is not implicitly introduced. This amendment does not accept ALM-7, start ALM-8, waive aggregate M0 exit requirements, promote the candidate manifest or authorize infrastructure changes.

## Exact proposed downstream qualification addendum

Append the following to the existing ALM-27 / M3-05 acceptance criteria, without
removing its current requirements. Cross-reference the same obligation in the
ALM-7 runtime report and plan. ALM-29 already explicitly requires these profiles
and twice-run independent evidence; retain its criteria and dependencies.

> Mandatory external-network qualification retained from ALM-7: before any remote-agent or production deployment-readiness claim, execute and independently verify (1) direct-public HTTPS/WSS and WebRTC media; (2) authorized fixed 1:1 NAT traversal; (3) remote-home-agent login and browser media; (4) correct public/private address advertisement; (5) ICE candidate and RTP routing correctness; (6) certificate hostname validation; (7) remote-agent supported call controls and documented reconnect behavior; (8) controlled recording and call/session cleanup under each tested topology; (9) no unintended public exposure of AMI, MariaDB or FastAGI; and (10) independently attributable evidence and explicit acceptance. No LAN or Multipass-only result substitutes for an external profile. Missing topology authority or infrastructure leaves the corresponding case NOT_RUN/BLOCKED and prevents remote-agent/deployment qualification. Repeat the accepted external matrix under ALM-29 / M4-01 before release acceptance. This requirement does not authorize public infrastructure, firewall changes, PSTN calls or spending.

Primary placement: **ALM-27 / M3-05**,
`ea1axf0373fggib2v5u7ciyv`, currently Backlog/unassigned. This task already
requires all §10.3 P0 cases and actual call/conference/media/recording evidence.
There is no authorized downstream owner yet. Do not assign one by inference.
ALM-25 / M3-03 supplies network/NAT configuration prerequisites; it is not the
acceptance owner. Existing sequence ALM-25 → ALM-26 → ALM-27 → ALM-28 → ALM-29
and ALM-29 → ALM-34 → ALM-35 retains deployment/release gates.

Release repetition: **ALM-29 / M4-01**, `xu7wxvch94ii2iw38hpb8d58`, currently
Backlog/unassigned. Its scope already names direct-public, 1:1 NAT and remote
agents; acceptance requires all P0 cases twice and independent evidence.
ALM-34 / M4-06 and ALM-35 / M4-07 retain operator/release acceptance obligations.
No new task, new edge or broader board restructuring is proposed.

## Proposed document edits and the separate M0 gate

After approval only, add the approved ALM-7 boundary immediately after the plan's
§10.3 table, leaving all priorities and expected outcomes intact:

> ALM-7's task-specific acceptance is local/LAN compatibility, with explicit approved deferral of the direct-public, 1:1 NAT and remote-home-agent profiles. Those profiles remain mandatory under ALM-27 / M3-05 and ALM-29 / M4-01 before remote-agent/deployment or release-readiness claims. Local acceptance is not public-network qualification. This task-specific boundary does not amend the aggregate M0 exit gate in §§9.3 and 14 or ALM-8's complete-baseline repetition requirements.

Change only the §14 M0-07 evidence cell to:

> Local/LAN registration, SRTP, ICE, audio, heartbeat, supported call controls, logout revocation, documented WSS recovery and independent QA; external profiles remain NOT_RUN pending mandatory qualification.

Add to the existing runtime report an approved scope-disposition section with the
actual approval source, exact approved text, downstream IDs, retained NOT_RUN
profiles and conditional QA status. Preserve all historical findings and hashes.
Both plan paths currently have identical bytes (SHA-256
`2a8effec4e1e40d3577382158303d4e3124e56712e9877d3cac387a68bc0cfd4`);
keep `docs/AlmaDial_Implementation_Plan.md` and `docs/implementation-plan.md`
consistent if this bounded edit is authorized. Do not edit either before approval.

**Aggregate-gate consequence requiring attention:** §§9.3 and 14, ALM-8 and the
earlier plan review still require the complete M0 baseline, every mandatory case,
second fresh VM and twice-run evidence before M0 exit. This proposal changes only
ALM-7's acceptance boundary. Even after local AGY acceptance, ALM-8 must not be
automatically started or completed; M0 does not become accepted and the stack
manifest remains candidate. Deferring external tests from the entire M0 exit to
M3 would require a separate explicit decision addressing those aggregate clauses
and ALM-8, because M3 currently follows M0. It cannot be inferred from approval of
this ALM-7-only proposal. Without that broader decision, external evidence must
still be supplied before the existing full M0 exit can be accepted. This avoids
silently bypassing the milestone ordering or creating a hidden circular gate.

## Evidence integrity and validation in this preparation

Both historical ledgers and all their listed artifact sizes/hashes were verified
again without writes to either directory:

| Evidence | Artifacts verified | Ledger SHA-256 |
|---|---:|---|
| Original O | 255 | `7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe` |
| Continuation C | 426 | `6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c` |

The runtime report still matches its protected final read-back SHA-256
`07ddc2aaf7e3c73c23a98a75d34bcfd782935fdf2b5be3aceac79f79d0dca156`.
The three ALM-7 scripts match their continuation provenance. Final deployed asset
manifest SHA-256 remains
`6510b421898c9b0c739e76167c7acbba4edfd245328702b581737221088f6325`.
Underlying browser, audio, heartbeat, strict SQL, call-control, revocation,
recovery, credential-audit and final preservation records were inspected. This
was evidence review, not a new live runtime execution. No VM was modified, no
snapshot was restored and the full integration suite was not rerun.

Starting HEAD for this assignment:
`6d23453452f00b96ae152f0e15a9a32de075c96f`. Two intervening operator commits
since the recorded `c3e5593921c55e65667418cd446d2d75ba541083` affect only
`operations/portal443/`; they do not change ALM-7 runtime scripts. Existing dirty
and untracked files are preserved; this draft is the only intended new repository
file. No staging, commit or push is authorized or performed.

| Required check | Result | Exit code |
|---|---|---:|
| `python3 scripts/validate_scaffold.py` | PASS; guidance/scaffold only | 0 |
| `python3 -m unittest discover -s scripts/tests -v` | PASS; 118 tests, including 22 ALM-7 tests | 0 |
| `python3 scripts/verify_candidate_stack.py` | PASS candidate consistency; package/source/runtime execution NOT_RUN | 0 |
| `python3 scripts/verify_config_generation.py` | PASS redacted fixture replay; runtime NOT_RUN | 0 |
| `python3 -m py_compile scripts/lab/m0_07_assets.py scripts/tests/test_m0_07_assets.py` | PASS | 0 |
| `node --check scripts/lab/m0_07_browser.js` | PASS | 0 |
| `git diff --check` | PASS | 0 |

## Conditional handoff and present disposition

Only after explicit authorization is recorded may Codex preserve history, apply
the precise approved changes, update the existing report and both plan paths,
append the authorized ALM-27 qualification addendum, and verify every tracker
write through MCP read-back. Reconcile any operator edits before applying.
Then, if the authorized local contract remains supported, move ALM-7 to QA Review
while retaining Codex ownership; never directly to Accepted.

Recommend a separate future Google Antigravity assignment with the highest
approved Gemini Pro-class reasoning model, High effort, as Principal
VICIphone/WebRTC Integration QA Engineer. Use Kaneo, Git, Multipass,
Playwright/Chromium, Asterisk CLI/AMI, MariaDB, PHP/JavaScript diagnostics,
Python, SHA-256 and existing audio-analysis tooling. Independently reproduce
source/asset integrity, real agent login/media, heartbeat, supported controls,
logout revocation, manual WSS recovery, certificate handling and accurate network
disposition. AGY must verify the recorded authorization and cannot invent it.

Current verified task state: ALM-6 Accepted; ALM-7 In Progress/Codex;
ALM-8 Backlog/unassigned. Native dependencies `rapwb42p3l0v6r8t9ym7dkz5`
(6 → 7) and `sq76uz00g3qczpiwnamj1d26` (7 → 8) are intact.
No Kaneo write, new comment, task reassignment or governance transition is
performed during this unapproved proposal preparation.

Preparation verdict before approval (superseded): **ALMADIAL_M0_07_SCOPE_DECISION_REQUIRED**.
