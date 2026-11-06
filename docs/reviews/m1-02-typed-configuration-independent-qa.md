# ALM-10 / M1-02 Independent Configuration Validation and Security QA Report

- **Task ID:** `bcyodkgz47sak45stno5um0b` (ALM-10 / M1-02)
- **Task Title:** Define typed non-secret configuration and validation
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-06T23:50:00Z
- **Repository Starting HEAD:** `cce3d155f31bfcdb5cfc2b1977f6ac5f21320150` (Accepted ALM-9)
- **Final Verdict:** `ALMADIAL_M1_02_ACCEPTED`

---

## 1. Reviewer and Governance Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-10 (`bcyodkgz47sak45stno5um0b`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Comment:** Verified existing implementation evidence comment `oe88psr9l2uo29q974tqs3q9`.
- **Dependency Graph:**
  - Upstream ALM-9 (`y04mu7040nia07ii8ny8b3yr`): Verified status `done` (Accepted).
  - Relation ALM-9 $\rightarrow$ ALM-10 (`ooozogngehsjgd9jkq1lu60z`): Verified intact (`blocks`).
  - Relation ALM-10 $\rightarrow$ ALM-11 (`i5uqxxrtgfqmhr0pvvq46ax1`): Verified intact (`blocks`).
  - Relation ALM-10 $\rightarrow$ ALM-15 (`g9n59p7l12t5rlb773kcyhqr`): Verified intact (`blocks`).
  - Downstream ALM-11 (`xfy4pagbrmppq3yijr7nz7d7`): Verified status `to-do` (Backlog, unassigned).
  - Downstream ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
In accordance with the authorized product goals and operator directives:
- **Core Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Configuration Boundary:** Defines schema v1 typed configuration, strict validation, deterministic defaults, and secret isolation.
- **IP-Only Installation:** Explicitly supports a single-server deployment without mandatory FQDN or WebRTC.
- **Zero Host Mutation:** Pure configuration decoding and validation only. No file I/O against secret sources, no environment resolution, no network DNS probes, no database queries, and no systemd/service operations.
- **Manifest State:** `manifests/stack.lock.yaml` remains candidate; no unvetted promotion.

---

## 2. Configuration Model and Strict JSON Decoding

### 2.1 Typed Data Model (`internal/config/config.go`)
- Strongly typed Go structures: `Config`, `Network`, `SIP`, `RTP`, `WebRTC`, `TLS`, `Credentials`, `SecretRef`.
- Addresses use standard library `net/netip.Addr`.
- Pointer integer fields (`*int`) for ports distinguish omitted fields (which receive documented defaults) from explicit zero values (which are rejected as invalid).

### 2.2 Strict Decoding Engine (`internal/config/decode.go`)
A two-pass parser guarantees complete rejection of ambiguous or malformed input:
1. **First Pass (`guard` token traversal):**
   - Size limit: Document cannot exceed 1 MiB (`1 << 20` bytes).
   - Character encoding: Strict UTF-8 validation (`utf8.Valid`).
   - Unknown field rejection: Any field not in the explicit `shape()` table is rejected.
   - Duplicate key rejection: Checked recursively across all object levels using a collision map.
   - Case-sensitivity: Rejects case-mismatched keys; prevents Go's standard JSON case-folding behavior.
   - Explicit null rejection: Prohibits `null` tokens where values are expected.
   - Type integrity: Validates integer representation (rejects floating-point notation and overflow).
   - Trailing data: Rejects trailing JSON documents or non-whitespace content.
2. **Second Pass (`dec.DisallowUnknownFields`):**
   - Populates typed Go structures.
   - Guarantees zero partial configuration leakage on parse failure (`Load` returns a zero `Config`).

---

## 3. Network, Port, and Conditional Feature Validation

### 3.1 IP Address and Hostname Validation (`internal/config/validate.go`)
- **Bind Address:** Requires valid unicast IPv4/IPv6 or wildcard (`0.0.0.0` / `::`). Mapped IPv4 (`Is4In6`) and scoped IPv6 (`Zone != ""`) are rejected.
- **Advertise Address:** Must be a global unicast address (RFC 1918 / RFC 4193 private addresses supported; loopback and link-local rejected). If bind is wildcard, explicit advertise address is mandatory and must match the bind address family.
- **Hostname:** Optional. When supplied, validated against RFC 1035 / RFC 1123 DNS syntax (max 253 chars, 1–63 chars per label, alphanumeric and hyphens, no leading/trailing hyphen, cannot be an IP literal). No network DNS lookups performed.

### 3.2 SIP and RTP Media Port Rules
- **SIP UDP Port:** 1–65535, default 5060.
- **RTP Range:** Inclusive range 1–65535, defaults 10000–20000. `RTP.Start` must be an even integer for RTP/RTCP pairing. `RTP.End` must strictly exceed `RTP.Start`.
- **Port Conflict Isolation:** SIP UDP port cannot overlap the UDP RTP range.

### 3.3 Conditional WebRTC and TLS References
- **WebRTC Disabled (`webrtc.enabled: false`):** Default configuration. No mandatory TLS or WSS fields. Setting WSS or TLS parameters when disabled produces an explicit validation error.
- **WebRTC Enabled (`webrtc.enabled: true`):** Requires WSS port (1–65535, default 8089; must differ from SIP port). Requires separate, non-identical absolute file references for `certificate_file` and `private_key_file`.

---

## 4. Secret Isolation and Non-Disclosing Diagnostics

