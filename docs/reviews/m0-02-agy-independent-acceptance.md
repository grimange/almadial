# AlmaDial Independent QA Acceptance — ALM-2 / M0-02 Candidate Stack Qualification

**Reviewer / Session:** Google Antigravity (AGY), separate independent review session  
**Reviewer Professional Role:** Principal Linux Packaging & Telephony Compatibility QA Engineer / Software Supply-Chain Verification Architect  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)  
**Implementer:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Target Task:** ALM-2 / M0-02 (`kzj273ypahs82zegi1wwjfag`), Kaneo Project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Repository Starting HEAD:** `bba4d144e246745fb0e4dae4cd364d5d009a397c`  
**Host Target:** `almadial-alma10` (read-only queries; no packages installed or modified)  
**Review Date / Time UTC:** 2026-11-05 UTC (Session local: 2026-11-06T02:02:00+08:00)  
**Historical Codex Self-Check:** Preserved in [`docs/reviews/m0-02-candidate-stack-independent-qa.md`](m0-02-candidate-stack-independent-qa.md)  
**Final Independent QA Verdict:** **`ALMADIAL_M0_02_ACCEPTED`**

---

## 1. Scope, Authority, and Independence Confirmation

This review constitutes the formal, independent quality assurance assessment of the candidate stack qualified under task **ALM-2 / M0-02: Qualify candidate versions and dependency availability**.

