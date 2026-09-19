package battles

import (
	"sort"
	"strconv"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
)

func sortUnitsByInitiative(units []Unit) []Unit {
	var masters []Unit
	pets := map[int64][]Unit{}
	var orphans []Unit
	for _, u := range units {
		if u.IsFollower() {
			if u.CharacterID != 0 {
				pets[u.CharacterID] = append(pets[u.CharacterID], u)
			} else {
				orphans = append(orphans, u)
			}
			continue
		}
		masters = append(masters, u)
	}
	sort.SliceStable(masters, func(i, j int) bool {
		return unitLess(masters[i], masters[j])
	})
	for owner := range pets {
		sort.SliceStable(pets[owner], func(i, j int) bool {
			a, b := pets[owner][i], pets[owner][j]
			if a.CompanionID != b.CompanionID {
				if a.CompanionID == 0 || b.CompanionID == 0 {
					return a.CompanionID > 0
				}
				return a.CompanionID < b.CompanionID
			}
			return a.ID < b.ID
		})
	}
	sort.SliceStable(orphans, func(i, j int) bool { return unitLess(orphans[i], orphans[j]) })
	out := make([]Unit, 0, len(units))
	for _, m := range masters {
		out = append(out, m)
		for _, p := range pets[m.CharacterID] {
			p.Initiative = m.Initiative
			out = append(out, p)
		}
		delete(pets, m.CharacterID)
	}
	for _, leftover := range pets {
		out = append(out, leftover...)
	}
	out = append(out, orphans...)
	for i := range out {
		out[i].SortOrder = i
	}
	return out
}

