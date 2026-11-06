# PR-01 native conference recording finalization

This is a bounded integration for `dialer01`, not a VICIdial fork or an AlmaDial
installer stage. It uses the installed, hash-pinned
`AST_CRON_audio_1_move_VDonly.pl` unchanged. It is suitable only for the current
8309 conference listener: the input WAV contains the conference mix and the output
WAV has no samples. Nonempty output, stereo modes and other codecs require review;
they are rejected rather than silently dropping a participant.

## Security boundary

**Browser publication is disabled.** Native report permissions and
`recording_log_redirect.php` do not authorize the subsequent static media request.
The portal authorizes an IP, including shared offices, rather than an individual
VICIdial recording reader. The installed redirect also has a password-reflecting
failed-login branch. No credentials are supplied to that endpoint by this tool.

There is no recording Alias, directory listing or new authentication service.
Apache explicitly denies `/RECORDINGS/` and the final storage directory. Files are
root-readable. A generated HTTPS database location is an association, **not proof
of accepted publication or usable playback**. Native User Stats/Lead Details
playback remains pending an approved individual media authorization solution.

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

The unmodified native script does not reliably propagate shell/SQL failures. The
wrapper therefore checks results and reports a failed stage with a nonzero exit
code. **File creation and native SQL are not one atomic transaction.** A failure
can leave a private final file or a native URL without a receipt. Originals are
retained, publication remains denied, and an operator must review the failure.
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

The last command must return 403 while publication is disabled. Avoid POSTing to
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
That URL stays denied. The original input is reused byte for byte, so no gain,
mixing or transcoding is introduced. Recording 1 is excluded and remains NULL.
Repeated recovery validates the receipt and returns `already_finalized`.

Do not invoke the upstream script's `--test` against the live database: that
mode can still update recording locations. `native_fixture.py` instead supplies
synthetic PCM and a private DBI test double and never connects to MariaDB.

For operator listening, use an approved private desktop and transfer only this
synthetic test recording by SSH/SCP (not the denied public URL). Restrict the local
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

## Deployment and rollback

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

## Required next acceptance

1. Approve a supported individual media authorization boundary; do not use IP-only
   publication as a substitute. Keep the failed-login password reflection separate
   from credentials used during acceptance.
2. After that boundary exists, map only the finalized storage path over HTTPS,
   deny listing, and verify authorized and unauthorized users on the **same** IP.
3. Have an operator make one authorized synthetic call through normal VICIdial,
   wait for automatic finalization, and verify User Stats, Lead Details, direct
   media denial, recording access logging and both voices. Do not start the call
   through this tooling.
4. Reboot survival and concurrent recording volume remain separate controlled
   acceptance tasks. Timer enablement is not evidence of a completed reboot test.
