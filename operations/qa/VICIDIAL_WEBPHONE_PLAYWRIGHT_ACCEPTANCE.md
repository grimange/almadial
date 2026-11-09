# PRD-WEBPHONE-00 — VICIphone and Playwright acceptance

## Executive result

**VICIDIAL_WEBPHONE_PLAYWRIGHT_VERIFIED**

One isolated browser phone, SIP/1002, registered through the existing trusted IP-only HTTPS origin, joined VICIdial ConfBridge 9600001, transmitted deterministic microphone audio and completed one answered local simulator call to SIP/1001. The operator confirmed hearing the browser tones and sending a phrase back. Browser/Asterisk media observations and the resulting native recording support both media directions. Two fresh CLI Playwright sessions subsequently registered, joined, transmitted, explicitly unregistered and logged out without Zoiper operation or customer calls.

This is functional acceptance on the observed browser/network path, performed by the implementation worker. It is not independent QA, a complete WebRTC certification, or promotion of AlmaDial's candidate stack. Native background telemetry errors and untested reconnect/revocation paths remain disclosed below. No upstream VICIdial PHP/Perl or schema was changed.

Evidence window: **2026-11-09 16:44–17:43 UTC**, corresponding to November 10 in the operator's Asia/Manila timezone. The final cleanup followed the repeated tests. Host-local time was America/New_York. New native user-log DATETIME/epoch pairs agree; the browser header legitimately displayed Manila time. The preceding timestamp correction was preserved.

## Preserved baseline and preflight

Reviewed the system audit, product plan, [PRD-01/02/03/14 acceptance](VICIDIAL_PRD_01_02_03_14_ACCEPTANCE.md), [functional-gap report](VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md), [outbound fixture](VICIDIAL_OUTBOUND_TEST_FIXTURE.md), recording playback/finalization reports and recording operations documentation. Their accepted provisioning, CSV, role, SIP/media and recording results were retained. The open preview/logout INCALL defect was not changed.

Target: dialer01, AlmaLinux 10.2, VICIdial 2.14b0.5 retained SVN r4037, Asterisk 20.21.0, existing MariaDB/Apache/PHP-FPM. All remote commands used noninteractive SSH on TCP 22222; no shell history or startup file was modified.

[Preflight](evidence/webphone-00/preflight.json) establishes HTTP initially disabled, working chan_sip, crypto/SRTP/WebSocket modules, existing PJSIP UDP 5061 and SIP UDP 5060, installed Apache proxy/Lua modules, and unused phone 1002. One initial inventory query used the nonexistent table name `phone_templates`; that read-only query failed and is retained transparently. Native forms and subsequent installed-schema queries supplied the actual inventory. It was not used as provisioning evidence.

The initial [health capture](evidence/webphone-00/before.json) reports a successful integrity-verified backup `20261109T084003-23e8842a`, approximately 1.36 GB, five payloads, 21,340 archive members, 376 database tables and seven recording files. This was an existing backup, not a new VM snapshot. Scoped root-only configuration backups were taken before deployment; [backup hashes](evidence/webphone-00/scoped-backup.json) identify their originals.

The initial Git HEAD was `d851d86279e2611b3fa70d57834f5cc938d8e200`. Unrelated untracked installer work, review documents, QA evidence and local browser output were preserved. A concurrent, unrelated M2 commit (`36c1b6e`) was preserved as this milestone's parent. Only this milestone's implementation, synthetic fixtures and sanitized evidence are included in its commit. No push was performed.

## Selected implementation

