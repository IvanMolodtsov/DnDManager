package characters

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

var (
	ErrNameRequired = errors.New("character name required")
	ErrNotFound     = errors.New("character not found")
	ErrForbidden    = errors.New("forbidden")
	ErrNotOwner     = errors.New("not character owner")
)

// Service hydrates sheets, persists drafts, and applies confirmed progression.
type Service struct {
	Repo      *Repository
	Campaigns *campaigns.Service
	Catalog   *catalog.Service
	Rules     *rules.Engine
	Events    *VitalsHub
}

func (s *Service) Get(id int64) (*Character, error) {
	ch, err := s.Repo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.hydrate(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Service) ListByOwner(ownerID int64) ([]Character, error) {
	list, err := s.Repo.ListByOwner(ownerID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		_ = s.hydrate(&list[i])
	}
	return list, nil
}

func (s *Service) ListByCampaign(campaignID int64) ([]campaigns.CharacterSummary, error) {
	list, err := s.Repo.ListByCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]campaigns.CharacterSummary, 0, len(list))
	for _, ch := range list {
		_ = s.hydrate(&ch)
		out = append(out, campaigns.CharacterSummary{
			ID:      ch.ID,
			Name:    ch.Name,
			Level:   ch.Level,
			Owner:   ch.OwnerName,
			OwnerID: ch.OwnerID,
		})
	}
	return out, nil
}

// CanView allows the owner (editable) or the campaign DM (combat-only mutations).
func (s *Service) CanView(ch *Character, userID int64) (readonly bool, err error) {
	if ch.OwnerID == userID {
		return false, nil
	}
	if s.Campaigns.IsDM(ch.CampaignID, userID) {
		return true, nil
	}
	return false, ErrForbidden
}

func (s *Service) RequireOwner(ch *Character, userID int64) error {
	if ch.OwnerID != userID {
		return ErrNotOwner
	}
	return nil
}

// RequireCombatEdit allows the owner or the campaign DM to change vitals and statuses.
func (s *Service) RequireCombatEdit(ch *Character, userID int64) error {
	if ch.OwnerID == userID {
		return nil
	}
	if s.Campaigns.IsDM(ch.CampaignID, userID) {
		return nil
	}
	return ErrForbidden
}

func (s *Service) EnsureDraft(ownerID, campaignID int64) (*Draft, error) {
	if _, err := s.Campaigns.RequireMember(campaignID, ownerID); err != nil {
		return nil, err
	}
	d, err := s.Repo.GetDraft(ownerID, campaignID)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	d = &Draft{OwnerID: ownerID, CampaignID: campaignID, Step: 1, BaseScores: defaultArray()}
	id, err := s.Repo.InsertDraft(d)
	if err != nil {
		return nil, err
	}
	d.ID = id
	return d, nil
}

func (s *Service) SaveDraft(d *Draft) error {
	if d.Step < 1 {
		d.Step = 1
	}
	if d.Step > 4 {
		d.Step = 4
	}
	return s.Repo.UpdateDraft(d)
}

func (s *Service) ResetDraft(d *Draft) error {
	*d = Draft{ID: d.ID, OwnerID: d.OwnerID, CampaignID: d.CampaignID, Step: 1, BaseScores: defaultArray()}
	return s.Repo.UpdateDraft(d)
}

// ConfirmDraft runs IntentCreate and inserts the live character.
func (s *Service) ConfirmDraft(d *Draft, ownerID int64, lang string) (*Character, error) {
	if d.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(d.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return nil, ErrNameRequired
	}
	if _, err := s.Campaigns.RequireMember(d.CampaignID, ownerID); err != nil {
		return nil, err
	}
	delta, err := s.Rules.Preview(rules.State{}, rules.Intent{
		Kind:         rules.IntentCreate,
		RaceID:       d.RaceID,
		BackgroundID: d.BackgroundID,
		ClassID:      d.ClassID,
		SubclassID:   d.SubclassID,
		BaseScores:   d.BaseScores,
		Lang:         lang,
	})
	if err != nil {
		return nil, err
	}
	ch := &Character{
		Name:             name,
		OwnerID:          ownerID,
		CampaignID:       d.CampaignID,
		Level:            delta.LevelAfter,
		STR:              delta.ScoresAfter.STR,
		DEX:              delta.ScoresAfter.DEX,
		CON:              delta.ScoresAfter.CON,
		INT:              delta.ScoresAfter.INT,
		WIS:              delta.ScoresAfter.WIS,
		CHA:              delta.ScoresAfter.CHA,
		RaceID:           d.RaceID,
		BackgroundID:     d.BackgroundID,
		HPMax:            delta.HPAfter,
		HPCurrent:        delta.HPAfter,
		ProficiencyBonus: delta.ProficiencyAfter,
	}
	ids := featureIDs(delta)
	id, err := s.Repo.InsertLive(ch, delta.ClassLevels, ids)
	if err != nil {
		return nil, err
	}
	_ = s.Repo.DeleteDraft(d.ID)
	return s.Get(id)
}

