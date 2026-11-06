# Shared Office Access acceptance — 2026-11-06

**PORTAL_SHARED_OFFICE_ACCESS_VERIFIED**

The existing live portal now offers My Access for 12 hours and Shared Office
Access for 24 hours. Office granting, effective web/SIP firewall authorization,
both coexistence revoke orders and last-grant denial were demonstrated through
live probes plus independent operator browser and fresh Zoiper registration
tests. This verdict applies to this bounded integration, not AlmaDial installer
or stack compatibility acceptance.

Portal: https://2.29.153.161/ . Existing destinations remain
https://2.29.153.161/agc/vicidial.php and
https://2.29.153.161/vicidial/admin.php . SSH remains on TCP22222.

## Implementation and behavior

Before: eligible users could issue and revoke personal 12-hour network grants.
After: personal access is retained; eligible supervisors can additionally issue
an independent 24-hour office grant on their server-verified current public IP.
The existing VICIdial-style page displays issuer, IP, type, status and expiry,
and starts its five-second redirect only after worker confirmation. Issuers can
view/revoke their own office grant after signing in again.

Office eligibility requires the existing active level8/9 supervisor policy,
`modify_users=1`, and the existing monitoring permissions
(`vdc_agent_api_access=1` plus `blind_monitor` or `ALL_FUNCTIONS`). API-only,
forced-password-change, inactive and ordinary agent accounts cannot issue office
grants. Numeric level alone is insufficient. Eligibility is rechecked by the
root worker. Only column SELECT on `vicidial_users.modify_users` was added to
the existing read-only portal DB account; no schema or user-data change occurred.

The existing queue, atomic state, Apache Lua boundary, timed firewalld rules and
nft SIP guard are reused. Grants have `personal` or `office` scope and stable
issuer/IP/type IDs, issued/expiry timestamps, services and revocation state.
Legacy grants remain personal with their original expiry. Repeated authorization
acknowledges the request without extending an active grant; fresh authorization
after expiry creates a new lease. Scoped revocation and tombstones reject stale
queued requests. Effective web/phone access uses the remaining valid grants on
the IP, with two UDP rules per authorized phone IP rather than per office user.

Each office user still needs native VICIdial credentials and each phone its SIP
credentials. SSO remains deferred. Office grants do not permit SSH, Cockpit,
AMI, FastAGI or arbitrary IP/range entry. Network authorization cannot isolate
users sharing one NAT address. Portal sessions remain 12 hours; an office grant
lasts 24 hours and can be managed after signing in again.

Last-grant removal blocks SIP, including existing conntrack flows, after worker
reconciliation (normally within 10 seconds). Previously authorized established
RTP retains the existing maximum 120-second grace; active calls may lose media
after that grace. New RTP is denied. Worker freshness remains capped at 35
seconds and failures do not produce false Access granted status.

## Exact changes and commits

Implementation commits: `3cadd31` (office grants/UI/policy/tests/deployment) and
`a7a8d96` (retry confirmation without extending expiry). Initial HEAD: `e99be47`.

Only these installed source files changed under `/opt/alma-portal-443/`:

- `common.py` — durations and grant identity.
- `phone_policy.py` — office eligibility from existing native permissions.
- `apply_requests.py` — typed leases, scoped revocation, idempotent processing.
- `portal.py` — selection, status, own-grant view/revoke and confirmation handling.
- `app.css` — compact option cards using the existing responsive styling.

Additional repository files changed/added under `operations/portal443/`:
`tests/test_office.py`, `tests/test_portal.py`, `worker_probe.py`,
`office_probe.py`, `office-deploy.sh`, `office-rollback.sh`, `office_evidence.py`,
`sanitize_office_evidence.py`, `OFFICE_ACCESS.md`, `OFFICE_BASELINE.md`, this report
and `office-evidence/` (test logs, operator results, source hashes and sanitized
server evidence).

Before/after SHA256 values are retained in
`office-evidence/sanitized-office-evidence.json`; local installed-source matches
are in `office-evidence/modified-source.sha256`. Ten control files are unchanged,
including native VICIdial PHP, PJSIP/dialplan, Apache/Lua, firewall XML, credential
verifier, SIP guard and Fail2Ban configuration. No VICIdial PHP/Perl or MariaDB
schema was modified. No Asterisk, MariaDB, Apache, firewalld, Fail2Ban or SSH
restart was needed. Only affected portal services/timer were restarted.
Unrelated local work was neither staged nor changed.

## Executed tests

