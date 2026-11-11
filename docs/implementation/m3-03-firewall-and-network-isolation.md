# M3-03 — Host firewall and isolated network surface

Developer qualification **PASS — ready for independent AGY QA**. The corrected
S09 policy is active, independent IPv4 and link-local IPv6 probes pass, and one
protected firewalld reload preserves the effective policy and SSH access. Fresh
NO_OP and copied-journal reconciliation verify that completed effects are not
replayed. Implementation remains uncommitted; this is not independent acceptance.

## Contract and ownership

Canonical task `b0o51bkbrb2ylh57lq5t6wzv` requires owned firewalld policy,
management preservation, SELinux Enforcing, local internal services and verified
advertised topology. The operator selected an **isolated** contract: preserve
SSH and every existing application binding; no external HTTPS, WSS, SIP or RTP
grants; install only reviewed signed firewalld dependencies. No external
address, NAT/media path or public A/AAAA identity is advertised. Public routing,
cloud forwarding, DNS publication and media verification remain deferred.
Unsupported public topology inputs are rejected. This does not establish
external telephony readiness.

The current task names this stage **S09**. The original draft plan combines
TLS/network in S08 and calls WebRTC S09. This implementation follows the explicit
current task numbering without rewriting that historical plan or the unresolved
M2 scope-governance decision.

S09 owns only `/etc/firewalld/policies/almadial-s09.xml`, plus a transient native
systemd recovery timer during activation. It neither assigns interfaces to zones
nor edits vendor zones, SSH, application bindings, routes or NAT configuration.
A future optional access portal must obtain its own explicitly authorized policy
ownership; no dynamic grants or portal interfaces are implemented here.

## Actual baseline and exposure

Authorized disposable VM: `almadial-alm16-rpm`; machine ID
`0fffa9a4683f35b696f0b6ee7330734b`; recovery snapshot
`alm25-pre-firewall-20261111`. The operator authorized stopping this VM for the
snapshot, starting it again, and one later controlled firewall reload. No reboot
or application reconfiguration was authorized or performed.

The host has `eth0`, private IPv4 management addressing, IPv6 link-local addressing
and loopback. SSH listens on TCP 22 for both address families. Before S09,
firewalld/nftables were absent; after the approved package transaction,
firewalld was enabled but inactive and the effective nftables ruleset was empty.

| Service | Actual binding | S09 inbound treatment | State |
|---|---|---|---|
| SSH | IPv4/IPv6 wildcard TCP 22 | Preserve TCP 22 | Active management |
| Apache HTTP/HTTPS | `127.0.0.1:80`, `127.0.0.1:443` | No external grant | Active |
| Asterisk HTTP/HTTPS/WSS | `127.0.0.1:8088`, `127.0.0.1:8089` | No external grant | Active local TLS |
| MariaDB | `127.0.0.1:3306` | No external grant | Active internal |
| AMI | `127.0.0.1:5038` | No external grant | Active internal |
| FastAGI | `127.0.0.1:4577` | No external grant | Supervisor and five workers |
| Asterisk UDP sockets | Wildcard dynamic IPv4/IPv6 ports | No external grant | Observed; no PJSIP transport |
| SIP/RTP | No authorized published transport | No grant | Deferred |
| Updater/listener and application cron | Disabled / zero active jobs | No change | Deferred |

A loopback binding, a firewall permission and remote reachability are assessed
separately. No external reachability claim follows from a local TLS handshake.

## Implementation

`NetworkRequest` accepts only the isolated topology, an existing non-loopback
interface, private IPv4 management address and explicit SSH port.
`NetworkInputs` retains the immutable accepted S08 parent. `NetworkNative`
reuses all existing foundation postcondition checkers before assessing firewalld.
`almadial-network` provides read-only `inspect`, pure `plan`, and authorized
`lab-run` / `lab-resume`, with the existing pinned source/manifest arguments.

The deterministic policy is:

```xml
<?xml version="1.0" encoding="utf-8"?>
<!-- AlmaDial S09: isolated host; SSH and essential IP control traffic only. -->
<policy target="DROP" priority="-20000">
  <short>AlmaDial isolated host</short>
  <ingress-zone name="ANY"/>
  <egress-zone name="HOST"/>
  <port port="22" protocol="tcp"/>
  <protocol value="icmp"/>
  <protocol value="ipv6-icmp"/>
</policy>
```

