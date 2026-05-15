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

	// Wrap operation to check for mission auto-completion after each exploit
	wrappedOp := func() *CommandResult {
		result := operation()
		if result != nil && result.Error == nil && h.user != nil && h.missionService != nil {
			if completion := h.missionService.TryAutoComplete(h.user.ID); completion != nil {
				result.MissionCompleted = completion
			}
		}
		return result
	}
	
	return &CommandResult{
		StartProgress: &ProgressOperationRequest{
			ID:        operationID,
			Message:   fmt.Sprintf("Exploiting %s with %s...", targetIP, toolName),
			Duration:  duration,
			Operation: wrappedOp,
		},
	}
}

// passwordCrackableServices returns services that can be targeted by password cracking tools
var passwordCrackableServices = []string{"ssh", "telnet", "ftp"}

// findPasswordCrackableService finds a password-crackable service on a server
// Returns the service and its name, or nil if none found
func (h *CommandHandler) findPasswordCrackableService(server *models.Server) (*models.Service, string) {
	for _, serviceName := range passwordCrackableServices {
		for i := range server.Services {
			if server.Services[i].Name == serviceName && server.Services[i].Vulnerable {
				// Check if it has password_cracking vulnerability
				for _, vuln := range server.Services[i].Vulnerabilities {
					if vuln.Type == "password_cracking" {
						return &server.Services[i], serviceName
					}
				}
			}
		}
	}
	return nil, ""
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
	// Privilege escalation tools
	case "privesc_scanner":
		return h.handlePrivescScanner(args)
	case "sudo_exploit":
		return h.handleSudoExploit(args)
	case "kernel_exploit":
		return h.handleKernelExploit(args)
	case "suid_finder":
		return h.handleSUIDFinder(args)
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

	// Find a password-crackable service (SSH, Telnet, FTP)
	targetService, serviceName := h.findPasswordCrackableService(server)
	if targetService == nil {
		return &CommandResult{Error: fmt.Errorf("no password-crackable service (SSH, Telnet, FTP) found on server")}
	}

	// Check if user has the tool and it can crack passwords
	if !h.toolService.UserHasTool(h.user.ID, "password_cracker") {
		return &CommandResult{Error: fmt.Errorf("tool password_cracker not owned")}
	}

	// Get effective tool to check exploit level
	tool, err := h.toolService.GetEffectiveTool(h.user.ID, "password_cracker")
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("tool not found")}
	}

	// Check if tool can crack this service's password vulnerability
	var canCrack bool
	var vulnLevel int
	for _, vuln := range targetService.Vulnerabilities {
		if vuln.Type == "password_cracking" {
			for _, exploit := range tool.Exploits {
				if exploit.Type == "password_cracking" && exploit.Level >= vuln.Level {
					canCrack = true
					vulnLevel = vuln.Level
					break
				}
			}
		}
	}
	if !canCrack {
		return &CommandResult{Error: fmt.Errorf("password_cracker cannot crack %s on this server (security too high)", serviceName)}
	}

	// Calculate server path
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	credService := h.credentialService
	userService := h.userService
	serverLogService := h.serverLogService
	actionTracker := h.actionTracker
	userID := h.user.ID
	sourceIP := h.GetEffectiveSourceIP()
	username := h.user.Username
	capturedServiceName := serviceName
	capturedServerPath := serverPath
	capturedServer := server
	capturedVulnLevel := vulnLevel

	return h.createExploitProgressResult("password_cracker", targetIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("Password Cracker Results") + "\n")
		output.WriteString(ui.DimStyle.Render("Target: ") + formatIP(targetIP) + " (" + capturedServiceName + ")\n\n")

		// Check if user has enumerated users (bonus: crack more accounts)
		discoveredUsers, _ := credService.GetDiscoveredUsers(userID, capturedServerPath)
		
		// Determine which users to crack
		var usersToCrack []struct{ username, role string }
		
		if len(discoveredUsers) > 0 {
			// Crack discovered users
			for _, u := range discoveredUsers {
				usersToCrack = append(usersToCrack, struct{ username, role string }{u.Username, u.Role})
			}
			output.WriteString(ui.SuccessStyle.Render("✓ Using enumerated user list") + "\n")
		} else {
			// Default: try common users based on server roles
			for _, role := range capturedServer.Roles {
				// Map role to typical username
				roleName := role.Role
				if roleName == "admin" {
					usersToCrack = append(usersToCrack, struct{ username, role string }{"admin", roleName})
				} else if roleName == "root" {
					usersToCrack = append(usersToCrack, struct{ username, role string }{"root", roleName})
				} else {
					usersToCrack = append(usersToCrack, struct{ username, role string }{roleName, roleName})
				}
			}
			// Always try at least one user
			if len(usersToCrack) == 0 {
				usersToCrack = append(usersToCrack, struct{ username, role string }{"admin", "admin"})
			}
			output.WriteString(ui.WarningStyle.Render("⚠ No enumerated users - trying common accounts") + "\n")
		}

		output.WriteString("\n")
		crackedCount := 0

		// Crack each user
		for _, user := range usersToCrack {
			// Generate password based on username and server
			password := services.GeneratePassword(user.username, capturedServer.IP, user.role)
			
			// Simulate cracking difficulty based on vulnerability level
			// Higher level = harder to crack = chance of failure
			// For now, always succeed if tool level is sufficient
			
			// Store the credential
			err := credService.SaveCredential(
				userID,
				capturedServerPath,
				capturedServiceName,
				user.username,
				password,
				user.role,
				models.CredentialTypeCracked,
				"password_cracker",
			)
			if err != nil {
				output.WriteString(ui.ErrorStyle.Render("✗ Failed to crack: "+user.username) + "\n")
				continue
			}

			crackedCount++
			output.WriteString(ui.SuccessStyle.Render("✓ Cracked: ") + 
				ui.InfoStyle.Render(user.username) + " : " + 
				ui.WarningStyle.Render(password) + 
				ui.DimStyle.Render(" ("+user.role+")") + "\n")
		}

		// Log the exploit attempt
		if serverLogService != nil {
			serverLogService.LogExploitAttempt(capturedServer.IP, sourceIP, username, &userID, "password_cracker", capturedServiceName, crackedCount > 0)
		}

		if crackedCount == 0 {
			return &CommandResult{Error: fmt.Errorf("failed to crack any passwords")}
		}

		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "password_cracker", capturedServerPath, capturedServiceName)
			actionTracker.TrackCredentialCrack(userID, "password_cracker", capturedServerPath, capturedServiceName)
		}

		// Add experience based on cracked accounts and difficulty
		xp := 5 * crackedCount * (1 + capturedVulnLevel/10)
		userService.AddExperience(userID, xp)

		output.WriteString("\n" + ui.SuccessStyle.Render(fmt.Sprintf("Cracked %d credential(s)! Use 'credentials' to view.", crackedCount)) + "\n")
		output.WriteString(ui.InfoStyle.Render(fmt.Sprintf("You can now connect with: %s %s", capturedServiceName, targetIP)) + "\n")
		
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

	// Check if user has the tool
	if !h.toolService.UserHasTool(h.user.ID, "ssh_exploit") {
		return &CommandResult{Error: fmt.Errorf("tool ssh_exploit not owned")}
	}

	// Get effective tool to check exploit level
	tool, err := h.toolService.GetEffectiveTool(h.user.ID, "ssh_exploit")
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("tool not found")}
	}

	// Check if tool can exploit RCE vulnerability
	var canExploit bool
	var exploitType string
	for _, vuln := range sshService.Vulnerabilities {
		if vuln.Type == "remote_code_execution" || vuln.Type == "buffer_overflow" {
			for _, exploit := range tool.Exploits {
				if exploit.Type == vuln.Type && exploit.Level >= vuln.Level {
					canExploit = true
					exploitType = vuln.Type
					break
				}
			}
		}
	}
	if !canExploit {
		return &CommandResult{Error: fmt.Errorf("ssh_exploit cannot exploit SSH on this server (no suitable RCE vulnerability)")}
	}

	// Calculate server path
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture variables for the closure
	credService := h.credentialService
	userService := h.userService
	serverLogService := h.serverLogService
	userID := h.user.ID
	sourceIP := h.GetEffectiveSourceIP()
	username := h.user.Username
	capturedServerPath := serverPath
	capturedExploitType := exploitType
	capturedServer := server

	return h.createExploitProgressResult("ssh_exploit", targetIP, func() *CommandResult {
		// Create backdoor access (RCE = direct shell access, no credentials needed)
		err := credService.CreateBackdoor(
			userID,
			capturedServerPath,
			"ssh",
			capturedExploitType,
			"ssh_exploit",
			"root", // RCE typically gives root access
		)
		if err != nil {
			return &CommandResult{Error: fmt.Errorf("failed to install backdoor: %w", err)}
		}

		// Log the exploit
		if serverLogService != nil {
			serverLogService.LogExploitAttempt(capturedServer.IP, sourceIP, username, &userID, "ssh_exploit", "ssh", true)
		}

		// Add experience
		userService.AddExperience(userID, 20)

		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("SSH Exploit Results") + "\n")
		output.WriteString(ui.DimStyle.Render("Target: ") + formatIP(targetIP) + " (ssh)\n\n")
		output.WriteString(ui.SuccessStyle.Render("✓ Exploited "+capturedExploitType+" vulnerability") + "\n")
		output.WriteString(ui.SuccessStyle.Render("✓ Backdoor installed (root access)") + "\n\n")
		output.WriteString(ui.InfoStyle.Render("Direct shell access granted! Connect with: ssh "+targetIP) + "\n")
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

	// Check if user has the tool
	if !h.toolService.UserHasTool(h.user.ID, "user_enum") {
		return &CommandResult{Error: fmt.Errorf("tool user_enum not owned")}
	}

	// Calculate server path
	serverPath := targetIP
	if h.currentServerPath != "" {
		serverPath = h.currentServerPath + ".localNetwork." + targetIP
	}

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	credentialService := h.credentialService
	roles := server.Roles
	actionTracker := h.actionTracker
	capturedServerPath := serverPath

	return h.createExploitProgressResult("user_enum", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "user_enum", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("User Enumeration Results") + "\n")
		output.WriteString(ui.DimStyle.Render("Target: ") + formatIP(targetIP) + "\n\n")
		
		if len(roles) == 0 {
			output.WriteString(ui.WarningStyle.Render("No users found on server") + "\n")
			return &CommandResult{Output: output.String()}
		}

		output.WriteString(ui.InfoStyle.Render("Discovered Users:") + "\n")
		
		discoveredCount := 0
		for _, role := range roles {
			// Map role to username (roles often match usernames in this game)
			username := role.Role
			if role.Role == "admin" {
				username = "admin"
			} else if role.Role == "root" {
				username = "root"
			}

			// Store the discovered user
			err := credentialService.DiscoverUser(
				userID,
				capturedServerPath,
				username,
				role.Role,
				"", // service agnostic
				"user_enum",
			)
			if err == nil {
				discoveredCount++
			}

			output.WriteString(ui.FormatListBullet(
				ui.ValueStyle.Render(username) + " " + 
				ui.DimStyle.Render(fmt.Sprintf("(role: %s, level: %d)", role.Role, role.Level)),
			))
		}

		// Add experience
		userService.AddExperience(userID, 5+discoveredCount)

		output.WriteString("\n" + ui.SuccessStyle.Render(fmt.Sprintf("Enumerated %d user(s)!", discoveredCount)) + "\n")
		output.WriteString(ui.InfoStyle.Render("Tip: Use password_cracker to crack these accounts") + "\n")

		return &CommandResult{Output: output.String()}
	})
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

	// Get connected servers (before async closure)
	connectedServers, err := h.serverService.GetConnectedServers(targetIP)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("lan_sniffer", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "lan_sniffer", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.FormatSectionHeader("Local network scan complete:", "🔍"))
		
		for _, connServer := range connectedServers {
			output.WriteString(ui.FormatListBullet(formatIP(connServer.IP) + " (" + formatIP(connServer.LocalIP) + ")"))
		}

		if len(connectedServers) == 0 {
			output.Reset()
			output.WriteString("No local network connections found\n")
		}

		// Add experience
		userService.AddExperience(userID, 5)

		return &CommandResult{Output: output.String()}
	})
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
	sourceIP := h.GetEffectiveSourceIP()

	return h.createExploitProgressResult("exploit_kit", targetIP, func() *CommandResult {
		// Try to exploit all vulnerable services
		exploitedCount := 0
		for _, service := range services {
			if service.Vulnerable {
				if err := exploitService.ExploitServer(userID, serverPath, "exploit_kit", service.Name, sourceIP); err == nil {
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	roles := server.Roles
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("password_sniffer", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "password_sniffer", targetIP, "")
		}

		// Sniff passwords from roles
		var output strings.Builder
		output.WriteString(ui.FormatSectionHeader("Sniffed passwords from user roles:", "🔓"))
		
		for _, role := range roles {
			output.WriteString(ui.FormatListBullet(ui.ValueStyle.Render(role.Role+": password123") + " " + ui.SuccessStyleNoBold.Render("(cracked)")))
		}

		if len(roles) == 0 {
			output.Reset()
			output.WriteString("No user roles found to sniff\n")
		}

		// Add experience
		userService.AddExperience(userID, 12)

		return &CommandResult{Output: output.String()}
	})
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
	sourceIP := h.GetEffectiveSourceIP()

	return h.createExploitProgressResult("advanced_exploit_kit", targetIP, func() *CommandResult {
		// Try to exploit all vulnerable services with advanced kit
		exploitedCount := 0
		for _, service := range services {
			if service.Vulnerable {
				if err := exploitService.ExploitServer(userID, serverPath, "advanced_exploit_kit", service.Name, sourceIP); err == nil {
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
	sourceIP := h.GetEffectiveSourceIP()

	return h.createExploitProgressResult("sql_injector", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "sql_injector", "http", sourceIP); err != nil {
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
	sourceIP := h.GetEffectiveSourceIP()

	return h.createExploitProgressResult("xss_exploit", targetIP, func() *CommandResult {
		if err := exploitService.ExploitServer(userID, serverPath, "xss_exploit", "http", sourceIP); err != nil {
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("packet_capture", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "packet_capture", targetIP, "")
		}

		// Simulate packet capture
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("📡 Packet capture complete on ") + formatIP(targetIP) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Packets captured:", "42") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Saved to:", "~/captures/"+targetIP+".pcap") + "\n")

		// Add experience
		userService.AddExperience(userID, 8)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("packet_decoder", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "packet_decoder", targetIP, "")
		}

		// Simulate packet decoding
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("🔓 Packets decoded from ") + formatIP(targetIP) + "\n")
		output.WriteString(ui.FormatSectionHeader("Decoded information:", ""))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Protocol:", "TCP")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Source:", "192.168.1.100:443")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Destination:", "10.0.0.5:8080")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Payload:", "[encrypted data]")))

		// Add experience
		userService.AddExperience(userID, 6)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("log_cleaner", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "log_cleaner", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ System logs cleared on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". All traces removed.") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 15)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("timestomper", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "timestomper", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ File timestamps modified on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Tracks covered.") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 12)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("phishing_kit", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "phishing_kit", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("📧 Phishing campaign complete against ") + formatIP(targetIP) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Emails sent:", "150") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Responses:", "23") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Credentials captured:", "8") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 20)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("audit_disable", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "audit_disable", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ System auditing disabled on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Future logs prevented.") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 18)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("log_analyzer", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "log_analyzer", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("📊 Log analysis complete on ") + formatIP(targetIP) + "\n")
		output.WriteString(ui.FormatSectionHeader("Intelligence gathered:", ""))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Failed login attempts:", "47")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Successful logins:", "12")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Suspicious IPs:", "3")))
		output.WriteString(ui.FormatListBullet(ui.FormatKeyValuePair("Admin access times:", "02:00-04:00")))
		output.WriteString(ui.FormatKeyValuePair("Saved to:", "~/logs/"+targetIP+"-analysis.txt") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 10)

		return &CommandResult{Output: output.String()}
	})
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

	// Capture for async closure
	userService := h.userService
	userID := h.user.ID
	actionTracker := h.actionTracker

	return h.createExploitProgressResult("backup_destroyer", targetIP, func() *CommandResult {
		// Track tool usage for mission validation
		if actionTracker != nil {
			actionTracker.TrackToolUse(userID, "backup_destroyer", targetIP, "")
		}

		var output strings.Builder
		output.WriteString(ui.SuccessStyle.Render("✅ Backups destroyed on ") + formatIP(targetIP) + ui.SuccessStyle.Render(". Recovery prevented.") + "\n")
		output.WriteString(ui.FormatKeyValuePair("Backup files deleted:", "8") + "\n")
		
		// Add experience
		userService.AddExperience(userID, 20)

		return &CommandResult{Output: output.String()}
	})
}

