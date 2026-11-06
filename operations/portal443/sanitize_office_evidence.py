#!/usr/bin/env python3
"""Produce a whitelisted, sanitized export on the server. Keep raw files private."""
import json
from pathlib import Path
import re

BACKUP = Path('/root/alma-portal-office-backup')
LOGS = ('staged-tests.txt', 'deploy.txt', 'active-office-rollback.txt',
        'rollback-hashes.txt', 'office-probe-final.txt', 'personal-probe-final.txt',
        'https-probe.txt', 'guard-probe.txt', 'unit-verify.txt')
HEALTH = ('health-office-active.json', 'health-final.json')
IPV4 = re.compile(r'\b(?:\d{1,3}\.){3}\d{1,3}\b')
ABSOLUTE = re.compile(r'(?<![A-Za-z0-9])/(?:[^\s"<>]+)')


def sanitize(value):
    if isinstance(value, str):
        value = '\n'.join(line for line in value.splitlines() if not line.startswith('AH00558:'))
        value = IPV4.sub('REDACTED_IP', value)
        value = ABSOLUTE.sub(lambda m: Path(m[0]).name, value)
        return value.replace('dialer01', 'TARGET_HOST')
    if isinstance(value, dict):
        return {sanitize(key): sanitize(item) for key,item in value.items()}
    if isinstance(value, list):
        return [sanitize(item) for item in value]
    return value


def main():
    result = {'format': 'sanitized-office-evidence-v1',
              'redaction': 'No client IPs, usernames, grant records, tokens, credentials or absolute server paths.'}
    for name in LOGS:
        result[name] = sanitize((BACKUP/name).read_text())
    for name in HEALTH:
        result[name] = sanitize(json.loads((BACKUP/name).read_text()))
    text = json.dumps(result, indent=2)
    assert not IPV4.search(text)
    for prefix in ('/root/', '/opt/', '/var/', '/etc/'):
        assert prefix not in text
    # Original sources and raw state/firewall files are never included.
    path = BACKUP/'sanitized-office-evidence.json'
    path.write_text(text + '\n')
    path.chmod(0o600)
    print('SANITIZED_OFFICE_EVIDENCE_READY')


if __name__ == '__main__':
    main()
