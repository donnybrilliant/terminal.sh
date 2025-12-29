// Package models provides data models for the terminal.sh game database.
// All models use GORM for ORM functionality and include UUID primary keys.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Resources represents user computational resources.
type Resources struct {
	CPU      int     `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// Wallet represents a user's currency balances.
type Wallet struct {
	Crypto float64 `json:"crypto"`
	Data   float64 `json:"data"`
}

// User represents a game user account with authentication, resources, and progress tracking.
type User struct {
	ID              uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	Username        string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash    string    `gorm:"not null" json:"-"`
	IP              string    `gorm:"uniqueIndex;not null" json:"ip"`
	LocalIP         string    `gorm:"not null" json:"local_ip"`
	MAC             string    `gorm:"uniqueIndex;not null" json:"mac"`
	Level           int       `gorm:"default:0" json:"level"`
	Experience      int       `gorm:"default:0" json:"experience"`
	Resources       Resources  `gorm:"type:text;serializer:json" json:"resources"`
	Wallet          Wallet     `gorm:"type:text;serializer:json" json:"wallet"`
	FileSystem      map[string]interface{} `gorm:"type:text;serializer:json" json:"file_system"` // User's filesystem changes
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	
	// Relationships
	Tools           []Tool           `gorm:"many2many:user_tools;" json:"tools,omitempty"`
	ToolStates      []UserToolState  `gorm:"foreignKey:UserID" json:"tool_states,omitempty"`
	Purchases       []UserPurchase   `gorm:"foreignKey:UserID" json:"purchases,omitempty"`
	OwnedPatches    []UserPatch      `gorm:"foreignKey:UserID" json:"owned_patches,omitempty"`
	Achievements    []UserAchievement `gorm:"foreignKey:UserID" json:"achievements,omitempty"`
	ExploitedServers []ExploitedServer `gorm:"foreignKey:UserID" json:"exploited_servers,omitempty"`
	ActiveMiners    []ActiveMiner     `gorm:"foreignKey:UserID" json:"active_miners,omitempty"`
	Sessions        []Session         `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
}

// BeforeCreate is a GORM hook that generates a UUID for the user if one doesn't exist.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

