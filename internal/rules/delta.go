// Package rules previews character creation and level-up without writing to the database.
package rules

import (
	"dndmanager/internal/catalog"
)

const (
	IntentCreate  = "create"
	IntentLevelUp = "levelup"

	HPAverage = "average"
	HPRoll    = "roll"
)

// Catalog is the subset of catalog.Service the engine needs.
type Catalog interface {
	Race(id int64) (*catalog.Race, error)
	Background(id int64) (*catalog.Background, error)
	Class(id int64) (*catalog.Class, error)
	Subclass(id int64) (*catalog.Subclass, error)
	FeaturesAt(kind string, sourceID int64, level int) ([]catalog.Feature, error)
}

// ClassProgress is one class (and optional subclass) on a character.
type ClassProgress struct {
	ClassID      int64
	Slug         string
	ClassName    string
	ClassNameEN  string
	ClassNameRU  string
	Levels       int
	SubclassID   int64
	SubclassSlug string
	Subclass     string
	SubclassEN   string
	SubclassRU   string
	HitDie       int
	SourceURL    string
	SourceEN     string
	SourceRU     string
}

// State is a character snapshot used as Preview input.
type State struct {
	Level            int
	Scores           AbilityScores
	HPMax            int
	ProficiencyBonus int
	RaceID           int64
	BackgroundID     int64
	Classes          []ClassProgress
	FeatureIDs       []int64
}

func (s State) ClassLevel(classID int64) ClassProgress {
	for _, c := range s.Classes {
		if c.ClassID == classID {
			return c
		}
	}
	return ClassProgress{}
}

func (s State) HasFeature(id int64) bool {
	for _, fid := range s.FeatureIDs {
		if fid == id {
			return true
		}
	}
	return false
}

// Intent is a create or level-up request (class, HP choice, ASI, language).
type Intent struct {
	Kind         string
	RaceID       int64
	BackgroundID int64
	ClassID      int64
	SubclassID   int64
	BaseScores   AbilityScores
	HPMode       string
	HPRoll       int
	ASI          AbilityScores
	Lang         string
}

// FeatureGrant is a newly acquired catalog feature for display and persistence.
type FeatureGrant struct {
	ID        int64
	Name      string
	NameEN    string
	NameRU    string
	SourceURL string
	SourceEN  string
	SourceRU  string
	Kind      string
	Level     int
}

// Delta is the result of Preview: scores, HP, class levels, and new features.
type Delta struct {
	LevelAfter       int
	ProficiencyAfter int
	ScoresAfter      AbilityScores
	BonusScores      AbilityScores
	ASIApplied       AbilityScores
	HPAfter          int
	HPGain           int
	HitDie           int
	AverageHP        int
	ClassLevels      []ClassProgress
	FeaturesAdded    []FeatureGrant
	ASIRequired      bool
	SubclassNeeded   bool
}

// Engine computes create and level-up deltas from catalog data.
type Engine struct {
	Catalog Catalog
}

// Preview returns the resulting Delta without mutating current.
func (e *Engine) Preview(current State, in Intent) (*Delta, error) {
	switch in.Kind {
	case IntentCreate:
		return e.previewCreate(in)
	case IntentLevelUp:
		return e.previewLevelUp(current, in)
	default:
		return nil, ErrUnknownIntent
	}
}

// Apply builds a new State from a Delta.
func Apply(_ State, d *Delta) State {
	ids := make([]int64, 0, len(d.FeaturesAdded))
	for _, f := range d.FeaturesAdded {
		ids = append(ids, f.ID)
	}
	return State{
		Level:            d.LevelAfter,
		Scores:           d.ScoresAfter,
		HPMax:            d.HPAfter,
		ProficiencyBonus: d.ProficiencyAfter,
		Classes:          append([]ClassProgress{}, d.ClassLevels...),
		FeatureIDs:       ids,
	}
}
