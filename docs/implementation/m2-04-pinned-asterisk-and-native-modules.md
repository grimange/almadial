# M2-04 — Pinned Asterisk and native modules

ALM-19 extends the accepted ALM-18 commit
`7b3aa6e7349c8a32658c354d5cc86953e2ec16ec`. The implementation and laboratory
qualification are described below. Independent AGY acceptance is a separate gate.
The manifest remains CANDIDATE and ordinary production installation stays denied.

## Source and build contract

| Input | Pinned identity |
| --- | --- |
| Asterisk | 22.12.0, `asterisk-22.12.0.tar.gz` |
| Asterisk SHA-256 | `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` |
| Detached-signature signer | `F2FC93DB7587BD1FB49E045A5D984BE337191CE7` |
| Bundled PJPROJECT | 2.17, `pjproject-2.17.tar.bz2` |
| PJPROJECT SHA-256 | `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` |

`LoadAsteriskSource` reuses ALM-11 artifact and pinned-key `gpgv` verification,
then retains independently hashed archive bytes. It checks the exact extraction
root, `.version`, configure/Makefile, bundled PJPROJECT contract, PJSIP,
ConfBridge and `res_agi.c`. No network source selection or system PJPROJECT
fallback is provided. The source/build inputs remain distinct from installation
policy so a bounded correction to installation does not recompile qualified code.

The accepted ALM-16 exact RPM inventory is reverified through the existing native
foundation inspector, including development libraries and repository provenance.
No RPM transaction or repository refresh is performed. GCC, G++ and GNU Make
version output is observed and hashed. Build binding includes source, build policy,
toolchain, x86_64 architecture and `/usr` prefix. The existing configuration,
manifest and parent plan identities bind the stage and authorization.

The fixed workspace is `/var/lib/almadial/asterisk-s04`, root-owned mode `0700`.
Configure uses `--with-pjproject-bundled`, the verified local external cache, and
`--without-dahdi`. Menuselect disables core/extra sounds, MOH and `BUILD_NATIVE`;
it enables the M0 PJSIP/WebSocket, SRTP/crypto, ConfBridge, Echo and configuration
wizard selection. Compilation uses `make -j4`; the shipped PJPROJECT MD5 check is
also retained. No M0 source patch is needed. Configure, menuselect and make
commands/results are bounded, saved privately, and sealed with the build inventory.
`make install DESTDIR=...` stages upstream output; only the explicit verified
runtime closure is copied to live paths. Downloads are disabled during make.
Asterisk binary reproducibility is not claimed.

## Installed resources and ownership

| Resource | Owner/mode and rule |
| --- | --- |
| `/usr/sbin/asterisk` | root:root `0755`, exact compiled executable |
| `/usr/lib/libasteriskssl.so.1`, `libasteriskpj.so.2` | root:root `0755`, versioned runtime libraries; refresh native linker cache |
| `/usr/lib/asterisk/modules/*.so` | root:root `0755`, explicit 40-module closure only |
| `/var/lib/asterisk/documentation/{core-en_US.xml,appdocsxml.xslt,appdocsxml.dtd}` | root:root `0644`, upstream generated XML needed by native Stasis initialization and CLI documentation |
| `/etc/asterisk/*.conf` managed by S04 | root:asterisk `0640`, absent base include plus `almadial-s04-*.conf` |
| `/etc/systemd/system/asterisk.service` | root:root `0644`, exact approved unit |
| Existing ALM-17 storage directories | Preserve approved account/group, modes, labels and contents |

No `make samples` or upstream script installation writes over live configuration.
The managed bases are `asterisk`, `modules`, `manager`, `http`, `pjsip`,
`extensions`, `confbridge` and `logger`. A base contains only its explicit owned
include. Existing different base/include content, partial configuration sets,
unsafe ancestors, symlinks, wrong ownership/modes or conflicting systemd units
stop qualification. Unrelated files, sounds, scripts, recordings and keys are
preserved. No sounds or AGI script payload is added in this stage; the approved
AGI directory is verified through ALM-17. Actual calls, announcements and recording
under call load remain later qualification work.

## Native module contract

