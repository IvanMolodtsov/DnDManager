package characters

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type itemSearchView struct {
	platform.BaseView
	Character *Character
	Query     string
	Results   []catalog.Item
	Kind      string
	ListPath  string
}

type featureHit struct {
	Kind string
	ID   int64
	Name string
	Sub  string
	URL  string
}

type itemWizardView struct {
	platform.BaseView
	Character    *Character
	Item         catalog.Item
	Slots        []string
	Suggested    string
	Advanced     bool
	Formula      string
	Roll         *rules.RollResult
	Input        AddItemInput
	ReadOnly     bool
	Versatile    bool
	ConflictName string
	Features     []rules.FeatureDTO
	FeatureQuery string
	FeatureHits  []featureHit
	Derived      rules.DerivedItemStats
	Mode         string
	Kind         string
	ListPath     string
	TableURL     string
	TableKey     string
	PlusNKey     string
	PlusNNote    string
	Spells       []catalog.Spell
	Skills       []rules.Skill
	DamageTypes  []rules.NamedOption
	Conditions   []rules.NamedOption
	Abilities    []string
}

func (c *Controller) renderInventory(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool) {
	v := c.sheetView(r, ch, readonly)
	c.Render.Render(w, "characters/sheet_inventory.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderInventoryAction(w http.ResponseWriter, r *http.Request, ch *Character, readonly bool, vitals bool) {
	v := c.sheetView(r, ch, readonly)
	v.OOBCombat = vitals
	if vitals {
		c.Render.Render(w, "characters/sheet_after_item.html", v.Lang, http.StatusOK, v)
		return
	}
	c.Render.Render(w, "characters/sheet_inventory.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) inventoryErr(w http.ResponseWriter, r *http.Request, ch *Character, err error) {
	v := c.sheetView(r, ch, false)
	v.Error = ErrorKey(err)
	var occ *rules.SlotOccupiedError
	if errors.As(err, &occ) {
		v.ConflictName = occ.OtherName
	}
	c.Render.Render(w, "characters/sheet_inventory.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func (c *Controller) newItemModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "characters/item_type.html", platform.LangFrom(r.Context()), http.StatusOK, itemSearchView{
		BaseView: c.base(r, ch.Name), Character: ch,
	})
}

func (c *Controller) weaponBases(w http.ResponseWriter, r *http.Request) {
	c.renderBases(w, r, "weapon")
}

func (c *Controller) armorBases(w http.ResponseWriter, r *http.Request) {
	c.renderBases(w, r, "armor")
}

func (c *Controller) jewelryBases(w http.ResponseWriter, r *http.Request) {
	c.renderBases(w, r, "jewelry")
}

func (c *Controller) renderBases(w http.ResponseWriter, r *http.Request, kind string) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	results, _ := c.Catalog.SearchBases(kind, "", 40)
	c.Render.Render(w, "characters/item_bases.html", platform.LangFrom(r.Context()), http.StatusOK, itemSearchView{
		BaseView: c.base(r, ch.Name), Character: ch, Kind: kind, Results: results, ListPath: baseListPath(kind),
	})
}

func (c *Controller) searchBaseWeapons(w http.ResponseWriter, r *http.Request) {
	c.searchBases(w, r, "weapon")
}

func (c *Controller) searchBaseArmor(w http.ResponseWriter, r *http.Request) {
	c.searchBases(w, r, "armor")
}

func (c *Controller) searchBaseJewelry(w http.ResponseWriter, r *http.Request) {
	c.searchBases(w, r, "jewelry")
}

func (c *Controller) searchBases(w http.ResponseWriter, r *http.Request, kind string) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		q = platform.FormTrim(r, "q")
	}
	results, _ := c.Catalog.SearchBases(kind, q, 40)
	c.Render.Render(w, "characters/item_base_results.html", platform.LangFrom(r.Context()), http.StatusOK, itemSearchView{
		BaseView: c.base(r, ch.Name), Character: ch, Query: q, Results: results, Kind: kind, ListPath: baseListPath(kind),
	})
}

func baseListPath(kind string) string {
	switch kind {
	case "armor":
		return "armor"
	case "jewelry":
		return "jewelry"
	default:
		return "weapons"
	}
}

func (c *Controller) consumableSearch(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "characters/item_search.html", platform.LangFrom(r.Context()), http.StatusOK, itemSearchView{
		BaseView: c.base(r, ch.Name), Character: ch, Kind: "consumable",
	})
}

func (c *Controller) searchItems(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		q = platform.FormTrim(r, "q")
	}
	var results []catalog.Item
	if len(q) >= 1 {
		results, _ = c.Catalog.SearchConsumables(q, 20)
	}
	c.Render.Render(w, "characters/item_search_results.html", platform.LangFrom(r.Context()), http.StatusOK, itemSearchView{
		BaseView: c.base(r, ch.Name), Character: ch, Query: q, Results: results, Kind: "consumable",
	})
}

