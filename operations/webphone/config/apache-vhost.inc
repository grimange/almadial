# Include inside the existing HTTPS VirtualHost, after its portal LocationMatch.
LuaAuthzProvider webphone00 /opt/alma-webphone-00/access.lua authorize
ProxyPassMatch "^(/ws)$" "ws://127.0.0.1:8088$1" connectiontimeout=5 timeout=120
<LocationMatch "^/ws$">
    <RequireAll>
        Require portal443
        Require webphone00
    </RequireAll>
    RequestHeader unset Authorization
    RequestHeader unset Proxy-Authorization
</LocationMatch>
<LocationMatch "^/viciphone-web00/debug(?:/|$)">
    Require all denied
</LocationMatch>
