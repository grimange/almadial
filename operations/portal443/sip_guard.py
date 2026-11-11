#!/usr/bin/env python3
"""Portal-owned early nftables gate; never flush firewalld or conntrack globally."""
import ipaddress
import json
import subprocess
from pathlib import Path

from common import HEARTBEAT, SIP_PORTS, public_ip

TABLE = 'alma_portal_sip'
GRACE = 120
MARK = '0x40000000'
POLICY = 'alma-phone-sip-' + '-'.join(SIP_PORTS) + '-v2-carriers'
CARRIER_CONFIG = Path(__file__).with_name('carriers.json')


def validate_carriers(values):
    if not isinstance(values, list) or any(not isinstance(ip, str) for ip in values):
        raise ValueError('Carrier configuration must be a list of public IPv4 addresses')
    for value in values:
        ip = ipaddress.ip_address(value)
        if ip.version != 4 or not ip.is_global or ip.is_multicast or ip.is_reserved:
            raise ValueError('Carrier addresses must be unicast public IPv4 addresses')
    if len(set(values)) != len(values):
        raise ValueError('Duplicate carrier address')
    return list(values)


def carriers():
    # Static operator configuration, independent of lease/grace state.
    return sorted(validate_carriers(json.loads(CARRIER_CONFIG.read_text())))


def nft(*args, script=None):
    return subprocess.run(['/usr/sbin/nft', *args], input=script, text=True,
                          capture_output=True, timeout=8, check=False)


def chain_rules(chain):
    direction, address, interface = ('dport', 'saddr', 'iifname') if chain == 'input' else ('sport', 'daddr', 'oifname')
    rules = [f'{interface} "lo" return']
    rules.append('ip saddr @carriers4 udp dport 5060 counter return' if chain == 'input' else
                 'ip daddr @carriers4 udp sport 5060 udp dport 5060 counter return')
    for port in SIP_PORTS:
        rules.extend((f'udp {direction} {port} ip {address} @agents4 return',
                      f'udp {direction} {port} counter drop'))
    rules.extend((
        f'udp {direction} 10000-20000 ip {address} @agents4 ct mark set ct mark | {MARK} return',
        f'udp {direction} 10000-20000 ip {address} @grace4 ct state established ct mark & {MARK} != 0 return',
        f'udp {direction} 10000-20000 counter drop comment "{POLICY}"'))
    return rules


def template():
    # priority -10 precedes firewalld's input/output priority +10, including
    # its unconditional established/related accept. Return still runs firewalld.
    lines = [f'table inet {TABLE} {{',
             'set agents4 { type ipv4_addr; flags timeout; }',
             'set grace4 { type ipv4_addr; flags timeout; }',
             'set carriers4 { type ipv4_addr; ' +
             ('elements = { ' + ', '.join(carriers()) + ' }; ' if carriers() else '') + '}']
    for chain in ('input', 'output'):
        lines.extend((f'chain {chain} {{', f'type filter hook {chain} priority -10; policy accept;'))
        lines.extend(chain_rules(chain))
        lines.append('}')
    return '\n'.join(lines + ['}'])


def bootstrap(command=nft):
    trusted = carriers()  # Validate configuration before any runtime mutation.
    result = command('-j', 'list', 'table', 'inet', TABLE)
    if result.returncode == 0:
        objects = json.loads(result.stdout)['nftables']
        sets = {item['set']['name']: item['set'] for item in objects if 'set' in item}
        installed = {item['rule']['chain'] for item in objects
                     if item.get('rule', {}).get('comment') == POLICY}
        current = sets.get('carriers4')
        if installed == {'input', 'output'} and current is not None:
            if sorted(current.get('elem', [])) == trusted:
                return
            lines = [f'flush set inet {TABLE} carriers4']
            if trusted:
                lines.append(f'add element inet {TABLE} carriers4 {{ ' + ', '.join(trusted) + ' }')
            if command('-f', '-', script='\n'.join(lines)).returncode:
                raise RuntimeError('Carrier membership update failed')
            return
        # Upgrade only this table's chains in one transaction. Preserve live
        # allowlist/grace members and their original native timeout deadlines.
        lines = ([f'add set inet {TABLE} carriers4 {{ type ipv4_addr; }}']
                 if current is None else [f'flush set inet {TABLE} carriers4'])
        if trusted:
            lines.append(f'add element inet {TABLE} carriers4 {{ ' + ', '.join(trusted) + ' }')
        for chain in ('input', 'output'):
            lines.append(f'flush chain inet {TABLE} {chain}')
            lines.extend(f'add rule inet {TABLE} {chain} {rule}' for rule in chain_rules(chain))
        if command('-f', '-', script='\n'.join(lines)).returncode:
            raise RuntimeError('SIP guard policy update failed')
        return
    # A syntax/check error must not be mistaken for a successful guard.
    result = command('-f', '-', script=template())
    if result.returncode:
        raise RuntimeError('SIP guard initialization failed')


def grace_state(sip, previous, grace, now):
    """Absolute, persisted grace deadlines; retries cannot extend revocation."""
    result = {public_ip(ip): end for ip, end in grace.items() if end > now and ip not in sip}
    for ip, end in previous.items():
        if ip not in sip and ip not in result and min(end, now) + GRACE > now:
            result[public_ip(ip)] = min(end, now) + GRACE
    return result


def refresh(sip, grace, now, command=nft):
    bootstrap(command)
    # Absolute lease deadline caps freshness. A dead worker cannot renew these.
    agents = {public_ip(ip): min(end, now + HEARTBEAT) for ip, end in sip.items() if end > now}
    media = {public_ip(ip): end for ip, end in grace.items() if end > now}
    for ip, end in agents.items():
        media[ip] = end + GRACE
    lines = [f'flush set inet {TABLE} agents4', f'flush set inet {TABLE} grace4']
    for name, values in (('agents4', agents), ('grace4', media)):
        if values:
            members = ', '.join(f'{ip} timeout {int(end-now)}s' for ip, end in sorted(values.items()))
            lines.append(f'add element inet {TABLE} {name} {{ {members} }}')
    # One atomic netlink transaction: never a partially refreshed allowlist.
    if command('-f', '-', script='\n'.join(lines) + '\n').returncode:
        raise RuntimeError('SIP guard refresh failed')
    for name, expected in (('agents4', agents), ('grace4', media)):
        result = command('-j', 'list', 'set', 'inet', TABLE, name)
        if result.returncode:
            raise RuntimeError('SIP guard confirmation failed')
        payload = json.loads(result.stdout)
        found = set()
        for item in payload['nftables']:
            if 'set' not in item:
                continue
            for element in item['set'].get('elem', []):
                if isinstance(element, dict):
                    element = element['elem']['val']
                found.add(str(ipaddress.ip_address(element)))
        if found != set(expected):
            raise RuntimeError('SIP guard membership mismatch')
    return set(agents)


if __name__ == '__main__':
    # The imported worker already holds this lock; standalone bootstrap must
    # coordinate with CLI address changes as well. No nested lock in refresh().
    import fcntl
    with open('/var/lib/alma-portal-443/worker.lock', 'r') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        bootstrap()
