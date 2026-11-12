# AlmaDial — ALM-33 / M4-05: Independent Recording Backup and Restoration QA Report

## Acceptance Status and Governance

- **Task ID:** `e3y45lk9xkrmim6ujhemrst3`
- **Canonical Work Package:** ALM-33 / M4-05
- **Task Title:** Prove recording backup and documented restoration
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Accepted Baseline Commit:** `559b03a99264db76f8b0811ca53ef91e9608de95`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Evaluated Source Host:** `almadial-alm29-clean-d` (`10.160.90.38`, Machine ID `4c7b492940923838a0700adba65ed3eb`, AlmaLinux 10.2, 731 RPMs, snapshot `alm29-pristine`)
- **Evaluated Target Host:** `almadial-alm29-clean-e` (`10.160.90.239`, Machine ID `ebf868d3a0853b4294cd85e589a12f5c`, AlmaLinux 10.2, 731 RPMs, snapshot `alm29-pristine`)
- **Developer Evidence Comment:** `go78hze2fk6637a89iqkgrjp`
- **Developer Evidence Ledger:** `/home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/SHA256SUMS` (`fa6bc01103f8c3f0ba40a606bbce8058599f1e61130fab0b1a024e16dff1ca32`, 58 files, 100% verified)
- **Independent QA Ledger:** `/home/ra/almadial-lab/evidence/m4-05/qa-20261113T075000Z-alm33/SHA256SUMS` (`6b4e802c14a8ed6c2c5b3e476ae24315badb8362bce36ce68bcaad7f39bafd54`, 10 files, 100% verified)
- **Developer Verdict:**
  ```text
  ALMADIAL_M4_05_RECORDING_RESTORE_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M4_05_RECORDING_RESTORE_ACCEPTED
  ```

---

## 1. Independent QA Acceptance Matrix

