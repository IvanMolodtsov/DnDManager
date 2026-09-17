package characters

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"dndmanager/internal/rules"
)

func TestPotionStackUseAndSpeed(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Alchemist", 2, 1)
	heal, err := svc.Catalog.ItemBySlug("potion-of-healing-common")
	if err != nil {
		heal, err = svc.Catalog.ItemBySlug("potion-of-healing")
		if err != nil {
			t.Fatal(err)
		}
	}
	speed, err := svc.Catalog.ItemBySlug("potion-of-speed")
	if err != nil {
		t.Fatal(err)
	}

	a, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: heal.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	b, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: heal.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("healing potions should stack: %d vs %d", a.ID, b.ID)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(a.ID)
	if !ok || got.Quantity != 2 {
		t.Fatalf("stacked qty %+v", got)
	}

	ch.HPCurrent = 5
	if err := svc.Repo.UpdateVitals(ch.ID, 5, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if err := svc.UseItem(ch, uid, got.ID); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPCurrent <= 5 {
		t.Fatalf("healing did not apply: hp %d", ch.HPCurrent)
	}
	got, ok = ch.InventoryByID(got.ID)
	if !ok || got.Quantity != 1 {
		t.Fatalf("after use %+v ok=%v", got, ok)
	}

	sp, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: speed.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UseItem(ch, uid, sp.ID); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	found := false
	for _, e := range ch.Effects {
		if e.Slug == "potion-of-speed" && e.Source == rules.SourcePotion && e.SpeedMult == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("speed status %+v", ch.Effects)
	}
	st := rules.DeriveCombat(ch.CombatInput())
	if st.Speed < 50 {
		t.Fatalf("hasted speed %d", st.Speed)
	}
}

func TestMoonbladeStubConfigure(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Elf", 5, 1)
	moon, err := svc.Catalog.ItemBySlug("moonblade")
	if err != nil || !moon.IsStub {
		t.Fatalf("moonblade %+v %v", moon, err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: moon.ID, AtkBonus: 1, Notes: "runes"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(it.ID)
	if !ok || got.AtkBonus != 1 || got.Notes != "runes" || got.Quantity != 1 {
		t.Fatalf("%+v", got)
	}
	_, err = svc.AddItem(ch, uid, AddItemInput{CatalogID: moon.ID, AtkBonus: 2})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	var moons int
	for _, row := range ch.Inventory {
		if row.CatalogID == moon.ID {
			moons++
		}
	}
	if moons != 2 {
		t.Fatalf("stubs must not stack: %d", moons)
	}
}

func TestEquipSecondArmorBlocked(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Tank", 4, 1)
	leather, err := svc.Catalog.ItemBySlug("leather-armor")
	if err != nil {
		t.Fatal(err)
	}
	chain, err := svc.Catalog.ItemBySlug("chain-mail")
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: leather.ID, EquipNow: true, EquipSlot: "armor"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	st := rules.DeriveCombat(ch.CombatInput())
	if st.AC != 13 { // 11+DEX(14→+2)
		t.Fatalf("leather ac %d", st.AC)
	}
	b, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: chain.ID})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.EquipItem(ch, uid, b.ID, "armor")
	if !errors.Is(err, rules.ErrSlotOccupied) {
		t.Fatalf("expected occupied, got %v", err)
	}
	var occ *rules.SlotOccupiedError
	if !errors.As(err, &occ) || occ.OtherName == "" {
		t.Fatalf("named conflict %v", err)
	}

	ch, _ = svc.Get(ch.ID)
	before := len(ch.Inventory)
	_, err = svc.AddItem(ch, uid, AddItemInput{CatalogID: chain.ID, EquipNow: true, EquipSlot: "armor"})
	if !errors.Is(err, rules.ErrSlotOccupied) {
		t.Fatalf("equip-now occupied, got %v", err)
	}
	ch, _ = svc.Get(ch.ID)
	if len(ch.Inventory) != before {
		t.Fatalf("failed equip-now must not insert: %d -> %d", before, len(ch.Inventory))
	}
	_ = a
}

func TestMoonbladeFeaturesDeriveFormula(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Elf", 5, 1)
	long, err := svc.Catalog.ItemBySlug("longsword")
	if err != nil {
		t.Fatal(err)
	}
	vorpal, err := svc.Catalog.ItemBySlug("vorpal-sword")
	if err != nil {
		t.Fatal(err)
	}
	feats := svc.catalogFeatures(*long)
	feats = append(feats, rules.CommonPlusN(3)...)
	feats = append(feats, rules.FeatureDTO{Origin: rules.OriginUnique, NameEN: "Improved critical", NameRU: "Улучшенный крит", Stat: rules.StatCritRange, Value: "19"})
	feats = append(feats, rules.FeaturesFromItem(*vorpal)...)
	feats = append(feats, rules.FeatureDTO{
		Origin: rules.OriginUnique, NameEN: "Personality", NameRU: "Личность",
		Stat: rules.StatPersonality, Value: "INT 12, WIS 10, CHA 12",
	})
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: long.ID, CustomName: "Moonblade", Features: feats, Notes: "runes"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(it.ID)
	if !ok {
		t.Fatal("missing moonblade")
	}
	if got.Item.Slug != "longsword" {
		t.Fatalf("builder item should point at base weapon, got %s", got.Item.Slug)
	}
	d := got.DerivedStats()
	if d.DamageLine != "1d8+6 slashing" {
		t.Fatalf("damage %q atk %d feats=%d", d.DamageLine, d.AtkBonus, len(got.Features))
	}
	if d.AtkBonus != 6 {
		t.Fatalf("atk %d", d.AtkBonus)
	}
	if d.CritRange != 19 {
		t.Fatalf("crit %d", d.CritRange)
	}
	sentient := false
	for _, f := range got.Features {
		if f.Stat == rules.StatPersonality && strings.Contains(f.Value, "12") {
			sentient = true
		}
	}
	if !sentient {
		t.Fatalf("sentient not persisted %+v", got.Features)
	}
	if !strings.Contains(itemRowStats(got), "1d8+6 slashing") {
		t.Fatalf("sheet stats %q", itemRowStats(got))
	}
}

func TestCommonLongswordPlusThree(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Fighter", 3, 1)
	long, err := svc.Catalog.ItemBySlug("longsword")
	if err != nil {
		t.Fatal(err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: long.ID, PlusN: 3})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(it.ID)
	if !ok {
		t.Fatal("missing longsword")
	}
	d := got.DerivedStats()
	if d.DamageLine != "1d8+3 slashing" || d.AtkBonus != 3 {
		t.Fatalf("common +3: %q atk %d", d.DamageLine, d.AtkBonus)
	}
	if len(got.Features) != 2 {
		t.Fatalf("want two +N features, got %+v", got.Features)
	}
	for _, f := range got.Features {
		if f.Stat != rules.StatAtkBonus && f.Stat != rules.StatDmgBonus {
			t.Fatalf("unexpected feature %+v", f)
		}
	}
}

func TestCommonBreastplatePlusOne(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Tank", 5, 1)
	bp, err := svc.Catalog.ItemBySlug("breastplate")
	if err != nil {
		t.Fatal(err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: bp.ID, PlusN: 1, EquipNow: true, EquipSlot: "armor"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(it.ID)
	if !ok {
		t.Fatal("missing breastplate")
	}
	if got.Armor == nil {
		t.Fatalf("expected ArmorDTO, weapon=%v", got.Weapon != nil)
	}
	if len(got.Features) != 1 || got.Features[0].Stat != rules.StatACBonus || got.Features[0].Value != "1" {
		t.Fatalf("ac_bonus %+v", got.Features)
	}
	if got.DerivedStats().ACBonus != 1 {
		t.Fatalf("derived ac %d", got.DerivedStats().ACBonus)
	}
	st := rules.DeriveCombat(ch.CombatInput())
	if st.AC != 17 { // 14 + DEX cap 2 + 1
		t.Fatalf("breastplate +1 ac %d", st.AC)
	}
}

func TestJewelryRingSlots(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Rings", 5, 1)
	ring, err := svc.Catalog.ItemBySlug("ring")
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: ring.ID, EquipNow: true, EquipSlot: "ring", CustomName: "Ring A"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, _ := ch.InventoryByID(a.ID)
	if got.EquippedSlot != rules.SlotRing1 || got.Jewelry == nil {
		t.Fatalf("first ring %+v", got)
	}
	b, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: ring.ID, EquipNow: true, EquipSlot: "ring", CustomName: "Ring B"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, _ = ch.InventoryByID(b.ID)
	if got.EquippedSlot != rules.SlotRing2 {
		t.Fatalf("second ring slot %s", got.EquippedSlot)
	}
	_, err = svc.AddItem(ch, uid, AddItemInput{CatalogID: ring.ID, EquipNow: true, EquipSlot: "ring", CustomName: "Ring C"})
	if !errors.Is(err, rules.ErrSlotOccupied) {
		t.Fatalf("third ring %v", err)
	}
}

func TestItemGrantSpellAndSkill(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Fighter", 1, 1)
	rc, err := svc.Catalog.SpellBySlug("remove-curse")
	if err != nil {
		t.Fatal(err)
	}
	ring, err := svc.Catalog.ItemBySlug("ring")
	if err != nil {
		t.Fatal(err)
	}
	feats := []rules.FeatureDTO{
		{Stat: rules.StatGrantSpell, Value: strconv.FormatInt(rc.ID, 10), CatalogSlug: "grant-spell-3", Origin: rules.OriginMagical, NameEN: "Cast a 3rd-level spell"},
		{Stat: rules.StatSkillProficiency, Value: "stealth", Origin: rules.OriginMagical, NameEN: "Skill proficiency"},
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: ring.ID, Features: feats, EquipNow: true, EquipSlot: "ring", CustomName: "Curse Ring"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if len(ch.Spells) != 0 {
		t.Fatalf("must not persist character_spells: %+v", ch.Spells)
	}
	found := false
	for _, g := range ch.GrantedSpells {
		if g.Spell.Slug == "remove-curse" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected virtual remove-curse, got %+v", ch.GrantedSpells)
	}
	stealth := ch.EffectiveSkillMark("stealth")
	if !stealth.Proficient {
		t.Fatal("stealth should be proficient from item")
	}
	bonus := rules.SkillBonus(ch.Scores(), ch.Level, stealth)
	plain := rules.SkillBonus(ch.BaseScores(), ch.Level, rules.SkillMark{Slug: "stealth"})
	if bonus <= plain {
		t.Fatalf("stealth should include PB: %d vs %d", bonus, plain)
	}
	before := append([]rules.Pool{}, ch.Resources...)
	if err := svc.CastSpell(ch, uid, rc.ID, rules.KindItem, 3, 1); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if !rules.PoolsEqual(before, ch.Resources) {
		t.Fatalf("item cast must not spend: before %+v after %+v", before, ch.Resources)
	}
	if err := svc.UnequipItem(ch, uid, it.ID); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if len(ch.GrantedSpells) != 0 {
		t.Fatalf("unequip should drop grants %+v", ch.GrantedSpells)
	}
	if ch.EffectiveSkillMark("stealth").Proficient {
		t.Fatal("stealth overlay should revert")
	}
	if len(ch.Spells) != 0 {
		t.Fatalf("learned list changed %+v", ch.Spells)
	}
}

func TestLeatherShieldCatalogAC(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Tank", 5, 1)
	leather, err := svc.Catalog.ItemBySlug("leather-armor")
	if err != nil {
		t.Fatal(err)
	}
	shield, err := svc.Catalog.ItemBySlug("shield")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: leather.ID, EquipNow: true, EquipSlot: "armor"}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if _, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: shield.ID, EquipNow: true, EquipSlot: "shield"}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	st := rules.DeriveCombat(ch.CombatInput())
	if st.AC != 15 {
		t.Fatalf("leather+shield ac %d", st.AC)
	}
}

func TestEquippedWeaponSheetLineUsesCharacterAttack(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Fighter", 1, 1)
	long, err := svc.Catalog.ItemBySlug("longsword")
	if err != nil {
		t.Fatal(err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: long.ID, EquipNow: true, EquipSlot: "main_hand"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, ok := ch.InventoryByID(it.ID)
	if !ok || !got.CanAttack() {
		t.Fatalf("equipped weapon %+v ok=%v", got, ok)
	}
	atk, ok := ch.WeaponAttack(got, false)
	if !ok || !atk.Proficient {
		t.Fatalf("fighter longsword attack %+v ok=%v", atk, ok)
	}
	if atk.Line != "+1 to hit · 1d8-1 slashing" {
		t.Fatalf("expected STR -1 + PB 2, got %q", atk.Line)
	}
	eq := equippedRows(ch)
	if len(eq) != 1 || eq[0].Stats != atk.Line {
		t.Fatalf("equipped stats %+v want %q", eq, atk.Line)
	}

	spare, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: long.ID, CustomName: "Spare blade"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	packItem, ok := ch.InventoryByID(spare.ID)
	if !ok {
		t.Fatal("missing pack sword")
	}
	packStat := itemRowStats(packItem)
	if strings.Contains(packStat, "to hit") {
		t.Fatalf("pack should stay item-only, got %q", packStat)
	}
	for _, row := range packRows(ch) {
		if row.ID == spare.ID && strings.Contains(row.Stats, "to hit") {
			t.Fatalf("pack row %q", row.Stats)
		}
	}

	wiz := insertLeveled(t, svc, uid, cid, "Wizard", 2, 1)
	wizSword, err := svc.AddItem(wiz, uid, AddItemInput{CatalogID: long.ID, EquipNow: true, EquipSlot: "main_hand"})
	if err != nil {
		t.Fatal(err)
	}
	wiz, _ = svc.Get(wiz.ID)
	got, _ = wiz.InventoryByID(wizSword.ID)
	wizAtk, _ := wiz.WeaponAttack(got, false)
	if wizAtk.Proficient || wizAtk.Line != "-1 to hit · 1d8-1 slashing" {
		t.Fatalf("wizard longsword %+v", wizAtk)
	}
}
