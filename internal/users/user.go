// Package users handles registration, login, and account language.
// The first registered account is RoleAdmin; later accounts are RolePlayer.
package users

import "time"

const (
	RolePlayer        = "Player"
	RoleDungeonMaster = "DungeonMaster"
	RoleAdmin         = "Admin"
)

// User is an account row. PasswordHash is bcrypt; do not send it to templates.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	Language     string
	CreatedAt    time.Time
}
