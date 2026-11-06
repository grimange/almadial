#!/usr/bin/python3
"""Install only PR-08 files over the operator-authorized SSH target."""
import base64
import json
from pathlib import Path
import subprocess

source = Path(__file__).parent
payload = {target: dict(mode=mode, data=base64.b64encode((source / name).read_bytes()).decode())
           for name, target, mode in [
               ('alma_portal_tls.py', '/usr/local/libexec/alma-portal-tls', 0o755),
               ('alma-portal-cert-deploy', '/usr/local/sbin/alma-portal-cert-deploy', 0o755),
               ('alma-portal-renew.service', '/etc/systemd/system/alma-portal-renew.service', 0o644)]}
remote = r"""
import base64, datetime, hashlib, json, os
from pathlib import Path
import socket, subprocess, tarfile, tempfile

def run(*argv):
    subprocess.run(argv, check=True)

assert os.geteuid() == 0 and socket.gethostname() == 'dialer01'
assert subprocess.run(['systemctl', 'is-active', '--quiet', 'alma-portal-renew.service']).returncode != 0
assert not Path('/usr/local/libexec/alma-portal-tls').exists(), 'PR-08 already installed; inspect before updating'
os.umask(0o077)
backup = Path('/root/alma-portal-tls-backup-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
backup.mkdir(mode=0o700)
originals = [
    '/usr/local/sbin/alma-portal-cert-deploy',
    '/etc/systemd/system/alma-portal-renew.service',
    '/etc/systemd/system/alma-portal-renew.timer',
    '/etc/letsencrypt/renewal/2.29.153.161.conf',
    '/etc/httpd/conf.d/00-alma-acme.conf',
    '/etc/httpd/conf.d/02-alma-vicidial-ssl.conf']
manifest = {}
for name in originals:
    p = Path(name)
    s = p.stat()
    manifest[name] = dict(sha256=hashlib.sha256(p.read_bytes()).hexdigest(),
                          mode=s.st_mode & 0o777, uid=s.st_uid, gid=s.st_gid)
run('tar', '--acls', '--xattrs', '--selinux', '-C', '/', '-cpf', str(backup / 'originals.tar'),
    *[p.lstrip('/') for p in originals])
(backup / 'manifest.json').write_text(json.dumps(manifest, indent=2))
# Prove the archive can recover byte-identical originals with original permissions,
# without writing any restored file onto the running installation.
with tempfile.TemporaryDirectory(dir=backup) as extracted:
    run('tar', '--acls', '--xattrs', '--selinux', '-C', extracted, '-xpf', str(backup / 'originals.tar'))
    for name, expected in manifest.items():
        p = Path(extracted) / name.lstrip('/')
        s = p.stat()
        assert hashlib.sha256(p.read_bytes()).hexdigest() == expected['sha256']
        assert (s.st_mode & 0o777, s.st_uid, s.st_gid) == (expected['mode'], expected['uid'], expected['gid'])
rollback = '''#!/usr/bin/bash
set -euo pipefail
test "$(id -u)" = 0
test "$(hostname)" = dialer01
backup="$(cd -- "$(dirname -- "$0")" && pwd)"
if systemctl is-active --quiet alma-portal-renew.service; then
  echo 'Renewal is running; wait for it to finish before rollback' >&2
  exit 1
fi
# Restore only modified pre-existing files. Keep current Certbot certificate lineage,
# timer, firewall and all portal grant state. Preserve TLS status as incident evidence.
tar --acls --xattrs --selinux -C / -xpf "$backup/originals.tar" \\
  usr/local/sbin/alma-portal-cert-deploy etc/systemd/system/alma-portal-renew.service
rm -f /usr/local/libexec/alma-portal-tls
systemctl daemon-reload
httpd -t
systemctl is-active httpd alma-portal-renew.timer
echo 'PR08_ROLLBACK_COMPLETE: old failure-masking hook restored; investigate before next renewal'
'''
(backup / 'rollback.sh').write_text(rollback)
(backup / 'rollback.sh').chmod(0o700)
run('bash', '-n', str(backup / 'rollback.sh'))
print('ROLLBACK_ARCHIVE_REHEARSAL_PASS', str(backup), flush=True)
for name, item in payload.items():
    path = Path(name)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + '.pr08-new')
    temporary.write_bytes(base64.b64decode(item['data']))
    temporary.chmod(item['mode'])
    os.replace(temporary, path)
run('restorecon', *payload.keys())
run('bash', '-n', '/usr/local/sbin/alma-portal-cert-deploy')
compile(Path('/usr/local/libexec/alma-portal-tls').read_text(), '/usr/local/libexec/alma-portal-tls', 'exec')
run('systemd-analyze', 'verify', '/etc/systemd/system/alma-portal-renew.service')
run('systemctl', 'daemon-reload')
run('httpd', '-t')
print('TLS_DEPLOYMENT_INSTALLED', json.dumps({name: hashlib.sha256(Path(name).read_bytes()).hexdigest() for name in payload}), flush=True)
"""
code = 'payload = ' + repr(payload) + '\n' + remote
subprocess.run(['ssh', '-p', '22222', '-o', 'BatchMode=yes', 'root@2.29.153.161',
                '/usr/bin/python3', '-B', '-'], input=code, text=True, check=True)
