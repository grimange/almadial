# AlmaLinux 10.3 independent operator handoff

Use the public prerelease [AlmaDial alm34-alma103-rc1](https://github.com/grimange/almadial/releases/tag/alm34-alma103-rc1).
This candidate passed one complete developer installation and bounded native
agent/media/recording, reboot and healthy NO_OP checks on a controlled network.
Independent human installation and AGY review remain pending; public VPS
installation is not yet accepted. No developer checkout, private evidence directory
or private artifact service is needed to follow this handoff.

## Prepare a separate pristine host

The owner should provide a fresh official **AlmaLinux 10.3 x86_64** host dedicated
to independent acceptance. If no suitable pristine 10.3 operator VM is available,
create one from an official signed image and verify its identity before the run.
Do not reuse the developer-qualified installation or upgrade a historical 10.2
operator VM in place as a substitute for a pristine host. Codex has not modified
the independent operator VM or performed this human acceptance run.

- Qualified baseline: **4 vCPU, 8 GiB RAM, 80 GiB disk**. Enforced minimums: 4 CPUs
  with SSE2, 7 GiB total / 4 GiB available RAM, 60 GiB free on each checked
  installation filesystem, and recording reserve of 1 GiB plus 10,000 free inodes.
- Use an authorized root shell, working SSH and synchronized time. A single
  effective SSH port, including a nonstandard port such as 2222, is supported.
  Backups/provider recovery are recommended at the operator's discretion.
- Prepare an FQDN resolving to the server's directly assigned IPv4 address, the
  corresponding local interface, and reachable HTTPS/WSS plus the selected UDP
  media range. Public-to-private NAT mappings, TURN and arbitrary cloud topologies
  are outside this contract. Configured addresses alone do not prove reachability.
- Prepare the matching trusted full-chain certificate, private key and issuing
  CA trust as root-owned regular PEM files with no symlink path components.
  Protect the private key with mode 0600. See the
  [certificate preparation instructions](almadial-installation.md#prepare-application-tls).
  The browser must trust the issuing CA. AlmaDial does not obtain ACME certificates.
- Keep SELinux enforcing and package/certificate verification enabled. Official
  BaseOS/AppStream/CRB/extras repository definitions must be enabled. Allow DNS,
  time synchronization and outbound HTTPS to GitHub/release assets, signed
  repositories and the pinned upstream sources. Permit outbound **TCP 3690** for
  authoritative VICIdial SVN revision retrieval where needed. The restricted EPEL
  Perl dependency source is selected by the approved package policy.

## Download and install

Connect as root, or use the authorized cloud account followed by `sudo -i`.
Use your existing SSH port; for a nonstandard port add `-p YOUR_PORT` to SSH.

```bash
ssh root@YOUR_SERVER

curl -fsSLO \
  https://github.com/grimange/almadial/releases/download/alm34-alma103-rc1/install.sh

printf '%s  %s\n' \
  5c7cd19846ef3b55f7860645f7452a55c2cb15877ec3a86406fa13a4e313e621 \
  install.sh | sha256sum -c -

# Inspect the verified script, then execute it.
bash install.sh
almadial --version

printf '%s  %s\n' \
  a6c38cbd76debf45eeb9a37e03209c14a9786d98e732bff8b8ed2de42746ce92 \
  /usr/local/sbin/almadial | sha256sum -c -

almadial install
```

Stop if either checksum fails. Expected version is `almadial alm34-alma103-rc1`,
source commit `4eb5833b62400ae1c5001087f6dfe81e1e386dcc`, target `linux/amd64`,
Go `go1.27.3`. No Go compiler or Git installation is needed on the server.
No `ALMADIAL_BINARY_URL`, `ALMADIAL_ARTIFACT_STORE` or laboratory CA is needed for
the standard GitHub distribution. Application TLS remains a separate prerequisite.

Select **remote**, then supply the FQDN, directly assigned address/interface,
certificate/key/CA paths and UDP media range. The installer discovers the current
SSH management port and proposes a valid address/interface pair when unambiguous.
It verifies pinned public sources and the exact signed RPM transaction before
showing the immutable plan. Review it and enter `yes` once. Blank, `no`, EOF and
cancel reject provisioning. No snapshot identifier, Kaneo reference, recovery
assertion or manual plan hash copying is required.

## Health and supported resume

After completion, use the paths reported by the installer. Default guided paths:

```bash
almadial status --plan /var/lib/almadial-inputs/plan.json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
almadial doctor --plan /var/lib/almadial-inputs/plan.json --ca-file /etc/pki/tls/almadial-s08/current/trust.pem --manifest /var/lib/almadial-inputs/stack.lock.yaml
```

For a supported interruption, preserve saved inputs, TLS files and journals;
restore the reported prerequisite and run:

```bash
almadial resume
```

The saved exact plan is reviewed again and actual state reconciled. Rerunning
`almadial install` also offers the saved-plan path. Never edit digests, delete
journals or substitute newer sources/packages. Stop at a reported recovery boundary
that requires owner intervention. Complete system rollback is not guaranteed.

## Bounded independent acceptance checklist

1. Record pristine OS/resources, CLI version/hash, selected SSH port/interface,
   actual approval and installation outcome. Confirm ordinary SSH remains usable.
   Preserve failures and any assistance; undocumented repair prevents an
   unassisted acceptance claim.
2. Require settled Status/Doctor exit 0 and inspect any warnings separately from
   blocking failures. Brief FastAGI worker-pool expansion after a call has caused
   strict idle-count failures in developer tests; retain the observation and
   verify settled health rather than suppressing a failure.
3. Open `https://YOUR_FQDN/vicidial/admin.php` and
   `https://YOUR_FQDN/agc/vicidial.php` with normal browser certificate validation.
   Obtain generated credentials only through a private server session as described
   in the [installation guide](almadial-installation.md#guided-configuration-and-completion).
4. Configure one ordinary synthetic agent/phone/campaign/lead using the installed
   WebRTC template. Verify embedded phone registration, agent conference entry,
   one internal synthetic customer call, answer, hangup, disposition and logout.
   Do not place a carrier/PSTN call as part of this bounded test.
5. Verify meaningful audio in both directions and a correlated non-silent native
   recording with the expected database record. Registration alone is insufficient.
   Preserve sanitized observations; keep credentials, private keys and recordings
   out of public issues.
6. Reboot through the owner's authorized host procedure. Reconnect on the same
   SSH port and repeat Status/Doctor, trusted web access and service health checks.
   Ensure the hostname remains resolvable after reboot.
7. Run `almadial install` again, review the same saved plan, and require healthy
   `NO_OP` / `no_op: true` with exit 0. Record the exit status and unchanged plan
   identity. Submit evidence for independent AGY review and owner acceptance.

ALM-34 remains In Progress under Codex until independent acceptance. Publication
does not accept ALM-34, reopen ALM-46 or start ALM-35.
