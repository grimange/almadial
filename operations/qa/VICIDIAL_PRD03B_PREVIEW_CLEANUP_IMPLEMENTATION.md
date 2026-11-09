# PRD-03B — Direct-preview logout correction: developer verification

## A. Baseline and scope

**The original defect was reproduced, a two-file native agent patch was deployed, and the corrected direct-logout sequence passed real Chromium/VICIphone/MariaDB verification. Independent QA remains required.** This later assignment explicitly authorized the small source correction that the earlier PRD-03B/03C assignments prohibited. Their reports remain historical evidence.

Target dialer01 / 2.29.153.161, SSH22222, AlmaLinux10.2, VICIdial2.14b0.5/schema1748/SVN4037, Asterisk20.21.0, MariaDB10.11.18, PHP8.3.36; VICIphone3.0.0/chan_sip1002/conference9600001 and Chromium149.0.7827.55. Execution: 2026-11-09UTC, 2026-11-10Asia/Manila. Server DATETIME is America/New_York, UTC−5 on the test date; epochs and captured UTC identify actions without changing timezones.

Git started at `7b3aa6e7349c8a32658c354d5cc86953e2ec16ec`, clean tracked state. Unrelated untracked installer work, reports and browser artifacts were preserved and excluded. The retained `/usr/src/astguiclient/trunk` checkout was r4037 with no local SVN changes. Before repair, the two deployed agent files exactly matched the pinned original source used by the local qualification. Existing Asterisk20/ConfBridge, WSS, recording and portal compatibility integration was retained; this is not a claim that all deployed files are vanilla upstream. The bounded installed-file inventory is in [final regression](evidence/preview-cleanup-repair/final-regression.json): 20 tracked source/configuration hashes, only the two intended agent files changed. Native admin/functions/report/playback, portal, Asterisk HTTP/SIP/PJSIP, cron and recording integration hashes remain identical to the preceding baseline.

Reviewed the prior preview/logout, functional-gap, fixture, webphone and product acceptance reports and existing runner. No installer stage, stack promotion, package, schema, timezone, cron, firewall, portal, endpoint/template or recording implementation change occurred. Noninteractive SSH with `-F /dev/null -T` was used; root shell history was untouched. All SQL issued by the test collector was read-only. Application writes occurred through authenticated native actions.

Evidence root: [preview-cleanup-repair](evidence/preview-cleanup-repair/). Scoped original-file backup: `/root/qa-backups/prd03b-20261109T220035Z`. [Backup health](evidence/preview-cleanup-repair/backup-health.json) reports outcome ok and verified successful backup `20261109T084003-23e8842a`, completed08:40:17UTC, five payloads/376 tables; the retained older last_failure is not the current outcome. No database restore or new snapshot was performed.

## B. Original reproduction

Native Administration activated only the synthetic fixture, MANUAL/0, PREVIEW_AND_SKIP, list910016, with bounded all-hours/no-hopper login settings temporarily permitted. The browser authenticated qaagent14, registered1002 and joined9600001. No diagnostic lead was dialed.

| Action | Lead57 / GAP01B_009 | Ownership/session evidence |
|---|---|---|
| Before | NEW / count0 / calledN | Unowned, no active agent/channel |
| Native preview,21:55:43UTC | INCALL / count1 / calledY | qaagent14, preview pointer57, current lead0; no customer Originate |
| Visible direct LOGOUT,21:55:44UTC | INCALL / count1 / calledY | Agent/conference ended |
| Observation21:56:07UTC | Still INCALL /1/Y | Zero live agents/channels |
| Fresh owning-agent login21:58:49UTC | ERI /1/Y, owner blank | Eligibility not restored |
| Native Lead Details21:59:33UTC | NI /1/Y | Only orphan57 recovered after session cleanup |

See [baseline](evidence/preview-cleanup-repair/baseline.json), [native acquisition/logout](evidence/preview-cleanup-repair/original-direct-diagnostic.json), [after logout](evidence/preview-cleanup-repair/original-after.json), [reconnect](evidence/preview-cleanup-repair/reconnect-before-state.json), and [native recovery](evidence/preview-cleanup-repair/native-orphan-recovery.json). HTTP responses are corroborated by row/pointer/runtime snapshots. Lead57 remains a diagnostic NI record; no fabricated call record was inserted.

## C. Root cause and recovery semantics

Line references below identify the original pinned source; the patch provides exact current offsets.

