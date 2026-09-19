package characters

import (
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

type Mutation struct {
	ID          int64
	CharacterID int64
	BodyPart    string
	MonsterID   int64
	NameEN      string
	NameRU      string
	Size        string
	Type        string
	CR          float64
	CRLabel     string
	SourceURL   string
	SourceURLRU string
	Features    []rules.FeatureDTO
}

func (m Mutation) Name(lang string) string {
	return catalog.Pick(lang, m.NameEN, m.NameRU)
}

func (m Mutation) ArticleURL() string {
	if u := strings.TrimSpace(m.SourceURLRU); u != "" {
		return u
	}
	return strings.TrimSpace(m.SourceURL)
}

func (m Mutation) Attacks() []rules.MutationAttack {
	return rules.GrantsFromFeatures(m.Features).Attacks
}

func (m Mutation) FeatureLine() string {
	var parts []string
	for _, f := range m.Features {
		if f.Stat == "" {
			continue
		}
		line := f.Stat
		if v := strings.TrimSpace(f.Value); v != "" {
			line += " " + v
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " · ")
}

func (c *Character) MutationGrants() rules.ItemGrants {
	var g rules.ItemGrants
	for _, m := range c.Mutations {
		g = rules.MergeGrants(g, rules.GrantsFromFeatures(m.Features))
	}
	return g
}

func (c *Character) AllGrants() rules.ItemGrants {
	return rules.MergeGrants(c.EquippedGrants(), c.MutationGrants())
}

func (c *Character) SkillCheckBonus(slug string) int {
	return rules.SkillBonus(c.Scores(), c.Level, c.EffectiveSkillMark(slug)) + c.AllGrants().SkillBonusOf(slug)
}

func (c *Character) HasDeadStatus() bool {
	return rules.HasDead(c.Effects) || c.DeathFail >= 3
}
