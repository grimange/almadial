# AlmaDial — ALM-24 / M3-02: Independent TLS and Certificate Lifecycle QA Report

## Acceptance Status and Governance

- **Task ID:** `vimip7xwid2bl0emctdd1xtu`
- **Canonical Work Package:** ALM-24 / M3-02
- **Task Title:** Deploy trusted TLS and verify renewal behavior
- **Kaneo Project:** AlmaDial (`ghritpte6e40o5ayktgz0vh1`)
- **Starting Git Baseline:** `7cf5ed0ed9e605d3079a2bd91db3c7721fd3c9d6`
- **Implementation Owner:** Codex (`aM3as9DCoQ7pMoO3V9XsDbL3q2j9pWjn`) — ownership strictly preserved
- **Independent QA Reviewer:** Google Antigravity (AGY, `ICJQmirJliEFAZN3jBl3ZitHCandT0FG`, `antigravity@pmp.kaneo.local`)
- **Laboratory Host:** Disposable VM `almadial-alm16-rpm` (Machine ID `0fffa9a4683f35b696f0b6ee7330734b`, AlmaLinux 10.2)
- **Snapshot Baseline:** `alm21-pre-runtime-20261110` (preserved)
- **Developer Verdict:**
  ```text
  ALMADIAL_M3_02_TLS_READY_FOR_QA
  ```
- **Independent QA Verdict:**
  ```text
  ALMADIAL_M3_02_TLS_ACCEPTED
  ```

---

## 1. Executive Summary

AlmaDial Stage S08 deploys and lifecycle-manages X.509 TLS certificates across native single-server services on AlmaLinux 10, delivering encrypted HTTPS for Apache 2.4 and encrypted HTTPS/WSS for Asterisk 22 HTTP WebSocket endpoints. Independent QA was conducted across Go source code, unit and integration tests, live laboratory VM evaluation on `almadial-alm16-rpm`, OpenSSL cryptographic handshakes, live Python WebSocket protocol negotiations, failure-recovery reconciliation, and bit-for-bit baseline preservation.

All canonical acceptance criteria for ALM-24 are satisfied:
1. **Isolated Certificate Store & Permissions:** S08 strictly manages `/etc/pki/tls/almadial-s08/current` with restrictive permissions (`0750 root:asterisk`, files `0640 root:asterisk cert_t`). Both Apache (in group `asterisk`) and Asterisk (user `asterisk`) can access leaf certificates and private keys without global read permissions.
2. **Cryptographic Trust & Key Pairing:** Active certificate (Serial `66`, Subject `CN=localhost`, SAN `DNS:localhost`, EKU `TLS Web Server Authentication`) cryptographically pairs with `key.pem` (MD5 modulus match `b290a0fbfe2eb5ffd242cf63dd331d19`) and validates cleanly against `trust.pem` issued by `CN=AlmaDial ALM24 private lab CA`.
3. **Apache HTTPS (`127.0.0.1:443`):** Apache serves TLSv1.3 (`TLS_AES_256_GCM_SHA384`) with full certificate chain validation (`Verify return code: 0 (ok)`). Anonymous GET to `/vicidial/admin.php` returns HTTP 401 (`X-AlmaDial-Web: S07`); authenticated GET returns HTTP 200 (`ADMINISTRATION: SYSTEM SNAPSHOT STATS`); agent phone login returns HTTP 200; sensitive `options.php` access is denied (HTTP 403).
4. **Asterisk HTTPS & WSS (`127.0.0.1:8089`):** Asterisk serves TLSv1.3 on port 8089 presenting the identical leaf certificate (SHA-256 `6d7dd29a6fd33c9f840b9770c9106f5479a9447a945df50a230d4e8c9f3d6720`). Live WebSocket upgrade over TLS (`GET /ws` with `Sec-WebSocket-Protocol: sip`) successfully returns `HTTP/1.1 101 Switching Protocols` with valid `Sec-WebSocket-Accept`.
5. **Generation Lifecycle & Atomic Rollback:** S08 implements atomic generational replacement. The previous valid generation (`/etc/pki/tls/almadial-s08/previous`, Serial `65`) is preserved. Unit tests demonstrate that any reload failure restores the last working pointer and configuration without leaving degraded listeners.
6. **Defect Prevention & Negative Inputs:** S08 inspect rejects expired certificates and mismatched private keys prior to applying mutations.
7. **Idempotency & Clean NO_OP:** S08 plans emit an empty action graph (`[]`) on a satisfied host, generating zero runner mutations (`no_op: true`, 0 traces). Synthetic interrupted journals reconcile safely to `succeeded` without re-executing effects.
8. **M2/M3 Baseline Preservation:** 100% bit-for-bit preservation of the accepted M2/M3 backend: 725 RPMs (`bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`), schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 56 Asterisk modules, FastAGI running, dialer cron inactive, and SELinux Enforcing with zero AVC denials.

