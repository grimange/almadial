require 'apache2'

local public = {
    ['/'] = true, ['/portal/'] = true, ['/portal/login'] = true,
    ['/portal/grant'] = true, ['/portal/status'] = true,
    ['/portal/continue'] = true, ['/portal/logout'] = true,
    ['/portal/app.js'] = true, ['/portal/app.css'] = true,
    ['/portal/logo.png'] = true
}

local function valid_host(r)
    local host = r.headers_in['Host'] or ''
    return host == '2.29.153.161' or host == '2.29.153.161:443'
end

local function active(r)
    local ip = r.useragent_ip or ''
    if not string.match(ip, '^%d+%.%d+%.%d+%.%d+$') or #ip > 15 then
        return false
    end
    -- Prevent even a future RemoteIPHeader change from trusting a spoofed source.
    if not r:expr("%{CONN_REMOTE_ADDR} == '" .. ip .. "'") then
        return false
    end
    local file = io.open('/var/lib/alma-portal-443/web/' .. ip, 'r')
    if not file then return false end
    local expiry = tonumber(file:read(32))
    file:close()
    return expiry and expiry > os.time()
end

function authorize(r)
    if not valid_host(r) or string.find(r.uri, '/%.') then return apache2.AUTHZ_DENIED end
    if public[r.uri] or active(r) then return apache2.AUTHZ_GRANTED end
    return apache2.AUTHZ_DENIED
end

function browser_redirect(r)
    -- This hook changes only the denial UX; the authz provider still gates
    -- every protected resource. Headers never confer authorization.
    if not valid_host(r) or r.method ~= 'GET' then return apache2.DECLINED end
    if r.uri ~= '/agc/vicidial.php' and r.uri ~= '/vicidial/admin.php' then
        return apache2.DECLINED
    end
    local mode = r.headers_in['Sec-Fetch-Mode']
    local dest = r.headers_in['Sec-Fetch-Dest']
    local accept = r.headers_in['Accept'] or ''
    if r.headers_in['X-Requested-With'] or not string.find(accept, 'text/html', 1, true) then
        return apache2.DECLINED
    end
    if (mode and mode ~= 'navigate') or (dest and dest ~= 'document') then
        return apache2.DECLINED
    end
    if active(r) then return apache2.DECLINED end
    r.headers_out['Location'] = 'https://2.29.153.161/'
    r.err_headers_out['Cache-Control'] = 'no-store, private'
    -- Fixed UX hint only; never used to authenticate or authorize.
    r.err_headers_out['Set-Cookie'] = '__Secure-AlmaNotice=required; Path=/portal/; Max-Age=120; Secure; HttpOnly; SameSite=Strict'
    return 303
end
