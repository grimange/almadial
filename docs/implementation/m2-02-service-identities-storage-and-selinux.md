# M2-02 — Service identities, storage and SELinux

ALM-17 implements the bounded S02 filesystem and identity stage on the accepted
ALM-16 RPM foundation. Developer qualification passed on the explicitly authorized
disposable `almadial-alm16-rpm` VM. Independent AGY acceptance remains required.
The stack manifest remains CANDIDATE, and normal installation authorization is
unchanged. No application was installed, configured, initialized or started.

## Requirements and ownership

The runtime identity and directory rules derive from
[`m0_08_replay.py`](../../scripts/lab/m0_08_replay.py), specifically `base()` and
its Asterisk AGI/sound paths. AlmaDial's private footprint derives from
[`implementation-plan.md`](../implementation-plan.md) §§5.3–5.4; that plan's
recording-path example supplies `monitorDONE`, alongside M0's live `monitor`.
Historical M0 recursive ownership commands are deliberately not reproduced.

| Identity | Management | Home | Shell | Required groups |
|---|---|---|---|---|
| `apache` | Existing RPM identity, preserved | `/usr/share/httpd` | `/sbin/nologin` | `apache` only |
| `mysql` | Existing RPM identity, preserved | `/var/lib/mysql` | `/sbin/nologin` | `mysql` only |
| `asterisk` | Reuse compatible account or create system group/account | `/var/lib/asterisk` | `/sbin/nologin` | `asterisk` only |

UIDs and GIDs are allocated by shadow-utils, within the host's system ranges from
`/etc/login.defs`. No laboratory numeric ID is a default. Primary group mismatch,
numeric collisions, interactive shells, unexpected homes, extra memberships and
missing vendor identities block provisioning. Existing numeric identities are
never reassigned. Account creation uses `groupadd --system` and `useradd --system
--gid asterisk --home-dir /var/lib/asterisk --shell /sbin/nologin --no-create-home
--no-log-init asterisk`; no password is invented or supplied. NSS inspection uses
bounded `getent passwd`, `getent group` and `id -nG` calls.

The stage neither invents a VICIdial account nor changes application users to
simplify permissions. Later application stages must qualify their process access.
Apache does not own the recording directories or join the recording writer group.

## Fixed directory policy

Only these sixteen directory inodes are managed. Existing contents are preserved;
files, secrets, application configurations and recording payloads are not written.
The type column is the distribution's existing persistent default; each full
context must be `system_u:object_r:TYPE:s0`.

| Path | Owner:group | Mode | SELinux type | Purpose |
|---|---|---|---|---|
| `/etc/almadial` | `root:root` | 0700 | `etc_t` | Installer configuration parent |
| `/etc/almadial/secrets` | `root:root` | 0700 | `etc_t` | Private credential-source footprint |
| `/var/lib/almadial` | `root:root` | 0700 | `var_lib_t` | Installer state parent |
| `/var/lib/almadial/artifacts` | `root:root` | 0700 | `var_lib_t` | Verified source cache |
| `/var/lib/almadial/backups` | `root:root` | 0700 | `var_lib_t` | Future owned-file snapshots |
| `/var/log/almadial` | `root:root` | 0700 | `var_log_t` | Private installer logs |
| `/var/log/almadial/evidence` | `root:root` | 0700 | `var_log_t` | Private diagnostics |
| `/etc/asterisk` | `root:asterisk` | 0750 | `etc_t` | Future application configuration |
| `/etc/asterisk/keys` | `root:asterisk` | 0750 | `etc_t` | Future application key references |
| `/var/lib/asterisk` | `asterisk:asterisk` | 0750 | `var_lib_t` | Application state |
| `/var/lib/asterisk/agi-bin` | `asterisk:asterisk` | 0750 | `bin_t` | Future verified AGI programs |
| `/var/lib/asterisk/sounds` | `asterisk:asterisk` | 0750 | `var_lib_t` | Future sound assets |
| `/var/log/asterisk` | `asterisk:asterisk` | 0750 | `var_log_t` | Application logs |
| `/var/spool/asterisk` | `asterisk:asterisk` | 0750 | `var_spool_t` | Application spool |
| `/var/spool/asterisk/monitor` | `asterisk:asterisk` | 0750 | `var_spool_t` | Live recording destination |
| `/var/spool/asterisk/monitorDONE` | `asterisk:asterisk` | 0750 | `var_spool_t` | Recording processing/archive destination |

