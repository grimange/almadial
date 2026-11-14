# Private distribution laboratory record

This document records a specific development environment. It is not a general
AlmaDial network requirement or a public release announcement. The
[operator guide](../operations/almadial-installation.md) describes the portable
configuration contract; the [retention guide](../operations/almadial-private-artifact-retention.md)
describes source policy and release ownership.

## Later remote-profile development observation

The sections below are the historical portability-correction record, not the
current publication or runtime state. During the later remote-profile work,
GitHub main was observed at `f885482478e2fa13fa95a1211c0ab688363b617d`.
Its immutable bootstrap URL returned HTTP success and matched its Git blob.
That script still pins the earlier laboratory CLI. Guided installation and
durable acquisition were reconciled from the retained implementation checkpoint
into the current tree, and source-store selection is now explicit rather than
an embedded workstation dependency.

Two dedicated development VMs provided a separate controlled HTTPS artifact
origin and an off-host deployment target. Their concrete network/trust inputs
remain in protected evidence, not generic operator defaults. The latest CLI is
`alm46-remote-dev4`; it is a private development artifact, not a public release.
See the [remote implementation report](../implementation/remote-single-server-controlled-qualification.md)
for exact identities, successful synthetic media checks and the remaining normal
agent-runtime P0. These observations do not update the historical server or
complete independent operator acceptance.

## Repository and publication observation

At this correction, local `main` and GitHub `main` resolve to
`18776fa0c7aaed9633b8c7e4d84e38d65cff295d`. The guided installation and durable
acquisition implementation remains on the retained local branch
`alm34/github-bootstrap-operator-acceptance`, commit
`a9d58baac3bd748f4d10fc7abfc029e6d468bfbc`. That checkpoint also contains the
original bootstrap template, generator and fixtures reused in this correction.
Only the bootstrap delivery files are brought into the current source tree;
the Go installer and its journals are unchanged.

Live HTTPS requests for `scripts/bootstrap/install.sh` at both immutable GitHub
commits returned **HTTP 404**. No working public bootstrap URL is claimed. This
assignment does not push or deploy the corrected portable script. A local commit
does not update either GitHub or the existing private server.

## Existing qualified private candidate

- Bootstrap currently served: `https://10.160.90.1:8447/install.sh`.
- CLI URL: `https://10.160.90.1:8447/releases/m4-06-acquisition-dev/almadial-linux-amd64`.
- CLI SHA-256: `4b484239a3939e9f1f12eb6fda22a0f74a2e3d035023d3b01ec8e9e43aaf78f9`.
- CLI version: `m4-06-acquisition-dev`; build identity:
  `65b40fb79570c0767ed563d6e25be2c980c58985-alm34-dirty`, Go 1.27.3, Linux amd64.
- Original served script SHA-256:
  `1d550b97f30acd96373d206c713c90efd1deefde6172a3a996b07cdf194b6183`.
- Approved source base: `https://10.160.90.1:8447/sources`.
- Store root: `/home/ra/almadial-lab/distribution/alm34`;
  source objects are under `sources/sha256/` and CLI payloads under `releases/`.

The served original script still has a fixed private URL. The corrected script
in this source tree pins the **same CLI version and checksum**, has no default
origin, and uses explicit `ALMADIAL_BINARY_URL`. Do not claim that the server
serves that corrected script until its owner separately deploys and verifies it.
The corrected script can use the existing CLI URL unchanged.

The public artifact CA is held outside the distribution root at
`~/almadial-lab/evidence/m4-06/acquisition-20261113T193809Z-alm34/artifact-ca.pem`.
Its independently recorded file SHA-256 is
`6e49e14cb26240689aa4ce67d1ff61bc8eac1c5dddf7fbdd73d0c6851fad2c0b`.
The previously inspected endpoint certificate expires on **2026-12-13
18:51:39 UTC**; check live validity and renew/enroll approved trust before expiry.
Never transfer its private key or bypass certificate verification.

For a root shell using the corrected reviewed script and approved public CA:

```bash
export ALMADIAL_BINARY_URL=https://10.160.90.1:8447/releases/m4-06-acquisition-dev/almadial-linux-amd64
export ALMADIAL_CA_FILE=/etc/pki/ca-trust/source/anchors/almadial-distribution.pem
bash almadial-install.sh
almadial install
```

