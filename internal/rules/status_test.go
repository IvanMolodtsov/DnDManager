package rules

import "testing"

func TestVisibleEffectsOmitsHidden(t *testing.T) {
	list := []Effect{
		{Slug: "shield", NameEN: "Shield", ACBonus: 5},
		{Slug: "curse", NameEN: "Hex Curse", Hidden: true, ACBonus: -4},
	}
	vis := VisibleEffects(list, false)
	if len(vis) != 1 || vis[0].Slug != "shield" {
		t.Fatalf("visible %+v", vis)
	}
	dm := VisibleEffects(list, true)
	if len(dm) != 2 {
		t.Fatalf("dm %+v", dm)
	}
}

func TestHiddenStatusDoesNotChangeDerivedAC(t *testing.T) {
	base := DeriveCombat(CombatInput{Scores: AbilityScores{DEX: 10}, RaceSpeed: 30})
	hidden := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 10}, RaceSpeed: 30,
		Effects: []Effect{{Slug: "curse", Hidden: true, ACBonus: -4, NameEN: "Hex Curse"}},
	})
	if hidden.AC != base.AC {
		t.Fatalf("hidden ac %d want %d", hidden.AC, base.AC)
	}
	shown := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 10}, RaceSpeed: 30,
		Effects: []Effect{{Slug: "shield-of-faith", ACBonus: 2, NameEN: "Shield of Faith"}},
	})
	if shown.AC != base.AC+2 {
		t.Fatalf("visible ac %d", shown.AC)
	}
}

func TestEndTurnEffectsRemovesAtZero(t *testing.T) {
	list := []Effect{
		{Slug: "burn", DurationTurns: 2, DamageFormula: "1", DamageType: "fire"},
		{Slug: "perm", DurationTurns: 0, NameEN: "Curse"},
	}
	once := EndTurnEffects(list)
	if len(once) != 2 || once[0].DurationTurns != 1 {
		t.Fatalf("after first %+v", once)
	}
	twice := EndTurnEffects(once)
	if len(twice) != 1 || twice[0].Slug != "perm" {
		t.Fatalf("after second %+v", twice)
	}
}

func TestCustomStatusSlug(t *testing.T) {
	if CustomStatusSlug("Hex Curse") != "custom-hex-curse" {
		t.Fatalf("%s", CustomStatusSlug("Hex Curse"))
	}
	if CustomStatusSlug("Проклятие") != "custom-проклятие" {
		t.Fatalf("%s", CustomStatusSlug("Проклятие"))
	}
}
