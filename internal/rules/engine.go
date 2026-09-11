package rules

import (
	"sort"

	"dndmanager/internal/catalog"
)

func (e *Engine) previewCreate(in Intent) (*Delta, error) {
	if in.RaceID == 0 {
		return nil, ErrRaceRequired
	}
	if in.BackgroundID == 0 {
		return nil, ErrBackgroundRequired
	}
	if in.ClassID == 0 {
		return nil, ErrClassRequired
	}
	if err := ValidateStandardArray(in.BaseScores); err != nil {
		return nil, err
	}

	race, err := e.Catalog.Race(in.RaceID)
	if err != nil {
		return nil, ErrCatalog
	}
	bg, err := e.Catalog.Background(in.BackgroundID)
	if err != nil {
		return nil, ErrCatalog
	}
	class, err := e.Catalog.Class(in.ClassID)
	if err != nil {
		return nil, ErrCatalog
	}

	subclassID, sub, err := e.resolveSubclass(class, 1, 0, in.SubclassID)
	if err != nil {
		return nil, err
	}

	bonuses := FromMap(race.AbilityBonuses).Add(FromMap(bg.AbilityBonuses)).Add(FromMap(class.AbilityBonuses))
	scores := in.BaseScores.Add(bonuses)
	conMod := Modifier(scores.CON)
	hp := class.HitDie + conMod + race.HPBonusPerLevel
	if hp < 1 {
		hp = 1
	}

	features, err := e.gatherCreateFeatures(in.RaceID, in.BackgroundID, class.ID, subclassID, in.Lang)
	if err != nil {
		return nil, err
	}

	progress := ClassProgress{
		ClassID:    class.ID,
		ClassName:  class.Name(in.Lang),
		Levels:     1,
		SubclassID: subclassID,
		HitDie:     class.HitDie,
		SourceURL:  class.Source(in.Lang),
	}
	if sub != nil {
		progress.Subclass = sub.Name(in.Lang)
	}

	return &Delta{
		LevelAfter:       1,
		ProficiencyAfter: ProficiencyBonus(1),
		ScoresAfter:      scores,
		BonusScores:      bonuses,
		HPAfter:          hp,
		HPGain:           hp,
		HitDie:           class.HitDie,
		AverageHP:        AverageHitPoints(class.HitDie),
		ClassLevels:      []ClassProgress{progress},
		FeaturesAdded:    features,
		SubclassNeeded:   class.SubclassLevel <= 1,
	}, nil
}

func (e *Engine) previewLevelUp(current State, in Intent) (*Delta, error) {
	if current.Level >= 20 {
		return nil, ErrMaxLevel
	}
	if in.ClassID == 0 {
		return nil, ErrClassRequired
	}
	class, err := e.Catalog.Class(in.ClassID)
	if err != nil {
		return nil, ErrCatalog
	}
	race, err := e.Catalog.Race(current.RaceID)
	if err != nil {
		return nil, ErrCatalog
	}

	existing := current.ClassLevel(in.ClassID)
	newClassLevel := existing.Levels + 1
	totalAfter := current.Level + 1

	keepSubclass := existing.SubclassID
	subclassID, sub, err := e.resolveSubclass(class, newClassLevel, keepSubclass, in.SubclassID)
	if err != nil {
		return nil, err
	}

	asiRequired := class.HasASI(newClassLevel)
	var asi AbilityScores
	if asiRequired {
		// TODO: feat picker — at ASI levels the player may take a feat instead of +2.
		// No feat UI in this slice; only Ability Score Improvement (+2, splittable into +1/+1).
		if err := ValidateASI(in.ASI); err != nil {
			return nil, err
		}
		asi = in.ASI
	}

	conBefore := Modifier(current.Scores.CON)
	hpChoice, err := hitPointGain(class.HitDie, in.HPMode, in.HPRoll)
	if err != nil {
		return nil, err
	}
	hpGain := hpChoice + conBefore + race.HPBonusPerLevel
	if hpGain < 1 {
		hpGain = 1
	}

	scores := current.Scores.Add(asi)
	conAfter := Modifier(scores.CON)
	if extra := (conAfter - conBefore) * totalAfter; extra != 0 {
		hpGain += extra
	}

	features, err := e.gatherLevelFeatures(class.ID, subclassID, newClassLevel, keepSubclass, current, in.Lang)
	if err != nil {
		return nil, err
	}

	classes := upsertClass(current.Classes, ClassProgress{
		ClassID:    class.ID,
		ClassName:  class.Name(in.Lang),
		Levels:     newClassLevel,
		SubclassID: subclassID,
		HitDie:     class.HitDie,
		SourceURL:  class.Source(in.Lang),
	}, sub, in.Lang)

	return &Delta{
		LevelAfter:       totalAfter,
		ProficiencyAfter: ProficiencyBonus(totalAfter),
		ScoresAfter:      scores,
		ASIApplied:       asi,
		HPAfter:          current.HPMax + hpGain,
		HPGain:           hpGain,
		HitDie:           class.HitDie,
		AverageHP:        AverageHitPoints(class.HitDie),
		ClassLevels:      classes,
		FeaturesAdded:    features,
		ASIRequired:      asiRequired,
		SubclassNeeded:   class.SubclassLevel == newClassLevel && keepSubclass == 0,
	}, nil
}

