package provision

import (
	"errors"
	"fmt"
	"strings"
)

var errRuntime = errors.New("S06 runtime contract failed; preserve state and reconcile the reported conflict")

// RuntimeComponent is the fixed ALM-21 laboratory contract. Deferred schedules
// are documentation, never executable cron lines. Activating them requires a
// new reviewed policy: native keepalive would compete with systemd FastAGI.
type RuntimeComponent struct {
	ID, Classification, Source, Command, User, Owner, Policy, Dependencies, Postcondition string
}

func RuntimeInventory() []RuntimeComponent {
	return []RuntimeComponent{
		{"fastagi", "REQUIRED_ACTIVE", "r4037 bin/FastAGI_log.pl; M0-06/M0-08", "/usr/bin/perl /usr/share/astguiclient/almadial-s06-FastAGI_log.pl", "asterisk", "almadial-s06-fastagi.service", "Restart=on-failure; enabled", "mariadb.service; protected configuration; DBI; Net::Server", "one parent and bounded prefork pool in unit cgroup; loopback 4577; database ready"},
		{"updater", "REQUIRED_CONFIGURED", "r4037 bin/AST_update.pl; M0-07/M0-08", "/usr/bin/perl /usr/share/astguiclient/almadial-s06-AST_update.pl", "asterisk", "almadial-s06-updater.service", "disabled/inactive; runtime AMI authorization deferred", "mariadb.service; asterisk.service; reviewed AMI Command policy", "unit and source configured; no process"},
		{"listener", "REQUIRED_CONFIGURED", "r4037 bin/AST_manager_listen.pl; M0-07/M0-08", "/usr/bin/perl /usr/share/astguiclient/almadial-s06-AST_manager_listen.pl", "asterisk", "almadial-s06-listener.service", "disabled/inactive; runtime AMI authorization deferred", "mariadb.service; asterisk.service; reviewed AMI Command policy", "unit and source configured; no process"},
		{"native-keepalive", "DEFERRED", "r4037 docs/BASE_INSTALL.txt:432-433", "/usr/share/astguiclient/ADMIN_keepalive_ALL.pl", "root", "none while S06 systemd owns FastAGI", "* * * * *; commented only; incompatible with concurrent systemd ownership", "separate activation review including maintenance side effects", "no executable cron definition; no process"},
		{"hopper", "DEFERRED", "r4037 docs/BASE_INSTALL.txt:466-467", "/usr/share/astguiclient/AST_VDhopper.pl -q", "root", "cron (activation deferred)", "* * * * *; commented only", "campaign/data mutation authorization", "no executable cron definition; no process"},
		{"foundation", "ALREADY_SATISFIED", "Accepted ALM-18/19/20", "mariadb; asterisk; httpd; php-fpm; crond", "accepted unit identities", "existing systemd units", "preserve units; enable required web services for reboot", "fresh S05 foundation checks", "existing services healthy; listeners remain local; SELinux Enforcing"},
	}
}

// RuntimeCron returns one exact, owned, distribution cron.d file. There are zero
// active application jobs in this explicitly authorized minimal contract.
func RuntimeCron() string {
	return "# AlmaDial S06 / ALM-21: dialing-inactive laboratory contract\n" +
		"# No active application cron jobs are authorized. Do not uncomment these\n" +
		"# examples without a new activation and process-ownership review.\n" +
		"SHELL=/bin/bash\nPATH=/usr/sbin:/usr/bin:/sbin:/bin\nCRON_TZ=UTC\nMAILTO=\"\"\n" +
		"# native-keepalive DEFERRED: r4037 docs/BASE_INSTALL.txt; conflicts with systemd FastAGI\n" +
		"# * * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl\n" +
		"# hopper DEFERRED: r4037 docs/BASE_INSTALL.txt; campaign processing inactive\n" +
		"# * * * * * root /usr/share/astguiclient/AST_VDhopper.pl -q\n"
}

