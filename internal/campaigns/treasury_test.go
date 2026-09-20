package campaigns

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"dndmanager/internal/platform"
	"dndmanager/internal/users"
)

func TestPartyGoldSumsHeldGold(t *testing.T) {
	sum := PartyGold([]CharacterSummary{
		{Gold: 10}, {Gold: 25}, {Gold: 0},
	})
	if sum != 35 {
		t.Fatalf("sum %d", sum)
	}
	if PartyGold(nil) != 0 {
		t.Fatalf("empty sum")
	}
}

func TestAdjustSoulsAndCapDMOnly(t *testing.T) {
	svc, dmID, cid, playerID := testCamp(t)
	if err := svc.AdjustSouls(cid, dmID, 100); err != nil {
		t.Fatal(err)
	}
	camp, err := svc.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	if camp.Souls != 100 {
		t.Fatalf("souls %d", camp.Souls)
	}
	if err := svc.SetSoulsCap(cid, dmID, 80); err != nil {
		t.Fatal(err)
	}
	camp, _ = svc.Get(cid)
	if camp.SoulsCap != 80 || camp.Souls != 80 {
		t.Fatalf("cap clamp souls=%d cap=%d", camp.Souls, camp.SoulsCap)
	}
	if err := svc.AdjustSouls(cid, dmID, 50); err != nil {
		t.Fatal(err)
	}
	camp, _ = svc.Get(cid)
	if camp.Souls != 80 {
		t.Fatalf("cap block %d", camp.Souls)
	}
	if err := svc.SetSoulsCap(cid, dmID, 0); !errors.Is(err, ErrSoulsCap) {
		t.Fatalf("cap 0 %v", err)
	}
	if err := svc.AdjustSouls(cid, playerID, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("player souls %v", err)
	}
	if err := svc.SetSoulsCap(cid, playerID, 9000); !errors.Is(err, ErrForbidden) {
		t.Fatalf("player cap %v", err)
	}
	camp, _ = svc.Get(cid)
	if camp.Souls != 80 || camp.SoulsCap != 80 {
		t.Fatalf("player mutate leaked souls=%d cap=%d", camp.Souls, camp.SoulsCap)
	}
}

func TestCampaignTreasuryHTTP(t *testing.T) {
	svc, dmID, cid, playerID := testCamp(t)
	chars := &stubChars{list: []CharacterSummary{
		{ID: 11, Name: "Hero", Owner: "dm", Gold: 40},
		{ID: 12, Name: "Sidekick", Owner: "player", Gold: 15},
	}}
	c := testCampController(t, svc, chars)
	idStr := strconv.FormatInt(cid, 10)

	dmShow := campaignReq(t, http.MethodGet, "/campaigns/"+idStr, "", dmID, cid, 0)
	rec := httptest.NewRecorder()
	c.show(rec, dmShow)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("dm show %d %s", rec.Code, body)
	}
	if !strings.Contains(body, `id="campaign-treasury"`) || !strings.Contains(body, `text-gold">55</strong>`) {
		t.Fatalf("party gold sum missing:\n%s", body)
	}
	if !strings.Contains(body, "/souls/adjust") || !strings.Contains(body, "/characters/11/gold/adjust") {
		t.Fatalf("dm controls missing:\n%s", body)
	}
	if !strings.Contains(body, "mutations/new") {
		t.Fatalf("dm mutation button missing:\n%s", body)
	}

	playerShow := campaignReq(t, http.MethodGet, "/campaigns/"+idStr, "", playerID, cid, 0)
	rec = httptest.NewRecorder()
	c.show(rec, playerShow)
	body = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("player show %d %s", rec.Code, body)
	}
	if !strings.Contains(body, `id="campaign-treasury"`) || !strings.Contains(body, "Party gold") {
		t.Fatalf("player should see read-only treasury:\n%s", body)
	}
	if strings.Contains(body, "/souls/adjust") || strings.Contains(body, "/souls/cap") || strings.Contains(body, "/gold/adjust") {
		t.Fatalf("player should not get ± / cap:\n%s", body)
	}
	if strings.Contains(body, "mutations/new") || strings.Contains(body, "/revive") {
		t.Fatalf("player should not get revive/mutation:\n%s", body)
	}

	playerModal := campaignReq(t, http.MethodGet, "/campaigns/"+idStr+"/souls/adjust?sign=%2B", "", playerID, cid, 0)
	rec = httptest.NewRecorder()
	c.soulsAdjustModal(rec, playerModal)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player souls modal %d", rec.Code)
	}
	playerGoldModal := campaignReq(t, http.MethodGet, "/campaigns/"+idStr+"/characters/11/gold/adjust?sign=%2B", "", playerID, cid, 11)
	rec = httptest.NewRecorder()
	c.goldAdjustModal(rec, playerGoldModal)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player gold modal %d", rec.Code)
	}

	playerSouls := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/souls", "sign=%2B&amount=10", playerID, cid, 0)
	rec = httptest.NewRecorder()
	c.applySouls(rec, playerSouls)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player souls POST %d %s", rec.Code, rec.Body.String())
	}

	playerGold := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/characters/11/gold", "sign=%2B&amount=10", playerID, cid, 11)
	rec = httptest.NewRecorder()
	c.applyGold(rec, playerGold)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player gold POST %d %s", rec.Code, rec.Body.String())
	}
	if chars.goldCalls != 0 {
		t.Fatalf("player gold reached AdjustCampaignGold")
	}

	playerRevive := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/characters/11/revive", "", playerID, cid, 11)
	rec = httptest.NewRecorder()
	c.applyRevive(rec, playerRevive)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player revive POST %d %s", rec.Code, rec.Body.String())
	}
	if chars.reviveCalls != 0 {
		t.Fatalf("player revive reached stub")
	}

	chars.list[0].Dead = true
	dmRevive := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/characters/11/revive", "", dmID, cid, 11)
	rec = httptest.NewRecorder()
	c.applyRevive(rec, dmRevive)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusOK {
		t.Fatalf("dm revive POST %d %s", rec.Code, rec.Body.String())
	}
	if chars.reviveCalls != 1 {
		t.Fatalf("dm revive calls %d", chars.reviveCalls)
	}

	dmSouls := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/souls", "sign=%2B&amount=25", dmID, cid, 0)
	rec = httptest.NewRecorder()
	c.applySouls(rec, dmSouls)
	out := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(out, "template error") {
		t.Fatalf("dm souls POST %d %s", rec.Code, out)
	}
	if !strings.Contains(out, `id="campaign-treasury"`) || !strings.Contains(out, `text-gold">25</strong>`) {
		t.Fatalf("souls island:\n%s", out)
	}
	camp, _ := svc.Get(cid)
	if camp.Souls != 25 {
		t.Fatalf("souls persisted %d", camp.Souls)
	}

	dmGold := campaignReq(t, http.MethodPost, "/campaigns/"+idStr+"/characters/11/gold", "sign=%2B&amount=10", dmID, cid, 11)
	rec = httptest.NewRecorder()
	c.applyGold(rec, dmGold)
	out = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(out, "template error") {
		t.Fatalf("dm gold POST %d %s", rec.Code, out)
	}
	if chars.list[0].Gold != 50 {
		t.Fatalf("hero gold %d", chars.list[0].Gold)
	}
	if !strings.Contains(out, `text-gold">65</strong>`) {
		t.Fatalf("updated party gold missing:\n%s", out)
	}
}

