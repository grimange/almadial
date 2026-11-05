# ALM-2 candidate-stack QA handoff and supporting self-check

Review date: 2026-11-05 UTC (2026-11-06 Asia/Manila).
Repository HEAD: `bba4d144e246745fb0e4dae4cd364d5d009a397c`.
Project: AlmaDial, `ghritpte6e40o5ayktgz0vh1`.
Task: ALM-2 / M0-02, `kzj273ypahs82zegi1wwjfag`.

**Independent QA verdict: BLOCKED — separate reviewer required.** This requested
report path contains supporting checks performed by **Codex, the implementation
worker**, and bounded corrections. It is not an AGY/Gemini review or independent
acceptance. The authenticated Kaneo identity is Codex,
`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`. An AGY workspace member exists, but that
membership does not authenticate this session as AGY. No separate Gemini runner
was available through the configured tools.

The repository review skill explicitly requires: “If a separate QA worker is
unavailable, report a self-check but do not describe it as independent acceptance.”
ALM-2 therefore remains in QA Review, owned by Codex. The local commit and Accepted
transition requested conditionally on independent QA passing have not occurred.
Application runtime compatibility remains **NOT_RUN**; the manifest stays
`candidate`. That expected runtime limitation is not the reason for this review
blocker.

## Scope and evidence preservation

Read the repository instructions, compatibility/review/Kaneo skills, source plan,
ALM-1 independent acceptance report, ALM-2 contract and implementation artifacts.
Read back implementation comment `unkrz9gdax4ihhq1okznwh47`; preserve its original
40-test result, source hashes and observations as historical evidence.

The starting working tree contained a modified manifest and five untracked ALM-2
artifacts: evaluation report, matrix, JSON companion, verifier and verifier tests.
The unrelated untracked `scripts/verify_agy_overlay.py` was preserved. No staging,
commit, package installation, service operation, snapshot restoration, source
execution, database initialization or telephony test was performed in this pass.
Only read-only package queries targeted `almadial-alma10`.

Original protected evidence, relative to the operator home:
`almadial-lab/evidence/m0-02/20261105T162726Z/`.
Its `validation/evidence-files.sha256` hash is
`95780bc1f6ddf7d2ba77cb94794153389f291cf44463020faba7a627945a41ef`.
**89/89 entries were read and matched.** The complete SVN export was separately
hashed using the documented canonical file-record algorithm. The original ledger
and original source artifacts were not changed.

New supporting evidence is retained separately under
`almadial-lab/evidence/m0-02/self-check-20261105T173232Z/` with mode-0700
directories and mode-0600 files. It contains fresh release/signature records,
read-only live package records, replay results, test results and MCP read-back.
Its evidence ledger is a separate integrity snapshot, not an independent review.
Raw host records and upstream source trees remain outside Git.

## Asterisk release reconciliation

