# VICIdial PRD-01 / PRD-02 / PRD-03 / PRD-14 acceptance

Assignment: `VICIDIAL_PRD_01_02_03_14_ACCEPTANCE`  
Execution date: 2026-11-09  
Worker: Codex; native UI functional execution and read-only host correlation  
Target: `dialer01`, SSH TCP 22222  
Final verdict: **`VICIDIAL_PRD_01_02_03_14_FAILED`**

## 1. Executive summary

Native campaign/list/account/group provisioning, separate-role authorization, agent phone login, list-driven lead preview and normal session logout worked. A dedicated synthetic fixture remains available and inactive. No automatic dialing or customer call occurred.

The mandatory **native CSV import workflow failed**. The installed `sheet2tab.pl` cannot load `Spreadsheet::Read`; the web loader then attempts to read its absent conversion output and terminates with a PHP TypeError. This is an observed installation dependency defect blocking an essential acceptance workflow, so the assignment's FAILED definition applies. The configured distribution repositories did not provide `perl-Spreadsheet-Read`. The operator authorized distribution-packaged converter dependencies only; no package was installed, no repository added, and no upstream code modified.

A native TSV upload of the same 20 synthetic records allowed downstream field mapping, duplicate rejection, campaign association and agent preview to be exercised. It is explicitly a diagnostic workaround and does not count as CSV acceptance. The mapper did not offer a status target, initialized all records NEW, and incorrectly suggested mapping the status header to State; the operator-facing workflow required clearing that mapping and applying DNC through native Lead Details while the fixture was inactive.

Ready/pause/resume and complete non-calling agent lead handling remain incomplete. The MANUAL/level-0 screen did not present a ready/pause toggle. Preview succeeded without dialing, but logout left that lead INCALL; the administrator cleared it to NI through Lead Details. PREVIEW_AND_SKIP persisted afterward, but repeat login was refused for an empty hopper even after native list reset. These gaps are not certified as working or assigned a speculative root cause.

The previous call/recording baseline was preserved by the bounded checks described below. Fresh customer-call/recording regression is NOT_RUN because it was not authorized.

## 2. Current system baseline and authority

Read before implementation: system product-readiness audit, product acceptance plan, recording playback acceptance, recording finalization acceptance and recording README. Repository goals/plans and relevant compatibility/media guidance were consulted. This task validates the existing installation only; it does not promote an AlmaDial installer stack lock or imply production readiness.

Observed retained SVN working-copy revision: **4037**. Asterisk CLI reports **20.21.0**; PHP CLI **8.3.36**. System baseline remains VICIdial 2.14b0.5 on the supplied AlmaLinux 10.2/MariaDB 10.11 installation. UI components carry their own builds, including agent `2.14-728c / 261101-1945`, admin `2.14-969a / 261102-0125`, and loader `2.14-85 / 260912-0956`; they should not be confused with SVN revision.

Before creating fixtures, existing identifiers and privileges were inspected. Existing campaign test01, lists 998/999, users 6666/1000 and inactive VDAD/VDCL, group ADMIN and phones gs102/callin/1000/1001 were distinguished from the new IDs. Existing leads in list 101 and lead 8 in list 998 were excluded. New IDs were checked available before native creation. The retained [inventory](evidence/prd-01-02-03-14/inventory.json) is a later snapshot after creation/import, not a pristine pre-creation database snapshot.

The [initial host snapshot](evidence/prd-01-02-03-14/before.json) records backup health, native source and portal/recording integration hashes, service state, permissions and recording metadata. Backup health was OK: completed backup `20261109T084003-23e8842a`, 1,364,673,208 bytes, integrity verification present. No new backup or dump was created by this assignment.

All remote shell execution used noninteractive `ssh -T -p 22222`; HISTFILE was empty. No root history or shell initialization files were changed. Existing credentials were read privately, not reset. Browser tests used existing installed Chromium/Playwright; no local dependencies were installed. SQL evidence contains SELECT queries only. Configuration mutations were native form submissions within the synthetic scope.

The operator completed the Secure Access Portal/admin prerequisites and authorized private native credentials. The operator reported Zoiper 1000 ready and authorized use of its existing private credentials for phone login. This authority covered the agent conference login, not a customer call.

