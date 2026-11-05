# AlmaDial Independent QA — M0-01 AlmaLinux 10.2 Host Baseline

**Reviewer / Session:** Google Antigravity (AGY), Principal Linux Infrastructure QA Engineer / Security & Reliability Verification Architect  
**Implementer:** Codex, Principal Linux Telephony Compatibility Engineer and Infrastructure Automation Architect  
**Kaneo Task:** ALM-1 / M0-01 (`p8urvq3yjtc76rktih9xlhe3`), project AlmaDial (`ghritpte6e40o5ayktgz0vh1`)  
**Task Implementer Assignment:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — preserved unchanged  
**Reviewer Kaneo Identity:** AGY (`ICJQmirJliEFAZN3jBl3ZitHCandT0FG`)  
**Target VM:** `almadial-alma10` (Multipass 1.17.0 / QEMU, AlmaLinux 10.2 x86_64) — read-only inspection  
**Accepted Tooling Baseline:** `18b97eea1438e0d1be72276fd3018659a33a636f`  
**Review Date / Time UTC:** 2026-11-05 (local session: 2026-11-05T23:08:00+08:00)  
**Host Access Authorized:** Yes (read-only inspection on named disposable VM `almadial-alma10`)  
**Production Access Authorized:** No  
**Final Independent Verdict:** **ALMADIAL_M0_01_ACCEPTED**

---

## 1. Executive Summary

This independent quality assurance review evaluates the implementation deliverables and evidence for **ALM-1 / M0-01: Establish the clean AlmaLinux 10 VM and host inventory**.

The implementer (Codex) executed real host-baseline collection against the authorized disposable virtual machine `almadial-alma10`, identified and corrected two runtime parsing defects, completed a bounded nftables kernel ruleset inspection, verified non-destructive snapshot recovery of `almadial-alma10.clean-baseline`, and assembled a complete evidence suite under `~/almadial-lab/evidence/m0-01/firewall-20261105T015153Z/`.

As an independent reviewer, Google Antigravity (AGY) independently:
1. Re-verified the repository source bytes and checksums against the accepted baseline `18b97ee`.
2. Reviewed the collector corrections for RPM exit-code handling and `timedatectl` property queries against systemd and RPM specifications.
3. Executed the complete automated validation suite (scaffold validation, 24 unit tests, `py_compile`, and `git diff --check`), achieving 100% pass rate.
4. Verified all 58 evidence files and the `SHA256SUMS.final` manifest, confirming byte-identical integrity.
5. Performed authorized, read-only direct guest commands on `almadial-alma10`, corroborating that the active kernel packet-filtering ruleset is empty (`sudo -n nft -j list ruleset` exited 0 with 0 tables/chains/rules), `nftables.service` is disabled/inactive, `firewalld` is absent, and SELinux is `Enforcing`.
6. Verified official AlmaLinux 10.2 image provenance (QCOW2 SHA-256 `cbee4726fc0f...`, GPG `VALIDSIG` by AlmaLinux OS 10 release key) and snapshot existence (`clean-baseline` and `snapshot2`).
7. Evaluated the collector exit code 1 as a valid, documented evidence qualification under `.agents/rules/almadial-evidence-integrity.md`, fully satisfying ALM-1 acceptance criteria.

No blocking defects or regressions were detected. All mandatory ALM-1 acceptance criteria have been verified with attributable evidence.

---

## 2. Source Baseline and Implementation History

The accepted repository baseline is commit `18b97eea1438e0d1be72276fd3018659a33a636f` ("Finalize independently reviewed M0 inventory tooling baseline").

Subsequent modifications for ALM-1 comprise the minimal, bounded defect corrections and live host evidence documentation:
- `scripts/collect_host_inventory.py`
- `scripts/tests/test_collect_host_inventory.py`
- `scripts/tests/fixtures/inventory/almalinux10.json`
- `docs/planning/m0-execution-readiness.md`
- `docs/compatibility/m0-01-live-host-baseline.md` (historical blocked attempt)
- `docs/compatibility/m0-01-alma10-live-evidence.md` (measured live evidence report)

### Verified Component Digests

| Component / Path | Reviewed SHA-256 | Actual Working Tree SHA-256 | Match |
|---|---|---|---|
| `scripts/collect_host_inventory.py` | `07a3bd79ca43a59a1ee55fc54adcce9d2964a83edf831d6cf827f7929973f282` | `07a3bd79ca43a59a1ee55fc54adcce9d2964a83edf831d6cf827f7929973f282` | PASS |
| `scripts/tests/test_collect_host_inventory.py` | `5624cb18b0794b600bb59cac74f971875dd5d45a0b672bd9ba4bddeef157675d` | `5624cb18b0794b600bb59cac74f971875dd5d45a0b672bd9ba4bddeef157675d` | PASS |
| `scripts/tests/fixtures/inventory/almalinux10.json` | `d3aede2234a3e10985643e41cc6dd929edabc7a91ff893ba4a626119596e9477` | `d3aede2234a3e10985643e41cc6dd929edabc7a91ff893ba4a626119596e9477` | PASS |
| 3-file git diff patch | `2cb2bfd251875be32248e651eea729f60b6552124e4674a6f2f7eee7e91f8014` | `2cb2bfd251875be32248e651eea729f60b6552124e4674a6f2f7eee7e91f8014` | PASS |

