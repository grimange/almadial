# PR-01C — native recording playback acceptance

Date: 2026-11-07 Asia/Manila. Target: dialer01 / 2.29.153.161, SSH 22222.

**Verdict: `VICIDIAL_RECORDING_PLAYBACK_VERIFIED`.** All seven scoped product
criteria passed. This combines deterministic developer tests, read-only deployed
checks and independent operator browser acceptance; it is not an independent code
review, a new-call/reboot acceptance or a production-readiness verdict.

## Executive result and pre-change baseline

An entitled supervisor can play the unchanged finalized recording through native
Lead Details and User Stats, using the browser's existing VICIdial Basic cache.
No portal cookie or extra login is required. The operator confirmed playback from
both pages, no extra prompt and seeking, plus denial for an unauthenticated
browser and an unentitled account behind the same authorized public IP.

Before deployment, recording 2 was finalized but intentionally private. Its native
location was already HTTPS; the native processor and metadata were not changed
for playback. Existing native source is VICIdial 2.14b0.5 / retained SVN r4037.
The recording belongs to agent **1000**, lead 8, dated **2026-11-07 02:43:37**;
the associated call is 1793990617.45. Looking at supervisor 6666's own statistics
or another date initially hid the recording link. The operator subsequently
confirmed that the User Stats recording link works. No rendering/source repair
was applied.

The final WAV remains 892,204 bytes, mono 8 kHz PCM16, 55.76 seconds:

```text
/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav
SHA-256 f726a1d5615d50767c8a0f4794f68a544322ffa70037d1ef1a7bb3667aa4be59
```

Both voices were confirmed by the operator in the previous finalized-file
listening acceptance. Byte identity is unchanged. No production audio was
downloaded or placed in this repository by this assignment.

## Exact changes

| Repository source | Installed destination / purpose |
|---|---|
| `operations/portal443/portal.py` | `/opt/alma-portal-443/portal.py`; dispatch protected media methods in existing process |
| `operations/portal443/vicidial_auth.py` | `/opt/alma-portal-443/vicidial_auth.py`; extract existing password verifier for reuse |
| `operations/portal443/recording_media.py` | `/opt/alma-portal-443/recording_media.py`; per-request authentication, native policy, safe file resolution and media |
| `operations/portal443/02-alma-vicidial-ssl.conf` | `/etc/httpd/conf.d/02-alma-vicidial-ssl.conf`; exact media proxy, existing IP gate and sanitized media logging |
| `operations/recording/enroll_playback.py` | `/usr/local/sbin/alma-vicidial-recording-enroll`; root receipt projection and narrow final-file read enrollment |
| `operations/recording/playback-backend.conf` | `/etc/systemd/system/alma-portal-443.service.d/recording-playback.conf`; feature flag |
| `operations/recording/playback-enroll.conf` | `/etc/systemd/system/alma-vicidial-recording.service.d/playback-enroll.conf`; enrollment after existing finalizer |

Additional local task files: `operations/recording/{collect_playback,deploy_playback,
verify_playback,test_enroll_playback}.py`, portal tests
`tests/{test_recording_media.py,synthetic_playback_browser.py,playback-browser.js}`,
this report, recording README/design amendment, and sanitized
`operations/qa/evidence/recording-playback/`. No upstream source or schema changed.

Only the existing read-only `alma_portal_auth@localhost` SELECT grants expanded:

- `system_settings`: `two_factor_auth_hours,allow_ip_lists,system_ip_blacklist,log_recording_access`.
- `vicidial_users`: `access_recordings,view_reports,user_group,failed_login_count,last_login_date`.
- `vicidial_user_groups`: `user_group,allowed_campaigns,allowed_reports,admin_viewable_groups`.
- `recording_log`: `recording_id,filename,end_epoch,length_in_sec,location`.

The scoped rollback records exactly which additions to revoke. No new account,
credential change or application-data SQL write was performed by the integration.

## Authentication, authorization and routing

1. Apache's unchanged Lua authorization checks the actual connection source IP.
   Source identity headers are overwritten; user-supplied forwarding headers
   cannot supply a grant. The backend also checks worker-confirmed IP expiry.
2. Only exact safe native WAV URL shapes proxy to loopback 19091. Legacy
   `/RECORDINGS/<name>-all.wav` returns fixed same-origin 307 to
   `/vicidial/recordings/<name>-all.wav`; no user redirect target is accepted.
3. The canonical handler requires Basic credentials with the native
   `CONTACT-CENTER-ADMIN` realm. It verifies the existing plaintext/bcrypt modes
   independently on every request. Missing/invalid credentials get 401; supplied
   usernames, IP grants and portal cookies never constitute authenticated identity.
