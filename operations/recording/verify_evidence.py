#!/usr/bin/python3
"""Verify the saved PR-01 evidence, never infer browser/audio acceptance."""
import json
from pathlib import Path

base=Path(__file__).parent.parent/'qa/evidence/recording-finalization'
def load(name):
    return json.loads((base/name).read_text())
b,a=load('before.json'),load('after.json')
results={}
def check(name,condition):
    assert condition,name
    results[name]='PASS'
check('unrelated_service_pid_start_and_state_unchanged',b['services']==a['services'])
check('all_expected_services_active',all('ActiveState=active' in v['stdout'] for v in a['services'].values()))
allowed={'/etc/cron.d/vicidial','/etc/httpd/conf.d/02-alma-vicidial-ssl.conf'}
check('all_preexisting_other_file_hashes_unchanged',all(a['hashes'].get(k)==v for k,v in b['hashes'].items() if k not in allowed))
check('firewalld_runtime_and_permanent_unchanged',b['firewall']==a['firewall'])
check('sip_gate_policy_unchanged',b['sip_gate_policy_sha256']==a['sip_gate_policy_sha256'])
check('tls_cert_unchanged_and_trusted_connection_succeeded',b['tls']['sha256']==a['tls']['sha256'])
old='*/3 * * * * root /usr/share/astguiclient/AST_CRON_audio_1_move_mix.pl --MIX'
check('only_conflicting_cron_line_changed',a['cron']==b['cron'].replace(old,'# PR01: replaced by alma-vicidial-recording.timer; old: '+old))
for label in ['alma-portal-renew.timer','alma-vicidial-backup.timer','alma-vicidial-recording.timer']:
    check(label+'_enabled_and_active',all(s in a['timers'][label]['stdout'] for s in ['ActiveState=active','UnitFileState=enabled']))
check('no_extra_full_backup_created',b['backup_complete_sets']==a['backup_complete_sets']==2)
check('backup_health_remains_ok',a['backup_health']['rc']==0)
before={r['path']:r for r in b['recordings']}
after={r['path']:r for r in a['recordings']}
check('all_four_original_wavs_identical_including_permissions',all(after.get(k)==v for k,v in before.items()) and len(before)==4)
final=after['/var/spool/asterisk/monitorDONE/20261107-024337_7000000001-all.wav']
raw=after['/var/spool/asterisk/monitor/20261107-024337_7000000001-in.wav']
check('real_final_byte_identical_valid_duration_and_private',final['sha256']==raw['sha256'] and final['mode']=='0o600' and final['seconds']==55.76 and final['size']==892204)
old_db=b['database']['stdout'].splitlines()
new_db=a['database']['stdout'].splitlines()
url='https://2.29.153.161/RECORDINGS/20261107-024337_7000000001-all.wav'
check('only_recording_2_location_changed_in_sampled_metadata',new_db==[s.replace('\tNULL','\t'+url) if s.startswith('2\t') else s for s in old_db])
check('real_native_processing_succeeded',load('recording-2.json')['results']==[{'id':2,'result':'finalized_private'}])
check('real_repeated_processing_idempotent',load('repeated-recording-2.json')['results']==[{'id':2,'result':'already_finalized'}])
check('recording_service_exit_success',a['recording_service']['stdout']=='Result=success\nExecMainStatus=0\n')
expected={('https','/',False):(303,'/portal/'),('https','/portal/',False):(200,None),
 ('https','/agc/vicidial.php',False):(303,'https://2.29.153.161/'),
 ('https','/vicidial/admin.php',False):(303,'https://2.29.153.161/'),
 ('https','/agc/vicidial.php',True):(403,None),('https','/vicidial/admin.php',True):(403,None)}
for r in a['http']:
    key=(r['scheme'],r['path'],r['ajax'])
    want=expected.get(key,(403,None))
    check('HTTP_'+str(key),(r['code'],r['location'])==want)
check('apache_syntax_ok',a['apache']['rc']==0 and 'Syntax OK' in a['apache']['stderr'])
check('ssh_22222_and_sip_udp_listeners_present',all(s in a['listeners']['stdout'] for s in [':22222',':5060',':5061']))
fixture=load('native-fixture.json')
check('unmodified_native_fixture_with_DBI_double',all(fixture[k] for k in ['byte_identical','https_location_generated','repeated_run_no_duplicate_update']))
failure=load('failure-probe.json')['tests']
check('isolated_installed_failure_status_nonzero',all(failure[k]['exit_code']==1 and failure[k]['status']=='failed' for k in ['missing_tools','source_pin','processing_failure']))
check('scoped_rollback_extract_rehearsed',load('deployment.json')['rollback_rehearsal']=='byte/metadata-identical extraction; bash -n PASS')
listening = load('operator-listening.json')
check('operator_reported_both_finalized_recording_voices',
      listening['result'] == 'Both voices confirmed' and
      listening['evidence_category'] == 'operator confirmed')
check('desktop_external_media_HEAD_denied',
      all(r['status'] == 403 and r['method'] == 'HEAD'
          for r in load('desktop-denial.json')['checks']))
print(json.dumps(dict(checks=results,limitations=['No accepted media publication','No same-IP individual-user playback test','Listening confirmation is operator evidence, not an automated intelligibility test','No new call or full reboot test','Failed-action probes use isolated state and mocked actions']),indent=2))
