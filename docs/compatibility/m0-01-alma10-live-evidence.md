# M0-01 AlmaLinux 10.2 live inventory and recovery evidence

Recorded 2026-11-05 UTC. Task **ALM-1 / M0-01**, `p8urvq3yjtc76rktih9xlhe3`, project `ghritpte6e40o5ayktgz0vh1`.

**ALMADIAL_M0_HOST_BASELINE_READY_FOR_QA.** The later firewall-closure assignment resolved the mandatory kernel-ruleset observation. ALM-1 is now **QA Review**, assigned to Codex, with only the M0 label; its Blocked attachment was removed and read back. This is implementation-side handoff, not independent acceptance or a verified telephony stack. No Accepted transition occurred. The earlier [no-VM attempt](m0-01-live-host-baseline.md) and original run below remain as history.

## Firewall closure and final inventory — 2026-11-05 UTC

The operator's assignment, “ALM-1 Firewall Evidence Closure and QA Handoff,” explicitly authorized installing the nftables inspection package and required dependencies on `almadial-alma10`, read-only queries, collector transfer and protected evidence. Local authorization reference: `operator-assignment-ALM-1-firewall-20261105T015153Z`; this records the operator assignment, not an external authorization ID. VM identity, AlmaLinux 10.2 x86_64, Multipass 1.17.0/QEMU and ownership of `almadial-alma10.clean-baseline` were reverified. The snapshot was **not restored again**.

At entry, `/usr/sbin/nft` and **nftables 1:1.1.5-6.el10_2.x86_64 were already present**. No installation was performed by this worker. Read-only DNF history transaction 1 records successful `install -y nftables` at 2026-11-05 09:47:52 guest local time (01:47:52 UTC), installing nftables and libnftnl 1.3.0-3.el10.x86_64 from BaseOS. This predates this closure's first inspection at 01:51:53Z. Installation actor/authorization is not inferred from package history. Firewalld was not installed or repaired.

| Actual command | Exit | Observation |
|---|---:|---|
| `sudo -n nft -j list ruleset` | 0 | JSON contains one `metainfo` object, nft version 1.1.5/schema 1; **zero ruleset objects**, hence no nftables tables/chains/rules |
| `sudo -n nft list ruleset` | 0 | Empty stdout, empty stderr; agrees with the successful JSON query |
| `systemctl is-active nftables firewalld` | 3 | Both inactive; separate property queries distinguish loaded/disabled nftables from not-found firewalld |
| `rpm -q nftables firewalld` | 1 | nftables present; firewalld explicitly not installed |

First JSON inspection completed at **01:52:14Z**. Repeated JSON/text queries after collection agree exactly. `nftables.service` remained loaded, disabled, inactive/dead before and after this work; `firewalld.service` remained not-found/inactive/dead. SELinux remained Enforcing. No service activation, rule mutation, network/SSH change or telephony installation was performed. The previous run could not inspect rules, so no before/after equality with a pre-install kernel ruleset is claimed. An empty nftables ruleset does not establish upstream NAT/security-group policy, reachability or legacy iptables rules; it is an observation, not approval of a release security posture. Firewalld remains a future security/integration requirement under the plan, not something to force-install during this inventory.

Protected final evidence: **`~/almadial-lab/evidence/m0-01/firewall-20261105T015153Z/`**, 0700 directories / 0600 evidence files. The JSON/text command records include exact argv, UTC start/finish, stdout, stderr and exit codes. Use `firewall-observation-verified.json` and `firewall-summary.md` for the supplemental observation. An initial derived draft, `firewall-observation.json`, incorrectly attributed installation to this worker; it is retained for transparency and explicitly superseded by the verified record. Raw command records always showed the utility already present. No raw host-specific inventory is stored in Git.

The collector remains unchanged from the prior correction: SHA-256 **`07a3bd79ca43a59a1ee55fc54adcce9d2964a83edf831d6cf827f7929973f282`**, HEAD `18b97eea1438e0d1be72276fd3018659a33a636f` plus the recorded, uncommitted RPM/time correction patch `2cb2bfd251875be32248e651eea729f60b6552124e4674a6f2f7eee7e91f8014`. Collector, tests and fixture digests still match the prior corrected sources. No additional defect was observed; no source, fixture or test was changed during this closure. The existing collector correctly reports an absent firewalld provider; direct privileged nftables evidence supplements it without changing its 23-probe manifest, executing sudo inside it, or relabelling unavailable data.

