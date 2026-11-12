# Recording backup and isolated restoration

This manual procedure preserves completed synthetic recordings, selected
installer-owned configuration and credentials, and the VICIdial application
schema. It is a recording/data recovery drill, not a whole-server image restore,
transactional call rollback, or an automatic `almadial backup/restore` command.
Use a separately authorized disposable destination. Never extract into `/`,
replace a live configuration, or import over an existing application database.

## Consistency and access

Run as an authorized local root operator in a private maintenance workspace
(`umask 077`, directories 0700, archives and SQL dumps 0600). Backups contain
passwords, private keys and call audio: do not upload them, attach them to tickets,
commit them, or leave them in shared `/tmp` directories. The helper does not
encrypt or authenticate an archive. Use approved encrypted storage/transport and
verify its SHA-256 against a separately trusted ledger before restoration.

Check `asterisk -rx 'core show channels count'` and the recording lifecycle.
Select only completed recordings with finalized `call_log.end_time`, no active
channels/writers, and stable file sizes. Absence of channels alone does not prove
that all writers are idle. The qualified drill used the accepted synthetic calls,
disabled application cron/updater/listener, no operator changes, and equal logical
DB dumps before and after collection. It did not stop or restart any service.

The accepted schema contains MyISAM and MEMORY tables. `--single-transaction`
alone does **not** make this database consistent. The following lock-free command
is qualified only for this verified quiescent dataset. On a busy system, stop and
arrange an explicitly authorized consistency boundary; this guide does not
implement online backup, database locking, or downtime automation.

## Prepare the backup

Use the reviewed `scripts/recording_backup.py` from the identified AlmaDial
checkout. It takes an explicit JSON inventory; it does not discover recordings,
select credentials, operate MariaDB, or alter services.

```bash
umask 077
WORK=$(mktemp -d /var/tmp/almadial-backup.XXXXXXXX)
chmod 700 "$WORK"
# Local root Unix-socket authentication; no password in argv or environment.
mariadb-dump --skip-comments --skip-dump-date --skip-lock-tables \
  --order-by-primary --hex-blob asterisk > "$WORK/asterisk.sql"
# Check the exit status before proceeding. Keep a failed/partial dump quarantined.
```

In a reviewed private `inventory.json`, list exact relative paths under `/`.
Include the dump's path without its leading slash. The qualified inventory was:

- Six completed files under `/var/spool/asterisk/monitor/`, using the call IDs
  `1794450528.0` and `1794450529.3`: mixed, `-rx`, and `-tx` WAVs for each call.
- Regular files under `/etc/asterisk` and `/etc/almadial/secrets`.
- Actual TLS generation files under `/etc/pki/tls/almadial-s08`; record the
  relative `current` symlink target separately in recovery metadata.
- `/etc/astguiclient.conf`, both deployed `dbconnect_mysqli.php` files, the
  installer-owned firewalld policy, and existing AlmaDial Apache/PHP, service
  and cron files selected by explicit inventory review.
- The logical SQL dump, a non-secret call-correlation TSV and recovery JSON.

Do not collect entire configuration trees indiscriminately. Inventory only
verified installer-owned regular files. The helper rejects source symlinks and
symlink parents; select their actual generation files and record any necessary
relative pointer separately. It does not copy arbitrary link targets.

Record call correlation with a bounded read-only query; adapt only the reviewed
synthetic IDs, not customer data:

```sql
SELECT uniqueid,channel,start_time,end_time,length_in_sec
FROM asterisk.call_log
WHERE uniqueid IN ('1794450528.0','1794450529.3') ORDER BY uniqueid;
```

Preserve this output privately. `uniqueid` matches the MixMonitor basename;
`channel` identifies the PJSIP channel. These qualified files have real `call_log`
rows, not fabricated `recording_log` rows. Recording duration can differ slightly
from the integer call duration; verify the exact original WAV frame count/rate.

```bash
python3 scripts/recording_backup.py create --root / \
  --inventory "$WORK/inventory.json" --output "$WORK/backup.tar"
python3 scripts/recording_backup.py verify "$WORK/backup.tar"
sha256sum "$WORK/backup.tar" > "$WORK/backup.sha256"
```

