#!/usr/bin/python3
"""Explicit controlled PR-02 failure acceptance; SSH stdin on test VPS only.

Requires a complete backup and an inactive backup timer. Changes only the new
backup config temporarily; always restores its exact original bytes and mode.
No database lock is acquired by the injected missing-dataset failure.
"""
import fcntl
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess

assert os.geteuid() == 0 and socket.gethostname() == 'dialer01'
base = Path('/var/lib/alma-vicidial-backup')
config = Path('/etc/alma-vicidial-backup.json')
original = config.read_bytes()
state = json.loads((base / 'status.json').read_text())
assert state['outcome'] == 'ok'
assert subprocess.run(['systemctl', 'is-active', '--quiet', 'alma-vicidial-backup.timer']).returncode != 0
before = {p.name: hashlib.sha256((p / 'SHA256SUMS').read_bytes()).hexdigest()
          for p in (base / 'complete').iterdir()}
result = dict(prior_success=state['last_success']['id'], complete_before=before)
with (base / 'job.lock').open('a') as lock:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    attempt = subprocess.run(['systemctl', 'start', 'alma-vicidial-backup.service'], capture_output=True, text=True)
    result['overlap_rejected_rc'] = attempt.returncode
    assert attempt.returncode != 0
    assert json.loads((base / 'status.json').read_text()) == state
injected = json.loads(original)
missing = '/run/pr02-intentionally-missing-required-dataset'
assert not Path(missing).exists()
injected['required'].append(missing)
try:
    temporary = config.with_suffix('.negative')
    temporary.write_text(json.dumps(injected))
    temporary.chmod(0o600)
    os.replace(temporary, config)
    p = subprocess.run(['systemctl', 'start', 'alma-vicidial-backup.service'], capture_output=True, text=True)
    result['injected_failure_rc'] = p.returncode
    assert p.returncode != 0
    result['failed_state'] = json.loads((base / 'status.json').read_text())
    assert result['failed_state']['last_failure']['reason'] == 'required_dataset_missing'
    assert result['failed_state']['last_success'] == state['last_success']
    health = subprocess.run(['/usr/local/sbin/alma-vicidial-backup', 'health'], capture_output=True, text=True)
    result['health_failure_rc'] = health.returncode
    assert health.returncode == 1
    result['systemd_failed'] = subprocess.run(['systemctl', 'show', 'alma-vicidial-backup.service',
        '-p', 'Result', '-p', 'ExecMainStatus', '-p', 'ActiveState'], capture_output=True, text=True).stdout
    assert 'ActiveState=failed' in result['systemd_failed']
finally:
    temporary = config.with_suffix('.restore')
    temporary.write_bytes(original)
    temporary.chmod(0o600)
    os.replace(temporary, config)
assert config.read_bytes() == original
result['complete_after'] = {p.name: hashlib.sha256((p / 'SHA256SUMS').read_bytes()).hexdigest()
                            for p in (base / 'complete').iterdir()}
assert result['complete_after'] == before
assert not list(base.glob('.partial-*'))
verify = subprocess.run(['/usr/local/sbin/alma-vicidial-backup', 'verify',
                        str(base / 'complete' / state['last_success']['id'])], capture_output=True, text=True)
assert verify.returncode == 0
result['prior_backup_verified_after_failure'] = json.loads(verify.stdout)
result['original_config_restored_sha256'] = hashlib.sha256(config.read_bytes()).hexdigest()
print(json.dumps(result, indent=2))