This assumes the script has already been obtained through the owner's trusted
channel and the public CA has been enrolled as described in the operator guide.
`ALMADIAL_CA_FILE` alone does not establish trust for the Go CLI's source downloads.
The candidate's embedded source policy still selects this private bridge; moving
only its binary does not remove that network dependency. Relocation needs a
reviewed source policy and newly approved CLI, not a checksum override.

## Maintainer service and evidence

The existing private static server runs as the lab user and binds only the
approved bridge, with no listings/uploads/execution. Certificates, keys and
protected evidence stay outside its served root. The retained implementation
checkpoint contains `serve_private.py` and `retain_sources.py`. Its service was
configured with these concrete values:

```bash
python3 -u scripts/bootstrap/serve_private.py --root /home/ra/almadial-lab/distribution/alm34 --certificate /home/ra/almadial-lab/evidence/m4-06/bootstrap-20261113T190219Z-alm34/tls/cert.pem --key /home/ra/almadial-lab/evidence/m4-06/bootstrap-20261113T190219Z-alm34/tls/key.pem --bind 10.160.90.1 --port 8447
```

Do not restart or reconfigure it merely to edit documentation. Keep private
service availability and trust under owner control. Upstream source rights and
source-delivery obligations remain subject to the
[distribution decision](../compliance/m4-04/distribution-rights-decision.md).

Existing developer installation/media/NO_OP evidence is preserved under
`~/almadial-lab/evidence/m4-06/acquisition-20261113T193809Z-alm34/`.
This correction performs only rootless bootstrap/CLI checks and repository tests;
it does not repeat or replace independent operator acceptance.

The operator VM `almadial-alm34-operator` was previously observed **Deleted**, with
its pristine snapshot retained. It is untouched here. Its owner must resolve that
operational prerequisite separately before the independent run. ALM-34 remains
In Progress under Codex; independent human execution and AGY assessment remain
outstanding. ALM-35 is not started by this correction.

## Portability correction verification

The corrected rendered script SHA-256 is
`8e3149cadb2cf8512a1dd87a2ea79c296d7ae59c665c49f84327bf4c056c22a0`.
It retains the existing approved CLI digest above and no default binary URL.
This script has not been published or deployed by this assignment.

- **PASS:** actual approved CLI retrieved with the corrected script from the
  existing private HTTPS endpoint and a separate disposable HTTPS origin serving
  identical binary bytes. Root, OS identity and destination were emulated;
  installation stayed in temporary directories. This is not a clean-host run.
- **PASS:** 11 bootstrap tests cover configured/alternate/default origins,
  checksum override rejection, untrusted TLS, missing configuration, unsafe URLs,
  HTTP failure/no redirect following, unsupported hosts, verified launch failure,
  release rendering and cleanup/preservation of the prior executable.
- **PASS:** full current-tree Python suite, 251 tests; scaffold validation;
  Bash syntax, tracked Go formatting and Git whitespace checks.
- **PASS:** Go 1.27.3 `go vet ./...`, `go test ./...`, `go test -race ./...`,
  `go mod verify`, and `go mod tidy -diff` on the current source tree.
- **PASS:** existing `internal/artifacts` suite and `TestGuided*` CLI tests from an
  isolated temporary archive of retained commit `a9d58ba...`; its implementation
  was not copied into the current Go tree.
- **PASS:** the approved CLI's version, install/help and diagnostic/help interfaces;
  four documents' 20 relative links/anchors and 10 Bash examples validated.
- **UNAVAILABLE:** GitHub bootstrap at the two observed immutable commits,
  HTTP 404. No script byte comparison or public availability is claimed.
- **NOT RUN:** new VM provisioning, media qualification or independent acceptance.
  Existing accepted evidence and the operator VM were untouched.

All 386 pre-existing unrelated untracked files were hash-checked unchanged.
Validation logs and rootless fixture output are held under
`/tmp/almadial-portability-*`; temporary test output is not historical acceptance
evidence and is not committed. Kaneo was accessed read-only.
