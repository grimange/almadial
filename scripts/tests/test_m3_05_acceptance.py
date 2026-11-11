"""Schema tests only: synthetic fixtures never assert real call acceptance."""
import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location(
    'acceptance', Path(__file__).parents[1] / 'lab/m3_05_acceptance.py')
acceptance = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(acceptance)


class AcceptanceTests(unittest.TestCase):
    def test_incomplete_template_cannot_pass(self):
        with tempfile.TemporaryDirectory() as root:
            result = acceptance.assess(acceptance.template(), root)
        self.assertFalse(result['ready_for_independent_qa'])
        self.assertEqual(len(result['outstanding']), len(acceptance.REQUIRED) + 1)

    def test_cannot_drop_agent_or_listening_requirements(self):
        for case in ['agent-transfer', 'operator-listening', 'reboot-persistence']:
            with self.subTest(case=case), tempfile.TemporaryDirectory() as root:
                index = acceptance.template()
                del index['checks'][case]
                with self.assertRaisesRegex(ValueError, 'inventory'):
                    acceptance.assess(index, root)

    def test_unattributed_pass_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            index = acceptance.template()
            index['checks']['register']['status'] = 'PASS'
            with self.assertRaisesRegex(ValueError, 'unattributed'):
                acceptance.assess(index, root)

    def test_hash_drift_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            Path(root, 'observation.json').write_text('changed')
            index = acceptance.template()
            index['checks']['register']['artifacts'] = [
                {'path': 'observation.json', 'sha256': hashlib.sha256(b'original').hexdigest()}]
            with self.assertRaisesRegex(ValueError, 'digest mismatch'):
                acceptance.assess(index, root)

    def test_reference_cannot_escape_evidence(self):
        with tempfile.TemporaryDirectory() as root:
            outside = Path(root, 'outside')
            outside.write_bytes(b'fixture')
            evidence = Path(root, 'evidence')
            evidence.mkdir()
            index = acceptance.template()
            index['checks']['register']['artifacts'] = [
                {'path': '../outside', 'sha256': hashlib.sha256(b'fixture').hexdigest()}]
            with self.assertRaisesRegex(ValueError, 'unsafe'):
                acceptance.assess(index, evidence)
