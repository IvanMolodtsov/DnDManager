package characters

import (
	"strings"

	"dndmanager/internal/rules"
)

// StatusSpec is the DM add-status payload (sheet or battle board).
type StatusSpec struct {
	CatalogID         int64
	NameEN            string
	NameRU            string
	DamageFormula     string
	DamageType        string
	ACBonus           int
	DurationTurns     int
	RemoveOnBattleEnd bool
	Hidden            bool
	SourceURL         string
}

func (s *Service) AddStatus(ch *Character, userID int64, spec StatusSpec) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	e, err := s.BuildStatus(spec)
	if err != nil {
		return err
	}
	if err := s.Repo.UpsertEffect(ch.ID, e); err != nil {
		return err
	}
	return s.syncTempFromEffects(ch)
}

func (s *Service) EndTurnStatuses(ch *Character, userID int64) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	return s.replaceEffects(ch, rules.EndTurnEffects(ch.Effects))
}

func (s *Service) StartTurnStatuses(ch *Character, userID int64) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	id := ch.ID
	snapshot := append([]rules.Effect{}, ch.Effects...)
	for _, e := range snapshot {
		if !e.TicksDamage() {
			continue
		}
		live, err := s.Get(id)
		if err != nil {
			return err
		}
		rolled, err := rules.RollFormula(e.DamageFormula)
		if err != nil {
			return err
		}
		if rolled.Total < 1 {
			continue
		}
		if _, err := s.ApplyHPDamage(live, userID, rolled.Total, e.DamageType); err != nil {
			return err
		}
	}
	got, err := s.Get(id)
	if err != nil {
		return err
	}
	*ch = *got
	return nil
}

func (s *Service) ClearBattleEndEffects(ch *Character, userID int64) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	var keep []rules.Effect
	for _, e := range ch.Effects {
		if e.RemoveOnBattleEnd {
			continue
		}
		keep = append(keep, e)
	}
	return s.replaceEffects(ch, keep)
}

func (s *Service) BuildStatus(spec StatusSpec) (rules.Effect, error) {
	if spec.DurationTurns < 0 {
		spec.DurationTurns = 0
	}
	if spec.DurationTurns > 99 {
		spec.DurationTurns = 99
	}
	spec.DamageFormula = strings.TrimSpace(spec.DamageFormula)
	spec.DamageType = strings.ToLower(strings.TrimSpace(spec.DamageType))
	if spec.DamageFormula != "" && spec.DamageType != "" && !rules.ValidDamageType(spec.DamageType) {
		return rules.Effect{}, rules.ErrDamageType
	}
	e := rules.Effect{
		Kind:              "condition",
		Source:            rules.SourceOther,
		Hidden:            spec.Hidden,
		RemoveOnBattleEnd: spec.RemoveOnBattleEnd,
		DurationTurns:     spec.DurationTurns,
		DamageFormula:     spec.DamageFormula,
		DamageType:        spec.DamageType,
		ACBonus:           spec.ACBonus,
		SourceURL:         rules.SanitizeSourceURL(spec.SourceURL),
		NameEN:            strings.TrimSpace(spec.NameEN),
		NameRU:            strings.TrimSpace(spec.NameRU),
	}
	if spec.CatalogID > 0 {
		c, err := s.Catalog.Condition(spec.CatalogID)
		if err != nil {
			return rules.Effect{}, err
		}
		e.Slug = c.Slug
		if e.NameEN == "" {
			e.NameEN = c.NameEN
		}
		if e.NameRU == "" {
			e.NameRU = c.NameRU
		}
		if e.SourceURL == "" {
			if c.SourceURLRU != "" {
				e.SourceURL = c.SourceURLRU
			} else {
				e.SourceURL = c.SourceURL
			}
		}
		if e.DamageFormula == "" {
			e.DamageFormula = c.DamageFormula
			e.DamageType = c.DamageType
		}
	}
	if e.NameEN == "" {
		e.NameEN = e.NameRU
	}
	if e.NameRU == "" {
		e.NameRU = e.NameEN
	}
	if e.NameEN == "" {
		return rules.Effect{}, ErrStatusName
	}
	if e.Slug == "" {
		e.Slug = rules.CustomStatusSlug(e.NameEN)
	}
	e.FormulaEN, e.FormulaRU = rules.EffectFormula(e)
	return e, nil
}
