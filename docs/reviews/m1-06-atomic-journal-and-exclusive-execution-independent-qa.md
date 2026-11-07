# ALM-14 / M1-06 Independent Atomic Journal and Exclusive Execution QA Report

- **Task ID:** `q2kgwrrzpn4zkzgvj08y6g1h` (ALM-14 / M1-06)
- **Task Title:** Persist atomic journal and enforce exclusive execution
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Date:** 2026-11-07T07:25:00Z
- **Repository Starting HEAD:** `a2de242a0d70596dc65d53da0d61cfaa957e9c56` (Accepted ALM-13)
- **Final Verdict:** `ALMADIAL_M1_06_ACCEPTED`

---

## 1. Governance and Reviewer Verification

### 1.1 Kaneo Task State and Ownership
- **Task Identity:** ALM-14 (`q2kgwrrzpn4zkzgvj08y6g1h`).
- **Implementation Ownership:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) is strictly preserved as the verified implementation owner. No reassignment or self-assignment took place.
- **Reviewer Identity:** Discovered via `kaneo-local` MCP `whoami` as Google Antigravity (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`).
- **Task Status Prior to Review:** `in-review` (QA Review).
- **Implementation Evidence Comment:** Verified comment `r63zmazurusdaul6ovji8pig`.
- **Dependency Graph:**
  - Upstream ALM-13 (`xrai81mxz5p8r600svlnqgts`): Verified status `done` (Accepted).
  - Relation ALM-13 $\rightarrow$ ALM-14 (`wwoamzsvoexj9ljfl0a0t7py`): Verified intact (`blocks`).
  - Relation ALM-14 $\rightarrow$ ALM-15 (`vjmmgg79z9g5580ijvqez48x`): Verified intact (`blocks`).
  - Downstream ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`): Verified status `to-do` (Backlog, unassigned).

### 1.2 Scope Discipline and Decision Record
- **Core Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Execution Boundary:** Supplies atomic filesystem persistence, exclusive advisory locking (`apply.lock`), durable execution progress records, and resume reconciliation primitives for the single-server installer.
- **Zero Host Installation:** The package executes zero real installation stages, installs zero RPM packages, modifies zero host configuration, touches zero database tables or telephony daemons, and preserves the `CANDIDATE` manifest designation.

---

## 2. Journal Document Model and Strict JSON Validation (`model.go`)

### 2.1 Bounded Schema and Invariant Enforcement
- **Document Structure:** Schema version 1, bound to seven 64-character lowercase hexadecimal SHA-256 identity digests (`Host`, `HostState`, `Config`, `Manifest`, `Plan`, `Policy`, `Graph`).
- **Action Records:** Explicitly bounded to a maximum of 4,096 actions; identifiers and stage IDs match strict ASCII identifier patterns (`^[a-z0-9][a-z0-9._-]{0,127}$`).
- **State Taxonomy:** Exact five-state lifecycle (`pending`, `running`, `succeeded`, `failed`, `reconciliation_required`).
- **Evidence Contract:**
  - `succeeded`: Requires 64-character hex evidence digest (`a.Evidence`), `a.ExitCode == 0`, and empty `Failure`.
  - `failed`: Requires a valid structured failure category from the ALM-13 taxonomy (`contract`, `precondition`, `process_start`, `nonzero_exit`, `timeout`, `canceled`, `output_limit`, `postcondition`, `dependency`, `internal`) and exit code between -1 and 255.
  - `pending`, `running`, `reconciliation_required`: Require empty `Evidence`, empty `Failure`, and exit code -1.
- **Graph Acyclicity in Stored Model:** The stored action list is sorted lexically by ID. `Validate()` verifies dependency existence and acyclicity without relying on topological storage order.

### 2.2 Strict Stream and Object Decoding (`Decode`)
- **Size Limits:** Enforces a hard 4 MiB limit (`MaxBytes = 4194304`) via `io.LimitReader(r, MaxBytes+1)`.
- **Character Encoding:** Validates strict UTF-8 before parsing (`utf8.Valid`).
- **Structural Integrity:**
  - Enforces maximum JSON nesting depth of 16 levels.
  - Rejects duplicate JSON object keys, case variations, null values for non-nullable collections, and trailing documents/tokens.
  - Disallows unknown fields (`dec.DisallowUnknownFields()`).
  - Strict key set enforcement on `identity` and `actions` objects.
