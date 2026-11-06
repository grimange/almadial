# M0-08 — clean baseline reproduction

Implementation replay, Codex, 2026-11-07 Asia/Manila. ALM-8
`ftlf2grvt89s3h4nv1yz4x11`. Starting HEAD:
`46d7948b701be274c17f5d7525f1529288fb64a8`.

This is the ordered replay checklist and execution record. The execution matrix below distinguishes verified local behavior from unexecuted
external qualification. The candidate lock
remains candidate. Independent AGY review and maintainer acceptance are separate
gates. No production-readiness claim is made.

## Authority and acceptance boundary

The current ALM-8 assignment authorizes installation only on a second clean
disposable VM, `almadial-alma10-repeat`. The accepted `almadial-alma10` is a
read-only reference. No snapshot restore, populated database clone, installed
runtime copy, external carrier, public call or campaign dialing is permitted.

Kaneo confirms ALM-7 Accepted and its blocking dependency on ALM-8. Codex was
discovered through MCP, assigned, and read back through Ready to In Progress.
ALM-8's canonical verification explicitly includes **reboot**, as well as two
executions of required cases without undocumented intervention. Earlier component
acceptances proved service restart, not automatic OS boot recovery. That missing
installation procedure must be made explicit and verified here.

The operator's subsequent ALM-8 product decision is authoritative: **“Simple,
reliable, single-server VICIdial installer for AlmaLinux 10.”** Direct-public,
1:1 NAT and remote-home-agent tests are deployment-specific, not universal
prerequisites for installer or local stack acceptance. All three remain NOT_RUN.
The current assignment explicitly authorizes the bounded ALM-8, aggregate M0,
ALM-27 and ALM-29 criteria changes recorded, with original/revised text, in
[the scope decision](../planning/alm-8-product-scope-realignment.md). This supersedes
the earlier ALM-7-only amendment's unchanged aggregate scope; it does not rewrite
that historical approval or establish any external test result. Qualify the
topology a deployment actually uses; provide evidence for any explicitly
advertised external capability. Local replay, independent acceptance, stack-lock
promotion and production readiness remain separate decisions.

## Inputs and source authentication

Use `manifests/stack.lock.yaml` together with accepted M0-02 provenance. SHA-256
alone is integrity, not source authentication. Verify signatures in isolated GPG
keyrings; preserve command results and the provenance of immutable cache copies.

| Input | Exact identity / SHA-256 | Authentication |
|---|---|---|
| AlmaLinux cloud image | `AlmaLinux-10-GenericCloud-10.2-20261012.0.x86_64.qcow2`; `cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca` | Accepted official CHECKSUM and detached signature, AlmaLinux 10 key |
| Asterisk | 22.12.0; `f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2` | Official archive/signature; signer `F2FC93DB7587BD1FB49E045A5D984BE337191CE7` |
| Bundled PJPROJECT | 2.17; `04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7` | Asterisk third-party pin and bundled checksum |
| VICIdial | r4037 official `vicidial-trunk-2026-11-05.zip`; `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` | Accepted upstream HTTPS archive and 2,511-file tree `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` |
| VICIphone | 3.0, commit `24e98fccfa7b7079fff970e02fca5f002ac1f2cc`; archive `f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8` | Accepted upstream commit/archive provenance |
| SIP.js | Embedded 0.20.1; `9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b` | Pinned VICIphone asset |
| Net::Telnet | CPAN 3.05; `677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743` | Accepted JROGERS CPAN distribution |

Primary distribution packages must match these NEVRAs; do not silently install a
newer version. DNF must retain GPG checking. Record the complete dependency
transaction, repository identities, RPM signatures and installed inventory.

| Component | Required package identity |
|---|---|
| MariaDB | `mariadb-server-3:10.11.18-1.el10_2.alma.2.x86_64` |
| Apache | `httpd-0:2.4.63-13.el10_2.8.x86_64` |
| PHP | `php-fpm-0:8.3.36-1.el10_2.x86_64` and matching extensions |
| Perl | `perl-4:5.40.2-515.el10_2.x86_64` |
| GCC | `gcc-0:14.3.1-4.4.el10.alma.2.x86_64` |
| SRTP | `libsrtp-devel-0:2.3.0-16.el10.x86_64` |
| Jansson | `jansson-devel-0:2.14-3.el10.x86_64` |

## Ordered replay checklist

Commands below are lab procedures, not a general installer. Every mutation must
first verify the exact repeat-VM hostname. Keep raw outputs, credentials, keys,
browser state and audio outside Git, in a new mode-0700 evidence directory.
Existing output directories and partial schema imports are stop boundaries.
Preserve failed evidence before correcting a step; repeat its affected check.