func (s *Service) PreviewCreate(d *Draft, lang string) (*rules.Delta, error) {
	return s.Rules.Preview(rules.State{}, rules.Intent{
		Kind:         rules.IntentCreate,
		RaceID:       d.RaceID,
		BackgroundID: d.BackgroundID,
		ClassID:      d.ClassID,
		SubclassID:   d.SubclassID,
		BaseScores:   d.BaseScores,
		Lang:         lang,
	})
}

func (s *Service) PreviewLevelUp(ch *Character, in rules.Intent) (*rules.Delta, error) {
	in.Kind = rules.IntentLevelUp
	return s.Rules.Preview(ch.State(), in)
}

// ApplyLevelUp previews IntentLevelUp and writes the new scores, HP, and features.
func (s *Service) ApplyLevelUp(ch *Character, ownerID int64, in rules.Intent) (*Character, error) {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return nil, err
	}
	in.Kind = rules.IntentLevelUp
	delta, err := s.Rules.Preview(ch.State(), in)
	if err != nil {
		return nil, err
	}
	hpCurrent := ch.HPCurrent + delta.HPGain
	if hpCurrent > delta.HPAfter {
		hpCurrent = delta.HPAfter
	}
	next := &Character{
		Level:            delta.LevelAfter,
		STR:              delta.ScoresAfter.STR,
		DEX:              delta.ScoresAfter.DEX,
		CON:              delta.ScoresAfter.CON,
		INT:              delta.ScoresAfter.INT,
		WIS:              delta.ScoresAfter.WIS,
		CHA:              delta.ScoresAfter.CHA,
		HPMax:            delta.HPAfter,
		HPCurrent:        hpCurrent,
		ProficiencyBonus: delta.ProficiencyAfter,
	}
	if err := s.Repo.ApplyProgress(ch.ID, next, delta.ClassLevels, featureIDs(delta)); err != nil {
		return nil, err
	}
	return s.Get(ch.ID)
}

func (s *Service) CreationBonuses(d *Draft) rules.AbilityScores {
	var b rules.AbilityScores
	if d.RaceID != 0 {
		if race, err := s.Catalog.Race(d.RaceID); err == nil {
			b = b.Add(rules.FromMap(race.AbilityBonuses))
		}
	}
	if d.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(d.BackgroundID); err == nil {
			b = b.Add(rules.FromMap(bg.AbilityBonuses))
		}
	}
	if d.ClassID != 0 {
		if class, err := s.Catalog.Class(d.ClassID); err == nil {
			b = b.Add(rules.FromMap(class.AbilityBonuses))
		}
	}
	return b
}

func (s *Service) BonusLines(d *Draft, lang string) []BonusLine {
	var lines []BonusLine
	if d.RaceID != 0 {
		if race, err := s.Catalog.Race(d.RaceID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Race", "Раса"),
				Text:  formatBonusMap(race.Name(lang), race.AbilityBonuses),
				URL:   race.Source(lang),
			})
		}
	}
	if d.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(d.BackgroundID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Background", "Предыстория"),
				Text:  formatBonusMap(bg.Name(lang), bg.AbilityBonuses),
				URL:   bg.Source(lang),
			})
		}
	}
	if d.ClassID != 0 {
		if class, err := s.Catalog.Class(d.ClassID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Class", "Класс"),
				Text:  formatBonusMap(class.Name(lang), class.AbilityBonuses),
				URL:   class.Source(lang),
			})
		}
	}
	return lines
}

func (s *Service) hydrate(ch *Character) error {
	lang := "en"
	if ch.RaceID != 0 {
		if race, err := s.Catalog.Race(ch.RaceID); err == nil {
			ch.Race = race
		}
	}
	if ch.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(ch.BackgroundID); err == nil {
			ch.Background = bg
		}
	}
	rows, err := s.Repo.ListClassLevels(ch.ID)
	if err != nil {
		return err
	}
	var classes []rules.ClassProgress
	for _, row := range rows {
		cl := rules.ClassProgress{ClassID: row.ClassID, Levels: row.Levels, SubclassID: row.SubclassID}
		if class, err := s.Catalog.Class(row.ClassID); err == nil {
			cl.Slug = class.Slug
			cl.ClassName = class.Name(lang)
			cl.HitDie = class.HitDie
			cl.SourceURL = class.Source(lang)
			cl.ClassNameEN = class.NameEN
			cl.ClassNameRU = class.NameRU
			cl.SourceEN = class.SourceURL
			cl.SourceRU = class.SourceURLRU
		}
		if row.SubclassID != 0 {
			if sub, err := s.Catalog.Subclass(row.SubclassID); err == nil {
				cl.SubclassSlug = sub.Slug
				cl.Subclass = sub.Name(lang)
				cl.SubclassEN = sub.NameEN
				cl.SubclassRU = sub.NameRU
			}
		}
		classes = append(classes, cl)
	}
	ch.ClassLevels = classes
	fids, err := s.Repo.ListFeatureIDs(ch.ID)
	if err != nil {
		return err
	}
	var feats []rules.FeatureGrant
	for _, id := range fids {
		f, err := s.Catalog.Feature(id)
		if err != nil {
			continue
		}
		feats = append(feats, rules.FeatureGrant{
			ID:        f.ID,
			Name:      f.Name(lang),
			SourceURL: f.Source(lang),
			Kind:      f.SourceKind,
			Level:     f.Level,
			NameEN:    f.NameEN,
			NameRU:    f.NameRU,
			SourceEN:  f.SourceURL,
			SourceRU:  f.SourceURLRU,
		})
	}
	ch.Features = feats
	if err := s.loadSpells(ch); err != nil {
		return err
	}
	if err := s.loadInventory(ch); err != nil {
		return err
	}
	if err := s.ensureSkills(ch); err != nil {
		return err
	}
	effects, err := s.Repo.ListEffects(ch.ID)
	if err != nil {
		return err
	}
	ch.Effects = effects
	ch.HPTemp = rules.SumTempHP(effects)
	return s.syncResources(ch)
}

