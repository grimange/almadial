# ALM-11 / M1-03 Independent Host Preflight and Immutable Plan QA Report

- **Task ID:** `xfy4pagbrmppq3yijr7nz7d7` (ALM-11 / M1-03)
- **Task Title:** Implement preflight and a zero-mutation immutable plan
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-07T00:50:00Z
- **Repository Starting HEAD:** `782f5e3bbfacf15c3687f9197da647b0b6b23faa` (Accepted ALM-10)
- **Prior Operator Commit HEAD:** `ac63e2d62f3154f1d222aaacd083818332c010c7`
- **Final Verdict:** `ALMADIAL_M1_03_ACCEPTED`

---

## 1. Reviewer and Governance Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-11 (`xfy4pagbrmppq3yijr7nz7d7`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Comment:** Verified existing implementation evidence comment `mjmofdzxirqc50gm3mac7x4m`.
- **Dependency Graph:**
  - Upstream ALM-10 (`bcyodkgz47sak45stno5um0b`): Verified status `done` (Accepted).
  - Relation ALM-10 $\rightarrow$ ALM-11 (`i5uqxxrtgfqmhr0pvvq46ax1`): Verified intact (`blocks`).
  - Relation ALM-12 $\rightarrow$ ALM-11 (`b2zbuex4u4qu0p9ydztpfg5t`): Verified intact (`blocks`).
  - Relation ALM-13 $\rightarrow$ ALM-11 (`curs65man9xc64rg8bu0e1j8`): Verified intact (`blocks`).
  - Relation ALM-11 $\rightarrow$ ALM-15 (`dgjpbo02awzebi4dhxhoc9ni`): Verified intact (`blocks`).
  - Downstream ALM-12 (`yrjd60b1at9ug4hetrkvz01v`): Verified status `to-do` (Backlog, unassigned).
  - Downstream ALM-13 (`xrai81mxz5p8r600svlnqgts`): Verified status `to-do` (Backlog, unassigned).
  - Downstream ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
In accordance with authorized product goals and repository directives:
- **Core Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Preflight & Planning Boundary:** Implements read-only host preflight inspection and deterministic immutable installation planning.
- **Zero Host Mutation:** Pure observation and planning only. No installer stages are executed, no packages installed, no files written, no services modified, no firewall/SELinux rules altered, and no operator credentials resolved.
- **Manifest State:** `manifests/stack.lock.yaml` remains candidate; manifest digest is treated as an input integrity token without premature lock promotion.

---

## 2. Preflight Architecture and Host Observation (`internal/preflight`)

### 2.1 Pure Separation of Fact and Assessment
- **`Snapshot`:** Captures raw, uninterpreted system telemetry across 9 essential domains:
  1. `OS`: Operating system distribution, release version (`10.2`), ID, and kernel release.
  2. `Architecture`: Hardware architecture derived from `uname -m`.
  3. `CPU`: Logical CPU count and CPU instruction set flags intersected across all detected cores.
  4. `Memory`: Total physical memory (`MemTotal`) and currently available memory (`MemAvailable`).
  5. `Disks`: Total and available bytes for installation paths (`/`, `/usr`, `/var`), with shared-mount disambiguation.
  6. `Network`: Interface enumeration, status, assigned IPv4 and IPv6 addresses, loopback identification, and default gateway.
  7. `Sockets`: Active listening TCP and UDP sockets with address bindings and inode tracking.
  8. `Security`: SELinux enforcement mode and firewall backend status (`firewalld` and `nftables`).
  9. `Time`: Time synchronization state and NTP client daemon operational status.
- **`Assessment` and `Finding`:** Pure functional evaluation of a `Snapshot` against typed `config.Config` and caller-provided `Requirements`. Findings strictly adhere to 4 discrete categories:
  - `PASS`: Requirement definitively satisfied.
  - `WARNING`: Suboptimal or non-critical condition detected (e.g. absent firewalld where nftables is present).
  - `FAIL`: Hard incompatibility or critical conflict that prevents successful installation.
  - `NOT_VERIFIED`: Telemetry could not be gathered due to permission or platform constraints.

### 2.2 Resource Policy Abstraction
- Preflight evaluates hardware resources strictly against an explicit `Requirements` struct supplied by policy, rejecting heuristic hardcoding.
- Validates CPU instruction flag requirements, total RAM floor, available RAM headroom, and dedicated per-filesystem disk space.

