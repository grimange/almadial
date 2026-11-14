package artifacts

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

type Disposition string

const (
	Verified    Disposition = "VERIFIED"
	Rejected    Disposition = "REJECTED"
	NotVerified Disposition = "NOT_VERIFIED"
)
const maxArchiveBytes int64 = 512 << 20

// Spec can only be obtained from a validated Manifest. Its private fields prevent
// callers from substituting expected digests, approved origins or trust modes.
type Spec struct {
	component, version, source, sha256, format, root, tree, embeddedPath, embeddedSHA string
	origins                                                                           []string
	files                                                                             int
}

func (s Spec) Component() string { return s.component }
func (s Spec) Source() string    { return s.source }
func (m Manifest) Artifact(component string) (Spec, error) {
	s, err := m.artifact(component)
	if err == nil {
		s.origins = append([]string{}, m.origins[component]...)
	}
	return s, err
}

func (m Manifest) artifact(component string) (Spec, error) {
	if m.digest == "" {
		return Spec{}, errors.New("manifest: uninitialized")
	}
	d := m.doc
	switch component {
	case "asterisk":
		return Spec{component: component, version: d.Asterisk.Version, source: d.Asterisk.Source, sha256: d.Asterisk.SHA256, format: "tgz"}, nil
	case "vicidial":
		return Spec{component: component, version: "r4037/schema1748", source: d.VICIdial.Archive, sha256: d.VICIdial.SHA256, format: "zip", root: "2026-11-05", tree: d.VICIdial.TreeSHA256, files: 2511}, nil
	case "viciphone":
		return Spec{component: component, version: "3.0/" + d.WebRTC.Revision, source: "https://codeload.github.com/vicimikec/ViciPhone/tar.gz/" + d.WebRTC.Revision, sha256: d.WebRTC.SHA256, format: "tgz", root: "ViciPhone-" + d.WebRTC.Revision, embeddedPath: "src/js/sip.js", embeddedSHA: d.WebRTC.SIPJSSHA256, files: 48}, nil
	case "pjproject":
		return Spec{component: component, version: d.PJProject.Version, source: d.PJProject.Source, sha256: d.PJProject.SHA256, format: "tbz"}, nil
	case "net-telnet":
		return Spec{component: component, version: d.NetTelnet.Version, source: d.NetTelnet.Source, sha256: d.NetTelnet.SHA256, format: "tgz"}, nil
	default:
		return Spec{}, errors.New("artifact: unsupported component")
	}
}

// Result distinguishes content identity, transport, publisher evidence and policy.
// VERIFIED means the stated property passed, never permission to install.
type Result struct {
	Component              string      `json:"component"`
	Version                string      `json:"version"`
	Source                 string      `json:"source"`
	ExpectedSHA256         string      `json:"expected_sha256"`
	ContentPolicy          string      `json:"content_policy"`
	ReferenceArchiveSHA256 string      `json:"reference_archive_sha256,omitempty"`
	ExpectedTreeSHA256     string      `json:"expected_tree_sha256,omitempty"`
	ActualSHA256           string      `json:"actual_sha256"`
	Size                   int64       `json:"size"`
	TreeSHA256             string      `json:"tree_sha256,omitempty"`
	Files                  int         `json:"files,omitempty"`
	Identity               Disposition `json:"identity"`
	Transport              Disposition `json:"transport"`
	Publisher              Disposition `json:"publisher"`
	Disposition            Disposition `json:"disposition"`
	Provenance             string      `json:"provenance"`
	Limitations            []string    `json:"limitations"`
	InstallationAuthorized bool        `json:"installation_authorized"`
}

func initial(s Spec) Result {
	return Result{Component: s.component, Version: s.version, Source: s.source, ExpectedSHA256: s.sha256, ContentPolicy: "archive-sha256", Identity: NotVerified, Transport: NotVerified, Publisher: NotVerified, Disposition: NotVerified, Limitations: []string{"candidate policy is not installation authorization", "local checksum alone does not authenticate publisher"}}
}
func (s Spec) valid() bool {
	b, e := hex.DecodeString(s.sha256)
	return s.component != "" && e == nil && len(b) == 32 && hex.EncodeToString(b) == s.sha256 && (s.format == "tgz" || s.format == "tbz" || s.format == "zip")
}

