package provision

// S05 uses the distribution httpd and php-fpm units. This complete, owned
// configuration is selected through OPTIONS; vendor httpd.conf and conf.d are
// preserved. S07 owns any later TLS or externally reachable web endpoint.
const vicidialHTTPD = `# AlmaDial S05 local application qualification
ServerRoot "/etc/httpd"
Listen 127.0.0.1:80
ServerName localhost
Include conf.modules.d/*.conf
User apache
Group apache
ServerAdmin root@localhost
ServerTokens Prod
ServerSignature Off
TraceEnable Off
ErrorLog "/var/log/httpd/almadial-s05-error.log"
LogLevel warn
PidFile /run/httpd/httpd.pid
DocumentRoot "/var/www/html"
DirectoryIndex index.php index.html
TypesConfig /etc/mime.types
<Directory />
    AllowOverride None
    Require all denied
</Directory>
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
`

const vicidialHTTPDDropIn = `[Service]
Environment="OPTIONS=-f /etc/httpd/conf/almadial-s05.conf"
`

func vicidialWebConfiguration() []vicidialOwnedFile {
	return []vicidialOwnedFile{
		{"/etc/httpd/conf/almadial-s05.conf", "root", "root", "httpd_config_t", 0644, []byte(vicidialHTTPD)},
		{"/etc/systemd/system/httpd.service.d/almadial-s05.conf", "root", "root", "systemd_unit_file_t", 0644, []byte(vicidialHTTPDDropIn)},
	}
}
