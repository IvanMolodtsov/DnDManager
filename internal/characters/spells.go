package characters

import (
	"net/http"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type searchView struct {
	platform.BaseView
	Character *Character
	Query     string
	Results   []catalog.Spell
	IsOwner   bool
}

type useView struct {
	platform.BaseView
	Character   *Character
	Spell       catalog.Spell
	Stats       string
	Formula     string
	Heal        string
	SlotLevel   int
	HasPact     bool
	HasSlots    bool
	Pact        rules.Pool
	SlotChoices []int
	DefaultKind string
	CanCast     bool
	ReadOnly    bool
	Roll        *rules.RollResult
	Amount      int
	ItemGrant   bool
}

func (c *Controller) ownerSheet(w http.ResponseWriter, r *http.Request) (*Character, bool) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.RequireOwner(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return ch, true
}

func (c *Controller) renderMagic(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_magic.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) searchSpells(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	var results []catalog.Spell
	if len(q) >= 1 {
		results, _ = c.Catalog.SearchSpells(q, 20)
	}
	c.Render.Render(w, "characters/spell_search.html", platform.LangFrom(r.Context()), http.StatusOK, searchView{
		BaseView: c.base(r, ch.Name), Character: ch, Query: q, Results: results, IsOwner: true,
	})
}

func (c *Controller) addSpell(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	spellID := platform.FormInt64(r, "spell_id")
	prepared := platform.FormTrim(r, "prepared") == "1"
	if err := c.Svc.AddLearned(ch, platform.UserFrom(r.Context()).ID, spellID, prepared); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderMagic(w, r, ch, false)
}

func (c *Controller) togglePrepared(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	spellID, err := platform.PathID(r, "spellID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	prepared := platform.FormTrim(r, "prepared") == "1"
	if err := c.Svc.SetPrepared(ch, platform.UserFrom(r.Context()).ID, spellID, prepared); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderMagic(w, r, ch, false)
}

func (c *Controller) removeSpell(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	spellID, err := platform.PathID(r, "spellID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.RemoveLearned(ch, platform.UserFrom(r.Context()).ID, spellID); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderMagic(w, r, ch, false)
}

func (c *Controller) useSpellModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, err := c.Svc.CanView(ch, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v, ok := c.buildUse(w, r, ch, readonly)
	if !ok {
		return
	}
	c.Render.Render(w, "characters/spell_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) rollSpell(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, err := c.Svc.CanView(ch, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	spellID, err := platform.PathID(r, "spellID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	slotLevel := platform.FormInt(r, "slot_level")
	res, _, err := c.Svc.RollSpell(ch, spellID, slotLevel)
	v, ok := c.buildUse(w, r, ch, readonly)
	if !ok {
		return
	}
	if err != nil {
		v.Error = ErrorKey(err)
	} else {
		v.Roll = &res
	}
	c.Render.Render(w, "characters/spell_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) castSpell(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	spellID, err := platform.PathID(r, "spellID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	kind := platform.FormTrim(r, "pool")
	slotLevel := platform.FormInt(r, "slot_level")
	amount := platform.FormInt(r, "amount")
	if amount < 1 {
		amount = 1
	}
	if err := c.Svc.CastSpell(ch, platform.UserFrom(r.Context()).ID, spellID, kind, slotLevel, amount); err != nil {
		v, ok := c.buildUse(w, r, ch, false)
		if !ok {
			return
		}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "characters/spell_use.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	w.Header().Set("HX-Trigger", "close-spell-modal")
	c.renderMagicCast(w, r, ch, false)
}

func (c *Controller) spendResource(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	kind := platform.FormTrim(r, "kind")
	slotLevel := platform.FormInt(r, "slot_level")
	amount := platform.FormInt(r, "amount")
	if amount < 1 {
		amount = 1
	}
	if err := c.Svc.SpendResource(ch, platform.UserFrom(r.Context()).ID, kind, slotLevel, amount); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderMagic(w, r, ch, false)
}

func (c *Controller) buildUse(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) (useView, bool) {
	spellID, err := platform.PathID(r, "spellID")
	if err != nil {
		http.NotFound(w, r)
		return useView{}, false
	}
	ls, ok := c.Svc.learned(ch, spellID)
	fromItem := false
	var sp catalog.Spell
	prepared := false
	if ok {
		sp = ls.Spell
		prepared = ls.Prepared
		if _, gok := c.Svc.grantedSpell(ch, spellID); gok {
			fromItem = platform.FormTrim(r, "pool") == rules.KindItem || r.URL.Query().Get("pool") == rules.KindItem
		}
	} else if g, gok := c.Svc.grantedSpell(ch, spellID); gok {
		sp = g.Spell
		prepared = true
		fromItem = true
	} else {
		http.NotFound(w, r)
		return useView{}, false
	}
	pact, hasPact := rules.PactPool(ch.Resources)
	hasSlots := rules.HasKind(ch.Resources, rules.KindSlots)
	slotLevel := platform.FormInt(r, "slot_level")
	if slotLevel < 1 {
		if hasPact && !hasSlots && !fromItem {
			slotLevel = pact.SlotLevel
		} else {
			slotLevel = sp.Level
		}
	}
	if hasPact && !hasSlots && !fromItem {
		slotLevel = pact.SlotLevel
	}
	if slotLevel < sp.Level {
		slotLevel = sp.Level
	}
	formula, _, heal := sp.FormulaAt(slotLevel, ch.Level)
	kind := platform.FormTrim(r, "pool")
	if kind == "" {
		kind = r.URL.Query().Get("pool")
	}
	if fromItem {
		kind = rules.KindItem
	} else if kind == "" {
		if hasPact && !hasSlots {
			kind = rules.KindPact
		} else {
			kind = rules.KindSlots
		}
	}
	var choices []int
	if !fromItem {
		for lv := sp.Level; lv <= 9; lv++ {
			if rules.SlotRemaining(ch.Resources, lv) > 0 {
				choices = append(choices, lv)
			}
		}
	}
	amount := platform.FormInt(r, "amount")
	if amount < 1 {
		amount = 1
	}
	return useView{
		BaseView:    c.base(r, sp.Name(platform.LangFrom(r.Context()))),
		Character:   ch,
		Spell:       sp,
		Stats:       sp.StatsLine(slotLevel, ch.Level),
		Formula:     formula,
		Heal:        heal,
		SlotLevel:   slotLevel,
		HasPact:     !fromItem && hasPact && pact.SlotLevel >= sp.Level && pact.Current > 0,
		HasSlots:    !fromItem && hasSlots && len(choices) > 0,
		Pact:        pact,
		SlotChoices: choices,
		DefaultKind: kind,
		CanCast:     !readonly && prepared && (fromItem || canCast(ch, sp)),
		ReadOnly:    readonly,
		Amount:      amount,
		ItemGrant:   fromItem,
	}, true
}
