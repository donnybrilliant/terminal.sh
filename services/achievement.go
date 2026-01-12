package services

import (
	"encoding/json"
	"fmt"
	"os"
	"terminal-sh/database"
	"terminal-sh/models"
	"time"

	"github.com/google/uuid"
)

// AchievementService handles achievement-related operations
type AchievementService struct {
	db          *database.Database
	achievements []models.AchievementDefinition
	dataPath    string
}

// NewAchievementService creates a new AchievementService and loads achievements from JSON
func NewAchievementService(db *database.Database, dataPath string) (*AchievementService, error) {
	service := &AchievementService{
		db:       db,
		dataPath: dataPath,
	}
	
	if err := service.LoadAchievements(); err != nil {
		return nil, fmt.Errorf("failed to load achievements: %w", err)
	}
	
	return service, nil
}

// LoadAchievements loads achievements from JSON file
func (s *AchievementService) LoadAchievements() error {
	path := s.dataPath
	if path == "" {
		path = "data/seed/achievements.json"
	}
	
	// Check if file exists, if not create default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := s.createDefaultAchievements(path); err != nil {
			return fmt.Errorf("failed to create default achievements: %w", err)
		}
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read achievement file: %w", err)
	}
	
	var achievementData models.AchievementData
	if err := json.Unmarshal(data, &achievementData); err != nil {
		return fmt.Errorf("failed to parse achievement file: %w", err)
	}
	
	s.achievements = achievementData.Achievements
	return nil
}

// UnlockAchievement unlocks an achievement for a user
func (s *AchievementService) UnlockAchievement(userID uuid.UUID, achievementName string) error {
	// Check if achievement already unlocked
	var existing models.UserAchievement
	err := s.db.Where("user_id = ? AND achievement_name = ?", userID, achievementName).First(&existing).Error
	if err == nil {
		// Already unlocked
		return nil
	}
	
	// Create new achievement
	achievement := &models.UserAchievement{
		UserID:         userID,
		AchievementName: achievementName,
		UnlockedAt:     time.Now(),
	}
	
	if err := s.db.Create(achievement).Error; err != nil {
		return fmt.Errorf("failed to unlock achievement: %w", err)
	}
	
	return nil
}

// GetUserAchievements retrieves all achievements for a user
func (s *AchievementService) GetUserAchievements(userID uuid.UUID) ([]models.UserAchievement, error) {
	var achievements []models.UserAchievement
	if err := s.db.Where("user_id = ?", userID).Find(&achievements).Error; err != nil {
		return nil, err
	}
	return achievements, nil
}

// GetAllAchievements returns all achievement definitions
func (s *AchievementService) GetAllAchievements() []models.AchievementDefinition {
	return s.achievements
}

// GetAchievementByName returns an achievement definition by name
func (s *AchievementService) GetAchievementByName(name string) *models.AchievementDefinition {
	for _, ach := range s.achievements {
		if ach.Name == name {
			return &ach
		}
	}
	return nil
}

// createDefaultAchievements creates a default achievements file
func (s *AchievementService) createDefaultAchievements(path string) error {
	defaultAchievements := models.AchievementData{
		Achievements: []models.AchievementDefinition{
			{
				ID:          "wifi_warrior",
				Name:        "WiFi Warrior",
				Description: "Successfully hacked public WiFi",
				Icon:        "📶",
			},
			{
				ID:          "social_engineer",
				Name:        "Social Engineer",
				Description: "Successfully phished 10 targets",
				Icon:        "🎣",
			},
			{
				ID:          "data_thief",
				Name:        "Data Thief",
				Description: "Extracted 1GB of data",
				Icon:        "💾",
			},
			{
				ID:          "ghost_in_the_machine",
				Name:        "Ghost in the Machine",
				Description: "Completed mission without detection",
				Icon:        "👻",
			},
			{
				ID:          "academic_misconduct",
				Name:        "Academic Misconduct",
				Description: "Modified academic records",
				Icon:        "🎓",
			},
		},
	}
	
	data, err := json.MarshalIndent(defaultAchievements, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	if err := os.MkdirAll("data/seed", 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}