- **Non-Disclosing Error Reporting:** Parser errors return standard sentinel `ErrInvalid` and never echo input bytes, paths, or parser error diagnostics that might contain sensitive operator data.

---

## 3. Linux Advisory File Locking and Concurrency Safety (`store_linux.go`)

### 3.1 Exclusive Advisory Lock (`apply.lock`)
- **Lock Mechanism:** Utilizes `syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)` on the `apply.lock` descriptor.
- **Fast Contention Failure:** Concurrent processes attempting `Acquire()` fail fast immediately with `ErrBusy` (exiting with code 23 in helper tests); no waiting, blocking, or queuing.
- **File Descriptor Hygiene:** Lock descriptors and directory descriptors are opened with `syscall.O_CLOEXEC` to prevent descriptor inheritance by subprocesses.

### 3.2 Directory and File Security Preconditions
- **Path Verification:** State directory path must be clean, absolute, and free of NUL, CR, or LF bytes. Traversal is verified component-by-component from `/` using `syscall.O_NOFOLLOW`.
- **Permission Enforcement:**
  - Directory mode must be strictly `0700`, owned by the process effective UID (`os.Geteuid()`).
  - Lock file and journal files must be regular files (`S_IFREG`), mode `0600`, owned by `os.Geteuid()`, with hard link count $\le 1$.
  - Symlinks, hard links, FIFOs, relative paths, and loose permissions are rejected with `ErrUnsafe`.

### 3.3 Lock Inode Stability and Kernel Cleanup
- **Inode Stability:** The `apply.lock` inode is created during initial `Acquire()` and remains pinned across all subsequent atomic operations. It is never unlinked or replaced during journal updates.
- **Displacement Detection:** `Session.owned()` verifies that the locked descriptor matches the current directory entry inode (`os.SameFile`). Unlinking or replacing `apply.lock` under a running session causes subsequent operations to fail with `ErrUnsafe`.
- **Kernel Release Verification:** Tested and verified across three distinct lifecycle termination modes:
  1. Normal clean shutdown (`s.Close()`, exit code 0): lock cleanly released.
  2. Abnormal crash termination (`os.Exit(41)`): kernel automatically releases flock upon process termination.
  3. Forcible termination (`SIGKILL`): kernel immediately cleans up file descriptors and releases flock, allowing subsequent callers to acquire without manual intervention or lockfile deletion.

---

## 4. Atomic Persistence, Durability, and Failure Poisoning (`store_linux.go`)

### 4.1 Seven-Stage Staging and Durability Sequence
All mutations commit via an exact, verified atomic replacement pipeline:
1. `create`: Allocate temporary file `.journal-<token>.tmp` with 16 random hex bytes, mode `0600`, using `O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC`.
2. `write`: Write canonical JSON bytes and verify complete write (short writes return `io.ErrShortWrite`).
3. `file_sync`: Execute `f.Sync()` to flush file data and inode metadata to disk.
4. `close`: Close temporary file descriptor and verify zero errors.
5. `rename`: Atomic replacement of `journal.json` via `syscall.Renameat` from temporary leaf.
6. `directory_sync`: Execute `s.dir.Sync()` on the directory file descriptor to guarantee durable directory metadata on disk.
7. `committed`: Update in-memory session snapshot.

### 4.2 Durability Uncertainty and Session Poisoning
- **Durability Classification:** Any error returns a structured `*PersistenceError`:
  - Failures occurring before rename (`create`, `write`, `file_sync`, `close`, `rename`) report `Uncertain: false`.
  - Failures occurring after rename (`directory_sync`, `committed`) report `Uncertain: true`, indicating the new journal is already visible on the filesystem but directory entry durability is uncertain.
- **Session Poisoning:** Any persistence error permanently poisons the session (`s.poisoned = true`). Subsequent mutating calls return `ErrClosed`, forcing the caller to exit, re-inspect, and reconcile state.
- **Temporary File Cleanup:** Deferred cleanup unlinks uncommitted temporary staging files on error.

