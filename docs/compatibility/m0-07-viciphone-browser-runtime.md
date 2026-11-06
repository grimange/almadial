# M0-07 — Self-hosted VICIphone browser and agent runtime

Date: 2026-11-06 UTC. Implementer: Codex. Kaneo: ALM-7 / M0-07,
`rwpendvaoqpi9mrzgskej6qb`, project `ghritpte6e40o5ayktgz0vh1`.
Current scope: **operator-approved local/LAN compatibility acceptance**.
Current verdict: **ALMADIAL_M0_07_VICIPHONE_READY_FOR_QA**. Kaneo reports
**QA Review**, retaining Codex as implementation owner. The recorded local agent
lifecycle, media, supported controls, logout revocation and manual WSS recovery
support the approved independent QA handoff. The three
external network profiles remain NOT_RUN and mandatory downstream. Codex retains
implementation ownership. Independent AGY acceptance is pending; no production
readiness or candidate-stack promotion is claimed.

## Approved scope disposition — 2026-11-06 UTC

The operator explicitly replied **“yes I approve”** to the exact question:

> Do you approve the ALM-7-only amendment in the proposal, including the ALM-27 external-qualification addendum, while leaving ALM-8 and the aggregate M0 exit criteria unchanged?

The [approved decision record](../planning/alm-7-network-scope-proposal.md)
contains the exact amended text and original-versus-proposed reconciliation.
Reviewed draft SHA-256:
`424370a6868c39cf086f1650ec000b6fc2ba73ceac8c7a82393dfa44968e843d`.
Kaneo approval/history comment `nuggyhfh2z7ysi1jc3bvls89` preserves the original
complete ALM-7 description before amendment; its content was read back exactly.
This is an actual operator decision, not an inferred approval from LAN results.

### Applicable acceptance contract

ALM-7 acceptance is limited to local/LAN compatibility of the identified pinned stack. Require attributable real-execution evidence and independent AGY verification of: (1) real VICIdial agent authentication; (2) correct phone/session association; (3) secure generated PJSIP registration; (4) bidirectional WebRTC conference media; (5) supported agent call controls; (6) agent heartbeat and initialization; (7) logout authorization revocation; (8) predictable documented WSS recovery; (9) certificate and credential enforcement; (10) deterministic asset/source compatibility; (11) preservation of the accepted telephony runtime; (12) complete local call/session cleanup; and (13) separate independent AGY QA and explicit maintainer acceptance. Preserve controlled recording observations, supported DTMF evidence and documented limitations. No signaling-only PASS, secret leakage or weakened TLS/SQL/SELinux protections is acceptable. Missing or failed mandatory local cases prevent acceptance.

Direct-public, 1:1 NAT and remote-home-agent profiles are NOT_RUN and deferred, not passed. They remain mandatory deployment/release qualifications under ALM-27 / M3-05 and ALM-29 / M4-01 before remote-agent or production-readiness claims. Restricted NAT, IPv6, physical-device and automatic same-call continuity limitations remain explicit; TURN is not implicitly introduced. This amendment does not accept ALM-7, start ALM-8, waive aggregate M0 exit requirements, promote the candidate manifest or authorize infrastructure changes.

The ALM-7 scope, deliverables, verification and acceptance sections now carry the
approved local boundary. ALM-27 / M3-05 (`ea1axf0373fggib2v5u7ciyv`)
has the exact approved external qualification addendum, read back through MCP.
It remains Backlog/unassigned; no downstream owner or infrastructure was created.
ALM-29 / M4-01 (`xu7wxvch94ii2iw38hpb8d58`) retains its existing
all-P0, twice-run and independent release qualification requirements.

| Profile | Implementation result | Approved ALM-7 disposition |
|---|---|---|
| Local/LAN | PASS in recorded actual browser media/agent lifecycle tests | Required for independent local compatibility QA |
| Direct-public | NOT_RUN; authorized public topology absent | Deferred from ALM-7 only; mandatory external qualification |
| Fixed 1:1 NAT | NOT_RUN; authorized mapping/gateway absent | Deferred from ALM-7 only; mandatory external qualification |
| Remote-home-agent | NOT_RUN; external home browser and public server path absent | Deferred from ALM-7 only; mandatory external qualification |

No runtime result was changed from FAIL/NOT_RUN to PASS by the amendment.
Physical-device audio, public NAT traversal, automatic reconnect and seamless
same-call continuation remain unclaimed. The qualified WSS contract is visible
disconnection, manual Register, explicit old-call termination where needed and a
successful new call. The recorded local controls are VICIdial park/resume and
blind transfer; SIP hold/REFER are not supported controls in this client UI.
Controlled recording is the existing AMI conference baseline, not an agent
recording-UI/metadata qualification.

Both plan paths retain all §10.3 priorities and expected outcomes. Only the
approved ALM-7 boundary and M0-07 evidence cell changed. **Aggregate M0 exit in
§§9.3 and 14, ALM-8 criteria, its Backlog/unassigned state, milestone ordering,
second-clean-VM repetition and candidate-manifest status remain unchanged.**
External evidence is still required before the existing full M0 exit can be
accepted; this ALM-7 decision is not a waiver of that wider gate. No ALM-8 work
was started.

### Evidence and verification for the scope application

Starting/current HEAD: `6d23453452f00b96ae152f0e15a9a32de075c96f`.
The two intervening commits since the recorded runtime baseline affect only
`operations/portal443/`; the three ALM-7 scripts still match their recorded
continuation hashes. No runtime implementation was changed and no full media
replay was needed. No VM modification or snapshot restoration occurred.

The original 255 and continuation 426 evidence artifacts were reverified
without modifying either historical ledger:

- Original: `7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`.
- Continuation: `6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c`.
- Preparation: `7845a6dd74ab414cf930b7c3b290f2e167bf2625db37b295ad860bb58cc554cd`.

Approval-application evidence is kept separately under
`~/almadial-lab/evidence/m0-07/scope-approved-20261106T175823Z/`.
Historical preparation and runtime evidence remain intact. The ledger and final
MCP read-back for this application are recorded below.

### Independent QA handoff

All seven required repository commands passed with exit code 0, including
118 tests (22 ALM-7). Candidate consistency and redacted configuration replay are
offline checks; their runtime fields remain NOT_RUN. Final added documentation
was checked again with scaffold validation and `git diff --check`.

Kaneo readiness comment `vzdqgb5yv0qqh06c2a6vhbdx` was posted and read back
exactly, then ALM-7 was moved to **QA Review** (`in-review`). MCP read-back
verified Codex ownership, the approved description, ALM-6 Accepted, ALM-8
Backlog/unassigned, and unchanged ALM-6 → ALM-7 → ALM-8 dependencies. ALM-27
remains Backlog/unassigned with its approved addendum; ALM-29 is unchanged.
There were five bounded Kaneo writes: approval/history comment, ALM-7 description,
ALM-27 addendum, readiness comment and QA Review transition. Each was read back.
No independent acceptance has occurred.

