# AlmaDial — Repository Instructions for Codex

## Product and current state

AlmaDial is a **Go** installer and diagnostic tool for **native, single-server VICIdial on AlmaLinux 10**, with Asterisk, PJSIP, ConfBridge, MariaDB, Apache/PHP, TLS, and an integrated WebRTC phone. It is an independent installer, not a fork of VICIdial or a continuously running control plane.

**Project status: design / M0 compatibility research. No stack version is accepted. No installer is production-ready.** The canonical proposed requirements and stage gates are in `docs/implementation-plan.md`. Read relevant sections before planning changes. When this file and the plan differ, identify the conflict and request an explicit technical decision; do not silently resolve it.

## Scope discipline

- Work only on the assigned bounded task. Read `docs/GOALS.md`, `docs/PLANS.md`, and the task acceptance criteria.
- **M0 blocks M1–M4 stack-install automation.** Do not claim that AlmaLinux 10, any Asterisk major, VICIdial revision, PHP stack, or WebRTC phone combination works until recorded tests prove it.
- Single server only. No clusters, Kubernetes, deployment agents, web-based control plane, custom distributions, or rewrite of upstream VICIdial Perl internals.
- Prefer the minimal direct implementation. Do not create new abstractions, dependencies, or compatibility shims without a demonstrated need.
- Keep `AGENTS.md` short; put specialized workflows in `.agents/skills/*/SKILL.md` and evidence in task-specific artifacts.

## Work authorization and safety

- Default mode is **repository-only** analysis, planning, or tests. Do not execute root commands, modify `/etc`, install RPMs, adjust networking/firewall/SSH, reboot, touch a server, contact a SIP provider, place a PSTN call, or use a real credential unless the **specific task** explicitly authorizes the exact target and action.
- Any mutation of a disposable lab VM requires explicit task authorization, host identity and snapshot record. Production targets are out of scope until separately approved.
- Do not disable SELinux, firewalld, certificate validation, signature checking, or audit controls to force success.
- Do not execute `curl | bash`, unchecked remote installers, or third-party scripts as root. Check provenance, pins, hashes, and licensing before fetching/building.
- Never expose secrets in source, test fixtures, logs, process arguments, issue comments, prompts, or evidence. Use fake documentation-only credentials in examples. Redact IPs, domain names, private paths, and user data when exporting evidence as required.
- Do not push, merge, publish a release, create external resources, or edit the issue tracker unless the current task authorizes that action.

## Go engineering contract (when implementation begins)

- Go stdlib by default; new dependencies require reason, license/provenance review, pinned version, and focused tests.
- Separate pure planning from effectful execution. The planner emits immutable intended changes, risk levels, and verification expectations; `plan` performs **zero mutations**.
- Stage implementations expose explicit preconditions, actions, postconditions, ownership, retry rules, and failure classes. Treat exit code 0 as necessary, not sufficient, evidence.
- Use structured, bounded subprocess execution; explicit argv (not shell injection), deadlines, captured/redacted diagnostics, and no secret-bearing argv.
- Installer state is persisted atomically. Reconcile actual system state before resuming; never assume a previous journal entry proves current health.
- Make config file ownership explicit. Back up before replacing installer-owned files; never clobber operator-owned or upstream-generated files blindly.
- Database schema imports, irreversible operations, and security-sensitive changes need a defined stop/recovery boundary and operator authorization.
- Follow `gofmt`, `go vet`, standard error wrapping, context cancellation, and table-driven tests once Go code exists.

## Source and licensing

- Treat VICIdial, Asterisk, VICIphone, and OS packages as distinct upstream projects with distinct licenses. MIT is proposed for AlmaDial-original code only, pending copyright ownership verification.
- Pin exact revisions, digests, and package origins before promoting a stack lock. `manifests/stack.lock.yaml` remains **candidate** until an M0 evidence-backed promotion.
- Preserve attribution and notices; do not copy upstream code into the Go project without a license review.

## Verification and evidence

- For each assignment: inspect -> articulate acceptance -> make bounded changes -> run targeted checks -> report evidence and remaining gaps.
- Never report PASS for unexecuted or simulated runtime checks. Distinguish `PASS`, `FAIL`, `BLOCKED`, and `NOT_RUN`.
- Independent QA must not claim independence if the implementation worker also performed final acceptance. A separate reviewer reproduces critical checks when available.
- Run `python3 scripts/validate_scaffold.py` for guidance/skill/template changes. When Go code exists, run applicable `go test ./...`, `go vet ./...`, and `gofmt` checks. Only claim commands you actually executed.
- Do not silently modify unrelated code or overwrite user changes. Include affected paths, commands, results, open risks, and explicit next task in handoffs.

## Task routing / skills

- **M0 source/VM research**: `.agents/skills/almadial-compatibility/SKILL.md`
- **Go installer stage design/implementation** (M1 onward): `.agents/skills/almadial-install-stage/SKILL.md`
- **PJSIP/WebRTC/conferencing validation**: `.agents/skills/almadial-webrtc-test/SKILL.md`
- **Independent evidence-based QA**: `.agents/skills/almadial-review/SKILL.md`

Consult `templates/` for task, experiment, evidence, review and handoff reports. Start with `prompts/M0-01-codex.md` only when the repository and authorized lab environment are ready.

## Code review rules

Prioritize: incorrect claims of compatibility, destructive reinstall behavior, unsafe retries, silent defaults, credential leaks, altered firewall/SSH policy, downgraded TLS/SELinux protections, unverifiable downloads, untested agent audio, and missing reproducibility evidence. Flag severity and file/line; explain the failure mode and a bounded correction. Cosmetic feedback is secondary.
