//go:build linux

package provision

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"
)

// Only the journaled configuration action creates this distinct AMI secret.
// S03 database credentials are consumed verbatim and never written here.
func vicidialAMISecret() (string, error) {
	parent, e := safeDirectory("/etc/almadial/secrets")
	if e != nil {
		return "", errVICIdial
	}
	defer parent.Close()
	var st syscall.Stat_t
	if syscall.Fstat(int(parent.Fd()), &st) != nil || st.Uid != 0 || st.Gid != 0 || st.Mode&07777 != 0700 {
		return "", errVICIdial
	}
	fd, e := syscall.Openat(int(parent.Fd()), "vicidial-ami", syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if errors.Is(e, syscall.ENOENT) {
		return "", nil
	}
	if e != nil {
		return "", errVICIdial
	}
	f := os.NewFile(uintptr(fd), "protected S05 AMI credential")
	defer f.Close()
	if syscall.Fstat(fd, &st) != nil || st.Mode&syscall.S_IFMT != syscall.S_IFREG || st.Uid != 0 || st.Gid != 0 || st.Nlink != 1 || st.Mode&07777 != 0600 || st.Size != 21 {
		return "", errVICIdial
	}
	label, e := labelFD(f)
	if e != nil || label != "system_u:object_r:etc_t:s0" {
		return "", errVICIdial
	}
	b, e := io.ReadAll(io.LimitReader(f, 22))
	if e != nil || len(b) != 21 || b[20] != '\n' || !vicidialHex(string(b[:20]), 20) {
		return "", errVICIdial
	}
	return string(b[:20]), nil
}

func provisionVICIdialConfiguration(ctx context.Context, runner vicidialInputRunner) (string, error) {
	if ctx == nil || runner == nil {
		return "", errVICIdial
	}
	secrets, e := databaseSecrets()
	if e != nil || len(secrets) != 2 {
		return "", errVICIdial
	}
	ami, e := vicidialAMISecret()
	if e != nil {
		return "", e
	}
	if ami == "" {
		b := make([]byte, 10)
		if _, e = rand.Read(b); e != nil {
			return "", e
		}
		candidate := hex.EncodeToString(b)
		input, _ := json.Marshal(struct{ AMI string }{candidate})
		r, e := runner.RunInput(ctx, vicidialFilesCommand("ami-create"), input)
		if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
			return "", errVICIdial
		}
		ami, e = vicidialAMISecret()
		if e != nil || ami != candidate {
			return "", errVICIdial
		}
	}
	files, e := vicidialConfiguration(secrets[0], ami)
	if e != nil {
		return "", e
	}
	return vicidialOwned(ctx, runner, "configuration", files, true)
}