The approval-application ledger contains **12 verified artifacts**, SHA-256
`7b348f5229d7a53c0d0bb48208fb9b5a97c1ef5f787e1de2f9d6189b15c5da8a`.
The final `mcp-final-readback.json`, `repository-final.json`, finalized report and
post-handoff validation are stored separately outside the seal because they
refer to this digest. The ledger explicitly identifies its pre-handoff snapshots
and exclusions. Only the two plan paths, approved decision record and this
existing runtime report changed in this application; 201 other inventoried files
and all three ALM-7 runtime scripts are unchanged. No staging, commit or push
occurred; HEAD remains unchanged.
Recommend a separate **Google Antigravity (AGY)** assignment, highest approved
Gemini Pro-class reasoning model, High effort, Principal VICIphone/WebRTC
Integration QA Engineer. Use `kaneo-local`, Git, Multipass, Playwright/Chromium,
Asterisk CLI/AMI, MariaDB, PHP/JavaScript diagnostics, Python, SHA-256 and existing
audio-analysis tools. Independently reproduce asset/source integrity, actual
agent login and two-way media, heartbeat prerequisites, strict SQL/calendar
behavior, supported controls, logout revocation, documented WSS recovery,
certificate/credential enforcement and complete cleanup. Verify this operator
approval and preserve the precise NOT_RUN external-profile disposition.

The following continuation and original implementation sections are historical
records. Their In Progress/BLOCKED governance wording describes the earlier
unamended contract; the approved scope disposition above controls current
ALM-7 acceptance. Their runtime findings, failures, prerequisites and evidence
hashes are preserved.

## Continuation scope and current result — 2026-11-06 16:49 UTC

The following continuation findings supersede open findings and deployed hashes
in the historical implementation record below. The original accepted media
baseline, its 255-artifact ledger and comment `uq74ayopwguhauf8hdpf4wl8` were
preserved. No installation was rebuilt. Only `almadial-alma10` was modified.
Starting/current HEAD remains `c3e5593921c55e65667418cd446d2d75ba541083`.

Continuation evidence: `~/almadial-lab/evidence/m0-07/continuation-20261106T150016Z/`.
Its private traces, authenticated DOM, HTTP POST bodies, original SQL failures,
audio and session identifiers must not be committed or exported as public logs.

| Gate | Current disposition |
|---|---|
| VICIphone 3.0 / SIP.js 0.20.1, self-hosting, trusted HTTPS/WSS | PASS, preserved; no external CDN required |
| Phone integrity | PASS, 46 phone assets; 59 total deployed files including lifecycle derivations/modules |
| Actual agent/phone authentication and 9051 association | PASS, fresh browser logins and embedded phone |
| Heartbeat and conference reporting | PASS with required bounded upstream updater running |
| Strict SQL and calendar exception | PASS after pinned source corrections |
| Logout, authorization revocation and fresh login | PASS; retained abandoned rows cannot authorize requests |
| Actual VICIdial park/resume and blind transfer | PASS to local destinations through the UI |
| Unexpected idle/active WSS loss | PASS for visible disconnect, manual Register recovery, explicit old-call termination and new call |
| Automatic same-call continuation / SIP REFER / phone SIP hold | NOT_APPLICABLE to the qualified recovery/control contract |
| LAN bidirectional browser media | PASS, unchanged objective thresholds |
| Direct-public / 1:1 NAT / remote-home-agent | NOT_RUN; mandatory and externally blocked |

### Heartbeat and conference initialization

This was missing application state, not demonstrated clock skew. NTP reports
`NTPSynchronized=yes`. The real agent periodically POSTs to
`conf_exten_check.php`. At pinned source line 1228 it reads
`UNIX_TIMESTAMP(last_update), UNIX_TIMESTAMP(db_time)` from `server_updater` for
the selected server. The table's `db_time` is a database-maintained timestamp
(`DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP`, schema lines 257–262).
The response drives the agent's time/health handling, including `TIME_SYNC` and
`SysteMDisablEBoX` at `vicidial.php:8484–8489`. The conference list uses
`live_sip_channels` and `live_channels`, filtered by server and conference
extension (`conf_exten_check.php:1621,1636`). Empty/stale application state can
therefore report unhealthy time or an empty session while RTP continues.

Upstream **`AST_update.pl`** creates the missing server row (lines 326–340), updates
`last_update` (1203), and populates live channel snapshots. This workflow requires
it; the warnings can disable actual controls and are not accepted as cosmetic.
The isolated server record is active, version 22.12.0, `conf_engine=CONFBRIDGE`,
with the accepted local context and clone database. The required process was run
as `asterisk` with `/etc/astguiclient-m007-updater.conf`, a dedicated local database
account and loopback-only AMI role. No campaign dialing suite was started.

The r4037 updater did not recognize Asterisk 22's channel format/AMI framing and
PJSIP channels. The pinned derivation selects its existing format 4 for 22.x,
normalizes successful AMI `Output:` responses, adds PJSIP to the existing phone
lookups, and preserves real PJSIP channel names. The upstream manager listener
also needed 22.x `Newstate`/`Hangup` field normalization at its four existing parser
sites. Other version branches remain unchanged. The listener and single-action
sender support the actual call queue; they do not replace VICIdial.

Healthy fresh agent runs showed current heartbeat and hidden startup warnings.
The final 15-minute test unit expired during the transfer preparation: the UI
correctly blocked controls on stale heartbeat. Restarting the same bounded
process, then dismissing the stale dialog, restored operation. This failure is
retained as a prerequisite test, not hidden. Both test processes were stopped at
the end. A later replay must start the updater/listener before opening an agent;
boot/service automation is outside this M0 experiment.

### SQL compatibility: remove the session-mode workaround

The previous connection-scoped SQL-mode filter has been removed. The isolated
dbconnect derivation is byte-identical to its reviewed pre-workaround input;
there is no `SET sql_mode`. Global, actual PHP connection before/after operations,
and an independent Perl DBI connection all retain:

`STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.

| Reproduced statement/value | MariaDB failure | Narrow correction |
|---|---|---|
| `vicidial_live_agents` INSERT with `lead_id=''`, agent PHP 5334/5413 | 1366, incorrect integer | Integer `0`, the existing no-lead meaning, in the two login INSERTs |
| `vicidial_manager values('','',...)`, 19 query-handler and 22 manager-handler INSERTs | 1366, incorrect integer primary key | `NULL` requests AUTO_INCREMENT allocation |
| Manager `callerid_name` exceeding varchar(20) | 1406, data too long | Fix r4037's literal-dot trimming expression with `substr($user_abb,-4)`; preserve the intended four-character user suffix |
| `manDiaLlogCaLL`, `alt_dial='MANUAL_DIALNOW'` into varchar(6) | 1406 | Translate that exact UI workflow label to existing dial type `MANUAL`, for both START and END |
| Lead update `date_of_birth=''`, query handler source 18691 | 1292, incorrect date | `NULLIF(escaped_value,'')` for the nullable date; real dates preserved |

`strict-sql-live-probe.json` independently reproduces 1366/1406/1292 on a temporary
InnoDB table with the actual PHP connection and accepts the corrected typed
values. Live fresh login, manual-call START/END, disposition and logout then
completed with no HTTP 500s. The original failures, including an intermediate
END request and DOB failure, remain in private evidence. These are version-pinned
operation corrections, not a general SQL framework or a relaxed database default.

A separate duplicate `call_log` key (1062) came from the local test fixture
logging both the agent conference entry and the customer Originate under the
same call identifier. The fixture now follows the existing conference-entry
contract without that extra agent-entry call-log invocation. Only the confirmed
duplicate fixture record was removed after preservation; accepted FastAGI code
and schemas were not rewritten.

### Calendar failure

Playwright reproduced `TypeError: Cannot read properties of undefined (reading
'length')` at `f_tcal_hideAll`, `calendar_db.js:306:40`, while scrolling the real
agent page. The scroll listener (323) is installed before any calendar necessarily
creates `window.A_TCALSIDX`; it is not malformed PHP data. A single pinned guard
returns only when that registry is absent. Existing instances still receive
`f_hide()`. The overlay preserves original CRLF bytes except for its addition.
The regression executes both absent-registry and existing-instance cases; final
browser logins and controls produced zero page exceptions. No global exception
suppression or calendar feature implementation was added.

### Logout and session authorization

`web_client_sessions` has `extension,server_ip,program,start_time,session_name` and
no expiry field. The original two rows belonged to failed/abandoned tabs; row
existence alone did not establish activity. Actual replay established a defect:
the old count-only check could authorize a retained row after the agent session
was gone. The three protected handlers now require the named web session to join
`vicidial_session_data` and the matching current `vicidial_live_agents` user,
server and conference, including the authenticated user match.

The agent's existing `$no_delete_sessions` setting is `0`, allowing its normal
logout path to release `vicidial_confbridges`. The current web-session row is
deleted by normal logout; abandoned records are retained. Final state contains
three abandoned web rows, zero live agents and zero occupied conference rows.
Original and abandoned credentials/session names were rejected by all three
handlers during fresh-session and post-logout tests (`revocation-all-endpoints-final.json`).
The last browser replay independently returned `Invalid session_name` from
`vdc_db_query.php`, `conf_exten_check.php` and `manager_send.php` after actual UI
LOGOUT (`controls-final-revocation.json`); its cookie list was empty. HTTP 200 is
the upstream error-response transport status, not authorization success.

Normal logout requires its manager queue items to be dispatched. The harness
uses the upstream sender with `--sendonlyone --nocounteroutput` for each verified
local request; its “PROCESS KILLED MANUALLY” exit message is the intentional
single-action boundary. Final queue count is zero. Both phone clients explicitly
hang up/unregister during browser cleanup; normal agent logout also issues the
conference release request. Final CLI confirms zero active channels, calls,
conferences and contacts. Fresh agent8/9/10/11 logins reused the conference without
duplicate live-agent or registration conflicts. Retained rows were not bulk-deleted.

### Actual application call controls and the local call fixture

The actual agent `ParkControl` invokes VICIdial's manager workflow, redirects the
selected customer channel to extension 8301, and runs upstream `park_CID.agi`.
Resume redirects that channel back to the agent conference. The Xfer control's
**Dial Blind Transfer** queues the actual customer-channel Redirect to selected
local conference 9600001. These are VICIdial application operations, not SIP hold
or a standalone AMI experiment. Only verified local queue items were dispatched.

The final customer leg must be represented as a real conference member. An early
fixture used `Dial(PJSIP/9052,30,o)` behind a Local leg, yielding working media but
an eventually correct “customer gone” warning: the PJSIP channel was in `Dial`,
not in the conference snapshots the agent searches. The corrected bounded
fixture uses installed Asterisk's documented `Dial(...,oG(alm7-agent^customerjoin^1))`
to put the answered PJSIP customer directly into the existing ConfBridge route.
No warning was suppressed. The agent conference entry stays in the generated
route. A retry also timed out because an old logout item was ahead of Originate;
the final run dispatches each inspected item before the UI timeout. Both failed
sequences are retained, with the successful final run identified below.

| Final UI operation | Actual state/evidence |
|---|---|
| Manual dial 99052; answer real 9052 phone | Manager 41; agent PJSIP/9051 and customer PJSIP/9052 directly in 9600000; UI selected the correct customer; no false warning after 37.9 seconds |
| Park | Manager 42; same customer enters upstream park AGI, matching `parked_channels` row and UI `customerparked=1`; agent stays in 9600000 |
| Resume | Manager 43; same customer returns to ConfBridge 9600000, parked count zero |
| Dial Blind Transfer to 9600001 | Manager 44; same customer moves to 9600001, agent remains 9600000; agent PAUSED, disposition `XFER`, no parked channel |
| Empty transfer destination (earlier control run) | UI rejects with minimum-one-digit alert; no manager request or call transition |
| Phone hangup, disposition SUBMIT, agent LOGOUT, unregister | Zero channels/calls/conferences/contacts, live agents, reservations and pending manager items |

VICIphone has no supported phone SIP hold or transfer button in this pinned UI;
its REFER handler is explicitly unimplemented. Those phone-level controls are
NOT_APPLICABLE, while application park/transfer above are PASS. Mute is separate.
Hangup and SIP INFO DTMF reception retain the accepted baseline evidence. A final
phone keypad click was exercised, but the final Asterisk log had no matching DTMF
event; this continuation does not claim an additional DTMF reception measurement.

### Unexpected WSS loss and supported recovery

The test killed one established server-side WSS socket using `ss -K` without
normal SIP logout and without changing TLS. Both idle and active-call cases were
executed. Transport transitions to `Disconnected` and the real UI immediately
shows **WS Disconnected**, although SIP.js Registerer can retain `Registered` for
the lost flow. Asterisk removes that contact. Active RTP did not immediately end:
the embedded-agent experiment observed continued connected ICE and packet-count
growth in both directions after the signaling interruption.

The existing Register button previously called REGISTER against a disconnected
transport and could remain “Registering”. Its pinned correction calls supported
SIP.js `UserAgent.reconnect()` only when disconnected, then the existing Registerer.
The REGISTER accept callback refreshes the existing UI even when the enum never
changes; active-call text is preserved. There is no second controller or retry
daemon. An unsuccessful reconnect reaches the existing failure display.

Recovery contract: restore network reachability, click **Register**, explicitly
hang up the old call if present, then place a new call. Remote BYE for an old
dialog may still target the previous WSS flow; seamless continuation of that
dialog is not claimed. `recovery-newcall-final-browser.json` captures unexpected
active disconnect → Connected/Incall after Register → old ICE closed on Hangup
→ a fresh Established call with connected ICE. CLI verifies one contact per
endpoint, new channels in 9600001, then zero resources after cleanup. Idle 9052
recovery was also followed by a successful new call. Embedded 9051 loss was
tested in the actual agent page; agent authorization remained separate from SIP
transport state. No credential re-entry or TLS bypass was necessary because the
authenticated page retains its in-memory Registerer configuration. Reopening a
closed page requires normal phone/agent authentication.

### Focused final media replay and credential protections

The final actual agent11 login, embedded 9051 registration, manual local customer
call to real 9052, two-way media, park/resume, blind transfer, disposition, hangup,
logout and resource cleanup passed. `controls-final-browser.json` records ICE/RTP
snapshots and UI state; `controls-final-media-analysis.json` records the existing
signal analyser results. Both browser clients use distinct synthetic microphones.

| Direction | Pattern | Correlation | Tone | Delay | RMS | Result |
|---|---:|---:|---:|---:|---:|---|
| 9051 → 9052 | 0.99209 | 0.99823 | 0.99782 | 60 ms | 0.14885 | PASS |
| 9052 → 9051 | 0.99209 | 0.98423 | 1.00000 | 40 ms | 0.15280 | PASS |

Thresholds remain pattern ≥0.90, correlation ≥0.85, tone ≥0.90, RMS ≥0.02 and delay
<2000 ms. No threshold was lowered. Both ICE/peer connections were connected.
The accepted recording/ALM-6 supervisor cases were not needlessly repeated.

The phone debug protections and escaped embedded POST remain. Real disposition
and logout also refreshed two upstream chat/email panels using credential-bearing
GET URLs; these exact navigation points now submit the existing parameters by
POST. Duplicate parameter order is preserved through `URLSearchParams` iteration.
The logout link setting is the existing plain-login value `0`. Apache retains
query-free access logs, `Referrer-Policy: no-referrer` and no-store behavior.

The final controls audit (`credential-audit-controls.json`) inspected 1,546 network
requests, two Referer headers, 22 console events, 14 history URLs and 161,642 bytes
of access logs: zero password leaks in those surfaces. The separate fresh-agent
and WSS audit also passed. Authenticated DOM and POST/SIP authentication material
remain sensitive private traces; protection does not mean passwords disappear
from the authenticated browser. Failed intermediate credential-bearing requests
are retained privately, not described as clean final runs.

### Missing external topology prerequisites and exact qualification procedure

Kaneo's canonical ALM-7 acceptance still requires every §10.3 P0 case. No operator
amendment defers them. Multipass host-to-guest/LAN success proves none of the
three missing profiles. No public service, paid infrastructure or carrier was
created. Current local deployment uses HTTPS 8443, direct WSS 8089 and UDP RTP
10000–10019, ICE enabled, strict RTP, DTLS-SRTP, `rtcp_mux`, symmetric RTP,
`force_rport` and contact rewriting. That narrow lab RTP range is not a production
capacity default. External settings remain unqualified.

| Profile / status | Required authorized layout and browser placement | Address/TLS/media requirements and missing prerequisite |
|---|---|---|
| Direct-public — NOT_RUN | This authorized target needs an explicitly approved directly attached public interface/address; independent browser on an outside network | Public FQDN resolves to that interface, trusted unexpired SAN certificate for HTTPS/WSS; route local endpoint media to that public address, ICE candidates reachable; supplied public address/routing and ingress approval absent |
| 1:1 NAT — NOT_RUN | This target behind an authorized fixed public↔private mapping; browser outside that NAT, avoiding hairpin-only proof | FQDN resolves to mapped public address; preserve/record TCP HTTPS/WSS and UDP RTP mapping; set/verify transport `local_net`, `external_signaling_address`, `external_media_address` and ICE mapping as required by actual SDP; controllable NAT gateway, mapping and authority absent |
| Remote-home-agent — NOT_RUN | Actual Chromium outside the server LAN, behind an authorized ordinary home router/NAT; server reachable via one qualified public layout | Trusted public hostname, outbound WSS and UDP media reachability from that home; inspect selected ICE pairs, NAT-translated RTP and DTLS. Supplied remote browser/router/network and public server path absent |

For each authorized profile, record the named target/gateway, approved source
networks, public/private addresses, DNS, certificate chain/SAN/expiry and firewall
mapping first. Permit only the selected HTTPS/WSS TCP ports and actual UDP RTP
range from the approved test sources; keep MariaDB, AMI, FastAGI and plain
Asterisk HTTP private. The current VM has no `firewall-cmd` executable; the
observed LAN access controls cannot be treated as an external firewall policy.
Do not open ingress without the topology-specific authorization.

Then run the existing browser harness from the stated external location with
certificate validation enabled: real agent login/embedded 9051; real 9052 phone;
REGISTER/contact uniqueness; local call and conference join; unchanged two-way
signal analysis; controlled recording for the direct-public case; park/transfer;
unexpected idle/active WSS loss and the supported recovery sequence; logout and
cleanup. Retain browser trace/WS events, SIP response summaries, selected ICE
candidate pairs and packet counters, DTLS/SRTP state, actual two-way recordings
and analyser output, Asterisk contacts/channels/conferences and database states.
Repeat registration and a new call after recovery. Every mandatory result must
be attributable to that actual layout. Symmetric/corporate NAT and IPv6 remain
unqualified P1 limitations; no TURN server was silently introduced.

### Reproducibility, preservation and validation

The existing asset overlay/harness/tests were extended, not replaced. All source
pins are declared in `scripts/lab/m0_07_assets.py`; unknown or ambiguous source
markers fail before output creation. `final-overlay/asset-manifest.json` records
the exact inputs and 59 deployed file hashes, SHA-256
`6510b421898c9b0c739e76167c7acbba4edfd245328702b581737221088f6325`.
The dbconnect input is explicitly an already-isolated M0-07 derivation, not
misidentified as pristine upstream.

| Source / derivation | Input SHA-256 | Deployed SHA-256 |
|---|---|---|
| `AST_manager_listen.pl` | `4111ce1d9714ee56c0eb243779bad94eb62f6ca8cfe23a3c229f661b08d587dd` | `504b3930b0315b425bf3a847e19d609afeae52d55c49d38a02b905148e44a865` |
| `AST_manager_send.pl` | `83be4e5ca25591adc6991273b84114b53164ef49b7bf04bee9e9b7413b7cb384` | `b411a4fa039e379179ca411712cdb6a303b93c31ed116a2024eb80bf6860e39f` |
| `AST_send_action_child.pl` | `46641e4f67c91122b520a932559e3b4f162609360cd4d15e38d9b7bf8bad15c9` | `ce731a06bb44344bc001101377676d87627937b8c99de66d9f127111d0f76436` |
| `AST_update.pl` | `ee0e5a752d47ababb0db4657a0825757b62fd41433bc4a04a4ed7ea968b8ef3b` | `2e976f7dffa2c5ad3f9cf1c707ad627ea57b925301428b4571ab4c1bcc468a6e` |
| `Asterisk.pm` | `26f266508ef3c9087bb19cad34b51c30a50853b60550e14626d2a7544c865915` | `26f266508ef3c9087bb19cad34b51c30a50853b60550e14626d2a7544c865915` |
| `Asterisk/AGI.pm` | `38f4e84f50e93e53d63b279584f5fca04553528c7660022d4b082aac30314c9a` | `38f4e84f50e93e53d63b279584f5fca04553528c7660022d4b082aac30314c9a` |
| `calendar_db.js` | `ee38674284705a761ee64a237efd9744095ef0e0d7da993cad3aed7f122ee7e7` | `5d85b5aa33071dc627a05d3a7938e484224c3909a9e2c860009526199b5a1cbc` |
| `conf_exten_check.php` | `1dcff537c57b61e2d8ea1bdc3770463fcc27a554982a39a444cea3fc3e628316` | `2cf23b3f696b2e3f9f00347f10e76d5e27f9b0b15849074f10be40817be98bad` |
| `dbconnect_mysqli.php` | `6cce68c7e7ecd919d7906476fb332e8978b7906224ae878b302c17de98d41a15` | `6cce68c7e7ecd919d7906476fb332e8978b7906224ae878b302c17de98d41a15` |
| `manager_send.php` | `8c53bbc25d595b0e0b982f06ad1eaf083ababb13ccb59d3678fa0f8252852a6a` | `417656d4c393b085f4b02a1f562cbf15fc6565fddd0b736be3afc4565785f32c` |
| `park_CID.agi` | `a41e746a0e5f87261f1510751321e7e4131703f2876dfb3351de99c2246e2b20` | `6aee1de6935f78f121b29906e50d737e6a701b975fe3e25af66a4940a0cc4f89` |
| `vdc_db_query.php` | `5aaeabd1d9b247dc563b9e94bb86d0b3323346b226b52d47a00ca2506897108d` | `c9c8de84495903b3716b4c61bc35a03dd78954c827d865529f10102226c277fd` |
| `vicidial.php` | `f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358` | `3698f50777cc36224b9de79cc4a9fdacfda72c0f707cfd351739dc6e70a9098c` |
| `viciphone/js/sip.js` | `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` | `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` |
| `viciphone/js/vici_phone.js` | `c7763fd3cc5a56d604c577291e6bbd25afe7f3b1cfa5fadbce914d30cf94779b` | `4fd3eff38aa505b43574cf23ae183b788f41bccee2e3e11423ab713992e5d6d0` |
| `viciphone/viciphone.php` | `e37d92df58d3c4f1b55a24a4364f9e4c9653c45820628fb657c42bd443fb6a6c` | `7c088f33c53e3ffdddd61c5c65d4d90282a359c48f699d35d15e7ee0328104ce` |

The only existing Asterisk configuration changes from continuation entry were
the task-owned `extensions-alm7.conf` fixture and a single include in
`manager.conf` for new isolated updater/listener/sender roles in
`manager-alm7.conf`. Existing role credentials/privileges were preserved. Those
roles bind through the existing localhost AMI listener; sender secrets use a
protected local file instead of child-process arguments/debug output. Fifteen
other baseline configuration/binary hashes match, including generated PJSIP,
ConfBridge, native dialplan, RTP and the Asterisk binary. Endpoint 9001 and its
`alm3-echo` Answer/Echo/Hangup route remain intact. No fresh native Echo media test
was required because its dependencies were unchanged.

`final-lifecycle-preservation.json` verifies all 59 deployed hashes, PHP lint,
five Perl syntax checks, zero call/session resources, live services, strict SQL,
MariaDB socket-only, localhost AMI/FastAGI/HTTP and SELinux Enforcing. The recovery
snapshot was read back again, never restored. Test processes are stopped; an
earlier failed transient updater unit remains as historical failed state, not
an active process. No full keepalive/autodialing suite ran.

All required repository commands passed: scaffold validator, **118 unittest
tests** (previous 107), candidate consistency, configuration fixture replay,
Python compilation, JavaScript syntax and `git diff --check`. Of those, 22 are
focused ALM-7 overlay tests. Offline tests validate source/data/state contracts;
they do not replace the live evidence above. Full command outputs and exit codes
are in `repository-validation.json`.

Task-owned repository paths changed: this report, `scripts/lab/m0_07_assets.py`,
`scripts/lab/m0_07_browser.js`, `scripts/tests/test_m0_07_assets.py` and ignored/
untracked task browser snippets under `.playwright-mcp/`. The original unrelated
review followup and `scripts/verify_agy_overlay.py` remain untouched. No commit,
push, reset, clean, restore or rebase occurred. The candidate lock was not changed.

The seal-time preservation scan also detected concurrent changes outside this task:
seven tracked files under `operations/portal443/` (`apply_requests.py`,
`common.py`, `guard_network_probe.py`, `sip_guard.py`, and tests for office,
portal and SIP guard), plus ignored `.idea/workspace.xml`. Codex did not edit or
revert them. Their entry/current hashes are recorded separately; they must not
be attributed to ALM-7 or claimed byte-identical to continuation entry. The
118-test result applies to the required `scripts/tests` suite, not those
concurrent portal changes. A later final status read-back also shows
`operations/portal443/register_observer.py` modified and two new untracked
portal scripts, `legacy-sip-deploy.sh` and `legacy-sip-rollback.sh`. These
ongoing operator changes were left untouched and recorded with the final
governance read-back, outside the already sealed ledger.

Continuation `evidence-ledger.json` contains **426 verified artifacts**
(158,379,530 bytes), SHA-256
`6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c`.
Browser profile caches and the later `mcp-final-readback.json` are excluded; the
read-back refers to this digest, avoiding a circular hash dependency. Private
traces, SQL/HTTP diagnostics and recordings remain outside Git.

The original ledger digest remains
`7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`;
all 255 original artifacts were reverified without changing them. Of 192 files
in the continuation entry inventory, 180 are unchanged, four are task-owned
changes, and eight are concurrent unrelated changes described above. The report
is finalized after sealing and is excluded from the evidence root.

### Kaneo disposition and next assignment

Canonical task `rwpendvaoqpi9mrzgskej6qb` remains In Progress / Codex. ALM-6 remains
Accepted; ALM-8 remains Backlog and unassigned. Native dependency IDs
`rapwb42p3l0v6r8t9ym7dkz5` and `sq76uz00g3qczpiwnamj1d26` remain intact.
The original evidence comment was retrieved, and no mandatory criterion was
amended. Continuation comment `rlg17bykh6nt8ya5rxn4b88y` was posted and its complete
content read back byte-for-byte through `kaneo-local`. Task states, Codex ownership
and both dependencies were re-read after posting. Protected
`mcp-final-readback.json` records that read-back separately from the sealed ledger.
ALM-7 was not moved to QA Review or Accepted.

Next work is to supply the authorized external topologies or obtain an explicit
project scope amendment; ALM-8 must not start. After all mandatory criteria are
satisfied, a separate Google Antigravity reviewer should reproduce real login,
media, supported controls, logout revocation, WSS loss/recovery, certificate
handling and accepted network profiles using the highest approved Gemini
Pro-class reasoning model, High effort, and the requested QA toolset. Codex does
not self-accept ALM-7.

## Historical implementation record — 2026-11-06 08:36 UTC

The remaining sections preserve the earlier accepted implementation-side record.
Their “final” hashes, PARTIAL verdict, SQL-mode workaround, allocated-room counts
and open lifecycle/control findings describe that earlier run only. Current
dispositions and hashes are in the continuation above.

## Baseline, authority and recovery

Starting and final Git HEAD: `c3e5593921c55e65667418cd446d2d75ba541083`.
ALM-6 was read back as Accepted before ALM-7 execution. The operator's assignment
authorizes this bounded experiment only on `almadial-alma10`; its local
`authorization.md` is an assignment record, not an externally issued approval ID.

Observed guest: AlmaLinux 10.2 x86_64; Multipass 1.17.0 / QEMU/KVM; 4 vCPU,
8 GiB RAM. Current virtual disk inventory is approximately 200 GiB, rather than
the historical 80 GiB allocation. Asterisk 22.12.0 / PJPROJECT 2.17, VICIdial r4037 /
schema 1748, MariaDB 10.11.18, PHP 8.3.36, Perl 5.40.2 and Apache 2.4.63 were reused.
The working Asterisk and VICIdial installations were not reinstalled.

The idle VM was stopped and snapshotted as `pre-alm7-viciphone-integration` at
2026-11-06T07:24:44Z, then started. Snapshot parent: `pre-alm6-telephony-runtime`.
Read-back verified the new snapshot and preserved prior snapshots. Required
services were explicitly started afterward; this is not boot-persistence evidence.
No snapshot was restored. Future authorized recovery must first preserve current
evidence, stop this VM, restore the named snapshot and verify guest health.

Protected evidence root: `~/almadial-lab/evidence/m0-07/20261106T072425Z/`.
Command records contain argv, UTC, exit status, stdout and stderr. Credentials,
browser traces/cookies, SDP, recordings, private configuration and original failure
records remain outside Git. Evidence directories are private; exported repository
content contains only sanitized observations and digests.

## Source identity, assets and licensing

| Input | Identity / SHA-256 |
|---|---|
| VICIphone | 3.0, commit `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` |
| Qualified source archive | `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` |
| Embedded SIP.js | 0.20.1; `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` |
| Original VICIphone PHP | `e37d92df58d3c4f1b55a24a4364f9e4c9653c45820628fb657c42bd443fb6a6c` |
| Original phone JavaScript | `c7763fd3cc5a56d604c577291e6bbd25afe7f3b1cfa5fadbce914d30cf94779b` |
| Original r4037 agent PHP | `f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358` |
| VICIphone repository license | `e7ae8b5da17709474ba25316441bc064148841cf101e7eef318c3c3ee5a0827b` |

Source provenance inherits the accepted M0-02/04 decisions; a digest alone is not
publisher authentication. Upstream files were preserved. VICIphone's AGPLv3
license and embedded dependency notices, including SIP.js MIT notices, remain in
the deployment. No upstream distribution was copied into the AlmaDial repository.
Modified deployed copies and their reproducible overlay remain distinct from
AlmaDial's original installer code; redistribution must retain the applicable
source and notice obligations.

The local deployment contains 45 source assets plus the repository-root LICENSE.
All 46 deployed phone assets match the final replay. Required JS, CSS, images and
phone assets load from the local HTTPS origin; the recorded working browser
sessions require no external CDN. Standalone VICIphone sessions had no fatal
JavaScript page errors. Negative-test directory/favicon HTTP errors are retained
and are not classified as certificate or phone-media results.

## Deployment and minimal compatibility overlays

Apache serves an isolated `/var/www/html/m007` document root. The actual phone URL
is `https://almadial-alma10.test:8443/viciphone/viciphone.php`; the document-root
name is not an extra URL prefix. Direct Asterisk WSS uses port 8089. The existing
lab CA is trusted by the test Chromium profile; HTTPS and WSS identity validation
remain enabled. The observed certificate expires 2026-11-07T18:26:13Z and must be
renewed before later reproduction. Its SAN includes the lab hostname and guest IP.

