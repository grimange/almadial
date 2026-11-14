# Remote single-server implementation and controlled qualification

## Result and acceptance boundary

Implementation task ALM-46 is **In Progress**, owned by Codex. The result is
`ALMADIAL_REMOTE_SINGLE_SERVER_P0_CHANGES_REQUIRED`.

Direct trusted HTTPS, same-origin WSS, an installed synthetic ConfBridge route,
meaningful bidirectional browser media and correlated recordings passed on a
controlled off-host network. **Normal VICIdial agent operation remains P0
blocked.** The installed runtime still deliberately disables the updater and
manager listener, and the database has no ordinary campaign. Administrator
authentication passed, but the additional administrator-navigation assertion
did not pass. These results do not qualify an operational dialer.

The owner confirmed that no authorized public host was available and authorized
new controlled development VMs instead. `REMOTE_PUBLIC_HOST_NOT_RUN` applies.
Neither this developer run nor the synthetic phone replaces ALM-34's independent
human operator acceptance or subsequent AGY assessment.

## Repository reconciliation and product contract

Work began on `main`, commit
`f885482478e2fa13fa95a1211c0ab688363b617d`, also the observed GitHub main.
The accepted ALM-33 baseline remains an ancestor. The retained local checkpoint
`a9d58baac3bd748f4d10fc7abfc029e6d468bfbc` contained the guided installation,
durable acquisition, embedded release inputs and associated harness changes
missing from current main. Those task-owned files were reconciled without a
reset or history rewrite. The historical M4-06 report was retained as historical
evidence, not relabeled as remote acceptance. All 386 pre-existing unrelated
untracked files were hash-checked unchanged.

The stale M0-only repository guidance conflicts with the accepted implementation
and the explicit owner direction. The owner's existing decision to use the
accepted implementation and this bounded task governs this work. The candidate
manifest, production restrictions and independent acceptance boundaries remain.

Both profiles use the existing Go installer, immutable plan, one interactive
approval, execution lock, durable journal and actual-state reconciliation:

| Behavior | Isolated lab | Remote single-server development candidate |
|---|---|---|
| Application access | Loopback HTTPS/WSS | Direct HTTPS at the configured FQDN |
| TLS | Generated lab or supplied localhost certificate | Supplied matching certificate/key/trust chain |
| Browser signaling | Existing loopback Asterisk WSS | Apache `/ws` to loopback Asterisk WS |
| Ingress | SSH and essential ICMP | SSH, TCP 443, configured UDP RTP range and essential ICMP |
| Application internals | Loopback | Loopback |
| Phone route | Registration only | Permanent synthetic conference/recording route |
| Agent runtime | Existing deferred policy | Still deferred: blocking implementation gap |

Remote is offered as the default profile selection, with the current agent P0
explicitly disclosed. It is restricted to authorized development use. The first
topology requires one directly attached IPv4; public/private NAT, TURN, carrier
and production qualification are outside this result.

## Provisioning changes

The guided CLI collects profile, FQDN, directly attached IPv4/interface, supplied
TLS inputs and an even-start RTP range. Network input validation rejects
loopback, invalid ranges and conflicting management ports. Saved inputs remain
private and immutable. Existing plan SHA-256 approval is passed directly to the
same executor; the operator does not copy a hash into another command.

For remote, Apache listens on IPv4 port 443 and serves the upstream admin and
agent paths and the existing phone launcher. It preserves upstream
authentication, denies the options path and proxies WebSocket upgrades at
`/ws` to `127.0.0.1:8088/ws`, with forward proxying disabled. The certificate
must match the FQDN. Live checks verify the TLS leaf, authenticated admin page,
agent login form, denied path and WebSocket upgrade. Internal Asterisk HTTP/WSS,
MariaDB, AMI and FastAGI remain loopback.

SELinux remains Enforcing. The TLS adapter verifies exact ownership of the 8088
TCP port label before adding `http_port_t`, rejects conflicting labels/ranges,
enables the required HTTP relay boolean and records rollback state. It does not
disable SELinux or certificate verification. Runtime and permanent firewalld
policies retain management access and allow only 443 and the configured media
range in addition to SSH/essential ICMP. Both policies and actual listeners are
verified; unrelated service ports are not opened.

The remote PJSIP endpoint uses the FQDN's same-origin WSS, ICE, DTLS-SRTP,
fingerprint verification, `actpass`, disabled direct media, the configured media
address and RTP range. The installed synthetic context answers extension 9271,
logs through the existing FastAGI service, records receive/transmit/mixed audio
and joins ConfBridge 9600271. This route is separately owned and verified by
S10. It has no carrier route. Parent-stage verifiers use explicit S08/S10
ownership handoffs rather than accepting arbitrary extra includes/listeners.

Status/Doctor now accept `--plan` to select the intended hostname, network and
listener expectations. Remote diagnostics require the updater/listener to be
active and enabled, so the currently deferred runtime produces **blocking
FAIL**, not a healthy installation summary. Lab retains its existing inactive,
disabled expectation. A focused test verifies both behaviors.

