package provision

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
)

var errDatabase = errors.New("S03 database state requires review; no destructive reconciliation or schema replay is permitted")

const databaseArchiveSHA = "ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058"
const databaseCreateSHA = "8a210454a6f42fd3ef0054033a5c4396a0cd3615442ebd53491dfd2efa38de40"
const databaseSeedSHA = "c215f549c3a6dc39e20224e666efd6d99aa85b93910fe8dbaf7daa62a962203e"

// DatabaseSource retains verified bytes, rather than trusting a path after
// verification. Its fields cannot be populated by JSON or caller declarations.
type DatabaseSource struct{ create, seed string }

func rawSHA(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func (s DatabaseSource) Digest() string {
	if s.create == "" || s.seed == "" {
		return ""
	}
	return hash(struct{ Archive, Create, Seed, Writer string }{databaseArchiveSHA, rawSHA([]byte(s.create)), rawSHA([]byte(s.seed)), "S03-only/r4037/1748"})
}
func LoadDatabaseSource(ctx context.Context, m artifacts.Manifest, path string) (DatabaseSource, error) {
	spec, e := m.Artifact("vicidial")
	if e != nil {
		return DatabaseSource{}, e
	}
	r, e := artifacts.VerifyLocal(ctx, spec, path)
	if e != nil || r.Identity != artifacts.Verified {
		return DatabaseSource{}, errDatabase
	}
	f, e := os.Open(path)
	if e != nil {
		return DatabaseSource{}, errDatabase
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 512<<20+1))
	if e != nil || len(b) > 512<<20 || rawSHA(b) != databaseArchiveSHA {
		return DatabaseSource{}, errDatabase
	}
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return DatabaseSource{}, errDatabase
	}
	var s DatabaseSource
	for _, f := range z.File {
		if f.Name != "2026-11-05/extras/MySQL_AST_CREATE_tables.sql" && f.Name != "2026-11-05/extras/first_server_install.sql" {
			continue
		}
		r, e := f.Open()
		if e != nil {
			return DatabaseSource{}, errDatabase
		}
		v, e := io.ReadAll(io.LimitReader(r, 1<<20))
		r.Close()
		if e != nil {
			return DatabaseSource{}, errDatabase
		}
		if strings.HasSuffix(f.Name, "MySQL_AST_CREATE_tables.sql") {
			if rawSHA(v) != databaseCreateSHA {
				return DatabaseSource{}, errDatabase
			}
			s.create = string(v)
			for _, old := range []string{"GRANT RELOAD ON *.* TO cron@'%';", "GRANT RELOAD ON *.* TO cron@localhost;", "flush privileges;"} {
				if strings.Count(s.create, old) != 1 {
					return DatabaseSource{}, errDatabase
				}
				s.create = strings.Replace(s.create, old, "-- AlmaDial M0-04: privilege provisioning owned by lab bootstrap.", 1)
			}
		} else {
			if rawSHA(v) != databaseSeedSHA {
				return DatabaseSource{}, errDatabase
			}
			s.seed = string(v)
		}
	}
	if s.Digest() == "" {
		return DatabaseSource{}, errDatabase
	}
	return s, nil
}
