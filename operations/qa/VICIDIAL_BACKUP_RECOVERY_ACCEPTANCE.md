# PR-02 — VICIdial backup and disaster recovery acceptance

Verdict: **VICIDIAL_BACKUP_RECOVERY_PARTIAL**

Assessment date: 7 November 2026 Asia/Manila; host evidence captured 6 November
2026 20:58–21:14 UTC. Target: `dialer01`, `2.29.153.161`, SSH 22222. This is a
scoped implementation and developer acceptance record, not independent third-party
QA or approval for customer deployment. No push was performed.

## Executive summary

A root-only, systemd-scheduled native backup workflow is installed and two real
database/configuration/recording backups completed and passed full payload
verification. The final set includes the current inventory, manually compiled
Asterisk and modules, local Perl patches, VICIdial, portal/TLS dependencies,
database account recovery data and raw/processed recording trees. The database
capture explicitly uses a global read lock suitable for the observed MyISAM-heavy
database. Failed and overlapping jobs were exercised without destroying the
previous complete set. A later successful run restored healthy current status.

**PR-02 remains open for disaster-recovery acceptance.** The operator answered
“No” when asked for an approved off-host destination and separately authorized
isolated restore environment. No backup archive left the VPS, no SQL was restored
and no service was started from restored configuration. Local archives do not
protect against loss of the VPS/storage. Provider backups remain UNKNOWN. Root-only
permissions were tested; archives are not encrypted at rest. External failure-alert
delivery, accepted customer RPO/RTO and an approved deletion policy remain pending.

No Asterisk, MariaDB, Apache, PHP-FPM, portal, SSH, firewall or Fail2Ban restart was
performed. No live call, registration, grant/revocation exercise, recording
conversion, application/schema edit or package installation was performed. Only
new backup components and their own scoped acceptance state were changed remotely.

## Discovery and source baseline

Sources reviewed: [production audit](VICIDIAL_PRODUCTION_READINESS_AUDIT.md),
[PR-02 handoff](VICIDIAL_BACKUP_RECOVERY_HANDOFF.md),
[TLS acceptance](VICIDIAL_TLS_RENEWAL_ACCEPTANCE.md), repository guidance and
actual installed implementation. Prior snapshot facts were rechecked rather than
treated as live state. Existing unrelated Git changes were preserved.

| Item | Current observation | Evidence category |
|---|---|---|
| Host | AlmaLinux 10.2; x86_64; kernel `6.12.0-211.73.1.el10_2.x86_64` | Direct discovery |
| Telephony | Asterisk 20.21.0, existing manually compiled binary and custom modules | Direct CLI/file discovery |
| Database | MariaDB 10.11.18; local socket `/var/lib/mysql/mysql.sock` | Direct SQL metadata |
| VICIdial engine inventory | `asterisk`: 346 tables, 338 MyISAM and eight MEMORY | Repeated before/after metadata |
| System DB | mysql includes Aria/InnoDB/CSV plus a view; privilege tables preserved | Metadata plus dump-structure validation |
| Authentication | Existing root socket authentication; no new account/password | Account metadata inspection; no hashes exported |
| PIT recovery | Binary logging OFF; no point-in-time recovery demonstrated | Direct variable inspection |
| Existing tools | Native dump, tar, Python/SQLite, hashing/flock/systemd available; no install needed | Tool discovery and actual execution |
| Existing historical material | Installation-era `/root` SQL dumps and portal/TLS rollback packages retained | Metadata inspection; not asserted complete recoverable sets |
| Existing recurring backups | No prior recurring whole application/database/recording backup identified in inspected schedules | Current scheduling inspection; external systems unknown |
| Existing VICIdial cron | Operational jobs exist, including audio job with `--MIX`; unchanged | Configuration inspection/hash comparison |
| Filesystem | ext4 root filesystem; about 140 GB free before capture; ample inodes | `df` and capacity checks |
| Time | Server America/New_York; time synchronized at preflight | Direct inspection; explicit zone in timer |
| Emergency access | Recovery console previously operator-reported; SSH 22222 works | Operator report plus current SSH sessions; console recovery not tested |
| External facilities | No approved off-host destination or isolated target supplied | Explicit operator answer; provider facilities UNKNOWN |

