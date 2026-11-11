#!/usr/bin/env python3
"""Root-only isolated qualification; no live logs or service changes.

Run on dialer01: python3 -B - < tests/runtime/qualify_agi_log_acl.py
Uses the installed tmpfiles file as input, replacing its paths with /tmp fixtures.
"""
import json
import os
from pathlib import Path
import pwd
import subprocess
import tempfile


def run_as(user, code, path):
    return subprocess.run(
        ['runuser', '-u', user, '--', 'python3', '-c', code, str(path)],
        capture_output=True, text=True, check=False,
    )


def attrs(path):
    return {key: os.getxattr(path, key) for key in os.listxattr(path)
            if key.startswith('system.posix_acl')}


def main():
    if os.geteuid() != 0:
        raise SystemExit('Requires root on the authorized lab host; uses /tmp only')
    pwd.getpwnam('asterisk')
    source = Path('/etc/tmpfiles.d/almadial-vicidial-agi-logs.conf').read_text()
    results = []
    with tempfile.TemporaryDirectory(prefix='alm44-acl-qualify-') as root:
        root = Path(root)
        root.chmod(0o755)
        logs = root / 'logs'
        logs.mkdir(mode=0o755)
        archive = logs / 'archive'
        archive.mkdir(mode=0o755)
        old = logs / 'action_process.previous'
        old.write_text('synthetic existing root log\n')
        old.chmod(0o644)
        initial = (old.stat().st_mode & 0o7777, attrs(old))
        config = root / 'test.conf'
        config.write_text(source.replace('/var/log/astguiclient', str(logs)))
        apply = ['systemd-tmpfiles', '--create', str(config)]
        subprocess.run(apply, check=True, capture_output=True)
        append = 'import sys;open(sys.argv[1],"a").write("synthetic\\n")'
        assert run_as('asterisk', append, old).returncode == 0
        results.append('existing root-created daily action log is appendable')
        fresh = logs / 'action_send.tomorrow'
        fresh.write_text('root creates tomorrow first\n')
        assert run_as('asterisk', append, fresh).returncode == 0
        results.append('future root-created 0666-open file inherits access')
        by_asterisk = logs / 'action_process.nextday'
        assert run_as('asterisk', append, by_asterisk).returncode == 0
        with by_asterisk.open('a') as stream:
            stream.write('root also appends\n')
        results.append('both creators remain compatible')
        assert run_as('nobody', append, old).returncode != 0
        assert run_as('nobody', append, logs / 'unauthorized').returncode != 0
        results.append('unrelated unprivileged user cannot append or create')
        assert not attrs(archive) and (archive.stat().st_mode & 0o7777) == 0o755
        results.append('archive access and permissions unchanged')
        # First reconciliation normalizes new files' inherited named-user ACL
        # from rwx/mask-rw to rw/mask-rw; effective access stays read/write.
        before = (attrs(logs), attrs(old))
        subprocess.run(apply, check=True, capture_output=True)
        assert before == (attrs(logs), attrs(old))
        assert run_as('asterisk', append, fresh).returncode == 0
        assert run_as('asterisk', 'import os,sys;sys.exit(os.access(sys.argv[1],os.X_OK))', fresh).returncode == 0
        before = (attrs(logs), attrs(old), attrs(fresh))
        subprocess.run(apply, check=True, capture_output=True)
        assert before == (attrs(logs), attrs(old), attrs(fresh))
        results.append('repeated tmpfiles reconciliation is idempotent')
        # Verify exact ACL/mode restoration technique without live rollback.
        for key in list(attrs(old)):
            os.removexattr(old, key)
        os.chmod(old, initial[0])
        for key, value in initial[1].items():
            os.setxattr(old, key, value)
        assert (old.stat().st_mode & 0o7777, attrs(old)) == initial
        assert run_as('asterisk', append, old).returncode != 0
        results.append('scoped metadata rollback restores original access')
    print(json.dumps({'result': 'PASS', 'checks': results}, indent=2))


if __name__ == '__main__':
    main()