| Order | Procedure and prerequisites | Verification and expected state | Evidence / recovery |
|---|---|---|---|
| 1 | Verify image SHA and detached CHECKSUM signature; record host resources and Multipass version/list | Exact image, adequate RAM/disk; repeat VM absent | Image/signature records; stop on mismatch |
| 2 | `multipass launch file:///home/ra/almadial-lab/images/AlmaLinux-10-GenericCloud-10.2-20261012.0.x86_64.qcow2 --name almadial-alma10-repeat --cpus 4 --memory 8G --disk 80G --cloud-init /home/ra/almadial-lab/cloud-init.yaml` | Exit 0; OS 10.2 x86_64, clean package inventory, SELinux Enforcing | Provision log; never replace an existing instance |
| 3 | Transfer/run `scripts/collect_host_inventory.py --collect` with explicit host, authorization and snapshot references; supplement raw RPM, routes, nftables, clock and service inventory | All mandatory observations attributable to repeat VM; missing probes classified honestly | Clean inventory; do not confuse absent firewall tooling with empty rules |
| 4 | Stop repeat VM; `multipass snapshot almadial-alma10-repeat --name clean-baseline`; inspect snapshot; start repeat VM | Read-back `almadial-alma10-repeat.clean-baseline`; no restore | Snapshot and start records |
| 5 | Verify cached source inputs above before transfer/extraction; reject symlinks/traversal; keep originals immutable | Version files, tree ledger, archive and signature checks pass | Source ledger; stop on unknown bytes |
| 6 | Install exact build/runtime dependencies from accepted BaseOS/AppStream/CRB; use reviewed, signed isolated EPEL packages only for required Perl modules | RPM versions/signatures match; no repository substitution or global security weakening | DNF transaction and full RPM inventory |
| 7 | Unprivileged Asterisk configure/build, then bounded root `make install`; no `make samples` or whole VICIdial installer | Native 22.12.0 / bundled 2.17; required modules present | Configure/build/install logs; preserve partial build on failure |
| 8 | Create dedicated Asterisk account, directories, base configuration and systemd unit; issue fresh isolated lab CA/server certificate for repeat hostname | Config root:asterisk 0640, writable runtime dirs asterisk, key protected, validated WSS | Config ledger, certificate verification; no copied runtime state |
| 9 | Configure MariaDB `skip-networking` in root:mysql 0640 file; start; create a fresh `almadial_m005` utf8mb3 database | Socket-only, no 3306, original strict mode | Listener/SQL-mode records; halt on any existing schema |
| 10 | Derive schema with `scripts/prepare_vicidial_r4037.py`; import derived schema then pinned `first_server_install.sql` | Schema 1748; 345 tables; 1,011 index-column records; expected warnings retained | Import logs; never drop/reimport partial state automatically |
| 11 | Create least-privilege local DB accounts; deploy immutable webroot; set `webroot_writable=0`, `use_slave_server=0`, actual upstream first-login initialization | Apache/PHP-FPM, DB connection, administration and authentication work | PHP lint/HTTP/admin results; secrets through protected files/stdin |
| 12 | Install pinned Perl modules in traversable versioned library paths; derive four process checks with existing tool | Non-root DBI/DBD::mysql/Net::Telnet/Net::Server/AGI load; original and derived syntax checks | Module inventory, derivation ledger; upstream unchanged |
| 13 | Seed local server/phone/template inputs; derive and execute actual r4037 generation using `scripts/lab/m0_05_generator.py` | Nine staged outputs; endpoints 9051/9052/9053; 300 conference slots | Generator inputs/output ledger; no whole keepalive/autodial suite |
| 14 | Publish generated includes under ALM-5 ownership; repeat unchanged generation | Deterministic bytes or explained equivalence; valid AOR/auth/dialplan/ConfBridge; no collisions/legacy MeetMe exposure | Two generation ledgers and Asterisk effective checks |
| 15 | Native endpoint 9001 Echo with accepted diagnostic client and paired capture | Trusted HTTPS/WSS; real microphone/remote tracks; connected ICE/DTLS-SRTP; existing signal thresholds | Fresh browser/audio records from repeat VM |
| 16 | Derive/start actual FastAGI with `scripts/lab/m0_06_fastagi.py`; use existing ALM-6 runtime/browser/media helpers | Two-party media, monitor/barge, AMI authorization/redirect/hangup, actual FastAGI, recording correlation and closed descriptors | Fresh per-call IDs, AMI events, database rows, decoded WAV and analysis |
| 17 | Derive/deploy VICIphone and r4037 overlays with `scripts/lab/m0_07_assets.py`; configure real test agent/manual campaign | Exact local assets, embedded SIP.js, POST credential protection, global/PHP/Perl strict mode | Source/deployment manifests, HTTP and SQL checks |
| 18 | Start upstream derived AST_update/listener and bounded queue sender before real browser agent login | Fresh heartbeat, no initialization warnings; correct phone/session/conference association | Real agent UI and DB/CLI state |
| 19 | Actual agent workflow with embedded 9051 and separate real 9052 VICIphone | Two-way audio, park/resume, blind local transfer, hangup, SIP INFO DTMF, calendar, logout revocation and fresh login | UI events + selected-channel/DB correlation; mute is not hold |
| 20 | Unexpected idle/active WSS interruption; restore reachability and click Register | Visible disconnect, supported manual reconnect/register; explicit old-call hangup; new call succeeds, one contact each | WS/client/contact/channel state; no automatic continuity claim |
| 21 | Verify invalid credentials/certificates, log/URL credential protection, security and cleanup | Enforcing; socket-only DB; loopback AMI/FastAGI; zero calls/channels/contacts/occupied slots/pending actions | Sanitized security summary and protected raw audit |
| 22 | After clean teardown restart Asterisk once; health-poll; repeat bounded fresh media call | Modules/endpoints/WSS/AMI/FastAGI recover; media and cleanup pass | Restart records and second call evidence |
| 23 | Inspect/enable only required service dependency chain; controlled full repeat-VM reboot; repeat mandatory checks | No undocumented startup action; dependencies healthy, new media/agent lifecycle succeeds | Boot ID, enablement/order, reboot and fresh post-boot evidence |
| 24 | Read-only compare reference and repeat inventories | Exact source/version/derivation contracts; explain host/secret/timestamp and build-byte variations | Structured cross-VM comparison; do not require identical secrets/IPs |
| 25 | Run repository checks; seal evidence; prepare lock proposal for separate AGY | No sensitive Git artifacts, candidate unchanged, all claims attributable | Validation exit codes/counts, SHA ledger and QA handoff |

Asterisk build command, inside the newly extracted source as the unprivileged
guest user (external cache contains the verified PJPROJECT archive):

```bash
./configure --with-pjproject-bundled --with-externals-cache=/home/ubuntu/alm3/cache --without-dahdi
make menuselect.makeopts
menuselect/menuselect --disable-category MENUSELECT_CORE_SOUNDS --disable-category MENUSELECT_EXTRA_SOUNDS --disable-category MENUSELECT_MOH --disable BUILD_NATIVE --enable chan_pjsip --enable res_pjsip --enable res_pjsip_transport_websocket --enable res_http_websocket --enable res_srtp --enable res_crypto --enable app_confbridge --enable app_echo --enable res_pjsip_config_wizard menuselect.makeopts
make -j4
sudo make install
```

Each command must exit 0 and its effective postcondition must pass. Never infer
successful Asterisk reload from exit code alone: use `module reload res_pjsip.so`,
`module reload res_pjsip_config_wizard.so`, `dialplan reload`, and explicit module
and endpoint inspection. Asterisk 22 has no `pjsip reload` command.

## Runtime evidence status

The second VM was provisioned from the verified clean image, compiled, initialized
from SQL source, and exercised with fresh browser sessions and media. No configured
snapshot or populated reference database was copied. The following sections record
the actual results and the corrections discovered during this replay.

