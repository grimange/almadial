# PRD-03B — Preview lead lifecycle and direct logout

## 1. Executive summary

**Direct logout from an open preview is a reproducible lifecycle gap in installed SVN r4037.** It closes the agent session and conference but omits the preview-release operation. The lead remains INCALL until another recovery action. A subsequent login by the same agent changes the orphan to ERI, retaining the called flag and count; this does not restore ordinary next-lead eligibility in QAOUT01.

The reliable native sequence is **SKIP LEAD → wait for cleared lead fields → unregister VICIphone → LOGOUT → wait for completion**. Three controls completed that sequence successfully. Two direct-logout trials stranded claims. Both were recovered through native Lead Details after sessions had closed. The visible LOGOUT control allows this sequence without a preview warning; there is insufficient basis to label it an unsupported operator action or to declare the gap resolved by training alone.

No upstream source, schema, host configuration, phone template, package, service, firewall, recording integration or timezone was changed. No customer/external call was placed. Final acceptance is PARTIAL because direct logout still requires recovery. This is engineering functional acceptance by the implementing worker, not independent QA or a new human audio acceptance.

## 2. Installed product baseline

Target: dialer01, SSH TCP 22222; AlmaLinux 10.2, Asterisk 20.21.0, MariaDB 10.11, VICIdial 2.14b0.5 / SVN r4037. Reused accepted VICIphone 3.0.0, chan_sip extension 1002 and conference 9600001. Actual CLI Chromium version: 149.0.7827.55. No new framework or dependency was installed.

Read the prior [functional-gap report](VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md), [original PRD report](VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md), [fixture inventory](VICIDIAL_OUTBOUND_TEST_FIXTURE.md), [WebRTC acceptance](VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md), [webphone instructions](../webphone/README.md) and product acceptance plan. Earlier accepted provisioning, CSV import, agent controls, roles, calls and recordings remain the baseline; only affected lifecycle paths were retested.

Execution occurred on **2026-11-09 UTC**, corresponding to 2026-11-10 in the operator's Asia/Manila timezone. Server DATETIME values are America/New_York, UTC−5 on this date. Times below are UTC, correlated with native event_epoch; no timezone setting or historical value was rewritten.

Git baseline: `ec101ecb61138fe3ccf6d613dac3d20c861b77c2`. Tracked files were initially clean; unrelated untracked work was preserved. SSH was noninteractive with `-T` and `-F /dev/null`; no root history or shell initialization was edited. The operator completed the normal access grant; credentials were read from existing private local handoffs and excluded from evidence.

Before previewing, [pre-session.json](evidence/preview-logout-03b/pre-session.json) recorded QAOUT01 inactive, MANUAL, level 0; all four lists inactive; no agents/channels/INCALL claims. Existing unused synthetic leads 51–55 in list 910016 were selected; no new database objects were necessary. Leads 8 and 69 with accepted call history were not used for diagnostics. Existing backup health was checked during the bounded test: last successful verified backup `20261109T084003-23e8842a`, approximately 10.7 hours old, outcome ok, five payloads and 376 tables. No new VM snapshot/backup or backup restore was performed.

## 3. Native preview-state implementation

Installed source was inspected read-only. [native-source.json](evidence/preview-logout-03b/native-source.json) contains exact file SHA-256 values and selected line-numbered excerpts. These are evidence excerpts, not deployed changes.

| Native path / line | Observed implementation and consequence |
|---|---|
| `agc/vdc_db_query.php:4783` | Claim sets lead INCALL, increments called_count and sets called_since_last_reset before the preview branch. Thus INCALL during preview is an expected reservation, not evidence of a customer call. |
| `agc/vdc_db_query.php:4855` | Preview YES sets live-agent preview_lead_id and does not send the customer Originate. In observed sessions the agent remained PAUSED, lead_id 0, preview_lead_id equal to the selected lead. |
| `agc/vicidial.php:12812` | Agent screen explicitly offers DIAL LEAD or SKIP LEAD. |
| `agc/vdc_db_query.php:6565` / `6589` | manDiaLskip clears preview_lead_id, restores prior status and decrements called_count; it writes a native skip log and returns LEAD REVERTED. It does not reset called_since_last_reset. |
| `agc/vicidial.php:13657` | Skip response handling clears the displayed lead fields after the native response; the error branch reports LEAD NOT REVERTED. Tests also corroborated the backend result. |
| `agc/vicidial.php:19549` / `19567` / `19623` | NormalLogout/LogouT guards call/dial states but does not guard the preview state or call ManualDialSkip. userLOGout request does not carry the prior status/release operation. |
| `agc/vdc_db_query.php:18977` / `19115` | userLOGout logs logout, clears conference/session references, requests agent hangup and deletes the live-agent row. The inspected branch has no equivalent preview-lead restoration. Its one-second sleep is not a lead-recovery interval. |
| `agc/vicidial.php:4971` | Fresh login marks the user's lingering QUEUE/INCALL leads ERI and clears user. This cleanup is by user, not restricted to the newly selected campaign. It does not restore the prior status, called flag or count. |
| `agc/vdc_db_query.php:4377` | The ordinary manual next-lead selection requires called_since_last_reset N and an allowed campaign dial status, together with other eligibility filters. ERI is absent from QAOUT01's `NI NEW` status set. |

