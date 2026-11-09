# VICIDIAL_PRD_FUNCTIONAL_GAPS_01 acceptance

Execution: 2026-11-09 UTC, continuing into 2026-11-10 Asia/Manila. Target: dialer01, authorized SSH TCP 22222. Worker: Codex; implementation and engineering verification by the same worker. No independent reviewer or independent operator acceptance is claimed.

## Executive summary

**VICIDIAL_PRD_FUNCTIONAL_GAPS_01_PARTIAL**

Native CSV conversion/import now works, including an isolated 20-record batch with verified Unicode, empty fields, native NEW statuses and duplicate rejection. Ready/pause/resume is demonstrated through visible native controls with matching database state and durations. The 13-hour timestamp error was reproduced at the PHP write / MariaDB DATETIME interpretation boundary and corrected with the existing dedicated PHP timezone override.

**Open-preview logout still retains INCALL if the agent omits SKIP LEAD.** The supported release-before-logout workflow passed; bare logout did not release its reservation. This is demonstrated behavior, not missing evidence. No supported setting that makes that logout branch release the preview was identified. The report therefore does not give unconditional cleanup acceptance. It documents the required native workflow and the remaining lifecycle limitation without changing upstream code or treating administrative recovery as a fix.

The fixture ends inactive, with no stale claims or live sessions. Existing provisioning/permissions, phone/dialplan configuration, accepted recording and protected playback remain intact under targeted regression checks. No customer, simulator-customer or external call was placed; only the authorized agent conference connection was exercised.

## Baseline, authorization and method

Reviewed the previous [acceptance](VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md), [fixture inventory](VICIDIAL_OUTBOUND_TEST_FIXTURE.md), system product-readiness audit, product acceptance plan and recording playback acceptance, plus the task's retained evidence. Repository scope and the compatibility skill were consulted; no Go installer work or stack-lock promotion is included.

Retained baseline: AlmaLinux 10.2; VICIdial 2.14b0.5 / SVN r4037; Asterisk 20.21.0; MariaDB 10.11; PHP 8.3. Existing successful call/recording acceptance is preserved. PRD-01A/B/C, PRD-02C/D, PRD-03A/B/E and PRD-14A/B/C/D were not repeated wholesale. PRD-01D active generation stays PARTIAL; phone 1014 was not activated.

Starting Git commit: `35b439365729ab2bc0984a148ad6801e14d0258c`. No tracked modifications at start. Unrelated untracked documentation, evidence, browser artifacts and scripts were preserved. The task commit is reported in the final handoff; resolve it with `git log -1 --format='%H %s' -- operations/qa/VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md`. No push. CSV-specific Git attributes preserve the exact CRLF test inputs. Raw native text captures and the converted TSV retain source whitespace, including empty terminal fields; full evidence diff whitespace warnings are expected and were not removed. Documentation whitespace, local links, JSON parsing, CSV structure and private known-credential scans passed; see artifact-validation.json.

All SSH used noninteractive `ssh -T -p 22222`; the inspected remote HISTFILE was empty. No interactive root shell, root history edit, SQL write, generated Asterisk-file edit or upstream source patch was used. Private authorized credentials were read from local private files without printing or committing them. All fixture mutations went through native VICIdial forms.

Playwright MCP was available, but its code execution context could not read the authorized private credential file (`require` and dynamic import unavailable). Existing local Playwright/Chromium was used for visible native forms and controls; no framework/package was installed for browser automation. DOM reads were used for inspection; configuration and agent changes used native form submissions, checkboxes and links. Screenshots and sanitized runtime evidence are in [evidence/functional-gaps-01](evidence/functional-gaps-01/).

Backup health returned success, with the existing integrity-verified backup `20261109T084003-23e8842a` completed 08:40:17 UTC, before task changes. Captured health evidence is in [before.json](evidence/functional-gaps-01/before.json) and [after.json](evidence/functional-gaps-01/after.json). The captured first health check is timestamped 15:39 UTC, after dependency installation and before subsequent configuration changes; it establishes that the earlier backup remained healthy, not that a fresh pre-install snapshot was created. No VM snapshot or database restore was performed.

## Confirmed CSV failure and correction

