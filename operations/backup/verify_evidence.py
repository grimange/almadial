#!/usr/bin/env python3
"""Compare sanitized PR-02 evidence; never open a production backup payload."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
EVIDENCE = ROOT.parent / 'qa/evidence/backup-recovery'


def read(name):
    return json.loads((EVIDENCE / (name + '.json')).read_text())


def stable_gate(value):
    if isinstance(value, list):
        return [stable_gate(item) for item in value]
    if isinstance(value, dict):
        # Preserve rule order, handles, grant addresses, timeout policy and ports.
        return {key: ({} if key == 'counter' else stable_gate(item))
                for key, item in value.items() if key != 'expires'}
    return value


def main():
    before, after = read('before'), read('after')
    final, failure = read('final-probe'), read('failure-probe')
    results = {}

    def check(name, passed):
        results[name] = 'PASS' if passed else 'FAIL'

    check('host_identity', before['host'] == after['host'] == 'dialer01')
    check('nine_existing_services_same_pids_start_times_and_restarts',
          before['services'] == after['services'] and
          all('ActiveState=active' in r['stdout'] for r in after['services'].values()))
    check('twenty_existing_critical_file_hashes_unchanged', len(before['hashes']) == 20 and
          all(after['hashes'].get(path) == value for path, value in before['hashes'].items()))
    mappings = {'backup.py': '/usr/local/sbin/alma-vicidial-backup',
                'config.json': '/etc/alma-vicidial-backup.json',
                'alma-vicidial-backup.service': '/etc/systemd/system/alma-vicidial-backup.service',
                'alma-vicidial-backup.timer': '/etc/systemd/system/alma-vicidial-backup.timer'}
    source_hashes = {name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest() for name in mappings}
    check('four_installed_files_match_current_sources', all(
        after['hashes'].get(target) == source_hashes[name] for name, target in mappings.items()))
    for name in ('channels', 'apache_syntax', 'listeners', 'firewall_runtime',
                 'firewall_permanent', 'tls_timer', 'worker'):
        check(name + '_unchanged_and_success', before['commands'][name] == after['commands'][name]
              and after['commands'][name]['rc'] == 0)
    check('nft_policy_and_effective_grant_members_unchanged', stable_gate(json.loads(
        before['commands']['sip_gate']['stdout'])) == stable_gate(json.loads(
        after['commands']['sip_gate']['stdout'])))
    check('raw_recording_bytes_and_inventory_unchanged', before['recordings'] == after['recordings'])
    check('trusted_served_tls_certificate_unchanged', before['served_tls'] == after['served_tls'])
    check('portal_and_unauthorized_redirect_resource_semantics_unchanged',
          before['http'] == after['http'] and [r['code'] for r in after['http']] ==
          [303, 200, 303, 303, 403, 403])
    check('two_complete_archives_fully_reverified', len(final['archive_verifications']) == 2
          and all(item['result']['database_tables'] == 376 and
                  item['result']['payloads'] == 5 and item['result']['recording_files'] == 6
                  for item in final['archive_verifications']))
    check('all_exported_backup_payloads_root_private', all(
        item['uid'] == 0 and item['mode'] in ('0o600', '0o700')
        for item in after['backup_permissions']))
    check('unprivileged_sql_read_denied', final['unprivileged_read_denied_rc'] == 1)
    check('failure_probe_preserved_prior_archive', failure['complete_before'] == failure['complete_after']
          and failure['prior_backup_verified_after_failure']['payloads'] == 5)
    check('overlap_and_injected_failure_detected', failure['overlap_rejected_rc'] == 1 and
          failure['injected_failure_rc'] == 1 and failure['health_failure_rc'] == 1 and
          'ActiveState=failed' in failure['systemd_failed'])
    check('successful_run_recovered_health_and_retained_failure_history',
          after['health']['rc'] == 0 and final['health']['outcome'] == 'ok' and
          final['health']['last_failure']['reason'] == 'required_dataset_missing')
    timer = after['commands']['backup_timer']['stdout']
    check('timer_enabled_active_persistent_explicit_timezone', all(
        value in timer for value in ('UnitFileState=enabled', 'ActiveState=active',
                                    'Persistent=yes', 'America/New_York')))
    check('unit_validation', final['unit_validation']['rc'] == 0)
    check('synthetic_scoped_rollback_rehearsal', final['rollback_rehearsal']['result'] == 'PASS')
    print(json.dumps(dict(results=results, source_sha256=source_hashes,
                         evidence_scope='Local comparisons of captured host evidence; no restore, '
                                        'off-host copy, live call, or reboot acceptance.'), indent=2))
    return 0 if all(result == 'PASS' for result in results.values()) else 1


if __name__ == '__main__':
    raise SystemExit(main())
