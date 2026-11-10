package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestReadOnlyCommandBoundary(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"help"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e != nil || !strings.Contains(out.String(), "never install") {
			t.Fatalf("help: %v %q", e, out.String())
		}
	}
	for _, args := range [][]string{{"install"}, {"lab-run"}, {"source"}, {"files"}, {"source", "--manifest", "missing"}, {"source", "--manifest", "missing", "--source", "missing", "trailing"}, {"files", "--unknown"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e == nil || out.Len() != 0 {
			t.Fatalf("unapproved request accepted: %q %v", args, e)
		}
	}
}
