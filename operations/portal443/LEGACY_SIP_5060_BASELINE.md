# Legacy SIP5060 baseline — 2026-11-07

Scope: explicit production-task authorization for the existing portal on
`root@2.29.153.161:22222`; all remote activity via desktop SSH/SCP.
Repository branch `main`, initial HEAD `c3e5593`. Shared Office Access is already
verified and deployed (`3cadd31`, `a7a8d96`, `91616c7`). Unrelated untracked
M0/browser/review work was inspected in git status and is preserved.

Installed worker and guard source were read over SSH and their hashes match the
repository. Existing phone policy is worker `PORTS=('5061','10000-20000')` and
nft input/output SIP5061 checks at priority-10. The root worker runs every10s,
uses aggregate IP leases with35s native freshness and existing120s RTP grace.
Personal12h and supervisor office24h ownership, revocation and permissions stay
unchanged. Portal status depends on confirmed firewall and gate state.

Asterisk PID143137 listens on UDP5060 and UDP5061. Portal, apply timer, guard,
Asterisk, Apache, MariaDB, firewalld, Fail2Ban and SSH are active. One active
personal grant and no office grants were observed. Runtime rules have the two
portal-owned priority10 UDP grants for that effective phone IP, plus exactly the
operator-confirmed unprioritized temporary5060 rule. No permanent rich rules.
Runtime/permanent public services: cockpit, dhcpv6-client, ssh; TCP22/80/443/22222
port configuration unchanged. No SIP/RTP service or global UDP opening observed.

The exact supplied temporary5060 rule was confirmed active. It is independent
of portal policy and will be removed only after the new early gate is installed.
Its temporary success is not counted as portal SIP acceptance.

Before live mutation, root-only `/root/alma-portal-5060-backup/` was created
(mode700) with `config.tar`, `before.sha256` for16 files, private runtime and
permanent firewall/nft listings, atomic authorization state and a baseline PID
record. Archives/state stay on the server. Staging uses
`/root/alma-portal-5060-stage/`. No real credentials were retrieved.

Acceptance: both SIP ports under the same phone grant; no-grant/expiry/revoke
blocks both before conntrack accepts; shared grants/RTP grace preserved;
idempotent restart reconciliation; independent fresh Linphone1001 and PJSIP
registration tests through all four states; no native application/configuration
or unrelated firewall/service change. Unit/isolated packet tests are separate
from actual REGISTER acceptance.
