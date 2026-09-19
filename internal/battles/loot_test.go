package battles

import (
	"math/rand"
	"testing"

	"dndmanager/internal/catalog"
)

func TestParseCR(t *testing.T) {
	if got := ParseCR("1/4", 0); got != 0.25 {
		t.Fatalf("1/4 got %v", got)
	}
	if got := ParseCR("1/8", 0); got != 0.125 {
		t.Fatalf("1/8 got %v", got)
	}
	if got := ParseCR("1/2", 0); got != 0.5 {
		t.Fatalf("1/2 got %v", got)
	}
	if got := ParseCR("0", 0); got != 0 {
		t.Fatalf("0 got %v", got)
	}
	if got := ParseCR("12", 0); got != 12 {
		t.Fatalf("12 got %v", got)
	}
	if got := ParseCR("", 18); got != 18 {
		t.Fatalf("numeric fallback %v", got)
	}
}

func TestSoulsForCR(t *testing.T) {
	if got := SoulsForCR(0.25); got != 12 {
		t.Fatalf("CR 1/4 souls %d", got)
	}
	if got := SoulsForCR(0); got != 0 {
		t.Fatalf("CR 0 souls %d", got)
	}
	if got := SoulsForCR(12); got != 600 {
		t.Fatalf("CR 12 souls %d", got)
	}
	if got := SoulsForCR(1); got != 50 {
		t.Fatalf("CR 1 souls %d", got)
	}
}

func TestDropTableBands(t *testing.T) {
	t0 := DropTableForCR(0)
	if t0.UncommonPct != 10 || t0.RarePct != 5 || t0.GoldMin != 10 || t0.GoldMax != 100 || t0.TreasurePct != 5 || t0.TreasureCount != 1 || t0.ScrollCount != 0 {
		t.Fatalf("CR 0 %+v", t0)
	}
	t10 := DropTableForCR(10)
	if t10.UncommonPct != 10 || t10.GoldMax != 100 {
		t.Fatalf("CR 10 should match 0–10 %+v", t10)
	}
	t12 := DropTableForCR(12)
	if t12.RarePct != 10 || t12.VeryRarePct != 10 || t12.GoldMin != 100 || t12.GoldMax != 500 || t12.ScrollPct != 15 || t12.ScrollCount != 1 || t12.TreasureCount != 1 {
		t.Fatalf("CR 12 %+v", t12)
	}
	t18 := DropTableForCR(18)
	if t18.VeryRarePct != 10 || t18.LegendaryPct != 5 || t18.GoldMin != 500 || t18.GoldMax != 1000 || t18.ScrollPct != 25 || t18.TreasureCount != 2 {
		t.Fatalf("CR 18 %+v", t18)
	}
	t22 := DropTableForCR(22)
	if t22.LegendaryPct != 10 || t22.ArtifactPct != 5 || t22.GoldMin != 1000 || t22.GoldMax != 2000 || t22.ScrollPct != 35 || t22.TreasureCount != 3 {
		t.Fatalf("CR 22 %+v", t22)
	}
	t26 := DropTableForCR(26)
	if t26.LegendaryPct != 15 || t26.ArtifactPct != 10 || t26.GoldMin != 1500 || t26.GoldMax != 2500 || t26.ScrollPct != 40 || t26.TreasurePct != 20 || t26.ScrollCount != 1 || t26.TreasureCount != 3 {
		t.Fatalf("CR 26 %+v", t26)
	}
	t28 := DropTableForCR(28)
	if t28.LegendaryPct != 25 || t28.ArtifactPct != 20 || t28.GoldMin != 2500 || t28.GoldMax != 3500 || t28.ScrollPct != 50 || t28.TreasurePct != 30 || t28.ScrollCount != 2 || t28.TreasureCount != 4 {
		t.Fatalf("CR 28 %+v", t28)
	}
}

func TestDeadLootMonstersSkipOthers(t *testing.T) {
	units := []Unit{
		{Kind: KindPC, Dead: true, CR: 10, CRLabel: "10"},
		{Kind: KindCompanion, Dead: true, CR: 10, CRLabel: "10"},
		{Kind: KindSummon, Dead: true, CR: 10, CRLabel: "10"},
		{Kind: KindMonster, Dead: true, CR: 0.25, CRLabel: "1/4"},
		{Kind: KindMonster, Dead: false, CR: 12, CRLabel: "12"},
		{Kind: KindMonster, Dead: true, Escaped: true, CR: 12, CRLabel: "12"},
	}
	got := DeadLootMonsters(units)
	if len(got) != 1 || got[0].CRLabel != "1/4" {
		t.Fatalf("dead loot %+v", got)
	}
	if TotalSouls(units) != 12 {
		t.Fatalf("souls %d", TotalSouls(units))
	}
}

func TestEmptyRarityPoolSkipped(t *testing.T) {
	src := catalogLootSource{magic: map[string][]catalog.Item{}, treasures: catalog.PHBTreasures()}
	rng := rand.New(rand.NewSource(1))
	rows := rollMagic(src, rng, "artifact", 100)
	if len(rows) != 0 {
		t.Fatalf("empty artifact pool leaked %+v", rows)
	}
	got := RollMonsterDrops(22, src, rand.New(rand.NewSource(42)))
	for _, row := range got.Rows {
		if row.Kind == LootMagic {
			t.Fatalf("magic drop with empty pools %+v", row)
		}
	}
	if got.Gold < 1000 || got.Gold > 2000 {
		t.Fatalf("CR 22 gold %d", got.Gold)
	}
}

func TestGenerateFromUnitsIndependentGold(t *testing.T) {
	units := []Unit{
		{Kind: KindMonster, Dead: true, CR: 0, CRLabel: "0"},
		{Kind: KindMonster, Dead: true, CR: 0, CRLabel: "0"},
	}
	src := catalogLootSource{magic: map[string][]catalog.Item{}, treasures: catalog.PHBTreasures()}
	got := GenerateFromUnits(units, src, rand.New(rand.NewSource(7)))
	if got.Gold < 20 || got.Gold > 200 {
		t.Fatalf("two CR0 gold %d", got.Gold)
	}
	if got.Souls != 0 {
		t.Fatalf("CR 0 souls %d", got.Souls)
	}
}

func TestScrollLevels(t *testing.T) {
	min, max := ScrollLevelsForCR(12)
	if min != 1 || max != 3 {
		t.Fatalf("CR 12 scroll %d-%d", min, max)
	}
	min, max = ScrollLevelsForCR(18)
	if min != 4 || max != 5 {
		t.Fatalf("CR 18 scroll %d-%d", min, max)
	}
	min, max = ScrollLevelsForCR(22)
	if min != 6 || max != 7 {
		t.Fatalf("CR 22 scroll %d-%d", min, max)
	}
	min, max = ScrollLevelsForCR(28)
	if min != 8 || max != 9 {
		t.Fatalf("CR 28 scroll %d-%d", min, max)
	}
}
