"""S10 ownership and registration-only policy; no root or live workload."""
import io
import json
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[2]
def adapter():
    ns={'__name__':'s10_fixture'}
    exec(compile((ROOT/'internal/provision/phone_native.py').read_text(),'phone_native.py','exec'),ns)
    return ns

class PhoneContract(unittest.TestCase):
    def test_registration_only_configuration(self):
        ns=adapter();text=ns['config']('fake-fixture-password').decode()
        self.assertIn('protocol=wss\nbind=127.0.0.1',text)
        self.assertIn('max_contacts=1',text)
        self.assertIn('context=alm-s10-no-calls',text)
        self.assertNotIn('type=registration',text)
        values=ns['desired_values']('fake-fixture-password')
        self.assertEqual(values['extension'],ns['IDENTITY'])
        self.assertEqual(values['webphone_settings'],ns['CONTAINER'])
        self.assertEqual(values['webphone_dialpad'],'N')
        self.assertLessEqual(len(values['login']),15)
        self.assertLessEqual(len(values['phone_context']),20)
        self.assertLessEqual(len(values['ext_context']),20)
        self.assertIn('dialRegExten:0',ns['SETTINGS'])
    def test_foreign_or_public_request_rejected_without_effects(self):
        for request in ({},{'Identity':'callin','Hostname':'localhost','WSSURL':'wss://localhost:8089/ws'},{'Identity':'alm_s10_9061','Hostname':'localhost','WSSURL':'wss://public.example/ws'}):
            ns=adapter();effects=[];ns['activate']=lambda p:effects.append(p)
            with patch('sys.argv',['adapter','phone-activate']),patch('sys.stdin',io.StringIO(json.dumps({'Request':request}))),patch('sys.stderr',io.StringIO()):
                self.assertEqual(ns['main'](),1)
            self.assertFalse(effects)
    def test_native_failure_does_not_expose_sql_or_secret(self):
        ns=adapter()
        with patch('subprocess.run') as run:
            run.return_value.returncode=7
            run.return_value.stderr=b'password=fixture-secret'
            with self.assertRaisesRegex(ValueError,'mariadb exit 7') as err:
                ns['sql']('sensitive fixture SQL')
            self.assertNotIn('fixture-secret',str(err.exception))
            self.assertNotIn('sensitive',str(err.exception))
    def test_stale_plan_precedes_any_write(self):
        ns=adapter();ns['inspect']=lambda p:{'Current':'actual','Health':'absent'}
        with patch('os.open') as opened:
            with self.assertRaisesRegex(ValueError,'stale S10 activation'):
                ns['activate']({'ExpectedCurrent':'stale'})
            opened.assert_not_called()
    def test_complete_configuration_missing_runtime_reloads_without_redeployment(self):
        ns=adapter();states=iter([{'Current':'same','Health':'partial'},{'Current':'same','Health':'ready'}]);calls=[]
        ns['inspect']=lambda p:next(states)
        ns['run']=lambda argv:calls.append(argv)
        with patch('os.open') as opened:
            result=ns['activate']({'ExpectedCurrent':'same'})
            opened.assert_not_called()
        self.assertEqual(result['Health'],'ready')
        self.assertEqual(calls,[['/usr/sbin/asterisk','-rx','module reload res_pjsip.so']])
    def test_launcher_uses_local_settings_and_post(self):
        text=adapter()['LAUNCHER'].decode()
        self.assertIn('JOIN vicidial_settings_containers',text)
        self.assertIn('method="post" action="viciphone.php"',text)
        self.assertIn('wss://localhost:8089/ws',text)
        self.assertNotIn('SELECT p.pass',text)
        self.assertNotIn('https://',text)

if __name__=='__main__':unittest.main()
