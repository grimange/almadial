//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"strings"
)

// Only application-owned columns are selected. Secrets are compared by digest,
// never emitted by MariaDB. Cardinality guards prevent updating an unexpected
// server or a second system_settings row.
const vicidialRegistrationQuery = `SELECT JSON_OBJECT('Servers',(SELECT COUNT(*) FROM asterisk.servers),'Settings',(SELECT COUNT(*) FROM asterisk.system_settings),'ID',server_id,'IP',server_ip,'Active',active,'Asterisk',active_asterisk_server,'Generate',generate_vicidial_conf,'Rebuild',rebuild_conf_files,'Version',asterisk_version,'Conference',conf_engine,'User',ASTmgrUSERNAME,'Update',ASTmgrUSERNAMEupdate,'Listen',ASTmgrUSERNAMElisten,'Send',ASTmgrUSERNAMEsend,'Secret',SHA2(ASTmgrSECRET,256),'Webroot',(SELECT webroot_writable FROM asterisk.system_settings LIMIT 1),'PassKey',(SELECT SHA2(COALESCE(pass_key,''),256) FROM asterisk.system_settings LIMIT 1),'PassKeyValid',(SELECT IF(pass_key REGEXP '^[A-Za-z0-9]{16,100}$',1,0) FROM asterisk.system_settings LIMIT 1)) FROM asterisk.servers ORDER BY server_id;`

type vicidialRegistration struct {
	PassKey                                                          string
	PassKeyValid                                                     int
	Servers, Settings                                                int
	ID, IP, Active, Asterisk, Generate, Rebuild, Version, Conference string
	User, Update, Listen, Send, Secret, Webroot                      string
}

func vicidialSeedRegistration() vicidialRegistration {
	return vicidialRegistration{PassKey: rawSHA(nil), Servers: 1, Settings: 1, ID: "TESTast", IP: "10.10.10.15", Active: "Y", Asterisk: "Y", Generate: "Y", Rebuild: "Y", Version: "1.4.21.2", Conference: "MEETME", User: "cron", Update: "updatecron", Listen: "listencron", Send: "sendcron", Secret: rawSHA([]byte("1234")), Webroot: "1"}
}

func (r vicidialRegistration) state(ami string) (string, error) {
	if !shaPattern.MatchString(r.PassKey) || (r.PassKeyValid != 0 && r.PassKeyValid != 1) || (r.PassKeyValid == 0 && r.PassKey != rawSHA(nil)) || !vicidialHex(ami, 20) || r.Servers != 1 || r.Settings != 1 || r.ID != "TESTast" || r.Active != "Y" || r.Asterisk != "Y" || r.Generate != "Y" || (r.Rebuild != "Y" && r.Rebuild != "N") {
		return "", errVICIdial
	}
	seed := r.IP == "10.10.10.15" && r.Version == "1.4.21.2" && r.Conference == "MEETME" && r.User == "cron" && r.Update == "updatecron" && r.Listen == "listencron" && r.Send == "sendcron" && r.Secret == rawSHA([]byte("1234")) && r.Rebuild == "Y"
	registered := r.IP == "127.0.0.1" && r.Version == "22.12.0" && r.Conference == "CONFBRIDGE" && r.User == "almadial-s05" && r.Update == r.User && r.Listen == r.User && r.Send == r.User && r.Secret == rawSHA([]byte(ami))
	if seed && r.Webroot == "1" {
		return "absent", nil
	}
	if registered && (r.Webroot == "1" || r.PassKey == rawSHA(nil)) {
		return "partial", nil
	}
	if registered && r.Webroot == "0" && r.PassKeyValid == 1 {
		// Generation owns the upstream rebuild flag. Changing only that flag
		// cannot invalidate the completed registration component identity.
		r.Rebuild = ""
		return hash(r), nil
	}
	return "", errVICIdial
}

func vicidialSQL(ctx context.Context, runner vicidialInputRunner, query string) (string, error) {
	if ctx == nil || runner == nil || query == "" {
		return "", errVICIdial
	}
	r, e := runner.RunInput(ctx, dbClient(), []byte(query))
	if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return "", errVICIdial
	}
	return strings.TrimSpace(r.Stdout), nil
}

