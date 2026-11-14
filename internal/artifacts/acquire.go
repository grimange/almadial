package artifacts

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var installComponents = []string{"vicidial", "asterisk", "pjproject", "net-telnet", "viciphone"}

// AcquireInstall explicitly fetches the five pinned archives from their reviewed
// HTTPS origins or VICIdial's tree-verified authoritative revision. Fetch
// revalidates cache hits and rejects redirects and corruption.
// Planning never calls this function. Completed downloads survive a later failure.
func AcquireInstall(ctx context.Context, m Manifest, directory string, out io.Writer) (map[string]string, error) {
	paths := make(map[string]string)
	for _, component := range installComponents {
		spec, err := m.Artifact(component)
		if err != nil {
			return nil, err
		}
		if _, err = fmt.Fprintf(out, "Verifying/acquiring pinned %s from %s\n", component, spec.Source()); err != nil {
			return nil, err
		}
		root, err := privateRoot(directory)
		if err != nil {
			return nil, err
		}
		err = root.Mkdir(spec.sha256, 0700)
		root.Close()
		if err != nil && !os.IsExist(err) {
			return nil, err
		}
		contentDir := filepath.Join(directory, spec.sha256)
		result, err := Fetch(ctx, spec, FetchOptions{Directory: contentDir})
		if err != nil {
			return nil, fmt.Errorf("%s acquisition unavailable or invalid: %w; preserve cache, restore the exact pinned origin or obtain an approved mirror; rerun almadial install", component, err)
		}
		fmt.Fprintf(out, "Verified %s representation SHA-256 %s; policy %s; tree %s (%s)\n", component, result.ActualSHA256, result.ContentPolicy, result.TreeSHA256, result.Provenance)
		paths[component] = filepath.Join(contentDir, component+".artifact")
	}
	return paths, nil
}
