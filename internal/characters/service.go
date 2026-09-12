package characters

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

var (
	ErrNameRequired = errors.New("character name required")
	ErrNotFound     = errors.New("character not found")
	ErrForbidden    = errors.New("forbidden")
	ErrNotOwner     = errors.New("not character owner")
)

// Service hydrates sheets, persists drafts, and applies confirmed progression.
type Service struct {
	Repo      *Repository
	Campaigns *campaigns.Service
	Catalog   *catalog.Service
	Rules     *rules.Engine
}

func (s *Service) Get(id int64) (*Character, error) {
	ch, err := s.Repo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.hydrate(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Service) ListByOwner(ownerID int64) ([]Character, error) {
	list, err := s.Repo.ListByOwner(ownerID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		_ = s.hydrate(&list[i])
	}
	return list, nil
}

func (s *Service) ListByCampaign(campaignID int64) ([]campaigns.CharacterSummary, error) {
	list, err := s.Repo.ListByCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]campaigns.CharacterSummary, 0, len(list))
	for _, ch := range list {
		_ = s.hydrate(&ch)
		out = append(out, campaigns.CharacterSummary{
			ID:      ch.ID,
			Name:    ch.Name,
			Level:   ch.Level,
			Owner:   ch.OwnerName,
			OwnerID: ch.OwnerID,
		})
	}
	return out, nil
}

// CanView allows the owner (editable) or the campaign DM (read-only).
func (s *Service) CanView(ch *Character, userID int64) (readonly bool, err error) {
	if ch.OwnerID == userID {
		return false, nil
	}
	if s.Campaigns.IsDM(ch.CampaignID, userID) {
		return true, nil
	}
	return false, ErrForbidden
}

func (s *Service) RequireOwner(ch *Character, userID int64) error {
	if ch.OwnerID != userID {
		return ErrNotOwner
	}
	return nil
}

func (s *Service) EnsureDraft(ownerID, campaignID int64) (*Draft, error) {
	if _, err := s.Campaigns.RequireMember(campaignID, ownerID); err != nil {
		return nil, err
	}
	d, err := s.Repo.GetDraft(ownerID, campaignID)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	d = &Draft{OwnerID: ownerID, CampaignID: campaignID, Step: 1, BaseScores: defaultArray()}
	id, err := s.Repo.InsertDraft(d)
	if err != nil {
		return nil, err
	}
	d.ID = id
	return d, nil
}

func (s *Service) SaveDraft(d *Draft) error {
	if d.Step < 1 {
		d.Step = 1
	}
	if d.Step > 4 {
		d.Step = 4
	}
	return s.Repo.UpdateDraft(d)
}

func (s *Service) ResetDraft(d *Draft) error {
	*d = Draft{ID: d.ID, OwnerID: d.OwnerID, CampaignID: d.CampaignID, Step: 1, BaseScores: defaultArray()}
	return s.Repo.UpdateDraft(d)
}

// ConfirmDraft runs IntentCreate and inserts the live character.
func (s *Service) ConfirmDraft(d *Draft, ownerID int64, lang string) (*Character, error) {
	if d.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(d.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return nil, ErrNameRequired
	}
	if _, err := s.Campaigns.RequireMember(d.CampaignID, ownerID); err != nil {
		return nil, err
	}
	delta, err := s.Rules.Preview(rules.State{}, rules.Intent{
		Kind:         rules.IntentCreate,
		RaceID:       d.RaceID,
		BackgroundID: d.BackgroundID,
		ClassID:      d.ClassID,
		SubclassID:   d.SubclassID,
		BaseScores:   d.BaseScores,
		Lang:         lang,
	})
	if err != nil {
		return nil, err
	}
	ch := &Character{
		Name:             name,
		OwnerID:          ownerID,
		CampaignID:       d.CampaignID,
		Level:            delta.LevelAfter,
		STR:              delta.ScoresAfter.STR,
		DEX:              delta.ScoresAfter.DEX,
		CON:              delta.ScoresAfter.CON,
		INT:              delta.ScoresAfter.INT,
		WIS:              delta.ScoresAfter.WIS,
		CHA:              delta.ScoresAfter.CHA,
		RaceID:           d.RaceID,
		BackgroundID:     d.BackgroundID,
		HPMax:            delta.HPAfter,
		HPCurrent:        delta.HPAfter,
		ProficiencyBonus: delta.ProficiencyAfter,
	}
	ids := featureIDs(delta)
	id, err := s.Repo.InsertLive(ch, delta.ClassLevels, ids)
	if err != nil {
		return nil, err
	}
	_ = s.Repo.DeleteDraft(d.ID)
	return s.Get(id)
}

