# AlmaDial

AlmaDial is a Go-based installer and operational management tool for a native,
single-server VICIdial system on supported AlmaLinux hosts. It provisions and
checks Asterisk, MariaDB, Apache/PHP, TLS, WebRTC and supporting services.
VICIdial remains the upstream dialer application; AlmaDial manages its installation.

The qualified profile is **AlmaLinux 10.2 on x86_64, in an isolated local/LAN
environment**. A precompiled private release candidate is available to authorized
operators. Independent operator acceptance is pending; production deployment and
unrestricted public distribution are not approved.

## Quick installation

**Distribution status:** a GitHub-hosted bootstrap is not currently available.
The published repository commit was checked and its bootstrap path returned HTTP
404. Obtain the reviewed bootstrap URL, approved CLI download URL and any required
public CA certificate from your release owner. These are deployment configuration,
not fixed AlmaDial network addresses. No public binary release is implied.

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

The qualified precompiled candidate provides:

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
Public DNS and ACME are unnecessary for this profile. Other operating systems,
public carrier/PSTN, public Internet access and NAT traversal are not qualified.

The CLI download location and upstream source-store policy are separate. Moving
the binary to a different approved HTTPS origin does not change its embedded
source origins. The current candidate still needs access to its approved private
source store, or its verified cache. Source-store relocation requires a reviewed
policy and a newly approved CLI build, as described in the
[retention guide](docs/operations/almadial-private-artifact-retention.md).

## Guided installation

Run `almadial install` in an interactive root terminal. It checks readiness and
prompts for the authorization/recovery references, private network address and
interface, and TLS configuration. The qualified application hostname is
`localhost`. Choose generated laboratory TLS or supply a reviewed matching
certificate, key and trusted CA. Credentials are stored in protected server files.

The installer acquires and verifies its pinned inputs, displays proposed changes
and the plan SHA-256, then asks once for `yes`. It executes that exact plan,
reports progress and verifies completion. See the
[operator installation guide](docs/operations/almadial-installation.md) for inputs,
credential locations, progress and recovery.

## Common commands

These forms are supported by the qualified precompiled candidate:

| Command | Purpose |
|---|---|
| `almadial install` | Guided configuration, exact-plan approval and installation |
| `almadial resume` | Review the saved plan and continue after actual-state reconciliation |
| `almadial plan --request FILE --manifest FILE` | Advanced read-only plan generation |
| `almadial status --ca-file FILE` | Inspect health; add `--json` for structured output |
| `almadial doctor --ca-file FILE` | Include expected conditions and diagnostic guidance |
| `almadial evidence export --output /absolute/private/support.tar.gz --ca-file FILE` | Create a sanitized bundle in an existing private directory |
| `almadial --version` | Show CLI version and build identity |
| `almadial install --help` | Show guided and explicit-plan options |

`FILE` means an existing reviewed input. Status/Doctor also accept explicit
manifest and journal paths. They inspect the host without repairing it.

## Accessing VICIdial

Application HTTPS and WSS listeners are bound to loopback in the qualified
profile. From your workstation, establish the TCP tunnels:

```bash
ssh -N -L 8443:127.0.0.1:443 -L 8089:127.0.0.1:8089 root@YOUR_SERVER
```

Trust the **public application CA** in the browser; never transfer the private key.
Then open:

| Interface | Address through the tunnel |
|---|---|
| Administration | `https://localhost:8443/vicidial/admin.php` |
| Agent login | `https://localhost:8443/agc/vicidial.php` |
| Browser phone | `https://localhost:8443/almadial-phone/` |

The phone uses trusted `wss://localhost:8089/ws`. **SSH TCP tunnels do not carry
WebRTC UDP media.** Registration alone proves neither two-way audio nor recording;
the separate documented functional media qualification requires its approved
network path. Do not open additional firewall ports to make this profile public.

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
The stack manifest remains candidate. The qualified private binary contains guided
installation/acquisition additions retained on a separate implementation branch;
this source tree's Go installer predates those additions. The portable bootstrap
changes do not replace that binary or constitute a new public release.

AlmaDial-original code is licensed under [MIT](LICENSE). Upstream applications,
libraries and OS packages retain their own licenses and notices. Technical SBOM
acceptance does not grant unrestricted redistribution rights. See the
[distribution decision](docs/compliance/m4-04/distribution-rights-decision.md) for
outstanding rights and source-delivery conditions.
