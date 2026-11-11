"""Deterministic S06 native observation fixtures; no root, systemd or workers."""
import base64
import hashlib
import os
import tempfile
from pathlib import Path
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


def adapter():
    ns = {'__name__': 's06_fixture'}
    exec(compile((ROOT / 'internal/provision/vicidial_files.py').read_text(), 's05-primitives', 'exec'), ns)
    source = (ROOT / 'internal/provision/runtime_native.py').read_text()
    exec(compile(source.removesuffix('sys.exit(runtime_main())\n'), 's06-adapter', 'exec'), ns)
    return ns


class RuntimeState(unittest.TestCase):
    def setUp(self):
        self.ns = adapter()
        self.units = {}
        for name in ('mariadb', 'asterisk', 'httpd', 'php-fpm', 'crond', 'almadial-s06-fastagi'):
            self.units[name + '.service'] = {'ActiveState': 'active', 'UnitFileState': 'enabled', 'NeedDaemonReload': 'no', 'MainPID': '100'}
        for name in ('updater', 'listener'):
            self.units['almadial-s06-' + name + '.service'] = {'ActiveState': 'inactive', 'UnitFileState': 'disabled'}
        self.selinux = 'Enforcing'
        self.sockets = 'tcp LISTEN 0 10 127.0.0.1:4577 0.0.0.0:* users:(("perl",pid=100,fd=3))'
        self.processes = [(100, 1)] + [(p, 100) for p in range(101, 106)]
        self.ns.update(runtime_schedules=lambda: 'f'*64, runtime_owned=lambda f, m: 0,
                       unit=lambda n: self.units[n], runtime_processes=lambda u: self.processes,
                       run=lambda a: self.selinux if a[0].endswith('getenforce') else self.sockets if a[0].endswith('ss') else '')

    def state(self):
        return self.ns['runtime_state']([{'Target': 'fixture', 'Content': 'excluded'}])

    def test_current_health_and_volatile_pid_independence(self):
        first = self.state()
        self.processes = [(500, 1)] + [(p, 500) for p in range(501, 506)]
        self.sockets = self.sockets.replace('pid=100', 'pid=500')
        self.assertEqual(first, self.state())
        self.assertEqual(len(first['Health']), 64)

    def test_existing_loopback_chronyd_is_preserved(self):
        self.sockets += '\nudp UNCONN 0 0 [::1]:323 [::]:* users:(("chronyd",pid=50,fd=5))'
        self.assertEqual(len(self.state()["Health"]), 64)

    def test_missing_child_is_unhealthy(self):
        self.processes = []
        self.sockets = ''
        self.assertEqual(self.state()['Health'], 'absent')

    def test_disabled_required_unit_needs_action(self):
        self.units['httpd.service']['UnitFileState'] = 'disabled'
        self.assertEqual(self.state()['Services'], 'absent')

    def test_missing_foundation_dependency_blocks(self):
        self.units['mariadb.service']['ActiveState'] = 'inactive'
        with self.assertRaises(ValueError): self.state()

    def test_deferred_activation_and_overrides_block(self):
        for field, value in [('ActiveState', 'active'), ('UnitFileState', 'enabled'), ('DropInPaths', '/foreign.conf')]:
            with self.subTest(field=field):
                self.setUp()
                self.units['almadial-s06-updater.service'][field] = value
                with self.assertRaises(ValueError): self.state()

    def test_public_management_sip_and_foreign_socket_owner_block(self):
        for sockets in [self.sockets.replace('127.0.0.1', '0.0.0.0'),
                        self.sockets.replace(':4577', ':5060'),
                        self.sockets.replace('pid=100', 'pid=999')]:
            with self.subTest(sockets=sockets):
                self.sockets = sockets
                with self.assertRaises(ValueError): self.state()

    def test_s08_readonly_listener_handoff_is_closed(self):
        original = self.sockets
        for addr, owner, allowed in [('127.0.0.1:443','httpd',True), ('127.0.0.1:8089','asterisk',True), ('0.0.0.0:443','httpd',False), ('127.0.0.1:8089','foreign',False), ('127.0.0.1:5060','asterisk',False)]:
            with self.subTest(addr=addr,owner=owner):
                self.sockets = original+'\n'+'tcp LISTEN 0 10 '+addr+' 0.0.0.0:* users:(("'+owner+'",pid=700,fd=3))'
                with self.assertRaises(ValueError): self.state()
                if allowed: self.assertEqual(len(self.ns['runtime_state']([{'Target':'fixture','Content':'excluded'}],True)['Health']),64)
                else:
                    with self.assertRaises(ValueError): self.ns['runtime_state']([{'Target':'fixture','Content':'excluded'}],True)

    def test_selinux_permissive_blocks(self):
        self.selinux = 'Permissive'
        with self.assertRaises(ValueError): self.state()

    def test_missing_configuration_not_complete(self):
        self.ns['runtime_owned'] = lambda f, m: len(f)
        result = self.state()
        self.assertEqual(result['Files'], 'absent')
        self.assertEqual(result['Health'], 'absent')

    def test_file_allowlist_rejects_foreign_path_before_write(self):
        ns = adapter()
        with self.assertRaises(ValueError): ns['runtime_owned']([{'Target': '/etc/crontab'}], True)


    def test_two_file_correction_preimages_and_foreign_metadata(self):
        ns = adapter()
        files = []
        for path in ns['S06_PATHS']:
            group, label, mode = 'root', 'usr_t', 0o644
            if path == '/etc/astguiclient-s06.conf': group, label, mode = 'asterisk', 'etc_t', 0o640
            elif path.startswith('/etc/cron.d/'): label = 'system_cron_spool_t'
            elif path.startswith('/etc/systemd/'): label = 'systemd_unit_file_t'
            raw = path.encode()
            files.append(dict(Target=path,Owner='root',Group=group,Label=label,Mode=mode,Content=base64.b64encode(raw),SHA256=hashlib.sha256(raw).hexdigest(),PriorSHA256='a'*64))
        def observe(f):
            return dict(f, Kind='regular', Links=1, SHA256='a'*64 if f['Target'] in ('/etc/astguiclient-s06.conf','/usr/share/astguiclient/almadial-s06-ready.pl') else f['SHA256'])
        ns['observation'] = observe
        corrected = []
        ns['correct_local_socket'] = lambda f,r,o: corrected.append(f['Target'])
        self.assertEqual(ns['runtime_owned'](files,False), 2)
        self.assertEqual(corrected, [])
        self.assertEqual(ns['runtime_owned'](files,True), 2)
        self.assertEqual(len(corrected), 2)
        ns['observation'] = lambda f: dict(observe(f), Mode=0o666)
        with self.assertRaises(ValueError): ns['runtime_owned'](files,True)
        ns['observation'] = lambda f: dict(observe(f), SHA256='f'*64)
        with self.assertRaises(ValueError): ns['runtime_owned'](files,True)

    def test_bounded_command_failure_missing_module_or_permission(self):
        ns = adapter()
        class Failure:
            returncode, stdout, stderr = 1, b'', b'fixture diagnostic'
        with patch.object(ns['subprocess'], 'run', return_value=Failure()):
            with self.assertRaisesRegex(ValueError, 'bounded command failed'):
                ns['run'](['/usr/bin/perl', '-MNotInstalledFixture', '-e', '1'])

    def test_duplicate_files_rejected(self):
        ns = adapter()
        files = [{'Target': p} for p in ns['S06_PATHS']]
        files[-1] = files[0]
        with self.assertRaises(ValueError): ns['runtime_owned'](files, True)



