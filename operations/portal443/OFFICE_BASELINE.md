# Shared Office baseline — 2026-11-06 UTC

Task authority: named VPS `root@2.29.153.161:22222`, existing portal only.
All remote activity used SSH/SCP from the desktop. Recovery-console availability
was confirmed by the operator in this conversation. No new infrastructure or
upstream VICIdial/telephony changes were authorized or made.

Local repository `/home/ra/Documents/almadial`, initial HEAD `e99be47`.
Relevant portal commits: `5619db8` (revocation acceptance/SSH22222), `1afdd6f`
(SIP gate before conntrack). Initial unrelated untracked files were
`docs/reviews/m0-05-acceptance-readback-followup.md` and
`scripts/verify_agy_overlay.py`; preserved without staging or modification.

Inspected source, HTTP/JS/Python tests, prior deployment/rollback scripts and
reports under `operations/portal443`, live `/opt/alma-portal-443`, and portal,
worker/timer and guard service definitions. Five proposed source files exactly
matched the local verified version before edits. Original password verifier,
phone policy, Lua access boundary, firewall integration and session/CSRF/rate
limiting behavior were inspected. The previous verdict was
`PORTAL_REVOCATION_REDIRECT_VERIFIED`.

Baseline records: two eligible personal phone grants on two distinct client IPs;
no office type. Existing leases use random request IDs, account/IP ownership,
role, absolute expiry, eligibility and optional revoked/requested-at/phone fields.
Root state is persisted atomically; the unprivileged portal queues requests;
the root worker rechecks accounts and publishes short-lived authorization status.
Legacy missing grant types are treated as personal without extending expiry.

Effective web and phone permissions are aggregated by IP using the maximum
remaining eligible expiry. Two timed priority10 firewalld UDP rich rules per
phone IP permit 5061 and RTP10000–20000. nft input/output priority-10 guards SIP
before downstream established accepts; existing marked RTP has a bounded
120-second last-grant grace. The Apache Lua boundary uses actual socket source,
root-published TTLs and an explicit public-route allowlist. Browser entry
navigation redirects to the fixed portal; unauthorized AJAX/API/assets get403.

Installed native `admin.php` was inspected: user management gates on
`LOGmodify_users==1` (including lines10114/10179/17894/18307); the variable is
populated from the authenticated user's row. Existing supervisor monitoring
policy matches `non_agent_api.php` blind_monitor checks. Read-only aggregate SQL
found one active eligible level9 management/monitoring account. No password was
queried for this investigation. The portal DB account initially had column SELECT
on active/api_allowed_functions/api_only_user/force_change_password/pass/pass_hash/
user/user_level/vdc_agent_api_access; `modify_users` was absent. No SHOW GRANTS or
secret-bearing configuration was exported.

Apache, portal, worker timer, SIP guard, Asterisk, MariaDB, SSH, firewalld and
Fail2Ban were active. Asterisk PID143137; SSH listener22222; no9091 listener.
Existing public ports22/80/443/22222 TCP, no permanent rich UDP grants. Port80
remains constrained by Apache to ACME. SELinux was already **Permissive**;
this assignment does not change its mode. Cockpit9090 had no listener in the
previous verified baseline; it is not part of the office grant.

Before any live changes, root-only backup `/root/alma-portal-office-backup` was
created with `config.tar`, `before.sha256`, runtime/permanent firewall listings,
private nft baseline and deidentified baseline counts. It includes the installed
portal, relevant service/Apache/firewall configuration and state. The archive
and raw client-bearing state/firewall files stay on the server. Do not use old
migration/revocation backups for this task's rollback.