| Native path | Established transition |
|---|---|
| `agc/vdc_db_query.php:4377`, `4783`, `4855` — manDiaLnextCaLL | Eligibility uses campaign/list/status and calledN. Acquisition sets INCALL, increments count and daily counters, advances N→Y, then stores live-agent preview_lead_id with current lead0. No customer call is needed to reserve. |
| `agc/vdc_db_query.php:6565` — manDiaLskip | Restores previous native status/count, clears preview and records skip; legacy calledY and phone daily counter remain. |
| `agc/vicidial.php:13602` — ManualDialSkip | Clears frontend state before its asynchronous result; blindly invoking it and immediately logging out cannot guarantee acknowledgement or safe retry. |
| `agc/vicidial.php:19549/19567` — NormalLogout/LogouT | Live-call checks do not release an outstanding non-dialed preview. |
| `agc/vdc_db_query.php:18977/19115` — userLOGout | Logs/removes agent session and disconnects conference without undoing its preview reservation. |
| `agc/vicidial.php:4971` — owning-agent login | Orphan INCALL/QUEUE becomes ERI and loses owner; original count/eligibility are not restored. |

The source of truth is vicidial_list plus the matching vicidial_live_agents preview/current-lead fields. vicidial_auto_calls and live-agent state distinguish actual calls. vicidial_agent_log, vicidial_user_log, native skip audit and daily counts provide correlation. The lead/live-agent/daily tables use MyISAM; web_client_sessions uses MEMORY, so an ordinary rollback transaction cannot solve this transition.

The optional `AST_flush_DBqueue.pl --reset-stuck-leads` remains disabled in the hourly `-q` invocation. Its previously isolated qualification demonstrated forced NEW rather than original status/counter restoration, date-limited guards and a check/update race. Current script and cron hashes remain unchanged. It was not executed live, globally enabled or selected as the correction. See [03C qualification](VICIDIAL_PRD_03C_PREVIEW_LOGOUT_CORRECTION.md). No supported campaign setting was identified that makes the native visible LOGOUT perform the missing release.

## D. Correction and deployment

Actual [patch](../vicidial/patches/0001-preview-logout.patch) and [pins/apply/rollback instructions](../vicidial/patches/README.md). Only `/var/www/html/agc/vicidial.php` and `vdc_db_query.php` were changed. Originals were backed up with metadata; staged files passed target PHP8.3 lint before replacement. No Asterisk/MariaDB restart was needed. [Deployment v2](evidence/preview-cleanup-repair/deployment-v2.json) records the installed hashes.

The frontend routes an unfinished preview through authenticated native manDiaLskip with a dedicated `preview_action=logout`, waits for exact PREVIEW RELEASED, then performs ordinary native logout. Pending flags prevent repeated clicks and competing next/skip/dial actions. Phone teardown and completed-logout display wait for a validated native logout acknowledgement. A release error/timeout leaves the session/phone available and displays a retry warning.

The server serializes claim/dial/skip/logout for the native user with connection-scoped GET_LOCK and rechecks the web session after waiting. A preview DIAL carries `preview_action=dial` and revalidates ownership after acquiring the lock. Logout refuses a remaining preview pointer. The release uses one MyISAM multi-table update across the lead, matching live-agent pointer and both daily counters. It requires exact user/server/conference/campaign, INCALL/current lead0, PAUSED or READY, unchanged count, calledY proving the supported priorN case, positive counters, and no matching auto-call or pending external dial. It restores the previous native status supplied by the same native preview workflow, count−1 and calledN, clears the pointer, and reverses both acquisition counters. It writes the existing skip audit rather than inventing a call/disposition. Already-released retries acknowledge only the matching released state and do not decrement or audit twice.

This reuses native request/authentication, session, status/count and audit fields without schema objects, a new state authority or a recovery daemon. Legacy SKIP semantics are deliberately unchanged. GPL upstream attribution remains intact; only a contextual diff is stored. It is a local compatibility patch, not a claim of upstream acceptance; upgrades require rebase/requalification against exact source pins.

**Development failure retained:** the first candidate used `format=logout-preview`; native debug-disabled processing overwrote format, sending the request down legacy SKIP. Its acknowledgement was correctly rejected. An initial harness cleanup attempted SKIP again and caused an unsigned daily-counter underflow at17:01:23EST. Candidate v2 uses a separate parameter; its harness no longer blindly skips after a failed release. Lead58 is NEW/count0/calledY and not INCALL. [First candidate](evidence/preview-cleanup-repair/corrected-first.json), [state](evidence/preview-cleanup-repair/corrected-first-state.json), and [error stacks](evidence/preview-cleanup-repair/error-stacks.json) retain this failed development result. Misexpected hopper-order attempts are retained as failed harness attempts; they did not place additional calls and used bounded native cleanup. They are not counted as passing test runs.

