# Independent client acceptance

Operator used private existing credentials; none were requested or retained.
Linphone extension 1001 uses UDP 5060; the existing PJSIP client uses UDP 5061.

| Phase | Operator response | Server corroboration |
| --- | --- | --- |
| A: no grant; force both fresh registrations | `yes` to “Do BOTH fail?” | No effective web/phone grant, no native set member and no owned rules for the test network; legacy REGISTER attempts received no replies. PJSIP arrival not observed in this initial phase. |
| B: authorize normally; force both registrations | `yes` to successful registration of each client | Three dynamic service rules and native phone membership; two positive-expiry REGISTER 200 responses on each SIP port. Legacy received four 401 challenges; PJSIP received two. |
| C: revoke; wait 15 seconds; force both | `done. confirm registration failed`, followed by `I can confirm that 5061 is blocked too` | No effective web/phone grant, native membership or owned service rules. Compared with B: 39 additional legacy and 13 additional PJSIP incoming REGISTERs with no replies. Both input drop counters increased. |
| D: reauthorize; force both | `yes it confirmed` to success of both clients | New active grant, three effective service rules and native membership; compared with C, two new positive-expiry REGISTER 200 responses on each SIP port, with four legacy and two PJSIP 401 challenges. |

The PJSIP server evidence arrived after an additional request to fully disable,
save and re-enable that account; this is why client indicators alone are not
treated as packet evidence. Successful deregistration replies are counted
separately from positive-expiry registrations. Counts are cumulative snapshots;
use per-phase deltas for later phases rather than reusing an old success.

An initial C response was not accepted as proof: the server still showed the
Test B personal grant and no new REGISTERs. The operator was asked to repeat
the explicit revoke action, using the current “Revoke My Access & sign out”
control. The successful no-grant snapshot above was recorded only afterward.
