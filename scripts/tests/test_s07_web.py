"""S07 effects and activation boundaries, without root or live services."""
import base64
import hashlib
import io
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[2]
def adapter():
    ns={'__name__':'s07_fixture'}
    for name in ('vicidial_files.py','web_native.py'):
        exec(compile((ROOT/'internal/provision'/name).read_text(),name,'exec'),ns)
    return ns

class WebContract(unittest.TestCase):
    def files(self,ns):
        return [dict(Target=p,Owner='root',Group='root',Mode=0o644,Label='httpd_config_t',Content=base64.b64encode(b'fixture').decode(),SHA256=hashlib.sha256(b'fixture').hexdigest()) for p in ns['S07_PATHS']]
    def test_absent_valid_foreign_and_partial_files(self):
        ns=adapter();files=self.files(ns);writes=[]
        ns['observation']=lambda f:dict(Kind='absent')
        ns['create']=lambda f,r:writes.append(f['Target'])
        self.assertEqual(ns['web_files'](files,False)['Verified'],0)
        self.assertFalse(writes)
        ns['web_files'](files,True);self.assertEqual(writes,list(ns['S07_PATHS']))
        ns['observation']=lambda f:dict(f,Kind='regular',Links=1)
        self.assertEqual(ns['web_files'](files,False)['Missing'],[])
        writes.clear();ns['web_files'](files,True);self.assertFalse(writes)
        ns['observation']=lambda f:dict(f,Kind='regular',Links=1,SHA256='foreign')
        with self.assertRaises(ValueError):ns['web_files'](files,True)
        self.assertFalse(writes)
    def test_invalid_digest_and_foreign_path_before_write(self):
        ns=adapter();files=self.files(ns);writes=[];ns['create']=lambda *a:writes.append(a)
        for field,value in [('Target','/etc/httpd/conf/httpd.conf'),('SHA256','bad')]:
            with self.subTest(field=field):
                changed=[dict(f) for f in files];changed[0][field]=value
                with self.assertRaises(ValueError):ns['web_files'](changed,True)
                self.assertFalse(writes)
    def test_configuration_validation_precedes_reload(self):
        for fail in ('/usr/sbin/httpd','/usr/sbin/php-fpm',None):
            with self.subTest(fail=fail):
                ns=adapter();calls=[]
                ns['web_files']=lambda f,m:{'Missing':[]}
                ns['web_request']=lambda p:(200,None,'phone_login')
                def run(a):
                    calls.append(a)
                    if a[0]==fail:raise ValueError('invalid config')
                    return ''
                ns['web_run']=run
                with patch('sys.argv',['adapter','web-services']),patch('sys.stdin',io.StringIO(__import__('json').dumps({'Files':self.files(ns)}))),patch('sys.stdout',io.StringIO()),patch('sys.stderr',io.StringIO()):
                    result=ns['web_main']()
                self.assertEqual(result,0 if fail is None else 1)
                reloads=[a for a in calls if 'reload' in a]
                self.assertEqual(len(reloads),1 if fail is None else 0)
                if reloads:self.assertEqual([a[0] for a in calls[:2]],['/usr/sbin/httpd','/usr/sbin/php-fpm'])
    def test_already_active_vhost_does_not_reload(self):
        ns=adapter();calls=[]
        ns['web_files']=lambda f,m:{'Missing':[]};ns['web_run']=lambda a:calls.append(a)
        ns['web_request']=lambda p:(200,'S07','phone_login')
        with patch('sys.argv',['adapter','web-services']),patch('sys.stdin',io.StringIO(__import__('json').dumps({'Files':self.files(ns)}))),patch('sys.stdout',io.StringIO()):
            self.assertEqual(ns['web_main'](),0)
        self.assertFalse(any('reload' in a for a in calls))
    def test_setup_updates_only_guarded_flag_and_preserves_credentials_and_phones(self):
        ns=adapter();queries=[];settings=iter(['Y','N'])
        ns['open_directory']=lambda p:123;ns['read_at']=lambda *a:b'a'*48+b'\n'
        def sql(q):
            queries.append(q)
            if q.startswith('SELECT COUNT'):return '1'
            if q.startswith('SELECT first'):return next(settings)
            return ''
        ns['web_sql']=sql
        with patch('os.close'):
            self.assertEqual(ns['web_setup'](True)[0],'N')
        update=[q for q in queries if 'UPDATE ' in q][0]
        self.assertIn("SET first_login_trigger='N' WHERE first_login_trigger='Y'",update)
        self.assertNotIn('UPDATE phones',update);self.assertNotIn('SET pass=',update)
        self.assertNotIn('ALTER ',update)

    def test_authentication_accepts_upstream_markup_not_unsecured_or_setup_pages(self):
        ns=adapter()
        markup='<a href="admin.php?force_logout=1">Logout</a> <FONT>(6666)</FONT>'
        self.assertTrue(ns['web_admin_ready'](200,markup))
        for status,text in [(401,markup),(200,'Initial Setup '+markup),(200,'Login'),(302,markup)]:
            self.assertFalse(ns['web_admin_ready'](status,text))
    def test_authenticated_probe_does_not_follow_redirects(self):
        ns=adapter()
        self.assertIsNone(ns['WebNoRedirect']().redirect_request(None,None,302,'redirect',{},'http://foreign.invalid'))
