package services

import (
	"fmt"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// CredentialService manages discovered credentials, users, and backdoor access.
type CredentialService struct {
	db *database.Database
}

// NewCredentialService creates a new CredentialService.
func NewCredentialService(db *database.Database) *CredentialService {
	return &CredentialService{db: db}
}

// --- Discovered Users ---

// DiscoverUser records a discovered username on a server.
func (s *CredentialService) DiscoverUser(userID uuid.UUID, serverPath, username, role, serviceName, toolUsed string) error {
	// Check if already discovered
	var existing models.DiscoveredUser
	if err := s.db.Where("user_id = ? AND server_path = ? AND username = ?",
		userID, serverPath, username).First(&existing).Error; err == nil {
		// Already discovered
		return nil
	}

	user := &models.DiscoveredUser{
		UserID:      userID,
		ServerPath:  serverPath,
		Username:    username,
		Role:        role,
		ServiceName: serviceName,
		ToolUsed:    toolUsed,
	}
	return s.db.Create(user).Error
}

// GetDiscoveredUsers returns all users discovered on a server.
func (s *CredentialService) GetDiscoveredUsers(userID uuid.UUID, serverPath string) ([]models.DiscoveredUser, error) {
	var users []models.DiscoveredUser
	err := s.db.Where("user_id = ? AND server_path = ?", userID, serverPath).Find(&users).Error
	return users, err
}

// HasDiscoveredUsers checks if user has discovered any users on a server.
func (s *CredentialService) HasDiscoveredUsers(userID uuid.UUID, serverPath string) bool {
	var count int64
	s.db.Model(&models.DiscoveredUser{}).Where("user_id = ? AND server_path = ?", userID, serverPath).Count(&count)
	return count > 0
}

// --- Discovered Credentials ---

// SaveCredential stores a discovered credential.
func (s *CredentialService) SaveCredential(userID uuid.UUID, serverPath, serviceName, username, password, role string, credType models.CredentialType, toolUsed string) error {
	return s.SaveCredentialWithRole(userID, serverPath, serviceName, username, password, role, "", "", credType, toolUsed)
}

// SaveCredentialWithRole stores a discovered credential with full role information.
func (s *CredentialService) SaveCredentialWithRole(userID uuid.UUID, serverPath, serviceName, username, password, role, homeDir string, roleType models.RoleType, credType models.CredentialType, toolUsed string) error {
	// Determine role type if not provided
	if roleType == "" {
		switch username {
		case "root":
			roleType = models.RoleTypeRoot
		case "admin", "administrator", "sysadmin":
			roleType = models.RoleTypeAdmin
		case "guest", "anonymous":
			roleType = models.RoleTypeGuest
		default:
			roleType = models.RoleTypeUser
		}
	}

	// Determine home directory if not provided
	if homeDir == "" {
		if username == "root" {
			homeDir = "/root"
		} else {
			homeDir = "/home/" + username
		}
	}

	// Check if already discovered
	var existing models.DiscoveredCredential
	if err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ? AND username = ?",
		userID, serverPath, serviceName, username).First(&existing).Error; err == nil {
		// Already have this credential - update if password is different
		if existing.Password != password {
			existing.Password = password
			existing.Type = credType
			existing.ToolUsed = toolUsed
			existing.RoleType = roleType
			existing.HomeDir = homeDir
			return s.db.Save(&existing).Error
		}
		return nil
	}

	cred := &models.DiscoveredCredential{
		UserID:      userID,
		ServerPath:  serverPath,
		ServiceName: serviceName,
		Username:    username,
		Password:    password,
		Role:        role,
		RoleType:    roleType,
		HomeDir:     homeDir,
		Type:        credType,
		ToolUsed:    toolUsed,
	}
	return s.db.Create(cred).Error
}

// GetCredentials returns all credentials discovered for a server.
func (s *CredentialService) GetCredentials(userID uuid.UUID, serverPath string) ([]models.DiscoveredCredential, error) {
	var creds []models.DiscoveredCredential
	err := s.db.Where("user_id = ? AND server_path = ?", userID, serverPath).Find(&creds).Error
	return creds, err
}

