# M4-02 — Failure injection, resume and second-run safety

Developer qualification, 2026-11-12. Task ALM-30 (`lcy5mrwd89v2hahs1qzn4uow`).
Starting and final implementation baseline: `d2805e4447aad47db91c48d3eb0e5456c22a118f`.
Independent AGY acceptance remains required. No release, commit or push is made.

## Contract and stage crosswalk

Canonical Kaneo acceptance requires every selected interruption to resume safely
**or explicitly block**, no duplicate state/data loss/secret overwrite, rejection
of a competing installer before mutation, and no untested rollback claims.
This qualification adds test-only fixtures around the accepted unified adapter,
journal and lock. It does not add a production injection switch or execution engine.

The authority for stage names is `installStages` in
`internal/provision/install_linux.go`, not earlier narrative M3 feature numbering.
Each S02–S10 stage consumes its mandatory preceding foundation; later stages cannot
replace the prerequisite checks. The authoritative crosswalk is:

| Internal ID | Responsibility | Mandatory foundation | Selected native interrupted action | Decisive postcondition |
|---|---|---|---|---|
| S01 | Packages/repositories | Approved host, sources and package plan | `rpm.apply` | Qualified transaction/inventory |
| S02 | Identities, storage, SELinux | S01 | `system.02.directory.15` | Owned identities/directories/labels |
| S03 | MariaDB/VICIdial schema | S02 | `database.05.restart` | Database health, catalog, grants |
| S04 | Asterisk/native modules | S03 and package/build inputs | `asterisk.05.restart` | Runtime and required modules |
| S05 | VICIdial application | S04 and database | `vicidial.06.verify` | Pinned deployment/configuration |
| S06 | FastAGI/deferred scheduling | S05 | `runtime.02.verify` | Parent/five workers, deferred jobs |
| S07 | Apache/PHP application delivery | S06 | `web.02.verify` | Application/configuration health |
| S08 | Supplied-certificate TLS | S07 | `tls.00.activate` | Trusted local HTTPS/WSS |
| S09 | Isolated host firewall | S08 | `network.00.activate` | Runtime/permanent policy and isolation |
| S10 | VICIphone/synthetic PJSIP | S09 | `phone.00.activate` | Pinned assets, generated phone/PJSIP |
| S11 | Explicit synthetic call/media qualification | S10 | Not an automatic provisioning stage | Existing functional acceptance harness |
| S12 | Read-only diagnostics | Provisioned stack | Not a mutating stage | Operational diagnostic report |

Thus the installed CLI reports eleven completed labels: S01–S10 plus S12.
S11 remains an explicitly invoked functional qualification. This corrects the
stage interpretation without rewriting ALM-29's historical QA report.

## Deterministic interruption matrix

`TestInstallationBoundaryInjection` exercises the real `driveInstallStage`,
exclusive ownership, durable journal and receipt handling with one deliberately
synthetic action per S01–S10 stage. There are twenty selected cases:

| Fixture boundary | Stages | Interrupted journal | Recovery outcome |
|---|---|---|---|
| Before effect | S01, S03 | RUNNING, no accepted receipt | Explicit reconciliation block; zero effects |
| Before effect | S02, S04–S10 | RUNNING, no accepted receipt | Idempotent recovery; exactly one effect |
| After effect, before persisted completion | S01–S10 | RUNNING, no accepted receipt | Actual satisfied observation reconciles; exactly one effect total |

Successful final receipts do not replay the fixture effect on another adapter
call. Original approved input and completed effects remain distinguishable.
These are adapter/journal fixtures, **not** interrupted DNF, DDL, compiler or
firewall operations. Existing component tests retain the native partial-operation
retry policy; ALM-29's actual S03 pause/resume is reused as live workflow evidence.
No new clean installation, destructive interruption, rollback or media campaign
was performed for ALM-30.

## Native copied-journal reconciliation

Target: `almadial-alm29-clean-d`, machine
`4c7b492940923838a0700adba65ed3eb`, AlmaLinux 10.2, 731 RPMs.
Recovery snapshot `alm29-pristine` was read back through Multipass. The accepted
M3 VM and all historical snapshots were untouched.

`TestInstallationNativeCopiedRecovery` is explicitly opted in, root/host/output
bounded, and skipped by ordinary repository tests. It first calls the existing
fresh `AssessInstall`. It then copies the canonical state into protected separate
directories, marks the selected final action RUNNING in each **copy**, and calls
`journal.Resume` with the existing native observer and original identity. It does
not delete accepted receipts, construct an action runner or execute provisioning.

Actual result: all ten subtests passed in 57.58 seconds. S01–S09 returned
`journal.ErrReconcile` and persisted an explicit block when their original
historical completion could not be accepted by the observer. S10 reconciled its
satisfied native state, retained its identity, and passed a second resume.
No uncertain completion was silently accepted and no native action was replayed.
This does not establish automatic recovery for the nine blocked histories; an
operator must preserve evidence and resolve their stage-specific review boundary.

Reproduction after building the test binary with Go 1.27.3:

```bash
go test -c -o provision.test ./internal/provision
# Transfer to the identified disposable host; select a fresh output directory.
sudo env ALM30_NATIVE_OUTPUT=/var/lib/almadial-alm30/NEW-RUN/copied \
  ./provision.test -test.run='^TestInstallationNativeCopiedRecovery$' -test.v
```

An earlier proposal to execute the provisioning path with removed copied receipts
was rejected by automatic approval review because it could replay provisioning.
It was not executed. The approved observer-only method above supplied native
reconciliation evidence without that risk. Initial invalid copied-journal fixture
identities were rejected before any action; the corrected fixture uses the
accepted pending-state resume template.

