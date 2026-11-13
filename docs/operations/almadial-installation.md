# AlmaDial laboratory installation

This procedure uses the qualified private precompiled candidate. GitHub bootstrap
delivery remains pending verification; the current remote `main` does not contain
the guided bootstrap/acquisition implementation. No public release is implied.
Enroll the private endpoint CA as described below before downloading the script.

```bash
ssh root@SERVER_IP

curl -fsSLO https://10.160.90.1:8447/install.sh

bash install.sh

almadial install
```

Use a fresh, owner-authorized **AlmaLinux 10.2 x86_64** lab server with four
CPUs, 8 GiB RAM, 80 GiB disk and a recorded recovery snapshot. Keep SELinux,
firewalld, certificate validation and signed repositories enabled. This isolated
single-server profile is a laboratory qualification, not production sizing.
On the supplied cloud image, log in as `ubuntu` and run `sudo -i` when root SSH
is unavailable; run the remaining commands in that root login shell.

The real endpoint is on the private Multipass bridge. The lab owner must keep
it running and give the operator its public CA through the approved trusted
channel. Before the four-command workflow, verify that public certificate's
SHA-256 is `6e49e14cb26240689aa4ce67d1ff61bc8eac1c5dddf7fbdd73d0c6851fad2c0b`,
then enroll it as root:

```bash
sha256sum /path/from-owner/alm34-artifact-ca.pem
install -o root -g root -m 0644 /path/from-owner/alm34-artifact-ca.pem /etc/pki/ca-trust/source/anchors/almadial-alm34-artifacts.pem
update-ca-trust
```

CA enrollment was developer-tested; the independent operator host has not been
changed. Guests must reach `10.160.90.1:8447` and approved AlmaLinux package
repositories; DNS and synchronized time must work. The private store supplies all
five pinned source archives automatically. Original upstream HTTPS sites are
fallbacks, not prerequisites when the store is available. No Go, Git, source
checkout, manual archive download or binary transfer is needed. Inspect
`install.sh` before execution; it verifies the precompiled CLI before executing
or installing it in `/usr/local/sbin/almadial`. Never use `curl -k`.

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

After installation, open a tunnel from your workstation:

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
Registration alone does not prove two-way audio or recording.

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
