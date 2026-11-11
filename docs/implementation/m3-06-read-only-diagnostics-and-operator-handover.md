# M3-06 — Read-only diagnostics and operator handover

## Contract and implementation

ALM-28 (`hptrjdzgsafy21edd6nqiigc`) consumes accepted ALM-27 at
`f71c68c898015d20dfeb9385f2c3903d2e576f9a`. Canonical Kaneo scope requires
status/doctor, explicit evidence export, service/listener/manifest/journal/contact/
certificate/job/database/DNS/AVC/recording-space inspection, concise URLs,
bootstrap/recovery pointers, named fault fixtures, and before/after read-only
verification. Independent M3 gate acceptance remains a separate reviewer duty;
this developer handoff does not accept that gate or resolve the pending M2
orchestration scope clarification.

The existing `cmd/almadial` CLI adds Linux commands:

```text
almadial status [--json] [--ca-file FILE] [--manifest FILE] [--journal FILE]
almadial doctor [same options]
almadial evidence export --output /absolute/new-support.tar.gz [same options]
almadial support-bundle --output /absolute/new-support.tar.gz [same options]
```

`internal/diagnostics` uses the accepted bounded execution runner, source-manifest
parser, strict journal decoder, required Asterisk module inventory, and S06 cron/
runtime classification. It does not instantiate a provisioning adapter, execution
session, journal store, recovery engine or mutation lock. Full stage qualification
requires additional pinned artifacts and inputs; these operational probes do not
pretend to replace the accepted S01–S11 postcondition suites.

The fixed probe inventory uses systemd property reads, OS/kernel/clock/SELinux
observations, RPM count, Asterisk show commands, socket-authenticated SELECTs,
localhost lookup, TCP socket inspection, firewalld policy information, exact
FastAGI command/user/unit observations, deferred-unit properties, exact owned cron
comparison, nonwriting file/access checks, trusted unauthenticated HTTP requests,
public certificate expiration and recording-filesystem capacity. Optional explicit
manifest and journal inspection returns identity digests and action counts only.
No newest journal is guessed and no journal success is treated as host health.

Output states are OK/WARN/FAIL/UNKNOWN/NOT_CONFIGURED. Exit 10 indicates a degraded
required check, 0 indicates healthy required checks, 1 indicates output failure,
and 2 indicates invalid command usage. Informational findings carry
`blocking: false`. Export exits 0 when the archive is written, while retaining
health findings inside the report. Doctor prints observations, expectations and
specific inspection guidance without invoking repair. An absent idle synthetic
phone contact and intentionally disabled updater/listener are normal.

## Read-only and support-export boundary

Commands are fixed argv, without shells or secret-bearing credentials, bounded to
four seconds each and a 90-second collection context. Captures are limited to
256 KiB stdout and 4 KiB stderr. Commands never restart/reload a service, execute
installer actions, rotate credentials, change policy, authenticate an application
session, write SQL, or open/copy a recording. HTTP requests check the expected
unauthenticated 401 admin and 426 WebSocket-route responses using the selected CA;
no insecure flag is used. Native permission failures and unavailable tools become
sanitized findings. Root Unix-socket access is required for the accepted database
probe; diagnostics do not read a password to work around denied access.

Export constructs exactly four members from allowlisted projected fields:

| Member | Contents |
|---|---|
| `report.json` | Fixed check IDs/states/guidance; validated versions, counts and public manifest/journal digests |
| `redaction.json` | Included/excluded categories, collection bounds and scope warnings |
| `manifest.json` | Size/SHA-256 for report and redaction manifest |
| `SHA256SUMS` | SHA-256 for report, redaction manifest and member manifest |

Raw stdout/stderr, configuration, logs, credentials, contact URIs, database rows,
caller data, private keys, cookies and recording contents are excluded before
serialization. This is projection, not regex sanitization of arbitrary copied
files. Export accepts no source directory or arbitrary collection list. Both
compressed archive and total member bytes are limited to 1 MiB. Member order and
headers are deterministic; timestamps/live observations in the report vary.

Output is an explicitly requested new absolute mode-0600 file. Directory traversal
uses fd-relative `O_NOFOLLOW`, and creation uses `O_EXCL`; existing destinations,
symlinks and traversal are rejected. No directory is created. Input manifest,
journal and owned cron reads reject symlink components, special files and oversized
input. A failed archive write leaves its newly created partial file for inspection;
it never unlinks a potentially substituted directory entry. Ordinary service
access/audit logging and access times are distinct from managed-state mutation.

## Laboratory qualification

Target: authorized disposable `almadial-alm16-rpm`, machine
`0fffa9a4683f35b696f0b6ee7330734b`, with recorded snapshot
`alm25-pre-firewall-20261111`. No service restart, firewall reload, reboot, call,
certificate replacement, package installation or database mutation was performed.
The qualification binary and explicit archive were staged under VM `/tmp`.

