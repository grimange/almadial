# Unified HTTPS portal deployment evidence

Verdict: **PORTAL_443_ROLE_REDIRECT_PARTIAL**.

Target: `2.29.153.161`, hostname `dialer01`. Deployment and checks executed
2026-11-05 UTC (desktop timezone Asia/Manila is already November 6).
All remote execution and file transfer used SSH from the desktop.
This is an operational change to the named installation, not an accepted AlmaDial stack.

## Architecture and changes

Apache serves the existing trusted IP certificate on public TCP 443. Only `/`
and exact `/portal/` action/asset paths reach the unprivileged backend on
`127.0.0.1:19091`. All remaining HTTPS paths require a fresh root-owned IP lease
through the Lua authorization provider. That boundary covers VICIdial PHP,
AJAX, styles, scripts, images, recordings, inherited aliases and alternate paths.
Host validation and connection-IP comparison reject spoofed forwarding headers.
Port 80 permits only static ACME challenge tokens, including through inherited
aliases; the AlmaLinux welcome page is denied.

Form authentication uses the existing column-restricted read-only VICIdial DB
account and original plaintext/bcrypt verifier. No DB user, schema, upstream PHP,
dialplan, PJSIP or Asterisk process change. Secure/HttpOnly/SameSite=Strict cookies
are restricted to `/portal/`; all mutations validate CSRF and origin. Apache strips
cached Basic credentials from portal requests. Rate limiting and the existing
Fail2Ban journal integration are retained, with the portal jail now on TCP 443.

Active, non-API users without forced password change retain the original policy:
levels 1–7 go to `/agc/vicidial.php`; levels 8–9 go to `/vicidial/admin.php`.
The installed native admin authentication and permissions were inspected before
retaining that policy. Supervisor destination takes precedence; the portal grants
network reachability, not application privileges. Native login remains required.

The root timer rechecks account eligibility, reconciles its own source-specific
timed UDP 5061/RTP 10000–20000 rules, queries confirmation and publishes status.
Queued/202 does not mean active. Status and continuation require the portal
session. Only confirmed grants start the five-second countdown; continuation
rechecks the server and chooses a fixed destination. Leases last at most 12h.
Each lease is independent; web access uses the latest valid shared-IP lease,
while SIP/RTP requires an agent lease. Web freshness fails closed after 35s
without worker confirmation. Firewall rules retain native absolute timeouts.

Installed files are recorded in [after.sha256](after.sha256), with the
pre-change hashes in [before.sha256](before.sha256). That post-migration manifest
predates the subsequent UI update: use [ui-after.sha256](ui-after.sha256) for
current portal Python/JS/CSS/Lua and the 443 vhost. Other migration files remain
as recorded. Local commits `0d36667` and `fba8f6f` contain migration and browser
Origin fixes; the follow-up commit contains UI, tests and these reports. No push.

## Executed verification

