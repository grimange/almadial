# M4-03 — Security defaults, exposure and credential leakage

## Contract and baseline

Developer qualification for ALM-31 (`mwbiriekq9liuvx3eggyjyko`), against accepted
Git baseline `045534f3f62ac34ace7b136395293ad4c76e9668`. The canonical Kaneo contract
requires attributable checks of source integrity, service privileges, TLS lifecycle,
listener/firewall policy, SELinux, bootstrap accounts and diagnostic exports, with
no unresolved critical/high release blocker, disabled security control or detected
credential leakage. This is Codex developer verification; independent AGY review
and acceptance remain separate.

The task authorizes the disposable local/LAN host `almadial-alm29-clean-d`, machine
`4c7b492940923838a0700adba65ed3eb`, snapshot `alm29-pristine`. The operator separately
authorized `almadial-alm29-clean-e` as the read-only network peer, machine
`ebf868d3a0853b4294cd85e589a12f5c`, also with verified pristine snapshot provenance.
Both use image SHA-256
`cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`.
The accepted M3 VM and historical snapshots were not modified.

Observed target: AlmaLinux 10.2 (Lavender Lion), kernel
`6.12.0-211.64.1.el10_2.x86_64`, 731 RPMs, MariaDB schema 1748, Asterisk 22.12.0,
Apache 2.4.63, PHP 8.3.36 and pinned VICIdial r4037. SELinux is Enforcing.
Installed manifest SHA-256:
`80940fac407ca7ee0041378b2e1e8b4ab4d95aa95749b503d5df7d4f356009fa`.
The accepted installed executable SHA-256 is
`30523badad6a726a11c512ca3e192914a7ad21562f398ca749164e19fffa3d45`.

The bounded threat model covers unauthorized LAN access to internal services,
invalid SIP authentication, disclosure to ordinary diagnostics/support recipients,
unsafe sensitive-file access, and corrupted source input. It does not claim
protection from a compromised root account, public carrier/NAT interoperability,
or a comprehensive penetration test. Local protected configuration legitimately
contains application credentials; those values are compared in memory and are
not exported into this evidence.

## Qualification matrix

| Required case | Result | Decisive evidence |
|---|---|---|
| Internal listener isolation | PASS | Loopback bindings plus independent IPv4/link-local IPv6 TCP probes |
| Management access | PASS | Peer SSH connection and OpenSSH banner in both families |
| Invalid/anonymous SIP registration | PASS | Real trusted WSS REGISTER: challenge/rejection, no contact |
| Authorized synthetic registration | PASS | 401 → 200, matching Asterisk contact, explicit unregister 200 |
| Sensitive permissions / privileges | PASS | Ownership, modes, parent traversal, ACL xattrs, worker identities |
| SELinux / AVCs | PASS | Enforcing, five expected labels verified, no boot AVC matches |
| Bootstrap default credentials | PASS | Protected count-only queries and generated-credential source review |
| Credential leakage | PASS | Six synthetic classes; actual argv/state/log/export comparisons find no values |
| Support bundle | PASS | Four-member allowlist, restrictive modes, both manifests verify |
| TLS positive/negative checks | PASS | Trusted TLS 1.3; wrong hostname and missing trust rejected on both endpoints |
| Certificate replacement/recovery | PASS, reused evidence | Accepted S08 native replacement and fixture failure recovery; no new rotation |
| Firewall persistence | PASS, reused evidence | Accepted S09 reload and ALM-29 reboot evidence; fresh runtime/permanent consistency |
| Source corruption rejection | PASS, local fixtures | Existing database/Asterisk/VICIdial/phone source rejection tests |
| Diagnostics preservation | PASS | 107 files, database, package inventory, rules and service invocations unchanged |
| Fresh installation assessment | PASS | Empty actions and `no_op: true`; normal web login audit timestamps distinguished below |
| Global IPv6 / public carrier / full media retest | NOT_RUN, outside profile | Link-local IPv6 tested; no public deployment or media changes |
| Hosted CI | NOT_RUN | Local qualification only |

## Exposure and authentication

