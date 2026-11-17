# ALM-34 release-specific RPM policy

The current standard target is AlmaLinux **10.3 x86_64**, using policy
`alm34-almalinux-10.3-v1`. Its immutable policy SHA-256 is
`11e4e87c992c0962154e83df6813da63d4fe7b05773940d011478bf616fdb025`.
The machine-readable policy is
[`packages-alma103.json`](../../internal/provision/packages-alma103.json):
45 direct requirements, 351 approved dependency payloads with exact NEVRAs,
repository identities and SHA-256 values, and no approved installed-package
replacements. A particular transaction excludes dependencies already satisfied
by trusted installed packages. On the qualified image after signed bootstrap
utility preparation, the approved transaction contained 347 payloads.

This policy is specific to the verified official GenericCloud image
`AlmaLinux-10-GenericCloud-10.3-20261116.0.x86_64.qcow2`, SHA-256
`845d3a4e41fde973bdbf9763b62a88538c1a1fadc2332a89f6ea01e2c17c78ee`.
The image CHECKSUM signature was verified with AlmaLinux's pinned key fingerprint
`EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572`. Future OS minor releases and
arbitrary preinstalled package combinations are not automatically supported.

## Failure and correction

The historical policy requested the following x86_64 AppStream packages. Epochs
are explicit. All six exact old builds remain in official **10.2 Vault** metadata
and are absent from **10.3 live** metadata; the target's configured `/10/`
mirrorlists follow the current minor release. This was minor-release repository
movement combined with a permanently frozen live-repository policy, rather than
a disabled AppStream repository or a misspelled epoch. No downgrades are needed
on a pristine official 10.3 image.

| Package | Historical 10.2 EVR | Current approved 10.3 EVR |
|---|---|---|
| gcc | 0:14.3.1-4.4.el10.alma.2 | 0:14.4.1-5.el10_3.alma.3 |
| gcc-c++ | 0:14.3.1-4.4.el10.alma.2 | 0:14.4.1-5.el10_3.alma.3 |
| libuuid-devel | 0:2.40.2-18.el10_2.20 | 0:2.40.2-24.el10_3.24 |
| libxml2-devel | 0:2.12.5-10.el10_2.4 | 0:2.12.5-15.el10_3.2 |
| openssl-devel | 1:3.5.8-2.el10_2.alma.1 | 1:3.5.9-1.el10_3.alma.1 |
| perl-DBI | 0:1.643-26.el10_2.7 | 0:1.643-30.el10_3.3 |

The other 38 historical direct requirements remain available at their exact
versions in 10.3. Firewalld is now included in the initial complete transaction
rather than discovered during later firewall provisioning. PHP, MariaDB, Apache
and Perl interpreter direct identities retain their accepted versions. Application
sources, Asterisk 22.12.0, PJPROJECT 2.17, VICIdial r4037 and VICIphone stay pinned
to the accepted source identities. Compiler and development-library changes are
qualified by compilation and the actual installed native runtime/media checks,
not merely package-name matching.

## Repository and transaction contract

The configured official BaseOS, AppStream, CRB and extras definitions must be
enabled, with the original trusted AlmaLinux key, TLS and GPG checking. Resolution
uses these fixed official base URLs in memory without editing repository files:

```text
https://repo.almalinux.org/almalinux/10.3/BaseOS/x86_64/os/
https://repo.almalinux.org/almalinux/10.3/AppStream/x86_64/os/
https://repo.almalinux.org/almalinux/10.3/CRB/x86_64/os/
https://repo.almalinux.org/almalinux/10.3/extras/x86_64/os/
https://dl.fedoraproject.org/pub/epel/10.3/Everything/x86_64/
```

EPEL is a transaction-local source restricted to the six reviewed Perl packages.
It is not enabled as a general system repository. Pinned AlmaLinux and Fedora
public keys verify RPM signatures in an isolated temporary RPM database before
approval. Every downloaded payload must also match its approved SHA-256. An
unsigned package, unreviewed signer, changed checksum or package outside the
reviewed dependency set is rejected. Signature verification is never disabled.

DNF performs dependency solving, downloads, and its native RPM transaction test
before plan approval; only metadata/package caches and temporary verification
state may be written. The test flag is asserted, and installed RPMs plus DNF
transaction history are checked for changes. DNF remains the dependency solver.
The plan binds the OS-specific policy digest and complete resolved transaction.
After approval, DNF re-resolves and must produce the identical transaction before
installation. Unavailable packages, dependencies, repository configuration drift
or changed payloads cause a failure, with no automatic version substitution.

