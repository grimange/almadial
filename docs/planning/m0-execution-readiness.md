# M0 execution readiness and first assignment

Date: 2026-11-05 (Asia/Manila). Full M0 gate remains **OPEN / NOT ACCEPTED**. ALM-1 implementation handoff: **ALMADIAL_M0_HOST_BASELINE_READY_FOR_QA**. Real host collection and implementer recovery verification executed on `almadial-alma10`; the later authorized firewall closure measured an empty kernel nftables ruleset using successful JSON/text queries. Final collector run remains accurately 20 COLLECTED / 3 firewalld-specific UNAVAILABLE, exit 1; separate real nftables evidence completes the mandatory firewall observation. Independent host acceptance and stack compatibility remain **NOT_RUN**. See [the live evidence and recovery report](../compatibility/m0-01-alma10-live-evidence.md).

Earlier live-collection assignment: the operator confirmed **no VM had been created**; [that blocked attempt and transfer protocol](../compatibility/m0-01-live-host-baseline.md) remain as history. Later assignments authorized the named VM and clean snapshot, then the bounded firewall closure. Accepted tooling commit `18b97eea1438e0d1be72276fd3018659a33a636f` ran unchanged first; two observed defects were corrected with regressions and a hashed, uncommitted rerun. The final run uses those same corrected bytes (collector SHA-256 `07a3bd79ca43a59a1ee55fc54adcce9d2964a83edf831d6cf827f7929973f282`), not a newly accepted commit. All runs remain protected outside Git. Recovery previously passed; no restoration was repeated. ALM-1 is now **QA Review**, assigned to Codex, M0 label retained and Blocked attachment removed, with MCP read-back. It is not Accepted.

## First canonical work package

- **M0-01 — Establish the clean AlmaLinux 10 VM and host inventory**
- Kaneo project AlmaDial `ghritpte6e40o5ayktgz0vh1`, project reference **ALM-1**, actual task ID `p8urvq3yjtc76rktih9xlhe3`.
- Source: plan §14 M0-01; §§4, 9.1, 20 B, 23; AGENTS.md and `prompts/M0-01-codex.md`.
- Current verified state, rechecked 2026-11-05: QA Review (`in-review`), assigned to the discovered Codex member, AlmaDial M0 label retained; Blocked attachment removed. Accepted was not entered. Evidence comment `gee7ljvyenf31ie77luzr273` was read back. The native board includes Ready; the former Ready-label fallback is deprecated.
- No prerequisite work package. Host, access, image checksums/signature, operator authority, source, clean-snapshot recovery and kernel firewall observation are evidenced for implementation handoff. Separate independent QA is the next gate. A synchronized task alone is not execution authority.

## Worker configuration and required tools

Recommend **OpenAI Codex CLI**, the highest-capability Codex model available in the authorized runtime, **High** reasoning effort. The worker must record its actual model/runtime; synchronization did not change model settings. Professional role: Principal Linux Telephony Compatibility Engineer with release-evidence responsibility.

Required tools: repository filesystem/terminal/Git, AGENTS and the `almadial-compatibility` skill, task/experiment/evidence/handoff templates, official AlmaLinux/Asterisk/VICIdial/VICIphone documentation/source, Python scaffold validation, and `kaneo-local` identity/membership/task/read-back operations. Host console/SSH, inventory commands and snapshot tooling are required only after explicit exact-target authorization.

The next worker must discover its own identity via MCP and verify membership. For a subsequently authorized assignment, self-assign before moving to In Progress and read back assignee/status. Do not assume this session's Codex member ID grants another runtime authority. Independent reviewers retain implementer ownership.

## Readiness checklist