func (s *Service) ensureSkills(ch *Character) error {
	raceSlug, bgSlug := "", ""
	if ch.Race != nil {
		raceSlug = ch.Race.Slug
	}
	if ch.Background != nil {
		bgSlug = ch.Background.Slug
	}
	for _, slug := range rules.GrantedSkillSlugs(raceSlug, bgSlug) {
		if err := s.Repo.GrantSkillIfNew(ch.ID, slug); err != nil {
			return err
		}
	}
	var classSlugs []string
	for _, cl := range ch.ClassLevels {
		classSlugs = append(classSlugs, cl.Slug)
	}
	for _, ab := range rules.GrantedSaveAbilities(classSlugs) {
		if err := s.Repo.GrantSaveIfNew(ch.ID, ab); err != nil {
			return err
		}
	}
	skills, err := s.Repo.ListSkills(ch.ID)
	if err != nil {
		return err
	}
	saves, err := s.Repo.ListSaves(ch.ID)
	if err != nil {
		return err
	}
	ch.SkillMarks, ch.SaveMarks = skills, saves
	return nil
}

func (s *Service) SetSkill(ch *Character, ownerID int64, slug string, proficient, expertise bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, ok := rules.SkillBySlug(slug); !ok {
		return rules.ErrUnknownCheck
	}
	if err := s.Repo.UpsertSkill(ch.ID, slug, proficient, expertise); err != nil {
		return err
	}
	return nil
}

func (s *Service) SetSave(ch *Character, ownerID int64, ability string, proficient bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if !rules.ValidAbilityKey(ability) {
		return rules.ErrUnknownCheck
	}
	return s.Repo.UpsertSave(ch.ID, strings.ToLower(ability), proficient)
}

// Localize fills class/feature display names from EN/RU catalog fields.
func (s *Service) Localize(ch *Character, lang string) {
	if ch.Race != nil {
		_ = lang
	}
	for i := range ch.ClassLevels {
		cl := &ch.ClassLevels[i]
		if cl.ClassNameEN != "" {
			cl.ClassName = catalog.Pick(lang, cl.ClassNameEN, cl.ClassNameRU)
		}
		if cl.SubclassEN != "" {
			cl.Subclass = catalog.Pick(lang, cl.SubclassEN, cl.SubclassRU)
		}
		if cl.SourceEN != "" {
			cl.SourceURL = catalog.Pick(lang, cl.SourceEN, cl.SourceRU)
		}
	}
	for i := range ch.Features {
		f := &ch.Features[i]
		if f.NameEN != "" {
			f.Name = catalog.Pick(lang, f.NameEN, f.NameRU)
			f.SourceURL = catalog.Pick(lang, f.SourceEN, f.SourceRU)
		}
	}
}

func defaultArray() rules.AbilityScores {
	return rules.AbilityScores{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 10, CHA: 8}
}

func featureIDs(d *rules.Delta) []int64 {
	ids := make([]int64, 0, len(d.FeaturesAdded))
	for _, f := range d.FeaturesAdded {
		ids = append(ids, f.ID)
	}
	return ids
}

func formatBonusMap(name string, m map[string]int) string {
	parts := make([]string, 0, 6)
	for _, k := range rules.AbilityKeys {
		if n := m[k]; n != 0 {
			sign := "+"
			if n < 0 {
				sign = ""
			}
			parts = append(parts, strings.ToUpper(k)+" "+sign+itoa(n))
		}
	}
	if len(parts) == 0 {
		return name
	}
	return name + " (" + strings.Join(parts, ", ") + ")"
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// ErrorKey maps a characters or rules error to a locales JSON key.
func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrNameRequired):
		return "error.character.name"
	case errors.Is(err, campaigns.ErrNotMember):
		return "error.not.member"
	case errors.Is(err, rules.ErrRaceRequired):
		return "error.wizard.race"
	case errors.Is(err, rules.ErrBackgroundRequired):
		return "error.wizard.background"
	case errors.Is(err, rules.ErrClassRequired):
		return "error.wizard.class"
	case errors.Is(err, rules.ErrSubclassRequired), errors.Is(err, rules.ErrSubclassInvalid):
		return "error.wizard.subclass"
	case errors.Is(err, rules.ErrArrayInvalid):
		return "error.wizard.array"
	case errors.Is(err, rules.ErrASIRequired), errors.Is(err, rules.ErrASIInvalid):
		return "error.wizard.asi"
	case errors.Is(err, rules.ErrHPRoll):
		return "error.wizard.hp"
	case errors.Is(err, rules.ErrMaxLevel):
		return "error.level.max"
	case errors.Is(err, ErrNotOwner), errors.Is(err, ErrForbidden):
		return "error.forbidden"
	case errors.Is(err, rules.ErrResourceEmpty):
		return "error.resource.empty"
	case errors.Is(err, rules.ErrSpellNotPrepared):
		return "error.spell.not_prepared"
	case errors.Is(err, rules.ErrSpellNotLearned):
		return "error.spell.not_learned"
	case errors.Is(err, rules.ErrBadFormula):
		return "error.spell.formula"
	case errors.Is(err, rules.ErrUnknownCheck):
		return "error.check.unknown"
	case errors.Is(err, rules.ErrSlotOccupied):
		return "error.item.slot"
	case errors.Is(err, rules.ErrSlotRequired):
		return "error.item.slot_required"
	case errors.Is(err, rules.ErrAttunementFull):
		return "error.item.attune"
	case errors.Is(err, rules.ErrItemNotHeld):
		return "error.item.missing"
	case errors.Is(err, rules.ErrAmount):
		return "error.hp.amount"
	case errors.Is(err, rules.ErrDamageType):
		return "error.hp.damage_type"
	default:
		return "error.generic"
	}
}

