# M4-01 — Unified clean-host installation and release qualification

Developer qualification is **complete and ready for independent AGY review**. Starting Git commit:
`1c20d8b9e641c068c059e6edc929eff01390b329`. Canonical task:
ALM-29 (`xu7wxvch94ii2iw38hpb8d58`). Independent AGY review and maintainer
release acceptance remain separate gates; no release is published here.

## Installation contract

The `almadial` CLI now composes the accepted stage planners and executors. It
holds the existing exclusive execution lock for the workflow and retains each
stage's existing action journal. Immutable, protected `*.input.json` files retain
the approved plan, stage inputs and accepted observations so resume reconstructs
the same identities. They do not replace action intents or journal recovery.

Stages run in order: S01 packages/repositories; S02 identities/storage/SELinux;
S03 MariaDB/schema; S04 Asterisk/modules; S05 VICIdial; S06 FastAGI/deferred
scheduling; S07 Apache/PHP; S08 supplied-certificate TLS; S09 isolated firewall;
S10 VICIphone/synthetic PJSIP. S11 is the existing explicitly invoked synthetic
functional acceptance harness, rather than permanent dialing configuration.
S12 read-only diagnostics run after provisioning. The installation report marks
browser/media, reboot and independent QA qualification as separate requirements.

Each downstream planner consumes verified upstream state. A final recursive
S01–S10 inspection must produce an empty effective graph; saved success alone
cannot establish installed health. Read-only installed-state assessment uses the
same inspectors and does not acquire a mutation lock or publish state.

## Supported qualification profile and inputs

This qualification uses AlmaLinux 10.2 x86_64, four vCPUs, 8 GiB RAM and an
80 GiB disk, Enforcing SELinux, synchronized time, approved AlmaLinux package
repositories and a private local/LAN network. These are laboratory qualification
inputs, not production capacity sizing. The manifest remains CANDIDATE.

The JSON request maps directly to `provision.InstallRequest`:

| Input | Required content |
|---|---|
| `Config` | Existing typed schema-version-1 configuration; host bind address, localhost identity, SIP/RTP/WSS parameters and protected credential references |
| `Requirements` | Explicit CPU, RAM, available-storage and CPU-flag requirements |
| `Recording` | Actual recording mount/device/filesystem and reserve requirements |
| `TLS` | `provided` mode, localhost certificate, matching private key and intended private-CA trust anchor |
| `Network` | `isolated` mode, actual interface/management address/SSH port |
| `Phone` | Accepted synthetic `alm_s10_9061`, localhost, `wss://localhost:8089/ws` |
| `Sources` | Local pinned VICIdial, Net::Telnet, Asterisk archive/signature/keyring, PJProject, VICIphone and scratch paths |
| `Authorization` | Task/host identity, recovery snapshot reference and operator authority digests, candidate and management-recovery acknowledgements |

Artifact identities and Asterisk signature verification use existing accepted
loaders. Supply the files identified by `manifests/stack.lock.yaml`; there is no
unreviewed installer download or hidden manual stage prerequisite. Certificate
issuance is an explicit operator input. Supplied private-CA material is validated
for identity, trust, validity and key pairing before any installation mutation.
Public ACME/DNS, carriers, public SIP and NAT/STUN/TURN are not prerequisites.
Never place passwords or private-key contents in the request or command line.

## Operator workflow

Build with the pinned Go toolchain using the repository build script. Stage the
binary, manifest, verified archives and protected certificate inputs on the
intended clean host. Protect the request and output directory with mode 0700;
use mode 0600 for request/plan files. A restrictive operator umask is supported.

```bash
sudo almadial plan --request /protected/request.json \
  --manifest /protected/stack.lock.yaml > /protected/plan.json
# Review all stage responsibilities and the SHA-256 printed on stderr.
sudo almadial install --plan /protected/plan.json \
  --manifest /protected/stack.lock.yaml --approve-sha256 REVIEWED_SHA256
```

Planning only inspects. Installation reports current stage/action events to
stderr and a non-secret JSON result to stdout. The result contains plan/build
identity, completed stages, diagnostics, endpoints and remaining qualification.
The endpoints are loopback HTTPS/WSS; use an authorized tunnel for a remote
browser, with the intended CA installed in its trust store. No public listener
or external application firewall grant is introduced.

