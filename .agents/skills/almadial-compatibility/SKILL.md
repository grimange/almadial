---
name: almadial-compatibility
description: Investigate and qualify exact AlmaLinux 10, VICIdial, Asterisk, PHP/Perl/MariaDB, and WebRTC versions before AlmaDial installer implementation. Use for M0 compatibility research, source audits, reproducible lab experiments, and stack lock promotion proposals.
---

# M0 compatibility investigation

Use this skill for M0, not to implement or announce a production installer.

1. Read `docs/implementation-plan.md` §§3, 7, 9, 10, 14, 16 and `docs/PLANS.md`.
2. Define the bounded compatibility claim: exact OS image, architecture, repository set, Asterisk release, VICIdial SVN revision, VICIphone revision, PHP, MariaDB and Perl versions.
3. Inventory upstream contracts and breaking changes from authoritative source. Asterisk 22/PJSIP/ConfBridge is a **candidate**, not a fact of VICIdial compatibility. Inspect dialplan generation, AMI, AGI, recording, background jobs and browser phone integration.
4. Record all unverified assumptions with `NOT_RUN`, not PASS. Build an experiment from `templates/compatibility-experiment.md` and a matrix from `templates/compatibility-matrix.md`.
5. If and only if the task authorizes a named disposable VM with a recorded snapshot, perform the approved experiments. No customer servers, real carrier secrets, unsolicited PSTN calls, or privilege expansion.
6. Collect sanitized commands, observed outputs, version and network topology facts. Name the precise reproducibility limitations.
7. Only propose `manifests/stack.lock.yaml` promotion to `accepted` after **all** blocking compatibility cases pass and independent reviewer evidence is present. Do not self-promote.
8. Report a concise verdict: `SUPPORTED_CANDIDATE`, `REJECTED`, or `BLOCKED`, with reasons and the next smallest experiment.

For each test distinguish unit/simulation checks from a live VM integration test. Sources that are unencrypted, mutable, unsigned, or unpinned require a provenance decision before installing anything with root access.
