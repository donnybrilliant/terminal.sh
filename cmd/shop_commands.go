package cmd

import (
	"fmt"
	"strings"

	"terminal-sh/models"

	"github.com/google/uuid"
)

// handleSHOP handles shop-related commands
func (h *CommandHandler) handleSHOP(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) == 0 {
		return h.handleShopList()
	}

	shopID := args[0]
	return h.handleShopBrowse(shopID)
}

// handleShopList lists all discovered shops
func (h *CommandHandler) handleShopList() *CommandResult {
	if h.shopDiscovery == nil {
		return &CommandResult{Error: fmt.Errorf("shop discovery not available")}
	}

	shops, err := h.shopDiscovery.DiscoverShops(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(shops) == 0 {
		return &CommandResult{Output: "No shops discovered yet. Scan servers to find shops.\n"}
	}

	var output strings.Builder
	output.WriteString("Discovered Shops:\n\n")
	
	for _, shop := range shops {
		output.WriteString(fmt.Sprintf("  [%s] %s\n", shop.ID.String()[:8], shop.Name))
		output.WriteString(fmt.Sprintf("    Type: %s\n", shop.ShopType))
		output.WriteString(fmt.Sprintf("    Location: %s\n", shop.ServerIP))
		output.WriteString(fmt.Sprintf("    %s\n\n", shop.Description))
	}

	output.WriteString("Usage: shop <shopID> - Browse shop inventory\n")
	output.WriteString("       buy <shopID> <itemID> - Purchase item\n")

	return &CommandResult{Output: output.String()}
}

// handleShopBrowse displays shop inventory
func (h *CommandHandler) handleShopBrowse(shopID string) *CommandResult {
	if h.shopService == nil {
		return &CommandResult{Error: fmt.Errorf("shop service not available")}
	}

	// Parse shop ID
	shopUUID, err := uuid.Parse(shopID)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("invalid shop ID")}
	}

	shop, err := h.shopService.GetShopByServerIP(shopID)
	if err != nil {
		// Try by UUID - get all shops and find by UUID
		shops, err := h.shopService.GetAllShops()
		if err == nil {
			for i := range shops {
				if shops[i].ID == shopUUID {
					shop = &shops[i]
					break
				}
			}
		}
		if shop == nil {
			return &CommandResult{Error: fmt.Errorf("shop not found")}
		}
	}

	items, err := h.shopService.GetShopItems(shop.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	var output strings.Builder
	output.WriteString("╔═══════════════════════════════════════╗\n")
	output.WriteString(fmt.Sprintf("║   %s\n", shop.Name))
	output.WriteString("╚═══════════════════════════════════════╝\n\n")
	output.WriteString(fmt.Sprintf("%s\n\n", shop.Description))
	output.WriteString("Items for sale:\n\n")

	if len(items) == 0 {
		output.WriteString("  No items available.\n")
	} else {
		for i, item := range items {
			output.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, item.Name))
			output.WriteString(fmt.Sprintf("      Type: %s\n", item.ItemType))
			if item.Description != "" {
				output.WriteString(fmt.Sprintf("      %s\n", item.Description))
			}
			if item.PriceCrypto > 0 {
				output.WriteString(fmt.Sprintf("      Price: %.2f crypto", item.PriceCrypto))
			}
			if item.PriceData > 0 {
				if item.PriceCrypto > 0 {
					output.WriteString(" + ")
				} else {
					output.WriteString("      Price: ")
				}
				output.WriteString(fmt.Sprintf("%.2f data", item.PriceData))
			}
			output.WriteString("\n")
			if item.Stock >= 0 {
				output.WriteString(fmt.Sprintf("      Stock: %d\n", item.Stock))
			} else {
				output.WriteString("      Stock: Unlimited\n")
			}
			output.WriteString("\n")
		}
	}

	output.WriteString("Usage: buy <shopID> <itemNumber> - Purchase item\n")

	return &CommandResult{Output: output.String()}
}

// handleBUY handles purchasing items from shops
func (h *CommandHandler) handleBUY(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: buy <shopID> <itemNumber>")}
	}

	if h.shopService == nil {
		return &CommandResult{Error: fmt.Errorf("shop service not available")}
	}

	shopID := args[0]
	itemNum := args[1]

	// Get shop
	shop, err := h.shopService.GetShopByServerIP(shopID)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("shop not found")}
	}

	// Get items
	items, err := h.shopService.GetShopItems(shop.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Parse item number
	var itemIndex int
	if _, err := fmt.Sscanf(itemNum, "%d", &itemIndex); err != nil || itemIndex < 1 || itemIndex > len(items) {
		return &CommandResult{Error: fmt.Errorf("invalid item number")}
	}

	item := items[itemIndex-1]

	// Purchase item
	if err := h.shopService.PurchaseItem(h.user.ID, shop.ID, item.ID); err != nil {
		return &CommandResult{Error: err}
	}

	// Handle item based on type
	output := fmt.Sprintf("Successfully purchased %s from %s\n", item.Name, shop.Name)

	switch item.ItemType {
	case models.ItemTypeTool:
		// Tool needs to be added via tool service
		output += fmt.Sprintf("Tool %s has been added to your inventory. Use 'get %s %s' to download it.\n", item.Name, shop.ServerIP, item.Name)
	case models.ItemTypePatch:
		// Patch needs to be added via patch service
		if h.patchService != nil {
			if err := h.patchService.AddUserPatch(h.user.ID, item.Name); err == nil {
				output += fmt.Sprintf("Patch %s has been added to your inventory. Use 'patch %s <toolName>' to apply it.\n", item.Name, item.Name)
			}
		}
	case models.ItemTypeResource:
		output += "Resource upgrade has been applied.\n"
	}

	return &CommandResult{Output: output}
}