type stubChars struct {
	list        []CharacterSummary
	goldCalls   int
	reviveCalls int
}

func (s *stubChars) ListByCampaign(campaignID int64) ([]CharacterSummary, error) {
	return s.list, nil
}

func (s *stubChars) AdjustCampaignGold(campaignID, characterID, userID int64, delta int) error {
	s.goldCalls++
	for i := range s.list {
		if s.list[i].ID == characterID {
			next := s.list[i].Gold + delta
			if next < 0 {
				next = 0
			}
			s.list[i].Gold = next
			return nil
		}
	}
	return ErrNotFound
}

func (s *stubChars) BroadcastWallet(campaignID int64) {}

func (s *stubChars) Revive(campaignID, characterID, userID int64) error {
	s.reviveCalls++
	for i := range s.list {
		if s.list[i].ID == characterID {
			s.list[i].Dead = false
			return nil
		}
	}
	return ErrNotFound
}

func testCamp(t *testing.T) (*Service, int64, int64, int64) {
	t.Helper()
	dir := t.TempDir()
	db, err := platform.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := platform.Migrate(db, platform.AssetDir("migrations")); err != nil {
		t.Fatal(err)
	}
	userRepo := &users.Repository{DB: db}
	dmID, err := userRepo.Create(&users.User{Username: "dm", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Repo: &Repository{DB: db}}
	camp, err := svc.Create("Table", dmID)
	if err != nil {
		t.Fatal(err)
	}
	playerID, err := userRepo.Create(&users.User{Username: "player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	return svc, dmID, camp.ID, playerID
}

func testCampController(t *testing.T, svc *Service, chars CharacterLister) *Controller {
	t.Helper()
	bundle, err := platform.LoadBundle(platform.AssetDir("locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(platform.AssetDir(filepath.Join("web", "templates")), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return &Controller{Svc: svc, Characters: chars, Render: render}
}

func campaignReq(t *testing.T, method, path, body string, uid, campaignID, characterID int64) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", strconv.FormatInt(campaignID, 10))
	if characterID != 0 {
		r.SetPathValue("cid", strconv.FormatInt(characterID, 10))
	}
	name := "dm"
	if uid != 0 {
		name = "u" + strconv.FormatInt(uid, 10)
	}
	return r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: uid, Username: name, Role: users.RolePlayer, Language: "en",
	}))
}
