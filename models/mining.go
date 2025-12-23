package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MiningResourceUsage represents resource usage for mining
type MiningResourceUsage struct {
	CPU      float64 `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// ActiveMiner represents an active mining session
type ActiveMiner struct {
	ID            uuid.UUID          `gorm:"type:text;primary_key" json:"id"`
	UserID        uuid.UUID          `gorm:"type:text;not null;index" json:"user_id"`
	ServerIP      string             `gorm:"not null;index" json:"server_ip"`
	StartTime     time.Time          `gorm:"not null" json:"start_time"`
	ResourceUsage MiningResourceUsage `gorm:"type:text;serializer:json" json:"resource_usage"`
}

// BeforeCreate hook to generate UUID
func (a *ActiveMiner) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

