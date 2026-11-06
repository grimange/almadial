#!/usr/bin/python3
"""Initial PR-02 deployment only, to the explicitly authorized test VPS."""
import base64
import hashlib
import json
from pathlib import Path
import subprocess

source = Path(__file__).parent
files = [
    ('backup.py', '/usr/local/sbin/alma-vicidial-backup', 0o700),
    ('config.json', '/etc/alma-vicidial-backup.json', 0o600),
    ('alma-vicidial-backup.service', '/etc/systemd/system/alma-vicidial-backup.service', 0o644),
    ('alma-vicidial-backup.timer', '/etc/systemd/system/alma-vicidial-backup.timer', 0o644)]
payload = {target: dict(data=base64.b64encode((source / name).read_bytes()).decode(), mode=mode)
           for name, target, mode in files}
remote = r"""
import base64, datetime, hashlib, json, os
from pathlib import Path
import socket, subprocess
assert os.geteuid() == 0 and socket.gethostname() == 'dialer01'
assert all(not Path(p).exists() for p in payload), 'Already installed; inspect instead of overwriting'
assert not Path('/var/lib/alma-vicidial-backup').exists(), 'Unexpected backup data; inspect first'
os.umask(0o077)
rollback = Path('/root/alma-vicidial-backup-deploy-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
rollback.mkdir(mode=0o700)
# All four destination files were absent; rollback removes only these new components.
(rollback / 'original-state.json').write_text(json.dumps({p: 'absent' for p in payload}, indent=2))
script = '''#!/usr/bin/bash
set -euo pipefail
test "$(id -u)" = 0
test "$(hostname)" = dialer01
if systemctl is-active --quiet alma-vicidial-backup.service; then
  echo 'Backup running: wait for completion before rollback' >&2
  exit 1
fi
systemctl disable --now alma-vicidial-backup.timer
rm -f -- /usr/local/sbin/alma-vicidial-backup /etc/alma-vicidial-backup.json \\
  /etc/systemd/system/alma-vicidial-backup.service /etc/systemd/system/alma-vicidial-backup.timer
systemctl daemon-reload
echo 'PR02 components removed; all backup archives and status files retained'
'''
(rollback / 'rollback.sh').write_text(script)
(rollback / 'rollback.sh').chmod(0o700)
subprocess.run(['bash','-n',str(rollback / 'rollback.sh')], check=True)
for target,item in payload.items():
    p = Path(target)
    new = p.with_name(p.name + '.pr02-new')
    new.write_bytes(base64.b64decode(item['data']))
    new.chmod(item['mode'])
    os.replace(new,p)
subprocess.run(['restorecon',*payload],check=True)
compile(Path('/usr/local/sbin/alma-vicidial-backup').read_text(), '/usr/local/sbin/alma-vicidial-backup', 'exec')
subprocess.run(['systemd-analyze','verify','/etc/systemd/system/alma-vicidial-backup.service','/etc/systemd/system/alma-vicidial-backup.timer'],check=True)
subprocess.run(['systemctl','daemon-reload'],check=True)
# Enable after the controlled first backup; do not start a backup on installation.
print(json.dumps(dict(rollback=str(rollback), installed={p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in payload}),indent=2))
"""
subprocess.run(['ssh','-p','22222','-o','BatchMode=yes','root@2.29.153.161','python3','-B','-'],
               input='payload = ' + repr(payload) + '\n' + remote,text=True,check=True)
