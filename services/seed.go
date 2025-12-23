package services

import (
	"fmt"
	"ssh4xx-go/database"
	"ssh4xx-go/models"
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

