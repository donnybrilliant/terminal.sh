// Package services provides business logic services for the terminal.sh game.
// This file implements the ActionTracker service, which provides centralized tracking
// of player actions for mission objective validation and gameplay analytics.
package services

import (
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// ActionTracker provides centralized tracking of player actions
type ActionTracker struct {
	db             *database.Database
	missionService *MissionService
}

// NewActionTracker creates a new ActionTracker service
func NewActionTracker(db *database.Database) *ActionTracker {
	return &ActionTracker{
		db: db,
	}
}

// SetMissionService sets the mission service for active mission lookup
func (t *ActionTracker) SetMissionService(missionService *MissionService) {
	t.missionService = missionService
}

// TrackToolUse records a tool being used on a target
func (t *ActionTracker) TrackToolUse(userID uuid.UUID, toolName, targetServer, serviceName string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionToolUse,
		ToolName:     toolName,
		TargetServer: targetServer,
		ServiceName:  serviceName,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackServerExploit records a successful server exploit
func (t *ActionTracker) TrackServerExploit(userID uuid.UUID, toolName, serverPath, serviceName string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionServerExploit,
		ToolName:     toolName,
		TargetServer: serverPath,
		ServiceName:  serviceName,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackPrivilegeEscalation records a privilege escalation
func (t *ActionTracker) TrackPrivilegeEscalation(userID uuid.UUID, toolName, serverPath, fromRole, toRole string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionPrivilegeEscalate,
		ToolName:     toolName,
		TargetServer: serverPath,
		Details:      fromRole + " -> " + toRole,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackCredentialCrack records a credential being cracked
func (t *ActionTracker) TrackCredentialCrack(userID uuid.UUID, toolName, serverPath, serviceName string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionCredentialCrack,
		ToolName:     toolName,
		TargetServer: serverPath,
		ServiceName:  serviceName,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackDataExtraction records data being extracted
func (t *ActionTracker) TrackDataExtraction(userID uuid.UUID, toolName, serverPath string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionDataExtract,
		ToolName:     toolName,
		TargetServer: serverPath,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackBackdoorInstall records a backdoor being installed
func (t *ActionTracker) TrackBackdoorInstall(userID uuid.UUID, toolName, serverPath string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionBackdoorInstall,
		ToolName:     toolName,
		TargetServer: serverPath,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackServerConnect records connecting to a server
func (t *ActionTracker) TrackServerConnect(userID uuid.UUID, serverPath, serviceName string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionServerConnect,
		TargetServer: serverPath,
		ServiceName:  serviceName,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// TrackToolDownload records downloading a tool from a server
func (t *ActionTracker) TrackToolDownload(userID uuid.UUID, toolName, serverPath string) error {
	missionID := t.getActiveMissionID(userID)

	action := &models.TrackedAction{
		UserID:       userID,
		ActionType:   models.ActionToolDownload,
		ToolName:     toolName,
		TargetServer: serverPath,
		MissionID:    missionID,
		CreatedAt:    time.Now(),
	}

	return t.db.Create(action).Error
}

// getActiveMissionID returns the ID of the user's active (in_progress) mission
func (t *ActionTracker) getActiveMissionID(userID uuid.UUID) string {
	if t.missionService == nil {
		return ""
	}

	// Get user's in-progress missions
	var userMissions []models.UserMission
	if err := t.db.Where("user_id = ? AND status = ?", userID, "in_progress").
		Order("started_at DESC").Find(&userMissions).Error; err != nil {
		return ""
	}

	if len(userMissions) > 0 {
		return userMissions[0].MissionID
	}
	return ""
}

// HasCompletedObjective checks if a user has completed a specific objective type
// This is used by mission validation to verify objectives were actually done
func (t *ActionTracker) HasCompletedObjective(userID uuid.UUID, missionID string, objective models.MissionObjective) bool {
	var count int64

	query := t.db.Model(&models.TrackedAction{}).Where("user_id = ?", userID)

	// Check based on objective type
	switch objective.Type {
	case "use_tool":
		query = query.Where("action_type = ? AND tool_name = ?", models.ActionToolUse, objective.Tool)
		if objective.TargetServer != "" {
			query = query.Where("target_server = ? OR target_server LIKE ?", objective.TargetServer, "%."+objective.TargetServer)
		}

	case "exploit_server":
		query = query.Where("action_type = ?", models.ActionServerExploit)
		if objective.TargetServer != "" {
			query = query.Where("target_server = ? OR target_server LIKE ?", objective.TargetServer, "%."+objective.TargetServer)
		} else if objective.TargetType != "" {
			// For server type matching, we need to check service name
			query = query.Where("service_name = ?", objective.TargetType)
		}

	case "privilege_escalate":
		query = query.Where("action_type = ?", models.ActionPrivilegeEscalate)
		if objective.Tool != "" {
			query = query.Where("tool_name = ?", objective.Tool)
		}

	case "extract_data":
		query = query.Where("action_type = ?", models.ActionDataExtract)
		if objective.Tool != "" {
			query = query.Where("tool_name = ?", objective.Tool)
		}

	case "crack_credentials":
		query = query.Where("action_type = ?", models.ActionCredentialCrack)
		if objective.Tool != "" {
			query = query.Where("tool_name = ?", objective.Tool)
		}

	case "install_backdoor":
		query = query.Where("action_type = ?", models.ActionBackdoorInstall)
		if objective.Tool != "" {
			query = query.Where("tool_name = ?", objective.Tool)
		}

	case "connect_server":
		query = query.Where("action_type = ?", models.ActionServerConnect)
		if objective.TargetServer != "" {
			// Match exact (e.g. "home.pc") or path suffix (e.g. "x.localNetwork.home.pc") so objectives complete in any order
			query = query.Where("target_server = ? OR target_server LIKE ?", objective.TargetServer, "%."+objective.TargetServer)
		}

	case "download_tool":
		query = query.Where("action_type = ? AND tool_name = ?", models.ActionToolDownload, objective.Tool)
		if objective.TargetServer != "" {
			query = query.Where("target_server = ? OR target_server LIKE ?", objective.TargetServer, "%."+objective.TargetServer)
		}

	default:
		// Unknown objective type - allow if action exists with the tool
		if objective.Tool != "" {
			query = query.Where("tool_name = ?", objective.Tool)
		} else {
			return false // No validation possible, leave objective incomplete
		}
	}

	query.Count(&count)
	return count > 0
}

// GetActionsForMission returns all actions recorded for a specific mission
func (t *ActionTracker) GetActionsForMission(userID uuid.UUID, missionID string) ([]models.TrackedAction, error) {
	var actions []models.TrackedAction
	err := t.db.Where("user_id = ? AND mission_id = ?", userID, missionID).
		Order("created_at ASC").Find(&actions).Error
	return actions, err
}

// GetRecentActions returns recent actions for a user
func (t *ActionTracker) GetRecentActions(userID uuid.UUID, limit int) ([]models.TrackedAction, error) {
	var actions []models.TrackedAction
	err := t.db.Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&actions).Error
	return actions, err
}

// GetToolUsageCount returns how many times a tool has been used
func (t *ActionTracker) GetToolUsageCount(userID uuid.UUID, toolName string) int64 {
	var count int64
	t.db.Model(&models.TrackedAction{}).
		Where("user_id = ? AND action_type = ? AND tool_name = ?", userID, models.ActionToolUse, toolName).
		Count(&count)
	return count
}

// GetExploitedServerCount returns how many servers the user has exploited
func (t *ActionTracker) GetExploitedServerCount(userID uuid.UUID) int64 {
	var count int64
	t.db.Model(&models.TrackedAction{}).
		Where("user_id = ? AND action_type = ?", userID, models.ActionServerExploit).
		Distinct("target_server").Count(&count)
	return count
}

// CleanupOldActions removes actions older than the specified duration
func (t *ActionTracker) CleanupOldActions(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return t.db.Where("created_at < ?", cutoff).Delete(&models.TrackedAction{}).Error
}