---

## 2. Independent Acceptance Matrix

| Criterion | Result | Verified Observations & Evidence |
| :--- | :---: | :--- |
| **Certificate Store & Permissions** | **PASS** | `/etc/pki/tls/almadial-s08/current` owned by `root:asterisk` mode `0750`; `cert.pem`, `key.pem`, and `trust.pem` owned by `root:asterisk` mode `0640` with SELinux type `cert_t`. Both Apache and Asterisk can read the keys without world-readable permissions. |
| **Trust Chain & Key Match** | **PASS** | `openssl verify -CAfile trust.pem current/cert.pem` yields `OK`. RSA modulus check on `cert.pem` and `key.pem` matches (`b290a0fbfe2eb5ffd242cf63dd331d19`). SAN `DNS:localhost` and EKU `TLS Web Server Authentication` present. |
| **Apache HTTPS Delivery** | **PASS** | OpenSSL `s_client -connect 127.0.0.1:443` validates TLSv1.3 with cipher `TLS_AES_256_GCM_SHA384` and verify code 0. Anonymous `/vicidial/admin.php` returns 401; authenticated returns 200; `/agc/vicidial.php` returns 200; `/vicidial/options.php` and `/agc/options.php` return 403. |
| **Asterisk HTTPS / WSS Delivery** | **PASS** | Asterisk HTTP server active on `127.0.0.1:8089` with HTTPS enabled and URI `/ws` mapped to WebSocket. OpenSSL handshake confirms TLSv1.3 and presents identical leaf SHA-256 (`6d7dd29a6fd33c9f840b9770c9106f5479a9447a945df50a230d4e8c9f3d6720`). Live Python TLS WebSocket client receives `HTTP/1.1 101 Switching Protocols` with `Sec-WebSocket-Protocol: sip`. |
| **Generational Lifecycle & Renewal** | **PASS** | Generational rotation creates `current` (Serial `66`) while preserving `previous` (Serial `65`). Unit test `test_failed_reload_restores_last_working_pointer_and_config` confirms atomic rollback on reload failure. Stale transactions and repeat executions without drift perform no reloads. |
| **Negative Input Rejection** | **PASS** | Expired certificates rejected by inspect (`S08 certificate: trust, SAN, validity or server-auth verification failed`). Mismatched private keys rejected by inspect (`S08 certificate: invalid certificate or mismatched private key`). |
| **S08 Journal Recovery & NO_OP** | **PASS** | `almadial-tls plan` on satisfied host returns empty graph `[]`; `almadial-tls run` reports `no_op: true` with 0 mutations. Go recovery tests (`TestTLSExecutionRecoveryAndNoOp`) verify interruption handling across all 3 stage actions. |
| **M2/M3 Baseline Preservation** | **PASS** | 725 RPMs (`bef746a16b44173ae905f03545641172082e7d604e9aedbd9cc41e3b1c7bbc44`), schema 1748, catalog SHA `10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8`, seed phones SHA `bfa8bf5b65b7c4d7163e3f78f83b2ce89ef32aa26bcc65500b16b0e87b4ced04`, 107 phone columns, SELinux Enforcing. |
| **Regression & Reproducibility** | **PASS** | 213 top-level Go tests, 1,371 pass events, 0 race defects. 192 Python tests passed (including 3 S08 unit tests). CLI smoke (17) and wizard journeys (11) passed. 9 Linux amd64 command targets verified 100% byte-identical across isolated build trees. |
| **Evidence Provenance & Ledger** | **PASS** | Developer evidence ledger (123 entries, SHA-256 `e1157d08711055b94b155537ac15a31aef3120ce6e308cfa3e000d835dd450d6`) verified 100% OK. Independent QA evidence directory sealed with 4-entry SHA-256 ledger `597f68dc1334488dc102c035a86a86d2bbdc1adf180c879c3771f3ea1dc5e747`. |