Equivalent actual final invocation (private guest home replaced by `$HOME`):

```bash
python3 "$HOME/almadial-firewall-20261105T015153Z/collect_host_inventory.py" \
  --collect --host-ref almadial-alma10 \
  --authorization-ref operator-assignment-ALM-1-firewall-20261105T015153Z \
  --snapshot-ref almadial-alma10.clean-baseline \
  --output-dir "$HOME/almadial-firewall-20261105T015153Z/inventory-final"
```

Collection started **2026-11-05T01:53:19Z**, exit **1**: **23 probes, 20 COLLECTED / 3 UNAVAILABLE / 0 ERROR / 0 NOT_RUN**, mode HOST_COLLECTION, every `simulated=false`. The unavailable IDs are exactly `firewalld`, `firewall_zones` and `firewall_rules` (`command_not_found`); no active firewalld zones/rules are asserted. The remaining 20 IDs and statuses match the earlier corrected-run table below. The mandatory kernel firewall observation is now measured by the separate successful nftables queries, rather than by those unavailable provider-specific probes. The unchanged collector intentionally returns partial exit 1; this is not an all-probe success claim. JSON and Markdown match exactly, all 23 expected IDs are present once, and guest/host output hashes agree. JSON host acceptance, compatibility and independent QA remain NOT_RUN pending the separate reviewer.

| Final artifact | SHA-256 |
|---|---|
| inventory-final/inventory.json | `cdc2164ac2435834de45b2f3eef8a57cb943532b2385c13547d142f708e469c0` |
| inventory-final/summary.md | `1049c7e75e35bdc9d0a047ffde3a4f36eb966e47b0651737297dabb187f2dd84` |
| nft-json.json (command record) | `5fff4344a95e0920ad58c36f487920a0d9060954bf0cb488e48d5bdf34a311d2` |
| firewall-observation-verified.json | `f1357821179efb8c6ea1bd9e99f9be72778b3a9be732d96874d37e52e8ad6f8a` |

Direct guest OS/kernel/CPU/SELinux/RPM observations agree with final inventory. The package probe correctly records exit 8 with eight absent operands; the time probe records synchronized NTP/RTC values using repeated property flags. There is no regression of either earlier correction. Volatile metrics and per-run HMAC identities are not compared as byte-identical facts. Exact records and comparisons are in `crosscheck-verification.json`; existing broader network/storage/service cross-checks remain in the prior package.

All **67 files** in the prior `20261105T013428Z/SHA256SUMS.final` were rehashed unchanged. That manifest's SHA-256 remains `0fd7ed19b4719ea613972d64caa57d4ca272566abe921ef6796e8b1119ef3595`. Prior corrected-run hashes remain `7e02e7d91ce229312f8a698f62b754be9f993f6882297d84440d310533b9bc42` (JSON) and `923aa5bb7de263c9ac09ccfa2715d584d6b0b11020b2e2f9cbd9d59c61e187f4` (Markdown). Successful recovery evidence is reused unchanged, not rerun. The current guest includes the already-installed inspection packages and is not asserted byte-identical to its clean snapshot.

Scaffold validation passed; all **24** existing unit tests passed, including the actual RPM/time regressions; py_compile and `git diff --check` passed. No new code defect or regression test was needed. Existing modifications and untracked evidence reports were preserved; no commit/push occurred.

Kaneo evidence comment **`gee7ljvyenf31ie77luzr273`** was created and read back with exact content equality; all three earlier comments remain unchanged. Only Blocked attachment `vqf1t54vdisps4occspb530u` was detached, preserving M0 attachment `m1i9dbovp0npawjf08n3oy24`. QA Review (`in-review`, column `in73pv6vxczf7usmb8c03msz`) was then read back with Codex owner `aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`, unchanged task identity/description/acceptance, and the existing ALM-1 blocks ALM-2 relation. No task was self-accepted or recreated.

