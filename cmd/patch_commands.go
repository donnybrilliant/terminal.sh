package cmd

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/patch"
	"terminal-sh/ui"
)

// handlePATCH handles patch-related commands for the progressive upgrade system
func (h *CommandHandler) handlePATCH(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if h.upgradeService == nil {
		return &CommandResult{Error: fmt.Errorf("upgrade service not available")}
	}

	// No args - list all tools with upgrade status
	if len(args) == 0 {
		return h.handlePatchList()
	}

	toolName := args[0]

	// Single arg - show upgrades for a specific tool
	if len(args) == 1 {
		return h.handlePatchTool(toolName)
	}

	// Two args - apply upgrade to tool
	if len(args) == 2 {
		upgradeTypeStr := args[1]
		return h.handlePatchApply(toolName, upgradeTypeStr)
	}

	return &CommandResult{Error: fmt.Errorf("usage: patches - List all tools with upgrade status\n       patch <tool> - Show available upgrades for a tool\n       patch <tool> <type> - Apply upgrade (exploit/cpu/ram/bw/full)")}
}

// handlePatchList lists all owned tools with their upgrade status
func (h *CommandHandler) handlePatchList() *CommandResult {
	// Get user's tool states
	var toolStates []models.UserToolState
	if err := h.db.Where("user_id = ?", h.user.ID).Preload("Tool").Find(&toolStates).Error; err != nil {
		return &CommandResult{Error: fmt.Errorf("failed to get tools: %w", err)}
	}

	if len(toolStates) == 0 {
		return &CommandResult{Output: "No tools owned yet. Download tools from the repo server.\n"}
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Your Tools & Upgrades:", "🔧"))

	for _, ts := range toolStates {
		totalUpgrades := ts.ExploitUpgrades + ts.CPUUpgrades + ts.RAMUpgrades + ts.BandwidthUpgrades

		output.WriteString(ui.FormatListBulletWithStyle(ui.AccentStyle.Render(ts.Tool.Name), ui.ListStyle))
		output.WriteString(fmt.Sprintf("    Version: %d | Total Upgrades: %d\n", ts.Version, totalUpgrades))

		if totalUpgrades > 0 {
			upgrades := []string{}
			if ts.ExploitUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d exploit", ts.ExploitUpgrades))
			}
			if ts.CPUUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d cpu", ts.CPUUpgrades))
			}
			if ts.RAMUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d ram", ts.RAMUpgrades))
			}
			if ts.BandwidthUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d bw", ts.BandwidthUpgrades))
			}
			output.WriteString("    " + ui.ValueStyle.Render("Upgrades: "+strings.Join(upgrades, ", ")) + "\n")
		}

		// Show effective stats
		if len(ts.EffectiveExploits) > 0 {
			maxLevel := 0
			for _, e := range ts.EffectiveExploits {
				if e.Level > maxLevel {
					maxLevel = e.Level
				}
			}
			output.WriteString(fmt.Sprintf("    Exploit Level: %d/%d | CPU: %.0f | RAM: %d | BW: %.1f\n",
				maxLevel, patch.MaxExploitLevel,
				ts.EffectiveResources.CPU, ts.EffectiveResources.RAM, ts.EffectiveResources.Bandwidth))
		}
		output.WriteString("\n")
	}

	output.WriteString(ui.FormatUsage("Usage: patch <tool> - View upgrade options for a tool"))

	return &CommandResult{Output: output.String()}
}

