# M3-01 — Apache/PHP and VICIdial application flows

Implementation owner: Codex. Independent QA: Google Antigravity (AGY), pending.
Baseline: `512cf11705255da556d977328774f64ce39a6947`.
Canonical task: ALM-23 / `ed6ilp2yw5ldla4ovm44i3vc`.
Qualification: 2026-11-10 UTC / 2026-11-11 Asia/Manila.

## Contract and existing runtime

ALM-23 requires an owned web provider and virtual host, preservation of foreign
sites, validation before reload, real admin/agent web transactions, PHP/database
verification, conflict handling and repeat-run evidence. HTTP 200 alone is
insufficient. HTTPS-ready ownership is recorded; certificates and HTTPS
activation belong to S08. The pending M2 all-in-one orchestration governance
clarification is unchanged. This stage does not provide a unified installer.

The authorized disposable target is Multipass `almadial-alm16-rpm`, machine ID
`0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2 (Lavender Lion). The existing
recovery record is snapshot `alm21-pre-runtime-20261110`; no new stop, snapshot,
reboot or destructive test was performed for ALM-23.

Accepted M2 already supplied Apache **2.4.63**, PHP-FPM **8.3.36**, MariaDB
**10.11.18**, VICIdial **r4037** and Asterisk **22.12.0**. Apache and PHP-FPM
were active/enabled. The selected Apache main file is
`/etc/httpd/conf/almadial-s05.conf`, selected by its existing service environment.
It loads `conf.modules.d/*.conf`, but does not load `conf.d/*.conf`. PHP uses
`/run/php-fpm/www.sock`. Required installed extensions include mysqli/mysqlnd,
mbstring, curl, gd and XML. No packages were installed or rebuilt.

## Implemented S07 ownership

`internal/provision/web*.go`, `web_native.py` and `cmd/almadial-web` add one
bounded provider using the accepted parent plan, runner, journal and lock.

| Action | Responsibility | Recovery |
|---|---|---|
| `web.00.files` | Create missing exact owned vhost and loader | Matching complete files reconcile; absent files are idempotent; foreign content conflicts |
| `web.01.services` | Validate Apache and PHP-FPM, then activate the vhost | Fresh active vhost reconciles; unknown transition blocks automatic replay |
| `web.02.verify` | Complete guarded secured bootstrap and verify authenticated admin/agent pages | Fresh satisfied state reconciles; unknown state blocks replay |

The only new configuration files are root:root 0644, `httpd_config_t`:

- `/etc/httpd/almadial-s07.conf`
- `/etc/httpd/conf.modules.d/99-almadial-s07.conf`

`WebVhost()` is the exact deterministic source. It defines
`<VirtualHost 127.0.0.1:80>`, `ServerName localhost`, `/var/www/html`, local-only
access, no directory overrides, the existing PHP Unix socket, authorization
forwarding and `X-AlmaDial-Web: S07`. It denies direct `.ht*`, `.user.ini` and
`options.php` access. The loader contains `Include /etc/httpd/almadial-s07.conf`.
Foreign Apache/PHP configuration is hashed and preserved. Conflicting owned
files are rejected, rather than overwritten. No certificate, public listener,
new authentication mechanism or optional portal was introduced.

Both native validations precede the one initial graceful Apache reload:

```sh
/usr/sbin/httpd -t -f /etc/httpd/conf/almadial-s05.conf
/usr/sbin/php-fpm -t
/usr/bin/systemctl reload httpd
```

Apache's main PID remained 849; PHP-FPM remained PID 796. Subsequent NO_OP and
reconciliation did not reload or restart them.

## Confirmed bootstrap and verification defects

Authenticated administration initially entered upstream first-login setup.
Pinned `admin.php`'s initial-settings handler changes the administrator password
and rewrites all phone passwords, conference secrets and other settings. Running
that handler would violate this task's preservation boundary. Source inspection
identified those writes; the form was not submitted.

S03 had already installed the protected random administrator credential.
`web_setup()` requires the root-only 0600, 48-hex credential reference and its
exact existing active level-9 ADMIN user 6666, with password-change flag N. It
accepts only one settings value Y or N and changes only
`asterisk.system_settings.first_login_trigger` from Y to N under table locks,
with the secured-account predicate. No credential, phone, schema or upstream
PHP file is changed.

The S07 downstream inspector explicitly normalizes only that flag to its known
S05 receipt preimage. Every other generator input remains checked. Default S05
inspection remains strict; S07 supplies the explicit downstream handoff for
this legitimate application-state transition.

The initial apply correctly made the intended changes, but its final verifier
rejected upstream markup separating the Logout link and `(6666)` with HTML tags.
That failed journal and binary are preserved. The correction checks normalized
visible text and rejects initial-setup, unauthenticated and redirect responses.
Authenticated native probes do not follow redirects. Focused tests reproduce
the markup defect and credential-forwarding boundary. A changed-policy journal
was not blindly resumed.

## Actual browser transactions

Playwright Chromium **157.0.8103.38** used host loopback
`http://127.0.0.1:18023`, temporarily forwarded to VM `127.0.0.1:80`. The VM's
listener policy was unchanged. Protected credentials were supplied in memory;
cookies, passwords and session tokens are excluded from committed evidence.

| Journey | Observed result |
|---|---|
| Administrator authentication | Anonymous endpoint rejected authentication; protected 6666 credential reached the authenticated system dashboard without onboarding or fatal PHP/database errors |
| Core navigation | Users, Campaigns, Lists and Scripts rendered their real application listings |
| Administrator write/read | Created disposable level-1 `ALM23QA`, edited its full name, then verified a fresh listing and database read |
| Authorization boundary | Add Script was denied by the existing `modify_scripts=0` permission; no grant was broadened |
| Agent entry/resources | Phone-login form and its CSS, JavaScript and image resources loaded; seven named resources returned the expected content types |
| Agent-facing authentication/write/read | Timeclock authenticated the fixture, clocked in, showed logged-in state on a fresh authenticated read, and clocked out; database recorded exactly one LOGIN and one LOGOUT |
| Invalid authentication | Timeclock rejected an invalid password with the expected inactive-user/password message |

The fixture and its timeclock status/events were removed after verifying LOGOUT
and zero live-agent rows. Application audit records remain attributable. Seed
phones and protected administrator credentials were preserved.

The pinned Timeclock password field has `maxlength=20`; the initial 48-character
fixture password was truncated and rejected. A separate 20-hex disposable
fixture password allowed the normal flow. The protected administrator password
was not shortened. The native 30-second timeclock cooldown also behaved as
expected. These upstream constraints are documented, not patched.

The agent phone form advances to campaign login with nonempty synthetic input;
the campaign selector is empty in this accepted dialing-inactive installation.
This is not evidence of phone credential validation or a completed dialer
session. No campaign, live-agent session, SIP registration, call, WebRTC, RTP or
media acceptance was attempted. Those capabilities remain downstream.

PHP-FPM application logs contain upstream undefined-variable warnings, but no
fatal/uncaught errors in the inspected log. The browser also requested a missing
favicon. These did not block the tested flows and remain nonblocking findings.
No broad PHP warning suppression or upstream source patch was applied.

## Identity, NO_OP and recovery

The CLI exposes read-only `inspect --accepted-runtime FILE`, pure
`plan --input FILE`, and explicit `lab-run` / `lab-resume`. All require the
accepted pinned manifest and VICIdial, Perl, Asterisk, signature, keyring,
scratch and pjproject inputs; `--help` lists exact flags. The lab operations
add `--authorization FILE --state-dir DIR`.

Authorization binds host, stage, parent, manifest, policy, snapshot and operator
evidence, requires `ALM-23/disposable-lab`, CANDIDATE acknowledgement and dialing
inactive. `journal.Acquire`, durable action intent, bounded execution, fresh
postconditions and persisted outcomes precede/complete effects. The stack
manifest remains CANDIDATE.

Final `inspect` and `plan` used the final reproducible binary and found the
accepted foundation and foreign configuration unchanged, all S07 postconditions
satisfied, and an empty graph. Authorized `lab-run` returned `no_op: true`, zero
traces and no host effect. A controlled copied-journal fixture with a running
verification record reconciled all three actions to succeeded with zero traces.
This is real installed-host reconciliation against a synthetic journal, not a
new live crash campaign. An earlier malformed fixture was rejected and retained.
Go tests additionally interrupt after each of the three effects, reject stale
identities/authorization, enforce lock contention, and detect fresh-state drift.

## Preservation and developer verification

Before/after read-only observations matched: 725 RPMs and repository definitions,
schema 1748/catalog, all 107 columns of `callin` and `gs102`, protected credential
hashes, Asterisk configuration/modules, application campaign/lead/hopper/call
tables, identities and recording storage. Phone digest remains
`bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`.
FastAGI retains one parent/five workers at loopback 4577. Updater/listener remain
disabled/inactive; application cron has zero active jobs. SELinux is Enforcing,
owned files have expected labels, and no AVC denial appeared in the inspected
kernel journal window. Existing MariaDB, AMI, Asterisk HTTP and Apache listeners
remain loopback-only. Existing Asterisk UDP sockets were unchanged; no SIP
transport was introduced.

| Check | Result |
|---|---|
| Go 1.27.3 fmt / repository fmt-check / vet | PASS |
| `go test ./...` and `go test -race ./...` | PASS each: 211 top-level tests, 1,349 test/subtest pass events |
| Focused `TestWeb` | PASS: 11 pass events, including interruption after each effect |
| `go mod verify`; `go mod tidy -diff` | PASS |
| Python discovery | PASS: 188 tests, including seven S07 native-adapter tests |
| Scaffold, candidate stack and config-generation checks | PASS |
| CLI smoke / terminal and nonterminal journeys | PASS: 17 / 11 |
| Apache/PHP-FPM config; PHP syntax of admin, agent and timeclock entry points | PASS |
| Linux amd64 reproducibility | PASS: all eight command targets identical across two isolated final builds |
| `git diff --check` | PASS |
| Hosted CI / HTTPS / telephony qualification | NOT_RUN; outside this developer verification |

Final `almadial-web` binary SHA-256:
`510215e0120c0e98de80d9e1c63493c14e0d986ac1cc52d94d901a4c21e91b68`.
All 372 previously inventoried unrelated local files retain their hashes.
No Git commit, push, M2 evidence rewrite or downstream task start was performed.

Protected evidence: `~/almadial-lab/evidence/m3-01/20261110T163346Z-alm23/`
(0700), with SHA-256 ledger, exact command results, before/after observations,
browser transactions/screenshots, failed original journal, successful NO_OP and
copied-journal reconciliation, test results and final build hashes. Kaneo handoff
is recorded separately after sealing the primary ledger. Developer verification
does not replace independent AGY QA or acceptance.
