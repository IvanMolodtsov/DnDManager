// Package characters owns live sheets, creation drafts, the wizard, and level-up.
package characters

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

// Character is a live sheet: scores, HP, class levels, and granted features.
type Character struct {
	ID               int64
	Name             string
	OwnerID          int64
	OwnerName        string
	CampaignID       int64
	Campaign         string
	Level            int
	STR              int
	DEX              int
	CON              int
	INT              int
	WIS              int
	CHA              int
	RaceID           int64
	BackgroundID     int64
	HPMax            int
	HPCurrent        int
	HPTemp           int
	DeathSuccess     int
	DeathFail        int
	ProficiencyBonus int
	Effects          []rules.Effect
	CreatedAt        time.Time
	Race             *catalog.Race
	Background       *catalog.Background
	ClassLevels      []rules.ClassProgress
	Features         []rules.FeatureGrant
	Resources        []rules.Pool
	Spells           []LearnedSpell
	SkillMarks       []rules.SkillMark
	SaveMarks        []rules.SaveMark
	Inventory        []InventoryItem
	GrantedSpells    []GrantedSpell
}

// LearnedSpell is a catalog spell on the character (prepared is a subset).
type LearnedSpell struct {
	Spell    catalog.Spell
	Prepared bool
}

func (c *Character) PreparedSpells() []LearnedSpell {
	var out []LearnedSpell
	for _, s := range c.Spells {
		if s.Prepared {
			out = append(out, s)
		}
	}
	return out
}

func (c *Character) ClassRules() []rules.ClassLevel {
	out := make([]rules.ClassLevel, 0, len(c.ClassLevels))
	for _, cl := range c.ClassLevels {
		out = append(out, rules.ClassLevel{Slug: cl.Slug, Levels: cl.Levels, SubclassSlug: cl.SubclassSlug})
	}
	return out
}

func (c *Character) BaseScores() rules.AbilityScores {
	return rules.AbilityScores{STR: c.STR, DEX: c.DEX, CON: c.CON, INT: c.INT, WIS: c.WIS, CHA: c.CHA}
}

func (c *Character) Scores() rules.AbilityScores {
	return c.BaseScores().Add(c.EquippedGrants().Ability)
}

func (c *Character) CombatInput() rules.CombatInput {
	speed := 30
	if c.Race != nil && c.Race.Speed > 0 {
		speed = c.Race.Speed
	}
	return rules.CombatInput{
		Scores: c.Scores(), Classes: c.ClassRules(), RaceSpeed: speed,
		HPCurrent: c.HPCurrent, HPMax: c.HPMax, TempHP: c.HPTemp,
		DeathSuccess: c.DeathSuccess, DeathFail: c.DeathFail, Effects: c.Effects,
		Gear: c.EquippedGear(),
	}
}

type ConsumableDTO struct {
	Slug     string
	Quantity int
}

type ArmorDTO struct {
	Category      string
	ACBase        int
	DexCap        int
	StealthDisadv bool
	StrengthMin   int
	Features      []rules.FeatureDTO
}

type JewelryDTO struct {
	BaseSlug string
	Slot     string
	Features []rules.FeatureDTO
}

func ArmorFromItem(item catalog.Item, feats []rules.FeatureDTO) *ArmorDTO {
	return &ArmorDTO{
		Category: item.ArmorCategory, ACBase: item.ACBase, DexCap: item.DexMax,
		StealthDisadv: item.StealthDisadv, StrengthMin: item.StrMin, Features: feats,
	}
}

func JewelryFromItem(item catalog.Item, feats []rules.FeatureDTO) *JewelryDTO {
	return &JewelryDTO{BaseSlug: item.Slug, Slot: item.SuggestedSlot, Features: feats}
}

type GrantedSpell struct {
	Spell      catalog.Spell
	ItemID     int64
	ItemNameEN string
	ItemNameRU string
}

func (g GrantedSpell) ItemName(lang string) string {
	return catalog.Pick(lang, g.ItemNameEN, g.ItemNameRU)
}

