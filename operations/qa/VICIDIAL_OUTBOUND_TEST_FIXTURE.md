# VICIdial outbound synthetic acceptance fixture

Created by `VICIDIAL_PRD_01_02_03_14_ACCEPTANCE`; updated through `PRD-WEBPHONE-00`, `PRD-03B` and `PRD-03C`, 2026-11-09 UTC / 2026-11-10 Asia/Manila. All configuration and lead changes used native interfaces; live database verification was read-only.

The fixture is **inactive at handoff**. No customer call, automatic dialing, predictive dialing or carrier provisioning is authorized by retaining it. See [original acceptance](VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md), [functional remediation acceptance](VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md), [accepted browser phone](VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md) and [PRD-03B lifecycle investigation](VICIDIAL_PRD_03B_PREVIEW_LOGOUT_ACCEPTANCE.md). CSV import and agent controls work; direct logout from an open preview remains a documented limitation.

## Current object register

| Object | Identifier | Retained configuration / purpose |
|---|---|---|
| Campaign | `QAOUT01` | `QA Outbound Synthetic Only`; active `N`; method `MANUAL`; automatic level `0`; owner group `ADMIN` |
| Original list | `910014` | `QA PRD synthetic only`; QAOUT01; active `N`; expiration `2099-12-31` |
| Diagnostic CSV list | `910015` | `GAP01CSVsyntheticOnly`; QAOUT01; active `N`; retain only as evidence of pre-correction Unicode storage |
| Clean CSV list | `910016` | `GAP01UTF8CSVsyntheticOnly`; QAOUT01; active `N`; preferred follow-up fixture |
| Browser call list | `910017` | `WEB00 local simulator only`; QAOUT01; active `N`; retained accepted call history |
| User group | `QAPRD14` | Restricted; allowed campaign only QAOUT01; admin-viewable and agent-status-viewable groups only QAPRD14 |
| Agent | `qaagent14` | Active `Y`, level `1`, QAPRD14; reports `0`, protected recording access `0`; users/campaigns/lists/phones modification and lead loading `0` |
| Supervisor | `qasup14` | Active `Y`, level `7`, QAPRD14; reports `1`, protected recording access `0`; users/campaigns/lists/phones modification and lead loading `0` |
| Inactive phone | `1014` | SIP, active `N`, status `SUSPENDED`, type `QA INACTIVE`, native login/extension `1014`, fullname `QA PRD inactive only`, group QAPRD14; no registered client |
| Browser phone | `1002` | SIP / chan_sip WebRTC, active `Y`, template `WEB00_SIP`, embedded VICIphone 3.0.0; unregistered at handoff; conference `9600001` |
| Original leads | `9`–`28` | List 910014; vendor IDs `PRD14_001`–`PRD14_020`; lead 9 NI, 10–28 DNC; preserved |
| Diagnostic leads | `29`–`48` | List 910015; `GAP01_001`–`GAP01_020`; 20 NEW; some non-ASCII values contain mojibake; do not reuse as canonical data |
| Clean leads | `49`–`68` | List 910016; `GAP01B_001`–`GAP01B_020`; 17 NEW, three NI (`50`, `52`, `54`); unused `57`–`68` retain NEW / called N / count 0; 56 is a consumed SKIP control |
| Browser call lead | `69` | List 910017; `WEB00_001`; NI / count 2; accepted recordings 3 and 4; preserve this history |

Campaign: both dial prefixes `2221`; manual list **910016**; manual dial/search filters `CAMPLISTS_ONLY`; recording `ONDEMAND`; local call time `12pm-5pm`; no-hopper-leads logins `N`; hopper minimum `1`; dialable statuses ` NI NEW -`; preview `PREVIEW_AND_SKIP`; inbound `N`; max-inbound outcome `DEFAULT`. No new route or carrier exists. A retained hopper READY row is not a live agent or INCALL claim; inactive campaign/list and MANUAL level zero prevent operation.

The group permits only `Real-Time Main Report`. Its queue-groups default remains `-ALL-GROUPS-`; no inbound group was assigned. Default agent `vicidial_recording=1` permits the native recording button, independently of protected-file retrieval. Both test users retain `access_recordings=0`; user 6666 remains unchanged.

Passwords are intentionally absent. Use the existing private credential handoff, or reset only synthetic accounts through native Administration if separately needed. Phone 1014 must remain suspended unless an assignment explicitly covers activation.

## CSV fixtures and native mapping

The original [prd-outbound-leads.csv](fixtures/prd-outbound-leads.csv) and its prior TSV acceptance remain historical evidence. Its source status column was not a native status import target: fifth-generation custom mapping guessed it as State. All leads initially loaded as NEW, then the original operator workflow changed 19 to DNC and lead 9 to NI. These were explicit native changes, not a demonstrated status-parser defect.