## 3. Native configuration workflow

Native Administration was exercised with an existing authorized administrator:

1. Campaigns → add QAOUT01 → save → reopen and configure MANUAL, automatic level 0, test-only prefixes/list filters, ONDEMAND recording, and preview.
2. Lists → add 910014 → associate QAOUT01 → save/reopen. Import and status preparation were done with the list inactive, then it was enabled for the bounded session.
3. User Groups → create QAPRD14 → restrict allowed campaigns, viewable groups and allowed report.
4. Users → create qaagent14 at level 1 and qasup14 at level 7 → group QAPRD14 → reopen and confirm privileges. The supervisor's report access was explicitly enabled; protected recording access stayed disabled for both.
5. Phones → add unused 1014 → inactive/SUSPENDED → save/reopen. No new active SIP endpoint or registration was introduced.
6. Native fifth-generation list loader → CSV attempt → observed conversion failure. Native TSV custom mapping → import → native Lead Details DNC changes → duplicate reupload.
7. Normal agent phone/campaign login → visible lead-preview controls → normal logout.
8. Native Administration → deactivate QAOUT01 and 910014 → reopen and confirm final persistence.

Native nested-menu clicks occasionally produced an interrupted navigation/empty automation page. Reopening the actual native link resolved navigation; transient empty captures were discarded. They are not counted as successful UI evidence or diagnosed as a server product failure.

Evidence: [campaign creation response](evidence/prd-01-02-03-14/campaign-create.txt), [list creation response](evidence/prd-01-02-03-14/list-create.txt), [campaign persisted settings](evidence/prd-01-02-03-14/campaign-persisted.json), [group settings](evidence/prd-01-02-03-14/group-persisted.json), [supervisor settings](evidence/prd-01-02-03-14/supervisor-persisted.json), [phone persisted settings](evidence/prd-01-02-03-14/phone-persisted.json), and final reopened [campaign](evidence/prd-01-02-03-14/campaign-final-ui.json)/[list](evidence/prd-01-02-03-14/list-final-ui.json) controls.

## 4. Exact created objects and campaign/list settings

| Type | ID | Final state |
|---|---|---|
| Campaign | QAOUT01 | Inactive, MANUAL, automatic dial level 0 |
| List | 910014 | Inactive; campaign QAOUT01; 20 synthetic leads |
| Group | QAPRD14 | QAOUT01 only; own-group visibility; Real-Time Main Report only |
| Agent | qaagent14 | Active, level 1, QAPRD14; no administration/report/protected-recording privileges |
| Supervisor | qasup14 | Active, level 7, QAPRD14; restricted report access; no provisioning/import/protected-recording privileges |
| Phone | 1014 | Inactive/SUSPENDED SIP record; no runtime peer |
| Leads | 9–28 | Vendor PRD14_001–020; lead 9 NI, leads 10–28 DNC |

Both campaign prefixes are 2221. Manual list 910014; manual dial and search filters CAMPLISTS_ONLY; recording ONDEMAND; local call time 12pm-5pm; hopper minimum 1; no-hopper-leads login N. Final dialable statuses ` NI NEW -`; final preview PREVIEW_AND_SKIP. The successful preview ran with PREVIEW_ONLY. The final skip mode was not successfully exercised. Campaign ownership remains ADMIN; agent campaign permission is enforced through QAPRD14's native allowed-campaign setting.

Phone 1014 persistence passed. The generator was enabled (`generate_vicidial_conf=Y`) and had no pending rebuild at final check; Asterisk reported no peer 1014, consistent with inactive/SUSPENDED. Active phone generation and registration were intentionally not exercised, so PRD-01D is PARTIAL rather than a claim that new active endpoint propagation works. Existing 1000 and 1001 configuration hashes were unchanged over the phone/session test interval.

See [fixture register, mappings, safeguards and reuse instructions](VICIDIAL_OUTBOUND_TEST_FIXTURE.md) for the complete retained fixture.

## 5. Lead import evidence

Canonical [CSV fixture](fixtures/prd-outbound-leads.csv): header plus 20 synthetic rows, nine fields, deterministic vendor IDs; first phone is the simulator, the remaining 19 are zero-leading non-routed strings. No customer data, real-world destination or carrier route was introduced.