## E. Developer regression matrix

All PASS entries below include real native execution and SELECT/runtime correlation. Local VM/SQLite tests supplement those results and are not substituted for them.

| Test | Result | Observed evidence |
|---|---|---|
| T01 Acquire preview | PASS | corrected-v2-state and repeated-next/preview: lead59 INCALL/1/Y, matching qaagent14 preview59/current0. |
| T02 Logout without dialing | PASS | corrected-v2: native release ACK then logout; lead59 NEW/0/N, daily counters reversed; no customer call, agent/channel cleared. |
| T03 Relogin | PASS | repeated-next: fresh native login retrieves **the same restored lead59** without ERI, proving actual eligibility rather than status alone. |
| T04 SKIP LEAD | PASS | skip-control: native visible SKIP on59 returns NEW/0/Y and clears preview; original legacy behavior preserved. |
| T05 Logout after SKIP | PASS | skip-control/after: no second cleanup/count change; clean session. |
| T06 Completed disposition | PASS | active-call/dispositioned and after: lead70 NI/1/Y preserved; stale release probe denied after native NI submission. Final call log NI/38sec and recording5 finalized. |
| T07 Active call protection | PASS | active-call/answered/protected: answered SIP1001 call, five channels, conference two participants, VLA INCALL/current70/preview0. Visible logout blocked; direct backend release denied; state/channels unchanged until intended hangup. |
| T08 Repeated logout | PASS | repeated-next double click produces one release; retry commits the real release then replaces only first response with503. UI retains session, retry ACK completes logout, lead61 NEW/0/N with one audit and zero daily count. |
| T09 Other-agent isolation | PASS | isolation: authenticated qasup14 release of qaagent14's lead60 denied; wrong-conference request also denied; snapshots retain claim/counters until owning native logout. Seventeen unrelated lead rows match baseline exactly. A second simultaneous preview owner was not created. |
| T10 Fresh-login logout | PASS | fresh-sessions: two new VICIphone login/conference/RTP/unregister/logout cycles, no preview/customer call. Final-state confirms no live agent/channel. |

Evidence: [corrected v2](evidence/preview-cleanup-repair/corrected-v2.json), [corrected state](evidence/preview-cleanup-repair/corrected-v2-state.json), [repeat/reconnect](evidence/preview-cleanup-repair/repeated-next.json), [SKIP control](evidence/preview-cleanup-repair/skip-control.json), [isolation](evidence/preview-cleanup-repair/isolation.json), [response-loss retry](evidence/preview-cleanup-repair/retry.json), [active-call test](evidence/preview-cleanup-repair/active-call.json), [fresh sessions](evidence/preview-cleanup-repair/fresh-sessions.json). Their sibling scenario directories contain before/preview/after and intermediate snapshots.

One authorized local simulator call used newly native-imported lead70 / GAP03B_CALL_001 in native-created list910018. [Route preflight](evidence/preview-cleanup-repair/call-routing-preflight.txt) proved `22217000000001@default` terminates at `Dial(SIP/1001,45,To)` then Hangup. No carrier/external call or automatic dialing occurred. Uniqueid1794262609.215; native call log NI/38sec, recording5 epoch1794262609–1794262647, finalized location populated. This test verifies active-state preservation and finalization, not renewed human audio-quality or protected-playback acceptance. Existing accepted playback integration and permissions were unchanged.

Local final checks: [local-final-verification](evidence/preview-cleanup-repair/local-final-verification.json): zero-fuzz apply/reverse restores both original digests, PHP lint,16 extracted JavaScript cases and15 admission predicate cases. Local PHP is8.4.27; target PHP8.3 execution is separately proven by native integration and deployment lint. Both JS files pass node --check. [Evidence assertions](evidence/preview-cleanup-repair/evidence-assertions.json) verify original failure, corrected counters/flags,17 preserved lead rows, final disposition, changed-file scope and idle state. [Reusable commands](../../tests/webphone/PREVIEW_CLEANUP.md).

## F. Errors, limitations, change control and final state