On failure preserve the reported stage journal and run the same workflow with
`resume` instead of `install`, using the same plan and approval digest. Existing
stage-specific observers decide whether effects can reconcile; an ambiguous
DNF transaction or partial schema is not blindly replayed. Do not remove state
to force a retry. State lives under `/var/lib/almadial-install`.

`--pause-after S03` is an optional verified checkpoint for controlled recovery
qualification. It stops after durable database acceptance, never within SQL or a
package transaction. `resume` continues the same approved plan.

```bash
sudo almadial plan --plan /protected/plan.json \
  --manifest /protected/stack.lock.yaml
sudo almadial install --plan /protected/plan.json \
  --manifest /protected/stack.lock.yaml --approve-sha256 REVIEWED_SHA256
almadial status --json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem \
  --manifest /protected/stack.lock.yaml
almadial doctor --ca-file /etc/pki/tls/almadial-s08/current/trust.pem \
  --manifest /protected/stack.lock.yaml
```

The installed-state plan must report `actions: []` and `no_op: true`. Repeat
execution freshly verifies the host without replaying completed effects. Short
laboratory certificates can correctly cause diagnostics WARN/exit 10. Without
explicit CA/manifest options, their checks report NOT_CONFIGURED rather than
assuming trust or guessing the source manifest.

## Confirmed integration corrections

1. A large S01 resolved transaction exceeded Linux's per-argument limit before
   Python could start. The native runner now sends large JSON requests through
   bounded stdin while retaining the immutable approved action. A real rootless
   subprocess test exercises a 160 KiB request.
2. Operator umask 077 filtered Asterisk runtime-directory creation to mode 0700,
   preventing its service account from loading runtime resources. New owned
   directories now receive explicit mode 0755; existing foreign metadata is
   rejected. The equivalent newly owned phone asset directories are explicit
   as well. A focused restrictive-umask fixture checks the Asterisk behavior.

Earlier failed disposable attempts are preserved as evidence. No accepted M3
host or historical snapshot was modified. Immediate S03/S04 service checks on
the latest attempts failed despite later healthy fresh inspection; their exact
transition cause remains unproven. Existing journal reconciliation continued
from actual state without SQL reimport or rebuilding completed Asterisk work.

## Actual qualification results

Protected evidence root:
`~/almadial-lab/evidence/m4-01/20261112T013822Z-alm29/`.

| Requirement | Actual result |
|---|---|
| First fresh installation | `almadial-alm29-clean-d`, machine `4c7b492940923838a0700adba65ed3eb`; image baseline 380 RPMs; final 731; S03 verified pause and resume passed |
| Second fresh installation | `almadial-alm29-clean-e`, machine `ebf868d3a0853b4294cd85e589a12f5c`; independent pristine image; same manifest; final 731 RPMs |
| Application browser smoke | On both hosts, real Chromium authenticated as the bootstrap administrator, loaded administrator navigation, agent Phone Login and Timeclock; all four routes returned 200 without trust bypass |
| Local/LAN telephony | On both hosts, two synthetic browser sessions registered over trusted WSS, answered calls, joined ConfBridge 9600271, exchanged meaningful audio in both directions and hung up; ICE/DTLS-SRTP connected |
| Recording | Actual MixMonitor PCM WAV files, meaningful tone sequences, timestamps and existing `call_log` rows correlated with the PJSIP channel unique IDs |
| Reboot | Both guarded reboots passed; boot IDs changed; services, 731 RPMs, SELinux Enforcing, five FastAGI workers and protected configuration hashes preserved; zero stranded channels or temporary media grants |
| Fresh assessment and repeat | Both hosts: `actions: []`, `no_op: true`, repeat exit 0; managed file/journal hashes, service invocation identities and effective nftables policy unchanged |
| Recovery | Actual S03 boundary pause/resume; completed-state reconciliation on copied interrupted media journal: `runner_mutations: 0`, `reconciled: true`; canonical journal untouched |
| Diagnostics | Status and doctor exit 10 on both hosts for short-lived lab certificates; mandatory checks have no blocking failures |
| Regression | Go test and race: each 1,432 passing test/subtest events, 21 passing packages; vet, module verification/tidy, formatting and scaffold passed; Python 222 tests; 14 executable CLI smoke cases; 11 terminal journeys; 12 Linux amd64 targets built twice with matching hashes |