`/var/www/html/vicidial/sheet2tab.pl` version 2.12 imports Spreadsheet::Read at line 44 and Spreadsheet::XLSX at line 45 unconditionally. Read was missing; XLSX was also absent. Text::CSV_XS 1.65 already existed. Text::CSV_PP was absent but was not required with XS available.

A harmless separate `/tmp/gap-before.csv` probe reproduced converter exit 2 with `Can't locate Spreadsheet/Read.pm in @INC`, followed by compilation abort at line 44. It did not retry or duplicate leads 9–28. See [conversion-before.txt](evidence/functional-gaps-01/conversion-before.txt) and [dependency-before.txt](evidence/functional-gaps-01/dependency-before.txt).

Native CSV path: uploaded temporary file → `sheet2tab.pl` → tab-delimited output → fifth-generation custom mapping → native lead insertion. The converter reads CSV with Spreadsheet::Read/CSV backend, chunks input, writes tab rows and scrubs unsafe delimiters/punctuation. TSV bypasses it. On conversion failure, the loader continued to `fopen()` the missing result and then `fgets(false)`, causing the PHP TypeError at line 3038. This is a downstream error, not evidence that PHP itself cannot import CSV. See [source-paths.txt](evidence/functional-gaps-01/source-paths.txt).

Configured repositories were checked and did not provide Spreadsheet::Read or Data::Peek. Installed only:

| Component | Version / provenance |
|---|---|
| Spreadsheet::Read | 0.95, pinned official CPAN release, HMBRAND, Perl 5 license |
| Data::Peek | 0.54, pinned official CPAN release, HMBRAND, Perl 5 license; required Read dependency |
| perl-Spreadsheet-XLSX | 0.20-1.el10_2, signed configured EPEL RPM |
| perl-Test-Warnings | 0.033-3.el10, signed configured CRB RPM, required build tests |
| perl-Test-NoWarnings | 1.06-10.el10, signed configured CRB RPM, required build tests |

No repository was added. CPAN archives were pinned and SHA-256 verified against official MetaCPAN metadata; dependencies and build instructions were reviewed before bounded, noninteractive builds. No automatic CPAN resolver, system Perl replacement, CPAN configuration change or optional parser installation. Existing compiler/build tools were reused. Site installs kept RPM-owned Perl intact. Data::Peek: 271 tests PASS; Spreadsheet::Read: 3,885 tests PASS. See [dependency record](../dependencies/VICIDIAL_PERL_CSV_IMPORT.md), [provenance](evidence/functional-gaps-01/cpan-provenance.json), both install logs and [installed paths/signatures](evidence/functional-gaps-01/provenance-installed.txt).

As the `apache` user, the unchanged converter generated 21 rows × nine fields from a synthetic CRLF CSV containing quoted commas, empty fields and UTF-8. Output was parsed and asserted locally. See [conversion-after.txt](evidence/functional-gaps-01/conversion-after.txt) and [converted.tsv](evidence/functional-gaps-01/converted.tsv). Actual native uploads subsequently succeeded without the PHP TypeError. Failure handling for a future converter failure is still upstream behavior; it was not patched.

## Native CSV import, mapping and status evidence

Created two previously unused native lists associated with QAOUT01, preserving original list 910014 and leads 9–28:

| Batch | Native list | Lead IDs | Purpose / final state |
|---|---:|---|---|
| GAP01_001–020 | 910015 | 29–48 | First dependency-corrected CSV import; exposed existing charset configuration error; inactive; 20 NEW retained for diagnosis |
| GAP01B_001–020 | 910016 | 49–68 | Clean import after native UTF-8 setting correction; inactive; 19 NEW and lead 50 NI after lifecycle recovery |

Fixtures: [diagnostic CSV](fixtures/prd-functional-gaps-01.csv), [clean CSV](fixtures/prd-functional-gaps-01-utf8.csv). Each contains a header and 20 entirely synthetic records; no status field is claimed as importable. Phone strings 0000000201–0000000220, example.invalid email addresses and NO CALLS comments are non-customer data. All 20 full 2221-prefixed destinations were inspected and had no dialplan extension. See [timestamps-and-routing.json](evidence/functional-gaps-01/timestamps-and-routing.json).

