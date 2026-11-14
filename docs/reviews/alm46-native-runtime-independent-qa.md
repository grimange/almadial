# AlmaDial — ALM-46: Independent Native VICIdial Runtime and Remote Single-Server QA Report

## 1. Governance and Implementation Identity

- **Kaneo Task:** `ALM-46` (`a1doi4qsyn1lspqtydw84smh`) — *Implement remote single-server HTTPS, WebRTC media and portable deployment*
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY)
- **Reviewed Implementation Commit:** `92b14eb4452b7bfb949e0ee7efc571a83d9e799c` (`fix(runtime): enable native VICIdial agent operation and health reconciliation`)
- **CLI Version Identity:** `alm46-native-rc1` (`go1.27.3`, `linux/amd64/v1`, `CGO_ENABLED=0`, `-trimpath`, `-mod=readonly`)
- **Verified Binary SHA-256:** `a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3` (independently rebuilt from `92b14eb4452b7bfb949e0ee7efc571a83d9e799c` via `scripts/build_go.py` and verified byte-identical to the tested binary on the QA host)
- **Developer Evidence Handoff:** Comment `g8wzig4a2cp4vx0203gdle05`
- **Developer Evidence Ledger:** `/home/ra/almadial-lab/evidence/remote-single-server/alm46-runtime-20261114T014703Z/SHA256SUMS` (SHA-256 `68bad5f0acd8af328c5a7af521ee1de5e5b75d9377a1f43faa046e98bd55d71b`, 997/997 files verified `OK`)
- **Independent QA Evidence Ledger:** `/home/ra/almadial-lab/evidence/remote-single-server/qa-alm46-native-runtime-20261114T033000Z/SHA256SUMS` (SHA-256 `335175cc74e19e1fc861bb8390f1d9a12b21f4cbba013ba6719b17915c3bb09f`, 21/21 files verified `OK`)
- **Downstream Tasks Unchanged:** `ALM-34` (`in-progress`, Codex) and `ALM-35` (`to-do`, unassigned) preserved untouched.
- **Final Independent Verdict:**
  ```text
  ALMADIAL_ALM46_INDEPENDENT_QA_PASS
  ```

---

## 2. QA Host and Network Topology

- **Authorized Disposable QA Host:** `almadial-alm46-remote-dev` (Multipass/QEMU, AlmaLinux 10.2 `x86_64`, 4 vCPUs, 8 GiB RAM, 80 GiB disk)
- **Machine Identity:** `436ca8ae883439638507ab64b02d98e8`
- **Independent QA Boot ID:** `df6281f9-b385-4ca8-8e99-f6458c8f63bc` (distinct from developer boot `61c65a93-5900-4419-9dc3-57d818042088`)
- **Configured Remote FQDN & IPv4:** `dialer.alm46.test` -> `10.160.90.183` (controlled off-host bridge `mpqemubrzone1`)
- **Workstation Browser Client:** Off-host Google Chrome (`10.160.90.1`) connecting directly to `https://dialer.alm46.test` (TCP 443) and `wss://dialer.alm46.test/ws` with `ignoreHTTPSErrors: false` and zero SSH tunnels.
- **TLS Certificate Verification:** Server leaf certificate (`CN=dialer.alm46.test`, SAN `DNS:dialer.alm46.test, IP Address:10.160.90.183`, validity `2026-11-13 23:53:35 GMT` to `2026-11-27 23:53:35 GMT`) validated against exact task CA (`1a0a68d13a864360f02880cf15513049c099b2b6ecb05ce52d0b0343b4d807d3`). Temporary NSS trust entry removed immediately after QA execution (`qa-browser-trust-cleanup.json`).
- **Historical & Accepted Hosts:** `almadial-alma10`, `almadial-alma10-repeat`, `almadial-alm16-rpm`, and `almadial-alm29-clean-{a..e}` remained `Stopped` and untouched.

---

## 3. Evidence Matrix: Developer Evidence Reviewed vs. Independently Executed QA vs. Not Run

