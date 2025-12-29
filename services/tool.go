package services

import (
	"encoding/json"
	"fmt"
	"os"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ToolService handles tool-related operations including retrieval, ownership, and effective tool calculations.
type ToolService struct {
	db            *database.Database
	serverService *ServerService
	patchService  *PatchService // Will be set after patch service is created
}

// NewToolService creates a new ToolService with the provided database and server service.
func NewToolService(db *database.Database, serverService *ServerService) *ToolService {
	return &ToolService{
		db:            db,
		serverService: serverService,
	}
}

// SetPatchService sets the patch service for this tool service (called after patch service is created).
func (s *ToolService) SetPatchService(patchService *PatchService) {
	s.patchService = patchService
}

// GetToolByName retrieves a tool by its name from the database.
func (s *ToolService) GetToolByName(name string) (*models.Tool, error) {
	var tool models.Tool
	if err := s.db.Where("name = ?", name).First(&tool).Error; err != nil {
		return nil, err
	}
	return &tool, nil
}

// GetAllTools retrieves all tools from the database.
func (s *ToolService) GetAllTools() ([]models.Tool, error) {
	var tools []models.Tool
	if err := s.db.Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

// GetUserTools retrieves all base tool definitions owned by a user.
func (s *ToolService) GetUserTools(userID uuid.UUID) ([]models.Tool, error) {
	var user models.User
	if err := s.db.Preload("Tools").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return user.Tools, nil
}

// GetUserToolState retrieves a user's specific tool state, including applied patches and calculated properties.
func (s *ToolService) GetUserToolState(userID uuid.UUID, toolName string) (*models.UserToolState, error) {
	// Get base tool
	tool, err := s.GetToolByName(toolName)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Get user's tool state
	var toolState models.UserToolState
	if err := s.db.Where("user_id = ? AND tool_id = ?", userID, tool.ID).First(&toolState).Error; err != nil {
		return nil, fmt.Errorf("tool state not found: %w", err)
	}

	return &toolState, nil
}

// GetEffectiveTool retrieves a tool with effective stats calculated from the user's tool state (with patches applied).
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
	if err := s.db.First(&baseTool, "id = ?", toolState.ToolID).Error; err != nil {
		return nil, err
	}

	return &baseTool, nil
}

// CreateUserToolState creates a new tool state for a user when they download a tool
func (s *ToolService) CreateUserToolState(userID uuid.UUID, toolID uuid.UUID) (*models.UserToolState, error) {
	// Check if tool state already exists
	var existing models.UserToolState
	if err := s.db.Where("user_id = ? AND tool_id = ?", userID, toolID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Get base tool
	var baseTool models.Tool
	if err := s.db.First(&baseTool, "id = ?", toolID).Error; err != nil {
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

	if err := s.db.Create(toolState).Error; err != nil {
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

	// Add tool to user (many-to-many relationship)
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Add tool to user's tools list
	if err := s.db.Model(&user).Association("Tools").Append(tool); err != nil {
		return fmt.Errorf("failed to add tool: %w", err)
	}

	// CRITICAL: Create UserToolState for user-specific tool state
	_, err = s.CreateUserToolState(userID, tool.ID)
	if err != nil {
		return fmt.Errorf("failed to create tool state: %w", err)
	}

	return nil
}

// UserHasTool checks if a user has a specific tool (checks UserToolState)
func (s *ToolService) UserHasTool(userID uuid.UUID, toolName string) bool {
	_, err := s.GetUserToolState(userID, toolName)
	return err == nil
}

// SeedTools seeds the database with default tools from JSON file
func (s *ToolService) SeedTools() error {
	// Load tools from JSON file
	data, err := os.ReadFile("data/seed/tools.json")
	if err != nil {
		return fmt.Errorf("failed to read tools.json: %w", err)
	}

	var toolData struct {
		Tools []models.Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &toolData); err != nil {
		return fmt.Errorf("failed to parse tools.json: %w", err)
	}

	tools := toolData.Tools

	for _, tool := range tools {
		// Check if tool already exists
		var existing models.Tool
		err := s.db.Where("name = ?", tool.Name).First(&existing).Error
		if err != nil && err == gorm.ErrRecordNotFound {
			// Tool doesn't exist, create it
			if err := s.db.Create(&tool).Error; err != nil {
				return fmt.Errorf("failed to seed tool %s: %w", tool.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check tool %s: %w", tool.Name, err)
		}
		// If tool exists, skip it
	}

	return nil
}