**Implementation disposition: ready for independent QA under the authorized
local/LAN contract.** The former external-scope blocker is resolved by the explicit
operator decision, not by new external test execution. Second-clean-installation
evidence and focused correction tests support the implementation submission;
independent AGY review, maintainer M0 acceptance and lock promotion are pending.
M0 is not finally closed and the manifest remains candidate. See plan §§9.3, 10.3
and 14 and the linked scope decision for the exact current contract.

Protected execution directory:
`~/almadial-lab/evidence/m0-08/20261106T192640Z/`.

## Executable installation procedure

The bounded helper is `scripts/lab/m0_08_replay.py`, not the final Go installer.
It checks root and the literal hostname `almadial-alma10-repeat` before input
reads or effects. Its `run` records exact argv, UTC, output and exit code under
`/root/almadial-m008/`; SQL containing secrets is stdin and is not recorded.
Existing command records/output directories are stop boundaries. Do not rerun a
partially completed phase: inspect its record and complete only the documented
failed operation, preserving the original record and backup. No phase restores
snapshots or drops/reimports an existing schema.

Use a private host directory `ALM8_INPUT_DIR` containing these logical filenames.
They are immutable source inputs, never installed runtime copies:

- `asterisk-22.12.0.tar.gz`, `pjproject-2.17.tar.bz2`, `vicidial-r4037.zip`,
  `viciphone.tar.gz`, `Net-Telnet-3.05.tar.gz`, with the hashes above.
- `perl-Switch.rpm`, `perl-Crypt-Eksblowfish.rpm`, `RPM-GPG-KEY-EPEL-10`, from the
  accepted M0 dependency cache. Verify their recorded hashes and RPM signatures.
  Exact dependency NEVRAs and the isolated signed EPEL transaction are in
  `packages()`; `package-specs.json` is copied from
  `scripts/lab/m0_08_package_specs.json`.
- Repository scripts `prepare_vicidial_r4037.py`, `m0_03_signal.py`,
  `m0_03_webrtc_client.html`, `m0_05_generator.py`, `m0_05_capture.js`,
  `m0_06_fastagi.py`, `m0_06_runtime.py`, `m0_06_media.py`, `m0_06_browser.js`,
  `m0_06_webrtc_client.html`, `m0_07_assets.py`, `m0_07_browser.js`,
  `m0_08_replay.py` (lab scripts are under `scripts/lab/`).

For this execution the final self-contained input bundle is `inputs-r12.tar`,
SHA-256 `9c20ce6a738dd2818127d2ca4fb40410e7646395ebbc83d9324b3caabe420174`,
under the protected evidence directory. Its `inputs.json` lists every input's
SHA-256. Earlier bundles and every failed attempt remain preserved. The VICIdial
archive is the exact accepted immutable copy, not a newly downloaded mutable
nightly URL. Accepted HTTPS-origin/tree matching is its authentication boundary;
there is no invented detached signature or SVN attestation.

After source/signature verification and before the VM installation, assemble a
fresh bundle using Python `tarfile` with plain filename members. Generate the
manifest with the following command; it refuses an existing manifest:

```bash
python3 - "$ALM8_INPUT_DIR" <<'PY'
import hashlib,json,sys
from pathlib import Path
root=Path(sys.argv[1])
files={p.name:hashlib.sha256(p.read_bytes()).hexdigest()
       for p in sorted(root.iterdir()) if p.is_file() and p.name!='inputs.json'}
with (root/'inputs.json').open('x') as f: json.dump(files,f,indent=2)
PY
```

To assemble those verified flat inputs into a new archive, choose an absent
output path outside the input directory and run:

```bash
ALM8_BUNDLE=/tmp/alm8-verified-inputs.tar
python3 - "$ALM8_INPUT_DIR" "$ALM8_BUNDLE" <<'PY'
import sys,tarfile
from pathlib import Path
root=Path(sys.argv[1]); output=Path(sys.argv[2])
members=sorted(root.iterdir())
assert all(p.is_file() and not p.is_symlink() for p in members)
assert (root/'inputs.json').is_file()
with tarfile.open(output,'x') as archive:
    for p in members: archive.add(p,arcname=p.name,recursive=False)
PY
```

Alternatively use the retained, already verified `inputs-r12.tar` directly.
For that exact archive, the following host-side check must pass before transfer:

```bash
ALM8_BUNDLE="$HOME/almadial-lab/evidence/m0-08/20261106T192640Z/inputs-r12.tar"
python3 - "$ALM8_BUNDLE" <<'PY'
import hashlib,sys
from pathlib import Path
assert hashlib.sha256(Path(sys.argv[1]).read_bytes()).hexdigest() == '9c20ce6a738dd2818127d2ca4fb40410e7646395ebbc83d9324b3caabe420174'
PY
```

Transfer the authenticated archive to the authorized **clean** repeat VM. The
following commands stop if the target input directory exists; they are recipe
instructions, not permission to overwrite an existing completed installation:

```bash
multipass transfer "$ALM8_BUNDLE" almadial-alma10-repeat:/home/ubuntu/alm8-inputs.tar
multipass exec almadial-alma10-repeat -- python3 -c '
import socket,tarfile
from pathlib import Path
assert socket.gethostname()=="almadial-alma10-repeat"
destination=Path("/home/ubuntu/alm8-inputs")
assert not destination.exists()
with tarfile.open("/home/ubuntu/alm8-inputs.tar") as archive:
    members=archive.getmembers()
    names=[m.name for m in members]
    assert len(names)==len(set(names))
    assert all(m.isfile() and Path(m.name).name==m.name and m.name not in (".","..") for m in members)
    destination.mkdir(mode=0o700)
    archive.extractall(destination,filter="data")
'
```

The replay helper validates all manifest entries, rejects path escape/symlinks,
and requires the exact 2,511-file VICIdial tree hash. No unchecked installer or
CPAN build script is executed as root. The verified Net::Telnet module and pinned
upstream Asterisk AGI modules are installed in traversable versioned site paths.

Run these phases **once, in this order**, each with exit 0 and the table's
postconditions before proceeding:

```bash
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py packages
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py sources
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py build
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py database
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py base
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py generator
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py telephony
multipass exec almadial-alma10-repeat -- sudo python3 /home/ubuntu/alm8-inputs/m0_08_replay.py application
```

`telephony()` calls `telephony-start()` itself. Do not run the latter again on a
successful installation. It is an explicit resume boundary for the originally
observed missing-AGI prerequisite, retained in failed-run evidence.

