# ALM-33 / M4-05 — Recording backup and documented restoration

Developer qualification: ready for independent QA; acceptance remains with AGY.
Starting HEAD: `559b03a99264db76f8b0811ca53ef91e9608de95`.

## Canonical scope and authority

Kaneo ALM-33 (`e3y45lk9xkrmim6ujhemrst3`) requires documented manual backup and
restoration of installer-owned configuration, protected credentials, database and
synthetic recordings, with consistency, permission, correlation and missing or
corrupt backup checks. It does not promise transactional call-state rollback or
automatic top-level backup/restore commands. ALM-28 is Accepted; the M3 gate and
ALM-27/29/30 accepted reports provide the existing functional foundation.

The operator explicitly authorized clean-d as the read-only source and isolated
restoration on clean-e in a separate directory and MariaDB schema, preserving
both hosts' active configuration/databases. Export of exactly six synthetic WAVs
to protected local evidence was separately authorized after automatic approval
review rejected that export without specific authorization. No customer data or
publication was involved. No ALM-32 work was reopened.

## Inventory and topology

| Item | Observed identity / result |
|---|---|
| Source | `almadial-alm29-clean-d`, machine `4c7b492940923838a0700adba65ed3eb` |
| Target | `almadial-alm29-clean-e`, machine `ebf868d3a0853b4294cd85e589a12f5c` |
| OS/image | AlmaLinux 10.2; image SHA-256 `cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca` |
| Recovery records | Each has `alm29-pristine`; these pre-install snapshots were inspected, never restored or represented as post-M3 backups |
| Installed foundation | 731 RPMs each, schema 1748, SELinux Enforcing; active accepted application/services |
| Backup | 48 regular files, 1,914,880-byte protected tar archive |
| Restored directory | `/var/tmp/20261112T232400Z-alm33/restored` on clean-e |
| Restored DB | New empty `alm33_20261112232400` on clean-e; original `asterisk` DB untouched |

The 48-file inventory includes six accepted ALM-29 MixMonitor recordings,
installer-owned Asterisk/application/service/firewall configuration, protected
credential and TLS generation files, the application SQL dump, correlation TSV
and recovery JSON. The archive and private manifest are evidence, not Git assets.
The TLS generation pointer was recorded and recreated only inside the isolated
tree. No configuration was activated, no restored credential was substituted
into the running target and no MySQL system grant table was replayed.

## Recording and database relationship

`call_log.uniqueid` → `call_log.channel` → MixMonitor filename
`alm-s11-<uniqueid>[-rx|-tx].wav` under `/var/spool/asterisk/monitor/`.

| Unique ID | Channel | Start → end, source DB time | Call length | Restored WAV duration |
|---|---|---|---:|---:|
| `1794450528.0` | `PJSIP/alm_s10_9061-00000000` | 2026-11-12 10:28:48 → 10:29:02 | 14 s | 14.24 s, each of 3 files |
| `1794450529.3` | `PJSIP/alm_s10_9061-00000001` | 2026-11-12 10:28:49 → 10:29:02 | 13 s | 12.68 s, each of 3 files |

Both are previously qualified synthetic calls; no calls or database rows were
created to manufacture correlation. `recording_log` publication/mixing is not
claimed. The raw mixed, receive and transmit files were all restored.

## Implementation and consistency

`scripts/recording_backup.py` is a small manual archive helper, not an installer
stage or new backup framework. Explicit inventories, deterministic ordering,
regular-file-only members, exclusive output creation and complete pre-extraction
verification bound its behavior. It records SHA-256, size, mode, UID/GID and mtime;
source changes during collection are rejected. It accepts no SQL/service actions.
Limits: 10,000 files, 1 GiB payload. New private directories are 0700, archives and
staged SQL 0600; source file modes/owners/mtimes are verified on restoration.

The application DB has 337 MyISAM and eight MEMORY tables, with no routines,
triggers or events. The drill used the approved quiescent dataset: zero active
channels, deferred runtime jobs and no operator writes. Deterministic lock-free
logical dumps before and after collection were byte-identical. The restored
schema's complete logical dump also matched, SHA-256
`f649fe248019cd849d12c1fd9964d8bf63c5b16e53e7423211209470d1c3e034`.
This is not an online consistency guarantee for busy MyISAM deployments.

