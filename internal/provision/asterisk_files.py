"""Fixed S04 filesystem adapter; no subprocesses or arbitrary copy destinations."""
import hashlib
import json
import os
from pathlib import Path
import pwd
import grp
import stat
import sys
import tarfile

W = Path('/var/lib/almadial/asterisk-s04')
T = W / 'asterisk-22.12.0'
p = {}

def sha(b):
    return hashlib.sha256(b).hexdigest()

def safe(path):
    path = Path(path)
    for x in [*reversed(path.parents), path]:
        if x.exists() or x.is_symlink():
            s = x.lstat()
            service_parent = str(x) == '/var/lib/asterisk' and s.st_uid == pwd.getpwnam('asterisk').pw_uid and s.st_gid == grp.getgrnam('asterisk').gr_gid and stat.S_IMODE(s.st_mode) == 0o750
            if stat.S_ISLNK(s.st_mode) or (s.st_uid != 0 and not service_parent) or s.st_mode & 0o022:
                raise ValueError('unsafe S04 path')
    return path

def read(path, perm=None, gid=0):
    path = safe(path)
    s = path.lstat()
    if not stat.S_ISREG(s.st_mode) or (perm is not None and stat.S_IMODE(s.st_mode) != perm) or s.st_gid != gid:
        raise ValueError('S04 metadata mismatch')
    return path.read_bytes()

def create(path, data, perm=0o600, gid=0):
    path = safe(path)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, perm)
    try:
        os.fchown(fd, 0, gid)
        os.fchmod(fd, perm)
        with os.fdopen(fd, 'wb', closefd=False) as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
    finally:
        os.close(fd)
    d = os.open(path.parent, os.O_DIRECTORY)
    os.fsync(d)
    os.close(d)

def receipt(name, content):
    create(W / (name + '.json'), json.dumps(content, sort_keys=True).encode())

def modules():
    result = []
    for name in p['modules']:
        paths = [T / d / (name + '.c') for d in ('res','apps','channels','pbx','bridges','codecs','formats','funcs')]
        src = next((x for x in paths if x.is_file()), None)
        built = src.with_suffix('.so') if src else None
        result.append((name, src, built))
    return result

def build():
    r = json.loads(read(W / 'build.json', 0o600))
    if r['binding'] != p['binding']:
        raise ValueError('stale build binding')
    if r['files'] != {str(x.relative_to(T)): sha(read(x)) for _, _, x in modules()}:
        raise ValueError('compiled module drift')
    if r['executable'] != sha(read(T / 'main/asterisk')):
        raise ValueError('compiled executable drift')
    return sha(json.dumps(r, sort_keys=True).encode())

def resource_paths():
    return [(T/'doc'/name, Path('/var/lib/asterisk/documentation')/name) for name in ('core-en_US.xml','appdocsxml.xslt','appdocsxml.dtd')]

def artifact_mode(dst):
    return 0o644 if dst.parent.name == 'documentation' else 0o755

def install_paths():
    return [(T / 'main/asterisk', Path('/usr/sbin/asterisk')), (T/'main/libasteriskssl.so.1',Path('/usr/lib/libasteriskssl.so.1')), (T/'main/libasteriskpj.so.2',Path('/usr/lib/libasteriskpj.so.2'))] + [(x, Path('/usr/lib/asterisk/modules') / (n+'.so')) for n, _, x in modules()] + resource_paths()

def install():
    build()
    r = json.loads(read(W / 'install.json', 0o600))
    if r['binding'] != p['binding']:
        raise ValueError('installed binding drift')
    legacy = [(src,dst) for src,dst in install_paths() if dst.name not in ('libasteriskssl.so.1','libasteriskpj.so.2') and dst.parent.name != 'documentation']
    libraries = [dst for _,dst in install_paths() if dst.name in ('libasteriskssl.so.1','libasteriskpj.so.2')]
    if r['files'] == {str(dst):sha(read(dst,0o755)) for _,dst in legacy} and not any(x.exists() or x.is_symlink() for x in libraries):
        return 'libraries-missing'
    base = [(src,dst) for src,dst in install_paths() if dst.parent.name != 'documentation']
    if r['files'] == {str(dst):sha(read(dst,0o755)) for _,dst in base} and not any(dst.exists() or dst.is_symlink() for _,dst in resource_paths()):
        return 'resources-missing'
    if r['files'] != {str(dst): sha(read(dst, artifact_mode(dst))) for _, dst in install_paths()}:
        raise ValueError('installed artifact drift')
    return sha(json.dumps(r, sort_keys=True).encode())

