package provision

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
)

const VICIdialRevision = "4037"
const vicidialTreeSHA = "04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed"
const vicidialRoot = "2026-11-05/"
const vicidialHome = "/usr/share/astguiclient"
const vicidialWeb = "/var/www/html"

var errVICIdial = errors.New("S05 VICIdial source or installation contract failed; preserve existing files and reconcile conflicting state")

// VICIdialSource is constructible only from the accepted artifact verifier.
// Retained bytes close the verification/use race; neither JSON nor a caller's
// inventory declaration can authorize an alternate upstream tree.
type VICIdialSource struct {
	files      map[string][]byte
	archive    []byte
	archiveSHA string // verified representation; source identity remains the canonical tree
}

type VICIdialSourceFile struct {
	Path, SHA256 string
	Size         int
}

func (s VICIdialSource) Inventory() []VICIdialSourceFile {
	r := make([]VICIdialSourceFile, 0, len(s.files))
	for p, b := range s.files {
		r = append(r, VICIdialSourceFile{p, rawSHA(b), len(b)})
	}
	sort.Slice(r, func(i, j int) bool { return r[i].Path < r[j].Path })
	return r
}

func (s VICIdialSource) Digest() string {
	if len(s.files) == 0 {
		return ""
	}
	return hash(struct {
		Revision, Archive, Tree string
		Files                   []VICIdialSourceFile
	}{VICIdialRevision, databaseArchiveSHA, vicidialTreeSHA, s.Inventory()})
}

func LoadVICIdialSource(ctx context.Context, m artifacts.Manifest, archive string) (VICIdialSource, error) {
	spec, e := m.Artifact("vicidial")
	if e != nil {
		return VICIdialSource{}, e
	}
	r, e := artifacts.VerifyInstallInput(ctx, spec, archive)
	if e != nil || r.Identity != artifacts.Verified {
		return VICIdialSource{}, errVICIdial
	}
	// Retain exactly the representation just verified, closing the reopen race.
	// Digest() binds the accepted reference inventory, not repackaged ZIP bytes.
	b, e := sourceBytes(archive, r.ActualSHA256)
	if e != nil {
		return VICIdialSource{}, errVICIdial
	}
	f, e := readVICIdialZIP(ctx, b)
	if e != nil || len(f) != 2511 || string(f["version"]) != "2.14b0.5\n" {
		return VICIdialSource{}, errVICIdial
	}
	for _, p := range []string{"install.pl", "docs/LICENSE.txt", "www/vicidial/admin.php", "www/agc/vicidial.php", "www/agc/dbconnect_mysqli.php", "www/vicidial/dbconnect_mysqli.php", "bin/ADMIN_keepalive_ALL.pl", "bin/AST_update.pl", "agi/agi-VDAD_ALL_outbound.agi", "extras/Asterisk/AGI.pm", "docs/conf_examples/pjsip.conf.sample-16", "docs/conf_examples/confbridge.conf.sample-18"} {
		if len(f[p]) == 0 {
			return VICIdialSource{}, errVICIdial
		}
	}
	return VICIdialSource{files: f, archive: b, archiveSHA: r.ActualSHA256}, nil
}

// Defense in depth on the retained ZIP. This helper never extracts or executes
// source; VerifyLocal additionally checks ZIP bounds and canonical tree identity.
func readVICIdialZIP(ctx context.Context, b []byte) (map[string][]byte, error) {
	if ctx == nil {
		return nil, errVICIdial
	}
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil || len(z.File) > 30000 {
		return nil, errVICIdial
	}
	files, seen := map[string][]byte{}, map[string]bool{}
	total := uint64(0)
	for _, f := range z.File {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		name := strings.TrimSuffix(f.Name, "/")
		if name != strings.TrimSuffix(vicidialRoot, "/") && !strings.HasPrefix(name, vicidialRoot) || path.Clean(name) != name || strings.ContainsAny(name, "\\\x00\r\n:") || seen[name] {
			return nil, errVICIdial
		}
		seen[name] = true
		if f.Mode().IsDir() {
			if f.UncompressedSize64 != 0 {
				return nil, errVICIdial
			}
			continue
		}
		if !f.Mode().IsRegular() || !strings.HasPrefix(name, vicidialRoot) || f.UncompressedSize64 > 256<<20 || total+f.UncompressedSize64 > 2<<30 {
			return nil, errVICIdial
		}
		total += f.UncompressedSize64
		r, e := f.Open()
		if e != nil {
			return nil, errVICIdial
		}
		v, e := io.ReadAll(io.LimitReader(r, int64(f.UncompressedSize64)+1))
		r.Close()
		if e != nil || uint64(len(v)) != f.UncompressedSize64 {
			return nil, errVICIdial
		}
		files[strings.TrimPrefix(name, vicidialRoot)] = v
	}
	// An archive entry cannot also be the parent of another entry.
	for name := range files {
		for p := path.Dir(name); p != "."; p = path.Dir(p) {
			if _, ok := files[p]; ok {
				return nil, errVICIdial
			}
		}
	}
	return files, nil
}

