# Appended to the unchanged accepted S05 file primitives with its main removed.
# Closed S06 policy; no caller-supplied paths, commands or activation switches.
import base64
import subprocess
import re

S06_SCRIPTS = ('FastAGI_log.pl', 'AST_update.pl', 'AST_manager_listen.pl', 'ready.pl')
S06_UNITS = ('fastagi', 'updater', 'listener')
S06_PATHS = ('/etc/cron.d/almadial-s06', '/etc/astguiclient-s06.conf') + tuple(
    '/usr/share/astguiclient/almadial-s06-' + n for n in S06_SCRIPTS) + tuple(
    '/etc/systemd/system/almadial-s06-' + n + '.service' for n in S06_UNITS)
OWNED_PATHS['runtime'] = S06_PATHS
old_set_label = set_label


def set_label(fd, expected):
    if expected == 'system_cron_spool_t':
        os.setxattr(fd, 'security.selinux', b'system_u:object_r:system_cron_spool_t:s0')
        if label(fd) != expected:
            raise ValueError('S06 cron label failed')
    else:
        old_set_label(fd, expected)


def runtime_owned(files, mutate):
    if sorted(f['Target'] for f in files) != sorted(S06_PATHS):
        raise ValueError('S06 file set conflict')
    staged = []
    for f in files:
        path = f['Target']
        metadata = ('root', 'root', 'usr_t', 0o644)
        if path == '/etc/astguiclient-s06.conf':
            metadata = ('root', 'asterisk', 'etc_t', 0o640)
        elif path == '/etc/cron.d/almadial-s06':
            metadata = ('root', 'root', 'system_cron_spool_t', 0o644)
        elif path.startswith('/etc/systemd/system/'):
            metadata = ('root', 'root', 'systemd_unit_file_t', 0o644)
        raw = base64.b64decode(f['Content'], validate=True)
        if len(raw) > 1 << 20 or sha(raw) != f['SHA256'] or (f['Owner'], f['Group'], f['Label'], f['Mode']) != metadata:
            raise ValueError('S06 metadata conflict')
        staged.append(raw)
    missing = []
    corrections = []
    for f, raw in zip(files, staged):
        o = observation(f)
        if o.get('SHA256') == f.get('PriorSHA256') and f['Target'] in ('/etc/astguiclient-s06.conf', '/usr/share/astguiclient/almadial-s06-ready.pl'):
            old = dict(f, SHA256=f['PriorSHA256'])
            verify_existing(old, o)
            corrections.append((f, raw, old))
        elif not verify_existing(f, o):
            missing.append((f, raw))
    if mutate:
        for f, raw in missing:
            create(f, raw)
        for f, raw, old in corrections:
            correct_local_socket(f, raw, old)
    return len(missing) + len(corrections)


def correct_local_socket(f, raw, old):
    # Fixed two-file correction, under the S06 lock and durable file intent.
    # Preserve a root-only exact preimage before atomic publication.
    parent = open_directory(str(Path(f['Target']).parent))
    tmp = str(Path(f['Target']).parent) + '/.almadial-s06-new-' + secrets.token_hex(16)
    backup = f['Target'] + '.alm21-before-local-socket'
    try:
        verify_existing(old, observation(old))
        fd = os.open(Path(f['Target']).name, os.O_RDONLY | os.O_NOFOLLOW, dir_fd=parent)
        try:
            before = content(fd)
        finally:
            os.close(fd)
        b = dict(old, Target=backup, Mode=0o600)
        # create() publishes only absent paths; prior recovery backups must match.
        try:
            create(b, before)
        except FileExistsError:
            fd = os.open(Path(backup).name, os.O_RDONLY | os.O_NOFOLLOW, dir_fd=parent)
            try:
                st = os.fstat(fd)
                if sha(content(fd)) != old['SHA256'] or stat.S_IMODE(st.st_mode) != 0o600 or st.st_uid != 0:
                    raise ValueError('S06 correction backup conflict')
            finally:
                os.close(fd)
        create(dict(f, Target=tmp), raw)
        verify_existing(old, observation(old))
        os.replace(Path(tmp).name, Path(f['Target']).name, src_dir_fd=parent, dst_dir_fd=parent)
        sync(parent)
    finally:
        try:
            os.unlink(Path(tmp).name, dir_fd=parent)
        except FileNotFoundError:
            pass
        os.close(parent)


def run(argv):
    p = subprocess.run(argv, capture_output=True, timeout=20)
    if p.returncode or len(p.stdout) > 1 << 20 or len(p.stderr) > 1 << 20:
        raise ValueError('S06 bounded command failed: ' + argv[0])
    return p.stdout.decode().strip()