def config_files():
    gid = grp.getgrnam('asterisk').gr_gid
    out = {}
    for name, data in p['files'].items():
        out['/etc/asterisk/'+name+'.conf'] = ('#include almadial-s04-'+name+'.conf\n').encode()
        out['/etc/asterisk/almadial-s04-'+name+'.conf'] = data.encode()
    if p.get('s05_handoff', False):
        # The S05 reader may recognize only these closed include transitions.
        # S04's ordinary inspector and all S04 mutations keep the original
        # configuration contract. S04 fragments remain byte-for-byte intact.
        if sys.argv[1] != 'inspect':
            raise ValueError('S05 handoff is read-only')
        for name, include in (
                ('manager', 'almadial-s05-manager.conf'),
                ('extensions', 'extensions-vicidial.conf'),
                ('pjsip', 'pjsip-vicidial.conf'),
                ('confbridge', 'confbridge-vicidial.conf')):
            if name == 'extensions':
                out['/etc/asterisk/'+name+'.conf'] += b'[globals]\n'
            out['/etc/asterisk/'+name+'.conf'] += ('#include "'+include+'"\n').encode()
        out['/etc/asterisk/pjsip_wizard.conf'] = b'#include "pjsip_wizard-vicidial.conf"\n'
    return out, gid

def configuration():
    files, gid = config_files()
    exists = [Path(x).exists() or Path(x).is_symlink() for x in files]
    if not any(exists):
        return 'absent'
    if not all(exists):
        return 'invalid'
    for path, data in files.items():
        if read(path, 0o640, gid) != data:
            return 'invalid'
    # No additional writer may override this closed initial configuration.
    return sha(json.dumps({k:sha(v) for k,v in files.items()}, sort_keys=True).encode())

def unit():
    path = Path('/etc/systemd/system/asterisk.service')
    if Path('/usr/lib/systemd/system/asterisk.service').exists() or Path('/etc/systemd/system/asterisk.service.d').exists() or Path('/run/systemd/system/asterisk.service').exists() or Path('/run/systemd/system/asterisk.service.d').exists():
        return 'invalid'
    if not path.exists() and not path.is_symlink():
        return 'absent'
    return sha(read(path, 0o644)) if read(path, 0o644) == p['unit'].encode() else 'invalid'

