# M4-04 — SBOM, licensing, source notices and ownership

Date: 2026-11-13 (Asia/Manila). Task: ALM-32 / `rjxsredbmh3hh6kka8ki8qcp`. Implementation owner: Codex. Baseline/unchanged HEAD: `229c0d40fbe76c07389fae1716180a2dcd4b8bb4`. Developer result: **ALMADIAL_M4_04_SBOM_CHANGES_REQUIRED**. This is an inventory/compliance developer review, not independent QA or legal signoff.

The original investigation below is retained as historical evidence. The focused follow-up at the end supersedes its artifact hashes, coverage gaps and regression counts where explicitly stated.

## Canonical contract and governance

Kaneo requires exact installer/OS/Asterisk/VICIdial/VICIphone/JavaScript/patch inventory, selected download-versus-bundle model, versioned SBOM, notices/source obligations, ownership/distribution decision and vulnerability review. Missing provenance/distribution authority blocks release; missing mandatory evidence prevents acceptance. ALM-31 is Accepted. Native dependency remains ALM-28 → ALM-32 → ALM-34; ALM-33 remains To Do. ALM-32 is In Progress with Codex ownership, not QA Review, because the findings below leave mandatory evidence incomplete.

Read AGENTS.md, applicable installer-stage/Kaneo skills, GOALS/PLANS, canonical implementation plan §18 and accepted ALM-29/31/S08/S09/S10/S12 evidence. Stale M0-only guidance is superseded for this authorized task by the accepted baseline and current operator scope; no manifest promotion or milestone reopening occurred.

## Inventory and provenance

Read-only target: `almadial-alm29-clean-d`, machine `4c7b492940923838a0700adba65ed3eb`, recorded snapshot `alm29-pristine`. AlmaLinux 10.2, kernel `6.12.0-211.64.1.el10_2.x86_64`, 731 RPM records, SELinux Enforcing. Installed manifest SHA-256 `80940fac407ca7ee0041378b2e1e8b4ab4d95aa95749b503d5df7d4f356009fa`.

