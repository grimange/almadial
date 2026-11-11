# M3-05 — Integrated call, conference, media and recording acceptance

Developer qualification of ALM-27 (`ea1axf0373fggib2v5u7ciyv`), 2026-11-12.
Developer verification passed for the real media experiment and mandatory agent application controls. This report does not constitute independent AGY acceptance.

## Contract and boundaries

The canonical Kaneo task requires agent login, join, dial, transfer, hangup and logout; real bidirectional audio; readable correlated recording; reconnect/cleanup; expired-certificate rejection; operator listening; and a separately authorized reboot checkpoint. All mandatory cases must have attributable evidence. Direct-public, fixed NAT and remote-home profiles remain NOT_RUN. The qualified topology is the disposable Multipass VM `almadial-alm16-rpm`, machine `0fffa9a4683f35b696f0b6ee7330734b`, on the host's private LAN. Recovery snapshot: `alm25-pre-firewall-20261111`.

The task adds a bounded acceptance CLI and fixtures, not a new installer stage or continuously running supervisor. The accepted S10 phone remains registration-only outside the temporary tests. No carrier, customer call, campaign dialer, permanent RTP grant, certificate rotation or public listener is introduced.

## Implemented interfaces and ownership

`cmd/almadial-media-lab` uses the existing action graph, bounded runner, exclusive journal locks, durable intent, fresh postconditions and completed-state reconciliation. Its `media`, `agent` and `reboot` scenarios bind helper source, manifest, graph and actual host identity. `plan` inspects without changing the VM. Apply requires the matching reviewed digest and explicit disposable-lab authorization. Interrupted unknown state is not automatically replayed.

Example, from the repository root, with a new protected directory:

```bash
/tmp/alm27-media-lab -scenario media -mode plan -evidence "$ALM27_EVIDENCE"
/tmp/alm27-media-lab -scenario media -mode apply -evidence "$ALM27_EVIDENCE" \
  -authorize-disposable-lab -approved-plan "$ALM27_PLAN_DIGEST"
```

The reboot scenario is separately authorized and must not be rerun casually. Exactly one reboot was executed for this assignment.

The media fixture temporarily adds destination `9271`, context `alm-s11-media`, ConfBridge room `9600271`, MixMonitor and the accepted FastAGI `call_log` hook. It preserves the S10 secret and restores the PJSIP wrapper atomically with its ownership, mode and SELinux label. A ten-minute, source-specific runtime UDP allowance for host `10.160.90.1/32`, ports `10000–10019`, permits the actual LAN media path to guest `10.160.90.233`; cleanup removes it. Permanent S09 policy is unchanged. HTTPS/WSS use localhost forwarding and the accepted CA in an isolated browser trust database. No certificate-warning bypass is used.

The agent fixture uses an owned clone database `almadial_alm27`, two temporary PJSIP identities `9061`/`9062`, synthetic user/campaign `alm27qa`/`ALM27`, one synthetic lead and ConfBridge rooms `9600000`/`9600001`. It uses the pinned r4037 application with the previously reviewed M0 compatibility overlays, in an isolated webroot. Temporary updater/listener instances read only the clone's configuration. Real visible application controls produce native manager queue requests; bounded dispatch invokes upstream `AST_manager_send.pl` for the first allowlisted request. Accepted application/database files and AMI probe grants remain unchanged. All cloned records, temporary credentials, include files, web aliases and runtime rules are removed afterward.

## Real media and recording results

`media-05` passed using Chromium 157.0.8103.38, pinned VICIphone/SIP.js 0.20.1, Asterisk 22.12.0 and PCMU at 8 kHz. Two real browser phones registered, called destination 9271, joined the same ConfBridge, exchanged distinct changing-tone patterns, and hung up. The measurement window was approximately 12 seconds starting 2026-11-11 17:14:35.784 UTC.

| Observation | Actual result |
|---|---|
| ICE and DTLS-SRTP | Connected, matching browser transport and Asterisk call observations |
| Browser A inbound/outbound | 944 / 945 RTP packets; 151040 / 151200 bytes |
| Browser B inbound/outbound | 625 / 626 RTP packets; 100000 / 100160 bytes |
| A to B audio | Pattern agreement 0.992095; envelope correlation 0.993071; tone coverage 1.0; RMS 0.107068 |
| B to A audio | Pattern agreement 0.992126; envelope correlation 0.981453; tone coverage 0.990950; RMS 0.114198 |
| Operator listening | Operator reported “Both samples are clear”; observation bound to both received WAV digests |
| Conference cleanup | Both browser legs ended; no remaining synthetic conference/channel |

Tone frequencies were 697, 941, 1209 and 1633 Hz. The audio measurements exceed the accepted M0 thresholds and establish meaningful received audio in both directions, beyond packet counters or a Connected indicator.

Actual MixMonitor PCM16 mono 8 kHz recordings were retrieved and checked with ffprobe and signal analysis. Call IDs `1794417268.12` and `1794417275.15` correlate with actual FastAGI-written `asterisk.call_log` rows, channels, destination and start/end times (19 and 12 seconds). The mixed WAVs are 303724 bytes / 18.98 seconds and 201964 bytes / 12.62 seconds. Recording content agrees with the intended tone sequences across 235 aligned analysis frames. Alignment accounts for recording start before the browser measurement window; it is not a latency measurement. The original failed epoch-alignment analysis is retained alongside the corrected analysis.

