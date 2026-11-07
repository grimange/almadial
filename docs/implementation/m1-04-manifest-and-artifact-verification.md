# M1-04 — Stack manifest and source artifact verification

ALM-12 adds `internal/artifacts` to the accepted Go 1.27.3 foundation. It reads
and verifies sources; it does not install, execute, extract, build, or configure
those sources. Acquisition is explicit and stays outside preflight and planning.
The checked-in stack lock remains **candidate**. Neither successful verification
nor a plan digest authorizes installation.

## Manifest contract and dependency

`Load(io.Reader) (Manifest, error)` accepts the existing
`manifests/stack.lock.yaml`, schema 1. No alternative manifest is introduced.
The private typed representation preserves OS/image, source versions and hashes,
VICIdial revision/schema/tree identity, embedded SIP.js identity, signing
fingerprint, exact RPM NEVRA/repository records, and verification state.

Go's standard library does not parse YAML. The sole runtime dependency is
`go.yaml.in/yaml/v3` **v3.0.4**, the official YAML organization's maintained v3
line ([upstream](https://github.com/yaml/go-yaml)). Its module and archive
checksums are pinned in `go.sum`. This avoids an incomplete custom YAML parser.
The parser's mixed MIT/Apache-2.0 LICENSE and NOTICE are preserved under
`third_party/go-yaml/`; preserve them when distributing code that links it.
`gopkg.in/check.v1` is a checksum-pinned, BSD-2-Clause dependency of upstream's
own tests, not an AlmaDial runtime dependency.

A fresh checkout first acquires pinned modules through the authenticated Go
proxy and checksum database:

```sh
GOTOOLCHAIN=local GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org go mod download
```

Subsequent build/verification is offline with populated module cache. CI adds
this one explicit acquisition step before the existing offline checks. Hosted
CI execution is separate from local verification.

The loader limits input to 64 KiB and rejects multiple documents, malformed YAML,
missing fields, duplicate/ambiguous keys, unknown fields, aliases, anchors, merge
keys, custom tags, unsupported structures/types, and identities that differ from
the independently compiled reviewed M0 policy in `policy.go`. The original
unquoted numeric versions remain supported; equivalent quoted versions are
allowed only for those fields. A version, origin, digest, revision or repository
change requires explicit review of both policy and manifest. Supplying a changed
URL and changed digest together cannot establish a new trusted identity.
Errors identify reviewed field paths where safe and never echo input values or
unrecognized key names.

`Manifest.Digest()` is SHA-256 of the **exact YAML bytes**, including comments and
whitespace. It is suitable for ALM-11's existing supplied manifest SHA-256 input.
It is not a semantic-normalization hash. `Packages()` returns a defensive copy
of the six RPM records, retaining repository origin without asserting that a
particular RPM signature or transaction has been checked.

## Trust and authorization

Each `Result` records component/version/source, expected and actual SHA-256,
size, file count/tree digest where applicable, provenance, limitations, and
separate `Identity`, `Transport`, `Publisher`, and aggregate `Disposition`:

| Property | Meaning of VERIFIED |
|---|---|
| Identity | Full pinned archive hash and required archive/tree/embedded asset checks passed |
| Transport | This invocation retrieved bytes from the approved HTTPS URL with validated certificates |
| Publisher | Applicable evidence passed: approved official HTTPS origin plus pinned content, or independently pinned Asterisk detached signature |
| Disposition | Identity and the applicable publisher evidence passed; this is not installation permission |

Categories are `VERIFIED`, `REJECTED`, and `NOT_VERIFIED`. A local checksum match
leaves acquisition/transport and publisher evidence `NOT_VERIFIED`. Recorded M0
provenance is described as historical evidence rather than fabricated current
HTTPS verification. Cache reuse likewise does not invent a new transport event.
Signature verification of a local Asterisk archive can establish publisher
evidence without asserting a fresh HTTPS acquisition.

`InstallationAuthorized()` and `Result.InstallationAuthorized` remain false:
the actual policy contains candidate/TBD markers. Candidate artifacts can be
inspected, but cannot be used as installation authorization. A supplied
`accepted` status or modified provenance/verification marker fails policy
validation. No executor is exposed. Future promotion requires independent QA
and maintainer authorization; it is not a parser option.

A `Result` is diagnostic evidence, not a cryptographic capability. Editing its
exported fields does not establish trust or consent. A future executor must
independently enforce verified policy, input stability, signature requirements,
actual authorization, and freshness.

## Supported artifact identities

Only `Manifest.Artifact(component)` constructs a usable `Spec`; expected hashes
and approved origins are private fields. Unknown components and zero-value specs
fail. The five supported source archives are:

| Component | Selected version | SHA-256 |
|---|---|---|
| Asterisk | 22.12.0 | `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` |
| VICIdial | r4037 / schema 1748 | `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` |
| PJPROJECT | bundled 2.17 | From `pjproject.sha256` in the reviewed lock |
| VICIphone | 3.0 / `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` | From `webrtc.sha256` in the reviewed lock |
| Net::Telnet | 3.05 | From `perl_net_telnet.sha256` in the reviewed lock |

Approved locations come from the reviewed lock. VICIphone uses the M0-recorded
GitHub codeload URL for that exact commit, not its current branch. Official
VICIdial nightly acquisition remains distinct from a signed SVN revision. Its
mutable URL is acceptable only when the bytes still match the exact preserved
archive and canonical tree; use the immutable local M0 copy if upstream changed.
No VICIdial signature is invented.

The other selected versions remain AlmaLinux 10.2, MariaDB 10.11.18, Apache
2.4.63, PHP 8.3.36 and Perl 5.40.2. Native signed RPM validation/installation is a
future package-stage responsibility. ALM-12 preserves those exact NEVRA/origin
records and does not replace DNF/RPM or CPAN.

`VerifyDerivations(ctx, repository)` checks the six compiled SHA-256 pins of the
reviewed M0 SQL, generator, FastAGI/runtime, VICIphone and replay helpers. It reads
files through a confined repository root, never runs those scripts, and rejects
changed/missing/nonregular/oversized files. Existing Python tests separately
check upstream source and derived-output semantics. There is no arbitrary patch
input or unchecked patch execution path.

## Local verification and errors

The internal API can be used by later Go commands. No new CLI command is claimed
or added. From a package inside this module:

```go
f, err := os.Open("manifests/stack.lock.yaml")
if err != nil { return err }
m, err := artifacts.Load(f)
f.Close()
if err != nil { return err }
spec, err := m.Artifact("vicidial")
if err != nil { return err }
result, err := artifacts.VerifyLocal(ctx, spec, "/private/cache/vicidial-r4037.zip")
if err != nil { return err } // Corrupt bytes return SHA-256 mismatch; never proceed.
// result.Identity == VERIFIED is not publisher authentication or approval.
```

`VerifyLocal` streams SHA-256, rejects special files/symlinks and reads at most
512 MiB compressed input. It never accesses the network. Missing, canceled,
corrupt, substituted, oversized or unsafe inputs return an error. Every cache
hit rehashes and reinspects; a prior successful run is never sufficient.
Do not ignore returned errors even if a partially populated result contains
successful checks. Diagnostics omit private input paths, URLs from failures,
HTTP bodies and GPG stderr.

## Archive inspection

No extraction is implemented. Read-only tar/gzip, tar/bzip2 and ordinary
single-disk ZIP inspection enforce:

- At most 512 MiB compressed, 2 GiB expanded, 256 MiB per regular file and
  30,000 entries; ZIP central directory is bounded to 16 MiB before allocation.
- No absolute, traversal, noncanonical, backslash, invalid UTF-8, NUL, CR/LF or
  colon-bearing paths; no duplicate paths, devices, hardlinks or sparse files.
- Generic symlinks are rejected. The pinned Asterisk archive's contained relative
  symlinks may reference only regular files under `asterisk-22.12.0/`; dangling
  links, chains, escapes and entries beneath links fail. Links are never followed
  on the filesystem. This exception permits inspection of the actual signed
  archive and is not authorization to extract links.
- Compression trailers and ZIP CRCs are checked by reading full contents.
  ZIP64 and multi-disk ZIP are intentionally unsupported.
- GitHub global PAX metadata is limited to a commit comment; VICIphone's comment
  must match its pinned commit. No metadata changes extraction behavior.

The VICIdial ZIP must have exactly **2,511 regular files** beneath `2026-11-05/`
and tree SHA-256
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
For each file, hash its contents, sort relative paths by **path components** as
accepted Python `sorted(pathlib.Path)` does, concatenate
`<lowercase-sha256>  <relative-posix-path>\n`, then hash that UTF-8 inventory.
Directories, modes and timestamps are excluded. A matching file count alone is
insufficient. This reproduces the accepted M0 rule, including names where sorting
whole path strings would differ.

VICIphone must have 48 regular source files beneath the pinned commit root and
embedded `src/js/sip.js` hash
`9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b`
(SIP.js 0.20.1). The complete archive pin binds all source assets. Existing
`m0_07_assets.py` tests retain the separate 46-deployed-asset/overlay contract;
ALM-12 does not redeploy or inspect a live VM's webroot.

## Explicit HTTPS acquisition

```go
spec, err := m.Artifact("viciphone")
if err != nil { return err }
result, err := artifacts.Fetch(ctx, spec, artifacts.FetchOptions{
    Directory: "/private/cache", // existing absolute directory, owned by caller, 0700
})
if err != nil { return err }
// Published only after digest and archive inspection; still not installable.
```

`Fetch` uses only the spec's compiled approved HTTPS location. No caller URL,
userinfo, query or fragment is accepted. OS certificate/hostname validation and
TLS >= 1.2 are mandatory; an explicit CA pool supports enterprise/test trust
without disabling verification. Proxy environment variables are not used. All
redirects are rejected, even same-origin ones; no silent mirror or HTTP fallback.
Timeouts are 15 seconds for dial/TLS, 20 seconds for response headers and two
minutes overall, with caller cancellation. Non-200, partial, oversized or
mismatched transfers fail.

The existing private staging root is confined with `os.OpenRoot`. A random
exclusive 0600 temporary file is bounded, synced and validated before atomic
no-overwrite publication as `<component>.artifact`. Incomplete temporaries are
removed. Existing private regular cache files are always reverified; corrupt
ones produce an error, never an overwrite or silent redownload. No host-wide
cache or directory is created by the API. Asterisk staging verifies identity
but remains pending detached signature verification.

## Asterisk detached signatures

`VerifyAsteriskSignature(ctx, spec, archive, signature, keyring, scratch)` verifies
both archive identity and a detached signature using fixed `/usr/bin/gpgv`, direct
argv, a 30-second deadline and bounded status output. It requires Linux descriptor
paths; no shell, keyserver, secret key, import, software execution or network
access is involved. The caller supplies an existing private scratch directory.
Public key and signature inputs are limited to 1 MiB. A missing `gpgv` fails.

The independent trust anchors are compiled policy, not values trusted solely
because they occur in the supplied manifest:

- Full signing fingerprint `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`.
- Preserved M0 armored public key SHA-256
  `2eb75c272ebd1f96742ef7b0b53ebe943c840c55addc22db2122a494df578145`.
- Dearmored public keyring SHA-256
  `645c8fb2b67133a59d9f7d1113e953c5dc79950fd0b28a6529a0efef0146727c`.

Prepare the public keyring outside the verifier from that preserved public key:

```sh
gpg --batch --yes --dearmor --output /private/cache/asterisk.gpg /path/to/asterisk-key.asc
sha256sum /private/cache/asterisk.gpg
```

Require the pinned keyring hash above. GPG's successful exit status and full
`VALIDSIG` fingerprint with SHA-2 are both required. Open descriptors bind hash
and signature checks to the same inputs; the archive is rehashed/reinspected
after signature verification. The M0 trust basis is the recorded maintainer-linked
public-key association, not an asserted OpenPGP Web of Trust. No detached
signature mechanism is invented for sources that do not publish one.

## Planning and future execution

Pass `m.Digest()` to ALM-11's manifest identity field only after strict loading.
Artifacts remain separately verified inputs; this task does not rewrite the
planner or download during plan generation. A future executor must enforce
approved exact stack policy, actual operator approval, fresh plan/config/host
binding, RPM/signature policy and stable private artifact storage. It must recheck
bytes at the point of use: a prior result cannot guarantee a subsequently changed
path. This verifier is not a general archive extractor or package manager.

## Verification

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod verify
go mod tidy -diff
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Tests use local fixtures and controlled HTTPS servers, not live upstream URLs.
They cover strict identities, unknown/duplicate keys, changed patches, checksum
and tree mismatch, unsafe archives/links, bounded ZIP directories, untrusted
certificates, hostname mismatch, redirects, oversized/partial/interrupted
transfers, canceled operations, corrupt cache reuse and safe cleanup. Default
unit tests skip the separately opt-in real M0 archive qualification.

To run that read-only qualification against preserved artifacts (no VM access):

```sh
ALMADIAL_M0_ARTIFACTS_DIR=/path/to/m0-08/inputs-final \
ALMADIAL_ASTERISK_KEYRING=/private/cache/asterisk.gpg \
ALMADIAL_ASTERISK_SIGNATURE=/path/to/asterisk-22.12.0.tar.gz.asc \
  go test ./internal/artifacts -run '^TestM0Artifacts$' -count=1 -v
```

The five expected filenames are `asterisk-22.12.0.tar.gz`, `vicidial-r4037.zip`,
`viciphone.tar.gz`, `pjproject-2.17.tar.bz2`, and `Net-Telnet-3.05.tar.gz`.
The opt-in test also checks the real Asterisk signature and rejects a corrupted
signature. Controlled reproducible builds use the accepted build helper with
fixed version/commit metadata from two separate source directories. Tests and
build evidence belong under protected `m1-04` evidence, not Git.

Independent AGY QA must reproduce critical checks and decide acceptance.
Hosted CI and fresh upstream downloads are not implied by local test success.
External topology tests, installer execution and manifest promotion remain outside
this task.
