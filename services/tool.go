package services

import (
	"fmt"

	"ssh4xx-go/database"
	"ssh4xx-go/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ToolService handles tool-related operations
type ToolService struct {
	serverService *ServerService
}

// NewToolService creates a new tool service
func NewToolService(serverService *ServerService) *ToolService {
	return &ToolService{
		serverService: serverService,
	}
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

// GetUserTools retrieves all tools owned by a user
func (s *ToolService) GetUserTools(userID uuid.UUID) ([]models.Tool, error) {
	var user models.User
	if err := database.DB.Preload("Tools").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return user.Tools, nil
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

	// Add tool to user (many-to-many relationship)
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user already has this tool
	var existingTool models.Tool
	if err := database.DB.Model(&user).Association("Tools").Find(&existingTool, "name = ?", toolName); err == nil && existingTool.ID != uuid.Nil {
		// Check if it's a patch
		if tool.IsPatch {
			// Apply patch (upgrade existing tool)
			return s.applyPatch(&user, tool)
		}
		return fmt.Errorf("tool already owned")
	}

	// Add tool to user
	if err := database.DB.Model(&user).Association("Tools").Append(tool); err != nil {
		return fmt.Errorf("failed to add tool: %w", err)
	}

	return nil
}

// applyPatch applies a patch tool to upgrade an existing tool
func (s *ToolService) applyPatch(user *models.User, patch *models.Tool) error {
	// Find the tool this patch upgrades
	// For now, we'll assume patches upgrade tools with the same base name
	// (e.g., "pass_patch" upgrades "password_cracker")
	baseName := ""
	if patch.Name == "pass_patch" {
		baseName = "password_cracker"
	} else if patch.Name == "ssh_patch" {
		baseName = "ssh_exploit"
	} else {
		return fmt.Errorf("unknown patch type")
	}

	// Find user's tool
	var userTool models.Tool
	if err := database.DB.Model(user).Association("Tools").Find(&userTool, "name = ?", baseName); err != nil {
		return fmt.Errorf("base tool not found for patch")
	}

	// Update tool exploits with patch levels
	// For simplicity, we'll just add the patch's exploits
	userTool.Exploits = patch.Exploits

	// Save updated tool
	if err := database.DB.Save(&userTool).Error; err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	return nil
}

// UserHasTool checks if a user has a specific tool
func (s *ToolService) UserHasTool(userID uuid.UUID, toolName string) bool {
	tools, err := s.GetUserTools(userID)
	if err != nil {
		return false
	}

	for _, tool := range tools {
		if tool.Name == toolName {
			return true
		}
	}

	return false
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

