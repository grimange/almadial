# SIP revocation and browser redirects — operator report

**PORTAL_REVOCATION_REDIRECT_VERIFIED**

Target: `dialer01`, `2.29.153.161`. Implementation commit: `1afdd6f`.
Final server evidence: **2026-11-05 23:18:19 UTC / 2026-11-06 07:18:19 Asia/Manila**.
All remote operations used SSH/SCP from the desktop. The operator changed SSH to
**22222** during the work; that change is preserved. No push was performed.
This verdict covers the bounded SIP revocation and browser redirect assignment;
it does not certify audio, barge, SSO, reboot testing, or an AlmaDial stack lock.

## Result and architecture

The original bypass was an established UDP conntrack accepted by firewalld before
its source-IP zone rules. Removing a timed allow rule alone could not reliably
block that tuple. Asterisk contact retention was inspected separately and was
never treated as proof of a fresh successful REGISTER.

A portal-owned nftables table, `inet alma_portal_sip`, now checks input and output
at priority **-10**, before firewalld's priority **+10** established/related accept.
It drops UDP5061 without a current phone grant, including previously established
tuples. Authorized packets return to the remaining firewall chains, preserving
firewalld and Fail2Ban enforcement. It also gates UDP10000–20000 and denies these
protected UDP ports for IPv6, for which the portal issues no grants. It does not
flush conntrack, alter unrelated tables, open UDP5060/FastAGI, or restart Asterisk.
There were no outbound PJSIP registrations/trunks requiring exceptions on this host.

The root worker rechecks native account eligibility and aggregates leases by IP.
Agents receive phone access. Per the operator's correction, eligible active level8/9
supervisors also receive phone access **only** when native
`vdc_agent_api_access='1'` and `api_allowed_functions` permit `blind_monitor` or
`ALL_FUNCTIONS`. This follows the installed `non_agent_api.php:3979–3995` checks;
VICIdial retains control over actual monitoring/barge actions. Supervisors without
that permission remain web-only. The browser cannot supply this capability.
Only column-level SELECT for those two permission fields was added to the
existing read-only portal DB account; no VICIdial user permissions or schema changed.

Sign-out now revokes **all grants for the authenticated account on its current
IP**, including older browser sessions. Persisted revocation timestamps reject
stale queued grants regardless of processing order. Other users sharing the IP,
and other IPs, retain their valid grants. A shared IP is a network boundary:
while any applicable phone grant remains, individual users cannot be isolated
at the IP firewall; native PJSIP and VICIdial credentials remain necessary.

Revocation becomes effective on worker reconciliation, normally within10seconds.
A stopped worker cannot extend early-gate authorization beyond35seconds from its
last refresh or the absolute lease deadline. Only established RTP conntracks
marked with reserved bit `0x40000000` while authorized may continue after the last
phone grant, for **at most120seconds**. Other mark bits are preserved. New RTP flows
and all SIP traffic are blocked during grace. This is a media-flow grace, not a
call-aware drain: SIP call control is blocked and media is cut at the deadline.
Asterisk channels/contacts may linger; no forced channel hangup is issued.
Absolute grace deadlines are persisted before effects, so retries cannot extend
them. A required boot unit installs the empty guard before firewalld/Asterisk.

Apache continues to enforce its source-IP, root-owned expiry-file boundary on
all protected resources. Only browser GET navigation to the two exact entry
scripts receives **303** to the fixed public root. AJAX, API, resource, POST,
invalid-host and non-navigation requests retain403. No requested destination,
password, token, or query string is copied into the redirect. A fixed short-lived
HttpOnly/Secure notice cookie displays the missing/expired/revoked message.
The VICIdial-style portal, CSRF, sessions, rate limiting and native VICIdial
login/2FA are preserved. No upstream VICIdial PHP was modified.

## Exact changes and backups

Runtime files changed or added:

- `/opt/alma-portal-443/{portal.py,access.lua,apply_requests.py,sip_guard.py,phone_policy.py}`
- `/etc/httpd/conf.d/02-alma-vicidial-ssl.conf`
- `/etc/systemd/system/{alma-portal-443-apply.service,alma-portal-sip-guard.service}`
- Required/wanted links for the guard under `asterisk.service.requires`,
  `firewalld.service.requires`, and `multi-user.target.wants`.
