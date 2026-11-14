#!/usr/bin/env python3
"""Guest half of the ALM-27 local media experiment, invoked by its locked harness.

A temporary endpoint include, runtime dialplan and ten-minute source-scoped RTP
allowance are restored during cleanup. Phone rows, secrets and permanent policy
are unchanged. The original PJSIP wrapper is backed up before modification.
The parent must hold M1 execution ownership and persist intent before prepare.
"""
import hashlib
import json
import os
import grp
from pathlib import Path
import subprocess
import sys
import tempfile

MACHINE = '0fffa9a4683f35b696f0b6ee7330734b'
MEDIA_ADDRESS = '10.160.90.233'
if len(sys.argv) == 3:
    import ipaddress, re
    target = json.loads(sys.argv[2])
    assert target['task'] == 'xu7wxvch94ii2iw38hpb8d58' or (target['task'] == 'it8aq4utqix0uz8sjgxrtssk' and target['vm'] == 'almadial-alm34-bootstrap-dev' and target['machine'] == 'd8e73788e2a9328a8655aa707be8816a')
    assert re.fullmatch(r'[a-f0-9]{32}', target['machine'])
    assert ipaddress.IPv4Address(target['address']).is_private
    MACHINE, MEDIA_ADDRESS = target['machine'], target['address']
CONTEXT = 'alm-s11-media'
PARENT = 'alm-s10-no-calls'
ROOM = '9600271'
RULE = 'rule family="ipv4" source address="10.160.90.1/32" port port="10000-10019" protocol="udp" accept'
WRAPPER = Path('/etc/asterisk/pjsip.conf')
BACKUP = Path('/run/almadial-s11-pjsip.original')
OVERLAY = Path('/etc/asterisk/almadial-s11-media.conf')
INCLUDE = b'\n#include almadial-s11-media.conf\n'
MEDIA_CONFIG = ("""; ALM-27 bounded LAN experiment; restored during cleanup.
[alm_s10_9061](+type=endpoint)
media_address={address}
bind_rtp_to_media_address=yes
rtp_port_start=10000
rtp_port_end=10019
""".format(address=MEDIA_ADDRESS)).encode()
ROUTES = [
    ('9271', 1, 'Answer()'),
    ('9271', 2, 'AGI(agi://127.0.0.1:4577/call_log)'),
    ('9271', 3, 'MixMonitor(/var/spool/asterisk/monitor/alm-s11-${UNIQUEID}.wav,r(/var/spool/asterisk/monitor/alm-s11-${UNIQUEID}-rx.wav)t(/var/spool/asterisk/monitor/alm-s11-${UNIQUEID}-tx.wav))'),
    ('9271', 4, 'ConfBridge(9600271,default_bridge,default_user)'),
    ('9271', 5, 'Hangup()'),
    ('h', 1, 'StopMixMonitor()'),
    ('h', 2, 'AGI(agi://127.0.0.1:4577/call_log--HVcauses--PRI-----NODEBUG-----${HANGUPCAUSE}-----${DIALSTATUS}-----${DIALEDTIME}-----${ANSWEREDTIME})'),
]


def run(argv):
    proc = subprocess.run(argv, capture_output=True, text=True, timeout=30)
    if proc.returncode:
        raise RuntimeError(argv[0] + ' failed: ' + proc.stderr[-1200:])
    return proc.stdout


def cli(command):
    output = run(['/usr/sbin/asterisk', '-rx', command])
    if 'failed.' in output or 'No such command' in output:
        raise RuntimeError('Asterisk operation rejected: ' + output[-1200:])
    return output


