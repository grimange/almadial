//go:build linux

package provision

import (
	"context"
	"encoding/json"
)

// Owned file bytes, including credentials, are passed through bounded stdin.
// The result and stage snapshots contain identities and missing paths only.
func vicidialOwned(ctx context.Context, runner vicidialInputRunner, component string, files []vicidialOwnedFile, mutate bool) (string, error) {
	if ctx == nil || runner == nil || len(files) == 0 {
		return "", errVICIdial
	}
	type file struct {
		Target, Owner, Group, Label, SHA256 string
		Mode                                uint32
		Content                             []byte
	}
	payload := struct{ Files []file }{Files: []file{}}
	for _, f := range files {
		payload.Files = append(payload.Files, file{f.path, f.owner, f.group, f.label, rawSHA(f.content), f.mode, f.content})
	}
	input, e := json.Marshal(payload)
	if e != nil {
		return "", errVICIdial
	}
	command := vicidialFilesCommand("owned-inspect")
	if mutate {
		command = vicidialFilesCommand("owned-install")
	}
	command.Args = append(command.Args, component)
	r, e := runner.RunInput(ctx, command, input)
	if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return "", errVICIdial
	}
	var result struct {
		Missing  []string
		Verified int
	}
	if decode(r.Stdout, &result) != nil || result.Verified+len(result.Missing) != len(files) {
		return "", errVICIdial
	}
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.path] = true
	}
	for _, p := range result.Missing {
		if !seen[p] {
			return "", errVICIdial
		}
		delete(seen, p)
	}
	if mutate {
		// The installation count is not a postcondition. Independently reopen
		// every destination and verify bytes, ownership, mode and label.
		identity, err := vicidialOwned(ctx, runner, component, files, false)
		if err != nil || !shaPattern.MatchString(identity) {
			return "", errVICIdial
		}
		return identity, nil
	}
	if len(result.Missing) == len(files) {
		return "absent", nil
	}
	if len(result.Missing) > 0 {
		return "partial", nil
	}
	// Never include secret bytes in even the private journal serialization.
	identities := []struct {
		Path, SHA256, Owner, Group, Label string
		Mode                              uint32
	}{}
	for _, f := range files {
		identities = append(identities, struct {
			Path, SHA256, Owner, Group, Label string
			Mode                              uint32
		}{f.path, rawSHA(f.content), f.owner, f.group, f.label, f.mode})
	}
	return hash(identities), nil
}

func (s VICIdialPerlSource) deployment() ([]vicidialOwnedFile, error) {
	if s.Digest() == "" {
		return nil, errVICIdial
	}
	return []vicidialOwnedFile{
		{"/usr/local/share/perl5/5.40/Net/Telnet.pm", "root", "root", "usr_t", 0644, s.module},
		{"/usr/share/doc/almadial-net-telnet/README", "root", "root", "usr_t", 0644, s.notice},
	}, nil
}
