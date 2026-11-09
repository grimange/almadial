# M1-07 — Accessible wizard and redacted operator logging

Task: ALM-15 (`r6qm3nzbahxpvcw8ejk71s0q`). Developer implementation based on
accepted ALM-14 commit `c32a88a24613330f7dd9576deebd00be55f0b463`.
Independent AGY QA and the separate M1 gate review remain required.

## Scope and integration

The `wizard` command prepares installation inputs, observes the **local** host,
shows the accepted immutable plan and establishes a transient approval contract.
It cannot install software, start/resume an installation, create a journal or
acquire execution locks. `install` and `resume` remain unsupported commands.
The target IP describes a local installation address, not a remote SSH target.
No VM is accessed by this implementation or its verification harness.

Reuse boundaries:

- `config.Load`, `Defaults`, `Validate`, `Digest` and `ValidateUnattended`: the
  accepted schema/version, strict JSON, defaults and credential reference policy.
- `preflight.Collect` with `preflight.Linux`, `Assess` and explicit `Requirements`:
  existing bounded read-only observations and eligibility classifications.
- `plan.New` and `Plan.Human`: ordered stages, risks, dependencies, intended
  changes, postconditions and immutable identities. Two read-only identity
  accessors expose config/manifest bindings without changing serialized plans.
- `artifacts.Load` and `InstallationAuthorized`: strict manifest identity policy.
  Artifact downloads/content/signatures are **NOT_RUN** in the wizard.
- ALM-13 `execution.Category` and `Trace`: structured progress/error integration.
- ALM-14 journal/locking APIs remain unchanged and are never called by the wizard.

The two implementation-plan documents are byte-identical at this baseline.
Historical AGENTS/GEMINI/project descriptions still describe M0 research; current
Kaneo read-back shows ALM-1 through ALM-14 Accepted. This assignment authorizes
M1-07 framework implementation, not candidate promotion or stack installation.
Proposed future `install` behavior and richer exit codes in plan §6 remain deferred;
this task preserves the accepted M1 CLI's 0/1/2 contract.

## Commands and bootstrap

```text
almadial wizard --help
almadial wizard --manifest manifests/stack.lock.yaml --requirements /private/budget.json
almadial wizard --manifest manifests/stack.lock.yaml --requirements /private/budget.json --config /private/config.json
```

Build with the pinned Go 1.27.3 toolchain using `make build`; the binary is
`dist/almadial`. Run preparation as an ordinary user. Missing privileges/tools
produce NOT_VERIFIED observations; the wizard never escalates privileges or
changes the host to force a passing result. `help`, `--help`, `-h`, `version` and
`--version` retain their existing routing and stdout/stderr semantics.

Requirements JSON maps directly to the existing `preflight.Requirements` type.
Supply an **explicitly reviewed** budget; this example is a synthetic test input,
not a production sizing recommendation:

```json
{
  "Profile": "synthetic-only-not-production",
  "MinCPUs": 1,
  "CPUFlags": ["sse2"],
  "MinTotalMemory": 1024,
  "MinAvailableMemory": 512,
  "MinDiskAvailable": 4096
}
```

Memory and disk fields are bytes. Missing/invalid budget observations block
eligibility through the existing assessor. Requirements are bounded to 64 KiB,
reject unknown fields/trailing documents, and have no hidden resource defaults.
Config and manifest inputs use their accepted strict parsers and byte limits.
CLI input files must be regular files; final symlinks, devices and FIFOs are
rejected without blocking reads. Parent input-directory symlinks are permitted;
these are read-only inputs, not trusted authorization sources.

## Interactive workflow

Both stdin and stdout must pass Linux terminal detection (`TCGETS`); a character
device alone is insufficient. `--unattended` always disables prompts. No color,
ANSI positioning, animations, full-screen state or mouse input is used. Ordinary
terminal wrapping supports narrow SSH terminals; long digests wrap as text.

Prompts appear in a fixed order: local bind IP; advertised IP; optional hostname;
SIP UDP port; RTP first/last port; optional WebRTC; four credential references;
and, when enabled, WSS port and certificate/private-key file references.
Current non-secret values are shown. Enter retains the displayed value; `-`
clears optional text. Defaults come exclusively from `config.Defaults`.
Syntax errors repeat the current prompt; model validation errors identify the
field and allow correction without restarting the entire questionnaire.

IP-only configuration needs neither a hostname nor WebRTC. Explicit wildcard
binding still requires an advertised IP through the accepted configuration rules.
Credential prompts accept `env:NAME` or `file:/absolute/path` references, never
passwords. They do not read environment values, credential files or TLS contents,
create a secret store, save configuration, or generate bootstrap credentials.
The loaded JSON remains unchanged. Operators who need reusable inputs should
prepare a schema-1 JSON file using the accepted ALM-10 examples.

Type `cancel` at any prompt. EOF, partial final lines, input failures, input lines
over the 4096-byte reader bound, Ctrl+C and SIGTERM fail closed. Cancellation
closes the process's stdin to wake an outstanding terminal read; package callers
must similarly close their input on cancellation. No prompt reader runs between
questions. Terminal output failure does not yield an approved decision.