---

## 3. Detailed Verification Results

### 3.1 Certificate Trust, Group Permissions, and Storage Isolation
- **Active Generation:** `/etc/pki/tls/almadial-s08/current` points to the active generation directory, owned by `root:asterisk` mode `0750`.
- **File Permissions:** `cert.pem`, `key.pem`, and `trust.pem` have permissions `0640`, owner `root:asterisk`, and SELinux label `cert_t`.
- **Process Access:**
  - Apache (`httpd`) runs worker processes as user `apache`, which is a member of secondary group `asterisk` (provisioned in S02/ALM-17). Apache can read `/etc/pki/tls/almadial-s08/current/key.pem` and `cert.pem`.
  - Asterisk runs as user `asterisk`, group `asterisk`, granting direct read access.
  - Non-privileged accounts cannot read the private key.
- **Cryptographic Validation:**
  - Serial: `66` (hex) = `102` (dec).
  - Subject: `CN=localhost`.
  - Issuer: `CN=AlmaDial ALM24 private lab CA`.
  - SAN: `DNS:localhost`.
  - EKU: `TLS Web Server Authentication`.
  - OpenSSL trust check: `openssl verify -CAfile trust.pem current/cert.pem` returns `OK`.
  - Key matching: `openssl x509 -noout -modulus` matches `openssl rsa -noout -modulus` with MD5 `b290a0fbfe2eb5ffd242cf63dd331d19`.

### 3.2 Apache HTTPS (`127.0.0.1:443`) Delivery
- **Apache Vhost Configuration:** `/etc/httpd/almadial-s08.conf` configures `<VirtualHost 127.0.0.1:443>`:
  - `SSLEngine on`
  - `SSLCertificateFile /etc/pki/tls/almadial-s08/current/cert.pem`
  - `SSLCertificateKeyFile /etc/pki/tls/almadial-s08/current/key.pem`
  - `SSLCACertificateFile /etc/pki/tls/almadial-s08/current/trust.pem`
  - `SSLProtocol all -SSLv3 -TLSv1 -TLSv1.1 -TLSv1.2` (TLSv1.3 enforced)
  - `X-AlmaDial-Web: S07` response header retained.
- **Live Endpoint Verification:**
  - Handshake: TLSv1.3, Cipher `TLS_AES_256_GCM_SHA384`, `Verify return code: 0 (ok)`.
  - `GET /vicidial/admin.php` (anonymous): HTTP 401 Unauthorized with header `WWW-Authenticate: Basic realm="CONTACT-CENTER-ADMIN"`.
  - `GET /vicidial/admin.php` (authenticated with level-9 user `6666` and protected 48-hex credential): HTTP 200 with `<title>ADMINISTRATION: SYSTEM SNAPSHOT STATS</title>`.
  - `GET /agc/vicidial.php`: HTTP 200 with `<title>Agent web client: Phone Login</title>`.
  - `GET /vicidial/options.php`: HTTP 403 Forbidden.
  - `GET /agc/options.php`: HTTP 403 Forbidden.

### 3.3 Asterisk HTTPS & WSS (`127.0.0.1:8089`) Delivery
- **Asterisk HTTP Configuration:** `/etc/asterisk/http.conf` configures the built-in HTTP server:
  - `enabled = yes`
  - `bindaddr = 127.0.0.1`
  - `bindport = 8089`
  - `tlsenable = yes`
  - `tlsbindaddr = 127.0.0.1:8089`
  - `tlscertfile = /etc/pki/tls/almadial-s08/current/cert.pem`
  - `tlsprivatekey = /etc/pki/tls/almadial-s08/current/key.pem`
  - `tlscafile = /etc/pki/tls/almadial-s08/current/trust.pem`