No implementation-side mandatory observation gap remains. Next assignment: independent review of the earlier RPM/time correction, all three live runs, final supplemental nftables evidence, source/output hashes, image provenance and preserved successful restoration. Reviewer must retain implementer ownership and independently reproduce critical observations under exact-target authority; this worker grants no acceptance. M0 remains open, the stack lock remains candidate, and ALM-2 must wait for ALM-1 acceptance and its own assignment.

## Original live run — preserved historical report

The sections below record the earlier assignment before the firewall-closure authorization. Their blocked verdict, no-install boundary and next-gap recommendation describe that earlier point in time; the later verified handoff above supersedes current-state statements without changing the original observations.

## Authorization and protected evidence

The operator's assignment, “M0-01 Live Host Inventory, Evidence Validation, and QA Handoff,” explicitly authorized inspection, collector transfer, protected evidence writes and bounded recovery on **`almadial-alma10` only**. Locally recorded reference: `operator-assignment-ALM-1-multipass-20261105T013428Z`. Its protected `authorization.json` records the assignment and exclusions; it is not an externally issued authorization ID.

Protected development-host evidence: `~/almadial-lab/evidence/m0-01/20261105T013428Z/`. Directories are 0700 and regular files 0600. Raw network addresses, guest paths, DNS, repository URLs and direct command outputs remain there, outside Git. No credentials or other VM records were copied. No global instance listing or commands against other instances were used.

The evidence contains the original accepted source archive, source checksums, authorization, image signature/key records, before/after VM and snapshot metadata, both unmodified inventories, command execution records, separate direct cross-checks, correction source/patch manifest, pre-recovery checksums and recovery verification. The ordinary guest account (UID 1000) performed collection; no sudo, package installation, network changes or security-policy changes were used.

## Measured host and image provenance

| Field | Actual observation |
|---|---|
| VM / access | `almadial-alma10`, scoped Multipass exec/transfer, Running after recovery |
| Hypervisor | Multipass/multipassd 1.17.0, `qemu` driver; guest reports `kvm` |
| OS | AlmaLinux 10.2 (Lavender Lion), ID `almalinux` |
| Kernel | `6.12.0-211.64.1.el10_2.x86_64` |
| Architecture / ISA | `x86_64`; 4 CPUs; measured flags include SSE4.1/4.2, POPCNT, AVX/AVX2, BMI1/2, F16C, FMA, MOVBE, XSAVE; guest loader reports x86-64-v2 and v3 “supported, searched” |
| RAM | Configured 8 GiB; guest MemTotal 8,313,655,296 bytes |
| Disk | 85,899,345,920 bytes (80 GiB); XFS root filesystem 84,546,662,400 bytes, approximately 82.1 GB available during collection; separate boot/EFI partitions |
| Network / DNS | Multipass NAT, private IPv4/default route, loopback and IPv6 observations recorded; configured resolver present; `getent ahostsv4 repo.almalinux.org` succeeded. Exact addresses remain protected. |
| Time | NTP synchronized `yes`, chronyd active/enabled, local RTC `no`, leap status Normal, stratum 2; direct status agrees |
| SELinux | Enforcing |
| Firewall | `firewalld` package absent, unit not-found/inactive; `firewall-cmd` and `nft` unavailable; nftables unit not-found. **Active kernel rules remain UNAVAILABLE**; absent units do not prove an empty ruleset. |
| Repositories | BaseOS, AppStream, CRB, Extras enabled; cache-only DNF listing succeeded; all four configure `gpgcheck=1` and the AlmaLinux 10 key. `repo_gpgcheck` is unspecified (collector effective default false); no controls changed. |
| Python | 3.12.15, satisfying collector's Python 3.12+ contract |
| Existing stack | Queried Asterisk, MariaDB, Apache/PHP and Perl packages absent; corresponding queried service units not-found; known daemon binary paths absent. This is not a full filesystem search for arbitrary source installations. |
| Listeners | TCP 22 on IPv4/IPv6 wildcard; UDP 323 on loopback IPv4/IPv6; no queried telephony/database/web listener conflict observed |

