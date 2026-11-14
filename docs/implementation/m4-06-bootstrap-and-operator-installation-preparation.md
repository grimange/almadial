# M4-06 — Durable acquisition and operator installation preparation

For the current accepted remote CLI and working private operator distribution,
see [accepted remote CLI distribution preparation](m4-06-accepted-remote-cli-distribution.md).
The release identities and hosts below are the historical acquisition qualification.

Developer qualification passed. The bootstrap is ready for independent human
operator execution; that acceptance and AGY assessment have not begun. This report supersedes the initial blocked
bootstrap report, preserved as `prior-bootstrap-report.md` in the new protected
evidence. Historical bootstrap evidence is unchanged.

## Authority and preservation

Baseline: `65b40fb79570c0767ed563d6e25be2c980c58985`, plus preserved uncommitted
ALM-34 changes. The owner explicitly resolved stale M0-only repository guidance
in favor of the accepted implementation and bounded ALM-34 instructions, while
retaining candidate-manifest and production restrictions. Kaneo read-back:
ALM-34 In Progress, Codex `aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`; ALM-29–33
Accepted, five incoming blocks unchanged, ALM-35 To Do/unassigned. Original
handoff `jztk9ihidvo0r2fq6woi131p` remains intact. Acquisition scope comment
`pmp5exy3oo7b0y68hambk9np` was posted and read back.

Only `almadial-alm34-bootstrap-dev` is mutable in this qualification. Its machine
ID is `d8e73788e2a9328a8655aa707be8816a`; the existing `alm34-pristine` snapshot
was restored after recording `alm34-before-acquisition`. AlmaLinux image SHA-256
is `cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`.
Accepted M3/ALM-29 and independent-operator VMs/evidence remain untouched.
No commit, push, tag, public release, self-acceptance or downstream implementation.

## Root cause, source identity and recovery

A fresh verified HTTPS request observed HTTP **404** for:
`https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip`.
This proves the acquisition location is unavailable. It does not establish that
SVN revision 4037 is unavailable or prove upstream's retention policy. Expiring
nightly retention is the suspected cause; relying on that one location was the
installer defect.

The accepted source is VICIdial **r4037**, database schema **1748**, archive root
`2026-11-05/`, ZIP SHA-256
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`,
and canonical exported-tree SHA-256
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
These identities have not changed. Manifest SHA-256 remains
`80940fac407ca7ee0041378b2e1e8b4ab4d95aa95749b503d5df7d4f356009fa`;
`stack.lock.yaml` remains candidate.

All five original archives were recovered read-only from accepted M0-08
`~/almadial-lab/evidence/m0-08/20261106T192640Z/inputs-final/`.
`scripts/prepare_compliance_source.py` checked each original archive SHA-256 and
its full independently accepted ALM-32 `source-files.json` inventory: **8,134
regular files**, including **2,511 VICIdial files**, all equal. No repin,
new nightly, reconstructed ZIP or SVN export was needed. SVN retrieval and
alternate source-tree acquisition are NOT_REQUIRED for this release; a future
reconstruction needs explicit tree-equivalence policy, not an old ZIP checksum.

## Durable acquisition and all five artifacts

Source identity, exact content identity, approved origins, cache identity and
verification policy are now separate in the existing artifact implementation.
`manifests/acquisition-origins.json` strictly binds reviewed private HTTPS store
locations to the exact unchanged manifest. The CLI embeds this policy with the
existing source manifest and signature inputs. Operators cannot enter arbitrary
mirrors; location changes require a reviewed release policy.

Priority is: reverified local content-addressed cache; approved private stores;
original pinned HTTPS upstream. Each archive must satisfy the same SHA-256,
layout and existing component/tree checks. Asterisk additionally requires the
accepted detached signature/keyring verification before installation. TLS uses
OS trust; redirects and credential-bearing URLs are rejected. Corrupt private
cache files are retained as `.rejected-*`, then replaced only by verified bytes.
All-origin failure reports the component, attempted origins and correction,
before any installation plan or provisioning action. Completed verified downloads
remain reusable. Saved-plan resume reacquires missing archives at the same bound
paths before decoding and approving the existing immutable plan.

Private store: `~/almadial-lab/distribution/alm34/sources`.
Approved HTTPS origin: **`https://10.160.90.1:8447/sources`**.
Layout: `sha256/<accepted archive SHA-256>/<component>.artifact`.
The offline maintainer tool `scripts/bootstrap/retain_sources.py` checks all
original archive hashes and all ALM-32 inventory entries before retaining any
object; existing objects must match and are never silently replaced. Private
directories are 0700, retained archive files 0400. No expiration or garbage
collector is installed. Original upstream notices remain inside intact archives.
Private retention/delivery is scoped to this authorized lab test; ALM-32 has not
granted unrestricted redistribution rights. No public third-party publication.

