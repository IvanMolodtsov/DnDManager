package rules

import (
	"strings"
	"testing"

	"dndmanager/internal/catalog"
)

func phbWeapon(t *testing.T, slug string) WeaponDTO {
	t.Helper()
	arm, ok := catalog.PHBArmBySlug(slug)
	if !ok {
		t.Fatalf("missing PHB arm %s", slug)
	}
	item := catalog.Item{
		Slug: arm.Slug, NameEN: arm.NameEN, Kind: "weapon", IsBase: true,
		DamageDice: arm.Dice, DamageType: arm.DamageType, WeaponCategory: arm.Category,
		Properties: arm.Properties, VersatileDice: arm.VersatileDice,
		RangeNormal: arm.RangeNormal, RangeLong: arm.RangeLong,
	}
	return *WeaponFromItem(item, arm.NameEN, CatalogItemFeatures(item, nil))
}

func TestWeaponProficientPHBLists(t *testing.T) {
	long := phbWeapon(t, "longsword")
	dagger := phbWeapon(t, "dagger")
	rapier := phbWeapon(t, "rapier")
	scimitar := phbWeapon(t, "scimitar")
	club := phbWeapon(t, "club")

	if !WeaponProficient([]string{"fighter"}, long.BaseSlug, long.Simple, long.Martial) {
		t.Fatal("fighter + longsword")
	}
	if WeaponProficient([]string{"wizard"}, long.BaseSlug, long.Simple, long.Martial) {
		t.Fatal("wizard + longsword should not be proficient")
	}
	if !WeaponProficient([]string{"wizard"}, dagger.BaseSlug, dagger.Simple, dagger.Martial) {
		t.Fatal("wizard + dagger")
	}
	if WeaponProficient([]string{"wizard"}, club.BaseSlug, club.Simple, club.Martial) {
		t.Fatal("wizard does not get all simple")
	}
	if !WeaponProficient([]string{"bard"}, rapier.BaseSlug, rapier.Simple, rapier.Martial) {
		t.Fatal("bard + rapier")
	}
	if !WeaponProficient([]string{"wizard", "fighter"}, long.BaseSlug, long.Simple, long.Martial) {
		t.Fatal("wizard/fighter multiclass + longsword")
	}
	if !WeaponProficient([]string{"druid"}, scimitar.BaseSlug, scimitar.Simple, scimitar.Martial) {
		t.Fatal("druid + scimitar")
	}
	if WeaponProficient([]string{"druid"}, long.BaseSlug, long.Simple, long.Martial) {
		t.Fatal("druid + longsword should not be proficient")
	}
}

func TestWeaponProficientMoonbladeInheritsLongsword(t *testing.T) {
	if !WeaponProficient([]string{"fighter"}, "moonblade", false, false) {
		t.Fatal("fighter moonblade via longsword inherit")
	}
	if !WeaponProficient([]string{"bard"}, "moonblade", false, false) {
		t.Fatal("bard moonblade via longsword extra")
	}
	if WeaponProficient([]string{"wizard"}, "moonblade", false, false) {
		t.Fatal("wizard moonblade")
	}
}

func TestDeriveWeaponAttackFighterLongsword(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "longsword"), false, false)
	if !atk.Proficient || atk.AbilityKey != "str" || atk.ToHit != 5 || atk.HitFormula != "1d20+5" {
		t.Fatalf("hit %+v", atk)
	}
	if atk.DamageFormula != "1d8+3" || atk.DamageLine != "1d8+3 slashing" {
		t.Fatalf("dmg %q %q", atk.DamageFormula, atk.DamageLine)
	}
	if atk.Line != "+5 to hit · 1d8+3 slashing" {
		t.Fatalf("line %q", atk.Line)
	}
}

func TestDeriveWeaponAttackWizardLongswordNoPB(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"wizard"}, phbWeapon(t, "longsword"), false, false)
	if atk.Proficient || atk.ToHit != 3 || atk.HitFormula != "1d20+3" {
		t.Fatalf("wizard longsword %+v", atk)
	}
}

func TestDeriveWeaponAttackWizardDaggerPB(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"wizard"}, phbWeapon(t, "dagger"), false, false)
	if !atk.Proficient || atk.ToHit != 5 {
		t.Fatalf("wizard dagger %+v", atk)
	}
}

func TestDeriveWeaponAttackBardRapier(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"bard"}, phbWeapon(t, "rapier"), false, false)
	if !atk.Proficient || atk.AbilityKey != "str" || atk.ToHit != 5 {
		t.Fatalf("bard rapier uses higher STR: %+v", atk)
	}
}

