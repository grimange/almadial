# PRD-03C — Preview lead recovery and direct logout correction

## 1. Executive summary

**Direct logout remains unresolved within the authorized configuration boundary.** Installed SVN r4037 has no identified campaign/agent option that makes LOGOUT await preview release. Its optional stuck-preview recovery does change eligible orphaned INCALL/ERI records to **NEW / called N**, rather than ERI, but does not restore the original status or counters. Isolated execution also reproduced missing active-state guards and a check/update race. It was not run against the live database or enabled in cron.

A concrete [two-file upstream patch proposal](proposals/prd-03c/preview-logout.patch) and [review/qualification instructions](proposals/prd-03c/README.md) are complete and **undeployed**. It sequences a bounded native preview-release acknowledgement before normal logout, guards ownership/calls/retries, and proposes restoring the eligibility flag and counters for the demonstrated list-preview case. Separate approval is required before changing native PHP/JavaScript. Local tests do not establish MariaDB locking correctness or corrected native browser acceptance.

One fresh native Chromium/VICIphone control on lead **56** preserved SKIP → LOGOUT behavior, count zero and clean sessions. It retained called Y, as legacy SKIP does; immediate next-lead eligibility is not claimed. No customer call, recovery command, direct SQL write, package installation, schema/source/cron change or infrastructure change was made. The final fixture is inactive with no live agent/channel or synthetic INCALL claim. This report is implementing-worker engineering acceptance, not independent QA.

Final verdict: **`VICIDIAL_PRD_03C_PARTIAL`**.

## 2. Baseline and authorization

Reviewed the [PRD-03B report](VICIDIAL_PRD_03B_PREVIEW_LOGOUT_ACCEPTANCE.md), [functional-gap acceptance](VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md), [fixture inventory](VICIDIAL_OUTBOUND_TEST_FIXTURE.md), [WebRTC acceptance](VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md), [webphone instructions](../webphone/README.md), product acceptance plan and relevant repository governance. No AlmaDial installer stage or stack-lock promotion is part of this task.

Target: dialer01, SSH TCP 22222, AlmaLinux 10.2, VICIdial 2.14b0.5 / retained SVN **4037**, Asterisk 20.21.0 and MariaDB 10.11. Reused VICIphone 3.0.0, chan_sip extension **1002**, conference **9600001**, and actual Chromium **149.0.7827.55**. Agent qaagent14 / group QAPRD14; supervisor qasup14 was preserved. No phone/template or access architecture changed.

Git began at `39dcfb4` with a clean tracked worktree. Unrelated untracked files and evidence were preserved. Unrelated changes to `internal/execution/runner.go` and new installer-work files appeared during execution and were also left untouched and excluded from this delivery. SSH was noninteractive with `-F /dev/null -T`; root history, shell initialization and SSH settings were untouched. Existing private credential files were used without including their contents in artifacts. Execution occurred **2026-11-09 UTC / 2026-11-10 Asia/Manila**; server DATETIME values are America/New_York (UTC−5 on this date). No timezone or historical timestamp change was made.

[Before state](evidence/preview-logout-03c/before.json) at 20:28:36 UTC established inactive QAOUT01/MANUAL/0, all four lists inactive, no INCALL/live agent/channel, and unused synthetic leads 56–68. Current installed source hashes matched the prior defect evidence. Thus another broad direct-logout/reconnect reproduction was unnecessary and would consume more leads without testing a correction.

Backup health was checked before/around bounded testing. The verified successful backup was `20261109T084003-23e8842a`, completed 08:40:17 UTC, approximately 12 hours old at the saved health check, outcome ok with five payloads and 376 database tables. The health document retains an older last_failure alongside current success; it is not a current failure. No new snapshot, backup schedule change or restore was performed. See [regression evidence](evidence/preview-logout-03c/regression.json).

## 3. Current native preview/logout behavior

Pinned source identities are in [recovery-source evidence](evidence/preview-logout-03c/native-recovery-source.json), [regression hashes](evidence/preview-logout-03c/regression.json) and the proposal README. Relevant installed locations, before any proposed changes:

