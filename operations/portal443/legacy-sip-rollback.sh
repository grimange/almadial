#!/usr/bin/bash
# Restore pre-5060 portal source without restoring stale grants or firewall state.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-5060-backup
test -s "$backup/config.tar"
systemctl stop alma-portal-443-apply.timer alma-portal-443-apply.service
for source in common.py apply_requests.py sip_guard.py; do
  tar -C / -xpf "$backup/config.tar" "opt/alma-portal-443/$source"
done
/opt/alma-portal-venv/bin/python - <<'PY'
import json,re,sys
sys.path.insert(0, '/opt/alma-portal-443')
import sip_guard as guard
from common import BASE
from apply_requests import firewall, rule
# Preserve live timed sets, but keep5060 unconditionally denied on rollback:
# existing5060 conntrack must not survive removal of the dynamic rich rules.
template = guard.template().replace('iifname "lo" return', 'iifname "lo" return\nudp dport 5060 counter drop').replace('oifname "lo" return', 'oifname "lo" return\nudp sport 5060 counter drop')
if guard.nft('list', 'table', 'inet', guard.TABLE).returncode == 0:
    lines=[]
    for chain in ('input','output'):
        lines.append(f'flush chain inet {guard.TABLE} {chain}')
        body=re.search(r'chain '+chain+r' \{(.*?)\}',template,re.S)[1]
        lines.extend(f'add rule inet {guard.TABLE} {chain} {line.strip()}'
            for line in body.splitlines() if line.strip() and not line.strip().startswith('type filter'))
    result=guard.nft('-f','-',script='\n'.join(lines))
else:
    result=guard.nft('-f','-',script=template)
assert result.returncode == 0, 'Rollback guard update failed'
state=json.loads((BASE/'state.json').read_text())
ips=set(state.get('owned',[])) | set(state.get('synced',{})) | {v['ip'] for v in state.get('leases',{}).values()}
for ip in ips:
    value=rule(ip,'5060')
    status=firewall('--query-rich-rule='+value)
    assert status in (0,1)
    if status==0: assert firewall('--remove-rich-rule='+value)==0
print('ROLLBACK_5060_DENIED_5061_AND_RTP_PRESERVED')
PY
restorecon /opt/alma-portal-443/{common,apply_requests,sip_guard}.py
systemctl start alma-portal-443-apply.service
systemctl start alma-portal-443-apply.timer
systemctl is-active alma-portal-443.service alma-portal-sip-guard.service firewalld asterisk sshd
echo LEGACY_SIP_ROLLBACK_COMPLETE