| Area | Developer Evidence Reviewed | Independently Executed by AGY | Status |
|---|---|---|:---:|
| **Implementation & Build Identity** | `release.json`, `regression-committed/summary.json` | Reviewed commit `92b14eb4452b7bfb949e0ee7efc571a83d9e799c`; ran `go vet`, `go test` on `internal/{cli,diagnostics,provision}`, Python 255-test suite, scaffold check, 26 r4037 derivations; independently rebuilt `alm46-native-rc1` matching `a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3` | **PASS** |
| **Evidence Ledger Integrity** | `alm46-runtime-20261114T014703Z/SHA256SUMS` (`68bad5f0...`) | Executed `sha256sum -c SHA256SUMS`: 997/997 files verified `OK` | **PASS** |
| **Native VICIdial Runtime & Supervision** | `post-reboot-runtime-inventory.json` | Live inspection on boot `df6281f9-b385-4ca8-8e99-f6458c8f63bc`: `almadial-native-{updater,listener,sender,autodial}.service` all `active`/`enabled`, user/group `asterisk`, `NRestarts=0`, clean cgroups; 6 `almadial-s06-fastagi.service` processes; zero competing `ADMIN_keepalive_ALL.pl` daemons | **PASS** |
| **Local AMI, Dialplan, Modules & Cron** | `post-reboot-runtime-inventory.json` | Verified live loopback AMI `127.0.0.1:5038` (`Login`, `Ping`, `Command`, `Logoff` as `almadial-native`), `/etc/astguiclient-native-ami.secret` (`0640 root:asterisk`, 20 hex chars, readable by `asterisk`), `app_dial.so`/`app_confbridge.so`/`res_agi.so`/`app_mixmonitor.so` running, `almadial-native-agents` and `8309` MixMonitor dialplan, `park_CID.agi` `@INC`, `/etc/cron.d/almadial-native` (`64d40327...`), fresh `server_updater` heartbeat (0 s lag) | **PASS** |
| **Network Isolation, Firewall & SELinux** | `post-reboot-runtime-inventory.json` | Verified `getenforce == Enforcing`, `ss -lntup` (3306, 5038, 4577, 8088, 8089, 80 on `127.0.0.1` only; 22 and 443 external), runtime & permanent firewalld policy `almadial-s09` (`DROP`, `22/tcp 443/tcp 10000-10019/udp`, ICMP) | **PASS** |
| **Administrator Direct HTTPS Workflow** | `ordinary-agent/admin-1794624939226.json` | Executed fresh-profile Chromium workflow over `https://dialer.alm46.test/vicidial/admin.php` (`HTTP 200`, `ignoreHTTPSErrors: false`, no SSH tunnel); verified Campaigns (`ALM46`), Users (`alm46qa`), and Phones (`9061`, `9062`) listings and records | **PASS** |
| **Ordinary Agent, VICIphone & Manual Call** | `ordinary-agent/agent-browser-1794625562920.json` (call `1794625522.5`) | Executed independent Chromium workflow (`qa-agent-workflow.json`): customer `9062` WSS registration + disconnect/re-register recovery, agent `alm46qa` login to `ALM46`, embedded VICIphone `9061` WSS registration, `Call Agent Webphone` -> ConfBridge `9600000`, `MANUAL DIAL` lead `46` (`Dial Now`) -> customer `9062` answer (uniqueid `1794627303.5`), zero JS errors | **PASS** |
| **Bidirectional WebRTC Audio** | `ordinary-agent/audio-analysis-1794625547802.json` | Captured 12 s peer streams (`capture_id: 1794627328606`) + connected ICE/DTLS-SRTP stats; verified with `m0_03_signal.py`: Agent→Customer pattern `0.9960`, envelope `0.9822`, tone `0.9648`, RMS `0.1083`, latency `60 ms`; Customer→Agent pattern `1.0000`, envelope `0.9904`, tone `0.9522`, RMS `0.1042`, latency `60 ms` | **PASS** |
| **Native Recording & DB Correlation** | `ordinary-agent/post-reboot-recording-correlation.json` (ID `4`, `ALM46_REC_4-all.wav`) | Verified new recording ID `5` (`/var/spool/asterisk/monitorDONE/ALM46_REC_5-all.wav`, SHA-256 `7f814ec8dd241dbeb9fe7cb7770627d67f08d1b598eded0bc35588bf7043f088`, 393,324 bytes, 24.58 s, `pcm_s16le` 8 kHz mono, `ffmpeg` decode exit 0, RMS `0.1284`), correlated via `uniqueid = 1794627303.5` across `recording_log` and `vicidial_log` | **PASS** |
| **Disposition, Hangup & Logout Cleanup** | `ordinary-agent/agent-browser-1794625562920.json` | Submitted `DC` disposition via agent UI, hung up agent webphone, clicked `LOGOUT`; verified `vicidial_log.status = 'DC'`, `0` active Asterisk channels, and `0` rows in `vicidial_live_agents` | **PASS** |
| **Stopped-Worker Reconciliation & Healthy NO_OP** | `stopped-worker-reconciliation-{plan,repeat}.json`, `final-committed-{plan,repeat}.json` | Reviewed developer stopped-autodial repair evidence (`no_op: false`, 2 repair actions); independently executed `almadial-native-rc1` after reboot and call test: `status`/`doctor` report 0 `FAIL`/`UNKNOWN` (exit `10` solely for truthful 14-day lab cert expiry `WARN`), `plan` returns `actions: []`, `no_op: true` (exit `0`), repeat `install` returns `no_op: true` and exit `0` | **PASS** |
| **Lab-Profile Preservation** | Automated unit/contract tests | Verified `TestNativeReadinessTransitionAndLabBoundary` and `report_linux.go` / `install_linux.go` branch isolation; isolated lab profile retains deferred workers and inactive cron contract | **PASS** |
| **Public VPS / Carrier / Physical Mic Audio** | Explicitly recorded as `NOT_RUN` | Outside controlled-network ALM-46 scope (`REMOTE_PUBLIC_HOST_NOT_RUN`, physical mic/speaker `NOT_RUN`, remote recording HTTP playback not exposed) | **NOT_RUN** |

