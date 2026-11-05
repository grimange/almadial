# M0-05: VICIdial configuration generation and native Asterisk preservation

Implementation date: 2026-11-05 UTC (2026-11-06 Asia/Manila). Implementer: Codex.
Task: **ALM-5 / M0-05**, `z6mn0h4rh66ip9hh7hh769eu`, project
`ghritpte6e40o5ayktgz0vh1`. Independent acceptance is pending.

The bounded generation experiment passed its implementation checks. Actual r4037
code generated the configuration, Asterisk recognized the generated endpoint,
dialplan and conference profiles, unchanged inputs regenerated identical files,
and the native 9001 browser Echo path passed after restart. This does not qualify
whole keepalive operation, agent conferences, campaigns, recording or the installer.
The stack lock remains CANDIDATE.

## Baseline, authority and recovery

Repository baseline: `387f5e3cdd7d482f03bf060b4e84f76641681945`, verified in
current history. The operator's ALM-5 assignment authorizes this named disposable
VM and the bounded changes described here. The protected `authorization.md`
records that assignment; no external authorization ID is implied.

Target: `almadial-alma10`, AlmaLinux 10.2 x86_64, Multipass 1.17.0/QEMU,
4 vCPU, 8 GiB RAM, 80 GiB disk. Asterisk 22.12.0 with PJPROJECT 2.17,
MariaDB 10.11.18, PHP 8.3.36 and Perl 5.40.2 were retained. SELinux remained
Enforcing. No firewall/TLS policy bypass, carrier contact, external call,
application-package installation or unrelated VM change was performed.

Stopped the instance, created **`pre-alm5-config-audit`** at
2026-11-05T22:24:50Z, read it back, then started the guest and verified services.
Its parent is `pre-vicidial-r4037`; all five existing snapshots were preserved.
No restoration was performed. Recovery requires first exporting new evidence,
then a separately authorized stopped-instance restore of this snapshot. Snapshot
restoration is destructive to subsequent guest changes and is not a retry step.

Private evidence: `~/almadial-lab/evidence/m0-05/20261105T222428Z/`.
Guest working/evidence root: `/root/almadial-m005/`. Directories are protected
0700 and evidence files 0600. Credentials, raw configurations, logs, browser
SDP and traces remain outside Git. The private helper archives retain exact
execution procedures, originals, failures, DB setup and before/after diffs.

## Source authority and actual execution

VICIdial r4037/schema 1748 uses the ALM-4 authenticated official HTTPS archive:
`https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip`.
Archive SHA-256:
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`.
Canonical exported tree (2,511 files):
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
The original upstream files were not overwritten.

The [ownership contract](m0-05-config-ownership.md) maps exact source lines and
all nine output paths. The complete generation block of
`bin/ADMIN_keepalive_ALL.pl` (3258–5871), actual settings reads, reserved numbers
and helpers were extracted from the byte-pinned source. The full keepalive daemon
was **NOT_RUN** because it also manages other processes, media and live paths.

Reproducible source preparation, on a machine with the accepted source available:

```bash
python3 scripts/lab/m0_05_generator.py \
  --source /protected/vicidial-r4037/bin/ADMIN_keepalive_ALL.pl \
  --output-dir /protected/new-generator-output
