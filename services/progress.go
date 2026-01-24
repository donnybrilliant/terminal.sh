package services

import (
	"math"
	"terminal-sh/models"
)

// OperationType represents the type of operation for progress calculation.
type OperationType string

const (
	OperationDownload OperationType = "download" // Tool download operation
	OperationExploit  OperationType = "exploit"  // Server exploitation operation
	OperationSSH      OperationType = "ssh"      // SSH connection operation
	OperationTransfer OperationType = "transfer" // Data transfer operation
)

// ProgressService handles progress calculation for game operations based on user resources.
type ProgressService struct{}

// NewProgressService creates a new ProgressService.
func NewProgressService() *ProgressService {
	return &ProgressService{}
}

// CalculateOperationTime calculates the time needed for an operation based on user resources.
// Returns the operation time in seconds.
// This is the basic version that only considers user resources (used for SSH, etc.)
func (s *ProgressService) CalculateOperationTime(operationType OperationType, userResources models.Resources) float64 {
	// Base times in seconds
	baseTimes := map[OperationType]float64{
		OperationDownload: 5.0,
		OperationExploit:  3.0,
		OperationSSH:      2.0,
		OperationTransfer: 4.0,
	}

	baseTime := baseTimes[operationType]
	if baseTime == 0 {
		baseTime = 3.0 // Default
	}

	multiplier := s.GetResourceMultiplier(userResources)
	return baseTime / multiplier
}

// CalculateToolOperationTime calculates time for tool-based operations considering both
// user resources and tool resources (base or upgraded).
// - userResources: The user's computational resources (affects overall speed)
// - toolResources: The tool's resource requirements/bonuses (upgraded tools are faster)
// - operationType: Type of operation being performed
// Returns the operation time in seconds.
func (s *ProgressService) CalculateToolOperationTime(operationType OperationType, userResources models.Resources, toolResources models.ToolResources) float64 {
	// Base times in seconds
	baseTimes := map[OperationType]float64{
		OperationDownload: 5.0,
		OperationExploit:  3.0,
		OperationSSH:      2.0,
		OperationTransfer: 4.0,
	}

	baseTime := baseTimes[operationType]
	if baseTime == 0 {
		baseTime = 3.0 // Default
	}

	// Get user resource multiplier (higher resources = faster)
	userMultiplier := s.GetResourceMultiplier(userResources)
	
	// Get tool efficiency multiplier (better tool stats = faster)
	toolMultiplier := s.GetToolResourceMultiplier(toolResources)
	
	// Combined multiplier: both user resources and tool quality affect speed
	// User resources have a stronger effect (they're the "hardware")
	// Tool resources provide additional optimization (software efficiency)
	combinedMultiplier := (userMultiplier * 0.6) + (toolMultiplier * 0.4)
	
	// Clamp final multiplier between 0.3 and 3.0
	combinedMultiplier = math.Max(0.3, math.Min(3.0, combinedMultiplier))
	
	return baseTime / combinedMultiplier
}

// GetResourceMultiplier calculates the speed multiplier based on user resources
func (s *ProgressService) GetResourceMultiplier(userResources models.Resources) float64 {
	// Normalize resources (assuming base values: CPU=200, Bandwidth=300, RAM=24)
	cpuFactor := float64(userResources.CPU) / 200.0
	bandwidthFactor := userResources.Bandwidth / 300.0
	ramFactor := float64(userResources.RAM) / 24.0

	// Calculate multiplier: average of factors, weighted, with minimum of 0.5
	avgFactor := (cpuFactor + bandwidthFactor + ramFactor) / 3.0
	
	// Clamp between 0.5 and 2.0
	multiplier := math.Max(0.5, math.Min(2.0, avgFactor*0.5+0.5))
	
	return multiplier
}

// GetToolResourceMultiplier calculates the speed multiplier based on tool resources.
// Tools with higher resource values (from upgrades) perform operations faster.
// The interpretation: higher tool resources = more powerful tool = faster execution.
func (s *ProgressService) GetToolResourceMultiplier(toolResources models.ToolResources) float64 {
	// Base tool resource values (average tool stats from tools.json)
	// Simple tools: CPU ~15-20, Bandwidth ~0.2-0.4, RAM ~4-8
	// Medium tools: CPU ~25-35, Bandwidth ~0.5-0.8, RAM ~10-14
	// Advanced tools: CPU ~45-60, Bandwidth ~0.8-1.2, RAM ~16-20
	// Using medium tool values as baseline (1.0 multiplier)
	baseCPU := 25.0
	baseBandwidth := 0.5
	baseRAM := 10.0
	
	// If tool has no resources defined (legacy tools), return neutral multiplier
	if toolResources.CPU == 0 && toolResources.Bandwidth == 0 && toolResources.RAM == 0 {
		return 1.0
	}
	
	// Normalize tool resources against base values
	cpuFactor := toolResources.CPU / baseCPU
	bandwidthFactor := toolResources.Bandwidth / baseBandwidth
	ramFactor := float64(toolResources.RAM) / baseRAM
	
	// Calculate average factor
	avgFactor := (cpuFactor + bandwidthFactor + ramFactor) / 3.0
	
	// Scale the factor: higher resources = better tool = faster
	// avgFactor of 1.0 (medium tool) -> multiplier of 1.0
	// avgFactor of 0.5 (weak tool) -> multiplier of ~0.75
	// avgFactor of 2.0 (powerful tool) -> multiplier of ~1.5
	multiplier := 0.5 + (avgFactor * 0.5)
	
	// Clamp between 0.5 and 2.0
	multiplier = math.Max(0.5, math.Min(2.0, multiplier))
	
	return multiplier
}

// GetProgressPercentage calculates progress percentage (0-100)
func (s *ProgressService) GetProgressPercentage(elapsed, total float64) float64 {
	if total <= 0 {
		return 0
	}
	percentage := (elapsed / total) * 100.0
	if percentage > 100 {
		return 100
	}
	return percentage
}

