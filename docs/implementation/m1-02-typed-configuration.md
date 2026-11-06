# M1-02 — Typed configuration and safe secret references

ALM-10 implements `internal/config` using Go 1.27.3 and the standard library.
The module remains `github.com/grimange/almadial`. The starting commit is
`cce3d155f31bfcdb5cfc2b1977f6ac5f21320150`, independently Accepted ALM-9.
This is an installation-input contract, with developer verification pending
independent AGY acceptance. It performs no installation or host inspection.

## Decision and boundaries

The operator's ALM-10 assignment explicitly permits an IP-only installation,
optional hostname and optional WebRTC. This supersedes the older proposed plan
§4.2 mandatory-FQDN example for this task. JSON replaces the plan's proposed YAML
example: `encoding/json` requires no new dependency. No stack versions, manifest
state, VM, accepted CLI command or process exit convention change.

The package owns no host resources and needs no privileges. Its only input read
is the reader explicitly passed by its caller. It never reads environment
variables, certificate/key/secret files, DNS, network interfaces or service state.
It starts no subprocesses and writes no host configuration. Invalid input returns
an error and a zero `Config` through `Load`; the caller must stop before any future
mutation. Retrying repeats pure evaluation; there is no recovery transaction.

## API and format

- `Decode(io.Reader) (Config, error)` checks syntax, shape and typed values.
- `Defaults(Config) Config` returns an independent normalized copy.
- `Validate(Config) error` checks effective values, in stable field order.
- `Load(io.Reader) (Config, error)` performs those three operations in order.
- `Digest(Config) (string, error)` validates normalized input and hashes its
  deterministic JSON with SHA-256.
- `ValidateUnattended(Config, Approval) error` checks explicit approval against
  that complete effective-input digest.

Only schema version **1** is supported. Version is required, never defaulted.
Only one JSON object, at most 1 MiB and valid UTF-8, is accepted. Unknown keys,
case aliases, duplicate keys, `null`, arrays in object fields, wrong scalar types,
fractional/exponent integer notation, integer overflow and trailing data/documents
are rejected. Optional values must be omitted instead of set to `null`.
No format autodetection, migration engine or generic configuration framework exists.
Decoding does not apply defaults or semantic validation: use `Load` normally.

`Config` contains typed `Network`, `SIP`, `RTP`, `WebRTC` and `Credentials`
structures. Addresses use `netip.Addr`; secret kinds use `ReferenceType`.
Pointer integer ports distinguish omission from explicitly invalid zero.

| Field | Requirement and default |
| --- | --- |
| `schema_version` | Required integer `1` |
| `network.bind_address` | Required IPv4/IPv6 address; never discovered |
| `network.advertise_address` | Defaults to a concrete bind IP; required explicitly with wildcard bind |
| `network.hostname` | Optional DNS hostname; omitted means use advertised IP as certificate identity |
| `sip.port` | UDP SIP port; default `5060` |
| `rtp.start`, `rtp.end` | Inclusive UDP media range; defaults `10000`, `20000` |
| `webrtc.enabled` | Defaults `false` |
| `webrtc.wss_port` | Defaults `8089` only when WebRTC is enabled |
| `webrtc.tls.certificate_file`, `private_key_file` | Required separate absolute file references when enabled |
| `credentials.database` | Required database credential reference |
| `credentials.ami` | Required local AMI credential reference |
| `credentials.sip` | Required generated phone credential reference |
| `credentials.admin` | Required initial VICIdial administrator credential reference |

These four credential purposes describe installer inputs, not a grant/account
implementation or an instruction to share passwords between services. Resolving,
creating and delivering them remains a later stage's responsibility.

## Network, ports and conditional requirements

Private IPv4/IPv6 unicast addresses are supported. Unspecified bind addresses
(`0.0.0.0`, `::`) require explicit unicast advertised addresses of the same family.
Concrete bind and advertised addresses may differ; NAT settings and connectivity
are not inferred or certified. Loopback, multicast, broadcast, link-local, scoped
IPv6 and IPv4-mapped IPv6 addresses are rejected for this single-server identity.
CIDR strings are rejected. A bare address ending in `.0` cannot be classified as a
subnet address without its prefix; host-subnet membership belongs to preflight.

