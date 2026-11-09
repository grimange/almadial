package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDatabaseCLI(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"help"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e != nil || !strings.Contains(out.String(), "CANDIDATE") {
			t.Fatal(e)
		}
	}
	for _, args := range [][]string{{"install"}, {"lab-run"}, {"lab-resume", "--state-dir", "/tmp/never-created"}, {"inspect"}, {"plan", "--input", "missing"}, {"plan", "--source", "missing", "--manifest", "missing"}, {"inspect", "--unknown"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e == nil {
			t.Fatal("invalid CLI accepted", args)
		}
	}
}