Pinned [VICIphone upstream](https://github.com/vicimikec/ViciPhone) revision **24e98fccfa7b7079fff970e02fca5f002ac1f2cc**, VICIphone 3.0.0, bundled SIP.js 0.20.1, AGPL-3.0. The 44 deployed non-debug files, including upstream LICENSE, match their pinned source hashes: [asset comparison](evidence/webphone-00/asset-comparison.json). No upstream modification or external hosted VICIphone credential dependency was introduced.

Selected **native chan_sip WebRTC** because this build's chan_sip already owns the SIP WebSocket subprotocol and the PJSIP WebSocket module is not running. Existing PJSIP/1000 and SIP/1001 remain on their accepted drivers and transports. The native phone template enables ulaw/PCMU, ICE, DTLS-SRTP, AVPF, symmetric RTP, RTCP multiplexing and disables direct media. This follows the installed driver and [VICIphone 3.0 requirements](https://viciphone.com/?page_id=353); the [Asterisk PJSIP guide](https://docs.asterisk.org/Configuration/WebRTC/Configuring-Asterisk-for-WebRTC-Clients/) was used for requirements, not as a chan_sip template.

Architecture: browser `wss://2.29.153.161/ws` → existing Apache 443 → private `127.0.0.1:8088/ws`. Asterisk HTTP was enabled on its preexisting loopback bind. Only the exact WebSocket path is proxied; ARI, AMI and other HTTP handlers were not published. No TCP 8089 service or firewall rule was added. Actual Chromium 149.0.7827.55 and OpenSSL validated the existing Let's Encrypt IP SAN certificate. Neither certificate validation nor browser security was disabled.

The WebSocket upgrade requires both the existing confirmed web grant and a worker-confirmed phone grant for the actual source IP, followed by independent SIP credentials. A narrow worker projection publishes only confirmed phone grants with a maximum 35-second freshness horizon; root:apache directory/file modes are 0750/0640. The portal's authentication, personal/office grant semantics, firewall reconciliation and recording authorization were unchanged. [Local projection tests](../../tests/webphone/test_phone_projection.py), actual authorized browser registration, and the ungranted-source 403 support the gate. An internal WebSocket probe returned HTTP 101 with SIP subprotocol; its curl exit 28 reflects the deliberately bounded open upgraded connection, not handshake failure. See [final runtime](evidence/webphone-00/final-runtime.json).

The [operations README](../webphone/README.md) contains exact configurations, paths, launch commands, native values, license/source pin, upgrade considerations and rollback. No new framework, gateway, npm dependency, RPM or service was installed.

## Native fixture and configuration persistence

Phone, template, settings container, server/system webphone fields, list and lead were created or edited using authenticated native Administration. Save/reopen and live generator results were checked. No SQL INSERT/UPDATE/DELETE was used.

| Object | Exact identity and configuration |
|---|---|
| Phone | **1002**, SIP, active Y/ACTIVE, QAPRD14; native login/dialplan/voicemail 1002; `QA WEB00 browser automation` |
| Template | **WEB00_SIP**, exact [template file](../webphone/config/WEB00_SIP.template); generated in sip-vicidial.conf |
| Settings container | **WEB00_SETTINGS**, WEBPHONE_SETTINGS, QAPRD14; `dialRegExten:1`, `autoGain:0`, `echoCan:0`, `noiseSup:0` |
| Phone web settings | Webphone Y; external server IP Y; auto-answer Y; debug N; qualify N; dialpad/dialbox N |
| Agent | Existing **qaagent14**, QAPRD14; no new account or permission escalation |
| Native server fields | External IP 2.29.153.161; internal/external Web Socket URL `wss://2.29.153.161/ws` |
| Native system field | Webphone URL `https://2.29.153.161/viciphone-web00/viciphone.php` |
| Campaign | Existing **QAOUT01**, temporary manual test settings; restored inactive MANUAL/0 afterward |
| New list | **910017**, `WEB00 local simulator only`, QAOUT01; final inactive N |
| New lead | **69**, WEB00_001, Webphone Synthetic, web00@example.invalid, local simulator 7000000001; final NI/called_count 2 |
| DTLS identity | Isolated webphone00 certificate/key under /etc/asterisk/keys; key not exported |

The new one-row [CSV](../../tests/webphone/fixtures/simulator.csv) completed native import with Good 1 / Bad 0 / Total 1: [import result](evidence/webphone-00/simulator-import.txt). It was not mixed with leads 9–68. [Native form readback](evidence/webphone-00/native-readback.json) and [SELECT/runtime evidence](evidence/webphone-00/final-runtime.json) corroborate persisted associations and settings.

An initial server-originated WebSocket conference callback did not complete. The supported native settings-container option `dialRegExten:1` made VICIphone dial its assigned conference after registration and resolved that integration path. This is conference connection, not customer automatic dialing. Generated configuration was inspected, never hand-edited.

## Browser, conference and microphone evidence

The actual Playwright MCP browser run established Registered → Established → ICE connected / DTLS connected, conference 9600001 and PCMU at 8 kHz: [session 1](evidence/webphone-00/session-1.json), [session 2](evidence/webphone-00/session-2.json). Asterisk showed SIP/1002 in the expected ConfBridge; the native agent session matched qaagent14/QAOUT01.

The original 12-second mono PCM fixture repeats 440/660/880 Hz with fades, generated by [make_audio.py](../../tests/webphone/make_audio.py). Chromium used fake-media-device, fake-UI and absolute-file microphone flags with microphone permission granted; autoplay policy was set without weakening TLS or origin security. Positive browser microphone energy, increasing outgoing RTP and the tone sequence in Asterisk's finalized recording prove the fixture reached Asterisk.

The [CLI repeat run](evidence/webphone-00/repeat-run.json) executed the final reusable workflow twice in fresh Chromium contexts. Each cycle asserted SIP registration, established conference media, outbound RTP beyond 16,000 bytes, native Unregister confirmation, native logout completion and browser closure. All four inspected browser states reported empty pageerror arrays in this final run. `customerCalls:0` applies to those two repeat cycles, not the earlier authorized call.

Closing an earlier browser immediately after logout left a SIP contact until expiry. The final test explicitly clicks the native Unregister control and waits for Unregistered **before** logout hides the iframe. The final host readback shows phone 1002 `(Unspecified)`, no active agent row, no conference and zero channels/calls. No stale registration is claimed as successful cleanup.

The default runner requires an already authorized web/phone IP grant; native Secure Access Portal login/grant remains an operator/MCP bootstrap step. It does not automate credential reset or circumvent that policy. The CLI does not need Zoiper to operate the agent phone. It does not claim autonomous grant renewal or recovery after network loss.

## Controlled local call and recording

Read-only routing preflight established the exact `22217000000001@default → Dial(SIP/1001,45,To)` route and that all carrier records were inactive. No external route was introduced. The existing Linphone 1001 endpoint was registered and the operator was ready to answer.

An initial manual request inherited omit-phone-code N, producing `222117000000001`, which did not match the simulator route and created no customer SIP leg. It was hung up and NI dispositioned; recording **3** contains only that agent conference attempt. The campaign mismatch was corrected through native Administration by setting **omit-phone-code Y**. No second answered/customer call was hidden in the evidence.

The corrected request produced exactly one answered SIP/1001 simulator call. See [sanitized observations and operator confirmation](evidence/webphone-00/call-observations.md), [browser media](evidence/webphone-00/call-media.json), [SQL logs](evidence/webphone-00/final-runtime.json), [recording analysis](evidence/webphone-00/recording-4-analysis.txt) and [protected playback](evidence/webphone-00/protected-playback.json).

| Correlation | Result |
|---|---|
| Lead / campaign / agent | 69 / QAOUT01 / qaagent14 |
| UI identifier | M1091220450000000069 |
| Native call uniqueid | **1794244845.86** |
| Browser / customer channels observed | SIP/1002-00000005 / SIP/1001-00000006 |
| Conference | 9600001 |
| Call talk interval | Epoch 1794244852–1794244889, **37 seconds**, NI; agent log 22 also talk_sec 37 |
| Recording | **4**, 20261109-122045_7000000001 |
| Recording interval | Epoch 1794244845–1794244889, logged 44 seconds; WAV **43.84 seconds**, 8 kHz mono 16-bit |
| Finalization | Existing timer populated protected location automatically; no finalizer change or manual SQL repair |
| Tone analysis | 250-ms windows detected 440 Hz 58 times, 660 Hz 57, 880 Hz 60; sequence agrees with original microphone |
| Playback | Native Lead Details link → entitled 6666 browser playback, readyState 4, currentTime > 0, no media error |
| Recording restrictions | qaagent14 **403**, qasup14 **403** for recording 4 |

The wider recording interval includes setup before the 37-second talk interval; this is consistent, not a log-duration mismatch. Browser transport was DTLS connected with AES_CM_128_HMAC_SHA1_80, with positive inbound audio energy and both RTP directions. The operator's combined **YES** confirms hearing the tones and sending a phrase. It was not a separate detailed assessment of both endpoints' perceived sound quality. No customer/private recording, cookie, profile, password or TLS key is committed.

## Error review and remaining limitations

[Sanitized PHP review](evidence/webphone-00/php-log-review.json) retains severity, canonical times and source locations, excluding values and SQL text.

1. At 17:07:11 UTC an early automation submission reached agc/vicidial.php:5812 before JS_browser_height was populated, causing a string-minus-integer TypeError. The workflow now waits for the normal positive field value and actual navigation. Subsequent complete sessions pass; no upstream patch was made.
2. At 17:33:29 and 17:33:42 UTC, agc/conf_exten_check.php:1751 called the native mysqli helper at agc/functions.php:2787 and hit duplicate key **visibleuser** in **vicidial_agent_visibility_log**. Source inspection identifies this as browser visibility telemetry. It did not prevent the observed registration, conference/media, SIP unregister, native logout events or cleanup. This is an unresolved native PHP/SQL compatibility error in auxiliary telemetry; no schema/source change or claim of error-free native operation is made.
3. Earlier exploratory pageerror TypeError events were observed; the final repeated browser runs returned no pageerror events. Existing undefined-value PHP warnings, including VICIphone optional parameters, remain. Runtime outcomes and persisted state were checked independently rather than classifying all warnings as failures.
4. Some unused ICE/STUN candidate checks logged permission failures, while the selected pair connected and carried the tested media. TURN, restrictive NAT, connection-loss recovery, device switching and alternate browser versions remain untested.
5. The WebSocket gate checks authorization at upgrade. Forced closure of an already open WebSocket on grant expiry/revocation was not implemented or tested. Existing SIP/RTP firewall revocation logic was unchanged; the local projection tests establish removal of the handshake grant, not active-call termination.
6. The prior open-preview/logout INCALL defect remains out of scope. This call was hung up and dispositioned before logout; no unintended INCALL claim remains.
7. One raw exploratory HTML diagnostic exposed the synthetic phone's native login password in tool output. It was acknowledged and **only phone 1002's login password was rotated through Administration**, with the private local credential file updated. No secret value is reproduced in evidence or committed. Existing phone credentials and SIP1002's separate secret were preserved. Raw iframe URLs, traces and HTML must not be collected.

No general infrastructure remediation, source alteration or authentication redesign was attempted. A separate bounded native-telemetry investigation is appropriate before relying on visibility reports; it does not invalidate the demonstrated phone/media lifecycle.

## Acceptance matrix

| ID | Result | Executed evidence / scope |
|---|---|---|
| WEB-01 | PASS | Installed modules, transports, codecs and Apache prerequisites in preflight.json; chan_sip selected for actual ownership |
| WEB-02 | PASS | Actual Chromium IP-origin registration/media; final-runtime.json trusted OpenSSL IP verification; no TLS bypass |
| WEB-03 | PASS | Private /ws HTTP 101, public ungranted 403, actual authorized Chromium SIP/WSS session |
| WEB-04 | PASS | Native phone/template/settings save and reopen in native-readback.json; generated SIP peer inspected |
| WEB-05 | PASS | Pinned embedded VICIphone loaded in actual Chromium; session-1/2 and repeat-run.json |
| WEB-06 | PASS | Native Registered state and Asterisk WS peer, repeated in fresh browser contexts |
| WEB-07 | PASS | Established conference 9600001 in browser/Asterisk; matching native agent session |
| WEB-08 | PASS | Positive synthetic source energy, outgoing RTP and expected three-tone sequence in finalized Asterisk recording |
| WEB-09 | PASS | One answered local SIP1001 call; exact isolated route, native call/agent logs and NI disposition |
| WEB-10 | PASS | Operator combined affirmative, both RTP directions, inbound browser energy and recorded browser tones; no separate quality certification |
| WEB-11 | PASS | Uniqueid 1794244845.86 / lead69 / agent log22, recording4 automatic finalization and entitled native playback; restricted users denied |
| WEB-12 | PASS | Final runner completed two fresh register/conference/media/unregister/logout cycles without manual softphone control; existing portal grant prerequisite |
| WEB-13 | PASS | Both Unregistered results, native logout events, final zero live agents/channels/conferences, no stale INCALL lead or SIP1002 contact |
| WEB-14 | PASS | Scoped baseline comparisons, retained endpoint registrations/configuration, new protected recording checks, 51 existing portal tests and healthy timers; no repeated full human Zoiper certification |
| WEB-15 | PASS | Actual three-file rollback restored original hashes/HTTP-disabled state, then reapplied deployed hashes with reloads; native temporary campaign values restored; isolated artifact deletion documented but intentionally not executed |

## Regression and tests

[Before](evidence/webphone-00/before.json), [after](evidence/webphone-00/after.json) and [comparison](evidence/webphone-00/regression-comparison.json) show unchanged portal authentication/access code, recording processor/config/timer, certificate deployment code, cron and the five checked upstream native admin/playback files. Only the expected Apache vhost changed among those files. Original recording 2 bytes/ownership/mode and its database metadata, user 6666 recording permissions and checked portal DB grants remain unchanged.

SIP and PJSIP base configuration hashes match the scoped originals. SIP1001's generated section matches its original; all **140 effective PJSIP1000 parameters** match the retained pre-provisioning acceptance baseline in [pjsip-1000-comparison.json](evidence/webphone-00/pjsip-1000-comparison.json). Existing endpoints remained registered/available; SIP UDP5060 and PJSIP UDP5061 remained present. Current test01 remains its accepted MANUAL/0 settings and exact SIP1001 route. No full pre-task hash of every test01 database column is claimed.

Permanent firewall configuration is identical. Runtime grant expiry counters naturally changed; no byte-for-byte equality of dynamic nftables leases is claimed. Main PIDs for Asterisk, MariaDB, Apache, PHP-FPM, portal, SIP guard and fail2ban match before/after captures. No full Asterisk/MariaDB restart occurred. Recording, backup, portal-apply and TLS renewal timers remain active. SSH22222 remained reachable and backup health remained successful.

Executed local checks:

- Four exact projection-block unit tests: PASS, including web-only/unconfirmed denial, freshness cap, modes, revocation and failed-reconfirmation removal.
- Existing portal/office/recording/SIP guard tests: **51 PASS**, [test output](evidence/webphone-00/portal-regression-tests.txt). The initial sandbox run could not open its loopback test servers; rerun with the authorized loopback capability passed.
- Node syntax check for runner and Python compilation for the small helpers: PASS.
- Actual MCP Chromium registration/conference/call/media and the final CLI two-cycle workflow: PASS, separate from local/mock tests.

## Final state and rollback

[Final native campaign readback](evidence/webphone-00/final-campaign.json), [screenshot](evidence/webphone-00/final-campaign.png) and [runtime/SELECT evidence](evidence/webphone-00/final-runtime.json) establish:

- QAOUT01 **inactive**, MANUAL, auto level **0**, no-hopper-login N, ONDEMAND, omit-code N, original `12pm-5pm` call time, original manual list 910016.
- Lists 910014–910017 **inactive**. Lead69 **NI**, called_count2. No INCALL claim in these lists.
- Phone1002 retained active for reuse, **unregistered**; phone1014 **N/SUSPENDED**. Restricted qaagent14/qasup14 unchanged.
- Zero live agents, active channels, active calls or ConfBridge participants. No external or automatic call was placed.
- Lead8, recording2, user6666 privileges and working endpoint1000/1001 settings preserved.

Rollback instructions are in [operations/webphone/README.md](../webphone/README.md#scoped-rollback). The guarded [scoped_config.py](../webphone/scoped_config.py) and [actual rollback/reapply evidence](evidence/webphone-00/scoped-rollback.txt) cover the three core file changes. Restore only those captured files after drift/zero-channel checks, then use native Administration to disable the synthetic phone, isolate its template/container and restore previously blank global webphone/server fields if no later consumer depends on them. Remove only the isolated assets/key/projection after checking references. Preserve call history and existing portal grants; never restore the whole database to remove a fixture.

The commit containing this report is available with `git log -1 --format=%H -- operations/qa/VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md`; its exact identifier is returned in the engineering handoff. Unrelated working-tree changes remain outside it. The retained endpoint is ready for separately authorized future milestones, with no automatic dialing enabled.

Final verdict: **VICIDIAL_WEBPHONE_PLAYWRIGHT_VERIFIED**.
