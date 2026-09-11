package characters

import (
	"net/http"
	"strconv"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type wizardView struct {
	platform.BaseView
	Campaign         *campaigns.Campaign
	Draft            *Draft
	Step             int
	Races            []catalog.Race
	Backgrounds      []catalog.Background
	Classes          []catalog.Class
	Subclasses       []catalog.Subclass
	SubclassRequired bool
	SubclassAt       int
	BonusLines       []BonusLine
	ScoreRows        []ScoreRow
	ArrayValues      []int
	Preview          *rules.Delta
}

func (c *Controller) showWizard(w http.ResponseWriter, r *http.Request) {
	camp, ok := c.memberCampaign(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	d, err := c.Svc.EnsureDraft(u.ID, camp.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	c.renderWizard(w, r, camp, d, http.StatusOK)
}

func (c *Controller) postWizard(w http.ResponseWriter, r *http.Request) {
	camp, ok := c.memberCampaign(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	d, err := c.Svc.EnsureDraft(u.ID, camp.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	action := platform.FormTrim(r, "action")
	if action == "reset" {
		_ = c.Svc.ResetDraft(d)
		platform.Redirect(w, r, "/campaigns/"+itoa64(camp.ID)+"/characters/new")
		return
	}
	c.applyWizardForm(d, r)
	lang := platform.LangFrom(r.Context())

	switch action {
	case "back":
		if d.Step > 1 {
			d.Step--
		}
		_ = c.Svc.SaveDraft(d)
		c.renderWizard(w, r, camp, d, http.StatusOK)
		return
	case "confirm":
		d.Step = 4
		ch, err := c.Svc.ConfirmDraft(d, u.ID, lang)
		if err != nil {
			v := c.wizardData(r, camp, d)
			v.Error = ErrorKey(err)
			c.Render.Render(w, "characters/wizard.html", v.Lang, http.StatusUnprocessableEntity, v)
			return
		}
		platform.Redirect(w, r, "/characters/"+itoa64(ch.ID))
		return
	default: // next
		if err := c.validateWizardStep(d); err != nil {
			v := c.wizardData(r, camp, d)
			v.Error = ErrorKey(err)
			c.Render.Render(w, "characters/wizard.html", v.Lang, http.StatusUnprocessableEntity, v)
			return
		}
		if d.Step < 4 {
			d.Step++
		}
		_ = c.Svc.SaveDraft(d)
		c.renderWizard(w, r, camp, d, http.StatusOK)
	}
}

func (c *Controller) applyWizardForm(d *Draft, r *http.Request) {
	switch d.Step {
	case 1:
		d.Name = platform.FormTrim(r, "name")
		d.RaceID = platform.FormInt64(r, "race_id")
		d.BackgroundID = platform.FormInt64(r, "background_id")
	case 2:
		d.ClassID = platform.FormInt64(r, "class_id")
		d.SubclassID = platform.FormInt64(r, "subclass_id")
		if class, err := c.Catalog.Class(d.ClassID); err == nil && class.SubclassLevel > 1 {
			d.SubclassID = 0
		}
	case 3:
		d.BaseScores = rules.AbilityScores{
			STR: platform.FormInt(r, "str"),
			DEX: platform.FormInt(r, "dex"),
			CON: platform.FormInt(r, "con"),
			INT: platform.FormInt(r, "int"),
			WIS: platform.FormInt(r, "wis"),
			CHA: platform.FormInt(r, "cha"),
		}
		d.HasScores = true
	}
}

func (c *Controller) validateWizardStep(d *Draft) error {
	switch d.Step {
	case 1:
		if len([]rune(d.Name)) == 0 || len([]rune(d.Name)) > 80 {
			return ErrNameRequired
		}
		if d.RaceID == 0 {
			return rules.ErrRaceRequired
		}
		if d.BackgroundID == 0 {
			return rules.ErrBackgroundRequired
		}
		if _, err := c.Catalog.Race(d.RaceID); err != nil {
			return rules.ErrRaceRequired
		}
		if _, err := c.Catalog.Background(d.BackgroundID); err != nil {
			return rules.ErrBackgroundRequired
		}
	case 2:
		if d.ClassID == 0 {
			return rules.ErrClassRequired
		}
		class, err := c.Catalog.Class(d.ClassID)
		if err != nil {
			return rules.ErrClassRequired
		}
		if class.SubclassLevel <= 1 && d.SubclassID == 0 {
			return rules.ErrSubclassRequired
		}
		if d.SubclassID != 0 {
			sub, err := c.Catalog.Subclass(d.SubclassID)
			if err != nil || sub.ClassID != class.ID {
				return rules.ErrSubclassInvalid
			}
		}
	case 3:
		if err := rules.ValidateStandardArray(d.BaseScores); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) subclassPartial(w http.ResponseWriter, r *http.Request) {
	camp, ok := c.memberCampaign(w, r)
	if !ok {
		return
	}
	classID := platform.FormInt64(r, "class_id")
	u := platform.UserFrom(r.Context())
	d, err := c.Svc.EnsureDraft(u.ID, camp.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	d.ClassID = classID
	d.SubclassID = 0
	v := c.wizardData(r, camp, d)
	c.Render.Render(w, "characters/wizard_subclass.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) scoresPartial(w http.ResponseWriter, r *http.Request) {
	camp, ok := c.memberCampaign(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	d, err := c.Svc.EnsureDraft(u.ID, camp.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	d.BaseScores = rules.AbilityScores{
		STR: platform.FormInt(r, "str"),
		DEX: platform.FormInt(r, "dex"),
		CON: platform.FormInt(r, "con"),
		INT: platform.FormInt(r, "int"),
		WIS: platform.FormInt(r, "wis"),
		CHA: platform.FormInt(r, "cha"),
	}
	d.HasScores = true
	v := c.wizardData(r, camp, d)
	c.Render.Render(w, "characters/wizard_scores.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderWizard(w http.ResponseWriter, r *http.Request, camp *campaigns.Campaign, d *Draft, status int) {
	v := c.wizardData(r, camp, d)
	c.Render.Render(w, "characters/wizard.html", v.Lang, status, v)
}

func (c *Controller) wizardData(r *http.Request, camp *campaigns.Campaign, d *Draft) wizardView {
	lang := platform.LangFrom(r.Context())
	v := wizardView{
		BaseView:    c.base(r, "character.create.title"),
		Campaign:    camp,
		Draft:       d,
		Step:        d.Step,
		ArrayValues: rules.StandardArray,
	}
	races, _ := c.Catalog.ListRaces()
	bgs, _ := c.Catalog.ListBackgrounds()
	classes, _ := c.Catalog.ListClasses()
	v.Races, v.Backgrounds, v.Classes = races, bgs, classes
	if d.ClassID != 0 {
		if class, err := c.Catalog.Class(d.ClassID); err == nil {
			v.SubclassAt = class.SubclassLevel
			v.SubclassRequired = class.SubclassLevel <= 1
			subs, _ := c.Catalog.ListSubclasses(d.ClassID)
			v.Subclasses = subs
		}
	}
	v.BonusLines = c.Svc.BonusLines(d, lang)
	bonus := c.Svc.CreationBonuses(d)
	base := d.BaseScores
	if !d.HasScores {
		base = defaultArray()
	}
	v.ScoreRows = scoreRows(base, bonus)
	if d.Step >= 4 && d.RaceID != 0 && d.ClassID != 0 {
		if prev, err := c.Svc.PreviewCreate(d, lang); err == nil {
			v.Preview = prev
		} else {
			v.Error = ErrorKey(err)
		}
	}
	return v
}

func itoa64(n int64) string {
	return strconv.FormatInt(n, 10)
}
