#!/usr/bin/python3
"""Root-only scoped carrier administration; carriers.json is the sole address authority."""
import argparse
from contextlib import contextmanager
import fcntl
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile
import time

import sip_guard as guard

LOCK = Path('/var/lib/alma-portal-443/worker.lock')
# Reserved ownership namespace. Only this exact rule shape is ours; agent rules
# and historical unmarked rules are never inferred to be owned from their IP.
CONFIG_OWNER = 0
PRIORITY = -73
OWNED = re.compile(r'^rule priority="-73" family="ipv4" source address="([0-9.]+)/32" port port="5060" protocol="udp" accept$')
MEDIA_OWNED = re.compile(r'^rule priority="-74" family="ipv4" source address="([0-9.]+)/32" port port="10000-20001" protocol="udp" accept$')


def rule(ip, media=False):
    return f'rule priority="{-74 if media else PRIORITY}" family="ipv4" source address="{ip}/32" port port="{guard.MEDIA_PORTS if media else "5060"}" protocol="udp" accept'


@contextmanager
def locked(path=LOCK, readonly=False, timeout=10):
    # Existing worker lock: do not replace the inode or create a separate lock.
    fd = os.open(path, (os.O_RDONLY if readonly else os.O_WRONLY) | os.O_NOFOLLOW)
    try:
        end = time.monotonic() + timeout
        while True:
            try:
                fcntl.flock(fd, (fcntl.LOCK_SH if readonly else fcntl.LOCK_EX) | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() >= end:
                    raise RuntimeError('Portal worker lock busy; retry when the current operation completes')
                time.sleep(0.05)
        yield
    finally:
        os.close(fd)


def load(path, media=False):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, 'rb') as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != CONFIG_OWNER or info.st_mode & 0o022:
            raise ValueError('Configuration must be a regular root-owned file without group/other write permission')
        raw = stream.read()
    return guard.validate_config(json.loads(raw))['media' if media else 'sip'], raw, stat.S_IMODE(info.st_mode)


def updated_config(raw, values, media=False):
    value = json.loads(raw)
    if isinstance(value, list) and not media:
        return (json.dumps(values) + '\n').encode()
    if isinstance(value, list):
        value = {'sip': value, 'media': []}
    value['media' if media else 'sip'] = values
    return (json.dumps(value, indent=2) + '\n').encode()


def atomic_write(path, raw, mode):
    fd, tmp = tempfile.mkstemp(prefix='.carriers-', dir=path.parent)
    try:
        with os.fdopen(fd, 'wb') as stream:
            os.fchmod(stream.fileno(), mode)
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(tmp, path)
        directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


