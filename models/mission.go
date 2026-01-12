package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MissionObjective represents a single objective in a mission
type MissionObjective struct {
	ID          int    `json:"id"`
	Type        string `json:"type"` // "exploit_server", "use_tool", "collect_data", etc.
	Description string `json:"description"`
	Tool        string `json:"tool,omitempty"`           // Required tool for "use_tool" type
	TargetType  string `json:"target_server_type,omitempty"` // Server type to target
	Hint        string `json:"hint,omitempty"`          // Tutorial-like hint explaining how to complete this objective
}

// MissionRewards represents rewards for completing a mission
type MissionRewards struct {
	Experience   int      `json:"experience"`
	Crypto       float64  `json:"crypto"`
	Tools        []string `json:"tools,omitempty"`        // Tools to unlock
	Patches      []string `json:"patches,omitempty"`     // Patches to unlock
	Achievements []string `json:"achievements,omitempty"` // Achievements to unlock
}

// Mission represents a story mission definition
type Mission struct {
	ID            string            `json:"id" gorm:"primaryKey"`
	ArcID         string            `json:"arc_id" gorm:"index"`
	ArcName       string            `json:"arc_name"`
	MissionNumber int              `json:"mission_number"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Prerequisites []string          `json:"prerequisites" gorm:"type:text;serializer:json"` // Previous mission IDs
	RequiredTools []string          `json:"required_tools" gorm:"type:text;serializer:json"`
	RequiredLevel int              `json:"required_level"`
	Objectives    []MissionObjective `json:"objectives" gorm:"type:text;serializer:json"`
	Rewards        MissionRewards    `json:"rewards" gorm:"type:text;serializer:json"`
	Unlocks        []string         `json:"unlocks" gorm:"type:text;serializer:json"` // Next mission/arc IDs
}

// UserMission represents a user's progress on a mission
type UserMission struct {
	ID            uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID        uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	MissionID     string    `gorm:"not null;index" json:"mission_id"`
	Status        string    `gorm:"not null;default:pending" json:"status"` // "pending", "in_progress", "completed"
	Progress      int       `gorm:"default:0" json:"progress"` // 0-100 percentage
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the user mission if one doesn't exist.
func (u *UserMission) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// MissionData represents the complete mission data structure containing all missions.
type MissionData struct {
	Missions []Mission `json:"missions"`
}