| Accepted input | Archive SHA-256 | Files | Durable private path / guest acquisition |
|---|---|---:|---|
| Asterisk 22.12.0 | `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` | 2,204 | Verified / acquired automatically |
| PJPROJECT 2.17, Asterisk third-party commit `52e2f32de1b11e40844bb007749a327862f75275` | `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` | 3,363 | Verified / acquired automatically |
| Net::Telnet 3.05 | `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` | 8 | Verified / acquired automatically |
| VICIdial r4037 / schema 1748 | `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` | 2,511 | Verified / acquired automatically; original nightly 404 |
| VICIphone `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` | `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` | 48 | Verified / acquired automatically |

The continuation's core/extra-sounds list differs from the accepted manifest.
The accepted build uses no-sounds configuration; neither archive is a pinned
input. No unpinned sounds were introduced or accepted PJPROJECT/Perl inputs
omitted. Signed RPM acquisition remains the existing DNF/NEVRA implementation;
no package mirror redesign.

## Bootstrap, binary and guided installation

Actual download: **`https://10.160.90.1:8447/install.sh`**. The private bridge
server serves CLI releases separately from retained source objects. Its bounded
static server has no uploads, execution or directory listing. Certificates and
private keys are outside the served root. The lab owner must keep it running,
retain referenced objects, back up the private store, and renew trust before
expiry. This is a verified private lab arrangement, not public hosting.

Public bootstrap CA PEM SHA-256:
`6e49e14cb26240689aa4ce67d1ff61bc8eac1c5dddf7fbdd73d0c6851fad2c0b`.
Validity ends **2026-12-13 18:51:39 UTC**, SAN `10.160.90.1`.
Developer CA enrollment was explicit; no TLS bypass or operator-VM modification.

| CLI provenance | Value |
|---|---|
| Version / commit | `m4-06-acquisition-dev` / `65b40fb79570c0767ed563d6e25be2c980c58985-alm34-dirty` |
| Toolchain / target | Go 1.27.3; linux/amd64 v1; CGO disabled; trimpath; buildvcs=false |
| Binary SHA-256 | `4b484239a3939e9f1f12eb6fda22a0f74a2e3d035023d3b01ec8e9e43aaf78f9` |
| Rendered bootstrap SHA-256 | `1d550b97f30acd96373d206c713c90efd1deefde6172a3a996b07cdf194b6183` |
| Release manifest SHA-256 | `d939b44aabd103378531cea4f864d1186094b065737159572dbad1cced49e233` |
| Original build-preparation source payload SHA-256 | `d4c301aa3e6c5ccc57e808d15034b92c65dd9a64373ba68cd34c3da719c06c50` |

Three builds were byte-identical, including a final rebuild after extending only
the existing S11 helper's allowed developer profile. Protected final input hashes
and source snapshot explain that helper change; original release metadata remains
an immutable build-preparation record. The baseline alone is not claimed to
produce this dirty-tree release. CLI notices cover AlmaDial, Go and linked YAML;
third-party applications are separate retained source inputs. The ALM-32 Go
vulnerability follow-up remains outside this bounded correction.

A pristine 4 CPU / 8 GiB / 80 GiB AlmaLinux 10.2 host downloaded the script and
installed its verified precompiled CLI as root:root 0755 in
`/usr/local/sbin/almadial`. Go and Git were absent before and after obtaining the
CLI. Root-login PATH/help/version passed. An initial non-login `sudo bash -c`
post-check lacked `/usr/local/sbin`; bootstrap itself succeeded. The documented
cloud-image path is `ubuntu` then `sudo -i`; SSH policy was not altered.