| Check | Result and evidence |
|---|---|
| Baseline before mutation | PASS: [BASELINE.md](BASELINE.md), root-private configuration archive and before hashes. |
| Staged preparation and rollback | PASS: loopback backend tested before public migration; original rollback rehearsed, original checksums restored before activation. |
| Python automated suite | PASS: 12 migration tests; expanded to 14 for the UI, run locally and in the server venv. Mock users/DB/firewall explicitly distinguished from live probes. |
| Apache / systemd / Fail2Ban syntax | PASS: `httpd -t` Syntax OK, `systemd-analyze verify` new units, `fail2ban-client -t`; preexisting global ServerName warning remains. |
| HTTPS trust and public-IP SAN | PASS: real TLS verification in `probe.py`; Let's Encrypt YE1, validity Nov 5 19:05:55–Nov 12 11:05:54 UTC 2026. |
| Unauthorized web boundary | PASS on host: default-deny PHP, application/supporting paths, recordings, aliases, dot/encoded/traversal paths, alternative Host values and spoofed source headers. |
| Authorized web boundary | PASS on host: temporary web-only lease for server's own IP reaches native agent/admin login pages and agent CSS/JS/images; expiry and revocation return 403. Not a password login. |
| HTTP and ACME | PASS: root/application/aliases denied; an actual temporary ACME token returns 200 and is removed. |
| Form transport | PASS: `form_probe.py` through real Apache/TLS/session/CSRF reaches expected 401 for deliberately fake credentials. Browser Origin bug fixed with same-origin Referrer-Policy; no relaxation of CSRF/origin checks. |
| Real authorization worker/firewall | PASS: `worker_probe.py`, read-only eligible account lookup without passwords; real queue/account recheck, two timed SIP/RTP rules queried before active status, root-owned web lease and ~12h deadline. |
| Shared-IP/revocation/failure | PASS live: two agent leases plus one supervisor lease; revoking one agent preserves remaining agent and supervisor; last agent revocation removes SIP/RTP but preserves supervisor web; final revocation removes web; ineligible request fails closed. Test rules/leases cleaned. |
| Redirects in browser | PASS operator reported after UI deployment: agent flow and supervisor flow, including confirmed access, expiry/countdown and intended destinations. Not independent security QA. |
| Renewal | PASS: `/opt/alma-portal-certbot/bin/certbot renew --dry-run --no-random-sleep-on-renew`; existing twice-daily timer enabled and webroot/deploy hook preserved. |
| 9091 retirement | PASS on host: old portal disabled, no listener, runtime/permanent 9091 removed; public ports are 22/tcp 80/tcp 443/tcp. Independent internet reachability test NOT_RUN. |
| Existing firewall preservation | PASS: permanent public-zone XML semantic comparison changed only removal of 9091 and addition of 443. Static source SIP/RTP allowances, other services and Fail2Ban retained; no global SIP/RTP/FastAGI opening. |
| Services / telephony process health | PASS inspection: httpd, backend, firewalld, Fail2Ban, Asterisk active; worker Result=success/ExecMainStatus=0; timers enabled. Asterisk PID 143137/start time unchanged; eight existing AST_script processes observed, zero active calls, one PJSIP contact in final inventory. |
| Native agent audio / Zoiper two-way media | NOT_RUN: registration object/contact inspection does not prove browser-phone or Zoiper media. |
| Cockpit | Preexisting limitation: public-zone cockpit service allowed, but no 9090 listener or cockpit.socket. Configuration preserved; operational acceptance cannot pass. |
| SELinux | Baseline Permissive unchanged; no recent AVC matches at baseline. Enforcing-mode validation NOT_RUN. |
| JavaScript-disabled manual workflow | PASS server-rendered/handler tests; real browser workflow NOT_RUN. |
| Reboot persistence | Units and permanent config inspected; backend restart exercised. Actual reboot/firewalld restart NOT_RUN to avoid disrupting telephony. |
| Independent unauthorized internet source | NOT_RUN: server-bound source probes do not establish an independent external client result. |

Commands and interpretation are documented in [README.md](README.md). The live
firewall test uses only the server's own source IP and refuses an existing grant.
No real password was captured or stored by the probes. False-success and worker
failure, invalid/disabled authentication, bcrypt, CSRF, IP binding, rate limit,
role-field escalation, expiry and fixed redirects have automated coverage.
Do not treat mocked disabled-user/account-revocation tests as production DB changes.

## Backups, rollback and limitations

Root-private migration backup: `/root/alma-portal-443-backup/config.tar`, mode 0600,
SHA256 `2e2c430172ab822b07a49c786cad7028a88c4bbbb0a1e1813f9188c32bd0adde`.
Rollback script SHA256 `874468e01c89dbc8d4e996fd0bf5ec21c51d530c0698de531c13e6c5ffe54268`.
Archives may contain private configuration and were not downloaded or committed.

```sh
ssh root@2.29.153.161 'bash /root/alma-portal-443-backup/rollback.sh'
```

This closes public 443 first, restores the previous source-firewall-protected
VICIdial configuration and 9091 portal, restores portal Fail2Ban settings and
starts the original worker. Existing SSH and unrelated firewall rules are retained.
New owned SIP rules expire naturally. No Asterisk restart. Recovery console was
confirmed available; snapshot status was not established.

The later UI-only rollback is separate; see [UI_SSO_REPORT.md](UI_SSO_REPORT.md).
Existing static SIP/RTP exceptions remain outside dynamic expiry. During worker
failure SIP access may remain until the already-installed firewall timeout;
web access denies within its freshness deadline. Normal logout revocation is
asynchronous, normally the next 10s worker tick. Native Basic credentials cached
by the browser are not cleared by portal logout. Supervisor SSO is not implemented.
The follow-up report documents a preexisting native admin failure-response
password reflection, making application-wide credential non-disclosure unverified.

Public portal: `https://2.29.153.161/`.
Agent destination: `https://2.29.153.161/agc/vicidial.php`.
Supervisor destination: `https://2.29.153.161/vicidial/admin.php`.
