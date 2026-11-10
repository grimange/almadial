# M2-06 — Cron, keepalive and persistent runtime services

Implementation owner: Codex. Independent reviewer: Google Antigravity (AGY).
Parent: accepted ALM-20 `0f36041d566e25699656160f502f73bafceec664`.
This is a disposable-laboratory S06 contract, not production or M3 acceptance.
The manifest remains CANDIDATE.

## Requirement decision and provenance

The canonical ALM-21 task requires M0-qualified runtime ownership and schedules,
actual stop/start and reboot evidence, foreign-schedule preservation and fresh
state assessment. Accepted M0 evidence does not provide a qualified cron
expression. It qualifies systemd-owned FastAGI, updater and manager listener,
while excluding the complete native `ADMIN_keepalive_ALL.pl` suite. See
[M0-06](../compatibility/m0-06-telephony-runtime.md),
[M0-07](../compatibility/m0-07-viciphone-browser-runtime.md) and
[M0-08](../compatibility/m0-08-baseline-reproduction.md).

The operator explicitly selected a minimal S06 laboratory qualification with
**dialing inactive**, then selected **updater/listener configured but inactive;
FastAGI and deferred cron definitions qualified**. The S05 AMI probe account
continues to deny Command actions; updater/listener require those actions.
The decision is recorded in ALM-21 comment `qwmqxczkksmuxlii474t0qsw` and its MCP
read-back. This decision resolves the missing M0 cron contract; it does not
claim that an executable application cron job was M0-qualified.

| Component | Classification | Source / purpose | Owner and policy | Postcondition |
| --- | --- | --- | --- | --- |
| FastAGI | REQUIRED_ACTIVE | r4037 `bin/FastAGI_log.pl`; M0-06/08 local AGI runtime | `almadial-s06-fastagi.service`, `asterisk:asterisk`, enabled, restart on failure | one parent, five native idle prefork children, exact argv/cgroup/identity, local 4577, database ready |
| AST_update | REQUIRED_CONFIGURED | r4037 `bin/AST_update.pl`; M0-07/08 | `almadial-s06-updater.service`, disabled/inactive, manual start refused | exact source/config/unit; no process |
| AST_manager_listen | REQUIRED_CONFIGURED | r4037 `bin/AST_manager_listen.pl`; M0-07/08 | `almadial-s06-listener.service`, disabled/inactive, manual start refused | exact source/config/unit; no process |
| ADMIN_keepalive_ALL | DEFERRED | r4037 `docs/BASE_INSTALL.txt:432-433` | no active owner; commented upstream schedule only | no process or executable cron line |
| AST_VDhopper | DEFERRED | r4037 `docs/BASE_INSTALL.txt:466-467` | commented upstream schedule only | no campaign/hopper processing |
| MariaDB, Asterisk, Apache/PHP-FPM, crond | ALREADY_SATISFIED foundation | accepted ALM-18/19/20 units and identities | existing systemd ownership; enable existing web units for persistence | active/enabled; previous configuration preserved |

There is one authoritative owner per active process: systemd owns FastAGI's
parent, and pinned Net::Server::PreFork owns its children. Cron launches none of
these. Native keepalive is not run, including its misleading `-test` option:
source inspection found that option does not suppress its maintenance writes.
Activating that suite concurrently would compete with systemd ownership.
No Go process supervisor or additional scheduler is introduced.

## Owned cron, scripts and configuration

`RuntimeInventory`, `RuntimeCron` and `runtimeUnit` are a fixed typed policy in
`internal/provision/runtime.go`. Generation has no caller-supplied schedule,
command, execution account or activation flag. The sole cron definition is
`/etc/cron.d/almadial-s06`, root:root 0644, `system_cron_spool_t`:

```cron
# AlmaDial S06 / ALM-21: dialing-inactive laboratory contract
# No active application cron jobs are authorized. Do not uncomment these
# examples without a new activation and process-ownership review.
SHELL=/bin/bash
PATH=/usr/sbin:/usr/bin:/sbin:/bin
CRON_TZ=UTC
MAILTO=""
# native-keepalive DEFERRED: r4037 docs/BASE_INSTALL.txt; conflicts with systemd FastAGI
# * * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl
# hopper DEFERRED: r4037 docs/BASE_INSTALL.txt; campaign processing inactive
# * * * * * root /usr/share/astguiclient/AST_VDhopper.pl -q
```

There are **zero active application cron entries**. UTC and the explicit PATH
apply if a later reviewed policy introduces entries; comments do not execute.
No credentials appear in cron or unit files. Assessment reads `/etc/crontab`,
`/etc/cron.d`, user spool and hourly/daily/weekly/monthly surfaces. It hashes
foreign contents and metadata, rejects active competing application schedules,
and never rewrites operator files. Exact managed content rejects duplicates,
wrong identities, altered commands and unapproved activation.

