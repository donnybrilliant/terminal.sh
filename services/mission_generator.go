// Package services provides business logic services for the terminal.sh game.
// This file implements the Mission Generator, which creates infinite missions
// when static missions are exhausted, ensuring players always have content.
package services

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

const (
	// MISSIONS_PER_USER is the target number of procedurally generated missions to maintain per user
	MISSIONS_PER_USER = 5
)

// PlayerState represents analyzed player state for mission generation.
// This data is used to create missions appropriate for the player's skill level.
type PlayerState struct {
	Level             int      // Player's current level
	OwnedTools        []string // Tools the player owns
	ExploitedCount    int      // Number of servers exploited
	CompletedMissions int      // Number of missions completed
	AvailableServers  int      // Number of servers available to target
}

// MissionGenerator handles procedurally generated mission creation.
// Generates missions based on player state, ensuring infinite gameplay when
// static missions are exhausted. Missions are personalized to player level,
// owned tools, and progress.
type MissionGenerator struct {
	db             *database.Database
	missionService *MissionService
	serverService  *ServerService
	toolService    *ToolService
	userService    *UserService
	rng            *rand.Rand
}

// NewMissionGenerator creates a new MissionGenerator
func NewMissionGenerator(db *database.Database, missionService *MissionService, serverService *ServerService, toolService *ToolService, userService *UserService) *MissionGenerator {
	return &MissionGenerator{
		db:             db,
		missionService: missionService,
		serverService:  serverService,
		toolService:    toolService,
		userService:    userService,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateMission creates a new procedurally generated mission for a user
func (g *MissionGenerator) GenerateMission(userID uuid.UUID) (*models.Mission, error) {
	// Analyze player state
	playerState, err := g.analyzePlayerState(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze player state: %w", err)
	}

	// Select mission type
	missionType := g.selectMissionType(playerState)

	// Generate objectives
	objectives := g.generateObjectives(missionType, playerState.Level, playerState)

	// Calculate rewards
	rewards := g.calculateRewards(playerState.Level, missionType, objectives)

	// Generate unique mission ID
	missionID := fmt.Sprintf("generated_mission_%s_%d", userID.String()[:8], time.Now().Unix())

	// Create mission
	mission := &models.Mission{
		ID:            missionID,
		ArcID:         "procedural_generated",
		ArcName:       "Procedurally Generated Missions",
		MissionNumber: 1,
		Name:          g.generateMissionName(missionType, playerState.Level),
		Description:   g.generateMissionDescription(missionType, playerState.Level, objectives),
		Prerequisites: []string{},
		RequiredTools: g.getRequiredTools(missionType, playerState),
		RequiredLevel: playerState.Level,
		Objectives:    objectives,
		Rewards:       rewards,
		Unlocks:       []string{},
	}

	// Store mission data as JSON for GeneratedMission tracking
	missionData, err := json.Marshal(mission)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mission data: %w", err)
	}

	// Create GeneratedMission tracking record
	generatedMission := &models.GeneratedMission{
		UserID:      userID,
		MissionID:   missionID,
		GeneratedAt: time.Now(),
		Difficulty:  playerState.Level,
		MissionData:  string(missionData),
	}

	if err := g.db.Create(generatedMission).Error; err != nil {
		return nil, fmt.Errorf("failed to create generated mission record: %w", err)
	}

	return mission, nil
}

// GenerateMissionForServer creates a mission targeting a specific server
func (g *MissionGenerator) GenerateMissionForServer(userID uuid.UUID, serverIP string) (*models.Mission, error) {
	mission, err := g.GenerateMission(userID)
	if err != nil {
		return nil, err
	}

	// Update mission to target specific server
	mission.Objectives = g.addServerTargetToObjectives(mission.Objectives, serverIP)

	// Update GeneratedMission record
	var generatedMission models.GeneratedMission
	if err := g.db.Where("user_id = ? AND mission_id = ?", userID, mission.ID).First(&generatedMission).Error; err == nil {
		generatedMission.ServerIP = serverIP
		missionData, _ := json.Marshal(mission)
		generatedMission.MissionData = string(missionData)
		g.db.Save(&generatedMission)
	}

	return mission, nil
}

// analyzePlayerState analyzes the player's current state
func (g *MissionGenerator) analyzePlayerState(userID uuid.UUID) (*PlayerState, error) {
	user, err := g.userService.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Get owned tools
	tools, err := g.toolService.GetUserTools(userID)
	if err != nil {
		return nil, err
	}
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}

	// Count exploited servers
	exploitedServers, err := g.serverService.GetAllTopLevelServers()
	if err != nil {
		exploitedServers = []models.Server{}
	}
	// For simplicity, we'll use a rough estimate - in production, check ExploitedServer table
	exploitedCount := len(exploitedServers) / 2 // Rough estimate

	// Count completed missions
	userMissions, err := g.missionService.GetUserMissions(userID)
	if err != nil {
		userMissions = []models.UserMission{}
	}
	completedCount := 0
	for _, um := range userMissions {
		if um.Status == "completed" {
			completedCount++
		}
	}

	// Count available servers
	availableServers, err := g.serverService.GetAllTopLevelServers()
	availableCount := 0
	if err == nil {
		availableCount = len(availableServers)
	}

	return &PlayerState{
		Level:              user.Level,
		OwnedTools:         toolNames,
		ExploitedCount:     exploitedCount,
		CompletedMissions:  completedCount,
		AvailableServers:   availableCount,
	}, nil
}