CSV native upload reached the column-file page but offered no mapping controls. At 22:57:55 Asia/Manila PHP logged `fopen(/tmp/prd-outbound-leads.txt)` failure at loader line 3034 and a fatal `fgets()` resource/false TypeError at line 3038. `perl -c sheet2tab.pl` stopped at its required Spreadsheet::Read module, line 44. Read and XLSX were not installed. A cache-only distribution transaction with `--assumeno` found no package matching perl-Spreadsheet-Read and exited 1. No partial package installation occurred. Evidence: [dependency check](evidence/prd-01-02-03-14/csv-dependency.txt), [CSV page text](evidence/prd-01-02-03-14/import-mapping.txt), [fatal and repository result](evidence/prd-01-02-03-14/final-checks.json).

For downstream assessment, the same CSV was converted locally to tab-separated text with Python's standard csv utilities and uploaded through the normal native loader. No direct database writes were used. Field mapping was available for vendor ID, phone code/number, first/last name, list, email and comments. The incorrect status→State suggestion was removed before processing. The tested custom mapper did not offer lead-status mapping.

Native summary: good 20, bad 1, total 21, invalid phone length 1 (header). Read-only SQL corroborated IDs 9–28, list 910014, all expected names/vendor IDs/phone strings/emails/comments, blank State, and initial status NEW for every record. Native Lead Details changes then set 19 non-routed leads DNC. Thus source statuses were **not retained by import**; their final controlled state required native administrative work.

Duplicate reupload with DUPLIST: good 0, bad 21, total 21, duplicates 20, invalid 1. All duplicate phone numbers were identified in list 910014. Final list count remained 20. Evidence: [mapping](evidence/prd-01-02-03-14/tsv-mapping.json), [first summary](evidence/prd-01-02-03-14/tsv-import.txt), [initial SQL result](evidence/prd-01-02-03-14/lead-import-db.json), [native DNC updates](evidence/prd-01-02-03-14/native-dnc-updates.json), [duplicate summary](evidence/prd-01-02-03-14/tsv-duplicate-import.txt), and [final counts/statuses](evidence/prd-01-02-03-14/final-checks.json).

The eligible imported record was actually retrieved in the native agent screen. This is list-driven preview evidence, not a manually typed phone number.

## 6. Agent session and non-calling lead handling

The normal phone-login form used existing authorized phone 1000 credentials; the separate campaign-login form used qaagent14. Focusing the native campaign selector offered only QAOUT01. Initial login with an inactive list was correctly refused for no hopper leads. After enabling the prepared list and hopper eligibility, normal login succeeded.

At canonical epoch **1794236920** (2026-11-09 15:08:40 UTC), qaagent14 joined QAOUT01, phone PJSIP/1000, session/conference 9600000. The live-agent row was PAUSED. Asterisk showed the PJSIP/1000 leg Up in ConfBridge and its native conference announcement channels. There was no SIP/1001 customer leg.

The MANUAL/level-0 page showed Dial Next Number and Lead Preview but no visible ready/pause/resume toggle. No hidden JavaScript variables or functions were used to manufacture state transitions. Ready/pause/resume is NOT_RUN; the observed initial PAUSED state alone is not acceptance of that lifecycle.

The visible Lead Preview checkbox was checked, then the visible Dial Next Number control was clicked. The screen displayed lead ID 9, vendor PRD14_001, list 910014, QA SYNTHETIC_001, phone 7000000001 and the synthetic comments. UID/customer channel remained blank; the screen instructed Preview the Lead then DIAL LEAD. **DIAL LEAD was never clicked.** Runtime corroboration still showed only the agent conference leg. Native preview set lead 9 INCALL.

Normal LOGOUT produced `LOGOUT PROCESS COMPLETE, YOU MAY NOW CLOSE YOUR BROWSER OR LOG BACK IN`. The user log records logout at 23:13:00 PHP wall time, corroborated by agent-log epoch **1794237180** / 15:13:00 UTC, 260 seconds after login. The live-agent row and Asterisk conference/channels disappeared. Lead 9, however, remained INCALL after logout. It was set to NI through native Lead Details. PREVIEW_AND_SKIP and an NI dialable status were then saved through the campaign form and the synthetic list's called flags reset through the list form. A repeated native login was still denied for no hopper leads; no second session or skip completion is claimed. Campaign and list were then deactivated.