Only `mod_ssl` (epoch 1, 2.4.63-13.el10_2.8) and dependency `sscg` (4.0.3-2.el10)
were installed. The default SSL virtual host was backed up and disabled in favor
of the isolated lab host. Apache configuration uses TLS 1.2/1.3, query-free access
logging, no-referrer/no-store headers and lab-subnet access restrictions. Web files
are root-owned with Apache read access; the webroot is not broadly writable.

[`m0_07_assets.py`](../../scripts/lab/m0_07_assets.py) validates exact source pins,
refuses symlink inputs/existing destinations, and writes copies into a fresh 0700
directory. Its offline manifest explicitly reports `live_runtime: NOT_RUN`.
The deployed corrections are:

1. Disable VICIphone access debug and SIP.js configuration/SIP tracing to avoid
   exporting browser credentials through routine diagnostics.
2. Replace r4037's credential-bearing embedded-phone URL with an escaped form POST
   through the two existing iframe integration points. The stored web-client URL
   is the base URL, without SIP-password query fields. This does not hide secrets
   from the authenticated browser; private traces still contain session material.
3. For the isolated ALM-7 database connection only, remove strict SQL modes to
   reproduce legacy empty-integer inserts. Global MariaDB SQL mode and accepted
   ALM-6 connections remain unchanged. This is a lab workaround, not approval of
   a production-wide non-strict SQL default.