Some discovery commands preceded the saved baseline collector. A final read-only
discovery snapshot persists OS/tool/time/socket-authentication metadata and confirms
healthy backup status. Database background work can resume after capture: that
snapshot observed one other database command, not an active backup/read lock.
Persisted evidence also contains DB/service/filesystem/firewall/HTTP/TLS state and
the exact deployed source hashes. No password, SQL body, private key or recording
audio is included in the local evidence directory.

## Architecture and consistency strategy

The installed utility uses standard Python facilities, native `mariadb-dump` and
tar. The exact inventory is [config.json](../backup/config.json); operational and
recovery instructions are in [the runbook](../backup/README.md).

1. Acquire a non-overlapping flock; require root and the expected hostname.
2. Reject missing required paths, inadequate capacity/inodes or a reached archive
   budget. Require zero Asterisk active channels, zero VICIdial live agents/auto
   calls, no other non-idle DB commands and no InnoDB transaction. Recheck idle
   state around capture.
3. Create a 0700 workspace; capture package/table/version/settings inventory.
4. Execute native all-databases logical capture with `--lock-all-tables --quick
   --routines --events --triggers --hex-blob`. Enforce a 30-second client deadline;
   kill/reap the client process group on timeout/interruption to release its
   connection/locks. No `--single-transaction` consistency assumption is used.
5. Capture AstDB with SQLite's online backup API. Archive configuration/custom
   binaries/source/dependencies and recording trees with ownership/ACL/xattr/
   SELinux metadata. Do not archive live AstDB files directly.
6. Require stable recording file hashes, size and mtime across capture. Verify the
   archived recording contents against those hashes. This is not an atomic host
   snapshot or support for capturing actively changing recordings.
7. Validate payload checksums, dump footer and table definitions, mysql privilege
   recovery, tar contents/required members and SQLite integrity. Flush files and
   directory entries; atomically rename to a complete backup directory.
8. Store current outcome, last success, retained last failure and integrity summary
   in root-readable status. Handled failure removes only its own workspace.