def unit(name):
    raw = run(['/usr/bin/systemctl', 'show', name, '--property=ActiveState,SubState,UnitFileState,MainPID,ControlGroup,FragmentPath,DropInPaths,User,Group,NeedDaemonReload'])
    return dict(line.split('=', 1) for line in raw.splitlines() if '=' in line)


def runtime_schedules():
    # Inspect every cron surface, without rewriting any foreign file. Snapshot
    # hashes include unrelated schedules so unexpected changes cannot be hidden.
    paths = ['/etc/crontab']
    for directory in ('/etc/cron.d', '/var/spool/cron', '/etc/cron.hourly', '/etc/cron.daily', '/etc/cron.weekly', '/etc/cron.monthly'):
        try:
            parent = open_directory(directory)
        except FileNotFoundError:
            continue
        try:
            paths.extend(directory + '/' + n for n in os.listdir(parent))
        finally:
            os.close(parent)
    foreign = []
    for path in sorted(paths):
        try:
            parent = open_directory(str(Path(path).parent))
        except FileNotFoundError:
            continue
        try:
            try:
                fd = os.open(Path(path).name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
            except FileNotFoundError:
                continue
            try:
                raw = content(fd)
                for line in raw.splitlines():
                    if line.strip() and not line.lstrip().startswith(b'#') and any(m in line.lower() for m in (b'astguiclient', b'fastagi', b'ast_', b'admin_keepalive')):
                        raise ValueError('S06 competing or unauthorized schedule')
                if path != '/etc/cron.d/almadial-s06':
                    s = os.fstat(fd)
                    foreign.append([path, sha(raw), s.st_uid, s.st_gid, stat.S_IMODE(s.st_mode), label(fd)])
            finally:
                os.close(fd)
        finally:
            os.close(parent)
    return sha(json.dumps(foreign, sort_keys=True).encode())


def runtime_processes(service):
    rows = []
    expected = [b'/usr/bin/perl', b'/usr/share/astguiclient/almadial-s06-FastAGI_log.pl']
    for name in os.listdir('/proc'):
        if not name.isdecimal() or int(name) == os.getpid():
            continue
        try:
            raw = Path('/proc/' + name + '/cmdline').read_bytes()
            argv = raw.rstrip(b'\0').split(b'\0')
            # Exclude Python adapter/qualification source text; Perl workers
            # and direct executable scripts remain fully inspected.
            if not argv or not (b'perl' in argv[0].lower() or any(a.endswith(b'.pl') for a in argv)):
                continue
            if not any(m in raw.lower() for m in (b'astguiclient', b'fastagi', b'ast_', b'admin_keepalive')):
                continue
            # Net::Server keeps the original argv in this pinned runtime.
            if argv != expected:
                raise ValueError('S06 unapproved application process')
            status = Path('/proc/' + name + '/status').read_text()
            values = dict(l.split(':', 1) for l in status.splitlines() if ':' in l)
            if set(values['Uid'].split()) != {str(uid('asterisk'))} or set(values['Gid'].split()) != {str(gid('asterisk'))}:
                raise ValueError('S06 process identity conflict')
            cg = Path('/proc/' + name + '/cgroup').read_text().strip()
            if cg != '0::/system.slice/almadial-s06-fastagi.service':
                raise ValueError('S06 competing FastAGI supervisor')
            rows.append((int(name), int(values['PPid'])))
        except (FileNotFoundError, ProcessLookupError):
            continue
    # r4037 enters package VDfastAGI before run(); its main-package pool
    # variables are not passed across that boundary. The qualified native
    # idle pool is Net::Server::PreFork default: one parent, five children.
    if rows:
        main = int(service.get('MainPID', '0'))
        if len(rows) > 6 or sum(pid == main for pid, _ in rows) != 1 or any(pid != main and parent != main for pid, parent in rows):
            raise ValueError('S06 prefork ownership/count failed')
    return rows


def runtime_state(files, tls_handoff=False, remote_tls_handoff=False):
    if run(['/usr/sbin/getenforce']) != 'Enforcing':
        raise ValueError('S06 SELinux must remain Enforcing')
    foreign = runtime_schedules()
    missing = runtime_owned(files, False)
    fast = unit('almadial-s06-fastagi.service')
    for n in ('updater', 'listener'):
        u = unit('almadial-s06-' + n + '.service')
        if u.get('ActiveState') not in ('inactive', 'failed') or u.get('UnitFileState') not in ('disabled', '') or u.get('DropInPaths'):
            raise ValueError('S06 deferred unit activated or overridden')
    rows = runtime_processes(fast)
    if fast.get('DropInPaths'):
        raise ValueError('S06 FastAGI override conflicts')
    services_ok = not missing
    for name in ('mariadb', 'asterisk', 'httpd', 'php-fpm', 'crond'):
        u = unit(name + '.service')
        if u.get('ActiveState') != 'active':
            raise ValueError('S06 foundation service unavailable: ' + name)
        services_ok = services_ok and u.get('UnitFileState') == 'enabled'
    services_ok = services_ok and fast.get('ActiveState') == 'active' and fast.get('UnitFileState') == 'enabled' and fast.get('NeedDaemonReload') == 'no'
    listeners = run(['/usr/sbin/ss', '-H', '-lntup'])
    local_fastagi = False
    for line in listeners.splitlines():
        parts = line.split()
        if len(parts) < 6:
            raise ValueError('S06 listener format conflict')
        local = parts[4]
        port = local.rsplit(':', 1)[-1]
        if parts[0] == 'udp' and local in ('127.0.0.1:323', '[::1]:323') and '"chronyd"' in line:
            continue
        if port == '22':
            continue
        # The unchanged S04 verifier separately constrains Asterisk's two
        # unconnected ephemeral PJPROJECT resolver sockets (not SIP transports).
        if parts[0] == 'udp' and '"asterisk"' in line:
            continue
        if tls_handoff and ((local==('0.0.0.0:443' if remote_tls_handoff else '127.0.0.1:443') and '"httpd"' in line) or (local=='127.0.0.1:8089' and '"asterisk"' in line)):
            continue
        if local not in ('127.0.0.1:3306', '127.0.0.1:5038', '127.0.0.1:8088', '127.0.0.1:80', '127.0.0.1:4577'):
            raise ValueError('S06 unauthorized application listener')
        if port == '4577':
            socket_pids = {int(p) for p in re.findall(r'pid=(\d+)', line)}
            if '"perl"' not in line or not rows or not socket_pids or not socket_pids.issubset({p for p, _ in rows}):
                raise ValueError('S06 FastAGI socket ownership failed')
            local_fastagi = True
    healthy = services_ok and len(rows) == 6 and local_fastagi
    if healthy:
        run(['/usr/sbin/runuser', '-u', 'asterisk', '--', '/usr/bin/perl', '/usr/share/astguiclient/almadial-s06-ready.pl'])
    return {'Foreign': foreign, 'Files': 'absent' if missing == len(files) else 'partial' if missing else sha(json.dumps([{k: v for k, v in f.items() if k != 'Content'} for f in files], sort_keys=True).encode()),
            'Services': sha(b'S06/enabled-existing-foundation-and-fastagi/deferred-disabled/v1') if services_ok else 'absent',
            'Health': sha(b'S06/ready-database/local-fastagi/owned-prefork/no-dialing/v1') if healthy else 'absent'}


def runtime_main():
    try:
        p = json.load(sys.stdin)
        mode = sys.argv[1:]
        if mode == ['runtime-inspect']:
            print(json.dumps(runtime_state(p['Files'], p.get('TLSHandoff',False), p.get('RemoteTLSHandoff',False))))
        elif mode == ['runtime-files']:
            runtime_state(p['Files'])  # Conflict check before first publication.
            runtime_owned(p['Files'], True)
            print('{}')
        elif mode == ['runtime-services']:
            state = runtime_state(p['Files'])
            if state['Files'] in ('absent', 'partial'):
                raise ValueError('S06 files incomplete')
            if state['Services'] == 'absent':
                if any(unit('almadial-s06-' + n + '.service').get('NeedDaemonReload') == 'yes' or not unit('almadial-s06-' + n + '.service').get('FragmentPath') for n in S06_UNITS):
                    run(['/usr/bin/systemctl', 'daemon-reload'])
                for name in ('httpd', 'php-fpm', 'almadial-s06-fastagi'):
                    if unit(name + '.service').get('UnitFileState') != 'enabled':
                        run(['/usr/bin/systemctl', 'enable', name + '.service'])
                if unit('almadial-s06-fastagi.service').get('ActiveState') != 'active':
                    run(['/usr/bin/systemctl', 'start', 'almadial-s06-fastagi.service'])
            print('{}')
        else:
            raise ValueError('invalid S06 operation')
        return 0
    except Exception as e:
        # Exception strings here contain fixed policy identifiers, never raw
        # subprocess output, source bytes, database errors or credentials.
        print('S06 native verification failed: ' + str(e) if isinstance(e, ValueError) else 'S06 native inspection failed', file=sys.stderr)
        return 1


sys.exit(runtime_main())