Recent-session M rows are created when a lead is selected even for non-calling preview. They are selection history, not proof of a continuing live reservation or a completed call. No outbound call log exists for test leads 51–55.

## 4. Normal SKIP LEAD workflow

Native Administration temporarily activated only QAOUT01 and list 910016. Temporary campaign settings: no-hopper-leads logins Y and local call time 24hours. MANUAL, level 0, PREVIEW_AND_SKIP and recording ONDEMAND stayed unchanged. [native-activation.json](evidence/preview-logout-03b/native-activation.json) records persistence. An initial Dial Next attempt before the minutely hopper populated the first lead made no claim; retry after population succeeded.

| Lead / vendor | Account | Preview | Explicit SKIP | Logout | Result |
|---|---|---|---|---|---|
| 51 / GAP01B_003 | qaagent14 | 19:14:11 | 19:14:34 | 19:14:42 | INCALL / count 1 → NEW / count 0; preview pointer cleared; no remaining session/channel. |
| 53 / GAP01B_005 | qasup14 | 19:16:27 | 19:16:35 | 19:16:37 | Same release result using the existing second permitted native account. |
| 55 / GAP01B_007 | qaagent14 | 19:21:59 | 19:21:59 | 19:22:00 | Same result in the reusable CLI runner after helper timing correction. |

Evidence: [control claim](evidence/preview-logout-03b/control-claimed.json), [control skip](evidence/preview-logout-03b/control-skipped.json), [control logout](evidence/preview-logout-03b/control-logout.json), [other-account preview](evidence/preview-logout-03b/other-agent-preview.json), [CLI browser output](evidence/preview-logout-03b/cli-control-repeat.json) and [CLI state correlation](evidence/preview-logout-03b/cli-control-repeat-state.json). Native skip logs preserve previous_status NEW and previous_called_count 0 for all three controls.

All skipped leads retain called_since_last_reset Y. Restoration to NEW does **not** promise immediate next-lead reselection without the normal subsequent list/reset lifecycle. No whole-list reset was performed merely to repeat acceptance.

## 5. Direct logout reproduction

Two independent unused leads were selected through visible Lead Preview / Dial Next. The diagnostic did not select DIAL LEAD. Phone unregistration preceded native visible LOGOUT so registration cleanup was consistent with the accepted browser runner; it is not a lead release.

| Trial | Claim | Native logout | Observed result |
|---|---|---|---|
| Lead 52 / GAP01B_004 | 19:15:10, INCALL, qaagent14, count 1, called Y | 19:15:32; event epoch 1794251732 | At 19:15:42.507: INCALL, no live agent/channel. At 19:17:48.551: still INCALL, 126 seconds after the first post-logout capture and approximately 136 seconds after logout. |
| Lead 54 / GAP01B_006 | 19:19:32, same claim state | 19:19:51; event epoch 1794251991 | Again INCALL after logout with no live agent/channel. A later login changed it to ERI. |

Evidence: [direct claim](evidence/preview-logout-03b/direct-claimed.json), [immediate result](evidence/preview-logout-03b/direct-immediate.json), [maintenance-window result](evidence/preview-logout-03b/direct-after-maintenance-window.json), [repeat claim](evidence/preview-logout-03b/reconnect-next-preview.json), [repeat logout](evidence/preview-logout-03b/direct-repeat-logout.json).

No status write or same-agent login occurred during the first observation window. Other-account testing and source inspection used that time. Hopper cron activity occurred during the window; it did not release lead 52. This is a bounded observation, not an assertion that the database would never change under any later maintenance condition.

## 6. Lead-state and agent-session correlation

