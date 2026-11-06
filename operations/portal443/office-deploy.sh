#!/usr/bin/bash
# Staged task-specific deployment; backup/rollback must already exist.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
stage=/root/alma-portal-office-stage
backup=/root/alma-portal-office-backup
test -s "$backup/config.tar"
test -x "$backup/rollback.sh"
httpd -t
/opt/alma-portal-venv/bin/python -m py_compile "$stage/"{common,phone_policy,apply_requests,portal}.py
trap 'rc=$?; trap - ERR; "$backup/rollback.sh"; exit "$rc"' ERR
systemctl stop alma-portal-443.service alma-portal-443-apply.timer alma-portal-443-apply.service
# One extra read-only column; no schema, user data, or password changes.
mysql -e "GRANT SELECT (modify_users) ON asterisk.vicidial_users TO 'alma_portal_auth'@'localhost'"
touch "$backup/office_column_added"
for source in common.py phone_policy.py apply_requests.py portal.py app.css; do
  install -m 0644 "$stage/$source" "/opt/alma-portal-443/$source"
done
restorecon -R /opt/alma-portal-443
systemctl start alma-portal-443-apply.service
systemctl start alma-portal-443-apply.timer alma-portal-443.service
systemctl is-active alma-portal-443.service httpd firewalld fail2ban alma-portal-sip-guard.service asterisk mariadb sshd
httpd -t
echo SHARED_OFFICE_DEPLOYED
