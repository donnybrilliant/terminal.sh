package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CredentialType represents how the credential was obtained.
type CredentialType string

const (
	CredentialTypeCracked    CredentialType = "cracked"    // Password was cracked
	CredentialTypeSniffed    CredentialType = "sniffed"    // Password was sniffed from network
	CredentialTypeExtracted  CredentialType = "extracted"  // Found in files/database
	CredentialTypePhished    CredentialType = "phished"    // Obtained via phishing
)

// DiscoveredCredential represents a credential (username/password) that a user has discovered.
type DiscoveredCredential struct {
	ID            uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	UserID        uuid.UUID      `gorm:"type:text;not null;index" json:"user_id"`
	ServerPath    string         `gorm:"not null;index" json:"server_path"`  // Server where credential is valid
	ServiceName   string         `gorm:"not null" json:"service_name"`       // Service (ssh, telnet, ftp, etc.)
	Username      string         `gorm:"not null" json:"username"`           // Discovered username
	Password      string         `gorm:"not null" json:"password"`           // Discovered password
	Role          string         `json:"role"`                               // Role name (e.g., "admin", "user")
	RoleType      RoleType       `json:"role_type"`                          // Privilege level (root, admin, user, guest)
	HomeDir       string         `json:"home_dir"`                           // Home directory for this user
	Type          CredentialType `gorm:"not null" json:"type"`               // How it was obtained
	ToolUsed      string         `json:"tool_used"`                          // Tool that discovered it
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// IsRoot returns true if this credential grants root access.
func (c *DiscoveredCredential) IsRoot() bool {
	return c.RoleType == RoleTypeRoot || c.Username == "root"
}

// GetPromptChar returns the shell prompt character for this credential.
func (c *DiscoveredCredential) GetPromptChar() string {
	if c.IsRoot() {
		return "#"
	}
	return "$"
}

// BeforeCreate generates a UUID for the credential if one doesn't exist.
func (c *DiscoveredCredential) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// DiscoveredUser represents a username discovered on a server (without password).
type DiscoveredUser struct {
	ID          uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID      uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`    // Player who discovered it
	ServerPath  string    `gorm:"not null;index" json:"server_path"`          // Server where user exists
	Username    string    `gorm:"not null" json:"username"`                   // Discovered username
	Role        string    `json:"role"`                                       // Role if known
	ServiceName string    `json:"service_name"`                               // Service where user was found
	ToolUsed    string    `json:"tool_used"`                                  // Tool that discovered it
	CreatedAt   time.Time `json:"created_at"`
}

// BeforeCreate generates a UUID for the discovered user if one doesn't exist.
func (u *DiscoveredUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BackdoorAccess represents direct shell access obtained through RCE exploits.
// This is different from credentials - it's a persistent backdoor.
type BackdoorAccess struct {
	ID          uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID      uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	ServerPath  string    `gorm:"not null;index" json:"server_path"`
	ServiceName string    `gorm:"not null" json:"service_name"`       // Service exploited
	ExploitType string    `gorm:"not null" json:"exploit_type"`       // Type of exploit (rce, buffer_overflow)
	ToolUsed    string    `json:"tool_used"`                          // Tool that created the backdoor
	AccessLevel string    `gorm:"default:'root'" json:"access_level"` // root, admin, user
	CreatedAt   time.Time `json:"created_at"`
}

// BeforeCreate generates a UUID for the backdoor if one doesn't exist.
func (b *BackdoorAccess) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// IsRoot returns true if this backdoor grants root access.
func (b *BackdoorAccess) IsRoot() bool {
	return b.AccessLevel == "root" || b.AccessLevel == ""
}

// PrivilegeEscalation represents a privilege escalation achieved on a server.
// This allows a user to elevate from a lower role to a higher one.
type PrivilegeEscalation struct {
	ID           uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:text;not null;index" json:"user_id"`
	ServerPath   string    `gorm:"not null;index" json:"server_path"`
	FromRole     string    `gorm:"not null" json:"from_role"`      // Original role (e.g., "user")
	ToRole       string    `gorm:"not null" json:"to_role"`        // Escalated role (e.g., "root")
	Method       string    `gorm:"not null" json:"method"`         // How escalation was achieved
	ToolUsed     string    `json:"tool_used"`                      // Tool that achieved escalation
	Persistent   bool      `json:"persistent"`                     // Whether escalation persists (e.g., added to sudoers)
	CreatedAt    time.Time `json:"created_at"`
}

// BeforeCreate generates a UUID for the privilege escalation if one doesn't exist.
func (p *PrivilegeEscalation) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

