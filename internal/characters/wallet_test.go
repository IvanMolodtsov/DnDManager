package characters

import (
	"errors"
	"testing"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/users"
)

func TestAdjustGoldOwnerAndDM(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	ch := insertLeveled(t, svc, dmID, cid, "Hero", 1, 1)
	if err := svc.AdjustGold(ch, dmID, 50); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Gold != 50 {
		t.Fatalf("gold %d", got.Gold)
	}
	if err := svc.AdjustGold(got, dmID, -80); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ch.ID)
	if got.Gold != 0 {
		t.Fatalf("gold clamp %d", got.Gold)
	}

	userRepo := &users.Repository{DB: svc.Repo.DB}
	outsider, err := userRepo.Create(&users.User{Username: "nosy-gold", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdjustGold(got, outsider, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider gold %v", err)
	}
}

func TestCampaignSoulsCapAndDM(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	ch := insertLeveled(t, svc, dmID, cid, "Hero", 1, 1)
	if ch.SoulsCap != campaigns.DefaultSoulsCap {
		t.Fatalf("default cap %d", ch.SoulsCap)
	}
	if err := svc.AdjustSouls(ch, dmID, 100); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ch.ID)
	if got.Souls != 100 {
		t.Fatalf("souls %d", got.Souls)
	}
	if err := svc.SetSoulsCap(got, dmID, 80); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ch.ID)
	if got.SoulsCap != 80 || got.Souls != 80 {
		t.Fatalf("cap clamp souls=%d cap=%d", got.Souls, got.SoulsCap)
	}
	if err := svc.AdjustSouls(got, dmID, 50); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ch.ID)
	if got.Souls != 80 {
		t.Fatalf("cap block %d", got.Souls)
	}
	if err := svc.SetSoulsCap(got, dmID, 0); !errors.Is(err, ErrSoulsCap) {
		t.Fatalf("cap 0 %v", err)
	}

	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "player-souls", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, _ := svc.Campaigns.Get(cid)
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdjustSouls(got, playerID, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("player souls %v", err)
	}
}

func TestPartyGoldSumAndCampaignGoldDMOnly(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	hero := insertLeveled(t, svc, dmID, cid, "Hero", 1, 1)
	if err := svc.AdjustGold(hero, dmID, 40); err != nil {
		t.Fatal(err)
	}

	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "gold-player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, _ := svc.Campaigns.Get(cid)
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	side := insertLeveled(t, svc, playerID, cid, "Sidekick", 1, 1)
	if err := svc.AdjustGold(side, dmID, 15); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListByCampaign(cid)
	if err != nil {
		t.Fatal(err)
	}
	if got := campaigns.PartyGold(list); got != 55 {
		t.Fatalf("party gold %d", got)
	}

	if err := svc.AdjustCampaignGold(cid, hero.ID, playerID, 10); !errors.Is(err, campaigns.ErrForbidden) {
		t.Fatalf("player campaign gold %v", err)
	}
	hero, _ = svc.Get(hero.ID)
	if hero.Gold != 40 {
		t.Fatalf("player gold leaked %d", hero.Gold)
	}

	if err := svc.AdjustCampaignGold(cid, hero.ID, dmID, 10); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.ListByCampaign(cid)
	if got := campaigns.PartyGold(list); got != 65 {
		t.Fatalf("after dm gold %d", got)
	}
}