The first real agent attempt failed because DELETE on `web_client_sessions` was
not granted. A dedicated local account was granted CRUD only on `almadial_m005`.
Subsequent empty-integer insert failures reproduced under strict SQL mode. An
intermediate attempt requiring `sys.list_drop` failed EXECUTE authorization; the
final PHP connection-mode filter avoids granting that privilege. Original failures
and intermediate copies are preserved, rather than rewritten as successful runs.

Final deployed hashes:

| File | SHA-256 |
|---|---|
| Phone PHP | `7c088f33c53e3ffdddd61c5c65d4d90282a359c48f699d35d15e7ee0328104ce` |
| Phone JavaScript | `6f2dc1a6a6a5c31de865b4c982178b6bbcbc23a643923e82cb420879fb02689d` |
| Agent PHP overlay | `0da29999505dab3e38136eaaa03223cb822e3e125ce0dfe323f7ba72b2e8764e` |
| Isolated dbconnect overlay | `f8a4d35ac6833ea652d2f966453363f05fda0f30d33c53fd7d864708aab63f16` |
| Complete replay manifest | `42c35f3423742dfb9fa2c1dfd4000a65dd44b1d011e2953ba7a22760f7d8738d` |

## Actual VICIdial agent and phone workflow

The browser used a real test-agent login, separate phone-login authentication,
test user group and MANUAL campaign with auto-dial level 0 and no hopper. No
database-seeded browser session was substituted for authentication. The agent
interface embedded the locally hosted VICIphone; it registered generated endpoint
9051 and attached to allocated room 9600000. A second actual VICIphone on generated
9052 joined the same room. Database and live Asterisk observations correlated the
agent identity, PAUSED state, phone and conference. Room allocation was read from
`vicidial_confbridges`; both 9600000 and 9600001 remain allocated.

