package cmd

import (
	"fmt"
	"terminal-sh/models"
	"strings"
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
	output.WriteString("Available Patches:\n\n")

	if len(allPatches) == 0 {
		output.WriteString("  No patches available.\n")
		output.WriteString("  Patches can be found in shop files or on servers.\n")
	} else {
		for _, patch := range allPatches {
			owned := ""
			if ownedMap[patch.Name] {
				owned = " [OWNED]"
			}
			output.WriteString(fmt.Sprintf("  - %s%s\n", patch.Name, owned))
			output.WriteString(fmt.Sprintf("    Target: %s\n", patch.TargetTool))
			if patch.Description != "" {
				output.WriteString(fmt.Sprintf("    %s\n", patch.Description))
			}
			if len(patch.Upgrades.Exploits) > 0 {
				output.WriteString("    Exploit upgrades: ")
				for i, exploit := range patch.Upgrades.Exploits {
					if i > 0 {
						output.WriteString(", ")
					}
					output.WriteString(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level))
				}
				output.WriteString("\n")
			}
			if patch.Upgrades.Resources.CPU != 0 || patch.Upgrades.Resources.Bandwidth != 0 || patch.Upgrades.Resources.RAM != 0 {
				output.WriteString("    Resource changes: ")
				changes := []string{}
				if patch.Upgrades.Resources.CPU != 0 {
					changes = append(changes, fmt.Sprintf("CPU %+.1f", patch.Upgrades.Resources.CPU))
				}
				if patch.Upgrades.Resources.Bandwidth != 0 {
					changes = append(changes, fmt.Sprintf("Bandwidth %+.1f", patch.Upgrades.Resources.Bandwidth))
				}
				if patch.Upgrades.Resources.RAM != 0 {
					changes = append(changes, fmt.Sprintf("RAM %+d", patch.Upgrades.Resources.RAM))
				}
				output.WriteString(strings.Join(changes, ", "))
				output.WriteString("\n")
			}
			output.WriteString("\n")
		}
	}

	output.WriteString("Usage: patch <patchName> <toolName> - Apply patch to tool\n")
	output.WriteString("       patch info <patchName> - Show detailed patch information\n")

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
		output := fmt.Sprintf("Patch %s successfully applied to %s\n", patchName, toolName)
		output += fmt.Sprintf("Tool version: %d\n", toolState.Version)
		output += fmt.Sprintf("Applied patches: %s\n", strings.Join(toolState.AppliedPatches, ", "))
		return &CommandResult{Output: output}
	}

	return &CommandResult{Output: fmt.Sprintf("Patch %s successfully applied to %s\n", patchName, toolName)}
}

// handlePatchInfo shows detailed patch information
func (h *CommandHandler) handlePatchInfo(patchName string) *CommandResult {
	patch, err := h.patchService.GetPatchByName(patchName)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("patch not found: %s", patchName)}
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("╔═══════════════════════════════════════╗\n"))
	output.WriteString(fmt.Sprintf("║   Patch: %s\n", patch.Name))
	output.WriteString(fmt.Sprintf("╚═══════════════════════════════════════╝\n\n"))
	output.WriteString(fmt.Sprintf("Target Tool: %s\n", patch.TargetTool))
	output.WriteString(fmt.Sprintf("Description: %s\n\n", patch.Description))

	if len(patch.Upgrades.Exploits) > 0 {
		output.WriteString("Exploit Upgrades:\n")
		for _, exploit := range patch.Upgrades.Exploits {
			output.WriteString(fmt.Sprintf("  - %s (level %d)\n", exploit.Type, exploit.Level))
		}
		output.WriteString("\n")
	}

	if patch.Upgrades.Resources.CPU != 0 || patch.Upgrades.Resources.Bandwidth != 0 || patch.Upgrades.Resources.RAM != 0 {
		output.WriteString("Resource Changes:\n")
		if patch.Upgrades.Resources.CPU != 0 {
			output.WriteString(fmt.Sprintf("  CPU: %+.1f\n", patch.Upgrades.Resources.CPU))
		}
		if patch.Upgrades.Resources.Bandwidth != 0 {
			output.WriteString(fmt.Sprintf("  Bandwidth: %+.1f\n", patch.Upgrades.Resources.Bandwidth))
		}
		if patch.Upgrades.Resources.RAM != 0 {
			output.WriteString(fmt.Sprintf("  RAM: %+d\n", patch.Upgrades.Resources.RAM))
		}
		output.WriteString("\n")
	}

	return &CommandResult{Output: output.String()}
}