Firewalld policy priority places this before zone rules and the vendor
`allow-host-ipv6` policy at `-15000`. ANY ingress excludes locally originated
HOST traffic. Existing outbound/established handling remains firewalld-owned.
The implementation verifies runtime/permanent policy agreement, exact managed
XML, active/enabled state, SELinux file labeling and generated inet nftables
rules. It rejects conflicting owned content, preceding foreign policies,
foreign runtime/permanent divergence and foreign nftables hooks requiring review.
It preserves such conflicts without attempting automatic replacement.

A single action, `network.00.activate`, uses the accepted exclusive execution
lock, durable action intent, bounded runner and journal outcome. Before mutation
it checks the fresh host/parent/foreign-policy identity, actual SSH configuration
and listener, management address, loopback application listeners, SELinux and
native `firewall-offline-cmd --check-config`. The owned file is published using
an atomic fsynced replacement and `restorecon`.

A native 120-second transient systemd timer protects interrupted activation.
Failure removes/restores only the owned file and restores the prior firewall
active/enabled state. Successful native verification cancels that timer;
independent fresh management connectivity remains a separate laboratory check.
No custom firewall daemon or second scheduling engine is introduced.

## Reviewed dependencies and integration correction

The existing S01 executor installed exactly six BaseOS packages, using a reviewed
transaction, signature verification, durable intent and exact post-install delta:

| Package | EVR | Architecture |
|---|---|---|
| firewalld | `0:2.4.3-4.el10_2` | noarch |
| firewalld-filesystem | `0:2.4.3-4.el10_2` | noarch |
| libnftnl | `0:1.3.0-3.el10` | x86_64 |
| nftables | `1:1.1.5-6.el10_2` | x86_64 |
| python3-firewall | `0:2.4.3-4.el10_2` | noarch |
| python3-nftables | `1:1.1.5-6.el10_2` | x86_64 |

There were no removals or upgrades. The inventory increased from 725 to 731 RPMs.
Repository definitions and signing policy were preserved. The existing RPM CLI
requires its historical `ALM-16/disposable-lab` authorization scope; its reuse was
limited to this ALM-25 transaction and did not change the accepted ALM-16 task.

A confirmed S01 request-construction defect omitted approved additional packages
when re-solving the reviewed transaction. `nativeCommand` now includes those
exact reviewed requests; transaction identity, signature and delta checks remain
in force. A focused regression covers the correction.

Existing S01–S08 receipts bind the original full RPM inventory. A closed S09
handoff verifies the physical inventory against the exact six pinned additions,
repository/key policy, signatures and committed DNF history before retaining
that accepted foundation receipt identity for upstream checks. It does not edit
receipts or weaken ordinary earlier-stage inspection. Extra/missing packages,
wrong signers, unreviewed hashes, unfinished history and host mismatch fail closed.

## Earlier qualification history

The following records describe earlier checkpoints, superseded by the final live
qualification below. Their failed attempts and rollback evidence are retained.

Two activation attempts failed and restored the prior enabled/inactive firewall
state. Fresh SSH inspection succeeded afterward; nftables returned to its empty
pre-activation state. The journal records failure rather than completion.
Read-only diagnosis identified a policy-order conflict: the first priority
`-500` fell behind vendor `allow-host-ipv6` at `-15000`, which the preceding-policy
check correctly rejected. The correction uses `-20000`; normalization also
accounts for firewalld's runtime `(active)` display marker without discarding
configuration fields. A fixture verifies both vendor-policy preservation and
rejection of a genuinely preceding conflicting policy.

Automatic approval review initially rejected another live activation. The
operator subsequently explicitly authorized one corrected activation, one
conditional reload, and exact-file historical restoration. Preflight showed
QEMU serial attached to a null device and no independent Multipass console;
the operator explicitly approved the native 120-second rollback timer plus
verified hypervisor snapshot as the recovery boundary instead. No independent
console or snapshot restoration was tested.

