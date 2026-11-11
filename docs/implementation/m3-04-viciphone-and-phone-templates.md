# ALM-26 / M3-04 — Pinned VICIphone and generated phone templates

Developer qualification: 2026-11-11–12 (Asia/Manila). Starting repository commit:
`00c8e1b24f39d58953f20ab3737a421fff98d816`. Independent AGY QA remains required.

## Contract and ownership

Canonical Kaneo task `wcip8y6y5jezllppbokixauc` requires the accepted M0 phone
revision, self-hosted resources, protected synthetic bootstrap identity,
VICIdial settings integration, real browser registration, invalid-credential
rejection, refresh/reconnect, and stable rerun credentials. ALM-24 and ALM-25
are Accepted prerequisites. Full call/media acceptance belongs to ALM-27.

The implementation uses **S10** and `cmd/almadial-phone`, following the accepted
S07 web, S08 TLS and S09 firewall stages. The original draft labels the phone
S09; that label predates the accepted firewall stage. This report identifies the
actual implemented stage without rewriting historical plans or the unresolved
M2 milestone-scope clarification.

The qualified topology is deliberately bounded: `localhost`, Apache HTTPS on
loopback 443, and direct Asterisk WSS on loopback 8089. The S10 request must be:

```json
{"Identity":"alm_s10_9061","Hostname":"localhost","WSSURL":"wss://localhost:8089/ws"}
```

No new administrator account is needed: the accepted S07 administrator remains
unchanged. One explicit synthetic phone and settings container are created.

## Pinned assets and notices

| Input | Immutable identity |
|---|---|
| VICIphone | `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` |
| Source archive SHA-256 | `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` |
| Bundled SIP.js | 0.20.1 |
| SIP.js SHA-256 | `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` |
| Upstream LICENSE SHA-256 | `e7ae8b5da17709474ba25316441bc064148841cf101e7eef318c3c3ee5a0827b` |

`LoadPhoneSource` verifies the candidate manifest artifact, archive digest, PHP,
phone JavaScript, SIP.js and license digests, and retains verified bytes. It
applies only the bounded M0 compatibility changes: suppress sensitive SIP debug
logging and reconnect the SIP.js transport before the visible Register action.
The upstream AGPL notice and bundled dependency notices remain in the deployed
assets. AlmaDial does not relicense them.

The local deployment has 47 files under `/var/www/html/almadial-phone`, including
`index.php`, `viciphone.php`, JavaScript, CSS and notices. No runtime CDN is
required. Browser resource observations contain only local paths, all successful.

## Generated phone and PJSIP configuration

| Object | Ownership and behavior |
|---|---|
| VICIdial phone | `alm_s10_9061`, protocol PJSIP, webphone enabled; synthetic extension 9061 |
| Settings container | `ALMADIAL_S10`, `WEBPHONE_SETTINGS` |
| Credential | Persistent random 32-character hexadecimal value; root-only `/etc/almadial/secrets/phone-s10`, mode 0600 |
| Transport | `almadial-s10-wss`, WSS, loopback bind |
| Endpoint / AOR | `alm_s10_9061`; one contact, replacement permitted, 60–600-second expiry |
| Authentication | `almadial-s10-auth`; generated credential matches the VICIdial phone |
| Context | `alm-s10-no-calls`; no dialplan routes are installed |
| PJSIP include | `/etc/asterisk/almadial-s10-phone.conf`, root:asterisk 0640 |
| Receipt / backup | `/var/lib/almadial/phone-s10`, root-only receipt and previous wrapper |

The endpoint uses `webrtc=yes`, `direct_media=no`, `rtp_symmetric=yes`,
`force_rport=yes`, `rewrite_contact=yes`, and ulaw/alaw. These are configuration
prerequisites; this assignment does not establish working media. The settings
container disables dial-on-registration and enables no campaign processing.

Only one exact include is appended to the accepted `pjsip.conf` wrapper.
Synthetic SQL inserts name explicit fields and execute through the protected
client defaults file and stdin. The complete managed phone/container rows are
sealed in the receipt. Inspection compares all 107 phone columns. Foreign
phones and containers remain outside S10 ownership.