func unitLess(a, b Unit) bool {
	if a.Initiative != b.Initiative {
		return a.Initiative > b.Initiative
	}
	if a.DEX != b.DEX {
		return a.DEX > b.DEX
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.ID < b.ID
}

func (s *Service) SyncCompanions(campaignID, characterID int64) {
	b, err := s.Repo.FindByCampaign(campaignID)
	if err != nil {
		return
	}
	if b.Status != StatusSetupInit && b.Status != StatusFighting {
		return
	}
	got, err := s.load(b.ID)
	if err != nil {
		return
	}
	if err := s.ensureCompanionUnits(got, characterID); err != nil {
		return
	}
	fresh, err := s.load(got.ID)
	if err != nil {
		return
	}
	sorted := sortUnitsByInitiative(fresh.Units)
	if err := s.persistUnits(sorted); err != nil {
		return
	}
	fresh.Units = sorted
	if fresh.Fighting() {
		if au := got.ActiveUnit(); au != nil {
			fresh.ActiveIndex = activeIndexFor(sorted, au.ID)
		}
		_ = s.Repo.Update(fresh)
	}
	s.broadcast(campaignID)
}

func (s *Service) ensureCompanionUnits(b *Battle, onlyCharacterID int64) error {
	have := map[int64]bool{}
	for _, u := range b.Units {
		if u.IsCompanion() && u.CompanionID != 0 {
			have[u.CompanionID] = true
		}
	}
	pcs, err := s.Characters.ListLiveByCampaign(b.CampaignID)
	if err != nil {
		return err
	}
	live := map[int64]bool{}
	for _, ch := range pcs {
		for _, row := range ch.Companions {
			live[row.ID] = true
		}
		if onlyCharacterID != 0 && ch.ID != onlyCharacterID {
			continue
		}
		masterInit := 0
		for _, u := range b.Units {
			if u.IsPC() && u.CharacterID == ch.ID {
				masterInit = u.Initiative
				break
			}
		}
		for _, row := range ch.Companions {
			if have[row.ID] {
				continue
			}
			u := unitFromCompanion(b.ID, ch, row, masterInit)
			id, err := s.Repo.InsertUnit(&u)
			if err != nil {
				return err
			}
			u.ID = id
			b.Units = append(b.Units, u)
		}
	}
	for _, u := range b.Units {
		if u.IsCompanion() && u.CompanionID != 0 && !live[u.CompanionID] {
			_ = s.Repo.DeleteUnit(u.ID)
		}
	}
	return nil
}

func unitFromCompanion(battleID int64, ch *characters.Character, row characters.Companion, init int) Unit {
	u := Unit{
		BattleID: battleID, Kind: KindCompanion,
		CharacterID: ch.ID, CompanionID: row.ID, CatalogMonsterID: row.CatalogMonsterID,
		Name: row.DisplayName("en"), Initiative: init,
		HPCurrent: row.HPCurrent, HPMax: row.HPMax, AC: row.AC,
		ResistJSON: row.ResistJSON, SourceURLRU: row.ArticleURL(),
		Dead: row.HPCurrent == 0, OwnerUserID: ch.OwnerID,
	}
	if strings.TrimSpace(u.Name) == "" {
		u.Name = row.Name
	}
	u.SetScores(row.Scores())
	return u
}

func (s *Service) AddSummon(campaignID, userID, casterID, catalogID int64, qty int, spellSlug, lang string) (*Battle, error) {
	b, err := s.memberBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	ch, err := s.Characters.Get(casterID)
	if err != nil {
		return nil, err
	}
	if err := s.Characters.RequireCombatEdit(ch, userID); err != nil {
		return nil, ErrForbidden
	}
	gate := characters.GateCompanions(ch)
	wantType := catalog.ConjureType(spellSlug)
	if wantType == "" {
		return nil, ErrCompanionSpell
	}
	okSpell := false
	for _, slug := range gate.Conjure {
		if slug == spellSlug {
			okSpell = true
			break
		}
	}
	if !okSpell {
		return nil, ErrCompanionSpell
	}
	if qty < 1 || qty > 8 {
		return nil, ErrQty
	}
	m, err := s.Catalog.Monster(catalogID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(m.Type), wantType) {
		return nil, ErrCompanionType
	}
	masterInit := 0
	for _, u := range b.Units {
		if u.IsPC() && u.CharacterID == ch.ID {
			masterInit = u.Initiative
			break
		}
	}
	base := m.Name(lang)
	for i := 1; i <= qty; i++ {
		name := base
		if i > 1 {
			name = base + " " + strconv.Itoa(i)
		}
		u := Unit{
			BattleID: b.ID, Kind: KindSummon, CharacterID: ch.ID, CatalogMonsterID: m.ID,
			Name: name, Initiative: masterInit,
			HPCurrent: m.FightHP(), HPMax: m.FightHP(), AC: m.FightAC(),
			ResistJSON: snapshotFromMonster(m), SourceURLRU: m.ArticleURL(),
			OwnerUserID: ch.OwnerID, CR: m.CR, CRLabel: strings.TrimSpace(m.CRLabel),
		}
		u.SetScores(scoresFromMonster(m))
		if _, err := s.Repo.InsertUnit(&u); err != nil {
			return nil, err
		}
	}
	if err := s.insertFollowersIntoOrder(b); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) insertFollowersIntoOrder(b *Battle) error {
	fresh, err := s.load(b.ID)
	if err != nil {
		return err
	}
	activeID := int64(0)
	if au := b.ActiveUnit(); au != nil {
		activeID = au.ID
	} else if au := fresh.ActiveUnit(); au != nil {
		activeID = au.ID
	}
	sorted := sortUnitsByInitiative(fresh.Units)
	if err := s.persistUnits(sorted); err != nil {
		return err
	}
	fresh.Units = sorted
	fresh.ActiveIndex = activeIndexFor(sorted, activeID)
	return s.Repo.Update(fresh)
}

func (s *Service) memberBattle(campaignID, userID int64) (*Battle, error) {
	if err := s.requireMember(campaignID, userID); err != nil {
		return nil, err
	}
	b, err := s.Repo.FindByCampaign(campaignID)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.load(b.ID)
}

func (s *Service) copyInitiativeToFollowers(b *Battle, master *Unit) error {
	if master == nil || !master.IsPC() {
		return nil
	}
	for i := range b.Units {
		u := &b.Units[i]
		if !u.IsFollower() || u.CharacterID != master.CharacterID {
			continue
		}
		u.Initiative = master.Initiative
		if err := s.Repo.UpdateUnit(u); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) persistCompanionHP(u *Unit) {
	if u == nil || !u.IsCompanion() || u.CompanionID == 0 || u.CharacterID == 0 {
		return
	}
	ch, err := s.Characters.Get(u.CharacterID)
	if err != nil {
		return
	}
	_ = s.Characters.ApplyCompanionHP(ch, ch.OwnerID, u.CompanionID, u.HPCurrent, u.HPMax)
}

func (s *Service) requireUnitCombatEdit(campaignID, userID, unitID int64) (*Battle, *Unit, error) {
	b, err := s.memberBattle(campaignID, userID)
	if err != nil {
		return nil, nil, err
	}
	if b.Status != StatusFighting {
		return nil, nil, ErrWrongStatus
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, nil, err
	}
	if s.Campaigns.IsDM(campaignID, userID) {
		return b, u, nil
	}
	if u.CanCombatHP(false, userID) {
		return b, u, nil
	}
	return nil, nil, ErrForbidden
}
