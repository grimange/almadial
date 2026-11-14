# AlmaDial operator installation

The published **`alm46-internet-rc1`** precompiled CLI targets fresh **AlmaLinux
10.2 x86_64** single-server installations. It preserves the accepted
`alm46-native-rc1` runtime with corrected public source acquisition. Native runtime,
ordinary manual
campaign/agent operation, bidirectional WebRTC audio, recording, reboot recovery
and healthy NO_OP passed independent controlled-network QA. Independent human
operator installation acceptance remains outstanding. Public VPS/DNS/CA topology,
physical headset perception and remote recording HTTP playback remain unqualified.

Qualification used four CPUs, 8 GiB RAM and 80 GiB disk: a recommended starting
point, not a tested minimum or production sizing. Obtain authorization and a
recovery snapshot. Keep SELinux, firewalld, TLS and repository verification enabled.

## Internet release availability

The [alm46-internet-rc1 GitHub prerelease](https://github.com/grimange/almadial/releases/tag/alm46-internet-rc1)
is public. Its bootstrap and CLI were downloaded without authentication and matched
the approved hashes. This is a prerelease, not a stable production release.
Independent internet-host installation remains **NOT_RUN** because no authorized
external host has been supplied; human operator acceptance remains separate.

## Standard installation contract

Connect to a fresh supported host as root. If the cloud image requires its
configured account, connect with that account and enter `sudo -i` first.

```bash
ssh root@YOUR_SERVER

curl -fsSLO \
  https://github.com/grimange/almadial/releases/download/alm46-internet-rc1/install.sh

printf '%s  %s\n' \
  26a99a84302a498afab5bd6a1d4dfd27c7bc10aa87b4180e99ac710bf0dabba4 \
  install.sh | sha256sum -c -

bash install.sh

almadial --version
sha256sum /usr/local/sbin/almadial
almadial install
```

Require script verification to succeed and inspect it before execution. The installed
version must be `alm46-internet-rc1`; binary SHA-256 must be
`618f4b0441cde7f17d1a35382b581da5669e0efa67b04f28175bc0cb38c8e6b5`.
Binary source is `e543c56821e306d348fd5bc9930bf563cd403d0e`; the release tag
points to distribution checkpoint `00c629bc2f0181664bd3c51ef829939b29d09968`.
These identities describe different steps. The approved `release.json` retains
its preparation-time `NOT_PUBLISHED` field; GitHub's published release state and
the subsequent download verification establish actual publication.

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
