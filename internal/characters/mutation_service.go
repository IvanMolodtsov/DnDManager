package characters

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

var (
	ErrDeadLocked   = errors.New("dead status is permanent")
	ErrNotDead      = campaigns.ErrNotDead
	ErrReviveSouls  = campaigns.ErrReviveSouls
	ErrNoMonster    = errors.New("no monster")
	ErrBodyPart     = errors.New("invalid body part")
	ErrMutationType = errors.New("monster type required")
	ErrMutationCR   = errors.New("invalid mutation CR")
)

func (s *Service) ensureDeadStatus(ch *Character) error {
	e := rules.DeadEffect()
	if err := s.Repo.UpsertEffect(ch.ID, e); err != nil {
		return err
	}
	effects, err := s.Repo.ListEffects(ch.ID)
	if err != nil {
		return err
	}
	ch.Effects = effects
	return nil
}

func (s *Service) Revive(campaignID, characterID, userID int64) error {
	ch, err := s.Get(characterID)
	if err != nil {
		return err
	}
	if ch.CampaignID != campaignID {
		return campaigns.ErrNotFound
	}
	if err := s.RequireDM(ch, userID); err != nil {
		return campaigns.ErrForbidden
	}
	if !ch.HasDeadStatus() {
		return ErrNotDead
	}
	camp, err := s.Campaigns.Get(campaignID)
	if err != nil {
		return err
	}
	if camp.Souls < rules.ReviveCost {
		return ErrReviveSouls
	}
	if err := s.Campaigns.AdjustSouls(campaignID, userID, -rules.ReviveCost); err != nil {
		return err
	}
	for _, e := range ch.Effects {
		if e.Slug == rules.DeadSlug {
			if err := s.Repo.DeleteEffect(ch.ID, e.ID); err != nil {
				return err
			}
		}
	}
	ch.DeathFail, ch.DeathSuccess = 0, 0
	if ch.HPCurrent < 1 {
		ch.HPCurrent = 1
	}
	if err := s.saveVitals(ch); err != nil {
		return err
	}
	s.BroadcastWallet(campaignID)
	return nil
}

func (s *Service) loadMutations(ch *Character) error {
	list, err := s.Repo.ListMutations(ch.ID)
	if err != nil {
		return err
	}
	ch.Mutations = list
	return nil
}

func (s *Service) ApplyMutation(ch *Character, userID int64, m Mutation, feats []rules.FeatureDTO) (Mutation, error) {
	if err := s.RequireDM(ch, userID); err != nil {
		return Mutation{}, err
	}
	if !rules.ValidBodyPart(m.BodyPart) {
		return Mutation{}, ErrBodyPart
	}
	m.CharacterID = ch.ID
	m.BodyPart = strings.ToLower(strings.TrimSpace(m.BodyPart))
	id, err := s.Repo.UpsertMutation(m)
	if err != nil {
		return Mutation{}, err
	}
	if err := s.Repo.ReplaceMutationFeatures(id, feats); err != nil {
		return Mutation{}, err
	}
	got, err := s.Repo.Mutation(id)
	if err != nil {
		return Mutation{}, err
	}
	if err := s.afterMutation(ch, got); err != nil {
		return Mutation{}, err
	}
	return got, nil
}

func (s *Service) AddMutationFeature(ch *Character, userID, mutationID int64, f rules.FeatureDTO) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	m, err := s.ownedMutation(ch, mutationID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(f.Stat) == "" {
		return rules.ErrAmount
	}
	if err := s.Repo.InsertMutationFeature(m.ID, f); err != nil {
		return err
	}
	got, err := s.Repo.Mutation(m.ID)
	if err != nil {
		return err
	}
	return s.afterMutation(ch, got)
}