// InventoryItem is a sheet instance: header (qty, slot, notes, charges, grip) plus typed payload.
type InventoryItem struct {
	ID             int64
	CatalogID      int64
	Item           catalog.Item
	Quantity       int
	EquippedSlot   string
	Attuned        bool
	ChargesCurrent int
	ChargesMax     int
	CustomName     string
	Notes          string
	AtkBonus       int
	DmgBonus       int
	ExtraDamage    string
	ACBonus        int
	ACBase         int
	ACFloor        int
	SpeedBonus     int
	SpeedMult      int
	TwoHanded      bool
	Features       []rules.FeatureDTO
	Weapon         *rules.WeaponDTO
	Consumable     *ConsumableDTO
	Armor          *ArmorDTO
	Jewelry        *JewelryDTO
}

func (it InventoryItem) DisplayName(lang string) string {
	if strings.TrimSpace(it.CustomName) != "" {
		return it.CustomName
	}
	return it.Item.Name(lang)
}

func (it InventoryItem) Equipped() bool { return it.EquippedSlot != "" }

func (it InventoryItem) HasOverlay() bool {
	if strings.TrimSpace(it.CustomName) != "" || strings.TrimSpace(it.Notes) != "" || it.TwoHanded {
		return true
	}
	for _, f := range it.Features {
		if f.Manual() {
			return true
		}
	}
	return false
}

func (it *InventoryItem) SyncOverlayFromFeatures() {
	d := rules.DeriveItemStats(it.Item.DamageDice, it.Item.DamageType, it.Features)
	it.AtkBonus, it.DmgBonus = d.AtkBonus, d.DmgBonus
	it.ACBonus, it.ACBase, it.ACFloor = d.ACBonus, d.ACBase, d.ACFloor
	it.SpeedBonus, it.SpeedMult = d.SpeedBonus, d.SpeedMult
	it.ExtraDamage = strings.Join(d.Extra, " + ")
}

func (it InventoryItem) DerivedStats() rules.DerivedItemStats {
	base, typ := it.Item.DamageDice, it.Item.DamageType
	if it.Weapon != nil {
		if it.Weapon.BaseHit != "" {
			base = it.Weapon.BaseHit
		}
		if it.Weapon.DamageType != "" {
			typ = it.Weapon.DamageType
		}
	}
	return rules.DeriveItemStats(base, typ, it.Features)
}

func (it *InventoryItem) AttachCatalog(cat catalog.Item) {
	it.Item = cat
	it.CatalogID = cat.ID
	it.Weapon, it.Consumable, it.Armor, it.Jewelry = nil, nil, nil, nil
	if cat.Consumable {
		it.Consumable = &ConsumableDTO{Slug: cat.Slug, Quantity: it.Quantity}
		return
	}
	if cat.IsArmor() {
		it.Armor = ArmorFromItem(cat, it.Features)
		return
	}
	if cat.IsJewelry() {
		it.Jewelry = JewelryFromItem(cat, it.Features)
		return
	}
	if cat.Kind == "weapon" || cat.IsWeaponBase() || cat.DamageDice != "" {
		it.Weapon = rules.WeaponFromItem(cat, it.DisplayName("en"), it.Features)
	}
}

func (c *Character) EquippedGrants() rules.ItemGrants {
	var g rules.ItemGrants
	for _, it := range c.Inventory {
		if !it.Equipped() {
			continue
		}
		g = rules.MergeGrants(g, rules.GrantsFromFeatures(it.Features))
	}
	return g
}

func (c *Character) EffectiveSkillMark(slug string) rules.SkillMark {
	mark := rules.SkillMark{Slug: slug}
	for _, m := range c.SkillMarks {
		if m.Slug == slug {
			mark = m
			break
		}
	}
	if c.EquippedGrants().HasSkill(slug) {
		mark.Proficient = true
	}
	return mark
}

