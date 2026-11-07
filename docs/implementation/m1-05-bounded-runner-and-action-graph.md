# M1-05 — Bounded runner and deterministic action graph

ALM-13 adds `internal/execution` to the accepted Go 1.27.3 module. Starting HEAD
is `3b229ee77185789d00f531c74c062d575ff03488` (Accepted ALM-12). This is
implementation evidence for independent AGY review, not independent acceptance.
No installer command, installation stage provider, journal or CLI change is added.

## Scope and authorization boundary

The package supplies local execution primitives for trusted Go installer code.
It does not read arbitrary commands from user configuration. It does not authorize
installation, assess host eligibility, resolve secrets, verify source artifacts,
or translate plan stages. The existing CLI remains help/version only.

Future integration is: validated effective configuration and fresh immutable plan
→ verified artifacts and actual operator authorization → trusted stage-specific
actions → graph validation → bounded runner → verified action results. Callers
must enforce every gate before invoking `Execute` or `Run`. Graph identity,
preconditions, manifest checksums and process exit zero do not establish consent.
The candidate stack lock and accepted configuration/plan contracts are unchanged.

No files, services or packages are owned by this foundation. It starts only the
explicit local executable requested by its authorized caller. There is no retry,
rollback or resume: return the failure and preserve safe results for a future
journal. Repeating a graph can repeat effects; stage-specific idempotency and
recovery contracts must precede any installation integration.

## Action contract and immutable graph

`Action` contains `ID`, `StageID`, description, dependency IDs, precondition IDs,
`Command`, and postcondition IDs. A command specifies an absolute executable,
non-secret argv, positive timeout, and explicit stdout/stderr byte limits.
Success means numeric process exit zero **and all postconditions passing**.
At least one precondition and one postcondition is mandatory. Assertions are
trusted Go read-only checks supplied through `Checker`, which declares `Supports`
and `Check(ctx, id)`. No registry or dynamic plugin loader is provided.
Checkers must be bounded and honor cancellation; Go cannot forcibly terminate a
misbehaving in-process callback. Unsupported assertion bindings reject the whole
graph before the first command.

`NewGraph([]Action)` rejects invalid IDs (1–128 ASCII alphanumeric/dot/underscore/
hyphen, beginning alphanumeric), empty stages/descriptions, invalid commands,
duplicate actions or assertion/dependency IDs, unknown dependencies, self edges
and cycles. Empty graphs are valid successful no-ops. A zero-value `Graph` is
uninitialized and cannot execute. Description length is limited to 4,096 bytes.

Topological sorting selects the **lexically smallest currently ready action ID**
after each selection. Dependency and assertion lists are sorted lexically.
Assertion ordering is therefore part of the contract, not declaration order.
Equivalent action sets produce identical ordering regardless of declaration order
or map iteration. All commands are validated through `CommandRunner.Validate`
before effects, including registered-secret argv rejection by the real runner.

`Graph` stores only private canonical JSON. `Actions()` and `JSON()` return fresh
copies; mutation of inputs or returned slices cannot change an existing graph.
`Digest()` is lowercase SHA-256 of those exact JSON bytes. Command timeouts are
JSON integer nanoseconds. There are no timestamps, random IDs, callbacks or
resolved secrets in graph metadata. Identity describes definitions, not the
implementation or truth of assertions. Only non-secret definitions may enter a
graph; parsing cannot recognize arbitrary unknown passwords disguised as args.

Execution is sequential. An unsuccessful predecessor blocks its dependents with
`dependency`; unrelated branches continue after ordinary failure. Cancellation
prevents further commands. Each action has one deadline spanning preconditions,
command and postconditions, bounded further by the caller context. The returned
error is the first failure in deterministic order; callers must also inspect all
traces. No failed process runs postconditions or marks dependencies satisfied.

## Bounded subprocess behavior

