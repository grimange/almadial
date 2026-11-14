# Internet distribution correction

## Product contract and confirmed cause

The [deployment contract](../operations/deployment-profiles.md#product-deployment-contract)
makes standard remote installation the product target. Laboratory distribution
is development infrastructure, not evidence of internet deployability.

| Constraint | Evidence | Why it blocks internet deployment | Concrete resolution |
|---|---|---|---|
| Laboratory acceptance substituted for product acceptance | Prior private release handoff used a Multipass-hosted release and CA | Ordinary internet hosts cannot route to it | External distribution is a separate mandatory gate; private tests cannot satisfy it |
| No public release channel | Live GitHub inspection: no releases or tags; public main at `f885482478e2fa13fa95a1211c0ab688363b617d` | Public bootstrap still selects an older laboratory binary | Prepare one first-party GitHub prerelease packet; obtain explicit publication approval, publish, then verify actual assets |
| Release-bound private source policy | Both acquisition policy files listed a developer LAN store; fresh wizard required a source-store choice | Changing binary URL does not change embedded source policy | Empty default mirror list and no mandatory store prompt; private stores remain advanced overrides |
| Expired nightly artifact | Original VICIdial November 5 ZIP returns HTTP 404 | Exact-byte archive retrieval alone exhausts public origins | Add authoritative r4037 fallback bound to the already accepted complete source-tree digest |
| Third-party rights treated as a blanket release obstacle | ALM-32 has specific application redistribution/source-offer findings | First-party CLI assets were conflated with upstream archive mirroring | Ship CLI/bootstrap/linked-code notices separately; retrieve upstream inputs directly; do not publicly upload application archives |
| Historical Git publication concern | Owner-published main already contains the previously identified history | Repeating the old history objection does not create a binary channel | Inspect actual outgoing payload; preserve history; request exact first-party publication scope |
| No external test resource | Owner answered “no” when asked for an authorized external host | No independent internet-only AlmaLinux installation can be observed | Owner supplies an authorized disposable external host; external gate remains NOT_RUN |

No new milestone is needed. ALM-34 remains the installation acceptance corridor;
ALM-46 remains Accepted and ALM-35 is not started.

## Implementation

- The release generator supports versioned GitHub Releases defaults, a pinned
  expected binary hash, explicit unpublished metadata and a deterministic bundle
  of licenses/notices for code actually linked into the CLI.
- The existing Bash bootstrap supports GitHub's HTTPS asset redirects, bounded
  to three hops, with no TLS downgrade, hash override or unverified execution.
  It obtains the SVN utility from signed AlmaLinux packages when missing.
- Default source policy has no private origins. Fresh guided installation does
  not request a private store. Explicit mirror overrides and saved-policy resume
  retain their previous binding and integrity safeguards.
- VICIdial fallback exports only authoritative r4037, ignores externals, verifies
  the complete accepted tree before use and records a distinct representation
  hash. Existing source loaders retain their accepted source identities and
  rehash the actual archive after reopening.
- No accepted runtime, Asterisk modules, dialplan, firewall, media configuration,
  native workers, journal format or plan approval logic was replaced.

## Five-source availability

Live observations were made from the development workstation's internet path.
They are not an external-host acceptance test. No new private distribution VM
was created and no third-party archives were publicly uploaded.

| Input | Public origin / identity | Observation and verification |
|---|---|---|
| Asterisk 22.12.0 | `downloads.asterisk.org/pub/telephony/asterisk/asterisk-22.12.0.tar.gz` | HTTP 200; exact SHA-256 `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` |
| PJPROJECT 2.17 | `raw.githubusercontent.com/asterisk/third-party/52e2f32de1b11e40844bb007749a327862f75275/pjproject/2.17/pjproject-2.17.tar.bz2` | HTTP 200; exact SHA-256 `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` |
| Net::Telnet 3.05 | `cpan.metacpan.org/authors/id/J/JR/JROGERS/Net-Telnet-3.05.tar.gz` | HTTP 200; exact SHA-256 `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` |
| VICIdial r4037/schema1748 | `svn://svn.eflo.net:3690/agc_2-X/trunk@4037` | Actual export succeeded; all 2,511 paths, sizes and SHA-256 values match the independently accepted compliance inventory; tree `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` |
| VICIphone 24e98fcc | `codeload.github.com/vicimikec/ViciPhone/tar.gz/24e98fccfa7b7079fff970e02fca5f002ac1f2cc` | HTTP 200; exact SHA-256 `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` |

The original VICIdial ZIP SHA-256 remains
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`.
The deterministic canonical ZIP produced here instead hashes to
`718a20d74624fbbe8d13770da68b46220f3a600c8f22843a86c362270d28eb4d`.
Those archive bytes are not equal. All source paths and file contents are equal.
The database source digest remains
`2b81ce6f910e5b533d0db4fa78b523c4eed71188588f01a750d22419f1c4b021`;
the VICIdial source digest remains
`3f9475a3f0a9426e2ce220f70719e9e4a08611bc63ef4fe2b924777536e8d25d`.

SVN uses unauthenticated transport on TCP 3690. Full accepted-tree verification
provides source integrity; no publisher authentication or encrypted SVN transport
is claimed. Operators with outbound policy restrictions may use an independently
authorized optional exact-byte mirror. No silent upstream upgrade is permitted.

## Release and external acceptance gates

The accepted native-runtime binary remains `alm46-native-rc1`, implementation
`92b14eb4452b7bfb949e0ee7efc571a83d9e799c`, SHA-256
`a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3`.
The corrected acquisition implementation requires **new bytes and a new version**:
`alm46-internet-rc1`. It must never be advertised under that old checksum.

The prepared destination is a GitHub prerelease in `grimange/almadial` named
`alm46-internet-rc1`. This is **not yet published or a verified working URL**.
Publication requires project-owner approval of the exact source/tag and asset
inventory after the local packet is built and reviewed. The operator guide will
receive a working URL only after HTTP retrieval and asset identity verification.

The owner confirmed no authorized external host is available. External bootstrap,
fresh target package acquisition, guided installation and ordinary agent/media
checks are **NOT_RUN** for this new distribution candidate. Prior accepted
controlled-network runtime evidence is preserved; it is not external acceptance.
No operator VM or accepted qualification host was modified.

The first-party packet contains the CLI, bootstrap, release metadata, checksums
and linked-code notices. It contains no upstream application archives. Outstanding
ALM-32 installed browser-code/source-offer conditions remain separate decisions;
direct upstream retrieval is not a legal clearance claim. Public archive mirroring
is neither necessary for the implemented fallback nor authorized here.

## Verification and protected evidence

Focused tests cover nightly 404 fallback, corrupt cache quarantine/reacquisition,
wrong source-tree rejection, offline verified cache, exhausted origins, unsafe
export links, explicit mirror/default policy, trusted HTTPS/CDN redirects,
untrusted final TLS, checksum rejection and safe binary replacement. Real recovered
sources passed both existing source loaders. Full repository regression and final
committed build results are recorded in the protected evidence manifest and Kaneo
handoff, rather than treating simulated checks as host qualification.

Evidence is stored under
`~/almadial-lab/evidence/internet-distribution/20261114T044638Z/`, with a SHA-256
ledger. It includes upstream HTTP observations, per-file SVN equivalence, actual
Go acquisition results, regression logs, build identity and the unpublished release
packet. Source archives remain private evidence, not Git content.