Resume uses the saved selection and historical before-state. Completed
installations inspect actual RPMs, signatures and runtime postconditions for
NO_OP; they do not select a new policy or upgrade the OS. Interrupted package
application still requires its approved repository transaction. Restore the
approved inputs/connectivity and follow the reported reconciliation boundary;
never delete journals, change saved hashes or silently run `dnf upgrade`.

## Historical AlmaLinux 10.2

The original manifest, RPM policy, serialized requests, command adapter and
journal interpretation remain frozen. Saved 10.2 plans cannot select the 10.3
policy. Their snapshot/task authorization representation, where present, remains
strict. New standard plans carry explicit standard authorization and a distinct
10.3 package selection. No placeholder recovery or task assertions are added.

All 44 exact historical direct pins were found in official 10.2 Vault metadata.
An empty-system DNF solve produced a 482-package dependency closure. Official
archive locations use `https://vault.almalinux.org/10.2/` followed by
`BaseOS/x86_64/os/`, `AppStream/x86_64/os/`, `CRB/x86_64/os/` or
`extras/x86_64/os/`; the restricted historical EPEL source is its official 10.2
Everything repository. This is metadata reconstruction evidence. A fresh full
10.2 archived-payload installation and signature qualification were **NOT_RUN**
in this correction. No private RPM mirror is required or proposed.

Do not retarget repository files on a partially installed historical host: the
saved plan binds its observed repository configuration. Preserve the original
host and evidence; a changed archive/repository selection needs an explicit
supported plan, rather than rewriting a historical identity. Fresh archived 10.2
installation is not the current standard workflow. Nothing in the 10.3
qualification retroactively changes historical 10.2 acceptance.

## Qualification boundary

Developer qualification uses a dedicated disposable host with 4 vCPU, 8 GiB RAM,
80 GiB disk, SELinux enforcing, direct controlled IPv4 and existing SSH port 2222.
It uses verified public application sources and one immutable approval, with no
installer snapshot/task prompts. Actual installation, trusted web access, native
agent/media/recording, reboot and healthy repeat results belong to the protected
developer report. Public VPS installation and independent human acceptance are
separate gates. The subsequent owner-authorized publication made
[`alm34-alma103-rc1`](https://github.com/grimange/almadial/releases/tag/alm34-alma103-rc1)
a public prerelease on 2026-11-17 UTC (2026-11-18 Asia/Manila), from commit
`4eb5833b62400ae1c5001087f6dfe81e1e386dcc`. All ten attached assets match the
qualified packet; unauthenticated GitHub bootstrap/CLI downloads passed normal
TLS and checksum verification. Publication does not add public VPS or independent
operator acceptance. Follow the [operator handoff](../operations/alm34-alma103-operator-handoff.md).

The canonical policy SHA-256 above identifies the compact Go policy representation.
The formatted release asset `package-policy.json` has file SHA-256
`b6fb6fd438afe0221aaa6abc53f04b8d15b8f4fb11f371553b67b04a42d48562`;
both identities are checked independently. The unchanged `release.json` records
packet preparation (`publication_status: NOT_PUBLISHED`); live GitHub metadata
records the subsequent publication. This preserves the qualified packet bytes.

The actual 10.3 installation completed using 347 signed RPM payloads, all five
public pinned application sources, and SSH port 2222. Documented saved-plan
resume retained digest
`328db7496e014d0c8f361980c233165698374b2b236113a20b47f056938f0025`.
Three concrete installation assumptions were corrected: staging now retains the
verified public r4037 archive representation instead of requiring the original
nightly ZIP bytes; S06 listener verification uses the selected SSH port; and
diagnostics checks required release-specific RPM identities instead of an old
host's total package count. Historical canonical source identity, port-22 adapter
bytes, absent optional fields and legacy RPM adapter remain unchanged.

An ordinary native agent manual call passed login, conference join, customer
answer, hangup, DC disposition and logout with zero remaining channels. Both
browser connections had connected ICE and DTLS; distinct 12-second audio signals
passed in both directions, with approximately 60 ms measured latency. Native
recording produced a 24.64-second WAV and matching database row. No physical
microphone/speaker or external carrier was tested.

Required services recovered after reboot. The controlled hostname fixture first
needed persistence in cloud-init's hosts template because cloud-init regenerated
`/etc/hosts`; this fixture correction did not repair application services or the
database. The repeated reboot passed Status and Doctor with exit 0 and SELinux
enforcing. Repeat installation returned healthy `no_op: true`, exit 0, with the
same approved plan. Transient FastAGI prefork expansion after the call caused
the existing strict idle-count health check to fail until its pool settled;
the failures and subsequent idle passes are retained rather than suppressed.

The detailed evidence and release identities are retained under
`/home/ra/almadial-lab/evidence/package-policy/20261118-alm34/`.
