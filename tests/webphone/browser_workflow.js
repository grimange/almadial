// Run with Playwright MCP browser_run_code_unsafe(filename), or runner.cjs.
// Credential files stay local and are read without logging their contents.
async (page) => {
  const cfg = page.webphoneConfig || {};
  const origin = cfg.origin || 'https://2.29.153.161';
  const phase = cfg.phase || 'start';
  if (phase === 'start') {
    if (page.web00) throw new Error('An existing test session requires cleanup');
    const browser = await page.context().browser().browserType().launch({
      headless: true,
      executablePath: cfg.executablePath || '/home/ra/.cache/ms-playwright/chromium-1228/chrome-linux64/chrome',
      args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream',
        '--use-file-for-fake-audio-capture=' + (cfg.audioPath || '/home/ra/Documents/almadial/tests/webphone/fixtures/microphone.wav'),
        '--autoplay-policy=no-user-gesture-required']
    });
    const ctx = await browser.newContext({permissions: ['microphone']});
    const p = await ctx.newPage();
    page.web00 = {b:browser, ctx, p, errors:[]};
    p.on('pageerror', e => page.web00?.errors.push(e.name));
    async function privateJSON(path) {
      await p.locator('body').waitFor();
      await p.evaluate(() => {const e=document.createElement('input');e.type='file';e.id='private-fixture';document.body.append(e)});
      await p.locator('#private-fixture').setInputFiles(path);
      return p.evaluate(async()=>{const e=document.querySelector('#private-fixture');const v=JSON.parse(await e.files[0].text());e.remove();return v});
    }
    await p.goto(origin + '/agc/vicidial.php', {waitUntil:'domcontentloaded'});
    const phone = await privateJSON(cfg.phoneCredentials || '/tmp/web00-phone.json');
    await p.locator('[name=phone_login]').fill(phone.login);
    await p.locator('[name=phone_pass]').fill(phone.password);
    await p.waitForFunction(()=>Number(document.querySelector('[name=JS_browser_height]')?.value)>0);
    await Promise.all([p.waitForNavigation({waitUntil:'domcontentloaded'}),p.locator('[name=login_sub]').click()]);
    const users = await privateJSON(cfg.agentCredentials || '/tmp/prd-users.json');
    const agent=users.find(u=>u.username===(cfg.agent || 'qaagent14'));
    if (!agent) throw new Error('Designated synthetic agent missing');
    await p.locator('[name=VD_login]').fill(agent.username);
    await p.locator('[name=VD_pass]').fill(agent.password);
    await p.locator('[name=VD_pass]').press('Tab');
    await p.locator('[name=VD_campaign] option[value="QAOUT01"]').waitFor({state:'attached'});
    await p.locator('[name=VD_campaign]').selectOption('QAOUT01');
    await p.waitForFunction(()=>Number(document.querySelector('[name=JS_browser_height]')?.value)>0);
    await Promise.all([p.waitForNavigation({waitUntil:'domcontentloaded'}),p.locator('[name=login_sub]').click()]);
    await p.waitForFunction(()=>Array.from(document.querySelectorAll('iframe')).some(f=>f.src.includes('viciphone-web00')),{},{timeout:30000});
    const frame=p.frames().find(f=>f.url().includes('/viciphone-web00/')) || await p.waitForEvent('framenavigated',{
      predicate:f=>f.url().includes('/viciphone-web00/'),timeout:30000
    });
    page.web00.phone=frame;
    await frame.waitForFunction(()=>typeof myRegisterer==='object' && myRegisterer.state==='Registered',null,{timeout:30000});
    await frame.waitForFunction(()=>typeof mySession==='object' && mySession.state==='Established',null,{timeout:30000});
    await frame.waitForFunction(()=>mySession.sessionDescriptionHandler.peerConnection.connectionState==='connected',null,{timeout:30000});
    // Phone initialization overlays can temporarily cover the old-session notice.
    await p.locator('#LoadingBox').waitFor({state:'hidden',timeout:30000});
    const oldSession=p.locator('#DeactivateDOlDSessioNSpan');
    if(await oldSession.isVisible()) await oldSession.getByRole('link',{name:'OK',exact:true}).click();
  }
  const w=page.web00;
  if (!w) throw new Error('No active workflow');
  if (phase === 'preview') {
    const expected = cfg.previewVendor || 'WEB00_001';
    if (expected !== 'WEB00_001' && expected !== 'GAP01B_NEXT' && expected !== 'GAP03B_CALL_001' && !/^GAP01B_\d{3}$/.test(expected)) throw new Error('Preview requires an identified synthetic lead');
    await w.p.locator('[name=LeadPreview]').check();
    await w.p.getByAltText('Dial Next Number',{exact:true}).click();
    await w.p.waitForFunction(()=>document.querySelector('[name=lead_id]').value!=='',null,{timeout:30000});
    const actual=await w.p.locator('[name=vendor_lead_code]').inputValue();
    if(expected==='GAP01B_NEXT' ? !/^GAP01B_\d{3}$/.test(actual) : actual!==expected) throw Error('Unexpected native preview selection');
    return {preview:actual, leadId:await w.p.locator('[name=lead_id]').inputValue(), customerCall:false};
  }
  if (phase === 'skip') {
    const vendor=await w.p.locator('[name=vendor_lead_code]').inputValue();
    if (!/^GAP01B_\d{3}$/.test(vendor) && vendor !== 'WEB00_001') throw new Error('Skip requires an identified synthetic lead');
    await w.p.getByText('SKIP LEAD',{exact:true}).click();
    await w.p.waitForFunction(()=>document.querySelector('[name=lead_id]').value==='');
    return {skipped:vendor, customerCall:false};
  }
  if (phase === 'probe-release') {
    // Exercise the authenticated native endpoint. Never return credentials or HTML.
    return w.p.evaluate(async override=>{
      const fields={server_ip,session_name,ACTION:'manDiaLskip',preview_action:'logout',
        conf_exten:session_id,user,pass,lead_id:document.vicidial_form.lead_id.value,
        stage:previous_dispo,called_count:previous_called_count,campaign};
      Object.assign(fields,override);
      const response=await fetch('vdc_db_query.php',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:new URLSearchParams(fields)});
      const text=(await response.text()).trim();
      return {status:response.status,releaseAcknowledged:text==='PREVIEW RELEASED',denied:text!=='PREVIEW RELEASED'};
    },cfg.probeOverride || {});
  }
  if (phase === 'call') {
    if (!cfg.authorizeSingleCall || w.callPlaced) throw new Error('Single simulator call requires explicit authorization');
    if (await w.p.locator('[name=vendor_lead_code]').inputValue() !== (cfg.previewVendor || 'WEB00_001')) throw new Error('Wrong lead');
    if(!['WEB00_001','GAP03B_CALL_001'].includes(cfg.previewVendor || 'WEB00_001'))throw Error('Unrecognized local simulator fixture');
    const dial=await w.p.evaluate(()=>({omit:omit_phone_code,prefix:manual_dial_prefix}));
    if(dial.omit!=='Y'||dial.prefix!=='2221') throw new Error('Unexpected simulator dialing settings');
    w.callPlaced=true;
    await w.p.getByText('DIAL LEAD',{exact:true}).click();
    return {singleSimulatorCallRequested:true};
  }
  if (phase === 'hangup') {
    await w.p.getByAltText('Hangup Customer',{exact:true}).click();
    await w.p.getByText('NI - Not Interested',{exact:true}).waitFor();
    await w.p.getByText('NI - Not Interested',{exact:true}).click();
    await w.p.locator('a:visible').filter({hasText:/^SUBMIT$/}).click();
    await w.p.waitForFunction(()=>document.querySelector('#DispoSelectBox').style.visibility==='hidden');
    return {hangup:true,disposition:'NI'};
  }
  if (phase === 'corrected-preview-logout') {
    const vendor=await w.p.locator('[name=vendor_lead_code]').inputValue();
    if (!cfg.authorizeProposedLogoutTest || w.callPlaced || !/^GAP01B_\d{3}$/.test(vendor) || vendor!==cfg.previewVendor || !await w.p.getByText('SKIP LEAD',{exact:true}).isVisible()) throw new Error('Proposed logout test requires an identified non-calling preview');
    if (!await w.p.evaluate(()=>typeof ReleasePreviewForLogout==='function')) throw new Error('Correction absent; release preview through SKIP');
    const responses=[];
    const pending=[];
    const listener=response=>{
      if(!response.url().endsWith('/vdc_db_query.php'))return;
      const form=new URLSearchParams(response.request().postData()||'');
      const action=form.get('ACTION');
      if(action!=='userLOGout' && !(action==='manDiaLskip' && form.get('preview_action')==='logout'))return;
      // Record only action/ACK, never credentials, request bodies or native HTML.
      const entry={action,ack:false};responses.push(entry);
      pending.push(response.text().then(body=>{
        const text=body.trim();
        entry.ack=response.status()===200 && (action==='manDiaLskip'?text==='PREVIEW RELEASED':/^\d+\|\d*\|\d+\|\d+\|[^|]*\|\d+$/.test(text));
      }));
    };
    w.p.on('response',listener);
    try {
      const logout=w.p.getByRole('link',{name:'LOGOUT',exact:true});
      if(cfg.dropFirstReleaseAck){
        let dropped=false;
        await w.p.route('**/vdc_db_query.php',async route=>{
          const form=new URLSearchParams(route.request().postData()||'');
          if(!dropped && form.get('ACTION')==='manDiaLskip' && form.get('preview_action')==='logout'){
            dropped=true;const native=await route.fetch();
            if((await native.text()).trim()!=='PREVIEW RELEASED')throw Error('Native release did not commit');
            await route.fulfill({status:503,body:'Synthetic lost acknowledgement'});
          }else await route.continue();
        });
        await logout.click();
        await w.p.locator('#AlertBoxContent').filter({hasText:'Preview release was not confirmed'}).waitFor({state:'visible'});
        if(!await w.p.evaluate(()=>in_lead_preview_state===1 && !preview_logout_pending))throw Error('Interrupted release lost frontend state');
        await w.p.locator('#alert_button').click();await logout.click();
      }else if(cfg.repeatLogout) await logout.dblclick({delay:30}); else await logout.click();
      await w.p.getByText('LOGOUT PROCESS COMPLETE, YOU MAY NOW CLOSE YOUR BROWSER OR LOG BACK IN',{exact:false}).waitFor({timeout:30000});
      await Promise.all(pending);
      const confirmed=cfg.dropFirstReleaseAck?responses.slice(1):responses;
      if((cfg.dropFirstReleaseAck && (responses.length!==3 || responses[0].ack)) || confirmed.length!==2 || confirmed[0].action!=='manDiaLskip' || confirmed[1].action!=='userLOGout' || confirmed.some(r=>!r.ack)) throw new Error('Release/logout acknowledgement sequence did not match');
      await w.p.waitForFunction(()=>document.getElementById('webphone').getAttribute('src')==='./images/blank.gif');
    } finally {w.p.off('response',listener)}
    // Do not unregister first: this test exercises actual native logout teardown.
    await w.b.close();delete page.web00;
    return {logout:true,browserClosed:true,releaseAcknowledged:true,duplicateClick:!!cfg.repeatLogout,lostAcknowledgementRetry:!!cfg.dropFirstReleaseAck,requiresReadOnlyCorrelation:true};
  }
  if (phase === 'finish' || phase === 'direct-preview-logout') {
    const previewOpen=await w.p.getByText('SKIP LEAD',{exact:true}).isVisible();
    if (phase === 'direct-preview-logout') {
      const vendor=await w.p.locator('[name=vendor_lead_code]').inputValue();
      if (!cfg.authorizePreviewLogoutDiagnostic || !previewOpen || !/^GAP01B_\d{3}$/.test(vendor) || vendor!==cfg.previewVendor || w.callPlaced) throw new Error('Direct logout requires an explicit non-calling synthetic preview diagnostic');
    } else if(previewOpen) throw new Error('Release preview through SKIP LEAD before logout');
    // Logout hides the embedded phone. Complete its native unregister action first.
    await w.phone.locator('#unregister').click();
    await w.phone.waitForFunction(()=>myRegisterer.state==='Unregistered',null,{timeout:15000});
    await w.p.getByRole('link',{name:'LOGOUT',exact:true}).click();
    await w.p.getByText('LOGOUT PROCESS COMPLETE, YOU MAY NOW CLOSE YOUR BROWSER OR LOG BACK IN',{exact:false}).waitFor({timeout:30000});
    await w.b.close();delete page.web00;
    return {registration:'Unregistered', logout:true, browserClosed:true, openPreviewDiagnostic:phase==='direct-preview-logout'};
  }
  return {browser:w.b.version(), errors:w.errors, phone: await w.phone.evaluate(async()=>{
    const pc=typeof mySession==='object'?mySession.sessionDescriptionHandler.peerConnection:null;
    const stats=pc?Array.from((await pc.getStats()).values()).filter(s=>['inbound-rtp','outbound-rtp','transport','codec','candidate-pair'].includes(s.type)).map(s=>Object.fromEntries(Object.entries(s).filter(([k])=>['type','mimeType','clockRate','bytesReceived','bytesSent','packetsReceived','packetsSent','packetsLost','totalAudioEnergy','totalSamplesDuration','audioLevel','dtlsState','iceState','srtpCipher','dtlsCipher','state','nominated'].includes(k)))):[];
    return {registration:myRegisterer.state,session:mySession?.state,conference:reg_exten,ice:pc?.iceConnectionState,connection:pc?.connectionState,signaling:pc?.signalingState,stats};
  })};
}
