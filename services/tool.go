package services

import (
	"fmt"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ToolService handles tool-related operations
type ToolService struct {
	serverService *ServerService
	patchService  *PatchService // Will be set after patch service is created
}

// NewToolService creates a new tool service
func NewToolService(serverService *ServerService) *ToolService {
	return &ToolService{
		serverService: serverService,
	}
}

// SetPatchService sets the patch service (called after patch service is created)
func (s *ToolService) SetPatchService(patchService *PatchService) {
	s.patchService = patchService
}

// GetToolByName retrieves a tool by name
func (s *ToolService) GetToolByName(name string) (*models.Tool, error) {
	var tool models.Tool
	if err := database.DB.Where("name = ?", name).First(&tool).Error; err != nil {
		return nil, err
	}
	return &tool, nil
}

// GetAllTools retrieves all tools
func (s *ToolService) GetAllTools() ([]models.Tool, error) {
	var tools []models.Tool
	if err := database.DB.Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

// GetUserTools retrieves all tools owned by a user (base tool definitions)
func (s *ToolService) GetUserTools(userID uuid.UUID) ([]models.Tool, error) {
	var user models.User
	if err := database.DB.Preload("Tools").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return user.Tools, nil
}

// GetUserToolState retrieves a user's specific tool state
func (s *ToolService) GetUserToolState(userID uuid.UUID, toolName string) (*models.UserToolState, error) {
	// Get base tool
	tool, err := s.GetToolByName(toolName)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Get user's tool state
	var toolState models.UserToolState
	if err := database.DB.Where("user_id = ? AND tool_id = ?", userID, tool.ID).First(&toolState).Error; err != nil {
		return nil, fmt.Errorf("tool state not found: %w", err)
	}

	return &toolState, nil
}

// GetEffectiveTool retrieves a tool with effective stats calculated from user's tool state
func (s *ToolService) GetEffectiveTool(userID uuid.UUID, toolName string) (*models.Tool, error) {
	toolState, err := s.GetUserToolState(userID, toolName)
	if err != nil {
		return nil, err
	}

	// Calculate effective stats using patch service
	if s.patchService != nil {
		effectiveTool := s.patchService.CalculateEffectiveStats(toolState)
		if effectiveTool != nil {
			return effectiveTool, nil
		}
	}

	// Fallback to base tool if no patches applied
	var baseTool models.Tool
	if err := database.DB.First(&baseTool, "id = ?", toolState.ToolID).Error; err != nil {
		return nil, err
	}

	return &baseTool, nil
}

// CreateUserToolState creates a new tool state for a user when they download a tool
func (s *ToolService) CreateUserToolState(userID uuid.UUID, toolID uuid.UUID) (*models.UserToolState, error) {
	// Check if tool state already exists
	var existing models.UserToolState
	if err := database.DB.Where("user_id = ? AND tool_id = ?", userID, toolID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Get base tool
	var baseTool models.Tool
	if err := database.DB.First(&baseTool, "id = ?", toolID).Error; err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Create initial tool state with base stats
	toolState := &models.UserToolState{
		UserID:            userID,
		ToolID:            toolID,
		AppliedPatches:    []string{},
		EffectiveExploits: baseTool.Exploits,
		EffectiveResources: baseTool.Resources,
		Version:           1,
	}

	if err := database.DB.Create(toolState).Error; err != nil {
		return nil, fmt.Errorf("failed to create tool state: %w", err)
	}

	return toolState, nil
}

// DownloadTool downloads a tool from a server to a user
func (s *ToolService) DownloadTool(userID uuid.UUID, serverIP, toolName string) error {
	// Get server
	server, err := s.serverService.GetServerByIP(serverIP)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Check if tool exists on server
	toolExists := false
	for _, serverTool := range server.Tools {
		if serverTool == toolName {
			toolExists = true
			break
		}
	}

	if !toolExists {
		return fmt.Errorf("tool %s not found on server %s", toolName, serverIP)
	}

	// Get tool definition
	tool, err := s.GetToolByName(toolName)
	if err != nil {
		return fmt.Errorf("tool definition not found: %w", err)
	}

	// Check if user already has this tool (check UserToolState)
	_, err = s.GetUserToolState(userID, toolName)
	if err == nil {
		return fmt.Errorf("tool already owned")
	}

	// Add tool to user (many-to-many relationship for backward compatibility)
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Add tool to user's tools list
	if err := database.DB.Model(&user).Association("Tools").Append(tool); err != nil {
		return fmt.Errorf("failed to add tool: %w", err)
	}

	// CRITICAL: Create UserToolState for user-specific tool state
	_, err = s.CreateUserToolState(userID, tool.ID)
	if err != nil {
		return fmt.Errorf("failed to create tool state: %w", err)
	}

	return nil
}

// applyPatch is deprecated - use PatchService.ApplyPatch instead
// This method is kept for backward compatibility but should not be used
func (s *ToolService) applyPatch(user *models.User, patch *models.Tool) error {
	// This old method is deprecated - patches should be applied via PatchService
	// which properly updates UserToolState
	return fmt.Errorf("deprecated: use PatchService.ApplyPatch instead")
}

// UserHasTool checks if a user has a specific tool (checks UserToolState)
func (s *ToolService) UserHasTool(userID uuid.UUID, toolName string) bool {
	_, err := s.GetUserToolState(userID, toolName)
	return err == nil
}

// SeedTools seeds the database with default tools
func (s *ToolService) SeedTools() error {
	tools := []models.Tool{
		{
			Name:     "password_cracker",
			Function: "Basic password cracking",
			Resources: models.ToolResources{
				CPU:      20,
				Bandwidth: 0.3,
				RAM:      8,
			},
			Exploits: []models.Exploit{
				{Type: "password_cracking", Level: 10},
			},
			Services: "ssh",
		},
		{
			Name:     "pass_patch",
			Function: "Upgrade for password_cracker",
			Resources: models.ToolResources{
				CPU:      0,
				Bandwidth: 0,
				RAM:      0,
			},
			Exploits: []models.Exploit{
				{Type: "password_cracking", Level: 20},
			},
			IsPatch: true,
		},
		{
			Name:     "ssh_exploit",
			Function: "Exploit SSH vulnerabilities",
			Resources: models.ToolResources{
				CPU:      25,
				Bandwidth: 0.5,
				RAM:      10,
			},
			Exploits: []models.Exploit{
				{Type: "remote_code_execution", Level: 20},
			},
			Services: "ssh",
		},
		{
			Name:     "ssh_patch",
			Function: "Upgrade for ssh_exploit",
			Resources: models.ToolResources{
				CPU:      0,
				Bandwidth: 0,
				RAM:      0,
			},
			Exploits: []models.Exploit{
				{Type: "remote_code_execution", Level: 30},
			},
			IsPatch: true,
		},
		{
			Name:     "crypto_miner",
			Function: "Mine cryptocurrency (passive income)",
			Resources: models.ToolResources{
				CPU:      50,
				Bandwidth: 1.0,
				RAM:      16,
			},
			Special: "Generates passive income over time.",
		},
		{
			Name:     "user_enum",
			Function: "Enumerate users and roles",
			Resources: models.ToolResources{
				CPU:      15,
				Bandwidth: 0.2,
				RAM:      4,
			},
		},
		{
			Name:     "lan_sniffer",
			Function: "Discover local network connections",
			Resources: models.ToolResources{
				CPU:      18,
				Bandwidth: 0.4,
				RAM:      6,
			},
		},
		{
			Name:     "rootkit",
			Function: "Install hidden backdoor access",
			Resources: models.ToolResources{
				CPU:      30,
				Bandwidth: 0.6,
				RAM:      12,
			},
		},
		{
			Name:     "exploit_kit",
			Function: "Multi-vulnerability exploitation",
			Resources: models.ToolResources{
				CPU:      40,
				Bandwidth: 0.8,
				RAM:      14,
			},
			Exploits: []models.Exploit{
				{Type: "remote_code_execution", Level: 15},
				{Type: "sql_injection", Level: 15},
				{Type: "xss", Level: 10},
			},
		},
		{
			Name:     "password_sniffer",
			Function: "Sniff and crack passwords from user roles",
			Resources: models.ToolResources{
				CPU:      22,
				Bandwidth: 0.4,
				RAM:      9,
			},
			Exploits: []models.Exploit{
				{Type: "password_cracking", Level: 15},
			},
		},
		{
			Name:     "advanced_exploit_kit",
			Function: "Advanced multi-vulnerability exploitation",
			Resources: models.ToolResources{
				CPU:      60,
				Bandwidth: 1.2,
				RAM:      20,
			},
			Exploits: []models.Exploit{
				{Type: "remote_code_execution", Level: 25},
				{Type: "sql_injection", Level: 25},
				{Type: "xss", Level: 20},
				{Type: "buffer_overflow", Level: 20},
			},
		},
		{
			Name:     "sql_injector",
			Function: "Perform SQL injection attacks",
			Resources: models.ToolResources{
				CPU:      28,
				Bandwidth: 0.5,
				RAM:      11,
			},
			Exploits: []models.Exploit{
				{Type: "sql_injection", Level: 20},
			},
			Services: "http",
		},
		{
			Name:     "xss_exploit",
			Function: "Exploit XSS vulnerabilities",
			Resources: models.ToolResources{
				CPU:      20,
				Bandwidth: 0.3,
				RAM:      7,
			},
			Exploits: []models.Exploit{
				{Type: "xss", Level: 15},
			},
			Services: "http",
		},
		{
			Name:     "packet_capture",
			Function: "Capture network packets",
			Resources: models.ToolResources{
				CPU:      16,
				Bandwidth: 0.3,
				RAM:      5,
			},
		},
		{
			Name:     "packet_decoder",
			Function: "Decode captured packets",
			Resources: models.ToolResources{
				CPU:      12,
				Bandwidth: 0.1,
				RAM:      3,
			},
		},
	}

	for _, tool := range tools {
		// Check if tool already exists
		var existing models.Tool
		err := database.DB.Where("name = ?", tool.Name).First(&existing).Error
		if err != nil && err == gorm.ErrRecordNotFound {
			// Tool doesn't exist, create it
			if err := database.DB.Create(&tool).Error; err != nil {
				return fmt.Errorf("failed to seed tool %s: %w", tool.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check tool %s: %w", tool.Name, err)
		}
		// If tool exists, skip it
	}

	return nil
}

