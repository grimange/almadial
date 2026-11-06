# M0-06 — ConfBridge, AMI, FastAGI and recording runtime

Date: 2026-11-06 UTC. Implementer: Codex. Task: ALM-6 / M0-06,
`xyu5rkqfhnnjl82bq5ywt9t8`, project `ghritpte6e40o5ayktgz0vh1`.
Implementation verdict: **READY FOR INDEPENDENT QA**. This is an implementation
handoff; independent acceptance and whole-stack qualification remain pending.

## Baseline and authority

The operator's ALM-6 assignment authorized this bounded experiment only on
`almadial-alma10`. The protected `authorization.md` records that assignment;
there is no invented external authorization identifier. Accepted ALM-5 baseline:
`e99be474bc24462ae7e24bc0ba71e4920bc9cfb4` (verified ancestor). During this work
the operator committed separate portal changes; observed HEAD advanced to
`91616c7792757479947a0d7a80401c5953225e72`. Those commits were preserved.
The candidate stack manifest was not changed or promoted.

Observed target: AlmaLinux 10.2 x86_64, Multipass 1.17.0 / QEMU/KVM, 4 vCPU,
8 GiB RAM, 80 GiB disk; Python 3.12.15; Asterisk 22.12.0 / PJPROJECT 2.17;
MariaDB 10.11.18, PHP 8.3.36, Perl 5.40.2, VICIdial r4037 / schema 1748.
SELinux remained Enforcing. Chromium 157 and SIP.js 0.20.1 were used.

Before runtime changes, the VM was confirmed idle, stopped, and snapshotted as
`pre-alm6-telephony-runtime` at 2026-11-06T01:18:45Z. Multipass read-back verified
the snapshot and existing snapshots were retained. The instance was then started.
MariaDB, PHP-FPM, Apache and Asterisk required explicit service starts; the new
FastAGI service was also started explicitly. These services are not boot-enabled.
This experiment proves one Asterisk service restart, not VM boot persistence.
No snapshot was restored. Future authorized recovery requires preserving evidence,
stopping this VM, then restoring this specific snapshot and verifying guest health.

## Evidence location and provenance

Protected host evidence root:
`~/almadial-lab/evidence/m0-06/20261106T011827Z/` (directory mode 0700).
Raw browser SDP/ICE information, traces, credentials, call recordings, SQL outputs,
configuration backups and Asterisk logs stay outside Git. The evidence ledger
records their file hashes; browser profile caches and credentials are excluded.
Command records contain argv, UTC, exit status, stdout and stderr. Original failed
attempts remain alongside successful runs. Do not replay every historical script.

Source authentication is inherited from accepted ALM-4, not inferred from a digest:
the authenticated upstream nightly archive was
`https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip`, SHA-256
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`.
The 2,511-file r4037 tree digest is
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
Upstream files were left intact. The accepted ALM-5 generator derivation has SHA-256
`8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda`.

## Call topology and generated configuration

Three independent browser contexts used different deterministic fake microphones:
9051 (A), 9052 (B), and supervisor 9053 (C). They used certificate-validated
HTTPS/WSS with the existing isolated lab CA and matching local test hostname.
`ignoreHTTPSErrors` was false; no certificate-bypass flags were used.
All three returned HTTPS 200 / TLS 1.3. Contacts and channels were observed in
Asterisk, beyond browser registration events. A wrong SIP password failed to register
and Asterisk logged the authentication rejection.

Endpoints 9051, 9052 and 9053 were produced by actual VICIdial r4037 generation.
9052 uses the accepted phone template; 9053 uses a dedicated `ALM6-MON` template
whose context is isolated from ordinary phones. The phone `conf_secret` field is
the generated authentication secret. Native endpoint 9001 was not repurposed.
The existing disposable database is `almadial_m005`; no database was dropped.

The actual entry path was:

```text
9051 / 9052 -> alm5-local -> alm6-test,9600001
  -> Answer / FastAGI call_log
  -> Goto(vicidial-auto-external,473782179600001,1)
  -> generated _473782179600XXX
  -> ConfBridge(${EXTEN:8},vici_agent_bridge,vici_audio_user)
  -> generated hangup FastAGI