Image: `AlmaLinux-10-GenericCloud-10.2-20261012.0.x86_64.qcow2`.

Full image SHA-256: `cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`.

The existing QCOW2 was rehashed; it matches the local CHECKSUM entry and Multipass's full image hash. The detached CHECKSUM signature was verified with GPG in an isolated evidence keyring; `VALIDSIG` matches the operator-supplied fingerprint **`EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572`**. Key, CHECKSUM and signature copies are protected with the evidence. Official AlmaLinux acquisition origin is attributed to the operator's provisioning record: the original download URL/request log was not among the available files and was not fabricated. Repository configuration/cache inspection does not prove a new metadata signature check or every installed RPM's provenance.

Historical provisioning observation, attributed to the operator: cloud-init attempted `dnf -y install pollinate` and failed with exit 1. This assignment neither repaired nor repeated it. The actual later `cloud-init status --long` query returned exit 2, “degraded done,” no current errors and a deprecated `system_info` warning. These are separate observations; the warning does not erase the historical failure. Guest access, DNS, cached repository listing and NTP worked during this run.

## Source integrity, observed defects and execution

Accepted commit is still HEAD: **`18b97eea1438e0d1be72276fd3018659a33a636f`**. Collector, tests and fixture initially matched their accepted bytes. The existing transfer archive passed its checksums, Git archive commit check and exact five-file comparison. Archive SHA-256: `2954ec090a4b161f16f709229c68a89c81387ffedbcbc2a91f8c5aa933850a89`. Accepted collector SHA-256: `b894e1d43500162ee59464f0290853235e6f1a8b39fcfdf34f8c4a8c6f5bb5f0`.

The archive and genuine authorization record were transferred to a new protected guest directory. Guest checksums matched before extraction/execution. The original accepted collector ran first, without substitution: exit **1**, 18 COLLECTED, 2 ERROR, 3 UNAVAILABLE. Its evidence is retained unchanged under `inventory-accepted/`.

Two defects were reproduced against actual output:

