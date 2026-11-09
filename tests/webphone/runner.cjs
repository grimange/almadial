// No new dependencies: point PLAYWRIGHT_MODULE at an existing installation.
const fs=require('fs');const path=require('path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const workflow=eval('('+fs.readFileSync(path.join(__dirname,'browser_workflow.js'),'utf8')+')');
(async()=>{
  if(!process.env.WEBPHONE_PHONE_CREDENTIALS||!process.env.WEBPHONE_AGENT_CREDENTIALS) throw Error('Set both private credential file paths');
  const executablePath=process.env.CHROMIUM_PATH;
  const controller=await chromium.launch({headless:true,...(executablePath?{executablePath}:{})});
  const page=await controller.newPage();const results=[];
  function state(label){
    if(!process.env.WEBPHONE_STATE_DIR)return;
    const dir=path.resolve(process.env.WEBPHONE_STATE_DIR);fs.mkdirSync(dir,{recursive:true});
    require('child_process').execFileSync('python3',[path.join(__dirname,'preview_state.py'),path.join(dir,label+'.json')],{stdio:'pipe',timeout:45000});
  }
  const cfg={phoneCredentials:process.env.WEBPHONE_PHONE_CREDENTIALS,agentCredentials:process.env.WEBPHONE_AGENT_CREDENTIALS,executablePath,audioPath:path.join(__dirname,'fixtures/microphone.wav')};
  try{
    const previewCase=process.env.WEBPHONE_PREVIEW_CASE;
    if(previewCase){
      let vendor=process.env.WEBPHONE_PREVIEW_VENDOR;
      const corrected=['corrected','repeated','isolation','retry'].includes(previewCase);
      if(!['skip','direct','corrected','repeated','isolation','retry'].includes(previewCase)||!(vendor==='GAP01B_NEXT'||/^GAP01B_\d{3}$/.test(vendor||'')))throw Error('Choose a supported preview case and an unused GAP01B vendor');
      if(previewCase==='direct'&&process.env.WEBPHONE_PREVIEW_DIAGNOSTIC!=='YES')throw Error('Direct logout requires explicit diagnostic opt-in and native recovery');
      if(corrected&&process.env.WEBPHONE_PREVIEW_CORRECTION_TEST!=='YES')throw Error('Corrected logout requires an explicitly approved patch test');
      state('before');
      page.webphoneConfig={...cfg,phase:'start',agent:process.env.WEBPHONE_AGENT||'qaagent14'};results.push(await workflow(page));
      // Fail before claiming a lead when the proposed correction is absent.
      if(corrected&&!await page.web00.p.evaluate(()=>typeof ReleasePreviewForLogout==='function'))throw Error('Proposed correction is not installed');
      page.webphoneConfig={...cfg,phase:'preview',previewVendor:vendor};const selection=await workflow(page);results.push(selection);vendor=selection.preview;state('preview');
      if(previewCase==='isolation'){
        const other=JSON.parse(fs.readFileSync(cfg.agentCredentials,'utf8')).find(u=>u.username==='qasup14');
        if(!other)throw Error('Restricted synthetic supervisor credentials unavailable');
        page.webphoneConfig={...cfg,phase:'probe-release',probeOverride:{user:other.username,pass:other.password}};
        const otherResult=await workflow(page);if(!otherResult.denied)throw Error('Unrelated user released preview');
        results.push({otherAccount:'qasup14',...otherResult});state('after-other-account');
        page.webphoneConfig={...cfg,phase:'probe-release',probeOverride:{conf_exten:'9600099'}};
        const wrongSession=await workflow(page);if(!wrongSession.denied)throw Error('Wrong conference released preview');
        results.push({wrongConference:true,...wrongSession});state('after-wrong-conference');
      }
      if(previewCase==='skip'){
        page.webphoneConfig={...cfg,phase:'skip'};results.push(await workflow(page));state('skipped');
      }
      page.webphoneConfig={...cfg,phase:corrected?'corrected-preview-logout':previewCase==='direct'?'direct-preview-logout':'finish',previewVendor:vendor,authorizePreviewLogoutDiagnostic:previewCase==='direct',authorizeProposedLogoutTest:corrected,repeatLogout:previewCase==='repeated',dropFirstReleaseAck:previewCase==='retry'};results.push(await workflow(page));
      state('after');
      console.log(JSON.stringify({customerCalls:0,previewCase,requiresReadOnlyCorrelation:true,requiresNativeRecovery:previewCase==='direct',results},null,2));
      return;
    }
    if(process.env.WEBPHONE_CALL_PROTECTION==='YES'){
      const vendor='GAP03B_CALL_001';state('before');
      page.webphoneConfig={...cfg,phase:'start'};results.push(await workflow(page));
      page.webphoneConfig={...cfg,phase:'preview',previewVendor:vendor};const callPreview=await workflow(page);results.push(callPreview);state('preview');
      page.webphoneConfig={...cfg,phase:'call',previewVendor:vendor,authorizeSingleCall:true};results.push(await workflow(page));
      await page.web00.p.waitForFunction(()=>VD_live_customer_call===1 && MD_channel_look!==1,null,{timeout:65000});
      state('answered');
      let blockedDialog=false;
      page.web00.p.once('dialog',async dialog=>{blockedDialog=/LIVE CALL|Dial attempt/i.test(dialog.message());await dialog.accept()});
      await page.web00.p.getByRole('link',{name:'LOGOUT',exact:true}).click();
      if(!blockedDialog)throw Error('Native logout did not protect active call');
      page.webphoneConfig={...cfg,phase:'probe-release',probeOverride:{stage:'NEW',called_count:'1'}};
      const probe=await workflow(page);if(!probe.denied)throw Error('Active call preview release was accepted');
      results.push({activeCallLogoutBlocked:blockedDialog,nativeReleaseDenied:probe.denied});state('protected');
      page.webphoneConfig={...cfg,phase:'hangup'};results.push(await workflow(page));state('dispositioned');
      page.webphoneConfig={...cfg,phase:'probe-release',probeOverride:{lead_id:callPreview.leadId,stage:'NEW',called_count:'1'}};
      const terminal=await workflow(page);if(!terminal.denied)throw Error('Final disposition was overwritten');
      results.push({completedDispositionReleaseDenied:terminal.denied});
      page.webphoneConfig={...cfg,phase:'finish'};results.push(await workflow(page));state('after');
      console.log(JSON.stringify({customerCalls:1,localSimulatorOnly:true,results},null,2));return;
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
      if(page.web00.callPlaced){page.webphoneConfig={...cfg,phase:"hangup"};await workflow(page);}
      // A failed expected-lead assertion can still leave a real preview open.
      // Release only a recognized synthetic preview through the visible control.
      if(['corrected','repeated','isolation','retry'].includes(process.env.WEBPHONE_PREVIEW_CASE) && await page.web00.p.getByText('SKIP LEAD',{exact:true}).isVisible()){
        const vendor=await page.web00.p.locator('[name=vendor_lead_code]').inputValue();
        page.webphoneConfig={...cfg,phase:'corrected-preview-logout',previewVendor:vendor,authorizeProposedLogoutTest:true};
        await workflow(page);console.error('Guarded native logout completed after test failure');
      }
      if(page.web00 && await page.web00.p.getByText('SKIP LEAD',{exact:true}).isVisible()){
        page.webphoneConfig={...cfg,phase:'skip'};await workflow(page);
      }
      if(page.web00){page.webphoneConfig={...cfg,phase:'finish'};await workflow(page);}
    }catch(_) {console.error('Native cleanup requires operator attention');await page.web00.b.close()}}
  }finally{await controller.close()}
})();
