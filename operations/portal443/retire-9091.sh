#!/usr/bin/bash
# Run only after portal 443 checks, including operator browser confirmation.
set -euo pipefail
test "$(hostname)" = dialer01
test "$(id -u)" = 0
httpd -t
/opt/alma-portal-venv/bin/python /root/alma-portal-443-stage/probe.py
systemctl is-active --quiet alma-portal-443.service
firewall-cmd --zone=public --query-port=443/tcp
firewall-cmd --permanent --zone=public --query-port=443/tcp
firewall-cmd --zone=public --remove-port=9091/tcp
firewall-cmd --permanent --zone=public --remove-port=9091/tcp
systemctl disable --now alma-portal.service
echo RETIRED_9091