---

## 3. Zero-Mutation Enforcement and Probe Safety

### 3.1 Closed Command Allowlist (`collect_linux.go`)
Execution of external diagnostic utilities is restricted to a hardcoded whitelist of 6 read-only binaries:
- `/usr/sbin/ip` (address and route queries)
- `/usr/sbin/ss` (socket listening queries)
- `/usr/bin/rpm` (package verification queries)
- `/usr/bin/systemctl` (service unit status queries)
- `/usr/sbin/nft` (firewall ruleset inspection)
- `/usr/bin/timedatectl` (system clock and NTP status)

Any attempt to execute an unlisted binary is aborted prior to execution with `unsupported read-only probe`.

### 3.2 Subprocess Safety and Isolation
- **Direct Exec:** Subprocesses are invoked with explicit argument vectors (`exec.CommandContext`); shell interpreters are prohibited, eliminating command injection vectors.
- **Output Bound:** Captured subprocess output is truncated at 4 MiB (`1 << 22` bytes) to prevent memory exhaustion from runaway diagnostics.
- **Execution Deadline:** Every probe is bound to an immutable context timeout of 5 seconds.
- **No Side Effects:** Probe execution causes zero modifications to host filesystem, service states, network interfaces, or kernel parameters.

---

## 4. Immutable Plan Architecture and Determinism (`internal/plan`)

### 4.1 Encapsulation and Tamper-Proof Immutability
- The `Plan` data structure encapsulates its canonical state within an unexported private field (`data string`).
- Public methods (`Stages()`, `Findings()`, `JSON()`, `Human()`) deserialize on-demand or return defensive copies. Callers cannot mutate the internal plan structure.

### 4.2 Standard Stage Progression (S00–S11)
The plan generates a deterministic sequence of 12 declarative installation stages:
- `S00_PREFLIGHT`: Environment, architecture, and resource preflight verification.
- `S01_BASE_PACKAGES`: Core system utilities, compilers, development headers, and runtime libraries.
- `S02_MARIADB`: Native MariaDB server installation, strict mode configuration, and tuning.
- `S03_APACHE_PHP`: Apache HTTP server, PHP 8.3 FPM, and required extensions.
- `S04_ASTERISK`: Asterisk 22 installation and core runtime configuration.
- `S05_DAHDI`: Virtual DAHDI dummy timing driver for Asterisk conferencing.
- `S06_VICIDIAL_SOURCE`: VICIdial source tree checkout and schema initialization.
- `S07_VICIDIAL_SERVICES`: VICIdial cron jobs, screen sessions, and background Perl daemons.
- `S08_FIREWALL_SELINUX`: SELinux policy enforcement and firewalld/nftables port rules.
- `S09_VICIPHONE`: Self-hosted VICIphone 3.0 assets and WebRTC phone provisioning.
- `S10_TLS_CERTIFICATES`: TLS certificate provisioning and web/SIP transport binding.
- `S11_VERIFICATION`: Post-install health checks and end-to-end smoke verification.

### 4.3 Conditional WebRTC Adaptation
- When `webrtc.enabled: false`, Stage `S09_VICIPHONE` is omitted from the plan. Stages `S04` and `S10` conditionally exclude WSS and browser certificate actions.
- When `webrtc.enabled: true`, Stage `S09` is inserted between `S08` and `S10`, requiring verified TLS paths and non-conflicting WSS ports.

### 4.4 Deterministic Digest Verification
Repeated invocations of `plan.Build()` with identical configuration and preflight inputs yield identical cryptographic digests:
- **Basic Single-Server Plan Digest:** `33ac7565c7bb03364c4b407b3d2fd82f103af23686e2b473d429fbbcbba1a0ea`
- **WebRTC Single-Server Plan Digest:** `560f2091428630f0480ea8de8c5ea535bf55f3c253c0600e465a7b16d22d2de2`

---

## 5. Plan Freshness and Revalidation Policy

### 5.1 Drift Detection Contract
- `Fingerprint()` generates a canonical SHA-256 fingerprint encompassing:
  - Effective configuration approval digest.
  - Manifest source digest.
  - Resource policy parameters.
  - Network and listening socket bindings.
  - Critical preflight assessment statuses.
- `Revalidate(currentSnapshot)` compares current system telemetry against the established fingerprint.
- Any change in socket occupancy, manifest digest, configuration parameters, or critical failure state triggers an immediate plan invalidation.

