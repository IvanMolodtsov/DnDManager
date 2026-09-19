package characters

import (
	"net/http"
	"strconv"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

func (c *Controller) MountMutations(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /campaigns/{id}/characters/{cid}/mutations/new", auth(http.HandlerFunc(c.mutationModal)))
	mux.Handle("POST /campaigns/{id}/characters/{cid}/mutations/roll", auth(http.HandlerFunc(c.mutationRoll)))
	mux.Handle("POST /campaigns/{id}/characters/{cid}/mutations/pick", auth(http.HandlerFunc(c.mutationPick)))
	mux.Handle("POST /campaigns/{id}/characters/{cid}/mutations", auth(http.HandlerFunc(c.mutationCreate)))
	mux.Handle("POST /campaigns/{id}/characters/{cid}/mutations/{mid}/features", auth(http.HandlerFunc(c.mutationWizardFeature)))
	mux.Handle("GET /characters/{id}/mutations", auth(http.HandlerFunc(c.mutationsIsland)))
	mux.Handle("POST /characters/{id}/mutations/{mid}/features", auth(http.HandlerFunc(c.mutationAddFeature)))
	mux.Handle("POST /characters/{id}/mutations/{mid}/features/{fid}/remove", auth(http.HandlerFunc(c.mutationRemoveFeature)))
	mux.Handle("POST /characters/{id}/mutations/{mid}/remove", auth(http.HandlerFunc(c.mutationRemove)))
	mux.Handle("GET /characters/{id}/mutations/{mid}/attack", auth(http.HandlerFunc(c.mutationAttackModal)))
	mux.Handle("POST /characters/{id}/mutations/{mid}/attack/roll", auth(http.HandlerFunc(c.mutationAttackRoll)))
}

type mutationView struct {
	platform.BaseView
	Character   *Character
	CampaignID  int64
	IsDM        bool
	IsOwner     bool
	Step        string
	CR          int
	Size        string
	Type        string
	TypeChoose  bool
	SizeFace    int
	TypeFace    int
	BodyFace    int
	BodyPart    string
	BodyChoose  bool
	Monster     *catalog.Monster
	Mutation    *Mutation
	Error       string
	PHBTypes    []string
	Sizes       []string
	BodyParts   []string
	AbilityKeys []string
	Skills      []rules.Skill
	DamageTypes []rules.NamedOption
	Slots       []string
	Conditions  []rules.NamedOption
}

func (c *Controller) mutationModal(w http.ResponseWriter, r *http.Request) {
	v, ok := c.mutationBase(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v.Step = "pick"
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationRoll(w http.ResponseWriter, r *http.Request) {
	v, ok := c.mutationBase(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v.Step = platform.FormTrim(r, "step")
	if v.Step == "part" {
		face, err := c.Svc.RollBodyPart()
		if err != nil {
			c.mutationErr(w, v, err)
			return
		}
		v.BodyFace = face
		part, choose := rules.MapBodyDie(face)
		v.BodyPart, v.BodyChoose = part, choose
		c.fillMonsterFromForm(r, &v)
		c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
		return
	}
	cr, sizeFace, typeFace, err := c.Svc.RollMutationFilters(v.Character.Level)
	if err != nil {
		c.mutationErr(w, v, err)
		return
	}
	v.Step = "pick"
	v.CR, v.SizeFace, v.TypeFace = cr, sizeFace, typeFace
	v.Size = rules.MapSizeDie(sizeFace)
	slug, choose := rules.MapTypeDie(typeFace)
	v.Type, v.TypeChoose = slug, choose
	if !choose {
		c.tryPickMonster(&v)
	}
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationPick(w http.ResponseWriter, r *http.Request) {
	v, ok := c.mutationBase(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v.Step = "pick"
	v.CR = rules.ClampMutationCR(platform.FormInt(r, "cr"), v.Character.Level)
	if face := platform.FormInt(r, "size_face"); face > 0 {
		v.SizeFace = face
		if s := rules.MapSizeDie(face); s != "" {
			v.Size = s
		}
	} else {
		v.Size = strings.ToLower(platform.FormTrim(r, "size"))
	}
	if face := platform.FormInt(r, "type_face"); face > 0 {
		v.TypeFace = face
		slug, choose := rules.MapTypeDie(face)
		v.Type, v.TypeChoose = slug, choose
	}
	if picked := strings.ToLower(platform.FormTrim(r, "type")); picked != "" {
		v.Type = picked
		v.TypeChoose = false
	}
	if mid := platform.FormInt64(r, "monster_id"); mid > 0 {
		if m, err := c.Catalog.Monster(mid); err == nil {
			v.Monster = m
		}
	}
	if v.TypeChoose || v.Type == "" {
		v.TypeChoose = true
		c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
		return
	}
	if platform.FormTrim(r, "next") == "part" && v.Monster != nil {
		v.Step = "part"
		c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
		return
	}
	c.tryPickMonster(&v)
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationCreate(w http.ResponseWriter, r *http.Request) {
	v, ok := c.mutationBase(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	c.fillMonsterFromForm(r, &v)
	v.BodyPart = strings.ToLower(platform.FormTrim(r, "body_part"))
	if v.Monster == nil {
		v.Step = "pick"
		c.mutationErr(w, v, ErrNoMonster)
		return
	}
	if !rules.ValidBodyPart(v.BodyPart) {
		v.Step = "part"
		c.mutationErr(w, v, ErrBodyPart)
		return
	}
	m := Mutation{
		BodyPart: v.BodyPart, MonsterID: v.Monster.ID,
		NameEN: v.Monster.NameEN, NameRU: v.Monster.NameRU,
		Size: v.Monster.Size, Type: v.Monster.Type, CR: v.Monster.CR, CRLabel: v.Monster.CRLabel,
		SourceURL: v.Monster.SourceURL, SourceURLRU: v.Monster.SourceURLRU,
	}
	got, err := c.Svc.ApplyMutation(v.Character, platform.UserFrom(r.Context()).ID, m, nil)
	if err != nil {
		v.Step = "part"
		c.mutationErr(w, v, err)
		return
	}
	v.Mutation = &got
	v.Step = "features"
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationsIsland(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_mutations.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationAddFeature(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f := mutationFeatureFromForm(r)
	if err := c.Svc.AddMutationFeature(ch, platform.UserFrom(r.Context()).ID, mid, f); err != nil {
		http.Error(w, ErrorKey(err), http.StatusUnprocessableEntity)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.renderMutationFollowup(w, r, ch)
}

func (c *Controller) mutationRemoveFeature(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fid, err := platform.PathID(r, "fid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.RemoveMutationFeature(ch, platform.UserFrom(r.Context()).ID, mid, fid); err != nil {
		http.Error(w, ErrorKey(err), http.StatusForbidden)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.renderMutationFollowup(w, r, ch)
}

func (c *Controller) mutationRemove(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.RemoveMutation(ch, platform.UserFrom(r.Context()).ID, mid); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.renderMutationFollowup(w, r, ch)
}

func (c *Controller) mutationAttackModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	m, err := c.Svc.ownedMutation(ch, mid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g := rules.GrantsFromFeatures(m.Features)
	if len(g.Attacks) == 0 {
		http.NotFound(w, r)
		return
	}
	at := g.Attacks[0]
	idx := platform.FormInt(r, "i")
	if idx >= 0 && idx < len(g.Attacks) {
		at = g.Attacks[idx]
	}
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.AdjustPool = "mutation"
	v.ClassLine = at.Name(v.Lang) + " · " + at.Dice + " " + at.DamageType
	c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationAttackRoll(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	m, err := c.Svc.ownedMutation(ch, mid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g := rules.GrantsFromFeatures(m.Features)
	if len(g.Attacks) == 0 {
		http.NotFound(w, r)
		return
	}
	at := g.Attacks[0]
	rolled, err := rules.RollFormula(at.Dice)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.AdjustPool = "mutation"
	v.ClassLine = at.Name(v.Lang) + " · " + rolled.Formula + " = " + strconv.Itoa(rolled.Total)
	c.Render.Render(w, "characters/check_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationWizardFeature(w http.ResponseWriter, r *http.Request) {
	v, ok := c.mutationBase(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.fillMonsterFromForm(r, &v)
	f := mutationFeatureFromForm(r)
	if err := c.Svc.AddMutationFeature(v.Character, platform.UserFrom(r.Context()).ID, mid, f); err != nil {
		v.Step = "features"
		c.mutationErr(w, v, err)
		return
	}
	got, err := c.Svc.ownedMutation(v.Character, mid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v.Mutation = &got
	v.Step = "features"
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderMutationFollowup(w http.ResponseWriter, r *http.Request, ch *Character) {
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	v := c.sheetView(r, ch, readonly)
	v.OOBCombat = true
	v.OOBInventory = true
	c.Render.Render(w, "characters/sheet_after_mutation.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) mutationBase(w http.ResponseWriter, r *http.Request) (mutationView, bool) {
	campID, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return mutationView{}, false
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return mutationView{}, false
	}
	ch, err := c.Svc.Get(cid)
	if err != nil || ch.CampaignID != campID {
		http.NotFound(w, r)
		return mutationView{}, false
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.CanView(ch, u.ID); err != nil {
		if _, err := c.Campaigns.RequireMember(campID, u.ID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return mutationView{}, false
		}
	}
	v := mutationView{
		BaseView: c.base(r, ch.Name), Character: ch, CampaignID: campID,
		IsDM: c.Svc.Campaigns.IsDM(campID, u.ID), IsOwner: ch.OwnerID == u.ID,
		PHBTypes: rules.PHBMonsterTypes, Sizes: rules.CreatureSizes, BodyParts: rules.BodyParts,
		AbilityKeys: rules.AbilityKeys, Skills: rules.Skills, DamageTypes: rules.DamageTypes(),
		Slots: rules.WearSlots, Conditions: rules.Conditions(),
	}
	return v, true
}

func (c *Controller) fillMonsterFromForm(r *http.Request, v *mutationView) {
	v.CR = platform.FormInt(r, "cr")
	v.Size = strings.ToLower(platform.FormTrim(r, "size"))
	v.Type = strings.ToLower(platform.FormTrim(r, "type"))
	v.BodyPart = strings.ToLower(platform.FormTrim(r, "body_part"))
	if mid := platform.FormInt64(r, "monster_id"); mid > 0 {
		if m, err := c.Catalog.Monster(mid); err == nil {
			v.Monster = m
		}
	}
}

func (c *Controller) tryPickMonster(v *mutationView) {
	m, cr, err := c.Svc.PickMutationMonster(v.CR, v.Size, v.Type)
	if err != nil {
		v.Error = ErrorKey(err)
		return
	}
	v.Monster = m
	v.CR = cr
}

func (c *Controller) mutationErr(w http.ResponseWriter, v mutationView, err error) {
	v.Error = ErrorKey(err)
	w.Header().Set("HX-Retarget", "#mutation-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "campaigns/mutation_modal.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func mutationFeatureFromForm(r *http.Request) rules.FeatureDTO {
	stat := platform.FormTrim(r, "stat")
	f := rules.FeatureDTO{Stat: stat, Origin: rules.OriginUnique}
	switch stat {
	case rules.StatAbilityBonus, rules.StatAbilityPenalty:
		f.Value = platform.FormTrim(r, "ability") + ":" + platform.FormTrim(r, "amount")
	case rules.StatSkillBonus, rules.StatSkillPenalty:
		f.Value = platform.FormTrim(r, "skill") + ":" + platform.FormTrim(r, "amount")
	case rules.StatGrantWeapon:
		f.Value = platform.FormTrim(r, "name") + "|" + platform.FormTrim(r, "dice") + "|" + platform.FormTrim(r, "damage_type")
	case rules.StatGrantSpell, rules.StatGrantCantrip:
		f.Value = platform.FormTrim(r, "spell_id")
	case rules.StatHands:
		kind := platform.FormTrim(r, "hands")
		if kind == rules.HandsExtraOffhand {
			n := platform.FormInt(r, "amount")
			if n < 1 {
				n = 1
			}
			f.Value = rules.HandsExtraOffhand + ":" + strconv.Itoa(n)
		} else {
			f.Value = rules.HandsTwoHandAndShield
		}
	case rules.StatSize:
		f.Value = platform.FormTrim(r, "size_value")
	case rules.StatBlockSlot:
		f.Value = platform.FormTrim(r, "slot")
	case rules.StatAddPart, rules.StatRemovePart:
		f.Value = platform.FormTrim(r, "part")
	case rules.StatResistance, rules.StatImmunity, rules.StatVulnerability:
		f.Value = platform.FormTrim(r, "damage_type")
	case rules.StatConditionImmunity, rules.StatConditionVulnerability:
		f.Value = platform.FormTrim(r, "condition")
	default:
		f.Value = platform.FormTrim(r, "value")
	}
	return f
}
