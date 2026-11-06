# PR-08 — TLS renewal and deployment acceptance

Date: **7 November 2026 Asia/Manila**. Host: `dialer01`, `2.29.153.161`,
SSH TCP 22222. Baseline collected 6 November 20:31:25 UTC; staging exercise
20:33:30–20:33:35 UTC. This is acceptance of the scoped certificate integration,
not production acceptance of the complete VICIdial installation.

**Verdict: `VICIDIAL_TLS_RENEWAL_VERIFIED`**

## Executive summary

Corrected the obsolete, failure-masking certificate deployment hook while retaining
the existing Certbot 5.9.0 installation, Let's Encrypt IP certificate, HTTP-01
webroot and twice-daily systemd timer. One real staging issuance succeeded with
`--dry-run --run-deploy-hooks`, exercising the actual production hook against the
current trusted certificate. Apache validated and reloaded gracefully; a trusted
IP-identity TLS handshake verified that TCP 443 served the expected certificate.

A safe rejected-lineage test proved a failing hook produces a nonzero exit, a
failed systemd unit and a persistent unhealthy state. Two subsequent scheduled-
service checks were successful no-ops with no extra Apache reload. No production
certificate issuance was forced: the existing certificate was not yet due.

External alert delivery has **not** been configured or tested. The implemented
local failed state, systemd status and executable health check are functional;
an approved notification destination and independent monitoring runner remain a
PR-07 dependency. Full reboot and live rollback were not performed. Timer boot
enablement/persistence, a timer restart and an isolated rollback-archive extraction
were verified. These boundaries are separate from the tested renewal/deploy path.

## Evidence index

All evidence below is sanitized metadata, configuration without secrets, HTTP
status/headers, or command output. No private key, token, cookie, database dump or
recording audio was copied into the repository.

| Reference | Artifact | Scope |
|---|---|---|
| E1 | [before.json](evidence/tls-renewal/before.json) | Original hook/unit/config hashes, file/served certificate, HTTP policy, services and firewall |
| E2 | [deployment.txt](evidence/tls-renewal/deployment.txt) | Scoped backup rehearsal, installed hashes and Apache syntax |
| E3 | [negative-test.json](evidence/tls-renewal/negative-test.json) | Real rejected hook, failed transient unit and failed health check |
| E4 | [staging-evidence.json](evidence/tls-renewal/staging-evidence.json) | Staging issuance, real hook receipt, reload, trusted served-certificate verification and recovered health |
| E5 | [normal-runs.json](evidence/tls-renewal/normal-runs.json) | Two installed service no-ops, timer restart/enablement and Apache journal |
| E6 | [after.json](evidence/tls-renewal/after.json) | Final service, TLS, HTTP and firewall snapshot |
| E7 | [final-details.json](evidence/tls-renewal/final-details.json) | Actual graceful reload command, timer boot link, assets, worker and portal-source hashes |
| E8 | [local-tests.txt](evidence/tls-renewal/local-tests.txt), [regression-results.json](evidence/tls-renewal/regression-results.json) | 14 deterministic tests and 34 assertions against collected host evidence |
| E9 | [recovery-preflight.json](evidence/tls-renewal/recovery-preflight.json) | Backup/schedule metadata, installed Certbot version and hook implementation |
| E10 | [renewal-config-diff.json](evidence/tls-renewal/renewal-config-diff.json) | Certbot-managed ARI retry metadata update; no manual renewal-policy edit |
| E11 | [scaffold-validation.txt](evidence/tls-renewal/scaffold-validation.txt) | Repository guidance validation; not a live runtime test |
| E12 | [acme-challenge.json](evidence/tls-renewal/acme-challenge.json) | Safe markers from Certbot logs: performing challenges, waiting for verification, cleanup and staging success |

## Original implementation and confirmed defects

The original renewal service ran:

```text
/opt/alma-portal-certbot/bin/certbot renew --quiet --deploy-hook /usr/local/sbin/alma-portal-cert-deploy
```

The original hook copied the certificate and key into `/etc/alma-portal/tls`, tried
to restart disabled `alma-portal.service`, and suppressed restorecon/restart/reload
failures with `|| true`. Apache already consumed the Let’s Encrypt lineage
directly. The actual Python backend is `alma-portal-443.service`, uses loopback
HTTP and does not consume the public TLS certificate. Neither portal service needs
a certificate-triggered restart.

The original hook also returned success when `httpd -t` failed because the reload
was inside a conditional. `try-reload-or-restart` could restart Apache and its
failure was suppressed. Successful no-op renewal checks in the old journal did
not prove successful issuance or deployment.

