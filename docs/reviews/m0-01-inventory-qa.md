# Independent QA — M0-01 inventory tooling

**Reviewer:** Codex independent QA agent `/root/inventory_independent_qa`; separate from implementation worker.  
**Implementer (must differ for independent claim):** Codex root worker, previous preparation turn; fixes, if requested, remain with that worker.  
**Baseline commit / change commit:** `1df743a4178fc32d9d78d53de8ea0870b1d2c30f`; reviewed additions are initially uncommitted. No Git mutations by reviewer.  
**Evidence reviewed:** `AGENTS.md`, review skill, `docs/GOALS.md`, `docs/PLANS.md`, implementation-plan §§14/16 and M0-01 references; collector, all 23 probes, synthetic fixture, inventory tests, preparation handoff, execution-readiness report and both Kaneo synchronization reports.  
**Verdict:** **ACCEPT — repository-only inventory tooling slice**, after the implementer corrected R1 and the separate reviewer reproduced the correction. Initial review was REJECT; the original failure evidence is retained below. Full M0-01 host baseline and compatibility remain NOT_RUN.  
**Review environment:** Repository-only developer checkout; Python 3.14.4; review date 2026-11-05 Asia/Manila. No disposable lab target was supplied. Python 3.12/AlmaLinux runtime remains NOT_RUN.

## Initial reproduced tests

| Test | Expected | Actual | Status | Evidence |
|---|---|---|---|---|
| Baseline identity | Expected documentation commit | HEAD matches `1df743a4178fc32d9d78d53de8ea0870b1d2c30f` | PASS | `git rev-parse HEAD`; status inspected before tests |
| Scaffold validation | Required guidance valid | 11 required files, 4 skills | PASS | `python3 scripts/validate_scaffold.py` |
| Full existing unit discovery | All offline tests pass | 19/19 pass, including fully mocked OS-gate test; no live probe executes | PASS | `python3 -m unittest discover -s scripts/tests -v` |
| Syntax compilation | Collector/tests compile | Exit 0 | PASS | `python3 -m py_compile scripts/collect_host_inventory.py scripts/tests/test_collect_host_inventory.py` |
| Fresh fixture CLI | New private JSON/Markdown; simulated provenance | Exit 0, 23 COLLECTED simulated probes, current collector digest; all acceptance fields NOT_RUN; directory 0700, files 0600 | PASS | Independent harness invokes CLI in a fresh temporary directory |
| Fixture isolation | No host fallback or subprocess execution | Complete, empty and failed fixtures work with `live_probe` and `Popen` patched to raise | PASS | Independent harness `fixture_isolation` |
| Malformed fixtures | Exit 2 without evidence creation | Null, array, wrong probes shape, Boolean schema, unknown probe and malformed JSON rejected | PASS | Independent harness `negative_cli` |
| Missing fixture observation | Partial collection | Deleted CPU observation gives CLI exit 1 | PASS | Independent harness `negative_cli` |
| Existing destination/symlink | Refuse without overwriting | Exit 2; existing JSON/Markdown bytes unchanged; symlink refused | PASS | Independent harness `negative_cli` |
| Failed probe diagnostics | Structured error without raw text | Missing command UNAVAILABLE; timeout/nonzero/missing exit ERROR; facts null and stderr sentinel absent | PASS | Independent harness `failed_probe_classification` |
| Controlled runner timeout | Bound synthetic child duration | Sleeping Python child killed; timeout returned in approximately 0.062 s for 0.06 s deadline | PASS | Independent harness `controlled_runner`; no inventory program executed |
| Controlled runner command/output/environment | Missing command categorized; combined output bounded; fixed environment | Missing synthetic path categorized; 2048-byte cap held; only five explicit environment keys retained | PASS | Independent harness `controlled_runner` |
| Redaction | Sensitive synthetic inputs absent in both formats | Host/domain/IP/MAC/private path/credential/repository/interface sentinels absent; stderr counts retained; HMAC stable within run and different across keys | PASS | Independent harness `redaction` |
| Incomplete values | Missing or malformed required facts are partial/error | Time contains unknown states/empty timezone, chrony only stratum, services only IDs, disk/mount empty objects and interface only name all COLLECTED; modified complete fixture exits 0 | FAIL | Independent harness `incomplete_values_observation`; finding R1 |
| Canonical service aliases | Observation only; runtime qualification deferred | Synthetic duplicate canonical `Id=mariadb.service` produces ERROR/parse_error | NOT_RUN for real host | Independent harness `systemd_alias_observation`; optional R2 |
| Artifact/source hashes | Reported hashes match bytes | All three artifact hashes match; both source plans byte-identical with recorded digest | PASS | Independent SHA-256 calculation against ledger |
| Tracked whitespace | No malformed tracked diff | Exit 0 | PASS | `git diff --check` |
| AlmaLinux host, package provenance, CPU ISA, security/runtime and telephony acceptance | Separately authorized measured evidence | No host collection or runtime tests performed | NOT_RUN | Exact-target authority/snapshot absent; intentionally out of scope |

