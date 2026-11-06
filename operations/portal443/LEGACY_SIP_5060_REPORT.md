# Legacy SIP UDP 5060 authorization

**PORTAL_LEGACY_SIP_5060_VERIFIED**

Independent client tests and server evidence confirm that both SIP transports
register with a portal phone grant, fail without one and after revocation, and
register again after reauthorization. The temporary allowance is absent; no
permanent or global UDP 5060 opening remains.

## Implementation and operation

This extension adds UDP 5060 to the existing phone policy. Previously the worker
authorized UDP 5061 and UDP 10000–20000. It now authorizes UDP 5060, UDP 5061 and
UDP 10000–20000 through the same aggregate grant, timeout and reconciliation
paths. Users do not select a SIP transport. Personal access remains 12 hours;
Shared Office Access remains 24 hours with the existing supervisor eligibility.
HTTP-only grants do not authorize either SIP port or RTP.

Only three production source files change in `/opt/alma-portal-443/`:

- `common.py`: central `SIP_PORTS = ('5060', '5061')`.
- `apply_requests.py`: derive phone service ports from `SIP_PORTS` plus RTP.
- `sip_guard.py`: generate both SIP port checks in the existing input/output
  chains, and upgrade an existing policy in one atomic nftables transaction.

The existing `inet alma_portal_sip` table has input/output hooks at priority -10,
before firewalld's established-connection acceptance. Both SIP ports require the
same timed `agents4` membership. Gate returns still pass through firewalld;
they do not override another firewall denial. Policy upgrades preserve timed
`agents4` and `grace4` elements and their deadlines. Repeated bootstrap does not
duplicate rules. There are 16 rules across the two chains after this extension.

The worker owns one source-specific rich rule per service per effective IP,
irrespective of the number of personal/office grants behind that IP. Revoking
one grant preserves independently valid grants. Removing the last phone grant
blocks new registrations on both SIP ports after reconciliation, normally within
10 seconds. Native 35-second freshness bounds stale worker permissions. Already
authorized, marked RTP flows retain the existing maximum 120-second grace;
new RTP flows do not acquire grace. No connection-tracking table is flushed.
Network grants provide no isolation between people behind the same NAT address;
native VICIdial and SIP credentials remain mandatory for each user/phone.

The exact operator-confirmed temporary, unprioritized UDP 5060 rule was present
at baseline and removed during deployment, after installing the early gate.
No permanent or globally open SIP rule was added. No upstream VICIdial source,
MariaDB schema, Asterisk configuration, role policy, UI or SSO behavior changed.

Repository files changed in implementation commit `47efc2f` are the three
production sources above; `tests/test_portal.py`, `tests/test_office.py`,
`tests/test_sip_guard.py`; `guard_network_probe.py`, `worker_probe.py`,
`office_evidence.py`, `register_observer.py`; and the new
`legacy-sip-deploy.sh`, `legacy-sip-rollback.sh`, `legacy_sip_evidence.py`.
The documentation change includes `OFFICE_ACCESS.md`, this report,
`LEGACY_SIP_5060_BASELINE.md` and `legacy-sip-evidence/`.
Production before/after hashes are in
`legacy-sip-evidence/production-checksums.tsv`; the sanitized snapshots include
all 16 control-file hash comparisons. The other 13 control files are unchanged,
including native PHP, SIP/PJSIP/dialplan, firewall XML, portal UI/authentication,
Apache and Fail2Ban configuration. Unrelated local M0/browser files are preserved.

## Deployment and rollback

The desktop used SSH port 22222 for all remote activity. Before deployment,
16 source/control files, private authorization state and firewall baselines were
backed up under root-only `/root/alma-portal-5060-backup/`. Do not export the
archive or raw state; they contain private configuration and authorization data.

