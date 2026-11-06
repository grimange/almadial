#!/usr/bin/env python3
"""Bounded ALM-6 guest-only lifecycle observer. Explicit invocation mutates test state.

Requires the authorized almadial-alma10 VM and pre-alm6-telephony-runtime snapshot.
Credentials stay in /root/almadial-m006. No carrier, daemon, schema creation or drop.
The lab controller owns allocation/recording metadata; it is not VICIdial's agent UI.
"""
import argparse
import datetime
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import time
import wave

ROOT = Path('/root/almadial-m006')


def command(argv, input=None):
    return subprocess.check_output(argv, input=input, text=True, timeout=15)


def sql(query):
    return command(['mariadb', '-N', 'almadial_m005'], query).strip()


def cli(query):
    return command(['asterisk', '-rx', query])


class AMI:
    def __init__(self, password):
        self.socket = socket.create_connection(('127.0.0.1', 5038), 3)
        self.socket.settimeout(.2)
        self.buffer = b''
        self.rows = []
        self.send('Login', Username='alm6-runtime', Secret=password, Events='on')

    def send(self, action, **fields):
        fields = {'Action': action, 'ActionID': action + '-' + str(time.monotonic_ns()), **fields}
        if any('\r' in v or '\n' in v for v in fields.values()):
            raise ValueError('AMI field injection refused')
        self.socket.sendall(('\r\n'.join(k + ': ' + v for k, v in fields.items()) + '\r\n\r\n').encode())
        return fields['ActionID']

    def require(self, action, **fields):
        identity = self.send(action, **fields)
        until = time.monotonic() + 5
        while time.monotonic() < until:
            self.pump(.2)
            matches = [r for r in self.rows if r.get('ActionID') == identity and 'Response' in r]
            if matches:
                if matches[-1]['Response'] != 'Success':
                    raise RuntimeError(action + ' rejected: ' + matches[-1].get('Message', ''))
                return matches[-1]
        raise TimeoutError(action + ' response absent')

    def pump(self, seconds=.3):
        until = time.monotonic() + seconds
        while time.monotonic() < until:
            try:
                data = self.socket.recv(65536)
            except socket.timeout:
                continue
            if not data:
                break
            self.buffer += data
            if len(self.buffer) > 1024 * 1024:
                raise ValueError('AMI message too large')
            while b'\r\n\r\n' in self.buffer:
                raw, self.buffer = self.buffer.split(b'\r\n\r\n', 1)
                row = dict(line.split(': ', 1) for line in raw.decode(errors='replace').split('\r\n') if ': ' in line)
                if row:
                    self.rows.append({'utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), **row})
            if len(self.rows) > 20000:
                raise ValueError('AMI event limit exceeded')


def recording_interval(events, duration):
    """Correlate actual recording events and decoded duration, never SQL update time."""
    starts = [r for r in events if r.get('Event') == 'ConfbridgeRecord']
    stops = [r for r in events if r.get('Event') == 'ConfbridgeStopRecord']
    if len(starts) != 1 or len(stops) != 1:
        raise ValueError('recording lifecycle events missing or ambiguous')
    start, stop = [datetime.datetime.fromisoformat(rows[0]['utc']).timestamp()
                   for rows in (starts, stops)]
    if stop <= start or abs(stop - start - duration) > 1:
        raise ValueError('recording duration disagrees with lifecycle events')
    return round(start), round(stop), round(duration)


def run(name, full=False):
    if not re.fullmatch(r'[a-z0-9-]{1,40}', name):
        raise ValueError('invalid run ID')
    if command(['hostname']).strip() != 'almadial-alma10':
        raise ValueError('wrong guest')
    if '0 active channels' not in cli('core show channels count'):
        raise ValueError('active calls before test')
    os.umask(0o077)
    out = ROOT / name
    out.mkdir(mode=0o700)
    def save(label, value):
        with (out / (label + '.json')).open('x') as target:
            json.dump(value, target, indent=2)
    marker = 'ALM6-' + name
    claim = sql("UPDATE vicidial_confbridges SET extension='" + marker + "' WHERE conf_exten=9600001 AND server_ip='127.0.0.1' AND extension=''; SELECT ROW_COUNT();")
    if claim != '1':
        raise ValueError('conference allocation unavailable')
    duplicate = sql("UPDATE vicidial_confbridges SET extension='ALM6-duplicate' WHERE conf_exten=9600001 AND server_ip='127.0.0.1' AND extension=''; SELECT ROW_COUNT();")
    save('allocation', {'claim_rows': claim, 'duplicate_claim_rows': duplicate, 'room': 9600001, 'owner': marker})
    ami = None
    recording_id = None
    try:
        ami = AMI((ROOT / 'ami-secret').read_text().strip())
        ami.pump()
        ami.send('Command', Command='core show version')
        ami.send('CoreStatus')
        ami.send('Ping')
        ami.pump()
        wrong = AMI('ALM6-deliberately-invalid')
        wrong.pump(3)
        save('negative-auth', wrong.rows)
        wrong.socket.close()
        def wait_for(ids, seconds=90):
            until = time.monotonic() + seconds
            while time.monotonic() < until:
                text = cli('confbridge list 9600001')
                if all('PJSIP/' + i + '-' in text for i in ids):
                    return text
                ami.pump(.2)
                time.sleep(.1)
            raise TimeoutError('expected conference participants absent')
        print('awaiting browser participants', flush=True)
        text = wait_for(['9051', '9052'])
        channels = cli('core show channels concise')
        save('join', {'conference': text, 'channels': channels})
        legs = {}
        for line in channels.splitlines():
            parts = line.split('!')
            for identity in ['9051', '9052']:
                if parts[0].startswith('PJSIP/' + identity + '-'):
                    legs[identity] = {'channel': parts[0], 'uniqueid': parts[-1]}
        # Explicit lab-owned application record, correlated with real AMI/channel IDs.
        # Native ConfBridge/MixMonitor control is tested, not VICIdial UI initiation.
        recording_id = sql("INSERT INTO recording_log (channel,server_ip,extension,start_time,start_epoch,filename,user,vicidial_id) VALUES ('" + legs['9051']['channel'] + "','127.0.0.1','9600001',NOW(),UNIX_TIMESTAMP(),'alm6-" + name + "','ALM6','" + legs['9051']['uniqueid'] + "'); SELECT LAST_INSERT_ID();")
        ami.send('ConfbridgeList', Conference='9600001')
        ami.send('Status')
        ami.require('ConfbridgeStartRecord', Conference='9600001', RecordFile='/var/spool/asterisk/monitor/alm6-' + name + '.wav')
        ami.pump(1)
        if full:
            text = wait_for(['9051', '9052', '9053'], 50)
            ami.send('ConfbridgeList', Conference='9600001')
            ami.pump(1)
            save('monitor', {'conference': text, 'channels': cli('core show channels concise')})
            if 'vici_monitor_user' not in text:
                raise ValueError('supervisor did not enter the generated monitor profile')
            for line in cli('core show channels concise').splitlines():
                parts = line.split('!')
                if parts[0].startswith('PJSIP/9053-'):
                    legs['9053'] = {'channel': parts[0], 'uniqueid': parts[-1]}
            ami.pump(9)
            supervisor = next(line.split('!')[0] for line in cli('core show channels concise').splitlines() if line.startswith('PJSIP/9053-'))
            ami.require('Redirect', Channel=supervisor, Context='alm6-barge', Exten='9600001', Priority='1')
            ami.pump(1)
            ami.send('ConfbridgeList', Conference='9600001')
            save('barge', {'conference': cli('confbridge list 9600001')})
            ami.pump(11)
            ami.require('Redirect', Channel=legs['9052']['channel'], Context='alm6-transfer', Exten='9600001', Priority='1')
            ami.pump(1)
            save('transfer', {'conference': cli('confbridge list 9600001'), 'channels': cli('core show channels concise')})
            ami.require('Hangup', Channel=legs['9052']['channel'], Cause='16')
            ami.pump(2)
            save('forced-hangup', {'conference': cli('confbridge list 9600001'), 'channels': cli('core show channels concise')})
        else:
            ami.pump(21)
        ami.require('ConfbridgeStopRecord', Conference='9600001')
        ami.pump(2)
        recordings = list(Path('/var/spool/asterisk/monitor').glob('alm6-' + name + '-*.wav'))
        if len(recordings) != 1:
            raise ValueError('expected exactly one conference recording')
        recording = recordings[0]
        with wave.open(str(recording), 'rb') as reader:
            duration = reader.getnframes() / reader.getframerate()
        open_fds = []
        for pid in command(['pgrep', '-x', 'asterisk']).splitlines():
            for fd in Path('/proc', pid, 'fd').iterdir():
                try:
                    if fd.resolve() == recording:
                        open_fds.append(str(fd))
                except FileNotFoundError:
                    pass
        if duration < 8 or open_fds:
            raise ValueError('recording is short or still open')
        start, stop, length = recording_interval(ami.rows, duration)
        sql("UPDATE recording_log SET start_time=FROM_UNIXTIME(" + str(start) + "),start_epoch=" + str(start)
            + ",end_time=FROM_UNIXTIME(" + str(stop) + "),end_epoch=" + str(stop)
            + ",length_in_sec=" + str(length) + ",length_in_min=" + str(length / 60)
            + ",location='" + str(recording) + "' WHERE recording_id=" + recording_id + ";")
        save('recording', {'path': str(recording), 'duration_seconds': duration, 'open_fds': open_fds,
                           'control': 'AMI ConfbridgeStartRecord/StopRecord; lab-owned recording_log row',
                           'metadata': sql('SELECT * FROM recording_log WHERE recording_id=' + recording_id + ';')})
        ami.pump(14)
        save('call-records', {'rows': sql("SELECT uniqueid,channel,start_time,end_time,length_in_sec FROM call_log WHERE channel LIKE 'PJSIP/905%' ORDER BY start_time DESC LIMIT 10;"), 'recording': sql('SELECT * FROM recording_log WHERE recording_id=' + recording_id + ';'), 'legs': legs})
    finally:
        if ami is not None:
            # Close only this experiment's three identities if an earlier phase failed.
            for line in cli('core show channels concise').splitlines():
                channel = line.split('!')[0]
                if re.fullmatch(r'PJSIP/905[123]-[0-9a-f]+', channel):
                    ami.send('Hangup', Channel=channel, Cause='16')
            ami.pump(1)
            ami.send('Logoff')
            ami.pump(.5)
            save('ami-events', ami.rows)
            ami.socket.close()
        sql("UPDATE vicidial_confbridges SET extension='' WHERE conf_exten=9600001 AND server_ip='127.0.0.1' AND extension='" + marker + "';")
        save('cleanup', {'channels': cli('core show channels count'), 'conference': cli('confbridge list'), 'contacts': cli('pjsip show contacts'), 'allocation': sql("SELECT * FROM vicidial_confbridges WHERE conf_exten=9600001;"), 'recording_id': recording_id})
    print('observer complete', flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run')
    parser.add_argument('--full', action='store_true')
    args = parser.parse_args()
    run(args.run, args.full)