### 4.1 Secret Reference Architecture
- Supports four required credential purposes: `database`, `ami`, `sip`, `admin`.
- Allowed reference types: `env` (environment variable name) and `file` (absolute Linux file path).
- Raw passwords, inline secrets, and PEM blocks are prohibited and rejected.

### 4.2 Security Boundaries During Pure Parsing
1. **Zero Secret Resolution:** Configuration decoding and validation never read `os.Getenv` or read file contents. Non-existent file paths pass pure configuration validation.
2. **No Data Leakage:** `FieldError` emits only the field path and a static reason string. It deliberately does not echo user-supplied values or unknown field names.
3. **Redacted Read Errors:** Reader failures (such as synthetic reader errors containing private tokens) are masked with generic diagnostic strings.

---

## 5. Unattended Approval-Digest Binding

### 5.1 Deterministic Canonical Digest (`Digest()`)
- Approval is bound to the normalized, validated effective configuration:
  - Defaults applied.
  - Hostname normalized (lowercase, trailing dot trimmed).
  - Explicit advertise address populated.
- Canonical JSON is hashed using SHA-256.
- Changes to any configuration group (bind address, advertise address, hostname, SIP port, RTP range, WebRTC state, TLS paths, or credential sources) alter the digest and invalidate approval.

### 5.2 Unattended Approval Evaluation (`ValidateUnattended()`)
- Explicit `Approval.Approved == true` is mandatory.
- `Approval.ConfigSHA256` must match the canonical digest.
- **Security Boundary:** The digest is an input integrity binding, not operator authentication or authorization to mutate the host.

---

## 6. Automated Verification and Coverage Results

### 6.1 Go Test Suite and Statement Coverage
- `make fmt-check`: **PASS** (Zero formatting issues).
- `go vet ./...`: **PASS** (Zero diagnostic warnings).
- `go test -v ./...`: **PASS** (13 top-level tests, 115 subtests, all pass).
  - `internal/config`: 10 top-level tests, 102 subtests.
  - `internal/cli`: 2 top-level tests, 13 subtests.
  - `internal/buildinfo`: 1 top-level test.
- `go test -race ./...`: **PASS** (Zero data race conditions detected).
- `go test -coverprofile`: **96.9% statement coverage** in `internal/config`.
- `go mod verify`: **PASS** (`all modules verified`).

### 6.2 CLI Preservation and Reproducible Builds
- ALM-9 CLI behavior preserved: 13/13 executable smoke tests pass with exact stream, exit code, and secret-masking contracts.
- Developer build artifact (`dist/almadial`): SHA-256 `76cbf7337fbb2aee91dfe64bfb2a89aea80f6378091eb3c66e6d7bc781209763` matches byte-for-byte.
- Independent reproducible build test: Two isolated trees with separate empty caches built with fixed metadata produced identical binaries (`ec573c838d12fe1528241c9805101b5d7f14d62769ae29a443b97b2790a02092`).

### 6.3 Baseline Repository Validations
- `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.928s).
- `python3 scripts/verify_candidate_stack.py`: **PASS**.
- `python3 scripts/verify_config_generation.py`: **PASS**.
- `git diff --check`: **PASS** (Zero whitespace issues).

---

## 7. Evidence Ledgers and Cryptographic Verification

### 7.1 Historical Implementation Ledgers
- **Core Ledger:** `/home/ra/almadial-lab/evidence/m1-02/20261106T232306Z/core/SHA256SUMS`
  - Digest: `fa71e2fa8e2a50fb053e62217e2af1db962126a634225b4fd1ebf31d552d8b93`
  - Verification: 59/59 entries verified PASS.
- **Handoff Ledger:** `/home/ra/almadial-lab/evidence/m1-02/20261106T232306Z/handoff/SHA256SUMS`
  - Digest: `30164e05660d8442139b0231a7ee2878f7f1a0eaba231f3180501c36f1897a9a`
  - Verification: 2/2 entries verified PASS.

### 7.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-02/qa-20261106T234500Z/` (Mode `0700`)
- **Sealed Ledger SHA256SUMS:** `99d344a32ab9d5c40b5bf048b9871950c79216ee2a41f05a35e51f5f4beb18fc`
- **Sealed Artifacts:**
  - `qa-historical-ledgers.json`: `6146edf15aee17df9ed415f4fcde656cf597ed9e88286618671ebc7087bcfc6d`
  - `qa-config-coverage.json`: `d42c2f59ccf2016d46d118456ff023e666c1d9a3098f89c94f5964b00c578904`
  - `qa-reproducible-builds.json`: `f4e975eb783f19e1ac934f2c951c91a90905e9ba3727fa4b162d9d18a9618671`
  - `qa-security-and-approval.json`: `ae7e2de4bae751d2a025d74a7497fd1a45877a6ff7e017791615d1028b57446d`
  - `qa-repository-validation.json`: `cc3a6db582a1f2510fd6180400672d89fcf7576032d984808606d08b47400e4e`
  - `qa-evidence-ledger.json`: `960bbf2336214304dbd10cbaf4df9fb803ae8d2c4bbf949d107a7e6b52750e32`

---

## 8. Final Acceptance Verdict

# `ALMADIAL_M1_02_ACCEPTED`

Task ALM-10 satisfies all canonical acceptance criteria. The configuration package provides strongly typed, strictly validated, non-secret installation inputs with reproducible approval binding.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-11 / M1-03** (`xfy4pagbrmppq3yijr7nz7d7`): *Implement preflight and a zero-mutation immutable plan*.
