# M0-05: configuration ownership contract

Date: 2026-11-05 UTC. Task: ALM-5 / M0-05, `z6mn0h4rh66ip9hh7hh769eu`.
Implementation evidence awaits independent QA. This is the r4037/Asterisk 22.12.0
lab contract; the whole stack manifest remains CANDIDATE.

## Authority and write boundaries

| Files | Authoritative writer | Contract |
|---|---|---|
| `asterisk.conf`, `modules.conf`, `http.conf`, `rtp.conf` | AlmaDial base stage | Preserve native paths, TLS, WSS and required modules; generator cannot replace them. |
| `pjsip.conf` | AlmaDial base stage | Own transports and native 9001 endpoint; include `pjsip-vicidial.conf` once. |
| `pjsip_wizard.conf` | AlmaDial base stage | Include `pjsip_wizard-vicidial.conf` once; do not duplicate wizard names in base endpoint objects. |
| `extensions.conf` | AlmaDial base stage | Own globals/include declaration and allowed context graph. Preserve `alm3-echo`; expose only qualified generated phone context through `alm5-local`. |
| `confbridge.conf` | AlmaDial base stage | Include `confbridge-vicidial.conf` once; keep native default profiles. |
| `manager.conf` | AlmaDial local AMI stage | Loopback-only listener and private credentials. No generator ownership demonstrated. |
| `pjsip-vicidial.conf`, `pjsip_wizard-vicidial.conf`, `extensions-vicidial.conf`, `confbridge-vicidial.conf` | VICIdial generator | DB is input authority. Publish verified output without editing it; never put operator overrides here. |
| `iax-vicidial.conf`, `sip-vicidial.conf`, `meetme-vicidial.conf`, `musiconhold-vicidial.conf` | VICIdial generator | Generated in staging; not included/activated by this experiment. SIP/IAX trunks and MeetMe remain blocked. |
| **`voicemail.conf`** | **VICIdial generator** | Important exception to suffix model: reads existing header before `[default]`, then replaces this unsuffixed file. No competing AlmaDial writer. Preserve reviewed header as generator input. |
| `BUILD*-vicidial.conf` | VICIdial temporary generation | Eight removed on success; `BUILDvoicemail-vicidial.conf` remains because upstream cleanup is commented out. Never include BUILD files. |
| `/etc/astguiclient.conf` | AlmaDial application configuration stage | Private DB/runtime settings and `PATHmonitor`; generator reads these inputs, does not own their contents. |
| `/etc/my.cnf.d/almadial-m004.cnf` | AlmaDial database stage | `0640 root:mysql`; effective `skip_networking=ON`, Unix-socket local applications. |
| `/usr/local/share/perl5` and required module directories | AlmaDial dependency stage | Shared traversal `0755`; module files remain `0644`. No recursive permission widening of credentials. |

An operator-owned override must use a separate explicitly declared file/context/object,
backed up and collision-checked before inclusion. No override file was invented or
installed here. No manual edits of generated files form part of the contract.

## Actual include graph

```text
pjsip.conf ----------------> pjsip-vicidial.conf
pjsip_wizard.conf ----------> pjsip_wizard-vicidial.conf
extensions.conf [globals] -> extensions-vicidial.conf
confbridge.conf -----------> confbridge-vicidial.conf

9001 endpoint -> alm3-echo -> Answer / Echo / Hangup
9051 wizard  -> alm5-local -> vicidial-auto-phones -> PJSIP_DIAL_CONTACTS(9051)
                                               -> alm5-local,9059 -> Hangup
```

Including the generated dialplan parses its other contexts but does not make them
reachable from these endpoints. `vicidial-auto`, `vicidial-auto-external` and
`vicidial-auto-internal` are not included in either phone's allowed context graph.
The generated external `MeetMe()` route is deliberately quarantined. Internal
recording/AGI routes have not been runtime-qualified. Do not attach agents or trunks
to the larger generated graph until M0-06 evidence permits it.

The PJSIP wizard creates endpoint, AOR and inbound auth objects from one definition;
transports remain base-owned. This matches the [official Asterisk wizard contract](https://docs.asterisk.org/Configuration/Channel-Drivers/SIP/Configuring-res_pjsip/PJSIP-Configuration-Wizard/).

## r4037 source-to-output map

Source: `bin/ADMIN_keepalive_ALL.pl`, exact SHA-256
`4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67`.
Line references apply only to that pinned file.

| Source lines | Observed responsibility |
|---|---|
| 210–213 | Reserved extension list. |
| 538–608 | System/server settings and generation flags from DB. |
| 3258–3275 | Active/generate/rebuild gate; clears rebuild flag **before** successful publication. |
| 3309–3339 | Legacy MeetMe route output; retained even with selected ConfBridge server. |
| 3629–3663 | Recording/version branch. Actual r4037 line 3655 duplicates 8309 instead of 8310; corrected in pinned derivation, not upstream. |
| 4317–4510 onward | PJSIP phone and wizard output; modern phone Dial uses `PJSIP_DIAL_CONTACTS`. |
| 5337 onward | ConfBridge bridge/user/menu profiles. |
| 5473 onward | Conference allocation: when empty, 300 server-associated numbers 9600000–9600299. |
| 5557–5871 | File comparison/publication, permissions, reload requests and BUILD cleanup. Unsuffixed voicemail exception included. |
| 7159 onward | `leading_zero`, filtering and version-parsing helpers. |

The isolated extraction includes the complete generation block plus its actual source
helpers and DB settings reads. Its 94 fixed Asterisk path occurrences are redirected
to staging and 15 screen requests recorded without dispatch. It performs real upstream
generation, with one documented source correction; it is not the whole keepalive daemon.
Source outside that block also manages processes/media/voicemail greetings. Do not run
whole `ADMIN_keepalive_ALL.pl` on these live paths on the strength of this experiment.

## Publication, failure and recovery rules

Only one writer may publish a generated path. This task uses the isolated generator
and controlled copying; the full keepalive process is not running concurrently.
All final copied bytes must equal staged bytes. Back up base and generated files,
check their preimage hashes, preserve mode/owner/SELinux context, then inspect effective
endpoints/dialplan/profiles after reload. Reconcile DB flag and files after a failure:
a cleared rebuild flag alone is not proof that generation succeeded. Upstream's
`File::Compare` error path and non-atomic `cp -f` are not a safe installer journal.

Before enabling the full keepalive in a later task, its screen dispatch, publication
paths, DB server identity and the restricted native reload adapter must be integrated
as one reviewed writer. That operational daemon integration remains unqualified here.

See [experiment evidence](m0-05-vicidial-config-generation.md) and the
[bounded adapter](asterisk-vicidial-adapter.md). Canonical architecture index:
[configuration ownership](../architecture/config-ownership.md).
