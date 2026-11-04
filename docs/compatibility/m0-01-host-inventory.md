# M0-01 host inventory preparation

Task: **ALM-1 / M0-01**, Kaneo ID `p8urvq3yjtc76rktih9xlhe3`. Preparation date: 2026-11-05 (Asia/Manila). Worker: Codex, Principal Linux Telephony Engineer and Infrastructure Automation Architect. This is an authorized repository-only preparation slice, not full work-package execution or independent acceptance.

Preparation verdict: **ALMADIAL_M0_INVENTORY_TOOLING_READY** after the fixture/contract checks below. Host baseline: **ALMADIAL_M0_HOST_BASELINE_NOT_RUN**. No disposable host has been inspected, no stack selected and no compatibility test executed.

## Reproducible source and ownership

The reviewed documentation/scaffold baseline is local commit `1df743a4178fc32d9d78d53de8ea0870b1d2c30f` on `main`. Its parent was `bab92445bf1fe338d6cca46b422c7fd651e106e2`. It explicitly includes the previously locally excluded AGENTS, skills, templates, prompt and candidate manifest. Only reviewed project-owned files were staged; private worker configuration, caches and raw evidence were excluded. Existing project-specific ignore rules were reviewed and preserved in that commit. A Markdown whitespace attribute preserves intentional hard breaks without changing the source-plan digest. A stale reference to a nonexistent Codex configuration example was corrected. No push occurred.

Source plan v1.0-draft remains byte-identical in both plan paths, SHA-256 `2a8effec4e1e40d3577382158303d4e3124e56712e9877d3cac387a68bc0cfd4`. The inventory collector/tests and this preparation handoff follow that documentation baseline. The separate [inventory QA report](../reviews/m0-01-inventory-qa.md) records the reviewed file digests and reproduction results; use the Git commit containing that report as the tooling revision. Each output embeds the collector SHA-256. The final handoff and Kaneo QA comment record the resulting complete commit SHA after local finalization; there is no self-referential commit digest in these files.

ALM-1 remains Backlog / Blocked / unassigned. Codex identity was discovered by `whoami` and confirmed in workspace membership; no ownership or status change was justified for the still-blocked host work package. Before future host execution the worker must discover its own authorized identity, self-assign before In Progress and verify both assignment and status by MCP read-back. Independent QA retains implementer ownership.

## Collector interface

The research utility is `scripts/collect_host_inventory.py`, Python 3.12+ standard library only. Go is absent in this workspace, and introducing the M1 Go product framework is outside this M0 slice. No dependency installation, Go module, permanent agent or background service was introduced. Future Go functionality should be based on accepted M0 contracts rather than assuming this research script is the installer architecture.

Offline replay, safe on a developer checkout (the output directory must not exist):

```bash
python3 scripts/collect_host_inventory.py \
  --fixture scripts/tests/fixtures/inventory/almalinux10.json \
  --output-dir /tmp/almadial-inventory-fixture
```

This writes `inventory.json` and `summary.md`, with mode **FIXTURE** and every replayed observation marked `simulated: true`. Missing fixture probes remain NOT_RUN; replay never invokes a host collector. Fixtures contain only synthetic data and documentation-only credential sentinels used to test redaction.

The following is a **future protocol, NOT_RUN**, and grants no host authority. After the operator supplies and verifies the prerequisites below, invoke locally on the named disposable host as its ordinary inspection user:

```bash
python3 scripts/collect_host_inventory.py --collect \
  --host-ref LAB-VM-01 --authorization-ref AUTH-ALM-1-READONLY \
  --snapshot-ref SNAP-CLEAN-01 \
  --output-dir ./m0-01-inventory
```

These are synthetic references to a separate protected operator record, not a VM address or credentials. CLI flags cannot authenticate a host, prove authority or verify snapshot recovery. The operator must perform those checks independently. There is no SSH, remote execution, sudo or privilege escalation in the collector. Do not rerun as root just to eliminate missing observations; return the specific gap for separately authorized inspection.

No default collection mode exists. Exit **0** means all defined probes produced parseable facts in the selected mode; it never means host acceptance or compatibility. Exit **1** means partial collection, including unavailable, malformed or NOT_RUN probes; sanitized evidence is still written. Exit **2** means invalid arguments/input or output failure; a partial new evidence directory may remain after an I/O failure. The collector refuses an existing directory or symlink and never overwrites evidence. Output directory/file modes are 0700/0600, subject to a more restrictive umask. Parent directories must already exist and be operator-controlled.

## Inventory coverage and evidence methods

