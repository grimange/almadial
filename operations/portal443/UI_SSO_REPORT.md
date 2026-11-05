# VICIdial UI integration and supervisor SSO decision

Verdict: **PORTAL_VICIDIAL_UI_SSO_PARTIAL**.

Executed on `dialer01` (`2.29.153.161`) over SSH, 2026-11-05 UTC.
The UI is deployed. The operator subsequently reported **Agent flow passed** and
**Supervisor flow passed; other checks untested**. Supervisor SSO is **not
implemented or verified**. Normal native VICIdial login remains required.

## Implemented changes

The login and authorization pages now use the installed VICIdial logo, the blue
`#015B91` header and pale-blue `#D9E6FE` panels taken from the installed
`admin_header.php` defaults, compact aligned fields, status panels and a responsive
single-column mobile layout. Accessible labels, required-field validation,
visible keyboard focus, skip link, headings and status announcements are included.
No frontend framework or new runtime dependency.

The signed-in page shows username, Agent/Supervisor access type, trusted public
IP, pending/active/failure state, remaining lifetime, absolute UTC expiry, countdown,
fixed destination and Continue Now. The countdown is initially hidden and appears
only after the script confirms active server status; without JavaScript the
server-confirmed Continue Now link and manual refresh remain available.

Only these deployed files changed:

- `/opt/alma-portal-443/portal.py`: integrated markup; exact fixed CSS/logo reads;
  external stylesheet and restrictive self-only style/image CSP.
- `/opt/alma-portal-443/app.css`: new responsive VICIdial styling.
- `/opt/alma-portal-443/app.js`: show countdown/transport error only when relevant.
- `/opt/alma-portal-443/access.lua` and
  `/etc/httpd/conf.d/02-alma-vicidial-ssl.conf`: allow/proxy only the two additional
  exact public asset paths `/portal/app.css` and `/portal/logo.png`.

The logo is read from
`/var/www/html/vicidial/images/vicidial_admin_web_logo.png`, without modifying the
installed artwork or copying it into portal source. No arbitrary file or path
parameter is accepted. All existing VICIdial resource paths remain protected.
No upstream branding license grant is inferred beyond using the installed asset
for this installation.

Existing form authentication, IP-bound session, CSRF/origin checks, rate limiting,
read-only account, queue/worker/firewall logic, TLS, Fail2Ban and service units
are retained. Only the portal backend restarted and Apache reloaded. Active
portal sessions require login again after that restart; root grants remain valid.
The original migration was not rerun and TCP 9091 stays closed.

## Installed authentication investigation and decision

The installed source was inspected without changing it:

| Source | Observed behavior |
|---|---|
| `admin.php:121` | Reads PHP_AUTH_USER and PHP_AUTH_PW. No portal-session or trusted-user-header authentication path found in this entry flow. |
| `admin.php:7968` | Calls `user_authorization`; native database authorization remains authoritative. |
| `admin.php:7974` | General admin entry requires level >7 and non-API user; level 7 report/QC routes are distinct. Granular user/group privileges restrict subsequent operations. `ast_admin_access` is for particular functions, not the general admin entry gate. |
| `admin.php:8043` | Sends HTTP Basic challenge in realm `CONTACT-CENTER-ADMIN`. |
| `functions.php:67` | Native authentication reads password-hash settings, checks active account/password/level and lockout; also applies configured IP allow/deny restrictions. |
| `functions.php:109` | Hash mode invokes the installed `bp.pl` verifier. Current configured mode is plaintext; portal's existing bcrypt compatibility remains tested with fake credentials. |
| `admin.php:8254` | Optional native 2FA checks configured hours/container/user override and nonexpired authentication state, routes an unverified user into the 2FA workflow. No setting or check was disabled. |

Aggregate inventory: one active level 1, one active level 9, two disabled level 1;
no real usernames/passwords are published. Current global 2FA settings are
hours-disabled/container-disabled; enabling or changing them was outside this UI
task. Actual 2FA enforcement with a configured account was **NOT_RUN**.

The existing portal policy matches the general native admin entry gate: active
levels 8–9, API-only excluded and forced-password-change excluded. Levels 1–7
navigate to the agent entry. Supervisor wins deterministically when a user can
use both roles. Native login rechecks the additional account/group/IP permissions;
the portal never fabricates admin privileges or an application session.

Comparison:

