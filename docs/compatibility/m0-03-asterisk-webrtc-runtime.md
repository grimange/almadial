# M0-03 — Native Asterisk 22.12.0 WebRTC Echo

Implementation evidence, 2026-11-05 UTC / 2026-11-06 Asia/Manila. Independent QA is pending.

## Scope and authority

ALM-3 (`h2d8fcqh6uqem6rczw1fckz0`) belongs to AlmaDial project
`ghritpte6e40o5ayktgz0vh1`. Implementer: Codex. Accepted input baseline:
`198f5e50f99007a26dc6376e9dc253b016d12f62`; ALM-2 was read back as Accepted.
The operator's ALM-3 assignment authorizes this build only on disposable VM
`almadial-alma10`. Its subsequent **WebRTC Testing Addendum** explicitly accepts
measured synthetic audio through the real browser WebRTC/Asterisk Echo path.
That addendum supersedes the earlier task wording requiring physical microphone,
speaker and operator listening evidence. The original Kaneo description is preserved;
the amendment is recorded in the evidence comment. Hardware audio is **NOT_RUN**.

This report establishes native Asterisk Echo behavior on one lab topology. It does
not establish VICIdial, VICIphone, recording, conference allocation, remote-NAT or
complete stack compatibility. The stack manifest remains unchanged, `status: candidate`.
Its whole-stack runtime gate is still open.

## Environment and recovery

| Property | Observation |
|---|---|
| VM | `almadial-alma10`, Multipass 1.17.0, QEMU/KVM |
| Guest | AlmaLinux 10.2 x86_64; kernel `6.12.0-211.64.1.el10_2.x86_64` |
| Resources | 4 vCPU, 8 GiB allocated RAM, 80 GiB disk; approximately 77 GiB initially free |
| Topology | Desktop browser to directly reachable Multipass NAT guest; private addresses retained only in protected evidence |
| Repositories used | BaseOS, AppStream, CRB, Extras; no EPEL enablement |
| SELinux | Enforcing before and after tests; no policy disablement or custom exception |
| Firewall | Empty inspected nftables ruleset; nftables/firewalld services inactive; no firewall changes |
| Recovery | `clean-baseline` retained; separate `pre-asterisk22-build` snapshot created and read back |

Multipass required stopping the guest for the pre-build snapshot. It was restarted
before package installation. Snapshot metadata: 2026-11-05 18:21:52 UTC (displayed
as 2026-11-06 02:21:52 PST), parent `clean-baseline`, 4 CPU / 8 GiB / 80 GiB.
No snapshot was restored or deleted in this assignment. Historical M0-01 recovery
evidence is preserved. A separately authorized rollback, after saving evidence, is:

```bash
multipass stop almadial-alma10
multipass restore almadial-alma10.pre-asterisk22-build
multipass start almadial-alma10
multipass exec almadial-alma10 -- cat /etc/os-release
```

These are recovery instructions, **not an executed restoration test**.

## Authenticated sources and build

| Artifact | Identity / SHA-256 |
|---|---|
| Asterisk source | `22.12.0`; `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` |
| PJPROJECT | bundled `2.17`; `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` |
| SIP.js | `0.20.1`; `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` |
| Installed Asterisk binary | `db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c` |

Asterisk's detached signature was independently rechecked using isolated GnuPG
storage: valid signature from fingerprint
`F2FC93DB7587BD1FB49E045A5D984BE337191CE7`, matching the accepted manifest.
Archive-internal `.version` is `22.12.0`. GnuPG's undefined owner trust is not
represented as a public Web of Trust endorsement. PJPROJECT matches the manifest
SHA-256 and the parent Asterisk source's bundled-dependency checksum. Guest transfer
checksums matched before execution. Source URLs remain those in
[stack.lock.yaml](../../manifests/stack.lock.yaml).

SIP.js was extracted from the previously qualified VICIphone archive, with its MIT
notice retained in the private served artifact. No VICIphone application or
VICIdial source was installed or executed. No upstream JavaScript is added to Git.

DNF installed the following 16 explicitly pinned top-level packages from the
approved repositories, plus their resolved dependencies. Exact argv, transaction
output and before/after RPM inventories are protected evidence.

