# M1-06 — Atomic journal and exclusive execution

ALM-14 adds `internal/journal` to the accepted ALM-13 baseline
`a2de242a0d70596dc65d53da0d61cfaa957e9c56`. It is a Linux local-filesystem
persistence API, not an installation executor or operator-approval mechanism.
No CLI commands, telephony components, laboratory VMs, configuration schema,
manifest trust anchors or plan algorithms change.

## Contract and identities

Schema v1 uses strict JSON, capped at 4 MiB and 4,096 actions. `Decode(io.Reader)`
rejects malformed/truncated JSON, trailing data, invalid UTF-8, nulls, duplicate,
unknown, case-variant and missing keys, unsupported schema versions, malformed
identifiers, invalid dependency graphs, invalid states and inconsistent evidence.
No automatic migration or corrupt-state reset exists.

`Identity` requires seven lowercase SHA-256 values:

| Field | Supplied by the authorized future caller |
|---|---|
| `Host` | Stable host identity digest, distinct from changing process IDs |
| `HostState` | `preflight.Fingerprint` of fresh material observations and reviewed requirements |
| `Config` | Accepted `config.Digest` of validated effective configuration |
| `Manifest` | ALM-12 verified manifest identity; candidate status remains a separate gate |
| `Plan` | Immutable `plan.Plan.Digest()` |
| `Policy` | Digest of the caller's explicit versioned approval/recovery policy |
| `Graph` | Accepted `execution.Graph.Digest()` |

Host-state and policy digests are not replacements for ALM-11's pure
`Plan.Revalidate`. The future caller must collect fresh observations, verify
eligibility and exact plan/configuration/manifest identity, and obtain actual
operator authorization before mutating installation state. A hash match proves
neither publisher trust nor consent. ALM-14 defines no new approval workflow and
does not make a candidate manifest installable.

`New(identity, graph, retryPolicies)` derives a pending document from the actual
ALM-13 graph. A policy (`Never` or `Idempotent`) must explicitly cover each action.
The document stores action/stage IDs, sorted dependencies/postconditions, retry
policy, state, evidence digest, ALM-13 failure category and numeric exit status.
Graph metadata identifies action definitions; journal schema identifies the
persistent contract. `JSON()` is deterministic: action order is lexical,
dependency/assertion order is sorted, and no runtime timestamps enter identity.
A private serialized copy is retained by a session; `Snapshot()` returns copies.

The journal has no fields for commands, argv, resolved credentials, raw process
output, callback errors or free-form diagnostics. IDs must be non-secret ASCII
identifiers. The future caller must never encode secrets in IDs or policy input.
Failure persistence retains only the accepted category and exit code. Evidence
hashes identify externally collected checks; they do not authenticate their author.

## Action lifecycle

| Operation/observation | Durable result |
|---|---|
| `Initialize` on absent journal | All actions `pending` |
| `Begin(id)` after dependencies succeed | `running`, committed before the caller may act |
| `Finish` with successful ALM-13 trace and sufficient fresh evidence | `succeeded` |
| `Finish` with structured failed trace | `failed`, category and exit status retained |
| Successful trace with absent/unknown postconditions | `reconciliation_required`; returns `ErrReconcile` |
| Resume with `running` record | Interrupted intent; never interpreted as completion by itself |

`Begin` refuses completed actions, unsatisfied dependencies, any outstanding
running action or unresolved reconciliation. The API is sequential. It never
executes commands. A future executor must call `Begin` and check its successful
durable return **before** the first external side effect, then use the bounded
runner and postconditions before `Finish`. Wrapping the existing whole-graph
`Graph.Execute` with one journal write is insufficient: the future executor must
interpose the journal at each action boundary. This translation is outside ALM-14.

## Exclusive ownership and filesystem boundary

`Acquire(absoluteStateDirectory)` may create only the final directory; parents
must already exist. Directory mode must be exactly 0700 and owner must equal the
current effective UID. On Linux, path components are opened relative to held
directory descriptors using `O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC`. Symlink
components, unsuitable paths and foreign/permissive leaf directories are rejected.
Files must be regular, singly linked, private (0600) and owned by that UID.

The directory contains separate files:

```text
apply.lock       stable inode; never renamed, unlinked or truncated by this API
journal.json     atomically replaced complete schema-v1 document
.journal-*.tmp   unique private staging files; never recovery input
```

