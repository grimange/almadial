# M0-04 — VICIdial r4037 application runtime qualification

Implementation evidence, 2026-11-05 UTC / 2026-11-06 Asia/Manila. Codex is the
implementer; independent acceptance is pending. This is a bounded application
experiment, not a complete dialer installation or whole-stack acceptance.

## Identity, authority and recovery

- Task ALM-4 / M0-04: `cf87vzpxxjnuchzwl19pdb6a`, project
  `ghritpte6e40o5ayktgz0vh1`. ALM-3 was read back as Accepted.
- Repository input: `4e2abd5281990949e4479230a4a4dcd0478e796e`.
  The pre-existing untracked `scripts/verify_agy_overlay.py` is untouched.
- The operator's ALM-4 assignment authorizes application dependencies, isolated
  database/application tests and protected evidence on **almadial-alma10 only**.
  It explicitly does not waive the VICIdial source-trust gate.
- AlmaLinux 10.2 x86_64, kernel `6.12.0-211.64.1.el10_2.x86_64`, Multipass
  1.17.0/QEMU, 4 vCPU / 8 GiB / 80 GiB. Private addresses remain outside Git.
- Snapshot **almadial-alma10.pre-vicidial-r4037**, created/read back
  `2026-11-05T20:15:54Z`, parent `pre-asterisk22-build`. Multipass required a
  stop/start before snapshot creation. Existing snapshots were preserved.
  No restoration was performed. Asterisk was started after the guest restart;
  its accepted service was not configured to start automatically at boot.

Separately authorized recovery, only after preserving new evidence:

```bash
multipass stop almadial-alma10
multipass restore almadial-alma10.pre-vicidial-r4037
multipass start almadial-alma10
```

These commands are recovery instructions, **NOT_RUN** here.

## Source trust: resolved for this exact artifact

