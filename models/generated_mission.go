package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GeneratedMission tracks procedurally generated missions per user
type GeneratedMission struct {
	ID          uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID      uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	MissionID   string    `gorm:"not null;index" json:"mission_id"` // Generated mission ID
	GeneratedAt time.Time `gorm:"not null" json:"generated_at"`
	Difficulty  int       `gorm:"default:0" json:"difficulty"` // Calculated difficulty
	ServerIP    string    `gorm:"type:text" json:"server_ip,omitempty"` // Target server (if mission-specific)
	MissionData string    `gorm:"type:text" json:"mission_data"` // JSON of mission
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the generated mission if one doesn't exist.
func (g *GeneratedMission) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