## Plan review and approval

The presentation separates configuration validity, host eligibility, manifest
status, unexecuted artifact verification, installation authorization and execution
status. It uses actual findings: PASS, WARNING, FAIL and NOT_VERIFIED. Mandatory
failures/missing observations remain blockers; optional warnings remain warnings.
Each blocker includes investigation guidance and states that installation cannot
continue. The accepted plan supplies the detailed finding and stage information.

```text
almadial wizard --manifest manifests/stack.lock.yaml --requirements /private/budget.json --config /private/config.json --review-only
```

Review-only returns success when review output is delivered, even if the displayed
plan is blocked. It explicitly reports approval NOT REQUESTED and execution NOT
STARTED. It cannot be combined with approval flags. It does not claim host readiness.

For eligible, authorized inputs, the interactive approval interface names both
config and plan SHA-256 identities, explains future mutation and listed risks,
and asks yes/no with default **no**. `yes`/`y` are affirmative, case-insensitively;
`no`/`n`, empty input, EOF or cancellation never approve. Invalid yes/no answers
repeat the question. The decision binds the immutable plan's own config/manifest
identities, effective config digest and plan digest, including material host input.
Changes invalidate the earlier decision. A digest identifies inputs; it does not
authenticate an operator or establish human consent by itself.

**The shipped manifest is CANDIDATE. Its accepted authorization method always
returns false.** The CLI does not request affirmative approval for blocked or
unauthorized plans. A prompt or approval flag cannot promote the manifest.
Positive decision tests use a private package test seam for synthetic authorized
inputs; no public override, trusted caller boolean or CLI bypass exists.
No successful future-mutating approval is possible with the current artifact policy.

## Unattended workflow

No-TTY use requires explicit `--unattended --config FILE`. Redirected stdin is
never treated as wizard answers or as configuration. A script cannot fall back
to interactive prompting. For read-only review:

```text
almadial wizard --unattended --review-only --config /private/config.json --manifest manifests/stack.lock.yaml --requirements /private/budget.json
```

The future approval invocation adds `--approve --config-sha256 DIGEST
--plan-sha256 DIGEST` and omits `--review-only`. Approval flags require unattended
mode. `config.ValidateUnattended` validates intent/effective digest; the same
preparation and immutable-plan contract checks exact plan identity, eligibility
and manifest authorization. Missing approval, wrong digests, changed config/plan,
failed preflight and candidate manifests reject deterministically. Validation
order is configuration/approval digest, immutable bindings, eligibility/manifest,
then plan digest; a blocked plan can therefore report the blocker before a missing
plan digest. No scenario executes installation actions.

Exit codes: **0** requested review/decision completed; **1** cancellation, input or
output failure, denied/blocked approval or logging failure; **2** invalid CLI,
invalid configuration/input file or missing unattended/terminal inputs.
Preparation errors identify stage, code, NOT STARTED status and a safe next action.

## Progress and failures

`operatorui.Event` supports stage_pending, stage_started, action_started,
action_completed, action_failed, stage_completed, stage_blocked,
execution_canceled and reconciliation_required. `EventFromTrace` adapts actual
ALM-13 outcomes. It does not infer an entire stage's success from one action.
`RenderEvent` preserves caller sequence and includes stage/action IDs, failure
category, safe diagnostics and static next-step guidance. No clocks, random IDs
or map iteration affect rendering. This task tests synthetic sequences; no CLI
option simulates installation progress for an operator.

ALM-13 categories remain contract, precondition, process_start, nonzero_exit,
timeout, canceled, output_limit, postcondition, dependency and internal. Arbitrary
error strings and `execution.Error.Detail` are not trusted diagnostic text.
Registered-secret diagnostics are redacted; unknown events/categories, missing
action IDs and contradictory completion/failure or cancellation categories are
rejected before any sink receives output.
The renderer is not an executor, event state machine, broker or daemon.

## Logging, privacy and redaction

`operatorlog.Logger` emits stable JSON lines with level, stage, action, code,
message and next. INFO routes to stdout; WARNING/ERROR to stderr. An optional file
receives the same already-redacted records. The CLI logs preparation outcome
summaries only; prompts, credential references, full configs, process output and
plans are not copied into its log. Argument/parser failures before presentation
are reported safely on stderr. No DEBUG or remote telemetry is provided.

Each textual field/merged diagnostic is bounded to 16 KiB before masking;
truncation is marked. JSON escaping/metadata add record overhead. Each sink receives
at most 1 MiB per logger, including complete JSON lines. A record exceeding the
remaining budget is rejected; partial/failed writes poison the logger. File sync
and close failures cause CLI failure. Disk-full and short-write behavior are
fault-tested with controlled writers, not by filling the workstation filesystem.

