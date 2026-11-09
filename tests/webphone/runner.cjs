// No new dependencies: point PLAYWRIGHT_MODULE at an existing installation.
const fs=require('fs');const path=require('path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const workflow=eval('('+fs.readFileSync(path.join(__dirname,'browser_workflow.js'),'utf8')+')');
(async()=>{
  if(!process.env.WEBPHONE_PHONE_CREDENTIALS||!process.env.WEBPHONE_AGENT_CREDENTIALS) throw Error('Set both private credential file paths');
  const executablePath=process.env.CHROMIUM_PATH;
  const controller=await chromium.launch({headless:true,...(executablePath?{executablePath}:{})});
  const page=await controller.newPage();const results=[];
  const cfg={phoneCredentials:process.env.WEBPHONE_PHONE_CREDENTIALS,agentCredentials:process.env.WEBPHONE_AGENT_CREDENTIALS,executablePath,audioPath:path.join(__dirname,'fixtures/microphone.wav')};
  try{
    const previewCase=process.env.WEBPHONE_PREVIEW_CASE;
    if(previewCase){
      const vendor=process.env.WEBPHONE_PREVIEW_VENDOR;
      if(!['skip','direct'].includes(previewCase)||!/^GAP01B_\d{3}$/.test(vendor||''))throw Error('Choose skip/direct and an unused GAP01B vendor');
      if(previewCase==='direct'&&process.env.WEBPHONE_PREVIEW_DIAGNOSTIC!=='YES')throw Error('Direct logout requires explicit diagnostic opt-in and native recovery');
      page.webphoneConfig={...cfg,phase:'start',agent:process.env.WEBPHONE_AGENT||'qaagent14'};results.push(await workflow(page));
      page.webphoneConfig={...cfg,phase:'preview',previewVendor:vendor};results.push(await workflow(page));
      if(previewCase==='skip'){
        page.webphoneConfig={...cfg,phase:'skip'};results.push(await workflow(page));
      }
      page.webphoneConfig={...cfg,phase:previewCase==='direct'?'direct-preview-logout':'finish',previewVendor:vendor,authorizePreviewLogoutDiagnostic:previewCase==='direct'};results.push(await workflow(page));
      console.log(JSON.stringify({customerCalls:0,previewCase,requiresReadOnlyCorrelation:true,requiresNativeRecovery:previewCase==='direct',results},null,2));
      return;
    }
    for(let i=0;i<2;i++){
      page.webphoneConfig={...cfg,phase:'start'};results.push(await workflow(page));
      await page.web00.phone.waitForFunction(async()=>Array.from((await mySession.sessionDescriptionHandler.peerConnection.getStats()).values()).some(s=>s.type==='outbound-rtp'&&s.bytesSent>16000),null,{timeout:15000});
      page.webphoneConfig={...cfg,phase:'inspect'};results.push(await workflow(page));
      page.webphoneConfig={...cfg,phase:'finish'};results.push(await workflow(page));
    }
    console.log(JSON.stringify({customerCalls:0,results},null,2));
  }catch(e){
    // Never dump page HTML, iframe URLs, console payloads or authenticated profiles.
    console.error(e.name+': workflow did not complete; inspect native session state before retry');
    // Source coordinates aid diagnosis without exposing credential-bearing URLs.
    console.error((e.stack||'').split('\n').slice(1).map(l=>l.match(/:(\d+):(\d+)\)?$/)?.[0]).filter(Boolean).join(' '));
    process.exitCode=1;
    if(page.web00) {try{
      // A failed expected-lead assertion can still leave a real preview open.
      // Release only a recognized synthetic preview through the visible control.
      if(await page.web00.p.getByText('SKIP LEAD',{exact:true}).isVisible()){
        page.webphoneConfig={...cfg,phase:'skip'};await workflow(page);
      }
      page.webphoneConfig={...cfg,phase:'finish'};await workflow(page);
    }catch(_) {console.error('Native cleanup requires operator attention');await page.web00.b.close()}}
  }finally{await controller.close()}
})();
