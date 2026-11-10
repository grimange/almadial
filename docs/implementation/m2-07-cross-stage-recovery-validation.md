# M2-07 — Cross-stage recovery validation

Developer verification: **PASS; independent AGY review required**.
Task: ALM-22 (`g1jf2c808sylx8twxhzas9oc`). Starting Git HEAD:
`86685e7a594ee2863cbd0357dd948803651e0b6e`.

## Scope and integration boundary

The accepted S01–S06 implementations have compatible nested input, observation,
plan and recovery contracts. This task adds five focused integration tests in
`internal/provision/backend_integration_test.go`; no provisioner, supervisor,
inspector, journal or orchestration implementation changes were needed.

There is no complete top-level native installation orchestrator. The stage
qualification CLIs and their explicit parent inputs remain the supported
integration boundary. This report establishes their dependency and recovery
compatibility, not a new end-to-end installation command or milestone acceptance.
The shipped manifest remains CANDIDATE; production installation stays blocked.

The current task explicitly authorizes nondestructive assessment of the existing
disposable VM and reuse of accepted interruption evidence. Canonical failure
injection requirements are covered by that evidence plus focused rootless
fixtures; no new live interruption campaign, installation, restart or reboot was
performed. Kaneo decision comment `lxzonl3kn6euittfcwt4jetd` records this boundary.
Independent review of the M1/M2 contracts remains an AGY acceptance prerequisite.

## Dependency and fresh-state matrix

| Stage | Actual contract | Fresh installed-host result |
| --- | --- | --- |
| S01 | `Snapshot.ValidateTrust` and `Assess` establish approved repositories, keys and direct package requirements. | SATISFIED: 725 RPMs; accepted inventory hash unchanged. |
| S02 | `SystemInputs` binds the immutable M1 configuration/preflight/manifest plan to matching S01 inventory. `AssessSystem` requires S01, service identities, managed directory metadata, Enforcing SELinux and the approved recording budget. | SATISFIED through the existing native foundation inspector. |
| S03 | `DatabaseInputs.Prepare` requires satisfied S02 and matching host/foundation/source and protected credential references. `VerifyDatabaseForApplication` exposes read-only complete-schema recognition to S05. | SATISFIED: MariaDB 10.11.18, schema 1748, accepted catalog, authentication and grants. |
| S04 | `AsteriskInputs.Prepare` requires satisfied S02/S03 and binds their snapshots, host and verified source. Binary existence alone does not satisfy the module/runtime contract. | SATISFIED: Asterisk 22.12.0 and all 40 required native modules. |
| S05 | `VICIdialInputs.Prepare` invokes S04 preparation, requires satisfied S04 plus `VerifyDatabaseForApplication`, and binds deployment/Perl sources and foundation digests. | SATISFIED: r4037 deployment, protected configuration, local web/AMI integration and accepted generated files. |
| S06 | `RuntimeInputs.Prepare` invokes S05 preparation, requires satisfied application configuration, and binds host/application/foreign-schedule observations. | SATISFIED: exact managed files, one FastAGI parent/five children, approved identities and deferred jobs inactive. |

The installed-host assessment uses `RuntimeNative.InspectRuntime`. Its closed
runtime guard runs before the existing private S06 handoff to
`VICIdialNative.inspectVICIdial`; that path freshly rechecks the native database,
system/RPM, Asterisk and application foundations. Immutable parent JSON remains
historical input, not evidence that today's host is healthy. The public S05
inspector retains its pre-S06 inactive-job restriction; it is not used to reject
legitimate accepted FastAGI activity or to justify stopping that activity.

The handoff recognizes only accepted S05 database/configuration transitions and
accepted service restart receipts. It freshly verifies current runtime identity
and health; PID equality across a reboot is not a postcondition.

S01 resolution is an explicit metadata/cache operation, not an installed-host
NO_OP assessment. This qualification calls existing read-only inventory checks
instead of scheduling DNF resolution. Complete synthetic S02–S06 inputs each
produce zero actions. Two complete native S06 inspections and pure plans were
identical, with zero actions and stage digest:
`142ce473799409602dbb3008e910f0920e9b6815a217fbb6a8f3fb6fbb4ea768`.
This is a fresh NO_OP **plan** on the actual host; no native mutation executor was
invoked for ALM-22. The rootless integration fixture additionally executes its
empty plan and asserts `no_op=true`, zero traces and zero runner calls.