The assigned agent room lacked a local entry route. The task-owned
`extensions-alm7.conf`, included once by the base dialplan, adds that local route,
calls the accepted FastAGI call-log entry, and enters the existing generated
conference route. It does not replace generated outputs or native endpoint 9001.
Its SHA-256 is `5a925724231913cac67b280f1fd9655269c2d85045a037dbc252d6708d1d161c`.

| Agent stage | Actual outcome |
|---|---|
| Web-agent authentication | PASS through the application UI |
| Phone-login authentication | PASS; distinct from SIP password |
| Embedded VICIphone registration | PASS, correct generated endpoint |
| Phone attachment / conference media | PASS, actual room 9600000 |
| Complete agent health | FAIL/open: “No one is in your session” and time-state warning |
| Logout | UI reports completion; live-agent row removed |
| Complete application session cleanup | Unresolved: two `web_client_sessions` rows remain |

The warning is not proof of an unsynchronized host clock. Source inspection of
`conf_exten_check.php` shows that it reads `server_updater.last_update/db_time` and
checks server/database/web time differences. The existing test database has **zero
`server_updater` rows**. The required bounded updater/agent-conference contract is
not yet established. No fake heartbeat was inserted and no unbounded dialer was
started to conceal this gap.

The agent page also raises `TypeError: Cannot read properties of undefined
(reading 'length')` in `calendar_db.js:306`, where `f_tcal_hideAll` reads
`window.A_TCALSIDX.length`. The protected trace preserves this separate legacy UI
failure. It was not silently attributed to VICIphone or patched without a scoped
reproduction. Full user-facing agent health and control acceptance remain open.