| Package | Epoch:version-release.arch |
|---|---|
| autoconf | `0:2.71-13.el10.noarch` |
| automake | `0:1.16.5-20.el10.noarch` |
| gcc, gcc-c++ | `0:14.3.1-4.4.el10.alma.2.x86_64` |
| jansson-devel | `0:2.14-3.el10.x86_64` |
| libedit-devel | `0:3.1-52.20230828cvs.el10.x86_64` |
| libsrtp-devel | `0:2.3.0-16.el10.x86_64` |
| libtool | `0:2.4.7-13.el10.x86_64` |
| libuuid-devel | `0:2.40.2-18.el10_2.20.x86_64` |
| libxml2-devel | `0:2.12.5-10.el10_2.4.x86_64` |
| make | `1:4.4.1-9.el10.x86_64` |
| ncurses-devel | `0:6.4-15.20240127.el10_1.x86_64` |
| openssl-devel | `1:3.5.8-2.el10_2.alma.1.x86_64` |
| pkgconf | `0:2.1.0-3.el10.x86_64` |
| sqlite-devel | `0:3.46.1-6.el10_2.x86_64` |
| zlib-ng-compat-devel | `0:2.2.3-3.el10_1.x86_64` |

The first configure attempt exposed missing `bzip2` for the bundled archive.
The bounded correction installed `bzip2-1.0.8-25.el10`, `patch-2.7.6-26.el10`,
and the latter's dependency `ed-1.20-5.el10`. No Asterisk source modification was
needed. GCC reports 14.3.1. No DAHDI or legacy chan_sip was installed.

Executed build recipe, as the unprivileged guest user until installation:

```bash
cd /home/ubuntu/alm3/asterisk-22.12.0
./configure --with-pjproject-bundled \
  --with-externals-cache=/home/ubuntu/alm3/cache --without-dahdi
make menuselect.makeopts
menuselect/menuselect \
  --disable-category MENUSELECT_CORE_SOUNDS \
  --disable-category MENUSELECT_EXTRA_SOUNDS \
  --disable-category MENUSELECT_MOH --disable BUILD_NATIVE \
  --enable chan_pjsip --enable res_pjsip \
  --enable res_pjsip_transport_websocket --enable res_http_websocket \
  --enable res_srtp --enable res_crypto --enable app_confbridge \
  --enable app_echo --enable res_pjsip_config_wizard menuselect.makeopts
make -j4
sudo make install
```

Configure, compilation and installation succeeded. Protected logs and final
`menuselect.makeopts` retain full build selections. `make samples` was not used.
Executable: `/usr/sbin/asterisk`; installed modules: `/usr/lib/asterisk/modules`.
This run records binary identity, not reproducible-build byte equality across hosts.

## Minimal native runtime and configuration ownership

A dedicated `asterisk` service account runs the foreground daemon under systemd;
it does not run as root. `/run/asterisk`, `/var/lib/asterisk`, `/var/spool/asterisk`
and `/var/log/asterisk` have suitable ownership. Configuration and TLS files are
root-owned, group-readable by Asterisk with mode 0640. Standard paths were
processed with `restorecon`; SELinux remains Enforcing.

The lab owns `/etc/asterisk/{asterisk,modules,http,pjsip,extensions,rtp,logger,
manager,ari,confbridge}.conf`, its private key/certificate files, the isolated test
page under `/var/lib/asterisk/static-http/`, and
`/etc/systemd/system/asterisk.service`. There were no prior Asterisk files to
replace. These are disposable native test files, **not ownership claims over
future VICIdial-generated configuration**. Later integration must explicitly
reconcile them before adopting any generated file.

The service unit uses `User=asterisk`, `Group=asterisk`, `RuntimeDirectory=asterisk`,
`UMask=0027`, `Restart=on-failure` and:

```ini
ExecStart=/usr/sbin/asterisk -f -C /etc/asterisk/asterisk.conf
ExecStop=/usr/sbin/asterisk -rx 'core stop now'
```

