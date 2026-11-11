# Permanent carrier SIP and media authorization — dialer01

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

There is no global SIP allowance, carrier 5061 permission, permanent direct OUTPUT rule or dependency on agents4/grace4. Separate, explicitly configured media authorization is described below; its live allowlist is currently empty. Existing agent SIP/RTP rules and grace semantics are unchanged. Current policy marker `alma-phone-sip-5060-5061-v3-carrier-media` triggers an atomic migration of only input/output chains in `inet alma_portal_sip`. It preserves dynamic sets and their native expiration deadlines. Repeated bootstrap does not duplicate rules; CLI membership changes update only the selected static set.

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

A live guard-service restart is intentionally deferred: this unit has RequiredBy/Requires relationships with Asterisk/firewalld, so its stop/start may propagate to dependencies. In a separately authorized maintenance window, the exact deferred test is `systemctl restart alma-portal-sip-guard.service`, followed by checking/restoring previously active dependent services, portal reconciliation, the current20-rule table, permanent/runtime rich rule and Jaintel3 qualification. Do not run it during calls. A firewalld reload likewise needs a planned window because it may remove temporary agent rich rules until the next reconciliation; it was unnecessary for this deployment.

## Original SIP deployment rollback (historical)

**Do not use this older rollback to reverse the current media deployment. Use the scoped media installer rollback below.** Original prepared remote script: `/root/qa-backups/peeredge-20261111T164330Z/rollback.sh`. Shell syntax and its old-policy chain transaction passed checks against the live kernel; **live rollback was not executed**. Run once through noninteractive SSH if reverting this deployment:

```bash
ssh -T -p 22222 root@2.29.153.161 'bash /root/qa-backups/peeredge-20261111T164330Z/rollback.sh'
```

It holds the existing worker lock, restores only the original guard, invokes its native atomic chain migration (preserving agents4/grace4), reinstates both exact diagnostic nft exceptions and the original runtime direct OUTPUT rule, removes only the newly added carrier rich rule in runtime/permanent config, and removes carriers.json. It never flushes the ruleset, deletes the portal table, restores the entire zone or changes SSH. The unused carriers4 set may remain after reversal, unreferenced by original policy. Confirm Jaintel3 reachability, grant membership and service health afterward. Avoid repeating the rollback blindly because the diagnostic insertions are intended for one execution. Reapply the permanent solution once the cause of rollback is understood.

## RTP acceptance boundary

The media-management implementation is deployed; no PeerEdge media address has been verified or authorized. The non-expiring `carrier_media4` set is empty and there are no managed media rich rules. SIP OPTIONS qualification, including the historical139ms result below, does not establish RTP connectivity. Jaintel3 remains intentionally deactivated. Do not infer media addresses from `38.147.130.74`, add them to agent grants or open RTP globally.

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

The legacy schema, an array of unique public unicast SIP IPv4 strings, remains supported. Its media collection is implicitly empty. The first media address addition converts it to the explicit object schema below; subsequent SIP/media operations preserve the other collection and any extra metadata fields. Objects must contain both arrays. Malformed JSON, invalid collection types, duplicate entries within either collection, IPv6, CIDR ranges, private/documentation/reserved, multicast, loopback, unspecified and broadcast addresses are rejected before changes. The documentation address `203.0.113.10` is rejected and was never authorized live. The config must be a regular root-owned file without group/other write permission; symlinks are rejected. Atomic replacements preserve its permission mode.

Only exact source-/32, UDP destination 5060, accept rich rules with reserved **priority -73** belong to this CLI. Do not use that exact namespace for unrelated rules. Other priorities, unmarked rules, other ports and dynamic agent grants remain untouched, even when their source IP matches a carrier. Consequently, removing a carrier does not revoke separately authorized agent access for the same IP. The nftables policy still permits only inbound UDP/5060 and outbound UDP source 5060/destination 5060 for carriers. SIP commands add no carrier RTP or 5061 access; media commands use a distinct set and rule namespace. Carrier addresses never enter agents4/grace4 and never expire with agent leases.

