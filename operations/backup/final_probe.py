#!/usr/bin/python3
"""Final bounded verification; all extraction is synthetic rollback rehearsal.

Verifies real archives by streaming only. Does not restore production data.
"""
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile

assert os.geteuid() == 0 and socket.gethostname() == 'dialer01'
base = Path('/var/lib/alma-vicidial-backup')
result = {}
result['archive_verifications'] = []
for path in sorted((base / 'complete').iterdir()):
    p = subprocess.run(['/usr/local/sbin/alma-vicidial-backup', 'verify', str(path)],
                       capture_output=True, text=True, check=True, timeout=120)
    result['archive_verifications'].append(dict(id=path.name, result=json.loads(p.stdout)))
result['health'] = json.loads(subprocess.run(['/usr/local/sbin/alma-vicidial-backup', 'health'],
    capture_output=True, text=True, check=True).stdout)
latest = base / 'complete' / result['health']['last_success']['id']
privacy = subprocess.run(['runuser', '-u', 'nobody', '--', 'test', '-r', str(latest / 'database.sql')])
assert privacy.returncode == 1
result['unprivileged_read_denied_rc'] = privacy.returncode
rollback = sorted(Path('/root').glob('alma-vicidial-backup-deploy-*'))[-1]
original_state = json.loads((rollback / 'original-state.json').read_text())
real_hashes = {p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in original_state}
with tempfile.TemporaryDirectory(prefix='rehearsal-', dir=rollback) as directory:
    root = Path(directory)
    for path in original_state:
        copy = root / path.lstrip('/')
        copy.parent.mkdir(parents=True, exist_ok=True)
        copy.write_text('synthetic component')
    keep = root / 'var/lib/alma-vicidial-backup/complete/synthetic-good'
    keep.parent.mkdir(parents=True)
    keep.write_text('preserve archive')
    script = (rollback / 'rollback.sh').read_text()
    for path in original_state:
        script = script.replace(path, str(root / path.lstrip('/')))
    stub = root / 'stub'
    stub.mkdir()
    (stub / 'systemctl').write_text('#!/usr/bin/bash\nif test "$1" = is-active; then exit 3; fi\nexit 0\n')
    (stub / 'systemctl').chmod(0o700)
    candidate = root / 'rollback-rehearsal.sh'
    candidate.write_text(script)
    subprocess.run(['bash', '-n', str(candidate)], check=True)
    p = subprocess.run(['bash', str(candidate)], env={**os.environ, 'PATH':str(stub)+':'+os.environ['PATH']},
                       capture_output=True, text=True, check=True)
    assert all(not (root / path.lstrip('/')).exists() for path in original_state)
    assert keep.read_text() == 'preserve archive'
assert real_hashes == {p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in original_state}
result['rollback_rehearsal'] = dict(result='PASS', scope='synthetic files plus stubbed systemctl; not a live rollback or data restore',
                                    path=str(rollback), installed_hashes=real_hashes)
for name, args in {
    'journal': ['journalctl', '-u', 'alma-vicidial-backup.service', '-n', '45', '--no-pager'],
    'unit_validation': ['systemd-analyze', 'verify', '/etc/systemd/system/alma-vicidial-backup.service', '/etc/systemd/system/alma-vicidial-backup.timer'],
    'schedule': ['systemd-analyze', 'calendar', '*-*-* 03:40:00 America/New_York'],
    'module': ['asterisk', '-rx', 'module show like res_monitor.so'],
    'active_calls': ['asterisk', '-rx', 'core show channels count'],
    'backup_directory': ['stat', '-c', '%a %U:%G %n', str(base), str(base / 'complete'), str(base / 'status.json')]
}.items():
    p = subprocess.run(args, capture_output=True, text=True, timeout=25)
    result[name] = dict(rc=p.returncode, stdout=p.stdout, stderr=p.stderr)
print(json.dumps(result, indent=2))
