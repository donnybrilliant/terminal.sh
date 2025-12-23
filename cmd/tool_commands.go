package cmd

import (
	"fmt"
	"ssh4xx-go/models"
)

// handleToolCommand handles tool-specific commands
func (h *CommandHandler) handleToolCommand(toolName string, args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Check if user has the tool
	if !h.toolService.UserHasTool(h.user.ID, toolName) {
		return &CommandResult{Error: fmt.Errorf("tool %s not owned", toolName)}
	}

	switch toolName {
	case "password_cracker":
		return h.handlePasswordCracker(args)
	case "ssh_exploit":
		return h.handleSSHExploit(args)
	case "user_enum":
		return h.handleUserEnum(args)
	case "lan_sniffer":
		return h.handleLanSniffer(args)
	case "rootkit":
		return h.handleRootkit(args)
	case "exploit_kit":
		return h.handleExploitKit(args)
	default:
		return &CommandResult{Error: fmt.Errorf("unknown tool command: %s", toolName)}
	}
}

func (h *CommandHandler) handlePasswordCracker(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: password_cracker <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find SSH service
	var sshService *models.Service
	for i := range server.Services {
		if server.Services[i].Name == "ssh" {
			sshService = &server.Services[i]
			break
		}
	}

	if sshService == nil {
		return &CommandResult{Error: fmt.Errorf("SSH service not found on server")}
	}

	// Exploit the server
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "password_cracker", "ssh"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 10)

	output := fmt.Sprintf("Successfully exploited SSH service on %s using password_cracker", targetIP)
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleSSHExploit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: ssh_exploit <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find SSH service
	var sshService *models.Service
	for i := range server.Services {
		if server.Services[i].Name == "ssh" {
			sshService = &server.Services[i]
			break
		}
	}

	if sshService == nil {
		return &CommandResult{Error: fmt.Errorf("SSH service not found on server")}
	}

	// Exploit the server
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "ssh_exploit", "ssh"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 15)

	output := fmt.Sprintf("Successfully exploited SSH service on %s using ssh_exploit", targetIP)
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleUserEnum(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: user_enum <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	output := "Users and roles:\n"
	for _, role := range server.Roles {
		output += fmt.Sprintf("  - %s (level %d)\n", role.Role, role.Level)
	}

	if len(server.Roles) == 0 {
		output = "No users found"
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 5)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleLanSniffer(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: lan_sniffer <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Get connected servers
	connectedServers, err := h.serverService.GetConnectedServers(targetIP)
	if err != nil {
		return &CommandResult{Error: err}
	}

	output := "Local network connections:\n"
	for _, connServer := range connectedServers {
		output += fmt.Sprintf("  - %s (%s)\n", connServer.IP, connServer.LocalIP)
	}

	if len(connectedServers) == 0 {
		output = "No local network connections found"
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 5)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleRootkit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: rootkit <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Check if server is already exploited
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before installing rootkit")}
	}

	output := fmt.Sprintf("Rootkit installed on %s. Hidden backdoor access established.", targetIP)
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 20)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleExploitKit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: exploit_kit <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Try to exploit all vulnerable services
	exploitedCount := 0
	for _, service := range server.Services {
		if service.Vulnerable {
			if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "exploit_kit", service.Name); err == nil {
				exploitedCount++
			}
		}
	}

	if exploitedCount == 0 {
		return &CommandResult{Error: fmt.Errorf("no vulnerabilities could be exploited")}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, exploitedCount*10)

	output := fmt.Sprintf("Successfully exploited %d service(s) on %s using exploit_kit", exploitedCount, targetIP)
	return &CommandResult{Output: output}
}