func runtimeUnit(id, script string, active bool) string {
	dependencies := "mariadb.service asterisk.service"
	if active {
		dependencies = "mariadb.service"
	}
	guard := ""
	if !active {
		guard = "RefuseManualStart=yes\n"
	}
	return fmt.Sprintf("# AlmaDial S06 / ALM-21\n[Unit]\nDescription=AlmaDial VICIdial %s\nRequires=%s\nAfter=%s\n%sStartLimitIntervalSec=60\nStartLimitBurst=3\n\n[Service]\nType=simple\nUser=asterisk\nGroup=asterisk\nWorkingDirectory=/usr/share/astguiclient\nEnvironment=PATH=/usr/sbin:/usr/bin:/sbin:/bin\nExecStartPre=/usr/bin/perl /usr/share/astguiclient/almadial-s06-ready.pl\nExecStart=/usr/bin/perl /usr/share/astguiclient/almadial-s06-%s\nRestart=on-failure\nRestartSec=5\nKillMode=control-group\nTimeoutStartSec=30\nTimeoutStopSec=30\nNoNewPrivileges=true\nPrivateTmp=true\nUMask=0027\n\n[Install]\nWantedBy=multi-user.target\n", id, dependencies, dependencies, guard, script)
}

func runtimeReplace(text, old, replacement string, count int) (string, error) {
	if strings.Count(text, old) != count {
		return "", errRuntime
	}
	return strings.ReplaceAll(text, old, replacement), nil
}

func (s VICIdialSource) runtimeScript(name string) ([]byte, error) {
	text := string(s.files["bin/"+name])
	if s.Digest() == "" || text == "" {
		return nil, errRuntime
	}
	count := 1
	if name == "FastAGI_log.pl" {
		if rawSHA([]byte(text)) != "f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a" {
			return nil, errRuntime
		}
		count = 2
		for _, p := range [][2]string{
			{"'$now_date_epoch','','','','','$callerid')", "'$now_date_epoch',NULL,0,0,0,'$callerid')"},
			{"port=>4577,", "host=>'127.0.0.1',\n\t\t\t\tport=>4577,"},
			{"user=>'root',", "user=>'asterisk',"},
			{"group=>'root',", "group=>'asterisk',"},
		} {
			var e error
			text, e = runtimeReplace(text, p[0], p[1], 1)
			if e != nil {
				return nil, e
			}
		}
	} else if name != "AST_update.pl" && name != "AST_manager_listen.pl" {
		return nil, errRuntime
	}
	text, e := runtimeReplace(text, "'/etc/astguiclient.conf'", "'/etc/astguiclient-s06.conf'", count)
	return []byte(text), e
}

// This readiness helper makes a bounded local SELECT before starting FastAGI.
// Credentials stay in the existing protected-file mechanism, never argv/logs.
const runtimeReadiness = `use strict;
use warnings;
use DBI;
$SIG{ALRM} = sub { die "local database readiness deadline\n"; };
alarm 10;
open(my $f, '<', '/etc/astguiclient-s06.conf') or die "runtime configuration unavailable\n";
my %c;
while (<$f>) { if (/^(VARDB_server|VARDB_database|VARDB_user|VARDB_pass|VARDB_port)\s*=>\s*(.*?)\s*$/) { $c{$1}=$2; } }
close $f;
die "local runtime configuration required\n" unless ($c{VARDB_server}//'') eq 'localhost' && ($c{VARDB_port}//'') eq '3306' && ($c{VARDB_database}//'') eq 'asterisk';
my $db = DBI->connect("DBI:mysql:database=asterisk;mysql_socket=/var/lib/mysql/mysql.sock", $c{VARDB_user}, $c{VARDB_pass}, {RaiseError=>0,PrintError=>0,mysql_connect_timeout=>5});
die "local database unavailable\n" unless $db;
my ($ok) = $db->selectrow_array('SELECT 1');
die "local database readiness failed\n" unless defined($ok) && $ok == 1;
$db->disconnect;
alarm 0;
`

// Exact preimage of the failed initial laboratory readiness contract.
// Only these two S06-owned files are eligible for the bounded correction.
var runtimeReadinessPrevious = strings.ReplaceAll(strings.ReplaceAll(runtimeReadiness, "eq 'localhost'", "eq '127.0.0.1'"), "mysql_socket=/var/lib/mysql/mysql.sock", "host=127.0.0.1;port=3306")
