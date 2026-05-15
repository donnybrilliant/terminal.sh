package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServerResources represents server computational resources.
type ServerResources struct {
	CPU      int     `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// ServerWallet represents a server's currency balances.
type ServerWallet struct {
	Crypto float64 `json:"crypto"`
	Data   float64 `json:"data"`
}

// Vulnerability represents a service vulnerability type and level.
type Vulnerability struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// Service represents a network service running on a server.
type Service struct {
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Port              int             `json:"port"`
	Vulnerable        bool            `json:"vulnerable"`
	Level             int             `json:"level"`
	Vulnerabilities   []Vulnerability `json:"vulnerabilities"`
	GrantsShellAccess bool            `json:"grants_shell_access"` // Whether exploiting this service grants shell access
	RequiresAuth      *bool           `json:"requires_auth,omitempty"` // If false, no credentials needed (e.g., your own PC)
}

// DefaultShellAccessServices returns service names that grant shell access by default.
// Used for backward compatibility when GrantsShellAccess is not explicitly set.
func DefaultShellAccessServices() map[string]bool {
	return map[string]bool{
		"ssh":    true,
		"telnet": true,
		"rdp":    true,
		"vnc":    true,
		// FTP only grants shell access if it has RCE vulnerability (checked separately)
	}
}

// ServiceGrantsShellAccess checks if a service grants shell access.
// Uses explicit GrantsShellAccess field if set, otherwise uses defaults.
func (s *Service) ServiceGrantsShellAccess() bool {
	// If explicitly set, use that value
	if s.GrantsShellAccess {
		return true
	}
	// Check default shell access services
	defaults := DefaultShellAccessServices()
	if defaults[s.Name] {
		return true
	}
	// FTP grants shell access only if it has RCE vulnerability
	if s.Name == "ftp" {
		for _, vuln := range s.Vulnerabilities {
			if vuln.Type == "remote_code_execution" {
				return true
			}
		}
	}
	return false
}

// RoleType represents the privilege level of a role.
type RoleType string

const (
	RoleTypeRoot  RoleType = "root"  // Full system access, UID 0
	RoleTypeAdmin RoleType = "admin" // Administrative access, can sudo
	RoleTypeUser  RoleType = "user"  // Regular user, limited access
	RoleTypeGuest RoleType = "guest" // Minimal access, read-only mostly
)

// Role represents a server role or user account on a server.
type Role struct {
	Role        string   `json:"role"`          // Username (e.g., "admin", "john", "backup")
	Level       int      `json:"level"`         // Security level (affects cracking difficulty)
	Type        RoleType `json:"type"`          // Privilege type (root, admin, user, guest)
	HomeDir     string   `json:"home_dir"`      // Home directory (e.g., "/root", "/home/john")
	Shell       string   `json:"shell"`         // Shell path (e.g., "/bin/bash", "/bin/sh")
	CanSudo     bool     `json:"can_sudo"`      // Whether this role can use sudo
	SudoNoPass  bool     `json:"sudo_no_pass"`  // Whether sudo requires password
	Groups      []string `json:"groups"`        // Groups this role belongs to
}

// GetRoleType returns the role type, defaulting based on role name if not set.
func (r *Role) GetRoleType() RoleType {
	if r.Type != "" {
		return r.Type
	}
	// Default based on role name
	switch r.Role {
	case "root":
		return RoleTypeRoot
	case "admin", "administrator", "sysadmin":
		return RoleTypeAdmin
	case "guest", "anonymous":
		return RoleTypeGuest
	default:
		return RoleTypeUser
	}
}

// IsRoot returns true if this role has root privileges.
func (r *Role) IsRoot() bool {
	return r.GetRoleType() == RoleTypeRoot
}

// CanWriteSystem returns true if this role can write to system directories.
func (r *Role) CanWriteSystem() bool {
	t := r.GetRoleType()
	return t == RoleTypeRoot || (t == RoleTypeAdmin && r.CanSudo)
}

// GetHomeDir returns the home directory, defaulting based on role.
func (r *Role) GetHomeDir() string {
	if r.HomeDir != "" {
		return r.HomeDir
	}
	if r.Role == "root" {
		return "/root"
	}
	return "/home/" + r.Role
}

// GetPromptChar returns the shell prompt character (# for root, $ for others).
func (r *Role) GetPromptChar() string {
	if r.IsRoot() {
		return "#"
	}
	return "$"
}

// Server represents a game server that can be scanned, exploited, and accessed.
type Server struct {
	ID                   uuid.UUID              `gorm:"type:text;primary_key" json:"id"`
	IP                   string                 `gorm:"uniqueIndex;not null" json:"ip"`
	LocalIP              string                 `gorm:"not null" json:"local_ip"`
	SecurityLevel        int                    `gorm:"default:100" json:"security_level"`
	Resources            ServerResources        `gorm:"type:text;serializer:json" json:"resources"`
	Wallet               ServerWallet           `gorm:"type:text;serializer:json" json:"wallet"`
	Tools                []string               `gorm:"type:text;serializer:json" json:"tools"`
	ConnectedIPs         []string               `gorm:"type:text;serializer:json" json:"connected_ips"`
	Services             []Service              `gorm:"type:text;serializer:json" json:"services"`
	Roles                []Role                 `gorm:"type:text;serializer:json" json:"roles"`
	LocalVulnerabilities []LocalVulnerability   `gorm:"type:text;serializer:json" json:"local_vulnerabilities"` // Privilege escalation vulns
	FileSystem           map[string]interface{} `gorm:"type:text;serializer:json" json:"file_system"`
	LocalNetwork         map[string]interface{} `gorm:"type:text;serializer:json" json:"local_network"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// LocalVulnerability represents a local privilege escalation vulnerability.
type LocalVulnerability struct {
	Type        string `json:"type"`         // sudo_misconfiguration, suid_binary, kernel_exploit, cron_job, writable_path
	Level       int    `json:"level"`        // Difficulty level to exploit
	Description string `json:"description"`  // Human-readable description
	Target      string `json:"target"`       // What to exploit (e.g., "/usr/bin/vim", "CVE-2021-4034")
	GrantsRoot  bool   `json:"grants_root"`  // Whether successful exploit grants root
}

// GetRoleByUsername finds a role by username.
func (s *Server) GetRoleByUsername(username string) *Role {
	for i := range s.Roles {
		if s.Roles[i].Role == username {
			return &s.Roles[i]
		}
	}
	return nil
}

// HasLocalVulnerabilities returns true if server has any local privesc vulnerabilities.
func (s *Server) HasLocalVulnerabilities() bool {
	return len(s.LocalVulnerabilities) > 0
}

// BeforeCreate is a GORM hook that generates a UUID for the server if one doesn't exist.
func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