func (e *Engine) resolveSubclass(class *catalog.Class, classLevel int, existingID, requestedID int64) (int64, *catalog.Subclass, error) {
	need := class.SubclassLevel <= classLevel && existingID == 0
	id := requestedID
	if id == 0 {
		id = existingID
	}
	if need && id == 0 {
		return 0, nil, ErrSubclassRequired
	}
	if id == 0 {
		return 0, nil, nil
	}
	sub, err := e.Catalog.Subclass(id)
	if err != nil {
		return 0, nil, ErrCatalog
	}
	if sub.ClassID != class.ID {
		return 0, nil, ErrSubclassInvalid
	}
	if class.SubclassLevel > classLevel {
		return 0, nil, nil
	}
	return id, sub, nil
}

func (e *Engine) gatherCreateFeatures(raceID, bgID, classID, subclassID int64, lang string) ([]FeatureGrant, error) {
	var out []FeatureGrant
	groups := []struct {
		kind string
		id   int64
		lv   int
	}{
		{catalog.KindRace, raceID, 1},
		{catalog.KindBackground, bgID, 1},
		{catalog.KindClass, classID, 1},
	}
	if subclassID != 0 {
		groups = append(groups, struct {
			kind string
			id   int64
			lv   int
		}{catalog.KindSubclass, subclassID, 1})
	}
	for _, g := range groups {
		feats, err := e.Catalog.FeaturesAt(g.kind, g.id, g.lv)
		if err != nil {
			return nil, err
		}
		for _, f := range feats {
			out = append(out, grant(f, lang))
		}
	}
	return out, nil
}

func (e *Engine) gatherLevelFeatures(classID, subclassID int64, newClassLevel int, previousSubclass int64, current State, lang string) ([]FeatureGrant, error) {
	var out []FeatureGrant
	add := func(kind string, id int64, level int) error {
		if id == 0 {
			return nil
		}
		feats, err := e.Catalog.FeaturesAt(kind, id, level)
		if err != nil {
			return err
		}
		for _, f := range feats {
			if current.HasFeature(f.ID) {
				continue
			}
			out = append(out, grant(f, lang))
		}
		return nil
	}
	if err := add(catalog.KindClass, classID, newClassLevel); err != nil {
		return nil, err
	}
	if subclassID != 0 {
		if err := add(catalog.KindSubclass, subclassID, newClassLevel); err != nil {
			return nil, err
		}
		if previousSubclass == 0 {
			for lv := 1; lv < newClassLevel; lv++ {
				if err := add(catalog.KindSubclass, subclassID, lv); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

func grant(f catalog.Feature, lang string) FeatureGrant {
	return FeatureGrant{
		ID:        f.ID,
		Name:      f.Name(lang),
		NameEN:    f.NameEN,
		NameRU:    f.NameRU,
		SourceURL: f.Source(lang),
		SourceEN:  f.SourceURL,
		SourceRU:  f.SourceURLRU,
		Kind:      f.SourceKind,
		Level:     f.Level,
	}
}

func upsertClass(existing []ClassProgress, next ClassProgress, sub *catalog.Subclass, lang string) []ClassProgress {
	if sub != nil {
		next.Subclass = sub.Name(lang)
	}
	out := make([]ClassProgress, 0, len(existing)+1)
	found := false
	for _, c := range existing {
		if c.ClassID == next.ClassID {
			if next.Subclass == "" {
				next.Subclass = c.Subclass
			}
			out = append(out, next)
			found = true
			continue
		}
		out = append(out, c)
	}
	if !found {
		out = append(out, next)
	}
	return out
}

func hitPointGain(hitDie int, mode string, roll int) (int, error) {
	switch mode {
	case "", HPAverage:
		return AverageHitPoints(hitDie), nil
	case HPRoll:
		if roll < 1 || roll > hitDie {
			return 0, ErrHPRoll
		}
		return roll, nil
	default:
		return 0, ErrHPRoll
	}
}

func ValidateStandardArray(a AbilityScores) error {
	got := []int{a.STR, a.DEX, a.CON, a.INT, a.WIS, a.CHA}
	sort.Ints(got)
	want := append([]int{}, StandardArray...)
	sort.Ints(want)
	for i := range want {
		if got[i] != want[i] {
			return ErrArrayInvalid
		}
	}
	return nil
}

func ValidateASI(a AbilityScores) error {
	if a.STR < 0 || a.DEX < 0 || a.CON < 0 || a.INT < 0 || a.WIS < 0 || a.CHA < 0 {
		return ErrASIInvalid
	}
	if a.Sum() != 2 {
		return ErrASIRequired
	}
	return nil
}
