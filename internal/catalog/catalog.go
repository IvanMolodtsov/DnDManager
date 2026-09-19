// Package catalog is the local PHB 5e (2014) reference data.
// Russian source URLs follow https://5e14.dnd.su/{type}/{numericId}-{english-slug}/
// (type = class | race | backgrounds | spells | items). Subclass and feature rows
// link to the parent page when 5e14 has no dedicated URL.
package catalog

import (
	"fmt"
	"strings"
)

const (
	KindRace       = "race"
	KindBackground = "background"
	KindClass      = "class"
	KindSubclass   = "subclass"
)

// Race is a playable race or subrace (e.g. hill dwarf) with ability bonuses.
type Race struct {
	ID              int64
	Slug            string
	NameEN          string
	NameRU          string
	SourceURL       string
	SourceURLRU     string
	AbilityBonuses  map[string]int
	Speed           int
	HPBonusPerLevel int
}

// Background is a PHB background. AbilityBonuses is empty on 2014 PHB rows.
type Background struct {
	ID             int64
	Slug           string
	NameEN         string
	NameRU         string
	SourceURL      string
	SourceURLRU    string
	AbilityBonuses map[string]int
}

// Class is a PHB class: hit die, subclass unlock level, and ASI levels.
type Class struct {
	ID             int64
	Slug           string
	NameEN         string
	NameRU         string
	HitDie         int
	SubclassLevel  int
	ASILevels      []int
	SourceURL      string
	SourceURLRU    string
	AbilityBonuses map[string]int
}

// HasASI reports whether this class level grants an Ability Score Improvement.
func (c Class) HasASI(classLevel int) bool {
	for _, lv := range c.ASILevels {
		if lv == classLevel {
			return true
		}
	}
	return false
}

// Subclass is a class option (archetype). ClassID is the parent class.
type Subclass struct {
	ID          int64
	ClassID     int64
	Slug        string
	NameEN      string
	NameRU      string
	SourceURL   string
	SourceURLRU string
}

// Feature is a race, background, class, or subclass grant at a given level.
type Feature struct {
	ID          int64
	SourceKind  string
	SourceID    int64
	Level       int
	Slug        string
	NameEN      string
	NameRU      string
	SourceURL   string
	SourceURLRU string
}

// Pick returns ru when lang is "ru" and ru is non-empty; otherwise en.
func Pick(lang, en, ru string) string {
	if lang == "ru" && ru != "" {
		return ru
	}
	return en
}

func (r Race) Name(lang string) string       { return Pick(lang, r.NameEN, r.NameRU) }
func (r Race) Source(lang string) string     { return Pick(lang, r.SourceURL, r.SourceURLRU) }
func (b Background) Name(lang string) string { return Pick(lang, b.NameEN, b.NameRU) }
func (b Background) Source(lang string) string {
	return Pick(lang, b.SourceURL, b.SourceURLRU)
}
func (c Class) Name(lang string) string    { return Pick(lang, c.NameEN, c.NameRU) }
func (c Class) Source(lang string) string  { return Pick(lang, c.SourceURL, c.SourceURLRU) }
func (s Subclass) Name(lang string) string { return Pick(lang, s.NameEN, s.NameRU) }
func (s Subclass) Source(lang string) string {
	return Pick(lang, s.SourceURL, s.SourceURLRU)
}
func (f Feature) Name(lang string) string   { return Pick(lang, f.NameEN, f.NameRU) }
func (f Feature) Source(lang string) string { return Pick(lang, f.SourceURL, f.SourceURLRU) }

// StatFeature is a generic one-stat item feature (atk bonus, crit, PHB property, …).
// Stored in catalog_stat_features — not the class/race table catalog_features.
type StatFeature struct {
	ID           int64
	Slug         string
	NameEN       string
	NameRU       string
	Stat         string
	DefaultValue string
	Origin       string
	SourceURL    string
	SortOrder    int
}

func (f StatFeature) Name(lang string) string { return Pick(lang, f.NameEN, f.NameRU) }

// Item is mundane or magic gear (mechanical fields + SRD desc + source links).
type Item struct {
	ID                 int64
	Slug               string
	NameEN             string
	NameRU             string
	Kind               string
	CostGP             float64
	WeightLB           float64
	DamageDice         string
	DamageType         string
	ArmorClass         string
	Properties         []string
	Rarity             string
	SourceURL          string
	SourceURLRU        string
	DescEN             string
	DescRU             string
	ArmorCategory      string
	ACBase             int
	DexMax             int
	StealthDisadv      bool
	StrMin             int
	WeaponCategory     string
	VersatileDice      string
	RangeNormal        int
	RangeLong          int
	SuggestedSlot      string
	RequiresAttunement bool
	Consumable         bool
	ChargesMax         int
	IsStub             bool
	IsBase             bool
}

func (i Item) Name(lang string) string   { return Pick(lang, i.NameEN, i.NameRU) }
func (i Item) Source(lang string) string { return Pick(lang, i.SourceURL, i.SourceURLRU) }

