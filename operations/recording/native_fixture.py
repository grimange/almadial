#!/usr/bin/python3
"""Run the installed UNMODIFIED Perl processor with synthetic audio and a DBI double.

Remote fixture only: never connects to MariaDB or reads real recordings.
Prints metadata, not audio. Temporary files are private and removed on exit.
"""
import hashlib
import json
import math
import os
from pathlib import Path
import struct
import subprocess
import tempfile
import wave

DBI = r'''
package DBI;
sub import {}
sub connect { bless {}, 'FakeDB' }
package FakeDB;
sub prepare { my ($self,$sql)=@_; bless {sql=>$sql}, 'FakeStatement' }
sub do { my ($self,$sql)=@_; open(my $f,'>>',$ENV{FIXTURE_SQL}) or die; print $f "$sql\n"; close($f); return 1; }
sub disconnect {1}
package FakeStatement;
sub execute {1}
sub rows {1}
sub finish {1}
sub fetchrow_array {
 my ($self)=@_;
 return (0,0,0) if $self->{sql} =~ /system_settings/;
 return (60) if $self->{sql} =~ /FROM servers/;
 return (999,1,0,'fixture','2026-01-01 00:00:00','2026-01-01 00:00:01','fixture',1);
}
1;
'''

os.umask(0o077)
native = Path('/usr/share/astguiclient/AST_CRON_audio_1_move_VDonly.pl')
expected = 'a9837929fcda93d4ca238eb0f06ac5d2fc3198c3029c2e6132f29fe3091c25c0'
assert hashlib.sha256(native.read_bytes()).hexdigest() == expected
with tempfile.TemporaryDirectory(prefix='pr01-native-', dir='/root') as tmp:
    base = Path(tmp)
    src, dest = base / 'raw', base / 'done'
    src.mkdir()
    dest.mkdir()
    (base / 'DBI.pm').write_text(DBI)
    stem = 'PR01_SYNTHETIC'
    for suffix in ('in', 'out'):
        with wave.open(str(src / f'{stem}-{suffix}.wav'), 'wb') as w:
            w.setparams((1, 2, 8000, 0, 'NONE', 'not compressed'))
            if suffix == 'in':
                w.writeframes(b''.join(struct.pack('<h', int(3000 * math.sin(i * .3))) for i in range(8000)))
    sha = hashlib.sha256((src / f'{stem}-in.wav').read_bytes()).hexdigest()
    env = dict(os.environ, PERL5LIB=str(base), FIXTURE_SQL=str(base / 'updates'))
    argv = ['perl', '-s', str(native), '-CLImonitor=1', '-PATHmonitor=' + str(src),
            '-CLIDONEmonitor=1', '-PATHDONEmonitor=' + str(dest), '--', '--HTTPS', '--quiet']
    p = subprocess.run(argv, env=env, capture_output=True, timeout=60)
    assert p.returncode == 0, 'native fixture process failed'
    output = dest / (stem + '-all.wav')
    assert hashlib.sha256(output.read_bytes()).hexdigest() == sha
    assert not list(src.iterdir()), 'native should consume only staged copies'
    updates = (base / 'updates').read_text()
    assert 'https://2.29.153.161/RECORDINGS/' + output.name in updates
    count = updates.count('UPDATE recording_log')
    assert count == 1
    p2 = subprocess.run(argv, env=env, capture_output=True, timeout=60)
    assert p2.returncode == 0 and (base / 'updates').read_text() == updates
    print(json.dumps(dict(native_sha256=expected, database='DBI double; NO live writes',
                         synthetic_seconds=1, empty_companion=True, byte_identical=True,
                         https_location_generated=True, repeated_run_no_duplicate_update=True,
                         native_returncode=p.returncode, repeated_returncode=p2.returncode)))
