package characters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

func TestDMApplySlashingWithResistance(t *testing.T) {
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
	ring, err := svc.Catalog.ItemBySlug("ring")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddItem(ch, playerID, AddItemInput{
		CatalogID: ring.ID, EquipNow: true, EquipSlot: "ring", CustomName: "Resist Ring",
		Features: []rules.FeatureDTO{{Stat: rules.StatResistance, Value: "slashing", Origin: rules.OriginMagical, NameEN: "Resistance"}},
	}); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPCurrent != 20 {
		t.Fatalf("start hp %d", ch.HPCurrent)
	}

	got, err := svc.ApplyHPDamage(ch, dmID, 10, "slashing")
	if err != nil {
		t.Fatal(err)
	}
	if got.Applied != 5 || !got.Resistant {
		t.Fatalf("resist result %+v", got)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPCurrent != 15 {
		t.Fatalf("after resist damage hp %d", ch.HPCurrent)
	}

	if err := svc.ApplyHPHeal(ch, dmID, 3); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPCurrent != 18 {
		t.Fatalf("after heal hp %d", ch.HPCurrent)
	}

	outsiderID, err := userRepo.Create(&users.User{Username: "nosy", PasswordHash: "x", Role: users.RolePlayer, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApplyHPDamage(ch, outsiderID, 1, "slashing"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider damage: %v", err)
	}
}

func TestApplyHPDamageSpendsTempFirst(t *testing.T) {
	svc, uid, cid := testSvc(t)
	ch := insertLeveled(t, svc, uid, cid, "Tank", 1, 1)
	if err := svc.AdjustTempHP(ch, uid, 5); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	ch.HPCurrent = 8
	if err := svc.saveVitals(ch); err != nil {
		t.Fatal(err)
	}
	ch, _ = svc.Get(ch.ID)
	got, err := svc.ApplyHPDamage(ch, uid, 10, "bludgeoning")
	if err != nil {
		t.Fatal(err)
	}
	if got.AbsorbedTemp != 5 || got.NewHP != 3 {
		t.Fatalf("temp absorb %+v", got)
	}
	ch, _ = svc.Get(ch.ID)
	if ch.HPCurrent != 3 || ch.HPTemp != 0 {
		t.Fatalf("persisted hp=%d temp=%d", ch.HPCurrent, ch.HPTemp)
	}
}

func TestVitalsHubBroadcast(t *testing.T) {
	h := NewVitalsHub()
	ch := h.Subscribe(5)
	defer h.Unsubscribe(5, ch)
	h.Broadcast(5)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected vitals signal")
	}
}

func TestVitalsSSEAndPartial(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Live", 1, 1)
	id := strconv.FormatInt(ch.ID, 10)

	r := levelUpForm(t, http.MethodGet, "/characters/"+id+"/vitals", "", uid, ch.ID)
	rec := httptest.NewRecorder()
	c.vitalsPartial(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `id="sheet-vitals"`) {
		t.Fatalf("vitals partial %d %s", rec.Code, body)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r = httptest.NewRequest(http.MethodGet, "/characters/"+id+"/events", nil)
	r.SetPathValue("id", id)
	r = r.WithContext(platform.WithUser(ctx, &platform.AuthUser{
		ID: uid, Username: "owner", Role: users.RolePlayer, Language: "en",
	}))
	stream := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		c.vitalsEvents(stream, r)
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)
	ct := stream.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		cancel()
		<-done
		t.Fatalf("content-type %q", ct)
	}
	c.Svc.broadcastVitals(ch.ID)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(stream.Body.String(), "event: vitals") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done
	out := stream.Body.String()
	if !strings.Contains(out, "event: vitals") {
		t.Fatalf("missing vitals event:\n%s", out)
	}
}

func TestHPAdjustModal(t *testing.T) {
	c, uid, cid := testController(t)
	ch := insertLeveled(t, c.Svc, uid, cid, "Modal", 1, 1)
	id := strconv.FormatInt(ch.ID, 10)
	r := levelUpForm(t, http.MethodGet, "/characters/"+id+"/combat/adjust?pool=hp&sign=-", "", uid, ch.ID)
	r.URL.RawQuery = "pool=hp&sign=-"
	rec := httptest.NewRecorder()
	c.hpAdjustModal(rec, r)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, body)
	}
	if strings.Contains(body, "template error") {
		t.Fatalf("template error:\n%s", body)
	}
	if !strings.Contains(body, `name="damage_type"`) || !strings.Contains(body, "slashing") {
		t.Fatalf("expected damage type select:\n%s", body)
	}

	r = levelUpForm(t, http.MethodGet, "/characters/"+id+"/combat/adjust?pool=hp&sign=%2B", "", uid, ch.ID)
	r.URL.RawQuery = "pool=hp&sign=%2B"
	rec = httptest.NewRecorder()
	c.hpAdjustModal(rec, r)
	body = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "template error") {
		t.Fatalf("heal modal %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "Heal") {
		t.Fatalf("expected heal title:\n%s", body)
	}
	if strings.Contains(body, `name="damage_type"`) {
		t.Fatalf("heal should not ask damage type:\n%s", body)
	}

	form := "pool=hp&sign=-&amount=10&damage_type=slashing"
	r = levelUpForm(t, http.MethodPost, "/characters/"+id+"/combat/adjust", form, uid, ch.ID)
	rec = httptest.NewRecorder()
	c.applyHPAdjust(rec, r)
	out := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(out, "template error") {
		t.Fatalf("apply damage %d %s", rec.Code, out)
	}
	if !strings.Contains(out, `id="sheet-vitals"`) || !strings.Contains(out, "10") || !strings.Contains(out, "Slashing") {
		t.Fatalf("expected vitals OOB and damage note:\n%s", out)
	}
	ch, _ = c.Svc.Get(ch.ID)
	if ch.HPCurrent != 10 {
		t.Fatalf("hp after 10 damage from 20: %d", ch.HPCurrent)
	}
}