```

This command only derives files. It refuses an existing output directory or unknown
source digest. The lab then executed the derived Perl, as recorded in the private
helpers. Do not execute it outside this explicitly prepared disposable context:
its fixed DB/staging paths intentionally do not constitute a general installer.
The final executed guest script is `/home/ubuntu/alm5-generator-corrected.pl`.
`/home/ubuntu/alm5-generator.pl` is the retained historical uncorrected derivation.
A final read-back initially checked that old path and failed its expected-final-hash
assertion; the corrected-path read-back passed. Do not rerun historical setup or
`final-checks.py` blindly: that older helper invokes the uncorrected script.
Use the corrected derivation and reviewed corrected-generation procedure for QA.

Input database **`almadial_m005`** was cloned from the accepted disposable
`almadial_m004_v2`. Original databases were retained. The clone had one active
local server, configuration generation enabled, version 22.12.0, CONFBRIDGE,
no carriers, and test phones 9051/9059. The ALM-3 phone 9001 was never repurposed.
The complete generator algorithm ran with 94 fixed Asterisk paths redirected to
staging and 15 screen commands recorded rather than dispatched. Generation used
real DBI/MariaDB reads and writes, not a Python emulation of the output.

Source and derivation SHA-256:

| Artifact | SHA-256 |
|---|---|
| Original `ADMIN_keepalive_ALL.pl` | `4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67` |
| Original extracted block | `41c7019e76f76e2aba8cac030b47fc17bce9b8544afcaa9980273491e313569f` |
| Initial isolated derivation, retained | `579ecdff820ef6965944ce6f5ea0f051a22fc78a1319ef71ca28ba7536c147c6` |
| Corrected final derivation | `8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda` |

## Confirmed defects and corrections

### MariaDB configuration readability

Before: `/etc/my.cnf.d/almadial-m004.cnf` was root:root 0600,
`skip_networking=OFF`, and TCP 3306 listened on wildcard addresses. Backed up the
file, changed only its owner/group/mode to **root:mysql 0640**, then performed
one controlled MariaDB restart. Effective result: `skip_networking=ON`, working
`/var/lib/mysql/mysql.sock`, no TCP 3306 listener. Both existing M0-04 databases
retained 345 tables. Non-root Apache DBI `SELECT 1` succeeded; the existing
localhost PHP admin request authenticated and returned HTTP 200 (59,384 bytes,
body SHA-256 `f3c86895bcc64bf25095af7f4a698b6f3f90b88a58b8e0e2a41f19db2829241f`).
Local application configuration already used localhost/socket connectivity and
was not silently changed to a broken TCP contract. Service logs were inspected;
minor unset MYSQLD_OPTS/WSREP environment warnings were retained.

### Perl shared-library traversal

Changed four exact root-owned directories from 0700 to 0755:
`/usr/local/share/perl5`, its `Net`, `5.40`, and `5.40/Net` directories.
Module files remained 0644. No recursive broadening of credentials occurred.
The non-root Asterisk account loaded DBI, DBD::mysql, Net::Telnet, Net::Server,
Time::HiRes, Switch and Crypt::Eksblowfish::Bcrypt successfully. The Apache account
also successfully executed its actual socket-based DBI query.

### r4037 duplicate recording-route priority

Actual reload/restart logs rejected duplicate extension 8309 priority 2.
Pinned source line 3655 emits `8309,2,MixMonitor` inside the **8310** GSM recording
branch. The narrowly pinned derivation changes this one occurrence to 8310 before
execution. Upstream source and original failing output are preserved. Generated
files were not manually patched. Final reload/restart had no duplicate-priority
warning. This proves configuration loading; recording media remains NOT_RUN.

```diff
 ; source-generated output from the 8310 branch
