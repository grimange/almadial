"""Deterministic PCM fixtures exercise the content check, never runtime acceptance."""
import array
import math
from pathlib import Path
import sys
import tempfile
import unittest
import wave

sys.path.insert(0, str(Path(__file__).parents[1] / 'lab'))
import m3_05_recording as recording


class RecordingTests(unittest.TestCase):
    def fixture(self, root, seconds=12, missing_second=False):
        samples = [array.array('h') for _ in range(3)]
        for index in range(16000 * seconds):
            time = index / 16000
            slot = int(time / .5) % 4
            left = .15 * math.sin(2 * math.pi * recording.signal.FREQUENCIES[slot] * time)
            right = .15 * math.sin(2 * math.pi * recording.signal.FREQUENCIES[(slot + 1) % 4] * time)
            for out, value in zip(samples, [left, right, left if missing_second else left + right]):
                out.append(round(32767 * value))
        paths = [Path(root, name + '.wav') for name in ['a', 'b', 'mixed']]
        for path, pcm in zip(paths, samples):
            if sys.byteorder != 'little':
                pcm.byteswap()
            with wave.open(str(path), 'wb') as output:
                output.setparams((1, 2, 16000, 0, 'NONE', 'not compressed'))
                output.writeframes(pcm.tobytes())
        return recording.analyze(paths[2], paths[0], paths[1])

    def test_both_changing_sources_required(self):
        with tempfile.TemporaryDirectory() as root:
            result = self.fixture(root)
        self.assertEqual(result['analysis_state'], 'PASS')
        self.assertEqual(result['runtime_state'], 'NOT_RUN')
        self.assertEqual(result['agreement'], 1)

    def test_one_missing_direction_cannot_pass(self):
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(self.fixture(root, missing_second=True)['analysis_state'], 'FAIL')

    def test_short_capture_cannot_pass(self):
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(self.fixture(root, seconds=3)['analysis_state'], 'FAIL')