| Prerequisite | State | Evidence or required input |
|---|---|---|
| Defined product and task acceptance | PASS for planning | Plan 1.0-draft and canonical M0-01 task description |
| Review/backlog traceability | PASS | 35 tasks, 44 proposed native dependency edges, MCP read-back |
| Reproducible documentation and tooling baseline | PASS for documentation; tooling finalization recorded separately | Documentation commit `1df743a4178fc32d9d78d53de8ea0870b1d2c30f` includes the reviewed scaffold/plan/reports and guidance/templates. Use the subsequent local commit containing [the independent tooling QA report](../reviews/m0-01-inventory-qa.md) for collector execution; verify its complete SHA and reviewed artifact digests before transfer. No push. |
| Named disposable AlmaLinux 10 host | PASS for implementer inspection | `almadial-alma10`, Multipass/QEMU; real host collection protected outside Git |
| Recoverable clean snapshot | PASS for implementer verification | `almadial-alma10.clean-baseline` restored; current collection state preserved in snapshot2; host evidence hashes unchanged |
| Exact host inspection/mutation authority | PASS for scoped assignments | Original inspection/transfer/evidence/recovery authority and later explicit nftables-package/required-dependency inspection authority are recorded separately. nftables was already present at closure entry; no installation, service activation, rule change or repeat restoration was performed by the closure worker. |
| Exact image/minor release and CPU ISA | PASS for observation | Image digest and CHECKSUM signature verified; AlmaLinux 10.2 x86_64; guest loader supports v2/v3; no stack compatibility inference |
| Repository/package/security inventory | PASS for implementation-side evidence completeness; independent QA pending | Final 23-probe run: 20 COLLECTED / 3 firewalld-specific UNAVAILABLE, exit 1, no errors. Separate successful privileged JSON/text nftables queries establish an empty kernel nftables ruleset. Firewalld absent; nftables service disabled/inactive; SELinux enforcing. No release security or stack-compatibility approval implied. |
| Independent inventory reproduction | NOT_RUN | Implementer cross-checks and recovery evidence exist; separate reviewer still required |
| Stack compatibility and lock promotion | NOT_RUN | M0-02–08 unexecuted; manifest remains candidate |

## Repository preparation slice — now implemented, host acceptance excluded

The subsequent operator assignments explicitly authorized tested executable inventory preparation, independent tooling QA, minimal defect corrections and a local tooling commit after acceptance. See [the protocol and handoff](../compatibility/m0-01-host-inventory.md), `scripts/collect_host_inventory.py`, its synthetic fixtures/tests and [the separate review](../reviews/m0-01-inventory-qa.md). The Python standard-library research utility has no default collection mode, no fixture fallback to live probes and no package/configuration mutations. Go is unavailable in this workspace; no M1 Go framework was introduced. Review acceptance is limited to this repository slice and cannot supply host evidence. The original preparation contract is retained below as its acceptance scope.

Objective: prepare an executable, minimal M0-01 collection protocol and source audit without touching a host or starting installer development.

Scope:

1. Re-read the current task and source baseline; identify any changed evidence. Read AGENTS, GOALS, PLANS, the compatibility skill and first-task prompt.
2. Draft `docs/compatibility/m0-01-host-inventory.md` with all unmeasured fields explicitly NOT_RUN, required environment/authority fields, collection commands, evidence locations and stop conditions.
3. Specify collection for image origin/signature/digest, minor OS release, architecture/CPU flags, kernel, memory/disk, enabled repository provenance, package/runtime baseline, hostname/DNS/NTP, listeners/interfaces/NAT, service state, SELinux/firewalld and snapshot/recovery details.
4. Identify the named disposable-host and snapshot information needed from the maintainer. Produce a task-specific experiment/authorization request with exact target/actions; do not execute it under this planning assignment.
5. Use official source documentation to prepare inventory rationale and safe command lists. Record mutable documentation dates and exact source identifiers where obtained. Do not choose or promote a working stack from documentation.
6. Run repository checks, write a sanitized handoff and describe the next M0-02 input.

Expected repository artifacts: `docs/compatibility/m0-01-host-inventory.md`; a task-specific collection protocol/evidence manifest with stable relative paths; a sanitized machine-readable inventory schema/example using fake data and explicit NOT_RUN markers; a handoff naming missing host authority. Avoid new generic frameworks, package managers or tooling dependencies.