| Source | Native transition |
|---|---|
| `agc/vdc_db_query.php:4783`, `4855` | Claim changes the lead to INCALL and increments called flag/count before the preview branch. Preview sets `vicidial_live_agents.preview_lead_id` without originating a customer call. |
| `agc/vdc_db_query.php:6565` | `manDiaLskip` clears the pointer, restores the previous status, decrements the lead count and writes a skip audit. It leaves called_since_last_reset Y and does not reverse the phone-number daily counter. |
| `agc/vicidial.php:13602` | `ManualDialSkip` clears frontend preview state before its asynchronous result. Its existing acknowledgement handling is insufficient for sequencing a safe logout. |
| `agc/vicidial.php:19549`, `19567` | NormalLogout/LogouT does not invoke preview release. Live call/dial guards are not an outstanding-preview guard. |
| `agc/vdc_db_query.php:18977`, `19115` | userLOGout closes/logs the session, requests conference hangup and removes the live-agent row without releasing its preview. |
| `agc/vicidial.php:4971` | Fresh owning-agent login converts remaining QUEUE/INCALL records to ERI and clears owner, without restoring status/count/called flag. |
| `agc/vdc_db_query.php:4377` | Normal manual next-lead selection requires called N plus allowed dial status and other campaign/list filters. A status label alone does not prove eligibility. |

The previously observed direct-logout defect remains applicable because these installed files are byte-identical. PRD-03B lead 52 remained INCALL approximately 136 seconds after logout; lead 54 reproduced the omission. Owning-agent reconnect converted them to ERI / count 1 / called Y, outside QAOUT01's dial statuses. Both were recovered to NI through native Lead Details in that assignment. This is **retained runtime evidence**, not a claim that those direct/reconnect scenarios were newly executed here. The visible LOGOUT control permits the problematic sequence; no basis was found to dismiss it as an unsupported operator action.

## 4. Native recovery mechanism qualification

Actual installed `/usr/share/astguiclient/AST_flush_DBqueue.pl`, SHA-256 `f3e4fe7762078cdc4578bddb759afa05e6ba1d0b4c6a554d4da88248860b678b`, lines 783–925, implements the following:

| Question | Established behavior |
|---|---|
| Enable option | `--reset-stuck-leads`; not enabled in the retained hourly `-q` invocation. |
| Scope | Optional `--stuck-lists=910016-910017` builds a list-ID filter for this branch. No campaign/user/individual-lead filter. Other maintenance operations in the same script are not restricted by this list option. |
| Candidates | ERI or INCALL; modify_date between now minus six hours and now minus **121 seconds**, ordered by modify_date, limit 10,000. `--seconds` does not change these fixed bounds. |
| Exclusions | Existing `vicidial_auto_calls` row; today's `vicidial_log`, `vicidial_closer_log` or `vicidial_dial_log`; a matching live-agent **preview_lead_id** whose last_state_change is since local midnight. |
| Missing guards | No check of live-agent lead_id alone, channels or pending origination queue; live preview guard is date-limited. Checks and final update are separate; no final status/ownership/active-state compare-and-set. |
| Exact recovery write | `UPDATE vicidial_list SET status='NEW',called_since_last_reset='N' WHERE lead_id=...`. Only these two columns are explicitly set; automatic modify_date behavior remains database-dependent. No other table is updated by this recovery branch. |
| Original state / counters | Original status is not recovered. User, called_count, last_local_call_time and daily counters are not restored. It can make a NEW-enabled lead selectable by status/flag, while retaining a fictional preview increment. Actual live next-lead retrieval after this command was NOT_RUN. |
| Repeat safety | Sequential repeat does not select an already-NEW row. This is not concurrent idempotence: the unconditional final update can overwrite a state changed after the last guard. |
| Timing | Current cron runs minute 11 hourly. If enabled, an otherwise eligible orphan could wait about 2–62 minutes; records older than six hours can be missed. It cannot supply immediate logout acknowledgement. |
| Test mode | `--test` / `-t` suppresses this branch's UPDATE, but the whole maintenance script and its logging are not a guaranteed read-only dry run. It was not invoked live. |

