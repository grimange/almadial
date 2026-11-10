# S07 reuses accepted S05 descriptor-based file primitives, unchanged.
import base64
import subprocess
import urllib.request
import urllib.error
import re
import html

class WebNoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

def web_admin_ready(status, text):
    visible = html.unescape(re.sub(r'<[^>]*>', ' ', text))
    visible = ' '.join(visible.lower().split())
    return status == 200 and 'initial setup' not in visible and 'logout (6666)' in visible

S07_PATHS = ('/etc/httpd/almadial-s07.conf', '/etc/httpd/conf.modules.d/99-almadial-s07.conf')
OWNED_PATHS['web07'] = S07_PATHS

def web_run(argv, data=None):
    p = subprocess.run(argv, input=data, capture_output=True, timeout=30)
    if p.returncode:
        raise ValueError('S07 native validation failed: ' + argv[0])
    return p.stdout.decode().strip()

def web_sql(q):
    return web_run(['/usr/bin/mariadb','--no-defaults','--protocol=SOCKET',
        '--socket=/var/lib/mysql/mysql.sock','--user=root','--batch',
        '--skip-column-names','--raw','--connect-timeout=5'], q.encode())

def web_setup(mutate):
    # Protected random S03 password is already installed. Never run upstream's
    # initial-settings form: it rotates this password and rewrites EVERY phone.
    parent = open_directory('/etc/almadial/secrets')
    try: key = read_at(parent, 'admin', 'root', 0o600, None)
    finally: os.close(parent)
    if key is None or len(key) != 49 or not __import__('re').fullmatch(b'[0-9a-f]{48}\n',key):
        raise ValueError('S07 protected administrator reference unavailable')
    condition = "user='6666' AND pass='"+key.decode().strip()+"' AND active='Y' AND user_level=9 AND user_group='ADMIN' AND force_change_password='N'"
    if web_sql('SELECT COUNT(*) FROM asterisk.vicidial_users WHERE '+condition+';') != '1':
        raise ValueError('S07 existing secured administrator required')
    setting = web_sql('SELECT first_login_trigger FROM asterisk.system_settings;')
    if setting not in ('Y','N'):
        raise ValueError('S07 ambiguous first-login setting')
    if mutate and setting == 'Y':
        web_sql("LOCK TABLES asterisk.system_settings WRITE, asterisk.vicidial_users READ;\nUPDATE asterisk.system_settings SET first_login_trigger='N' WHERE first_login_trigger='Y' AND (SELECT COUNT(*) FROM asterisk.vicidial_users WHERE "+condition+")=1;\nUNLOCK TABLES;\n")
        setting = web_sql('SELECT first_login_trigger FROM asterisk.system_settings;')
        if setting != 'N': raise ValueError('S07 setup postcondition failed')
    return setting, key.decode().strip()

def web_request(path, key=None):
    req=urllib.request.Request('http://127.0.0.1'+path)
    if key: req.add_header('Authorization','Basic '+base64.b64encode(('6666:'+key).encode()).decode())
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}), WebNoRedirect())
    try: response=opener.open(req,timeout=10)
    except urllib.error.HTTPError as e: response=e
    with response:
        raw=response.read((2<<20)+1)
        text=raw.decode(errors='replace').lower()
        if len(raw)>2<<20 or any(x in text for x in ('fatal error','uncaught ','database connect error','<?php')):
            raise ValueError('S07 application response failed')
        return response.status, response.headers.get('X-AlmaDial-Web'),text

def web_foreign():
    entries=[]
    for directory in ('/etc/httpd/conf','/etc/httpd/conf.d','/etc/httpd/conf.modules.d','/etc/php.d','/etc/php-fpm.d'):
        for p in sorted(Path(directory).rglob('*')):
            if str(p) in S07_PATHS: continue
            s=p.lstat()
            if stat.S_ISLNK(s.st_mode): raise ValueError('S07 foreign configuration symlink')
            if stat.S_ISREG(s.st_mode):
                entries.append([str(p),s.st_uid,s.st_gid,stat.S_IMODE(s.st_mode),sha(p.read_bytes())])
    return sha(json.dumps(entries,separators=(',',':')).encode())

def web_files(files, mutate):
    if sorted(f['Target'] for f in files)!=sorted(S07_PATHS): raise ValueError('S07 exact file set')
    staged=[]
    for f in files:
        raw=base64.b64decode(f['Content'],validate=True)
        if len(raw)>1<<20 or sha(raw)!=f['SHA256']: raise ValueError('S07 file identity')
        if not verify_existing(f,observation(f)): staged.append((f,raw))
    if mutate:
        for f,raw in staged: create(f,raw)
    return {'Missing':[f['Target'] for f,raw in staged], 'Verified':len(files)-len(staged)}

def web_inspect(files):
    result=web_files(files,False)
    files_state='absent' if result['Verified']==0 else 'partial' if result['Missing'] else sha(json.dumps([(f['Target'],f['SHA256']) for f in files]).encode())
    setting,key=web_setup(False)
    web_run(['/usr/sbin/httpd','-t','-f','/etc/httpd/conf/almadial-s05.conf'])
    web_run(['/usr/sbin/php-fpm','-t'])
    for name in ('httpd','php-fpm'):
        if web_run(['/usr/bin/systemctl','is-active',name])!='active' or web_run(['/usr/bin/systemctl','is-enabled',name])!='enabled':
            raise ValueError('S07 web service must be enabled and active')
    status,header,text=web_request('/agc/vicidial.php')
    if status!=200 or 'phone_login' not in text: raise ValueError('S07 agent form unavailable')
    services=sha(b'S07 active local vhost and PHP-FPM') if header=='S07' and not result['Missing'] else 'absent'
    health='absent'
    if setting=='N':
        status,_,text=web_request('/vicidial/admin.php',key)
        if not web_admin_ready(status,text):
            raise ValueError('S07 authenticated administration unavailable')
        health=sha(b'S07 secured administrator and agent form')
    return dict(Foreign=web_foreign(),Files=files_state,Services=services,Health=health)

def web_main():
    try:
        payload=json.load(sys.stdin)
        files=payload['Files']
        if len(sys.argv)!=2 or len(files)!=2: raise ValueError('S07 operation')
        # Validate exact paths, content digests and metadata before any effect.
        for f in files:
            if f['Target'] not in S07_PATHS or (f['Owner'],f['Group'],f['Mode'],f['Label'])!=('root','root',0o644,'httpd_config_t'):
                raise ValueError('S07 file contract')
        mode=sys.argv[1]
        if mode=='web-files':
            # Existing owned files must match, never overwrite foreign content.
            web_files(files,True)
        elif mode=='web-services':
            if web_files(files,False)['Missing']: raise ValueError('S07 configuration incomplete')
            web_run(['/usr/sbin/httpd','-t','-f','/etc/httpd/conf/almadial-s05.conf'])
            web_run(['/usr/sbin/php-fpm','-t'])
            if web_request('/agc/vicidial.php')[1]!='S07':
                web_run(['/usr/bin/systemctl','reload','httpd'])
        elif mode=='web-verify': web_setup(True)
        elif mode!='web-inspect': raise ValueError('S07 operation')
        if mode=='web-inspect': print(json.dumps(web_inspect(files)))
        else: print('{}')
        return 0
    except Exception:
        sys.stderr.write('S07 web operation failed; inspect protected configuration and application diagnostics\n')
        return 1
if __name__=='__main__': sys.exit(web_main())