A nonblocking exclusive `flock(2)` on `apply.lock` is retained through the complete
apply/resume session. A competing process receives `ErrBusy` before it can read
for resume, write a journal or execute an action through the supported API.
`Close()` releases ownership; the kernel also releases it after process exit,
`os.Exit` and SIGKILL. CLOEXEC prevents descriptor inheritance through exec.
Every transition checks that the named lock file still has the held inode.
The implementation never removes/replaces the lock, including during errors.

This is advisory coordination among cooperating callers in an owner-controlled
local directory. It is not protection against root or a malicious same-UID process
that ignores locking, replaces ancestor directories or writes state directly.
Ancestors must be administratively controlled. Network filesystems and storage
which do not honor Linux flock/rename/fsync semantics are unsupported. No daemon,
PID-file-only lock, remote coordination or in-memory-only substitute is used.

## Atomic persistence and failure boundaries

Every accepted transition performs this sequence under exclusive ownership:

1. Validate and serialize the complete document before staging.
2. Create a random, exclusive 0600 temporary file in the held journal directory.
3. Write all bytes and reject short writes/errors.
4. Synchronize the temporary file with `File.Sync()`.
5. Close it, checking the close result.
6. Validate any existing journal before replacing it; corruption is not reset.
7. Atomically `renameat` the staging file to `journal.json` in the same directory.
8. Synchronize the held parent-directory descriptor.
9. Publish the new in-memory snapshot only after synchronization succeeds.

Creating a new state directory synchronizes its parent; acquisition synchronizes
the state directory after creating the stable lock entry. Failure paths attempt
to remove their incomplete staging file and preserve the published journal.
Crash leftovers are deliberately ignored, not promoted, replayed or automatically
removed during inspection. Only `journal.json` is authoritative.

`PersistenceError` reports a static operation and `Uncertain`. Before rename,
the last published journal survives; the staging file is not authoritative.
After rename but before successful directory synchronization, the new journal may
be visible while power-loss durability is uncertain. There is no claim that the
old journal was restored. **Any persistence failure poisons the session:** stop,
close, inspect and perform identity checks/reconciliation in a new session.
A failed `Begin` never licenses external effects, even if the running record has
become visible. Runtime file/directory sync, close, rename, permission and disk-full
errors fail closed without leaking private syscall paths.

Process-kill tests demonstrate process-crash behavior on the developer's Linux
filesystem. They do not emulate power removal, controller caches or every storage
implementation. The fsync/rename/directory-sync protocol is implemented; physical
power-loss testing is NOT_RUN and is not claimed.

## Resume and reconciliation

`Resume(ctx, expectedPendingDocument, observer)` reads and validates the existing
journal. The expected document must be newly derived from the current graph and
explicit policies. All seven identity fields and all action metadata must match
before callbacks or journal replacement. `IdentityError` exposes only the static
mismatched field and supports `errors.Is(err, journal.ErrIdentity)`.
Missing/corrupt/unknown-version records are explicit errors, never fresh installs.

The observer receives a defensive record copy. It is trusted, bounded, read-only
stage code which must honor the supplied context, independently evaluate **all**
postconditions, and return one of:

- `Satisfied` plus a valid evidence digest: sufficient evidence of the entire
  desired action result, including non-idempotent effects where relevant.
- `Absent` plus a valid evidence digest: a known safe pre-action state, **not**
  merely a missing file while other irreversible side effects might remain.
- `Unknown`, invalid evidence, error or cancellation: evidence is insufficient.

Callbacks must not reenter the same session. No callback stdout/error text is
persisted or echoed. There is no generic telephony, SQL or systemd reconciler.

| Persisted assertion | Fresh observation | Recovery decision |
|---|---|---|
| Succeeded | Satisfied | Remain succeeded with fresh evidence |
| Succeeded | Absent/unknown | Reconciliation required: drift, never automatic replay |
| Running/failed | Satisfied | Succeeded based on sufficient observed state |
| Running/failed | Absent; explicitly idempotent | Pending; eligible for later explicit execution, no retry executed here |
| Running/failed | Absent; never retry | Reconciliation required |
| Any previously started action | Unknown/unavailable | Reconciliation required |
| Reconciliation required | Satisfied | Succeeded with sufficient fresh evidence |
| Reconciliation required | Absent/unknown | Still blocked; repeated resume cannot erase a previous drift decision |