It is running but **not enabled for boot**; this task tests service restart,
not guest reboot persistence. AMI and ARI are disabled. Plain HTTP binds loopback
port 8088; HTTPS/WSS binds guest port 8089. There is no carrier/trunk, outbound
dialplan, public port forwarding, database or unrelated application listener.
The PJSIP WSS transport's displayed bind port is not a UDP SIP listener: WSS
uses Asterisk's HTTPS `/ws` handler. Final socket evidence is preserved.

Relevant configuration, with credential deliberately omitted:

```ini
; http.conf [general]
enabled=yes
bindaddr=127.0.0.1
bindport=8088
enablestatic=yes
tlsenable=yes
tlsbindaddr=0.0.0.0:8089
tlscertfile=/etc/asterisk/keys/server.crt
tlsprivatekey=/etc/asterisk/keys/server.key

; pjsip.conf
[transport-wss]
type=transport
protocol=wss
bind=0.0.0.0
[9001]
type=aor
max_contacts=1
remove_existing=yes
[9001]
type=auth
auth_type=userpass
username=9001
; password must be generated and supplied through a protected local file
[9001]
type=endpoint
transport=transport-wss
context=alm3-echo
disallow=all
allow=ulaw,alaw
auth=9001
aors=9001
webrtc=yes
dtls_auto_generate_cert=yes
ice_support=yes
use_avpf=yes
rtcp_mux=yes
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes

; extensions.conf
[alm3-echo]
exten => 9001,1,Answer()
 same => n,Echo()
 same => n,Hangup()

; rtp.conf [general]
rtpstart=10000
rtpend=10019
icesupport=yes
strictrtp=yes
```

Final `modules.conf` uses `autoload=no`. Its explicit dependency closure is:
`res_crypto`, `res_http_websocket`, `res_pjproject`, `res_sorcery_config`,
`res_sorcery_memory`, `res_sorcery_astdb`, `res_sorcery_memory_cache`, `res_pjsip`,
`res_pjsip_config_wizard`, `res_pjsip_authenticator_digest`,
`res_pjsip_endpoint_identifier_user`, `res_pjsip_endpoint_identifier_ip`,
`res_pjsip_registrar`, `res_pjsip_session`, `res_pjsip_sdp_rtp`, `res_pjsip_nat`,
`res_pjsip_transport_websocket`, `res_rtp_asterisk`, `res_srtp`,
`res_timing_timerfd`, `res_pjsip_pubsub`, `chan_pjsip`, `pbx_config`, `app_echo`,
`app_confbridge`, `bridge_softmix`, `bridge_simple`, `bridge_builtin_features`,
`bridge_builtin_interval_features`, `codec_ulaw`, `codec_alaw`, `codec_resample`,
`format_pcm`, `format_wav`, `func_callerid`, `func_volume`, `func_jitterbuffer`.
Use `load=NAME.so` for each. With built-ins, 53 modules are listed.

An initial broad autoload run generated errors for unconfigured optional modules.
The first explicit module list omitted `res_pjsip_pubsub`; chan_pjsip declined to
load. Its authenticated source declares that dependency. Adding it resolved the
failure. Both final-configuration Echo calls below occurred after the correction.
No optional-module errors were suppressed to claim successful loading.

| Required module | Final runtime result |
|---|---|
| chan_pjsip, res_pjsip | Running; actual authenticated call channel |
| res_pjsip_transport_websocket, res_http_websocket | Running; real browser WSS SIP exchange |
| res_srtp, res_crypto | Running; established DTLS-SRTP transport |
| app_echo | Running; CLI showed live channel in `Echo()` |
| app_confbridge | Running; CLI application help available; conference test NOT_RUN |
| res_pjsip_config_wizard | Running; generated VICIdial configuration test NOT_RUN |

## Trusted browser and negative tests

Playwright MCP supports actual Chromium execution, JavaScript evaluation,
microphone permission, file-backed fake capture, downloads, CDP and tracing.
The default browser rejected the lab's untrusted certificate. A separate,
task-local Chromium profile used an isolated NSS trust database containing only
the lab CA. No desktop-wide trust store or browser security policy was changed.

Browser: Chromium/Chrome `157.0.8103.38`, headless. Explicit launch configuration:
`ignoreHTTPSErrors=false`, microphone permission, `--use-fake-device-for-media-stream`,
`--use-fake-ui-for-media-stream`, `--use-file-for-fake-audio-capture=PROTECTED_SIGNAL_WAV`
and a host-resolver mapping of `almadial-alma10.test` to the current guest address.
No certificate-ignore switch was used. A stable `.test` name required no public DNS.

