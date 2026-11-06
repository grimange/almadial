#!/usr/bin/env python3
"""Root-only isolated UDP/conntrack integration test. No host routes are changed."""
import json
import os
import subprocess
import sys
import time

import sip_guard

SERVER = '198.18.0.1'
CLIENT = '158.62.74.253'
prefix = f'alma-guard-test-{os.getpid()}'
server_ns, client_ns = prefix + '-s', prefix + '-c'
created, children = [], []


def run(*args, data=None):
    return subprocess.run(args, input=data, text=True, capture_output=True, check=True, timeout=10)


def nft(*args, script=None):
    return subprocess.run(['ip', 'netns', 'exec', server_ns, 'nft', *args],
                          input=script, text=True, capture_output=True, timeout=10)


def child(ns, code):
    p = subprocess.Popen(['ip', 'netns', 'exec', ns, sys.executable, '-u', '-c', code],
                         stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    children.append(p)
    return p


try:
    for name in (server_ns, client_ns):
        run('ip', 'netns', 'add', name)
        created.append(name)
    run('ip', '-n', server_ns, 'link', 'add', 'veth-s', 'type', 'veth', 'peer', 'name', 'veth-c', 'netns', client_ns)
    for ns, interface, address, peer in ((server_ns, 'veth-s', SERVER, CLIENT), (client_ns, 'veth-c', CLIENT, SERVER)):
        run('ip', '-n', ns, 'addr', 'add', address + '/32', 'dev', interface)
        run('ip', '-n', ns, 'link', 'set', interface, 'up')
        run('ip', '-n', ns, 'link', 'set', 'lo', 'up')
        run('ip', '-n', ns, 'route', 'add', peer + '/32', 'dev', interface)
    sip_guard.bootstrap(nft)
    # Reproduce the problematic downstream established accept. Only these
    # isolated namespaces receive fixture accepts; the host is untouched.
    fixture = '''table inet fixture_firewalld {
      chain input { type filter hook input priority 10; policy drop;
        ct state established,related accept
        udp dport { 5060, 5061, 10000-20000 } accept
      }
      chain output { type filter hook output priority 10; policy accept;
        ct state established,related accept
      }
    }'''
    assert nft('-f', '-', script=fixture).returncode == 0
    echo = child(server_ns, '''import socket,select
sockets=[]
for port in (5060,5061,10000):
 s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('198.18.0.1',port));sockets.append(s)
print('READY',flush=True)
while True:
 for s in select.select(sockets,[],[])[0]:
  data,peer=s.recvfrom(1024);s.sendto(data,peer)
''')
    assert echo.stdout.readline().strip() == 'READY'
    client = child(client_ns, '''import socket,json,sys
sockets={}
print('READY',flush=True)
for line in sys.stdin:
 q=json.loads(line);key=q['key'];port=q['port']
 if key not in sockets:
  s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.settimeout(.35);sockets[key]=s
 s=sockets[key];payload=key.encode();s.sendto(payload,('198.18.0.1',port))
 try: ok=s.recvfrom(1024)[0]==payload
 except socket.timeout: ok=False
 print(json.dumps(ok),flush=True)
''')
    assert client.stdout.readline().strip() == 'READY'

    def check(label, key, port, expected):
        client.stdin.write(json.dumps(dict(key=key, port=port)) + '\n')
        client.stdin.flush()
        actual = json.loads(client.stdout.readline())
        assert actual == expected, f'{label}: expected {expected}, got {actual}'
        print('PASS ' + label, flush=True)

    now = int(time.time())
    sip_guard.refresh({}, {}, now, nft)
    for port in (5060, 5061):
        check(f'no grant fresh SIP{port} blocked', f'sip-none-{port}', port, False)
    sip_guard.refresh({CLIENT: now + 300}, {}, now, nft)
    for port in (5060, 5061):
        check(f'valid agent fresh SIP{port} passes', f'sip-old-{port}', port, True)
    check('valid agent RTP passes and is marked', 'rtp-old', 10000, True)
    before = nft('-j', 'list', 'set', 'inet', sip_guard.TABLE, 'agents4').stdout
    # Empty fixture chains to exercise an existing-table upgrade (isolated only).
    assert nft('-f', '-', script='\n'.join(
        f'flush chain inet {sip_guard.TABLE} {chain}' for chain in ('input', 'output'))).returncode == 0
    sip_guard.bootstrap(nft)
    after = nft('-j', 'list', 'set', 'inet', sip_guard.TABLE, 'agents4').stdout
    assert json.loads(before)['nftables'][-1]['set']['elem'][0]['elem']['val'] == json.loads(after)['nftables'][-1]['set']['elem'][0]['elem']['val']
    for port in (5060, 5061):
        check(f'policy upgrade preserves established SIP{port} grant', f'sip-old-{port}', port, True)
    now = int(time.time())
    sip_guard.refresh({}, {CLIENT: now + 3}, now, nft)
    for port in (5060, 5061):
        check(f'revoked established SIP{port} blocked before downstream ct accept', f'sip-old-{port}', port, False)
        check(f'revoked fresh SIP{port} blocked', f'sip-new-{port}', port, False)
    check('established authorized RTP survives within grace', 'rtp-old', 10000, True)
    check('new RTP flow cannot use grace', 'rtp-new', 10000, False)
    time.sleep(3.1)
    check('established RTP blocked after native grace timeout', 'rtp-old', 10000, False)
    now = int(time.time())
    sip_guard.refresh({CLIENT: now + 2}, {}, now, nft)
    for port in (5060, 5061):
        check(f'short valid grant fresh SIP{port} passes', f'sip-expiry-{port}', port, True)
    time.sleep(2.1)
    for port in (5060, 5061):
        check(f'native grant expiry blocks established SIP{port} without worker', f'sip-expiry-{port}', port, False)
    sip_guard.refresh({}, {}, int(time.time()), nft)
    for port in (5060, 5061):
        check(f'web-only supervisor has no SIP{port} grant', f'sip-supervisor-{port}', port, False)
    print('ISOLATED_NETWORK_PROBE_PASS', flush=True)
finally:
    for p in children:
        p.terminate()
        try:
            p.wait(timeout=3)
        except subprocess.TimeoutExpired:
            p.kill()
            p.wait()
    for ns in reversed(created):
        run('ip', 'netns', 'del', ns)
