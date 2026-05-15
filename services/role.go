package services

import (
	"fmt"
	"strings"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// RoleService manages role-based permissions and privilege escalation.
type RoleService struct {
	db                *database.Database
	credentialService *CredentialService
}

// NewRoleService creates a new RoleService.
func NewRoleService(db *database.Database) *RoleService {
	return &RoleService{db: db}
}

// SetCredentialService sets the credential service for best-credential lookup.
func (s *RoleService) SetCredentialService(cs *CredentialService) {
	s.credentialService = cs
}

// --- Privilege Escalation ---

// RecordPrivilegeEscalation records a successful privilege escalation.
func (s *RoleService) RecordPrivilegeEscalation(userID uuid.UUID, serverPath, fromRole, toRole, method, toolUsed string, persistent bool) error {
	// Check if already escalated to this role
	var existing models.PrivilegeEscalation
	if err := s.db.Where("user_id = ? AND server_path = ? AND to_role = ?",
		userID, serverPath, toRole).First(&existing).Error; err == nil {
		// Already escalated
		return nil
	}

	escalation := &models.PrivilegeEscalation{
		UserID:     userID,
		ServerPath: serverPath,
		FromRole:   fromRole,
		ToRole:     toRole,
		Method:     method,
		ToolUsed:   toolUsed,
		Persistent: persistent,
	}
	return s.db.Create(escalation).Error
}

// HasEscalatedTo checks if user has escalated to a specific role on a server.
func (s *RoleService) HasEscalatedTo(userID uuid.UUID, serverPath, toRole string) bool {
	var count int64
	s.db.Model(&models.PrivilegeEscalation{}).
		Where("user_id = ? AND server_path = ? AND to_role = ?", userID, serverPath, toRole).
		Count(&count)
	return count > 0
}

// HasRootAccess checks if user has root access on a server (via escalation or backdoor).
func (s *RoleService) HasRootAccess(userID uuid.UUID, serverPath string) bool {
	// Check for privilege escalation to root
	if s.HasEscalatedTo(userID, serverPath, "root") {
		return true
	}

	// Check for root backdoor
	var backdoor models.BackdoorAccess
	if err := s.db.Where("user_id = ? AND server_path = ? AND access_level = ?",
		userID, serverPath, "root").First(&backdoor).Error; err == nil {
		return true
	}

	// Check for root credentials
	var cred models.DiscoveredCredential
	if err := s.db.Where("user_id = ? AND server_path = ? AND (username = ? OR role_type = ?)",
		userID, serverPath, "root", models.RoleTypeRoot).First(&cred).Error; err == nil {
		return true
	}

	return false
}

// GetEffectiveRole returns the highest privilege role user has on a server.
func (s *RoleService) GetEffectiveRole(userID uuid.UUID, serverPath string) string {
	// Check for root access first
	if s.HasRootAccess(userID, serverPath) {
		return "root"
	}

	// Check for admin escalation
	if s.HasEscalatedTo(userID, serverPath, "admin") {
		return "admin"
	}

	// Check for admin credentials
	var adminCred models.DiscoveredCredential
	if err := s.db.Where("user_id = ? AND server_path = ? AND role_type = ?",
		userID, serverPath, models.RoleTypeAdmin).First(&adminCred).Error; err == nil {
		return adminCred.Username
	}

	// Return first available credential
	var cred models.DiscoveredCredential
	if err := s.db.Where("user_id = ? AND server_path = ?",
		userID, serverPath).First(&cred).Error; err == nil {
		return cred.Username
	}

	return "user"
}

// GetPrivilegeEscalations returns all privilege escalations for a server.
func (s *RoleService) GetPrivilegeEscalations(userID uuid.UUID, serverPath string) ([]models.PrivilegeEscalation, error) {
	var escalations []models.PrivilegeEscalation
	err := s.db.Where("user_id = ? AND server_path = ?", userID, serverPath).Find(&escalations).Error
	return escalations, err
}

// GetAllPrivilegeEscalations returns all privilege escalations by a user.
func (s *RoleService) GetAllPrivilegeEscalations(userID uuid.UUID) ([]models.PrivilegeEscalation, error) {
	var escalations []models.PrivilegeEscalation
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&escalations).Error
	return escalations, err
}

// --- Permission Checking ---

// FilePermission represents permissions for a file/directory.
type FilePermission struct {
	Path       string
	Owner      string   // Username who owns this
	Group      string   // Group that owns this
	Mode       string   // e.g., "rwxr-xr-x"
	RequiresRoot bool   // Whether only root can access
}

// DefaultFilePermissions returns default file permissions for common paths.
func DefaultFilePermissions() map[string]FilePermission {
	return map[string]FilePermission{
		"/etc/passwd":       {Path: "/etc/passwd", Owner: "root", Group: "root", Mode: "rw-r--r--", RequiresRoot: false},
		"/etc/shadow":       {Path: "/etc/shadow", Owner: "root", Group: "shadow", Mode: "rw-r-----", RequiresRoot: true},
		"/etc/sudoers":      {Path: "/etc/sudoers", Owner: "root", Group: "root", Mode: "r--r-----", RequiresRoot: true},
		"/root":             {Path: "/root", Owner: "root", Group: "root", Mode: "rwx------", RequiresRoot: true},
		"/var/log/auth.log": {Path: "/var/log/auth.log", Owner: "root", Group: "adm", Mode: "rw-r-----", RequiresRoot: false},
		"/var/log/syslog":   {Path: "/var/log/syslog", Owner: "root", Group: "adm", Mode: "rw-r-----", RequiresRoot: false},
	}
}

