# ALM-32 inventory snapshot and regeneration

Baseline: `229c0d40fbe76c07389fae1716180a2dcd4b8bb4`. Qualification date: 2026-11-13 Asia/Manila. `baseline.json` uses a normalized midnight date for deterministic serialization, not the actual collection time.

Regenerate offline from repository root:

```bash
python3 scripts/generate_sbom.py
python3 scripts/generate_sbom.py --check
python3 -m unittest scripts.tests.test_generate_sbom -v
python3 - <<'PY'
import json
from pathlib import Path
import jsonschema
p = Path('docs/compliance/m4-04')
jsonschema.validate(json.loads((p/'almadial.spdx.json').read_bytes()),
                    json.loads((p/'spdx-2.3.schema.json').read_bytes()))
PY
```

Generator: Python 3 standard library, `scripts/generate_sbom.py`, format SPDX 2.3 JSON. Validation uses python-jsonschema 4.19.2 and the unmodified official schema identified in `schema-provenance.json`. No network, root access, host inspection or package installation occurs during regeneration. Inputs have deterministic ordering; the namespace binds their content. Evidence text hashes are verified before output. Tests compare independent generations and the checked-in output and reject changed license evidence. SBOM dataLicense CC0-1.0 applies to document data; it does not relicense described software or quoted upstream licenses.

Updating this historical snapshot requires a new attributable inventory review, not silently rescanning a different host into the same inputs. Local regeneration is supported; independently recollecting the installed inventory requires authorized access to the named qualification host and the five exact source archives. No secrets or raw third-party source trees are required here.

| Input | Meaning / collection |
|---|---|
| baseline.json | Git, host, manifest identity and operator MIT authority attestation |
| rpm-inventory.json | `rpm -qa --qf` identity, source RPM, license/vendor/URL metadata; cached `dnf --cacheonly repoquery --installed` origin where available; selected package-name membership from internal/provision/packages.json |
| go-modules.json | `go list -m -json all`; exact Go modules, sums/replacements; standard library and src/vendor/modules.txt separately in components.json |
| components.json | Reviewed top-level and selected embedded declarations; source URLs and exact evidence paths; NOASSERTION where conclusions are unavailable |
| source-files.json | SHA-256 of each of five installed input archives and every regular file within them, computed with Python tarfile/zipfile and hashlib; no source-tree copies |
| source-headers.json | Bounded first 100 lines for selected embedded JavaScript assets; original file digests are in source-files.json |
| license-evidence.json; licenses/ | Full text license/notice evidence selected from pinned archives, module cache, Go toolchain and official upstream; exact SHA-256; original bytes preserved |
| repository-files.json | Baseline tracked-file digests from `git ls-files`; excludes unrelated untracked operator work and this new compliance artifact set |
| patch-and-build-inventory.json | Three tracked derived patches plus transformation/build input digests; modifications are documented, not called unmodified upstream |
| runtime-observations.json; preservation.json | Read-only service/module/Perl/link inspection and final RPM/source/service comparisons |
| vulnerability-review.json; GO-2026-6859.json | Timestamped scanner limitations, package-level findings and exact advisory data; not a vulnerability clearance |
| inventory-summary.json; validation-results.json | Counts, artifact identity and actually executed validation results |

The SBOM has 772 packages: one qualification-profile container, 40 curated component/notice records, and 731 OS RPMs. Nested notice records are evidence markers, not claims of exact independently built package versions. `filesAnalyzed: false` is intentional: supplemental inventories contain file digests, but a complete SPDX per-file license analysis has not been performed. Package copyright conclusions remain NOASSERTION; notices preserve attributable source declarations.

Full licensing/source-distribution closure is **not complete**. See [implementation findings](../../implementation/m4-04-sbom-licensing-source-notices-and-ownership.md) and [third-party notices](../../THIRD_PARTY_NOTICES.md). Do not use schema validity as release permission.

## Focused gap-resolution artifacts

- `line-js-finding.json`: full pinned notice, SHA-256, deployed page references and UNRESOLVED_RIGHTS decision.
- `component-coverage.json`: deterministic mapping of the existing 40 curated records; unresolved declarations and build closure remain visible. Counts describe this bounded inventory, not universal license clearance.
- `distribution-model.json`: actual delivery categories and finite source/rights decisions.
- `licenses/SIP.js-0.20.1/embedded-notices.txt`: complete leading and embedded MD5 comments, including MIT and WTFPL v2 portions. This source correction requires regeneration of the affected SIP.js SPDX declaration; package count remains 772.

Regenerate/check coverage:

```bash
python3 scripts/prepare_compliance_source.py
python3 scripts/prepare_compliance_source.py --check
python3 -m unittest scripts.tests.test_prepare_compliance_source -v
```

Prepare a private local review candidate using the five original archives in an operator-selected directory; output must not already exist:

```bash
python3 scripts/prepare_compliance_source.py --check \
  --archives /path/to/verified-inputs --output /path/to/new-source-candidate.tar
```

Only reads occur against upstream archives and repository inputs. Output is created exclusively with mode 0600, normalized archive ordering/metadata and an internal SHA256SUMS. Original archive and every regular source-file hash are checked before packaging; archive members are never extracted to the host. Existing upstream symlink entries remain inside the intact original archives and are not followed. Six exact modified files/diffs are reproduced from hash-bound accepted Go transformation literals, preserving upstream headers. The package includes tracked installer Go/embedded/build inputs and operator instructions, license evidence and separately identified operations patches. SOURCE-DELIVERY.json identifies modifications and limitations.

Generate twice to distinct output paths and compare SHA-256 for byte reproducibility. The result is a **local review candidate**, not an authorized release, deployed source-access mechanism, legal clearance, or proof that all corresponding-source duties have been met. Go/module build dependencies remain identified in go.mod/go.sum; rebuilding Go itself or obtaining toolchain source is outside this source candidate. No OS RPM source bundles are inferred from installed inventory. A future binary/image payload still needs exact linked-build and distribution review.

## Distribution-rights decision

[Formal decision and approval register](distribution-rights-decision.md) records the owner-directed technical acceptance amendment and preserves the earlier BLOCKED disposition as history. It retains release-model alternatives, all exception rows, source-delivery conditions and actual downstream dependencies. **Technical SBOM audit: Qualified for independent acceptance. Third-party redistribution clearance: Not yet established. External publication or release of affected third-party components: Not authorized by ALM-32 acceptance.** Technical artifact validation does not grant redistribution rights. The additive canonical Kaneo amendment and comment `j4e71lj3ftsqky84zcm4p7a3` separate these gates without changing downstream dependencies.


The current evidence-text inventory has **39 files**, including the later embedded SIP.js MD5 MIT/WTFPL notice correction; the earlier 38-file count is historical. The 772-package SPDX and existing notices are unchanged by this acceptance-scope amendment. All 40 curated embedded records remain accounted for: 16 without exceptions, 24 with exceptions, including 18 unresolved licensing conclusions. This bounded coverage does not assert exhaustive legal clearance of every source file.
