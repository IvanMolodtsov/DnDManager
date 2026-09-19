// Package platform is shared infrastructure: SQLite, migrations, sessions, CSRF, i18n, and HTML rendering.
package platform

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// OpenDB creates dataDir if needed and opens dataDir/dnd.db with foreign keys on.
// Tests and local Air use this (ephemeral temp dirs or DATA_DIR/dnd.db).
func OpenDB(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(filepath.Join(dataDir, "dnd.db")))
	return openSQLiteFile(dsn)
}

// OpenFromEnv opens the campaign database.
//
// Local (Air / go run): DATA_DIR/dnd.db via modernc.org/sqlite (default DATA_DIR=data).
// Hosted: set DATABASE_URL or TURSO_DATABASE_URL to a persistent libSQL/Turso URL.
// The Vercel function filesystem is ephemeral — do not rely on DATA_DIR there.
// Remote driver wiring is paused until Ivan provisions Marketplace Turso (see AGENTS.md).
func OpenFromEnv() (*sql.DB, error) {
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		return openConfiguredDSN(dsn, os.Getenv("TURSO_AUTH_TOKEN"))
	}
	if dsn := strings.TrimSpace(os.Getenv("TURSO_DATABASE_URL")); dsn != "" {
		return openConfiguredDSN(dsn, os.Getenv("TURSO_AUTH_TOKEN"))
	}
	return OpenDB(Getenv("DATA_DIR", "data"))
}

func openConfiguredDSN(dsn, authToken string) (*sql.DB, error) {
	if IsRemoteSQLiteDSN(dsn) {
		return nil, fmt.Errorf("remote sqlite %s: Turso/libSQL driver is not wired yet (paused before Marketplace provision). Local Air still uses DATA_DIR/dnd.db. When ready: vercel integration add turso, set TURSO_DATABASE_URL + TURSO_AUTH_TOKEN, then go get turso.tech/database/tursogo-serverless", redactDSN(dsn))
	}
	_ = authToken
	if strings.HasPrefix(dsn, "file:") {
		return openSQLiteFile(ensureSQLitePragmas(dsn))
	}
	// Bare path → local file (same as DATA_DIR/dnd.db).
	dir := filepath.Dir(dsn)
	base := filepath.Base(dsn)
	if base == "dnd.db" || strings.HasSuffix(strings.ToLower(base), ".db") {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
		fileDSN := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(dsn))
		return openSQLiteFile(fileDSN)
	}
	return OpenDB(Getenv("DATA_DIR", "data"))
}

// IsRemoteSQLiteDSN is true for Turso / libSQL HTTP URLs (not local file: DSNs).
func IsRemoteSQLiteDSN(dsn string) bool {
	s := strings.ToLower(strings.TrimSpace(dsn))
	return strings.HasPrefix(s, "libsql://") ||
		strings.HasPrefix(s, "turso://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://")
}

func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.Host == "" {
		return "(remote)"
	}
	if u.Scheme == "" {
		return u.Host
	}
	return u.Scheme + "://" + u.Host
}

func ensureSQLitePragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	return dsn + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
}

func openSQLiteFile(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	return db, nil
}
