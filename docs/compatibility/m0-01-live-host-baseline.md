# M0-01 live host baseline — blocked; transfer package prepared

**Historical no-VM attempt.** The later authorized `almadial-alma10` execution is recorded in [the measured live inventory and recovery report](m0-01-alma10-live-evidence.md). Real host evidence now exists; recovery was tested. That later run remains blocked on mandatory active firewall-rule evidence. The original report below is preserved without turning its NOT_RUN observations into results.

Recorded: 2026-11-04 23:46 UTC / 2026-11-05 Asia/Manila. Task: **ALM-1 / M0-01**, ID `p8urvq3yjtc76rktih9xlhe3`, project AlmaDial `ghritpte6e40o5ayktgz0vh1`.

**Verdict: ALMADIAL_M0_HOST_BASELINE_BLOCKED.** The operator explicitly confirmed **“no vm created yet”** during this assignment. No host was contacted, no inventory collector was run, and no runtime observations or compatibility results exist for this attempt. The developer workstation and synthetic fixture were not substituted for a lab VM. This report is a blocked-execution handoff, not measured host evidence.

## Verified source and missing observations

HEAD is the independently reviewed local commit `18b97eea1438e0d1be72276fd3018659a33a636f`; the initial working tree was clean. Collector, tests and fixture bytes were compared to that commit and to the accepted SHA-256 values in [the independent review](../reviews/m0-01-inventory-qa.md). All matched. Completed repository-only QA was not repeated; source and transfer integrity were checked for this assignment.

| Required observation | Actual evidence | State |
|---|---|---|
| VM identifier, provider, disposable scope, console/SSH route | Operator reports no VM created | BLOCKED |
| Image filename, architecture variant, signed checksum, origin and installation method | None supplied | NOT_RUN |
| Exact AlmaLinux release, kernel, x86_64 and CPU ISA | No host exists for collection | NOT_RUN |
| CPU count, memory, block devices, capacity and inodes | Not measured | NOT_RUN |
| Interfaces, routes, hostname, DNS and topology | Not measured | NOT_RUN |
| Repository configuration, signatures and package origins | Not measured | NOT_RUN |
| Python runtime and required collection commands | Not measured on target | NOT_RUN |
| SELinux, firewalld/rules, time synchronization | Not measured | NOT_RUN |
| Existing telephony/database/web packages, services and listeners | Not measured | NOT_RUN |
| Virtualization and potential resource/port conflicts | Not measured | NOT_RUN |
| Snapshot identifier, recovery procedure and restore evidence | No VM/snapshot supplied | NOT_RUN |
| Direct reproduction and independent host review | No collected observations | NOT_RUN |
| Package/build dependency availability and network/TLS suitability | No host repository or connectivity evidence | NOT_RUN |

No actual incompatibility is confirmed. Compiler, development libraries, MariaDB, Apache/PHP, Perl and PJSIP/SRTP dependencies remain unqualified. Required changes cannot be prescribed from absent observations. No stack version or manifest status was changed.

## Collection versus final acceptance

The latest operator instruction supersedes the older preparation report's requirement to prove snapshot restoration **before read-only inventory**. Restoration testing is not a collection prerequisite. Recoverability evidence remains required for full ALM-1 acceptance and subsequent destructive experiments. No task acceptance criteria were overwritten.

The accepted collector still requires nonempty `--host-ref`, `--authorization-ref` and `--snapshot-ref`; it has no documented absent-snapshot mode. If no snapshot exists, do not invent a reference, use `NONE`, omit the argument, or change the collector to bypass provenance. Obtain a real snapshot record before using this accepted interface, or request a separately reviewed provenance-contract change. Snapshot creation/restoration is outside this collection attempt. This interface limitation is distinct from requiring an actual restoration test before collection.

## Prepared transfer artifact

A private local bundle was generated at `/tmp/almadial-m0-01-transfer-4_7pzdxn` (directory 0700; files 0600). It is temporary and must be retained securely or regenerated before use. It contains:

- `almadial-m0-01.tar`: Git archive with prefix `almadial-m0-01/`, containing only collector, test, synthetic fixture, original collection protocol and independent QA report from the accepted commit.
- `SOURCE_COMMIT`, `SOURCE_SHA256SUMS`: exact commit and five source-file digests.
- `SHA256SUMS`: checksums of the archive and the two provenance files.

