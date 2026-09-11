package users

import "time"

const (
	RolePlayer        = "Player"
	RoleDungeonMaster = "DungeonMaster"
	RoleAdmin         = "Admin"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	Language     string
	CreatedAt    time.Time
}
