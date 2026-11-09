#!/usr/bin/env python3
"""Execute the PINNED native Perl recovery branch against isolated SQLite.

No SSH, credentials, DBI dependency or live writes. This qualifies branch SQL and
control flow, not MariaDB/MyISAM concurrency or the complete maintenance script.
The operator supplies an inspected copy of AST_flush_DBqueue.pl; fail on drift.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile

SOURCE_SHA256 = 'f3e4fe7762078cdc4578bddb759afa05e6ba1d0b4c6a554d4da88248860b678b'
BRIDGE = r'''
use JSON::PP;
$|=1;
sub rpc {
 print encode_json({sql=>$_[0]})."\n";
 my $line=<STDIN>; die 'bridge closed' unless defined $line;
 my $r=decode_json($line); die $r->{error} if $r->{error}; return $r;
}
package LocalDB;
sub prepare { bless {sql=>$_[1]}, 'LocalStatement' }
sub do { main::rpc($_[1])->{changed} }
sub errstr { 'isolated SQL failure' }
package LocalStatement;
sub execute { my $s=shift; $s->{rows}=main::rpc($s->{sql})->{rows}; $s->{i}=0; 1 }
sub rows { scalar @{$_[0]->{rows}} }
sub fetchrow_array { my $s=shift; @{$s->{rows}->[$s->{i}++]} }
sub finish { 1 }
package main;
$dbhA=bless {}, 'LocalDB';
$reset_stuck_leads=1; $Q=1; $DB=0; $DBX=0; $T=0;
$SQLdate_NEG_sixhour='2026-11-09 06:00:00';
$SQLdate_NEG_2min='2026-11-09 11:57:59';
$SQLdate_MIDNIGHT='2026-11-09 00:00:00';
$SQLdate_NOW='2026-11-09 12:00:00';
$stuck_listsSQL="and list_id IN('910016')";
$STUCKLOGfile=$ARGV[0];
'''


def database():
    db = sqlite3.connect(':memory:')
    db.executescript('''
    CREATE TABLE vicidial_list(lead_id INTEGER PRIMARY KEY,list_id INTEGER,
      status TEXT,user TEXT,last_local_call_time TEXT,called_count INTEGER,
      modify_date TEXT,called_since_last_reset TEXT);
    CREATE TABLE vicidial_auto_calls(lead_id INTEGER);
    CREATE TABLE vicidial_log(lead_id INTEGER,call_date TEXT);
    CREATE TABLE vicidial_closer_log(lead_id INTEGER,call_date TEXT);
    CREATE TABLE vicidial_dial_log(lead_id INTEGER,call_date TEXT);
    CREATE TABLE vicidial_live_agents(preview_lead_id INTEGER,
      last_state_change TEXT,lead_id INTEGER);
    CREATE TABLE vicidial_lead_call_daily_counts(lead_id INTEGER,
      called_count_total INTEGER,called_count_manual INTEGER);
    ''')
    return db


def lead(db, ident=1, status='INCALL', modified='2026-11-09 11:55:00', list_id=910016):
    db.execute('INSERT INTO vicidial_list VALUES(?,?,?,?,?,?,?,?)',
               (ident, list_id, status, 'synthetic', modified, 1, modified, 'Y'))
    db.execute('INSERT INTO vicidial_lead_call_daily_counts VALUES(?,?,?)', (ident, 1, 1))


def state(db):
    return [dict(zip(('lead','status','called','count','owner'), r)) for r in
            db.execute('SELECT lead_id,status,called_since_last_reset,called_count,user FROM vicidial_list ORDER BY lead_id')]


def run_branch(db, branch, race=False):
    queries = []
    with tempfile.TemporaryDirectory(prefix='prd03c-recovery-') as directory:
        script = Path(directory) / 'qualified.pl'
        script.write_text(BRIDGE + branch)
        proc = subprocess.Popen(['perl', str(script), str(Path(directory) / 'stuck.log')],
                                stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, text=True)
        try:
            for line in proc.stdout:
                sql = json.loads(line)['sql']
                queries.append(sql)
                if race and sql.startswith('UPDATE vicidial_list'):
                    # A legitimate call appears AFTER the last native guard SELECT.
                    db.execute("UPDATE vicidial_list SET status='SALE' WHERE lead_id=1")
                    db.execute('INSERT INTO vicidial_auto_calls VALUES(1)')
                try:
                    cur = db.execute(sql)
                    answer = {'rows': cur.fetchall() if cur.description else [], 'changed': cur.rowcount}
                except sqlite3.Error as error:
                    answer = {'error': str(error)}
                proc.stdin.write(json.dumps(answer) + '\n')
                proc.stdin.flush()
            proc.stdin.close()
            rc = proc.wait(timeout=10)
            err = proc.stderr.read()
            if rc:
                raise RuntimeError(f'Isolated Perl branch exited {rc}: {err}')
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait()
    return queries


def qualify(source):
    content = source.read_bytes()
    assert hashlib.sha256(content).hexdigest() == SOURCE_SHA256, 'Native source drift'
    text = content.decode()
    branch = text[text.index('if ($reset_stuck_leads > 0)'):text.index('if ($check_xfers > 0)')]
    results = []
    # Expected outcomes describe actual native behavior, including safety defects.
    for name, table, expected in [
        ('orphan', None, 'NEW'),
        ('active_call', 'vicidial_auto_calls', 'INCALL'),
        ('outbound_today', 'vicidial_log', 'INCALL'),
        ('inbound_today', 'vicidial_closer_log', 'INCALL'),
        ('dial_today', 'vicidial_dial_log', 'INCALL'),
        ('active_preview_today', 'today', 'INCALL'),
        ('active_preview_before_midnight', 'yesterday', 'NEW'),
        ('active_lead_without_preview_pointer', 'lead_only', 'NEW'),
    ]:
        db = database()
        lead(db)
        if table == 'vicidial_auto_calls':
            db.execute('INSERT INTO vicidial_auto_calls VALUES(1)')
        elif table in ('vicidial_log', 'vicidial_closer_log', 'vicidial_dial_log'):
            db.execute(f'INSERT INTO {table} VALUES(?,?)', (1, '2026-11-09 11:59:00'))
        elif table:
            db.execute('INSERT INTO vicidial_live_agents VALUES(?,?,?)',
                       (0 if table == 'lead_only' else 1,
                        '2026-11-08 23:59:00' if table == 'yesterday' else '2026-11-09 11:50:00', 1))
        queries = run_branch(db, branch)
        observed = state(db)
        assert observed[0]['status'] == expected, name
        assert observed[0]['count'] == 1, 'Native branch unexpectedly restored count'
        assert db.execute('SELECT called_count_total,called_count_manual FROM vicidial_lead_call_daily_counts').fetchone() == (1, 1)
        results.append({'case': name, 'assertion': 'PASS', 'observed': observed, 'queries': queries})
    for name, modified, list_id, status, expected in [
        ('younger_than_121_seconds', '2026-11-09 11:58:00', 910016, 'INCALL', 'INCALL'),
        ('older_than_six_hours', '2026-11-09 05:59:59', 910016, 'INCALL', 'INCALL'),
        ('other_list', '2026-11-09 11:55:00', 910017, 'INCALL', 'INCALL'),
        ('ERI_candidate', '2026-11-09 11:55:00', 910016, 'ERI', 'NEW'),
        ('other_status', '2026-11-09 11:55:00', 910016, 'NI', 'NI'),
    ]:
        db = database()
        lead(db, status=status, modified=modified, list_id=list_id)
        run_branch(db, branch)
        assert state(db)[0]['status'] == expected, name
        results.append({'case': name, 'assertion': 'PASS', 'observed': state(db)})
    db = database()
    lead(db)
    run_branch(db, branch)
    before = state(db)
    repeat = run_branch(db, branch)
    assert state(db) == before and len(repeat) == 1
    results.append({'case': 'sequential_repeat', 'assertion': 'PASS', 'observed': state(db)})
    db = database()
    lead(db)
    queries = run_branch(db, branch, race=True)
    assert state(db)[0]['status'] == 'NEW' and db.execute('SELECT COUNT(*) FROM vicidial_auto_calls').fetchone()[0] == 1
    results.append({'case': 'call_after_guard_race', 'assertion': 'PASS',
                    'meaning': 'SAFETY FAILURE reproduced: native update overwrites a newly active lead', 'observed': state(db), 'queries': queries})
    return {'sourceSha256': SOURCE_SHA256, 'execution': 'unaltered native Perl recovery branch; isolated SQLite SQL bridge',
            'notProven': ['full maintenance script', 'MariaDB/MyISAM locking', 'live host recovery'],
            'cases': results, 'caseCount': len(results)}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = qualify(args.source)
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(f"{result['caseCount']} isolated native-branch assertions passed; inspect safety failures in output")
