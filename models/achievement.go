package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserAchievement represents a user's achievement
type UserAchievement struct {
	ID           uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	AchievementName string `gorm:"not null" json:"achievement_name"`
	UnlockedAt   time.Time `gorm:"not null" json:"unlocked_at"`
}

// BeforeCreate hook to generate UUID
func (u *UserAchievement) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