The final Linux amd64 CLI SHA-256 was
`f1f5c77deede25a5693e14ad0d35e8beb35cac8f38f63307dfec3a9a8f4ab9e3`.
Qualification invoked `status --json`, `doctor`, and
`evidence export --output /tmp/alm28-support-final.tar.gz`, each with:

```text
--ca-file /etc/pki/tls/almadial-s08/current/trust.pem
--manifest /tmp/alm19-qualification/stack.lock.yaml
--journal /var/lib/almadial/alm26-noop/journal.json
```

| Observation | Actual result |
|---|---|
| Status / doctor | Exit 10; all checks OK except `tls.expiration` WARN |
| Certificate warning | Accepted lab certificate expires 2026-11-25 00:45:48 UTC, within 30 days; trusted HTTPS/WSS probes pass |
| Manifest / selected journal | Valid CANDIDATE manifest; selected completed NO_OP journal has zero actions; neither rewritten |
| Runtime | Main services active, six FastAGI processes with expected identity/unit, deferred units inactive/disabled, owned cron unchanged |
| Synthetic phone | Configured endpoint present; no active browser contact, correctly informational |
| Export | Exit 0; 3,564 compressed bytes, 13,965 member bytes; mode 0600 |
| Bundle integrity | Exactly four members; manifest sizes/hashes and all SHA256SUMS verified |
| Secret exclusion | Five protected secret-file values checked in memory; absent from archive, along with private-key, SIP authorization, cookie and password markers |

The final archive SHA-256 is
`c7099378be4861dca3665edb27a34455acb71f2077f8652c68ac94050ce5fd1f`.
The archive remains on the VM; it was not uploaded or published.

Before/after comparison was identical for all 27 captured baseline fields:
repository/RPM inventory, database catalog, both complete 107-column legacy phone
records, selected application tables including recording/call metadata, protected
credentials, Asterisk modules/configuration, Apache/PHP and VICIdial configuration,
TLS generations, firewalld files and effective nftables rules, journals, identities,
cron, service states/invocation/start identities, sockets and existing recording
trees. RPM count remains 731, schema 1748, SELinux Enforcing. In particular:

- Catalog SHA-256: `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`.
- Legacy phone digest: `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`.
- RPM digest: `113ac0fae3aabc42859282ccce65a1c882ff34bbc68e0c3ebeb0ff9f2a012a20`.

The initial live pass exposed two diagnostic assumptions: `/etc/os-release` is a
symlink, and the pinned administrator PHP file exceeds the bounded metadata input
size. The collector now reads the known canonical `/usr/lib/os-release` regular
file and uses nonwriting readable/nonempty tests for the PHP entry point. No
application or operating-system files were altered to satisfy the diagnostics.

## Verification and limitations

Focused fixtures cover healthy/deterministic reports, stopped/missing components,
command failure and timeout, missing modules/endpoint, wrong FastAGI count, unsafe
bind/policy (including rich-rule continuation lines), low recording space,
certificate warning, secret-bearing outputs, integrity/limits, safe destinations,
and CLI rendering/exit behavior. Every emitted executable/argv is inspected against
the read-only inventory. Missing manifest/journal input stays explicit.

Repository verification: `go vet ./...`, `go test -json ./...`,
`go test -race -json ./...`, `go mod verify`, `go mod tidy -diff`,
`make fmt-check`, scaffold validation, Python unittest discovery and
`git diff --check` all passed. Go and race each produced **1,442 passing events**;
Python ran **221 tests**. The affected CLI passed **23 executable smoke cases** and
**11 terminal/nonterminal journeys**. Two reproducible builds of affected
`cmd/almadial` matched the SHA-256 above. Unchanged stage command targets were not
rebuilt. Hosted CI and independent QA are NOT_RUN by this implementation worker.

This is a current accepted-profile diagnostic surface, not full application,
external-network or media acceptance. RPM count does not validate every NEVRA;
policy summary does not replace full S09 rules/reachability qualification; owned
cron comparison does not audit foreign schedules. AVC observations are bounded
kernel boot entries, not the complete audit store. No raw logs are collected.
TLS expiration warning combines short remaining life and unavailable certificate
inspection; trusted route checks separately detect current trust/identity failure.
The operator can inspect public certificate dates to distinguish the warning.

Operator URLs, bootstrap cautions, daily checks, recovery pointers and archive
verification are in [the operator handover](../operations/almadial-operator-handover.md).
Protected execution evidence is under
`~/almadial-lab/evidence/m3-06/20261111T204150Z-alm28/`; its SHA256SUMS ledger is
sealed after final repository/governance capture. No credentials, raw bundle or
protected evidence are committed. Changes remain uncommitted for AGY review.