The isolated RSA lab CA signed a matching server certificate. Browser security
details report TLS 1.3, issuer `AlmaDial ALM-3 isolated lab CA`, and subject
`almadial-alma10.test`. Certificate expiry is **2026-11-07 18:26:13 UTC**.
A later rerun must check validity and, if necessary, issue and trust a new
matching short-lived lab certificate while retaining these historical artifacts.
The HTTPS certificate is distinct from the auto-generated DTLS fingerprint
validated through SDP.

| Case | Observed result |
|---|---|
| Untrusted CA | Browser rejected with `ERR_CERT_AUTHORITY_INVALID` |
| Trusted matching HTTPS name | HTTP 200, TLS 1.3, expected issuer/subject |
| Trusted matching WSS | Actual REGISTER/INVITE/ACK/BYE and unregister exchanges over `wss://.../ws` |
| Mismatched HTTPS name | `ERR_CERT_COMMON_NAME_INVALID` |
| Mismatched WSS name | WebSocket rejected; browser console explicitly reports `ERR_CERT_COMMON_NAME_INVALID` |
| Incorrect SIP password | Initial and authenticated REGISTER both received 401; no registered event |
| Correct SIP password | 401 challenge followed by 200; live PJSIP contact observed |

Negative certificate tests produced TLS handshake/shutdown errors in Asterisk's
logs, which are preserved. The logs also contain historical optional-module
errors and the resolved missing-pubsub dependency. A clean final runtime is not
a claim that every historical log line is error-free.

During initial client setup, an overridden SIP.js `transportOptions` object
omitted its required `server` property. Supplying the explicit WSS URL fixed the
client's `Invalid WebSocket Server URL` failure; no Asterisk patch was involved.

## Measured Echo media and restart

[m0_03_signal.py](../../scripts/lab/m0_03_signal.py) generates a 40-second,
16 kHz mono PCM16 fixture: a fixed 16-symbol sequence over 697, 941, 1209 and
1633 Hz, 400 ms tone / 100 ms silence per symbol, with deterministic amplitude
changes and ramps. Fixture SHA-256:
`2d513b4c3eb112000f9da0626854904e00ac14f9d0a737f0e3a24a09d3c23efd`.

[m0_03_webrtc_client.html](../../scripts/lab/m0_03_webrtc_client.html) uses the
verified SIP.js client and native `getUserMedia`/`RTCPeerConnection`. It captures
the actual input stream and **the remote track from the real peer connection**
simultaneously through Web Audio. It never connects the local stream to the
remote stream. Only the browser's fake microphone source is synthetic; signaling,
DTLS-SRTP, Asterisk processing, return media and remote-track analysis are live.
The remote audio element is attached for playback; physical output hardware is
not evaluated by a headless browser.

The analyzer checks the input against the known fixture, computes 20 ms RMS
envelopes and Goertzel tone labels, aligns input and return within a two-second
window, and evaluates returned energy and pattern agreement. Required thresholds:
input pattern ≥0.90, envelope correlation ≥0.85, tone agreement ≥0.90, remote
RMS ≥0.02, all four frequencies observed, estimated delay <2000 ms. Startup
one second is excluded from correlation. Delay is an estimate with 20 ms frame
resolution and possible Web Audio callback alignment uncertainty (1024 samples,
64 ms), not a precision network-latency or acoustic measurement.

| Observation | Final minimal configuration | Same configuration after service restart |
|---|---:|---:|
| Capture start UTC | 18:36:06.965 | 18:37:15.010 |
| Requested capture duration | 16 s | 16 s |
| Known input pattern agreement | 0.9672 | 0.9807 |
| Input/remote envelope correlation | 0.9525 | 0.9644 |
| Tone label agreement | 0.9851 | 0.9630 |
| Estimated delay | 60 ms | 60 ms |
| Remote RMS | 0.1150 | 0.1150 |
| Returned frequencies | All four | All four |
| Outbound RTP packets / bytes at final snapshot | 1624 / 259840 | 1561 / 249760 |
| Inbound RTP packets / bytes at final snapshot | 1616 / 258560 | 1553 / 248480 |
| Inbound packets lost / concealed samples | 8 / 1233 | 8 / 1233 |
| Signal result | PASS | PASS |

