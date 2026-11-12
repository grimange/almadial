# AlmaDial — ALM-32 / M4-04: Independent SBOM, Licensing Evidence and Source-Provenance QA Report

## Acceptance Status and Governance

- **Task ID:** `rjxsredbmh3hh6kka8ki8qcp`
- **Canonical Work Package:** ALM-32 / M4-04
- **Task Title:** Audit SBOM, licensing, source notices and ownership
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted Baseline Commit:** `229c0d40fbe76c07389fae1716180a2dcd4b8bb4`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Project-Owner Scope Amendment Comment:** `j4e71lj3ftsqky84zcm4p7a3`
- **Developer Completion Comment:** `h7zst7oc4vs0e9yfjclqaixk`
- **Developer Evidence Ledgers:**
  - Primary Scope Ledger: `/home/ra/almadial-lab/evidence/m4-04/20261112T204104Z-alm32-scope/SHA256SUMS` (`476f65d9492093dee9c478febd63afdd79c3fc6052fc5f5409eed715c85341dd`, 21 files, 100% verified)
  - Qualification Ledger: `/home/ra/almadial-lab/evidence/m4-04/20261112T204104Z-alm32-scope/QUALIFICATION-SHA256SUMS` (`28973309c692273af19005edd8eed7dd0c3f81b7049a5917f88396a1c4ff41fe`, 19 files, 100% verified)
  - Decision Ledger: `/home/ra/almadial-lab/evidence/m4-04/20261112T202131Z-alm32-decision/SHA256SUMS` (`1bb5ee0222f25b290dfc71e06f85949d638dc5d70b0c60965e6d01157fbe1330`, 18 files, 100% verified)
  - Focused Ledger: `/home/ra/almadial-lab/evidence/m4-04/20261112T195545Z-alm32-focused/SHA256SUMS` (`da788f61536b5ec6ef88cf2c62c938ec4bf8fa6874eb2fe919a7ee16e6d338f2`, 14 files, 100% verified)
  - Initial Ledger: `/home/ra/almadial-lab/evidence/m4-04/20261112T192449Z-alm32/SHA256SUMS` (`25032ae0e8976ceb1d64c09d5be0d4b9b39da796328a9b3d0c92694931a7c29e`, 10 files, 100% verified)
- **Independent QA Ledger:** `/home/ra/almadial-lab/evidence/m4-04/qa-20261113T045500Z-alm32/SHA256SUMS` (`b0321d14e6ebb1d240097f31f3aed711f4d5b127057cae140696c36b2ac81559`, 8 files, 100% verified)
- **Developer Verdict:**
  ```text
  ALMADIAL_M4_04_SBOM_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M4_04_SBOM_ACCEPTED
  ```

---

## 1. Independent QA Acceptance Matrix

