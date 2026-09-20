package platform

import (
	"io/fs"
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

// UnpackAssets copies fsys (typically the module-root embed of web/locales/migrations)
// into a temp directory so AssetDir/Migrate/FileServer can keep using paths.
func UnpackAssets(fsys fs.FS) (string, error) {
	root, err := os.MkdirTemp("", "dnd-assets-")
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		dest := filepath.Join(root, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0o644)
	})
	if err != nil {
		os.RemoveAll(root)
		return "", err
	}
	return root, nil
}
