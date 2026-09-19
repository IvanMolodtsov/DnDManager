package characters

import (
	"errors"
	"testing"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

func TestDeadStatusOnThreeFails(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Hero", 1, 1)
	ch.HPCurrent = 0
	if err := svc.Repo.UpdateVitals(ch.ID, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	for i := 0; i < 3; i++ {
		if err := svc.RecordDeathSave(ch, uid, 1); err != nil {
			t.Fatal(err)
		}
		ch, _ = svc.Get(ch.ID)
	}
	if ch.DeathFail < 3 || !rules.HasDead(ch.Effects) {
		t.Fatalf("dead status missing fails=%d effects=%v", ch.DeathFail, ch.Effects)
	}
	var deadID int64
	for _, e := range ch.Effects {
		if e.Slug == rules.DeadSlug {
			deadID = e.ID
			if e.Hidden || e.RemoveOnBattleEnd || e.Source != rules.SourceOther {
				t.Fatalf("dead flags %+v", e)
			}
		}
	}
	if err := svc.DismissEffect(ch, uid, deadID); !errors.Is(err, ErrDeadLocked) {
		t.Fatalf("owner remove dead %v", err)
	}
}

func TestReviveSpendsSoulsAndBlockedBelow(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Hero", 1, 1)
	ch.HPCurrent = 0
	if err := svc.Repo.UpdateVitals(ch.ID, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	for i := 0; i < 3; i++ {
		_ = svc.RecordDeathSave(ch, uid, 1)
		ch, _ = svc.Get(ch.ID)
	}
	if err := svc.Campaigns.AdjustSouls(cid, uid, 4999); err != nil {
		t.Fatal(err)
	}
	if err := svc.Revive(cid, ch.ID, uid); !errors.Is(err, campaigns.ErrReviveSouls) {
		t.Fatalf("low souls %v", err)
	}
	if err := svc.Campaigns.AdjustSouls(cid, uid, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.Revive(cid, ch.ID, uid); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rules.HasDead(got.Effects) || got.DeathFail != 0 || got.HPCurrent < 1 {
		t.Fatalf("after revive hp=%d fail=%d dead=%v", got.HPCurrent, got.DeathFail, got.Effects)
	}
	camp, _ := svc.Campaigns.Get(cid)
	if camp.Souls != 0 {
		t.Fatalf("souls left %d", camp.Souls)
	}
	if err := svc.Revive(cid, got.ID, uid); !errors.Is(err, campaigns.ErrNotDead) {
		t.Fatalf("second revive %v", err)
	}

	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "rev-player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, _ = svc.Campaigns.Get(cid)
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	got.HPCurrent = 0
	_ = svc.Repo.UpdateVitals(got.ID, 0, 0, 0, 0)
	got, _ = svc.Get(got.ID)
	for i := 0; i < 3; i++ {
		_ = svc.RecordDeathSave(got, uid, 1)
		got, _ = svc.Get(got.ID)
	}
	_ = svc.Campaigns.AdjustSouls(cid, uid, 5000)
	if err := svc.Revive(cid, got.ID, playerID); !errors.Is(err, campaigns.ErrForbidden) {
		t.Fatalf("player revive %v", err)
	}
}

func TestMutationReplaceSamePart(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Hero", 1, 1)
	first, err := svc.ApplyMutation(ch, uid, Mutation{BodyPart: "head", NameEN: "Wolf"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.ApplyMutation(ch, uid, Mutation{BodyPart: "head", NameEN: "Bear"}, []rules.FeatureDTO{
		{Stat: rules.StatSkillBonus, Value: "athletics:2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids %d %d", first.ID, second.ID)
	}
	got, _ := svc.Get(ch.ID)
	if len(got.Mutations) != 1 || got.Mutations[0].NameEN != "Bear" {
		t.Fatalf("list %+v", got.Mutations)
	}
	if len(got.Mutations[0].Features) != 1 {
		t.Fatalf("features replaced %+v", got.Mutations[0].Features)
	}
}

func TestMutationSkillBonusAppliesToCheck(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Hero", 1, 1)
	before := ch.SkillCheckBonus("athletics")
	if _, err := svc.ApplyMutation(ch, uid, Mutation{BodyPart: "arms", NameEN: "Ape"}, []rules.FeatureDTO{
		{Stat: rules.StatSkillBonus, Value: "athletics:2"},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ch.ID)
	if got.SkillCheckBonus("athletics") != before+2 {
		t.Fatalf("bonus %d want %d", got.SkillCheckBonus("athletics"), before+2)
	}
}

func TestMutationBlockSlotUnequipsBoots(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Hero", 1, 1)
	boots, err := svc.Catalog.ItemBySlug("boots")
	if err != nil {
		t.Fatal(err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: boots.ID, EquipNow: true, EquipSlot: "boots"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApplyMutation(ch, uid, Mutation{BodyPart: "legs", NameEN: "Naga"}, []rules.FeatureDTO{
		{Stat: rules.StatBlockSlot, Value: rules.SlotBoots},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ch.ID)
	held, ok := got.InventoryByID(it.ID)
	if !ok || held.Equipped() {
		t.Fatalf("boots still on %+v", held)
	}
	if err := svc.EquipItem(got, uid, it.ID, "boots"); err == nil {
		t.Fatal("equip boots should fail")
	}
}

func TestPickMutationMonsterBumpsCR(t *testing.T) {
	svc, _, _ := testSvc(t)
	m, cr, err := svc.PickMutationMonster(0, "medium", "beast")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || cr < 0 {
		t.Fatalf("pick %+v %d", m, cr)
	}
	_, _, err = svc.PickMutationMonster(0, "gargantuan", "not-a-type")
	if !errors.Is(err, ErrMutationType) {
		t.Fatalf("bad type %v", err)
	}
}
