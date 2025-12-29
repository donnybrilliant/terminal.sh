package services

import (
	"encoding/json"
	"fmt"
	"os"
	"terminal-sh/database"
	"terminal-sh/models"

	"gorm.io/gorm"
)

// SeedInitialData seeds the database with initial game data from JSON files (servers, shops, patches).
// This should be called during application initialization.
func SeedInitialData(db *database.Database) error {
	// Seed servers from JSON
	if err := seedServersFromJSON(db); err != nil {
		return fmt.Errorf("failed to seed servers: %w", err)
	}
	
	// Seed patches
	serverService := NewServerService(db)
	toolService := NewToolService(db, serverService)
	patchService := NewPatchService(db, toolService)
	if err := patchService.SeedPatches(); err != nil {
		return fmt.Errorf("failed to seed patches: %w", err)
	}
	
	// Seed shops from JSON
	if err := seedShopsFromJSON(db); err != nil {
		return fmt.Errorf("failed to seed shops: %w", err)
	}
	
	return nil
}

// seedServersFromJSON loads and seeds servers from JSON file
func seedServersFromJSON(db *database.Database) error {
	data, err := os.ReadFile("data/seed/servers.json")
	if err != nil {
		return fmt.Errorf("failed to read servers.json: %w", err)
	}

	var serverData struct {
		Servers []models.Server `json:"servers"`
	}
	if err := json.Unmarshal(data, &serverData); err != nil {
		return fmt.Errorf("failed to parse servers.json: %w", err)
	}

	for _, server := range serverData.Servers {
		// Check if server already exists
		var existing models.Server
		err := db.Where("ip = ?", server.IP).First(&existing).Error
		if err != nil && err == gorm.ErrRecordNotFound {
			// For repo server, populate tools list with all available tools
			if server.IP == "repo" {
				var tools []models.Tool
				if err := db.Find(&tools).Error; err == nil {
					toolNames := make([]string, len(tools))
					for i, tool := range tools {
						toolNames[i] = tool.Name
					}
					server.Tools = toolNames
				}
			}
			
			if err := db.Create(&server).Error; err != nil {
				return fmt.Errorf("failed to seed server %s: %w", server.IP, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check server %s: %w", server.IP, err)
		}
		// If server exists, skip it
	}

	return nil
}

// Deprecated: Use seedServersFromJSON instead
func createRepoServer(db *database.Database) (*models.Server, error) {
	// Check if repo server already exists
	var existing models.Server
	err := db.Where("ip = ?", "repo").First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Get all tools
	var tools []models.Tool
	if err := db.Find(&tools).Error; err != nil {
		return nil, err
	}
	
	// Create tool names list
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}
	
	// Create repo server
	repoServer := &models.Server{
		IP:            "repo",
		LocalIP:       "10.0.0.1",
		SecurityLevel: 200, // High security
		Resources: models.ServerResources{
			CPU:      50000,
			Bandwidth: 100000,
			RAM:      2048,
		},
		Wallet: models.ServerWallet{
			Crypto: 0,
			Data:   0,
		},
		Tools:        toolNames,
		ConnectedIPs: []string{},
		Services: []models.Service{
			{
				Name:        "ssh",
				Description: "Secure Shell",
				Port:        22,
				Vulnerable:  false, // Repo is secure
				Level:       200,
				Vulnerabilities: []models.Vulnerability{},
			},
		},
		Roles: []models.Role{
			{Role: "admin", Level: 200},
		},
		FileSystem:   make(map[string]interface{}),
		LocalNetwork: make(map[string]interface{}),
	}
	
	if err := db.Create(repoServer).Error; err != nil {
		return nil, fmt.Errorf("failed to create repo server: %w", err)
	}
	
	return repoServer, nil
}

func createTestServer(db *database.Database) (*models.Server, error) {
	// Check if test server already exists
	var existing models.Server
	err := db.Where("ip = ?", "test").First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Create test server with low security for easy exploitation
	testServer := &models.Server{
		IP:            "test",
		LocalIP:       "192.168.1.100",
		SecurityLevel: 15, // Low security - easy to exploit
		Resources: models.ServerResources{
			CPU:      5000,
			Bandwidth: 10000,
			RAM:      512,
		},
		Wallet: models.ServerWallet{
			Crypto: 100,
			Data:   1000,
		},
		Tools:        []string{"password_cracker", "ssh_exploit"}, // Some tools for testing
		ConnectedIPs: []string{},
		Services: []models.Service{
			{
				Name:        "ssh",
				Description: "Secure Shell",
				Port:        22,
				Vulnerable:  true, // Vulnerable for easy exploitation
				Level:       10,
				Vulnerabilities: []models.Vulnerability{
					{Type: "weak_password", Level: 5},
					{Type: "ssh_vulnerability", Level: 10},
				},
			},
		},
		Roles: []models.Role{
			{Role: "user", Level: 5},
			{Role: "admin", Level: 10},
		},
		FileSystem:   make(map[string]interface{}),
		LocalNetwork: make(map[string]interface{}),
	}
	
	if err := db.Create(testServer).Error; err != nil {
		return nil, fmt.Errorf("failed to create test server: %w", err)
	}
	
	return testServer, nil
}

// seedShopsFromJSON loads and seeds shops from JSON file
func seedShopsFromJSON(db *database.Database) error {
	// Load shops from JSON file
	data, err := os.ReadFile("data/seed/shops.json")
	if err != nil {
		return fmt.Errorf("failed to read shops.json: %w", err)
	}

	var shopData struct {
		Shops      []struct {
			ServerIP        string `json:"server_ip"`
			Server          models.Server `json:"server"`
			ShopType        string `json:"shop_type"`
			ShopName        string `json:"shop_name"`
			ShopDescription string `json:"shop_description"`
			Items           []struct {
				ItemType    string `json:"item_type"`
				ItemID      string `json:"item_id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				CryptoPrice  float64 `json:"crypto_price"`
				DataPrice    float64 `json:"data_price"`
				Stock        int     `json:"stock"`
			} `json:"items"`
		} `json:"shops"`
		PatchFiles []struct {
			ServerIP string `json:"server_ip"`
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		} `json:"patch_files"`
	}
	if err := json.Unmarshal(data, &shopData); err != nil {
		return fmt.Errorf("failed to parse shops.json: %w", err)
	}

	serverService := NewServerService(db)
	shopService := NewShopService(db, serverService)

	// Seed shops
	for _, shopDef := range shopData.Shops {
		// Check if shop already exists
		var existingShop models.Shop
		if err := db.Where("server_ip = ?", shopDef.ServerIP).First(&existingShop).Error; err == nil {
			continue // Shop already exists
		}

		// Create shop server if it doesn't exist
		var shopServer models.Server
		if err := db.Where("ip = ?", shopDef.ServerIP).First(&shopServer).Error; err != nil {
			// Create shop server from JSON
			shopServer = shopDef.Server
			if err := db.Create(&shopServer).Error; err != nil {
				return fmt.Errorf("failed to create shop server %s: %w", shopDef.ServerIP, err)
			}
		}

		// Determine shop type
		var shopType models.ShopType
		switch shopDef.ShopType {
		case "tools":
			shopType = models.ShopTypeTools
		case "resources":
			shopType = models.ShopTypeResources
		default:
			shopType = models.ShopTypeTools
		}

		// Create shop
		shop, err := shopService.CreateShop(shopDef.ServerIP, shopType, shopDef.ShopName, shopDef.ShopDescription)
		if err != nil {
			return fmt.Errorf("failed to create shop: %w", err)
		}

		// Add shop items
		for _, item := range shopDef.Items {
			var itemType models.ItemType
			switch item.ItemType {
			case "tool":
				itemType = models.ItemTypeTool
			case "patch":
				itemType = models.ItemTypePatch
			case "resource":
				itemType = models.ItemTypeResource
			default:
				itemType = models.ItemTypeTool
			}

			_, err = shopService.AddShopItem(shop.ID, itemType, item.Name, item.Description, item.CryptoPrice, item.DataPrice, item.Stock)
			if err != nil {
				return fmt.Errorf("failed to add shop item %s: %w", item.Name, err)
			}
		}
	}

	// Seed patch files on servers
	for _, patchFile := range shopData.PatchFiles {
		var server models.Server
		if err := db.Where("ip = ?", patchFile.ServerIP).First(&server).Error; err != nil {
			continue // Server doesn't exist, skip
		}

		if server.FileSystem == nil {
			server.FileSystem = make(map[string]interface{})
		}

		// Parse file path to create directory structure
		parts := []string{}
		current := ""
		for _, char := range patchFile.FilePath {
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

		// Create directory structure
		currentLevel := server.FileSystem
		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part is the file
				currentLevel[part] = map[string]interface{}{
					"content": patchFile.Content,
				}
			} else {
				// Directory
				if _, ok := currentLevel[part].(map[string]interface{}); !ok {
					currentLevel[part] = make(map[string]interface{})
				}
				currentLevel = currentLevel[part].(map[string]interface{})
			}
		}

		if err := db.Save(&server).Error; err != nil {
			return fmt.Errorf("failed to save patch file on server %s: %w", patchFile.ServerIP, err)
		}
	}

	return nil
}

// Deprecated: Use seedShopsFromJSON instead
func seedShops(db *database.Database) error {
	serverService := NewServerService(db)
	shopService := NewShopService(db, serverService)
	
	// Create tool shop on a new server
	toolShopServerIP := "1.1.1.1"
	
	// Check if shop already exists
	var existingShop models.Shop
	if err := db.Where("server_ip = ?", toolShopServerIP).First(&existingShop).Error; err == nil {
		return nil // Shop already exists
	}
	
	// Create shop server if it doesn't exist
	var shopServer models.Server
	if err := db.Where("ip = ?", toolShopServerIP).First(&shopServer).Error; err != nil {
		// Create shop server
		shopServer = models.Server{
			IP:            toolShopServerIP,
			LocalIP:       "10.0.0.10",
			SecurityLevel: 50,
			Resources: models.ServerResources{
				CPU:      10000,
				Bandwidth: 20000,
				RAM:      1024,
			},
			Wallet: models.ServerWallet{
				Crypto: 5000,
				Data:   50000,
			},
			Tools:        []string{},
			ConnectedIPs: []string{},
			Services: []models.Service{
				{
					Name:        "ssh",
					Description: "Secure Shell",
					Port:        22,
					Vulnerable:  true,
					Level:       20,
					Vulnerabilities: []models.Vulnerability{
						{Type: "remote_code_execution", Level: 15},
					},
				},
			},
			Roles: []models.Role{
				{Role: "admin", Level: 20},
			},
			FileSystem:   make(map[string]interface{}),
			LocalNetwork: make(map[string]interface{}),
		}
		if err := db.Create(&shopServer).Error; err != nil {
			return fmt.Errorf("failed to create shop server: %w", err)
		}
	}
	
	// Create tool shop
	shop, err := shopService.CreateShop(toolShopServerIP, models.ShopTypeTools, "Elite Tools Shop", "Premium tools for professional hackers")
	if err != nil {
		return fmt.Errorf("failed to create shop: %w", err)
	}
	
	// Add shop items
	_, err = shopService.AddShopItem(shop.ID, models.ItemTypeTool, "advanced_exploit_kit", "Advanced multi-vulnerability exploitation tool", 1000, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	_, err = shopService.AddShopItem(shop.ID, models.ItemTypePatch, "pass_patch_v2", "Enhanced password cracking algorithm", 0, 500, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	_, err = shopService.AddShopItem(shop.ID, models.ItemTypePatch, "ssh_patch_v2", "Improved SSH exploitation techniques", 0, 750, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	_, err = shopService.AddShopItem(shop.ID, models.ItemTypeResource, "cpu_boost", "Increase CPU by 50", 500, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	// Create resource shop on another server
	resourceShopServerIP := "2.2.2.2"
	
	var resourceShopServer models.Server
	if err := db.Where("ip = ?", resourceShopServerIP).First(&resourceShopServer).Error; err != nil {
		resourceShopServer = models.Server{
			IP:            resourceShopServerIP,
			LocalIP:       "10.0.0.20",
			SecurityLevel: 30,
			Resources: models.ServerResources{
				CPU:      8000,
				Bandwidth: 15000,
				RAM:      512,
			},
			Wallet: models.ServerWallet{
				Crypto: 2000,
				Data:   20000,
			},
			Tools:        []string{},
			ConnectedIPs: []string{},
			Services: []models.Service{
				{
					Name:        "ssh",
					Description: "Secure Shell",
					Port:        22,
					Vulnerable:  true,
					Level:       15,
					Vulnerabilities: []models.Vulnerability{
						{Type: "password_cracking", Level: 10},
					},
				},
			},
			Roles: []models.Role{
				{Role: "admin", Level: 15},
			},
			FileSystem:   make(map[string]interface{}),
			LocalNetwork: make(map[string]interface{}),
		}
		if err := db.Create(&resourceShopServer).Error; err != nil {
			return fmt.Errorf("failed to create resource shop server: %w", err)
		}
	}
	
	resourceShop, err := shopService.CreateShop(resourceShopServerIP, models.ShopTypeResources, "Resource Boost Shop", "Upgrade your computational resources")
	if err != nil {
		return fmt.Errorf("failed to create resource shop: %w", err)
	}
	
	_, err = shopService.AddShopItem(resourceShop.ID, models.ItemTypeResource, "bandwidth_boost", "Increase Bandwidth by 50", 300, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	_, err = shopService.AddShopItem(resourceShop.ID, models.ItemTypeResource, "ram_boost", "Increase RAM by 8", 400, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	_, err = shopService.AddShopItem(resourceShop.ID, models.ItemTypeResource, "full_boost", "Increase all resources significantly", 2000, 1000, -1)
	if err != nil {
		return fmt.Errorf("failed to add shop item: %w", err)
	}
	
	// Seed patch files on servers
	if err := seedPatchFiles(db); err != nil {
		return fmt.Errorf("failed to seed patch files: %w", err)
	}
	
	return nil
}

// seedPatchFiles creates patch metadata files on various servers
func seedPatchFiles(db *database.Database) error {
	// Create a patch file on test server
	var testServer models.Server
	if err := db.Where("ip = ?", "test").First(&testServer).Error; err == nil {
		// Create patches directory in filesystem
		if testServer.FileSystem == nil {
			testServer.FileSystem = make(map[string]interface{})
		}
		
		patchesDir := map[string]interface{}{
			"ssh_patch_v2.json": map[string]interface{}{
				"content": `{
  "name": "ssh_patch_v2",
  "target_tool": "ssh_exploit",
  "description": "Improved SSH exploitation techniques found on this server",
  "upgrades": {
    "exploits": [{"type": "remote_code_execution", "level": 30}],
    "resources": {"cpu": -3, "bandwidth": -0.1, "ram": -2}
  }
}`,
			},
		}
		
		testServer.FileSystem["patches"] = patchesDir
		if err := db.Save(&testServer).Error; err != nil {
			return fmt.Errorf("failed to save patch file: %w", err)
		}
	}
	
	// Create patch file on shop server
	var shopServer models.Server
	if err := db.Where("ip = ?", "1.1.1.1").First(&shopServer).Error; err == nil {
		if shopServer.FileSystem == nil {
			shopServer.FileSystem = make(map[string]interface{})
		}
		
		patchesDir := map[string]interface{}{
			"exploit_kit_advanced.json": map[string]interface{}{
				"content": `{
  "name": "exploit_kit_advanced",
  "target_tool": "exploit_kit",
  "description": "Advanced multi-vulnerability exploitation patch",
  "upgrades": {
    "exploits": [
      {"type": "remote_code_execution", "level": 20},
      {"type": "sql_injection", "level": 20},
      {"type": "xss", "level": 15}
    ],
    "resources": {"cpu": -5, "bandwidth": -0.2, "ram": -3}
  }
}`,
			},
		}
		
		shopServer.FileSystem["patches"] = patchesDir
		if err := db.Save(&shopServer).Error; err != nil {
			return fmt.Errorf("failed to save patch file: %w", err)
		}
	}
	
	return nil
}

