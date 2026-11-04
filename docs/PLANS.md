# Milestone and execution conventions

Source of detailed contracts: `docs/implementation-plan.md` §14–§16.

| Milestone | Gate | Product work allowed |
|---|---|---|
| M0 Compatibility | Verify exact upstream revision + environment with repeatable call-path evidence | Research, lab experiments, candidate manifest only |
| M1 Go foundation | M0 compatibility gate accepted | CLI, `plan`, state journal, runner, unit tests |
| M2 Native integration | M1 contract tests accepted | Native package and application provisioning |
| M3 HTTPS + WebRTC | M2 on-host checks accepted | TLS, phone provisioning, real browser call-path validation |
| M4 Release | M3 acceptance evidence reviewed | Recovery, security, install matrix, packaging and signoff |

No milestone is implicitly accepted by checking its own implementation. A failed M0 candidate can be replaced with another *candidate*; record the reason and repeat tests before promotion.

## Bounded task format

Use `templates/task.md`: purpose; current baseline; paths; permitted tools and hosts; non-goals; inputs; acceptance cases; independent QA; handoff. For multi-step work, maintain a short `docs/work/<task-id>.md` execution plan, if needed. Do not create a heavyweight plan for a trivial change.

## Research-to-code handoff

Every compatibility discovery must include exact source versions, commands/inputs used, expected vs actual outputs, environmental limitations, timestamp, and an explicit `PASS | FAIL | BLOCKED | NOT_RUN`. Never convert a hypothesis into a locked version without its evidence.
