package catalog

import (
	"path/filepath"
	"strings"
	"testing"

	"dndmanager/internal/platform"
)

func TestPHBSeedAndWizardOptions(t *testing.T) {
	dir := t.TempDir()
	db, err := platform.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	mig := filepath.Join("..", "..", "migrations")
	if err := platform.Migrate(db, mig); err != nil {
		t.Fatal(err)
	}

	svc := &Service{Repo: &Repository{DB: db}}
	races, err := svc.ListRaces()
	if err != nil {
		t.Fatal(err)
	}
	bgs, err := svc.ListBackgrounds()
	if err != nil {
		t.Fatal(err)
	}
	classes, err := svc.ListClasses()
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListItems()
	if err != nil {
		t.Fatal(err)
	}
	spells, err := svc.ListSpells()
	if err != nil {
		t.Fatal(err)
	}

	if len(races) != 11 {
		t.Fatalf("races %d want 11", len(races))
	}
	if len(bgs) != 13 {
		t.Fatalf("backgrounds %d want 13", len(bgs))
	}
	if len(classes) != 12 {
		t.Fatalf("classes %d want 12", len(classes))
	}
	if len(items) < 500 {
		t.Fatalf("items %d want >= 500 (generated 5eapi + stubs)", len(items))
	}
	itemBySlug := map[string]Item{}
	stubs := 0
	for i := range items {
		itemBySlug[items[i].Slug] = items[i]
		if items[i].IsStub {
			stubs++
		}
	}
	if stubs < 1 {
		t.Fatal("expected 5e14 stubs such as moonblade")
	}
	leather, ok := itemBySlug["leather-armor"]
	if !ok || leather.ACBase != 11 || leather.ArmorCategory != "light" || leather.SuggestedSlot != "armor" {
		t.Fatalf("leather %+v", leather)
	}
	shield, ok := itemBySlug["shield"]
	if !ok || shield.ACBase != 2 || !shield.IsShield() {
		t.Fatalf("shield %+v", shield)
	}
	if _, ok := itemBySlug["potion-of-healing"]; !ok {
		if _, ok := itemBySlug["potion-of-healing-common"]; !ok {
			t.Fatal("missing healing potion")
		}
	}
	moon, ok := itemBySlug["moonblade"]
	if !ok || !moon.IsStub || moon.SuggestedSlot != "main_hand" {
		t.Fatalf("moonblade %+v", moon)
	}
	if moon.SourceURLRU == "" || !strings.Contains(moon.SourceURLRU, "5e14.dnd.su/items/") {
		t.Fatalf("moonblade RU URL %s", moon.SourceURLRU)
	}
	long, ok := itemBySlug["longsword"]
	if !ok || !long.IsBase || long.DamageDice != "1d8" || long.DamageType != "slashing" || long.WeaponCategory != "martial" {
		t.Fatalf("longsword %+v", long)
	}
	if !long.HasProperty("versatile") || long.VersatileDice != "1d10" {
		t.Fatalf("longsword versatile %+v", long)
	}
	if long.SourceURLRU != ArmsTableURL {
		t.Fatalf("longsword arms URL %s", long.SourceURLRU)
	}
	if long.NameRU != "Длинный меч" {
		t.Fatalf("longsword RU %q", long.NameRU)
	}
	wantSlugs := map[string]bool{}
	for _, a := range PHBArms() {
		wantSlugs[a.Slug] = true
	}
	if len(wantSlugs) != 37 {
		t.Fatalf("PHB arms table %d want 37", len(wantSlugs))
	}
	bases := 0
	for slug := range wantSlugs {
		it, ok := itemBySlug[slug]
		if !ok || !it.IsBase {
			t.Fatalf("missing PHB base %s ok=%v is_base=%v", slug, ok, it.IsBase)
		}
		arm, _ := PHBArmBySlug(slug)
		if it.DamageDice != arm.Dice || it.DamageType != arm.DamageType || it.WeaponCategory != arm.Category {
			t.Fatalf("%s stats %+v want %s %s %s", slug, it, arm.Dice, arm.DamageType, arm.Category)
		}
		bases++
	}
	if bases != 37 {
		t.Fatalf("is_base count %d", bases)
	}
	listed, err := svc.ListBaseWeapons()
	if err != nil || len(listed) != 37 {
		t.Fatalf("ListBaseWeapons %d %v", len(listed), err)
	}
	armors, err := svc.ListBaseArmor()
	if err != nil || len(armors) != 13 {
		t.Fatalf("ListBaseArmor %d %v", len(armors), err)
	}
	for _, a := range PHBArmor() {
		it, ok := itemBySlug[a.Slug]
		if !ok || !it.IsBase || it.ArmorCategory != a.Category || it.ACBase != a.ACBase {
			t.Fatalf("PHB armor %s ok=%v %+v", a.Slug, ok, it)
		}
		if it.SourceURLRU != ArmorTableURL {
			t.Fatalf("%s armor URL %s", a.Slug, it.SourceURLRU)
		}
	}
	jewels, err := svc.ListBaseJewelry()
	if err != nil || len(jewels) != 7 {
		t.Fatalf("ListBaseJewelry %d %v", len(jewels), err)
	}
	ring, ok := itemBySlug["ring"]
	if !ok || !ring.IsBase || ring.Kind != "jewelry" || ring.SuggestedSlot != "ring" {
		t.Fatalf("ring base %+v", ring)
	}
	stats, err := svc.ListStatFeatures()
	if err != nil || len(stats) < 40 {
		t.Fatalf("catalog_stat_features %d %v", len(stats), err)
	}
	statSlug := map[string]bool{}
	for _, sf := range stats {
		statSlug[sf.Slug] = true
	}
	for _, want := range []string{"skill-proficiency", "grant-spell-1", "artifact-ac-1", "artifact-spell-3", "ability-bonus"} {
		if !statSlug[want] {
			t.Fatalf("missing stat feature %s", want)
		}
	}
	if len(spells) != 378 {
		t.Fatalf("spells %d want 378", len(spells))
	}
	var mm *Spell
	for i := range spells {
		if spells[i].Slug == "magic-missile" {
			mm = &spells[i]
			break
		}
	}
	if mm == nil {
		t.Fatal("missing magic-missile")
	}
	if mm.DamageFormula == "" || mm.DamageType != "force" {
		t.Fatalf("magic missile formula %+v", mm)
	}
	if !strings.Contains(mm.SourceURLRU, "5e14.dnd.su/spells/") || !strings.Contains(mm.SourceURLRU, "-") {
		t.Fatalf("magic missile invented 5e14 path: %s", mm.SourceURLRU)
	}
	if mm.SourceURL == "" || !strings.HasPrefix(mm.SourceURL, "https://www.dnd5eapi.co/api/2014/spells/") {
		t.Fatalf("magic missile 5eapi %s", mm.SourceURL)
	}

	bySlug := map[string]Spell{}
	for i := range spells {
		bySlug[spells[i].Slug] = spells[i]
	}
	hex, ok := bySlug["hex"]
	if !ok || hex.DamageFormula != "1d6" || hex.DamageType != "necrotic" {
		t.Fatalf("hex formula %+v", hex)
	}
	wb, ok := bySlug["witch-bolt"]
	if !ok {
		t.Fatal("missing witch-bolt")
	}
	f, dt, _ := wb.FormulaAt(2, 1)
	if f != "2d12" || dt != "lightning" {
		t.Fatalf("witch-bolt slot 2: %s %s", f, dt)
	}

	var raceCount, classCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_subclasses`).Scan(&classCount); err != nil {
		t.Fatal(err)
	}
	if classCount != 15 {
		t.Fatalf("subclasses %d want 15", classCount)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_features`).Scan(&raceCount); err != nil {
		t.Fatal(err)
	}
	if raceCount < 90 {
		t.Fatalf("features %d want >= 90", raceCount)
	}

	for _, r := range races {
		if r.SourceURLRU == "" || !strings.Contains(r.SourceURLRU, "5e14.dnd.su/race/") {
			t.Fatalf("race %s missing 5e14 URL: %q", r.Slug, r.SourceURLRU)
		}
		if strings.Contains(r.SourceURLRU, "/race/human/") || strings.Contains(r.SourceURLRU, "dnd.su/race/elf/") {
			t.Fatalf("race %s still has invented slug path %s", r.Slug, r.SourceURLRU)
		}
	}
	for _, c := range classes {
		if !strings.Contains(c.SourceURLRU, "5e14.dnd.su/class/") || !strings.Contains(c.SourceURLRU, "-") {
			t.Fatalf("class %s bad 5e14 URL %s", c.Slug, c.SourceURLRU)
		}
		if c.SourceURL == "" || !strings.HasPrefix(c.SourceURL, "https://www.dnd5eapi.co/api/2014/classes/") {
			t.Fatalf("class %s bad 5eapi URL %s", c.Slug, c.SourceURL)
		}
	}

	warlock, err := svc.Class(12)
	if err != nil || warlock.Slug != "warlock" || warlock.SubclassLevel != 1 {
		t.Fatalf("warlock row: %+v %v", warlock, err)
	}
	subs, err := svc.ListSubclasses(12)
	if err != nil || len(subs) == 0 || subs[0].Slug != "fiend" {
		t.Fatalf("warlock subclasses: %+v %v", subs, err)
	}
	if !strings.Contains(warlock.SourceURLRU, "104-warlock") {
		t.Fatalf("warlock RU URL %s", warlock.SourceURLRU)
	}
}

func TestSearchItemsAndSpellsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	db, err := platform.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := platform.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Repo: &Repository{DB: db}}

	swordUpper, err := svc.SearchItems("SWORD", 20)
	if err != nil {
		t.Fatal(err)
	}
	swordLower, err := svc.SearchItems("sword", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(swordUpper) == 0 || len(swordLower) == 0 {
		t.Fatalf("sword search empty upper=%d lower=%d", len(swordUpper), len(swordLower))
	}
	hasLong := false
	for _, it := range swordUpper {
		if it.Slug == "longsword" {
			hasLong = true
		}
	}
	if !hasLong {
		t.Fatalf("SWORD should match longsword: %+v", slugs(swordUpper))
	}

	ru, err := svc.SearchItems("ЛУННЫЙ", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSlug(ru, "moonblade") {
		t.Fatalf("Cyrillic case should match moonblade: %+v", slugs(ru))
	}

	spUpper, err := svc.SearchSpells("FIRE", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(spUpper) == 0 {
		t.Fatal("FIRE should match fire spells")
	}
}

func slugs(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Slug
	}
	return out
}

func containsSlug(items []Item, slug string) bool {
	for _, it := range items {
		if it.Slug == slug {
			return true
		}
	}
	return false
}
