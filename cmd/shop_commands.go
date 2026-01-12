package cmd

import (
	"fmt"
	"strings"

	"terminal-sh/models"

	"github.com/charmbracelet/lipgloss"
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
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(headerStyle.Render("🛒 Discovered Shops:") + "\n\n")
	
	listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	shopNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true) // Magenta
	
	for _, shop := range shops {
		output.WriteString(listStyle.Render(fmt.Sprintf("  [%s] ", shop.ID.String()[:8])) + shopNameStyle.Render(shop.Name) + "\n")
		output.WriteString(labelStyle.Render("    Type:") + " " + valueStyle.Render(string(shop.ShopType)) + "\n")
		output.WriteString(labelStyle.Render("    Location:") + " " + formatIP(shop.ServerIP) + "\n")
		output.WriteString("    " + valueStyle.Render(shop.Description) + "\n\n")
	}

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	output.WriteString(infoStyle.Render("Usage: shop <shopID> - Browse shop inventory\n"))
	output.WriteString(infoStyle.Render("       buy <shopID> <itemID> - Purchase item\n"))

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
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	shopNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true) // Magenta
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	
	output.WriteString("╔═══════════════════════════════════════╗\n")
	output.WriteString("║   " + shopNameStyle.Render(shop.Name) + "\n")
	output.WriteString("╚═══════════════════════════════════════╝\n\n")
	output.WriteString(valueStyle.Render(shop.Description) + "\n\n")
	output.WriteString(headerStyle.Render("🛍️ Items for sale:") + "\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	itemNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
	priceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green
	listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
	
	if len(items) == 0 {
		output.WriteString("  No items available.\n")
	} else {
		for i, item := range items {
			output.WriteString(listStyle.Render(fmt.Sprintf("  [%d] ", i+1)) + itemNameStyle.Render(item.Name) + "\n")
			output.WriteString(labelStyle.Render("      Type:") + " " + valueStyle.Render(string(item.ItemType)) + "\n")
			if item.Description != "" {
				output.WriteString("      " + valueStyle.Render(item.Description) + "\n")
			}
			priceStr := labelStyle.Render("      Price: ")
			if item.PriceCrypto > 0 {
				priceStr += priceStyle.Render(fmt.Sprintf("%.2f crypto", item.PriceCrypto))
			}
			if item.PriceData > 0 {
				if item.PriceCrypto > 0 {
					priceStr += " + "
				}
				priceStr += priceStyle.Render(fmt.Sprintf("%.2f data", item.PriceData))
			}
			output.WriteString(priceStr + "\n")
			stockStr := labelStyle.Render("      Stock: ")
			if item.Stock >= 0 {
				stockStr += valueStyle.Render(fmt.Sprintf("%d", item.Stock))
			} else {
				stockStr += valueStyle.Render("Unlimited")
			}
			output.WriteString(stockStr + "\n\n")
		}
	}

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	output.WriteString(infoStyle.Render("Usage: buy <shopID> <itemNumber> - Purchase item\n"))

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
	var output strings.Builder
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true)
	itemNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
	
	output.WriteString(successStyle.Render("✅ Successfully purchased ") + itemNameStyle.Render(item.Name) + successStyle.Render(fmt.Sprintf(" from %s", shop.Name)) + "\n")

	switch item.ItemType {
	case models.ItemTypeTool:
		// Tool needs to be added via tool service
		output.WriteString("Tool " + itemNameStyle.Render(item.Name) + " has been added to your inventory. Use 'get " + formatIP(shop.ServerIP) + " " + item.Name + "' to download it.\n")
	case models.ItemTypePatch:
		// Patch needs to be added via patch service
		if h.patchService != nil {
			if err := h.patchService.AddUserPatch(h.user.ID, item.Name); err == nil {
				output.WriteString("Patch " + itemNameStyle.Render(item.Name) + " has been added to your inventory. Use 'patch " + item.Name + " <toolName>' to apply it.\n")
			}
		}
	case models.ItemTypeResource:
		output.WriteString(successStyle.Render("Resource upgrade has been applied.") + "\n")
	}

	return &CommandResult{Output: output.String()}
}

