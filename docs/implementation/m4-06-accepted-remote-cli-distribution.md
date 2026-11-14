# Accepted remote CLI distribution preparation

## Result and boundary

The authorized private bootstrap now installs the independently accepted
`alm46-native-rc1` CLI. A separate pristine AlmaLinux 10.2 target downloaded the
script and binary through trusted HTTPS, verified the installed executable,
acquired all five pinned source objects and generated a remote-profile plan.
The developer deliberately declined approval before provisioning. This is a
distribution/integration smoke test, not independent human operator acceptance.

The unchanged CLI's ordinary native agent workflow, WebRTC audio, recording,
reboot persistence and healthy NO_OP are covered by the accepted
[ALM-46 independent QA](../reviews/alm46-native-runtime-independent-qa.md).
Those campaigns were not repeated and ALM-46 was not reopened.

## Immutable candidate identity

| Item | Verified identity |
|---|---|
| Implementation | `92b14eb4452b7bfb949e0ee7efc571a83d9e799c` |
| Version | `alm46-native-rc1` |
| Binary SHA-256 | `a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3` |
| Build | Go 1.27.3, Linux amd64/v1, CGO disabled, trimpath |
| Independent QA documentation | `85fb7abaad247398fb40305cd3eb1c51e6cf6bc1` |
| Accepted QA comment | `xl7enlkb6mwb6slvmzuf290s` |

The retained approved binary was hashed and inspected using `--version` and
`go version -m`. Two separate clean builds from an archive of the exact accepted
commit reproduced its SHA-256 byte for byte. No dirty-tree binary was substituted.

## Bootstrap and private service

The existing bootstrap template, generator and HTTPS server are reused. The
repository's portable rendered script pins the accepted version/digest and has
no implicit binary endpoint. The private rendered script supplies its reviewed
default URL; `ALMADIAL_BINARY_URL` can relocate the same immutable bytes.
TLS verification, checksum-before-execution, root ownership, mode 0755 and
atomic replacement of an existing executable remain intact.

The generator now optionally checks `--expected-sha256` before executing the
input binary, verifies its version and exact source identity, and distinguishes
binary `source_commit` from `packaging_commit` and packaging-tree hashes.
Self-reported metadata is supplemented by the independent accepted checksum and
clean rebuild evidence; it is not an authentication mechanism by itself.

The previous dedicated artifact VM was unreachable from the workstation and the
accepted development network after a graceful restart. A new dedicated
AlmaLinux VM serves preserved approved objects through the existing private
HTTPS server. It has its own disk, no checkout mount, an enabled systemd service,
SELinux Enforcing and a listener bound to its private address. This is controlled
private-network distribution; no public exposure or hardened public artifact
service is claimed. Firewalld is absent on this distribution image; no firewall
qualification claim is made for that auxiliary VM.

