# VICIdial backup operations and recovery runbook

Scope: PR-02 on the authorized test host `dialer01` (`2.29.153.161`, SSH 22222).
This is a native Python/MariaDB/tar backup utility, not an AlmaDial installer stage.
It does not establish an accepted stack, customer RPO/RTO or complete disaster
recovery. See [acceptance evidence](../qa/VICIDIAL_BACKUP_RECOVERY_ACCEPTANCE.md).

## Architecture and operating boundary

The root-only `alma-vicidial-backup.service` runs one Python utility. The matching
systemd timer runs daily at **03:40 America/New_York**, the server's current zone.
This is 08:40 UTC / 16:40 Asia/Manila in EST, and 07:40 UTC / 15:40 Manila in EDT.
`Persistent=true` catches a missed calendar event after timer activation/boot.
A missed run may execute outside the nominal window; the same idle checks apply.
Timer configuration was verified; an actual reboot and calendar-triggered run
have not been observed in this task.

Storage is `/var/lib/alma-vicidial-backup/complete/<backup-id>/`, outside Apache's
document root. Directories are root-owned 0700, payloads 0600, and process umask
0077. Archives include passwords, account privilege data, private keys and audio.
**They are not encrypted at rest.** Root access or loss of the VPS/storage defeats
these local controls. No external destination, encryption-key custody or isolated
restore target is approved. Do not copy data to an improvised destination.

The job acquires a nonblocking flock and checks host identity, required paths,
storage budget, free space and inodes. It refuses capture while Asterisk has active
channels, VICIdial has live agents/auto calls, another non-idle database command is
present, or an InnoDB transaction exists. Checks repeat around database/file capture.
These are observations, not a call-admission lock: a new call can race the checks.
Use an approved idle/maintenance window for manual runs; do not use this provisional
schedule on a busy customer host without reviewing the operating window.

