# Goals and guardrails (proposed)

## Outcome

A new operator on a supported clean AlmaLinux 10 x86_64 host can install a native single-server VICIdial dialer with HTTPS and working browser WebRTC using a guided Go executable. Installation is reviewable, resumable where safe, and verifiable without undocumented manual repair.

## Invariants

1. An M0-tested version combination precedes installation automation.
2. Root mutation is explicitly gated and limited to an authorized disposable test VM during research.
3. The installer owns its state, not the running application’s operational decisions.
4. `plan` cannot mutate host state.
5. Stages verify actual postconditions and preserve evidence without secrets.
6. Native Linux operations remain understandable after installation.

## Not in first release

Multi-server, HA, migrations from existing hosts, full system upgrades, provider configuration automation, live dialing without approval, an always-running agent, and rewriting VICIdial's existing Perl code.

## Success criteria

See `docs/implementation-plan.md` §16. Final success requires complete clean-host deployment; authenticated browser phone; two-way audio; ConfBridge lifecycle; dialer behavior, recording and reboot persistence; independently reviewed evidence; and license/security review.