def main():
    global p
    mode = sys.argv[1]
    p = json.loads(sys.argv[2])
    try:
        if mode == 'prepare':
            safe(W)
            W.mkdir(mode=0o700)
            (W / 'cache').mkdir(mode=0o700)
        elif mode == 'extract':
            if sha(read(W/'asterisk.tar.gz',0o600)) != p['archive'] or sha(read(W/'cache/pjproject-2.17.tar.bz2',0o600)) != p['pj']:
                raise ValueError('source changed')
            with tarfile.open(W/'asterisk.tar.gz','r:gz') as t:
                # Pinned archive; data filter additionally rejects unsafe members.
                t.extractall(W, filter='data')
        elif mode == 'seal-build':
            opts = read(T/'makeopts').decode()
            if not any(line.replace(' ','').replace('\t','') == 'PJPROJECT_BUNDLED=yes' for line in opts.splitlines()):
                raise ValueError('system PJPROJECT selected')
            if 'BUILD_NATIVE' in read(T/'menuselect.makeopts').decode().split('MENUSELECT_BUILDOPTS=')[-1].split('\n')[0]:
                raise ValueError('native CPU build disallowed')
            receipt('build', {'binding':p['binding'],'files':{str(x.relative_to(T)):sha(read(x)) for _,_,x in modules()},'executable':sha(read(T/'main/asterisk')),'makeopts':sha(read(T/'makeopts')),'menuselect':sha(read(T/'menuselect.makeopts'))})
        elif mode == 'install':
            build()
            for directory in ('/usr/lib/asterisk','/usr/lib/asterisk/modules','/var/lib/asterisk/documentation'):
                safe(directory).mkdir(mode=0o755,exist_ok=True)
            # Inspect every destination before copying the first byte.
            for src,dst in install_paths():
                staged = W/'dest'/str(dst).lstrip('/')
                if read(staged) != read(src) or safe(dst).exists():
                    raise ValueError('installation conflict or staging mismatch')
            for src,dst in install_paths():
                create(dst,read(src),artifact_mode(dst))
            receipt('install',{'binding':p['binding'],'files':{str(dst):sha(read(dst,artifact_mode(dst))) for _,dst in install_paths()}})
        elif mode == 'libraries':
            if install() != 'libraries-missing':
                raise ValueError('uncertain library reconciliation')
            pairs=[(src,dst) for src,dst in install_paths() if dst.name in ('libasteriskssl.so.1','libasteriskpj.so.2') or dst.parent.name == 'documentation']
            safe('/var/lib/asterisk/documentation').mkdir(mode=0o755,exist_ok=True)
            for src,dst in pairs:
                if read(W/'dest'/str(dst).lstrip('/')) != read(src) or safe(dst).exists():
                    raise ValueError('library staging conflict')
            create(W/'install.before-libraries.json',read(W/'install.json',0o600))
            for src,dst in pairs:
                create(dst,read(src),artifact_mode(dst))
            create(W/'install.next.json',json.dumps({'binding':p['binding'],'files':{str(dst):sha(read(dst,artifact_mode(dst))) for _,dst in install_paths()}},sort_keys=True).encode())
            os.replace(W/'install.next.json',W/'install.json')
            fd=os.open(W,os.O_DIRECTORY);os.fsync(fd);os.close(fd)
        elif mode == 'resources':
            if install() != 'resources-missing':
                raise ValueError('uncertain resource reconciliation')
            for src,dst in resource_paths():
                if read(W/'dest'/str(dst).lstrip('/')) != read(src) or safe(dst).exists():
                    raise ValueError('resource staging conflict')
            safe('/var/lib/asterisk/documentation').mkdir(mode=0o755,exist_ok=True)
            create(W/'install.before-resources.json',read(W/'install.json',0o600))
            for src,dst in resource_paths():
                create(dst,read(src),artifact_mode(dst))
            create(W/'install.next.json',json.dumps({'binding':p['binding'],'files':{str(dst):sha(read(dst,artifact_mode(dst))) for _,dst in install_paths()}},sort_keys=True).encode())
            os.replace(W/'install.next.json',W/'install.json')
            fd=os.open(W,os.O_DIRECTORY);os.fsync(fd);os.close(fd)
        elif mode == 'config':
            files,gid=config_files()
            for path in files:
                if safe(path).exists():
                    raise ValueError('configuration already exists; preserve')
            for path,data in files.items():
                create(path,data,0o640,gid)
        elif mode == 'unit':
            if unit() != 'absent':
                raise ValueError('service unit conflict')
            create('/etc/systemd/system/asterisk.service',p['unit'].encode(),0o644)
        elif mode == 'restart':
            receipt('restart',{'binding':p['binding'],'service':p['service']})
        elif mode == 'inspect':
            b,i='absent','absent'
            if W.exists() or W.is_symlink():
                safe(W)
                try: b=build()
                except (OSError,ValueError,KeyError,TypeError): b='invalid'
            present = Path('/usr/sbin/asterisk').exists() or Path('/usr/sbin/asterisk').is_symlink() or Path('/usr/lib/asterisk/modules').exists()
            if present:
                try: i=install()
                except (OSError,ValueError,KeyError,TypeError): i='invalid'
            restart=False
            try:
                r=json.loads(read(W/'restart.json',0o600))
                restart=r['binding']==p['binding'] and r['service']==p['service'] and bool(p['service'])
            except (OSError,ValueError,KeyError,TypeError): pass
            matrix=[]
            for name,src,obj in modules():
                matrix.append({'Name':name,'Source':bool(src),'Built':bool(obj and obj.is_file()),'Installed':(Path('/usr/lib/asterisk/modules')/(name+'.so')).is_file(),'Loaded':False,'Capability':'','Pass':False})
            print(json.dumps({'Build':b,'Install':i,'Configuration':configuration(),'Unit':unit(),'RestartVerified':restart,'Modules':matrix}))
        else:
            raise ValueError('unknown S04 operation')
    except (OSError,ValueError,KeyError,TypeError,tarfile.TarError):
        sys.exit('S04 filesystem conflict, unsafe metadata, or invalid artifact; preserve and reconcile')

if __name__ == "__main__":
    main()