M0's per-experiment build workspaces and web roots are not universal installation
paths. Build workspace policy, source extraction, application files, VICIdial data,
log files and temporary runtime paths belong to their later bounded stages. This
stage creates the accepted source cache and filesystem parents only.

Each component is opened relative to a retained parent descriptor using
`O_DIRECTORY|O_NOFOLLOW`; regular files, symlinks, special files and inaccessible
paths are rejected. Creation uses `mkdirat`, initially 0700. Ownership, mode and
label changes address the held directory descriptor; no recursive operation,
pathname replacement or deletion exists. Both the directory and its parent are
synced. Postconditions check the actual named inode as well as its metadata.

Compatible existing directories are reused. Incorrect existing owner or mode is a
CONFLICT requiring an explicit revised policy; the stage does not assume it owns
an unfamiliar directory. A label-only repair is permitted on the same compatible
inode. Existing file contents and metadata are never replaced, so this stage has
no application-file backup/replace operation. ALM-14 continues owning atomic
journal replacement and its crash recovery contract.

## SELinux and storage

Inspection requires Enforcing mode, targeted policy, effective vendor default
contexts and unchanged persistent context definitions. It hashes the vendor/local
`file_contexts` and substitution files and calls `matchpathcon -n` for every path.
A missing or different mapping/type blocks the plan. This qualified layout needs
no custom `semanage fcontext` entry; existing persistent vendor mappings supply
all sixteen defaults. Unexpected custom overrides are rejected.

The fixed helper rechecks mode and defaults, then applies the full default context
through the retained inode's `security.selinux` attribute. This avoids following
a substituted pathname. Kernel rejection fails the action. This is persistent
mapping reuse, not a temporary custom-label policy: independent `matchpathcon -V`
and `restorecon -n -v` verified every directory after execution. No `chcon`, policy
module, semanage mutation, broad relabel or enforcement change is performed.
Application confinement and future web playback access are not qualified by these
empty-directory checks; their application stages must verify actual access and
AVCs while preserving Enforcing mode.

`RecordingBudget` is a mandatory separate input, with no production defaults:
`Profile`, `Mount`, `Source`, `Filesystem`, `MinAvailableBytes`, `MinFreeInodes`.
Both recording paths must resolve to the same reviewed writable local XFS/ext4
mount. Inspection uses descriptor-based `statfs` and bounded `findmnt`; an unsafe
path, unexpected mount/source/filesystem, missing budget, insufficient bytes or
insufficient inodes blocks execution. Space and inode counts are rechecked before
and after actions; incidental count changes do not invalidate the host identity.
No reservation, quota, partition, mount or filesystem is created.

The operator explicitly approved a **laboratory-only reserve of 1 GiB free and
10,000 free inodes**, on `/dev/sda4`, XFS, mounted at `/`. After provisioning the
captured observation was 81,446,563,840 bytes and 41,269,717 free inodes. These are
point-in-time observations, not production sizing recommendations. Production
recording capacity remains a required reviewed operator input.

## Implementation and execution contract

The additions are `internal/provision/system.go` (typed policy and pure
assessment), `system_native_linux.go` (native read-only inspection and fixed
inode helper), `system_stage_linux.go` (S02 integration), their table-driven tests,
and the explicit `cmd/almadial-system` qualification command.

