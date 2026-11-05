require 'apache2'

local public = {
    ['/'] = true, ['/portal/'] = true, ['/portal/login'] = true,
    ['/portal/grant'] = true, ['/portal/status'] = true,
    ['/portal/continue'] = true, ['/portal/logout'] = true,
    ['/portal/app.js'] = true
}

function authorize(r)
    local host = r.headers_in['Host'] or ''
    if host ~= '2.29.153.161' and host ~= '2.29.153.161:443' then
        return apache2.AUTHZ_DENIED
    end
    -- Never allow dot files, even when another Location overrides Files authz.
    if string.find(r.uri, '/%.') then return apache2.AUTHZ_DENIED end
    if public[r.uri] then return apache2.AUTHZ_GRANTED end
    local ip = r.useragent_ip or ''
    if not string.match(ip, '^%d+%.%d+%.%d+%.%d+$') or #ip > 15 then
        return apache2.AUTHZ_DENIED
    end
    -- Prevent even a future RemoteIPHeader change from trusting a spoofed source.
    if not r:expr("%{CONN_REMOTE_ADDR} == '" .. ip .. "'") then
        return apache2.AUTHZ_DENIED
    end
    local file = io.open('/var/lib/alma-portal-443/web/' .. ip, 'r')
    if not file then return apache2.AUTHZ_DENIED end
    local expiry = tonumber(file:read(32))
    file:close()
    if expiry and expiry > os.time() then return apache2.AUTHZ_GRANTED end
    return apache2.AUTHZ_DENIED
end
