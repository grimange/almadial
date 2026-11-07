# ALM-13 / M1-05 Independent Bounded Runner and Action Graph QA Report

- **Task ID:** `xrai81mxz5p8r600svlnqgts` (ALM-13 / M1-05)
- **Task Title:** Implement bounded runner, action graph and error taxonomy
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-07T03:55:00Z
- **Repository Starting HEAD:** `3b229ee77185789d00f531c74c062d575ff03488` (Accepted ALM-12)
- **Final Verdict:** `ALMADIAL_M1_05_ACCEPTED`

---

## 1. Governance and Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-13 (`xrai81mxz5p8r600svlnqgts`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the implementation owner.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Evidence Comment:** Verified comment `mo2ueg4yqdltak8zhxk64z8g`.
- **Dependency Graph:**
  - Upstream ALM-9 (`y04mu7040nia07ii8ny8b3yr`): Verified status `done` (Accepted).
  - Relation ALM-9 $\rightarrow$ ALM-13 (`rrtlv1ifc6hep7abbdn2mzmv`): Verified intact (`blocks`).
  - Relation ALM-13 $\rightarrow$ ALM-11 (`curs65man9xc64rg8bu0e1j8`): Verified intact (`blocks`; ALM-11 already Accepted).
  - Relation ALM-13 $\rightarrow$ ALM-14 (`wwoamzsvoexj9ljfl0a0t7py`): Verified intact (`blocks`).
  - Downstream ALM-14 (`q2kgwrrzpn4zkzgvj08y6g1h`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Core Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Execution Boundary:** Supplies bounded local execution primitives and deterministic action dependency graphs for trusted installer code.
- **Zero Host Installation:** The package executes zero real installation stages, installs zero RPM packages, modifies zero host configuration, creates zero durable execution journals, and resolves zero operator secrets.

---

## 2. Action Graph and Deterministic Execution (`graph.go`)

### 2.1 Action Contracts and Graph Validation
- Actions require explicit alphanumeric `ID` (1–128 ASCII chars), `StageID`, non-empty `Description` ($\le 4096$ bytes), at least one `Preconditions` ID, a validated `Command`, and at least one `Postconditions` ID.
- Strict validation rejects:
  - Malformed action, stage, dependency, or assertion identifiers.
  - Duplicate action IDs within the graph.
  - Unknown dependencies or self-dependencies (`id == a.ID`).
  - Dependency cycles (direct and multi-node cycles detected via topological sort).
  - Duplicate dependency or assertion entries.
- Empty graphs are accepted as valid no-op executions.

### 2.2 Topological Sorting and Lexical Tie-Breaking
- Action dependency ordering is deterministic: ready actions are selected by the **lexically smallest ready action ID**, eliminating reliance on Go map iteration order.
- In-place assertion lists (`Dependencies`, `Preconditions`, `Postconditions`) are normalized via lexical sort.
- Repeated equivalent action sets produce bit-for-bit identical canonical JSON and SHA-256 graph digests (e.g. synthetic test graph digest `95acb73b1da248ab6547fb4e346a7755e5052a01018efd4ee0a68e9989e15b6f`).

### 2.3 Failure Propagation and Branch Isolation
- Dependent actions are strictly blocked (`Outcome: "blocked"`, category `dependency`) if any required predecessor fails.
- Independent graph branches continue execution after an ordinary action failure.
- Global context cancellation immediately blocks all unstarted actions (`Outcome: "blocked"`, category `canceled`).

### 2.4 Immutability Guarantee
- The `Graph` struct encapsulates canonical JSON in private unexported `data string`.
- Public methods `Actions()` and `JSON()` deserialize fresh copies; caller mutations on input slices or returned structures cannot mutate established graph definitions.

---

## 3. Bounded Subprocess Runner and Process Lifecycle (`runner.go`, `process_linux.go`)

### 3.1 Direct Execution and Environment Isolation
- Executed via `exec.CommandContext` using direct argument vectors. Shell interpreters (`sh`, `bash`, `dash`, `zsh`, `ksh`, `fish`) and shell interpolation are prohibited.
- Executables must be clean absolute paths with zero embedded NUL, CR, or LF bytes.
- Environment isolation: Process environment is strictly sanitized to `LANG=C` and `LC_ALL=C`. No host environment variables or credentials leak to subprocesses.
- Argv limits: $\le 4,096$ arguments, $\le 1$ MiB total argv size, zero embedded NUL bytes.

### 3.2 Linux Process-Group Descendant Cleanup
- Subprocesses are spawned in private process groups (`syscall.SysProcAttr{Setpgid: true}`).
- Context timeout or cancellation triggers `cmd.Cancel`, which sends `SIGKILL` to the negative PID group (`syscall.Kill(-c.Process.Pid, syscall.SIGKILL)`).
- Subprocess completion triggers `cleanupProcess`, ensuring all ordinary descendants within the group are terminated.
- Pipe termination is bounded by a 200 ms `WaitDelay`. If descendant processes hold stdout/stderr open, the delay terminates them and returns category `internal`.
- Tested and verified: descendant processes spawned by helpers are confirmed dead (or non-running zombies awaiting host PID 1 reaping) via `/proc/<pid>/stat`.

### 3.3 Diagnostic Output Bounds and Stream Limits
- Separate limits enforced for stdout and stderr (1 byte to 16 MiB).
- Excess output beyond limit is drained and discarded until process exit. Any stream overflow causes a fatal `output_limit` failure, even if the subprocess returned exit code 0.
- Truncation is tracked independently (`StdoutTruncated`, `StderrTruncated`).

### 3.4 Secret Rejection and Diagnostic Redaction
- Registered sensitive patterns (up to 64 patterns, $\le 4096$ bytes each, $\le 64$ KiB total) in executable paths or argv are rejected before process launch (`contract`).
- Exact-pattern redaction masks sensitive tokens in captured stdout, stderr, and failure diagnostics to `[REDACTED]`.
- Redaction algorithm masks across original byte buffers before replacement, correctly protecting overlapping patterns, split-chunk writes, and partial token prefixes at stream truncation boundaries.

---

## 4. Structured Error Taxonomy and Pre/Postconditions (`errors.go`)

### 4.1 Ten Distinct Failure Categories
The runner and graph enforce explicit error classification supporting `errors.Is` and `errors.As`:
1. `contract`: Invalid graph, command, or assertion binding specification.
2. `precondition`: Required eligibility assertion failed; command not started.
3. `process_start`: Executable could not start (missing binary, permission denied).
4. `nonzero_exit`: Process returned unsuccessful exit code ($\ne 0$).
5. `timeout`: Execution deadline expired.
6. `canceled`: Caller canceled execution context.
7. `output_limit`: Diagnostic capture exceeded configured limit.
8. `postcondition`: Process exited 0, but required result assertion failed.
9. `dependency`: Prerequisite action failed; dependent action blocked.
10. `internal`: Unexpected runner pipe or process state failure.

### 4.2 Postcondition Enforcement Contract
- **Exit code 0 alone does NOT constitute success.**
- If a subprocess exits 0 but an associated postcondition predicate fails, the action is marked `failed` with category `postcondition`.
- Precondition failure aborts execution before the subprocess is started.

---

## 5. Limitations and Operational Dispositions

1. **Coverage Instrumentation Interaction (Qualified Limitation):**
   - Running `go test -cover ./internal/execution` exhibits a subtest failure in `TestRunnerResults`. Investigation confirmed this is an artifact of Go's test coverage instrumentation: when the test binary invokes itself as a helper process via `os.Executable()`, the coverage runtime writes diagnostic statistics to stderr/stdout upon exit, triggering the runner's strict output overflow check.
   - Non-instrumented tests (`go test -v ./internal/execution`), race-detector tests (`go test -race ./internal/execution`), and full repository suites pass with 100% success. This is documented and accepted as a test harness interaction, not an implementation defect.
2. **In-Process Callback Cooperation:**
   - Go does not support forcible goroutine termination; caller-provided `Checker` callbacks must honor `context.Context` cancellation.
3. **Process-Group Escape Boundary:**
   - Processes that explicitly detach via `setsid` or daemonize escape process-group cleanup. AlmaDial commands must run as foreground processes.
4. **Redaction Scope:**
   - Redaction operates on exact registered byte patterns. Callers must register sensitive tokens; arbitrary unannounced or encoded secrets cannot be inferred automatically.

---

## 6. Automated Verification and Reproducible Builds

### 6.1 Go Test Suite and Race Detection
- `make fmt-check`: **PASS** (Zero formatting drift).
- `go vet ./...`: **PASS** (Zero compiler/static analysis warnings).
- `go test -v ./...`: **PASS** (56 top-level tests passed, 1 opt-in test skipped; 334 passing events).
- `go test -v ./internal/execution`: **PASS** (16 top-level tests, 60 passing events).
- `go test -race ./...`: **PASS** (Zero data races across all 7 packages).
- `go mod verify`: **PASS** (`all modules verified`).
- `go mod tidy -diff`: **PASS** (Zero module drift).

### 6.2 Controlled Reproducible Build Verification
- Two independent builds from separate isolated source directories with fixed metadata (`ALMADIAL_VERSION=dev`, `ALMADIAL_COMMIT=3b229ee77185789d00f531c74c062d575ff03488-alm13`) produced bit-for-bit identical Linux amd64 ELF binaries:
  - SHA-256: `18b561908499e5bc76bd6bda270f826d7dadfaab20367a70e193a7e5c23b52db`.
- CLI smoke tests: 14/14 tests pass with exact stream separation, exit code semantics, and non-disclosing error messages.

### 6.3 Python Regression and Repository Scaffold
- `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.841s).
- `python3 scripts/verify_candidate_stack.py`: **PASS**.
- `python3 scripts/verify_config_generation.py`: **PASS**.
- `git diff --check`: **PASS** (Zero whitespace issues).

---

## 7. Evidence Ledgers and Cryptographic Verification

### 7.1 Historical Implementation Ledgers
- **Core Ledger:** `/home/ra/almadial-lab/evidence/m1-05/20261107T025036Z/SHA256SUMS`
  - SHA-256 Digest: `d5b58df6545d5c8df0dd714312445bff5592207d4f0be0181e8f68dba3074037` (44/44 files verified **PASS**).
- **Handoff Ledger:** `/home/ra/almadial-lab/evidence/m1-05/20261107T025036Z/handoff/SHA256SUMS`
  - SHA-256 Digest: `c81544620881ab5ca48b20ec81fb57883a6d77f1d1137150bdfc318b397ec75a` (1/1 file verified **PASS**).

### 7.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-05/qa-20261107T035000Z/` (Mode `0700`)
- **Sealed Ledger SHA-256 (`SHA256SUMS`):** `dcb2a52ea722c8a24da2e8c55acddd602c445d2111e9311417a910f73b67b461`
- **Sealed Artifacts:**
  - `qa-evidence-ledger.json`: `82a35509565e75d73f6adf52c88a6670b476a07918d7f07e5ca306bcce63d5c2`
  - `qa-go-and-python-suites.json`: `3f9f2cf8e9ccbed75630650b267f537c1092e3b14c87582c08ee79cabaa80f5c`
  - `qa-graph-and-execution-tests.json`: `d74233050a70366e91c7f1228dbb41365c53cde9f5b374507586facf52aa24f5`
  - `qa-historical-ledgers.json`: `9edc69ffc6b58ee367ecc82d2e4da31b29302372f87a5e19490ccb831f94ff5a`
  - `qa-reproducible-builds-and-smokes.json`: `2e0396a37d672469f9e01123df2d2e117e2815d9996111f90260fe0b261db683`

---

## 8. Final Acceptance Verdict

# `ALMADIAL_M1_05_ACCEPTED`

Task ALM-13 satisfies all canonical acceptance criteria. The execution package provides deterministic acyclic action dependency graphs with lexical tie-breaking, direct argument-vector subprocess execution with Linux process-group cleanup, fatal stdout/stderr overflow handling, robust diagnostic secret redaction, and an unambiguous 10-category structured error taxonomy.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-14 / M1-06** (`q2kgwrrzpn4zkzgvj08y6g1h`): *Persist atomic journal and enforce exclusive execution*.