func (s *Service) loadInventory(ch *Character) error {
	rows, err := s.Repo.ListCharacterItems(ch.ID)
	if err != nil {
		return err
	}
	var out []InventoryItem
	for _, row := range rows {
		it, err := s.Catalog.Item(row.CatalogID)
		if err != nil {
			continue
		}
		resolved := s.resolveItem(*it)
		row.AttachCatalog(resolved)
		out = append(out, row)
	}
	ch.Inventory = out
	s.loadGrantedSpells(ch)
	return nil
}

func (s *Service) loadGrantedSpells(ch *Character) {
	var out []GrantedSpell
	seen := map[int64]bool{}
	for _, it := range ch.Inventory {
		if !it.Equipped() {
			continue
		}
		g := rules.GrantsFromFeatures(it.Features)
		for _, id := range g.SpellIDs {
			if seen[id] {
				continue
			}
			sp, err := s.Catalog.Spell(id)
			if err != nil {
				continue
			}
			seen[id] = true
			out = append(out, GrantedSpell{
				Spell: *sp, ItemID: it.ID,
				ItemNameEN: it.DisplayName("en"), ItemNameRU: it.DisplayName("ru"),
			})
		}
	}
	ch.GrantedSpells = out
}

func (s *Service) held(ch *Character, id int64) (InventoryItem, bool) {
	return ch.InventoryByID(id)
}

type AddItemInput struct {
	CatalogID      int64
	Quantity       int
	EquipSlot      string
	EquipNow       bool
	Attune         bool
	ChargesCurrent int
	ChargesMax     int
	CustomName     string
	Notes          string
	Features       []rules.FeatureDTO
	PlusN          int
	AtkBonus       int
	DmgBonus       int
	ExtraDamage    string
	ACBonus        int
	ACBase         int
	ACFloor        int
	SpeedBonus     int
	SpeedMult      int
	TwoHanded      bool
}

func (in AddItemInput) overlayFeatures(cat catalog.Item) []rules.FeatureDTO {
	var out []rules.FeatureDTO
	if in.PlusN != 0 {
		out = append(out, rules.CommonPlusFor(cat, in.PlusN)...)
	}
	push := func(stat, value, nameEN, nameRU string) {
		if value == "" || value == "0" {
			return
		}
		out = append(out, rules.FeatureDTO{
			NameEN: nameEN, NameRU: nameRU, Stat: stat, Value: value, Origin: rules.OriginCommon,
		})
	}
	if in.AtkBonus != 0 {
		push(rules.StatAtkBonus, strconv.Itoa(in.AtkBonus), "Attack bonus", "Бонус атаки")
	}
	if in.DmgBonus != 0 {
		push(rules.StatDmgBonus, strconv.Itoa(in.DmgBonus), "Damage bonus", "Бонус урона")
	}
	if x := strings.TrimSpace(in.ExtraDamage); x != "" {
		push(rules.StatExtraDice, x, "Extra dice", "Доп. кости")
	}
	if in.ACBonus != 0 {
		push(rules.StatACBonus, strconv.Itoa(in.ACBonus), "AC bonus", "Бонус КД")
	}
	if in.ACBase != 0 {
		push(rules.StatACBase, strconv.Itoa(in.ACBase), "AC base", "База КД")
	}
	if in.ACFloor != 0 {
		push(rules.StatACFloor, strconv.Itoa(in.ACFloor), "AC floor", "Минимум КД")
	}
	if in.SpeedBonus != 0 {
		push(rules.StatSpeedBonus, strconv.Itoa(in.SpeedBonus), "Speed bonus", "Бонус скорости")
	}
	if in.SpeedMult != 0 {
		push(rules.StatSpeedMult, strconv.Itoa(in.SpeedMult), "Speed ×", "Скорость ×")
	}
	return out
}

