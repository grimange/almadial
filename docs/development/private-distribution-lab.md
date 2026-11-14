# Private distribution laboratory record

This document records a specific development environment. It is not a general
AlmaDial network requirement or a public release announcement. The
[operator guide](../operations/almadial-installation.md) describes the portable
configuration contract; the [retention guide](../operations/almadial-private-artifact-retention.md)
describes source policy and release ownership.

## Current accepted release candidate

This section supersedes the historical observations below for the authorized
ALM-34 operator installation. It is a private controlled-network candidate,
not a public release. The old artifact VM was unreachable after a graceful
restart; a new dedicated AlmaLinux HTTPS distribution VM was prepared from
preserved, verified approved objects. It does not mount or serve the developer
checkout. Keep this VM running for the operator test.

| Item | Verified value |
|---|---|
| Distribution VM | `almadial-alm34-distribution`, `10.160.90.48`, machine ID `67e3617bb8c730f1a37e4204da527f23` |
| Bootstrap | `https://10.160.90.48:8447/releases/alm46-native-rc1/install.sh` |
| Script SHA-256 | `622a530700763d97923b605bacbb8cd25b8c0594c91d096d2e0e099d4c5e38f4` |
| Binary | `https://10.160.90.48:8447/releases/alm46-native-rc1/almadial-linux-amd64` |
| Version / implementation | `alm46-native-rc1` / `92b14eb4452b7bfb949e0ee7efc571a83d9e799c` |
| Binary SHA-256 | `a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3` |
| Release manifest | Same release directory, `release.json`, SHA-256 `4c47e85a8043a069c75598eed40c0d03341eac663c3439979ca72af2df59ab3a` |
| Source store | `https://10.160.90.48:8447/sources` |
| Distribution public CA file SHA-256 | `037628408101278818defbb225840a48b439e7ac4ec0210c94a0ec1f0c63f8e1` |

The owner supplies **only the public CA** as `almadial-distribution.pem` through
an independently trusted channel. The public operator kit is at
`~/almadial-lab/operator-kits/alm34-native-rc1/`; it contains no binary, source
archive, credential or private key. Verify the CA file hash above before enrolling
it. The distribution certificate is laboratory-scoped, valid for the recorded
address and `artifacts.alm34.test`; check validity before later reuse. The CA
expires 2027-02-12 and the server leaf expires 2027-01-13 (UTC). Renewal is an
owner maintenance responsibility; never bypass TLS validation.

From an authorized target that can route to this service, using a root login
shell (`ssh ubuntu@SERVER` then `sudo -i` for the current cloud image):

```bash
sha256sum almadial-distribution.pem
# Compare with the independent CA hash above, then enroll the reviewed public CA.
install -o root -g root -m 0644 almadial-distribution.pem /etc/pki/ca-trust/source/anchors/almadial-distribution.pem
update-ca-trust

curl -fsSLo almadial-install.sh \
  https://10.160.90.48:8447/releases/alm46-native-rc1/install.sh
printf '%s  %s\n' 622a530700763d97923b605bacbb8cd25b8c0594c91d096d2e0e099d4c5e38f4 almadial-install.sh | sha256sum -c -
# Inspect the script before execution.
bash almadial-install.sh
almadial --version
printf '%s  %s\n' a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3 /usr/local/sbin/almadial | sha256sum -c -

export ALMADIAL_ARTIFACT_STORE=https://10.160.90.48:8447/sources
almadial install
```

Select `remote`; provide the authorized snapshot reference, FQDN, directly
attached IPv4/interface, matching application certificate/key/CA files, and the
intended RTP range (default 10000–10019 UDP). Target and browser must resolve the
FQDN correctly; the browser must trust the application certificate. Application
trust is separate from the distribution CA. Review the exact plan and answer
`yes` once. Do not copy hashes into another install command.

The installed binary retains one obsolete pre-QA runtime warning banner.
Independent ALM-46 QA supersedes that text; actual Status/Doctor failures must
still be investigated. No binary bytes were changed to remove the banner.

### Operator environment recovery — owner actions only

Read-only inspection found `almadial-alm34-operator` **Deleted**, with
`alm34-pristine` retained (4 CPUs, 8 GiB, 80 GiB, no mounts). Codex did not recover,
start or modify it. The owner can recover the existing instance and restore its
pristine snapshot while stopped:

```bash
multipass recover almadial-alm34-operator
multipass restore almadial-alm34-operator.alm34-pristine
multipass start almadial-alm34-operator
multipass info almadial-alm34-operator
```

Keep the restore command interactive so Multipass can offer to preserve the
current state. Do not purge or recreate the VM. Then connect using the reported
address and configured `ubuntu` account, and use `sudo -i`. Supply the reviewed
public distribution CA and separate application TLS/DNS inputs before starting.

### Independent operator P0 checklist

- Bootstrap from the versioned HTTPS URL and verify installed CLI identity above;
  no Go, Git, binary transfer or source staging.
- Select remote, acquire all five exact sources automatically, review and approve
  the plan once, and complete installation without implementer assistance.
- Open administration, agent and browser-phone paths through the configured HTTPS
  hostname without a tunnel. Authenticate, enter a campaign and register the phone.
- Complete an authorized ordinary internal manual call with two-way audio,
  verify a correlated non-silent recording on the server, disposition and logout.
- Check Status/Doctor using the operator guide; investigate failures. A truthful
  short-lived application certificate warning is not a silent runtime pass.
- Reboot and recheck functionality. Repeating `almadial install` must verify
  `no_op: true`, zero actions and exit 0. For interruptions, preserve inputs and
  journals and use `almadial resume`; report P0 failures before owner repair.

Public-host topology, physical headset perception, browser archived-recording
HTTP retrieval, carrier/PSTN and capacity remain outside this qualified scope.
Loopback recording URLs are a recorded follow-up, not proof of remote playback.
The human run and subsequent AGY assessment are separate from developer evidence.

## Historical distribution observations

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
