package artifacts

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Trust anchors are reviewed M0 policy, never taken from a supplied manifest.
const asteriskFingerprint = "F2FC93DB7587BD1FB49E045A5D984BE337191CE7"

// Binary public key obtained by dearmoring the independently recorded M0 key
// (armored SHA256 2eb75c272ebd1f96742ef7b0b53ebe943c840c55addc22db2122a494df578145).
const asteriskKeyringSHA256 = "645c8fb2b67133a59d9f7d1113e953c5dc79950fd0b28a6529a0efef0146727c"

// VerifyAsteriskSignature uses the host's trusted /usr/bin/gpgv, no shell or key
// server. Linux descriptor paths bind hashing and signature checks to the same
// open artifact/key/signature. scratch must be an existing private directory.
func VerifyAsteriskSignature(ctx context.Context, s Spec, artifact, signature, keyring, scratch string) (Result, error) {
	r := initial(s)
	if !s.valid() || s.component != "asterisk" {
		return r, errors.New("signature: supported Asterisk specification required")
	}
	root, err := privateRoot(scratch)
	if err != nil {
		return r, err
	}
	defer root.Close()
	a, err := openRegular(artifact, maxArchiveBytes)
	if err != nil {
		return r, err
	}
	defer a.Close()
	sig, err := openRegular(signature, 1<<20)
	if err != nil {
		return r, err
	}
	defer sig.Close()
	key, err := openRegular(keyring, 1<<20)
	if err != nil {
		return r, err
	}
	defer key.Close()
	kb, err := io.ReadAll(io.LimitReader(key, (1<<20)+1))
	if err != nil || len(kb) > 1<<20 || digestBytes(kb) != asteriskKeyringSHA256 {
		r.Publisher = Rejected
		r.Disposition = Rejected
		return r, errors.New("signature: untrusted public keyring")
	}
	if _, err = key.Seek(0, io.SeekStart); err != nil {
		return r, errors.New("signature: cannot read keyring")
	}
	r, err = verifyFile(ctx, s, a)
	if err != nil {
		return r, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/gpgv", "--homedir", scratch, "--keyring", "/proc/self/fd/5", "--status-fd", "1", "/proc/self/fd/4", "/proc/self/fd/3")
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{a, sig, key}
	var out limitedOutput
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err = cmd.Run(); err != nil || !validSignatureStatus(out.text, asteriskFingerprint) {
		r.Publisher = Rejected
		r.Disposition = Rejected
		return r, errors.New("signature: trusted detached signature verification failed")
	}
	// Recheck bytes after gpgv; this API does not confer lasting trust on a path.
	if _, err = a.Seek(0, io.SeekStart); err != nil {
		return r, errors.New("signature: cannot revalidate artifact")
	}
	r, err = verifyFile(ctx, s, a)
	if err != nil {
		return r, err
	}
	r.Publisher = Verified
	r.Disposition = Verified
	r.Provenance = "gpgv detached signature; independently pinned M0 public key and fingerprint"
	r.Limitations = []string{"M0 maintainer-linked signing key association; OpenPGP Web-of-Trust not asserted", "candidate policy is not installation authorization"}
	return r, nil
}
func openRegular(path string, max int64) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("artifact: cannot open regular input")
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > max {
		f.Close()
		return nil, errors.New("artifact: invalid input file")
	}
	return f, nil
}

type limitedOutput struct{ text string }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(b.text)+len(p) > 64<<10 {
		return 0, errors.New("signature: excessive status output")
	}
	b.text += string(p)
	return len(p), nil
}
func validSignatureStatus(output, fingerprint string) bool {
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) == 12 && f[0] == "[GNUPG:]" && f[1] == "VALIDSIG" && f[2] == fingerprint && f[11] == fingerprint && (f[9] == "8" || f[9] == "9" || f[9] == "10" || f[9] == "11") {
			return true
		}
	}
	return false
}
