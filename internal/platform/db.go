// Package platform is shared infrastructure: SQLite, migrations, sessions, CSRF, i18n, and HTML rendering.
package platform

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// Remote HTTP pool size for Turso/libSQL (serverless; a handful of table players).
const remoteMaxOpenConns = 4

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
// Hosted: set DATABASE_URL or TURSO_DATABASE_URL to a libsql:// (or https://) URL and
// TURSO_AUTH_TOKEN. Opened with the HTTP libSQL driver (no CGO). The Vercel function
// filesystem is ephemeral — do not rely on DATA_DIR there.
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
		return openLibSQL(dsn, authToken)
	}
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

func openLibSQL(dsn, authToken string) (*sql.DB, error) {
	full, err := libsqlDSN(dsn, authToken)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("libsql", full)
	if err != nil {
		return nil, fmt.Errorf("open libsql %s: %w", redactDSN(dsn), err)
	}
	db.SetMaxOpenConns(remoteMaxOpenConns)
	return db, nil
}

// libsqlDSN returns a driver DSN: libsql://host?authToken=… (Turso HTTP form).
// TURSO_AUTH_TOKEN is appended when the URL has no authToken/auth_token/jwt yet.
func libsqlDSN(dsn, authToken string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid remote sqlite dsn %s", redactDSN(dsn))
	}
	if strings.EqualFold(u.Scheme, "turso") {
		u.Scheme = "libsql"
	}
	q := u.Query()
	hasToken := q.Get("authToken") != "" || q.Get("auth_token") != "" || q.Get("jwt") != ""
	token := strings.TrimSpace(authToken)
	if !hasToken {
		if token == "" {
			return "", fmt.Errorf("remote sqlite %s: TURSO_AUTH_TOKEN is required", redactDSN(dsn))
		}
		q.Set("authToken", token)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
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
