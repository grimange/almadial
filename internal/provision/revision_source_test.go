package provision

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
)

// The real canonical export must retain SQL pins and the full native source
// inventory used by accepted plans; this never executes upstream source.
func TestAcceptedRevisionSourceLoaders(t *testing.T) {
	path := os.Getenv("ALMADIAL_TEST_CANONICAL_INPUT")
	if path == "" {
		t.Skip("accepted export evidence not requested")
	}
	b, err := os.ReadFile("../../manifests/stack.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := artifacts.Load(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	db, err := LoadDatabaseSource(context.Background(), m, path)
	if err != nil || db.Digest() == "" {
		t.Fatal(db.Digest(), err)
	}
	vd, err := LoadVICIdialSource(context.Background(), m, path)
	if err != nil || vd.Digest() == "" {
		t.Fatal(vd.Digest(), err)
	}
	t.Logf("database source identity %s; VICIdial source identity %s", db.Digest(), vd.Digest())
}
