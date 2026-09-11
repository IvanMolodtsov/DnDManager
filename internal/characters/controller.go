package characters

import (
	"net/http"
	"strings"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type Controller struct {
	Svc       *Service
	Campaigns *campaigns.Service
	Catalog   *catalog.Service
	Render    *platform.Renderer
}

func (c *Controller) Mount(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /characters", auth(http.HandlerFunc(c.list)))
	mux.Handle("GET /campaigns/{id}/characters/new", auth(http.HandlerFunc(c.showWizard)))
	mux.Handle("POST /campaigns/{id}/characters/wizard", auth(http.HandlerFunc(c.postWizard)))
	mux.Handle("GET /campaigns/{id}/characters/wizard/subclass", auth(http.HandlerFunc(c.subclassPartial)))
	mux.Handle("GET /campaigns/{id}/characters/wizard/scores", auth(http.HandlerFunc(c.scoresPartial)))
	mux.Handle("GET /characters/{id}", auth(http.HandlerFunc(c.show)))
	mux.Handle("GET /characters/{id}/level-up", auth(http.HandlerFunc(c.showLevelUp)))
	mux.Handle("POST /characters/{id}/level-up/preview", auth(http.HandlerFunc(c.previewLevelUp)))
	mux.Handle("POST /characters/{id}/level-up", auth(http.HandlerFunc(c.postLevelUp)))
}

func (c *Controller) base(r *http.Request, title string) platform.BaseView {
	return platform.NewBase(r, title)
}

type listView struct {
	platform.BaseView
	Characters []Character
}

func (c *Controller) list(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	chars, err := c.Svc.ListByOwner(u.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	lang := platform.LangFrom(r.Context())
	for i := range chars {
		c.Svc.Localize(&chars[i], lang)
	}
	v := listView{BaseView: c.base(r, "nav.characters"), Characters: chars}
	c.Render.Render(w, "characters/list.html", v.Lang, http.StatusOK, v)
}

type showView struct {
	platform.BaseView
	Character *Character
	ReadOnly  bool
	IsOwner   bool
	IsDM      bool
	ClassLine string
}

func (c *Controller) show(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ch, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, err := c.Svc.CanView(ch, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	v := showView{
		BaseView:  c.base(r, ch.Name),
		Character: ch,
		ReadOnly:  readonly,
		IsOwner:   ch.OwnerID == u.ID,
		IsDM:      readonly,
		ClassLine: ch.ClassLine(),
	}
	c.Render.Render(w, "characters/show.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) memberCampaign(w http.ResponseWriter, r *http.Request) (*campaigns.Campaign, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	camp, err := c.Campaigns.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	if _, err := c.Campaigns.RequireMember(id, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return camp, true
}

func (c *Controller) loadLive(w http.ResponseWriter, r *http.Request) (*Character, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	ch, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.CanView(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	return ch, true
}

func parseASI(r *http.Request) rules.AbilityScores {
	var a rules.AbilityScores
	for _, field := range []string{"asi_a", "asi_b"} {
		key := strings.ToLower(platform.FormTrim(r, field))
		if rules.ValidAbilityKey(key) {
			a.AddKey(key, 1)
		}
	}
	return a
}
