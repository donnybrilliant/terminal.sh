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

// PatchService handles patch-related operations
type PatchService struct {
	db          *database.Database
	toolService *ToolService
}

// NewPatchService creates a new patch service
func NewPatchService(db *database.Database, toolService *ToolService) *PatchService {
	return &PatchService{
		db:          db,
		toolService: toolService,
	}
}

// GetPatchByName retrieves a patch by name
func (s *PatchService) GetPatchByName(name string) (*models.Patch, error) {
	var patch models.Patch
	if err := s.db.Where("name = ?", name).First(&patch).Error; err != nil {
		return nil, err
	}
	return &patch, nil
}

// GetAllPatches retrieves all patches
func (s *PatchService) GetAllPatches() ([]models.Patch, error) {
	var patches []models.Patch
	if err := s.db.Find(&patches).Error; err != nil {
		return nil, err
	}
	return patches, nil
}

// GetPatchesForTool retrieves all patches that target a specific tool
func (s *PatchService) GetPatchesForTool(toolName string) ([]models.Patch, error) {
	var patches []models.Patch
	if err := s.db.Where("target_tool = ?", toolName).Find(&patches).Error; err != nil {
		return nil, err
	}
	return patches, nil
}

// LoadPatchFromFile parses patch metadata from JSON file content
func (s *PatchService) LoadPatchFromFile(fileContent string) (*models.Patch, error) {
	var patchData struct {
		Name        string `json:"name"`
		TargetTool  string `json:"target_tool"`
		Description string `json:"description"`
		Upgrades    struct {
			Exploits  []models.Exploit      `json:"exploits"`
			Resources models.ToolResources  `json:"resources"`
		} `json:"upgrades"`
	}

	if err := json.Unmarshal([]byte(fileContent), &patchData); err != nil {
		return nil, fmt.Errorf("failed to parse patch file: %w", err)
	}

	patch := &models.Patch{
		Name:        patchData.Name,
		TargetTool:  patchData.TargetTool,
		Description: patchData.Description,
		Upgrades: models.PatchUpgrades{
			Exploits:  patchData.Upgrades.Exploits,
			Resources: patchData.Upgrades.Resources,
		},
	}

	return patch, nil
}

// ApplyPatch applies a patch to a user's tool instance
func (s *PatchService) ApplyPatch(userID uuid.UUID, toolName, patchName string) error {
	// Get user's tool state
	toolState, err := s.toolService.GetUserToolState(userID, toolName)
	if err != nil {
		return fmt.Errorf("tool not owned by user: %w", err)
	}

	// Check if patch already applied
	for _, appliedPatch := range toolState.AppliedPatches {
		if appliedPatch == patchName {
			return fmt.Errorf("patch %s already applied to this tool", patchName)
		}
	}

	// Get patch definition
	patch, err := s.GetPatchByName(patchName)
	if err != nil {
		return fmt.Errorf("patch not found: %w", err)
	}

	// Verify patch targets this tool
	if patch.TargetTool != toolName {
		return fmt.Errorf("patch %s targets %s, not %s", patchName, patch.TargetTool, toolName)
	}

	// Add patch to applied patches
	toolState.AppliedPatches = append(toolState.AppliedPatches, patchName)

	// Recalculate effective stats
	effectiveTool := s.CalculateEffectiveStats(toolState)
	toolState.EffectiveExploits = effectiveTool.Exploits
	toolState.EffectiveResources = effectiveTool.Resources
	toolState.Version++

	// Save updated tool state
	if err := s.db.Save(toolState).Error; err != nil {
		return fmt.Errorf("failed to save tool state: %w", err)
	}

	return nil
}

// CalculateEffectiveStats calculates effective tool stats from base tool + applied patches
func (s *PatchService) CalculateEffectiveStats(toolState *models.UserToolState) *models.Tool {
	// Get base tool
	var baseTool models.Tool
	if err := s.db.First(&baseTool, "id = ?", toolState.ToolID).Error; err != nil {
		return nil
	}

	// Start with base tool stats
	effectiveTool := &models.Tool{
		ID:        baseTool.ID,
		Name:      baseTool.Name,
		Function:  baseTool.Function,
		Resources: baseTool.Resources,
		Exploits:  make([]models.Exploit, len(baseTool.Exploits)),
		Services:  baseTool.Services,
		Special:   baseTool.Special,
	}

	// Copy base exploits
	copy(effectiveTool.Exploits, baseTool.Exploits)

	// Apply each patch
	for _, patchName := range toolState.AppliedPatches {
		var patch models.Patch
		if err := s.db.Where("name = ?", patchName).First(&patch).Error; err != nil {
			continue // Skip if patch not found
		}

		// Apply resource modifications
		effectiveTool.Resources.CPU += patch.Upgrades.Resources.CPU
		effectiveTool.Resources.Bandwidth += patch.Upgrades.Resources.Bandwidth
		effectiveTool.Resources.RAM += patch.Upgrades.Resources.RAM

		// Ensure resources don't go negative
		if effectiveTool.Resources.CPU < 0 {
			effectiveTool.Resources.CPU = 0
		}
		if effectiveTool.Resources.Bandwidth < 0 {
			effectiveTool.Resources.Bandwidth = 0
		}
		if effectiveTool.Resources.RAM < 0 {
			effectiveTool.Resources.RAM = 0
		}

		// Apply exploit upgrades (replace or add exploits)
		for _, patchExploit := range patch.Upgrades.Exploits {
			found := false
			for i, existingExploit := range effectiveTool.Exploits {
				if existingExploit.Type == patchExploit.Type {
					// Upgrade existing exploit if patch level is higher
					if patchExploit.Level > existingExploit.Level {
						effectiveTool.Exploits[i].Level = patchExploit.Level
					}
					found = true
					break
				}
			}
			if !found {
				// Add new exploit type
				effectiveTool.Exploits = append(effectiveTool.Exploits, patchExploit)
			}
		}
	}

	return effectiveTool
}