// GetCredentialsForService returns credentials for a specific service on a server.
func (s *CredentialService) GetCredentialsForService(userID uuid.UUID, serverPath, serviceName string) ([]models.DiscoveredCredential, error) {
	var creds []models.DiscoveredCredential
	err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ?",
		userID, serverPath, serviceName).Find(&creds).Error
	return creds, err
}

// credentialPrivilegeRank returns a numeric rank for sorting (higher = more privileged).
func credentialPrivilegeRank(c *models.DiscoveredCredential) int {
	switch c.RoleType {
	case models.RoleTypeRoot:
		return 4
	case models.RoleTypeAdmin:
		return 3
	case models.RoleTypeUser:
		return 2
	case models.RoleTypeGuest:
		return 1
	default:
		// Fallback: check username
		if c.Username == "root" {
			return 4
		}
		if c.Username == "admin" || c.Username == "administrator" {
			return 3
		}
		return 2
	}
}

// GetBestCredentialForService returns the highest-privilege credential for a service (root > admin > user > guest).
func (s *CredentialService) GetBestCredentialForService(userID uuid.UUID, serverPath, serviceName string) (*models.DiscoveredCredential, error) {
	creds, err := s.GetCredentialsForService(userID, serverPath, serviceName)
	if err != nil || len(creds) == 0 {
		return nil, err
	}
	best := &creds[0]
	for i := 1; i < len(creds); i++ {
		if credentialPrivilegeRank(&creds[i]) > credentialPrivilegeRank(best) {
			best = &creds[i]
		}
	}
	return best, nil
}

// HasCredentialsForService checks if user has any credentials for a service.
func (s *CredentialService) HasCredentialsForService(userID uuid.UUID, serverPath, serviceName string) bool {
	var count int64
	s.db.Model(&models.DiscoveredCredential{}).
		Where("user_id = ? AND server_path = ? AND service_name = ?", userID, serverPath, serviceName).
		Count(&count)
	return count > 0
}

// GetAllCredentials returns all credentials discovered by a user.
func (s *CredentialService) GetAllCredentials(userID uuid.UUID) ([]models.DiscoveredCredential, error) {
	var creds []models.DiscoveredCredential
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&creds).Error
	return creds, err
}

// --- Backdoor Access ---

// CreateBackdoor records a backdoor created through RCE exploit.
func (s *CredentialService) CreateBackdoor(userID uuid.UUID, serverPath, serviceName, exploitType, toolUsed, accessLevel string) error {
	// Check if already have backdoor
	var existing models.BackdoorAccess
	if err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ?",
		userID, serverPath, serviceName).First(&existing).Error; err == nil {
		// Already have backdoor - update if needed
		if existing.AccessLevel != accessLevel {
			existing.AccessLevel = accessLevel
			return s.db.Save(&existing).Error
		}
		return nil
	}

	backdoor := &models.BackdoorAccess{
		UserID:      userID,
		ServerPath:  serverPath,
		ServiceName: serviceName,
		ExploitType: exploitType,
		ToolUsed:    toolUsed,
		AccessLevel: accessLevel,
	}
	return s.db.Create(backdoor).Error
}

// HasBackdoor checks if user has a backdoor on a server.
func (s *CredentialService) HasBackdoor(userID uuid.UUID, serverPath string) bool {
	var count int64
	s.db.Model(&models.BackdoorAccess{}).Where("user_id = ? AND server_path = ?", userID, serverPath).Count(&count)
	return count > 0
}

// HasBackdoorForService checks if user has a backdoor for a specific service.
func (s *CredentialService) HasBackdoorForService(userID uuid.UUID, serverPath, serviceName string) bool {
	var count int64
	s.db.Model(&models.BackdoorAccess{}).
		Where("user_id = ? AND server_path = ? AND service_name = ?", userID, serverPath, serviceName).
		Count(&count)
	return count > 0
}

// GetBackdoors returns all backdoors on a server.
func (s *CredentialService) GetBackdoors(userID uuid.UUID, serverPath string) ([]models.BackdoorAccess, error) {
	var backdoors []models.BackdoorAccess
	err := s.db.Where("user_id = ? AND server_path = ?", userID, serverPath).Find(&backdoors).Error
	return backdoors, err
}