**Retain Asterisk 22.12.0 LTS.** The fresh official download index lists the final
archive and detached signature at 2026-11-05 13:15. The official
[22.12.0 release](https://github.com/asterisk/asterisk/releases/tag/22.12.0)
and freshly retrieved release API identify tag `22.12.0`, `prerelease: false`,
published `2026-11-05T13:15:10Z`. The retained archive's `.version` is exactly
`22.12.0`; the official final changelog agrees. Earlier search-index references to
22.11.0 stable and 22.12.0-rc4 do not describe this final artifact.

The retained archive matches the freshly retrieved official SHA-256. Its freshly
retrieved detached signature verifies with exit code 0 and GPG `VALIDSIG`
fingerprint `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`. An isolated temporary
keyring was used. The upstream maintainer key-ID association was re-read from the
[Asterisk discussion](https://community.asterisk.org/t/gpg-key-for-downloads-asterisk-org/87164).
GPG Web-of-Trust is **undefined**, not independently certified. No stronger key
authentication claim is made.

Authenticated-parent `third-party/versions.mak` selects PJPROJECT 2.17,
Jansson 2.15.1 if bundled, and libjwt 1.15.3 if that optional dependency is used.
The selected PJPROJECT archive matches the bundled checksum and retained SHA-256.
The proposed system Jansson 2.14 meets the source's >=2.11 requirement; it is not
being mislabeled as the bundled Jansson release. No affected source pin changed.
Ten required module/interface source files were present, including
`res_pjsip_config_wizard`; their presence is not module-load or runtime evidence.

## Candidate identities and source integrity

| Component | Exact candidate | Source and verification boundary |
|---|---|---|
| Asterisk | 22.12.0 LTS | Official final archive, fresh official checksum, valid detached signature; build NOT_RUN |
| VICIdial | r4037; 2.14b0.5; schema 1748 | Official SVN metadata and exact 2,511-file export; publisher authentication unresolved |
| MariaDB | 3:10.11.18-1.el10_2.alma.2.x86_64 | AppStream exact server/client/devel/backup RPM identities |
| Apache | 0:2.4.63-13.el10_2.8.x86_64 | AppStream; matching mod_ssl has epoch 1 |
| PHP | 0:8.3.36-1.el10_2.x86_64 | AppStream FPM/CLI and extension providers |
| Perl | 4:5.40.2-515.el10_2.x86_64 | AppStream; application/module execution NOT_RUN |
| PJPROJECT | 2.17 at third-party commit 52e2f32de1b11e40844bb007749a327862f75275 | Pinned Asterisk third-party artifact, parent checksum agrees |
| SRTP | libsrtp-devel 0:2.3.0-16.el10.x86_64 | CRB; actual SRTP negotiation NOT_RUN |
| Jansson | 0:2.14-3.el10.x86_64 | CRB, system mode |
| Net::Telnet | 3.05 | Attributable CPAN archive; no module build/execution |
| VICIphone | 3.0; commit 24e98fccfa7b7079fff970e02fca5f002ac1f2cc | Exact maintainer commit/archive; commit API reproduced |
| SIP.js | 0.20.1 | Embedded header and LIBRARY_VERSION agree; embedded file digest reproduced |

| Artifact | Reproduced SHA-256 |
|---|---|
| Asterisk archive | `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` |
| VICIdial canonical exported tree | `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` |
| PJPROJECT archive | `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` |
| Net::Telnet archive | `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` |
| VICIphone archive | `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` |
| Embedded SIP.js | `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` |

Exact URLs and individual RPM identities remain in the
[matrix](../compatibility/m0-02-dependency-matrix.md),
[JSON companion](../compatibility/m0-02-candidate-stack.json) and
[candidate manifest](../../manifests/stack.lock.yaml).

## Dependency accounting and live reproduction

All 113 normalized mappings were re-counted: **85 enabled-repository mappings +
17 isolated-EPEL mappings + 11 without a selected RPM = 113**.
The 30 exact requested names absent from enabled repositories equal
**17 EPEL names + 11 without a selected RPM + 2 capability replacements**.
These are different accounting dimensions, not contradictory counts.

The eleven without RPMs comprise bundled PJPROJECT, CPAN Net::Telnet, the absent
`perl-core` metapackage and eight excluded optional feature dependencies. Their
absence is not an observed platform incompatibility. Later core module-load tests
and optional-feature scope remain explicit requirements.

Fresh guest queries reproduced AlmaLinux 10.2 x86_64, BaseOS/AppStream/CRB/Extras,
all exact selected enabled-repository pins and unchanged installed RPM sets.
Repository files record `gpgcheck=1`; this is policy evidence, not direct RPM
signature verification. EPEL was not enabled. Fresh capability inspection proves
the selected `php-common` advertises `php-curl` and
`zlib-ng-compat-devel` advertises `zlib-devel` and `pkgconfig(zlib)`. Those are
feature-provider observations, not deductions from package-name similarity.

The retained EPEL compressed primary checksum and open checksum were recomputed
against `repomd.xml`. All **18 recorded EPEL package records** matched exact
EVRA/architecture, content digest and license in that primary metadata: the 17
selected matrix entries plus supporting `perl-Text-CSV_XS`. Compressed primary
SHA-256: `d4fff60dd06658d71a399208107640c7f000d50d56f2c347410b6ecc02315c16`;
open SHA-256: `291980706b0d78a34b9904ff7d80391515ff35839248a3a28430903f485618ad`.
This establishes consistency with retained official HTTPS metadata, not direct
signature verification of downloaded RPMs or signed repository metadata. Actual
RPM signatures and installation transaction closure remain pre-install gates.

## VICIdial, PHP, Perl, database and WebRTC findings

Source checks reproduced the Asterisk 18/20/22 branch at `install.pl:3009`, PJSIP
wizard generation at `bin/ADMIN_keepalive_ALL.pl:4403`, the ConfBridge integration
note for r3894, and PJSIP AMI channel handling at `bin/AST_update_AMI2.pl:819`.
VICIdial schema 1748 and background-process/module findings remain attributable to
the pinned export. No upstream program was executed.

**VICIdial privileged execution/distribution remains on HOLD.** Unauthenticated
SVN transport, equal digests across two ports, a failed HTTPS hostname check and
an older HTTPS ZIP do not authenticate r4037. No supported authenticated r4037
channel was established. M0-04 must obtain authenticated source or an explicitly
approved independent attestation before executing it. This downstream gate does
not by itself reject ALM-2 candidate qualification.

The main agent page selects mysqli; the legacy `dbconnect.php` calls
`mysql_connect`, which is incompatible with PHP 8 **if reached**. No evidence in
this pass proves that legacy route is reached in the selected workflow or that
the whole application fails. PHP mysqli exception behavior, MariaDB SQL modes,
MyISAM schema import, local authentication, privileges and Perl 5.40 module loading
remain M0-04 runtime tests. S03/M2-03 owns database bootstrap/schema import under
the plan; downstream VICIdial application registration must consume that state
and must not introduce a second authoritative schema import.

VICIphone commit identity, version 3.0, embedded SIP.js 0.20.1 and AGPLv3 text were
checked from the retained archive. SIP.js/JsSIP MIT notices, PJPROJECT GPLv2 terms,
Net::Telnet Perl licensing and VICIdial's upstream Affero GPL v2 text remain
distinct from AlmaDial's proposed original-code license. No upstream source was
vendored. Any future distribution requires the applicable notices/source duties.
WSS, PJSIP endpoint/rtcp_mux, trusted TLS, ICE/STUN and topology-dependent TURN,
DTLS-SRTP and codec assumptions still require M0-03/M0-05/M0-06/M0-07 execution.
Browser registration, bidirectional audio, reconnect and recording are NOT_RUN.

No earlier manual Asterisk 20.21.0 installation evidence was located in the
reviewed repository documentation. The user's historical observations are inputs
for later include-path, external-IP, process-detection, AMI/conference and media
checks; they are not reproduced evidence for Asterisk 22.

## Confirmed defect and bounded correction

**MEDIUM — local verifier accepted contradictory candidate identities.**
Before correction, changing the JSON Asterisk source version to `0.0.0`, its
digest to 64 zeroes, or the selected PHP-FPM version to `0.0.0` still returned no
local consistency issues. The verifier compared manifest leaves against copied
JSON `manifest_pins` without also comparing the actual source and primary RPM
records. This could report PASS for internally contradictory candidate data.

Corrected [check_local](../../scripts/verify_candidate_stack.py) to compare the five
required source identities/digests/URLs and six primary runtime RPM
NEVRA/repository identities directly with the actual manifest. Missing/duplicate
required sources now fail. Three regression tests cover contradictory
version/hash/URL, missing/duplicate source and primary RPM version drift. No
candidate versions, acquired source hashes or manifest pins changed.

Current corrected verifier SHA-256:
`e8882bd395919d9292f2314dd0452ffee6c78170764b52258980962433412236`.
Current tests SHA-256:
`ab8e3f201188cf4f1389e2f1f7058e5bb3e02bbc2deceab9d4bfbb36fd898148`.
The original implementation comment's hashes remain historical, not current
identities for these corrected files. A separate reviewer must review the fix.

## Executed validation

| Command / check | Result |
|---|---|
| `python3 scripts/validate_scaffold.py` | Exit 0; 11 required files, 4 expected skills |
| `python3 -m unittest discover -s scripts/tests -v` | Exit 0; **43 tests**, including **19 candidate-verifier tests**; zero failures/errors |
| `python3 -m py_compile scripts/verify_candidate_stack.py scripts/tests/test_verify_candidate_stack.py` | Exit 0 |
| `python3 scripts/verify_candidate_stack.py` | Exit 0; LOCAL candidate consistency PASS |
| Verifier `--live --output-dir /tmp/almadial-alm2-self-check-20261105T173232Z/live-corrected --source-dir` original protected sources | Exit 0; exact package/source consistency PASS |
| Verifier `--replay /tmp/almadial-alm2-self-check-20261105T173232Z/live --source-dir` original protected sources | Exit 0; offline package/source consistency PASS |
| `git diff --check` | Exit 0 |
| Original evidence ledger | 89/89 matching files |
| Fresh Asterisk signature verification | Exit 0; expected full VALIDSIG fingerprint |
| EPEL primary metadata and package checks | Compressed/open hashes and 18 records match |

All modes retain `runtime_compatibility: NOT_RUN`. Local mode checks candidate
consistency only. Live mode performs fixed-target, bounded argv-based read-only
queries and records failures. Replay executes no VM query. Source mode checks
hashes, not signature trust; EPEL trust checks require the separate retained
metadata audit above. No mode is an installer or full dependency solver.
Tests also exercise malformed/empty metadata, command failures/timeouts, wrong
OS/repositories, missing pins, installed-RPM changes, source tampering/symlinks,
output collisions, forbidden promotion and fixed-VM isolation.

## Kaneo and final handoff

Live read-back confirms ALM-1 Accepted; ALM-2 QA Review with Codex ownership; ALM-3
Backlog and unassigned. Native relations `jkinkto2gkdunoukh415zhgy` (ALM-1 blocks
ALM-2) and `j296alx8um418ub9x1ludhzf` (ALM-2 blocks ALM-3) remain intact. Task
acceptance criteria, assignment, labels, original evidence and dependencies were
not changed. A supporting Codex self-check comment records this correction; it
must not be interpreted as an AGY acceptance comment.

Supporting comment `hhxyxgtaphinvmw85ncgvsa6` was retrieved through
`list_task_comments` with exactly matching content. Subsequent `get_task` and
`get_task_relations` responses matched the pre-comment task fields and native
relations; the original implementation comment remained byte-for-byte unchanged.

**Next assignment:** a genuinely separate AGY/Gemini Pro-class reviewer reproduces
the targeted checks, reviews the bounded verifier correction and source trust
decisions, then conditionally commits only the reviewed ALM-2 slice (excluding
`scripts/verify_agy_overlay.py` and raw evidence), records the complete commit SHA
and independent comment, and accepts ALM-2 through verified MCP read-back.

After that gate, ALM-3 / M0-03 (`h2d8fcqh6uqem6rczw1fckz0`) is the next native
dependency. Recommend Codex, highest approved coding model, high reasoning, Linux
telephony/build engineer, using Multipass/DNF/RPM, the pinned authenticated source,
compiler/build tools, Asterisk CLI, trusted TLS/WebRTC browser/media diagnostics
and Kaneo. A separate task must explicitly authorize the exact build/package and
runtime actions on `almadial-alma10` with the clean snapshot/recovery record.
Acceptance requires required modules to load and execute, valid and invalid PJSIP
registration, certificate rejection, trusted WSS, real ICE/DTLS-SRTP browser echo
with operator microphone/playback observation, and sanitized binary/build/config,
browser/topology/codec and log evidence. No carrier/PSTN use or security bypass;
build success alone cannot pass. ALM-3 was not started here.