## Distribution and source identity

The existing bootstrap template/generator is reused. `ALMADIAL_BINARY_URL`
selects a location for the same pinned binary; it cannot change version or
SHA-256. `ALMADIAL_CA_FILE` supplies approved download trust without disabling
verification. The guided CLI accepts `ALMADIAL_ARTIFACT_STORE` or prompts for
an approved HTTPS source-store base. It derives content-addressed locations,
retains every source hash and saves the reviewed policy for resume. Changing a
store cannot silently change content. No new downloader or journal was added.

A separate disposable artifact VM served the first-party release and exact
approved source objects. Installation no longer needed the developer workstation
as a runtime file server. This is controlled private distribution, not proof of
internet reachability or authorization to redistribute third-party sources.

The five inputs actually required by the current installer are **VICIdial,
Asterisk, PJPROJECT, Net::Telnet and VICIphone**. All five were acquired
automatically and remain in the verified target cache. Sound archives are not
silently represented as members of this installer input set. VICIdial remains
r4037, version 2.14b0.5, original ZIP SHA-256
`ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058`,
canonical source-tree digest
`04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed`.
No upstream version was repinned and no third-party archive was added to Git.
ALM-32's redistribution restrictions remain in force.

Latest developer CLI identity:

| Field | Value |
|---|---|
| Version | `alm46-remote-dev4` |
| Source identity | `f885482478e2fa13fa95a1211c0ab688363b617d-dirty-alm46` |
| Toolchain/target | Go 1.27.3; CGO disabled; Linux amd64 |
| Binary SHA-256 | `a32bccfab1cec49bb6e6a488180044d5b42465bc288f77b0ff574c79553ff8ee` |
| Source payload SHA-256 | `cf59fff6f13c98867c6f6259f103556890700e924b9299b369a94a3834b4fe51` |
| Rendered bootstrap SHA-256 | `9d1253bec4982d5d0e4d6df78fe17ca7848386b61eed058440725a88e716e6da` |

Two independent trimmed builds produced identical binary bytes. The protected
`release4/release.json` records identity and build provenance. This tested binary
predates the implementation commit and is explicitly a dirty-source build;
it is not claimed to embed the eventual commit SHA.

The immutable GitHub source bootstrap at
<https://raw.githubusercontent.com/grimange/almadial/f885482478e2fa13fa95a1211c0ab688363b617d/scripts/bootstrap/install.sh>
returned HTTP success and matched its committed Git blob, SHA-256
`8e3149cadb2cf8512a1dd87a2ea79c296d7ae59c665c49f84327bf4c056c22a0`.
It pins the **earlier lab CLI**, not this remote build. No push, tag or public
remote-profile release is performed here. Operators require the release owner's
matching reviewed script, binary URL, source-store URL and independently
established trust. The generic README contains no developer LAN address.

## Controlled host qualification

The protected record is under
`~/almadial-lab/evidence/remote-single-server/alm46-20261114/`.
Host identities, concrete addresses, certificate inputs and topology are recorded
there, not made into generic installation defaults. Two newly created VMs are
the only mutation targets: `almadial-alm46-remote-dev` and
`almadial-alm46-artifacts-dev`, each with an `alm46-pristine` snapshot. Accepted
ALM-29/M3 hosts, historical evidence, unrelated VMs and the deleted independent
operator VM were untouched.

The verified AlmaLinux 10.2 cloud image SHA-256 is
`cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca`.
The target used four CPUs, 8 GiB RAM and 80 GiB disk; the separate artifact host
used one CPU, 1 GiB RAM and 10 GiB disk. An ordinary off-host Chromium client
resolved the controlled `.test` FQDN to the target and trusted the test CA. The
browser and server were distinct network endpoints, without SSH tunnels or a
temporary media overlay. This is a controlled LAN, not a public host. Physical
microphone/speaker testing was not run; distinct synthetic PCM inputs traversed
the actual browser/WebRTC/network/Asterisk path.

The target downloaded and executed the reviewed HTTPS bootstrap, which verified
and installed the precompiled CLI. Go and Git were absent. Guided installation
acquired all inputs and executed S01–S10 against exact approved plan
`10442fdd938f54603b33d3344925f542415c33d7626e78f43079d9331e226e4c`.
Documented resume was required. Earlier bounded attempts exposed a service
postcondition timing failure and then the SELinux proxy port issue; their evidence
was preserved before restoring the dedicated target. The proxy issue was fixed.
The transient S04 service postcondition required another resume and remains a
usability limitation, not a claimed seamless install.