The two VMs use the same AlmaLinux 10.2 image SHA-256
`cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`
and manifest SHA-256
`80940fac407ca7ee0041378b2e1e8b4ab4d95aa95749b503d5df7d4f356009fa`.
Each has a verified `alm29-pristine` snapshot. Installation used binary
`e18cd660ba70ddcd74fd187296d79d49c13399232b36c9aa44ec6c3c0cf59847`;
the final assessment/report correction binary is
`30523badad6a726a11c512ca3e192914a7ad21562f398ca749164e19fffa3d45`.
No stage installation was manually coordinated: documented install/resume
commands drove the accepted planners and journals. Final recursive inspection
and repeat execution used the corrected final binary on both hosts.

D's first S03 service postcondition and E's first S04 service postcondition
failed before later fresh inspection found healthy state. The existing resume
workflow reconciled completed effects; it did not reimport SQL, rebuild accepted
Asterisk output or require manual configuration repair. Those failures and the
earlier unsuccessful A/B/C attempts remain recorded. Their presence must not be
represented as uninterrupted first-attempt success. The controlled S03 checkpoint
was a deliberate durable stage boundary rather than killing a database operation.

Audio analysis verified both four-tone sequences independently on each host.
E's mixed recording scored 0.993 content agreement. D's fixed-offset mixed-track
checker scored 0.886 (the other participant scored 0.831), below its 0.90
threshold; those results are retained. Independently aligned native receive and
transmit recordings scored 1.000 and 0.995 agreement, with envelope correlations
0.996 and 0.982 and RMS 0.115 and 0.107. Both contain all four expected tones.
The separate-track check accommodates independent browser source/capture clocks
without lowering the content threshold. Mixed audio is valid and nonzero;
recorded bidirectional content is established from those actual native tracks.

D call unique IDs `1794450528.0` and `1794450529.3`, and E IDs
`1794450257.0` and `1794450259.3`, match real `call_log` rows and recording
filenames. No recording metadata was fabricated. Synthetic contexts and bounded
LAN media grants were removed afterward; the persistent S10 endpoint returns to
`alm-s10-no-calls`. Calling is an explicit acceptance fixture, not permanent
campaign activation. Browser administrator smoke does not claim complete agent
or campaign acceptance. Public/NAT/carrier profiles and hosted CI are NOT_RUN.

Both hosts preserve schema 1748, accepted Asterisk modules, pinned VICIdial,
private TLS generations, isolated firewalld policy, deferred updater/listener,
zero active application cron and recording storage. Machine identities,
credentials, certificate bytes and timestamps intentionally differ. The accepted
M3 VM and historical snapshots were not changed. Earlier failed disposable VMs
remain stopped for evidence; no destructive restore was used.

The default sandbox denied loopback sockets in one redundant race run; the same
suite passed under authorized local-test execution. No product test was weakened.
A first malformed copied-journal fixture (running state with exit code 0) was
correctly rejected; the valid interrupted fixture uses exit code -1 and reconciles.
Evidence retains both outcomes. Independent AGY reproduction and maintainer
release acceptance remain pending. Nothing was pushed, tagged or released.

## M3 report correction provenance

The M3 milestone report receives only factual errata: accepted Apache/PHP
versions, fixture-only TLS activation-failure rollback coverage, firewalld
ANY-to-HOST policy terminology, and native ConfBridge versus legacy MeetMe
terminology. Its original acceptance commit and results remain intact. Canonical
ALM-29 and the accepted M3 gate place unified clean-host orchestration in M4;
this work does not retroactively approve the pending M2 scope clarification.

## Evidence handoff

Protected evidence root:
`/home/ra/almadial-lab/evidence/m4-01/20261112T013822Z-alm29/`.
The sealed `qualification/SHA256SUMS` ledger has SHA-256
`0f34fab6f4bd7edd584a27fb7aa4ee84205274e0e30c55531777a8b191d9b850`.
The separate `diagnostics-final` supplement retains the final documentation and
explicit CA/manifest status and doctor results from both hosts. All four exit 10
with the expected certificate-expiration warning and no blocking failures.
One preceding collection wrapper timed out after 120 seconds; it is retained as
a failed collection attempt, followed by successful direct verification.
The sealed primary evidence was not overwritten. Private keys, credentials,
browser profiles and recording contents are excluded from these handoff bundles.
