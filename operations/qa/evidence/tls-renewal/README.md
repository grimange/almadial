# PR-08 evidence provenance

Collected 6 November 2026 UTC (7 November Asia/Manila), from `dialer01` via root
SSH on TCP 22222. `before.json` precedes deployment. `after.json` is the final
read-only snapshot; see its embedded UTC timestamp. Command output in the remote
journal uses the server's EST timezone; JSON event times are Unix seconds.

Only the three TLS integration source/unit paths listed in the acceptance report
were deliberately installed. The deployer also created the scoped root-only
rollback archive/state. Certbot exercised its existing challenge/log/account
mechanism and updated its ARI retry metadata. No firewall reload, grant change,
call, SIP registration, database write, Asterisk/MariaDB/portal restart or upstream
VICIdial edit was performed by PR-08.

`negative-test.json` records an intentionally rejected lineage, before any Apache
reload. Its failed systemd unit was reset after preserving evidence. The persisted
deployment failure was recovered through the successful staging exercise.
`staging-evidence.json` proves actual staging issuance and production hook execution
using the current trusted certificate. It does not prove issuance of a new
production serial. `normal-runs.json` records two subsequent due-only no-ops and
the certificate timer restart. `final-details.json` includes actual graceful
reload command and public portal assets/status checks without response bodies,
cookies or credentials.

`before.json` conserves a collector mistake: `/opt/alma-portal-443/app.py` is
recorded as nonexistent. The actual entry point is `portal.py`; the final snapshot
records it correctly. No comparison treats the nonexistent filename as evidence
of portal integrity. `final-details.json` records the actual portal/auth/phone-
policy hashes, checked against the existing local deployed source. Common policy,
worker, SIP guard and Lua hashes have proper before/after comparisons.

`verify_evidence.py` checks the host artifacts without contacting the server. Its
34 assertions are evidence-consistency checks, not synthetic proof of customer
calls. `local-tests.txt` reports 14 mocked/subprocess failure-boundary tests, not
live reload-failure injection. `scaffold-validation.txt` is repository validation
only. All functional claims retain these distinctions in the acceptance report.

`SHA256SUMS` binds the task source, reports and sanitized evidence as of acceptance.
Run `sha256sum -c operations/qa/evidence/tls-renewal/SHA256SUMS` from the repository
root. Private keys, secret-bearing account files, database dumps and audio are
excluded. No external alerts were sent. Existing unrelated work was not staged.
