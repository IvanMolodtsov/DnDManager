package characters

import (
	"errors"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/rules"
)

var ErrSoulsCap = errors.New("invalid souls cap")

func (s *Service) AdjustGold(ch *Character, userID int64, delta int) error {
	if err := s.RequireCombatEdit(ch, userID); err != nil {
		return err
	}
	if delta == 0 {
		return rules.ErrAmount
	}
	next := ch.Gold + delta
	if next < 0 {
		next = 0
	}
	if err := s.Repo.UpdateGold(ch.ID, next); err != nil {
		return err
	}
	ch.Gold = next
	s.broadcastVitals(ch.ID)
	return nil
}

func (s *Service) AdjustSouls(ch *Character, userID int64, delta int) error {
	if err := s.Campaigns.AdjustSouls(ch.CampaignID, userID, delta); err != nil {
		return mapWalletErr(err)
	}
	s.broadcastCampaignSheets(ch.CampaignID)
	return nil
}

func (s *Service) SetSoulsCap(ch *Character, userID int64, cap int) error {
	if err := s.Campaigns.SetSoulsCap(ch.CampaignID, userID, cap); err != nil {
		return mapWalletErr(err)
	}
	s.broadcastCampaignSheets(ch.CampaignID)
	return nil
}

func (s *Service) AdjustCampaignGold(campaignID, characterID, userID int64, delta int) error {
	ch, err := s.Get(characterID)
	if err != nil {
		return campaigns.ErrNotFound
	}
	if ch.CampaignID != campaignID {
		return campaigns.ErrNotFound
	}
	if err := s.RequireDM(ch, userID); err != nil {
		return campaigns.ErrForbidden
	}
	if err := s.AdjustGold(ch, userID, delta); err != nil {
		if errors.Is(err, rules.ErrAmount) {
			return campaigns.ErrAmount
		}
		if errors.Is(err, ErrForbidden) {
			return campaigns.ErrForbidden
		}
		return err
	}
	return nil
}

func (s *Service) BroadcastWallet(campaignID int64) {
	s.broadcastCampaignSheets(campaignID)
}

func (s *Service) broadcastCampaignSheets(campaignID int64) {
	pcs, err := s.ListLiveByCampaign(campaignID)
	if err != nil {
		return
	}
	for _, row := range pcs {
		s.broadcastVitals(row.ID)
	}
}

func mapWalletErr(err error) error {
	switch {
	case errors.Is(err, campaigns.ErrForbidden):
		return ErrForbidden
	case errors.Is(err, campaigns.ErrSoulsCap):
		return ErrSoulsCap
	case errors.Is(err, campaigns.ErrAmount):
		return rules.ErrAmount
	default:
		return err
	}
}
