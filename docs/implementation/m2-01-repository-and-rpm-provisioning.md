# M2-01 — Repository and RPM provisioning

ALM-16 implements the bounded S01 dependency stage in `internal/provision` and
an explicit qualification executable in `cmd/almadial-rpm`. The normal
`almadial` installer/wizard remains subject to the accepted manifest approval
contract. `manifests/stack.lock.yaml` remains CANDIDATE.

The Go stage uses the existing ALM-11 plan, ALM-13 action graph/process runner,
and ALM-14 lock/journal. Its embedded fixed Python adapter calls the AlmaLinux
DNF 4/RPM libraries already installed with DNF. It does not resolve dependencies
itself, accept arbitrary executable arguments, or provide a package plugin API.

## Qualified input and package policy

The direct requests derive from `scripts/lab/m0_08_package_specs.json`, the
accepted M0-08 replay, M0-02 package metadata, and actual RPM epochs observed on
the qualified M0 repeat host. `internal/provision/packages.json` is the executable
inventory. Exact epoch:version-release, architecture and repository must match.
The policy intentionally stops when the qualified versions are no longer
available; current repositories are not a permanent snapshot service.

Asterisk 22.12.0 with bundled PJPROJECT 2.17, VICIdial r4037, VICIphone 3.0 and
Net::Telnet 3.05 remain source-managed under the accepted artifact contract.
AGI.pm and compatibility derivations remain application-stage responsibilities.
Legacy `perl-Net-Telnet`/`perl-core` names are not blindly requested. Optional
M0 investigations such as sox/lame are not represented as required RPMs without
an accepted direct requirement. ALM-16 neither builds applications nor initializes
MariaDB.

All RPMs below are exact constraints. `noarch` is explicit; all others require
`x86_64`. The installer host is restricted to AlmaLinux **10.2 x86_64**, the
observed qualified input, rather than claiming untested 10.x releases.

| Direct RPM | Epoch:version-release | Architecture | Repository |
|---|---|---|---|
| `autoconf` | `0:2.71-13.el10` | noarch | appstream |
| `automake` | `0:1.16.5-20.el10` | noarch | appstream |
| `bzip2` | `0:1.0.8-25.el10` | x86_64 | baseos |
| `ed` | `0:1.20-5.el10` | x86_64 | baseos |
| `gcc` | `0:14.3.1-4.4.el10.alma.2` | x86_64 | appstream |
| `gcc-c++` | `0:14.3.1-4.4.el10.alma.2` | x86_64 | appstream |
| `httpd` | `0:2.4.63-13.el10_2.8` | x86_64 | appstream |
| `jansson-devel` | `0:2.14-3.el10` | x86_64 | crb |
| `libedit-devel` | `0:3.1-52.20230828cvs.el10` | x86_64 | appstream |
| `libsrtp-devel` | `0:2.3.0-16.el10` | x86_64 | crb |
| `libtool` | `0:2.4.7-13.el10` | x86_64 | appstream |
| `libuuid-devel` | `0:2.40.2-18.el10_2.20` | x86_64 | appstream |
| `libxml2-devel` | `0:2.12.5-10.el10_2.4` | x86_64 | appstream |
| `make` | `1:4.4.1-9.el10` | x86_64 | baseos |
| `mariadb` | `3:10.11.18-1.el10_2.alma.2` | x86_64 | appstream |
| `mariadb-backup` | `3:10.11.18-1.el10_2.alma.2` | x86_64 | appstream |
| `mariadb-server` | `3:10.11.18-1.el10_2.alma.2` | x86_64 | appstream |
| `mod_ssl` | `1:2.4.63-13.el10_2.8` | x86_64 | appstream |
| `ncurses-devel` | `0:6.4-15.20240127.el10_1` | x86_64 | appstream |
| `openssl-devel` | `1:3.5.8-2.el10_2.alma.1` | x86_64 | appstream |
| `patch` | `0:2.7.6-26.el10` | x86_64 | appstream |
| `perl` | `4:5.40.2-515.el10_2` | x86_64 | appstream |
| `perl-Class-Mix` | `0:0.006-22.el10_0` | noarch | epel-almadial |
| `perl-Crypt-Eksblowfish` | `0:0.009-42.el10_0` | x86_64 | epel-almadial |
| `perl-DBD-MySQL` | `0:5.007-4.el10` | x86_64 | appstream |
| `perl-DBI` | `0:1.643-26.el10_2.7` | x86_64 | appstream |
| `perl-Devel-CallChecker` | `0:0.009-7.el10_0` | x86_64 | epel-almadial |
| `perl-DynaLoader-Functions` | `0:0.004-5.el10_0` | noarch | epel-almadial |
| `perl-Net-Server` | `0:2.014-9.el10` | noarch | appstream |
| `perl-Params-Classify` | `0:0.015-18.el10_0` | x86_64 | epel-almadial |
| `perl-Switch` | `0:2.17-33.el10_1` | noarch | epel-almadial |
| `perl-Time-HiRes` | `4:1.9777-511.el10` | x86_64 | appstream |
| `php-cli` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-common` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-fpm` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-gd` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-ldap` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-mbstring` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-mysqlnd` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-opcache` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `php-xml` | `0:8.3.36-1.el10_2` | x86_64 | appstream |
| `pkgconf` | `0:2.1.0-3.el10` | x86_64 | baseos |
| `sqlite-devel` | `0:3.46.1-6.el10_2` | x86_64 | appstream |
| `zlib-ng-compat-devel` | `0:2.2.3-3.el10_1` | x86_64 | appstream |

## Repository and trust contract

| Repository | Approved source | Use |
|---|---|---|
| baseos | `https://mirrors.almalinux.org/mirrorlist/10/baseos` | Enabled Alma definition |
| appstream | `https://mirrors.almalinux.org/mirrorlist/10/appstream` | Enabled Alma definition |
| crb | `https://mirrors.almalinux.org/mirrorlist/10/crb` | Enabled Alma definition |
| extras | `https://mirrors.almalinux.org/mirrorlist/10/extras` | Enabled Alma definition |
| epel-almadial | `https://dl.fedoraproject.org/pub/epel/10.2/Everything/x86_64/` | Transaction-local, only the six listed Perl RPMs |

