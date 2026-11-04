# M0 compatibility matrix

**Matrix revision:** draft  
**Stack candidate ID:** TBD-M0  
**All results:** NOT_RUN until measured

| Case | Required evidence | Status | Evidence reference | Issue |
|---|---|---|---|---|
| OS/PHP/MariaDB/Perl compatibility | Installed versions, successful scripts, DB schema | NOT_RUN | | |
| Asterisk module compilation | PJSIP, WSS, SRTP, ConfBridge loaded | NOT_RUN | | |
| VICIdial DB and background tasks | Schema, cron, keepalive health | NOT_RUN | | |
| AMI and generated configuration | Auth, generated dialplan, reload/restart | NOT_RUN | | |
| WebRTC HTTPS + WSS | Trusted certificate, browser handshake | NOT_RUN | | |
| PJSIP registration | Browser contact visible and stable | NOT_RUN | | |
| ICE + DTLS-SRTP + bidirectional audio | Audio observed in both directions | NOT_RUN | | |
| ConfBridge lifecycle | Join, audio, leave, cleanup | NOT_RUN | | |
| Dialer test workflow | Agent/campaign flow with synthetic test data | NOT_RUN | | |
| Recording lifecycle | File generated, database reference and playback | NOT_RUN | | |
| NAT topology(s) | Tested network scenario(s) | NOT_RUN | | |
| Reboot/restart persistence | Health after reboot | NOT_RUN | | |

**Gate:** candidate may only be promoted after blocking tests PASS with independent review and no unresolved critical incompatibility.