Every entry below is **NOT_RUN on a disposable host**. The fixture validates parsing, not the values of an actual environment. The exact fixed argv catalog is in `COMMANDS` in the collector; no shell is used. Files are read as data, never sourced or executed.

| Inventory | Method / probe | Evidence and limits |
|---|---|---|
| Distribution and exact minor version | Read `/etc/os-release` | Require ID almalinux and version 10.x before other live probes. This does not establish signed image provenance or its architecture variant. |
| Kernel | `uname -r` | Exact release, not a compatibility result. |
| CPU architecture, count, instruction flags | `lscpu --json` | Numeric count, architecture and sorted flags. ISA qualification stays NOT_RUN; package/image requirements need separate review. |
| Memory | Read `/proc/meminfo` | Total and available bytes; no process inventory. |
| Block devices | `lsblk --json --bytes` with explicit columns | Recursive sizes/types/filesystems, opaque device and mount identities. No serials or UUIDs. |
| Mounted filesystems | `findmnt --json --bytes` with explicit columns | Source/target identities, type, size and availability. Mount options are excluded because they may contain credentials. |
| Interfaces, addresses, IPv4/IPv6 routing | `ip -json address show`; `ip -json -4/-6 route show table all` | Opaque interface, address and prefix identities, family/scope, prefix length, route metric and type. MACs are omitted. NAT reachability is not tested or inferred. |
| Hostname and DNS configuration | `hostname`; read `/etc/resolv.conf` | Opaque hostname/domain/nameserver identities. No DNS request. Does not inspect every NetworkManager or systemd-resolved override. |
| Enabled package repository configuration | Read `/etc/dnf/dnf.conf` and `/etc/yum.repos.d/*.repo` | Configured enabled and signature-check flags, location-present flag. URLs, names and credentials are omitted. No DNF invocation/cache refresh/network contact. CLI/plugin overrides, remote availability, cached metadata, signing-key authenticity and package origins remain NOT_RUN. |
| Relevant installed RPMs | Fixed `rpm -q --qf` query | Exact queried versions/architecture or explicit absent packages. An exit 1 is only parsed as package absence when every requested package has an understood result. Includes OS, kernel, Python, PHP, Perl, MariaDB, Asterisk, HTTP, time and security packages. Not a full RPM provenance/license inventory. |
| SELinux | `getenforce` | Enforcing/Permissive/Disabled, no policy changes. A collected undesirable state is still a fact, not a health PASS. |
| firewalld state, active zones and runtime rules | `firewall-cmd --state`, `--get-active-zones`, `--list-all-zones` | Exit codes/errors, standard service names, explicit ports, selected policy flags, opaque sources/interfaces and fingerprints of other rule lines. Rich/custom rules require local protected inspection; fingerprints cannot prove their semantics. No reload or permanent-rule write. |
| Time synchronization | `timedatectl show` selected properties; `chronyc -n tracking` | Reported synchronization, opaque timezone, chrony offsets/stratum/leap state. Chrony queries its local daemon; no external NTP request or clock adjustment. No independent time-accuracy assertion. |
| Existing telephony/database/web services | `systemctl show` selected properties for fixed units | Load, active, substate and enablement for Asterisk, MariaDB/MySQL, HTTP/PHP, chrony and firewall. No start/restart/enable/disable. |
| Unmanaged installation indicators | File existence at five standard application binary paths | Presence only; application binaries are never executed. Nonstandard paths/source-build versions and VICIdial SVN/runtime state need separately reviewed inspection. |
| Relevant listeners | Numeric listening TCP/UDP sockets via `ss` | Port/protocol and opaque bind address classification. No process command lines, active connections or remote connections. |
| Virtualization | `systemd-detect-virt` | Recognized technology; exit 1 with `none` is recorded as a valid negative observation. No host UUID or hypervisor access. |