func (s *Service) RemoveMutationFeature(ch *Character, userID, mutationID, featureID int64) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	m, err := s.ownedMutation(ch, mutationID)
	if err != nil {
		return err
	}
	if err := s.Repo.DeleteMutationFeature(m.ID, featureID); err != nil {
		return err
	}
	got, err := s.Repo.Mutation(m.ID)
	if err != nil {
		return err
	}
	return s.afterMutation(ch, got)
}

func (s *Service) RemoveMutation(ch *Character, userID, mutationID int64) error {
	if err := s.RequireDM(ch, userID); err != nil {
		return err
	}
	if _, err := s.ownedMutation(ch, mutationID); err != nil {
		return err
	}
	if err := s.Repo.DeleteMutation(ch.ID, mutationID); err != nil {
		return err
	}
	live, err := s.Get(ch.ID)
	if err != nil {
		return err
	}
	*ch = *live
	s.broadcastVitals(ch.ID)
	return nil
}

func (s *Service) ownedMutation(ch *Character, mutationID int64) (Mutation, error) {
	m, err := s.Repo.Mutation(mutationID)
	if err != nil {
		return Mutation{}, ErrNotFound
	}
	if m.CharacterID != ch.ID {
		return Mutation{}, ErrNotFound
	}
	return m, nil
}

func (s *Service) afterMutation(ch *Character, m Mutation) error {
	if err := s.unequipBlocked(ch, rules.GrantsFromFeatures(m.Features).BlockedSlots); err != nil {
		return err
	}
	live, err := s.Get(ch.ID)
	if err != nil {
		return err
	}
	*ch = *live
	s.broadcastVitals(ch.ID)
	return nil
}

func (s *Service) unequipBlocked(ch *Character, slots []string) error {
	if len(slots) == 0 {
		return nil
	}
	blocked := map[string]bool{}
	for _, sl := range slots {
		blocked[strings.ToLower(strings.TrimSpace(sl))] = true
	}
	for _, it := range ch.Inventory {
		if !it.Equipped() {
			continue
		}
		for _, taken := range rules.SlotsTaken(it.GearPiece()) {
			if !blocked[taken] {
				continue
			}
			it.EquippedSlot = ""
			it.Attuned = false
			if err := s.Repo.UpdateCharacterItem(ch.ID, it); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (s *Service) PickMutationMonster(crInt int, size, typ string) (*catalog.Monster, int, error) {
	size = strings.TrimSpace(size)
	typ = strings.TrimSpace(typ)
	if !rules.ValidCreatureSize(size) {
		return nil, 0, ErrMutationType
	}
	if !rules.ValidPHBType(typ) {
		return nil, 0, ErrMutationType
	}
	if crInt < 0 {
		crInt = 0
	}
	var last *catalog.Monster
	for cr := crInt; cr <= rules.MutationCRCap; cr++ {
		label := rules.CRLabelFromInt(cr)
		hits, err := s.Catalog.SearchMonstersFilter("", label, typ, size, 400)
		if err != nil {
			return nil, 0, err
		}
		if len(hits) == 0 {
			continue
		}
		idx, err := randIndex(len(hits))
		if err != nil {
			return nil, 0, err
		}
		m := hits[idx]
		last = &m
		return last, cr, nil
	}
	return nil, crInt, ErrNoMonster
}

func (s *Service) RollMutationFilters(maxLevel int) (cr, sizeFace, typeFace int, err error) {
	if maxLevel < 0 {
		maxLevel = 0
	}
	cr, err = rollRange(0, maxLevel)
	if err != nil {
		return 0, 0, 0, err
	}
	sizeFace, err = rollFace(6)
	if err != nil {
		return 0, 0, 0, err
	}
	typeFace, err = rollFace(15)
	return cr, sizeFace, typeFace, err
}

func (s *Service) RollBodyPart() (int, error) {
	return rollFace(6)
}

func rollFace(sides int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(sides)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}

func rollRange(min, max int) (int, error) {
	if max < min {
		max = min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

func randIndex(n int) (int, error) {
	if n <= 1 {
		return 0, nil
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}