### 4.3 Concurrent-Reader Snapshot Safety
- **Atomic-Reader Guarantee:** Concurrent reader goroutines executing `Inspect()` observe only complete, valid journal documents across rapid atomic replacements; zero partial or corrupt states are visible.
- **Unlinked Inode Handling:** When an atomic rename replaces `journal.json`, an already-opened reader descriptor observes `st.Nlink == 0`. The implementation explicitly accommodates `st.Nlink == 0` for read-only snapshot descriptors while strictly enforcing `st.Nlink == 1` for `apply.lock` and newly opened files.

---

## 5. Crash Recovery, Identity Binding, and Resume Reconciliation (`recovery_linux.go`)

### 5.1 Cryptographic Identity Verification
Before inspecting progress or invoking observation checks, `Resume()` validates the persisted state against the current caller's expected `Document`:
- All seven cryptographic digests (`Host`, `HostState`, `Config`, `Manifest`, `Plan`, `Policy`, `Graph`) must match exactly. Any mismatch returns `*IdentityError` with the mismatched field name.
- Action metadata (action IDs, stages, dependencies, postconditions, and retry policies) is normalized and compared against the expected document. Any modification returns `*IdentityError{Field: "action_metadata"}`.
- Zero state mutation or observer probing occurs if identity validation fails.

### 5.2 Fresh Observation and Safe Retry Policy
- **Fresh Postcondition Audit:** For all previously started actions (`Running`, `Succeeded`, `Failed`), `Resume()` invokes trusted `Observer` callbacks with defensive copies:
  - If observed state is `Satisfied` with a valid evidence hash: action becomes `Succeeded` with verified exit code 0.
  - If observed state is `Absent`:
    - If prior state was `Running` or `Failed` AND retry policy is `Idempotent`: action transitions to `Pending` for safe re-execution.
    - If retry policy is `Never` (non-idempotent) or prior state was `Succeeded`: action transitions to `ReconciliationRequired`.
  - If observed state is `Unknown` or observer returns an error: action transitions to `ReconciliationRequired`.
- **Fixed-Point Dependency Drift Cascade:**
  - If an action's postconditions drift from `Satisfied` to `Absent`/`Unknown`, any downstream actions that previously succeeded can no longer remain valid.
  - The reconciliation engine iterates over the action graph until reaching a fixed point, cascading all dependent actions to `ReconciliationRequired`.
- **Durable Drift Blocking:** If any action remains in `ReconciliationRequired`, `Resume()` durably commits the reconciled document to disk and returns `ErrReconcile`. Subsequent invocations cannot erase drift or bypass operator intervention.

---

## 6. Limitations, Operational Dispositions, and Ext4 Verification

1. **Physical Filesystem Durability Verification:**
   - Linux directory `fsync` and atomic rename durability depend on underlying filesystem journaling semantics.
   - The test suite was executed 5 consecutive times on the physical `ext4` filesystem (`/home/ra/almadial-lab/...`) with zero failures, zero race conditions, and complete file integrity.
2. **Advisory Locking Contract:**
   - Linux `flock` is advisory. Uncooperating host processes running as root can bypass advisory locks; AlmaDial components must all cooperate through the `journal.Acquire` entrypoint.
3. **Observer Isolation:**
   - Observer implementations are trusted in-process Go components. They receive defensive copies of action records, and observer errors are classified safely as `Unknown`. Observers must respect `context.Context` cancellation.
4. **Zero-Mutation Read-Only Invariant:**
   - Tested and verified: `Inspect()`, `config.Defaults()`, `preflight.Assess()`, and `plan.New()` perform zero filesystem writes and create zero files or lockfiles.

---

## 7. Automated Verification and Reproducible Builds

### 7.1 Go Test Suite and Race Detection
- `make fmt-check`: **PASS** (Zero formatting drift).
- `go vet ./...`: **PASS** (Zero compiler/static analysis warnings).
- `go test -v ./internal/journal`: **PASS** (32 top-level tests, 86 subtests, 118 passing events; 476 total events).
- `go test -v ./...`: **PASS** (88 top-level tests passed, 1 opt-in test skipped; 364 subtests passed; 452 passing events; 2046 total events).
- `go test -race ./...`: **PASS** (Zero data races detected across all packages).
- `go mod verify`: **PASS** (`all modules verified`).
- `go mod tidy -diff`: **PASS** (Zero module drift).

