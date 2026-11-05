#!/usr/bin/env python3
"""Bounded live test observer: aggregate REGISTER counts/status codes only.

Never writes packets, headers, credentials, challenge values, or identifiers.
Packet buffers are inspected in memory and discarded. Scoped to the test IP
and UDP5061. Run through SSH, as root, during the operator's private tests.
"""
from collections import Counter
import ipaddress
import json
import os
from pathlib import Path
import re
import socket
import struct
import time

CLIENT = ipaddress.ip_address('158.62.74.253').packed
SERVER = ipaddress.ip_address('2.29.153.161').packed
OUTPUT = Path('/root/alma-portal-revoke-backup/register-counts.json')
STATE = Path('/var/lib/alma-portal-443/state.json')
os.umask(0o077)
started = int(time.time())
counts, buckets = Counter(), {}
# ETH_P_ALL is necessary to receive outbound packets as well as inbound IP.
# Filter immediately below before examining any application payload.
sock = socket.socket(socket.AF_PACKET, socket.SOCK_RAW, socket.htons(0x0003))
sock.bind(('eth0', 0))
sock.settimeout(1)
last_save = 0
while time.time() - started < 900:
    now = int(time.time())
    if now != last_save:
        tmp = OUTPUT.with_suffix('.tmp')
        tmp.write_text(json.dumps({'started_at': started, 'updated_at': now,
                                  'counts': dict(counts), 'ten_second_buckets': buckets}, indent=2))
        os.replace(tmp, OUTPUT)
        last_save = now
    try:
        packet = sock.recv(8192)
    except socket.timeout:
        continue
    if len(packet) < 42 or packet[12:14] != b'\x08\x00' or packet[23] != 17:
        continue
    src, dst = packet[26:30], packet[30:34]
    incoming = src == CLIENT and dst == SERVER
    outgoing = src == SERVER and dst == CLIENT
    if not (incoming or outgoing):
        continue
    offset = 14 + (packet[14] & 15) * 4
    sport, dport = struct.unpack('!HH', packet[offset:offset+4])
    if (incoming and dport != 5061) or (outgoing and sport != 5061):
        continue
    payload = packet[offset+8:]
    if not re.search(br'\r\nCSeq:\s*\d+\s+REGISTER\r\n', payload, re.I):
        continue
    if incoming and payload.startswith(b'REGISTER '):
        event = 'incoming_REGISTER'
    elif outgoing and payload.startswith(b'SIP/2.0 '):
        code = payload[8:11]
        if not code.isdigit():
            continue
        event = 'outgoing_' + code.decode('ascii')
        if code == b'200':
            # Distinguish accepted registrations from deregistration replies.
            expires = re.search(br'(?:;expires=|\r\nExpires:\s*)(\d+)', payload, re.I)
            event += '_binding' if expires and int(expires[1]) > 0 else '_unregister_or_unspecified'
    else:
        continue
    try:
        leases = json.loads(STATE.read_text()).get('leases', {}).values()
        active = any(l.get('eligible') and (l['role'] == 'agent' or l.get('sip_enabled', False)) and
                     l['ip'] == '158.62.74.253' and l['expires_at'] > now for l in leases)
    except Exception:
        active = False
    label = ('phone_granted:' if active else 'no_phone:') + event
    counts[label] += 1
    bucket = str(now // 10 * 10)
    buckets.setdefault(bucket, {})[label] = buckets.get(bucket, {}).get(label, 0) + 1
print('REGISTER_METADATA_OBSERVER_FINISHED', flush=True)
