package characters

import (
	"encoding/json"
	"errors"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

var (
	ErrCompanionGate   = errors.New("companion not available")
	ErrCompanionLimit  = errors.New("companion limit reached")
	ErrCompanionKind   = errors.New("invalid companion")
	ErrCompanionCR     = errors.New("beast companion CR too high")
	ErrCompanionType   = errors.New("companion does not match the gate")
	ErrCompanionMissing = errors.New("companion not found")
)

type AddCompanionInput struct {
	Kind             string
	CatalogMonsterID int64
	ItemID           int64
	Name             string
	AC               int
	HPMax            int
	Scores           rules.AbilityScores
	Resistances      []string
	Immunities       []string
	Vulnerabilities  []string
}

func (s *Service) loadCompanions(ch *Character) error {
	rows, err := s.Repo.ListCompanions(ch.ID)
	if err != nil {
		return err
	}
	ch.Companions = rows
	return nil
}

func (s *Service) AddCompanion(ch *Character, ownerID int64, in AddCompanionInput) (*Companion, error) {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return nil, err
	}
	if err := s.hydrate(ch); err != nil {
		return nil, err
	}
	gate := GateCompanions(ch)
	kind := strings.TrimSpace(in.Kind)
	if !gate.CanAdd(kind) {
		return nil, ErrCompanionGate
	}
	if kind == KindBeast && ch.CompanionCount(KindBeast) >= 1 {
		return nil, ErrCompanionLimit
	}
	if kind == KindFamiliar && ch.CompanionCount(KindFamiliar) >= 1 {
		return nil, ErrCompanionLimit
	}
	row := Companion{CharacterID: ch.ID, Kind: kind, ItemID: in.ItemID, SortOrder: len(ch.Companions)}
	if in.CatalogMonsterID > 0 {
		m, err := s.Catalog.Monster(in.CatalogMonsterID)
		if err != nil {
			return nil, ErrCompanionType
		}
		if err := validateCompanionMonster(kind, gate, *m); err != nil {
			return nil, err
		}
		applyMonsterSnapshot(&row, *m)
		if kind == KindFamiliar {
			row.CanAttack = gate.Chain && catalog.IsChainFamiliar(m.Slug)
		}
	} else if kind == KindWeapon {
		row.Name = strings.TrimSpace(in.Name)
		if row.Name == "" {
			row.Name = "Weapon companion"
		}
		row.Type = "construct"
		row.AC = clampAC(in.AC)
		row.HPMax = clampHPMax(in.HPMax)
		row.HPCurrent = row.HPMax
		row.SetScores(normalizeScores(in.Scores))
		row.ResistJSON = encodeResist(in.Resistances, in.Immunities, in.Vulnerabilities)
		row.CanAttack = true
		if in.ItemID == 0 {
			if id := firstWeaponCompanionItem(ch); id != 0 {
				row.ItemID = id
			}
		}
	} else {
		return nil, ErrCompanionType
	}
	if strings.TrimSpace(in.Name) != "" {
		row.Name = strings.TrimSpace(in.Name)
	}
	id, err := s.Repo.InsertCompanion(&row)
	if err != nil {
		return nil, err
	}
	row.ID = id
	s.notifyCompanion(ch)
	return &row, nil
}

func (s *Service) UpdateCompanion(ch *Character, ownerID, companionID int64, in AddCompanionInput) (*Companion, error) {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return nil, err
	}
	row, err := s.Repo.Companion(ch.ID, companionID)
	if err != nil {
		return nil, ErrCompanionMissing
	}
	if strings.TrimSpace(in.Name) != "" {
		row.Name = strings.TrimSpace(in.Name)
	}
	row.AC = clampAC(in.AC)
	row.HPMax = clampHPMax(in.HPMax)
	if in.HPMax > 0 && row.HPCurrent > row.HPMax {
		row.HPCurrent = row.HPMax
	}
	if in.HPMax > 0 && row.HPCurrent < 1 && row.HPMax > 0 {
		row.HPCurrent = row.HPMax
	}
	row.SetScores(normalizeScores(in.Scores))
	row.ResistJSON = encodeResist(in.Resistances, in.Immunities, in.Vulnerabilities)
	if err := s.Repo.UpdateCompanion(row); err != nil {
		return nil, err
	}
	s.notifyCompanion(ch)
	return row, nil
}

func (s *Service) RemoveCompanion(ch *Character, ownerID, companionID int64) error {
	if err := s.RequireOwner(ch, ownerID); err != nil {
		return err
	}
	if err := s.Repo.DeleteCompanion(ch.ID, companionID); err != nil {
		return err
	}
	s.notifyCompanion(ch)
	return nil
}

