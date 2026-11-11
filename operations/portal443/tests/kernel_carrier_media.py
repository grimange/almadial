#!/usr/bin/python3
"""Opt-in real nft test: run ONLY with unshare -n on an empty network namespace.

Uses real nftables and temporary JSON, but mocks firewalld ownership storage.
Never run destructive lifecycle tests in the host network namespace.
"""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import carrier_cli as cli
import sip_guard as guard


def main():
    if os.readlink('/proc/self/ns/net') == os.readlink('/proc/1/ns/net'):
        raise RuntimeError('Refusing host network namespace')
    if cli.Firewall().command('/usr/sbin/nft', 'list', 'tables'):
        raise RuntimeError('Requires empty disposable network namespace')
    spec = importlib.util.spec_from_file_location('installer', Path(cli.__file__).with_name('install-carriers.py'))
    installer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(installer)
    with tempfile.TemporaryDirectory() as directory:
        config = Path(directory) / 'carriers.json'
        config.write_text('["38.147.130.74"]\n')
        guard.CARRIER_CONFIG = config
        # Reconstruct the submitted pre-media policy from the pinned source.
        spec = importlib.util.spec_from_file_location('old_guard', Path(cli.__file__).with_name('old_sip_guard.py'))
        old = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(old)
        old.CARRIER_CONFIG = config
        old.bootstrap()
        fw = cli.Firewall()
        before = fw.policy_snapshot()
        result = guard.nft('-f', '-', script='add element inet alma_portal_sip agents4 { 8.8.4.4 timeout 300s }\nadd element inet alma_portal_sip grace4 { 8.8.4.4 timeout 420s }')
        assert result.returncode == 0, result.stderr
        def dynamic():
            objects = json.loads(fw.command('/usr/sbin/nft', '-j', 'list', 'table', 'inet', guard.TABLE))['nftables']
            return {x['set']['name']: x['set'] for x in objects if x.get('set', {}).get('name') in ('agents4', 'grace4')}
        agents_before = dynamic()
        def checked(*args, script=None):
            if args[:2] == ('-f', '-'):
                result = guard.nft('-c', *args, script=script)
                assert result.returncode == 0, result.stderr
            return guard.nft(*args, script=script)
        guard.bootstrap(checked)
        assert fw.nft() == ({'38.147.130.74'}, True)
        media = cli.Firewall(True)
        assert media.nft() == (set(), True)
        agents_after = dynamic()
        for key in agents_before:
            assert agents_before[key]['handle'] == agents_after[key]['handle']
            a, b = agents_before[key]['elem'][0]['elem'], agents_after[key]['elem'][0]['elem']
            assert a['val'] == b['val'] and a['timeout'] == b['timeout']
            assert 0 <= a['expires'] - b['expires'] < 5000
        print('PASS real kernel: old-policy migration, nft -c, dynamic set handles/deadlines preserved')

        class KernelFirewall(cli.Firewall):
            rules = {False: {cli.rule('38.147.130.74'), 'unrelated'}, True: {cli.rule('38.147.130.74')}}
            def rich(self, permanent=False):
                return set(self.rules[permanent])
            def set_rich(self, desired, permanent=False):
                self.rules[permanent] -= self.managed(permanent)
                self.rules[permanent] |= desired
        media = KernelFirewall(True)
        sip = KernelFirewall()
        for op, ip in [('add', '8.8.8.8'), ('add', '8.8.8.8'), ('sync', None), ('remove', '8.8.8.8')]:
            assert cli.execute(op, ip, config, media, True) == 0
        assert cli.execute('validate', path=config, firewall=sip) == 0
        assert media.nft() == (set(), True)
        print('PASS real kernel: media add/duplicate/sync/remove; simulated firewalld; SIP preserved')
        # Restore legacy array because it is the desired state for old source.
        config.write_text('["38.147.130.74"]\n')
        fw.restore_policy(before)  # Calls actual nft -c and JSON netlink transaction.
        assert installer.policy_content(fw.policy_snapshot()) == installer.policy_content(before)
        assert dynamic()['agents4']['handle'] == agents_before['agents4']['handle']
        print('PASS real kernel: scoped old-policy rollback; media set removed; agents preserved')
        # Fresh-table/bootstrap persistence with configured media, only inside namespace.
        result = guard.nft('delete', 'table', 'inet', guard.TABLE)
        assert result.returncode == 0
        config.write_text(json.dumps({'sip': ['38.147.130.74'], 'media': ['8.8.8.8']}))
        guard.bootstrap(checked)
        snapshot = fw.policy_snapshot()
        guard.bootstrap(checked)
        assert fw.policy_snapshot() == snapshot
        assert cli.Firewall(True).nft() == ({'8.8.8.8'}, True)
        print('PASS real kernel: fresh template with media; repeated bootstrap unchanged')
    print('All isolated kernel checks passed. Namespace destroyed on exit; host firewall never used.')


if __name__ == '__main__':
    main()
