# PRD-03C — proposed preview release before logout

**Review artifact only. Not installed on dialer01.** This is a targeted proposed change to installed VICIdial SVN r4037, not an accepted compatibility patch or an AlmaDial installer feature. The assignment explicitly requires separate approval before upstream PHP/JavaScript is changed. See the [acceptance report](../../VICIDIAL_PRD_03C_PREVIEW_LOGOUT_CORRECTION.md).

## Exact source boundary

The [unified diff](preview-logout.patch) changes only `agc/vicidial.php` and `agc/vdc_db_query.php`. It requires these exact installed-source SHA-256 values:

| File | Required SHA-256 |
|---|---|
| vicidial.php | `f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358` |
| vdc_db_query.php | `5aaeabd1d9b247dc563b9e94bb86d0b3323346b226b52d47a00ca2506897108d` |

The native source retains its upstream copyright and AGPL notices. The diff contains the context needed to review the proposed derivative change; it does not vendor the full native application. Treat VICIdial licensing separately from AlmaDial-original code. Retain those notices and the applicable upstream source offer if a derivative is later distributed.

## Intended transition

Visible LOGOUT with an unfinished native list preview sends `manDiaLskip` with a new `logout-preview` format. The frontend waits for the exact `PREVIEW RELEASED` acknowledgement before issuing ordinary `userLOGout`. It keeps the phone, session and preview displayed on release failure, provides a retry message, and ignores a late acknowledgement after timeout. Repeated clicks while either request is pending are ignored. Native logout completion and embedded-phone removal wait for the native six-field response, rather than any HTTP 200 body.

Calling the old JavaScript `ManualDialSkip` without changes is insufficient: that code clears frontend preview state before confirmation and the backend leaves the called flag Y. Normal next-lead selection requires called N. The proposed bounded branch therefore uses the existing skip action, audit and prior-status/count semantics, with additional ownership and call guards and restoration of the list-preview eligibility flag/counters. It does not create a call, disposition, table or background service.

The backend serializes native claim/dial/skip/logout requests per user using a connection-scoped advisory lock. After waiting it rechecks the web session. A preview DIAL must still own its pointer; a logout cannot discard an outstanding pointer. The release UPDATE requires the matching user, campaign, server, conference, preview pointer, INCALL status, expected count, called Y, no current agent lead, PAUSED/READY state, no pending external dial, no auto-call row and positive lead/phone daily counters. One MyISAM multi-table statement restores the previous status and count, called N, clears the preview pointer, and reverses the two daily counter increments. There is no unsupported transaction/rollback claim for MyISAM. An exact already-released-state check permits an acknowledgement-loss retry without another decrement or skip audit.

## Deliberate limits and review points

- Scope is an ordinary list-driven MAIN-number preview originally selected with called N, transformed by native claim to Y. Manual-entry/callback/alternate-number and Y1/Y2 reset histories are not qualified; they fail closed and require the existing native workflow. This is a proposal for the demonstrated defect, not universal preview certification.
- Client prior status/count use the existing native skip contract; they are not newly persisted server-side history. Review that trust boundary and qualification of allowed original statuses before broader adoption.
- The actual multi-table UPDATE, named-lock behavior and affected-row semantics have **not** been executed against MariaDB 10.11/MyISAM. SQLite tests execute the generated selection predicates only. PHP lint/control-flow checks used local PHP 8.4, not deployed PHP 8.3.
- The skip audit INSERT is a separate MyISAM write. If it fails after release, the response reports `PREVIEW AUDIT FAILED`; a subsequent exact-state retry can acknowledge the release without recreating the missing audit. This observability limit needs explicit fault-injection review before acceptance. No transactional rollback is implied.
- The per-user lock covers these native action paths, not every administrative or maintenance writer. Disabled stuck-preview recovery must remain disabled; the proposal does not make that script safe.
- Cached old agent pages/API clients do not send the new preview-call marker. A later approved deployment must end affected sessions and refresh clients; mixed old/new clients are not qualified. Changes affect the shared native files, so deployment is not campaign-local even though acceptance uses QAOUT01.
- Automatic pause/shift logout reasons, custom logout links, both dial methods, network interruption, concurrent dial/release, cross-midnight counters and real active-call preservation still need native integration acceptance. No new call is authorized by these instructions.

