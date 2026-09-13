package rules

import (
	"strings"
	"testing"

	"dndmanager/internal/catalog"
)

func TestDeriveMoonbladePlusThreeTwice(t *testing.T) {
	feats := []FeatureDTO{
		{Origin: OriginCommon, NameEN: "+3 attack", Stat: StatAtkBonus, Value: "3"},
		{Origin: OriginCommon, NameEN: "+3 damage", Stat: StatDmgBonus, Value: "3"},
		{Origin: OriginMagical, NameEN: "Vorpal Sword", Stat: StatAtkBonus, Value: "3"},
		{Origin: OriginMagical, NameEN: "Vorpal Sword", Stat: StatDmgBonus, Value: "3"},
		{Origin: OriginMagical, NameEN: "Vorpal Sword", Stat: StatNote, Value: "Decapitate on a 20"},
	}
	d := DeriveItemStats("1d8", "slashing", feats)
	if d.DamageLine != "1d8+6 slashing" {
		t.Fatalf("damage %q", d.DamageLine)
	}
	if d.AtkBonus != 6 {
		t.Fatalf("atk %d", d.AtkBonus)
	}
	if d.RollFormula != "1d8+6" {
		t.Fatalf("roll %q", d.RollFormula)
	}
}

func TestCatalogLongswordFeatures(t *testing.T) {
	item := catalog.Item{
		Slug: "longsword", NameEN: "Longsword", Kind: "weapon", IsBase: true,
		DamageDice: "1d8", DamageType: "slashing", Properties: []string{"versatile"},
		WeaponCategory: "martial", VersatileDice: "1d10",
	}
	feats := CatalogItemFeatures(item, nil)
	var hasDmg, hasVers, hasMartial bool
	for _, f := range feats {
		if f.Stat == StatAtkBonus || f.Stat == StatDmgBonus || strings.Contains(strings.ToLower(f.NameEN), "base damage") {
			t.Fatalf("base hit must not be a feature: %+v", f)
		}
		if f.Stat == StatVersatile && f.Value == "1d10" && strings.Contains(strings.ToLower(f.NameEN), "versatile") {
			hasVers = true
		}
		if f.Stat == StatProperty && f.Value == "martial" {
			hasMartial = true
		}
		if f.PropertySlug() == "universal" || strings.EqualFold(f.NameEN, "universal") {
			t.Fatalf("do not invent universal: %+v", f)
		}
	}
	_ = hasDmg
	if !hasVers || !hasMartial {
		t.Fatalf("longsword features %+v", feats)
	}
}

func TestMoonbladeInheritsLongsword(t *testing.T) {
	moon := catalog.Item{Slug: "moonblade", NameEN: "Moonblade", Kind: "weapon", IsStub: true}
	long := catalog.Item{
		Slug: "longsword", DamageDice: "1d8", DamageType: "slashing",
		Properties: []string{"versatile"}, WeaponCategory: "martial", VersatileDice: "1d10", IsBase: true,
	}
	if InheritWeaponSlug(moon.Slug) != "longsword" {
		t.Fatal("moonblade should inherit longsword")
	}
	feats := CatalogItemFeatures(moon, &long)
	for _, f := range feats {
		if strings.Contains(strings.ToLower(f.NameEN), "base damage") {
			t.Fatalf("no base-damage feature: %+v", f)
		}
	}
	d := DeriveItemStats(long.DamageDice, long.DamageType, feats)
	if d.DamageLine != "1d8 slashing" {
		t.Fatalf("inherited %q", d.DamageLine)
	}
}

func TestMagicWeaponBonusVorpal(t *testing.T) {
	desc := "You gain a +3 bonus to attack and damage rolls made with this magic weapon. In addition, the weapon ignores resistance to slashing damage.\nWhen you attack a creature that has at least one head with this weapon and roll a 20 on the attack roll, you cut off one of the creature's heads."
	item := catalog.Item{Slug: "vorpal-sword", NameEN: "Vorpal Sword", NameRU: "Меч головоруб", DescEN: desc}
	feats := FeaturesFromItem(item)
	var atk, dmg bool
	note := false
	if len(feats) < 3 {
		t.Fatalf("vorpal should unpack into several features: %+v", feats)
	}
	for _, f := range feats {
		if f.Stat == StatAtkBonus && f.Value == "3" {
			atk = true
		}
		if f.Stat == StatDmgBonus && f.Value == "3" {
			dmg = true
		}
		if f.Stat == StatNote && strings.Contains(strings.ToLower(f.Value), "decapitate") {
			note = true
		}
	}
	if !atk || !dmg || !note {
		t.Fatalf("vorpal unpack %+v", feats)
	}
}