All CLI operations use the existing `/var/lib/alma-portal-443/worker.lock`; reads take a shared lock, mutations an exclusive lock with a ten-second acquisition deadline. Standalone guard bootstrap now takes that same lock; the imported worker already holds it and does not acquire it twice. The installed portal and persistent worker-lock file are prerequisites. No additional timer or daemon is introduced.

Mutations validate input and configuration, snapshot carrier membership and owned rules, atomically replace JSON if needed, reconcile permanent/runtime rich rules, run native guard bootstrap and verify all layers. Detected failures attempt exact restoration of the previous JSON bytes, carrier set and owned rich rules, preserving unrelated rules. A nonzero result states whether restoration succeeded or identifies remaining inconsistencies. A failed firewall query causes no speculative reset. An abrupt process/host crash cannot provide an all-layer transaction: after interruption, run `validate`, review the authoritative JSON, then `sync`. If a reported rollback is incomplete, repair the named layer and run `sync`/`validate`; do not restore the whole firewalld zone or flush nftables.

### Installation and configuration ownership

From a reviewed checkout on the target, with the existing portal installed:

```bash
sudo python3 operations/portal443/install-carriers.py
```

For the one-time migration from the original accepted PeerEdge rich rule, use `--adopt-peeredge`. This explicitly migrates **only** that exact historical rule in runtime/permanent configuration, after the marked replacement is verified. It does not infer ownership of other rules from matching IPs. Future installs normally omit this flag. Optional `--expect-guard-sha256 <INSPECTED_SHA256>` refuses a changed deployed guard. The installer first checks that effective Asterisk RTP settings match the supported10000–20000 range, then atomically installs the two modules and executable, restores SELinux contexts, synchronizes SIP/media permissions and saves a private scoped backup. It preserves the live JSON and performs no firewall reload or service restart. Keep its reviewed source directory for rollback; it must contain the CLI, guard, entry point and the existing compatible `common.py`.

**Operational authority is `/opt/alma-portal-443/carriers.json`.** `operations/portal443/carriers.json` is the source-controlled deployment inventory. After an operational add/remove, read the live JSON through authorized SSH, review the exact address-only diff, update that repository file and commit it under the normal change process. Never deploy an older repository JSON over live operational changes. This installer deliberately never copies its checkout's JSON. A separate reviewed configuration rollout must also use the same worker lock and run CLI `sync`/`validate` while accounting for rollback. There is no hidden second address inventory or automatic repository write from the server.

### Original CLI deployment and rollback (historical)

2026-11-11 17:16 UTC (2026-11-12 Asia/Manila). Source baseline `b8b165c8b7eb7e933a3450fb7f33e6c607651578`; 68 portal tests and 15 isolated CLI tests on target Python passed. Sanitized evidence: `operations/qa/evidence/carrier-cli/{tests.txt,deployment.txt,target-tests.txt}`. Private installer source is `/root/qa-backups/carrier-cli-source-20261111T171620Z`; scoped backup is `/root/qa-backups/carrier-cli-20261111T171621186352Z`.

Live `list`, `validate`, duplicate PeerEdge `add`, two `sync` calls, standalone bootstrap, normal portal reconciliation and final validation passed. One carrier remains in each layer; JSON bytes are unchanged. The 18 nft rules and agent/grace membership were unchanged; three unrelated runtime rich rules were preserved. Permanent configuration check and all six inspected services passed; calls/channels stayed zero with 142 processed calls. The SIP peer is intentionally deactivated by the operator: `Peer Jaintel3 not found.` New qualification is NOT_RUN; no peer configuration was changed or reachability claimed. Full add/remove and rollback were tested only against isolated file/mocked-firewall fixtures, never by removing PeerEdge or adding arbitrary live addresses.

Rollback of this CLI installation (not the earlier permanent carrier integration):

