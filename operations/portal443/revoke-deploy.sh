#!/usr/bin/bash
# This task only: use after the private pre-change backup and isolated probe.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
stage=/root/alma-portal-revoke-stage
backup=/root/alma-portal-revoke-backup
test -s "$backup/config.tar"
test -x "$backup/rollback.sh"
tar -tf "$backup/config.tar" >/dev/null
trap 'rc=$?; trap - ERR; "$backup/rollback.sh"; exit "$rc"' ERR
systemctl stop alma-portal-443-apply.timer alma-portal-443-apply.service
for source in sip_guard.py phone_policy.py apply_requests.py; do
  install -m 0644 "$stage/$source" "/opt/alma-portal-443/$source"
done
for unit in alma-portal-sip-guard.service alma-portal-443-apply.service; do
  install -m 0644 "$stage/$unit" "/etc/systemd/system/$unit"
done
restorecon -R /opt/alma-portal-443
systemctl daemon-reload
systemd-analyze verify /etc/systemd/system/alma-portal-sip-guard.service \
  /etc/systemd/system/alma-portal-443-apply.service
systemctl enable --now alma-portal-sip-guard.service
# Column-only SELECT extension; no VICIdial data, schema or passwords change.
mysql -e "GRANT SELECT (vdc_agent_api_access,api_allowed_functions) ON asterisk.vicidial_users TO 'alma_portal_auth'@'localhost'"
systemctl start alma-portal-443-apply.service
nft list table inet alma_portal_sip >/dev/null
for source in access.lua portal.py; do
  install -m 0644 "$stage/$source" "/opt/alma-portal-443/$source"
done
install -m 0644 "$stage/02-alma-vicidial-ssl.conf" /etc/httpd/conf.d/02-alma-vicidial-ssl.conf
restorecon -R /opt/alma-portal-443 /etc/httpd/conf.d/02-alma-vicidial-ssl.conf
httpd -t
systemctl restart alma-portal-443.service
systemctl reload httpd
systemctl start alma-portal-443-apply.timer
systemctl is-active asterisk httpd firewalld fail2ban alma-portal-443.service alma-portal-sip-guard.service
echo REVOCATION_REDIRECT_DEPLOYED
