# AlmaDial — Codex preparation kit

AlmaDial is a proposed independent Go-based installer for a native single-server VICIdial deployment on AlmaLinux 10, including Asterisk and WebRTC.

**This checkout is an engineering starter kit, not an operational installer.** No production compatibility or install readiness is asserted.

## Start here

1. Review `docs/implementation-plan.md` (proposed baseline).
2. Read `AGENTS.md` (Codex automatically reads project guidance).
3. Inspect `docs/GOALS.md` and `docs/PLANS.md`.
4. Review `docs/CODEX_SETUP.md` for safe local Codex settings and optional integrations.
5. Run `python3 scripts/validate_scaffold.py` (Python standard library only; **not** a runtime dependency of AlmaDial).
6. Give Codex the bounded M0 task in `prompts/M0-01-codex.md`. Do not authorize remote or VM mutation until a named disposable VM and snapshot exist.

## Skills

Repo-scoped Codex skills live in `.agents/skills/`. Invoke by name (`$almadial-compatibility`, `$almadial-install-stage`, `$almadial-webrtc-test`, `$almadial-review`) or let Codex discover them by description.

## Project status

- Plan: draft, needs operator approval.
- M0 source/VM compatibility: not run.
- Stack lock: candidate only; no pinned accepted versions.
- Go installer: not implemented.
- Licensing: MIT for AlmaDial-original code under the recorded rights-holder authority; upstream code retains separate licenses. See [third-party notices and unresolved release conditions](docs/THIRD_PARTY_NOTICES.md).

## Developer workflow

Create a bounded task -> implement or investigate in a branch -> attach redacted evidence -> independent QA -> maintainer approval. The `prompts/` and `templates/` directories support this workflow without adding a dependency to an external issue tracker.
