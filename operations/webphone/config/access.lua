-- PRD-WEBPHONE-00: read the existing portal worker's phone grant projection.
require 'apache2'
function authorize(r)
    local host = r.headers_in['Host'] or ''
    local ip = r.useragent_ip or ''
    if (host ~= '2.29.153.161' and host ~= '2.29.153.161:443') or
       r.unparsed_uri ~= '/ws' or not string.match(ip, '^%d+%.%d+%.%d+%.%d+$') or #ip > 15 or
       not r:expr("%{CONN_REMOTE_ADDR} == '" .. ip .. "'") then
        return apache2.AUTHZ_DENIED
    end
    local file = io.open('/var/lib/alma-portal-443/phone-websocket/' .. ip, 'r')
    if not file then return apache2.AUTHZ_DENIED end
    local expiry = tonumber(file:read(32)); file:close()
    if expiry and expiry > os.time() then return apache2.AUTHZ_GRANTED end
    return apache2.AUTHZ_DENIED
end
