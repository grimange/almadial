# M2-05 — Pinned VICIdial and approved configuration integration

ALM-20 implements S05 on the accepted ALM-19 parent
`34509d29c21e519a0cea9253234d71b7b9e22cee`. Developer qualification passed on the authorized disposable VM.
Independent AGY acceptance remains required. The stack manifest is
CANDIDATE; the ordinary production authorization gate remains closed.

## Source and deployment contract

The source is VICIdial SVN r4037, version 2.14b0.5, with ZIP SHA-256
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`.
The canonical 2,511-file inventory has tree identity
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
The loader checks the archive digest, exact inventory, regular file types,
duplicate names, safe relative paths and mandatory source components before
retaining verified bytes. It derives an explicit 2,153-file deployment:
1,349 web files, 135 application scripts, 69 AGI files, 471 sounds and 129
library/attribution/support files, including the two upstream Asterisk Perl
modules in the M0-qualified `/usr/local/share/perl5/5.40` interpreter path. The source manifest emitted by the qualifier
is authoritative for the individual paths, modes, owners, labels and hashes.

The deployment uses `/var/www/html`, `/usr/share/astguiclient`,
`/var/lib/asterisk/agi-bin` and `/var/lib/asterisk/sounds`, following M0. It does
not run upstream `install.pl`, copy SQL into the deployment, or install a source
tree over an existing application. Four scripts receive only the previously
qualified procps command adjustment; original source bytes remain authenticated.

The accepted pure-Perl Net::Telnet 3.05 source archive is checked separately:
`677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743`.
Its module and attribution are installed without CPAN, RPM changes or repository
changes. Native prerequisite checks require the qualified Perl/PHP versions and
module inventory. A missing dependency fails qualification rather than triggering
an unreviewed installation.

## File and configuration ownership

Every managed file has an explicit byte identity, owner, group, mode and vendor
SELinux type. A matching existing file is preserved. An absent file can be
installed after the complete target preflight passes. A different file, unsafe
symlink, hardlink or unexpected metadata blocks automatic replacement. The
fixed Python filesystem adapter receives bounded structured input from Go,
uses held no-follow directory descriptors and publishes staged files atomically.
This is per-file publication, not a transaction across the application tree.
Unrelated files and runtime data are preserved.

AlmaDial owns six application settings files. `/etc/astguiclient.conf` is
root:apache 0640; `/etc/astguiclient-asterisk.conf` is root:asterisk 0640. Both
reuse the existing S03 application database credential. The local AMI include is
root:asterisk 0640. Two protected `options.php` files disable slave database use,
and `vicidial/old_clear` satisfies the immutable webroot prerequisite. These
files are checked exactly; populated operator configuration is not overwritten.
A separate private AMI credential is generated once and preserved thereafter.
Neither credentials nor configuration bodies appear in public plan output.

The S04 fragments remain intact. Five exact predecessor wrapper transitions add
S05 manager, dialplan, PJSIP, PJSIP wizard and ConfBridge includes. Arbitrary
wrapper edits are rejected. Generated configuration publication retains checked
preimages and durable intent; a foreign edit is not silently regenerated.

## Database ownership and application registration

`VerifyDatabaseForApplication` consumes the complete S03 schema: version 1748,
345 tables and catalog SHA-256
`10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`.
S05 never invokes initial bootstrap, imports DDL/seeds, changes grants, rotates
S03 credentials or performs an implicit schema upgrade.

The separately owned S05 writes register the exact accepted TESTast server
preimage at 127.0.0.1 with Asterisk 22.12.0, ConfBridge and configuration generation,
and set `webroot_writable=0`. A blank `system_settings.pass_key` is initialized
with database cryptographic random bytes under the same guarded action; an
existing valid key is preserved. Invalid nonblank keys require reconciliation.
Registration locks the relevant MyISAM tables, verifies exact preimages, retains
protected recovery material and verifies the postimage. This does not provide
transaction-wide atomicity across MyISAM writes.

The isolated upstream generator owns 300 inactive local conference slots
9600000–9600299 and the rebuild flag. Its input identity covers twelve actual
source tables, retains NULL and raw string identity, and excludes only declared
S05-owned registration/settings and local bridge fields. It preserves nonlocal
rows. No phones, users or campaigns are created or migrated.

The operator explicitly chose to preserve both legacy seed phones, including
all fields and their remote server references. Real endpoint configuration is
assigned to M3. A fixture executes the actual pinned upstream PJSIP generation
branch for a synthetic normal phone, a reserved extension and an empty case,
using an isolated read-only statement harness. It neither connects to MariaDB
nor publishes a phone configuration. Fixture evidence is separate from runtime
application and telephony acceptance.

## Asterisk and application integration

The M0-qualified generator block is derived deterministically from pinned
`ADMIN_keepalive_ALL.pl`. Paths are redirected into private staging, the accepted
GSM helper correction is applied, and direct reload operations are suppressed.
The generator uses the accepted local database connection and is not the full
keepalive process. Only four modern outputs are published. Legacy SIP/IAX/MeetMe
output remains private and is not activated.

The executor verifies generator input stability, compares a repeat generation,
and issues five bounded native reload commands. The accepted Asterisk 22.12.0
process, all 40 required modules and S04 configuration are freshly checked.
AMI stays at 127.0.0.1:5038. Its sole S05 probe user has `read=system` and
`write=system`; login, Ping and CoreStatus must work and Command must be denied.
This account is an S05 integration probe, not a claim that all future dialer
roles are authorized. Asterisk HTTP remains at 127.0.0.1:8088. No PJSIP transport
is created and no SIP registration, calling, RTP or browser-phone claim is made.

## Apache, PHP, Perl and scheduling

An AlmaDial-owned `/etc/httpd/conf/almadial-s05.conf` is selected with an exact
systemd OPTIONS drop-in. The vendor global configuration and existing PHP-FPM
relationship are preserved. The selected listener is 127.0.0.1:80, uses the
vendor PHP-FPM Unix socket and denies options.php, hidden Apache files and
.user.ini. Apache syntax, actual socket ownership and HTTP behavior are checked.
The web action starts and restarts Apache and verifies both observations:
anonymous admin returns 401, the agent entry returns its 200 phone-login form,
and both options.php paths return 403. Requests use no credentials, cookies,
redirect following or environmental proxy. This does not qualify an authenticated
VICIdial session. Final web/TLS deployment belongs to S07.

The final verification action performs bounded syntax checks on nine selected
Perl/AGI scripts and all 307 PHP source files, repeats the isolated phone fixture
and native reload checks, then verifies all fresh components before sealing a
private runtime receipt. S06 owns active scheduling. S05 verifies that keepalive,
dialer, FastAGI and managed cron activation are absent. It starts no automatic
calling workflow and creates no duplicate scheduled jobs.

SELinux must remain Enforcing. Managed files use distribution context types;
service operation and relevant AVC evidence are part of native qualification.
No permissive mode, broad relabel, new policy module or firewall change is used.

## Execution, NO_OP and interrupted state

`cmd/almadial-vicidial` exposes read-only `source`, `files`, `prerequisites`,
`perl-source`, `phone-fixture` and `inspect` commands. `plan` consumes a protected
inspection and emits a pure immutable graph. Native commands require all pinned
S03/S04/S05 source inputs. Only `lab-run` and `lab-resume` mutate, under the
existing ALM-13 runner, ALM-14 lock, durable journal and narrow ALM-20 laboratory
authorization. Authorization binds host, parent/stage/manifest/source/dependency
identities, recovery snapshot reference and operator scope. Production remains
denied for the CANDIDATE manifest.

The seven deterministic actions are files, Perl dependency, protected
configuration, guarded registration, isolated generation, local web activation
and independent runtime verification. Intent is persisted before mutation and
fresh postconditions are required before journal completion. Every subsequent
action must preserve the accepted foundation and unrelated component identities.

A complete fresh assessment produces an empty graph and NO_OP, without recopy,
credential rotation, initial import, regeneration or service restart. A missing
matching application file can be safely completed. An interrupted uncertain
configuration, registration or publication is blocked unless fresh observations
prove the action completed; journal success alone is insufficient. Foreign edits
and incompatible predecessor state require reconciliation. There is no generic
rollback engine.

## Qualification record and limitations

Protected evidence lives under
`~/almadial-lab/evidence/m2-05/20261110T025748Z-alm20/`.
The verified disposable target is `almadial-alm16-rpm`, machine ID
`0fffa9a4683f35b696f0b6ee7330734b`; pre-S05 snapshot is
`alm20-pre-vicidial-20261110`.

The first native run completed through generation, then correctly failed the
web postcondition when upstream initialized pass_key during its first HTTP
request. The failed state is retained in snapshot
`alm20-web-key-defect-20261110`. After the bounded correction, the VM was restored
to the verified pre-S05 snapshot and independently re-inspected before retry.
The recovered catalog, seed-phone hash and 725-package inventory match the
original observations; SELinux remains Enforcing.

Evidence handling incident: an obsolete baseline helper overwrote files named
`pre-s05-baseline` with failed-run observations. These files are not original
pre-install evidence. The helper now uses exclusive file creation and unique
labels. `recovered-pre-s05-*` captures the restored snapshot independently;
`failed-run-confirmed-*` records the failed state. Historical snapshots are
preserved. A rejected raw configuration archive was not exported; subsequent
exports use redacted diagnostics and integrity hashes.

A second native run passed the web gate and correctly stopped at final Perl
verification: the upstream AGI module was in the application library directory
rather than the interpreter search path. The two modules now use the exact
M0-qualified versioned runtime path, without altering upstream bytes. Snapshot
`alm20-agi-path-defect-20261110` preserves the failed state before restoring the
pre-S05 snapshot. Non-root module loading is now an explicit runtime check.

The third clean installation passed all seven actions. The final verification
loaded `Asterisk::AGI` as the `asterisk` account, checked nine Perl/AGI scripts
and all 307 PHP files, repeated the isolated phone fixture, reloaded the five
owned Asterisk integration points and sealed the runtime receipt. Fresh native
inspection passed with all 40 accepted Asterisk modules operational. The service
still runs as `asterisk`; Apache and PHP-FPM run in `httpd_t`, MariaDB in
`mysqld_t`, and Asterisk retains the accepted `unconfined_service_t` domain.
SELinux is Enforcing and the boot AVC query returned no matches. This records
the actual service domain; it does not claim new production confinement.

Observed management/application TCP listeners are exactly localhost port 3306
(MariaDB), 5038 (AMI), 8088 (Asterisk HTTP), and 80 (Apache). Existing SSH
listeners and Asterisk's accepted ephemeral UDP sockets were preserved. No
PJSIP transport is configured. The PHP-FPM socket is root:root 0660 with its
vendor access ACL; local application responses prove PHP handling. No firewall,
RPM, repository, service-identity or recording-storage changes occurred.

Two independent fresh assessments and executions returned `no_op=true` with
zero action traces. The complete observed file identity (content, inode,
mtime, mode and ownership) and all four service PID/invocation/start-time
observations remained identical across NO_OP and recovery. The final qualifier
binary was rebuilt reproducibly and exercised on the VM before the final
assessment. The final catalog, both complete seed-phone rows, the original
three protected credentials, recording tree, repositories, identities and all
725 RPMs match the independently restored baseline. The only additional secret
is the explicitly owned S05 AMI credential; existing secrets were preserved.

A native foreign-edit test appended a comment to the installer-owned generated
ConfBridge file while holding the stage lock. Fresh inspection rejected it at
the generated-configuration gate. A finally block restored the exact preimage
without reloading the changed file. A controlled crash checkpoint copied the
completed journal into an isolated private directory and changed only the last
action to running with no outcome. `lab-resume` re-inspected every completed
component against the actual host, reconciled all seven actions, and returned
NO_OP with no replay. This is a controlled journal interruption fixture, not a
claim of killing the production process. The two earlier actual failed native
installations, their incomplete journals and snapshots separately demonstrate
that partial installation does not pass the completion gate. Deterministic
fault injection additionally covers interrupted file deployment, registration,
generation and safe matching-file completion; uncertain non-idempotent states
remain blocked. A concurrent native execution attempt failed with no actions
while the first execution held the exclusive lock.

| Developer verification | Actual result |
| --- | --- |
| Go 1.27.3 `go test ./...` | PASS: 197 top-level tests, 1,295 passing test/subtest events |
| `go test -race ./...` | PASS: same 197 / 1,295; no failed events |
| `go fmt ./...`, formatting check, `go vet ./...` | PASS |
| `go mod tidy -diff`, `go mod verify` | PASS; no module changes |
| `python3 -m unittest discover -s scripts/tests -v` | PASS: 164 tests |
| `python3 scripts/validate_scaffold.py` | PASS |
| `python3 scripts/verify_candidate_stack.py` | PASS local consistency; its runtime check is NOT_RUN |
| `python3 scripts/verify_config_generation.py` | PASS redacted fixture replay; separate native evidence above |
| Main CLI smoke / real PTY journeys | PASS: 17 / 11 |
| Repeated Linux amd64 Go builds | PASS: main CLI and S05 qualifier pairs have matching SHA-256 |
| `git diff --check` | PASS |
| Hosted CI | NOT_RUN |

The main CLI build SHA-256 is
`06ebba4b4b4f121784766cdddaa03d9b42d3709af6295fb2688e4797fca7f5d3`.
The S05 qualifier build SHA-256 is
`1e1b77d959d98ea83f9c0f760073966feaa08facc8796aefc2840c219bf9f053`.
These are Go binary reproducibility checks, not a byte-for-byte reproducibility
claim for source-built Asterisk. Tests that need loopback sockets were rerun
with appropriate sandbox permission after the restricted run could not bind;
the successful full runs are retained separately from the failed attempt.

The evidence ledger records protected source/deployment manifests, native
initial/NO_OP/recovery results, redacted diagnostics, baseline comparisons,
regression logs and the bounded task-owned source snapshot. Raw configuration
and credentials are excluded from exported evidence. All source changes remain
uncommitted on the accepted parent; unrelated tracked/untracked operator work
is preserved. No push, merge or release occurred.

For reproduction, build `./cmd/almadial-vicidial` with Go 1.27.3. Use its `source`
and `perl-source` commands with the manifest and exact pinned archives, then
`inspect` with the eight common source flags documented by `--help` and the
accepted S04 inspection. Feed that protected inspection to `plan`. `lab-run`
and `lab-resume` additionally require the explicit ALM-20 authorization and
private journal directory. The protected evidence retains exact argv, plan,
authorization identity, source manifest and host observations for independent
QA; ordinary commands do not grant production installation permission.

Independent AGY QA remains required. S06 scheduling, S07 final web/TLS setup,
authenticated VICIdial sessions, real endpoints, registration, calls, RTP and
browser-media acceptance are deferred to their assigned stages. ALM-21 remains
unstarted.

## Independent review request

Google Antigravity (AGY), using the highest approved Gemini Pro-class reasoning
model with high effort, should independently reproduce source/inventory checks,
owned-file preservation, S03 non-mutating verification, local AMI authorization,
generator derivations and reload behavior, Apache/PHP responses, non-root Perl
loading, SELinux observations, NO_OP and recovery/foreign-edit tests. Use
`kaneo-local`, Git, Go 1.27.3, Python, Perl, MariaDB/Asterisk CLI, systemd,
SELinux tools and the specifically authorized disposable VM. Retain Codex
ownership, preserve the original evidence and journals, and do not start ALM-21
or accept ALM-20 solely from this developer report.