// selectMissionType chooses a mission type based on player state
func (g *MissionGenerator) selectMissionType(playerState *PlayerState) string {
	missionTypes := []string{
		"exploitation",
		"data_extraction",
		"tool_mastery",
		"network_exploration",
		"resource_gathering",
		"stealth",
	}

	// Filter based on level and tools
	availableTypes := []string{}
	for _, mt := range missionTypes {
		if g.isMissionTypeAvailable(mt, playerState) {
			availableTypes = append(availableTypes, mt)
		}
	}

	if len(availableTypes) == 0 {
		// Default to exploitation if nothing else available
		return "exploitation"
	}

	return availableTypes[g.rng.Intn(len(availableTypes))]
}

// isMissionTypeAvailable checks if a mission type is available for the player
func (g *MissionGenerator) isMissionTypeAvailable(missionType string, playerState *PlayerState) bool {
	switch missionType {
	case "exploitation":
		return true // Always available
	case "data_extraction":
		return playerState.Level >= 3 && playerState.AvailableServers > 0
	case "tool_mastery":
		return len(playerState.OwnedTools) > 0
	case "network_exploration":
		return playerState.Level >= 2 && playerState.AvailableServers > 0
	case "resource_gathering":
		return playerState.Level >= 1
	case "stealth":
		return playerState.Level >= 5
	default:
		return false
	}
}

// generateObjectives creates objectives based on mission type and level
func (g *MissionGenerator) generateObjectives(missionType string, level int, playerState *PlayerState) []models.MissionObjective {
	var objectives []models.MissionObjective

	switch missionType {
	case "exploitation":
		count := g.calculateObjectiveCount(level, 2, 5)
		securityLevel := g.calculateSecurityLevel(level)
		objectives = append(objectives, models.MissionObjective{
			ID:          1,
			Type:       "exploit_server",
			Description: fmt.Sprintf("Exploit %d servers with security level < %d", count, securityLevel),
		})

	case "data_extraction":
		dataAmount := g.calculateDataAmount(level)
		objectives = append(objectives, models.MissionObjective{
			ID:          1,
			Type:       "collect_data",
			Description: fmt.Sprintf("Extract %.1f GB of data from servers", dataAmount),
		})

	case "tool_mastery":
		if len(playerState.OwnedTools) > 0 {
			tool := playerState.OwnedTools[g.rng.Intn(len(playerState.OwnedTools))]
			count := g.calculateObjectiveCount(level, 3, 8)
			objectives = append(objectives, models.MissionObjective{
				ID:          1,
				Type:       "use_tool",
				Description: fmt.Sprintf("Use %s to exploit %d servers", tool, count),
				Tool:       tool,
			})
		}

	case "network_exploration":
		count := g.calculateObjectiveCount(level, 2, 4)
		objectives = append(objectives, models.MissionObjective{
			ID:          1,
			Type:       "discover_servers",
			Description: fmt.Sprintf("Discover %d servers on local networks", count),
		})

	case "resource_gathering":
		cryptoAmount := g.calculateCryptoAmount(level)
		objectives = append(objectives, models.MissionObjective{
			ID:          1,
			Type:       "mine_crypto",
			Description: fmt.Sprintf("Mine %.1f cryptocurrency", cryptoAmount),
		})

	case "stealth":
		count := g.calculateObjectiveCount(level, 2, 5)
		objectives = append(objectives, models.MissionObjective{
			ID:          1,
			Type:       "cover_tracks",
			Description: fmt.Sprintf("Cover your tracks on %d servers", count),
		})
	}

	return objectives
}

// addServerTargetToObjectives adds server targeting to objectives
func (g *MissionGenerator) addServerTargetToObjectives(objectives []models.MissionObjective, serverIP string) []models.MissionObjective {
	for i := range objectives {
		if objectives[i].Type == "exploit_server" || objectives[i].Type == "collect_data" {
			objectives[i].Description = fmt.Sprintf("%s (target: %s)", objectives[i].Description, serverIP)
		}
	}
	return objectives
}

