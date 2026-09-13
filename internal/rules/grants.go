package rules

import "strings"

// NamedOption is a slug + localized label for slim pickers.
type NamedOption struct {
	Slug   string
	NameEN string
	NameRU string
}

func (o NamedOption) Name(lang string) string {
	if lang == "ru" && o.NameRU != "" {
		return o.NameRU
	}
	return o.NameEN
}

// DamageTypes is the PHB damage list plus disease (special immunity).
func DamageTypes() []NamedOption {
	return []NamedOption{
		{"acid", "Acid", "Кислота"},
		{"cold", "Cold", "Холод"},
		{"fire", "Fire", "Огонь"},
		{"force", "Force", "Силовое"},
		{"lightning", "Lightning", "Молния"},
		{"necrotic", "Necrotic", "Некротика"},
		{"poison", "Poison", "Яд"},
		{"psychic", "Psychic", "Психика"},
		{"radiant", "Radiant", "Излучение"},
		{"thunder", "Thunder", "Звук"},
		{"bludgeoning", "Bludgeoning", "Дробящий"},
		{"piercing", "Piercing", "Колющий"},
		{"slashing", "Slashing", "Рубящий"},
		{"disease", "Disease", "Болезнь"},
	}
}

func Conditions() []NamedOption {
	return []NamedOption{
		{"blinded", "Blinded", "Ослепление"},
		{"charmed", "Charmed", "Очарование"},
		{"deafened", "Deafened", "Глухота"},
		{"frightened", "Frightened", "Испуг"},
		{"grappled", "Grappled", "Схвачен"},
		{"incapacitated", "Incapacitated", "Недееспособен"},
		{"invisible", "Invisible", "Невидимость"},
		{"paralyzed", "Paralyzed", "Паралич"},
		{"petrified", "Petrified", "Окаменение"},
		{"poisoned", "Poisoned", "Отравление"},
		{"prone", "Prone", "Опрокинут"},
		{"restrained", "Restrained", "Опутан"},
		{"stunned", "Stunned", "Ошеломление"},
		{"unconscious", "Unconscious", "Бессознательность"},
		{"disease", "Disease", "Болезнь"},
		{"exhaustion", "Exhaustion", "Истощение"},
	}
}

// ItemGrants is the mechanical overlay derived from equipped item features.
type ItemGrants struct {
	Ability         AbilityScores
	Skills          []string
	Resistances     []string
	Immunities      []string
	Vulnerabilities []string
	Conditions      []string
	SpellIDs        []int64
}

func (g ItemGrants) HasSkill(slug string) bool {
	slug = strings.ToLower(slug)
	for _, s := range g.Skills {
		if s == slug {
			return true
		}
	}
	return false
}

func (g ItemGrants) DefenseLine() string {
	var parts []string
	if s := joinUnique(g.Resistances); s != "" {
		parts = append(parts, "resist "+s)
	}
	if s := joinUnique(g.Immunities); s != "" {
		parts = append(parts, "immune "+s)
	}
	if s := joinUnique(g.Vulnerabilities); s != "" {
		parts = append(parts, "vuln "+s)
	}
	if s := joinUnique(g.Conditions); s != "" {
		parts = append(parts, "cond. immune "+s)
	}
	return strings.Join(parts, " · ")
}

func GrantsFromFeatures(feats []FeatureDTO) ItemGrants {
	var g ItemGrants
	seenSkill := map[string]bool{}
	seenSpell := map[int64]bool{}
	for _, f := range feats {
		switch f.Stat {
		case StatAbilityBonus, StatAbilityPenalty:
			key, n := ParseAbilityValue(f.Value)
			if key != "" && n != 0 {
				g.Ability.AddKey(key, n)
			}
		case StatSkillProficiency:
			slug := strings.ToLower(strings.TrimSpace(f.Value))
			if slug != "" && !seenSkill[slug] {
				if _, ok := SkillBySlug(slug); ok {
					seenSkill[slug] = true
					g.Skills = append(g.Skills, slug)
				}
			}
		case StatResistance:
			g.Resistances = appendToken(g.Resistances, f.Value)
		case StatImmunity:
			g.Immunities = appendToken(g.Immunities, f.Value)
		case StatVulnerability:
			g.Vulnerabilities = appendToken(g.Vulnerabilities, f.Value)
		case StatConditionImmunity:
			g.Conditions = appendToken(g.Conditions, f.Value)
		case StatGrantSpell, StatGrantCantrip:
			if id := f.SpellID(); id > 0 && !seenSpell[id] {
				seenSpell[id] = true
				g.SpellIDs = append(g.SpellIDs, id)
			}
		}
	}
	return g
}

func MergeGrants(dst ItemGrants, src ItemGrants) ItemGrants {
	dst.Ability = dst.Ability.Add(src.Ability)
	dst.Skills = mergeUnique(dst.Skills, src.Skills)
	dst.Resistances = mergeUnique(dst.Resistances, src.Resistances)
	dst.Immunities = mergeUnique(dst.Immunities, src.Immunities)
	dst.Vulnerabilities = mergeUnique(dst.Vulnerabilities, src.Vulnerabilities)
	dst.Conditions = mergeUnique(dst.Conditions, src.Conditions)
	seen := map[int64]bool{}
	for _, id := range dst.SpellIDs {
		seen[id] = true
	}
	for _, id := range src.SpellIDs {
		if !seen[id] {
			dst.SpellIDs = append(dst.SpellIDs, id)
			seen[id] = true
		}
	}
	return dst
}

func appendToken(dst []string, raw string) []string {
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		dst = mergeUnique(dst, []string{p})
	}
	return dst
}

func mergeUnique(dst, extra []string) []string {
	seen := map[string]bool{}
	for _, s := range dst {
		seen[s] = true
	}
	for _, s := range extra {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		dst = append(dst, s)
	}
	return dst
}

func joinUnique(in []string) string {
	return strings.Join(mergeUnique(nil, in), ", ")
}
