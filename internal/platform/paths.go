package platform

import (
	"os"
	"path/filepath"
)

// AssetDir resolves a repo-relative directory (web/templates, locales, …).
// Order: ASSET_ROOT, cwd and parents, then the executable directory.
// At each root it checks rel first (local `go run ./cmd/web`), then api/rel
// (Vercel: @vercel/go includeFiles globs from api/, after install copies).
func AssetDir(rel string) string {
	if root := os.Getenv("ASSET_ROOT"); root != "" {
		if p := firstDir(root, rel); p != "" {
			return p
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if p := walkFor(wd, rel); p != "" {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		if p := walkFor(filepath.Dir(exe), rel); p != "" {
			return p
		}
	}
	return rel
}

func walkFor(start, rel string) string {
	dir := start
	for i := 0; i < 8; i++ {
		if p := firstDir(dir, rel); p != "" {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func firstDir(root, rel string) string {
	for _, p := range []string{
		filepath.Join(root, rel),
		filepath.Join(root, "api", rel),
	} {
		if isDir(p) {
			return p
		}
	}
	return ""
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
