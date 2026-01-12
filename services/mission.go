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
	db       *database.Database
	missions []models.Mission
	dataPath string
	rewardService *RewardService
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

// GetMissionByID returns a mission by its ID
func (s *MissionService) GetMissionByID(id string) (*models.Mission, error) {
	for _, mission := range s.missions {
		if mission.ID == id {
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
						Hint:        "First, exploit a server with HTTP services. Then use `phishing_kit <targetIP>` to create a phishing campaign. This tool generates realistic-looking emails and sites to gather credentials. You'll receive this tool as a reward when you complete this mission!",
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
						Hint:        "First, find a server with HTTP services and SQL injection vulnerabilities. Use `sql_injector <targetIP>` to exploit it. Then use `database_dumper <targetIP>` to extract all database contents. This tool dumps entire databases - perfect for data heists! You'll receive this tool as a reward.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   750,
					Crypto:       150.0,
					Tools:        []string{"database_dumper"},
					Patches:      []string{"sql_injector_stealth"},
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
						Hint:        "Find and exploit the audit server first. Then use `log_cleaner <targetIP>` to delete and clear all system logs. This removes evidence of your activities. You'll receive this tool as a reward!",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Use timestomper to modify file timestamps",
						Tool:        "timestomper",
						Hint:        "After clearing logs, use `timestomper <targetIP>` on the same audit server. This modifies file timestamps to make forensic analysis harder. Combined with log_cleaner, you'll be nearly untraceable! You'll receive this tool as a reward.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   500,
					Crypto:       100.0,
					Tools:        []string{"log_cleaner", "timestomper"},
					Patches:      []string{"stealth_patch_v1"},
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
