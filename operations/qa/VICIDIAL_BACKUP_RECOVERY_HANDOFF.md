# PR-02 backup and recovery handoff from PR-08

Date: 7 November 2026 Asia/Manila (6 November UTC). Host: `dialer01`,
`2.29.153.161`, SSH 22222. This handoff records inspection only, except for the
explicitly authorized PR-08 configuration rollback archive. No production database
backup, restore, provider snapshot or recording backup was run.

## Findings and confidence

| Facility | Observation | Classification |
|---|---|---|
| Provider/VPS snapshots | No provider account/API was inspected. Existence, coverage and consistency remain unknown. | UNVERIFIED; do not infer absent |
| Emergency console | Operator previously confirmed recovery-console access. No console recovery was exercised during PR-08. | Operator-reported, untested |
| Existing local dumps | Several installation-era `.sql` files under `/root`, dated 4 November; metadata only inspected. Includes `asterisk-before-pjsip.sql` (1,231,978 bytes). | CONFIRMED files; completeness/integrity/recoverability UNVERIFIED |
| Existing configuration rollback | Root-only portal 443/UI/office/revocation/5060 backup directories and rollback scripts exist. Prior 5060 rollback rehearsal metadata exists. | CONFIRMED scoped recovery material; not complete host/data backup |
| Scheduled backup | Root/asterisk have no user crontab. System cron and all eight visible timers do not register a database/config/recording backup. No backup/restic/borg/duplicity unit found. | CONFIRMED in inspected schedules; external facilities unknown |
| Recording processing | Current `/etc/cron.d/vicidial` includes a mixing entry, unlike the earlier audit snapshot. This is not a backup job; its correctness is outside PR-08. | CONFIRMED scheduling drift, no workflow acceptance claimed |
| Off-host backup | No accepted destination, recent off-host manifest or retrieval evidence established. | UNVERIFIED |
| Isolated restore environment | No authorized restore target identified for this host's current data/configuration. Other project labs are not automatically authorized recovery environments. | UNVERIFIED |
| RPO/RTO and restoration | No accepted objectives or successful full restore record established. | UNVERIFIED |
| `/var/backups` | Path does not exist at this inspection. The earlier audit described it as empty; neither observation proves external backups absent. | CONFIRMED current path state |

Evidence: [recovery-preflight.json](evidence/tls-renewal/recovery-preflight.json),
[original production audit](VICIDIAL_PRODUCTION_READINESS_AUDIT.md), and the scoped
PR-08 archive rehearsal in [deployment.txt](evidence/tls-renewal/deployment.txt).
The earlier audit's DB engine inventory was not repeated here: it reported 338
MyISAM and eight MEMORY tables. Validate that inventory before designing the backup;
do not assume an InnoDB-only `--single-transaction` backup is consistent.

## Scoped TLS recovery provided by PR-08

Root-only directory: `/root/alma-portal-tls-backup-20261106T203248Z` (0700).
`originals.tar` (0600) preserves original TLS hook, service, timer, renewal
configuration and both Apache TLS/ACME virtual hosts with owner/mode/ACL/xattr/SELinux
metadata. `manifest.json` contains original hashes and basic metadata.
`rollback.sh` is 0700. Isolated extraction verified byte identity, UID/GID and modes;
live rollback was not performed. Private keys were not copied. Existing Certbot
archive/live lineage files were not manually replaced or deleted.

This package restores the certificate integration configuration only. It is
**not** a recoverable VICIdial/database/recording set, off-host disaster recovery
copy, verified certificate-account recovery package or proof of PR-02 completion.
The old hook restored by rollback contains known failure masking.

## Recommended dedicated PR-02 assignment

1. Inventory any provider/off-host solution with the operator before duplicating it;
   define ownership, recovery-console procedure, approved encrypted storage, RPO,
   RTO, retention, data confidentiality and the cost/space envelope.
2. Define a consistent backup method for the actual MariaDB engines and operational
   workload. Obtain approval for any lock/window or snapshot. Cover current DB,
   VICIdial/Asterisk/custom source and configuration, dependencies, portal persistent
   state and reconciliation behavior, recordings, certificate/ACME account recovery
   material and secrets through protected storage. Never put these in Git.
3. Implement the separately authorized recurring schedule with success/failure
   state, disk and age checks, protected off-host transfer, integrity manifests,
   retention and delivered alerts. Avoid treating configuration or job exit alone
   as a complete backup acceptance result.
4. Obtain an isolated target and restore authorization. Restore a representative
   complete set there, keeping SIP/PSTN, IP certificate challenges and production
   addresses isolated. Verify users/campaigns, call/recording consistency, protected
   playback, service dependencies, portal grant expiration and firewall reconciliation.
5. Record backup age, manifest integrity, measured recovery duration, missing data,
   actual RPO/RTO achieved and repeatable operator recovery steps. Separately approve
   any service restart/reboot or host disaster-recovery exercise.

Acceptance requires evidence of **backup configured → completed → integrity checked
→ off-host retrieval → isolated restore tested**, with a delivered failure alert.
PR-02 remains open. Provider snapshots and emergency-console availability alone do
not establish application-consistent or recording-complete recovery.
