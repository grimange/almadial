#!/usr/bin/env python3
"""Read-only redacted live evidence; never export raw state/DB config/firewall IPs."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tarfile
import time
sys.path.insert(0, '/opt/alma-portal-443')
from common import BASE
from apply_requests import aggregate, rule, PORTS

BACKUP = Path('/root/alma-portal-office-backup')


def run(*argv):
    return subprocess.run(argv, capture_output=True, text=True, timeout=30)


def main():
    now = int(time.time())
    state = json.loads((BASE/'state.json').read_text())
    with tarfile.open(BACKUP/'config.tar') as archive:
        before = json.load(archive.extractfile('var/lib/alma-portal-443/state.json'))
    preserved = all(k in state['leases'] and state['leases'][k]['expires_at'] == v['expires_at']
        and state['leases'][k].get('revoked') == v.get('revoked')
        for k,v in before.get('leases', {}).items() if v['ip'] != '2.29.153.161')
    changes = []
    unchanged = []
    hashes = {}
    for line in (BACKUP/'before.sha256').read_text().splitlines():
        old, name = line.split(maxsplit=1)
        digest = hashlib.sha256(Path(name).read_bytes()).hexdigest()
        hashes[name] = {'before': old, 'after': digest}
        (unchanged if old == digest else changes).append(name)
    web, sip = aggregate(state['leases'], now)
    rules_present = all(run('firewall-cmd', '--zone=public', '--query-rich-rule=' + rule(ip,port)).returncode == 0
                        for ip in sip for port in PORTS)
    runtime = run('firewall-cmd', '--zone=public', '--list-rich-rules').stdout.strip().splitlines()
    permanent = run('firewall-cmd', '--permanent', '--zone=public', '--list-rich-rules').stdout.strip().splitlines()
    nft = json.loads(run('nft', '-j', 'list', 'table', 'inet', 'alma_portal_sip').stdout)['nftables']
    agent_elements = []
    priorities = {}
    for item in nft:
        if 'chain' in item and 'prio' in item['chain']:
            priorities[item['chain']['name']] = item['chain']['prio']
        if item.get('set', {}).get('name') == 'agents4':
            agent_elements = item['set'].get('elem', [])
    office_networks = {v['ip'] for v in state['leases'].values()
        if v.get('grant_type') == 'office' and v['ip'] != '2.29.153.161'}
    office_enforcement = dict(
        historical_office_network_count=len(office_networks),
        networks_with_effective_web=sum(ip in web for ip in office_networks),
        networks_with_effective_phone=sum(ip in sip for ip in office_networks),
        networks_with_native_web_file=sum((BASE/'web'/ip).exists() for ip in office_networks),
        networks_with_native_phone_member=sum(run('nft','get','element','inet','alma_portal_sip','agents4','{',ip,'}').returncode == 0 for ip in office_networks),
        networks_with_native_UDP_rule=sum(any(run('firewall-cmd','--zone=public','--query-rich-rule='+rule(ip,port)).returncode == 0 for port in PORTS) for ip in office_networks))
    services = {name:run('systemctl', 'is-active', name).stdout.strip() for name in
        ('alma-portal-443.service', 'alma-portal-443-apply.timer', 'alma-portal-sip-guard.service',
         'asterisk', 'httpd', 'mariadb', 'sshd', 'firewalld', 'fail2ban')}
    pid = run('systemctl', 'show', 'asterisk', '-p', 'MainPID').stdout.strip()
    baseline_pid = json.loads((BACKUP/'baseline.json').read_text())['asterisk_pid']
    output = dict(at=now, office_network_aggregate=office_enforcement, baseline_grant_expiries_revocations_preserved=preserved,
        changed_files=changes, unchanged_files=unchanged, checksums=hashes,
        active_grants_by_type={kind:sum(1 for v in state['leases'].values()
             if v.get('grant_type','personal')==kind and v['expires_at']>now and v.get('eligible') and not v.get('revoked'))
             for kind in ('personal','office')},
        effective_web_IP_count=len(web), effective_phone_IP_count=len(sip),
        expected_UDP_rule_count=2*len(sip), runtime_rich_rule_count=len(runtime),
        effective_rules_present=rules_present, permanent_rich_rule_count=len(permanent),
        guard_agent_element_count=len(agent_elements), guard_hook_priorities=priorities,
        firewall_ports=run('firewall-cmd','--zone=public','--list-ports').stdout.strip(),
        services=services, asterisk_pid=pid, asterisk_pid_unchanged=pid==baseline_pid,
        asterisk_version=run('asterisk','-rx','core show version').stdout.strip(),
        asterisk_channels=run('asterisk','-rx','core show channels count').stdout.strip(),
        selinux=run('getenforce').stdout.strip(),
        apache_syntax=run('httpd','-t').returncode,
        ssh_22222_listener=run('ss','-Hln','sport = :22222').stdout.strip()!='',
        legacy_9091_listener=run('ss','-Hln','sport = :9091').stdout.strip()!='',
        certificate=run('openssl','x509','-in','/etc/letsencrypt/live/2.29.153.161/fullchain.pem',
                        '-noout','-dates','-ext','subjectAltName').stdout.strip())
    print(json.dumps(output, indent=2))


if __name__ == '__main__':
    main()
