package cmd

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/services"
	"terminal-sh/ui"
	"time"
)

// showExploitProgress shows a progress bar for exploitation
func (h *CommandHandler) showExploitProgress(toolName, targetIP string) {
	if h.progressService != nil && h.user != nil {
		duration := h.progressService.CalculateOperationTime(services.OperationExploit, h.user.Resources)
		durationSeconds := time.Duration(duration * float64(time.Second))
		
		showProgressBar(fmt.Sprintf("Exploiting %s with %s...", targetIP, toolName), durationSeconds)
	}
}

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
	case "password_sniffer":
		return h.handlePasswordSniffer(args)
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
	case "advanced_exploit_kit":
		return h.handleAdvancedExploitKit(args)
	case "sql_injector":
		return h.handleSQLInjector(args)
	case "xss_exploit":
		return h.handleXSSExploit(args)
	case "packet_capture":
		return h.handlePacketCapture(args)
	case "packet_decoder":
		return h.handlePacketDecoder(args)
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

	h.showExploitProgress("password_cracker", targetIP)

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "password_cracker", "ssh"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 10)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited SSH service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using password_cracker") + "\n")
	return &CommandResult{Output: output.String()}
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

	h.showExploitProgress("ssh_exploit", targetIP)

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "ssh_exploit", "ssh"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 15)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited SSH service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using ssh_exploit") + "\n")
	return &CommandResult{Output: output.String()}
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

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Users and roles:", "👥"))
	
	for _, role := range server.Roles {
		output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render(role.Role) + " " + ui.ValueStyle.Render(fmt.Sprintf("(level %d)", role.Level))))
	}

	if len(server.Roles) == 0 {
		output.Reset()
		output.WriteString("No users found\n")
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 5)

	return &CommandResult{Output: output.String()}
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

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Local network connections:", "🔍"))
	
	for _, connServer := range connectedServers {
		output.WriteString(ui.FormatListBullet(formatIP(connServer.IP) + " (" + formatIP(connServer.LocalIP) + ")"))
	}

	if len(connectedServers) == 0 {
		output.Reset()
		output.WriteString("No local network connections found\n")
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 5)

	return &CommandResult{Output: output.String()}
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

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Rootkit installed on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Hidden backdoor access established.") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 20)

	return &CommandResult{Output: output.String()}
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

	h.showExploitProgress("exploit_kit", targetIP)

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

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("✅ Successfully exploited %d service(s) on ", exploitedCount)) + formatIP(targetIP) + ui.SuccessStyle.Render(" using exploit_kit") + "\n")
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handlePasswordSniffer(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: password_sniffer <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Sniff passwords from roles
	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Sniffed passwords from user roles:", "🔓"))
	
	for _, role := range server.Roles {
		output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render(role.Role+": password123") + " " + ui.SuccessStyleNoBold.Render("(cracked)")))
	}

	if len(server.Roles) == 0 {
		output.Reset()
		output.WriteString("No user roles found to sniff\n")
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 12)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleAdvancedExploitKit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: advanced_exploit_kit <targetIP>")}
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

	h.showExploitProgress("advanced_exploit_kit", targetIP)

	// Try to exploit all vulnerable services with advanced kit
	exploitedCount := 0
	for _, service := range server.Services {
		if service.Vulnerable {
			if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "advanced_exploit_kit", service.Name); err == nil {
				exploitedCount++
			}
		}
	}

	if exploitedCount == 0 {
		return &CommandResult{Error: fmt.Errorf("no vulnerabilities could be exploited")}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, exploitedCount*15)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("✅ Successfully exploited %d service(s) on ", exploitedCount)) + formatIP(targetIP) + ui.SuccessStyle.Render(" using advanced_exploit_kit") + "\n")
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleSQLInjector(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: sql_injector <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find HTTP service
	var httpService *models.Service
	for i := range server.Services {
		if server.Services[i].Name == "http" {
			httpService = &server.Services[i]
			break
		}
	}

	if httpService == nil {
		return &CommandResult{Error: fmt.Errorf("HTTP service not found on server")}
	}

	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	h.showExploitProgress("sql_injector", targetIP)

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "sql_injector", "http"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 18)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited HTTP service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using sql_injector") + "\n")
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleXSSExploit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: xss_exploit <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find HTTP service
	var httpService *models.Service
	for i := range server.Services {
		if server.Services[i].Name == "http" {
			httpService = &server.Services[i]
			break
		}
	}

	if httpService == nil {
		return &CommandResult{Error: fmt.Errorf("HTTP service not found on server")}
	}

	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	h.showExploitProgress("xss_exploit", targetIP)

	if err := h.exploitationService.ExploitServer(h.user.ID, serverPath, "xss_exploit", "http"); err != nil {
		return &CommandResult{Error: err}
	}

	// Add experience
	h.userService.AddExperience(h.user.ID, 12)

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited HTTP service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using xss_exploit") + "\n")
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handlePacketCapture(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: packet_capture <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Simulate packet capture
	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("📡 Capturing packets on ") + formatIP(targetIP) + "...\n")
	output.WriteString(ui.FormatKeyValuePair("Packets captured:", "42") + "\n")
	output.WriteString(ui.FormatKeyValuePair("Saved to:", "/tmp/captured_packets.pcap") + "\n")

	// Add experience
	h.userService.AddExperience(h.user.ID, 8)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handlePacketDecoder(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: packet_decoder <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Simulate packet decoding
	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("🔓 Decoding packets from ") + formatIP(targetIP) + "...\n")
	output.WriteString(ui.FormatSectionHeader("Decoded information:", ""))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Protocol:", "TCP")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Source:", "192.168.1.100:443")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Destination:", "10.0.0.5:8080")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Payload:", "[encrypted data]")))

	// Add experience
	h.userService.AddExperience(h.user.ID, 6)

	return &CommandResult{Output: output.String()}
}