The MyISAM global-read-lock approach is documented by
[MariaDB's native dump documentation](https://mariadb.com/docs/server/clients-and-utilities/backup-restore-and-import-clients/mariadb-dump).
The observed full dump runtime bounds the lock-hold duration; exact lock acquisition
and release timestamps were not separately instrumented. No live contention/load
test was run. Idle checks do not atomically prevent new call admission. Scheduled
or manual capture on a busier host requires an approved window and renewed sizing.

The all-databases dump captures `asterisk` and mysql account/privilege definitions.
Generated diagnostic schemas (`sys`, information/performance_schema) are recreated
with matching packages. MEMORY rows are preserved in SQL but must not be treated
as resumable live channels/agents/sessions. No database or recording reset was
executed. Restoring stale portal grants, queue entries or sessions is explicitly
prohibited without a target-only invalidation plan.

## Dataset coverage

| Dataset | Capture and verification | Boundary |
|---|---|---|
| Application database/account privileges | Complete native SQL; 376 table definitions across captured databases; mysql.global_priv required | SQL import and privilege activation NOT_RUN |
| VICIdial and compatibility patches | Installed application/scripts, source checkout, local Perl libraries; required four compatibility scripts verified in tar | No upstream files modified; future package/SVN updates need review |
| Asterisk and AstDB | Config, binary/modules/sounds, full existing source/build trees and Perl dependency; required res_monitor/chan_sip members; SQLite copy integrity | No portable rebuild or service startup from backup demonstrated |
| Portal/TLS/web/database | Both environments, integration code/state, TLS lineage/accounts/private keys, corrected hooks, Apache/PHP/MariaDB configs | Sensitive archive; fresh grant/session state required on restore |
| Boot/scheduling/security | Existing systemd/cron, firewalld/nft integration scripts, SSH/Fail2Ban, UID/account metadata and host settings | Selective restore only; no automatic stale permissions/networking activation |
| Recordings | monitor/monitorDONE plus existing ancillary recording roots; six dataset files verified; four primary raw WAV files unchanged | Six is a file count, not a claim of six audio recordings; publication remains separate PR-01 |
| Reproducible/optional/transient | RPM inventory, upstream source and current venvs retained; exclusions/optional absences in manifest | No whole-disk image, OS reinstall or full host rebuild acceptance |

The final backup contains 79 required root/critical member assertions and 21,368
tar members. A capture stores five payloads plus SHA256SUMS and a verification
report. The exact selected roots and exclusions are described in the runbook and
config; raw database datadir, transient sockets/caches/live calls, outgoing call
queue and general logs are not restored datasets. Existing historical dumps and
rollback archives are left untouched rather than silently replaced.

## Exact installed changes

| Repository source | Installed path | Owner/mode |
|---|---|---|
| `operations/backup/backup.py` | `/usr/local/sbin/alma-vicidial-backup` | root:root 0700 |
| `operations/backup/config.json` | `/etc/alma-vicidial-backup.json` | root:root 0600 |
| `operations/backup/alma-vicidial-backup.service` | `/etc/systemd/system/alma-vicidial-backup.service` | root:root 0644 |
| `operations/backup/alma-vicidial-backup.timer` | `/etc/systemd/system/alma-vicidial-backup.timer` | root:root 0644 |

Also created: the timer enablement symlink under `timers.target.wants`, private
backup/state directories under `/var/lib/alma-vicidial-backup`, and scoped rollback
material under `/root/alma-vicidial-backup-deploy-20261106T205857Z`.
`systemctl daemon-reload` and enable/start of the new timer were used. Only the
new backup service was started for acceptance. None of the four target component
files existed before the assignment. The temporary failure test modified and
restored only the new backup config; the final inventory was then extended to
cover PHP-FPM main/host settings before the second accepted capture.

Other new local task files: `deploy.py`, `test_backup.py`, `collect_evidence.py`,
`failure_probe.py`, `final_probe.py`, `verify_evidence.py`, the runbook, this report
and sanitized evidence under `operations/qa/evidence/backup-recovery/`. No backup
archive, SQL dump, certificate key, credential value or audio is committed.

Final installed SHA-256 values, compared against current repository sources:

```text
b38462d24d9636659561f46d4c88c617690b68e4314ea67dd00377a4a4b27a81  backup.py
523dfe94a2006641f8e87a1e234099452c7cdd28aa2f2c6a19eb7d951044b3ab  config.json
2f61a59980b49bd5b08b3a07844c491fbac8dc195188e545afe68fa17a76b256  alma-vicidial-backup.service
d6b2aa807caaf60572cea1f2a99f7e87a04c8b899938aa08a3a6d7a1b83d775d  alma-vicidial-backup.timer
```

## Real backup and negative-test evidence

Both accepted sets are on the VPS under `/var/lib/alma-vicidial-backup/complete/`.
They were not downloaded.

| ID | Completion UTC | Total payload bytes | Dump runtime / lock-hold upper bound | Full verification |
|---|---|---:|---:|---|
| `20261106T210108-9b19fad6` | 2026-11-06 21:01:17 | 1,358,927,507 | 0.8166 seconds | 5 payloads, 21,306 members, 376 table definitions, 6 recording dataset files |
| `20261106T210331-1ea99037` | 2026-11-06 21:03:39 | 1,363,750,901 | 0.7662 seconds | 5 payloads, 21,368 members, 376 table definitions, 6 recording dataset files |

The second run took approximately 9.79 seconds end-to-end and contains the final
expanded inventory. Its `files.tar` SHA-256 is
`b867bcf61a63cf9a498f25ac48620392a042d329f79988f80f8e878b638e54b6`.
Its SHA256SUMS file SHA-256 is
`5c980a1ae4e6fb19cdf1eb08839871800c1c20d0d393bd7d56fa67bcc97d3a94`.
This provides integrity identity, not an independent signature or restore result.

An initial run at 20:59 UTC was conservatively rejected by the SQL structure
validator. Investigation found native `CREATE TABLE IF NOT EXISTS` system-table
syntax and generated sys-schema omission; the validator was corrected and a
regression test added. No incomplete backup was published. This was a resolved
validator defect, not an accepted capture or a claimed restore test.

Actual post-success negative tests:

- Holding the job lock caused a second service invocation to fail with
  `execution_lock_busy` and left the existing job/status intact.
- Adding a deliberately missing required dataset to the new backup configuration
  caused service exit 1, `ActiveState=failed`, `ExecMainStatus=1`, a retained
  `required_dataset_missing` reason and health exit 1. This preflight failed before
  any database dump/lock. Original config was restored exactly.
- The prior complete-set checksum identity remained unchanged and full verification
  passed after the injected failure. No handled-failure workspace remained.
- The next real successful backup restored health exit 0 and retained last-failure
  history for diagnosis.
- Unprivileged `nobody` could not read database.sql (test exit 1); directories and
  payloads were verified root-owned 0700/0600.
- Unit syntax validation passed. Scoped rollback was rehearsed on synthetic files
  with stubbed systemctl; actual installed hashes and archives were unaffected.

The journal reported roughly 1.4 GB peak service-accounted memory on the final
run, including filesystem cache accounting. This small idle-server observation is
not customer load acceptance; future dataset growth requires budget/impact review.

## Scheduling, retention and failure reporting

The timer is enabled, active and persistent. The next recorded trigger is
**7 November 2026 03:40 EST / 08:40 UTC / 16:40 Asia/Manila**. It runs as a system
service independently of SSH. Summer EDT shifts the UTC/Manila equivalent by one
hour. No actual reboot or calendar-triggered run was performed in this assignment.

Provisional local limits: seven complete backups, 32 GiB pool, 4 GiB per run/file,
5 GiB free-space reserve and at least 10,000 free inodes. The safe implemented
retention action is **refusal**, not deletion. No historical recovery material is
deleted, and the final known-good backup cannot be purged by this utility. Two
accepted sets exist; five more successful runs reach the cap. Approve retention,
off-host custody and any deletion before the following run. This ceiling is safe
bounded storage, not an indefinitely unattended rotating archive service or an
approved customer retention/RPO policy.

The root-readable health command reports last success age, last failure/reason,
outcome, verification marker, current lock state, free bytes and free inodes.
Failures remain nonzero in systemd/journal. Age greater than 30 hours fails health.
The final check found healthy status, no held lock, approximately 146.56 GB free
and 9.78 million free inodes. The health command does not rehash all archives.
**No approved external notification destination exists; no delivered alert is
claimed.** Operators must inspect health/systemd until a separate integration is
approved. Local failure observability and delivered alerts are different gates.

## Regression results

`verification-results.json` contains reproducible comparisons:

| Check | Result | Evidence and limits |
|---|---|---|
| Asterisk, MariaDB, Apache, PHP-FPM, portal, firewalld, Fail2Ban, SSH, crond | PASS | All nine active; exact master PID/start/NRestarts unchanged |
| Application/telephony/TLS/cron source | PASS | Twenty pre-existing critical file hashes identical; no upstream/schema/config edits |
| HTTP/TLS | PASS | Trusted certificate SHA unchanged; root 303, portal 200, both unauthorized browser entry paths 303 and AJAX requests 403 before/after |
| Firewalld | PASS | Runtime and permanent public-zone state identical |
| SIP/RTP gate and grants | PASS | Rule order/ports/priorities/timeouts and effective IP members identical; only packet counters and remaining timeout lifetime normalized |
| Listeners/SSH 22222 | PASS | Listener inventory unchanged; repeated SSH commands succeeded |
| Portal authorization worker/TLS timer | PASS | Worker success; TLS timer remains enabled, active and persistent |
| Raw recordings | PASS | Four primary files, 2,147,696 bytes; exact aggregate path/size/content hash unchanged |
| Calls/agents | PASS at capture | Repeated idle checks and final zero active channels/calls; no calls initiated |
| Fresh authenticated agent login/call/reboot | NOT_RUN | No credentials requested, calls made or unrelated service/reboot acceptance repeated; prior functional verdicts retained, not re-certified |

No successful SIP registration, audible media or full customer workflow is inferred
from the listener or firewall checks. No active authorization was revoked.

## Acceptance matrix

| ID | Criterion | Result | Evidence |
|---|---|---|---|
| BAK-01 | Recovery inventory is complete and documented | PASS | Current scoped inventory/config, required-path checks, runbook classification/exclusions; full-host reproducibility still unaccepted |
| BAK-02 | Database backup method is MyISAM-consistent | PASS | Actual native global-read-lock dumps during idle preflight; 30-second client deadline; measured runtime upper bounds 0.8166/0.7662 seconds |
| BAK-03 | Required configuration and custom patches are preserved | PASS | Final 21,368-member archive, 79 mandatory roots/critical members, full payload hashing; no restore assertion |
| BAK-04 | Raw and processed recordings are covered | PASS | Relevant roots included; six stable dataset files checked against archived hashes; processed tree covered even where empty |
| BAK-05 | Backups have appropriate confidentiality | PASS | Root 0700/0600, umask 0077, outside webroot, unprivileged SQL read denied; root/storage compromise and at-rest encryption remain limitations |
| BAK-06 | Automatic scheduling enabled and persistent | PASS | Explicit-zone timer active/enabled/Persistent=yes; unit validation; actual reboot/calendar run not exercised |
| BAK-07 | A real backup completes successfully | PASS | Two native SQL/filesystem captures; second is final inventory |
| BAK-08 | Manifest and integrity verification pass | PASS | Re-read both complete sets; checksums/table presence/tar stream/custom members/recordings/SQLite all pass; not a SQL restore |
| BAK-09 | Failed backups detectable and preserve prior valid sets | PASS | Actual overlap/missing-path failures, failed unit and health, prior archive unchanged and reverified, subsequent success recovered health |
| BAK-10 | Retention documented and safely implemented | PASS | Non-destructive seven-set/size/free-space ceilings, no purge; operator rotation policy required before cap, not a customer policy acceptance |
| BAK-11 | Approved off-host recovery copy exists and verified | NOT_RUN | Operator supplied no approved destination; no archive transferred; provider systems unknown |
| BAK-12 | Database restoration succeeds in isolation | NOT_RUN | No separately authorized restore environment; no SQL import performed |
| BAK-13 | Configuration/recording restoration succeeds in isolation | NOT_RUN | Archive read/hash validation only; no representative production extraction/restoration |
| BAK-14 | Recovery runbook actionable and tested to supported scope | PARTIAL | Detailed staged procedure and integrity/rollback checks; actual data restore/application startup/RTO untested |
| BAK-15 | Existing application/telephony/portal/TLS preserved | PASS | Bounded regression above; same process identities/files/network policy/recording bytes; no new call/reboot/customer acceptance claimed |

## Evidence index and repeatability

All paths below are relative to this report. These are sanitized metadata, not
production data archives:

- [before.json](evidence/backup-recovery/before.json): pre-deployment health,
  source checksums, recording aggregate, database engines, firewall, services,
  TLS and HTTP baseline.
- [discovery.json](evidence/backup-recovery/discovery.json): final read-only OS,
  tool, time-sync, socket-authentication, historical-dump metadata and health check.
- [deployment.json](evidence/backup-recovery/deployment.json): initial installation
  and scoped rollback path. This is the initial revision; final installed hashes
  are in after/final-probe/verification-results.
- [local-tests.txt](evidence/backup-recovery/local-tests.txt): 17 synthetic unit
  tests, including corrupt/incomplete payload rejection, idle/timeout/error
  handling, required custom files, non-destructive ceiling and cleanup.
- [failure-probe.json](evidence/backup-recovery/failure-probe.json): actual negative
  unit/health results and preserved/reverified prior set.
- [final-probe.json](evidence/backup-recovery/final-probe.json): two full archive
  verifications, unit/calendar validation, private access check, scoped synthetic
  rollback rehearsal, module/channel status and bounded service journal.
- [after.json](evidence/backup-recovery/after.json): final host health, archive
  summary, permissions, schedule, firewall and existing service/config state.
- [verification-results.json](evidence/backup-recovery/verification-results.json):
  automated baseline comparisons and final installed/source SHA-256 values.
- [SHA256SUMS](evidence/backup-recovery/SHA256SUMS): local sanitized evidence hashes.

Repeat local tests with `python3 -m unittest discover -s operations/backup -p
'test_*.py' -v`, and baseline comparisons with
`python3 operations/backup/verify_evidence.py`. These do not contact the server.
Live verification command and private paths are in the runbook. Do not rerun
deployment/failure tooling automatically on a customer host.

## Remaining dependencies and required operator acceptance

1. Approve off-host storage/custody, retention and encrypted transfer; copy a
   complete set and verify independent retrieval/integrity. Provider snapshot
   existence alone would not prove application consistency or retrieval.
2. Authorize a separate matching isolated restore target with no live telephony
   route. Execute the runbook; verify SQL import, users/campaigns, recordings,
   custom modules, configuration and fresh portal state separately. Record errors,
   exact counts, durations and data-loss window. No substitute restore on this VPS.
3. Approve customer RPO/RTO and steady-state rotation before local ceilings stop
   capture. Reassess the 30-second dump budget and idle-window admission strategy
   for actual workload/recording growth.
4. Provide an approved notification destination or an explicitly owned polling
   procedure. No one is guaranteed to receive a local failed-unit condition.
5. Test target service startup and recovery/reboot separately after restoration.
   Never reactivate archived portal grants, sessions, queue entries or SIP contact
   state. Different-IP recovery requires configuration and certificate review.

Recording finalization, SELinux enforcement, credential/file hardening, password
hashing, recording access control, general monitoring and installer reproducibility
remain separate assignments. Backup coverage preserves these components without
changing their current behavior. PR-08 TLS renewal was preserved, not replaced.

## Rollback

Wait for the backup service to finish, then use the existing SSH recovery path:

```bash
ssh -p 22222 root@2.29.153.161 /root/alma-vicidial-backup-deploy-20261106T205857Z/rollback.sh
```

It refuses an active job; disables/stops only the new timer; removes the four new
component files; reloads the systemd manager; and preserves all complete backups,
status and historical recovery material. Original absence of those four paths is
recorded in the root-only deployment package. Syntax validation and a synthetic
removal rehearsal passed. A live rollback, database restore and host recovery were
not executed. Do not remove the retained backups or restore older TLS hooks as
part of this rollback.

## Final conclusion

The independently safe local portion is implemented and accepted to the measured
scope: actual MyISAM-consistent capture, protected configuration/audio coverage,
recurring schedule, integrity validation and detectable non-destructive failures.
Off-host protection and isolated restoration are not available for acceptance.
The installation therefore has a defensible **local backup baseline**, while
complete disaster recovery and closure of PR-02 remain pending.

**VICIDIAL_BACKUP_RECOVERY_PARTIAL**
