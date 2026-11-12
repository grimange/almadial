# Third-party notices and distribution review

ALM-32 review snapshot: `229c0d40fbe76c07389fae1716180a2dcd4b8bb4`, 2026-11-13.
This notice set records source evidence and outstanding release conditions. It is not legal signoff or permission to publish.

The root MIT LICENSE covers AlmaDial-original code only. On 2026-11-13 the operator confirmed authority to license that code under MIT, including the employer/publication authority asked about. This is an attributable rights-holder attestation, not an assignment of third-party rights. Copyright: Ronniel Allan Castañito, 2026. Upstream code, derived patches and embedded assets retain their own terms.

## Primary components and required handling

| Component | Exact identity and evidence | Attribution / handling |
|---|---|---|
| Go standard library | Go 1.27.3; BSD-3-Clause; [full license](compliance/m4-04/licenses/Go/LICENSE) and [patent grant](compliance/m4-04/licenses/Go/PATENTS) | The Go Authors, copyright 2009; include notices in binary distributions. Toolchain vendored module versions and their separate license/patent files are listed in the inventory. |
| Go YAML | go.yaml.in/yaml/v3 v3.0.4; [LICENSE](../third_party/go-yaml/LICENSE), [NOTICE](../third_party/go-yaml/NOTICE), [Apache-2.0 full text](compliance/m4-04/licenses/Apache-2.0.txt) | Kirill Simonov, 2006–2011, MIT for libyaml-derived files; Canonical Ltd., 2011–2019, Apache-2.0 for remaining files. Include both license families and NOTICE with linked binaries. |
| Go dependency tests | gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405; [BSD-2-Clause](compliance/m4-04/licenses/check.v1/LICENSE) | Gustavo Niemeyer, 2010–2013; test dependency, not installer binary content. Preserve if distributing test sources. |
| Asterisk | 22.12.0; [LICENSE](compliance/m4-04/licenses/asterisk-22.12.0.tar.gz/asterisk-22.12.0/LICENSE), [COPYING](compliance/m4-04/licenses/asterisk-22.12.0.tar.gz/asterisk-22.12.0/COPYING) | Sangoma US Inc. and contributors; GPL-2.0-only core with exact linking exceptions described in LICENSE. Bundled code has separate terms; inspect nested notices, do not apply core GPL as a blanket conclusion. Retain corresponding source/build details if distributing binaries. |
| PJPROJECT | 2.17; Asterisk third-party commit 52e2f32de1b11e40844bb007749a327862f75275; [COPYING](compliance/m4-04/licenses/pjproject-2.17.tar.bz2/pjproject-2.17/COPYING) | Teluu and other contributors; GPL-2.0-or-later core. Nested source notice inventory includes separately licensed codec, SRTP, WebRTC, Abseil and other code. Presence does not prove compilation; build-level license closure remains open. |
| Net::Telnet | 3.05; [README terms](compliance/m4-04/licenses/Net-Telnet-3.05.tar.gz/Net-Telnet-3.05/README) | Jay Rogers, 1997, 2000, 2002, 2013, 2021; same terms as Perl itself. LicenseRef preserves the exact declaration rather than guessing a specific Perl-license expression. |
| VICIdial | r4037; official 2026-11-05 archive; [LICENSE.txt](compliance/m4-04/licenses/vicidial-r4037.zip/2026-11-05/docs/LICENSE.txt) | Exact file calls itself AFFERO GENERAL PUBLIC LICENSE, AGPLv2, November 2007, copyright 2007 Affero Inc. Use LicenseRef-VICIdial-AGPLv2-2007, not GNU AGPLv3. Individual files have additional notices. Network-source and modified-source obligations need qualified review for actual hosting/delivery. |
| VICIphone | 24e98fccfa7b7079fff970e02fca5f002ac1f2cc; full LICENSE in evidence inventory | Michael Cargile, copyright 2016; source headers say AGPL-3.0-or-later. Retain notices and provide corresponding modified source as applicable. Generated deployment changes are listed below. |
| SIP.js | 0.20.1; [full release license](compliance/m4-04/licenses/SIP.js-0.20.1/LICENSE.md) | Per-portion MIT AND WTFPL; Junction Networks copyright 2020 in release LICENSE, 2014–2022 in asset header; JsSIP portions copyright 2012–2013 José Luis Millán. Embedded MD5 notice credits Joseph Myers, André Cruz and Raymond Hill; Raymond Hill copyright 2014 MIT, André Cruz copyright 2015 WTFPL v2. [Complete embedded notices](compliance/m4-04/licenses/SIP.js-0.20.1/embedded-notices.txt) are included; do not apply MIT alone to the complete asset. |
| SPDX schema | Unmodified SPDX 2.3 JSON schema, SPDX specification contributors; [provenance](compliance/m4-04/schema-provenance.json), [CC-BY-3.0 text](compliance/m4-04/licenses/SPDX-spec/LICENSE) | Included as validation data; attribution and license retained. Not runtime installer code. |

