# Native recording finalization and protected playback

This is a bounded integration for `dialer01`, not a VICIdial fork or an AlmaDial
installer stage. It uses the installed, hash-pinned
`AST_CRON_audio_1_move_VDonly.pl` unchanged. It is suitable only for the current
8309 conference listener: the input WAV contains the conference mix and the output
WAV has no samples. Nonempty output, stereo modes and other codecs require review;
they are rejected rather than silently dropping a participant.

## Security boundary

**PR-01C protected browser playback is deployed and operator accepted.** The
existing IP gate and individual native VICIdial credentials must both succeed.
There is no recording Alias, directory listing, portal-session requirement or new
authentication service. Storage directories remain denied to Apache. Only exact
native-shaped finalized WAV URLs reach the existing Python backend.

The native stored `/RECORDINGS/<name>-all.wav` URL returns a fixed same-origin
307 to `/vicidial/recordings/<name>-all.wav`. The browser reuses its native
`CONTACT-CENTER-ADMIN` Basic credentials; every request independently verifies
the current native password, account status and recording permissions. Authorized
User Stats and Lead Details playback, no extra prompt, seeking, unauthenticated
same-IP denial and unentitled same-IP denial were confirmed by the operator.

Only active, explicitly entitled level 8/9 global recording supervisors are
supported: `access_recordings >= 1`, View Reports and unrestricted native campaign,
report and administrative-group visibility. Level alone is insufficient. User
6666 alone was approved and enabled through native Administration. The installed
UI hides Access Recordings unless System Settings → Log Recording Access is 1;
the operator temporarily enabled that setting, saved user 6666's permission, then
restored it to 0. Do not repeat that toggle without approval. Current media
integration requires Log Recording Access = 0 and writes its own sanitized access
journal. Native 2FA/IP-list policies, when enabled, cause fail-closed 503 responses
until their actual authentication boundary is supported; never disable them to
make playback work.

Native `recording_log_redirect.php` is not used. Its failed-login password
reflection remains a separate upstream concern. Portal UI requests still strip
native Authorization headers, and playback does not accept portal cookies as
individual identity.

Verified final files alone receive root:alma-portal 0640 access. The root-only
finalizer receipt remains private; a root-owned, group-readable projection maps
each recording ID to its verified inode. Raw files are never enrolled or served.
Inspection found the preexisting raw spool/WAV permissions were 0755/0644, owned
by asterisk; this task did not change them. Raw audio is therefore not root-only
against local accounts, although it remains unpublished over HTTPS. Address local
recording confidentiality separately.

## Pipeline and layout

1. Existing Asterisk `Monitor()` on extension 8309 writes the raw pair into
   `/var/spool/asterisk/monitor/`. Recording capture configuration is unchanged.
2. The recording timer starts a root oneshot 90 seconds after boot and three
   minutes after its previous run completes. It replaces only the conflicting
   `--MIX` cron line; unrelated VICIdial cron jobs remain in place.
3. The wrapper selects up to 20 unlocated records per run, beginning at the
   deployment timestamp in `/etc/alma-vicidial-recording.json`. Older recordings
   remain on hold unless explicitly selected for recovery.
4. Closed database metadata, a 60-second end/mtime margin, all-process writable-FD
   checks, file identity and byte checks reject active/changing recordings.
   A nonblocking file lock prevents overlap, including manual recovery.
5. The raw pair is copied into a root-only temporary staging directory. Perl's
   `-s` initializes existing upstream path-override variables; no source edit or
   persistent astguiclient path change is made. The native script uses `--HTTPS`,
   checks stable staged sizes for 15 seconds, moves its input to `*-all.wav` in
   `/var/spool/asterisk/monitorDONE/`, and updates the native recording location.
6. The wrapper verifies original bytes, final WAV structure, duration, byte
   identity, SoX decoding and the exact native database update. Only then does it
   write a root-only receipt in `/var/lib/alma-vicidial-recording/<id>.json`.
   Originals remain untouched; temporary copies are removed. No compression is
   scheduled and no raw-audio retention/deletion policy is introduced.
7. A service `ExecStartPost` runs `/usr/local/sbin/alma-vicidial-recording-enroll`.
   It validates the existing root receipt, native closed-record association, WAV
   format/duration and SHA-256 before granting group read to that final inode.
   It writes `/var/lib/alma-recording-playback/<id>.json` (root:alma-portal 0640,
   directory 0750) with the fingerprint and original permissions for rollback.
   It shares the finalizer lock, is idempotent, and creates no audio copy.

The unmodified native script does not reliably propagate shell/SQL failures. The
wrapper therefore checks results and reports a failed stage with a nonzero exit
code. **File creation and native SQL are not one atomic transaction.** A failure
can leave a private final file or a native URL without a receipt. Originals are
retained, unreceipted media stays ineligible, and an operator must review the failure.
No custom SQL repair is attempted. A matching final file can be retried; a
different existing file or URL is rejected.