Repeat the same dump into a second private file after collection and compare it
with `cmp`; recheck channels and file stability. Stop if any input changed. Do
not label an inconsistent backup successful. Preserve a protected inventory with
source host/machine ID, package/manifest identity, tool revision, file list,
consistency observations, call IDs and timestamp. The helper manifest records
file SHA-256, relative path, size, numeric UID/GID, mode and nanosecond mtime.

## Restore into an isolated directory and schema

Transfer the verified backup using the approved protected transport. Verify its
trusted SHA-256 and run `verify` again at the destination. The destination parent
must already exist and be private, with no symlink components.

```bash
python3 scripts/recording_backup.py restore /private/drill/backup.tar \
  --destination /private/drill/restored
```

`restored` must not exist. All members are checked before the destination is
created. Only inventoried regular files are written, exclusively. Numeric file
ownership, modes, contents and mtimes are restored and checked; new parent
directories are deliberately 0700. This is not restoration of original directory
ACLs, SELinux labels, capabilities or arbitrary xattrs. Do not activate these
files in operational paths without a separate ownership/label/credential review.
Recreate a reviewed TLS `current` pointer only inside this isolated tree, after
verifying its target is the backed-up relative generation directory.

For the database, use a **new, uniquely named empty schema**. Check that the name
is absent first; a conflict is a stop condition. Use local socket authentication.
Review the trusted dump before import: the qualified dump contains no `CREATE
DATABASE`, `USE`, original-schema-qualified objects, routines, triggers or events.
If these occur in another backup, stop for an explicit restore plan. Never use
`--databases` or a dump that selects the original database in this drill.

```bash
# Example only: choose and verify a new absent alphanumeric/underscore name.
mariadb -e 'CREATE DATABASE `almadial_restore_drill`;'
mariadb almadial_restore_drill < /private/drill/restored/ACTUAL/DUMP/PATH/asterisk.sql
# Check each command's status. Do not continue following a failed import.
mariadb-dump --skip-comments --skip-dump-date --skip-lock-tables \
  --order-by-primary --hex-blob almadial_restore_drill > /private/drill/restored.sql
cmp /private/drill/restored/ACTUAL/DUMP/PATH/asterisk.sql /private/drill/restored.sql
```

The actual path is recorded in `manifest.json`; do not guess it. Compare schema
version, call rows, legacy phone identities and the complete logical dump. No
application is pointed at the restored schema. MySQL system users/grants are
**not** imported. Matching source credential files does not authorize rotating
the target's credentials or granting the isolated schema to application users.

## Validate and handle failures

For every selected recording, compare source/manifest/restored SHA-256, byte
count, mode and numeric UID/GID; preserve relative paths and mtime. Use `ffprobe`
to check PCM format, channels, sample rate and duration, and `ffmpeg -v error -i
FILE -f null -` to decode the complete restored audio. Check meaningful non-silent
samples, not just a nonempty header. Audio inspection/export needs authorization.
The qualified drill used six synthetic tone files, not customer calls.

Run `almadial status --json` and `almadial doctor` against both existing hosts.
Verify application health, schema, TLS, firewall and service invocation identities
remain unchanged. Exit 10 for an expiring laboratory certificate is expected.
Administrator HTTP 401 and non-upgraded WSS HTTP 426 are intentional boundaries.
These checks do not claim the isolated recovered database/config is a newly
activated application or that a fresh SIP/media campaign was run.

| Condition | Required response |
|---|---|
| Valid complete archive | Verify all members before restoring |
| Missing member / checksum mismatch | Exit 1, no restore destination created |
| Unsafe path / symlink / duplicate member | Exit 1, no extraction |
| Existing output archive or restore directory | Reject; never overwrite/merge |
| Existing schema | Stop; do not import or drop it |
| Source changed / SQL import error / disk or metadata error | Stop; retain private partial output for diagnosis; choose a new destination after correction |

The helper returns JSON `PASS`/exit 0 or `REJECT`/exit 1. Reasons contain only
exception classes, not file contents. It is bounded to 10,000 files and 1 GiB of
payload, for trusted private operator workspaces. It is not a hostile concurrent
filesystem sandbox or an atomic multi-file/database transaction. After an I/O
failure, partial output may remain; never merge a retry into it. Keep the verified
backup and isolated restored schema/directory private for independent QA.