Native interface: `vicidial/admin_listloader_fifth_gen.php`, custom layout. Mappings, zero-based: vendor_lead_code 0; phone_code 1; phone_number 2; first_name 3; last_name 4; list_id 5; email 6; comments 7; city 8. Every unused target cleared to -1. DUPLIST, USA/Canada NONE, invalid-phone replacement DISABLED. See [final mapping](evidence/functional-gaps-01/csv-utf8-mapping-final.json).

For each new list, actual CSV upload → native mapping → processing completed with good 20 / bad 1 / total 21 / invalid 1. The header was rejected as an invalid phone row. A duplicate upload with DUPLIST completed good 0 / bad 21 / duplicates 20 / invalid 1; list count stayed 20. Duplicate protection is list-scoped; using a separate list intentionally permits the same synthetic phones. No duplicate-mode bypass or SQL correction was used. Evidence: [first clean import](evidence/functional-gaps-01/csv-utf8-result.txt), [duplicate import](evidence/functional-gaps-01/csv-utf8-duplicate-result.txt), corresponding PNGs, and the diagnostic batch's matching evidence.

Dependency correction alone exposed another native configuration issue. With `system_settings.use_non_latin=0`, the converter emitted correct UTF-8 but the loader's connection interpreted it as Latin-1: José became JosÃ©, HEX `4A6F73C383C2A95F303031`. Native UI could mask this through inverse connection interpretation, so page appearance was insufficient evidence.

Through native System Settings, changed **Use Non-Latin (UTF8) 0 → 1**. Private before/after form comparison found only that control changed. The unchanged loader enables `SET NAMES UTF8` when this setting is positive. The separate clean batch then stored José correctly, HEX `4A6F73C3A95F303031`; München `4DC3BC6E6368656E`. All 20 rows, every mapped field, empty cities and NEW statuses matched the declared fixture. Native Lead Details and agent preview displayed the expected data. See [charset change](evidence/functional-gaps-01/charset-change.json), [initial SQL field verification](evidence/functional-gaps-01/csv-field-verification.json), [native lead screenshot](evidence/functional-gaps-01/imported-utf8-lead-native.png) and [preview screenshot](evidence/functional-gaps-01/preview-50-settled.png). Diagnostic rows were retained inactive rather than rewritten.

The converter intentionally removes double quotes and semicolons. Expected comment after conversion is `SYNTHETIC ONLY, quoted CSV NO CALLS`; quoted comma parsing passes. This does not claim lossless punctuation or multiline-field support.

Status finding: the installed fifth-generation loader initializes imported leads to NEW and offers no custom status target. The prior TSV source's status column was auto-guessed as State; subsequent NI/DNC states came from native administrative actions. This was unsupported mapping/operator workflow, not demonstrated failure to preserve a supported status mapping. New fixture expectation is explicitly all NEW at import; final lead 50 NI is a separately recorded lifecycle recovery, not an import discrepancy. Original lead 9 NI and 19 DNC records remain unchanged.

## Agent-state qualification

The original MANUAL screen's absent Ready/Pause controls are expected. The native frontend downgrades INBOUND_MAN to MANUAL when no inbound groups are selected and MI_PAUSE is absent. The campaign option **Max Inbound Calls Outcome = ALLOW_MI_PAUSE** supplies MI_PAUSE. See [agent-state-source.txt](evidence/functional-gaps-01/agent-state-source.txt).

Minimum tested configuration: QAOUT01 INBOUND_MAN, ALLOW_MI_PAUSE, no assigned inbound groups, campaign inbound N, pause codes N, no-hopper logins Y, manual list 910016, PREVIEW_AND_SKIP. Native Administration only exposes the outcome selector when inbound is enabled; it was briefly enabled with no groups, the option saved, then disabled again before login. The native UI normalizes INBOUND_MAN dial level to 1.0; an attempted zero did not persist. This is explicitly recorded in [campaign-pause-config.json](evidence/functional-gaps-01/campaign-pause-config.json), not represented as zero.

The outbound daemon explicitly excludes MANUAL and INBOUND_MAN (`AST_VDauto_dial.pl:1718`, MANUAL DIAL CAMPAIGN, NO DIALING). INBOUND_MAN did not enable automatic outbound dialing. No inbound groups or customer routes were introduced; no customer calls occurred. Final campaign settings are MANUAL / zero / inactive.