// handlePatchTool shows available upgrades for a specific tool
func (h *CommandHandler) handlePatchTool(toolName string) *CommandResult {
	// Get upgrade info
	info, err := h.upgradeService.GetToolUpgradeInfo(h.user.ID, toolName)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Get user's wallet
	user, _ := h.userService.GetUserByID(h.user.ID)

	var output strings.Builder
	output.WriteString("╔═══════════════════════════════════════╗\n")
	output.WriteString("║   " + ui.HeaderStyle.Render("Tool Patches: ") + ui.AccentBoldStyle.Render(toolName) + "\n")
	output.WriteString("╚═══════════════════════════════════════╝\n\n")

	// Current stats
	output.WriteString(ui.FormatSectionHeader("Current Stats:", "📊"))
	if len(info.EffectiveExploits) > 0 {
		output.WriteString(fmt.Sprintf("  Exploit Level: %d (max %d)\n", info.MaxExploitLevel, patch.MaxExploitLevel))
	}
	output.WriteString(fmt.Sprintf("  CPU: %.0f | RAM: %d | Bandwidth: %.1f\n",
		info.EffectiveResources.CPU, info.EffectiveResources.RAM, info.EffectiveResources.Bandwidth))

	if info.TotalUpgrades > 0 {
		upgrades := []string{}
		if info.ExploitUpgrades > 0 {
			upgrades = append(upgrades, fmt.Sprintf("%d exploit", info.ExploitUpgrades))
		}
		if info.CPUUpgrades > 0 {
			upgrades = append(upgrades, fmt.Sprintf("%d cpu", info.CPUUpgrades))
		}
		if info.RAMUpgrades > 0 {
			upgrades = append(upgrades, fmt.Sprintf("%d ram", info.RAMUpgrades))
		}
		if info.BandwidthUpgrades > 0 {
			upgrades = append(upgrades, fmt.Sprintf("%d bandwidth", info.BandwidthUpgrades))
		}
		output.WriteString("  Upgrades Applied: " + ui.ValueStyle.Render(strings.Join(upgrades, ", ")) + "\n")
	}
	output.WriteString("\n")

	// Available upgrades
	output.WriteString(ui.FormatSectionHeader("Available Patches:", "🔧"))

	for _, upgrade := range info.AvailableUpgrades {
		typeKey := string(upgrade.Type)
		if upgrade.Type == patch.UpgradeBandwidth {
			typeKey = "bw"
		}

		costStr := fmt.Sprintf("%.0f crypto", upgrade.Cost)
		if !upgrade.CanApply {
			costStr = ui.ErrorStyle.Render(upgrade.Reason)
		}

		output.WriteString(fmt.Sprintf("  [%s] %s\n", ui.AccentStyle.Render(typeKey), upgrade.Name))
		output.WriteString(fmt.Sprintf("       %s\n", ui.ValueStyle.Render(upgrade.Description)))
		output.WriteString(fmt.Sprintf("       Cost: %s\n", costStr))
	}

	output.WriteString("\n")
	output.WriteString(ui.FormatKeyValuePair("Your wallet:", fmt.Sprintf("%.0f crypto", user.Wallet.Crypto)) + "\n\n")
	output.WriteString(ui.FormatUsage("Usage: patch " + toolName + " <type>"))
	output.WriteString(ui.FormatUsage("Types: exploit, cpu, ram, bw, full"))

	return &CommandResult{Output: output.String()}
}

// handlePatchApply applies an upgrade to a tool
func (h *CommandHandler) handlePatchApply(toolName, upgradeTypeStr string) *CommandResult {
	// Parse upgrade type
	upgradeType, valid := patch.ParseUpgradeType(upgradeTypeStr)
	if !valid {
		return &CommandResult{Error: fmt.Errorf("invalid upgrade type: %s\nValid types: exploit, cpu, ram, bw, full", upgradeTypeStr)}
	}

	// Check if user owns the tool
	if !h.toolService.UserHasTool(h.user.ID, toolName) {
		return &CommandResult{Error: fmt.Errorf("tool %s not owned", toolName)}
	}

	// Get the cost before applying
	toolState, _ := h.toolService.GetUserToolState(h.user.ID, toolName)
	currentCount := 0
	switch upgradeType {
	case patch.UpgradeExploit:
		currentCount = toolState.ExploitUpgrades
	case patch.UpgradeCPU:
		currentCount = toolState.CPUUpgrades
	case patch.UpgradeRAM:
		currentCount = toolState.RAMUpgrades
	case patch.UpgradeBandwidth:
		currentCount = toolState.BandwidthUpgrades
	case patch.UpgradeFullTune:
		currentCount = toolState.CPUUpgrades + toolState.RAMUpgrades + toolState.BandwidthUpgrades
	}
	cost := patch.CalculateUpgradeCost(upgradeType, currentCount)

	// Apply upgrade
	if err := h.upgradeService.ApplyUpgrade(h.user.ID, toolName, upgradeType); err != nil {
		return &CommandResult{Error: err}
	}

	// Get updated tool state
	toolState, _ = h.toolService.GetUserToolState(h.user.ID, toolName)
	def := patch.GetUpgradeDefinition(upgradeType)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ "+def.Name+" applied to "+toolName) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Cost:", fmt.Sprintf("%.0f crypto", cost)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Tool version:", fmt.Sprintf("%d", toolState.Version)) + "\n")

	// Show new stats
	if len(toolState.EffectiveExploits) > 0 {
		maxLevel := 0
		for _, e := range toolState.EffectiveExploits {
			if e.Level > maxLevel {
				maxLevel = e.Level
			}
		}
		output.WriteString(fmt.Sprintf("New stats: Exploit %d | CPU %.0f | RAM %d | BW %.1f\n",
			maxLevel, toolState.EffectiveResources.CPU, toolState.EffectiveResources.RAM, toolState.EffectiveResources.Bandwidth))
	}

	return &CommandResult{Output: output.String()}
}
