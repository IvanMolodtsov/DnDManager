package characters

import (
	"encoding/json"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

const (
	KindBeast    = catalog.CompanionBeast
	KindFamiliar = catalog.CompanionFamiliar
	KindItem     = catalog.CompanionItem
	KindWeapon   = catalog.CompanionWeapon
)

// Companion is a persisted sheet creature (not a character row).
type Companion struct {
	ID               int64
	CharacterID      int64
	Kind             string
	CatalogMonsterID int64
	ItemID           int64
	Name             string
	NameEN           string
	NameRU           string
	Size             string
	Type             string
	AC               int
	HPCurrent        int
	HPMax            int
	STR              int
	DEX              int
	CON              int
	INT              int
	WIS              int
	CHA              int
	ResistJSON       string
	SourceURL        string
	SourceURLRU      string
	CanAttack        bool
	SortOrder        int
}

func (c Companion) DisplayName(lang string) string {
	if strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	return catalog.Pick(lang, c.NameEN, c.NameRU)
}

func (c Companion) ArticleURL() string {
	if u := strings.TrimSpace(c.SourceURLRU); u != "" {
		return u
	}
	return strings.TrimSpace(c.SourceURL)
}

func (c Companion) Scores() rules.AbilityScores {
	return rules.AbilityScores{STR: c.STR, DEX: c.DEX, CON: c.CON, INT: c.INT, WIS: c.WIS, CHA: c.CHA}
}

func (c *Companion) SetScores(s rules.AbilityScores) {
	if c == nil {
		return
	}
	c.STR, c.DEX, c.CON, c.INT, c.WIS, c.CHA = s.STR, s.DEX, s.CON, s.INT, s.WIS, s.CHA
}

func (c Companion) Score(key string) int { return c.Scores().Get(key) }

func (c Companion) Resist() battlesResist {
	return parseCompanionResist(c.ResistJSON)
}

type battlesResist struct {
	Resistances     []string
	Immunities      []string
	Vulnerabilities []string
}

func parseCompanionResist(raw string) battlesResist {
	s := struct {
		Resistances     []string `json:"resistances"`
		Immunities      []string `json:"immunities"`
		Vulnerabilities []string `json:"vulnerabilities"`
	}{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &s)
	}
	return battlesResist{Resistances: s.Resistances, Immunities: s.Immunities, Vulnerabilities: s.Vulnerabilities}
}

func (c Companion) HasResist(slug string) bool { return sliceHasFold(c.Resist().Resistances, slug) }
func (c Companion) HasImmune(slug string) bool { return sliceHasFold(c.Resist().Immunities, slug) }
func (c Companion) HasVuln(slug string) bool   { return sliceHasFold(c.Resist().Vulnerabilities, slug) }

func (c Companion) SaveHint(key string) string {
	n := rules.Modifier(c.Scores().Get(key))
	if n >= 0 {
		return "+" + itoa(n)
	}
	return itoa(n)
}

// CompanionGate says which Add pickers the sheet may show.
type CompanionGate struct {
	BeastMaster bool
	Familiar    bool
	Chain       bool
	Item        bool
	Weapon      bool
	Conjure     []string
}

func (g CompanionGate) AnyPersist() bool {
	return g.BeastMaster || g.Familiar || g.Item || g.Weapon
}

func (g CompanionGate) CanAdd(kind string) bool {
	switch kind {
	case KindBeast:
		return g.BeastMaster
	case KindFamiliar:
		return g.Familiar
	case KindItem:
		return g.Item
	case KindWeapon:
		return g.Weapon
	default:
		return false
	}
}

func GateCompanions(ch *Character) CompanionGate {
	g := CompanionGate{}
	if ch == nil {
		return g
	}
	for _, cl := range ch.ClassLevels {
		if cl.Slug == "ranger" && cl.Levels >= 3 && cl.SubclassSlug == "beast-master" {
			g.BeastMaster = true
		}
	}
	g.Chain = hasFeatureSlug(ch, "pact-of-the-chain")
	if g.Chain || hasSpellAccess(ch, "find-familiar") {
		g.Familiar = true
	}
	for _, it := range ch.Inventory {
		if !it.Equipped() {
			continue
		}
		if hasGrantCompanion(it) || catalog.ItemGrantsCompanion(it.Item.Slug, it.Item.Kind) {
			g.Item = true
			if it.Item.Kind == "weapon" || catalog.ItemIsWeaponCompanion(it.Item.Slug, it.Item.Kind) {
				g.Weapon = true
			}
		}
	}
	for _, slug := range catalog.ConjureSpellSlugs() {
		if hasSpellAccess(ch, slug) {
			g.Conjure = append(g.Conjure, slug)
		}
	}
	return g
}

func hasFeatureSlug(ch *Character, slug string) bool {
	for _, f := range ch.Features {
		if f.Slug == slug {
			return true
		}
	}
	return false
}

func hasSpellAccess(ch *Character, slug string) bool {
	for _, ls := range ch.Spells {
		if ls.Spell.Slug == slug {
			return true
		}
	}
	for _, g := range ch.GrantedSpells {
		if g.Spell.Slug == slug {
			return true
		}
	}
	return false
}

func hasGrantCompanion(it InventoryItem) bool {
	for _, f := range it.Features {
		if f.Stat == rules.StatGrantCompanion {
			return true
		}
	}
	return false
}

func (c *Character) CompanionCount(kind string) int {
	n := 0
	for _, row := range c.Companions {
		if row.Kind == kind {
			n++
		}
	}
	return n
}

func sliceHasFold(list []string, slug string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, s := range list {
		if strings.ToLower(strings.TrimSpace(s)) == slug {
			return true
		}
	}
	return false
}
