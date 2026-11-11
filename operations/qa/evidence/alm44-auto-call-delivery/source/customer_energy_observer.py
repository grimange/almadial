# Capture-specific passive diagnostic. Adjust the tuple only after verifying the live SIP1001 channel.
# Cleartext PCMU only: never decode encrypted WebRTC/SRTP as audio. No payload is stored.
"""Bounded passive RTP metrics; discards payload, emits no audio or credentials."""
import socket,select,struct,math,time,json,datetime,re,subprocess
peer=subprocess.run(['asterisk','-rx','sip show peer 1001'],capture_output=True,text=True,check=True).stdout
host=re.search(r'Addr->IP\s*:\s*(\d+\.\d+\.\d+\.\d+)',peer).group(1)
s=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));s.bind(('eth0',0));s.setblocking(False)
end=time.monotonic()+15;last=time.monotonic();stats={}
def linear(v):
 u=(~v)&255;n=((u&15)<<3)+132;n<<=(u&112)>>4
 return (132-n) if (u&128) else (n-132)
table=[linear(v) for v in range(256)]
print(json.dumps({'observer':'started','scope':'SIP1001 host inbound/outbound RTP metadata and PCMU energy only','utc':datetime.datetime.now(datetime.timezone.utc).isoformat()}),flush=True)
while time.monotonic()<end:
 if select.select([s],[],[],0.2)[0]:
  p=s.recv(65535)
  if len(p)<42 or p[12:14]!=b'\x08\x00' or p[23]!=17:continue
  h=(p[14]&15)*4;src=socket.inet_ntoa(p[26:30]);dst=socket.inet_ntoa(p[30:34])
  if host not in [src,dst]:continue
  off=14+h;sp,dp,ln=struct.unpack('!HHH',p[off:off+6]);data=p[off+8:off+ln]
  if (sp,dp) not in [(19582,25201),(25201,19582)]:continue
  if len(data)<12 or data[0]>>6!=2 or not(10000<=sp<=20000 or 10000<=dp<=20000):continue
  pt=data[1]&127;head=12+4*(data[0]&15)
  if data[0]&16:
   if len(data)<head+4:continue
   head+=4+4*int.from_bytes(data[head+2:head+4],'big')
  payload=data[head:];key=('incoming' if src==host else 'outgoing',sp,dp,pt)
  d=stats.setdefault(key,{'packets':0,'samples':0,'squares':0,'peak':0})
  d['packets']+=1
  if pt==0:
   values=[table[x] for x in payload];d['samples']+=len(values);d['squares']+=sum(v*v for v in values);d['peak']=max(d['peak'],max(map(abs,values),default=0))
 if time.monotonic()-last>=2:
  for (direction,sp,dp,pt),d in stats.items():
   print(json.dumps({'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'direction':direction,'source_port':sp,'dest_port':dp,'payload_type':pt,'packets':d['packets'],'rms':round(math.sqrt(d['squares']/d['samples']),3) if d['samples'] else None,'peak':d['peak']}),flush=True)
  stats={};last=time.monotonic()
s.close();print(json.dumps({'observer':'ended'}),flush=True)
