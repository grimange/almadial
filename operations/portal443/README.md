# Bounded portal 443 migration

Target `2.29.153.161` only. All remote execution uses SSH from the desktop.
This is an operational upgrade of an existing custom portal, not an AlmaDial
installer stage or a promoted compatibility lock. See BASELINE.md before deployment.

The latest bounded change is SIP revocation and browser navigation redirects;
see [REVOCATION_REPORT.md](REVOCATION_REPORT.md) and
[revocation-evidence/BASELINE.md](revocation-evidence/BASELINE.md).
Use `revoke-deploy.sh` / `revoke-rollback.sh` for that change, with its dedicated
private backup. The original migration scripts below are historical deployment
evidence from commit `0d36667`; do not rerun them against the updated source or
restore their older firewall archive for this task.

The current root worker also maintains `inet alma_portal_sip` at nftables
priority -10, before firewalld's established UDP accept. Timed agent sets enforce
SIP immediately on revocation of the last phone-enabled lease. Only established RTP conntracks marked
during a valid grant have up to 120 seconds of grace. A web-only supervisor cannot
keep SIP/RTP open. Per the operator's later policy correction, supervisors with
native `blind_monitor` permission also receive phone access while keeping their
administration redirect. Each refresh is capped at 35 seconds and the absolute lease
deadline; stale workers fail closed. A required boot unit installs an empty
guard before firewalld/Asterisk. No global conntrack flush or Asterisk restart.
Browser GET navigation to the two exact entry scripts redirects to the fixed
portal root; all other unauthorized protected requests retain 403.

Architecture:

- Apache terminates the existing IP-address certificate and proxies only the exact
  root and `/portal/` endpoints to `127.0.0.1:19091`. Its Lua authorization provider
  protects every other HTTPS URL, including aliases, supporting files and PHP.
  It compares the request IP with the underlying connection IP and reads root-owned
  expiring files. HTTP permits only static ACME challenge tokens.
- Browser login is form-based against the existing column-limited, read-only DB
  account; the original verifier is preserved byte-for-byte. No VICIdial user data or schema change; the phone policy adds column-only
  SELECT access to two native monitoring-permission fields. Active levels 1–7 navigate to agent; active non-API levels 8–9 navigate
  to administration. Existing forced-change exclusions remain. Supervisor wins
  when both uses are possible. VICIdial enforces its own login and privileges.
- Portal cookies are Secure, HttpOnly, SameSite=Strict, Path=/portal/. Apache strips
  Authorization only on portal routes; VICIdial's own Basic auth remains usable.
  Portal and HTTPS access logs omit credentials, tokens, query strings and Referer.
  CSRF tokens protect every action; status and continue require the IP-bound session.
- Unprivileged queue -> root timer -> live eligibility recheck -> runtime-only
  timed SIP/RTP rules -> query confirmation -> root-owned status and web expiry.
  An HTTP redirect after queueing means only that the status page should be shown.
  The five-second countdown and Continue Now require confirmed status. Every
  countdown tick and server continuation recheck expiry. No user-controlled URL.
- Each lease is independent. Web uses the latest eligible lease for an IP; SIP/RTP
  uses agent leases and eligible supervisors with native monitoring permission.
  Sign-out revokes every lease for that account on that IP, including older
  browser sessions; it preserves other users and IPs. Tombstones reject stale
  queued grants even if queue processing order differs. Shared IPs share
  network reachability, never VICIdial identity or application permissions.
- Web freshness expires after 35 seconds without a worker confirmation. SIP rules
  have native firewalld timeouts capped at their absolute lease expiry. The worker
  persists ownership before effect, reconciles after firewalld reload/reboot,
  and touches only priority=10 rules it owns. No permanent SIP/RTP grants remain for the residential test client. The early
  gate also denies protected UDP when an unrelated downstream accept exists.
  Authorized traffic still traverses firewalld and its existing Fail2Ban rules.
- Backend sessions are memory-only; a backend restart requires portal login again.
  Existing root-owned grants survive a service restart until expiry/revocation.
  Logout queues revocation; web revocation takes the next worker tick (normally
  10 seconds). A failing worker causes web denial within the freshness window;
  The early SIP gate expires within 35 seconds of its last successful refresh;
  only previously marked RTP flows can continue for a further 120 seconds. No instant-revocation
  guarantee is made during firewalld/worker failure.

Validation:

```sh
python3 -m unittest discover -s operations/portal443/tests -v
python3 -m compileall -q operations/portal443
for file in operations/portal443/*.sh; do bash -n "$file" || exit; done
node --check operations/portal443/app.js
node operations/portal443/tests/test_app.js
```

