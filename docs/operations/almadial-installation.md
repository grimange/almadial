# AlmaDial operator installation

The precompiled development candidate targets fresh **AlmaLinux 10.2
x86_64** single-server installations. The corrected remote candidate passed an
ordinary manual campaign and agent workflow, including two-way browser audio,
recording, disposition and logout on a controlled off-host network. Native
runtime, reboot recovery and zero-action reinstallation passed developer checks;
independent QA is outstanding. Remote public-host acceptance has not run;
controlled testing and isolated qualification are separate results. Qualification used four CPUs,
8 GiB RAM and 80 GiB disk; these are a starting recommendation, not production
sizing or a tested minimum. Obtain owner authorization and a recovery snapshot.
Keep SELinux, firewalld, certificate validation and signed repositories enabled.

The [published bootstrap](https://raw.githubusercontent.com/grimange/almadial/f885482478e2fa13fa95a1211c0ab688363b617d/scripts/bootstrap/install.sh)
was retrieved and verified against that commit. It pins an earlier laboratory
binary and cannot select a new remote-profile binary by changing its URL.
Obtain the release owner’s reviewed rendered script and matching CLI URL for
this implementation. A public remote-profile release has not been published.

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

DNS, synchronized time, signed package repositories and approved HTTPS sources
must be reachable. Set `ALMADIAL_ARTIFACT_STORE` to the approved `/sources` base
before `almadial install`, or enter it at the prompt. The CLI derives
`/sha256/DIGEST/COMPONENT.artifact` locations while retaining every manifest hash.
OS trust must include that source service’s CA. This is independent of the
bootstrap binary URL. Cached verified archives work offline.

## Guided configuration and completion

Choose `remote` (the default) for direct browser access, or `lab` to retain
loopback isolation. Supply snapshot and authorization references for your
authorized disposable target. Remote inputs are the FQDN, directly attached
IPv4/interface, matching certificate, private key and CA paths, and RTP range
(default 10000–10019 UDP). The server must resolve the hostname to that address;
browsers must resolve it too and trust its certificate. The initial profile does
not support a separate NAT/public address. Certificate files must be root-owned,
private regular files; keep keys on the server. Lab TLS can be generated.

The CLI verifies pinned archives, displays the installation summary and plan
SHA-256, and asks once for `yes`. It executes that exact plan and reports progress
and diagnostics. It never asks you to copy the digest into another command.

For remote access, open the configured hostname directly:
`https://YOUR_FQDN/vicidial/admin.php`, `/agc/vicidial.php` and
`/almadial-phone/`. The phone signals through `wss://YOUR_FQDN/ws`; no SSH tunnel
is needed. HTTPS 443 and the exact configured UDP media range must be reachable
from the browser. No database or internal management ports need external access.
See [deployment profiles](deployment-profiles.md) for lab access and limitations.

Administrator `6666` uses `/etc/almadial/secrets/admin`; the synthetic phone
`alm_s10_9061` uses `/etc/almadial/secrets/phone-s10`. Read these only in a private
operator session. The synthetic phone is not a campaign agent. Create ordinary
phones, users and campaigns in VICIdial administration. Browser phones use the
installed `ALMADIAL_WEBRTC` configuration template and `default` context;
assign the phone and campaign to the intended agent. The remote installer owns
the required native updater, listener, sender, autodial and maintenance setup.
Do not enable a competing keepalive supervisor or the lab's deferred units.
Never export credentials, private keys or recordings into public support reports.

Verify the completion summary and run:

```bash
almadial status --plan /var/lib/almadial-inputs/plan.json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
almadial doctor --plan /var/lib/almadial-inputs/plan.json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
```

Short-lived lab certificates can produce a documented warning (exit 10).
Required service, schema, HTTPS and security failures need investigation.
Remote diagnostics use `--plan` to select the configured hostname and policy.
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