Operator confirmed **1000 ready**. Native qaagent14 login connected PJSIP/1000 to ConfBridge 9600000. The visible YOU ARE PAUSED button was clicked to Ready; YOU ARE ACTIVE clicked to Pause; paused clicked to Resume; active clicked to Pause before preview. No hidden JavaScript call was used as a substitute for controls.

| Action | Canonical epoch | Server local time (New York) | Corroboration |
|---|---:|---|---|
| Session 2 login, PAUSED | 1794240079 | 11:01:19 | User LOGIN, agent log 14, PJSIP/1000 Up |
| Ready | 1794240110 | 11:01:50 | Live READY; log 14 pause_sec 31 |
| Pause | 1794240155 | 11:02:35 | Live PAUSED; log 14 wait_sec 45; new log 15 |
| Resume | 1794240191 | 11:03:11 | Live READY; log 15 pause_sec 36 |
| Pause | 1794240234 | 11:03:54 | Log 15 wait_sec 43; new log 16 |
| Logout | 1794240286 | 11:04:46 | User LOGOUT; log 16 pause_sec 52; session/channels absent |

Epoch differences equal stored intervals. Talk/disposition seconds remain zero; this was non-calling acceptance. UI screenshot capture follows each action by a few seconds. Screenshots `session-2-login`, `session-2-ready`, `session-2-pause`, `session-2-resume` and the matching `*-db.json` files corroborate controls/state. [Canonical timestamp evidence](evidence/functional-gaps-01/timestamps-and-routing.json) contains final durations. Pause codes were unnecessary for these state transitions; no additional code set was created.

## Preview / INCALL investigation

Lead 49: native Lead Preview checkbox and Dial Next Number reserved the existing list lead, displayed synthetic data and set INCALL. No DIAL LEAD action was invoked. Visible SKIP LEAD restored NEW, called_since_last_reset Y and called_count 0. UI reported “Lead skipped, go on to next lead”; the agent log opened an MDSKIP pause row. Normal logout completed. Later evidence still shows lead 49 NEW and no retained claim. Evidence: [claim](evidence/functional-gaps-01/preview-claim-db.json), [skip UI](evidence/functional-gaps-01/preview-skip.png), [skip/logout UI](evidence/functional-gaps-01/session-1-logout.txt), [final state](evidence/functional-gaps-01/final-state-db.json).

Lead 50 comparison: in the second session, preview displayed GAP01B_002 / QA_002 / 0000000202. Clicking LOGOUT directly, with no Skip, completed logout and removed all agent/conference channels, but lead 50 remained INCALL at the post-logout observation (about 23 seconds afterward) and subsequent native administrative inspection. See [preview](evidence/functional-gaps-01/preview-50-settled.png), [logout](evidence/functional-gaps-01/preview-unreleased-logout.txt), [post-logout SQL/runtime](evidence/functional-gaps-01/preview-unreleased-logout-db.json).

Native NormalLogout/LogouT sends userLOGout without performing ManualDialSkip. Source review found no preview release in the inspected logout branch; Skip separately restores the preview's prior status. No supported configuration correction for bare-logout release was identified. This is not proof that all future revisions or background cleanup jobs never clear a claim; it is a reproducible short-term stale reservation on the tested normal logout path. A test of eventual cleanup over an extended idle period was not run.

Required safe operator workflow is **SKIP LEAD → verify release → LOGOUT**. That normal workflow passes. Bare LOGOUT is not equivalent to Skip. Administrative recovery is distinct: Lead Details did not offer NEW for the INCALL record, so the administrator selected available NI for synthetic lead 50, saved/reopened and verified. This recovery is not counted as automatic logout cleanup. See [restored lead screenshot](evidence/functional-gaps-01/lead-50-native-restored.png). No source or SQL patch was applied. GAP-10 remains PARTIAL because the successful explicit-release path does not establish unconditional logout cleanup; the observed negative branch is retained in full.

## Timestamp investigation and correction

