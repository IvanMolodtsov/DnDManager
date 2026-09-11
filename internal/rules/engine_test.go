package rules

import (
	"errors"
	"testing"

	"dndmanager/internal/catalog"
)

type memCat struct {
	races       map[int64]catalog.Race
	backgrounds map[int64]catalog.Background
	classes     map[int64]catalog.Class
	subclasses  map[int64]catalog.Subclass
	features    []catalog.Feature
}

func (m *memCat) Race(id int64) (*catalog.Race, error) {
	r, ok := m.races[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return &r, nil
}
func (m *memCat) Background(id int64) (*catalog.Background, error) {
	b, ok := m.backgrounds[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return &b, nil
}
func (m *memCat) Class(id int64) (*catalog.Class, error) {
	c, ok := m.classes[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return &c, nil
}
func (m *memCat) Subclass(id int64) (*catalog.Subclass, error) {
	s, ok := m.subclasses[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return &s, nil
}
func (m *memCat) FeaturesAt(kind string, sourceID int64, level int) ([]catalog.Feature, error) {
	var out []catalog.Feature
	for _, f := range m.features {
		if f.SourceKind == kind && f.SourceID == sourceID && f.Level == level {
			out = append(out, f)
		}
	}
	return out, nil
}

func fixture() *Engine {
	return &Engine{Catalog: &memCat{
		races: map[int64]catalog.Race{
			1: {ID: 1, NameEN: "Human", AbilityBonuses: map[string]int{"str": 1, "dex": 1, "con": 1, "int": 1, "wis": 1, "cha": 1}},
			2: {ID: 2, NameEN: "Hill Dwarf", AbilityBonuses: map[string]int{"con": 2, "wis": 1}, HPBonusPerLevel: 1},
		},
		backgrounds: map[int64]catalog.Background{
			1: {ID: 1, NameEN: "Acolyte"},
		},
		classes: map[int64]catalog.Class{
			1: {ID: 1, NameEN: "Fighter", HitDie: 10, SubclassLevel: 3, ASILevels: []int{4, 6, 8, 12, 14, 16, 19}},
			2: {ID: 2, NameEN: "Wizard", HitDie: 6, SubclassLevel: 2, ASILevels: []int{4, 8, 12, 16, 19}},
			3: {ID: 3, NameEN: "Cleric", HitDie: 8, SubclassLevel: 1, ASILevels: []int{4, 8, 12, 16, 19}},
		},
		subclasses: map[int64]catalog.Subclass{
			1: {ID: 1, ClassID: 1, NameEN: "Champion"},
			3: {ID: 3, ClassID: 2, NameEN: "Evocation"},
			5: {ID: 5, ClassID: 3, NameEN: "Life"},
		},
		features: []catalog.Feature{
			{ID: 20, SourceKind: catalog.KindClass, SourceID: 1, Level: 1, NameEN: "Second Wind"},
			{ID: 22, SourceKind: catalog.KindClass, SourceID: 1, Level: 2, NameEN: "Action Surge"},
			{ID: 23, SourceKind: catalog.KindClass, SourceID: 1, Level: 3, NameEN: "Martial Archetype"},
			{ID: 24, SourceKind: catalog.KindClass, SourceID: 1, Level: 4, NameEN: "ASI"},
			{ID: 50, SourceKind: catalog.KindSubclass, SourceID: 1, Level: 3, NameEN: "Improved Critical"},
			{ID: 30, SourceKind: catalog.KindClass, SourceID: 2, Level: 1, NameEN: "Spellcasting"},
			{ID: 32, SourceKind: catalog.KindClass, SourceID: 2, Level: 2, NameEN: "Arcane Tradition"},
			{ID: 52, SourceKind: catalog.KindSubclass, SourceID: 3, Level: 2, NameEN: "Sculpt Spells"},
			{ID: 40, SourceKind: catalog.KindClass, SourceID: 3, Level: 1, NameEN: "Spellcasting"},
			{ID: 56, SourceKind: catalog.KindSubclass, SourceID: 5, Level: 1, NameEN: "Disciple of Life"},
			{ID: 1, SourceKind: catalog.KindRace, SourceID: 1, Level: 1, NameEN: "Extra Language"},
			{ID: 10, SourceKind: catalog.KindBackground, SourceID: 1, Level: 1, NameEN: "Shelter"},
		},
	}}
}

func TestCreateHumanFighter(t *testing.T) {
	e := fixture()
	d, err := e.Preview(State{}, Intent{
		Kind: IntentCreate, RaceID: 1, BackgroundID: 1, ClassID: 1,
		BaseScores: AbilityScores{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 10, CHA: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.ScoresAfter.STR != 16 || d.ScoresAfter.CHA != 9 {
		t.Fatalf("racial bonuses: %+v", d.ScoresAfter)
	}
	if d.HPAfter != 10+Modifier(14) { // CON 13+1=14, mod +2
		t.Fatalf("hp=%d want %d", d.HPAfter, 12)
	}
	if d.ProficiencyAfter != 2 || d.LevelAfter != 1 {
		t.Fatalf("level/prof %d/%d", d.LevelAfter, d.ProficiencyAfter)
	}
	if d.SubclassNeeded {
		t.Fatal("fighter does not need subclass at 1")
	}
	if len(d.FeaturesAdded) < 3 {
		t.Fatalf("features: %+v", d.FeaturesAdded)
	}
}

func TestCreateClericRequiresSubclass(t *testing.T) {
	e := fixture()
	in := Intent{
		Kind: IntentCreate, RaceID: 1, BackgroundID: 1, ClassID: 3,
		BaseScores: AbilityScores{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 10, CHA: 8},
	}
	if _, err := e.Preview(State{}, in); !errors.Is(err, ErrSubclassRequired) {
		t.Fatalf("want subclass required, got %v", err)
	}
	in.SubclassID = 5
	d, err := e.Preview(State{}, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.ClassLevels[0].SubclassID != 5 {
		t.Fatal("expected life domain")
	}
}

func TestStandardArrayValidation(t *testing.T) {
	if err := ValidateStandardArray(AbilityScores{STR: 15, DEX: 15, CON: 13, INT: 12, WIS: 10, CHA: 8}); err == nil {
		t.Fatal("duplicate 15 should fail")
	}
}

func TestWizardSubclassAtTwo(t *testing.T) {
	e := fixture()
	st := mustCreate(t, e, Intent{
		Kind: IntentCreate, RaceID: 1, BackgroundID: 1, ClassID: 2,
		BaseScores: AbilityScores{STR: 8, DEX: 14, CON: 13, INT: 15, WIS: 12, CHA: 10},
	})
	_, err := e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 2, HPMode: HPAverage})
	if !errors.Is(err, ErrSubclassRequired) {
		t.Fatalf("want subclass at wizard 2, got %v", err)
	}
	d, err := e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 2, SubclassID: 3, HPMode: HPAverage})
	if err != nil {
		t.Fatal(err)
	}
	if d.LevelAfter != 2 || d.ClassLevels[0].SubclassID != 3 {
		t.Fatalf("delta %+v", d)
	}
	found := false
	for _, f := range d.FeaturesAdded {
		if f.ID == 52 {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing sculpt spells: %+v", d.FeaturesAdded)
	}
}

func TestLevelUpAverageAndRollHP(t *testing.T) {
	e := fixture()
	st := mustCreate(t, e, Intent{
		Kind: IntentCreate, RaceID: 2, BackgroundID: 1, ClassID: 1,
		BaseScores: AbilityScores{STR: 15, DEX: 10, CON: 14, INT: 8, WIS: 13, CHA: 12},
	})
	// Hill dwarf CON 14+2=16 (+3), HP at 1 = 10+3+1 = 14
	if st.HPMax != 14 {
		t.Fatalf("create hp %d", st.HPMax)
	}
	avg, err := e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage})
	if err != nil {
		t.Fatal(err)
	}
	// average d10=6 + CON 3 + dwarf 1 = 10
	if avg.HPGain != 10 {
		t.Fatalf("avg gain %d", avg.HPGain)
	}
	_, err = e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPRoll, HPRoll: 11})
	if !errors.Is(err, ErrHPRoll) {
		t.Fatalf("want roll error, got %v", err)
	}
	roll, err := e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPRoll, HPRoll: 10})
	if err != nil {
		t.Fatal(err)
	}
	if roll.HPGain != 10+3+1 {
		t.Fatalf("roll gain %d", roll.HPGain)
	}
}

