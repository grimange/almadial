#!/usr/bin/env python3
"""ALM-46 controlled off-host media scenario for the existing guarded S11 tool.

Uses installed permanent remote configuration. No tunnels, media overlays,
firewall exceptions or temporary application/database clones are created.
"""
import hashlib
import base64
import math
import re
import time
import wave
import array
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
E = Path(sys.argv[2])
PROFILE = json.loads(Path(sys.argv[3]).read_text())
VM = PROFILE['vm']
assert PROFILE['task'] == 'a1doi4qsyn1lspqtydw84smh'
assert VM == 'almadial-alm46-remote-dev'
assert PROFILE['machine'] == '436ca8ae883439638507ab64b02d98e8'
os.umask(0o077)


def guest(code):
    return subprocess.check_output(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', 'python3', '-c', code], timeout=40)


def save(name, value):
    with (E/name).open('x') as out:
        json.dump(value, out, indent=2, sort_keys=True)


def inspect():
    raw = guest('''import hashlib,json,pathlib,subprocess
P=pathlib.Path
assert P('/etc/machine-id').read_text().strip() == '436ca8ae883439638507ab64b02d98e8'
def run(*a): return subprocess.check_output(a,text=True,timeout=20).strip()
assert run('/usr/sbin/getenforce')=='Enforcing'
assert run('/usr/bin/systemctl','is-active','asterisk','httpd','firewalld','almadial-s06-fastagi').split()==['active']*4
channels=run('/usr/sbin/asterisk','-rx','core show channels concise')
assert not channels, 'active calls prevent bounded experiment'
paths=['/etc/asterisk/pjsip.conf','/etc/asterisk/extensions.conf','/etc/asterisk/almadial-s10-phone.conf','/etc/asterisk/almadial-s10-remote.conf','/etc/httpd/almadial-s08.conf','/etc/firewalld/policies/almadial-s09.xml','/etc/almadial/secrets/phone-s10']
assert not P('/etc/asterisk/almadial-s11-media.conf').exists()
print(json.dumps({'preserved':{p:hashlib.sha256(P(p).read_bytes()).hexdigest() for p in paths},'topology':'controlled off-host; installed remote profile; no tunnel; no temporary overlay','public_host':'NOT_RUN','channels':channels},sort_keys=True))
''')
    return json.loads(raw)


def recordings():
    # Export only this installed synthetic route's numeric-uniqueid WAVs.
    data = json.loads(guest("""import base64,json,pathlib,re,subprocess
root=pathlib.Path('/var/spool/asterisk/monitor')
rows=[]
for p in sorted(root.glob('alm-remote-*.wav')):
    m=re.fullmatch(r'alm-remote-([0-9]+[.][0-9]+)(-rx|-tx)?[.]wav',p.name)
    assert m and p.is_file() and not p.is_symlink() and p.stat().st_size < 16<<20
    uid=m.group(1)
    q="SELECT JSON_ARRAY(uniqueid,channel,number_dialed,start_epoch,end_epoch,length_in_sec,extension) FROM asterisk.call_log WHERE uniqueid='"+uid+"';"
    row=subprocess.check_output(['/usr/bin/mariadb','--no-defaults','--protocol=SOCKET','--socket=/var/lib/mysql/mysql.sock','--user=root','--batch','--skip-column-names','--raw'],input=q.encode(),timeout=10).decode().strip()
    rows.append({'name':p.name,'uid':uid,'call':json.loads(row) if row else None,'wav':base64.b64encode(p.read_bytes()).decode()})
print(json.dumps(rows))
"""))
    baseline = set(json.loads((E/'recording-baseline.json').read_text()))
    result = []
    directory = E/'recordings'
    directory.mkdir(mode=0o700)
    for row in data:
        if row['name'] in baseline:
            continue
        assert re.fullmatch(r'alm-remote-[0-9]+[.][0-9]+(-rx|-tx)?[.]wav',row['name'])
        raw=base64.b64decode(row.pop('wav'),validate=True)
        path=directory/row['name']
        with path.open('xb') as out: out.write(raw)
        with wave.open(str(path),'rb') as wav:
            assert wav.getnchannels()==1 and wav.getsampwidth()==2
            duration=wav.getnframes()/wav.getframerate()
            samples=array.array('h',wav.readframes(wav.getnframes()))
            if sys.byteorder!='little': samples.byteswap()
            rms=math.sqrt(sum((v/32768)**2 for v in samples)/max(1,len(samples)))
        call=row['call']
        row.update(sha256=hashlib.sha256(raw).hexdigest(),duration_seconds=duration,rms=rms,
                   pass_=duration>=8 and rms>=.015 and bool(call) and call[0]==row['uid'] and call[2]=='9271' and call[4]>call[3] and call[5]>=8 and call[6]=='alm_s10_9061')
        result.append(row)
    uids={r['uid'] for r in result}
    passed=len(uids)==2 and len(result)==6 and all(r['pass_'] for r in result)
    save('recording-analysis.json',{'pass':passed,'rows':result,'scope':'synthetic recordings only; protected; not for Git'})
    assert passed, 'recordings must decode, contain audio and correlate with completed call_log rows'


def verify():
    state = inspect()
    assert state == json.loads((E/'before.json').read_text())
    browser = json.loads((E/'browser.json').read_text())
    assert browser['pass'], 'actual browser media did not complete'
    assert json.loads((E/'audio-analysis.json').read_text())['pass'], 'two-way signal verification failed'
    assert json.loads((E/'recording-analysis.json').read_text())['pass'], 'recording correlation failed'
    return {'preserved':state['preserved'], 'browser_sha256':hashlib.sha256((E/'browser.json').read_bytes()).hexdigest(), 'public_host':'NOT_RUN'}


def experiment():
    save('before.json', inspect())
    save('recording-baseline.json',json.loads(guest("import json,pathlib; print(json.dumps([p.name for p in pathlib.Path('/var/spool/asterisk/monitor').glob('alm-remote-*.wav')]))")))
    nss = E/'browser-home/.pki/nssdb'
    nss.mkdir(parents=True)
    (E/'ca.pem').write_bytes(guest("from pathlib import Path; print(Path('/etc/pki/tls/almadial-s08/current/trust.pem').read_text())"))
    for args in [['certutil','-N','--empty-password','-d','sql:'+str(nss)], ['certutil','-A','-d','sql:'+str(nss),'-n','ALM46 controlled test CA','-t','C,,','-i',str(E/'ca.pem')]]:
        subprocess.run(args, check=True, capture_output=True, timeout=30)
    sys.path.insert(0,str(ROOT/'scripts/lab'))
    import m0_03_signal as signal
    original = signal.PATTERN
    signal.fixture(E/'signal-a.wav', seconds=90)
    other = (3,1,2,0,2,3,0,1,3,0,1,2,1,0,3,2)
    signal.PATTERN = other
    signal.fixture(E/'signal-b.wav', seconds=90)
    # Also hold the actual target installation lock throughout the experiment.
    lock = subprocess.Popen(['/snap/bin/multipass','exec',VM,'--','sudo','python3','-c',"import fcntl,sys; f=open('/var/lib/almadial-install/apply.lock','a'); fcntl.flock(f,fcntl.LOCK_EX|fcntl.LOCK_NB); print('locked',flush=True); sys.stdin.buffer.read()"],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    capture = None
    try:
        assert lock.stdout.readline().strip()==b'locked', 'installer owns target lock'
        capture = subprocess.Popen(['/snap/bin/multipass','exec',VM,'--','sudo','/usr/bin/timeout','90','/usr/sbin/tcpdump','-n','-l','-i','eth0','-c','100','udp and portrange 10000-10019'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        p = subprocess.run(['/usr/bin/node',str(ROOT/'scripts/lab/m3_05_browser.cjs'),str(E),VM,sys.argv[3]],capture_output=True,text=True,timeout=210)
        save('browser-command.json',{'exit_code':p.returncode,'stdout':p.stdout[-1000:],'stderr':p.stderr[-1000:]})
        assert p.returncode==0, 'browser failed; protected browser.json has phase evidence'
        # Cross-participant captures must match the other participant's input,
        # ruling out local echo or two independently non-silent but unrelated legs.
        signal.PATTERN=original
        ab=signal.compare(E/'a-local.wav',E/'b-remote.wav')
        signal.PATTERN=other
        ba=signal.compare(E/'b-local.wav',E/'a-remote.wav')
        save('audio-analysis.json',{'a_to_b':ab,'b_to_a':ba,'pass':ab['state']==ba['state']=='PASS'})
        assert ab['state']==ba['state']=='PASS', 'meaningful two-way audio verification failed'
    finally:
        lock.stdin.close()
        lock.wait(timeout=15)
        if capture:
            capture.terminate()
            out,err=capture.communicate(timeout=15)
            save('packet-headers.json',{'stdout':out.decode(),'stderr':err.decode(),'payload_capture':False})
    for _ in range(20):
        if not guest("import subprocess; print(subprocess.check_output(['/usr/sbin/asterisk','-rx','core show channels concise']).decode().strip())").strip(): break
        time.sleep(.5)
    recordings()
    return verify()


if __name__ == '__main__':
    assert sys.argv[1] in ('inspect','verify','experiment')
    print(json.dumps(globals()[sys.argv[1]](),sort_keys=True))
