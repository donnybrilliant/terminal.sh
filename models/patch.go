package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PatchUpgrades represents the upgrades a patch provides
type PatchUpgrades struct {
	Exploits  []Exploit      `json:"exploits,omitempty"`  // New or upgraded exploits
	Resources ToolResources  `json:"resources,omitempty"` // Resource modifications (can be negative)
}

// Patch represents a tool patch/upgrade
type Patch struct {
	ID          uuid.UUID     `gorm:"type:text;primary_key" json:"id"`
	Name        string        `gorm:"uniqueIndex;not null" json:"name"`
	TargetTool  string        `gorm:"not null" json:"target_tool"` // Tool this patch upgrades
	Description string        `json:"description"`
	Upgrades    PatchUpgrades `gorm:"type:text;serializer:json" json:"upgrades"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (p *Patch) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// UserPatch represents a patch owned by a user (for shop purchases)
type UserPatch struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID    uuid.UUID `gorm:"not null;index" json:"user_id"`
	PatchID   uuid.UUID `gorm:"not null" json:"patch_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Patch Patch `gorm:"foreignKey:PatchID" json:"patch,omitempty"`
}

// BeforeCreate hook to generate UUID
func (up *UserPatch) BeforeCreate(tx *gorm.DB) error {
	if up.ID == uuid.Nil {
		up.ID = uuid.New()
	}
	return nil
}

