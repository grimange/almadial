//go:build linux

package provision

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed database_files.py
var databaseFiles string

type DatabaseNative struct {
	Native Native
	Budget RecordingBudget
	Source DatabaseSource
}

func dbCommand(exe string, args ...string) execution.Command {
	return execution.Command{Executable: exe, Args: args, Timeout: 3 * time.Minute, StdoutLimit: 4 << 20, StderrLimit: 64 << 10}
}
func dbFileCommand(mode string) execution.Command {
	return dbCommand("/usr/bin/python3", "-B", "-c", databaseFiles, mode)
}
func dbClient(args ...string) execution.Command {
	return dbCommand("/usr/bin/mariadb", append([]string{"--no-defaults", "--protocol=SOCKET", "--socket=/var/lib/mysql/mysql.sock", "--user=root", "--batch", "--skip-column-names", "--raw", "--connect-timeout=5"}, args...)...)
}

func databaseSecrets() ([]string, error) {
	values := []string{}
	for _, p := range []string{DatabaseSecret, DatabaseAdminSecret} {
		f, e := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if e != nil {
			return nil, errDatabase
		}
		st, e := f.Stat()
		if e != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() != 49 {
			f.Close()
			return nil, errDatabase
		}
		sys, ok := st.Sys().(*syscall.Stat_t)
		if !ok || sys.Uid != 0 || sys.Gid != 0 || sys.Nlink != 1 {
			f.Close()
			return nil, errDatabase
		}
		b := make([]byte, 49)
		count, e := f.Read(b)
		f.Close()
		if e != nil || count != 49 || b[48] != '\n' {
			return nil, errDatabase
		}
		for _, c := range b[:48] {
			if !strings.ContainsRune("0123456789abcdef", rune(c)) {
				return nil, errDatabase
			}
		}
		values = append(values, string(b[:48]))
	}
	return values, nil
}
func (n DatabaseNative) sql(ctx context.Context, query string) (string, error) {
	sensitive, _ := databaseSecrets()
	r, e := execution.NewRunner(sensitive)
	if e != nil {
		return "", e
	}
	result, e := r.RunInput(ctx, dbClient(), []byte(query))
	if e != nil {
		return "", e
	}
	return databaseSafeSQLResult(result.Stdout), nil
}

const databaseCatalogQuery = `SELECT CONCAT('T:',JSON_ARRAY(TABLE_NAME,ENGINE,TABLE_COLLATION)) FROM information_schema.TABLES WHERE TABLE_SCHEMA='asterisk' ORDER BY BINARY TABLE_NAME;
SELECT CONCAT('C:',JSON_ARRAY(TABLE_NAME,ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT,EXTRA,CHARACTER_SET_NAME,COLLATION_NAME)) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='asterisk' ORDER BY BINARY TABLE_NAME,ORDINAL_POSITION;
SELECT CONCAT('I:',JSON_ARRAY(TABLE_NAME,INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,COLUMN_NAME,COLLATION,SUB_PART,NULLABLE,INDEX_TYPE)) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='asterisk' ORDER BY BINARY TABLE_NAME,BINARY INDEX_NAME,SEQ_IN_INDEX;
SELECT CONCAT('V:',JSON_ARRAY(TABLE_NAME,VIEW_DEFINITION,CHECK_OPTION,IS_UPDATABLE,SECURITY_TYPE)) FROM information_schema.VIEWS WHERE TABLE_SCHEMA='asterisk' ORDER BY BINARY TABLE_NAME;
SELECT CONCAT('R:',JSON_ARRAY(ROUTINE_NAME,ROUTINE_TYPE,ROUTINE_DEFINITION)) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA='asterisk' ORDER BY BINARY ROUTINE_NAME;
SELECT CONCAT('G:',JSON_ARRAY(TRIGGER_NAME,EVENT_MANIPULATION,EVENT_OBJECT_TABLE,ACTION_STATEMENT,ACTION_TIMING)) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA='asterisk' ORDER BY BINARY TRIGGER_NAME;
`

