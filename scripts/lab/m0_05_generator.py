#!/usr/bin/env python3
"""Derive an isolated r4037 generator experiment. Does not execute or deploy it."""
import argparse
import hashlib
import json
import os
from pathlib import Path

SOURCE_SHA256 = '4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67'
STAGE = '/root/almadial-m005/generated'
OUTPUTS = ('extensions-vicidial.conf', 'iax-vicidial.conf', 'sip-vicidial.conf',
           'pjsip-vicidial.conf', 'pjsip_wizard-vicidial.conf', 'voicemail.conf',
           'musiconhold-vicidial.conf', 'meetme-vicidial.conf', 'confbridge-vicidial.conf')


def correct_gsm_priority(block):
    """r4037 line 3655 mislabels the modern GSM recording route as 8309.

    Apply to a derived source only, before execution. Never edit generated files.
    Recording itself remains outside this experiment's runtime qualification.
    """
    marker = '$Vext .= "exten => 8310,1,Answer\\n";'
    if block.count(marker) != 1:
        raise ValueError('unexpected GSM recording branch')
    prefix, suffix = block.split(marker)
    old = 'exten => 8309,2,MixMonitor('
    if suffix.count(old) != 1:
        raise ValueError('unexpected duplicate priority correction count')
    return prefix + marker + suffix.replace(old, 'exten => 8310,2,MixMonitor(', 1)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def derive(data):
    if digest(data) != SOURCE_SHA256:
        raise ValueError('unreviewed generator source')
    text = data.decode('utf-8')
    start = text.index('if ( ($active_asterisk_server =~ /Y/) && ($generate_vicidial_conf =~ /Y/) && ($rebuild_conf_files =~ /Y/) )')
    end = text.index('#####  END Creation of', start)
    # The preceding complete block includes its closing brace, no keepalive jobs.
    block = text[start:end]
    helpers = text[text.index('sub leading_zero($)'):]
    settings = text[text.index('##### Get the settings from system_settings #####'):text.index('if ($VARactive_keepalives =~ /X/)')]
    reserved = '\n'.join(text.splitlines()[209:213])
    prefix = '''#!/usr/bin/perl
# AlmaDial r4037 isolated generation experiment; not the keepalive daemon.
use DBI;
use File::Compare;
umask 0077;
$ENV{PATH}='/root/almadial-m005/tools:/usr/bin:/bin';
$server_ip='127.0.0.1'; $VARserver_ip=$server_ip;
$PATHmonitor='/var/spool/asterisk/monitor';
$dbhA=DBI->connect('DBI:mysql:database=almadial_m005;mysql_socket=/var/lib/mysql/mysql.sock','root','',{RaiseError=>1,PrintError=>0});
$DB=0;$DBX=0;$DBXXX=0;
'''
    original_block = digest(block.encode())
    block = correct_gsm_priority(block)
    count = block.count('/etc/asterisk/')
    block = block.replace('/etc/asterisk/', STAGE + '/')
    screen_count = block.count('`screen -XS asterisk')
    block = block.replace('`screen -XS asterisk', '`m005-screen -XS asterisk')
    result = (prefix + reserved + '\n' + settings + '\n' + block + '\n' + helpers).encode()
    return result, {'source_sha256':SOURCE_SHA256,'original_block_sha256':original_block,
                    'derived_sha256':digest(result),'path_replacements':count,
                    'suppressed_screen_requests':screen_count,'database':'almadial_m005',
                    'stage':STAGE,'outputs':OUTPUTS,'upstream_algorithm_patches':1,
                    'correction':'r4037 line 3655: 8309 -> 8310 in GSM MixMonitor branch',
                    'recording_path':'/var/spool/asterisk/monitor',
                    'execution':'NOT_RUN','scope':'generation block and source helpers only; reload requests recorded, not dispatched'}


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source',type=Path,required=True)
    p.add_argument('--output-dir',type=Path,required=True)
    a=p.parse_args()
    if a.source.is_symlink(): p.error('source symlink refused')
    result,ledger=derive(a.source.read_bytes())
    a.output_dir.mkdir(mode=0o700)
    for name,content in [('generator.pl',result),('derivation.json',(json.dumps(ledger,indent=2)+'\n').encode())]:
        fd=os.open(a.output_dir/name,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
        with os.fdopen(fd,'wb') as f:f.write(content)
    print(json.dumps(ledger,indent=2))


if __name__=='__main__':main()
