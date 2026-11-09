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
    await p.goto(origin + '/agc/vicidial.php');
    const phone = await privateJSON(cfg.phoneCredentials || '/tmp/web00-phone.json');
    await p.locator('[name=phone_login]').fill(phone.login);
    await p.locator('[name=phone_pass]').fill(phone.password);
    await p.waitForFunction(()=>Number(document.querySelector('[name=JS_browser_height]')?.value)>0);
    await Promise.all([p.waitForEvent('load'),p.locator('[name=login_sub]').click()]);
    const users = await privateJSON(cfg.agentCredentials || '/tmp/prd-users.json');
    const agent=users.find(u=>u.username===(cfg.agent || 'qaagent14'));
    if (!agent) throw new Error('Designated synthetic agent missing');
    await p.locator('[name=VD_login]').fill(agent.username);
    await p.locator('[name=VD_pass]').fill(agent.password);
    await p.locator('[name=VD_pass]').press('Tab');
    await p.locator('[name=VD_campaign] option[value="QAOUT01"]').waitFor({state:'attached'});
    await p.locator('[name=VD_campaign]').selectOption('QAOUT01');
    await p.waitForFunction(()=>Number(document.querySelector('[name=JS_browser_height]')?.value)>0);
    await Promise.all([p.waitForEvent('load'),p.locator('[name=login_sub]').click()]);
    await p.waitForFunction(()=>Array.from(document.querySelectorAll('iframe')).some(f=>f.src.includes('viciphone-web00')),{},{timeout:30000});
    const oldSession=p.locator('#DeactivateDOlDSessioNSpan');
    if(await oldSession.isVisible()) await oldSession.getByRole('link',{name:'OK',exact:true}).click();
    const frame=p.frames().find(f=>f.url().includes('/viciphone-web00/'));
    page.web00.phone=frame;
    await frame.waitForFunction(()=>typeof myRegisterer==='object' && myRegisterer.state==='Registered',null,{timeout:30000});
    await frame.waitForFunction(()=>typeof mySession==='object' && mySession.state==='Established',null,{timeout:30000});
    await frame.waitForFunction(()=>mySession.sessionDescriptionHandler.peerConnection.connectionState==='connected',null,{timeout:30000});
  }
  const w=page.web00;
  if (!w) throw new Error('No active workflow');
  if (phase === 'preview') {
    await w.p.locator('[name=LeadPreview]').check();
    await w.p.getByAltText('Dial Next Number',{exact:true}).click();
    await w.p.waitForFunction(()=>document.querySelector('[name=vendor_lead_code]').value==='WEB00_001');
    return {preview:'WEB00_001', customerCall:false};
  }
  if (phase === 'call') {
    if (!cfg.authorizeSingleCall || w.callPlaced) throw new Error('Single simulator call requires explicit authorization');
    if (await w.p.locator('[name=vendor_lead_code]').inputValue() !== 'WEB00_001') throw new Error('Wrong lead');
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
  if (phase === 'finish') {
    if(await w.p.getByText('SKIP LEAD',{exact:true}).isVisible()) throw new Error('Release preview through SKIP LEAD before logout');
    // Logout hides the embedded phone. Complete its native unregister action first.
    await w.phone.locator('#unregister').click();
    await w.phone.waitForFunction(()=>myRegisterer.state==='Unregistered',null,{timeout:15000});
    await w.p.getByRole('link',{name:'LOGOUT',exact:true}).click();
    await w.p.getByText('LOGOUT PROCESS COMPLETE, YOU MAY NOW CLOSE YOUR BROWSER OR LOG BACK IN',{exact:false}).waitFor({timeout:30000});
    await w.b.close();delete page.web00;
    return {registration:'Unregistered', logout:true, browserClosed:true};
  }
  return {browser:w.b.version(), errors:w.errors, phone: await w.phone.evaluate(async()=>{
    const pc=typeof mySession==='object'?mySession.sessionDescriptionHandler.peerConnection:null;
    const stats=pc?Array.from((await pc.getStats()).values()).filter(s=>['inbound-rtp','outbound-rtp','transport','codec','candidate-pair'].includes(s.type)).map(s=>Object.fromEntries(Object.entries(s).filter(([k])=>['type','mimeType','clockRate','bytesReceived','bytesSent','packetsReceived','packetsSent','packetsLost','totalAudioEnergy','totalSamplesDuration','audioLevel','dtlsState','iceState','srtpCipher','dtlsCipher','state','nominated'].includes(k)))):[];
    return {registration:myRegisterer.state,session:mySession?.state,conference:reg_exten,ice:pc?.iceConnectionState,connection:pc?.connectionState,signaling:pc?.signalingState,stats};
  })};
}