The reviewer wrote and executed `/tmp/almadial-independent-qa.py`; its detailed local result record is `/tmp/almadial-independent-qa-results.json`. These are disposable local reproduction aids, not committed host evidence. The substantive inputs and observed outcomes are recorded above and below. The original unit suite's call to `inventory()` without a fixture is completely intercepted by its `live_probe` mock; the root explicitly clarified that mocked control-flow testing was allowed. The independent harness always supplies a fixture to `inventory()`.

## Findings

| Severity | File:line or component | Observed failure | Reproduction | Bounded correction |
|---|---|---|---|---|
| MEDIUM — R1, resolved in independent re-review | `scripts/collect_host_inventory.py:134`, `:151`, `:241`, `:247`, `:264`, `:415`, `:497` | Required facts can be absent or invalid while the probe is COLLECTED and full fixture CLI exits 0. A time probe with `NTPSynchronized=unexpected`, `LocalRTC=unexpected` and empty `Timezone` exports unknown states; chrony containing only `Stratum : 3` omits offsets/leap state; services containing only all seven `Id=` lines export entirely unknown service states. Disk/mount `[{}]` and an interface containing only its name also pass. This conceals incomplete/malformed observations in the collection-success signal, contrary to the handoff's partial/error behavior. Acceptance fields remain NOT_RUN, so this is a tooling-completeness bug, not an improper host PASS. | Replace the corresponding synthetic fixture stdout values with the examples above; all affected rows are COLLECTED. Combined CLI replay exits 0. | Validate required fields and recognized values before COLLECTED; reject incomplete/malformed required data as ERROR/parse_error and CLI exit 1. Preserve legitimate optional/null observations such as a whole disk's null filesystem, an interface without assigned addresses, omitted optional route metrics, and empty enablement for a not-found unit. Add focused incomplete-value regressions. |
| LOW — R2, optional qualification | `scripts/collect_host_inventory.py:268`, `:274` | Exact requested-name/canonical-Id matching rejects alias-shaped data when `mysqld.service` resolves to `mariadb.service`. The synthetic case is confirmed; no real systemd alias behavior was measured. | Replace only `Id=mysqld.service` with `Id=mariadb.service` in the fixture; service probe becomes ERROR/parse_error. | Qualify this behavior on the eventual authorized lab. If relevant, preserve requested identity through a bounded alias-aware query/parse contract. Do not claim this synthetic case proves AlmaLinux behavior. |

No command-catalog mutation, shell execution, inherited credentials, unsafe output overwrite, raw diagnostic export, fixture-to-live fallback or unsupported compatibility promotion was found. All 18 command argv entries are fixed read/query operations; the other five probes read OS/memory/DNS, repository configuration and fixed binary-presence paths. Subprocess time and combined output are bounded. Files are read as data, output filenames are fixed, and existing final directories/symlinks are refused. Parent directories remain an explicitly documented operator-controlled prerequisite. Some firewall semantics and package/source provenance intentionally remain NOT_RUN rather than inferred from opaque or incomplete authority.

Initial reviewed artifact hashes:

| Path | SHA-256 |
|---|---|
| `scripts/collect_host_inventory.py` | `6433b68fc64f156ac3a351a1ec088e0cfe360ad5698a64924079c8433ccc2b1e` |
| `scripts/tests/test_collect_host_inventory.py` | `8fca2b989c59aeb88360498e4df3245c4771df013b8b162f131fabb1e07aea00` |
| `scripts/tests/fixtures/inventory/almalinux10.json` | `c86c3963a7719087cb5b2a36c17a912a47cff72ead44786cf142bde4b1c4199d` |

Source plan SHA-256: `2a8effec4e1e40d3577382158303d4e3124e56712e9877d3cac387a68bc0cfd4`, independently confirmed for both plan paths. Documentation-only synchronization history was read for consistency; external Kaneo state was not queried or changed by this reviewer.

