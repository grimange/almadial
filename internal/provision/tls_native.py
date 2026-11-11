"""S08 closed supplied-certificate transaction. Private input arrives on stdin only."""
import os,sys,json,stat,hashlib,base64,ssl,socket,subprocess,pwd,grp,time
from pathlib import Path
R=Path('/etc/pki/tls/almadial-s08')
AP=Path('/etc/asterisk/http.conf')
VH=Path('/etc/httpd/almadial-s08.conf')
LD=Path('/etc/httpd/conf.modules.d/99-almadial-s08.conf')

def sha(b): return hashlib.sha256(b).hexdigest()
def run(a):
    p=subprocess.run(a,capture_output=True,timeout=30)
    if p.returncode: raise ValueError('native command failed: '+a[0])
    return p.stdout.decode().strip()
def fsync_dir(p):
    fd=os.open(p,os.O_DIRECTORY|os.O_NOFOLLOW);os.fsync(fd);os.close(fd)
def safe(p):
    for x in [*reversed(p.parents),p]:
        try: s=x.lstat()
        except FileNotFoundError: continue
        if not (stat.S_ISDIR(s.st_mode) or x==p and stat.S_ISREG(s.st_mode)) or s.st_uid!=0 or s.st_mode&0o022:
            raise ValueError('unsafe TLS path')
def read(p,mode=None,gid=None,label=None):
    safe(p)
    fd=os.open(p,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK)
    try:
        s=os.fstat(fd)
        if not stat.S_ISREG(s.st_mode) or s.st_size>4<<20 or (mode is not None and stat.S_IMODE(s.st_mode)!=mode) or (gid is not None and s.st_gid!=gid): raise ValueError('TLS metadata')
        if label and os.getxattr(fd,'security.selinux').decode().strip('\0').split(':')[2]!=label: raise ValueError('TLS label')
        return os.read(fd,(4<<20)+1)
    finally: os.close(fd)
def write(p,data,mode,gid,label):
    safe(p.parent)
    tmp=p.with_name('.'+p.name+'.s08-next')
    fd=os.open(tmp,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,mode)
    try:
        os.fchown(fd,0,gid);os.fchmod(fd,mode)
        os.setxattr(fd,'security.selinux',('system_u:object_r:'+label+':s0').encode())
        with os.fdopen(fd,'wb',closefd=False) as f: f.write(data);f.flush();os.fsync(f.fileno())
    finally: os.close(fd)
    os.replace(tmp,p);fsync_dir(p.parent)
def pointer(name,target):
    p=R/name;tmp=R/('.'+name+'.next')
    os.symlink(target,tmp);os.replace(tmp,p);fsync_dir(R)
def current():
    if not R.exists(): return None
    safe(R)
    p=R/'current'
    if not p.is_symlink():
        if p.exists(): raise ValueError('foreign TLS pointer')
        return None
    s=p.lstat();target=os.readlink(p)
    if s.st_uid!=0 or len(target)!=64 or any(c not in '0123456789abcdef' for c in target): raise ValueError('foreign TLS generation')
    return target

def inventory():
    out=[]
    if R.exists():
        safe(R)
        for p in sorted(R.rglob('*')):
            s=p.lstat()
            if p.name in ('current','previous') and p.is_symlink():
                target=os.readlink(p)
                if p.parent!=R or s.st_uid!=0 or len(target)!=64 or any(c not in '0123456789abcdef' for c in target): raise ValueError('TLS pointer')
                out.append([str(p),target]);continue
            safe(p)
            if stat.S_ISREG(s.st_mode): out.append([str(p),s.st_uid,s.st_gid,stat.S_IMODE(s.st_mode),sha(read(p))])
            elif stat.S_ISDIR(s.st_mode): out.append([str(p),s.st_uid,s.st_gid,stat.S_IMODE(s.st_mode)])
    for p in (AP,VH,LD):
        if p.exists(): out.append([str(p),sha(read(p))])
    return sha(json.dumps(out,separators=(',',':')).encode())

