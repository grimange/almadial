# AlmaDial operator handover

This guide covers the accepted single-server laboratory stack. Diagnostics do not
provision, repair, restart, reload, authenticate to VICIdial, or place calls. Run
as an authorized local root operator so process, socket and database checks have
access. The installer is not a monitoring daemon or a production-readiness audit.

## Daily inspection

```bash
sudo almadial status --ca-file /etc/pki/tls/almadial-s08/current/trust.pem
sudo almadial doctor --ca-file /etc/pki/tls/almadial-s08/current/trust.pem
sudo almadial status --json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem
```

`status` prints one line per check. `doctor` adds the expected condition and local
inspection guidance. JSON includes check IDs, states, observations, expectations,
guidance and whether the finding affects the exit code. Exit codes are 0 for
healthy required checks, 10 for degraded required checks, 1 for output failure,
and 2 for usage errors. Informational checks can report WARN, UNKNOWN or
NOT_CONFIGURED without making the command fail; consult their `blocking` field.

Checks cover services, OS/kernel/clock, SELinux, bounded kernel AVC observations,
RPM count, Asterisk/version/modules, synthetic PJSIP configuration and contacts,
MariaDB/version/schema, localhost resolution, accepted loopback TCP listeners,
managed firewall summary, FastAGI ownership/count, deferred services, owned cron,
application-file availability, trusted TLS responses, expiration, and recording
space/access. Missing or denied command access produces a finding, not repair.

The current accepted profile has 731 RPMs, schema 1748, Asterisk 22.12.0, one
FastAGI parent plus five workers, disabled updater/listener, and no active owned
application cron jobs. The synthetic `alm_s10_9061` contact may be absent after the
browser session ends. This is normal. Registration does not prove audio.

Supply optional `--manifest /absolute/reviewed/stack.lock.yaml` and
`--journal /absolute/known/state-directory/journal.json` to inspect those exact
records. No latest journal is guessed. These input paths must have no symlink
components. Without them the corresponding checks are NOT_CONFIGURED. The shipped
manifest remains CANDIDATE; its presence is not production authorization.

## Application access and bootstrap

On the VM, the administrator URL is `https://localhost/vicidial/admin.php`,
Timeclock is `https://localhost/agc/timeclock.php`, the agent page is
`https://localhost/agc/vicidial.php`, and the self-hosted phone is
`https://localhost/almadial-phone/`. Asterisk WSS uses
`wss://localhost:8089/ws`. All application listeners remain loopback-only.

For a remote browser use an authorized SSH tunnel to the VM's loopback HTTPS/WSS
ports, with a browser-trusted approved CA and matching `localhost` certificate
identity. An example mapping is local 8443 to VM 443 and local 8089 to VM 8089;
the browser HTTPS URL then uses port 8443. Account for the browser's actual network
namespace. Do not open firewall ports or bypass certificate checks for access.

Use the already provisioned secured administrator and protected S10 phone
credential reference `/etc/almadial/secrets/phone-s10` through the approved local
credential workflow. Never paste credentials into command arguments, support
bundles or tracker comments. Do not rerun VICIdial's global first-run credential
reset or replace legacy phones `callin` and `gs102`. For a genuinely missing
bootstrap, stop and use the reviewed S07/S10 stage procedure with separate
mutation authorization; diagnostics never bootstrap accounts.

## Troubleshooting