| # | Acceptance Criterion | Result | Detailed Evaluation & Evidence |
|---|---|:---:|---|
| 1 | **Scope Amendment & Governance** | **PASS** | Project-owner scope amendment (`j4e71lj3ftsqky84zcm4p7a3`) authoritatively distinguishes technical compliance qualification from legal third-party redistribution clearance. Neither downstream task dependencies nor task ownership were altered. [docs/implementation/m4-04-sbom-licensing-source-notices-and-ownership.md](file:///home/ra/Documents/almadial/docs/implementation/m4-04-sbom-licensing-source-notices-and-ownership.md). |
| 2 | **SPDX 2.3 Schema Validation** | **PASS** | `docs/compliance/m4-04/almadial.spdx.json` (`d5976e79109211ade884a9e26ee04dc4a5e10dd54a832767a4924180fc03d917`) validated against official SPDX 2.3 JSON schema via `test_generate_sbom.py`. Zero validation warnings or structural anomalies. |
| 3 | **Offline Reproducibility** | **PASS** | `python3 scripts/generate_sbom.py --check` and `python3 scripts/prepare_compliance_source.py --check` executed strictly offline without network access, matching checked-in artifacts bit-for-bit. |
| 4 | **Package & File Inventory Consistency** | **PASS** | Exactly 772 packages (1 root installer profile, 40 curated components, 731 AlmaLinux 10.2 base packages). Exactly 771 `DEPENDS_ON` relationships. 5 pinned source archives catalogued with 8,134 regular-file digests. Zero fabricated RPM payload checksums or synthetic licenses. |
| 5 | **License Evidence Completeness** | **PASS** | Exactly 39 distinct license text files catalogued under `docs/compliance/m4-04/licenses/`, reflecting the verified addition of the SIP.js embedded MD5 algorithm license (MIT / WTFPL v2). |
| 6 | **Third-Party Notices Agreement** | **PASS** | `docs/THIRD_PARTY_NOTICES.md` contains accurate component headers, declared and concluded licenses, and copyright statements matching all 40 curated records and the root AlmaDial installer. |
| 7 | **Embedded Exceptions & Unresolved Rights** | **PASS** | Exactly 16 curated components clean of exceptions; 24 records document specific exceptions; 18 unresolved license conclusions explicitly catalogued and bounded. |
| 8 | **Corresponding Source & Patch Provenance** | **PASS** | Candidate source archive `build/compliance/m4-04/almadial-compliance-source-candidate.tar.gz` (`8cb5466d3c5fec21c374a867d2bd7a5ecdbc9e69ba59fde8e635da4f68f14f94`, 274 members) validated. All 6 AlmaDial source patches reconstructed with exact SHA-256 matches and zero fuzz. Unsafe archive traversal and symlink escapes rejected. |
| 9 | **Distribution Classification** | **PASS** | Candidate source preparation is explicitly designated as internal technical audit preparation, NOT a fulfilled public distribution offer. Commercial redistribution clearance is explicitly withheld. |
| 10 | **Upstream & Original Ownership Boundaries** | **PASS** | AlmaDial original Go code and tooling strictly classified under MIT; upstream VICIdial, Asterisk, VICIphone, and OS components preserved under their native upstream licenses and copyright headers. |
| 11 | **Release Restrictions & Downstream Impact** | **PASS** | Unresolved rights in `line.js` (conflicting GPLv2 vs proprietary no-charge condition) and conditional network-source obligations (VICIphone AGPLv3 Section 13, VICIdial Nov 2007 AGPLv2 Section 2(d)) correctly documented as release gates without blocking technical milestone qualification. |
| 12 | **Vulnerability & Supply-Chain Follow-Up** | **PASS** | Advisory `GO-2026-6859` (CVE-2026-3864) regarding Go 1.27.3 compiler issue documented with reachability `NOT_VERIFIED`. Fixed upstream in Go 1.27.4; tracked as maintenance follow-up, non-blocking for ALM-32. |
| 13 | **Mandatory Regression Suite** | **PASS** | Local execution on Go 1.27.3: 1,464 Go test events (0 failures, 2 deliberate native skips), 0 data races under `-race`, 232 Python tests (including 10 compliance suite tests), scaffold validation clean, `git diff --check` clean. |
| 14 | **Evidence Integrity** | **PASS** | All developer ledgers verified across 5 distinct timestamps. Independent QA ledger established under `/home/ra/almadial-lab/evidence/m4-04/qa-20261113T045500Z-alm32/` (`b0321d14e6ebb1d240097f31f3aed711f4d5b127057cae140696c36b2ac81559`, 8 JSON records, 100% verified). |

---

## 2. Scope Amendment and Governance Findings

The canonical ALM-32 scope was amended with explicit project-owner authority via Kaneo comment `j4e71lj3ftsqky84zcm4p7a3` and updated in the authoritative task record.

### Key Governance Verifications
1. **Separation of Technical Audit and Legal Clearance:**
   The amendment recognizes that an open-source technical audit can verify inventory completeness, reproducible tooling, patch provenance, and explicit exception cataloguing, while explicitly withholding final third-party commercial redistribution clearance.
2. **Preservation of Exception History:**
   The amendment did not erase or whitewash historical licensing ambiguities (such as the contradictory terms in `line.js`). Instead, it mandates strict, explicit accounting of these items as documented exceptions.
3. **No Downstream Task Cascades:**
   The amendment did not alter downstream dependency graphs. ALM-32 remains the prerequisite for ALM-34, and ALM-28 remains accepted.
4. **Ownership Preservation:**
   Task ownership remained assigned to Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) throughout development and QA review. AGY performed independent QA without taking ownership of implementation tasks.

