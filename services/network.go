package services

import (
	"fmt"
	"terminal-sh/config"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// NetworkService handles network scanning operations including internet and local network scans.
type NetworkService struct {
	serverService            *ServerService
	shopService              *ShopService // Will be set if available
	serverGenerator *ServerGenerator // Optional server generator
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

// ScanInternet scans the internet for top-level servers (servers not in local networks).
// If procedural server generator is available and server count is low, generates new servers.
func (n *NetworkService) ScanInternet() ([]models.Server, error) {
	servers, err := n.serverService.GetAllTopLevelServers()
	if err != nil {
		return nil, fmt.Errorf("failed to scan internet: %w", err)
	}
	
	// Check if we need to generate more servers
	if n.serverGenerator != nil && len(servers) < config.MinServersOnline {
		// Try to get user ID from context if available
		// For now, we'll generate servers without user context
		// In production, this would be passed from the command handler
		if err := n.serverGenerator.CheckAndGenerateServers(uuid.Nil); err != nil {
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
	
	if len(server.Tools) > 0 {
		result += fmt.Sprintf("Available Tools (use 'get %s <toolName>' to download): %v\n", server.IP, server.Tools)
	}
	
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
	
	if len(server.ConnectedIPs) > 0 {
		result += fmt.Sprintf("Connected IPs: %v\n", server.ConnectedIPs)
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