### 5.2 Jitter Tolerance
- Revalidation ignores harmless fluctuations in available memory and disk space, provided values remain above configured policy thresholds.

---

## 6. Automated Verification and Reproducible Builds

### 6.1 Go Test Suite and Static Analysis
- `make fmt-check`: **PASS** (Zero code formatting violations).
- `go vet ./...`: **PASS** (Zero compiler or static analysis warnings).
- `go test -v ./...`: **PASS** (29 top-level tests, 196 subtests, 225 pass events).
  - `internal/preflight`: 11 top-level tests, 78 subtests.
  - `internal/plan`: 5 top-level tests, 38 subtests.
  - `internal/config`: 10 top-level tests, 102 subtests.
  - `internal/cli`: 2 top-level tests, 13 subtests.
  - `internal/buildinfo`: 1 top-level test.
- `go test -race ./...`: **PASS** (Zero data race conditions detected across all packages).
- `go mod verify`: **PASS** (`all modules verified`).

### 6.2 CLI Preservation and Reproducible Builds
- ALM-9 CLI contract preserved: 14/14 smoke tests pass with exact stdout/stderr and exit code semantics.
- Developer build artifact (`dist/almadial`): SHA-256 `5735a7d5e94f6dc2daa8074a5966cea2613d7aa3eca8100792992831b9b7d549` matches byte-for-byte.
- Independent reproducible build test: Two separate clean checkouts built with fixed timestamp and vcs metadata produced identical binaries (`a8427a4d6150f93804aeded8cc0b7cecf61382a9d086bc05696dd51ea8e508fd`).

### 6.3 Baseline Repository Validations
- `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.840s).
- `python3 scripts/verify_candidate_stack.py`: **PASS**.
- `python3 scripts/verify_config_generation.py`: **PASS**.
- `git diff --check`: **PASS** (Zero whitespace issues).

---

## 7. Evidence Ledgers and Cryptographic Verification

### 7.1 Historical Implementation Ledgers
- **Core Ledger:** `/home/ra/almadial-lab/evidence/m1-03/20261107T001618Z/SHA256SUMS`
  - Digest: `427428eca46175cbae1372c9b19b43994e4161d8e4980f075a569f65d54683e7`
  - Verification: 46/46 entries verified PASS.
- **Handoff Ledger:** `/home/ra/almadial-lab/evidence/m1-03/20261107T001618Z/SHA256SUMS.handoff`
  - Digest: `e9ca857e804c9a95b85d1b54eb2ef6a583c863dec7eddb7d28c4c464b2fd52e8`
  - Verification: 1/1 entry verified PASS.

### 7.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-03/qa-20261107T004500Z/` (Mode `0700`)
- **Sealed Ledger SHA256SUMS:** `85dbddb49e2da9c297397149dbe2499daca1a717853d9368ae9c211112d4db0b`
- **Sealed Artifacts:**
  - `qa-evidence-ledger.json`: `178ed6b6031c95c5f6c9fb8a04f8a3f5b457cfd9f6fbcf8f9eebb126354c3524`
  - `qa-historical-ledgers.json`: `24c903313079c85f6967943c0bbe8dcb9111b2a5b1c47d81a727c024add757c1`
  - `qa-plan-determinism-and-freshness.json`: `00d432b119595aad19580188b58906bd72e5ad44b271a3bf6c44fd870cc4ca28`
  - `qa-preflight-and-plan-tests.json`: `71d92e29c52fea9f5af411c3c03dc12b9208b9275a114a4b672ff8a3f53573a5`
  - `qa-reproducible-builds.json`: `a264058a456e140018b7b18750b0eca8f355087c3140b7decd85fa1adc07ce73`
  - `qa-repository-validation.json`: `8df43742906499e8707cda6099b1cf45eb357ef5cbfb1e5cb902edd8f05392d3`

---

## 8. Final Acceptance Verdict

# `ALMADIAL_M1_03_ACCEPTED`

Task ALM-11 satisfies all canonical acceptance criteria. The preflight and plan packages provide read-only host telemetry gathering, strict assessment isolation, tamper-proof immutable installation plans, and deterministic freshness revalidation.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-12 / M1-04** (`yrjd60b1at9ug4hetrkvz01v`): *Verify stack manifests and source artifacts*.
