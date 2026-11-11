# AlmaDial carrier CLI — implementation and verification

## Outcome and baseline

The five-command administrative CLI is installed on dialer01 at `/usr/local/sbin/alma-portal-carriers`. Existing PeerEdge firewall authorization is intact. Full add/remove, injected failure and rollback tests used isolated files and a mocked firewall; production verification used only the existing address, read-only checks and idempotent synchronization. No arbitrary test address was authorized live.

Repository starting HEAD: `b8b165c8b7eb7e933a3450fb7f33e6c607651578`. Tracked tree was clean; unrelated untracked work was retained. Target: AlmaLinux 10.2, system Python 3.12.15, existing public/eth0 firewalld and `inet alma_portal_sip`. Local tests used Python 3.14. No packages installed. This is a bounded portal enhancement, not general AlmaDial installer or stack acceptance.

The operator confirmed Jaintel3 was intentionally deactivated. `sip show peer Jaintel3` consequently reports `Peer Jaintel3 not found.` SIP reachability/qualification is NOT_RUN, not a CLI failure or a verified live trunk. No native SIP peer configuration was altered.

## Changes and ownership

| File | Purpose |
|---|---|
| `operations/portal443/carrier_cli.py` | List, validate, add, remove, sync; atomic JSON, shared lock, scoped firewall reconciliation, rollback and read-back |
| `operations/portal443/alma-portal-carriers` | Fixed-path system-Python executable entry point |
| `operations/portal443/install-carriers.py` | Existing-portal deployment, optional explicit legacy PeerEdge adoption, backup and guarded rollback |
| `operations/portal443/sip_guard.py` | Reuse stricter IPv4/schema validation; lock standalone bootstrap on existing worker lock |
| `operations/portal443/tests/test_carrier_cli.py` | Fifteen focused tests using isolated files and mocked firewall commands |
| `operations/portal443/PEEREDGE_CARRIER_FIREWALL.md` | Current rule ownership, installation, commands, recovery, live/source configuration workflow and rollback |
| This report and `operations/qa/evidence/carrier-cli/` | Sanitized verification evidence |

The JSON array schema is unchanged. Live `/opt/alma-portal-443/carriers.json` remains the operational source of truth, byte-for-byte unchanged at SHA256 `0fcc9b5ba559f72068de55ab0a2d96f2785d366cac0cea1269b130cb0f017eff` and contains only `38.147.130.74`. CLI changes must be reviewed back into repository `carriers.json`; deployment preserves live JSON instead of overwriting it with checkout defaults.

Managed firewalld rules use this exact reserved namespace:

```text
rule priority="-73" family="ipv4" source address="<PUBLIC_IPV4>/32" port port="5060" protocol="udp" accept
```

Only exact rules of that shape are reconciled or removed. The installer explicitly adopted the historical unmarked PeerEdge rule after verifying its replacement. Other rich rules, agent grants and grace memberships remain outside CLI ownership. Outgoing carrier permission remains UDP source 5060/destination 5060 in the existing guard; no RTP/5061 permission is added. Guard rule expressions and policy marker are unchanged.

Mutations and standalone bootstrap share `/var/lib/alma-portal-443/worker.lock` with the ten-second worker. Snapshots and writes occur under an exclusive lock; list/validate take a shared lock. Config replacement uses same-directory temporary files, fsync and atomic replace. Detected errors restore old bytes and managed firewall state; failed restoration returns a nonzero result with remaining inconsistencies. Hard process/host interruption still requires validate/sync against the authoritative JSON; no multi-layer crash-atomicity claim is made.

## Verification matrix

