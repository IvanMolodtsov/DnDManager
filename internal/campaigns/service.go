package campaigns

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrNameRequired   = errors.New("campaign name required")
	ErrInviteRequired = errors.New("invite required")
	ErrInviteInvalid  = errors.New("invite invalid")
	ErrAlreadyMember  = errors.New("already member")
	ErrNotMember      = errors.New("not member")
	ErrNotFound       = errors.New("campaign not found")
)

const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Service struct {
	Repo *Repository
}

func (s *Service) Create(name string, createdBy int64) (*Campaign, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return nil, ErrNameRequired
	}
	code, err := s.uniqueInvite()
	if err != nil {
		return nil, err
	}
	id, err := s.Repo.Create(name, code, createdBy)
	if err != nil {
		return nil, err
	}
	return s.Repo.FindByID(id)
}

func (s *Service) Join(invite string, userID int64) (*Campaign, error) {
	invite = strings.ToUpper(strings.TrimSpace(invite))
	if invite == "" {
		return nil, ErrInviteRequired
	}
	c, err := s.Repo.FindByInvite(invite)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInviteInvalid
		}
		return nil, err
	}
	if _, err := s.Repo.Membership(c.ID, userID); err == nil {
		return nil, ErrAlreadyMember
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err := s.Repo.AddMember(c.ID, userID, MemberPlayer); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) Get(id int64) (*Campaign, error) {
	c, err := s.Repo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) ListForUser(userID int64) ([]CampaignListItem, error) {
	return s.Repo.ListForUser(userID)
}

func (s *Service) RequireMember(campaignID, userID int64) (*Membership, error) {
	m, err := s.Repo.Membership(campaignID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotMember
		}
		return nil, err
	}
	return m, nil
}

func (s *Service) IsDM(campaignID, userID int64) bool {
	m, err := s.Repo.Membership(campaignID, userID)
	return err == nil && m.Role == MemberDM
}

func (s *Service) Members(campaignID int64) ([]Membership, error) {
	return s.Repo.Members(campaignID)
}

func (s *Service) uniqueInvite() (string, error) {
	for i := 0; i < 8; i++ {
		code, err := randomInvite(8)
		if err != nil {
			return "", err
		}
		exists, err := s.Repo.InviteExists(code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("could not allocate invite code")
}

func randomInvite(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range b {
		out[i] = inviteAlphabet[int(b[i])%len(inviteAlphabet)]
	}
	return string(out), nil
}

func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrNameRequired):
		return "error.campaign.name"
	case errors.Is(err, ErrInviteRequired):
		return "error.invite.required"
	case errors.Is(err, ErrInviteInvalid):
		return "error.invite.invalid"
	case errors.Is(err, ErrAlreadyMember):
		return "error.already.member"
	default:
		return "error.generic"
	}
}
