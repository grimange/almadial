---
name: almadial-review
description: Perform independent AlmaDial review of bounded changes, safety controls, test evidence, and milestone acceptance. Use for read-only QA, release-gate evaluation, recovery/secret checks, and documenting reproducible failures; do not self-approve implementation.
---

# Independent QA / gate review

1. Confirm scope, assigned task, baseline commit, changed paths and who implemented it. Do not silently alter the implementation while acting as independent reviewer.
2. Read the task contract, root `AGENTS.md`, affected architecture sections, and submitted sanitized evidence.
3. Inspect behavior first: no-mutation `plan`, root/production scope, data loss, secrets, external source verification, firewall/SSH/TLS/SELinux integrity, clean recovery, actual telephony and recording postconditions.
4. Re-run available tests independently. If necessary to reproduce a destructive case, require a separately authorized disposable VM and snapshot.
5. Compare claims to observed outputs. Distinguish repository tests from live VM/browser validation. `NOT_RUN` is not PASS.
6. Provide file:line findings ranked `CRITICAL`, `HIGH`, `MEDIUM`, `LOW`; attach expected behavior, evidence, and bounded correction. Note unverified coverage explicitly.
7. State one verdict: `ACCEPT`, `REJECT`, or `BLOCKED`; identify whether the relevant milestone gate is still open. Use `templates/independent-review.md`.
8. Never declare a release production-ready without the full applicable acceptance matrix and operator signoff.

If a separate QA worker is unavailable, report a self-check but do not describe it as independent acceptance.
