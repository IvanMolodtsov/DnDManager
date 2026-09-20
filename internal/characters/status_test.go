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

func TestPlayerCannotAddStatus(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "hero-owner", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, err := svc.Campaigns.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	ch := insertLeveled(t, svc, playerID, cid, "Hero", 1, 1)
	if err := svc.AddStatus(ch, playerID, StatusSpec{NameEN: "Poisoned"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("owner add: %v", err)
	}
	if err := svc.AddStatus(ch, dmID, StatusSpec{NameEN: "Poisoned"}); err != nil {
		t.Fatal(err)
	}
}

func TestHiddenCurseHiddenOnPlayerSheetAndSkillUnchanged(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "hero-owner", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, err := svc.Campaigns.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	ch := insertLeveled(t, svc, playerID, cid, "Hero", 1, 1)
	before := rules.SkillBonus(ch.Scores(), ch.Level, ch.EffectiveSkillMark("athletics"))
	acBefore := rules.DeriveCombat(ch.CombatInput()).AC

	if err := svc.AddStatus(ch, dmID, StatusSpec{
		NameEN: "Hex Curse", Hidden: true, ACBonus: -4,
	}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	after := rules.SkillBonus(ch.Scores(), ch.Level, ch.EffectiveSkillMark("athletics"))
	if after != before {
		t.Fatalf("skill bonus %d -> %d", before, after)
	}
	if rules.DeriveCombat(ch.CombatInput()).AC != acBefore {
		t.Fatalf("hidden ac leak %d", rules.DeriveCombat(ch.CombatInput()).AC)
	}
	found := false
	for _, e := range ch.Effects {
		if e.Slug == "custom-hex-curse" && e.Hidden {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stored curse %+v", ch.Effects)
	}

	ctrl, _, _ := testControllerFrom(t, svc, dmID, cid)
	id := strconv.FormatInt(ch.ID, 10)
	r := levelUpForm(t, http.MethodGet, "/characters/"+id+"/vitals", "", playerID, ch.ID)
	rec := httptest.NewRecorder()
	ctrl.vitalsPartial(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("player vitals %d %s", rec.Code, body)
	}
	if strings.Contains(body, "Hex Curse") {
		t.Fatalf("player saw hidden name:\n%s", body)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+id+"/vitals", "", dmID, ch.ID)
	rec = httptest.NewRecorder()
	ctrl.vitalsPartial(rec, r)
	dmBody := rec.Body.String()
	if !strings.Contains(dmBody, "Hex Curse") {
		t.Fatalf("dm should see curse:\n%s", dmBody)
	}
}

func TestVisibleACStatusApplies(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	ch := insertLeveled(t, svc, dmID, cid, "Hero", 1, 1)
	before := rules.DeriveCombat(ch.CombatInput()).AC
	if err := svc.AddStatus(ch, dmID, StatusSpec{NameEN: "Shield of Faith", ACBonus: 2}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got := rules.DeriveCombat(ch.CombatInput()).AC
	if got != before+2 {
		t.Fatalf("ac %d want %d", got, before+2)
	}
}

func TestOwnerCannotDismissHidden(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "hero-owner", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, _ := svc.Campaigns.Get(cid)
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	ch := insertLeveled(t, svc, playerID, cid, "Hero", 1, 1)
	if err := svc.AddStatus(ch, dmID, StatusSpec{NameEN: "Hex Curse", Hidden: true}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if err := svc.DismissEffect(ch, playerID, ch.Effects[0].ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("owner dismiss hidden: %v", err)
	}
}

func testControllerFrom(t *testing.T, svc *Service, uid, cid int64) (*Controller, int64, int64) {
	t.Helper()
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
