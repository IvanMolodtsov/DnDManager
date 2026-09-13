package rules

import "testing"

func TestSlotConflicts(t *testing.T) {
	leather := GearPiece{Slot: SlotArmor, Name: "Leather Armor", ArmorCategory: "light", ACBase: 11, DexMax: -1}
	if err := CheckEquip(nil, leather); err != nil {
		t.Fatal(err)
	}
	err := CheckEquip([]GearPiece{leather}, GearPiece{Slot: SlotArmor, Name: "Chain Mail", ArmorCategory: "heavy", ACBase: 16})
	occ, ok := err.(*SlotOccupiedError)
	if !ok || occ.OtherName != "Leather Armor" {
		t.Fatalf("second armor: %v", err)
	}

	greatsword := GearPiece{Slot: SlotMainHand, Name: "Greatsword", TwoHanded: true}
	shield := GearPiece{Slot: SlotShield, Name: "Shield", IsShield: true, ACBase: 2}
	err = CheckEquip([]GearPiece{greatsword}, shield)
	if err == nil {
		t.Fatal("two-handed vs shield")
	}
	if occ, ok = err.(*SlotOccupiedError); !ok || occ.OtherName != "Greatsword" {
		t.Fatalf("got %v", err)
	}

	r1 := GearPiece{Slot: SlotRing1, Name: "Ring A"}
	r2 := GearPiece{Slot: SlotRing2, Name: "Ring B"}
	if err := CheckEquip([]GearPiece{r1}, r2); err != nil {
		t.Fatal(err)
	}
	occMap := OccupiedMap([]GearPiece{r1, r2})
	if _, ok := PickRingSlot(occMap); ok {
		t.Fatal("both rings taken")
	}

	need := GearPiece{Slot: SlotNeck, Name: "Amulet", RequiresAttune: true, Attuned: true}
	cur := []GearPiece{
		{Slot: SlotRing1, Name: "R1", Attuned: true},
		{Slot: SlotRing2, Name: "R2", Attuned: true},
		{Slot: SlotCloak, Name: "Cloak", Attuned: true},
	}
	if err := CheckEquip(cur, need); err != ErrAttunementFull {
		t.Fatalf("attune cap %v", err)
	}
}

func TestLeatherShieldAC(t *testing.T) {
	st := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 14},
		Gear: []GearPiece{
			{Slot: SlotArmor, Name: "Leather", ArmorCategory: "light", ACBase: 11, DexMax: -1},
			{Slot: SlotShield, Name: "Shield", IsShield: true, ACBase: 2},
		},
	})
	if st.AC != 15 { // 11+2 DEX +2 shield
		t.Fatalf("leather+shield ac %d", st.AC)
	}
}

func TestMonkUDOffWhenArmored(t *testing.T) {
	base := CombatInput{
		Scores:    AbilityScores{DEX: 16, WIS: 14, CON: 12},
		Classes:   []ClassLevel{{Slug: "monk", Levels: 2}},
		RaceSpeed: 30,
	}
	unarmored := DeriveCombat(base)
	if unarmored.AC != 15 || unarmored.Speed != 40 {
		t.Fatalf("unarmored %+v", unarmored)
	}
	armored := DeriveCombat(CombatInput{
		Scores: base.Scores, Classes: base.Classes, RaceSpeed: 30,
		Gear: []GearPiece{{Slot: SlotArmor, ArmorCategory: "light", ACBase: 11, DexMax: -1}},
	})
	if armored.AC != 14 { // 11+3, no UD
		t.Fatalf("armored monk ac %d", armored.AC)
	}
	if armored.Speed != 30 {
		t.Fatalf("armored monk speed %d", armored.Speed)
	}
}

func TestMageArmorIgnoredWhileWearingArmor(t *testing.T) {
	st := DeriveCombat(CombatInput{
		Scores: AbilityScores{DEX: 14},
		Gear:   []GearPiece{{Slot: SlotArmor, ArmorCategory: "light", ACBase: 11, DexMax: -1}},
		Effects: []Effect{
			{Slug: "mage-armor", ACBase: 13},
		},
	})
	if st.AC != 13 { // 11+2, mage 13+2 ignored as base
		t.Fatalf("mage vs leather ac %d", st.AC)
	}
}

func TestBarbarianUDWithShield(t *testing.T) {
	st := DeriveCombat(CombatInput{
		Scores:  AbilityScores{DEX: 14, CON: 16},
		Classes: []ClassLevel{{Slug: "barbarian", Levels: 1}},
		Gear:    []GearPiece{{Slot: SlotShield, IsShield: true, ACBase: 2}},
	})
	if st.AC != 17 { // 10+2+3 +2 shield
		t.Fatalf("barb+shield %d", st.AC)
	}
}