def contracts(p):
    gid=grp.getgrnam('asterisk').gr_gid
    if read(AP,0o640,gid,'etc_t') not in (p['Base'].encode(),p['HTTP'].encode()): raise ValueError('foreign Asterisk HTTP wrapper')
    for path,data in ((VH,p['Vhost']),(LD,p['Loader'])):
        if path.exists() and read(path,0o644,0,'httpd_config_t')!=data.encode(): raise ValueError('foreign TLS vhost')
    if R.exists():
        s=R.stat()
        if stat.S_IMODE(s.st_mode)!=0o750 or s.st_gid!=gid: raise ValueError('TLS directory permissions')
    return gid

def context(p):
    ctx=ssl.create_default_context(cafile=str(R/current()/'trust.pem'))
    ctx.minimum_version=ssl.TLSVersion.TLSv1_2
    return ctx

def request(p,port,path,auth=None,websocket=False):
    ctx=context(p)
    with socket.create_connection(('127.0.0.1',port),5) as raw:
        with ctx.wrap_socket(raw,server_hostname=p['Hostname']) as s:
            if sha(s.getpeercert(binary_form=True))!=p['Leaf']: raise ValueError('wrong active certificate')
            headers='GET '+path+' HTTP/1.1\r\nHost: '+p['Hostname']+'\r\nConnection: '+('Upgrade' if websocket else 'close')+'\r\n'
            if auth: headers+='Authorization: Basic '+base64.b64encode(('6666:'+auth).encode()).decode()+'\r\n'
            if websocket: headers+='Upgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: MDEyMzQ1Njc4OWFiY2RlZg==\r\nSec-WebSocket-Protocol: sip\r\n'
            s.sendall((headers+'\r\n').encode()); data=b''
            while len(data)<2<<20:
                b=s.recv(65536)
                if not b: break
                data+=b
                if websocket and b'\r\n\r\n' in data:
                    # RFC 6455 masked empty close: no SIP or session is sent.
                    s.sendall(bytes([0x88,0x80,0,0,0,0]));s.recv(1024)
                    break
            if len(data)>=2<<20: raise ValueError('TLS response too large')
            return data.decode(errors='replace')

def health(p):
    if current()!=p['Generation']: return False
    for path,data in ((AP,p['HTTP']),(VH,p['Vhost']),(LD,p['Loader'])):
        if not path.exists() or read(path)!=data.encode(): return False
    for port,process in ((443,'httpd'),(8089,'asterisk')):
        lines=[l for l in run(['/usr/sbin/ss','-H','-lntp']).splitlines() if l.split()[3].endswith(':'+str(port))]
        if len(lines)!=1 or lines[0].split()[3]!='127.0.0.1:'+str(port) or '"'+process+'"' not in lines[0]: return False
    try:
        admin=request(p,443,'/vicidial/admin.php')
        if ' 401 ' not in admin.split('\r\n')[0] or 'www-authenticate: basic' not in admin.lower(): return False
        key=read(Path('/etc/almadial/secrets/admin'),0o600,0).decode().strip()
        admin=request(p,443,'/vicidial/admin.php',key)
        visible=' '.join(__import__('re').sub('<[^>]*>',' ',admin.lower()).split())
        if ' 200 ' not in admin.split('\r\n')[0] or 'logout (6666)' not in visible or 'initial setup' in visible: return False
        agent=request(p,443,'/agc/vicidial.php')
        if ' 200 ' not in agent.split('\r\n')[0] or 'phone_login' not in agent.lower(): return False
        for path in ('/agc/options.php','/vicidial/options.php'):
            if ' 403 ' not in request(p,443,path).split('\r\n')[0]: return False
        wss=request(p,8089,'/ws',websocket=True)
        if ' 101 ' not in wss.split('\r\n')[0] or 'Sec-WebSocket-Accept: BACScCJPNqyz+UBoqMH89VmURoA='.lower() not in wss.lower(): return False
        return True
    except (OSError,ValueError,ssl.SSLError): return False

