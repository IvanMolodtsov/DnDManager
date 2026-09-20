package characters

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

func testController(t *testing.T) (*Controller, int64, int64) {
	t.Helper()
	svc, uid, cid := testSvc(t)
	bundle, err := platform.LoadBundle(platform.AssetDir("locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(platform.AssetDir(filepath.Join("web", "templates")), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return &Controller{Svc: svc, Catalog: svc.Catalog, Campaigns: svc.Campaigns, Render: render}, uid, cid
}

func levelUpForm(t *testing.T, method, path, body string, uid, charID int64) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", strconv.FormatInt(charID, 10))
	return r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: uid, Username: "owner", Role: users.RolePlayer, Language: "en",
	}))
}

func TestShowLevelUpSubclassOutsideDynamic(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	r := levelUpForm(t, http.MethodGet, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/level-up", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.showLevelUp(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "template error") {
		t.Fatalf("template error:\n%s", body)
	}
	sub := strings.Index(body, `name="subclass_id"`)
	dyn := strings.Index(body, `id="levelup-dynamic"`)
	if sub < 0 || dyn < 0 || sub > dyn {
		t.Fatalf("subclass select must sit outside #levelup-dynamic:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#levelup-dynamic"`) {
		t.Fatalf("preview must target #levelup-dynamic:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#levelup-after-class"`) {
		t.Fatalf("class change must target #levelup-after-class:\n%s", body)
	}
}

func TestLevelUpPreviewDoesNotSwapSubclassSelect(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	r := levelUpForm(t, http.MethodPost, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/level-up/preview",
		"class_id=1&subclass_id=1&hp_mode=average", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.previewLevelUp(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `name="subclass_id"`) {
		t.Fatalf("preview swap must not include subclass select:\n%s", body)
	}
	if !strings.Contains(body, "Champion") {
		t.Fatalf("preview missing Champion:\n%s", body)
	}
}

func TestLevelUpPanelKeepsFighterChampion(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	r := levelUpForm(t, http.MethodPost, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/level-up/preview-panel",
		"class_id=1&subclass_id=1&hp_mode=average", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.previewLevelUpPanel(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="subclass_id"`) {
		t.Fatalf("subclass select missing:\n%s", body)
	}
	if !strings.Contains(body, `value="1" selected`) {
		t.Fatalf("Champion not selected after class-panel swap:\n%s", body)
	}
	if !strings.Contains(body, "Champion") {
		t.Fatalf("preview missing Champion:\n%s", body)
	}

	next, err := c.Svc.ApplyLevelUp(ch, uid, rules.Intent{Kind: rules.IntentLevelUp, ClassID: 1, SubclassID: 1, HPMode: rules.HPAverage})
	if err != nil {
		t.Fatal(err)
	}
	if next.Level != 3 || next.ClassLevels[0].SubclassID != 1 {
		t.Fatalf("applied %+v", next.ClassLevels)
	}
}

func TestLevelUpPanelKeepsWizardEvocation(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Wizard", 2, 1)
	r := levelUpForm(t, http.MethodPost, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/level-up/preview-panel",
		"class_id=2&subclass_id=3&hp_mode=average", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.previewLevelUpPanel(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="3" selected`) {
		t.Fatalf("Evocation not selected after class-panel swap:\n%s", body)
	}
	if !strings.Contains(body, "Evocation") {
		t.Fatalf("preview missing Evocation:\n%s", body)
	}

	next, err := c.Svc.ApplyLevelUp(ch, uid, rules.Intent{Kind: rules.IntentLevelUp, ClassID: 2, SubclassID: 3, HPMode: rules.HPAverage})
	if err != nil {
		t.Fatal(err)
	}
	if next.Level != 2 || next.ClassLevels[0].SubclassID != 3 {
		t.Fatalf("applied %+v", next.ClassLevels)
	}
}

func TestLevelUpPanelDropsChampionWhenClassChanges(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	r := levelUpForm(t, http.MethodPost, "/characters/"+strconv.FormatInt(ch.ID, 10)+"/level-up/preview-panel",
		"class_id=3&subclass_id=1&hp_mode=average", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.previewLevelUpPanel(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `value="1" selected`) {
		t.Fatalf("Champion should not stay selected for Cleric:\n%s", body)
	}
	if !strings.Contains(body, "Life Domain") {
		t.Fatalf("want Cleric subclass options:\n%s", body)
	}
}

func TestLevelUpIntentDropsMismatchedSubclass(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	r := levelUpForm(t, http.MethodPost, "/", "class_id=2&subclass_id=1&hp_mode=average", uid, ch.ID)
	in := c.levelUpIntent(r, ch)
	if in.SubclassID != 0 {
		t.Fatalf("want Champion dropped when class is Wizard, got %d", in.SubclassID)
	}
}

func TestLevelUpApplyRequiresSubclass(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Fighter", 1, 2)
	_, err := c.Svc.ApplyLevelUp(ch, uid, rules.Intent{Kind: rules.IntentLevelUp, ClassID: 1, HPMode: rules.HPAverage})
	if !errors.Is(err, rules.ErrSubclassRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestWizardSubclassPartialKeepsLifeDomain(t *testing.T) {
	c, uid, cid := testController(t)
	r := httptest.NewRequest(http.MethodGet,
		"/campaigns/"+strconv.FormatInt(cid, 10)+"/characters/wizard/subclass?class_id=3&subclass_id=5", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: uid, Username: "owner", Role: users.RolePlayer, Language: "en",
	}))
	rec := httptest.NewRecorder()
	c.subclassPartial(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="5" selected`) {
		t.Fatalf("Life Domain not selected:\n%s", body)
	}
}

func TestWizardSubclassPartialDropsChampionForCleric(t *testing.T) {
	c, uid, cid := testController(t)
	r := httptest.NewRequest(http.MethodGet,
		"/campaigns/"+strconv.FormatInt(cid, 10)+"/characters/wizard/subclass?class_id=3&subclass_id=1", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: uid, Username: "owner", Role: users.RolePlayer, Language: "en",
	}))
	rec := httptest.NewRecorder()
	c.subclassPartial(rec, r)
	body := rec.Body.String()
	if strings.Contains(body, `value="1" selected`) {
		t.Fatalf("Champion should not stay selected for Cleric:\n%s", body)
	}
	if !strings.Contains(body, "Life Domain") {
		t.Fatalf("want Cleric options:\n%s", body)
	}
}
