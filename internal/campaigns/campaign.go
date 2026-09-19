// Package campaigns owns tables, invite codes, and membership (Dungeon Master vs player).
package campaigns

import "time"

const (
	MemberPlayer = "Player"
	MemberDM     = "DungeonMaster"

	DefaultSoulsCap = 5000
)

// Campaign is a table. InviteCode is the join token shown to players.
type Campaign struct {
	ID         int64
	Name       string
	InviteCode string
	CreatedBy  int64
	CreatedAt  time.Time
	Souls      int
	SoulsCap   int
}

// Membership is a user's role in a campaign.
type Membership struct {
	CampaignID int64
	UserID     int64
	Role       string
	Username   string
}

// CampaignListItem is a campaign plus the current user's membership role.
type CampaignListItem struct {
	Campaign
	MemberRole string
}
