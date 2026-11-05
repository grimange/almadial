# ALM-2 / M0-02 — Candidate stack evaluation and QA handoff

Date: 2026-11-05 UTC (2026-11-06 Asia/Manila). Implementer: Codex. Repository input: `bba4d144e246745fb0e4dae4cd364d5d009a397c`. These changes are uncommitted. Task: ALM-2, `kzj273ypahs82zegi1wwjfag`, AlmaDial project `ghritpte6e40o5ayktgz0vh1`.

**SUPPORTED_CANDIDATE — implementation-side qualification complete; independent QA pending.** The manifest remains `candidate`. No application build, installation, PHP/Perl/SQL execution, WebRTC deployment or telephony compatibility test was performed. Runtime compatibility is **NOT_RUN**.

## Canonical deliverables

- [Dependency matrix and source/provenance/distribution decisions](m0-02-dependency-matrix.md): 113 investigated package names, exact identities, source references, module/interface findings, risk owners and reproduction commands.
- [Machine-readable candidate evidence](m0-02-candidate-stack.json): exact RPM NEVRAs/origins, EPEL metadata and package hashes, five upstream source identities, 51 Perl module/pragmas with example source locations, manifest pins and open gates.
- [Candidate manifest](../../manifests/stack.lock.yaml): existing schema version and keys retained, with exact component pins and explicit provenance/runtime states. `TBD-M0` remains only for full-M0 acceptance evidence; it is not presented as a verified component pin.
- [Reproduction verifier](../../scripts/verify_candidate_stack.py) and [negative-case tests](../../scripts/tests/test_verify_candidate_stack.py): stdlib-only, fixed named VM, explicit argv, deadlines, existing-output rejection and machine-readable results. No package installation or upstream script execution.

## Candidate and availability decision

| Component | Candidate | Qualification boundary |
|---|---|---|
| Host | AlmaLinux 10.2 x86_64, accepted ALM-1 image | Current OS/architecture and named snapshot inspected; no repeated host inventory or restore |
| Asterisk | 22.12.0 LTS | Official archive/checksum and detached signature verified; build/runtime NOT_RUN |
| VICIdial | SVN r4037, version 2.14b0.5, schema 1748 | Exact source export inspected; authenticated acquisition unresolved; privileged execution HOLD |
| MariaDB | 3:10.11.18-1.el10_2.alma.2.x86_64 | AppStream server/client/devel/backup identities; schema/authentication NOT_RUN |
| Apache | 0:2.4.63-13.el10_2.8.x86_64 | AppStream; mod_ssl epoch 1, same version/release |
| PHP | 0:8.3.36-1.el10_2.x86_64 | AppStream CLI/FPM/extensions; primary mysqli route requires later tests |
| Perl | 4:5.40.2-515.el10_2.x86_64 | AppStream; module load/compilation NOT_RUN |
| PJPROJECT / SRTP / Jansson | 2.17 bundled / 0:2.3.0-16.el10 / 0:2.14-3.el10 | Pinned Asterisk third-party artifact; SRTP/Jansson development RPMs in CRB |
| VICIphone / SIP.js | 3.0 at `24e98fccfa7b7079fff970e02fca5f002ac1f2cc` / 0.20.1 | Exact archive/bundle digests; browser/media NOT_RUN |
| Net::Telnet | 3.05 | Attributable CPAN archive and hash; source build/test NOT_RUN |

85 matrix rows resolve to enabled AlmaLinux repositories, including `php-curl` → `php-common` and `zlib-devel` → `zlib-ng-compat-devel`. Seventeen resolve to official isolated EPEL 10.2 metadata; EPEL was not enabled on the guest. Eleven have no selected RPM: bundled PJPROJECT, CPAN Net::Telnet, the absent `perl-core` metapackage and eight excluded optional feature dependencies. Thirty exact names are absent from enabled repositories. None of those counts is a count of proven runtime incompatibilities.

Required core choices include EPEL `perl-Switch` and `perl-Crypt-Eksblowfish`, distribution DBI/DBD::mysql and the upstream-bundled Asterisk::AGI module. Missing optional email/SFTP/report-path dependencies are visible and excluded from the first core telephony experiment, not silently treated as installed or supported. Package transaction closure and RPM signature verification remain later pre-install gates.

