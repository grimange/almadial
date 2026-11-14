package releaseassets

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestPinnedInputs(t *testing.T) {
	b, err := os.ReadFile("../../manifests/stack.lock.yaml")
	if err != nil || !bytes.Equal(b, Manifest) {
		t.Fatal("embedded lock drift", err)
	}
	b, err = os.ReadFile("../../manifests/acquisition-origins.json")
	if err != nil || !bytes.Equal(b, Origins) {
		t.Fatal("embedded origin policy drift", err)
	}
	for _, tc := range []struct {
		b   []byte
		sha string
	}{{Keyring, "645c8fb2b67133a59d9f7d1113e953c5dc79950fd0b28a6529a0efef0146727c"}, {Signature, "17763b5903a3962b0278ffd8d581e913eedd6c0fd21149f4855db67eb40d890d"}} {
		if fmt.Sprintf("%x", sha256.Sum256(tc.b)) != tc.sha {
			t.Fatal("embedded public trust drift")
		}
	}
}
