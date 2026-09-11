package characters

import (
	"net/http"
	"strconv"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type levelUpView struct {
	platform.BaseView
	Character        *Character
	Classes          []catalog.Class
	ExistingIDs      map[int64]int
	SelectedClass    int64
	Subclasses       []catalog.Subclass
	SubclassRequired bool
	SubclassAt       int
	CurrentSubclass  string
	HPMode           string
	HPRoll           int
	HitDie           int
	AverageHP        int
	ASIRequired      bool
	ASIA             string
	ASIB             string
	Preview          *rules.Delta
	AbilityKeys      []string
}

func (c *Controller) showLevelUp(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.RequireOwner(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if ch.Level >= 20 {
		v := c.base(r, "character.levelup.title")
		v.Error = "error.level.max"
		c.Render.Render(w, "characters/levelup.html", v.Lang, http.StatusUnprocessableEntity, levelUpView{
			BaseView: v, Character: ch,
		})
		return
	}
	in := c.levelUpIntent(r, ch)
	v := c.levelUpData(r, ch, in)
	if prev, err := c.Svc.PreviewLevelUp(ch, in); err == nil {
		v.Preview = prev
		v.ASIRequired = prev.ASIRequired
		v.SubclassRequired = prev.SubclassNeeded
		v.HitDie = prev.HitDie
		v.AverageHP = prev.AverageHP
	} else if v.Error == "" {
		v.Error = ErrorKey(err)
	}
	c.Render.Render(w, "characters/levelup.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) previewLevelUp(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.RequireOwner(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	in := c.levelUpIntent(r, ch)
	v := c.levelUpData(r, ch, in)
	if prev, err := c.Svc.PreviewLevelUp(ch, in); err == nil {
		v.Preview = prev
		v.ASIRequired = prev.ASIRequired
		v.SubclassRequired = prev.SubclassNeeded
		v.HitDie = prev.HitDie
		v.AverageHP = prev.AverageHP
	} else {
		v.Error = ErrorKey(err)
	}
	c.Render.Render(w, "characters/levelup_dynamic.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) postLevelUp(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.RequireOwner(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	in := c.levelUpIntent(r, ch)
	next, err := c.Svc.ApplyLevelUp(ch, u.ID, in)
	if err != nil {
		v := c.levelUpData(r, ch, in)
		v.Error = ErrorKey(err)
		if prev, e2 := c.Svc.PreviewLevelUp(ch, in); e2 == nil {
			v.Preview = prev
		}
		c.Render.Render(w, "characters/levelup.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	platform.Redirect(w, r, "/characters/"+strconv.FormatInt(next.ID, 10))
}

func (c *Controller) levelUpIntent(r *http.Request, ch *Character) rules.Intent {
	classID := platform.FormInt64(r, "class_id")
	if classID == 0 && r.Method == http.MethodGet {
		if len(ch.ClassLevels) > 0 {
			classID = ch.ClassLevels[0].ClassID
		}
	}
	mode := platform.FormTrim(r, "hp_mode")
	if mode == "" {
		mode = rules.HPAverage
	}
	asiA := platform.FormTrim(r, "asi_a")
	asiB := platform.FormTrim(r, "asi_b")
	if asiA == "" {
		asiA = "str"
	}
	if asiB == "" {
		asiB = "str"
	}
	in := rules.Intent{
		Kind:       rules.IntentLevelUp,
		ClassID:    classID,
		SubclassID: platform.FormInt64(r, "subclass_id"),
		HPMode:     mode,
		HPRoll:     platform.FormInt(r, "hp_roll"),
		ASI:        parseASI(r),
		Lang:       platform.LangFrom(r.Context()),
	}
	if in.ASI.Sum() == 0 && (asiA != "" || asiB != "") {
		in.ASI = rules.AbilityScores{}
		if rules.ValidAbilityKey(asiA) {
			in.ASI.AddKey(asiA, 1)
		}
		if rules.ValidAbilityKey(asiB) {
			in.ASI.AddKey(asiB, 1)
		}
	}
	return in
}

func (c *Controller) levelUpData(r *http.Request, ch *Character, in rules.Intent) levelUpView {
	classes, _ := c.Catalog.ListClasses()
	existing := map[int64]int{}
	for _, cl := range ch.ClassLevels {
		existing[cl.ClassID] = cl.Levels
	}
	v := levelUpView{
		BaseView:      c.base(r, "character.levelup.title"),
		Character:     ch,
		Classes:       classes,
		ExistingIDs:   existing,
		SelectedClass: in.ClassID,
		HPMode:        in.HPMode,
		HPRoll:        in.HPRoll,
		ASIA:          firstASIKey(in.ASI, 0),
		ASIB:          firstASIKey(in.ASI, 1),
		AbilityKeys:   rules.AbilityKeys,
		HitDie:        8,
		AverageHP:     5,
	}
	if in.ClassID != 0 {
		if class, err := c.Catalog.Class(in.ClassID); err == nil {
			v.HitDie = class.HitDie
			v.AverageHP = rules.AverageHitPoints(class.HitDie)
			v.SubclassAt = class.SubclassLevel
			nextLevel := existing[in.ClassID] + 1
			current := ch.State().ClassLevel(in.ClassID)
			v.CurrentSubclass = current.Subclass
			v.SubclassRequired = class.SubclassLevel == nextLevel && current.SubclassID == 0
			v.ASIRequired = class.HasASI(nextLevel)
			subs, _ := c.Catalog.ListSubclasses(in.ClassID)
			v.Subclasses = subs
		}
	}
	if v.ASIA == "" {
		v.ASIA = "str"
	}
	if v.ASIB == "" {
		v.ASIB = "str"
	}
	return v
}

func firstASIKey(a rules.AbilityScores, which int) string {
	seen := 0
	for _, k := range rules.AbilityKeys {
		n := a.Get(k)
		for i := 0; i < n; i++ {
			if seen == which {
				return k
			}
			seen++
		}
	}
	return "str"
}
