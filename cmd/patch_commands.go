package cmd

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/ui"
)

// handlePATCH handles patch-related commands
func (h *CommandHandler) handlePATCH(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if h.patchService == nil {
		return &CommandResult{Error: fmt.Errorf("patch service not available")}
	}

	if len(args) == 0 {
		return h.handlePatchList()
	}

	if args[0] == "info" && len(args) == 2 {
		return h.handlePatchInfo(args[1])
	}

	if len(args) == 2 {
		// patch <patchName> <toolName>
		return h.handlePatchApply(args[0], args[1])
	}

	return &CommandResult{Error: fmt.Errorf("usage: patches - List available patches\n       patch <patchName> <toolName> - Apply patch to tool\n       patch info <patchName> - Show patch details")}
}

// handlePatchList lists all available patches
func (h *CommandHandler) handlePatchList() *CommandResult {
	// Get all patches
	allPatches, err := h.patchService.GetAllPatches()
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Get user's owned patches
	ownedPatches, err := h.patchService.GetUserPatches(h.user.ID)
	if err != nil {
		ownedPatches = []models.Patch{} // Continue even if error
	}

	ownedMap := make(map[string]bool)
	for _, patch := range ownedPatches {
		ownedMap[patch.Name] = true
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Available Patches:", "🔧"))
	
	if len(allPatches) == 0 {
		output.WriteString("  No patches available.\n")
		output.WriteString("  Patches can be found in shop files or on servers.\n")
	} else {
		for _, patch := range allPatches {
			owned := ""
			if ownedMap[patch.Name] {
				owned = " " + ui.SuccessStyleNoBold.Render("[OWNED]")
			}
			output.WriteString(ui.FormatListBulletWithStyle(ui.AccentStyle.Render(patch.Name)+owned, ui.ListStyle))
			output.WriteString(ui.FormatKeyValuePair("    Target:", patch.TargetTool) + "\n")
			if patch.Description != "" {
				output.WriteString("    " + ui.ValueStyle.Render(patch.Description) + "\n")
			}
			if len(patch.Upgrades.Exploits) > 0 {
				output.WriteString(ui.LabelStyle.Render("    Exploit upgrades:") + " ")
				for i, exploit := range patch.Upgrades.Exploits {
					if i > 0 {
						output.WriteString(", ")
					}
					output.WriteString(ui.ValueStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
				}
				output.WriteString("\n")
			}
			if patch.Upgrades.Resources.CPU != 0 || patch.Upgrades.Resources.Bandwidth != 0 || patch.Upgrades.Resources.RAM != 0 {
				output.WriteString(ui.LabelStyle.Render("    Resource changes:") + " ")
				changes := []string{}
				if patch.Upgrades.Resources.CPU != 0 {
					changes = append(changes, ui.ValueStyle.Render(fmt.Sprintf("CPU %+.1f", patch.Upgrades.Resources.CPU)))
				}
				if patch.Upgrades.Resources.Bandwidth != 0 {
					changes = append(changes, ui.ValueStyle.Render(fmt.Sprintf("Bandwidth %+.1f", patch.Upgrades.Resources.Bandwidth)))
				}
				if patch.Upgrades.Resources.RAM != 0 {
					changes = append(changes, ui.ValueStyle.Render(fmt.Sprintf("RAM %+d", patch.Upgrades.Resources.RAM)))
				}
				output.WriteString(strings.Join(changes, ", "))
				output.WriteString("\n")
			}
			output.WriteString("\n")
		}
	}

	output.WriteString(ui.FormatUsage("Usage: patch <patchName> <toolName> - Apply patch to tool"))
	output.WriteString(ui.FormatUsage("       patch info <patchName> - Show detailed patch information"))

	return &CommandResult{Output: output.String()}
}

// handlePatchApply applies a patch to a tool
func (h *CommandHandler) handlePatchApply(patchName, toolName string) *CommandResult {
	// Check if user owns the tool
	if !h.toolService.UserHasTool(h.user.ID, toolName) {
		return &CommandResult{Error: fmt.Errorf("tool %s not owned", toolName)}
	}

	// Check if user owns the patch (if it's a purchased patch)
	if !h.patchService.UserOwnsPatch(h.user.ID, patchName) {
		// Patch might be free/discoverable, continue anyway
	}

	// Apply patch
	if err := h.patchService.ApplyPatch(h.user.ID, toolName, patchName); err != nil {
		return &CommandResult{Error: err}
	}

	// Get updated tool state to show version
	toolState, err := h.toolService.GetUserToolState(h.user.ID, toolName)
		if err == nil {
		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Patch ") + ui.AccentStyle.Render(patchName) + ui.SuccessStyle.Render(fmt.Sprintf(" successfully applied to %s", toolName)) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Tool version:", fmt.Sprintf("%d", toolState.Version)) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Applied patches:", strings.Join(toolState.AppliedPatches, ", ")) + "\n")
		return &CommandResult{Output: output.String()}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Patch ") + ui.AccentStyle.Render(patchName) + ui.SuccessStyle.Render(fmt.Sprintf(" successfully applied to %s", toolName)) + "\n")
	return &CommandResult{Output: output.String()}
}

// handlePatchInfo shows detailed patch information
func (h *CommandHandler) handlePatchInfo(patchName string) *CommandResult {
	patch, err := h.patchService.GetPatchByName(patchName)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("patch not found: %s", patchName)}
	}

	var output strings.Builder
	output.WriteString("╔═══════════════════════════════════════╗\n")
	output.WriteString("║   " + ui.HeaderStyle.Render("Patch: ") + ui.AccentBoldStyle.Render(patch.Name) + "\n")
	output.WriteString("╚═══════════════════════════════════════╝\n\n")
	output.WriteString(ui.FormatKeyValuePair("Target Tool:", patch.TargetTool) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Description:", patch.Description) + "\n\n")

	if len(patch.Upgrades.Exploits) > 0 {
		output.WriteString(ui.FormatSectionHeader("Exploit Upgrades:", ""))
		for _, exploit := range patch.Upgrades.Exploits {
			output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level))))
		}
		output.WriteString("\n")
	}

	if patch.Upgrades.Resources.CPU != 0 || patch.Upgrades.Resources.Bandwidth != 0 || patch.Upgrades.Resources.RAM != 0 {
		output.WriteString(ui.FormatSectionHeader("Resource Changes:", ""))
		if patch.Upgrades.Resources.CPU != 0 {
			output.WriteString(ui.FormatKeyValuePair("  CPU:", fmt.Sprintf("%+.1f", patch.Upgrades.Resources.CPU)) + "\n")
		}
		if patch.Upgrades.Resources.Bandwidth != 0 {
			output.WriteString(ui.FormatKeyValuePair("  Bandwidth:", fmt.Sprintf("%+.1f", patch.Upgrades.Resources.Bandwidth)) + "\n")
		}
		if patch.Upgrades.Resources.RAM != 0 {
			output.WriteString(ui.FormatKeyValuePair("  RAM:", fmt.Sprintf("%+d", patch.Upgrades.Resources.RAM)) + "\n")
		}
		output.WriteString("\n")
	}

	return &CommandResult{Output: output.String()}
}