Every row is required in source, compiled output, installed inventory and the
running process. Modules are explicitly loaded with `autoload=no`; `res_speech`
loads before `res_agi`. Built-in core modules appear additionally in CLI output.

| Family | Required modules |
| --- | --- |
| PJPROJECT/PJSIP | res_pjproject, res_pjsip, chan_pjsip, res_pjsip_config_wizard, res_pjsip_authenticator_digest, res_pjsip_endpoint_identifier_user, res_pjsip_endpoint_identifier_ip, res_pjsip_registrar, res_pjsip_session, res_pjsip_sdp_rtp, res_pjsip_nat, res_pjsip_transport_websocket, res_pjsip_pubsub |
| Sorcery | res_sorcery_config, res_sorcery_memory, res_sorcery_astdb, res_sorcery_memory_cache |
| Security/HTTP | res_crypto, res_http_websocket, res_srtp |
| RTP/timing | res_rtp_asterisk, res_timing_timerfd |
| Dialplan/applications | pbx_config, app_echo, app_confbridge, res_speech, res_agi, app_mixmonitor |
| Bridges | bridge_simple, bridge_builtin_features, bridge_builtin_interval_features, bridge_softmix |
| Codecs/formats | codec_ulaw, codec_alaw, codec_resample, format_pcm, format_wav |
| Functions | func_callerid, func_volume, func_jitterbuffer |

The serialized module matrix contains Source/Built/Installed/Loaded, CLI capability
and Pass fields for each individual module. Disk presence alone never passes.
The verifier checks `core show version`, `module show like NAME.so`,
`pjsip show transports`, and `core show application` for ConfBridge, AGI and
MixMonitor. Unsupported commands and missing functionality fail verification.
There is no chan_sip fallback. An empty PJSIP transport inventory is required.

## Service and isolation

The unit runs foreground Asterisk with `Type=simple`, `User=asterisk`,
`Group=asterisk`, `RuntimeDirectory=asterisk` mode `0750`, umask `0027`, and
`Restart=on-failure`. Native health is checked independently of systemd's start
exit status: executable/version, PID and UID/GID, duplicate executable processes,
exact unit/drop-in state, modules, applications, sockets and SELinux contexts.
Stop must produce inactive/PID0; start must restore native health with a new
systemd InvocationID. A protected restart receipt binds that observed invocation.

AMI is enabled only on `127.0.0.1:5038`, without management users. HTTP is enabled
only on `127.0.0.1:8088`; HTTP TLS and web-manager access are disabled. The loaded
WebSocket support is loopback only. No management password or broad privilege is
created. `ss -H -lntup` verifies actual sockets and ownership; public AMI/HTTP and
SIP bindings fail. PJSIP has no endpoint or transport configuration.

Bundled PJPROJECT's DNS resolver opens one wildcard ephemeral UDP socket per IP
family (`pjlib-util/src/pjlib-util/resolver.c`, socket/bind initialization). These
are response sockets, independent of configured SIP transports. The verifier
permits at most one per family with an ephemeral port (32768–65535), while requiring
empty PJSIP transport output, closed owned configuration and exact local TCP
management sockets. No firewall or networking rule is altered.

SELinux stays Enforcing. Exact newly installed paths receive vendor-default
`restorecon`; native checks use `matchpathcon -V`. The observed Asterisk process
domain is `unconfined_service_t` under the distribution policy. This is operational
Enforcing verification, not a claim of a dedicated confined Asterisk policy.
No policy module, boolean or enforcement bypass is introduced.

## Execution, repeat assessment and recovery

`AsteriskInputs.Prepare` emits an immutable S04 dependency chain using ALM-13:
build, staged installation, owned configuration, unit, native startup and verified
restart. `ExecuteAsteriskLab` reuses ALM-14 exclusive locking, durable Begin/Finish,
structured errors, reconciliation and redacted operator logs. It provides no
replacement runner or generic filesystem rollback. Commands use explicit argv,
deadlines and bounded output. Normal unit tests are rootless fixture tests.

