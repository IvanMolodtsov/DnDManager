// Package battles owns campaign encounters: monster setup, initiative, and the fight board.
package battles

import (
	"encoding/json"
	"strconv"
	"strings"

	"dndmanager/internal/rules"
)

const (
	StatusSetupMonsters = "setup_monsters"
	StatusSetupInit     = "setup_init"
	StatusFighting      = "fighting"
	StatusEnded         = "ended"

	KindPC      = "pc"
	KindMonster = "monster"
)

// Battle is one encounter for a campaign (unique until dismissed).
type Battle struct {
	ID          int64
	CampaignID  int64
	Status      string
	Round       int
	ActiveIndex int
	CreatedBy   int64
	Units       []Unit
}

func (b *Battle) Active() bool {
	return b != nil && b.Status != "" && b.Status != StatusEnded
}

func (b *Battle) Setup() bool {
	return b != nil && (b.Status == StatusSetupMonsters || b.Status == StatusSetupInit)
}

func (b *Battle) Fighting() bool {
	return b != nil && b.Status == StatusFighting
}

func (b *Battle) Ended() bool {
	return b != nil && b.Status == StatusEnded
}

func (b *Battle) ActiveUnit() *Unit {
	if b == nil {
		return nil
	}
	for i := range b.Units {
		if b.Units[i].SortOrder == b.ActiveIndex {
			return &b.Units[i]
		}
	}
	if len(b.Units) == 0 {
		return nil
	}
	if b.ActiveIndex >= 0 && b.ActiveIndex < len(b.Units) {
		return &b.Units[b.ActiveIndex]
	}
	return nil
}

// Unit is one creature in initiative order. 3 goblins → 3 rows.
type Unit struct {
	ID               int64
	BattleID         int64
	Kind             string
	CharacterID      int64
	CatalogMonsterID int64
	Name             string
	Initiative       int
	HPCurrent        int
	HPMax            int
	TempHP           int
	AC               int
	STR              int
	DEX              int
	CON              int
	INT              int
	WIS              int
	CHA              int
	DeathSuccess     int
	DeathFail        int
	Dead             bool
	Escaped          bool
	Knocked          bool
	SortOrder        int
	ResistJSON       string
	SourceURLRU      string
	Effects          []rules.Effect
}

// ArticleURL is the snapshotted 5e14 bestiary link (or source_url fallback stored there).
func (u Unit) ArticleURL() string {
	return strings.TrimSpace(u.SourceURLRU)
}

func (u Unit) IsPC() bool      { return u.Kind == KindPC }
func (u Unit) IsMonster() bool { return u.Kind == KindMonster }

func (u Unit) Able() bool {
	return !u.Dead && !u.Escaped
}

func (u Unit) Stable() bool {
	return u.IsPC() && u.Knocked && !u.Dead && u.DeathSuccess >= 3
}

func (u Unit) ShowDeathSaves() bool {
	return u.IsPC() && !u.Escaped && (u.Knocked || u.Dead || u.DeathSuccess > 0 || u.DeathFail > 0)
}

func (u Unit) SuccessPips() []bool {
	return []bool{u.DeathSuccess >= 1, u.DeathSuccess >= 2, u.DeathSuccess >= 3}
}

func (u Unit) FailPips() []bool {
	return []bool{u.DeathFail >= 1, u.DeathFail >= 2, u.DeathFail >= 3}
}

func (u Unit) Scores() rules.AbilityScores {
	return rules.AbilityScores{STR: u.STR, DEX: u.DEX, CON: u.CON, INT: u.INT, WIS: u.WIS, CHA: u.CHA}
}

func (u *Unit) SetScores(s rules.AbilityScores) {
	if u == nil {
		return
	}
	u.STR, u.DEX, u.CON, u.INT, u.WIS, u.CHA = s.STR, s.DEX, s.CON, s.INT, s.WIS, s.CHA
}

func (u Unit) Grants() rules.ItemGrants {
	return ParseSnapshot(u.ResistJSON).Grants()
}

func (u Unit) ResistLine() string {
	return ParseSnapshot(u.ResistJSON).Line()
}

func (u Unit) ScoreLine() string {
	return formatScoreLine(u.Scores())
}

func (u Unit) SaveHint(key string) string {
	return formatMod(rules.Modifier(u.Scores().Get(key)))
}

func (u Unit) DisplayAC() int {
	if u.IsPC() {
		return u.AC
	}
	ac := u.AC
	for _, e := range rules.DeriveEffects(u.Effects) {
		ac += e.ACBonus
		if e.ACFloor > ac {
			ac = e.ACFloor
		}
	}
	return ac
}

func (u Unit) BoardEffects(isDM bool) []rules.Effect {
	if u.IsMonster() && !isDM {
		return nil
	}
	return rules.VisibleEffects(u.Effects, isDM)
}

// MonsterGroup is the wizard list of types with counts.
type MonsterGroup struct {
	CatalogID       int64
	Name            string
	Count           int
	CRLabel         string
	HPCurrent       int
	HPMax           int
	AC              int
	STR             int
	DEX             int
	CON             int
	INT             int
	WIS             int
	CHA             int
	Resistances     []string
	Immunities      []string
	Vulnerabilities []string
	Saves           []string
	SourceURLRU     string
}

