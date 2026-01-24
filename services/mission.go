package services

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// MissionService handles mission-related operations
type MissionService struct {
	db                *database.Database
	missions          []models.Mission
	dataPath          string
	rewardService     *RewardService
	missionGenerator *MissionGenerator // Optional mission generator
}

// NewMissionService creates a new MissionService and loads missions from JSON
func NewMissionService(db *database.Database, dataPath string, rewardService *RewardService) (*MissionService, error) {
	service := &MissionService{
		db:            db,
		dataPath:      dataPath,
		rewardService: rewardService,
	}
	
	if err := service.LoadMissions(); err != nil {
		return nil, fmt.Errorf("failed to load missions: %w", err)
	}
	
	return service, nil
}

// LoadMissions loads missions from JSON file
func (s *MissionService) LoadMissions() error {
	path := s.dataPath
	if path == "" {
		path = "data/seed/missions.json"
	}
	
	// Check if file exists, if not create default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := s.createDefaultMissions(path); err != nil {
			return fmt.Errorf("failed to create default missions: %w", err)
		}
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read mission file: %w", err)
	}
	
	var missionData models.MissionData
	if err := json.Unmarshal(data, &missionData); err != nil {
		return fmt.Errorf("failed to parse mission file: %w", err)
	}
	
	s.missions = missionData.Missions
	return nil
}

// GetAllMissions returns all available missions
func (s *MissionService) GetAllMissions() []models.Mission {
	return s.missions
}

// GetMissionByID returns a mission by its ID (checks both static and procedurally generated missions)
func (s *MissionService) GetMissionByID(id string) (*models.Mission, error) {
	// Check static missions first
	for _, mission := range s.missions {
		if mission.ID == id {
			return &mission, nil
		}
	}
	
	// Check procedurally generated missions
	var generatedMission models.GeneratedMission
	if err := s.db.Where("mission_id = ?", id).First(&generatedMission).Error; err == nil {
		var mission models.Mission
		if err := json.Unmarshal([]byte(generatedMission.MissionData), &mission); err == nil {
			return &mission, nil
		}
	}
	
	return nil, fmt.Errorf("mission not found: %s", id)
}

// GetUserMissions retrieves all missions for a user
func (s *MissionService) GetUserMissions(userID uuid.UUID) ([]models.UserMission, error) {
	var userMissions []models.UserMission
	if err := s.db.Where("user_id = ?", userID).Find(&userMissions).Error; err != nil {
		return nil, err
	}
	return userMissions, nil
}

// GetUserMission retrieves a specific user mission
func (s *MissionService) GetUserMission(userID uuid.UUID, missionID string) (*models.UserMission, error) {
	var userMission models.UserMission
	if err := s.db.Where("user_id = ? AND mission_id = ?", userID, missionID).First(&userMission).Error; err != nil {
		return nil, err
	}
	return &userMission, nil
}

// StartMission starts a mission for a user
func (s *MissionService) StartMission(userID uuid.UUID, missionID string) error {
	// Check if mission exists
	mission, err := s.GetMissionByID(missionID)
	if err != nil {
		return err
	}
	
	// Check prerequisites
	userMissions, err := s.GetUserMissions(userID)
	if err != nil {
		return err
	}
	
	completedMissionIDs := make(map[string]bool)
	for _, um := range userMissions {
		if um.Status == "completed" {
			completedMissionIDs[um.MissionID] = true
		}
	}
	
	for _, prereq := range mission.Prerequisites {
		if !completedMissionIDs[prereq] {
			return fmt.Errorf("prerequisite mission %s not completed", prereq)
		}
	}
	
	// Check if already started/completed
	existing, _ := s.GetUserMission(userID, missionID)
	if existing != nil {
		if existing.Status == "completed" {
			return fmt.Errorf("mission already completed")
		}
		// Already in progress, return success
		return nil
	}
	
	// Grant tools at mission start if mission requires using those tools
	// This allows players to complete objectives that require tools they don't have yet
	if s.rewardService != nil && len(mission.Rewards.Tools) > 0 {
		// Check if any objective requires a tool that's in the rewards
		needsToolAtStart := false
		for _, obj := range mission.Objectives {
			if obj.Type == "use_tool" && obj.Tool != "" {
				for _, rewardTool := range mission.Rewards.Tools {
					if obj.Tool == rewardTool {
						needsToolAtStart = true
						break
					}
				}
			}
			if needsToolAtStart {
				break
			}
		}
		
		// Grant tools early if needed for objectives
		if needsToolAtStart {
			// Create a temporary rewards struct with just the tools
			toolRewards := models.MissionRewards{
				Tools: mission.Rewards.Tools,
			}
			if err := s.rewardService.GrantRewards(userID, toolRewards); err != nil {
				// Log error but continue - mission can still start
				// Tools will be granted again on completion (which is idempotent)
			}
		}
	}
	
	// Create new user mission
	userMission := &models.UserMission{
		UserID:    userID,
		MissionID: missionID,
		Status:    "in_progress",
		Progress:  0,
		StartedAt: time.Now(),
	}
	
	if err := s.db.Create(userMission).Error; err != nil {
		return fmt.Errorf("failed to start mission: %w", err)
	}
	
	return nil
}

