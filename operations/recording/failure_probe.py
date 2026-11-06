#!/usr/bin/python3
"""Exercise installed status/error paths with isolated state and mocked actions.

No database connection, native processing, production status change or live audio.
"""
import contextlib
import fcntl
import importlib.machinery
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
from unittest.mock import patch

loader=importlib.machinery.SourceFileLoader('recording','/usr/local/sbin/alma-vicidial-recording')
spec=importlib.util.spec_from_loader(loader.name,loader)
f=importlib.util.module_from_spec(spec)
loader.exec_module(f)
os.umask(0o077)
result={}
with tempfile.TemporaryDirectory(prefix='pr01-failure-',dir='/root') as tmp:
    base=Path(tmp)
    f.STATE=base/'state'
    f.STATE.mkdir(mode=0o700)
    f.CONFIG=base/'policy.json'
    f.CONFIG.write_text('{"automatic_start_epoch":0}')
    f.CONFIG.chmod(0o600)
    def exercise(name,expected,context):
        with context,patch('sys.argv',['recording']),contextlib.redirect_stdout(io.StringIO()):
            try:
                rc=f.main()
            except f.Refused as e:
                rc=1
                assert str(e)==expected
        state=json.loads((f.STATE/'status.json').read_text())
        assert state['outcome']=='failed' and state['reason']==expected and rc==1
        result[name]=dict(exit_code=rc,status=state['outcome'],reason=state['reason'])
    exercise('missing_tools','audio_tools_missing',patch.object(f.Path,'is_file',return_value=False))
    exercise('source_pin','native_source_changed_review_required',patch.object(f,'digest',return_value='wrong'))
    row=dict(id=999,name='SYNTHETIC',start=1,end=2,length=1,location='')
    with patch.object(f,'command',return_value='0\t0'),patch.object(f,'rows',return_value=[row]):
        exercise('processing_failure','synthetic_native_failure',patch.object(f,'process',side_effect=f.Refused('synthetic_native_failure')))
    with (f.STATE/'lock').open('a') as lock,patch('sys.argv',['recording']):
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        try:
            f.main()
            raise AssertionError('overlap accepted')
        except f.Refused as e:
            assert str(e)=='processing_already_running'
        result['overlap']='rejected without overwriting active-job status'
    with patch.object(f,'command',return_value='0\t0'),patch.object(f,'rows',return_value=[]),patch('sys.argv',['recording']),contextlib.redirect_stdout(io.StringIO()):
        assert f.main()==0
        assert json.loads((f.STATE/'status.json').read_text())['outcome']=='ok'
    result['recovery_from_failure']='successful empty fixture run restores healthy status'
    with patch('sys.argv',['recording','--health']),contextlib.redirect_stdout(io.StringIO()):
        assert f.main()==0
    result['health']='root-only fixture status accepted'
print(json.dumps(dict(scope='installed code; private temporary state; mocked actions; NO live database/audio writes',tests=result),indent=2))
