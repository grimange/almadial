// Actual Chromium, normal VICIphone controls, synthetic PCM microphone input.
// No mocked SIP/RTP, TLS bypass, hidden call controls or exported auth headers.
const fs = require('fs'), cp = require('child_process');
const {chromium} = require('/home/ra/.npm/_npx/51691537fc71f2b0/node_modules/playwright');
const VM = process.argv[3] || 'almadial-alm16-rpm';
const remote = process.argv[4] ? JSON.parse(fs.readFileSync(process.argv[4], 'utf8')) : null;
if (remote && (VM !== 'almadial-alm46-remote-dev' || remote.machine !== '436ca8ae883439638507ab64b02d98e8' || !/^[a-z0-9.-]+$/.test(remote.hostname) || !/^([0-9]{1,3}\.){3}[0-9]{1,3}$/.test(remote.address))) throw Error('invalid controlled remote target');
if (remote && cp.execFileSync('/snap/bin/multipass',['exec',VM,'--','cat','/etc/machine-id'],{encoding:'utf8'}).trim() !== remote.machine) throw Error('target identity changed');
if (process.argv[3] && !remote && VM !== 'almadial-alm34-bootstrap-dev' && !/^almadial-alm29-clean-[a-z]+$/.test(VM)) throw Error('invalid target VM');
const E = process.argv[2], contexts = [], pages = [];
const result = {pass:false, trust_bypass:false, topology:remote?'controlled-off-host-no-tunnel':'local-LAN', participants:[], sip:[]};
function native(command) {return cp.execFileSync('/snap/bin/multipass',['exec',VM,'--','sudo','/usr/sbin/asterisk','-rx',command],{encoding:'utf8',timeout:20000});}
(async () => {
 try {
  const secret = cp.execFileSync('/snap/bin/multipass',['exec',VM,'--','sudo','cat','/etc/almadial/secrets/phone-s10'],{encoding:'utf8',timeout:20000}).trim();
  for (const [i,label] of ['a','b'].entries()) {
   const context = await chromium.launchPersistentContext(E+'/profile-'+label, {
    executablePath:'/opt/google/chrome/chrome',headless:true,ignoreHTTPSErrors:false,permissions:['microphone'],
    env:{...process.env,HOME:E+'/browser-home'},
    args:['--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream',
     '--use-file-for-fake-audio-capture='+E+'/signal-'+label+'.wav',
     ...(remote?['--host-resolver-rules=MAP '+remote.hostname+' '+remote.address]:[]),
     '--autoplay-policy=no-user-gesture-required','--disable-background-timer-throttling',
     '--disable-renderer-backgrounding','--disable-features=WebRtcHideLocalIpsWithMdns']});
   contexts.push(context); const p = await context.newPage();pages.push(p);
   p.on('websocket',ws=>{ws.on('framesent',f=>{const first=String(f.payload).split('\r\n')[0];if (/^(REGISTER|INVITE|BYE|ACK) /.test(first))result.sip.push({label,line:first.split(' ')[0]+' sent'});});ws.on('framereceived',f=>{const first=String(f.payload).split('\r\n')[0];if(/^SIP\/2.0 \d+/.test(first))result.sip.push({label,line:first});});});
   await p.addInitScript(() => {
    window.alm={events:[],captures:{}};
    const get=navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
    navigator.mediaDevices.getUserMedia=async x=>{const s=await get(x);alm.localStream=s;return s;};
    const PC=window.RTCPeerConnection;
    window.RTCPeerConnection=class extends PC {constructor(...x){super(...x);alm.pc=this;this.addEventListener('track',e=>{alm.remoteStream=new MediaStream([e.track]);});}};
    alm.snapshot=async()=>({utc:new Date().toISOString(),ice:alm.pc?.iceConnectionState,connection:alm.pc?.connectionState,
     rows:alm.pc?[...(await alm.pc.getStats()).values()].filter(r=>['inbound-rtp','outbound-rtp','transport','candidate-pair','codec','local-candidate','remote-candidate'].includes(r.type)):[]});
   });
   const response=await p.goto(remote?'https://'+remote.hostname+'/almadial-phone/':'https://localhost:8443/almadial-phone/');
   result.participants.push({label,http:response.status(),tls:await response.securityDetails()});
   await p.locator('[name=password]').fill(secret);await p.getByRole('button',{name:'Open phone'}).click();await p.getByRole('button',{name:'Connect phone'}).click();
   await p.waitForFunction(()=>document.querySelector('#reg_status')?.value.startsWith('Registered'),{},{timeout:20000});
   await p.locator('#digits').fill('9271');await p.locator('#dial').click();
   result.phase='await-session-'+label;
   await p.waitForFunction(()=>typeof mySession!=='undefined'&&mySession?.state==='Established',{},{timeout:25000});
   result.phase='await-media-'+label;
   await p.waitForFunction(()=>alm.pc?.connectionState==='connected',{},{timeout:25000});
   await p.addScriptTag({path:__dirname+'/m0_05_capture.js'});
   if(i===0){await p.locator('#unregister').click();await p.waitForFunction(()=>document.querySelector('#reg_status').value==='Unregistered');}
  }
  result.conference=native('confbridge list 9600271');result.channels=native('core show channels concise');
  result.before=await Promise.all(pages.map(p=>p.evaluate(()=>alm.snapshot())));
  result.capture=await Promise.all(pages.map(p=>p.evaluate(()=>alm.capture(12))));
  result.after=await Promise.all(pages.map(p=>p.evaluate(()=>alm.snapshot())));
  for(const [i,p] of pages.entries())for(const name of ['local','remote']) {
   const encoded=await p.evaluate(async n=>{const b=await alm.captures[n].arrayBuffer();let s='';for(const c of new Uint8Array(b))s+=String.fromCharCode(c);return btoa(s);},name);
   fs.writeFileSync(E+'/'+(i?'b':'a')+'-'+name+'.wav',Buffer.from(encoded,'base64'),{flag:'wx',mode:0o600});
  }
  result.pass=result.after.every(s=>s.connection==='connected'&&['connected','completed'].includes(s.ice))&&result.capture.every(c=>c.localBytes>32000&&c.remoteBytes>32000);
 } catch(e) {result.error=String(e).split('Call log:')[0].slice(0,700);
  result.failureState=await Promise.all(pages.map(async p=>{try{return await p.evaluate(async()=>({session:typeof mySession==='undefined'?null:mySession?.state,snapshot:await alm.snapshot(),localSdp:alm.pc?.localDescription?.sdp?.replace(/^a=ice-(pwd|ufrag):.*$/gm,'a=ice-$1:[redacted]'),remoteSdp:alm.pc?.remoteDescription?.sdp?.replace(/^a=ice-(pwd|ufrag):.*$/gm,'a=ice-$1:[redacted]')}));}catch{return {observation:'unavailable'};}}));
 }
 finally {
  for(const p of pages){try {if(await p.evaluate(()=>typeof incall!=='undefined'&&incall))await p.locator('#dial').click();if(await p.evaluate(()=>typeof myRegisterer!=='undefined'&&myRegisterer.state==='Registered'))await p.locator('#unregister').click();}catch{}}
  for(const c of contexts)await c.close();
  fs.writeFileSync(E+'/browser.json',JSON.stringify(result,null,2),{flag:'wx',mode:0o600});
  console.log(JSON.stringify({pass:result.pass,error:result.error}));if(!result.pass)process.exitCode=1;
 }
})().catch(e=>{console.error(e.name);process.exitCode=1;});
