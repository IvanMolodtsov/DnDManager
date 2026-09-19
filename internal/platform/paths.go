package platform

import (
	"os"
	"path/filepath"
)

// AssetDir resolves a repo-relative directory (web/templates, locales, …).
// Order: ASSET_ROOT, cwd and parents, then the executable directory.
func AssetDir(rel string) string {
	if root := os.Getenv("ASSET_ROOT"); root != "" {
		p := filepath.Join(root, rel)
		if isDir(p) {
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
		p := filepath.Join(dir, rel)
		if isDir(p) {
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

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