## Recovery and failure-injection matrix

| Case | Evidence and actual outcome |
| --- | --- |
| Complete chain | `TestBackendCompleteChain`: existing S01/S02 assessment satisfied; S02–S06 empty graphs, deterministic repeated S06 plan and execution without effects. PASS. |
| A: complete state, interrupted copied journal | `TestBackendCopiedJournalRecovery/interrupted`: copy a validated completed fixture journal, replace its final outcome with RUNNING, retain complete observed state, resume through the real S06 executor/journal. Final outcome reconciles to SUCCEEDED, with no runner call or state change. PASS. |
| B: incomplete upstream | `TestBackendUpstreamBlocksDependents`: absent required GCC, missing managed directory, partial database, missing loaded Asterisk module, or absent S05 configuration. Descendant snapshot bindings are updated, so rejection proves postcondition enforcement rather than merely stale digests. S03/S05/S06 reject at applicable boundaries and S06 emits no usable plan. PASS. |
| C: stale recovery identity | Copied journals with changed configuration, manifest, plan, host, graph or action metadata fail with the corresponding journal identity reason; rejected journal bytes and observed state remain unchanged. PASS. |
| Material input changes | `TestBackendMaterialIdentityPropagates`: configuration changes and a valid candidate-manifest document identity change propagate to dependent plans; old authorization cannot execute them. The manifest fixture changes a comment, preserving the pinned stack policy. PASS. |
| Concurrent execution | `TestBackendCrossStageExecutionLock`: both S03 and S06 reject an existing installation-directory lock with `journal.ErrBusy`, before any mutation runner call. PASS. |

The accepted lock is scoped to the selected installation state directory, not
host-global. Callers must use the same canonical installation directory for
competing operations. Stage-specific qualification directories remain separate;
this task does not claim they collectively form a host-global installer session.

All mutating stage APIs retain explicit lab authorization, immutable
configuration/manifest/source/host/plan bindings, `journal.Acquire`, durable
`Begin` before the bounded action, fresh postcondition verification and persisted
outcomes. Existing component tests were rerun through the full regression suite;
their recovery semantics were not replaced with a universal retry policy:

| Stage | Preserved recovery decision | Accepted evidence reused |
| --- | --- | --- |
| S01 | Unfinished or unknown DNF transactions require reconciliation; no blind replay. | M2-01 actual interrupted transaction at 387 RPMs and deterministic refusal, plus `TestInterruptedProvisioning`. |
| S02 | Known completed effects reconcile; unknown ownership/metadata conflicts block. | M2-02 process kill after group creation, resumed remaining actions, native journal crash boundaries and preserved sentinel. |
| S03 | Partial schema blocks reimport; verified complete schema reconciles without SQL replay. | M2-03 bounded isolated DDL timeout/partial-table refusal and complete-schema reconciliation. |
| S04 | Complete source/build/install/config/unit/module/runtime evidence is required. | M2-04 accepted native module, restart and NO_OP evidence; existing interrupted-action fixtures. |
| S05 | Valid application files/configuration reconcile without redeployment or foreign overwrite. | M2-05 failed-postcondition recovery records and existing interrupted-action fixtures. |
| S06 | Completed state reconciles; uncertain service outcomes block; no duplicate schedules/supervisors. | M2-06 controlled stop/start, child recovery, authorized reboot and completed-journal NO_OP; ALM-22 copied-journal integration fixture. |

These sources are the corresponding implementation and independent QA reports
under `docs/implementation/` and `docs/reviews/`. Historical evidence paths and
ledger hashes are indexed in protected `preservation-results.json`; their
original artifacts were not overwritten or represented as new ALM-22 experiments.

## Actual laboratory observations and preservation

Target: disposable `almadial-alm16-rpm`, AlmaLinux 10.2 x86_64; machine ID
`0fffa9a4683f35b696f0b6ee7330734b`, independently checked before guest collection.
The existing recovery snapshot `alm21-pre-runtime-20261110` was read back; no new
snapshot, stop, restore or reboot was requested. Boot ID remained
`9ecd6163-6300-4898-8e93-262019adc5e1`.