Hostnames are ASCII DNS labels (1–63 characters each, total at most 253), with
letters/digits and interior hyphens. Single labels are accepted. Case is normalized
to lowercase and one terminal DNS dot is removed. URL schemes, whitespace,
underscores, empty labels and IP literals in the hostname field are rejected.
No DNS resolution occurs. International names must be supplied in ASCII form.

All ports must be integers in `1..65535`. The RTP start must be even and the end
must exceed it, providing room for at least one even RTP/odd RTCP pair. The
inclusive end may be even (as in the accepted Asterisk range); this is an allowed
range, not a promise every port is allocated to a pair. The SIP UDP listener must
not overlap that range. As a simple schema policy, SIP and WSS listener numbers
must differ, although their transports could technically share a number. WSS TCP
may overlap UDP media numbers. No listener availability is checked.

WebRTC remains disabled unless explicitly enabled. When disabled, omitted TLS
and WSS fields are valid; supplied WSS/TLS fields are rejected to catch forgotten
feature enablement. When enabled, an IP certificate is acceptable without DNS.
The future TLS verifier must match the supplied hostname, if present, otherwise
the advertised IP, against certificate SANs. Configuration validation does not
establish certificate trust, SAN coverage, key match, validity or readability.
SIP here is UDP; this model does not claim SIP-TLS transport configuration.

## TLS and secrets

TLS values and `file` secret sources must be clean absolute Linux file paths,
other than `/`, with no NUL/control characters, backslashes, `..` normalization,
repeated separators, trailing separator or surrounding whitespace. Inline PEM
markers are rejected. Spaces inside valid Linux paths are permitted. References
may point to files that do not yet exist; validation never opens them.

A `SecretRef` has exactly `type` and `source`:

```json
{"type": "env", "source": "ALMADIAL_DB_PASSWORD"}
```

```json
{"type": "file", "source": "/run/almadial-secrets/database"}
```

`env` accepts `[A-Za-z_][A-Za-z0-9_]*`, identifying a variable name, without `$`,
expansion syntax or assignment. `file` identifies a local file, not its contents.
Unknown reference kinds fail. Raw `password`, `token`, `private_key` and similar
value fields are unknown and fail. The model cannot prove that an identifier
belongs to a secure source; it never resolves that source. Later execution must
check permissions, ownership and credential completeness, redact secret values
and avoid secret-bearing argv. No retrieval, rotation or storage is implemented.
JSON serialization contains references only, never resolved values.

All package-produced diagnostics contain a known schema field and a static
reason, never supplied values. Even unknown field names, underlying reader errors
and JSON parser excerpts are omitted because they can contain secrets. Semantic
validation returns the first error in fixed field order; decoding reports the
first malformed field in document order. `*FieldError` exposes `Field` and
`Reason`; callers can use `errors.As`. Example: `sip.port: must be an integer from
1 through 65535`. This package assigns no process exit codes.

## Examples and invalid input

The following files are executable fake-only fixtures. Documentation addresses
and `.invalid` names are not an operating deployment.

Basic IP-only configuration, with WebRTC disabled by default:

```json
{
  "schema_version": 1,
  "network": {"bind_address": "192.0.2.10"},
  "credentials": {
    "database": {"type": "env", "source": "ALMADIAL_DB_PASSWORD"},
    "ami": {"type": "env", "source": "ALMADIAL_AMI_PASSWORD"},
    "sip": {"type": "env", "source": "ALMADIAL_SIP_PASSWORD"},
    "admin": {"type": "env", "source": "ALMADIAL_ADMIN_PASSWORD"}
  }
}
```

WebRTC with optional hostname and TLS/secret references:

```json
{
  "schema_version": 1,
  "network": {"bind_address": "2001:db8::10", "hostname": "dial.example.invalid"},
  "webrtc": {
    "enabled": true,
    "tls": {
      "certificate_file": "/etc/almadial/tls/fullchain.pem",
      "private_key_file": "/etc/almadial/tls/private.key"
    }
  },
  "credentials": {
    "database": {"type": "file", "source": "/run/almadial-secrets/database"},
    "ami": {"type": "env", "source": "ALMADIAL_AMI_PASSWORD"},
    "sip": {"type": "env", "source": "ALMADIAL_SIP_PASSWORD"},
    "admin": {"type": "env", "source": "ALMADIAL_ADMIN_PASSWORD"}
  }
}
```