Inspection of the **installed** Certbot `_internal/hooks.py` showed `_run_hook`
reports an unsuccessful external command but does not raise from its return
status. Therefore fixing the shell exit alone is insufficient to guarantee the
renewal systemd unit reflects deployment failure. A per-run hook receipt is used
to make this boundary explicit. E9 preserves the relevant installed source lines.

Hook source of truth previously existed in historical `/root/alma-portal-kit` and
`/root/alma-portal-v2` provisioning material. `operations/tls/` is now the maintained
source; do not rerun historical provisioning over the corrected hook.

## Exact changes

Only these three production code/unit paths were deliberately installed:

| Installed path | Change | SHA-256 after |
|---|---|---|
| `/usr/local/sbin/alma-portal-cert-deploy` | Replace obsolete hook with strict Python entry point; root:root 0755 | `fad1df74eb9800038ec2e9b57b886df73692cd85445fbb9a358df95d51a21403` |
| `/usr/local/libexec/alma-portal-tls` | New stdlib Python renewal/deploy/health wrapper; root:root 0755 | `614a6a55f46a7d71957a3db55af2593403444cb9758b75ac965cbbd7469522ff` |
| `/etc/systemd/system/alma-portal-renew.service` | Call wrapper, 720-second bound, UMask 0077; root:root 0644 | `f22f962fca67f28cd825acca0460bacb969291df7719a8033e9b380280d598e4` |

Original hook SHA-256:
`751858a7bb089e0e94e51d56eadaacea5ab66529a47373e116f6ca1356ae231b`.
Original service SHA-256:
`c417155c12ff5e3b914a2028db88db9b42fc03dcbf030d360415bb8f55198302`.

New persistent runtime state: `/var/lib/alma-portal-tls/` root:root 0700,
`status.json` and `deployment.json` 0600, plus private lock files. It records no
credentials. Certbot normally created/cleaned challenge material and used its
existing logs/accounts. Its own renewal check changed only
`ari_retry_after` from `2026-11-06T09:38:33` to `2026-11-06T21:34:05` and tightened
renewal-config mode from 0644 to 0600. That file's hash changed from
`1709c195237c554ecc88fbe1d9a98b623ef6a87af84714edb0678b89c8b89965` to
`1cb2440080e04102f2ae9d69ed6e39e7d6295e43548bf1164e48ecd653db3231`.
No renewal policy, account, webroot, issuer or certificate path was manually edited.

Unchanged hashes confirm the existing timer, both Apache virtual hosts, Lua access
control, portal common policy, firewall worker and SIP guard were preserved.
Current portal/auth/phone-policy source hashes match existing local source. No
VICIdial PHP/Perl, Asterisk, MariaDB, authorization logic or firewall file was edited.

Local assignment files are under `operations/tls/` (wrapper, hook, unit, deployment
script, collector, evidence verifier, tests and runbook), this report, the PR-02
handoff and `operations/qa/evidence/tls-renewal/`. Git commit IDs and final working
tree are recorded in the operator completion message; no push is authorized.

## Corrected behavior

1. The timer invokes the existing Certbot through the installed renewal wrapper.
   The wrapper serializes runs and records a running/ok/failed status atomically.
2. Certbot retains its own renewal eligibility and ACME configuration. Normal
   no-ops do not run the hook or reload Apache.
3. The hook validates its lineage, trusted certificate chain, IP SAN, validity and
   certificate/key agreement without copying key material. It validates Apache
   configuration, requires active `httpd.service`, then uses `systemctl reload`.
4. Reload success is followed by a fresh trusted TLS handshake and leaf fingerprint
   match. No restart fallback or swallowed important exit code remains.
5. The parent requires the correct successful hook receipt for a changed certificate
   or dry-run exercise. A failed receipt turns even Certbot exit 0 into wrapper
   failure. A subsequent no-op can recover a previously failed/missed deployment.
6. A standalone root health check detects persisted failure, stalled/stale checks,
   timer disablement, certificate mismatch/trust failure and insufficient lifetime.

An expired current certificate does not block calling Certbot for renewal. Post-
renewal acceptance requires at least 24 hours remaining. Certificate validation
and reload failures were injected only in mocks; live invalid configuration or
interruption of Apache was deliberately avoided.

## Certificate before and after

| Property | Before | After |
|---|---|---|
| SAN | IP Address `2.29.153.161` | Same |
| Issuer infrastructure | Existing Let's Encrypt production lineage | Same |
| Valid from | 5 November 2026 19:05:55 UTC | Same |
| Expires | **12 November 2026 11:05:54 UTC** | **Same** |
| Serial | `0608FF6565BEA6657F26AB0FFF3DCBFA379A` | Same |
| SHA-256 leaf | `0f0f23c3a6dca0a5040fe419df141f1b03d515236766486ba2b89abbcec4fd6c` | Same |
| Chain and IP identity | Verified | Verified |
| Served on public IP TCP 443 | Matches disk; trusted TLS 1.3 | Matches disk; trusted TLS 1.3 |
| Served on loopback TCP 443 using public IP identity | Matches disk; trusted | Matches disk; trusted |