## Commands and health

Use noninteractive SSH throughout; it does not alter the operator's Bash history.

```bash
ssh -T -p 22222 root@2.29.153.161 'systemctl status alma-vicidial-recording.timer alma-vicidial-recording.service --no-pager'
ssh -T -p 22222 root@2.29.153.161 'journalctl -u alma-vicidial-recording.service -n 20 --no-pager'
ssh -T -p 22222 root@2.29.153.161 '/usr/local/sbin/alma-vicidial-recording --health'
ssh -T -p 22222 root@2.29.153.161 'find /var/spool/asterisk/monitor /var/spool/asterisk/monitorDONE -maxdepth 1 -type f -name "*.wav" -printf "%f %s bytes\n"'
```

Health exits nonzero for a failed/running/stale (>600 seconds) status. Journal and
systemd expose failures locally; no external alert delivery is configured. A
pending active writer is deferred rather than treated as a conversion failure.
One malformed old pending record stops its batch for review. The 20-record limit
and per-file native delay are a small test-server baseline, not capacity acceptance.

Read-only metadata query (does not print credentials):

```bash
ssh -T -p 22222 root@2.29.153.161 \
  "mariadb --no-defaults -N -B -e 'SELECT recording_id,filename,length_in_sec,location FROM asterisk.recording_log ORDER BY recording_id DESC LIMIT 20'"
ssh -T -p 22222 root@2.29.153.161 'soxi /var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav'
ssh -T -p 22222 root@2.29.153.161 'curl --head --silent --show-error https://2.29.153.161/RECORDINGS/PR01_SYNTHETIC-all.wav'
```

The synthetic nonexistent URL must not return audio: an unauthorized source gets
403; a permitted source reaches 307 and then individual authentication/mapping
checks. Avoid POSTing to
native recording redirects or putting credentials in any URL, command, log or
browser storage. Normal portal/native authentication remains unchanged.

## Historical recovery: recording 2

Before running recovery, review database identity/closed timestamps, source WAV
metadata, writable-FD state and existing native processor activity. Existing
verified backups and the scoped deployment rollback must be available. Do not
run both native processors on the same directory.

```bash
ssh -T -p 22222 root@2.29.153.161 \
  '/usr/local/sbin/alma-vicidial-recording --recording-id 2'
```

This explicitly authorizes the known completed synthetic test recording through
the wrapper. It stages copies and leaves **both original raw files unchanged**.
It creates the private `20261107-024337_7000000001-all.wav` and permits the native
script to change recording 2's location from NULL to its HTTPS `/RECORDINGS/` URL.
That URL is now gated by PR-01C individual authentication. The original input is reused byte for byte, so no gain,
mixing or transcoding is introduced. Recording 1 is excluded and remains NULL.
Repeated recovery validates the receipt and returns `already_finalized`.

Do not invoke the upstream script's `--test` against the live database: that
mode can still update recording locations. `native_fixture.py` instead supplies
synthetic PCM and a private DBI test double and never connects to MariaDB.

For private recovery listening, use an approved desktop and transfer only this
synthetic test recording by SSH/SCP. Normal entitled browser playback is also
available through VICIdial. Restrict the local
directory before copying and do not upload the audio to this repository or chat:

```bash
mkdir -m 700 -p "$HOME/pr01-private-listening"
scp -P 22222 root@2.29.153.161:/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav "$HOME/pr01-private-listening/"
chmod 600 "$HOME/pr01-private-listening/20261107-024337_7000000001-all.wav"
```

Play with an existing trusted local player. Report only whether both recognizable
voices are present. Metadata and nonzero samples do not establish intelligibility.

Acceptance result for recording 2: the operator confirmed **both voices** in this
finalized WAV. This is listening evidence from the operator, separate from the
worker's byte-identity, duration and decoder checks. It does not establish browser
publication or a new automatic recording lifecycle.

## Original finalizer deployment and rollback (PR-01 history)

`deploy.py` consumes the saved pre-change hashes and refuses concurrent changes or
an existing PR-01 installation. It checks idle telephony, backup health and audio
job conflicts, creates a root-only metadata-preserving rollback archive, rehearses
extraction into a temporary directory, and validates the rollback script. It
installs only the wrapper/policy/units, replaces the single mixer schedule and
adds the explicit Apache deny. `httpd -t` must pass before a graceful Apache reload.
No Asterisk, MariaDB, portal, firewall or TLS service restart is needed.