The phases contain the complete base configuration, fresh credentials, local
fixture SQL, ownership, TLS, includes and systemd units. Important effective
values: `astmoddir=/usr/lib/asterisk/modules`; keys directory root:asterisk 0750
and files 0640; generated global definitions included from `[globals]`; AMI secret
20 hex characters to fit `servers.ASTmgrSECRET`; ENUM fixture values quoted as
strings; manual campaign dial prefix `X` (upstream no-prefix sentinel), local
number `99052`; VICIphone options separated by `--`. Endpoint 9001 remains the
native Echo fixture. 9051/9052 are generated phones and 9053 is the monitor/barge
fixture. No provider/trunk is configured.

For clean inventory, transfer the repository collector and run on the repeat VM:

```bash
python3 /home/ubuntu/collect_host_inventory.py --collect \
  --host-ref almadial-alma10-repeat \
  --authorization-ref operator-ALM8-second-clean-replay \
  --snapshot-ref almadial-alma10-repeat.clean-baseline \
  --output-dir /home/ubuntu/alm8-clean-inventory
```

The original clean image lacks a complete firewall-tool inventory. The collector's
exit 1 is retained as an incomplete observation, not relabeled PASS. Installing
signed `nftables-1.1.5-6.el10_2` permitted the missing read-only probe; the recorded
ruleset had no chains/rules. No firewall or SELinux bypass was introduced.

### Administration first-login prerequisite

The immutable webroot must contain `vicidial/old_clear`, root:apache 0640, before
Apache executes `admin.php`. The helper now creates it. Upstream lines 143–151
otherwise attempt cleanup/marker creation before checking `webroot_writable`,
causing PHP `fwrite(false)` TypeError. No writable document-root workaround is
used. `webroot_writable='0'` and `use_slave_server=0` remain effective.

Complete actual upstream first-login setup through trusted
`https://almadial-alma10.test:8443/vicidial/admin.php`, using the protected initial
admin identity from `/root/almadial-m008/credentials.json`. Accept the upstream
welcome/copyright step, then submit POST `ADD=999996`, `stage=SUBMIT`, `user=6666`,
a newly generated alphanumeric `pass` different from the old one,
`default_phone_registration_password`, `default_phone_login_password`,
`default_server_password` (fresh alphanumeric values), `default_local_gmt=8.00`,
`default_voicemail_timezone=eastern`. Do not put HTTP Basic credentials in URLs.
Expected HTTP 200 contains “Passwords have been updated”; `first_login_trigger`
becomes `N`. Save the new administrator secret only in a protected file.

This upstream operation updates **all phone and server default secrets**. In this
replay administration was tested after telephony. Preserve the already-created
isolated fixture values before setup and restore only those values afterwards;
otherwise SIP registration and generated-file credentials diverge. Run the
following guest Python as root, before setup, to create the protected restoration
SQL (SQL stdout stays in the file, never in terminal/evidence exports):

```python
import os, subprocess
from pathlib import Path
os.umask(0o077)
q = """SELECT CONCAT('UPDATE phones SET conf_secret=',QUOTE(conf_secret),
 ',pass=',QUOTE(pass),',local_gmt=',QUOTE(local_gmt),
 ',voicemail_timezone=',QUOTE(voicemail_timezone),' WHERE extension=',
 QUOTE(extension),' AND server_ip=',QUOTE(server_ip),';') FROM phones;
SELECT CONCAT('UPDATE servers SET conf_secret=',QUOTE(conf_secret),
 ',rebuild_conf_files=',QUOTE(rebuild_conf_files),' WHERE server_ip=',
 QUOTE(server_ip),';') FROM servers;"""
r=subprocess.run(['mariadb','-NB','almadial_m005'],input=q,text=True,
                 capture_output=True,check=True,timeout=30)
with Path('/root/almadial-m008/admin-fixture-restore.sql').open('x') as f:
    f.write(r.stdout)
```

After successful setup, run `mariadb almadial_m005` with that protected file as
stdin; require exit 0. Do not restore the administrator password or
`first_login_trigger`. Verify fresh SIP login and the two administration create/
read operations. This is the recorded bounded lab procedure; moving first-login
setup ahead of fixture creation is a possible future simplification, not a
procedure claimed to have been executed here.

### Browser, queue and operational replay

Use an isolated browser home/NSS store with **only the fresh repeat-VM CA** added
via `certutil -A -d sql:<isolated-home>/.pki/nssdb -n ALM8-repeat -t C,, -i ca.crt`.
Chromium must retain `ignoreHTTPSErrors:false`. Map `almadial-alma10.test` to the
current repeat-VM address using `--host-resolver-rules`; the certificate SAN is
that DNS name. A separate untrusted browser must reject the certificate.

Generate fresh signal A/B/C using `m0_03_signal.py`/`m0_06_media.py`; fixtures are
synthetic microphones, while transport and recording are actual live WebRTC.
Use the existing ALM-6 `m0_06_browser.js` asynchronous function and `m0_06_runtime.py RUN --full
--authorized-host almadial-alma10-repeat` observer for each unique run. The
observer claims only its own conference, verifies call/recording identifiers,
and releases its own resources. Do not invoke its default original-VM target.
Analyze two-party captures with `m0_06_media.py`; compare native Echo's **paired
local and remote captures from the same run** with `m0_03_signal.py compare`.
Monitor audio must contain both participant patterns; monitor microphone must
remain isolated until the actual AMI barge transition. Recording metadata is a
lab-owned row correlated with real AMI recording start/stop and channel IDs; this
is not a claim that VICIdial's recording button was tested.

Invoke the browser function through Playwright MCP with a controller page whose
URL query contains `evidence=<private-host-evidence>`,
`alm3Evidence=<private-evidence-with-browser-home>`, `repo=<absolute-repository>`,
`vmAddress=<repeat-VM-IPv4>` and `run=<unique-run>`. These are paths and synthetic
run identifiers, not secrets. The function reads `phone-9051`, `phone-9052` and
`phone-9053` from the protected evidence directory using file inputs. Start the
guest runtime observer concurrently, using that same run identifier; follow the
accepted [ALM-6 procedure](m0-06-telephony-runtime.md) for capture/event timing.
Unlike the ALM-7 start/capture/finish interface, this ALM-6 function executes the
two-party, monitor and barge stages in one invocation.