After activation, run `probe.py`, `form_probe.py`, and `worker_probe.py` with the
server's existing Python environment through SSH. The first exercises Apache's
boundary with a temporary web-only fixture for the server IP; the second submits
deliberately fake credentials through real HTTPS; the third uses eligible account
names from the read-only connection, without passwords, to test actual queue,
firewall confirmation, shared-IP leases, revocation and ineligible requests. These
probes refuse to overwrite an existing server-IP grant. None establishes an actual
agent password login, browser behavior or media. See EVIDENCE.md for executed results.

The local test runner uses a mock DB driver if PyMySQL is absent. On the server it
uses the existing installed bcrypt 4.3.0/PyMySQL 1.2.3 environment. No dependencies
are installed. The verifier was obtained from the authorized server; its origin
hash is recorded in deployment evidence, not represented as a licensed upstream
library. Upstream PHP, DB, Asterisk configuration and working workers are untouched.

Deployment order (scripts must be copied over SSH into root-only staging):

1. `prepare.sh`: refuse an existing backup directory; archive configs with private
   root permissions, record checksums, install/start loopback service, run tests.
2. `activate.sh`: install Apache boundary, syntax-check, reload, run `probe.py`,
   update portal Fail2Ban jail, enable worker, stop old worker, open public 443.
   A failing step invokes rollback. The legacy portal remains available until
   browser verification. Automatic rollback of unit preparation is not implied.
3. Operator browser checks below; then `retire-9091.sh` closes runtime/permanent
   9091 and disables its service. The old files and rollback archive remain.
4. Re-run probes and health inventory, record checksums and exact verdict.

Current task rollback via SSH on the operator-updated port:

```sh
ssh -p 22222 root@2.29.153.161 'bash /root/alma-portal-revoke-backup/rollback.sh'
```

This restores the pre-revocation 443 portal and its known UDP conntrack
limitation. It preserves the operator's SSH configuration and the rotated portal
DB service secret. Do not invoke the original migration rollback for this task:
its historical archive predates subsequent firewall and SSH changes.

Operator browser checks (use existing credentials privately; never send passwords):

1. Fresh browser profile: `https://2.29.153.161/` has trusted TLS, shows the heading
   and your actual public IP. Before authorizing, `/agc/vicidial.php`,
   `/vicidial/admin.php`, `/agc/vdc_db_query.php`, `/recordings/` are denied.
2. Sign in as an agent, authorize, observe pending then confirmed access, displayed
   IP/type/lifetime, full five-second countdown and redirect to `/agc/vicidial.php`.
   Sign in to VICIdial normally. Confirm CSS/AJAX/images, agent conference and
   existing Zoiper registration/audio without changing phone/provider settings.
3. Sign in as supervisor in another profile; confirm `/vicidial/admin.php` redirect
   and native application permissions. No automatic VICIdial credential forwarding.
4. Disable JavaScript: queue, manually refresh, confirm Continue Now appears only
   after activation and reaches the fixed intended destination.
5. Two users sharing an IP: revoking one must not revoke the other. When the final
   lease is revoked, confirm new application requests are denied after a worker
   tick. Existing static SIP exceptions do not test dynamic SIP expiry.
6. After retirement, confirm TCP 9091 fails from an independent internet client.
   A listener/firewall inspection is not independent external verification.

Do not reboot an operational dialer merely to fill an acceptance box. Boot-unit
configuration inspection and service restarts do not prove reboot persistence.
Record unexecuted real-user/media/reboot checks as NOT_RUN, and use PARTIAL unless
every mandatory acceptance check is actually evidenced.

The subsequent VICIdial visual update is deployed. See [UI_SSO_REPORT.md](UI_SSO_REPORT.md)
for its separate backup, exact file changes, browser evidence and SSO decision.
`ui-deploy.sh` updates an existing 443 deployment; it must not be confused with
the original migration scripts. The installed VICIdial logo is read through a
fixed public portal route; its artwork is not copied into this source tree.
Supervisor SSO is not implemented. Native VICIdial authentication and optional
2FA remain authoritative. Portal logout revokes its account's grants on the current IP and its session; it does
not clear credentials cached by a browser for native HTTP Basic authentication.

References: [Apache Lua authorization providers](https://httpd.apache.org/docs/2.4/mod/mod_lua.html),
[Apache configuration-section merge order](https://httpd.apache.org/docs/2.4/sections.html).