```bash
ssh -T -p 22222 root@2.29.153.161 \
  'python3 /root/qa-backups/carrier-cli-source-20261111T171620Z/install-carriers.py --rollback /root/qa-backups/carrier-cli-20261111T171621186352Z'
```

The script holds the worker lock, restores the exact old PeerEdge rules before removing newly owned rules, restores only the snapshotted carrier membership and affected source/entry files, then verifies read-back. It preserves other nftables sets, rich rules, SSH and services. The backup includes original JSON for evidence, but rollback refuses to run if live carrier configuration changed after installation; review and adapt the carrier-specific reversal in that case rather than discarding later changes. The source directory must be retained until rollback is no longer needed. Rollback passed an isolated execute-and-read-back test; it was not executed against the live firewall. Live code was compiled with target Python 3.12.15 before installation. A guard-service restart and firewalld reload remain unnecessary and deferred for the dependency reasons described above.


## Carrier media management

Run only with provider-confirmed public media IPv4 addresses. The following placeholders are intentionally not executable addresses:

```bash
sudo alma-portal-carriers media list
sudo alma-portal-carriers media add <MEDIA_PUBLIC_IP>
sudo alma-portal-carriers media validate
sudo alma-portal-carriers media remove <MEDIA_PUBLIC_IP>
sudo alma-portal-carriers media sync
```

All five media operations use the same lock, validation, atomic writes, exit statuses and rollback behavior as SIP operations. Removal can interrupt that carrier's media and prints a warning without an extra confirmation prompt. `list`/`validate` are read-only; `sync` repairs only the selected collection's managed permissions. A source may legitimately appear in both collections, but neither implies membership in the other. Media access never grants SIP5060/5061 access.

Explicit configuration example (no media endpoint authorized):

```json
{
  "sip": ["38.147.130.74"],
  "media": []
}
```

The live and repository JSON retain their original legacy array bytes until a media address is actually configured. No empty-schema migration is necessary. After an operational update, review and commit the live address-only JSON diff back into the repository as described above. The installer never overwrites operational JSON with its checkout copy.

### Local and remote ports

On dialer01, effective `rtp show settings` and `rtp.conf` agree on local RTP10000–20000. The current implementation supports this exact local range and rejects installation against a different effective range. It does not edit Asterisk's range or NAT settings. Non-multiplexed RTCP uses the adjacent local port; the pinned[Asterisk20.21.0 implementation](https://raw.githubusercontent.com/asterisk/asterisk/20.21.0/res/res_rtp_asterisk.c) can allocate the upper even RTP port20000 on retry and sets standard RTCP to RTP+1. Therefore carrier media permissions cover UDP10000–20001. The extra20001 is the derived RTCP boundary. Existing agent rules remain10000–20000.

The actual SDP remote port is independent of this local range. Inbound exceptions constrain **local destination** ports and verified media source addresses; outbound exceptions constrain **local source** ports and verified media destination addresses. They deliberately do not constrain the remote UDP port to Asterisk's local range:

```nft
ip saddr @carrier_media4 udp dport 10000-20001 counter return
ip daddr @carrier_media4 udp sport 10000-20001 counter return
```

Each rule appears before the agent RTP/grace DROP path, after SIP handling. `carrier_media4` has no timeout. There are20 rules total, including the original three DROP rules per direction. Both carrier verdicts remain `return`, allowing subsequent firewalld filtering. The public zone requires this exact source-scoped inbound media allowance for each verified address:

```text
rule priority="-74" family="ipv4" source address="<MEDIA_PUBLIC_IP>/32" port port="10000-20001" protocol="udp" accept
```

Only that exact rule shape and priority are managed by media operations; SIP owns priority-73/UDP5060. Do not use these namespaces for unrelated rules. The current firewalld output path permits outbound traffic after the guard; no direct OUTPUT exception or broad port allowance is needed. Agent expiration, conntrack marking and grace handling are untouched. Unauthorized sources still encounter the existing RTP10000–20000 guard DROP path and firewalld's inbound policy; this deployment introduces no global RTP acceptance.

### Obtain endpoints and complete the later call test

