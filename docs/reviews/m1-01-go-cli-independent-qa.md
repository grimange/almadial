# ALM-9 / M1-01 Independent Go CLI and Reproducible Build QA Report

- **Task ID:** `y04mu7040nia07ii8ny8b3yr` (ALM-9 / M1-01)
- **Task Title:** Create the Go module, CLI and reproducible build
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-06T23:00:00Z
- **Repository Starting HEAD:** `3f331301882c3a72a14886f3617f964265acfb7e` (Accepted M0 baseline)
- **Repository Current HEAD:** `67da129838f4fc7d16498be4b5b49111dd268ca6` (Preserves operator recording-finalization commit)
- **Final Verdict:** `ALMADIAL_M1_01_ACCEPTED`

---

## 1. Governance and Scope Boundary Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-9 (`y04mu7040nia07ii8ny8b3yr`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Comment:** Verified existing implementation evidence comment `dqh4dp8td9gjbjiua7goh78t`.
- **Dependency Graph:**
  - Upstream ALM-8 (`ftlf2grvt89s3h4nv1yz4x11`): Verified status `done` (Accepted).
  - Relation ALM-8 $\rightarrow$ ALM-9 (`yuko5stwg99z60pmy48byx7g`): Verified intact (`blocks`).
  - Relation ALM-9 $\rightarrow$ ALM-10 (`ooozogngehsjgd9jkq1lu60z`): Verified intact (`blocks`).
  - Relation ALM-9 $\rightarrow$ ALM-12 (`b1xg3ie3qi0seiwzqdq24n18`): Verified intact (`blocks`).
  - Relation ALM-9 $\rightarrow$ ALM-13 (`rrtlv1ifc6hep7abbdn2mzmv`): Verified intact (`blocks`).
  - Downstream ALM-10 (`bcyodkgz47sak45stno5um0b`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline
ALM-9 establishes the foundational Go executable, CLI routing, build metadata, and reproducible build pipeline. In accordance with the project contract:
- No installer workflows, package installations, or OS mutations are implemented.
- No network requests are made during CLI command execution.
- No database access or Asterisk configuration alterations are performed.
- `manifests/stack.lock.yaml` retains its authorized candidate state and is not modified or promoted.

---

## 2. Toolchain Provenance and Go Module Audit

### 2.1 Compiler and Toolchain Verification
- **Go Version:** `go1.27.3 linux/amd64`.
- **Pinned Version Record:** `.go-version` specifies `1.27.3`.
- **Compiler Archive:** `go1.27.3.linux-amd64.tar.gz`.
- **Official Archive SHA-256:** `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40` (Matches official Go distribution metadata from `https://go.dev/dl/?mode=json`).
- **Environment Isolation:** Toolchain configured with `GOTOOLCHAIN=local` and `GOENV=off`, preventing automatic background downloads.

### 2.2 Go Module & Dependency Audit
- **Module Identity:** `github.com/grimange/almadial` (matches Git repository origin).
- **Direct Dependencies:** Zero external module dependencies. Standard library packages only (`fmt`, `io`, `os`, `runtime`, `strings`, `testing`, `bytes`, `errors`).
- **Module Verification:** `go list -m all` lists only `github.com/grimange/almadial`. `go mod verify` reports `all modules verified`.

---

## 3. Go Architecture and Code Review

- **`cmd/almadial/main.go`:** Minimal process entry point forwarding arguments and standard streams to `cli.Run()`, calling `os.Exit()` strictly on the return value.
- **`internal/cli/cli.go`:** Fully decoupled, testable CLI router. Accepts `io.Writer` streams for stdout and stderr.
  - Implements three explicit exit codes: `Success = 0`, `Failure = 1`, `Usage = 2`.
  - Rejects unknown commands, unknown options, extra arguments, and unimplemented workflows with exit code 2.
  - **Security Protection:** Does not echo arbitrary user input back to stderr, preventing command-line argument secrets or ANSI escape injection from leaking.
- **`internal/buildinfo/buildinfo.go`:** Emits link-time injected version and commit metadata alongside compiled `runtime.GOOS`, `runtime.GOARCH`, and `runtime.Version()`. Contains no uncontrolled wall-clock timestamps or host filesystem paths.
- **`scripts/build_go.py`:** Bounded, reproducible build driver in Python. Strictly validates compiler version (`go1.27.3`), forces `CGO_ENABLED=0`, `GOAMD64=v1`, `-trimpath`, `-buildvcs=false`, and `-mod=readonly`, and validates metadata strings against safe ASCII regular expressions before linker injection.

---

## 4. CLI Smoke Testing and Exit-Code Contract

All 13 CLI smoke-test cases were executed against the compiled `dist/almadial` binary:

| Invocation | Arguments | Expected Code | Actual Code | Stdout Observation | Stderr Observation | Verdict |
|---|---|---|---|---|---|---|
| `./dist/almadial` | `[]` | 0 | 0 | `Usage: almadial [help \| version]` | *(empty)* | **PASS** |
| `./dist/almadial help` | `["help"]` | 0 | 0 | `Usage: almadial [help \| version]` | *(empty)* | **PASS** |
| `./dist/almadial --help` | `["--help"]` | 0 | 0 | `Usage: almadial [help \| version]` | *(empty)* | **PASS** |
| `./dist/almadial -h` | `["-h"]` | 0 | 0 | `Usage: almadial [help \| version]` | *(empty)* | **PASS** |
| `./dist/almadial version` | `["version"]` | 0 | 0 | `almadial dev\ncommit: ...\ntarget: linux/amd64` | *(empty)* | **PASS** |
| `./dist/almadial --version` | `["--version"]` | 0 | 0 | Matches `version` output | *(empty)* | **PASS** |
| `./dist/almadial secret-val` | `["secret-value"]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` (No secret leaked) | **PASS** |
| `./dist/almadial --bad` | `["--bad"]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` | **PASS** |
| `./dist/almadial install` | `["install"]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` (Explicit rejection) | **PASS** |
| `./dist/almadial check` | `["check"]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` (Explicit rejection) | **PASS** |
| `./dist/almadial verify` | `["verify"]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` (Explicit rejection) | **PASS** |
| `./dist/almadial version extra`| `["version", "extra"]` | 2 | 2 | *(empty)* | `almadial: expected one command; use --help` | **PASS** |
| `./dist/almadial ""` | `[""]` | 2 | 2 | *(empty)* | `almadial: unknown command or option; use --help` | **PASS** |

### Metadata Injection and Rejection
- Linker metadata injection verified: `ALMADIAL_VERSION=v1.2.3 ALMADIAL_COMMIT=feedbeef0123` correctly appears in binary `version` output.
- Malformed metadata rejected: inputs containing spaces, shell semicolons, leading hyphens, or backticks are rejected with process exit code 2 before invocation of the Go toolchain, and no binary is produced.

---

## 5. Independent Reproducible Builds Verification

### 5.1 Development Artifact Verification
- **Executable Path:** `dist/almadial`
- **Binary Architecture:** ELF 64-bit LSB executable, x86-64, statically linked, no CGO dependencies.
- **SHA-256 Digest:** `69f6c4635d0862b761faf7cff44276fe5368e751e9c6489806a3a8f905b91ef0`
- **Result:** Matches the developer's reported artifact hash byte-for-byte.

### 5.2 Independent Multi-Directory Multi-Cache Reproduction
Two isolated source worktrees (`/tmp/agy-repro/src_a` and `/tmp/agy-repro/src_b`) with separate empty caches (`/tmp/agy-repro/cache_a` and `/tmp/agy-repro/cache_b`) were compiled using identical fixed inputs (`ALMADIAL_VERSION=v1.0.0-rc1`, `ALMADIAL_COMMIT=67da129838f4fc7d16498be4b5b49111dd268ca6`):
- **Build A Output:** `/tmp/agy-repro/bin_a/almadial` (Size: 2,366,671 bytes, SHA-256: `e3760aa9b188391cb0e0457171285c2495aeb94db6212b9cb718874468ccbeed`)
- **Build B Output:** `/tmp/agy-repro/bin_b/almadial` (Size: 2,366,671 bytes, SHA-256: `e3760aa9b188391cb0e0457171285c2495aeb94db6212b9cb718874468ccbeed`)
- **Byte Comparison:** `cmp` confirmed 100% byte-for-byte identity.
- **Result:** Deterministic build pipeline is verified reproducible across independent directories and caches.

---

## 6. Automated Test Suites and Validations

1. **Go Code Quality:**
   - `make fmt-check`: **PASS** (Zero formatting issues).
   - `go vet ./...`: **PASS** (Zero diagnostic warnings).
   - `go test -v ./...`: **PASS** (3 top-level tests, 13 subtests, 16 test pass events).
   - `go test -race ./...`: **PASS** (Go race detector passed on Linux amd64).
   - `go mod verify`: **PASS** (`all modules verified`).
2. **Repository Regression Validations:**
   - `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
   - `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.853s).
   - `python3 scripts/verify_candidate_stack.py`: **PASS** (Candidate stack manifest valid).
   - `python3 scripts/verify_config_generation.py`: **PASS** (Deterministic configuration generation verified).
   - `python3 -m py_compile scripts/build_go.py`: **PASS** (Build script compiles cleanly).
   - `git diff --check`: **PASS** (Zero whitespace issues).

