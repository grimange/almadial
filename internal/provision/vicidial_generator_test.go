package provision

import "testing"

func TestVICIdialGeneratorRequiresPinnedSource(t *testing.T) {
	for _, data := range [][]byte{nil, {}, []byte("unreviewed generation block"), []byte(vicidialGeneratorSourceSHA)} {
		if result, d, e := deriveVICIdialGenerator(data); e == nil || result != nil || d.Derived != "" {
			t.Fatal("unverified source produced executable generator", d, e)
		}
	}
	if _, _, e := vicidialFixture().Generator(); e == nil {
		t.Fatal("synthetic source authorized native generation")
	}
}

func TestVICIdialGeneratorSelection(t *testing.T) {
	for _, tc := range []struct {
		text, first, last string
		ok                bool
	}{
		{"start middle end", "start", "end", true},
		{"start start end", "start", "end", false},
		{"start end end", "start", "end", false},
		{"end start", "start", "end", false},
		{"start only", "start", "end", false},
		{"end only", "start", "end", false},
	} {
		result, e := generatorPart(tc.text, tc.first, tc.last)
		if (e == nil) != tc.ok || (tc.ok && result != "start middle ") {
			t.Fatal(tc, result, e)
		}
	}
}
