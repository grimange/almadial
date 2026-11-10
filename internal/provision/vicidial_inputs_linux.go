//go:build linux

package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// This closed list comes from the pinned generator block plus system_settings.
// Raw rows stay transient or in protected preimages; only their digest enters
// the plan. No SQL DDL, source bootstrap or application write is performed here.
var vicidialGeneratorTables = []string{"conferences", "phones", "servers", "system_settings", "vicidial_call_menu", "vicidial_call_menu_options", "vicidial_conf_templates", "vicidial_confbridges", "vicidial_conferences", "vicidial_music_on_hold", "vicidial_server_carriers", "vicidial_voicemail"}
var vicidialRegistrationColumns = map[string]bool{"server_ip": true, "active": true, "active_asterisk_server": true, "generate_vicidial_conf": true, "rebuild_conf_files": true, "asterisk_version": true, "conf_engine": true, "ASTmgrUSERNAME": true, "ASTmgrUSERNAMEupdate": true, "ASTmgrUSERNAMElisten": true, "ASTmgrUSERNAMEsend": true, "ASTmgrSECRET": true}

func vicidialSQLIdentifier(s string) bool {
	return s != "" && len(s) <= 64 && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_") == ""
}

type vicidialInputTable struct {
	Name    string
	Columns []string
	Rows    []string
}

func inspectVICIdialGeneratorInputs(ctx context.Context, runner vicidialInputRunner) (string, []byte, error) {
	return inspectVICIdialGeneratorInputsWeb(ctx, runner, false)
}
func inspectVICIdialGeneratorInputsWeb(ctx context.Context, runner vicidialInputRunner, handoff bool) (string, []byte, error) {
	tables := []vicidialInputTable{}
	for _, name := range vicidialGeneratorTables {
		columnsText, e := vicidialSQL(ctx, runner, "SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='asterisk' AND TABLE_NAME='"+name+"' ORDER BY ORDINAL_POSITION;")
		if e != nil {
			return "", nil, errVICIdial
		}
		columns := strings.Split(columnsText, "\n")
		if len(columns) > 1000 {
			return "", nil, errVICIdial
		}
		expressions := []string{}
		seen := map[string]bool{}
		for _, column := range columns {
			if !vicidialSQLIdentifier(column) || seen[column] {
				return "", nil, errVICIdial
			}
			seen[column] = true
			// HEX(number) encodes its numeric value, not its decimal bytes, and
			// can produce odd-length strings. Cast every column to bytes first.
			expressions = append(expressions, "HEX(CAST(`"+column+"` AS BINARY))")
		}
		query := "SELECT JSON_ARRAY(" + strings.Join(expressions, ",") + ") FROM asterisk.`" + name + "`"
		// The upstream generator alone allocates this server's 300 bridge slots.
		// That narrowly owned set is validated separately before and after execution.
		if name == "vicidial_confbridges" {
			query += " WHERE server_ip IS NULL OR server_ip<>'127.0.0.1'"
		}
		text, e := vicidialSQL(ctx, runner, query+" LIMIT 10001;")
		if e != nil {
			return "", nil, errVICIdial
		}
		// S07 completes the already-secured bootstrap without invoking the
		// upstream blanket phone rewrite. Normalize only this nongenerator flag
		// to its accepted preimage for the immutable S05 receipt.
		if handoff && name == "system_settings" {
			var err error
			text, err = webNormalizeSetup(columns, text)
			if err != nil {
				return "", nil, err
			}
		}
		rows, e := normalizeVICIdialGeneratorRows(name, columns, text)
		if e != nil {
			return "", nil, e
		}
		tables = append(tables, vicidialInputTable{name, columns, rows})
	}
	preimage, _ := json.Marshal(tables)
	return rawSHA(preimage), preimage, nil
}
func normalizeVICIdialGeneratorRows(name string, columns []string, text string) ([]string, error) {
	rows := []string{}
	if text == "" {
		return rows, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 10000 {
		return nil, errVICIdial
	}
	for _, line := range lines {
		var cells []*string
		if json.Unmarshal([]byte(line), &cells) != nil || len(cells) != len(columns) {
			return nil, errVICIdial
		}
		for i, cell := range cells {
			if cell != nil {
				if len(*cell)%2 != 0 {
					return nil, errVICIdial
				}
				if _, e := hex.DecodeString(*cell); e != nil {
					return nil, errVICIdial
				}
			}
			// Registration's exact pre/post values are checked by its own inspector.
			// Excluding only those declared columns makes the immutable generator input
			// anchor stable across registration and the upstream rebuild-flag update.
			if name == "servers" && vicidialRegistrationColumns[columns[i]] || name == "system_settings" && (columns[i] == "webroot_writable" || columns[i] == "pass_key") {
				v := "S05-owned"
				cells[i] = &v
			}
		}
		b, _ := json.Marshal(cells)
		rows = append(rows, string(b))
	}
	sort.Strings(rows)
	return rows, nil
}
