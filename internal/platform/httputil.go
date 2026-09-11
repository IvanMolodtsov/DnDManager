package platform

import (
	"net/http"
	"strconv"
	"strings"
)

func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func FormInt64(r *http.Request, name string) int64 {
	n, _ := strconv.ParseInt(FormTrim(r, name), 10, 64)
	return n
}

func FormInt(r *http.Request, name string) int {
	n, _ := strconv.Atoi(FormTrim(r, name))
	return n
}

func WantsPartial(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func PathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func FormTrim(r *http.Request, name string) string {
	if err := r.ParseForm(); err != nil {
		return ""
	}
	return strings.TrimSpace(r.FormValue(name))
}

func SafeReturn(r *http.Request, fallback string) string {
	ret := r.FormValue("return")
	if ret == "" {
		ret = r.URL.Query().Get("return")
	}
	if ret == "" || !strings.HasPrefix(ret, "/") || strings.HasPrefix(ret, "//") {
		return fallback
	}
	return ret
}
