# Permanent PeerEdge SIP exception — dialer01

## Ownership and behavior

The source of truth is `operations/portal443/sip_guard.py` plus `carriers.json`, deployed together under `/opt/alma-portal-443/`. The guard service runs the script at startup, before firewalld/Asterisk; `alma-portal-443-apply.service` calls its `refresh()` through the existing ten-second timer. Static carrier configuration is root-owned mode0644 and contains no SIP secrets. It is separate from `/var/lib/alma-portal-443/state.json` and expiring agent grants.

`carriers4` is an IPv4 set **without timeout**, currently containing only `38.147.130.74`. Immediately after loopback handling, before agent DROP rules:

```nft
ip saddr @carriers4 udp dport 5060 counter return
ip daddr @carriers4 udp sport 5060 udp dport 5060 counter return
```

Return permits later firewalld filtering; it does not bypass it. Public/eth0 also has this exact runtime **and permanent** rich rule:

```text
rule family="ipv4" source address="38.147.130.74/32" port port="5060" protocol="udp" accept
```

There is no global SIP allowance, carrier 5061 permission, carrier RTP permission, permanent direct OUTPUT rule or dependency on agents4/grace4. Existing agent SIP/RTP rules and grace semantics are unchanged. New policy marker `alma-phone-sip-5060-5061-v2-carriers` triggers an atomic migration of only input/output chains in `inet alma_portal_sip`. It preserves dynamic sets and their native expiration deadlines. Repeated bootstrap does not duplicate rules; changed carrier membership updates only carriers4.

Missing or malformed configuration fails before guard mutation. Future carrier changes require coordinated source-controlled `carriers.json` and explicit matching source-scoped firewalld rules. Grant reconciliation never edits permanent firewalld configuration. Install **both files**, not only sip_guard.py. The historical `revoke-deploy.sh` is a different task's deployment artifact and must not be reused as the carrier deployment procedure.

## Deployment performed

2026-11-11UTC (2026-11-12Asia/Manila). Noninteractive root SSH22222. Original deployed guard matched repository SHA256 `c5b5b5d3de2d3832605f122cc9fba1dd1db6a2d7be88e8ccd89c87022e11a2bf`. No active calls/channels. Both TEMP_PEEREDGE rules and one runtime direct OUTPUT exception were present; no temporary carrier rich rule existed.

Scoped private backup: `/root/qa-backups/peeredge-20261111T164330Z/` contains original guard, public.xml, original nftables table, direct-rule inventory, migration and rollback scripts. No credentials exported. Deployment:

1. Save originals and rule state; generate rollback and validate with `bash -n` and `nft -c`.
2. Add the exact rich rule above with `firewall-cmd --permanent --zone=public --add-rich-rule=...` and separately at runtime. Validate using `firewall-cmd --check-config`. No runtime-to-permanent or reload.
3. Under the existing `/var/lib/alma-portal-443/worker.lock`, atomically install each root-owned file, restore SELinux contexts, import the new guard and run bootstrap. Every generated nft transaction passed `nft -c -f -` before actual application. Dynamic members/timeouts were compared excluding naturally decreasing remaining-expiration milliseconds.
4. Chain migration replaces the two TEMP rules atomically with permanent policy. Remove only the exact runtime direct rule `ipv4 filter OUTPUT 0 -d 38.147.130.74 -p udp --dport 5060 -j ACCEPT`. Assert no remaining carrier direct or TEMP rules.
5. Run the actual portal reconciliation service and repeated bootstrap, then observe forced and natural SIP qualifications. Do not restart Asterisk, SSH, networking, portal or firewalld.

Installed guard SHA256: `90ca6836a7876fa300d98b04a5220d5ba869db08bc84ef6ab8e078a5845f5991`.
Installed carriers.json SHA256: `0fcc9b5ba559f72068de55ab0a2d96f2785d366cac0cea1269b130cb0f017eff`.

## Verification

Evidence: `operations/qa/evidence/peeredge-firewall/{deployment.txt,verification.txt,local-tests.txt}`. All 53 portal tests passed, including atomic migration, grant/grace handling, HTTP authentication and carrier isolation/idempotence checks. Initial sandbox socket restrictions required running the suite with local socket permission; they were not product failures.

