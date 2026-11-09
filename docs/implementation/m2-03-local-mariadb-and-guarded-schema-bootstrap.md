# M2-03 — Local MariaDB and guarded VICIdial schema bootstrap

ALM-18 developer qualification passed on the explicitly authorized disposable
AlmaLinux 10.2 VM. Independent AGY QA and maintainer acceptance remain separate
gates. This document does not grant production installation authority; the
stack manifest remains CANDIDATE. The ordinary installer still rejects it.

## S03/S05 ownership contract

S03 is the sole writer of the initial r4037/schema1748 database. It owns local MariaDB configuration, protected bootstrap/application credentials, initial database creation, the verified SQL import, baseline reference/example data, grant verification, and bootstrap recovery evidence. The source is the authenticated M0 VICIdial archive pinned by `manifests/stack.lock.yaml`. The import order is the accepted M0 derivation of `extras/MySQL_AST_CREATE_tables.sql`, then the original `extras/first_server_install.sql`. The derivation removes exactly the three legacy privilege statements identified by `scripts/prepare_vicidial_r4037.py`; it does not change the upstream table definitions or reference data.

S05 consumes and verifies this schema identity. It may install the pinned application files and perform separately reviewed server/application registration and configuration updates. It must not import either initial SQL script, recreate the database, reset seeds or passwords, or run a schema upgrade implicitly. The pinned upstream `install.pl` reads the expected schema version from the CREATE script; this version read is permitted. Future schema changes require a separate reviewed assignment.

A completed journal entry is not proof of schema health. S03 rechecks actual database metadata, required seed records, accounts, transport and service state. A complete schema with interrupted journal persistence can be reconciled from those observations. Partial DDL, conflicting credentials, incompatible metadata or unknown database state stops for explicit recovery review. No automatic DROP, TRUNCATE, whole-schema replay or inverse transaction is permitted. MariaDB DDL is not transaction-wide atomic.

Before any database mutation, the operator must authorize the exact disposable target and bounded operations and a fresh recovery snapshot must be recorded and verified. Restoring a snapshot is a separate operator recovery decision, not an automatic retry. This boundary also applies to irreversible initial import and credential persistence.

## Accepted inputs and implementation

The recorded working HEAD is `a8ba1afecc54ead750c6b7c4c992cca832e1d723`.
The accepted ALM-17 baseline `39dcfb4f965c4a9a5cdf351b4290702b974bf4ac`,
ALM-16 and the accepted M1 gate are ancestors. The intervening operator QA
commit changes no Go command, package or module input and is preserved.
The implementation extends `internal/provision/`:

| File | Responsibility |
| --- | --- |
| `database.go` | Fixed S03 policy, typed observations, schema classification and read-only S05 recognition contract |
| `database_source.go` | Manifest/artifact verification, retained verified SQL bytes and exact M0 derivation |
| `database_native_linux.go` | Native package/filesystem, MariaDB, systemd, listener and SELinux observations; fixed approved commands |
| `database_files.py` | Embedded bounded helper for the three protected credential files and one owned configuration file |
| `database_stage_linux.go` | Immutable S03 graph and ALM-13/14 execution/reconciliation integration |
| `database_test.go` | Rootless state, authorization, postcondition, interruption and NO_OP fixtures |
| `cmd/almadial-db/` | Explicit disposable-lab qualification interface |

The existing runner gains `RunInput`: transient input is copied and limited to
4 MiB, and uses the same deadlines, cancellation, process-group cleanup and
bounded/redacted output as `Run`. Input is absent from command serialization,
graphs, traces and journals. SQL and credentials do not become argv values.
There are no new Go dependencies or alternate runner, journal or planner.

## MariaDB baseline and local transport

The required server/client baseline is MariaDB
`3:10.11.18-1.el10_2.alma.2.x86_64`. Inspection reuses the accepted RPM and
filesystem providers, including repository/key verification, `mysql` identity,
SELinux Enforcing/targeted policy and the explicit storage budget. A changed
foundation, vendor configuration or systemd unit invalidates the saved plan.

The selected contract combines Unix socket access with loopback-only TCP:

```ini
[mariadb]
bind-address=127.0.0.1
port=3306
skip-name-resolve
socket=/var/lib/mysql/mysql.sock
character-set-server=utf8mb3
collation-server=utf8mb3_unicode_ci
sql-mode=STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION
```

