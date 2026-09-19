package app

import (
	"net/http"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/characters"
	"dndmanager/internal/platform"
	"dndmanager/internal/users"
)

type homeController struct {
	Users      *users.Service
	Campaigns  *campaigns.Service
	Characters *characters.Service
	Render     *platform.Renderer
}

type landingView struct {
	platform.BaseView
}

type dashView struct {
	platform.BaseView
	Campaigns  []campaigns.CampaignListItem
	Characters []characters.Character
}

func (h *homeController) index(w http.ResponseWriter, r *http.Request) {
	base := platform.NewBase(r, "app.name")
	if base.User == nil {
		h.Render.Render(w, "landing.html", base.Lang, http.StatusOK, landingView{BaseView: base})
		return
	}
	camps, err := h.Campaigns.ListForUser(base.User.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	chars, err := h.Characters.ListByOwner(base.User.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	v := dashView{BaseView: base, Campaigns: camps, Characters: chars}
	v.Title = "dash.welcome"
	h.Render.Render(w, "dashboard.html", v.Lang, http.StatusOK, v)
}

func (h *homeController) admin(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	if u == nil || !u.IsAdmin() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v := platform.NewBase(r, "admin.title")
	h.Render.Render(w, "admin.html", v.Lang, http.StatusOK, v)
}