| Finding | First local inspection |
|---|---|
| Service down / denied probe | `systemctl status UNIT`; `journalctl -u UNIT --since '-15 min' --no-pager` |
| Web application unavailable | `httpd -t`, Apache/PHP-FPM logs, accepted S05/S07 deployment identity |
| Database unavailable / wrong schema | MariaDB unit and root Unix-socket access; preserve catalog evidence before recovery |
| PJSIP registration expected but absent | `asterisk -rx 'pjsip show endpoint alm_s10_9061'`; `asterisk -rx 'pjsip show contacts'`; browser HTTPS trust and WSS URL |
| WebRTC signaling or media | Separate SIP registration from ICE/DTLS/media; preserve bounded sanitized observations and consult ALM-27 evidence before new call testing |
| TLS warning / handshake failure | `openssl x509 -in /etc/pki/tls/almadial-s08/current/fullchain.pem -noout -dates`; intended CA and hostname |
| Firewall mismatch | `firewall-cmd --info-policy=almadial-s09`; permanent equivalent; `nft list ruleset` |
| Recording access / capacity | `df -h /var/spool/asterisk/monitor`; `namei -l` and SELinux labels; preserve existing recordings |
| AVC warning | `journalctl -k -b`; authorized local `ausearch` inspection; never disable SELinux |

These are inspection pointers, not instructions to run an automatic repair. Raw
local output can contain private data; do not attach it without review. The
firewall check summarizes the owned policy, not every foreign rule or actual
external reachability. The AVC query covers at most 50 matching kernel entries
from the current boot, not all audit-file records. Asterisk endpoint details,
SDP, caller data and recording contents are excluded from exported output.

## Recovery and escalation

Stage state and receipts are under `/var/lib/almadial`; execution journals are
`journal.json` in the explicit state directory selected for that stage. Preserve
the journal, plan/config/manifest identities and fresh observations before any
resume. Use the relevant accepted stage's inspect/plan and guarded recovery
workflow; there is no unified top-level install/resume path. Diagnostics never
open an execution session, acquire the mutation lock, reconcile, or replay an
action. A prior SUCCEEDED journal entry alone does not prove current host health.

Do not blindly retry unfinished DNF transactions, partial SQL bootstrap, unknown
file ownership, or conflicting runtime supervisors. Preserve evidence and
escalate to the responsible implementation/operations owner. Never reimport a
partial schema, rotate established credentials, or rerun earlier installation
stages to resolve a diagnostic warning. Certificate replacement and firewall
activation each require their existing authorized recovery boundary.

## Explicit support collection

```bash
sudo almadial evidence export --output /absolute/existing/private-directory/support.tar.gz \
  --ca-file /etc/pki/tls/almadial-s08/current/trust.pem
# Equivalent alias: almadial support-bundle --output ...
```

The destination must be a new absolute file in an existing directory, without
symlink components. It is created exclusively with mode 0600. No other output
file or directory is created by the command. A failed write can leave that newly
created partial archive; inspect it locally and choose a new destination on retry.
An export exits 0 when written even if `report.json` contains health failures.

The archive contains exactly `report.json`, `redaction.json`, `manifest.json` and
`SHA256SUMS`. It contains allowlisted observations, not raw logs/configuration,
passwords, contact URIs, database rows, private keys, cookies or recordings. Each
probe is bounded to four seconds; the collection budget is 90 seconds; stdout is
bounded to 256 KiB and stderr to 4 KiB per probe. Both total member contents and
the compressed archive are limited to 1 MiB. Archive member order and metadata
are stable; the report timestamp and live observations naturally vary.

Inspect in a private scratch directory before authorized sharing:

```bash
tar -tzf /absolute/existing/private-directory/support.tar.gz
# In a newly created private review directory:
tar -xzf /absolute/existing/private-directory/support.tar.gz
sha256sum -c SHA256SUMS
```

Review the report and redaction manifest, and verify manifest member sizes and
hashes if using automation. SHA256SUMS covers the report, redaction manifest and
member manifest; it does not hash itself. Checksums establish integrity, not the
identity or trustworthiness of a sender. Keep the bundle private; the tool never
uploads or publishes it. Normal probe access may append service/audit logs or
update filesystem access times; managed application state remains unchanged.

## Recording and data restoration

For the bounded manual recording/configuration/database drill, use
[recording backup and isolated restoration](recording-backup-and-restore.md).
It restores only into a new private directory and separate schema; it does not
replace live files or promise whole-server or transactional call recovery.