Only the absent `/etc/my.cnf.d/zz-almadial.cnf` is created, mode `0640`,
`root:mysql`. Different existing content is a conflict. Vendor files are hashed
and preserved. Effective `mariadbd --print-defaults` options are checked before
first service start; fresh runtime queries and `ss -H -lntp` verify the actual
bind and reject any additional nonlocal MariaDB listener. General, slow and
binary SQL logging must be off before secret-bearing SQL is sent.

The M0 application contract uses `localhost` and Unix sockets, as documented in
[M0-04](../compatibility/m0-04-vicidial-runtime.md) and
[M0-05](../compatibility/m0-05-vicidial-config-generation.md). The application
account is therefore `almadial@localhost`, authenticated through the socket.
An explicit TCP authentication probe was rejected under `skip-name-resolve`;
there is no TCP application grant. S05 must preserve the qualified socket
client configuration. The loopback TCP listener is local only and does not
establish an independently qualified application TCP connection contract.

The vendor data directory remains `mysql:mysql`, mode `0755`,
`mysqld_db_t`. The native socket is `mysql:mysql`, mode `0777`,
`mysqld_var_run_t`; this is the vendor socket type and metadata, not a regular
world-writable credential or configuration file. SQL authentication remains
mandatory. The existing default SELinux policy is verified with `matchpathcon
-V`; no policy module, boolean, firewall change or enforcement bypass is used.

## Credentials and grants

Bootstrap administration uses MariaDB's existing local root `unix_socket`
authentication. No root database password is invented. The configuration must
reference the following approved files through ALM-10 `file` secret references:

| Path | Owner/mode | Purpose |
| --- | --- | --- |
| `/etc/almadial/secrets/database` | `root:root 0600` | Unique application database password |
| `/etc/almadial/secrets/admin` | `root:root 0600` | Unique VICIdial initial administrator `6666` password |
| `/etc/almadial/secrets/database-client.cnf` | `root:root 0600` | Private native client option file using the same application password |

Each new password uses 24 cryptographically random bytes encoded as 48 hex
characters. Parents must be root-owned, non-writable by other users and free of
symlinks. Files must be regular, singly linked, and have the exact owner/mode.
Creation uses `O_EXCL`, fsyncs the file and its parent, and never replaces an
existing file. Incomplete file sets are conflicts requiring review; normal
reruns preserve credentials. The helper prints only safe state and a digest of
the high-entropy credentials. SQL outputs are cleared from mutating traces and
the runner redacts registered secrets. Evidence contains metadata, never file
contents or password hashes.

The M0 schema-local policy is retained: `SELECT, INSERT, UPDATE, DELETE,
LOCK TABLES ON asterisk.*`, with no GRANT OPTION, global privileges, additional
host identity, role, column, table or routine grant. Root performs initial DDL;
the application account has no schema-management privilege. Existing conflicting
accounts or unknown authentication are rejected without rotation or escalation.
The schema's `6666` administrator password is set from the protected reference
after the original seeds, before bootstrap can be recognized complete.

## Verified SQL and database identity

The archive is the accepted M0 `vicidial-r4037.zip`, not a current trunk
download. `artifacts.VerifyLocal` verifies the manifest artifact; retained bytes
are independently hashed again before ZIP extraction. Exact identities are:

| Input | SHA-256 |
| --- | --- |
| r4037 archive | `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` |
| Original `MySQL_AST_CREATE_tables.sql` | `8a210454a6f42fd3ef0054033a5c4396a0cd3615442ebd53491dfd2efa38de40` |
| Original `first_server_install.sql` | `c215f549c3a6dc39e20224e666efd6d99aa85b93910fe8dbaf7daa62a962203e` |
| Derived CREATE SQL, matching M0 | `0992448a2607dbad6841250a09b68dafd360cb4fa44f8589d89526ef16522af8` |
| Source/writer identity | `2b81ce6f910e5b533d0db4fa78b523c4eed71188588f01a750d22419f1c4b021` |
| Native catalog reference | `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8` |

Import order is database creation with explicit `utf8mb3_unicode_ci`, derived
CREATE SQL, original first-server seeds, then the protected administrator
password update. The exact three legacy global-privilege statements are removed
once each, matching `scripts/prepare_vicidial_r4037.py`. Missing or changed
statements fail source verification. There is no generic SQL parser or schema
generator. Privileges are a separate S03 action.

The reference catalog was derived with native MariaDB from an independently
journaled import into the separately authorized `almadial_alm18_test` fixture
database before the main bootstrap. Recognition checks schema version **1748**,
**345 tables**, **1,011 index-column metadata rows**, and a deterministic SHA-256
over ordered table/engine/collation, column/default/nullability, index, view,
routine and trigger metadata. No routine/view/trigger is present in this pinned
schema. Auto-increment counters and mutable application rows are excluded.
Table count alone cannot establish success. The original MyISAM/MEMORY engines
and MariaDB's actual accepted index prefixes are preserved; no InnoDB rewrite
or foreign-key guarantee is implied.