The database uses **338 MyISAM and eight MEMORY tables** in `asterisk` at acceptance.
The installed `mariadb-dump` captures all native eligible databases, including
`asterisk` and the `mysql` privilege tables, with `--lock-all-tables --quick
--routines --events --triggers --hex-blob`. This uses a global read lock for the
dump; `--single-transaction` alone does not protect MyISAM. See the
[MariaDB dump documentation](https://mariadb.com/docs/server/clients-and-utilities/backup-restore-and-import-clients/mariadb-dump).
Root's existing socket authentication is used; no password appears in argv and
no new database account is created. `--no-defaults` avoids unexpected client
option inheritance. The client is killed and reaped on a **30-second deadline**
or interruption, closing its connection and releasing its locks. The 15-minute
unit deadline also kills the whole service control group. Actual observed dump
duration was below one second, not a guarantee for a future larger database.

MyISAM and captured account state are a logical snapshot during the lock, not a
whole-host point-in-time image. MEMORY rows are included but live agent/channel,
hopper/session state is not valid resumed telephony state. Binary logging remains
OFF; there is no point-in-time recovery between daily backups. RPO and RTO remain
unaccepted. The generated `sys`, `information_schema` and `performance_schema`
diagnostics are not required dump payloads; recreate them with matching MariaDB
packages, rather than treating them as missing customer data.

AstDB uses Python's online SQLite backup API; the live SQLite files are excluded
from tar. Configurations, source, custom binaries and recordings are captured with
native tar preserving numeric ownership, ACLs, xattrs and SELinux metadata.
Recording hashes/size/mtime must be identical before and after capture, and their
archived bytes must match. This supports the observed idle recording set; it is
**not** an atomic database/filesystem snapshot or a guarantee for actively written
recordings. No production recording is mixed, moved, converted or deleted.

The job publishes a complete directory only after full verification and fsync.
Its own `.partial-*` workspace is removed on handled failure. Abrupt power loss or
SIGKILL can leave a private partial directory; inspect it manually, never promote
it as complete. A failure preserves all previously complete backups.

## Recovery inventory

[config.json](config.json) is the exact required/optional dataset inventory.
An absent required path fails the job. Optional paths are recorded as included or
absent in every manifest. Review this inventory when adding a custom dependency.

| Class | Included data | Recovery reason |
|---|---|---|
| Mandatory database | Native all-databases SQL, schema/data, routines/events/triggers if present, `mysql` account/privilege definitions; settings and table-engine inventory | Application and credential recovery on a matching isolated MariaDB |
| Mandatory VICIdial | `/etc/astguiclient.conf`, `/usr/share/astguiclient`, `/var/www/html`, SVN checkout under `/usr/src/astguiclient/trunk` | Installed application, local Perl compatibility patches and upstream reference |
| Mandatory telephony | `/etc/asterisk`, `/usr/sbin/asterisk`, `/usr/lib64/asterisk`, `/var/lib/asterisk` plus consistent AstDB copy | Configuration, sounds, manually built runtime, `res_monitor.so` and `chan_sip.so` |
| Manual build dependencies | `/usr/src/asterisk-20.21.0`, `/usr/src/asterisk-perl-0.08`, local Perl libraries | Preserve build selections and non-RPM dependencies while reproducibility is incomplete |
| Mandatory recordings | `monitor`, `monitorDONE`; existing voicemail/dictate/recording/monitorP/monitorS/monitorTRASH roots | Raw and any processed customer audio, without processing it |
| Mandatory web/database | Apache, PHP/PHP-FPM, MariaDB configurations including `91-vicidial-sql-compat.cnf` | Restore compatibility and routing |
| Mandatory portal/TLS | `/etc/alma-portal`, active and legacy portal directories/state, both Python environments, `/etc/letsencrypt`, ACME webroot, TLS state, `/usr/local/sbin`, `/usr/local/libexec` | Separate integration, ACME account/lineage/private keys and corrected renewal hooks |
| Mandatory operations/security | systemd definitions, cron definitions, firewalld/Fail2Ban/SSH configuration, authorized keys, account/UID metadata | Startup, local access and authorization integration; review before activation |
| Host-specific recovery context | Existing network connection profiles, hosts/hostname, timezone, audit/sudo/SELinux/logrotate/repository settings | Selective recovery with explicit IP and ownership review |
| Reproducible components | RPM version inventory and kernel version; upstream source/build trees; current virtual environments also retained conservatively | Reinstall matching packages; binaries/venvs alone do not prove portability |

Known transient/excluded material: raw MariaDB datadir (logical SQL instead), live
AstDB files (online SQLite copy instead), sockets, caches/pycache, live channels,
outgoing call queue, general logs and temporary files. Runtime channels cannot be
recovered as calls. Standard RPM payloads and the OS/bootloader are reinstalled
from trusted matching sources, not a filesystem image. Existing `/root` SQL dumps
and old rollback archives remain untouched; they are historical material, not
the current dataset. An inventory is not proof that rebuilding the OS succeeds.

**Archived portal grants, queued requests, sessions and firewall runtime state
must not be reactivated automatically.** Their capture is for recovery context
and forensic consistency, not permission to resume old grants. AstDB/SIP contacts,
live agents and MEMORY contents also require a deliberate runtime reset on the
isolated target before application startup. No generic destructive reset SQL or
state deletion is executed by this utility.

## Payload and manifest

Each complete directory contains:

| File | Meaning |
|---|---|
| `database.sql` | Complete native logical dump, including sensitive privilege/account data |
| `files.tar` | Uncompressed recovery file archive; includes private keys/configurations/audio |
| `astdb.sqlite3` | Consistent online SQLite copy |
| `inventory.json` | Package/kernel/application versions, DB settings and per-table metadata |
| `manifest.json` | Format `alma-vicidial-backup/v1`, ID, hostname, UTC time, included/absent roots, consistency method/deadline/duration, capacity, recording hashes, required members and archive count/size |
| `SHA256SUMS` | SHA-256 of the five payload files above |
| `VERIFIED.json` | Verification timestamp, payload/member/table/recording counts |

`manifest.created_at` is after capture; the directory ID is allocated before the
dump. Database completion time and status attempt/completion time define the
capture window. Do not interpret a single timestamp as an atomic host snapshot.
`status.json` records latest outcome/attempt, retained last failure and last success
(including archive hash, total size and verification summary). Checksums detect
corruption; they are not signatures against a privileged attacker.

## Daily operation, health and bounded retention

Run these through SSH 22222 as root. Do not print SQL, archive contents, secrets
or recording filenames into tickets or public terminal logs.

```bash
systemctl status alma-vicidial-backup.timer
systemctl show alma-vicidial-backup.service -p Result -p ExecMainStatus
/usr/local/sbin/alma-vicidial-backup health
journalctl -u alma-vicidial-backup.service --since '2 days ago' --no-pager
```

Health exits nonzero for failed/running/missing state, a missing verification
marker, insufficient free inodes or last success older than 30 hours. It reports
last failure, lock state, age and free bytes/inodes. A historical `last_failure`
remains after a subsequent successful run; inspect the current `outcome` too.
The lightweight health command does not rehash payloads; run `verify` for that.
Systemd failure and root-readable local state were exercised. **No external alert
destination or polling service is configured.** An operator must check these;
this does not establish delivered alerting or proactive failure discovery.

For a manual backup, first obtain an idle execution window and inspect Asterisk
channels and VICIdial agents. The utility repeats its own checks:

```bash
asterisk -rx 'core show channels count'
systemctl start alma-vicidial-backup.service
/usr/local/sbin/alma-vicidial-backup health
```

Do not run the failure injection scripts on a populated customer service without
separate approval. `failure_probe.py` temporarily makes only the backup inventory
invalid; it is acceptance tooling, not the recurring job. `deploy.py` is an initial
installation script and deliberately refuses an existing installation. Do not
rerun it as an upgrade mechanism.

Provisional retention is a **non-destructive ceiling**, not a rolling purge:
seven complete backups, 32 GiB total pool, 4 GiB maximum run/individual file,
5 GiB filesystem reserve and 10,000 free inodes. Files must fit preflight bounds;
future recording growth can require a reviewed budget. At the count/budget limit
the job fails visibly and leaves recovery material intact. No archive is deleted.
With two accepted backups already present, five further successful runs reach
the seven-backup cap. **Approve off-host custody and a deletion/rotation policy
before the next run after that cap.** Never remove the final known-good set, old
installation dumps or rollback archives merely to clear an alarm. A customer
retention/RPO policy remains to be approved. Seven daily snapshots is a proposal,
not a customer retention agreement.

## Integrity verification and off-host prerequisite

Select an exact complete directory, not `.partial-*`. On the current source host:

```bash
/usr/local/sbin/alma-vicidial-backup verify /var/lib/alma-vicidial-backup/complete/20261106T210331-1ea99037
```

Verification reads all five payload hashes, dump completion/schema/table presence,
the tar stream and mandatory custom members, recording content hashes, member
count and SQLite integrity. It does not import SQL, extract into live paths,
start services or prove restorability. Verify the SQL import on a separate target.

Minimum off-host setup: operator-approved destination/access constraints, sufficient
capacity/retention, authenticated encrypted transfer, least-privilege storage,
independent key custody if encrypting, and retrieval/integrity evidence. Keep a
complete directory and manifest together. Retrieve a copy in isolation and check
the same SHA-256 values against an independently retained manifest. No off-host
copy was made here. Provider snapshots remain UNKNOWN, not asserted absent.

## Isolated recovery procedure — not yet executed

Only proceed after the operator authorizes a separate disposable restore target
and access window. **Never run these steps over the working VPS.** This runbook
describes the supported sequence; it is not evidence of a successful restore.

1. Record the target identity, isolation boundary, source backup ID, operator,
   start time and expected RPO. Block all outbound live SIP/PSTN/AMI/portal-worker
   access and public ingress before using copied credentials. Keep a provider
   console and explicit emergency SSH path. No routing to customer systems.
2. Provision matching AlmaLinux architecture and **MariaDB 10.11.18** from trusted
   sources. Use `inventory.json` for exact package versions, Perl/Python dependencies
   and build metadata. Validate custom Asterisk modules against the matching binary
   and shared libraries. Do not assume a copied Python venv is relocatable or a
   source directory proves reproducibility. Preserve software provenance/licensing.
3. Place the approved recovered directory in a root-only workspace outside every
   webroot. Retain the original read-only. Run `sha256sum -c SHA256SUMS` from that
   directory, then the verification routine against a work copy using a reviewed
   target-specific config. The source utility is host-guarded: do not disable that
   guard on production. Use its `verify()` function on the isolated path if needed;
   verification never calls `main()` when imported. Record errors and hashes.
4. Stop/disable copied application/cron/portal/telephony automation on the target
   before extraction. Stage the tar into an empty private directory, using
   `tar --acls --xattrs --selinux --numeric-owner -xpf files.tar -C <private-stage>`.
   This is a template: substitute a verified empty directory on the isolated target
   only. Never extract directly over `/`. Review symlinks, permissions, UIDs/GIDs,
   account metadata and sensitive files. Selectively install configurations; do not
   blindly overwrite target SSH keys, `/etc/shadow`, networking, machine identity
   or systemd enablement links. Keep the archive itself confidential.
5. On the isolated matching MariaDB instance only, keep VICIdial/cron/Asterisk and
   event scheduling disabled, and ensure no customer connection can arrive. Use
   a socket-authenticated administrator to import `database.sql` with the native
   `mariadb` client via standard input and a private error log. Require a zero exit
   code; never use `--force`. Include the mysql privilege tables only on the matching
   version. Review privilege activation and socket administration on that target
   using the native MariaDB recovery procedure; any required target-only privilege
   reload/restart is a separately controlled restore step, not a source-host action.
   An example import, **only inside that authorized isolated target's private backup
   directory**, is `umask 077` followed by
   `mariadb --no-defaults --protocol=socket --socket=/var/lib/mysql/mysql.sock --user=root < database.sql 2>restore-errors.private.log`.
   Check its exit code immediately and protect the error log; it can contain SQL
   excerpts. Do not print account hashes or try archived passwords in public logs.
6. Compare schema/table counts against `inventory.json` (338 MyISAM and eight MEMORY
   in `asterisk` at this backup); do not equate approximate metadata row counts with
   an exact SQL row audit. Compare users, campaigns, lead/recording linkage and
   representative exact counts captured for the restore exercise. Confirm SQL mode,
   collation and `91-vicidial-sql-compat.cnf`. Recreate generated diagnostic schemas
   through matching packages. Record import duration, errors and all discrepancies.
7. Selectively restore application/custom code, Asterisk binaries/modules/config,
   Perl libraries and operational definitions from staging. Restore recordings
   byte-for-byte to the intended private paths and rehash against the manifest.
   Do not mix/compress audio or populate recording URLs as part of recovery. Place
   `astdb.sqlite3` only with target Asterisk stopped; review stale contact/device
   state. No running calls, agent sessions or registrations survive recovery.
8. Inventory and invalidate stale VICIdial live/session/MEMORY state and archived
   portal sessions/grants/queues under a reviewed target-only reset plan. Do not
   start copied portal firewall automation until it has a fresh empty authorization
   state and correct trusted source-IP configuration. Persistent configuration is
   restored; archived network entitlement is not automatically reauthorized.
   Initially keep SIP/RTP denied apart from the approved recovery test path.
9. Restore the complete Let's Encrypt lineage including archive/live symlinks,
   ACME account and private-key permissions only into protected target storage.
   Review the existing PR-08 renewal service/hook and verify certificate validity,
   served fingerprint and ACME routing before enabling its timer. A different IP
   requires configuration/NAT/database server-IP review and an appropriate new
   certificate, not reuse of an IP-mismatched cert or disabled verification.
10. Validate Apache/systemd/config syntax offline. After the target reset and network
    review, start dependencies in order: database/time synchronization, restricted
    firewall and fresh portal worker/backend, PHP-FPM/Apache, Asterisk, then VICIdial
    operational services/cron. Consult the actual unit dependencies. Keep external
    telephony blocked until the operator authorizes an acceptance call. Enable TLS
    and backup timers only after target paths/host identity/window have been reviewed.
11. Record separate results for database import, configuration recovery, recording
    byte recovery, application startup, authorized/unauthorized portal behavior,
    native user login and separately approved call/audio/recording test. Measure
    recovery duration and newest available data age; account for data created after
    the backup window. Reboot acceptance is a separate authorized test. These gates
    cannot be inferred from successful archive extraction.

If a restore fails, keep the original backup intact, capture sanitized errors,
discard or roll back only the disposable target through its approved mechanism,
and retry from a clean matching instance. Do not replay SQL over a partly restored
production database. No complete host RTO or application startup acceptance exists
until this procedure is performed and corrected against actual restore evidence.

## Scoped deployment rollback

The original four backup component paths were absent. The root-only deployment
record and rollback script are in:

`/root/alma-vicidial-backup-deploy-20261106T205857Z/`

After any running backup finishes, rollback from the desktop is:

```bash
ssh -p 22222 root@2.29.153.161 /root/alma-vicidial-backup-deploy-20261106T205857Z/rollback.sh
```

It refuses an active job, disables/stops only the new timer, removes only the four
new component files and reloads the systemd manager. It preserves every backup,
status file, pre-existing service, configuration and historical recovery artifact.
It does not restart Asterisk/MariaDB/Apache or change network rules. Script syntax
and a synthetic file-removal rehearsal with stubbed systemctl passed; live rollback
and full restoration were not performed. Reinstall from reviewed task sources if
needed; the initial deploy tool intentionally refuses an existing backup data tree.

## Development checks

```bash
python3 -m unittest discover -s operations/backup -p 'test_*.py' -v
python3 operations/backup/verify_evidence.py
```

The unit tests use synthetic SQL/tar/SQLite fixtures. Evidence verification compares
captured host state; neither command substitutes for an isolated restore. Remote
collectors run through SSH stdin and transfer sanitized metadata only. They must
not be used to export SQL, recordings or certificate private material into Git.