| Requirement | Result | Evidence |
|---|---|---|
| All five commands | PASS | Fifteen focused tests cover both mutation directions, no-ops, list, validate and sync; existing-address commands also executed live |
| Input/schema/ownership validation | PASS | Invalid, unsuitable, IPv6, duplicate, invalid JSON/object schema, permissions and symlink tests; invalid input leaves bytes intact |
| Atomic updates and preservation | PASS | Atomic replacement/mode/cleanup/failure tests; existing carrier order and entries retained; unsupported object schemas rejected without rewriting |
| Complete add/remove firewall lifecycle | PASS, mocked | Desired nft membership plus runtime/permanent rules; selected removal preserves agents, unrelated rules and remaining carriers |
| Scoped ownership and port limits | PASS | Actual command-generation tests; live exact marked rule in both layers; all 18 nft rules unchanged |
| Idempotence and drift repair | PASS | Mocked drift repair/stale removal and live existing-address add plus two syncs; one rule per layer |
| Partial failure and rollback | PASS, mocked | Failures after rich-rule/nft mutation restore prior bytes and prior drift; rollback failure explicitly reported; installer reversal read-back tested |
| Concurrent add/add and add/remove | PASS, mocked | Separate spawned processes serialize on the same lock, preserving both changes |
| Worker coexistence | PASS | Lock-contention tests plus native bootstrap and actual reconciliation service; subsequent timer cycle success |
| Existing PeerEdge firewall authorization | PASS | Non-expiring carriers4 and exact runtime/permanent rule; address remains independent of agent grants |
| Existing agent restrictions | PASS, bounded comparison | Existing 1 agent/1 grace member and 18 rule expressions unchanged; 3 unrelated runtime rich rules retained |
| Installed CLI/source consistency | PASS | Target compiles, live operations, installed hash read-back, root755 entry and root644 modules/config |
| Persistence | PASS, supported paths | Native bootstrap, actual worker, subsequent timer cycle, permanent config check; no reboot/reload required |
| Scoped rollback | PASS, isolated/syntax | Execute-and-read-back installer test; real-kernel `nft -c` carrier-set restoration check; live rollback NOT_RUN |
| SIP peer qualification | NOT_RUN | Operator intentionally deactivated Jaintel3; no configuration change or call attempted |
| Service restart/full firewall reload | NOT_RUN | Unnecessary; guard dependency chain/reload risks documented in firewall report |

Local full portal suite: **68 tests passed**. Command:

```bash
python3 -m unittest discover -s operations/portal443/tests -v
```

Target system Python: **15 focused tests passed** using the same source in a private staging directory. No test called a real firewall mutation backend. Initial target test run exposed default-fork inheritance of the parent test's lock; tests now explicitly use `spawn` to model independent CLI processes on both Python versions. The deployed implementation did not require alteration. The initial failed test trace is retained as `target-tests-first.txt`; final success is in `target-tests.txt`.

Evidence files:

- `operations/qa/evidence/carrier-cli/tests.txt`: complete 68-test result.
- `operations/qa/evidence/carrier-cli/deployment.txt`: baseline, installer backup/hash output, all live commands and invariants.
- `operations/qa/evidence/carrier-cli/target-tests.txt`: target 15-test result, rollback syntax check, final permissions and post-timer validation.
- `operations/qa/evidence/carrier-cli/target-tests-first.txt`: diagnosed test-harness failure, retained for provenance.

## Deployment and final runtime state

Deployment: 2026-11-11 17:16 UTC, through grouped noninteractive root SSH22222. Installer source: `/root/qa-backups/carrier-cli-source-20261111T171620Z`. Scoped backup: `/root/qa-backups/carrier-cli-20261111T171621186352Z`. The installer required the inspected old guard hash, checked the unchanged common module and JSON, and explicitly migrated only the historical PeerEdge rule. No service restart or firewalld reload.

Installed SHA256 values:

- `sip_guard.py`: `e58e3afe58305f6a5a685147c5f51b8cd6fa0612948d691a65fbf16379a1eb60`
- `carrier_cli.py`: `cf11cd729da2f99ba6b7b1090b7d21c0807201dc1024475e6ea4a56ea83eb4cb`
- `alma-portal-carriers`: `c5c2b28f532dc173f6180b74c69e8cb6daa7f1b4be041b3142e4fc976d2f17d3`

Final list:

```text
38.147.130.74  Configured  nftables=OK  runtime=OK  permanent=OK
Validation: OK
```

`firewall-cmd --check-config`: success. Firewalld, Asterisk, httpd, portal, apply timer and guard all active. Reconciliation `Result=success`, `ExecMainStatus=0`, with subsequent native timer exit recorded. Zero active channels/calls before and after; processed calls unchanged 142. No Asterisk, VICIdial, portal authentication, RTP, WSS, SSH, TLS, recording or backup configuration was changed.

## Rollback and change control

Use the retained installer source and backup:

```bash
ssh -T -p 22222 root@2.29.153.161 \
  'python3 /root/qa-backups/carrier-cli-source-20261111T171620Z/install-carriers.py --rollback /root/qa-backups/carrier-cli-20261111T171621186352Z'
```

Rollback first restores the old scoped PeerEdge permissions, then previous managed rules/carrier membership and only the affected files. It never restores a whole zone or resets agents. It refuses changed live carrier JSON, avoiding accidental loss of subsequent operational changes. See the firewall document for recovery of a failed CLI update and repository propagation. Rollback was tested in isolation and its carrier-set transaction syntax checked on the real kernel; it was not applied live.

Only the paths listed above belong to this change. Unrelated untracked project work is preserved. One focused implementation commit is recorded in the final handoff; no push is authorized or performed. These are developer verification results, not independent QA certification.

Final verdict: `ALMADIAL_CARRIER_CLI_PASS` — CLI and firewall authorization scope. Fresh SIP qualification remains NOT_RUN because the operator deactivated the peer.
