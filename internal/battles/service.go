package battles

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
	"dndmanager/internal/rules"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrNotFound       = errors.New("battle not found")
	ErrNotMember      = campaigns.ErrNotMember
	ErrQty            = errors.New("quantity required")
	ErrWrongStatus    = errors.New("battle is not in that step")
	ErrUnit           = errors.New("unit not found")
	ErrAlreadyDead    = errors.New("already dead")
	ErrNotDying       = errors.New("not making death saves")
	ErrStats          = errors.New("invalid combat stats")
	ErrCompanionSpell = errors.New("summon spell not available")
	ErrCompanionType  = errors.New("summon type mismatch")
	ErrLootName       = errors.New("loot name required")
)

// Service mutates encounters. View is RequireMember; start/mutate is campaign DM.
type Service struct {
	Repo       *Repository
	Campaigns  *campaigns.Service
	Catalog    *catalog.Service
	Characters *characters.Service
	Events     *Hub
}

func (s *Service) requireDM(campaignID, userID int64) error {
	if s.Campaigns.IsDM(campaignID, userID) {
		return nil
	}
	return ErrForbidden
}

func (s *Service) requireMember(campaignID, userID int64) error {
	if _, err := s.Campaigns.RequireMember(campaignID, userID); err != nil {
		if errors.Is(err, campaigns.ErrNotMember) {
			return ErrNotMember
		}
		return err
	}
	return nil
}

func (s *Service) load(id int64) (*Battle, error) {
	b, err := s.Repo.Find(id)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	units, err := s.Repo.ListUnits(id)
	if err != nil {
		return nil, err
	}
	if err := s.hydratePCs(b, units); err != nil {
		return nil, err
	}
	if err := s.attachEffects(units); err != nil {
		return nil, err
	}
	loot, err := s.Repo.ListLoot(id)
	if err != nil {
		return nil, err
	}
	b.Units = units
	b.Loot = loot
	return b, nil
}