Evidence: [campaign selector](evidence/prd-01-02-03-14/agent-campaign-choices.json), [successful login screen text](evidence/prd-01-02-03-14/agent-login-retry.txt), [session screenshot](evidence/prd-01-02-03-14/agent-session.png), [login SQL/runtime](evidence/prd-01-02-03-14/session-login-db.json), [preview screenshot](evidence/prd-01-02-03-14/lead-preview.png), [preview field values](evidence/prd-01-02-03-14/preview-fields.json), [preview runtime](evidence/prd-01-02-03-14/session-preview-db.json), [logout screen](evidence/prd-01-02-03-14/logout-first.txt), [post-logout SQL/runtime](evidence/prd-01-02-03-14/session-first-logout-db.json), and [final state](evidence/prd-01-02-03-14/session-final-db.json).

## 7. Native permissions and shared-IP authorization

Separate native accounts were used for administrator, qaagent14 and qasup14; each restricted account had its own Basic-auth browser context. All contexts used the same portal-granted public source IP. This tested native application authorization independently of the common network grant.

| Account / action | Observed result |
|---|---|
| Agent: direct campaign/user/phone creation, foreign test01 edit, list loader, realtime/detail reports | HTTP 401 native denial; no administration forms |
| Agent: protected existing recording HEAD | HTTP 403 |
| Agent: native campaign selector | QAOUT01 only; successful native agent login proves credentials valid |
| Supervisor: direct campaign/user/phone provisioning or foreign campaign/user edit | Reports-only administration home, no modification controls; HTTP 200 is not treated as successful administration |
| Supervisor: list loader | HTTP 401 |
| Supervisor: allowed Real-Time Main Report | HTTP 200; rendered qaagent14, PAUSED, PJSIP/1000, QAOUT01 during the live session |
| Supervisor: Agent Time Detail report | HTTP 401, explicit report-not-allowed message |
| Supervisor: direct user_stats for existing 6666 | Native `This user does not exist` filtering response; no foreign statistics |
| Supervisor: groups[]=test01 on realtime | Unauthorized campaign filtered out; falls back to authorized ALL-ACTIVE scope. During session only QAOUT01 agent rendered; after deactivation no agents/campaign statistics rendered |
| Supervisor: protected existing recording HEAD | HTTP 403 |
| Existing authorized administrator/recording user | Synthetic fixture management succeeds; same protected recording HEAD HTTP 200, audio/wav, expected content length |

The native source's actual realtime selector is `groups[]`; an earlier probe with singular `group` did not select a campaign and is not counted as a negative visibility test. No foreign campaign was activated or foreign agent logged in for a simultaneous multi-agent visibility scenario; that belongs to a later milestone. The tested foreign user exists, so its filtered statistics response is meaningful negative evidence.

Evidence: [role checks](evidence/prd-01-02-03-14/permission-checks.json), [additional checks](evidence/prd-01-02-03-14/permission-more.json), [live supervisor visibility](evidence/prd-01-02-03-14/supervisor-visibility.json), [foreign-user/direct-URL checks](evidence/prd-01-02-03-14/supervisor-foreign-access.json), [authorized recording HEAD](evidence/prd-01-02-03-14/authorized-recording-head.json), and [effective privileges](evidence/prd-01-02-03-14/final-checks.json). No recording audio bytes were exported.

Observed compatibility/security concern: the native failed-administration-login response reflected the supplied password in its denial text. Evidence substitutes `[REDACTED]`; no raw page/credential dump is committed. This was not changed within the assignment. Separate follow-up should address that native response behavior and any credential rotation warranted by exposure during diagnostics. Permission enforcement itself worked in the tested routes.

## 8. Database/runtime correlation and time handling

Read-only MariaDB queries corroborated native UI results, not substituted for UI acceptance. Retained queries are embedded in the evidence JSON. At final handoff: QAOUT01 N/MANUAL/0, list 910014 N, 20 records (19 DNC/1 NI), zero QA outbound-call rows, zero QA recording rows, zero QA live agents. Asterisk had zero active calls/channels and no occupied ConfBridge.

