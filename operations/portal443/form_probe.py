"""Real Apache -> backend cookie/CSRF smoke test using deliberately fake credentials."""
import http.client
import re
import ssl
from urllib.parse import urlencode

IP = '2.29.153.161'
def request(path, body=None, cookie=None):
    c = http.client.HTTPSConnection(IP, context=ssl.create_default_context(), timeout=10,
                                    source_address=(IP, 0))
    headers = {'Origin': 'https://' + IP}
    if cookie:
        headers['Cookie'] = cookie
    if body is not None:
        headers['Content-Type'] = 'application/x-www-form-urlencoded; charset=UTF-8'
    c.request('GET' if body is None else 'POST', path, body, headers)
    r = c.getresponse()
    result = r.status, r.getheader('Set-Cookie'), r.read().decode()
    c.close()
    return result

status, cookie, body = request('/portal/')
assert status == 200
csrf = re.search('name="csrf" value="([a-f0-9]+)"', body)[1]
status, _, _ = request('/portal/login', urlencode({'csrf': csrf, 'username': 'nonexistent-test-account',
                                                   'password': 'fake-test-password-only'}), cookie.split(';')[0])
assert status == 401, ('Expected credential rejection after passing CSRF', status)
print('PASS real HTTPS form/session/CSRF transport; fake credentials rejected')