-exten => 8309,2,MixMonitor(...)
+exten => 8310,2,MixMonitor(...)
```

The first experiment harness omitted `PATHmonitor`, producing an incorrect recording
path. The harness was corrected to supply the actual configured
`/var/spool/asterisk/monitor`; this was not attributed to upstream code.

## Ownership, generation and idempotency

AlmaDial owns base files and includes; VICIdial owns generated files and the
**unsuffixed `voicemail.conf`** exception. Operators use separately owned,
collision-checked overrides, never edits of generated content. See the full
[ownership matrix](m0-05-config-ownership.md) and
[bounded native adapter](asterisk-vicidial-adapter.md).

Nine outputs were produced. Eight BUILD files are removed on successful execution;
`BUILDvoicemail-vicidial.conf` remains because r4037 comments out its deletion.
The final raw hashes below apply to private unredacted files; redacted fixture
bytes necessarily have different hashes.

| Generated file | Final SHA-256 |
|---|---|
| `extensions-vicidial.conf` | `05f60b460cdf15d195d9cd147082bc9fbb136073b8d6a171946b3ec7df0897b7` |
| `pjsip_wizard-vicidial.conf` | `c8890111d351229660a8efc2ee1b2cf2b6e10fa34ea67c6f6cfd90c44e66270e` |
| `pjsip-vicidial.conf` | `3b7becc1690ac25c249340a832017833f300f1b5c2380e325390f63082254d32` |
| `confbridge-vicidial.conf` | `dfc6be250bb6dae4daf5ea860081507d4b6e75ac975048bfd5d0ece9b73c706f` |
| `iax-vicidial.conf` | `d0e40e9b0d683023c3872d65c20c5855e9a805631130f29a4545886fd78ac4c4` |
| `sip-vicidial.conf` | `8210f12ef53d0644b3bb61d0f8cc5f78c760c43c155ee89ead663eae274b0d90` |
| `meetme-vicidial.conf`, `musiconhold-vicidial.conf` | `c42dcad1900a0819240b81811f0bf9184809ba441e53f07f819ffd2d27252efb` |
| `voicemail.conf`, retained BUILD copy | `c68dc9c41bdcbf51abbd092e9e4d66c63290a595821fd81954e7cff90cb61aef` |

Repeated unchanged-input generation was byte-identical. The test update changed
only the expected phone/wizard and dialplan output; unchanged re-execution matched.
Deactivating phone 9051 removed its wizard; reactivation restored identical output.
A test-only loop credential exposed by verbose diagnostic output was rotated in
the disposable DB and regenerated; the final hashes above refer to the rotated
state. Original diagnostics remain protected and are not exported to Git.

Sanitized observed phone update:

```diff
-endpoint/callerid="ALM5 Generated Phone" <9051>
+endpoint/callerid="ALM5 Updated Phone" <9051>
-exten => 9051,1,Dial(${PJSIP_DIAL_CONTACTS(9051)},20,)
+exten => 9051,1,Dial(${PJSIP_DIAL_CONTACTS(9051)},21,)
```

The full actual dialplan is retained in the redacted fixture. Published generated files were
root:asterisk 0640 and byte-equal to staging. No competing daemon was started.

## Effective PJSIP, dialplan and ConfBridge

New base `pjsip_wizard.conf` includes `pjsip_wizard-vicidial.conf` once. Existing
`pjsip.conf` includes `pjsip-vicidial.conf`. Asterisk loaded wizard 9051, AOR 9051,
and inbound auth 9051-iauth using transport-wss. Effective endpoint inspection
confirmed WebRTC, ICE, AVPF, RTCP mux, DTLS, disabled direct media, NAT/contact
settings, and ulaw/alaw. An unregistered generated endpoint shows Unavailable;
that is not a configuration-load failure. Registration of 9051 was not claimed.

`extensions.conf` includes generated globals/dialplan and the new isolated context:

```ini
[alm5-local]
include => vicidial-auto-phones
exten => 9059,1,Hangup()
```

Asterisk `dialplan show` recognized the generated 9051 route using
`PJSIP_DIAL_CONTACTS(9051)`, timeout 21 and fallback 9059. `app_dial.so` existed but
was not loaded in the minimal ALM-3 module list; it was loaded and added persistently.
No SIP trunk, chan_sip or external route was activated.

ConfBridge configuration loaded the agent bridge (8 kHz, 20 ms, max 10 members,
recording off), nine generated user profiles and the monitor menu (4/5/6 actions).
The generator allocated **300** server-associated conference numbers
**9600000–9600299** in the disposable DB. No conference participation or recording
PASS is inferred from profile loading/allocation.

Actual generated `vicidial-auto-external` still contains a legacy MeetMe route.
It is **quarantined**: neither 9001 nor 9051 can reach that context. Internal
AGI/recording routes also remain outside the allowed test graph. Macro/Monitor
were not emitted by the selected modern branch; old branches remain in source.
Full application/module availability and agent lifecycle belong to M0-06.
The fixture verifier rejects exposing removed applications to the test graph.

The native service has no `screen -XS asterisk` session. Explicit verified reloads
were used (`module reload res_pjsip.so`, `module reload res_pjsip_config_wizard.so`,
`dialplan reload`, `module reload app_confbridge.so`). A `pjsip reload` attempt
returned an unsupported-command diagnostic despite CLI exit zero; retained as a
failure, not reported as a successful reload.

## AMI and native preservation

AMI is enabled only on **127.0.0.1:5038**, with web AMI disabled. The private
`alm5-local` account permits loopback, read `system,call`, write `system`.
Observed: valid Login Success, invalid-password Authentication failed, Ping/Pong,
CoreStatus Success, and Command Permission denied. This is a bounded test profile;
full VICIdial daemon privilege requirements are not yet established. No AMI
Originate, public exposure or campaign processes were used.

Five existing base files changed with backups and recorded diffs:
`pjsip.conf`, `extensions.conf`, `confbridge.conf`, `modules.conf`, `manager.conf`.
`pjsip_wizard.conf` is new. Original native 9001 and Echo blocks remain byte-contained
in the final files. Native `asterisk.conf`, `http.conf`, `rtp.conf`, `logger.conf`,
`ari.conf` and TLS key/certificate were unchanged. Asterisk executable SHA-256
remained `db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c`.
Generation itself did not change base files. Full before/after hashes are private.

After final generation, Asterisk was restarted, CLI readiness awaited, and the
required endpoints, transport, ConfBridge menu and native Echo path rechecked.
An early immediate CLI query failed during startup; later bounded readiness checks
succeeded. Minimal ALM-3 startup still reports missing optional cdr/cel/features/
acl/udptl/pjproject configuration and keys/keys diagnostics. These are retained
baseline limitations; this report does not claim an error-free startup journal.
The final corrected run had no duplicate-extension warning.

## Browser media regression

Chrome 157.0.8103.38 used the existing private lab CA trust store with
`ignoreHTTPSErrors=false`, not a certificate bypass. HTTPS/WSS identity matched
the lab hostname, TLS 1.3 was validated, phone 9001 registered and called native
Echo. Physical microphone/speaker hardware: **NOT_RUN**; the authorized deterministic
697/941/1209/1633 Hz fixture traversed the real browser/Asterisk media path.

An early post-integration capture passed. Two later independent ScriptProcessor
captures failed fixed thresholds with unequal local/remote frame counts. Their
files and FAIL results remain preserved. To resolve the capture defect, the small
`scripts/lab/m0_05_capture.js` helper records both actual streams in one AudioWorklet
on the same audio clock. It changes capture only, not signaling, source audio,
Asterisk or thresholds. Two subsequent final-state captures passed:

| Measure | First paired capture | Repeated paired capture |
|---|---:|---:|
| Fixture pattern agreement | 1.000000 | 0.995542 |
| Envelope correlation | 0.999131 | 0.982155 |
| Tone-label agreement | 1.000000 | 0.980952 |
| Estimated delay | 40 ms | 40 ms |
| Remote RMS | 0.113329 | 0.113329 |
| Local / remote analysis frames | 802 / 802 | 802 / 802 |
| Outbound / inbound RTP packets | 804 / 796 | 806 / 797 |
| Reported packets lost | 8 | 8 |

Both observed all four tones, connected ICE/DTLS, PCMU and
`SRTP_AES128_CM_HMAC_SHA1_80`. Fixed thresholds remain pattern >=0.90, envelope
>=0.85, tone agreement >=0.90, RMS >=0.02, delay <2000 ms. Concurrent guest
sampling observed `PJSIP/9001` executing `Echo()` during the repeated passing run.
Hangup/unregister completed; final state was **0 active channels, 0 active calls**.
Asterisk, MariaDB, Apache and PHP-FPM remained active; SELinux Enforcing.
No success is based solely on RTP counters or registration.

## Evidence identities and reproduction

| Protected artifact | SHA-256 |
|---|---|
| `guest-qa-evidence.tar.gz` (historical uncorrected generation) | `a91b268110b4a6455c64e8380fee7895a02f3dbf735ed59f306eca25b95f1905` |
| `final-generated-private.tar.gz` | `71fcfcd2cf04a9759f00fc0e8cf7db562b2f337f3537e1905a9b5c61afbd458e` |
| `paired-echo-local.wav` | `ef5fe5b3f71b4fc7a62fbebe059d3a9326d1f5068548085f25053c308b21ec46` |
| `paired-echo-remote.wav` | `9823ead855e39f47d074e4e909963f70908dabe1b7ebace71d0678af4f50c9ae` |
| `paired-echo-browser.json` | `bfc6e339d21440d2d6473ba459b73abc9618cd79ad9e3b3269f1e2c3d9101902` |
| `paired-echo-playwright-trace.zip` | `f50fafd38bf3391607683866ee97d2a580f6e37eb191405a948f799389fc25ce` |
| `paired-echo-signal-analysis.json` | `bdc07025f8ac95b1ac1cc0a5e102f3fba9fd6ee263d4b6f74e639351ab9a66ef` |
| `paired-repeat-echo-signal-analysis.json` | `45b36502d1e5f32a469aad41dbc7021e1e2a69f915def01f47b5e37b91920fc0` |
| `paired-echo-runtime.json` | `d34501b60c5bd499f93438fd966eb88aad1dc8bca7d40625b57eef120abb226e` |
| `qa-reproduction-procedures.tar.gz` | `02543de342bcc4a67464ff63ee71c955e663c8833c22179acfd76445d6565cd0` |
| Repository redacted fixture | `c293b89524cfaece225449ed3bad7e27458afa6f815b86fe2c9549558acbe5cf` |

The final protected evidence ledger includes repeated capture, runtime sampler,
helper archive and command-result hashes. Independent QA must verify the ledger,
original/derived source identity, raw generation diffs, effective Asterisk state,
permission corrections and browser analysis. Do not publish private archives.

Offline structural reproduction:

```bash
python3 scripts/verify_config_generation.py
python3 -m unittest discover -s scripts/tests -v
```

The fixture contains redacted actual output and recorded metadata. Its PASS is
**REDACTED_FIXTURE_REPLAY**, runtime **NOT_RUN**, not new live proof. Live reproduction
requires this authorized VM and protected helper/input evidence. Media replay:

```bash
python3 scripts/lab/m0_03_signal.py compare \
  /protected/evidence/paired-echo-local.wav \
  /protected/evidence/paired-echo-remote.wav
