# AlmaDial operator installation

The standard-installer correction targets fresh **AlmaLinux 10.2 x86_64**
single-server installations. `alm34-standard-rc1` is a locally prepared candidate;
publication and independent operator acceptance are separate gates. The previous
public `alm46-internet-rc1` files and checksums remain unchanged. Its accepted
native runtime and controlled-network functional QA remain historical evidence;
that release does not include this usability correction. Public-host qualification
must be reported separately from controlled-topology developer tests.

## Requirements and resource guidance

| Resource | Enforced fresh-install threshold | Recommended qualified baseline |
|---|---|---|
| CPU | 4 CPUs with SSE2 | 4 vCPU |
| Total RAM | 7 GiB | 8 GiB RAM |
| Available RAM | 4 GiB at preflight | Leave memory available for installation |
| Installation filesystem free space | 60 GiB on each checked `/`, `/usr`, `/var` filesystem | 80 GiB disk |
| Recording filesystem | 1 GiB available and 10,000 free inodes | Additional space for retention and backups |

GiB means 1,073,741,824 bytes. These are installation checks, not a capacity
benchmark or a production sizing guarantee. Resource failures report the affected
resource, required and observed values and corrective action. Allocate more CPUs
or RAM, free memory, expand/free the named filesystem, or resolve inode exhaustion
before retrying. Do not lower saved policy to evade a failure.

## Standard installation contract

Connect to your supported host as root, or connect with the cloud-image account
and enter `sudo -i`. After an approved corrected version is publicly released:

```bash
ssh root@YOUR_SERVER
VERSION='APPROVED_VERSION'
curl -fsSLO "https://github.com/grimange/almadial/releases/download/${VERSION}/install.sh"
# Compare its SHA-256 with the independently approved release packet.
sha256sum install.sh
# Inspect the verified script before execution.
bash install.sh
almadial --version
sha256sum /usr/local/sbin/almadial
almadial install
```

The version and binary hash must match the approved release packet. Distribution
TLS uses the OS trust store; application/browser TLS requires separate preparation.
A backup, provider recovery option or snapshot is recommended at the operator's
discretion. AlmaDial does not verify external snapshots or guarantee full rollback.
No hypervisor, Kaneo task, internal authorization reference or recovery assertion
is a prerequisite for standard installation, including the isolated lab profile.

No Go, Git, manual executable/source transfer, source edit, stage commands or
copied plan hashes are required. The rendered release script automatically chooses
its versioned GitHub binary location and pins its checksum. Failed downloads,
untrusted TLS, mismatches and failed launch preserve the previous executable.
GitHub asset redirects are HTTPS-only and bounded.

The bootstrap obtains the SVN acquisition utility, when missing, through DNF
with package signature checking enabled. It does not provision VICIdial in Bash.
DNS, synchronized time, HTTPS release/upstream access, signed AlmaLinux package
repositories and outbound TCP 3690 to the authoritative VICIdial SVN service must
be reachable. Standard distribution uses OS CA trust. Application/browser TLS is
a separate prerequisite: provide your FQDN's matching trusted certificate, key
and chain; the installer does not provision ACME.

The new source policy requires no `ALMADIAL_ARTIFACT_STORE`. Asterisk, PJPROJECT,
Net::Telnet and VICIphone use their pinned public HTTPS origins and exact archive
hashes. VICIdial first tries a valid cache, optional authorized mirrors and the
original archive, then retrieves authoritative r4037 and verifies all 2,511 files
against the independently accepted source-tree digest. A reconstructed ZIP has
its own recorded SHA-256; it is not claimed to equal the original nightly ZIP.
SVN transport does not authenticate a publisher; the trusted accepted tree digest
binds all paths and contents before provisioning. Cached inputs are reverified
before every use. No unverified source is executed.

## Optional private and offline deployments

`ALMADIAL_BINARY_URL` selects an alternate approved binary location while the
release checksum remains fixed. `ALMADIAL_CA_FILE` supplies a reviewed public CA
bundle for that binary request only. Neither is required by the standard release.
For private source mirrors, enroll the independently authenticated CA through
normal AlmaLinux OS trust and optionally set `ALMADIAL_ARTIFACT_STORE` before the
first guided install. The mirror must supply the original exact archive bytes.
Saved policy is reused on resume; do not edit saved inputs to bypass reconciliation.
See the [retention guide](almadial-private-artifact-retention.md). Never disable
certificate verification. A bootstrap-specific CA variable does not configure
Go source acquisition or application browser trust.

## Guided configuration and completion

Standard installation presents the following prompts, in order (one address is
proposed only when exactly one suitable assigned address/interface pair exists):

