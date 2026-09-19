package catalog

import "strings"

const (
	CompanionBeast    = "beast"
	CompanionFamiliar = "familiar"
	CompanionItem     = "item"
	CompanionWeapon   = "weapon"
)

// BeastMasterCRCap is PHB 2014 Ranger's Companion: CR 1/4 or lower.
const BeastMasterCRCap = 0.25

var familiarPHB = []string{
	"bat", "cat", "crab", "frog", "hawk", "lizard", "octopus", "owl",
	"poisonous-snake", "quipper", "rat", "raven", "sea-horse", "spider", "weasel",
}

var familiarChain = []string{"imp", "pseudodragon", "quasit", "sprite"}

var conjureSpellTypes = map[string]string{
	"conjure-animals":           "beast",
	"conjure-woodland-beings":   "fey",
	"conjure-minor-elementals":  "elemental",
	"conjure-elemental":         "elemental",
	"conjure-celestial":         "celestial",
}

var sizeRank = map[string]int{
	"tiny": 1, "small": 2, "medium": 3, "large": 4, "huge": 5, "gargantuan": 6,
}

// FamiliarSlugs is the PHB Find Familiar list, plus Pact of the Chain extras when chain is true.
func FamiliarSlugs(chain bool) []string {
	out := append([]string{}, familiarPHB...)
	if chain {
		out = append(out, familiarChain...)
	}
	return out
}

func IsChainFamiliar(slug string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, s := range familiarChain {
		if s == slug {
			return true
		}
	}
	return false
}

func IsFamiliarSlug(slug string, chain bool) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, s := range FamiliarSlugs(chain) {
		if s == slug {
			return true
		}
	}
	return false
}

func ConjureType(spellSlug string) string {
	return conjureSpellTypes[strings.ToLower(strings.TrimSpace(spellSlug))]
}

func ConjureSpellSlugs() []string {
	out := make([]string, 0, len(conjureSpellTypes))
	for slug := range conjureSpellTypes {
		out = append(out, slug)
	}
	return out
}

func IsMediumOrSmaller(size string) bool {
	n := sizeRank[strings.ToLower(strings.TrimSpace(size))]
	return n > 0 && n <= 3
}

func BeastMasterEligible(m Monster) bool {
	if !strings.EqualFold(strings.TrimSpace(m.Type), "beast") {
		return false
	}
	if m.CR > BeastMasterCRCap+0.0001 {
		return false
	}
	return IsMediumOrSmaller(m.Size)
}

// ItemGrantsCompanion reports whether an equipped catalog item is a known companion source
// (figurines of wondrous power, dancing sword). Manual of Golems is a crafting book, not a pet.
func ItemGrantsCompanion(slug, kind string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if strings.HasPrefix(slug, "figurine-of-wondrous-power") {
		return true
	}
	if slug == "dancing-sword" {
		return true
	}
	return false
}

func ItemIsWeaponCompanion(slug, kind string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	return slug == "dancing-sword" || kind == "weapon"
}