class Firewall:
    def __init__(self, media=False):
        self.media = media
        self.set_name = 'carrier_media4' if media else 'carriers4'

    def command(self, *args):
        result = subprocess.run(args, capture_output=True, text=True, timeout=8)
        if result.returncode:
            raise RuntimeError(f'{args[0]} failed ({result.returncode}): {result.stderr.strip() or result.stdout.strip()}')
        return result.stdout.strip()

    def rich(self, permanent=False):
        mode = ('--permanent',) if permanent else ()
        return set(self.command('/usr/bin/firewall-cmd', *mode, '--zone=public', '--list-rich-rules').splitlines())

    def managed(self, permanent=False):
        return {r for r in self.rich(permanent) if (MEDIA_OWNED if self.media else OWNED).fullmatch(r)}

    def set_rich(self, desired, permanent=False):
        current = self.managed(permanent)
        mode = ('--permanent',) if permanent else ()
        for operation, values in (('add', desired - current), ('remove', current - desired)):
            for value in sorted(values):
                self.command('/usr/bin/firewall-cmd', *mode, '--zone=public', f'--{operation}-rich-rule={value}')

    def nft(self):
        data = json.loads(self.command('/usr/sbin/nft', '-j', 'list', 'table', 'inet', guard.TABLE))['nftables']
        carriers = next(x['set'] for x in data if x.get('set', {}).get('name') == self.set_name)
        if 'timeout' in carriers.get('flags', []):
            raise RuntimeError(f'{self.set_name} unexpectedly has timeout semantics')
        members = set(carriers.get('elem', []))
        rules = [x['rule'] for x in data if 'rule' in x]
        markers = {guard.POLICY} if self.media else {guard.POLICY, 'alma-phone-sip-5060-5061-v2-carriers'}
        policy = {r['chain'] for r in rules if r.get('comment') in markers}
        refs = [r['chain'] for r in rules if '@' + self.set_name in json.dumps(r)]
        # nft JSON serializes named-set references as "@carriers4".
        return members, policy == {'input', 'output'} and sorted(refs) == ['input', 'output']

    def restore_nft(self, members):
        guard.validate_carriers(sorted(members))
        script = f'flush set inet {guard.TABLE} {self.set_name}\n'
        if members:
            script += f'add element inet {guard.TABLE} {self.set_name} {{ ' + ', '.join(sorted(members)) + ' }\n'
        result = guard.nft('-f', '-', script=script)
        if result.returncode:
            raise RuntimeError('Could not restore nftables carrier membership')

    def sync_nft(self):
        guard.bootstrap(membership=self.set_name)

    def policy_snapshot(self):
        objects = json.loads(self.command('/usr/sbin/nft', '-j', 'list', 'table', 'inet', guard.TABLE))['nftables']
        return {'rules': [x['rule'] for x in objects if 'rule' in x],
                'media_set': next((x['set'] for x in objects if x.get('set', {}).get('name') == 'carrier_media4'), None)}

    def restore_policy(self, saved):
        # Restore only our two chains and static media set. Dynamic sets and
        # their kernel expiration deadlines remain untouched.
        commands = [{'flush': {'chain': {'family': 'inet', 'table': guard.TABLE, 'name': chain}}}
                    for chain in ('input', 'output')]
        for original in saved['rules']:
            value = {k: v for k, v in original.items() if k not in ('handle', 'index')}
            commands.append({'add': {'rule': value}})
        current = self.policy_snapshot()['media_set']
        old = saved['media_set']
        if current is not None:
            commands.append({'flush': {'set': {'family': 'inet', 'table': guard.TABLE, 'name': 'carrier_media4'}}})
            if old is None:
                commands.append({'delete': {'set': {'family': 'inet', 'table': guard.TABLE, 'name': 'carrier_media4'}}})
        elif old is not None:
            commands.append({'add': {'set': {k: v for k, v in old.items() if k not in ('handle', 'elem')}}})
        if old and old.get('elem'):
            commands.append({'add': {'element': {'family': 'inet', 'table': guard.TABLE,
                                                  'name': 'carrier_media4', 'elem': old['elem']}}})
        script = json.dumps({'nftables': commands})
        if guard.nft('-c', '-j', '-f', '-', script=script).returncode:
            raise RuntimeError('Rollback policy syntax validation failed')
        if guard.nft('-j', '-f', '-', script=script).returncode:
            raise RuntimeError('Rollback policy restoration failed')


def snapshot(firewall):
    members, policy = firewall.nft()
    return members, policy, firewall.managed(), firewall.managed(True)


def diagnostics(configured, state, media=False):
    members, policy, runtime, permanent = state
    desired = {rule(ip, media) for ip in configured}
    errors = []
    if set(configured) != members:
        errors.append('nftables carrier membership differs from configuration')
    if not policy:
        errors.append('nftables carrier policy missing or inconsistent; inspect guard policy')
    for name, actual in (('runtime', runtime), ('permanent', permanent)):
        if desired != actual:
            errors.append(f'firewalld {name}: missing {len(desired - actual)}, obsolete managed {len(actual - desired)}')
    return errors