// =====================================
// PRIVILEGE ESCALATION TOOLS
// =====================================

// handlePrivescScanner scans for local privilege escalation vulnerabilities.
// This tool must be run on a server you have shell access to.
func (h *CommandHandler) handlePrivescScanner(args []string) *CommandResult {
	// This tool is run locally on the server, no args needed
	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("privesc_scanner must be run on a remote server (connect first)")}
	}

	// Check if we're already root
	if h.IsCurrentRoleRoot() {
		return &CommandResult{Output: ui.InfoStyle.Render("Already running as root - no privilege escalation needed.")}
	}

	// Get the current server
	pathParts := strings.Split(h.currentServerPath, ".")
	serverIP := pathParts[len(pathParts)-1]
	
	server, err := h.serverService.GetServerByIP(serverIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found")}
	}

	// Capture variables for async operation
	userService := h.userService
	userID := h.user.ID
	capturedLocalVulns := server.LocalVulnerabilities // Capture for async closure

	return h.createExploitProgressResult("privesc_scanner", serverIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("🔍 Scanning for privilege escalation vectors...") + "\n\n")

		// Check for local vulnerabilities
		if len(capturedLocalVulns) == 0 {
			output.WriteString(ui.WarningStyle.Render("No obvious privilege escalation vectors found.") + "\n")
			output.WriteString(ui.DimStyle.Render("System appears to be well-configured.") + "\n")
		} else {
			output.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("Found %d potential privilege escalation vectors:", len(capturedLocalVulns))) + "\n\n")
			
			for i, vuln := range capturedLocalVulns {
				vulnType := formatPrivescType(vuln.Type)
				levelStr := fmt.Sprintf("Level %d", vuln.Level)
				rootStr := ""
				if vuln.GrantsRoot {
					rootStr = ui.SuccessStyle.Render(" → root")
				}
				
				output.WriteString(fmt.Sprintf("  %d. %s %s%s\n", i+1, vulnType, ui.DimStyle.Render(levelStr), rootStr))
				output.WriteString(fmt.Sprintf("     %s\n", ui.DimStyle.Render(vuln.Description)))
				if vuln.Target != "" {
					output.WriteString(fmt.Sprintf("     Target: %s\n", ui.ValueStyle.Render(vuln.Target)))
				}
				output.WriteString("\n")
			}
			
			output.WriteString(ui.InfoStyle.Render("Use sudo_exploit, kernel_exploit, or suid_finder to exploit these vectors.") + "\n")
		}

		// Add experience
		userService.AddExperience(userID, 15)

		return &CommandResult{Output: output.String()}
	})
}