---

## 4. Independent Runtime & Workflow Observations

### 4.1 Native Runtime Supervision and Local AMI (Boot `df6281f9-b385-4ca8-8e99-f6458c8f63bc`)
- **Systemd Native Workers:**
  - `almadial-native-updater.service`: `active (running)`, `enabled`, `MainPID=1378`, `User=asterisk`, `/usr/bin/perl /usr/share/astguiclient/AST_update.pl`, `NRestarts=0`
  - `almadial-native-listener.service`: `active (running)`, `enabled`, `MainPID=1376`, `User=asterisk`, `/usr/bin/perl /usr/share/astguiclient/AST_manager_listen.pl`, `NRestarts=0`
  - `almadial-native-sender.service`: `active (running)`, `enabled`, `MainPID=1382`, `User=asterisk`, `/usr/bin/perl /usr/share/astguiclient/AST_manager_send.pl`, `NRestarts=0`
  - `almadial-native-autodial.service`: `active (running)`, `enabled`, `MainPID=1380`, `User=asterisk`, `/usr/bin/perl /usr/share/astguiclient/AST_VDauto_dial.pl`, `NRestarts=0`
- **FastAGI & Maintenance:** 6 `almadial-s06-fastagi.service` processes owned by `asterisk`; `ADMIN_keepalive_ALL.pl` configured with `$VARactive_keepalives='X'` and run via oneshot `almadial-native-maintenance.service` and `/etc/cron.d/almadial-native` (`sha256: 64d4032724c07d8b5d1f2796c4109b3f1ac5729553df20b48fd1042fae3f2fbe`).
- **Database & AMI Readiness:** `server_updater` heartbeat fresh (`0 s` delta); `servers` row (`TESTast`, `127.0.0.1`) configured with `CONFBRIDGE`, `22.12.0`, `almadial-native`, and `wss://dialer.alm46.test/ws`.