`SystemInputs.Prepare` consumes the existing typed configuration, ALM-11 preflight
and requirements, manifest identity, and accepted ALM-16 inventory. It requires
all 44 direct RPM requirements and existing repository trust to be satisfied.
There is no DNF resolution, metadata refresh or RPM mutation. Assessments return
SATISFIED, CREATE_REQUIRED, REQUIRED, RECONCILIATION_REQUIRED, CONFLICT,
INCOMPATIBLE or NOT_VERIFIED findings. Equivalent NSS enumeration orders produce
the same immutable stage and action graph.

The S02 graph contains only missing group/account actions and missing directories
or compatible default-label repairs. Actions have fixed IDs, linear dependencies,
preconditions and postconditions. Every command has an absolute executable,
direct argv, 30-second deadline, 4 MiB stdout and 16 KiB stderr limits. The helper
accepts a fixed policy index, not an arbitrary destination. It requires its live
S02 parent executable, the parent's retained ALM-14 lock and a matching durable
running record; direct helper invocation is rejected.

`ExecuteSystemLab` validates exact ALM-17 scope and host/stage/parent/manifest
bindings before acquiring ALM-14 ownership. It reobserves host state, creates the
existing journal document, calls `Begin` before each mutation, invokes the ALM-13
bounded graph primitive, independently verifies postconditions and persists the
outcome. ALM-14 checks the original graph dependencies before the single-action
execution primitive is invoked. ALM-15 redacted progress events describe actions
without dumping NSS, configuration or command output into operator logs.

The authorization record requires scope `ALM-17/disposable-lab`, host hash,
stage/parent/manifest digests, SHA-256 references to the recorded recovery snapshot
and operator permission, and explicit candidate acknowledgement. It records
already-given permission; hashes do not themselves establish operator consent.
ALM-16 permission is rejected. `InstallationAuthorized()` and normal wizard
candidate rejection are unchanged. The normal `almadial` CLI does not expose a
production install/resume implementation.

## Supported qualification commands

Build with the repository's pinned Go 1.27.3 toolchain. Examples below match the
actual qualified guest files; mutating commands are only for the separately
reviewed, authorized disposable host and original saved inputs.

```bash
go build -trimpath -buildvcs=false -o /tmp/almadial-system ./cmd/almadial-system
umask 077
/tmp/almadial-system inventory > inventory.json
/tmp/almadial-system inspect \
  --config /tmp/alm16-qualification/config.json \
  --requirements /tmp/alm16-qualification/requirements.json \
  --budget /tmp/alm17-qualification/budget.json > system-inputs.json
/tmp/almadial-system assess --input system-inputs.json > findings.json
/tmp/almadial-system plan --input system-inputs.json \
  --manifest /tmp/alm16-qualification/stack.lock.yaml > system-plan.json
```

`inventory` and `inspect` perform local reads only. `assess` and `plan` operate on
saved inputs without probing the host. Redirection is an explicit operator file
write, not an inspection side effect. Treat inventory, saved config and journals
as protected host data. Inputs reject unknown fields, trailing JSON, symlinks,
nonregular files and oversized input; generic input errors avoid leaking paths.

After reviewing the exact graph, recording a new recovery snapshot and permission,
and preparing the bound `SystemLabAuthorization` JSON:

```bash
/tmp/almadial-system lab-run \
  --input /tmp/alm17-qualification/approved.stdout \
  --manifest /tmp/alm16-qualification/stack.lock.yaml \
  --authorization /tmp/alm17-qualification/approved-authorization.json \
  --state-dir /tmp/alm17-qualification/journal
```

Use `lab-resume` with those same four arguments to reconcile an interrupted plan.
Use a new private journal directory for a separately reviewed fresh plan. Retain
the same installation state directory for competing/resuming operations; the
accepted lock is scoped to that installation directory. Do not unlink its lock.

## Recovery, failures and live qualification

The operator authorized this exact VM, a fresh snapshot, the sixteen directories,
asterisk identity, bounded negative fixtures/cleanup, interruption and protected
evidence. Snapshot `alm17-baseline-20261110` was taken before mutation, with
`alm16-qualified-20261110` as parent. The host SHA-256 is
`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`.
The guest was freshly verified as AlmaLinux 10.2 x86_64, targeted/Enforcing, with
725 RPMs and the 44 direct ALM-16 requirements satisfied. Required utilities were
already present; no packages were installed.