For VICIphone use `m0_07_browser.js` with explicit current IP, evidence directory,
unique run, and `embeddedAgent:true` after real agent/phone login. Load protected
agent and phone fields through file inputs/POST, never URL queries. After login,
manual dialing, park/resume, transfer and logout, dispatch the exact pending local
queue actions, **one action per direct invocation**:

```bash
multipass exec almadial-alma10-repeat -- sudo -u asterisk perl /opt/almadial-m007/AST_manager_send.pl --sendonlyone
```

Do not wrap this command in a parent shell whose command text contains the script
name: upstream process scanning can mistake the parent for a second sender. Do
not add `--conf` (unsupported), run keepalive, or start the campaign suite. The
one-shot sender's “PROCESS KILLED MANUALLY” exit diagnostic is normal. Its dead
listener suggestion does not launch keepalive: those launch lines are commented
in pinned upstream source. Inspect actual service state instead.

In the actual agent UI choose campaign M007, manual dial local `99052`, accept on
9052, select PARK then resume, and use XFER with local destination `9600001` and
blind transfer. Correlate 9052's selected channel moving conferences while agent
9051 remains in 9600000; inspect `parked_channels`, `vicidial_live_agents` and
`vicidial_manager`. Send supported keypad DTMF through actual VICIphone (SIP INFO).
After XFER disposition and LOGOUT, replay a formerly authorized protected request:
HTTP 200 with `Invalid session` is rejection, not authorization success. Retained
history rows alone are not defects. Drain remaining bounded logout actions;
require zero live agents, park rows, conference reservations, pending queue,
Asterisk channels/calls/conferences and contacts.

Interrupt the actual WSS TCP connection unexpectedly in idle and active conditions.
Capture WebSocket/transport/registerer/UI state before and after. The accepted
contract is visible disconnection plus manual Register, explicit old-call hangup,
and a successful new call with one contact; automatic same-call continuation is
not claimed. Active RTP can survive signaling loss. Do not disable TLS validation
or invoke a normal SIP logout as a substitute for unexpected loss.

Identify the exact browser TCP flow with `ss -tnp` on the repeat VM, then kill
only that established flow. The active-call experiment used the command below;
the peer port is an observed input, not a reusable constant:

```bash
multipass exec almadial-alma10-repeat -- sudo ss -K src 10.160.90.147 sport = :8089 dst 10.160.90.1 dport = :53636
```

Require exactly the intended socket match and preserve the observation before
execution. Restore connectivity without changing certificate policy, then use
the actual VICIphone Register control.

Before restart/reboot, require clean calls. Restart only Asterisk once, then verify
modules/endpoints/listeners/AMI/FastAGI and a fresh Echo call. For each full reboot,
record `/proc/sys/kernel/random/boot_id`, use `multipass restart
almadial-alma10-repeat`, health-poll required services, verify a changed boot ID
and fresh browser call. Both recorded reboots recovered all seven enabled services
without manual starts. Browser credential injection, synthetic fixture generation,
and media analyses reuse accepted tools rather than a competing phone harness.

## Executed local result matrix

All live observations below name `almadial-alma10-repeat`; first-VM recordings
were not used as second-VM runtime proof. Signal analyses are offline measurements
of fresh live captures, joined with browser TLS/ICE and Asterisk channel evidence.
Their own `runtime_state: NOT_RUN` correctly means the analyzer does not execute
a call. Synthetic microphone input traversed actual browser WebRTC; physical
microphone/speaker and human speech listening remain NOT_RUN.

| Requirement | Actual result | Attributable protected evidence |
|---|---|---|
| Clean OS and recovery point | PASS: AlmaLinux 10.2 x86_64, kernel `6.12.0-211.64.1.el10_2.x86_64`, 4 vCPU, 8 GiB, 80 GiB; verified clean image and snapshot; no restore | Provisioning, image authentication, clean inventory and snapshot read-back records |
| Sources and native build | PASS: Asterisk 22.12.0, bundled PJPROJECT 2.17, native GCC, 64 loaded modules; original r4037 and VICIphone inputs verified | Input manifest, guest build records, two inventory records and `cross-vm-comparison.json` |
| Database and language runtime | PASS: new schema 1748, 345 tables, 1,011 index-column entries; socket-only MariaDB 10.11.18, strict mode preserved; PHP 8.3.36, Perl 5.40.2 and non-root modules | Guest SQL import/module records; comparison and final health records |
| Real administration | PASS twice: authenticated create/read of isolated users `m008admin1` and `m008admin2`; anonymous requests rejected 401 twice; actual upstream first-login initialization completed | `admin-initialization.json`, administration create/read and anonymous-rejection records |
| Configuration generation | PASS: actual derived upstream generator, nine staged outputs; unchanged-input repeat byte-identical; four published generated includes; 300 conference slots | Guest generator ledgers and effective endpoint/dialplan/ConfBridge observations |
| Native WebRTC Echo | PASS before restart, after Asterisk restart, and after second full reboot; trusted HTTPS/WSS, registration, ICE/DTLS-SRTP and paired local/remote capture | `paired-echo-analysis.json`, restart/reboot browser records and fresh WAVs |
| ALM-6 conference/AMI/FastAGI | PASS twice: `repeat-telephony-1` and `repeat-telephony-2` (after first reboot); two-party media, monitor isolation, AMI barge/redirect/hangup and actual Asterisk-invoked derived FastAGI | `telephony-runtime1.json`, `telephony-runtime2.json`, browser traces, `telephony1.tar`, `telephony2.tar` |
| Controlled recording | PASS twice: decodable 41.78 s / 41.74 s recordings, both patterns present, real AMI/channel metadata correlated and file descriptors closed | Runtime archives and `consolidated-media-analysis.json` |
| Actual self-hosted agent/phone media | PASS twice: `repeat-agent-5` and `repeat-agent-postboot`; real VICIdial authentication, embedded 9051 plus independent 9052 VICIphone, two-way conference media | Agent browser traces/WAVs, `agent5-media-analysis.json`, `postboot-agent-media-analysis.json` |
| Heartbeat/calendar/strict SQL | PASS: upstream updater state current, initialized agent without calendar exception, original strict SQL mode retained | Post-boot agent traces, SQL/heartbeat observations, `post-final-correction-health.json` |
| Park/resume and transfer | PASS through actual agent UI in both workflows; prolonged post-boot park/resume twice; selected 9052 channel transferred to 9600001 while agent 9051 remained in 9600000 and PAUSED | `agent8-transfer-state.json`, `postboot-park-state.json`, `postboot-park2-dtmf.json`, `postboot-transfer2-state.json` |
| Hangup and DTMF | PASS: actual VICIphone hangup; two keypad operations observed as SIP INFO digit 6, durations 100 ms and 250 ms, received by Asterisk | `dtmf-final-read.json` and agent traces; no claim that mute is hold |
| Unexpected idle/active WSS loss | PASS for documented manual recovery contract: visible WSDisconnected, manual Register, one contact, explicit old-call cleanup and new successful call; active media could continue during signaling loss | Agent traces, post-boot WSS cut/recovery/new-call state records |
| Logout authorization | PASS in both main workflows and focused final-patch replay: old protected request rejected with `Invalid session`; expected history retained; fresh login succeeds | `agent5-logout-revocation.json`, `postboot-logout-revocation.json`, `visibility-final-logout-revocation.json` |
| Certificate/credential rejection | PASS: wrong SIP password rejected in restart and second-reboot checks; untrusted CA rejected by two fresh isolated browsers | Restart/reboot browser records, `untrusted-certificate-rejection.json` |
| Credential/dependency audit | PASS within recorded coverage: 20 traces, 3,487 requests, 11 VICIphone POST launches, zero GET launches, 67 console entries, 31 history rows, 11 tested passwords; no tested URL/Referer/console/access-log/history/source leaks, no external request origins | `credential-transport-audit-all-traces.json`; protected raw POST/Authorization/session evidence excluded from export |
| Security and final cleanup | PASS: Enforcing; socket-only DB; loopback AMI/FastAGI; intended non-root execution; zero active channels/calls/contacts/conferences/live agents/park rows/reservations/pending actions | `post-final-correction-health.json`, `final-queue-cleanup.json` |
| Operational persistence | PASS: one Asterisk restart and two full VM reboots with distinct boot IDs; all seven required services enabled and active without manual starts, fresh media after recovery | Restart, both full-reboot and post-boot health/browser records |
| Direct-public / 1:1 NAT / home agent | NOT_RUN: deployment-specific, no authorized external topology; not a core M0/installer gate | Current authorized scope decision; no LAN-to-public inference |
| Independent AGY and maintainer decisions | NOT_RUN / pending; implementer checks cannot satisfy these gates | Candidate manifest unchanged; no acceptance or promotion |

