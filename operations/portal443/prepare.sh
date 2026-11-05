#!/usr/bin/bash
# Run only via SSH on the task's named host; no public listener/firewall changes.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
test -f /etc/httpd/conf.d/02-alma-vicidial-ssl.conf
backup=/root/alma-portal-443-backup
test ! -e "$backup"
install -d -m 0700 "$backup"
tar -C / -cpf "$backup/config.tar" etc/httpd etc/firewalld etc/fail2ban \
  etc/systemd/system/alma-portal.service etc/systemd/system/alma-portal-apply.service \
  etc/systemd/system/alma-portal-apply.timer opt/alma-portal
chmod 0600 "$backup/config.tar"
sha256sum /etc/httpd/conf.d/00-alma-acme.conf /etc/httpd/conf.d/02-alma-vicidial-ssl.conf \
  /etc/fail2ban/filter.d/alma-portal.conf /etc/fail2ban/jail.d/alma-portal.local \
  /opt/alma-portal/*.py > "$backup/before.sha256"
firewall-cmd --zone=public --list-all > "$backup/firewall-runtime.txt"
firewall-cmd --permanent --zone=public --list-all > "$backup/firewall-permanent.txt"
cp /root/alma-portal-443-stage/rollback.sh "$backup/rollback.sh"
chmod 0700 "$backup/rollback.sh"
tar -tf "$backup/config.tar" >/dev/null
install -d -m 0755 /opt/alma-portal-443 /var/lib/alma-portal-443 /var/lib/alma-portal-443/web
install -d -o root -g alma-portal -m 0750 /var/lib/alma-portal-443/status
install -d -o alma-portal -g alma-portal -m 0700 /var/lib/alma-portal/requests-443
for source in common.py portal.py vicidial_auth.py apply_requests.py app.js app.css access.lua; do
  install -o root -g root -m 0644 "/root/alma-portal-443-stage/$source" "/opt/alma-portal-443/$source"
done
for unit in alma-portal-443.service alma-portal-443-apply.service alma-portal-443-apply.timer; do
  install -m 0644 "/root/alma-portal-443-stage/$unit" "/etc/systemd/system/$unit"
done
restorecon -R /opt/alma-portal-443 /var/lib/alma-portal-443 /var/lib/alma-portal/requests-443
systemctl daemon-reload
systemd-analyze verify /etc/systemd/system/alma-portal-443.service /etc/systemd/system/alma-portal-443-apply.service /etc/systemd/system/alma-portal-443-apply.timer
/opt/alma-portal-venv/bin/python -m unittest discover -s /root/alma-portal-443-stage/tests -v
systemctl start alma-portal-443.service
systemctl start alma-portal-443-apply.service
systemctl is-active alma-portal-443.service
ss -lnt 'sport = :19091'
echo PREPARED_NO_PUBLIC_CHANGE