```text
Deployment profile: remote (direct HTTPS/WSS and UDP media) or lab (loopback) [remote]
Server FQDN matching the supplied trusted certificate
Directly attached server IPv4 management/media address [unique assigned IPv4, if any]
Management network interface for selected IPv4 [matching assigned interface, if unique]
Preserve existing SSH management port: <discovered-port>/tcp   (observation, not a prompt)
TLS: provided (certificate/key/CA files), or lab (localhost private CA only) [provided]
Hostname certificate full-chain PEM absolute path
Matching private key PEM absolute path
Issuing CA trust PEM absolute path
RTP UDP first port (even, at least 1024) [10000]
RTP UDP last port (open this exact range on the server network) [10019]
<verified source acquisition, installation warning, actual immutable plan JSON and SHA-256>
Approve this exact plan and execute the installation (yes/no) [no] (cancel to stop):
```

There are no snapshot/task prompts and no separate recovery confirmation. `yes`
approves the displayed exact plan once. `no`, blank input, EOF and cancel reject
provisioning. Preparation can download verified sources and save protected inputs;
package/service/database/firewall provisioning begins only after approval.

Effective `sshd -T` and actual sshd-owned listeners must agree on a single port;
22 and nonstandard ports such as 2222 are supported. The selected port is bound
into the immutable plan and allowed through the firewall. AlmaDial does not
reconfigure SSH. Multiple ports, conflicting listener addresses, or unsupported
socket activation require administrator reconciliation before installation.
The selected IPv4 must actually be assigned to the selected active interface;
no literal `eth0` default is assumed. With several choices, select the appropriate
assigned pair. The FQDN's IPv4 resolution must equal that address. This does not
prove internet reachability: arrange external DNS, HTTPS and the exact UDP media
range yourself. A public DNS address translated onto a private host address,
arbitrary NAT, multiple public media addresses and general TURN are unsupported
and must be resolved before provisioning.

## Prepare application TLS

Remote mode requires an FQDN/SAN match, matching unencrypted private key, valid
full chain and explicit CA trust PEM. Browsers must trust the same issuing CA.
Do not use `localhost` lab certificates for remote mode. AlmaDial has no ACME
client. Certificate-manager paths containing symlinks are deliberately rejected.
Prepare root-owned regular copies in a real directory; for example, as root:

```bash
install -d -o root -g root -m 0700 /etc/almadial-tls-inputs
install -o root -g root -m 0644 /YOUR_RESOLVED_SOURCE/fullchain.pem /etc/almadial-tls-inputs/fullchain.pem
install -o root -g root -m 0600 /YOUR_RESOLVED_SOURCE/privkey.pem /etc/almadial-tls-inputs/key.pem
install -o root -g root -m 0644 /YOUR_RESOLVED_SOURCE/ca-bundle.pem /etc/almadial-tls-inputs/trust.pem
openssl x509 -in /etc/almadial-tls-inputs/fullchain.pem -noout -subject -issuer -dates -ext subjectAltName
```

Resolve and inspect your certificate-manager source paths first. No symlink may
remain in any supplied destination path or parent directory. Inputs must be at
most 4 MiB, without group/world writes; private keys must have owner read/write,
no world access and only root or the accepted asterisk group. Prefer 0600
root:root for the input key. The CLI names the failing input/path and restriction;
it never prints key contents. Keep these original PEM bytes at their original
paths through resume: a saved plan binds certificate, chain, CA, hostname and
paths. Replacing certificate material can invalidate saved-plan reconciliation.
There is no automatic renewal/replanning of an installation; preserve the working
generation and obtain operator-led reconciliation before certificate replacement.
Never edit plan hashes or delete journals to force renewal. Bootstrap CA overrides
affect distribution downloads only, not application or browser trust.

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

## Current candidate qualification limitation

`alm34-standard-rc1` is prepared locally, unpublished and **not yet qualified for
operator retest**. Its clean AlmaLinux 10.2 developer run reached standard
exact-plan approval and verified public application source acquisition, then
stopped safely at RPM resolution. The configured repositories offered 10.3
replacements for the accepted `gcc`, `gcc-c++`, `libuuid-devel`, `libxml2-devel`,
`openssl-devel` and `perl-DBI` pins. The installer did not substitute those
versions. Preserve the saved plan and journals; a reviewed repository-retention
or stack-policy decision is required before qualification can complete. Do not
manually install replacements, alter the saved plan, or infer runtime health
from this partial installation. Historical ALM-46 runtime acceptance remains
limited to its recorded versions and topology.