The SAN/chain/validity checks used installed OpenSSL and the OS trust store; actual
TLS handshakes used Python's default validating SSL context. No validation bypass
was used. Public-IP probes ran over SSH on the server, not from an independent
off-network browser. No new production certificate serial was issued during this
task: the real due-only checks reported **not yet due**.

## Renewal, deployment and persistence evidence

The staging exercise ran independently under a systemd transient oneshot:

```text
systemd-run --unit=alma-portal-tls-acceptance --property=Type=oneshot --property=TimeoutStartSec=720 --wait /usr/bin/python3 /usr/local/libexec/alma-portal-tls renew --dry-run
```

The installed wrapper supplied `--dry-run --run-deploy-hooks` to Certbot. The journal
reported `Simulating renewal ... for 2.29.153.161`, the real hook's successful
`apache_config`, `apache_active`, `apache_reload`, and `deployment_verified`
stages, then `all simulated renewals succeeded`. Unit result was success/status 0.
Apache's actual `ExecReload` is `/usr/sbin/httpd $OPTIONS -k graceful`.

Dry-run hooks use the current trusted lineage, not the staging certificate. Thus
staging proves supported ACME issuance/challenge and explicitly exercises the
production deploy action, but does not prove rotation to a newly issued production
serial. This is sufficient exercise under the assignment's supported staging
procedure; future normal production issuance remains for the existing timer.

Two `systemctl start alma-portal-renew.service` invocations returned status 0 and
recorded `dry_run=false`, `certificate_changed=false`, `deployed=false`. The
Apache journal contains exactly **one** reload during the entire acceptance
window, from the staging hook. Apache master PID stayed **26973**.

The unchanged timer remains enabled/active/persistent, linked from
`timers.target.wants`. Its own restart succeeded. Schedule: 04:15/16:15 server
local time, randomized delay ≤600 seconds; final sampled next firing
6 November 16:23:11 EST (**21:23:11 UTC**). `TimeoutStartSec=720` bounds renewal
while the wrapper bounds Certbot to 600 seconds. Root-owned installed paths and
persistent state do not depend on an interactive shell. Full host reboot was
**NOT_RUN**; boot configuration and timer-restart evidence support persistence.

## Failure detection and alertability

The transient negative test passed an unexpected `RENEWED_LINEAGE` to the real
hook. It failed before certificate reads/reload, preserving the running TLS service:

```text
Result=exit-code
ExecMainStatus=1
ActiveState=failed
health-check exit=1: renewal or deployment has a persistent failure
```

Successful staging subsequently replaced the failed receipt with a verified
deployment; health then returned 0. The deliberate failed transient unit was reset
after its evidence was saved; final systemd failed-unit inventory is empty.
No live Apache reload failure was induced. Unit tests cover configuration failure,
reload failure, missing receipt, served mismatch, and Certbot hiding a failed hook.

This is **local failure observability beyond a log entry**: a failed systemd result,
persistent JSON outcome and a health executable with a tested nonzero exit. There
is **no accepted external alert delivery**, no approved notification destination
and no independent poller established by PR-08. Supply an approved destination,
recipient/escalation owner and monitoring runner under PR-07. Poll the root health
command and alert on nonzero; do not equate this report with notification acceptance.

Health additionally fails after 30 hours without success, a run stuck beyond 720
seconds, certificate lifetime below 24 hours or a disabled/inactive timer. A
stopped timer cannot notify an operator on its own; independent health polling is
the outstanding operational dependency. State survives reboot; journal retention
remains subject to existing system configuration.

## Regression results