The continuation invoked `almadial-network lab-run` once through
`qualified-run`, using the reviewed `network.00.activate` plan and priority
`-20000`. It returned exit 1, journaled the action as failed, and rolled back.
Fresh SSH and read-only inspection verified enabled/inactive firewalld, empty
nftables rules, absent owned policy, no armed recovery timer, and exactly the
same S09 snapshot/foreign-configuration identities as before activation.
Offline validation of the exact generated policy in a temporary configuration
copy passed.

The retained subprocess result contains only the generic `nonzero_exit` error;
it does not establish which inner native postcondition failed. Read-only
firewalld logs show start followed by stop, without that detail. Investigation
confirmed a diagnostic defect: `NetworkNative.adapter` returned the bounded
runner error before checking its captured stderr. The smallest correction now
retains the native diagnostic on nonzero exit, with a focused regression test.
At the end of that continuation the diagnostic correction had **not** been
deployed or exercised in another live activation, and the cause remained
unresolved. The subsequent read-only investigation below identifies a concrete
post-start verification defect without another activation.

The conditional reload, independent allowed/denied IPv4/IPv6 tests, actual-host
NO_OP and copied-journal reconciliation remain NOT_RUN because activation did
not qualify. Fresh planning still requires the activation action. A successful
fixture is not evidence of an installed filtering policy.

Deterministic fixtures pass completed-effect journal reconciliation without
replay, repeat-plan NO_OP, stale-state and authorization rejection, lock
contention, invalid topology, exact additive dependency verification, foreign
runtime/nftables conflict rejection and bounded activation rollback. Fixture
results are not substituted for the outstanding host evidence.

Fresh preservation checks after recovery confirm schema 1748, accepted catalog
digest `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`,
and the two legacy phones `callin` / `gs102` across 107 columns, digest
`bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`.
SELinux remains Enforcing. The consolidated read-only comparison also confirms unchanged repositories,
credentials, Asterisk configuration/modules, recording trees, application data
and service identities. Both trusted loopback TLS endpoints serve the accepted
leaf fingerprint. FastAGI is active; updater/listener are disabled and the managed
application cron has zero active jobs.

## Evidence incident

An old S08 qualification helper was mistakenly invoked while diagnosing the RPM
handoff. It overwrote historical
`m3-02/20261111T003910Z-alm24/noop-inspect.command.json` with the new failed
inspection and emptied the disposable VM's corresponding temporary output.
No runtime configuration changed through that invocation. The original ledger
was not changed. The overwritten failure is copied into this task's evidence.

A reconstruction from preserved S08 inspection/plan records matches the original
sealed file SHA-256 exactly:
`d4316114a6e2cd109051bc89e299d5838cac4244c52aac6f3cc34c61e5cb8499`.
After separate explicit operator authorization, only that exact file was
restored from the hash-matching candidate. The original historical ledger
verified successfully and its bytes remained unchanged. A before/after inventory
confirmed no other historical file changed. The overwritten failure remains
preserved in both S09 evidence and the continuation supplement. The offending
local `/tmp/alm24-qualify.py` helper now uses exclusive-create semantics for
guest and host output files; it was not rerun. No S08 qualification was repeated.

## Final verification summary

| Check | Final result |
|---|---|
| Go 1.27.3 vet / test / race | PASS: 1,392 passing events, 223 top-level tests in each test run |
| Go module verify / tidy consistency / formatting | PASS |
| Python tests | PASS: 203 tests, including 11 focused native S09 tests |
| Scaffold / diff whitespace | PASS |
| CLI smoke / terminal journeys | PASS: 17 / 11 |
| Reproducible Linux amd64 builds | PASS: ten targets, two clean source copies |
| Native policy validation / live activation | PASS |
| Independent IPv4 / link-local IPv6 traffic | PASS: SSH allowed; all tested protected ports denied |
| Protected firewall reload / persistence | PASS |
| Actual-host fresh NO_OP / copied-journal reconciliation | PASS; zero runner effects |
| Hosted CI | NOT_RUN |

Earlier protected evidence remains unchanged at
`~/almadial-lab/evidence/m3-03/20261111T020839Z-alm25/` and the independently sealed
continuation and diagnosis supplements. Historical S08 restoration is complete
and was not revisited. The final qualification supplement is listed below.

## Read-only root-cause investigation — 2026-11-11

This historical checkpoint preceded the final authorized qualification below.
Its statements about inactive state and outstanding work apply to that checkpoint.

