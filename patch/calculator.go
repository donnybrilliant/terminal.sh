// Package patch provides pure calculation logic for progressive tool upgrades.
package patch

import "terminal-sh/models"

// Calculator calculates effective tool stats from base stats + applied upgrades.
// It is a pure calculator with no database or side effects.
type Calculator struct{}

// NewCalculator creates a new Calculator instance.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// ToolStats represents the stats needed for calculation (subset of models.Tool).
type ToolStats struct {
	Exploits  []models.Exploit
	Resources models.ToolResources
}

// CalculateWithUpgrades applies progressive upgrades to base tool stats.
// exploitUpgrades: number of exploit boost upgrades applied (+3 level each)
// cpuUpgrades: number of CPU optimizer upgrades applied (-2 CPU each)
// ramUpgrades: number of RAM optimizer upgrades applied (-1 RAM each)
// bandwidthUpgrades: number of bandwidth optimizer upgrades applied (-0.1 bandwidth each)
func (c *Calculator) CalculateWithUpgrades(
	base ToolStats,
	exploitUpgrades, cpuUpgrades, ramUpgrades, bandwidthUpgrades int,
) ToolStats {
	// Start with a copy of base stats
	effective := ToolStats{
		Exploits:  make([]models.Exploit, len(base.Exploits)),
		Resources: base.Resources,
	}
	copy(effective.Exploits, base.Exploits)

	// Apply progressive exploit upgrades (+3 per upgrade, capped at MaxExploitLevel)
	for i := range effective.Exploits {
		newLevel := effective.Exploits[i].Level + (exploitUpgrades * 3)
		if newLevel > MaxExploitLevel {
			newLevel = MaxExploitLevel
		}
		effective.Exploits[i].Level = newLevel
	}

	// Apply resource upgrades
	effective.Resources.CPU -= float64(cpuUpgrades) * 2
	effective.Resources.RAM -= ramUpgrades * 1
	effective.Resources.Bandwidth -= float64(bandwidthUpgrades) * 0.1

	// Clamp to minimums (tools always need some resources)
	if effective.Resources.CPU < MinResourceCPU {
		effective.Resources.CPU = MinResourceCPU
	}
	if effective.Resources.RAM < MinResourceRAM {
		effective.Resources.RAM = MinResourceRAM
	}
	if effective.Resources.Bandwidth < MinResourceBandwidth {
		effective.Resources.Bandwidth = MinResourceBandwidth
	}

	return effective
}

// ToolStatsFromTool extracts ToolStats from a models.Tool.
func ToolStatsFromTool(tool *models.Tool) ToolStats {
	exploits := make([]models.Exploit, len(tool.Exploits))
	copy(exploits, tool.Exploits)
	return ToolStats{
		Exploits:  exploits,
		Resources: tool.Resources,
	}
}

// GetMaxExploitLevel returns the highest exploit level on a tool.
func GetMaxExploitLevel(stats ToolStats) int {
	maxLevel := 0
	for _, exploit := range stats.Exploits {
		if exploit.Level > maxLevel {
			maxLevel = exploit.Level
		}
	}
	return maxLevel
}

// CanApplyExploitUpgrade checks if any exploit on the tool is below max level.
func CanApplyExploitUpgrade(stats ToolStats) bool {
	for _, exploit := range stats.Exploits {
		if exploit.Level < MaxExploitLevel {
			return true
		}
	}
	return false
}
