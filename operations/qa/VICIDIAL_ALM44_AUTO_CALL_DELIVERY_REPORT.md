# ALM-44 — Native automatic outbound call delivery

## Result

**VICIDIAL_ALM44_AUTO_CALL_DELIVERY_PARTIAL**

The original answered-call handoff failure is corrected. Three native automatic calls reached READY agent `qaagent14` through VICIphone1002 and ConfBridge9600000, completed native NI disposition, and automatically finalized recordings6–8. Bidirectional usable audio is **not accepted**: the operator heard browser tones, but reported that recordings6 and7 contained no return speech. A third, explicitly requested diagnostic call isolated incoming SIP1001 RTP as digital silence, including after the operator changed its capture/mute settings. ALM-44 remains In Progress under Codex; it is not ready for independent QA, and ALM-43 cannot claim nine accepted workflows yet.

This is developer verification, not independent QA or promotion of the candidate AlmaDial installer stack. No external calls were placed.

## A. Verified baseline

Target `dialer01`, SSH22222, AlmaLinux10.2; VICIdial2.14b0.5/SVN r4037/schema1748; Asterisk20.21.0; MariaDB10.11.18; PHP8.3.36; VICIphone3.0.0; Chromium149.0.7827.55. ALM-45 was independently Accepted and was not reopened. ALM-36 and ALM-42 were preserved.

Initial repository HEAD: `bcbc9057f6b5054ba1daf552ac9514d569f3b5e8`. Tracked working tree was clean; substantial unrelated untracked runtime/QA work was preserved. Only this task's exact paths are committed. No push.

Evidence uses server UTC and native epochs. The observed host was still November11 UTC while the operator's Manila date was November12. Native call DATETIME values are New York local; no timezone change was made. Call/agent/recording correlation uses epochs/uniqueids rather than comparing unconverted displays.

The installed local optimizer and outbound AGIs match pinned r4037 hashes, respectively:

- `agi-VDAD_local_optimize.agi`: `18e93f195785481e3e190fed9682c8e0bf645623066958699458160d7068829a`.
- `agi-VDAD_ALL_outbound.agi`: `e1ac82b6ae1d24ba99af61a45b58db10f7cb130ccf80eff8ff861c3981b3b148`.

See [source inspection](evidence/alm44-auto-call-delivery/source/agi-inspection.json), [preservation](evidence/alm44-auto-call-delivery/preservation.json), and [final readback](evidence/alm44-auto-call-delivery/final-readback.json). Existing successful backup service result and active backup, recording, portal-apply and renewal timers were verified. The actual backup unit is `alma-vicidial-backup`; an early exploratory check of nonexistent `alma-backup` units is not a backup-health failure.

## B. Manual control path

Reused the recent accepted manual simulator calls on leads69/70, recordings4/5, including operator/media evidence in [Webphone acceptance](VICIDIAL_WEBPHONE_PLAYWRIGHT_ACCEPTANCE.md) and [fixture inventory](VICIDIAL_OUTBOUND_TEST_FIXTURE.md). No extra manual control call was needed to diagnose the demonstrated optimizer failure. Existing HTTPS/WSS, phone1002, native login and conference join worked in the new tests.

The approved destination is synthetic `7000000001`, existing extension `22217000000001@default`, ending in `Dial(SIP/1001,45,To)`. Working manual calling and automatic dialing share the internal destination and conference media, but automatic dialing first executes its Local-channel optimizer and outbound agent-selection AGI. No carrier route, phone credential, endpoint template, firewall, or WSS setting was changed.

Prior audio acceptance is historical evidence; it does not override the current operator's missing-voice observations.

## C. Automatic path and first verified divergence

[Historical failure](evidence/alm44-auto-call-delivery/historical-automatic-failure.json) captures lead76/list910022, manager155 Originate to `138368` and manager156 Redirect of `SIP/1001-00000036` to `8368`. Destination answered; agent remained READY and no customer handoff occurred; outcome NA. The associated Asterisk journal at November10 14:47:31 UTC contains:

```text
Can't open /var/log/astguiclient/action_process.2026-11-10: Permission denied
```

The native optimizer selects the correct real SIP leg from `cid_channels_recent_002029153161` (line105), inserts a manager Redirect with status SENT (124–128), then calls `eventLogger(...,'process',...)` at135. `eventLogger` dies if append fails (201–206). Only later does it log the launch at154 and execute `AST_send_action_child.pl` at158. The error therefore prevents the actual AMI Redirect from being launched, despite the already-inserted SENT row. SENT alone is not evidence of an AMI failure or success.

The root-created log was owned root/root, mode0644, with no ACL; the root-owned log directory was0755. The AGI executes as Asterisk and could not append that daily file. The shared root-worker/Asterisk log ownership mismatch is the earliest demonstrated divergence. The upstream optimizer, dialplan selection and channel lookup did not require a source patch.

