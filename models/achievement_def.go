package models

// AchievementDefinition represents an achievement definition
type AchievementDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"` // Emoji or icon identifier
}

// AchievementData represents the complete achievement data structure
type AchievementData struct {
	Achievements []AchievementDefinition `json:"achievements"`
}
