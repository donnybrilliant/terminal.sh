package services

import (
	"fmt"
	"terminal-sh/database"
	"terminal-sh/models"
	"terminal-sh/patch"

	"github.com/google/uuid"
)

// UpgradeService handles progressive tool upgrade operations.
type UpgradeService struct {
	db          *database.Database
	stateStore  ToolStateStore
	userService *UserService
	calculator  *patch.Calculator
}

// NewUpgradeService creates a new UpgradeService with the provided dependencies.
func NewUpgradeService(db *database.Database, stateStore ToolStateStore, userService *UserService) *UpgradeService {
	return &UpgradeService{
		db:          db,
		stateStore:  stateStore,
		userService: userService,
		calculator:  patch.NewCalculator(),
	}
}

// GetUpgradeCost returns the cost of an upgrade based on how many have been applied.
func (s *UpgradeService) GetUpgradeCost(upgradeType patch.UpgradeType, currentCount int) float64 {
	return patch.CalculateUpgradeCost(upgradeType, currentCount)
}

// GetCurrentUpgradeCount returns the current count for a specific upgrade type on a tool.
func (s *UpgradeService) GetCurrentUpgradeCount(toolState *models.UserToolState, upgradeType patch.UpgradeType) int {
	switch upgradeType {
	case patch.UpgradeExploit:
		return toolState.ExploitUpgrades
	case patch.UpgradeCPU:
		return toolState.CPUUpgrades
	case patch.UpgradeRAM:
		return toolState.RAMUpgrades
	case patch.UpgradeBandwidth:
		return toolState.BandwidthUpgrades
	case patch.UpgradeFullTune:
		// Full tune-up cost scales based on total upgrades applied
		return toolState.CPUUpgrades + toolState.RAMUpgrades + toolState.BandwidthUpgrades
	default:
		return 0
	}
}

// CanUpgrade checks if an upgrade can be applied and returns the reason if not.
func (s *UpgradeService) CanUpgrade(userID uuid.UUID, toolName string, upgradeType patch.UpgradeType) (bool, float64, string) {
	// Get tool state
	toolState, err := s.stateStore.GetUserToolState(userID, toolName)
	if err != nil {
		return false, 0, fmt.Sprintf("tool %s not owned", toolName)
	}

	// Get base tool for max level check
	baseTool, err := s.stateStore.GetBaseTool(toolState.ToolID)
	if err != nil {
		return false, 0, "failed to get base tool"
	}

	// Check if at max level for exploit upgrades
	if upgradeType == patch.UpgradeExploit {
		baseStats := patch.ToolStatsFromTool(baseTool)
		effectiveStats := s.calculator.CalculateWithUpgrades(
			baseStats,
			toolState.ExploitUpgrades,
			toolState.CPUUpgrades,
			toolState.RAMUpgrades,
			toolState.BandwidthUpgrades,
		)
		if !patch.CanApplyExploitUpgrade(effectiveStats) {
			return false, 0, fmt.Sprintf("all exploits are at max level (%d)", patch.MaxExploitLevel)
		}
	}

	// Calculate cost
	currentCount := s.GetCurrentUpgradeCount(toolState, upgradeType)
	cost := s.GetUpgradeCost(upgradeType, currentCount)

	// Check if user has enough crypto
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return false, cost, "failed to get user"
	}

	if user.Wallet.Crypto < cost {
		return false, cost, fmt.Sprintf("not enough crypto (need %.0f, have %.0f)", cost, user.Wallet.Crypto)
	}

	return true, cost, ""
}

// ApplyUpgrade applies an upgrade to a user's tool, deducting the cost.
func (s *UpgradeService) ApplyUpgrade(userID uuid.UUID, toolName string, upgradeType patch.UpgradeType) error {
	// Check if upgrade is possible
	canUpgrade, cost, reason := s.CanUpgrade(userID, toolName, upgradeType)
	if !canUpgrade {
		return fmt.Errorf("%s", reason)
	}

	// Get tool state
	toolState, err := s.stateStore.GetUserToolState(userID, toolName)
	if err != nil {
		return fmt.Errorf("tool not owned: %w", err)
	}

	// Deduct cost from user wallet
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	user.Wallet.Crypto -= cost
	if err := s.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to deduct cost: %w", err)
	}

	// Apply the upgrade
	s.applyUpgradeToState(toolState, upgradeType)

	// Recalculate effective stats
	baseTool, err := s.stateStore.GetBaseTool(toolState.ToolID)
	if err != nil {
		return fmt.Errorf("failed to get base tool: %w", err)
	}

	baseStats := patch.ToolStatsFromTool(baseTool)
	effectiveStats := s.calculator.CalculateWithUpgrades(
		baseStats,
		toolState.ExploitUpgrades,
		toolState.CPUUpgrades,
		toolState.RAMUpgrades,
		toolState.BandwidthUpgrades,
	)

	toolState.EffectiveExploits = effectiveStats.Exploits
	toolState.EffectiveResources = effectiveStats.Resources
	toolState.Version++

	// Save updated tool state
	if err := s.stateStore.SaveToolState(toolState); err != nil {
		return fmt.Errorf("failed to save tool state: %w", err)
	}

	return nil
}

