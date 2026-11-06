# ALM-8 product-scope realignment — authorized decision

Operator decision, current ALM-8 Product Scope Realignment assignment (2026-11-07 Asia/Manila): "Simple, reliable, single-server VICIdial installer for AlmaLinux 10." "Direct-public, 1:1 NAT, and remote-home-agent tests are NOT universal prerequisites for validating the installer or accepting the local software stack." This assignment explicitly authorizes aligning ALM-8, aggregate M0, ALM-27 and ALM-29. External profiles remain NOT_RUN until executed; qualification applies to the topology actually used or explicitly advertised. Local functional, provenance, security, reproducibility, independent-QA and maintainer gates remain required.

## Decision boundary

This replaces the earlier ALM-7-only deferral, which explicitly left aggregate M0 and ALM-8 unchanged. It changes applicability of external tests, not their outcomes. Direct-public, 1:1 NAT and remote-home-agent remain **NOT_RUN**. Their procedures stay in plan §10.3 and the ALM-7/ALM-8 reports. A deployment must qualify the topology it uses before claiming that configuration works. There is no universal requirement to construct every topology for M0 or a core installer release.

The plan copies in `docs/implementation-plan.md` and `docs/AlmaDial_Implementation_Plan.md` remain identical. M0 retains source integrity, two executions of required local cases, a second clean installation, security, reboot persistence, independent AGY QA and explicit maintainer lock-promotion approval. ALM-8 QA submission is not M0 closure or production readiness. ALM-27/ALM-29 remain Backlog/unassigned; ALM-9 stays unstarted. No dependency is removed.

## Exact Kaneo before/after changes

Full pre-change task descriptions and relations are preserved in the protected continuation evidence and the scope-history comment on each affected task. The following diffs show every description/title change; unchanged sections retain their original text.

### ALM8 — `ftlf2grvt89s3h4nv1yz4x11`

```diff
--- before

+++ authorized-after

@@ -17 +17 @@

-- M0-07: Exact phone/browser/JS identities; WSS topology decision; per-topology browser/SIP/ICE/DTLS/SRTP evidence; actual two-way speech and recording observations.
+- M0-07: Independently accepted local/LAN phone/browser/JS identities, secure registration, ICE/DTLS/SRTP, two-way media, supported agent lifecycle and call-control evidence. Unexecuted external profiles remain NOT_RUN and are deployment-specific under ALM-27.
@@ -26 +26 @@

-Reproduce every mandatory package/application/call/conference/WebRTC/recording and reboot case twice without undocumented edits; independently reproduce critical failures/successes; hash sanitized acceptance evidence and verify all pins.
+Reproduce every mandatory local/LAN package/application/call/conference/WebRTC/recording and reboot case twice without undocumented edits on the accepted reference and second clean AlmaLinux 10.2 installation. Verify pinned software/source/package provenance, fresh schema initialization, deterministic compatibility derivations and configuration generation, agent workflow, service restart/reboot persistence, security boundaries and cross-VM functional equivalence. Record corrections and repeat affected checks. A third clean installation is required only by a demonstrated substantive reproducibility defect, not automatically after a targeted correction. Independently audit the final executable recipe and reproduce critical failures/successes; hash sanitized acceptance evidence and verify all pins.
@@ -30 +30 @@

-Every required case passes twice; independent reviewer differs from experiment author; maintainer explicitly accepts the gate; only then promote the accepted lock. Otherwise record BLOCKED_COMPATIBILITY/NOT_READY and retain candidate.
+Every required local/LAN case passes twice with attributable reference/second-installation evidence and no unresolved undocumented intervention. Required scope: second clean OS installation; pinned stack and provenance; deterministic derivations; fresh database; actual Asterisk/VICIdial generation; native WebRTC, ConfBridge, AMI, FastAGI and recording; actual VICIphone agent lifecycle and supported controls; restart and reboot persistence; security/cleanup; executable replay procedure and cross-VM comparison. Direct-public, 1:1 NAT and remote-home-agent success are not prerequisites for ALM-8 or aggregate M0; each remains NOT_RUN until tested. Implementation completion permits QA Review only. Independent reviewer differs from experiment author; maintainer explicitly accepts the M0 gate; only then promote the accepted lock under reviewed path/schema policy. Otherwise retain candidate and record the actual unresolved implementation or acceptance gate.
@@ -42,0 +43,4 @@

+
+## Authorized product-scope reconciliation — 2026-11-07
+Operator decision, current ALM-8 Product Scope Realignment assignment (2026-11-07 Asia/Manila): "Simple, reliable, single-server VICIdial installer for AlmaLinux 10." "Direct-public, 1:1 NAT, and remote-home-agent tests are NOT universal prerequisites for validating the installer or accepting the local software stack." This assignment explicitly authorizes aligning ALM-8, aggregate M0, ALM-27 and ALM-29. External profiles remain NOT_RUN until executed; qualification applies to the topology actually used or explicitly advertised. Local functional, provenance, security, reproducibility, independent-QA and maintainer gates remain required.
+Historical criteria and exact before/after changes are preserved in the scope-realignment comment and docs/planning/alm-8-product-scope-realignment.md. This amendment does not assert any external profile PASS or authorize infrastructure, spending or production deployment.
```