// VerifyLocal always reopens and rehashes the supplied regular file. No cache hit
// skips verification. The caller must provide a stable artifact; a private copy is
// needed before a later executor uses it (this package exposes no executor).
func VerifyLocal(ctx context.Context, s Spec, path string) (Result, error) {
	return verifyLocal(ctx, s, path, false)
}

// VerifyInstallInput additionally accepts the explicit canonical SVN export
// representation. Ordinary archives still require their exact pinned SHA-256.
// Export verification uses the independently accepted manifest tree digest and
// full inventory; a revision label or ZIP comment alone never authorizes input.
func VerifyInstallInput(ctx context.Context, s Spec, path string) (Result, error) {
	return verifyLocal(ctx, s, path, true)
}

func verifyLocal(ctx context.Context, s Spec, path string, canonical bool) (Result, error) {
	r := initial(s)
	if !s.valid() {
		return r, errors.New("artifact: uninitialized specification")
	}
	f, err := openRegular(path, maxArchiveBytes)
	if err != nil {
		return r, errors.New("artifact: cannot open local file")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return r, errors.New("artifact: expected regular file")
	}
	if canonical {
		return verifyInstallFile(ctx, s, f)
	}
	return verifyFile(ctx, s, f)
}

func verifyInstallFile(ctx context.Context, s Spec, f *os.File) (Result, error) {
	r, err := verifyFile(ctx, s, f)
	if err == nil || ctx.Err() != nil || !revisionSpec(s) || r.Size > maxArchiveBytes {
		return r, err
	}
	// Only the tagged export representation opts into canonical verification.
	// A damaged original archive must not silently become a tree-verified input.
	if zipBounds(f, r.Size) != nil {
		return r, err
	}
	z, e := zip.NewReader(f, r.Size)
	if e != nil || z.Comment != revisionComment(s) {
		return r, err
	}
	r.ContentPolicy = "svn-revision-tree-sha256"
	r.ReferenceArchiveSHA256, r.ExpectedSHA256 = s.sha256, ""
	r.ExpectedTreeSHA256 = s.tree
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return r, e
	}
	r.Files, r.TreeSHA256, e = inspect(ctx, s, f, r.Size)
	if e != nil {
		return r, fmt.Errorf("canonical revision export: %w", e)
	}
	r.Identity, r.Disposition = Verified, NotVerified
	r.Source = vicidialSVN + "@4037"
	r.Provenance = "exact r4037 source-tree identity from the independently accepted manifest; reconstructed ZIP bytes differ from the reference archive"
	r.Limitations = []string{"SVN transport is unauthenticated; publisher identity is not independently authenticated", "all 2511 file contents and paths are bound by the accepted tree digest", "candidate policy is not installation authorization"}
	return r, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func verifyFile(ctx context.Context, s Spec, f *os.File) (Result, error) {
	r := initial(s)
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(contextReader{ctx, f}, maxArchiveBytes+1))
	r.Size = n
	r.ActualSHA256 = hex.EncodeToString(h.Sum(nil))
	if err != nil || n > maxArchiveBytes {
		return r, errors.New("artifact: unreadable, canceled or exceeds size limit")
	}
	if r.ActualSHA256 != s.sha256 {
		r.Identity = Rejected
		r.Disposition = Rejected
		return r, errors.New("artifact: SHA-256 mismatch")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return r, errors.New("artifact: cannot inspect archive")
	}
	r.Files, r.TreeSHA256, err = inspect(ctx, s, f, n)
	if err != nil {
		r.Identity = Rejected
		r.Disposition = Rejected
		return r, fmt.Errorf("artifact: %w", err)
	}
	r.Identity = Verified
	r.Provenance = "pinned local content; acquisition not independently observed by this run"
	if s.component == "vicidial" {
		r.Provenance = "M0 recorded official HTTPS snapshot; full exported tree equality; no signed SVN revision"
	}
	if s.component == "asterisk" {
		r.Limitations = append(r.Limitations, "detached signature required; call VerifyAsteriskSignature with reviewed keyring")
	}
	// Signature/transport proofs are intentionally distinct from local identity.
	return r, nil
}