| Check | Result and evidence |
| --- | --- |
| Local Python suite | PASS, 36 tests; mocked users/clock/firewall plus real loopback HTTP handler. `office-evidence/local-python-tests.txt`. Inherited HTTP regressions are included in this count. |
| Staged remote Python suite | PASS, 36 tests before activation; `staged-tests.txt` in sanitized bundle. |
| JavaScript | PASS, seven countdown/poll/error/destination cases; `office-evidence/local-js-tests.txt`. |
| Deployment scripts, Python and units | PASS, shell syntax, Python compilation/imports, `systemd-analyze verify` on portal/worker/guard units. |
| Office integration probe | PASS, nine live checks using real account eligibility, worker, firewall/nft and trusted HTTPS with a server-IP fixture. `office-probe-final.txt`. No SIP password/REGISTER was used by this probe. |
| Personal regression probe | PASS, seven live checks including idempotence, independent issuer preservation and stale-queue rejection; `personal-probe-final.txt`. |
| HTTPS boundary probe | PASS, nine checks: trusted IP TLS, public portal/logo, fixed browser redirect, API/AJAX/resource403, spoofed headers/Host, encoded paths, native resources, expired/revoked denial and ACME-only HTTP; `https-probe.txt`. |
| SIP/conntrack regression | PASS, eleven isolated namespace UDP checks for priority gate, established SIP revocation, native expiry and established-only RTP grace; `guard-probe.txt`. These packets are not actual PJSIP registration tests. |
| Independent office browser and SIP | PASS — operator confirmed near24h/type/IP, five-second admin redirect, fresh browser native login requirement and successful forced fresh Zoiper registration. |
| Independent coexistence and last revoke | PASS — operator confirmed both revoke orders, then fresh registration blocked and both native entry URLs redirected to the portal after final removal. Exact replies: `office-evidence/operator-results.md`. |
| Rollback | PASS, executed before deployment and again with an active office fixture. Office access drained, current personal grants/revocations preserved, all15 baseline hashes restored. Final implementation redeployed and probes rerun. |
| Repository checks | PASS, `git diff --check`; scaffold validator (11 required files, four skills). These checks do not establish runtime compatibility. |

Policy/HTTP tests cover inactive/invalid credentials, office role escalation,
forged IP/duration input, session/CSRF/rate controls, fixed agent/supervisor
destinations and failed-worker behavior. Lease tests/probes cover exact
43200/86400-second lifetimes, both revoke orders, other issuers, idempotence,
accelerated expiry, stale requests and reauthorization. No real passwords were
requested, stored or added to evidence. Existing native authentication and 2FA
handling are unchanged; a new independent 2FA exercise was not performed.

## Firewall and service evidence

During operator office authorization, aggregate evidence showed one office and
one personal grant across two IPs, four expected UDP rules present and two nft
phone members. After operator final revocation:

- Tested office network: zero effective web/phone grants, zero native web TTL
  files, zero nft phone members and zero portal UDP allow rules.
- Unrelated network: one valid personal grant remains, with its two UDP rules
  and one nft phone member. Effective-rule verification passed.
- No permanent rich grants. nft input/output hooks remain priority-10; source
  firewall XML is byte-identical. Unrelated runtime rules are preserved.
- Portal, apply timer, SIP guard, Apache, Asterisk, MariaDB, SSH, firewalld and
  Fail2Ban are active. Asterisk PID143137 is unchanged; no calls were active at
  collection. SSH22222 listens, legacy9091 does not. Cockpit9090 has no listener,
  unchanged from baseline.
- Apache syntax validates. TLS trust/IP match passes; current certificate expires
  2026-11-12 11:05:54 UTC. Renewal timer remains enabled/active with last renewal
  service result success; ACME routing passes. No new issuance/dry-run was forced.

The final health comparison to the initial lease snapshot is false because the
operator deliberately revoked a baseline personal grant during acceptance.
Deployment/fixture cleanup probes separately asserted preservation before those
operator actions. The earlier active-office snapshot's permanent-rule count of
one is a collector blank-line counting defect; corrected final collection reports
zero. It is retained transparently rather than rewritten.

Server logs/checksums are packaged in
`office-evidence/sanitized-office-evidence.json`. Raw authorization state,
firewall listings and backups remain private on the server. Automatic approval
review rejected a detailed client-state collector and raw evidence download;
the successful replacement exports only sanitized aggregate summaries with IPs
and absolute server paths removed. No requested acceptance work remains blocked.

## Limits and rollback

SELinux was already **Permissive** and is unchanged. This is a pre-existing host
security limitation; enforcing-mode compatibility is NOT_RUN. Full24h elapsed
expiry, reboot, new audio/barge monitoring, mobile and JavaScript-disabled browser
checks are NOT_RUN in this assignment. Expiry was tested by advancing only the
fixture lease expiry and by actual native short-timeout tests. Existing agent
and supervisor browser acceptance from the previous task remains the baseline;
current role destinations also have HTTP regression tests.

Root-only pre-change backup: `/root/alma-portal-office-backup/config.tar` with
`before.sha256`. Tested rollback command from the desktop:

```bash
ssh -p 22222 root@2.29.153.161 /root/alma-portal-office-backup/rollback.sh
```

It drains office grants/requests, restores the five previous source files and
removes only the added DB column privilege. Current personal grants and their
revocations survive. It does not restore stale authorization state or overwrite
firewall/SSH configuration. All office grants end on rollback. Keep SSH/recovery
console available. Usage and repeatable test commands: `OFFICE_ACCESS.md`.
