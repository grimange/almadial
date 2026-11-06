#!/usr/bin/bash
# Existing private backup and tested rollback required. No unrelated restarts.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
stage=/root/alma-portal-5060-stage
backup=/root/alma-portal-5060-backup
test -s "$backup/config.tar"
test -x "$backup/rollback.sh"
/opt/alma-portal-venv/bin/python -m py_compile "$stage/"{common,apply_requests,sip_guard}.py
trap 'rc=$?; trap - ERR; "$backup/rollback.sh"; exit "$rc"' ERR
systemctl stop alma-portal-443-apply.timer alma-portal-443-apply.service
for source in common.py apply_requests.py sip_guard.py; do
  install -m 0644 "$stage/$source" "/opt/alma-portal-443/$source"
done
restorecon /opt/alma-portal-443/{common,apply_requests,sip_guard}.py
# Upgrade existing guard chains atomically BEFORE authorizing the additional port.
/opt/alma-portal-venv/bin/python /opt/alma-portal-443/sip_guard.py
# Remove only the operator-confirmed, unprioritized five-minute test rule.
test_rule='rule family="ipv4" source address="158.62.74.253/32" port port="5060" protocol="udp" accept'
if firewall-cmd --zone=public --query-rich-rule="$test_rule" >/dev/null; then
  firewall-cmd --zone=public --remove-rich-rule="$test_rule"
  touch "$backup/temporary_test_rule_removed"
fi
systemctl start alma-portal-443-apply.service
systemctl start alma-portal-443-apply.timer
systemctl is-active alma-portal-443.service alma-portal-sip-guard.service firewalld fail2ban asterisk httpd mariadb sshd
echo LEGACY_SIP_DEPLOYED