func (s *Service) PreviewCreate(d *Draft, lang string) (*rules.Delta, error) {
	return s.Rules.Preview(rules.State{}, rules.Intent{
		Kind:         rules.IntentCreate,
		RaceID:       d.RaceID,
		BackgroundID: d.BackgroundID,
		ClassID:      d.ClassID,
		SubclassID:   d.SubclassID,
		BaseScores:   d.BaseScores,
		Lang:         lang,
	})
}

func (s *Service) PreviewLevelUp(ch *Character, in rules.Intent) (*rules.Delta, error) {
	in.Kind = rules.IntentLevelUp
	return s.Rules.Preview(ch.State(), in)
}

// ApplyLevelUp previews IntentLevelUp and writes the new scores, HP, and features.
func (s *Service) ApplyLevelUp(ch *Character, ownerID int64, in rules.Intent) (*Character, error) {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return nil, err
	}
	in.Kind = rules.IntentLevelUp
	delta, err := s.Rules.Preview(ch.State(), in)
	if err != nil {
		return nil, err
	}
	hpCurrent := ch.HPCurrent + delta.HPGain
	if hpCurrent > delta.HPAfter {
		hpCurrent = delta.HPAfter
	}
	next := &Character{
		Level:            delta.LevelAfter,
		STR:              delta.ScoresAfter.STR,
		DEX:              delta.ScoresAfter.DEX,
		CON:              delta.ScoresAfter.CON,
		INT:              delta.ScoresAfter.INT,
		WIS:              delta.ScoresAfter.WIS,
		CHA:              delta.ScoresAfter.CHA,
		HPMax:            delta.HPAfter,
		HPCurrent:        hpCurrent,
		ProficiencyBonus: delta.ProficiencyAfter,
	}
	if err := s.Repo.ApplyProgress(ch.ID, next, delta.ClassLevels, featureIDs(delta)); err != nil {
		return nil, err
	}
	return s.Get(ch.ID)
}

func (s *Service) CreationBonuses(d *Draft) rules.AbilityScores {
	var b rules.AbilityScores
	if d.RaceID != 0 {
		if race, err := s.Catalog.Race(d.RaceID); err == nil {
			b = b.Add(rules.FromMap(race.AbilityBonuses))
		}
	}
	if d.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(d.BackgroundID); err == nil {
			b = b.Add(rules.FromMap(bg.AbilityBonuses))
		}
	}
	if d.ClassID != 0 {
		if class, err := s.Catalog.Class(d.ClassID); err == nil {
			b = b.Add(rules.FromMap(class.AbilityBonuses))
		}
	}
	return b
}

func (s *Service) BonusLines(d *Draft, lang string) []BonusLine {
	var lines []BonusLine
	if d.RaceID != 0 {
		if race, err := s.Catalog.Race(d.RaceID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Race", "Раса"),
				Text:  formatBonusMap(race.Name(lang), race.AbilityBonuses),
				URL:   race.Source(lang),
			})
		}
	}
	if d.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(d.BackgroundID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Background", "Предыстория"),
				Text:  formatBonusMap(bg.Name(lang), bg.AbilityBonuses),
				URL:   bg.Source(lang),
			})
		}
	}
	if d.ClassID != 0 {
		if class, err := s.Catalog.Class(d.ClassID); err == nil {
			lines = append(lines, BonusLine{
				Label: catalog.Pick(lang, "Class", "Класс"),
				Text:  formatBonusMap(class.Name(lang), class.AbilityBonuses),
				URL:   class.Source(lang),
			})
		}
	}
	return lines
}