- **Browser HTTP Basic reuse:** possible only after a real Basic authentication
  exchange in an applicable protection space. The form POST does not establish
  that exchange. A `/portal/` challenge does not automatically cover `/vicidial/`.
  RFC 7617 section 2.2 defines path scope and optional preemptive reuse; common
  origin alone is not a demonstrated single-login implementation. Changing to a
  root Basic challenge would require separate browser, scope, logout and cached
  credential-leakage qualification. It was not introduced or asserted to work.
  [RFC 7617](https://www.rfc-editor.org/rfc/rfc7617.html#section-2.2).
- **Apache server-side form-to-Basic bridge:** `mod_auth_form` can store login
  details in a session and `AuthFormFakeBasicAuth` can provide username/password
  to the backend. These form/session modules are not loaded here. Browser-cookie
  credential storage conflicts with this assignment; a protected server-side
  credential store/replay integration would require a separately designed
  lifecycle, revocation and 2FA/browser validation. No installed supported
  passwordless session handoff to VICIdial was found. This is a potential future
  integration, not evidence of a tested safe mechanism on this server.
  [Apache mod_auth_form](https://httpd.apache.org/docs/2.4/mod/mod_auth_form.html).
- **Retained implementation:** secure portal form/session plus normal native
  VICIdial Basic authentication. No SSO feature is enabled; no inactive flag or
  simulated identity header is represented as working SSO. A feature flag would
  be required if a real SSO mechanism is later demonstrated and implemented.

**Preexisting credential-disclosure limitation:** `admin.php:8045` interpolates
PHP_AUTH_PW into its failed-authentication response. This was established by source
inspection; no real password was submitted to trigger it. Consequently the
application-wide requirement for no credential leakage in responses cannot be
declared passed. Portal failure responses never include the submitted password,
and the portal does not retain/replay it to native admin. The upstream issue is
left unchanged under the task's restriction on upstream modifications. A bounded
upstream fix and independent review are a separate follow-up; do not test this
failure path using a real password.

## Verification results

| Check | Result / evidence |
|---|---|
| Python authentication/session/worker/HTTP tests | PASS: 14 locally and in staged server venv. Valid/invalid/plaintext/bcrypt, disabled account, levels 8/9, API/forced-change exclusions, CSRF/origin/IP binding, limits/log redaction, role-field escalation, fixed redirects, pending/failure/expiry/stale worker, shared leases/logout. All fixture users are fake. |
| Real JavaScript with mocked transport/timers | PASS: seven cases from `node tests/test_app.js`; pending cannot start countdown, full five active seconds, fixed continuation ignoring destination input, failed/expired/revoked grants stop, nearly-expired grant stops, network failure removes link and explains retry. |
| Syntax | PASS: Python compileall, Node syntax, every shell script checked, `git diff --check`; Apache Syntax OK on server. |
| Layout and installed logo | PASS local browser mock: login and success pages at 320/375/768/1366px, no horizontal overflow, image natural width 170, empty required form invalid, no localStorage/sessionStorage entries. Screenshot review on desktop and mobile; keyboard Tab exposes skip link with 3px visible outline. These are layout checks, not a native password/SSO test. |
| Live logo/CSS/JS and login | PASS on-host TLS probe: exact routes return correct content types; logo markup and restrictive CSP present. User subsequently confirmed both live browser flows. |
| Unauthorized application paths | PASS live `probe.py`: protected PHP/AJAX/assets/recordings/aliases/encoded/Host/header variants, plus attempts to append protected paths to the new CSS/logo routes denied. Server-IP source, not independent internet-source acceptance. |
| Expired/revoked web grant | PASS live fixture: expiry/revocation return 403; fixture cleaned. |
| Real form/CSRF/invalid password | PASS real HTTPS `form_probe.py`: deliberately nonexistent account/fake password receives expected 401, not form/CSRF 403. |
| Portal credential disclosure | PASS scoped checks: fake test password absent from failure response, portal journal and vhost access/error logs. No passwords in generated JS/HTML/URLs/session storage. Real browser passwords were never obtained by the worker. Native upstream exception described above remains. |
| Agent login, grant, countdown and destination | PASS operator: **Agent flow passed** after UI deployment. |
| Supervisor login, UI and destination | PASS operator: **Supervisor flow passed; other checks untested** after UI deployment. Installed active level 9 tested; level 8 covered with mocks. |
| Supervisor opens admin without second prompt | NOT_IMPLEMENTED / NOT_VERIFIED: native second login intentionally retained. Existing Basic cache reuse in an individual profile would not prove portal SSO. |
| Native 2FA | Source unchanged; real enforcement NOT_RUN because currently disabled in existing global configuration. |
| Logout/account revocation | PASS mocked handler/worker tests; prior real shared-lease revocation probe PASS. Actual disabling of a real DB account/browser Basic-cache clearing NOT_RUN; no production user mutation. Eligibility is rechecked by worker and VICIdial. |
| No-JavaScript manual browser continuation | Server markup/handler tests PASS; operator browser test NOT_RUN. |
| Telephony preservation | PASS file/process inspection: upstream PHP/logo, PJSIP/dialplan, firewall/F2B and worker/auth files match before hashes. Asterisk PID 143137/start time unchanged; one PJSIP contact, no active calls. Registration/audio and full agent runtime media NOT_RUN. |

Captured layout evidence uses fake fixture identity/IP and an illustrative fixed
countdown; it does not establish live authorization or SSO:
[mobile login](evidence/mock-login-mobile.png),
[mobile authorized layout](evidence/mock-active-mobile.png),
[desktop authorized layout](evidence/mock-active-desktop.png).

The deployment ended with `UI_DEPLOYMENT_AND_SECURITY_PROBES_PASS`. Current
checksums were verified on host. Apache has only its preexisting global ServerName
warning. Backend/httpd/Asterisk/firewalld/Fail2Ban remain active; authorization
worker Result=success, ExecMainStatus=0; authorization and certificate timers enabled.

Repeatable commands from repository root:

```sh
python3 -m unittest discover -s operations/portal443/tests -v
node operations/portal443/tests/test_app.js
node --check operations/portal443/app.js
python3 -m compileall -q operations/portal443
for file in operations/portal443/*.sh; do bash -n "$file" || exit; done
git diff --check
# Live HTTPS probes, through SSH only; temporary server-IP web fixture:
ssh root@2.29.153.161 '/opt/alma-portal-venv/bin/python /root/alma-portal-ui-stage/probe.py'
ssh root@2.29.153.161 '/opt/alma-portal-venv/bin/python /root/alma-portal-ui-stage/form_probe.py'
```

`worker_probe.py` from the original migration was executed successfully; it briefly
creates test-specific SIP/RTP rules only for the server IP and cleans them. It was
not repeated for this UI-only change because the worker/firewall code is unchanged.
See [EVIDENCE.md](EVIDENCE.md) for migration, renewal, retirement and remaining
external-network/reboot/Cockpit/SELinux acceptance gaps.

## Checksums and rollback

Manifests: [ui-before.sha256](ui-before.sha256), [ui-after.sha256](ui-after.sha256),
[ui-protected.sha256](ui-protected.sha256). The protected manifest was checked
after deployment with every entry OK. All installed portal files match local
source. Archive and rollback remain root-private on the VPS; no credentials or
private backup files are committed.

UI archive `/root/alma-portal-ui-backup/config.tar`, mode 0600:
`cf81ab1c5bd2dda2df2233c4b417c1ef02617aca5d5478945536517907e5cd07`.
UI rollback `/root/alma-portal-ui-backup/rollback.sh`, mode 0700:
`cd4697e4ec33baa468d63810fade97e71fd1f5c799ab579c81eaed9f717af2ee`.
UI staging archive checked at both ends:
`c1d1679624d72ac3e1fd657d16c80cc40945ef9ca5f21aa8514e3d680fea6280`.

```sh
# Restore the preceding working 443 UI/access configuration:
ssh root@2.29.153.161 'bash /root/alma-portal-ui-backup/rollback.sh'
# Full migration rollback to the old protected VICIdial / 9091 portal:
ssh root@2.29.153.161 'bash /root/alma-portal-443-backup/rollback.sh'
```

The UI rollback restores the archived app and 443 vhost, checks Apache syntax,
restarts only the backend, reloads Apache and verifies pre-UI checksums. It leaves
the new unused CSS file on disk, outside the restored exact public route list.
Its backup/archive and script were validated, but UI rollback itself was not
executed against the successful deployment. The original migration rollback was
rehearsed before opening public 443. Do not run UI rollback after full migration
rollback; choose the appropriate backup for the desired state.

Portal: `https://2.29.153.161/`.
Agent: `https://2.29.153.161/agc/vicidial.php`.
Supervisor: `https://2.29.153.161/vicidial/admin.php`.