`legacy-sip-deploy.sh` requires that backup and the executable rollback script.
It compiles the staged sources, stops the apply timer/worker, installs the three
files, restores file labels, upgrades the gate before adding UDP 5060 permits,
removes only the confirmed temporary test rule, and starts the worker/timer.
An error invokes rollback. Asterisk, MariaDB, Apache, firewalld, Fail2Ban, SSH
and the HTTP portal were not restarted for this change.

Rollback from the desktop:

```bash
ssh -p 22222 root@2.29.153.161 /root/alma-portal-5060-backup/rollback.sh
```

Rollback was rehearsed successfully before deployment, including checksum
verification of all 16 backed-up files. It restores the previous three sources,
removes only portal-owned UDP 5060 rich rules and reconciles current grants.
It preserves live timed sets and RTP grace, but keeps an unconditional early
UDP 5060 drop in both directions so old conntrack entries cannot survive as a
legacy SIP bypass. UDP 5061 and RTP continue under the previous policy. It does
not restore stale grants, old firewall XML or the temporary test allowance.
Re-deployment with `legacy-sip-deploy.sh` replaces that conservative gate with
the versioned dual-port policy. Keep the recovery console available.

## Repeatable validation

```bash
python3 -m unittest discover -s operations/portal443/tests -v
node --test operations/portal443/tests/test_app.js
bash -n operations/portal443/legacy-sip-deploy.sh operations/portal443/legacy-sip-rollback.sh
git diff --check
```

Run remote Python checks with `/opt/alma-portal-venv/bin/python`, rather than
the system interpreter: the portal's existing bcrypt dependency is in its venv.
The server-side probes run through SSH in private staging:

- `guard_network_probe.py`: actual isolated network namespaces, nftables, UDP
  echo and conntrack tests on both ports; native timeout and RTP grace tests;
  existing-policy migration with live timed members. These are packet tests,
  not authenticated SIP registration tests.
- `office_probe.py`: real eligibility, worker, firewall and HTTPS checks using
  a server-IP fixture, including grant coexistence and accelerated expiry.
- `worker_probe.py`: personal grants, reconciliation, revocation and retries.
- `probe.py`: trusted HTTPS, public portal, protected paths, fixed browser
  redirects, API/resource denials, source-header spoofing and ACME routing.

`register_observer.py` is an opt-in, bounded raw-socket observer for the explicitly
identified test network. It inspects packets only in memory and stores REGISTER
counts/status codes by port and grant state, never packets, credentials, headers
or SIP identities. Positive-expiry REGISTER 200 responses are distinguished from
deregistration replies. Existing native Asterisk notices provide corroborating
event counts without enabling protocol debug. No tcpdump package was installed.

`legacy_sip_evidence.py` records aggregate health and checksums in the private
backup, then exports only whitelisted, sanitized evidence. Local evidence is
under `legacy-sip-evidence/`; raw firewall/state/configuration stay on the server.

## Executed results

Implementation commit: `47efc2f` (`feat(portal): authorize legacy SIP through
existing phone grants`). Documentation/evidence are committed separately.

| Check | Result | Evidence and boundary |
| --- | --- | --- |
| Local Python regression suite | PASS, 38 tests | Mock users, clocks, firewall and real loopback HTTP; no production credentials |
| Local JavaScript suite | PASS, 7 checks | Confirmed-active countdown, fixed destination, expiry/failure behavior |
| Remote staged Python suite | PASS, 38 tests | Existing portal venv |
| Isolated kernel/network probe | PASS, 20 checks | Both SIP ports, established conntrack, native expiry, web-only denial, migration, RTP grace; UDP echo rather than real SIP |
| Live office probe | PASS, 9 checks | Real worker/firewall/nft/HTTPS, server-IP fixture, both coexistence orders, stale queue and accelerated expiry |
| Live personal worker probe | PASS, 7 checks | Existing personal grant policy and effective three-service rules |
| Live HTTPS probe | PASS, 9 checks | Portal, native paths/assets, redirects/403, TLS, spoofing, ACME |
| Worker/timer reconciliation | PASS, 2 checks | Lease data retained; exactly 16 gate rules and both policy markers, no duplicates |
| Rollback rehearsal | PASS | Previous source restored and all 16 baseline checksums matched before deployment |
| Python compilation / shell syntax / diff whitespace | PASS | Modified Python, both shell helpers, `git diff --check` |
| Permanent firewalld configuration validation | PASS | `firewall-cmd --check-config`, exit 0 |
| Final service/control-file health | PASS | Nine services active; Asterisk PID unchanged; both SIP listeners and SSH 22222 present; Apache syntax exit 0; other 13 hashes unchanged |