| Mandatory acceptance | Result and evidence |
|---|---|
| SIP qualification | PASS — sip show peer and sip show peers: Jaintel3 /38.147.130.74:5060 /OK139ms |
| No recurring permission errors | PASS — fresh Asterisk messages observation16:44:14–16:45:21UTC, zero new Operation not permitted/__sip_xmit carrier errors |
| Independent of agent leases | PASS — address only in non-expiring carriers4; absent from agents4/grace4 throughout |
| Unauthorized SIP remains denied | PASS — live ordered rules retain both 5060/5061 DROP rules per direction, carrier match restricted as above; six SIP/RTP drops retained. No new external unauthorized-packet probe was performed |
| Authoritative persistent configuration | PASS — deployed hashes match source; exact rich rule confirmed runtime/permanent, firewalld config check passed |
| Regeneration/reconciliation | PASS — bootstrap repeated, actual apply.service reconciliation and subsequent timer cycles retained exactly18 rules, one carrier rule per direction |
| No diagnostic-rule dependency | PASS — TEMP rules and runtime/permanent carrier direct rules absent; carrier counters increased1→3 packets in each direction |
| No observed outage/regression | PASS — dynamic sets preserved at migration; one agent/grace member remained; portal/firewalld/Asterisk/httpd/timer active, zero calls/channels before/after, processed-call count unchanged142 |

No real call or complete media acceptance is claimed. No VICIdial carrier configuration/credentials, VICIdial source, Asterisk, portal access policy, systemd unit, TLS, backup, SSH or network configuration was changed. Private backup includes public.xml for evidence only; restoring the entire zone would overwrite unrelated changes and is not the rollback procedure.

## Persistence and restart boundary

The actual owner bootstrap (same entry point as service ExecStart), worker reconciliation and subsequent ten-second timer cycles were exercised successfully. Static set creation also exists in the fresh-table template, covered by tests; migration syntax checked against the real kernel. No reboot or firewalld reload was performed. Permanent rich XML is validated separately from dynamic runtime grants.

A live guard-service restart is intentionally deferred: this unit has RequiredBy/Requires relationships with Asterisk/firewalld, so its stop/start may propagate to dependencies. In a separately authorized maintenance window, the exact deferred test is `systemctl restart alma-portal-sip-guard.service`, followed by checking/restoring previously active dependent services, portal reconciliation, the18-rule table, permanent/runtime rich rule and Jaintel3 qualification. Do not run it during calls. A firewalld reload likewise needs a planned window because it may remove temporary agent rich rules until the next reconciliation; it was unnecessary for this deployment.

## Scoped rollback

Prepared remote script: `/root/qa-backups/peeredge-20261111T164330Z/rollback.sh`. Shell syntax and its old-policy chain transaction passed checks against the live kernel; **live rollback was not executed**. Run once through noninteractive SSH if reverting this deployment:

```bash
ssh -T -p 22222 root@2.29.153.161 'bash /root/qa-backups/peeredge-20261111T164330Z/rollback.sh'
```

It holds the existing worker lock, restores only the original guard, invokes its native atomic chain migration (preserving agents4/grace4), reinstates both exact diagnostic nft exceptions and the original runtime direct OUTPUT rule, removes only the newly added carrier rich rule in runtime/permanent config, and removes carriers.json. It never flushes the ruleset, deletes the portal table, restores the entire zone or changes SSH. The unused carriers4 set may remain after reversal, unreferenced by original policy. Confirm Jaintel3 reachability, grant membership and service health afterward. Avoid repeating the rollback blindly because the diagnostic insertions are intended for one execution. Reapply the permanent solution once the cause of rollback is understood.

## RTP follow-up

RTP10000–20000 is still limited to authorized agents and marked established grace flows. A carrier media IP outside those sets will encounter the same default drop. No verified PeerEdge media ranges were supplied or inferred from its signaling IP. Obtain authoritative carrier media addresses and authorize a separately scoped RTP exception/test; do not add the signaling IP to agent grants or open RTP globally.

Final verdict: `PEEREDGE_CARRIER_FIREWALL_PASS` — persistent SIP signaling only.
