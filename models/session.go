package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session represents an SSH or terminal session for a user.
type Session struct {
	ID              uuid.UUID  `gorm:"type:text;primary_key" json:"id"`
	UserID          uuid.UUID  `gorm:"type:text;not null;index" json:"user_id"`
	SSHConnID       string     `gorm:"not null;index" json:"ssh_conn_id"` // Unique identifier for SSH connection
	CurrentServerPath string   `gorm:"" json:"current_server_path"` // Current server path, empty if on user's local system
	ParentSessionID *uuid.UUID `gorm:"type:text;index" json:"parent_session_id,omitempty"` // For nested SSH sessions
	CreatedAt       time.Time  `gorm:"not null" json:"created_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the session if one doesn't exist.
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