| Inventory | Evidence / boundary |
|---|---|
| AlmaDial source | 898 baseline tracked files; original code distinguished from operations/ third-party-derived patches; root MIT does not cover all repository content |
| Go | Go 1.27.3 standard library; go.yaml.in/yaml/v3 v3.0.4 linked; check.v1 v0.0.0-20161208181325-20d25e280405 test-only; four Go-vendored x/* module identities recorded separately |
| OS | 731 exact NEVRA/source-RPM/vendor/license metadata records; 44 names selected by installer package inventory, other records include base OS and dependencies; this is not an RPM redistribution list |
| Pinned downloads | Asterisk 22.12.0, PJPROJECT 2.17, Net::Telnet 3.05, VICIdial r4037 official 2026-11-05 archive, VICIphone 24e98fccfa7b7079fff970e02fca5f002ac1f2cc |
| Source evidence | Five verified archive SHA-256 identities and 8,134 regular-file digests; no full upstream source tree copied into repository |
| Embedded review | Eight distinct runtime asset records, including SIP.js 0.20.1 and two Chart.js versions; 18 nested source-notice markers; these do not assert complete transitive component/build closure |
| Perl/native runtime | RPM-level Perl module versions, Net::Telnet source, native module listing, ldd dependencies and RPM ownership observations; locally built libasteriskssl/libasteriskpj are not assigned invented RPM owners |

All five archive hashes match the accepted pins; full identities/URLs are in [components.json](../compliance/m4-04/components.json) and [source-files.json](../compliance/m4-04/source-files.json). Cached DNF installed-origin fields are missing for some base-image packages. Their repository is explicitly unknown; RPM metadata does not establish source-file copyright or final legal conclusions. Actual RPM payload archive checksums are unavailable and are not fabricated.

## SBOM and reproducibility

[SPDX 2.3 JSON](../compliance/m4-04/almadial.spdx.json) contains 772 packages: profile + 40 curated records + 731 RPMs. Generator uses Python standard library, deterministic ordering/date/namespace, source-backed declarations, custom LicenseRef records for ambiguous terms, and NOASSERTION for unresolved conclusions. Go build/test dependencies, downloaded sources, embedded assets and installed OS records have distinct categories. Supplemental file inventories are explicitly referenced; `filesAnalyzed: false` avoids pretending that all source-file licenses were classified.

SHA-256: `d1eb57fd60e8ce2b55cdb89483407f65fdb5a5198f1d315afe32716c6da50104`. Official schema validation PASS with jsonschema 4.19.2; two generations byte-identical; six focused tests PASS, including relationship/license-reference integrity and rejection of altered license evidence. [Regeneration instructions](../compliance/m4-04/README.md) require no network or host mutation. Schema validation is structural, not license clearance. CycloneDX was not required or generated.

## Licensing, ownership and distribution decision

The operator explicitly confirmed authority to license AlmaDial-original code under MIT, in response to the question naming Ronniel Allan Castañito and applicable employer/publication authority. Exact answer and question ID are retained in baseline.json. Root LICENSE remains unchanged; README and [THIRD_PARTY_NOTICES](../THIRD_PARTY_NOTICES.md) now accurately distinguish this authority from upstream rights. No third-party assignment, exception or publication permission is invented.

| Component / claim | Review outcome |
|---|---|
| Go/YAML/test module | Source-backed BSD-3-Clause, per-file MIT AND Apache-2.0, BSD-2-Clause respectively; full license/NOTICE/patent evidence preserved; YAML Apache text included |
| Asterisk/PJPROJECT | GPL core terms plus separate nested licenses/linking exceptions retained. Source presence is not equivalent to linked build inclusion; complete nested build/license closure remains unresolved |
| VICIdial | Exact license labels itself AGPLv2 (November 2007); custom LicenseRef-VICIdial-AGPLv2-2007 preserves text instead of silently assigning GNU AGPLv3 |
| VICIphone/SIP.js | Pinned AGPL-3.0-or-later source headers and MIT SIP.js 0.20.1 release license retained; additional embedded contributor/license closure remains incomplete |
| line.js | **Conflicting declaration**: GPL of unspecified version plus no-charge restriction in exact pinned source header. LicenseRef-LineJS-Conflicting retains both claims; release-blocking review required |
| Derived files | VICIdial procps transformations in four Perl files; VICIphone logging/register/reconnect transformations in PHP/JS; three tracked VICIdial patches. These are modifications, not pristine upstream delivery |
| Ownership | Original-code operator attestation VERIFIED as supplied evidence. Third-party copyrights remain separate; unavailable assignments/permissions are not presumed |

| Delivery category | Actual model | Applicable action before release |
|---|---|---|
| Source repository | Go/Python/docs plus derived patches, no five complete upstream source archives | Include notices/full required license texts; preserve upstream patch attribution; root MIT limited to original code |
| Installer binary | Static Go + YAML | Include Go/YAML license/NOTICE material and applicable vendor attribution; actual final release payload has not been published or approved |
| Installer downloads | Five pinned archives staged by operator; Asterisk/PJ/Net::Telnet compiled locally, VICIdial/VICIphone deployed with documented modifications | Preserve exact sources, modifications and build instructions; establish applicable corresponding-source/network-source availability and notices for hosted/distributed modified applications |
| RPMs | Base image plus signed repository installation | Distinct OS inventory; any later image/RPM redistribution requires exact payload/source/notice review |
| Compliance data | License texts, official schema and metadata | Retain upstream text/attribution; schema is unmodified CC-BY-3.0 validation data, not application runtime |

Decision: original-code MIT authority confirmed; **no universal redistribution clearance**. Current download/local-build boundary documented. A future final payload or modified hosted application requires qualified review of line.js, embedded components and corresponding-source delivery. Documentation alone does not satisfy or implement every possible source-offer obligation.

## Vulnerability review

Official govulncheck v1.1.4 installed only in isolated developer /tmp, with no go.mod/go.sum change. Live official Go database metadata modified `2026-11-11T02:51:11Z`; scan reviewed on 2026-11-13 local date. Full symbol analysis failed with exit 2: x/tools v0.29.0 SSA panic `unexpected expr: *ast.KeyValueExpr` on Go 1.27.3. Symbol reachability **NOT_VERIFIED**.

Supported `-scan=package -json ./...` completed exit 0 and found [GO-2026-6859 / CVE-2026-97054](https://pkg.go.dev/vuln/GO-2026-6859), Go 1.27.3/net/http version/package match, fixed in 1.27.4. Advisory describes an HTTP/1 server MaxBytesReader concurrent-header race. Source search found no MaxBytesReader/MaxBytesHandler/server call sites in cmd/internal; AlmaDial is a CLI. This narrows applicability but does not replace unavailable symbol reachability analysis. No dependency upgrade was performed. Exact advisory and limitations are retained in vulnerability-review.json and GO-2026-6859.json.

Complete current RPM and other upstream advisory mapping is **NOT_VERIFIED**. No claim of CVE-free software, complete vulnerability clearance or exploitation was made. Inventory provides exact versions for a bounded follow-up advisory review; lack of that coverage is explicit.

## Preservation and validation

Only read-only VM commands were used: rpm metadata, cached DNF query, archive hashing, systemctl observations, Asterisk module/version inspection, ldd/ownership mapping and Perl metadata. No package installation, database writes, journal execution, service reload/restart, certificate change, firewall mutation or media test occurred. Final comparison: all 731 RPM identities and five archive hashes unchanged; MariaDB/Asterisk/httpd/php-fpm/firewalld/FastAGI active; machine identity unchanged; SELinux Enforcing. This is representative verification, not a new full managed-state or WebRTC campaign.

| Check | Result |
|---|---|
| go vet ./... | PASS |
| go test ./... / -race | PASS after authorized loopback-socket execution; 1,461 passing test/subtest events + 24 package PASS events, five skip events each |
| Initial sandbox test/race | Environment failure: httptest could not create loopback socket; recorded, not concealed as a code failure |
| go mod verify; go mod tidy -diff | PASS, no module edits |
| scaffold; Python unittest discovery | PASS; 228 Python tests |
| gofmt; git diff --check | PASS |
| Focused SBOM tests; schema; regeneration | PASS, six focused tests; byte-identical output |
| CLI smoke | Existing TestCommands exercised in successful Go suite; standalone historical harness not rerun |
| Reproducible Go builds | NOT_RUN: no Go/build inputs changed; Python SPDX output reproducibility verified |
| Hosted CI | NOT_RUN |

## Findings and handoff

| ID | Classification | Required next evidence |
|---|---|---|
| C01 | Blocking release / unresolved third-party authority | Qualified upstream/licensing clarification for line.js contradictory GPL/no-charge terms; do not silently choose a license or replace upstream code under ALM-32 |
| C02 | Mandatory evidence incomplete | Complete nested third-party/license/build mapping, including embedded SIP.js contributor notices, and applicable full notice/source-delivery coverage for exact payload |
| C03 | Distribution decision conditional | Implement/review corresponding-source/network-source delivery for actual modified copyleft hosting/distribution; no unsupported legal signoff |
| C04 | Explicit vulnerability limitation | Symbol scanner/toolchain compatibility and full RPM/other upstream advisory applicability remain unverified; no automatic upgrade scope expansion |
| C05 | Informational provenance limitation | Base-image RPM repository origins and payload archive checksums unavailable; preserve unknown fields |

Task-owned changes: README licensing sentence; existing docs/THIRD_PARTY_NOTICES.md; docs/compliance/m4-04/; this report; scripts/generate_sbom.py and scripts/tests/test_generate_sbom.py. Unrelated operator/QA/local-AI files remain intact. All task changes remain uncommitted. No push, merge, tag, release, downstream implementation or task acceptance.

Protected supplemental evidence: `/home/ra/almadial-lab/evidence/m4-04/20261112T192449Z-alm32/`; its independent SHA256SUMS records source observations, canonical task identity, validation and findings. ALM-32 remains In Progress with Codex; Kaneo receives the findings and evidence digest for the next bounded compliance decision. Independent AGY acceptance must wait for mandatory evidence closure.


## Focused compliance-gap resolution — 2026-11-13

This pass continues developer comment `d299uneis8djdstg2wekq9vl` without restarting inventory. HEAD remains `229c0d40fbe76c07389fae1716180a2dcd4b8bb4`; ALM-32 remains In Progress / Codex, ALM-31 Accepted, ALM-33/34 To Do and dependency relations unchanged. Canonical criteria explicitly make missing provenance/distribution authority release-blocking and missing mandatory evidence acceptance-blocking. Documented exceptions therefore do not authorize QA Review in the present state.

### Exact line.js decision

[Exact finding](../compliance/m4-04/line-js-finding.json): `vicidial-r4037.zip`, SHA-256 `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`, member `2026-11-05/www/vicidial/line.js`. Balamurugan S's 2005 header says GPL without a version, then expressly says users “cannot charge for this software.” It also requires enhancements to be made available free to the public. The complete relevant comment is preserved in the finding. The charging statement is phrased as a restriction rather than a price description, and no separate scope resolves the tension. A bounded upstream clarification search found no authoritative resolution; source wording cannot establish the author's intended legal reconciliation.

Classification: **UNRESOLVED_RIGHTS / DOCUMENTED_EXCEPTION**, release authority **BLOCKING**. The unchanged file is installed in the web application and referenced by 15 installed PHP reporting pages. It is browser-served source even though absent from the Go binary and current Git payload. It cannot be excluded as NOT_DISTRIBUTED. Required decision: rights-holder clarification of applicable GPL version and charging/conveyance terms, or qualified legal review for the actual delivery. No header, installed file or accepted workflow was changed.

### Embedded-component coverage

[Deterministic coverage](../compliance/m4-04/component-coverage.json) evaluates the existing 40 curated records: 16 mapped without recorded exceptions; 24 with explicit exceptions; zero missing evidence-file references or notice-reference coverage within that scope; 18 unresolved license conclusions and one conflicting-evidence record. Two records are outside installer runtime (test dependency and validation data). No additional unmatched LICENSE/COPYING/NOTICE/COPYRIGHT filename candidate was found in the existing inventory. These counts describe mapping coverage, not legal clearance or a guarantee that every source header or linked build component has been classified.

The pinned SIP.js asset contains a complete embedded MD5 notice identifying Raymond Hill's MIT portion and André Cruz's WTFPL v2 portion. The original comment and leading SIP.js notices are now preserved in `licenses/SIP.js-0.20.1/embedded-notices.txt`; coverage includes two explicit MD5 subrecords. The affected SIP.js declaration is corrected to `MIT AND WTFPL`; no blanket relicensing is asserted. Full-text evidence files increase from 38 to 39. Exact source membership, copyright evidence, modification status, distribution category, notice reference and remaining source-obligation review are machine-readable. Nested build inclusion and 18 NOASSERTION conclusions remain conspicuous exceptions requiring qualified review.

### Distribution and locally prepared source

[Distribution model](../compliance/m4-04/distribution-model.json) distinguishes repository-derived patches, the future Go binary, five operator-staged archives/local builds, browser-served modified applications, and the 731 OS/repository RPMs. Installed inventory does not imply RPM redistribution. Original-code MIT authority remains the operator's supplied attestation and does not grant third-party rights.

[`prepare_compliance_source.py`](../../scripts/prepare_compliance_source.py) verifies all five archive hashes and the existing 8,134 regular-file digests. It reads the exact hash-bound accepted Go transformations to reconstruct four VICIdial Perl modifications and VICIphone PHP/JS modifications. The private review candidate includes intact archives, six modified files and unified diffs, build/install source and scripts, required evidence texts, modification manifest, and three tracked operations patches explicitly marked not-applied with unresolved base applicability. It uses exclusive creation, restrictive permissions and deterministic tar metadata. No upstream tree is added to Git.

Candidate qualification: two byte-identical outputs, SHA-256 `8cb5466d3c5fec21c374a867d2bd7a5ecdbc9e69ba59fde8e635da4f68f14f94`, 72,601,600 bytes, mode 0600; 275 members including the checksum manifest, all 274 member checksums verified. Each of six packaged diffs independently applied to its original archive member and reproduced the recorded modified digest. [Source manifest](../compliance/m4-04/source-delivery-manifest.json) and regeneration instructions identify inputs and outputs. This is a locally validated source preparation mechanism, not a qualified legal determination of complete corresponding source or a published source offer.

Outstanding source-access decisions are exact: modified VICIphone's AGPLv3 section 13 requires a prominent source offer to its network users; no deployed source-access mechanism is established. VICIdial's exact custom November 2007 AGPLv2 text has a conditional section 2(d) source-request clause; its upstream-facility condition needs qualified determination rather than importing AGPLv3 terms. Applicable prominent dated modification notices on conveyed modified files also need verification/correction; a separate candidate manifest alone does not prove the installed files meet that requirement. Public publication and installed-system changes remain outside this pass.

### Validation and finite closure

| Case | Actual result / closure |
|---|---|
| C01 line.js terms | DOCUMENTED_EXCEPTION; UNRESOLVED_RIGHTS; required authority BLOCKING |
| C02 embedded notice gap | SIP.js MD5 notice and declaration RESOLVED technically; nested build/conclusion exceptions remain acceptance-blocking |
| C03 source preparation | RESOLVED local provenance, reconstruction and reproducibility; deployed source access / modification-notice decisions BLOCKING |
| SPDX schema and offline regeneration | PASS, 772 SPDX 2.3 packages; current SHA-256 `d5976e79109211ade884a9e26ee04dc4a5e10dd54a832767a4924180fc03d917` |
| Coverage / source hashes / notices | PASS for stated mapping and artifact scope; no universal redistribution clearance |
| go vet; go test; go test -race | PASS; 1,464 passing test/subtest events and two skips per test/race run |
| go mod verify; tidy -diff | PASS, no module edits |
| Scaffold / Python | PASS; 232 tests, including six SBOM and four focused source-preparation tests |
| gofmt / git diff --check | PASS |
| Go builds / hosted CI | NOT_RUN; no Go/build input changes requiring rebuild / no hosted execution |

GO-2026-6859 remains version-affected for Go 1.27.3, fixed in 1.27.4; prior symbol-level scanner crash leaves reachability NOT_VERIFIED. No exploitability assertion or upgrade was introduced. Full RPM advisory applicability remains NOT_VERIFIED.

No VM connection, installed-system command, service operation, credential/database/certificate/firewall change or media test occurred during this follow-up. Inputs were read locally and output was confined to repository, temporary and separately designated protected evidence locations. Earlier host observations remain historical; no new live health claim is made. Historical evidence is preserved.

Additional task-owned files: `scripts/prepare_compliance_source.py`, its four-test module, coverage/distribution/line.js/source-delivery manifests and embedded SIP.js notice; existing notices, SBOM and evidence index updated only for the new findings. No commits or publication. **ALMADIAL_M4_04_SBOM_CHANGES_REQUIRED**: the finite next step is qualified rights/build-scope review plus an approved source-access/modification-notice delivery decision, followed by verification of those specific conditions. ALM-32 stays In Progress / Codex; independent QA acceptance is not claimed.

Protected follow-up: `/home/ra/almadial-lab/evidence/m4-04/20261112T195545Z-alm32-focused/`. SHA256SUMS SHA-256: `8132a6591462841960508a6a3c0b0cdfe330204b570502baf96b5af918a9d2f6`; all 13 members verified. Historical ledger digest remains `a38af0040c7827dba94a0e3b3e716e6fdac2626b90d8e659fb5ce145b07e435e`. The sealed findings copy predates this digest/read-back bookkeeping.

Kaneo follow-up comment `dvnnby4nf0u6gedk90ngjp7o` independently retrieved through MCP and matched byte-for-byte to submitted content. Final task read-back: In Progress, Codex.


## Formal distribution-rights disposition — 2026-11-13

[Distribution-rights decision](../compliance/m4-04/distribution-rights-decision.md) concludes this development cycle: **ALMADIAL_M4_04_SBOM_BLOCKED**. Canonical ALM-32 requires attributable distribution authority and actual source availability; accurate exceptions do not permit QA Review. No release model or source-access mechanism was authorized by prior technical acceptance.

The record evaluates private/internal, installer-only, bundled and hosted models, covers all 24 exception rows, and identifies the authorities required for line.js rights, 18 nested notice conclusions, actual payload/build scope, VICIphone source access and VICIdial conditional applicability. Modification-notice and patch-base technical work remains explicitly contingent on that defined scope; legal approval alone cannot complete those checks. Original-code MIT authority and SIP.js MD5 MIT/WTFPL attribution remain intact.

Fresh checks: official SPDX 2.3 schema PASS; offline SBOM and coverage checks PASS; unchanged candidate hash, all 274 member checksums and six independent patch reconstructions PASS, current source-delivery manifest matches candidate. No SBOM or source candidate regeneration. All mandatory repository checks PASS: go vet, go test and race (1,464 passing test/subtest events, two skips each), go mod verify, tidy -diff, scaffold, Python (232 tests), gofmt and diff check. Hosted CI and new builds NOT_RUN; no Go/build changes. GO-2026-6859 remains the recorded version-based release follow-up with reachability NOT_VERIFIED.

Live MCP dependencies: ALM-31 Accepted; ALM-33 depends on accepted ALM-28 and has no ALM-32 edge, so separately authorized private backup/restore work can proceed. ALM-34 depends on ALM-32 and ALM-33; ALM-35 depends on ALM-34 and accepted license/security/operator gates. ALM-33/34/35 remain unstarted and unassigned; no relation changed.

No Blocked column exists. ALM-32 remains In Progress with Codex using the explicitly authorized BLOCKED-comment fallback; it is not QA-ready. No live host, service, database, credential, firewall, certificate or recording was touched. No publication, commit, push or downstream implementation. Supplemental evidence and exact comment read-back are recorded below after sealing.

Protected decision supplement: `/home/ra/almadial-lab/evidence/m4-04/20261112T202131Z-alm32-decision/`. Final `SHA256SUMS` SHA-256: `bbe6310f50f3b6328474322c532c7c2506e813cfbe565b16c7bfe5ae0a09f392`; ten members verified. The immutable pre-handoff `QUALIFICATION-SHA256SUMS` digest `0c758b6d7fceb79cc8017ee4fca64c4129497238e731f4e0c026cee7d39cfeed` is the digest cited in Kaneo; the final ledger additionally covers comment/read-back and handoff verification. The sealed implementation snapshot predates this digest bookkeeping.

Kaneo comment `w3qgl5zvjjm3ay8arrdky1ga` read back byte-for-byte through MCP. Final read-back: ALM-32 In Progress / Codex with explicit BLOCKED disposition; ALM-31 Accepted; ALM-33/34/35 To Do and unassigned. No downstream task or relation changed.


## Owner-directed technical acceptance realignment — 2026-11-13

The project owner explicitly authorizes ALM-32 technical SBOM and licensing-audit acceptance independently of final third-party redistribution clearance. The original canonical description is preserved verbatim and an additive acceptance amendment was read back exactly through Kaneo. Scope-revision comment: `j4e71lj3ftsqky84zcm4p7a3`. Earlier CHANGES_REQUIRED/BLOCKED conclusions above remain historical; their release restrictions remain effective, while their universal-clearance prerequisite for technical QA is superseded.

**Technical SBOM audit: Qualified for independent acceptance. Third-party redistribution clearance: Not yet established. External publication or release of affected third-party components: Not authorized by ALM-32 acceptance.** Independent AGY review remains required; this developer qualification is not task acceptance.

The technical acceptance inventory remains 731 RPMs, five pinned archives, 8,134 source-file digests, 772 SPDX 2.3 packages, and 40 curated embedded-component records. The current evidence collection contains 39 license/notice texts, following the documented SIP.js embedded MD5 notice correction from the historical 38. Existing source candidates contain 274 checksummed members and six verified patch reconstructions. No inventory, SPDX, notices or source candidates were rebuilt for this documentation amendment.

Known exceptions remain conspicuous: line.js UNRESOLVED_RIGHTS; 24 embedded records with documented exceptions and 18 unresolved licensing conclusions; VICIphone network-source availability; VICIdial conditional source applicability; modification notices, exact release payload/build closure and final source-delivery approval. The local candidate is technically reproducible source preparation, not a fulfilled source offer. Original-code MIT authority does not extend to upstream components. See the [current decision and preserved exception register](../compliance/m4-04/distribution-rights-decision.md).

No downstream edge is changed. ALM-33 has no ALM-32 prerequisite and remains unstarted; ALM-34 retains its existing prerequisites. GO-2026-6859 remains the separate release-security follow-up, with Go 1.27.3 version-affected, 1.27.4 fixed and reachability NOT_VERIFIED. Hosted CI and builds are NOT_RUN in this pass; no build/source changes require a binary rebuild. No VM, service, database, TLS, firewall or recording operation was performed. Validation and final handoff evidence follow below.

Fresh scope-handoff validation: official SPDX 2.3 schema PASS; offline byte-identical SBOM and coverage/notice consistency PASS; existing source candidate SHA-256 and all 274 member checksums PASS; six independent patch reconstructions PASS; source manifest matches the candidate. Ten focused compliance tests PASS. Mandatory regressions PASS: go vet, go test and go test -race (1,464 passing test/subtest events and two skips each), go mod verify, tidy -diff, scaffold, 232 Python tests, gofmt and git diff --check. SPDX digest remains `d5976e79109211ade884a9e26ee04dc4a5e10dd54a832767a4924180fc03d917`; notices digest remains `23340d00098e5fce67daa8032ff621bba26cbd86e2e2abf1699606e8327cc0ec` (see verified ledger for authoritative digest).

Final scope handoff: ALM-32 **QA Review / Codex**, verified through MCP; implementation remains uncommitted for independent AGY review. Scope comment `j4e71lj3ftsqky84zcm4p7a3` and completion comment `h7zst7oc4vs0e9yfjclqaixk` each matched submitted content byte-for-byte on read-back. Original canonical description remains verbatim before its additive amendment. ALM-31 remains Accepted; ALM-33/34 remain To Do and unassigned; targeted dependency edges match the pre-handoff inventory.

Protected supplement: `/home/ra/almadial-lab/evidence/m4-04/20261112T204104Z-alm32-scope/`. Final SHA256SUMS SHA-256: `476f65d9492093dee9c478febd63afdd79c3fc6052fc5f5409eed715c85341dd`; all 21 members verified. The immutable pre-handoff QUALIFICATION-SHA256SUMS digest cited in Kaneo is `28973309c692273af19005edd8eed7dd0c3f81b7049a5917f88396a1c4ff41fe`. Final ledger additionally covers the exact handoff comment and task read-back; the sealed documentation snapshot predates this bookkeeping. No historical ledger changed. **ALMADIAL_M4_04_SBOM_READY_FOR_QA** under the owner-directed technical acceptance scope; no third-party redistribution clearance or publication authorization is implied.