### 4.2 Independent Ordinary Agent Call, Media and Recording
- **Call Identity:** `uniqueid = 1794627303.5`, `lead_id = 46`, `list_id = 4600`, `campaign_id = ALM46`, `user = alm46qa`, `phone_number = 062`, `status = DC`, `length_in_sec = 24`.
- **Conference & Channels Observed During Call:**
  - Agent leg: `PJSIP/9061-00000001` in `ConfBridge(9600000,vici_agent_bridge,vici_agent_user)`
  - Customer leg: `PJSIP/9062-00000002` bridged via `Local/9600000@default` into `ConfBridge(9600000,vici_agent_bridge,vici_customer_user)`
  - Recording leg: `Local/59600000@default` running `MixMonitor(/var/spool/asterisk/monitor/${CALLERID(name)}-in.wav)` at `8309@vicidial-auto-internal`
- **Bidirectional Audio Verification (`capture_id: 1794627328606`):**
  - Agent → Customer: `PASS` (pattern match `0.9960`, envelope correlation `0.9822`, tone agreement `0.9648`, remote RMS `0.1083`, latency `60 ms`)
  - Customer → Agent: `PASS` (pattern match `1.0000`, envelope correlation `0.9904`, tone agreement `0.9522`, remote RMS `0.1042`, latency `60 ms`)
- **Archived Recording Verification:**
  - `recording_id = 5`, file `/var/spool/asterisk/monitorDONE/ALM46_REC_5-all.wav`
  - Size: `393,324` bytes; SHA-256: `7f814ec8dd241dbeb9fe7cb7770627d67f08d1b598eded0bc35588bf7043f088`
  - Format: `pcm_s16le`, `8000 Hz`, mono, `24.58 s`, normalized RMS `0.1284`, `ffmpeg` decode exit `0`.

### 4.3 Diagnostics, Reconciliation and Repeat Installation NO_OP
- **`almadial status` / `almadial doctor` (`alm46-native-rc1`):**
  - Exit code: `10`
  - 45 total checks: 43 `OK` (including `runtime.fastagi`, `runtime.updater`, `runtime.listener`, `runtime.sender`, `runtime.autodial`, `runtime.cron`, `runtime.supervision`, `runtime.ami`, `runtime.readiness`), 1 `WARN` (`tls.expiration` — truthful warning for 14-day lab certificate valid until `2026-11-27T23:53:35Z`), 1 `NOT_CONFIGURED` (`installer.journal`, non-blocking). Zero `FAIL` or `UNKNOWN`.
- **`almadial plan` (`alm46-native-rc1`):**
  - Exit code: `0`
  - Output: `{"plan_sha256":"10442fdd938f54603b33d3344925f542415c33d7626e78f43079d9331e226e4c","actions":[],"no_op":true}`
- **Repeat `almadial install` (`alm46-native-rc1`):**
  - Exit code: `0`, `no_op: true`, all stages `S01`–`S10` (with `S06 FastAGI and native VICIdial runtime`) and `S12` completed.

---

## 5. QA Findings and Explicit Limitations

- **P0 Blockers:** None (`0`).
- **Nonblocking Observations & Explicit Scope Boundaries:**
  1. **Short-lived Lab Certificate Warning (`tls.expiration: WARN`):** Expected 14-day laboratory certificate lifetime causes `status`/`doctor` to exit `10`; certificate is currently valid and trusted.
  2. **Public Host Topology (`REMOTE_PUBLIC_HOST_NOT_RUN`):** Qualified on a controlled off-host bridge network (`10.160.90.0/24`); public VPS/DNS/certificate deployment remains unexecuted.
  3. **Physical Acoustic Transducers (`NOT_RUN`):** Deterministic browser WebRTC media injection and capture verified; physical microphone/speaker subjective listening not run.
  4. **Remote Recording Playback/Download:** Recordings are archived locally under `/var/spool/asterisk/monitorDONE/` and correlated in MariaDB; `recording_log.location` retains upstream loopback `https://127.0.0.1/RECORDINGS/...` and public HTTP recording download is not exposed in this task.
  5. **Installed Binary Path Note on Dev Host:** On `almadial-alm46-remote-dev`, `/usr/local/sbin/almadial` retained the initial bootstrap binary (`alm46-remote-dev4`), while the final committed `alm46-native-rc1` binary was staged at `/tmp/alm46-native-final` pending release-owner bootstrap rendering in `ALM-35`. All independent CLI checks used the verified `alm46-native-rc1` binary (`a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3`).