`NewRunner(sensitivePatterns)` copies up to 64 exact byte patterns (1–4,096 bytes
each, at most 64 KiB total). A zero-value `Runner` has no registered patterns.
A nil runner is invalid. Each invocation owns its process and buffers; runner
instances can be shared by concurrent callers without mutable global state.

The runner uses `exec.CommandContext` and explicit argv. There is no shell
interpolation, PATH lookup, SSH, download or installation logic. Executable paths
must be absolute and clean with no NUL/CR/LF. Common shell executables (`sh`,
`bash`, `dash`, `zsh`, `ksh`, `fish`) are rejected. This is not an executable
allowlist or sandbox: aliases, interpreters and trusted binaries still require
review by their caller. Absolute paths avoid PATH substitution but do not prove
a file's ownership, signature, stability or executable identity. Later execution
integration must establish those properties at point of use.

At most 4,096 args / 1 MiB of argv bytes are accepted, with no embedded NUL.
The inherited environment is replaced with `LANG=C` and `LC_ALL=C`. Help/version
and the accepted CLI environment remain unaffected. No credentials are supplied
through argv, environment or stdin by this foundation. Future credential delivery
may use restricted descriptors or controlled stdin after its own reviewed
contract; no resolver or credential transport is implemented here. Registered
sensitive patterns in either executable or argv reject the command before start.
Unregistered or encoded secrets cannot be automatically identified; callers must
never construct secret-bearing action metadata.

Timeout must be positive and at most 24 hours. Each stream has an explicit limit
of 1 byte through 16 MiB. Capture keeps the first limit bytes and drains/discards
remaining bytes until exit or deadline. Excess output is always a failure, even
if the process exits zero. Truncation is reported separately for stdout/stderr.
Binary output is captured as bytes in Go strings; JSON consumers receive Go's
normal invalid-UTF-8 replacement behavior rather than a claim of binary fidelity.

On Linux, the child has a private process group. Cancellation/deadline sends
SIGKILL to that group; `Wait` reaps the direct child. Remaining ordinary descendants
are killed when the child completes, even after success. A 200 ms `WaitDelay`
bounds inherited output-pipe shutdown. A successful parent leaving pipes open
causes `internal` failure, not successful verification. Processes that escape the
group using `setsid`, change ownership, or daemonize are outside this trusted-code
contract; no cgroup/container supervision is claimed. Use foreground commands.
An orphan zombie may await host PID 1 reaping; tests distinguish this nonrunning
state from a surviving process. Other OS targets reject execution before start.

## Diagnostics, redaction and errors

Capture is bounded before diagnostics are published. Full registered patterns,
overlaps, and any captured suffix matching a token prefix are masked together
against original bytes, then replaced with `[REDACTED]`. This handles split writes,
truncation inside a token and abrupt process termination. Redacted output is capped
again to the stream limit; the marker itself may be shortened at tiny limits.
Conservative masking can hide harmless suffixes. Arbitrary encodings, tokens
split across separate streams, or tokens not registered are outside exact-pattern
redaction. No unredacted buffer is returned or logged. Test evidence records
outcomes, not artificial secret output.

`Error` carries stable category, safe action/stage/assertion IDs where known,
static diagnostic guidance, numeric exit status (`-1` unavailable or signaled),
and only safe context causes. Raw assertion, OS start and wait errors, executable
paths and argv are never interpolated into package-produced messages.

| Category | Meaning |
|---|---|
| `contract` | Invalid graph/command/binding; do not execute |
| `precondition` | Eligibility assertion failed; command not started |
| `process_start` | Explicit executable could not start |
| `nonzero_exit` | Started process returned unsuccessful status |
| `timeout` | Deadline expired |
| `canceled` | Caller canceled execution |
| `output_limit` | A stream exceeded its capture budget |
| `postcondition` | Exit zero did not establish expected result |
| `dependency` | Required predecessor did not succeed |
| `internal` | Unexpected runner result or pipe-lifecycle failure |

