#!/usr/bin/env python3
"""Read-only aggregate acceptance snapshots and sanitized log export. No raw state export."""
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import time
sys.path.insert(0, '/opt/alma-portal-443')
from common import BASE
from apply_requests import aggregate, PORTS, rule
from sip_guard import TABLE, POLICY

BACKUP = Path('/root/alma-portal-5060-backup')
TEST_IP = '158.62.74.253'
SERVER_IP = '2.29.153.161'


def run(*args):
    return subprocess.run(args, capture_output=True, text=True, timeout=20)


def snapshot():
    now = int(time.time())
    state = json.loads((BASE/'state.json').read_text())
    web, phone = aggregate(state['leases'], now)
    with tarfile.open(BACKUP/'config.tar') as archive:
        initial = json.load(archive.extractfile('var/lib/alma-portal-443/state.json'))
    unrelated_preserved = all(k in state['leases'] and
        state['leases'][k]['expires_at'] == v['expires_at'] and
        state['leases'][k].get('revoked') == v.get('revoked')
        for k,v in initial.get('leases',{}).items() if v['ip'] not in (TEST_IP, SERVER_IP))
    checksums = {}
    for line in (BACKUP/'before.sha256').read_text().splitlines():
        old, path = line.split(maxsplit=1)
        checksums[Path(path).name] = {'before':old, 'after':hashlib.sha256(Path(path).read_bytes()).hexdigest()}
    nft = json.loads(run('nft','-j','list','table','inet',TABLE).stdout)['nftables']
    rules = [v['rule'] for v in nft if 'rule' in v]
    owned_counts = {port:sum(run('firewall-cmd','--zone=public','--query-rich-rule='+rule(ip,port)).returncode==0 for ip in phone) for port in PORTS}
    test_owned = {port:run('firewall-cmd','--zone=public','--query-rich-rule='+rule(TEST_IP,port)).returncode==0 for port in PORTS}
    permanent = run('firewall-cmd','--permanent','--zone=public','--list-rich-rules').stdout
    runtime = run('firewall-cmd','--zone=public','--list-rich-rules').stdout
    temp = f'rule family="ipv4" source address="{TEST_IP}/32" port port="5060" protocol="udp" accept'
    pid = run('systemctl','show','asterisk','-p','MainPID').stdout.strip()
    # Existing native logger only; never enable protocol debug, which could log
    # Authorization headers. Retain event counts, discard every raw log line.
    log = Path('/var/log/asterisk/messages')
    start = json.loads((BACKUP/'asterisk-log-offset.json').read_text())
    with log.open('rb') as stream:
        stream.seek(start['offset'] if log.stat().st_ino == start['inode'] else 0)
        notices = stream.read(2*1024*1024).decode(errors='replace')
    native_events = {'legacy_registered':len(re.findall(r'Registered SIP ',notices)),
        'pjsip_contact_added':len(re.findall(r'Added contact ',notices)),
        'pjsip_contact_removed':len(re.findall(r'Removed contact |Deleted contact ',notices))}
    return dict(at=now, checksums=checksums, unrelated_grants_preserved=unrelated_preserved,
        phone_ports=list(PORTS), effective_web_IPs=len(web), effective_phone_IPs=len(phone),
        owned_rule_counts=owned_counts, test_network=dict(web=TEST_IP in web, phone=TEST_IP in phone,
            owned_rules=test_owned, native_phone=run('nft','get','element','inet',TABLE,'agents4','{',TEST_IP,'}').returncode==0),
        temporary_test_rule_present=run('firewall-cmd','--zone=public','--query-rich-rule='+temp).returncode==0,
        permanent_rich_rules=len([v for v in permanent.splitlines() if v.strip()]),
        runtime_unprioritized_SIP_accepts=len([v for v in runtime.splitlines() if 'priority=' not in v and 'accept' in v and re.search(r'port="506[01]"',v)]),
        global_ports=run('firewall-cmd','--zone=public','--list-ports').stdout.strip(),
        permanent_ports=run('firewall-cmd','--permanent','--zone=public','--list-ports').stdout.strip(),
        guard_hooks={v['chain']['name']:v['chain']['prio'] for v in nft if 'chain' in v and 'prio' in v['chain']},
        guard_policy_chains=sorted({v['chain'] for v in rules if v.get('comment')==POLICY}),
        guard_rule_count=len(rules),
        guard_SIP_drop_counters={chain:{port:sum(next((e['counter']['packets'] for e in v['expr'] if 'counter' in e),0)
            for v in rules if v['chain']==chain and any('drop' in e for e in v['expr']) and
            any(e.get('match',{}).get('right')==int(port) for e in v['expr'])) for port in ('5060','5061')}
            for chain in ('input','output')},
        services={name:run('systemctl','is-active',name).stdout.strip() for name in
            ('alma-portal-443.service','alma-portal-443-apply.timer','alma-portal-sip-guard.service','asterisk','httpd','mariadb','firewalld','fail2ban','sshd')},
        asterisk_pid_unchanged=pid==json.loads((BACKUP/'baseline.json').read_text())['asterisk_pid'],
        native_logger_events_since_observer_start=native_events,
        asterisk_channels=run('asterisk','-rx','core show channels count').stdout.strip(),
        sip_listeners={port:bool(run('ss','-Hlun','sport = :'+port).stdout.strip()) for port in ('5060','5061')},
        ssh_22222_listener=bool(run('ss','-Hlnt','sport = :22222').stdout.strip()),
        legacy_9091_listener=bool(run('ss','-Hlnt','sport = :9091').stdout.strip()),
        cockpit_9090_listener=bool(run('ss','-Hlnt','sport = :9090').stdout.strip()),
        apache_syntax_rc=run('httpd','-t').returncode, selinux=run('getenforce').stdout.strip())


