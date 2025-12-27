package services

import (
	"math"
	"terminal-sh/models"
)

// OperationType represents the type of operation
type OperationType string

const (
	OperationDownload OperationType = "download"
	OperationExploit  OperationType = "exploit"
	OperationSSH      OperationType = "ssh"
	OperationTransfer OperationType = "transfer"
)

// ProgressService handles progress calculation for operations
type ProgressService struct{}

// NewProgressService creates a new progress service
func NewProgressService() *ProgressService {
	return &ProgressService{}
}

// CalculateOperationTime calculates the time needed for an operation based on user resources
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