- `/etc/alma-portal/vicidial-db.ini`: portal service secret rotation only;
  ownership, mode and SELinux context preserved. Native VICIdial credentials unchanged.
- Root-owned portal queue/state/status/web files and portal-owned runtime
  firewall rules/sets, as part of normal grant reconciliation.

Pre-change archive and task rollback: `/root/alma-portal-revoke-backup/`, private
root-only directory. Mutable state was archived for evidence but is deliberately
not restored by rollback. Root-only staging: `/root/alma-portal-revoke-stage/`.
Source/test/deployment helper changes are recorded by `git show --stat 1afdd6f`.
The report and evidence are committed separately.

[before.sha256](revocation-evidence/before.sha256) and
[after.sha256](revocation-evidence/after.sha256) record configuration hashes.
[protected.sha256](revocation-evidence/protected.sha256) records untouched native
PHP, PJSIP, dialplan, verifier and Fail2Ban files. The operator's SSH change altered
`public.xml` independently; no old firewall XML is restored. Credential-config
before/after hashes and its private backup remain on the server and are not exported.

The old migration archive proves static residential UDP5061/RTP test allowances
previously existed. Inspection found them already removed **before this task**;
we did not claim or repeat their removal. No permanent rich rules or direct rules
remain. The unrelated residential TCP443 allowance is harmless to the Apache
boundary and was preserved.

During diagnostics a SHOW GRANTS response inadvertently included the internal
portal DB account verifier in tool output. The service secret was subsequently
rotated server-side, the config replaced atomically, and a new read-only connection
verified. No verifier/plaintext secret is included in this report or exported
artifacts. No real VICIdial user password was captured or changed.
[Rotation result](revocation-evidence/credential-rotation.log).

## Executed verification

| Check | Result and evidence |
|---|---|
| No-grant fresh Zoiper REGISTER | **PASS — independent operator**: registration blocked; both browser redirects passed. |
| Agent grant / fresh REGISTER / agent interface / revoke | **PASS — independent operator**: authorized registration succeeds; agent page works; registration blocked after revoke. |
| Permission-qualified supervisor administration / fresh REGISTER / revoke | **PASS — independent operator**: administration and registration pass; revoked registration blocked. |
| Final re-login and account-scoped revoke | **PASS — operator and server**: final repeated fresh registration blocked; server records revoke and zero residential leases. |
| Browser redirect without grant | **PASS — independent browser plus real HTTPS probe**: exact agent/admin entries redirect to fixed portal root; useful notice appears. |
| Unauthorized API/AJAX/resources/POST | **PASS — real HTTPS probe**:403, including HTML Accept and AJAX headers. |
| Open redirect, loops, encoded/alias paths and spoofed host/source headers | **PASS — real HTTPS probe**: no untrusted Location; protected paths denied; public root remains reachable. |
| Authorized supporting CSS/JS/images and native login pages | **PASS — real HTTPS probe**, temporary server-IP web fixture, then removed. |
| TLS trust/IP SAN and HTTP ACME route | **PASS — real HTTPS probe**; no certificate or renewal configuration changed. No ACME issuance was forced in this task. |
| Established UDP bypass and fresh SIP packets | **PASS — isolated real-kernel network probe**: existing tuple and fresh tuple blocked after revoke, despite downstream established accept. |
| RTP grace/new flow denial/expiry | **PASS — isolated real-kernel network probe**: old marked RTP survives grace, new flow denied, old flow denied after native timeout. Shortened test timeout; no production call involved. |
| Absolute grant expiry without worker | **PASS — isolated real-kernel network probe**: expired native timeout blocks established SIP. Unit tests verify absolute deadline and35second freshness cap. |
| Shared-IP aggregate / web-only supervisor | **PASS — live worker fixture plus tests**: one user's revoke preserves another grant; web-only does not grant SIP; monitoring supervisor capability rechecked from DB. |
| Account disable/ineligible, invalid passwords, role spoof, stale queue, failed worker | **PASS — mocked Python tests and live ineligible queue fixture**; no real accounts disabled or passwords collected. |
| Python tests | **PASS —20 locally and20 on server**. Local sandbox initially blocked the loopback listener; rerun with loopback permission passed. |
| Countdown JavaScript | **PASS —7 cases**, plus syntax check. |
| Apache / systemd / shell / Python syntax | **PASS**. Apache emits its existing global ServerName warning and `Syntax OK`; no unrelated change made. |
| Runtime safety | **PASS**: all8 checked services active; Asterisk PID143137 unchanged,0 active calls/channels; Fail2Ban jails remain `alma-portal, asterisk`. |
| SSH / Cockpit /9091 | **PASS for preservation**: SSH22222 live and allowed in runtime/permanent config;9091 has no listener; Cockpit9090 still has no listener, as at baseline. No Cockpit change was made. |
| Boot ordering | **PASS for configuration only**: guard enabled/required before firewalld/Asterisk; unit validation passes. Production reboot **NOT_RUN**. |
| Two-way audio / monitoring / barge / live120second call grace | **NOT_RUN**, explicitly reported by operator. No claim of media acceptance. |