1. The fixed RPM query returned **8** for eight absent operands with complete recognizable output. The old collector accepted only exit 1. The correction requires a positive exit code equal to the count of explicit known absent-package records; the existing parser still requires every requested package to be represented. Mismatched counts/error codes remain errors. The synthetic fixture's inaccurate single failure code was corrected to 7 for its seven absent packages.
2. `timedatectl show --property=NTPSynchronized,Timezone,LocalRTC` returned exit 0 and empty stdout. Repeating `--property` for each field returned the required values. The fixed catalog now uses repeated flags, consistent with [systemd's property-option documentation](https://www.freedesktop.org/software/systemd/man/latest/timedatectl.html).

Two regression test methods cover these failures, including mismatched RPM error codes and successful-but-empty time output. All seven requested systemd service IDs were returned on this clean host; the historical service-alias concern was **NOT_REPRODUCED**, so that parser was left unchanged. An installed database alias remains outside this host's observed coverage.

The corrected working-tree collector was transferred separately with file/patch checksums and rerun into a fresh directory. It is **not** the previously accepted commit's unchanged collector and is **not independently accepted** by this implementation run.

| Correction artifact | SHA-256 |
|---|---|
| Collector | `07a3bd79ca43a59a1ee55fc54adcce9d2964a83edf831d6cf827f7929973f282` |
| Tests | `5624cb18b0794b600bb59cac74f971875dd5d45a0b672bd9ba4bddeef157675d` |
| Synthetic fixture | `d3aede2234a3e10985643e41cc6dd929edabc7a91ff893ba4a626119596e9477` |
| Three-file Git diff patch | `2cb2bfd251875be32248e651eea729f60b6552124e4674a6f2f7eee7e91f8014` |

The protected `collector-accepted.json` and `collector-corrected.json` record exact argv, timestamps, exit codes and diagnostics. This equivalent corrected command uses a home-relative path to avoid publishing the private guest home:

```bash
cd "$HOME/almadial-m0-01-20261105T013428Z"
python3 correction/collect_host_inventory.py --collect \
  --host-ref almadial-alma10 \
  --authorization-ref operator-assignment-ALM-1-multipass-20261105T013428Z \
  --snapshot-ref almadial-alma10.clean-baseline \
  --output-dir ./inventory-corrected
```

Corrected collection: **2026-11-05T01:37:29Z**, mode `HOST_COLLECTION`, all `simulated=false`, exit **1**, **20 COLLECTED / 0 ERROR / 3 UNAVAILABLE**. Exit 1 is intentional partial evidence, not a health PASS. The native states are COLLECTED, ERROR, UNAVAILABLE and NOT_RUN; no unsupported FAILED/NOT_APPLICABLE state was inserted. Compatibility, host acceptance and independent QA remain NOT_RUN in both original JSON documents.

## All 23 probes and direct cross-checks

| Probe | Accepted run | Corrected run | Cross-check / limitation |
|---|---|---|---|
| os | COLLECTED | COLLECTED | Separate os-release read agrees |
| memory | COLLECTED | COLLECTED | meminfo / `free --bytes`; MemTotal agrees, availability changes naturally |
| dns | COLLECTED | COLLECTED | Separate resolv.conf read; direct repository hostname resolution succeeds |
| repositories | COLLECTED | COLLECTED | Local repo files versus cache-only enabled DNF listing |
| binaries | COLLECTED | COLLECTED | Known absent paths corroborated by package/unit queries; arbitrary source installs not exhaustively excluded |
| kernel | COLLECTED | COLLECTED | `uname -r`, `uname -a` agree |
| cpu | COLLECTED | COLLECTED | Separate lscpu JSON agrees; guest loader confirms supported v2/v3 |
| disks | COLLECTED | COLLECTED | Separate explicit-column lsblk; 80 GiB device confirmed |
| mounts | COLLECTED | COLLECTED | Separate findmnt plus byte-unit df/inodes; sizes agree, free space changes |
| interfaces | COLLECTED | COLLECTED | Separate IP JSON / brief address output; address identities remain protected |
| routes_v4 | COLLECTED | COLLECTED | Separate IPv4 route query / direct route listing |
| routes_v6 | COLLECTED | COLLECTED | Separate IPv6 route query; no public IPv6 reachability claim |
| hostname | COLLECTED | COLLECTED | Separate hostname query; identity redacted |
| packages | ERROR, exit 8 | COLLECTED, exit 8 | Complete RPM output, eight installed and eight absent requested packages |
| selinux | COLLECTED | COLLECTED | Separate getenforce: Enforcing |
| firewalld | UNAVAILABLE | UNAVAILABLE | command absent; package absent and unit not-found corroborate it |
| firewall_zones | UNAVAILABLE | UNAVAILABLE | command absent; no active-zone claim |
| firewall_rules | UNAVAILABLE | UNAVAILABLE | command absent; nft also absent; active kernel rules unresolved |
| time | ERROR, parse_error | COLLECTED | Repeated properties plus timedatectl status agree |
| chrony | COLLECTED | COLLECTED | Separate tracking query: normal leap / synchronized; transient offsets differ |
| services | COLLECTED | COLLECTED | Seven IDs reproduced; only queried chronyd active/enabled |
| listeners | COLLECTED | COLLECTED | Separate ss query and `ss -lntup` agree on ports/bindings |
| virtualization | COLLECTED | COLLECTED | Separate systemd-detect-virt: kvm, consistent with QEMU |

Protected `direct-crosschecks-corrected.json` contains **35 bounded command results and 13 file reads**, each with actual outputs and command result states. `crosscheck-verification.json` records comparison of stable allowlisted facts; HMAC identities are normalized for comparison, volatile available-memory/free-space/time-offset metrics are excluded from literal equality. Raw identity-bearing output was inspected locally. These are implementer cross-checks, **not independent QA**. No collection command was run against the development host.

## Evidence hashes and recovery

| Preserved artifact | SHA-256 |
|---|---|
| Accepted-run inventory.json | `918f704b9f16a00ca023275427f662ab8eccdbdfe748b64388a6c00e5bd98fe0` |
| Accepted-run summary.md | `77be36a1aac96dd584243647899f5c9756f417d1fa4b47233c8cb359bbfab24e` |
| Corrected-run inventory.json | `7e02e7d91ce229312f8a698f62b754be9f993f6882297d84440d310533b9bc42` |
| Corrected-run summary.md | `923aa5bb7de263c9ac09ccfa2715d584d6b0b11020b2e2f9cbd9d59c61e187f4` |

Both inventories and summaries were transferred with supported Multipass transfer and verified against guest-side hashes **before** stopping or restoring the VM. `pre-recovery-manifest.json` recorded 51 protected evidence files. Recovery then used only:

```bash
multipass stop almadial-alma10
multipass restore almadial-alma10.clean-baseline
# Answered yes to preserve the current state in a snapshot.
multipass start almadial-alma10
```

All exited 0. Default restoration created **`almadial-alma10.snapshot2`** before restoring **`almadial-alma10.clean-baseline`**; no `--destructive`, snapshot deletion or other-instance operation was used. Both snapshots were read back. The clean snapshot belongs to this VM, records 4 CPUs / 8 GiB / 80 GiB, and was created before this collection. At **2026-11-05T01:39:27Z**, guest access worked, transferred collection work was absent (expected rollback), and OS/architecture/kernel/SELinux matched the baseline. All pre-recovery host evidence matched its recorded hashes before the mutable run-state record was updated. Snapshot restoration is **PASS for implementer verification**, independent reproduction still NOT_RUN. The VM is left Running at the clean baseline; the pre-restore collection state remains in snapshot2 and protected host storage.

## Validation, governance and handoff

Executed successfully after the code correction:

```bash
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 -m py_compile scripts/collect_host_inventory.py scripts/tests/test_collect_host_inventory.py
```

Scaffold validation passed (11 required files, 4 skills); **24 tests passed** (20 inventory tests and 4 validator tests). `git diff --check`, `git status --short` and `git diff --stat` also executed successfully; exact command results are protected in `repository-validation.json`. No Go implementation exists in this change. No commit or push was made.

Kaneo identity/membership was discovered before work. Codex self-assignment was verified before the In Progress transition; the transition was read back. Original task description/acceptance, existing comments, M0 and Blocked labels were preserved. ALM-1's native blocks relation to ALM-2 remains `jkinkto2gkdunoukh415zhgy`. The current project read contained 35 existing tasks; none was recreated. Earlier no-VM prerequisites are resolved for this run; the retained Blocked label now represents missing mandatory firewall-rule evidence and the need to resolve that gap before full QA handoff.

Evidence comment **`kjysm088e0l0rv2jzpj060zm`** was created and retrieved through MCP, with exact content equality verified. All three comments (including both historical comments) remain. Final task read-back confirmed In Progress and Codex user `aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`; the description was unchanged. No label was removed or added. No task was self-accepted. MCP evidence records are protected with the host evidence.

Next bounded assignment: resolve the firewall observation gap on this exact VM using an authorized available read-only method, or obtain a separately scoped provisioning/acceptance decision. Do not install tooling under this inventory assignment, equate absent firewalld with no kernel rules, or waive mandatory evidence silently. A separate reviewer must review both corrections and reproduce critical live observations/recovery from the protected evidence and named snapshot. Retain Codex as implementer; enter QA Review only when implementation-side requirements are satisfied; no self-acceptance.

After ALM-1 is independently Accepted, **ALM-2 / M0-02 — Qualify candidate versions and dependency availability**, `kzj273ypahs82zegi1wwjfag`, is the next native dependent task. Recommend Codex CLI / GPT-6.1 Sol (or approved highest available model), High reasoning, Linux telephony compatibility engineer; use Git, pinned official sources, cache-aware DNF/RPM, Python/Bash and Kaneo. Deliver an exact candidate version/dependency/source-provenance matrix and narrow experiment plan, with unavailable packages and unexecuted compatibility cases explicit. Configured repositories alone do not validate Asterisk/VICIdial/PHP/Perl/WebRTC compatibility. ALM-2 remains Backlog/unassigned and was not started.
