package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserToolState represents a user's specific state for a tool (CRITICAL: user-specific tool states)
type UserToolState struct {
	ID                uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	UserID            uuid.UUID      `gorm:"not null;index" json:"user_id"`
	ToolID            uuid.UUID      `gorm:"not null;index" json:"tool_id"`
	AppliedPatches    []string       `gorm:"type:text;serializer:json" json:"applied_patches"` // List of patch names applied
	EffectiveExploits []Exploit      `gorm:"type:text;serializer:json" json:"effective_exploits"` // Calculated exploits after patches
	EffectiveResources ToolResources `gorm:"type:text;serializer:json" json:"effective_resources"` // Calculated resources after patches
	Version           int            `gorm:"default:1" json:"version"` // Tool version (increments with each patch)
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Tool Tool `gorm:"foreignKey:ToolID" json:"tool,omitempty"`
}

// BeforeCreate hook to generate UUID
func (uts *UserToolState) BeforeCreate(tx *gorm.DB) error {
	if uts.ID == uuid.Nil {
		uts.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name for UserToolState
func (UserToolState) TableName() string {
	return "user_tool_states"
}

