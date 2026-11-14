"""S09 native firewalld policy only; no zone, listener, route or NAT changes."""
import hashlib
import difflib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile

ROOT = Path('/etc/firewalld')
POLICY = ROOT/'policies/almadial-s09.xml'

def digest(value):
    return hashlib.sha256(json.dumps(value,sort_keys=True,separators=(',',':')).encode()).hexdigest()

def run(argv, optional=False):
    p=subprocess.run(argv,capture_output=True,text=True,timeout=45)
    if p.returncode and not optional:
        # Only fixed native firewall/service commands reach this boundary.
        # Bound diagnostics here; the outer Go runner also redacts its output.
        operation=Path(argv[0]).name
        if operation=='firewall-cmd': operation+=' '+' '.join(argv[1:])
        raise ValueError('native operation rejected: '+operation+' (exit '+str(p.returncode)+'); stdout: '+p.stdout.strip()[:1024]+'; stderr: '+p.stderr.strip()[:1024])
    return p.stdout.strip() if p.returncode==0 else None

def fw(*args, permanent=False, optional=False):
    return run(['/usr/bin/firewall-cmd']+(['--permanent'] if permanent else [])+list(args),optional)

def safe(path):
    for p in [path]+list(path.parents):
        if not p.exists():
            if p.is_symlink(): raise ValueError('symlink')
            continue
        s=p.lstat()
        if stat.S_ISLNK(s.st_mode) or s.st_uid!=0 or s.st_mode&0o022:
            raise ValueError('unsafe firewall ownership')
        if p==path and p.exists() and not stat.S_ISREG(s.st_mode):
            raise ValueError('not regular')

def clean_nft(value):
    if isinstance(value,dict):
        return {k:clean_nft(v) for k,v in value.items() if k not in ('handle','packets','bytes')}
    if isinstance(value,list): return [clean_nft(v) for v in value]
    return value

def nft():
    return json.loads(run(['/usr/sbin/nft','-j','list','ruleset']))['nftables']

def foreign():
    files=[]
    for p in sorted(ROOT.rglob('*')):
        if p.is_symlink(): raise ValueError('foreign firewall symlink requires review')
        if p==POLICY or p.is_dir(): continue
        safe(p)
        files.append([str(p),p.stat().st_mode&0o777,hashlib.sha256(p.read_bytes()).hexdigest()])
    rules=[]
    for item in nft():
        if 'metainfo' in item: continue
        obj=next(iter(item.values()))
        table=obj.get('table', obj.get('name') if 'table' in item else None)
        if obj.get('family')=='inet' and table=='firewalld': continue
        rules.append(clean_nft(item))
    # Unknown external filter hooks can accept/drop traffic ahead of policy.
    # Preserve them and refuse to claim an isolated baseline automatically.
    if rules: raise ValueError('foreign nftables state requires review')
    return digest(files)

def service():
    states=[]
    for operation in ('is-active','is-enabled'):
        p=subprocess.run(['/usr/bin/systemctl',operation,'firewalld'],capture_output=True,text=True,timeout=15)
        states.append(p.stdout.strip())
    return states

def normalized(text):
    # firewalld 2.4 adds "active" to the default-zone header as well.
    # Ignore only this runtime annotation, retaining every policy field and
    # every other header attribute (including "default" and "disabled").
    def header(match):
        attributes=[a for a in match[2].split(', ') if a!='active']
        return match[1]+(' ('+', '.join(attributes)+')' if attributes else '')
    return re.sub(r'^([^\s]+) \(([^)]+)\)$',header,text or '',flags=re.MULTILINE)

def normalized_policy(text):
    value=normalized(text)
    # Disabled vendor policies have a runtime-only header annotation.
    # Retain the semantic disable field and reject an inconsistent marker.
    if '  disable: yes' in value.splitlines():
        value=re.sub(r'^([^\s]+) \(disabled\)$',r'\1',value,flags=re.MULTILINE)
    return value