func (s *Service) ApplyCompanionHP(ch *Character, userID, companionID int64, current, max int) error {
	if err := s.RequireCombatEdit(ch, userID); err != nil {
		return err
	}
	row, err := s.Repo.Companion(ch.ID, companionID)
	if err != nil {
		return ErrCompanionMissing
	}
	if max > 0 {
		row.HPMax = clampHPMax(max)
	}
	row.HPCurrent = rules.ClampHP(current, row.HPMax)
	if row.HPCurrent == 0 {
		// persist 0 HP; battle unit marks dead
	}
	return s.Repo.UpdateCompanion(row)
}

func (s *Service) SearchCompanionMonsters(ch *Character, kind, q, cr, typ string) ([]catalog.Monster, error) {
	gate := GateCompanions(ch)
	if !gate.CanAdd(kind) {
		return nil, ErrCompanionGate
	}
	limit := 20
	if kind == KindFamiliar {
		return s.familiarHits(gate.Chain, q), nil
	}
	if kind == KindBeast {
		hits, err := s.Catalog.SearchMonsters(q, cr, "beast", 40)
		if err != nil {
			return nil, err
		}
		var out []catalog.Monster
		for _, m := range hits {
			if catalog.BeastMasterEligible(m) {
				out = append(out, m)
			}
			if len(out) >= limit {
				break
			}
		}
		return out, nil
	}
	if typ == "" && kind == KindItem {
		typ = ""
	}
	return s.Catalog.SearchMonsters(q, cr, typ, limit)
}

func (s *Service) familiarHits(chain bool, q string) []catalog.Monster {
	var out []catalog.Monster
	needle := strings.ToLower(strings.TrimSpace(q))
	for _, slug := range catalog.FamiliarSlugs(chain) {
		m, err := s.Catalog.MonsterBySlug(slug)
		if err != nil {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(m.NameEN+" "+m.NameRU+" "+m.Slug), needle) {
			continue
		}
		out = append(out, *m)
	}
	return out
}

func validateCompanionMonster(kind string, gate CompanionGate, m catalog.Monster) error {
	switch kind {
	case KindBeast:
		if !catalog.BeastMasterEligible(m) {
			return ErrCompanionCR
		}
	case KindFamiliar:
		if !catalog.IsFamiliarSlug(m.Slug, gate.Chain) {
			return ErrCompanionType
		}
	case KindItem, KindWeapon:
		// catalog snapshot optional; any fightable card is fine
	default:
		return ErrCompanionKind
	}
	return nil
}

func applyMonsterSnapshot(row *Companion, m catalog.Monster) {
	row.CatalogMonsterID = m.ID
	row.Name = m.NameEN
	row.NameEN = m.NameEN
	row.NameRU = m.NameRU
	row.Size = m.Size
	row.Type = m.Type
	row.AC = m.FightAC()
	row.HPMax = m.FightHP()
	row.HPCurrent = row.HPMax
	row.DEX = m.FightDEX()
	row.STR = m.FightSTR()
	row.CON = m.FightCON()
	row.INT = m.FightINT()
	row.WIS = m.FightWIS()
	row.CHA = m.FightCHA()
	row.SourceURL = m.SourceURL
	row.SourceURLRU = m.SourceURLRU
	row.ResistJSON = encodeResist(m.Resistances, m.Immunities, m.Vulnerabilities)
	row.CanAttack = true
}

func firstWeaponCompanionItem(ch *Character) int64 {
	for _, it := range ch.Inventory {
		if !it.Equipped() {
			continue
		}
		if hasGrantCompanion(it) || catalog.ItemGrantsCompanion(it.Item.Slug, it.Item.Kind) {
			if it.Item.Kind == "weapon" || catalog.ItemIsWeaponCompanion(it.Item.Slug, it.Item.Kind) {
				return it.ID
			}
		}
	}
	return 0
}

func encodeResist(res, imm, vuln []string) string {
	payload := struct {
		Resistances     []string `json:"resistances"`
		Immunities      []string `json:"immunities"`
		Vulnerabilities []string `json:"vulnerabilities"`
	}{nonNil(res), nonNil(imm), nonNil(vuln)}
	b, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func clampAC(n int) int {
	if n < 0 {
		return 10
	}
	if n > 40 {
		return 40
	}
	if n == 0 {
		return 10
	}
	return n
}

func clampHPMax(n int) int {
	if n < 1 {
		return 1
	}
	if n > 999 {
		return 999
	}
	return n
}

func normalizeScores(s rules.AbilityScores) rules.AbilityScores {
	keys := []int{s.STR, s.DEX, s.CON, s.INT, s.WIS, s.CHA}
	out := [6]int{}
	for i, n := range keys {
		if n < 1 {
			n = 10
		}
		if n > 30 {
			n = 30
		}
		out[i] = n
	}
	return rules.AbilityScores{STR: out[0], DEX: out[1], CON: out[2], INT: out[3], WIS: out[4], CHA: out[5]}
}

func (s *Service) notifyCompanion(ch *Character) {
	if s.OnCompanionChange != nil && ch != nil {
		s.OnCompanionChange(ch.CampaignID, ch.ID)
	}
	s.notifyCampaign(ch)
}
