# Ready-to-paste Codex assignment — ALMADIAL-M0-01

**Worker:** Codex CLI  
**Model:** Best available reliable coding/research model; prefer high reasoning for compatibility investigation  
**Role:** Principal Linux Telephony Compatibility Engineer / Reproducibility Architect  
**Initial mode:** Read-only repository and upstream research (no host mutation)

Use `$almadial-compatibility` and read `AGENTS.md`, `docs/implementation-plan.md` (especially §§3, 7, 9, 10, 14, 16), `docs/PLANS.md`, and the template files.

**Goal:** Establish the reproducible *research* baseline for a fresh AlmaLinux 10 single-server VICIdial deployment with a browser WebRTC phone. Determine exactly which version combinations and code paths must be verified. The Asterisk 22 / PJSIP / ConfBridge option is a candidate, not an already accepted stack.

**Bounded work:**
1. Inspect current repository state, existing docs, and manifest; record baseline revision and working-tree state.
2. Audit trusted source-level compatibility boundaries: VICIdial's Asterisk-version handling, generated PJSIP configuration, ConfBridge, AMI, AGI/recording, Perl/PHP/MariaDB, and browser phone integration.
3. Identify a **candidate** version matrix with exact pinning/provenance requirements and credible primary references; separate proven code observations from assumptions.
4. Create an M0 experiment backlog and a concrete clean-VM evidence collection protocol using `templates/compatibility-experiment.md` and `templates/compatibility-matrix.md`.
5. If a disposable AlmaLinux 10 VM with snapshot and explicit root authorization is **not** supplied, stop before any installation. Mark all unexecuted VM/browser/call tests `NOT_RUN` or `BLOCKED`.
6. Recommend the first smallest VM experiment, prerequisites, execution commands, expected evidence, stop conditions, and one independent QA check.

**Do not:** install the full stack, execute unknown remote scripts, touch a production server, edit firewall/SSH or security controls, fetch credentials, call a carrier, start M1 code, or mark M0 accepted.

**Expected changes:** documentation and **candidate** manifests only; no Go product implementation, no installer releases. Use a task branch if the environment supports Git and the operator authorized branch creation.

**Tests:** run `python3 scripts/validate_scaffold.py`, inspect relevant diffs, and run any available non-destructive source/static checks. Report which runtime checks were not possible.

**Output:** changed paths; precise source references and version candidates; machine/VM evidence if authorized; results labeled `PASS | FAIL | BLOCKED | NOT_RUN`; compatibility blockers; next bounded task; explicit M0 verdict (default still OPEN); and a request for independent review.