Before/after comparisons passed for RPM inventory and repository files, database
catalog, both complete legacy phone records, protected credential file hashes
and metadata, selected campaign/lead/hopper/call/recording tables, service
identities, Asterisk configuration and module inventory, and both recording
directories (`monitor` and `monitorDONE`). Catalog SHA-256:
`10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`;
phone-record SHA-256:
`bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`.

All eight observed service PID/invocation/activation/restart-count records were
unchanged, and invocation IDs match the accepted M2-06 post-reboot witness.
MariaDB, Asterisk, Apache, PHP-FPM, crond and FastAGI remain active/enabled;
updater/listener remain inactive/disabled. Managed cron has zero active dialer
entries. Repeated native foreign-schedule digests match. No credential rotation,
phone modification, application deployment or package transaction occurred.

SELinux remains Enforcing. The boot AVC search returned no matches (ausearch
exit 1). Local TCP listeners remain `127.0.0.1` ports 80, 3306, 4577, 5038 and
8088; existing SSH and accepted Asterisk ephemeral UDP resolver sockets remain.
No PJSIP transport or deferred dialer workload was activated. Asterisk reports
zero active channels, zero active calls and zero calls processed. This is not
agent-session, RTP/media, outbound calling or production acceptance.

## Reproduction, regressions and evidence

Using Go 1.27.3, all commands below exited 0:

```bash
go test -count=1 -json ./internal/provision -run '^TestBackend'
go fmt ./...
make fmt-check
go vet ./...
go test -json ./...
go test -race -json ./...
go mod verify
go mod tidy -diff
python3 scripts/validate_scaffold.py
python3 -m unittest discover -s scripts/tests -v
python3 scripts/verify_candidate_stack.py
python3 scripts/verify_config_generation.py
git diff --check
```

Focused integration: five top-level tests / 19 passing test/subtest events.
Full ordinary and race runs: each 207 top-level tests / 1,337 passing events,
zero failures. The opt-in `TestM0Artifacts` full-archive test remains skipped;
native assessment verified the actual pinned archives/signature separately.
Python: 181 passing tests. CLI smoke: 17 passing cases. Existing executable
terminal/nonterminal journeys: 11 passing cases, using
`scripts/test_terminal_wizard.py --binary BINARY --output PRIVATE_DIRECTORY`.

All seven Linux amd64 commands were built in two isolated source directories;
each pair was byte-identical. Main CLI uses `scripts/build_go.py`; qualification
commands use `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly
-trimpath -buildvcs=false`. Runtime SHA-256 is
`c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a`, matching
the accepted guest qualifier used for native inspection. Main CLI SHA-256 is
`41f842fc3bae3cf2a2ca943929384c71af9d5190b4a3bc296361480bd56ccea6`.

Protected evidence:
`~/almadial-lab/evidence/m2-07/20261110T135237Z-alm22/`.
It includes exact native collection scripts/argv, two inspections/plans, snapshot
and governance read-back, before/after hashes, service/process/listener evidence,
focused and full regression logs, CLI/PTY outputs and all build hashes.
`SHA256SUMS` seals implementation evidence; a separate handoff ledger covers
tracker read-back. Private logs, secrets and generated binaries are not committed.

Native reproduction uses the accepted `/tmp/alm21-qualification/almadial-runtime`
`inspect --accepted-vicidial /tmp/alm20-qualification/sealed-final-inspect.json`
and `plan --input PRIVATE_INSPECTION.json` operations, with the exact eight pinned
source/manifest/signature/keyring/scratch flags retained in `alm22-assess.py` in
protected evidence (reusing `scripts/lab/m2_06_qualify.py` constants). Inspection
and pure plan create private temporary exports only; no authorization-bearing
`lab-run` or `lab-resume` operation is invoked on the actual VM for this task.

Hosted CI and independent ALM-22/M2 gate acceptance are NOT_RUN by Codex. AGY must
review the dependency matrix, recovery cases, lock scope, accepted historical
evidence and fresh native results before acceptance or M3. No downstream work,
Git commit, push, tag or release is authorized by this qualification.
