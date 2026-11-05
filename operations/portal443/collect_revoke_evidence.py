#!/usr/bin/env python3
"""Read-only, secret-free config/health evidence for this task's named host."""
import hashlib
import json
from pathlib import Path
import subprocess
import time

BACKUP = Path('/root/alma-portal-revoke-backup')
assert subprocess.check_output(['hostname'], text=True).strip() == 'dialer01'


def run(*args):
    result = subprocess.run(args, text=True, capture_output=True, timeout=30)
    return {'rc': result.returncode, 'out': result.stdout.strip(), 'err': result.stderr.strip()}


changed = [Path('/opt/alma-portal-443')/n for n in (
    'sip_guard.py', 'phone_policy.py', 'apply_requests.py', 'portal.py', 'access.lua')]
changed += [Path('/etc/systemd/system')/n for n in (
    'alma-portal-sip-guard.service', 'alma-portal-443-apply.service')]
changed += [Path('/etc/httpd/conf.d/02-alma-vicidial-ssl.conf')]
protected = [Path(p) for p in (
    '/etc/asterisk/pjsip.conf', '/etc/asterisk/extensions.conf',
    '/var/www/html/vicidial/admin.php', '/var/www/html/vicidial/non_agent_api.php',
    '/var/www/html/vicidial/AST_timeonVDADall.php',
    '/opt/alma-portal-443/vicidial_auth.py', '/opt/alma-portal-443/common.py',
    '/etc/fail2ban/filter.d/alma-portal.conf', '/etc/fail2ban/jail.d/alma-portal.local',
    '/etc/firewalld/zones/public.xml')]
for name, files in (('after.sha256', changed), ('protected.sha256', protected)):
    (BACKUP/name).write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p)+'\n' for p in files))
# Record credential-config hashes privately; never export them with reports.
secret_files = [Path('/etc/alma-portal/vicidial-db.ini'), BACKUP/'db-service-config.before.ini']
(BACKUP/'private-credential-config.sha256').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p)+'\n' for p in secret_files))
(BACKUP/'private-credential-config.sha256').chmod(0o600)

health = {'at': int(time.time())}
for key, args in {
    'apache_config': ['httpd','-t'],
    'service_health': ['systemctl','is-active','asterisk','httpd','sshd','firewalld','fail2ban',
                       'alma-portal-443','alma-portal-443-apply.timer','alma-portal-sip-guard'],
    'asterisk_pid': ['systemctl','show','asterisk','-p','MainPID'],
    'calls': ['asterisk','-rx','core show channels count'],
    'guard_boot': ['systemctl','is-enabled','alma-portal-sip-guard','alma-portal-443-apply.timer'],
    'guard_order': ['systemctl','show','alma-portal-sip-guard','-p','Before'],
    'unit_syntax': ['systemd-analyze','verify','/etc/systemd/system/alma-portal-sip-guard.service',
                    '/etc/systemd/system/alma-portal-443-apply.service'],
    'ssh_22222': ['ss','-lnt','sport = :22222'],
    'legacy_9091': ['ss','-lnt','sport = :9091'],
    'cockpit_9090': ['ss','-lnt','sport = :9090'],
    'firewall_ports': ['firewall-cmd','--zone=public','--list-ports'],
    'firewall_permanent_ports': ['firewall-cmd','--permanent','--zone=public','--list-ports'],
    'permanent_rich': ['firewall-cmd','--permanent','--zone=public','--list-rich-rules'],
    'direct_rules': ['firewall-cmd','--direct','--get-all-rules'],
    'guard_exists': ['nft','list','tables'],
    'fail2ban': ['fail2ban-client','status'],
}.items():
    health[key] = run(*args)
state = json.loads(Path('/var/lib/alma-portal-443/state.json').read_text())
leases = [l for l in state.get('leases',{}).values() if l.get('eligible') and l['expires_at']>time.time()]
health['residential_valid_leases'] = sum(l['ip']=='158.62.74.253' for l in leases)
health['residential_phone_leases'] = sum(l['ip']=='158.62.74.253' and l.get('sip_enabled',l['role']=='agent') for l in leases)
health['other_valid_phone_leases'] = sum(l['ip']!='158.62.74.253' and l.get('sip_enabled',l['role']=='agent') for l in leases)
health['residential_web_file_exists'] = Path('/var/lib/alma-portal-443/web/158.62.74.253').exists()
health['residential_revoke_recorded'] = any(k.endswith('@158.62.74.253') for k in state.get('revoked_before', {}))
health['residential_runtime_rules_present'] = {}
for port in ('5061', '10000-20000'):
    value = 'rule priority="10" family="ipv4" source address="158.62.74.253/32" port port="'+port+'" protocol="udp" accept'
    result = run('firewall-cmd','--zone=public','--query-rich-rule='+value)
    assert result['rc'] in (0,1), 'Unable to confirm runtime rule state'
    health['residential_runtime_rules_present'][port] = result['rc'] == 0
health['guard_counters'] = run('nft','-j','list','table','inet','alma_portal_sip')
# Export counters only, omitting unrelated clients' IP addresses/set elements.
if health['guard_counters']['rc'] == 0:
    data = json.loads(health['guard_counters']['out'])
    counters = []
    health['residential_in_guard_sets'] = {}
    for item in data['nftables']:
        if 'set' in item:
            members = []
            for element in item['set'].get('elem', []):
                members.append(element['elem']['val'] if isinstance(element, dict) else element)
            health['residential_in_guard_sets'][item['set']['name']] = '158.62.74.253' in members
        rule = item.get('rule', {})
        for expression in rule.get('expr', []):
            if 'counter' in expression:
                counters.append({'chain':rule['chain'], **expression['counter']})
    health['guard_counters'] = counters
(BACKUP/'health.json').write_text(json.dumps(health, indent=2)+'\n')
print('SECRET_FREE_EVIDENCE_COLLECTED')
