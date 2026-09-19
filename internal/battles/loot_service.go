package battles

import (
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
)

func (s *Service) lootSource() (LootSource, error) {
	src := catalogLootSource{
		magic:     map[string][]catalog.Item{},
		treasures: catalog.PHBTreasures(),
	}
	for _, rarity := range []string{"uncommon", "rare", "very rare", "legendary", "artifact"} {
		items, err := s.Catalog.LootMagicByRarity(rarity)
		if err != nil {
			return nil, err
		}
		src.magic[rarity] = items
	}
	spells, err := s.Catalog.ListSpells()
	if err != nil {
		return nil, err
	}
	src.spells = spells
	return src, nil
}

func (s *Service) GenerateLoot(campaignID, userID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusEnded {
		return nil, ErrWrongStatus
	}
	src, err := s.lootSource()
	if err != nil {
		return nil, err
	}
	got := GenerateFromUnits(b.Units, src, rand.New(rand.NewSource(time.Now().UnixNano())))
	souls := TotalSouls(b.Units)
	got.Souls = souls

	if err := s.Repo.DeleteGeneratedLoot(b.ID); err != nil {
		return nil, err
	}
	order := 10
	goldRow := LootRow{
		BattleID: b.ID, Kind: LootGold, NameEN: "Gold", NameRU: "Золото",
		Qty: got.Gold, SortOrder: 0,
	}
	if _, err := s.Repo.InsertLoot(&goldRow); err != nil {
		return nil, err
	}
	for i := range got.Rows {
		got.Rows[i].BattleID = b.ID
		got.Rows[i].SortOrder = order
		order++
		if _, err := s.Repo.InsertLoot(&got.Rows[i]); err != nil {
			return nil, err
		}
	}
	if err := s.Repo.UpsertSoulsLoot(b.ID, souls); err != nil {
		return nil, err
	}
	if !b.SoulsApplied {
		if err := s.applySouls(b.CampaignID, souls); err != nil {
			return nil, err
		}
		b.SoulsApplied = true
	}
	b.LootGenerated = true
	if err := s.Repo.Update(b); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	s.broadcastSheets(campaignID)
	return s.load(b.ID)
}

func (s *Service) applySouls(campaignID int64, gained int) error {
	if gained <= 0 {
		return nil
	}
	camp, err := s.Campaigns.Get(campaignID)
	if err != nil {
		return err
	}
	next := camp.Souls + gained
	if camp.SoulsCap < 1 {
		camp.SoulsCap = campaigns.DefaultSoulsCap
	}
	if next > camp.SoulsCap {
		next = camp.SoulsCap
	}
	if next < 0 {
		next = 0
	}
	return s.Campaigns.SetSouls(campaignID, next, camp.SoulsCap)
}

func (s *Service) broadcastSheets(campaignID int64) {
	if s == nil || s.Characters == nil {
		return
	}
	pcs, err := s.Characters.ListLiveByCampaign(campaignID)
	if err != nil {
		return
	}
	for _, ch := range pcs {
		s.Characters.BroadcastVitals(ch.ID)
	}
}

func (s *Service) AddLootItem(campaignID, userID, catalogID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusEnded {
		return nil, ErrWrongStatus
	}
	item, err := s.Catalog.Item(catalogID)
	if err != nil {
		return nil, err
	}
	row := LootRow{
		BattleID: b.ID, Kind: LootItem, NameEN: item.NameEN, NameRU: item.NameRU,
		CatalogItemID: item.ID, Qty: 1, Rarity: catalog.NormalizeRarity(item.Rarity),
		SortOrder: nextLootOrder(b.Loot),
	}
	if _, err := s.Repo.InsertLoot(&row); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) AddQuestItem(campaignID, userID int64, name string) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusEnded {
		return nil, ErrWrongStatus
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return nil, ErrLootName
	}
	row := LootRow{
		BattleID: b.ID, Kind: LootQuest, NameEN: name, NameRU: name, Qty: 1,
		SortOrder: nextLootOrder(b.Loot),
	}
	if _, err := s.Repo.InsertLoot(&row); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func (s *Service) RemoveLoot(campaignID, userID, lootID int64) (*Battle, error) {
	b, err := s.dmBattle(campaignID, userID)
	if err != nil {
		return nil, err
	}
	if b.Status != StatusEnded {
		return nil, ErrWrongStatus
	}
	row, err := s.Repo.Loot(lootID)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if row.BattleID != b.ID {
		return nil, ErrNotFound
	}
	if row.Kind == LootSouls {
		return nil, ErrWrongStatus
	}
	if err := s.Repo.DeleteLoot(lootID, b.ID); err != nil {
		return nil, err
	}
	s.broadcast(campaignID)
	return s.load(b.ID)
}

func nextLootOrder(rows []LootRow) int {
	max := 20
	for _, r := range rows {
		if r.SortOrder > max {
			max = r.SortOrder
		}
	}
	return max + 1
}
