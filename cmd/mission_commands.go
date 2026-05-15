package cmd

import (
	"fmt"
	"sort"
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
	case "stop":
		if len(args) < 2 {
			return &CommandResult{Error: fmt.Errorf("usage: mission stop <missionID>")}
		}
		return h.handleMissionStop(args[1])
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
	// Re-check completion so missions done in any order (e.g. before trigger) can complete when viewing
	missionCompleted := h.missionService.TryAutoComplete(h.user.ID)

	userLevel := h.user.Level
	availableMissions := h.missionService.GetAvailableMissions(h.user.ID, userLevel)
	userMissions, _ := h.missionService.GetUserMissions(h.user.ID)
	
	userMissionMap := make(map[string]string) // missionID -> status
	for _, um := range userMissions {
		userMissionMap[um.MissionID] = um.Status
	}

	var output strings.Builder

	// Split into story missions (trigger-based) and board missions (manual start)
	var storyMissions, boardMissions []models.Mission
	for _, m := range availableMissions {
		if m.Trigger != nil {
			storyMissions = append(storyMissions, m)
		} else {
			boardMissions = append(boardMissions, m)
		}
	}

	if len(storyMissions) > 0 {
		output.WriteString(ui.FormatSectionHeader("Story Missions:", "📖"))
		output.WriteString(ui.DimStyle.Render("(Start automatically on triggers)\n\n"))
		for _, mission := range storyMissions {
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
			// Line 1: ID - Name - Description (keeps related content together)
			output.WriteString(fmt.Sprintf("  %s %s - %s - %s\n",
				statusIcon, statusStyle.Render(mission.ID), mission.Name, ui.ValueStyle.Render(mission.Description)))
			// Line 2: Trigger hint (when applicable)
			if mission.Trigger != nil && mission.Trigger.Type == "cat_file" && mission.Trigger.Path != "" {
				output.WriteString(fmt.Sprintf("    %s\n", ui.DimStyle.Render("starts when you: cat "+mission.Trigger.Path)))
			}
		}
		output.WriteString("\n")
	}

	if len(boardMissions) > 0 {
		output.WriteString(ui.FormatSectionHeader("Mission Board:", "📋"))
		output.WriteString(ui.DimStyle.Render("(Use 'mission start <id>' to accept)\n\n"))
		arcs := make(map[string][]models.Mission)
		for _, m := range boardMissions {
			arcs[m.ArcName] = append(arcs[m.ArcName], m)
		}
		// Sort arcs: story arcs first, "Procedurally Generated Missions" last
		arcNames := make([]string, 0, len(arcs))
		for name := range arcs {
			arcNames = append(arcNames, name)
		}
		sort.Slice(arcNames, func(i, j int) bool {
			if arcNames[i] == "Procedurally Generated Missions" {
				return false
			}
			if arcNames[j] == "Procedurally Generated Missions" {
				return true
			}
			return arcNames[i] < arcNames[j]
		})
		for _, arcName := range arcNames {
			missions := arcs[arcName]
			output.WriteString(ui.AccentBoldStyle.Render("  "+arcName) + "\n")
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
				// Procedural missions: show "Name (id)" so the readable name comes first
				var firstLine string
				if strings.HasPrefix(mission.ID, "generated_mission_") {
					firstLine = fmt.Sprintf("    %s %s (%s)\n", statusIcon, mission.Name, statusStyle.Render(mission.ID))
				} else {
					firstLine = fmt.Sprintf("    %s %s - %s\n", statusIcon, statusStyle.Render(mission.ID), mission.Name)
				}
				output.WriteString(firstLine)
				output.WriteString(fmt.Sprintf("      %s\n", ui.ValueStyle.Render(mission.Description)))
			}
			output.WriteString("\n")
		}
	}

	if len(availableMissions) == 0 {
		output.WriteString("No missions available. Complete the getting started tutorial.\n")
	}

	output.WriteString(ui.FormatKeyValuePair("Commands:", "mission <id> - View mission details") + "\n")
	output.WriteString(ui.FormatKeyValuePair("", "mission start <id> - Accept a board mission") + "\n")
	output.WriteString(ui.FormatKeyValuePair("", "mission stop <id> - Abandon a mission") + "\n")
	output.WriteString(ui.FormatKeyValuePair("", "mission status - View your progress") + "\n")

	return &CommandResult{Output: output.String(), MissionCompleted: missionCompleted}
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
		if mission.Trigger != nil && mission.Trigger.Type == "cat_file" && mission.Trigger.Path != "" {
			output.WriteString(ui.FormatKeyValuePair("Starts when you:", fmt.Sprintf("cat %s", mission.Trigger.Path)) + "\n")
		} else {
			output.WriteString(ui.FormatKeyValuePair("To accept:", fmt.Sprintf("mission start %s", missionID)) + "\n")
		}
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
	output.WriteString(ui.FormatKeyValuePair("View progress:", "mission status") + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleMissionStop(missionID string) *CommandResult {
	if err := h.missionService.StopMission(h.user.ID, missionID); err != nil {
		return &CommandResult{Error: err}
	}
	mission, _ := h.missionService.GetMissionByID(missionID)
	name := missionID
	if mission != nil {
		name = mission.Name
	}
	return &CommandResult{Output: ui.SuccessStyle.Render("Mission abandoned: ") + name + "\n"}
}

func (h *CommandHandler) handleMissionStatus() *CommandResult {
	// Re-check completion so missions done in any order can complete when viewing progress
	missionCompleted := h.missionService.TryAutoComplete(h.user.ID)

	var output strings.Builder

	// Show Story Arc Progress
	arcProgress := h.missionService.GetStoryArcProgress(h.user.ID)
	if len(arcProgress) > 0 {
		output.WriteString(ui.FormatSectionHeader("Story Arc Progress:", "📖"))
		output.WriteString("\n")
		
		for _, arc := range arcProgress {
			statusIcon := "🔄"
			var statusStyle lipgloss.Style = ui.WarningStyle
			if arc.IsCompleted {
				statusIcon = "✅"
				statusStyle = ui.SuccessStyleNoBold
			}
			
			output.WriteString(fmt.Sprintf("  %s %s\n", statusIcon, ui.AccentBoldStyle.Render(arc.ArcName)))
			output.WriteString(fmt.Sprintf("     %s (%d/%d missions)\n", 
				statusStyle.Render(fmt.Sprintf("%d%%", arc.PercentComplete)),
				arc.CompletedCount, arc.TotalMissions))
		}
		output.WriteString("\n")
	}
	
	// Show Endless Mode Status
	endlessMode := h.missionService.GetEndlessModeStatus(h.user.ID)
	if endlessMode.IsUnlocked {
		output.WriteString(ui.FormatSectionHeader("Endless Mode:", "♾️"))
		output.WriteString("\n")
		output.WriteString(ui.SuccessStyle.Render("  ✓ UNLOCKED") + "\n")
		output.WriteString(fmt.Sprintf("  Story Arcs Completed: %d/%d\n", endlessMode.CompletedArcs, endlessMode.TotalArcs))
		output.WriteString(fmt.Sprintf("  Procedural Missions: %d\n", endlessMode.ProceduralMissions))
		output.WriteString(fmt.Sprintf("  Highest Tier Reached: %d/10\n", endlessMode.HighestTierReached))
		output.WriteString(fmt.Sprintf("  Servers Exploited: %d\n", endlessMode.ServersExploited))
		output.WriteString("\n")
	} else if len(arcProgress) > 0 {
		output.WriteString(ui.DimStyle.Render("  Endless Mode: Complete a story arc to unlock") + "\n\n")
	}
	
	// Show Current Mission Progress
	userMissions, err := h.missionService.GetUserMissions(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	output.WriteString(ui.FormatSectionHeader("Active Missions:", "📊"))
	output.WriteString("\n")

	if len(userMissions) == 0 {
		output.WriteString("  No missions started yet. Use 'mission' to see available missions.\n")
		return &CommandResult{Output: output.String(), MissionCompleted: missionCompleted}
	}

	hasActive := false
	for _, um := range userMissions {
		if um.Status == "completed" {
			continue // Skip completed missions in active list
		}
		hasActive = true
		
		mission, err := h.missionService.GetMissionByID(um.MissionID)
		if err != nil {
			continue
		}

		var statusStyle lipgloss.Style
		switch um.Status {
		case "in_progress":
			statusStyle = ui.WarningStyle
		default:
			statusStyle = ui.GrayStyle
		}

		output.WriteString(ui.FormatKeyValuePair("  Mission:", mission.Name) + "\n")
		output.WriteString(ui.FormatKeyValuePair("    Status:", statusStyle.Render(um.Status)) + "\n")
		
		// Calculate real-time progress from action tracker
		realProgress := h.missionService.GetMissionProgress(h.user.ID, um.MissionID)
		output.WriteString(ui.FormatKeyValuePair("    Progress:", fmt.Sprintf("%d%%", realProgress)) + "\n")
		
		// Show incomplete objectives
		incomplete := h.missionService.GetIncompleteObjectives(h.user.ID, um.MissionID)
		if len(incomplete) > 0 {
			output.WriteString("    " + ui.DimStyle.Render("Remaining:") + "\n")
			for _, obj := range incomplete {
				output.WriteString("      • " + ui.DimStyle.Render(obj.Description) + "\n")
			}
		}
		output.WriteString("\n")
	}
	
	if !hasActive {
		output.WriteString("  No active missions. Use 'mission' to see available missions.\n")
	}

	return &CommandResult{Output: output.String(), MissionCompleted: missionCompleted}
}
