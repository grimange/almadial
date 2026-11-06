# PR-01 / PRD-05 recording finalization acceptance

Date: 2026-11-07 Asia/Manila. Host: `dialer01`, `2.29.153.161`, SSH TCP
22222. Implementation and verification worker: Codex. This is implementation
acceptance evidence, not an independent reviewer certification.

**Verdict: VICIDIAL_RECORDING_FINALIZATION_PARTIAL.**

## Executive summary

The known successful test call now has a valid private finalized WAV and a native
HTTPS recording location. The unmodified installed VICIdial VD-only processor
finalized recording 2 from staged copies, leaving all four original WAV files for
recordings 1 and 2 unchanged. The operator listened to the finalized file through
private SSH access and confirmed **both recognizable voices**.

A small guarded systemd integration replaces the single mismatched `--MIX` cron
entry. It admits completed conference-listener recordings, preserves originals,
checks native results and reports failures. Real historical recovery, repeat
recovery, timer operation and isolated error handling were exercised. A newly
captured call entering the automatic queue has **not** been tested.

**Secure browser playback is not accepted or enabled.** Installed native report
permission checks do not authorize the subsequent static media request. No
configuration-only individual media authorization solution was demonstrated using
the installed components. Publishing behind the portal's shared-IP boundary
would not satisfy REC-10. Apache explicitly denies the recording URL and storage
directory; there is no Alias. User Stats and Lead Details playback therefore
remain unavailable. This follows the assignment's mandatory security decision
gate rather than weakening the required policy.

## Scope, baseline and evidence

Reviewed the existing product audit, PRD acceptance plan, production audit, audio
acceptance and backup acceptance reports, then inspected the live recording
scripts, cron, native PHP paths, storage and metadata. Existing reports are
historical context; current checks are saved separately under
[`evidence/recording-finalization/`](evidence/recording-finalization/README.md).

Task discovery began from Git `665d4d650e07b8379842c2095ae90730fc117232`.
An unrelated M0 acceptance commit advanced HEAD to `3f33130` during this work;
its files and other untracked work were preserved. Only this assignment's
recording integration, report and evidence are committed. The final operator
handoff records the task commit ID; this document does not embed its own hash.

All remote execution used noninteractive `ssh -T -p 22222`; no root shell history
file, shell startup file or SSH configuration was changed. No packages were
installed, calls initiated, phones reconfigured, grants revoked or full backups
started. No raw recording, secret, database dump or private key was copied into
this repository. Narrow source excerpts contain variable names, not credentials.

Before evidence: 2026-11-06 22:18:09 UTC / November 7 06:18:09 Manila.
Deployment: 22:21:19 UTC. After evidence: 22:22:44 UTC. Later timer and operator
listening evidence is recorded separately; it does not alter the earlier snapshot.

## Pre-change recording state

* Extension 8309 in the active dialplan uses `Answer()`, native
  `Monitor(wav,${CALLERID(name)})`, `Wait(3600)`, then `Hangup()`. It writes to
  `/var/spool/asterisk/monitor/`. Capture configuration was not changed.
* The active three-minute mixer cron used `AST_CRON_audio_1_move_mix.pl --MIX`,
  which selects `monitor/MIX/`. The source-directory mismatch was confirmed.
* Recording IDs 1 and 2 had NULL locations and no corresponding finalized WAV.
* SoX, LAME, Perl and flock were already installed. No competing recording
  processing job or active Asterisk call was found during deployment preflight.
* Existing backup health was successful: two complete sets, seven-set ceiling;
  last verified set `20261106T210331-1ea99037`. Its manifest/integrity evidence
  covered database, configuration and recording data. No new backup was required.
  Off-host and isolated restore gaps remain separate PR-02 limitations.

Known successful call association, from bounded read-only queries:

| Field | Value |
|---|---|
| Call identifier | `1793990617.45` |
| Call date | `2026-11-07 02:43:41` in the database |
| Historical campaign value | `TEST01` |
| Lead / recording | `8` / `2` |
| Synthetic number | `7000000001` |
| Call length / disposition | 52 seconds / `SALE` |
| Recording start / end epoch | `1793990617` / `1793990673` |
| Recording metadata length | 56 seconds |
| Base filename | `20261107-024337_7000000001` |

The recorder input is the conference listener's already mixed audio, not a
participant-specific track. Its companion output is a 44-byte header. Empty
output alone does not establish missing customer audio. Historical two-way call
acceptance and the new finalized-file listening confirmation are separate evidence.

## Native processing decision and safeguards

Selected **Option B**, the installed `AST_CRON_audio_1_move_VDonly.pl`, unchanged.
For this verified conference mode it moves the populated input to `*-all.wav`,
removes the staged empty output and updates `recording_log.location` using its
own intended database code. With `--HTTPS` it constructs the expected HTTPS URL.
The general mixer was not selected: it does not solve the source mismatch by
itself and mixing/transcoding adds no value to this already mixed input.
Both processors are never scheduled against the same recordings.

The native processor's SHA-256 is
`a9837929fcda93d4ca238eb0f06ac5d2fc3198c3029c2e6132f29fe3091c25c0`.
The wrapper refuses changed source until reviewed. Perl `-s` initializes existing
upstream `CLImonitor`, `PATHmonitor`, `CLIDONEmonitor` and `PATHDONEmonitor`
variables for a private staging directory and the final directory. This uses
existing native controls without editing Perl or `/etc/astguiclient.conf`.

The wrapper is necessary because native shell/SQL failures are not reliably
propagated. It performs these bounded checks:

1. Root-only policy/state, nonblocking shared execution lock and required tools.
2. Closed recording metadata, unique safe filename, 60-second end/mtime margin,
   all-process writable-FD checks, unchanged file identity and sufficient capacity.
3. Mono 8 kHz, 16-bit PCM with genuine input samples and an empty output stream;
   stereo modes or a nonempty output fail for review rather than losing audio.
4. Copy originals into private staging; recheck metadata, file identity, writable
   descriptors and hashes before invoking the single native processor.
5. Verify retained originals, final byte identity, format/duration, SoX decode and
   exact native database association. Publish a private receipt only on success.

The native size-stability delay is 15 seconds. Originals are never processing
inputs and remain retained even on failure. Existing identical final files are
safe to retry; conflicting files, receipts or locations fail. The wrapper itself
executes SELECT queries only; all location writes are from the native script.

**Limits:** native file operations and SQL are not a transaction. A failed run can
leave a private output or URL without a receipt. The wrapper reports failure and
preserves raw files; it does not perform custom SQL repair. A malformed pending
record stops the batch for review. Up to 20 oldest unlocated records are inspected
per run; concurrent volume and alternative recording formats are not accepted.
No compression, raw deletion or retention policy is introduced.

## Scheduling and historical recovery

`alma-vicidial-recording.timer` is enabled. It starts 90 seconds after boot and
three minutes after the previous service completion, with 10-second accuracy.
It is a monotonic timer, not a calendar timer with `Persistent=true`.
The oneshot uses root, `UMask=0077`, a 12-minute timeout and control-group kill
semantics. Reboot operation is configured but a full host reboot was not tested.

Automatic eligibility starts at epoch `1794003679`, the deployment boundary.
Older IDs 1 and 2 are held from automatic discovery. After the native fixture
passed and the private recovery effects were explained, recording 2 was explicitly
selected with `--recording-id 2`. Recording 1 remains untouched and NULL.

Real native recovery returned `finalized_private` at epoch `1794003728`.
Repeat recovery returned `already_finalized` at `1794003760`. Timer-driven empty
queue execution and a restart of only the new recording timer succeeded.
These checks do not substitute for a new operator call entering the automatic queue.

## Final WAV and database evidence

Final path:
`/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav`

