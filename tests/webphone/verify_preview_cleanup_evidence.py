#!/usr/bin/env python3
"""Assert bounded native-runtime evidence; does not execute host commands or SQL."""
import argparse
import json
from pathlib import Path


def load(root, name):
    return json.loads((root / name).read_text())


def leads(snapshot):
    rows = snapshot['database']['stdout'].splitlines()
    start = next(i for i, row in enumerate(rows) if row.startswith('lead_id\tlist_id\tvendor_lead_code\tstatus\tuser\tcalled_count'))
    keys = rows[start].split('\t')
    result = {}
    for line in rows[start + 1:]:
        values = line.split('\t')
        if len(values) != len(keys) or not values[0].isdigit():
            break
        result[int(values[0])] = dict(zip(keys, values))
    return result


def state(row):
    return row['status'], int(row['called_count']), row['called_since_last_reset']


def verify(root):
    baseline = leads(load(root, 'baseline.json'))
    final = load(root, 'final-state.json')
    after = leads(final)
    for lead in [8, *range(49, 57), *range(62, 70)]:
        assert baseline[lead] == after[lead], ('unrelated lead changed', lead)
    assert state(leads(load(root, 'original-after.json'))[57]) == ('INCALL', 1, 'Y')
    for folder, lead in [('repeated-next', 59), ('isolation', 60), ('retry', 61)]:
        preview = leads(load(root, folder + '/preview.json'))[lead]
        assert state(preview) == ('INCALL', 1, 'Y'), folder
        assert state(leads(load(root, folder + '/after.json'))[lead]) == ('NEW', 0, 'N'), folder
    assert state(leads(load(root, 'active-call/after.json'))[70]) == ('NI', 1, 'Y')
    assert state(after[70]) == ('NI', 1, 'Y')
    assert state(after[61]) == ('NEW', 0, 'N')
    assert '\nincall_leads\n0\n' in final['database']['stdout']
    assert '\nlive_agents\n0\n' in final['database']['stdout']
    assert final['core show channels count']['stdout'].startswith('0 active channels\n0 active calls')
    regression = load(root, 'final-regression.json')
    changed = [p for p, value in regression['comparison'].items() if value == 'changed']
    assert sorted(changed) == ['/var/www/html/agc/vdc_db_query.php', '/var/www/html/agc/vicidial.php']
    assert regression['phone1002_unregistered']
    return {'result': 'PASS', 'unrelatedLeadRowsPreserved': 17, 'nativeDirectLogoutCases': 3,
            'activeCallFinalDisposition': 'NI / count 1 / called Y',
            'finalIncallClaims': 0, 'finalLiveAgents': 0, 'finalChannels': 0,
            'scope': 'Recorded native integration evidence; independent QA remains required'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('evidence', type=Path)
    args = parser.parse_args()
    print(json.dumps(verify(args.evidence), indent=2))