The exact unaltered Perl branch was executed through an isolated in-memory SQLite DB bridge by [qualify_preview_recovery.py](../../tests/webphone/qualify_preview_recovery.py). No package, database server, new schema on dialer01 or live maintenance invocation was required. [Results](evidence/preview-logout-03c/native-recovery-isolated.json) contain actual branch queries and 15 assertions:

- Orphan INCALL and ERI became NEW / called N, retaining count 1 and owner.
- Existing auto-call, today's outbound/inbound/dial logs and today's matching preview blocked recovery.
- Age boundaries, other-list filtering, NI exclusion and sequential repeat behaved as source specifies.
- A matching active preview with last_state_change before midnight was reset. A live-agent current-lead reference without a preview pointer was also reset when other guards were absent.
- Injecting an active-call/new status transition after the last guard reproduced an unconditional overwrite to NEW. The exact native update lacks a predicate to reject that race.

**All 15 assertion passes mean the harness reproduced the native semantics, including unsafe outcomes. They are not 15 safety passes.** SQLite is not MariaDB/MyISAM locking qualification, and the full maintenance script was not executed. The observed semantic failures are sufficient to reject enabling it as the primary correction. It neither changes or preserves ERI as its final recovery status nor restores full original preview state; its forced NEW result answers the prior uncertainty but does not satisfy this mission.

## 5. Correction decision and proposed diff

**Option A, native configuration:** no sufficient setting was identified in the installed preview/skip/logout branches or reviewed campaign/agent settings. PREVIEW_AND_SKIP exposes release but does not sequence it during LOGOUT. Changing dial method or adding ERI to dial statuses would not repair counts, called flags or ownership and was not done.

**Option C, scheduled recovery:** rejected for the reasons above. No cron option was enabled and no recovery command was run on the live database.

**Option B, release before logout:** selected as the proposed source correction. The assignment says, “Do not modify upstream VICIdial source or database schema without separate explicit approval.” Therefore the [patch](proposals/prd-03c/preview-logout.patch) is a reviewable artifact only. It reuses the native skip action/audit and prior-status/count convention, adds a bounded acknowledged format, and invokes ordinary logout only after release. Simply calling existing ManualDialSkip would retain called Y and its weak asynchronous failure behavior.

The proposal changes only `agc/vicidial.php` and `agc/vdc_db_query.php`; no table, service, dependency or recovery daemon. It guards active call/dial state, coalesces repeated clicks, preserves UI/phone on release failure, serializes claim/dial/skip/logout per native user, rechecks the queued session and guards ownership/current claim. A single guarded multi-table statement is proposed for MyISAM rather than a fictitious rollback transaction. It restores the previous status/count, called N and daily counters for the ordinary N→Y MAIN list-preview case. An exact-state retry handles a lost ACK without a second decrement. No call or disposition is invented.

This is deliberately bounded. Callback/manual-entry/alternate-number and previously-called flag histories are not qualified. Server concurrency with writers outside the affected native actions is not proven. A [read-only engine query](evidence/preview-logout-03c/table-engines.json) confirms the lead/live-agent/daily-counter tables use MyISAM; web_client_sessions uses MEMORY. The skip audit is a separate write: failure after release is reported, but retry can acknowledge the already-released state without reconstructing that audit. Cached old agent pages/API clients need a refresh during any approved deployment. These limits and precise pins, licensing, upgrade/rebase and deployment/reversal steps are in the [proposal README](proposals/prd-03c/README.md). A successful local harness does not remove those review points.

## 6. Native control and before/after correlation

Native Administration temporarily activated only QAOUT01 and list 910016, retained MANUAL/0/PREVIEW_AND_SKIP/ONDEMAND, and used no-hopper logins Y / local call time 24hours for the bounded test. [Native activation](evidence/preview-logout-03c/native-activation.json) records persistence. No new object, lead, account or phone was created.

The existing runner operated actual Chromium and VICIphone; [browser output](evidence/preview-logout-03c/skip-control.json) and [post-control read-only snapshot](evidence/preview-logout-03c/after-control.json) agree:

