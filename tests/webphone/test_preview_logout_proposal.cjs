// Local-only tests of the actual proposed agent JavaScript. No browser or host access.
const fs=require('fs'),vm=require('vm'),assert=require('node:assert/strict');
const source=fs.readFileSync(process.argv[2],'utf8');
const helper=source.slice(source.indexOf('// PRD-03C proposal: release only'),source.indexOf('// End PRD-03C proposal helper.'));
const start=source.indexOf('function LogouT(tempreason,temppause,temp_pause_max_url_trigger)');
const logout=source.slice(start,source.indexOf('<?php\nif ($useIE > 0)',start)).replace(/<\?php[\s\S]*?\?>/g,'TRANSLATED');
const cases=[];
function setup(){
 const requests=[],alerts=[],nodes={};
 const ctx={Object,encodeURIComponent,console:{log(){}},requests,alerts,
  server_ip:'192.0.2.1',session_name:'synthetic_session',session_id:'9600001',
  user:'synthetic_agent',pass:'FAKE&TEST=ONLY',campaign:'SYNTHETIC',
  previous_dispo:'NEW',previous_called_count:'1',in_lead_preview_state:1,
  manual_dial_in_progress:0,MD_channel_look:0,VD_live_customer_call:0,alt_dial_status_display:0,
  button_click_log:'',SQLdate:'2026-11-09',isVisible:1,aec:0,
  alert_box:message=>alerts.push(message),alert:message=>alerts.push(message),
  onHidden(){},onVisible(){},check_for_conf_calls(){},agent_events(){},
  hideDiv:id=>nodes[id]={hidden:true},showDiv:id=>nodes[id]={visible:true},
  is_webphone:'Y',agent_logout_link:0,agcPAGE:'synthetic.php',
  document:{vicidial_form:{lead_id:{value:'100'}},getElementById(id){return nodes[id]||(nodes[id]={src:'KEEP'});}},
  XMLHttpRequest:class {
   constructor(){requests.push(this);}
   open(method,path){this.method=method;this.path=path;}
   setRequestHeader(){}
   send(body){this.body=body;if(ctx.throwSend)throw Error('synthetic transport failure');}
   respond(status,text){this.status=status;this.responseText=text;this.readyState=4;this.onreadystatechange();}
  }
 };
 // Non-secret native logout fields, irrelevant to these request sequencing tests.
 for(const name of ['extension','protocol','agent_log_id','no_delete_sessions','phone_ip','enable_sipsak_messages','LogouTKicKAlL','ext_context','qm_extension','dial_method','reselect_alt_dial'])ctx[name]='0';
 vm.createContext(ctx);vm.runInContext(helper+'\n'+logout,ctx);
 return {ctx,requests,alerts,nodes};
}
function test(name,fn){fn();cases.push({case:name,result:'PASS'});}
test('release acknowledgement precedes native logout and phone teardown',()=>{
 const {ctx,requests,nodes}=setup();ctx.LogouT('NORMAL','','');
 assert.equal(requests.length,1);assert.match(requests[0].body,/ACTION=manDiaLskip/);
 assert.equal(new URLSearchParams(requests[0].body).get('pass'),'FAKE&TEST=ONLY');
 assert.equal(ctx.in_lead_preview_state,1);assert.equal(nodes.webphone,undefined);
 ctx.LogouT('NORMAL','','');assert.equal(requests.length,1);
 requests[0].respond(200,'PREVIEW RELEASED\n');assert.equal(requests.length,2);
 assert.match(requests[1].body,/ACTION=userLOGout/);assert.equal(nodes.webphone,undefined);
 ctx.LogouT('NORMAL','','');assert.equal(requests.length,2);
 requests[1].respond(200,'1|1|1|1|Local/synthetic|0\n');
 assert.equal(nodes.webphone.src,'./images/blank.gif');assert.equal(nodes.MainPanel.hidden,true);
 assert.equal(ctx.logout_stop_timeouts,1);assert.equal(ctx.document.vicidial_form.lead_id.value,'');
});
for(const [name,change] of [['dial in progress',{MD_channel_look:1}],['active customer call',{VD_live_customer_call:1}],['manual entry or callback',{manual_dial_in_progress:1}],['alternate dial',{alt_dial_status_display:1}]]){
 test(name+' cannot release',()=>{const {ctx,requests,alerts}=setup();Object.assign(ctx,change);ctx.LogouT('NORMAL','','');assert.equal(requests.length,0);assert.equal(alerts.length,1);assert.equal(ctx.in_lead_preview_state,1);});
}
for(const [name,status,body] of [['HTTP denial',403,'denied'],['wrong 200 response',200,'LEAD REVERTED'],['native denial',200,'PREVIEW NOT RELEASED'],['malformed success',200,'PREVIEW RELEASED\nwarning']]){
 test(name+' retains session and claim UI',()=>{const {ctx,requests,nodes,alerts}=setup();ctx.LogouT('NORMAL','','');requests[0].respond(status,body);assert.equal(requests.length,1);assert.equal(ctx.in_lead_preview_state,1);assert.equal(nodes.webphone,undefined);assert.equal(ctx.preview_logout_pending,false);assert.equal(alerts.length,1);});
}
for(const event of ['onerror','ontimeout','onabort']){
 test(event+' permits retry without accepting a late acknowledgement',()=>{const {ctx,requests,nodes}=setup();ctx.LogouT('NORMAL','','');requests[0][event]();requests[0].respond(200,'PREVIEW RELEASED');assert.equal(requests.length,1);assert.equal(nodes.webphone,undefined);ctx.LogouT('NORMAL','','');assert.equal(requests.length,2);});
}
test('synchronous send failure retains preview',()=>{const {ctx,requests}=setup();ctx.throwSend=true;ctx.LogouT('NORMAL','','');assert.equal(requests.length,1);assert.equal(ctx.in_lead_preview_state,1);assert.equal(ctx.preview_logout_pending,false);});
test('ordinary logout without preview sends no release',()=>{const {ctx,requests}=setup();ctx.in_lead_preview_state=0;ctx.LogouT('NORMAL','','');assert.equal(requests.length,1);assert.match(requests[0].body,/ACTION=userLOGout/);});
test('failed native logout does not blank phone or claim completion',()=>{const {ctx,requests,nodes,alerts}=setup();ctx.in_lead_preview_state=0;ctx.LogouT('NORMAL','','');requests[0].respond(200,'PREVIEW NOT RELEASED');assert.equal(nodes.webphone,undefined);assert.equal(ctx.logout_stop_timeouts,undefined);assert.equal(alerts.length,1);assert.equal(ctx.native_logout_pending,false);});
test('native logout parser rejects a generic HTTP 200 body',()=>{const {ctx}=setup();assert.equal(ctx.NativeLogoutResponseOK('OK'),false);assert.equal(ctx.NativeLogoutResponseOK('1|1|1|1|Local/synthetic|0'),true);assert.equal(ctx.NativeLogoutResponseOK('1||1|1||0'),true);});
console.log(JSON.stringify({execution:'actual proposed helper and native LogouT in Node VM with fake XHR; no host/browser runtime',cases},null,2));
