//go:build linux

package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
)

func TestVICIdialGeneratorInputNormalization(t *testing.T) {
	encode := func(v string) string { return hex.EncodeToString([]byte(v)) }
	row := func(values ...any) string { b, _ := json.Marshal(values); return string(b) }
	columns := []string{"server_ip", "rebuild_conf_files", "server_description"}
	before := row(encode("10.10.10.15"), encode("Y"), encode("operator description"))
	after := row(encode("127.0.0.1"), encode("N"), encode("operator description"))
	a, err := normalizeVICIdialGeneratorRows("servers", columns, before)
	if err != nil {
		t.Fatal(err)
	}
	b, err := normalizeVICIdialGeneratorRows("servers", columns, after)
	if err != nil || hash(a) != hash(b) {
		t.Fatal("declared registration invalidated input anchor", err)
	}
	changed, _ := normalizeVICIdialGeneratorRows("servers", columns, row(encode("127.0.0.1"), encode("N"), encode("foreign edit")))
	if hash(a) == hash(changed) {
		t.Fatal("operator column excluded")
	}
	// Phone columns retain even the legacy server reference and endpoint secret.
	for _, column := range []string{"server_ip", "pass", "protocol", "extension"} {
		a, _ := normalizeVICIdialGeneratorRows("phones", []string{column}, row(encode("seed")))
		b, _ := normalizeVICIdialGeneratorRows("phones", []string{column}, row(encode("changed")))
		if hash(a) == hash(b) {
			t.Fatal("phone drift ignored", column)
		}
	}
	a, _ = normalizeVICIdialGeneratorRows("phones", []string{"notes"}, row(nil)+"\n"+row(encode(""))+"\n"+row(encode("line\nnext")))
	b, _ = normalizeVICIdialGeneratorRows("phones", []string{"notes"}, row(encode("line\nnext"))+"\n"+row(nil)+"\n"+row(encode("")))
	if hash(a) != hash(b) || len(a) != 3 || a[0] == a[1] {
		t.Fatal("row ordering or NULL identity lost")
	}
	for _, text := range []string{`["0"]`, `["GG"]`, `[1]`, `{}`, `null`, `[]`, `["00","01"]`, `["00"] trailing`, strings.Repeat("[null]\n", 10000) + "[null]"} {
		if _, err := normalizeVICIdialGeneratorRows("phones", []string{"notes"}, text); err == nil {
			t.Fatal("malformed or oversized rows accepted")
		}
	}
}

type vicidialInputsRunner struct {
	t              *testing.T
	columns        string
	fail, truncate bool
	queries        []string
}

func (r *vicidialInputsRunner) RunInput(_ context.Context, c execution.Command, input []byte) (execution.Result, error) {
	r.t.Helper()
	q := string(input)
	r.queries = append(r.queries, q)
	if c.Executable != "/usr/bin/mariadb" || !strings.HasPrefix(q, "SELECT ") || strings.Contains(q, "; UPDATE") {
		r.t.Fatal("input inspection mutated database")
	}
	if strings.Contains(q, "JSON_ARRAY(") && !strings.Contains(q, "HEX(CAST(`id` AS BINARY))") {
		r.t.Fatal("numeric columns must be encoded as bytes before HEX")
	}
	out := `["73656564"]`
	if strings.Contains(q, "information_schema.COLUMNS") {
		out = r.columns
	}
	return execution.Result{Started: true, ExitCode: map[bool]int{false: 0, true: 1}[r.fail], Stdout: out, StdoutTruncated: r.truncate}, nil
}
func TestVICIdialGeneratorInputInspection(t *testing.T) {
	for _, tc := range []struct {
		name, columns         string
		fail, truncate, valid bool
	}{
		{"valid", "id", false, false, true}, {"missing metadata", "", false, false, false},
		{"unsafe identifier", "id`", false, false, false}, {"duplicate metadata", "id\nid", false, false, false},
		{"failed query", "id", true, false, false}, {"truncated", "id", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &vicidialInputsRunner{t: t, columns: tc.columns, fail: tc.fail, truncate: tc.truncate}
			digest, preimage, err := inspectVICIdialGeneratorInputs(context.Background(), r)
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
			if !tc.valid {
				return
			}
			if digest != rawSHA(preimage) || len(r.queries) != 2*len(vicidialGeneratorTables) {
				t.Fatal("incomplete inventory")
			}
			found := false
			for _, q := range r.queries {
				if strings.Contains(q, "FROM asterisk.`vicidial_confbridges`") {
					found = strings.Contains(q, "WHERE server_ip IS NULL OR server_ip<>'127.0.0.1'")
				}
			}
			if !found {
				t.Fatal("nonlocal or NULL bridge records excluded")
			}
			var tables []vicidialInputTable
			if json.Unmarshal(preimage, &tables) != nil || len(tables) != len(vicidialGeneratorTables) {
				t.Fatal("preimage not complete")
			}
		})
	}
}

func TestVICIdialPhoneHandoffPreservesOtherGeneratorInputs(t *testing.T) {
	r := &vicidialInputsRunner{t: t, columns: "id"}
	if _, _, err := inspectVICIdialGeneratorInputsPhone(context.Background(), r, false, true); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, q := range r.queries {
		if strings.Contains(q, "extension<>") {
			if !strings.Contains(q, "FROM asterisk.`phones`") || !strings.Contains(q, "extension<>'alm_s10_9061' OR extension IS NULL") {
				t.Fatal("handoff excluded foreign inputs")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("synthetic phone not handed off")
	}
}
