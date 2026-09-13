package characters

import (
	"net/http"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type checkView struct {
	platform.BaseView
	Character *Character
	Kind      string
	Slug      string
	Label     string
	Ability   string
	Bonus     int
	Formula   string
	ReadOnly  bool
	Roll      *rules.RollResult
}

func (c *Controller) renderSkills(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_skills.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) toggleSkill(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	slug := r.PathValue("slug")
	prof := platform.FormTrim(r, "proficient") == "1"
	exp := platform.FormTrim(r, "expertise") == "1"
	if err := c.Svc.SetSkill(ch, platform.UserFrom(r.Context()).ID, slug, prof, exp); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderSkills(w, r, ch, false)
}

func (c *Controller) toggleSave(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	ab := strings.ToLower(r.PathValue("ability"))
	prof := platform.FormTrim(r, "proficient") == "1"
	if err := c.Svc.SetSave(ch, platform.UserFrom(r.Context()).ID, ab, prof); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderSkills(w, r, ch, false)
}

func (c *Controller) skillCheckModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	readonly, _ := c.Svc.CanView(ch, platform.UserFrom(r.Context()).ID)
	c.renderCheck(w, r, ch, readonly, "skill", r.PathValue("slug"), nil)
}

func (c *Controller) saveCheckModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	readonly, _ := c.Svc.CanView(ch, platform.UserFrom(r.Context()).ID)
	c.renderCheck(w, r, ch, readonly, "save", strings.ToLower(r.PathValue("ability")), nil)
}

func (c *Controller) rollSkillCheck(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	c.rollCheck(w, r, ch, "skill", r.PathValue("slug"))
}

func (c *Controller) rollSaveCheck(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	c.rollCheck(w, r, ch, "save", strings.ToLower(r.PathValue("ability")))
}

func (c *Controller) rollCheck(w http.ResponseWriter, r *http.Request, ch *Character, kind, key string) {
	v := c.checkData(r, ch, false, kind, key)
	if v.Formula == "" {
		v.Error = "error.check.unknown"
		c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	roll, err := rules.RollFormula(v.Formula)
	if err != nil {
		v.Error = ErrorKey(err)
		c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	v.Roll = &roll
	c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderCheck(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool, kind, key string, roll *rules.RollResult) {
	v := c.checkData(r, ch, readonly, kind, key)
	v.Roll = roll
	if v.Formula == "" {
		http.NotFound(w, r)
		return
	}
	isOwner := ch.OwnerID == platform.UserFrom(r.Context()).ID
	v.ReadOnly = readonly || !isOwner
	c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) checkData(r *http.Request, ch *Character, readonly bool, kind, key string) checkView {
	v := checkView{
		BaseView:  c.base(r, ch.Name),
		Character: ch,
		Kind:      kind,
		Slug:      key,
		ReadOnly:  readonly,
	}
	if kind == "skill" {
		sk, ok := rules.SkillBySlug(key)
		if !ok {
			return v
		}
		mark := ch.EffectiveSkillMark(key)
		v.Ability = sk.Ability
		v.Label = catalog.Pick(platform.LangFrom(r.Context()), sk.NameEN, sk.NameRU)
		v.Bonus = rules.SkillBonus(ch.Scores(), ch.Level, mark)
		v.Formula = rules.CheckFormula(v.Bonus)
		return v
	}
	if !rules.ValidAbilityKey(key) {
		return v
	}
	mark := rules.SaveMark{Ability: key}
	for _, m := range ch.SaveMarks {
		if strings.EqualFold(m.Ability, key) {
			mark = m
			break
		}
	}
	v.Ability = key
	v.Bonus = rules.SaveBonus(ch.Scores(), ch.Level, mark)
	v.Formula = rules.CheckFormula(v.Bonus)
	return v
}