Native user logs record complete login/logout pairs for the successful scenarios. Control 51: login epoch 1794251555, logout 1794251682. Direct 52: login 1794251702, logout 1794251732. Other account: login 1794251768, logout 1794251797. Reconnect/repeat direct: login 1794251916, logout 1794251991. CLI control: login 1794252112, logout 1794252120.

Observed preview pointers, PAUSED live state, conference 9600001, skip logs and post-logout zero channels agree with those actions. Agent log rows retain lead references and pause accounting for preview activity; wait/talk/disposition durations were zero for these non-calling sessions. Lead Details recovery later sets NI in associated native history; NI is not evidence of a call made during the preview test. A transient failed CLI initialization created an extra login row at epoch 1794252017 without its own normal logout; the subsequent native login replaced the stale session. Final live-agent and channel checks are zero.

Evidence is sanitized browser action output plus SELECT/CLI snapshots. No screenshots, full browser traces, customer audio, cookies or complete database dumps are included. Browser labels alone were not used to accept cleanup.

## 7. Subsequent login behavior

At 19:18:36 qaagent14 logged in afresh. Lead 52 changed **INCALL → ERI**, user became blank, called_count stayed 1 and called flag stayed Y. The agent screen did not reopen that lead. The next preview selected lead 54 instead. See [same-agent-relogin.json](evidence/preview-logout-03b/same-agent-relogin.json) and [reconnect-next-preview.json](evidence/preview-logout-03b/reconnect-next-preview.json).

After the second direct logout, the next qaagent14 login at 19:20:17 likewise changed lead 54 to ERI. [cli-control-repeat-state.json](evidence/preview-logout-03b/cli-control-repeat-state.json) records both orphans as ERI before administrative recovery. Fresh login therefore provides partial native orphan classification, not restoration of normal lead eligibility.

## 8. Other-agent lead visibility

The existing qasup14 account, already authorized for QAOUT01 through QAPRD14, was used sequentially with phone 1002. No account or group permission changed. It retrieved lead 53 while lead 52 remained INCALL owned by qaagent14. Its login did not recover the other user's claim. Source eligibility filters and this native next-lead result demonstrate exclusion in the configured workflow. Arbitrary searches, alternate phone entry, exhaustive draining of every eligible lead and a new concurrent-agent model were not tested.

## 9. Root-cause classification and native maintenance

**Classification: reproducible open-preview logout lifecycle gap, with partial recovery on the owning agent's next login.** Expected preview reservation and explicit SKIP work. The visible logout path lacks release/guarding and leaves an actual lead-availability consequence. This is not a stuck customer channel, a phantom live-agent row or merely a time-display artifact.

No applicable campaign/agent setting was identified that makes this logout path call the existing preview release. MANUAL/PREVIEW_AND_SKIP is the path under test. INBOUND_MAN was not retested: its earlier Ready/Pause acceptance is separate, and changing dial mode would not establish correctness of the observed MANUAL logout path. No external documentation was found or relied upon to declare direct logout unsupported; classification rests on installed source and actual native actions.

Installed `/usr/share/astguiclient/AST_flush_DBqueue.pl` supports `--reset-stuck-leads` and list restriction `--stuck-lists=X-Y-Z`. The source selects ERI/INCALL older than two minutes but newer than six hours, checks for absence of active preview and qualifying call/dial history, and can set eligible leads to NEW / called N. It does not restore arbitrary prior status/count. Current `/etc/cron.d/vicidial` invokes it hourly at minute 11 with **only `-q`**, so the option is disabled. The minutely hopper is not an equivalent preview release. Agent-log cleanup at minute 33 is not this status recovery. See [source evidence](evidence/preview-logout-03b/native-source.json) and [actual cron](evidence/preview-logout-03b/regression.json).

The optional recovery mode was not executed or scheduled. Its two-minute threshold is **not** an enabled two-minute cleanup guarantee. Its finite age window and forced NEW result need separate operational review before use. It cannot establish direct-logout acceptance in this deployment.

## 10. Native workflow and correction boundary

For an uncalled preview, use the discoverable **SKIP LEAD** control, wait for lead fields to clear, then unregister the embedded phone and log out. Corroborate zero live session/channel when ending an automated run. The ordinary test finish phase rejects an open preview. This preserves reliable native operations without a source patch, but remains a workaround for the allowed direct-logout path.

For a stranded synthetic lead: identify its list/vendor/lead ID; confirm its owning session and channels are gone; inspect its current status through Lead Details. A same-agent login may already have converted it to ERI. Change only that identified orphan through native Lead Details to an available appropriate status. In this run NEW was not offered; NI was deliberately chosen to isolate the two diagnostics. Do not claim that NI restores next-lead eligibility in this fixture: called Y remains.

