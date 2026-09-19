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

func companionSvc(t *testing.T) (*Service, int64, int64) {
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
	cid, err := campRepo.Create("Table", "INV1", uid)
	if err != nil {
		t.Fatal(err)
	}
	cat := &catalog.Service{Repo: &catalog.Repository{DB: db}}
	svc := &Service{
		Repo:      &Repository{DB: db},
		Campaigns: &campaigns.Service{Repo: campRepo},
		Catalog:   cat,
		Rules:     &rules.Engine{Catalog: cat},
		Events:    NewVitalsHub(),
	}
	return svc, uid, cid
}

func livePC(t *testing.T, svc *Service, uid, cid int64, classes []rules.ClassProgress, feats []int64) *Character {
	t.Helper()
	ch := &Character{
		Name: "Hero", OwnerID: uid, CampaignID: cid, Level: 3,
		STR: 12, DEX: 14, CON: 13, INT: 10, WIS: 15, CHA: 8,
		RaceID: 1, BackgroundID: 1, HPMax: 24, HPCurrent: 24, ProficiencyBonus: 2,
	}
	id, err := svc.Repo.InsertLive(ch, classes, feats)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBeastMasterGateAndCRCap(t *testing.T) {
	svc, uid, cid := companionSvc(t)
	fighter := livePC(t, svc, uid, cid, []rules.ClassProgress{{ClassID: 1, Levels: 3}}, nil)
	if GateCompanions(fighter).BeastMaster {
		t.Fatal("fighter should not get beast master")
	}
	if _, err := svc.AddCompanion(fighter, uid, AddCompanionInput{Kind: KindBeast, CatalogMonsterID: mustMonster(t, svc, "wolf")}); err != ErrCompanionGate {
		t.Fatalf("fighter add: %v", err)
	}

	ranger := livePC(t, svc, uid, cid, []rules.ClassProgress{{ClassID: 9, Levels: 3, SubclassID: 16}}, nil)
	g := GateCompanions(ranger)
	if !g.BeastMaster {
		t.Fatalf("beast master gate %+v classes %+v", g, ranger.ClassLevels)
	}
	wolfID := mustMonster(t, svc, "wolf")
	row, err := svc.AddCompanion(ranger, uid, AddCompanionInput{Kind: KindBeast, CatalogMonsterID: wolfID})
	if err != nil {
		t.Fatal(err)
	}
	if row.Kind != KindBeast || row.CatalogMonsterID != wolfID {
		t.Fatalf("%+v", row)
	}
	if _, err := svc.AddCompanion(ranger, uid, AddCompanionInput{Kind: KindBeast, CatalogMonsterID: wolfID}); err != ErrCompanionLimit {
		t.Fatalf("second beast: %v", err)
	}
	ranger, _ = svc.Get(ranger.ID)
	if err := svc.RemoveCompanion(ranger, uid, row.ID); err != nil {
		t.Fatal(err)
	}
	ranger, _ = svc.Get(ranger.ID)
	dragonID := mustMonster(t, svc, "adult-red-dragon")
	if _, err := svc.AddCompanion(ranger, uid, AddCompanionInput{Kind: KindBeast, CatalogMonsterID: dragonID}); err != ErrCompanionCR {
		t.Fatalf("dragon: %v", err)
	}
	axeID := mustMonster(t, svc, "axe-beak")
	if _, err := svc.AddCompanion(ranger, uid, AddCompanionInput{Kind: KindBeast, CatalogMonsterID: axeID}); err != ErrCompanionCR {
		t.Fatalf("large axe beak: %v", err)
	}
}

func TestFamiliarGateUnicodeAndChainAttack(t *testing.T) {
	svc, uid, cid := companionSvc(t)
	ch := livePC(t, svc, uid, cid, []rules.ClassProgress{{ClassID: 2, Levels: 1}}, nil)
	if GateCompanions(ch).Familiar {
		t.Fatal("wizard without find-familiar")
	}
	fam, err := svc.Catalog.SpellBySlug("find-familiar")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.UpsertCharacterSpell(ch.ID, fam.ID, true); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if !GateCompanions(ch).Familiar || GateCompanions(ch).Chain {
		t.Fatalf("familiar gate %+v", GateCompanions(ch))
	}
	owlID := mustMonster(t, svc, "owl")
	row, err := svc.AddCompanion(ch, uid, AddCompanionInput{Kind: KindFamiliar, CatalogMonsterID: owlID})
	if err != nil {
		t.Fatal(err)
	}
	if row.CanAttack {
		t.Fatal("PHB familiar cannot attack")
	}
	impID := mustMonster(t, svc, "imp")
	if _, err := svc.AddCompanion(ch, uid, AddCompanionInput{Kind: KindFamiliar, CatalogMonsterID: impID}); err != ErrCompanionLimit && err != ErrCompanionType {
		t.Fatalf("imp without chain: %v", err)
	}
	if err := svc.RemoveCompanion(ch, uid, row.ID); err != nil {
		t.Fatal(err)
	}
	if !tryAddFeature(t, svc, ch.ID, 701) {
		t.Fatal("attach pact-of-the-chain")
	}
	ch, _ = svc.Get(ch.ID)
	if !GateCompanions(ch).Chain {
		t.Fatalf("expected chain after feature, feats %+v", ch.Features)
	}
	imp, err := svc.AddCompanion(ch, uid, AddCompanionInput{Kind: KindFamiliar, CatalogMonsterID: impID})
	if err != nil {
		t.Fatal(err)
	}
	if !imp.CanAttack {
		t.Fatal("chain imp should attack")
	}
}

func tryAddFeature(t *testing.T, svc *Service, characterID, featureID int64) bool {
	t.Helper()
	_, err := svc.Repo.DB.Exec(`INSERT OR IGNORE INTO character_features (character_id, feature_id) VALUES (?, ?)`, characterID, featureID)
	return err == nil
}

func TestItemCompanionRequiresEquip(t *testing.T) {
	svc, uid, cid := companionSvc(t)
	ch := livePC(t, svc, uid, cid, []rules.ClassProgress{{ClassID: 1, Levels: 1}}, nil)
	if GateCompanions(ch).Item {
		t.Fatal("no item yet")
	}
	sword, err := svc.Catalog.ItemBySlug("dancing-sword")
	if err != nil {
		t.Fatal(err)
	}
	it, err := svc.AddItem(ch, uid, AddItemInput{CatalogID: sword.ID, EquipNow: true, EquipSlot: "main_hand"})
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	g := GateCompanions(ch)
	if !g.Item || !g.Weapon {
		t.Fatalf("dancing sword gate %+v item %+v", g, it)
	}
	row, err := svc.AddCompanion(ch, uid, AddCompanionInput{Kind: KindWeapon, Name: "Singing blade", HPMax: 20, AC: 16})
	if err != nil {
		t.Fatal(err)
	}
	if row.Name != "Singing blade" || row.HPMax != 20 {
		t.Fatalf("%+v", row)
	}
}

func mustMonster(t *testing.T, svc *Service, slug string) int64 {
	t.Helper()
	m, err := svc.Catalog.MonsterBySlug(slug)
	if err != nil {
		t.Fatal(slug, err)
	}
	return m.ID
}
