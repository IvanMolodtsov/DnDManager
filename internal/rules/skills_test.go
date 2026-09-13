package rules

import "testing"

func TestSkillBonusProficiencyAndExpertise(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10} // +3 STR
	plain := SkillBonus(scores, 1, SkillMark{Slug: "athletics"})
	if plain != 3 {
		t.Fatalf("untrained athletics: %d", plain)
	}
	prof := SkillBonus(scores, 1, SkillMark{Slug: "athletics", Proficient: true})
	if prof != 5 {
		t.Fatalf("proficient L1: %d", prof)
	}
	exp := SkillBonus(scores, 1, SkillMark{Slug: "athletics", Proficient: true, Expertise: true})
	if exp != 7 {
		t.Fatalf("expertise L1: %d", exp)
	}
	// ASI to 18 (+4) and total level 5 (PB +3)
	scores.STR = 18
	after := SkillBonus(scores, 5, SkillMark{Slug: "athletics", Proficient: true})
	if after != 7 {
		t.Fatalf("after ASI and PB bump: %d", after)
	}
}

func TestSaveBonusFighter(t *testing.T) {
	scores := AbilityScores{STR: 15, CON: 14, DEX: 10, INT: 10, WIS: 10, CHA: 8}
	str := SaveBonus(scores, 1, SaveMark{Ability: "str", Proficient: true})
	if str != 2+2 {
		t.Fatalf("STR save: %d", str)
	}
	cha := SaveBonus(scores, 1, SaveMark{Ability: "cha", Proficient: false})
	if cha != -1 {
		t.Fatalf("CHA save: %d", cha)
	}
}

func TestGrants(t *testing.T) {
	sk := GrantedSkillSlugs("high-elf", "acolyte")
	if !contains(sk, "perception") || !contains(sk, "insight") || !contains(sk, "religion") {
		t.Fatalf("grants: %v", sk)
	}
	sv := GrantedSaveAbilities([]string{"fighter", "wizard"})
	if !contains(sv, "str") || !contains(sv, "con") || !contains(sv, "int") || !contains(sv, "wis") {
		t.Fatalf("saves: %v", sv)
	}
}

func TestCheckFormula(t *testing.T) {
	if CheckFormula(4) != "1d20+4" {
		t.Fatal(CheckFormula(4))
	}
	if CheckFormula(-1) != "1d20-1" {
		t.Fatal(CheckFormula(-1))
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
