# AlmaDial carrier RTP/RTCP media integration

Execution: 2026-11-11 UTC / 2026-11-12 Asia/Manila. Developer verification, not independent QA.

## Outcome and acceptance boundary

Separate carrier media management is implemented and deployed on dialer01. The existing SIP CLI remains compatible. The live media allowlist is **empty**: no provider-confirmed media addresses or carrier-call authorization were available. No carrier was activated, no call was originated by this task, and no packet/audio capture was taken. Empty media validation establishes configuration alignment only.

Implementation verdict: `ALMADIAL_CARRIER_MEDIA_IMPLEMENTATION_PASS`.
Live media verdict: `PEEREDGE_RTP_NOT_TESTED`.

## Baseline and discovery

Repository baseline: `84beb29cd69f86b924f6302eb9b61276994afe3f`, newer than the accepted SIP CLI commit `bcbc9057f6b5054ba1daf552ac9514d569f3b5e8`. Tracked files were initially clean; unrelated untracked work was retained. No reset, push, package installation or framework addition occurred. Target was accessed with grouped noninteractive root SSH on port 22222. No shell history or auditing setting was changed.

Evidence: [predeploy.txt](evidence/carrier-media/predeploy.txt).

| Item | Actual finding |
|---|---|
| Target | dialer01 / 2.29.153.161 |
| Asterisk | 20.21.0, existing chan_sip |
| Effective local RTP | `rtp show settings`: start 10000, end 20000; agrees with `/etc/asterisk/rtp.conf` |
| RTP configuration | `[general] rtpstart=10000, rtpend=20000`; existing `[ice_host_candidates]` section |
| RTP behavior | Strict RTP Yes; probation 4; replay protection Yes; ICE Yes; STUN disabled |
| SIP listener | UDP 0.0.0.0:5060 |
| Global SIP media/NAT settings | Direct RTP setup No; SIP remapping Disabled; Externhost none; Externaddr null; Localnet 192.168.0.0/16; Symmetric RTP Yes; RTCP mux No |
| Jaintel3 | `Peer Jaintel3 not found.` Operator-deactivated; preserved |
| Effective trunk directmedia / actual media topology | NOT_RUN: inactive peer and no authorized call; global settings do not prove per-peer media anchoring |
| Existing guard | `inet alma_portal_sip`, input/output priority -10; original 18 rules, v2 carrier policy |
| Other firewall layer | public/eth0; firewalld input/output priority +10, output policy accept |
| SIP authorization | carriers4 contains only 38.147.130.74; exact source-/32 UDP5060 rich rule, priority -73, runtime and permanent |
| Dynamic access | agents4/grace4 timed membership, SIP5060/5061 and RTP10000–20000 restrictions |
| Ownership | `/opt/alma-portal-443/sip_guard.py`; existing guard service and ten-second apply worker |

Current local configuration/documentation did not identify customer-specific PeerEdge media IPs, a media set, Anchor Media state or remote media-port policy. No provider credential was accessed. The signaling IP is not evidence of a media endpoint. The operator must obtain the assigned primary/redundant RTP/RTCP endpoint inventory and media policy from PeerEdge/46Labs before authorization. Individual verified public IPv4 addresses are supported; CIDR authorization is outside this implementation.

### RTP and RTCP port reasoning

Asterisk's local range is not the remote negotiated UDP range. Inbound media matches a verified source plus the **local destination** port; outbound media matches a verified destination plus the **local source** port. No remote port is incorrectly constrained to 10000–20000.