## Source and integration decisions

Retain Asterisk 22 as the preferred candidate. VICIdial r4037 has an explicit 18/20/22 installation branch, PJSIP wizard generation and integrated ConfBridge code. This establishes source-level paths only. M0-03 must include `res_pjsip_config_wizard` alongside the requested PJSIP/WebSocket/SRTP/crypto/ConfBridge/AMI/AGI modules. M0-05 and M0-06 verify actual generated configuration, conference allocation, AMI/AGI actions, dialing and recording.

The legacy PHP `mysql_connect` route is incompatible with PHP 8 if executed; the main agent page selects mysqli instead. Route reachability, mysqli exceptions, MariaDB authentication/SQL modes and schema imports require M0-04 execution. No broad upstream patch, SQL-mode weakening or version substitution is approved.

**VICIdial source trust decision: HOLD before privileged execution/distribution.** r4037 was exported from the official unauthenticated SVN endpoint; a tested HTTPS endpoint failed certificate hostname validation. Two SVN ports and a local digest do not authenticate the source. The older official HTTPS ZIP does not establish r4037 identity. M0-04 requires an upstream-authenticated r4037 artifact or independently approved trusted mirror/attestation. This unresolved downstream gate must remain visible during QA; no provenance exception has been granted.

The Asterisk signature is cryptographically valid with fingerprint `F2FC93DB7587BD1FB49E045A5D984BE337191CE7`. The key ID was linked by an upstream maintainer; GPG Web-of-Trust remains undefined. Independent QA reviews the association. Source/license inventories distinguish VICIdial's upstream Affero GPL v2 text, VICIphone AGPLv3, bundled JavaScript notices, Asterisk/PJPROJECT terms and Perl-licensed Net::Telnet. No third-party source was vendored or redistributed in AlmaDial.

## Evidence and validation

Protected operator-home evidence reference: `almadial-lab/evidence/m0-02/20261105T162726Z/`. The private mode-0700 parent contains original UTC argv/exit-code/stdout/stderr records, downloads, SVN export, source inspection and GPG records. Historical failed queries remain intact and are explained in the matrix. No raw VM metadata is stored in Git.

`validation/evidence-files.sha256` covers 89 retained records/artifacts, including the canonical SVN file-digest manifest and reproduction records. Its SHA-256 is `95780bc1f6ddf7d2ba77cb94794153389f291cf44463020faba7a627945a41ef`. This ledger is an integrity snapshot, not independent acceptance or source authentication. The complete SVN tree is checked separately by the verifier.

| Executed command/check | Actual result |
|---|---|
| `python3 scripts/validate_scaffold.py` | Exit 0; 11 required files / 4 expected skills |
| `python3 -m unittest discover -s scripts/tests -v` | Exit 0; 40 tests, including 16 candidate-verifier tests |
| `python3 -m py_compile scripts/verify_candidate_stack.py scripts/tests/test_verify_candidate_stack.py` | Exit 0 |
| `python3 scripts/verify_candidate_stack.py` | Exit 0; LOCAL consistency PASS; runtime NOT_RUN |
| `python3 scripts/verify_candidate_stack.py --live --output-dir "$HOME/almadial-lab/evidence/m0-02/20261105T162726Z/reproduction" --source-dir "$HOME/almadial-lab/evidence/m0-02/20261105T162726Z/sources"` | Exit 0; live package/source consistency PASS; runtime NOT_RUN |
| Same verifier with `--replay` replacing `--live --output-dir` | Exit 0; retained package/source consistency PASS |
| `git diff --check` | Exit 0 |
| Official Asterisk SHA-256 and EPEL compressed/open primary metadata checksums | All match retained upstream records |
| Guest installed RPM inventory before/after | Identical; no packages installed, removed or upgraded |

Command records for the final local/replay checks are in `validation/`; the live run is represented by its original `reproduction/` records. Tests cover missing packages/pins, newer versions, malformed/empty metadata, wrong OS/repositories, command failures/timeouts, RPM mutations, source tampering/symlinks, manifest promotion/duplicate keys, matrix drift, output collisions and isolation to the authorized VM. Passing these checks does not establish application integration.

