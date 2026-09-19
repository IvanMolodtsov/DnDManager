package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if IsRemoteSQLiteDSN("file:data/dnd.db") {
		t.Fatal("file dsn is local")
	}
}

func TestOpenFromEnvRemotePaused(t *testing.T) {
	t.Setenv("DATABASE_URL", "libsql://example.turso.io")
	_, err := OpenFromEnv()
	if err == nil || !strings.Contains(err.Error(), "not wired yet") {
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