| Event | UTC / canonical evidence | Lead/session result |
|---|---|---|
| qaagent14 login | 20:35:39; epoch 1794256539 | Phone 1002 registered; established conference 9600001, connected ICE/DTLS, PCMU. |
| Native list preview | 20:35:46; recent-selection row | Lead 56 / GAP01B_008 selected, no DIAL LEAD/customer call. |
| Visible SKIP LEAD | 20:35:47; native skip log NEW / previous count 0 | Display cleared; backend lead NEW / count 0 / called Y. |
| Unregister and native LOGOUT | 20:35:47; epoch 1794256547 | Browser completed logout; subsequent state zero live agents/channels/conferences. |

Agent log 37 correlates lead 56 with eight seconds pause and zero wait/talk/disposition; row 38 is the zero-second post-skip row. The M recent-session record is selection history, not an invented completed call. There are zero outbound records for diagnostic leads 51–68, campaign outbound count remains two and recording count remains four. Legacy SKIP preserves count correctness but leaves called Y: this control **does not** prove reselectability or satisfy the direct-logout requirement.

No direct-logout, duplicate corrected logout or new reconnect claim was deliberately created on the unchanged host. Historical deterministic reproductions are retained; corrected browser scenarios are NOT_RUN pending approval. No active-call test call was originated. Active-call protection is source/isolated analysis only.

## 7. Proposed regression verification

[proposal-verification.json](evidence/preview-logout-03c/proposal-verification.json) records:

- Exact pinned-source verification, zero-fuzz patch application and reversal restoring both original SHA-256 values byte for byte.
- Both patched PHP files linted with local PHP **8.4.27**, not host PHP 8.3.
- **16** Node VM cases executing the actual proposed JavaScript helper and native LogouT with fake XHR/DOM: acknowledgement order, duplicate-click guard, four unsafe-call/preview guards, denial/malformed responses, timeout/error/abort/retry/late ACK, send failure, ordinary logout and failed native logout retaining the phone.
- **15** cases executing the exact generated SQL admission JOIN/WHERE in SQLite: one intended preview admitted and 14 unsafe/mismatched states rejected. Actual MariaDB multi-table UPDATE, named locks and counter effects are NOT_RUN.
- Actual proposed PHP release block executed with stubbed database functions, including matching/mismatching already-released retry control flow. This is not database-write acceptance.

Existing browser helpers now offer `corrected` and `repeated` preview cases with an explicit `WEBPHONE_PREVIEW_CORRECTION_TEST=YES` gate. They reject a page without the proposed helper **before preview selection**, click visible LOGOUT without pre-unregistering, check exact release ACK before logout ACK and verify the webphone iframe is cleared. Their output requires separate read-only lead/session/SIP correlation. These modes were syntax-checked, not run against a patched native host. The existing SKIP path was actually executed before the additive corrected-mode extension; it was not needlessly repeated afterward.

Commands executed for local qualification:

```bash
python3 tests/webphone/qualify_preview_recovery.py /tmp/prd03c-source/AST_flush_DBqueue.pl \
  --output operations/qa/evidence/preview-logout-03c/native-recovery-isolated.json
python3 tests/webphone/verify_preview_logout_proposal.py /tmp/pv03-source \
  --output operations/qa/evidence/preview-logout-03c/proposal-verification.json
node --check tests/webphone/runner.cjs
node --check tests/webphone/browser_workflow.js
node --check tests/webphone/test_preview_logout_proposal.cjs
python3 tests/webphone/preview_state.py operations/qa/evidence/preview-logout-03c/final-state.json
```

The source paths above are private temporary copies, not repository deliverables. Reproduction with differently located copies is documented in the proposal README. Existing Playwright/Chromium and private phone/agent credential paths were used for the native SKIP run; no authenticated profile, request body, credential or recording audio was saved. No screenshots were collected. [Regression comparison](evidence/preview-logout-03c/regression-comparison.json) contains 14 offline consistency checks over the saved runtime evidence; it is not another independent runtime test.

