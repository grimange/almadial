---
name: almadial-install-stage
description: Design or implement one bounded Go installer stage for AlmaDial after its prerequisites are accepted. Use for preflight, immutable plans, controlled command execution, atomic journal, file ownership, service verification, and safe resume behavior.
---

# One Go installer stage

Use this only when a named M1/M2/M3 task is authorized and earlier milestone gates have passed.

1. Read `docs/implementation-plan.md` §§5–8, 11–13, 14–17 and the task specification.
2. Define one stage's contract: inputs, owned resources, dependencies, preconditions, intended mutations, postconditions, evidence, idempotent rerun and recovery boundary.
3. Keep pure planning apart from the executor. Ensure `plan` has no host writes, no installs, no network mutations, and no credential disclosure.
4. Use explicit process argv, timeouts/context cancellation and redacted output. Never interpolate untrusted data into a shell command.
5. Respect files the stage does not own; for owned files, use backup + atomic replace + verification. Never broaden permissions or reset other firewall/SSH policy.
6. Persist journal safely: stage version, input digest (no secrets), observed state, checks and errors. A completed entry must be re-validated on resume.
7. Add table-driven unit tests and fixture-backed behavior tests for the stage; include interruption/retry, drift, reject-unsafe-input and no-mutation tests.
8. Run the available Go formatting/tests/vet and relevant integration checks; mark unavailable tooling `BLOCKED` or `NOT_RUN`.
9. Finish with changed paths, exact test outputs, risks, and an independent review request using `templates/independent-review.md`.

Do not invent support for unverified Asterisk/VICIdial versions, perform a whole-stack install as a stage test, or claim root/VM execution you did not perform.