Runner precedence is context timeout/cancellation, overflow, then process/wait
failure. An overflow plus nonzero exit retains the exit code while reporting
`output_limit`. Use `errors.Is(err, &execution.Error{Category: execution.Timeout})`
and `errors.As` for structured details. `context.Canceled` and
`context.DeadlineExceeded` are also available through `errors.Is`.

A `Trace` records action/stage, `succeeded`/`failed`/`blocked`, start flag, exit code,
separate bounded diagnostics, stream truncation flags and structured failure.
Timeout/cancel classifications are explicit failure categories. Trace order and
outcomes are deterministic for identical observed results; runtime output and
process results naturally remain observations, not deterministic predictions.
No timestamp is included. Custom runners/checkers are trusted integration code
and must honor these safety contracts; they are not untrusted input plugins.

## Safe API example

Within the module, a harmless explicit command can demonstrate the primitives:

```go
runner, err := execution.NewRunner(nil)
if err != nil { return err }
graph, err := execution.NewGraph([]execution.Action{{
    ID: "probe", StageID: "verification", Description: "Harmless process check",
    Preconditions: []string{"approved-local-example"},
    Command: execution.Command{
        Executable: "/usr/bin/true", Timeout: time.Second,
        StdoutLimit: 4096, StderrLimit: 4096,
    },
    Postconditions: []string{"expected-process-result"},
}})
if err != nil { return err }
// checks is a caller-provided Checker implementing these two read-only assertions.
traces, err := graph.Execute(ctx, runner, checks)
// Inspect err and every trace. This example is not an installation command.
```

## Verification and independent QA

Tests execute the Go test binary as a harmless helper process. They exercise
literal shell metacharacters, environment isolation, success/nonzero/start errors,
per-stream bounds and binary output, deadline/cancellation, descendant cleanup,
redaction overlaps and every token truncation position, assertion failures,
unbound checks, graph validation, identity/immutability and partial failure.
Synthetic checker results are not evidence of host installation readiness.

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod verify
go mod tidy -diff
go test ./internal/execution -count=1 -v
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Use the accepted `scripts/build_go.py` with fixed `ALMADIAL_VERSION` and
`ALMADIAL_COMMIT` in two separate source directories; compare executable SHA-256
and exercise accepted help/version/invalid-command exits. Existing artifact HTTPS
unit tests use loopback test servers and may require a sandbox network allowance.
No upstream acquisition, M0 VM inspection or actual installation is needed.
Protected results belong under `~/almadial-lab/evidence/m1-05/`, outside Git.
Hosted CI and independent AGY acceptance remain separate gates.

Developer verification on 2026-11-07 passed with Go 1.27.3: 56 top-level Go
tests (334 passing test events including subtests), with identical counts under
the race detector. The execution package contributes 16 top-level tests / 60
passing test events. The existing opt-in `TestM0Artifacts` was skipped; this task
does not rerun accepted M0 source qualification. All 126 Python regression tests,
vet, formatting, module verification/tidy consistency, scaffold, candidate-stack,
configuration-generation and whitespace checks passed. Hosted CI was NOT_RUN.

Two separate source directories built Linux amd64 ELF executables with fixed
`ALMADIAL_VERSION=dev` and
`ALMADIAL_COMMIT=3b229ee77185789d00f531c74c062d575ff03488-alm13`.
Both SHA-256 values were
`18b561908499e5bc76bd6bda270f826d7dadfaab20367a70e193a7e5c23b52db`.
Eleven executable smoke cases preserved existing help/version and usage-error
behavior. No CLI feature or accepted source was changed.

An optional coverage-instrumented execution test run failed because the helper
binary emits coverage-runtime diagnostics into streams tested for exact content.
Coverage instrumentation of these subprocess fixtures is not qualified here;
ordinary, focused and race-instrumented execution tests passed. An initial
sandbox-only full-suite attempt could not bind existing local HTTPS test servers;
the final full and race suites passed with the required loopback allowance.