OS and MariaDB used America/New_York (SYSTEM/EST, UTC−5 at test time); VICIdial server local_gmt was −5. PHP's existing `/etc/php.d/99-vicidial.ini` override used Asia/Manila (UTC+8). Native PHP writes DATETIME via `date('Y-m-d H:i:s')` and a paired epoch via `date('U')`. MariaDB DATETIME has no attached timezone; interpreting Manila wall time as New York introduces 13 hours.

Old LOGIN: true epoch **1794236920**, UTC 2026-11-09 15:08:40, Manila 23:08:40, New York 10:08:40. Stored event_date was 23:08:40; MariaDB UNIX_TIMESTAMP(event_date) returned **1794283720**, delta **46,800 seconds**. Agent log 11 has the same mismatch. This is an actual writing/configuration mismatch, not just two legitimate displays. [Investigation](evidence/functional-gaps-01/investigation.json) records the before state; native write sites are captured in agent-state-source.txt.

Small correction: changed only `date.timezone` in the existing dedicated PHP override to **America/New_York**. Preserved `/root/99-vicidial.ini.gap01-before` mode 0600, validated `php-fpm -t`, then reloaded PHP-FPM. All other override bytes/settings stayed unchanged. No OS/MariaDB timezone, server GMT, schema or historical record change. See [timezone-change.json](evidence/functional-gaps-01/timezone-change.json).

New logins/logouts at epochs 1794239589, 1794240008, 1794240079 and 1794240286 all have **UNIX_TIMESTAMP(event_date) − event_epoch = 0**. New agent logs 12–16 have matching server-local dates/epochs, and elapsed states agree. Session 2 LOGIN 11:01:19 New York equals 16:01:19 UTC and **2026-11-10 00:01:19 Manila**. The agent browser header follows browser local time and legitimately shows the next date. See [explicit conversions and SQL](evidence/functional-gaps-01/timestamps-and-routing.json) and [final PHP/DB timezone check](evidence/functional-gaps-01/final-config-verification.txt).

Historical wrong DATETIME values were not rewritten and should be correlated using stored epochs; sorting mixed old/new rows by DATETIME can misorder them. The −5 server field matches the current EST season. Future DST/calling-window qualification remains a separate prerequisite before callbacks/time-rule acceptance; this test does not certify all summer offsets or historical timestamp consumers.

## Separate authentication finding

Confirmed with synthetic native accounts only: a denied level-1 qaagent14 request to `/vicidial/admin.php?ADD=31&campaign_id=QAOUT01`, and denied requests to `/vicidial/admin_listloader_fifth_gen.php`, return a plaintext/HTML denial containing the submitted native password. The response itself includes the value; this is browser-visible response content, also demonstrated in the retained previous browser acceptance. Current evidence stores only a boolean and a redacted context:

```text
Login incorrect, please try again: |qaagent14|[REDACTED]|BAD|
```

The retained SVN BASE source contains the behavior at admin.php:8045 and admin_listloader_fifth_gen.php:766; it is not introduced by the dependency correction. [Source proof](evidence/functional-gaps-01/provenance-installed.txt), [sanitized response/recording checks](evidence/functional-gaps-01/protected-playback-auth.json). A supervisor's allowed admin page also embeds its own native password in JavaScript; this is separate from the denied-response echo and was redacted in evidence.

No native configuration mitigation was identified for the literal password echo. Existing HTTPS/Portal and role enforcement remain operational but do not remove response disclosure. Recommend a separate narrowly authorized upstream-compatible change to omit passwords from denial content, with synthetic positive/negative authentication tests and log/response inspection. Do not redesign authentication or change 6666 under this task. No credential values, cookies or auth headers are retained in evidence.

## PHP/Asterisk errors and targeted regression

[Sanitized PHP review](evidence/functional-gaps-01/php-log-review.json) aggregates severity/source location from 14:30 UTC through final cleanup, interpreting each log line's stated timezone. Exactly one fatal event is retained: the pre-correction native loader TypeError at **14:57:55 UTC**, line 3038. No later fatal/parse event occurred in the tested CSV, campaign, agent or cleanup workflows. Existing undefined-variable/array warnings remain at native admin, style, database connection and report locations. Their presence alone is not classified as failed acceptance: the tested persistence, fields, states and denials were checked independently. Fresh error-priority journal entries for Asterisk/httpd/PHP-FPM were absent in the review window. This is not a comprehensive audit of every application log.

