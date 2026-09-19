package characters

import (
	"net/http"
	"strconv"
	"strings"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

func (c *Controller) statusAddModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.dmSheet(w, r)
	if !ok {
		return
	}
	v := c.statusFormView(r, ch, r.URL.Query().Get("q"))
	c.Render.Render(w, "characters/status_add.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) searchConditions(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.dmSheet(w, r)
	if !ok {
		return
	}
	v := c.statusFormView(r, ch, r.URL.Query().Get("q"))
	c.Render.Render(w, "characters/status_results.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) addStatus(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.dmSheet(w, r)
	if !ok {
		return
	}
	spec := statusSpecFromForm(r)
	u := platform.UserFrom(r.Context())
	if err := c.Svc.AddStatus(ch, u.ID, spec); err != nil {
		v := c.statusFormView(r, ch, platform.FormTrim(r, "q"))
		v.Error = ErrorKey(err)
		w.Header().Set("HX-Retarget", "#status-add-body")
		w.Header().Set("HX-Reswap", "innerHTML")
		c.Render.Render(w, "characters/status_add.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.OOBCombat = true
	c.Render.Render(w, "characters/status_add_after.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) statusFormView(r *http.Request, ch *Character, q string) showView {
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.StatusQuery = q
	v.DefaultRemoveEnd = false
	v.StatusNewURL = "/characters/" + strconv.FormatInt(ch.ID, 10) + "/effects/new"
	v.StatusSearchURL = "/characters/" + strconv.FormatInt(ch.ID, 10) + "/effects/search"
	v.StatusPostURL = "/characters/" + strconv.FormatInt(ch.ID, 10) + "/effects"
	limit := 20
	if strings.TrimSpace(q) == "" {
		limit = 16
	}
	results, _ := c.Catalog.SearchConditions(q, limit)
	v.StatusResults = results
	cid := platform.FormInt64(r, "condition_id")
	if cid == 0 {
		if n := r.URL.Query().Get("condition_id"); n != "" {
			cid, _ = strconv.ParseInt(n, 10, 64)
		}
	}
	if cid > 0 {
		if cond, err := c.Catalog.Condition(cid); err == nil {
			v.StatusPick = cond
		}
	}
	v.DamageTypes = rules.DamageTypes()
	return v
}

func statusSpecFromForm(r *http.Request) StatusSpec {
	return StatusSpec{
		CatalogID:         platform.FormInt64(r, "condition_id"),
		NameEN:            platform.FormTrim(r, "name_en"),
		NameRU:            platform.FormTrim(r, "name_ru"),
		DamageFormula:     platform.FormTrim(r, "damage_formula"),
		DamageType:        platform.FormTrim(r, "damage_type"),
		ACBonus:           platform.FormInt(r, "ac_bonus"),
		DurationTurns:     platform.FormInt(r, "duration_turns"),
		RemoveOnBattleEnd: platform.FormTrim(r, "remove_on_battle_end") == "1",
		Hidden:            platform.FormTrim(r, "hidden") == "1",
		SourceURL:         platform.FormTrim(r, "source_url"),
	}
}
