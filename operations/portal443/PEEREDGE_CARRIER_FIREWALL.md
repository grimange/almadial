# Permanent PeerEdge SIP exception — dialer01

## Ownership and behavior

The source of truth is `operations/portal443/sip_guard.py` plus `carriers.json`, deployed together under `/opt/alma-portal-443/`. The guard service runs the script at startup, before firewalld/Asterisk; `alma-portal-443-apply.service` calls its `refresh()` through the existing ten-second timer. Static carrier configuration is root-owned mode0644 and contains no SIP secrets. It is separate from `/var/lib/alma-portal-443/state.json` and expiring agent grants.

`carriers4` is an IPv4 set **without timeout**, currently containing only `38.147.130.74`. Immediately after loopback handling, before agent DROP rules:

```nft
ip saddr @carriers4 udp dport 5060 counter return
ip daddr @carriers4 udp sport 5060 udp dport 5060 counter return
```

Return permits later firewalld filtering; it does not bypass it. Public/eth0 now has this exact runtime **and permanent** CLI-managed rich rule (the original deployment used the same rule without priority):

```text
rule priority="-73" family="ipv4" source address="38.147.130.74/32" port port="5060" protocol="udp" accept
```

There is no global SIP allowance, carrier 5061 permission, carrier RTP permission, permanent direct OUTPUT rule or dependency on agents4/grace4. Existing agent SIP/RTP rules and grace semantics are unchanged. New policy marker `alma-phone-sip-5060-5061-v2-carriers` triggers an atomic migration of only input/output chains in `inet alma_portal_sip`. It preserves dynamic sets and their native expiration deadlines. Repeated bootstrap does not duplicate rules; changed carrier membership updates only carriers4.

Missing or malformed configuration fails before guard mutation. Use the carrier CLI described below for future carrier changes; it updates JSON, nftables and both firewalld layers together. Grant reconciliation never edits permanent firewalld configuration. Use the scoped CLI installer rather than copying a guard or a stale JSON file alone. The historical `revoke-deploy.sh` is a different task's deployment artifact and must not be reused as the carrier deployment procedure.

## Original permanent-firewall deployment (historical)

2026-11-11UTC (2026-11-12Asia/Manila). Noninteractive root SSH22222. Original deployed guard matched repository SHA256 `c5b5b5d3de2d3832605f122cc9fba1dd1db6a2d7be88e8ccd89c87022e11a2bf`. No active calls/channels. Both TEMP_PEEREDGE rules and one runtime direct OUTPUT exception were present; no temporary carrier rich rule existed.

Scoped private backup: `/root/qa-backups/peeredge-20261111T164330Z/` contains original guard, public.xml, original nftables table, direct-rule inventory, migration and rollback scripts. No credentials exported. Deployment:

1. Save originals and rule state; generate rollback and validate with `bash -n` and `nft -c`.
2. Add the exact rich rule above with `firewall-cmd --permanent --zone=public --add-rich-rule=...` and separately at runtime. Validate using `firewall-cmd --check-config`. No runtime-to-permanent or reload.
3. Under the existing `/var/lib/alma-portal-443/worker.lock`, atomically install each root-owned file, restore SELinux contexts, import the new guard and run bootstrap. Every generated nft transaction passed `nft -c -f -` before actual application. Dynamic members/timeouts were compared excluding naturally decreasing remaining-expiration milliseconds.
4. Chain migration replaces the two TEMP rules atomically with permanent policy. Remove only the exact runtime direct rule `ipv4 filter OUTPUT 0 -d 38.147.130.74 -p udp --dport 5060 -j ACCEPT`. Assert no remaining carrier direct or TEMP rules.
5. Run the actual portal reconciliation service and repeated bootstrap, then observe forced and natural SIP qualifications. Do not restart Asterisk, SSH, networking, portal or firewalld.

Original installed guard SHA256: `90ca6836a7876fa300d98b04a5220d5ba869db08bc84ef6ab8e078a5845f5991`.
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


## Carrier management CLI

The CLI is installed at `/usr/local/sbin/alma-portal-carriers`, with modules `carrier_cli.py` and `sip_guard.py` under `/opt/alma-portal-443`. It uses system Python 3 without additional packages or a new virtual environment. Run as root; read-only commands also require access to the existing worker lock and firewall diagnostics.

```bash
sudo alma-portal-carriers list
sudo alma-portal-carriers add <CARRIER_PUBLIC_IP>
sudo alma-portal-carriers validate
sudo alma-portal-carriers remove <CARRIER_PUBLIC_IP>
sudo alma-portal-carriers sync
```

`list` and `validate` never change configuration or firewall state. Both report carrier-set membership, guard-policy markers and runtime/permanent firewalld drift, including obsolete managed rules; drift returns status 1. `add` preserves every existing address; adding an existing address repairs drift without rewriting JSON. `remove` warns that carrier traffic may disconnect, requires no confirmation and removes only the selected carrier's managed permissions. Removing an absent valid address is an explicit no-op. `sync` repairs drift against the current JSON and removes obsolete **managed** rules. Success returns 0, operational/validation errors 1, and invalid command syntax 2. Use `--help` for command syntax.

The existing schema is a JSON array of unique public unicast IPv4 strings. There are no metadata fields to preserve or extend. Unsupported object schemas, malformed JSON, duplicate entries, IPv6, private/documentation/reserved, multicast, loopback, unspecified and broadcast addresses are rejected before changes. The documentation address `203.0.113.10` is rejected and was never authorized live. The config must be a regular root-owned file without group/other write permission; symlinks are rejected. Atomic replacements preserve its permission mode.

