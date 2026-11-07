# M1-03 — Read-only host preflight and immutable installation plan

ALM-11 adds `internal/preflight` and `internal/plan` to module
`github.com/grimange/almadial`, using Go 1.27.3 and the standard library. Starting
HEAD is `782f5e3bbfacf15c3687f9197da647b0b6b23faa` (Accepted ALM-10).
This is developer verification, pending separate AGY QA. It installs nothing,
resolves no secrets, and does not promote the candidate stack lock.

## Boundaries and inputs

The operator authorizes this pure implementation after ALM-10 acceptance. Kaneo's
proposed ALM-12 manifest and ALM-13 action dependencies remain intact. Their
contracts are represented by a **supplied manifest SHA-256**, not a competing
manifest parser, downloader, artifact verifier or execution engine. A syntactically
valid digest does not prove source provenance. Those future tasks must supply a
verified manifest and enforce approval before effects.

The existing CLI remains unchanged: help/version are available; `check`, `plan`
and installation commands remain usage errors. The canonical task requires APIs
with JSON/human rendering, not a new CLI. Future integration must use the APIs
below and preserve existing stdout/stderr and exit semantics.

There are no owned host resources, journals, checkpoints, lock files or recovery
transactions. Retry repeats observation or pure planning. Missing inspection
permissions produce blockers; preflight never elevates privilege or repairs a
host. No M0 VM was accessed or changed for ALM-11. Live AlmaLinux before/after
qualification is **NOT_RUN**; fake-host evidence must not be described as VM proof.

## Resource and CPU policy

The accepted M0 evidence qualifies **AlmaLinux 10.2 x86_64**. Other 10.x minor
versions require compatibility review; different kernel patches alone do not
block. The actual host architecture comes from `uname`, not the executable's
compiled target. CPU flags are intersected across every reported CPU.

`Requirements` supplies an explicit named policy: CPU count, required CPU flags,
minimum total memory, minimum currently available memory, and minimum available
disk bytes on each installation filesystem. All are required; omission yields
blocking NOT_VERIFIED. The profile and budget values participate in plan identity.
They are trusted policy inputs, not operator authorization or automatic discovery.

M0's 4 vCPU / 8 GiB / 80 GiB lab profile is explicitly **not a minimum** in the
implementation plan §4. The recorded guest supported v2/v3 instructions, but that
measurement does not establish a universal package ISA minimum. This package
therefore does not invent a minimum or require every instruction observed in the
lab. A later verified manifest/preflight integration must supply the ISA floor
for its exact selected OS image/packages and a reviewed build/runtime space
budget. Missing policy blocks planning eligibility. The tiny test budgets are
synthetic fixtures only and must never be reused as production sizing defaults.

Disk facts use read-only `statfs` for `/`, `/usr`, `/var`, recording total and
available bytes. Shared filesystem capacity is not added together. The supplied
per-filesystem budget must cover peak simultaneous installation usage. A future
stage that introduces a separate build/data filesystem must extend this explicit
inspection contract before execution. Memory facts distinguish MemTotal and
MemAvailable. Both are checked against the supplied policy, not an arbitrary
percentage of installed RAM.

## Observation and assessment APIs

```go
snapshot := preflight.Collect(ctx, preflight.Linux{}) // Linux only, read-only
findings, err := preflight.Assess(effectiveConfig, snapshot, reviewedRequirements)
// Handle err; inspect every blocking finding before proceeding.
p, err := plan.New(effectiveConfig, snapshot, reviewedRequirements, verifiedManifestSHA256)
// Handle err; a blocked plan remains renderable for diagnosis.
jsonBytes, err := p.JSON()
humanText, err := p.Human()
```

`Snapshot` records facts and per-probe `Known` flags. `Assess` normalizes and
validates config using the accepted ALM-10 API, then interprets facts separately.
`Finding` contains a stable ID, status, blocking flag and actionable message:

- PASS: the stated requirement was established.
- WARNING: review is needed without necessarily blocking installation.
- FAIL: a required condition is false.
- NOT_VERIFIED: a required observation or policy is incomplete; blocks eligibility.

No observations means no eligibility. Unsupported/existing installations stop.
Installed Asterisk, database server, httpd and VICIdial packages/units/processes or
known application/data paths require ownership review. Client connector libraries
alone are not a server installation. `/var/lib/almadial` also blocks: its existence
alone cannot authenticate ownership or authorize overwrites. Resume/managed-host
reconciliation belongs to later tasks, not a guessed exemption here.

Inspection includes:

| Probe | Read-only source / decision |
| --- | --- |
| Identity, OS, kernel | Machine ID, os-release, proc kernel release; exact qualified distribution/minor |
| Architecture / CPU | uname, proc cpuinfo; count and common capabilities against explicit policy |
| Memory / disk | proc meminfo, statfs; invalid/missing facts never pass |
| Network / DNS | `ip -j address show`; bind IP must be assigned, or wildcard must have a usable matching-family IP; resolver presence is informational |
| Listeners | `ss -H -lntu -n`; TCP/UDP are separate; wildcards conservatively overlap IPv4/IPv6; scoped/unparseable sockets block |
| Existing software | RPM query, service-unit observations, proc process names, known application/data path existence |
| SELinux | sysfs enforcing state; Enforcing required, unknown does not pass |
| Firewall | nftables stateless JSON ruleset and firewalld/nftables service observations; unavailable/denied inspection blocks |
| Clock | timedatectl NTPSynchronized; unavailable, malformed or unsynchronized state blocks |

DNS is not required for IP-only installations. The collector does no DNS lookup,
external clock query, certificate read, private-key read or secret-reference
resolution. Resolver presence does not prove DNS functionality; a supplied hostname
without observed resolver settings produces a warning. Certificate identity/trust,
network reachability, repositories/signatures and exact package capability checks
remain future stage preconditions, not fake successes from host inspection.

SIP, optional WSS and inclusive UDP RTP allocation come from effective config.
Existing HTTP/HTTPS and management sockets (80/443/3306/5038/4577 TCP) are also
review blockers for this fresh single-server installation. Address-specific SIP/WSS
listeners only conflict when binds overlap. Unknown IPv6 wildcard v6-only behavior
is treated conservatively. A clear instantaneous socket inventory does **not**
reserve RTP ports or establish future availability. Apply must recheck.

Firewall observation establishes that rules were read, not that they permit a
call. Its finding is a warning even when the ruleset is empty. No firewalld
installation is required. External-public, NAT and remote-agent profiles are not
qualified by these probes.

## No-mutation contract

`Source` exposes read, statfs, lstat, uname, process-name inspection and a closed
`Operation` enumeration. Production command argv are fixed:

```text
/usr/sbin/ip -j address show
/usr/sbin/ss -H -lntu -n
/usr/bin/rpm -qa --qf '%{NAME}\n'
/usr/bin/systemctl list-units --all --type=service --no-legend --no-pager --plain
/usr/sbin/nft --json --stateless list ruleset
/usr/bin/timedatectl show --property=NTPSynchronized --value
```

No shell, downloads, package metadata refresh, start/stop/reload, policy changes,
config writes or evidence writes occur. Commands use explicit executable paths,
a fixed locale/environment, a five-second context deadline, and bounded 4 MiB
stdout. Stderr is discarded. Failures retain sanitized probe identity/exit code;
raw command diagnostics are not rendered. File reads are bounded; proc process
inspection reads `comm` names, never secret-bearing argv or environment contents.
A fake `Source` operation trace asserts the exact file/command allowlist, and
unsupported mutation operations are rejected by the production dispatcher before
process creation. Files written by the developer test/evidence harness are separate
from production preflight behavior.

This is source/fixture evidence that preflight requests no mutation. It is not a
claim of exhaustive live filesystem tracing: kernel audit logs, access-time
accounting and read-tool implementation internals were not measured on a VM.
Ordinary read-only administrative tools are used without attempting to suppress
host auditing. No target-side evidence collector is embedded in this package.

## Immutable plan and stage contracts

Plan schema **1** binds ALM-10's effective configuration digest, the supplied
manifest digest, and a relevant host fingerprint. JSON uses typed field order;
findings, set-like observation collections and requirement flags have stable
ordering. No timestamps, random IDs, process IDs, working-directory names,
resolved secrets or mutable global configuration exist. The plan stores canonical
JSON privately. `JSON`, `Stages` and `Findings` return independent data; mutating
inputs or returned nested slices cannot change its digest.

Each ordered stage includes prerequisites, required inputs, intended changes,
risk and expected postconditions. These describe future effects; none runs:

| ID | Stage | Principal prerequisites / recovery boundary |
| --- | --- | --- |
| S00 | Fresh supported-host revalidation | Config/manifest/host identities and all mandatory checks; no effects |
| S01 | Approved dependencies | S00; verified package/source identities; no foreign repositories |
| S02 | Filesystem/service identities | S01; explicit ownership, modes and SELinux labels |
| S03 | Socket-only MariaDB/schema | S02; irreversible initialization; stop on partial schema |
| S04 | Native Asterisk | S01/S02; pinned build, owned base files, local AMI |
| S05 | VICIdial/derivations/generation | S03/S04; preserve upstream, verify schema and generated PJSIP/dialplan/ConfBridge |
| S06 | Runtime scheduling | S05; bounded upstream heartbeat/FastAGI, no duplicate jobs |
| S07 | Apache/PHP | S05; webroot ownership and authentication |
| S08 | Bounded network/security policy | S06/S07; actual bind/advertised addresses, SIP/RTP, optional TLS/WSS |
| S09 | Optional VICIphone | S04–S08; pinned assets, secure media, credential POST and lifecycle checks |
| S10 | Integrated verification | S06–S08 plus S09 when enabled; explicit local test fixtures and separately authorized reboot |
| S11 | Handover | S10; redacted results, limitations and cleanup |

