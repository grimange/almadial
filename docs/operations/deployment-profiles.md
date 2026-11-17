# Deployment profiles

Current fresh installations use the qualified AlmaLinux 10.3 package policy.
Historical 10.2 plans retain their original identity; this is not an OS upgrade
path. See the [release package policy](../compatibility/alm34-release-package-policy.md).

Both profiles use the same immutable plan, exclusive execution lock, durable
journal and postcondition checks. Select the profile in `almadial install`.
Remote is the default selection; it is not a production readiness assertion.

## Product deployment contract

**Standard remote deployment is the product acceptance target.** A supported
AlmaLinux 10.3 x86_64 physical server, cloud VM or VPS must obtain the published,
versioned, integrity-verified CLI and every pinned source through ordinary
internet access, without developer-LAN routes, Multipass, Go, Git, manual source
transfers or mandatory distribution overrides. Public source defaults and optional
private mirrors have separate roles. A private release test cannot satisfy the
external distribution gate. This contract supersedes earlier laboratory-specific
distribution assumptions.

Remote requires a configured FQDN, matching trusted application TLS, direct
HTTPS/WSS, reachable UDP media, native VICIdial runtime, durable recovery and
healthy repeat installation. Initial supported topology uses a directly attached
reachable IPv4; arbitrary NAT/firewall topologies are not implied.

The isolated laboratory profile describes network exposure, independently of
authorization policy. Standard host-bound exact-plan approval works in either
profile without task/snapshot metadata. Historical managed plans retain their
original strict validation, bytes, digests and resume contract; managed governance
is selected explicitly through that existing request representation.

AI-worker infrastructure authorization is a separate repository control: task
authority, host identity and snapshots govern developer mutations. End users
installing on their own supported servers do not need Kaneo evidence.

The laboratory profile is suitable for disposable development/regression environments.
Private addresses, isolated trust and private stores are acceptable there, but
must not define standard distribution or imply internet acceptance.

The remote native runtime and ordinary campaign workflow passed
[independent QA](../reviews/alm46-native-runtime-independent-qa.md). That acceptance
is preserved. Public-host deployment is **NOT_RUN**: the owner confirmed that no
external test host is available. The `alm46-internet-rc1` prerelease is published
and its public bootstrap/CLI downloads are verified. External installation is separate;
see the [distribution correction](../implementation/internet-distribution-correction.md).

| Input or behavior | Remote single-server | Isolated lab |
|---|---|---|
| Hostname | Canonical FQDN resolving to the server IPv4 | `localhost` |
| Address | Directly attached reachable IPv4 on the selected interface | Private management IPv4 |
| Application certificate | Supplied matching certificate, key and trust chain | Generated lab certificate or supplied localhost certificate |
| Apache HTTPS | IPv4 wildcard port 443; authenticated upstream interfaces | Loopback port 443 |
| Browser signaling | Same-origin `/ws`, Apache to loopback Asterisk WS | Loopback Asterisk WSS port 8089 |
| Media | Selected even-start UDP range; endpoint bound to the configured address; ICE and DTLS-SRTP | Separate controlled functional-media qualification |
| Inbound firewall | Existing SSH port, 443 TCP, exact UDP media range, essential ICMP | Existing SSH port and essential ICMP |
| Internal services | Database, AMI, FastAGI and Asterisk HTTP/WS stay loopback | Loopback |
| VICIdial workers | Native updater, listener, sender and autodial supervised by systemd | Existing deferred policy |
| Maintenance | Native keepalive configuration generation and upstream cron jobs | Existing inactive schedule |

Remote requires working client/server DNS and certificate trust before use.
The first topology is a server with one directly attached address, not a
public-to-private NAT mapping. A restrictive client network may require TURN;
TURN infrastructure and carrier/PSTN qualification are outside this profile.
A supplied certificate is required; AlmaDial does not provision ACME.

The remote synthetic phone uses the existing protected credential and a
synthetic-only conference route at extension `9271`. It supports acceptance
calling and recording without a carrier. It is not a replacement for VICIdial
agent/campaign/phone administration. A successful synthetic call does not by
itself qualify a campaign workflow.

The remote installer provisions a separate authenticated loopback AMI account
for native operations; the diagnostic account remains read-only. Systemd owns
the four workers and FastAGI. Cron runs native maintenance, configuration
generation, hopper maintenance and WAV recording archival. Do not start a second
keepalive supervisor or manually enable the lab's deferred units. The installer
reconciles the native contract through the existing lock and durable journals.

Create ordinary phones, users and campaigns through VICIdial administration.
For a browser phone, select the installed `ALMADIAL_WEBRTC` configuration
template and the `default` context. This template supplies the configured remote
media address, TLS and WSS transport. Test accounts and campaigns used in
qualification are disposable fixtures, not general installation defaults.

For an isolated lab, optional SSH TCP forwarding can expose local HTTPS/WSS to
an administrator:

```bash
ssh -N -L 8443:127.0.0.1:443 -L 8089:127.0.0.1:8089 root@YOUR_SERVER
```

Open `https://localhost:8443/vicidial/admin.php`, `/agc/vicidial.php` or
`/almadial-phone/` after trusting the public application CA. This forwarding does
not transport WebRTC UDP media.

## Diagnostics and correction

Use `almadial doctor --plan /var/lib/almadial-inputs/plan.json --ca-file
/etc/pki/tls/almadial-s08/current/trust.pem --manifest
/var/lib/almadial-inputs/stack.lock.yaml` as one command. The explicit plan selects
the intended hostname, listeners and firewall. Diagnostics are read-only.

For remote browser failures, check DNS and certificate SAN/trust first, then
Apache `/ws` proxy and Asterisk registration. For no audio, inspect the selected
ICE candidate pair, DTLS state, packet counters and the exact UDP firewall range.
An answered channel or silent WAV is not a media pass. Keep sensitive browser
profiles, SIP authentication and recordings out of public reports.

Preserve saved inputs and journals. Rerun `almadial resume` to reconcile the
approved plan; do not edit a plan or replace hashes to bypass a failed check.


## Management discovery and operator recovery

The installer observes active assigned IPv4/interface pairs and effective SSH
configuration plus actual listeners before standard provisioning. A single SSH
port, including 2222, is preserved in the immutable plan and firewalld policy.
It does not change sshd configuration. Multiple configured ports, mismatched
listeners, an unassigned selected address or DNS-to-private NAT mapping fail
before provisioning with an explanation. Configured addresses do not establish
public reachability. Direct-address remote mode remains the supported topology.

Use `almadial install` or `almadial resume` after correcting an interruption.
Review and approve the same saved plan. Locks, stage journals and fresh
postconditions determine safe progress; completion receipts alone are not health
proof. Backups/snapshots are recommended, without an installer assertion that
recovery has been tested. Full system rollback is not guaranteed. Foreign file
changes, changed plan-bound TLS material or uncertain database operations require
operator reconciliation rather than blind retry. See the
[installation guide](almadial-installation.md) for resource thresholds and commands.
