package platform

import (
	"os"
	"strings"
)

// Getenv returns the environment value or def when unset/empty.
func Getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// ListenAddr is ADDR, else :$PORT (Vercel Go preset), else :8080.
func ListenAddr() string {
	if v := os.Getenv("ADDR"); v != "" {
		return v
	}
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return ":8080"
}

// CookieSecure is true when COOKIE_SECURE is 1/true/yes, or when running on Vercel.
func CookieSecure() bool {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))); v != "" {
		return v == "1" || v == "true" || v == "yes"
	}
	return os.Getenv("VERCEL") == "1"
}