Archive SHA-256:
`9971a9448c35eaa01f84a99eeb99aaac1c6a76bb4700d008fe15c2f50fea7ff5`.
The manual workflow, prerequisites, stop conditions and commands are in
[the operator guide](../operations/recording-backup-and-restore.md).

## Actual qualification

| Check | Result | Evidence |
|---|---|---|
| Backup integrity | PASS | All 48 required members verified, protected archive/manifest |
| Actual isolated restoration | PASS | 48 contents, sizes, modes, numeric ownership and mtimes checked after writing |
| DB/schema/call correlation | PASS | Full logical dump equality, exact two restored call-log rows, schema 1748 |
| Audio integrity/playback decode | PASS | Six actual restored WAVs match original hashes; ffprobe PCM s16le, mono, 8 kHz; ffmpeg complete decode exit 0 |
| Meaningful samples | PASS | Normalized PCM RMS 0.1172–0.2292, unchanged frame counts/durations; no new human listening acceptance claimed |
| Native corrupt/missing/symlink input | PASS | All three rejected on clean-e before destination creation |
| Repeat restore | PASS | Existing destination rejected; no merging or overwrite |
| Focused fixtures | PASS | Eight tests cover round trip, corruption, missing member, unsafe path, symlink source/destination/member, duplicate and existing output |
| Existing application health | PASS | Both status/doctor exit 10 only for expected short-lived certificate; trusted admin route 401 and WSS route 426 checked by accepted diagnostics |
| Preservation | PASS | Both hosts' selected managed files/journals, protected DB/schema digest, package inventory, nft policy and service invocation identities unchanged |

The final preservation inventory covers 107 files on each host. Preservation
compares contents, file mode and numeric
ownership, not access times or expected audit/health request logs. Original
recordings are hashed in both snapshots. No source recording, credential, TLS
file, firewall policy or canonical installer journal was rewritten; no package
transaction, service restart or provisioning replay occurred. The only target
writes were under the task directory and the new isolated schema. Accepted M3 VM
and historical evidence were not touched.

The first attempted archive transfer embedded a long base64 literal in a Python
stdin line; Multipass truncated it, causing a SyntaxError before restoration.
The verified archive was then transferred as binary stdin. Its hash and all
members were checked at the destination. An initial health assertion expected
HTTP 200 incorrectly; actual unauthenticated admin 401 is the accepted security
boundary. Accepted diagnostics independently verified HTTPS 401 and WSS 426;
no application change was made to alter those responses.

## Regression and limitations

Go vet, normal/race tests, module verify/tidy-diff, formatting, scaffold and diff
checks passed. Normal and race runs each had 1,464 test/subtest passes and two
existing native opt-in skips. Python: 240 tests, including eight focused new
archive tests. Six additional helper CLI smoke cases passed (help, create,
verify, restore, repeat rejection and invalid-archive rejection). No Go code
changed; rebuilding unaffected binaries was unnecessary. Hosted CI: NOT_RUN.

This drill restores selected configuration/credentials as data, a complete
application schema and six real recordings into isolated destinations. It does
not claim a bootable whole-server recovery, operational replacement of target
configuration, restoration of MySQL accounts/grants, directory ACL/xattr/SELinux
labels, transactional call state, or a fresh WebRTC campaign. Activation in live
paths requires separate ownership, label, user/grant and credential reconciliation.
Quiescence and trusted private workspaces are prerequisites. Partial output from
I/O/ownership failure remains private for diagnosis; retries require a fresh path.

## Evidence and handoff

Protected evidence:
`/home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/`.
Includes host/snapshot identities, private archive/manifest, restoration, actual
audio validation, native rejection checks, before/after preservation, health and
regression results. SHA-256 ledger and Kaneo read-back are preserved there.
Task-owned changes remain uncommitted for AGY; no push, release, acceptance or
downstream implementation is authorized.