def sanitize(value):
    if isinstance(value,str):
        value='\n'.join(s for s in value.splitlines() if not s.startswith('AH00558:'))
        value=re.sub(r'\b(?:\d{1,3}\.){3}\d{1,3}\b','REDACTED_IP',value)
        return re.sub(r'(?<![A-Za-z0-9])/(?:[^\s"<>]+)',lambda m:Path(m[0]).name,value)
    if isinstance(value,dict):return {sanitize(k):sanitize(v) for k,v in value.items()}
    if isinstance(value,list):return [sanitize(v) for v in value]
    return value


if __name__=='__main__':
    if len(sys.argv)==3 and sys.argv[1]=='snapshot':
        name=sys.argv[2]
        assert name in ('deployed','no-grant','authorized','revoked','reauthorized','final')
        data=snapshot()
        (BACKUP/('health-'+name+'.json')).write_text(json.dumps(data,indent=2)+'\n')
        print(json.dumps(data,indent=2))
    elif sys.argv[1:]==['export']:
        result={'format':'sanitized-legacy-sip-evidence-v1'}
        for name in ('staged-tests.txt','guard-probe.txt','rollback-rehearsal.txt','rollback-hashes.txt','deploy.txt','office-probe.txt','personal-probe.txt','https-probe.txt','restart-check.txt'):
            result[name]=sanitize((BACKUP/name).read_text())
        for name in ('health-deployed.json','health-no-grant.json','health-authorized.json','health-revoked.json','health-reauthorized.json','health-final.json','register-counts.json',
                     'register-no-grant.json','register-authorized.json','register-revoked.json','register-reauthorized.json'):
            if (BACKUP/name).exists():result[name]=sanitize(json.loads((BACKUP/name).read_text()))
        text=json.dumps(result,indent=2)
        assert not re.search(r'\b(?:\d{1,3}\.){3}\d{1,3}\b',text)
        assert not any(v in text for v in ('/root/','/opt/','/var/','/etc/'))
        (BACKUP/'sanitized-evidence.json').write_text(text+'\n')
        print('SANITIZED_EVIDENCE_READY')
    else:raise SystemExit('Use snapshot PHASE or export')
