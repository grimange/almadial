# r4037 / native Asterisk 22: bounded adaptation

ALM-5 implementation evidence; independent acceptance pending. No Go installer or
new daemon is implemented. The upstream source tree remains unchanged.

## Demonstrated reasons

1. r4037 dispatches reloads with `screen -XS asterisk ...`. The accepted Asterisk
   runtime is a native systemd service, without that screen session.
2. The installed Asterisk 22.12.0 CLI does not implement `pjsip reload`; it returns
   a diagnostic even when the CLI process exit code is zero.
3. r4037 line 3655 emits extension **8309**, priority **2** a second time in its
   **8310** recording branch. Actual Asterisk reload/restart logs reject the duplicate.

## Applied experiment adapter

`scripts/lab/m0_05_generator.py` accepts only the pinned original source SHA-256.
It extracts the complete actual generation block and helpers, points its DB reads
at the disposable `almadial_m005` clone, supplies the real `PATHmonitor` input,
redirects 94 `/etc/asterisk/` occurrences to protected staging and replaces 15 screen
invocations with a recording-only stub. It does not execute or deploy the result.
The lab explicitly executes the derived Perl after review. Original files, original
outputs and failed diagnostics are preserved privately.

One source correction is applied **before** generation: after the unique
`8310,1,Answer` marker, change the single modern `8309,2,MixMonitor` to
`8310,2,MixMonitor`. The byte-pinned source and occurrence-count assertions reject
unknown revisions/branches. No generated file is manually patched. Regression tests
reproduce both the original duplicate and the exact limited correction. Recording
media is NOT_RUN; this only fixes priority identity and configuration loading.

Original source SHA-256:
`4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67`.
Original extracted block SHA-256:
`41c7019e76f76e2aba8cac030b47fc17bce9b8544afcaa9980273491e313569f`.
Final derived script SHA-256:
`8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda`.
The earlier uncorrected derivation was
`579ecdff820ef6965944ce6f5ea0f051a22fc78a1319ef71ca28ba7536c147c6`.

## Verified native dispatch

After verified publication, these actual CLI operations succeeded:

```bash
asterisk -rx 'module reload res_pjsip.so'
asterisk -rx 'module reload res_pjsip_config_wizard.so'
asterisk -rx 'dialplan reload'
asterisk -rx 'module reload app_confbridge.so'
```

They require successful CLI output **and** postcondition queries, not exit zero alone.
Transport changes require a deliberate service restart rather than assumptions about
PJSIP reload. No transport change was needed here. `app_dial.so` was already installed
but absent from the ALM-3 minimal module list; it was loaded and declared persistently.

The generator is not granted arbitrary AMI Command/Originate access. The local test
AMI account demonstrates authentication, Ping/CoreStatus and rejection of Command.
Full VICIdial daemon permissions are a later M0-06 measurement, not established by
this smaller profile. No permanent dispatcher, framework or screen shim was installed.

The source's legacy MeetMe route remains present but unreachable from the two test
phone contexts. Full auto/internal/external contexts, missing AGI/recording application
modules and complete keepalive startup remain blocked until separately qualified.
