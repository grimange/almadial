#!/usr/bin/python3
"""Existing Certbot + Apache integration; no certificate/key files are written here."""
import argparse
import contextlib
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import sys
import time
import uuid

STATE = Path('/var/lib/alma-portal-tls')
CERTBOT = '/opt/alma-portal-certbot/bin/certbot'
HOOK = '/usr/local/sbin/alma-portal-cert-deploy'
IP_FILE = Path('/etc/alma-portal/public-ip')
SYSTEMCTL = '/usr/bin/systemctl'
HTTPD = '/usr/sbin/httpd'
OPENSSL = '/usr/bin/openssl'


class Failure(Exception):
    pass


def event(stage, **fields):
    print(json.dumps(dict(stage=stage, time=int(time.time()), **fields)), flush=True)


def command(stage, argv, *, timeout=30, env=None, passthrough=False):
    event(stage, action='start')
    try:
        p = subprocess.run(argv, timeout=timeout, env=env, text=True,
                           capture_output=not passthrough)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Failure(stage + ': ' + type(exc).__name__) from exc
    if p.returncode:
        # Do not export command output, configuration contents or secret material.
        raise Failure(stage + ': exit ' + str(p.returncode))
    event(stage, action='ok')
    return p.stdout or ''


def read_state(name):
    try:
        return json.loads((STATE / name).read_text())
    except FileNotFoundError:
        return {}


def save_state(name, data):
    STATE.mkdir(mode=0o700, parents=True, exist_ok=True)
    temporary = STATE / (name + '.' + uuid.uuid4().hex)
    try:
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as f:
            json.dump(data, f, sort_keys=True)
            f.write('\n')
            f.flush()
            os.fsync(f.fileno())
        os.replace(temporary, STATE / name)
    finally:
        temporary.unlink(missing_ok=True)


@contextlib.contextmanager
def lock(name):
    STATE.mkdir(mode=0o700, parents=True, exist_ok=True)
    with (STATE / name).open('a') as f:
        try:
            fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise Failure('lock: operation already running') from exc
        yield


def identity():
    address = str(ipaddress.IPv4Address(IP_FILE.read_text().strip()))
    return address, Path('/etc/letsencrypt/live') / address


def certificate(address, lineage, min_seconds=0):
    cert = lineage / 'cert.pem'
    command('certificate_chain', [OPENSSL, 'verify', '-purpose', 'sslserver',
            '-verify_ip', address, '-CAfile', '/etc/pki/tls/certs/ca-bundle.crt',
            '-untrusted', str(lineage / 'chain.pem'), str(cert)])
    command('certificate_lifetime', [OPENSSL, 'x509', '-in', str(cert),
            '-noout', '-checkend', str(min_seconds)])
    # Verify that the actual Apache certificate and private key match, in memory.
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(str(lineage / 'fullchain.pem'), str(lineage / 'privkey.pem'))
    dates = command('certificate_dates', [OPENSSL, 'x509', '-in', str(cert),
                    '-noout', '-startdate', '-enddate', '-serial'])
    return dict(fingerprint=disk_fingerprint(lineage),
                **dict(line.split('=', 1) for line in dates.splitlines()))


def disk_fingerprint(lineage):
    return hashlib.sha256(ssl.PEM_cert_to_DER_cert((lineage / 'cert.pem').read_text())).hexdigest()


def served(address):
    context = ssl.create_default_context()
    with socket.create_connection(('127.0.0.1', 443), timeout=5) as raw:
        with context.wrap_socket(raw, server_hostname=address) as connection:
            return hashlib.sha256(connection.getpeercert(binary_form=True)).hexdigest()


def wait_served(address, expected):
    for attempt in range(20):
        try:
            if served(address) == expected:
                return
        except (OSError, ssl.SSLError):
            pass
        if attempt < 19:
            time.sleep(0.5)
    raise Failure('served_certificate: trusted IP certificate does not match disk')