The small local launcher reads the phone/settings association from VICIdial and
accepts the synthetic password through normal POST forms. No password is put in
a URL or logged. It does not read the protected password into a public page.
The operator obtains that synthetic credential through authorized protected
access. `Open phone` then `Connect phone` loads the pinned phone.

## Installer and recovery

The immutable parent is the accepted S09 input. `PhoneInputs.Prepare` verifies
parent, manifest, source, request, policy and host identities. The only mutating
action is `phone.00.activate`. It uses the existing execution lock, durable
journal intent, bounded native runner and fresh postconditions. Owned files use
atomic replacement, fsync, explicit modes and restored SELinux labels.

The CLI accepts the same eight source flags used by the accepted application
qualification (`--manifest`, `--source`, `--perl-source`, `--asterisk-source`,
`--signature`, `--keyring`, `--scratch`, `--pjproject`) plus `--phone-source`.
Its operations are:

```text
almadial-phone inspect <source flags> --accepted-network FILE --request FILE
almadial-phone plan <source flags> --input FILE
almadial-phone lab-run <source flags> --input FILE --authorization FILE --state-dir DIR
almadial-phone lab-resume <source flags> --input FILE --authorization FILE --state-dir DIR
```

`inspect` is read-only, `plan` is pure, and execution needs the exact disposable
lab authorization bound to the plan and recovery snapshot. Actual qualification
used `/tmp/alm26-almadial-phone`; exact argv and outputs are retained in the
protected command records.

A narrow read-only S04/S05 handoff recognizes the exact S10 wrapper and excludes
only the dedicated synthetic phone from the old generator input seal. It does
not hide other phone changes or accept arbitrary wrapper content. Original S05
adapter bytes and policy identity remain unchanged. S10 verifies its own include,
assets, row, settings, credential, receipt and loaded PJSIP objects.

Unknown partial filesystem/database state fails closed; it is not blindly
reimported or overwritten. Complete files/rows with missing runtime objects can
reconcile by reloading PJSIP only. Completed-state copied-journal recovery
succeeded after fresh postcondition checks, with `no_op: true` and `traces: []`.
The canonical journal at `/var/lib/almadial/alm26-reload` was unchanged. An invalid
first copied fixture was correctly rejected; the corrected fixture used the
accepted running-action exit sentinel -1.

Fresh installed-host inspection produced an empty graph `[]`. Authorized repeat
execution returned `no_op: true`, no runner traces, no credential rotation, and
no Apache/Asterisk reload. Runtime readiness checks installed objects rather than
requiring a browser contact after the browser has closed.

## Confirmed defects and bounded corrections

- An initial synthetic identity exceeded VICIdial's login width. It was replaced
  with the stable `alm_s10_9061` and a context within the schema width. That failed
  attempt inserted no phone row; explicit owned-state cleanup was recorded.
- `pjsip reload` is unavailable on the accepted Asterisk CLI despite a zero CLI
  process exit. Activation now uses `module reload res_pjsip.so` and checks the
  resulting loaded objects. The existing valid deployment reconciled without
  rewriting assets or credentials.
- The accepted S05 generator seal initially rejected the additional owned
  include. The exact read-only handoff above fixes that integration boundary;
  targeted Go/Python tests preserve rejection of foreign changes.

Two browser-harness sequencing failures were corrected in the harness, with no
additional application change. An initial regression harness used umask 077 in
child processes, invalidating pre-existing permission fixtures; the final run
used the repository's normal umask 022 and all tests passed. A final native
service/cron probe initially used obsolete names; corrected probes confirmed
the actual `almadial-s06-*` units and `/etc/cron.d/almadial-s06`. The cron
probe also recognizes `CRON_TZ` as an environment assignment. All earlier failed observations remain attributable
in the evidence directory.

## Actual laboratory and browser results

VM `almadial-alm16-rpm`, machine ID
`0fffa9a4683f35b696f0b6ee7330734b`, recovery snapshot
`alm25-pre-firewall-20261111`. AlmaLinux 10.2, Asterisk 22.12.0, VICIdial r4037,
MariaDB 10.11.18; existing accepted Apache/PHP and private laboratory CA retained.