S09 is absent when WebRTC is disabled. Basic S08 does not require certificate
paths or a hostname. WebRTC S08 states the actual WSS port and the optional
hostname or advertised-IP certificate identity. This follows ALM-10's explicit
operator-approved IP-only/optional-WebRTC contract rather than reviving the older
proposed FQDN-only table. Risk labels describe effects, not permission to perform
them. The future executor must implement per-stage backup, retry, ownership and
postcondition checks; these descriptions are not executable action objects.

## Freshness and approval

`p.Revalidate(config, freshSnapshot, requirements, verifiedManifestDigest)` is pure.
A zero/blocked plan is rejected. A new mandatory failure/incomplete observation,
changed configuration, host identity, qualified OS, architecture, required CPU
capability, resource threshold result, relevant listener/service/package/process
conflict, ownership conflict, security/clock state or stateless firewall identity
requires regeneration and renewed approval. Budget/profile changes also invalidate.
Usable local IP identity is bound. Equivalent set ordering is normalized.

Kernel patch/uptime/PID changes, added irrelevant CPU flags/listeners and available
memory/disk changes that remain above the supplied budget do not invalidate.
Stateless nft output excludes changing rule counters; changing material rules
invalidates. No timestamp expiry is guessed.

The future apply boundary must:

1. Verify manifest provenance and the original plan's identity.
2. Establish actual operator authorization for that plan and its effects.
3. Collect a **fresh** complete snapshot immediately before applying.
4. Call `Revalidate`; stop and request regeneration/renewed approval on mismatch.
5. Recheck susceptible resources/listeners at each effectful stage to address races.

Passing a caller-fabricated or old snapshot is not fresh verification. A matching
hash is content identity, never operator authorization. No apply engine exists in
ALM-11, so execution and enforcement inside a real executor are **NOT_RUN**.

## Examples and verification

Synthetic passing IP-only fixture, with an explicit test-only budget:

```text
os: PASS — verified
network: PASS — verified
memory-budget: PASS — verified
selinux: PASS — verified
firewall: WARNING — ruleset observed; future bounded network policy requires review, no remote reachability claim
Installation eligibility: ELIGIBLE
Host changes requested by preflight: NONE
S00 ... S08, S10, S11 (no S09)
No stages executed. Separate operator approval and fresh revalidation required.
```

Blocked example (same configuration, UDP SIP already occupied):

```text
listener-udp-05060-192.0.2.10: FAIL — required listener or RTP allocation overlaps existing socket; identify owner or change configuration
Installation eligibility: BLOCKED
Host changes requested by preflight: NONE
```

Run with the approved compiler and offline module/cache settings:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod verify
go test -v ./internal/preflight ./internal/plan
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Tests cover supported IPv4/IPv6, optional WebRTC and hostname, unsupported hosts,
resource/ISA shortages, socket relationships, existing installations, failed and
malformed probes, sanitized diagnostics, exact read-only argv/operation traces,
ordering, immutability, hashes and material/irrelevant freshness changes. Accepted
CLI/config/build tests run unchanged. Reproducible builds use the existing
`scripts/build_go.py` in separate source/output directories with identical explicit
version/commit metadata; compare SHA-256 values and smoke-test the executable.
Hosted CI and live VM mutation tracing remain NOT_RUN. Independent AGY review
must reproduce these checks before acceptance; ALM-12 is not started by this work.

## Implementation verification record

Developer verification on 2026-11-07 used Go 1.27.3 linux/amd64, with module
network access disabled. All validation commands above exited 0: 29 top-level Go
tests and 196 subtests (225 passing test events), race detector, vet, module
verification, formatting, and 126 Python regression tests. Module tidy reported
no changes. Eleven executable CLI smoke cases preserved accepted exit/stream
behavior, including rejection of unimplemented `check` and `plan` commands.

Two builds in separate source directories used the existing build script with
identical explicit metadata (`version=dev`,
`commit=782f5e3bbfacf15c3687f9197da647b0b6b23faa-alm11`) and produced Linux amd64
ELF executables with the same SHA-256:
`5735a7d5e94f6dc2daa8074a5966cea2613d7aa3eca8100792992831b9b7d549`.
This identifies controlled development builds, not a release or clean Git commit.
The CLI does not yet link the new APIs; their behavioral evidence comes from
the focused preflight/planner tests, not help/version execution.

Protected evidence is recorded under
`~/almadial-lab/evidence/m1-03/20261107T001618Z/`, including exact commands,
source copies, synthetic plan JSON/human examples, probe traces, build artifacts,
and a separate SHA-256 ledger. No live host eligibility or runtime installation
claim is made. During implementation an unrelated operator commit advanced HEAD
to `ac63e2d62f3154f1d222aaacd083818332c010c7`; the four initially modified
operator files retained their entry hashes, and accepted CLI/config/build code
and the stack manifest were unchanged.
