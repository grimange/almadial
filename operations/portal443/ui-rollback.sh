#!/usr/bin/bash
# Restore the working 443 portal before the visual update; keep the 443 boundary.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
backup=/root/alma-portal-ui-backup
tar -tf "$backup/config.tar" >/dev/null
tar -C / -xpf "$backup/config.tar"
restorecon -R /opt/alma-portal-443
httpd -t
systemctl restart alma-portal-443.service
systemctl reload httpd.service
sha256sum -c "$backup/before.sha256"
echo UI_ROLLBACK_RESTORED_443_PORTAL