Real Chromium 157.0.8103.38 was driven through Playwright's Chromium API. The test
used a dedicated temporary NSS trust store containing the public S08 lab CA,
with `ignoreHTTPSErrors: false`. Local forwarding carried browser
`https://localhost:8443` to VM loopback 443 and `wss://localhost:8089/ws` to VM
loopback 8089 through authorized Multipass execution channels. This preserved the
certificate hostname and S09 policy without adding public listeners. This was a
real Chromium test, not a curl-only or WebSocket-only substitute.

| Check | Actual result |
|---|---|
| Invalid synthetic password | SIP 401; UI Unregistered; zero Asterisk contacts |
| Valid password | REGISTER challenge/authentication followed by SIP 200; UI Registered; exactly one matching Asterisk AOR contact |
| Page refresh | Registered again, exactly one contact |
| Unregister | Zero matching contacts |
| Visible Register reconnect | Registered, exactly one contact |
| Sustained registration | PASS at 65 seconds and 610 seconds; one contact, further REGISTER authenticated successfully |
| Resource loading | Local successful HTTP responses; no browser JavaScript errors |

Server observations used `asterisk -rx 'pjsip show aor alm_s10_9061'` and were
captured independently of the UI. SIP evidence stores only message start lines,
never Authorization headers or passwords. Browser profiles/cookies are excluded
from exported evidence. No call was placed; Asterisk reported zero active calls,
zero channels and zero calls processed.

## Preservation and regression

Before/after comparison passed for the 731-package inventory, repositories,
MariaDB catalog/schema 1748, all 107 columns of `callin` and `gs102`, existing
credential files, identities, recording trees and representative campaign/lead/
hopper/call data. Only the expected S10 PJSIP include and wrapper changed within
Asterisk configuration. Existing modules and accepted parent inspections pass.
FastAGI has one parent and five workers; updater/listener remain disabled and
inactive, managed application cron has zero active jobs. SELinux is Enforcing.
Firewalld remains active; application and infrastructure listeners remain local.

The S08 certificate generation remains
`30ee4d4176b8783d4e085b06b8f1fa51a2421a498a2d1f61464fda039fbc56c1`, with
fullchain-file SHA-256 `37772764ae226ea5cb6f06904b3fb5767318690fd719da507fbf60b40c6bb008`.

| Validation | Actual result |
|---|---|
| `go vet ./...` | PASS |
| `go test -json ./...` | PASS, 1,422 passing test/subtest events |
| `go test -race -json ./...` | PASS, 1,422 passing events |
| `go mod verify`, `go mod tidy -diff`, `make fmt-check` | PASS |
| Scaffold validation | PASS |
| Python discovery | PASS, 210 tests |
| PHP syntax, Apache configuration | PASS; existing ServerName warning is nonblocking |
| CLI smoke / terminal journeys | PASS, 17 / 11 |
| Linux amd64 reproducible builds | PASS, all 11 command targets, two identical builds each |
| `git diff --check` | PASS |
| Hosted CI | NOT_RUN |

## Evidence and remaining boundary

Protected evidence:
`/home/ra/almadial-lab/evidence/m3-04/20261111T153126Z-alm26/`.
Decisive artifacts include `reload-run.command.json`, `healthy-inspect.command.json`,
`noop-plan.command.json`, `noop-run.command.json`,
`reconciled-valid-run.command.json`, `browser-registration-complete.json`,
`browser-registration-renewed.json`, `baseline-comparison.json`,
`native-confirmed-checks.json`, `boundary-confirmed-checks.json`,
`preservation-final.json`,
`final-regression-results.json`, `reproducible-builds.json`, and `cli-smoke.json`.
The SHA-256 ledger is generated after final evidence collection; no historical
bundle is overwritten. Private keys, passwords, browser profiles and cookies are
excluded.

This establishes local trusted browser registration. Media, RTP, two-way audio,
agent/campaign workflows, external NAT profiles and the optional Dynamic Portal
remain outside ALM-26. ALM-27 is left unstarted. AGY must independently review
this implementation and reproduce the decisive registration path before
acceptance. No implementation commit, push, merge, release or manifest promotion
is performed.