New fixtures, UTF-8 with CRLF, header plus 20 records:

- [prd-functional-gaps-01.csv](fixtures/prd-functional-gaps-01.csv): diagnostic batch, list 910015 / vendor prefix GAP01_. Imported before the supported UTF-8 connection setting was enabled; retain isolated.
- [prd-functional-gaps-01-utf8.csv](fixtures/prd-functional-gaps-01-utf8.csv): clean batch, list 910016 / vendor prefix GAP01B_. All declared fields and initial NEW statuses verified against SQL and native views.

| Zero-based column | Native custom mapping |
|---:|---|
| 0 | Vendor Lead Code |
| 1 | Phone Code |
| 2 | Phone Number |
| 3 | First Name |
| 4 | Last Name |
| 5 | List ID |
| 6 | Email |
| 7 | Comments |
| 8 | City |

Use `admin_listloader_fifth_gen.php`, custom layout, `DUPLIST`, USA/Canada validation `NONE`, and invalid-phone replacement `DISABLED`. Clear every unused mapping to `-1`. Keep phone numbers as strings. No status column is supplied; expected import status is native NEW. Header is predictably rejected as one invalid phone row: summary good 20, bad 1, total 21. Reupload with DUPLIST: good 0, bad 21, duplicates 20, invalid 1; count stays 20. DUPLIST is per-list: duplicate phones deliberately exist across the two new lists. Do not assume DUPCAMP has the same behavior.

Phones `0000000201`–`0000000220`, names SYNTHETIC, email domain `example.invalid`, and explicit NO CALLS comments are synthetic. Quoted commas are parsed correctly. Native conversion strips embedded double quotes and semicolons, so expected stored comment is `SYNTHETIC ONLY, quoted CSV NO CALLS`. José, München and empty cities verify encoding and empty fields. The clean batch required native **Use Non-Latin (UTF8) = 1**; see [dependency record](../dependencies/VICIDIAL_PERL_CSV_IMPORT.md).

## Agent controls and preview lifecycle

MANUAL's lack of Ready/Pause controls is expected for this screen. To qualify those controls, the remediation temporarily used **INBOUND_MAN**, no inbound groups, inbound disabled and **ALLOW_MI_PAUSE**. Without ALLOW_MI_PAUSE and without groups, the frontend falls back to MANUAL. Administration coerces INBOUND_MAN's displayed/stored dial level to 1.0; the outbound dialer explicitly excludes both MANUAL and INBOUND_MAN, so no automatic outbound call was enabled or placed. Final mode is MANUAL / zero.

The outcome selector is visible only with inbound enabled. To repeat this configuration through the UI, keep the campaign inactive, choose INBOUND_MAN, briefly enable inbound with no assigned groups, save/reopen, select ALLOW_MI_PAUSE, then disable inbound and save/reopen. Verify the persisted controls and absence of assigned groups before login. This is a bounded manual agent-state test configuration, not authorization for inbound service or automatic dialing.

Session evidence demonstrated visible Ready → Pause → Resume transitions and corresponding READY/PAUSED live state and 31/45/36/43-second pause/wait intervals. Agent conference used existing PJSIP/1000 and ConfBridge 9600000 only.

**Always select SKIP LEAD before LOGOUT when merely previewing.** Lead 49: preview claimed INCALL; visible SKIP LEAD restored NEW, called flag Y, called count 0; subsequent normal logout left no claim. Skip does not automatically reset the list's called flag.

Comparison: lead 50 preview followed directly by LOGOUT closed the session/channels but retained INCALL. Native logout does not perform ManualDialSkip. Lead Details offered ordinary dispositions but not NEW for that claimed lead, so the administrator selected NI to recover it. No SQL write was used.

PRD-03B repeated the native comparison using VICIphone 1002. SKIP released leads `51`, `53` (qasup14) and `55` to NEW / count 0 / called Y. Direct logout left `52` and `54` INCALL / count 1 / called Y. Another account did not release the original agent's claim. A fresh login by the owning agent changed each stale claim to **ERI with blank user**, retaining count 1 and called Y; it did not restore the preview or normal next-lead eligibility. ERI is outside QAOUT01's dial statuses. Native Lead Details then recovered only `52` and `54` to **NI** after all agent channels had closed. All five diagnostic leads have no outbound call records.

The installed `AST_flush_DBqueue.pl --reset-stuck-leads` can reset eligible orphaned ERI/INCALL previews to NEW / called N, but the current hourly `-q` cron invocation does not enable that option. It was neither executed nor scheduled under PRD-03B. Do not assume the minutely hopper releases orphaned previews, and do not reset a whole list or all INCALL leads as routine cleanup.

