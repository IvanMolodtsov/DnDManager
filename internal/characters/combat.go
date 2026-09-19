package characters

import (
	"net/http"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

// HPResult is the short apply line shown in the HP modal and on #sheet-vitals.
type HPResult struct {
	Kind     string
	Incoming int
	TypeName string
	ModKey   string
	Applied  int
	Absorbed int
	Amount   int
}

func (c *Controller) renderCombat(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_vitals.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderMagicCast(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	v.OOBCombat = true
	c.Render.Render(w, "characters/sheet_after_cast.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) afterCombatEdit(w http.ResponseWriter, r *http.Request, ch *Character) {
	c.afterCombatEditNote(w, r, ch, nil)
}

func (c *Controller) afterCombatEditNote(w http.ResponseWriter, r *http.Request, ch *Character, note *HPResult) {
	ch, _ = c.Svc.Get(ch.ID)
	u := platform.UserFrom(r.Context())
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.HPResult = note
	c.Render.Render(w, "characters/sheet_vitals.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) hpAdjustModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.combatSheet(w, r)
	if !ok {
		return
	}
	pool := r.URL.Query().Get("pool")
	sign := r.URL.Query().Get("sign")
	if !validAdjust(pool, sign) {
		http.Error(w, "error.generic", http.StatusBadRequest)
		return
	}
	v := c.adjustView(r, ch, pool, sign, 1, "slashing")
	c.Render.Render(w, "characters/hp_adjust.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) applyHPAdjust(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.combatSheet(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	pool := platform.FormTrim(r, "pool")
	sign := platform.FormTrim(r, "sign")
	amount := platform.FormInt(r, "amount")
	dmgType := platform.FormTrim(r, "damage_type")
	if !validAdjust(pool, sign) {
		http.Error(w, ErrorKey(rules.ErrAmount), http.StatusUnprocessableEntity)
		return
	}
	lang := platform.LangFrom(r.Context())
	if pool == "temp" {
		delta := amount
		if sign == "-" {
			delta = -amount
		}
		if amount < 1 {
			c.hpAdjustErr(w, r, ch, pool, sign, amount, dmgType, rules.ErrAmount)
			return
		}
		if err := c.Svc.AdjustTempHP(ch, u.ID, delta); err != nil {
			c.hpAdjustErr(w, r, ch, pool, sign, amount, dmgType, err)
			return
		}
		kind := "temp_up"
		if sign == "-" {
			kind = "temp_down"
		}
		c.renderAdjustResult(w, r, ch, pool, sign, amount, dmgType, &HPResult{Kind: kind, Amount: amount})
		return
	}
	if sign == "+" {
		if err := c.Svc.ApplyHPHeal(ch, u.ID, amount); err != nil {
			c.hpAdjustErr(w, r, ch, pool, sign, amount, dmgType, err)
			return
		}
		c.renderAdjustResult(w, r, ch, pool, sign, amount, dmgType, &HPResult{Kind: "heal", Amount: amount})
		return
	}
	res, err := c.Svc.ApplyHPDamage(ch, u.ID, amount, dmgType)
	if err != nil {
		c.hpAdjustErr(w, r, ch, pool, sign, amount, dmgType, err)
		return
	}
	c.renderAdjustResult(w, r, ch, pool, sign, amount, dmgType, &HPResult{
		Kind:     "damage",
		Incoming: res.Incoming,
		TypeName: rules.DamageTypeName(res.DamageType, lang),
		ModKey:   res.NoteKey(),
		Applied:  res.Applied,
		Absorbed: res.AbsorbedTemp,
	})
}

func (c *Controller) renderAdjustResult(w http.ResponseWriter, r *http.Request, ch *Character, pool, sign string, amount int, dmgType string, note *HPResult) {
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.adjustView(r, ch, pool, sign, amount, dmgType)
	v.HPResult = note
	v.OOBCombat = true
	v.ReadOnly = readonly
	c.Render.Render(w, "characters/hp_adjust_after.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) hpAdjustErr(w http.ResponseWriter, r *http.Request, ch *Character, pool, sign string, amount int, dmgType string, err error) {
	v := c.adjustView(r, ch, pool, sign, amount, dmgType)
	v.Error = ErrorKey(err)
	w.Header().Set("HX-Retarget", "#hp-adjust-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "characters/hp_adjust.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func (c *Controller) adjustView(r *http.Request, ch *Character, pool, sign string, amount int, dmgType string) showView {
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.AdjustPool = pool
	v.AdjustSign = sign
	v.AdjustAmount = amount
	v.AdjustDamageType = dmgType
	v.DamageTypes = rules.DamageTypes()
	return v
}

func validAdjust(pool, sign string) bool {
	if pool != "hp" && pool != "temp" {
		return false
	}
	return sign == "+" || sign == "-"
}

func (c *Controller) toggleDeath(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.combatSheet(w, r)
	if !ok {
		return
	}
	fail := platform.FormTrim(r, "track") == "fail"
	pip := platform.FormInt(r, "pip")
	if err := c.Svc.ToggleDeath(ch, platform.UserFrom(r.Context()).ID, fail, pip); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	c.afterCombatEdit(w, r, ch)
}

func (c *Controller) dismissEffect(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.combatSheet(w, r)
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
	c.afterCombatEdit(w, r, ch)
}