### ALM27 — `ea1axf0373fggib2v5u7ciyv`

```diff
--- before

+++ authorized-after

@@ -26 +26 @@

-Run all §10.3 P0 cases with exact browser/topology/duration and codec records; collect ICE/DTLS/SRTP counters plus operator listening observation; reject expired cert, failed media and stranded resources.
+Run all core local/LAN §10.3 P0 cases with exact browser/topology/duration and codec records; collect ICE/DTLS/SRTP counters plus operator listening observation; reject expired cert, failed media and stranded resources. For a deployment-specific or explicitly advertised external capability, qualify its actual selected topology using the preserved external procedures; all three profiles are not universal prerequisites.
@@ -33 +33 @@

-Mandatory external-network qualification retained from ALM-7: before any remote-agent or production deployment-readiness claim, execute and independently verify (1) direct-public HTTPS/WSS and WebRTC media; (2) authorized fixed 1:1 NAT traversal; (3) remote-home-agent login and browser media; (4) correct public/private address advertisement; (5) ICE candidate and RTP routing correctness; (6) certificate hostname validation; (7) remote-agent supported call controls and documented reconnect behavior; (8) controlled recording and call/session cleanup under each tested topology; (9) no unintended public exposure of AMI, MariaDB or FastAGI; and (10) independently attributable evidence and explicit acceptance. No LAN or Multipass-only result substitutes for an external profile. Missing topology authority or infrastructure leaves the corresponding case NOT_RUN/BLOCKED and prevents remote-agent/deployment qualification. Repeat the accepted external matrix under ALM-29 / M4-01 before release acceptance. This requirement does not authorize public infrastructure, firewall changes, PSTN calls or spending.
+Deployment-specific external-network qualification: retain the direct-public, fixed 1:1 NAT and remote-home-agent procedures and their expected registration/media evidence. An individual deployment qualifies the topology it actually uses; an installer release does not need all three profiles. For each selected or explicitly advertised external configuration, execute and independently verify HTTPS/WSS and browser media; correct public/private advertisement; ICE candidates and RTP routing; certificate hostname validation; supported call controls and documented reconnect; controlled recording and session/channel cleanup; no unintended AMI/MariaDB/FastAGI exposure; and attributable evidence with explicit acceptance. Untested profiles remain NOT_RUN, unsupported profiles are explicitly identified, and LAN/Multipass evidence cannot substitute for external qualification. A missing selected-topology test prevents claiming that configuration works, rather than blocking core local/LAN installer acceptance. ALM-29 must verify evidence for any external capability actually advertised by that release. This requirement authorizes no public infrastructure, firewall changes, PSTN calls or spending.
@@ -44,0 +45,4 @@

+
+## Authorized product-scope reconciliation — 2026-11-07
+Operator decision, current ALM-8 Product Scope Realignment assignment (2026-11-07 Asia/Manila): "Simple, reliable, single-server VICIdial installer for AlmaLinux 10." "Direct-public, 1:1 NAT, and remote-home-agent tests are NOT universal prerequisites for validating the installer or accepting the local software stack." This assignment explicitly authorizes aligning ALM-8, aggregate M0, ALM-27 and ALM-29. External profiles remain NOT_RUN until executed; qualification applies to the topology actually used or explicitly advertised. Local functional, provenance, security, reproducibility, independent-QA and maintainer gates remain required.
+Historical criteria and exact before/after changes are preserved in the scope-realignment comment and docs/planning/alm-8-product-scope-realignment.md. This amendment does not assert any external profile PASS or authorize infrastructure, spending or production deployment.
```

### ALM29 — `xu7wxvch94ii2iw38hpb8d58`

Title before: M4-01 — Reproduce clean installations across direct-IP and NAT cases

Title after: M4-01 — Reproduce clean single-server installations and qualify installer release