func (it InventoryItem) GearPiece() rules.GearPiece {
	acBase := it.Item.ACBase
	if it.ACBase > 0 {
		acBase = it.ACBase
	}
	return rules.GearPiece{
		Slot:           it.EquippedSlot,
		Name:           it.DisplayName("en"),
		ArmorCategory:  it.Item.ArmorCategory,
		ACBase:         acBase,
		DexMax:         it.Item.DexMax,
		ACBonus:        it.ACBonus,
		ACFloor:        it.ACFloor,
		SpeedBonus:     it.SpeedBonus,
		SpeedMult:      it.SpeedMult,
		TwoHanded:      it.TwoHanded || it.Item.IsTwoHanded(),
		IsShield:       it.Item.IsShield() || it.EquippedSlot == rules.SlotShield,
		RequiresAttune: it.Item.RequiresAttunement,
		Attuned:        it.Attuned,
	}
}

func (c *Character) EquippedGear() []rules.GearPiece {
	var out []rules.GearPiece
	for _, it := range c.Inventory {
		if !it.Equipped() {
			continue
		}
		out = append(out, it.GearPiece())
	}
	return out
}

func (c *Character) InventoryByID(id int64) (InventoryItem, bool) {
	for _, it := range c.Inventory {
		if it.ID == id {
			return it, true
		}
	}
	return InventoryItem{}, false
}

// State is the snapshot the rules engine uses for previews.
func (c *Character) State() rules.State {
	ids := make([]int64, 0, len(c.Features))
	for _, f := range c.Features {
		ids = append(ids, f.ID)
	}
	return rules.State{
		Level:            c.Level,
		Scores:           c.BaseScores(),
		HPMax:            c.HPMax,
		ProficiencyBonus: c.ProficiencyBonus,
		RaceID:           c.RaceID,
		BackgroundID:     c.BackgroundID,
		Classes:          append([]rules.ClassProgress{}, c.ClassLevels...),
		FeatureIDs:       ids,
	}
}

// ClassLine is a display string like "Fighter 3 (Champion) / Wizard 1".
func (c *Character) ClassLine() string {
	if len(c.ClassLevels) == 0 {
		return ""
	}
	out := ""
	for i, cl := range c.ClassLevels {
		if i > 0 {
			out += " / "
		}
		name := cl.ClassName
		if name == "" {
			name = "Class"
		}
		out += fmt.Sprintf("%s %d", name, cl.Levels)
		if cl.Subclass != "" {
			out += " (" + cl.Subclass + ")"
		}
	}
	return out
}

// Draft is an in-progress wizard, one per owner+campaign.
type Draft struct {
	ID           int64
	OwnerID      int64
	CampaignID   int64
	Step         int
	Name         string
	RaceID       int64
	BackgroundID int64
	ClassID      int64
	SubclassID   int64
	BaseScores   rules.AbilityScores
	HasScores    bool
}

func (d *Draft) ScoresJSON() string {
	if !d.HasScores {
		return ""
	}
	b, _ := json.Marshal(d.BaseScores)
	return string(b)
}

func parseScoresJSON(s string) (rules.AbilityScores, bool) {
	if s == "" {
		return rules.AbilityScores{}, false
	}
	var a rules.AbilityScores
	if err := json.Unmarshal([]byte(s), &a); err != nil {
		return rules.AbilityScores{}, false
	}
	return a, true
}

// ScoreRow is one ability on the wizard scores step (base + bonus = total).
type ScoreRow struct {
	Key      string
	LabelKey string
	Base     int
	Bonus    int
	Total    int
	Mod      string
}

func scoreRows(base, bonus rules.AbilityScores) []ScoreRow {
	keys := []struct{ key, label string }{
		{"str", "ability.str"},
		{"dex", "ability.dex"},
		{"con", "ability.con"},
		{"int", "ability.int"},
		{"wis", "ability.wis"},
		{"cha", "ability.cha"},
	}
	var rows []ScoreRow
	for _, k := range keys {
		b := base.Get(k.key)
		n := bonus.Get(k.key)
		total := b + n
		rows = append(rows, ScoreRow{
			Key: k.key, LabelKey: k.label, Base: b, Bonus: n, Total: total,
			Mod: formatMod(rules.Modifier(total)),
		})
	}
	return rows
}

func formatMod(m int) string {
	if m >= 0 {
		return "+" + strconv.Itoa(m)
	}
	return strconv.Itoa(m)
}

// BonusLine is a labeled race/background/class bonus for the wizard UI.
type BonusLine struct {
	Label string
	Text  string
	URL   string
}