The concrete defect is in `network_native.py:normalized`, used by
`foreign_runtime` during `inspect` immediately after firewalld starts. Installed
firewalld `2.4.3-4.el10_2` always includes the default zone in
`FirewallZone.get_active_zones()`. Its D-Bus `getActiveZones` preserves that entry,
and `FirewallCommand.print_zone_policy_info` renders the same zone as:

```text
runtime:   public (default, active)
permanent: public (default)
```

The previous regular expression removed only standalone ` (active)`. It left
` (default, active)` unchanged, so the first runtime/permanent comparison raised
`foreign runtime-only zone state` even with identical settings. The default-zone
inclusion makes this comparison fail independently of NetworkManager timing or
interface assignment. The existing fixture used `public (active)` and missed
the installed version's combined annotation. The priority correction to
`-20000` does not resolve this separate verifier defect.

Evidence combines the actual installed source and a read-only reproduction using
its default-zone selector and formatter, with no daemon or D-Bus connection.
Feeding those outputs into the previous S09 verifier reproduces the exact named
rejection. Feeding them into the corrected verifier passes. A focused fixture
also confirms that changing a service or retaining a different header attribute
still fails comparison. Only the runtime `active` annotation is ignored; zone
settings, other attributes and foreign policy safeguards remain intact.

The historical activation exited 1 in `network.00.activate`. Native records
show firewalld started and was then stopped cleanly during rollback: latest
`ExecMainStatus=0`, `Result=success`, 463 ms service duration, empty
`/var/log/firewalld`, and no relevant kernel nftables/firewalld/AVC error. Thus
there is no recorded daemon startup error. The old adapter discarded the inner
stderr; its original text cannot be recovered. The installed-source reproduction
establishes a mandatory failing verification check, rather than recovering that
missing historical stderr or claiming an observed successful live activation.

The activation path is CLI `lab-run` → immutable S09 graph → execution lock and
journal intent → bounded Python adapter → rollback timer → atomic owned policy
write and offline validation → `systemctl enable --now firewalld` → fresh
`inspect`/`contracts`/`foreign_runtime` → effective-policy checks → timer disarm.
Any exception restores only the owned policy and prior service state before
returning failure. The previous failure prevented reload, traffic qualification,
actual-host NO_OP and copied-journal verification.

Diagnostics now preserve the bounded runner's native result, both redacted output
streams and typed execution failure through the network runner. The journal
retains action/category/exit status; diagnostics remain in the trace, rather than
being copied into the durable journal. Native Python command failures identify
the operation and exit code, with bounded stdout/stderr. Deterministic tests
verify distinct streams and status 7 through the adapter, S09 trace and failed
journal action. No execution-framework or policy architecture change was made.

Fresh read-only comparison against the previous post-failure baseline confirms
unchanged repositories, credentials, schema/catalog, both 107-column phone
records, recordings, Asterisk configuration/modules, application data and
service identities. There are 731 authorized RPMs; SELinux remains Enforcing.
Both loopback TLS endpoints retain the accepted trusted certificate. FastAGI
has one parent and five children; updater/listener remain disabled, application
cron has zero active jobs, and PJSIP reports no transports. SSH access succeeds.
Firewalld remains enabled/inactive, the owned policy absent, and nftables empty.
No firewall activation, reload, restart or effective-policy mutation occurred.

Investigation evidence is separate and append-only:
`~/almadial-lab/evidence/m3-03/20261111T060312Z-alm25-diagnosis/`.
It includes installed-source evidence, before/after formatter reproduction,
activation-path findings, native logs, preservation comparisons, focused tests
and final regressions, with its own `SHA256SUMS`. The new source is uncommitted
and has not been deployed. ALM-25 stays In Progress with Codex ownership.

Next requires a separately authorized single activation of this corrected source,
fresh plan/identity checks, the verified snapshot and operator-approved recovery
boundary. Stop on failure and retain the now-preserved diagnostics. A subsequent
reload, independent allowed/denied traffic tests and actual-host NO_OP/recovery
remain outstanding live qualification; this investigation grants no new mutation
authority and does not advance the task to QA Review.

## Final live qualification — 2026-11-11