Time inconsistency is demonstrated, not corrected:

| Value | Observed interpretation |
|---|---|
| PHP/native user_log LOGIN string | 2026-11-09 23:08:40 (Asia/Manila wall clock) |
| Agent-log pause_epoch | 1794236920 = 2026-11-09 15:08:40 UTC |
| MariaDB system/session timezone | EST / SYSTEM |
| MariaDB UNIX_TIMESTAMP(user_log.event_date) | 1794283720, 46,800 seconds (13 hours) later than the actual epoch |
| Native LOGOUT string | 2026-11-09 23:13:00 |
| Agent-log wait_epoch after logout | 1794237180 = 15:13:00 UTC |
| Agent-log pause_sec / wait_sec / talk_sec / dispo_sec | 260 / 0 / 0 / 0 |

Login-to-logout duration agrees at 260 seconds and talk time remains zero. The initial paused interval is accounted for; ready/pause/resume intervals were not executed. PRD-03F remains PARTIAL because the complete lifecycle correlation is missing and naive DATETIME-to-epoch conversion is wrong on this installation. No system/PHP/database timezone was changed.

## 9. PHP/Asterisk error and compatibility review

Fresh relevant PHP output was inspected during the workflow, with an end-of-run aggregate covering 22:46–23:22 Asia/Manila. [The aggregate](evidence/prd-01-02-03-14/php-log-review.json) retains severity/file/line only, excluding request values and SQL text. The actionable fatal is the CSV loader's absent converter-output file followed by TypeError. [Final checks](evidence/prd-01-02-03-14/final-checks.json) retain the exact sanitized fatal and the missing-package result.

Undefined-variable warnings in native administration, stylesheet/dbconnect and report paths occurred. Campaign/list/user/phone saves and reopen checks succeeded despite these warnings. They are not individually classified as failed product tests without demonstrated behavioral impact. The realtime page also emitted a small literal `\\n\";` prefix; its permitted agent data rendered. This is a presentation compatibility issue, not evidence that report authorization failed.

Asterisk's error-priority journal for the test interval returned no entries. CLI correlation showed the expected agent conference and clean teardown; no demonstrated configuration-generation error affected 1000/1001. An inactive phone record cannot validate active endpoint generation. No broad compatibility fixes were attempted.

## 10. Functional acceptance matrix

PASS applies only to the bounded behavior and cited evidence. PARTIAL separates completed portions from missing mandatory evidence; NOT_RUN identifies an unexecuted behavior. TSV success does not override the CSV failure.

| ID | Requirement | Result | Evidence / limit |
|---|---|---|---|
| PRD-01A | Create campaign through native UI | PASS | Native creation response, reopened campaign settings, final UI persistence |
| PRD-01B | Create/associate list through native UI | PASS | Native list response, reopened association, final UI and SQL |
| PRD-01C | Provision agent and supervisor through native UI | PASS | Native saved roles/groups; actual distinct-account login/permission behavior; effective privileges |
| PRD-01D | Native phone configuration persists correctly | PARTIAL | 1014 saved/reopened inactive/SUSPENDED; no active peer expected. Active generation/registration not tested |
| PRD-02A | Synthetic CSV imports successfully | FAIL | Missing Spreadsheet::Read; conversion output absent; PHP TypeError; no distro package available |
| PRD-02B | Imported fields and counts are correct | PARTIAL | TSV gives 20 correct data rows and predictable header/duplicate rejection; source statuses not retained and CSV failed |
| PRD-02C | Campaign/list/lead association works | PASS | List 910014→QAOUT01; SQL count20; imported lead9 retrieved in authorized campaign |
| PRD-02D | Native lead retrieval or preview works | PASS | Visible Lead Preview/Dial Next Number retrieved lead9 with deterministic fields; no customer channel |
| PRD-03A | Agent login succeeds | PASS | Normal qaagent14 campaign login; UI and user_log LOGIN |
| PRD-03B | Phone/session connection succeeds | PASS | PJSIP/1000 Up in ConfBridge9600000, native session header, live-agent row |
| PRD-03C | Ready/pause/resume functions correctly | NOT_RUN | MANUAL screen had no visible ready/pause/resume control; initial PAUSED state alone insufficient |
| PRD-03D | Agent lead handling works | PARTIAL | Preview works; complete non-calling skip/disposition not completed; preview logout left INCALL, cleared administratively |
| PRD-03E | Logout and session cleanup succeed | PASS | Native completion message, LOGOUT event, live-agent row removed, conference/channels empty; lead status caveat under03D |
| PRD-03F | Session logs agree with observed actions | PARTIAL | Login/logout/260-second paused duration and zero talk agree; full transition coverage missing; 13-hour DATETIME interpretation discrepancy |
| PRD-14A | Agent administrative restrictions enforced | PASS | Actual agent account direct URLs denied; protected recording403; authorized campaign login succeeds |
| PRD-14B | Supervisor privileges behave as configured | PASS | Allowed realtime data rendered; disallowed report/import denied; provisioning forms absent; recording403 |
| PRD-14C | Native role/group visibility enforced | PASS | QAOUT01-only selector; own agent visible; actual foreign6666 stats filtered; unauthorized campaign request filtered |
| PRD-14D | Shared-IP network access does not bypass native permissions | PASS | Same source IP and existing portal grant: admin management/recording200, restricted accounts denied by role |

