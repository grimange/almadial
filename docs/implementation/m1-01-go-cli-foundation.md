# M1-01 — Go CLI foundation

ALM-9 establishes an executable CLI and controlled build, without installing or
inspecting the telephony stack. Implementation evidence is developer verification;
independent AGY QA remains required. No production readiness is claimed.

## Inputs and boundaries

Repository: `https://github.com/grimange/almadial.git`; module:
`github.com/grimange/almadial`. Implementation starts at
`3f331301882c3a72a14886f3617f964265acfb7e`, the accepted M0 commit.
Kaneo ALM-8 was read back as Accepted before ALM-9 entered In Progress.
The operator's ALM-9 assignment explicitly authorizes this foundation while
retaining the existing candidate manifest and its separate promotion process.
Historical plan text describing an unaccepted M0 is not a new compatibility test.
This task does not change stack versions, promote a manifest or access either VM.

Inputs are command arguments and build metadata. The executable owns no host
files, database, service, journal or configuration. It has no privileged
prerequisites and performs no network operations. Repeated execution simply
repeats output; there is no persistent state or recovery transaction. Failed
output returns an error; rerun after correcting the output destination.
Future effectful commands require their own authorized stage contracts.

## Compiler and provenance

Use **Go 1.27.3**, pinned in `.go-version` and the `go.mod` directive. There is no
toolchain directive: `GOTOOLCHAIN=local` prevents automatic compiler downloads.
There are no external Go modules; `go.sum` is therefore unnecessary. Runtime
uses the Go standard library only. Building also requires Python 3, Git (unless
commit metadata is explicitly supplied) and Make for the convenience targets.
The race test requires a supported native compiler, such as GCC, and cgo.

Official download metadata: <https://go.dev/dl/?mode=json>.
Verified Linux amd64 compiler archive:

- URL: `https://go.dev/dl/go1.27.3.linux-amd64.tar.gz`
- SHA-256: `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`
- Identity after verification: `go version go1.27.3 linux/amd64`

For an isolated compiler installation, download the archive to a fresh directory,
verify the digest **before** extracting, and add its `go/bin` directory to PATH.
Example (run in a writable scratch directory, not as root):

```bash
mkdir go-toolchain
curl --fail --location --proto '=https' --tlsv1.2 \
  https://go.dev/dl/go1.27.3.linux-amd64.tar.gz \
  --output go-toolchain/go.tar.gz
printf '%s  %s\n' \
  65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40 \
  go-toolchain/go.tar.gz | sha256sum --check
# Stop if verification fails.
tar -xzf go-toolchain/go.tar.gz -C go-toolchain
export PATH="$PWD/go-toolchain/go/bin:$PATH"
go version
```

Go upstream license and bundled notices remain with the compiler distribution.
No third-party code is copied into the AlmaDial source. The build helper does
not download or install a compiler; a missing/wrong compiler stops the build.

## Source organization

- `cmd/almadial/main.go`: forwards argv and streams, then exits with the result.
- `internal/cli`: testable routing, output and minimal exit semantics.
- `internal/buildinfo`: static version/commit plus compiled target and Go version.
- `scripts/build_go.py`: explicit argv build with controlled environment/metadata.
- `Makefile`: developer checks and Linux artifact build.

There is no CLI framework, background process, configuration parser or installer
implementation in this milestone.

## Commands and process contract

```bash
make build
./dist/almadial --help
./dist/almadial help
./dist/almadial --version
./dist/almadial version
```

No arguments, `help`, `--help` and `-h` print help and return 0. `version` and
`--version` print version/build information and return 0. Successful output goes
to stdout, with empty stderr. Help states that installation and host diagnostics
are not implemented. Unknown commands/flags, extra arguments and empty command
arguments return 2, with a short stderr message and empty stdout. Argument values
are not echoed, avoiding accidental secret/control-character disclosure.
Unimplemented `install`, `check`, `verify`, `plan` and other workflows are rejected.

Exit 1 means an output/application failure (currently a failed stdout write).
Exit 2 means usage/input validation failure. There are no separate host,
installation or postcondition exit codes yet; later stages will implement the
relevant proposed plan contract. No custom signal handler is needed for these
short, stateless operations; interruption follows normal OS process semantics.
The CLI neither intercepts nor reports successful recovery from interruption.

Default direct `go run ./cmd/almadial version` reports `dev` and commit `unknown`.
`make build` injects the current Git commit, adding `-dirty` when Git reports any
tracked or untracked changes. Unrelated untracked files therefore also mark a
build dirty. It reports, for example:

```text
almadial dev
commit: 67da129838f4fc7d16498be4b5b49111dd268ca6-dirty
target: linux/amd64
go: go1.27.3
```

Set `ALMADIAL_VERSION` and `ALMADIAL_COMMIT` explicitly for controlled release or
source-archive builds. Both permit 1–128 ASCII letters/digits/dot/underscore/plus/
hyphen, beginning with a letter or digit. They are identifiers, never secrets.
A source archive without `.git` requires explicit commit metadata. The caller is
responsible for matching supplied metadata to its source. The executable reports
provenance identifiers; calculate its SHA-256 externally rather than attempting
to embed its own hash. No manifest schema/support claim is made by this CLI.

