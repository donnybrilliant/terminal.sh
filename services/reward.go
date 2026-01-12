package services

import (
	"fmt"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// RewardService handles granting rewards for mission completion
type RewardService struct {
	db                *database.Database
	userService       *UserService
	toolService       *ToolService
	patchService      *PatchService
	achievementService *AchievementService
}

// NewRewardService creates a new RewardService
func NewRewardService(db *database.Database, userService *UserService, toolService *ToolService, patchService *PatchService, achievementService *AchievementService) *RewardService {
	return &RewardService{
		db:                db,
		userService:       userService,
		toolService:       toolService,
		patchService:      patchService,
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
	
	// Grant patches
	for _, patchName := range rewards.Patches {
		patch, err := s.patchService.GetPatchByName(patchName)
		if err != nil {
			// Patch might not exist yet, skip
			continue
		}
		if err := s.patchService.GrantPatchToUser(userID, patch.ID); err != nil {
			// Patch might already be owned, continue
			continue
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
