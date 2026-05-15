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

// Server difficulty tiers scale with player level (1-10 maps to tiers 1-10).
// Each tier defines a range of security levels, vulnerability levels, and rewards.
//
// Tier | Player Level | Security Range | Typical Vulnerabilities | Services
// -----|--------------|----------------|------------------------|----------
//  1   | 1            | 1-10           | password_cracking 1-5  | SSH, Telnet
//  2   | 2            | 10-20          | password_cracking 5-10 | SSH, Telnet, FTP
//  3   | 3            | 20-30          | RCE 10-15, SQL 10-15   | SSH, FTP, HTTP
//  4   | 4            | 25-35          | RCE 15-20, XSS 10-15   | SSH, HTTP
//  5   | 5            | 30-40          | RCE 20-25, SQL 15-20   | SSH, HTTP
//  6   | 6            | 35-50          | RCE 25-30, all types   | All services
//  7   | 7            | 45-60          | RCE 30-35, buffer_overflow | All services
//  8   | 8            | 55-70          | Advanced exploits      | All services
//  9   | 9            | 65-85          | High-level exploits    | All services + local vulns
// 10   | 10+          | 80-100         | Maximum difficulty     | All services + local vulns
//
// Local privilege escalation vulnerabilities appear at tier 6+ (player level 6+).
// Local network depth increases with tier (tier 1-3: 0-1, tier 4-6: 1-2, tier 7+: 1-3).