// CompleteMission marks a mission as completed and grants rewards
func (s *MissionService) CompleteMission(userID uuid.UUID, missionID string) error {
	// Get user mission
	userMission, err := s.GetUserMission(userID, missionID)
	if err != nil {
		return fmt.Errorf("mission not started: %w", err)
	}
	
	if userMission.Status == "completed" {
		return fmt.Errorf("mission already completed")
	}
	
	// Get mission definition
	mission, err := s.GetMissionByID(missionID)
	if err != nil {
		return err
	}
	
	// Grant rewards
	if s.rewardService != nil {
		if err := s.rewardService.GrantRewards(userID, mission.Rewards); err != nil {
			return fmt.Errorf("failed to grant rewards: %w", err)
		}
	}
	
	// Update user mission
	now := time.Now()
	userMission.Status = "completed"
	userMission.Progress = 100
	userMission.CompletedAt = &now
	
	if err := s.db.Save(userMission).Error; err != nil {
		return fmt.Errorf("failed to complete mission: %w", err)
	}
	
	return nil
}

// UpdateMissionProgress updates the progress of a mission
func (s *MissionService) UpdateMissionProgress(userID uuid.UUID, missionID string, progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	
	userMission, err := s.GetUserMission(userID, missionID)
	if err != nil {
		return err
	}
	
	userMission.Progress = progress
	if err := s.db.Save(userMission).Error; err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}
	
	return nil
}

// SetMissionGenerator sets the mission generator for this service
func (s *MissionService) SetMissionGenerator(generator *MissionGenerator) {
	s.missionGenerator = generator
}

// GetAvailableMissions returns missions available to a user (prerequisites met, level met)
func (s *MissionService) GetAvailableMissions(userID uuid.UUID, userLevel int) []models.Mission {
	userMissions, _ := s.GetUserMissions(userID)
	completedMissionIDs := make(map[string]bool)
	for _, um := range userMissions {
		if um.Status == "completed" {
			completedMissionIDs[um.MissionID] = true
		}
	}
	
	var available []models.Mission
	for _, mission := range s.missions {
		// Check level requirement
		if mission.RequiredLevel > userLevel {
			continue
		}
		
		// Check prerequisites
		prereqsMet := true
		for _, prereq := range mission.Prerequisites {
			if !completedMissionIDs[prereq] {
				prereqsMet = false
				break
			}
		}
		
		if prereqsMet {
			available = append(available, mission)
		}
	}
	
	// If we have less than 3 available missions and generator is available, generate some
	if len(available) < 3 && s.missionGenerator != nil {
		// Check if we've already generated missions for this user recently
		var existingGeneratedMissions []models.GeneratedMission
		s.db.Where("user_id = ?", userID).Find(&existingGeneratedMissions)
		
		// Generate missions if we don't have enough
		needed := 3 - len(available)
		if len(existingGeneratedMissions) < needed {
			for i := 0; i < needed; i++ {
				generatedMission, err := s.missionGenerator.GenerateMission(userID)
				if err == nil {
					// Add to available missions if level requirement is met
					if generatedMission.RequiredLevel <= userLevel {
						available = append(available, *generatedMission)
					}
				}
			}
		} else {
			// Load existing procedurally generated missions
			for _, generatedMissionRecord := range existingGeneratedMissions {
				var generatedMission models.Mission
				if err := json.Unmarshal([]byte(generatedMissionRecord.MissionData), &generatedMission); err == nil {
					if generatedMission.RequiredLevel <= userLevel {
						// Check if not already completed
						if !completedMissionIDs[generatedMission.ID] {
							available = append(available, generatedMission)
						}
					}
				}
			}
		}
	}
	
	return available
}