## Embedded application assets

| Asset | Pinned version | Source-backed declaration | Attribution |
|---|---|---|---|
| Chart.js | 3.6.1 and 2.1.6, separate files | MIT | Chart.js contributors, 2021; Nick Downie, 2016 |
| dygraphs | exact r4037 file digest | MIT | Dan Vanderkam, Google, mirabilos and other authors in preserved header |
| pure-knob | VERM asset and r4037 asset, separate digests | Apache-2.0 | André Plötze, 2018; full Apache text included above |
| wz_jsgraphics | 2.33 | LGPL-2.1-or-later | Walter Zorn, 2002–2004; preserve header and applicable source/relinking obligations |
| line.js | exact r4037 file digest | **CONFLICT: GPL of unspecified version plus “cannot charge” restriction** | Balamurugan S., 2005; Jexp support attribution. Do not normalize this to GPL or claim redistribution clearance. Upstream clarification/qualified review required. |

Exact paths, file digests and bounded source headers are in [source-headers.json](compliance/m4-04/source-headers.json) and [source-files.json](compliance/m4-04/source-files.json). Extracted full-text license files are indexed by [license-evidence.json](compliance/m4-04/license-evidence.json). Header review is not a complete file-by-file legal audit.

## Modifications and delivery boundary

The installer downloads five digest-pinned archives into an operator-controlled cache and compiles/deploys locally. It does not bundle those archives in the repository. OS RPMs come from signed repositories or the base image. The installer binary links Go and YAML code. Source repository delivery also includes VICIdial-derived patches; the root MIT license does not supersede upstream terms.

The installation is **not unmodified upstream source**: `internal/provision/vicidial_source.go` changes procps invocations in ADMIN_keepalive_ALL.pl, AST_manager_send.pl, AST_VDauto_dial.pl and AST_VDremote_agents.pl. `internal/provision/phone_source.go` changes VICIphone PHP logging and JavaScript logging/register/reconnect behavior. SIP.js is not modified by these transformations. Three tracked VICIdial patches and the transformation source digests are in [patch-and-build-inventory.json](compliance/m4-04/patch-and-build-inventory.json).

Before any binary/image/source release or hosted modified application is approved, select its exact payload, include required full license texts/notices, resolve the line.js conflict, complete embedded license/build mapping, and establish corresponding-source/network-source delivery for modified copyleft components. Documentation here does not itself implement or legally approve that delivery. No release/image publication occurred in this task.

## Machine-readable inventory

[SPDX 2.3 SBOM](compliance/m4-04/almadial.spdx.json), [component inventory](compliance/m4-04/components.json), [731-RPM host inventory](compliance/m4-04/rpm-inventory.json), and [ALM-32 report](implementation/m4-04-sbom-licensing-source-notices-and-ownership.md). Unknown package repository origins and license conclusions remain explicit. Installed OS inventory is not a claim that AlmaDial redistributes the OS.

## Focused ALM-32 continuation

[Component coverage](compliance/m4-04/component-coverage.json) maps every existing curated component/notice record to evidence, membership, attribution, modification and delivery categories. NOASSERTION declarations remain explicitly unresolved; mapping completeness is not legal clearance or proof of linked build inclusion. All source archive notice-file candidates identified by the bounded filename check are already represented in the evidence set; license-bearing source comments outside that heuristic are not certified complete.

[`line.js` finding](compliance/m4-04/line-js-finding.json): **UNRESOLVED_RIGHTS**. The explicit no-charge wording is not merely a statement that downloading costs nothing. It appears in the same grant as the unspecified GPL assertion, with no distinct rights/use explanation. The file is installed unchanged and referenced by 15 deployed report pages; it is not excluded from this review. No rights-holder clarification was obtained.

[Distribution model](compliance/m4-04/distribution-model.json) separates repository/binary delivery, operator-staged upstream downloads, browser-served code, local builds and OS packages. The local source-preparation command described in the [regeneration guide](compliance/m4-04/README.md) packages exact upstream archives, six modified files and their diffs, tracked build/install sources, three separate not-applied derived patches, full license evidence and this notice. It does not publish an archive, deploy a network source offer, resolve conditional network-source terms, or prove that installed modified files carry the required prominent dated notices. Those release/deployment decisions remain explicit blockers.