func (in AddItemInput) asItem(cat catalog.Item) InventoryItem {
	qty := in.Quantity
	if qty < 1 {
		qty = 1
	}
	chargesMax := in.ChargesMax
	if chargesMax < 0 {
		chargesMax = 0
	}
	if chargesMax == 0 {
		chargesMax = cat.ChargesMax
	}
	cur := in.ChargesCurrent
	if cur < 0 {
		cur = 0
	}
	if cur == 0 && chargesMax > 0 {
		cur = chargesMax
	}
	if cur > chargesMax {
		cur = chargesMax
	}
	return InventoryItem{
		CatalogID: cat.ID, Item: cat, Quantity: qty,
		ChargesCurrent: cur, ChargesMax: chargesMax,
		CustomName: strings.TrimSpace(in.CustomName), Notes: strings.TrimSpace(in.Notes),
		TwoHanded: in.TwoHanded, Features: append([]rules.FeatureDTO{}, in.Features...),
	}
}

func (s *Service) resolveItem(cat catalog.Item) catalog.Item {
	if slug := rules.InheritWeaponSlug(cat.Slug); slug != "" {
		if row, err := s.Catalog.ItemBySlug(slug); err == nil {
			if cat.DamageDice == "" {
				cat.DamageDice = row.DamageDice
				cat.DamageType = row.DamageType
			}
			if len(cat.Properties) == 0 {
				cat.Properties = append([]string{}, row.Properties...)
			}
			if cat.WeaponCategory == "" {
				cat.WeaponCategory = row.WeaponCategory
			}
			if cat.VersatileDice == "" {
				cat.VersatileDice = row.VersatileDice
			}
			if cat.RangeNormal == 0 {
				cat.RangeNormal, cat.RangeLong = row.RangeNormal, row.RangeLong
			}
		}
	}
	return cat
}

func (s *Service) catalogFeatures(cat catalog.Item) []rules.FeatureDTO {
	var inherit *catalog.Item
	if slug := rules.InheritWeaponSlug(cat.Slug); slug != "" {
		if row, err := s.Catalog.ItemBySlug(slug); err == nil {
			inherit = row
		}
	}
	return rules.CatalogItemFeatures(cat, inherit)
}

func (s *Service) FeaturesFromRef(raw string) []rules.FeatureDTO {
	raw = strings.TrimSpace(raw)
	ref, ok := rules.ParseDND14URL(raw)
	if !ok {
		return rules.FeaturesFromUnknownURL(raw)
	}
	if ref.Type == "spells" {
		if sp, err := s.Catalog.Spell(ref.ID); err == nil {
			return []rules.FeatureDTO{rules.FeatureFromSpell(*sp)}
		}
		if sp, err := s.Catalog.SpellBySlug(ref.Slug); err == nil {
			return []rules.FeatureDTO{rules.FeatureFromSpell(*sp)}
		}
	} else {
		if it, err := s.Catalog.Item(ref.ID); err == nil {
			return rules.FeaturesFromItem(*it)
		}
		if it, err := s.Catalog.ItemBySlug(ref.Slug); err == nil {
			return rules.FeaturesFromItem(*it)
		}
	}
	return rules.FeaturesFromURL(ref)
}

func (s *Service) AddItem(ch *Character, ownerID int64, in AddItemInput) (InventoryItem, error) {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return InventoryItem{}, err
	}
	cat, err := s.Catalog.Item(in.CatalogID)
	if err != nil {
		return InventoryItem{}, err
	}
	resolved := s.resolveItem(*cat)
	it := in.asItem(resolved)
	it.Features = append(append([]rules.FeatureDTO{}, in.Features...), in.overlayFeatures(resolved)...)
	it.SyncOverlayFromFeatures()
	it.AttachCatalog(resolved)
	if in.EquipNow {
		slot := in.EquipSlot
		if slot == "" {
			slot = cat.SuggestedSlot
		}
		piece := it.GearPiece()
		if cat.RequiresAttunement {
			piece.Attuned = true
			piece.RequiresAttune = true
		}
		if piece.TwoHanded {
			piece.Slot = rules.SlotMainHand
		} else if piece.Shield() {
			piece.Slot = rules.SlotShield
		} else {
			piece.Slot = slot
		}
		if err := rules.CheckEquip(ch.EquippedGear(), piece); err != nil {
			return InventoryItem{}, err
		}
	}
	if cat.Stackable() && !it.HasOverlay() && !in.EquipNow {
		for i := range ch.Inventory {
			ex := ch.Inventory[i]
			if ex.CatalogID == cat.ID && !ex.Equipped() && !ex.HasOverlay() && ex.Item.Stackable() {
				ex.Quantity += it.Quantity
				if err := s.Repo.UpdateCharacterItem(ch.ID, ex); err != nil {
					return InventoryItem{}, err
				}
				return ex, s.loadInventory(ch)
			}
		}
	}
	id, err := s.Repo.InsertCharacterItem(ch.ID, it)
	if err != nil {
		return InventoryItem{}, err
	}
	it.ID = id
	if err := s.loadInventory(ch); err != nil {
		return it, err
	}
	if in.EquipNow {
		slot := in.EquipSlot
		if slot == "" {
			slot = cat.SuggestedSlot
		}
		if err := s.EquipItem(ch, ownerID, id, slot); err != nil {
			return it, err
		}
		got, _ := s.held(ch, id)
		return got, nil
	}
	return it, nil
}

