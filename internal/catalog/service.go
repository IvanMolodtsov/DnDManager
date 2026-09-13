package catalog

import (
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("catalog entry not found")

// Service reads catalog rows. Lookups return ErrNotFound instead of sql.ErrNoRows.
type Service struct {
	Repo *Repository
}

func (s *Service) ListRaces() ([]Race, error)             { return s.Repo.ListRaces() }
func (s *Service) ListBackgrounds() ([]Background, error) { return s.Repo.ListBackgrounds() }
func (s *Service) ListClasses() ([]Class, error)          { return s.Repo.ListClasses() }
func (s *Service) ListSubclasses(classID int64) ([]Subclass, error) {
	return s.Repo.ListSubclasses(classID)
}

func (s *Service) Race(id int64) (*Race, error) {
	return wrapNotFound(s.Repo.Race(id))
}

func (s *Service) Background(id int64) (*Background, error) {
	return wrapNotFound(s.Repo.Background(id))
}

func (s *Service) Class(id int64) (*Class, error) {
	return wrapNotFound(s.Repo.Class(id))
}

func (s *Service) Subclass(id int64) (*Subclass, error) {
	return wrapNotFound(s.Repo.Subclass(id))
}

func (s *Service) Feature(id int64) (*Feature, error) {
	return wrapNotFound(s.Repo.Feature(id))
}

func (s *Service) FeaturesAt(kind string, sourceID int64, level int) ([]Feature, error) {
	return s.Repo.FeaturesAt(kind, sourceID, level)
}

func (s *Service) ListItems() ([]Item, error) { return s.Repo.ListItems() }
func (s *Service) SearchItems(q string, limit int) ([]Item, error) {
	return s.Repo.SearchItems(q, limit)
}
func (s *Service) ListBaseWeapons() ([]Item, error) { return s.Repo.ListBaseWeapons() }
func (s *Service) ListBaseArmor() ([]Item, error)   { return s.Repo.ListBaseArmor() }
func (s *Service) ListBaseJewelry() ([]Item, error) { return s.Repo.ListBaseJewelry() }
func (s *Service) SearchBaseWeapons(q string, limit int) ([]Item, error) {
	return s.Repo.SearchBaseWeapons(q, limit)
}
func (s *Service) SearchBaseArmor(q string, limit int) ([]Item, error) {
	return s.Repo.SearchBaseArmor(q, limit)
}
func (s *Service) SearchBaseJewelry(q string, limit int) ([]Item, error) {
	return s.Repo.SearchBaseJewelry(q, limit)
}
func (s *Service) SearchBases(kind, q string, limit int) ([]Item, error) {
	return s.Repo.SearchBases(kind, q, limit)
}
func (s *Service) SearchConsumables(q string, limit int) ([]Item, error) {
	return s.Repo.SearchConsumables(q, limit)
}
func (s *Service) ListStatFeatures() ([]StatFeature, error) { return s.Repo.ListStatFeatures() }
func (s *Service) SearchStatFeatures(q string, limit int) ([]StatFeature, error) {
	return s.Repo.SearchStatFeatures(q, limit)
}
func (s *Service) StatFeature(id int64) (*StatFeature, error) {
	return wrapNotFound(s.Repo.StatFeature(id))
}
func (s *Service) ListSpells() ([]Spell, error) { return s.Repo.ListSpells() }
func (s *Service) SearchSpells(q string, limit int) ([]Spell, error) {
	return s.Repo.SearchSpells(q, limit)
}

func (s *Service) Item(id int64) (*Item, error) {
	return wrapNotFound(s.Repo.Item(id))
}

func (s *Service) ItemBySlug(slug string) (*Item, error) {
	return wrapNotFound(s.Repo.ItemBySlug(slug))
}

func (s *Service) Spell(id int64) (*Spell, error) {
	return wrapNotFound(s.Repo.Spell(id))
}

func (s *Service) SpellBySlug(slug string) (*Spell, error) {
	return wrapNotFound(s.Repo.SpellBySlug(slug))
}

func wrapNotFound[T any](v *T, err error) (*T, error) {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return v, nil
}