The initial restricted local test run could not bind loopback sockets; the
subsequent run with loopback permission passed all 38 tests. A snapshot command
initially used system Python and failed to import bcrypt; rerunning with the
existing portal venv succeeded. Neither failure altered production state.

Independent tests use Linphone extension 1001 on UDP 5060 and the existing
PJSIP client on UDP 5061. Each phase requires fresh registration, rather than
retained contacts or an old client indicator:

1. Revoke all applicable grants, wait 15 seconds, force both registrations.
2. Authorize through the normal portal, wait for Access granted, force both.
3. Revoke, wait 15 seconds, force both, including previously established flows.
4. Reauthorize through the portal and force both again.

| Independent client phase | UDP 5060 / Linphone 1001 | UDP 5061 / PJSIP |
| --- | --- | --- |
| A: no grant | PASS, fresh registration failed | PASS, operator reported failure; packet arrival corroborated during C |
| B: authorized | PASS, two positive-expiry REGISTER 200 responses | PASS, two positive-expiry REGISTER 200 responses |
| C: revoked | PASS, 39 additional incoming REGISTERs, no replies | PASS, 13 additional incoming REGISTERs, no replies |
| D: reauthorized | PASS, two new positive-expiry REGISTER 200 responses | PASS, two new positive-expiry REGISTER 200 responses |

Phase C counts are deltas from the completed B snapshot, before D began.
Both early-gate input drop counters increased during the revoked phase.
The first C response was not accepted until a new snapshot confirmed that the
actual grant, all three owned service rules and native membership were removed.
Separate PJSIP retries were requested until its traffic actually reached the
observer. Positive-expiry replies were distinguished from deregistration 200s.
See `legacy-sip-evidence/operator-results.md` and the sanitized phase snapshots.

Final state intentionally retains the operator's new valid personal grant.
There is one effective phone IP, with one owned rule each for UDP 5060,
UDP 5061 and UDP 10000–20000. There are zero permanent rich rules, zero
unprioritized runtime SIP accepts and no temporary test rule. Runtime and
permanent global ports remain TCP 22/80/443/22222; no SIP/RTP UDP port is globally open.
Unrelated grants and expiries are preserved. The observer was stopped after
acceptance. The observer retained no packet captures, SIP headers, credentials
or SIP identifiers; the exported evidence contains only sanitized aggregates
and hashes.

The public portal remains `https://2.29.153.161/`. Agent and supervisor
destinations remain `/agc/vicidial.php` and `/vicidial/admin.php` respectively,
with separate native VICIdial authentication. Supervisor SSO is deferred.

## Limits

Unit and namespace fixtures are distinguished from actual client registrations.
Accelerated expiry/native timeout tests do not represent waiting 12 or 24 hours.
No new call/audio, monitoring/barge, full reboot or client mobile-layout test is
claimed by this SIP-only change. SELinux was already Permissive at baseline and
was preserved; changing that policy is outside this assignment. Cockpit had no
TCP 9090 listener at baseline and remains unchanged. SSH port 22222 is preserved.

Automatic approval review rejected installation of the missing tcpdump RPM as
an unapproved persistent production change. The operator selected the existing
no-install metadata observer; no package was installed and this did not prevent
the actual SIP acceptance checks.