All 33 seeded tables must be nonempty. Additional mandatory checks recognize
administrator `6666`, group `ADMIN`, statuses `NEW`/`SALE`, three QC codes and
server `TESTast`; the administrator password must match its protected reference.
Schema charset/collation, SQL mode, local transport, root authentication,
application authentication and exact grant policy are independently checked.
This verifies the required baseline, not a checksum of mutable application data.
Upstream example phone/server registrations remain inactive source seed data;
S05 must review them and configure appropriate protected runtime credentials
before activating any application. ALM-18 starts no telephony or web service.

## Inspection, action graph and recovery

Read-only inspection uses native RPM/NSS/filesystem, systemd, socket and SQL
queries. It creates no directories or cache and performs no DNF refresh. Host
output contains private identity/network observations and must be protected.
`Prepare` is pure and builds the existing immutable parent plan and ALM-13
graph. It binds configuration, manifest, source, policy and host identities.
It emits only necessary operations in the following dependency order:

1. `database.00.config` — create the absent owned local configuration.
2. `database.01.credentials` — create absent protected credential files.
3. `database.02.service` — initialize/enable/start MariaDB through systemd.
4. `database.03.bootstrap` — verified initial schema into an observed empty target.
5. `database.04.grants` — create only the missing local application account.
6. `database.05.restart` — restart and verify persistence, including a changed systemd invocation ID.

Missing configuration/credentials on an already running incompatible instance
are not silently repaired. All commands use fixed executable paths/argv,
bounded runtime (three minutes by default) and bounded output. Secrets travel
through the existing runner's transient stdin or protected client option file.

`DatabaseLabAuthorization` binds the exact disposable host, stage, parent plan,
manifest, source, explicit operator evidence and verified fresh snapshot record.
This gate is specific to `ALM-18/disposable-lab`; it cannot authorize the ordinary
installer or change `InstallationAuthorized()`. Snapshot creation and verification
are operator/lab responsibilities recorded before invocation, not a caller
request to create or automatically restore a snapshot.

Mutations acquire the existing ALM-14 exclusive lock, persist `Running` intent,
execute one accepted action, inspect fresh state and persist the outcome.
Postcondition failure is an ALM-13 failure even after process exit 0. All SQL and
credential actions use `RetryNever`. Completed actions are reverified on resume.
Current state is classified as EMPTY, COMPLETE, PARTIAL, INCOMPATIBLE or
NOT_VERIFIED. Only verified eligible empty/complete states can form a new plan.

A Running/failed import with a fully complete actual catalog, seeds and protected
administrator identity can reconcile without replay. A partial schema, missing
seed, altered credential set, changed metadata or unknown SQL connection stops
with reconciliation required. Even an apparently empty target after an uncertain
started import is not automatically retried. A restart with no durable proof of
its changed invocation cannot be inferred successful on resume. A complete host
produces an empty graph, independently verifies all postconditions and returns
`no_op: true`.

Recovery procedure: stop the stage; retain failed journals and fresh observations;
confirm the exact VM and verified pre-mutation snapshot; have the operator choose
an explicit recovery boundary. Do not DROP/TRUNCATE, reimport into partial state,
delete credential files or restore a snapshot automatically. A reviewed restore
must consider all changes after the snapshot and must be followed by fresh host,
source, database and plan inspection. Snapshot restoration was **NOT_RUN** during
this task; snapshot existence/identity and safe stop behavior were verified.

## Implemented qualification commands

`almadial-db` is a separate laboratory command; the main CLI does not yet run a
complete installation. The following examples assume protected input files and
the actual pinned source archive already available. Authorization JSON must be
prepared from the reviewed emitted plan and actual recorded approval/snapshot;
the command does not manufacture operator authorization.

```bash
go build -trimpath -buildvcs=false -o dist/almadial-db ./cmd/almadial-db
dist/almadial-db inspect --config config.json --requirements requirements.json \
  --budget budget.json --manifest manifests/stack.lock.yaml \
  --source vicidial-r4037.zip > before.json
dist/almadial-db plan --input before.json \
  --manifest manifests/stack.lock.yaml --source vicidial-r4037.zip > plan.json
dist/almadial-db lab-run --input before.json \
  --manifest manifests/stack.lock.yaml --source vicidial-r4037.zip \
  --authorization authorization.json --state-dir /var/lib/almadial/state/s03
```