`--log-file /private/new-run.jsonl` explicitly authorizes a new file. The parent
must already exist, be owned by the effective user and have mode 0700. Linux
component-by-component descriptor traversal rejects every symlink. Creation uses
0600, O_EXCL and O_NOFOLLOW; an existing file is never overwritten/appended.
No directories, journal, locks, retention daemon or rotation files are created.
Retention policy is one bounded file per explicit destination; the operator
chooses another new path and handles retention. Logging is the only optional
write performed by preparation. File output is diagnostic, not durable action state.

Register up to 64 exact nonempty byte patterns, each at most 4096 bytes, with
64 KiB total registration. Mask all overlapping occurrences against original
bytes, including repeated patterns and incomplete suffixes at capture/truncation
boundaries. Strip terminal controls after masking. Redaction occurs before any
sink receives a record. Temporary bounded byte copies are cleared; the logger
retains counters, sinks and registered patterns, not raw message history. Go does
not guarantee secure erasure of caller strings or runtime copies.

**Supported cross-stream boundary:** submit consecutive chunks/streams in caller
order as fragments of ONE record (`Redactor.Text(fragments...)` or
`Logger.Write(record, fragments...)`). Fragments are joined without separators
before masking. `EventFromTrace` passes stdout then stderr as a combined diagnostic;
this is deterministic, not a claim about original process interleaving. ALM-13's
own separate capture/redaction contract remains intact. Independent log records
are not correlated; callers must not publish secret fragments as separate
records or write unredacted fragments to a sink first. Unknown, transformed or
encoded secrets are not automatically detected. Callers must register all known
sensitive material before submitting diagnostics. The CLI resolves no secrets.

## Verification and M1 integration readiness

Run the repository Go/Python checks and:

```text
go test -count=1 -v ./internal/operatorui ./internal/operatorlog ./internal/cli
python3 scripts/test_terminal_wizard.py --binary /tmp/almadial --output /tmp/new-pty-evidence
```

The PTY harness tests a real executable, ordinary/narrow terminals, IP-only input,
invalid correction, EOF, Ctrl+C, cancel, redirected output and unattended behavior.
Eligible preflight/positive approval fixtures are synthetic package tests. Local
CLI findings are workstation observations, not accepted VM qualification.
Hosted CI and independent AGY QA are NOT_RUN by this implementation worker.

| Work package | Accepted baseline / integration boundary |
|---|---|
| ALM-9 / M1-01 | Accepted; CLI help/version and pinned reproducible build |
| ALM-10 / M1-02 | Accepted; strict typed config, defaults, digest and unattended intent |
| ALM-11 / M1-03 | Accepted; read-only observations, findings and immutable plan |
| ALM-12 / M1-04 | Accepted; manifest/artifact verification; candidate remains unauthorized |
| ALM-13 / M1-05 | Accepted; bounded runner, action graph and error taxonomy |
| ALM-14 / M1-06 | Accepted; atomic journal, exclusive ownership and reconciliation |
| ALM-15 / M1-07 | Developer implementation; independent QA and M1 gate pending |

Future executor responsibilities remain: approved manifest authorization; selected
artifact verification; effective config validation; fresh host eligibility and
plan identity; explicit consent at the mutation boundary; exclusive journal
ownership; durable intent; bounded action; independently checked postcondition;
and durable result. Observability cannot satisfy any missing execution gate.
Subsequent stage work must define owned resources, retry/recovery boundaries and
real lab postconditions before implementing native package/database/telephony/web
changes. This report does not accept M1 or authorize M2. Next assignment is AGY
independent ALM-15 QA, followed by the separate M1 integration gate review.

### Developer verification record (2026-11-07)

Baseline and current HEAD: `c32a88a24613330f7dd9576deebd00be55f0b463`.
Go toolchain: `go1.27.3 linux/amd64`; no dependency changes.

- `go fmt ./...`, `make fmt-check`, `go vet ./...`: PASS, exit 0.
- `go test -count=1 -json ./...`: PASS, 104 top-level tests and 530 passing
  test events including subtests; one opt-in `TestM0Artifacts` skip.
- `go test -race -count=1 -json ./...`: PASS, the same counts and skip.
- `go mod verify`, `go mod tidy -diff`: PASS, exit 0 with no module changes.
- Scaffold validation, candidate-stack verification, configuration-generation
  verification, and `git diff --check`: PASS, exit 0.
- Python unittest discovery: PASS, 126 tests.
- Focused wizard/logging/CLI Go tests: PASS. Executable smoke checks: PASS, 17 cases.
- Real executable PTY/nonterminal harness: PASS, 11 journeys, including a 32-column
  terminal and actual EOF/Ctrl+C keystrokes.
- Two Linux amd64 builds from separate source directories with fixed metadata:
  identical SHA-256 `3213f3631068449095722c36cab98df990938ee1f06e721679d2c4d0a9077876`.

The initial sandboxed full Go runs failed because existing artifact HTTP tests
could not bind a loopback socket. Both complete suites passed when rerun with
local socket access; original failures and successful reruns are retained as
separate evidence. No external artifact replay or live installation was run.
The CLI reads local workstation observations; eligible-host and successful
approval tests use explicitly synthetic fixtures. Developer checks do not replace
AGY acceptance or the separate M1 gate review.