The [current distribution record](../development/private-distribution-lab.md#current-accepted-release-candidate)
contains the actual working URL, service identity, public CA, certificate
validity, operator commands and recovery instructions. Both server and target
have recorded pristine snapshots. Keep the distribution VM running for the
human run; its address and certificate validity require owner maintenance.

| Delivery object | SHA-256 |
|---|---|
| Served `install.sh` | `622a530700763d97923b605bacbb8cd25b8c0594c91d096d2e0e099d4c5e38f4` |
| Served `release.json` | `4c47e85a8043a069c75598eed40c0d03341eac663c3439979ca72af2df59ab3a` |

The GitHub main reference was read back as
`f885482478e2fa13fa95a1211c0ab688363b617d`. Its immutable raw bootstrap downloaded
successfully and matched its committed bytes (SHA-256
`8e3149cadb2cf8512a1dd87a2ea79c296d7ae59c665c49f84327bf4c056c22a0`),
but it pins `m4-06-acquisition-dev`. It must not be used for this candidate.
This work does not push, tag, merge or publish the corrected script to GitHub.

## Pinned source availability

The CLI accepts an explicit approved source-store base through
`ALMADIAL_ARTIFACT_STORE` or its guided prompt, independently of the binary URL.
The actual target saved a manifest-bound origin policy and retrieved every
content-addressed object from the dedicated service. Source identities and
verification policy were unchanged.

| Source | Identity | SHA-256 | Target acquisition |
|---|---|---|---|
| Asterisk | 22.12.0 | `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` | PASS |
| PJPROJECT | 2.17 | `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` | PASS |
| Net::Telnet | 3.05 | `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` | PASS |
| VICIdial | 2.14b0.5, r4037 | `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` | PASS |
| VICIphone | `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` | `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` | PASS |

Server access logs corroborate target downloads. Cached bytes were independently
hashed after the guided acquisition. The approved private store retains licenses
and notices; ALM-32's unresolved public redistribution boundary is unchanged.
No third-party archive, private key or protected evidence is committed.

## Pristine-target smoke and focused regression

`almadial-alm34-release-dev` started from a recorded pristine snapshot with no Go
or Git installed and no mounted repository. In a root login shell, bootstrap
installation produced `/usr/local/sbin/almadial`, root:root, mode 0755, with the
exact accepted version and SHA-256. `install --help` worked. A wrong-byte binary
origin was rejected on the actual target and preserved the installed executable.

The interactive remote wizard accepted explicit FQDN/interface/IPv4, supplied
application TLS and source-store inputs, verified all five archives and produced
plan SHA-256 `a8b5962e44b32aeb86d40446de12e21493d95da51a312ea557687745d3eabf09`.
Approval was deliberately declined; exit 1 was expected. No provisioning journal
or approved plan was applied. Go/Git remained absent. Full fresh provisioning,
media and NO_OP were **NOT_RUN in this packaging task**; their accepted evidence
is reused because binary bytes and runtime implementation are unchanged.

Two harness mistakes are retained transparently: the first non-login shell did
not include `/usr/local/sbin` in PATH; the corrected documented `sudo -i` path
passed. An initial cache-inspection command assumed the distribution layout
inside the local cache; the corrected cache path verified every object. Neither
required an installer change.

Executed checks passed:

- `go vet ./...`, `go test ./...`, `go test -race ./...`.
- `go mod verify`, `go mod tidy -diff`.
- Scaffold validation and Python unittest discovery: 256 tests, including 12
  bootstrap tests covering correct-version delivery, approved alternate origins,
  wrong checksum, untrusted TLS, missing origin, unsupported OS/architecture,
  existing-executable preservation and release-generator identity rejection.
- Bash syntax for the template and rendered bootstrap; Git whitespace checks.
- Documentation links, generic README address check and Go formatting.
- Two reproducible builds from the accepted source revision.

## Handoff and limitations

Follow the [operator guide](../operations/almadial-installation.md) and the current
private procedure linked above. The owner-held operator kit contains the reviewed
public CA, release contract and instructions, without private keys or archives.
The owner must recover the protected operator VM and supply application DNS/TLS
inputs before the independent human run. Codex only inspected that VM; it remains
Deleted with its retained pristine snapshot.

The exact accepted CLI has one stale pre-QA runtime-warning banner. Accepted QA
supersedes that text; real Status/Doctor failures remain actionable. Public
VPS/DNS/CA topology, physical headset perception, remote archived-recording HTTP
playback, carrier/PSTN and capacity remain outside the qualified scope. Loopback
recording URLs remain a separate product follow-up. No implementation or task
acceptance is inferred from distribution preparation.

Protected evidence:
`~/almadial-lab/evidence/m4-06/accepted-remote-cli-20261114T040454Z/`.
Its qualification SHA-256 ledger covers the recorded delivery, source inventory,
rebuilds, target smoke and regression results. The subsequent tracker read-back
has a separate handoff ledger. ALM-34 remains In Progress under Codex for the
independent human installation and subsequent AGY assessment; ALM-35 is unchanged.