func (s *Service) RemoveItem(ch *Character, ownerID, invID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, ok := s.held(ch, invID); !ok {
		return rules.ErrItemNotHeld
	}
	return s.Repo.DeleteCharacterItem(ch.ID, invID)
}

func (s *Service) AdjustItemQty(ch *Character, ownerID, invID int64, delta int) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	it, ok := s.held(ch, invID)
	if !ok {
		return rules.ErrItemNotHeld
	}
	it.Quantity += delta
	if it.Quantity < 1 {
		return s.Repo.DeleteCharacterItem(ch.ID, invID)
	}
	return s.Repo.UpdateCharacterItem(ch.ID, it)
}

func (s *Service) EquipItem(ch *Character, ownerID, invID int64, slot string) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	it, ok := s.held(ch, invID)
	if !ok {
		return rules.ErrItemNotHeld
	}
	piece := it.GearPiece()
	if slot == "" {
		slot = it.Item.SuggestedSlot
	}
	occ := rules.OccupiedMap(ch.EquippedGear())
	norm, ok := rules.NormalizeEquipSlot(slot, it.Item.SuggestedSlot, occ)
	if !ok {
		return rules.ErrSlotRequired
	}
	if piece.TwoHanded {
		norm = rules.SlotMainHand
	}
	if piece.Shield() {
		norm = rules.SlotShield
	}
	piece.Slot = norm
	if it.Item.RequiresAttunement {
		piece.Attuned = true
		piece.RequiresAttune = true
	}
	if err := rules.CheckEquip(ch.EquippedGear(), piece); err != nil {
		return err
	}
	it.EquippedSlot = norm
	it.Attuned = it.Item.RequiresAttunement
	if it.TwoHanded || it.Item.IsTwoHanded() {
		it.TwoHanded = it.TwoHanded || it.Item.IsTwoHanded()
	}
	if err := s.Repo.UpdateCharacterItem(ch.ID, it); err != nil {
		return err
	}
	return s.loadInventory(ch)
}

func (s *Service) UnequipItem(ch *Character, ownerID, invID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	it, ok := s.held(ch, invID)
	if !ok {
		return rules.ErrItemNotHeld
	}
	it.EquippedSlot = ""
	it.Attuned = false
	if err := s.Repo.UpdateCharacterItem(ch.ID, it); err != nil {
		return err
	}
	return s.loadInventory(ch)
}

func (s *Service) UseItem(ch *Character, ownerID, invID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	it, ok := s.held(ch, invID)
	if !ok {
		return rules.ErrItemNotHeld
	}
	if err := s.applyConsumable(ch, it); err != nil {
		return err
	}
	it.Quantity--
	if it.Quantity < 1 {
		if err := s.Repo.DeleteCharacterItem(ch.ID, it.ID); err != nil {
			return err
		}
	} else if err := s.Repo.UpdateCharacterItem(ch.ID, it); err != nil {
		return err
	}
	if err := s.loadInventory(ch); err != nil {
		return err
	}
	return s.syncTempFromEffects(ch)
}

func (s *Service) applyConsumable(ch *Character, it InventoryItem) error {
	nameEN, nameRU := it.Item.NameEN, it.Item.NameRU
	if it.CustomName != "" {
		nameEN, nameRU = it.CustomName, it.CustomName
	}
	if e, ok := rules.PotionCombatEffect(it.Item.Slug, nameEN, nameRU); ok {
		if err := s.Repo.UpsertEffect(ch.ID, e); err != nil {
			return err
		}
		return nil
	}
	formula := rules.HealingPotionFormula(it.Item.Slug)
	if formula == "" && it.Item.DamageType == "healing" {
		formula = it.Item.DamageDice
	}
	if formula == "" {
		return nil
	}
	res, err := rules.RollFormula(formula)
	if err != nil {
		return err
	}
	ch.HPCurrent = rules.ClampHP(ch.HPCurrent+res.Total, ch.HPMax)
	return s.saveVitals(ch)
}

func (s *Service) AppendItemNotes(ch *Character, ownerID, invID int64, note string) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	it, ok := s.held(ch, invID)
	if !ok {
		return rules.ErrItemNotHeld
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return nil
	}
	if it.Notes != "" {
		it.Notes += "\n"
	}
	it.Notes += note
	return s.Repo.UpdateCharacterItem(ch.ID, it)
}

func (s *Service) loadSpells(ch *Character) error {
	rows, err := s.Repo.ListCharacterSpells(ch.ID)
	if err != nil {
		return err
	}
	var out []LearnedSpell
	for _, row := range rows {
		sp, err := s.Catalog.Spell(row.SpellID)
		if err != nil {
			continue
		}
		out = append(out, LearnedSpell{Spell: *sp, Prepared: row.Prepared})
	}
	ch.Spells = out
	return nil
}

// TODO: long rest — restore all pool Current to Max, and allow rewriting which
// learned spells are marked prepared (PHB prepare-after-rest). Not in this pass.
func (s *Service) syncResources(ch *Character) error {
	maxes := rules.MaxResources(ch.ClassRules())
	cur, err := s.Repo.ListResources(ch.ID)
	if err != nil {
		return err
	}
	next := rules.SyncPools(cur, maxes)
	if !rules.PoolsEqual(cur, next) {
		if err := s.Repo.ReplaceResources(ch.ID, next); err != nil {
			return err
		}
	}
	ch.Resources = next
	return nil
}

