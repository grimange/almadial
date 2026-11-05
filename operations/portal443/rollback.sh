#!/usr/bin/bash
# Close public 443 BEFORE restoring the v2 Apache configuration.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-443-backup
tar -tf "$backup/config.tar" >/dev/null
firewall-cmd --zone=public --remove-port=443/tcp
firewall-cmd --permanent --zone=public --remove-port=443/tcp
firewall-cmd --zone=public --add-port=9091/tcp
firewall-cmd --permanent --zone=public --add-port=9091/tcp
systemctl disable --now alma-portal-443-apply.timer
systemctl stop alma-portal-443-apply.service
systemctl disable --now alma-portal-443.service
tar -C / -xpf "$backup/config.tar" \
  etc/httpd/conf.d/00-alma-acme.conf etc/httpd/conf.d/02-alma-vicidial-ssl.conf \
  etc/fail2ban/filter.d/alma-portal.conf etc/fail2ban/jail.d/alma-portal.local
httpd -t
systemctl reload httpd
fail2ban-client reload alma-portal
systemctl enable --now alma-portal.service alma-portal-apply.timer
systemctl is-active httpd alma-portal firewalld fail2ban asterisk
echo ROLLED_BACK_TO_9091
