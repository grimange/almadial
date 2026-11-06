// Playwright MCP browser_run_code_unsafe function. ALM-7 lab only, no TLS bypass.
// Set page.alm7Config with {evidence, alm3Evidence, repo, vmAddress, run, phase}.
// Phases: start -> capture -> finish. Always finish, including failed captures.
// Requires protected phone-post.json/phone-post-9052.json and signal-a/b.wav.
// Traces contain session material: keep them outside Git in a fresh private root.
async (page) => {
 const c = page.alm7Config;
 if (!c || !['start', 'capture', 'finish'].includes(c.phase)) throw Error('Explicit phase required');
 if (c.phase === 'start') {
  if (page.alm7Run) throw Error('Finish the previous run before starting another');
  if (![c.evidence, c.alm3Evidence, c.repo].every(x => typeof x === 'string' && x.startsWith('/')) ||
      !/^[0-9.]+$/.test(c.vmAddress || '') || !/^[a-z0-9-]{1,40}$/.test(c.run || '')) {
   throw Error('Explicit protected paths, current VM IPv4 and unique run ID required');
  }
  const s = page.alm7Run = {config: {...c}, contexts: [], pages: [],
   result: {run: c.run, client: 'self-hosted pinned VICIphone', participants: [], errors: [], requests: []}};
  for (const [i, id] of ['9051', '9052'].entries()) {
   if (i === 0 && c.embeddedAgent === true) {
    const owner = page.alm7?.p;
    const frame = owner?.frames().find(f => f.url().includes('/viciphone/viciphone.php'));
    if (!frame) throw Error('An authenticated agent with its embedded phone is required');
    await frame.waitForFunction(() => myRegisterer.state === 'Registered' && mySession?.state === 'Established');
    await frame.evaluate(() => {
     const pc = mySession.sessionDescriptionHandler.peerConnection;
     window.alm = {events: [], captures: {}, pc,
      localStream: new MediaStream(pc.getSenders().map(s => s.track).filter(Boolean)),
      remoteStream: new MediaStream(pc.getReceivers().map(r => r.track).filter(Boolean))};
     alm.snapshot = async () => ({utc: new Date().toISOString(), ice: pc.iceConnectionState,
      connection: pc.connectionState, rows: [...(await pc.getStats()).values()].filter(r =>
       ['inbound-rtp', 'outbound-rtp', 'transport', 'candidate-pair', 'codec'].includes(r.type))});
     alm.download = name => {const a = document.createElement('a');
      a.href = URL.createObjectURL(alm.captures[name]); a.download = name + '.wav'; a.click();};
     alm.downloadJson = value => {const a = document.createElement('a');
      a.href = URL.createObjectURL(new Blob([JSON.stringify(value, null, 2)])); a.download = 'run.json'; a.click();};
    });
    await frame.addScriptTag({path: c.repo + '/scripts/lab/m0_05_capture.js'});
    s.pages.push(frame); s.agentOwner = owner;
    s.result.participants.push({id, attachment: 'actual VICIdial agent iframe', tlsValidation: true});
    continue;
   }
   const ctx = await page.context().browser().browserType().launchPersistentContext(
    c.evidence + '/profile-' + c.run + '-' + id, {
     executablePath: '/opt/google/chrome/chrome', headless: true,
     ignoreHTTPSErrors: false, permissions: ['microphone'],
     env: {HOME: c.alm3Evidence + '/browser-home', PATH: '/usr/bin:/bin'},
     args: ['--disable-background-timer-throttling', '--disable-renderer-backgrounding',
      '--disable-backgrounding-occluded-windows', '--autoplay-policy=no-user-gesture-required',
      '--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream',
      '--use-file-for-fake-audio-capture=' + (c.fixtures || c.evidence) + '/signal-' + (i ? 'b' : 'a') + '.wav',
      '--host-resolver-rules=MAP almadial-alma10.test ' + c.vmAddress]});
   s.contexts.push(ctx);
   await ctx.tracing.start({screenshots: true, snapshots: true, sources: true});
   const p = await ctx.newPage(); s.pages.push(p);
   p.on('pageerror', e => s.result.errors.push({id, error: String(e)}));
   // Export only origin/path, never URL query values or POST bodies.
   p.on('request', r => {const u = new URL(r.url());
    s.result.requests.push({id, origin: u.origin, path: u.pathname, method: r.method()});});
   await p.addInitScript(() => {
    window.alm = {events: [], captures: {}};
    const original = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
    navigator.mediaDevices.getUserMedia = async options => {
     const stream = await original(options); alm.localStream = stream; return stream;
    };
    const PC = window.RTCPeerConnection;
    window.RTCPeerConnection = class extends PC {
     constructor(...args) {super(...args); alm.pc = this;
      this.addEventListener('track', event => {alm.remoteStream = new MediaStream([event.track]);});}
    };
    alm.snapshot = async () => ({utc: new Date().toISOString(),
     ice: alm.pc?.iceConnectionState, connection: alm.pc?.connectionState,
     localSdp: alm.pc?.localDescription?.sdp, remoteSdp: alm.pc?.remoteDescription?.sdp,
     rows: alm.pc ? [...(await alm.pc.getStats()).values()].filter(r =>
      ['inbound-rtp', 'outbound-rtp', 'transport', 'candidate-pair', 'codec',
       'local-candidate', 'remote-candidate'].includes(r.type)) : []});
    alm.download = name => {const a = document.createElement('a');
     a.href = URL.createObjectURL(alm.captures[name]); a.download = name + '.wav'; a.click();};
    alm.downloadJson = value => {const a = document.createElement('a');
     a.href = URL.createObjectURL(new Blob([JSON.stringify(value, null, 2)], {type: 'application/json'}));
     a.download = 'run.json'; a.click();};
   });
   const response = await p.goto('https://almadial-alma10.test:8443/');
   s.result.participants.push({id, http: response.status(), tls: await response.securityDetails()});
   await p.evaluate(() => {const input = document.createElement('input');
    input.type = 'file'; input.id = 'alm7-params'; input.onchange = async () => {
     const data = JSON.parse(await input.files[0].text()), form = document.createElement('form');
     form.action = '/viciphone/viciphone.php'; form.method = 'POST';
     for (const [key, value] of Object.entries(data)) {const field = document.createElement('input');
      field.type = 'hidden'; field.name = key; field.value = value; form.append(field);}
     document.body.append(form); form.submit();
    }; document.body.append(input);});
   await p.locator('#alm7-params').setInputFiles((c.fixtures || c.evidence) + (i ? '/phone-post-9052.json' : '/phone-post.json'));
   await p.waitForFunction(() => typeof myRegisterer === 'object' && myRegisterer.state === 'Registered');
   await p.addScriptTag({path: c.repo + '/scripts/lab/m0_05_capture.js'});
  }
  if (!c.embeddedAgent) for (const p of s.pages) {await p.locator('#digits').fill('9600001'); await p.locator('#dial').click();}
  if (c.embeddedAgent) return {state: 'AGENT_AND_PEER_REGISTERED', run: c.run};
  await Promise.all(s.pages.map(p => p.waitForFunction(() => alm.pc?.connectionState === 'connected')));
  return {state: 'CONNECTED', run: c.run}; // Capture plus CLI correlation still required.
 }
 const s = page.alm7Run;
 if (!s) throw Error('No started run');
 if (c.phase === 'capture') {
  if (s.result.capture) throw Error('Capture already exists; use a fresh run');
  s.result.before = await Promise.all(s.pages.map(p => p.evaluate(() => alm.snapshot())));
  s.result.capture = await Promise.all(s.pages.map(p => p.evaluate(() => alm.capture(12))));
  s.result.after = await Promise.all(s.pages.map(p => p.evaluate(() => alm.snapshot())));
  for (const [i, p] of s.pages.entries()) for (const name of ['local', 'remote']) {
   const wait = (i === 0 && s.agentOwner ? s.agentOwner : p).waitForEvent('download'); await p.evaluate(n => alm.download(n), name);
   await (await wait).saveAs(s.config.evidence + '/' + s.config.run + '-' +
    s.result.participants[i].id + '-' + name + '.wav');
  }
  return {state: 'CAPTURED', run: s.config.run, capture: s.result.capture,
   transport: s.result.after.map(r => ({ice: r.ice, connection: r.connection}))};
 }
 try {
  for (const p of s.pages) {
   if (await p.evaluate(() => typeof incall !== 'undefined' && incall)) await p.locator('#dial').click();
   await p.waitForTimeout(500);
   if (await p.evaluate(() => typeof myRegisterer !== 'undefined' && myRegisterer.state === 'Registered')) {
    await p.locator('#unregister').click();
    await p.waitForFunction(() => myRegisterer.state === 'Unregistered');
   }
  }
  if (s.pages[0]) {const wait = (s.agentOwner || s.pages[0]).waitForEvent('download');
   await s.pages[0].evaluate(value => alm.downloadJson(value), s.result);
   await (await wait).saveAs(s.config.evidence + '/' + s.config.run + '-browser.json');}
 } finally {
  for (const [i, ctx] of s.contexts.entries()) {
   try {await ctx.tracing.stop({path: s.config.evidence + '/' + s.config.run + '-trace-' + i + '.zip'});}
   finally {await ctx.close();}
  }
  delete page.alm7Run;
 }
 return {state: 'BROWSERS_CLOSED', note: 'Verify Asterisk channels/conferences/contacts independently'};
}
