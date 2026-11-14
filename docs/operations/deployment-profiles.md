# Deployment profiles

Both profiles use the same immutable plan, exclusive execution lock, durable
journal and postcondition checks. Select the profile in `almadial install`.
Remote is the default selection; it is not a production readiness assertion.

**Current gate:** trusted direct HTTPS/WSS and the synthetic call, two-way media
and recording path passed controlled off-host testing. Normal VICIdial agent
operation has not passed: updater/listener services are deliberately disabled
by the existing lab runtime policy and there is no ordinary campaign. Remote
Doctor now reports this as a blocking P0. Keep the profile on authorized
development hosts until a corrected runtime and agent workflow are qualified.
No public test host was available; public-host acceptance is `NOT_RUN`.

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

Do not manually enable the deferred services, broaden the read-only AMI account,
or use temporary functional-test overlays as an installed-runtime repair. The
remaining correction must be represented in the same source-bound immutable
plan and durable provisioning stages.

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