Enabled foreign repositories, duplicate definitions, alternate active sources,
wrong release/architecture, disabled mandatory Alma repos, missing pinned keys,
`gpgcheck=0` or `sslverify=0` are rejected. Disabled unrelated definitions are
preserved. Repo metadata is obtained over verified TLS; Alma's existing
`repo_gpgcheck` setting is observed, not silently changed or represented as
signed-metadata enforcement. Package signatures are mandatory.

| Key | Full fingerprint | SHA-256 of qualified ASCII armor |
|---|---|---|
| AlmaLinux 10 | `EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572` | `de57c3c0a2dcde88ff5b894e74335f55a0ec074815f36d472270d937c16f077b` |
| EPEL 10 | `7D8D15CBFC4E62688591FB2633D98517E37ED158` | `de390fc168eae5ab2852e9e93d34a0b9ddf05cf9ce90ee28d97de26a4b1f6b93` |

Alma's repository key file and both imported public keys are checked by hash.
The one `rpm.trust` action can start with only the qualified EPEL key missing.
It imports exactly the embedded public key, then verifies that only its expected
keyring/hash and `gpg-pubkey` RPM were added. Other host/repo checks still apply.
A colliding key identity fails closed. No `.repo` file is generated or rewritten.
There is no automatic EPEL-release installation or broad EPEL enablement.

The retained M0 Switch and Crypt-Eksblowfish RPMs also have fixed payload hashes
in `PinnedRPMHash`. All new RPMs are bound to native metadata SHA-256, checked
again after download, checked cryptographically by native RPM/DNF, and required
to bear the expected repository signing identity before transaction execution.
Already-installed RPM assessment checks RPM database identity/signature headers
and imported trust; it does not claim fresh payload verification of every
installed file or infer historical download origin from an installed header.

## Inspection, native resolution and immutable plans

`inventory` and `inspect` read OS release, machine-id hash, repository definitions,
public key identities, installed RPM headers and committed DNF history. They do
not call `fill_sack`, refresh repository metadata, initialize a history database,
install packages or modify configuration. Native RPM/SQLite readers may update
`-shm` read-coordination files; this is observed explicitly and is not a repository
refresh or a persistent RPM/history-content change. History is opened using SQLite's
read-only URI; unknown/incomplete history fails closed. Its identity hashes
canonical transaction records, avoiding incidental SQLite initialization and
checkpoint layout differences. A changing database or unknown schema is rejected.

`assess` and `plan` consume saved JSON without host probes. Assessment returns
SATISFIED, REQUIRED, INCOMPATIBLE, UNAVAILABLE or NOT_VERIFIED. Missing packages
are NOT_VERIFIED before native availability resolution; a failed solver process
is an explicit failed operation rather than fabricated candidate evidence.