// handleSudoExploit attempts to exploit sudo misconfigurations.
func (h *CommandHandler) handleSudoExploit(args []string) *CommandResult {
	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("sudo_exploit must be run on a remote server (connect first)")}
	}

	if h.IsCurrentRoleRoot() {
		return &CommandResult{Output: ui.InfoStyle.Render("Already running as root.")}
	}

	// Get the current server
	pathParts := strings.Split(h.currentServerPath, ".")
	serverIP := pathParts[len(pathParts)-1]
	
	server, err := h.serverService.GetServerByIP(serverIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found")}
	}

	// Check for sudo misconfiguration vulnerability
	var sudoVuln *models.LocalVulnerability
	for i := range server.LocalVulnerabilities {
		if server.LocalVulnerabilities[i].Type == "sudo_misconfiguration" {
			sudoVuln = &server.LocalVulnerabilities[i]
			break
		}
	}

	if sudoVuln == nil {
		return &CommandResult{Error: fmt.Errorf("no sudo misconfiguration found - use privesc_scanner first")}
	}

	// Capture variables for async
	userService := h.userService
	userID := h.user.ID
	roleService := h.roleService
	serverPath := h.currentServerPath
	currentRole := "user"
	if h.currentRole != nil {
		currentRole = h.currentRole.Username
	}

	return h.createExploitProgressResult("sudo_exploit", serverIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("🔓 Exploiting sudo misconfiguration...") + "\n\n")
		output.WriteString(fmt.Sprintf("Target: %s\n", ui.ValueStyle.Render(sudoVuln.Target)))
		output.WriteString(ui.DimStyle.Render(sudoVuln.Description) + "\n\n")

		if sudoVuln.GrantsRoot {
			// Record privilege escalation
			if roleService != nil {
				roleService.RecordPrivilegeEscalation(
					userID, serverPath, currentRole, "root",
					"sudo_misconfiguration", "sudo_exploit", true,
				)
			}
			output.WriteString(ui.SuccessStyle.Render("✅ PRIVILEGE ESCALATION SUCCESSFUL!") + "\n")
			output.WriteString(ui.SuccessStyle.Render("Now running as: root") + "\n\n")
			output.WriteString(ui.InfoStyle.Render("Reconnect to the server to use root privileges.") + "\n")
		} else {
			output.WriteString(ui.WarningStyle.Render("Exploit executed but did not grant root access.") + "\n")
		}

		userService.AddExperience(userID, 50)
		return &CommandResult{Output: output.String()}
	})
}

