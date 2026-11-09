# VICIdial outbound synthetic acceptance fixture

Assignment: `VICIDIAL_PRD_01_02_03_14_ACCEPTANCE`  
Executed: 2026-11-09 on `dialer01`  
Owner: native administrator; fixture retained for bounded follow-up acceptance.

This fixture was created through native VICIdial Administration. It is **inactive at handoff**. It does not authorize customer calls, automatic dialing, predictive dialing, carrier provisioning, or changes to the known-good baseline. Acceptance results and evidence are in [the acceptance report](VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md).

## Object register and final settings

| Object | Identifier | Final configuration / purpose |
|---|---|---|
| Campaign | `QAOUT01` | `QA Outbound Synthetic Only`; active `N`; method `MANUAL`; automatic dial level `0`; administration ownership group `ADMIN` |
| List | `910014` | `QA PRD synthetic only`; campaign `QAOUT01`; active `N`; expiration `2099-12-31` |
| User group | `QAPRD14` | `QA PRD synthetic restricted`; allowed campaign only `QAOUT01`; admin-viewable and agent-status-viewable groups only `QAPRD14` |
| Agent | `qaagent14` | Active `Y`, level `1`, group `QAPRD14`; reports `0`, protected recording access `0`; users/campaigns/lists/phones modification and lead loading `0` |
| Supervisor | `qasup14` | Active `Y`, level `7`, group `QAPRD14`; reports `1`, protected recording access `0`; users/campaigns/lists/phones modification and lead loading `0` |
| Inactive phone | `1014` | Extension, dialplan number, voicemail, outbound CID and native login `1014`; protocol `SIP`; active `N`; status `SUSPENDED`; type `QA INACTIVE`; fullname `QA PRD inactive only`; group `QAPRD14`; blank template |
| Leads | `9`–`28` | 20 synthetic records; vendor IDs `PRD14_001`–`PRD14_020`; list `910014` |
| Preview lead | `9` | `PRD14_001`; phone `7000000001`; final status `NI`, called flag `N` after native list reset |
| Non-routed leads | `10`–`28` | Phones `0000000102`–`0000000120`; final status `DNC`; no matching full prefixed Asterisk route |

Campaign settings reopened and verified through UI: both dial prefixes `2221`; manual list `910014`; manual dial and manual search filters `CAMPLISTS_ONLY`; recording `ONDEMAND`; local call time `12pm-5pm`; no-hopper-leads logins `N`; hopper minimum `1`. Final dialable status string is ` NI NEW -`. `NI` was added through the campaign UI for a repeat preview attempt after lead 9 was cleared from `INCALL`. Other 19 leads remain `DNC` and are not in the campaign's dialable statuses.

Final manual preview mode is `PREVIEW_AND_SKIP`. The successful preview used the earlier `PREVIEW_ONLY` setting. The final skip mode persisted, but a second agent login was refused because the hopper remained empty after native status change/list reset. Skip completion is **not accepted**.

The group permits only `Real-Time Main Report`. Its queue-groups default remains `-ALL-GROUPS-`; no inbound-group routing or inbound acceptance was configured. Do not reuse this group for inbound testing without reviewing that setting. Default agent `vicidial_recording=1` permits the native recording button; it does not confer protected-file retrieval. Both test users have `access_recordings=0`.

Passwords are intentionally absent. Test-user credentials were generated privately for this execution. Use the existing private credential handoff or have the authorized administrator reset only these synthetic accounts through native Administration if a later operator needs access. Phone 1014 has no registered client and must remain inactive unless a later task explicitly covers it.

## Lead file and native loader mappings

Canonical fixture: [prd-outbound-leads.csv](fixtures/prd-outbound-leads.csv). UTF-8, comma-separated, header plus 20 records, nine columns. Treat phone numbers as strings to preserve leading zeros.

| CSV column | Zero-based index | Native custom mapping |
|---|---:|---|
| vendor_lead_code | 0 | Vendor Lead Code |
| phone_code | 1 | Phone Code |
| phone_number | 2 | Phone Number |
| first_name | 3 | First Name |
| last_name | 4 | Last Name |
| status | 5 | No status target offered by the tested fifth-generation custom mapper |
| list_id | 6 | List ID |
| email | 7 | Email |
| comments | 8 | Comments |

Expected source statuses: first record `NEW`, remaining 19 `DNC`. All names are `QA SYNTHETIC_nnn`, addresses use `example.invalid`, and comments state `SYNTHETIC ONLY - NO CALL AUTHORIZATION`.