Nine exact files are owned: that cron file; `/etc/astguiclient-s06.conf`;
`/usr/share/astguiclient/almadial-s06-{FastAGI_log,AST_update,AST_manager_listen,ready}.pl`;
and `/etc/systemd/system/almadial-s06-{fastagi,updater,listener}.service`.
Scripts are root:root 0644 `usr_t`, invoked by `/usr/bin/perl`. Units are
root:root 0644 `systemd_unit_file_t`. Protected configuration is root:asterisk
0640 `etc_t`. The existing root-only S03/S05 credential references supply its
bytes through bounded transient stdin, never argv, public plans or journals.

FastAGI derivation verifies pinned source SHA-256
`f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a`.
It reuses the M0 corrections: strict-mode call_log NULL/zero values, loopback
bind, asterisk identity, and a separate protected configuration path. The
optional main-package pool settings are retained in configuration but r4037's
`VDfastAGI` package boundary means they do not control the native pool. The
observed qualified idle model is one parent and five children. No client
requests, telephony workload or enlarged pool is qualified here.

Deferred updater/listener copies change only the protected configuration path.
Their activation additionally requires review of the M0 Asterisk 22/PJSIP
adaptations, runtime AMI grants and workload authority. Their definitions are
not a claim that those deferred workers are ready to run unchanged.

## systemd and readiness

The active unit has `Requires=mariadb.service`, `After=mariadb.service`,
`User=asterisk`, `Group=asterisk`, working directory `/usr/share/astguiclient`,
explicit PATH, and these commands:

```ini
ExecStartPre=/usr/bin/perl /usr/share/astguiclient/almadial-s06-ready.pl
ExecStart=/usr/bin/perl /usr/share/astguiclient/almadial-s06-FastAGI_log.pl
Restart=on-failure
RestartSec=5
KillMode=control-group
TimeoutStartSec=30
TimeoutStopSec=30
NoNewPrivileges=true
PrivateTmp=true
UMask=0027
```

Start limiting is three attempts in 60 seconds. The readiness helper reads the
protected configuration, requires local database/schema/port settings and
performs a bounded `SELECT 1` over `/var/lib/mysql/mysql.sock` as the existing
application account. It has a ten-second alarm and five-second connect timeout.
Unit activation alone is insufficient: assessment repeats database readiness,
requires all foundation units active/enabled, verifies the exact parent and
child argv, all effective/real/saved UID/GIDs, PPIDs and unit cgroup, and binds
socket PIDs to those observed workers. Missing children remain unhealthy;
foreign owners, wrong users and excess workers conflict. Health evidence omits
volatile PIDs and invocation IDs.

Deferred units require and order after MariaDB and Asterisk, include
`RefuseManualStart=yes`, and must remain disabled/inactive without drop-ins.
The accepted MariaDB, Asterisk and web unit contents are preserved. Apache and
PHP-FPM were running but disabled at entry: the authorized pre-S06 snapshot
stop/start exposed that gap. Their previous running state was restored, then
the journaled S06 service action enabled those existing units. It did not
restart them or change their configuration.

## Execution, idempotency and recovery

`cmd/almadial-runtime` exposes read-only `inspect`, pure `plan`, and guarded
`lab-run` / `lab-resume`. It consumes the accepted S05 input, verified pinned
VICIdial/Perl/Asterisk sources and the immutable parent graph. It does not replay
S01–S05. The S05 default verifier still rejects active jobs; a private handoff
permits S06's closed runtime only after the S06 guard has verified it.

S06 uses the accepted bounded runner, structured graph, operator logger and
atomic journal. Actions are `runtime.00.files`, `runtime.01.services`, and
`runtime.02.verify`, with dependencies and fresh postconditions. Correct
components are omitted. Policy identity binds the inventory, generated bytes,
source derivation and native adapter. Authorization binds exact host, stage,
parent, manifest, policy, snapshot and operator-evidence hashes and requires
CANDIDATE acknowledgment and dialing inactive. An exclusive `journal.Acquire`
lock and durable `Begin` precede every provisioning action.

New files use no-follow directory checks, exact ownership/mode/label checks,
fsync and atomic create-if-absent publication. Foreign or unknown content is a
conflict. Valid completed interrupted actions reconcile from current state;
missing matching files may complete idempotently. Unknown service transitions
block automatic replay and require a new assessment. A successful old journal
never substitutes for current processes, files, sockets or database readiness.

The initial laboratory run reproduced a substantive readiness defect: TCP
`127.0.0.1` did not match the accepted socket-local database grant. No grant was
broadened. A bounded two-file correction recognizes only the exact initial
S06 configuration/readiness preimages, verifies metadata, preserves root-only
`.alm21-before-local-socket` backups and atomically publishes corrected bytes
under the file action's durable intent. Other mismatches remain conflicts.
Failed initial/service journals and diagnostics are preserved. Unit tests
inject interruption after each action and verify reconciliation without replay.

Fresh NO_OP performs full inspection and returns an empty graph, `no_op: true`
and zero traces. It does not rewrite files, reload units, restart a service or
launch another parent. A stopped FastAGI unit produces service/verification
actions; the actual controlled stop/start qualification executed both.

## Laboratory evidence and security

