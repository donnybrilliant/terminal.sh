package cmd

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/ui"

	"github.com/charmbracelet/lipgloss"
)

func (h *CommandHandler) handleMISSION(args []string) *CommandResult {
	if h.missionService == nil {
		return &CommandResult{Error: fmt.Errorf("mission service not available")}
	}

	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// If no args, list available missions
	if len(args) == 0 {
		return h.handleMissionList()
	}

	// Handle subcommands
	switch args[0] {
	case "start":
		if len(args) < 2 {
			return &CommandResult{Error: fmt.Errorf("usage: mission start <missionID>")}
		}
		return h.handleMissionStart(args[1])
	case "complete":
		if len(args) < 2 {
			return &CommandResult{Error: fmt.Errorf("usage: mission complete <missionID>")}
		}
		return h.handleMissionComplete(args[1])
	case "list":
		return h.handleMissionList()
	case "status":
		return h.handleMissionStatus()
	default:
		// Assume it's a mission ID to view
		return h.handleMissionView(args[0])
	}
}

func (h *CommandHandler) handleMissionList() *CommandResult {
	userLevel := h.user.Level
	availableMissions := h.missionService.GetAvailableMissions(h.user.ID, userLevel)
	userMissions, _ := h.missionService.GetUserMissions(h.user.ID)
	
	userMissionMap := make(map[string]string) // missionID -> status
	for _, um := range userMissions {
		userMissionMap[um.MissionID] = um.Status
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Available Missions:", "🎯"))
	output.WriteString("\n")

	if len(availableMissions) == 0 {
		output.WriteString("No missions available. Complete tutorials to unlock missions.\n")
		return &CommandResult{Output: output.String()}
	}

	// Group by arc
	arcs := make(map[string][]models.Mission)
	for _, mission := range availableMissions {
		arcs[mission.ArcName] = append(arcs[mission.ArcName], mission)
	}

	for arcName, missions := range arcs {
		output.WriteString(ui.AccentBoldStyle.Render("📖 " + arcName) + "\n")
		
		for _, mission := range missions {
			status := userMissionMap[mission.ID]
			var statusIcon string
			var statusStyle lipgloss.Style
			
			switch status {
			case "completed":
				statusIcon = "✅"
				statusStyle = ui.SuccessStyleNoBold
			case "in_progress":
				statusIcon = "🔄"
				statusStyle = ui.WarningStyle
			default:
				statusIcon = "⭕"
				statusStyle = ui.GrayStyle
			}
			
			output.WriteString(fmt.Sprintf("  %s %s - %s\n", statusIcon, statusStyle.Render(mission.ID), mission.Name))
			output.WriteString(fmt.Sprintf("    %s\n", ui.ValueStyle.Render(mission.Description)))
		}
		output.WriteString("\n")
	}

	output.WriteString(ui.FormatKeyValuePair("Commands:", "mission <id> - View mission details") + "\n")
	output.WriteString(ui.FormatKeyValuePair("", "mission start <id> - Start a mission") + "\n")
	output.WriteString(ui.FormatKeyValuePair("", "mission status - View your mission progress") + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleMissionView(missionID string) *CommandResult {
	mission, err := h.missionService.GetMissionByID(missionID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	userMission, _ := h.missionService.GetUserMission(h.user.ID, missionID)

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader(mission.Name, "🎯"))
	output.WriteString("\n")
	output.WriteString(ui.FormatKeyValuePair("Arc:", mission.ArcName) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Description:", mission.Description) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Required Level:", fmt.Sprintf("%d", mission.RequiredLevel)) + "\n")
	
	if len(mission.RequiredTools) > 0 {
		output.WriteString(ui.FormatKeyValuePair("Required Tools:", strings.Join(mission.RequiredTools, ", ")) + "\n")
	}

	if userMission != nil {
		output.WriteString(ui.FormatKeyValuePair("Status:", userMission.Status) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Progress:", fmt.Sprintf("%d%%", userMission.Progress)) + "\n")
	}

	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Objectives:", ""))
	for _, obj := range mission.Objectives {
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("%d. %s", obj.ID, obj.Description)))
		if obj.Tool != "" {
			output.WriteString(ui.FormatListBullet(fmt.Sprintf("   Tool: %s", obj.Tool)))
		}
		if obj.Hint != "" {
			// Display hint in a tutorial-like style
			hintStyle := ui.WarningStyle.Italic(true)
			output.WriteString("   " + hintStyle.Render("💡 Hint: " + obj.Hint) + "\n")
		}
	}

	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Rewards:", ""))
	rewards := mission.Rewards
	if rewards.Experience > 0 {
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("Experience: %d XP", rewards.Experience)))
	}
	if rewards.Crypto > 0 {
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("Cryptocurrency: %.2f", rewards.Crypto)))
	}
	if len(rewards.Tools) > 0 {
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("Tools: %s", strings.Join(rewards.Tools, ", "))))
	}
	if len(rewards.ToolUpgrades) > 0 {
		upgrades := []string{}
		for _, u := range rewards.ToolUpgrades {
			upgrades = append(upgrades, fmt.Sprintf("%s +%d %s", u.ToolName, u.Count, u.UpgradeType))
		}
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("Tool Upgrades: %s", strings.Join(upgrades, ", "))))
	}
	if len(rewards.Achievements) > 0 {
		output.WriteString(ui.FormatListBullet(fmt.Sprintf("Achievements: %s", strings.Join(rewards.Achievements, ", "))))
	}

	if userMission == nil || userMission.Status != "completed" {
		output.WriteString("\n")
		output.WriteString(ui.FormatKeyValuePair("To start:", fmt.Sprintf("mission start %s", missionID)) + "\n")
	}

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleMissionStart(missionID string) *CommandResult {
	if err := h.missionService.StartMission(h.user.ID, missionID); err != nil {
		return &CommandResult{Error: err}
	}

	mission, _ := h.missionService.GetMissionByID(missionID)
	
	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Mission started: ") + mission.Name + "\n")
	output.WriteString(ui.FormatKeyValuePair("Objectives:", fmt.Sprintf("%d objectives to complete", len(mission.Objectives))) + "\n")
	output.WriteString(ui.FormatKeyValuePair("View progress:", fmt.Sprintf("mission status") + "\n"))

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleMissionComplete(missionID string) *CommandResult {
	if err := h.missionService.CompleteMission(h.user.ID, missionID); err != nil {
		return &CommandResult{Error: err}
	}

	mission, _ := h.missionService.GetMissionByID(missionID)
	
	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("🎉 Mission completed: ") + mission.Name + "\n")
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Rewards granted:", ""))
	rewards := mission.Rewards
	if rewards.Experience > 0 {
		output.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("+%d XP", rewards.Experience))))
	}
	if rewards.Crypto > 0 {
		output.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("+%.2f cryptocurrency", rewards.Crypto))))
	}
	if len(rewards.Tools) > 0 {
		output.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Tools unlocked: %s", strings.Join(rewards.Tools, ", ")))))
	}
	if len(rewards.ToolUpgrades) > 0 {
		upgrades := []string{}
		for _, u := range rewards.ToolUpgrades {
			upgrades = append(upgrades, fmt.Sprintf("%s +%d %s", u.ToolName, u.Count, u.UpgradeType))
		}
		output.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Tool upgrades applied: %s", strings.Join(upgrades, ", ")))))
	}
	if len(rewards.Achievements) > 0 {
		output.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Achievements unlocked: %s", strings.Join(rewards.Achievements, ", ")))))
	}

	if len(mission.Unlocks) > 0 {
		output.WriteString("\n")
		output.WriteString(ui.FormatKeyValuePair("New missions unlocked:", strings.Join(mission.Unlocks, ", ")) + "\n")
	}

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleMissionStatus() *CommandResult {
	userMissions, err := h.missionService.GetUserMissions(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Your Mission Progress:", "📊"))
	output.WriteString("\n")

	if len(userMissions) == 0 {
		output.WriteString("No missions started yet. Use 'mission' to see available missions.\n")
		return &CommandResult{Output: output.String()}
	}

	for _, um := range userMissions {
		mission, err := h.missionService.GetMissionByID(um.MissionID)
		if err != nil {
			continue
		}

		var statusStyle lipgloss.Style
		switch um.Status {
		case "completed":
			statusStyle = ui.SuccessStyleNoBold
		case "in_progress":
			statusStyle = ui.WarningStyle
		default:
			statusStyle = ui.GrayStyle
		}

		output.WriteString(ui.FormatKeyValuePair("Mission:", mission.Name) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Status:", statusStyle.Render(um.Status)) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Progress:", fmt.Sprintf("%d%%", um.Progress)) + "\n")
		output.WriteString("\n")
	}

	return &CommandResult{Output: output.String()}
}