class ProcessOwnership(unittest.TestCase):
    def setUp(self):
        self.ns = adapter()
        self.rows = {100: (1, 999, b'/usr/bin/perl\0/usr/share/astguiclient/almadial-s06-FastAGI_log.pl\0')}
        self.rows.update({p: (100, 999, self.rows[100][2]) for p in range(101,106)})
        rows = self.rows
        class ProcPath:
            def __init__(self, path): self.path = path
            def read_bytes(self): return rows[int(self.path.split('/')[2])][2]
            def read_text(self):
                parent, user, _ = rows[int(self.path.split('/')[2])]
                if self.path.endswith('/cgroup'): return '0::/system.slice/almadial-s06-fastagi.service\n'
                return 'PPid: '+str(parent)+'\nUid: '+(' '.join([str(user)]*4))+'\nGid: 999 999 999 999\n'
        self.ns.update(Path=ProcPath, uid=lambda u:999, gid=lambda g:999)
        self.listing = patch.object(self.ns['os'], 'listdir', return_value=list(map(str, rows)))
        self.listing.start()
        self.addCleanup(self.listing.stop)

    def test_native_pool_has_one_owner(self):
        self.assertEqual(len(self.ns['runtime_processes']({'MainPID':'100'})),6)

    def test_duplicate_supervisor_and_wrong_identity_block(self):
        for problem in ('parent','uid','command'):
            with self.subTest(problem=problem):
                original=self.rows[101]
                if problem=='parent': self.rows[101]=(1,999,original[2])
                if problem=='uid': self.rows[101]=(100,0,original[2])
                if problem=='command': self.rows[101]=(100,999,b'/usr/bin/perl\0/usr/share/astguiclient/ADMIN_keepalive_ALL.pl\0')
                with self.assertRaises(ValueError): self.ns['runtime_processes']({'MainPID':'100'})
                self.rows[101]=original

    def test_disappearing_process_observation_is_reconciled(self):
        del self.rows[105]
        # /proc entries may disappear during a snapshot; a missing native child
        # is classified by health, never replaced by launching another owner.
        class Gone(dict):
            def __getitem__(self,key):
                if key==105:raise FileNotFoundError()
                return super().__getitem__(key)
        # The live scanner already handles FileNotFoundError; simulate directly.
        original=self.ns['Path']
        def path(value):
            if value.startswith('/proc/105/'): raise FileNotFoundError()
            return original(value)
        self.ns['Path']=path
        self.assertEqual(len(self.ns['runtime_processes']({'MainPID':'100'})),5)

class CronOwnership(unittest.TestCase):
    def test_foreign_preservation_and_competing_schedule_rejection(self):
        ns=adapter()
        with tempfile.TemporaryDirectory() as root:
            base=Path(root)
            for directory in ('etc/cron.d','etc/cron.hourly','etc/cron.daily','etc/cron.weekly','etc/cron.monthly','var/spool/cron'): (base/directory).mkdir(parents=True,exist_ok=True)
            foreign=base/'etc/cron.d/operator'
            foreign.write_text('12 3 * * * root /usr/bin/true\n')
            ns['open_directory']=lambda path: os.open(base/path.lstrip('/'),os.O_RDONLY|os.O_DIRECTORY)
            ns['label']=lambda fd:'system_cron_spool_t'
            before=foreign.read_bytes()
            first=ns['runtime_schedules']()
            self.assertEqual(first,ns['runtime_schedules']())
            self.assertEqual(foreign.read_bytes(),before)
            managed=base/'etc/cron.d/almadial-s06'
            managed.write_text('# * * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl\n')
            self.assertEqual(first,ns['runtime_schedules']())
            managed.write_text('* * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl\n')
            with self.assertRaises(ValueError): ns['runtime_schedules']()
            self.assertEqual(foreign.read_bytes(),before)

if __name__ == '__main__':
    unittest.main()
