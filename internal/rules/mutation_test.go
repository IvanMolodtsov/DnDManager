package rules

import "testing"

func TestMapSizeDie(t *testing.T) {
	want := []string{"tiny", "small", "medium", "large", "huge", "gargantuan"}
	for i, s := range want {
		if got := MapSizeDie(i + 1); got != s {
			t.Fatalf("face %d: %s", i+1, got)
		}
	}
	if MapSizeDie(0) != "" || MapSizeDie(7) != "" {
		t.Fatal("out of range")
	}
}

func TestMapTypeDieChoose(t *testing.T) {
	slug, choose := MapTypeDie(15)
	if slug != "" || !choose {
		t.Fatalf("face 15: %s %v", slug, choose)
	}
	slug, choose = MapTypeDie(1)
	if slug != "aberration" || choose {
		t.Fatalf("face 1: %s %v", slug, choose)
	}
	if PHBMonsterTypes[9] != "humanoid" {
		t.Fatalf("stable order %v", PHBMonsterTypes)
	}
}

func TestMapBodyDie(t *testing.T) {
	part, choose := MapBodyDie(6)
	if part != "" || !choose {
		t.Fatalf("face 6: %s %v", part, choose)
	}
	part, choose = MapBodyDie(1)
	if part != BodyHead || choose {
		t.Fatalf("face 1: %s %v", part, choose)
	}
}

func TestBumpCRUntilHit(t *testing.T) {
	hits := map[string]int{"5": 2}
	label, cr, ok := BumpCRUntilHit(1, "tiny", "ooze", func(crLabel, size, typ string) int {
		return hits[crLabel]
	})
	if !ok || cr != 5 || label != "5" {
		t.Fatalf("bump got %s %d %v", label, cr, ok)
	}
	_, _, ok = BumpCRUntilHit(0, "tiny", "ooze", func(string, string, string) int { return 0 })
	if ok {
		t.Fatal("cap should fail")
	}
}

func TestCRLabelFromInt(t *testing.T) {
	if CRLabelFromInt(0) != "0" || CRLabelFromInt(3) != "3" {
		t.Fatal("labels")
	}
}

func TestSkillBonusApplies(t *testing.T) {
	g := GrantsFromFeatures([]FeatureDTO{
		{Stat: StatSkillBonus, Value: "athletics:2"},
		{Stat: StatSkillPenalty, Value: "stealth:1"},
	})
	if g.SkillBonusOf("athletics") != 2 {
		t.Fatalf("athletics %d", g.SkillBonusOf("athletics"))
	}
	if g.SkillBonusOf("stealth") != -1 {
		t.Fatalf("stealth %d", g.SkillBonusOf("stealth"))
	}
}

func TestBlockSlotAndRemoveLegs(t *testing.T) {
	g := GrantsFromFeatures([]FeatureDTO{
		{Stat: StatBlockSlot, Value: SlotBoots},
		{Stat: StatRemovePart, Value: BodyLegs},
	})
	blocked := false
	for _, s := range g.BlockedSlots {
		if s == SlotBoots {
			blocked = true
		}
	}
	if !blocked {
		t.Fatalf("boots not blocked: %v", g.BlockedSlots)
	}
}
