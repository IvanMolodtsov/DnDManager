package battles

import (
	"testing"
)

func TestGenerateLootSoulsNotDoubleApplied(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	for _, u := range b.Units {
		if !u.IsMonster() {
			continue
		}
		if _, _, err := svc.ApplyUnitDamage(cid, dmID, u.ID, 999, "slashing"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Repo.InsertUnit(&Unit{
		BattleID: b.ID, Kind: KindCompanion, Name: "Wolf", Dead: true, CR: 20, CRLabel: "20",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Repo.InsertUnit(&Unit{
		BattleID: b.ID, Kind: KindSummon, Name: "Bear", Dead: true, CR: 20, CRLabel: "20",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.End(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateLoot(cid, dmID); err != nil {
		t.Fatal(err)
	}
	camp, err := svc.Campaigns.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	wantSouls := 24 // two goblins at CR 1/4 → floor(12.5)*2
	if camp.Souls != wantSouls {
		t.Fatalf("campaign souls %d want %d", camp.Souls, wantSouls)
	}
	got, err := svc.GetForCampaign(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LootGenerated || !got.SoulsApplied {
		t.Fatalf("flags generated=%v applied=%v", got.LootGenerated, got.SoulsApplied)
	}
	soulsRow := 0
	goldRow := 0
	for _, row := range got.Loot {
		switch row.Kind {
		case LootSouls:
			soulsRow = row.Qty
		case LootGold:
			goldRow = row.Qty
		}
	}
	if soulsRow != wantSouls {
		t.Fatalf("loot souls %d", soulsRow)
	}
	if goldRow < 20 || goldRow > 200 {
		t.Fatalf("two goblin gold pile %d", goldRow)
	}

	if _, err := svc.AddQuestItem(cid, dmID, "Crown of the Goblin King"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateLoot(cid, dmID); err != nil {
		t.Fatal(err)
	}
	camp, err = svc.Campaigns.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	if camp.Souls != wantSouls {
		t.Fatalf("re-roll doubled souls to %d", camp.Souls)
	}
	got, err = svc.GetForCampaign(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	soulsRow = 0
	quest := false
	for _, row := range got.Loot {
		if row.Kind == LootSouls {
			soulsRow = row.Qty
		}
		if row.Kind == LootQuest && row.NameEN == "Crown of the Goblin King" {
			quest = true
		}
	}
	if soulsRow != wantSouls {
		t.Fatalf("re-roll souls row %d", soulsRow)
	}
	if !quest {
		t.Fatal("quest item lost on re-roll")
	}
}

func TestGenerateLootRequiresEndedFight(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateLoot(cid, dmID); err != ErrWrongStatus {
		t.Fatalf("want ErrWrongStatus got %v", err)
	}
}
