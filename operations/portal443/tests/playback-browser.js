async (page) => {
 const browser=page.context().browser(), results=[];
 for(const test of ['cached-range-head','embedded-audio-seek','direct-navigation','unauthenticated']) {
  const context=await browser.newContext(), p=await context.newPage();
  const cdp=await context.newCDPSession(p), challenges=[], errors=[];
  p.on('console',m=>{if(m.type()==='error')errors.push(m.text())});
  cdp.on('Fetch.requestPaused',e=>cdp.send('Fetch.continueRequest',{requestId:e.requestId}));
  cdp.on('Fetch.authRequired',async e=>{
   const initial=test!=='unauthenticated' && e.request.url.endsWith('/vicidial/admin.php') && challenges.length===0;
   challenges.push({url:e.request.url,realm:e.authChallenge.realm,initial});
   await cdp.send('Fetch.continueWithAuth',{requestId:e.requestId,authChallengeResponse:initial?
    {response:'ProvideCredentials',username:'synthetic',password:'synthetic-pass'}:{response:'CancelAuth'}});
  });
  await cdp.send('Fetch.enable',{handleAuthRequests:true});
  let result={};
  if(test==='unauthenticated') {
   try{result.status=(await p.goto('http://127.0.0.1:18766/vicidial/recordings/20261107-024337_SYNTHETIC-all.wav')).status()}
   catch(e){result.denied=String(e).slice(0,120)}
  } else {
   result.initial=(await p.goto('http://127.0.0.1:18766/vicidial/admin.php')).status();
   if(test==='cached-range-head') {
    result.responses=await p.evaluate(async()=>{
     const url=document.querySelector('#recording').getAttribute('href');
     const r=await fetch(url,{headers:{Range:'bytes=44-143'}}),h=await fetch(url,{method:'HEAD'});
     return {status:r.status,range:r.headers.get('Content-Range'),bytes:(await r.arrayBuffer()).byteLength,
      head:h.status,length:h.headers.get('Content-Length')};
    });
   } else {
    if(test==='direct-navigation')await p.click('#recording');
    result.media=await p.evaluate(()=>new Promise(resolve=>{
     const a=document.querySelector('audio,video');
     if(!a){resolve({missingMedia:true});return}
     const seek=()=>{a.onseeked=()=>resolve({duration:a.duration,time:a.currentTime,ready:a.readyState});a.currentTime=1.5};
     a.onerror=()=>resolve({error:a.error?.code});
     if(a.readyState>=1)seek();else a.onloadedmetadata=seek;
     a.load();setTimeout(()=>resolve({timeout:true}),7000);
    }));
   }
  }
  results.push({test,result,challenges,errors});await context.close();
 }
 return {browser:browser.version(),syntheticOnly:true,results};
}
