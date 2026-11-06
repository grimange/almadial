# Shared Office Access operator guide

Scope: the existing portal at https://2.29.153.161/ on `dialer01`.
SSH uses TCP **22222**. This integration does not change native VICIdial login,
2FA, account permissions, SIP credentials, or supervisor SSO.

## Use

- **My Access — 12 hours:** available to existing eligible active accounts.
  Agent phone access and supervisor monitoring phone permissions are unchanged.
- **Shared Office Access — 24 hours:** available to active portal supervisors
  (level 8/9, not API-only, no forced password change), with `modify_users=1`
  and the existing `vdc_agent_api_access=1` / `blind_monitor` or `ALL_FUNCTIONS`
  permission. Native `admin.php` uses `modify_users` for user management;
  `non_agent_api.php` uses the monitoring permissions for phone monitoring.
  Numeric level alone does not permit office grants.

Sign in, choose the option, and wait for **Access granted**. The existing
five-second countdown and Continue Now use the account's usual destination:
`/agc/vicidial.php` for agents, `/vicidial/admin.php` for supervisors.
With JavaScript disabled, refresh until confirmed, then use Continue Now.

Office access covers only the issuer's server-verified current public IPv4.
Devices sharing it can reach VICIdial, UDP 5060 (legacy SIP), UDP 5061 (PJSIP)
and RTP 10000–20000 without additional portal logins. Each still needs its own
VICIdial and SIP credentials.
Network grants **cannot isolate different users behind the same NAT address**.
No SSH, AMI, FastAGI, Cockpit or other management permission is added.

Open `/portal/` and sign in again to view your own active office grant and its
expiry. Viewing grants does not start a countdown. Use **Revoke Shared Office
Access** to revoke it, or **Continue with Shared Office Access** to navigate.
**Revoke My Access & sign out** revokes only your personal grants on this IP;
your office grant continues until separately revoked or expired. A separate
Revoke My Access action keeps the portal session open.

Repeating authorization selects the current grant without extending its expiry.
After expiry, a fresh request creates a new full-duration grant. Revocation is
account/IP/type scoped and includes older sessions for that same type; other
accounts and the other grant type remain independent. Firewalls use the latest
expiry of the remaining eligible grants on each IP, with three UDP rules per IP.
The worker rechecks identities and office permissions each cycle. Browser
sessions still expire after 12 hours; office grants persist for 24 hours and can
be viewed/revoked after signing in again.

Worker processing normally takes up to 10 seconds. Both native firewall and
nft confirmation are required before access is reported active. A stale worker
fails closed through the existing 35-second freshness cap. Last-grant revocation
blocks SIP, including existing SIP conntrack flows, on reconciliation; only
previously authorized established RTP can continue for the existing maximum
120-second grace. Active calls can lose media after that grace. Other valid
shared-IP grants preserve access. Repeated revocation does not extend grace.

## Deployment and recovery

Task scripts: `office-deploy.sh`, `office-rollback.sh`. The private pre-change
backup is `/root/alma-portal-office-backup/config.tar`; never export it because
it contains authorization state. Root-only checksums and raw firewall baseline
are beside it. Deployment installs only `common.py`, `phone_policy.py`,
`apply_requests.py`, `portal.py`, `app.css` in `/opt/alma-portal-443` and adds
column-only SELECT on `vicidial_users.modify_users` to the existing read-only
portal DB account. No table/schema/user-data changes occur.

Rollback from the desktop:

```bash
ssh -p 22222 root@2.29.153.161 /root/alma-portal-office-backup/rollback.sh
```

Rollback stops the affected portal services, revokes office leases and drains
queued office requests using the current reconciler, restores the five prior
source files, removes the added SELECT permission, and restarts the portal and
worker. It preserves current personal grants, their revocations, firewall/SSH
configuration and the verified SIP guard. It deliberately does not restore an
old authorization-state snapshot or old firewall XML. Office authorization will
end on rollback. Keep an SSH session/recovery console available.

## Repeatable checks and acceptance limits

```bash
python3 -m unittest discover -s operations/portal443/tests -v
node --test operations/portal443/tests/test_app.js
bash -n operations/portal443/office-{deploy,rollback}.sh
```

`office_probe.py` runs on the server through SSH with the existing venv. It uses
real DB eligibility, the real worker/firewall/nft and trusted HTTPS, but only a
server-IP fixture; it never uses a password. It proves native login requirements,
aggregate permissions, scoped revocation, retry behavior and accelerated expiry.
`probe.py` checks unauthorized browser redirects, API/resource denial, source
spoofing, TLS, ACME routing and native web resources. `guard_network_probe.py`
uses isolated namespaces for UDP/conntrack/grace/expiry regression checks.
These probes do not replace independent browser and actual SIP REGISTER tests.
Legacy SIP deployment/acceptance and its task-specific rollback are documented in
`LEGACY_SIP_5060_REPORT.md`. After that extension, use its rollback for SIP5060
changes; do not use an older full office rollback for the later SIP policy.

Independent checks needed for office acceptance:

1. Agent sees only My Access; sign in/authorize, confirm expiry near 12h,
   countdown and agent destination. Revoke it when no longer needed.
2. Eligible supervisor sees both options. Authorize Shared Office Access,
   confirm expiry near 24h, IP/issuer/type and administration redirect.
3. In a fresh browser behind that same IP, open the native agent/admin URLs
   without portal login. Native VICIdial credentials must still be required.
4. Force a fresh Zoiper registration with valid SIP credentials on that network.
   It should succeed. An external IP with no grant must remain blocked.
5. Coexistence: issue personal plus office access; revoke one, confirming fresh
   registration still works through the other. Revoke the last applicable grant,
   wait 15 seconds, and confirm fresh registration fails and native browser
   navigation returns to the portal. Other accounts' independent grants can
   legitimately keep the IP authorized; coordinate their revocation with owners.
6. Check desktop/mobile layout, manual continuation without JavaScript, and
   existing audio/monitoring if practical. Do not send passwords or SIP secrets.

See `OFFICE_REPORT.md` and `office-evidence/` for executed results and limits.
