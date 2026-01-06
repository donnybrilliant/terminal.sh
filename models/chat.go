package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatRoom represents a chat room or channel for player communication.
type ChatRoom struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"` // e.g., "#public", "mygroup"
	Type      string    `gorm:"not null" json:"type"`             // "public", "private", "password"
	Password  string    `gorm:"" json:"-"`                         // hashed, only for password-protected rooms
	CreatedBy uuid.UUID `gorm:"type:text;not null;index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the chat room if one doesn't exist.
func (r *ChatRoom) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ChatMessage represents a message sent in a chat room.
type ChatMessage struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	RoomID    uuid.UUID `gorm:"type:text;not null;index" json:"room_id"`
	UserID    uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	Username  string    `gorm:"not null" json:"username"` // denormalized for quick display
	Content   string    `gorm:"not null" json:"content"`
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the chat message if one doesn't exist.
func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ChatRoomMember represents a user's membership in a chat room.
type ChatRoomMember struct {
	RoomID   uuid.UUID `gorm:"type:text;primary_key" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:text;primary_key" json:"user_id"`
	JoinedAt time.Time `gorm:"not null" json:"joined_at"`
}

