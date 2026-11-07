# ALM-12 / M1-04 Independent Manifest and Artifact Supply-Chain QA Report

- **Task ID:** `yrjd60b1at9ug4hetrkvz01v` (ALM-12 / M1-04)
- **Task Title:** Verify stack manifests and source artifacts
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-07T02:25:00Z
- **Repository Starting HEAD:** `baaf4f647b9f5ce5b8cf6b2694a9939ee68d1192` (Accepted ALM-11)
- **Final Verdict:** `ALMADIAL_M1_04_ACCEPTED`

---

## 1. Independent Reviewer and Task Governance

### 1.1 Kaneo Task State and Assignment
- **Task Identity:** ALM-12 (`yrjd60b1at9ug4hetrkvz01v`).
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the task owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Evidence Comment:** Verified comment `hb3vzh31b510txu8uqrj77dw`.
- **Dependency Graph:**
  - Upstream ALM-9 (`y04mu7040nia07ii8ny8b3yr`): Verified status `done` (Accepted).
  - Relation ALM-9 $\rightarrow$ ALM-12 (`b1xg3ie3qi0seiwzqdq24n18`): Verified intact (`blocks`).
  - Relation ALM-12 $\rightarrow$ ALM-11 (`b2zbuex4u4qu0p9ydztpfg5t`): Verified intact (`blocks`).
  - Downstream ALM-11 (`xfy4pagbrmppq3yijr7nz7d7`): Verified status `done` (Accepted).
  - Downstream ALM-13 (`xrai81mxz5p8r600svlnqgts`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Core Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Artifact Verification Boundary:** Implements strict manifest parsing, real cryptographic detached-signature verification, canonical archive tree verification, authenticated HTTPS acquisition, and private staging cache management.
- **Zero Host Installation:** The package reads and verifies source artifacts only. It installs zero RPM packages, executes zero scripts/binaries, modifies zero system configurations, and does not promote the candidate stack lock.

---

## 2. Manifest Schema and Strict Parsing (`internal/artifacts`)

### 2.1 Schema-1 Decoding Engine (`manifest.go`)
- Decodes the canonical `manifests/stack.lock.yaml` (64 KiB buffer limit).
- Exact-byte SHA-256 digest: `80940fac407ca7ee0041378b2e1e8b4ab4d95aa95749b503d5df7d4f356009fa`. Available through `Manifest.Digest()` for binding into ALM-11 immutable installation plans.
- Uses pinned `go.yaml.in/yaml/v3 v3.0.4` (`go.sum` verified, notices preserved in `third_party/go-yaml/` and `docs/THIRD_PARTY_NOTICES.md`).
- Strict structure and type enforcement:
  - Rejects unknown root and nested fields without echoing unrecognized names.
  - Rejects duplicate mapping keys and duplicate component identifiers.
  - Rejects YAML anchors, aliases, and merge keys.
  - Rejects unsupported schema versions (only schema 1 permitted).
  - Validates all leaf values against compiled `approvedLeaves` in `policy.go`.

### 2.2 Independent Trust Anchors (`policy.go`)
- Trust anchors are compiled Go policy, never adopted from untrusted caller input or manifest self-assertions:
  - Operating system: AlmaLinux 10.2 x86_64 (`cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`).
  - Asterisk 22.12.0: Source URL, SHA-256, and full signing fingerprint (`F2FC93DB7587BD1FB49E045A5D984BE337191CE7`).
  - VICIdial r4037 / schema 1748: Source URL, archive SHA-256, and canonical tree SHA-256 (`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`).
  - VICIphone 3.0: Pinned commit (`24e98fccfa7b7079fff970e02fca5f002ac1f2cc`), archive SHA-256, and embedded SIP.js 0.20.1 SHA-256 (`9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b`).
  - PJPROJECT 2.17: Pinned archive SHA-256 (`04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7`).
  - Net::Telnet 3.05: Pinned archive SHA-256 (`677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743`).
- **Candidate Status Preserved:** `Manifest.InstallationAuthorized()` returns `false` unconditionally.

---

## 3. Real M0 Source Artifact and Signature Verification

### 3.1 Real M0 Preserved Artifacts Audit
Independently verified against preserved M0 artifacts (`/home/ra/almadial-lab/evidence/m0-08/20261106T192640Z/inputs-final/`):
- **Asterisk 22.12.0 (`asterisk-22.12.0.tar.gz`):**
  - SHA-256: `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` (**PASS**)
  - Size: 26,782,027 bytes; Files: 2,204 regular files.
  - Tree SHA-256: `a84d36aa0d165ac86c7911d1e543ac30f334f105ffffa6f4d4e119cd995f2583` (**PASS**)
  - Contained symlinks strictly verified: all 18 relative symlinks target regular files under `asterisk-22.12.0/`.
- **Asterisk Detached Cryptographic Signature (`signature.go`):**
  - Armored public key SHA-256: `2eb75c272ebd1f96742ef7b0b53ebe943c840c55addc22db2122a494df578145` (**PASS**)
  - Dearmored public keyring SHA-256: `645c8fb2b67133a59d9f7d1113e953c5dc79950fd0b28a6529a0efef0146727c` (**PASS**)
  - Detached signature SHA-256: `17763b5903a3962b0278ffd8d581e913eedd6c0fd21149f4855db67eb40d890d` (**PASS**)
  - Execution via fixed `/usr/bin/gpgv`: Validated full `VALIDSIG` with fingerprint `F2FC93DB7587BD1FB49E045A5D984BE337191CE7` and SHA-2 digest (**PASS**).
  - Corrupted signature test: Intentionally altered signature was definitively **REJECTED**.
- **VICIdial r4037 / schema 1748 (`vicidial-r4037.zip`):**
  - SHA-256: `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` (**PASS**)
  - Size: 30,258,976 bytes; File Count: exactly 2,511 regular files (**PASS**).
  - Canonical Tree SHA-256: `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` (**PASS**).
  - Matches Python's `sorted(pathlib.Path)` component-wise ordering.
- **VICIphone 3.0 (`viciphone.tar.gz`):**
  - SHA-256: `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` (**PASS**)
  - Size: 639,858 bytes; Files: 48 regular files; Pinned Commit: `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`.
  - Embedded `src/js/sip.js` (SIP.js 0.20.1) SHA-256: `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` (**PASS**).
- **PJPROJECT 2.17 (`pjproject-2.17.tar.bz2`):**
  - SHA-256: `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` (**PASS**)
  - Size: 8,975,382 bytes; Files: 3,363 regular files.
- **Net::Telnet 3.05 (`Net-Telnet-3.05.tar.gz`):**
  - SHA-256: `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` (**PASS**)
  - Size: 42,553 bytes; Files: 8 regular files.
- **Reviewed M0 Compatibility Helpers (`derivations.go`):**
  - All 6 helper scripts verified read-only against compiled hashes without execution (**PASS**).

---

## 4. Authenticated HTTPS Retrieval and Cache Integrity (`fetch.go`)

### 4.1 Controlled HTTPS Transport Verification
- **Mandatory TLS:** Minimum TLS 1.2 enforced. Certificate and hostname validation mandatory; explicit `RootCAs` pool supported for controlled testing.
- **Redirect Rejection:** `CheckRedirect` unconditionally rejects all HTTP/HTTPS redirects to prevent redirect-based provenance bypass.
- **Bounded Resource Limits:** Dial/TLS timeout 15s, header timeout 20s, overall deadline 2 minutes. Maximum download capped at 512 MiB (`maxArchiveBytes`).
- **Negative Transport Reproduction:**
  - Untrusted TLS certificates: **REJECTED**.
  - Hostname mismatches: **REJECTED**.
  - Unapproved redirects: **REJECTED**.
  - Oversized payloads: **REJECTED**.
  - Interrupted / truncated transfers: **REJECTED**.
  - HTTP 404 / 500 error responses: **REJECTED**.

### 4.2 Private Staging and Cache Confinement
- Staging directory strictly confined using `os.OpenRoot(dir)`. Directory must be owned by the current EUID with mode `0700`.
- Downloads write to a private temporary file (`.download-<nonce>`, mode `0600`).
- On any error, network interruption, or checksum mismatch, the temporary file is immediately removed.
- Completed downloads are published atomically using filesystem link (`root.Link`), preventing partial files from ever appearing as verified cache entries.
- **Cache Reverification:** Every cache hit reopens and streams SHA-256 validation. Corrupted cache entries return an error and are never silently overwritten or used.

---

## 5. Bounded Archive Inspection Safety (`archive.go`)

### 5.1 Pure Non-Extracting Inspection
- Inspects archives without extracting contents to live disk paths.
- Enforces hard resource limits: max 512 MiB compressed, max 2 GiB expanded, max 256 MiB per regular file, max 30,000 entries.
- ZIP central directory bounded to 16 MiB prior to allocation, preventing ZIP-bomb memory exhaustion.
- Compression trailers (gzip / bzip2) and ZIP CRC32 checksums are validated by reading full stream contents.

### 5.2 Traversal and Entry Rejection
- Prohibits absolute paths, parent directory traversal (`../`), backslashes, NUL bytes, colons, and non-canonical paths.
- Prohibits duplicate entry names within a single archive.
- Rejects FIFOs, devices, sockets, hard links, and sparse files.
- Rejects generic symbolic links. Contained relative symlinks are permitted only for Asterisk, where targets are strictly verified to reside within `asterisk-22.12.0/` and point to existing regular files.

---

## 6. Trust Separation and Authorization Boundaries

The implementation strictly decouples distinct supply-chain trust states:
1. **Artifact Content Identity:** Determined strictly by streaming SHA-256 match against pinned specification.
2. **Transport Authentication:** Validated only when fetched during the current invocation over approved HTTPS with valid TLS certificates. Local cache hits report `Transport: NOT_VERIFIED`, preventing fabricated transport claims.
3. **Publisher Provenance:** Verified via detached GPG signature (`asterisk`) or verified official HTTPS repository origin. Local checksums leave publisher evidence `NOT_VERIFIED`.
4. **Installation Authorization:** Unconditionally `false`. Candidate status is preserved; verification never grants permission to install or execute.

---

## 7. Automated Verification and Reproducible Builds

### 7.1 Go Test Suite and Static Analysis
- `make fmt-check`: **PASS** (Zero formatting issues).
- `go vet ./...`: **PASS** (Zero compiler/static analysis warnings).
- `go test -v ./...`: **PASS** (274 passing test and subtest events).
- `go test -race ./...`: **PASS** (Zero data race conditions detected across all 6 packages).
- `go mod verify`: **PASS** (`all modules verified`).
- `go mod tidy -diff`: **PASS** (Zero module drift).

### 7.2 Controlled Reproducible Build Verification
- Two independent builds from separate isolated source checkouts with fixed build metadata produced bit-for-bit identical Linux amd64 ELF binaries:
  - SHA-256: `545858053c4f19c0d5143ff087abb3f70c9762753a197d03d8f80a1e21974aeb`.
- Executable CLI smoke tests: 14/14 tests pass with exact stream separation, exit code semantics, and non-disclosing error messages.

### 7.3 Repository Scaffolding and Regressions
- `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.847s).
- `python3 scripts/verify_candidate_stack.py`: **PASS**.
- `python3 scripts/verify_config_generation.py`: **PASS**.
- `git diff --check`: **PASS** (Zero whitespace issues).

---

## 8. Evidence Ledgers and Cryptographic Verification

### 8.1 Historical Implementation Ledgers
- **Core Ledger:** `/home/ra/almadial-lab/evidence/m1-04/20261107T013742Z/SHA256SUMS`
  - SHA-256 Digest: `2c7c2848ca301f5ee7e6aa0e3cb8cb1dfe28a8bc2a607ea0d973384d9e2aa388` (117/117 files verified **PASS**).
- **Governance Ledger:** `/home/ra/almadial-lab/evidence/m1-04/20261107T013742Z/governance/SHA256SUMS`
  - SHA-256 Digest: `98a72fb743c8f0ffec0fd79211ed543f776964182cc0c54fe6601181f1fda123` (1/1 file verified **PASS**).

### 8.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-04/qa-20261107T022000Z/` (Mode `0700`)
- **Sealed Ledger SHA256SUMS:** `26d2164c45063d517ab3533eb231c9a65fd2196c3aaaf3d45611f22080b2124b`
- **Sealed Artifacts:**
  - `qa-evidence-ledger.json`: `cf479e4df535f61f84b1859f339636f3b25db0229947aa446c1ab8ee74208b2f`
  - `qa-go-and-python-suites.json`: `84ab3e309d518a37405a5fde7a13090d428f439cd076ef2416db8375fcc48d31`
  - `qa-historical-ledgers.json`: `1335f3e8d63ec62fecb5f6568d3cc1addc8eb3c0f1bf350fae50cc6152d9d21f`
  - `qa-real-artifacts-and-signatures.json`: `ff3e1b65d5d255e5d7609314b72aabe229047ec34a523f6436fbc1cf9f20decb`
  - `qa-reproducible-builds-and-smokes.json`: `2617ecfc0d682f66de21c63f58147994314d4e7f1ec6fce4195df1768e805e58`

---

## 9. Final Acceptance Verdict

# `ALMADIAL_M1_04_ACCEPTED`

Task ALM-12 satisfies all canonical acceptance criteria. The artifact package strictly validates the schema-1 stack manifest, authenticates HTTPS acquisitions, validates real Asterisk detached GPG signatures, validates the 2,511-file canonical VICIdial tree, enforces archive inspection boundaries without live extraction, rejects cache corruption, and preserves the boundary between diagnostic verification and installation authorization.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-13 / M1-05** (`xrai81mxz5p8r600svlnqgts`): *Implement bounded runner, action graph and error taxonomy*.