---

## 3. SPDX 2.3 Document and Inventory Validation

The technical SBOM was validated against the official SPDX 2.3 specification:
- **Canonical SBOM Location:** [docs/compliance/m4-04/almadial.spdx.json](file:///home/ra/Documents/almadial/docs/compliance/m4-04/almadial.spdx.json)
- **SHA-256 Digest:** `d5976e79109211ade884a9e26ee04dc4a5e10dd54a832767a4924180fc03d917`
- **Total Packages:** 772
  - **1 Profile Package:** `SPDXRef-Package-almadial-installer`
  - **40 Curated Components:** Embedded web assets, JavaScript libraries, Perl CPAN modules, audio sound packs, and upstream applications.
  - **731 Host RPM Packages:** AlmaLinux 10.2 base packages captured from the accepted clean-host baseline. Zero fabricated payload digests or synthetic licenses were injected for OS packages; RPM license tags were preserved as declared.
- **Relationships:** Exactly 771 `DEPENDS_ON` relationships linking the root installer package to each dependency.
- **Source Archive Inventory:** 5 pinned upstream source archives (`asterisk`, `asterisk-core-sounds-en-wav`, `asterisk-extra-sounds-en-wav`, `vicidial`, `viciphone`), covering 8,134 regular files with verified SHA-256 digests.
- **Offline Generation Verification:** `python3 scripts/generate_sbom.py --check` executed offline and produced exact bit-for-bit agreement with the checked-in document.

---

## 4. Licensing Evidence, Third-Party Notices, and Curated Exceptions

### License Evidence Catalog
Under [docs/compliance/m4-04/licenses/](file:///home/ra/Documents/almadial/docs/compliance/m4-04/licenses/), exactly 39 license text files are maintained.
The previous discrepancy of 38 files was resolved by Codex:
- The embedded MD5 implementation in SIP.js contains an explicit dual-license header (MIT / WTFPL v2).
- The corresponding license text file was added to the license directory, bringing the total count to exactly 39 files.
- Verified in [scripts/tests/test_generate_sbom.py](file:///home/ra/Documents/almadial/scripts/tests/test_generate_sbom.py) via `test_all_curated_license_files_exist`.

### Third-Party Notices
[docs/THIRD_PARTY_NOTICES.md](file:///home/ra/Documents/almadial/docs/THIRD_PARTY_NOTICES.md) was reviewed for exhaustiveness and accuracy:
- All 40 curated records have dedicated notice sections detailing component identity, upstream origin, declared license, concluded license, copyright holders, and relevant license text.
- AlmaDial original code is cleanly distinguished under MIT licensing.

### Curated Records Exception Tally
- **Total Curated Records:** 40
- **Clean Records (Without Exceptions):** 16
- **Records with Documented Exceptions:** 24
- **Unresolved License Conclusions:** 18 (e.g., dual-licensing conflicts, missing license files in legacy Perl modules, ambiguous headers).
- **Outcome:** Every exception is explicitly identified in `almadial.spdx.json` and documented in `docs/THIRD_PARTY_NOTICES.md`.

---

## 5. Corresponding Source Preparation and Patch Provenance

AlmaDial provides reproducible scripts to reconstruct modified upstream source packages:
- **Preparation Tool:** [scripts/prepare_compliance_source.py](file:///home/ra/Documents/almadial/scripts/prepare_compliance_source.py)
- **Candidate Archive:** `build/compliance/m4-04/almadial-compliance-source-candidate.tar.gz`
- **Candidate Digest:** `8cb5466d3c5fec21c374a867d2bd7a5ecdbc9e69ba59fde8e635da4f68f14f94`
- **Total Members:** 274 files and directories.
- **Patch Reconstruction Tests:**
  All 6 AlmaDial source patches applied during installation were verified against upstream archives:
  1. MariaDB initial schema and configuration patch
  2. Asterisk build and PJSIP configuration patch
  3. VICIdial AST_DB_optimize and crontab patch
  4. VICIphone configuration and template patch
  5. WebRTC confbridge template patch
  6. PHP-FPM Apache pool integration patch
  All 6 patches applied cleanly with zero fuzz, producing bit-for-bit identical trees verified against target checksums.
- **Security Boundaries:**
  The archive generator rejects unsafe members (path traversal `../`, absolute paths, symlinks pointing outside the archive). Tested via `test_prepare_compliance_source.py`.

---

## 6. Distribution Boundaries, Rights Exceptions, and Legal Clearance

In accordance with the scope amendment, the following distribution restrictions are formally documented:

| Component | Status | Restriction Details | Resolution Gate |
|---|---|---|---|
| `line.js` | `UNRESOLVED_RIGHTS` | Upstream VICIdial admin asset contains contradictory licensing in its header: GPLv2 boilerplate combined with proprietary no-charge restriction. | Commercial distribution blocked. Requires upstream clarification or replacement with compliant alternative prior to public release. |
| `VICIphone` | `CONDITIONAL_AGPLv3_OFFER_REQUIRED` | VICIphone is licensed under GNU AGPLv3. Section 13 mandates that network-interacting users be offered the corresponding source code. | A network-facing source download mechanism must be deployed before external distribution or hosted service offering. |
| `VICIdial` | `CONDITIONAL_AGPLv2_NOV2007_REVIEW_REQUIRED` | VICIdial uses a custom November 2007 revision of AGPLv2. Section 2(d) imposes specific source disclosure requirements. | Legal counsel review required prior to commercial binary redistribution. |
| `Go 1.27.3` | `SECURITY_ADVISORY_FOLLOWUP` | Advisory `GO-2026-6859` (CVE-2026-3864) affects the Go 1.27.3 compiler. | Symbol reachability is `NOT_VERIFIED`. Fixed upstream in Go 1.27.4. Non-blocking for ALM-32; scheduled for toolchain maintenance. |

**Formal Legal Clearance Status:** `NOT_GRANTED`. AlmaDial is accepted for technical compliance preparation, not approved for external commercial redistribution.

---

## 7. Regressions and Verification Evidence

### Test Execution Summary
All mandatory regressions passed locally under Go 1.27.3 on Linux x86_64:
- **`go vet ./...`:** Clean, zero warnings.
- **`go mod verify`:** All modules verified.
- **`go mod tidy check`:** Clean, zero drift.
- **`go test ./...`:** 1,464 test events completed successfully (0 failures, 2 deliberate native environment skips).
- **`go test -count=1 -race ./internal/...`:** Zero data races detected.
- **Scaffold Validation:** `python3 scripts/validate_scaffold.py` verified 11 required files and 4 skills.
- **Python Test Suite:** 232 unit tests passed in 2.736s (`scripts/tests/test_generate_sbom.py`, `scripts/tests/test_prepare_compliance_source.py`, and full repository test discovery).
- **`git diff --check`:** Clean, zero whitespace errors or merge markers.

### Evidence Ledgers
- Developer evidence ledgers in `/home/ra/almadial-lab/evidence/m4-04/20261112T204104Z-alm32-scope/` (21 files, SHA-256 `476f65d9492093dee9c478febd63afdd79c3fc6052fc5f5409eed715c85341dd`) and `QUALIFICATION-SHA256SUMS` (19 files, SHA-256 `28973309c692273af19005edd8eed7dd0c3f81b7049a5917f88396a1c4ff41fe`) were 100% verified.
- Independent QA evidence ledger recorded under `/home/ra/almadial-lab/evidence/m4-04/qa-20261113T045500Z-alm32/` (8 files, SHA-256 `b0321d14e6ebb1d240097f31f3aed711f4d5b127057cae140696c36b2ac81559`).

---

## 8. Final Independent Acceptance Determination and Next Task Recommendation

### Verdict
```text
ALMADIAL_M4_04_SBOM_ACCEPTED
```

All 14 acceptance criteria under the authorized ALM-32 amended scope have been independently verified and passed. Task ALM-32 is accepted (`done`). Codex's role as implementation owner is preserved.

### Recommended Next Task
The next canonical task in milestone M4 is:
- **Task ID:** `e3y45lk9xkrmim6ujhemrst3`
- **Canonical Ticket:** **ALM-33 / M4-05**
- **Task Title:** Prove recording backup and documented restoration
- **Status:** Unstarted / Ready for planning.
