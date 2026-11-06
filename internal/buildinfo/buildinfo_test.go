package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestStableMetadata(t *testing.T) {
	a, b := String(), String()
	if a != b {
		t.Fatal("metadata changed between reads")
	}
	for _, s := range []string{"almadial dev", "commit: unknown", "target: " + runtime.GOOS + "/" + runtime.GOARCH, "go: " + runtime.Version()} {
		if !strings.Contains(a, s) {
			t.Fatalf("missing %q: %q", s, a)
		}
	}
}