func (g MonsterGroup) ArticleURL() string {
	return strings.TrimSpace(g.SourceURLRU)
}

func (g MonsterGroup) Scores() rules.AbilityScores {
	return rules.AbilityScores{STR: g.STR, DEX: g.DEX, CON: g.CON, INT: g.INT, WIS: g.WIS, CHA: g.CHA}
}

func (g *MonsterGroup) SetScores(s rules.AbilityScores) {
	if g == nil {
		return
	}
	g.STR, g.DEX, g.CON, g.INT, g.WIS, g.CHA = s.STR, s.DEX, s.CON, s.INT, s.WIS, s.CHA
}

func (g MonsterGroup) Score(key string) int { return g.Scores().Get(key) }

func (g MonsterGroup) HasResist(slug string) bool { return sliceHas(g.Resistances, slug) }
func (g MonsterGroup) HasImmune(slug string) bool { return sliceHas(g.Immunities, slug) }
func (g MonsterGroup) HasVuln(slug string) bool   { return sliceHas(g.Vulnerabilities, slug) }

func (g MonsterGroup) SaveHint(key string) string {
	return formatMod(rules.Modifier(g.Scores().Get(key)))
}

func (g *MonsterGroup) ApplyStats(st GroupStats) {
	if g == nil {
		return
	}
	g.HPCurrent, g.HPMax, g.AC = st.HPCurrent, st.HPMax, st.AC
	g.SetScores(st.Scores)
	g.Resistances = st.Resist.Resistances
	g.Immunities = st.Resist.Immunities
	g.Vulnerabilities = st.Resist.Vulnerabilities
	if len(st.Resist.Saves) > 0 {
		g.Saves = st.Resist.Saves
	}
}

// GroupStats is the DM group-edit payload (every copy of that catalog type).
type GroupStats struct {
	HPCurrent int
	HPMax     int
	AC        int
	Scores    rules.AbilityScores
	Resist    ResistSnapshot
}

// Stats is the end-of-fight tally (no loot).
type Stats struct {
	PCAlive        int
	PCDead         int
	PCEscaped      int
	MonsterAlive   int
	MonsterDead    int
	MonsterEscaped int
	Units          []Unit
}

func (b *Battle) ComputeStats() Stats {
	st := Stats{}
	if b == nil {
		return st
	}
	st.Units = b.Units
	for _, u := range b.Units {
		switch {
		case u.Escaped:
			if u.IsPC() {
				st.PCEscaped++
			} else {
				st.MonsterEscaped++
			}
		case u.Dead:
			if u.IsPC() {
				st.PCDead++
			} else {
				st.MonsterDead++
			}
		default:
			if u.IsPC() {
				st.PCAlive++
			} else {
				st.MonsterAlive++
			}
		}
	}
	return st
}

type ResistSnapshot struct {
	Resistances     []string `json:"resistances"`
	Immunities      []string `json:"immunities"`
	Vulnerabilities []string `json:"vulnerabilities"`
	Saves           []string `json:"saves,omitempty"`
}

func (s ResistSnapshot) Grants() rules.ItemGrants {
	return rules.ItemGrants{
		Resistances:     s.Resistances,
		Immunities:      s.Immunities,
		Vulnerabilities: s.Vulnerabilities,
	}
}

func (s ResistSnapshot) Line() string {
	g := s.Grants()
	var parts []string
	if len(g.Resistances) > 0 {
		parts = append(parts, "resist "+strings.Join(g.Resistances, ", "))
	}
	if len(g.Immunities) > 0 {
		parts = append(parts, "immune "+strings.Join(g.Immunities, ", "))
	}
	if len(g.Vulnerabilities) > 0 {
		parts = append(parts, "vuln "+strings.Join(g.Vulnerabilities, ", "))
	}
	return strings.Join(parts, " · ")
}

func ParseSnapshot(raw string) ResistSnapshot {
	s := ResistSnapshot{}
	if strings.TrimSpace(raw) == "" {
		return s
	}
	_ = json.Unmarshal([]byte(raw), &s)
	return s
}

func EncodeSnapshot(s ResistSnapshot) string {
	s.Resistances = nonNil(s.Resistances)
	s.Immunities = nonNil(s.Immunities)
	s.Vulnerabilities = nonNil(s.Vulnerabilities)
	b, err := json.Marshal(s)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func sliceHas(list []string, slug string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return false
	}
	for _, s := range list {
		if strings.ToLower(strings.TrimSpace(s)) == slug {
			return true
		}
	}
	return false
}

func formatScoreLine(s rules.AbilityScores) string {
	return "STR " + strconv.Itoa(s.STR) + " DEX " + strconv.Itoa(s.DEX) + " CON " + strconv.Itoa(s.CON) +
		" INT " + strconv.Itoa(s.INT) + " WIS " + strconv.Itoa(s.WIS) + " CHA " + strconv.Itoa(s.CHA)
}

func formatMod(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