Archive SHA-256: `2954ec090a4b161f16f709229c68a89c81387ffedbcbc2a91f8c5aa933850a89`.

Collector SHA-256: `b894e1d43500162ee59464f0290853235e6f1a8b39fcfdf34f8c4a8c6f5bb5f0`.

`git get-tar-commit-id` returned the accepted full commit SHA. All five regular archive members matched their committed bytes; the exact member set was checked. `sha256sum -c SHA256SUMS` passed for all three entries. This package avoids reliance on a remote branch: the commit has not been pushed.

To regenerate the archive locally, use the pinned revision and the same ordered path list, then regenerate and verify the provenance/checksum files before transfer:

```bash
git archive --format=tar --prefix=almadial-m0-01/ \
  18b97eea1438e0d1be72276fd3018659a33a636f \
  scripts/collect_host_inventory.py \
  scripts/tests/test_collect_host_inventory.py \
  scripts/tests/fixtures/inventory/almalinux10.json \
  docs/compatibility/m0-01-host-inventory.md \
  docs/reviews/m0-01-inventory-qa.md > almadial-m0-01.tar
```

## Future transfer and execution — NOT_RUN

First obtain a named disposable VM, verify its console identity and SSH host key, record the image/signature evidence, establish exact read-only/evidence-write authorization, and identify a real snapshot. Provisioning or software installation is a separate assignment. Discover the implementing worker's Kaneo identity/membership, self-assign ALM-1, verify ownership, then move to In Progress and read back. Preserve the Blocked label until its underlying conditions are resolved with evidence. A separate QA reviewer must retain implementer ownership.

On the source workstation, set `ALM_SSH_ALIAS` to the operator-approved SSH alias. The following is an unexecuted transfer recipe; no address or credentials are embedded:

```bash
set -eu
: "${ALM_SSH_ALIAS:?Set the approved disposable-host SSH alias}"
ALM_BUNDLE=/tmp/almadial-m0-01-transfer-4_7pzdxn
(cd "$ALM_BUNDLE" && sha256sum -c SHA256SUMS)
# Verify the remote home has no destination with this bundle name before transfer.
scp -rp "$ALM_BUNDLE" "${ALM_SSH_ALIAS}:"
```

Use the ordinary authorized inspection account on the VM, with a protected home directory. Do not disable SSH host-key checks or escalate privileges to fill missing observations. Run these commands there only after target confirmation:

```bash
set -eu
umask 077
cd "$HOME/almadial-m0-01-transfer-4_7pzdxn"
sha256sum -c SHA256SUMS
printf '%s  %s\n' \
  2954ec090a4b161f16f709229c68a89c81387ffedbcbc2a91f8c5aa933850a89 \
  almadial-m0-01.tar | sha256sum -c -
test "$(cat SOURCE_COMMIT)" = 18b97eea1438e0d1be72276fd3018659a33a636f
test ! -e almadial-m0-01 && test ! -L almadial-m0-01
tar --extract --file=almadial-m0-01.tar --keep-old-files --no-same-owner --no-same-permissions
sha256sum -c SOURCE_SHA256SUMS
python3 --version
: "${ALM_HOST_REF:?Set the actual protected VM record reference}"
: "${ALM_AUTHORIZATION_REF:?Set the actual task-scoped authorization reference}"
: "${ALM_SNAPSHOT_REF:?Set the real snapshot record reference}"
ALM_EVIDENCE="$HOME/almadial-m0-01-evidence-$(date -u +%Y%m%dT%H%M%SZ)"
if python3 almadial-m0-01/scripts/collect_host_inventory.py --collect \
  --host-ref "$ALM_HOST_REF" \
  --authorization-ref "$ALM_AUTHORIZATION_REF" \
  --snapshot-ref "$ALM_SNAPSHOT_REF" \
  --output-dir "$ALM_EVIDENCE"; then
  ALM_COLLECT_RC=0
else
  ALM_COLLECT_RC=$?
fi
printf 'Collector exit code: %s\n' "$ALM_COLLECT_RC"
# Continue evidence packaging only for exit 0 or 1; exit 2 needs investigation.
test "$ALM_COLLECT_RC" -eq 0 || test "$ALM_COLLECT_RC" -eq 1
printf '%s\n' "$ALM_COLLECT_RC" > "$ALM_EVIDENCE/collector-exit-code.txt"
cp -n SOURCE_COMMIT SOURCE_SHA256SUMS "$ALM_EVIDENCE/"
(cd "$ALM_EVIDENCE" && sha256sum inventory.json summary.md \
  collector-exit-code.txt SOURCE_COMMIT SOURCE_SHA256SUMS > SHA256SUMS)
```

