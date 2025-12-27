package services

import (
	"fmt"
	"terminal-sh/models"
)

// NetworkService handles network scanning operations
type NetworkService struct {
	serverService *ServerService
	shopService   *ShopService // Will be set if available
}

// NewNetworkService creates a new network service
func NewNetworkService(serverService *ServerService) *NetworkService {
	return &NetworkService{
		serverService: serverService,
	}
}

// SetShopService sets the shop service (called after shop service is created)
func (n *NetworkService) SetShopService(shopService *ShopService) {
	n.shopService = shopService
}

// ScanInternet scans the internet for top-level servers
func (n *NetworkService) ScanInternet() ([]models.Server, error) {
	servers, err := n.serverService.GetAllTopLevelServers()
	if err != nil {
		return nil, fmt.Errorf("failed to scan internet: %w", err)
	}
	return servers, nil
}

// ScanIP scans a specific IP address for services and vulnerabilities
func (n *NetworkService) ScanIP(ip string) (*models.Server, error) {
	server, err := n.serverService.GetServerByIP(ip)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return server, nil
}

// ScanLocalNetwork scans the local network of a server (connected IPs)
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
