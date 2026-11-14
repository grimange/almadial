# AlmaDial

AlmaDial is a Go-based installer and operational management tool for a native,
single-server VICIdial system on supported AlmaLinux hosts. It provisions and
checks Asterisk, MariaDB, Apache/PHP, TLS, WebRTC and supporting services.
VICIdial remains the upstream dialer application; AlmaDial manages its installation.

AlmaDial targets **AlmaLinux 10.2 on x86_64** with two deployment profiles:
remote single-server HTTPS access and an isolated laboratory profile.
The remote profile is a development candidate: controlled off-host HTTPS/WSS,
two-way synthetic media and recording have passed, but ordinary VICIdial agent
operation is blocked by the deferred agent runtime. Public-host acceptance has
not run. A precompiled private development build is available to authorized
operators. Independent operator acceptance is pending; production deployment and
unrestricted public distribution are not approved.

## Quick installation

**Distribution status:** the source bootstrap is available on GitHub at the
[verified immutable reference](https://raw.githubusercontent.com/grimange/almadial/f885482478e2fa13fa95a1211c0ab688363b617d/scripts/bootstrap/install.sh).
That published script pins the earlier laboratory CLI, **not this remote-profile
build**. For remote installation, obtain the reviewed script and matching
precompiled CLI URL from your release owner. No public remote-profile binary
release is available. The configured distribution and source stores must be
reachable from your server; no developer network address is built into the
installation workflow.

After completing the [trust and endpoint setup](docs/operations/almadial-installation.md#distribution-configuration),
connect to the fresh server and run in a root login shell:

```bash
ssh root@YOUR_SERVER

read -r -p "Approved bootstrap HTTPS URL: " ALMADIAL_BOOTSTRAP_URL
read -r -p "Approved CLI HTTPS URL: " ALMADIAL_BINARY_URL
export ALMADIAL_BINARY_URL
curl -fsSLo almadial-install.sh "${ALMADIAL_BOOTSTRAP_URL:?Set the approved bootstrap URL}"

# Inspect the downloaded script before running it.
bash almadial-install.sh

almadial install
```

Export `ALMADIAL_BINARY_URL` to select the approved CLI location. Its version and
SHA-256 stay pinned in the reviewed script; changing the URL cannot select a
different binary. Private deployments may also set `ALMADIAL_CA_FILE` to an
approved public CA file. The installation guide gives the complete procedure.
If the cloud image requires an operator account, SSH with that account and use
`sudo -i` before configuring these variables.

The target needs no Go compiler, Git checkout, manual binary transfer, source
archive staging, individual stage commands or copied plan hashes.

## Features

The installer provides these capabilities, subject to the deployment limitations below:

- Guided configuration, host preflight and one approval of the exact immutable plan.
- Automatic acquisition of pinned source archives with SHA-256 verification,
  approved-store fallback and offline reuse of verified cached artifacts.
- MariaDB, Asterisk/PJSIP, VICIdial and VICIphone integration with HTTPS/WSS,
  ConfBridge and SELinux/firewalld-aware configuration.
- Durable journals, actual postcondition checks and documented resume/reconciliation.
  Repeating a healthy completed installation produces `NO_OP`.
- Read-only status and diagnostics, sanitized support export, and documented
  recording backup and isolated restoration.

## Supported environments and prerequisites

| Item | Requirement or qualified baseline |
|---|---|
| OS and CPU | Fresh AlmaLinux **10.2**, **x86_64 / Linux amd64** |
| Resources | **4 vCPUs, 8 GiB RAM, 80 GiB disk** used in qualification; a recommended starting point, not a tested minimum or production capacity estimate |
| Access | Root or an authorized account with `sudo`, working SSH and a recovery snapshot |
| Network | DNS, synchronized time, HTTPS access to configured approved distribution sources and signed package repositories |
| Trust | Trusted distribution CA where private HTTPS is used; separately, trusted application TLS for the browser |
| Source availability | The release owner maintains the exact approved source objects at origins authorized by the CLI's release policy |

Keep SELinux enforcing, firewalld enabled and signature/certificate validation
active. Reserve additional recording/backup storage for your retention needs.
The remote profile requires a hostname resolving to the server’s directly
attached IPv4 address and a matching trusted certificate. Supply that certificate;
AlmaDial does not provision ACME. Other operating systems, carrier/PSTN, NAT
traversal and production capacity are not qualified.

The binary location and source store are independent configuration. Set
`ALMADIAL_ARTIFACT_STORE` to an approved HTTPS content-addressed source-store base,
or enter it when prompted. Source versions and SHA-256 identities stay pinned.
The saved store policy is reused on resume; relocation requires deliberate
review of saved inputs. See the [retention guide](docs/operations/almadial-private-artifact-retention.md).

## Guided installation

Run `almadial install` in an interactive root terminal. Choose **remote** for
normal browser access or **lab** for isolated loopback access. For remote, supply
the FQDN, directly attached IPv4/interface, matching certificate/key/CA paths,
and the UDP media range. DNS, trust and host readiness are checked before
provisioning. Credentials are generated into protected server files.

The installer verifies pinned archives, displays its plan and SHA-256, then asks
once for `yes`. It executes that exact plan and reports progress and completion
checks. See the [installation guide](docs/operations/almadial-installation.md)
for prerequisites and recovery.

## Common commands

These forms are supported by the qualified precompiled candidate:

| Command | Purpose |
|---|---|
| `almadial install` | Guided configuration, exact-plan approval and installation |
| `almadial resume` | Review the saved plan and continue after actual-state reconciliation |
| `almadial plan --request FILE --manifest FILE` | Advanced read-only plan generation |
| `almadial status --plan FILE --ca-file FILE` | Inspect health; add `--json` for structured output |
| `almadial doctor --plan FILE --ca-file FILE` | Include expected conditions and diagnostic guidance |
| `almadial evidence export --output /absolute/private/support.tar.gz --ca-file FILE` | Create a sanitized bundle in an existing private directory |
| `almadial --version` | Show CLI version and build identity |
| `almadial install --help` | Show guided and explicit-plan options |

`FILE` means an existing reviewed input. Status/Doctor also accept explicit
manifest and journal paths. They inspect the host without repairing it.

## Accessing VICIdial

For the **remote** profile, open the configured hostname directly in a browser
that trusts the application certificate. For example, with your own hostname:

| Interface | URL path |
|---|---|
| Administration | `https://YOUR_FQDN/vicidial/admin.php` |
| Agent login | `https://YOUR_FQDN/agc/vicidial.php` |
| Browser phone | `https://YOUR_FQDN/almadial-phone/` |

WSS uses the same origin at `/ws`; Asterisk’s internal WebSocket listener stays
on loopback. Browser media uses the configured UDP range directly to the server.
The included synthetic phone and conference support a bounded call check. The
remote candidate still disables VICIdial's updater and manager listener, and
does not provision an ordinary agent campaign. This is an implementation blocker,
not a missing operator configuration step. Doctor reports it as a blocking
failure; do not use this candidate for normal agent operation or manually enable
those services to bypass the installer. A registration or answered call alone
does not prove audio.

The **lab** profile retains loopback HTTPS/WSS and optional SSH TCP forwarding;
see the [deployment profiles](docs/operations/deployment-profiles.md).
An SSH TCP tunnel does not carry WebRTC UDP media.

## Troubleshooting

| Situation | Next step |
|---|---|
| No binary endpoint configured | Obtain the release owner's approved CLI URL and export `ALMADIAL_BINARY_URL`; no source edit or checksum override is needed. |
| Download cannot connect | Check connectivity to the configured source and ask its owner to verify availability. |
| Certificate validation fails | Check system time and approved CA trust; obtain corrected trust from the owner. Keep verification enabled. |
| Checksum mismatch | Stop. The download is rejected before execution/replacement; ask the owner to restore the exact approved binary. |
| Source acquisition fails | Preserve the exhausted-origin diagnostic; restore the exact pinned object or obtain a reviewed policy correction. Do not substitute newer sources. |
| Installation stops | Follow the diagnostic, preserve saved inputs/journals and use `almadial resume`. Escalate conflicting inputs or unsafe recovery. |
| Status/Doctor warns | Investigate its findings. Exit 10 means degraded status; service, schema, TLS or security failures need attention. |
| Phone registration/media fails | Check application CA trust, WSS and the separately qualified media path. Follow the operator handover. |

Keep credentials, keys and recordings out of support submissions.

## Documentation

- [Operator installation and recovery](docs/operations/almadial-installation.md)
- [Private artifact retention and release configuration](docs/operations/almadial-private-artifact-retention.md)
- [Operational handover and troubleshooting](docs/operations/almadial-operator-handover.md)
- [Recording backup and isolated restoration](docs/operations/recording-backup-and-restore.md)
- [Maintainer laboratory distribution details](docs/development/private-distribution-lab.md)
- [Third-party notices](docs/THIRD_PARTY_NOTICES.md)

## Project status and licensing

Clean-host installation, recovery, diagnostics, isolated HTTPS/WSS, browser media,
recordings and repeat-run behavior have qualification evidence. Developer
qualification does not replace the pending independent operator run and review.
The stack manifest remains candidate. This source tree includes the guided
installer, durable acquisition and remote-profile implementation. Public-host
and independent operator acceptance remain separate gates; a local commit does
not publish a new bootstrap or binary.

The remote candidate is not ready for independent acceptance until its normal
agent runtime and campaign workflow are implemented and tested. The successful
controlled synthetic call does not establish that workflow or public-network
operation. The isolated lab remains a separate regression profile.

AlmaDial-original code is licensed under [MIT](LICENSE). Upstream applications,
libraries and OS packages retain their own licenses and notices. Technical SBOM
acceptance does not grant unrestricted redistribution rights. See the
[distribution decision](docs/compliance/m4-04/distribution-rights-decision.md) for
outstanding rights and source-delivery conditions.
