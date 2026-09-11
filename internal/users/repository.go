package users

import (
	"database/sql"
	"time"
)

// Repository is SQLite access for users.
type Repository struct {
	DB *sql.DB
}

func (r *Repository) Count() (int, error) {
	var n int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *Repository) Create(u *User) (int64, error) {
	res, err := r.DB.Exec(
		`INSERT INTO users (username, password_hash, role, language) VALUES (?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Role, u.Language,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) FindByID(id int64) (*User, error) {
	return scanUser(r.DB.QueryRow(
		`SELECT id, username, password_hash, role, language, created_at FROM users WHERE id = ?`, id,
	))
}

func (r *Repository) FindByUsername(username string) (*User, error) {
	return scanUser(r.DB.QueryRow(
		`SELECT id, username, password_hash, role, language, created_at FROM users WHERE username = ?`, username,
	))
}

func (r *Repository) UpdateLanguage(id int64, lang string) error {
	_, err := r.DB.Exec(`UPDATE users SET language = ? WHERE id = ?`, lang, id)
	return err
}

func scanUser(row *sql.Row) (*User, error) {
	u := &User{}
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Language, &created); err != nil {
		return nil, err
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", created, time.UTC); err == nil {
		u.CreatedAt = t
	}
	return u, nil
}
