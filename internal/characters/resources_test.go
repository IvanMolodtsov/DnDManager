package characters

import (
	"path/filepath"
	"testing"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

func testSvc(t *testing.T) (*Service, int64, int64) {
	t.Helper()
	dir := t.TempDir()
	db, err := platform.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := platform.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	userRepo := &users.Repository{DB: db}
	uid, err := userRepo.Create(&users.User{Username: "owner", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	campRepo := &campaigns.Repository{DB: db}
	cid, err := campRepo.Create("Table", "INVITE1", uid)
	if err != nil {
		t.Fatal(err)
	}
	cat := &catalog.Service{Repo: &catalog.Repository{DB: db}}
	svc := &Service{
		Repo:      &Repository{DB: db},
		Campaigns: &campaigns.Service{Repo: campRepo},
		Catalog:   cat,
		Rules:     &rules.Engine{Catalog: cat},
	}
	return svc, uid, cid
}

func insertLeveled(t *testing.T, svc *Service, uid, cid int64, name string, classID int64, levels int) *Character {
	t.Helper()
	ch := &Character{
		Name: name, OwnerID: uid, CampaignID: cid, Level: levels,
		STR: 8, DEX: 14, CON: 13, INT: 15, WIS: 10, CHA: 12,
		RaceID: 1, BackgroundID: 1, HPMax: 20, HPCurrent: 20, ProficiencyBonus: 2,
	}
	id, err := svc.Repo.InsertLive(ch, []rules.ClassProgress{{ClassID: classID, Levels: levels}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestWarlock4PactOnSheet(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Lock", 12, 4)
	pact, ok := rules.PactPool(ch.Resources)
	if !ok || pact.Max != 2 || pact.SlotLevel != 2 || pact.Current != 2 {
		t.Fatalf("warlock 4 resources %+v", ch.Resources)
	}
	if rules.HasKind(ch.Resources, rules.KindSlots) {
		t.Fatalf("pact must not merge into spell slots: %+v", ch.Resources)
	}
}

func TestWizard5ConsumeThirdSlot(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Wiz", 2, 5)
	if rules.SlotRemaining(ch.Resources, 3) != 2 {
		t.Fatalf("wizard 5 slots %+v", ch.Resources)
	}
	var mm int64
	spells, err := svc.Catalog.ListSpells()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range spells {
		if s.Slug == "magic-missile" {
			mm = s.ID
			break
		}
	}
	if mm == 0 {
		t.Fatal("magic-missile missing")
	}
	if err := svc.AddLearned(ch, uid, mm, true); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if err := svc.CastSpell(ch, uid, mm, rules.KindSlots, 3, 1); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if rules.SlotRemaining(ch.Resources, 3) != 1 {
		t.Fatalf("after 3rd-level cast %+v", ch.Resources)
	}
	if rules.SlotRemaining(ch.Resources, 2) != 3 {
		t.Fatalf("other slots changed %+v", ch.Resources)
	}
}

func TestMonkWarlockSeparateGroups(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := &Character{
		Name: "MC", OwnerID: uid, CampaignID: cid, Level: 6,
		STR: 8, DEX: 14, CON: 13, INT: 10, WIS: 15, CHA: 12,
		RaceID: 1, BackgroundID: 1, HPMax: 30, HPCurrent: 30, ProficiencyBonus: 3,
	}
	id, err := svc.Repo.InsertLive(ch, []rules.ClassProgress{
		{ClassID: 7, Levels: 2},
		{ClassID: 12, Levels: 4},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	groups := resourceGroups(got.Resources)
	if len(groups) != 2 {
		t.Fatalf("want pact + ki groups, got %+v", groups)
	}
	if groups[0].Kind != rules.KindPact || groups[1].Kind != rules.KindKi {
		t.Fatalf("group order %+v", groups)
	}
	if groups[0].Pools[0].Max != 2 || groups[0].Pools[0].SlotLevel != 2 {
		t.Fatalf("pact group %+v", groups[0])
	}
	if groups[1].Pools[0].Max != 2 {
		t.Fatalf("ki group %+v", groups[1])
	}
}

func TestMagicMissileRollThroughService(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Wiz", 2, 5)
	var mm int64
	spells, _ := svc.Catalog.ListSpells()
	for _, s := range spells {
		if s.Slug == "magic-missile" {
			mm = s.ID
		}
	}
	if err := svc.AddLearned(ch, uid, mm, true); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	res, sp, err := svc.RollSpell(ch, mm, 1)
	if err != nil {
		t.Fatal(err)
	}
	if sp.Slug != "magic-missile" {
		t.Fatalf("spell %+v", sp)
	}
	if res.Total < 6 || res.Total > 15 {
		t.Fatalf("3d4+3 total %d", res.Total)
	}
}

func TestArmorOfAgathysSetsTempHP(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Wiz", 2, 1)
	var id int64
	spells, _ := svc.Catalog.ListSpells()
	for _, s := range spells {
		if s.Slug == "armor-of-agathys" {
			id = s.ID
		}
	}
	if id == 0 {
		t.Fatal("armor-of-agathys missing")
	}
	if err := svc.AddLearned(ch, uid, id, true); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if err := svc.CastSpell(ch, uid, id, rules.KindSlots, 1, 1); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPTemp != 5 {
		t.Fatalf("temp hp %d", ch.HPTemp)
	}
	if len(ch.Effects) != 1 || ch.Effects[0].Slug != "armor-of-agathys" {
		t.Fatalf("effects %+v", ch.Effects)
	}
	st := rules.DeriveCombat(ch.CombatInput())
	if st.TempHP != 5 {
		t.Fatalf("combat temp %d", st.TempHP)
	}
}

func TestTempHPStacksAcrossSources(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Wiz", 2, 1)
	ag, _ := rules.SpellCombatEffect("armor-of-agathys", "Armor of Agathys", "Доспех Агатиса", 1, 5)
	if err := svc.Repo.UpsertEffect(ch.ID, ag); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.UpsertEffect(ch.ID, rules.OtherTempEffect(7)); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HPTemp != 12 {
		t.Fatalf("stacked temp %d", got.HPTemp)
	}
	if err := svc.DismissEffect(got, uid, got.Effects[0].ID); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ch.ID)
	if rules.SumTempHP(got.Effects) != 7 {
		t.Fatalf("after dismiss sum %d %+v", rules.SumTempHP(got.Effects), got.Effects)
	}
}
