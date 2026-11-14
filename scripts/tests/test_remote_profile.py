"""Focused remote profile contracts; runtime acceptance uses separate host evidence."""
import copy
import io
import json
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[2]
def adapter(name):
    ns={'__name__':'remote_fixture'}
    exec(compile((ROOT/'internal/provision'/name).read_text(),name,'exec'),ns)
    return ns

def payload():
    return {'Request':{'Identity':'alm_s10_9061','Hostname':'dialer.example.test','WSSURL':'wss://dialer.example.test/ws'},'Network':{'Mode':'remote','ManagementAddress':'192.0.2.10','RTPStart':12000,'RTPEnd':12019}}

class RemoteProfile(unittest.TestCase):
    def test_same_origin_and_media_bound_to_request(self):
        ns=adapter('phone_native.py');p=payload();ns['validate_request'](p)
        text=ns['config']('fake-password',p).decode()
        for item in ('media_address=192.0.2.10','rtp_port_start=12000','rtp_port_end=12019','ice_support=yes','dtls_verify=fingerprint','dtls_setup=actpass','bind=127.0.0.1','context=alm-remote-synthetic'):
            self.assertIn(item,text)
        launcher=ns['launcher'](p).decode()
        self.assertIn('wss://dialer.example.test/ws',launcher)
        self.assertNotIn('wss://localhost',launcher)
        self.assertNotIn('fake-password',launcher)
        self.assertIn('MixMonitor(',ns['ROUTES'].decode())
        self.assertNotIn('Dial(',ns['ROUTES'].decode())
    def test_invalid_request_stops_before_effects(self):
        for change in ('url','hostname','address','ports'):
            ns=adapter('phone_native.py');p=payload()
            if change=='url':p['Request']['WSSURL']='wss://other.example.test/ws'
            if change=='hostname':p['Request']['Hostname']="evil'.test"
            if change=='address':p['Network']['ManagementAddress']='127.0.0.1'
            if change=='ports':p['Network']['RTPStart']=12001
            ns['activate']=lambda p:self.fail('unvalidated remote activation')
            with patch('sys.argv',['adapter','phone-activate']),patch('sys.stdin',io.StringIO(json.dumps(p))),patch('sys.stderr',io.StringIO()):
                self.assertEqual(ns['main'](),1)
    def test_proxy_port_ownership_is_exact_and_conflicts_fail(self):
        ns=adapter('tls_native.py')
        for output, expected in [('',None),('http_port_t tcp 8088','http_port_t')]:
            ns['run']=lambda args, output=output:output
            self.assertEqual(ns['proxy_port'](),expected)
        for output in ('other_port_t tcp 8088','http_port_t tcp 8080-8090','http_port_t tcp 8088\nhttp_port_t tcp 8088'):
            ns['run']=lambda args, output=output:output
            with self.assertRaises(ValueError): ns['proxy_port']()

    def test_lab_still_has_no_call_route_or_public_media(self):
        ns=adapter('phone_native.py');text=ns['config']('fake-password').decode()
        self.assertIn('context=alm-s10-no-calls',text)
        self.assertNotIn('media_address=',text)
        self.assertNotIn('rtp_port_start=',text)
        self.assertIn('wss://localhost:8089/ws',ns['launcher']({}).decode())

if __name__=='__main__':unittest.main()
