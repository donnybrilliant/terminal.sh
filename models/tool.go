package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Exploit represents a vulnerability exploit
type Exploit struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// ToolResources represents tool resource requirements
type ToolResources struct {
	CPU      float64 `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// Tool represents a game tool
type Tool struct {
	ID        uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Function  string         `gorm:"not null" json:"function"`
	Resources ToolResources  `gorm:"type:text;serializer:json" json:"resources"`
	Exploits  []Exploit      `gorm:"type:text;serializer:json" json:"exploits,omitempty"`
	Services  string         `gorm:"" json:"services,omitempty"`
	Special   string         `gorm:"" json:"special,omitempty"`
	IsPatch   bool           `gorm:"default:false" json:"is_patch"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	
	// Relationships
	Users []User `gorm:"many2many:user_tools;" json:"users,omitempty"`
}

// BeforeCreate hook to generate UUID
func (t *Tool) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// UserTool represents the many-to-many relationship between users and tools
type UserTool struct {
	UserID uuid.UUID `gorm:"type:text;primary_key" json:"user_id"`
	ToolID uuid.UUID `gorm:"type:text;primary_key" json:"tool_id"`
}