| Layer | Accepted manual path | Failed automatic path | Corrected automatic path |
|---|---|---|---|
| Agent | Logged in, conference joined | READY, correct campaign | READY before acquisition; INCALL with selected lead |
| Destination | Internal SIP1001 | Internal SIP1001 rings/answers | Internal SIP1001 rings/answers |
| Handoff | Manual native path | Optimizer dies before AMI child | Optimizer launches Redirect to8368 |
| Conference | Accepted agent/customer media | Customer not delivered | Customer joins agent's9600000 |
| Media | Historical two-way acceptance | No agent/customer conversation | Browser tones heard; SIP1001 currently sends silence |
| Outcome | Native disposition/recording | NA | NI, matching agent/call/recording epochs |

The current path is native hopper selection → `AST_VDauto_dial.pl` → manager Originate `Local/22217000000001@default` with extension138368 → answer → optimizer → real SIP leg Redirect8368 → outbound AGI selection of READY agent → ConfBridge9600000 → INCALL/XFER. No parallel dialer or AMI controller was introduced. The temporary AMI observer was passive and whitelisted event fields.

## D. Smallest correction and deployment

Added [systemd-tmpfiles configuration](../vicidial/runtime/almadial-vicidial-agi-logs.conf), deployed root:root0644 as `/etc/tmpfiles.d/almadial-vicidial-agi-logs.conf`:

```text
a+ /var/log/astguiclient - - - - u:asterisk:rwx,d:u::rwx,d:u:asterisk:rwx,d:g::r-x,d:m::rwx,d:o::r-x
a+ /var/log/astguiclient/action_* - - - - u:asterisk:rw-,m::rw-
```

Applied with `systemd-tmpfiles --create /etc/tmpfiles.d/almadial-vicidial-agi-logs.conf`. Existing root ownership remains. Named-user ACLs permit Asterisk to append existing action logs; the directory default ACL permits future normal0666-open daily logs created first by root. No recursion into archived logs, package installation, new worker, upstream source modification, or service restart. This deliberately grants the Asterisk service write access to the shared native log directory; it is not a general log-directory permission reset.

Deployed/repository SHA256: `8e093ffd38469185f62234e9564b2edfcdf2863ee1fc65ff525dcc2760045205`. Captured original metadata in root-only `/root/alm44-auto-call-delivery/permissions-before.json`; sanitized metadata is in [deployment evidence](evidence/alm44-auto-call-delivery/cause-and-deployment.json).

[Isolated host qualification](../../tests/runtime/qualify_agi_log_acl.py) passed seven checks: existing append, future root-created inheritance, either creator, unrelated-user denial, unchanged archive, repeated reconciliation, and exact metadata rollback. This tests future-file behavior in temporary fixtures; no claim of an actual overnight rollover is made. Existing systemd-tmpfiles boot processing owns persistence. Reboot was unnecessary and not executed.

## E. Native test fixtures and results

Native Administration created list910023, name `ALM44 Single Internal Auto Cal`, under QAOUT01, with one fresh synthetic lead per attempt. Native agent actions used qaagent14/VICIphone1002. Tests temporarily used RATIO1,24hours, omit-phone-codeY, ALLFORCE and no-hopper-loginY. RATIO auto level0 is clamped to1 by this native UI; the inactive list and only one fresh uncalled lead bounded each test. No list-wide reset or direct SQL write was used.

| Lead | Vendor identifier | Native customer uniqueid | Answer / conference join UTC | Talk/call seconds | Result / recording |
|---|---|---|---|---|---|
|77|ALM44AUTO01|1794419522.338|17:52:05 /17:52:06|69/69|NI; recording6,69s|
|78|ALM44AUTO02|1794420002.348|18:00:06 /18:00:07|74/75|NI; recording7,75s|
|79|ALM44AUTO03|1794420843.358|18:14:07 /18:14:07|224/225|NI; recording8,225s|

All final leads are NI/1/Y; each has one call record. Native one-second agent/call accounting differences are disclosed. Call and recording start/end epochs and durations agree. All three recording locations automatically populated protected HTTPS URLs, without manually running the finalizer. The operator used native entitled playback for recordings6/7. Recording8 finalization is database-confirmed; no separate human playback claim is made.

Attempt77 established corrected routing. Attempt78 followed an explicit microphone-unmute/input-meter check to investigate missing return energy. Attempt79 was explicitly requested by the operator after both earlier recordings lacked speech, and isolated the cleartext customer RTP. No additional calls are planned.