// CanAccessPath checks if a role can access a given path.
func CanAccessPath(path string, role *models.Role, isWrite bool) bool {
	if role == nil {
		return false
	}

	// Root can access everything
	if role.IsRoot() {
		return true
	}

	// Check default permissions
	perms := DefaultFilePermissions()
	
	// Check exact path match
	if perm, ok := perms[path]; ok {
		if perm.RequiresRoot {
			return false
		}
		// Owner check
		if perm.Owner == role.Role {
			return true
		}
	}

	// Check if path is under a restricted directory
	restrictedPaths := []string{"/root", "/etc/shadow", "/etc/sudoers"}
	for _, restricted := range restrictedPaths {
		if strings.HasPrefix(path, restricted) {
			return false
		}
	}

	// Check if trying to write to system directories
	if isWrite {
		systemDirs := []string{"/etc", "/usr", "/bin", "/sbin", "/var"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(path, sysDir) {
				return role.CanWriteSystem()
			}
		}
	}

	// Default: allow access
	return true
}

// CanRunCommand checks if a role can run a specific command.
func CanRunCommand(command string, role *models.Role) (bool, string) {
	if role == nil {
		return false, "no role"
	}

	// Root can run everything
	if role.IsRoot() {
		return true, ""
	}

	// Commands that require root
	rootCommands := map[string]bool{
		"useradd":    true,
		"userdel":    true,
		"passwd":     true,
		"visudo":     true,
		"systemctl":  true,
		"service":    true,
		"mount":      true,
		"umount":     true,
		"fdisk":      true,
		"iptables":   true,
		"reboot":     true,
		"shutdown":   true,
		"init":       true,
	}

	// Extract base command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return true, ""
	}
	baseCmd := parts[0]

	// Check if it's a sudo command
	if baseCmd == "sudo" {
		if role.CanSudo {
			return true, ""
		}
		return false, "user is not in sudoers"
	}

	// Check root-only commands
	if rootCommands[baseCmd] {
		if role.CanSudo {
			return true, "requires sudo"
		}
		return false, "permission denied"
	}

	return true, ""
}

// --- Helper Functions ---

// GetRoleForConnection determines the role to use when connecting.
// Returns the role info based on credential or backdoor.
type ConnectionRole struct {
	Username    string
	RoleType    models.RoleType
	HomeDir     string
	IsRoot      bool
	PromptChar  string
	AccessMethod string // "credentials" or "backdoor"
}

func (s *RoleService) GetConnectionRole(userID uuid.UUID, serverPath, serviceName string, server *models.Server) *ConnectionRole {
	// Check for backdoor first (backdoors typically give root)
	var backdoor models.BackdoorAccess
	if err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ?",
		userID, serverPath, serviceName).First(&backdoor).Error; err == nil {
		role := &ConnectionRole{
			Username:     "root",
			RoleType:     models.RoleTypeRoot,
			HomeDir:      "/root",
			IsRoot:       true,
			PromptChar:   "#",
			AccessMethod: "backdoor",
		}
		if backdoor.AccessLevel != "root" && backdoor.AccessLevel != "" {
			role.Username = backdoor.AccessLevel
			role.IsRoot = false
			role.PromptChar = "$"
			role.RoleType = models.RoleTypeUser
			role.HomeDir = "/home/" + backdoor.AccessLevel
		}
		return role
	}

	// Check for credentials - use highest privilege (root > admin > user > guest)
	var cred *models.DiscoveredCredential
	if s.credentialService != nil {
		var err error
		cred, err = s.credentialService.GetBestCredentialForService(userID, serverPath, serviceName)
		if err != nil {
			cred = nil
		}
	}
	if cred == nil {
		var fallback models.DiscoveredCredential
		if err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ?",
			userID, serverPath, serviceName).First(&fallback).Error; err == nil {
			cred = &fallback
		}
	}
	if cred != nil {
		connRole := &ConnectionRole{
			Username:     cred.Username,
			RoleType:     cred.RoleType,
			HomeDir:      cred.HomeDir,
			IsRoot:       cred.IsRoot(),
			PromptChar:   cred.GetPromptChar(),
			AccessMethod: "credentials",
		}
		if connRole.HomeDir == "" {
			if connRole.IsRoot {
				connRole.HomeDir = "/root"
			} else {
				connRole.HomeDir = "/home/" + cred.Username
			}
		}
		return connRole
	}

	// Check for privilege escalation to root
	if s.HasRootAccess(userID, serverPath) {
		return &ConnectionRole{
			Username:     "root",
			RoleType:     models.RoleTypeRoot,
			HomeDir:      "/root",
			IsRoot:       true,
			PromptChar:   "#",
			AccessMethod: "escalation",
		}
	}

	// Default to generic user
	return &ConnectionRole{
		Username:     "user",
		RoleType:     models.RoleTypeUser,
		HomeDir:      "/home/user",
		IsRoot:       false,
		PromptChar:   "$",
		AccessMethod: "unknown",
	}
}

// DescribePrivilegeMethod returns a human-readable description of a privesc method.
func DescribePrivilegeMethod(method string) string {
	descriptions := map[string]string{
		"sudo_misconfiguration": "Exploited misconfigured sudo permissions",
		"suid_binary":           "Exploited SUID binary vulnerability",
		"kernel_exploit":        "Exploited kernel vulnerability",
		"cron_job":              "Exploited writable cron job",
		"writable_path":         "Exploited writable PATH directory",
		"docker_escape":         "Escaped from Docker container",
		"capability_abuse":      "Abused Linux capabilities",
	}
	if desc, ok := descriptions[method]; ok {
		return desc
	}
	return fmt.Sprintf("Exploited %s", method)
}