Acceptance for the **preparation slice**: every required baseline field has a command/evidence method and status; no unknown is presented as measured; inventory distinguishes minor release/image/ISA; no secrets or unsafe bypasses; source baseline/digests are recorded; required host/snapshot/actions are explicit; scaffold validation passes. This slice is not full M0-01 acceptance and must not mark the canonical task Accepted.

## Full work-package execution after prerequisites

Only after explicit host-specific authority and a recoverable clean snapshot are supplied, collect the approved inventory on the named disposable host. Record exact commands, outputs, timestamp, image/repository provenance, CPU ISA, topology and security baseline. Stop on production ambiguity, absent recovery boundary, unexpected credentials or requested mutation beyond authority. Do not install packages or alter networking/security merely to fill an inventory without separate exact-target authorization. The later operator assignment explicitly authorized nftables and required dependencies only; it did not authorize service activation or rule changes.

Full artifacts: measured Markdown and sanitized machine-readable host inventory; image/package/kernel/repository evidence; topology and security-state description; host-identity/authorization/snapshot record; reproducible command log and evidence digests; independent review and next-experiment handoff.

Full acceptance: mandatory host fields are measured, provenance and ISA are established, snapshot recovery is verified and inventory is independently reproduced. Unavailable mandatory observations prevent acceptance. Every PASS is backed by actual output; missing runtime checks remain BLOCKED/NOT_RUN. Any initial requirement wording permitting marked unknowns is an inventory-completeness rule, not a waiver of mandatory host evidence.

## Independent QA and milestone gate

A separate reviewer, using the `almadial-review` skill and independent-review template, repeats critical inventory checks against the recorded snapshot, verifies sanitized evidence hashes and documents discrepancies. The implementation worker cannot self-approve. Reviewer must not take over task ownership. The native QA Review column retains the `in-review` status slug; native Accepted retains `done` and is used only after evidence review and explicit acceptance.

After accepted M0-01, M0-02 qualifies exact OS/Asterisk/VICIdial/PHP/Perl/MariaDB/phone candidates and authenticated source acquisition. M0-03–07 provide actual module, generated PJSIP/dialplan, ConfBridge allocation, AMI/AGI, dialing, two-way WebRTC media and recording evidence. M0-08 repeats the required suite on a second clean VM and proposes the reviewed manifest/validator transition. M1–M4 implementation remains blocked until the complete M0 gate is accepted.

No production access, carrier/PSTN calls, installer implementation, package installation, real credentials, blanket policy disablement, push or publication is authorized by this readiness report. The operator separately authorized the local documentation baseline commit and repository-only collector preparation; no host authority followed from that authorization.

## 2026-11-06 update — scoped M0-04 implementation evidence

The preceding sections preserve their historical readiness snapshot. Subsequent
independent reviews accepted M0-01–03; the current M0-04 assignment separately
authorizes the named disposable VM's application experiments. It does not grant
production authority or whole-stack acceptance.

See [the M0-04 runtime report](../compatibility/m0-04-vicidial-runtime.md).
The official HTTPS nightly ZIP exactly matches all 2,511 pinned r4037 files,
resolving that artifact's publisher-origin hold. MariaDB 10.11.18 imports the
reviewed schema derivation to version 1748; PHP 8.3.36 executes bounded admin and
agent application requests; Perl 5.40.2 loads required modules and executes a
controlled empty-queue sender. Schema grant separation and the reproduced process
query truncation correction are explicit, hash-pinned derivations. Original source
and failed evidence remain preserved. A pre-VICIdial snapshot exists; it was not
restored. SELinux remains enforcing and accepted Asterisk files remain unchanged.

The agent request test uses a seeded, nontelephony application session. Full phone
login, AMI/AGI operation, generated PJSIP/ConfBridge configuration, dialing and
recording remain downstream experiments. M0-04 requires a separate reviewer;
the implementing Codex worker cannot grant acceptance. The complete manifest
remains candidate, and M1–M4 stack automation remains gated on complete M0 evidence.