Evidence: [connected77](evidence/alm44-auto-call-delivery/connected-state.json), [AMI77](evidence/alm44-auto-call-delivery/ami-events.jsonl), [AMI78](evidence/alm44-auto-call-delivery/ami-media-recheck.jsonl), [AMI79](evidence/alm44-auto-call-delivery/ami-voice-retest.jsonl), [preflight79](evidence/alm44-auto-call-delivery/third-preflight.json), [connected79](evidence/alm44-auto-call-delivery/third-connected.json), [final state](evidence/alm44-auto-call-delivery/final-state.json). The bounded third AMI observer ended before hangup while the operator adjusted capture; its termination is corroborated by native end_epoch/disposition and final zero-channel/session state, not an invented AMI Hangup event.

## F. Return-audio isolation

The operator reported browser tones, spoke during calls, and explicitly confirmed recordings6/7 had **tones only; voice missing**. Packet counts alone were not accepted as usable return audio.

Third call's customer leg was SIP1001, Asterisk RTP UDP19582, learned symmetric remote UDP25201, PCMU. The browser's encrypted WebRTC flow used another server port12916 behind the same public IP. The energy evidence excludes that encrypted flow: ciphertext cannot be interpreted as PCMU audio.

A bounded passive observer retained only packet counts, decoded PCMU RMS/peak and timestamps, discarding all payloads. Before and after the operator confirmed an in-call capture/mute change, incoming customer packets arrived at approximately50 packets/sec but **every measured incoming window had RMS0 and peak0**. Outgoing customer-leg tone windows had RMS approximately5800 and peak8828. Native customer channel statistics showed1640 received packets with zero loss during the check. Browser received-audio energy stayed at0.34211843139622417, audioLevel0, including the final sample at14511 received packets.

See [customer RTP windows](evidence/alm44-auto-call-delivery/rtp-voice-retest.jsonl), [post-change windows](evidence/alm44-auto-call-delivery/rtp-after-microphone-change.jsonl), and [operator observations](evidence/alm44-auto-call-delivery/operator-observations.json). No audio capture or private recording is committed.

Conclusion: current return-speech failure is upstream of server mixing, in the audio emitted by the Linphone endpoint/capture chain. Exact local device/driver cause is not established remotely. Changing server RTP/firewall or automatic routing cannot recover speech already encoded as digital silence. The smallest next step is to verify Linphone's own capture/device path produces non-silent outgoing audio, then perform one authorized end-to-end check and entitled playback. A system input meter by itself did not establish that Linphone captured that input.

## G. U05 acceptance matrix

| ID | Status | Evidence and actual result |
|---|---|---|
|U05-01|PASS|Native list910023 and fresh NEW/0/N leads; third-preflight.json; native form writes, SELECT corroboration|
|U05-02|PASS|Native auto-dialer consumed eligible leads through its hopper path and emitted correlated manager Originate/AMI calls; hopper empty after consumption. No instantaneous retained READY-hopper snapshot or historical hopper-log row is claimed|
|U05-03|PASS|qaagent14 READY/lead0/campaignQAOUT01/conf9600000 before activation; phone registered and joined|
|U05-04|PASS|Native manager159 Originate and corresponding AMI Local/SIP call events; all three uniqueids correlate|
|U05-05|PASS|AMI DialEnd ANSWER and operator answering internal SIP1001|
|U05-06|PASS|AMI ConfbridgeJoin of each exact customer uniqueid in9600000; live CLI membership|
|U05-07|PASS|Browser receives lead fields; native agent INCALL and auto-call XFER identify same SIP channel/lead|
|U05-08|FAIL|Browser tones heard; return speech absent in recordings6/7; third customer RTP digitally silent even after capture adjustment|
|U05-09|PASS|Native Hangup Customer, NI selection and SUBMIT; NI in lead/call/agent records|
|U05-10|PASS|Uniqueid, lead, campaign, epoch, duration and recording6–8 correlation; automatic finalization|
|U05-11|PASS|Final inactive MANUAL/0 campaign and inactive lists, zero hopper/INCALL/live agents/auto calls/channels/conferences; browser closed and phone registration expired|

`PASS` for hopper traversal rests on the installed native execution path plus actual origination; it is not a claim of directly capturing transient hopper membership. An exploratory read of nonexistent `vicidial_hopper_log` returned1146; no schema addition or import correction followed.

## H. Regression, errors and final environment

Protected PHP/AGI/dialplan/SIP/PJSIP/HTTP/ConfBridge hashes match the retained baseline. ALM-36 preview-release source, ALM-42 activation, ALM-45 scheduling/runtime, phone1000/1001/1002 configurations, portal/firewall, recording finalizer/playback, certificate renewal and backup implementation were not changed. Source/config comparisons and actual native login/call/disposition/finalization were used; a full repeat of the eight other usability workflows was not performed.