The implementation slice comprises this report, the matrix, JSON companion, candidate manifest, verifier and its tests. The pre-existing untracked `scripts/verify_agy_overlay.py` was preserved untouched. No commit, push, release, service operation, snapshot operation or downstream task execution occurred.

## Independent QA and next gate

ALM-2 is in **QA Review** (`in-review`), owned by **Codex**, authorized member ID `aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`. The initial prerequisite ALM-1 was read back as Accepted before assignment. Self-assignment, Ready and In Progress transitions were individually read back. Evidence comment `unkrz9gdax4ihhq1okznwh47`, created `2026-11-05T16:52:09.235Z`, was retrieved with exactly matching content before the QA transition. A subsequent `get_task` verified QA Review and retained ownership, task ID, project, title, description, priority, dates and position.

Native relation IDs `jkinkto2gkdunoukh415zhgy` (ALM-1 → ALM-2) and `j296alx8um418ub9x1ludhzf` (ALM-2 → ALM-3) were retrieved again and are unchanged. ALM-1 remains Accepted; ALM-3 remains Backlog and unassigned. Task labels, acceptance requirements and historical records were not edited. No other task was changed and ALM-2 was not Accepted.

Protected `validation/kaneo-final-readback.json` retains the assignment/status readbacks, final task, native relations, exact comment and comparison results. SHA-256: `b4931d8389300b0cb99a79934828c7023673d8362d70ab35ec4c3a2463f04939`. It was created after the 89-file evidence ledger, so its digest is recorded separately. The comment records SHA-256s for the five implementation/matrix/manifest artifacts; the verifier's source SHA-256 is `19570820015fb45b63cfa4af941a39667f3441994e93b1d448d9959202f388df`. No new commit identity is claimed.

Independent reviewer: reproduce local tests and exact source digests, inspect package query records and EPEL metadata, verify the Asterisk signer association, and review license boundaries and the VICIdial source HOLD. Preserve implementer ownership. Acceptance concerns candidate qualification, not runtime compatibility or full M0 promotion.

After independent ALM-2 acceptance, the native graph identifies **ALM-3 / M0-03 — Prove pinned Asterisk modules and native PJSIP WebRTC echo**, task `h2d8fcqh6uqem6rczw1fckz0`. Recommend Codex with the highest approved coding model, high reasoning, acting as Linux telephony/build engineer. Use the pinned source/cache, Multipass, DNF/RPM, compiler tools, Asterisk CLI, TLS/media diagnostics and Kaneo. Obtain separate authorization for its exact package installations/build/runtime actions on the named disposable VM and preserve recovery evidence. Verify modules, authenticated WSS/PJSIP registration and real native echo evidence under its own acceptance contract; no VICIdial root execution while the source HOLD remains unresolved.

## Supporting self-check amendment — 2026-11-05 UTC

The [QA handoff and supporting self-check](../reviews/m0-02-candidate-stack-independent-qa.md) preserves the distinction between this Codex implementation session and the requested separate AGY reviewer. Independent acceptance and the conditional local commit remain pending. Fresh official evidence confirms final stable Asterisk 22.12.0, rather than an RC; the candidate pins did not change. The original 89-file ledger matches in full, and corrected live/replay checks reproduce package/source identities without application installations.

A confirmed local-verifier consistency defect was corrected: source rows and primary runtime RPM rows are now compared directly with the actual manifest, rather than relying solely on the copied JSON manifest pins. Three regression tests increase the suite from the historical **40/16** total/candidate tests above to **43/19**, all passing. Corrected verifier SHA-256: `e8882bd395919d9292f2314dd0452ffee6c78170764b52258980962433412236`; corrected tests SHA-256: `ab8e3f201188cf4f1389e2f1f7058e5bb3e02bbc2deceab9d4bfbb36fd898148`. These supersede those files' historical implementation hashes for the current uncommitted slice; the original comment and ledger remain intact.

Separate protected evidence: `almadial-lab/evidence/m0-02/self-check-20261105T173232Z/`. ALM-2 remains QA Review with Codex ownership, the manifest remains `candidate`, runtime compatibility remains NOT_RUN, and the VICIdial source-execution HOLD is unchanged. No commit or acceptance is claimed by this amendment.
