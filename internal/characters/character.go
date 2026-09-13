// Package characters owns live sheets, creation drafts, the wizard, and level-up.
package characters

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func (c *Character) Scores() rules.AbilityScores {
	return rules.AbilityScores{STR: c.STR, DEX: c.DEX, CON: c.CON, INT: c.INT, WIS: c.WIS, CHA: c.CHA}
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
	}
}

// State is the snapshot the rules engine uses for previews.
func (c *Character) State() rules.State {
	ids := make([]int64, 0, len(c.Features))
	for _, f := range c.Features {
		ids = append(ids, f.ID)
	}
	return rules.State{
		Level:            c.Level,
		Scores:           c.Scores(),
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