1. Obtain the customer-specific PeerEdge/46Labs trunk-group media configuration: Anchor Media enabled/disabled, assigned media set, all primary/redundant public RTP and RTCP addresses, remote port policy and any separately negotiated RTCP address. If only CIDRs are provided, get the exact endpoint inventory or request a separate reviewed CIDR feature; this CLI accepts individual IPv4 addresses only.
2. Review provider evidence against authorized topology. Jaintel3 is currently absent from runtime, so its effective `directmedia` and actual SDP cannot be established. Confirm Asterisk media anchoring (`directmedia=no`) and effective NAT configuration when a separately authorized carrier activation is performed. Do not change routing or reactivate it as part of firewall management.
3. Add only verified endpoints with `media add`; run both SIP and media `validate`. Preserve the source-controlled inventory. A successful empty media validation proves configuration alignment, not working carrier audio.
4. Under separate call/activation authorization, record the test start, initial source-scoped rule counters and effective local RTP settings. Capture only the approved peer/media hosts with a short, bounded `tcpdump` capture on eth0; keep SIP/authentication content and audio private, and retain only sanitized endpoint/SDP summaries in Git.
5. For one controlled call, correlate INVITE/200OK SDP `c=`/`m=` and RTCP attributes with actual packet IPs/ports in both directions. Inspect the guard's media and DROP counters, confirm media crosses Asterisk, and obtain audible confirmation from both endpoints. Packet counts alone do not prove usable audio. Successful SIP OPTIONS alone proves neither RTP nor audio.
6. End the call, return the carrier to its approved operating state, validate SIP/media permissions and confirm normal agent authorization/expiration. Publish separate actual RTP and audio outcomes. No such call or capture was performed in this deployment.

### Media deployment, verification and rollback

Deployed2026-11-11 18:45UTC (2026-11-12 Asia/Manila), from baseline `84beb29cd69f86b924f6302eb9b61276994afe3f`. See[the media acceptance report](../qa/ALMADIAL_CARRIER_MEDIA_ACCEPTANCE.md) and `operations/qa/evidence/carrier-media/`. All80 Python portal tests, seven browser-JavaScript checks, and27 target Python focused tests passed. Real nftables migration, media add/remove, bootstrap and rollback also passed inside a disposable network namespace; firewalld effects in destructive tests were mocked. The live installation exercised empty media, unchanged PeerEdge SIP permissions, standalone bootstrap and the actual reconciliation service without restarts or firewall reloads.

Private reviewed source: `/root/qa-backups/carrier-media-source-20261111T184306Z`. Scoped backup: `/root/qa-backups/carrier-cli-20261111T184553704474Z`. To reverse **this media deployment**:

```bash
ssh -T -p 22222 root@2.29.153.161 \
  'python3 /root/qa-backups/carrier-media-source-20261111T184306Z/install-carriers.py --rollback /root/qa-backups/carrier-cli-20261111T184553704474Z'
```

This holds the worker lock, restores only the previous SIP/media managed rules, prior owned guard chains/static media set and affected installation files. It preserves agents4/grace4 handles and kernel deadlines, all unrelated rich rules, live SIP JSON, SSH and services. The JSON backup is evidence; rollback refuses if live configuration changed since deployment. Review later operational edits before reversing them rather than discarding them. The chain restoration is syntax-checked against nft before applying and its semantics are read back afterward. Both unit execute/read-back and real kernel scoped policy rollback passed in isolation. A live rollback was unnecessary and was not performed.

For an interrupted CLI update, inspect both `validate` outputs, review authoritative live JSON and run the matching `sync`; a failed operation returns nonzero and reports rollback success or remaining inconsistencies. Keep the backup and reviewed installer source available. Never restore the whole public zone, flush the ruleset or use runtime-to-permanent to repair carrier state.

Implementation: `ALMADIAL_CARRIER_MEDIA_IMPLEMENTATION_PASS`.
Live media: `PEEREDGE_RTP_NOT_TESTED`.