def inspect(p):
    contracts(p)
    c=current()
    if c:
        g=R/c;safe(g)
        for name,field in (('fullchain.pem','Chain'),('key.pem','Key'),('trust.pem','Trust')):
            raw=read(g/name,0o640,grp.getgrnam('asterisk').gr_gid,'cert_t')
            if c==p['Generation'] and raw!=base64.b64decode(p[field],validate=True): raise ValueError('installed TLS generation drift')
    ready=health(p)
    return dict(Current=inventory(),Files=sha(b'S08 files') if c==p['Generation'] else 'absent',Services=sha(b'S08 endpoints') if ready else 'absent',Health=p['CertificateDigest'] if ready else 'absent',Foreign=sha(b'S08 closed local TLS ownership'))

def reload_services():
    run(['/usr/sbin/httpd','-t','-f','/etc/httpd/conf/almadial-s05.conf'])
    run(['/usr/bin/systemctl','reload','httpd'])
    text=run(['/usr/sbin/asterisk','-rx','module reload http'])
    if 'not found' in text.lower() or 'no such' in text.lower(): raise ValueError('HTTP core reload failed')

def activate(p):
    if inspect(p)['Current']!=p['ExpectedCurrent']: raise ValueError('stale TLS transaction')
    gid=contracts(p)
    if health(p): return
    old=current()
    before={path:read(path) if path.exists() else None for path in (AP,VH,LD)}
    if not R.exists():
        R.mkdir(mode=0o750);os.chown(R,0,gid);os.chmod(R,0o750)
        run(['/usr/sbin/restorecon','-F',str(R)])
    gen=R/p['Generation']
    if not gen.exists():
        gen.mkdir(mode=0o750);os.chown(gen,0,gid);os.chmod(gen,0o750)
        run(['/usr/sbin/restorecon','-F',str(gen)])
    for name,field in (('fullchain.pem','Chain'),('key.pem','Key'),('trust.pem','Trust')):
        raw=base64.b64decode(p[field],validate=True)
        dst=gen/name
        if dst.exists():
            if read(dst,0o640,gid,'cert_t')!=raw: raise ValueError('TLS generation conflict')
        else: write(dst,raw,0o640,gid,'cert_t')
    # ssl also verifies the actual staged certificate/key pair before publication.
    check=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);check.load_cert_chain(str(gen/'fullchain.pem'),str(gen/'key.pem'))
    try:
        pointer('current',p['Generation'])
        for path,data in ((AP,p['HTTP']),(VH,p['Vhost']),(LD,p['Loader'])):
            if before[path]!=data.encode(): write(path,data.encode(),0o640 if path==AP else 0o644,gid if path==AP else 0,'etc_t' if path==AP else 'httpd_config_t')
        reload_services()
        for _ in range(50):
            if health(p): break
            time.sleep(.1)
        else: raise ValueError('TLS endpoint verification failed')
        if old and old!=p['Generation']: pointer('previous',old)
    except Exception:
        # Bounded reversal of precisely this transaction, never generic rollback.
        if old: pointer('current',old)
        elif (R/'current').is_symlink(): (R/'current').unlink();fsync_dir(R)
        for path,data in before.items():
            if data is None:
                if path.exists(): path.unlink();fsync_dir(path.parent)
            elif read(path)!=data: write(path,data,0o640 if path==AP else 0o644,gid if path==AP else 0,'etc_t' if path==AP else 'httpd_config_t')
        reload_services()
        raise ValueError('TLS activation failed; previous configuration restored, verify endpoints')

def main():
    try:
        p=json.load(sys.stdin)
        if os.geteuid()!=0 or len(sys.argv)!=2 or sys.argv[1] not in ('tls-inspect','tls-activate'): raise ValueError('S08 operation')
        if len(p['Generation'])!=64 or any(c not in '0123456789abcdef' for c in p['Generation']): raise ValueError('S08 generation identity')
        if sys.argv[1]=='tls-activate': activate(p)
        print(json.dumps(inspect(p)));return 0
    except Exception:
        sys.stderr.write('S08 TLS operation failed; inspect protected postconditions; no key material logged\n');return 1
if __name__=='__main__': sys.exit(main())
