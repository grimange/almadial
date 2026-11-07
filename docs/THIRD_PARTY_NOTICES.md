# Third-party software and license-review tracker

**Review required before distribution.** This is an inventory aid, not a claim of verified compatibility or legal advice.

| Component | Upstream | License status | Distribution action |
|---|---|---|---|
| VICIdial | Official project SVN | Confirm exact revision's license and notices | Preserve notices and source/offer obligations as applicable |
| Asterisk | Official Asterisk releases | Confirm license and bundled dependencies for selected release | Preserve notices and source obligations as applicable |
| VICIphone | `vicimikec/ViciPhone` | AGPLv3 seen in upstream v3.0 LICENSE; recheck pinned commit | Assess packaging/hosting/distribution and corresponding source requirements |
| AlmaLinux packages | Approved signed repositories | Per-package | Capture package manifest and sources |
| Go YAML parser | `go.yaml.in/yaml/v3` v3.0.4, official YAML organization | MIT for libyaml-derived files; Apache-2.0 for remaining files; reviewed pinned LICENSE/NOTICE | Retain `third_party/go-yaml/LICENSE` and `NOTICE` with distributions containing this dependency |
| YAML dependency tests only | `gopkg.in/check.v1` v0.0.0-20161208181325-20d25e280405 | BSD-2-Clause; module checksum pinned in go.sum | Not linked into AlmaDial; retain upstream license if distributing dependency test sources |
| AlmaDial original code | This repository | MIT proposed, rights-holder TBD | Add actual LICENSE only after verifying ownership |

Do not copy third-party code/assets into this repo until the relevant license conditions and attribution have been reviewed. Distinguish installation-time downloads from any bundled binary distribution.