`lab-resume` accepts the same flags and reconciles the original immutable plan.
Native inspection/execution on the lab requires its authorized privileged
context. The `--budget` file is the accepted ALM-17 `RecordingBudget`; laboratory
thresholds are not production sizing. A saved input with stale host/source state
is rejected before mutation. Protect redirected JSON with an appropriate umask.

## Actual laboratory qualification

The operator explicitly authorized `almadial-alm16-rpm`, host identity hash
`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`.
Fresh snapshot **`almadial-alm16-rpm.alm18-pre-database-20261110`** was verified
before database changes, with parent `alm17-baseline-20261110`. The verified
initial data directory was empty, MariaDB inactive/disabled, and SELinux
Enforcing. All 725 installed RPMs and the ALM-17 foundation were inspected.

The authorized execution created only the owned config, three protected files,
the local application account, `asterisk` database and MariaDB runtime state.
The separately approved reference/failure fixture database was removed after
qualification. Journaled actions established the pinned schema, checked grants
and restarted MariaDB. Native checks confirmed the actual local-only listener,
socket access, default labels, all schema/seed postconditions and restart
persistence. All 33 seed-table row counts after restart and repeated NO_OP
matched the original reference import. The final reproducible binary re-inspected the live host, generated
an empty graph and resumed the completed original journal with NO_OP.

Two implementation defects were caught by postconditions and corrected:

- MariaDB renders disabled log variables as `OFF`; the initial checker expected
  `0`. It rejected the service result. Fresh service inspection reconciled that
  action after correction, without another initialization.
- The first seed checker confused group `ADMIN` with the `---ALL---` access-list
  value. It rejected an actually complete import. Inspection of the pinned seed
  fixed the required key; fresh catalog/seed recognition reconciled the original
  failed import without importing SQL again.

Both failed runs and their journals are preserved separately from successful
qualification evidence. A real isolated missing-column mutation changed the
catalog fingerprint. A bounded DDL timeout left one fixture table, classified
PARTIAL; ALM-14 resume required reconciliation and did not replay it. All fixture
mutations and cleanup had durable intent and bounded execution. The main schema
observations remained unchanged. Native competing-lock testing was rejected.

Final comparisons showed identical RPM inventory, repositories, keys, DNF
history, vendor identities, managed path metadata and SELinux mappings. The
recording filesystem remained XFS on `/dev/sda4`, with **81,219,338,240 bytes**
and **41,268,345 inodes** available at final inspection, above the explicitly
approved lab-only 1 GiB / 10,000 inode reserve. Production capacity remains a
required operator input. No broad RPM provisioning, firewall or SELinux change
was executed.

## Developer verification and evidence

Go 1.27.3: **145 top-level tests, 804 passing test/subtest events** in both normal
and race runs, zero failures. One existing optional M0 artifact fixture test
(`TestM0Artifacts`) skipped without its artifact-directory environment input;
ALM-18 verified the real pinned SQL archive independently on the VM. All
formatting, vet, module tidy-consistency and module verification checks exited 0.
The 131 Python tests, scaffold, candidate-stack and config-generation checks
passed. All 17 CLI smoke cases and 11 executable terminal/nonterminal journeys
passed. Two isolated Linux amd64 builds of each command were byte-identical:

| Command | SHA-256 |
| --- | --- |
| `almadial` | `95e3045bde6c6d960e7ab091e5ff4162db109912c7e5cd4a57f4a4234a4f5b33` |
| `almadial-rpm` | `155117a46878d6af7fbb30ac67d24dc1d820dc875314c5812d6e16bb130d6810` |
| `almadial-system` | `7b0155dca7db626d22f85f87fafbc134698e76267fd5802bea00a407318aad05` |
| `almadial-db` | `527a47938928aa8fc3ea3190269c676b6b40db72019e08f2a653a1b9145ab3e9` |

Protected evidence is under
`~/almadial-lab/evidence/m2-03/20261110T045300-alm18/`, including authorization,
snapshot read-back, original failed and successful journals, source/catalog
identities, native observations, fixture provenance, test outputs and a SHA-256
ledger. No credential contents, protected SQL data or generated binaries are
committed. Hosted CI is NOT_RUN. Independent AGY QA is pending.

Remaining M2 work includes application/source deployment, registration and
runtime credential consumption. This task neither implements S05 nor authorizes
production deployment, a schema migration, automatic restore or telephony use.
