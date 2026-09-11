package platform

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
)

type UserLookup func(id int64) (*AuthUser, error)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec, "path", r.URL.Path)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Sessions(store *SessionStore, lookup UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}
			var sess *Session
			if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
				s, err := store.Get(c.Value)
				if err == nil {
					sess = s
				}
			}
			if sess == nil {
				s, err := store.Create(sql.NullInt64{})
				if err != nil {
					slog.Error("create session", "err", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				sess = s
				WriteSessionCookie(w, sess)
			}

			ctx := WithSession(r.Context(), sess)
			var user *AuthUser
			if sess.UserID.Valid && lookup != nil {
				u, err := lookup(sess.UserID.Int64)
				if err == nil {
					user = u
					ctx = WithUser(ctx, user)
				}
			}
			lang := RequestLang(r, user)
			ctx = WithLang(ctx, lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFrom(r.Context()) == nil {
			Redirect(w, r, "/login")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r.Context())
		if u == nil {
			Redirect(w, r, "/login")
			return
		}
		if !u.IsAdmin() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