func (c *Controller) previewItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "characters/item_preview.html", platform.LangFrom(r.Context()), http.StatusOK, itemWizardView{
		BaseView: c.base(r, item.Name(platform.LangFrom(r.Context()))), Character: ch, Item: *item, Mode: "consumable", Kind: "consumable",
	})
}

func (c *Controller) rarityItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "characters/item_rarity.html", platform.LangFrom(r.Context()), http.StatusOK, c.wizardView(r, ch, *item, AddItemInput{}, nil, "rarity"))
}

func (c *Controller) configureItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	mode := wizardMode(r, *item)
	if mode == "rarity" {
		c.Render.Render(w, "characters/item_rarity.html", platform.LangFrom(r.Context()), http.StatusOK, c.wizardView(r, ch, *item, AddItemInput{}, nil, mode))
		return
	}
	in := addInputFromForm(r, item)
	in.Features = c.featuresForWizard(r, *item, mode)
	c.Render.Render(w, "characters/item_configure.html", platform.LangFrom(r.Context()), http.StatusOK, c.wizardView(r, ch, *item, in, nil, mode))
}

func (c *Controller) mutateItemFeatures(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	mode := wizardMode(r, *item)
	in := addInputFromForm(r, item)
	in.Features = applyFeatureMutations(r, c.featuresForWizard(r, *item, mode), c.Svc)
	c.Render.Render(w, "characters/item_configure.html", platform.LangFrom(r.Context()), http.StatusOK, c.wizardView(r, ch, *item, in, nil, mode))
}

func (c *Controller) searchItemFeatures(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		q = platform.FormTrim(r, "q")
	}
	lang := platform.LangFrom(r.Context())
	c.Render.Render(w, "characters/feature_search_results.html", lang, http.StatusOK, itemWizardView{
		BaseView: c.base(r, ch.Name), Character: ch, Item: *item, FeatureQuery: q, FeatureHits: c.featureHits(q, lang), Mode: "magic",
	})
}

