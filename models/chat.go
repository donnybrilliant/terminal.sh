package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatRoom represents a chat room/channel
type ChatRoom struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"` // e.g., "#public", "mygroup"
	Type      string    `gorm:"not null" json:"type"`             // "public", "private", "password"
	Password  string    `gorm:"" json:"-"`                         // hashed, only for password-protected rooms
	CreatedBy uuid.UUID `gorm:"type:text;not null;index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID
func (r *ChatRoom) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ChatMessage represents a message in a chat room
type ChatMessage struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	RoomID    uuid.UUID `gorm:"type:text;not null;index" json:"room_id"`
	UserID    uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	Username  string    `gorm:"not null" json:"username"` // denormalized for quick display
	Content   string    `gorm:"not null" json:"content"`
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}

// BeforeCreate hook to generate UUID
func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ChatRoomMember represents membership in a chat room
type ChatRoomMember struct {
	RoomID   uuid.UUID `gorm:"type:text;primary_key" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:text;primary_key" json:"user_id"`
	JoinedAt time.Time `gorm:"not null" json:"joined_at"`
}

