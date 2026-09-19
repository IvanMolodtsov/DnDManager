package battles

import (
	"net/http"
	"strconv"
	"strings"

	"dndmanager/internal/characters"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

func (c *Controller) statusAddModal(w http.ResponseWriter, r *http.Request) {
	v, u, ok := c.dmUnitView(w, r)
	if !ok {
		return
	}
	c.fillStatusForm(v, r, u, r.URL.Query().Get("q"))
	c.Render.Render(w, "battles/status_add.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) searchConditions(w http.ResponseWriter, r *http.Request) {
	v, u, ok := c.dmUnitView(w, r)
	if !ok {
		return
	}
	c.fillStatusForm(v, r, u, r.URL.Query().Get("q"))
	c.Render.Render(w, "characters/status_results.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) addUnitStatus(w http.ResponseWriter, r *http.Request) {
	v, unit, ok := c.dmUnitView(w, r)
	if !ok {
		return
	}
	id, _ := platform.PathID(r, "id")
	user := platform.UserFrom(r.Context())
	spec := statusSpecFromForm(r)
	if _, err := c.Svc.AddUnitStatus(id, user.ID, unit.ID, spec); err != nil {
		c.fillStatusForm(v, r, unit, platform.FormTrim(r, "q"))
		v.Error = ErrorKey(err)
		w.Header().Set("HX-Retarget", "#status-add-body")
		w.Header().Set("HX-Reswap", "innerHTML")
		c.Render.Render(w, "battles/status_add.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	fresh, ok := c.loadView(w, r)
	if !ok {
		return
	}
	fresh.OOBBattle = true
	fresh.StatusUnit = unit
	c.Render.Render(w, "battles/status_add_after.html", fresh.Lang, http.StatusOK, fresh)
}

func (c *Controller) removeUnitStatus(w http.ResponseWriter, r *http.Request) {
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	eid, err := platform.PathID(r, "eid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) {
		return c.Svc.RemoveUnitStatus(campaignID, userID, uid, eid)
	})
}

func (c *Controller) dmUnitView(w http.ResponseWriter, r *http.Request) (*boardView, *Unit, bool) {
	v, ok := c.loadView(w, r)
	if !ok {
		return nil, nil, false
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, nil, false
	}
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return nil, nil, false
	}
	if v.Battle == nil {
		http.NotFound(w, r)
		return nil, nil, false
	}
	u, err := c.Svc.unitIn(v.Battle, uid)
	if err != nil {
		http.NotFound(w, r)
		return nil, nil, false
	}
	return v, u, true
}

func (c *Controller) fillStatusForm(v *boardView, r *http.Request, u *Unit, q string) {
	v.StatusUnit = u
	v.StatusQuery = q
	v.DefaultRemoveEnd = true
	v.StatusNewURL = "/campaigns/" + strconv.FormatInt(v.Campaign.ID, 10) + "/battle/units/" + strconv.FormatInt(u.ID, 10) + "/effects/new"
	v.StatusSearchURL = "/campaigns/" + strconv.FormatInt(v.Campaign.ID, 10) + "/battle/units/" + strconv.FormatInt(u.ID, 10) + "/effects/search"
	v.StatusPostURL = "/campaigns/" + strconv.FormatInt(v.Campaign.ID, 10) + "/battle/units/" + strconv.FormatInt(u.ID, 10) + "/effects"
	v.DamageTypes = rules.DamageTypes()
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
}

func statusSpecFromForm(r *http.Request) characters.StatusSpec {
	return characters.StatusSpec{
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