The bounded17:00–17:59EST error review contains six fatal exceptions: one explained first-candidate counter underflow, four known visibility telemetry duplicate-key exceptions (`visibleuser`, conf_exten_check.php:1751 via functions.php:2787), and **one separate native call-log PRIMARY-key duplicate at17:17:20EST**, vdc_db_query.php:9156. That existing branch performs a check followed by INSERT INTO vicidial_log; a duplicate insert attempt was rejected while the simulator call was active. A concurrent check/insert race is a plausible cause, not separately proven. This branch was not changed by the patch. The final bounded query shows one record for uniqueid1794262609.215, NI/38sec, and recording5 finalized; no duplicate call row or incorrect final lead state was produced. This remains a separate native PHP/SQL compatibility finding for QA, not a clean call-log error claim. No v2 counter-underflow or preview-release SQL error was observed. Browser TypeError names also persist from the existing baseline and remain unassigned. Raw credential-bearing logs were not exported. [Per-exception blocks](evidence/preview-cleanup-repair/error-blocks.json) are authoritative; the wider contextual [error stacks](evidence/preview-cleanup-repair/error-stacks.json) can include neighboring exception frames. See also [bounded counts](evidence/preview-cleanup-repair/final-regression.json).

Limits: supported scope is list-driven previously uncalled N→Y preview. Callback/manual-entry/alternate/prior-called paths fail closed and need separate runtime qualification. Non-NEW prior statuses are supported by native previous-status semantics and isolated predicate tests, but no separate native non-NEW fixture was consumed. No simultaneous multi-agent stress test, process-crash injection, expired-browser recovery or active WebSocket revocation test was performed. Other native writers do not acquire this new advisory lock; it is not a universal database concurrency guarantee. MyISAM cannot atomically roll back the separate audit insert: failure is explicit, but audit reconciliation remains an operator review boundary. Original native previous-status/count client inputs are retained; this task does not redesign their trust model.

[Final native cleanup](evidence/preview-cleanup-repair/native-final-cleanup.json) and [final state](evidence/preview-cleanup-repair/final-state.json): QAOUT01 inactive/MANUAL/0/PREVIEW_AND_SKIP, manual list910016, no-hopperN, call time12pm-5pm, recordingONDEMAND, omit-phone-codeN; lists910014–910018 inactive. Phone1014 stays suspended;1002 unregistered. Zero synthetic INCALL, live agents, channels or diagnostic outbound records for51–68. Existing lead8/69, accepted call1793990617.45, recordings2–4 and user6666 permissions remain unchanged. Services active; TLS/backup/recording timers retained. No broad permission, portal or telephony recertification is claimed.

Retained changed/new synthetic data:57 NI/1/Y after original-defect recovery;58 NEW/0/Y first candidate;59 NEW/0/Y after legacy SKIP control;60 NEW/0/Y after native harness cleanup;61 NEW/0/N after retry;62–68 unused NEW/0/N;70 NI/1/Y with one simulator call/recording5; new inactive list910018. Existing QA user/group/phone settings remain unchanged. No automated bulk lead reset or deletion was used.

Rollback: with inactive fixture and zero sessions/calls, restore only the two original files from the scoped private backup, preserving ownership/mode, lint and verify original hashes. Local reverse restored both exact hashes; no live rollback was executed. Reopen fresh browser sessions. The native application rollback reintroduces the original defect; use SKIP before logout if reverted. Deactivate/retain910018 and lead70 through native Administration, preserving call history. Never restore the whole database. Exact source pins, steps and upgrade implications are in the patch README.

Authorized repository changes: the patch/README, this report, fixture inventory update, tests/webphone/{runner.cjs,browser_workflow.js,preview_state.py,verify_preview_logout_proposal.py,verify_preview_cleanup_evidence.py,PREVIEW_CLEANUP.md,fixtures/preview-cleanup-simulator.csv}, and sanitized evidence under preview-cleanup-repair. No push. The Git commit identifier is reported in the final handoff and Kaneo comment; it is intentionally not embedded in its own committed content.

## G. Kaneo handoff

Existing project ALM (`ghritpte6e40o5ayktgz0vh1`), PersonalProjects workspace. No existing matching task was found before creating **ALM-36**, ID `kw5j5afcslviwtt8pt35uw0x`. Authorized worker **Codex**, ID `aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`, self-assigned and read back through MCP. Task was In Progress during investigation/deployment. Developer verification passed; final handoff moves it to **QA Review** (`in-review`) with ownership preserved, evidence/source summary and commit reference. The MCP final read-back is saved with evidence. It must not be marked Accepted by this implementation worker.

## H. Final verdict

`VICIDIAL_PRD03B_PREVIEW_CLEANUP_READY_FOR_QA`

The original sequence now releases the supported untouched preview, restores native eligibility/counters, survives retry, preserves actual calls/dispositions and closes the agent/phone session. This is developer readiness for a separate AGY review, not independent acceptance.
