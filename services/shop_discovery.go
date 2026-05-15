package services

import (
	"encoding/json"
	"fmt"

	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// ShopDiscovery handles shop discovery logic, automatically creating shops on servers when discovered.
type ShopDiscovery struct {
	db            *database.Database
	shopService   *ShopService
	serverService *ServerService
	toolService   *ToolService
	userService   *UserService
}

// NewShopDiscovery creates a new ShopDiscovery service with the provided dependencies.
func NewShopDiscovery(shopService *ShopService, serverService *ServerService, toolService *ToolService) *ShopDiscovery {
	return &ShopDiscovery{
		shopService:   shopService,
		serverService: serverService,
		toolService:   toolService,
	}
}

// SetDatabase sets the database connection for mission queries
func (s *ShopDiscovery) SetDatabase(db *database.Database) {
	s.db = db
}

// SetUserService sets the user service for level checks
func (s *ShopDiscovery) SetUserService(userService *UserService) {
	s.userService = userService
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

// findShopMetadataInFilesystem searches for shop metadata in filesystem structure using FileReader.
func (s *ShopDiscovery) findShopMetadataInFilesystem(fs map[string]interface{}) (string, bool) {
	reader := filesystem.NewMapFileReader(fs)
	locations := []string{"shop.json", "shop_config.json", "etc/shop.json", "var/shop.json"}

	for _, location := range locations {
		content, err := reader.ReadFile(location)
		if err == nil && content != "" {
			return content, true
		}
	}

	return "", false
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

// DiscoverShops finds all shops accessible to a user (filtered by mission completion and level)
func (s *ShopDiscovery) DiscoverShops(userID uuid.UUID) ([]models.Shop, error) {
	// Get all shops
	allShops, err := s.shopService.GetAllShops()
	if err != nil {
		return nil, err
	}

	// Filter shops based on requirements
	var accessibleShops []models.Shop
	for _, shop := range allShops {
		accessible, _ := s.CanAccessShop(userID, &shop)
		if accessible {
			accessibleShops = append(accessibleShops, shop)
		}
	}

	return accessibleShops, nil
}

// CanAccessShop checks if a user can access a shop based on mission completion and level
func (s *ShopDiscovery) CanAccessShop(userID uuid.UUID, shop *models.Shop) (bool, string) {
	// Check level requirement
	if shop.RequiredLevel > 0 && s.userService != nil {
		user, err := s.userService.GetUserByID(userID)
		if err != nil {
			return false, "Failed to check user level"
		}
		if user.Level < shop.RequiredLevel {
			return false, fmt.Sprintf("Requires level %d (you are level %d)", shop.RequiredLevel, user.Level)
		}
	}

	// Check mission requirement
	if shop.RequiredMission != "" && s.db != nil {
		var userMission models.UserMission
		err := s.db.Where("user_id = ? AND mission_id = ? AND status = ?", 
			userID, shop.RequiredMission, "completed").First(&userMission).Error
		if err != nil {
			return false, fmt.Sprintf("Requires completion of mission: %s", shop.RequiredMission)
		}
	}

	return true, ""
}

// GetAllShopsWithStatus returns all shops with their accessibility status for a user
func (s *ShopDiscovery) GetAllShopsWithStatus(userID uuid.UUID) ([]ShopWithStatus, error) {
	allShops, err := s.shopService.GetAllShops()
	if err != nil {
		return nil, err
	}

	var result []ShopWithStatus
	for _, shop := range allShops {
		accessible, reason := s.CanAccessShop(userID, &shop)
		result = append(result, ShopWithStatus{
			Shop:       shop,
			Accessible: accessible,
			Reason:     reason,
		})
	}

	return result, nil
}

// ShopWithStatus represents a shop with its accessibility status
type ShopWithStatus struct {
	Shop       models.Shop
	Accessible bool
	Reason     string // Reason why shop is inaccessible (empty if accessible)
}

