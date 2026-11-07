package artifacts

import (
	"context"
	"errors"
	"io"
	"os"
	"sort"
)

// Reviewed derivation programs from accepted M0/ALM11 HEAD. These are identity
// checks only: do not run Python, patch upstream sources, or permit arbitrary patches.
var derivationPins = map[string]string{
	"scripts/prepare_vicidial_r4037.py": "4a77bda0c1e0584585b249e0b8b57362497c97cd632f603abb98e4eb345031ba",
	"scripts/lab/m0_05_generator.py":    "d5bf82964816b3429b22577fd7b574c4ef9d3cadf1d02b0aee00440cd77f5d40",
	"scripts/lab/m0_06_fastagi.py":      "a52c5492a08e8d18e837023ae8056991401f967efe27203bcfc1006180001059",
	"scripts/lab/m0_06_runtime.py":      "67613d816c44f53d8b5cd8e618673e79579de344f91d9f00f43b44ded7be0dd8",
	"scripts/lab/m0_07_assets.py":       "7dce7f689031d068ee0de99d55dbb691890d99c6ce3110e3c9cc8256eb69c3da",
	"scripts/lab/m0_08_replay.py":       "4aebca442f3562394bc007b344d82b8565fb787527fdf7c10ab02e23bc76705d",
}

type Derivation struct {
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ActualSHA256   string `json:"actual_sha256"`
}

// VerifyDerivations checks all six reviewed helper identities inside the given
// repository root. Existing Python regression tests separately verify derivation
// semantics and source/output pins. No helper is executed by this function.
func VerifyDerivations(ctx context.Context, repository string) ([]Derivation, error) {
	root, err := os.OpenRoot(repository)
	if err != nil {
		return nil, errors.New("derivation: cannot open repository")
	}
	defer root.Close()
	paths := make([]string, 0, len(derivationPins))
	for p := range derivationPins {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	results := make([]Derivation, 0, len(paths))
	for _, p := range paths {
		st, err := root.Lstat(p)
		if err != nil || !st.Mode().IsRegular() || st.Size() > 1<<20 {
			return results, errors.New("derivation: missing or invalid reviewed file")
		}
		f, err := root.Open(p)
		if err != nil {
			return results, errors.New("derivation: cannot open reviewed file")
		}
		b, e := io.ReadAll(io.LimitReader(contextReader{ctx, f}, (1<<20)+1))
		f.Close()
		if e != nil || len(b) > 1<<20 {
			return results, errors.New("derivation: unreadable or excessive input")
		}
		d := Derivation{p, derivationPins[p], digestBytes(b)}
		results = append(results, d)
		if d.ActualSHA256 != d.ExpectedSHA256 {
			return results, errors.New("derivation: reviewed patch identity mismatch")
		}
	}
	return results, nil
}
