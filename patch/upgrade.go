// Package patch provides upgrade type definitions and cost calculations for the progressive tool upgrade system.
package patch

import "math"

// UpgradeType represents the type of upgrade that can be applied to a tool.
type UpgradeType string

const (
	UpgradeExploit   UpgradeType = "exploit"
	UpgradeCPU       UpgradeType = "cpu"
	UpgradeRAM       UpgradeType = "ram"
	UpgradeBandwidth UpgradeType = "bandwidth"
	UpgradeFullTune  UpgradeType = "full"
)

// MaxExploitLevel is the maximum level any exploit can reach.
const MaxExploitLevel = 100

// CostScaleFactor determines how quickly upgrade costs increase.
// Cost = BaseCost * (CostScaleFactor ^ currentUpgradeCount)
const CostScaleFactor = 1.5

// MinResourceCPU is the minimum CPU a tool can have after upgrades.
const MinResourceCPU = 1.0

// MinResourceRAM is the minimum RAM a tool can have after upgrades.
const MinResourceRAM = 1

// MinResourceBandwidth is the minimum bandwidth a tool can have after upgrades.
const MinResourceBandwidth = 0.1

// UpgradeEffect represents the stat changes from an upgrade.
type UpgradeEffect struct {
	ExploitBoost       int     // Added to all exploit levels on the tool
	CPUReduction       float64 // Subtracted from CPU usage
	RAMReduction       int     // Subtracted from RAM usage
	BandwidthReduction float64 // Subtracted from bandwidth usage
}

// UpgradeDefinition defines a type of upgrade with its name, description, cost, and effect.
type UpgradeDefinition struct {
	Type        UpgradeType
	Name        string
	Description string
	BaseCost    float64
	Effect      UpgradeEffect
}

// AllUpgrades returns all available upgrade definitions.
func AllUpgrades() []UpgradeDefinition {
	return []UpgradeDefinition{
		{
			Type:        UpgradeExploit,
			Name:        "Exploit Booster",
			Description: "+3 exploit level to all exploits on tool",
			BaseCost:    100,
			Effect: UpgradeEffect{
				ExploitBoost: 3,
			},
		},
		{
			Type:        UpgradeCPU,
			Name:        "CPU Optimizer",
			Description: "-2 CPU usage",
			BaseCost:    50,
			Effect: UpgradeEffect{
				CPUReduction: 2,
			},
		},
		{
			Type:        UpgradeRAM,
			Name:        "RAM Optimizer",
			Description: "-1 RAM usage",
			BaseCost:    50,
			Effect: UpgradeEffect{
				RAMReduction: 1,
			},
		},
		{
			Type:        UpgradeBandwidth,
			Name:        "Bandwidth Optimizer",
			Description: "-0.1 bandwidth usage",
			BaseCost:    75,
			Effect: UpgradeEffect{
				BandwidthReduction: 0.1,
			},
		},
		{
			Type:        UpgradeFullTune,
			Name:        "Full Tune-up",
			Description: "-1 CPU, -1 RAM, -0.05 bandwidth",
			BaseCost:    150,
			Effect: UpgradeEffect{
				CPUReduction:       1,
				RAMReduction:       1,
				BandwidthReduction: 0.05,
			},
		},
	}
}

// GetUpgradeDefinition returns the definition for a specific upgrade type.
func GetUpgradeDefinition(upgradeType UpgradeType) *UpgradeDefinition {
	for _, def := range AllUpgrades() {
		if def.Type == upgradeType {
			return &def
		}
	}
	return nil
}

// CalculateUpgradeCost calculates the cost of an upgrade based on how many have already been applied.
// Cost = BaseCost * (CostScaleFactor ^ currentCount)
func CalculateUpgradeCost(upgradeType UpgradeType, currentCount int) float64 {
	def := GetUpgradeDefinition(upgradeType)
	if def == nil {
		return 0
	}
	return def.BaseCost * math.Pow(CostScaleFactor, float64(currentCount))
}

// ParseUpgradeType converts a string to an UpgradeType.
func ParseUpgradeType(s string) (UpgradeType, bool) {
	switch s {
	case "exploit":
		return UpgradeExploit, true
	case "cpu":
		return UpgradeCPU, true
	case "ram":
		return UpgradeRAM, true
	case "bandwidth", "bw":
		return UpgradeBandwidth, true
	case "full":
		return UpgradeFullTune, true
	default:
		return "", false
	}
}