| Check | Result | Evidence |
|---|---|---|
| Apache syntax and HTTPS | PASS | `httpd -t`: Syntax OK; trusted file/served certificate match |
| Graceful operation | PASS | One reload, same Apache master PID; no Apache restart |
| Portal root and login page | PASS | `/` 303 to `/portal/`; `/portal/` 200 before/after |
| UI assets/status protection | PASS | CSS/JS/logo 200; unauthenticated `/portal/status` 401 |
| Unauthorized native entry navigation | PASS | Agent and admin URLs 303 to fixed public root |
| Unauthorized AJAX/resource | PASS | Both entry URLs with AJAX/JSON and a resource request return 403 |
| HTTP ACME restriction | PASS | HTTP `/` 403, missing allowed challenge file 404; real staging HTTP-01 challenge succeeded |
| Apache/Lua/portal policy unchanged | PASS | Installed config/policy hashes preserved; portal/auth/phone sources match existing source |
| Firewall/phone authorization | PASS, bounded | Runtime/permanent `--list-all` identical; both early-gate SIP ports present; worker result 0. No reload/revoke performed |
| Asterisk, MariaDB, portal | PASS, service health | Original master PIDs 143137 / 61646 / 232072 retained, active, no restarts |
| firewalld, Fail2Ban, SSH | PASS | PIDs 142982 / 178535 / 993 retained; SSH 22222 usable throughout |
| SIP/RTP/listeners | PASS, configuration/runtime | Listener bindings unchanged; UDP 5060/5061 still Asterisk; existing source-limited RTP rules unchanged |
| Timer | PASS | Enabled, persistent, future schedule and successful timer restart |
| New authenticated agent/admin/phone tests | NOT_RUN | No credentials, registrations, revocations or calls needed for scoped TLS change; existing accepted policy preserved |
| Full host reboot | NOT_RUN | No reboot authorized/performed; boot enablement inspected |

Apache's pre-existing global ServerName and unset `OPTIONS` warnings remain;
configuration and graceful reload succeeded. They were not changed under PR-08.

## Acceptance matrix

| ID | Criterion | Result | Supporting evidence / boundary |
|---|---|---|---|
| TLS-01 | Current certificate and expiry accurately identified | PASS | E1/E6 file metadata and trusted served TLS |
| TLS-02 | Existing Certbot renewal configuration validated | PASS | Existing 5.9.0 config succeeded in staging and two normal checks |
| TLS-03 | ACME challenge mechanism verified | PASS | Real staging issuance using existing restricted HTTP-01 webroot |
| TLS-04 | Hook references correct active services | PASS | Validates/reloads `httpd.service`; no portal restart |
| TLS-05 | Deployment errors propagate | PASS | E3 real nonzero hook; mocked reload/config/Certbot-swallow failure tests |
| TLS-06 | Renewal/deployment sufficiently exercised | PASS | Real staging issuance **with production deploy hook**, Apache reload and served match; not just a no-op |
| TLS-07 | Apache serves expected deployed certificate | PASS | Hook TLS handshake plus independent collector handshakes to loopback/public IP |
| TLS-08 | Timer enabled and persistent | PASS | Boot link, enabled/active/Persistent=yes, future firing, timer restart; full reboot NOT_RUN |
| TLS-09 | Failure observable | PASS — local scope | Actual failed unit, persistent failed state and nonzero executable check, followed by recovery. External delivery NOT_RUN/unconfigured, PR-07 dependency |
| TLS-10 | Portal/Apache/SSH/firewall regression | PASS — bounded scope | E6/E7/E8; service health, HTTP semantics, unchanged rule sets and policy; no new call/login acceptance claimed |
| TLS-11 | Scoped rollback available/verified | PASS | Isolated archive extraction reproduced hashes/owner/group/mode; script syntax valid. Live rollback NOT_RUN |
| TLS-12 | PR-02 handoff documented | PASS | [Backup/recovery handoff](VICIDIAL_BACKUP_RECOVERY_HANDOFF.md); PR-02 remains open |

These results were executed by the implementation worker. There is no claim of
separate-person independent review, externally delivered alerts, a customer call
test or complete server production readiness.

## Rollback and remaining work

The root-only package is `/root/alma-portal-tls-backup-20261106T203248Z`.
It preserves original hook/service/timer/renewal/Apache files and metadata. No
private key was copied. Archive extraction into a temporary directory reproduced
original hashes, UID/GID and modes before installation. Shell syntax was checked.

Wait for any active renewal/staging/deploy operation to finish, then:

```bash
ssh -p 22222 root@2.29.153.161 \
  '/root/alma-portal-tls-backup-20261106T203248Z/rollback.sh'
```

The script restores only the modified old hook/service, removes the new wrapper,
reloads systemd metadata and checks Apache. It leaves the live certificate lineage,
timer, Certbot-managed renewal metadata, firewall/grants and incident state intact.
It does not restart Apache or the portal. **Rollback reinstates the original
failure-masking behavior**, so it is temporary incident recovery only. A future
actual certificate rollback would need a separate, validated Certbot-lineage
procedure; this task did not replace a production certificate.

Before expiry, monitor the existing scheduled service and confirm the first due
production issuance records `certificate_changed=true`, `deployed=true` and a new
served fingerprint. No forced reissuance is recommended now. Arrange the approved
independent health poll/alert route. Retain the scoped archive securely and complete
the separate PR-02 backup/restore work. The overall production audit's other
findings remain open; they were not remediated by this task.

Operator commands and future controlled staging procedure:
[TLS operations runbook](../tls/README.md).
