package characters

import (
	"net/http"

	"dndmanager/internal/platform"
)

func (c *Controller) renderCombat(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_vitals.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderMagicCast(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	v.OOBCombat = true
	c.Render.Render(w, "characters/sheet_after_cast.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) adjustHP(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	delta := platform.FormInt(r, "delta")
	if err := c.Svc.AdjustHP(ch, platform.UserFrom(r.Context()).ID, delta); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderCombat(w, r, ch, false)
}

func (c *Controller) adjustTempHP(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	delta := platform.FormInt(r, "delta")
	if err := c.Svc.AdjustTempHP(ch, platform.UserFrom(r.Context()).ID, delta); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderCombat(w, r, ch, false)
}

func (c *Controller) toggleDeath(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	fail := platform.FormTrim(r, "track") == "fail"
	pip := platform.FormInt(r, "pip")
	if err := c.Svc.ToggleDeath(ch, platform.UserFrom(r.Context()).ID, fail, pip); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderCombat(w, r, ch, false)
}

func (c *Controller) dismissEffect(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	id, err := platform.PathID(r, "effectID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.DismissEffect(ch, platform.UserFrom(r.Context()).ID, id); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderCombat(w, r, ch, false)
}
