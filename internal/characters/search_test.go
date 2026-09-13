package characters

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestItemTypePickerAndConsumableSearch(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Searcher", 2, 1)

	r := levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/new", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.newItemModal(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	open := rec.Body.String()
	if strings.Contains(open, "template error") {
		t.Fatalf("template error:\n%s", open)
	}
	if !strings.Contains(open, "/items/weapons") || !strings.Contains(open, "/items/consumables") {
		t.Fatalf("type picker missing weapon/consumable:\n%s", open)
	}
	if !strings.Contains(open, "/items/armor") || !strings.Contains(open, "/items/jewelry") {
		t.Fatalf("type picker missing armor/jewelry:\n%s", open)
	}
	if strings.Contains(open, `disabled title="`) {
		t.Fatalf("armor/jewelry should be enabled:\n%s", open)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/consumables", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.consumableSearch(rec, r)
	open = rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(open, `id="item-search-q"`) {
		t.Fatalf("consumable search input: %d %s", rec.Code, open)
	}
	if !strings.Contains(open, `hx-target="#item-search-results"`) || !strings.Contains(open, `hx-trigger="keyup changed delay:300ms, search"`) {
		t.Fatalf("consumable debounce:\n%s", open)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/search?q=potion", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.searchItems(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d body %s", rec.Code, body)
	}
	if strings.Contains(body, `<input`) {
		t.Fatalf("results swap must not remount the input:\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "potion") {
		t.Fatalf("expected potion results:\n%s", body)
	}
}

func TestBaseWeaponSearchCaretAndCase(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Searcher", 2, 1)

	r := levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/weapons", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.weaponBases(rec, r)
	open := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, open)
	}
	if !strings.Contains(open, `id="base-search-q"`) {
		t.Fatalf("stable base search input:\n%s", open)
	}
	if !strings.Contains(open, `hx-target="#base-search-results"`) || !strings.Contains(open, `hx-trigger="keyup changed delay:300ms, search"`) {
		t.Fatalf("base debounce:\n%s", open)
	}
	if !strings.Contains(open, "Longsword") {
		t.Fatalf("empty query should list PHB bases:\n%s", open)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/weapons/search?q=SWORD", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.searchBaseWeapons(rec, r)
	upper := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("search %d %s", rec.Code, upper)
	}
	if strings.Contains(upper, `<input`) {
		t.Fatalf("base results must not remount the input:\n%s", upper)
	}
	if !strings.Contains(upper, "Longsword") && !strings.Contains(strings.ToLower(upper), "sword") {
		t.Fatalf("case-insensitive base search failed:\n%s", upper)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/weapons/search?q=длинн", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.searchBaseWeapons(rec, r)
	ru := rec.Body.String()
	if !strings.Contains(strings.ToLower(ru), "longsword") && !strings.Contains(ru, "Длинный") {
		t.Fatalf("RU base search:\n%s", ru)
	}
}

func TestBaseArmorAndJewelrySearch(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Searcher", 2, 1)

	r := levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/armor", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.armorBases(rec, r)
	open := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("armor bases %d %s", rec.Code, open)
	}
	if !strings.Contains(open, `id="base-search-q"`) || !strings.Contains(open, `hx-trigger="keyup changed delay:300ms, search"`) {
		t.Fatalf("armor debounce:\n%s", open)
	}
	if !strings.Contains(open, "/items/armor/search") {
		t.Fatalf("armor search path:\n%s", open)
	}
	if !strings.Contains(strings.ToLower(open), "leather") {
		t.Fatalf("empty armor query should list PHB bases:\n%s", open)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/armor/search?q=КИРАС", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.searchBaseArmor(rec, r)
	body := rec.Body.String()
	if strings.Contains(body, `<input`) {
		t.Fatalf("armor results must not remount the input:\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "breastplate") && !strings.Contains(body, "Кираса") {
		t.Fatalf("RU armor search:\n%s", body)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/jewelry", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.jewelryBases(rec, r)
	jew := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(jew, `id="base-search-q"`) {
		t.Fatalf("jewelry bases %d %s", rec.Code, jew)
	}
	if !strings.Contains(strings.ToLower(jew), "ring") {
		t.Fatalf("jewelry slots:\n%s", jew)
	}
}

func TestSpellSearchDoesNotSwapInput(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Caster", 2, 1)

	r := levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/spells/search?q=fire", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.searchSpells(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "template error") {
		t.Fatalf("template error:\n%s", body)
	}
	if strings.Contains(body, `type="search"`) || strings.Contains(body, `name="q"`) {
		t.Fatalf("spell results must not include the search field:\n%s", body)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/spells/search?q=FIRE", "", uid, ch.ID)
	rec = httptest.NewRecorder()
	c.searchSpells(rec, r)
	if rec.Code != http.StatusOK || !strings.Contains(strings.ToLower(rec.Body.String()), "fire") {
		t.Fatalf("spell case-insensitive search: %d %s", rec.Code, rec.Body.String())
	}
}

func TestConfigureShowsCatalogFeaturesAndStableSearch(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Searcher", 2, 1)
	long, err := c.Catalog.ItemBySlug("longsword")
	if err != nil {
		t.Fatal(err)
	}

	path := "/characters/" + strconv.FormatInt(ch.ID, 10) + "/items/" + strconv.FormatInt(long.ID, 10) + "/configure?rarity=magic"
	r := levelUpForm(t, http.MethodGet, path, "", uid, ch.ID)
	r.SetPathValue("itemID", strconv.FormatInt(long.ID, 10))
	rec := httptest.NewRecorder()
	c.configureItem(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "template error") {
		t.Fatalf("template error:\n%s", body)
	}
	if !strings.Contains(body, "1d8") || !strings.Contains(strings.ToLower(body), "versatile") {
		t.Fatalf("magic longsword should auto-fill base properties:\n%s", body)
	}
	if strings.Contains(strings.ToLower(body), "base damage") {
		t.Fatalf("base hit must not be a feature row:\n%s", body)
	}
	if !strings.Contains(body, `id="feature-search-q"`) {
		t.Fatalf("stable feature search input:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#feature-search-results"`) {
		t.Fatalf("feature search must target results only:\n%s", body)
	}
	if !strings.Contains(body, "96-arms") {
		t.Fatalf("PHB arms table link missing:\n%s", body)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/"+strconv.FormatInt(long.ID, 10)+"/features/search?q=vorpal", "", uid, ch.ID)
	r.SetPathValue("itemID", strconv.FormatInt(long.ID, 10))
	rec = httptest.NewRecorder()
	c.searchItemFeatures(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("feature search %d", rec.Code)
	}
	hits := rec.Body.String()
	if strings.Contains(hits, `<input`) {
		t.Fatalf("feature results must not remount the input:\n%s", hits)
	}
	if !strings.Contains(strings.ToLower(hits), "vorpal") {
		t.Fatalf("expected vorpal hit:\n%s", hits)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/items/"+strconv.FormatInt(long.ID, 10)+"/features/search?q=BONUS", "", uid, ch.ID)
	r.SetPathValue("itemID", strconv.FormatInt(long.ID, 10))
	rec = httptest.NewRecorder()
	c.searchItemFeatures(rec, r)
	bonus := rec.Body.String()
	if !strings.Contains(strings.ToLower(bonus), "attack") && !strings.Contains(strings.ToLower(bonus), "atk") {
		t.Fatalf("generic +N/stat features should rank first:\n%s", bonus)
	}
}
