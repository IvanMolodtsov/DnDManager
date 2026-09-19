package battles

import (
	"math"
	"math/rand"
	"strconv"
	"strings"

	"dndmanager/internal/catalog"
)

const (
	LootGold     = "gold"
	LootSouls    = "souls"
	LootMagic    = "magic"
	LootScroll   = "scroll"
	LootTreasure = "treasure"
	LootQuest    = "quest"
	LootItem     = "item"
)

// LootRow is one line on the battle statistics loot table.
type LootRow struct {
	ID             int64
	BattleID       int64
	Kind           string
	NameEN         string
	NameRU         string
	CatalogItemID  int64
	CatalogSpellID int64
	Qty            int
	Rarity         string
	Notes          string
	SortOrder      int
}

func (l LootRow) Name(lang string) string {
	return catalog.Pick(lang, l.NameEN, l.NameRU)
}

func (l LootRow) Generated() bool {
	switch l.Kind {
	case LootGold, LootMagic, LootScroll, LootTreasure:
		return true
	default:
		return false
	}
}

func (l LootRow) IsPile() bool {
	return l.Kind == LootGold || l.Kind == LootSouls
}

// DropTable is Ivan's per-monster drop chances for one CR band (plus over-25 bumps).
type DropTable struct {
	UncommonPct   int
	RarePct       int
	VeryRarePct   int
	LegendaryPct  int
	ArtifactPct   int
	GoldMin       int
	GoldMax       int
	ScrollPct     int
	ScrollCount   int
	TreasurePct   int
	TreasureCount int
}

// ParseCR turns a catalog label (1/4, 1/8, 12) into a float. Numeric is a fallback.
func ParseCR(label string, numeric float64) float64 {
	s := strings.TrimSpace(label)
	if s != "" {
		if n, ok := parseCRLabel(s); ok {
			return n
		}
	}
	if numeric > 0 {
		return numeric
	}
	if s == "0" || numeric == 0 {
		return 0
	}
	return 0
}