func (c *Controller) rollCatalogItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	item, ok := c.loadCatalogItem(w, r)
	if !ok {
		return
	}
	mode := wizardMode(r, *item)
	in := addInputFromForm(r, item)
	in.Features = c.featuresForWizard(r, *item, mode)
	formula := platform.FormTrim(r, "formula")
	if formula == "" {
		formula = rules.DeriveItemStats(item.DamageDice, item.DamageType, in.Features).RollFormula
	}
	if formula == "" {
		formula = item.DamageDice
	}
	res, err := rules.RollFormula(formula)
	v := c.wizardView(r, ch, *item, in, nil, mode)
	if err != nil {
		v.Error = ErrorKey(err)
	} else {
		v.Roll = &res
		note := fmt.Sprintf("%s = %d", res.Formula, res.Total)
		if in.Notes != "" {
			in.Notes += "\n"
		}
		in.Notes += note
		v.Input = in
	}
	v.Formula = formula
	c.Render.Render(w, "characters/item_configure.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) addItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	itemID := platform.FormInt64(r, "item_id")
	item, err := c.Catalog.Item(itemID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	mode := wizardMode(r, *item)
	in := addInputFromForm(r, item)
	in.CatalogID = item.ID
	in.Features = c.featuresForWizard(r, *item, mode)
	if mode == "common" {
		in.Features = nil
	}
	if _, err := c.Svc.AddItem(ch, platform.UserFrom(r.Context()).ID, in); err != nil {
		v := c.wizardView(r, ch, *item, in, nil, mode)
		v.Error = ErrorKey(err)
		var occ *rules.SlotOccupiedError
		if errors.As(err, &occ) {
			v.ConflictName = occ.OtherName
		}
		w.Header().Set("HX-Retarget", "#item-add-body")
		w.Header().Set("HX-Reswap", "innerHTML")
		c.Render.Render(w, "characters/item_configure.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	w.Header().Set("HX-Trigger", "close-item-modal")
	c.renderInventoryAction(w, r, ch, false, in.EquipNow)
}

func (c *Controller) equipItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	slot := platform.FormTrim(r, "slot")
	if err := c.Svc.EquipItem(ch, platform.UserFrom(r.Context()).ID, invID, slot); err != nil {
		c.inventoryErr(w, r, ch, err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderInventoryAction(w, r, ch, false, true)
}

func (c *Controller) unequipItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.UnequipItem(ch, platform.UserFrom(r.Context()).ID, invID); err != nil {
		c.inventoryErr(w, r, ch, err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderInventoryAction(w, r, ch, false, true)
}

func (c *Controller) qtyItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	delta := platform.FormInt(r, "delta")
	if err := c.Svc.AdjustItemQty(ch, platform.UserFrom(r.Context()).ID, invID, delta); err != nil {
		c.inventoryErr(w, r, ch, err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderInventory(w, r, ch, false)
}

func (c *Controller) useItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.UseItem(ch, platform.UserFrom(r.Context()).ID, invID); err != nil {
		c.inventoryErr(w, r, ch, err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderInventoryAction(w, r, ch, false, true)
}

func (c *Controller) removeItem(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := c.Svc.RemoveItem(ch, platform.UserFrom(r.Context()).ID, invID); err != nil {
		c.inventoryErr(w, r, ch, err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	c.renderInventory(w, r, ch, false)
}

func (c *Controller) loadCatalogItem(w http.ResponseWriter, r *http.Request) (*catalog.Item, bool) {
	id, err := platform.PathID(r, "itemID")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	item, err := c.Catalog.Item(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	return item, true
}

func (c *Controller) wizardView(r *http.Request, ch *Character, item catalog.Item, in AddItemInput, roll *rules.RollResult, mode string) itemWizardView {
	feats := in.Features
	if mode == "common" {
		feats = append(append([]rules.FeatureDTO{}, in.Features...), rules.CommonPlusFor(item, in.PlusN)...)
	}
	derived := rules.DeriveItemStats(item.DamageDice, item.DamageType, feats)
	formula := platform.FormTrim(r, "formula")
	if formula == "" {
		formula = derived.RollFormula
	}
	if formula == "" {
		formula = item.DamageDice
	}
	slots := append([]string{""}, rules.WearSlots...)
	if item.SuggestedSlot == "ring" {
		slots = append([]string{"", "ring"}, rules.WearSlots...)
	}
	versatile := item.HasProperty("versatile") || rules.HasProperty(in.Features, "versatile")
	kind := item.BuilderKind()
	tableURL, tableKey := "", ""
	plusKey, plusNote := "character.item.plus_n", "character.item.plus_n.note"
	switch kind {
	case "armor":
		tableURL, tableKey = catalog.ArmorTableURL, "character.item.open_armor"
		plusKey, plusNote = "character.item.plus_n.ac", "character.item.plus_n.ac.note"
	case "jewelry":
		plusKey, plusNote = "character.item.plus_n.ac", "character.item.plus_n.ac.note"
	default:
		tableURL, tableKey = catalog.ArmsTableURL, "character.item.open_arms"
	}
	var spells []catalog.Spell
	if mode == "magic" {
		spells, _ = c.Catalog.ListSpells()
	}
	return itemWizardView{
		BaseView:  c.base(r, item.Name(platform.LangFrom(r.Context()))),
		Character: ch, Item: item, Slots: slots, Suggested: item.SuggestedSlot,
		Advanced: item.IsStub || item.IsMagic(), Formula: formula, Roll: roll, Input: in,
		Versatile: versatile, Features: in.Features, Derived: derived,
		Mode: mode, Kind: kind, ListPath: item.BuilderListPath(),
		TableURL: tableURL, TableKey: tableKey, PlusNKey: plusKey, PlusNNote: plusNote,
		Spells: spells, Skills: rules.Skills, DamageTypes: rules.DamageTypes(),
		Conditions: rules.Conditions(), Abilities: rules.AbilityKeys,
	}
}

func wizardMode(r *http.Request, item catalog.Item) string {
	mode := platform.FormTrim(r, "rarity")
	if mode == "" {
		mode = strings.TrimSpace(r.URL.Query().Get("rarity"))
	}
	if item.Consumable {
		return "consumable"
	}
	switch mode {
	case "common", "magic", "rarity":
		return mode
	}
	if item.IsBase {
		return "rarity"
	}
	return "consumable"
}

func (c *Controller) featuresForWizard(r *http.Request, item catalog.Item, mode string) []rules.FeatureDTO {
	if mode == "common" || mode == "rarity" {
		return nil
	}
	feats := parseFeatures(r)
	if platform.FormTrim(r, "feat_loaded") == "1" {
		return feats
	}
	if len(feats) > 0 {
		return feats
	}
	if mode == "magic" {
		return c.Svc.catalogFeatures(item)
	}
	return nil
}

func (c *Controller) featureHits(q, lang string) []featureHit {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	var out []featureHit
	if stats, err := c.Catalog.SearchStatFeatures(q, 8); err == nil {
		for _, sf := range stats {
			out = append(out, featureHit{
				Kind: "stat", ID: sf.ID, Name: sf.Name(lang), Sub: sf.Stat, URL: sf.SourceURL,
			})
		}
	}
	if items, err := c.Catalog.SearchItems(q, 8); err == nil {
		for _, it := range items {
			if it.IsBase {
				continue
			}
			out = append(out, featureHit{
				Kind: "item", ID: it.ID, Name: it.Name(lang), Sub: it.StatsLine(), URL: it.Source5e14(),
			})
		}
	}
	if spells, err := c.Catalog.SearchSpells(q, 8); err == nil {
		for _, sp := range spells {
			url := sp.SourceURLRU
			if url == "" {
				url = sp.SourceURL
			}
			out = append(out, featureHit{
				Kind: "spell", ID: sp.ID, Name: sp.Name(lang), Sub: sp.StatsLine(sp.Level, 1), URL: url,
			})
		}
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

func applyFeatureMutations(r *http.Request, feats []rules.FeatureDTO, svc *Service) []rules.FeatureDTO {
	if rm := platform.FormTrim(r, "feat_remove"); rm != "" {
		idx, err := strconv.Atoi(rm)
		if err == nil && idx >= 0 && idx < len(feats) {
			feats = append(feats[:idx], feats[idx+1:]...)
		}
	}
	if platform.FormTrim(r, "feat_add_blank") == "1" {
		feats = append(feats, rules.FeatureDTO{Origin: rules.OriginUnique, NameEN: "Note", NameRU: "Заметка", Stat: rules.StatNote})
	}
	if id := platform.FormInt64(r, "feat_add_stat"); id > 0 {
		if sf, err := svc.Catalog.StatFeature(id); err == nil {
			feats = append(feats, rules.FeatureFromStat(*sf))
		}
	}
	if id := platform.FormInt64(r, "feat_add_item"); id > 0 {
		if it, err := svc.Catalog.Item(id); err == nil {
			feats = append(feats, rules.FeaturesFromItem(*it)...)
		}
	}
	if id := platform.FormInt64(r, "feat_add_spell"); id > 0 {
		if sp, err := svc.Catalog.Spell(id); err == nil {
			feats = append(feats, rules.FeatureFromSpell(*sp))
		}
	}
	if platform.FormTrim(r, "feat_add_url") == "1" {
		if url := platform.FormTrim(r, "feature_url"); url != "" {
			feats = append(feats, svc.FeaturesFromRef(url)...)
		}
	}
	return feats
}

var featKeyRe = regexp.MustCompile(`^feat\.(\d+)\.`)

func parseFeatures(r *http.Request) []rules.FeatureDTO {
	if err := r.ParseForm(); err != nil {
		return nil
	}
	max := -1
	for k := range r.Form {
		m := featKeyRe.FindStringSubmatch(k)
		if len(m) != 2 {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n > max {
			max = n
		}
	}
	if max < 0 {
		return nil
	}
	out := make([]rules.FeatureDTO, 0, max+1)
	for i := 0; i <= max; i++ {
		p := fmt.Sprintf("feat.%d.", i)
		out = append(out, rules.FeatureDTO{
			NameEN:      strings.TrimSpace(r.FormValue(p + "name_en")),
			NameRU:      strings.TrimSpace(r.FormValue(p + "name_ru")),
			Stat:        r.FormValue(p + "stat"),
			Value:       strings.TrimSpace(r.FormValue(p + "value")),
			Origin:      r.FormValue(p + "origin"),
			CatalogKind: r.FormValue(p + "catalog_kind"),
			CatalogID:   platform.FormInt64(r, p+"catalog_id"),
			CatalogSlug: r.FormValue(p + "catalog_slug"),
			SourceURL:   strings.TrimSpace(r.FormValue(p + "source_url")),
		})
		if ab := strings.TrimSpace(r.FormValue(p + "ability")); ab != "" {
			delta := strings.TrimSpace(r.FormValue(p + "delta"))
			if delta == "" {
				delta = "0"
			}
			out[len(out)-1].Value = strings.ToLower(ab) + ":" + delta
		}
	}
	return out
}

func addInputFromForm(r *http.Request, item *catalog.Item) AddItemInput {
	qty := platform.FormInt(r, "quantity")
	if qty < 1 {
		qty = 1
	}
	slot := platform.FormTrim(r, "slot")
	if slot == "" {
		slot = item.SuggestedSlot
	}
	in := AddItemInput{
		CatalogID:      item.ID,
		Quantity:       qty,
		EquipSlot:      slot,
		EquipNow:       platform.FormTrim(r, "equip_now") == "1",
		Attune:         platform.FormTrim(r, "attune") == "1",
		ChargesCurrent: platform.FormInt(r, "charges_current"),
		ChargesMax:     platform.FormInt(r, "charges_max"),
		CustomName:     platform.FormTrim(r, "custom_name"),
		Notes:          platform.FormTrim(r, "notes"),
		TwoHanded:      platform.FormTrim(r, "two_handed") == "1",
		PlusN:          platform.FormInt(r, "plus_n"),
	}
	if in.ChargesMax == 0 {
		in.ChargesMax = item.ChargesMax
	}
	return in
}