`cmd/almadial-asterisk` exposes `inspect`, `plan`, `lab-run` and `lab-resume` with
explicit source/signature/keyring/cache inputs. The lab authorization must match
`ALM-19/disposable-lab`, exact host/stage/parent/manifest/source identities,
operator evidence, snapshot evidence and candidate acknowledgment. Inspect and
plan perform no provisioning writes; signature verification uses its explicitly
provided private scratch directory. The ordinary production gate is unchanged.

Each effect is preceded by durable intent and followed by fresh native inspection,
including unchanged RPM, repository, identity, directory and database baselines.
Build/install are not transaction-wide atomic. Unknown or partial receipts and
configuration stop automatically; executable presence is insufficient. Completed
independently verified actions may reconcile an interrupted journal. Incomplete
build/copy or uncertain restart outcomes require explicit review; no blind replay.

Two exact early developer-install defects were reproduced and retained: omission
of the versioned libraries, then omission of generated XML (native Stasis startup
failure). Receipt-backed reconciliation recognizes only complete prior inventories
with the required added files entirely absent and preserved compiled/staged bytes.
New authorized `asterisk.01.libraries` or `asterisk.01.resources` actions stop the
failed service, preserve the old receipt, add the absent resources and reseal the
inventory. Partial new resource sets remain blocked. Restart-loop process identity
churn is normalized only for these proven incomplete states. All other identity
drift rejects a stale plan. The initial failed journals are retained.

A qualified repeated assessment emits an empty graph and NO_OP after rechecking
all runtime and foundation postconditions. It performs no compilation, install,
configuration replacement or service restart. External service/configuration drift
is requalified rather than trusted from a completed journal.

## Laboratory evidence and remaining gates

The authorized target is disposable `almadial-alm16-rpm`, AlmaLinux 10.2 x86_64.
A fresh stopped-VM snapshot `alm19-pre-asterisk-20261110` was created and verified
before mutation; no snapshot was restored. The accepted MariaDB baseline is
10.11.18, schema1748, 345 tables, 1011 indexes, local-only access, and 725 RPMs.
Database source imports and password changes are outside S04 and are not executed.

Protected evidence is under
`~/almadial-lab/evidence/m2-04/20261110T060000-alm19/`, including failed and corrected
runs, source/build identities, native module matrix, service/listener/SELinux
observations, restart, NO_OP, baseline preservation, regression and reproducible
Go build outputs. Raw material is private and excluded from Git.

Hosted CI is NOT_RUN unless separately executed. Live SIP registration, public
transport/NAT, RTP audio exchange, conference calls, recording playback and VICIdial
application operation remain M3 or their separately assigned stages. AGY must
independently reproduce the relevant native runtime and installer safety checks;
Codex does not self-accept ALM-19.

Developer runtime qualification passed: all 40 required module rows have
source/build/installed/load/PASS evidence; native CLI applications and the default
ConfBridge bridge profile load. Stop/start persistence passed and a fresh exact
assessment returned `no_op: true`, an empty action graph and no execution traces.
The systemd InvocationID remained unchanged during NO_OP. The complete accepted
MariaDB observations are identical before and after; foundation observations are
identical apart from available bytes/inodes consumed on the shared root build
filesystem. Both recording inventories are empty at final inspection; no S04
command writes to recording paths. Audit inspection reported no matching
Asterisk AVC/USER_AVC records for this boot.

Repository validation passed: Go fmt/format check, vet, full test/race suites,
module tidy consistency and module verification; 137 Python tests, scaffold,
candidate-stack and configuration-generation scripts; 17 CLI smoke cases and
11 terminal/nonterminal journeys. Five Linux amd64 Go commands were built in two
separate source directories with matching SHA-256 hashes. Detailed counts and
hashes are retained in protected evidence. The one ordinary Go test skip is the
opt-in full M0 archive-set test; actual Asterisk archive and detached signature
verification ran during every native assessment.

The final Go test and race runs each recorded 156 top-level tests, 1065 passing
test/subtest events, zero failures and one opt-in skip. The starting Git HEAD was
`7b3aa6e7349c8a32658c354d5cc86953e2ec16ec`. Concurrent operator work advanced HEAD
to `631cbe61b2ac6f4d7fa5665438c87fce2318dbd6` with an unrelated VICIdial preview
logout correction; S04 files remain uncommitted. No S04 commit or push was made.
