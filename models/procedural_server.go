package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProceduralServer tracks procedurally generated servers
type ProceduralServer struct {
	ID           uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	ServerID     uuid.UUID `gorm:"type:text;not null;index" json:"server_id"` // Reference to Server.ID
	GeneratedFor uuid.UUID `gorm:"type:text;index" json:"generated_for"`      // User ID or mission ID
	GeneratedAt  time.Time `gorm:"not null" json:"generated_at"`
	Reason       string    `gorm:"type:text;not null" json:"reason"` // "exhaustion", "mission", "proactive"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the procedural server if one doesn't exist.
func (p *ProceduralServer) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