Dependencies are checked to a fixed point: a completed descendant cannot remain
verified after an upstream prerequisite drifts. Any unresolved record causes
`ErrReconcile` and prevents every new `Begin`. Operators must resolve ambiguous
non-idempotent state through stage-specific recovery; there is no force/reset or
automatic rollback API. The caller must stop on resume errors. Observation and
journal persistence are not an atomic transaction with the host: a future
executor must recheck relevant conditions immediately before effects.

## API examples

These snippets belong inside this Go module because `internal/journal` is not a
public API. `identity`, `graph`, `retryPolicies` and `observer` are supplied by the
future authorized caller; no installation commands exist in this package.

```go
pending, err := journal.New(identity, graph, retryPolicies)
if err != nil { return err }
session, err := journal.Acquire(stateDirectory)
if err != nil { return err } // errors.Is(err, journal.ErrBusy) identifies contention
defer session.Close()
if err := session.Initialize(pending); err != nil { return err }
if err := session.Begin(actionID); err != nil { return err }
// Only now may a separately authorized caller run this action via ALM-13.
// trace contains its actual process and postcondition result.
if err := session.Finish(ctx, trace, observer); err != nil { return err }
```

For interruption recovery, acquire a new session and call
`session.Resume(ctx, pending, observer)`. An unknown interrupted result returns
`ErrReconcile`, blocks new actions and preserves inspectable state. For read-only
inspection use `journal.Inspect(stateDirectory)`; corrupted JSON returns
`ErrInvalid`, and a missing directory/journal returns `os.ErrNotExist` without
creating anything. `Inspect` is a consistent file snapshot, not a substitute for
exclusive ownership or current host verification.

## Verification and independent QA

Use the pinned Go 1.27.3 toolchain:

```bash
go test -count=1 -v ./internal/journal
go test -race ./...
go vet ./...
go mod verify
go mod tidy -diff
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Tests cover strict parsing and identities, invalid transitions, success evidence,
failure diagnostic exclusion, drift propagation, repeated resume, read-only
configuration/preflight/planning/inspection, safe paths/modes, short writes and
ENOSPC, failures around create/write/file-sync/close/rename/directory-sync, and
concurrent atomic readers. Actual helper processes prove lock contention,
normal/crash/SIGKILL release and crashes before/after rename, including partial
staging writes. Fault hooks are private test seams; production callers cannot
inject persistence operations. Tests require neither root nor live services.
The existing opt-in M0 artifact replay is not needed for this package.

Developer verification is implementation evidence, not independent acceptance.
AGY must reproduce the critical filesystem/process checks, audit identity and
recovery boundaries, review unchanged CLI/build/config/plan/artifact contracts,
and decide acceptance separately. Hosted CI remains NOT_RUN unless recorded.

### Developer verification record

Using Go 1.27.3 on Linux amd64, the implementation run passed 88 top-level Go
tests (452 passing test/subtest events), including 32 journal top-level tests
(118 passing journal test/subtest events). The same counts passed with `-race`.
The opt-in M0 artifact replay remained skipped. All 126 Python regressions,
scaffold/candidate/config-generation checks, vet, module verification and
`go mod tidy -diff` passed. Two builds in different source/output directories with
fixed metadata produced identical Linux amd64 SHA-256
`9d8f2d95d83ba6c20185798c54ba79b5e5d9bb92908572cc49f106b2227c8c3b`.
All 11 existing executable CLI smoke cases passed. The journal package is not
linked into the help/version-only CLI yet; its runtime behavior is established by
the dedicated package/helper-process tests, not by those CLI smoke cases.
Independent AGY acceptance and hosted CI remain separate, unexecuted gates.

A repeated atomic-reader check exposed a race in the initial safety check: an
opened old journal can have zero links after atomic replacement. Read-only journal
snapshots now allow that case; lock files still require a linked inode, and both
files reject multiple hard links. The journal suite passed 100 repetitions after
the correction. Filesystem protocol checks cover tmpfs and a local disk filesystem;
these process-crash and syscall checks do not prove hardware power-loss behavior.