The pinned [Asterisk 20.21.0 RTP implementation](https://raw.githubusercontent.com/asterisk/asterisk/20.21.0/res/res_rtp_asterisk.c), `rtp_allocate_transport` and standard RTCP setup, permits the upper even RTP port on allocation retry and assigns standard RTCP to RTP+1. Carrier media rules therefore cover local UDP **10000–20001**, including the derived RTCP boundary. This is not a change to Asterisk's range. Existing agent rules remain exactly 10000–20000. Actual call SDP/RTCP behavior still requires live qualification.

## Implementation and architecture

The authoritative live inventory remains `/opt/alma-portal-443/carriers.json`. The existing legacy SIP array remains valid and unchanged on disk. Its media collection is implicitly empty. A media addition converts it atomically to:

```json
{"sip": ["38.147.130.74"], "media": []}
```

Both arrays are validated; updates preserve the other collection and extra object metadata. Addresses must pass the existing public unicast IPv4 validation; duplicates within a collection and unsuitable addresses are rejected. The same IP may be explicitly authorized for both roles, without either role implying the other.

| Category | nftables set | Managed firewalld namespace | Scope |
|---|---|---|---|
| SIP carrier | carriers4, no timeout | priority -73, source /32, UDP5060 | Inbound dport5060; outbound sport5060/dport5060 |
| Media carrier | carrier_media4, no timeout | priority -74, source /32, UDP10000–20001 | Inbound local dport; outbound local sport; remote port unconstrained |
| Agent | agents4, timed | Existing portal rules | Existing SIP/RTP authorization |
| Grace | grace4, timed | Existing portal rules | Existing marked established media policy |

New rules, before the existing agent RTP/grace DROP path:

```nft
ip saddr @carrier_media4 udp dport 10000-20001 counter return
ip daddr @carrier_media4 udp sport 10000-20001 counter return
```

They follow SIP processing, so media membership does not authorize SIP5060/5061. `return` preserves subsequent firewalld enforcement. The source-scoped media rich rule is synchronized in both runtime and permanent public-zone configuration. Current firewalld outbound policy needs no added OUTPUT rule. There is no unrestricted RTP acceptance, implicit SIP-to-media authorization, new daemon, schema/database change or change to agent marking/expiration. The existing RTP guard DROP rules and firewalld inbound restrictions remain; no new unauthorized-packet probe was sent.

Policy marker `alma-phone-sip-5060-5061-v3-carrier-media` installs the extra set and atomically migrates only the two owned guard chains. Dynamic sets are neither flushed nor recreated. Normal reconciliation updates both static collections; a CLI mutation updates only its selected collection. Twenty rules are present after migration, ten in each chain, including the original three DROP rules per direction.

### Exact implementation files

- `operations/portal443/sip_guard.py`: compatible schema parsing, static media set, ordered media exceptions and scoped reconciliation.
- `operations/portal443/carrier_cli.py`: nested media commands using the existing lock, atomic writes, validation, firewall snapshots and rollback; distinct rule ownership; scoped policy snapshot/reversal for installer rollback.
- `operations/portal443/install-carriers.py`: effective RTP-range prerequisite, both-collection reconciliation and verified policy/rule/file rollback.
- `operations/portal443/tests/test_carrier_media.py`: focused configuration/firewall/CLI/concurrency/installer tests.
- `operations/portal443/tests/kernel_carrier_media.py`: opt-in real-kernel tests confined to a disposable empty network namespace; refuses the host namespace.
- Existing `test_carrier_cli.py` and `test_sip_guard.py`: updated fixtures for the current policy and media-aware installer.
- `operations/portal443/PEEREDGE_CARRIER_FIREWALL.md`: operator guide, discovery requirements, commands and rollback.
- This report and sanitized evidence under `operations/qa/evidence/carrier-media/`.

`carriers.json`, the CLI entry point, portal worker/common modules, systemd units and all Asterisk/VICIdial files were not changed by this implementation.

### Operations and failure handling

```bash
sudo alma-portal-carriers media list
sudo alma-portal-carriers media add <VERIFIED_MEDIA_PUBLIC_IP>
sudo alma-portal-carriers media remove <VERIFIED_MEDIA_PUBLIC_IP>
sudo alma-portal-carriers media validate
sudo alma-portal-carriers media sync
```

Old SIP commands are unchanged. Shared worker locking serializes JSON mutations with reconciliation and other CLI operations. Only exact managed rich-rule shapes are reconciled, preserving unrelated rules even for matching source IPs. Failures restore previous JSON bytes, selected set membership and selected managed rich rules; an incomplete rollback reports remaining errors and returns nonzero. A process/host crash is not an atomic multi-firewall transaction: review live JSON, run both validations and sync the affected collection. No hidden inventory or recurring recovery worker was added.

Operational changes must be reviewed back into the repository's carrier inventory before a later configuration rollout. The installer deliberately preserves live JSON rather than copying an older checkout over operator changes.

## Automated verification

| Check | Result | Evidence |
|---|---|---|
| Complete Python portal suite | PASS: 80 tests | [local-python-tests.txt](evidence/carrier-media/local-python-tests.txt) |
| Browser-JavaScript syntax and behavioral checks | PASS: syntax plus seven checks | [local-js-tests.txt](evidence/carrier-media/local-js-tests.txt) |
| Target Python 3.12 focused SIP/media suites | PASS: 27 tests | [target-isolated-tests.txt](evidence/carrier-media/target-isolated-tests.txt) |
| Actual nft syntax and old-policy migration | PASS in disposable network namespace | Same target evidence |
| Actual nft media add, duplicate add, sync, remove | PASS in disposable namespace; firewalld mocked | Same target evidence |
| Scoped old-policy rollback | PASS: real nft JSON transaction and read-back; dynamic handles retained | Same target evidence |
| Fresh bootstrap and repeated bootstrap | PASS in disposable namespace with configured media | Same target evidence |
| Python compilation / diff whitespace | PASS | Executed locally; target installer source compiled before deployment |

Tests cover invalid/unsuitable inputs, malformed configuration, duplicates, no-op behavior, metadata and other-collection preservation, atomic replacement, runtime/permanent drift, stale managed-rule removal, scoped firewall generation, failure after partial application and reply loss, incomplete rollback reporting, concurrent media adds and SIP-add/media-remove, and contention on the worker's shared lock. Existing portal tests cover agent/grace expiration and authorization. Destructive lifecycle tests use temporary configuration and injected firewalld state. Fake public addresses were used only in mocks or the disposable namespace, never in host authorization.

The kernel script needs an `old_sip_guard.py` alongside the staged modules. Recreate it from this task's exact baseline, then run only in the isolated namespace:

```bash
git show 84beb29cd69f86b924f6302eb9b61276994afe3f:operations/portal443/sip_guard.py > <PRIVATE_STAGE>/old_sip_guard.py
sudo unshare -n python3 <PRIVATE_STAGE>/tests/kernel_carrier_media.py
```

## Deployment and persistence

Reviewed source stage: `/root/qa-backups/carrier-media-source-20261111T184306Z`.
Scoped installer backup: `/root/qa-backups/carrier-cli-20261111T184553704474Z`.

The existing installer was invoked with the inspected old guard digest, under the existing worker lock. Each generated nft transaction was syntax-checked before actual application. Agent/grace set handles, members, configured timeouts and decreasing expiration deadlines were compared around migration and preserved. No Asterisk, SSH, network, firewalld or portal service was restarted, and no firewall reload or runtime-to-permanent operation occurred.

Evidence: [deployment.txt](evidence/carrier-media/deployment.txt), [verification.txt](evidence/carrier-media/verification.txt).

| Artifact | Installed SHA256 |
|---|---|
| sip_guard.py | `2eb256d4b731ffc6e58d5dff8d36cf4ba9dbe85b9f01e4a1d884d64626d23ff5` |
| carrier_cli.py | `8452d8dc907ecdc64cdb9583da4801f018c7782cd5e123df2d1fd8a9521ad26b` |
| CLI entry point, unchanged | `c5c2b28f532dc173f6180b74c69e8cb6daa7f1b4be041b3142e4fc976d2f17d3` |
| Live carriers.json, unchanged bytes | `0fcc9b5ba559f72068de55ab0a2d96f2785d366cac0cea1269b130cb0f017eff` |

Live list/validate passed for SIP and media. Duplicate SIP add, SIP sync, two media syncs, standalone guard bootstrap and the actual `alma-portal-443-apply.service` path passed. The oneshot returned Result=success/ExecMainStatus=0. Read-back confirmed exactly 20 rules, unchanged SIP configuration and runtime/permanent rich rules, empty static media membership without timeout, no managed media rich rules, unchanged dynamic membership and three DROP rules per direction. Firewalld configuration validation passed. This exercises the actual bootstrap and reconciliation entry points, not a simulated service.

A guard-service restart, firewalld reload and reboot were unnecessary and not performed. The guard has dependency relationships with Asterisk/firewalld; a separate maintenance window is required to test a real stop/start without disrupting calls or expiring grants.

## Acceptance matrix

| Implementation criterion | Result and evidence |
|---|---|
| Existing SIP permissions preserved | PASS: JSON bytes, SIP nft members and runtime/permanent rules unchanged |
| Separate explicit RTP/RTCP configuration | PASS: validated object schema and independent carrier_media4 |
| All five media CLI operations | PASS: focused tests, real nft lifecycle in isolation; live list/validate/sync |
| nft/firewalld consistency | PASS: lifecycle tests and live empty-media/SIP read-back |
| Persistence through reconciliation | PASS: actual standalone bootstrap and apply service; isolated fresh bootstrap |
| Agent expiration unchanged | PASS: real migration retained handles/deadlines; portal expiration regressions pass |
| Unauthorized media remains restricted | PASS: ordered guard DROP rules and firewalld source scope preserved; no global allow rule; no external traffic probe |
| Existing CLI compatibility | PASS: all prior carrier tests, old JSON and live SIP no-op/sync |
| Focused and portal tests | PASS: counts above |
| Deployment | PASS: installed digest and scoped host assertions |
| Rollback validated | PASS in isolation, including real nft syntax/application and read-back; not executed live |
| Documentation | PASS: operator guide updated |

| Live media gate | Result |
|---|---|
| Verified PeerEdge media endpoint information | NOT_RUN: provider inventory unavailable |
| Authorized carrier call | NOT_RUN: no carrier activation/call authorization |
| Bidirectional RTP observed | NOT_RUN |
| Bidirectional audio confirmed | NOT_RUN |
| No carrier media drops during call | NOT_RUN |

## Preservation and final state

Hash comparisons preserved `/etc/asterisk/rtp.conf`, `sip.conf`, `sip-vicidial.conf`, `pjsip.conf`, `extensions.conf`, firewalld `public.xml`, portal `common.py` and `apply_requests.py`. Existing Asterisk/firewalld/httpd process IDs and all pre-existing rich rules remained unchanged at deployment. No upstream code, database, phone configuration, carrier configuration, recording or routing state was edited.

SIP carriers4: only 38.147.130.74. Media carrier_media4: empty. Jaintel3 remains absent/deactivated. Agent/grace membership remains under normal portal control. At deployment, channels/calls were zero and processed calls 163. During later read-back an existing call briefly appeared (two channels/one call, count164); this task issued no originate or hangup and did not intervene. A subsequent bounded read-only check found no channels or conference members: [final-session-state.txt](evidence/carrier-media/final-session-state.txt). This observation is recorded rather than claiming the host processed no unrelated calls throughout verification.

## Scoped rollback

Before installation: guard SHA256 `e58e3afe58305f6a5a685147c5f51b8cd6fa0612948d691a65fbf16379a1eb60`; CLI SHA256 `cf11cd729da2f99ba6b7b1090b7d21c0807201dc1024475e6ea4a56ea83eb4cb`. The private manifest preserves prior files, owned rules and scoped policy state.

```bash
ssh -T -p 22222 root@2.29.153.161 \
  'python3 /root/qa-backups/carrier-media-source-20261111T184306Z/install-carriers.py --rollback /root/qa-backups/carrier-cli-20261111T184553704474Z'
```

This restores affected installation files, pre-change owned chain semantics/static media set and managed SIP/media rich rules. It leaves dynamic sets, native deadlines, unrelated rules and SSH untouched. The installer checks unchanged live JSON before reversal; if operational configuration changed, review the carrier-specific changes first. Do not overwrite later updates with backup JSON. The rollback transaction is nft-syntax-checked and semantically read back. No whole-table/ruleset flush, public-zone restore or service restart is used. Unit installer rollback and real-kernel scoped-policy reversal passed; live rollback was not necessary.

After reversal, run the original SIP list/validate, inspect dynamic membership and normal service health. The old CLI intentionally lacks media commands. Retain the reviewed source/backup until rollback is no longer needed.

## Remaining qualification and operator action

Obtain provider-confirmed media/RTCP endpoints and Anchor Media configuration; approve carrier activation and one controlled call separately. Add only confirmed endpoints, validate both collections, correlate sanitized SDP and real RTP endpoints/counters, and confirm audible conversation in both directions. Follow the bounded checklist in the operator guide. Existing SIP qualification is historical; it was not repeated against a deactivated trunk.

No RTP ranges or carrier endpoints are auto-discovered. This implementation supports individual public IPv4 addresses and the inspected local RTP range only. RTCP source/address behavior and actual PeerEdge topology require call evidence. Abrupt host failure still requires normal validate/sync recovery. Advanced carrier routing, CIDRs, IPv6 and end-to-end audio certification are outside this change.

Implementation: `ALMADIAL_CARRIER_MEDIA_IMPLEMENTATION_PASS`.
Live media: `PEEREDGE_RTP_NOT_TESTED`.