## Controlled Linux amd64 build

```bash
make build
# Optional explicit metadata/output:
ALMADIAL_VERSION=dev ALMADIAL_COMMIT="$(git rev-parse HEAD)-dirty" \
  python3 scripts/build_go.py --output dist/almadial
sha256sum dist/almadial
file dist/almadial
```

The build helper checks the compiler version before creating output. It builds
`linux/amd64`, baseline `GOAMD64=v1`, with `CGO_ENABLED=0`, `-mod=readonly`,
`-trimpath` and `-buildvcs=false`. It clears ambient GOFLAGS/GOEXPERIMENT and
ignores Go environment/workspace files. Dependency proxies and sumdb are off;
no external dependency resolution is needed. Commit and version are injected
using linker `-X` arguments. No build timestamp is embedded, so wall clock time
and SOURCE_DATE_EPOCH do not affect the artifact. No username, hostname or
checkout path is injected. Compiler/Git probes have 30-second deadlines; the build has a 300-second deadline.
The compiler may populate its normal build cache;
use a writable `GOCACHE`/`GOPATH` for an isolated environment.

Two builds must use identical sources, compiler, target and metadata. A checkout
with different source contents is not promised to produce the same binary.
To test both path and cache independence from the repository root:

```bash
scratch=$(mktemp -d)
for run in a b; do
  mkdir -p "$scratch/$run/scripts"
  cp go.mod .go-version "$scratch/$run/"
  cp -R cmd internal "$scratch/$run/"
  cp scripts/build_go.py "$scratch/$run/scripts/"
  ALMADIAL_VERSION=dev ALMADIAL_COMMIT="$(git rev-parse HEAD)-dirty" \
    GOCACHE="$scratch/cache-$run" \
    python3 "$scratch/$run/scripts/build_go.py" --output "$scratch/$run/almadial"
done
cmp "$scratch/a/almadial" "$scratch/b/almadial"
sha256sum "$scratch/a/almadial" "$scratch/b/almadial"
"$scratch/a/almadial" --version
```

A digest mismatch is a failure to investigate, not an accepted variation.
Generated `dist/` and `build/` artifacts are ignored by Git. `make clean` removes
only the default `dist/almadial`; caller-selected outputs remain caller-owned.

## Verification and CI

```bash
export GOTOOLCHAIN=local GOENV=off GOWORK=off GOPROXY=off GOSUMDB=off
go fmt ./...
make fmt-check module-check
go test ./...
go vet ./...
go test -race ./...
make build
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

`fmt-check` detects formatting changes without rewriting files. `module-check`
uses `go mod tidy -diff` and `go mod verify`; no module downloads should occur.
Unit tests cover routing/streams, aliases, invalid arguments, unsupported
operations, output failure and stable metadata. Actual executable smoke checks
and isolated duplicate builds supplement unit tests.

The existing GitHub Actions workflow has a minimal Go job pinned through
`.go-version`: formatting, module consistency, tests, vet, race, build, byte
comparison and help/version smoke checks. This job needs no deployment secrets.
Local execution of its commands is implementation evidence; a hosted GitHub run
is NOT_RUN until the operator publishes the changes. Independent AGY review
must reproduce the checks and determine acceptance. ALM-10 remains unstarted.

## Implementation verification — 2026-11-07

Starting HEAD: `3f331301882c3a72a14886f3617f964265acfb7e`.
Current HEAD: `67da129838f4fc7d16498be4b5b49111dd268ca6`, an unrelated
operator recording-finalization commit preserved during implementation. It does
not overlap the ALM-9 files. Task changes are uncommitted; the artifact reports
the current HEAD with `-dirty`.
The normal build and two builds from separate source directories with independent
empty Go caches produced the same statically linked ELF Linux amd64 executable:

```text
69f6c4635d0862b761faf7cff44276fe5368e751e9c6489806a3a8f905b91ef0
```

PASS: formatting, module tidy/verify, vet, unit tests and native race tests.
Go tests contain three top-level tests, including 13 command-table subtests
(16 passing test events when counting their parent). Thirteen actual executable
smoke cases verify output/exit behavior. Malformed linker metadata is rejected
with exit 2 and no output artifact. Compiled source paths do not contain the
workstation checkout or temporary build/cache directories.

PASS: scaffold validation; all 126 existing Python tests; candidate-stack and
configuration-generation verification; build-helper syntax; diff whitespace.
No accepted M0 source or manifest was modified. CI job commands passed locally;
a hosted run and independent acceptance remain NOT_RUN. The CLI implements only
help/version; stack inspection, configuration and installation remain future work.
Protected execution logs, source identities, smoke outputs and build comparison
are collected under the task-specific `~/almadial-lab/evidence/m1-01/` run. The
Kaneo evidence comment records its exact path and sealed ledger digest.