| Inspection | Result |
|---|---|
| Size | 892,204 bytes |
| Format | PCM WAV, mono, 8,000 Hz, 16-bit |
| Frames / duration | 446,080 / 55.76 seconds |
| Decode | Existing SoX full-stream inspection succeeded |
| SHA-256 | `f726a1d5615d50767c8a0f4794f68a544322ffa70037d1ef1a7bb3667aa4be59` |
| Input comparison | Byte-for-byte identical to the original input WAV |
| Ownership / mode | root / `0600` |
| Original raw pairs | All four hashes and permissions unchanged |
| Listening | Operator replied “Both voices confirmed” for this finalized path |

Recording 2 now has this native location:
`https://2.29.153.161/RECORDINGS/20261107-024337_7000000001-all.wav`.
It is a correct association to an existing finalized file, but intentionally
returns 403. Recording 1 and the sampled successful call record are unchanged.
The 55.76-second file agrees with 56-second recording metadata and reasonably
surrounds the 52-second call. No audio was downloaded or transcribed by the worker.

## Recording authorization decision gate

Direct installed-source findings are saved in `native-authorization.json`,
`media-source-discovery.json` and `native-download-analysis.json`:

* Existing users have `access_recordings=0`; `log_recording_access=0`.
  No permission settings were changed merely to make a link appear.
* User Stats can use stored locations directly when recording logging is off.
  Lead Details can embed the stored media location directly.
* `recording_log_redirect.php` authenticates credentials, checks recording access
  flags, records access and redirects. That authenticates the redirect request,
  not the subsequent media-byte request. It is not a protected media streamer.
* Its failed-login branch can reflect the submitted password. This was established
  from source, not by submitting a real or synthetic password to the endpoint.
* The installed `recording_lookup*.php` files are `removed` stubs.
* Targeted native streaming-source discovery found custom-report HTML download
  and Lead Details GDPR ZIP export. The latter fetches each stored recording URL
  with cURL before creating a ZIP; it depends on media being retrievable already
  and does not implement the requested protected normal playback workflow.
  That state-changing export was not executed.
* No recording Alias or document-root recording symlink was found. An installed
  MySQL Apache DBD authentication driver was not found. No new authentication
  database, service, module or credential copy was introduced.

These findings establish the limitation of the **inspected installed paths**;
they are not a claim that every possible upstream release or integration lacks
media authorization. A Shared Office grant provides network reachability for
several people; it cannot distinguish their recording privileges.

Decision: keep media private. Apache denies `/RECORDINGS/` and the final directory,
sets `Options -Indexes -FollowSymLinks`, and allows no `.htaccess` override there.
There is **no media Alias**. Files are root-readable. Thus no user, including an
otherwise authorized supervisor, has accepted browser playback yet.

The smallest compatible follow-up is to qualify an upstream-supported retrieval
path that authorizes every media request against existing VICIdial permissions,
or obtain a separate scope for a reviewed standard web-server integration with
that identity policy. Decide recording-reader permissions and logging explicitly.
A shared Apache password or IP-only publication would not establish the required
existing-user permission boundary. No such workaround is implemented here.

## Exact deployment changes

| Live path | Change |
|---|---|
| `/etc/cron.d/vicidial` | Comment only the conflicting three-minute `--MIX` job; all other bytes preserved |
| `/etc/httpd/conf.d/02-alma-vicidial-ssl.conf` | Explicit recording directory/URL deny; no Alias or other route changes |
| `/usr/local/sbin/alma-vicidial-recording` | New guarded native processor wrapper, root `0700` |
| `/etc/alma-vicidial-recording.json` | New private deployment boundary/hold policy, root `0600` |
| `/etc/systemd/system/alma-vicidial-recording.service` | New oneshot, `0644` |
| `/etc/systemd/system/alma-vicidial-recording.timer` | New enabled timer, `0644` |
| `/var/lib/alma-vicidial-recording/` | New root `0700` runtime lock, status and private recording 2 receipt |
| `/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav` | New private finalized file; originals retained |
| `asterisk.recording_log`, recording 2 | Native script populates HTTPS location; no custom SQL write |