```

## Validation and QA boundary

Executed successfully (exit 0):

- `python3 scripts/validate_scaffold.py`: PASS; 11 files, 4 skills, no runtime claim.
- `python3 -m unittest discover -s scripts/tests -v`: **81 tests PASS**, including
  25 new configuration/permission/source/AMI regression cases.
- `python3 scripts/verify_candidate_stack.py`: candidate consistency PASS;
  live package/source/runtime checks NOT_RUN in this invocation.
- `python3 scripts/verify_config_generation.py`: redacted contract PASS.
- `python3 -m py_compile scripts/lab/m0_05_generator.py scripts/verify_config_generation.py scripts/tests/test_verify_config_generation.py`.
- Both final paired media-analysis commands: PASS, exit 0.

Final whitespace, document-link/hash and unrelated-file preservation results are
recorded with the evidence ledger and Kaneo handoff. `git diff --check` exited 0;
all task-document links and listed host evidence hashes resolved; all 19 pre-existing
modified/untracked file bytes matched the initial inventory. No commit or push is made.

Next assignment: independent AGY ALM-5 review, retaining Codex ownership. Reproduce
critical checks before acceptance, especially the source derivation correction,
voicemail ownership exception, quarantined legacy routes, socket-only DB behavior,
non-root Perl loads, AMI negative tests and passing remote media evidence. ALM-6
must wait for that acceptance; it is not started here.
