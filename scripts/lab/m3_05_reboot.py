#!/usr/bin/env python3
"""One separately authorized reboot checkpoint, called only by the M1 guard.

No reboot on inspect/verify/resume. The caller holds ownership and durable intent.
The checkpoint asserts an idle, restored fixture and the recorded recovery point.
"""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

os.umask(0o077)
VM = 'almadial-alm16-rpm'
MACHINE = '0fffa9a4683f35b696f0b6ee7330734b'
SNAPSHOT = 'alm25-pre-firewall-20261111'
E = Path(sys.argv[2])
GUEST = r'''
import hashlib,json,pathlib,subprocess,sys
def run(argv):
 return subprocess.check_output(argv,text=True,timeout=30).strip()
assert pathlib.Path('/etc/machine-id').read_text().strip()=='0fffa9a4683f35b696f0b6ee7330734b'
assert run(['getenforce'])=='Enforcing'
services=['mariadb','asterisk','httpd','php-fpm','firewalld','almadial-s06-fastagi','sshd']
assert run(['systemctl','is-active']+services).split()==['active']*len(services)
assert not run(['asterisk','-rx','core show channels concise'])
assert '9271' not in run(['asterisk','-rx','dialplan show alm-s11-media'])
assert not run(['firewall-cmd','--policy=almadial-s09','--list-rich-rules'])
assert not pathlib.Path('/run/almadial-s11-pjsip.original').exists()
assert not pathlib.Path('/etc/asterisk/almadial-s11-media.conf').exists()
assert len(run(['rpm','-qa']).splitlines())==731
parent=run(['systemctl','show','almadial-s06-fastagi','-p','MainPID','--value'])
assert len(run(['pgrep','-P',parent]).splitlines())==5
paths=['/etc/asterisk/pjsip.conf','/etc/asterisk/almadial-s10-phone.conf','/etc/almadial/secrets/phone-s10','/etc/firewalld/policies/almadial-s09.xml']
hashes={p:hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest() for p in paths}
tls=json.loads(pathlib.Path('/tmp/alm25-corrected-qualification/accepted-tls.json').read_text())['Request']
ca=tls['TrustFile']
http=run(['curl','--silent','--show-error','--cacert',ca,'--output','/dev/null','--write-out','%{http_code}','https://localhost/vicidial/admin.php'])
assert http in ('200','401')
probe=subprocess.run(['openssl','s_client','-connect','127.0.0.1:8089','-servername','localhost','-verify_hostname','localhost','-verify_return_error','-CAfile',ca],input='',text=True,capture_output=True,timeout=15)
assert probe.returncode==0, 'Asterisk trusted TLS failed'
print(json.dumps({'machine':pathlib.Path('/etc/machine-id').read_text().strip(),'services':services,'fastagi_workers':5,'rpm_count':731,'selinux':'Enforcing','channels':0,'temporary_media_files':0,'temporary_rules':0,'preserved':hashes,'https_status':http,'asterisk_tls_verified':True},sort_keys=True))
'''


def guest(code):
    return subprocess.check_output(['/snap/bin/multipass', 'exec', VM, '--', 'sudo', 'python3', '-c', code], text=True, timeout=100)


def inspect():
    return json.loads(guest(GUEST))


def save(name, value):
    with (E / name).open('x') as output:
        json.dump(value, output, indent=2, sort_keys=True)
        output.flush()
        os.fsync(output.fileno())


def boot_id():
    return guest("from pathlib import Path; print(Path('/proc/sys/kernel/random/boot_id').read_text().strip())").strip()


def verify():
    before = json.loads((E / 'before.json').read_text())
    after = inspect()
    checkpoint = json.loads((E / 'checkpoint.json').read_text())
    assert before == after, 'material post-reboot state differs'
    assert checkpoint['boot_before'] != boot_id() == checkpoint['boot_after'], 'reboot identity checkpoint differs'
    return {'state': 'PASS', 'preserved': after, 'checkpoint_sha256': hashlib.sha256((E / 'checkpoint.json').read_bytes()).hexdigest()}


def experiment():
    snapshots = json.loads(subprocess.check_output(['/snap/bin/multipass', 'info', VM, '--snapshots', '--format', 'json'], text=True, timeout=30))
    assert SNAPSHOT in snapshots['info'][VM]['snapshots']
    save('snapshot.json', snapshots)
    before = inspect()
    first_boot = boot_id()
    save('before.json', before)
    save('reboot-intent.json', {'machine': MACHINE, 'snapshot': SNAPSHOT,
                               'authority': 'Explicit operator authorization in ALM-27 conversation for one controlled idle lab reboot',
                               'command': ['/snap/bin/multipass', 'restart', VM], 'boot_before': first_boot})
    operation = subprocess.run(['/snap/bin/multipass', 'restart', VM], text=True, capture_output=True, timeout=180)
    save('restart-command.json', {'exit_code': operation.returncode, 'stdout': operation.stdout, 'stderr': operation.stderr})
    assert operation.returncode == 0, 'reboot command failed; do not retry automatically'
    deadline = time.monotonic() + 120
    while True:
        try:
            after = inspect()
            break
        except (subprocess.SubprocessError, ValueError):
            if time.monotonic() >= deadline:
                raise
            time.sleep(2)
    last_boot = boot_id()
    assert first_boot != last_boot and before == after
    save('checkpoint.json', {'state': 'PASS', 'boot_before': first_boot, 'boot_after': last_boot, 'preserved': after,
                             'boundary': 'Accepted persistent backend and restored synthetic fixture; not a full agent-flow acceptance'})
    return verify()


if __name__ == '__main__':
    assert sys.argv[1] in ('inspect', 'verify', 'experiment')
    print(json.dumps(globals()[sys.argv[1]](), sort_keys=True))