Only exact source-/32, UDP destination 5060, accept rich rules with reserved **priority -73** belong to this CLI. Do not use that exact namespace for unrelated rules. Other priorities, unmarked rules, other ports and dynamic agent grants remain untouched, even when their source IP matches a carrier. Consequently, removing a carrier does not revoke separately authorized agent access for the same IP. The nftables policy still permits only inbound UDP/5060 and outbound UDP source 5060/destination 5060 for carriers. It adds no carrier RTP or 5061 access; carrier addresses never enter agents4/grace4 and never expire with agent leases.

All CLI operations use the existing `/var/lib/alma-portal-443/worker.lock`; reads take a shared lock, mutations an exclusive lock with a ten-second acquisition deadline. Standalone guard bootstrap now takes that same lock; the imported worker already holds it and does not acquire it twice. The installed portal and persistent worker-lock file are prerequisites. No additional timer or daemon is introduced.

Mutations validate input and configuration, snapshot carrier membership and owned rules, atomically replace JSON if needed, reconcile permanent/runtime rich rules, run native guard bootstrap and verify all layers. Detected failures attempt exact restoration of the previous JSON bytes, carrier set and owned rich rules, preserving unrelated rules. A nonzero result states whether restoration succeeded or identifies remaining inconsistencies. A failed firewall query causes no speculative reset. An abrupt process/host crash cannot provide an all-layer transaction: after interruption, run `validate`, review the authoritative JSON, then `sync`. If a reported rollback is incomplete, repair the named layer and run `sync`/`validate`; do not restore the whole firewalld zone or flush nftables.

### Installation and configuration ownership

From a reviewed checkout on the target, with the existing portal installed:

```bash
sudo python3 operations/portal443/install-carriers.py
```

For the one-time migration from the original accepted PeerEdge rich rule, use `--adopt-peeredge`. This explicitly migrates **only** that exact historical rule in runtime/permanent configuration, after the marked replacement is verified. It does not infer ownership of other rules from matching IPs. Future installs normally omit this flag. Optional `--expect-guard-sha256 <INSPECTED_SHA256>` refuses a changed deployed guard. The installer atomically installs the two modules and executable, restores SELinux contexts, synchronizes permissions and saves a private scoped backup. It preserves the live JSON and performs no firewall reload or service restart. Keep its reviewed source directory for rollback; it must contain the CLI, guard, entry point and the existing compatible `common.py`.

**Operational authority is `/opt/alma-portal-443/carriers.json`.** `operations/portal443/carriers.json` is the source-controlled deployment inventory. After an operational add/remove, read the live JSON through authorized SSH, review the exact address-only diff, update that repository file and commit it under the normal change process. Never deploy an older repository JSON over live operational changes. This installer deliberately never copies its checkout's JSON. A separate reviewed configuration rollout must also use the same worker lock and run CLI `sync`/`validate` while accounting for rollback. There is no hidden second address inventory or automatic repository write from the server.

### This deployment and rollback

2026-11-11 17:16 UTC (2026-11-12 Asia/Manila). Source baseline `b8b165c8b7eb7e933a3450fb7f33e6c607651578`; 68 portal tests and 15 isolated CLI tests on target Python passed. Sanitized evidence: `operations/qa/evidence/carrier-cli/{tests.txt,deployment.txt,target-tests.txt}`. Private installer source is `/root/qa-backups/carrier-cli-source-20261111T171620Z`; scoped backup is `/root/qa-backups/carrier-cli-20261111T171621186352Z`.

Live `list`, `validate`, duplicate PeerEdge `add`, two `sync` calls, standalone bootstrap, normal portal reconciliation and final validation passed. One carrier remains in each layer; JSON bytes are unchanged. The 18 nft rules and agent/grace membership were unchanged; three unrelated runtime rich rules were preserved. Permanent configuration check and all six inspected services passed; calls/channels stayed zero with 142 processed calls. The SIP peer is intentionally deactivated by the operator: `Peer Jaintel3 not found.` New qualification is NOT_RUN; no peer configuration was changed or reachability claimed. Full add/remove and rollback were tested only against isolated file/mocked-firewall fixtures, never by removing PeerEdge or adding arbitrary live addresses.

Rollback of this CLI installation (not the earlier permanent carrier integration):

```bash
ssh -T -p 22222 root@2.29.153.161 \
  'python3 /root/qa-backups/carrier-cli-source-20261111T171620Z/install-carriers.py --rollback /root/qa-backups/carrier-cli-20261111T171621186352Z'
```

The script holds the worker lock, restores the exact old PeerEdge rules before removing newly owned rules, restores only the snapshotted carrier membership and affected source/entry files, then verifies read-back. It preserves other nftables sets, rich rules, SSH and services. The backup includes original JSON for evidence, but rollback refuses to run if live carrier configuration changed after installation; review and adapt the carrier-specific reversal in that case rather than discarding later changes. The source directory must be retained until rollback is no longer needed. Rollback passed an isolated execute-and-read-back test; it was not executed against the live firewall. Live code was compiled with target Python 3.12.15 before installation. A guard-service restart and firewalld reload remain unnecessary and deferred for the dependency reasons described above.
