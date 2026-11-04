---
name: almadial-webrtc-test
description: Verify AlmaDial's VICIdial browser phone path from HTTPS/WSS registration through PJSIP, ICE/DTLS-SRTP, ConfBridge, two-way audio, and recording. Use for M0 WebRTC compatibility experiments or M3 end-to-end acceptance on explicitly authorized lab systems.
---

# Real WebRTC and telephony acceptance

1. Read `docs/implementation-plan.md` §§9–10, 11, 16 and the authorized task.
2. Capture exact browser, phone source revision, Asterisk revision, VICIdial revision, certificates, NAT topology, selected codec, and relevant listener/port policy.
3. Use a disposable lab with synthetic identities and approved test phones. Do not call live numbers or attach real carriers without explicit task approval.
4. Verify HTTPS and WSS certificate trust separately; then PJSIP registration/contact lifetime, actual SIP/WebSocket flow, ICE/DTLS-SRTP establishment, bidirectional media, agent conference join/leave, and dialer recording lifecycle.
5. Check media both directions. Registration alone is not a successful WebRTC test. Preserve sanitized browser diagnostics, Asterisk CLI evidence, call topology, and test identifiers.
6. Test supported NAT cases separately (direct reachable host, single NAT, remote browser); label blocked tests and conditions explicitly.
7. Distinguish actual audio tests from simulated SIP/SDP traces; if manual listening is needed, require an operator observation, not an AI-inferred PASS.
8. Use `templates/acceptance-evidence.md` to report every pass/fail and follow-on defect. Never suppress a failed call path to produce an overall PASS.

No blanket firewall exposure, insecure TLS shortcuts, unknown certificates, `chan_sip` assumptions, or legacy MeetMe assumptions for an Asterisk 22 candidate.