func (s *Service) hydrate(ch *Character) error {
	lang := "en"
	if ch.RaceID != 0 {
		if race, err := s.Catalog.Race(ch.RaceID); err == nil {
			ch.Race = race
		}
	}
	if ch.BackgroundID != 0 {
		if bg, err := s.Catalog.Background(ch.BackgroundID); err == nil {
			ch.Background = bg
		}
	}
	rows, err := s.Repo.ListClassLevels(ch.ID)
	if err != nil {
		return err
	}
	var classes []rules.ClassProgress
	for _, row := range rows {
		cl := rules.ClassProgress{ClassID: row.ClassID, Levels: row.Levels, SubclassID: row.SubclassID}
		if class, err := s.Catalog.Class(row.ClassID); err == nil {
			cl.Slug = class.Slug
			cl.ClassName = class.Name(lang)
			cl.HitDie = class.HitDie
			cl.SourceURL = class.Source(lang)
			cl.ClassNameEN = class.NameEN
			cl.ClassNameRU = class.NameRU
			cl.SourceEN = class.SourceURL
			cl.SourceRU = class.SourceURLRU
		}
		if row.SubclassID != 0 {
			if sub, err := s.Catalog.Subclass(row.SubclassID); err == nil {
				cl.SubclassSlug = sub.Slug
				cl.Subclass = sub.Name(lang)
				cl.SubclassEN = sub.NameEN
				cl.SubclassRU = sub.NameRU
			}
		}
		classes = append(classes, cl)
	}
	ch.ClassLevels = classes
	fids, err := s.Repo.ListFeatureIDs(ch.ID)
	if err != nil {
		return err
	}
	var feats []rules.FeatureGrant
	for _, id := range fids {
		f, err := s.Catalog.Feature(id)
		if err != nil {
			continue
		}
		feats = append(feats, rules.FeatureGrant{
			ID:        f.ID,
			Name:      f.Name(lang),
			SourceURL: f.Source(lang),
			Kind:      f.SourceKind,
			Level:     f.Level,
			NameEN:    f.NameEN,
			NameRU:    f.NameRU,
			SourceEN:  f.SourceURL,
			SourceRU:  f.SourceURLRU,
		})
	}
	ch.Features = feats
	if err := s.loadSpells(ch); err != nil {
		return err
	}
	return s.syncResources(ch)
}

// Localize fills class/feature display names from EN/RU catalog fields.
func (s *Service) Localize(ch *Character, lang string) {
	if ch.Race != nil {
		_ = lang
	}
	for i := range ch.ClassLevels {
		cl := &ch.ClassLevels[i]
		if cl.ClassNameEN != "" {
			cl.ClassName = catalog.Pick(lang, cl.ClassNameEN, cl.ClassNameRU)
		}
		if cl.SubclassEN != "" {
			cl.Subclass = catalog.Pick(lang, cl.SubclassEN, cl.SubclassRU)
		}
		if cl.SourceEN != "" {
			cl.SourceURL = catalog.Pick(lang, cl.SourceEN, cl.SourceRU)
		}
	}
	for i := range ch.Features {
		f := &ch.Features[i]
		if f.NameEN != "" {
			f.Name = catalog.Pick(lang, f.NameEN, f.NameRU)
			f.SourceURL = catalog.Pick(lang, f.SourceEN, f.SourceRU)
		}
	}
}

func defaultArray() rules.AbilityScores {
	return rules.AbilityScores{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 10, CHA: 8}
}

func featureIDs(d *rules.Delta) []int64 {
	ids := make([]int64, 0, len(d.FeaturesAdded))
	for _, f := range d.FeaturesAdded {
		ids = append(ids, f.ID)
	}
	return ids
}

func formatBonusMap(name string, m map[string]int) string {
	parts := make([]string, 0, 6)
	for _, k := range rules.AbilityKeys {
		if n := m[k]; n != 0 {
			sign := "+"
			if n < 0 {
				sign = ""
			}
			parts = append(parts, strings.ToUpper(k)+" "+sign+itoa(n))
		}
	}
	if len(parts) == 0 {
		return name
	}
	return name + " (" + strings.Join(parts, ", ") + ")"
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// ErrorKey maps a characters or rules error to a locales JSON key.
func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrNameRequired):
		return "error.character.name"
	case errors.Is(err, campaigns.ErrNotMember):
		return "error.not.member"
	case errors.Is(err, rules.ErrRaceRequired):
		return "error.wizard.race"
	case errors.Is(err, rules.ErrBackgroundRequired):
		return "error.wizard.background"
	case errors.Is(err, rules.ErrClassRequired):
		return "error.wizard.class"
	case errors.Is(err, rules.ErrSubclassRequired), errors.Is(err, rules.ErrSubclassInvalid):
		return "error.wizard.subclass"
	case errors.Is(err, rules.ErrArrayInvalid):
		return "error.wizard.array"
	case errors.Is(err, rules.ErrASIRequired), errors.Is(err, rules.ErrASIInvalid):
		return "error.wizard.asi"
	case errors.Is(err, rules.ErrHPRoll):
		return "error.wizard.hp"
	case errors.Is(err, rules.ErrMaxLevel):
		return "error.level.max"
	case errors.Is(err, ErrNotOwner), errors.Is(err, ErrForbidden):
		return "error.forbidden"
	case errors.Is(err, rules.ErrResourceEmpty):
		return "error.resource.empty"
	case errors.Is(err, rules.ErrSpellNotPrepared):
		return "error.spell.not_prepared"
	case errors.Is(err, rules.ErrSpellNotLearned):
		return "error.spell.not_learned"
	case errors.Is(err, rules.ErrBadFormula):
		return "error.spell.formula"
	default:
		return "error.generic"
	}
}

