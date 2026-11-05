# Bounded portal 443 migration

Target `2.29.153.161` only. All remote execution uses SSH from the desktop.
This is an operational upgrade of an existing custom portal, not an AlmaDial
installer stage or a promoted compatibility lock. See BASELINE.md before deployment.

Architecture:

- Apache terminates the existing IP-address certificate and proxies only the exact
  root and `/portal/` endpoints to `127.0.0.1:19091`. Its Lua authorization provider
  protects every other HTTPS URL, including aliases, supporting files and PHP.
  It compares the request IP with the underlying connection IP and reads root-owned
  expiring files. HTTP permits only static ACME challenge tokens.
- Browser login is form-based against the existing column-limited, read-only DB
  account; the original verifier is preserved byte-for-byte. No account/schema
  change. Active levels 1–7 navigate to agent; active non-API levels 8–9 navigate
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
  uses only agent leases. Revoking one lease preserves others. Shared IPs share
  network reachability, never VICIdial identity or application permissions.
- Web freshness expires after 35 seconds without a worker confirmation. SIP rules
  have native firewalld timeouts capped at their absolute lease expiry. The worker
  persists ownership before effect, reconciles after firewalld reload/reboot,
  and touches only priority=10 rules it owns. Existing static telephony rules
  remain explicit exceptions. Fail2Ban reject rules take precedence.
- Backend sessions are memory-only; a backend restart requires portal login again.
  Existing root-owned grants survive a service restart until expiry/revocation.
  Logout queues revocation; web revocation takes the next worker tick (normally
  10 seconds). A failing worker causes web denial within the freshness window;
  SIP rules may remain until their existing native timeout. No instant-revocation
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

Rollback via existing SSH/recovery console:

```sh
ssh root@2.29.153.161 'bash /root/alma-portal-443-backup/rollback.sh'
```

Rollback closes public 443 before restoring the source-firewall-protected v2
Apache config, opens 9091, restores Apache and Fail2Ban portal configuration,
and starts the original portal/worker. It preserves unrelated firewall settings.
New runtime SIP rules expire naturally; no Asterisk restart is involved.

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
2FA remain authoritative. Portal logout revokes its lease and session; it does
not clear credentials cached by a browser for native HTTP Basic authentication.

References: [Apache Lua authorization providers](https://httpd.apache.org/docs/2.4/mod/mod_lua.html),
[Apache configuration-section merge order](https://httpd.apache.org/docs/2.4/sections.html).