The seven startup services are `asterisk`, `mariadb`, `httpd`, `php-fpm`,
`almadial-m006-fastagi`, `almadial-m007-updater` and
`almadial-m007-listener`. The bounded sender is deliberately not an unbounded
campaign service.

### Quantitative media results and retained negative evidence

Native paired Echo pattern/correlation/tone agreement was
0.98611/0.98651/0.98894 before restart, 0.99209/0.98819/1.0 after restart, and
0.99209/0.99900/1.0 after the second reboot, with 40 ms measured delay each.
Both ALM-6 runs passed both independent source directions. Monitor mix agreement
was 1.0 over 294 distinguishable frames and 0.96271 over 295; recording mix
agreement was 1.0 over 442 frames in both runs.

VICIphone's first completed agent run passed both directions: pattern agreement
1.0 / 0.99408, correlation 0.99436 / 0.99646, tone agreement 1.0 / 0.98004.
The post-boot run passed with pattern 1.0 / 0.99605, correlation 0.97998 / 0.99258,
tone agreement 0.99781 / 0.98222 and 60 / 40 ms measured delay. Existing thresholds
remain pattern >=0.90, correlation >=0.85, tone agreement >=0.90, RMS >=0.02,
delay <2,000 ms. Mixed-content checks retain their own accepted thresholds.

Failed attempts remain in evidence. In particular, the first consolidated Echo
analysis incorrectly compared a later capture against the start of a long fixture;
its FAIL remains preserved. `paired-echo-analysis.json` compares simultaneous
local and remote captures and supersedes that invalid alignment, without lowering
thresholds. The muted monitor's source-positive test returns FAIL as expected:
it confirms source C was absent before barge, joined with the actual muted CLI
state. Neither result is silently relabeled as a successful positive test.

## Reproducibility defects and minimal corrections

These are actual interventions discovered during this second installation.
They are now explicit in the helper or instructions; original failed records and
pre-correction backups remain protected. The consolidated final recipe was not
executed as a third clean installation. Independent QA must inspect the recorded
corrections and reproduce affected behavior rather than infer a fresh third pass.

| Observed failure / cause | Minimal documented correction | Repeated verification |
|---|---|---|
| Asterisk did not find modules under assumed `/usr/lib64` | Set installed `astmoddir=/usr/lib/asterisk/modules` | Required modules, live calls, restart and both reboots |
| Restrictive key-directory umask prevented service traversal | Root:asterisk directory 0750, private files 0640 | Validated WSS and non-root Asterisk operation |
| Generated global definitions were outside `[globals]` | Place generated include inside explicit globals section | Real generated dialplan, conferences and FastAGI |
| AGI runtime failed because pinned Asterisk Perl modules were missing | Install original r4037 `Asterisk.pm`/`Asterisk/AGI.pm` into versioned traversable module path | Non-root load and actual Asterisk FastAGI twice |
| 48-character AMI fixture exceeded `ASTmgrSECRET` varchar(20) | Generate and consistently use 20 hex characters | Strict insertion, generator and AMI authentication |
| Numeric fixture values selected unintended ENUM members, including error 1265 | Quote `webroot_writable='0'` and `agentcall_manual='1'` | Immutable webroot, strict inserts and real manual dialing |
| Embedded phone options and no-prefix dialing did not select intended path | Use upstream `--` separator and `X` no-prefix sentinel | Embedded registration and local 99052 manual call twice |
| Runtime role could not originate its local test channel | Add only required `originate` permission to the existing AMI role | Actual originations; unauthorized AMI Command remains rejected |
| Updater removed valid parked PJSIP channels after six misses in legacy channel table | For pinned Asterisk 22 PJSIP park only, query `live_sip_channels` with actual `park_CID.agi` data | Two prolonged post-boot parks (~54/~50 s), DB rows retained and resume succeeds |
| Shell wrapper appeared as a duplicate manager sender to upstream process detection | Direct argv, one queued local action per invocation | Actual dial/park/resume/transfer/logout actions complete and queue drains |
| h-extension closure lacked explicit module dependencies | Load `func_hangupcause` and `app_exec` | Real call closure and clean teardown in both telephony runs |
| Admin first request HTTP 500, `fwrite(false)` TypeError before immutable-webroot check | Precreate root:apache 0640 `vicidial/old_clear`; perform upstream first-login password initialization | Two real admin create/read operations, anonymous rejection and fresh agent/SIP login |
| Repeated visibility audit insert caused strict SQL duplicate-key error 1062 | Narrow `ON DUPLICATE KEY UPDATE event_end_epoch=VALUES(event_end_epoch)` in pinned `conf_exten_check.php`; preserve first observation | Original [0,1062] versus derived [0,0], one row; invalid ENUM still 1265; two actual browser POSTs 200 with no fatal; logout revoked |