## Concurrency, approval and installed-state repeat execution

`TestInstallationLockProcess` coordinates a separate lock-owning process through
an explicit pipe handshake, verifies competing install/resume ownership is
rejected, then verifies acquisition after release. It uses the existing lock,
not timing sleeps or a substitute lock manager.

On the actual D host, a synchronous `flock` held the canonical `apply.lock` while
the real accepted CLI was invoked. Both `install` and `resume` rejected the
competing execution with exit 1 before provisioning. After release both succeeded
with exit 0 and `no_op: true`. Missing approval returned exit 2; incorrect approval
returned exit 1. Fresh `plan` returned `actions: []` and `no_op: true`.

Existing passing tests cover immutable configuration/source/host/manifest/graph
identity and approval changes: `TestInstallationImmutableIdentity`,
`TestResumeIdentity`, `TestM1GateIntegration_ManifestToPlanIdentity`, and
`TestM1GateIntegration_ApprovalBoundaries`. Native missing/wrong approval and lock
rejection were exercised; the wider stale-identity matrix is fixture evidence.
SHA-256 approval binds the intended plan; it is not authentication or a signature.

Native repeat commands used all original receipts and the original approved plan:

```bash
sudo /var/lib/almadial-inputs/almadial plan \
  --plan /var/lib/almadial-inputs/plan.json \
  --manifest /var/lib/almadial-inputs/stack.lock.yaml
sudo /var/lib/almadial-inputs/almadial resume \
  --plan /var/lib/almadial-inputs/plan.json \
  --manifest /var/lib/almadial-inputs/stack.lock.yaml \
  --approve-sha256 abc65d197c89b958ae9a651bd01ff641a6906e6dd8e250c6e0b925ba3c6069f8
```

The accepted native binary SHA-256 was
`30523badad6a726a11c512ca3e192914a7ad21562f398ca749164e19fffa3d45`.
Production code is unchanged; the new test binary contains only qualification tests.

## Meaningful-state preservation and functional integrity

Before/after comparisons preserved 102 managed files, including canonical journals,
receipts, Asterisk/Apache/PHP configuration, cron, firewall, TLS material and six
existing recording files. A separate NO_OP comparison also preserved their modes,
UIDs and GIDs. Package inventory, effective nftables digest, SELinux Enforcing,
service invocation/start identities and restart counters were unchanged. No
package transaction, rebuild, service activation/reload, certificate rotation,
firewall change or recording deletion was required.

The first combined database digest differed. A controlled fresh NO_OP comparison
isolated 531 column/catalog/grant/recording checks: only
`vicidial_users.last_login_date` and `vicidial_users.modify_stamp` changed; the
other 529 checks remained equal. The pinned application's `functions.php` updates
`last_login_date=NOW()` after authentication; the row's modification timestamp
follows that update. Existing S07/S08 authenticated health observations exercise
this native login path. Therefore **NO_OP does not mean zero database writes by
the application**: login audit timestamps advance during authenticated health
checks. This qualification does not hide that difference or change upstream code.
Phone rows (including legacy identities), credentials, settings, catalog, grants
and recording metadata remained equal in the controlled repeat comparison.
The earlier aggregate digest cannot retrospectively identify its changed columns;
the subsequent per-column reproduction provides the attributed evidence.

Fresh unified assessment passed mandatory stage postconditions. The returned
operational report showed MariaDB/schema 1748, Asterisk/modules, Apache/PHP,
FastAGI parent plus five workers, trusted loopback HTTPS/WSS and S09 policy healthy.
Updater/listener remain deferred, cron inactive and PJSIP configuration preserved.
The short laboratory certificate yields the expected expiration WARN; a browser
registration need not persist after the accepted browser session ends. No new
call/media acceptance was claimed.

## Regression results

All executed on Go 1.27.3, Linux amd64:

| Check | Actual result |
|---|---|
| Focused injection/process-lock tests | PASS |
| Explicit native copied recovery | PASS: 10 subtests |
| `go vet ./...` | Exit 0 |
| `go test -json ./...` | Exit 0; 1,456 passing test/subtest events, 21 packages, 2 skipped tests |
| `go test -race -json ./...` | Exit 0; same passing event count |
| `go mod verify` / `go mod tidy -diff` | Exit 0 / exit 0 |
| Scaffold validation | Exit 0 |
| Python unittest discovery | 222 tests, exit 0 |
| CLI smoke | 14 cases PASS |
| PTY/nonterminal journeys | 11 PASS |
| Linux amd64 reproducibility | 12 command targets, two matching builds each |
| Formatting / `git diff --check` | PASS |
| Hosted CI | NOT_RUN |

The initial sandboxed Go run could not bind the loopback socket used by an
existing HTTP test. The same complete suite passed with authorized host execution;
no implementation change was needed. Normal test runs skip the explicitly gated
native qualification; its separate executed result is recorded above.

## Evidence and handoff

Protected supplement:
`/home/ra/almadial-lab/evidence/m4-02/20261112T155041Z-alm30/`.
It contains native copied journals, plan/approval/lock/NO_OP results, state hashes,
per-column database digests, regression logs, terminal results and reproducible
build hashes. Private file contents, credentials and recording samples are not
exported. `SHA256SUMS` covers the bounded supplement. Historical evidence is unchanged.

Task-owned repository changes are the two `install_recovery_*linux_test.go` files
and this report. Unrelated operator/QA/local AI files remain untouched. ALM-30 is
handed to independent AGY review only; no downstream task is started. The native
safe-block outcomes and normal login-audit timestamp changes are explicit review
boundaries, not claims of universal automatic recovery or byte-identical databases.
