package services

import (
	"fmt"
	"math/rand"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"
)

// ServerService handles server-related operations
type ServerService struct {
	rng *rand.Rand
}

// NewServerService creates a new server service
func NewServerService() *ServerService {
	return &ServerService{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetServerByIP retrieves a server by its IP address (checks both IP and LocalIP fields)
func (s *ServerService) GetServerByIP(ip string) (*models.Server, error) {
	var server models.Server
	// Try IP field first (hostname)
	if err := database.DB.Where("ip = ?", ip).First(&server).Error; err == nil {
		return &server, nil
	}
	// Try LocalIP field (actual IP address)
	if err := database.DB.Where("local_ip = ?", ip).First(&server).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// GetServerByPath retrieves a server by its path (supports nested paths)
func (s *ServerService) GetServerByPath(path string) (*models.Server, error) {
	// Parse path: "1.1.1.1.localNetwork.10.0.0.5"
	parts := parseServerPath(path)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid server path")
	}

	// Get root server
	server, err := s.GetServerByIP(parts[0])
	if err != nil {
		return nil, err
	}

	// Navigate through nested localNetwork
	for i := 1; i < len(parts); i++ {
		// Navigate through localNetwork structure
		// This is a simplified version - in reality, we'd need to parse the JSON structure
		// For now, we'll search for servers with matching IPs in the localNetwork
		nestedIP := parts[i]
		found := false
		
		// Check if this IP exists in the localNetwork
		// We'll need to implement proper JSON traversal
		// For now, search for a server with this IP
		nestedServer, err := s.GetServerByIP(nestedIP)
		if err == nil {
			server = nestedServer
			found = true
		}
		
		if !found {
			return nil, fmt.Errorf("server not found at path: %s", path)
		}
	}

	return server, nil
}

// GetAllTopLevelServers retrieves all top-level servers (not in localNetwork)
func (s *ServerService) GetAllTopLevelServers() ([]models.Server, error) {
	var servers []models.Server
	// For now, return all servers - we'll filter by checking if they're referenced in localNetwork later
	if err := database.DB.Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetConnectedServers retrieves servers connected to a given server
func (s *ServerService) GetConnectedServers(serverIP string) ([]models.Server, error) {
	server, err := s.GetServerByIP(serverIP)
	if err != nil {
		return nil, err
	}

	var connectedServers []models.Server
	for _, ip := range server.ConnectedIPs {
		connectedServer, err := s.GetServerByIP(ip)
		if err == nil {
			connectedServers = append(connectedServers, *connectedServer)
		}
	}

	return connectedServers, nil
}

// CreateServer creates a new server with randomized properties
func (s *ServerService) CreateServer(ip, localIP string) (*models.Server, error) {
	// Check if server already exists
	var existing models.Server
	if err := database.DB.Where("ip = ?", ip).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("server with IP %s already exists", ip)
	}

	server := &models.Server{
		IP:            ip,
		LocalIP:       localIP,
		SecurityLevel: s.rng.Intn(100) + 1, // 1-100
		Resources: models.ServerResources{
			CPU:      s.rng.Intn(10000) + 1000,  // 1000-11000
			Bandwidth: float64(s.rng.Intn(50000) + 5000), // 5000-55000
			RAM:      s.rng.Intn(1048) + 256,   // 256-1304
		},
		Wallet: models.ServerWallet{
			Crypto: float64(s.rng.Intn(10000) + 1000), // 1000-11000
			Data:   float64(s.rng.Intn(50000000) + 1000000), // 1000000-51000000
		},
		Tools:        []string{},
		ConnectedIPs: []string{},
		Services:    generateRandomServices(s.rng),
		Roles:        []models.Role{{Role: "admin", Level: 100}},
		FileSystem:   make(map[string]interface{}),
		LocalNetwork: make(map[string]interface{}),
	}

	if err := database.DB.Create(server).Error; err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return server, nil
}

// CreateLocalServer creates a server within another server's localNetwork
func (s *ServerService) CreateLocalServer(parentServerIP, ip, localIP string) (*models.Server, error) {
	// Create the server first
	server, err := s.CreateServer(ip, localIP)
	if err != nil {
		return nil, err
	}

	// Add to parent's localNetwork
	parentServer, err := s.GetServerByIP(parentServerIP)
	if err != nil {
		return nil, fmt.Errorf("parent server not found: %w", err)
	}

	// Update parent's localNetwork (simplified - would need proper JSON manipulation)
	if parentServer.LocalNetwork == nil {
		parentServer.LocalNetwork = make(map[string]interface{})
	}
	parentServer.LocalNetwork[ip] = server.ID.String()

	if err := database.DB.Save(parentServer).Error; err != nil {
		return nil, fmt.Errorf("failed to update parent server: %w", err)
	}

	return server, nil
}

// Helper functions

func parseServerPath(path string) []string {
	parts := []string{}
	current := ""
	
	for _, char := range path {
		if char == '.' {
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
	
	return parts
}

func generateRandomServices(rng *rand.Rand) []models.Service {
	services := []models.Service{
		{
			Name:        "ssh",
			Description: "Secure Shell",
			Port:        22,
			Vulnerable:  true,
			Level:       rng.Intn(50) + 10, // 10-60
			Vulnerabilities: []models.Vulnerability{
				{Type: "remote_code_execution", Level: rng.Intn(30) + 10},
				{Type: "buffer_overflow", Level: rng.Intn(40) + 20},
			},
		},
	}
	
	// Randomly add more services
	if rng.Float32() < 0.5 {
		services = append(services, models.Service{
			Name:        "http",
			Description: "Hypertext Transfer Protocol",
			Port:        80,
			Vulnerable:  rng.Float32() < 0.7,
			Level:       rng.Intn(40) + 5,
			Vulnerabilities: []models.Vulnerability{
				{Type: "sql_injection", Level: rng.Intn(30) + 10},
				{Type: "xss", Level: rng.Intn(25) + 5},
			},
		})
	}
	
	return services
}