// createDefaultMissions creates a default missions file
func (s *MissionService) createDefaultMissions(path string) error {
	defaultMissions := models.MissionData{
		Missions: []models.Mission{
			{
				ID:            "corp_espionage_01",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 1,
				Name:          "The Coffee Shop WiFi",
				Description:   "Hack public WiFi to find employee credentials",
				Prerequisites: []string{},
				RequiredTools: []string{"packet_capture", "password_sniffer"},
				RequiredLevel: 1,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "exploit_server",
						Description: "Exploit the coffee shop WiFi server",
						Hint:        "First, scan the internet with `scan` to find the coffee shop server. Then use `packet_capture` to capture network traffic, and `password_sniffer` to extract credentials from the captured packets.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   100,
					Crypto:       20.0,
					Achievements: []string{"wifi_warrior"},
				},
				Unlocks: []string{"corp_espionage_02"},
			},
			{
				ID:            "corp_espionage_02",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 2,
				Name:          "Phishing for Answers",
				Description:   "Create phishing campaign targeting corporate email",
				Prerequisites: []string{"corp_espionage_01"},
				RequiredTools: []string{},
				RequiredLevel: 2,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Use phishing_kit on target server",
						Tool:        "phishing_kit",
						Hint:        "The `phishing_kit` tool will be granted to you when you start this mission. First, scan for servers and exploit one with HTTP services (look for port 80 in scan results). Then use `phishing_kit <targetIP>` to create a phishing campaign. This tool generates realistic-looking emails and sites to gather credentials.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   300,
					Crypto:       50.0,
					Tools:        []string{"phishing_kit"},
					Achievements: []string{"social_engineer"},
				},
				Unlocks: []string{"corp_espionage_03"},
			},
			{
				ID:            "corp_espionage_03",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 3,
				Name:          "The Database Heist",
				Description:   "Steal sensitive corporate data",
				Prerequisites: []string{"corp_espionage_02"},
				RequiredTools: []string{"sql_injector"},
				RequiredLevel: 3,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Use database_dumper to extract data",
						Tool:        "database_dumper",
						Hint:        "The `database_dumper` tool will be granted to you when you start this mission. First, scan for servers and find one with HTTP services and SQL injection vulnerabilities. Use `sql_injector <targetIP>` to exploit it first (you should have sql_injector from the repo). Then use `database_dumper <targetIP>` to extract all database contents. This tool dumps entire databases - perfect for data heists!",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 750,
					Crypto:     150.0,
					Tools:      []string{"database_dumper"},
					ToolUpgrades: []models.ToolUpgradeReward{
						{ToolName: "sql_injector", UpgradeType: "exploit", Count: 2},
					},
					Achievements: []string{"data_thief"},
				},
				Unlocks: []string{"corp_espionage_04"},
			},
			{
				ID:            "corp_espionage_04",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 4,
				Name:          "Cover Your Tracks",
				Description:   "They're onto you. Cover your tracks before they trace your attacks back to you.",
				Prerequisites: []string{"corp_espionage_03"},
				RequiredTools: []string{"sql_injector", "database_dumper"},
				RequiredLevel: 5,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Use log_cleaner on audit server",
						Tool:        "log_cleaner",
						Hint:        "The `log_cleaner` and `timestomper` tools will be granted to you when you start this mission. First, scan for servers and find the audit server (look for servers with 'audit' in scan results, or any server you've previously exploited). Then use `log_cleaner <targetIP>` to delete and clear all system logs. This removes evidence of your activities.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Use timestomper to modify file timestamps",
						Tool:        "timestomper",
						Hint:        "After clearing logs with log_cleaner, use `timestomper <targetIP>` on the same audit server. This modifies file timestamps to make forensic analysis harder. Combined with log_cleaner, you'll be nearly untraceable!",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 500,
					Crypto:     100.0,
					Tools:      []string{"log_cleaner", "timestomper"},
					ToolUpgrades: []models.ToolUpgradeReward{
						{ToolName: "sql_injector", UpgradeType: "cpu", Count: 1},
						{ToolName: "database_dumper", UpgradeType: "exploit", Count: 1},
					},
					Achievements: []string{"ghost_in_the_machine"},
				},
				Unlocks: []string{"academic_hacking_01"},
			},
		},
	}
	
	data, err := json.MarshalIndent(defaultMissions, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	if err := os.MkdirAll("data/seed", 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}
