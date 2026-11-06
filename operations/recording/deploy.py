#!/usr/bin/python3
"""PR-01 scoped deployment. No credentials, calls, native source edits or media alias."""
import base64
import json
from pathlib import Path
import subprocess

source = Path(__file__).parent
evidence = source.parent / 'qa/evidence/recording-finalization'
baseline = json.loads((evidence / 'before.json').read_text())
payload = {target: dict(mode=mode, data=base64.b64encode((source/name).read_bytes()).decode())
           for name,target,mode in [
               ('finalize.py','/usr/local/sbin/alma-vicidial-recording',0o700),
               ('alma-vicidial-recording.service','/etc/systemd/system/alma-vicidial-recording.service',0o644),
               ('alma-vicidial-recording.timer','/etc/systemd/system/alma-vicidial-recording.timer',0o644)]}
remote = r'''
import datetime, hashlib, json, os
from pathlib import Path
import base64, shutil, socket, subprocess, sys, tarfile, tempfile, time

def run(*argv):
    return subprocess.run(argv,check=True,capture_output=True,text=True,timeout=60).stdout
def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()

assert os.geteuid()==0 and socket.gethostname()=='dialer01'
os.umask(0o077)
cron=Path('/etc/cron.d/vicidial')
apache=Path('/etc/httpd/conf.d/02-alma-vicidial-ssl.conf')
native=Path('/usr/share/astguiclient/AST_CRON_audio_1_move_VDonly.pl')
assert sha(native)=='a9837929fcda93d4ca238eb0f06ac5d2fc3198c3029c2e6132f29fe3091c25c0'
for p in (cron,apache):
    assert sha(p)==expected[str(p)], 'Concurrent configuration change; inspect before deployment'
for name in payload:
    assert not Path(name).exists(), 'Already deployed; inspect rather than overwrite'
assert not Path('/etc/alma-vicidial-recording.json').exists()
assert not Path('/var/lib/alma-vicidial-recording').exists()
assert '0 active channels' in run('asterisk','-rx','core show channels count'), 'Wait for idle test-server window'
health=json.loads(run('/usr/local/sbin/alma-vicidial-backup','health'))
assert health['outcome']=='ok' and health['last_success_verification_present'] and health['newest_complete_age_seconds']<86400, 'Inspect backup health before deployment'
# No backup creation, deletion, retention adjustment or archive copying.
assert shutil.disk_usage('/var/spool/asterisk/monitorDONE').free>1024**3
old='*/3 * * * * root /usr/share/astguiclient/AST_CRON_audio_1_move_mix.pl --MIX'
assert cron.read_text().count(old)==1
assert apache.read_text().count('</VirtualHost>')==1
assert 'PR01_PRIVATE_RECORDINGS' not in apache.read_text()
backup=Path('/root/alma-recording-pr01-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
backup.mkdir(mode=0o700)
manifest={str(p):dict(sha256=sha(p),mode=p.stat().st_mode&0o777,uid=p.stat().st_uid,gid=p.stat().st_gid) for p in (cron,apache)}
run('tar','--acls','--xattrs','--selinux','-C','/','-cpf',str(backup/'originals.tar'),*[str(p).lstrip('/') for p in (cron,apache)])
(backup/'manifest.json').write_text(json.dumps(manifest,indent=2))
with tempfile.TemporaryDirectory(dir=backup) as tmp:
    run('tar','--acls','--xattrs','--selinux','-C',tmp,'-xpf',str(backup/'originals.tar'))
    for name,want in manifest.items():
        p=Path(tmp)/name.lstrip('/')
        s=p.stat()
        assert sha(p)==want['sha256'] and (s.st_mode&0o777,s.st_uid,s.st_gid)==(want['mode'],want['uid'],want['gid'])
rollback=r"""#!/usr/bin/bash
set -euo pipefail
test "$(id -u)" = 0
test "$(hostname)" = dialer01
backup="$(cd -- "$(dirname -- "$0")" && pwd)"
if systemctl cat alma-vicidial-recording.timer >/dev/null 2>&1; then
  systemctl disable --now alma-vicidial-recording.timer
fi
if systemctl is-active --quiet alma-vicidial-recording.service; then
  echo 'Wait for recording service to finish before rollback' >&2
  exit 1
fi
if test -d /var/lib/alma-vicidial-recording && ! flock -n /var/lib/alma-vicidial-recording/lock true; then
  echo 'Manual processing is active; wait before rollback' >&2
  exit 1
fi
tar --acls --xattrs --selinux -C / -xpf "$backup/originals.tar"
httpd -t
systemctl reload httpd
rm -f /usr/local/sbin/alma-vicidial-recording /etc/alma-vicidial-recording.json
rm -f /etc/systemd/system/alma-vicidial-recording.service /etc/systemd/system/alma-vicidial-recording.timer
systemctl daemon-reload
echo 'PR01_ROLLBACK_COMPLETE: originals restored; raw/final audio, receipts and native database URL retained'
"""
(backup/'rollback.sh').write_text(rollback)
(backup/'rollback.sh').chmod(0o700)
run('bash','-n',str(backup/'rollback.sh'))
print('ROLLBACK_READY '+str(backup),file=sys.stderr,flush=True)
# Disable only the conflicting mixer. All other operational cron lines stay identical.
cron.write_text(cron.read_text().replace(old,'# PR01: replaced by alma-vicidial-recording.timer; old: '+old))
# Refuse deployment while any previously launched native audio job is still active.
for cmdline in Path('/proc').glob('[0-9]*/cmdline'):
    try:
        data=cmdline.read_bytes().split(b'\0')
    except FileNotFoundError:
        continue
    assert not any(x.endswith((b'/AST_CRON_audio_1_move_mix.pl',b'/AST_CRON_audio_1_move_VDonly.pl',b'/AST_CRON_audio_2_compress.pl')) for x in data), 'Audio processor active; keep cron disabled and retry after review'
state=Path('/var/lib/alma-vicidial-recording')
state.mkdir(mode=0o700)
policy=dict(automatic_start_epoch=int(time.time()),historical_hold=[1,2],publication='denied_pending_individual_authorization')
Path('/etc/alma-vicidial-recording.json').write_text(json.dumps(policy,indent=2)+'\n')
Path('/etc/alma-vicidial-recording.json').chmod(0o600)
for name,item in payload.items():
    p=Path(name)
    tmp=p.with_name(p.name+'.pr01-new')
    tmp.write_bytes(base64.b64decode(item['data']))
    tmp.chmod(item['mode'])
    os.replace(tmp,p)
deny="""
  # PR01_PRIVATE_RECORDINGS: native links are not individual media authorization.
  # No Alias is enabled; final WAVs remain root-readable pending a policy decision.
  <Directory "/var/spool/asterisk/monitorDONE">
    Options -Indexes -FollowSymLinks
    AllowOverride None
    Require all denied
  </Directory>
  <LocationMatch "(?i)^/RECORDINGS(?:/|$)">
    Require all denied
  </LocationMatch>
"""
apache.write_text(apache.read_text().replace('</VirtualHost>',deny+'</VirtualHost>'))
run('restorecon',str(cron),str(apache),*payload.keys(),str(state),'/etc/alma-vicidial-recording.json')
compile(Path('/usr/local/sbin/alma-vicidial-recording').read_text(),'recording','exec')
run('systemd-analyze','verify','/etc/systemd/system/alma-vicidial-recording.service','/etc/systemd/system/alma-vicidial-recording.timer')
run('httpd','-t')
run('systemctl','daemon-reload')
run('systemctl','reload','httpd')
run('systemctl','enable','--now','alma-vicidial-recording.timer')
print(json.dumps(dict(backup=str(backup),rollback_rehearsal='byte/metadata-identical extraction; bash -n PASS',
    policy=policy,apache_validation='Syntax OK',apache_action='graceful reload',
    modified={str(p):sha(p) for p in [cron,apache,*[Path(n) for n in payload],Path('/etc/alma-vicidial-recording.json')]},
    previous=manifest),indent=2))
'''
code = 'payload='+repr(payload)+'\nexpected='+repr(baseline['hashes'])+'\n'+remote
subprocess.run(['ssh','-T','-p','22222','-o','BatchMode=yes','root@2.29.153.161',
                '/usr/bin/python3','-B','-'],input=code,text=True,check=True)
