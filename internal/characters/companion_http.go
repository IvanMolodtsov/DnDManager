package characters

import (
	"errors"
	"net/http"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

func (c *Controller) companionsIsland(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	c.renderCompanions(w, r, ch, readonly, http.StatusOK)
}

func (c *Controller) searchCompanions(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	if readonly {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v := c.sheetView(r, ch, readonly)
	v.CompanionKind = r.URL.Query().Get("kind")
	if v.CompanionKind == "" {
		v.CompanionKind = platform.FormTrim(r, "kind")
	}
	v.CompanionQuery = r.URL.Query().Get("q")
	v.CompanionCR = r.URL.Query().Get("cr")
	v.CompanionType = r.URL.Query().Get("type")
	hits, err := c.Svc.SearchCompanionMonsters(ch, v.CompanionKind, v.CompanionQuery, v.CompanionCR, v.CompanionType)
	if err != nil {
		http.Error(w, "error", http.StatusForbidden)
		return
	}
	v.CompanionHits = hits
	c.fillCompanionFilters(&v)
	c.Render.Render(w, "characters/companion_results.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) addCompanion(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	if readonly {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u := platform.UserFrom(r.Context())
	_, err := c.Svc.AddCompanion(ch, u.ID, companionInput(r))
	v := c.sheetView(r, ch, readonly)
	if err != nil {
		v.Error = ErrorKey(err)
		c.fillCompanionFilters(&v)
		c.Render.Render(w, "characters/sheet_companions.html", v.Lang, statusFor(err), v)
		return
	}
	got, _ := c.Svc.Get(ch.ID)
	c.Svc.Localize(got, v.Lang)
	c.renderCompanions(w, r, got, readonly, http.StatusOK)
}

func (c *Controller) editCompanion(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	if readonly {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := c.Svc.Repo.Companion(ch.ID, cid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v := c.sheetView(r, ch, readonly)
	v.EditCompanion = row
	v.DamageTypes = rules.DamageTypes()
	c.Render.Render(w, "characters/companion_edit.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) saveCompanion(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	if readonly {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	_, err = c.Svc.UpdateCompanion(ch, u.ID, cid, companionInput(r))
	if err != nil {
		row, _ := c.Svc.Repo.Companion(ch.ID, cid)
		v := c.sheetView(r, ch, readonly)
		v.EditCompanion = row
		v.DamageTypes = rules.DamageTypes()
		v.Error = ErrorKey(err)
		c.Render.Render(w, "characters/companion_edit.html", v.Lang, statusFor(err), v)
		return
	}
	got, _ := c.Svc.Get(ch.ID)
	c.Svc.Localize(got, platform.LangFrom(r.Context()))
	c.renderCompanions(w, r, got, readonly, http.StatusOK)
}

func (c *Controller) removeCompanion(w http.ResponseWriter, r *http.Request) {
	ch, readonly, ok := c.loadSheet(w, r)
	if !ok {
		return
	}
	if readonly {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.RemoveCompanion(ch, u.ID, cid); err != nil {
		http.Error(w, "error", http.StatusForbidden)
		return
	}
	got, _ := c.Svc.Get(ch.ID)
	c.Svc.Localize(got, platform.LangFrom(r.Context()))
	c.renderCompanions(w, r, got, readonly, http.StatusOK)
}

func (c *Controller) renderCompanions(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool, status int) {
	v := c.sheetView(r, ch, readonly)
	c.fillCompanionFilters(&v)
	c.Render.Render(w, "characters/sheet_companions.html", v.Lang, status, v)
}

func (c *Controller) fillCompanionFilters(v *showView) {
	crs, _ := c.Catalog.ListMonsterCRs()
	v.CompanionCRs = crs
	types, _ := c.Catalog.ListMonsterTypes()
	v.CompanionTypes = types
}

func (c *Controller) loadSheet(w http.ResponseWriter, r *http.Request) (*Character, bool, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false, false
	}
	ch, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false, false
	}
	u := platform.UserFrom(r.Context())
	readonly, err := c.Svc.CanView(ch, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false, false
	}
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	return ch, readonly, true
}

func companionInput(r *http.Request) AddCompanionInput {
	return AddCompanionInput{
		Kind:             platform.FormTrim(r, "kind"),
		CatalogMonsterID: platform.FormInt64(r, "monster_id"),
		ItemID:           platform.FormInt64(r, "item_id"),
		Name:             platform.FormTrim(r, "name"),
		AC:               platform.FormInt(r, "ac"),
		HPMax:            platform.FormInt(r, "hp_max"),
		Scores: rules.AbilityScores{
			STR: platform.FormInt(r, "str"),
			DEX: platform.FormInt(r, "dex"),
			CON: platform.FormInt(r, "con"),
			INT: platform.FormInt(r, "int"),
			WIS: platform.FormInt(r, "wis"),
			CHA: platform.FormInt(r, "cha"),
		},
		Resistances:     platform.FormList(r, "resist"),
		Immunities:      platform.FormList(r, "immune"),
		Vulnerabilities: platform.FormList(r, "vuln"),
	}
}

func statusFor(err error) int {
	switch {
	case errorsIsCompanion(err):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusBadRequest
	}
}

func errorsIsCompanion(err error) bool {
	return errors.Is(err, ErrCompanionGate) || errors.Is(err, ErrCompanionLimit) ||
		errors.Is(err, ErrCompanionKind) || errors.Is(err, ErrCompanionCR) ||
		errors.Is(err, ErrCompanionType) || errors.Is(err, ErrCompanionMissing) ||
		errors.Is(err, ErrNotOwner) || errors.Is(err, ErrForbidden)
}
