#!/usr/bin/bash
# First phase: install HTTP boundary and verify it, keeping 9091 available.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-443-backup
test -s "$backup/config.tar"
tar -tf "$backup/config.tar" >/dev/null
test -x "$backup/rollback.sh"
systemctl is-active --quiet alma-portal-443.service
# Roll back automatically on any failure before successful activation.
trap 'bash /root/alma-portal-443-backup/rollback.sh' ERR
for config in 00-alma-acme.conf 02-alma-vicidial-ssl.conf; do
  install -m 0644 "/root/alma-portal-443-stage/$config" "/etc/httpd/conf.d/$config"
done
httpd -t
systemctl reload httpd
/opt/alma-portal-venv/bin/python /root/alma-portal-443-stage/probe.py
install -m 0644 /root/alma-portal-443-stage/alma-portal.local /etc/fail2ban/jail.d/alma-portal.local
install -m 0644 /root/alma-portal-443-stage/alma-portal.conf /etc/fail2ban/filter.d/alma-portal.conf
fail2ban-client -t
fail2ban-client reload alma-portal
systemctl enable --now alma-portal-443.service alma-portal-443-apply.timer
systemctl disable --now alma-portal-apply.timer
systemctl stop alma-portal-apply.service
# The authorization boundary has been verified before this public exposure.
firewall-cmd --zone=public --add-port=443/tcp
firewall-cmd --permanent --zone=public --add-port=443/tcp
echo ACTIVATED_443_LEGACY_9091_RETAINED_FOR_VERIFICATION
