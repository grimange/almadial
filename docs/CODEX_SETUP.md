# Codex setup and workflow

## Local prerequisites

- Git and Codex CLI; Go later when Go code begins.
- Python 3 for the optional repository scaffold validator (standard library only).
- A **disposable** AlmaLinux 10 x86_64 VM with console/snapshot access for M0 runtime experiments; do not use a customer or production server.

## Add the kit to a new repository

Copy the contents of this starter kit into the root of `almadial` without overwriting existing user files. Before first push, inspect the license/ownership decision, Git diff, and contents of `docs/implementation-plan.md`.

```bash
cd /path/to/almadial
python3 scripts/validate_scaffold.py
codex --version
codex
```

Codex recognizes root `AGENTS.md` and repo skills in `.agents/skills`. From a Codex session, reference the first task in `prompts/M0-01-codex.md` or explicitly invoke `$almadial-compatibility`.

## Optional user-level config (not installed automatically)

Review your user-level `~/.codex/config.toml` directly; this repository does not ship a configuration example or install user settings. Avoid project-supplied settings that disable user approvals or broaden network privileges. Use workspace-limited permissions for routine development, read-only permissions for inspection-only review when possible, and isolated lab workflows for root operations.

## Tool access and permissions

- Repository analysis: local Git/search/files, no privileged host access.
- M0 VM research: only VM address/name, console/snapshot mechanism, and commands explicitly permitted by the task.
- GitHub or issue tracker MCP: optional; do not infer hostnames, member IDs, user permissions, or tracker status. If connected, read actual tool schemas and verify ownership before updates.
- Browser/WebRTC test: browser developer tools, approved disposable SIP endpoints, test audio loop; no public telephone calls without explicit authorization.
- Independent review: separate worker/session with no implementation ownership, evidence reproduced where feasible.

## Recommended worker allocation

- **M0** Codex: Principal Linux/VoIP Compatibility Engineer; high reasoning. Tools: source inspection, VM console, snapshots, `dnf`, Asterisk CLI, MariaDB, browser tools only if authorized.
- **M1–M2** Codex: Principal Go Infrastructure Engineer; high reasoning for executor/journal/security contracts, otherwise medium when tasks are straightforward.
- **M3** Codex implementation + independent VoIP/WebRTC QA reviewer.
- **M4** Independent reviewer + operator/maintainer signoff.

Model selection should follow what the installed Codex CLI actually offers; no model name is hardcoded in the repo. Do not automatically assign yourself tracker tasks unless the tracker is connected and the task explicitly grants authorization.

## Verification

Run `python3 scripts/validate_scaffold.py` after editing instructions or skills. This verifies the starter files and skill metadata only: it **does not** run Go tests, deploy AlmaLinux, or demonstrate telephony functionality.
