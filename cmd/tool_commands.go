package cmd

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/services"
	"terminal-sh/ui"
	"time"
)

// getExploitDuration calculates the duration for an exploit operation based on user resources and tool resources
func (h *CommandHandler) getExploitDuration(toolName string) float64 {
	if h.progressService == nil || h.user == nil {
		return 2.0 // default 2 seconds
	}
	
	// Try to get the user's tool state with effective resources (includes upgrades)
	toolResources := h.getToolEffectiveResources(toolName)
	
	// Use the combined calculation if we have tool resources
	if toolResources.CPU > 0 || toolResources.Bandwidth > 0 || toolResources.RAM > 0 {
		return h.progressService.CalculateToolOperationTime(services.OperationExploit, h.user.Resources, toolResources)
	}
	
	// Fallback to user-only calculation
	return h.progressService.CalculateOperationTime(services.OperationExploit, h.user.Resources)
}

// getToolEffectiveResources gets the effective resources for a tool (including upgrades)
func (h *CommandHandler) getToolEffectiveResources(toolName string) models.ToolResources {
	if h.user == nil || h.db == nil {
		return models.ToolResources{}
	}
	
	// First, try to get the user's tool state (which has effective resources from upgrades)
	var toolState models.UserToolState
	if err := h.db.Preload("Tool").
		Where("user_id = ?", h.user.ID).
		Joins("JOIN tools ON tools.id = user_tool_states.tool_id").
		Where("tools.name = ?", toolName).
		First(&toolState).Error; err == nil {
		// Return effective resources if they exist
		if toolState.EffectiveResources.CPU > 0 || toolState.EffectiveResources.Bandwidth > 0 || toolState.EffectiveResources.RAM > 0 {
			return toolState.EffectiveResources
		}
		// Fall back to base tool resources
		return toolState.Tool.Resources
	}
	
	// If no tool state, try to get the base tool resources
	var tool models.Tool
	if err := h.db.Where("name = ?", toolName).First(&tool).Error; err == nil {
		return tool.Resources
	}
	
	return models.ToolResources{}
}

// createExploitProgressResult creates a CommandResult with async progress for exploits
func (h *CommandHandler) createExploitProgressResult(toolName, targetIP string, operation func() *CommandResult) *CommandResult {
	duration := h.getExploitDuration(toolName)
	operationID := fmt.Sprintf("exploit-%s-%s-%d", toolName, targetIP, time.Now().UnixNano())
	
	return &CommandResult{
		StartProgress: &ProgressOperationRequest{
			ID:        operationID,
			Message:   fmt.Sprintf("Exploiting %s with %s...", targetIP, toolName),
			Duration:  duration,
			Operation: operation,
		},
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
	case "log_cleaner":
		return h.handleLogCleaner(args)
	case "timestomper":
		return h.handleTimestomper(args)
	case "database_dumper":
		return h.handleDatabaseDumper(args)
	case "phishing_kit":
		return h.handlePhishingKit(args)
	case "audit_disable":
		return h.handleAuditDisable(args)
	case "hash_cracker":
		return h.handleHashCracker(args)
	case "log_analyzer":
		return h.handleLogAnalyzer(args)
	case "backup_destroyer":
		return h.handleBackupDestroyer(args)
	default:
		return &CommandResult{Error: fmt.Errorf("unknown tool command: %s", toolName)}
	}
}

func (h *CommandHandler) handlePasswordCracker(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: password_cracker <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find SSH service - validate before starting progress
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

	// Calculate server path
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("password_cracker", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "password_cracker", "ssh"); err != nil {
			return &CommandResult{Error: err}
		}

		// Add experience
		userService.AddExperience(userID, 10)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited SSH service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using password_cracker") + "\n")
		return &CommandResult{Output: output.String()}
	})
}

func (h *CommandHandler) handleSSHExploit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: ssh_exploit <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find SSH service - validate before starting progress
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

	// Calculate server path
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("ssh_exploit", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "ssh_exploit", "ssh"); err != nil {
			return &CommandResult{Error: err}
		}

		// Add experience
		userService.AddExperience(userID, 15)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited SSH service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using ssh_exploit") + "\n")
		return &CommandResult{Output: output.String()}
	})
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
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID
	services := server.Services

	return h.createExploitProgressResult("exploit_kit", targetIP, func() *CommandResult {
		// Try to exploit all vulnerable services
		exploitedCount := 0
		for _, service := range services {
			if service.Vulnerable {
				if err := exploitService.ExploitServer(userID, serverPath, "exploit_kit", service.Name); err == nil {
					exploitedCount++
				}
			}
		}

		if exploitedCount == 0 {
			return &CommandResult{Error: fmt.Errorf("no vulnerabilities could be exploited")}
		}

		// Add experience
		userService.AddExperience(userID, exploitedCount*10)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("✅ Successfully exploited %d service(s) on ", exploitedCount)) + formatIP(targetIP) + ui.SuccessStyle.Render(" using exploit_kit") + "\n")
		return &CommandResult{Output: output.String()}
	})
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
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID
	services := server.Services

	return h.createExploitProgressResult("advanced_exploit_kit", targetIP, func() *CommandResult {
		// Try to exploit all vulnerable services with advanced kit
		exploitedCount := 0
		for _, service := range services {
			if service.Vulnerable {
				if err := exploitService.ExploitServer(userID, serverPath, "advanced_exploit_kit", service.Name); err == nil {
					exploitedCount++
				}
			}
		}

		if exploitedCount == 0 {
			return &CommandResult{Error: fmt.Errorf("no vulnerabilities could be exploited")}
		}

		// Add experience
		userService.AddExperience(userID, exploitedCount*15)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("✅ Successfully exploited %d service(s) on ", exploitedCount)) + formatIP(targetIP) + ui.SuccessStyle.Render(" using advanced_exploit_kit") + "\n")
		return &CommandResult{Output: output.String()}
	})
}