def foreign_runtime():
    zones=normalized(fw('--list-all-zones'))
    permanent_zones=normalized(fw('--list-all-zones',permanent=True))
    # NetworkManager owns active connection bindings, not firewalld zone XML.
    # Accept an extra runtime binding only when its active NM profile agrees.
    permanent_interfaces={}
    zone=None
    for line in permanent_zones.splitlines():
        if line and not line[0].isspace(): zone=line.split(' (',1)[0]
        elif line.startswith('  interfaces:'): permanent_interfaces[zone]=line
    lines=zones.splitlines()
    zone=None
    for index,line in enumerate(lines):
        if line and not line[0].isspace(): zone=line.split(' (',1)[0]
        elif line.startswith('  interfaces:') and zone in permanent_interfaces:
            persistent=permanent_interfaces[zone]
            extra=set(line.split(':',1)[1].split())-set(persistent.split(':',1)[1].split())
            for interface in sorted(extra):
                uuid=run(['/usr/bin/nmcli','-g','GENERAL.CON-UUID','device','show',interface])
                if not uuid or not re.fullmatch(r'[0-9a-fA-F-]{36}',uuid):
                    raise ValueError('foreign runtime interface lacks active NetworkManager profile')
                profile_zone=run(['/usr/bin/nmcli','-g','connection.zone','connection','show','uuid',uuid])
                if (profile_zone or fw('--get-default-zone'))!=zone:
                    raise ValueError('foreign runtime interface zone differs from NetworkManager profile')
            if extra:
                remaining=set(line.split(':',1)[1].split())-extra
                if remaining==set(persistent.split(':',1)[1].split()): lines[index]=persistent
    zones='\n'.join(lines)
    permanent_zones='\n'.join(permanent_zones.splitlines())
    if zones!=permanent_zones:
        difference='\n'.join(difflib.unified_diff(permanent_zones.splitlines(),zones.splitlines(),fromfile='permanent',tofile='runtime',n=1))
        raise ValueError('foreign runtime-only zone state: '+difference[:4096])
    for option in ('--direct --get-all-rules','--direct --get-all-passthroughs','--get-ipsets'):
        args=option.split()
        if fw(*args)!=fw(*args,permanent=True): raise ValueError('foreign runtime-only configuration')
    runtime=set((fw('--get-policies') or '').split())-{'almadial-s09'}
    permanent=set((fw('--get-policies',permanent=True) or '').split())-{'almadial-s09'}
    if runtime!=permanent: raise ValueError('foreign runtime-only policy')
    # A preceding foreign policy can override the managed isolation decision.
    for name in sorted(runtime):
        observed=normalized_policy(fw('--info-policy='+name))
        persistent=normalized_policy(fw('--info-policy='+name,permanent=True))
        if observed!=persistent:
            difference='\n'.join(difflib.unified_diff(persistent.splitlines(),observed.splitlines(),fromfile='permanent',tofile='runtime',n=1))
            raise ValueError('foreign policy divergence '+name+': '+difference[:4096])
        # firewalld 2.4 exposes policy priority only through permanent CLI.
        # The full runtime/permanent policy was compared immediately above.
        priority=int(fw('--policy='+name,'--get-priority',permanent=True))
        if priority<=-20000: raise ValueError('competing policy priority')
    return digest([fw('--list-all-zones',permanent=True),sorted(permanent)])

def contracts(p):
    r=p['Request']
    if r['Mode'] not in ('isolated','remote') or not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,14}',r['Interface']): raise ValueError('topology')
    if run(['/usr/sbin/getenforce'])!='Enforcing': raise ValueError('SELinux')
    links=json.loads(run(['/usr/sbin/ip','-j','address','show','dev',r['Interface']]))
    if r['ManagementAddress'] not in [a['local'] for l in links for a in l['addr_info']]: raise ValueError('management address changed')
    ssh=run(['/usr/sbin/sshd','-T'])
    ports=[int(l.split()[1]) for l in ssh.splitlines() if l.startswith('port ')]
    if ports!=[r['SSHPort']]: raise ValueError('SSH configuration changed')
    sockets=run(['/usr/sbin/ss','-H','-lntup'])
    if not any('sshd' in l and (':'+str(r['SSHPort'])+' ') in l for l in sockets.splitlines()): raise ValueError('SSH not listening')
    for l in sockets.splitlines():
        fields=l.split()
        if len(fields)<6: raise ValueError('socket format')
        if any('"'+name+'"' in l for name in ('httpd','asterisk','mariadbd','perl')) and fields[0]=='tcp':
            remote_https = r['Mode']=='remote' and '"httpd"' in l and fields[4]=='0.0.0.0:443'
            if not remote_https and not fields[4].startswith(('127.0.0.1:','[::1]:')): raise ValueError('public application listener')
    safe(POLICY)
    if POLICY.exists() and POLICY.read_text()!=p['XML']: raise ValueError('conflicting owned policy')
    run(['/usr/bin/firewall-offline-cmd','--check-config'])
    state=service()
    if state[0]=='active': foreign_runtime()
    elif state[0]!='inactive' or state[1] not in ('disabled','enabled'): raise ValueError('ambiguous firewalld state')
    return state