func inspectVICIdialRegistration(ctx context.Context, runner vicidialInputRunner, ami string) (vicidialRegistration, string, error) {
	var row vicidialRegistration
	output, e := vicidialSQL(ctx, runner, vicidialRegistrationQuery)
	if e != nil || decode(output, &row) != nil {
		return row, "", errVICIdial
	}
	state, e := row.state(ami)
	return row, state, e
}

// This batch is private to the S05 registration action. One MariaDB connection
// holds the table locks across predicate checks and both updates. MyISAM has no
// rollback: a failure between updates is observed as partial, never COMPLETE.
// All unrelated columns and application tables remain untouched.
func guardedVICIdialRegistrationSQL(before vicidialRegistration, ami string) (string, error) {
	state, e := before.state(ami)
	if e != nil || (state != "absent" && state != "partial") {
		return "", errVICIdial
	}
	q, e := vicidialRegistrationSQL(ami)
	if e != nil {
		return "", e
	}
	updates := strings.Split(strings.TrimSpace(q), "\n")
	if len(updates) != 2 {
		return "", errVICIdial
	}
	const seedGuard = " AND active='Y' AND active_asterisk_server='Y' AND generate_vicidial_conf='Y' AND rebuild_conf_files='Y' AND asterisk_version='1.4.21.2' AND conf_engine='MEETME' AND ASTmgrUSERNAME='cron' AND ASTmgrUSERNAMEupdate='updatecron' AND ASTmgrUSERNAMElisten='listencron' AND ASTmgrUSERNAMEsend='sendcron' AND SHA2(ASTmgrSECRET,256)='"
	first := strings.TrimSuffix(updates[0], ";") + seedGuard + rawSHA([]byte("1234")) + "' AND @almadial_servers=1 AND @almadial_settings=1;\n"
	if state == "partial" {
		first = ""
	}
	post := "server_id='TESTast' AND server_ip='127.0.0.1' AND active='Y' AND active_asterisk_server='Y' AND generate_vicidial_conf='Y' AND asterisk_version='22.12.0' AND conf_engine='CONFBRIDGE' AND ASTmgrUSERNAME='almadial-s05' AND ASTmgrUSERNAMEupdate='almadial-s05' AND ASTmgrUSERNAMElisten='almadial-s05' AND ASTmgrUSERNAMEsend='almadial-s05' AND SHA2(ASTmgrSECRET,256)='" + rawSHA([]byte(ami)) + "'"
	// Initialize only an empty upstream password-hashing key before the first
	// HTTP request. Existing valid keys are preserved; application HTTP therefore
	// cannot hide a database mutation inside a nominal verification GET.
	second := "UPDATE asterisk.system_settings SET webroot_writable='0',pass_key=IF(pass_key IS NULL OR pass_key='',LOWER(HEX(RANDOM_BYTES(16))),pass_key) WHERE SHA2(COALESCE(pass_key,''),256)='" + before.PassKey + "' AND @almadial_servers=1 AND @almadial_settings=1 AND (SELECT COUNT(*) FROM asterisk.servers WHERE " + post + ")=1;\n"
	return "LOCK TABLES asterisk.servers WRITE, asterisk.system_settings WRITE;\nSET @almadial_servers=(SELECT COUNT(*) FROM asterisk.servers), @almadial_settings=(SELECT COUNT(*) FROM asterisk.system_settings);\n" + first + second + "UNLOCK TABLES;\n", nil
}

// The journaled caller must durably retain the exact protected preimage before
// invoking this helper. It is not exposed by the CLI or a general SQL endpoint.
func registerVICIdial(ctx context.Context, runner vicidialInputRunner, ami string, preimageSHA string) (string, error) {
	before, state, e := inspectVICIdialRegistration(ctx, runner, ami)
	if e != nil || hash(before) != preimageSHA {
		return "", errVICIdial
	}
	if shaPattern.MatchString(state) {
		return state, nil
	}
	query, e := guardedVICIdialRegistrationSQL(before, ami)
	if e != nil {
		return "", e
	}
	if _, e = vicidialSQL(ctx, runner, query); e != nil {
		return "", e
	}
	_, state, e = inspectVICIdialRegistration(ctx, runner, ami)
	if e != nil || !shaPattern.MatchString(state) {
		return "", errVICIdial
	}
	return state, nil
}

// Preserve typed preimages without reflecting credentials into errors.
func (r vicidialRegistration) protectedPreimage() []byte { b, _ := json.Marshal(r); return b }