func (h *CommandHandler) handleSQLInjector(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: sql_injector <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find HTTP service - validate before starting progress
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

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("sql_injector", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "sql_injector", "http"); err != nil {
			return &CommandResult{Error: err}
		}

		// Add experience
		userService.AddExperience(userID, 18)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited HTTP service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using sql_injector") + "\n")
		return &CommandResult{Output: output.String()}
	})
}

func (h *CommandHandler) handleXSSExploit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: xss_exploit <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find HTTP service - validate before starting progress
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

	// Capture variables for the closure
	exploitService := h.exploitationService
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("xss_exploit", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "xss_exploit", "http"); err != nil {
			return &CommandResult{Error: err}
		}

		// Add experience
		userService.AddExperience(userID, 12)

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Successfully exploited HTTP service on ") + formatIP(targetIP) + ui.SuccessStyle.Render(" using xss_exploit") + "\n")
		return &CommandResult{Output: output.String()}
	})
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

func (h *CommandHandler) handleLogCleaner(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: log_cleaner <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Check if server is exploited
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before cleaning logs")}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ System logs cleared on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". All traces removed.") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 15)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleTimestomper(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: timestomper <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Check if server is exploited
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before modifying timestamps")}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ File timestamps modified on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Tracks covered.") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 12)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleDatabaseDumper(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: database_dumper <targetIP>")}
	}

	targetIP := args[0]
	
	// Get server - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Find HTTP service - validate before starting progress
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

	// Check if server is exploited - validate before starting progress
	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before dumping database")}
	}

	// Capture variables for the closure
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("database_dumper", targetIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Database contents extracted from ") + formatIP(targetIP) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Tables dumped:", "12") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Records extracted:", "1,234") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Data size:", "45.2 MB") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 25)

		return &CommandResult{Output: output.String()}
	})
}

func (h *CommandHandler) handlePhishingKit(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: phishing_kit <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("📧 Phishing campaign launched against ") + formatIP(targetIP) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Emails sent:", "150") + "\n")
	output.WriteString(ui.FormatKeyValuePair("Responses:", "23") + "\n")
	output.WriteString(ui.FormatKeyValuePair("Credentials captured:", "8") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 20)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleAuditDisable(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: audit_disable <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Check if server is exploited
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before disabling audit")}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ System auditing disabled on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Future logs prevented.") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 18)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleHashCracker(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: hash_cracker <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists - validate before starting progress
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Capture variables for the closure
	userService := h.userService
	userID := h.user.ID

	return h.createExploitProgressResult("hash_cracker", targetIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("🔓 Cracking hashes on ") + formatIP(targetIP) + "...\n")
		output.WriteString(ui.FormatSectionHeader("Cracked hashes:", ""))
		output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render("admin:") + " " + ui.SuccessStyleNoBold.Render("password123") + " " + ui.FormatKeyValuePair("(MD5)", "")))
		output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render("user1:") + " " + ui.SuccessStyleNoBold.Render("qwerty") + " " + ui.FormatKeyValuePair("(SHA256)", "")))
		output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render("user2:") + " " + ui.SuccessStyleNoBold.Render("admin123") + " " + ui.FormatKeyValuePair("(bcrypt)", "")))
		
		// Add experience
		userService.AddExperience(userID, 22)

		return &CommandResult{Output: output.String()}
	})
}

func (h *CommandHandler) handleLogAnalyzer(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: log_analyzer <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("📊 Analyzing logs on ") + formatIP(targetIP) + "...\n")
	output.WriteString(ui.FormatSectionHeader("Intelligence gathered:", ""))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Failed login attempts:", "47")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Successful logins:", "12")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Suspicious IPs:", "3")))
	output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Admin access times:", "02:00-04:00")))
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 10)

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleBackupDestroyer(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: backup_destroyer <targetIP>")}
	}

	targetIP := args[0]
	
	// Check if server exists
	if _, err := h.serverService.GetServerByIP(targetIP); err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
	}

	// Check if server is exploited
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	if !h.exploitationService.IsServerExploited(h.user.ID, serverPath) {
		return &CommandResult{Error: fmt.Errorf("server must be exploited before destroying backups")}
	}

	var output strings.Builder
	output.WriteString(ui.SuccessStyle.Render("✅ Backups destroyed on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Recovery prevented.") + "\n")
	output.WriteString(ui.FormatKeyValuePair("Backup files deleted:", "8") + "\n")
	
	// Add experience
	h.userService.AddExperience(h.user.ID, 20)

	return &CommandResult{Output: output.String()}
}
