#!/usr/bin/bash
# This task only. Never restore old firewall XML, SSH or a stale state snapshot.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-office-backup
tar -tf "$backup/config.tar" >/dev/null
systemctl stop alma-portal-443.service alma-portal-443-apply.timer alma-portal-443-apply.service
if test -f "$backup/office_column_added"; then
  /opt/alma-portal-venv/bin/python - <<'PY'
import json, sys, time
sys.path.insert(0, '/opt/alma-portal-443')
from common import BASE, QUEUE, atomic_json
from apply_requests import process
now = int(time.time())
state = json.loads((BASE/'state.json').read_text())
for lease in state.get('leases', {}).values():
    if lease.get('grant_type') == 'office':
        lease['expires_at'] = min(lease['expires_at'], now)
        lease['revoked'] = True
for identity, at in list(state.get('revoked_before', {}).items()):
    if identity.endswith(':personal'):
        old = identity.removesuffix(':personal')
        state['revoked_before'][old] = max(state['revoked_before'].get(old, 0), at)
for path in QUEUE.glob('*.request'):
    if json.loads(path.read_text()).get('grant_type') == 'office':
        path.unlink()
atomic_json(BASE/'state.json', state, 0o600)
process(state, now)
PY
fi
for source in common.py phone_policy.py apply_requests.py portal.py app.css; do
  tar -C / -xpf "$backup/config.tar" "opt/alma-portal-443/$source"
done
if test -f "$backup/office_column_added"; then
  mysql -e "REVOKE SELECT (modify_users) ON asterisk.vicidial_users FROM 'alma_portal_auth'@'localhost'"
  rm "$backup/office_column_added"
fi
restorecon -R /opt/alma-portal-443
httpd -t
systemctl start alma-portal-443-apply.service alma-portal-443-apply.timer alma-portal-443.service
systemctl is-active alma-portal-443.service firewalld alma-portal-sip-guard.service
echo SHARED_OFFICE_ROLLBACK_COMPLETE
