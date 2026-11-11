#!/usr/bin/env python3
"""Read-only ALM-27 evidence index. Validation is not independent runtime QA.

Every mandatory case needs an attributable observation and hash-bound artifacts.
Never infer media, application controls or operator listening from a SIP handshake.
Private media and browser/session material belong outside the repository.
"""
import argparse
import hashlib
import json
from pathlib import Path


REQUIRED = (
    'trusted-browser', 'register', 'agent-login', 'agent-join', 'agent-dial',
    'agent-transfer', 'agent-hangup', 'agent-logout', 'conference-lifecycle',
    'ice-dtls-srtp', 'audio-browser-to-server', 'audio-server-to-browser',
    'recording-content', 'recording-correlation', 'reconnect-cleanup',
    'expired-certificate-rejected', 'operator-listening', 'fresh-no-op',
    'completed-state-reconciliation', 'baseline-preserved', 'reboot-persistence',
)
STATES = {'PASS', 'FAIL', 'BLOCKED', 'NOT_RUN'}


def template():
    return {'schema': 'almadial.m3-05.acceptance/v1',
            'topology': 'local/LAN', 'execution_id': '', 'git_head': '',
            'host_identity': '', 'browser': '', 'codec': '', 'duration_seconds': 0,
            'checks': {key: {'status': 'NOT_RUN', 'observation': '', 'artifacts': []}
                       for key in REQUIRED},
            'external_profiles': {'direct-public': 'NOT_RUN', 'fixed-nat': 'NOT_RUN',
                                  'remote-home': 'NOT_RUN'}}


def assess(document, root):
    """Fail closed on missing cases, forged/missing references and ambiguous claims."""
    root = Path(root).resolve(strict=True)
    if document.get('schema') != template()['schema']:
        raise ValueError('unsupported acceptance schema')
    if document.get('topology') != 'local/LAN':
        raise ValueError('this bounded harness qualifies only local/LAN')
    checks = document.get('checks', {})
    if set(checks) != set(REQUIRED):
        raise ValueError('mandatory case inventory differs')
    blocked = []
    for name in REQUIRED:
        check = checks[name]
        if check.get('status') not in STATES:
            raise ValueError('invalid status: ' + name)
        artifacts = check.get('artifacts', [])
        for artifact in artifacts:
            relative = Path(artifact['path'])
            target = (root / relative).resolve(strict=True)
            if relative.is_absolute() or not target.is_relative_to(root) or not target.is_file():
                raise ValueError('unsafe evidence reference: ' + name)
            if hashlib.sha256(target.read_bytes()).hexdigest() != artifact['sha256']:
                raise ValueError('evidence digest mismatch: ' + name)
        if check['status'] == 'PASS':
            if not artifacts or not check.get('observation', '').strip():
                raise ValueError('unattributed PASS: ' + name)
            if name == 'operator-listening' and not check.get('operator'):
                raise ValueError('operator identity and observation required')
        else:
            blocked.append({'case': name, 'status': check['status']})
    metadata = ('execution_id', 'git_head', 'host_identity', 'browser', 'codec')
    if not all(document.get(key) for key in metadata) or document.get('duration_seconds', 0) <= 0:
        blocked.append({'case': 'runtime-identity', 'status': 'NOT_RUN'})
    return {'schema_valid': True, 'mandatory_cases': len(REQUIRED),
            'ready_for_independent_qa': not blocked, 'outstanding': blocked,
            'note': 'Evidence indexing does not replace runtime observation or independent QA.'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['template', 'check'])
    parser.add_argument('--index', type=Path)
    parser.add_argument('--evidence', type=Path)
    args = parser.parse_args()
    if args.command == 'template':
        print(json.dumps(template(), indent=2))
        return 0
    if not args.index or not args.evidence:
        parser.error('check requires --index and --evidence')
    result = assess(json.loads(args.index.read_text()), args.evidence)
    print(json.dumps(result, indent=2))
    return 0 if result['ready_for_independent_qa'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