Bare `almadial install` prompted for snapshot/authorization references, network
and TLS choice, acquired all five archives automatically, generated protected
localhost lab TLS and credentials, displayed its summary and digest, and accepted
one explicit confirmation. Exact plan SHA-256:
`241d9f0f4fd8f14c4201a38dc0c96099234cc79d7021d46021563c7fb7fa98b7`.
No manual hash copy, compiler, checkout, binary/source transfer or stage commands.

## Developer runtime qualification

The first execution stopped at S03's immediate MariaDB service postcondition.
The documented guided resume reverified all cached inputs, displayed the same
plan, reconciled the healthy service and continued without manual service repair
or database reimport. Both attempts are preserved; uninterrupted first-attempt
success is not claimed. Resume completed S01–S10 and S12 with exit 0. Actual
postconditions show all seven services active, 731 accepted RPMs, Asterisk
22.12.0 with all 40 required modules running, MariaDB 10.11.18/schema 1748,
loopback-only protected listeners, six correctly owned FastAGI processes,
deferred updater/listener disabled, zero active application cron jobs, SELinux
Enforcing, and matching isolated permanent/runtime firewall policy. Trusted
HTTPS/WSS return the expected unauthenticated 401/426 responses.

Standalone Status and Doctor each returned exit 10: the expected short-lived
laboratory certificate expiration warning, plus nonblocking journal
NOT_CONFIGURED because no particular stage journal was selected. There were no
FAIL or UNKNOWN checks. This laboratory warning is recorded, not suppressed.

The actual bootstrap checksum-negative test returned 1 with “nothing installed
or executed”; the installed CLI hash remained unchanged. Go and Git are absent
on the target. The endpoint was rechecked with its trusted public CA.

A controlled cache-loss fixture preserved the original five files, emptied only
the active artifact cache, then ran bare guided install. All five original
objects were reacquired automatically through the private HTTPS store. It
approved the same plan once, returned exit 0 and genuine `no_op: true`. Before/
after comparisons prove unchanged journal bytes, Asterisk/AlmaDial/firewall
configuration hashes, schema catalog hash, call-log count, all seven service
PIDs, accepted artifact hashes and plan identity. A second run with the private
artifact endpoint stopped used the verified cache and returned NO_OP. Deterministic
fixtures additionally prove zero network requests for valid cache hits; the
live guest retained general network connectivity. The private HTTPS endpoint was
restored immediately and verified again.

The first offline run's Multipass interactive output transport stalled after
8,192 bytes of final JSON despite no remaining guest installer process. That
partial transcript and unchanged-state evidence are retained. A complete repeat
used a guest terminal captured over noninteractive transport, preserving the
same one-confirmation CLI path; it supplies the complete exit/result evidence.

The existing locked S11 media helper is extended only for ALM-34's exact task,
developer VM/machine ID and media scenario. Tests reject the independent operator
VM, accepted M3 target, wrong task/machine and reboot scenario. Its existing
approval, journal, bounded source-scoped RTP allowance and cleanup remain intact.
No second installer or acceptance engine was added.

That existing S11 harness passed on the developer host with two real headless
Chrome 157.0.8103.38 VICIphone clients. Both registered through trusted WSS,
joined ConfBridge 9600271 by dialing 9271, and negotiated PCMU, connected DTLS and
SRTP_AES128_CM_HMAC_SHA1_80. Each client received 605 and transmitted 606 packets
during the bounded capture; no TLS bypass was used. Accepted PCM analysis passed
in both directions: pattern agreement 0.9960/0.9921, envelope correlation
0.9688/0.9937, and tone agreement 0.9669/0.9865. All four synthetic tones were
recognized. This is measured bidirectional browser media, not a claim of human
microphone/speaker listening or Internet NAT coverage.

Both mixed MixMonitor WAVs and their receive/transmit legs were collected
privately. Call-log unique IDs `1794600130.0` and `1794600137.3` match browser
channel evidence and filenames. Both mixed recordings passed accepted content
analysis (278 distinguishable frames, agreement 0.9496/0.9892). The second call
lasted 12 seconds and the first 19 seconds. Content-alignment offsets are not
latency measurements. Cleanup removed the temporary media overlay and runtime
RTP allowance, preserved permanent policy/configuration/secret hashes, and left
no active channels. Recordings and browser profiles remain protected.