This verifies raw capture and real call-log correlation. Recording publication/mixing and a VICIdial recording playback UI are not claimed. No fabricated `recording_log` row was inserted.

## Agent controls and confirmed integration findings

The agent path is checked separately from the media experiment. Login and the visible Call Agent Webphone control authenticate the temporary phone and join its actual ConfBridge. Manual dial uses the existing-lead control against synthetic lead 27, never a carrier route.

Confirmed findings while exercising the application:

- The accepted 40-module Asterisk closure does not include `app_dial.so`; a real manual dial failed with `No application 'Dial'`. The task fixture loads only the existing pinned 22.12.0 build's module for this isolated test. It stages the module in Asterisk's actual module directory, asserts absence first, records its SHA-256, loads by basename, and unloads/removes only that hash-matched owned file in cleanup. Accepted `modules.conf` and the baseline module set remain unchanged.
- VICIdial's native `Local_out_prefix=9` still prefixes the synthetic destination with a blank campaign prefix. A dedicated `999062` alias routes only to the intended test phone.
- The customer launcher does not automatically answer incoming calls. The real browser test must click its visible answer/dial button.
- The new-number manual entry path encounters an upstream PHP/MariaDB strict decimal error for an empty `vicidial_list.gmt_offset_now`. The representative accepted-lead path avoids a broad upstream patch; the new-number limitation remains explicit.

The final `agent-28` run passed customer registration and visible reconnect, agent authentication, Call Agent Webphone / conference join, existing-lead manual dial and customer answer, blind transfer to room 9600001, customer hangup, disposition, explicit agent-phone hangup and logout. Server observations confirmed live PJSIP channels and conference state at the relevant phases, then zero channels and zero agent reservations. Browser page-error count was zero. The temporary database and runtime configuration were removed.

Two harness corrections were required: allow only the exact native logout-cleanup channel `Local/55559600000@alm27-agent`, and dispatch manager requests asynchronously so a synchronous Originate cannot block the browser driver from clicking the visible incoming-call answer control. The Local Dial(G) leg's native notification is dismissed using Go Back while server evidence confirms the actual PJSIP customer remains connected. Failed runs remain recorded; they are not counted as passes.

## Security, reboot and preservation

An isolated expired-certificate negative test failed in Chromium with `ERR_CERT_DATE_INVALID`; the same private CA's valid certificate control returned HTTP 200. No accepted TLS material was replaced. Browser test CA material and temporary private keys are confined to the fixture and removed after testing.

The separately authorized single reboot passed after calls and temporary media state were cleaned. Boot identity changed, accepted services returned, FastAGI retained its parent plus five workers, SELinux remained Enforcing, application listeners remained restricted, and no stranded calls or temporary contexts/rules remained. This is persistence evidence for the accepted baseline, not persistence of deliberately temporary acceptance fixtures.

Representative preservation checks compare the 731-RPM inventory, schema 1748/catalog digest, all 107 columns of `callin` and `gs102`, protected credential/configuration hashes, S10 phone identity, S08 certificate references, S09 permanent policy, existing recording files and accepted modules. Only attributable synthetic `call_log` additions are expected in the accepted database. Updater/listener remain disabled and active application cron count remains zero outside the isolated fixture.

## Evidence and verification

Protected evidence: `/home/ra/almadial-lab/evidence/m3-05/20261111T164624Z-alm27/`.
The evidence index lists every mandatory case with status, observation and SHA-256-bound artifacts; its validator fails closed on missing cases, altered artifacts or unattributed PASS. Failed experiments remain recorded. Browser profiles, cookies, ephemeral TLS/ICE secrets and private test keys are excluded from the sealed evidence. Synthetic received audio is protected and not committed.

Fresh assessment of the final successful run returned `actions: []` and `no_op: true`. Authorized repeat execution and an interrupted copied-journal fixture reconciled with `runner_mutations: 0`; the canonical journal digest remained unchanged. These checks re-inspect current host state and hash-bound browser results. They do not replay calls or persist temporary acceptance dialplans.

Go 1.27.3: vet, normal tests (1,402 passing test/subtest events), race tests (1,402), module verification and tidy consistency passed. Python scaffold validation and 221 tests passed. CLI smoke: 17; terminal/nonterminal journeys: 11; reproducible Linux amd64 builds: 12 command targets. Node syntax checks and formatting/diff checks passed. An intermediate regression run under umask 077 incorrectly made a deliberately public cache fixture private; the full suites passed under normal test umask 022 without changing artifact code. Logs retain both outcomes.

The 21-case acceptance index and SHA-256 ledger are stored beside the protected evidence. Hosted CI is NOT_RUN. Physical microphone/speaker testing, public/NAT profiles, production carrier routing and recording-publication UI acceptance are NOT_RUN. The synthetic new-number decimal limitation above remains nonblocking for the qualified existing-lead scenario. Independent AGY QA remains mandatory; ALM-28 is left unstarted.

Qualification ledger `SHA256SUMS`: `1de0981c58005fd4079cb6200890ca66bf44fd314a9062bb0f166a9829a1b47d` (520 files; all entries verified). The 21-case index reports no outstanding mandatory cases. Task work began against accepted `f96be291725862334037a63bfcd31b74279edc3f`; concurrent operator commits advanced HEAD to `d2b6736692745478006effec301c348d19ca6f76`. Those unrelated operations changes were preserved. Task-owned implementation remains uncommitted for independent AGY review.