func (s *Service) learned(ch *Character, spellID int64) (LearnedSpell, bool) {
	for _, ls := range ch.Spells {
		if ls.Spell.ID == spellID {
			return ls, true
		}
	}
	return LearnedSpell{}, false
}

func (s *Service) grantedSpell(ch *Character, spellID int64) (GrantedSpell, bool) {
	for _, g := range ch.GrantedSpells {
		if g.Spell.ID == spellID {
			return g, true
		}
	}
	return GrantedSpell{}, false
}

func (s *Service) spellForUse(ch *Character, spellID int64) (catalog.Spell, bool, bool) {
	if ls, ok := s.learned(ch, spellID); ok {
		fromItem := false
		if _, gok := s.grantedSpell(ch, spellID); gok {
			fromItem = true
		}
		return ls.Spell, ls.Prepared, fromItem
	}
	if g, ok := s.grantedSpell(ch, spellID); ok {
		return g.Spell, true, true
	}
	return catalog.Spell{}, false, false
}

func (s *Service) AddLearned(ch *Character, ownerID, spellID int64, prepared bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, err := s.Catalog.Spell(spellID); err != nil {
		return err
	}
	return s.Repo.UpsertCharacterSpell(ch.ID, spellID, prepared)
}

func (s *Service) RemoveLearned(ch *Character, ownerID, spellID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	return s.Repo.DeleteCharacterSpell(ch.ID, spellID)
}

func (s *Service) SetPrepared(ch *Character, ownerID, spellID int64, prepared bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, ok := s.learned(ch, spellID); !ok {
		return rules.ErrSpellNotLearned
	}
	return s.Repo.SetPrepared(ch.ID, spellID, prepared)
}

func (s *Service) SpendResource(ch *Character, ownerID int64, kind string, slotLevel, amount int) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	next, err := rules.Consume(ch.Resources, kind, slotLevel, amount)
	if err != nil {
		return err
	}
	if err := s.Repo.ReplaceResources(ch.ID, next); err != nil {
		return err
	}
	ch.Resources = next
	return nil
}

func (s *Service) CastSpell(ch *Character, ownerID, spellID int64, kind string, slotLevel, amount int) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	sp, prepared, fromItem := s.spellForUse(ch, spellID)
	if sp.ID == 0 {
		return rules.ErrSpellNotLearned
	}
	if !prepared {
		return rules.ErrSpellNotPrepared
	}
	if kind == rules.KindItem || (fromItem && kind == rules.KindItem) {
		if !fromItem {
			return rules.ErrSpellNotLearned
		}
		if sp.Level == 0 {
			return s.applyCastCombat(ch, sp, slotLevel)
		}
		if slotLevel < sp.Level {
			slotLevel = sp.Level
		}
		return s.applyCastCombat(ch, sp, slotLevel)
	}
	if sp.Level == 0 {
		return s.applyCastCombat(ch, sp, slotLevel)
	}
	ls, learned := s.learned(ch, spellID)
	if !learned {
		if fromItem {
			if slotLevel < sp.Level {
				slotLevel = sp.Level
			}
			return s.applyCastCombat(ch, sp, slotLevel)
		}
		return rules.ErrSpellNotLearned
	}
	if kind == "" {
		if rules.HasKind(ch.Resources, rules.KindPact) && !rules.HasKind(ch.Resources, rules.KindSlots) {
			kind = rules.KindPact
		} else {
			kind = rules.KindSlots
		}
	}
	if kind == rules.KindPact {
		if p, ok := rules.PactPool(ch.Resources); ok {
			slotLevel = p.SlotLevel
		}
	}
	if slotLevel < ls.Spell.Level && kind == rules.KindSlots {
		slotLevel = ls.Spell.Level
	}
	if err := s.SpendResource(ch, ownerID, kind, slotLevel, amount); err != nil {
		return err
	}
	return s.applyCastCombat(ch, ls.Spell, slotLevel)
}

func (s *Service) applyCastCombat(ch *Character, sp catalog.Spell, slotLevel int) error {
	formula, _, _ := sp.FormulaAt(slotLevel, ch.Level)
	temp := rules.ParseFlatAmount(formula)
	e, ok := rules.SpellCombatEffect(sp.Slug, sp.NameEN, sp.NameRU, sp.ID, temp)
	if !ok {
		return nil
	}
	if err := s.Repo.UpsertEffect(ch.ID, e); err != nil {
		return err
	}
	return s.syncTempFromEffects(ch)
}

func (s *Service) syncTempFromEffects(ch *Character) error {
	list, err := s.Repo.ListEffects(ch.ID)
	if err != nil {
		return err
	}
	ch.Effects = list
	ch.HPTemp = rules.SumTempHP(list)
	return s.saveVitals(ch)
}

func (s *Service) replaceEffects(ch *Character, next []rules.Effect) error {
	cur, err := s.Repo.ListEffects(ch.ID)
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, e := range next {
		keep[e.Slug] = true
		if err := s.Repo.UpsertEffect(ch.ID, e); err != nil {
			return err
		}
	}
	for _, e := range cur {
		if keep[e.Slug] {
			continue
		}
		if err := s.Repo.DeleteEffect(ch.ID, e.ID); err != nil {
			return err
		}
	}
	return s.syncTempFromEffects(ch)
}

