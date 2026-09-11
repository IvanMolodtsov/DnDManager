package catalog

import (
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("catalog entry not found")

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

func (s *Service) ListItems() ([]Item, error)   { return s.Repo.ListItems() }
func (s *Service) ListSpells() ([]Spell, error) { return s.Repo.ListSpells() }

func (s *Service) Item(id int64) (*Item, error) {
	return wrapNotFound(s.Repo.Item(id))
}

func (s *Service) Spell(id int64) (*Spell, error) {
	return wrapNotFound(s.Repo.Spell(id))
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