```diff
--- before

+++ authorized-after

@@ -10 +10 @@

-M4-01 — Reproduce clean installations across direct-IP and NAT cases. Produce the concrete outcome below within the native, single-server product; do not infer readiness from priority, numbering or synchronization.
+M4-01 — Reproduce clean single-server installations and qualify installer release. Produce the concrete outcome below within the native, single-server product; do not infer readiness from priority, numbering or synchronization.
@@ -13 +13 @@

-Run the released-candidate installer on at least two fresh authorized VMs using identical accepted manifest and supported direct-public/1:1 NAT/remote-agent cases.
+Run the release-candidate installer on at least two fresh authorized supported AlmaLinux VMs using the identical accepted manifest. Verify dependency resolution, source identities, database initialization, required generated configuration, services after reboot, local/LAN telephony and recording, predictable failure/recovery behavior and installer security controls. Preserve deployment-specific direct-public/1:1 NAT/remote-agent procedures; these three profiles are not universal installer-release blockers. Any explicitly advertised external capability requires supporting evidence for that particular configuration.
@@ -30 +30 @@

-Two fresh installations pass every P0 case with zero undocumented edits; supported browser/topology limitations explicit; no inherited manual lab state or implementer-only repair.
+Two fresh installations pass every core installer/local-LAN P0 case with zero undocumented edits; security, predictable error handling, reboot recovery and reproducible evidence pass; no inherited manual lab state or implementer-only repair. Supported browser/deployment limitations are explicit. Evidence supports every external capability actually advertised by the release; untested external profiles remain NOT_RUN and are not claimed supported. Independent QA and maintainer release acceptance remain mandatory; publication requires separate authorization.
@@ -42,0 +43,4 @@

+
+## Authorized product-scope reconciliation — 2026-11-07
+Operator decision, current ALM-8 Product Scope Realignment assignment (2026-11-07 Asia/Manila): "Simple, reliable, single-server VICIdial installer for AlmaLinux 10." "Direct-public, 1:1 NAT, and remote-home-agent tests are NOT universal prerequisites for validating the installer or accepting the local software stack." This assignment explicitly authorizes aligning ALM-8, aggregate M0, ALM-27 and ALM-29. External profiles remain NOT_RUN until executed; qualification applies to the topology actually used or explicitly advertised. Local functional, provenance, security, reproducibility, independent-QA and maintainer gates remain required.
+Historical criteria and exact before/after changes are preserved in the scope-realignment comment and docs/planning/alm-8-product-scope-realignment.md. This amendment does not assert any external profile PASS or authorize infrastructure, spending or production deployment.
```

## Aggregate plan reconciliation

| Contract | Previous applicability | Authorized applicability | Preserved gate |
|---|---|---|---|
| M0 §§9.3, 14 / ALM-8 | Every required test inherited all external §10.3 P0 profiles; previous ALM-7 amendment explicitly did not waive them | Selected single-server software stack, local/LAN compatibility and second-clean-VM reproduction | Twice-executed local cases, provenance, fresh schema, generation, media/recording, agent lifecycle, security, reboot, AGY and maintainer acceptance |
| §10.3 / ALM-27 | Three external P0 profiles mandatory before remote/deployment claims, with all repeated for release | Reusable deployment-specific procedures; qualify the actual selected topology | Real media, valid TLS, correct addressing, controls/recovery/recording/cleanup, management isolation and independent evidence for any claimed external capability |
| §14 M4 / §16.3 / ALM-29 | Two clean installs covering direct-public/1:1 NAT/remote-agent; every topology P0 a release gate | Two clean installs pass core installer/local-LAN P0; evidence for each specifically advertised external capability | Dependencies/source verification, generated configuration, reboot, local telephony, security, deterministic error/recovery, zero undocumented edits, independent QA |

## Evidence and handoff

The existing ALM-8 execution ledger remains immutable: `20261106T192640Z/SHA256SUMS`, SHA-256 `ebaa490c6650d672121d20f1dbcfdf86038703435b0f49255326b907310619fc` (475 artifacts). Current read-back of prior comment `lbj84v0teo61f1k23gb2dk0b` confirms its former BLOCKED verdict was the unchanged aggregate external scope. That comment is historical, not edited. Current correction/evidence review and stack-lock proposal are in [the existing reproduction report](../compatibility/m0-08-baseline-reproduction.md).

This is the operator-authorized scope decision, not an independent acceptance. Manifest remains candidate. Final task-description read-backs, comment receipts and dependency comparisons are recorded in the protected continuation evidence and final Kaneo comment.

MCP verified each original-history comment, exact replacement comment and final
task description before any QA status transition:

| Task | Preserved original criteria | Authorized replacement and decision |
|---|---|---|
| ALM-8 | `csixuzkhnmgezh1p5gawtp5l` | `vc2cvehnuiaaeq27fdg6180z` |
| ALM-27 | `bdwcxmqb8yhlvgdym0pmk4xa` | `kyge9hcvycxji7b596tbm57f` |
| ALM-29 | `jp5eiogj2iz5tzuwh2aghgxw` | `vvdmcrlv3auiscpcmbndbnj5` |

Continuation validation: 126 unit tests passed, including the focused source
derivation and replay boundary cases. Scaffold, candidate consistency, generated
configuration fixture checks, six Python compile checks and browser JavaScript
syntax checks exited 0. These offline checks supplement the preserved second-VM
runtime evidence; no new media execution is claimed. Both laboratory VMs and
the candidate manifest remain unchanged by this scope-realignment continuation.