4. Active level 8/9, non-API-only, no forced password change, native lock state,
   `access_recordings >= 1`, View Reports and unrestricted native campaign/report/
   admin-group visibility are required. Restricted groups are denied, not mapped
   into an invented record-level scope. Permission withdrawal is checked anew.
5. Unsupported native 2FA, IP-list/blacklist or recording redirect/log policy fails
   closed with 503. No native 2FA evidence is inferred from Basic authentication.
6. Native metadata must identify exactly one completed recording with the exact
   expected HTTPS location. A root-issued finalizer receipt and enrolled inode
   fingerprint must agree. Raw/unreceipted/missing/ambiguous files are denied.

User 6666 alone was approved for Access Recordings = 1 through native UI. The
operator temporarily enabled System Settings → Log Recording Access to reveal
the hidden field and restored it to **0** afterward. SELECT confirmed active Y,
level 9, View Reports = 1, ADMIN with global native visibility, Access Recordings
= 1, Log Recording Access = 0. No other user was opted in. Native password
reflection in `recording_log_redirect.php` is avoided entirely, not patched.

## File and media boundary

The existing guarded finalizer is unchanged. Its post-success enrollment helper
checks native association, root receipt SHA, PCM shape and duration before
changing **only enrolled final inodes** from root:root 0600 to root:alma-portal
0640. A root-owned 0750 projection directory holds group-readable verification
receipts, including original permissions for rollback. No playback audio copies
or second catalog are created. Raw originals and root-only native receipts stay
unchanged.

The backend opens relative to the fixed directory using `O_NOFOLLOW`, rejects
nonregular files, multiple hardlinks, non-root ownership, unexpected permissions
or changed inode/size/mtime/ctime. The verified open descriptor supplies bytes;
the path is not reopened after authorization. The backend group cannot write the
final file or projection. A compromised backend can still read enrolled final
audio; this is the explicitly bounded read privilege of the design.

GET/HEAD and a single byte range are supported: 200/206, `audio/wav`, correct
length/range, 416 with total length, 405 for unsupported methods. Delivery reads
at most 64 KiB per block, limits concurrent authenticated streams to four and
limits each response to 120 seconds. IP state is rechecked during streaming.
Responses use `no-store, private`, nosniff, same-origin media CSP, no Referer and
cross-origin request denial. Failed account attempts are bounded per IP/user and
per IP. Logs contain status, user, recording ID and IP, never passwords or audio.
Apache media logs omit filenames, query strings, cookies and Authorization.

## Tests and evidence

Evidence directory: [recording-playback](evidence/recording-playback/README.md).

| Layer | Result | Evidence and scope |
|---|---|---|
| Portal Python suite | PASS, 51 tests | `python-tests.txt`; existing portal/auth/grant tests plus synthetic media authentication, policies, mappings, methods, ranges, large-file bounded reads and concurrency |
| Recording Python suite | PASS, 28 tests | `finalizer-tests.txt`; existing finalizer tests and synthetic enrollment, idempotence, mismatch/symlink/hardlink failure preservation and private-umask directory access |
| Chromium synthetic browser | PASS, 4 journeys | `chromium.json`, Chromium 157.0.8103.38; cached credentials through 307, GET/HEAD/206, embedded seeking, built-in player seeking, unauthenticated denial. Only initial mock Administration challenge accepted; further prompts cancelled |
| Installed Python/systemd/Apache | PASS | `deployment.txt`; compile, systemd verification, `httpd -t`, portal restart and graceful Apache reload |
| Deployed metadata-only checks | PASS | `integration.json`; matching-realm 401, fixed 307, service-user SELECT/open verified inode without reading audio; strict trusted HTTPS homepage and denial probes |
| External IP and spoofing | PASS | Unauthorized canonical/legacy/raw/directory requests 403, spoofed source headers 403; agent/admin navigation 303, API 403; HTTP root 403 |
| Native files and firewall | PASS | `regression.json`, `final.json`, `final-comparison.json`; pre/post hashes, identical permanent/runtime firewalld and unchanged SIP gate policy/membership; live nft expiry countdowns and packet counters naturally differ |
| Automatic enrollment | PASS | `supplemental.json`; repeated scheduled finalizer post-hook reports `already_enrolled`, service success |
| Operator actual browser | PASS | `operator-acceptance.json`; actual native pages and same-IP negative tests, not synthetic substitutes |

Local commands executed:

```bash
python3 -m unittest discover -s operations/portal443/tests -v
python3 -m unittest discover -s operations/recording -p 'test*.py' -v
python3 operations/recording/verify_playback.py
python3 operations/recording/collect_playback.py operations/qa/evidence/recording-playback/final.json
```

