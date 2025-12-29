package services

import (
	"encoding/json"
	"fmt"

	"terminal-sh/models"

	"github.com/google/uuid"
)

// ShopDiscovery handles shop discovery logic, automatically creating shops on servers when discovered.
type ShopDiscovery struct {
	shopService    *ShopService
	serverService  *ServerService
	patchService   *PatchService
	toolService    *ToolService
}

// NewShopDiscovery creates a new ShopDiscovery service with the provided dependencies.
func NewShopDiscovery(shopService *ShopService, serverService *ServerService, patchService *PatchService, toolService *ToolService) *ShopDiscovery {
	return &ShopDiscovery{
		shopService:   shopService,
		serverService: serverService,
		patchService:  patchService,
		toolService:  toolService,
	}
}

// DiscoverShopsOnServer checks if a server has a shop configuration and creates it if found.
func (s *ShopDiscovery) DiscoverShopsOnServer(serverIP string) (*models.Shop, error) {
	// Check if shop already exists
	shop, err := s.shopService.GetShopByServerIP(serverIP)
	if err == nil {
		return shop, nil
	}

	// Try to find shop metadata file
	shop, err = s.FindShopFiles(serverIP)
	if err == nil && shop != nil {
		return shop, nil
	}

	return nil, fmt.Errorf("no shop found on server")
}

// FindShopFiles searches filesystem for shop metadata files
func (s *ShopDiscovery) FindShopFiles(serverIP string) (*models.Shop, error) {
	// Get server
	server, err := s.serverService.GetServerByIP(serverIP)
	if err != nil {
		return nil, err
	}

	// Search filesystem for shop.json or shop metadata
	// This is a simplified version - in reality, we'd traverse the filesystem
	// For now, we'll check if server has shop metadata in a known location
	if shopData, found := s.findShopMetadataInFilesystem(server.FileSystem); found {
		return s.ParseShopMetadata(serverIP, shopData)
	}

	return nil, fmt.Errorf("shop metadata file not found")
}

// findShopMetadataInFilesystem searches for shop metadata in filesystem structure
func (s *ShopDiscovery) findShopMetadataInFilesystem(filesystem map[string]interface{}) (string, bool) {
	// Look for shop.json in common locations
	locations := []string{"shop.json", "shop_config.json", "etc/shop.json", "var/shop.json"}

	for _, location := range locations {
		if content := s.getFileContent(filesystem, location); content != "" {
			return content, true
		}
	}

	return "", false
}

// getFileContent extracts file content from filesystem structure
func (s *ShopDiscovery) getFileContent(filesystem map[string]interface{}, path string) string {
	// Simple path traversal - split by /
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

	// Traverse filesystem
	currentLevel := filesystem
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - should be file
			if file, ok := currentLevel[part].(map[string]interface{}); ok {
				if content, ok := file["content"].(string); ok {
					return content
				}
			}
		} else {
			// Directory
			if dir, ok := currentLevel[part].(map[string]interface{}); ok {
				currentLevel = dir
			} else {
				return ""
			}
		}
	}

	return ""
}

// ParseShopMetadata parses shop metadata from JSON content
func (s *ShopDiscovery) ParseShopMetadata(serverIP, fileContent string) (*models.Shop, error) {
	var shopData struct {
		ShopType    string `json:"shop_type"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Items       []struct {
			Type        string  `json:"type"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			PriceCrypto float64 `json:"price_crypto"`
			PriceData   float64 `json:"price_data"`
			Stock       int     `json:"stock"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(fileContent), &shopData); err != nil {
		return nil, fmt.Errorf("failed to parse shop metadata: %w", err)
	}

	// Create shop
	shopType := models.ShopType(shopData.ShopType)
	if shopType == "" {
		shopType = models.ShopTypeMixed
	}

	shop, err := s.shopService.CreateShop(serverIP, shopType, shopData.Name, shopData.Description)
	if err != nil {
		return nil, err
	}

	// Add items
	for _, itemData := range shopData.Items {
		itemType := models.ItemType(itemData.Type)
		if itemType == "" {
			continue
		}

		stock := itemData.Stock
		if stock == 0 {
			stock = -1 // Unlimited
		}

		_, err := s.shopService.AddShopItem(
			shop.ID,
			itemType,
			itemData.Name,
			itemData.Description,
			itemData.PriceCrypto,
			itemData.PriceData,
			stock,
		)
		if err != nil {
			// Log error but continue
			continue
		}
	}

	return shop, nil
}

// DiscoverShops finds all shops for a user (via scanning)
func (s *ShopDiscovery) DiscoverShops(userID uuid.UUID) ([]models.Shop, error) {
	// Get all shops
	shops, err := s.shopService.GetAllShops()
	if err != nil {
		return nil, err
	}

	return shops, nil
}

