#!/usr/bin/env python3
"""Verify the bounded PRD-03B patch against private, pinned native source copies.

No SSH, live DB, package installs, or host deployment. Source is not vendored.
SQLite executes the exact release JOIN/WHERE, not MariaDB's multi-table UPDATE.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile

BASE = Path(__file__).resolve().parents[2]
PINS = {
    'vicidial.php': 'f701a0dbc2e11fb7e4c85e7bee3f08f2967b43b3f3e6497daa84a8abfb55d358',
    'vdc_db_query.php': '5aaeabd1d9b247dc563b9e94bb86d0b3323346b226b52d47a00ca2506897108d',
}
PATCH = BASE / 'operations/vicidial/patches/0001-preview-logout.patch'


def run(argv, **kwargs):
    return subprocess.run(argv, text=True, capture_output=True, check=True, timeout=30, **kwargs).stdout


def qualify_predicate(sql):
    """Exercise actual proposed admission predicates with isolated synthetic rows."""
    join = sql.split(' SET ', 1)[0].removeprefix('UPDATE')
    where = sql[sql.index('WHERE'):]
    query = 'SELECT l.lead_id FROM' + join + ' ' + where
    cases = [
        ('eligible list preview', '', True),
        ('active call', 'INSERT INTO vicidial_auto_calls VALUES(100)', False),
        ('agent INCALL', "UPDATE vicidial_live_agents SET status='INCALL'", False),
        ('agent current lead', 'UPDATE vicidial_live_agents SET lead_id=100', False),
        ('other user', "UPDATE vicidial_list SET user='other'", False),
        ('other session conference', "UPDATE vicidial_live_agents SET conf_exten='other'", False),
        ('other campaign', "UPDATE vicidial_lists SET campaign_id='OTHER'", False),
        ('changed status', "UPDATE vicidial_list SET status='SALE'", False),
        ('changed count', 'UPDATE vicidial_list SET called_count=2', False),
        ('previously called flag', "UPDATE vicidial_list SET called_since_last_reset='Y1'", False),
        ('preview already cleared', 'UPDATE vicidial_live_agents SET preview_lead_id=0', False),
        ('pending external dial', "UPDATE vicidial_live_agents SET external_dial='DIALONLY'", False),
        ('missing lead counter', 'DELETE FROM vicidial_lead_call_daily_counts', False),
        ('missing phone counter', 'DELETE FROM vicidial_phone_number_call_daily_counts', False),
        ('zero counter', 'UPDATE vicidial_lead_call_daily_counts SET called_count_manual=0', False),
    ]
    results = []
    for name, mutation, expected in cases:
        with sqlite3.connect(':memory:') as db:
            db.executescript('''
              CREATE TABLE vicidial_list(lead_id INTEGER,list_id INTEGER,status TEXT,user TEXT,called_count INTEGER,called_since_last_reset TEXT,phone_number TEXT);
              CREATE TABLE vicidial_live_agents(preview_lead_id INTEGER,user TEXT,server_ip TEXT,conf_exten TEXT,campaign_id TEXT,lead_id INTEGER,status TEXT,external_dial TEXT);
              CREATE TABLE vicidial_lists(list_id INTEGER,campaign_id TEXT);
              CREATE TABLE vicidial_lead_call_daily_counts(lead_id INTEGER,called_count_total INTEGER,called_count_manual INTEGER);
              CREATE TABLE vicidial_phone_number_call_daily_counts(phone_number TEXT,called_count INTEGER);
              CREATE TABLE vicidial_auto_calls(lead_id INTEGER);
              INSERT INTO vicidial_list VALUES(100,910016,'INCALL','synthetic_agent',1,'Y','0000000001');
              INSERT INTO vicidial_live_agents VALUES(100,'synthetic_agent','192.0.2.1','9600001','SYNTHETIC',0,'PAUSED','');
              INSERT INTO vicidial_lists VALUES(910016,'SYNTHETIC');
              INSERT INTO vicidial_lead_call_daily_counts VALUES(100,1,1);
              INSERT INTO vicidial_phone_number_call_daily_counts VALUES('0000000001',1);
            ''')
            if mutation:
                db.execute(mutation)
            admitted = bool(db.execute(query).fetchall())
            assert admitted == expected, name
            results.append({'case': name, 'admitted': admitted, 'assertion': 'PASS'})
    return results


def verify(source):
    results = {'patchSha256': hashlib.sha256(PATCH.read_bytes()).hexdigest(), 'sourcePins': PINS,
               'notProven': ['MariaDB multi-table UPDATE and locking', 'PHP 8.3 execution',
                             'corrected native Chromium workflow', 'live next-lead eligibility after correction']}
    with tempfile.TemporaryDirectory(prefix='prd03c-proposal-') as directory:
        root = Path(directory)
        (root / 'agc').mkdir()
        for name, digest in PINS.items():
            data = (source / name).read_bytes()
            assert hashlib.sha256(data).hexdigest() == digest, 'Source drift: ' + name
            (root / 'agc' / name).write_bytes(data)
        run(['patch', '--batch', '--fuzz=0', '-p1', '-i', str(PATCH)], cwd=root)
        results['phpVersion'] = run(['php', '-v']).splitlines()[0]
        results['lint'] = [run(['php', '-l', str(root / 'agc' / name)]).strip().split(' in ')[0] for name in PINS]
        results['javascript'] = json.loads(run(['node', str(BASE / 'tests/webphone/test_preview_logout_proposal.cjs'), str(root / 'agc/vicidial.php')]))
        # Run actual PHP release code, with extension-free stubs to capture its SQL.
        native = (root / 'agc/vdc_db_query.php').read_text()
        assert "$preview_action = isset($_POST['preview_action'])" in native
        assert '&preview_action=dial' in (root / 'agc' / 'vicidial.php').read_text()
        begin = native.index("\tif ($preview_action === 'logout')")
        block = native[begin:native.index('\t// End PRD-03C bounded release.', begin)]
        harness = root / 'release.php'
        queries = root / 'queries.jsonl'
        harness.write_text('''<?php
$preview_action='logout';$lead_id='100';$called_count='1';$stage='NEW';
$user='synthetic_agent';$server_ip='192.0.2.1';$conf_exten='9600001';$campaign='SYNTHETIC';$link=null;
function mysql_to_mysqli($sql,$link) {file_put_contents(getenv('QUERY_FILE'),json_encode($sql)."\\n",FILE_APPEND);return true;}
function mysqli_affected_rows($link) {return 4;}
''' + block)
        import os
        response = run(['php', '-n', '-d', 'extension=ctype', str(harness)], env={**os.environ, 'QUERY_FILE': str(queries)})
        assert response.strip() == 'PREVIEW RELEASED', response
        sql = [json.loads(line) for line in queries.read_text().splitlines()]
        assert len(sql) == 2 and sql[1].startswith('INSERT INTO vicidial_agent_skip_log')
        results['phpReleaseCode'] = 'actual proposed block constructs one guarded UPDATE and native skip audit; DB calls stubbed'
        results['admissionPredicates'] = qualify_predicate(sql[0])
        # Follow the actual no-change retry branch; this is control flow, not a DB write test.
        retry_harness = harness.read_text().replace('return 4;', 'return 0;')
        retry_harness = retry_harness.replace("function mysqli_affected_rows", "function mysqli_num_rows($result) {return 1;}\nfunction mysqli_affected_rows")
        harness.write_text(retry_harness)
        queries.unlink()
        response = run(['php', '-n', '-d', 'extension=ctype', str(harness)], env={**os.environ, 'QUERY_FILE': str(queries)})
        retry_sql = [json.loads(line) for line in queries.read_text().splitlines()]
        assert response.strip() == 'PREVIEW RELEASED' and len(retry_sql) == 2
        assert retry_sql[1].startswith('SELECT 1') and not any(q.startswith('INSERT') for q in retry_sql)
        results['retryControlFlow'] = 'PASS: zero affected rows plus matching released state ACKs without a second skip audit'
        harness.write_text(retry_harness.replace('return 1;', 'return 0;'))
        queries.unlink()
        response = run(['php', '-n', '-d', 'extension=ctype', str(harness)], env={**os.environ, 'QUERY_FILE': str(queries)})
        assert response.strip() == 'PREVIEW NOT RELEASED'
        results['retryMismatch'] = 'PASS: zero affected rows plus mismatched retry state refuses logout release'
        results['generatedUpdate'] = sql[0]
        # Reverse it and compare byte for byte, not merely a successful patch exit.
        run(['patch', '--batch', '--fuzz=0', '-R', '-p1', '-i', str(PATCH)], cwd=root)
        for name, digest in PINS.items():
            assert hashlib.sha256((root / 'agc' / name).read_bytes()).hexdigest() == digest
        results['applyReverse'] = 'PASS: zero-fuzz apply; reverse restores both exact source digests'
    return results


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path, help='Private directory containing the two pinned native PHP files')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = verify(args.source)
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({'javascriptCases': len(result['javascript']['cases']), 'predicateCases': len(result['admissionPredicates']), 'applyReverse': result['applyReverse'], 'notProven': result['notProven']}, indent=2))