// GetAllBackdoors returns all backdoors created by a user.
func (s *CredentialService) GetAllBackdoors(userID uuid.UUID) ([]models.BackdoorAccess, error) {
	var backdoors []models.BackdoorAccess
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&backdoors).Error
	return backdoors, err
}

// --- Access Checking ---

// CanAccessService checks if user can access a service (either via credentials or backdoor).
func (s *CredentialService) CanAccessService(userID uuid.UUID, serverPath, serviceName string) (bool, string) {
	// Check for backdoor first (direct access)
	if s.HasBackdoorForService(userID, serverPath, serviceName) {
		return true, "backdoor"
	}

	// Check for credentials
	if s.HasCredentialsForService(userID, serverPath, serviceName) {
		return true, "credentials"
	}

	return false, ""
}

// CanAccessServer checks if user can access any shell-granting service on a server.
// Returns true and the access method if access is possible.
func (s *CredentialService) CanAccessServer(userID uuid.UUID, serverPath string) (bool, string, string) {
	// Check for any backdoor
	var backdoor models.BackdoorAccess
	if err := s.db.Where("user_id = ? AND server_path = ?", userID, serverPath).First(&backdoor).Error; err == nil {
		return true, "backdoor", backdoor.ServiceName
	}

	// Check for any credentials on shell-granting services
	shellServices := []string{"ssh", "telnet", "ftp", "rdp", "vnc"}
	for _, svc := range shellServices {
		if s.HasCredentialsForService(userID, serverPath, svc) {
			return true, "credentials", svc
		}
	}

	return false, "", ""
}

// GetAccessInfo returns detailed access information for a server.
type AccessInfo struct {
	HasAccess     bool
	AccessMethod  string // "backdoor" or "credentials"
	ServiceName   string
	Username      string // For credentials
	AccessLevel   string // For backdoor
}

func (s *CredentialService) GetAccessInfo(userID uuid.UUID, serverPath, serviceName string) AccessInfo {
	info := AccessInfo{}

	// Check backdoor
	var backdoor models.BackdoorAccess
	if err := s.db.Where("user_id = ? AND server_path = ? AND service_name = ?",
		userID, serverPath, serviceName).First(&backdoor).Error; err == nil {
		info.HasAccess = true
		info.AccessMethod = "backdoor"
		info.ServiceName = serviceName
		info.AccessLevel = backdoor.AccessLevel
		return info
	}

	// Check credentials - use highest privilege (root > admin > user > guest)
	if cred, err := s.GetBestCredentialForService(userID, serverPath, serviceName); err == nil && cred != nil {
		info.HasAccess = true
		info.AccessMethod = "credentials"
		info.ServiceName = serviceName
		info.Username = cred.Username
		info.AccessLevel = cred.Role
		return info
	}

	return info
}

// --- Utility ---

// GeneratePassword generates a realistic-looking password based on username and server.
func GeneratePassword(username, serverIP string, role string) string {
	// Generate somewhat realistic passwords
	passwords := map[string][]string{
		"admin":    {"admin123", "Admin@2026", "P@ssw0rd!", "administrator1"},
		"root":     {"toor", "r00t123", "Root@2026!", "rootpass"},
		"user":     {"user123", "password", "qwerty123", "letmein"},
		"backup":   {"backup2026", "b4ckup!", "BackupAdmin1"},
		"dbadmin":  {"database1", "mysql123", "Db@dmin2026"},
		"guest":    {"guest", "guest123", "Welcome1"},
		"test":     {"test123", "testing", "Test@123"},
		"operator": {"operator1", "Op3r@tor", "console123"},
	}

	if pwList, ok := passwords[username]; ok {
		// Use server IP to deterministically pick a password
		idx := 0
		for _, c := range serverIP {
			idx += int(c)
		}
		return pwList[idx%len(pwList)]
	}

	// Default password pattern
	return fmt.Sprintf("%s_%s_2026", username, serverIP[len(serverIP)-2:])
}
