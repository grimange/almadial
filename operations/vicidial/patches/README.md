# Direct-preview logout repair — r4037

`0001-preview-logout.patch` is a local, upstream-compatible two-file patch to VICIdial's native agent application. It is **deployed on dialer01 for developer verification**, pending independent QA (ALM-36). It is not an upstream release or an AlmaDial stack promotion. Original VICIdial licensing and notices remain applicable; full upstream source is not vendored here.

The current patch supersedes the undeployed PRD-03C proposal. Its retained PRD-03C source comments identify that proposal's origin, not an approval boundary for this later authorized assignment.

| File under web root | Original SHA-256 | Deployed SHA-256 |
|---|---|---|
| `agc/vicidial.php` | `f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358` | `aea2d4637be9c9ed48a9e74472a3569639cd39c0a711ae1b17e16343fafb4084` |
| `agc/vdc_db_query.php` | `5aaeabd1d9b247dc563b9e94bb86d0b3323346b226b52d47a00ca2506897108d` | `5c34b104716ec9c69d93b833d2b63934b4475ae6af4682b7f80d3fc6a1ab7cb5` |

## Apply and validate

First verify the exact target, empty agent/channel state and source digests. Back up both originals with metadata in a private directory outside the web root. Apply to a private source copy with `patch --batch --fuzz=0 -p1 -i <absolute-patch-path>` from the directory containing `agc/`. Run PHP lint on both files and the local qualifier below. Stage each file beside its destination, preserve ownership/mode and rename into place while agents are absent. These two files must be deployed as a pair. Existing browsers must start a fresh session. Do not apply across a source mismatch or during active calls.

```bash
python3 tests/webphone/verify_preview_logout_proposal.py /private/pinned-source --output /tmp/preview-qualification.json
```

The source argument contains the two original pinned PHP files directly. Local tests prove apply/reverse, extracted JavaScript behavior and admission predicates. They do not replace the real MariaDB/PHP/Chromium tests documented in the [implementation report](../../qa/VICIDIAL_PRD03B_PREVIEW_CLEANUP_IMPLEMENTATION.md).

## Rollback

On dialer01 the original scoped backup is `/root/qa-backups/prd03b-20261109T220035Z`. With QAOUT01 inactive, no agents/channels and all previews released, verify deployed hashes, restore **only** its `vicidial.php` and `vdc_db_query.php` to `/var/www/html/agc/` preserving metadata, lint, verify the original hashes above and start a fresh browser session. Alternatively reverse the exact patch on a verified private copy (`patch --batch --fuzz=0 -R -p1`), verify both original digests, then stage/rename the pair. Local reverse testing restored both original digests exactly. A live rollback was not performed because it would reintroduce the demonstrated defect.

No service restart, database restore, schema rollback, cron change or cleanup daemon is required. Retain synthetic records and call history. Reverting the patch restores the original direct-logout defect, so use SKIP LEAD before logout on the reverted system.

Upgrades must requalify against the new upstream revision; never apply this patch blindly. Its bounded release supports the demonstrated list-driven N→Y preview. Callback/manual-entry/alternate/prior-called previews fail closed and require separate qualification. MyISAM cannot roll back the separate audit insert; an audit failure is explicitly reported and requires review.

The stored unified diff preserves upstream whitespace in context lines. Git whitespace checks should exclude this patch artifact; its exact zero-fuzz apply/reverse check is the relevant integrity test.