| # | Acceptance Criterion | Result | Detailed Evaluation & Evidence |
|---|---|:---:|---|
| 1 | **Canonical ALM-33 Scope** | **PASS** | Bounded strictly to manual recording/data recovery drill, isolated restoration, file/metadata integrity, SQL schema import, and failure handling. No whole-server disaster recovery or transactional rollback claimed. [docs/implementation/m4-05-recording-backup-and-documented-restoration.md](file:///home/ra/Documents/almadial/docs/implementation/m4-05-recording-backup-and-documented-restoration.md). |
| 2 | **Actual Backup Archive** | **PASS** | Protected archive `backup.private.tar` (1,914,880 bytes, SHA-256 `9971a9448c35eaa01f84a99eeb99aaac1c6a76bb4700d008fe15c2f50fea7ff5`) verified offline via `scripts/recording_backup.py verify`. Format `almadial-manual-backup-v1`. [backup-result.json](file:///home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/backup-result.json). |
| 3 | **Complete 48-File Inventory** | **PASS** | Exactly 48 regular files catalogued and packaged: 6 WAV recordings, 23 configuration files, 16 credential/TLS files, application SQL dump, call correlation TSV, and recovery JSON. Zero missing or extraneous members. |
| 4 | **Isolated Restoration** | **PASS** | Restored to isolated directory `/var/tmp/20261112T232400Z-alm33/restored` on `almadial-alm29-clean-e` (mode 0700). Active system paths (`/etc`, `/var/spool/asterisk/monitor`) preserved without modification. [restore-result.json](file:///home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/restore-result.json). |
| 5 | **File Hash & Metadata Preservation** | **PASS** | 48/48 restored files match source SHA-256 digests, byte sizes, file permission modes, numeric UID/GID ownership, and nanosecond modification times. |
| 6 | **Isolated SQL Restoration** | **PASS** | Application SQL dump (`f649fe248019cd849d12c1fd9964d8bf63c5b16e53e7423211209470d1c3e034`) imported into isolated schema `alm33_20261112232400`. Complete logical dump equality verified; 345 tables (337 MyISAM, 8 MEMORY). Active `asterisk` schema untouched. |
| 7 | **Two Call Identities** | **PASS** | Call IDs `1794450528.0` (14 s, `PJSIP/alm_s10_9061-00000000`) and `1794450529.3` (13 s, `PJSIP/alm_s10_9061-00000001`) successfully correlated between restored `call_log` database rows and MixMonitor recordings. |
| 8 | **Six Restored WAV Files** | **PASS** | Exactly 6 WAV files restored and verified (mixed, rx, and tx for each call). Restored file hashes agree bit-for-bit with original MixMonitor source recordings. [audio-verification.json](file:///home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/audio-verification.json). |
| 9 | **Audio Format & Content** | **PASS** | `ffprobe` verified format: PCM s16le, 8000 Hz, 1 channel (mono); durations 14.24 s and 12.68 s. `ffmpeg` decoded all 6 files with exit code 0 and zero decode errors. Normalized RMS (0.1172–0.2292) confirmed meaningful audio. |
| 10 | **Archive Corruption Rejection** | **PASS** | Tested in unit suite and natively on VM: modified/corrupted payload bytes cause checksum mismatch and fail with exit code 1 before destination directory creation. [native-invalid-backup-results.json](file:///home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/native-invalid-backup-results.json). |
| 11 | **Unsafe Member Rejection** | **PASS** | Archive verification rejects symlinks (`tarfile.SYMTYPE`), path traversal (`../`), absolute paths, duplicate entries, and excessive members prior to extraction. |
| 12 | **Repeat Restoration Safety** | **PASS** | Restoring over an existing destination fails with `FileExistsError` / `REJECT_EXISTING_DESTINATION`. No files are overwritten or merged into existing trees. Tested in CLI smoke and unit tests. |
| 13 | **Sensitive Backup Protection** | **PASS** | Backup archives (0600) and directories (0700) restricted to local root. No secrets in CLI argv or error outputs; no sensitive SQL dumps or audio files committed to Git. |
| 14 | **Source/Destination Preservation** | **PASS** | 107 managed configuration files, active databases, nftables rulesets, 731 packages, and service invocation identities remained bit-for-bit identical before and after the drill on both hosts D and E. [almadial-alm29-clean-d-preservation.json](file:///home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/almadial-alm29-clean-d-preservation.json). |
| 15 | **Operator Documentation** | **PASS** | Practical NOC workflow provided in [docs/operations/recording-backup-and-restore.md](file:///home/ra/Documents/almadial/docs/operations/recording-backup-and-restore.md). Covers prerequisites, commands, consistency constraints, failure modes, and recovery boundaries. |
| 16 | **Regression & Evidence** | **PASS** | All regressions passed: 1,464 Go test events (0 failures, 2 deliberate skips), 0 data races under `-race`, 240 Python unit tests (including 8 focused backup tests), 6 CLI smoke cases, scaffold and diff checks clean. |

---

## 2. Backup Archive and Restoration Verification

### Inventory and Archive Structure
The backup archive was created using `scripts/recording_backup.py create` from source host `almadial-alm29-clean-d`:
- **Archive Size:** 1,914,880 bytes
- **SHA-256 Digest:** `9971a9448c35eaa01f84a99eeb99aaac1c6a76bb4700d008fe15c2f50fea7ff5`
- **Total Payload:** 1,847,233 bytes across 48 regular files
- **Inventory Coverage:**
  1. **Recordings (6 files):** `/var/spool/asterisk/monitor/alm-s11-1794450528.0[-rx|-tx].wav` and `/var/spool/asterisk/monitor/alm-s11-1794450529.3[-rx|-tx].wav`
  2. **Configuration (23 files):** Asterisk configurations (`/etc/asterisk/*`), `astguiclient.conf`, database connectors (`dbconnect_mysqli.php`), Apache HTTPD, PHP-FPM pools, systemd unit definitions, cron definitions, and firewalld policy (`almadial-s09.xml`).
  3. **Credentials & TLS (16 files):** Secret files under `/etc/almadial/secrets/`, TLS generation files and certificates under `/etc/pki/tls/almadial-s08/`.
  4. **Database & Metadata (3 files):** Quiescent logical database dump (`asterisk.sql`), call correlation export (`call-correlation.tsv`), and recovery metadata (`recovery.json`).

### Restoration Integrity
Restoration was performed on target host `almadial-alm29-clean-e` under isolated path `/var/tmp/20261112T232400Z-alm33/restored`:
- **Bit-for-Bit Hash Agreement:** All 48 files verified against their original SHA-256 digests.
- **File Modes and Ownership:** Restored files retained their original permissions (0600/0640) and numeric UIDs/GIDs (`root`, `asterisk`, `mysql`, `apache`).
- **Timestamps:** Nanosecond modification times preserved.
- **No Operational Clashing:** No files were copied or symlinked into live paths (`/etc`, `/var/www`, `/var/spool/asterisk`).

---

## 3. Database Recovery and Recording Correlation

### Database Restoration
- **Target Schema:** `alm33_20261112232400`
- **Quiescence Verification:** Asterisk confirmed 0 active channels during backup; logical dumps taken before and after backup collection were byte-identical.
- **Dump Verification:** The database dump (`f649fe248019cd849d12c1fd9964d8bf63c5b16e53e7423211209470d1c3e034`) was free of `CREATE DATABASE`, `USE`, original-schema qualifiers, triggers, or routines.
- **Logical Dump Equivalence:** A clean re-dump of the restored schema produced an identical SHA-256 digest (`f649fe248019cd849d12c1fd9964d8bf63c5b16e53e7423211209470d1c3e034`).
- **Table Count:** 345 tables imported (337 MyISAM tables, 8 MEMORY tables).
- **Isolation:** Active application database `asterisk` remained untouched; zero MySQL system grants or credentials were replayed.

### Call Identity and Audio Correlation
The restored database and audio files demonstrate complete end-to-end correlation:

| Call UniqueID | PJSIP Channel | Start Time | End Time | DB Length | Restored Audio Files | Restored Duration |
|---|---|---|---|---:|---|---:|
| `1794450528.0` | `PJSIP/alm_s10_9061-00000000` | 2026-11-12 10:28:48 | 2026-11-12 10:29:02 | 14 s | `alm-s11-1794450528.0.wav`<br>`alm-s11-1794450528.0-rx.wav`<br>`alm-s11-1794450528.0-tx.wav` | 14.24 s<br>14.24 s<br>14.24 s |
| `1794450529.3` | `PJSIP/alm_s10_9061-00000001` | 2026-11-12 10:28:49 | 2026-11-12 10:29:02 | 13 s | `alm-s11-1794450529.3.wav`<br>`alm-s11-1794450529.3-rx.wav`<br>`alm-s11-1794450529.3-tx.wav` | 12.68 s<br>12.68 s<br>12.68 s |

### Audio Integrity
- **Codec:** PCM 16-bit little-endian (`pcm_s16le`), mono (1 channel), 8,000 Hz sample rate.
- **Decoding:** Full ffmpeg decode verification (`ffmpeg -v error -i <file> -f null -`) returned exit code 0 with zero errors across all six files.
- **Audio Quality:** Normalized RMS levels range from 0.1172 to 0.2292, confirming meaningful signal rather than digital silence.

---

## 4. Failure Handling and Repeat Safety

The archive utility and restore workflow implement defensive validation:
1. **Pre-Extraction Verification:** The entire archive, manifest structure, member counts, and payload checksums are verified before any extraction starts.
2. **Corruption Rejection:** Corrupted archives fail with checksum error; no destination directory or partial files are created.
3. **Missing Members:** Archives missing declared manifest entries are rejected immediately.
4. **Symlink and Path Traversal Protection:** Symlink members, source/destination symlink parents, and paths containing `..` or absolute prefixes are rejected.
5. **Repeat Restoration Protection:** Re-running restore against an existing directory raises `FileExistsError` (`REJECT_EXISTING_DESTINATION`) without overwriting or merging.
6. **Test Coverage:** All 8 unit tests in `scripts/tests/test_recording_backup.py` and all 6 CLI smoke scenarios passed.

---

## 5. Installed-State Preservation

The developer's 107-item managed state inventory was verified across both hosts:
- **Host D (`almadial-alm29-clean-d`):** All 107 managed files (Asterisk configs, secrets, TLS, database configs, crons, systemd units), package inventory (731 RPMs), nftables ruleset, and service identities remained identical before and after backup collection.
- **Host E (`almadial-alm29-clean-e`):** All 107 managed files, package inventory (731 RPMs), nftables ruleset, and service identities remained identical before and after isolated restoration.
- **Health Checks:** `almadial status` and `almadial doctor` exit code 10 only for the expected short-lived lab certificate expiration warning; admin HTTPS (401 unauthenticated boundary) and Asterisk WSS (426 non-upgraded boundary) verified healthy.

---

## 6. Operator Recovery Documentation

[docs/operations/recording-backup-and-restore.md](file:///home/ra/Documents/almadial/docs/operations/recording-backup-and-restore.md) was reviewed and found practical and accurate:
- Provides clear step-by-step procedures for backup collection, validation, transfer, isolated restoration, and post-restore verification.
- Explicitly warns against unsafe patterns (extracting into `/`, overwriting active databases, busy-host backup without quiescence).
- Accurately states known limitations: no whole-server disaster recovery, no live configuration activation, and no transactional call rollback.

---

## 7. Mandatory Regression Suite

Conducted with Go 1.27.3 and Python 3.12.3:
- **`go vet ./...`:** PASS (clean, 0 warnings).
- **`go mod verify`:** PASS (all modules verified).
- **`go mod tidy -diff`:** PASS (zero drift).
- **`go test ./...`:** PASS (1,464 test events passed, 2 deliberate skips).
- **`go test -race ./...`:** PASS (0 data races detected).
- **`python3 scripts/validate_scaffold.py`:** PASS (11 required files, 4 skills).
- **`python3 -m unittest discover -s scripts/tests -v`:** PASS (240 tests passed in 2.760s, including 8 recording backup tests).
- **`git diff --check`:** PASS (clean).

---

## 8. Evidence Ledgers

### Developer Evidence
- Directory: `/home/ra/almadial-lab/evidence/m4-05/20261112T232400Z-alm33/`
- Ledger SHA-256 (`SHA256SUMS`): `fa6bc01103f8c3f0ba40a606bbce8058599f1e61130fab0b1a024e16dff1ca32` (58 files, 100% verified)

### Independent QA Evidence
- Directory: `/home/ra/almadial-lab/evidence/m4-05/qa-20261113T075000Z-alm33/`
- Ledger SHA-256 (`SHA256SUMS`): `6b4e802c14a8ed6c2c5b3e476ae24315badb8362bce36ce68bcaad7f39bafd54`
- Verified files:
  - `identity.json`
  - `ledger-verification.json`
  - `backup-inventory-verification.json`
  - `restoration-and-metadata.json`
  - `database-and-call-correlation.json`
  - `audio-decoding-results.json`
  - `failure-handling-and-repeat-safety.json`
  - `source-state-preservation.json`
  - `regression-summary.json`
  - `qa-results-summary.json`

---

## 9. Final Independent Acceptance Determination and Next Task Recommendation

### Final Verdict
```text
ALMADIAL_M4_05_RECORDING_RESTORE_ACCEPTED
```

All 16 acceptance criteria for ALM-33 / M4-05 have been independently verified and passed. Task ALM-33 is accepted (`done`). Codex's ownership as implementation owner is preserved.

### Recommended Next Task
The next canonical task in milestone M4 is:
- **Task ID:** `it8aq4utqix0uz8sjgxrtssk`
- **Canonical Ticket:** **ALM-34 / M4-06**
- **Task Title:** Complete unassisted independent operator acceptance
- **Status:** Unstarted / Ready for assignment.
