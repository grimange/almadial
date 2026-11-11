import socketserver,subprocess,threading,sys
REMOTE="""import socket,sys,threading
s=socket.create_connection(('127.0.0.1',int(sys.argv[1])))
def send():
 try:
  while True:
   b=sys.stdin.buffer.read1(65536)
   if not b:break
   s.sendall(b)
 except OSError:pass
 finally:
  try:s.shutdown(socket.SHUT_WR)
  except OSError:pass
threading.Thread(target=send,daemon=True).start()
try:
 while True:
  b=s.recv(65536)
  if not b:break
  sys.stdout.buffer.write(b);sys.stdout.buffer.flush()
except OSError:pass
"""
class Handler(socketserver.BaseRequestHandler):
 def handle(self):
  p=subprocess.Popen(['multipass','exec','almadial-alm16-rpm','--','python3','-u','-c',REMOTE,str(self.server.remote)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
  def send():
   try:
    while True:
     b=self.request.recv(65536)
     if not b:break
     p.stdin.write(b);p.stdin.flush()
   except (OSError,BrokenPipeError):pass
   finally:
    try:p.stdin.close()
    except OSError:pass
  threading.Thread(target=send,daemon=True).start()
  try:
   while True:
    b=p.stdout.read1(65536)
    if not b:break
    self.request.sendall(b)
  except OSError:pass
  finally:p.terminate()
class Server(socketserver.ThreadingTCPServer):
 allow_reuse_address=True
 daemon_threads=True
https_port=8443 if sys.argv[1:]==['--agent-fixture'] else 443
assert not sys.argv[1:] or sys.argv[1:]==['--agent-fixture']
for local,remote in [(8443,https_port),(8089,8089)]:
 s=Server(('127.0.0.1',local),Handler);s.remote=remote
 threading.Thread(target=s.serve_forever,daemon=True).start()
print('loopback forwards ready',flush=True)
threading.Event().wait()