`internal/config/testdata/webrtc.json` also makes the default ports explicit.
An invalid inline-secret example is
`{"schema_version":1,"password":"FAKE_INLINE_SECRET_DO_NOT_USE"}`; the error
omits both the unknown key name and its value. `invalid-inline.json` and
`invalid-unattended.json` are rejected fixtures. Unsupported versions, missing
credential sources, zero ports, ambiguous wildcard binds and incomplete enabled
TLS settings also fail before a validated configuration is returned.

## CLI and unattended decision record

There is no interactive wizard or config CLI command in ALM-10; ALM-9's help,
version, stdout/stderr and exit contracts remain unchanged. A future wizard
constructs the same typed `Config` as a file decoder and applies the same defaults
and validation. Tests compare directly constructed and file-derived effective
input digests; this is API equivalence, not a claim of a tested interactive UI.

Unattended execution must not infer approval from the existence of a config
file. After validation and presentation, the future caller obtains explicit
operator approval and supplies `Approval{Approved: true, ConfigSHA256: digest}`.
`ValidateUnattended` rejects missing approval, incomplete inputs and any changed
effective input. Approval is separate from configuration; an `approval` JSON key
is unknown. The digest is not authentication, a signature, a host readiness check
or an installation plan approval. The future execution layer must control the
approval channel, preserve the validated inputs and bind any effectful plan to
its own checks. Neither ALM-10 nor a caller-supplied boolean authorizes a host
mutation. There is no hidden discovery or credential generation to fill missing
non-discoverable inputs. Ports have the explicit, documented defaults above.

## Verification and extension boundary

From a checkout with Go 1.27.3 in PATH:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod verify
go test -v ./internal/config
make fmt-check
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Use ALM-9's `scripts/build_go.py` with fixed `ALMADIAL_VERSION` and
`ALMADIAL_COMMIT` for two Linux amd64 builds in separate source directories,
then compare SHA-256 and help/version/error process outputs. The CLI does not yet
import configuration, so unchanged executable bytes are expected; the new package
is compiled and exercised by its own tests. No hosted CI execution is claimed.
The existing CI Go job automatically tests and formats all `internal` packages;
no new CI provider or workflow is required.

Future host preflight must check address/interface/subnet selection, DNS if used,
port availability, real TLS material and service access, secure secret-source
permissions and actual value availability. Installer stages must separately define
plans, privileged execution, journal/recovery and postconditions. Deployment
network qualifications remain deployment-specific. None is simulated as a PASS
by these pure configuration tests.

## Implementation verification record

At the accepted ALM-9 HEAD, the ALM-10 developer run passed:

- Go 1.27.3: formatting, vet, tests, race detector, module verification and
  `go mod tidy -diff`; all exit 0. Tests: 13 top-level tests plus 115 subtests
  across all packages, including 10 top-level configuration tests and 102
  configuration subtests. Configuration statement coverage: 96.8%.
- All 126 existing Python tests, scaffold validation, candidate-stack and
  generated-configuration checks; all exit 0. The offline verifiers' live-runtime
  cases remain NOT_RUN; no M0 VM was accessed.
- Separate source-directory Linux amd64 builds with controlled `dev` version and
  commit `cce3d155f31bfcdb5cfc2b1977f6ac5f21320150-dirty` produced identical
  SHA-256 `76cbf7337fbb2aee91dfe64bfb2a89aea80f6378091eb3c66e6d7bc781209763`.
  An identically built accepted ALM-9 CLI had the same bytes. No arguments,
  help/version aliases, unknown command, invalid flag and extra-argument smoke
  cases matched the accepted CLI, including streams and exit codes 0/2.

No accepted tracked file was modified. New task-owned files are this guide and
`internal/config/` (implementation, tests and four fake-only JSON fixtures).
Concurrent unrelated recording/QA documentation remained outside this task.
No commit, push, release, stack-lock promotion or independent acceptance occurred.
Evidence is retained separately under `~/almadial-lab/evidence/m1-02/`; Kaneo's
implementation comment records the execution directory and ledger identity.
AGY must independently verify the API, strict parser, safe defaults, secret
isolation, unattended binding and unchanged CLI/build contract before acceptance.
