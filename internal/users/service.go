package users

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"dndmanager/internal/platform"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken      = errors.New("username taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameRequired   = errors.New("username required")
	ErrUsernameLength     = errors.New("username length")
	ErrUsernameChars      = errors.New("username chars")
	ErrPasswordRequired   = errors.New("password required")
	ErrPasswordLength     = errors.New("password length")
	ErrNotFound           = errors.New("user not found")
)

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// Service validates credentials and persists users.
type Service struct {
	Repo *Repository
}

// Register creates an account. The first user in the database becomes Admin.
func (s *Service) Register(username, password, lang string) (*User, error) {
	username, err := validateUsername(username)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.Repo.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return nil, err
	}
	role := RolePlayer
	if n == 0 {
		role = RoleAdmin
	}

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists); err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, ErrUsernameTaken
	}

	lang = platform.NormalizeLang(lang)
	res, err := tx.Exec(
		`INSERT INTO users (username, password_hash, role, language) VALUES (?, ?, ?, ?)`,
		username, string(hash), role, lang,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Repo.FindByID(id)
}

func (s *Service) Authenticate(username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	u, err := s.Repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *Service) FindByID(id int64) (*User, error) {
	u, err := s.Repo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (s *Service) SetLanguage(id int64, lang string) error {
	return s.Repo.UpdateLanguage(id, platform.NormalizeLang(lang))
}

func (s *Service) AuthUser(id int64) (*platform.AuthUser, error) {
	u, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	return ToAuth(u), nil
}

// ToAuth copies public fields onto the request-scoped identity used by middleware.
func ToAuth(u *User) *platform.AuthUser {
	if u == nil {
		return nil
	}
	return &platform.AuthUser{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Language: u.Language,
	}
}

func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", ErrUsernameRequired
	}
	n := utf8.RuneCountInString(username)
	if n < 3 || n > 32 {
		return "", ErrUsernameLength
	}
	if !usernameRE.MatchString(username) {
		return "", ErrUsernameChars
	}
	return username, nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if utf8.RuneCountInString(password) < 8 {
		return ErrPasswordLength
	}
	return nil
}

// ErrorKey maps a users error to a locales JSON key.
func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrUsernameRequired):
		return "error.username.required"
	case errors.Is(err, ErrUsernameLength):
		return "error.username.length"
	case errors.Is(err, ErrUsernameChars):
		return "error.username.chars"
	case errors.Is(err, ErrUsernameTaken):
		return "error.username.taken"
	case errors.Is(err, ErrPasswordRequired):
		return "error.password.required"
	case errors.Is(err, ErrPasswordLength):
		return "error.password.length"
	case errors.Is(err, ErrInvalidCredentials):
		return "error.credentials"
	default:
		return "error.generic"
	}
}