// calculateRewards calculates rewards based on level and mission type
func (g *MissionGenerator) calculateRewards(level int, missionType string, objectives []models.MissionObjective) models.MissionRewards {
	baseXP := 100
	baseCrypto := 20.0

	// Scale with level
	experience := int(float64(baseXP) * (1.0 + float64(level)*0.1))
	crypto := baseCrypto * (1.0 + float64(level)*0.15)

	// Adjust based on mission type
	switch missionType {
	case "data_extraction":
		experience = int(float64(experience) * 1.5)
		crypto *= 1.5
	case "tool_mastery":
		experience = int(float64(experience) * 1.2)
		crypto *= 1.2
	case "stealth":
		experience = int(float64(experience) * 1.3)
		crypto *= 1.3
	}

	rewards := models.MissionRewards{
		Experience:   experience,
		Crypto:       crypto,
		Tools:        []string{},
		ToolUpgrades: []models.ToolUpgradeReward{},
		Achievements: []string{},
	}

	// Add tool unlocks for advanced missions
	if level >= 10 && g.rng.Float32() < 0.3 {
		// Could unlock a tool, but we'll leave it empty for now
		// In production, this would select from available tools
	}

	return rewards
}

// getRequiredTools returns required tools for a mission type
func (g *MissionGenerator) getRequiredTools(missionType string, playerState *PlayerState) []string {
	switch missionType {
	case "tool_mastery":
		if len(playerState.OwnedTools) > 0 {
			return []string{playerState.OwnedTools[g.rng.Intn(len(playerState.OwnedTools))]}
		}
	case "data_extraction":
		return []string{"database_dumper"}
	case "stealth":
		return []string{"log_cleaner", "timestomper"}
	}
	return []string{}
}

// Helper functions for objective generation

func (g *MissionGenerator) calculateObjectiveCount(level int, min, max int) int {
	count := min + (level / 3)
	if count > max {
		count = max
	}
	if count < min {
		count = min
	}
	return count
}

func (g *MissionGenerator) calculateSecurityLevel(level int) int {
	baseLevel := 20 + (level * 5)
	return baseLevel + g.rng.Intn(20) - 10 // Add some randomness
}

func (g *MissionGenerator) calculateDataAmount(level int) float64 {
	baseAmount := 10.0 + float64(level)*5.0
	return baseAmount + float64(g.rng.Intn(20))
}

func (g *MissionGenerator) calculateCryptoAmount(level int) float64 {
	baseAmount := 50.0 + float64(level)*10.0
	return baseAmount + float64(g.rng.Intn(50))
}

// generateMissionName creates a mission name based on type and level
func (g *MissionGenerator) generateMissionName(missionType string, level int) string {
	names := map[string][]string{
		"exploitation": {
			"Network Infiltration",
			"Server Breach Operation",
			"Target Acquisition",
			"System Penetration",
		},
		"data_extraction": {
			"Data Heist",
			"Information Extraction",
			"Database Raid",
			"Corporate Espionage",
		},
		"tool_mastery": {
			"Tool Mastery Challenge",
			"Skill Demonstration",
			"Expertise Test",
		},
		"network_exploration": {
			"Network Discovery",
			"Local Network Scan",
			"Infrastructure Mapping",
		},
		"resource_gathering": {
			"Cryptocurrency Mining",
			"Resource Extraction",
			"Digital Currency Operation",
		},
		"stealth": {
			"Ghost Protocol",
			"Cover Your Tracks",
			"Stealth Operation",
		},
	}

	typeNames, ok := names[missionType]
	if !ok || len(typeNames) == 0 {
		return "Procedurally Generated Mission"
	}

	return typeNames[g.rng.Intn(len(typeNames))]
}

// generateMissionDescription creates a mission description
func (g *MissionGenerator) generateMissionDescription(missionType string, level int, objectives []models.MissionObjective) string {
	descriptions := map[string]string{
		"exploitation":      "Exploit target servers to gain access and control.",
		"data_extraction":    "Extract valuable data from compromised systems.",
		"tool_mastery":       "Demonstrate mastery of your hacking tools.",
		"network_exploration": "Explore and map local network infrastructure.",
		"resource_gathering": "Mine cryptocurrency from compromised servers.",
		"stealth":            "Cover your tracks and remain undetected.",
	}

	desc, ok := descriptions[missionType]
	if !ok {
		desc = "Complete the objectives to earn rewards."
	}

	if len(objectives) > 0 {
		desc += " " + objectives[0].Description
	}

	return desc
}