// handleKernelExploit attempts to exploit kernel vulnerabilities.
func (h *CommandHandler) handleKernelExploit(args []string) *CommandResult {
	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("kernel_exploit must be run on a remote server (connect first)")}
	}

	if h.IsCurrentRoleRoot() {
		return &CommandResult{Output: ui.InfoStyle.Render("Already running as root.")}
	}

	// Get the current server
	pathParts := strings.Split(h.currentServerPath, ".")
	serverIP := pathParts[len(pathParts)-1]
	
	server, err := h.serverService.GetServerByIP(serverIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found")}
	}

	// Check for kernel vulnerability
	var kernelVuln *models.LocalVulnerability
	for i := range server.LocalVulnerabilities {
		if server.LocalVulnerabilities[i].Type == "kernel_exploit" {
			kernelVuln = &server.LocalVulnerabilities[i]
			break
		}
	}

	if kernelVuln == nil {
		return &CommandResult{Error: fmt.Errorf("no kernel vulnerability found - use privesc_scanner first")}
	}

	// Capture variables for async
	userService := h.userService
	userID := h.user.ID
	roleService := h.roleService
	serverPath := h.currentServerPath
	currentRole := "user"
	if h.currentRole != nil {
		currentRole = h.currentRole.Username
	}

	return h.createExploitProgressResult("kernel_exploit", serverIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("💀 Exploiting kernel vulnerability...") + "\n\n")
		output.WriteString(fmt.Sprintf("Target: %s\n", ui.ValueStyle.Render(kernelVuln.Target)))
		output.WriteString(ui.DimStyle.Render(kernelVuln.Description) + "\n\n")
		output.WriteString(ui.DimStyle.Render("Compiling exploit payload...") + "\n")
		output.WriteString(ui.DimStyle.Render("Triggering kernel bug...") + "\n")
		output.WriteString(ui.DimStyle.Render("Overwriting credentials structure...") + "\n\n")

		if kernelVuln.GrantsRoot {
			// Record privilege escalation
			if roleService != nil {
				roleService.RecordPrivilegeEscalation(
					userID, serverPath, currentRole, "root",
					"kernel_exploit", "kernel_exploit", true,
				)
			}
			output.WriteString(ui.SuccessStyle.Render("✅ KERNEL EXPLOIT SUCCESSFUL!") + "\n")
			output.WriteString(ui.SuccessStyle.Render("uid=0(root) gid=0(root)") + "\n\n")
			output.WriteString(ui.InfoStyle.Render("Reconnect to the server to use root privileges.") + "\n")
		} else {
			output.WriteString(ui.ErrorStyle.Render("Kernel exploit failed - system may have been patched.") + "\n")
		}

		userService.AddExperience(userID, 75)
		return &CommandResult{Output: output.String()}
	})
}