## Independent re-review after correction

The implementation worker made the bounded R1 correction and added focused tests; this reviewer did not alter implementation or tests. Required storage/interface/route/time/chrony/service data now undergoes explicit validation, and missing/invalid required facts yield ERROR/parse_error. Valid nullable mount metrics, whole-disk filesystem absence, empty interface address arrays, negative routes without a device and not-found service enablement remain distinguishable. Chrony now retains the offset direction.

The initial `/tmp` harness was re-executed against the corrected source. Its incomplete-input observations now contain ERROR/null facts for memory, disks, mounts, interfaces, routes, time, chrony and services; the combined modified fixture exits 1. An explicit empty repository configuration remains a collected empty fact with remote verification NOT_RUN and is not an acceptance claim.

| Re-review check | Actual result | Status |
|---|---|---|
| `python3 scripts/validate_scaffold.py` | 11 required files, 4 skills | PASS |
| `python3 -m unittest discover -s scripts/tests -v` | 22/22 pass, including 18 inventory tests and 4 scaffold regressions | PASS |
| `python3 -m py_compile scripts/collect_host_inventory.py scripts/tests/test_collect_host_inventory.py` | Exit 0 | PASS |
| Fresh CLI and original independent harness | 23 simulated COLLECTED facts on complete fixture; all acceptance fields NOT_RUN; output permissions/hash correct; no host fallback | PASS |
| Separate independent per-probe malformed CLI replay | Nine cases independently run: memory, disks, mounts, interfaces, IPv4 route, IPv6 route, time, chrony and services. Each exits 1, writes ERROR/parse_error/null facts to JSON and ERROR in Markdown; live probe and Popen forbidden by mocks | PASS |
| Valid absent/nullable facts | DOWN interface without addresses, blackhole default route without dev and proc mount with null capacity remain COLLECTED; not-found enablement is not_applicable and chrony direction retained | PASS |
| Redaction / controlled runner / failed-probe categories | Original independent negative harness re-executed successfully, including timeout, command absence, combined output cap, fixed environment, stderr protection and HMAC behavior | PASS |
| Final output/path/file checks | Broken final symlink refused before inventory; oversized and invalid-UTF8 local synthetic files rejected; existing directory/symlink left unchanged | PASS |
| `git diff --check` | Exit 0 | PASS |

Reviewed corrected artifact hashes:

| Path | SHA-256 |
|---|---|
| `scripts/collect_host_inventory.py` | `b894e1d43500162ee59464f0290853235e6f1a8b39fcfdf34f8c4a8c6f5bb5f0` |
| `scripts/tests/test_collect_host_inventory.py` | `45340233e7559fe5b62331f06b32f21af3fd1502196ab343d21125aeb31c0376` |
| `scripts/tests/fixtures/inventory/almalinux10.json` | `c86c3963a7719087cb5b2a36c17a912a47cff72ead44786cf142bde4b1c4199d` |

R1 is resolved. R2 remains an optional lab qualification, not a confirmed compatibility failure or an unresolved tooling acceptance blocker. There are no remaining confirmed blocking findings in this repository-only tooling slice. No host actions were added during correction/re-review.

## Decision

- Acceptance criteria satisfied: repository-only isolation, explicit collection mode/authority references, synthetic status/provenance, artifact hashes, redaction, private fresh outputs, bounded command execution, existing offline regressions and preserved M0 gate.
- Missing evidence / environmental blockers: none for the bounded repository tooling acceptance after R1 re-review. Named disposable host/image/snapshot/authority and independently reproduced baseline remain absent for full M0-01.
- Host/root actions taken: none. No `--collect`, real live probe, host inventory subprocess, remote action, root command, server mutation, Kaneo write or Git mutation.
- What remains NOT_RUN: AlmaLinux/Python 3.12 live behavior, image/signature/repository/package provenance, CPU ISA qualification, snapshot restoration, host/network/security baseline and all compatibility/call/media/recording tests.
- Milestone gate: **OPEN**. Tooling acceptance alone cannot close M0-01 or the full M0 gate, select/promote a stack, mark ALM-1 Accepted or start M1–M4 installation automation.
- Follow-up task: root completes supporting documentation/governance checks and the separately authorized local commit of this accepted tooling snapshot. Obtain separate exact-target lab authorization, image/snapshot evidence and reviewed tooling revision before any host collection; qualify systemd alias behavior then.
