package characters

import (
	"net/http"
	"strings"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

// Controller serves character list, sheet, creation wizard, and level-up.
type Controller struct {
	Svc       *Service
	Campaigns *campaigns.Service
	Catalog   *catalog.Service
	Render    *platform.Renderer
}

// Mount registers authenticated character routes, including HTMX partials.
func (c *Controller) Mount(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /characters", auth(http.HandlerFunc(c.list)))
	mux.Handle("GET /campaigns/{id}/characters/new", auth(http.HandlerFunc(c.showWizard)))
	mux.Handle("POST /campaigns/{id}/characters/wizard", auth(http.HandlerFunc(c.postWizard)))
	mux.Handle("GET /campaigns/{id}/characters/wizard/subclass", auth(http.HandlerFunc(c.subclassPartial)))
	mux.Handle("GET /campaigns/{id}/characters/wizard/scores", auth(http.HandlerFunc(c.scoresPartial)))
	mux.Handle("GET /characters/{id}", auth(http.HandlerFunc(c.show)))
	mux.Handle("GET /characters/{id}/level-up", auth(http.HandlerFunc(c.showLevelUp)))
	mux.Handle("POST /characters/{id}/level-up/preview", auth(http.HandlerFunc(c.previewLevelUp)))
	mux.Handle("POST /characters/{id}/level-up/preview-panel", auth(http.HandlerFunc(c.previewLevelUpPanel)))
	mux.Handle("POST /characters/{id}/level-up", auth(http.HandlerFunc(c.postLevelUp)))
	mux.Handle("GET /characters/{id}/spells/search", auth(http.HandlerFunc(c.searchSpells)))
	mux.Handle("POST /characters/{id}/spells", auth(http.HandlerFunc(c.addSpell)))
	mux.Handle("POST /characters/{id}/spells/{spellID}/prepared", auth(http.HandlerFunc(c.togglePrepared)))
	mux.Handle("POST /characters/{id}/spells/{spellID}/remove", auth(http.HandlerFunc(c.removeSpell)))
	mux.Handle("GET /characters/{id}/spells/{spellID}/use", auth(http.HandlerFunc(c.useSpellModal)))
	mux.Handle("POST /characters/{id}/spells/{spellID}/roll", auth(http.HandlerFunc(c.rollSpell)))
	mux.Handle("POST /characters/{id}/spells/{spellID}/cast", auth(http.HandlerFunc(c.castSpell)))
	mux.Handle("POST /characters/{id}/resources/use", auth(http.HandlerFunc(c.spendResource)))
	mux.Handle("POST /characters/{id}/skills/{slug}", auth(http.HandlerFunc(c.toggleSkill)))
	mux.Handle("POST /characters/{id}/saves/{ability}", auth(http.HandlerFunc(c.toggleSave)))
	mux.Handle("GET /characters/{id}/checks/skill/{slug}", auth(http.HandlerFunc(c.skillCheckModal)))
	mux.Handle("GET /characters/{id}/checks/save/{ability}", auth(http.HandlerFunc(c.saveCheckModal)))
	mux.Handle("POST /characters/{id}/checks/skill/{slug}/roll", auth(http.HandlerFunc(c.rollSkillCheck)))
	mux.Handle("POST /characters/{id}/checks/save/{ability}/roll", auth(http.HandlerFunc(c.rollSaveCheck)))
	mux.Handle("GET /characters/{id}/vitals", auth(http.HandlerFunc(c.vitalsPartial)))
	mux.Handle("GET /characters/{id}/events", auth(http.HandlerFunc(c.vitalsEvents)))
	mux.Handle("GET /characters/{id}/combat/adjust", auth(http.HandlerFunc(c.hpAdjustModal)))
	mux.Handle("POST /characters/{id}/combat/adjust", auth(http.HandlerFunc(c.applyHPAdjust)))
	mux.Handle("POST /characters/{id}/combat/death", auth(http.HandlerFunc(c.toggleDeath)))
	mux.Handle("POST /characters/{id}/effects/{effectID}/remove", auth(http.HandlerFunc(c.dismissEffect)))
	mux.Handle("GET /characters/{id}/effects/new", auth(http.HandlerFunc(c.statusAddModal)))
	mux.Handle("GET /characters/{id}/effects/search", auth(http.HandlerFunc(c.searchConditions)))
	mux.Handle("POST /characters/{id}/effects", auth(http.HandlerFunc(c.addStatus)))
	mux.Handle("GET /characters/{id}/items/new", auth(http.HandlerFunc(c.newItemModal)))
	mux.Handle("GET /characters/{id}/items/weapons", auth(http.HandlerFunc(c.weaponBases)))
	mux.Handle("GET /characters/{id}/items/weapons/search", auth(http.HandlerFunc(c.searchBaseWeapons)))
	mux.Handle("GET /characters/{id}/items/armor", auth(http.HandlerFunc(c.armorBases)))
	mux.Handle("GET /characters/{id}/items/armor/search", auth(http.HandlerFunc(c.searchBaseArmor)))
	mux.Handle("GET /characters/{id}/items/jewelry", auth(http.HandlerFunc(c.jewelryBases)))
	mux.Handle("GET /characters/{id}/items/jewelry/search", auth(http.HandlerFunc(c.searchBaseJewelry)))
	mux.Handle("GET /characters/{id}/items/consumables", auth(http.HandlerFunc(c.consumableSearch)))
	mux.Handle("GET /characters/{id}/items/search", auth(http.HandlerFunc(c.searchItems)))
	mux.Handle("GET /characters/{id}/items/{itemID}/preview", auth(http.HandlerFunc(c.previewItem)))
	mux.Handle("GET /characters/{id}/items/{itemID}/rarity", auth(http.HandlerFunc(c.rarityItem)))
	mux.Handle("GET /characters/{id}/items/{itemID}/configure", auth(http.HandlerFunc(c.configureItem)))
	mux.Handle("POST /characters/{id}/items/{itemID}/configure", auth(http.HandlerFunc(c.mutateItemFeatures)))
	mux.Handle("GET /characters/{id}/items/{itemID}/features/search", auth(http.HandlerFunc(c.searchItemFeatures)))
	mux.Handle("POST /characters/{id}/items/{itemID}/roll", auth(http.HandlerFunc(c.rollCatalogItem)))
	mux.Handle("POST /characters/{id}/items", auth(http.HandlerFunc(c.addItem)))
	mux.Handle("GET /characters/{id}/items/{invID}/attack", auth(http.HandlerFunc(c.weaponAttackModal)))
	mux.Handle("POST /characters/{id}/items/{invID}/attack/roll", auth(http.HandlerFunc(c.rollWeaponAttack)))
	mux.Handle("POST /characters/{id}/items/{invID}/equip", auth(http.HandlerFunc(c.equipItem)))
	mux.Handle("POST /characters/{id}/items/{invID}/unequip", auth(http.HandlerFunc(c.unequipItem)))
	mux.Handle("POST /characters/{id}/items/{invID}/qty", auth(http.HandlerFunc(c.qtyItem)))
	mux.Handle("POST /characters/{id}/items/{invID}/use", auth(http.HandlerFunc(c.useItem)))
	mux.Handle("POST /characters/{id}/items/{invID}/remove", auth(http.HandlerFunc(c.removeItem)))
	mux.Handle("GET /characters/{id}/companions", auth(http.HandlerFunc(c.companionsIsland)))
	mux.Handle("GET /characters/{id}/companions/search", auth(http.HandlerFunc(c.searchCompanions)))
	mux.Handle("POST /characters/{id}/companions", auth(http.HandlerFunc(c.addCompanion)))
	mux.Handle("GET /characters/{id}/companions/{cid}/edit", auth(http.HandlerFunc(c.editCompanion)))
	mux.Handle("POST /characters/{id}/companions/{cid}/edit", auth(http.HandlerFunc(c.saveCompanion)))
	mux.Handle("POST /characters/{id}/companions/{cid}/remove", auth(http.HandlerFunc(c.removeCompanion)))
	mux.Handle("GET /characters/{id}/inventory", auth(http.HandlerFunc(c.inventoryPartial)))
	mux.Handle("GET /characters/{id}/gold/adjust", auth(http.HandlerFunc(c.goldAdjustModal)))
	mux.Handle("POST /characters/{id}/gold", auth(http.HandlerFunc(c.applyGold)))
	mux.Handle("GET /characters/{id}/souls/adjust", auth(http.HandlerFunc(c.soulsAdjustModal)))
	mux.Handle("GET /characters/{id}/souls/cap", auth(http.HandlerFunc(c.soulsCapModal)))
	mux.Handle("POST /characters/{id}/souls", auth(http.HandlerFunc(c.applySouls)))
	c.MountMutations(mux, auth)
}

func (c *Controller) base(r *http.Request, title string) platform.BaseView {
	return platform.NewBase(r, title)
}

type listView struct {
	platform.BaseView
	Characters []Character
}

func (c *Controller) list(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	chars, err := c.Svc.ListByOwner(u.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	lang := platform.LangFrom(r.Context())
	for i := range chars {
		c.Svc.Localize(&chars[i], lang)
	}
	v := listView{BaseView: c.base(r, "nav.characters"), Characters: chars}
	c.Render.Render(w, "characters/list.html", v.Lang, http.StatusOK, v)
}

type showView struct {
	platform.BaseView
	Character        *Character
	ReadOnly         bool
	IsOwner          bool
	IsDM             bool
	ClassLine        string
	ResourceGroups   []ResourceGroup
	Prepared         []SpellRow
	Learned          []SpellRow
	Skills           []SkillRow
	Saves            []SaveRow
	Combat           rules.CombatStats
	OOBCombat        bool
	OOBInventory     bool
	WearSlots        []string
	CreatureSizes    []string
	BodyParts        []string
	Conditions       []rules.NamedOption
	Slots            []string
	Sizes            []string
	HPResult         *HPResult
	AdjustPool       string
	AdjustSign       string
	AdjustAmount     int
	AdjustDamageType string
	DamageTypes      []rules.NamedOption
	Tab              string
	Equipped         []ItemRow
	Pack             []ItemRow
	Consumables      []ItemRow
	ConflictName     string
	DefenseLine      string
	StatusQuery      string
	StatusResults    []catalog.Condition
	StatusPick       *catalog.Condition
	DefaultRemoveEnd bool
	StatusNewURL     string
	StatusSearchURL  string
	StatusPostURL    string
	CompanionGate    CompanionGate
	CompanionKind    string
	CompanionQuery   string
	CompanionCR      string
	CompanionType    string
	CompanionHits    []catalog.Monster
	CompanionCRs     []string
	CompanionTypes   []catalog.MonsterTypeOption
	EditCompanion    *Companion
	AbilityKeys      []string
}

func (c *Controller) show(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ch, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, err := c.Svc.CanView(ch, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/show.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) sheetView(r *http.Request, ch *Character, readonly bool) showView {
	tab := r.URL.Query().Get("tab")
	if tab != "learned" {
		tab = "prepared"
	}
	u := platform.UserFrom(r.Context())
	isOwner := ch.OwnerID == u.ID
	v := showView{
		BaseView:       c.base(r, ch.Name),
		Character:      ch,
		ReadOnly:       !isOwner,
		IsOwner:        isOwner,
		IsDM:           c.Svc.Campaigns.IsDM(ch.CampaignID, u.ID),
		ClassLine:      ch.ClassLine(),
		ResourceGroups: resourceGroups(ch.Resources),
		Prepared:       spellRows(ch, true, platform.LangFrom(r.Context())),
		Learned:        spellRows(ch, false, platform.LangFrom(r.Context())),
		Skills:         skillRows(ch, platform.LangFrom(r.Context())),
		Saves:          saveRows(ch),
		Combat:         sheetCombat(ch, c.Svc.Campaigns.IsDM(ch.CampaignID, u.ID)),
		DefenseLine:    ch.AllGrants().DefenseLine(),
		Tab:            tab,
		Equipped:       equippedRows(ch),
		Pack:           packRows(ch),
		Consumables:    consumableRows(ch),
		CompanionGate:  GateCompanions(ch),
		AbilityKeys:    rules.AbilityKeys,
		WearSlots:      rules.WearSlots,
		CreatureSizes:  rules.CreatureSizes,
		Slots:          rules.WearSlots,
		Sizes:          rules.CreatureSizes,
		BodyParts:      rules.BodyParts,
		Conditions:     rules.Conditions(),
		DamageTypes:    rules.DamageTypes(),
	}
	c.fillCompanionFilters(&v)
	return v
}

func sheetCombat(ch *Character, isDM bool) rules.CombatStats {
	st := rules.DeriveCombat(ch.CombatInput())
	st.Effects = rules.VisibleEffects(ch.Effects, isDM)
	return st
}

func (c *Controller) memberCampaign(w http.ResponseWriter, r *http.Request) (*campaigns.Campaign, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	camp, err := c.Campaigns.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	if _, err := c.Campaigns.RequireMember(id, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return camp, true
}

func (c *Controller) loadLive(w http.ResponseWriter, r *http.Request) (*Character, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	ch, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.CanView(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	return ch, true
}

func parseASI(r *http.Request) rules.AbilityScores {
	var a rules.AbilityScores
	for _, field := range []string{"asi_a", "asi_b"} {
		key := strings.ToLower(platform.FormTrim(r, field))
		if rules.ValidAbilityKey(key) {
			a.AddKey(key, 1)
		}
	}
	return a
}