## 8. Acceptance matrix

Results describe the installed system and the stated requirement, not merely whether an investigation assertion passed. Uncorrected direct/reconnect failures reuse clearly identified PRD-03B runtime evidence with current identical source hashes.

| ID | Result | Evidence / limit |
|---|---|---|
| PV-C01 | PASS | Exact recovery options, SQL, scope, guards, counters and age window established; installed branch plus 15 isolated cases. |
| PV-C02 | FAIL | Existing recovery has date-limited preview guard and unconditional check/update race; unsafe resets reproduced in isolation. It was not enabled live. |
| PV-C03 | PASS | Actual Chromium native SKIP lead 56 restored NEW/count 0 and cleared display; native skip log and successful logout. |
| PV-C04 | FAIL | Installed direct logout still omits release; prior 52/54 native reproductions remain applicable. Proposed corrected browser path NOT_RUN. |
| PV-C05 | FAIL | Direct/logout reconnect did not preserve normal eligibility; ERI and called Y persist. Native reselection after proposed called-N restoration NOT_RUN. |
| PV-C06 | PARTIAL | Fresh SKIP control count 0; retained direct-logout diagnostics still count 1 without a call. Proposed counter restoration has no MariaDB acceptance. |
| PV-C07 | FAIL | Retained owning-agent reconnect converted 52/54 to ERI. No approved source correction has been deployed. |
| PV-C08 | PARTIAL | Local proposed duplicate/ACK-loss guards pass; actual corrected double-click/concurrency scenario NOT_RUN. |
| PV-C09 | PASS | Fresh accepted control logout/unregister and final zero live agents, conferences and channels; 1002 unregistered. Direct native phone-teardown correction NOT_RUN. |
| PV-C10 | PASS | Only lead 56 changed among observed lead snapshots; campaign/test01 and list settings restored/preserved, 20 shared file hashes unchanged, baseline call/record counts retained. Scope is bounded comparisons, not a whole-database equality claim. |
| PV-C11 | PASS | Existing phone/role/recording settings, native source and infrastructure fingerprints preserved; VICIphone actual session succeeds; services/timers healthy. No new customer-call/audio/playback retest claimed. |
| PV-C12 | PARTIAL | Existing native control ran successfully; proposal local tests pass and guarded corrected Playwright modes are supplied, but corrected browser/eligibility acceptance remains pending. |
| PV-C13 | PASS | Final native SELECT: zero synthetic INCALL claims and live agents; no whole-list reset or new native recovery required. |
| PV-C14 | PASS | Native activation reversed and verified; proposal apply/reverse restores exact source hashes on local copies; scoped rollback documented. Host source rollback not needed because no patch was deployed. |

## 9. Regression and error review

[regression.json](evidence/preview-logout-03c/regression.json) and [comparison](evidence/preview-logout-03c/regression-comparison.json) establish **20 unchanged shared fingerprints** against PRD-03B, including native agent/admin/playback files, recovery script, cron, Asterisk SIP/PJSIP/HTTP configuration, Apache HTTPS, portal code and recording/TLS components. Database checks preserve user 6666's recording permission, restricted qaagent14/qasup14 settings, QAPRD14 visibility, phones 1000/1001/1002 and suspended 1014, and recording rows 2–4. test01 remains unchanged. This is bounded regression evidence; external softphone registrations were already absent at the initial snapshot and are not newly certified here.

Asterisk, MariaDB, Apache, PHP-FPM and recording/backup timers were active. Timer listing retains certificate renewal and portal apply timers. No reload/restart, route, grant, firewall, TLS renewal, backup schedule, recording processor/playback, timezone or Perl dependency change was made. Current native agent and recovery source match their exact pre-test hashes.

Fresh selected PHP/Apache error review covered 15:35–15:39 EST in the last 10,000 lines, filtered to executed agent endpoints: one PHP warning and no selected fatal error. Browser output records one TypeError name without an attributed payload. These are limitations on a clean-console claim; observed preview/SKIP/logout and backend results completed. Broader telemetry duplicate-key, password reflection and active WebSocket revocation findings remain separate; no repair or new acceptance claim was made for them.