// ApplyFreeUpgrade applies an upgrade without cost (for mission rewards).
func (s *UpgradeService) ApplyFreeUpgrade(userID uuid.UUID, toolName string, upgradeType patch.UpgradeType) error {
	// Get tool state
	toolState, err := s.stateStore.GetUserToolState(userID, toolName)
	if err != nil {
		return fmt.Errorf("tool not owned: %w", err)
	}

	// Check max level for exploit upgrades
	if upgradeType == patch.UpgradeExploit {
		baseTool, err := s.stateStore.GetBaseTool(toolState.ToolID)
		if err != nil {
			return fmt.Errorf("failed to get base tool: %w", err)
		}
		baseStats := patch.ToolStatsFromTool(baseTool)
		effectiveStats := s.calculator.CalculateWithUpgrades(
			baseStats,
			toolState.ExploitUpgrades,
			toolState.CPUUpgrades,
			toolState.RAMUpgrades,
			toolState.BandwidthUpgrades,
		)
		if !patch.CanApplyExploitUpgrade(effectiveStats) {
			return nil // Silently skip if at max level
		}
	}

	// Apply the upgrade
	s.applyUpgradeToState(toolState, upgradeType)

	// Recalculate effective stats
	baseTool, err := s.stateStore.GetBaseTool(toolState.ToolID)
	if err != nil {
		return fmt.Errorf("failed to get base tool: %w", err)
	}

	baseStats := patch.ToolStatsFromTool(baseTool)
	effectiveStats := s.calculator.CalculateWithUpgrades(
		baseStats,
		toolState.ExploitUpgrades,
		toolState.CPUUpgrades,
		toolState.RAMUpgrades,
		toolState.BandwidthUpgrades,
	)

	toolState.EffectiveExploits = effectiveStats.Exploits
	toolState.EffectiveResources = effectiveStats.Resources
	toolState.Version++

	// Save updated tool state
	if err := s.stateStore.SaveToolState(toolState); err != nil {
		return fmt.Errorf("failed to save tool state: %w", err)
	}

	return nil
}

// applyUpgradeToState increments the appropriate upgrade counter.
func (s *UpgradeService) applyUpgradeToState(toolState *models.UserToolState, upgradeType patch.UpgradeType) {
	switch upgradeType {
	case patch.UpgradeExploit:
		toolState.ExploitUpgrades++
	case patch.UpgradeCPU:
		toolState.CPUUpgrades++
	case patch.UpgradeRAM:
		toolState.RAMUpgrades++
	case patch.UpgradeBandwidth:
		toolState.BandwidthUpgrades++
	case patch.UpgradeFullTune:
		// Full tune-up increments CPU, RAM, and bandwidth (at reduced rates)
		toolState.CPUUpgrades++
		toolState.RAMUpgrades++
		toolState.BandwidthUpgrades++
	}
}

// GetToolUpgradeInfo returns information about a tool's current upgrades and available upgrades.
type ToolUpgradeInfo struct {
	ToolName          string
	ExploitUpgrades   int
	CPUUpgrades       int
	RAMUpgrades       int
	BandwidthUpgrades int
	TotalUpgrades     int
	EffectiveExploits []models.Exploit
	EffectiveResources models.ToolResources
	MaxExploitLevel   int
	AvailableUpgrades []AvailableUpgrade
}

// AvailableUpgrade represents an upgrade that can be applied to a tool.
type AvailableUpgrade struct {
	Type        patch.UpgradeType
	Name        string
	Description string
	Cost        float64
	CanApply    bool
	Reason      string
}

// GetToolUpgradeInfo returns upgrade information for a specific tool.
func (s *UpgradeService) GetToolUpgradeInfo(userID uuid.UUID, toolName string) (*ToolUpgradeInfo, error) {
	toolState, err := s.stateStore.GetUserToolState(userID, toolName)
	if err != nil {
		return nil, fmt.Errorf("tool not owned: %w", err)
	}

	baseTool, err := s.stateStore.GetBaseTool(toolState.ToolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base tool: %w", err)
	}

	baseStats := patch.ToolStatsFromTool(baseTool)
	effectiveStats := s.calculator.CalculateWithUpgrades(
		baseStats,
		toolState.ExploitUpgrades,
		toolState.CPUUpgrades,
		toolState.RAMUpgrades,
		toolState.BandwidthUpgrades,
	)

	info := &ToolUpgradeInfo{
		ToolName:          toolName,
		ExploitUpgrades:   toolState.ExploitUpgrades,
		CPUUpgrades:       toolState.CPUUpgrades,
		RAMUpgrades:       toolState.RAMUpgrades,
		BandwidthUpgrades: toolState.BandwidthUpgrades,
		TotalUpgrades:     toolState.ExploitUpgrades + toolState.CPUUpgrades + toolState.RAMUpgrades + toolState.BandwidthUpgrades,
		EffectiveExploits: effectiveStats.Exploits,
		EffectiveResources: effectiveStats.Resources,
		MaxExploitLevel:   patch.GetMaxExploitLevel(effectiveStats),
	}

	// Get available upgrades
	for _, def := range patch.AllUpgrades() {
		canApply, cost, reason := s.CanUpgrade(userID, toolName, def.Type)
		info.AvailableUpgrades = append(info.AvailableUpgrades, AvailableUpgrade{
			Type:        def.Type,
			Name:        def.Name,
			Description: def.Description,
			Cost:        cost,
			CanApply:    canApply,
			Reason:      reason,
		})
	}

	return info, nil
}
