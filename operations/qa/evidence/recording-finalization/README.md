# PR-01 sanitized evidence index

All live checks used noninteractive SSH TCP 22222. No audio, secrets, dumps or
private keys are included. Known synthetic call identifiers are retained to make
the database/file association reproducible. Full report:
[`../../VICIDIAL_RECORDING_FINALIZATION_ACCEPTANCE.md`](../../VICIDIAL_RECORDING_FINALIZATION_ACCEPTANCE.md).

| File | Meaning |
|---|---|
| `before.json` | Pre-change hashes, service metadata, selected recordings/call, policy hashes and backup health |
| `after.json` | Same bounded checks after real private finalization; GET navigation / HEAD media |
| `deployment.json` | Scoped rollback, prior/current configuration hashes, policy and Apache action |
| `native-fixture.json` | Unmodified installed Perl with synthetic audio and private DBI double; not a live SQL test |
| `recording-2.json` | Actual recovery of completed recording 2; native live location update |
| `repeated-recording-2.json` | Actual repeat, `already_finalized` |
| `native-authorization.json` | Narrow installed native recording permissions/link-source findings |
| `media-source-discovery.json` | Targeted discovery of installed native file-stream primitives |
| `native-download-analysis.json` | Lead Details ZIP path first fetches stored URLs; not a private media streamer |
| `failure-probe.json` | Installed wrapper against private fixture state/mocked actions; no live DB/audio writes |
| `timer-probe.json` | Empty scheduled/manual cycles and restart only of the new recording timer |
| `operator-listening.json` | Operator confirmed both voices in the finalized WAV; no worker listening claim |
| `desktop-denial.json` | External development desktop HEAD checks: recording directory and two media names denied; no audio fetched |
| `final-health.json` | Last noninteractive SSH recheck of recording health, enabled timer and Apache syntax |
| `local-tests.txt` | 24 local unit test outcomes |
| `local-tests-initial.txt` | Earlier desktop test run could not read unrelated users' `/proc` FDs; unit fixture now scopes enumeration to its own process, while production still scans all processes |
| `verification-results.json` | Repeatable assertions over saved evidence, produced by `verify_evidence.py` |
| `SHA256SUMS` | Integrity of this evidence set excluding this checksum file itself |

Before navigation used HEAD; after navigation uses GET because the portal does
not implement HEAD in the same way. Only recording requests use HEAD in the final
collector. Differences between the earlier HEAD response and later GET response
must not be interpreted as changed authorization behavior.

The operator listening reply arrived after `after.json`; it is intentionally
separate evidence, not retroactively inserted into the system snapshot. Failure
fixtures and source inspection do not prove accepted browser playback. Media
remains denied; no authorized native UI playback or same-IP identity test passed.
