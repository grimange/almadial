#!/usr/bin/env python3
"""Read-only VICIdial user verification, independent of VICIdial PHP.

Implements the documented legacy VICIdial pass_hash scheme: the bcrypt
configuration uses system_settings.pass_key/pass_cost; only the 31-byte
bcrypt hash suffix is stored in vicidial_users.pass_hash.
"""
import configparser
import hmac
from pathlib import Path
import re

import bcrypt
import pymysql

CONFIG = Path('/etc/alma-portal/vicidial-db.ini')
ALPHABET = './ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
USER_PATTERN = re.compile(r'^[A-Za-z0-9._-]{2,64}$')


class AuthConfigurationError(Exception):
    pass


def _config():
    p = configparser.ConfigParser(interpolation=None)
    if not p.read(CONFIG) or 'db' not in p:
        raise AuthConfigurationError('Missing private VICIdial database config')
    return p['db']


def _connect():
    cfg = _config()
    return pymysql.connect(host='127.0.0.1', port=3306,
                           user=cfg['user'], password=cfg['password'],
                           database=cfg.get('database', 'asterisk'),
                           charset='utf8mb4', connect_timeout=3, read_timeout=3,
                           write_timeout=3, autocommit=True,
                           cursorclass=pymysql.cursors.DictCursor)


def _bcrypt_base64(raw):
    """Crypt::Eksblowfish::Bcrypt::en_base64 equivalent for 16 bytes."""
    if len(raw) != 16:
        raise AuthConfigurationError('Invalid VICIdial pass_key length')
    output = []
    i = 0
    while i < len(raw):
        c1 = raw[i]
        i += 1
        output.append(ALPHABET[c1 >> 2])
        c1 = (c1 & 3) << 4
        if i >= len(raw):
            output.append(ALPHABET[c1]); break
        c2 = raw[i]
        i += 1
        c1 |= (c2 >> 4) & 15
        output.append(ALPHABET[c1])
        c1 = (c2 & 15) << 2
        if i >= len(raw):
            output.append(ALPHABET[c1]); break
        c2 = raw[i]
        i += 1
        c1 |= (c2 >> 6) & 3
        output.append(ALPHABET[c1])
        output.append(ALPHABET[c2 & 63])
    result = ''.join(output)
    if len(result) != 22:
        raise AuthConfigurationError('Invalid bcrypt salt encoding')
    return result


def _role_from_row(row):
    if row is None or str(row.get('active', '')).upper() != 'Y':
        return None
    try:
        level = int(row['user_level'])
    except (TypeError, ValueError):
        return None
    if not (1 <= level <= 9):
        return None
    # Service-only accounts must not be able to open the public network gate.
    api_only = str(row.get('api_only_user', '')).upper()
    if api_only not in ('0', 'N', 'NO', ''):
        return None
    force_change = str(row.get('force_change_password', '')).upper()
    if force_change in ('Y', 'YES', '1'):
        return None
    return 'web' if level >= 8 else 'agent'


def _find_user(cursor, username, credentials):
    cols = 'user,active,user_level,api_only_user,force_change_password'
    if credentials:
        cols += ',pass,pass_hash'
    cursor.execute('SELECT ' + cols + ' FROM vicidial_users WHERE user=%s LIMIT 1', (username,))
    return cursor.fetchone()


def current_role(username):
    """Recheck active user / eligibility at grant application time."""
    if not isinstance(username, str) or not USER_PATTERN.fullmatch(username):
        return None
    conn = _connect()
    try:
        with conn.cursor() as cursor:
            return _role_from_row(_find_user(cursor, username, False))
    finally:
        conn.close()


def authenticate(username, password):
    """Return web/agent after verifying VICIdial password; otherwise None.
    Fail closed on missing schema, unsupported modes, or unreachable database.
    """
    if (not isinstance(username, str) or not USER_PATTERN.fullmatch(username)
        or not isinstance(password, str) or not 1 <= len(password) <= 256):
        return None
    conn = _connect()
    try:
        with conn.cursor() as cursor:
            row = _find_user(cursor, username, True)
            role = _role_from_row(row)
            if role is None:
                return None
            cursor.execute('SELECT pass_hash_enabled,pass_key,pass_cost FROM system_settings LIMIT 1')
            settings = cursor.fetchone()
            if settings is None:
                raise AuthConfigurationError('VICIdial system_settings is empty')
            return role if verify_password(row, settings, password) else None
    finally:
        conn.close()


def verify_password(row, settings, password):
    """Shared existing plaintext / native bcrypt-suffix verifier; no persistence."""
    mode = str(settings['pass_hash_enabled']).upper()
    if mode in ('N', 'NO', '0', 'DISABLED', ''):
        stored = row.get('pass')
        if not isinstance(stored, str):
            return None
        return hmac.compare_digest(password.encode('utf-8'), stored.encode('utf-8'))
    if mode not in ('Y', 'YES', '1', 'ENABLED'):
        raise AuthConfigurationError('Unknown VICIdial pass_hash_enabled value')
    stored_hash = row.get('pass_hash')
    if not isinstance(stored_hash, str) or not re.fullmatch(r'[./A-Za-z0-9]{31}', stored_hash):
        return None
    try:
        cost = int(settings['pass_cost'])
    except (TypeError, ValueError) as e:
        raise AuthConfigurationError('Invalid VICIdial bcrypt cost') from e
    if not 4 <= cost <= 15:
        raise AuthConfigurationError('Unsupported or excessive VICIdial bcrypt cost')
    key = settings['pass_key']
    if isinstance(key, bytes):
        key = key.decode('utf-8')
    if not isinstance(key, str):
        raise AuthConfigurationError('Invalid VICIdial bcrypt salt')
    if len(key) == 16:
        salt = _bcrypt_base64(key.encode('utf-8'))
    elif len(key) == 22 and re.fullmatch(r'[./A-Za-z0-9]{22}', key):
        salt = key
    else:
        raise AuthConfigurationError('Unsupported VICIdial pass_key encoding')
    setting = f'$2a${cost:02d}${salt}'.encode('ascii')
    actual = bcrypt.hashpw(password.encode('utf-8'), setting)
    return hmac.compare_digest(actual[29:60], stored_hash.encode('ascii'))
