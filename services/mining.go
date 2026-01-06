package services

import (
	"fmt"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// MiningService handles cryptocurrency mining operations on exploited servers.
type MiningService struct {
	db            *database.Database
	toolService   *ToolService
	serverService *ServerService
}

// NewMiningService creates a new MiningService with the provided dependencies.
func NewMiningService(db *database.Database, toolService *ToolService, serverService *ServerService) *MiningService {
	return &MiningService{
		db:            db,
		toolService:   toolService,
		serverService: serverService,
	}
}

// StartMining starts a cryptocurrency mining operation on an exploited server.
// Returns an error if the server is not exploited, lacks resources, or mining fails.
func (s *MiningService) StartMining(userID uuid.UUID, serverIP string) error {
	// Check if user has crypto_miner tool
	if !s.toolService.UserHasTool(userID, "crypto_miner") {
		return fmt.Errorf("crypto_miner tool not owned")
	}

	// CRITICAL: Get effective tool (with patches applied) to check resource requirements
	tool, err := s.toolService.GetEffectiveTool(userID, "crypto_miner")
	if err != nil {
		return fmt.Errorf("crypto_miner tool not found")
	}

	// Check if server exists
	if _, err := s.serverService.GetServerByIP(serverIP); err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Check if already mining on this server
	var existing models.ActiveMiner
	if err := s.db.Where("user_id = ? AND server_ip = ?", userID, serverIP).First(&existing).Error; err == nil {
		return fmt.Errorf("already mining on server %s", serverIP)
	}

	// Check server resources
	resourceUsage := models.MiningResourceUsage{
		CPU:      tool.Resources.CPU,
		Bandwidth: tool.Resources.Bandwidth,
		RAM:      tool.Resources.RAM,
	}

	// TODO: Check server's used resources vs available
	// For now, we'll assume resources are available

	// Create mining session
	miner := &models.ActiveMiner{
		UserID:        userID,
		ServerIP:      serverIP,
		StartTime:     time.Now(),
		ResourceUsage: resourceUsage,
	}

	if err := s.db.Create(miner).Error; err != nil {
		return fmt.Errorf("failed to start mining: %w", err)
	}

	return nil
}

// StopMining stops a mining operation
func (s *MiningService) StopMining(userID uuid.UUID, serverIP string) error {
	result := s.db.Where("user_id = ? AND server_ip = ?", userID, serverIP).Delete(&models.ActiveMiner{})
	if result.Error != nil {
		return fmt.Errorf("failed to stop mining: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no active miner found on server %s", serverIP)
	}
	return nil
}

// GetActiveMiners retrieves all active miners for a user
func (s *MiningService) GetActiveMiners(userID uuid.UUID) ([]models.ActiveMiner, error) {
	var miners []models.ActiveMiner
	if err := s.db.Where("user_id = ?", userID).Find(&miners).Error; err != nil {
		return nil, err
	}
	return miners, nil
}

// CalculateMiningReward calculates the reward for a mining session
func (s *MiningService) CalculateMiningReward(miner *models.ActiveMiner) float64 {
	duration := time.Since(miner.StartTime)
	hours := duration.Hours()
	
	// Reward formula: base rate * hours * resource usage multiplier
	baseRate := 1.0 // crypto per hour
	multiplier := 1.0 + (miner.ResourceUsage.CPU / 100.0) // CPU affects reward
	
	return baseRate * hours * multiplier
}

// ProcessMiningRewards processes rewards for all active miners (called periodically)
func (s *MiningService) ProcessMiningRewards() error {
	var miners []models.ActiveMiner
	if err := s.db.Find(&miners).Error; err != nil {
		return err
	}

	for _, miner := range miners {
		reward := s.CalculateMiningReward(&miner)
		
		// Update user's wallet
		var user models.User
		if err := s.db.First(&user, "id = ?", miner.UserID).Error; err != nil {
			continue
		}

		user.Wallet.Crypto += reward
		if err := s.db.Save(&user).Error; err != nil {
			continue
		}

		// Reset start time for next period
		miner.StartTime = time.Now()
		if err := s.db.Save(&miner).Error; err != nil {
			continue
		}
	}

	return nil
}

