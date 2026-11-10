"""S05 explicit-file adapter. No subprocesses, recursive copies or removals.

Only the Go journaled S05 runner may invoke installation. Input is the reviewed
source/destination manifest, never a shell command. Source bytes are read from
the private, digest-checked pinned ZIP; every destination is checked before the
first write. Existing files must match exactly and are never replaced.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import grp
import secrets
import stat
import sys
import zipfile

ROOT = Path('/')
ROOT_UID = 0
ARCHIVE = '/var/lib/almadial/vicidial-s05/source.zip'
ARCHIVE_SHA = 'ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058'
AGI_MODULES = ('/usr/local/share/perl5/5.40/Asterisk.pm',
               '/usr/local/share/perl5/5.40/Asterisk/AGI.pm')
ROOTS = ('/var/www/html', '/usr/share/astguiclient',
         '/var/lib/asterisk/agi-bin', '/var/lib/asterisk/sounds',
         '/usr/share/doc/almadial-vicidial')
SERVICE_PARENTS = ('/var/lib/asterisk', '/var/lib/asterisk/agi-bin',
                   '/var/lib/asterisk/sounds')
OWNED_PATHS = {
    'configuration': (
        '/etc/astguiclient.conf', '/etc/astguiclient-asterisk.conf',
        '/etc/asterisk/almadial-s05-manager.conf',
        '/var/www/html/agc/options.php', '/var/www/html/vicidial/options.php',
        '/var/www/html/vicidial/old_clear'),
    'perl': ('/usr/local/share/perl5/5.40/Net/Telnet.pm',
             '/usr/share/doc/almadial-net-telnet/README'),
    'web': ('/etc/httpd/conf/almadial-s05.conf',
            '/etc/systemd/system/httpd.service.d/almadial-s05.conf'),
}
OWNED_DIRECTORIES = ('/usr/local/share/perl5', '/usr/local/share/perl5/5.40',
                     '/usr/local/share/perl5/5.40/Net',
                     '/usr/local/share/perl5/5.40/Asterisk',
                     '/usr/share/doc/almadial-net-telnet',
                     '/etc/systemd/system/httpd.service.d')


def sha(data):
    return hashlib.sha256(data).hexdigest()


def uid(name):
    return pwd.getpwnam(name).pw_uid


def gid(name):
    return grp.getgrnam(name).gr_gid


def label(fd):
    return os.getxattr(fd, 'security.selinux').decode().strip('\0').split(':')[2]


def set_label(fd, expected):
    # Label before publication, so interruption cannot leave a final inode
    # whose bytes match but whose label makes safe repeat execution fail.
    if expected not in ('httpd_sys_content_t', 'usr_t', 'bin_t', 'var_lib_t',
                        'etc_t', 'httpd_config_t', 'systemd_unit_file_t'):
        raise ValueError('unapproved S05 file label')
    os.setxattr(fd, 'security.selinux',
                ('system_u:object_r:' + expected + ':s0').encode())
    if label(fd) != expected:
        raise ValueError('S05 file label postcondition failed')


def sync(fd):
    os.fsync(fd)


def directory_ok(fd, logical):
    s = os.fstat(fd)
    service = (logical in SERVICE_PARENTS and s.st_uid == uid('asterisk')
               and s.st_gid == gid('asterisk') and stat.S_IMODE(s.st_mode) == 0o750)
    if not stat.S_ISDIR(s.st_mode) or (s.st_uid != ROOT_UID and not service) or s.st_mode & 0o022:
        raise ValueError('unsafe S05 parent directory')


def target_ok(target):
    p = Path(target)
    return (p.is_absolute() and str(p) == target and '..' not in p.parts
            and not any(c in target for c in ('\\', '\0', '\r', '\n', ':'))
            and (any(target.startswith(root + '/') for root in ROOTS)
                 or target in AGI_MODULES
                 or any(target in paths for paths in OWNED_PATHS.values())))


def dir_metadata(logical):
    if logical.startswith('/var/www/html/'):
        return gid('apache'), 0o750
    if logical.startswith('/var/lib/asterisk/'):
        return gid('asterisk'), 0o750
    return gid('root'), 0o755


def open_directory(logical, create=False):
    """Walk each component through a held descriptor, refusing every symlink."""
    fd = os.open(ROOT, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        directory_ok(fd, '/')
        current = ''
        for part in Path(logical).parts[1:]:
            current += '/' + part
            try:
                child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            except FileNotFoundError:
                if not create:
                    raise
                # Creating vendor/system ancestors is never allowed.
                if current not in OWNED_DIRECTORIES and not any(current == root or current.startswith(root + '/') for root in ROOTS):
                    raise ValueError('missing prerequisite directory')
                group, mode = dir_metadata(current)
                os.mkdir(part, mode=0o700, dir_fd=fd)
                child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
                os.fchown(child, ROOT_UID, group)
                os.fchmod(child, mode)
                sync(child)
                sync(fd)
            try:
                directory_ok(child, current)
            except BaseException:
                os.close(child)
                raise
            os.close(fd)
            fd = child
        return fd
    except BaseException:
        os.close(fd)
        raise


def content(fd):
    before = os.fstat(fd)
    if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or before.st_size > 256 << 20:
        raise ValueError('unsafe S05 file')
    with os.fdopen(os.dup(fd), 'rb') as stream:
        data = stream.read((256 << 20) + 1)
    after = os.fstat(fd)
    if (before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns) != (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns) or len(data) != before.st_size:
        raise ValueError('S05 file changed during inspection')
    return data


def observation(f):
    target = f['Target']
    if not target_ok(target):
        raise ValueError('unapproved S05 destination')
    try:
        parent = open_directory(str(Path(target).parent))
    except FileNotFoundError:
        return {'Target': target, 'Kind': 'absent'}
    try:
        try:
            fd = os.open(Path(target).name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        except FileNotFoundError:
            return {'Target': target, 'Kind': 'absent'}
        try:
            data = content(fd)
            s = os.fstat(fd)
            if s.st_uid != uid(f['Owner']) or s.st_gid != gid(f['Group']):
                raise ValueError('S05 file ownership conflict')
            return {'Target': target, 'Kind': 'regular', 'SHA256': sha(data),
                    'Owner': f['Owner'], 'Group': f['Group'], 'Mode': stat.S_IMODE(s.st_mode),
                    'Label': label(fd), 'Links': s.st_nlink}
        finally:
            os.close(fd)
    finally:
        os.close(parent)


def verify_existing(f, o):
    if o['Kind'] == 'absent':
        return False
    if (o['SHA256'], o['Owner'], o['Group'], o['Mode'], o['Label'], o['Links']) != (
            f['SHA256'], f['Owner'], f['Group'], f['Mode'], f['Label'], 1):
        raise ValueError('S05 file content or metadata conflict; preserve existing file')
    return True


def create(f, data):
    parent = open_directory(str(Path(f['Target']).parent), create=True)
    tmp = '.almadial-s05-' + secrets.token_hex(16)
    fd = None
    try:
        fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=parent)
        with os.fdopen(os.dup(fd), 'wb') as out:
            out.write(data)
            out.flush()
        os.fchown(fd, uid(f['Owner']), gid(f['Group']))
        os.fchmod(fd, f['Mode'])
        set_label(fd, f['Label'])
        sync(fd)
        # link is an atomic create-if-absent publication. A concurrent writer
        # wins with EEXIST; replacement/rename-overwrite is never used.
        os.link(tmp, Path(f['Target']).name, src_dir_fd=parent, dst_dir_fd=parent,
                follow_symlinks=False)
        os.unlink(tmp, dir_fd=parent)
        sync(parent)
    finally:
        if fd is not None:
            os.close(fd)
        try:
            os.unlink(tmp, dir_fd=parent)
        except FileNotFoundError:
            pass
        os.close(parent)


def derive(name, data):
    if name in ('bin/ADMIN_keepalive_ALL.pl', 'bin/AST_manager_send.pl',
                'bin/AST_VDauto_dial.pl', 'bin/AST_VDremote_agents.pl'):
        old = b'/bin/ps -o "%p %a" --no-headers -A'
        expected = 3 if name == 'bin/ADMIN_keepalive_ALL.pl' else 1
        if data.count(old) != expected:
            raise ValueError('procps source contract changed')
        return data.replace(old, b'/bin/ps -o pid,args -ww --no-headers -A')
    return data


def install(files):
    observed = [observation(f) for f in files]
    missing = [f for f, o in zip(files, observed) if not verify_existing(f, o)]
    parent = open_directory(str(Path(ARCHIVE).parent))
    try:
        fd = os.open(Path(ARCHIVE).name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        try:
            data = content(fd)
            s = os.fstat(fd)
            if s.st_uid != ROOT_UID or s.st_gid != gid('root') or stat.S_IMODE(s.st_mode) != 0o600 or sha(data) != ARCHIVE_SHA:
                raise ValueError('S05 source archive drift')
        finally:
            os.close(fd)
    finally:
        os.close(parent)
    import io
    with zipfile.ZipFile(io.BytesIO(data)) as source:
        staged = []
        # Verify all requested source bytes and derivations before any mutation.
        for f in missing:
            raw = source.read('2026-11-05/' + f['Source'])
            if sha(raw) != f['SourceSHA256']:
                raise ValueError('S05 source inventory conflict')
            result = derive(f['Source'], raw)
            if sha(result) != f['SHA256']:
                raise ValueError('S05 derived content mismatch')
            staged.append((f, result))
        for f, raw in staged:
            create(f, raw)
    return {'Installed': len(missing), 'Preserved': len(files) - len(missing)}


def owned(files, component, mutate):
    # Closed component lists prevent this endpoint becoming an arbitrary
    # filesystem writer. The Go caller derives all bytes from verified inputs;
    # credentials travel only through transient stdin, never argv or output.
    import base64
    if component not in OWNED_PATHS or sorted(f['Target'] for f in files) != sorted(OWNED_PATHS[component]):
        raise ValueError('unapproved S05 owned component')
    staged = []
    for f in files:
        raw = base64.b64decode(f['Content'], validate=True)
        if component == 'perl':
            metadata = ('root', 'root', 'usr_t', 0o644)
        elif component == 'web':
            context = 'httpd_config_t' if f['Target'].startswith('/etc/httpd/') else 'systemd_unit_file_t'
            metadata = ('root', 'root', context, 0o644)
        else:
            group = 'asterisk' if f['Target'] in ('/etc/astguiclient-asterisk.conf', '/etc/asterisk/almadial-s05-manager.conf') else 'apache'
            context = 'httpd_sys_content_t' if f['Target'].startswith('/var/www/') else 'etc_t'
            metadata = ('root', group, context, 0o640)
        if len(raw) > 1 << 20 or sha(raw) != f['SHA256'] or (f['Owner'], f['Group'], f['Label'], f['Mode']) != metadata:
            raise ValueError('invalid S05 owned file')
        staged.append(raw)
    observed = [observation(f) for f in files]
    missing = [(f, raw) for f, o, raw in zip(files, observed, staged) if not verify_existing(f, o)]
    if mutate:
        for f, raw in missing:
            create(f, raw)
    return {'Missing': [f['Target'] for f, _ in missing],
            'Verified': len(files) - len(missing)}


# Closed S05 generation publication. The upstream generator writes only into
# this private directory; its legacy outputs are never loaded by Asterisk.
WORK = '/var/lib/almadial/vicidial-s05'
MODERN = ('extensions-vicidial.conf', 'pjsip-vicidial.conf',
          'pjsip_wizard-vicidial.conf', 'confbridge-vicidial.conf')
WRAPPERS = {
    'manager.conf': (b'#include almadial-s04-manager.conf\n', b'#include "almadial-s05-manager.conf"\n'),
    'extensions.conf': (b'#include almadial-s04-extensions.conf\n', b'[globals]\n#include "extensions-vicidial.conf"\n'),
    'pjsip.conf': (b'#include almadial-s04-pjsip.conf\n', b'#include "pjsip-vicidial.conf"\n'),
    'confbridge.conf': (b'#include almadial-s04-confbridge.conf\n', b'#include "confbridge-vicidial.conf"\n'),
    'pjsip_wizard.conf': (b'', b'#include "pjsip_wizard-vicidial.conf"\n'),
}


def private_directory(path, create=False):
    # The accepted S03 parent already exists. All new work directories are
    # private and no intermediate symlink can redirect a protected write.
    parent = open_directory('/var/lib/almadial')
    try:
        for part in Path(path).relative_to('/var/lib/almadial').parts:
            if create:
                try:
                    os.mkdir(part, 0o700, dir_fd=parent)
                    sync(parent)
                except FileExistsError:
                    pass
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
            s = os.fstat(child)
            if (s.st_uid, s.st_gid, stat.S_IMODE(s.st_mode)) != (ROOT_UID, gid('root'), 0o700):
                os.close(child)
                raise ValueError('unsafe private S05 directory')
            os.close(parent)
            parent = child
        return parent
    except BaseException:
        os.close(parent)
        raise


def private_read(name):
    if '/' in name or name in ('', '.', '..'):
        raise ValueError('invalid private S05 name')
    parent = private_directory(WORK)
    try:
        return read_at(parent, name, 'root', 0o600, None)
    finally:
        os.close(parent)


def read_at(parent, name, group, mode, context):
    try:
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
    except FileNotFoundError:
        return None
    try:
        s = os.fstat(fd)
        if (s.st_uid, s.st_gid, stat.S_IMODE(s.st_mode)) != (ROOT_UID, gid(group), mode):
            raise ValueError('S05 protected file metadata conflict')
        if context is not None and label(fd) != context:
            raise ValueError('S05 protected file label conflict')
        return content(fd)
    finally:
        os.close(fd)


def publish_at(parent, name, raw, group='root', mode=0o600, context=None, predecessor=None):
    # predecessor=None always means create-if-absent, never blind replacement.
    tmp = '.almadial-s05-' + secrets.token_hex(16)
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=parent)
    try:
        with os.fdopen(os.dup(fd), 'wb') as out:
            out.write(raw)
            out.flush()
        os.fchown(fd, ROOT_UID, gid(group))
        os.fchmod(fd, mode)
        if context:
            set_label(fd, context)
        sync(fd)
        if predecessor is None:
            os.link(tmp, name, src_dir_fd=parent, dst_dir_fd=parent, follow_symlinks=False)
        else:
            if read_at(parent, name, group, mode, context) != predecessor:
                raise ValueError('S05 wrapper changed before replacement')
            os.replace(tmp, name, src_dir_fd=parent, dst_dir_fd=parent)
        sync(parent)
    finally:
        os.close(fd)
        try:
            os.unlink(tmp, dir_fd=parent)
        except FileNotFoundError:
            pass


def private_create(name, raw):
    parent = private_directory(WORK, True)
    try:
        existing = read_at(parent, name, 'root', 0o600, None)
        if existing is not None:
            if existing != raw:
                raise ValueError('S05 protected preimage conflict')
            return
        publish_at(parent, name, raw)
    finally:
        os.close(parent)


def generation_files():
    generated = private_directory(WORK + '/generated')
    try:
        files = {}
        for name in MODERN:
            raw = read_at(generated, name, 'root', 0o600, None)
            if raw is None or not raw or len(raw) > 8 << 20:
                raise ValueError('missing S05 generated output')
            # Configuration includes/exec and a SIP transport cannot be added
            # by an unreviewed database template. No network activation in S05.
            text = raw.decode('utf-8')
            if any(line.lstrip().startswith('#') for line in text.splitlines()):
                raise ValueError('unapproved generated configuration directive')
            import re
            if name.startswith('pjsip') and any(re.match(r'^\s*type\s*=\s*transport\s*(;.*)?$', line, re.I) for line in text.splitlines()):
                raise ValueError('unapproved generated SIP transport')
            files[name] = raw
        return files
    finally:
        os.close(generated)


def generation_prepare(p):
    import base64
    program = base64.b64decode(p['Program'], validate=True)
    preimage = base64.b64decode(p['Preimage'], validate=True)
    if sha(program) != 'a830cc4f92febd2ddd044fb04552e096a434b261f0209db7ba33842b3a11e14f' or len(preimage) > 3 << 20:
        raise ValueError('S05 generator derivation conflict')
    private_create('generator-inputs-before.json', preimage)
    private_create('generator.pl', program)
    generated = private_directory(WORK + '/generated', True)
    try:
        if os.listdir(generated):
            raise ValueError('uncertain previous generator execution')
        publish_at(generated, 'voicemail.conf', b'[general]\n[default]\n')
    finally:
        os.close(generated)
    tools = private_directory(WORK + '/tools', True)
    try:
        stub = b'#!/bin/sh\n/usr/bin/printf "%s\\n" "$*" >> /var/lib/almadial/vicidial-s05/suppressed-reloads.log\n'
        existing = read_at(tools, 'almadial-s05-no-reload', 'root', 0o700, None)
        if existing is None:
            publish_at(tools, 'almadial-s05-no-reload', stub, mode=0o700)
        elif existing != stub:
            raise ValueError('S05 reload stub conflict')
    finally:
        os.close(tools)
    return {'Prepared': True}


def generated_identity():
    return {name: sha(raw) for name, raw in generation_files().items()}


def generation_state(p, mutate=False):
    binding, inputs = p['Binding'], p['Inputs']
    if any(not isinstance(x, str) or len(x) != 64 or any(c not in '0123456789abcdef' for c in x) for x in (binding, inputs)):
        raise ValueError('invalid S05 generation identity')
    parent = open_directory('/etc/asterisk')
    try:
        wrapper_bytes = {name: read_at(parent, name, 'asterisk', 0o640, 'etc_t') for name in WRAPPERS}
        published = {name: read_at(parent, name, 'asterisk', 0o640, 'etc_t') for name in MODERN}
        try:
            sealed = private_read('generation.json')
        except FileNotFoundError:
            sealed = None
        if sealed is not None:
            receipt = json.loads(sealed)
            files = generation_files()
            expected = {'Binding': binding, 'Inputs': inputs, 'Files': {name: sha(raw) for name, raw in files.items()}}
            if receipt != expected:
                raise ValueError('S05 generator receipt or input drift')
            if any(wrapper_bytes[name] != before + extra for name, (before, extra) in WRAPPERS.items()) or published != files:
                raise ValueError('S05 generated configuration foreign edit')
            return {'State': sha(sealed), 'Wrapped': True}
        before_ok = all(wrapper_bytes[name] == (before or None) for name, (before, _) in WRAPPERS.items())
        if not mutate:
            if not before_ok or any(raw is not None for raw in published.values()):
                raise ValueError('unsealed S05 publication requires reconciliation')
            return {'State': 'absent', 'Wrapped': False}
        files = generation_files()
        receipt = {'Binding': binding, 'Inputs': inputs, 'Files': {name: sha(raw) for name, raw in files.items()}}
        intent = json.dumps(receipt, sort_keys=True).encode()
        # Before the first replacement retain both the exact declared preimages
        # and publication intent. An interrupted publication fails closed until
        # explicit reconciliation; its executable never implies completion.
        for name, (before, extra) in WRAPPERS.items():
            if wrapper_bytes[name] != (before or None):
                raise ValueError('S05 wrapper preimage conflict')
        if any(raw is not None for raw in published.values()):
            raise ValueError('S05 generated destination conflict')
        private_create('publication-intent.json', intent)
        for name, (before, _) in WRAPPERS.items():
            private_create('before-' + name, before)
        for name, raw in files.items():
            publish_at(parent, name, raw, 'asterisk', 0o640, 'etc_t')
        for name, (before, extra) in WRAPPERS.items():
            publish_at(parent, name, before + extra, 'asterisk', 0o640, 'etc_t', before or None)
        private_create('generation.json', intent)
        return {'State': sha(intent), 'Wrapped': True}
    finally:
        os.close(parent)


def runtime_receipt(p, mutate=False):
    expected = p['Receipt']
    if not isinstance(expected, dict) or set(expected) != {'Binding', 'Components'}:
        raise ValueError('invalid S05 runtime identity')
    components = expected['Components']
    if not isinstance(components, dict) or set(components) != {'Files', 'Perl', 'Configuration', 'Registration', 'Generated', 'Web', 'Inputs'}:
        raise ValueError('incomplete S05 runtime identity')
    for value in [expected['Binding'], *components.values()]:
        if not isinstance(value, str) or len(value) != 64 or any(c not in '0123456789abcdef' for c in value):
            raise ValueError('invalid S05 component identity')
    raw = json.dumps(expected, sort_keys=True).encode()
    try:
        existing = private_read('runtime.json')
    except FileNotFoundError:
        existing = None
    if existing is not None and existing != raw:
        raise ValueError('S05 runtime receipt drift')
    if mutate:
        private_create('runtime.json', raw)
        existing = private_read('runtime.json')
    return {'State': sha(existing) if existing is not None else 'absent'}


def deferred_inspect():
    markers = (b'astguiclient', b'AST_', b'ADMIN_keepalive_ALL', b'FastAGI')

    def contains_job(raw):
        return any(marker in raw for marker in markers)

    # Inspect actual process arguments, including jobs started outside systemd.
    proc = open_directory('/proc')
    try:
        for name in os.listdir(proc):
            if not name.isdecimal() or int(name) == os.getpid():
                continue  # This fixed adapter's own argv contains its source.
            try:
                process = os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=proc)
            except FileNotFoundError:
                continue
            try:
                try:
                    fd = os.open('cmdline', os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=process)
                except FileNotFoundError:
                    continue  # The process exited during inspection.
                try:
                    raw = os.read(fd, 65537)
                    if len(raw) > 65536 or contains_job(raw):
                        raise ValueError('S05 deferred application process is active')
                finally:
                    os.close(fd)
            finally:
                os.close(process)
    finally:
        os.close(proc)
    paths = ['/etc/crontab']
    for directory in ('/etc/cron.d', '/var/spool/cron', '/etc/cron.hourly',
                      '/etc/cron.daily', '/etc/cron.weekly', '/etc/cron.monthly'):
        try:
            parent = open_directory(directory)
        except FileNotFoundError:
            continue
        try:
            paths.extend(directory + '/' + name for name in os.listdir(parent))
        finally:
            os.close(parent)
    for path in paths:
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
                # Comments and unrelated maintenance entries are preserved.
                if any(contains_job(line) for line in raw.splitlines()
                       if line.strip() and not line.lstrip().startswith(b'#')):
                    raise ValueError('S05 deferred application schedule is active')
            finally:
                os.close(fd)
        finally:
            os.close(parent)
    return {'Inactive': True}


def main():
    try:
        p = json.load(sys.stdin)
        if sys.argv[1:] == ['deferred-inspect']:
            print(json.dumps(deferred_inspect()))
            return 0
        if sys.argv[1:] in (['runtime-inspect'], ['runtime-seal']):
            print(json.dumps(runtime_receipt(p, sys.argv[1] == 'runtime-seal')))
            return 0
        if sys.argv[1:] == ['generation-prepare']:
            print(json.dumps(generation_prepare(p)))
            return 0
        if sys.argv[1:] == ['generated-identity']:
            print(json.dumps(generated_identity()))
            return 0
        if sys.argv[1:] == ['registration-preimage']:
            import base64
            raw = base64.b64decode(p['Preimage'], validate=True)
            if len(raw) > 65536:
                raise ValueError('oversized S05 registration preimage')
            private_create('registration-before.json', raw)
            print(json.dumps({'Preserved': sha(raw)}))
            return 0
        if sys.argv[1:] in (['generation-inspect'], ['generation-publish']):
            print(json.dumps(generation_state(p, sys.argv[1] == 'generation-publish')))
            return 0
        if sys.argv[1:] == ['ami-create']:
            secret = p['AMI']
            if not isinstance(secret, str) or len(secret) != 20 or any(c not in '0123456789abcdef' for c in secret):
                raise ValueError('invalid local AMI credential')
            # A lost credential cannot authorize overwriting an existing
            # application configuration. Reconciliation is required instead.
            for target in OWNED_PATHS['configuration']:
                try:
                    parent = open_directory(str(Path(target).parent))
                except FileNotFoundError:
                    continue
                try:
                    try:
                        os.stat(Path(target).name, dir_fd=parent, follow_symlinks=False)
                    except FileNotFoundError:
                        continue
                    raise ValueError('configuration exists without protected AMI identity')
                finally:
                    os.close(parent)
            create({'Target': '/etc/almadial/secrets/vicidial-ami', 'Owner': 'root',
                    'Group': 'root', 'Mode': 0o600, 'Label': 'etc_t'},
                   (secret + '\n').encode())
            print(json.dumps({'Created': True}))
            return 0
        files = p['Files']
        if not files or len(files) > 3000 or len({f['Target'] for f in files}) != len(files):
            raise ValueError('invalid S05 file manifest')
        if sys.argv[1:] == ['inspect']:
            result = [observation(f) for f in files]
        elif sys.argv[1:] == ['install']:
            result = install(files)
        elif len(sys.argv) == 3 and sys.argv[1] in ('owned-inspect', 'owned-install'):
            result = owned(files, sys.argv[2], sys.argv[1] == 'owned-install')
        else:
            raise ValueError('unsupported S05 adapter operation')
        print(json.dumps(result))
    except (OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile):
        # Never print input, source bytes or platform exception paths.
        print('S05 file verification failed; preserve state for reconciliation', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