The exact authorized target is disposable Multipass VM `almadial-alm16-rpm`,
machine ID `0fffa9a4683f35b696f0b6ee7330734b`, host identity SHA-256
`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`.
The operator authorized its stop/snapshot/start and one later controlled reboot.
Recovery snapshot: `alm21-pre-runtime-20261110`. Historical evidence and snapshots
are preserved.

Protected evidence root:
`~/almadial-lab/evidence/m2-06/20261110T122348Z-alm21/` (0700).
It contains canonical/ownership MCP read-backs, exact source/plan/authorization,
failed and corrected execution evidence, service/process observations,
controlled stop/start/restart, native-child recovery, reboot checkpoints,
baseline comparisons, regressions and a SHA-256 ledger. Large evidence uses
private file transfer because Multipass stdout collection stalled for large
JSON; failed collection attempts remain recorded separately.

Actual checks preserve SELinux Enforcing, loopback MariaDB 3306, AMI 5038,
Asterisk HTTP 8088, Apache 80 and FastAGI 4577. Existing SSH and loopback chronyd
are preserved. The unchanged S04 verifier checks Asterisk's accepted ephemeral
PJPROJECT UDP resolver sockets and absence of public SIP transport. No AMI
grants, firewall rules or PJSIP transports are changed. Process contexts are
reported as observed, without claiming a new dedicated Asterisk confinement
policy. No SELinux exception or custom policy is installed.

Preservation checks compare schema 1748/catalog identity, both legacy phones,
725 RPMs, repository definitions, credentials, service identities, Asterisk
configuration, recording files and campaign/list/hopper/manager/auto-call/
recording/call-log data. S05 foundation verification also checks Asterisk
22.12.0, all 40 required modules, pinned application files, Perl modules,
protected integration and local interfaces. No schema import, broad RPM action,
application reinstall or call generation occurs.

## Reproduction and qualification results

Build with Go 1.27.3:

```bash
go build -trimpath -buildvcs=false -o dist/alm21/almadial-runtime ./cmd/almadial-runtime
python3 scripts/lab/m2_06_qualify.py inspect LABEL --evidence PRIVATE_EVIDENCE
python3 scripts/lab/m2_06_qualify.py plan PLAN --input LABEL --evidence PRIVATE_EVIDENCE
python3 scripts/lab/m2_06_qualify.py lab-run RUN --input LABEL --plan PLAN --state STATE --evidence PRIVATE_EVIDENCE
```

The driver has a fixed VM/machine identity, fixed pinned archive paths and
snapshot read-back. Its generated authorization does not authorize another
host. Exact CLI source flags are in `--help`; private accepted inputs and
credential files remain in the protected lab. Reboots are explicit operator
checkpoints, not automatic installer behavior.

The single authorized reboot completed successfully. Boot ID changed from
`cdd8675e-8099-4f19-acc7-eed24dc1079b` to
`9ecd6163-6300-4898-8e93-262019adc5e1`. No manual service repair followed it.
MariaDB, Asterisk, Apache, PHP-FPM, crond and FastAGI returned active and enabled;
updater/listener stayed inactive and disabled. FastAGI had one parent and five
native children, running as `asterisk` in the unit cgroup with observed SELinux
domain `unconfined_service_t`. SELinux remained Enforcing; the post-boot AVC
query returned no matches. Asterisk reported zero channels, zero active calls
and zero calls processed.

Post-reboot inspection produced an empty plan; execution returned `no_op: true`
with zero traces. Resume of the completed controlled-start journal also returned
fresh NO_OP after reboot. The controlled stopped-state run executed exactly
service and verification actions. Controlled restart and native child recovery
both restored the expected six-process pool without duplicate supervisors.

Developer verification, all exit 0:

- Go 1.27.3: formatting, vet, module verification and tidy consistency; ordinary
  and race runs each passed 202 top-level tests / 1,318 test and subtest events.
- Python: 181 tests, including 17 focused S06 adapter tests; scaffold, candidate
  stack and configuration-generation verification passed.
- CLI smoke: 17 cases; terminal/nonterminal PTY journeys: 11 cases.
- Linux amd64 reproducible builds: each pair was byte-identical. Main CLI
  SHA-256 `1a9a15c5a1edb21ad7d36c41c375d90d38650d61cf136b4a2d5e7d10841c3b9a`;
  runtime qualification CLI SHA-256
  `c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a`.
  Main CLI used `scripts/build_go.py`; runtime builds used `CGO_ENABLED=0
  GOOS=linux GOARCH=amd64 go build -mod=readonly -trimpath -buildvcs=false`.

Before/after baseline comparisons passed for the database catalog, both phones,
RPM inventory, repository definitions, protected credentials, identities,
Asterisk configuration, monitored recordings and selected application tables.
The recording comparison covers `/var/spool/asterisk/monitor`; a final
`monitorDONE` inventory is retained separately and is not claimed as a measured
pre/post comparison. Failed early qualification and collection attempts remain
in protected evidence alongside their corrections.

Hosted CI and independent AGY QA are NOT_RUN by Codex. Browser agent sessions,
outbound calls, RTP/audio and runtime AMI privilege changes remain outside this
task. The minimal cron contract contains zero active application entries;
activating the deferred examples requires a later explicit technical decision.
