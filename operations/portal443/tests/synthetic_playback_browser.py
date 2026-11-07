"""Loopback-only actual-handler browser fixture; all credentials/media synthetic."""
import base64
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from test_recording_media import Playback, NAME, PATH, portal, media

fixture = Playback()
fixture.setUp()
media.ORIGIN = 'http://127.0.0.1:18766'
fixture.records[0]['location'] = media.ORIGIN + '/RECORDINGS/' + NAME
TOKEN = 'Basic ' + base64.b64encode(b'synthetic:synthetic-pass').decode()


class Handler(portal.Handler):
    def checks(self):
        # Synthetic proxy context only. Production checks remain unmodified.
        self.headers.replace_header('Host', portal.PUBLIC_IP)
        self.headers['X-Alma-Source'] = '8.8.8.8'
        return super().checks()

    def do_GET(self):
        if self.path == '/vicidial/admin.php':
            if self.headers.get('Authorization') != TOKEN:
                self.send_response(401)
                self.send_header('WWW-Authenticate', 'Basic realm="CONTACT-CENTER-ADMIN"')
                self.send_header('Content-Length', '0')
                self.end_headers()
                return
            body = ('<!doctype html><title>Synthetic supervisor</title><a id="recording" href="/RECORDINGS/'
                    + NAME + '">Recording</a><audio controls id="audio" preload="none" src="/RECORDINGS/'
                    + NAME + '"></audio>').encode()
            self.send_response(200)
            self.send_header('Content-Type', 'text/html')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        super().do_GET()


if __name__ == '__main__':
    try:
        portal.Server(('127.0.0.1', 18766), Handler).serve_forever()
    finally:
        fixture.tearDown()
