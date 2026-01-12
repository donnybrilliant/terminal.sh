// Package services provides business logic services for the terminal.sh game.
// This file implements the Server Generator, which automatically creates
// servers when the available server count drops below the minimum threshold,
// ensuring players always have targets to exploit.
package services

import (
	"fmt"
	"math/rand"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

const (
	// MIN_SERVERS_ONLINE is the minimum number of servers to keep available.
	// When server count drops below this, new servers are automatically generated.
	MIN_SERVERS_ONLINE = 10
)

// ServerGenerator handles procedural server generation.
// Automatically creates servers with appropriate difficulty when servers are
// exhausted, ensuring infinite gameplay. Servers are scaled to player level
// and may include local network connections for exploration depth.
type ServerGenerator struct {
	db            *database.Database
	serverService *ServerService
	rng           *rand.Rand
}

// NewServerGenerator creates a new ServerGenerator
func NewServerGenerator(db *database.Database, serverService *ServerService) *ServerGenerator {
	return &ServerGenerator{
		db:            db,
		serverService: serverService,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CheckAndGenerateServers checks if more servers are needed and generates them
func (g *ServerGenerator) CheckAndGenerateServers(userID uuid.UUID) error {
	shouldGenerate, count := g.shouldGenerateServers(userID)
	if !shouldGenerate {
		return nil
	}

	// Get user level for difficulty scaling
	user, err := g.getUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Generate servers
	for i := 0; i < count; i++ {
		server, err := g.generateServerWithDifficulty(user.Level, "")
		if err != nil {
			return fmt.Errorf("failed to generate server: %w", err)
		}

		// Track as procedural server
		proceduralServer := &models.ProceduralServer{
			ServerID:     server.ID,
			GeneratedFor: userID,
			GeneratedAt:  time.Now(),
			Reason:       "exhaustion",
		}
		if err := g.db.Create(proceduralServer).Error; err != nil {
			return fmt.Errorf("failed to track procedural server: %w", err)
		}

		// Optionally create local servers (30% chance)
		if g.rng.Float32() < 0.3 {
			localCount := g.rng.Intn(3) + 1 // 1-3 local servers
			for j := 0; j < localCount; j++ {
				localServer, err := g.generateLocalServerForParent(server.IP, user.Level)
				if err != nil {
					// Log error but continue
					continue
				}

				// Track local server
				localProcedural := &models.ProceduralServer{
					ServerID:     localServer.ID,
					GeneratedFor: userID,
					GeneratedAt:  time.Now(),
					Reason:       "exhaustion",
				}
				g.db.Create(localProcedural)
			}
		}
	}

	return nil
}

// GenerateServerForMission creates a server specifically for a mission
func (g *ServerGenerator) GenerateServerForMission(mission *models.Mission) (*models.Server, error) {
	// Determine server type from mission requirements
	serverType := g.determineServerTypeFromMission(mission)

	// Use mission required level or default
	level := mission.RequiredLevel
	if level == 0 {
		level = 5
	}

	server, err := g.generateServerWithDifficulty(level, serverType)
	if err != nil {
		return nil, err
	}

	// Track as procedural server
	proceduralServer := &models.ProceduralServer{
		ServerID:     server.ID,
		GeneratedFor: uuid.Nil, // Could store mission ID if we had it
		GeneratedAt:  time.Now(),
		Reason:        "mission",
	}
	if err := g.db.Create(proceduralServer).Error; err != nil {
		return nil, fmt.Errorf("failed to track procedural server: %w", err)
	}

	return server, nil
}

// GenerateServerChain creates a chain of connected servers
func (g *ServerGenerator) GenerateServerChain(count int, parentIP string) ([]models.Server, error) {
	var servers []models.Server
	var currentParentIP = parentIP

	for i := 0; i < count; i++ {
		// Generate server with decreasing difficulty
		level := 10 - (i * 2)
		if level < 1 {
			level = 1
		}

		server, err := g.generateServerWithDifficulty(level, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate server in chain: %w", err)
		}

		// Connect to parent if not first server
		if currentParentIP != "" {
			parentServer, err := g.serverService.GetServerByIP(currentParentIP)
			if err == nil {
				// Add to parent's ConnectedIPs
				parentServer.ConnectedIPs = append(parentServer.ConnectedIPs, server.IP)
				g.serverService.db.Save(parentServer)
			}
		}

		servers = append(servers, *server)
		currentParentIP = server.IP
	}

	return servers, nil
}

// shouldGenerateServers checks if generation is needed
func (g *ServerGenerator) shouldGenerateServers(userID uuid.UUID) (bool, int) {
	// Count all top-level servers
	servers, err := g.serverService.GetAllTopLevelServers()
	if err != nil {
		// If we can't get servers, generate some
		return true, MIN_SERVERS_ONLINE
	}

	currentCount := len(servers)
	if currentCount < MIN_SERVERS_ONLINE {
		needed := MIN_SERVERS_ONLINE - currentCount
		// Generate a few extra to ensure we have buffer
		generateCount := needed + 5
		if generateCount > 15 {
			generateCount = 15 // Cap at 15 servers per generation
		}
		return true, generateCount
	}

	return false, 0
}

// generateServerWithDifficulty creates a server with appropriate difficulty
func (g *ServerGenerator) generateServerWithDifficulty(level int, serverType string) (*models.Server, error) {
	// Generate random IPs
	ip := g.generateRandomIP()
	localIP := g.generateRandomLocalIP()

	// Calculate security level based on player level
	baseLevel := 10 + (level * 2)
	securityLevel := baseLevel + g.rng.Intn(20) - 10 // Add randomness
	if securityLevel < 1 {
		securityLevel = 1
	}
	if securityLevel > 100 {
		securityLevel = 100
	}

	// Create server using ServerService
	server, err := g.serverService.CreateServer(ip, localIP)
	if err != nil {
		return nil, err
	}

	// Update security level and services based on difficulty
	server.SecurityLevel = securityLevel
	server.Services = g.generateServicesForDifficulty(level, serverType)

	// Update resources based on level (higher level = better resources)
	server.Resources.CPU = 1000 + (level * 100) + g.rng.Intn(500)
	server.Resources.Bandwidth = float64(5000 + (level * 200) + g.rng.Intn(1000))
	server.Resources.RAM = 256 + (level * 10) + g.rng.Intn(100)

	// Update wallet
	server.Wallet.Crypto = float64(1000 + (level * 50) + g.rng.Intn(500))
	server.Wallet.Data = float64(1000000 + (level * 100000) + g.rng.Intn(1000000))

	if err := g.serverService.db.Save(server).Error; err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	return server, nil
}

// generateLocalServerForParent creates a local server for a parent
func (g *ServerGenerator) generateLocalServerForParent(parentIP string, parentLevel int) (*models.Server, error) {
	// Local servers are easier than parent
	localLevel := parentLevel - (g.rng.Intn(20) + 10)
	if localLevel < 1 {
		localLevel = 1
	}

	ip := g.generateRandomIP()
	localIP := g.generateRandomLocalIP()

	server, err := g.serverService.CreateLocalServer(parentIP, ip, localIP)
	if err != nil {
		return nil, err
	}

	// Update security level (lower than parent)
	server.SecurityLevel = parentLevel - (g.rng.Intn(20) + 10)
	if server.SecurityLevel < 1 {
		server.SecurityLevel = 1
	}

	// Update services
	server.Services = g.generateServicesForDifficulty(localLevel, "")

	if err := g.serverService.db.Save(server).Error; err != nil {
		return nil, fmt.Errorf("failed to update local server: %w", err)
	}

	return server, nil
}

// generateServicesForDifficulty creates services appropriate for difficulty level
func (g *ServerGenerator) generateServicesForDifficulty(level int, serverType string) []models.Service {
	var services []models.Service

	// Determine which services to include
	hasSSH := true
	hasHTTP := false

	if serverType == "http" {
		hasHTTP = true
		hasSSH = g.rng.Float32() < 0.5
	} else if serverType == "ssh" {
		hasSSH = true
		hasHTTP = false
	} else {
		// Random selection
		hasHTTP = g.rng.Float32() < 0.5
	}

	// Generate SSH service
	if hasSSH {
		sshLevel := 10 + (level * 2) + g.rng.Intn(20) - 10
		if sshLevel < 5 {
			sshLevel = 5
		}
		services = append(services, models.Service{
			Name:        "ssh",
			Description: "Secure Shell",
			Port:        22,
			Vulnerable:  true,
			Level:       sshLevel,
			Vulnerabilities: []models.Vulnerability{
				{Type: "remote_code_execution", Level: sshLevel - 5 + g.rng.Intn(10)},
				{Type: "buffer_overflow", Level: sshLevel - 3 + g.rng.Intn(10)},
			},
		})
	}

	// Generate HTTP service
	if hasHTTP {
		httpLevel := 5 + (level * 2) + g.rng.Intn(15) - 5
		if httpLevel < 3 {
			httpLevel = 3
		}
		services = append(services, models.Service{
			Name:        "http",
			Description: "Hypertext Transfer Protocol",
			Port:        80,
			Vulnerable:  g.rng.Float32() < 0.7,
			Level:       httpLevel,
			Vulnerabilities: []models.Vulnerability{
				{Type: "sql_injection", Level: httpLevel - 3 + g.rng.Intn(10)},
				{Type: "xss", Level: httpLevel - 5 + g.rng.Intn(10)},
			},
		})
	}

	return services
}

// determineServerTypeFromMission determines server type needed for mission
func (g *ServerGenerator) determineServerTypeFromMission(mission *models.Mission) string {
	// Check required tools to determine server type
	for _, tool := range mission.RequiredTools {
		if tool == "sql_injector" || tool == "database_dumper" {
			return "http"
		}
		if tool == "password_cracker" || tool == "ssh_exploit" {
			return "ssh"
		}
	}

	// Check objectives
	for _, obj := range mission.Objectives {
		if obj.Tool == "sql_injector" || obj.Tool == "database_dumper" {
			return "http"
		}
		if obj.Tool == "password_cracker" || obj.Tool == "ssh_exploit" {
			return "ssh"
		}
	}

	return "" // No specific type
}

// Helper functions

func (g *ServerGenerator) generateRandomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		g.rng.Intn(254)+1,
		g.rng.Intn(255),
		g.rng.Intn(255),
		g.rng.Intn(254)+1)
}

func (g *ServerGenerator) generateRandomLocalIP() string {
	return fmt.Sprintf("10.%d.%d.%d",
		g.rng.Intn(255),
		g.rng.Intn(255),
		g.rng.Intn(254)+1)
}

func (g *ServerGenerator) getUser(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := g.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
