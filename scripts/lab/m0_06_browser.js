// Playwright MCP browser_run_code_unsafe function. See M0-06 report for setup.
async (page) => {
 // Supply reviewed, private paths through the invoking page URL; no credentials in URLs.
 const options=new URL(page.url()).searchParams;
 const H=options.get('evidence'); const A=options.get('alm3Evidence');
 const run=options.get('run'); const address=options.get('vmAddress');
 const root=options.get('repo');
 if(!H?.startsWith('/')||!A?.startsWith('/')||!root?.startsWith('/')||!address?.match(/^[0-9.]+$/)||!run?.match(/^[a-z0-9-]{1,40}$/))throw Error('Explicit private evidence/repository paths, VM IPv4 address and unique run ID required');const contexts=[],pages=[],result={run,participants:[]};
 try{
  for(const [i,id] of ['9051','9052'].entries()){
   const ctx=await page.context().browser().browserType().launchPersistentContext(H+'/profile-'+run+'-'+id,{
    executablePath:'/opt/google/chrome/chrome',headless:true,ignoreHTTPSErrors:false,permissions:['microphone'],
    env:{HOME:A+'/browser-home',PATH:'/usr/bin:/bin'},args:['--disable-background-timer-throttling','--disable-renderer-backgrounding','--disable-backgrounding-occluded-windows','--autoplay-policy=no-user-gesture-required','--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream','--use-file-for-fake-audio-capture='+H+'/signal-'+(i?'b':'a')+'.wav','--autoplay-policy=no-user-gesture-required','--host-resolver-rules=MAP almadial-alma10.test '+address+'']});
   contexts.push(ctx);await ctx.tracing.start({screenshots:true,snapshots:true,sources:true});
   const p=await ctx.newPage();pages.push(p);
   const response=await p.goto('https://almadial-alma10.test:8089/static/alm6.html?user='+id+'&dest=9600001');
   result.participants.push({id,tls:await response.securityDetails(),http:response.status()});
   await p.addScriptTag({path:root+'/scripts/lab/m0_05_capture.js'});
   await p.evaluate(()=>{const input=document.createElement('input');input.type='file';input.id='credential';input.onchange=async()=>{try{await alm.register((await input.files[0].text()).trim());}catch(e){alm.error=String(e);}};document.body.append(input);});
   await p.locator('#credential').setInputFiles(H+'/phone-'+id);
   await p.waitForFunction(()=>alm.events.some(e=>e.type==='registered'),{timeout:15000});
  }
  await Promise.all(pages.map(p=>p.evaluate(()=>alm.call())));
  await Promise.all(pages.map(p=>p.waitForFunction(()=>alm.pc?.connectionState==='connected')));
  await pages[0].waitForTimeout(3000);
  async function capturePhase(phase, selected, seconds){
   const before=await Promise.all(selected.map(p=>p.evaluate(()=>alm.snapshot())));
   const captures=await Promise.all(selected.map(p=>p.evaluate(s=>alm.capture(s),seconds)));
   const after=await Promise.all(selected.map(p=>p.evaluate(()=>alm.snapshot())));
   for(const [j,p] of selected.entries()){
    const i=pages.indexOf(p),id=result.participants[i].id;
    const record={before:before[j],after:after[j],capture:captures[j]};
    result.participants[i][phase]=record;
    for(const name of ['local','remote']){
     const wait=p.waitForEvent('download');await p.evaluate(n=>alm.download(n),name);await(await wait).saveAs(H+'/'+run+'-'+phase+'-'+id+'-'+name+'.wav');
    }
   }
  }
  await capturePhase('two-party',pages.slice(),12);
  const ctx=await page.context().browser().browserType().launchPersistentContext(H+'/profile-'+run+'-9053',{
   executablePath:'/opt/google/chrome/chrome',headless:true,ignoreHTTPSErrors:false,permissions:['microphone'],env:{HOME:A+'/browser-home',PATH:'/usr/bin:/bin'},
   args:['--disable-background-timer-throttling','--disable-renderer-backgrounding','--disable-backgrounding-occluded-windows','--autoplay-policy=no-user-gesture-required','--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream','--use-file-for-fake-audio-capture='+H+'/signal-c.wav','--host-resolver-rules=MAP almadial-alma10.test '+address+'']});
  contexts.push(ctx);await ctx.tracing.start({screenshots:true,snapshots:true,sources:true});const p=await ctx.newPage();pages.push(p);
  const response=await p.goto('https://almadial-alma10.test:8089/static/alm6.html?user=9053&dest=9600001');result.participants.push({id:'9053',http:response.status(),tls:await response.securityDetails()});
  await p.addScriptTag({path:root+'/scripts/lab/m0_05_capture.js'});
  await p.evaluate(()=>{const i=document.createElement('input');i.type='file';i.id='credential';i.onchange=async()=>{try{await alm.register((await i.files[0].text()).trim());}catch(e){alm.error=String(e);}};document.body.append(i);});
  await p.locator('#credential').setInputFiles(H+'/phone-9053');await p.waitForFunction(()=>alm.events.some(e=>e.type==='registered'));
  await p.evaluate(()=>alm.call());await p.waitForFunction(()=>alm.pc?.connectionState==='connected');
  await capturePhase('monitor',pages.slice(),8);
  await pages[0].waitForTimeout(4000);
  await pages[1].evaluate(()=>alm.localStream.getAudioTracks().forEach(t=>t.enabled=false));
  await capturePhase('barge',[pages[0],pages[2]],8);
  await pages[1].evaluate(()=>alm.localStream.getAudioTracks().forEach(t=>t.enabled=true));
  await pages[0].waitForTimeout(6000);
  for(const [i,p] of pages.entries())result.participants[i].events=await p.evaluate(()=>alm.events);
  await Promise.all(pages.map(p=>p.evaluate(async()=>{try{await alm.user.hangup();}catch{}await alm.user.unregister();await alm.user.disconnect();})));
  await pages[0].waitForTimeout(1000);
  result.state='CAPTURED';
 }catch(e){result.error=String(e);result.state='FAILED';for(const p of pages){try{result.diagnostics=(result.diagnostics||[]).concat(await p.evaluate(()=>({events:alm.events,error:alm.error,pc:alm.pc?.connectionState})));}catch{}}}
 finally{
  if(pages[0]){const wait=pages[0].waitForEvent('download');await pages[0].evaluate(r=>alm.downloadJson('run',r),result);await(await wait).saveAs(H+'/'+run+'-browser.json');}
  for(const [i,c] of contexts.entries()){await c.tracing.stop({path:H+'/'+run+'-trace-'+i+'.zip'});await c.close();}
 }
 return {state:result.state,error:result.error,participants:result.participants.map(p=>({id:p.id,http:p.http,tls:p.tls,phases:Object.keys(p).filter(k=>['two-party','monitor','barge'].includes(k))})),diagnostics:result.diagnostics};
}