The actual rollback package path is in the acceptance evidence. Review hashes for
later operator edits before restoring. Run its `rollback.sh` through SSH after any
recording job finishes. It restores original cron/Apache files and removes the new
units, wrapper and policy; raw/final audio, receipts and the native database URL
are intentionally retained. It does not issue SQL updates or delete audio. The
old broken `--MIX` schedule is restored. No recording Alias existed in that
configuration, so restoring it does not publish the final storage directory.

Existing backups already include final/raw recordings, systemd units and the
installed wrapper. This task's dedicated policy and receipt directory are outside
the current full-backup inventory; retain the scoped package and source, and add
those two paths during the next approved backup-inventory maintenance. A restored
deployment must set a new automatic timestamp and review historical files rather
than blindly replaying old processing state. Backup retention/scheduling is not
changed here.

## Playback operations and troubleshooting (PR-01C)

Use Administration as an entitled supervisor, then Lead Details → lead 8, or User
Stats → **agent 1000**, date **2026-11-07** → Recordings for this Time Period.
Recording 2 belongs to agent 1000, not supervisor 6666. An empty recordings section
for another user/date is expected. Native date/time filtering should be checked
before diagnosing a missing playback link.

```bash
ssh -T -p 22222 root@2.29.153.161 'systemctl status alma-portal-443 alma-vicidial-recording.timer --no-pager'
ssh -T -p 22222 root@2.29.153.161 'journalctl -u alma-portal-443 -n 30 --no-pager'
ssh -T -p 22222 root@2.29.153.161 'journalctl -u alma-vicidial-recording.service -n 20 --no-pager'
ssh -T -p 22222 root@2.29.153.161 'httpd -t'
```

`recording_access` journal events contain username, recording ID, source IP and
status only; Apache's dedicated media log omits recording filenames/query strings
and Referer. Never add Authorization, cookies, database credentials or failed
passwords to logs. These local logs are not delivered external alerts.

- 401: normal Basic challenge/missing or invalid credentials.
- 403: IP denied, account disabled, missing recording permission, restricted
  group or prohibited cross-origin request.
- 404: no unique completed native association, no receipt, changed inode or
  missing finalized file. Do not manually repair native recording locations.
- 416: invalid/unsupported byte range; only one byte range is supported.
- 429: bounded failed-authentication rate limit; allow its 15-minute window to
  expire rather than changing credentials or disabling protection.
- 503: unsupported native authentication/logging policy, database failure, feature
  disabled, or four concurrent media streams already active. Inspect locally.
- 499 journal outcome: client disconnected or a stream ended early, including
  IP expiry or the 120-second per-request limit. Browser Range requests can resume.

Enroller failure leaves raw audio and the finalizer unchanged and fails the
recording service visibly. Review the native receipt, final file and association
before rerunning `/usr/local/sbin/alma-vicidial-recording-enroll`; never chmod a
whole spool or bypass the receipt check. Projection receipts are verification
state, not a second recording catalog. Preserve them for permission rollback.

Basic credentials are cached by the browser, with native logout limitations.
Portal sign-out does not clear that cache. Account/password/permission changes are
checked on each new media request; an already opened stream may finish. IP expiry
is additionally checked during streaming. A reload cannot retract bytes already
downloaded. Responses use `no-store, private` and disallow cross-origin retrieval.

### Scoped playback rollback

The root-only package is `/root/alma-recording-pr01c-20261106T234734Z`. Archive
extraction, ownership/mode/hash comparison, Python compilation and `--check` were
verified. Live rollback was not executed against the accepted deployment.
First wait until the finalizer oneshot is inactive, and inspect for later edits:

```bash
ssh -T -p 22222 root@2.29.153.161 '/root/alma-recording-pr01c-20261106T234734Z/rollback.py --check'
ssh -T -p 22222 root@2.29.153.161 '/root/alma-recording-pr01c-20261106T234734Z/rollback.py'
```

The second command changes production: it restores the prior Apache/portal/auth
files, validates and gracefully reloads Apache, removes the playback module,
enroller and two drop-ins, restarts only the portal, revokes only added SELECT
columns, and restores enrolled files' prior groups/modes after hash verification.
It preserves raw/final audio, native metadata, the finalizer/timer and operator's
approved user permission. It does **not** restore the old mismatched MIX cron.
Do not run the older PR-01 rollback to remove playback alone.

`deploy_playback.py` is a guarded one-time deployment for the recorded baseline,
not a general update command. A repeat refuses existing enrollment or changed
hashes. Review and prepare a new scoped baseline for subsequent changes.

### Remaining operational acceptance

No new call, reboot, load test, full browser matrix, strict Basic logout guarantee,
group-specific recording policy or native 2FA integration was tested here. Future
recordings are enrolled after guarded finalization, with timer execution observed;
a newly captured call was not needed or initiated. Back up the added projection
directory, enrollment helper and source with approved inventory maintenance; this
task did not expand the backup system or run another full backup.