```

Thus the final conference traversed generated phone and conference dialplan
contracts. Room 9600001 belongs to the existing 300-number VICIdial allocation.
A conditional database claim succeeded once; a competing claim updated zero rows.
The claim was released at teardown. Allocation ownership here is a lab controller
contract, not a claim that the VICIdial agent UI performed login or allocation.

Supervisor 9053 entered `alm6-supervisor` using generated `vici_monitor_user` and
`vici_monitor_menu`. Authorized AMI Redirect changed its channel into isolated
`alm6-barge`, using generated `vici_barge_user`. Ordinary phone contexts do not
include the supervisor/barge contexts. Legacy MeetMe routes remain quarantined.
Participant B was separately redirected through `alm6-transfer` back to the actual
generated conference route, then forcibly hung up by AMI. Both transitions were
observed in bridge/channel records. No external number or carrier was involved.

## Per-case runtime matrix

| Mandatory case | Observed result | Attributable evidence |
|---|---|---|
| Unique room allocation | PASS: first claim 1, competing claim 0, released | `final-export/final/allocation.json`, `cleanup.json` |
| Generated endpoint login/register | PASS: 9051/9052/9053 registered; invalid password rejected | `final-browser.json`, `sip-negative-and-fastagi-crosscheck.json` |
| Local dial and conference admission | PASS: two channels, then supervisor, room 9600001 | `final-export/final/join.json`, `monitor.json`, AMI events |
| Two independent audio directions | PASS: distinct source signals returned to the other participant | `final-media-analysis.json`, independent remote WAVs |
| Monitor and restricted barge | PASS: monitor receives A/B while muted; authorized redirect adds C audio | `monitor.json`, `barge.json`, audio analyses, context graph |
| Redirect/transfer and forced hangup | PASS: effective route and participant removal observed | `transfer.json`, `forced-hangup.json` |
| AMI authentication/authorization | PASS: valid Login/Ping/CoreStatus, wrong Login rejected, Command denied | `final-export/final/ami-events.json`, `negative-auth.json` |
| Actual VICIdial FastAGI start/hangup | PASS: Asterisk invocation, successful result, closed call rows | FastAGI log, Asterisk `full`, `call-records.json` |
| Recording start/capture/stop | PASS: AMI lifecycle, closed decodable file, both signals, correlated row | `recording.json`, recording analysis, FFprobe/FFmpeg records |
| Failure/hangup recovery | PASS: short test call closed; forced hangup cleaned its channel | browser final checks, FastAGI cross-check, cleanup |
| Unregister/logout/cleanup | PASS: zero channels/calls/conferences/contacts and blank allocation | `cleanup.json`, `handoff-security-readback.json` |
| Restart and native preservation | PASS: full repeated conference test after one restart; fresh 9001 Echo | `restart-export-v2/`, `native-echo-analysis.json` |

## Measured conference media

Final exact-source replay `final` ran around 02:03:30–02:04:17Z. Each participant
had its own real browser RTCPeerConnection and transmitted its own microphone
track. Remote capture came from the received track, independently of local audio.
The accepted paired AudioWorklet recorded synchronized inputs without padding or
truncation. Each two-party capture contained 602 paired frames over 12 seconds.

ICE and DTLS reached connected state; selected codec was PCMU / 8 kHz and SRTP
cipher was AES128_CM_HMAC_SHA1_80. A's RTP deltas were 604 sent / 604 received;
B's were 605 / 604. Both reported zero packet loss and zero jitter in the sampled
stats. Supervisor inbound stats were initially absent: its reported 404 received
packets are since stream creation, not a fabricated before/after delta.

| Actual path | Fixture pattern | Envelope correlation | Tone agreement | Estimated delay |
|---|---:|---:|---:|---:|
| A 9051 → B 9052 | 0.994059 | 0.994686 | 0.984848 | 40 ms |
| B 9052 → A 9051 | 0.998020 | 0.997359 | 1.000000 | 60 ms |
| C 9053 barge → A | 0.988201 | 0.970590 | 1.000000 | 60 ms |

Signals use distinguishable sequences of 697, 941, 1209 and 1633 Hz.
Inherited thresholds were unchanged: pattern ≥0.90, correlation ≥0.85, tone
agreement ≥0.90, remote RMS ≥0.02 and estimated delay <2000 ms. Delay includes
capture-start skew; it is not a calibrated one-way network measurement.
The supervisor's received mixture matched both sources in 281 distinguishable
frames with agreement 0.950178. While muted, C→A correctly failed the positive
source test (tone agreement 0.375587); CLI also showed the monitor mute state.
After authorized barge, its distinct signal reached A. B was muted only during
that isolated C→A measurement and then re-enabled.

Offline analyzers deliberately report `runtime_state: NOT_RUN`. Runtime PASS here
joins their numerical result with actual browser TLS/ICE/DTLS evidence and Asterisk
conference/channel events. RTP counters or local fixture matching alone do not pass.

## AMI, FastAGI and actual correction

AMI remained bound to 127.0.0.1:5038. The separate test account used read/write
`system,call,reporting,agent` for the exercised actions. Login, Ping, CoreStatus,
Status, ConfbridgeList, recording start/stop, Redirect and Hangup were exercised.
Wrong authentication and AMI Command were rejected. This verifies the selected
operations, not permissions for every VICIdial background daemon or a full
supervisor application. No public AMI listener was added.

Actual VICIdial `bin/FastAGI_log.pl` was invoked by Asterisk for call start and
hangup. Under MariaDB strict mode the original INSERT supplied empty strings for
datetime/numeric fields. That real failure was reproduced. A hash-bound r4037
derivation changes the single exact INSERT preimage to `NULL,0,0,0`; SQL mode was
not weakened. It also redirects the private config path, loopback listener and
service user/group to the test context. No upstream source file was edited.

| Artifact | SHA-256 |
|---|---|
| Original `FastAGI_log.pl` | `f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a` |
| Derived runtime script | `9e90d5a32f1794213f1ea36d33facb9503666fa64f78b3142e37c57163d67f43` |
| Original Asterisk Perl AGI module | `38f4e84f50e93e53d63b279584f5fca04553528c7660022d4b082aac30314c9a` |

FastAGI runs as `asterisk` on 127.0.0.1:4577, with pinned Perl modules, private
root:asterisk 0640 configuration, protected logs, NoNewPrivileges and PrivateTmp.
The database account is limited to the test database; DELETE is limited to the
required `live_inbound` table. Idle prefork workers are expected service workers,
not orphaned call requests. `res_speech` must load before `res_agi`.

Final actual call rows closed automatically:
9051 `1793930610.8` (47 s), 9052 `1793930610.9` (45 s), and supervisor 9053
`1793930631.13` (26 s). UTC evidence and database local +08 timestamps are
distinguished. Final Asterisk channels were `PJSIP/9051-00000005`,
`PJSIP/9052-00000006`, and `PJSIP/9053-00000007`; raw evidence preserves SIP Call-ID,
Uniqueid and Linkedid correlation without exporting private SDP into Git.

## Recording lifecycle and precise scope

AMI ConfbridgeStartRecord/StopRecord controlled native Asterisk MixMonitor.
The lab controller inserted and correlated `recording_log` row 4; this is not
evidence of recording initiation through the VICIdial agent UI.
Final file: `alm6-final-1793930610.wav`, mono PCM signed 16-bit / 8 kHz,
46.70 seconds. FFprobe and FFmpeg decoding exited 0, and no Asterisk file descriptor
remained open after stop. Both expected source tones were present with agreement
0.904968 across 463 distinguishable frames (threshold 0.90, a relatively narrow
margin). A separate post-restart recording achieved 1.0 across 441 frames.
These observations should be independently reproduced rather than generalized to
arbitrary speech quality or all network conditions.

Final metadata uses actual AMI recording-event times, not the later SQL update
time: start epoch 1793930611, end 1793930657, rounded decoded length 47 seconds,
correct file location and channel Uniqueid. The event/decoded-duration tolerance
is one second. The corresponding call records closed independently via FastAGI.

## Repeatability, preservation and observed failures

The core test passed in multiple runs, including a full supervisor/recording run
after one controlled Asterisk restart and the final replay with the exact current
repository runtime/browser helpers. An immediate post-restart CLI socket race was
resolved by bounded health polling, not another restart. Native endpoint 9001 had
a fresh actual Echo call: pattern 0.998020, correlation 0.999335, tone agreement 1.0,
40 ms estimated delay and 602 paired frames. It terminated cleanly.

Asterisk executable SHA-256 remained
`db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c`.
Explicit configuration changes were limited to `modules.conf`, `extensions.conf`,
`manager.conf`, `pjsip_wizard-vicidial.conf` and `extensions-vicidial.conf`.
Base HTTP/TLS, PJSIP transports, RTP, Asterisk base, wizard base and ConfBridge
configuration hashes remained unchanged. Generated changes came from the actual
accepted generator; no silent manual edits to generated files were made.

Original failures are retained: wrong initial phone credential field; browser
background throttling causing empty capture; shared phone template overriding the
supervisor context; missing pinned Perl library path; AGI module prerequisite;
strict SQL insertion failure; and a missing supervisor-barge hangup handler.
The handler was added and later call rows closed automatically. One historical
open row (`1793929301.15`) was explicitly reconciled using the recorded AMI hangup;
it is not claimed as an automatic-cleanup success. Earlier recording metadata
rows had incomplete locations or controller-time intervals; they remain historical
and are not the final acceptance evidence. Final event-time metadata was tested
and replayed. A first incomplete restart export caused by a failed tar glob was
preserved, followed by a checked complete `restart-evidence-v2.tar.gz` export.

The final 02:06:50Z read-back showed all five required services active, SELinux
Enforcing, AMI/FastAGI loopback only, `skip_networking=ON`, no TCP 3306 listener,
zero channels/calls/conferences/contacts, blank allocation and all three final
call rows closed. No full dialer or external carrier was started.

## Reproduction interface and limits

These are bounded lab tools, not an installer or a blind provisioning script.
Use the already configured authorized VM, verify the recovery snapshot and idle
state, and verify reviewed helper hashes against `task-source-hashes.json` first.
Protected setup command records preserve the private deployment and exact config
diffs. Review them before reproducing a changed environment. Required contracts:
three generated phone records/templates, the contexts above, test AMI account,
private credentials, actual derived FastAGI service and the existing trusted CA.

1. Create a new 0700 evidence directory. Securely copy only required reviewed
   helpers/client and verify their SHA-256 identities in the guest. Retain the
   private `/root/almadial-m006` configuration and test credentials outside Git.
2. Generate distinct A/B/C WAV inputs with `m0_06_media.py`'s fixture functions;
   use the existing trusted browser CA home and secure credential files. Serve
   the reviewed `m0_06_webrtc_client.html` as the existing isolated `alm6.html`.
3. On this VM run `sudo python3 /root/almadial-m006/m0_06_runtime.py UNIQUE-RUN --full`
   with a fresh lowercase/digit/hyphen run identifier. It refuses active channels,
   a different hostname, an existing output directory or an unavailable allocation.
4. In Playwright MCP navigate to an `about:blank` URL with `evidence`,
   `alm3Evidence`, `repo`, `run` and current `vmAddress` query parameters. These are
   paths/identity only, never passwords. Execute the reviewed function through
   `browser_run_code_unsafe` using `scripts/lab/m0_06_browser.js` as its filename.
   Start this alongside the waiting guest observer, not after its deadline.
5. Preserve observer exit status, browser JSON/traces, both directions' WAVs,
   guest per-case JSON, AMI/FastAGI/Asterisk logs and recording. Transfer them to
   the protected host directory and verify hashes before analysis.
6. Reuse `m0_06_media.py` analysis functions with unchanged thresholds, FFprobe,
   FFmpeg decoding and the guest's recording-event contract. Check the remote
   mixture and negative monitor source, not only RTP counts. Inspect zero channels,
   conferences, contacts, closed recording, closed call rows and released allocation.

The browser helper requires the existing `m0_05_capture.js` paired AudioWorklet.
Source hashes, exact executed setup/transfer commands and the final guest archive
allow independent QA to reproduce without putting credentials into arguments.

Physical microphone/speaker hardware, full VICIphone/agent UI, campaign dialing,
PSTN, Internet/remote-NAT/TURN media, VM boot persistence and independent QA are
NOT_RUN in this task. DTMF menu operation is not part of the canonical mandatory
lifecycle cases and was not claimed. Existing browser certificate rejection tests
were not repeated; actual positive HTTPS/WSS validation was enforced in every run.
The diagnostic SIP.js client, not full VICIphone, was tested. No production-readiness
or complete VICIdial user-interface acceptance is implied.

## Repository validation and handoff

Task-owned files are this report, five `scripts/lab/m0_06_*` helpers/client files
and `scripts/tests/test_m0_06_runtime.py`. The focused tests cover hash/preimage
rejection, deterministic derivation, negative audio cases, unchanged thresholds,
mixed-source checks, AMI header injection, rejected actions, event-time recording
validation and allocation release after AMI connection failure. Offline tests
cannot claim live runtime PASS.

Final checks executed on 2026-11-06, all exit 0:

| Command | Result |
|---|---|
| `python3 scripts/validate_scaffold.py` | PASS: 11 required files, 4 skills; runtime NOT_RUN |
| `python3 -m unittest discover -s scripts/tests -v` | PASS: 96 tests, including 15 new ALM-6 tests |
| `python3 scripts/verify_candidate_stack.py` | PASS: local candidate consistency; package/source/runtime checks NOT_RUN |
| `python3 scripts/verify_config_generation.py` | PASS: redacted fixture replay; live runtime NOT_RUN |
| `python3 -m py_compile scripts/lab/m0_06_fastagi.py scripts/lab/m0_06_media.py scripts/lab/m0_06_runtime.py scripts/tests/test_m0_06_runtime.py` | PASS |
| `node --check scripts/lab/m0_06_browser.js` | PASS |
| `git diff --check` | PASS; untracked source whitespace separately checked |

Important protected evidence SHA-256 identities:

| Artifact | SHA-256 |
|---|---|
| `final-evidence.tar.gz` | `24afd6dc6d8ee84cebb8cf22a03dae2b6493894b5ec68ed2f2e85be394b345fc` |
| Final recording WAV | `462c40b545c29436b6689416d3a91f9dbebdbca7f5ad709b1b11a2bb0a5bbcba` |
| `final-media-analysis.json` | `3b7bcac2eb5c42fedb1e7a37ca08bbb5fa228e7cf7e4e0bb90c3c02b4ff203b1` |
| `final-recording-analysis.json` | `282b6035487f094d08c20d2069ccfd3c7a9a37993010c893239244bffa022890` |
| `final-browser-sanitized.json` | `3d099d4af0a889270cfd65da8cdbb95fc20e0a8652ebd99f5a781a6403fba154` |
| `native-echo-analysis.json` | `ad431c2ea6c14643a845f2c22e0c3e91fccee01b0293d80e7b8d5656dcf75666` |
| `restart-evidence-v2.tar.gz` | `45cb8fb0838c140010311b9e05d9b60bba158a07a718e7e2afd88fc5cc0e50e7` |
| `handoff-security-readback.json` | `1088818978e7aad3dc85a96547e29cc7e93d35666e1b9cb8936b2270de4462d6` |

`task-source-hashes.json` binds all six code/test files; protected `reviewed-source/`
contains copies of their final bytes. The evidence ledger excludes credentials,
browser profiles, itself and later Kaneo read-back records. Its identity and the
Kaneo handoff result are recorded below.

No local commit or push was created. Pre-existing reviewer follow-up and helper
files remained byte-identical. Concurrent portal changes were committed by the
operator and their current tracked bytes match HEAD. Playwright operational artifacts
remain outside the task-owned source set. Independent AGY review must preserve
Codex ownership, reproduce critical runtime/media/recording/cleanup cases and review
the scoped FastAGI derivation before final acceptance. ALM-7 remains unstarted.

Final ledger: **294 files**, SHA-256
`07089e48285554765b98dcb658442caa0787d178b26e0a7b08cdf85a33bae544`.
Every included entry was read and rehashed successfully. The final guest archive,
recording, source copies and exact runtime transfer bytes matched their pins.

Kaneo handoff comment: `kmz52p0ej6ggnng20ojbjb7y`, created
2026-11-06T02:13:31.714Z. Its full content was retrieved and compared exactly.
Afterward ALM-6 moved from `in-progress` to `in-review` (QA Review); a separate
get-task read-back verified project identity and Codex assignment
`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`. Relation read-back preserved
`aqcxzgb5f9rnsngw5n2jg7hg` (accepted ALM-5 → ALM-6) and
`rapwb42p3l0v6r8t9ym7dkz5` (ALM-6 → ALM-7). ALM-7 remained `to-do`, unassigned.
The subsequent protected `kaneo-final-readback.json` is outside the closed evidence
ledger. No acceptance transition was performed.

**ALMADIAL_M0_06_TELEPHONY_RUNTIME_READY_FOR_QA**
