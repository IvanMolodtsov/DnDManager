package users

import (
	"net/http"

	"dndmanager/internal/platform"
)

// Controller serves register, login, logout, and language switch.
type Controller struct {
	Svc      *Service
	Sessions *platform.SessionStore
	Render   *platform.Renderer
}

// Mount registers unauthenticated auth routes on mux.
func (c *Controller) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /register", c.showRegister)
	mux.HandleFunc("POST /register", c.register)
	mux.HandleFunc("GET /login", c.showLogin)
	mux.HandleFunc("POST /login", c.login)
	mux.HandleFunc("POST /logout", c.logout)
	mux.HandleFunc("POST /lang", c.setLang)
}

func (c *Controller) base(r *http.Request, title string) platform.BaseView {
	return platform.NewBase(r, title)
}

type formView struct {
	platform.BaseView
	Username string
}

func (c *Controller) showRegister(w http.ResponseWriter, r *http.Request) {
	if platform.UserFrom(r.Context()) != nil {
		platform.Redirect(w, r, "/")
		return
	}
	v := formView{BaseView: c.base(r, "auth.register.title")}
	c.Render.Render(w, "users/register.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) register(w http.ResponseWriter, r *http.Request) {
	if platform.UserFrom(r.Context()) != nil {
		platform.Redirect(w, r, "/")
		return
	}
	username := platform.FormTrim(r, "username")
	password := r.FormValue("password")
	lang := platform.LangFrom(r.Context())
	u, err := c.Svc.Register(username, password, lang)
	if err != nil {
		v := formView{BaseView: c.base(r, "auth.register.title"), Username: username}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "users/register.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	c.establishSession(w, r, u)
	platform.WriteLangCookie(w, u.Language)
	platform.Redirect(w, r, "/")
}

func (c *Controller) showLogin(w http.ResponseWriter, r *http.Request) {
	if platform.UserFrom(r.Context()) != nil {
		platform.Redirect(w, r, "/")
		return
	}
	v := formView{BaseView: c.base(r, "auth.login.title")}
	c.Render.Render(w, "users/login.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) login(w http.ResponseWriter, r *http.Request) {
	if platform.UserFrom(r.Context()) != nil {
		platform.Redirect(w, r, "/")
		return
	}
	username := platform.FormTrim(r, "username")
	password := r.FormValue("password")
	u, err := c.Svc.Authenticate(username, password)
	if err != nil {
		v := formView{BaseView: c.base(r, "auth.login.title"), Username: username}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "users/login.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	c.establishSession(w, r, u)
	platform.WriteLangCookie(w, u.Language)
	platform.Redirect(w, r, "/")
}

func (c *Controller) logout(w http.ResponseWriter, r *http.Request) {
	if s := platform.SessionFrom(r.Context()); s != nil {
		_ = c.Sessions.Destroy(s.ID)
	}
	platform.ClearSessionCookie(w)
	platform.Redirect(w, r, "/")
}

func (c *Controller) setLang(w http.ResponseWriter, r *http.Request) {
	lang := platform.NormalizeLang(platform.FormTrim(r, "lang"))
	platform.WriteLangCookie(w, lang)
	if u := platform.UserFrom(r.Context()); u != nil {
		_ = c.Svc.SetLanguage(u.ID, lang)
	}
	platform.Redirect(w, r, platform.SafeReturn(r, "/"))
}

func (c *Controller) establishSession(w http.ResponseWriter, r *http.Request, u *User) {
	old := platform.SessionFrom(r.Context())
	oldID := ""
	if old != nil {
		oldID = old.ID
	}
	s, err := c.Sessions.Rotate(oldID, u.ID)
	if err != nil {
		http.Error(w, "session", http.StatusInternalServerError)
		return
	}
	platform.WriteSessionCookie(w, s)
}