Broad reboot, full application/operator journeys, physical listening, recovery
and backup acceptance were not repeated. Accepted ALM-29 qualification is reused
where applicable; the human must produce the independent ALM-34 acceptance
record, followed by AGY assessment.

## Regression and preservation

PASS: `go vet ./...`; `go test ./...`; `go test -race ./...`; `go mod verify`;
`go mod tidy -diff`; `python3 scripts/validate_scaffold.py`;
`python3 -m unittest discover -s scripts/tests -v` (243 tests);
`git diff --check`; all Go files gofmt clean; Bash syntax; reproducible builds;
focused acquisition/CLI/release/harness-profile tests. The first restricted
regression attempt could not create loopback HTTPS fixture sockets. Re-execution
with required loopback permission passed; both logs are retained.

Deterministic HTTPS fixtures cover unavailable primary 404 and approved fallback,
corrupt-cache quarantine and verified retry, wrong-revision digest rejection,
offline valid cache with zero requests, exhausted origins without verified cache
publication, and untrusted-store rejection. Existing canonical-tree/count mismatch
and archive-policy checks remain. Bootstrap fixtures prove checksum mismatch
never executes or installs, and unsupported OS/version/architecture reject early.
Guided tests cover exact digest display/one confirmation, refusal/cancel/EOF,
nonterminal refusal, TLS preservation and private saved-input protection.

S01–S10 planners/execution, S11 functional boundary, S12 diagnostics, plan/source/
host binding, `apply.lock`, durable journals, reconciliation, secret redaction,
trusted TLS, signed packages, SELinux and firewall isolation remain the accepted
implementation. No new database, stage orchestration or journal format.

## Operator kit, evidence and handoff

Use [the short installation guide](../operations/almadial-installation.md), with
explicit private CA enrollment, then:

```bash
ssh root@SERVER_IP
curl -fsSLO https://10.160.90.1:8447/install.sh
bash install.sh
almadial install
```

The supplied cloud image uses `ssh ubuntu@SERVER_IP`, then `sudo -i` before the
remaining commands. The guide includes application CA trust/tunnels, access URLs,
private credentials, Status/Doctor, failed-prerequisite diagnostics and resume.
[Private retention operations](../operations/almadial-private-artifact-retention.md)
belong to the maintainer, not the operator.

New protected evidence:
`~/almadial-lab/evidence/m4-06/acquisition-20261113T193809Z-alm34/`.
Credentials, private keys and raw browser/recording evidence stay outside Git.
Qualification evidence includes exact archive recovery/inventory, private store
inventory and HTTPS GET records, pristine host/bootstrap/checksum rejection,
guided installation plus recovery, S11 media/recordings/cleanup, both NO_OP runs
and state comparisons, Status/Doctor, complete regressions, reproducible build
identity and reproducible source inputs. `QUALIFICATION_SHA256SUMS` seals the
qualification; final tracker read-back and documentation are sealed separately
under `handoff/`. The human
operator must separately execute the guide without Codex assistance, record the
canonical mandatory acceptance evidence/assistance, and submit it for AGY
assessment before owner acceptance. ALM-34 remains Codex-owned In Progress.

The qualification ledger covers 654 regular files; its SHA-256 is
`2b7c62135bb9eeaaaee77ebb2502266e4474ac4d626f3728d21dc4289f0e8472`.
A fourth build from the retained 995-file source snapshot reproduced the exact
served CLI. A final guest check again found neither Go nor Git installed and
SELinux Enforcing.

The initial final-handoff attempt encountered a Kaneo connection failure. Its
prepared comment and failure evidence remain sealed under `handoff/`; historical
evidence has not been overwritten. After the owner reported restored connectivity,
Codex retried through `kaneo-local` without making tracker infrastructure changes.
The exact prepared handoff was posted as comment
`gmc6hm2pxy10tj36lrv56d4b` at 2026-11-13 21:48:32 UTC and read back with an exact
content match. Fresh MCP reads confirm Codex/In Progress, accepted ALM-29–33,
unchanged dependency relations, and ALM-35 To Do/unassigned. No task state,
assignment or dependency was changed. The sealed qualification ledger and real
trusted HTTPS bootstrap checksum were reverified during the retry.