def execute(operation, ip=None, path=guard.CARRIER_CONFIG, firewall=None, media=False):
    firewall = firewall or Firewall(media)
    configured, raw, mode = load(path, media)
    if ip is not None:
        guard.validate_carriers([ip])
    before = snapshot(firewall)
    if operation in ('list', 'validate'):
        errors = diagnostics(configured, before, media)
        if operation == 'list':
            print('Trusted Carrier Media (UDP/' + guard.MEDIA_PORTS + ')' if media else 'Trusted SIP Carriers (UDP/5060 only)')
            if not configured:
                print('(none configured)')
            for address in configured:
                print(f'{address}  Configured  nftables={"OK" if address in before[0] else "MISSING"}  '
                      f'runtime={"OK" if rule(address, media) in before[2] else "MISSING"}  '
                      f'permanent={"OK" if rule(address, media) in before[3] else "MISSING"}')
        for error in errors:
            print('DRIFT: ' + error)
        if not errors:
            print('Validation: OK')
        return 1 if errors else 0
    desired = list(configured)
    if operation == 'add' and ip not in desired:
        desired.append(ip)
    if operation == 'remove':
        if ip not in desired:
            print(f'Carrier {ip} not configured; no changes made.')
            return 0
        print(f'Warning: removing carrier {ip} may disconnect active carrier traffic.', file=sys.stderr)
        desired.remove(ip)
    guard.validate_carriers(sorted(before[0]))  # Ensure the snapshot can be restored before writing.
    changed = desired != configured
    try:
        if changed:
            atomic_write(path, updated_config(raw, desired, media), mode)
        target = {rule(address, media) for address in desired}
        firewall.set_rich(target, True)
        firewall.set_rich(target)
        firewall.sync_nft()
        errors = diagnostics(desired, snapshot(firewall), media)
        if errors:
            raise RuntimeError('; '.join(errors))
    except Exception as failure:
        rollback_errors = []
        actions = [('configuration', lambda: atomic_write(path, raw, mode))] if changed else []
        actions += [('firewalld permanent', lambda: firewall.set_rich(before[3], True)),
                    ('firewalld runtime', lambda: firewall.set_rich(before[2])),
                    ('nftables', lambda: firewall.restore_nft(before[0]))]
        for name, action in actions:
            try:
                action()
            except Exception as rollback_failure:
                rollback_errors.append(f'{name}: {rollback_failure}')
        try:
            remaining = snapshot(firewall)
            if remaining != before or path.read_bytes() != raw:
                rollback_errors.append('read-back differs from pre-change configuration/firewall snapshot')
        except Exception as check_failure:
            rollback_errors.append(f'read-back: {check_failure}')
        raise RuntimeError(f'Update failed: {failure}. ' +
                           ('ROLLBACK INCOMPLETE: ' + '; '.join(rollback_errors) if rollback_errors else 'Previous state restored.')) from failure
    action = 'added' if operation == 'add' and changed else 'removed' if changed else 'already configured; synchronized (no address change)' if operation == 'add' else 'synchronized'
    print(f'Carrier {ip or "configuration"}: {action}.\nConfiguration: OK\nnftables: OK\nfirewalld runtime: OK\nfirewalld permanent: OK')
    return 0


def main(argv=None):
    parser = argparse.ArgumentParser(description='Manage separate permanent IPv4 SIP and RTP/RTCP carrier permissions; no Asterisk changes.')
    commands = parser.add_subparsers(dest='operation', required=True)
    def operations(parent):
        for operation in ('list', 'validate', 'sync', 'add', 'remove'):
            command = parent.add_parser(operation)
            if operation in ('add', 'remove'):
                command.add_argument('ipv4')
    operations(commands)
    media_parser = commands.add_parser('media', help='Manage verified carrier RTP/RTCP addresses separately from SIP')
    operations(media_parser.add_subparsers(dest='media_operation', required=True))
    args = parser.parse_args(argv)
    media = args.operation == 'media'
    operation = args.media_operation if media else args.operation
    readonly = operation in ('list', 'validate')
    if not readonly and os.geteuid() != 0:
        parser.exit(1, 'Error: root privileges required for changes.\n')
    try:
        with locked(readonly=readonly):
            return execute(operation, getattr(args, 'ipv4', None), media=media)
    except (OSError, ValueError, RuntimeError, StopIteration, TypeError, subprocess.SubprocessError) as error:
        print(f'Error: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
