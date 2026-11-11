#!/usr/bin/env python3
"""Bounded real agent acceptance, invoked through the existing M1 guard only."""
import base64
import hashlib
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import time
import zipfile
import zlib

os.umask(0o077)
os.environ.update(HOME=pwd.getpwuid(os.getuid()).pw_dir, PATH="/usr/bin:/bin:/snap/bin")
ROOT = Path(__file__).resolve().parents[2]
E = Path(sys.argv[2])
VM = 'almadial-alm16-rpm'
ARCHIVE = Path('/home/ra/almadial-lab/evidence/m0-08/20261106T192640Z/vicidial-r4037.zip')
sys.path.insert(0, str(ROOT / 'scripts/lab'))
import m0_07_assets as assets


def guest(mode, payload=None):
    script = (ROOT / 'scripts/lab/m3_05_agent_guest.py').read_text()
    # Compress the pinned source payload below Multipass stdin's transport bound.
    code = 'import sys,json,io,zlib,base64; data=json.loads(zlib.decompress(base64.b64decode(sys.stdin.buffer.read())));sys.argv=["fixture",data["mode"]];sys.stdin=io.StringIO(json.dumps(data["payload"]));exec(compile(data["script"],"agent-fixture","exec"))'
    wire = base64.b64encode(zlib.compress(json.dumps({'script': script, 'mode': mode, 'payload': payload}).encode())).decode()
    p = subprocess.run(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', 'python3', '-c', code], input=wire, text=True, capture_output=True, timeout=200)
    if p.returncode:
        raise RuntimeError('agent ' + mode + ': ' + p.stderr[-2500:])
    return json.loads(p.stdout)


def save(name, value):
    with (E/name).open('x') as f:
        json.dump(value, f, indent=2, sort_keys=True)
        f.flush()
        os.fsync(f.fileno())


def inspect():
    result = guest('inspect')
    assert not result['present'] and not result['database_present'] and not result['channels']
    return result


def verify():
    state = inspect()
    assert state == json.loads((E/'before.json').read_text())
    browser = json.loads((E/'agent-browser.json').read_text())
    assert browser['pass'], 'agent browser checks incomplete'
    return {'state': 'PASS', 'preserved': state, 'browser_sha256': hashlib.sha256((E/'agent-browser.json').read_bytes()).hexdigest()}


def payload():
    raw = ARCHIVE.read_bytes()
    assert hashlib.sha256(raw).hexdigest() == 'ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058'
    z = zipfile.ZipFile(ARCHIVE)
    def read(name):
        return z.read(next(n for n in z.namelist() if n.endswith('/' + name)))
    files = {'web/agc/vicidial.php': assets.agent_lifecycle_overlay(read('www/agc/vicidial.php')),
             'web/agc/calendar_db.js': assets.calendar_overlay(read('www/agc/calendar_db.js')),
             'bin/AST_update.pl': assets.updater_overlay(read('bin/AST_update.pl')),
             'bin/AST_manager_listen.pl': assets.call_runtime_overlay('AST_manager_listen.pl', read('bin/AST_manager_listen.pl'))}
    for name in assets.SESSION_PINS:
        files['web/agc/' + name] = assets.session_overlay(name, read('www/agc/' + name))
    for name in assets.MANAGER_PINS:
        files['bin/' + name] = assets.manager_overlay(name, read('bin/' + name))
    native = (ROOT / 'internal/provision/phone_native.py').read_text()
    launcher = native.split("LAUNCHER=r'''", 1)[1].split("'''.encode()", 1)[0].encode()
    files['web/phone/index.php'] = launcher.replace(b'/var/www/html/agc/', b'/opt/almadial-alm27/web/agc/').replace(b'alm_s10_9061', b'9062').replace(b"'callerid'=>'9061'", b"'callerid'=>'9062'")
    files = {name: data.replace(b'/etc/astguiclient-m007-updater.conf', b'/etc/astguiclient-alm27.conf').replace(b'/etc/almadial-m007-ami.secret', b'/opt/almadial-alm27/ami.secret') for name, data in files.items()}
    save('derivation.json', {'upstream_r4037_zip_sha256': hashlib.sha256(raw).hexdigest(), 'reviewed_m0_derivations': True, 'files': {k: hashlib.sha256(v).hexdigest() for k,v in files.items()}})
    return {'intent': 'M1 media.02.agent durable intent in parent execution journal', 'files': {k: base64.b64encode(v).decode() for k,v in files.items()}}


def experiment():
    save('before.json', inspect())
    nss = E/'browser-home/.pki/nssdb'
    nss.mkdir(parents=True)
    ca = subprocess.check_output(['/snap/bin/multipass','exec',VM,'--','sudo','python3','-c',"import json,pathlib;p=json.loads(pathlib.Path('/tmp/alm25-corrected-qualification/accepted-tls.json').read_text());print(pathlib.Path(p['Request']['TrustFile']).read_text())"], timeout=30)
    (E/'public-lab-ca.pem').write_bytes(ca)
    for argv in [['certutil','-N','--empty-password','-d','sql:'+str(nss)], ['certutil','-A','-d','sql:'+str(nss),'-n','AlmaDial lab CA','-t','C,,','-i',str(E/'public-lab-ca.pem')]]:
        subprocess.run(argv,check=True,capture_output=True,timeout=30)
    import m0_03_signal as signal
    signal.fixture(E/'signal-a.wav', seconds=180)
    tunnel = None
    try:
        save('prepared.json', guest('prepare', payload()))
        tunnel = subprocess.Popen(['/usr/bin/python3',str(ROOT/'scripts/lab/m3_05_tunnel.py'),'--agent-fixture'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        time.sleep(1)
        assert tunnel.poll() is None
        p = subprocess.run(['/usr/bin/node',str(ROOT/'scripts/lab/m3_05_agent_browser.cjs'),str(E)],text=True,capture_output=True,timeout=310)
        save('browser-command.json',{'exit_code':p.returncode,'stdout':p.stdout[-1000:],'stderr':p.stderr[-1500:]})
        save('agent-final-state.json',guest('state'))
        assert p.returncode == 0, 'actual agent workflow failed; see protected result'
    finally:
        if tunnel:
            tunnel.terminate()
            tunnel.wait(timeout=10)
        if guest('inspect')['present']:
            save('cleanup.json',guest('cleanup'))
    return verify()


if __name__ == '__main__':
    assert sys.argv[1] in ('inspect','verify','experiment','state','dispatch')
    print(json.dumps(guest(sys.argv[1]) if sys.argv[1] in ('state','dispatch') else globals()[sys.argv[1]](),sort_keys=True))
