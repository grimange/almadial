# AlmaDial — Milestone M2 (Native Backend Installation) Independent Integration & Acceptance Gate Report

- **Milestone:** M2 — Native Backend Installation
- **Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Review Scope:** Work packages ALM-16 through ALM-22 (M2-01 through M2-07; Stages S01 through S06 and cross-stage recovery)
- **Independent Gate Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Repository Baseline Commit:** `34ed58b7f2f8083fe22ead4653888c448f5524e2` (Accepted ALM-22 commit)
- **Date:** 2026-11-10T23:50:00+08:00 (2026-11-10T15:50:00Z)
- **Master Gate Evidence Ledger:** `/home/ra/almadial-lab/evidence/m2-gate/20261110T154706Z-m2gate/SHA256SUMS` (SHA-256: `018497180cd39f2d424e67bef407f285ecc646a418621aa5e95079c4cf814dad`, 7 entries verified)
- **Final Milestone Verdict:** `ALMADIAL_M2_INTEGRATION_GATE_ACCEPTED`

---

## 1. Milestone Identity and Governance

### 1.1 Authority and Roles
- **Product Goal:** AlmaDial is a simple, reliable, single-server VICIdial installer for AlmaLinux 10.
- **Milestone Objective:** M2 delivers the core native backend telephony and database stack on AlmaLinux 10: approved RPM packages, service identities, storage permissions, SELinux policies, local MariaDB with guarded schema bootstrap, pinned Asterisk 22.12.0 with 40 required native modules, pinned VICIdial r4037 application files with pure-Perl Net::Telnet 3.05, native systemd supervision of FastAGI, deferred cron definitions, and cross-stage recovery validation.
- **Governance Finding:** In Kaneo project AlmaDial (`ghritpte6e40o5ayktgz0vh1`), all seven M2 work packages (ALM-16 through ALM-22) are confirmed in `done` (Accepted) status with Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) strictly preserved as the implementation owner. As with Milestone M1, no separate standalone milestone-gate task entity exists in the native Kaneo tracker. The milestone gate is formally audited, sealed, and documented through this independent cross-stage integration review, protected evidence ledger, and Git acceptance commit.

### 1.2 Accepted Work-Package Inventory

All seven M2 work packages have been independently reviewed, accepted, and recorded in Git ancestry:

| Work Package | Stage | Kaneo ID | Status | Assignee | Accepted Commit | QA Report |
|---|---|---|---|---|---|---|
| **ALM-16 / M2-01** — Approved Repos & RPMs | S01 | `g3ibghcq6p3y55o1bczk5a7n` | `done` | Codex | `36c1b6eb8e183e241e1cb58d3e7c0fa95ad73e1a` | `docs/reviews/m2-01-repository-and-rpm-provisioning-independent-qa.md` |
| **ALM-17 / M2-02** — Identities, Storage & SELinux | S02 | `ou5231ijs6ilp28howhd6xs1` | `done` | Codex | `39dcfb4f965c4a9a5cdf351b4290702b974bf4ac` | `docs/reviews/m2-02-service-identities-storage-and-selinux-independent-qa.md` |
| **ALM-18 / M2-03** — MariaDB & Schema Bootstrap | S03 | `ldl2ak2bk3pcq7bwvk9w54wc` | `done` | Codex | `7b3aa6e7349c8a32658c354d5cc86953e2ec16ec` | `docs/reviews/m2-03-local-mariadb-and-guarded-schema-bootstrap-independent-qa.md` |
| **ALM-19 / M2-04** — Pinned Asterisk & Modules | S04 | `nwk5zbmgwuuhsnlsh92sxl1v` | `done` | Codex | `34509d29c21e519a0cea9253234d71b7b9e22cee` | `docs/reviews/m2-04-pinned-asterisk-and-native-modules-independent-qa.md` |
| **ALM-20 / M2-05** — VICIdial App Integration | S05 | `jj53k7whedrsmfdamutw1wgz` | `done` | Codex | `0f36041d566e25699656160f502f73bafceec664` | `docs/reviews/m2-05-pinned-vicidial-and-approved-configuration-integration-independent-qa.md` |
| **ALM-21 / M2-06** — Runtime Persistence & FastAGI | S06 | `zoueucy7dz57wpf1jbbgwp8m` | `done` | Codex | `86685e7a594ee2863cbd0357dd948803651e0b6e` | `docs/reviews/m2-06-cron-keepalive-and-persistent-runtime-services-independent-qa.md` |
| **ALM-22 / M2-07** — Cross-Stage Recovery QA | Integration | `g1jf2c808sylx8twxhzas9oc` | `done` | Codex | `34ed58b7f2f8083fe22ead4653888c448f5524e2` | `docs/reviews/m2-07-cross-stage-recovery-independent-qa.md` |