func parseCRLabel(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if i := strings.IndexByte(s, '/'); i > 0 {
		num, err1 := strconv.ParseFloat(strings.TrimSpace(s[:i]), 64)
		den, err2 := strconv.ParseFloat(strings.TrimSpace(s[i+1:]), 64)
		if err1 != nil || err2 != nil || den == 0 {
			return 0, false
		}
		return num / den, true
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (u Unit) Challenge() float64 {
	return ParseCR(u.CRLabel, u.CR)
}

func DropTableForCR(cr float64) DropTable {
	var t DropTable
	switch {
	case cr <= 10:
		t = DropTable{
			UncommonPct: 10, RarePct: 5,
			GoldMin: 10, GoldMax: 100,
			TreasurePct: 5, TreasureCount: 1,
		}
	case cr <= 15:
		t = DropTable{
			RarePct: 10, VeryRarePct: 10,
			GoldMin: 100, GoldMax: 500,
			ScrollPct: 15, ScrollCount: 1,
			TreasurePct: 15, TreasureCount: 1,
		}
	case cr <= 20:
		t = DropTable{
			VeryRarePct: 10, LegendaryPct: 5,
			GoldMin: 500, GoldMax: 1000,
			ScrollPct: 25, ScrollCount: 1,
			TreasurePct: 15, TreasureCount: 2,
		}
	default:
		t = DropTable{
			LegendaryPct: 10, ArtifactPct: 5,
			GoldMin: 1000, GoldMax: 2000,
			ScrollPct: 35, ScrollCount: 1,
			TreasurePct: 15, TreasureCount: 3,
		}
		if cr > 25 {
			extra := int(math.Floor(cr)) - 25
			if extra < 0 {
				extra = 0
			}
			bump := extra * 5
			t.UncommonPct += bump
			t.RarePct += bump
			t.VeryRarePct += bump
			t.LegendaryPct += bump
			t.ArtifactPct += bump
			t.ScrollPct += bump
			t.TreasurePct += bump
			t.GoldMin += extra * 500
			t.GoldMax += extra * 500
			slots := extra / 3
			t.ScrollCount += slots
			t.TreasureCount += slots
		}
	}
	return t
}

func SoulsForCR(cr float64) int {
	if cr <= 0 {
		return 0
	}
	return int(math.Floor(cr * 50))
}

func DeadLootMonsters(units []Unit) []Unit {
	var out []Unit
	for _, u := range units {
		if u.Kind == KindMonster && u.Dead && !u.Escaped {
			out = append(out, u)
		}
	}
	return out
}

func TotalSouls(units []Unit) int {
	n := 0
	for _, u := range DeadLootMonsters(units) {
		n += SoulsForCR(u.Challenge())
	}
	return n
}

func ScrollLevelsForCR(cr float64) (min, max int) {
	switch {
	case cr <= 10:
		return 0, -1
	case cr <= 15:
		return 1, 3
	case cr <= 20:
		return 4, 5
	case cr <= 25:
		return 6, 7
	default:
		return 8, 9
	}
}

// LootSource picks catalog items, spells, and encoded treasures for a drop roll.
type LootSource interface {
	MagicByRarity(rarity string) []catalog.Item
	SpellsInRange(min, max int) []catalog.Spell
	Treasures() []catalog.PHBTreasure
}

type GeneratedLoot struct {
	Gold  int
	Souls int
	Rows  []LootRow
}

func GenerateFromUnits(units []Unit, src LootSource, rng *rand.Rand) GeneratedLoot {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	out := GeneratedLoot{}
	for _, u := range DeadLootMonsters(units) {
		got := RollMonsterDrops(u.Challenge(), src, rng)
		out.Gold += got.Gold
		out.Souls += got.Souls
		out.Rows = append(out.Rows, got.Rows...)
	}
	return out
}

func RollMonsterDrops(cr float64, src LootSource, rng *rand.Rand) GeneratedLoot {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	t := DropTableForCR(cr)
	out := GeneratedLoot{Gold: rollRange(rng, t.GoldMin, t.GoldMax), Souls: SoulsForCR(cr)}
	out.Rows = append(out.Rows, rollMagic(src, rng, "uncommon", t.UncommonPct)...)
	out.Rows = append(out.Rows, rollMagic(src, rng, "rare", t.RarePct)...)
	out.Rows = append(out.Rows, rollMagic(src, rng, "very rare", t.VeryRarePct)...)
	out.Rows = append(out.Rows, rollMagic(src, rng, "legendary", t.LegendaryPct)...)
	out.Rows = append(out.Rows, rollMagic(src, rng, "artifact", t.ArtifactPct)...)
	minL, maxL := ScrollLevelsForCR(cr)
	for i := 0; i < t.ScrollCount; i++ {
		if !chance(rng, t.ScrollPct) {
			continue
		}
		if row, ok := rollScroll(src, rng, minL, maxL); ok {
			out.Rows = append(out.Rows, row)
		}
	}
	for i := 0; i < t.TreasureCount; i++ {
		if !chance(rng, t.TreasurePct) {
			continue
		}
		if row, ok := rollTreasure(src, rng); ok {
			out.Rows = append(out.Rows, row)
		}
	}
	return out
}

func rollMagic(src LootSource, rng *rand.Rand, rarity string, pct int) []LootRow {
	if !chance(rng, pct) || src == nil {
		return nil
	}
	pool := src.MagicByRarity(rarity)
	if len(pool) == 0 {
		return nil
	}
	it := pool[rng.Intn(len(pool))]
	return []LootRow{{
		Kind: LootMagic, NameEN: it.NameEN, NameRU: it.NameRU,
		CatalogItemID: it.ID, Qty: 1, Rarity: catalog.NormalizeRarity(it.Rarity),
	}}
}

func rollScroll(src LootSource, rng *rand.Rand, min, max int) (LootRow, bool) {
	if src == nil {
		return LootRow{}, false
	}
	pool := src.SpellsInRange(min, max)
	if len(pool) == 0 {
		pool = src.SpellsInRange(0, 9)
	}
	if len(pool) == 0 {
		return LootRow{}, false
	}
	sp := pool[rng.Intn(len(pool))]
	ruName := catalog.Pick("ru", sp.NameEN, sp.NameRU)
	return LootRow{
		Kind: LootScroll, NameEN: "Scroll of " + sp.NameEN, NameRU: "Свиток: " + ruName,
		CatalogSpellID: sp.ID, Qty: 1,
	}, true
}

func rollTreasure(src LootSource, rng *rand.Rand) (LootRow, bool) {
	if src == nil {
		return LootRow{}, false
	}
	pool := src.Treasures()
	if len(pool) == 0 {
		return LootRow{}, false
	}
	t := pool[rng.Intn(len(pool))]
	return LootRow{
		Kind: LootTreasure, NameEN: t.NameEN, NameRU: t.NameRU, Qty: 1,
		Notes: t.Kind,
	}, true
}

func chance(rng *rand.Rand, pct int) bool {
	if pct <= 0 {
		return false
	}
	if pct >= 100 {
		return true
	}
	return rng.Intn(100) < pct
}

func rollRange(rng *rand.Rand, min, max int) int {
	if max < min {
		max = min
	}
	if min < 0 {
		min = 0
	}
	if max < 0 {
		return 0
	}
	return min + rng.Intn(max-min+1)
}

type catalogLootSource struct {
	magic     map[string][]catalog.Item
	spells    []catalog.Spell
	treasures []catalog.PHBTreasure
}

func (s catalogLootSource) MagicByRarity(rarity string) []catalog.Item {
	return s.magic[catalog.NormalizeRarity(rarity)]
}

func (s catalogLootSource) SpellsInRange(min, max int) []catalog.Spell {
	if max < min {
		return nil
	}
	var out []catalog.Spell
	for _, sp := range s.spells {
		if sp.Level >= min && sp.Level <= max {
			out = append(out, sp)
		}
	}
	return out
}

func (s catalogLootSource) Treasures() []catalog.PHBTreasure {
	return s.treasures
}