The operator's standing ALM-25 authorization covered bounded corrected retries,
activation, reload, network tests and recovery. Snapshot
`alm25-pre-firewall-20261111` was verified again. There is no independent VM
console; the explicitly approved recovery boundary is the native 120-second
systemd rollback timer plus the hypervisor snapshot. No snapshot restoration or
VM reboot occurred.

Further native diagnostic failures exposed three precise compatibility defects.
Each failed transaction rolled back and was checked before a corrected retry:

- NetworkManager supplies runtime `interfaces: eth0` while the permanent zone
  contains no interface assignment. The verifier now accepts this addition only
  after checking the active NetworkManager connection UUID and its effective
  zone (including the default zone when the profile leaves it blank). Other zone
  differences still fail. S09 does not modify the connection or zone assignment.
- Installed firewalld 2.4.3 supports policy `--get-priority` and `--get-target`
  only with `--permanent`. Those scalar queries now use the supported form;
  complete runtime/permanent policy comparison remains mandatory.
- The vendor `gateway-dmz-to-HOST` runtime header includes `(disabled)` while
  the permanent header omits it. Both contain `disable: yes`. Normalization
  removes this header annotation only when that semantic field is present;
  meaningful policy differences remain conflicts.

Bounded mismatch diagnostics identify the affected zone/policy and differing
lines. Native status and stdout/stderr remain preserved by the corrected Go
adapter. Eleven focused native fixtures cover these comparisons, strict mismatch
rejection, effective policy, and rollback behavior.

The final deployed executable SHA-256 is
`7d88fde2af116574007e435e9e504664432506b35ba9db6bedccaf04e40450e9`.
`ready-run` completed `network.00.activate` successfully with one runner effect.
Firewalld is enabled and active, the owned XML matches the contract above,
runtime/permanent comparisons pass, and the inet nftables policy is effective.
The activation timer was disarmed after native verification.

Independent probes from the hypervisor bridge reached SSH TCP 22 through both
the VM's private IPv4 address and IPv6 link-local address. TCP 443, 8089, 3306,
5038, 4577 and an unallocated control port 18025 timed out in both families.
These are actual bridge-to-guest traffic observations, separate from local socket
and ruleset inspection. No global IPv6, public Internet, NAT/media or UDP/RTP
traffic path is claimed. Effective inet rules provide no SIP/RTP or external
application grant, and Asterisk reports no PJSIP transports.

One controlled `firewall-cmd --reload` ran with a separately armed native
120-second recovery timer. After reload, independent SSH and denied-port probes,
trusted loopback TLS checks and native effective-policy checks passed; the timer
was then disarmed. The daemon invocation remained unchanged. Foreign files and
the owned XML were unchanged, and normalized nftables rules remained equivalent.

Fresh inspection and planning returned an empty graph `[]`. Repeat execution
returned `no_op: true` with zero runner traces. A copied successful journal was
changed to a valid interrupted running record and resumed against fresh host
postconditions: it reconciled to success without replay. The canonical journal
hash remained unchanged. An initial malformed copied fixture (running state with
exit code 0 instead of -1) was rejected before execution and retained separately.
Policy hash/mtime, foreign configuration and exact post-reload nftables rules
remained unchanged through NO_OP and reconciliation.

Before/after comparisons preserve the 731-RPM inventory, repository definitions,
credentials, identities, recordings, Asterisk configuration and required modules,
VICIdial application data and configuration. Schema 1748 and catalog digest
`10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`
remain unchanged. Both legacy phones `callin` and `gs102` retain the 107-column
digest `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`.
Apache HTTPS and Asterisk HTTPS/WSS pass CA trust and localhost identity checks
with the accepted leaf certificate unchanged. FastAGI has one parent and five
workers running as asterisk. Updater/listener remain disabled, application cron
has zero active jobs, and SELinux remains Enforcing. No database, certificate,
application-binding or dialing change occurred.

Final protected evidence:
`~/almadial-lab/evidence/m3-03/20261111T081118Z-alm25-finish/`.
It contains failed retry diagnostics, successful activation, both network probe
rounds, reload protection, preservation comparisons, fresh NO_OP, copied-journal
recovery, regression outputs and reproducible build hashes. Its `SHA256SUMS`
ledger digest is recorded in the Kaneo qualification comment. ALM-26 remains
unstarted; independent AGY review is the next step.
