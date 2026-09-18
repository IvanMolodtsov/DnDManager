package battles

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
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
	uid, err := userRepo.Create(&users.User{Username: "dm", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	campRepo := &campaigns.Repository{DB: db}
	cid, err := campRepo.Create("Table", "INVITE1", uid)
	if err != nil {
		t.Fatal(err)
	}
	cat := &catalog.Service{Repo: &catalog.Repository{DB: db}}
	charSvc := &characters.Service{
		Repo:      &characters.Repository{DB: db},
		Campaigns: &campaigns.Service{Repo: campRepo},
		Catalog:   cat,
		Rules:     &rules.Engine{Catalog: cat},
		Events:    characters.NewVitalsHub(),
	}
	svc := &Service{
		Repo:       &Repository{DB: db},
		Campaigns:  &campaigns.Service{Repo: campRepo},
		Catalog:    cat,
		Characters: charSvc,
		Events:     NewHub(),
	}
	return svc, uid, cid
}

func insertPC(t *testing.T, svc *Service, uid, cid int64, name string) *characters.Character {
	t.Helper()
	ch := &characters.Character{
		Name: name, OwnerID: uid, CampaignID: cid, Level: 1,
		STR: 8, DEX: 14, CON: 13, INT: 15, WIS: 10, CHA: 12,
		RaceID: 1, BackgroundID: 1, HPMax: 20, HPCurrent: 20, ProficiencyBonus: 2,
	}
	id, err := svc.Characters.Repo.InsertLive(ch, []rules.ClassProgress{{ClassID: 1, Levels: 1}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Characters.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func goblinID(t *testing.T, svc *Service) int64 {
	t.Helper()
	m, err := svc.Catalog.MonsterBySlug("goblin")
	if err != nil {
		t.Fatal(err)
	}
	return m.ID
}

func fightingWithTwoGoblins(t *testing.T, svc *Service, dmID, cid int64, pc *characters.Character) *Battle {
	t.Helper()
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, goblinID(t, svc), 2, "en"); err != nil {
		t.Fatal(err)
	}
	b, err := svc.BeginInitiative(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	var pcUnit, g1, g2 *Unit
	for i := range b.Units {
		u := &b.Units[i]
		if u.IsPC() {
			pcUnit = u
		} else if g1 == nil {
			g1 = u
		} else {
			g2 = u
		}
	}
	if pcUnit == nil || g1 == nil || g2 == nil {
		t.Fatalf("units %+v", b.Units)
	}
	if _, err := svc.SetInitiative(cid, dmID, pcUnit.ID, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetInitiative(cid, dmID, g1.ID, 15); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetInitiative(cid, dmID, g2.ID, 10); err != nil {
		t.Fatal(err)
	}
	b, err = svc.ConfirmInitiative(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAddTwoGoblins(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	b, err := svc.AddMonsters(cid, dmID, goblinID(t, svc), 2, "en")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, u := range b.Units {
		if u.IsMonster() {
			n++
			if u.HPMax != 7 || u.HPCurrent != 7 {
				t.Fatalf("goblin hp %+v", u)
			}
			if u.DEX != 14 || u.STR != 10 || u.CON != 10 {
				t.Fatalf("goblin default scores %+v", u)
			}
		}
	}
	if n != 2 {
		t.Fatalf("goblin rows %d", n)
	}
	groups := svc.Groups(b, "en")
	if len(groups) != 1 || groups[0].Count != 2 {
		t.Fatalf("groups %+v", groups)
	}
}

func TestInitiativeOrderAndDexTie(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	if b.Status != StatusFighting || b.Round != 1 {
		t.Fatalf("status %s round %d", b.Status, b.Round)
	}
	if len(b.Units) != 3 {
		t.Fatalf("units %d", len(b.Units))
	}
	if b.Units[0].Initiative < b.Units[1].Initiative {
		t.Fatalf("not sorted desc %+v", b.Units)
	}
	if !b.Units[0].IsPC() || b.ActiveIndex != b.Units[0].SortOrder {
		t.Fatalf("PC should go first %+v active %d", b.Units[0], b.ActiveIndex)
	}
}

func TestNextSkipsDeadAndEscaped(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	var g1, g2 Unit
	for _, u := range b.Units {
		if u.IsMonster() && g1.ID == 0 {
			g1 = u
		} else if u.IsMonster() {
			g2 = u
		}
	}
	if _, _, err := svc.ApplyUnitDamage(cid, dmID, g1.ID, 7, "slashing"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkEscaped(cid, dmID, g2.ID); err != nil {
		t.Fatal(err)
	}
	b, err := svc.GetForCampaign(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	start := b.ActiveIndex
	b, err = svc.Next(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	active := b.ActiveUnit()
	if active == nil || !active.IsPC() {
		t.Fatalf("should wrap to PC, active %+v from %d", active, start)
	}
	if b.Round < 2 {
		t.Fatalf("expected wrap round %d", b.Round)
	}
}

func TestPCZeroHPKnockedDeathAndHeal(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	var pcUnit Unit
	for _, u := range b.Units {
		if u.IsPC() {
			pcUnit = u
		}
	}
	if _, _, err := svc.ApplyUnitDamage(cid, dmID, pcUnit.ID, 20, "slashing"); err != nil {
		t.Fatal(err)
	}
	b, _ = svc.GetForCampaign(cid, dmID)
	for _, u := range b.Units {
		if u.IsPC() {
			pcUnit = u
		}
	}
	if !pcUnit.Knocked || pcUnit.Dead || pcUnit.HPCurrent != 0 {
		t.Fatalf("want knocked %+v", pcUnit)
	}
	if _, err := svc.RecordDeathSave(cid, dmID, pcUnit.ID, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDeathSave(cid, dmID, pcUnit.ID, 4); err != nil {
		t.Fatal(err)
	}
	ch, _ := svc.Characters.Get(pc.ID)
	if ch.DeathFail != 2 {
		t.Fatalf("fails %d", ch.DeathFail)
	}
	if err := svc.Characters.ApplyHPHeal(ch, dmID, 5); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	if ch.HPCurrent != 5 || ch.DeathFail != 0 {
		t.Fatalf("heal should stand up hp=%d fail=%d", ch.HPCurrent, ch.DeathFail)
	}
	b, _ = svc.GetForCampaign(cid, dmID)
	for _, u := range b.Units {
		if u.IsPC() && (u.Knocked || u.Dead) {
			t.Fatalf("stood up unit %+v", u)
		}
	}

	if _, _, err := svc.ApplyUnitDamage(cid, dmID, pcUnit.ID, 5, "slashing"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDeathSave(cid, dmID, pcUnit.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDeathSave(cid, dmID, pcUnit.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDeathSave(cid, dmID, pcUnit.ID, 3); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	if ch.DeathFail < 3 {
		t.Fatalf("want dead fails %d", ch.DeathFail)
	}
	if err := svc.Characters.ApplyHPHeal(ch, dmID, 10); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	if ch.HPCurrent != 0 {
		t.Fatalf("heal after death should no-op hp=%d", ch.HPCurrent)
	}
}

func TestDamageAtZeroAddsDeathFail(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	pc.HPCurrent = 0
	if err := svc.Characters.Repo.UpdateVitals(pc.ID, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	ch, _ := svc.Characters.Get(pc.ID)
	if _, err := svc.Characters.ApplyHPDamage(ch, dmID, 4, "slashing"); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	if ch.DeathFail != 1 {
		t.Fatalf("fail %d", ch.DeathFail)
	}
}

func TestMemberSeesNamesNotMonsterHP(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	fightingWithTwoGoblins(t, svc, dmID, cid, pc)

	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
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

	bundle, err := platform.LoadBundle(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(filepath.Join("..", "..", "web", "templates"), bundle)
	if err != nil {
		t.Fatal(err)
	}
	c := &Controller{Svc: svc, Campaigns: svc.Campaigns, Characters: svc.Characters, Catalog: svc.Catalog, Render: render}

	r := httptest.NewRequest(http.MethodGet, "/campaigns/"+strconv.FormatInt(cid, 10)+"/battle", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: playerID, Username: "player", Role: users.RolePlayer, Language: "en",
	}))
	rec := httptest.NewRecorder()
	c.show(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("status %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "Goblin") || !strings.Contains(body, "Hero") {
		t.Fatalf("expected names:\n%s", body)
	}
	if strings.Contains(body, "HP 7/7") {
		t.Fatalf("player should not see monster HP:\n%s", body)
	}
	for _, leak := range []string{"STR ", "DEX ", "CON ", "resist ", "immune ", "vuln ", `name="str"`, `name="resist"`, "5e14.dnd.su/bestiary/"} {
		if strings.Contains(body, leak) {
			t.Fatalf("player should not see monster stat %q:\n%s", leak, body)
		}
	}
	if !strings.Contains(body, "HP 20/20") {
		t.Fatalf("player should see PC HP:\n%s", body)
	}
	if strings.Contains(body, "Next unit") || strings.Contains(body, "End battle") {
		t.Fatalf("player should not get DM controls:\n%s", body)
	}

	if _, err := svc.AddMonsters(cid, playerID, goblinID(t, svc), 1, "en"); !errors.Is(err, ErrForbidden) && !errors.Is(err, ErrWrongStatus) {
		t.Fatalf("player mutate: %v", err)
	}
}

func TestPlayerSetupHidesMonsterCRAndEdit(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, goblinID(t, svc), 2, "en"); err != nil {
		t.Fatal(err)
	}
	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
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
	bundle, err := platform.LoadBundle(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(filepath.Join("..", "..", "web", "templates"), bundle)
	if err != nil {
		t.Fatal(err)
	}
	c := &Controller{Svc: svc, Campaigns: svc.Campaigns, Characters: svc.Characters, Catalog: svc.Catalog, Render: render}
	r := httptest.NewRequest(http.MethodGet, "/campaigns/"+strconv.FormatInt(cid, 10)+"/battle", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: playerID, Username: "player", Role: users.RolePlayer, Language: "en",
	}))
	rec := httptest.NewRecorder()
	c.show(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("status %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "Goblin") {
		t.Fatalf("expected name:\n%s", body)
	}
	for _, leak := range []string{"CR 1/4", "HP 7/7", "DEX 14", "STR ", "Edit", "resist ", `name="str"`} {
		if strings.Contains(body, leak) {
			t.Fatalf("player setup leaked %q:\n%s", leak, body)
		}
	}
}

func TestAddStubMonsterDefaultsHP(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	m, err := svc.Catalog.MonsterBySlug("spiderdragon")
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsStub {
		t.Fatalf("expected stub %+v", m)
	}
	b, err := svc.AddMonsters(cid, dmID, m.ID, 1, "en")
	if err != nil {
		t.Fatal(err)
	}
	var u *Unit
	for i := range b.Units {
		if b.Units[i].IsMonster() {
			u = &b.Units[i]
			break
		}
	}
	if u == nil || u.HPMax != 1 || u.HPCurrent != 1 {
		t.Fatalf("stub hp %+v", u)
	}
	if u.STR != 10 || u.DEX != 10 || u.CON != 10 || u.INT != 10 || u.WIS != 10 || u.CHA != 10 {
		t.Fatalf("stub scores default %+v", u)
	}
	if u.SourceURLRU != "https://5e14.dnd.su/bestiary/15712-spiderdragon/" {
		t.Fatalf("stub preview URL %s", u.SourceURLRU)
	}
}

func TestEditGoblinGroupHP(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	gid := goblinID(t, svc)
	m, err := svc.Catalog.Monster(gid)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.AddMonsters(cid, dmID, gid, 3, "en")
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.UpdateMonsterGroupStats(cid, dmID, gid, GroupStats{
		HPCurrent: 20, HPMax: 20, AC: m.FightAC(), Scores: scoresFromMonster(m),
		Resist: ParseSnapshot(snapshotFromMonster(m)),
	})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, u := range b.Units {
		if !u.IsMonster() {
			continue
		}
		n++
		if u.HPMax != 20 || u.HPCurrent != 20 {
			t.Fatalf("edited goblin hp %+v", u)
		}
		if u.AC != m.FightAC() || u.DEX != m.FightDEX() {
			t.Fatalf("ac/dex should stay catalog %+v", u)
		}
	}
	if n != 3 {
		t.Fatalf("goblin rows %d", n)
	}
	groups := svc.Groups(b, "en")
	if len(groups) != 1 || groups[0].Count != 3 || groups[0].HPMax != 20 {
		t.Fatalf("groups %+v", groups)
	}

	b, err = svc.AddMonsters(cid, dmID, gid, 1, "en")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range b.Units {
		if u.IsMonster() && (u.HPMax != 20 || u.HPCurrent != 20) {
			t.Fatalf("later add should inherit group stats %+v", u)
		}
	}

	if _, err := svc.UpdateMonsterGroupStats(cid, dmID, gid, GroupStats{
		HPCurrent: 0, HPMax: 20, AC: 15, Scores: scoresFromMonster(m),
	}); !errors.Is(err, ErrStats) {
		t.Fatalf("want ErrStats, got %v", err)
	}
}

func TestEditStubACDEX(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	m, err := svc.Catalog.MonsterBySlug("spiderdragon")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, m.ID, 1, "en"); err != nil {
		t.Fatal(err)
	}
	b, err := svc.UpdateMonsterGroupStats(cid, dmID, m.ID, GroupStats{
		HPCurrent: 87, HPMax: 87, AC: 18,
		Scores: rules.AbilityScores{STR: 20, DEX: 14, CON: 18, INT: 8, WIS: 12, CHA: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	var u *Unit
	for i := range b.Units {
		if b.Units[i].IsMonster() {
			u = &b.Units[i]
			break
		}
	}
	if u == nil || u.HPMax != 87 || u.HPCurrent != 87 || u.AC != 18 || u.DEX != 14 {
		t.Fatalf("stub stats %+v", u)
	}
	if u.STR != 20 || u.CON != 18 || u.INT != 8 || u.WIS != 12 || u.CHA != 10 {
		t.Fatalf("stub abilities %+v", u)
	}

	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
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
	if _, err := svc.UpdateMonsterGroupStats(cid, playerID, m.ID, GroupStats{
		HPCurrent: 10, HPMax: 10, AC: 12, Scores: rules.AbilityScores{STR: 12, DEX: 12, CON: 12, INT: 12, WIS: 12, CHA: 12},
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("player mutate: %v", err)
	}
}

func TestEditMonsterStatsModal(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	gid := goblinID(t, svc)
	if _, err := svc.AddMonsters(cid, dmID, gid, 2, "en"); err != nil {
		t.Fatal(err)
	}
	bundle, err := platform.LoadBundle(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(filepath.Join("..", "..", "web", "templates"), bundle)
	if err != nil {
		t.Fatal(err)
	}
	c := &Controller{Svc: svc, Campaigns: svc.Campaigns, Characters: svc.Characters, Catalog: svc.Catalog, Render: render}
	cidStr := strconv.FormatInt(cid, 10)
	midStr := strconv.FormatInt(gid, 10)
	auth := func(r *http.Request) *http.Request {
		r.SetPathValue("id", cidStr)
		return r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
			ID: dmID, Username: "dm", Role: users.RolePlayer, Language: "en",
		}))
	}

	show := httptest.NewRequest(http.MethodGet, "/campaigns/"+cidStr+"/battle", nil)
	show = auth(show)
	rec := httptest.NewRecorder()
	c.show(rec, show)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("show %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "Edit") || !strings.Contains(body, "HP 7/7") || !strings.Contains(body, "monster-stats-modal") {
		t.Fatalf("expected edit control:\n%s", body)
	}

	edit := httptest.NewRequest(http.MethodGet, "/campaigns/"+cidStr+"/battle/monsters/"+midStr+"/edit", nil)
	edit.SetPathValue("mid", midStr)
	edit = auth(edit)
	rec = httptest.NewRecorder()
	c.editMonsterStats(rec, edit)
	form := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(form, "template error") {
		t.Fatalf("edit %d %s", rec.Code, form)
	}
	if !strings.Contains(form, `name="hp_max"`) || !strings.Contains(form, `value="7"`) {
		t.Fatalf("expected catalog HP defaults:\n%s", form)
	}
	if !strings.Contains(form, `name="str"`) || !strings.Contains(form, `name="dex"`) || !strings.Contains(form, `name="resist"`) {
		t.Fatalf("expected ability and resist fields:\n%s", form)
	}
	m, err := svc.Catalog.Monster(gid)
	if err != nil {
		t.Fatal(err)
	}
	if m.ArticleURL() == "" || !strings.Contains(form, m.ArticleURL()) || !strings.Contains(form, `target="_blank"`) || !strings.Contains(form, `rel="noopener"`) {
		t.Fatalf("expected 5e14 article link %s:\n%s", m.ArticleURL(), form)
	}

	post := httptest.NewRequest(http.MethodPost, "/campaigns/"+cidStr+"/battle/monsters/"+midStr+"/edit", strings.NewReader("hp_max=20&hp_current=20&ac=15&str=8&dex=14&con=10&int=10&wis=8&cha=8&resist=fire"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.SetPathValue("mid", midStr)
	post = auth(post)
	rec = httptest.NewRecorder()
	c.saveMonsterStats(rec, post)
	roster := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(roster, "template error") {
		t.Fatalf("save %d %s", rec.Code, roster)
	}
	if !strings.Contains(roster, "HP 20/20") {
		t.Fatalf("expected updated roster HP:\n%s", roster)
	}
}

func TestGoblinEditScoresAndFireResist(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	gid := goblinID(t, svc)
	m, err := svc.Catalog.Monster(gid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, gid, 2, "en"); err != nil {
		t.Fatal(err)
	}
	st := GroupStats{
		HPCurrent: m.FightHP(), HPMax: m.FightHP(), AC: m.FightAC(),
		Scores: rules.AbilityScores{STR: 8, DEX: 16, CON: 10, INT: 10, WIS: 8, CHA: 8},
		Resist: ResistSnapshot{Resistances: []string{"fire"}},
	}
	b, err := svc.UpdateMonsterGroupStats(cid, dmID, gid, st)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range b.Units {
		if !u.IsMonster() {
			continue
		}
		if u.STR != 8 || u.DEX != 16 {
			t.Fatalf("edited scores %+v", u)
		}
		g := u.Grants()
		if len(g.Resistances) != 1 || g.Resistances[0] != "fire" {
			t.Fatalf("fire resist snapshot %+v json %s", g, u.ResistJSON)
		}
	}

	b, err = svc.AddMonsters(cid, dmID, gid, 1, "en")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, u := range b.Units {
		if !u.IsMonster() {
			continue
		}
		n++
		if u.STR != 8 || u.DEX != 16 {
			t.Fatalf("later add should inherit scores %+v", u)
		}
		if !sliceHas(u.Grants().Resistances, "fire") {
			t.Fatalf("later add should inherit fire resist %s", u.ResistJSON)
		}
	}
	if n != 3 {
		t.Fatalf("goblin rows %d", n)
	}

	if _, err := svc.BeginInitiative(cid, dmID); err != nil {
		t.Fatal(err)
	}
	b, err = svc.GetForCampaign(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	var pcUnit, gob *Unit
	for i := range b.Units {
		u := &b.Units[i]
		if u.IsPC() {
			pcUnit = u
		} else if gob == nil {
			gob = u
		}
	}
	if pcUnit == nil || gob == nil {
		t.Fatalf("units %+v", b.Units)
	}
	if _, err := svc.SetInitiative(cid, dmID, pcUnit.ID, 20); err != nil {
		t.Fatal(err)
	}
	for _, u := range b.Units {
		if u.IsMonster() {
			if _, err := svc.SetInitiative(cid, dmID, u.ID, 10); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := svc.ConfirmInitiative(cid, dmID); err != nil {
		t.Fatal(err)
	}
	res, got, err := svc.ApplyUnitDamage(cid, dmID, gob.ID, 10, "fire")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Resistant || res.Applied != 5 {
		t.Fatalf("10 fire should halve: %+v", res)
	}
	for _, u := range got.Units {
		if u.ID == gob.ID && u.HPCurrent != gob.HPMax-5 {
			t.Fatalf("hp after resist %+v applied %d", u, res.Applied)
		}
	}
}

func TestDoTStartOfTurnAndDuration(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, goblinID(t, svc), 2, "en"); err != nil {
		t.Fatal(err)
	}
	b, err := svc.BeginInitiative(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	var pcUnit, g1, g2 *Unit
	for i := range b.Units {
		u := &b.Units[i]
		if u.IsPC() {
			pcUnit = u
		} else if g1 == nil {
			g1 = u
		} else {
			g2 = u
		}
	}
	if _, err := svc.SetInitiative(cid, dmID, pcUnit.ID, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetInitiative(cid, dmID, g1.ID, 15); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetInitiative(cid, dmID, g2.ID, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddUnitStatus(cid, dmID, pcUnit.ID, characters.StatusSpec{
		NameEN: "Burning", DamageFormula: "1", DamageType: "fire", DurationTurns: 2, RemoveOnBattleEnd: true,
	}); err != nil {
		t.Fatal(err)
	}
	b, err = svc.ConfirmInitiative(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	ch, _ := svc.Characters.Get(pc.ID)
	if ch.HPCurrent != 19 {
		t.Fatalf("start-of-fight tick hp %d", ch.HPCurrent)
	}
	if _, err := svc.Next(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Next(cid, dmID); err != nil {
		t.Fatal(err)
	}
	b, err = svc.Next(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	if ch.HPCurrent != 18 {
		t.Fatalf("second turn tick hp %d", ch.HPCurrent)
	}
	still := false
	for _, e := range ch.Effects {
		if e.Slug == "custom-burning" || e.Slug == "burning" {
			still = true
			if e.DurationTurns != 1 {
				t.Fatalf("duration after second start %+v", e)
			}
		}
	}
	if !still {
		t.Fatalf("status should remain for the rest of this turn %+v", ch.Effects)
	}
	if _, err := svc.Next(cid, dmID); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Characters.Get(pc.ID)
	for _, e := range ch.Effects {
		if e.Slug == "custom-burning" || e.Slug == "burning" {
			t.Fatalf("should be gone after second end %+v", ch.Effects)
		}
	}
	_ = b
}

func TestDoTAtZeroIncrementsDeathFail(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	pc.HPCurrent = 0
	if err := svc.Characters.Repo.UpdateVitals(pc.ID, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(cid, dmID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMonsters(cid, dmID, goblinID(t, svc), 1, "en"); err != nil {
		t.Fatal(err)
	}
	b, err := svc.BeginInitiative(cid, dmID)
	if err != nil {
		t.Fatal(err)
	}
	var pcUnit *Unit
	for i := range b.Units {
		if b.Units[i].IsPC() {
			pcUnit = &b.Units[i]
		} else if _, err := svc.SetInitiative(cid, dmID, b.Units[i].ID, 5); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.SetInitiative(cid, dmID, pcUnit.ID, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddUnitStatus(cid, dmID, pcUnit.ID, characters.StatusSpec{
		NameEN: "Burning", DamageFormula: "1", DamageType: "fire", DurationTurns: 2,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmInitiative(cid, dmID); err != nil {
		t.Fatal(err)
	}
	ch, _ := svc.Characters.Get(pc.ID)
	if ch.DeathFail != 1 || ch.HPCurrent != 0 {
		t.Fatalf("dot at 0: hp=%d fail=%d", ch.HPCurrent, ch.DeathFail)
	}
}

func TestRemoveOnEndVsPermanent(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	var pcUnit *Unit
	for i := range b.Units {
		if b.Units[i].IsPC() {
			pcUnit = &b.Units[i]
			break
		}
	}
	if _, err := svc.AddUnitStatus(cid, dmID, pcUnit.ID, characters.StatusSpec{
		NameEN: "Haste", ACBonus: 2, RemoveOnBattleEnd: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddUnitStatus(cid, dmID, pcUnit.ID, characters.StatusSpec{
		NameEN: "Geas", RemoveOnBattleEnd: false,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.End(cid, dmID); err != nil {
		t.Fatal(err)
	}
	ch, _ := svc.Characters.Get(pc.ID)
	slugs := map[string]bool{}
	for _, e := range ch.Effects {
		slugs[e.Slug] = true
	}
	if slugs["custom-haste"] || slugs["haste"] {
		t.Fatalf("remove-on-end should be gone %+v", ch.Effects)
	}
	if !slugs["custom-geas"] {
		t.Fatalf("permanent should remain %+v", ch.Effects)
	}
}

func TestMonsterStatusDMOnlyHTML(t *testing.T) {
	svc, dmID, cid := testSvc(t)
	pc := insertPC(t, svc, dmID, cid, "Hero")
	b := fightingWithTwoGoblins(t, svc, dmID, cid, pc)
	var gob *Unit
	for i := range b.Units {
		if b.Units[i].IsMonster() {
			gob = &b.Units[i]
			break
		}
	}
	if _, err := svc.AddUnitStatus(cid, dmID, gob.ID, characters.StatusSpec{
		NameEN: "Burning", DamageFormula: "1", DamageType: "fire",
	}); err != nil {
		t.Fatal(err)
	}
	userRepo := &users.Repository{DB: svc.Repo.DB}
	playerID, err := userRepo.Create(&users.User{Username: "player", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	camp, _ := svc.Campaigns.Get(cid)
	if _, err := svc.Campaigns.Join(camp.InviteCode, playerID); err != nil {
		t.Fatal(err)
	}
	bundle, err := platform.LoadBundle(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatal(err)
	}
	render, err := platform.NewRenderer(filepath.Join("..", "..", "web", "templates"), bundle)
	if err != nil {
		t.Fatal(err)
	}
	c := &Controller{Svc: svc, Campaigns: svc.Campaigns, Characters: svc.Characters, Catalog: svc.Catalog, Render: render}
	r := httptest.NewRequest(http.MethodGet, "/campaigns/"+strconv.FormatInt(cid, 10)+"/battle", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: playerID, Username: "player", Role: users.RolePlayer, Language: "en",
	}))
	rec := httptest.NewRecorder()
	c.show(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("player %d %s", rec.Code, body)
	}
	if strings.Contains(body, "Burning") {
		t.Fatalf("player saw monster status:\n%s", body)
	}
	r = httptest.NewRequest(http.MethodGet, "/campaigns/"+strconv.FormatInt(cid, 10)+"/battle", nil)
	r.SetPathValue("id", strconv.FormatInt(cid, 10))
	r = r.WithContext(platform.WithUser(r.Context(), &platform.AuthUser{
		ID: dmID, Username: "dm", Role: users.RolePlayer, Language: "en",
	}))
	rec = httptest.NewRecorder()
	c.show(rec, r)
	dmBody := rec.Body.String()
	if !strings.Contains(dmBody, "Burning") {
		t.Fatalf("dm should see monster status:\n%s", dmBody)
	}
	if gob.ArticleURL() == "" || !strings.Contains(dmBody, gob.ArticleURL()) {
		t.Fatalf("dm should see monster article %s:\n%s", gob.ArticleURL(), dmBody)
	}
	if strings.Contains(body, gob.ArticleURL()) {
		t.Fatalf("player saw monster article:\n%s", body)
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe(3)
	defer h.Unsubscribe(3, ch)
	h.Broadcast(3)
	select {
	case <-ch:
	default:
		t.Fatal("expected battle signal")
	}
}