var databaseSeedTables = []string{"system_settings", "vicidial_call_menu", "vicidial_call_menu_options", "vicidial_call_times", "vicidial_conf_templates", "vicidial_configuration", "vicidial_custom_leadloader_templates", "vicidial_drop_rate_groups", "vicidial_inbound_dids", "vicidial_inbound_groups", "vicidial_lead_filters", "vicidial_lists", "vicidial_music_on_hold", "vicidial_music_on_hold_files", "vicidial_override_ids", "vicidial_process_triggers", "vicidial_qc_codes", "vicidial_screen_colors", "vicidial_scripts", "vicidial_server_carriers", "vicidial_settings_containers", "vicidial_shifts", "vicidial_state_call_times", "vicidial_status_categories", "vicidial_statuses", "vicidial_user_groups", "vicidial_users", "wallboard_reports", "wallboard_widgets", "servers", "phones", "conferences", "vicidial_conferences"}

func (n DatabaseNative) InspectDatabase(ctx context.Context) (DatabaseSnapshot, error) {
	var s DatabaseSnapshot
	f, e := n.Native.InspectSystem(ctx)
	if e != nil {
		return s, e
	}
	s.Host = f.RPM.Host
	s.Foundation = f.Digest()
	s.EnvironmentOK = systemSatisfied(f, n.Budget)
	r, e := n.Native.Runner.Run(ctx, dbFileCommand("inspect"))
	if e != nil {
		return s, e
	}
	var fields struct {
		Config, Secrets, CredentialID, ConfigInputs, ServiceID string
		Initialized, Active, Enabled                           bool
	}
	if decode(r.Stdout, &fields) != nil {
		return s, errDatabase
	}
	s.Config, s.Secrets, s.CredentialID, s.ConfigInputs, s.ServiceID = fields.Config, fields.Secrets, fields.CredentialID, fields.ConfigInputs, fields.ServiceID
	s.Initialized, s.Active, s.Enabled = fields.Initialized, fields.Active, fields.Enabled
	if !s.Active {
		return s, nil
	}
	runtime, e := n.sql(ctx, `SELECT CONCAT_WS('|',VERSION(),@@bind_address,@@port,@@socket,@@character_set_server,@@collation_server,@@sql_mode,@@general_log,@@slow_query_log,@@log_bin);`)
	if e != nil {
		return s, e
	}
	want := "10.11.18-MariaDB|127.0.0.1|3306|/var/lib/mysql/mysql.sock|utf8mb3|utf8mb3_unicode_ci|STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION|OFF|OFF|OFF"
	s.RuntimeOK = runtime == want
	listeners, e := n.Native.Runner.Run(ctx, dbCommand("/usr/sbin/ss", "-H", "-lntp"))
	if e != nil {
		return s, e
	}
	seen := false
	for _, line := range strings.Split(listeners.Stdout, "\n") {
		v := strings.Fields(line)
		if len(v) > 3 && (strings.HasSuffix(v[3], ":3306") || strings.Contains(line, "mariadbd")) {
			if v[3] != "127.0.0.1:3306" {
				s.RuntimeOK = false
			}
			seen = true
		}
	}
	s.RuntimeOK = s.RuntimeOK && seen
	rootAuth, authErr := n.sql(ctx, "SELECT COUNT(*) FROM mysql.global_priv WHERE User='root' AND Host='localhost' AND JSON_SEARCH(Priv,'one','unix_socket') IS NOT NULL;")
	s.RuntimeOK = s.RuntimeOK && authErr == nil && rootAuth == "1"
	labels, labelsErr := n.Native.Runner.Run(ctx, dbCommand("/usr/sbin/matchpathcon", "-V", "/var/lib/mysql/mysql.sock", databaseConfigPath, DatabaseSecret, DatabaseAdminSecret, "/etc/almadial/secrets/database-client.cnf"))
	s.RuntimeOK = s.RuntimeOK && labelsErr == nil && labels.ExitCode == 0

	x, e := n.sql(ctx, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='asterisk'; SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='asterisk'; SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='asterisk'; SELECT (SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA='asterisk')+(SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA='asterisk');")
	if e != nil {
		return s, e
	}
	var exists int
	if _, e = fmt.Sscanf(x, "%d\n%d\n%d\n%d", &exists, &s.Tables, &s.Indexes, &s.OtherObjects); e != nil {
		return s, errDatabase
	}
	s.Exists = exists == 1
	s.SchemaVerified = true
	account, e := n.sql(ctx, "SELECT COUNT(*) FROM mysql.user WHERE User='almadial';")
	if e != nil {
		return s, e
	}
	s.AccountExists = account != "0"
	if s.AccountExists {
		grants, e := n.sql(ctx, `SELECT CONCAT_WS('|',User,Host) FROM mysql.user WHERE User='almadial' ORDER BY Host;
SELECT PRIVILEGE_TYPE FROM information_schema.USER_PRIVILEGES WHERE GRANTEE="'almadial'@'localhost'" AND PRIVILEGE_TYPE<>'USAGE' ORDER BY PRIVILEGE_TYPE;
SELECT CONCAT_WS('|',TABLE_SCHEMA,PRIVILEGE_TYPE,IS_GRANTABLE) FROM information_schema.SCHEMA_PRIVILEGES WHERE GRANTEE="'almadial'@'localhost'" ORDER BY TABLE_SCHEMA,PRIVILEGE_TYPE;
SELECT COUNT(*) FROM information_schema.TABLE_PRIVILEGES WHERE GRANTEE="'almadial'@'localhost'";
SELECT COUNT(*) FROM information_schema.COLUMN_PRIVILEGES WHERE GRANTEE="'almadial'@'localhost'";
SELECT COUNT(*) FROM mysql.procs_priv WHERE User='almadial';
SELECT COUNT(*) FROM mysql.roles_mapping WHERE User='almadial';`)
		if e != nil {
			return s, e
		}
		s.GrantsOK = grants == "almadial|localhost\nasterisk|DELETE|NO\nasterisk|INSERT|NO\nasterisk|LOCK TABLES|NO\nasterisk|SELECT|NO\nasterisk|UPDATE|NO\n0\n0\n0\n0"
		if s.Secrets == "valid" {
			runner, _ := execution.NewRunner(nil)
			c := dbCommand("/usr/bin/mariadb", "--defaults-file=/etc/almadial/secrets/database-client.cnf", "--protocol=SOCKET", "--socket=/var/lib/mysql/mysql.sock", "--batch", "--skip-column-names", "--connect-timeout=5", "--database=asterisk", "--execute=SELECT 1")
			r, e := runner.Run(ctx, c)
			s.ApplicationAuthOK = e == nil && strings.TrimSpace(r.Stdout) == "1"
		}
	}
	if s.Tables == 0 {
		return s, nil
	}
	catalog, e := n.sql(ctx, databaseCatalogQuery)
	if e != nil {
		return s, e
	}
	s.CatalogSHA = rawSHA([]byte(catalog))
	ver, e := n.sql(ctx, "SELECT db_schema_version FROM asterisk.system_settings;")
	if e == nil {
		s.Version = ver
	}
	seedQuery := ""
	for _, table := range databaseSeedTables {
		seedQuery += "SELECT COUNT(*)>0 FROM asterisk.`" + table + "`;\n"
	}
	seedValues, seedErr := n.sql(ctx, seedQuery)
	s.SeedsOK = seedErr == nil && seedValues == strings.TrimSuffix(strings.Repeat("1\n", len(databaseSeedTables)), "\n")
	charset, charsetErr := n.sql(ctx, "SELECT CONCAT_WS('|',DEFAULT_CHARACTER_SET_NAME,DEFAULT_COLLATION_NAME) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='asterisk';")
	s.SeedsOK = s.SeedsOK && charsetErr == nil && charset == "utf8mb3|utf8mb3_unicode_ci"
	keys, e := n.sql(ctx, `SELECT COUNT(*) FROM asterisk.vicidial_users WHERE user='6666'; SELECT COUNT(*) FROM asterisk.vicidial_user_groups WHERE user_group='ADMIN'; SELECT COUNT(*) FROM asterisk.vicidial_statuses WHERE status IN ('NEW','SALE'); SELECT COUNT(*) FROM asterisk.vicidial_qc_codes WHERE code IN ('QCPASS','QCFAIL','QCCANCEL'); SELECT COUNT(*) FROM asterisk.servers WHERE server_id='TESTast';`)
	s.SeedsOK = s.SeedsOK && e == nil && keys == "1\n1\n2\n3\n1"
	if sec, e := databaseSecrets(); e == nil {
		ok, e := n.sql(ctx, "SELECT COUNT(*) FROM asterisk.vicidial_users WHERE user='6666' AND pass='"+sec[1]+"';")
		s.AdminCredentialOK = e == nil && ok == "1"
	}
	return s, nil
}

// Validate accepts only the immutable S03 graph's fixed commands. RunInput
// supplies trusted SQL bytes transiently; secrets never enter Command or Trace.
type databaseRunner struct {
	native  DatabaseNative
	actions map[string]string
	source  string
}

func (r databaseRunner) Validate(c execution.Command) error {
	if _, ok := r.actions[jsonString(c)]; !ok || r.source != r.native.Source.Digest() {
		return errDatabase
	}
	return r.native.Native.Runner.Validate(c)
}
func (r databaseRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if e := r.Validate(c); e != nil {
		return execution.Result{}, e
	}
	id := r.actions[jsonString(c)]
	if id == "database.02.service" {
		// Check effective options before starting, so conflicting administrator
		// configuration cannot briefly expose a public database listener.
		v, e := r.native.Native.Runner.Run(ctx, dbCommand("/usr/libexec/mariadbd", "--print-defaults"))
		if e != nil {
			return execution.Result{}, e
		}
		bind, socket := "", ""
		for _, v := range strings.Fields(v.Stdout) {
			if strings.HasPrefix(v, "--bind-address=") {
				bind = strings.TrimPrefix(v, "--bind-address=")
			}
			if strings.HasPrefix(v, "--socket=") {
				socket = strings.TrimPrefix(v, "--socket=")
			}
			if v == "--skip-networking" {
				return execution.Result{}, errDatabase
			}
		}
		if bind != "127.0.0.1" || socket != "/var/lib/mysql/mysql.sock" {
			return execution.Result{}, errDatabase
		}
	}
	if id != "database.03.bootstrap" && id != "database.04.grants" {
		return r.native.Native.Runner.Run(ctx, c)
	}
	sec, e := databaseSecrets()
	if e != nil {
		return execution.Result{}, errDatabase
	}
	var sql string
	if id == "database.03.bootstrap" {
		if r.native.Source.Digest() == "" {
			return execution.Result{}, errDatabase
		}
		s, e := r.native.InspectDatabase(ctx)
		if e != nil || s.State() != DatabaseEmpty {
			return execution.Result{}, errDatabase
		}
		if !s.Exists {
			sql = "CREATE DATABASE asterisk CHARACTER SET utf8mb3 COLLATE utf8mb3_unicode_ci;\n"
		}
		sql += "USE asterisk;\n" + r.native.Source.create + "\n" + r.native.Source.seed + "\nUPDATE vicidial_users SET pass='" + sec[1] + "' WHERE user='6666';\n"
	} else {
		s, e := r.native.InspectDatabase(ctx)
		if e != nil || s.State() != DatabaseComplete || s.AccountExists {
			return execution.Result{}, errDatabase
		}
		sql = "CREATE USER 'almadial'@'localhost' IDENTIFIED BY '" + sec[0] + "'; GRANT SELECT,INSERT,UPDATE,DELETE,LOCK TABLES ON asterisk.* TO 'almadial'@'localhost';\n"
	}
	runner, e := execution.NewRunner(sec)
	if e != nil {
		return execution.Result{}, e
	}
	out, e := runner.RunInput(ctx, c, []byte(sql))
	out.Stdout = ""
	out.Stderr = ""
	return out, e
}