## Local verification already performed

Use private copies of the two pinned installed files in a directory outside Git; no passwords or full source need to be committed. Existing local PHP, Node, patch, Python and Perl are sufficient. No package installation is required.

```bash
python3 tests/webphone/verify_preview_logout_proposal.py /private/native-source \
  --output /tmp/preview-proposal-verification.json
python3 tests/webphone/qualify_preview_recovery.py /private/native-source/AST_flush_DBqueue.pl \
  --output /tmp/preview-recovery-qualification.json
node --check tests/webphone/browser_workflow.js
node --check tests/webphone/runner.cjs
```

The first verifier rejects source drift, applies the diff with zero fuzz in a temporary directory, lints both PHP files, runs 16 actual proposed-JavaScript/LogouT VM cases, executes the proposed PHP release block with stubbed DB calls, tests 15 exact generated admission predicates in SQLite, checks two retry outcomes and reverses the patch to both exact source hashes. These are local tests, not corrected host/browser acceptance. The recovery verifier separately executes the unaltered native Perl recovery branch against isolated SQLite fixtures and deliberately reproduces unsafe outcomes.

## Approval and deployment boundary

The next operator decision is whether to authorize an isolated MariaDB/MyISAM qualification and subsequent scoped test deployment of this two-file patch on dialer01. This repository artifact is not deployment authorization. Before any approved deployment, review the limits above, recheck exact source hashes, back up both files with metadata, confirm a verified backup and no active agents/calls, run deployed-PHP syntax checks, and review affected clients. No cron recovery option, schema change, package installation or Asterisk/MariaDB restart is part of this proposal.

Once separately approved and installed, reuse the existing Playwright runner with the normal portal grant and native QAOUT01/list activation. Select a currently unused eligible vendor by read-only verification. For example, lead 57 / GAP01B_009 was unused at this handoff; do not assume it stays unused.

```bash
# Use the existing PLAYWRIGHT_MODULE, CHROMIUM_PATH and private credential paths.
WEBPHONE_PREVIEW_CASE=corrected WEBPHONE_PREVIEW_CORRECTION_TEST=YES \
WEBPHONE_PREVIEW_VENDOR=GAP01B_009 node tests/webphone/runner.cjs
```

Use a different unused lead for `WEBPHONE_PREVIEW_CASE=repeated`. Both modes reject an unpatched page before claiming a lead. The corrected phase clicks visible LOGOUT, checks release then logout acknowledgements, and verifies the embedded phone is removed without unregistering it first. Read-only correlation must independently establish final lead flag/count/status, no live agent/channel, SIP unregistration and actual next-lead retrieval of the released lead after reconnect. The runner's ACK assertions alone cannot establish eligibility. These corrected browser modes were **NOT_RUN** in this assignment.

Repeat ordinary SKIP as a control. Test lost acknowledgements, rejected release, double-click and concurrent DIAL in an isolated fixture; verify active-call guards with no real customer state. Restore the inactive native fixture after each bounded acceptance sequence.

## Reversal and upgrade

No host patch was deployed, so no host source rollback is presently required. Temporary native campaign/list changes were already reversed. Zero-fuzz reverse application on private copies restored both original byte digests.

For a later approved deployment, rollback only these two owned changes after confirming no active agents/calls: restore the scoped original files and metadata, or reverse only when the deployed patched file hashes match the reviewed result. Check syntax and original hashes and refresh agent pages; preserve call/lead history. Do not restore the whole database or change phones, cron, portal, recordings or timers. If any source hash drifts, stop automatic application/reversal and review a new diff. Every VICIdial update requires rebase and fresh lifecycle/concurrency acceptance; do not apply by fuzz to a new revision.