func TestFighterASIAndProficiency(t *testing.T) {
	e := fixture()
	st := mustCreate(t, e, Intent{
		Kind: IntentCreate, RaceID: 1, BackgroundID: 1, ClassID: 1,
		BaseScores: AbilityScores{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 10, CHA: 8},
	})
	for lv := 2; lv <= 3; lv++ {
		in := Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage}
		if lv == 3 {
			in.SubclassID = 1
		}
		d, err := e.Preview(st, in)
		if err != nil {
			t.Fatal(err)
		}
		st = applyKeepIDs(st, d)
	}
	_, err := e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage})
	if !errors.Is(err, ErrASIRequired) {
		t.Fatalf("want ASI at 4, got %v", err)
	}
	d, err := e.Preview(st, Intent{
		Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage,
		ASI: AbilityScores{STR: 1, CON: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.ScoresAfter.STR != st.Scores.STR+1 || d.LevelAfter != 4 {
		t.Fatalf("asi %+v", d.ScoresAfter)
	}
	if d.ProficiencyAfter != 2 {
		t.Fatalf("prof at 4 should still be 2, got %d", d.ProficiencyAfter)
	}
	st = applyKeepIDs(st, d)
	d, err = e.Preview(st, Intent{Kind: IntentLevelUp, ClassID: 2, HPMode: HPAverage})
	if err != nil {
		t.Fatal(err)
	}
	if d.LevelAfter != 5 || d.ProficiencyAfter != 3 {
		t.Fatalf("multiclass total level/prof %d/%d", d.LevelAfter, d.ProficiencyAfter)
	}
	if len(d.ClassLevels) != 2 {
		t.Fatalf("expected two classes, got %+v", d.ClassLevels)
	}
}

func TestCONASIRetroactiveHP(t *testing.T) {
	e := fixture()
	st := mustCreate(t, e, Intent{
		Kind: IntentCreate, RaceID: 1, BackgroundID: 1, ClassID: 1,
		BaseScores: AbilityScores{STR: 15, DEX: 10, CON: 13, INT: 12, WIS: 14, CHA: 8},
	})
	// CON 14 (+2)
	for lv := 2; lv <= 3; lv++ {
		in := Intent{Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage}
		if lv == 3 {
			in.SubclassID = 1
		}
		d, err := e.Preview(st, in)
		if err != nil {
			t.Fatal(err)
		}
		st = applyKeepIDs(st, d)
	}
	before := st.HPMax
	d, err := e.Preview(st, Intent{
		Kind: IntentLevelUp, ClassID: 1, HPMode: HPAverage,
		ASI: AbilityScores{CON: 2}, // 14 -> 16, mod +2 -> +3
	})
	if err != nil {
		t.Fatal(err)
	}
	// average 6 + old con 2 + retro +1*4 = 12
	if d.HPGain != 6+2+4 {
		t.Fatalf("hp gain %d want 12 (before %d after %d)", d.HPGain, before, d.HPAfter)
	}
}

func TestProficiencyTable(t *testing.T) {
	if ProficiencyBonus(1) != 2 || ProficiencyBonus(4) != 2 || ProficiencyBonus(5) != 3 {
		t.Fatal("prof 1-5")
	}
	if ProficiencyBonus(9) != 4 || ProficiencyBonus(13) != 5 || ProficiencyBonus(17) != 6 {
		t.Fatal("prof 9+")
	}
}

func mustCreate(t *testing.T, e *Engine, in Intent) State {
	t.Helper()
	d, err := e.Preview(State{}, in)
	if err != nil {
		t.Fatal(err)
	}
	st := Apply(State{}, d)
	st.RaceID = in.RaceID
	st.BackgroundID = in.BackgroundID
	st.FeatureIDs = dFeatureIDs(d)
	return st
}

func applyKeepIDs(st State, d *Delta) State {
	next := Apply(st, d)
	next.RaceID = st.RaceID
	next.BackgroundID = st.BackgroundID
	ids := append([]int64{}, st.FeatureIDs...)
	ids = append(ids, dFeatureIDs(d)...)
	next.FeatureIDs = ids
	return next
}

func dFeatureIDs(d *Delta) []int64 {
	var ids []int64
	for _, f := range d.FeaturesAdded {
		ids = append(ids, f.ID)
	}
	return ids
}