Official [AlmaLinux 10.0 release notes](https://wiki.almalinux.org/release-notes/10.0), consulted 2026-11-05, document Python 3.12, separately identify x86_64 and x86_64_v2, and describe signed-image checksum verification. This motivates collecting exact release and CPU flags and requiring separate image provenance. It does not qualify later minor versions, external RPM ISA requirements or the telephony stack. Image choice remains an operator/M0 evidence decision.

## JSON and redaction contract

The version-1 document contains `schema_version`, `mode`, `collector_sha256`, UTC start/finish timestamps, opaque operator references, sanitization policy and a stable ordered `probes` array. `host_baseline_acceptance`, `compatibility` and `independent_qa` remain **NOT_RUN** even after host collection; a collector cannot grant those decisions.

Each probe records its ID, exact fixed argv (empty for file reads), method, UTC observation timestamp, simulated flag, exit code (null when no process ran), collection status, structured facts, error category, stderr presence and byte count. Status is **COLLECTED**, **UNAVAILABLE**, **ERROR** or **NOT_RUN**. COLLECTED means a parser produced facts, not PASS for the host requirement. Failures include missing commands/files, permissions, nonzero exits, timeouts, size limits and malformed results. Missing required memory, time, chrony, service, device, interface or route fields produce ERROR/parse_error and partial-collection exit 1. Valid absence remains distinct: unformatted devices have an unknown filesystem, pseudo-filesystems may have null space metrics, an absent unit has not_applicable enablement, and a blackhole route can have no interface. Chrony retains offset direction. Standard error content is discarded after classification; counts and categories preserve diagnostic evidence without exporting arbitrary secrets.

The fixed service probe requires the requested canonical unit identities. Actual systemd alias behavior (for example mysqld resolving to mariadb) has not been qualified on AlmaLinux; an unrecognized or incomplete result fails closed as ERROR and requires protected local inspection. Empty repository or listener lists are explicit collected negative observations, not proof of usable repositories or service health. Missing tools, unknown states and inaccessible privileged observations must not be repaired through unapproved installs or privilege escalation.

Only selected parsed values are exported. Hostnames, domains, IPs, private paths, interface/repository identities and unknown firewall rules use keyed HMAC identities consistent within a run. The random key is never saved; different runs cannot correlate identities automatically. Exact package/kernel versions, known binary paths, CPU flags, ports and security states are intentionally retained for engineering review. MACs, URLs, credentials, raw stdout/stderr, process arguments, machine IDs and hardware serials are omitted. Privacy is a bounded output contract, not a guarantee that an artifact is public: review sanitized evidence before exporting it. Preserve the host-to-opaque-reference mapping separately under operator control.

Commands run with fixed C locale and PATH, closed stdin, no inherited credential-bearing environment, a 10-second deadline per command and a 1 MiB combined-output cap. File reads are capped at 1 MiB; repository count/aggregate size are also bounded. The collector does not change host configuration or install anything. It writes only its requested evidence directory. Ordinary kernel/service access accounting and service-side query logging can still occur.

## Missing host prerequisites and stop conditions

Before actual collection, the operator must provide:

1. Named disposable VM identity and console/access route, with an explicit statement excluding production/customer resources. Verify identity through the console before collection.
2. Exact image filename, architecture variant, source, signed checksum/key verification record, digest and installation method. An installed `/etc/os-release` file cannot establish these.
3. Clean snapshot identifier/time and verified restore procedure. Snapshot declaration is not restore proof.
4. Exact task-scoped authority for local read-only commands/file reads and evidence writes. No package, service, firewall, network or credential mutation is included.
5. Reviewed tooling revision/digests and Python availability. Missing utilities remain UNAVAILABLE; do not install them under inventory authority.
6. Protected evidence location, identity-redaction mapping and a separate reviewer able to repeat critical observations.

Stop if target identity or production scope is ambiguous, image origin is unknown, recovery is unproven, root/mutation is requested, credentials appear, the distribution gate fails or authority is incomplete. The collector's references are guardrails, not substitutes for this evidence. Mandatory missing observations block ALM-1 acceptance.

## Executed local verification and handoff

Run the repository's existing commands:

```bash
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
git diff --check
```

Fixture tests cover complete and missing observations, incomplete RPM results, malformed JSON, service states, redaction of synthetic credentials/host information, no fixture-to-host fallback, OS rejection, explicit CLI authority requirements, private output modes, refusal to overwrite, missing commands, process exit/error capture, timeouts and output caps. Synthetic child processes test the runner; none collects host inventory. The final execution results and source-to-Kaneo checks are recorded in `docs/planning/kaneo-sync-report.md`.

Full ALM-1 acceptance additionally requires measured host inventory, authenticated image/repository provenance, CPU ISA qualification, recoverable clean snapshot, attributable commands/results/digests and separate reproduction/review. No host evidence is PASS here. Independent tooling QA is recorded separately and does not accept ALM-1. The next bounded engineering assignment is actual collection on the explicitly authorized disposable AlmaLinux 10 VM, after all prerequisites and assignment governance above are satisfied. A separate reviewer must repeat critical observations from the recorded snapshot, compare structured facts and source digests, and classify mandatory gaps; per-run UTC timestamps and HMAC identities will differ.