Both calls registered a real contact, showed a live `PJSIP/9001-...` channel in
native `Echo()`, and used PCMU/G.711 μ-law, 8000 Hz, mono. ICE and peer connection
states were `connected`; the selected candidate pair carried traffic. Browser
transport statistics report DTLS `connected`, cipher
`TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256`, SRTP
`SRTP_AES128_CM_HMAC_SHA1_80`; DTLS wire version `FEFD` denotes DTLS 1.2.
Full SDP, fingerprints, selected candidate pairs, RTP/RTCP statistics and SIP
messages are in the protected JSON. RTP counters cover the call through the
snapshot, not just the 16-second capture.

Packet loss and concealment are genuine observations; their cause was not
isolated in this bounded test. They did not prevent either measured signal from
passing. Independent QA should inspect these statistics and captures. No claim
of lossless audio or production media quality is made.

Each call ended with SIP BYE, unregister and transport disconnect. Direct CLI
read-back showed **zero active channels, zero active calls and no contacts**.
After the first final-configuration call, Asterisk was restarted, waited fully
booted, and its modules, effective endpoint and HTTP/WSS transport read back
before the second call. A preliminary broader-autoload call also passed
(0.9849 envelope correlation, 0.9854 tone agreement, estimated 40 ms); it is
retained as historical evidence, not substituted for the final-configuration pair.

## Protected evidence and integrity

Evidence root: `~/almadial-lab/evidence/m0-03/20261105T182243Z/`.
Raw addresses, credentials, CA/private keys, SIP authentication material and
browser profiles remain outside Git in protected storage. Directory permissions
are restrictive; evidence files were set to 0600. Do not publish the raw trace.

| Evidence file | SHA-256 |
|---|---|
| `echo-minimal.json` | `e65fb2bc8472d187b1e51cc12873e9c7bd1bb64c8b40cc6e289e5d95e5dd9331` |
| `echo-restart.json` | `daede3b643b130828e86c2bfecbfe6d2cdbaa98d41030c67c3178ef01d8a5706` |
| `echo-minimal-local.wav` | `98048cd4afe7535db5ca73c16bbf6b0a6481b92342e8fb4c8517f1de6d928c07` |
| `echo-minimal-remote.wav` | `60cf9c24235d58b8fcfcd301cc676fd0d5da337766aa73bc3b5ab18d7cd7d7dc` |
| `echo-restart-local.wav` | `647592dc49a4c0161cdda60f9369f004a3287f0812660e149ed62b8bece2ecf7` |
| `echo-restart-remote.wav` | `601ff82348fe2a63633570c961d0ecae57f805b869f7671afa26504949f39f85` |
| `signal-final-minimal.json` | `ff90be9052e15288c81ac82b16956398ab8888ca8abf28c2b411a389368b1fb2` |
| `signal-final-restart.json` | `7c2775bf9ba54edc4f56502d4e48817a53c454d0fb12e9319b4ab018f27c24a5` |
| `browser-final.json` | `7259704a325d2b2d1cf14c66c7de0ecb0bec9541255975c3c019679c4e1511bc` |
| `playwright-trace.zip` | `4e137cc782c7072e70dafbbee08543e29ea70cfd29b1803d02f79f0ccd0885f6` |

Other evidence includes source/signature verification, snapshot metadata, DNF
transactions, final binary/module/configuration digests, endpoint settings,
active-Echo CLI snapshots, cleanup, `guest-build-logs/`, browser snapshots and
`transfer-integrity.json`. All eight transferred guest logs matched guest-side
SHA-256 observations. Exact uncommitted repository helpers are also archived in
`repository-slice/`; this assignment creates no Git commit.

## Reproduction and validation

Reproduction requires authorized access to this disposable guest or an equivalent
authorized snapshot-backed lab, the authenticated source pins, a valid trusted
lab certificate and a new protected output directory. Do not execute this recipe
on a production host or restore a snapshot merely to review evidence.