[Regression comparison](evidence/functional-gaps-01/regression-comparison.json), [before](evidence/functional-gaps-01/before.json), [after](evidence/functional-gaps-01/after.json), and telephony-before/after show:

- Protected portal, Apache routing, recording finalizer/playback-related files and retained Asterisk configuration hashes unchanged. test01, lead 8, phone records 1000/1001/1014 and 6666's recording permissions unchanged.
- Existing route 22217000000001 still resolves to Dial(SIP/1001,45,To). PJSIP/1000 was exercised for the new conference; SIP/1001 configuration/peer inspection was read-only. No fresh customer media test.
- Recording 2 remains 56 seconds, end_epoch 1793990673, same location and file size 892,204 bytes; SHA-256 `f726a1d5615d50767c8a0f4794f68a544322ffa70037d1ef1a7bb3667aa4be59` unchanged. HEAD from native administrator returned 200 audio/wav; both restricted accounts returned 403. No audio downloaded into evidence.
- Native role/configuration rows remained restricted; qaagent14 direct admin/loader denial and qasup14 loader denial remained enforced. Shared source IP did not confer recording or import permission. Existing broader group-visibility acceptance is retained, not repeated in full.
- Apache syntax OK; HTTPD, Asterisk, MariaDB, PHP-FPM and portal main PIDs unchanged. PHP-FPM was reloaded once. Recording, backup, portal-apply and TLS-renewal timers active. Backup health success. SSH remained reachable on 22222 throughout.
- Permanent firewall equal. nft rule text equal after excluding expiring grant lifetimes and packet/byte counters; those runtime values changed normally. No portal/firewall authorization mutation was performed. Personal/Shared Office portal UX was not re-enacted; accepted deployment/configuration and live grant behavior were preserved.
- QA customer call count 0, QA recording count 0, live agents 0, active channels/calls 0. Fresh call/recording/audio regression **NOT_RUN**, with no customer-call authorization requested or inferred from “1000 ready.”

## Acceptance matrix

PASS means the specified bounded result has corresponding executed evidence. Prior accepted features are retained separately from fresh end-to-end call acceptance.

| ID | Result | Evidence / limits |
|---|---|---|
| GAP-01 | PASS | conversion-before.txt; missing unconditional Read/XLSX imports identified, XS available, PP optional |
| GAP-02 | PASS | cpan-provenance.json; RPM/build logs; 271 + 3,885 tests; dependency record and packlists |
| GAP-03 | PASS | apache-user converter result, converted.tsv; actual native CSV mapping reached |
| GAP-04 | PASS | csv-utf8-result.txt/png; isolated list 910016, leads 49–68, 20 CSV records |
| GAP-05 | PASS | csv-field-verification.json; all declared fields/initial NEW statuses; native screenshot; UTF-8 correction documented |
| GAP-06 | PASS | csv-utf8-duplicate-result.txt/png; 20 duplicate rejects, zero additional leads; predictable header invalid row |
| GAP-07 | PASS | session-2 visible control screenshots and READY/PAUSED DB captures; INBOUND_MAN/MI_PAUSE scope explained |
| GAP-08 | PASS | timestamps-and-routing.json; logs 14–16 and exact 31/45/36/43-second intervals |
| GAP-09 | PASS | lead 49 preview → visible SKIP LEAD → NEW; called count 0; native MDSKIP log |
| GAP-10 | PARTIAL | Skip-before-logout releases correctly; **direct logout of previewed lead 50 leaves INCALL** despite session/channel cleanup; no supported automatic release correction found; native NI recovery is not a fix |
| GAP-11 | PASS | old +46,800-second error reproduced; PHP override corrected; four new LOGIN/LOGOUT pairs zero delta, explicit UTC/zone conversions |
| GAP-12 | PASS | unchanged roles/groups/6666; native denied agent admin/import, denied supervisor import; protected recording 403; previously accepted provisioning retained |
| GAP-13 | PASS | targeted preservation: unchanged route/config/recording hashes/rows, services and protected HEAD; fresh customer call/audio/finalization NOT_RUN |
| GAP-14 | PASS | campaign-final-native.json; lists-final-native.json; final-state-db.json; final-config-verification.txt; inactive MANUAL/0, no live agents/channels/INCALL, 1014 suspended |

