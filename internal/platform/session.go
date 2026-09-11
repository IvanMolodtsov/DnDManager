package platform

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	SessionCookie = "dnd_session"
	sessionTTL    = 30 * 24 * time.Hour
)

// Session is a cookie-backed row: optional user, CSRF token, expiry.
type Session struct {
	ID        string
	UserID    sql.NullInt64
	CSRFToken string
	ExpiresAt time.Time
}

// SessionStore persists sessions in SQLite.
type SessionStore struct {
	DB *sql.DB
}

func (st *SessionStore) Get(id string) (*Session, error) {
	s := &Session{}
	var expires string
	err := st.DB.QueryRow(
		`SELECT id, user_id, csrf_token, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.CSRFToken, &expires)
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, err = parseSQLiteTime(expires)
	if err != nil {
		return nil, err
	}
	if time.Now().After(s.ExpiresAt) {
		_ = st.Destroy(id)
		return nil, sql.ErrNoRows
	}
	return s, nil
}

func (st *SessionStore) Create(userID sql.NullInt64) (*Session, error) {
	id, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	csrf, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	exp := time.Now().Add(sessionTTL)
	_, err = st.DB.Exec(
		`INSERT INTO sessions (id, user_id, csrf_token, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, csrf, exp.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, err
	}
	return &Session{ID: id, UserID: userID, CSRFToken: csrf, ExpiresAt: exp}, nil
}

func (st *SessionStore) BindUser(sessionID string, userID int64) error {
	_, err := st.DB.Exec(`UPDATE sessions SET user_id = ? WHERE id = ?`, userID, sessionID)
	return err
}

// Rotate destroys oldID (if any) and issues a new session bound to userID.
func (st *SessionStore) Rotate(oldID string, userID int64) (*Session, error) {
	_ = st.Destroy(oldID)
	return st.Create(sql.NullInt64{Int64: userID, Valid: true})
}

func (st *SessionStore) Destroy(id string) error {
	_, err := st.DB.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func WriteSessionCookie(w http.ResponseWriter, s *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    s.ID,
		Path:     "/",
		Expires:  s.ExpiresAt,
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func parseSQLiteTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Parse(time.RFC3339, s)
}
