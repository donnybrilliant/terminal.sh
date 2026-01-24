package cmd

import (
	"fmt"
	"strings"

	"terminal-sh/models"
	"terminal-sh/patch"
	"terminal-sh/ui"

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
	output.WriteString(ui.FormatSectionHeader("Discovered Shops:", "🛒"))
	
	for _, shop := range shops {
		output.WriteString(ui.ListStyle.Render(fmt.Sprintf("  [%s] ", shop.ID.String()[:8])) + ui.AccentBoldStyle.Render(shop.Name) + "\n")
		output.WriteString(ui.FormatKeyValuePair("    Type:", string(shop.ShopType)) + "\n")
		output.WriteString(ui.FormatKeyValuePair("    Location:", formatIP(shop.ServerIP)) + "\n")
		output.WriteString("    " + ui.ValueStyle.Render(shop.Description) + "\n\n")
	}

	output.WriteString(ui.FormatUsage("Usage: shop <shopID> - Browse shop inventory"))
	output.WriteString(ui.FormatUsage("       buy <shopID> <itemID> - Purchase item"))

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
	output.WriteString("║   " + ui.AccentBoldStyle.Render(shop.Name) + "\n")
	output.WriteString("╚═══════════════════════════════════════╝\n\n")
	output.WriteString(ui.ValueStyle.Render(shop.Description) + "\n\n")
	output.WriteString(ui.FormatSectionHeader("Items for sale:", "🛍️"))
	
	if len(items) == 0 {
		output.WriteString("  No items available.\n")
	} else {
		for i, item := range items {
			output.WriteString(ui.ListStyle.Render(fmt.Sprintf("  [%d] ", i+1)) + ui.AccentStyle.Render(item.Name) + "\n")
			output.WriteString(ui.FormatKeyValuePair("      Type:", string(item.ItemType)) + "\n")
			if item.Description != "" {
				output.WriteString("      " + ui.ValueStyle.Render(item.Description) + "\n")
			}
			priceStr := ui.LabelStyle.Render("      Price: ")
			if item.PriceCrypto > 0 {
				priceStr += ui.PriceStyle.Render(fmt.Sprintf("%.2f crypto", item.PriceCrypto))
			}
			if item.PriceData > 0 {
				if item.PriceCrypto > 0 {
					priceStr += " + "
				}
				priceStr += ui.PriceStyle.Render(fmt.Sprintf("%.2f data", item.PriceData))
			}
			output.WriteString(priceStr + "\n")
			stockStr := ui.LabelStyle.Render("      Stock: ")
			if item.Stock >= 0 {
				stockStr += ui.ValueStyle.Render(fmt.Sprintf("%d", item.Stock))
			} else {
				stockStr += ui.ValueStyle.Render("Unlimited")
			}
			output.WriteString(stockStr + "\n\n")
		}
	}

	output.WriteString(ui.FormatUsage("Usage: buy <shopID> <itemNumber> - Purchase item"))

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
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully purchased ") + ui.AccentStyle.Render(item.Name) + ui.SuccessStyle.Render(fmt.Sprintf(" from %s", shop.Name)) + "\n")

	switch item.ItemType {
	case models.ItemTypeTool:
		// Tool needs to be added via tool service
		output.WriteString("Tool " + ui.AccentStyle.Render(item.Name) + " has been added to your inventory. Use 'get " + formatIP(shop.ServerIP) + " " + item.Name + "' to download it.\n")
	case models.ItemTypeUpgradeToken:
		// Upgrade token - user needs to select a tool to apply it to
		output.WriteString("Upgrade token " + ui.AccentStyle.Render(item.Name) + " purchased!\n")
		output.WriteString("Use 'patch <toolName> " + h.getUpgradeTypeFromItemName(item.Name) + "' to apply this upgrade to a tool.\n")
	case models.ItemTypeResource:
		output.WriteString(ui.SuccessStyle.Render("Resource upgrade has been applied.") + "\n")
	}

	return &CommandResult{Output: output.String()}
}

// getUpgradeTypeFromItemName extracts the upgrade type from an upgrade token item name
func (h *CommandHandler) getUpgradeTypeFromItemName(itemName string) string {
	nameLower := strings.ToLower(itemName)
	if strings.Contains(nameLower, "exploit") {
		return "exploit"
	}
	if strings.Contains(nameLower, "cpu") {
		return "cpu"
	}
	if strings.Contains(nameLower, "ram") {
		return "ram"
	}
	if strings.Contains(nameLower, "bandwidth") || strings.Contains(nameLower, "bw") {
		return "bw"
	}
	if strings.Contains(nameLower, "full") || strings.Contains(nameLower, "tune") {
		return "full"
	}
	return "exploit" // default
}

// handleBuyUpgradeToken handles purchasing upgrade tokens from shops
func (h *CommandHandler) handleBuyUpgradeToken(args []string) *CommandResult {
	if len(args) < 3 {
		return &CommandResult{Error: fmt.Errorf("usage: buy <shopID> <itemNumber> <toolName>")}
	}

	shopID := args[0]
	itemNum := args[1]
	toolName := args[2]

	// Check if user owns the tool
	if !h.toolService.UserHasTool(h.user.ID, toolName) {
		return &CommandResult{Error: fmt.Errorf("you don't own tool %s", toolName)}
	}

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

	// Verify it's an upgrade token
	if item.ItemType != models.ItemTypeUpgradeToken {
		return &CommandResult{Error: fmt.Errorf("item is not an upgrade token")}
	}

	// Purchase item
	if err := h.shopService.PurchaseItem(h.user.ID, shop.ID, item.ID); err != nil {
		return &CommandResult{Error: err}
	}

	// Determine upgrade type from item name and apply it
	upgradeTypeStr := h.getUpgradeTypeFromItemName(item.Name)
	upgradeType, _ := patch.ParseUpgradeType(upgradeTypeStr)

	// Apply the upgrade for free (already paid for it)
	if err := h.upgradeService.ApplyFreeUpgrade(h.user.ID, toolName, upgradeType); err != nil {
		return &CommandResult{Error: fmt.Errorf("failed to apply upgrade: %w", err)}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully purchased and applied ") + ui.AccentStyle.Render(item.Name) + ui.SuccessStyle.Render(" to "+toolName) + "\n")

	return &CommandResult{Output: output.String()}
}

