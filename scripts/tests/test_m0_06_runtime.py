"""Offline tests cannot establish live conference, recording or AGI success."""
import array
import hashlib
import wave
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

LAB = Path(__file__).resolve().parents[1] / 'lab'
sys.path.insert(0, str(LAB))
import m0_06_fastagi as fastagi
import m0_06_media as media
import m0_03_signal as signal


class DerivationTests(unittest.TestCase):
    def test_unreviewed_source_rejected(self):
        with self.assertRaises(ValueError):
            fastagi.derive(b'unreviewed source')

    def test_pin_is_exact_digest(self):
        self.assertEqual(len(fastagi.SOURCE_SHA256), 64)
        self.assertEqual(fastagi.SOURCE_SHA256, 'f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a')

    def test_exact_substitutions_and_strict_sql(self):
        source = ("'/etc/astguiclient.conf'\n'/etc/astguiclient.conf'\nport=>4577,\n"
                  "user=>'root',\ngroup=>'root',\n'$now_date_epoch','','','','','$callerid')")
        with mock.patch.object(fastagi, 'SOURCE_SHA256', hashlib.sha256(source.encode()).hexdigest()):
            a = fastagi.derive(source.encode())
            self.assertEqual(a, fastagi.derive(source.encode()))
            self.assertIn(b"'$now_date_epoch',NULL,0,0,0,'$callerid')", a)
            self.assertIn(b"host=>'127.0.0.1'", a)
            self.assertIn(b"user=>'asterisk'", a)

    def test_ambiguous_preimage_rejected_even_with_hash(self):
        source = b"'/etc/astguiclient.conf'"
        with mock.patch.object(fastagi, 'SOURCE_SHA256', hashlib.sha256(source).hexdigest()):
            with self.assertRaises(ValueError):
                fastagi.derive(source)


class MediaTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.temp.name)
        old = signal.PATTERN
        try:
            for identity, pattern in media.PATTERNS.items():
                signal.PATTERN = pattern
                signal.fixture(cls.root / (identity + '.wav'), 10)
        finally:
            signal.PATTERN = old
        import wave
        with wave.open(str(cls.root / 'silence.wav'), 'wb') as out:
            out.setparams((1, 2, 16000, 0, 'NONE', 'not compressed'))
            out.writeframes(b'\0' * 320000)

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def test_signal_self_check_is_offline_only(self):
        r = media.compare(self.root/'9051.wav', self.root/'9051.wav', '9051')
        self.assertEqual(r['analysis_state'], 'PASS')
        self.assertEqual(r['runtime_state'], 'NOT_RUN')
        self.assertEqual(r['thresholds']['tone_agreement'], .90)

    def test_silence_rejected(self):
        self.assertEqual(media.compare(self.root/'9051.wav', self.root/'silence.wav', '9051')['analysis_state'], 'FAIL')

    def test_local_loopback_from_wrong_participant_rejected(self):
        self.assertEqual(media.compare(self.root/'9051.wav', self.root/'9052.wav', '9051')['analysis_state'], 'FAIL')

    def test_wrong_injected_pattern_rejected(self):
        self.assertEqual(media.compare(self.root/'9052.wav', self.root/'9052.wav', '9051')['analysis_state'], 'FAIL')

    def test_empty_capture_rejected(self):
        p = self.root/'empty.wav'
        import wave
        with wave.open(str(p), 'wb') as out:
            out.setparams((1, 2, 16000, 0, 'NONE', 'not compressed'))
            out.writeframes(b'')
        with self.assertRaises(ValueError):
            media.compare(self.root/'9051.wav', p, '9051')

    def test_pattern_selection_restored(self):
        before = signal.PATTERN
        media.compare(self.root/'9052.wav', self.root/'9052.wav', '9052')
        self.assertEqual(signal.PATTERN, before)


class RecordingTests(unittest.TestCase):
    def test_mixed_content_and_single_source_negative(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            old = signal.PATTERN
            try:
                for identity in ['9051', '9052']:
                    signal.PATTERN = media.PATTERNS[identity]
                    signal.fixture(root / (identity + '.wav'), 10)
            finally:
                signal.PATTERN = old
            a, b = [signal.read_pcm(root / (i + '.wav')) for i in ['9051', '9052']]
            pcm = array.array('h', [round((x+y)*.5*32767) for x,y in zip(a,b)])
            if sys.byteorder != 'little': pcm.byteswap()
            with wave.open(str(root/'mix.wav'), 'wb') as f:
                f.setparams((1,2,16000,0,'NONE','not compressed'))
                f.writeframes(pcm.tobytes())
            result = media.mixed_recording(root/'mix.wav',root/'9051.wav',root/'9052.wav',0)
            self.assertEqual(result['analysis_state'], 'PASS')
            self.assertEqual(result['runtime_state'], 'NOT_RUN')
            self.assertEqual(media.mixed_recording(root/'9051.wav',root/'9051.wav',root/'9052.wav',0)['analysis_state'], 'FAIL')


class AMITests(unittest.TestCase):
    def test_injected_header_rejected_before_send(self):
        import m0_06_runtime as runtime
        ami = runtime.AMI.__new__(runtime.AMI)
        ami.socket = mock.Mock()
        with self.assertRaises(ValueError):
            ami.send('Ping', Bad='value\r\nAction: Command')
        ami.socket.sendall.assert_not_called()

    def test_failed_action_cannot_be_reported_successful(self):
        import m0_06_runtime as runtime
        ami = runtime.AMI.__new__(runtime.AMI)
        ami.rows = [{'ActionID':'test','Response':'Error','Message':'Permission denied'}]
        ami.send = mock.Mock(return_value='test')
        ami.pump = mock.Mock()
        with self.assertRaisesRegex(RuntimeError, 'rejected'):
            ami.require('Redirect')


class RecordingProvenanceTests(unittest.TestCase):
    def test_event_duration_contract(self):
        import m0_06_runtime as runtime
        rows = [{'Event':'ConfbridgeRecord','utc':'2026-11-06T01:50:35.565681+00:00'},
                {'Event':'ConfbridgeStopRecord','utc':'2026-11-06T01:51:22.013626+00:00'}]
        self.assertEqual(runtime.recording_interval(rows,46.44)[2],46)
        with self.assertRaises(ValueError): runtime.recording_interval(rows,20)
        with self.assertRaises(ValueError): runtime.recording_interval(rows[:1],46)
        with self.assertRaises(ValueError): runtime.recording_interval(rows+rows,46)


class CleanupTests(unittest.TestCase):
    def test_ami_connection_failure_releases_claim(self):
        import m0_06_runtime as runtime
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root/'ami-secret').write_text('fake-fixture-secret')
            calls = []
            def query(sql):
                calls.append(sql)
                return '1' if 'ROW_COUNT' in sql else ''
            with mock.patch.object(runtime,'ROOT',root), \
                 mock.patch.object(runtime,'command',return_value='almadial-alma10'), \
                 mock.patch.object(runtime,'cli',return_value='0 active channels'), \
                 mock.patch.object(runtime,'sql',side_effect=query), \
                 mock.patch.object(runtime,'AMI',side_effect=ConnectionError('fixture refusal')):
                with self.assertRaises(ConnectionError): runtime.run('failure-fixture')
            self.assertTrue(any("SET extension=''" in q for q in calls))
