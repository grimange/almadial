# ALM-34 private artifact retention

These maintainer tools belong to the retained implementation checkpoint
`a9d58baac3bd748f4d10fc7abfc029e6d468bfbc`, on local branch
`alm34/github-bootstrap-operator-acceptance`; they are not present in the current
remote `main`. Use that reviewed checkout for maintainer operations. The operator
uses the existing precompiled candidate and needs no development checkout.

This is lab-maintainer documentation. The [operator guide](almadial-installation.md)
requires no source-staging work. Private ALM-34 retention/delivery is authorized
for this bounded test; no public third-party source publication is authorized.
The [ALM-32 rights decision](../compliance/m4-04/distribution-rights-decision.md)
and outstanding rights/source obligations still apply. Intact upstream archives
retain their licenses/notices. AlmaDial CLI distribution remains separate.

The accepted `stack.lock.yaml` retains exact versions, original provenance URLs,
archive SHA-256 and VICIdial source-tree identity. The release-reviewed
`manifests/acquisition-origins.json` binds private stores to that exact manifest.
The CLI embeds both documents. A changed location does not authorize a changed
version or content. A new lab location requires an explicitly reviewed policy and
CLI rebuild; there is no operator-entered arbitrary mirror or trust bypass.

Run the offline maintainer tool with an existing approved archive directory:

```bash
python3 scripts/bootstrap/retain_sources.py --archives APPROVED_ARCHIVE_DIRECTORY --store /home/ra/almadial-lab/distribution/alm34/sources
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

Current store: `/home/ra/almadial-lab/distribution/alm34/sources`.
Current approved source endpoint: `https://10.160.90.1:8447/sources`.
CLI payloads are separate immutable version directories under `releases/`.
The root `install.sh` selects the reviewed CLI version with a pinned checksum.
Never replace a versioned binary or source object under its old identity.

The bounded static server runs as the local lab user, binds only the approved
private bridge and has no directory listings, uploads or execution. Its root
contains only approved distribution files. CA/private key and evidence are
outside that root. Start it explicitly after confirming that bridge address:

```bash
python3 -u scripts/bootstrap/serve_private.py --root /home/ra/almadial-lab/distribution/alm34 --certificate /home/ra/almadial-lab/evidence/m4-06/bootstrap-20261113T190219Z-alm34/tls/cert.pem --key /home/ra/almadial-lab/evidence/m4-06/bootstrap-20261113T190219Z-alm34/tls/key.pem --bind 10.160.90.1 --port 8447
```

Keep the process and lab network available throughout independent acceptance.
The current private CA is short-lived; renew it and explicitly enroll reviewed
replacement trust before expiry, updating the operator kit fingerprint. This is
a private lab distribution arrangement, not a public availability promise or a
production artifact service. Do not open public ingress. Monitor free space and
rehash retained objects against the inventory before handoff; the CLI independently
rehashes and validates archive content before every use.

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
