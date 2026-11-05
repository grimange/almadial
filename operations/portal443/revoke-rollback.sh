#!/usr/bin/bash
# Restores this task's exact pre-change config, including its known UDP
# conntrack revocation limitation. Never restore old migration firewall grants.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-revoke-backup
tar -tf "$backup/config.tar" >/dev/null
systemctl stop alma-portal-443-apply.timer alma-portal-443-apply.service
systemctl disable alma-portal-sip-guard.service 2>/dev/null || true
tar -C / -xpf "$backup/config.tar" opt/alma-portal-443 \
  etc/httpd/conf.d/02-alma-vicidial-ssl.conf \
  etc/systemd/system/alma-portal-443-apply.service
rm -f /etc/systemd/system/alma-portal-sip-guard.service /opt/alma-portal-443/sip_guard.py /opt/alma-portal-443/phone_policy.py
# Remove Requires links and reload BEFORE stopping an active guard: stopping a
# required unit while dependents still reference it could stop Asterisk.
systemctl daemon-reload
systemctl stop alma-portal-sip-guard.service 2>/dev/null || true
if nft list table inet alma_portal_sip >/dev/null 2>&1; then
  nft delete table inet alma_portal_sip
fi
httpd -t
systemctl restart alma-portal-443.service
systemctl reload httpd
systemctl start alma-portal-443-apply.service alma-portal-443-apply.timer
python3 - <<'PY'
import subprocess
sql="SELECT COLUMN_NAME FROM information_schema.COLUMN_PRIVILEGES WHERE TABLE_SCHEMA='asterisk' AND TABLE_NAME='vicidial_users' AND GRANTEE=CONCAT(CHAR(39),'alma_portal_auth',CHAR(39),'@',CHAR(39),'localhost',CHAR(39)) AND PRIVILEGE_TYPE='SELECT'"
columns=subprocess.run(['mysql','-NBe',sql],capture_output=True,text=True,check=True).stdout.splitlines()
for column in ('vdc_agent_api_access','api_allowed_functions'):
    if column in columns:
        subprocess.run(['mysql','-e',"REVOKE SELECT ("+column+") ON asterisk.vicidial_users FROM 'alma_portal_auth'@'localhost'"],check=True)
PY
echo REVOCATION_REDIRECT_ROLLBACK_COMPLETE
