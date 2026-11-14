# Source retention and release distribution

This is maintainer documentation. The [operator guide](almadial-installation.md)
describes the standard workflow. Private mirrors are optional; they must not be
required to compensate for a missing official release. Current publication and
external-test gates are in the [distribution correction report](../implementation/internet-distribution-correction.md).

## Standard source acquisition

The embedded `stack.lock.yaml` retains exact versions, original archive SHA-256,
provenance URLs and the accepted VICIdial source-tree digest. The embedded
`acquisition-origins.json` has an explicit empty mirror list. A fresh guided
installation uses public upstreams without prompting for a private source store.
An existing saved installation retains its original reviewed policy on resume.

Acquisition tries a reverified private host cache, optional approved HTTPS mirrors,
and the original pinned HTTPS archive. Four inputs remain available at public
pinned origins: Asterisk 22.12.0, PJPROJECT 2.17, Net::Telnet 3.05 and VICIphone
commit `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`. Their archive bytes must match
exactly. HTTPS source redirects remain rejected; these tested URLs return content
directly. Asterisk's accepted detached signature and keyring remain independently
verified by the existing source loader.

The expired VICIdial nightly ZIP is not the sole origin. When its original exact
bytes are unavailable, the installer exports authoritative
`svn://svn.eflo.net:3690/agc_2-X/trunk@4037` at revision 4037, with externals ignored
and native EOL fixed to LF. It verifies all **2,511** paths and contents against
accepted tree SHA-256
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
The real export was also compared file-by-file with the independently accepted
compliance inventory. A revision label alone is insufficient.

SVN transport is unauthenticated; the full independently accepted tree digest
provides content identity, not a new publisher-authentication claim. This requires
outbound TCP 3690. The bootstrap obtains the `subversion` utility through signed
AlmaLinux repositories when needed. Exports are private, bounded, timed and never
executed before verification. Failure stops prerequisite acquisition before any
provisioning. No Go or Git is needed on the target.

The canonical ZIP representation is explicitly tagged and records its actual
archive SHA-256 separately from the original reference ZIP hash. `VerifyLocal`
continues enforcing original archive bytes; install-input verification additionally
supports this narrowly scoped full-tree policy. Source loaders rehash the actual
representation after reopening and retain their accepted SQL/source identities.
The cache namespace uses the original manifest artifact reference; that directory
name must not be interpreted as the reconstructed ZIP's own checksum. Every cache
hit rechecks its applicable policy. Corrupt inputs are quarantined as `.rejected-*`.
Canonical exports also work offline after successful acquisition.

## Optional private mirrors

Set `ALMADIAL_ARTIFACT_STORE` before the first guided installation to select an
authorized HTTPS store. The installer derives
`sha256/REFERENCE_ARCHIVE_SHA256/COMPONENT.artifact` locations without changing any
source identity. Mirrors currently serve original exact archive bytes. Canonical
SVN representations are host-cache inputs, not silently accepted HTTPS substitutes.
The saved manifest-bound policy is at
`/var/lib/almadial-inputs/acquisition-origins.json`. Relocation requires deliberate
reconciliation rather than changing a resumed installation's environment.

For approved existing archives, retain sources using:

```bash
python3 scripts/bootstrap/retain_sources.py --archives "$APPROVED_ARCHIVES" --store "$APPROVED_SOURCE_STORE"
```

This offline maintainer tool verifies all five original archive hashes and the
accepted provenance inventory. It does not fetch, reconstruct or authorize
redistribution. Preserve licenses/notices, immutable objects, inventories and
approved manifests for the full lifetime of their releases; back up outside
any disposable VM. Private CA enrollment is optional infrastructure configuration,
separate from application/browser trust. Historical laboratory details remain in
the [maintainer lab record](../development/private-distribution-lab.md).

## First-party GitHub release preparation

Use a reproducibly built, committed Linux amd64 CLI with a new identity when its
source differs from the accepted runtime binary. Prepare a versioned default:

```bash
python3 scripts/bootstrap/prepare_release.py \
  --binary "$APPROVED_CLI" --version "$APPROVED_VERSION" \
  --expected-sha256 "$APPROVED_SHA256" \
  --github-repository grimange/almadial --output "$NEW_RELEASE_DIRECTORY"
```

This generates `install.sh` with the versioned GitHub binary URL and fixed hash,
`release.json`, `SHA256SUMS`, the binary and a deterministic CLI notice bundle.
The generator never publishes, pushes or creates tags. Metadata says
`NOT_PUBLISHED` until actual publication is independently verified. Source commit
and packaging commit are recorded separately. Review reproducible-build evidence;
a binary's self-reported commit is not proof by itself.

GitHub binary asset redirects are followed for at most three hops, with TLS
verification and HTTPS-only redirect targets. The fixed SHA-256 gates launch and
replacement. `ALMADIAL_BINARY_URL` is an advanced mirror override; it cannot change
the trusted hash. `ALMADIAL_CA_FILE` only changes bootstrap binary-download trust.
The custom `--endpoint` option remains available for specialized distributions.
An unconfigured template/reference script is not the standard release asset.

Publish only the specifically approved first-party payload after owner approval.
Do not replace bytes under an existing version. No VICIdial, VICIphone or other
upstream application archives are in this release packet. Direct upstream retrieval
avoids AlmaDial public mirroring; it does not erase installed-software licensing
or corresponding-source obligations. The [ALM-32 decision](../compliance/m4-04/distribution-rights-decision.md)
still governs affected application distribution and source delivery. No unrestricted
third-party redistribution permission is inferred from technical SBOM acceptance.