GAP-10 is a demonstrated limitation, not an unexecuted test. Supported explicit release and ordinary session cleanup function; unconditional open-preview logout cleanup remains unaccepted. That distinction is why the overall verdict is PARTIAL rather than VERIFIED. No remediation regression was observed.

## Exact changes and final fixture

Persisted host changes: the five bounded dependency components above; native use_non_latin 0→1; dedicated PHP timezone override Manila→New York. Native fixture additions: lists 910015/910016 and 40 synthetic leads 29–68. No new users/phones/campaign were needed. Temporary QAOUT01 INBOUND_MAN/MI_PAUSE/inbound/no-hopper/active settings were restored; permanent synthetic manual-list association now points to clean list 910016. Lead 49 was skipped normally; lead 50 was set NI through Lead Details after the comparison.

Final state: QAOUT01 inactive MANUAL level 0, inbound N, max outcome DEFAULT, no-hopper logins N, PREVIEW_AND_SKIP, manual list 910016. Lists 910014/15/16 inactive. Original list unchanged: one NI / 19 DNC. Diagnostic list: 20 NEW. Clean list: 19 NEW / one NI (lead 50). Test accounts retain prior active/restricted roles. Phone 1014 N/SUSPENDED. No synthetic INCALL, live agent or Asterisk channel. Full inventory and reuse procedure: [fixture document](VICIDIAL_OUTBOUND_TEST_FIXTURE.md).

Operator confirmation is limited to extension 1000 readiness. The connected phone session is evidenced by PJSIP/1000 Up in ConfBridge; no separate operator confirmation of answering is claimed. No operator confirmation of two-way audio, new recording or final business UX was claimed. Local browser engineering evidence is not independent QA.

## Rollback and remaining work

Deactivate synthetic campaign/lists before any rollback. Prefer retaining data inactive; do not delete production history or restore an entire database.

1. Native Campaigns → QAOUT01: inactive N, MANUAL, zero, inbound N, outcome DEFAULT, no-hopper N; preserve PREVIEW_AND_SKIP. Native Lists → all three inactive. If restoring the original fixture association, select manual list 910014 through the campaign form and save/reopen. Only enable 910016 for a later authorized clean-data test.
2. Disable qaagent14/qasup14 through Users if retiring; leave 6666 unchanged. Revoke only QAPRD14-specific rights if retiring the group. Keep 1014 suspended. Identify leads by list ID plus vendor prefix; retain diagnostic mojibake rows isolated.
3. Any synthetic open-preview claim must be released by SKIP LEAD before logout. If already logged out, use native Lead Details and an available appropriate disposition; final lead 50 recovery used NI. Do not use SQL. Recheck live agents/channels/INCALL after cleanup.
4. Encoding rollback, if explicitly needed: native System Settings Use Non-Latin 1→0, save/reopen. This reintroduces Unicode storage risk and does not repair historical data. No such rollback was performed.
5. Timezone rollback: restore only `/etc/php.d/99-vicidial.ini` from `/root/99-vicidial.ini.gap01-before`, check PHP-FPM configuration and reload. This reintroduces the demonstrated mismatch; it does not undo new log records. Never change OS/MariaDB timezone or rewrite history just for this rollback.
6. Dependency rollback: review and remove only files owned by the two site-module packlists; review an RPM removal transaction for only newly installed XLSX/Test-Warnings/Test-NoWarnings. Preserve pre-existing Perl/CSV_XS and stop on unrelated removals. Full instructions and upgrade implications are in the dependency record. No Asterisk/MariaDB restart or global database restore.

Remaining bounded decisions: accept/enforce the native Skip-before-logout operator procedure or separately authorize a narrowly reviewed upstream preview-logout correction; separately remediate the password-reflecting native denial; qualify DST/callback/calling-window behavior and historical timestamp consumers before scheduling features. Fresh call/recording regression remains separately authorized work. Active phone 1014 generation, webphone and all excluded dialing features remain outside this remediation.

Final verdict: **VICIDIAL_PRD_FUNCTIONAL_GAPS_01_PARTIAL**.
