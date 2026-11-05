import array
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import wave

SPEC = importlib.util.spec_from_file_location('signal_fixture',Path(__file__).parents[1]/'lab/m0_03_signal.py')
mod = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(mod)

class SignalTests(unittest.TestCase):
    def test_fixture_and_delayed_echo(self):
        with tempfile.TemporaryDirectory() as d:
            a,b=Path(d)/'a.wav',Path(d)/'b.wav'
            digest=mod.fixture(a,12)
            self.assertEqual(len(digest),64)
            with wave.open(str(a),'rb') as f: data=f.readframes(f.getnframes())
            with wave.open(str(b),'wb') as f:
                f.setparams((1,2,mod.RATE,0,'NONE','not compressed')); f.writeframes(b'\x00\x00'*3200+data[:-6400])
            result=mod.compare(a,b)
            self.assertEqual(result['state'],'PASS',result)
            self.assertEqual(result['estimated_latency_ms'],200)
    def test_silence_is_not_echo(self):
        with tempfile.TemporaryDirectory() as d:
            a,b=Path(d)/'a.wav',Path(d)/'b.wav';mod.fixture(a,9)
            with wave.open(str(b),'wb') as f:
                f.setparams((1,2,mod.RATE,0,'NONE','not compressed'));f.writeframes(b'\0\0'*mod.RATE*9)
            self.assertEqual(mod.compare(a,b)['state'],'FAIL')
    def test_short_capture_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'a.wav';mod.fixture(p,1)
            with self.assertRaisesRegex(ValueError,'eight seconds'):mod.compare(p,p)
    def test_same_envelope_with_wrong_tones_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            a,b=Path(d)/'a.wav',Path(d)/'b.wav';mod.fixture(a,12)
            with patch.object(mod,'FREQUENCIES',mod.FREQUENCIES[1:]+mod.FREQUENCIES[:1]):
                mod.fixture(b,12)
            result=mod.compare(a,b)
            self.assertGreater(result['envelope_correlation'],.99)
            self.assertEqual(result['state'],'FAIL')
    def test_unrecognized_input_pattern_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'a.wav'
            with patch.object(mod,'PATTERN',(0,)*16):mod.fixture(p,12)
            self.assertEqual(mod.compare(p,p)['state'],'FAIL')
    def test_two_second_latency_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            a,b=Path(d)/'a.wav',Path(d)/'b.wav';mod.fixture(a,12)
            with wave.open(str(a),'rb') as f:data=f.readframes(f.getnframes())
            with wave.open(str(b),'wb') as f:
                f.setparams((1,2,mod.RATE,0,'NONE','not compressed'))
                f.writeframes(b'\x00\x00'*(2*mod.RATE)+data[:-(4*mod.RATE)])
            self.assertEqual(mod.compare(a,b)['state'],'FAIL')
