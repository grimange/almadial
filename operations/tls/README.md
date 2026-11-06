# PR-08: IP certificate renewal operations

This directory is the maintained source for the existing Certbot/Apache integration
on `dialer01` (`2.29.153.161`, SSH TCP 22222). It does not manage portal grants or
telephony. Do not rerun the retired `/root/alma-portal-kit` certificate provisioning
script over these files: that historical script generates the obsolete hook.

## Installed files and ownership

| Source | Installed path | Ownership/mode |
|---|---|---|
| `alma_portal_tls.py` | `/usr/local/libexec/alma-portal-tls` | root:root 0755 |
| `alma-portal-cert-deploy` | `/usr/local/sbin/alma-portal-cert-deploy` | root:root 0755 |
| `alma-portal-renew.service` | `/etc/systemd/system/alma-portal-renew.service` | root:root 0644 |

The existing `alma-portal-renew.timer` remains enabled, persistent, twice daily at
04:15/16:15 **server local time**, with up to 600 seconds randomized delay.
Apache consumes the existing `/etc/letsencrypt/live/2.29.153.161/` lineage directly.
The renewal configuration retains the `shortlived` profile and HTTP-01 webroot
`/var/www/alma-acme`. Port 80 remains ACME-only. No TLS files are copied to the
retired Python portal backend.

## Normal operation

The timer invokes `renew`, which calls the existing Certbot 5.9.0 executable.
The deploy hook validates the chain, IP identity, lifetime, key pair and Apache
configuration, then calls **`systemctl reload httpd.service`** and verifies the
trusted certificate actually served on loopback TCP 443 with the public IP as
the verified identity. It never falls back to a restart. The wrapper verifies
deployment receipts because this installed Certbot can log a failed hook without
returning failure. Missing/failed receipts cannot turn a changed certificate or
staging exercise into success.

No-op checks validate disk/served certificate agreement without reloading Apache.
A later no-op check can retry a previous failed/missed deployment. A currently
expired certificate does not prevent Certbot being called to renew it.

Read-only operational checks:

```bash
ssh -p 22222 root@2.29.153.161 \
  '/usr/bin/python3 /usr/local/libexec/alma-portal-tls check'
ssh -p 22222 root@2.29.153.161 \
  'systemctl status alma-portal-renew.timer alma-portal-renew.service --no-pager'
ssh -p 22222 root@2.29.153.161 \
  'journalctl -u alma-portal-renew.service --since "2 days ago" --no-pager'
```

`check` returns nonzero for a recorded renewal/deployment failure, no successful
check within 30 hours, a run stuck beyond 720 seconds, less than 24 hours of valid
certificate lifetime, an untrusted/mismatching served certificate or an inactive/
disabled timer. Root-only state survives reboot under `/var/lib/alma-portal-tls/`:

- `status.json`: latest attempt, outcome, last success, dry-run/changed/deployed flags.
- `deployment.json`: latest hook attempt, run identifier, outcome and certificate.
- Lock files prevent overlapping wrapper or hook executions.

State JSON files are written by atomic replacement with mode 0600; the directory
is root:root 0700. Certbot output and structured stage messages go to the journal.
No private keys, account credentials or application passwords are logged by this
integration. Failure state should be investigated rather than manually cleared.

**Alert delivery dependency:** no approved notification recipient/destination or
functioning external monitoring integration was established. Local failed units,
persistent JSON and the executable health check are implemented and tested.
Assign an operator/approved monitoring runner to poll `check` (for example every
five minutes) and deliver nonzero results to an approved channel under PR-07.
No external alert has been sent or accepted. A vanished/stopped timer is detectable
by this check but cannot notify anyone unless that independent polling is arranged.

## Controlled exercises (not routine health checks)

Normal due-only check, using the same installed systemd service:

```bash
ssh -p 22222 root@2.29.153.161 'systemctl start alma-portal-renew.service'
```

An explicitly authorized staging exercise can run through systemd independently
of the SSH session. Choose a unique transient unit name for each exercise:

```bash
ssh -p 22222 root@2.29.153.161 \
  'systemd-run --unit=alma-portal-tls-staging-YYYYMMDD --property=Type=oneshot --property=TimeoutStartSec=720 --wait /usr/bin/python3 /usr/local/libexec/alma-portal-tls renew --dry-run'
```

The wrapper adds `--run-deploy-hooks`: a successful staging issuance executes the
real hook with the **current trusted certificate**, including an Apache graceful
reload. The staging certificate is not installed. This proves challenge issuance
and the production hook separately; it does not prove that a new production
certificate serial has been issued. Never add forced production renewal just to
make an acceptance report show a changed serial. Check CA limits before repeating.

## Deployment and rollback

Local tests: `python3 -m unittest discover -s operations/tls -p 'test_*.py' -v`.
`collect_evidence.py` is a bounded read-only collector intended for SSH stdin; it outputs
sanitized JSON including public-IP HTTP probes, never response cookies or bodies.
`deploy.py` is the initial PR-08 installer for the fixed authorized host. It refuses
to overwrite an already installed wrapper; review the diff and make a new scoped
backup before future updates. It does not represent a general AlmaDial installer.

The accepted initial rollback archive is:
`/root/alma-portal-tls-backup-20261106T203248Z/originals.tar`.
Extraction into an isolated temporary directory reproduced original hashes, owner,
group and modes. The archive and directory are root-only. No certificate private
key was copied into this archive or repository; Certbot retains its own lineage.

```bash
ssh -p 22222 root@2.29.153.161 \
  '/root/alma-portal-tls-backup-20261106T203248Z/rollback.sh'
```

Wait for any active renewal/staging run before rollback. The script checks the
scheduled renewal service and restores only the two modified pre-existing files,
removes the new executable, reloads systemd metadata and validates Apache. It does
not restore stale grants, firewall rules, renewal metadata or certificate symlinks.
It retains state for diagnosis. **The original failure-masking hook returns after
rollback**; treat rollback as incident recovery, not an accepted long-term solution.
Live rollback and full host reboot were not exercised, to preserve the working
deployment. Provider-console availability remains operator-reported.

See [acceptance report](../qa/VICIDIAL_TLS_RENEWAL_ACCEPTANCE.md) and
[PR-02 handoff](../qa/VICIDIAL_BACKUP_RECOVERY_HANDOFF.md).
