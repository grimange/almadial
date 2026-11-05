"""Read-only supervisor phone eligibility, matching native blind_monitor checks.

VICIdial non_agent_api.php:3979-3995 requires the function allowlist and
vdc_agent_api_access=1, active account, level>6. Portal supervisors are levels8/9.
Native VICIdial still controls which actual monitor/barge actions are allowed.
"""
from vicidial_auth import _connect, _role_from_row, USER_PATTERN


def monitor_allowed(row):
    if _role_from_row(row) != 'web':
        return False
    functions = str(row.get('api_allowed_functions', ''))
    return str(row.get('vdc_agent_api_access', '')) == '1' and (
        ' blind_monitor ' in functions or 'ALL_FUNCTIONS' in functions.split())


def supervisor_phone(username):
    if not isinstance(username, str) or not USER_PATTERN.fullmatch(username):
        return False
    conn = _connect()
    try:
        with conn.cursor() as cursor:
            cursor.execute('SELECT active,user_level,api_only_user,force_change_password,'
                           'vdc_agent_api_access,api_allowed_functions FROM vicidial_users '
                           'WHERE user=%s LIMIT 1', (username,))
            return monitor_allowed(cursor.fetchone())
    finally:
        conn.close()
