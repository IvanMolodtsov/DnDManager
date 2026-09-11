package campaigns

import "time"

const (
	MemberPlayer = "Player"
	MemberDM     = "DungeonMaster"
)

type Campaign struct {
	ID         int64
	Name       string
	InviteCode string
	CreatedBy  int64
	CreatedAt  time.Time
}

type Membership struct {
	CampaignID int64
	UserID     int64
	Role       string
	Username   string
}

type CampaignListItem struct {
	Campaign
	MemberRole string
}