// VICIdialFile describes a single approved destination, rather than a recursive
// copy or permission rule. Content and metadata are both installation conditions.
type VICIdialFile struct {
	Source, Target, SourceSHA256, SHA256 string
	Owner, Group, Label, Writer          string
	Mode                                 uint32
}

func vicidialProcps(name string, b []byte) ([]byte, error) {
	pins := map[string]string{
		"bin/ADMIN_keepalive_ALL.pl": "4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67",
		"bin/AST_manager_send.pl":    "83be4e5ca25591adc6991273b84114b53164ef49b7bf04bee9e9b7413b7cb384",
		"bin/AST_VDauto_dial.pl":     "cd2115f06d8c7730af73d67d468612c4ac127f815b31689a3112b374a14d9d87",
		"bin/AST_VDremote_agents.pl": "0187ca67f99c0b435280fe76a2cfd2c4816ad7f82411f2112bc988c37ee4a596",
	}
	if want, ok := pins[name]; ok {
		old := []byte(`/bin/ps -o "%p %a" --no-headers -A`)
		n := 1
		if name == "bin/ADMIN_keepalive_ALL.pl" {
			n = 3
		}
		if rawSHA(b) != want || bytes.Count(b, old) != n {
			return nil, errVICIdial
		}
		return bytes.ReplaceAll(b, old, []byte(`/bin/ps -o pid,args -ww --no-headers -A`)), nil
	}
	return b, nil
}

func (s VICIdialSource) deployment() ([]VICIdialFile, map[string][]byte, error) {
	if s.Digest() == "" {
		return nil, nil, errVICIdial
	}
	files, content := []VICIdialFile{}, map[string][]byte{}
	for _, src := range s.Inventory() {
		name := src.Path
		dst, group, label, mode := "", "root", "usr_t", uint32(0644)
		switch {
		case strings.HasPrefix(name, "www/"):
			dst, group, label, mode = vicidialWeb+"/"+strings.TrimPrefix(name, "www/"), "apache", "httpd_sys_content_t", 0640
		case strings.HasPrefix(name, "bin/"):
			dst, mode = vicidialHome+"/"+strings.TrimPrefix(name, "bin/"), 0755
		case strings.HasPrefix(name, "agi/"):
			dst, group, label, mode = "/var/lib/asterisk/agi-bin/"+strings.TrimPrefix(name, "agi/"), "asterisk", "bin_t", 0750
		case strings.HasPrefix(name, "sounds/"):
			dst, group, label, mode = "/var/lib/asterisk/sounds/"+strings.TrimPrefix(name, "sounds/"), "asterisk", "var_lib_t", 0640
		case strings.HasPrefix(name, "libs/"):
			dst = vicidialHome + "/" + name
		case name == "extras/Asterisk.pm":
			dst = "/usr/local/share/perl5/5.40/Asterisk.pm"
		case name == "extras/Asterisk/AGI.pm":
			dst = "/usr/local/share/perl5/5.40/Asterisk/AGI.pm"
		case strings.HasPrefix(name, "extras/Asterisk/"):
			dst = vicidialHome + "/libs/Asterisk/" + strings.TrimPrefix(name, "extras/Asterisk/")
		case name == "experimental/FastAGIServer/FastAGIServer.pl":
			dst, mode = vicidialHome+"/FastAGIServer/FastAGIServer.pl", 0755
		case strings.HasPrefix(name, "experimental/FastAGIServer/AGIHandlers/"):
			dst = vicidialHome + "/FastAGIServer/AGIHandlers/" + strings.TrimPrefix(name, "experimental/FastAGIServer/AGIHandlers/")
		case strings.HasPrefix(name, "docs/") || name == "INSTALL" || name == "UPGRADE" || name == "version":
			dst = "/usr/share/doc/almadial-vicidial/" + name
		default:
			continue
		}
		b, e := vicidialProcps(name, s.files[name])
		if e != nil || content[dst] != nil {
			return nil, nil, errVICIdial
		}
		writer := "upstream-r4037"
		if rawSHA(b) != src.SHA256 {
			writer = "almadial-m0-procps"
		}
		files = append(files, VICIdialFile{name, dst, src.SHA256, rawSHA(b), "root", group, label, writer, mode})
		content[dst] = bytes.Clone(b)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Target < files[j].Target })
	return files, content, nil
}

func (s VICIdialSource) Deployment() ([]VICIdialFile, error) {
	f, _, e := s.deployment()
	return f, e
}