func (s *Service) GetForCampaign(campaignID, userID int64) (*Battle, error) {
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

func (s *Service) SummaryForCampaign(campaignID int64) (*Battle, error) {
	b, err := s.Repo.FindByCampaign(campaignID)
	if err != nil {
		if isNoRows(err) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

func (s *Service) HasBattle(campaignID int64) (bool, error) {
	b, err := s.SummaryForCampaign(campaignID)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

func (s *Service) Start(campaignID, userID int64) (*Battle, error) {
	if err := s.requireDM(campaignID, userID); err != nil {
		return nil, err
	}
	if existing, err := s.Repo.FindByCampaign(campaignID); err == nil {
		return s.load(existing.ID)
	} else if !isNoRows(err) {
		return nil, err
	}
	id, err := s.Repo.Insert(campaignID, userID, StatusSetupMonsters)
	if err != nil {
		return nil, err
	}
	b, err := s.load(id)
	if err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return b, nil
}

func (s *Service) AddMonsters(campaignID, userID, catalogID int64, qty int, lang string) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupMonsters && b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	if qty < 1 || qty > 20 {
		return nil, ErrQty
	}
	m, err := s.Catalog.Monster(catalogID)
	if err != nil {
		return nil, err
	}
	hpMax, hpCur := m.FightHP(), m.FightHP()
	ac := m.FightAC()
	scores := scoresFromMonster(m)
	snap := snapshotFromMonster(m)
	source := m.ArticleURL()
	cr, crLabel := m.CR, strings.TrimSpace(m.CRLabel)
	if crLabel == "" {
		crLabel = "0"
	}
	existing := 0
	for _, u := range b.Units {
		if u.Kind == KindMonster && u.CatalogMonsterID == catalogID {
			existing++
			if existing == 1 {
				hpMax, hpCur, ac = u.HPMax, u.HPCurrent, u.AC
				scores = u.Scores()
				snap = u.ResistJSON
				cr, crLabel = u.CR, u.CRLabel
				if strings.TrimSpace(u.SourceURLRU) != "" {
					source = u.SourceURLRU
				}
			}
		}
	}
	base := m.Name(lang)
	newIDs := map[int64]bool{}
	for i := 1; i <= qty; i++ {
		n := existing + i
		u := Unit{
			BattleID: b.ID, Kind: KindMonster, CatalogMonsterID: m.ID,
			Name:      fmt.Sprintf("%s %d", base, n),
			HPCurrent: hpCur, HPMax: hpMax, AC: ac,
			ResistJSON: snap, SourceURLRU: source, SortOrder: len(b.Units) + i - 1,
			CR: cr, CRLabel: crLabel,
		}
		u.SetScores(scores)
		id, err := s.Repo.InsertUnit(&u)
		if err != nil {
			return nil, err
		}
		newIDs[id] = true
		u.ID = id
		b.Units = append(b.Units, u)
	}
	if b.Status == StatusFighting {
		if err := s.insertIntoFightOrder(b, newIDs); err != nil {
			return nil, err
		}
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) UpdateMonsterGroupStats(campaignID, userID, catalogID int64, st GroupStats) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupMonsters && b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	if !validGroupStats(st) {
		return nil, ErrStats
	}
	found := false
	var saved []string
	for i := range b.Units {
		u := &b.Units[i]
		if !u.IsMonster() || u.CatalogMonsterID != catalogID {
			continue
		}
		if !found {
			saved = ParseSnapshot(u.ResistJSON).Saves
		}
		found = true
		u.HPCurrent = st.HPCurrent
		u.HPMax = st.HPMax
		u.AC = st.AC
		u.SetScores(st.Scores)
		snap := st.Resist
		snap.Saves = saved
		u.ResistJSON = EncodeSnapshot(snap)
		if err := s.Repo.UpdateUnit(u); err != nil {
			return nil, err
		}
	}
	if !found {
		return nil, ErrUnit
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func validGroupStats(st GroupStats) bool {
	if st.HPMax < 1 || st.HPMax > 999 || st.HPCurrent < 1 || st.HPCurrent > st.HPMax {
		return false
	}
	if st.AC < 0 || st.AC > 40 {
		return false
	}
	for _, v := range []int{st.Scores.STR, st.Scores.DEX, st.Scores.CON, st.Scores.INT, st.Scores.WIS, st.Scores.CHA} {
		if v < 1 || v > 30 {
			return false
		}
	}
	return true
}

func scoresFromMonster(m *catalog.Monster) rules.AbilityScores {
	if m == nil {
		return rules.AbilityScores{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10}
	}
	return rules.AbilityScores{
		STR: m.FightSTR(), DEX: m.FightDEX(), CON: m.FightCON(),
		INT: m.FightINT(), WIS: m.FightWIS(), CHA: m.FightCHA(),
	}
}

func filterDamageTypes(values []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" || seen[v] || !rules.ValidDamageType(v) {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func (s *Service) RemoveMonsterType(campaignID, userID, catalogID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupMonsters {
		return nil, ErrWrongStatus
	}
	if err := s.Repo.DeleteUnitsByMonster(b.ID, catalogID); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) BeginInitiative(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupMonsters {
		return nil, ErrWrongStatus
	}
	pcs, err := s.Characters.ListLiveByCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	have := map[int64]bool{}
	for _, u := range b.Units {
		if u.IsPC() {
			have[u.CharacterID] = true
		}
	}
	for _, ch := range pcs {
		if have[ch.ID] {
			continue
		}
		combat := rules.DeriveCombat(ch.CombatInput())
		u := Unit{
			BattleID: b.ID, Kind: KindPC, CharacterID: ch.ID, Name: ch.Name,
			HPCurrent: ch.HPCurrent, HPMax: ch.HPMax, TempHP: ch.HPTemp,
			AC:           combat.AC,
			DeathSuccess: ch.DeathSuccess, DeathFail: ch.DeathFail,
			Dead: ch.DeathFail >= 3, Knocked: ch.HPCurrent == 0 && ch.DeathFail < 3,
			SortOrder: len(b.Units),
		}
		u.SetScores(ch.Scores())
		id, err := s.Repo.InsertUnit(&u)
		if err != nil {
			return nil, err
		}
		u.ID = id
		b.Units = append(b.Units, u)
	}
	if err := s.ensureCompanionUnits(b, 0); err != nil {
		return nil, err
	}
	b.Status = StatusSetupInit
	if err := s.Repo.Update(b); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func initiativeFormula(dex int) string {
	return fmt.Sprintf("1d20%+d", rules.Modifier(dex))
}

func rollInitiativeOnto(u *Unit) (rules.RollResult, error) {
	res, err := rules.RollFormula(initiativeFormula(u.DEX))
	if err != nil {
		return res, err
	}
	u.Initiative = res.Total
	return res, nil
}

func activeIndexFor(units []Unit, activeID int64) int {
	for _, u := range units {
		if u.ID == activeID {
			return u.SortOrder
		}
	}
	return firstAbleIndex(units)
}

func (s *Service) persistUnits(units []Unit) error {
	for i := range units {
		if err := s.Repo.UpdateUnit(&units[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) insertIntoFightOrder(b *Battle, newIDs map[int64]bool) error {
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
	for i := range fresh.Units {
		u := &fresh.Units[i]
		if !newIDs[u.ID] {
			continue
		}
		if _, err := rollInitiativeOnto(u); err != nil {
			return err
		}
		if err := s.Repo.UpdateUnit(u); err != nil {
			return err
		}
	}
	sorted := sortUnitsByInitiative(fresh.Units)
	if err := s.persistUnits(sorted); err != nil {
		return err
	}
	fresh.Units = sorted
	fresh.ActiveIndex = activeIndexFor(sorted, activeID)
	return s.Repo.Update(fresh)
}

func (s *Service) SetInitiative(campaignID, userID, unitID int64, value int) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupInit {
		return nil, ErrWrongStatus
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, err
	}
	u.Initiative = value
	if err := s.Repo.UpdateUnit(u); err != nil {
		return nil, err
	}
	if err := s.copyInitiativeToFollowers(b, u); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) RollInitiative(campaignID, userID, unitID int64) (rules.RollResult, *Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return rules.RollResult{}, nil, err
	}
	if b.Status != StatusSetupInit {
		return rules.RollResult{}, nil, ErrWrongStatus
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return rules.RollResult{}, nil, err
	}
	res, err := rollInitiativeOnto(u)
	if err != nil {
		return res, nil, err
	}
	if err := s.Repo.UpdateUnit(u); err != nil {
		return res, nil, err
	}
	if err := s.copyInitiativeToFollowers(b, u); err != nil {
		return res, nil, err
	}
	s.broadcast(campaignID)
	got, err := s.load(b.ID)
	return res, got, err
}

func (s *Service) RollMonsterInitiatives(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupInit {
		return nil, ErrWrongStatus
	}
	for i := range b.Units {
		u := &b.Units[i]
		if !u.IsMonster() {
			continue
		}
		if _, err := rollInitiativeOnto(u); err != nil {
			return nil, err
		}
		if err := s.Repo.UpdateUnit(u); err != nil {
			return nil, err
		}
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) ConfirmInitiative(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusSetupInit {
		return nil, ErrWrongStatus
	}
	units := sortUnitsByInitiative(b.Units)
	if err := s.persistUnits(units); err != nil {
		return nil, err
	}
	b.Units = units
	b.Status = StatusFighting
	b.Round = 1
	b.ActiveIndex = firstAbleIndex(units)
	if err := s.Repo.Update(b); err != nil {
		return nil, err
	}
	got, err := s.load(b.ID)
	if err != nil {
		return nil, err
	}
	if err := s.startUnitTurn(got, got.ActiveUnit(), userID); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) Next(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	if prev := b.ActiveUnit(); prev != nil {
		if err := s.endUnitTurn(b, prev, userID); err != nil {
			return nil, err
		}
		fresh, err := s.load(b.ID)
		if err != nil {
			return nil, err
		}
		b = fresh
	}
	next, wrapped := nextAble(b.Units, b.ActiveIndex)
	b.ActiveIndex = next
	if wrapped {
		b.Round++
	}
	if err := s.Repo.Update(b); err != nil {
		return nil, err
	}
	got, err := s.load(b.ID)
	if err != nil {
		return nil, err
	}
	if err := s.startUnitTurn(got, got.ActiveUnit(), userID); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) End(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.clearBattleStatuses(b, userID); err != nil {
		return nil, err
	}
	b.Status = StatusEnded
	if err := s.Repo.Update(b); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) Dismiss(campaignID, userID int64) error {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return err
	}
	if err := s.Repo.Delete(b.ID); err != nil {
		return err
	}
	s.broadcast(campaignID)
	return nil
}

func (s *Service) MarkEscaped(campaignID, userID, unitID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, err
	}
	u.Escaped = true
	u.Knocked = false
	if err := s.Repo.UpdateUnit(u); err != nil {
		return nil, err
	}
	if b.ActiveIndex == u.SortOrder {
		next, wrapped := nextAble(b.Units, b.ActiveIndex)
		b.ActiveIndex = next
		if wrapped {
			b.Round++
		}
		if err := s.Repo.Update(b); err != nil {
			return nil, err
		}
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) ApplyUnitDamage(campaignID, userID, unitID int64, amount int, dmgType string) (rules.DamageResult, *Battle, error) {
	b, u, err := s.requireUnitCombatEdit(campaignID, userID, unitID)
	if err != nil {
		return rules.DamageResult{}, nil, err
	}
	if amount < 1 {
		return rules.DamageResult{}, nil, rules.ErrAmount
	}
	if !rules.ValidDamageType(dmgType) {
		return rules.DamageResult{}, nil, rules.ErrDamageType
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return rules.DamageResult{}, nil, err
		}
		res, err := s.Characters.ApplyHPDamage(ch, userID, amount, dmgType)
		if err != nil {
			return res, nil, err
		}
		got, err := s.load(b.ID)
		s.broadcast(campaignID)
		return res, got, err
	}
	if u.Dead {
		return rules.DamageResult{}, nil, ErrAlreadyDead
	}
	effects := []rules.Effect{}
	if u.TempHP > 0 {
		effects = append(effects, rules.OtherTempEffect(u.TempHP))
	}
	res := rules.ApplyDamage(u.HPCurrent, u.HPMax, effects, u.Grants(), amount, dmgType)
	u.HPCurrent = res.NewHP
	u.TempHP = rules.SumTempHP(res.NewEffects)
	if u.HPCurrent == 0 {
		u.Dead = true
		u.Knocked = false
	}
	if err := s.Repo.UpdateUnit(u); err != nil {
		return res, nil, err
	}
	s.persistCompanionHP(u)
	s.broadcast(campaignID)
	got, err := s.load(b.ID)
	return res, got, err
}

func (s *Service) ApplyUnitHeal(campaignID, userID, unitID int64, amount int) (*Battle, error) {
	b, u, err := s.requireUnitCombatEdit(campaignID, userID, unitID)
	if err != nil {
		return nil, err
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return nil, err
		}
		if err := s.Characters.ApplyHPHeal(ch, userID, amount); err != nil {
			return nil, err
		}
		s.broadcast(campaignID)
		return s.load(b.ID)
	}
	if u.Dead {
		return b, nil
	}
	if amount < 1 {
		return nil, rules.ErrAmount
	}
	u.HPCurrent = rules.ClampHP(u.HPCurrent+amount, u.HPMax)
	if u.HPCurrent > 0 {
		u.Dead = false
	}
	if err := s.Repo.UpdateUnit(u); err != nil {
		return nil, err
	}
	s.persistCompanionHP(u)
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) AdjustUnitTemp(campaignID, userID, unitID int64, delta int) (*Battle, error) {
	b, u, err := s.requireUnitCombatEdit(campaignID, userID, unitID)
	if err != nil {
		return nil, err
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return nil, err
		}
		if err := s.Characters.AdjustTempHP(ch, userID, delta); err != nil {
			return nil, err
		}
		s.broadcast(campaignID)
		return s.load(b.ID)
	}
	u.TempHP = rules.ClampTempHP(u.TempHP + delta)
	if err := s.Repo.UpdateUnit(u); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) RollDeathSave(campaignID, userID, unitID int64) (rules.RollResult, *Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return rules.RollResult{}, nil, err
	}
	res, err := rules.RollFormula("1d20")
	if err != nil {
		return res, nil, err
	}
	got, err := s.load(b.ID)
	return res, got, err
}

func (s *Service) RecordDeathSave(campaignID, userID, unitID int64, roll int) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusFighting {
		return nil, ErrWrongStatus
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, err
	}
	if !u.IsPC() {
		return nil, ErrNotDying
	}
	ch, err := s.Characters.Get(u.CharacterID)
	if err != nil {
		return nil, err
	}
	if err := s.Characters.RecordDeathSave(ch, userID, roll); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) Groups(b *Battle, lang string) []MonsterGroup {
	if b == nil {
		return nil
	}
	order := []int64{}
	by := map[int64]*MonsterGroup{}
	for _, u := range b.Units {
		if !u.IsMonster() {
			continue
		}
		g, ok := by[u.CatalogMonsterID]
		if !ok {
			name := u.Name
			cr := ""
			article := strings.TrimSpace(u.SourceURLRU)
			if m, err := s.Catalog.Monster(u.CatalogMonsterID); err == nil {
				name = m.Name(lang)
				cr = m.CRLabel
				if article == "" {
					article = m.ArticleURL()
				}
			}
			g = &MonsterGroup{
				CatalogID: u.CatalogMonsterID, Name: name, CRLabel: cr,
				HPCurrent: u.HPCurrent, HPMax: u.HPMax, AC: u.AC,
				SourceURLRU: article,
			}
			g.SetScores(u.Scores())
			snap := ParseSnapshot(u.ResistJSON)
			g.Resistances = snap.Resistances
			g.Immunities = snap.Immunities
			g.Vulnerabilities = snap.Vulnerabilities
			g.Saves = snap.Saves
			by[u.CatalogMonsterID] = g
			order = append(order, u.CatalogMonsterID)
		}
		g.Count++
	}
	out := make([]MonsterGroup, 0, len(order))
	for _, id := range order {
		out = append(out, *by[id])
	}
	return out
}

func (s *Service) dmBattle(campaignID, userID int64) (*Battle, error) {
	if err := s.requireDM(campaignID, userID); err != nil {
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

func (s *Service) unitIn(b *Battle, unitID int64) (*Unit, error) {
	for i := range b.Units {
		if b.Units[i].ID == unitID {
			return &b.Units[i], nil
		}
	}
	return nil, ErrUnit
}

func (s *Service) hydratePCs(b *Battle, units []Unit) error {
	owners := map[int64]int64{}
	for i := range units {
		u := &units[i]
		if !u.IsPC() || u.CharacterID == 0 {
			continue
		}
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			continue
		}
		owners[ch.ID] = ch.OwnerID
		combat := rules.DeriveCombat(ch.CombatInput())
		u.Name = ch.Name
		u.HPCurrent = ch.HPCurrent
		u.HPMax = ch.HPMax
		u.TempHP = ch.HPTemp
		u.AC = combat.AC
		u.SetScores(ch.Scores())
		u.DeathSuccess = ch.DeathSuccess
		u.DeathFail = ch.DeathFail
		u.Dead = ch.DeathFail >= 3
		u.Knocked = ch.HPCurrent == 0 && !u.Dead && !u.Escaped
		u.Effects = ch.Effects
		u.OwnerUserID = ch.OwnerID
		if err := s.Repo.UpdateUnit(u); err != nil {
			return err
		}
	}
	for i := range units {
		u := &units[i]
		if !u.IsFollower() {
			continue
		}
		u.OwnerUserID = owners[u.CharacterID]
		if u.OwnerUserID == 0 && u.CharacterID != 0 {
			if ch, err := s.Characters.Get(u.CharacterID); err == nil {
				u.OwnerUserID = ch.OwnerID
			}
		}
		if u.IsCompanion() && u.CompanionID != 0 && u.CharacterID != 0 {
			if row, err := s.Characters.Repo.Companion(u.CharacterID, u.CompanionID); err == nil {
				u.HPCurrent = row.HPCurrent
				u.HPMax = row.HPMax
				u.Name = row.DisplayName("en")
				if row.HPCurrent == 0 {
					u.Dead = true
				}
			}
		}
	}
	_ = b
	return nil
}

func (s *Service) attachEffects(units []Unit) error {
	if len(units) == 0 {
		return nil
	}
	byUnit, err := s.Repo.ListEffectsForBattle(units[0].BattleID)
	if err != nil {
		return err
	}
	for i := range units {
		if units[i].IsPC() {
			continue
		}
		units[i].Effects = byUnit[units[i].ID]
	}
	return nil
}

func (s *Service) AddUnitStatus(campaignID, userID, unitID int64, spec characters.StatusSpec) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, err
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return nil, err
		}
		if err := s.Characters.AddStatus(ch, userID, spec); err != nil {
			return nil, err
		}
		s.broadcast(campaignID)
		return s.load(b.ID)
	}
	e, err := s.Characters.BuildStatus(spec)
	if err != nil {
		return nil, err
	}
	if err := s.Repo.UpsertUnitEffect(u.ID, e); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) RemoveUnitStatus(campaignID, userID, unitID, effectID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	u, err := s.unitIn(b, unitID)
	if err != nil {
		return nil, err
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return nil, err
		}
		if err := s.Characters.DismissEffect(ch, userID, effectID); err != nil {
			return nil, err
		}
		s.broadcast(campaignID)
		return s.load(b.ID)
	}
	if err := s.Repo.DeleteUnitEffect(u.ID, effectID); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) startUnitTurn(b *Battle, u *Unit, userID int64) error {
	if b == nil || u == nil || !u.Able() {
		return nil
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return err
		}
		return s.Characters.StartTurnStatuses(ch, userID)
	}
	for _, e := range u.Effects {
		if !e.TicksDamage() {
			continue
		}
		live, err := s.unitIn(b, u.ID)
		if err != nil {
			return err
		}
		if live.Dead {
			return nil
		}
		rolled, err := rules.RollFormula(e.DamageFormula)
		if err != nil {
			return err
		}
		if rolled.Total < 1 {
			continue
		}
		if _, _, err := s.ApplyUnitDamage(b.CampaignID, userID, u.ID, rolled.Total, e.DamageType); err != nil {
			if errors.Is(err, ErrAlreadyDead) {
				return nil
			}
			return err
		}
		fresh, err := s.load(b.ID)
		if err != nil {
			return err
		}
		*b = *fresh
	}
	return nil
}

