// Package catalog is the local PHB 5e (2014) reference data.
// Russian source URLs follow https://5e14.dnd.su/{type}/{numericId}-{english-slug}/
// (type = class | race | backgrounds | spells | items). Subclass and feature rows
// link to the parent page when 5e14 has no dedicated URL.
package catalog

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

// Item is mundane or magic gear for a later inventory picker (mechanical fields + source links).
type Item struct {
	ID          int64
	Slug        string
	NameEN      string
	NameRU      string
	Kind        string
	CostGP      float64
	WeightLB    float64
	DamageDice  string
	DamageType  string
	ArmorClass  string
	Properties  []string
	Rarity      string
	SourceURL   string
	SourceURLRU string
}

func (i Item) Name(lang string) string   { return Pick(lang, i.NameEN, i.NameRU) }
func (i Item) Source(lang string) string { return Pick(lang, i.SourceURL, i.SourceURLRU) }

// Spell is a later spell-picker row: stats + class list + source links, not lore text.
type Spell struct {
	ID          int64
	Slug        string
	NameEN      string
	NameRU      string
	Level       int
	School      string
	Ritual      bool
	CastingTime string
	Range       string
	Duration    string
	Components  string
	Classes     []string
	SourceURL   string
	SourceURLRU string
}

func (s Spell) Name(lang string) string   { return Pick(lang, s.NameEN, s.NameRU) }
func (s Spell) Source(lang string) string { return Pick(lang, s.SourceURL, s.SourceURLRU) }