Tests use the existing unittest runner; pytest was unavailable and was not
installed. Local HTTP fixtures require loopback permission. An initial sandbox
socket denial was rerun with the appropriate execution permission and passed.
The browser harness uses synthetic accounts/audio only. An initial private-umask
enrollment directory traversal failure was corrected to root:alma-portal 0750 and
covered by a regression test; final installed service-user access passed.

## Core acceptance matrix

| ID | Criterion | Result | Evidence category and reference |
|---|---|---|---|
| PLAY-01 | Authorized supervisor plays recording without an extra prompt | PASS | Operator confirmed both native pages and no extra prompt; synthetic browser corroboration |
| PLAY-02 | Lead Details playback works | PASS | Operator confirmed lead 8 / recording 2 in actual native browser |
| PLAY-03 | User Stats playback works | PASS | Operator explicitly confirmed recording link works; use agent 1000 and call date |
| PLAY-04 | Unauthorized user on same public IP is denied | PASS | Operator confirmed ordinary-account denial on the same authorized network; synthetic permission tests |
| PLAY-05 | Direct media URL requires valid individual authentication | PASS | Operator confirmed fresh-private-window prompt cancellation gives no audio; deployed 401 and external 403 checks |
| PLAY-06 | HTML5 audio playback and seeking work | PASS | Operator confirmed seeking in both pages; synthetic embedded/direct player and 206/HEAD checks |
| PLAY-07 | Existing recording, portal and telephony functionality remain intact | PASS | Unchanged audio/native/Lua/finalizer/TLS/cron hashes; firewall snapshots; unchanged Asterisk/DB/PHP/Fail2Ban PIDs, healthy timers and public portal. Bounded regression, no new call |

## Regression and scope limits

Only the portal service was restarted; Apache was gracefully reloaded and retained
its main PID. Asterisk, MariaDB, PHP-FPM, Fail2Ban, SIP gate and SSH were not
restarted. UDP 5060/5061, RTP rules, SSH 22222, Personal/Shared Office grant policy
and worker timers remain unchanged. No live grant was revoked. The backup and
TLS timers remain enabled; existing verified backup health was checked, with no
new full backup. No calls or registrations were initiated.

Important limitations:

- Basic browser caching/logout has native limitations. Portal logout does not
  clear native Basic credentials. Current account state is checked on subsequent
  requests; an already opened stream can finish, and downloaded bytes cannot be
  revoked. No separate idle-session manager was introduced.
- Only entitled global supervisors and current supported native authentication
  policy are accepted. Enabling native 2FA or incompatible native logging/IP
  policy deliberately disables this feature until integration is separately
  reviewed. Group-scoped entitlements are outside this initial policy.
- Current raw spool/WAV permissions were already asterisk-owned 0755/0644. They
  remain unpublished and unchanged, but are not root-only against local accounts.
  Local recording-permission hardening is a separate dependency, not a claim made
  by this web authorization acceptance.
- No new call, reboot, load test, exhaustive browser matrix, delivered external
  alert or complete independent security review was performed. Timers are enabled
  and automatic enrollment observed; this is not a reboot test.
- Backup inventory maintenance should preserve the new projection state/helper
  alongside the existing finalizer state. This task did not expand backup scope.

## Rollback and Git provenance

Root-only scoped package: `/root/alma-recording-pr01c-20261106T234734Z`.
Original archive metadata/hash comparisons and extraction rehearsal passed;
rollback compiled and `--check` passed again after acceptance. Live rollback was
not executed, to preserve accepted operation. Wait for the finalizer to be idle
and review later file edits before applying:

```bash
ssh -T -p 22222 root@2.29.153.161 '/root/alma-recording-pr01c-20261106T234734Z/rollback.py --check'
ssh -T -p 22222 root@2.29.153.161 '/root/alma-recording-pr01c-20261106T234734Z/rollback.py'
```

The second command restores old Apache/portal/auth files, validates/reloads
Apache, removes playback components/drop-ins, restarts only the portal, revokes
only added SELECT grants and restores enrolled file group/mode after hash checks.
Raw/final audio, native metadata, the finalizer/timer, existing grants and user
6666's separately approved native permission remain. Projection receipts are
retained as a private recovery ledger. No old mismatched MIX cron is restored.

The deployment helper is one-shot and refuses a changed baseline or existing
enrollment; do not rerun it blindly. This task's source/tests/docs/evidence are
committed separately from unrelated work; no push. Obtain the exact acceptance
commit with `git log -1 --format=%H -- operations/qa/VICIDIAL_RECORDING_PLAYBACK_ACCEPTANCE.md`;
the final operator handoff records that identifier. Other local audit reports and
unrelated repository edits are preserved outside this commit.

**Final verdict: `VICIDIAL_RECORDING_PLAYBACK_VERIFIED`.**
