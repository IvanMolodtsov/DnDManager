package platform

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// CSRF rejects unsafe methods unless the form or X-CSRF-Token header matches the session.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		s := SessionFrom(r.Context())
		if s == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		token := r.FormValue("csrf_token")
		if token == "" {
			token = r.Header.Get("X-CSRF-Token")
		}
		if subtle.ConstantTimeCompare([]byte(s.CSRFToken), []byte(token)) != 1 {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func WantsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || accept == "" || strings.Contains(accept, "*/*")
}
