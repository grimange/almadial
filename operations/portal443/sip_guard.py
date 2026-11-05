#!/usr/bin/env python3
"""Portal-owned early nftables gate; never flush firewalld or conntrack globally."""
import ipaddress
import json
import subprocess
import time

from common import HEARTBEAT, public_ip

TABLE = 'alma_portal_sip'
GRACE = 120
MARK = '0x40000000'


def nft(*args, script=None):
    return subprocess.run(['/usr/sbin/nft', *args], input=script, text=True,
                          capture_output=True, timeout=8, check=False)


def template():
    # priority -10 precedes firewalld's input/output priority +10, including
    # its unconditional established/related accept. Return still runs firewalld.
    return f'''table inet {TABLE} {{
      set agents4 {{ type ipv4_addr; flags timeout; }}
      set grace4 {{ type ipv4_addr; flags timeout; }}
      chain input {{
        type filter hook input priority -10; policy accept;
        iifname "lo" return
        udp dport 5061 ip saddr @agents4 return
        udp dport 5061 counter drop
        udp dport 10000-20000 ip saddr @agents4 ct mark set ct mark | {MARK} return
        udp dport 10000-20000 ip saddr @grace4 ct state established ct mark & {MARK} != 0 return
        udp dport 10000-20000 counter drop
      }}
      chain output {{
        type filter hook output priority -10; policy accept;
        oifname "lo" return
        udp sport 5061 ip daddr @agents4 return
        udp sport 5061 counter drop
        udp sport 10000-20000 ip daddr @agents4 ct mark set ct mark | {MARK} return
        udp sport 10000-20000 ip daddr @grace4 ct state established ct mark & {MARK} != 0 return
        udp sport 10000-20000 counter drop
      }}
    }}'''


def bootstrap(command=nft):
    result = command('list', 'table', 'inet', TABLE)
    if result.returncode == 0:
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
    bootstrap()