The real first graph had 18 actions. The test controller killed only the S02
qualification process after the new group existed while its durable record was
still RUNNING. Resume independently found that group satisfied and completed the
remaining 17 actions. All final journal records are SUCCEEDED. The dynamic
asterisk UID/GID was 993/993; apache 48/48 and mysql 27/27 were preserved. These
numbers are evidence, not policy constants. Fresh second assessment produced zero
actions; both fresh-plan execution and completed original-plan resume returned
NO_OP, with no user/group or directory changes.

Completed journal entries are reobserved; evidence includes dynamic identities
and inode metadata. Known absent actions can retry only after reconciliation.
A partially created directory with unexpected owner/mode, changed completed inode,
changed host, changed policy or other unknown result blocks automatic replay.
There is no rollback engine or blanket ownership repair. Filesystem operations
are not described as transaction-wide atomic. ALM-13 handles timeout, cancellation,
nonzero exits and postcondition errors; unknown partial outcomes remain explicit.

Live negative cases rejected an incompatible test-created account, foreign regular
file, symlink component, incorrect directory mode, oversized byte/inode budgets,
old ALM-16 authorization, stale plan and competing ALM-14 lock. Only explicitly
created fixtures were cleaned up. Foreign bytes were checked before cleanup;
a separate nonempty-directory sentinel retained its hash, inode, owner/group and
mode through completed resume. All sixteen labels passed native verification and
`restorecon` dry run. Repositories, keys, persistent context files, DNF history and
sorted RPM inventory were unchanged. No service was activated.

The accepted journal crash tests also ran on the actual VM: eight process-crash
boundaries around atomic replacement, plus persistence fault/write tests passed.
Unit tests additionally cover incomplete directory metadata, unknown recovery,
incorrect architecture/RPM baseline via ALM-16 regression, vendor preservation,
identity collisions, unsupported contexts, permission failure classes, bounded
observation failures, postcondition failure, cancellation, lock contention,
unauthorized mutation, safe input handling and deterministic ordering.

## Verification and evidence

Protected evidence is under
`~/almadial-lab/evidence/m2-02/20261109T181303Z/`, with a SHA-256 ledger. It includes
original host/RPM/NSS observations, snapshot and permission bindings, exact graphs,
action traces, journals before/after interruption, all native postconditions,
negative cases, idempotency, qualification harnesses, regression logs and source
hashes. Raw host data and binaries are not committed.

Required developer checks passed: Go formatting, vet, tests, race detector, module
verification/tidy consistency; repository formatting, scaffold, Python suite,
candidate-stack and configuration-generation checks; existing RPM/journal tests;
137 top-level Go tests and 743 passing test/subtest events in each of the ordinary
and race runs, 131 Python tests, 17 CLI smoke cases and 11 terminal/nonterminal
journeys. Required checks exited zero; intentional rejection journeys have their
expected nonzero exit codes recorded in protected `tests/` results. Two isolated Linux/amd64 builds of
all three commands matched. The qualification binary SHA-256 is
`5cff10332626f0d13621f93386cf07b3b7be5afbadbaadcb9b125f04d278c927`.
Hosted CI is NOT_RUN. The pre-existing optional real-source fixture test remains
skipped when its external fixture is absent.

The starting and final HEAD remain
`ec101ecb61138fe3ccf6d613dac3d20c861b77c2`, with accepted ALM-16 commit
`36c1b6eb8e183e241e1cb58d3e7c0fa95ad73e1a` verified as an ancestor. Existing operator,
recording, portal, playback, QA and local AI files were preserved. No commit,
push, merge, release, manifest promotion or downstream implementation occurred.

Independent AGY review must verify this bounded stage and its actual protected
lab evidence. Later M2 stages own application installation/configuration and
service behavior; production recording sizing and any future alternate storage
mapping require explicit reviewed policy inputs.
