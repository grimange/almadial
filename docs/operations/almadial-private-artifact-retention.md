# Private artifact retention and release configuration

The source-retention and private-server tools belong to the retained implementation checkpoint
`a9d58baac3bd748f4d10fc7abfc029e6d468bfbc`, on local branch
`alm34/github-bootstrap-operator-acceptance`; they are not present in the current
remote `main`. Use that reviewed checkout for source-retention/server operations.
The current source tree contains the portable bootstrap and release generator.
The operator uses the existing precompiled candidate and needs no development checkout.

This is release-maintainer documentation. The [operator guide](almadial-installation.md)
requires no source-staging work. Private ALM-34 retention/delivery is authorized
for this bounded test; no public third-party source publication is authorized.
The [ALM-32 rights decision](../compliance/m4-04/distribution-rights-decision.md)
and outstanding rights/source obligations still apply. Intact upstream archives
retain their licenses/notices. AlmaDial CLI distribution remains separate.

The accepted `stack.lock.yaml` retains exact versions, original provenance URLs,
archive SHA-256 and VICIdial source-tree identity. The release-reviewed
`manifests/acquisition-origins.json` binds private stores to that exact manifest.
The CLI embeds the source manifest. Guided installation asks for an approved
HTTPS source-store base, defaulting to `ALMADIAL_ARTIFACT_STORE`, and saves the
derived manifest-bound location policy in protected inputs. A changed location
does not authorize a changed version or content. No checksum or TLS trust bypass
is provided; a mirror must supply the exact approved bytes.

Run the offline maintainer tool with an existing approved archive directory:

```bash
python3 scripts/bootstrap/retain_sources.py --archives "$APPROVED_ARCHIVES" --store "$APPROVED_SOURCE_STORE"
```

The tool verifies all five exact archive hashes and every accepted ALM-32 file
record before adding objects. It never fetches or reconstructs source. Existing
objects must match byte-for-byte; unsafe metadata or changed bytes cause refusal.
Objects have private directories and read-only files at
`sha256/ARCHIVE_SHA256/COMPONENT.artifact`. Retain all referenced objects for the
lifetime of their approved release, including older versions after adding new
ones. No expiration or garbage collector is installed. Back up this directory,
its inventory and approved manifests using the lab owner's protected storage
procedure; a release must not rely solely on an ephemeral VM cache or nightly.

## CLI delivery and source-store policy

The bootstrap script, CLI bytes and upstream source objects have separate
identities. `scripts/bootstrap/prepare_release.py` prepares a local release
contract; it never publishes. Use an already approved Linux amd64 binary:

```bash
python3 scripts/bootstrap/prepare_release.py --binary "$APPROVED_CLI" --version "$APPROVED_VERSION" --output "$NEW_RELEASE_DIRECTORY"
```

The output script pins that binary's SHA-256 and version, and `release.json`
records the supplied binary's build identity and the packaging checkout identity.
The packaging checkout is not proof of the supplied binary's source revision.
Review the binary's identity against the approved release before distribution;
this generator does not authorize a new release.
With no `--endpoint`, operators explicitly set `ALMADIAL_BINARY_URL` to an approved
full HTTPS URL. Maintainers may set `--endpoint` to a reviewed default binary base
URL for compatibility with a private deployment. An operator can select another
approved mirror through the environment without editing the script; its trusted
checksum remains fixed. URLs with credentials, query strings or fragments are
rejected. Redirects are not followed. `ALMADIAL_CA_FILE` optionally selects a public
CA bundle for the bootstrap download of the binary; otherwise OS trust applies.

Publish the reviewed script only through an authenticated owner-approved channel.
A checksum inside an attacker-supplied script is not an independent trust anchor.
Never replace a versioned binary or source object under its old identity. Keep
CLI notices/license files with the distribution. Third-party source retention
and delivery require their own applicable authorization.

`ALMADIAL_BINARY_URL` selects first-party CLI delivery only. For this remote-profile
implementation, `ALMADIAL_ARTIFACT_STORE` (or the interactive source-store prompt)
selects a separately authorized HTTPS source base. The installer derives each
content-addressed object URL from the embedded manifest and persists that policy
at `/var/lib/almadial-inputs/acquisition-origins.json`. Resume reads that protected
policy rather than silently adopting a changed environment variable. A policy
relocation needs deliberate owner review and reconciliation of saved inputs.
Earlier binaries may still embed release-bound origins; use the reviewed script
and binary contract for the intended version. Trust for Go source acquisition
comes from OS CA enrollment, independently of bootstrap `ALMADIAL_CA_FILE`.

Keep retained sources, inventory and approved manifests for the full lifetime of
each release. Monitor space, back up outside disposable VMs, and rehash objects
before handoff. Source objects live at
`sha256/ARCHIVE_SHA256/COMPONENT.artifact`. The owner must maintain DNS, endpoint
availability and current trusted certificates; none is an AlmaDial-wide hardcoded
network requirement. Private CA enrollment for the CLI/source store is separate
from browser application trust. Do not open public ingress as a shortcut.

The current bounded private deployment, concrete endpoint and CA details, and
static-server configuration are recorded in the
[maintainer lab record](../development/private-distribution-lab.md).

Acquisition order is verified content-addressed host cache, approved private
stores, then the accepted original HTTPS URL. Every redirect is rejected and TLS
uses OS trust. All candidates must satisfy the same archive SHA-256 and existing
archive/tree rules. Asterisk's detached signature is checked separately by the
accepted installer. Corrupt private cache files are retained as `.rejected-*`
for diagnosis; the next verified download is published without overwriting an
existing object. Exhausted origins stop prerequisite acquisition before planning
or provisioning. SVN reconstruction is deliberately unsupported for this release:
all original bytes were recovered, so no alternate verification mode or repin is
needed. A future reconstruction requires reviewed tree equivalence and an explicit
policy change, never changing ZIP bytes under the old SHA-256.

The accepted five are Asterisk 22.12.0, PJPROJECT 2.17, Net::Telnet 3.05, VICIdial
r4037/schema 1748 and VICIphone commit 24e98fccfa7b7079fff970e02fca5f002ac1f2cc.
Core/extra sound archives are not pinned inputs: the accepted Asterisk build uses
no-sounds configuration. DNF still acquires signed RPM dependencies from the
accepted repositories; package mirroring is outside this correction.
