package provision

import "errors"

var errWeb = errors.New("S07 web contract failed; preserve state and inspect the blocking web postcondition")

// The accepted S05 main configuration includes conf.modules.d, but not conf.d.
// This exact owned loader leaves vendor sites and the S05 files untouched.
const webLoader = "# AlmaDial S07 application virtual host loader\nInclude /etc/httpd/almadial-s07.conf\n"

// TLS-ready ownership only: S08 must supply certificates and a reviewed listener.
// S07 retains loopback HTTP and the existing Unix-socket PHP-FPM integration.
func WebVhost() string {
	return `# AlmaDial S07: local application delivery; TLS belongs to S08
<VirtualHost 127.0.0.1:80>
    ServerName localhost
    DocumentRoot "/var/www/html"
    DirectoryIndex index.php index.html
    Header always set X-AlmaDial-Web "S07"
    <Directory "/var/www/html">
        Options None
        AllowOverride None
        Require local
    </Directory>
    <FilesMatch "^(\.ht|\.user\.ini$|options\.php$)">
        Require all denied
    </FilesMatch>
    SetEnvIfNoCase ^Authorization$ "(.+)" HTTP_AUTHORIZATION=$1
    <FilesMatch "\.php$">
        SetHandler "proxy:unix:/run/php-fpm/www.sock|fcgi://localhost"
    </FilesMatch>
</VirtualHost>
`
}
