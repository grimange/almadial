# AlmaDial

AlmaDial is a Go-based installation and operations tool for deploying a native,
single-server VICIdial system on AlmaLinux. It provisions and checks Asterisk,
MariaDB, Apache/PHP, TLS, WebRTC and their supporting system services. VICIdial
remains the upstream dialer application; AlmaDial manages its installation.

The qualified deployment profile is **AlmaLinux 10.2, x86_64, on an isolated
local/LAN server**, with browser access through SSH tunnels. Installation is
currently available as a private laboratory candidate. Independent operator
acceptance is pending; this is not an unrestricted production release.

## Quick installation

Use a fresh, authorized server with a recovery snapshot. The current bootstrap
and its precompiled CLI are served from the private lab endpoint below. The
server must reach `10.160.90.1:8447`; this address is not publicly accessible.
**Enroll the lab owner's trusted public CA before downloading.** See the
[CA setup below](#private-ca-setup) or the [installation guide](docs/operations/almadial-installation.md).

```bash
ssh root@SERVER_IP

curl -fsSLo almadial-install.sh https://10.160.90.1:8447/install.sh

bash almadial-install.sh

almadial install
```

On the supplied cloud image, use `ssh ubuntu@SERVER_IP`, then `sudo -i`, and run
the remaining commands in that root login shell. Inspect the downloaded script
before execution. It verifies the precompiled CLI's pinned SHA-256 before running
or installing it in `/usr/local/sbin/almadial`.

You do not need Go, Git, a checkout, manual binary transfer, source-archive
staging, individual stage commands or plan-hash copying on the target.

A GitHub-hosted bootstrap is not yet available. This workflow uses the qualified
private candidate.

### Private CA setup

Obtain `almadial-artifact-ca.pem` from the lab owner through a trusted channel.
In the server's root login shell, verify the current public CA file and enroll it:

```bash
printf '%s  %s\n' \
  6e49e14cb26240689aa4ce67d1ff61bc8eac1c5dddf7fbdd73d0c6851fad2c0b \
  almadial-artifact-ca.pem | sha256sum --check &&
  install -o root -g root -m 0644 almadial-artifact-ca.pem \
    /etc/pki/ca-trust/source/anchors/almadial-artifacts.pem &&
  update-ca-trust
```

The current endpoint certificate expires on **13 December 2026 (UTC)**. Ask the
owner for reviewed replacement trust if it has expired or its fingerprint differs.
Keep certificate validation enabled. The application certificate used by the
browser is separate from this download endpoint's certificate.

## Features

- Guided configuration, host preflight, a readable installation plan and one
  explicit confirmation bound to the exact plan SHA-256.
- Automatic acquisition of pinned source archives, verified private-store
  fallback and reuse of verified cached artifacts, including offline reuse.
- Native MariaDB, Asterisk/PJSIP, VICIdial and VICIphone installation with
  HTTPS/WSS, ConfBridge and SELinux/firewalld-aware configuration.
- Durable installation journals, actual postcondition checks and documented
  resume/reconciliation. Repeating a healthy completed install produces `NO_OP`.
- Read-only status and diagnostics, a sanitized support-bundle command, and
  documented recording backup and isolated restoration.

## System requirements

| Item | Current qualified baseline / requirement |
|---|---|
| Operating system | Fresh AlmaLinux **10.2**, with SELinux enforcing and firewalld enabled |
| Architecture | **x86_64 / Linux amd64** |
| Resources | **4 vCPUs, 8 GiB RAM, 80 GiB disk** used in qualification; recommended starting point for the lab, not a tested minimum or production capacity estimate |
| Access | Local root or an authorized account with `sudo`; working SSH access and a recovery snapshot |
| Network | DNS, synchronized time, HTTPS access to the private artifact endpoint and the approved signed package repositories |
| TLS | Trusted artifact CA; generated lab application CA or supplied matching localhost certificate, private key and trusted CA |
| Sources | Lab owner keeps the approved private binary/source store available; verified local cache can be reused |

Reserve additional disk space for recordings and backups according to retention
needs. Public DNS and ACME are unnecessary for this profile. Public carrier,
Internet NAT and public browser access are outside the qualified configuration.

## What the installer does

Run `almadial install` in an interactive root terminal. It checks host readiness
and asks for the recovery snapshot and authorization references, private network
address/interface, and TLS configuration. The supported profile uses `localhost`
through an SSH tunnel. Choose `lab` for a generated private application CA, or
`provided` for reviewed certificate/key/CA files. Credentials are generated in
protected server files; the installation guide identifies their locations.

The installer acquires and verifies the pinned inputs, displays the proposed
changes and plan digest, then asks once for `yes`. It executes that exact approved
plan, reports progress, verifies postconditions and finishes with diagnostics.
Browser media and recording checks remain a separate functional qualification;
successful provisioning alone does not establish usable two-way audio.

## After installation

Check the completion summary, then run on the server:

```bash
almadial status --ca-file /etc/pki/tls/almadial-s08/current/trust.pem \
  --manifest /var/lib/almadial-inputs/stack.lock.yaml
almadial doctor --ca-file /etc/pki/tls/almadial-s08/current/trust.pem \
  --manifest /var/lib/almadial-inputs/stack.lock.yaml
systemctl is-active mariadb asterisk httpd php-fpm firewalld crond almadial-s06-fastagi
```

From your workstation, open the qualified HTTPS/WSS tunnel:

```bash
ssh -N -L 8443:127.0.0.1:443 -L 8089:127.0.0.1:8089 root@SERVER_IP
```

Trust the **public application CA** in your browser before opening these pages.
For generated lab TLS it is `/var/lib/almadial-inputs/tls/trust.pem` on the server;
never transfer the private key. Use your authorized SSH account if root login is
unavailable.

| Interface | Address through the tunnel |
|---|---|
| VICIdial administration | `https://localhost:8443/vicidial/admin.php` |
| Agent login | `https://localhost:8443/agc/vicidial.php` |
| Browser phone | `https://localhost:8443/almadial-phone/` |

Use the protected credentials described in the installation guide. Confirm the
phone uses trusted `wss://localhost:8089/ws`. Registration, two-way media and
recording are separate checks. Application listeners are loopback-only; do not
open firewall ports to make these URLs public.

## Common commands

Run installation and host diagnostics with authorized root access. These forms
are supported by the qualified candidate:

| Command | Purpose |
|---|---|
| `almadial install` | Guided configuration, plan approval and installation |
| `almadial resume` | Review the saved exact plan and continue after reconciliation |
| `almadial plan --request FILE --manifest FILE` | Advanced: inspect a reviewed request and render a plan without provisioning |
| `almadial status --ca-file FILE` | Inspect service and installation health; add `--json` for structured output |
| `almadial doctor --ca-file FILE` | Include expected conditions and diagnostic guidance |
| `almadial evidence export --output /absolute/private/support.tar.gz --ca-file FILE` | Write a bounded sanitized bundle to a new file in an existing private directory |
| `almadial --version` | Display CLI version and build identity |
| `almadial install --help` | Show guided and explicit-plan installation options |

`FILE` denotes an existing reviewed input, not a literal filename. Normal guided
installation does not require an operator-created request or manual digest entry.
Status/Doctor accept optional explicit manifest and journal paths; they never
guess a journal or repair the host.

## Troubleshooting

| Situation | Next step |
|---|---|
| Bootstrap endpoint cannot be reached | Check private lab routing and ask the owner to confirm the artifact service is running. It is not an Internet endpoint. |
| Certificate validation fails | Check time, the approved CA fingerprint, enrollment and certificate validity. Obtain corrected trust from the owner. |
| Artifact acquisition fails | Preserve the component/origin diagnostic. The owner must restore the exact pinned object or review a controlled repin; do not substitute a newer archive or change its digest. |
| Installation stops at a postcondition | Preserve the diagnostic, correct the documented prerequisite and run `almadial resume`. Keep `/var/lib/almadial-inputs` and `/var/lib/almadial-install`; escalate conflicting inputs or unsafe recovery. |
| Status/Doctor exits 10 | Read the findings. Short-lived lab certificates can produce an expected warning; required service, schema, TLS or security failures need investigation. |
| Phone fails to register or has no audio | Check both tunnels, browser CA trust and WSS configuration. Registration alone does not prove media or recording. Follow the operator handover. |

The [installation guide](docs/operations/almadial-installation.md) covers resume
and required inputs. The [operator handover](docs/operations/almadial-operator-handover.md)
provides detailed read-only diagnostics and escalation guidance. Preserve private
logs and keep credentials, private keys and recordings out of support submissions.

## Documentation

- [Operator installation guide](docs/operations/almadial-installation.md)
- [Private artifact retention and availability](docs/operations/almadial-private-artifact-retention.md) — lab maintainers
- [Operational handover and troubleshooting](docs/operations/almadial-operator-handover.md)
- [Recording backup and isolated restoration](docs/operations/recording-backup-and-restore.md)
- [Third-party notices and release conditions](docs/THIRD_PARTY_NOTICES.md)

## Project status and licensing

Clean-host installation, recovery, diagnostics, isolated HTTPS/WSS, browser media,
recordings and repeat-run behavior have qualification evidence. The guided
bootstrap candidate has developer qualification; independent unassisted operator
acceptance and its separate review remain outstanding. The stack manifest remains
candidate, and public release and production use require separate approval.

The guided bootstrap and automatic acquisition additions are currently supplied
by the private candidate binary; the repository's published `main` source predates
those additions. GitHub bootstrap delivery remains pending verification.

AlmaDial-original code is licensed under [MIT](LICENSE). VICIdial, Asterisk,
PJPROJECT, VICIphone, Perl modules, browser libraries and OS packages retain their
own licenses and notices. Technical SBOM acceptance does not grant unrestricted
redistribution rights. The private artifact arrangement is scoped to the
authorized lab; outstanding source-delivery and rights conditions are recorded in
the [distribution decision](docs/compliance/m4-04/distribution-rights-decision.md).