**CSV upload currently fails before field mapping** because the native converter cannot load `Spreadsheet::Read`. No CSV record was imported. A tab-separated derivative of the same file was uploaded through the same native loader to assess subsequent behavior. Derive it locally with Python's standard `csv` reader/writer and delimiter `\t`; no server-side converter replacement is needed for this diagnostic workaround.

The native TSV custom mapper auto-detected the source `status` column as **State**. Clear `state_field` to `-1` before processing. Map the other eight fields as above. Select `DUPLIST`, `NONE` for USA/Canada validation, and leave invalid-phone replacement disabled. The file supplies list `910014`. The loader has no status mapping and initialized every imported lead as `NEW`. Keep campaign and list inactive while importing. Through native Lead Details, set leads 10–28 to `DNC` before enabling either object.

First TSV import: good `20`, bad `1`, total `21`; the header was counted as an invalid phone-length row. Duplicate reupload: good `0`, bad `21`, total `21`, duplicates `20`, invalid `1`. List count remained 20. `DUPLIST` compares phone numbers within the specified list; it does not make arbitrary reimports with another duplicate mode safe. No deliberately invalid customer-like telephone row was needed: the header exercised invalid-row reporting.

## Routing and operation limits

Only lead 9 uses the established simulator number. Full `22217000000001@default` resolves to the existing `Dial(SIP/1001,45,To)` route. All 19 full `2221`-prefixed zero-leading numbers were checked with `asterisk -rx 'dialplan show NUMBER@default'` and had no matching extension. Existing carriers were inactive when inspected. No carrier or dialplan was created.

The existing agent phone 1000 was used only for the authorized agent conference login. There was no customer call. Phone 1014 deliberately produces no active Asterisk peer. Campaign 0 dial level, MANUAL mode, list/campaign inactivity, restricted campaign access, DNC statuses and verified route absence are the fixture safeguards; none should be removed casually.

## Safe reuse procedure

1. Open native Administration as the authorized administrator. Verify IDs, current count 20, statuses, MANUAL method and automatic level 0. Recheck route absence and carrier state if either may have changed.
2. Review the acceptance report's unresolved CSV, hopper and manual lifecycle issues. Do not assume the retained configuration completes a repeat session unchanged.
3. Keep campaign/list inactive for import or lead edits. Reupload only with `DUPLIST` and inspect the summary; do not use SQL to repair import results.
4. For an authorized non-calling session, activate only QAOUT01 and list 910014, keep dial level 0, inspect hopper eligibility and use the visible Lead Preview checkbox. Do not click DIAL LEAD. The prior login required a hopper lead; a later repeat attempt after NI/reset had none and was refused.
5. Use qaagent14 and the existing authorized phone login. Answer only the agent's conference login call. Verify native session, runtime association and visible controls. If the installed MANUAL workflow still lacks ready/pause/resume, record the gap; do not switch to automatic dialing to force acceptance.
6. Complete preview using visible native controls if available. Verify status and called flag after skip/logout. If logout strands this synthetic lead in `INCALL`, record the behavior and use native Lead Details to set `NI`; do not edit tables.
7. Log out normally. Confirm no QA live agents or Asterisk channels. Deactivate campaign and list again through native Administration and reopen to verify persistence.
8. A customer call and fresh recording regression require separate explicit operator authorization even for simulator lead 9.

## Cleanup / rollback

Safe deactivation is preferred. This assignment left campaign QAOUT01 and list 910014 inactive, phone 1014 inactive/SUSPENDED, and no live session.

- Native Campaigns → QAOUT01 → Active `N` → save/reopen. Preserve MANUAL and level 0.
- Native Lists → 910014 → Active `N` → save/reopen. Retain synthetic lead history and DNC safeguards.
- Native Users → qaagent14 / qasup14 → Active `N` → save/reopen if retiring accounts. Do not change 6666 or agent 1000.
- Native Phones → 1014 → inactive/SUSPENDED. No endpoint/client cleanup is needed; none was registered.
- If revoking fixture access, change only QAPRD14's QAOUT01 and report permissions after disabling its accounts. Do not edit ADMIN.
- Keep leads isolated in inactive list 910014. Delete only after a separate explicit decision; identify them by both list ID and vendor prefix, not a global phone-number match.
- No global native settings, source, generated config, credentials of existing users, SIP endpoints, recording integration, timers or firewall settings were changed; no whole-database restore is appropriate.

Never delete or repurpose test01, lead 8, recording 2, the previous successful call, phone 1000/1001, or user 6666 to clean up this fixture.
