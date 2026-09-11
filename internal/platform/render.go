package platform

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Renderer executes HTML templates with the i18n t() helper bound to lang.
type Renderer struct {
	base   *template.Template
	bundle *Bundle
}

// BaseView is the layout data: user, language, CSRF, and optional Error key.
type BaseView struct {
	Title string
	User  *AuthUser
	Lang  string
	CSRF  string
	Error string
	Path  string
}

func NewBase(r *http.Request, title string) BaseView {
	return BaseView{
		Title: title,
		User:  UserFrom(r.Context()),
		Lang:  LangFrom(r.Context()),
		CSRF:  CSRFFrom(r),
		Path:  r.URL.RequestURI(),
	}
}

func (v BaseView) LoggedIn() bool { return v.User != nil }
func (v BaseView) IsAdmin() bool  { return v.User != nil && v.User.IsAdmin() }

// NewRenderer parses all *.html files under templatesDir.
func NewRenderer(templatesDir string, bundle *Bundle) (*Renderer, error) {
	base := template.New("").Funcs(template.FuncMap{
		"t":      func(string) string { return "" },
		"mod":    AbilityMod,
		"modFmt": FormatMod,
		"add":    func(a, b int) int { return a + b },
	})
	err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		rel, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = base.New(filepath.ToSlash(rel)).Parse(string(b))
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Renderer{base: base, bundle: bundle}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name, lang string, status int, data any) {
	lang = NormalizeLang(lang)
	clone, err := r.base.Clone()
	if err != nil {
		http.Error(w, "template clone", http.StatusInternalServerError)
		return
	}
	clone.Funcs(template.FuncMap{
		"t": func(key string) string { return r.bundle.T(lang, key) },
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := clone.ExecuteTemplate(w, name, data); err != nil {
		_, _ = w.Write([]byte("<!-- template error -->"))
	}
}

func AbilityMod(score int) int {
	m := score - 10
	if m >= 0 {
		return m / 2
	}
	return (m - 1) / 2
}

func FormatMod(score int) string {
	m := AbilityMod(score)
	if m >= 0 {
		return "+" + strconv.Itoa(m)
	}
	return strconv.Itoa(m)
}
