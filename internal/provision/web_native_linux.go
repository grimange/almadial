//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed web_native.py
var webAdapterSource string

func webAdapter() string {
	return strings.TrimSuffix(vicidialFilesAdapter, "if __name__ == '__main__':\n    sys.exit(main())\n") + webAdapterSource
}

type WebNative struct{ Runtime RuntimeNative }

func webFiles() []vicidialOwnedFile {
	return []vicidialOwnedFile{
		{"/etc/httpd/almadial-s07.conf", "root", "root", "httpd_config_t", 0644, []byte(WebVhost())},
		{"/etc/httpd/conf.modules.d/99-almadial-s07.conf", "root", "root", "httpd_config_t", 0644, []byte(webLoader)},
	}
}
func (n WebNative) adapter(ctx context.Context, mode string) (WebSnapshot, error) {
	var out WebSnapshot
	type file struct {
		Target, Owner, Group, Label, SHA256 string
		Mode                                uint32
		Content                             []byte
	}
	payload := struct{ Files []file }{Files: []file{}}
	for _, f := range webFiles() {
		payload.Files = append(payload.Files, file{f.path, f.owner, f.group, f.label, rawSHA(f.content), f.mode, f.content})
	}
	input, e := json.Marshal(payload)
	if e != nil {
		return out, errWeb
	}
	r, e := n.Runtime.Application.input()
	if e != nil {
		return out, e
	}
	result, e := r.RunInput(ctx, dbCommand("/usr/bin/python3", "-c", webAdapter(), mode), input)
	if e != nil || !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		return out, errWeb
	}
	if mode == "web-inspect" && decode(result.Stdout, &out) != nil {
		return out, errWeb
	}
	return out, nil
}
func (n WebNative) InspectWeb(ctx context.Context) (WebSnapshot, error) {
	s, e := n.adapter(ctx, "web-inspect")
	if e != nil {
		return s, e
	}
	// Only this downstream inspector accepts the closed first-login transition.
	n.Runtime.Application.WebHandoff = true
	parent, e := n.Runtime.InspectRuntime(ctx)
	if e != nil || !runtimeSatisfied(parent) {
		return s, errWeb
	}
	s.Host, s.Foundation = parent.Host, parent.Digest()
	return s, nil
}

type webRunner struct {
	native WebNative
	stage  WebStage
}

func (n WebNative) Runner(s WebStage) execution.CommandRunner { return webRunner{n, s} }
func (r webRunner) Validate(c execution.Command) error {
	if r.stage.doc().Policy != WebPolicyDigest() || r.stage.doc().Source != r.native.Runtime.Application.Source.Digest() {
		return errWeb
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(c) == jsonString(a.Command) {
			return nil
		}
	}
	return errWeb
}
func (r webRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 {
		return execution.Result{}, errWeb
	}
	modes := map[string]string{"web.00.files": "web-files", "web.01.services": "web-services", "web.02.verify": "web-verify"}
	mode, ok := modes[c.Args[1]]
	if !ok {
		return execution.Result{}, errWeb
	}
	_, e := r.native.adapter(ctx, mode)
	result := execution.Result{Started: true}
	if e != nil {
		result.ExitCode = 1
	}
	return result, e
}

// Only N is mapped to the known Y preimage; no other material application
// setting can be hidden from the S05 receipt. The S07 inspector separately
// verifies the protected administrator, single settings row and live pages.
func webNormalizeSetup(columns []string, text string) (string, error) {
	lines := strings.Split(text, "\n")
	if len(lines) != 1 {
		return "", errWeb
	}
	var cells []*string
	if json.Unmarshal([]byte(text), &cells) != nil || len(cells) != len(columns) {
		return "", errWeb
	}
	found := false
	for i, c := range columns {
		if c == "first_login_trigger" {
			found = true
			if cells[i] == nil || (*cells[i] != "59" && *cells[i] != "4E") {
				return "", errWeb
			}
			v := "59"
			cells[i] = &v
		}
	}
	if !found {
		return "", errWeb
	}
	b, _ := json.Marshal(cells)
	return string(b), nil
}
