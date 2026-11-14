#!/usr/bin/env python3
"""Explicitly scoped private-lab HTTPS static artifact server (ALM-34).

No directory listing, uploads, execution, redirects or automatic publication.
Keep the root restricted to approved CLI releases and retained source objects.
"""
import argparse
from functools import partial
import http.server
import ipaddress
from pathlib import Path
import ssl

class FilesOnly(http.server.SimpleHTTPRequestHandler):
    def list_directory(self, path):
        self.send_error(403, 'Directory listing disabled')
        return None

    def send_head(self):
        target = Path(self.translate_path(self.path))
        if not target.is_file() or target.is_symlink():
            self.send_error(404)
            return None
        return super().send_head()

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--root',type=Path,required=True)
    p.add_argument('--certificate',type=Path,required=True)
    p.add_argument('--key',type=Path,required=True)
    p.add_argument('--bind',required=True)
    p.add_argument('--port',type=int,default=8447)
    a=p.parse_args()
    address=ipaddress.ip_address(a.bind)
    if not address.is_private or address.is_unspecified or address.is_loopback or address.version!=4:
        p.error('explicit authorized private IPv4 interface required')
    root=a.root.resolve(strict=True)
    if not root.is_dir() or root.stat().st_mode & 0o077:
        p.error('private distribution root required')
    context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version=ssl.TLSVersion.TLSv1_2
    context.load_cert_chain(a.certificate,a.key)
    server=http.server.ThreadingHTTPServer((a.bind,a.port),partial(FilesOnly,directory=str(root)))
    server.socket=context.wrap_socket(server.socket,server_side=True)
    server.serve_forever()

if __name__=='__main__':
    main()
