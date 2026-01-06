package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShopType represents the type of shop.
type ShopType string

const (
	ShopTypeRepo      ShopType = "repo"      // Free downloadable resources
	ShopTypeTools     ShopType = "tools"     // Purchasable tools
	ShopTypeResources ShopType = "resources" // CPU/RAM/Bandwidth upgrades
	ShopTypeMixed     ShopType = "mixed"     // Combination of above
)

// Shop represents a shop on a server where users can purchase items.
type Shop struct {
	ID          uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	ServerIP    string    `gorm:"uniqueIndex;not null" json:"server_ip"`
	ShopType    ShopType  `gorm:"not null" json:"shop_type"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Items []ShopItem `gorm:"foreignKey:ShopID" json:"items,omitempty"`
}

// BeforeCreate is a GORM hook that generates a UUID for the shop if one doesn't exist.
func (s *Shop) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// ItemType represents the type of shop item.
type ItemType string

const (
	ItemTypeTool     ItemType = "tool"     // Hacking tool
	ItemTypePatch    ItemType = "patch"    // Tool upgrade patch
	ItemTypeResource ItemType = "resource" // Resource upgrade
)

// ShopItem represents an item for sale in a shop.
type ShopItem struct {
	ID          uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	ShopID      uuid.UUID `gorm:"not null" json:"shop_id"`
	ItemType    ItemType  `gorm:"not null" json:"item_type"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	PriceCrypto float64   `gorm:"default:0" json:"price_crypto"`
	PriceData   float64   `gorm:"default:0" json:"price_data"`
	Stock       int       `gorm:"default:-1" json:"stock"` // -1 means unlimited
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Shop Shop `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
}

// BeforeCreate is a GORM hook that generates a UUID for the shop item if one doesn't exist.
func (si *ShopItem) BeforeCreate(tx *gorm.DB) error {
	if si.ID == uuid.Nil {
		si.ID = uuid.New()
	}
	return nil
}

// UserPurchase represents a purchase made by a user from a shop.
type UserPurchase struct {
	ID        uuid.UUID `gorm:"type:text;primary_key" json:"id"`
	UserID    uuid.UUID `gorm:"not null;index" json:"user_id"`
	ShopID    uuid.UUID `gorm:"not null" json:"shop_id"`
	ItemID    uuid.UUID `gorm:"not null" json:"item_id"`
	ItemName  string    `gorm:"not null" json:"item_name"`
	ItemType  ItemType  `gorm:"not null" json:"item_type"`
	PriceCrypto float64 `json:"price_crypto"`
	PriceData   float64 `json:"price_data"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Shop Shop `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
}

// BeforeCreate is a GORM hook that generates a UUID for the user purchase if one doesn't exist.
func (up *UserPurchase) BeforeCreate(tx *gorm.DB) error {
	if up.ID == uuid.Nil {
		up.ID = uuid.New()
	}
	return nil
}

