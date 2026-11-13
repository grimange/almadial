# AlmaDial operator installation

The currently qualified precompiled candidate targets fresh **AlmaLinux 10.2
x86_64** single-server local/LAN installations. Qualification used four CPUs,
8 GiB RAM and 80 GiB disk; these are a starting recommendation, not production
sizing or a tested minimum. Obtain owner authorization and a recovery snapshot.
Keep SELinux, firewalld, certificate validation and signed repositories enabled.

Public GitHub bootstrap delivery is not available at the verified published
commit. The release owner supplies the reviewed bootstrap and approved CLI URL
through an authorized distribution channel. The portable script in this source
tree is not automatically deployed to an existing private server by a local edit
or commit. See the [maintainer distribution record](../development/private-distribution-lab.md)
for the current candidate's location, identity and deployment limitations.

## Distribution configuration

In a root login shell on the target, set and export the owner-supplied values:

| Setting | Meaning |
|---|---|
| `ALMADIAL_BOOTSTRAP_URL` | Trusted HTTPS location of the reviewed rendered script; used by your download command |
| `ALMADIAL_BINARY_URL` | Full approved HTTPS URL of `almadial-linux-amd64`; the script accepts a different location only for the same pinned binary bytes |
| `ALMADIAL_CA_FILE` | Optional readable public CA bundle for the binary download; omitted when OS trust suffices |

Configure these in the target root shell after `sudo -i`, if needed; sudo need
not preserve your original environment. Obtain URLs and trust through an
independently trusted release-owner channel. Do not execute an unreviewed script
merely because its server uses HTTPS. The script pins the approved CLI version
and SHA-256; neither is configurable through environment variables.

For private HTTPS, verify the public CA fingerprint supplied independently by
the owner. Enroll that CA using the normal AlmaLinux trust procedure, for example
with the reviewed public file named `almadial-distribution.pem`:

```bash
sha256sum almadial-distribution.pem
# Compare with the owner's independently supplied fingerprint before continuing.
install -o root -g root -m 0644 almadial-distribution.pem /etc/pki/ca-trust/source/anchors/almadial-distribution.pem
update-ca-trust
```

Enrollment supports the bootstrap download, CLI and later source acquisition.
`ALMADIAL_CA_FILE` affects only the bootstrap's binary download; it does not enroll
trust for the initial curl command, the Go CLI or your browser. Distribution trust
and application HTTPS/WSS trust are separate. Do not bypass TLS verification.

## Install

After obtaining both approved HTTPS URLs and establishing trust:

```bash
ssh root@YOUR_SERVER

read -r -p "Approved bootstrap HTTPS URL: " ALMADIAL_BOOTSTRAP_URL
read -r -p "Approved CLI HTTPS URL: " ALMADIAL_BINARY_URL
export ALMADIAL_BINARY_URL
curl -fsSLo almadial-install.sh "$ALMADIAL_BOOTSTRAP_URL"

# Inspect the script and its release identity before execution.
bash almadial-install.sh

almadial install
```

If root SSH is unavailable, use the cloud image's configured operator account and
`sudo -i`. No Go, Git, checkout, manual binary transfer or source archive staging
is required on the target. The bootstrap verifies the binary before launching it
or replacing `/usr/local/sbin/almadial`; a failed download/checksum/launch leaves
the existing executable intact. A missing URL produces an actionable error.
An approved mirror changes location only, never the pinned binary identity.

DNS, synchronized time, approved signed package repositories and authorized
source origins must be reachable. The CLI separately embeds its reviewed
source-store policy: changing `ALMADIAL_BINARY_URL` does not relocate that store.
For the existing candidate, its private source service must remain reachable
unless the host already has all verified cached objects. See
[artifact retention](almadial-private-artifact-retention.md) for release-owner
policy changes. Operators do not repair unavailable sources by changing hashes.

## Guided configuration and completion

The installer asks for the snapshot and authorization references, private IPv4
address/interface, and TLS choice. The supported hostname is `localhost` through
an SSH tunnel. Choose `lab` to generate a private application CA and localhost
certificate, or `provided` to give paths to a valid localhost certificate,
matching private key and trusted CA. Public DNS and ACME are unnecessary. This
application CA is separate from the bootstrap endpoint CA. Credentials are
created in protected server files; no passwords belong in logs or evidence.

The CLI verifies pinned archives, displays the installation summary and plan
SHA-256, and asks once for `yes`. It executes that exact plan and reports progress
and diagnostics. It never asks you to copy the digest into another command.

After installation, open HTTPS/WSS TCP tunnels from your workstation:

```bash
ssh -N -L 8443:127.0.0.1:443 -L 8089:127.0.0.1:8089 root@SERVER_IP
```

Import the **public application CA** into browser trust using the approved
certificate procedure. Generated lab TLS stores it at
`/var/lib/almadial-inputs/tls/trust.pem`; never copy `key.pem`. Open
`https://localhost:8443/vicidial/admin.php`, `/agc/vicidial.php` or
`/almadial-phone/`. Administrator `6666` uses `/etc/almadial/secrets/admin`;
synthetic phone `alm_s10_9061` uses `/etc/almadial/secrets/phone-s10`. Read these
only in a private operator session. See the [operator handover](almadial-operator-handover.md)
for application use and recording recovery.

Verify the completion summary and run:

```bash
almadial status --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
almadial doctor --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
```

Short-lived lab certificates can produce a documented warning (exit 10).
Required service, schema, HTTPS and security failures need investigation.
SSH TCP tunnels do not carry WebRTC UDP media. Registration alone does not prove
two-way audio or recording; use the separately approved functional media path
and qualification procedure in the operator handover.

For a failure, follow its diagnostic and rerun `almadial install` or
`almadial resume`. The saved exact plan is reviewed again; actual state is
reconciled and completed work preserved. Verified archives work offline.
The older handover's stage-specific recovery section describes the earlier CLI;
use this guided resume procedure for the precompiled candidate.
Corrupt cache bytes are quarantined and acquisition retried. If every approved
origin fails, stop and give the owner the component and exhausted-origin error;
the owner must restore the exact hash-bound object or review a controlled repin.
Do not change digests or substitute newer sources. Preserve
`/var/lib/almadial-inputs` and `/var/lib/almadial-install`; differing saved inputs
require owner reconciliation. Repeating a completed installation must report
`NO_OP`.

A P0 blocker is a missing mandatory installation, trusted access, media,
recording, security, reboot, recovery or repeat result, an unsafe retry or a
credential leak. Preserve private logs and report the command, exit status and
sanitized finding. Independent human acceptance follows the published procedure
without implementer repair; AGY separately assesses that evidence. Developer
qualification does not establish independent acceptance.