// getTierForLevel returns the difficulty tier (1-10) for a given player level.
func getTierForLevel(level int) int {
	tier := level
	if tier < 1 {
		tier = 1
	}
	if tier > 10 {
		tier = 10
	}
	return tier
}

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

	// Calculate security level based on tier system (10 tiers)
	tier := getTierForLevel(level)
	
	// Security level ranges per tier:
	// Tier 1: 1-10, Tier 2: 10-20, ..., Tier 10: 80-100
	minSecurity := (tier - 1) * 10
	if tier == 1 {
		minSecurity = 1
	}
	maxSecurity := tier * 10
	if tier == 10 {
		maxSecurity = 100
	}
	
	securityLevel := minSecurity + g.rng.Intn(maxSecurity-minSecurity+1)
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
	server.Roles = g.generateRolesForLevel(level)
	server.LocalVulnerabilities = g.generateLocalVulnerabilities(level)
	server.FileSystem = g.generateFileSystem(server.Roles)

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
	server.Roles = g.generateRolesForLevel(localLevel)
	server.LocalVulnerabilities = g.generateLocalVulnerabilities(localLevel)
	server.FileSystem = g.generateFileSystem(server.Roles)

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
	hasTelnet := false
	hasFTP := false
	hasHTTP := false

	if serverType == "http" {
		hasHTTP = true
		hasSSH = g.rng.Float32() < 0.5
	} else if serverType == "ssh" {
		hasSSH = true
		hasHTTP = false
	} else if serverType == "telnet" {
		hasTelnet = true
		hasSSH = g.rng.Float32() < 0.3 // Sometimes also has SSH
	} else if serverType == "ftp" {
		hasFTP = true
		hasSSH = g.rng.Float32() < 0.5
	} else {
		// Random selection for variety
		hasHTTP = g.rng.Float32() < 0.4
		hasTelnet = g.rng.Float32() < 0.2 // Less common
		hasFTP = g.rng.Float32() < 0.25
	}

	// Generate SSH service (primary shell access)
	if hasSSH {
		sshLevel := 10 + (level * 2) + g.rng.Intn(20) - 10
		if sshLevel < 5 {
			sshLevel = 5
		}
		services = append(services, models.Service{
			Name:              "ssh",
			Description:       "Secure Shell",
			Port:              22,
			Vulnerable:        true,
			Level:             sshLevel,
			GrantsShellAccess: true,
			Vulnerabilities: []models.Vulnerability{
				{Type: "remote_code_execution", Level: sshLevel - 5 + g.rng.Intn(10)},
				{Type: "password_cracking", Level: sshLevel - 3 + g.rng.Intn(8)},
			},
		})
	}

	// Generate Telnet service (legacy shell access, weaker)
	if hasTelnet {
		telnetLevel := 5 + (level * 1) + g.rng.Intn(10) - 5
		if telnetLevel < 3 {
			telnetLevel = 3
		}
		services = append(services, models.Service{
			Name:              "telnet",
			Description:       "Telnet Service",
			Port:              23,
			Vulnerable:        true,
			Level:             telnetLevel,
			GrantsShellAccess: true,
			Vulnerabilities: []models.Vulnerability{
				{Type: "password_cracking", Level: telnetLevel - 2 + g.rng.Intn(5)},
			},
		})
	}

	// Generate FTP service (shell access only with RCE)
	if hasFTP {
		ftpLevel := 8 + (level * 1) + g.rng.Intn(15) - 5
		if ftpLevel < 5 {
			ftpLevel = 5
		}
		hasRCE := g.rng.Float32() < 0.4 // 40% chance of RCE vulnerability
		vulns := []models.Vulnerability{
			{Type: "password_cracking", Level: ftpLevel - 3 + g.rng.Intn(8)},
		}
		if hasRCE {
			vulns = append(vulns, models.Vulnerability{Type: "remote_code_execution", Level: ftpLevel + g.rng.Intn(5)})
		}
		services = append(services, models.Service{
			Name:              "ftp",
			Description:       "FTP Server",
			Port:              21,
			Vulnerable:        true,
			Level:             ftpLevel,
			GrantsShellAccess: hasRCE, // Only grants shell with RCE
			Vulnerabilities:   vulns,
		})
	}

	// Generate HTTP service (no shell access - data extraction only)
	if hasHTTP {
		httpLevel := 5 + (level * 2) + g.rng.Intn(15) - 5
		if httpLevel < 3 {
			httpLevel = 3
		}
		services = append(services, models.Service{
			Name:              "http",
			Description:       "Web Server",
			Port:              80,
			Vulnerable:        g.rng.Float32() < 0.7,
			Level:             httpLevel,
			GrantsShellAccess: false, // HTTP doesn't grant shell access
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
	for _, obj := range mission.Objectives {
		if obj.TargetType != "" {
			return obj.TargetType
		}
	}
	// Check required tools to determine server type
	for _, tool := range mission.RequiredTools {
		if tool == "sql_injector" || tool == "database_dumper" || tool == "xss_exploit" {
			return "http"
		}
		if tool == "password_cracker" || tool == "ssh_exploit" {
			return "ssh"
		}
		if tool == "telnet_exploit" {
			return "telnet"
		}
		if tool == "ftp_exploit" {
			return "ftp"
		}
	}

	// Check objectives
	for _, obj := range mission.Objectives {
		if obj.Tool == "sql_injector" || obj.Tool == "database_dumper" || obj.Tool == "xss_exploit" {
			return "http"
		}
		if obj.Tool == "password_cracker" || obj.Tool == "ssh_exploit" {
			return "ssh"
		}
		if obj.Tool == "telnet_exploit" {
			return "telnet"
		}
		if obj.Tool == "ftp_exploit" {
			return "ftp"
		}
	}

	return "" // No specific type
}

func (g *ServerGenerator) generateRolesForLevel(level int) []models.Role {
	userLevel := level / 2
	if userLevel < 1 {
		userLevel = 1
	}
	adminLevel := level
	if adminLevel < 5 {
		adminLevel = 5
	}
	return []models.Role{
		{
			Role:       "user",
			Level:      userLevel,
			Type:       models.RoleTypeUser,
			HomeDir:    "/home/user",
			Shell:      "/bin/bash",
			CanSudo:    false,
			SudoNoPass: false,
			Groups:     []string{"users"},
		},
		{
			Role:       "admin",
			Level:      adminLevel,
			Type:       models.RoleTypeAdmin,
			HomeDir:    "/home/admin",
			Shell:      "/bin/bash",
			CanSudo:    true,
			SudoNoPass: false,
			Groups:     []string{"users", "sudo", "adm"},
		},
		{
			Role:       "root",
			Level:      adminLevel + 20,
			Type:       models.RoleTypeRoot,
			HomeDir:    "/root",
			Shell:      "/bin/bash",
			CanSudo:    true,
			SudoNoPass: true,
			Groups:     []string{"root"},
		},
	}
}

func (g *ServerGenerator) generateLocalVulnerabilities(level int) []models.LocalVulnerability {
	tier := getTierForLevel(level)
	
	// Local privilege escalation vulnerabilities only appear at tier 6+
	// This aligns with the privilege escalation mission (corp_espionage_04)
	if tier < 6 {
		return nil
	}
	
	// Vulnerability levels scale with tier
	baseVulnLevel := (tier - 4) * 5 // Tier 6 = level 10, tier 10 = level 30
	
	vulns := []models.LocalVulnerability{
		{
			Type:        "sudo_misconfiguration",
			Level:       baseVulnLevel + g.rng.Intn(5),
			Description: "User can run a privileged binary without a password",
			Target:      "/usr/bin/vim",
			GrantsRoot:  true,
		},
		{
			Type:        "suid_binary",
			Level:       baseVulnLevel + 3 + g.rng.Intn(5),
			Description: "SUID binary with unsafe input handling",
			Target:      "/opt/backup/backup_tool",
			GrantsRoot:  true,
		},
		{
			Type:        "kernel_exploit",
			Level:       baseVulnLevel + 8 + g.rng.Intn(8),
			Description: "Outdated kernel vulnerable to privilege escalation",
			Target:      "CVE-2021-4034 (PwnKit)",
			GrantsRoot:  true,
		},
	}

	// Higher tiers have more local vulnerabilities
	vulnCount := 1
	if tier >= 8 && g.rng.Float32() < 0.5 {
		vulnCount = 2
	}
	if tier >= 9 && g.rng.Float32() < 0.3 {
		vulnCount = 3
	}

	// Select vulnerabilities randomly
	result := make([]models.LocalVulnerability, 0, vulnCount)
	used := make(map[int]bool)
	for len(result) < vulnCount && len(result) < len(vulns) {
		idx := g.rng.Intn(len(vulns))
		if !used[idx] {
			used[idx] = true
			result = append(result, vulns[idx])
		}
	}

	return result
}

func (g *ServerGenerator) generateFileSystem(roles []models.Role) map[string]interface{} {
	homeDirs := map[string]interface{}{}
	for _, role := range roles {
		if role.Role == "root" {
			homeDirs["root"] = map[string]interface{}{
				".bash_history": map[string]interface{}{"content": "sudo -l\ncat /etc/shadow\n"},
			}
			continue
		}
		homeDirs[role.Role] = map[string]interface{}{
			"notes.txt": map[string]interface{}{"content": "TODO: rotate credentials\n"},
		}
	}

	return map[string]interface{}{
		"etc": map[string]interface{}{
			"motd":  map[string]interface{}{"content": "Authorized access only."},
			"passwd": map[string]interface{}{"content": "root:x:0:0:root:/root:/bin/bash\nuser:x:1000:1000:User:/home/user:/bin/bash\nadmin:x:1001:1001:Admin:/home/admin:/bin/bash"},
		},
		"var": map[string]interface{}{
			"log": map[string]interface{}{
				"auth.log": map[string]interface{}{"content": "Jan 24 03:14:22 sshd[1234]: Failed password for admin from 10.0.0.5\n"},
			},
		},
		"home": homeDirs,
	}
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

// Server Lifecycle Management

const (
	// MAX_PROCEDURAL_SERVERS is the maximum number of procedural servers to keep
	MAX_PROCEDURAL_SERVERS = 100
	
	// SERVER_DEPLETION_THRESHOLD is the percentage of resources that must be taken before a server is considered depleted
	SERVER_DEPLETION_THRESHOLD = 0.9 // 90% of resources extracted
)

// IsServerDepleted checks if a procedural server has been fully exploited
func (g *ServerGenerator) IsServerDepleted(server *models.Server) bool {
	// Check if resources are mostly extracted
	// Original resources are estimated based on security level
	baseResources := float64(server.SecurityLevel * 100)
	currentResources := server.Wallet.Crypto + (server.Wallet.Data / 1000) // Normalize data
	
	// If current resources are less than 10% of estimated original, consider depleted
	if baseResources > 0 && currentResources < (baseResources * (1 - SERVER_DEPLETION_THRESHOLD)) {
		return true
	}
	
	return false
}

// CleanupDepletedServers removes procedural servers that have been fully exploited
// and are no longer useful. Returns the number of servers removed.
func (g *ServerGenerator) CleanupDepletedServers() (int, error) {
	// Get all procedural servers
	var proceduralServers []models.ProceduralServer
	if err := g.db.Find(&proceduralServers).Error; err != nil {
		return 0, fmt.Errorf("failed to get procedural servers: %w", err)
	}
	
	removed := 0
	for _, ps := range proceduralServers {
		// Get the actual server
		var server models.Server
		if err := g.db.Where("id = ?", ps.ServerID).First(&server).Error; err != nil {
			// Server doesn't exist, clean up tracking record
			g.db.Delete(&ps)
			continue
		}
		
		// Check if server is depleted
		if g.IsServerDepleted(&server) {
			// Remove server and tracking record
			if err := g.db.Delete(&server).Error; err != nil {
				continue
			}
			g.db.Delete(&ps)
			removed++
		}
	}
	
	return removed, nil
}

// EnforceServerLimit ensures the number of procedural servers doesn't exceed the maximum
// Removes the oldest depleted servers first, then oldest non-depleted if needed.
func (g *ServerGenerator) EnforceServerLimit() (int, error) {
	var proceduralServers []models.ProceduralServer
	if err := g.db.Order("generated_at ASC").Find(&proceduralServers).Error; err != nil {
		return 0, fmt.Errorf("failed to get procedural servers: %w", err)
	}
	
	if len(proceduralServers) <= MAX_PROCEDURAL_SERVERS {
		return 0, nil
	}
	
	excess := len(proceduralServers) - MAX_PROCEDURAL_SERVERS
	removed := 0
	
	// First pass: remove depleted servers (oldest first)
	for _, ps := range proceduralServers {
		if removed >= excess {
			break
		}
		
		var server models.Server
		if err := g.db.Where("id = ?", ps.ServerID).First(&server).Error; err != nil {
			g.db.Delete(&ps)
			removed++
			continue
		}
		
		if g.IsServerDepleted(&server) {
			g.db.Delete(&server)
			g.db.Delete(&ps)
			removed++
		}
	}
	
	// Second pass: remove oldest non-depleted servers if still over limit
	for _, ps := range proceduralServers {
		if removed >= excess {
			break
		}
		
		var server models.Server
		if err := g.db.Where("id = ?", ps.ServerID).First(&server).Error; err != nil {
			continue // Already handled
		}
		
		// Don't remove servers that haven't been touched at all
		// (give players a chance to exploit them)
		if server.Wallet.Crypto > 0 || server.Wallet.Data > 0 {
			// Only remove if it's been partially exploited
			g.db.Delete(&server)
			g.db.Delete(&ps)
			removed++
		}
	}
	
	return removed, nil
}

// RecycleServer regenerates a depleted server with new content
// Keeps the same IP but resets resources and vulnerabilities
func (g *ServerGenerator) RecycleServer(server *models.Server, playerLevel int) error {
	tier := getTierForLevel(playerLevel)
	
	// Reset security level based on player tier
	minSecurity := (tier - 1) * 10
	if tier == 1 {
		minSecurity = 1
	}
	maxSecurity := tier * 10
	if tier == 10 {
		maxSecurity = 100
	}
	
	server.SecurityLevel = minSecurity + g.rng.Intn(maxSecurity-minSecurity+1)
	
	// Regenerate services and vulnerabilities
	server.Services = g.generateServicesForDifficulty(playerLevel, "")
	server.Roles = g.generateRolesForLevel(playerLevel)
	server.LocalVulnerabilities = g.generateLocalVulnerabilities(playerLevel)
	server.FileSystem = g.generateFileSystem(server.Roles)
	
	// Reset resources
	server.Resources.CPU = 1000 + (playerLevel * 100) + g.rng.Intn(500)
	server.Resources.Bandwidth = float64(5000 + (playerLevel * 200) + g.rng.Intn(1000))
	server.Resources.RAM = 256 + (playerLevel * 10) + g.rng.Intn(100)
	
	// Reset wallet
	server.Wallet.Crypto = float64(1000 + (playerLevel * 50) + g.rng.Intn(500))
	server.Wallet.Data = float64(1000000 + (playerLevel * 100000) + g.rng.Intn(1000000))
	
	return g.db.Save(server).Error
}

// GetProceduralServerCount returns the current count of procedural servers
func (g *ServerGenerator) GetProceduralServerCount() int64 {
	var count int64
	g.db.Model(&models.ProceduralServer{}).Count(&count)
	return count
}