| Service | Actual bind / transport | Independent peer TCP result, IPv4 and link-local IPv6 |
|---|---|---|
| SSH | `0.0.0.0:22`, `[::]:22` | Connected |
| Apache HTTP / HTTPS | `127.0.0.1:80`, `127.0.0.1:443` | Unreachable |
| Asterisk HTTP / WSS | `127.0.0.1:8088`, `127.0.0.1:8089` | Unreachable |
| MariaDB | `127.0.0.1:3306` | Unreachable |
| AMI | `127.0.0.1:5038` | Unreachable |
| FastAGI | `127.0.0.1:4577` | Unreachable |
| SIP TCP / TLS | No listener on 5060 / 5061 | Unreachable |

Twenty bounded TCP connections from clean-e exercised these ten ports in both
families, with two-second deadlines and at most five concurrent probes. All
18 denied paths timed out; SSH succeeded in both families. This is independent
peer evidence, not an inference from localhost. Global IPv6 was unavailable.
UDP was not actively scanned. Asterisk's ephemeral UDP sockets are distinguished
from public SIP transport; no UDP/TCP SIP transport is configured, and the effective
host policy provides no external application/media grant.

The only PJSIP transport is `almadial-s10-wss` (logical WSS bind
`127.0.0.1:5060`, served through Asterisk HTTPS). The native S09 policy
`almadial-s09` has priority `-20000`, ANY → HOST, target DROP, with only management
22/tcp and ICMP/IPv6-ICMP allowances. No application, SIP/RTP, rich-rule or NAT grant
was added. Runtime and permanent observations agree after ignoring the accepted
runtime `active` annotation; offline configuration validation exits 0.

Real WSS SIP tests used the existing protected synthetic identity. An unauthenticated
request received 401; wrong-password and unknown-identity attempts remained 401
and created no contact. The authorized request received 200 following its challenge,
and Asterisk reported the expected `alm_s10_9061` contact. Expires-zero unregister
returned 200, leaving no contact. No SIP Authorization headers or credential values
were retained. This bounded test is not password spraying or a general SIP attack audit.

## Permissions, identities and secret handling

`/etc/almadial` and its secrets directory are root:root 0700; generated scalar
credentials are root:root 0600. Asterisk's directory is root:asterisk 0750 and
credential configuration is 0640. TLS generations are root:asterisk 0750 with
0640 private keys. VICIdial database configuration is root:apache 0640 under
its protected application directory. Thirteen actual stage journals are
root:root 0600. Parent-directory traversal was inspected.

`getfacl` was absent (127); no package was installed. Native extended-attribute
inspection found no POSIX access/default ACL on the inspected sensitive paths,
so no ACL overrides undermine the reported modes. The initial missing-tool
collector failure and its corrected observation are preserved rather than hidden.
MariaDB runs as mysql; Asterisk and FastAGI run as asterisk. Apache/PHP root masters
retain their expected setup privileges and their request workers run as apache.
`matchpathcon -V` verified all five selected sensitive/runtime paths. Boot AVC
search returned the expected no-match status 1, with zero events.

Count-only checks found no tested common default password in administrator,
synthetic or legacy phone records, or runtime AMI/PJSIP secrets. Source review
confirms random generated database/admin/AMI credentials (192 bits) and phone
credentials (128 bits), rather than shared shipped passwords. This is a bounded
default-credential check, not password-strength cracking.

The new `internal/cli/security_qualification_test.go` sends six fake secret classes
(database, SIP, AMI, administrator, private-key marker, session token) through
native failure capture, argv rejection, operator logs, strict journal decoding,
diagnostic projection and all support-bundle members. Native exit status 7 remains
identifiable while stdout/stderr are redacted; secret-bearing argv is rejected
before execution. These local fixtures are explicitly separate from live checks.

On the VM, four protected scalar values were checked without exporting them:
160 process argument sets, 53 ordinary state/log files (3,911,811 bytes), 300 bounded
service log lines, additional progress inputs and the support export had zero
matches. No private-key marker was found. Redaction handles registered secrets;
it is not a universal secret detector. Explicit structured collection and exclusion
of raw configurations/logs provide the support-export boundary.

## TLS and support export