Before/after hashes, file metadata, policy and rollback identity are recorded in
`deployment.json`. Locally changed source/test/tooling files are exactly the
`operations/recording/` directory; this report and the dedicated sanitized evidence
directory are also new. No upstream source or database schema was modified.

`httpd -t` passed before one graceful Apache reload. The master PID and start
timestamp remained unchanged. No Asterisk, MariaDB, PHP-FPM, portal, firewall,
SSH or cron daemon restart/reload was performed. Only the new recording units
were enabled/started and the recording timer subsequently restarted for its test.

## Tests and regressions

| Evidence category | Executed result |
|---|---|
| Local unit tests | 24 tests PASS; active/recent files, linked/truncated files, collisions, duplicate names, stale metadata, duration, failures, receipts and repeat safety |
| Installed native fixture | Synthetic PCM + header-only output, private DBI test double; real installed Perl source, byte identity and HTTPS SQL construction pass; no live DB writes |
| Real historical finalization | Native file creation and location update PASS for recording 2; repeat PASS |
| Isolated installed failure probe | Missing tools, native hash mismatch and processing failure return nonzero/failed state; overlap refusal and health recovery pass; mocked actions/private state |
| Timer | Actual empty runs and recording-timer restart PASS; new-call automatic handoff and host reboot NOT RUN |
| Apache | Syntax OK; existing global ServerName warning remains unrelated |
| Portal | GET `/` 303 to `/portal/`; GET `/portal/` 200 |
| Unauthorized navigation | GET both native entry points 303 to the public root |
| AJAX/resource requests | Both native entries with AJAX/JSON headers return 403 |
| Recording media | HEAD directory, synthetic name and known finalized name return 403; HTTP recording path also 403; no audio bytes requested |
| External desktop denial | A separate desktop-origin HEAD check of the recording directory, synthetic name and finalized name returns 403 with trusted TLS; `desktop-denial.json` |
| System preservation | Asterisk, MariaDB, httpd, PHP-FPM, portal, firewalld, Fail2Ban, sshd and crond active, same PIDs/start state |
| Telephony boundary | UDP 5060/5061 and SSH TCP 22222 listeners present; original SIP/PJSIP/RTP/dialplan hashes unchanged |
| Enforcement | Firewalld runtime/permanent hashes and normalized nftables SIP gate policy unchanged |
| TLS / backup | Existing automation hashes unchanged; timers remain enabled/active; backup health succeeds and complete-set count remains two |

The earlier baseline collector used HEAD for navigation (including an unsupported
portal HEAD response). After evidence uses GET for browser navigation and HEAD
only for recordings. These different method results are not a portal regression.
An initial desktop unit run hit permission errors enumerating unrelated users'
`/proc` descriptors. The portable unit fixture now enumerates only its own process
and exercises actual writable/read-only descriptors there; the production root
wrapper still scans every process. Both the initial result and final passing run
are retained, separately from the live wrapper's successful root execution.
No new agent login, native admin login, phone registration or call is claimed.
No accepted same-IP authorized-versus-unauthorized playback test exists.
The final SSH recheck again returned healthy recording status, an enabled/active
timer and Apache Syntax OK; see `final-health.json`.

## PRD-05 acceptance matrix

