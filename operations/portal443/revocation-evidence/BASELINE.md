# SIP revocation / browser redirect baseline

Inspection over SSH on dialer01 / 2.29.153.161, 2026-11-05 21:50–21:57 UTC.

- Portal HTTPS/UI/agent/supervisor browser flows already deployed; this task preserves those changes.
- No active portal leases, no synced portal firewall rules. No permanent rich rules.
- Original migration backup confirms obsolete residential 158.62.74.253 UDP5061/RTP static accepts. Those rules were already removed before this task; current permanent and runtime configuration contain no such UDP accepts. Runtime retains a residential TCP443 accept and unrelated Fail2Ban rejects; preserve them.
- nftables inet firewalld INPUT/OUTPUT hook priority +10 accepts established/related before zone rules. `/proc/net/nf_conntrack` contains an ASSURED residential UDP5061 flow, mark=0, timeout repeatedly renewed near 120 seconds. This is a bypass of zone-rule removal, even with no current grant. A retained contact alone is not registration evidence.
- No other nftables tables, no direct firewalld rules, only public zone/eth0. No existing conntrack mark setters. Reserve bit 0x40000000 while preserving all other mark bits.
- Asterisk/httpd/firewalld/Fail2Ban/portal services active; zero active Asterisk calls at inspection. Asterisk PID143137 since 2026-11-05 10:25:33 EST. No restart planned.
- SSH22, ACME80, HTTPS443 permitted. Cockpit allowance unchanged; no Cockpit listener is baseline. No public9091 listener.
- Apache authorizes every non-public path against root-owned IP expiry files; unauthorized protected requests currently receive plain-text403. Public portal exact allowlist and source-IP checks retained.
- nft/ip tools installed; conntrack/tcpdump absent. No packages need installation. SELinux baseline permissive, unchanged.

Accepted operator policy: after the last applicable agent grant, block SIP immediately at worker reconciliation (normal timer interval10s), allow only already-marked established RTP flows for at most120s. No new RTP flows during grace. Other valid agent leases on a shared IP keep access; web-only supervisor leases never independently grant SIP/RTP. Revocation can disrupt calls at the120s deadline; no forced Asterisk channel hangup or restart.

Subsequent operator correction during acceptance: supervisors need Zoiper for monitoring/barge. Therefore the final policy uses the last **phone-enabled** lease, including supervisors whose native monitoring permission allows it. Source inspection of installed `non_agent_api.php:3979–3995` shows the `blind_monitor` function allowlist plus `vdc_agent_api_access='1'`, active status and level>6 checks. Portal supervisors remain eligible levels8/9. Only the two permission columns need additional SELECT access for the portal DB account; its existing authentication verifier is unchanged. No VICIdial user permission is changed. The installed active level9 supervisor has both monitoring permissions. Supervisors lacking those permissions remain web-only.

Plan: a separate portal nftables table at priority-10 checks source/destination authorization before firewalld's established accept. Native timeout sets cap a failed worker's authorization freshness at35s. Boot ordering installs an empty fail-closed guard before firewalld/Asterisk. Mark only RTP flows seen during a valid agent grant. Do not flush conntrack globally. Exact unauthorized browser GET navigations to the two VICIdial entry scripts receive a fixed303 to the public root; API/AJAX/resources retain403. A short-lived non-authentication cookie supplies an expired/revoked notice without URL parameters.

Priority semantics: [Netfilter hooks](https://wiki.netfilter.org/wiki-nftables/index.php/Netfilter_hooks). Apache hook: [mod_lua](https://httpd.apache.org/docs/2.4/mod/mod_lua.html). Runtime packet behavior must be tested; source reasoning alone is not acceptance.

Later operator changes and final integration:

- The operator changed SSH from22 to22222 during this task. Passwordless SSH22222 and runtime/permanent22222 firewall allowances were verified. The task does not restore the old firewall XML or SSH configuration. Port22 remains allowed in firewalld but sshd listens on22222; it was not removed as an unrelated change.
- The final sign-out action revokes all leases for the authenticated account on its current IP, including grants orphaned by an earlier browser session or portal backend restart. Other users and IPs remain authorized. Persisted timestamps reject stale queued grants regardless of queue order.
- A diagnostic SHOW GRANTS command inadvertently returned the internal portal DB account verifier to tool output. The service account secret was rotated server-side and the private config updated atomically. No VICIdial user password was requested, logged, or changed. Secret-bearing backups and config hashes remain root-only and are excluded from exported evidence.
