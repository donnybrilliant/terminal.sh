package services

import (
	"fmt"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShopService handles shop-related operations
type ShopService struct {
	db            *database.Database
	serverService *ServerService
}

// NewShopService creates a new shop service
func NewShopService(db *database.Database, serverService *ServerService) *ShopService {
	return &ShopService{
		db:            db,
		serverService: serverService,
	}
}

// GetShopByServerIP retrieves a shop by server IP
func (s *ShopService) GetShopByServerIP(serverIP string) (*models.Shop, error) {
	var shop models.Shop
	if err := s.db.Where("server_ip = ?", serverIP).Preload("Items").First(&shop).Error; err != nil {
		return nil, err
	}
	return &shop, nil
}

// GetAllShops retrieves all shops
func (s *ShopService) GetAllShops() ([]models.Shop, error) {
	var shops []models.Shop
	if err := s.db.Preload("Items").Find(&shops).Error; err != nil {
		return nil, err
	}
	return shops, nil
}

// GetShopItems retrieves all items for a shop
func (s *ShopService) GetShopItems(shopID uuid.UUID) ([]models.ShopItem, error) {
	var items []models.ShopItem
	if err := s.db.Where("shop_id = ?", shopID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// PurchaseItem purchases an item from a shop
func (s *ShopService) PurchaseItem(userID uuid.UUID, shopID uuid.UUID, itemID uuid.UUID) error {
	// Get shop item
	var item models.ShopItem
	if err := s.db.Where("id = ? AND shop_id = ?", itemID, shopID).First(&item).Error; err != nil {
		return fmt.Errorf("item not found in shop")
	}

	// Check stock
	if item.Stock >= 0 && item.Stock == 0 {
		return fmt.Errorf("item out of stock")
	}

	// Get user
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Check if user has enough currency
	if item.PriceCrypto > 0 && user.Wallet.Crypto < item.PriceCrypto {
		return fmt.Errorf("insufficient crypto currency (need %.2f, have %.2f)", item.PriceCrypto, user.Wallet.Crypto)
	}
	if item.PriceData > 0 && user.Wallet.Data < item.PriceData {
		return fmt.Errorf("insufficient data currency (need %.2f, have %.2f)", item.PriceData, user.Wallet.Data)
	}

	// Deduct currency
	user.Wallet.Crypto -= item.PriceCrypto
	user.Wallet.Data -= item.PriceData
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	// Reduce stock if not unlimited
	if item.Stock > 0 {
		item.Stock--
		if err := s.db.Save(&item).Error; err != nil {
			return fmt.Errorf("failed to update stock: %w", err)
		}
	}

	// Record purchase
	purchase := &models.UserPurchase{
		UserID:      userID,
		ShopID:      shopID,
		ItemID:      itemID,
		ItemName:    item.Name,
		ItemType:    item.ItemType,
		PriceCrypto: item.PriceCrypto,
		PriceData:   item.PriceData,
	}
	if err := s.db.Create(purchase).Error; err != nil {
		return fmt.Errorf("failed to record purchase: %w", err)
	}

	// Handle item based on type
	switch item.ItemType {
	case models.ItemTypeTool:
		// Tool will be added via tool service
		return nil
	case models.ItemTypePatch:
		// Patch will be added via patch service
		return nil
	case models.ItemTypeResource:
		// Resource upgrade
		return s.applyResourceUpgrade(&user, item.Name)
	default:
		return fmt.Errorf("unknown item type")
	}
}

// applyResourceUpgrade applies a resource upgrade to a user
func (s *ShopService) applyResourceUpgrade(user *models.User, upgradeName string) error {
	// Define resource upgrades
	upgrades := map[string]models.Resources{
		"cpu_boost":      {CPU: 50, Bandwidth: 0, RAM: 0},
		"bandwidth_boost": {CPU: 0, Bandwidth: 50, RAM: 0},
		"ram_boost":      {CPU: 0, Bandwidth: 0, RAM: 8},
		"full_boost":      {CPU: 100, Bandwidth: 100, RAM: 16},
	}

	upgrade, exists := upgrades[upgradeName]
	if !exists {
		return fmt.Errorf("unknown resource upgrade: %s", upgradeName)
	}

	user.Resources.CPU += upgrade.CPU
	user.Resources.Bandwidth += upgrade.Bandwidth
	user.Resources.RAM += upgrade.RAM

	if err := s.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to apply resource upgrade: %w", err)
	}

	return nil
}

// CreateShop creates a new shop on a server
func (s *ShopService) CreateShop(serverIP string, shopType models.ShopType, name, description string) (*models.Shop, error) {
	// Check if shop already exists
	var existing models.Shop
	if err := s.db.Where("server_ip = ?", serverIP).First(&existing).Error; err == nil {
		return &existing, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	shop := &models.Shop{
		ServerIP:    serverIP,
		ShopType:    shopType,
		Name:        name,
		Description: description,
	}

	if err := s.db.Create(shop).Error; err != nil {
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	return shop, nil
}

// AddShopItem adds an item to a shop
func (s *ShopService) AddShopItem(shopID uuid.UUID, itemType models.ItemType, name, description string, priceCrypto, priceData float64, stock int) (*models.ShopItem, error) {
	item := &models.ShopItem{
		ShopID:      shopID,
		ItemType:    itemType,
		Name:        name,
		Description: description,
		PriceCrypto:  priceCrypto,
		PriceData:    priceData,
		Stock:        stock,
	}

	if err := s.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to create shop item: %w", err)
	}

	return item, nil
}

