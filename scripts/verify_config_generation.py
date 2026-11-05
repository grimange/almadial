#!/usr/bin/env python3
"""Replay the redacted M0-05 configuration contract; never contacts a host.

This validates configuration and recorded observations, not live compatibility.
Private, unsanitized evidence remains necessary for independent runtime review.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

SOURCE_SHA256 = '4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67'
DERIVED_SHA256 = '8eac12c46d7e20892bac03747c5f51b08d1e6982177767551d46ae4de21bdcda'
OWNERS = {
    **dict.fromkeys(('asterisk.conf', 'modules.conf', 'http.conf', 'rtp.conf',
                     'pjsip.conf', 'pjsip_wizard.conf', 'extensions.conf',
                     'confbridge.conf', 'manager.conf'), 'AlmaDial'),
    **dict.fromkeys(('pjsip-vicidial.conf', 'pjsip_wizard-vicidial.conf',
                     'extensions-vicidial.conf', 'confbridge-vicidial.conf',
                     'voicemail.conf', 'iax-vicidial.conf', 'sip-vicidial.conf',
                     'musiconhold-vicidial.conf', 'meetme-vicidial.conf'), 'VICIdial'),
}
INCLUDES = {
    'pjsip.conf': 'pjsip-vicidial.conf',
    'pjsip_wizard.conf': 'pjsip_wizard-vicidial.conf',
    'extensions.conf': 'extensions-vicidial.conf',
    'confbridge.conf': 'confbridge-vicidial.conf',
}
UNCHANGED = {'asterisk.conf', 'http.conf', 'rtp.conf', 'logger.conf', 'ari.conf',
             'keys/server.key', 'keys/server.crt'}
BASE_CHANGES = {'confbridge.conf', 'extensions.conf', 'manager.conf',
                'modules.conf', 'pjsip.conf'}


def sections(text):
    """Keep duplicate sections (legal when PJSIP object types differ)."""
    result = []
    current = None
    for raw in text.splitlines():
        line = raw.split(';', 1)[0].strip()
        match = re.fullmatch(r'\[([^]]+)\](?:\([^)]*\))?', line)
        if match:
            current = (match[1], [])
            result.append(current)
        elif current and line:
            current[1].append(line)
    return result


def values(lines):
    return dict(line.split('=', 1) for line in lines if '=' in line and '=>' not in line)


def validate(record):
    """Return failures and explicit limitations for the isolated lab contract."""
    errors = []
    files = record.get('files', {})
    if not isinstance(files, dict) or any(not isinstance(v, str) for v in files.values()):
        return {'state': 'FAIL', 'errors': ['invalid files mapping']}
    def check(condition, message):
        if not condition:
            errors.append(message)
    check(record.get('source_sha256') == SOURCE_SHA256, 'source digest mismatch')
    check(record.get('derivation_sha256') == DERIVED_SHA256, 'derivation digest mismatch')
    for base, generated in INCLUDES.items():
        check(base in files and generated in files, f'missing file: {base} or {generated}')
        names = re.findall(r'^\s*#include\s+["<]([^">]+)[">]\s*$', files.get(base, ''), re.M)
        check(names.count(generated) == 1, f'include relationship: {base}')
    for name, text in files.items():
        for included in re.findall(r'^\s*#include\s+["<]([^">]+)[">]', text, re.M):
            check(included in files, f'missing required include: {included}')
        check(not re.search(r'YOUR[_ -]?(?:SERVER|EXTERNAL|PUBLIC)?[_ -]?IP|EXTERNAL_IP|SERVER_IP_HERE', text, re.I),
              f'unresolved IP placeholder: {name}')
    objects = set()
    endpoints = {}
    for name in ('pjsip.conf', 'pjsip-vicidial.conf', 'pjsip_wizard-vicidial.conf'):
        for section, lines in sections(files.get(name, '')):
            obj = values(lines)
            kind = obj.get('type')
            key = (section, kind)
            check(key not in objects, f'duplicate PJSIP definition: {key}')
            objects.add(key)
            if kind in ('wizard', 'endpoint'):
                check(section not in endpoints, f'endpoint/wizard collision: {section}')
                endpoints[section] = obj
    check('9001' in endpoints and endpoints['9001'].get('context') == 'alm3-echo', 'native endpoint changed')
    phone = endpoints.get('9051', {})
    for key, expected in {'type': 'wizard', 'transport': 'transport-wss',
                          'endpoint/context': 'alm5-local', 'endpoint/webrtc': 'yes',
                          'endpoint/direct_media': 'no', 'accepts_auth': 'yes',
                          'accepts_registrations': 'yes'}.items():
        check(phone.get(key) == expected, f'generated phone setting: {key}')
    for section, lines in sections(files.get('confbridge-vicidial.conf', '')):
        if section == 'vici_agent_bridge':
            check(values(lines).get('type') == 'bridge', 'invalid agent bridge')
    conf = {name: values(lines) for name, lines in sections(files.get('confbridge-vicidial.conf', ''))}
    check(conf.get('vici_agent_bridge', {}).get('type') == 'bridge', 'missing agent bridge')
    check(conf.get('vici_agent_user', {}).get('type') == 'user', 'missing agent user')
    check(conf.get('vici_monitor_menu', {}).get('type') == 'menu', 'missing monitor menu')
    check(record.get('allocation') == {'count': 300, 'min': 9600000, 'max': 9600299}, 'conference allocation mismatch')
    contexts = dict(sections(files.get('extensions-vicidial.conf', '') + '\n' + files.get('extensions.conf', '')))
    for context, lines in contexts.items():
        priorities = set()
        for line in lines:
            match = re.match(r'exten\s*=>\s*([^,]+),(\d+),', line)
            if match:
                key = match.groups()
                check(key not in priorities, f'duplicate dialplan priority: {context}/{key}')
                priorities.add(key)
    reachable, pending = set(), ['alm3-echo', 'alm5-local']
    while pending:
        context = pending.pop()
        if context in reachable:
            continue
        reachable.add(context)
        check(context in contexts, f'missing reachable context: {context}')
        for line in contexts.get(context, []):
            match = re.match(r'include\s*=>\s*([^,;\s]+)', line)
            if match:
                pending.append(match[1])
            check(not re.search(r'\b(?:MeetMe|Macro|Monitor)\s*\(|\bSIP/', line, re.I),
                  f'unsupported reachable application/channel: {context}')
    check('vicidial-auto-external' not in reachable, 'legacy external context exposed')
    check('PJSIP_DIAL_CONTACTS(9051)' in '\n'.join(contexts.get('vicidial-auto-phones', [])), 'generated phone route missing')
    echo = '\n'.join(contexts.get('alm3-echo', []))
    check('9001,1,Answer()' in echo and 'Echo()' in echo, 'Echo dialplan changed')
    manager = {name: values(lines) for name, lines in sections(files.get('manager.conf', ''))}
    general, user = manager.get('general', {}), manager.get('alm5-local', {})
    check(general.get('bindaddr') == '127.0.0.1' and general.get('webenabled') == 'no', 'AMI listener must be loopback only')
    check(user.get('write') == 'system' and user.get('permit') == '127.0.0.1/255.255.255.255', 'AMI permission boundary changed')
    check(all(record.get('ami', {}).get(k) is True for k in ('valid_auth', 'invalid_rejected', 'command_denied')), 'AMI observation missing')
    check(record.get('repeat_equal') is True, 'regeneration drift')
    check(set(record.get('base_changes', [])) == BASE_CHANGES, 'unapproved base change')
    check(set(record.get('unchanged_native_files', [])) == UNCHANGED, 'native preservation evidence missing')
    p = record.get('permissions', {})
    check(p.get('mariadb_mode') == '0640' and p.get('mariadb_group') == 'mysql', 'MariaDB permissions regression')
    check(p.get('skip_networking') == 'ON' and p.get('tcp_3306') is False, 'MariaDB effective networking regression')
    check(p.get('perl_directories') == ['0755'] * 4 and p.get('nonroot_modules') is True
          and p.get('nonroot_database') is True, 'Perl traversal/database regression')
    return {'state': 'FAIL' if errors else 'PASS', 'errors': errors,
            'evidence_class': 'REDACTED_FIXTURE_REPLAY', 'runtime': 'NOT_RUN',
            'reachable_contexts': sorted(reachable),
            'limitations': ['Legacy external MeetMe route quarantined; not runtime qualified.',
                            'Full keepalive daemon, campaigns and agent conferencing NOT_RUN.']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--fixture', type=Path, default=Path(__file__).parent / 'tests/fixtures/config_generation/r4037.json')
    args = parser.parse_args()
    try:
        data = args.fixture.read_bytes()
        result = validate(json.loads(data))
        result['fixture_sha256'] = hashlib.sha256(data).hexdigest()
    except (OSError, ValueError, TypeError, AttributeError) as exc:
        result = {'state': 'FAIL', 'errors': [str(exc)], 'runtime': 'NOT_RUN'}
    print(json.dumps(result, indent=2))
    return 0 if result['state'] == 'PASS' else 1


if __name__ == '__main__':
    raise SystemExit(main())
