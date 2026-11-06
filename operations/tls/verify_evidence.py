#!/usr/bin/python3
"""Validate collected host evidence; does not contact or change the server."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
evidence = root / 'qa/evidence/tls-renewal'
before = json.loads((evidence / 'before.json').read_text())
after = json.loads((evidence / 'after.json').read_text())
staging = json.loads((evidence / 'staging-evidence.json').read_text())
negative = json.loads((evidence / 'negative-test.json').read_text())
normal = json.loads((evidence / 'normal-runs.json').read_text())
details = json.loads((evidence / 'final-details.json').read_text())
checks = {}


def expect(name, value):
    checks[name] = bool(value)
    assert value, name


expect('trusted_public_and_loopback_certificate', before['served'] == after['served'] and
       all(v.get('verified_ip') == '2.29.153.161' for v in after['served'].values()))
expect('chain_verified', after['certificate_chain']['rc'] == 0)
expect('apache_syntax', after['commands']['apache_syntax']['rc'] == 0)
expect('http_portal_redirect_denial_semantics_preserved', before['http'] == after['http'])
expected_codes = [303, 200, 303, 303, 403, 403, 403, 403, 404]
expect('expected_http_codes', [row['code'] for row in after['http']] == expected_codes)
expect('services_and_master_pids_preserved', before['services'] == after['services'])
for key in ('firewall_runtime', 'firewall_permanent'):
    expect(key + '_unchanged', before['commands'][key] == after['commands'][key])
for key in ('/etc/systemd/system/alma-portal-renew.timer',
            '/etc/httpd/conf.d/00-alma-acme.conf', '/etc/httpd/conf.d/02-alma-vicidial-ssl.conf',
            '/opt/alma-portal-443/access.lua',
            '/opt/alma-portal-443/common.py', '/opt/alma-portal-443/apply_requests.py',
            '/opt/alma-portal-443/sip_guard.py'):
    expect(key + '_unchanged', before['files'][key] == after['files'][key] and
           after['files'][key].get('exists') is not False)
for source, target in [('alma_portal_tls.py', '/usr/local/libexec/alma-portal-tls'),
                       ('alma-portal-cert-deploy', '/usr/local/sbin/alma-portal-cert-deploy'),
                       ('alma-portal-renew.service', '/etc/systemd/system/alma-portal-renew.service')]:
    digest = hashlib.sha256((Path(__file__).parent / source).read_bytes()).hexdigest()
    expect(source + '_installed_hash_matches', after['files'][target]['sha256'] == digest)


def listeners(snapshot):
    # A graceful reload replaces Apache child PIDs/fds, without changing bindings.
    return sorted(tuple(line.split()[:5]) for line in snapshot['commands']['listeners']['stdout'].splitlines()[1:])


expect('listener_bindings_preserved', listeners(before) == listeners(after))
for snapshot in (before, after):
    gate = snapshot['commands']['sip_guard']
    expect(snapshot['time_utc'] + '_sip_gate_present', gate['rc'] == 0 and
           '5060' in gate['stdout'] and '5061' in gate['stdout'])
expect('staging_issuance_and_hook_succeeded', staging['state']['status.json']['dry_run'] and
       staging['state']['status.json']['outcome'] == 'ok' and
       staging['state']['status.json']['deployed'] and
       'all simulated renewals succeeded' in staging['acceptance_journal']['stdout'])
expect('failure_observable_beyond_log', negative['rejected_lineage']['rc'] == 1 and
       'ActiveState=failed' in negative['failed_unit']['stdout'] and negative['health_check']['rc'] == 1)
for index in ('1', '2'):
    state = normal['state_' + index]
    expect('noop_' + index, state['outcome'] == 'ok' and not state['deployed'] and not state['certificate_changed'])
expect('one_graceful_reload_only', normal['apache_journal']['stdout'].count('Reloaded httpd.service') == 1)
expect('timer_enabled_active_persistent', all(value in normal['timer_details']['stdout'] for value in
       ['Persistent=yes', 'ActiveState=active', 'UnitFileState=enabled']))
expect('healthy_after_recovery', normal['health']['rc'] == 0)
expect('assets_public_status_authenticated', [item['status'] for item in details['routes'].values()] ==
       [200, 200, 200, 401])
for name, digest in details['portal_files'].items():
    expect(name + '_matches_existing_source',
           hashlib.sha256((root / 'portal443' / name).read_bytes()).hexdigest() == digest)
expect('grant_worker_healthy', 'Result=success' in details['active_grants_worker']['stdout'] and
       'ExecMainStatus=0' in details['active_grants_worker']['stdout'])
expect('graceful_reload_command', '-k graceful' in details['httpd_reload_command']['stdout'])
print(json.dumps(dict(result='PASS', checks=checks), indent=2))