// UserOwnsPatch checks if a user owns a patch (for shop purchases)
func (s *PatchService) UserOwnsPatch(userID uuid.UUID, patchName string) bool {
	var patch models.Patch
	if err := s.db.Where("name = ?", patchName).First(&patch).Error; err != nil {
		return false
	}

	var userPatch models.UserPatch
	if err := s.db.Where("user_id = ? AND patch_id = ?", userID, patch.ID).First(&userPatch).Error; err != nil {
		return false
	}

	return true
}

// AddUserPatch adds a patch to a user's inventory (for shop purchases)
func (s *PatchService) AddUserPatch(userID uuid.UUID, patchName string) error {
	var patch models.Patch
	if err := s.db.Where("name = ?", patchName).First(&patch).Error; err != nil {
		return fmt.Errorf("patch not found: %w", err)
	}

	// Check if user already owns this patch
	if s.UserOwnsPatch(userID, patchName) {
		return fmt.Errorf("user already owns patch %s", patchName)
	}

	userPatch := &models.UserPatch{
		UserID:  userID,
		PatchID: patch.ID,
	}

	if err := s.db.Create(userPatch).Error; err != nil {
		return fmt.Errorf("failed to add patch to user: %w", err)
	}

	return nil
}

// GetUserPatches retrieves all patches owned by a user
func (s *PatchService) GetUserPatches(userID uuid.UUID) ([]models.Patch, error) {
	var userPatches []models.UserPatch
	if err := s.db.Where("user_id = ?", userID).Preload("Patch").Find(&userPatches).Error; err != nil {
		return nil, err
	}

	patches := make([]models.Patch, len(userPatches))
	for i, up := range userPatches {
		patches[i] = up.Patch
	}

	return patches, nil
}

// DiscoverPatchesFromServer searches for patch files on a server's filesystem
func (s *PatchService) DiscoverPatchesFromServer(serverPath string, filesystem map[string]interface{}) ([]*models.Patch, error) {
	var discoveredPatches []*models.Patch
	
	// Search common locations for patch files
	locations := []string{"patches", "tools/patches", "var/patches", "etc/patches", "home/admin/patches"}
	
	for _, location := range locations {
		if content := s.getFileContent(filesystem, location+".json"); content != "" {
			patch, err := s.LoadPatchFromFile(content)
			if err == nil {
				discoveredPatches = append(discoveredPatches, patch)
			}
		}
	}
	
	// Also search for individual patch files
	if patchesDir, ok := filesystem["patches"].(map[string]interface{}); ok {
		for fileName, fileData := range patchesDir {
			if fileMap, ok := fileData.(map[string]interface{}); ok {
				if content, ok := fileMap["content"].(string); ok && len(fileName) > 5 && fileName[len(fileName)-5:] == ".json" {
					patch, err := s.LoadPatchFromFile(content)
					if err == nil {
						discoveredPatches = append(discoveredPatches, patch)
					}
				}
			}
		}
	}
	
	return discoveredPatches, nil
}

// getFileContent extracts file content from filesystem structure
func (s *PatchService) getFileContent(filesystem map[string]interface{}, path string) string {
	parts := []string{}
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	currentLevel := filesystem
	for i, part := range parts {
		if i == len(parts)-1 {
			if file, ok := currentLevel[part].(map[string]interface{}); ok {
				if content, ok := file["content"].(string); ok {
					return content
				}
			}
		} else {
			if dir, ok := currentLevel[part].(map[string]interface{}); ok {
				currentLevel = dir
			} else {
				return ""
			}
		}
	}
	return ""
}

// SeedPatches seeds the database with default patches from JSON file
func (s *PatchService) SeedPatches() error {
	// Load patches from JSON file
	data, err := os.ReadFile("data/seed/patches.json")
	if err != nil {
		return fmt.Errorf("failed to read patches.json: %w", err)
	}

	var patchData struct {
		Patches []models.Patch `json:"patches"`
	}
	if err := json.Unmarshal(data, &patchData); err != nil {
		return fmt.Errorf("failed to parse patches.json: %w", err)
	}

	patches := patchData.Patches

	for _, patch := range patches {
		var existing models.Patch
		err := s.db.Where("name = ?", patch.Name).First(&existing).Error
		if err != nil && err == gorm.ErrRecordNotFound {
			if err := s.db.Create(&patch).Error; err != nil {
				return fmt.Errorf("failed to seed patch %s: %w", patch.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check patch %s: %w", patch.Name, err)
		}
	}

	return nil
}

