# Supplementary executed console checks — 2026-11-06

These results were observed over the persistent desktop SSH session. They
supplement the captured sanitized health/probe bundle; they are an operator
record, not a transcript or independent QA report.

- `systemd-analyze verify` on portal service, apply service and SIP guard unit:
  exit0, no diagnostics. Captured `unit-verify.txt` is empty on success.
- Portal, apply timer, SIP guard and renewal timer are enabled.
- Renewal timer active, twice daily, next firing about seven hours after check;
  last renewal service `Result=success`, `ExecMainStatus=0`.
- Cockpit9090: no listener, consistent with pre-change baseline.
- Fail2Ban jails: portal and Asterisk active.
- Active-office rollback rehearsal: actual office fixture drained, all original
  personal grant expiries/revocations preserved, all15 baseline hashes restored.
  Final code was redeployed; office/personal probes passed afterward.
- The pre-existing global Apache ServerName warning was observed with syntax
  exit0. Sanitized export omits that warning line because it contains an address.

No reboot, full24h wait, new ACME issuance/dry-run or audio/barge test was run.