The official [VICIdial wiki](https://wiki.vicidial.org/doku.php) links the
[official nightly directory](https://www.vicidial.org/svn_trunk_nightly/).
The [2026-11-05 archive](https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip)
was retrieved with normal HTTPS hostname/CA validation. Its complete extracted
file set matches the previously pinned SVN r4037 export: **2,511 files, zero
missing, extra or changed files**. The schema marker is 1748.

| Artifact | SHA-256 |
|---|---|
| Authenticated nightly ZIP | `ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058` |
| Canonical exported tree | `04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed` |
| Original schema SQL | `8a210454a6f42fd3ef0054033a5c4396a0cd3615442ebd53491dfd2efa38de40` |
| Original first-server seed | `c215f549c3a6dc39e20224e666efd6d99aa85b93910fe8dbaf7daa62a962203e` |
| Original install.pl | `8ee133ad3f324b544f5d07b87e974433475a3498bef09174c656ec99de56609f` |

Decision: the official authenticated HTTPS origin plus full byte equality supplies
the missing publisher-origin chain for the exact candidate. This permits the
bounded execution authorized here. It is **not** a detached signature, signed SVN
revision attestation or Web-of-Trust endorsement. The original SVN transport
remains unauthenticated. No operator override was invented or required.

Canonical tree hashing uses `verify_candidate_stack.tree_digest`: sorted Python
Path entries, file SHA-256 followed by two spaces, relative POSIX path and newline;
SHA-256 of that ledger. An earlier diagnostic used string-path ordering and
produced a different aggregate; `source-trust-decision.json` records the correction.
The final guest check reconfirms all original source bytes are unchanged.

The manifest retains `status: candidate`, the original SVN identity and all version
pins; only VICIdial's current provenance decision/archive reference is added.
M0-02's historical hold remains in its original evidence, superseded for this
artifact by this decision. Whole-stack `verification.runtime_state: NOT_RUN`
remains the aggregate gate, despite the scoped runtime observations below.

Upstream `docs/LICENSE.txt` identifies **Affero General Public License version 2,
November 2007** (SHA-256
`5f27d58d7bbfb242781ff69aaf82694d7e3e05bd55abf87ed1f4e84205f6eeaf`).
Preserve its notices and review applicable network-use/distribution obligations
before packaging. No upstream application source is added to AlmaDial's Git tree;
do not relabel the entire tree AGPLv3 based on VICIphone's separate license.

## Installed candidates and isolation

| Runtime | Observed package/version |
|---|---|
| MariaDB server/client/backup | `3:10.11.18-1.el10_2.alma.2.x86_64` |
| Apache | `2.4.63-13.el10_2.8.x86_64` |
| PHP CLI/FPM/common/mysqlnd/GD/mbstring/XML/LDAP/opcache | `8.3.36-1.el10_2.x86_64` |
| Perl | `4:5.40.2-515.el10_2.x86_64` |
| DBI / DBD::mysql | `1.643-26.el10_2.7` / `5.007-4.el10` |
| Net::Server / Time::HiRes | `2.014-9.el10` / `1.9777-511.el10` |

Approved AlmaLinux repositories supplied these packages. Exact NEVRAs and
transactions are in `app-package-install.json` and `app-package-query.json`.
No candidate version was silently replaced.

MariaDB is socket-only (`skip-networking` in a new lab-owned
`/etc/my.cnf.d/almadial-m004.cnf`); no TCP 3306 listener. Apache listens only at
`127.0.0.1:8080`, with PHP-FPM over its distribution socket. Its original config
is backed up. This isolated HTTP experiment does not replace or downgrade the
accepted Asterisk HTTPS/WSS certificate path. AMI is not publicly exposed.

`/etc/astguiclient.conf` is lab-owned, root:apache mode 0640; credentials are
random, protected and excluded from Git. The application account is localhost-only
and has schema-local SELECT/INSERT/UPDATE/DELETE/LOCK TABLES, no global RELOAD.
Active administrator and first-setup passwords were generated and rotated.
Inactive VDAD/VDCL seed identities remain disabled. One diagnostic response
included an ephemeral password; it was rotated and its raw evidence is private.
Privilege inspection also captured a test password hash; that credential was
rotated before closure. Do not export raw response bodies or grant output.

SELinux remains **Enforcing**. Web files have distribution labels via `restorecon`;
no broad policy exception, network-connect boolean or security disablement was
needed for local socket access. The inspected nftables ruleset remains empty;
no firewall/SSH rules changed. Apache/PHP-FPM configuration checks exit 0; Apache
reports a non-fatal missing global ServerName warning.

## MariaDB schema and ownership

The first import into `almadial_m004` failed at original SQL line 5817:
ERROR 1133, GRANT RELOAD to nonexistent `cron@'%'`. It left 345 tables and schema
marker **0**. That partial database remains preserved; it was not dropped or
silently replayed.

The reviewed derivation removes only the two global `cron` grants and
`flush privileges;`, replacing each with an ownership comment. Schema definitions,
seed statements and schema-version update remain byte-for-byte otherwise.
**AlmaDial's future bootstrap is the one initialization owner**, applying the
approved upstream schema and explicitly provisioning scoped accounts separately.
Upstream SQL is authoritative for definitions; there is no competing schema generator.

Derived SQL SHA-256:
`0992448a2607dbad6841250a09b68dafd360cb4fa44f8589d89526ef16522af8`.
It was imported once into separate database `almadial_m004_v2`, explicitly
`utf8mb3 / utf8mb3_unicode_ci`, followed by unmodified `first_server_install.sql`.
Both exit 0. Observed: **schema 1748; 345 tables; 337 MyISAM / 8 MEMORY;
1,011 information_schema.statistics rows**; initial user/server seeds recognized.
These are index-column rows, not 1,011 distinct indexes. No foreign-key conversion
was introduced; these two observed engines do not provide InnoDB foreign-key enforcement.

Global SQL mode was not relaxed:
`STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION`.
The server default remains latin1; the application schema uses explicit upstream
utf8mb3. Its datetime/timestamp definitions imported without a fatal compatibility
error, but all date-boundary application behavior is not qualified.

Import notes are retained:

- Two Code 1071 notes: MyISAM's 1,000-byte key limit yields effective
  `www_phrases.phrase_text(333)` and
  `vicidial_language_phrases.english_text(333)` prefix indexes. Both are nonunique;
  table creation succeeds. Do not mistake them for full 10,000-character indexes.
  Translation/search performance is unqualified; no silent DDL rewrite was applied.
- Code 1831: duplicate index `vdlga` is deprecated. It is accepted by this exact
  MariaDB candidate; future database-version changes must revalidate it.

`mariadb-check --check --quick` exits 0: **337 tables OK**; eight MEMORY tables
explicitly do not support CHECK. This is not a claim that all 345 engines passed
the same physical-integrity check.

A deliberate repeat import exits 1 immediately with ERROR 1050 (`phones` already
exists). Recovery must classify before mutation:

| Observed database | Required behavior |
|---|---|
| No tables | Eligible for authorized, single initialization |
| Tables present, marker absent/0 or contract incomplete | Stop as partial; preserve evidence; no blind replay/drop |
| Marker 1748 plus expected tables/seed/index contract | Verify/reuse, do not replay CREATE statements |
| Other marker or unknown existing tables | Stop for an explicit migration/ownership decision |

A version marker alone does not prove completeness. MyISAM imports are not an
atomic transaction; a future installer must journal the boundary and reconcile
actual schema state.

## PHP: static checks and actual application execution

**307/307** pinned `www/**/*.php` files pass `php -l`. CLI/FPM load MySQLi,
PDO MySQL, mbstring, curl (provided by php-common), XML, GD, LDAP and session support.
Apache/FPM execute the pinned files, not a mock HTTP application.

The first admin requests failed with PHP 8 `fwrite(false)` TypeError when the
upstream authentication path expected a writable marker in an immutable webroot.
Resolution uses existing upstream configuration, without changing PHP source:
`system_settings.webroot_writable=0`, protected `old_clear` marker and
`options.php` with `$use_slave_server=0` in admin and agent trees. Code is readable
by Apache but not writable by it. Do not copy upstream world-writable permissions.

First-login setup also matters: an initial HTTP 200 returned the setup page rather
than performing a requested admin write. The final run executes the real first-run
setup, verifies `first_login_trigger=N`, and checks database effects after requests.

| Executed case | Result / corroboration |
|---|---|
| Anonymous admin | HTTP 401 |
| Authenticated first-run setup and administrator page | HTTP 200; first-login trigger cleared; generated passwords |
| Admin creates and reads `m004qa` user | HTTP 200; database row count 1 and expected full name |
| Agent campaign lookup (`LogiNCamPaigns`) | HTTP 200; upstream application response |
| Agent favorites write without a session | HTTP 200 **with rejection**, not a success claim |
| Seeded non-telephony application session | Real `vdc_db_query.php` accepts favorites `9001`; database readback agrees |
| Session expiration | Named test session deleted; second write to `9002` rejected; database remains `9001` |
| After credential rotation | Authenticated admin HTTP 200; corrected Perl sender still connects |

The session is explicitly **seeded in web_client_sessions**, exercising real
application session validation/read/write. It is not a complete browser agent
phone login, conference join or dialer session. Those integration cases remain for
M0-05 onward. HTTP status alone was never used as proof of a write.

The final nine-request run has **0 new PHP fatals and 26 warnings**, principally
undefined admin variables at r4037 admin.php lines 4836, 4858, 5079–5080, 5540 and
one null-array-offset warning near 54268. They did not prevent the measured
workflows; unrelated optional pages are unqualified. Warnings were preserved,
not globally suppressed or patched speculatively.

The exercised entry points require `dbconnect_mysqli.php` (admin.php:10,
vicidial.php:776, vdc_db_query.php:589). Historical mysql_connect references in
the inspected admin paths are commented out. This disproves an application-wide
failure inferred merely from that string; it does not prove every legacy/optional
page is PHP 8 compatible. Invalid-login output can echo an entered password;
requests/bodies must remain private and this behavior needs security review
before operator-facing deployment.

## Perl modules, syntax and bounded runtime

Runtime module loads pass for DBI 1.643, DBD::mysql 5.007, Net::Telnet 3.05,
Net::Server 2.014, Time::HiRes 1.9777, Switch 2.17 and
Crypt::Eksblowfish::Bcrypt 0.009. Upstream `Asterisk::AGI` loads with the pinned
`extras` include path; its file SHA-256 is
`38f4e84f50e93e53d63b279584f5fca04553528c7660022d4b082aac30314c9a`.
Core modules (Getopt::Long, POSIX, Scalar::Util, Time::Local, Term::ANSIColor,
strict/vars) load through the representative syntax checks.

Net::Telnet came from authenticated, pinned CPAN 3.05 archive SHA-256
`677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743`.
The initial unversioned site path did not load on this Perl; using the runtime's
actual `/usr/local/share/perl5/5.40` site path resolved it. That failed observation
is retained. No unverified CPAN installer was executed as root.

Missing Switch and Crypt::Eksblowfish were installed from a **temporary EPEL 10.2
repository definition**, with package signature checking; EPEL was not persistently
enabled. Selected packages: `perl-Switch-2.17-33.el10_1`,
`perl-Crypt-Eksblowfish-0.009-42.el10_0`, `perl-Class-Mix-0.006-22.el10_0` plus
Devel::CallChecker, DynaLoader::Functions and Params::Classify dependencies.
The official EPEL key fingerprint is
`7D8D15CBFC4E62688591FB2633D98517E37ED158`; direct signature verification of the
downloaded Switch/Crypt RPMs reports OK. The initial missing-Class::Mix transaction
failure is retained, followed by the successful dependency-resolved transaction.

Nine original scripts pass `perl -I SOURCE/extras -c`:
ADMIN_keepalive_ALL, AST_update, AST_update_AMI2, AST_manager_listen,
AST_manager_send, AST_VDauto_dial, AST_VDadapt, AST_conf_update and FastAGI_log.
Syntax validation can execute Perl BEGIN blocks and was performed **after** source
authentication. It is not daemon runtime acceptance.

Both original and corrected **AST_manager_send.pl actually ran** for bounded
1.5-second intervals against MariaDB, polling an empty NEW/QUEUE queue and exiting
0 on controlled SIGTERM. Queue counts remained zero. No AMI action, call or trunk
was created. Keepalive/autodial/conference daemons were not started; their writer
and telephony effects require the next integration experiment.

## Reproduced procps defect and reproducible derivations

The exact legacy command exits 0 but truncates a controlled long argv before the
target script name:

```bash
/bin/ps -o "%p %a" --no-headers -A
/bin/ps -o pid,args -ww --no-headers -A
```

The first hides the target; the second preserves it. This can cause duplicate job
starts or mistaken liveness checks. The smallest correction replaces that exact
command in four pinned files (six total occurrences). Originals remain intact.

| Derived file | SHA-256 |
|---|---|
| ADMIN_keepalive_ALL.pl (3 replacements) | `51abc2f7e65f8e6722e218b9714d7142e2cb76c1fba755d8a42e195a9fbf555a` |
| AST_manager_send.pl | `007fc3012ea5c37e5c5a330719ed51a621aa9ad84488ade3ef13eac9d64756d3` |
| AST_VDauto_dial.pl | `458e6fe7ad864760688e73d3031fe5de0d28e1fae6a31bbdabfc5448542f3d3e` |
| AST_VDremote_agents.pl | `c6093d1551290b283e7cae7bbe8ef684e4e93298c2b1880e79b758083eb6b173` |

[prepare_vicidial_r4037.py](../../scripts/prepare_vicidial_r4037.py) checks exact
source-file SHA-256 and occurrence counts before creating an exclusive protected
output directory. It derives only those four files and the privilege-separated
SQL, emits source/derived hashes, never executes anything and rejects already
modified input. All four derived Perl files pass live syntax validation; the
corrected sender's real database polling also passes.

```bash
python3 scripts/prepare_vicidial_r4037.py \
  --source-dir /path/to/authenticated-r4037 \
  --output-dir /new/protected/r4037-derived
```

Paths are illustrative; the actual protected run is identified below. Five fixture
tests cover unknown/repatched bytes, exact process replacement, preservation of
schema statements, occurrence-count refusal, exclusive outputs, modes and source
symlink rejection. They are not substitutes for the recorded live reproductions.

## Upstream installer effects and configuration ownership

`install.pl` was inspected, **not executed**. Its direct effect categories are:

| r4037 lines | Effects / future installation policy |
|---|---|
| 1382–1761, 2145, 2184 | Create configured home/log/AGI/web/sound/recording trees; legacy ifconfig discovery. Validate paths; no implicit ownership of foreign trees. |
| 2650–2759 | Read expected schema marker and overwrite astguiclient.conf, losing comments. It does not execute the schema import. AlmaDial owns protected configuration and the single explicit DB bootstrap. |
| 2762–2826 | Create libs, inbound/preprocess/VTIGER/update/FastAGI queues, recording staging/archive/codec/FTP and log archive trees; broad 0766 permissions. Replace with scoped service ownership, not blanket writable trees. |
| 2829–2857 | Force-copy bin/libs/Asterisk modules, experimental FastAGI/handlers, schema and AGI; chmod; optional Unicode backup/sed changes. Pin inputs and inventory derived changes before installation. |
| 2861–2880 | Copy sounds, create TTS/temp dirs; copy legacy i386 ip_relay, symlink into system paths and start it. This experiment did none of those optional relay actions. Native architecture/provenance gate required before any future use. |
| 2886–2930 | Copy web tree/robots files, save/restore custom.css; create/truncate auth files; remove legacy listloader files; world-writable 0777 paths. Do not execute wholesale; preserve operator assets and use supported nonwritable-webroot mode. |
| 2948–2960 | sed-rewrite config paths in bin/AGI/web connectors/loaders. Prefer explicit path ownership and version-specific derived inputs. |
| 2966–3042 | Force-copy version-selected base Asterisk configs, generic MeetMe/MOH/logger/DNS/features and truncate generated placeholders. Would clobber ALM-3; prohibited in this experiment. |
| 3049–3123 | DB connection, pwd/SVN discovery, updates to servers.svn_revision/svn_info and system_settings.svn_revision. Exported ZIP has no SVN working-copy metadata; identity must be recorded explicitly, not inferred as revision 0. |

The Asterisk 18/20/22 branch selects extensions sample-18, manager sample-18,
confbridge sample-18, PJSIP/wizard sample-16 and older IAX/SIP/voicemail samples.
This is source-level support, not proof those defaults safely merge into ALM-3.
It even copies MeetMe defaults; ConfBridge remains the required integration path.

`ADMIN_keepalive_ALL.pl` is also a **configuration writer**, not just a watchdog:
lines 5557–5565 create BUILD files for extensions/IAX/SIP/PJSIP/PJSIP-wizard/
voicemail/MOH/MeetMe/ConfBridge; lines 5691–5716 create missing placeholders;
5720–5870 compare, publish, reload and delete BUILD files. It can launch additional
jobs/relay processes. Neither keepalive nor an upstream installer was started.
`generate_vicidial_conf=N` and `rebuild_conf_files=N` are confirmed at closure.

| Ownership | Required boundary for M0-05 |
|---|---|
| Accepted native Asterisk | Preserve existing base configs, TLS keys/cert, endpoint 9001, transport and Echo context |
| AlmaDial future templates | Own explicitly declared static includes/service/credential paths; backup and verify before changing |
| VICIdial generator | Own only declared `*-vicidial.conf` and BUILD/publish paths, including pjsip_wizard and confbridge |
| Operator | Keep separate overrides/custom assets; no force-copy or sed over foreign files |

Required includes: extensions-vicidial.conf, pjsip-vicidial.conf,
pjsip_wizard-vicidial.conf and confbridge-vicidial.conf. Upstream install.pl's
four generic placeholders do not cover all PJSIP/ConfBridge includes; M0-05 must
verify explicit creation/order before a reload. `res_pjsip_config_wizard` must be
loaded before generated wizard endpoints are evaluated.

AMI consumers AST_update/AST_update_AMI2/AST_manager_listen/AST_conf_update use
server DB settings and default port 5038; the sender consumes vicidial_manager.
M0-05 must define unique local-only AMI credentials, least privilege, matching DB
server identity and events before starting them. None were exposed/configured here.
FastAGI_log uses Net::Server::PreFork and local 4577; the dialplan references that
listener. Verify its bind address/DB grants/AGI path before starting call flows.
Start order: database/schema/accounts → application/server/phone settings →
owned config/includes → Asterisk/module/reload verification → bounded AMI/AGI
consumers → watchdog and dialer jobs only with explicit integration acceptance.

Final preservation checks: all **10 originally inventoried top-level Asterisk
configuration hashes unchanged**, installed binary unchanged
(`db5a50e40b6ca9a89a317d32783b1e50421e15912672956855624a29bb986d2c`),
service active, endpoint 9001 and HTTP/WSS listeners still present, zero calls.
The recursive final inventory additionally hashes existing certificate/key files;
they were not in the original top-level-only hash list. The browser Echo test was
**not repeated**, so this assignment claims file/runtime preservation, not a new
end-to-end WebRTC PASS.

## Protected evidence, commands and QA reproduction

Host evidence: `~/almadial-lab/evidence/m0-04/20261105T201527Z/`, directories 0700,
files 0600. Guest evidence is root-protected. Raw bodies, logs, database identities
and credentials remain outside Git; exported archives omit credentials.json and
the source tree, but still contain sensitive diagnostic bodies. Keep them private.

| Evidence artifact | SHA-256 |
|---|---|
| guest-evidence.tar.gz | `5e7075062b9da9590c83c3434cde77c21333a8b305832fdb584c82d73a5e25c1` |
| closure-evidence.tar.gz | `40b4d3df0ff6e1e2698a5cbfdb70b009bfaac35f14204738f1937f79bb413eed` |

Both transfer digests match the guest originals. The protected ledger includes
source authentication/downloads, snapshots, package transactions, failed first
attempts, final results and reproducible command drivers. Important guest records:
schema-import-original/v2, schema-observation, schema-rerun-refusal, php-lint-all,
web-flow-results, php-flow-errors.log, perl-sender-runtime,
procps-truncation-reproduction, derived-syntax-*, source-final-check,
schema-final-details, final-services, post-rotation-db and db-integrity.
Driver files record exact argv/UTC/exit codes; generated passwords travel through
private files/stdin/HTTP headers, not command-line arguments.

Independent QA should verify the ledger/archive hashes, source trust chain and
derivation; query schema/index/seed/privileges without printing password hashes;
repeat read-only PHP/Perl checks and scoped application requests using private
credentials; create a uniquely named test session/user when testing mutations.
Do not blindly rerun the historical driver labels or schema imports: they deliberately
refuse overwriting evidence, and imports are not idempotent. Preserve original
failures and create a separate QA evidence directory. Do not run keepalive/autodial
or overwrite Asterisk to reproduce syntax/module checks.

Independent reviewer acceptance, full telephony agent login, AMI/AGI
interoperability, generated PJSIP/ConfBridge behavior, dialing and recording remain
separate downstream gates. This report does not accept the complete stack.

### Reproduction boundaries

The protected command records include exact package-install argv and resolved RPM
identities. The protected `guest-base.py`, `guest-app.py`, `guest-checks.py`,
`guest-flows-v2.py`, `guest-final.py` and `guest-closure.py` preserve the manual
configuration/import/request sequence. They are evidence drivers for this unique
lab run, **not idempotent installers**. Review them before reproducing on a separate
authorized database or snapshot; do not replay their historical output destinations.
Do not print their credential files or raw HTTP bodies.

The reusable, repository-owned preparation step is:

```bash
python3 scripts/prepare_vicidial_r4037.py \
  --source-dir /protected/path/to/verified-r4037 \
  --output-dir /protected/new-derived-r4037
```

It verifies all five pinned input hashes before writing, refuses an existing
destination and produces `derivation.json`, the derived SQL and four derived Perl
files. It never imports SQL, starts services or runs upstream code. The manifest
consistency checker now permits only the exact documented M0-04 provenance
transition; original M0-02 evidence is unchanged. This check validates identity
consistency, not a new HTTPS download or independent acceptance.

### Repository validation and handoff

Executed validation:

| Command | Exit / result |
|---|---|
| `python3 scripts/validate_scaffold.py` | 0; scaffold PASS |
| `python3 -m unittest discover -s scripts/tests -v` | 0; **56 tests PASS** |
| `python3 -m py_compile scripts/prepare_vicidial_r4037.py scripts/tests/test_prepare_vicidial_r4037.py scripts/verify_candidate_stack.py scripts/tests/test_verify_candidate_stack.py` | 0 |
| `python3 scripts/verify_candidate_stack.py` | 0; LOCAL candidate consistency PASS, no new live/source/runtime claim |
| `git diff --check` | 0 |

The initially reproduced candidate-validator failure was an exact comparison with
M0-02's historical unauthenticated provenance state. Its bounded correction admits
only the seven exact M0-04 provenance fields, while retaining all version, source,
package and candidate-state checks. Regression cases cover field tampering and
the unchanged historical state.

Protected `pre-handoff-ledger.json` covers **78 evidence files**, SHA-256
`b9da84757071537f846cb5fc5b3417689662c190bf2b77c72ac5ea7198108f43`.
Five copied driver files initially had 0664 permissions inside the protected 0700
directory; they were restricted to 0600 before this ledger. All evidence
directories/files then passed 0700/0600 checks. Their contents were unchanged.
Final MCP evidence is separately preserved in `kaneo-final.json`, after this ledger
checkpoint; it must not be mistaken for a file covered by the earlier checkpoint.

Kaneo evidence comment **ew8m1c86hnxa8dl43pwx7dar** was read back and matched
exactly. ALM-4 was moved to **QA Review (`in-review`)** and independently retrieved:
Codex ownership, project identity and original description unchanged. Both native
ALM-3 → ALM-4 and ALM-4 → ALM-5 relations remain unchanged; ALM-3 is Accepted and
ALM-5 remains Backlog/unassigned. No task was accepted by this implementer.

Implementation verdict: `ALMADIAL_M0_04_VICIDIAL_RUNTIME_READY_FOR_QA`.
Next assignment is separate independent ALM-4 source/schema/PHP/Perl evidence QA,
preserving Codex ownership. No commit, push, downstream task or new browser Echo
test was performed.