def policy_health(p):
    if not POLICY.exists() or POLICY.read_text()!=p['XML'] or service()!=['active','enabled']: return False
    if run(['/usr/sbin/matchpathcon','-V',str(POLICY)],True) is None: return False
    if 'almadial-s09' not in (fw('--get-active-policies') or '').split(): return False
    if normalized(fw('--info-policy=almadial-s09'))!=normalized(fw('--info-policy=almadial-s09',permanent=True)): return False
    ports={str(p['Request']['SSHPort'])+'/tcp'}
    if p['Request'].get('Mode')=='remote':
        ports.update(('443/tcp',str(p['Request']['RTPStart'])+'-'+str(p['Request']['RTPEnd'])+'/udp'))
    for permanent in (False,True):
        args={'--get-target':'DROP','--get-priority':'-20000','--list-ingress-zones':'ANY','--list-egress-zones':'HOST','--list-ports':str(p['Request']['SSHPort'])+'/tcp','--list-protocols':'icmp ipv6-icmp'}
        for arg,want in args.items():
            got=fw('--policy=almadial-s09',arg,permanent=permanent or arg in ('--get-target','--get-priority'))
            if arg=='--list-ports':
                if set(got.split())!=ports: return False
            elif arg=='--list-protocols':
                if set(got.split())!=set(want.split()): return False
            elif got!=want: return False
        for arg in ('--list-services','--list-source-ports','--list-rich-rules','--list-forward-ports'):
            if fw('--policy=almadial-s09',arg,permanent=permanent): return False
    # Observe generated inet rules, rather than equating daemon-active with filtering.
    rules=nft()
    owned=[v['rule'] for v in rules if 'rule' in v and 'almadial' in v['rule'].get('chain','')]
    body=json.dumps(owned,sort_keys=True)
    return bool(owned) and '"drop"' in body and '"accept"' in body and '"tcp"' in body and str(p['Request']['SSHPort']) in body

def inspect(p):
    state=contracts(p)
    f=foreign()
    files=digest(p['XML']) if POLICY.exists() else 'absent'
    services=digest(state) if state==['active','enabled'] else 'absent'
    health=digest([p['XML'],'native inet policy verified']) if policy_health(p) else 'absent'
    return dict(Foreign=f,Files=files,Services=services,Health=health,Current=digest([f,files,state,health]))

def write_policy(text):
    POLICY.parent.mkdir(mode=0o755,exist_ok=True)
    safe(POLICY)
    fd,name=tempfile.mkstemp(prefix='.almadial-',dir=POLICY.parent)
    try:
        with os.fdopen(fd,'w') as f:
            os.fchmod(f.fileno(),0o644);f.write(text);f.flush();os.fsync(f.fileno())
        os.replace(name,POLICY)
        run(['/usr/sbin/restorecon',str(POLICY)])
        fd=os.open(POLICY.parent,os.O_RDONLY|os.O_DIRECTORY)
        try:os.fsync(fd)
        finally:os.close(fd)
    finally:
        Path(name).unlink(missing_ok=True)

def activate(p):
    before=inspect(p)
    if before['Current']!=p['ExpectedCurrent']: raise ValueError('stale network transaction')
    if policy_health(p): return
    old=POLICY.read_text() if POLICY.exists() else None
    prior=service()
    active=prior[0]=='active'
    # A native transient timer restores precisely the prior owned definition and
    # prior inactive service state if the caller disappears during activation.
    rollback="from pathlib import Path;import subprocess;p=Path("+repr(str(POLICY))+");"+("p.write_text("+repr(old)+");" if old is not None else "p.unlink(missing_ok=True);")+("subprocess.run(['/usr/bin/firewall-cmd','--reload'],check=True)" if active else "subprocess.run(['/usr/bin/systemctl','stop','firewalld'],check=True);subprocess.run(['/usr/bin/systemctl',"+repr("enable" if prior[1]=="enabled" else "disable")+",'firewalld'],check=True)")
    unit='almadial-s09-recovery'
    run(['/usr/bin/systemd-run','--unit='+unit,'--on-active=120s','/usr/bin/python3','-B','-c',rollback])
    try:
        write_policy(p['XML'])
        run(['/usr/bin/firewall-offline-cmd','--check-config'])
        if active: fw('--reload')
        else: run(['/usr/bin/systemctl','enable','--now','firewalld'])
        after=inspect(p)
        if after['Foreign']!=before['Foreign'] or after['Health']=='absent': raise ValueError('network postcondition')
        # Cancel only after fresh effective-policy and management-port checks.
        # Independent remote reachability remains a separate qualification.
        run(['/usr/bin/systemctl','stop',unit+'.timer'])
    except Exception as error:
        run(['/usr/bin/python3','-B','-c',rollback])
        run(['/usr/bin/systemctl','stop',unit+'.timer'],True)
        raise ValueError('firewall activation rejected; prior managed state restored: '+str(error)) from None

if __name__=='__main__':
    try:
        p=json.load(sys.stdin)
        if sys.argv[1]=='network-activate': activate(p)
        elif sys.argv[1]!='network-inspect': raise ValueError('mode')
        print(json.dumps(inspect(p)))
    except Exception as e:
        print('S09 '+str(e),file=sys.stderr)
        sys.exit(1)
