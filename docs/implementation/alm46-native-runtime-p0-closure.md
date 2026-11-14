# Native VICIdial runtime and ordinary agent qualification

This correction follows the network qualification in
[remote single-server controlled qualification](remote-single-server-controlled-qualification.md).
It retains the lab profile and the existing Go installer. Developer tests are
not independent AGY review or ALM-34 human operator acceptance.

## Diagnosis and runtime contract

The inherited S06 policy intentionally disabled the updater and listener and
installed no active application maintenance. The remote profile had inherited
that policy. Its synthetic phone route worked independently of the normal
VICIdial manager queue, conference allocation and agent lifecycle. Recursive
verification also treated changing application database/generated configuration
as immutable installation state. An empty file-action plan therefore could
coexist with failed live application health.

Pinned inputs remain VICIdial 2.14b0.5, SVN r4037, database schema 1748,
Asterisk 22.12.0. VICIdial archive SHA-256:
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`;
accepted source-tree identity:
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
There was no upstream repin, schema alteration or SQL-mode relaxation.

| Component | Required remote state | Original observation | Correction and observed result |
|---|---|---|---|
| `AST_update.pl` | Running, fresh `server_updater` heartbeat | Deferred; modern AMI/PJSIP parsing incomplete | Native systemd worker; bounded Asterisk 22/PJSIP derivation; heartbeat and call lifecycle pass |
| `AST_manager_listen.pl` | Running, current event parser | Deferred; event ordering incompatible | Native worker and pinned event-field adaptation; call state/disposition pass |
| `AST_manager_send.pl`, `AST_VDauto_dial.pl` | Running for native queue/session handling | Not supervised for ordinary operation | Two native workers, exact process/cgroup ownership; normal UI dialing passes |
| `AST_send_action_child.pl` | Authenticated local AMI actions | Upstream passed the secret in argv | Protected local input file; ordinary originate/hangup pass without secret argv |
| FastAGI / `park_CID.agi` | One parent, five workers; usable AGI module lookup | FastAGI present, ordinary park path incomplete | Retain FastAGI; explicit installed Perl module path and pinned park compatibility |
| AMI | Loopback, authenticated application operations | Diagnostic account intentionally read-only | Separate `almadial-native` role; DB/native scripts share protected input; diagnostic role unchanged |
| Asterisk modules/dialplan | Dial, ConfBridge, AGI, MixMonitor; native contexts | Synthetic context sufficient only for synthetic calls | Install retained pinned-build `app_dial.so`; native default/conference/park contexts and generated phone includes |
| `ADMIN_keepalive_ALL.pl` | Native maintenance/configuration generation without a second daemon owner | No active application maintenance | Root oneshot, `active_keepalives=X`; systemd owns long-running workers; native generation/reloads pass |
| Native cron | Hopper, cleanup, GMT, VM/conference, DB and recording jobs | Inactive lab schedule | Exact owned cron with per-job locks; upstream jobs retained for the selected single-server scope |
| Recording archive | Native mixed, lossless WAV retained and correlated | Raw files existed; optional mixer/compressor prerequisites unavailable | Native MixMonitor mixed WAV plus upstream VD-only archive; no uninstalled SoX/GSM pipeline scheduled |
| Agent PHP/JavaScript | Strict SQL-compatible sessions and normal UI lifecycle | Empty numeric/date values, manager inserts, frame/session defects | Hash-bound minimal r4037 derivations; POST credentials, typed NULLs, live-session checks, guarded calendar and logout cleanup |

Application-owned phones, campaigns, sessions and generated Asterisk files can
change through normal VICIdial administration. AlmaDial does not implement a
second dialer or replace the native Perl workers. Root configuration publication
uses protected preimages, explicit ownership, atomic replacement and SELinux
labels. Conflicting foreign content stops rather than being overwritten.

The AMI secret is root-owned, group-readable by `asterisk`, under a traversable
root-owned `/etc` parent; its 20 hex characters fit the pinned schema's
`varchar(20)` field. Health tests both authenticated AMI and readability as the
actual worker user. Neither the secret nor command output containing it is
exported. AMI, MariaDB, FastAGI and internal WS remain loopback.

Optional upstream FTP, lossy compression, QueueMetrics, inbound email and
destructive retention examples are not enabled. Required native maintenance is
not replaced by the synthetic-call harness. Recordings are retained locally;
public recording download/authentication is not established by this test.

## Desired state, repair and NO_OP

After fresh recursive S01–S10 verification, a protected ownership handoff binds
host, approved plan, manifest, source and native policy to the immutable file
inventory. It is a file-ownership receipt, not another execution journal.
Immutable foundation files remain checked; explicitly identified native
generated configuration is application-owned and checked through metadata,
actual dialplan, database and live runtime readiness.

The native completion uses the existing runtime action graph, exclusive lock,
durable stage journals and reconciliation. Prior attempts are retained. Missing
runtime produces file/service/verification actions instead of a successful
NO_OP. A deliberate stopped-autodial test produced two repair actions and
blocking diagnostics; the approved install restored the worker. Subsequent
healthy planning returned `actions: []`, `no_op: true`; installation exited 0.
Blocking final diagnostics also prevent an install result from claiming NO_OP.

The lab branch retains deferred workers and its original inactive maintenance
contract. Automated tests explicitly distinguish lab and remote transitions and
reject foreign foundation changes. This correction did not repeat a lab-host
installation or change accepted lab hosts.

## Ordinary authenticated workflow

Authorized host: `almadial-alm46-remote-dev`, AlmaLinux 10.2, 4 CPUs, 8 GiB,
80 GiB; machine identity and pre-change snapshots are in protected evidence.
The workstation Chromium client and VM server were distinct hosts on the
controlled bridge. Browser HTTPS/WSS used the configured FQDN and trusted task
CA, without SSH forwarding or certificate bypass. Public internet topology:
**`REMOTE_PUBLIC_HOST_NOT_RUN`**. No authorized public host was available.

The bounded database fixture follows the repository's accepted
`scripts/lab/m3_05_agent_guest.py` fixture pattern. It creates one restricted
agent, one MANUAL campaign/list/lead, the agent browser phone and an internal
customer test phone; generated credentials stay private. It changes no schema.
The administrator UI authenticated and navigated the actual campaign, user and
phone. The browser driver then used visible ordinary agent controls:

1. Log in, choose the manual campaign and register the embedded VICIphone.
2. Call/answer the agent webphone and establish the native agent conference.
3. From the normal manual campaign state, select MANUAL DIAL, enter the fixture
   lead ID and click Dial Now. Native manager processing calls the internal
   customer phone; the customer answers through its browser control.
4. Observe both real PJSIP legs and connected ICE/DTLS transports; capture the
   distinct transmitted/received browser signals in both directions.
5. Click Hangup Customer, select DC and SUBMIT; hang up the agent phone and
   LOGOUT. There are zero remaining channels and no live fixture agent row.

No hidden queue insertion, Asterisk origination or synthetic conference-only
call substitutes for those agent actions. A deliberate customer WSS disconnect
was recovered through its visible Register control. The latest successful
browser run reported no JavaScript errors.

The sequence passed before and after reboot. Post-reboot evidence:

| Check | Observed result |
|---|---|
| Agent → customer signal | PASS; pattern 0.9960, envelope 0.9629, tone agreement 0.9886, RMS 0.1075, latency 40 ms |
| Customer → agent signal | PASS; pattern 0.9960, envelope 0.9667, tone agreement 0.9649, RMS 0.1043, latency 60 ms |
| Native call record | Uniqueid `1794625522.5`, lead 46, campaign ALM46, user alm46qa, internal destination 062, DC disposition, 25 seconds |
| Recording | ID 4, `ALM46_REC_4-all.wav`, 24.80 seconds, 8 kHz mono PCM16, RMS 0.1347 |
| Recording SHA-256 | `4510e51662f9e98613ab7095d8251cfae7c83ca382d3f8548e0c14c1c5a42a2f` |
| Session cleanup | Zero live fixture agents and channels after logout |

Both media directions use real browser WebRTC transport and distinct synthetic
microphone signals; physical microphones/speakers and subjective speech quality
are `NOT_RUN`. The reused signal analyzer's legacy “Echo” note does not describe
this topology: this call used native agent/customer ConfBridge legs. The recording
is archived by native maintenance and correlated through `recording_log` and
`vicidial_log`. Its upstream location field still names loopback `/RECORDINGS`;
remote playback/download is not claimed or exposed by this correction.

## Health, regression and build identity

After reboot, all four native workers were enabled/active, with the correct
user, command and systemd cgroup. Native AMI, required modules, generated recording
dialplan, maintenance, updater heartbeat, SELinux Enforcing and persistent
firewall checks passed. Status/Doctor contain no FAIL/UNKNOWN. They return
**exit 10** for the actual short-lived test certificate's expiration warning;
the optional journal diagnostic is NOT_CONFIGURED when no journal was supplied.
These findings are retained, not suppressed. The healthy install returns exit 0.

Executed regression checks: `go vet ./...`, `go test ./...`,
`go test -race ./...`, `go mod verify`, `go mod tidy -diff`, scaffold validation,
255 Python tests, formatting, bootstrap/template Bash syntax and `git diff
--check`: PASS. Focused native tests cover source mismatch, cron parity,
readiness transitions, foreign mutation rejection, unhealthy NO_OP rejection and
lab contract preservation. All 26 native derivations were also generated against
the actual accepted source tree. No network/media redesign or historical
multi-host campaign was repeated.

The source identity is the scoped commit containing this report. The final
candidate is built **after that commit** with Go 1.27.3, Linux amd64/v1,
CGO disabled, `-trimpath`, readonly modules and explicit commit/version identity.
Protected `release.json` and the Kaneo handoff record the full commit, binary
SHA-256, reproducible second-build comparison and final-binary smoke results.
The earlier dirty binary hash is historical and is not the final candidate.

The GitHub bootstrap still pins the earlier laboratory CLI. A release owner
must review/render a new bootstrap and matching binary checksum before operator
distribution. This assignment does not publish a binary, source bundle, tag or
release. Pinned third-party redistribution restrictions remain in effect.

## Evidence and handoff

Protected evidence:
`~/almadial-lab/evidence/remote-single-server/alm46-runtime-20261114T014703Z/`.
It contains baseline/snapshot identity, retained attempts, runtime inventory,
ordinary browser drivers/results, protected synthetic audio, database correlation,
reboot and stopped-worker reconciliation results, regressions, final build and
smoke results. `SHA256SUMS` seals the files; its own SHA-256 is recorded in the
Kaneo handoff. Credentials, profiles, private keys, dumps and recordings are not
committed. Earlier evidence remains intact.

ALM-46 remains Codex-owned for independent AGY assessment. ALM-34 human
acceptance, ALM-35 and existing dependencies are unchanged. Accepted development
and QA hosts and the protected operator VM were not modified. No push, merge,
tag, public redistribution or self-acceptance is part of this correction.