// Source5e14 is the canonical 5e14 article URL when genitems matched an index card.
func (i Item) Source5e14() string { return i.SourceURLRU }

func (i Item) Desc(lang string) string { return Pick(lang, i.DescEN, i.DescRU) }

func (i Item) HasProperty(p string) bool {
	p = strings.ToLower(p)
	for _, x := range i.Properties {
		if strings.ToLower(x) == p {
			return true
		}
	}
	return false
}

func (i Item) IsMagic() bool {
	return i.IsStub || i.Rarity != ""
}

func NormalizeRarity(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func (i Item) IsSpellScroll() bool {
	slug := strings.ToLower(i.Slug)
	name := strings.ToLower(i.NameEN)
	return strings.Contains(slug, "spell-scroll") || strings.Contains(name, "spell scroll")
}

func (i Item) IsJewelrySlotBase() bool {
	return i.IsBase && i.IsJewelry()
}

func (i Item) LootMagicEligible() bool {
	if i.IsBase || i.IsJewelrySlotBase() || i.IsSpellScroll() {
		return false
	}
	switch NormalizeRarity(i.Rarity) {
	case "uncommon", "rare", "very rare", "legendary", "artifact":
		return true
	default:
		return false
	}
}

func (i Item) IsShield() bool {
	return strings.EqualFold(i.ArmorCategory, "shield") || i.HasProperty("shield") || i.SuggestedSlot == "shield"
}

func (i Item) IsTwoHanded() bool {
	return i.HasProperty("two-handed")
}

func (i Item) IsRangedWeapon() bool {
	if arm, ok := PHBArmBySlug(i.Slug); ok {
		return arm.Ranged
	}
	return i.HasProperty("ammunition")
}

func (i Item) IsArmor() bool {
	if i.Kind == "weapon" || i.Kind == "jewelry" || i.Consumable {
		return false
	}
	return i.Kind == "armor" || i.ArmorCategory != "" || i.IsShield()
}

func (i Item) IsJewelry() bool {
	return i.Kind == "jewelry"
}

func (i Item) IsWeaponBase() bool {
	if !i.IsBase || i.IsArmor() || i.IsJewelry() {
		return false
	}
	return i.Kind == "weapon" || i.WeaponCategory != "" || i.DamageDice != ""
}

func (i Item) BuilderKind() string {
	switch {
	case i.Consumable:
		return "consumable"
	case i.IsJewelry():
		return "jewelry"
	case i.IsArmor():
		return "armor"
	default:
		return "weapon"
	}
}

func (i Item) BuilderListPath() string {
	switch i.BuilderKind() {
	case "armor":
		return "armor"
	case "jewelry":
		return "jewelry"
	default:
		return "weapons"
	}
}

func (i Item) Stackable() bool {
	if i.IsStub || (i.Rarity != "" && !i.Consumable) {
		return false
	}
	return i.Consumable || i.Rarity == ""
}

// StatsLine is the catalog subtitle, e.g. "1d8 slashing · versatile 1d10".
func (i Item) StatsLine() string {
	var parts []string
	if i.ArmorClass != "" {
		parts = append(parts, "AC "+i.ArmorClass)
	}
	if i.DamageDice != "" {
		s := i.DamageDice
		if i.DamageType != "" {
			s += " " + i.DamageType
		}
		parts = append(parts, s)
	}
	if i.VersatileDice != "" {
		parts = append(parts, "versatile "+i.VersatileDice)
	}
	if i.Rarity != "" {
		parts = append(parts, i.Rarity)
	}
	if i.RequiresAttunement {
		parts = append(parts, "attunement")
	}
	if i.IsStub {
		parts = append(parts, "stub")
	}
	out := ""
	for n, p := range parts {
		if n > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

// Spell is a later spell-picker row: stats + class list + source links, not lore text.
type Spell struct {
	ID                int64
	Slug              string
	NameEN            string
	NameRU            string
	Level             int
	School            string
	Ritual            bool
	CastingTime       string
	Range             string
	Duration          string
	Components        string
	Classes           []string
	SourceURL         string
	SourceURLRU       string
	DamageFormula     string
	DamageType        string
	HealFormula       string
	ScaleKind         string
	Upcast            bool
	Concentration     bool
	DamageAtSlot      map[string]string
	DamageAtCharacter map[string]string
}

func (s Spell) Name(lang string) string   { return Pick(lang, s.NameEN, s.NameRU) }
func (s Spell) Source(lang string) string { return Pick(lang, s.SourceURL, s.SourceURLRU) }

// FormulaAt picks the stored dice string for a slot or character level.
func (s Spell) FormulaAt(slotLevel, charLevel int) (formula, dmgType, heal string) {
	formula, dmgType, heal = s.DamageFormula, s.DamageType, s.HealFormula
	switch s.ScaleKind {
	case "character":
		if v := pickLevelMap(s.DamageAtCharacter, charLevel); v != "" {
			formula = v
		}
	case "slot":
		if v := pickLevelMap(s.DamageAtSlot, slotLevel); v != "" {
			if s.DamageFormula == "" && s.HealFormula != "" {
				heal = v
			} else {
				formula = v
			}
		}
	}
	return formula, dmgType, heal
}

// StatsLine is the sheet subtitle, e.g. "3d4 + 3 force".
func (s Spell) StatsLine(slotLevel, charLevel int) string {
	f, dt, heal := s.FormulaAt(slotLevel, charLevel)
	var parts []string
	if f != "" {
		if dt != "" {
			parts = append(parts, f+" "+dt)
		} else {
			parts = append(parts, f)
		}
	}
	if heal != "" {
		parts = append(parts, heal)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

// Monster is a fightable 5e 2014 stat block (HP/AC/CR/DEX + resist snapshot).
// 5e14-only cards are stubs (IsStub): name + URL + CR/type from the index, no HP.
type Monster struct {
	ID                  int64
	Slug                string
	NameEN              string
	NameRU              string
	Size                string
	Type                string
	ArmorClass          int
	HitPoints           int
	HitDice             string
	Speed               string
	Dexterity           int
	Strength            int
	Constitution        int
	Intelligence        int
	Wisdom              int
	Charisma            int
	SaveProficiencies   []string
	CR                  float64
	CRLabel             string
	XP                  int
	Resistances         []string
	Immunities          []string
	Vulnerabilities     []string
	ConditionImmunities []string
	ActionsJSON         string
	DescEN              string
	SourceURL           string
	SourceURLRU         string
	IsStub              bool
}

func (m Monster) Name(lang string) string   { return Pick(lang, m.NameEN, m.NameRU) }
func (m Monster) Source(lang string) string { return Pick(lang, m.SourceURL, m.SourceURLRU) }
func (m Monster) Source5e14() string        { return m.SourceURLRU }

// ArticleURL is the 5e14 bestiary page when present, else the 5eapi source_url.
func (m Monster) ArticleURL() string {
	if u := strings.TrimSpace(m.SourceURLRU); u != "" {
		return u
	}
	return strings.TrimSpace(m.SourceURL)
}

func (m Monster) ResistLine() string {
	var parts []string
	if len(m.Resistances) > 0 {
		parts = append(parts, "resist "+strings.Join(m.Resistances, ", "))
	}
	if len(m.Immunities) > 0 {
		parts = append(parts, "immune "+strings.Join(m.Immunities, ", "))
	}
	if len(m.Vulnerabilities) > 0 {
		parts = append(parts, "vuln "+strings.Join(m.Vulnerabilities, ", "))
	}
	return strings.Join(parts, " · ")
}

func (m Monster) StatsLine() string {
	if m.IsStub {
		s := "stub"
		if m.CRLabel != "" {
			s = "CR " + m.CRLabel + " · " + s
		}
		if m.Type != "" {
			s += " · " + m.Type
		}
		return s
	}
	s := fmt.Sprintf("CR %s · AC %d · HP %d", m.CRLabel, m.ArmorClass, m.HitPoints)
	if m.HitDice != "" {
		s += " (" + m.HitDice + ")"
	}
	return s
}

// FightHP is catalog HP, or 1 for stubs so the DM can add them and edit immediately.
func (m Monster) FightHP() int {
	if m.HitPoints > 0 {
		return m.HitPoints
	}
	return 1
}

func (m Monster) FightAC() int {
	if m.ArmorClass > 0 {
		return m.ArmorClass
	}
	return 10
}

func (m Monster) FightDEX() int {
	return fightScore(m.Dexterity)
}

func (m Monster) FightSTR() int { return fightScore(m.Strength) }
func (m Monster) FightCON() int { return fightScore(m.Constitution) }
func (m Monster) FightINT() int { return fightScore(m.Intelligence) }
func (m Monster) FightWIS() int { return fightScore(m.Wisdom) }
func (m Monster) FightCHA() int { return fightScore(m.Charisma) }

func fightScore(v int) int {
	if v > 0 {
		return v
	}
	return 10
}

// Condition is a PHB 2014 Appendix A condition (or a common combat overlay like burning).
type Condition struct {
	ID            int64
	Slug          string
	NameEN        string
	NameRU        string
	SourceURL     string
	SourceURLRU   string
	DamageFormula string
	DamageType    string
	IsPHB         bool
}

func (c Condition) Name(lang string) string { return Pick(lang, c.NameEN, c.NameRU) }

func (c Condition) Source(lang string) string {
	if c.SourceURLRU != "" {
		return c.SourceURLRU
	}
	return c.SourceURL
}

func pickLevelMap(m map[string]string, level int) string {
	if len(m) == 0 {
		return ""
	}
	bestK, bestV := -1, ""
	for k, v := range m {
		n := 0
		for _, r := range k {
			if r < '0' || r > '9' {
				n = -1
				break
			}
			n = n*10 + int(r-'0')
		}
		if n < 0 {
			continue
		}
		if n == level {
			return v
		}
		if n <= level && n > bestK {
			bestK, bestV = n, v
		}
	}
	if bestV != "" {
		return bestV
	}
	return ""
}
