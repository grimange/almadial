# AlmaDial — ALM-25 / M3-03: Independent Firewall and Network Isolation QA Report

## Acceptance Status and Governance

- **Task ID:** `b0o51bkbrb2ylh57lq5t6wzv`
- **Canonical Work Package:** ALM-25 / M3-03
- **Task Title:** Configure host firewall and isolated network surface
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Starting Git Baseline:** `71e0780976b22466416ca1b1fa255e2ac668d0b8`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (`10.160.90.233`, Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `almadial-alm16-rpm.alm25-pre-firewall-20261111` (preserved)
- **Developer Verdict:**
  ```text
  ALMADIAL_M3_03_NETWORK_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_03_NETWORK_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S09 configures native single-server host network isolation and firewalld filtering on AlmaLinux 10. Independent QA was conducted across Go source code, focused Python integration tests, live laboratory VM evaluation on `almadial-alm16-rpm`, hypervisor-bridge socket probing over both IPv4 and link-local IPv6, reload persistence verification, failure-recovery reconciliation, and bit-for-bit baseline preservation.

All canonical acceptance criteria for ALM-25 are satisfied:
1. **firewalld Active & Enabled:** firewalld 2.4.3 is active (running) and enabled under systemd supervision.
2. **Managed Policy Ownership & Priority:** S09 installs and owns `/etc/firewalld/policies/almadial-s09.xml`. The policy targets `DROP` with priority `-20000`, filtering ingress from `ANY` zones to the `HOST` egress zone. It executes ahead of vendor policies (such as `allow-host-ipv6` at priority `-15000`) and standard zone rules.
3. **Explicit Permitted Traffic:** Only TCP port 22 (SSH) and ICMP/IPv6-ICMP protocols are permitted in `almadial-s09`. No external application (443), Asterisk WSS (8089), MariaDB (3306), AMI (5038), FastAGI (4577), control (18025), SIP (5060), or RTP grants exist.
4. **Foreign Configuration Preservation:** Pre-existing firewall zones (including `public`) and vendor policies remain intact in permanent and runtime configuration.
5. **Effective Network Isolation:** Socket probes from the hypervisor bridge (`mpqemubrzone1`) verify that SSH port 22 is reachable over both IPv4 (`10.160.90.233`) and link-local IPv6 (`fe80::5054:ff:fe63:52a6`), while all protected ports (443, 8089, 3306, 5038, 4577, 18025) are silently dropped and time out.
6. **Functional Loopback Services:** Localhost Apache HTTPS (`https://localhost/vicidial/admin.php` returning HTTP 401 with header `X-AlmaDial-Web: S07`) and Asterisk WSS (`GET /ws` returning HTTP 101 Switching Protocols with `Sec-WebSocket-Protocol: sip`) remain fully operational with accepted S08 certificates.
7. **Reload Persistence & Management Safety:** A 120-second native systemd recovery timer (`almadial-s09-reload-recovery.timer`) protected the controlled `firewall-cmd --reload`. The policy survived reload and the timer was safely disarmed.
8. **Idempotency & Clean NO_OP:** S09 plans emit an empty action graph (`[]`) on a satisfied host, reporting `no_op: true` with zero mutations and zero file rewrites. Copied-journal recovery test reconciles `network.00.activate` to `succeeded` without replaying actions.
9. **S01–S08 Baseline Preservation:** 731 authorized RPMs (incorporating 6 signed firewalld dependencies), schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 107 phone columns, 56 Asterisk modules, FastAGI running, dialer cron inactive, and SELinux Enforcing with zero AVC denials.

---

## 2. Independent Acceptance Matrix

| Criterion | Result | Verified Observations & Evidence |
| :--- | :---: | :--- |
| **firewalld active/enabled** | **PASS** | `systemctl is-active firewalld` -> `active`; `systemctl is-enabled firewalld` -> `enabled`; `firewall-cmd --state` -> `running`. |
| **Managed policy and ownership** | **PASS** | Owned `/etc/firewalld/policies/almadial-s09.xml` installed; policy priority `-20000`, target `DROP`, ingress `ANY`, egress `HOST`. Permitted ports: `22/tcp`; permitted protocols: `icmp`, `ipv6-icmp`. |
| **Runtime/permanent consistency** | **PASS** | `firewall-cmd --info-policy=almadial-s09` matches permanent configuration bit-for-bit. Normalizer correctly ignores only the runtime `(active)` header suffix without ignoring semantic divergence. |
| **IPv4 isolation** | **PASS** | Bridge probe from `10.160.90.1`: TCP 22 reachable (`SSH-2.0-OpenSSH_9.9`); TCP 443, 8089, 3306, 5038, 4577, 18025 all timed out (packet dropped by nftables `filter_IN_policy_almadial-s09`). |
| **IPv6 link-local isolation** | **PASS** | Bridge probe from `fe80::5054:ff:fe63:52a6%mpqemubrzone1`: TCP 22 reachable (`SSH-2.0-OpenSSH_9.9`); TCP 443, 8089, 3306, 5038, 4577, 18025 all timed out. |
| **SSH management continuity** | **PASS** | Non-disruptive SSH connectivity maintained over both IPv4 and link-local IPv6 throughout activation, inspection, and verification. |
| **Firewall reload/persistence** | **PASS** | `firewall-cmd --reload` executed under 120s rollback timer; policy and nftables chains survived intact; recovery timer disarmed after verification. |
| **Foreign configuration preservation** | **PASS** | Zone `public` and vendor policy `allow-host-ipv6` preserved intact. No unmanaged zones or policies clobbered. |
| **S09 recovery** | **PASS** | Copied interrupted journal fixture reconciles `network.00.activate` to `succeeded` without replaying commands; canonical journal unchanged. |
| **Fresh NO_OP** | **PASS** | `almadial-network plan` on satisfied host returns empty graph `[]`; `almadial-network lab-run` reports `no_op: true` with zero runner traces and zero mutations. |
| **S01–S08 preservation** | **PASS** | 731 RPMs (6 authorized firewalld dependencies added to accepted 725), MariaDB schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 107 phone columns, SELinux Enforcing. |
| **Regression** | **PASS** | Go 1.27.3 vet/fmt/test/race passed (223 top-level tests, 1,392 pass events, 0 race defects); Python passed (203 tests, including 11 focused S09 tests); scaffold, candidate stack, config generation, 17 CLI smoke, and 11 PTY wizard journeys passed; 10 Linux amd64 command targets verified byte-identical. |
| **Evidence integrity** | **PASS** | Developer primary ledger (144 entries, SHA-256 `34ae8120fc3d46a33fb9c3aad18d24a651c70c66988bb0aa68bbc68e588f03c5`) verified 100% OK; independent QA evidence directory sealed with 4-entry SHA-256 ledger `c73ec34e8066853c5e74baa43a3866f52b2658d796ef77170b0414370f7547ff`. |

---

## 3. Network Policy and Isolation Details

### 3.1 Managed firewalld Policy
```text
almadial-s09 (active)
  disable: no
  priority: -20000
  target: DROP
  ingress-zones: ANY
  egress-zones: HOST
  services: 
  ports: 22/tcp
  protocols: icmp ipv6-icmp
  masquerade: no
  forward-ports: 
  source-ports: 
  icmp-blocks: 
  rich rules: 
```

### 3.2 Effective nftables Chains
In table `inet firewalld`:
- Chain `filter_INPUT_POLICIES` evaluates `almadial-s09` first due to priority `-20000`:
  `iifname "eth0" jump filter_IN_policy_almadial-s09`
- Chain `filter_IN_policy_almadial-s09_allow`:
  - `tcp dport 22 accept`
  - `meta l4proto icmp accept`
  - `meta l4proto ipv6-icmp accept`
- Chain `filter_IN_policy_almadial-s09`:
  Immediately drops any packet not matched in the allow chain (`drop # handle 197`).
  No traffic reaches lower-priority policies (`allow-host-ipv6` at `-15000`) or standard zone rules.

### 3.3 Independent Bridge Probing Results
Target VM: `10.160.90.233` (IPv4) / `fe80::5054:ff:fe63:52a6` (IPv6 link-local)
Vantage: Host bridge `mpqemubrzone1` (`10.160.90.1`)

| Port / Protocol | Service | IPv4 Status | IPv6 Link-Local Status | Isolation Mechanism |
| :--- | :--- | :--- | :--- | :--- |
| **TCP 22** | OpenSSH 9.9 | **REACHABLE** | **REACHABLE** | Explicitly allowed in `almadial-s09` |
| **TCP 443** | Apache HTTPS | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy + bound 127.0.0.1 |
| **TCP 8089** | Asterisk WSS | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy + bound 127.0.0.1 |
| **TCP 3306** | MariaDB | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy + bound 127.0.0.1 |
| **TCP 5038** | Asterisk AMI | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy + bound 127.0.0.1 |
| **TCP 4577** | FastAGI Perl | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy + bound 127.0.0.1 |
| **TCP 18025** | Control port | **BLOCKED (timeout)** | **BLOCKED (timeout)** | Dropped by `almadial-s09` policy |

---

## 4. Evidence Integrity and Ledger Audits

### 4.1 Developer Qualification Ledger
- **Path:** `/home/ra/almadial-lab/evidence/m3-03/20261111T081118Z-alm25-finish/`
- **Ledger File:** `SHA256SUMS`
- **Ledger SHA-256:** `34ae8120fc3d46a33fb9c3aad18d24a651c70c66988bb0aa68bbc68e588f03c5`
- **Verification:** 144 of 144 entries verified `OK` via `sha256sum -c`.

### 4.2 Independent QA Evidence Ledger
- **Path:** `~/almadial-lab/evidence/m3-03/qa-20261111T092620Z-alm25/`
- **Ledger File:** `SHA256SUMS`
- **Ledger SHA-256:** `c73ec34e8066853c5e74baa43a3866f52b2658d796ef77170b0414370f7547ff`
- **Contents:**
  - `2be700b09c03ba366dc54ee821b7a5b4fa61c99fbf7eb80b10329c5fb59f221f`  `baseline-preservation.json`
  - `0b1b54011822856e58f80f7b684ae9111f1eb69c7f2e88f7e899a6666e3cc8bb`  `network-isolation-acceptance.json`
  - `4e7a936867b21e95c7e18b914e335601ba58b22bdf9ce5c9db619014570bb710`  `regression-summary.json`
  - `3ea58ccc531b9fdcb70a2c759eb5df7f85501063a4ad0eaa7a7659c194b48dff`  `reproducible-builds.txt`
- **Verification:** 4 of 4 entries verified `OK` via `sha256sum -c`. Permissions set to `0700` (directory) and `0600` (files).

---

## 5. Scope Boundaries and Next Work Package

- **Boundaries Preserved:** Global IPv6 routing, public Internet reachability, provider NAT traversal, and live RTP audio streams were not qualified, consistent with the authorized single-server isolated contract. No unmanaged firewall rules or external interfaces were touched.
- **Next Canonical Work Package:** **ALM-26 / M3-04** (`Integrate pinned VICIphone and generated phone templates`), task ID `wcip8y6y5jezllppbokixauc`. It remains `to-do` and unassigned until authorized.
