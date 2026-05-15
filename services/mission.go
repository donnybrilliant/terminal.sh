package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	missionGenerator  *MissionGenerator  // Optional mission generator
	actionTracker     *ActionTracker     // Optional action tracker for objective validation
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

// HasCompletedMission checks if a user has completed a specific mission
func (s *MissionService) HasCompletedMission(userID uuid.UUID, missionID string) bool {
	userMission, err := s.GetUserMission(userID, missionID)
	if err != nil {
		return false
	}
	return userMission.Status == "completed"
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

	// Enforce required tools after any start-of-mission grants
	if len(mission.RequiredTools) > 0 {
		if s.rewardService == nil || s.rewardService.toolService == nil {
			return fmt.Errorf("tool service unavailable for requirement checks")
		}
		var missing []string
		for _, toolName := range mission.RequiredTools {
			if !s.rewardService.toolService.UserHasTool(userID, toolName) {
				missing = append(missing, toolName)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
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

// MissionCompletionResult contains info about an auto-completed mission (for UI display)
type MissionCompletionResult struct {
	Mission   *models.Mission
	UserMission *models.UserMission
}

// TryTriggerMission checks if a trigger fires and starts a matching story mission.
// Returns the mission that was started, or nil if none matched.
func (s *MissionService) TryTriggerMission(userID uuid.UUID, triggerType, triggerPath string) *models.Mission {
	for _, mission := range s.missions {
		if mission.Trigger == nil || mission.Trigger.Type != triggerType {
			continue
		}
		if triggerType == "cat_file" && mission.Trigger.Path != "" {
			// Match if path ends with trigger path (e.g., README.txt matches home/user/README.txt)
			if !strings.HasSuffix(triggerPath, mission.Trigger.Path) && triggerPath != mission.Trigger.Path {
				continue
			}
		}
		// Prerequisites met?
		if err := s.StartMission(userID, mission.ID); err != nil {
			continue
		}
		return &mission
	}
	return nil
}

// TryAutoComplete checks all in-progress missions and auto-completes any that have all objectives done.
// Returns completion result for the first completed mission (caller can display rewards).
func (s *MissionService) TryAutoComplete(userID uuid.UUID) *MissionCompletionResult {
	userMissions, err := s.GetUserMissions(userID)
	if err != nil {
		return nil
	}
	for _, um := range userMissions {
		if um.Status != "in_progress" {
			continue
		}
		mission, err := s.GetMissionByID(um.MissionID)
		if err != nil || len(mission.Objectives) == 0 {
			continue
		}
		if s.actionTracker == nil {
			continue
		}
		incomplete := s.GetIncompleteObjectives(userID, um.MissionID)
		if len(incomplete) > 0 {
			continue
		}
		// All objectives done - complete the mission
		if err := s.completeMissionInternal(userID, um.MissionID); err != nil {
			continue
		}
		// Refresh user mission for completion time
		completedUM, _ := s.GetUserMission(userID, um.MissionID)
		return &MissionCompletionResult{Mission: mission, UserMission: completedUM}
	}
	return nil
}

// completeMissionInternal performs the completion logic (grants rewards, updates status).
func (s *MissionService) completeMissionInternal(userID uuid.UUID, missionID string) error {
	userMission, err := s.GetUserMission(userID, missionID)
	if err != nil || userMission.Status == "completed" {
		return fmt.Errorf("mission not in progress")
	}
	mission, err := s.GetMissionByID(missionID)
	if err != nil {
		return err
	}
	if s.rewardService != nil {
		if err := s.rewardService.GrantRewards(userID, mission.Rewards); err != nil {
			return err
		}
	}
	now := time.Now()
	userMission.Status = "completed"
	userMission.Progress = 100
	userMission.CompletedAt = &now
	return s.db.Save(userMission).Error
}

// StopMission abandons an in-progress mission (mission board only).
func (s *MissionService) StopMission(userID uuid.UUID, missionID string) error {
	userMission, err := s.GetUserMission(userID, missionID)
	if err != nil {
		return fmt.Errorf("mission not found")
	}
	if userMission.Status == "completed" {
		return fmt.Errorf("mission already completed")
	}
	if userMission.Status != "in_progress" {
		return fmt.Errorf("mission not in progress")
	}
	// Don't allow stopping story missions (those with triggers)
	mission, err := s.GetMissionByID(missionID)
	if err == nil && mission.Trigger != nil {
		return fmt.Errorf("cannot abandon story mission")
	}
	return s.db.Delete(userMission).Error
}

// CompleteMission marks a mission as completed and grants rewards
// DEPRECATED: Use TryAutoComplete - missions now complete automatically.
// Kept for backward compatibility; validates objectives and calls completeMissionInternal.
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
	
	// Validate objectives if action tracker is available
	if s.actionTracker != nil && len(mission.Objectives) > 0 {
		incompleteObjectives := s.GetIncompleteObjectives(userID, missionID)
		if len(incompleteObjectives) > 0 {
			// Build error message with incomplete objectives
			var objDescriptions []string
			for _, obj := range incompleteObjectives {
				objDescriptions = append(objDescriptions, obj.Description)
			}
			return fmt.Errorf("incomplete objectives: %s", strings.Join(objDescriptions, "; "))
		}
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

// GetIncompleteObjectives returns objectives that haven't been completed yet
func (s *MissionService) GetIncompleteObjectives(userID uuid.UUID, missionID string) []models.MissionObjective {
	mission, err := s.GetMissionByID(missionID)
	if err != nil || s.actionTracker == nil {
		return nil
	}
	
	var incomplete []models.MissionObjective
	for _, obj := range mission.Objectives {
		if !s.actionTracker.HasCompletedObjective(userID, missionID, obj) {
			incomplete = append(incomplete, obj)
		}
	}
	return incomplete
}

// GetMissionProgress calculates the progress percentage for a mission based on completed objectives
func (s *MissionService) GetMissionProgress(userID uuid.UUID, missionID string) int {
	mission, err := s.GetMissionByID(missionID)
	if err != nil || len(mission.Objectives) == 0 {
		return 0
	}
	
	if s.actionTracker == nil {
		return 0
	}
	
	completed := 0
	for _, obj := range mission.Objectives {
		if s.actionTracker.HasCompletedObjective(userID, missionID, obj) {
			completed++
		}
	}
	
	return (completed * 100) / len(mission.Objectives)
}

// Story Arc Completion and Post-Story Hooks

// StoryArcProgress represents progress through a story arc
type StoryArcProgress struct {
	ArcID           string
	ArcName         string
	TotalMissions   int
	CompletedCount  int
	IsCompleted     bool
	PercentComplete int
}

// GetStoryArcProgress returns progress for all story arcs
func (s *MissionService) GetStoryArcProgress(userID uuid.UUID) []StoryArcProgress {
	// Group missions by arc
	arcMissions := make(map[string][]models.Mission)
	arcNames := make(map[string]string)
	
	for _, mission := range s.missions {
		arcMissions[mission.ArcID] = append(arcMissions[mission.ArcID], mission)
		arcNames[mission.ArcID] = mission.ArcName
	}
	
	// Get completed missions
	completedMissions := make(map[string]bool)
	userMissions, _ := s.GetUserMissions(userID)
	for _, um := range userMissions {
		if um.Status == "completed" {
			completedMissions[um.MissionID] = true
		}
	}
	
	// Calculate progress for each arc
	var progress []StoryArcProgress
	for arcID, missions := range arcMissions {
		completed := 0
		for _, m := range missions {
			if completedMissions[m.ID] {
				completed++
			}
		}
		
		arcProgress := StoryArcProgress{
			ArcID:           arcID,
			ArcName:         arcNames[arcID],
			TotalMissions:   len(missions),
			CompletedCount:  completed,
			IsCompleted:     completed == len(missions),
			PercentComplete: 0,
		}
		if len(missions) > 0 {
			arcProgress.PercentComplete = (completed * 100) / len(missions)
		}
		progress = append(progress, arcProgress)
	}
	
	return progress
}

// IsStoryComplete checks if all story arcs have been completed
func (s *MissionService) IsStoryComplete(userID uuid.UUID) bool {
	arcProgress := s.GetStoryArcProgress(userID)
	for _, arc := range arcProgress {
		if !arc.IsCompleted {
			return false
		}
	}
	return len(arcProgress) > 0 // Return false if no arcs exist
}

// GetCompletedArcCount returns the number of completed story arcs
func (s *MissionService) GetCompletedArcCount(userID uuid.UUID) int {
	count := 0
	for _, arc := range s.GetStoryArcProgress(userID) {
		if arc.IsCompleted {
			count++
		}
	}
	return count
}

// EndlessMode represents post-story endless gameplay state
type EndlessMode struct {
	IsUnlocked          bool
	CompletedArcs       int
	TotalArcs           int
	ProceduralMissions  int // Number of procedural missions completed
	HighestTierReached  int
	ServersExploited    int
}

// GetEndlessModeStatus returns the player's endless mode progression
func (s *MissionService) GetEndlessModeStatus(userID uuid.UUID) EndlessMode {
	arcProgress := s.GetStoryArcProgress(userID)
	
	completedArcs := 0
	for _, arc := range arcProgress {
		if arc.IsCompleted {
			completedArcs++
		}
	}
	
	// Count procedural missions completed
	var proceduralCount int64
	s.db.Model(&models.GeneratedMission{}).
		Joins("JOIN user_missions ON generated_missions.mission_id = user_missions.mission_id").
		Where("user_missions.user_id = ? AND user_missions.status = ?", userID, "completed").
		Count(&proceduralCount)
	
	// Get user level for tier calculation
	var user models.User
	highestTier := 1
	if err := s.db.Where("id = ?", userID).First(&user).Error; err == nil {
		highestTier = user.Level
		if highestTier > 10 {
			highestTier = 10
		}
	}
	
	// Count exploited servers
	var exploitedCount int64
	s.db.Model(&models.ExploitedServer{}).Where("user_id = ?", userID).Count(&exploitedCount)
	
	return EndlessMode{
		IsUnlocked:         completedArcs >= 1, // Unlock after completing at least one arc
		CompletedArcs:      completedArcs,
		TotalArcs:          len(arcProgress),
		ProceduralMissions: int(proceduralCount),
		HighestTierReached: highestTier,
		ServersExploited:   int(exploitedCount),
	}
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

// SetActionTracker sets the action tracker for objective validation
func (s *MissionService) SetActionTracker(tracker *ActionTracker) {
	s.actionTracker = tracker
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
						Type:        "use_tool",
						Description: "Capture WiFi traffic with packet_capture",
						Tool:        "packet_capture",
						TargetType:  "ssh",
						Hint:        "Scan the internet to find the coffee shop WiFi server, then run `packet_capture <targetIP>` to collect traffic for analysis.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Extract credentials with password_sniffer",
						Tool:        "password_sniffer",
						TargetType:  "ssh",
						Hint:        "Use `password_sniffer <targetIP>` on the same WiFi server to extract employee credentials from captured traffic. These credentials unlock access to internal systems.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   100,
					Crypto:       20.0,
					Tools:        []string{"packet_capture", "password_sniffer"},
					Achievements: []string{"wifi_warrior"},
				},
				Unlocks: []string{"corp_espionage_02"},
			},
			{
				ID:            "corp_espionage_02",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 2,
				Name:          "Legacy Access",
				Description:   "Break into legacy systems using password-based access",
				Prerequisites: []string{"corp_espionage_01"},
				RequiredTools: []string{"password_cracker"},
				RequiredLevel: 2,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Crack a Telnet account with password_cracker",
						Tool:        "password_cracker",
						TargetType:  "telnet",
						Hint:        "Find a legacy Telnet server (port 23). Run `password_cracker <targetIP>` to obtain credentials, then connect with `telnet <targetIP>`.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Crack an FTP account with password_cracker",
						Tool:        "password_cracker",
						TargetType:  "ftp",
						Hint:        "Find an FTP server (port 21). Use `password_cracker <targetIP>` to obtain credentials, then connect with `ftp <targetIP>`.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 150,
					Crypto:     30.0,
					Tools:      []string{"password_cracker"},
				},
				Unlocks: []string{"corp_espionage_03"},
			},
			{
				ID:            "corp_espionage_03",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 3,
				Name:          "Secure Shell Access",
				Description:   "Escalate from credentials to remote code execution",
				Prerequisites: []string{"corp_espionage_02"},
				RequiredTools: []string{"ssh_exploit", "user_enum"},
				RequiredLevel: 2,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Enumerate users on an SSH server",
						Tool:        "user_enum",
						TargetType:  "ssh",
						Hint:        "Use `user_enum <targetIP>` to identify usernames and roles on the target.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Exploit SSH for shell access",
						Tool:        "ssh_exploit",
						TargetType:  "ssh",
						Hint:        "Use `ssh_exploit <targetIP>` against SSH vulnerabilities to gain direct shell access.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 200,
					Crypto:     50.0,
					Tools:      []string{"ssh_exploit", "user_enum"},
				},
				Unlocks: []string{"corp_espionage_04"},
			},
			{
				ID:            "corp_espionage_04",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 4,
				Name:          "Privilege Escalation",
				Description:   "Escalate from user access to root using local vulnerabilities",
				Prerequisites: []string{"corp_espionage_03"},
				RequiredTools: []string{"privesc_scanner", "sudo_exploit", "kernel_exploit", "suid_finder"},
				RequiredLevel: 3,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Scan for local privilege escalation vectors",
						Tool:        "privesc_scanner",
						Hint:        "Connect to a compromised server first, then run `privesc_scanner` to list local privilege escalation options.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Exploit a sudo misconfiguration",
						Tool:        "sudo_exploit",
						Hint:        "After scanning, use `sudo_exploit` on a vulnerable server to escalate privileges.",
					},
					{
						ID:          3,
						Type:        "use_tool",
						Description: "Exploit a vulnerable SUID binary",
						Tool:        "suid_finder",
						Hint:        "Use `suid_finder` to locate and exploit a vulnerable SUID binary for root access.",
					},
					{
						ID:          4,
						Type:        "use_tool",
						Description: "Exploit a kernel vulnerability",
						Tool:        "kernel_exploit",
						Hint:        "Use `kernel_exploit` on a server with a vulnerable kernel to gain root access.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 300,
					Crypto:     75.0,
					Tools:      []string{"privesc_scanner", "sudo_exploit", "kernel_exploit", "suid_finder"},
				},
				Unlocks: []string{"corp_espionage_05"},
			},
			{
				ID:            "corp_espionage_05",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 5,
				Name:          "Phishing for Answers",
				Description:   "Create phishing campaign targeting corporate email",
				Prerequisites: []string{"corp_espionage_04"},
				RequiredTools: []string{"phishing_kit"},
				RequiredLevel: 4,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Use phishing_kit on target server",
						Tool:        "phishing_kit",
						TargetType:  "http",
						Hint:        "The `phishing_kit` tool is granted at mission start. Find a server with HTTP services and run `phishing_kit <targetIP>` to launch a campaign.",
					},
				},
				Rewards: models.MissionRewards{
					Experience:   300,
					Crypto:       50.0,
					Tools:        []string{"phishing_kit"},
					Achievements: []string{"social_engineer"},
				},
				Unlocks: []string{"corp_espionage_06"},
			},
			{
				ID:            "corp_espionage_06",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 6,
				Name:          "The Database Heist",
				Description:   "Steal sensitive corporate data",
				Prerequisites: []string{"corp_espionage_05"},
				RequiredTools: []string{"sql_injector", "database_dumper", "hash_cracker"},
				RequiredLevel: 5,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Exploit the database with sql_injector",
						Tool:        "sql_injector",
						TargetType:  "http",
						Hint:        "Find a server with HTTP services and SQL injection vulnerabilities, then run `sql_injector <targetIP>` to gain access.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Use database_dumper to extract data",
						Tool:        "database_dumper",
						TargetType:  "http",
						Hint:        "The `database_dumper` tool is granted at mission start. Run `database_dumper <targetIP>` after successful SQL injection.",
					},
					{
						ID:          3,
						Type:        "use_tool",
						Description: "Crack extracted hashes with hash_cracker",
						Tool:        "hash_cracker",
						TargetType:  "http",
						Hint:        "Use `hash_cracker <targetIP>` on the same target to crack dumped password hashes.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 750,
					Crypto:     150.0,
					Tools:      []string{"database_dumper", "hash_cracker"},
					ToolUpgrades: []models.ToolUpgradeReward{
						{ToolName: "sql_injector", UpgradeType: "exploit", Count: 2},
					},
					Achievements: []string{"data_thief"},
				},
				Unlocks: []string{"corp_espionage_07"},
			},
			{
				ID:            "corp_espionage_07",
				ArcID:         "corp_espionage",
				ArcName:       "Corporate Espionage",
				MissionNumber: 7,
				Name:          "Cover Your Tracks",
				Description:   "They're onto you. Cover your tracks before they trace your attacks back to you.",
				Prerequisites: []string{"corp_espionage_06"},
				RequiredTools: []string{"log_cleaner", "timestomper", "audit_disable"},
				RequiredLevel: 6,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Use log_cleaner on audit server",
						Tool:        "log_cleaner",
						Hint:        "The `log_cleaner` tool is granted at mission start. Find and exploit the audit server first, then run `log_cleaner <targetIP>` to delete system logs.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Use timestomper to modify file timestamps",
						Tool:        "timestomper",
						Hint:        "After clearing logs, run `timestomper <targetIP>` on the same server to modify file timestamps and reduce forensic traces.",
					},
					{
						ID:          3,
						Type:        "use_tool",
						Description: "Disable auditing with audit_disable",
						Tool:        "audit_disable",
						Hint:        "Finish by running `audit_disable <targetIP>` to prevent new logs from being created.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 500,
					Crypto:     100.0,
					Tools:      []string{"log_cleaner", "timestomper", "audit_disable"},
					ToolUpgrades: []models.ToolUpgradeReward{
						{ToolName: "sql_injector", UpgradeType: "cpu", Count: 1},
						{ToolName: "database_dumper", UpgradeType: "exploit", Count: 1},
					},
					Achievements: []string{"ghost_in_the_machine"},
				},
				Unlocks: []string{"academic_hacking_01"},
			},
			{
				ID:            "field_ops_01",
				ArcID:         "field_ops",
				ArcName:       "Field Operations",
				MissionNumber: 1,
				Name:          "Network Recon",
				Description:   "Map internal networks and decode traffic",
				Prerequisites: []string{"corp_espionage_07"},
				RequiredTools: []string{"lan_sniffer", "packet_decoder", "log_analyzer"},
				RequiredLevel: 6,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Identify nearby hosts with lan_sniffer",
						Tool:        "lan_sniffer",
						TargetType:  "ssh",
						Hint:        "Use `lan_sniffer <targetIP>` to reveal nearby systems and internal routes.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Decode captured traffic with packet_decoder",
						Tool:        "packet_decoder",
						TargetType:  "ssh",
						Hint:        "Run `packet_decoder <targetIP>` to analyze captured network traffic.",
					},
					{
						ID:          3,
						Type:        "use_tool",
						Description: "Extract intelligence from logs with log_analyzer",
						Tool:        "log_analyzer",
						TargetType:  "ssh",
						Hint:        "Use `log_analyzer <targetIP>` to find admin access patterns and weak points.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 200,
					Crypto:     40.0,
					Tools:      []string{"lan_sniffer", "packet_decoder", "log_analyzer"},
				},
				Unlocks: []string{"field_ops_02"},
			},
			{
				ID:            "field_ops_02",
				ArcID:         "field_ops",
				ArcName:       "Field Operations",
				MissionNumber: 2,
				Name:          "Persistent Access",
				Description:   "Install a backdoor for long-term control",
				Prerequisites: []string{"field_ops_01"},
				RequiredTools: []string{"rootkit"},
				RequiredLevel: 7,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Install a rootkit on a compromised server",
						Tool:        "rootkit",
						TargetType:  "ssh",
						Hint:        "Exploit a server first, then run `rootkit <targetIP>` to establish persistent access.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 250,
					Crypto:     60.0,
					Tools:      []string{"rootkit"},
				},
				Unlocks: []string{"field_ops_03"},
			},
			{
				ID:            "field_ops_03",
				ArcID:         "field_ops",
				ArcName:       "Field Operations",
				MissionNumber: 3,
				Name:          "Exploit at Scale",
				Description:   "Use multi-vulnerability tools for faster breaches",
				Prerequisites: []string{"field_ops_02"},
				RequiredTools: []string{"exploit_kit", "advanced_exploit_kit", "xss_exploit"},
				RequiredLevel: 8,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Exploit multiple services with exploit_kit",
						Tool:        "exploit_kit",
						Hint:        "Run `exploit_kit <targetIP>` to attack multiple vulnerable services at once.",
					},
					{
						ID:          2,
						Type:        "use_tool",
						Description: "Exploit an HTTP service with xss_exploit",
						Tool:        "xss_exploit",
						TargetType:  "http",
						Hint:        "Find a web server with XSS and run `xss_exploit <targetIP>`.",
					},
					{
						ID:          3,
						Type:        "use_tool",
						Description: "Run advanced_exploit_kit for high-level targets",
						Tool:        "advanced_exploit_kit",
						Hint:        "Use `advanced_exploit_kit <targetIP>` against tougher targets to chain vulnerabilities.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 350,
					Crypto:     100.0,
					Tools:      []string{"exploit_kit", "advanced_exploit_kit", "xss_exploit"},
				},
				Unlocks: []string{"field_ops_04"},
			},
			{
				ID:            "field_ops_04",
				ArcID:         "field_ops",
				ArcName:       "Field Operations",
				MissionNumber: 4,
				Name:          "Resource Hijack",
				Description:   "Use compromised systems to mine cryptocurrency",
				Prerequisites: []string{"field_ops_03"},
				RequiredTools: []string{"crypto_miner"},
				RequiredLevel: 9,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Start a crypto miner on a compromised server",
						Tool:        "crypto_miner",
						Hint:        "Exploit a server with enough resources, then run `crypto_miner <targetIP>` to start mining.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 200,
					Crypto:     80.0,
					Tools:      []string{"crypto_miner"},
				},
				Unlocks: []string{"field_ops_05"},
			},
			{
				ID:            "field_ops_05",
				ArcID:         "field_ops",
				ArcName:       "Field Operations",
				MissionNumber: 5,
				Name:          "Backup Erasure",
				Description:   "Destroy backups to prevent recovery",
				Prerequisites: []string{"field_ops_04"},
				RequiredTools: []string{"backup_destroyer"},
				RequiredLevel: 9,
				Objectives: []models.MissionObjective{
					{
						ID:          1,
						Type:        "use_tool",
						Description: "Destroy backups on a compromised server",
						Tool:        "backup_destroyer",
						Hint:        "Exploit the target first, then run `backup_destroyer <targetIP>` to remove backup files.",
					},
				},
				Rewards: models.MissionRewards{
					Experience: 220,
					Crypto:     90.0,
					Tools:      []string{"backup_destroyer"},
				},
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