## Measured browser media

Chromium 157.0.8103.38 used trusted HTTPS/WSS, SIP.js 0.20.1 and two independent deterministic
fake microphones. Signals contain identifiable sequences at 697, 941, 1209 and
1633 Hz. Received tracks were captured independently through the accepted paired
AudioWorklet. Local source files were not compared to themselves as remote audio.
Physical microphone/speaker hardware was NOT_RUN; synthetic audio traversed the
actual browser WebRTC, Asterisk and ConfBridge paths.

Each 12-second test required actual two-channel conference membership plus
connected ICE/DTLS, SDP, RTP and remote-waveform evidence. PCMU at 8 kHz was
negotiated; the final replay reports SRTP_AES128_CM_HMAC_SHA1_80. Inherited thresholds
were unchanged: pattern agreement >=0.90, envelope correlation >=0.85, tone-label
agreement >=0.90, remote RMS >=0.02 and estimated delay <2000 ms. Delay includes
independent capture-start skew and is not a precise network-latency measurement.

| Actual client/test | Direction | Pattern | Envelope correlation | Tone agreement | Delay | Result |
|---|---|---:|---:|---:|---:|---|
| Agent-embedded VICIphone / second VICIphone, room 9600000 | 9051 → 9052 | 0.992095 | 0.980645 | 1.000000 | 40 ms | PASS |
| Same agent session | 9052 → 9051 | 1.000000 | 0.987930 | 1.000000 | 40 ms | PASS |
| Two VICIphones after Asterisk restart, room 9600001 | 9051 → 9052 | 0.998016 | 0.994692 | 1.000000 | 60 ms | PASS |
| Same restart repeat | 9052 → 9051 | 0.994083 | 0.982528 | 0.990971 | 60 ms | PASS |
| Final portable-helper replay, room 9600001 | 9051 → 9052 | 0.994071 | 0.986833 | 0.993228 | 40 ms | PASS |
| Same portable replay | 9052 → 9051 | 0.992095 | 0.978698 | 0.964912 | 60 ms | PASS |

In the final replay, received RTP packet counts were 1357 and 1356 respectively,
with zero reported packet loss and zero jitter in the final snapshots. Captures
retain unequal independent frame counts (604 and 602); they were not truncated or
padded to manufacture agreement. The analyzer itself reports offline runtime
NOT_RUN; the runtime conclusions above join it to browser and CLI evidence.

## Controls, reconnect and recording

| Case | Result and boundary |
|---|---|
| VICIphone hangup/unregister | PASS; channels, conferences and contacts cleared |
| VICIphone DTMF delivery | PASS after loading `res_pjsip_dtmf_info.so`; a fresh call delivered digit 6, SIP 200 and Asterisk DTMF end, 250 ms |
| Initial SIP INFO DTMF | FAIL/408; preserved before module correction |
| Conference-menu action | NOT_RUN beyond proven DTMF reception |
| Mute | Track-state behavior only; not park or hold |
| Agent application park/transfer | NOT_RUN; complete agent health/control path remains open |
| Asterisk restart / new two-party call | PASS; modules, generated endpoints and media recovered |
| Explicit WSS disconnect / reload / reauthenticate / new call | PASS with observable disconnected state and successful new media |
| Unexpected network loss / automatic reconnect | NOT_RUN; Playwright offline toggle did not close the existing WSS socket |
| Interrupted live-call media restoration | NOT_RUN; reload/new-call success is not continuity |

The DTMF module is explicitly loaded in `modules.conf`, with a protected backup.
The first CLI check after restart raced startup and failed; later healthy runtime
and successful media are the relevant restart evidence. No failure was discarded.

AMI `ConfbridgeStartRecord` / `ConfbridgeStopRecord` recorded the two-VICIphone
conference. The resulting WAV is PCM16 mono at 8 kHz, 274,924 bytes and 17.18 s;
FFprobe decoded it and signal analysis found both source patterns with agreement
1.0. Final inspection found no remaining open recording descriptor. This proves
preserved native AMI conference recording, **not** VICIdial recording-UI operation
or recording-database metadata acceptance.

## TLS and topology results

| Case | Observation |
|---|---|
| Trusted HTTPS and direct WSS | PASS; real registration and media without bypass |
| Untrusted HTTPS authority | Rejected: `ERR_CERT_AUTHORITY_INVALID` |
| Expired HTTPS | Rejected: `ERR_CERT_DATE_INVALID` |
| Expired WSS | Rejected before registration: `ERR_CERT_DATE_INVALID` |
| Wrong HTTPS hostname | Rejected: `ERR_CERT_COMMON_NAME_INVALID` |
| Wrong WSS hostname | Rejected before registration: `ERR_CERT_COMMON_NAME_INVALID` |
| Wrong SIP password | Transport connected, registration rejected |
| Desktop → Multipass LAN path | Actual registration and bidirectional media PASS |
| Direct-public, 1:1 NAT, remote-home-agent | NOT_RUN: no authorized public endpoint/topology supplied |

