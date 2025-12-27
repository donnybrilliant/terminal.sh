package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Resources represents user computational resources
type Resources struct {
	CPU      int     `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// Wallet represents user's currency
type Wallet struct {
	Crypto float64 `json:"crypto"`
	Data   float64 `json:"data"`
}

// User represents a game user
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

// BeforeCreate hook to generate UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