func (s *Service) endUnitTurn(b *Battle, u *Unit, userID int64) error {
	if b == nil || u == nil {
		return nil
	}
	if u.IsPC() {
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			return err
		}
		return s.Characters.EndTurnStatuses(ch, userID)
	}
	next := rules.EndTurnEffects(u.Effects)
	keep := map[string]bool{}
	for _, e := range next {
		keep[e.Slug] = true
		if err := s.Repo.UpsertUnitEffect(u.ID, e); err != nil {
			return err
		}
	}
	for _, e := range u.Effects {
		if keep[e.Slug] {
			continue
		}
		if err := s.Repo.DeleteUnitEffect(u.ID, e.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) clearBattleStatuses(b *Battle, userID int64) error {
	if b == nil {
		return nil
	}
	for _, u := range b.Units {
		if !u.IsPC() {
			continue
		}
		ch, err := s.Characters.Get(u.CharacterID)
		if err != nil {
			continue
		}
		if err := s.Characters.ClearBattleEndEffects(ch, userID); err != nil {
			return err
		}
	}
	return s.Repo.DeleteEffectsForBattle(b.ID)
}

func firstAbleIndex(units []Unit) int {
	for _, u := range units {
		if u.Able() {
			return u.SortOrder
		}
	}
	if len(units) == 0 {
		return 0
	}
	return units[0].SortOrder
}

func nextAble(units []Unit, from int) (idx int, wrapped bool) {
	n := len(units)
	if n == 0 {
		return 0, false
	}
	order := append([]Unit(nil), units...)
	sort.Slice(order, func(i, j int) bool { return order[i].SortOrder < order[j].SortOrder })
	pos := 0
	for i, u := range order {
		if u.SortOrder == from {
			pos = i
			break
		}
	}
	for step := 1; step <= n; step++ {
		j := pos + step
		if j >= n {
			j = j % n
			wrapped = true
		}
		if order[j].Able() {
			return order[j].SortOrder, wrapped
		}
	}
	return from, false
}

func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, characters.ErrForbidden):
		return "error.forbidden"
	case errors.Is(err, ErrNotFound):
		return "error.notfound"
	case errors.Is(err, ErrNotMember):
		return "error.not.member"
	case errors.Is(err, ErrQty):
		return "error.battle.qty"
	case errors.Is(err, ErrWrongStatus):
		return "error.battle.status"
	case errors.Is(err, ErrUnit):
		return "error.battle.unit"
	case errors.Is(err, ErrAlreadyDead):
		return "error.battle.dead"
	case errors.Is(err, ErrNotDying):
		return "error.battle.not_dying"
	case errors.Is(err, ErrStats):
		return "error.battle.stats"
	case errors.Is(err, characters.ErrStatusName):
		return "error.status.name"
	case errors.Is(err, rules.ErrAmount):
		return "error.hp.amount"
	case errors.Is(err, rules.ErrDamageType):
		return "error.hp.damage_type"
	case errors.Is(err, catalog.ErrNotFound):
		return "error.notfound"
	case errors.Is(err, ErrCompanionSpell):
		return "error.companion.spell"
	case errors.Is(err, ErrCompanionType), errors.Is(err, characters.ErrCompanionType):
		return "error.companion.kind"
	case errors.Is(err, ErrLootName):
		return "error.loot.name"
	default:
		return "error.generic"
	}
}
