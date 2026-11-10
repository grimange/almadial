package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"syscall"

	"github.com/grimange/almadial/internal/artifacts"
)

const AsteriskVersion = "22.12.0"
const asteriskArchiveSHA = "f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2"
const pjArchiveSHA = "04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7"

var errAsterisk = errors.New("S04 Asterisk precondition or runtime contract failed; preserve evidence and explicitly reconcile uncertain installation state")

// AsteriskSource retains authenticated bytes; serialized declarations cannot
// manufacture source verification or change the selected bundled PJPROJECT.
type AsteriskSource struct{ archive, pj []byte }

func (s AsteriskSource) Digest() string {
	if len(s.archive) == 0 || len(s.pj) == 0 {
		return ""
	}
	return hash(struct{ Archive, PJ, Signer string }{rawSHA(s.archive), rawSHA(s.pj), "F2FC93DB7587BD1FB49E045A5D984BE337191CE7"})
}
func sourceBytes(path, sha string) ([]byte, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, errAsterisk
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() > 512<<20 {
		return nil, errAsterisk
	}
	b, e := io.ReadAll(io.LimitReader(f, 512<<20+1))
	if e != nil || rawSHA(b) != sha {
		return nil, errAsterisk
	}
	return b, nil
}
func LoadAsteriskSource(ctx context.Context, m artifacts.Manifest, archive, signature, keyring, scratch, pj string) (AsteriskSource, error) {
	spec, e := m.Artifact("asterisk")
	if e != nil {
		return AsteriskSource{}, e
	}
	r, e := artifacts.VerifyLocal(ctx, spec, archive)
	if e != nil || r.Identity != artifacts.Verified {
		return AsteriskSource{}, errAsterisk
	}
	r, e = artifacts.VerifyAsteriskSignature(ctx, spec, archive, signature, keyring, scratch)
	if e != nil || r.Publisher != artifacts.Verified {
		return AsteriskSource{}, errAsterisk
	}
	b, e := sourceBytes(archive, asteriskArchiveSHA)
	if e != nil {
		return AsteriskSource{}, e
	}
	p, e := sourceBytes(pj, pjArchiveSHA)
	if e != nil {
		return AsteriskSource{}, e
	}
	if e = verifyAsteriskLayout(b); e != nil {
		return AsteriskSource{}, e
	}
	return AsteriskSource{b, p}, nil
}
func verifyAsteriskLayout(b []byte) error {
	z, e := gzip.NewReader(bytes.NewReader(b))
	if e != nil {
		return errAsterisk
	}
	defer z.Close()
	t := tar.NewReader(z)
	want := map[string]bool{"configure": false, "Makefile": false, ".version": false, "third-party/pjproject/Makefile": false, "third-party/pjproject/pjproject-2.17.tar.bz2.md5": false, "res/res_agi.c": false, "apps/app_confbridge.c": false, "channels/chan_pjsip.c": false}
	for {
		h, e := t.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return errAsterisk
		}
		const prefix = "asterisk-22.12.0/"
		if len(h.Name) < len(prefix) || h.Name[:len(prefix)] != prefix {
			return errAsterisk
		}
		name := h.Name[len(prefix):]
		if _, ok := want[name]; ok {
			if h.Typeflag != tar.TypeReg {
				return errAsterisk
			}
			want[name] = true
			if name == ".version" {
				v, e := io.ReadAll(t)
				if e != nil || string(bytes.TrimSpace(v)) != AsteriskVersion {
					return errAsterisk
				}
			}
		}
	}
	for _, v := range want {
		if !v {
			return errAsterisk
		}
	}
	return nil
}
