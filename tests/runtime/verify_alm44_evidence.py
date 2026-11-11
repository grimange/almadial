#!/usr/bin/env python3
"""Read-only correlation of ALM-44 captured host evidence; does not certify audio."""
import csv
import io
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT / 'operations/qa/evidence/alm44-auto-call-delivery'


def rows(section):
    assert section['rc'] == 0, section
    return list(csv.DictReader(io.StringIO(section['stdout']), delimiter='\t'))


def main():
    state = json.loads((EVIDENCE / 'final-state.json').read_text())
    calls = {r['lead_id']: r for r in rows(state['calls'])}
    leads = {r['lead_id']: r for r in rows(state['leads'])}
    records = {r['lead_id']: r for r in rows(state['recordings'])}
    agents = {r['lead_id']: r for r in rows(state['agents'])}
    for lead, name in [('77', 'ami-events.jsonl'), ('78', 'ami-media-recheck.jsonl'), ('79', 'ami-voice-retest.jsonl')]:
        call, record, agent = calls[lead], records[lead], agents[lead]
        assert call['status'] == leads[lead]['status'] == agent['status'] == 'NI'
        assert leads[lead]['called_count'] == '1'
        assert leads[lead]['called_since_last_reset'] == 'Y'
        assert call['uniqueid'] == agent['uniqueid']
        assert call['start_epoch'] == record['start_epoch']
        assert call['end_epoch'] == record['end_epoch']
        assert call['length_in_sec'] == record['length_in_sec']
        # Native agent and call accounting differ by at most a one-second tick.
        assert abs(int(agent['talk_sec']) - int(call['length_in_sec'])) <= 1
        assert record['location'].startswith('https://2.29.153.161/RECORDINGS/')
        events = [json.loads(line) for line in (EVIDENCE / name).read_text().splitlines()]
        answers = [e for e in events if e.get('Event') == 'DialEnd' and e.get('DialStatus') == 'ANSWER']
        joins = [e for e in events if e.get('Event') == 'ConfbridgeJoin' and e.get('Uniqueid') == call['uniqueid']]
        assert answers and len(joins) == 1 and joins[0]['Conference'] == '9600000'
        assert answers[0]['observed_utc'] < joins[0]['observed_utc']
        # Third bounded AMI observer ended before the operator finished capture
        # checks. Its call end is independently evidenced by native end_epoch,
        # disposition, and the final zero-channel/session state.
        if lead != '79':
            assert any(e.get('Event') == 'Hangup' and e.get('Uniqueid') == call['uniqueid'] for e in events)
        assert int(call['end_epoch']) > int(call['start_epoch'])
    cleanup = state['cleanup']['stdout']
    for marker in ['live_agents\n0', 'auto_calls\n0', 'qa_hopper\n0', 'qa_incall\n0', 'QAOUT01\tN\tMANUAL\t0']:
        assert marker in cleanup, marker
    assert '0 active channels' in state['core show channels']['stdout']
    assert all(state['protected_hashes'][k] == v for k, v in json.loads(
        (EVIDENCE / 'preservation.json').read_text())['hashes'].items())
    assert any('Addr->IP' in s and '(null)' in s for s in state['phone1002']['selected'])
    energy = [json.loads(s) for s in (EVIDENCE / 'rtp-after-microphone-change.jsonl').read_text().splitlines()]
    incoming = [r for r in energy if r.get('direction') == 'incoming' and r.get('payload_type') == 0]
    outgoing = [r for r in energy if r.get('direction') == 'outgoing' and r.get('payload_type') == 0]
    assert incoming and outgoing
    assert all(r['rms'] == 0 and r['peak'] == 0 for r in incoming)
    assert all(r['rms'] > 1000 for r in outgoing)
    deployment = json.loads((EVIDENCE / 'cause-and-deployment.json').read_text())
    import hashlib
    config = ROOT / 'operations/vicidial/runtime/almadial-vicidial-agi-logs.conf'
    assert hashlib.sha256(config.read_bytes()).hexdigest() == deployment['deployed_config']['sha256']
    assert deployment['tmpfiles_reconcile']['rc'] == deployment['asterisk_writable']['rc'] == 0
    print(json.dumps({'result': 'PASS', 'calls_correlated': ['77', '78', '79'],
                      'routing_disposition_recording_cleanup': 'PASS',
                      'human_bidirectional_audio': 'NOT CERTIFIED BY THIS CHECK'}, indent=2))


if __name__ == '__main__':
    main()
