# M0 execution readiness and first assignment

Date: 2026-11-05 (Asia/Manila). Full M0 execution verdict: **BLOCKED**. Repository-only source/protocol preparation is the next bounded assignment, but no engineering work is started by synchronization.

## First canonical work package

- **M0-01 — Establish the clean AlmaLinux 10 VM and host inventory**
- Kaneo project AlmaDial `ghritpte6e40o5ayktgz0vh1`, project reference **ALM-1**, actual task ID `p8urvq3yjtc76rktih9xlhe3`.
- Source: plan §14 M0-01; §§4, 9.1, 20 B, 23; AGENTS.md and `prompts/M0-01-codex.md`.
- Current verified state, rechecked 2026-11-05: Backlog (unchanged `to-do` status slug)/unassigned, AlmaDial M0 and Blocked labels. No task is currently marked Ready. The native board now includes Ready; the former Ready-label fallback is deprecated. The two retained extra columns do not resolve M0 prerequisites.
- No prerequisite task; external prerequisites are missing. A synchronized task is not implementation authority.

## Worker configuration and required tools

Recommend **OpenAI Codex CLI**, the highest-capability Codex model available in the authorized runtime, **High** reasoning effort. The worker must record its actual model/runtime; synchronization did not change model settings. Professional role: Principal Linux Telephony Compatibility Engineer with release-evidence responsibility.

Required tools: repository filesystem/terminal/Git, AGENTS and the `almadial-compatibility` skill, task/experiment/evidence/handoff templates, official AlmaLinux/Asterisk/VICIdial/VICIphone documentation/source, Python scaffold validation, and `kaneo-local` identity/membership/task/read-back operations. Host console/SSH, inventory commands and snapshot tooling are required only after explicit exact-target authorization.

The next worker must discover its own identity via MCP and verify membership. For a subsequently authorized assignment, self-assign before moving to In Progress and read back assignee/status. Do not assume this session's Codex member ID grants another runtime authority. Independent reviewers retain implementer ownership.

## Readiness checklist

| Prerequisite | State | Evidence or required input |
|---|---|---|
| Defined product and task acceptance | PASS for planning | Plan 1.0-draft and canonical M0-01 task description |
| Review/backlog traceability | PASS | 35 tasks, 44 proposed native dependency edges, MCP read-back |
| Reproducible committed source baseline | BLOCKED | Original scaffold/plan mostly untracked; guidance/templates locally excluded. Record an authorized reviewed snapshot before engineering handoff; no commit is authorized here. |
| Named disposable AlmaLinux 10 host | BLOCKED | No host identity, access/console route or image identity supplied |
| Recoverable clean snapshot | BLOCKED | No snapshot identifier, timestamp or restore proof supplied |
| Exact host inspection/mutation authority | BLOCKED | Current assignment is repository/project planning only |
| Exact image/minor release and CPU ISA | NOT_RUN | Measure OS and v2/v3 flags; do not assume x86_64 suffices |
| Repository/package/security inventory | NOT_RUN | Host not inspected |
| Independent inventory reproduction | NOT_RUN | Separate reviewer and lab access not supplied |
| Stack compatibility and lock promotion | NOT_RUN | M0-02–08 unexecuted; manifest remains candidate |

## Bounded next assignment: repository preparation

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

Only after explicit host-specific authority and a recoverable clean snapshot are supplied, collect the approved inventory on the named disposable host. Record exact commands, outputs, timestamp, image/repository provenance, CPU ISA, topology and security baseline. Stop on production ambiguity, absent recovery boundary, unexpected credentials or requested mutation beyond authority. Do not install packages or alter networking/security merely to fill an inventory.

Full artifacts: measured Markdown and sanitized machine-readable host inventory; image/package/kernel/repository evidence; topology and security-state description; host-identity/authorization/snapshot record; reproducible command log and evidence digests; independent review and next-experiment handoff.

Full acceptance: mandatory host fields are measured, provenance and ISA are established, snapshot recovery is verified and inventory is independently reproduced. Unavailable mandatory observations prevent acceptance. Every PASS is backed by actual output; missing runtime checks remain BLOCKED/NOT_RUN. Any initial requirement wording permitting marked unknowns is an inventory-completeness rule, not a waiver of mandatory host evidence.

## Independent QA and milestone gate

A separate reviewer, using the `almadial-review` skill and independent-review template, repeats critical inventory checks against the recorded snapshot, verifies sanitized evidence hashes and documents discrepancies. The implementation worker cannot self-approve. Reviewer must not take over task ownership. The native QA Review column retains the `in-review` status slug; native Accepted retains `done` and is used only after evidence review and explicit acceptance.

After accepted M0-01, M0-02 qualifies exact OS/Asterisk/VICIdial/PHP/Perl/MariaDB/phone candidates and authenticated source acquisition. M0-03–07 provide actual module, generated PJSIP/dialplan, ConfBridge allocation, AMI/AGI, dialing, two-way WebRTC media and recording evidence. M0-08 repeats the required suite on a second clean VM and proposes the reviewed manifest/validator transition. M1–M4 implementation remains blocked until the complete M0 gate is accepted.

No production access, carrier/PSTN calls, installer implementation, package installation, real credentials, blanket policy disablement, commit, push or publication is authorized by this readiness report.
