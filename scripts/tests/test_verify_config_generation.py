"""Negative contract cases; fixtures do not assert live runtime execution."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest

SCRIPTS = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('config_contract', SCRIPTS / 'verify_config_generation.py')
contract = importlib.util.module_from_spec(spec)
spec.loader.exec_module(contract)
spec2 = importlib.util.spec_from_file_location('generator', SCRIPTS / 'lab/m0_05_generator.py')
generator = importlib.util.module_from_spec(spec2)
spec2.loader.exec_module(generator)


class ConfigurationContractTests(unittest.TestCase):
    def setUp(self):
        self.data = json.loads((SCRIPTS / 'tests/fixtures/config_generation/r4037.json').read_text())

    def fails(self, text):
        result = contract.validate(self.data)
        self.assertEqual(result['state'], 'FAIL', result)
        self.assertTrue(any(text in err for err in result['errors']), result)

    def test_observed_redacted_contract(self):
        result = contract.validate(self.data)
        self.assertEqual(result['errors'], [])
        self.assertEqual(result['runtime'], 'NOT_RUN')

    def test_ownership_includes_voicemail_exception(self):
        self.assertEqual(contract.OWNERS['voicemail.conf'], 'VICIdial')
        for base, generated in contract.INCLUDES.items():
            self.assertNotEqual(contract.OWNERS[base], contract.OWNERS[generated])

    def test_missing_generated_include_file(self):
        del self.data['files']['pjsip-vicidial.conf']
        self.fails('missing required include')

    def test_missing_include_declaration(self):
        self.data['files']['pjsip_wizard.conf'] = ''
        self.fails('include relationship')

    def test_duplicate_pjsip_object(self):
        self.data['files']['pjsip_wizard-vicidial.conf'] *= 2
        self.fails('duplicate PJSIP')

    def test_endpoint_wizard_name_collision(self):
        self.data['files']['pjsip_wizard-vicidial.conf'] += '\n[9001]\ntype=wizard\n'
        self.fails('endpoint/wizard collision')

    def test_legal_native_same_name_different_types(self):
        self.assertEqual(contract.validate(self.data)['state'], 'PASS')

    def test_unresolved_external_ip(self):
        self.data['files']['pjsip-vicidial.conf'] += '\nexternal_media_address=YOUR_SERVER_IP\n'
        self.fails('unresolved IP')

    def test_legacy_sip_route(self):
        self.data['files']['extensions.conf'] += '\nexten => 9060,1,Dial(SIP/9060)\n'
        self.fails('unsupported reachable')

    def test_r4037_duplicate_recording_priority(self):
        self.data['files']['extensions-vicidial.conf'] = self.data['files']['extensions-vicidial.conf'].replace('exten => 8310,2,MixMonitor(', 'exten => 8309,2,MixMonitor(')
        self.fails('duplicate dialplan priority')

    def test_version_specific_source_correction(self):
        source = ('exten => 8309,2,MixMonitor(first)\n'
                  '$Vext .= "exten => 8310,1,Answer\\n";\n'
                  'exten => 8309,2,MixMonitor(second)\n')
        corrected = generator.correct_gsm_priority(source)
        self.assertEqual(corrected, source.replace('8309,2,MixMonitor(second)', '8310,2,MixMonitor(second)'))
        with self.assertRaises(ValueError):
            generator.correct_gsm_priority(corrected)

    def test_fixture_secrets_redacted(self):
        for text in self.data['files'].values():
            for line in text.splitlines():
                if __import__('re').match(r'(?i)^(?:[^;\n]*(?:password|secret)|TRUNK\w*)\s*=', line):
                    self.assertTrue(line.endswith('<REDACTED>'), line.split('=',1)[0])

    def test_removed_applications(self):
        for app in ('MeetMe', 'Macro', 'Monitor'):
            with self.subTest(app=app):
                data = copy.deepcopy(self.data)
                data['files']['extensions.conf'] += f'\nexten => 9060,1,{app}(test)\n'
                self.assertEqual(contract.validate(data)['state'], 'FAIL')

    def test_exposing_external_legacy_context(self):
        self.data['files']['extensions.conf'] += '\ninclude => vicidial-auto-external\n'
        self.fails('legacy external context exposed')

    def test_missing_conference_profile(self):
        self.data['files']['confbridge-vicidial.conf'] = ''
        self.fails('missing agent bridge')

    def test_conference_range_drift(self):
        self.data['allocation']['count'] = 299
        self.fails('conference allocation')

    def test_repeat_drift(self):
        self.data['repeat_equal'] = False
        self.fails('regeneration drift')

    def test_source_derivation_tampering(self):
        for key in ('source_sha256', 'derivation_sha256'):
            data = copy.deepcopy(self.data)
            data[key] = '0' * 64
            self.assertEqual(contract.validate(data)['state'], 'FAIL')
        with self.assertRaisesRegex(ValueError, 'unreviewed'):
            generator.derive(b'unreviewed source')

    def test_base_configuration_preservation(self):
        self.data['base_changes'].append('http.conf')
        self.fails('unapproved base change')

    def test_native_echo_preservation(self):
        self.data['files']['extensions.conf'] = self.data['files']['extensions.conf'].replace('Echo()', 'Hangup()')
        self.fails('Echo dialplan changed')

    def test_ami_exposure(self):
        self.data['files']['manager.conf'] = self.data['files']['manager.conf'].replace('bindaddr=127.0.0.1', 'bindaddr=0.0.0.0')
        self.fails('AMI listener')

    def test_ami_privilege_escalation(self):
        self.data['files']['manager.conf'] = self.data['files']['manager.conf'].replace('write=system', 'write=all')
        self.fails('AMI permission')

    def test_mariadb_unreadable_configuration(self):
        self.data['permissions']['mariadb_mode'] = '0600'
        self.fails('MariaDB permissions')

    def test_mariadb_effective_networking(self):
        self.data['permissions']['tcp_3306'] = True
        self.fails('MariaDB effective networking')

    def test_nonroot_perl_traversal(self):
        self.data['permissions']['perl_directories'][2] = '0700'
        self.fails('Perl traversal')


if __name__ == '__main__':
    unittest.main()