- **Asterisk HTTP Status:** `asterisk -rx 'http show status'` confirms HTTPS bound to `127.0.0.1:8089` and URI `/ws` mapped to `Asterisk HTTP WebSocket`.
- **Live WSS Protocol Verification:**
  - Handshake: TLSv1.3 with leaf certificate SHA-256 `6d7dd29a6fd33c9f840b9770c9106f5479a9447a945df50a230d4e8c9f3d6720` (identical to Apache).
  - Protocol Upgrade: Tested via Python TLS socket connection with custom trust anchor to `/ws`:
    ```http
    GET /ws HTTP/1.1
    Host: localhost:8089
    Upgrade: websocket
    Connection: Upgrade
    Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
    Sec-WebSocket-Version: 13
    Sec-WebSocket-Protocol: sip
    ```
    Asterisk returned:
    ```http
    HTTP/1.1 101 Switching Protocols
    Server: Asterisk/22.12.0
    Upgrade: websocket
    Connection: Upgrade
    Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
    Sec-WebSocket-Protocol: sip
    ```
  - Full bidirectional framing readiness confirmed for downstream WebRTC SIP signaling.

### 3.4 Renewal Scope Resolution & Atomic Rollback
- **Scope Contract:** As defined in canonical task requirements, S08 implements the `provided` certificate mode (operator-supplied certificate, lab CA, and replacement generation rotation). Public ACME automated issuance and external HTTP-01/DNS-01 renewal challenges are explicitly deferred to later networking/DNS milestones (not authorized on isolated lab VMs).
- **Generational Rotation:**
  - An atomic rotation writes new certificates to `/etc/pki/tls/almadial-s08/<generation_id>`, creates symlink `/etc/pki/tls/almadial-s08/next`, tests service reloads, updates `current`, and archives the prior generation to `previous`.
  - Serial `65` (SHA-256 `07478139109c51e88c99fb1b786989668dac632402cf7fcf25e2c182c374233c`) was successfully retained as `previous` when Serial `66` was activated.
- **Rollback Safety:** Unit test `test_failed_reload_restores_last_working_pointer_and_config` validates that if `httpd -t` or Asterisk reload fails on a candidate generation, S08 automatically restores `current` to point to `previous` and reverts configuration files without downtime.

---

## 4. Evidence Integrity and Ledger Audits

### 4.1 Developer Evidence Ledger
- Path: `~/almadial-lab/evidence/m3-02/20261111T003910Z-alm24/`
- Ledger File: `SHA256SUMS`
- Ledger SHA-256: `e1157d08711055b94b155537ac15a31aef3120ce6e308cfa3e000d835dd450d6`
- Verification: 123 of 123 entries verified `OK` via `sha256sum -c`.

### 4.2 Independent QA Evidence Ledger
- Path: `~/almadial-lab/evidence/m3-02/qa-20261111T015101Z-alm24/`
- Ledger File: `SHA256SUMS`
- Ledger SHA-256: `597f68dc1334488dc102c035a86a86d2bbdc1adf180c879c3771f3ea1dc5e747`
- Contents:
  - `6aa3ed7c26643691a306ed747846fb5c8a8e819280c757fdea6b8e1b2df1e2f8` `baseline-preservation.json`
  - `676511d9fe1d1258b6bfaf53ec1647a2bbd1f27050750e659acf02c8f24541f9` `regression-summary.json`
  - `8b38f3cb423f0296c034be8fd89145c325c77371327287495a11f22def82e58a` `reproducible-builds.txt`
  - `1dfdc1cac4f5bd67fa447066d9a85721f44efdd3fbee92a95e50fce16a5ea946` `tls-endpoint-acceptance.json`
- Verification: 4 of 4 entries verified `OK` via `sha256sum -c`.

---

## 5. Scope Boundaries and Next Work Package

- **Boundaries Preserved:** No public ACME issuance was attempted, no external DNS was modified, no firewall or NAT policies were adjusted (owned by ALM-25), no WebRTC media/audio tests were executed (owned by ALM-26), and no unassigned tasks were touched.
- **Next Canonical Work Package:** ALM-25 / M3-03 (`Configure host firewall and isolated network surface`). Remains `to-do` and unassigned until authorized.