Run each verification step with stop-on-failure behavior; do not continue after a failed checksum, identity, extraction or exit-code check. Python must meet the documented 3.12+ contract; missing Python/tools require a separate provisioning decision, not installation under inventory authority. Exit 1 is partial evidence, not a health PASS. Exit 0 also grants no host or compatibility acceptance. Preserve original evidence unchanged and use a fresh directory for every run.

## Reproduction and dependency-readiness handoff — NOT_RUN

On the same authorized VM, compare each critical fact to the exact fixed argv/file-read catalog in the accepted collector. Examples include `lscpu --json`, `uname -r`, `getenforce`, `timedatectl show`, `chronyc -n tracking`, and the catalog's explicit-column storage/network/service/RPM queries. Record command, UTC timestamp, exit status and the observation compared. Keep potentially identifying direct outputs under protected operator control; sanitize summaries before repository/MCP export. Hash comparison checks transfer integrity, not image authenticity.

Review OS/image architecture variant and flags, capacity/inodes, resources, routing/DNS/topology, SELinux/firewall/time, installed packages/services and port conflicts individually. Use PASS/FAIL for reproduced requirements, UNAVAILABLE for inaccessible/missing collection facilities, NOT_RUN for unexecuted checks, and NOT_APPLICABLE only with a stated applicability reason. Preserve collector-native states separately; do not rewrite ERROR/UNAVAILABLE/NOT_RUN as PASS. A real service-alias error must retain its original protected output and be reduced to a synthetic regression before a bounded correction is proposed.

Package availability requires a separate metadata evidence record: configured repositories alone do not establish it. If available, use bounded cache-only DNF queries against the candidate package set and record cache age, origins, exact versions and missing/plugin errors. Empty or stale cache does not prove package absence. Network metadata refresh, DNS/TLS checks and their approved endpoints/actions must be identified in the host authorization; no public-address discovery, firewall changes or package installation is implicit. Compiler/toolchain, MariaDB, Apache/PHP, Perl and PJSIP/SRTP library availability stay NOT_RUN until these observations exist. Candidate selection and build dependency qualification continue in **ALM-2 / M0-02**, blocked by M0-01 acceptance.

Before QA Review, assemble inventory JSON/Markdown, their hashes, collector/source identity, timestamps/exit codes, direct comparison evidence, image/repository provenance, CPU ISA assessment, protected authorization mapping and recoverable-snapshot evidence. Mandatory gaps prevent promotion. A different reviewer repeats critical observations, verifies digests and documents differences; timestamps/HMAC identities naturally differ across runs. Full ALM-1 acceptance and the overall M0 gate remain open.

## Assignment outcome and repository checks

Only this blocked-attempt report and the clarification linked from readiness documentation were added/updated; accepted collector, tests, fixtures, source plan and candidate manifest remain unchanged. No local commit or push was made. No live `inventory.json`, `summary.md`, host-evidence hashes, probe exit codes or collection timestamps exist to report.

Executed repository checks: `python3 scripts/validate_scaffold.py` passed (11 required files, four skills); `git diff --check` passed; a focused Python check passed local Markdown links, fence balance, final newlines and `bash -n` for the fenced recipes. No recipe was executed. Accepted collector/test/fixture bytes were compared again to the pinned commit and remained unchanged. The completed unit/fixture QA suite was deliberately not repeated. Bundle checks above are source/transport checks, not host evidence.

MCP read-back at 2026-11-04 23:48 UTC confirmed the new blocked-attempt comment `wpecfd8yo9obml09akqp9ngv` with exact content, preserved the existing QA comment, and found task metadata/description, ownership, native relation and complete task/label listing unchanged. ALM-1 remains `to-do` in Backlog, unassigned, with Blocked and AlmaDial M0 labels. No self-assignment or status transition was justified without an actual target. No implementation, provisioning, source-plan change or stack promotion occurred.