func TestDeriveWeaponAttackFinesseUsesHigherDEX(t *testing.T) {
	scores := AbilityScores{STR: 12, DEX: 16, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"bard"}, phbWeapon(t, "rapier"), false, false)
	if atk.AbilityKey != "dex" || atk.AbilityMod != 3 || atk.ToHit != 5 {
		t.Fatalf("finesse DEX %+v", atk)
	}
	if !atk.Finesse {
		t.Fatal("expected finesse")
	}
}

func TestDeriveWeaponAttackLongbowDEX(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	atk := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "longbow"), true, false)
	if atk.AbilityKey != "dex" || atk.ToHit != 4 || atk.DamageFormula != "1d8+2" {
		t.Fatalf("longbow %+v", atk)
	}
}

func TestDeriveWeaponAttackThrownHandaxeSTR(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 14, CON: 10, INT: 10, WIS: 10, CHA: 10}
	melee := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "handaxe"), false, false)
	thrown := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "handaxe"), false, true)
	if !melee.HasThrown || melee.AbilityKey != "str" || thrown.AbilityKey != "str" {
		t.Fatalf("handaxe melee=%+v thrown=%+v", melee, thrown)
	}
	if thrown.DamageFormula != "1d6+3" {
		t.Fatalf("handaxe thrown dmg %q", thrown.DamageFormula)
	}
}

func TestDeriveWeaponAttackVersatileTwoHanded(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10}
	one := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "longsword"), false, false)
	two := DeriveWeaponAttack(scores, 1, []string{"fighter"}, phbWeapon(t, "longsword"), true, false)
	if one.DamageDice != "1d8" || two.DamageDice != "1d10" {
		t.Fatalf("versatile 1H %q 2H %q", one.DamageDice, two.DamageDice)
	}
	if two.DamageFormula != "1d10+3" {
		t.Fatalf("2H formula %q", two.DamageFormula)
	}
}

func TestDeriveWeaponAttackThrownSpearUsesOneHandDice(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10}
	w := phbWeapon(t, "spear")
	twoH := DeriveWeaponAttack(scores, 1, []string{"fighter"}, w, true, false)
	thrown := DeriveWeaponAttack(scores, 1, []string{"fighter"}, w, true, true)
	if twoH.DamageDice != "1d8" {
		t.Fatalf("spear 2H %q", twoH.DamageDice)
	}
	if thrown.DamageDice != "1d6" || thrown.DamageFormula != "1d6+3" {
		t.Fatalf("thrown spear should stay 1H dice, got %q", thrown.DamageFormula)
	}
}

func TestDeriveWeaponAttackPlusOneAndExtraDice(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10}
	w := phbWeapon(t, "longsword")
	w.Features = append(w.Features, CommonPlusN(1)...)
	w.Features = append(w.Features, FeatureDTO{Stat: StatExtraDice, Value: "2d6 fire", Origin: OriginMagical})
	atk := DeriveWeaponAttack(scores, 1, []string{"fighter"}, w, false, false)
	if atk.ToHit != 6 || atk.HitFormula != "1d20+6" {
		t.Fatalf("plus 1 hit %+v", atk)
	}
	if atk.DamageFormula != "1d8+4" || atk.DamageLine != "1d8+4 slashing" {
		t.Fatalf("plus 1 dmg %q", atk.DamageLine)
	}
	if len(atk.Extra) != 1 || atk.Extra[0] != "2d6 fire" {
		t.Fatalf("extra %+v", atk.Extra)
	}
	if ExtraRollFormula(atk.Extra[0]) != "2d6" {
		t.Fatalf("extra roll %q", ExtraRollFormula(atk.Extra[0]))
	}
	if !strings.Contains(atk.Line, "+6 to hit") || !strings.Contains(atk.Line, "1d8+4 slashing") || !strings.Contains(atk.Line, "2d6 fire") {
		t.Fatalf("line %q", atk.Line)
	}
}

func TestDeriveWeaponAttackCatalogPropertiesWithoutFeatures(t *testing.T) {
	scores := AbilityScores{STR: 16, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10}
	w := phbWeapon(t, "longsword")
	w.Features = nil
	atk := DeriveWeaponAttack(scores, 1, []string{"fighter"}, w, true, false)
	if atk.DamageDice != "1d10" {
		t.Fatalf("versatile from catalog: %q", atk.DamageDice)
	}
	d := phbWeapon(t, "dagger")
	d.Features = nil
	thrown := DeriveWeaponAttack(scores, 1, []string{"wizard"}, d, false, true)
	if !thrown.HasThrown || !thrown.Throwing || !thrown.Finesse {
		t.Fatalf("dagger flags %+v", thrown)
	}
}