---

## 2. Native Backend Integration Matrix

The six independently qualified backend stages have verified dependency and recovery contracts. This matrix does not establish a unified clean-host installation command. The accepted VM evidence identifies AlmaLinux **10.2 (Lavender Lion), x86_64**, kernel **`6.12.0-211.64.1.el10_2.x86_64`**; these are historical qualification facts, not a new live-host inspection:

| Stage | Responsibility | Acceptance & Integration Postconditions | Laboratory Host Status (`almadial-alm16-rpm`) |
|---|---|---|---|
| **S01** | RPM dependencies & repositories | Validated GPG trust; persistent AlmaLinux BaseOS, AppStream, CRB and Extras definitions; 725-RPM qualified inventory. Six Perl RPMs used transaction-local `epel-almadial`, without a persistent EPEL repo file; all nine pre-existing Alma repo files were preserved. | **SATISFIED** (725 RPMs, SHA-256 `bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`). |
| **S02** | Identities, storage & SELinux | `asterisk:asterisk` account/group; `/var/spool/asterisk/monitor` and `monitorDONE` mode 0750; SELinux `Enforcing`. | **SATISFIED** (Identities, 0750 permissions, SELinux Enforcing, 0 AVC denials). |
| **S03** | MariaDB & schema bootstrap | MariaDB 10.11.18, schema 1748, 345 tables, 1,011 indexes, local-only socket grant (`almadial@localhost`). | **SATISFIED** (Catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, legacy phones `callin` and `gs102`, preserved across all 107 columns, SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`). |
| **S04** | Asterisk 22.12.0 & native modules | Pinned source build, 40 required native modules loaded, zero public SIP listeners. | **SATISFIED** (Asterisk 22.12.0, 40 modules loaded, `pjsip show transports` $\rightarrow$ `No objects found.`). |
| **S05** | VICIdial r4037 application | Pinned SVN r4037 deployed (2,153 files), Net::Telnet 3.05, loopback Apache/PHP-FPM, AMI probe account. | **SATISFIED** (App deployed, web endpoints loopback-only, AMI Login/Ping/CoreStatus pass, Command rejected). |
| **S06** | Persistent runtime & FastAGI | Systemd FastAGI (1 parent, 5 native prefork children), `127.0.0.1:4577`, `/var/lib/mysql/mysql.sock` readiness, 0 active application cron jobs. | **SATISFIED** (FastAGI active/enabled, updater/listener configured but disabled/inactive with `RefuseManualStart=yes`, crond active). |

---

## 3. Recovery, Locking, and Idempotency Audit

### 3.1 Complete-State Reconciliation Without Replay
- When an interrupted journal state is resumed against an already satisfied host, the Go runner validates observed postconditions and transitions the journal action state to `SUCCEEDED` without re-executing mutations (`calls: 0`, 0 traces emitted).

### 3.2 Incomplete Upstream Dependency Blocking
- Injected upstream defects (missing RPM in S01, missing storage directory in S02, reduced table count in S03, unloaded native module in S04, absent application configuration in S05) hard-block downstream stage preparation.
- Observation digests are updated to reflect defects, proving that downstream stages enforce actual postcondition contracts rather than relying on cached digest matches.

### 3.3 Stale Identity Rejection
- Discrepancies in configuration, manifest, plan, target host, execution graph, or action metadata trigger immediate failure with `journal.ErrIdentity`. On-disk journal files and host files are preserved without modification.

### 3.4 Execution Locking Contract
- Exclusive file locking is enforced via `syscall.Flock(apply.lock, LOCK_EX|LOCK_NB)` within the installation state directory. Both S03 (`ExecuteDatabaseLab`) and S06 (`ExecuteRuntimeLab`) reject lock contention with `journal.ErrBusy` before initiating runner calls.
- **Scope Classification:** The lock is directory-scoped to the designated installation state directory (e.g. `/var/lib/almadial/...`). This satisfies the single-installation safety contract for AlmaDial and is classified as an accepted design boundary.

### 3.5 Actual Host Fresh NO_OP Plan
- Execution of `almadial-runtime inspect` and `plan` on the live VM `almadial-alm16-rpm` produced an empty action graph `[]` (0 mutating actions) with stage digest:
  `142ce473799409602dbb3008e910f0920e9b6815a217fbb6a8f3fb6fbb4ea768`
- Fresh inspection and planning required no provisioning actions: zero package transactions, database imports, Asterisk builds, installed-configuration changes or service restarts. Inspection exports and evidence files are not runtime mutations. This was a stage assessment/NO_OP plan, not a unified installer execution.

---

## 4. Automated Regressions and Reproducible Builds

### 4.1 Test Suites Summary

These are the accepted historical regression results; this documentation erratum did not rerun the suites.

- **Go 1.27.3 Test Suite:** 207 top-level tests, 1,337 total pass events (100% PASS).
- **Go Race Suite:** 207 top-level tests, 1,337 pass events (clean, 0 data races).
- **Go Code Hygiene:** `go vet ./...`, `go fmt ./...`, `make fmt-check`, `go mod verify`, and `go mod tidy -diff` all pass (exit 0).
- **Python Test Suite:** 181 unit and integration tests (100% PASS).
- **Scaffold Validation:** 11 required files, 4 skills (PASS).
- **Candidate Stack & Config Verification:** Verified consistent candidate manifest and generated configurations (PASS).
- **PTY Terminal Wizard:** 11 executable terminal and nonterminal journeys (PASS).
- **Git Whitespace:** `git diff --check` clean.

### 4.2 Reproducible Linux amd64 Builds
Independent builds in isolated directories verified byte-for-byte reproducibility across all seven command targets:
- `cmd/almadial`: SHA-256 `41f842fc3bae3cf2a2ca943929384c71af9d5190b4a3bc296361480bd56ccea6`
- `cmd/almadial-runtime`: SHA-256 `c197cdbbe3ab42ec8a36b57a54753f6fcabda4ff69fb3864177fa028a639f00a`
- `cmd/almadial-rpm`: SHA-256 `d17d5071eb01d1a0d64f4039f6743c8fb1a7b542718d7bffae8ea299bfa0140b`
- `cmd/almadial-system`: SHA-256 `15338e72a736561bd14438183f9a1fd0100e428463b59a70f72e84bcf1872ed5`
- `cmd/almadial-db`: SHA-256 `3c4c22f2ca24fdf3fa5429ac75c76e651a4739f4c12f35deffd04117559fc99c`
- `cmd/almadial-asterisk`: SHA-256 `85ea2ef3d52878f1ffdc34f337daca7406793526c7c407b310a2ef3d7ab850da`
- `cmd/almadial-vicidial`: SHA-256 `e9ec17b719d51a0c4349d8165f63b7ad5222dd41309338730f2b1e3ac4ea6c3d`

---

## 5. Milestone Boundaries and Accepted Limitations

### 5.1 Completed in Milestone M2
- Pinned RPM dependency management and repository GPG verification on AlmaLinux 10.
- Service account (`asterisk:asterisk`), recording storage directories (`/var/spool/asterisk/monitor` and `monitorDONE`), and SELinux Enforcing policy.
- Local MariaDB 10.11.18 installation, secure socket transport (`/var/lib/mysql/mysql.sock`), least-privilege grants (`almadial@localhost`), and guarded initial schema bootstrap (version 1748, 345 tables, 1,011 indexes).
- Pinned Asterisk 22.12.0 build with 40 required native modules under systemd supervision.
- Pinned VICIdial SVN r4037 application deployment (2,153 files) with pure-Perl Net::Telnet 3.05.
- Local Apache/PHP-FPM web server integration, loopback AMI authentication with least-privilege probe permissions, and first-request web-key initialization.
- FastAGI native systemd supervision (1 parent, 5 Net::Server prefork children) with protected MariaDB socket readiness.
- Commented deferred cron definitions (keepalive and hopper) with 0 active application cron jobs.
- Cross-stage recovery, dependency gating, identity propagation, and state-directory locking.

### 5.2 Explicitly Deferred Capabilities
- **Top-Level Orchestrator:** A unified operator-facing clean-host installation path has not been demonstrated; stage qualification CLIs remain the verified boundary. No explicit approval assigning this missing capability to M4 was found. The original M2 exit-criterion discrepancy awaits scope clarification (§8).
- **Telephony & WebRTC:** M2 establishes native modules and application configuration prerequisites only. Public SIP transport, browser registration/calling, real media and production telephony are not accepted M2 capabilities. M3 owns application/browser-phone/session acceptance; external deployment qualification follows the selected network profile in the canonical plan.
- **Application and Release Boundaries:** M3 includes admin/agent flows (M3-01), browser registration (M3-04), and call/ConfBridge/media/recording acceptance (M3-05). Dialer activation remains deferred under the accepted S06 policy. M4 includes clean-VM installer replay/release qualification and unassisted operator acceptance; that does not itself assign implementation of the missing orchestrator to M4.
- **Production Manifest Promotion:** `manifests/stack.lock.yaml` remains CANDIDATE. Accepted M2 laboratory qualification does not authorize production installation or manifest promotion.
- **Optional Dynamic Secure Access Portal:** An optional operator convenience component; not part of the core AlmaDial installer.

---

## 6. Milestone Acceptance Verdict

The historical M2 technical acceptance is preserved for the following verified S01–S06 and integration results. The broader canonical all-in-one installation exit criterion has not been demonstrated; its scope discrepancy is explicitly recorded in §8:

1. All seven M2 work packages (ALM-16 through ALM-22) are independently accepted in Kaneo.
2. The individually qualified native backend stages satisfy their accepted database, Asterisk, application and runtime postconditions; end-to-end telephony is outside these results.
3. Cross-stage dependencies, recovery, identity propagation, and directory locking are proven.
4. The laboratory host `almadial-alm16-rpm` satisfies all postconditions and demonstrates a fresh NO_OP plan.
5. All automated regressions, code hygiene checks, and reproducible builds pass with zero defects.
6. Verified native provisioning and deferred product capabilities are distinguished; approval of a revised milestone boundary remains pending (§8).

**Final Milestone Verdict:** `ALMADIAL_M2_INTEGRATION_GATE_ACCEPTED`

---

## 7. Next Milestone Recommendation

- **Milestone:** **M3 — HTTPS, WebRTC, and Integration**
- **Next Task:** **ALM-23 / M3-01** (`ed6ilp2yw5ldla4ovm44i3vc`): "Provision Apache/PHP and verify application flows".
- **Accepted Gate Baseline:** `154ebfd4699c8fcff2ab945bb1b639f91e7fe936`, whose parent is accepted ALM-22 commit `34ed58b7f2f8083fe22ead4653888c448f5524e2`.
- **Recommended Worker:** Codex for implementation; Google Antigravity (AGY) for independent QA. ALM-23 remains unstarted and unassigned; this recommendation does not start or assign it.

---

## 8. Post-Acceptance Erratum — 2026-11-11

This documentation-only correction applies to accepted gate commit `154ebfd4699c8fcff2ab945bb1b639f91e7fe936`. It preserves the independent technical acceptance of ALM-16 through ALM-22 and does not initiate another acceptance cycle.

### 8.1 Provenance and Configuration Corrections

- **Git provenance:** The first three hashes originally printed in §1.2 do not resolve to Git objects in this repository. Neither do the alternative references supplied for this erratum (`27161bca…`, `da0fc251…`, `fe5d8628…`). The corrected full hashes in §1.2 exist, are ancestors of the gate commit, and match original Kaneo acceptance comments: ALM-16 `fzuqxnn9stu43gbhu27u3ufs`, ALM-17 `sm5r1kpc4u8pd9ux2jdlwec9`, ALM-18 `o58fc7vojjiagjr6zuhqpld9`. Each of these three commits contains both implementation changes and its independent QA report; they are combined implementation/acceptance commits, not competing implementation-only and QA-only identities. Their respective parents are `d851d86279e2611b3fa70d57834f5cc938d8e200`, `1996b951f68503fdabef87e0e8ba332075fb30b1`, and `a8ba1afecc54ead750c6b7c4c992cca832e1d723`. The S01, S02 and S04 QA report paths are also corrected. The gate commit adds this report on top of ALM-22; the sealed gate verdict's `accepted_commit` value identifies that reviewed ALM-22 baseline.
- **Legacy phones:** §2 now names `callin` and `gs102` and identifies the 107-column preservation digest. Sources: accepted S05 QA, S06/S07 preservation findings, and S07 `before-baseline.json`. The checked-out gate omitted the identifiers; it did not contain `cc100`/`cc101`.
- **FastAGI and cron:** §2 now states loopback port **4577**, verified by S06 QA and sealed `fastagi-runtime-verification.json`. The checked-out gate omitted the port; it did not contain 4573. `/etc/cron.d/almadial-s06` contains commented keepalive/hopper examples only: **zero active application jobs**. FastAGI is active; updater/listener remain configured, disabled and inactive. Sources: S06 QA and `cron-and-deferred-services.json`; the authorized minimal activation decision is Kaneo comment `qwmqxczkksmuxlii474t0qsw`.
- **Repositories, OS and telephony:** S01 QA distinguishes persistent Alma repositories from the temporary in-memory EPEL transaction source. Sealed S07 host/preflight observations establish AlmaLinux 10.2 (Lavender Lion) and the kernel recorded in §2. M2 modules/configuration are prerequisites, not accepted WebRTC calling or production telephony. The M3/M4 responsibility descriptions now follow the canonical plan instead of assigning all application workflows to M4.
- **Historical test count:** S07 sealed `sealed-go-test.stdout` and `sealed-go-race.stdout` each contain **207** top-level passing tests and **1,337** test/subtest pass events. §4 corrects the gate's 205 count; historical QA reports remain unchanged.

### 8.2 Milestone Scope Clarification — Approval Pending

The canonical [implementation plan](../AlmaDial_Implementation_Plan.md), §14 M2 exit criterion, still states: **“Clean all-in-one backend install is automated without undocumented manual repair and without destroying state on rerun.”** Plan history contains no amendment to that sentence. Accepted S07 implementation/QA explicitly documents the absence of a unified top-level installer; its verified complete-host path is fresh stage assessment and a NO_OP plan.

The reviewed Kaneo decisions and acceptance records acknowledge that boundary but do not explicitly supersede the original M2 exit criterion or assign orchestrator implementation to M4. In particular, ALM-22's nondestructive verification decision (`lxzonl3kn6euittfcwt4jetd`), its QA acceptance (`e0c5bzur6naaeezaalsp396v`), the M2 closure (`ucxlyz3nvmcyfcjerzcxm3js`), and M1's broad M2–M4 deferral (`ejevfmvlqp9uv7suxd65mu8i`) do not establish that specific scope change. M4's planned clean-install replay and release checks are not evidence that a unified installation path already exists.

**Verified achievement:** Individually qualified native backend provisioning stages with consistent integration and recovery contracts. **Deferred product capability:** Unified clean-host, operator-facing installation and end-to-end application/telephony readiness.

**Proposed operator clarification, not an approved decision:** Confirm whether the accepted S01–S06 technical boundary should replace the original M2 all-in-one exit criterion, and explicitly assign delivery of the missing unified installation path to a milestone/task. Preserve M3 application/telephony acceptance and M4 clean-install release qualification. Until approved, the canonical criterion remains unchanged and the milestone scope discrepancy remains unresolved. This documentation question alone does not block ALM-23; its native dependencies remain intact.

### 8.3 Verification and Unchanged State

Git object existence, parentage, changed-file lists, ancestry and original acceptance comments were checked. Historical evidence was read without modification; all ledger entries were rehashed successfully:

| Protected evidence directory (under `~/almadial-lab/evidence/`) | Entries verified | SHA-256 of `SHA256SUMS` |
|---|---|---|
| `m2-gate/20261110T154706Z-m2gate/` | 7/7 | `018497180cd39f2d424e67bef407f285ecc646a418621aa5e95079c4cf814dad` |
| `m2-06/qa-20261110T131204Z-alm21/` | 10/10 | `73744d60b54eaece6a3a19ae9394d67b7cefe9fe7a4bfe816cc832a5660bbb25` |
| `m2-07/20261110T135237Z-alm22/` | 169/169 | `ba50b10cc59f24bcacff85685cc8acb7b499b886eb02800401b0faac9ba222bf` |

The sealed gate provenance inventory contains the original incorrect references; its integrity is preserved, and this erratum corrects their interpretation without rewriting the inventory or ledger. No VM access, runtime configuration change, database mutation, service operation, installation, rebuild, reboot or functional regression rerun was performed for this erratum. ALM-16 through ALM-22 remain Accepted with Codex ownership; ALM-23 remains unstarted. Kaneo has no separate native milestone-status entity; the existing ALM-22 closure remains the M2 record.