The changes relative to `18b97ee` are strictly confined to the two verified collector defects, their regression tests, and corresponding documentation. No unrelated source or configuration modifications were introduced.

---

## 3. Independent Assessment of Collector Corrections

### Defect A — RPM Exit-Code Handling
- **Failure Mode:** When querying $N$ packages where $k$ packages are absent, `rpm -q` exits with code $k$ (the count of missing operands). The original collector accepted only exit code 1 in `allowed_nonzero`. On a clean AlmaLinux 10 host where 8 queried packages were absent, `rpm` returned exit code 8, causing the collector to flag `ERROR: nonzero_exit` despite all 16 requested packages having complete and recognized output lines.
- **Correction:** The collector now dynamically determines `absent_packages` by checking lines matching `package <name> is not installed` where `<name>` is in the static `PACKAGES` set. It permits a non-zero exit code if and only if `code == len(absent_packages) and 0 < code <= len(PACKAGES)`. Additionally, `parse()` enforces `{x["name"] for x in installed} | set(absent) == set(PACKAGES)`, preventing truncated or partial outputs from succeeding.
- **QA Verification:**
  - Exit code interpretation is verified against RPM standard semantics.
  - Package absence is cleanly separated from command failure; codes not matching `len(absent_packages)` (e.g. 1, 9, 127, -9) produce `status="ERROR", error="nonzero_exit"`.
  - Regression test `test_rpm_multiple_absent_packages_on_almalinux_10` executes and passes.

### Defect B — `timedatectl` Property Query Parsing
- **Failure Mode:** The original command `timedatectl show --property=NTPSynchronized,Timezone,LocalRTC` supplied a single comma-separated argument. On AlmaLinux 10.2 / systemd 256, comma-separated property arguments are not supported and `timedatectl show` returned exit code 0 with empty stdout. `parse()` subsequently failed because required fields were missing.
- **Correction:** In accordance with systemd specification, the command in `COMMANDS["time"]` was updated to pass repeated property options: `("timedatectl", "show", "--property=NTPSynchronized", "--property=Timezone", "--property=LocalRTC")`. In `parse()`, `require_fields` verifies that all three properties are present and non-empty, and `required_enum` validates values against `"yes no"`.
- **QA Verification:**
  - Repeated flags yield expected `Key=Value` lines on systemd.
  - Empty stdout or omitted fields raise `ValueError`, resulting in `status="ERROR", error="parse_error"`.
  - Regression test `test_timedatectl_repeats_property_flags` executes and passes.

---

## 4. Automated Validation Results

The full test and validation suite was executed locally in the Antigravity session:

| Command | Exit Code | Observed Result | Status |
|---|---:|---|---|
| `python3 scripts/validate_scaffold.py` | 0 | PASS: 11 required files, 4 skills | PASS |
| `python3 -m unittest discover -s scripts/tests -v` | 0 | Ran 24 tests in 0.111s; 24 passing, 0 failures, 0 errors | PASS |
| `python3 -m py_compile scripts/collect_host_inventory.py scripts/tests/test_collect_host_inventory.py` | 0 | Clean compilation without warnings | PASS |
| `git diff --check` | 0 | Clean diff, zero whitespace errors | PASS |

---

## 5. Evidence Manifest and Hash Verification

Independent verification of protected evidence directory `~/almadial-lab/evidence/m0-01/firewall-20261105T015153Z/`:

| Evidence File | Expected SHA-256 | Independently Calculated SHA-256 | Verification |
|---|---|---|---|
| `inventory-final/inventory.json` | `cdc2164ac2435834de45b2f3eef8a57cb943532b2385c13547d142f708e469c0` | `cdc2164ac2435834de45b2f3eef8a57cb943532b2385c13547d142f708e469c0` | PASS |
| `inventory-final/summary.md` | `1049c7e75e35bdc9d0a047ffde3a4f36eb966e47b0651737297dabb187f2dd84` | `1049c7e75e35bdc9d0a047ffde3a4f36eb966e47b0651737297dabb187f2dd84` | PASS |
| `ruleset.json` | `ac22c2ca22387cf7169203886a6f07b168f66e798fccd0ab2b90c0c2b109e9bd` | `ac22c2ca22387cf7169203886a6f07b168f66e798fccd0ab2b90c0c2b109e9bd` | PASS |
| `ruleset.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | PASS |
| `SHA256SUMS.final` | `50c7210689493d522e80a0a0934d345d686fbf121f297cffdd92f3c023eb3015` | `50c7210689493d522e80a0a0934d345d686fbf121f297cffdd92f3c023eb3015` | PASS |

Execution of `sha256sum -c SHA256SUMS.final` verified all 58 files with status `OK`.

### Inventory Breakdown
- **Total probes:** 23
- **COLLECTED:** 20
- **UNAVAILABLE:** 3 (`firewalld`, `firewall_zones`, `firewall_rules`)
- **ERROR:** 0
- **NOT_RUN:** 0
- **Collector Exit Code:** 1
- **JSON / Markdown Consistency:** Exact match across all 23 probes and structured facts.

---

## 6. Live VM Firewall and Host State Corroboration

Direct, read-only inspection was executed against `almadial-alma10`:

```bash
# 1. Query kernel nftables ruleset in JSON format
multipass exec almadial-alma10 -- sudo -n nft -j list ruleset
# Output: {"nftables": [{"metainfo": {"version": "1.1.5", "release_name": "Commodore Bullmoose #6", "json_schema_version": 1}}]}
# Exit code: 0

