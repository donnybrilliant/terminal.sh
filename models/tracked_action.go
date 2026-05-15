package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActionType represents the type of tracked action
type ActionType string

const (
	ActionToolUse           ActionType = "tool_use"
	ActionServerExploit     ActionType = "server_exploit"
	ActionPrivilegeEscalate ActionType = "privilege_escalate"
	ActionDataExtract       ActionType = "data_extract"
	ActionCredentialCrack   ActionType = "credential_crack"
	ActionServerScan        ActionType = "server_scan"
	ActionFileAccess        ActionType = "file_access"
	ActionMiningStart       ActionType = "mining_start"
	ActionBackdoorInstall   ActionType = "backdoor_install"
	ActionServerConnect     ActionType = "server_connect"  // Connecting to a server
	ActionToolDownload      ActionType = "tool_download"   // Downloading a tool from a server
)

// TrackedAction represents a recorded player action for mission validation and analytics
type TrackedAction struct {
	ID           uuid.UUID  `gorm:"type:text;primary_key" json:"id"`
	UserID       uuid.UUID  `gorm:"type:text;not null;index" json:"user_id"`
	ActionType   ActionType `gorm:"not null;index" json:"action_type"`
	ToolName     string     `json:"tool_name,omitempty"`
	TargetServer string     `json:"target_server,omitempty"` // Server IP or path
	ServiceName  string     `json:"service_name,omitempty"`
	Details      string     `gorm:"type:text" json:"details,omitempty"` // JSON for additional context
	MissionID    string     `json:"mission_id,omitempty"`               // Active mission at time of action
	CreatedAt    time.Time  `gorm:"index" json:"created_at"`
}

// BeforeCreate generates a UUID if not set
func (a *TrackedAction) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