No corrective source change is authorized here. A bounded follow-up could guard LogouT when a preview is open and visibly require SKIP; alternatively it could await the existing ManualDialSkip acknowledgment before logout, with server-side validation of ownership/current reservation. Test failures, duplicate clicks and in-flight release before accepting such a patch. Do not introduce a blanket INCALL reset, schema change or new daemon. No such proposal was deployed in this assignment.

## 11. Acceptance matrix

PASS below refers to the stated investigation/test criterion; in particular PV-03 means successful reproduction, not acceptance of direct logout.

| ID | Result | Evidence / practical meaning |
|---|---|---|
| PV-01 | PASS | Native claim and preview branch identified; source 4783/4855 plus control/direct live snapshots. |
| PV-02 | PASS | Leads 51, 53 and 55 explicitly skipped to NEW/count 0, preview cleared; skip logs and CLI/control evidence. |
| PV-03 | PASS | Direct logout independently reproduced on 52 and 54; both left INCALL after the session closed. |
| PV-04 | PASS | Logout omission, same-agent ERI recovery, disabled optional maintenance mode and observation window explained from source/cron/runtime. |
| PV-05 | PASS | Fresh owning-agent login changed 52 to ERI; next preview selected 54. Same transition subsequently observed for 54. |
| PV-06 | PASS | Other-account next preview selected 53 while 52 stayed claimed; next owning-agent preview skipped orphan 52; configured eligibility filters explain impact. Scope limits in §8. |
| PV-07 | PARTIAL | Explicit SKIP workflow verified repeatedly; no deployed native setting repairs/guards direct logout. Manual recovery remains necessary for that path. |
| PV-08 | PASS | Native recovery of only 52/54 to NI; final SELECT reports zero synthetic INCALL claims. |
| PV-09 | PASS | Successful logout completes; final zero live agents, channels/calls and conference participants; 1002 unregistered. |
| PV-10 | PASS | Actual Chromium registration, established conference, ICE/DTLS connected and subsequent unregister/CLI repeat; no architecture changes. No new two-way-call acceptance claimed. |
| PV-11 | PASS | Preserved campaign/lead snapshots, prior 14 file hashes, recording rows/counts, role/phone settings and active services; bounded regression only. |
| PV-12 | PASS | Existing helpers extended, actual CLI control executed and MCP diagnostic phases exercised; read-only correlation collector and commands below. Direct CLI orchestration is available but was not separately rerun end-to-end after the MCP diagnostics. |

[verification.json](evidence/preview-logout-03b/verification.json) records deterministic checks over saved snapshots. It is an evidence consistency check, not another runtime test or independent review.

## 12. Exact changes and fixture records

No new users, groups, phones, lists or leads were created. Temporary native settings and final recovery are recorded in [native-cleanup.json](evidence/preview-logout-03b/native-cleanup.json). Final QAOUT01 matches its pre-test relevant settings: active N, MANUAL, auto level 0, manual list 910016, PREVIEW_AND_SKIP, no-hopper logins N, local time 12pm-5pm, ONDEMAND. List 910016 returned to N; the other three lists remained N.

| Lead | Final status | User | Called count / flag | Retained purpose |
|---|---|---|---|---|
| 51 | NEW | qaagent14 | 0 / Y | Explicit SKIP control |
| 52 | NI | blank | 1 / Y | Direct logout → INCALL → fresh login ERI → native recovery NI |
| 53 | NEW | qasup14 | 0 / Y | Other-account explicit SKIP control |
| 54 | NI | blank | 1 / Y | Second direct logout and recovery |
| 55 | NEW | qaagent14 | 0 / Y | CLI explicit SKIP regression |

List 910016 retains 20 leads: 17 NEW / 3 NI (50,52,54). Unused 56–68 are NEW / called N / count 0. Earlier 49/50, original 9–28, diagnostic 29–48, lead 8 and accepted browser-call lead 69 were preserved. Preview/skip/user/agent history created by these tests is retained.

Authorized repository changes:

- This acceptance report and `operations/qa/evidence/preview-logout-03b/*.json`.
- `operations/qa/VICIDIAL_OUTBOUND_TEST_FIXTURE.md`: current browser objects, exact lead states and recovery instructions.
- `tests/webphone/browser_workflow.js`: synthetic preview selector/assertions, visible SKIP and opt-in direct-logout phase; waits for actual iframe navigation and loading overlay dismissal.
- `tests/webphone/runner.cjs`: skip/direct diagnostic orchestration, synthetic account selection, guarded failure cleanup and sanitized errors; existing default two non-calling cycles retained.
- `tests/webphone/preview_state.py`: SELECT-only remote state/runtime collector with host identity assertion, bounded SSH/subprocess timeouts, no recovery writes.

No remote configuration file was modified, so there is no host file backup/restore operation for this task. The Git delivery commit identifies these scoped files; its exact ID and final working-tree state are recorded in the delivery response. Unrelated untracked files/evidence remain outside the commit.

## 13. Playwright regression and reproducibility

Use existing installed Playwright/Chromium; no npm install is required. Private phone JSON has login/password; private agent JSON is an array of username/password objects. Keep files private and outside Git. Complete the normal portal web and phone-network grant first. Do not share browser profiles between accounts.

1. Inspect the current inactive fixture with the read-only collector. Through native Administration activate only QAOUT01/list 910016, preserve MANUAL/0, set no-hopper logins Y and local call time 24hours for the bounded test. Verify no active customer call and current eligible hopper lead. Do not reset all leads to force an expected ID.
2. Choose the next unused synthetic vendor by current native/SELECT state. At this handoff lead 56 / GAP01B_008 is unused; that is an example, not a permanent test prerequisite. The runner asserts the selected vendor after native Dial Next. An unexpected recognized synthetic preview is released via visible SKIP in error cleanup; no arbitrary lead mutation is performed.
3. Run a control, capturing read-only state before/after. Module/browser paths below reflect the existing installation; credential paths are placeholders.

```bash
python3 tests/webphone/preview_state.py /tmp/preview-before.json
export PLAYWRIGHT_MODULE=/home/ra/.npm/_npx/51691537fc71f2b0/node_modules/playwright
export CHROMIUM_PATH=/home/ra/.cache/ms-playwright/chromium-1228/chrome-linux64/chrome
export WEBPHONE_PHONE_CREDENTIALS=/private/path/phone.json
export WEBPHONE_AGENT_CREDENTIALS=/private/path/agents.json
export WEBPHONE_PREVIEW_CASE=skip
export WEBPHONE_PREVIEW_VENDOR=GAP01B_008
node tests/webphone/runner.cjs > /tmp/preview-control-browser.json
python3 tests/webphone/preview_state.py /tmp/preview-after.json
```

The control makes no customer call. It previews, skips, waits for field clearance, unregisters and logs out. Inspect exit status and backend state: CLI success alone does not verify the database status/count or zero live-agent rows. Read-only collection must corroborate them.

For an authorized reproduction, select a different currently unused vendor, set `WEBPHONE_PREVIEW_CASE=direct` and `WEBPHONE_PREVIEW_DIAGNOSTIC=YES`. This intentionally bypasses the helper's ordinary open-preview guard, uses visible LOGOUT and **requires native recovery**. Capture the immediate lead state and native maintenance-window observation **before** another owning-agent login. Another fresh login may turn INCALL into ERI and change the evidence. Use bounded polling plus actual cron/event evidence; no long blind sleep. To inspect reconnect without consuming another lead, unset the three WEBPHONE_PREVIEW variables and use the existing default non-calling session cycles, with SELECT snapshots around login. To exercise the existing second account sequentially, set `WEBPHONE_AGENT=qasup14` for the preview cases. Do not run both against 1002 concurrently.

For stepwise MCP inspection, load the same `browser_workflow.js` and set `page.webphoneConfig.phase` to start, preview, inspect, skip, finish; diagnostic direct-preview-logout additionally requires `authorizePreviewLogoutDiagnostic=true` and the same GAP01B previewVendor. The diagnostic phase was exercised in actual MCP Chromium sessions in this run; the CLI skip orchestration was also executed. Server lead statuses are read by the Python collector, not written from browser JavaScript.

The first CLI trial exposed an iframe timing race (src present before a navigated Frame existed). The helper now waits for frame navigation. A separate initial load timeout and an old-session notice covered by a phone-loading overlay were handled using DOM-content navigation and overlay/state assertions. One failed trial left a PAUSED row without a customer lead/channel; subsequent native login cleared/replaced it. These failures were not counted as successful sessions. The corrected CLI control exited 0 and completed native cleanup; the final small output-metadata additions and collector count/host assertions were syntax/read-only checked rather than repeating browser acceptance unnecessarily.

## 14. Final cleanup and regression state