The final visibility correction does not use INSERT IGNORE or loosen SQL modes.
Global and effective sessions retain
`STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.
The updater correction is bound to the actual PJSIP/Asterisk 22 table contract,
not a replacement heartbeat daemon. Upstream source trees remain unchanged.
Missing optional audio prompts were reported as nonfatal; no substitute media
or speech proof was fabricated. Guest `rg` absence affected a diagnostic only;
the recorded probe was rerun with Python. Apparent PHP/SQL time-age disagreement
was UTC versus local presentation, not evidence of a bad NTP clock.

## Cross-installation comparison

`cross-vm-comparison.json` joins read-only reference observations and fresh
repeat-VM observations. Both have 744 RPMs. OS, running kernel, compiler, PHP
and extension inventory, schema and strict SQL-mode observations match. The sole
RPM inventory difference is `kernel-headers` release `211.73.1.el10_2` on the
reference versus `211.74.1.el10_2` on the repeat VM; this dependency variation is
recorded rather than called exact package-byte reproduction. Primary selected
component versions match the pinned identities above.

Native Asterisk SHA-256 differs:

- Reference: `db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c`.
- Repeat: `9b114b1ca4795b22baead24f5dac02d2f85e194c1df8d0c373a953c66df7059d`.

Build hostname, time and paths differ. Effective 22.12.0 version, required
modules and tested behavior agree; bit-for-bit build reproducibility is not
claimed. IPs, fresh secrets/CA, machine/boot IDs, SIP Call-IDs, channel IDs,
timestamps and recording durations are expected environmental differences.
Reference AGI modules load with its explicit `/opt/almadial-m007` library path;
repeat uses identical pinned module bytes in the versioned site library.
Reference runtime was not modified to apply this replay's new corrections.

The final 59 deployed application assets have zero mismatches against fresh
derivation manifest SHA-256
`a2084f42f6a7efa7ac23ffcd5f4b0a7491902984eb943755c22b3887d8cba963`.
Important derivation SHA-256 identities:

| Derivation | SHA-256 |
|---|---|
| Schema 1748 | `0992448a2607dbad6841250a09b68dafd360cb4fa44f8589d89526ef16522af8` |
| Configuration generator | `8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda` |
| AST_update.pl park correction | `33cb29f0828e755a387c32530febf425e0e2a8d5b120a1432bc9b9877d5dad86` |
| conf_exten_check.php visibility correction | `f4a888e45503105befc02fca276a4473e9375609bdb5d36a41ed9eab66a21e9f` |

All source pins, other derived hashes and exact deployment paths are in the
protected source/deployment manifests. The original ALM-7 ledger (255 artifacts,
`7c8a73396a0f27d22c4ea9bf195e289398be4b3fb73ccabbfba9efc6a9adabbe`)
and continuation ledger (426 artifacts,
`6e495f927362f0ef0fba9e9e59265225a6456d275727ed366e0fe90c4093398c`)
were independently rehashed read-only; all recorded bytes matched.

## Stack-lock acceptance proposal — HOLD

Retain `manifests/stack.lock.yaml` as **candidate**. Propose eventual compatibility
acceptance of the exact selected AlmaLinux 10.2 / Asterisk 22.12.0 / PJPROJECT
2.17 / VICIdial r4037 schema 1748 / MariaDB 10.11.18 / Apache 2.4.63 / PHP 8.3.36 /
Perl 5.40.2 / VICIphone 3.0 / SIP.js 0.20.1 identities, with the hash-bound
derivations and package provenance recorded above. This proposes compatibility
acceptance only, not production readiness.

The canonical deliverable names `manifests/almalinux10-single.lock.yaml`.
Proposed path/schema policy for maintainer decision: keep the current candidate
as historical input; create that separate accepted-lock path only after AGY and
maintainer approval, using a reviewed schema that records exact component pins,
required derivation hashes, both evidence-ledger identities, approved scope,
review identity and promotion authorization. Candidate-only validators must receive
a separately authorized bounded policy change before any accepted lock is written
or consumed. This report is the proposal; no accepted manifest or acceptance
schema has been created, and existing validators are not bypassed.

Proposed accepted scope: **AlmaLinux 10.2 x86_64 single-server VICIdial installation
and tested local/LAN operation**, using the exact versions, authenticated inputs,
package inventory and hash-bound derivations above. Supported evidence covers
native Echo, generated PJSIP/ConfBridge, AMI/FastAGI, controlled recording, actual
self-hosted VICIphone agent login/media, park/resume/blind transfer/DTMF/hangup,
logout revocation, documented manual WSS recovery, restart and reboot persistence.
No automatic reconnect, transparent call continuation, physical acoustic test,
universal NAT traversal or production readiness is claimed. External profiles
remain NOT_RUN and require evidence before claiming that particular configuration
works. This is a proposal, not manifest promotion.

Evidence supports local functional equivalence on a clean second VM, including
the defects and corrections above. Independent AGY audit/reproduction and explicit
maintainer acceptance remain prerequisites for M0 closure and accepted-lock
promotion. The final consolidated recipe has not been executed on a third clean
VM. That is not automatically required: original-source derivation replay,
focused strict-SQL tests, two extended park/resume checks, final deployment hashes
and post-correction browser/session verification cover the changed operations.
No substantive unrecorded step or contradictory required runtime outcome remains
in this review. AGY must audit the final recipe and reproduce critical checks;
a newly demonstrated defect may justify another affected-step replay. ALM-27
retains selected-deployment qualification; ALM-29 retains two-clean-install
installer release, security/error/reboot and explicitly advertised capability
evidence requirements. Codex retains ALM-8; only QA Review is authorized at this
handoff, not acceptance. ALM-9 and later implementation are not started.

## Repository validation and evidence handling

Repository validation recorded 126 passing unit tests (previous baseline 118).
`validate_scaffold.py`, unittest discovery, `verify_candidate_stack.py`,
`verify_config_generation.py`, Python compilation of all six affected Python
files, and `node --check scripts/lab/m0_07_browser.js` each exited 0. The two
verifiers remain offline fixture checks; they do not themselves establish live
compatibility. `git diff --check` is recorded after the final documentation edit.

Starting HEAD was `46d7948b701be274c17f5d7525f1529288fb64a8`; a concurrent
operator commit advanced it to `692047ebcefc350721d08a12b275a8b532df14cb`.
That TLS work and unrelated reviewer/operations files are preserved. Codex made
no commit, stage, reset, clean, rebase or push. Task-owned changes are this guide,
`m0_08_replay.py`, its package specification and five focused tests, plus the
explicit authorized-host support in the existing ALM-6 runtime helper and the
two minimal ALM-7 derivations with regression coverage.

Raw credentials, cookies, POST bodies, private keys, browser profiles, traces,
database backups and recordings remain outside Git. The execution evidence
directory is mode 0700. Its final SHA-256 ledger covers retained regular evidence
files; mutable browser profile/cache trees are explicitly excluded and all final
traces, exports, analyses, command records and manifests remain covered. The
ledger's hash and artifact count are reported in the Kaneo evidence comment;
tracker read-back receipts created afterwards are separately hashed to avoid
circular provenance. No historical evidence ledger is overwritten.


## Scope-realignment continuation — 2026-11-07

Starting/current continuation HEAD is
`665d4d650e07b8379842c2095ae90730fc117232`; the operator's TLS commit
`692047ebcefc350721d08a12b275a8b532df14cb` and subsequent backup commit are preserved.
This continuation changes requirements/docs and rechecks evidence and offline
regressions. It performs no VM rebuild, installation, reboot or runtime mutation.
Original recorded runtime outcomes above are reviewed evidence, not a claim of
fresh media execution in this continuation.

The protected original ledger has SHA-256
`ebaa490c6650d672121d20f1dbcfdf86038703435b0f49255326b907310619fc`;
all **475** covered regular artifacts verify, with zero mismatches. Mutable
browser/cache exclusions remain as specified in `ledger-scope.json`. A separate
continuation directory is
`~/almadial-lab/evidence/m0-08/scope-realignment-20261106T213215Z/`.
It retains before/after task and plan records, integrity/correction checks,
repository validation and MCP read-backs. Its ledger identity is in the final
Kaneo handoff comment; post-ledger tracker receipts are separately hashed.

### Final compatibility corrections reviewed

- **Park lookup:** input `bin/AST_update.pl` SHA-256
  `ee0e5a752d47ababb0db4657a0825757b62fd41433bc4a04a4ed7ea968b8ef3b`.
  `m0_07_assets.updater_overlay` retains the original legacy query and overrides
  it only for Asterisk 22 PJSIP channels: `live_sip_channels`, exact server/channel,
  and `channel_data='park_CID.agi'` or its path suffix. Derived hash is
  `33cb29f0828e755a387c32530febf425e0e2a8d5b120a1432bc9b9877d5dad86`.
  Existing Perl branch tests retain non-22 and non-PJSIP behavior. The post-boot
  UI samples show two sustained parks (~54/~50 seconds) followed by resume;
  channel/DB records correlate the parked 9052 channel. No unrelated channel
  handler is replaced.
- **Visibility insert:** input `www/agc/conf_exten_check.php` SHA-256
  `1dcff537c57b61e2d8ea1bdc3770463fcc27a554982a39a444cea3fc3e628316`.
  `session_overlay` replaces the single pinned audit insertion tail with
  `ON DUPLICATE KEY UPDATE event_end_epoch=VALUES(event_end_epoch)`.
  It preserves the original first observation while allowing repeated batches
  for the same audit key. Derived hash is
  `f4a888e45503105befc02fca276a4473e9375609bdb5d36a41ed9eab66a21e9f`.
  `visibility-idempotence-runtime.json` records original exits [0,1062], derived
  [0,0], exactly one row, and invalid ENUM error 1265 in both cases under the same
  strict mode. Two actual browser batches returned 200 without a fatal error;
  final logout replay remained `Invalid session`. No global/session SQL-mode
  relaxation or `INSERT IGNORE` is used.

The original `postboot-park2-dtmf.json` diagnostic exits **127**, because `rg` was
not installed in the guest. Its preceding channel/park observations are retained;
this command is not counted as an all-successful DTMF probe. Successful DTMF
reads are `dtmf-final-read.json` / `dtmf-log-read.json`; UI/resume evidence is
`postboot-park-ui-samples.json` plus the dispatch/channel records. A future
operator can use Python filtering instead of assuming guest `rg` exists. No
required runtime outcome relies solely on the failed diagnostic command.

### Replay documentation audit and QA contract

The ordered phases above specify the mandatory base installation operations,
input pins, commands, effective ownership/permissions, expected exit and runtime
checks, stop boundaries and observed corrections. The final r12 input bundle
includes both corrections; `application()` invokes the same pinned overlay and
provisions `old_clear`. Replaying that final input produces the recorded final
application hashes without modifying upstream source. Earlier ad hoc failed-run
repairs are historical evidence, not additional secret steps in the final recipe.
Per-phase failures must stop for inspection rather than automatically restart
irreversible database initialization. Preserve the immutable source/bundle and
its provenance; if a required artifact is absent or differs, stop rather than
substitute a newer version or copy installed reference state.

Cross-VM classification remains: 744 RPMs each; selected versions, running
kernel/compiler, PHP extensions, schema/strict mode and required Asterisk modules
agree. `kernel-headers` 211.73.1 versus 211.74.1 and native binary build bytes are
documented equivalent runtime variation. VM identity, CA/secrets, timestamps and
channel identifiers are expected environmental variation. The repeat VM's two
new application derivations intentionally differ from the reference; final
repeat deployment integrity verifies all 59 files. Binary reproducibility and
identical host-specific values are not claimed.

The QA reviewer must verify the authorized scope decision and MCP read-back,
source/derivation pins, the clean second installation and recorded functional
outcomes, executable recipe and documented variations. Independent acceptance
and any manifest/schema promotion remain separate decisions.