## 11. Operator-confirmed tests and remaining operator acceptance

Received operator statements: administrator ready after portal access, private credential-file path supplied, Zoiper 1000 ready with permission to use existing credentials, and distro-only converter dependency installation authorized. Actual phone session connection was additionally demonstrated by runtime evidence. No independent operator statement confirming complete UI usability, two-way audio, preview/skip completion or ready/pause/resume was received or inferred.

Fresh manual customer call / ringing / answer / two-way audio / fresh recording capture-finalization-playback regression: **NOT_RUN**, no explicit authorization. Previously accepted recording playback is preserved, not reaccepted through a new call.

Follow-up operator acceptance after bounded blockers are resolved:

1. Administrator uploads the unchanged CSV through the native loader and observes mapping, summary, list contents and statuses; use inactive synthetic objects and DUPLIST.
2. Agent completes normal phone/campaign login. Verify the supported native non-calling path for ready/pause/resume and preview-and-skip. Record each visible action and canonical epoch. Do not enable automatic dialing to satisfy this test.
3. Verify preview completion returns the lead to an intentional native status and logout removes live state/channels. Investigate the empty hopper after NI/reset before claiming repeatability.
4. Repeat supervisor allowed realtime and agent/supervisor denied direct URL/recording checks from the same network grant.
5. Authorize a separate simulator customer call only if the optional fresh recording regression is wanted.

## 12. Remaining issues and bounded next work

| Issue | Classification | Effect / next action |
|---|---|---|
| Spreadsheet::Read unavailable to native converter and configured distro package resolution | Installation dependency | Blocks mandatory CSV. Find an authorized supported dependency source/package plan; current authorization excludes new repositories/CPAN. No upstream edit workaround |
| Native loader TypeError after converter failure | PHP compatibility/error handling | UI cannot present useful mapping/result; retain evidence for a separate scoped remediation decision |
| No status field mapping, status→State auto-detection | Native loader configuration/usability | Source DNC not preserved. Keep fixture inactive until native status controls are applied; assess a supported native loader/status workflow |
| No ready/pause/resume control in tested MANUAL screen | Missing test prerequisite / workflow gap | Determine supported native configuration and repeat bounded lifecycle; no demonstrated transition defect yet |
| Preview logout leaves INCALL; repeat NI/reset had empty hopper | Native lead/session configuration behavior; root cause unresolved | Administrator recovery worked; complete preview/skip and deterministic refill remain open |
| PHP/DB DATETIME interpretation differs by13h | Demonstrated timestamp inconsistency | Use event epochs; separate remediation requires bounded scope, no timezone changes here |
| Native auth denial reflects submitted password | Demonstrated native response concern | Evidence redacted; separate remediation/credential-handling follow-up |
| Active new-phone propagation | Missing runtime evidence | Inactive persistence accepted only; do not claim active endpoint generation |