# 2. Query kernel nftables ruleset in standard text format
multipass exec almadial-alma10 -- sudo -n nft list ruleset
# Output: (empty stdout, empty stderr)
# Exit code: 0

# 3. Query rpm packages
multipass exec almadial-alma10 -- rpm -q nftables firewalld
# Output:
# nftables-1.1.5-6.el10_2.x86_64
# package firewalld is not installed
# Exit code: 1

# 4. Check SELinux mode
multipass exec almadial-alma10 -- getenforce
# Output: Enforcing
# Exit code: 0

# 5. Check service active state
multipass exec almadial-alma10 -- systemctl is-active nftables firewalld
# Output:
# inactive
# inactive
# Exit code: 3
```

### Semantic Analysis and Exit-Code Qualification
1. **Firewalld Absence:** The three unavailable probes (`firewalld`, `firewall_zones`, `firewall_rules`) accurately reflect the absence of `firewalld` and `firewall-cmd`. Marking missing utilities as `UNAVAILABLE` adheres strictly to the collector contract.
2. **Active Ruleset Determination:** The direct invocation of `nft -j list ruleset` proves semantically that the Linux kernel contains netfilter metadata only and zero tables, chains, or rules.
3. **Acceptance Contract Alignment:** Under `.agents/rules/almadial-evidence-integrity.md` line 12 ("A documented supplemental command can satisfy a mandatory observation when the acceptance contract allows it; retain original probe state and nonzero exit code"), retaining collector exit code 1 alongside attributable supplemental nftables evidence satisfies the acceptance requirement for measuring kernel packet-filtering posture without inventing synthetic success or forcing an unneeded `firewalld` installation.

---

## 7. Image Provenance and Snapshot Recovery

### Image Authentication
- **Image File:** `AlmaLinux-10-GenericCloud-10.2-20261012.0.x86_64.qcow2`
- **SHA-256 Digest:** `cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`
- **GPG Verification:** Verified against `CHECKSUM.asc` using AlmaLinux OS 10 release key (`EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572`). Output: `GOODSIG DEE5C11CC2A1E572 AlmaLinux OS 10 <packager@almalinux.org>`, `VALIDSIG EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572`.
- **Multipass Association:** `multipass info almadial-alma10` reports `Image hash: cbee4726fc0f`.

### Snapshot and Recovery State
- `multipass list --snapshots` confirms:
  - `clean-baseline`: Parent `--`, Comment: `AlmaDial M0-01 AlmaLinux 10.2 clean baseline`
  - `snapshot2`: Parent `clean-baseline`, Comment: `Before restoring clean-baseline`
- Implementer verification demonstrated non-destructive restore of `clean-baseline`, successful rollback of collection artifacts, post-restore guest accessibility, and unchanged host-side evidence hashes.
- In accordance with read-only inspection safety boundaries, snapshot restoration was not destructively repeated during this QA turn.

---

## 8. Findings and Accepted Limitations

### Findings Matrix
- **Critical / High / Medium Defects:** None.
- **Low / Informational Observations:**
  - Firewalld is absent by default in the official AlmaLinux 10 GenericCloud image. Telephony-specific firewall rules and zone configurations remain a requirement for future installation stages (M1/M3).
  - External network security group, NAT reachability, and upstream carrier network filtering are not inferred from local guest queries.

### Accepted Limitations
1. Telephony stack software (Asterisk 22, VICIdial, WebRTC, ConfBridge, MariaDB, PHP-FPM, Apache) is not installed and compatibility is **NOT_RUN**. Host baseline qualification does not imply telephony stack readiness.
2. The VM is configured as a standalone single-server lab environment on Multipass/QEMU; cluster or multi-server capabilities are out of scope.

---

## 9. Independent QA Verdict

All mandatory ALM-1 acceptance criteria have been verified with attributable, reproducible evidence:
- Real AlmaLinux 10.2 host inventory collected and verified.
- Image provenance cryptographically authenticated.
- Kernel firewall ruleset measured and proven empty via nftables.
- Snapshot recoverability confirmed.
- Collector corrections verified without regression.
- Scaffold and unit test suites passing 24/24.

**Final Verdict Token:**  
**`ALMADIAL_M0_01_ACCEPTED`**
