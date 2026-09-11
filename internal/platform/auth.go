package platform

import (
	"context"
	"net/http"
)

type contextKey int

const (
	sessionKey contextKey = iota
	userKey
	langKey
)

// AuthUser is the request-scoped identity (no password hash).
type AuthUser struct {
	ID       int64
	Username string
	Role     string
	Language string
}

func (u *AuthUser) IsAdmin() bool {
	return u != nil && u.Role == "Admin"
}

func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

func SessionFrom(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionKey).(*Session)
	return s
}

func WithUser(ctx context.Context, u *AuthUser) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func UserFrom(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(userKey).(*AuthUser)
	return u
}

func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, langKey, lang)
}

func LangFrom(ctx context.Context) string {
	lang, _ := ctx.Value(langKey).(string)
	if lang == "" {
		return "en"
	}
	return lang
}

func CSRFFrom(r *http.Request) string {
	if s := SessionFrom(r.Context()); s != nil {
		return s.CSRFToken
	}
	return ""
}