| ID | Acceptance criterion | Result | Evidence and limit |
|---|---|---|---|
| REC-01 | Native recording capture remains functional | PARTIAL | Historical accepted capture; active 8309 and raw bytes unchanged; no fresh capture test |
| REC-02 | Recording automatically enters the correct processing path | PARTIAL | Source mismatch corrected, enabled timer/empty cycles pass; fresh-call automatic handoff pending |
| REC-03 | Finalized recording exists and is valid | PASS | Actual 892,204-byte / 55.76-second WAV, hash identity and SoX decode |
| REC-04 | Native workflow updates recording location | PASS | Real unmodified processor populated recording 2 HTTPS URL |
| REC-05 | Published HTTPS location resolves correctly | NOT_RUN | URL exists in DB but publication deliberately disabled; HEAD 403 |
| REC-06 | Lead Details displays usable recording playback | NOT_RUN | Individual media boundary unresolved; no playback accepted |
| REC-07 | User Stats displays usable recording playback | NOT_RUN | Same security dependency |
| REC-08 | Operator confirms both voices in finalized recording | PASS | Operator explicitly confirmed both voices in the finalized file via private listening |
| REC-09 | Unauthorized external access is denied | PASS | Explicit Apache deny/no Alias/private file plus external-source HEAD 403; no content fetched |
| REC-10 | Unauthorized same-IP user cannot bypass required recording permissions | PARTIAL | All media denied; no accepted individual authorized-user allow/unauthorized-user deny workflow |
| REC-11 | Processing failures preserve original audio | PASS | Failure fixtures preserve originals; all real raw originals unchanged; not production fault injection |
| REC-12 | Repeated processing and restarts do not corrupt recordings | PARTIAL | Real repeat and recording-timer restart pass; full host reboot/new-call test pending |
| REC-13 | Existing portal and telephony functionality is preserved | PASS | Bounded HTTP, file-hash, service/PID, listener and policy comparison; no new call/login acceptance |
| REC-14 | Rollback is documented and verified | PASS | Scoped archive byte/metadata extraction rehearsal and rollback shell syntax pass; live rollback not executed |

## Rollback and recovery

Root-only scoped package:
`/root/alma-recording-pr01-20261106T222119Z`.
Original cron and Apache bytes/metadata are in `originals.tar`; `manifest.json`
records hashes, owner and modes. Its rollback script passed `bash -n`. Extraction
into a private temporary directory reproduced original bytes and metadata.

After checking for later operator configuration edits and letting any recording
job finish, use:

```bash
ssh -T -p 22222 root@2.29.153.161 \
  '/root/alma-recording-pr01-20261106T222119Z/rollback.sh'
```

The script disables the new timer, refuses a running service/manual lock, restores
cron and Apache configuration, validates Apache, reloads it gracefully, removes
the new wrapper/policy/units and reloads systemd definitions. It deliberately
retains all raw/final audio, status/receipts and the native database location.
It does not execute SQL undo or delete recordings. The original mismatched
`--MIX` schedule is restored; it is a rollback, not a working recording pipeline.
No original recording Alias existed, so rollback does not publish final storage.

Existing full backups cover the wrapper, systemd definitions and recording trees.
The new dedicated policy and receipt directory are **not** covered by the current
backup inventory. Retain the scoped recovery package, source and deployment
policy evidence; add those paths in a separately approved backup-inventory update.
Backup schedule/retention was not changed by this task.

## Remaining acceptance and prioritized follow-up

1. **Blocking browser workflow:** qualify and separately approve individual
   media-byte authorization using the existing VICIdial identity/permissions.
   Do not enable IP-only publication. Address the native password-reflection
   branch through an approved upstream path before using that endpoint.
2. After the boundary exists, enable only the intended HTTPS final-file path,
   verify logging/permissions and same-IP allow/deny including direct media,
   then accept both User Stats and Lead Details playback.
3. Have the operator place one separately authorized synthetic call through
   normal VICIdial. Confirm new raw capture, automatic finalization, native
   location, playback and both voices. The worker did not initiate a call.
4. Add the new policy/state paths to the next approved backup inventory review;
   verify reboot and concurrent recording behavior under their own test window.

The operator runbook is [`../recording/README.md`](../recording/README.md).
Successful private finalization and operator listening do not close the protected
browser playback dependency. The complete PRD-05 workflow remains partial.
