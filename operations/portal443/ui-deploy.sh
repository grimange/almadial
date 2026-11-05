#!/usr/bin/bash
# Bounded UI-only update of the already verified 443 deployment. Run via SSH.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
stage=/root/alma-portal-ui-stage
backup=/root/alma-portal-ui-backup
test ! -e "$backup"
test -f /opt/alma-portal-443/portal.py
test -r /var/www/html/vicidial/images/vicidial_admin_web_logo.png
/opt/alma-portal-venv/bin/python -m unittest discover -s "$stage/tests" -v
install -d -m 0700 "$backup"
tar -C / -cpf "$backup/config.tar" opt/alma-portal-443 etc/httpd/conf.d/02-alma-vicidial-ssl.conf
chmod 0600 "$backup/config.tar"
sha256sum /opt/alma-portal-443/*.py /opt/alma-portal-443/*.js /opt/alma-portal-443/*.lua \
  /etc/httpd/conf.d/02-alma-vicidial-ssl.conf > "$backup/before.sha256"
sha256sum /var/www/html/vicidial/admin.php /var/www/html/vicidial/functions.php \
  /var/www/html/vicidial/images/vicidial_admin_web_logo.png \
  /etc/asterisk/pjsip.conf /etc/asterisk/extensions.conf \
  /etc/firewalld/zones/public.xml /etc/fail2ban/filter.d/alma-portal.conf \
  /etc/fail2ban/jail.d/alma-portal.local /opt/alma-portal-443/vicidial_auth.py \
  /opt/alma-portal-443/common.py /opt/alma-portal-443/apply_requests.py > "$backup/protected.sha256"
systemctl show -p MainPID -p ActiveEnterTimestamp asterisk > "$backup/asterisk-before.txt"
install -m 0700 "$stage/ui-rollback.sh" "$backup/rollback.sh"
tar -tf "$backup/config.tar" >/dev/null
trap 'rc=$?; trap - ERR; bash "$backup/rollback.sh"; exit "$rc"' ERR
for source in app.css portal.py app.js access.lua; do
  install -o root -g root -m 0644 "$stage/$source" "/opt/alma-portal-443/$source"
done
install -o root -g root -m 0644 "$stage/02-alma-vicidial-ssl.conf" /etc/httpd/conf.d/02-alma-vicidial-ssl.conf
restorecon -R /opt/alma-portal-443
httpd -t
systemctl restart alma-portal-443.service
systemctl reload httpd.service
# A fresh process must be serving before network probes (is-active is insufficient).
for attempt in {1..15}; do
  if [[ $(ss -H -lnt 'sport = :19091') == *127.0.0.1:19091* ]]; then break; fi
  sleep 1
done
/opt/alma-portal-venv/bin/python "$stage/probe.py"
/opt/alma-portal-venv/bin/python "$stage/form_probe.py"
sha256sum -c "$backup/protected.sha256"
systemctl show -p MainPID -p ActiveEnterTimestamp asterisk > "$backup/asterisk-after.txt"
cmp "$backup/asterisk-before.txt" "$backup/asterisk-after.txt"
systemctl is-active alma-portal-443.service httpd.service asterisk.service firewalld.service fail2ban.service
sha256sum /opt/alma-portal-443/*.py /opt/alma-portal-443/*.js /opt/alma-portal-443/*.css \
  /opt/alma-portal-443/*.lua /etc/httpd/conf.d/02-alma-vicidial-ssl.conf > "$backup/after.sha256"
trap - ERR
echo UI_DEPLOYMENT_AND_SECURITY_PROBES_PASS