Primary evidence:
[HTTPS](revocation-evidence/https-probe.log),
[kernel packet probe](revocation-evidence/network-probe.log),
[live worker](revocation-evidence/worker-probe-final.log),
[server Python](revocation-evidence/python-tests.log),
[local Python](revocation-evidence/local-python-tests.log),
[operator results](revocation-evidence/operator-results.json),
[final health](revocation-evidence/health.json).
The REGISTER observer corroborated ingress attempts only
([aggregate counts](revocation-evidence/first-registration-counts.json));
no packet capture, stored authentication headers or observed SIP200 is claimed.
The actual Zoiper registration outcomes above are the independent operator's results.

Final state: residential valid web/phone leases **0/0**, web file absent,
portal-owned runtime5061/RTP rules absent, early `agents4` and `grace4` membership
absent. One valid phone grant on another IP remains preserved; that IP and user
identity are excluded from exported evidence. Root private config/state archives
and credential-bearing files were not downloaded or committed.

## Repeatable checks

Local (HTTP tests need a temporary loopback listener):

```sh
python3 -m unittest discover -s operations/portal443/tests -v
node --check operations/portal443/app.js
node operations/portal443/tests/test_app.js
python3 -m compileall -q operations/portal443
bash -n operations/portal443/revoke-deploy.sh
bash -n operations/portal443/revoke-rollback.sh
git diff --check
```

The named-host probes are already staged. `worker_probe.py` requests the lock-protected root worker through systemd and
refuses an existing server-IP grant; its temporary eligible-account fixtures
exercise the worker without real user passwords. The kernel probe uses isolated
namespaces with cleanup, never production routes or registrations. Run during
an authorized maintenance/test window:

```sh
ssh -p 22222 root@2.29.153.161 'cd /root/alma-portal-revoke-stage && /opt/alma-portal-venv/bin/python probe.py'
ssh -p 22222 root@2.29.153.161 'cd /root/alma-portal-revoke-stage && /opt/alma-portal-venv/bin/python worker_probe.py'
ssh -p 22222 root@2.29.153.161 'cd /root/alma-portal-revoke-stage && /opt/alma-portal-venv/bin/python guard_network_probe.py'
```

## Rollback and limitations

Keep an SSH session/recovery console available. Use **this task's** backup:

```sh
ssh -p 22222 root@2.29.153.161 'bash /root/alma-portal-revoke-backup/rollback.sh'
ssh -p 22222 root@2.29.153.161 'httpd -t && systemctl is-active asterisk httpd sshd firewalld fail2ban alma-portal-443 alma-portal-443-apply.timer'
```

Rollback restores the pre-task443 portal, Apache and worker; removes only this
task's nft table/unit/permission-column grants; preserves SSH22222 and the rotated
DB service secret. It removes Requires links and reloads systemd **before** stopping
the guard, so stopping a required unit does not stop Asterisk. It does not restore
firewall XML, stale grants, database passwords, or the old9091 deployment. The
pre-task rollback was rehearsed with the Asterisk PID unchanged; the later
SELECT-column cleanup extension was syntax checked, not rehearsed against live
supervisor traffic. Rollback reintroduces the known established-UDP limitation.
Do **not** invoke the original `/root/alma-portal-443-backup/rollback.sh` for this task.

SELinux was already permissive and remains unchanged. Audio/barge and a production
reboot remain untested. Existing calls receive only the documented bounded RTP
flow grace. Do not use the VERIFIED verdict to claim those separate outcomes.

Public portal: **https://2.29.153.161/**  
Agent destination: **https://2.29.153.161/agc/vicidial.php**  
Supervisor destination: **https://2.29.153.161/vicidial/admin.php**