def deploy():
    with lock('deploy.lock'):
        state = dict(operation='deploy', outcome='running', started_at=int(time.time()),
                     run_id=os.environ.get('ALMA_TLS_RUN_ID'))
        save_state('deployment.json', state)
        try:
            address, lineage = identity()
            renewed = os.environ.get('RENEWED_LINEAGE')
            if renewed and Path(renewed) != lineage:
                raise Failure('lineage: unexpected renewal target')
            info = certificate(address, lineage)
            command('apache_config', [HTTPD, '-t'])
            command('apache_active', [SYSTEMCTL, 'is-active', '--quiet', 'httpd.service'])
            command('apache_reload', [SYSTEMCTL, 'reload', 'httpd.service'])
            wait_served(address, info['fingerprint'])
            state.update(outcome='ok', certificate=info)
            event('deployment_verified', fingerprint=info['fingerprint'])
        except Exception as exc:
            state.update(outcome='failed', error=str(exc) if isinstance(exc, Failure) else type(exc).__name__)
            raise
        finally:
            state['finished_at'] = int(time.time())
            save_state('deployment.json', state)


def renew(dry_run=False):
    with lock('renew.lock'):
        previous = read_state('status.json')
        state = dict(operation='renew', outcome='running', dry_run=dry_run,
                     run_id=uuid.uuid4().hex, started_at=int(time.time()),
                     last_success_at=previous.get('last_success_at'))
        save_state('status.json', state)
        try:
            address, lineage = identity()
            # An expired current certificate must not prevent Certbot repairing it.
            before = disk_fingerprint(lineage)
            env = dict(os.environ, ALMA_TLS_RUN_ID=state['run_id'])
            argv = [CERTBOT, 'renew', '--non-interactive', '--cert-name', address,
                    '--no-random-sleep-on-renew', '--deploy-hook', HOOK]
            if dry_run:
                argv += ['--dry-run', '--run-deploy-hooks']
            command('certbot_dry_run' if dry_run else 'certbot_renew', argv,
                    timeout=600, env=env, passthrough=True)
            deployment = read_state('deployment.json')
            if deployment.get('run_id') == state['run_id'] and deployment.get('outcome') != 'ok':
                raise Failure('deployment: hook failed despite Certbot return status')
            after = certificate(address, lineage, min_seconds=86400)
            # Retry a previously failed/missed deployment on a later no-op renewal.
            try:
                matches = served(address) == after['fingerprint']
            except (OSError, ssl.SSLError):
                matches = False
            if deployment.get('outcome') == 'failed' or not matches:
                command('deployment_recovery', [HOOK], env=env, passthrough=True)
                deployment = read_state('deployment.json')
            deployed = deployment.get('run_id') == state['run_id'] and deployment.get('outcome') == 'ok'
            if (dry_run or before != after['fingerprint']) and not deployed:
                raise Failure('deployment: successful deploy-hook receipt missing')
            wait_served(address, after['fingerprint'])
            state.update(outcome='ok', certificate=after, deployed=deployed,
                         certificate_changed=before != after['fingerprint'],
                         last_success_at=int(time.time()))
            event('renewal_verified', dry_run=dry_run, deployed=deployed,
                  certificate_changed=state['certificate_changed'])
        except Exception as exc:
            state.update(outcome='failed', error=str(exc) if isinstance(exc, Failure) else type(exc).__name__)
            raise
        finally:
            state['finished_at'] = int(time.time())
            save_state('status.json', state)


def check():
    state = read_state('status.json')
    if state.get('outcome') == 'failed' or read_state('deployment.json').get('outcome') == 'failed':
        raise Failure('health: renewal or deployment has a persistent failure')
    if time.time() - state.get('last_success_at', 0) > 30 * 3600:
        raise Failure('health: no successful check within 30 hours')
    if state.get('outcome') == 'running' and time.time() - state['started_at'] > 720:
        raise Failure('health: renewal check stalled')
    address, lineage = identity()
    info = certificate(address, lineage, min_seconds=86400)
    wait_served(address, info['fingerprint'])
    command('timer_enabled', [SYSTEMCTL, 'is-enabled', '--quiet', 'alma-portal-renew.timer'])
    command('timer_active', [SYSTEMCTL, 'is-active', '--quiet', 'alma-portal-renew.timer'])
    event('health_ok', certificate=info, last_success_at=state['last_success_at'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=['renew', 'deploy', 'check'])
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()
    if args.dry_run and args.operation != 'renew':
        parser.error('--dry-run applies only to renew')
    if os.geteuid() != 0:
        parser.error('root is required to read protected certificate state')
    os.umask(0o077)
    try:
        if args.operation == 'renew':
            renew(args.dry_run)
        elif args.operation == 'deploy':
            deploy()
        else:
            check()
    except Exception as exc:
        event('failure', error=str(exc) if isinstance(exc, Failure) else type(exc).__name__)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
