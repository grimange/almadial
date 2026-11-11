package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTLSCLIBoundary(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e != nil || !strings.Contains(out.String(), "zero host mutations") {
			t.Fatal(e, out.String())
		}
	}
	for _, args := range [][]string{{"install"}, {"inspect"}, {"plan"}, {"lab-run"}, {"lab-resume"}, {"plan", "--input", "missing", "--unknown"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out); e == nil || out.Len() != 0 {
			t.Fatal("unapproved command accepted", args, e)
		}
	}
}
