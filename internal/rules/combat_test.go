package rules

import "testing"

func TestMonkUnarmoredACAndSpeed(t *testing.T) {
	st := DeriveCombat(CombatInput{
		Scores:    AbilityScores{DEX: 16, WIS: 14, CON: 12},
		Classes:   []ClassLevel{{Slug: "monk", Levels: 2}},
		RaceSpeed: 30,
		HPCurrent: 15, HPMax: 15,
	})
	if st.AC != 15 { // 10+3+2
		t.Fatalf("monk ac %d", st.AC)
	}
	if st.Speed != 40 {
		t.Fatalf("monk speed %d", st.Speed)
	}
}

func TestMageArmorAndSanctuary(t *testing.T) {
	st := DeriveCombat(CombatInput{
		Scores:    AbilityScores{DEX: 14},
		RaceSpeed: 25,
		Effects: []Effect{
			{Slug: "mage-armor", ACBase: 13, NameEN: "Mage Armor"},
			{Slug: "sanctuary", Tags: "sanctuary", NameEN: "Sanctuary"},
		},
	})
	if st.AC != 15 { // 13+2
		t.Fatalf("mage armor ac %d", st.AC)
	}
	if st.Speed != 25 {
		t.Fatalf("speed %d", st.Speed)
	}
	if !st.Effects[1].HasTag("sanctuary") {
		t.Fatal("sanctuary tag")
	}
}

func TestArmorOfAgathysTempAndHaste(t *testing.T) {
	e, ok := SpellCombatEffect("armor-of-agathys", "Armor of Agathys", "", 1, 10)
	if !ok || e.TempHP != 10 {
		t.Fatalf("%+v", e)
	}
	st := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 10}, RaceSpeed: 30, TempHP: 10,
		Effects: []Effect{
			e,
			{Slug: "haste", ACBonus: 2, SpeedMult: 2},
		},
	})
	if st.AC != 12 || st.Speed != 60 || st.TempHP != 10 {
		t.Fatalf("haste+agathys %+v", st)
	}
}

func TestStackedTempHPHouseRule(t *testing.T) {
	a, _ := SpellCombatEffect("armor-of-agathys", "Armor of Agathys", "", 1, 5)
	other := OtherTempEffect(8)
	if SumTempHP([]Effect{a, other}) != 13 {
		t.Fatalf("stack %d", SumTempHP([]Effect{a, other}))
	}
	st := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 10}, RaceSpeed: 30,
		Effects: []Effect{a, other},
	})
	if st.TempHP != 13 {
		t.Fatalf("derived temp %d", st.TempHP)
	}
	next := ReduceStackedTemp([]Effect{a, other}, 8)
	if SumTempHP(next) != 5 {
		t.Fatalf("after spend other first %d %+v", SumTempHP(next), next)
	}
}

func TestDeathPips(t *testing.T) {
	if ToggleDeathPip(0, 1) != 1 || ToggleDeathPip(2, 2) != 1 {
		t.Fatal("toggle")
	}
	if ClampHP(-3, 20) != 0 || ClampHP(99, 20) != 20 {
		t.Fatal("clamp hp")
	}
}