def replace_wrapper(expected, content):
    """Replace only the observed owned wrapper, preserving metadata atomically."""
    assert not WRAPPER.is_symlink(), 'symlink wrapper is not owned by this fixture'
    assert WRAPPER.read_bytes() == expected, 'wrapper changed concurrently; refuse overwrite'
    metadata = WRAPPER.stat()
    fd, temporary = tempfile.mkstemp(prefix='.alm-s11-', dir=WRAPPER.parent)
    try:
        with os.fdopen(fd, 'wb') as output:
            os.fchmod(output.fileno(), metadata.st_mode & 0o7777)
            os.fchown(output.fileno(), metadata.st_uid, metadata.st_gid)
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        assert WRAPPER.read_bytes() == expected, 'wrapper changed while staging'
        os.replace(temporary, WRAPPER)
        directory = os.open(WRAPPER.parent, os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    run(['/usr/sbin/restorecon', str(WRAPPER)])


def inspect():
    assert Path('/etc/machine-id').read_text().strip() == MACHINE
    assert run(['/usr/sbin/getenforce']).strip() == 'Enforcing'
    assert run(['/usr/bin/systemctl', 'is-active', 'asterisk', 'firewalld', 'almadial-s06-fastagi']).split() == ['active'] * 3
    paths = ['/etc/asterisk/pjsip.conf', '/etc/asterisk/almadial-s10-phone.conf',
             '/etc/firewalld/policies/almadial-s09.xml', '/etc/almadial/secrets/phone-s10']
    hashes = {p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in paths}
    query = subprocess.run(['/usr/bin/firewall-cmd', '--policy=almadial-s09', '--query-rich-rule=' + RULE], capture_output=True, text=True, timeout=15)
    assert query.returncode in (0, 1)
    route = run(['/usr/sbin/asterisk', '-rx', 'dialplan show ' + CONTEXT])
    return {'host': MACHINE, 'preserved': hashes, 'temporary_rtp_allowance': query.returncode == 0,
            'fixture_context': '9271' in route, 'route': route,
            'media_backup': BACKUP.exists(), 'media_overlay': OVERLAY.exists(),
            'channels': cli('core show channels concise'),
            'conference': cli('confbridge list ' + ROOM)}


def prepare():
    before = inspect()
    assert not before['channels'].strip(), 'active channels prevent fixture preparation'
    assert not before['temporary_rtp_allowance'] and not before['fixture_context'], 'existing fixture needs inspection'
    assert not BACKUP.exists() and not OVERLAY.exists(), 'foreign or interrupted media configuration'
    with BACKUP.open('xb') as f:
        os.chmod(BACKUP, 0o600)
        f.write(WRAPPER.read_bytes())
        f.flush()
        os.fsync(f.fileno())
    with OVERLAY.open('xb') as f:
        os.fchmod(f.fileno(), 0o640)
        os.fchown(f.fileno(), 0, grp.getgrnam('asterisk').gr_gid)
        f.write(MEDIA_CONFIG)
    run(['/usr/sbin/restorecon', str(OVERLAY)])
    original = BACKUP.read_bytes()
    replace_wrapper(original, original + INCLUDE)
    cli('module reload res_pjsip.so')
    endpoint = cli('pjsip show endpoint alm_s10_9061')
    assert MEDIA_ADDRESS in endpoint and '10019' in endpoint, 'media endpoint override rejected'
    # A timeout on this narrow LAN allowance is independent of the test process.
    run(['/usr/bin/firewall-cmd', '--policy=almadial-s09', '--add-rich-rule=' + RULE, '--timeout=600'])
    for extension, priority, application in ROUTES:
        cli('dialplan add extension ' + extension + ',' + str(priority) + ',' + application + ' into ' + CONTEXT)
    cli('dialplan add include ' + CONTEXT + ' into ' + PARENT)
    after = inspect()
    assert after['temporary_rtp_allowance'] and after['fixture_context']
    assert all(after['preserved'][p] == h for p, h in before['preserved'].items() if p != str(WRAPPER))
    return after


def cleanup():
    # Refuse to terminate arbitrary calls. Only this context/room's channels qualify.
    for line in cli('core show channels concise').splitlines():
        fields = line.split('!')
        if len(fields) > 5 and fields[1] == CONTEXT and fields[0].startswith('PJSIP/alm_s10_9061-'):
            cli('channel request hangup ' + fields[0])
    state = inspect()
    if state['fixture_context']:
        if CONTEXT in cli('dialplan show ' + PARENT):
            cli('dialplan remove include ' + CONTEXT + ' from ' + PARENT)
        cli('dialplan remove context ' + CONTEXT)
    if state['temporary_rtp_allowance']:
        run(['/usr/bin/firewall-cmd', '--policy=almadial-s09', '--remove-rich-rule=' + RULE])
    if BACKUP.exists():
        original = BACKUP.read_bytes()
        assert WRAPPER.read_bytes() in (original, original + INCLUDE), 'wrapper changed concurrently; refuse overwrite'
        if OVERLAY.exists():
            assert OVERLAY.read_bytes() == MEDIA_CONFIG, 'media overlay changed concurrently'
        replace_wrapper(WRAPPER.read_bytes(), original)
        if OVERLAY.exists():
            OVERLAY.unlink()
        cli('module reload res_pjsip.so')
        BACKUP.unlink()
    after = inspect()
    assert not after['temporary_rtp_allowance'] and not after['fixture_context']
    assert not after['media_backup'] and not after['media_overlay']
    assert not after['channels'].strip(), 'call cleanup incomplete'
    return after


if __name__ == '__main__':
    operation = sys.argv[1]
    assert operation in ('inspect', 'prepare', 'cleanup')
    print(json.dumps(globals()[operation](), sort_keys=True))
