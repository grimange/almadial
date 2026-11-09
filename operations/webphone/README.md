# PRD-WEBPHONE-00: one browser test phone

This is the deployed test fixture for `dialer01`, not an AlmaDial installer stage or a qualified general WebRTC stack. See the [acceptance report](../qa/VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md) for executed tests and limits.

## Pinned upstream and compatibility

VICIphone 3.0.0 is self-hosted from [vicimikec/ViciPhone](https://github.com/vicimikec/ViciPhone), commit **24e98fccfa7b7079fff970e02fca5f002ac1f2cc**. Its bundled SIP.js is 0.20.1. The deployed PHP, JS, CSS and image files match that revision; the root upstream LICENSE is installed alongside them. No upstream code was changed. The debug directory contains a new empty log instead of the upstream sample log and is denied by Apache. The deployment archive SHA-256 was `cf15a8546ac30fa2622bd955a35d718f66edd9e31ca8735b88038581cab2b4dd`; the file-by-file comparison is the reproducible integrity check, independent of archive timestamps.

The upstream license is AGPL-3.0. Preserve its notices and provide the exact corresponding upstream source revision when redistributing the hosted assets. AlmaDial-original files retain the repository's licensing status; this deployment does not relicense VICIphone.

The installed Asterisk 20.21.0 has working chan_sip, WebSocket, crypto and SRTP modules. chan_sip owns the SIP WebSocket subprotocol on this build; `res_pjsip_transport_websocket` is not running. The isolated phone therefore uses **chan_sip**, with no driver migration for phones 1000 or 1001. [VICIphone's configuration guidance](https://viciphone.com/?page_id=353) requires RTCP multiplexing for version 3.0. The [Asterisk WebRTC guide](https://docs.asterisk.org/Configuration/WebRTC/Configuring-Asterisk-for-WebRTC-Clients/) describes the corresponding ICE/DTLS/SRTP requirements; its PJSIP example was not copied into a chan_sip endpoint.

## HTTPS/WSS and authorization

```text
Chromium, trusted https://2.29.153.161
  └─ wss://2.29.153.161/ws, Apache :443
       ├─ existing confirmed portal web grant
       ├─ confirmed phone grant, checked for the actual connection IP
       └─ exact /ws proxy → Asterisk 127.0.0.1:8088/ws
                                  └─ SIP authentication → phone 1002
```

Only `enabled=yes` changed in `/etc/asterisk/http.conf`. The existing loopback bind and port 8088 remain. Apache exposes the exact `/ws` route, not Asterisk's other HTTP handlers. No public 8088/8089 listener, TLS key copy to Asterisk, firewall exception, new gateway or new domain was needed. Asterisk HTTP was activated with `module reload http`; Apache was gracefully reloaded.

The existing Let's Encrypt IP SAN certificate validates in Chromium 149.0.7827.55 and OpenSSL. The IP-literal HTTPS origin, WSS host, SIP URL construction, browser microphone permission and selected ICE path worked without disabling certificate or browser security. This establishes this observed client/network path; no TURN, restrictive remote NAT or alternate browser qualification is implied.

Deployment files:

| Host path | Source / purpose |
|---|---|
| `/opt/alma-webphone-00/access.lua` | [access.lua](config/access.lua), phone-grant authorization |
| `/opt/alma-webphone-00/apache-vhost.inc` | [apache-vhost.inc](config/apache-vhost.inc), exact proxy and debug denial |
| `/etc/httpd/conf.d/02-alma-vicidial-ssl.conf` | One Include of the preceding file inside the existing TLS virtual host |
| `/opt/alma-portal-443/apply_requests.py` | [bounded projection patch](config/portal-phone-projection.patch) after confirmed SIP rule reconciliation |
| `/var/lib/alma-portal-443/phone-websocket/` | Derived short-lived phone grant files, root:apache 0750 / 0640 |
| `/var/www/html/viciphone-web00/` | Pinned upstream `src/` assets and root LICENSE |
| `/etc/asterisk/keys/webphone00.crt` and `.key` | Dedicated self-signed DTLS certificate and private key; not the HTTPS certificate |

The projection contains only SIP grants confirmed by the existing worker. Each expiry is capped at the earlier of its lease and the worker's 35-second freshness horizon. Failed reconfirmation or revocation removes its file. Lua additionally requires the existing web authorization provider, validates the Host and actual connection source, and accepts only an unparsed URI of `/ws`. Authentication headers are stripped before proxying to Asterisk. The authorization check runs at WebSocket upgrade; it does not forcibly close an already upgraded connection at lease expiry. Existing SIP/RTP revocation remains in place. Revocation during an active browser call was not tested here.

The projection patch uses AlmaLinux's existing apache GID 48, verified on this host. Check that mapping before reusing it elsewhere. It does not grant network access or change the portal's authentication or office/personal grant policy.

## Native VICIdial configuration

All database mutations were performed through Administration or the agent UI. Database CLI use was read-only. The native configuration generator produced SIP/1002; generated files were not edited.

| Object | Retained configuration |
|---|---|
| Phone 1002 | SIP, active Y / ACTIVE, QAPRD14, extension/login/dialplan/voicemail 1002, fullname `QA WEB00 browser automation` |
| Phone credentials | Separate strong native phone-login password and SIP secret; private local storage only |
| Template WEB00_SIP | Exact [native template](config/WEB00_SIP.template) |
| Webphone settings | is_webphone Y, auto-answer Y, external-server-IP Y, debug N, qualify N, dialpad N, dialbox N |
| WEB00_SETTINGS | Native WEBPHONE_SETTINGS container in QAPRD14, assigned to phone 1002 |
| Agent | Existing qaagent14, unchanged restricted role; no new agent or supervisor |
| Native server | external_server_ip `2.29.153.161`; both internal/external Web Socket URL fields `wss://2.29.153.161/ws` |
| Native system | webphone_url `https://2.29.153.161/viciphone-web00/viciphone.php` |
| List 910017 | `WEB00 local simulator only`, QAOUT01, final active N |
| Lead 69 | WEB00_001, Webphone Synthetic, `web00@example.invalid`, simulator 7000000001; final NI, called_count 2 |

The settings container contains exactly:

```text
dialRegExten:1
autoGain:0
echoCan:0
noiseSup:0
```

`dialRegExten:1` makes the embedded phone dial its assigned conference after registration. The initial server-originated WebSocket callback did not complete; this supported native option produced the working conference path. It does not enable customer auto-dialing.

The template enables ICE, AVPF, DTLS-SRTP, `rtcp_mux=yes`, `nat=force_rport,comedia`, WS/WSS, ulaw only and `directmedia=no`. Its dedicated RSA-2048 DTLS certificate was generated for this fixture for 365 days, with CN `PRD-WEBPHONE-00 DTLS`; the Asterisk-owned private key is mode 0640. Renew it before expiry using the same scoped maintenance procedure. The browser verifies the SDP fingerprint; public HTTPS continues using the existing trusted certificate and renewal timer.

Final QAOUT01: inactive N, MANUAL, automatic dial level 0, no-hopper-login N, ONDEMAND recording, omit-phone-code N, call time `12pm-5pm`, manual list 910016. Phone 1014 remains suspended. No browser registration, live agent, channel or INCALL claim remains.

## Repeatable no-call Playwright run

Use existing Node.js, Playwright and Chromium; no npm package installation is required. The suite runs two fresh sessions, each registering, joining a conference, asserting outgoing microphone RTP, explicitly unregistering, logging out and closing its browser. It never selects a lead or places a customer call.

1. In the existing Secure Access Portal, obtain the normal authorized personal/office web grant **and phone-network grant** for the browser's public IP. The suite does not bypass this gate. Grant bootstrap remains the existing native portal/operator workflow; the test runs within that grant.
2. In Administration, temporarily set **only QAOUT01** active Y and No Hopper Leads Logins Y. Keep MANUAL, automatic level 0, recording ONDEMAND, all lists inactive. Do not change test01 or phone 1000/1001.
3. Supply two mode-0600 JSON files outside Git. Phone file: keys `login` and `password`; agent file: array of objects with `username` and `password`, including qaagent14. The SIP secret is supplied by VICIdial to its embedded phone and need not be logged or passed on the command line.
4. Run from the repository, adjusting paths to existing installations:

```bash
PLAYWRIGHT_MODULE=/absolute/existing/node_modules/playwright \
CHROMIUM_PATH=/absolute/existing/chrome \
WEBPHONE_PHONE_CREDENTIALS=/private/phone.json \
WEBPHONE_AGENT_CREDENTIALS=/private/agents.json \
node tests/webphone/runner.cjs
```

5. Restore QAOUT01 active N and No Hopper Leads Logins N through Administration. Check `vicidial_live_agents` is empty for the fixture, no INCALL test lead remains, and Asterisk has no test channels. Phone 1002 should show `(Unspecified)` in `sip show peers`.

The runner returns safe state/statistics JSON, not browser profiles, credentials, console bodies or iframe URLs. Never enable a raw Playwright trace, HAR or unfiltered SIP logging with real credentials: the native webphone launch URL contains base64 credential material. On a failed run the runner attempts native cleanup and reports when operator attention is required. Inspect the native session before retrying.

The login waits for VICIdial's positive JS_browser_height value and actual page navigation before submission. Submitting before onload populated that value reproduced a PHP 8 string-minus-integer error; waiting for normal form readiness fixed the test without modifying PHP.

[browser_workflow.js](../../tests/webphone/browser_workflow.js) also runs directly through Playwright MCP's `browser_run_code_unsafe` filename capability. Set `page.webphoneConfig` before invoking it. Supported phases are `start`, `inspect`, `preview`, `call`, `hangup`, `finish`. `call` requires `authorizeSingleCall:true`, the unique WEB00_001 lead, prefix 2221 and omit-phone-code Y, and refuses a second call in the same session. This is a control hook, not authorization for a future milestone. Finish refuses an open preview: use the visible native SKIP LEAD action first. Ready/pause/resume uses the installed campaign's applicable native controls; the retained MANUAL fixture has no such controls. The previous functional-gap report remains its state-transition evidence.

## Synthetic audio and the one-call hook

[make_audio.py](../../tests/webphone/make_audio.py) generates the committed original [microphone.wav](../../tests/webphone/fixtures/microphone.wav): mono 48 kHz PCM, 12 seconds, repeating 440/660/880 Hz tones with fades. It contains no private speech or recording. Regenerate with `python3 tests/webphone/make_audio.py`.

Chromium launches with:

```text
--use-fake-ui-for-media-stream
--use-fake-device-for-media-stream
--use-file-for-fake-audio-capture=<absolute microphone.wav path>
--autoplay-policy=no-user-gesture-required
```

The isolated call used QAOUT01 MANUAL/0, list 910017 active, manual-list 910017, local call time 24hours, recording ALLFORCE and **omit-phone-code Y**. The approved synthetic number 7000000001 and prefix 2221 resolve exactly to `22217000000001@default → Dial(SIP/1001,45,To)`. Read the live dialplan and inactive-carrier inventory before any future authorized call. An omit-phone-code N request produced the wrong extension and never reached SIP/1001; it was hung up and dispositioned normally. Do not silently retry a call.

For a separately authorized repeat: activate that single list, use native Lead Details/list reset for the one NI fixture if necessary, preview and verify WEB00_001, then invoke the explicitly authorized `call` phase. Arrange the existing Linphone 1001 answerer. Inspect RTP/WebRTC statistics, invoke `hangup` for native NI disposition, then `finish`. Restore all temporary campaign/list settings above. This milestone's answered call has already been performed; the default CLI does not repeat it.

Browser statistics prove transport and signal flow, not perceived audio quality. The operator confirmed hearing tones and sending a phrase; recording 4 contains the tone sequence and the browser had inbound audio energy. No private recording is stored in Git.

Future PRD-06/07/08/09/10/13 can reuse this one identity, microphone fixture, state assertions and explicit call hook after their own scope and routing approval. Concurrent agents need distinct phone/account identities. None of those future scenarios ran here.

## Checks and limitations

```bash
python3 -m unittest discover -s tests/webphone -p 'test_*.py' -v
node --check tests/webphone/runner.cjs
python3 -m unittest discover -s operations/portal443/tests -p 'test_*.py'
```

The four projection tests execute the exact added block with real temporary files, modes and atomic replacement; only privileged ownership calls are mocked. Existing portal tests use mocked accounts/firewall and loopback HTTP. They are supporting tests, separate from live Chromium/host evidence.

The existing native preview/logout INCALL defect is unchanged; release the preview or disposition before logout. Native visibility telemetry produced a duplicate `visibleuser` key exception in conf_exten_check.php during two rapid runs; registration, conference media, SIP unregister, logged logout and channel cleanup still succeeded. No telemetry/schema patch was made. Initial browser TypeError events and PHP warnings remain documented in the report; the final repeated browser runs reported no pageerror events. Active-WebSocket revocation, reconnect, alternate network topologies and human audio-quality certification remain outside this acceptance.

## Scoped rollback

Original backups and the tested deployed snapshots are root-only at `/root/prd-webphone-00-backup`. They include the original HTTP/vhost/portal-worker files, base/generated SIP configuration and native phone/server/system readbacks. Do not publish that directory: it contains secrets. Existing backup health was verified before and after changes; no full-database restore is needed.

1. Through native Administration deactivate QAOUT01 and list 910017, release/disposition any open lead, and log out the fixture. Require zero active Asterisk channels before changing HTTP routing.
2. Disable only phone 1002 through Administration. Keep phone 1000/1001 and suspended 1014 unchanged. Restore the native system webphone URL and the server's external-IP/internal/external Web Socket URL fields to their captured original blank values **only if no subsequent webphone depends on them**. Keep WEB00_SIP and WEB00_SETTINGS isolated or remove them through native Administration after removing their only reference. Preserve call/recording history.
3. Copy [scoped_config.py](scoped_config.py) to a private root location on dialer01 if needed. With no active channels, run `python3 scoped_config.py check`, then `python3 scoped_config.py rollback`. It refuses wrong host, missing/mismatched backups and configuration drift, restores only the three files, validates Apache, reloads Asterisk HTTP and gracefully reloads Apache. This actual rollback and `restore-deployed` were executed successfully during acceptance; original and reapplied hashes matched. It never restarts Asterisk or MariaDB.
4. After checking no new consumers use them, remove only `/opt/alma-webphone-00`, the derived `phone-websocket` directory, `/var/www/html/viciphone-web00`, and the two webphone00 DTLS files. These isolated artifacts were retained for reuse; their deletion was not exercised. Do not remove any existing portal grants, recordings or backup/certificate files.
5. Confirm HTTP returns to its previous disabled state, the prior Apache routes and worker hashes match, existing endpoints still register, and timers remain active. Retain scoped backups until review is complete.

`restore-deployed` reverses the tested three-file rollback after the same drift and zero-channel checks. Native fixture deletion, DTLS deletion and asset deletion were documented rather than executed, to preserve this reusable endpoint.