### 7.2 Controlled Reproducible Build Verification
- Two independent builds from separate isolated source directories with fixed metadata (`ALMADIAL_VERSION=dev`, `ALMADIAL_COMMIT=a2de242a0d70596dc65d53da0d61cfaa957e9c56-alm14`) produced bit-for-bit identical Linux amd64 ELF binaries:
  - SHA-256 Digest: `9d8f2d95d83ba6c20185798c54ba79b5e5d9bb92908572cc49f106b2227c8c3b` (Exact match with developer handoff).
- CLI smoke tests: 14/14 tests pass with exact stream separation, exit code semantics, and non-disclosing error messages.

### 7.3 Python Regression and Repository Scaffold
- `python3 scripts/validate_scaffold.py`: **PASS** (11 required files, 4 skills).
- `python3 -m unittest discover -s scripts/tests -v`: **PASS** (126/126 unit tests passed in 1.835s).
- `python3 scripts/verify_candidate_stack.py`: **PASS**.
- `python3 scripts/verify_config_generation.py`: **PASS**.
- `git diff --check`: **PASS** (Zero whitespace issues).

---

## 8. Evidence Ledgers and Cryptographic Verification

### 8.1 Historical Implementation Ledgers
- **Core Ledger:** `/home/ra/almadial-lab/evidence/m1-06/20261107T054414Z/SHA256SUMS`
  - SHA-256 Digest: `a379911c3869a0d2d38c0e800c10c4ba5923c8ff5bf6c8f63d4fcdbec944e331` (42/42 files verified **PASS**).
- **Handoff Ledger:** `/home/ra/almadial-lab/evidence/m1-06/20261107T054414Z/handoff/SHA256SUMS`
  - SHA-256 Digest: `b69c4c81d68a30c009367ee07f7384e312826c79b224c0f37d4a5991b69edc0e` (6/6 files verified **PASS**).

### 8.2 Independent QA Evidence Directory
- **Directory Path:** `/home/ra/almadial-lab/evidence/m1-06/qa-20261107T072500Z/` (Mode `0700`)
- **Sealed Ledger SHA-256 (`SHA256SUMS`):** `751f8ae066eda015311400533de4c42abd90aa1acbaf8dc6294cf8967835dd85`
- **Sealed Artifacts:**
  - `qa-evidence-ledger.json`: `a4b087a3bfb2cb79a32c3228807d47efeaae97e14ba57c430e71f4fc1fa98099`
  - `qa-go-and-python-suites.json`: `94d6e91febe89f25fb3eb3d57d42cfc089fa08cc4bbf7a188f5f190ddb05f6cb`
  - `qa-historical-ledgers.json`: `a5c0b1f237bf4cb003b15dc6f9cf53a2588c2278479e0a248f219cebca17c677`
  - `qa-journal-and-locking-tests.json`: `e1e2d78bfb0c8008a1d7c3bb83b27b3704efc354673bbab994a50dca8705ecbe`
  - `qa-reproducible-builds-and-smokes.json`: `34559df4a329d99d107a61d157a414e21a2247fb7d1ce57579db4a9b58941783`

---

## 9. Final Acceptance Verdict

# `ALMADIAL_M1_06_ACCEPTED`

Task ALM-14 satisfies all canonical acceptance criteria. The persistence package provides strict schema version 1 document validation, exclusive advisory process locking via `apply.lock` with guaranteed kernel release across crashes and signals, atomic file replacement with directory metadata synchronization, strict durability uncertainty tracking and failure poisoning, cryptographic identity binding across seven digests, and safe fixed-point resume reconciliation with dependency drift cascade.

**Recommended Next Assignment:**
Authorize Codex to begin **ALM-15 / M1-07** (`r6qm3nzbahxpvcw8ejk71s0q`): *Deliver accessible wizard and redacted operator logging*.