---

## 7. CI Workflow Audit

- Inspected `.github/workflows/harness-checks.yml`.
- Permissions scoped strictly to `contents: read`.
- Retains existing Python starter-kit validation job.
- Adds minimal Go foundation validation job (`make fmt-check module-check test vet race build`, byte comparison, and smoke tests).
- Hosted GitHub Actions execution status: `HOSTED_CI_NOT_RUN` (Repository changes are local-only).

---

## 8. Historical and QA Evidence Ledger Integrity

### 8.1 Historical Implementation Ledgers
- **Implementation Ledger:** `/home/ra/almadial-lab/evidence/m1-01/20261106T224135Z/implementation/SHA256SUMS`
  - Digest: `665118c97fe245060bce4f378c0598300640c606115c19cf6f938bcf5137883d`
  - Status: 62/62 files verified, 0 failures.
- **Handoff Ledger:** `/home/ra/almadial-lab/evidence/m1-01/20261106T224135Z/handoff/SHA256SUMS`
  - Digest: `8ce78c2d07b1d42c04c6bb79759624490f2eb9e9a0ef4a8c93ee79e29ee4bf06`
  - Status: 10/10 files verified, 0 failures.

### 8.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-01/qa-20261106T225500Z/` (Mode `0700`)
- **QA Ledger SHA256SUMS Digest:** `761a2c3e364fa330a88f812b3b25559c27f2f6715546a423685afc2b059ed05a`
- **Sealed Artifacts:**
  - `qa-historical-ledgers.json`: `eb84d3950e5a0c84c391f51e77850f8239f06871eb8852734d294691ff599d19`
  - `qa-compiler-provenance.json`: `3154c378ac3993ba0041e78fd10580a720be6897e382e36a4ef7b973c100d81f`
  - `qa-cli-smoke-tests.json`: `e4a66a5f714e2bfd9ea8afb02335c9b0bdb3433644070b7d4cde456bda70ad67`
  - `qa-reproducible-builds.json`: `d4adc3fc84d7d1d0fc9dce9d2c1402b2f45206240dbe582b2874506dc651d1ee`
  - `qa-tests-and-validations.json`: `c6531977dfa2b433e9b5da02ba3a1b2c5f43bc6677a711edcb9c816e7a9a69f2`
  - `qa-evidence-ledger.json`: `e28408beaa32050e82c58a69d12463e26715f5cbb12eb88b835ec450953a8ae3`

---

## 9. Final Acceptance Verdict

# `ALMADIAL_M1_01_ACCEPTED`

The Go CLI foundation, testable routing, buildinfo metadata, toolchain isolation, reproducible build helper, and automated test validations satisfy all canonical ALM-9 acceptance criteria.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-10 / M1-02** (`bcyodkgz47sak45stno5um0b`): *Define typed non-secret configuration and validation*.