No essential data corruption outside the synthetic scope was demonstrated. Missing lifecycle evidence is not itself called a product failure; the actual CSV defect determines the final verdict.

## 13. Regression findings

[Before](evidence/prd-01-02-03-14/before.json), [after](evidence/prd-01-02-03-14/after.json), [Asterisk/known-good object comparison](evidence/prd-01-02-03-14/protected-after.json), and [comparison summary](evidence/prd-01-02-03-14/regression-comparison.json) support these bounded findings:

- test01 remains active MANUAL/0, prefixes2221, ALLFORCE recording, ADMIN ownership; lead8/recording2 and call1793990617.45 were not repurposed.
- Existing simulator route remains `Dial(SIP/1001,45,To)`. PJSIP1000 and SIP1001 remain present; protected Asterisk files have identical hashes across the phone/session interval. All 19 non-routed fixture numbers still have no matching dialplan extension.
- Native administrator and agent login worked. Existing user6666 recording permission remained active with access_recordings1. Authorized protected recording HEAD200 and restricted HEAD403 were demonstrated.
- Recording finalizer/config/timer and protected playback/native integration files have identical pre/post hashes. Existing recording file size/hash and recording metadata are unchanged. Finalizer timer remains active. A fresh finalization event was NOT_RUN.
- Portal authentication/personal/shared-office source, access policy source and Apache HTTPS route configuration hashes are unchanged; portal and apply/SIP-guard services are active. Existing portal-granted native access worked. A new personal-grant/shared-office-grant lifecycle was not reexecuted; this is preservation evidence, not fresh portal feature acceptance.
- SSH22222 was used successfully; HTTPS responded; Apache syntax check passed with its preexisting FQDN warning. Firewall permanent/runtime configuration is identical. Dynamic nft state was captured, not edited; grant counters/expiry may change normally.
- HTTPD, PHP-FPM, MariaDB, Asterisk and portal services remained active without restart by this assignment. Recording, TLS renewal and backup timers remain active/enabled. Backup health remains OK; no restore or timer-job execution was attempted.
- Final Asterisk state: zero active channels and calls. QA call/recording counts are zero. No customer audio was exported or listened to.

These checks preserve the accepted baseline while avoiding unapproved calls or unrelated infrastructure tests.

## 14. Cleanup, evidence and Git handoff

Retain the fixture. Both synthetic campaign and list are inactive; phone1014 inactive/SUSPENDED; test users remain active with restricted permissions. Safe retirement: deactivate only QAOUT01, list910014, qaagent14 and qasup14 through their native edit forms; keep phone1014 inactive; revoke only QAPRD14 fixture permissions if needed. Preserve its synthetic leads/history in the inactive list rather than deleting records. Full steps and safe reuse prerequisites are in [the fixture document](VICIDIAL_OUTBOUND_TEST_FIXTURE.md). No global rollback or database restore is warranted.

Evidence directory: [prd-01-02-03-14](evidence/prd-01-02-03-14/). It contains selected native UI text/controls/screenshots, query results, hashes and bounded health checks; no passwords, SIP secrets, cookies, audio or database dump. Raw native HTML and private browser/credential helpers were not committed. Native denial texts are sanitized; trailing whitespace in captured UI text is normalized. A SHA-256 manifest binds retained evidence.

Git baseline: `c32a88a24613330f7dd9576deebd00be55f0b463`. Unrelated untracked work was preserved. Only the two assignment reports, synthetic CSV and this assignment's evidence are included in the scoped commit; no push. The resulting commit identifier is recorded in the final engineering handoff (and is available with `git log -1 --format=%H -- operations/qa/VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md`). No installer/source changes were made, so Go tests/scaffold guidance checks were not applicable. Local verification checks CSV structure/counts, evidence JSON parsing, final invariants, known-secret exclusion and staged whitespace.

## 15. Final verdict

**`VICIDIAL_PRD_01_02_03_14_FAILED`**

Mandatory native CSV import has a reproduced blocking dependency/error-handling defect. Provisioning, downstream TSV association/preview, phone session and the tested native permissions work within the stated evidence limits. Complete lifecycle and preview completion remain open. The safe retained fixture and previous accepted call/recording baseline remain available for a bounded remediation and acceptance rerun.
