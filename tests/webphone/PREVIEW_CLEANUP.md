# Preview logout regression

Use the existing runner and private credential files. No new package or browser profile is required. The runner performs real native UI actions; `preview_state.py` captures SELECT-only database/runtime correlation. Network access must already be authorized through the existing portal.

Only use QAOUT01/MANUAL/0, qaagent14, VICIphone1002, the clean synthetic list910016 and unused GAP01B leads. Activate only that list/campaign through Administration, allow no-hopper login for the bounded test if needed, and verify eligibility and absence of active sessions. `GAP01B_NEXT` accepts the next native synthetic lead rather than assuming hopper ordering; snapshots must prove the actual lead is eligible. It never substitutes a manually entered phone number.

```bash
export PLAYWRIGHT_MODULE=/absolute/existing/playwright
export CHROMIUM_PATH=/absolute/existing/chromium
export WEBPHONE_PHONE_CREDENTIALS=/private/phone.json
export WEBPHONE_AGENT_CREDENTIALS=/private/agents.json
export WEBPHONE_PREVIEW_VENDOR=GAP01B_NEXT
export WEBPHONE_PREVIEW_CORRECTION_TEST=YES
export WEBPHONE_PREVIEW_CASE=corrected
export WEBPHONE_STATE_DIR=/tmp/preview-corrected
node tests/webphone/runner.cjs > /tmp/preview-corrected.json
```

Supported cases: `corrected` (visible direct LOGOUT), `repeated` (duplicate click), `retry` (server commits the release but the test replaces only its first response with HTTP503, then retries visible LOGOUT), `isolation` (another native account and wrong-conference release requests are denied), `skip` (unchanged visible SKIP then logout). Unpatched `direct` requires `WEBPHONE_PREVIEW_DIAGNOSTIC=YES` and native administrative orphan recovery; do not use it routinely. Assertions inspect native acknowledgements, live state and final counters, not HTTP200 alone.

Unset all preview options to repeat two fresh login/RTP/logout sessions without customer calls. The finish phase rejects an open preview; corrected logout deliberately tests the native phone teardown rather than unregistering first. Do not blindly invoke SKIP twice after an acknowledgement failure. Inspect ownership, pointers, status and counters first.

`WEBPHONE_CALL_PROTECTION=YES` is a separately authorized **one-call** hook. It requires a fresh exact `GAP03B_CALL_001` lead and the preflight-proven local SIP1001 route. Lead70 is already consumed and must not be reused/reset blindly. Native answered-call LOGOUT and stale release probes must be denied; native NI disposition must remain intact. This hook must not run as part of routine non-calling regression.

```bash
python3 tests/webphone/verify_preview_cleanup_evidence.py operations/qa/evidence/preview-cleanup-repair
python3 tests/webphone/verify_preview_logout_proposal.py /private/pinned-source --output /tmp/preview-local.json
node --check tests/webphone/runner.cjs
node --check tests/webphone/browser_workflow.js
```

After testing, use Administration to deactivate QAOUT01 and lists910014–910018, restore no-hopper login N and local call time12pm-5pm, keep MANUAL/0 and recordingONDEMAND, verify no INCALL/live-agent/channel and phone1002 unregistered. Never reset an entire list to make the test repeat. See the [implementation report](../../operations/qa/VICIDIAL_PRD03B_PREVIEW_CLEANUP_IMPLEMENTATION.md) for evidence and limitations. Independent QA must reproduce the important cases in a fresh session.
