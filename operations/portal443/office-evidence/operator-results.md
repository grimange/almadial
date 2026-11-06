# Independent operator results

Source: user replies in this conversation, 2026-11-06. Passwords/SIP credentials
were neither requested nor received. This is operator-reported browser/client
acceptance, separate from Codex server-side checks.

- Initial office test reply: “done”. This was treated as ambiguous, not a PASS.
- Clarification reply: **“All listed checks passed”** for Shared Office Access
  showing near24h, correct IP/type and five-second administration redirect;
  a fresh same-network browser reaching native VICIdial login without portal
  authorization; and a forced fresh Zoiper registration succeeding.
- Coexistence/last-grant test reply: **“check it now”**. Server aggregate evidence
  subsequently confirmed no effective web/phone grant, web TTL file, native nft
  phone membership, or portal UDP rule on the office network. This reply alone
  was not counted as proof of a fresh rejected SIP registration.

Final clarification reply: **“Registration blocked; both redirects passed;
coexistence tested”**. The question explicitly requested both coexistence revoke
orders (personal revoke preserves office; office revoke preserves personal),
then removal of the last grant, a forced fresh Zoiper registration and both
native browser entry URLs. Count these as PASS — independent operator report.
The server-side office-network aggregate separately shows zero effective web,
phone, native web file, nft phone member and portal UDP rules after revocation.

NOT_RUN: a new 24-hour wall-clock wait, a reboot, audio/barge monitoring and
independent mobile/JavaScript-disabled checks in this assignment. The prior
assignment's agent/supervisor redirect acceptance is preserved; both redirect
mappings were also regression-tested with mocked HTTP users here.