[final-state.json](evidence/preview-logout-03b/final-state.json) is the final read-only host capture. QAOUT01/list settings are inactive as above; 1014 remains suspended. Browser phone 1002 is unregistered. No live agent, active channel/call, conference participant or synthetic INCALL lead remains. A hopper READY entry for an unused lead is not a live reservation and is retained behind an inactive campaign/list.

After confirming no active owning session/channel, native Lead Details changed only 52/54 from ERI to NI at 19:23:18 and 19:23:20. No SQL update, history deletion, broad reset or database restore was used.

[regression.json](evidence/preview-logout-03b/regression.json) corroborates SVN r4037; all 14 file hashes shared with the prior WebRTC after-state match, including portal, Apache, native admin/playback and recording integration files. Six additional current fingerprints cover Asterisk base/HTTP configuration and native agent/maintenance source; those extra fingerprints are not asserted to have an equivalent earlier snapshot. This is comparison to prior accepted evidence, not a claim that every service was newly exercised. QAOUT01 outbound call rows remain 2 and global recording rows remain 4. Lead 8 and 69 snapshots and recording rows 2/3/4 are preserved. No outbound rows exist for diagnostic leads 51–55. User 6666 retains protected recording access; qaagent14/qasup14 restrictions and QAPRD14 scope remain unchanged. Phones 1000/1001 retain configuration; SIP 1001 was already unregistered at preflight and is not presented as a new regression.

Asterisk, MariaDB, Apache, PHP-FPM, recording and backup timers were active in the bounded health check; SSH/HTTPS/native access worked. No fresh two-way-media call, protected playback, office/personal grant permutation, active WebSocket revocation or timer execution was repeated under PRD-03B. Those previously accepted functions are preserved through unchanged source/configuration and bounded state checks, not newly certified here.

Fresh error review: [error-review.json](evidence/preview-logout-03b/error-review.json) and [error-correlation.json](evidence/preview-logout-03b/error-correlation.json). Four native duplicate-key exceptions in agent visibility telemetry correlate with both explicit-release logout and direct logout (`conf_exten_check.php:1751`, `functions.php:2787`). They do not explain the missing release branch; both scenario types close sessions. Other PHP warnings and 39 Asterisk warning/error lines were counted without copying potentially sensitive full logs. No new general warning cleanup was attempted. A browser TypeError name was recorded during the successful CLI initialization; its cause is unassigned, and the subsequent asserted phone/preview/skip/logout sequence passed. This is not a clean-console claim.

## 15. Outstanding limitations

Direct open-preview logout remains problematic and is not accepted as automatically releasing a lead. Native fresh login changes it to ERI but does not restore eligibility. The administrator's NI recovery safely isolates the synthetic lead and does not substitute for a product correction. No configuration-only fix was established.

MANUAL was tested; INBOUND_MAN comparison, exhaustive alternate lead retrieval and long-duration optional recovery-mode trials were not run. No independent reviewer reproduced these tests. Earlier password reflection, Basic-auth cache behavior, active WebSocket revocation and future DST/calling-window qualification remain separate findings and were neither repaired nor recertified here.

## 16. Rollback and recommended follow-up

All temporary native settings have already been restored. If a later regression run is interrupted: inspect the owning live session and channels; release an open known synthetic preview via SKIP before logout if still possible; unregister 1002; confirm no live agent/channel; recover only the identified orphan through native Lead Details when necessary. Deactivate QAOUT01 and lists 910014–910017. Restore MANUAL/0, no-hopper N, local time 12pm-5pm and the retained manual list 910016. Leave 1014 suspended. Save/reopen through native Administration and corroborate with SELECT/CLI.

Do not delete diagnostic history or restore a database backup to undo previews. Repository rollback can revert this scoped commit after preserving any subsequent work; it must not revert the accepted VICIphone infrastructure or prior functional-gap corrections. No host file/service rollback is necessary for this task.

Next bounded action: decide whether to authorize a native logout guard/release patch and its focused regression, or to retain the explicit SKIP operator requirement with the direct-logout defect open. Do not proceed to blind INCALL recycling or general agent-state development.

## 17. Final verdict

`VICIDIAL_PRD_03B_PARTIAL`

The explicit native release workflow is repeatable and leaves no unintended reservation. Direct logout remains an allowed native UI action with a demonstrated lead-availability consequence and administrative recovery requirement. All synthetic claims were cleaned up, accepted baselines were preserved, and the defect is bounded precisely enough for a separately authorized correction.
