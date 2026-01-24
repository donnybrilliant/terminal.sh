package services

import (
	"fmt"
	"terminal-sh/database"
	"terminal-sh/models"
	"terminal-sh/patch"

	"github.com/google/uuid"
)

// RewardService handles granting rewards for mission completion
type RewardService struct {
	db                 *database.Database
	userService        *UserService
	toolService        *ToolService
	upgradeService     *UpgradeService
	achievementService *AchievementService
}

// NewRewardService creates a new RewardService
func NewRewardService(db *database.Database, userService *UserService, toolService *ToolService, upgradeService *UpgradeService, achievementService *AchievementService) *RewardService {
	return &RewardService{
		db:                 db,
		userService:        userService,
		toolService:        toolService,
		upgradeService:     upgradeService,
		achievementService: achievementService,
	}
}

// GrantRewards grants all rewards from a mission to a user
func (s *RewardService) GrantRewards(userID uuid.UUID, rewards models.MissionRewards) error {
	// Grant experience
	if rewards.Experience > 0 {
		if err := s.userService.AddExperience(userID, rewards.Experience); err != nil {
			return fmt.Errorf("failed to grant experience: %w", err)
		}
	}

	// Grant cryptocurrency
	if rewards.Crypto > 0 {
		user, err := s.userService.GetUserByID(userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		user.Wallet.Crypto += rewards.Crypto
		if err := s.db.Save(user).Error; err != nil {
			return fmt.Errorf("failed to grant crypto: %w", err)
		}
	}

	// Grant tools
	for _, toolName := range rewards.Tools {
		tool, err := s.toolService.GetToolByName(toolName)
		if err != nil {
			// Tool might not exist yet, skip
			continue
		}
		// Check if already owned
		if s.toolService.UserHasTool(userID, toolName) {
			continue
		}
		if err := s.toolService.GrantToolToUser(userID, tool.ID); err != nil {
			// Tool might already be owned, continue
			continue
		}
	}

	// Grant tool upgrades (free upgrades as mission rewards)
	for _, upgradeReward := range rewards.ToolUpgrades {
		// Check if user owns the tool
		if !s.toolService.UserHasTool(userID, upgradeReward.ToolName) {
			continue
		}

		// Parse upgrade type
		upgradeType, valid := patch.ParseUpgradeType(upgradeReward.UpgradeType)
		if !valid {
			continue
		}

		// Apply the upgrade(s) for free
		for i := 0; i < upgradeReward.Count; i++ {
			if err := s.upgradeService.ApplyFreeUpgrade(userID, upgradeReward.ToolName, upgradeType); err != nil {
				// Upgrade might fail (e.g., at max level), continue
				break
			}
		}
	}

	// Grant achievements
	for _, achievementName := range rewards.Achievements {
		if err := s.achievementService.UnlockAchievement(userID, achievementName); err != nil {
			// Achievement might already be unlocked, continue
			continue
		}
	}

	return nil
}