| Acceptance item | Actual result |
|---|---|
| Verified precompiled HTTPS bootstrap without target Go/Git | PASS |
| Automatic five-input acquisition | PASS |
| S01–S10 provisioning with documented resume | PASS |
| Trusted direct administrator authentication | PASS |
| Additional administrator navigation assertion | NOT QUALIFIED |
| Agent login form | PASS; form presence only |
| Ordinary authenticated campaign/agent workflow | BLOCKED/P0; campaign list empty and runtime deferred |
| Trusted WSS registration and synthetic browser call | PASS on controlled off-host path |
| Meaningful bidirectional media | PASS; cross-participant signal verification |
| Correlated decoded non-silent recording | PASS |
| Reboot services, listener isolation and firewall persistence | PASS after correcting controlled DNS fixture |
| Same-plan stage reconciliation | PASS: zero actions, `no_op:true` |
| Overall repeated install and Status/Doctor health | FAIL/P0: deferred updater/listener |
| Public host, public DNS, public trusted certificate topology | NOT_RUN |
| Independent human acceptance and AGY QA | NOT_RUN |

In `media2/audio-analysis.json`, A→B envelope correlation was 0.999107 and B→A
0.988231, with distinct fixture-pattern matches 0.998020/0.994071, tone agreement
1.0 and measured signal delays 40/60 ms. Browser evidence records connected
ICE, DTLS/SRTP and packet counters. Bounded UDP packet headers corroborate the
configured media path. Two call unique IDs correlate to six decoded PCM16 WAVs
(receive, transmit and mix per participant), 12.62/13.44 seconds, RMS approximately
0.150–0.232, with completed `call_log` rows. Recordings remain protected.

The initial recording assertion incorrectly expected the dialed number in the
phone-extension field. The corrected assertion verifies `number_dialed=9271`,
the installed phone identity, matching unique ID and completed timestamps; both
the failed fixture run and corrected successful run are retained.

Cloud-init removed the controlled `/etc/hosts` fixture on the first reboot.
A global cloud-init override did not supersede the image's user-data policy.
The fixture was then preserved explicitly in the backed-up cloud-init hosts
template. The final reboot resolved the test FQDN and restored TLS checks.
This fixture correction is not a public-DNS implementation or installer change.
`after-reboot-template-dns-*` contains the final results: read-only assessment
has zero actions and `no_op:true`; repeated install also reports `no_op:true`
but correctly exits 1 at final diagnostics. Status/Doctor exit 10 with updater
and listener FAIL, short-lived certificate WARN and no explicitly selected
diagnostic journal. Services and exact firewall policy persisted, with SELinux
Enforcing and five cache objects. A healthy overall NO_OP result is not claimed.

## Remaining P0 and implementation work

S06's accepted lab policy has `RefuseManualStart` on the updater/listener units,
an exact FastAGI-only process inventory and no active dialing schedules. S05's
AMI account permits only its local integration probe. The installed Asterisk
module/dialplan subset and source adaptations also target synthetic acceptance,
not the full normal agent lifecycle. Merely opening network access cannot correct
these constraints. The S11 temporary cloned-database workers and queue pumping
are test tools, not a normal installed runtime.

A complete remote runtime still needs source-bound, profile-specific daemon and
queue ownership, the required narrowly scoped local AMI actions, necessary
module/dialplan and PHP/Perl compatibility integration, and reconciliation of
legitimate runtime data without weakening foundation checks. It then needs real
administrator navigation, ordinary agent login/workflow, media and recording,
reboot and a healthy repeat-install result. This work is **not completed by this
commit**. No manual service activation, broad AMI permission or hidden test overlay
was used to manufacture an agent acceptance result.

Automatic approval review rejected an additional browser probe because it reused
a HOME override that could expose session material. That probe was stopped; no
indirect retry was used and its navigation check remains unverified. Earlier
successful controlled browser/media evidence is retained separately.

## Regression and handoff

Executed with Go 1.27.3:

- PASS: `go vet ./...`, `go test ./...`, `go test -race ./...`.
- PASS: `go mod verify`, `go mod tidy -diff`.
- PASS: `python3 scripts/validate_scaffold.py`.
- PASS: `python3 -m unittest discover -s scripts/tests -v`, 255 tests.
- PASS: Go formatting, Bash bootstrap/template syntax and Git whitespace checks.
- PASS: focused guided approval, source acquisition/cache, checksum/TLS rejection,
  remote network/TLS/phone validation and lab-preservation fixtures.
- PASS: relevant CLI help and reproducible Linux amd64 builds.
- PASS: relative documentation links and absence of developer addresses or
  private-key material in the generic README.

Lab regression here is automated contract coverage plus the preserved accepted
host evidence; no new full lab installation/media campaign is claimed. The
existing immutable-plan, journal, resume and lock tests remain in the mandatory
suite. No broad security, licensing, carrier or recording-restoration campaign
was repeated.

Protected logs, source/build identities, host/network snapshots, browser/audio
results and final checks are covered by `SHA256SUMS` in the evidence directory.
The local implementation commit is an implementation checkpoint, not acceptance.
The ALM-46 handoff records its commit and ledger identity, retains Codex ownership
and In Progress status, and leaves ALM-34/35 ownership, state and dependencies
unchanged. The next required work is the remote agent-runtime correction above;
independent QA and public-host qualification must not be started as if it passed.