func (s *Service) loadSpells(ch *Character) error {
	rows, err := s.Repo.ListCharacterSpells(ch.ID)
	if err != nil {
		return err
	}
	var out []LearnedSpell
	for _, row := range rows {
		sp, err := s.Catalog.Spell(row.SpellID)
		if err != nil {
			continue
		}
		out = append(out, LearnedSpell{Spell: *sp, Prepared: row.Prepared})
	}
	ch.Spells = out
	return nil
}

// TODO: long rest — restore all pool Current to Max, and allow rewriting which
// learned spells are marked prepared (PHB prepare-after-rest). Not in this pass.
func (s *Service) syncResources(ch *Character) error {
	maxes := rules.MaxResources(ch.ClassRules())
	cur, err := s.Repo.ListResources(ch.ID)
	if err != nil {
		return err
	}
	next := rules.SyncPools(cur, maxes)
	if !rules.PoolsEqual(cur, next) {
		if err := s.Repo.ReplaceResources(ch.ID, next); err != nil {
			return err
		}
	}
	ch.Resources = next
	return nil
}

func (s *Service) learned(ch *Character, spellID int64) (LearnedSpell, bool) {
	for _, ls := range ch.Spells {
		if ls.Spell.ID == spellID {
			return ls, true
		}
	}
	return LearnedSpell{}, false
}

func (s *Service) AddLearned(ch *Character, ownerID, spellID int64, prepared bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, err := s.Catalog.Spell(spellID); err != nil {
		return err
	}
	return s.Repo.UpsertCharacterSpell(ch.ID, spellID, prepared)
}

func (s *Service) RemoveLearned(ch *Character, ownerID, spellID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	return s.Repo.DeleteCharacterSpell(ch.ID, spellID)
}

func (s *Service) SetPrepared(ch *Character, ownerID, spellID int64, prepared bool) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if _, ok := s.learned(ch, spellID); !ok {
		return rules.ErrSpellNotLearned
	}
	return s.Repo.SetPrepared(ch.ID, spellID, prepared)
}

func (s *Service) SpendResource(ch *Character, ownerID int64, kind string, slotLevel, amount int) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	next, err := rules.Consume(ch.Resources, kind, slotLevel, amount)
	if err != nil {
		return err
	}
	if err := s.Repo.ReplaceResources(ch.ID, next); err != nil {
		return err
	}
	ch.Resources = next
	return nil
}

func (s *Service) CastSpell(ch *Character, ownerID, spellID int64, kind string, slotLevel, amount int) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	ls, ok := s.learned(ch, spellID)
	if !ok {
		return rules.ErrSpellNotLearned
	}
	if !ls.Prepared {
		return rules.ErrSpellNotPrepared
	}
	if ls.Spell.Level == 0 {
		return nil
	}
	if kind == "" {
		if rules.HasKind(ch.Resources, rules.KindPact) && !rules.HasKind(ch.Resources, rules.KindSlots) {
			kind = rules.KindPact
		} else {
			kind = rules.KindSlots
		}
	}
	if kind == rules.KindPact {
		if p, ok := rules.PactPool(ch.Resources); ok {
			slotLevel = p.SlotLevel
		}
	}
	if slotLevel < ls.Spell.Level && kind == rules.KindSlots {
		slotLevel = ls.Spell.Level
	}
	return s.SpendResource(ch, ownerID, kind, slotLevel, amount)
}

func (s *Service) RollSpell(ch *Character, spellID int64, slotLevel int) (rules.RollResult, catalog.Spell, error) {
	ls, ok := s.learned(ch, spellID)
	if !ok {
		return rules.RollResult{}, catalog.Spell{}, rules.ErrSpellNotLearned
	}
	if slotLevel < 1 {
		if p, ok := rules.PactPool(ch.Resources); ok && !rules.HasKind(ch.Resources, rules.KindSlots) {
			slotLevel = p.SlotLevel
		} else {
			slotLevel = ls.Spell.Level
		}
	}
	formula, _, heal := ls.Spell.FormulaAt(slotLevel, ch.Level)
	expr := formula
	if expr == "" {
		expr = heal
	}
	res, err := rules.RollFormula(expr)
	return res, ls.Spell, err
}