No fresh action-process permission-denied fatal occurred after correction. Existing logs are not empty: visibility telemetry duplicate-key exceptions, undefined native PHP variables, ICE candidate STUN `Operation not permitted` messages, SIP retransmit/WSS closure and already-empty ConfKick warnings appeared. Actual selected ICE was connected and customer delivery succeeded; no evidence tied those warnings to the native optimizer failure. One-second native accounting differences are retained. No new call-log duplicate was observed in this run.

First and third logout attempts showed the native frontend's “Logout was not confirmed” alert; read-only checks showed the server had nevertheless removed the agent and all channels. The second PAUSE/unregister/logout returned the expected acknowledgement. These observations are retained as a separate logout-acknowledgement limitation; no speculative PHP patch was applied. The final browser was closed only after confirming server cleanup.

Final QAOUT01: inactive, MANUAL, auto0,12pm-5pm, omit-phone-codeN, ONDEMAND, no-hopper-loginN. Lists910014–910023 inactive. Leads77–79 retained NI/1/Y; historical leads/calls/recordings retained. No unintended INCALL, hopper, agent, customer call or ConfBridge member. Phone1002 unregistered/expired; phone1001 remains configured and registered. Actual final timestamps/readback are in evidence. Temporary local test credential files were removed; existing operator credential files were preserved.

## I. Reproducible checks, change scope and rollback

Executed checks:

```bash
# Read-only evidence assertions locally
python3 tests/runtime/verify_alm44_evidence.py
# Authorized root lab qualification, temporary fixtures only
ssh -F /dev/null -T -p 22222 root@2.29.153.161 \
  'python3 -B -' < tests/runtime/qualify_agi_log_acl.py
```

The evidence checker certifies routing/disposition/recording/cleanup, and explicitly does **not** certify human bidirectional audio. Test scope does not include speculative additional runtime inventory or predictive dialing.

Task files: this report; the runtime tmpfiles configuration; `tests/runtime/qualify_agi_log_acl.py`; `tests/runtime/verify_alm44_evidence.py`; sanitized `evidence/alm44-auto-call-delivery/`. No native source patch exists because correcting native log access resolved the handoff. Find the focused commit with `git log -1 --format=%H -- operations/qa/VICIDIAL_ALM44_AUTO_CALL_DELIVERY_REPORT.md`; exact hash is returned in the handoff.

Rollback is scoped to metadata/configuration and requires an idle synthetic test state. It intentionally restores the original failure condition; do not use it to troubleshoot the independent Linphone microphone problem.

1. Confirm zero active calls/agents and that the deployed tmpfiles file still has the SHA above. Preserve any later operator changes rather than overwriting them.
2. Move only `/etc/tmpfiles.d/almadial-vicidial-agi-logs.conf` out of tmpfiles search paths (retain in the existing root-only backup directory).
3. Read `/root/alm44-auto-call-delivery/permissions-before.json`. For each recorded path, confirm current ownership still matches recorded uid/gid, remove only `system.posix_acl_access` and `system.posix_acl_default` xattrs introduced here, restore recorded mode, then restore original base64-decoded ACL xattrs. The retained snapshot has empty original ACL maps. Do not change file bytes or ownership, recurse into archives, or touch other paths.
4. Daily action files created after the snapshot inherit the new default ACL. For an exact pre-change policy, separately inventory only new `action_*` files in this directory, confirm their ACLs are solely those introduced here, remove those ACLs and restore native root-created0644 or owner-Asterisk native mode. Stop on any unrelated ACL drift. Restore the directory's original0755/no-default-ACL from the snapshot to stop future inheritance. No blanket recursive chmod/chown is appropriate.
5. Verify original metadata and unrelated source hashes. No service restart, firewall reset or database restoration is needed. Reapply the checksum-verified config plus the scoped tmpfiles command to reinstate the correction.

Exact old-file metadata rollback and repeated application were executed in isolated temporary host fixtures by the qualifier; a live rollback and reboot were not performed.

## J. Kaneo and readiness

ALM-44 ID `y7py73mnpdr6w0b8wf5sk6ie`, projectALM, assigned Codex. Verified and moved Ready→In Progress for this implementation. The root cause, correction, evidence, limitations and commit are published back to this task with read-back. It remains In Progress while U05-08 fails. ALM-45 stays Accepted; ALM-43 remains In Progress. No independent acceptance is claimed.

Remaining acceptance boundary: non-silent Linphone capture and verified intelligible customer-to-agent audio on one completed native automatic call. Routing is developer-verified; the complete U05 lifecycle is not yet ready for independent AGY QA. Once that media item passes, publish the correlated result and move ALM-44 to QA Review under Codex ownership, then reconcile U05 with the eight previously verified ALM-43 workflows after independent acceptance.

Final verdict: **VICIDIAL_ALM44_AUTO_CALL_DELIVERY_PARTIAL**.