## 10. Exact changes and final fixture

Remote mutations were native UI activation/reversion of QAOUT01/list 910016 and normal agent preview/SKIP/session history for **lead 56**. No new account, campaign, list, lead, call or recording. No stranded-lead administrative recovery was needed in this assignment. [Native cleanup](evidence/preview-logout-03c/native-cleanup.json) and [final state at 20:51:17 UTC](evidence/preview-logout-03c/final-state.json) confirm:

| Item | Final state |
|---|---|
| QAOUT01 | Inactive N; MANUAL; level 0; PREVIEW_AND_SKIP; manual list 910016; no-hopper logins N; local time 12pm-5pm; recording ONDEMAND. |
| Lists 910014–910017 | All inactive N; original campaign associations retained. |
| Clean list 910016 | 20 records: 17 NEW, three NI (50,52,54). |
| Lead 56 / GAP01B_008 | NEW, qaagent14, count 0, called Y; retained SKIP control with no outbound call. |
| Unused leads 57–68 | NEW, blank owner, count 0, called N; smallest available future acceptance fixtures. |
| Earlier leads | 49–55 unchanged this assignment; 52/54 remain NI/count 1/called Y from prior manual recovery. |
| Accepted history | Lead 8 SALE/count 2/Y2 and lead 69 NI/count 2/Y unchanged; QAOUT01 outbound rows 2, total recording rows 4. |
| Sessions/media | Zero live agents, active channels/calls, conference participants and synthetic INCALL claims; 1002 unregistered. |
| Other phones | 1014 inactive/SUSPENDED; 1000/1001 configuration unchanged. |

Authorized repository changes are this report; the updated fixture inventory; `operations/qa/evidence/preview-logout-03c/*.json`; the proposal README/diff; new `qualify_preview_recovery.py`, `verify_preview_logout_proposal.py` and `test_preview_logout_proposal.cjs`; additive corrected-mode changes to existing runner/browser workflow; and expansion of the read-only diagnostic call query through lead 68. No installer code or upstream source was installed. The delivery response records the exact scoped Git commit and final worktree state; `git log -1 --format=%H -- operations/qa/VICIDIAL_PRD_03C_PREVIEW_LOGOUT_CORRECTION.md` retrieves that commit without a self-referential hash in this file. No push.

## 11. Rollback and pending acceptance

Temporary native changes are already reversed. To repeat reversal after a later test, native Campaigns → QAOUT01: inactive N, MANUAL, level 0, manual list 910016, PREVIEW_AND_SKIP, no-hopper logins N, local call time 12pm-5pm, ONDEMAND; save/reopen. Native Lists 910014–910017: inactive N. Release any open non-calling preview through visible SKIP before session teardown. Confirm zero live sessions/channels and identify any synthetic claim by lead/list/vendor before native Lead Details recovery. Do not reset whole lists or alter active-call state/history.

No source change was deployed, so source rollback on dialer01 is unnecessary. For any separately approved future patch, back up and restore only the two pinned files with their metadata after draining affected sessions; validate syntax/hashes and refresh agent pages. Do not restore a whole database. Reverse application was verified locally byte for byte, not on a live deployment. Full details and upgrade consequences are in the proposal README.

Remaining acceptance requires explicit approval for isolated MariaDB/MyISAM concurrency/fault qualification and then a scoped native two-file test deployment. After that, actual Chromium must prove direct LOGOUT, repeated/interrupted logout, same-agent reconnect, next-lead retrieval of the released lead, call-count/flag/counter correctness, no unwanted ERI and active-call protection. End each test with the current inactive fixture. No external call or infrastructure change is required or implicitly authorized.

## 12. Final verdict

`VICIDIAL_PRD_03C_PARTIAL`

The normal release workflow remains usable and the native recovery uncertainty is resolved. Direct logout is still defective on the unchanged installation. A pinned, locally tested correction proposal is ready for review; deployment and corrected native acceptance remain outside the current source-change authorization.
