package rules

import "testing"

func TestWarlock4PactSlots(t *testing.T) {
	pools := MaxResources([]ClassLevel{{Slug: "warlock", Levels: 4}})
	pact, ok := PactPool(pools)
	if !ok || pact.Max != 2 || pact.SlotLevel != 2 {
		t.Fatalf("warlock 4 pact: %+v ok=%v (want 2×2nd)", pact, ok)
	}
	if HasKind(pools, KindSlots) {
		t.Fatalf("warlock-only must not use the full-caster slot table: %+v", pools)
	}
}

func TestWarlock5PactIsThird(t *testing.T) {
	pools := MaxResources([]ClassLevel{{Slug: "warlock", Levels: 5}})
	pact, ok := PactPool(pools)
	if !ok || pact.Max != 2 || pact.SlotLevel != 3 {
		t.Fatalf("warlock 5 pact: %+v (want 2×3rd)", pact)
	}
}

func TestWizard5HasThirdLevelSlots(t *testing.T) {
	pools := MaxResources([]ClassLevel{{Slug: "wizard", Levels: 5}})
	if SlotRemaining(pools, 3) != 2 {
		t.Fatalf("wizard 5 third-level slots: %+v", pools)
	}
	if SlotRemaining(pools, 1) != 4 || SlotRemaining(pools, 2) != 3 {
		t.Fatalf("wizard 5 lower slots: %+v", pools)
	}
	if SlotRemaining(pools, 4) != 0 {
		t.Fatalf("wizard 5 should not have 4th-level slots")
	}
	if HasKind(pools, KindPact) {
		t.Fatal("wizard must not gain pact slots")
	}
}

func TestMonk2Warlock4SeparatePools(t *testing.T) {
	pools := MaxResources([]ClassLevel{
		{Slug: "monk", Levels: 2},
		{Slug: "warlock", Levels: 4},
	})
	ki, ok := FindPool(pools, KindKi, 0)
	if !ok || ki.Max != 2 {
		t.Fatalf("monk 2 ki: %+v", ki)
	}
	pact, ok := PactPool(pools)
	if !ok || pact.Max != 2 || pact.SlotLevel != 2 {
		t.Fatalf("warlock 4 pact: %+v", pact)
	}
	if HasKind(pools, KindSlots) {
		t.Fatalf("monk/warlock must not share a wizard slot table: %+v", pools)
	}
}

func TestWizardPlusWarlockKeepsPoolsApart(t *testing.T) {
	pools := MaxResources([]ClassLevel{
		{Slug: "wizard", Levels: 5},
		{Slug: "warlock", Levels: 4},
	})
	if SlotRemaining(pools, 3) != 2 {
		t.Fatalf("wizard 5 slots should ignore warlock levels: %+v", pools)
	}
	pact, ok := PactPool(pools)
	if !ok || pact.SlotLevel != 2 || pact.Max != 2 {
		t.Fatalf("pact still from warlock 4: %+v", pact)
	}
}

func TestConsumeThirdLevelSlot(t *testing.T) {
	pools := MaxResources([]ClassLevel{{Slug: "wizard", Levels: 5}})
	next, err := Consume(pools, KindSlots, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if SlotRemaining(next, 3) != 1 {
		t.Fatalf("after consume: %+v", next)
	}
	if SlotRemaining(next, 2) != 3 {
		t.Fatalf("other slot levels must stay: %+v", next)
	}
	if _, err := Consume(next, KindSlots, 3, 2); err != ErrResourceEmpty {
		t.Fatalf("want empty, got %v", err)
	}
}

func TestConsumePactDoesNotTouchSlots(t *testing.T) {
	pools := MaxResources([]ClassLevel{
		{Slug: "wizard", Levels: 5},
		{Slug: "warlock", Levels: 4},
	})
	next, err := Consume(pools, KindPact, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	pact, _ := PactPool(next)
	if pact.Current != 1 {
		t.Fatalf("pact current %d", pact.Current)
	}
	if SlotRemaining(next, 3) != 2 {
		t.Fatalf("spell slots changed: %+v", next)
	}
}

func TestKiAndSorceryAndChannel(t *testing.T) {
	pools := MaxResources([]ClassLevel{
		{Slug: "monk", Levels: 5},
		{Slug: "sorcerer", Levels: 4},
		{Slug: "cleric", Levels: 6},
	})
	if p, _ := FindPool(pools, KindKi, 0); p.Max != 5 {
		t.Fatalf("ki %+v", p)
	}
	if p, _ := FindPool(pools, KindSorcery, 0); p.Max != 4 {
		t.Fatalf("sorcery %+v", p)
	}
	if p, _ := FindPool(pools, KindChannel, 0); p.Max != 2 {
		t.Fatalf("cleric 6 CD %+v", p)
	}
}

func TestChannelDivinityMulticlassTakesBest(t *testing.T) {
	pools := MaxResources([]ClassLevel{
		{Slug: "cleric", Levels: 6},
		{Slug: "paladin", Levels: 4},
	})
	if p, _ := FindPool(pools, KindChannel, 0); p.Max != 2 {
		t.Fatalf("CD should be max(2,1)=2, got %+v", p)
	}
}

func TestSyncRaisesMaxWithoutWipingSpent(t *testing.T) {
	cur := []Pool{{Kind: KindKi, Current: 1, Max: 2}}
	max := []Pool{{Kind: KindKi, Current: 3, Max: 3}}
	next := SyncPools(cur, max)
	if next[0].Current != 2 || next[0].Max != 3 {
		t.Fatalf("sync %+v", next)
	}
}