`rpm.resolve` is a separately authorized operation because DNF metadata/cache
and its normal state files may change. It calls native `Base.install` for exact
qualified requests and `Base.resolve(allow_erasing=False)`. Plugins and weak
additions are disabled; missing repositories are fatal. Its complete proposal
contains NEVRAs, origins, payload hashes and native removals. Canonical sorting
makes equivalent proposals and graphs deterministic. The full proposal must be
reviewed, saved as `input.transaction`, replanned and separately authorized.

The native API is documented by [DNF Base](https://dnf.readthedocs.io/en/latest/api_base.html)
and [DNF transaction](https://dnf.readthedocs.io/en/latest/api_transaction.html).
DNF selects the closure; AlmaDial does not maintain a transitive dependency solver.

A clean-image resolution required **344 additions and seven coupled updates**.
`internal/provision/upgrades.json` restricts the updates to libblkid, libfdisk,
libmount, libsmartcols, libuuid, util-linux and util-linux-core, all x86_64,
from `0:2.40.2-18.el10` to `0:2.40.2-18.el10_2.20`, exact BaseOS hashes. These
match the qualified repeat host and satisfy the pinned libuuid-devel. Arbitrary
updates, downgrades, reinstalls, cross-architecture replacements and unpaired
removals are rejected. Native `rpm.labelCompare` independently confirms update
direction. There is no blanket `dnf upgrade` or permission for arbitrary solver
substitutions.

## Implemented qualification commands

Build with the pinned Go 1.27.3 toolchain:

```bash
go build -trimpath -buildvcs=false -o /tmp/almadial-rpm ./cmd/almadial-rpm
/tmp/almadial-rpm --help
/tmp/almadial-rpm inventory > /protected/inventory.json
/tmp/almadial-rpm assess --input /protected/inventory.json
/tmp/almadial-rpm inspect --config /protected/config.json \
  --requirements /protected/requirements.json > /protected/inspection.json
/tmp/almadial-rpm plan --input /protected/inputs.json \
  --manifest manifests/stack.lock.yaml > /protected/plan.json
```

`/protected` stands for an existing operator-owned private evidence directory;
create and protect it first. Config and resource requirements use the accepted
M1 schemas. The `inspect` result is an `Inputs` document. For the key stage set
`prepare_repository_trust` to true; for resolution omit that field and
`transaction`; for package execution attach the reviewed native transaction.
Fresh inspection is needed after key import or any relevant state change.

The generated plan includes `stage`, `stage_sha256` and `graph`. A separate
`LabAuthorization` JSON must contain the exact scope `ALM-16/disposable-lab`,
host SHA-256, stage SHA-256, parent plan SHA-256, manifest SHA-256,
snapshot-reference SHA-256, operator-evidence SHA-256 and
`candidate_acknowledged: true`. These bindings record externally verified
permission; arbitrary digest strings do not establish operator permission.
The worker must first verify the exact disposable VM and recovery record.

Only after review and exact-target authorization:

```bash
/tmp/almadial-rpm lab-run --input /protected/inputs.json \
  --manifest /protected/stack.lock.yaml \
  --authorization /protected/authorization.json \
  --state-dir /var/lib/almadial-qualification/stage
/tmp/almadial-rpm lab-resume --input /protected/inputs.json \
  --manifest /protected/stack.lock.yaml \
  --authorization /protected/authorization.json \
  --state-dir /var/lib/almadial-qualification/stage
```

Use the **original** inputs, approval and state directory for resume. Each
separately authorized laboratory graph has its own durable journal; do not run
qualification stages concurrently. This entry point is scoped to S01, not the
complete multi-stage executor. Normal `almadial install` remains unsupported.
Normal `InstallationAuthorized()` still rejects the candidate manifest.

## Action, lock, journal and recovery contract

Each immutable specialization binds M1 configuration, manifest and parent plan
identities plus the fixed package/repository policy, observed host state and
ALM-13 graph. The fixed actions are `rpm.trust`, `rpm.resolve`, and `rpm.apply`.
They belong to S01 and each has `rpm.fresh` preconditions and an independently
checked postcondition. A specialization has one action, or zero for an already
satisfied key/package stage. There is no arbitrary command field in input JSON.

Commands use `/usr/bin/python3` with direct argv, a fixed embedded adapter,
20-minute mutation/resolution deadlines, 30-second inspection deadlines, 4 MiB
stdout and 64 KiB stderr limits, cancellation and the existing ALM-13 process
group cleanup. Diagnostics contain safe exception class/adapter line numbers,
not private DNF exception strings. CLI JSON is protected host evidence, not a
public log. No credential values are accepted or resolved by this stage.

After permission and host validation, ALM-14 acquires exclusive journal
ownership, persists action intent, rechecks state under that ownership, and runs
the action. Native DNF also takes its package-manager lock and performs the
transaction test. Apply re-solves and compares the **entire** transaction to the
reviewed proposal before download and again checks actual host state before
commit. No implicit key imports are allowed during package application.

The postcondition checks the actual full RPM inventory delta, exact required
NEVRAs, architectures, expected signers, repository/key policy and complete DNF
history. Extra/missing RPMs, retained replaced versions, or exit 0 with incorrect
state fail. The accepted journal records success only after independent
observation. Optional operator logging uses the accepted event/logger contract.

Retry policy is NEVER. Resume re-inspects real RPM/DNF state. A completed
transaction can reconcile as succeeded without replay. Partial/absent packages,
incomplete native history, unknown result or lost resolver output become durable
`reconciliation_required`. Resume never repairs RPM databases, guesses success,
replays an uncertain DNF transaction, or provides RPM rollback. Recovery requires
an operator decision or the recorded disposable-VM snapshot.

## Recorded disposable-laboratory results

Qualification ran on 2026-11-09 UTC / 2026-11-10 Asia/Manila. Operator
permission was recorded separately for each target and for the final full
transaction. These permissions authorize laboratory qualification only.

| Target | Recovery record | Actual result |
| --- | --- | --- |
| `almadial-alma10-repeat` | `alm16-recovery-20261110`; historical `clean-baseline` preserved | 744 RPMs; all 44 direct requirements satisfied; journaled native resolution returned no changes; package execution returned NO_OP. |
| `almadial-alm16-rpm` | `alm16-baseline-20261110`; then `alm16-qualified-20261110` | Fresh AlmaLinux 10.2 x86_64 image had 380 RPMs and none of the 44 direct requirements. Exact EPEL key import produced 381 RPMs. Reviewed native transaction produced 725 RPMs: 344 new RPMs and the seven approved updates. All 44 exact direct NEVRAs/architectures/signers and the complete inventory delta passed independent inspection. |

The image SHA-256 was
`cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`.
The fresh target machine-ID SHA-256 was
`685c7a0bc529efffdc5329d7d9c856d85c460bd15c3dc8772234c0744fd07141`.
The reviewed transaction SHA-256 was
`79eba1f6bc2f0fdd4e9960fb95f22f57e2721b0cb673e9f494dbe6ee7ef0f7f8`.
Its first successful package-stage digest was
`bf02fff361596420b7536a743a2a613e6707931c04d638118bc3325d0b1889cd`.
DNF transaction 1 committed successfully; ALM-14 recorded `rpm.apply` succeeded.
A second native resolution returned zero installs/removals. The resulting empty
graph returned `no_op: true` after package verification. Completed-journal resume
re-observed the installed state without replay.

The initial additive policy rejected DNF's seven coupled updates before any
package installation. The operator then reviewed and authorized the exact full
proposal. There was no blanket upgrade or unreviewed package substitution.
Final native `dnf --cacheonly --disablerepo=* --setopt=plugins=0 check --dependencies`
passed. A read audit checked 58 files: persistent repository, RPM/history database
and metadata-cache content remained unchanged; SQLite shared-memory read
coordination was recorded separately. Existing repository-file hashes remained unchanged. The EPEL repository existed
only in the transaction's native DNF configuration. No Asterisk or VICIdial build,
MariaDB initialization, telephony configuration, firewall or SELinux change was
performed by AlmaDial.

## Live rejection and interruption evidence

All canonical live cases used actual RPM/DNF tools on the separate authorized
VM; unit fixtures are reported separately.

| Case | Observation |
| --- | --- |
| Required package unavailable | Native DNF, cache-only with deliberate `autoconf` exclusion, raised `PackageNotFoundError`; RPM count stayed 381. This fixture tests unavailable resolution without adding a repository. |
| Untrusted key | Official qualified EPEL RPM signature inspection failed before the EPEL key import; passed after the exact key import. The rejected RPM was never installed. |
| Invalid payload | RPM signature/digest verification rejected a corrupted copy of the qualified RPM; never installed. |
| Incompatible repository configuration | A temporary, task-owned duplicate `baseos` definition caused actual adapter inventory to fail. The exact fixture was removed after verifying ownership/content. Existing repository files were preserved. |
| Interrupted installation | Controller sent SIGTERM after the real RPM count reached 386. Cancellation stopped with 387 RPMs; native history recorded transaction 1 unfinished (`dt_end=0`, state 0). Apply exited 1. Resume exited 1, durably recorded `reconciliation_required`, and preserved RPM count/history. The final reproducible binary independently refused replay using the original plan/journal. |
| Repeat execution | Native resolution returned an empty transaction and verified package execution returned NO_OP. |

The first interruption experiment completed normally because its trigger waited
for a history observation that was not visible early enough. It is retained as
an experiment result, not counted as interruption coverage. The corrected
controller watched actual RPM database growth. Evidence was captured before
restoring **only** the separate VM's qualified snapshot. There is no automatic
transaction replay, package rollback, or repair engine.

Nonempty package postconditions also require a changed, committed DNF history
observation. Correct RPM headers with unchanged pre-transaction history cannot
establish successful completion. Unit tests cover timeout, cancellation, network
and nonzero failures, postcondition failure, wrong version/architecture,
unapproved sources/keys, stale inputs, authorization rejection, lock contention,
interruption and unknown reconciliation state. These controlled failures are
not presented as additional live network/timeout experiments.

## Regression and reproducibility

Go 1.27.3 validation passed: `go fmt ./...`, `make fmt-check`, `go vet ./...`,
`go test ./...`, `go test -race ./...`, `go mod tidy -diff`, and `go mod verify`.
Normal and race runs each recorded 125 passing top-level tests and 644 passing
test/subtest events, zero failures, and one skipped optional real-M0-artifact
test. Existing process-group cleanup tests ran. Python discovery passed 131
tests, including five new native-adapter boundary tests using real local SQLite
fixtures without root or package mutation. Scaffold, candidate-stack and config
generation validation passed. CLI regression passed 17 smoke cases and all 11
terminal/nonterminal journeys. Hosted CI was NOT_RUN.

Two isolated source copies produced identical Linux amd64 binaries:

| Binary | SHA-256 |
| --- | --- |
| `almadial` | `1eaf7bd52421b2b4928083894be5fc8a79276d0f1307536b46910b10b8606173` |
| `almadial-rpm` | `d6895beb4b8e7809514baee2ddf372012bb68361db6e3eb8c98c618ae71fe0b1` |

Build metadata identifies
`35b439365729ab2bc0984a148ad6801e14d0258c-alm16-dirty` for the normal CLI.
Starting HEAD was `35b439365729ab2bc0984a148ad6801e14d0258c`; final observed
HEAD is `d851d86279e2611b3fa70d57834f5cc938d8e200`, an operator-authored
commit affecting unrelated operations evidence and `.gitattributes`. The Go
source/build inputs were unchanged by that commit. Both descend from accepted
M1 gate `c761930235a7810008dc3fb45d6325796ff5b208`.
Task source changes remain uncommitted for independent review. No push occurred.

## Evidence, limitations and handoff

Protected evidence is sealed under
`~/almadial-lab/evidence/m2-01/20261109T161558Z/` with a SHA-256 ledger.
It includes operator approvals, image/snapshots/host identities, native inventories,
repo-file hashes, actual transaction proposals and traces, journals, negative
experiments, interrupted RPM/DNF state, recovery, exact validation commands and
results, reproducible hashes, source hashes and Git status. Raw protected host
JSON, binaries and evidence are not committed. Kaneo handoff read-backs are
stored separately from the sealed qualification ledger to avoid a hash cycle.

The qualified contract is AlmaLinux **10.2 x86_64**, with exact direct NEVRAs.
Other AlmaLinux 10 minor releases/architectures are unqualified. The four standard
Alma repositories must already have their qualified configuration; this stage
rejects conflicts rather than rewriting administrator files. `python3-dnf`, RPM
Python bindings and native DNF/RPM must already exist. Rolling repositories can
retire exact versions: absence fails explicitly rather than selecting a substitute.
RPM signer/header inspection proves package identity, not a full filesystem
integrity audit of every pre-existing package. The fixed adapter currently passes
bounded inventories through direct argv; unusually large inventories can hit the
Linux argument limit and fail closed. Repository metadata signatures are not
claimed where the distribution config has `repo_gpgcheck=0`; HTTPS and verified
RPM signatures remain enforced.

Source compilation, application compatibility helpers, service identities,
MariaDB initialization, service configuration, firewall/SELinux policy, complete
installer execution and production manifest promotion remain later bounded work.
Independent Google Antigravity QA must reproduce the critical boundaries and
review the protected native evidence before acceptance. ALM-17 remains unstarted.