func (s *Service) saveVitals(ch *Character) error {
	ch.HPCurrent = rules.ClampHP(ch.HPCurrent, ch.HPMax)
	ch.HPTemp = rules.ClampTempHP(ch.HPTemp)
	if ch.DeathSuccess < 0 {
		ch.DeathSuccess = 0
	}
	if ch.DeathSuccess > 3 {
		ch.DeathSuccess = 3
	}
	if ch.DeathFail < 0 {
		ch.DeathFail = 0
	}
	if ch.DeathFail > 3 {
		ch.DeathFail = 3
	}
	if ch.HPCurrent > 0 {
		ch.DeathSuccess, ch.DeathFail = 0, 0
	}
	return s.persistVitals(ch)
}

func (s *Service) persistVitals(ch *Character) error {
	if err := s.Repo.UpdateVitals(ch.ID, ch.HPCurrent, ch.HPTemp, ch.DeathSuccess, ch.DeathFail); err != nil {
		return err
	}
	s.broadcastVitals(ch.ID)
	return nil
}

func (s *Service) ApplyHPDamage(ch *Character, userID int64, amount int, dmgType string) (rules.DamageResult, error) {
	if err := s.RequireCombatEdit(ch, userID); err != nil {
		return rules.DamageResult{}, err
	}
	if amount < 1 {
		return rules.DamageResult{}, rules.ErrAmount
	}
	if !rules.ValidDamageType(dmgType) {
		return rules.DamageResult{}, rules.ErrDamageType
	}
	res := rules.ApplyDamage(ch.HPCurrent, ch.HPMax, ch.Effects, ch.EquippedGrants(), amount, dmgType)
	ch.HPCurrent = res.NewHP
	if res.AbsorbedTemp > 0 || len(res.NewEffects) != len(ch.Effects) {
		if err := s.replaceEffects(ch, res.NewEffects); err != nil {
			return res, err
		}
		return res, nil
	}
	if err := s.saveVitals(ch); err != nil {
		return res, err
	}
	return res, nil
}

func (s *Service) ApplyHPHeal(ch *Character, userID int64, amount int) error {
	if err := s.RequireCombatEdit(ch, userID); err != nil {
		return err
	}
	if amount < 1 {
		return rules.ErrAmount
	}
	ch.HPCurrent = rules.ClampHP(ch.HPCurrent+amount, ch.HPMax)
	return s.saveVitals(ch)
}

func (s *Service) AdjustHP(ch *Character, ownerID int64, delta int) error {
	if err := s.RequireCombatEdit(ch, ownerID); err != nil {
		return err
	}
	ch.HPCurrent = rules.ClampHP(ch.HPCurrent+delta, ch.HPMax)
	return s.saveVitals(ch)
}

func (s *Service) AdjustTempHP(ch *Character, ownerID int64, delta int) error {
	if err := s.RequireCombatEdit(ch, ownerID); err != nil {
		return err
	}
	list := append([]rules.Effect{}, ch.Effects...)
	if delta > 0 {
		found := false
		for i := range list {
			if list[i].Slug != rules.OtherTempSlug {
				continue
			}
			list[i].TempHP += delta
			list[i].FormulaEN, list[i].FormulaRU = rules.EffectFormula(list[i])
			found = true
			break
		}
		if !found {
			list = append(list, rules.OtherTempEffect(delta))
		}
		return s.replaceEffects(ch, list)
	}
	return s.replaceEffects(ch, rules.ReduceStackedTemp(list, -delta))
}

func (s *Service) ToggleDeath(ch *Character, ownerID int64, fail bool, pip int) error {
	if err := s.RequireCombatEdit(ch, ownerID); err != nil {
		return err
	}
	if fail {
		ch.DeathFail = rules.ToggleDeathPip(ch.DeathFail, pip)
	} else {
		ch.DeathSuccess = rules.ToggleDeathPip(ch.DeathSuccess, pip)
	}
	return s.persistVitals(ch)
}

func (s *Service) DismissEffect(ch *Character, ownerID, effectID int64) error {
	if err := s.RequireCombatEdit(ch, ownerID); err != nil {
		return err
	}
	if err := s.Repo.DeleteEffect(ch.ID, effectID); err != nil {
		return err
	}
	return s.syncTempFromEffects(ch)
}

func (s *Service) RollSpell(ch *Character, spellID int64, slotLevel int) (rules.RollResult, catalog.Spell, error) {
	sp, _, _ := s.spellForUse(ch, spellID)
	if sp.ID == 0 {
		return rules.RollResult{}, catalog.Spell{}, rules.ErrSpellNotLearned
	}
	if slotLevel < 1 {
		if p, ok := rules.PactPool(ch.Resources); ok && !rules.HasKind(ch.Resources, rules.KindSlots) {
			slotLevel = p.SlotLevel
		} else {
			slotLevel = sp.Level
		}
	}
	formula, _, heal := sp.FormulaAt(slotLevel, ch.Level)
	expr := formula
	if expr == "" {
		expr = heal
	}
	res, err := rules.RollFormula(expr)
	return res, sp, err
}