// handleSUIDFinder finds and exploits SUID binaries.
func (h *CommandHandler) handleSUIDFinder(args []string) *CommandResult {
	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("suid_finder must be run on a remote server (connect first)")}
	}

	if h.IsCurrentRoleRoot() {
		return &CommandResult{Output: ui.InfoStyle.Render("Already running as root.")}
	}

	// Get the current server
	pathParts := strings.Split(h.currentServerPath, ".")
	serverIP := pathParts[len(pathParts)-1]
	
	server, err := h.serverService.GetServerByIP(serverIP)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("server not found")}
	}

	// Check for SUID vulnerability
	var suidVuln *models.LocalVulnerability
	for i := range server.LocalVulnerabilities {
		if server.LocalVulnerabilities[i].Type == "suid_binary" {
			suidVuln = &server.LocalVulnerabilities[i]
			break
		}
	}

	if suidVuln == nil {
		return &CommandResult{Error: fmt.Errorf("no exploitable SUID binary found - use privesc_scanner first")}
	}

	// Capture variables for async
	userService := h.userService
	userID := h.user.ID
	roleService := h.roleService
	serverPath := h.currentServerPath
	currentRole := "user"
	if h.currentRole != nil {
		currentRole = h.currentRole.Username
	}

	return h.createExploitProgressResult("suid_finder", serverIP, func() *CommandResult {
		var output strings.Builder
		output.WriteString(ui.HeaderStyle.Render("🔍 Scanning for SUID binaries...") + "\n\n")
		
		// Show some fake SUID binaries first
		output.WriteString(ui.DimStyle.Render("Found SUID binaries:") + "\n")
		output.WriteString(ui.DimStyle.Render("  /usr/bin/passwd (expected)") + "\n")
		output.WriteString(ui.DimStyle.Render("  /usr/bin/sudo (expected)") + "\n")
		output.WriteString(ui.DimStyle.Render("  /usr/bin/su (expected)") + "\n")
		output.WriteString(fmt.Sprintf("  %s %s\n\n", ui.WarningStyle.Render(suidVuln.Target), ui.SuccessStyle.Render("← EXPLOITABLE!")))

		output.WriteString(fmt.Sprintf("Exploiting: %s\n", ui.ValueStyle.Render(suidVuln.Target)))
		output.WriteString(ui.DimStyle.Render(suidVuln.Description) + "\n\n")

		if suidVuln.GrantsRoot {
			// Record privilege escalation
			if roleService != nil {
				roleService.RecordPrivilegeEscalation(
					userID, serverPath, currentRole, "root",
					"suid_binary", "suid_finder", true,
				)
			}
			output.WriteString(ui.SuccessStyle.Render("✅ SUID EXPLOIT SUCCESSFUL!") + "\n")
			output.WriteString(ui.SuccessStyle.Render("Spawned root shell via SUID binary") + "\n\n")
			output.WriteString(ui.InfoStyle.Render("Reconnect to the server to use root privileges.") + "\n")
		} else {
			output.WriteString(ui.WarningStyle.Render("SUID binary found but could not escalate to root.") + "\n")
		}

		userService.AddExperience(userID, 40)
		return &CommandResult{Output: output.String()}
	})
}

// formatPrivescType formats a privilege escalation type for display
func formatPrivescType(vulnType string) string {
	types := map[string]string{
		"sudo_misconfiguration": ui.WarningStyle.Render("SUDO Misconfiguration"),
		"suid_binary":           ui.WarningStyle.Render("SUID Binary"),
		"kernel_exploit":        ui.ErrorStyle.Render("Kernel Vulnerability"),
		"cron_job":              ui.WarningStyle.Render("Writable Cron Job"),
		"writable_path":         ui.WarningStyle.Render("Writable PATH"),
		"docker_escape":         ui.ErrorStyle.Render("Docker Escape"),
		"capability_abuse":      ui.WarningStyle.Render("Capability Abuse"),
	}
	if formatted, ok := types[vulnType]; ok {
		return formatted
	}
	return ui.InfoStyle.Render(vulnType)
}
