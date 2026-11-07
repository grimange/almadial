# PR-01C sanitized evidence index

Collected 2026-11-06 UTC / 2026-11-07 Asia/Manila for dialer01.
No passwords, credential hashes, database dumps, private keys or audio are stored.
Phone number 7000000001 and recording metadata refer to the authorized synthetic
test fixture. Network/configuration metadata is retained for this operator report.

| File | Meaning |
|---|---|
| `before.json` | Scoped pre-deployment hashes, permissions, native policy, services, grant/firewall state and backup health |
| `deployment.txt` | Scoped archive/rehearsal, added column SELECT grants, installed hashes and service actions |
| `enrollment-fix.txt` | Correction after private umask blocked receipt-directory traversal; final helper hash |
| `after.json` | Immediate deployed snapshot |
| `regression.json` | Existing files changed, byte-identical audio and exact firewall comparison |
| `final-comparison.json` | Final service/native-file comparison; nft policy/member equality after excluding live countdowns and packet counters |
| `final.json` | Final read-only installed state and service health |
| `integration.json` | Metadata-only real-host probes, service-user SELECT and verified inode open, no audio consumption |
| `python-tests.txt` | 51 portal/auth/media tests, synthetic data only |
| `finalizer-tests.txt` | 28 finalizer/enrollment tests, synthetic data only |
| `chromium.json` | Four isolated Chromium journeys using synthetic Basic users/audio; only the initial mock Administration challenge was supplied credentials |
| `supplemental.json` | Raw file hashes/permissions, automatic repeated enrollment and rollback check results |
| `operator-acceptance.json` | Human acceptance replies separated from automated evidence |

`deployment.txt` predates the narrow receipt-directory umask fix; the final
enrollment helper is SHA-256
`849bb015fdd8baebc8a0737b2134cd46b3af705246b1deeb02bd236703ec6513`.
`final.json` covers the three previously installed changed files; the deployment
and fix records cover the added files. Test fixtures never use production audio.

The native/portal UI, firewall, TLS and Asterisk checks here are bounded regression
checks, not a fresh telephony or broader production-readiness acceptance. Browser
results attributed to the operator were not witnessed by the implementation worker.