In compliance with repository governance (`AGENTS.md`, `.agents/skills/almadial-review/SKILL.md`, and `.agents/rules/almadial-kaneo-governance.md`):
- The implementer (Codex) conducted the initial version audits, identified and corrected a Medium-severity verifier defect, and submitted a self-check report with the explicit finding that independent QA remained blocked until a separate reviewer session verified the evidence.
- Google Antigravity (AGY) operates as a distinct reviewer, discovering its own authorized member identity through `kaneo-local` (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`). AGY does not assume ownership of the implementation task.
- The review is bounded to candidate qualification, dependency availability, cryptographic provenance, and tool verification. Application runtime execution, Asterisk compilation, database initialization, and WebRTC media paths remain **NOT_RUN** and are deferred to downstream tasks M0-03 through M0-07.

---

## 2. Source Baseline and Repository State

- **Starting Baseline:** Commit `bba4d144e246745fb0e4dae4cd364d5d009a397c` (`test(m0): accept AlmaLinux host baseline and inventory corrections`).
- **Reviewed ALM-2 Artifacts:**
  - `manifests/stack.lock.yaml` (retained in `status: candidate`)
  - `docs/compatibility/almalinux10-stack-evaluation.md`
  - `docs/compatibility/m0-02-dependency-matrix.md`
  - `docs/compatibility/m0-02-candidate-stack.json`
  - `docs/reviews/m0-02-candidate-stack-independent-qa.md` (historical self-check)
  - `scripts/verify_candidate_stack.py`
  - `scripts/tests/test_verify_candidate_stack.py`
- **Preserved Unrelated Artifacts:** Pre-existing untracked helper `scripts/verify_agy_overlay.py` remains uncommitted and unmodified.

---

## 3. Asterisk Release Identity Verification

1. **Official Release Status:**
   - Fresh official GitHub release API inspection confirmed tag `22.12.0`, `prerelease: false`, published `2026-11-05T13:15:10Z`.
   - The candidate is **Asterisk 22.12.0 LTS (final stable)**, resolving earlier RC index discrepancies.
2. **Artifact Integrity and Version Verification:**
   - Archive: `asterisk-22.12.0.tar.gz`
   - Archive SHA-256: `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` (verified against official `asterisk-22.12.0.sha256`).
   - Internal version file `asterisk-22.12.0/.version` extracted and confirmed: exactly `22.12.0`.
3. **Cryptographic Signature:**
   - Detached signature `asterisk-22.12.0.tar.gz.asc` verified with exit code 0 using GPG in an isolated keyring.
   - Good signature from `Asterisk Development Team <asteriskteam@digium.com>`, `VALIDSIG` fingerprint: `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`.
   - Web-of-Trust remains undefined (documented limitation; no stronger cryptographic attestation claimed).
4. **Bundled PJPROJECT Alignment:**
   - Extraction of `asterisk-22.12.0/third-party/versions.mak` confirms: `PJPROJECT_VERSION=2.17`.
   - Bundled checksum `asterisk-22.12.0/third-party/pjproject/pjproject-2.17.tar.bz2.md5` matches MD5 `baacc87418a95107657039269bc71538` and SHA-256 `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7`.

---

## 4. Assessment of Verifier Correction (Medium Defect)

### Defect Analysis
- **Observed Defect:** `check_local()` in `scripts/verify_candidate_stack.py` previously compared manifest leaf keys against JSON `manifest_pins` without cross-checking the actual `sources` and `packages` data rows. Consequently, contradictory source versions/digests or primary RPM versions in the JSON evidence were accepted as long as `manifest_pins` matched the manifest.
- **Correction:** `check_local()` was updated to directly enforce:
  1. The 5 required source entries (`Asterisk`, `PJPROJECT`, `Net::Telnet`, `VICIdial`, `VICIphone`) match the exact manifest versions, digests, and upstream URLs. Missing or duplicated source components fail.
  2. The 6 primary runtime RPM components (`mariadb`, `php`, `perl`, `apache`, `srtp`, `jansson`) match the exact manifest NEVRAs and repository origins.
- **Regression Tests Verified:**
  - `test_source_records_must_agree_with_manifest`: Rejects mutated version `0.0.0`, all-zero SHA-256, or mismatched URL.
  - `test_required_source_cannot_be_missing_or_duplicated`: Rejects deleted or duplicate source component entries.
  - `test_selected_runtime_package_must_agree_with_manifest`: Rejects mutated package NEVRA/version in `packages`.
- **Verdict:** PASS. The fix is minimal, robust, and correctly tested without collateral breakage.

---

## 5. Dependency Accounting and Live RPM Qualification

1. **Matrix Accounting (113 Total Entries):**
   - **Enabled AlmaLinux Repositories (85):**
     - AppStream: 68
     - BaseOS: 11
     - CRB: 6
   - **Isolated EPEL 10.2 Metadata (17):** Includes `perl-Switch`, `perl-Crypt-Eksblowfish`, `perl-Text-CSV_XS`, etc. (EPEL repository remains disabled on VM).
   - **No Selected RPM (11):** Bundled PJPROJECT (1), CPAN Net::Telnet (1), absent `perl-core` metapackage (1), and 8 excluded optional-feature dependencies (`perl-HTML-Strip`, `perl-Mail-Box`, `perl-Mail-IMAPClient`, `perl-Mail-POP3Client`, `perl-Net-FTPSSL`, `perl-Net-SFTP`, `perl-Net-SSH2`, `perl-Spreadsheet-Read`).
   - **Reconciliation of 30 Missing Requested Names:** 17 (EPEL) + 11 (No RPM) + 2 (Capability Replacements) = 30.
2. **Capability Replacements Verified on `almadial-alma10`:**
   - `php-curl` → `php-common-8.3.36-1.el10_2.x86_64`: Verified provides `php-curl` and `php-curl(x86-64)`.
   - `zlib-devel` → `zlib-ng-compat-devel-2.2.3-3.el10_1.x86_64`: Verified provides `zlib-devel = 1.3.1`, `zlib-devel(x86-64) = 1.3.1`, `pkgconfig(zlib) = 1.3.1.zlib-ng`, and `cmake(zlib) = 1.3.1`.
3. **Primary Runtime Component NEVRAs:**
   - MariaDB: `mariadb-server-3:10.11.18-1.el10_2.alma.2.x86_64` (AppStream)
   - Apache: `httpd-0:2.4.63-13.el10_2.8.x86_64` (AppStream; `mod_ssl-1:2.4.63-13.el10_2.8.x86_64`)
   - PHP: `php-fpm-0:8.3.36-1.el10_2.x86_64` (AppStream)
   - Perl: `perl-4:5.40.2-515.el10_2.x86_64` (AppStream)
   - libsrtp: `libsrtp-devel-0:2.3.0-16.el10.x86_64` (CRB)
   - Jansson: `jansson-devel-0:2.14-3.el10.x86_64` (CRB)

---

## 6. Source Integrity, Trust, and Licensing Boundaries

1. **VICIdial Source Trust HOLD:**
   - Candidate: SVN revision r4037, version 2.14b0.5, schema version 1748.
   - Canonical tree digest: `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` (verified across 2,511 exported files).
   - **Trust Boundary:** The official SVN transport (`svn://svn.eflo.net:3690`) does not authenticate upstream publisher identity, and HTTPS certificate validation fails. A reproducible tree hash confirms artifact identity across runs, not publisher authenticity.
   - **Enforced Boundary:** Privileged execution and distribution of VICIdial remain on **HOLD** pending an upstream-authenticated export or an independently approved trusted mirror in task M0-04. This hold is properly documented and does not block ALM-2 candidate qualification.
2. **VICIphone & SIP.js:**
   - Candidate: VICIphone 3.0 at commit `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`.
   - Archive SHA-256: `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8`.
   - Embedded SIP.js version 0.20.1 verified (SHA-256 `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b`).
   - Licensing: VICIphone AGPLv3, SIP.js/JsSIP MIT notices, and bundled licenses verified distinct from AlmaDial original code.
3. **Net::Telnet:**
   - CPAN release 3.05 (SHA-256 `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743`). Perl Artistic/GPL licensing confirmed.

---

## 7. Automated Test Reproduction and Artifact Integrity

1. **Executed Test Suite:**
   - `python3 scripts/validate_scaffold.py`: Exit code 0 (PASS: 11 required files, 4 skills).
   - `python3 -m unittest discover -s scripts/tests -v`: Exit code 0 (43/43 tests PASS in 0.131s, including 19 candidate verifier tests).
   - `python3 -m py_compile scripts/verify_candidate_stack.py scripts/tests/test_verify_candidate_stack.py`: Exit code 0 (clean compilation).
   - `python3 scripts/verify_candidate_stack.py`: Exit code 0 (LOCAL mode: candidate consistency PASS).
   - `python3 scripts/verify_candidate_stack.py --replay ... --source-dir ...`: Exit code 0 (REPLAY mode: package and source consistency PASS).
   - `git diff --check`: Exit code 0.
2. **Evidence Ledgers Verified:**
   - Original 89-file evidence manifest `validation/evidence-files.sha256`: SHA-256 `95780bc1f6ddf7d2ba77cb94794153389f291cf44463020faba7a627945a41ef` (89/89 files OK).
   - Self-check 45-file evidence manifest `evidence-files.sha256`: SHA-256 `666c34e720de44b2bf9b3cf2eed976ee9470897815a882509669cdde5101fddb` (45/45 files OK).
   - EPEL compressed primary metadata (`d4fff60dd066...`) and uncompressed primary metadata (`291980706b0d...`) verified against upstream `repomd.xml`.

---

## 8. Candidate Manifest State

- File: [`manifests/stack.lock.yaml`](file:///home/ra/Documents/almadial/manifests/stack.lock.yaml)
- Status: **`candidate`** (strictly preserved; not promoted to verified or production-ready).
- Verification fields: `runtime_state: NOT_RUN`, `fixture_id: TBD-M0`, `evidence_digest: TBD-M0`.
- All component pins, digests, and repository mappings match the verified candidate stack.

---

## 9. Confirmed Defects and Accepted Limitations

1. **Defects:** Zero remaining defects. The Medium-severity verifier consistency issue was completely and properly resolved.
2. **Accepted Limitations:**
   - Runtime execution, compilation, database initialization, WebRTC audio, and telephony integration remain `NOT_RUN`.
   - VICIdial SVN transport lacks cryptographic authentication; privileged execution is held until M0-04.
   - EPEL repository packages are qualified via isolated metadata only and remain uninstalled.
   - GPG Web-of-Trust for the Asterisk release key is undefined.

---

## 10. Independent QA Conclusion

All ALM-2 acceptance criteria have been satisfied with rigorous, reproducible evidence. The candidate stack is formally qualified for downstream milestone experiments.

**Final Independent QA Verdict:**  
**`ALMADIAL_M0_02_ACCEPTED`**
