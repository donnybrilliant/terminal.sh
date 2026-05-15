package services

import (
	"fmt"
	"terminal-sh/config"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// NetworkService handles network scanning operations including internet and local network scans.
type NetworkService struct {
	serverService   *ServerService
	shopService     *ShopService     // Will be set if available
	serverGenerator *ServerGenerator // Optional server generator
	missionService  *MissionService  // For checking mission completion (internet gating)
}

// NewNetworkService creates a new NetworkService with the provided server service.
func NewNetworkService(serverService *ServerService) *NetworkService {
	return &NetworkService{
		serverService: serverService,
	}
}

// SetShopService sets the shop service for this network service (called after shop service is created).
func (n *NetworkService) SetShopService(shopService *ShopService) {
	n.shopService = shopService
}

// SetServerGenerator sets the server generator for this network service.
func (n *NetworkService) SetServerGenerator(generator *ServerGenerator) {
	n.serverGenerator = generator
}

// SetMissionService sets the mission service for internet gating.
func (n *NetworkService) SetMissionService(missionService *MissionService) {
	n.missionService = missionService
}

// ScanInternet scans the internet for top-level servers (servers not in local networks).
// Deprecated: Use ScanInternetForUser for proper internet gating.
func (n *NetworkService) ScanInternet() ([]models.Server, error) {
	return n.ScanInternetForUser(uuid.Nil)
}

// ScanInternetForUser scans the internet for a specific user.
// If user hasn't completed 'home_recovery' mission, only shows home.pc (local network only).
// If procedural server generator is available and server count is low, generates new servers.
func (n *NetworkService) ScanInternetForUser(userID uuid.UUID) ([]models.Server, error) {
	// Check if user has completed the initial home_recovery mission
	// If not, they're "offline" and can only see their local home.pc
	if userID != uuid.Nil && n.missionService != nil {
		if !n.missionService.HasCompletedMission(userID, "home_recovery") {
			// User is offline - only show home.pc
			homePC, err := n.serverService.GetServerByIP("home.pc")
			if err != nil {
				return nil, fmt.Errorf("failed to find home.pc: %w", err)
			}
			return []models.Server{*homePC}, nil
		}
	}
	
	// User has internet access - show all servers
	servers, err := n.serverService.GetAllTopLevelServers()
	if err != nil {
		return nil, fmt.Errorf("failed to scan internet: %w", err)
	}
	
	// Check if we need to generate more servers
	if n.serverGenerator != nil && len(servers) < config.MinServersOnline {
		if err := n.serverGenerator.CheckAndGenerateServers(userID); err != nil {
			// Log error but continue - return existing servers
		} else {
			// Re-fetch servers after generation
			servers, err = n.serverService.GetAllTopLevelServers()
			if err != nil {
				return nil, fmt.Errorf("failed to scan internet: %w", err)
			}
		}
	}
	
	return servers, nil
}

// ScanIP scans a specific IP address for services and vulnerabilities.
func (n *NetworkService) ScanIP(ip string) (*models.Server, error) {
	server, err := n.serverService.GetServerByIP(ip)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return server, nil
}

// ScanLocalNetwork scans the local network of a server (retrieves connected servers).
func (n *NetworkService) ScanLocalNetwork(serverIP string) ([]models.Server, error) {
	servers, err := n.serverService.GetConnectedServers(serverIP)
	if err != nil {
		return nil, fmt.Errorf("failed to scan local network: %w", err)
	}
	return servers, nil
}

// FormatScanResult formats a server scan result for display
func (n *NetworkService) FormatScanResult(server *models.Server) string {
	result := fmt.Sprintf("IP: %s\n", server.IP)
	result += fmt.Sprintf("Local IP: %s\n", server.LocalIP)
	result += fmt.Sprintf("Security Level: %d\n", server.SecurityLevel)
	result += fmt.Sprintf("Resources: CPU=%d, Bandwidth=%.1f, RAM=%d\n",
		server.Resources.CPU, server.Resources.Bandwidth, server.Resources.RAM)
	result += fmt.Sprintf("Wallet: Crypto=%.2f, Data=%.2f\n",
		server.Wallet.Crypto, server.Wallet.Data)
	
	if len(server.Services) > 0 {
		result += "Services:\n"
		for _, service := range server.Services {
			result += fmt.Sprintf("  - %s (port %d): %s\n", service.Name, service.Port, service.Description)
			if service.Vulnerable && len(service.Vulnerabilities) > 0 {
				result += "    Vulnerabilities:\n"
				for _, vuln := range service.Vulnerabilities {
					result += fmt.Sprintf("      - %s (level %d)\n", vuln.Type, vuln.Level)
				}
			}
		}
	}
	
	if len(server.LocalNetwork) > 0 {
		result += fmt.Sprintf("Local Network Hosts: %d\n", len(server.LocalNetwork))
	}
	
	// Check if server has a shop
	if n.shopService != nil {
		shop, err := n.shopService.GetShopByServerIP(server.IP)
		if err == nil {
			result += fmt.Sprintf("Shop: [%s] %s (%s)\n", shop.ShopType, shop.Name, shop.Description)
		}
	}
	
	return result
}
