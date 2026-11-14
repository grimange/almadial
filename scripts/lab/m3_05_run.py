#!/usr/bin/env python3
"""One bounded ALM-27 experiment; invoke through almadial-media-lab only.

Inspection is read-only. The experiment needs the parent's M1 lock and intent.
Raw browser profiles remain protected; no SIP headers or private keys exported.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import time

os.umask(0o077)
os.environ.update(HOME=pwd.getpwuid(os.getuid()).pw_dir, PATH='/usr/bin:/bin:/snap/bin')
ROOT = Path(__file__).resolve().parents[2]
VM = 'almadial-alm16-rpm'
PROFILE = json.loads(Path(sys.argv[3]).read_text()) if len(sys.argv) == 4 else None
if PROFILE:
    import ipaddress, re
    assert (PROFILE['task'] == 'xu7wxvch94ii2iw38hpb8d58' and re.fullmatch(r'almadial-alm29-clean-[a-z]+', PROFILE['vm'])) or (PROFILE['task'] == 'it8aq4utqix0uz8sjgxrtssk' and PROFILE['vm'] == 'almadial-alm34-bootstrap-dev' and PROFILE['machine'] == 'd8e73788e2a9328a8655aa707be8816a')
    assert re.fullmatch(r'[a-f0-9]{32}', PROFILE['machine'])
    assert ipaddress.IPv4Address(PROFILE['address']).is_private
    VM = PROFILE['vm']
E = Path(sys.argv[2])


def native(mode):
    p = subprocess.run(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', 'python3', '-', mode] + ([json.dumps(PROFILE)] if PROFILE else []),
                       input=(ROOT/'scripts/lab/m3_05_native.py').read_text(),
                       text=True, capture_output=True, timeout=90)
    if p.returncode:
        raise RuntimeError('native ' + mode + ': ' + p.stderr[-1600:])
    return json.loads(p.stdout)


def save(name, data):
    with (E/name).open('x') as f:
        json.dump(data, f, indent=2, sort_keys=True)


def clean(state):
    assert not state['temporary_rtp_allowance'] and not state['fixture_context']
    assert not state['channels'].strip()
    assert not state['media_backup'] and not state['media_overlay']


def inspect():
    state = native('inspect')
    clean(state)
    return state


def verify():
    state = inspect()
    before = json.loads((E/'before.json').read_text())
    assert state['preserved'] == before['preserved']
    browser = json.loads((E/'browser.json').read_text())
    assert browser['pass'], 'real browser experiment did not pass'
    return {'preserved': state['preserved'], 'cleaned': True,
            'browser_sha256': hashlib.sha256((E/'browser.json').read_bytes()).hexdigest()}


def experiment():
    save('before.json', inspect())
    home = E/'browser-home'
    nss = home/'.pki/nssdb'
    nss.mkdir(parents=True)
    code = "import json,pathlib; p=json.loads(pathlib.Path('/tmp/alm25-corrected-qualification/accepted-tls.json').read_text()); print(pathlib.Path(p['Request']['TrustFile']).read_text())"
    if PROFILE:
        code = "from pathlib import Path; print(Path('/etc/pki/tls/almadial-s08/current/trust.pem').read_text())"
    ca = subprocess.check_output(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', 'python3', '-c', code], timeout=30)
    (E/'public-lab-ca.pem').write_bytes(ca)
    for argv in [['certutil', '-N', '--empty-password', '-d', 'sql:'+str(nss)],
                 ['certutil', '-A', '-d', 'sql:'+str(nss), '-n', 'AlmaDial lab CA', '-t', 'C,,', '-i', str(E/'public-lab-ca.pem')]]:
        subprocess.run(argv, check=True, capture_output=True, timeout=30)
    sys.path.insert(0, str(ROOT/'scripts/lab'))
    import m0_03_signal as signal
    signal.fixture(E/'signal-a.wav', seconds=60)
    signal.PATTERN = (3, 1, 2, 0, 2, 3, 0, 1, 3, 0, 1, 2, 1, 0, 3, 2)
    signal.fixture(E/'signal-b.wav', seconds=60)
    tunnel = None
    capture = None
    try:
        save('prepared.json', native('prepare'))
        capture = subprocess.Popen(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', '/usr/bin/timeout', '60', '/usr/sbin/tcpdump', '-n', '-l', '-i', 'eth0', '-c', '100', 'udp and host 10.160.90.1'], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        tunnel = subprocess.Popen(['/usr/bin/python3', str(ROOT/'scripts/lab/m3_05_tunnel.py')] + (['--vm', VM] if PROFILE else []), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        time.sleep(1)
        assert tunnel.poll() is None, 'loopback tunnel failed'
        p = subprocess.run(['/usr/bin/node', str(ROOT/'scripts/lab/m3_05_browser.cjs'), str(E)] + ([VM] if PROFILE else []), capture_output=True, text=True, timeout=220)
        save('browser-command.json', {'exit_code': p.returncode, 'stdout': p.stdout[-2000:], 'stderr': p.stderr[-2000:]})
        assert p.returncode == 0, 'browser experiment failed; see protected browser.json'
    finally:
        if tunnel:
            tunnel.terminate()
            tunnel.wait(timeout=10)
        save('cleanup.json', native('cleanup'))
        if capture:
            capture.terminate()
            out, err = capture.communicate(timeout=10)
            save('packet-headers.json', {'stdout': out, 'stderr': err, 'payload_capture': False})
    return verify()


if __name__ == '__main__':
    assert sys.argv[1] in ('inspect', 'verify', 'experiment')
    print(json.dumps(globals()[sys.argv[1]](), sort_keys=True))
