package platform

import (
	"net/http"
	"time"
)

const LangCookie = "dnd_lang"

func WriteLangCookie(w http.ResponseWriter, lang string) {
	lang = NormalizeLang(lang)
	http.SetCookie(w, &http.Cookie{
		Name:     LangCookie,
		Value:    lang,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
}

func RequestLang(r *http.Request, user *AuthUser) string {
	if c, err := r.Cookie(LangCookie); err == nil && c.Value != "" {
		return NormalizeLang(c.Value)
	}
	if user != nil && user.Language != "" {
		return NormalizeLang(user.Language)
	}
	return AcceptLanguage(r.Header.Get("Accept-Language"))
}