Both HTTPS 443 and WSS 8089 complete TLS 1.3 with hostname and private-lab-CA chain
validation. Incorrect hostname fails with verification code 62 and missing trust
fails with code 20. No `-k` or certificate-warning bypass was used. The unchanged
localhost/127.0.0.1 certificate is valid November 12–26, 2026; its leaf SHA-256 is
`5aa26aa999821e1d0a7dcbbf4d476c9fd285f746f3e7acff7b7d343c547bbe75`.
This is private laboratory trust, not public issuance. Status and Doctor exit 10
for the expected short-lived certificate warning. A journal not selected through
`--journal` is reported NOT_CONFIGURED informationally, not falsely inspected.

S08 current/previous generation ownership and atomic supplied-certificate handling
were reviewed. Historical S08 and final S09 ledgers verify completely (123 and 144
entries). Native accepted replacement and reload results are reused; failed TLS
activation rollback remains fixture-qualified. This audit rotated no certificate,
reloaded no firewall and restarted no service.

The actual bundle `/tmp/alm31-support-20261112T173255Z-alm31.tar.gz` is 3,516 bytes,
mode 0600, SHA-256
`accf73e377d4e52b1afdb9ef1b26f52dcf5dde6809a8b8b61e650433149c01c5`.
Its only members are `report.json`, `manifest.json`, `redaction.json`, and
`SHA256SUMS`, all regular files with mode 0600. Checksums and manifest sizes/digests
verify. No protected credential or private-key marker appears. Collection retains
existing four-second probe/90-second overall bounds, bounded command output and
1 MiB archive/uncompressed limits. Existing exclusive destination, symlink,
collection-path and oversize rejection fixtures pass. No raw DB dump, configuration,
recording, browser session or private key is collected or uploaded.

## Preservation and NO_OP

Across status, Doctor, support collection and authentication checks, 107 managed
files retained bytes/mode/owner/group. Database state, packages, effective nftables
rules, and six service invocation/restart identities were unchanged. Only the
explicit support archive was written to the target filesystem.

Fresh accepted installation planning exits 0 with `actions: []`, `no_op: true`.
It preserves those files, packages, rules and service identities. As already
attributed in ALM-30, its authenticated VICIdial readiness probe updates normal
administrator access audit timestamps. A second assessment compared 531 column,
catalog and grant digests: only `vicidial_users.last_login_date` and `modify_stamp`
changed; 529 checks were unchanged, including all 107 phone columns, credentials,
grants, schema and system settings. This is not a claim of zero database writes
for installation assessment. The diagnostics themselves preserved database state.
No mutating install/resume was needed for this security audit.

FastAGI remains active with six tasks (parent plus five workers). The correct
`almadial-s06-updater.service` and `almadial-s06-listener.service` are disabled and
inactive; an earlier observation used nonexistent abbreviations and is superseded
by `final-runtime.json`. Application cron has zero active jobs. Legacy phones
`callin` and `gs102`, recording storage, certificate generations, S09 policy and
accepted PJSIP configuration are preserved. No outbound dialing or media retest ran.

## Regression and handoff

| Check | Actual result |
|---|---|
| Go 1.27.3 vet / tests / race | PASS; 1,464 passing test/subtest events per suite, two explicit native opt-in skips, no race failure |
| Focused security/source/TLS/export tests | 40 passing events |
| `go mod verify`, `go mod tidy -diff` | Exit 0 |
| Scaffold / Python unittest discovery | PASS / 222 tests PASS |
| Formatting / diff checks | PASS |
| Compiled CLI smoke | 13 invocations PASS (existing `TestCommands` includes its parent as a 14th event) |
| Terminal journeys | 11 PASS |
| Linux amd64 reproducibility | 12 command targets, two byte-identical builds each |
| Hosted CI | NOT_RUN |

No critical/high or other blocking security finding was reproduced. No production
security certification or independent acceptance is implied. Task-owned changes
are this report and the synthetic leakage regression test; provisioning code is
unchanged. Unrelated operator/QA/local-AI files remain untouched and nothing is
committed, pushed, merged, tagged or released.

Protected evidence: `/home/ra/almadial-lab/evidence/m4-03/20261112T173255Z-alm31/`.
It includes host/source/Kaneo identity, bounded observations, helper provenance,
peer results, TLS/SIP outcomes, permission/leak checks, preservation, regressions
and a verified `SHA256SUMS`. Credentials, private keys, raw archive and recordings
are excluded. Handoff retains Codex ownership, requests independent AGY QA, and
leaves downstream work unstarted.
