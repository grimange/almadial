"""ALM-32 offline artifact checks; no host or network access."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('generate_sbom', ROOT / 'scripts/generate_sbom.py')
SBOM = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SBOM)


class SBOMTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.document = SBOM.generate(SBOM.DEFAULT)

    def test_reproducible_and_current(self):
        self.assertEqual(SBOM.encoded(self.document), SBOM.encoded(SBOM.generate(SBOM.DEFAULT)))
        self.assertEqual(SBOM.encoded(self.document), (SBOM.DEFAULT / 'almadial.spdx.json').read_bytes())

    def test_relationship_and_license_references(self):
        ids = {p['SPDXID'] for p in self.document['packages']}
        self.assertEqual(len(ids), len(self.document['packages']))
        for relation in self.document['relationships']:
            self.assertIn(relation['spdxElementId'], ids)
            self.assertIn(relation['relatedSpdxElement'], ids)
        references = {x['licenseId'] for x in self.document['hasExtractedLicensingInfos']}
        for package in self.document['packages']:
            if package['licenseDeclared'].startswith('LicenseRef-'):
                self.assertIn(package['licenseDeclared'], references)

    def test_host_packages_are_not_guessed_licenses_or_payload_hashes(self):
        rpms = [p for p in self.document['packages'] if p['name'].startswith('rpm:')]
        self.assertEqual(len(rpms), 731)
        for package in rpms:
            self.assertEqual(package['licenseDeclared'], 'NOASSERTION')
            self.assertNotIn('checksums', package)
            self.assertIn('Verbatim RPM license metadata', package['licenseComments'])

    def test_conflicting_license_is_not_normalized(self):
        conflict = next(x for x in self.document['hasExtractedLicensingInfos'] if 'LineJS' in x['licenseId'])
        self.assertIn('cannot charge', conflict['extractedText'])
        self.assertIn('GPL', conflict['extractedText'])

    def test_tampered_evidence_rejected(self):
        original = Path.read_bytes
        evidence = SBOM.DEFAULT / json.loads((SBOM.DEFAULT / 'license-evidence.json').read_bytes())[0]['path']
        def read(path):
            return b'altered' if path == evidence else original(path)
        with patch.object(Path, 'read_bytes', read):
            with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
                SBOM.generate(SBOM.DEFAULT)

    def test_schema(self):
        try:
            import jsonschema
        except ImportError:
            self.skipTest('Install python-jsonschema to run official schema validation')
        jsonschema.validate(self.document, json.loads((SBOM.DEFAULT / 'spdx-2.3.schema.json').read_bytes()))


if __name__ == '__main__':
    unittest.main()