Successful read-back, final documentation and their SHA-256 ledger are retained
separately under `handoff-retry-20261113T214832Z/`. Developer installation
qualification and handoff are complete. The next step is a separate human's
unassisted operator run, followed by AGY evidence assessment; neither has been
started or claimed as passed by Codex.

Final developer verdict:
`ALMADIAL_M4_06_BOOTSTRAP_READY_FOR_OPERATOR_ACCEPTANCE`.

## GitHub publication checkpoint — 2026-11-14

The owner authorized a bounded implementation commit and GitHub branch push.
The intended branch is `alm34/github-bootstrap-operator-acceptance`, retaining
accepted baseline `65b40fb79570c0767ed563d6e25be2c980c58985`. GitHub repository
`grimange/almadial` is public; its remote `main` was
`bab92445bf1fe338d6cca46b422c7fd651e106e2`, 65 commits behind that baseline.
Consequently a branch from the accepted baseline would publish all 65 ancestors.

**Publication is held under the owner's explicit outgoing-history audit rule.**
Commit `1df743a4178fc32d9d78d53de8ea0870b1d2c30f` contains `AGENTS.md` and
`.agents/skills/*`, which the current task excludes as local AI workflow state.
Commit `8b2d1e08f9731159156ea43a5ba8dbbf0de8da19` deletes those files from the
tree but retains them in the history that a push would publish. Preserving the
accepted ancestry and excluding those historical files require a further owner
decision; this checkpoint neither rewrites history nor publishes that history.

The outgoing payload also includes the VICIdial-derived
`operations/qa/proposals/prd-03c/preview-logout.patch`, introduced by
`a8ba1afecc54ead750c6b7c4c992cca832e1d723`, and
`operations/vicidial/patches/0001-preview-logout.patch`, introduced by
`631cbe61b2ac6f4d7fa5665438c87fce2318dbd6`. ALM-32's distribution decision
records unresolved base applicability and source-delivery conditions for these
patches. The present original-code/bootstrap authorization does not clear those
third-party publication conditions. This is a specific payload issue, not a
claim that upstream archives or `line.js` are embedded in the installer.

The local publication payload now separates the release template
`scripts/bootstrap/install.sh.in` from the configured operator script
`scripts/bootstrap/install.sh`. The latter is byte-identical to the verified
private HTTPS bootstrap: SHA-256
`1d550b97f30acd96373d206c713c90efd1deefde6172a3a996b07cdf194b6183`,
pinning the previously qualified CLI checksum and private URL above. The release
preparer and rootless bootstrap fixtures use the template; an additional check
binds the configured script to that approved release. No source or CLI repin.

The operator guide retains its actually verified private HTTPS workflow.
No GitHub raw URL is claimed to work before publication and HTTP/content
verification. The independent operator VM remains untouched. New publication
audit, checkpoint and regression evidence is separate from historical evidence:
`~/almadial-lab/evidence/m4-06/github-20261113T220347Z/`.

Fresh checkpoint validation passed `go vet ./...`, `go test ./...`,
`go test -race ./...`, module verification and `go mod tidy -diff`, scaffold,
all 244 Python tests, both Bash syntax checks, Go formatting, focused acquisition/
source-integrity/guided-install tests, CLI help/version and diff checks. Two
Linux amd64 builds again match the already approved CLI SHA-256 exactly.
The configured bootstrap was retrieved through trusted private HTTPS and its
bytes matched the recorded script checksum. No full VM installation or media
campaign was repeated for this Git checkpoint.

This GitHub-delivery gate is separate from the earlier successful developer
installation qualification. Independent human acceptance and AGY assessment
remain outstanding, with ALM-34 Codex-owned and In Progress and ALM-35 unstarted.

GitHub-delivery verdict: `ALMADIAL_M4_06_GITHUB_BOOTSTRAP_BLOCKED`.