func TestUnknownURLIsNote(t *testing.T) {
	feats := FeaturesFromUnknownURL("https://example.com/mystery")
	if len(feats) != 1 || feats[0].Stat != StatNote || feats[0].SourceURL == "" {
		t.Fatalf("%+v", feats)
	}
}

func TestCommonPlusNTwoFeatures(t *testing.T) {
	feats := CommonPlusN(3)
	if len(feats) != 2 {
		t.Fatalf("want 2 features, got %+v", feats)
	}
	d := DeriveItemStats("1d8", "slashing", feats)
	if d.DamageLine != "1d8+3 slashing" || d.AtkBonus != 3 {
		t.Fatalf("%+v", d)
	}
}

func TestCommonPlusAC(t *testing.T) {
	feats := CommonPlusAC(1)
	if len(feats) != 1 || feats[0].Stat != StatACBonus || feats[0].Value != "1" {
		t.Fatalf("%+v", feats)
	}
	d := DeriveItemStats("", "", feats)
	if d.ACBonus != 1 || !strings.Contains(d.Line, "+1 AC") {
		t.Fatalf("%+v", d)
	}
}

func TestParseAbilityValue(t *testing.T) {
	key, n := ParseAbilityValue("str:2")
	if key != "str" || n != 2 {
		t.Fatalf("%s %d", key, n)
	}
	key, n = ParseAbilityValue("dex-1")
	if key != "dex" || n != -1 {
		t.Fatalf("%s %d", key, n)
	}
}

func TestGrantsFromFeatures(t *testing.T) {
	g := GrantsFromFeatures([]FeatureDTO{
		{Stat: StatSkillProficiency, Value: "stealth"},
		{Stat: StatGrantSpell, Value: "42", CatalogSlug: "grant-spell-3"},
		{Stat: StatResistance, Value: "fire"},
		{Stat: StatAbilityBonus, Value: "str:2"},
	})
	if !g.HasSkill("stealth") || len(g.SpellIDs) != 1 || g.SpellIDs[0] != 42 {
		t.Fatalf("%+v", g)
	}
	if g.Ability.STR != 2 {
		t.Fatalf("str %+v", g.Ability)
	}
	if !strings.Contains(g.DefenseLine(), "fire") {
		t.Fatalf("line %s", g.DefenseLine())
	}
}

func TestCatalogLeatherFeatures(t *testing.T) {
	item := catalog.Item{
		Slug: "leather-armor", NameEN: "Leather Armor", Kind: "armor", IsBase: true,
		ArmorCategory: "light", ACBase: 11, DexMax: -1,
	}
	feats := CatalogItemFeatures(item, nil)
	var hasAC, hasLight bool
	for _, f := range feats {
		if f.Stat == StatACBase && f.Value == "11" {
			hasAC = true
		}
		if f.Stat == StatProperty && f.Value == "light" {
			hasLight = true
		}
	}
	if !hasAC || !hasLight {
		t.Fatalf("%+v", feats)
	}
}

func TestParseDND14URL(t *testing.T) {
	ref, ok := ParseDND14URL("https://5e14.dnd.su/items/142-vorpal-sword/")
	if !ok || ref.ID != 142 || ref.Slug != "vorpal-sword" || ref.Type != "items" {
		t.Fatalf("%+v ok=%v", ref, ok)
	}
	ref, ok = ParseDND14URL("https://5e14.dnd.su/item/2231-moonblade")
	if !ok || ref.ID != 2231 || ref.Type != "items" {
		t.Fatalf("item alias %+v", ref)
	}
	ref, ok = ParseDND14URL("https://5e14.dnd.su/spells/123-remove-curse/")
	if !ok || ref.Type != "spells" || ref.Slug != "remove-curse" {
		t.Fatalf("spell %+v", ref)
	}
	if _, ok := ParseDND14URL("https://5e14.dnd.su/item/moonblade/"); ok {
		t.Fatal("invented slug path must not parse as id-slug")
	}
}
