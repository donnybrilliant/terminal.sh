package services

import (
	"fmt"
	"terminal-sh/database"
	"terminal-sh/models"
	"gorm.io/gorm"
)

// SeedInitialData seeds the database with initial game data
func SeedInitialData() error {
	// Seed tools (already done in main.go, but we can add server seeding here)
	
	// Create "repo" server with all tools
	repoServer, err := createRepoServer()
	if err != nil {
		return fmt.Errorf("failed to create repo server: %w", err)
	}
	// Note: repoServer will be nil if it already exists, which is fine
	_ = repoServer
	
	// Create test server for easy connection testing
	testServer, err := createTestServer()
	if err != nil {
		return fmt.Errorf("failed to create test server: %w", err)
	}
	// Note: testServer will be nil if it already exists, which is fine
	_ = testServer
	
	// Seed patches
	patchService := NewPatchService(NewToolService(NewServerService()))
	if err := patchService.SeedPatches(); err != nil {
		return fmt.Errorf("failed to seed patches: %w", err)
	}
	
	// Seed shops
	if err := seedShops(); err != nil {
		return fmt.Errorf("failed to seed shops: %w", err)
	}
	
	return nil
}

func createRepoServer() (*models.Server, error) {
	// Check if repo server already exists
	var existing models.Server
	err := database.DB.Where("ip = ?", "repo").First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Get all tools
	var tools []models.Tool
	if err := database.DB.Find(&tools).Error; err != nil {
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
	
	if err := database.DB.Create(repoServer).Error; err != nil {
		return nil, fmt.Errorf("failed to create repo server: %w", err)
	}
	
	return repoServer, nil
}

func createTestServer() (*models.Server, error) {
	// Check if test server already exists
	var existing models.Server
	err := database.DB.Where("ip = ?", "test").First(&existing).Error
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
	
	if err := database.DB.Create(testServer).Error; err != nil {
		return nil, fmt.Errorf("failed to create test server: %w", err)
	}
	
	return testServer, nil
}

// seedShops seeds initial shops
func seedShops() error {
	serverService := NewServerService()
	shopService := NewShopService(serverService)
	
	// Create tool shop on a new server
	toolShopServerIP := "1.1.1.1"
	
	// Check if shop already exists
	var existingShop models.Shop
	if err := database.DB.Where("server_ip = ?", toolShopServerIP).First(&existingShop).Error; err == nil {
		return nil // Shop already exists
	}
	
	// Create shop server if it doesn't exist
	var shopServer models.Server
	if err := database.DB.Where("ip = ?", toolShopServerIP).First(&shopServer).Error; err != nil {
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
		if err := database.DB.Create(&shopServer).Error; err != nil {
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
	if err := database.DB.Where("ip = ?", resourceShopServerIP).First(&resourceShopServer).Error; err != nil {
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
		if err := database.DB.Create(&resourceShopServer).Error; err != nil {
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
	if err := seedPatchFiles(); err != nil {
		return fmt.Errorf("failed to seed patch files: %w", err)
	}
	
	return nil
}

// seedPatchFiles creates patch metadata files on various servers
func seedPatchFiles() error {
	// Create a patch file on test server
	var testServer models.Server
	if err := database.DB.Where("ip = ?", "test").First(&testServer).Error; err == nil {
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
		if err := database.DB.Save(&testServer).Error; err != nil {
			return fmt.Errorf("failed to save patch file: %w", err)
		}
	}
	
	// Create patch file on shop server
	var shopServer models.Server
	if err := database.DB.Where("ip = ?", "1.1.1.1").First(&shopServer).Error; err == nil {
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
		if err := database.DB.Save(&shopServer).Error; err != nil {
			return fmt.Errorf("failed to save patch file: %w", err)
		}
	}
	
	return nil
}