The first proposed hostname-negative test used the guest IP, which is a valid
certificate SAN; it correctly registered and is not rejection evidence. The
corrected test used a hostname outside the SAN. Initial expired-certificate test
hosting on 8444 was denied by SELinux and interrupted Apache's graceful reload;
the isolated host was disabled and Apache restored immediately. Tests then used
already-permitted port 443, with correctly dated/SAN test certificates. The
temporary 443 host was removed after testing. SELinux remained Enforcing; no
certificate bypass, SELinux disablement or port-policy weakening was used.

Multipass NAT is not a public/remote-home-agent qualification. Missing mandatory
profiles prevent canonical ALM-7 acceptance unless the operator explicitly changes
that acceptance contract. No public exposure or TURN service was invented.

## Security, preservation and cleanup

Asterisk's executable SHA-256 remains
`db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c`.
All accepted generated PJSIP/dialplan/ConfBridge outputs and native HTTP/RTP/PJSIP
base files were unchanged. Only the task-owned agent route, its base include and
the DTMF module load changed Asterisk configuration. Native endpoint 9001 and
its Answer/Echo/Hangup dialplan remain present; a fresh native 9001 audio regression
was NOT_RUN during ALM-7. ALM-7 media exercised the shared WSS/PJSIP path directly.

Final checks show zero active channels/calls, no active conference or contacts,
SELinux Enforcing and healthy Asterisk/Apache/PHP-FPM/MariaDB/FastAGI services.
MariaDB remains socket-only (`skip_networking=ON`), AMI remains 127.0.0.1:5038 and
FastAGI 127.0.0.1:4577. No carrier/trunk or external dialing was introduced.
Two residual web-client database sessions remain an explicit cleanup investigation;
telephony cleanup does not prove complete application cleanup.

## Reproduction and validation

Use only the authorized VM, current address and an unexpired trusted lab certificate.
Do not blindly replay historical failed provisioning/overlay scripts.

1. Read protected `authorization.md`, source records, input copies and manifests.
2. Reproduce copies without deployment using
   `python3 scripts/lab/m0_07_assets.py SOURCE/src FRESH_OUTPUT --agent AGENT.php --dbconnect LAB_DB.php`.
   Inputs must match the exact pins; retain original source and license.
3. Use [`m0_07_browser.js`](../../scripts/lab/m0_07_browser.js) through Playwright MCP,
   configuring `page.alm7Config` with private evidence paths, accepted ALM-3 browser
   HOME, repository path, current VM address, unique run ID and explicit phase.
   Execute `start`, observe two channels/room in Asterisk, execute `capture`, then
   always `finish`. Private phone POST files and signal fixtures are prerequisites.
4. Analyze cross-participant local/remote WAV pairs with the accepted ALM-6 signal
   analyzer, preserving its thresholds, and correlate with browser SDP/stats and
   actual Asterisk conference state. Verify cleanup independently afterward.
   This portable helper reproduces phone media, not full agent login or reconnect.
5. Agent-specific raw procedure and traces are protected evidence; reproduce real
   UI authentication using test-only credentials, never a seeded session.

Executed repository checks (all exit 0):

```bash
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
python3 -m py_compile scripts/lab/m0_07_assets.py scripts/tests/test_m0_07_assets.py
node --check scripts/lab/m0_07_browser.js
git diff --check
```

**107 tests passed**, including 11 new overlay tests. Scaffold validation passed.
Candidate consistency passed with package/source/runtime checks explicitly NOT_RUN
in local mode. Generated-configuration verifier passed its redacted fixture replay,
not live-runtime acceptance. Deployed phone/agent/dbconnect PHP syntax checks,
phone JavaScript/SIP.js syntax checks and all 46 deployed asset hashes passed.
The portable browser helper was actually executed (`tooling-replay`), rather than
only syntax-checked. No Go installer code was introduced.

## Evidence index and handoff

Selected protected evidence identities:

| Artifact | SHA-256 |
|---|---|
| Agent session two-party CLI observation | `b55d4f768508d0fabd8f6e0ffa9a0313328875f390b6dd65c00be81d79338433` |
| Portable replay browser results | `5cbb1150db32126591ccb5f9f3c1a1b0ae5f592db358985362199ff546310169` |
| Portable replay signal analysis | `37c4976f77d7a8754676146bcc834817556a8e0c4e7e9eb3a8147e7e02aa3ac3` |
| Portable replay live CLI correlation | `1baf8fa34876ad3610424aa8cbfa7f84b87d5c5b09655cec82ed3360429802a0` |
| Portable replay final cleanup | `0ec203c9cca187dc71abfc560a996d3813d148312fb2b283874656fcf1738a5f` |
| Conference recording WAV | `4cfc13bfbbe58007a10204ffd020014c71ed60f0036041d0834047600dfafdfc` |
| Recording content analysis | `71c809f1c335e07b8fdb5f5b0de581f527d376aaa27e6e5af5ba3e1ac5c3b605` |
| Final allocated-room/session query | `314132e7bdc9148e49ecfefe4a6910c69f800ce606c12f9ed77814574d2882ff` |

The final evidence ledger and source-provenance snapshot identify the uncommitted
task-owned file bytes. Browser profiles/caches, passwords and keys are excluded
from the ledger; sensitive traces remain private even when their digests are listed.
Earlier manifests and failed attempts remain preserved alongside final records.

Sealed `evidence-ledger.json`: 255 entries, SHA-256
`7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`.
`source-provenance-final.json` records the three uncommitted tooling files and
confirms all 210 pre-existing files remain byte-identical. The report is excluded
from that source snapshot to avoid cyclic report/ledger hashes. Subsequent MCP
read-back is retained separately as `mcp-final-readback.json`.

Kaneo implementation evidence comment: `uq74ayopwguhauf8hdpf4wl8`, created
2026-11-06T08:36:49.655Z. A separate comments read-back exactly matched the submitted
content. Task, ownership, both native dependency relations and unstarted ALM-8
were read back afterward. No QA Review or acceptance transition was made.

ALM-7 remains assigned to Codex and In Progress. ALM-6 → ALM-7 → ALM-8 relations
remain unchanged; ALM-8 remains unassigned and unstarted. No acceptance transition,
commit or push is authorized by this handoff. The candidate manifest is unchanged.
Pre-existing operator/reviewer files are preserved byte-for-byte.

Next bounded work remains within ALM-7: establish the minimal authorized updater /
agent-conference heartbeat contract, reproduce healthy agent state and session
cleanup, close the calendar UI failure if it affects required controls, then test
actual application park/transfer and unexpected WSS reconnect. Separately obtain
authorized direct-public, 1:1 NAT and remote-home-agent environments for the
remaining canonical P0 cases. Independent AGY acceptance follows only after those
mandatory results exist; do not start ALM-8.
