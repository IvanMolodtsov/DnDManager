package platform

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestListenAddr(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "")
	if got := ListenAddr(); got != ":8080" {
		t.Fatalf("default ADDR: got %q", got)
	}
	t.Setenv("ADDR", ":9090")
	if got := ListenAddr(); got != ":9090" {
		t.Fatalf("ADDR: got %q", got)
	}
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "3000")
	if got := ListenAddr(); got != ":3000" {
		t.Fatalf("PORT: got %q", got)
	}
}

func TestCookieSecure(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("VERCEL", "")
	if CookieSecure() {
		t.Fatal("expected insecure cookies locally")
	}
	t.Setenv("VERCEL", "1")
	if !CookieSecure() {
		t.Fatal("expected Secure on Vercel")
	}
	t.Setenv("COOKIE_SECURE", "0")
	if CookieSecure() {
		t.Fatal("COOKIE_SECURE=0 should win")
	}
	t.Setenv("COOKIE_SECURE", "true")
	if !CookieSecure() {
		t.Fatal("COOKIE_SECURE=true")
	}
}

func TestIsRemoteSQLiteDSN(t *testing.T) {
	if !IsRemoteSQLiteDSN("libsql://example.turso.io") {
		t.Fatal("libsql")
	}
	if !IsRemoteSQLiteDSN("https://example.turso.io") {
		t.Fatal("https")
	}
	if IsRemoteSQLiteDSN("file:data/dnd.db") {
		t.Fatal("file dsn is local")
	}
}

func TestLibsqlDSNAuthToken(t *testing.T) {
	got, err := libsqlDSN("libsql://example.turso.io", "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "libsql" || u.Host != "example.turso.io" {
		t.Fatalf("shape: %q", got)
	}
	if u.Query().Get("authToken") != "secret-token" {
		t.Fatalf("authToken: %q", got)
	}

	already := "libsql://example.turso.io?authToken=embedded"
	got, err = libsqlDSN(already, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("authToken") != "embedded" {
		t.Fatalf("kept embedded token, got %q", got)
	}

	_, err = libsqlDSN("libsql://example.turso.io", "")
	if err == nil || !strings.Contains(err.Error(), "TURSO_AUTH_TOKEN") {
		t.Fatalf("expected token required, got %v", err)
	}
}

func TestOpenFromEnvRemoteRequiresToken(t *testing.T) {
	t.Setenv("DATABASE_URL", "libsql://example.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "")
	_, err := OpenFromEnv()
	if err == nil || !strings.Contains(err.Error(), "TURSO_AUTH_TOKEN") {
		t.Fatalf("got %v", err)
	}
}

func TestOpenFromEnvLocal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("TURSO_DATABASE_URL", "")
	t.Setenv("DATA_DIR", dir)
	db, err := OpenFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := os.Stat(filepath.Join(dir, "dnd.db")); err != nil {
		t.Fatal(err)
	}
}

func TestAssetDir(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "locales")
	if err := os.Mkdir(want, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSET_ROOT", root)
	got := AssetDir("locales")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssetDirAPINested(t *testing.T) {
	t.Setenv("ASSET_ROOT", "")
	root := t.TempDir()
	want := filepath.Join(root, "api", "locales")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "tmp")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	got := AssetDir("locales")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestUnpackAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"locales/en.json":  {Data: []byte(`{"ok":"1"}`)},
		"web/static/x.css": {Data: []byte("body{}")},
	}
	dir, err := UnpackAssets(fsys)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	raw, err := os.ReadFile(filepath.Join(dir, "locales", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":"1"}` {
		t.Fatalf("got %q", raw)
	}
	if _, err := fs.Stat(os.DirFS(dir), "web/static/x.css"); err != nil {
		t.Fatal(err)
	}
}