1. Verify source/signature/checksums; review the pinned DNF transaction and build
   recipe above. Confirm the final module list and effective endpoint settings.
2. Generate a fresh known fixture with
   `python3 scripts/lab/m0_03_signal.py generate NEW_SIGNAL.wav`.
3. Serve the original test client plus the pinned SIP.js from Asterisk's static
   directory. Use a fresh private browser profile/NSS trust store and the explicit
   fake-audio settings above; never bypass certificate validation.
4. Supply the protected credential file with Playwright's file-input mechanism;
   read it in the page and remove the input. Never put the password into tool
   arguments, source, console output or the issue comment.
5. Call `alm.register(secret)`, wait for `registered`, call `alm.call()`, wait for
   connected ICE/DTLS, capture `alm.snapshot()`, `alm.capture(16)`, and another
   snapshot. Download local/remote WAVs and JSON; preserve SIP frame diagnostics.
6. Independently run `asterisk -rx 'core show channels verbose'` and
   `asterisk -rx 'pjsip show contacts'` while the call is active. Analyze the
   downloaded streams with:

   ```bash
   python3 scripts/lab/m0_03_signal.py compare LOCAL.wav REMOTE.wav
   ```

7. Call `alm.hangup()`, verify channel/contact cleanup, restart the service and
   repeat. Save Playwright tracing before closing its browser context. Hash the
   immutable results and compare all transfers before the QA handoff.

Executed repository checks:

```bash
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 -m py_compile scripts/lab/m0_03_signal.py scripts/tests/test_m0_03_signal.py
git diff --check
```

Scaffold validation passed; **49/49 tests passed**, including six signal tests:
known delayed echo, silence rejection, short-input rejection, wrong tones with
the same envelope, unknown input pattern, and the two-second latency boundary.
Compilation checks passed. Whitespace and final working-tree checks are recorded
with the final handoff. These unit tests are separate from the live browser/guest
results above. No Go code was added.

## QA handoff and remaining gates

Implementation verdict: **`ALMADIAL_M0_03_ASTERISK_WEBRTC_READY_FOR_QA`**.
Final Kaneo MCP read-back confirmed ALM-3 in **QA Review** (`in-review`, column
`in73pv6vxczf7usmb8c03msz`) with Codex still assigned. Evidence comment
`zembo8pew5yczbf42ma6ojyk` was read back and matched its submitted content exactly.
The original description, priority and both native ALM-2 → ALM-3 → ALM-4
dependency identities were preserved. ALM-2 remains Accepted; ALM-4 remains
Backlog and unassigned. Independent acceptance is not granted by this report.

The immutable protected `ledger.json` indexes 127 artifacts; its SHA-256 is
`43e728dc96aea009fbdde0f3491fa94ff113cbcc182181939941f8eb9c97b87b`.
Final MCP write/read-back evidence is retained separately as
`kaneo-final-readback.json`, outside that frozen ledger. The final report is
also archived separately to avoid a circular evidence hash.

Next assignment: separate independent ALM-3 QA, preferably Google Antigravity
with the approved high-capability reasoning model and a Linux telephony/WebRTC
QA engineer role. Review source/signature, module closure, configs and binary
digests; replay signal analysis; independently reproduce trusted registration,
negative certificate/authentication tests, native Echo, cleanup and restart.
Use Multipass, Asterisk CLI, Python, Playwright and Kaneo; retain Codex ownership.
Assess the packet-loss observation and keep hardware audio NOT_RUN. Acceptance
requires agreement between browser media, returned signal, Asterisk Echo and
artifact provenance. Do not start ALM-4 as part of this review.

VICIdial source authentication remains a separate M0-04 hold. ConfBridge/VICIdial
configuration generation, dialer/AMI/AGI workflows, full browser-phone integration,
recording, remote NAT/TURN, reboot persistence and production hardening remain
outside this task's demonstrated compatibility claim.

Technical references:
[Asterisk WebRTC configuration](https://docs.asterisk.org/Configuration/WebRTC/Configuring-Asterisk-for-WebRTC-Clients/),
[Chromium Linux certificate management](https://chromium.googlesource.com/chromium/src.git/+/refs/heads/main/docs/linux/cert_management.md).