[PRD-03C qualification](VICIDIAL_PRD_03C_PREVIEW_LOGOUT_CORRECTION.md) established that this recovery forces NEW without restoring original status/counts and has date-limited live-preview guards and a check/update race. Isolated execution reproduced unsafe resets; it remains disabled and was not run live. It is not an approved logout correction. The [two-file proposed correction](proposals/prd-03c/README.md) is undeployed and requires separate upstream-source approval. The new native control used lead **56 / GAP01B_008**: SKIP restored NEW / count 0 / called Y, then normal logout/unregister completed. No new call, recording or administrative lead recovery was performed. QAOUT01/list settings were restored; no INCALL claim, live agent or channel remains.

Final state has no synthetic INCALL claim, live agent or channel. Open-preview logout remains a product limitation; the accepted operational sequence is **SKIP LEAD → wait for the lead to clear → unregister VICIphone → LOGOUT → wait for completion**. The automation's ordinary finish phase rejects an open preview. Its direct-logout phase is an explicit diagnostic requiring read-only correlation and native recovery.

## Time and routing limits

PHP `date.timezone` in `/etc/php.d/99-vicidial.ini` now matches OS/MariaDB `America/New_York`; backup `/root/99-vicidial.ini.gap01-before`. Native paired event-date/epoch fields now agree. Existing records written with Asia/Manila retain their demonstrated 13-hour DATETIME interpretation error; use stored epochs for correlation. Browser header time uses the browser's Asia/Manila clock and can display the next date. No historical data was rewritten; OS/MariaDB timezone and server GMT settings were not changed.

Original lead 9 and browser-call lead 69 have the established simulator `7000000001`, full route `22217000000001@default` to SIP/1001. Clean CSV leads use non-routed numbers; recheck exact full prefixed destinations and carrier state before reuse. PRD-WEBPHONE-00 accepted a controlled local simulator call and protected recording playback; PRD-03B placed no customer calls and did not change that route. New calls require authorization in their own assignment.

## Safe reuse

1. Verify current campaign/list settings, counts, statuses, route absence and inactive carriers. Prefer clean list 910016; leave diagnostic 910015 and original 910014 inactive.
2. Keep campaign/list inactive for CSV import. Confirm dependencies and Use Non-Latin = 1. Use DUPLIST and review both summary and field values; never repair via SQL.
3. For authorized non-calling preview, activate only QAOUT01 and clean list, keep MANUAL / zero, and ensure a NEW eligible hopper lead. Use native list reset only when intended; Skip sets the called flag Y. Empty-hopper login can instead be allowed by the native no-hopper-leads logins option within the synthetic campaign.
4. Prefer the accepted VICIphone 1002 and existing [browser runner](../../tests/webphone/runner.cjs); it answers only the agent conference automatically. Extension 1000 remains available for separately authorized manual interaction. Authenticate qaagent14, select QAOUT01 and check Lead Preview. Do not click DIAL LEAD. Select an unused lead, currently `57`–`68`, instead of resetting consumed leads merely to rerun a test.
5. Use visible SKIP LEAD to release preview, then LOGOUT. Verify lead status, no QA live agent and no Asterisk channels. Direct logout with an open preview requires the administrative recovery described above.
6. If Ready/Pause testing is authorized, use the bounded INBOUND_MAN configuration above; restore MANUAL/zero afterward. Do not use RATIO, ADAPT or predictive modes.
7. Deactivate campaign and all four lists; leave 1014 suspended and 1002 unregistered. Confirm no INCALL claims or live sessions. Do not grant recording permissions to test accounts merely to make playback succeed.

## Cleanup and rollback

Prefer deactivation; preserve synthetic history. Native Campaigns → QAOUT01: active N, MANUAL, level 0, inbound N, outcome DEFAULT, no-hopper logins N, local call time 12pm-5pm, manual list 910016, save/reopen. Native Lists → 910014–910017: active N, save/reopen. Native Users → qaagent14/qasup14: active N if retiring. Revoke only QAPRD14 test-specific rights if retiring it, after disabling accounts. Leave phone 1014 inactive/SUSPENDED. Keep the accepted WebRTC infrastructure; unregister 1002 after use.

Identify leads by both list ID and vendor prefix. Retain diagnostic bad-Unicode batch inactive rather than silently rewriting it. Remove records only after a separate deletion decision. Recover an unintended synthetic INCALL claim through Lead Details using an available appropriate disposition; verify the correct lead ID first.

The remediation changed native use_non_latin from 0 to 1, the dedicated PHP timezone override, and the documented Perl dependencies. Rollback for those changes is in the [acceptance report](VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md) and [dependency record](../dependencies/VICIDIAL_PERL_CSV_IMPORT.md). Reverting encoding/timezone reintroduces demonstrated problems; no whole-database restore is appropriate.

Never delete or repurpose test01, lead 8, recording 2, the accepted call, phones 1000/1001 or user 6666 to clean up this fixture. Do not modify portal/firewall, recording integration, backup/TLS timers or generated Asterisk configuration.
