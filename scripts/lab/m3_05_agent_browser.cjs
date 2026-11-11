// Real visible VICIdial controls against the bounded clone; no hidden UI calls.
const fs=require('fs'),cp=require('child_process');
const {chromium}=require('/home/ra/.npm/_npx/51691537fc71f2b0/node_modules/playwright');
const E=process.argv[2], contexts=[], result={pass:false,phases:[],errors:[]};
function guest(mode){return JSON.parse(cp.execFileSync('/usr/bin/python3',[__dirname+'/m3_05_agent.py',mode,E],{encoding:'utf8',timeout:40000}));}
function phase(name){result.phase=name;result.phases.push({name,utc:new Date().toISOString(),state:guest('state')});}
function dispatch(){return new Promise((resolve,reject)=>cp.execFile('/usr/bin/python3',[__dirname+'/m3_05_agent.py','dispatch',E],{encoding:'utf8',timeout:40000},(error,stdout,stderr)=>error?reject(Error(stderr.slice(-1800))):resolve(JSON.parse(stdout))));}
async function pump(seconds=5){for(let n=0;n<seconds;n++){await dispatch();await new Promise(r=>setTimeout(r,1000));}}
(async()=>{let agent, customer;
 try{
 const secret=JSON.parse(cp.execFileSync('/snap/bin/multipass',['exec','almadial-alm16-rpm','--','sudo','cat','/opt/almadial-alm27/credentials.json'],{encoding:'utf8',timeout:20000}));
 for(const name of ['customer','agent']){
 const context=await chromium.launchPersistentContext(E+'/profile-'+name,{executablePath:'/opt/google/chrome/chrome',headless:true,ignoreHTTPSErrors:false,permissions:['microphone'],env:{...process.env,HOME:E+'/browser-home'},args:['--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream','--use-file-for-fake-audio-capture='+E+'/signal-a.wav','--autoplay-policy=no-user-gesture-required','--disable-background-timer-throttling','--disable-renderer-backgrounding','--disable-features=WebRtcHideLocalIpsWithMdns']});contexts.push(context);
 const p=await context.newPage();p.on('dialog',d=>d.accept());p.on('pageerror',e=>result.errors.push(e.name+': '+e.message.slice(0,160)));
 if(name==='customer'){customer=p;const response=await p.goto('https://localhost:8443/alm27-phone/index.php');result.customerHttp=response.status();await p.locator('[name=password]').fill(secret['9062']);await p.getByRole('button',{name:'Open phone'}).click();await p.getByRole('button',{name:'Connect phone'}).click();await p.waitForFunction(()=>document.querySelector('#reg_status')?.value.startsWith('Registered'),{},{timeout:25000});}
 else agent=p;
 }
 phase('customer-registered');
 // Deliberate transport fault only; recovery uses the visible Register control.
 await customer.evaluate(()=>myUserAgent.transport.disconnect());
 await customer.waitForFunction(()=>myUserAgent.transport.state==='Disconnected');
 await customer.locator('#register').click();
 await customer.waitForFunction(()=>myUserAgent.transport.state==='Connected'&&document.querySelector('#reg_status')?.value.startsWith('Registered'),{},{timeout:25000});
 phase('customer-reconnected');
 await agent.goto('https://localhost:8443/alm27-agc/vicidial.php');
 for(let step=0;step<4;step++){
 for(const [name,value] of Object.entries({phone_login:'9061',phone_pass:secret['9061'],VD_login:'alm27qa',VD_pass:secret.agent})){
 const l=agent.locator('[name="'+name+'"]');if(await l.count()&&await l.first().isVisible())await l.first().fill(value);
 }
 const campaign=agent.locator('[name=VD_campaign]');if(await campaign.count()&&await campaign.isVisible()){if(await campaign.evaluate(e=>e.tagName)==='SELECT'){await campaign.focus();await campaign.selectOption('ALM27');}else await campaign.fill('ALM27');}
 const submit=agent.locator('input[type=submit],button[type=submit]');if(await submit.count()&&await submit.first().isVisible()){await submit.first().click();await agent.waitForTimeout(1800);}else break;
 }
 phase('agent-authenticated');
 // Phone is the application's own embedded VICIphone; register through its button.
 let phone;
 for(let n=0;n<15;n++){phone=agent.frames().find(f=>f.url().includes('viciphone.php'));if(phone)break;await agent.waitForTimeout(1000);}
 if(!phone)throw Error('agent VICIphone frame absent');
 const register=phone.locator('#register');if(await register.isVisible())await register.click();
 await phone.waitForFunction(()=>document.querySelector('#reg_status')?.value.startsWith('Registered'),{},{timeout:25000});
 await pump(3);
 await agent.getByText('Call Agent Webphone',{exact:false}).click();
 // Originate may wait for an answer. Keep the browser driver responsive so
 // an incoming agent call can be answered with its actual visible control.
 const joining=pump(8);
 await phone.waitForFunction(()=>typeof mySession!=='undefined'&&(mySession?.state==='Established'||ringing),{},{timeout:25000});
 if(await phone.evaluate(()=>ringing))await phone.locator('#dial').click();
 await joining;phase('agent-conference');
 if(!guest('state').channels.includes('PJSIP/9061-'))throw Error('agent conference channel absent');
 const back=agent.locator('#NoneInSessionBox').getByText('Go Back',{exact:true});if(await back.isVisible())await back.click();
 await agent.getByText('MANUAL DIAL',{exact:true}).click();await agent.locator('#MDLeadIDEntry').fill('27');
 const code=agent.locator('#MDDiaLCodE');if(await code.isVisible())await code.fill('');
 await agent.getByText('Dial Now',{exact:true}).click();await pump(10);phase('customer-dialed');
 await customer.waitForFunction(()=>typeof ringing!=='undefined'&&ringing,{},{timeout:5000});
 await customer.locator('#dial').click();
 await customer.waitForFunction(()=>typeof mySession!=='undefined'&&mySession?.state==='Established',{},{timeout:15000});
 await pump(2);
 // The Local Dial(G) leg ends when its two real PJSIP legs enter ConfBridge.
 // Dismiss the application's resulting notification using its visible control;
 // server-side phase evidence still requires the live customer channel.
 const gone=agent.locator('#CustomerGoneBox');
 if(await gone.isVisible())await gone.getByRole('link',{name:'Go Back',exact:true}).click();
 phase('customer-answered');
 await agent.locator('#XferControl').click();await agent.locator('#xfernumber').fill('9600001');await agent.locator('#DialBlindTransfer a').click();await pump(6);phase('blind-transfer');
 if(!guest('state').channels.includes('9600001'))throw Error('transfer destination not observed');
 await customer.locator('#dial').click();await pump(3);phase('customer-hangup');
 const disposition=agent.locator('#DispoSelectBox');
 if(await disposition.isVisible()){await disposition.getByRole('link',{name:'DC - Disconnected Number',exact:true}).click();await disposition.getByRole('link',{name:'SUBMIT',exact:true}).click();await pump(3);phase('call-disposition');}
 // Explicit agent-side hangup is distinct from the customer hangup and logout.
 await phone.locator('#dial').click();await pump(2);phase('agent-hangup');
 if(guest('state').channels.includes('PJSIP/9061-'))throw Error('agent phone hangup left its channel active');
 const logout=agent.getByRole('link',{name:'LOGOUT',exact:true});await logout.click();await pump(5);phase('agent-logout');
 result.pass=!guest('state').channels&&guest('state').reservations==='0';
 }catch(e){result.error=String(e).split('Call log:')[0].slice(0,650);if(!agent)agent=customer;if(agent){result.visibleText=(await agent.locator('body').innerText()).slice(0,7500);result.frames=agent.frames().map(f=>f.url().split('?')[0]);result.campaignOptions=await agent.locator('[name=VD_campaign] option').evaluateAll(es=>es.map(e=>({value:e.value,text:e.textContent})));result.inputs=await agent.locator('input,select').evaluateAll(es=>es.map(e=>({name:e.name,id:e.id,type:e.type,visible:!!e.offsetParent})));}try{result.failedState=guest('state');}catch{}}
 finally{for(const c of contexts)await c.close();fs.writeFileSync(E+'/agent-browser.json',JSON.stringify(result,null,2),{flag:'wx',mode:0o600});console.log(JSON.stringify({pass:result.pass,phase:result.phase,error:result.error}));if(!result.pass)process.exitCode=1;}
})().catch(e=>{console.error(e.name);process.exitCode=1;});
