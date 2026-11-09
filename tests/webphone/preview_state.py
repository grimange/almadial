#!/usr/bin/env python3
"""Read-only dialer01 correlation for PRD-03B/C; never performs recovery writes."""
import argparse
import json
from pathlib import Path
import subprocess

SQL = """
SELECT UNIX_TIMESTAMP() epoch,NOW() server_time;
SELECT campaign_id,active,dial_method,auto_dial_level,manual_preview_dial,manual_dial_list_id,no_hopper_leads_logins,local_call_time,campaign_recording,dial_statuses FROM vicidial_campaigns WHERE campaign_id IN('QAOUT01','test01');
SELECT list_id,active,campaign_id FROM vicidial_lists WHERE list_id BETWEEN 910014 AND 910017;
SELECT lead_id,list_id,vendor_lead_code,status,user,called_count,called_since_last_reset,modify_date FROM vicidial_list WHERE list_id=910016 OR lead_id IN(8,69);
SELECT user,campaign_id,status,lead_id,preview_lead_id,conf_exten,agent_log_id FROM vicidial_live_agents;
SELECT lead_id,list_id,status,user FROM vicidial_hopper WHERE campaign_id='QAOUT01';
SELECT lead_id,user,campaign_id,conf_exten,call_type,call_date FROM vicidial_sessions_recent WHERE lead_id BETWEEN 49 AND 68 ORDER BY call_date DESC LIMIT 20;
SELECT user,event,event_date,event_epoch,campaign_id FROM vicidial_user_log WHERE user IN('qaagent14','qasup14') ORDER BY user_log_id DESC LIMIT 18;
SELECT agent_log_id,user,event_time,campaign_id,lead_id,pause_sec,wait_sec,talk_sec,dispo_sec,status,comments FROM vicidial_agent_log WHERE user IN('qaagent14','qasup14') ORDER BY agent_log_id DESC LIMIT 20;
SELECT lead_id,user,previous_status,previous_called_count,event_date FROM vicidial_agent_skip_log WHERE campaign_id='QAOUT01' ORDER BY event_date DESC LIMIT 15;
SELECT COUNT(*) call_records FROM vicidial_log WHERE campaign_id='QAOUT01';
SELECT COUNT(*) recordings FROM recording_log;
SELECT COUNT(*) incall_leads FROM vicidial_list WHERE list_id BETWEEN 910014 AND 910017 AND status='INCALL';
SELECT COUNT(*) live_agents FROM vicidial_live_agents;
SELECT COUNT(*) diagnostic_outbound_calls FROM vicidial_log WHERE lead_id BETWEEN 51 AND 68;
"""

def capture():
    remote = "import subprocess,json,datetime,socket\nassert socket.gethostname()=='dialer01', 'Unexpected host'\n"
    remote += "sql=" + repr(SQL) + "\n"
    remote += "out={'host':socket.gethostname(),'utc':datetime.datetime.now(datetime.timezone.utc).isoformat()}\n"
    remote += "commands={'database':['mariadb','--no-defaults','-B','asterisk','-e',sql]}\n"
    remote += "commands.update({c:['asterisk','-rx',c] for c in ['core show channels count','confbridge list','sip show peers']})\n"
    remote += "for k,a in commands.items():\n p=subprocess.run(a,capture_output=True,text=True,timeout=25);out[k]={'rc':p.returncode,'stdout':p.stdout,'stderr':p.stderr}\n"
    remote += "print(json.dumps(out))\n"
    p = subprocess.run(['ssh','-F','/dev/null','-T','-p','22222','-o','BatchMode=yes','root@2.29.153.161','python3 -B -'], input=remote, text=True, capture_output=True, check=True, timeout=45)
    result=json.loads(p.stdout)
    if result['database']['rc']: raise RuntimeError('Read-only state query failed')
    return result

if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output',type=Path)
    args=parser.parse_args()
    value=capture()
    args.output.write_text(json.dumps(value,indent=2)+'\n')
    print(json.dumps(value,indent=2))
