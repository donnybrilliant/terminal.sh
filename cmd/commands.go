// Package cmd provides command handlers for terminal commands executed in the game shell.
package cmd

import (
	"fmt"
	"math/rand"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"
	"terminal-sh/services"
	"terminal-sh/ui"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

// CommandResult represents the result of executing a command, including output, errors, and optional filesystem nodes.
type CommandResult struct {
	Output     string
	Error      error
	Nodes      []*filesystem.Node // For ls command
	LongFormat bool                // For ls -l format
	// MissionCompleted is set when a mission auto-completes (for UI to append rewards message)
	MissionCompleted *services.MissionCompletionResult
	// ASCII animation trigger (for ascii -a command)
	StartASCIIAnimation *ASCIIAnimationRequest
	// Progress operation (for long-running operations)
	StartProgress *ProgressOperationRequest
}

// ProgressOperationRequest contains parameters for starting a progress operation
type ProgressOperationRequest struct {
	ID        string                                     // Unique ID for this operation
	Message   string                                     // Message to display (e.g., "Downloading...")
	Duration  float64                                    // Duration in seconds
	Operation func() *CommandResult                      // The actual operation to run
}

// ASCIIAnimationRequest contains parameters for starting an ASCII animation
type ASCIIAnimationRequest struct {
	Text       string
	Colors     []string
	CharWidth  int
	CharHeight int
	SizeScale  int // Size as integer multiplier (1-10), 0 means use CharWidth/CharHeight directly
}

// CommandHandler handles execution of terminal commands and manages game state.
type CommandHandler struct {
	db              *database.Database
	vfs            *filesystem.VFS
	user           *models.User
	userService    *services.UserService
	serverService  *services.ServerService
	networkService *services.NetworkService
	sessionService *services.SessionService
	toolService    *services.ToolService
	exploitationService *services.ExploitationService
	miningService  *services.MiningService
	tutorialService *services.TutorialService
	shopService    *services.ShopService
	shopDiscovery  *services.ShopDiscovery
	upgradeService *services.UpgradeService
	progressService *services.ProgressService
	chatService    *services.ChatService
	missionService *services.MissionService
	achievementService *services.AchievementService
	rewardService  *services.RewardService
	missionGenerator *services.MissionGenerator
	serverGenerator *services.ServerGenerator
	serverLogService    *services.ServerLogService
	credentialService   *services.CredentialService
	roleService         *services.RoleService
	actionTracker       *services.ActionTracker
	homeVFS             *filesystem.VFS // User's home filesystem (never changes; used for downloads)
	currentServerPath   string     // Current server path if connected to a server
	currentServiceType  string     // Service type used for current connection (ssh, ftp, telnet, etc.)
	currentAccessMethod string     // How we accessed current server (credentials, backdoor)
	currentRole         *services.ConnectionRole // Current role/user on the connected server
	sessionID           *uuid.UUID // Current session ID
	onConnect           func(serverPath string) error // Callback for server connection
	onDisconnect        func() error                  // Callback for server disconnection
	// Deprecated: use onConnect instead
	onSSHConnect    func(serverPath string) error
	// Deprecated: use onDisconnect instead
	onSSHDisconnect func() error
}

// NewCommandHandler creates a new CommandHandler with the provided dependencies.
// Initializes all required services for command execution.
func NewCommandHandler(db *database.Database, vfs *filesystem.VFS, user *models.User, userService *services.UserService, chatService *services.ChatService) *CommandHandler {
	serverService := services.NewServerService(db)
	toolService := services.NewToolService(db, serverService)
	// UpgradeService for progressive tool upgrades
	upgradeService := services.NewUpgradeService(db, toolService, userService)
	networkService := services.NewNetworkService(serverService)
	shopService := services.NewShopService(db, serverService)
	networkService.SetShopService(shopService) // Link shop service to network service
	shopDiscovery := services.NewShopDiscovery(shopService, serverService, toolService)
	shopDiscovery.SetDatabase(db)
	shopDiscovery.SetUserService(userService)
	progressService := services.NewProgressService()
	sessionService := services.NewSessionService(db, serverService)
	exploitationService := services.NewExploitationService(db, toolService, serverService)
	miningService := services.NewMiningService(db, toolService, serverService)
	tutorialService, _ := services.NewTutorialService("") // Initialize tutorial service with default path (data/seed/tutorials.json), ignore error for now
	achievementService, _ := services.NewAchievementService(db, "") // Initialize achievement service
	rewardService := services.NewRewardService(db, userService, toolService, upgradeService, achievementService)
	missionService, _ := services.NewMissionService(db, "", rewardService) // Initialize mission service

	// Initialize mission generator
	missionGenerator := services.NewMissionGenerator(db, missionService, serverService, toolService, userService)
	missionService.SetMissionGenerator(missionGenerator)

	// Initialize server generator
	serverGenerator := services.NewServerGenerator(db, serverService)
	networkService.SetServerGenerator(serverGenerator)
	networkService.SetMissionService(missionService) // For internet gating

	// Initialize server log service
	serverLogService := services.NewServerLogService(db)

	// Initialize credential service
	credentialService := services.NewCredentialService(db)

	// Initialize role service (needs credential service for best-privilege selection)
	roleService := services.NewRoleService(db)
	roleService.SetCredentialService(credentialService)

	// Initialize action tracker for mission validation
	actionTracker := services.NewActionTracker(db)
	actionTracker.SetMissionService(missionService)
	missionService.SetActionTracker(actionTracker)

	return &CommandHandler{
		db:              db,
		vfs:            vfs,
		homeVFS:        vfs, // User's home VFS - never changes when connecting to servers
		user:           user,
		userService:    userService,
		serverService:  serverService,
		networkService: networkService,
		sessionService: sessionService,
		toolService:   toolService,
		exploitationService: exploitationService,
		miningService: miningService,
		tutorialService: tutorialService,
		shopService:    shopService,
		shopDiscovery:  shopDiscovery,
		upgradeService: upgradeService,
		progressService: progressService,
		chatService:   chatService,
		missionService: missionService,
		achievementService: achievementService,
		rewardService: rewardService,
		missionGenerator: missionGenerator,
		serverGenerator: serverGenerator,
		serverLogService: serverLogService,
		credentialService: credentialService,
		roleService: roleService,
		actionTracker: actionTracker,
	}
}

// SetSessionID sets the current session ID for this command handler.
func (h *CommandHandler) SetSessionID(sessionID uuid.UUID) {
	h.sessionID = &sessionID
}

// SetConnectionCallbacks sets callbacks for server connect and disconnect events.
func (h *CommandHandler) SetConnectionCallbacks(onConnect func(serverPath string) error, onDisconnect func() error) {
	h.onConnect = onConnect
	h.onDisconnect = onDisconnect
}

// SetSSHCallbacks sets callbacks for SSH connect and disconnect events.
// Deprecated: Use SetConnectionCallbacks instead.
func (h *CommandHandler) SetSSHCallbacks(onConnect func(serverPath string) error, onDisconnect func() error) {
	h.SetConnectionCallbacks(onConnect, onDisconnect)
}

// GetCurrentServiceType returns the service type used for the current connection.
func (h *CommandHandler) GetCurrentServiceType() string {
	if h.currentServiceType == "" {
		return "ssh" // default for backward compatibility
	}
	return h.currentServiceType
}

// SetCurrentServiceType sets the service type for the current connection.
func (h *CommandHandler) SetCurrentServiceType(serviceType string) {
	h.currentServiceType = serviceType
}

// GetCurrentServerPath returns the current server path (empty if on user's local system).
func (h *CommandHandler) GetCurrentServerPath() string {
	return h.currentServerPath
}

// SetCurrentServerPath sets the current server path (used for restoring from session stack).
func (h *CommandHandler) SetCurrentServerPath(path string) {
	h.currentServerPath = path
}

// GetPromptInfo returns the username and hostname for the shell prompt.
// When connected to a server, returns the actual role username and the server IP.
// When on local system, returns the user's username and "terminal.sh".
func (h *CommandHandler) GetPromptInfo() (username string, hostname string) {
	if h.currentServerPath == "" {
		// On local system
		username = "guest"
		if h.user != nil && h.user.Username != "" {
			username = h.user.Username
		}
		return username, "terminal.sh"
	}

	// Connected to a server - extract IP from path
	// Path format: "ip" or "ip.localNetwork.ip2.localNetwork.ip3"
	parts := strings.Split(h.currentServerPath, ".")
	serverIP := parts[len(parts)-1]

	// Use actual role username if available
	if h.currentRole != nil {
		return h.currentRole.Username, serverIP
	}

	// Fallback to root
	return "root", serverIP
}

// GetPromptChar returns the shell prompt character (# for root, $ for others).
func (h *CommandHandler) GetPromptChar() string {
	if h.currentServerPath == "" {
		return "$" // Local system always uses $
	}
	if h.currentRole != nil {
		return h.currentRole.PromptChar
	}
	return "#" // Default to root prompt for backward compatibility
}

// GetCurrentRole returns the current role on the connected server.
func (h *CommandHandler) GetCurrentRole() *services.ConnectionRole {
	return h.currentRole
}

// SetCurrentRole sets the current role on the connected server.
func (h *CommandHandler) SetCurrentRole(role *services.ConnectionRole) {
	h.currentRole = role
}

// IsCurrentRoleRoot returns true if the current role has root privileges.
func (h *CommandHandler) IsCurrentRoleRoot() bool {
	if h.currentRole != nil {
		return h.currentRole.IsRoot
	}
	return true // Default to root for backward compatibility
}

// GetEffectiveSourceIP returns the IP address that should appear in logs as the source.
// If the user is SSH'd to a server, returns that server's IP (the hop).
// If the user is on their local machine, returns the user's IP.
// trackToolUse records a tool being used for mission objective validation
func (h *CommandHandler) trackToolUse(toolName, targetServer, serviceName string) {
	if h.actionTracker != nil && h.user != nil {
		h.actionTracker.TrackToolUse(h.user.ID, toolName, targetServer, serviceName)
	}
}

// trackServerExploit records a successful server exploit for mission validation
func (h *CommandHandler) trackServerExploit(toolName, serverPath, serviceName string) {
	if h.actionTracker != nil && h.user != nil {
		h.actionTracker.TrackServerExploit(h.user.ID, toolName, serverPath, serviceName)
	}
}

// trackCredentialCrack records a credential being cracked for mission validation
func (h *CommandHandler) trackCredentialCrack(toolName, serverPath, serviceName string) {
	if h.actionTracker != nil && h.user != nil {
		h.actionTracker.TrackCredentialCrack(h.user.ID, toolName, serverPath, serviceName)
	}
}

func (h *CommandHandler) GetEffectiveSourceIP() string {
	if h.currentServerPath == "" {
		// On local machine - use user's IP
		if h.user != nil {
			return h.user.IP
		}
		return "unknown"
	}

	// SSH'd to a server - the source IP is the current server's IP
	// Extract IP from path (last segment)
	parts := strings.Split(h.currentServerPath, ".")
	return parts[len(parts)-1]
}

// SetVFS sets the VFS (Virtual FileSystem) for the command handler.
func (h *CommandHandler) SetVFS(vfs *filesystem.VFS) {
	h.vfs = vfs
}

// CreateServerVFS creates a VFS for a server by loading its filesystem from the database.
// Returns the created VFS or an error if the server is not found.
func (h *CommandHandler) CreateServerVFS(serverPath string) (*filesystem.VFS, error) {
	// Get server by path
	server, err := h.serverService.GetServerByPath(serverPath)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	
	// Create standard VFS (use "root" as username for server filesystems)
	vfs, err := filesystem.NewVFSFromMap("root", server.FileSystem)
	if err != nil {
		// Fall back to standard VFS if merge fails
		vfs = filesystem.NewVFS("root")
	}
	
	// Set up save callback for server filesystem
	vfs.SetServerID(serverPath)
	vfs.SetSaveCallback(func(changes map[string]interface{}) error {
		// Update server's filesystem in database
		server.FileSystem = changes
		return h.db.Model(&server).Update("file_system", changes).Error
	})
	
	return vfs, nil
}

// SyncUserToolsToVFS syncs user's owned tools to /usr/bin in the VFS
func (h *CommandHandler) SyncUserToolsToVFS() error {
	if h.user == nil {
		return nil // No user, nothing to sync
	}
	
	tools, err := h.toolService.GetUserTools(h.user.ID)
	if err != nil {
		return err
	}
	
	for _, tool := range tools {
		// Add tool as command in /usr/bin if it doesn't exist
		_, err := h.vfs.GetCommandDescription(tool.Name)
		if err != nil {
			// Command doesn't exist, add it
			if err := h.vfs.AddUserCommand(tool.Name, tool.Function); err != nil {
				// Log error but continue with other tools
				continue
			}
		}
	}
	
	return nil
}

func (h *CommandHandler) Execute(command string) *CommandResult {
	// Parse command
	parts := parseCommand(command)
	if len(parts) == 0 {
		return &CommandResult{Output: ""}
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "pwd":
		return h.handlePWD()
	case "ls":
		return h.handleLS(args)
	case "cd":
		return h.handleCD(args)
	case "cat":
		return h.handleCAT(args)
	case "clear":
		return h.handleCLEAR()
	case "help":
		return h.handleHELP()
	case "chat":
		return h.handleChat(args)
	case "tutorial":
		return h.handleTUTORIAL(args)
	case "mission":
		return h.handleMISSION(args)
	case "login":
		return h.handleLOGIN(args)
	case "logout":
		return h.handleLOGOUT()
	case "register":
		return h.handleREGISTER(args)
	case "userinfo":
		return h.handleUSERINFO()
	case "info":
		return h.handleINFO()
	case "whoami":
		return h.handleWHOAMI()
	case "name":
		return h.handleNAME(args)
	case "ifconfig":
		return h.handleIFCONFIG()
	case "scan":
		return h.handleSCAN(args)
	case "server":
		return h.handleSERVER()
	case "connect":
		return h.handleConnect(args, "") // Auto-detect service
	case "ssh":
		return h.handleConnect(args, "ssh")
	case "telnet":
		return h.handleConnect(args, "telnet")
	case "ftp":
		return h.handleConnect(args, "ftp")
	case "exit":
		return h.handleEXIT()
	case "get":
		return h.handleGET(args)
	case "download", "dl":
		return h.handleDOWNLOAD(args)
	case "tools":
		return h.handleTOOLS()
	case "exploited":
		return h.handleEXPLOITED()
	case "credentials", "creds":
		return h.handleCREDENTIALS(args)
	case "backdoors":
		return h.handleBACKDOORS()
	case "shop":
		return h.handleSHOP(args)
	case "buy":
		return h.handleBUY(args)
	case "patches":
		return h.handlePATCH(args)
	case "patch":
		return h.handlePATCH(args)
	case "crypto_miner":
		return h.handleCRYPTOMINER(args)
	case "ascii":
		return h.handleASCII(args)
	case "stop_mining":
		return h.handleSTOPMINING(args)
	case "miners":
		return h.handleMINERS()
	case "wallet":
		return h.handleWALLET()
	case "password_cracker", "password_sniffer", "ssh_exploit", "user_enum", "lan_sniffer", "rootkit", "exploit_kit", "advanced_exploit_kit", "sql_injector", "xss_exploit", "packet_capture", "packet_decoder", "log_cleaner", "timestomper", "database_dumper", "phishing_kit", "audit_disable", "hash_cracker", "log_analyzer", "backup_destroyer":
		return h.handleToolCommand(cmd, args)
	case "touch":
		return h.handleTOUCH(args)
	case "mkdir":
		return h.handleMKDIR(args)
	case "rm":
		return h.handleRM(args)
	case "cp":
		return h.handleCP(args)
	case "mv":
		return h.handleMV(args)
	case "edit", "vi", "nano":
		return h.handleEDIT(args)
	default:
		return &CommandResult{Error: fmt.Errorf("unknown command: %s. Type 'help' for available commands", cmd)}
	}
}

func (h *CommandHandler) handlePWD() *CommandResult {
	path := h.vfs.GetCurrentPath()
	if path != "" {
		path = ui.ListStyle.Render(path) + "\n"
	}
	return &CommandResult{Output: path}
}

func (h *CommandHandler) handleLS(args []string) *CommandResult {
	// Parse flags
	showAll := false    // -a flag
	longFormat := false // -l flag
	
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			// Parse flags (order-independent: -la = -al)
			for _, char := range arg[1:] {
				switch char {
				case 'a':
					showAll = true
				case 'l':
					longFormat = true
				}
			}
		}
	}
	
	nodes := h.vfs.ListDirWithOptions(showAll)
	
	if longFormat {
		// Return nodes for long format rendering
		return &CommandResult{Nodes: nodes, LongFormat: true}
	}
	
	return &CommandResult{Nodes: nodes}
}

func (h *CommandHandler) handleCD(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: cd <directory>")}
	}

	if err := h.vfs.ChangeDir(args[0]); err != nil {
		return &CommandResult{Error: err}
	}

	return &CommandResult{Output: ""}
}

func (h *CommandHandler) handleCAT(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: cat <filename>")}
	}

	filePath := args[0]
	
	// Check if we're on a server and reading a dynamic log file
	if h.currentServerPath != "" && h.serverLogService != nil {
		// Extract server IP from path
		parts := strings.Split(h.currentServerPath, ".")
		serverIP := parts[len(parts)-1]
		
		// Normalize the file path for comparison
		absPath := filePath
		if !strings.HasPrefix(filePath, "/") {
			absPath = h.vfs.GetCurrentPath() + "/" + filePath
		}
		
		// Check for dynamic log files
		switch {
		case strings.HasSuffix(absPath, "/var/log/auth.log") || absPath == "/var/log/auth.log":
			// Dynamic auth log - combine seeded content with dynamic logs
			content := h.getDynamicAuthLog(serverIP, filePath)
			if content != "" {
				return &CommandResult{Output: content}
			}
		case strings.HasSuffix(absPath, "/var/log/system.log") || absPath == "/var/log/system.log":
			// Dynamic system log - combine seeded content with dynamic logs
			content := h.getDynamicSystemLog(serverIP, filePath)
			if content != "" {
				return &CommandResult{Output: content}
			}
		}
	}

	content, err := h.vfs.ReadFile(filePath)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Ensure content ends with newline if not empty
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	output := content

	// Story mission trigger: cat README.txt at home starts home_recovery
	var missionCompleted *services.MissionCompletionResult
	if h.user != nil && h.missionService != nil && h.currentServerPath == "" {
		absPath := filePath
		if !strings.HasPrefix(filePath, "/") {
			curPath := h.vfs.GetCurrentPath()
			if curPath == "/" {
				absPath = "/" + filePath
			} else {
				absPath = curPath + "/" + filePath
			}
		}
		if started := h.missionService.TryTriggerMission(h.user.ID, "cat_file", absPath); started != nil {
			output += "\n" + ui.SuccessStyle.Render("📋 Mission started: ")+started.Name+ui.SuccessStyle.Render(" — objectives will complete automatically")+"\n"
			// If objectives were already done (e.g. user did connect/get before reading README), complete immediately
			missionCompleted = h.missionService.TryAutoComplete(h.user.ID)
		}
	}

	return &CommandResult{Output: output, MissionCompleted: missionCompleted}
}

// getDynamicAuthLog combines seeded auth.log content with dynamic server logs.
func (h *CommandHandler) getDynamicAuthLog(serverIP, filePath string) string {
	var content strings.Builder
	
	// First, try to read the seeded/static content
	if staticContent, err := h.vfs.ReadFile(filePath); err == nil && staticContent != "" {
		content.WriteString(staticContent)
		if !strings.HasSuffix(staticContent, "\n") {
			content.WriteString("\n")
		}
	}
	
	// Then append dynamic logs
	if dynamicContent, err := h.serverLogService.FormatAuthLog(serverIP, 50); err == nil && dynamicContent != "" {
		content.WriteString(dynamicContent)
		if !strings.HasSuffix(dynamicContent, "\n") {
			content.WriteString("\n")
		}
	}
	
	return content.String()
}

// getDynamicSystemLog combines seeded system.log content with dynamic server logs.
func (h *CommandHandler) getDynamicSystemLog(serverIP, filePath string) string {
	var content strings.Builder
	
	// First, try to read the seeded/static content
	if staticContent, err := h.vfs.ReadFile(filePath); err == nil && staticContent != "" {
		content.WriteString(staticContent)
		if !strings.HasSuffix(staticContent, "\n") {
			content.WriteString("\n")
		}
	}
	
	// Then append dynamic logs
	if dynamicContent, err := h.serverLogService.FormatSystemLog(serverIP, 50); err == nil && dynamicContent != "" {
		content.WriteString(dynamicContent)
		if !strings.HasSuffix(dynamicContent, "\n") {
			content.WriteString("\n")
		}
	}
	
	return content.String()
}

func (h *CommandHandler) handleCLEAR() *CommandResult {
	// ANSI escape sequence to clear screen
	return &CommandResult{Output: "\033[2J\033[H"}
}

func (h *CommandHandler) handleHELP() *CommandResult {
	// Read commands from filesystem
	binCommands, _ := h.vfs.ListCommands()
	
	// Emoji constants (can't import terminal due to import cycle)
	const (
		emojiFolder   = "📁"
		emojiTools    = "🛠️"
		emojiUser     = "👤"
		emojiNetwork  = "🌐"
		emojiScan     = "🔍"
		emojiSSH      = "🔌"
		emojiServer   = "🖥️"
		emojiTool     = "🛠️"
		emojiExploit  = "⚡"
		emojiMoney    = "💰"
		emojiTutorial = "📚"
		emojiShop     = "🛒"
		emojiBuy      = "🛍️"
		emojiPatch    = "🔧"
		emojiHelp     = "❓"
	)
	
	var output strings.Builder
	
	// Title
	output.WriteString(ui.HeaderStyle.Render("Available Commands") + "\n\n")
	
	// Helper function to format list items
	formatListItem := func(text, emoji string) string {
		return ui.FormatListItem(text, emoji)
	}
	
	// System Commands (Filesystem commands from VFS)
	if len(binCommands) > 0 {
		output.WriteString(ui.SectionStyle.Render(emojiFolder + " Filesystem:") + "\n")
		
	for _, cmd := range binCommands {
		desc, err := h.vfs.GetCommandDescription(cmd)
		if err != nil {
			desc = "No description available"
		}
		// Pad command name to 20 chars for alignment
		cmdPadded := cmd
		if len(cmdPadded) < 20 {
			cmdPadded += strings.Repeat(" ", 20-len(cmdPadded))
		}
			// Use color for command name
			output.WriteString("  " + ui.FilesystemCommandStyle.Render(cmdPadded) + " - " + desc + "\n")
		}
		output.WriteString("\n")
	}
	
	// Tool Commands (only show tools the user owns)
	if h.user != nil && h.toolService != nil {
		tools, err := h.toolService.GetUserTools(h.user.ID)
		if err == nil && len(tools) > 0 {
			output.WriteString(ui.AccentBoldStyle.Render(emojiTools + " Tool Commands:") + "\n")
			for _, tool := range tools {
				// Pad command name to 20 chars for alignment
				cmdPadded := tool.Name
				if len(cmdPadded) < 20 {
					cmdPadded += strings.Repeat(" ", 20-len(cmdPadded))
				}
				// Use color for command name
				output.WriteString("  " + ui.ToolCommandStyle.Render(cmdPadded) + " - " + tool.Function + "\n")
			}
			output.WriteString("\n")
		}
	}
	
	// User commands
	output.WriteString(ui.SuccessStyle.Render(emojiUser + " User:") + "\n")
	output.WriteString(formatListItem("userinfo            - Show user information", ""))
	output.WriteString(formatListItem("whoami              - Display current username", ""))
	output.WriteString(formatListItem("name <newName>      - Change username", ""))
	output.WriteString("\n")
	
	// Network commands
	output.WriteString(ui.InfoStyle.Render(emojiNetwork + " Network:") + "\n")
	output.WriteString(formatListItem("ifconfig            - Show network interfaces", ""))
	output.WriteString(formatListItem("scan [targetIP]     - Scan internet or IP", ""))
	output.WriteString(formatListItem("connect <targetIP>  - Connect via any exploited service", ""))
	output.WriteString(formatListItem("ssh <targetIP>      - Connect via SSH", ""))
	output.WriteString(formatListItem("telnet <targetIP>   - Connect via Telnet", ""))
	output.WriteString(formatListItem("ftp <targetIP>      - Connect via FTP (requires RCE)", ""))
	output.WriteString(formatListItem("exit                - Disconnect from server", ""))
	output.WriteString(formatListItem("server              - Show current server info", ""))
	output.WriteString("\n")
	
	// Tools/Game commands
	output.WriteString(ui.AccentBoldStyle.Render(emojiTool + " Tools:") + "\n")
	output.WriteString(formatListItem("get <targetIP> <tool> - Download tool from server", ""))
	output.WriteString(formatListItem("download <path>      - Download file to ~/Downloads/", ""))
	output.WriteString(formatListItem("tools                - List owned tools", ""))
	output.WriteString(formatListItem("exploited            - Show all server access", ""))
	output.WriteString(formatListItem("credentials          - List discovered credentials", ""))
	output.WriteString(formatListItem("backdoors            - List installed backdoors", ""))
	output.WriteString(formatListItem("wallet               - Show wallet balance", ""))
	output.WriteString("\n")
	
	// Learning
	output.WriteString(ui.InfoStyle.Render(emojiTutorial + " Learning:") + "\n")
	output.WriteString(formatListItem("tutorial             - Show available tutorials", ""))
	output.WriteString(formatListItem("tutorial <id>        - Start a tutorial", ""))
	output.WriteString("\n")
	
	// Story Missions
	output.WriteString(ui.AccentBoldStyle.Render("🎯 Story Missions:") + "\n")
	output.WriteString(formatListItem("mission              - List missions (story + board)", ""))
	output.WriteString(formatListItem("mission <id>         - View mission details", ""))
	output.WriteString(formatListItem("mission start <id>   - Accept a board mission", ""))
	output.WriteString(formatListItem("mission stop <id>    - Abandon a mission", ""))
	output.WriteString(formatListItem("mission status       - View your progress", ""))
	output.WriteString("\n")
	
	// Shopping
	output.WriteString(ui.AccentBoldStyle.Render(emojiShop + " Shopping:") + "\n")
	output.WriteString(formatListItem("shop                 - List discovered shops", ""))
	output.WriteString(formatListItem("shop <shopID>        - Browse shop inventory", ""))
	output.WriteString(formatListItem("buy <shopID> <item>  - Purchase item from shop", ""))
	output.WriteString("\n")
	
	// Tool Upgrades
	output.WriteString(ui.AccentBoldStyle.Render(emojiPatch + " Tool Upgrades:") + "\n")
	output.WriteString(formatListItem("patches              - List available patches", ""))
	output.WriteString(formatListItem("patch <name> <tool>  - Apply patch to tool", ""))
	output.WriteString(formatListItem("patch info <name>    - Show patch details", ""))
	output.WriteString(formatListItem("patch discover       - Scan server for patches", ""))
	output.WriteString("\n")
	
	// System
	output.WriteString(ui.ValueStyle.Render("⚙️ System:") + "\n")
	output.WriteString(formatListItem("clear                - Clear the screen", ""))
	output.WriteString(formatListItem("help                 - Show this help message", ""))
	output.WriteString("\n")
	output.WriteString(ui.GrayStyle.Render("Tip: use PgUp/PgDn or Ctrl+U/Ctrl+D to scroll output.") + "\n")
	
	// Ensure trailing newline
	helpText := output.String()
	if !strings.HasSuffix(helpText, "\n") {
		helpText += "\n"
	}

	return &CommandResult{Output: helpText}
}

func (h *CommandHandler) handleLOGIN(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: login <username> <password>")}
	}
	return &CommandResult{Output: ui.InfoStyle.Render("🔐 Please authenticate via SSH password authentication") + "\n"}
}

func (h *CommandHandler) handleLOGOUT() *CommandResult {
	return &CommandResult{Output: ui.FormatSuccessMessage("Logout successful. Goodbye!", "✅")}
}

func (h *CommandHandler) handleREGISTER(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: register <username> <password>")}
	}
	return &CommandResult{Output: ui.InfoStyle.Render("📝 Please register via SSH password authentication (login with new credentials)") + "\n"}
}

// GetASCIISize calculates charWidth and charHeight based on integer multiplier
// scale: 1-10 (multiplier for base pattern dimensions)
// viewportWidth, viewportHeight: terminal dimensions (unused but kept for API compatibility)
// Returns: charWidth, charHeight based on base pattern (5x7) multiplied by scale
func GetASCIISize(scale, viewportWidth, viewportHeight int) (int, int, error) {
	if scale < 1 || scale > 10 {
		return 0, 0, fmt.Errorf("size must be between 1 and 10 (got: %d)", scale)
	}
	
	// Base pattern dimensions (typical ASCII art pattern size)
	baseWidth := 5
	baseHeight := 7
	
	// Simply multiply by scale factor
	charWidth := baseWidth * scale
	charHeight := baseHeight * scale
	
	return charWidth, charHeight, nil
}

// getASCIIColorPalette returns color codes for a predefined palette
func getASCIIColorPalette(palette string) ([]string, error) {
	switch strings.ToLower(palette) {
	case "white":
		return []string{"255", "252", "248", "244", "248", "252", "255"}, nil
	case "orange":
		return []string{"208", "214", "220", "226", "220", "214", "208"}, nil
	case "green":
		return []string{"46", "82", "118", "154", "118", "82", "46"}, nil
	case "purple":
		return []string{"129", "135", "141", "147", "141", "135", "129"}, nil
	case "blue":
		return []string{"33", "39", "45", "51", "45", "39", "33"}, nil
	case "red":
		return []string{"196", "202", "208", "214", "208", "202", "196"}, nil
	case "cyan":
		return []string{"51", "87", "123", "159", "123", "87", "51"}, nil
	case "yellow":
		return []string{"226", "227", "228", "229", "228", "227", "226"}, nil
	case "pink", "magenta":
		return []string{"205", "213", "207", "219", "218", "212", "205"}, nil
	default:
		return nil, fmt.Errorf("unknown color palette: %s (use: white, orange, green, purple, blue, red, cyan, yellow, pink/magenta)", palette)
	}
}

// handleASCII handles the ascii command for testing ASCII art functionality
func (h *CommandHandler) handleASCII(args []string) *CommandResult {
	if len(args) == 0 {
		return h.handleASCIIHelp()
	}

	var text string
	var colors []string
	var sizeScale int = 1 // Default: 1x (base size)
	var charWidth, charHeight int = 5, 7 // Will be calculated from sizeScale
	var animate bool
	var showHelp bool

	// Parse flags and arguments
	i := 0
	for i < len(args) {
		arg := args[i]
		
		switch arg {
		case "-h", "--help":
			showHelp = true
		case "-a", "--animate":
			animate = true
		case "-c", "--color", "--colors":
			if i+1 < len(args) {
				paletteName := args[i+1]
				palette, err := getASCIIColorPalette(paletteName)
				if err != nil {
					return &CommandResult{Error: err}
				}
				colors = palette
				i++
			} else {
				return &CommandResult{Error: fmt.Errorf("color flag requires a palette name (white, orange, green, purple, blue, red, cyan, yellow, pink)")}
			}
		case "-s", "--size":
			if i+1 < len(args) {
				sizeStr := args[i+1]
				var err error
				sizeScale, err = strconv.Atoi(sizeStr)
				if err != nil || sizeScale < 1 || sizeScale > 10 {
					return &CommandResult{Error: fmt.Errorf("size must be a number between 1 and 10 (got: %s)", sizeStr)}
				}
				i++
			} else {
				return &CommandResult{Error: fmt.Errorf("size flag requires a number between 1 and 10")}
			}
		default:
			// Treat as text to convert
			if text == "" && !strings.HasPrefix(arg, "-") {
				text = arg
			} else if text != "" {
				// Multiple words - join them
				text += " " + arg
			}
		}
		i++
	}

	if showHelp {
		return h.handleASCIIHelp()
	}

	if text == "" {
		return &CommandResult{Error: fmt.Errorf("no text provided. Usage: ascii <text> [flags]")}
	}

	// Calculate character dimensions from size scale
	// Use default viewport size for static version (will be recalculated for animated version with actual viewport)
	defaultWidth, defaultHeight := 80, 24
	calculatedWidth, calculatedHeight, err := GetASCIISize(sizeScale, defaultWidth, defaultHeight)
	if err != nil {
		return &CommandResult{Error: err}
	}
	charWidth = calculatedWidth
	charHeight = calculatedHeight

	var output strings.Builder

	if animate {
		// Trigger ASCII animation in terminal UI
		// Return a special result that will start the full animation sequence
		// The actual dimensions will be recalculated when animation starts with real viewport size
		return &CommandResult{
			StartASCIIAnimation: &ASCIIAnimationRequest{
				Text:       text,
				Colors:     colors,
				CharWidth:  charWidth,
				CharHeight: charHeight,
				SizeScale:  sizeScale,
			},
		}
	} else {
		// Static version
		output.WriteString(ui.FormatSectionHeader("ASCII Art", "✨"))
		output.WriteString(ui.FormatKeyValuePair("Text", text) + "\n")
		output.WriteString(ui.FormatKeyValuePair("Size", fmt.Sprintf("%dx%d (scale: %dx)", charWidth, charHeight, sizeScale)) + "\n")
		if len(colors) > 0 {
			output.WriteString(ui.FormatKeyValuePair("Color Palette", "custom") + "\n")
		}
		output.WriteString("\n")
		
		// Generate and display ASCII art
		asciiArt := ui.StringToASCIIArt(text, charWidth, charHeight)
		styledArt := ui.RenderASCIIArtWithGradient(asciiArt, 0, colors)
		output.WriteString(styledArt)
		output.WriteString("\n")
	}

	return &CommandResult{Output: output.String()}
}

// handleASCIIHelp shows help for the ascii command
func (h *CommandHandler) handleASCIIHelp() *CommandResult {
	var output strings.Builder
	
	output.WriteString(ui.FormatSectionHeader("ASCII Art Command", "✨"))
	output.WriteString("\n")
	output.WriteString(ui.FormatKeyValuePair("Usage", "ascii <text> [flags]") + "\n")
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Flags:", ""))
	output.WriteString(ui.FormatListBullet("-h, --help - Show this help message"))
	output.WriteString(ui.FormatListBullet("-a, --animate - Create animated welcome animation (gradient → ASCII → fall away)"))
	output.WriteString(ui.FormatListBullet("-c, --color <palette> - Color palette: white, orange, green, purple, blue, red, cyan, yellow, pink"))
	output.WriteString(ui.FormatListBullet("-s, --size <scale> - Size scale multiplier: 1-10 (default: 1)"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Size:", ""))
	output.WriteString(ui.FormatListBullet("Size is a scale multiplier (1-10) that controls character dimensions"))
	output.WriteString(ui.FormatListBullet("Larger values = bigger ASCII characters (more blocks per character)"))
	output.WriteString(ui.FormatListBullet("Minimum size is enforced to ensure text is readable"))
	output.WriteString(ui.FormatListBullet("Examples: 1 (small), 3 (medium), 5 (large), 8 (extra large), 10 (maximum)"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Color Palettes:", ""))
	output.WriteString(ui.FormatListBullet("white, orange, green, purple, blue, red, cyan, yellow, pink/magenta"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Examples:", ""))
	output.WriteString(ui.FormatListBullet("ascii HELLO - Convert \"HELLO\" to ASCII art"))
	output.WriteString(ui.FormatListBullet("ascii WELCOME -a - Create animated welcome with \"WELCOME\" centered"))
	output.WriteString(ui.FormatListBullet("ascii TEST -c green - Use green color palette"))
	output.WriteString(ui.FormatListBullet("ascii \"HELLO WORLD\" -s 5 - Large size (scale 5x)"))
	output.WriteString(ui.FormatListBullet("ascii TERMINAL -a -c purple -s 3 - Animated with purple palette, scale 3x"))
	output.WriteString("\n")
	output.WriteString(ui.InfoStyle.Render("Note: Text is automatically converted to uppercase for better rendering.\n"))
	output.WriteString(ui.InfoStyle.Render("The -a flag creates a full-screen animation: gradient fills screen → ASCII art appears centered → everything falls away.\n"))
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleUSERINFO() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	var output strings.Builder
	
	// Header
	output.WriteString(ui.FormatSectionHeader("User Information", "👤"))
	
	// Labels and values
	output.WriteString(ui.FormatKeyValuePair("Username:", h.user.Username) + "\n")
	output.WriteString(ui.FormatKeyValuePair("IP:", ui.FormatIP(h.user.IP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Local IP:", ui.FormatIP(h.user.LocalIP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("MAC:", h.user.MAC) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Level:", fmt.Sprintf("%d", h.user.Level)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Experience:", fmt.Sprintf("%d", h.user.Experience)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Resources:", fmt.Sprintf("CPU=%d, Bandwidth=%.1f, RAM=%d", 
		h.user.Resources.CPU, h.user.Resources.Bandwidth, h.user.Resources.RAM)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Wallet:", fmt.Sprintf("Crypto=%.2f, Data=%.2f", 
		h.user.Wallet.Crypto, h.user.Wallet.Data)) + "\n")
	
	// Show achievements if available
	if h.achievementService != nil {
		achievements, err := h.achievementService.GetUserAchievements(h.user.ID)
		if err == nil && len(achievements) > 0 {
			output.WriteString("\n")
			output.WriteString(ui.FormatSectionHeader("Achievements", "🏆"))
			for _, ach := range achievements {
				achDef := h.achievementService.GetAchievementByName(ach.AchievementName)
				icon := "🏆"
				if achDef != nil && achDef.Icon != "" {
					icon = achDef.Icon
				}
				output.WriteString(ui.FormatListBullet(fmt.Sprintf("%s %s", icon, ach.AchievementName)))
			}
		}
	}
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleINFO() *CommandResult {
	// Display browser/client information (SSH session info)
	var output strings.Builder
	
	output.WriteString(ui.FormatSectionHeader("Client Information", "ℹ️"))
	
	output.WriteString(ui.FormatKeyValuePair("  Connection:", "SSH") + "\n")
	output.WriteString(ui.FormatKeyValuePair("  Protocol:", "SSH2") + "\n")
	if h.user != nil {
		output.WriteString(ui.FormatKeyValuePair("  Username:", h.user.Username) + "\n")
		output.WriteString(ui.FormatKeyValuePair("  IP Address:", ui.FormatIP(h.user.IP)) + "\n")
	}
	if h.sessionID != nil {
		output.WriteString(ui.FormatKeyValuePair("  Session ID:", h.sessionID.String()) + "\n")
	}
	output.WriteString(ui.FormatKeyValuePair("  Terminal:", "ANSI compatible") + "\n")
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleWHOAMI() *CommandResult {
	if h.user == nil {
		return &CommandResult{Output: ui.GrayStyle.Render("guest") + "\n"}
	}
	username := ui.SuccessStyle.Render(h.user.Username)
	if username != "" {
		username += "\n"
	}
	return &CommandResult{Output: username}
}

func (h *CommandHandler) handleNAME(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: name <newName>")}
	}
	
	newUsername := args[0]
	if err := h.userService.UpdateUsername(h.user.ID, newUsername); err != nil {
		return &CommandResult{Error: err}
	}
	
	// Rename the home directory in the VFS
	if err := h.vfs.RenameHomeDirectory(newUsername); err != nil {
		// If renaming fails, we still updated the username in DB, so log error but continue
		// In practice, this should rarely fail, but we want to be safe
		return &CommandResult{Error: fmt.Errorf("username updated but failed to rename home directory: %w", err)}
	}
	
	// Update local user object
	h.user.Username = newUsername
	
	return &CommandResult{Output: ui.SuccessStyle.Render("✅ Username changed to ") + ui.AccentBoldStyle.Render(newUsername) + "\n"}
}

func (h *CommandHandler) handleIFCONFIG() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	var output strings.Builder
	
	output.WriteString(ui.FormatSectionHeader("Network Configuration", "🌐"))
	
	output.WriteString(ui.FormatKeyValuePair("IP:", ui.FormatIP(h.user.IP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Local IP:", ui.FormatIP(h.user.LocalIP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("MAC:", h.user.MAC) + "\n")
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleSCAN(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// If no args, scan internet (top-level servers)
	if len(args) == 0 {
		servers, err := h.networkService.ScanInternetForUser(h.user.ID)
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		if len(servers) == 0 {
			return &CommandResult{Output: "No servers found\n"}
		}
		
		var output strings.Builder
		output.WriteString(ui.FormatSectionHeader("Found servers:", "🔍"))
		
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = ui.AccentStyle.Render(fmt.Sprintf(" [SHOP: %s]", shop.ShopType))
				}
			}
			output.WriteString(ui.FormatListBullet(formatIP(server.IP) + " (" + formatIP(server.LocalIP) + ")" + shopIndicator))
		}
		return &CommandResult{Output: output.String()}
	}

	// If in SSH mode (currentServerPath is set), scan local network
	if h.currentServerPath != "" {
		// Get the current server to scan its local network
		currentServer, err := h.serverService.GetServerByPath(h.currentServerPath)
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		servers, err := h.networkService.ScanLocalNetwork(currentServer.IP)
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		if len(servers) == 0 {
			return &CommandResult{Output: "No connected servers found\n"}
		}
		
		var output strings.Builder
		output.WriteString(ui.FormatSectionHeader("Connected servers:", "🔍"))
		
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = ui.AccentStyle.Render(fmt.Sprintf(" [SHOP: %s]", shop.ShopType))
				}
			}
			output.WriteString(ui.FormatListBullet(formatIP(server.IP) + " (" + formatIP(server.LocalIP) + ")" + shopIndicator))
		}
		return &CommandResult{Output: output.String()}
	}

	// Scan specific IP
	if len(args) == 1 {
		server, err := h.networkService.ScanIP(args[0])
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		// Log the scan (scans are detected by the target server)
		// Use effective source IP (the server we're on, or user's IP if local)
		if h.serverLogService != nil && h.user != nil {
			h.serverLogService.LogScan(server.IP, h.GetEffectiveSourceIP(), &h.user.ID)
		}

		// Format with colors and emojis
		var output strings.Builder
		output.WriteString(ui.FormatSectionHeader("Scan Results:", "🔍"))
		
		// IP addresses
		output.WriteString(ui.LabelStyle.Bold(true).Render("📍 IP:") + " " + formatIP(server.IP) + "\n")
		output.WriteString(ui.LabelStyle.Bold(true).Render("📍 Local IP:") + " " + formatIP(server.LocalIP) + "\n")
		
		// Security level with color
		secLevelStyle := ui.GetSecurityStyle(server.SecurityLevel)
		output.WriteString(ui.LabelStyle.Bold(true).Render("🔒 Security Level:") + " " + secLevelStyle.Render(fmt.Sprintf("%d", server.SecurityLevel)) + "\n")
		
		// Resources
		output.WriteString(ui.LabelStyle.Bold(true).Render("💻 Resources:") + " ")
		output.WriteString(ui.AccentStyle.Render(fmt.Sprintf("CPU=%d", server.Resources.CPU)) + ", ")
		output.WriteString(ui.AccentStyle.Render(fmt.Sprintf("Bandwidth=%.1f", server.Resources.Bandwidth)) + ", ")
		output.WriteString(ui.AccentStyle.Render(fmt.Sprintf("RAM=%d", server.Resources.RAM)) + "\n")
		
		// Wallet
		output.WriteString(ui.LabelStyle.Bold(true).Render("💰 Wallet:") + " ")
		output.WriteString(ui.PriceStyle.Render(fmt.Sprintf("Crypto=%.2f", server.Wallet.Crypto)) + ", ")
		output.WriteString(ui.PriceStyle.Render(fmt.Sprintf("Data=%.2f", server.Wallet.Data)) + "\n")
		
		// Tools: show if user has access (credentials/backdoor) or server needs no auth (e.g. home PC)
		if len(server.Tools) > 0 {
			serverPath := server.IP
			if h.currentServerPath != "" {
				serverPath = h.currentServerPath + ".localNetwork." + server.IP
			}
			hasAccess := false
			if h.credentialService != nil {
				hasAccess, _, _ = h.credentialService.CanAccessServer(h.user.ID, serverPath)
			}
			if !hasAccess && h.exploitationService != nil {
				hasAccess = h.exploitationService.CanAccessServer(h.user.ID, serverPath)
			}
			noAuthRequired := false
			for _, svc := range server.Services {
				if svc.RequiresAuth != nil && !*svc.RequiresAuth && svc.ServiceGrantsShellAccess() {
					noAuthRequired = true
					break
				}
			}
			if hasAccess {
				output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🛠️ Available Tools:") + "\n")
				output.WriteString(ui.ValueStyle.Render(fmt.Sprintf("  Use 'get %s <toolName>' to download", server.IP)) + "\n")
				for _, tool := range server.Tools {
					output.WriteString(ui.FormatListBulletWithStyle(tool, ui.AccentStyle))
				}
			} else if noAuthRequired {
				output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🛠️ Available Tools:") + "\n")
				for _, tool := range server.Tools {
					output.WriteString(ui.FormatListBulletWithStyle(tool, ui.AccentStyle))
				}
				output.WriteString(ui.DimStyle.Render("  Connect first to download (no password needed).\n"))
			} else {
				output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🛠️ Available Tools:") + "\n")
				output.WriteString(ui.DimStyle.Render("  Access required to view available tools.\n"))
			}
		}
		
		// Services
		if len(server.Services) > 0 {
			output.WriteString("\n" + ui.SectionStyle.Render("🌐 Services:") + "\n")
			for _, service := range server.Services {
				output.WriteString(ui.FormatListBulletWithStyle(
					fmt.Sprintf("%s (port %d): %s", service.Name, service.Port, service.Description),
					ui.InfoStyle,
				))
				if service.Vulnerable && len(service.Vulnerabilities) > 0 {
					output.WriteString("    " + ui.WarningStyle.Render("⚠️ Vulnerabilities:") + "\n")
					for _, vuln := range service.Vulnerabilities {
						// Check if this vulnerability has been exploited (is "open")
						isExploited := false
						if h.user != nil && h.exploitationService != nil {
							isExploited = h.exploitationService.IsVulnerabilityExploited(h.user.ID, args[0], service.Name, vuln.Type)
						}
						if isExploited {
							// Exploited vulnerabilities show in green with "OPEN" indicator
							output.WriteString("      " + ui.SuccessStyle.Render(fmt.Sprintf("✓ %s (level %d) [OPEN]", vuln.Type, vuln.Level)) + "\n")
						} else {
							// Unexploited vulnerabilities show in red with "CLOSED" indicator
							output.WriteString("      " + ui.ErrorStyle.Render(fmt.Sprintf("✗ %s (level %d) [CLOSED]", vuln.Type, vuln.Level)) + "\n")
						}
					}
				}
			}
		}
		
		// Calculate server path for access checking
		scanServerPath := args[0]
		if h.currentServerPath != "" {
			scanServerPath = h.currentServerPath + ".localNetwork." + args[0]
		}

		// Show discovered users
		if h.credentialService != nil && h.user != nil {
			discoveredUsers, _ := h.credentialService.GetDiscoveredUsers(h.user.ID, scanServerPath)
			if len(discoveredUsers) > 0 {
				output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("👥 Enumerated Users:") + "\n")
				for _, u := range discoveredUsers {
					output.WriteString(ui.FormatListBullet(
						ui.ValueStyle.Render(u.Username) + " " + ui.DimStyle.Render("("+u.Role+")"),
					))
				}
			}
		}

		// Show access status
		if h.credentialService != nil && h.user != nil {
			hasAccess, method, svc := h.credentialService.CanAccessServer(h.user.ID, scanServerPath)
			noAuthRequired := false
			for _, s := range server.Services {
				if s.RequiresAuth != nil && !*s.RequiresAuth && s.ServiceGrantsShellAccess() {
					noAuthRequired = true
					break
				}
			}
			if hasAccess {
				output.WriteString("\n" + ui.SuccessStyle.Render("✓ ACCESS AVAILABLE") + "\n")
				if method == "backdoor" {
					output.WriteString("  " + ui.InfoStyle.Render("Backdoor installed on "+svc+" (root access)") + "\n")
					output.WriteString("  " + ui.DimStyle.Render("Connect with: connect "+args[0]) + "\n")
				} else {
					creds, _ := h.credentialService.GetCredentialsForService(h.user.ID, scanServerPath, svc)
					if len(creds) > 0 {
						output.WriteString("  " + ui.InfoStyle.Render("Credentials available for "+svc) + "\n")
						output.WriteString("  " + ui.DimStyle.Render("Connect with: "+svc+" "+args[0]) + "\n")
					}
				}
			} else if noAuthRequired {
				output.WriteString("\n" + ui.SuccessStyle.Render("✓ No password needed (your computer)") + "\n")
				output.WriteString("  " + ui.DimStyle.Render("Connect with: connect "+args[0]) + "\n")
			} else {
				output.WriteString("\n" + ui.ErrorStyle.Render("✗ NO ACCESS") + "\n")
				output.WriteString("  " + ui.DimStyle.Render("Use user_enum + password_cracker, or ssh_exploit for SSH, or crack Telnet/FTP credentials") + "\n")
			}
		}

		// Local network hosts
		if len(server.LocalNetwork) > 0 {
			output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🔗 Local Network Hosts:") + "\n")
			for ip := range server.LocalNetwork {
				output.WriteString(ui.FormatListBullet(formatIP(ip)))
			}
		}
		
		// Shop
		if h.shopService != nil {
			if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
				output.WriteString("\n" + ui.AccentStyle.Render("🛒 Shop:") + " ")
				output.WriteString(ui.AccentStyle.Render(fmt.Sprintf("[%s] %s", shop.ShopType, shop.Name)) + "\n")
				output.WriteString(ui.ValueStyle.Render(fmt.Sprintf("  %s", shop.Description)) + "\n")
			}
		}
		
		return &CommandResult{Output: output.String()}
	}

	return &CommandResult{Error: fmt.Errorf("usage: scan [targetIP]")}
}

func (h *CommandHandler) handleSERVER() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("not connected to a server")}
	}

	server, err := h.serverService.GetServerByPath(h.currentServerPath)
	if err != nil {
		return &CommandResult{Error: err}
	}

	var output strings.Builder
	
	output.WriteString(ui.FormatSectionHeader("Server Information", "🖥️"))
	
	output.WriteString(ui.FormatKeyValuePair("Server:", formatIP(server.IP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Local IP:", formatIP(server.LocalIP)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Security Level:", fmt.Sprintf("%d", server.SecurityLevel)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Resources:", fmt.Sprintf("CPU=%d, Bandwidth=%.1f, RAM=%d",
		server.Resources.CPU, server.Resources.Bandwidth, server.Resources.RAM)) + "\n")
	output.WriteString(ui.FormatKeyValuePair("Wallet:", fmt.Sprintf("Crypto=%.2f, Data=%.2f",
		server.Wallet.Crypto, server.Wallet.Data)) + "\n")

	return &CommandResult{Output: output.String()}
}

func generateRandomIP() string {
	// Simple random IP generation
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(254)+1,
		rand.Intn(255),
		rand.Intn(255),
		rand.Intn(254)+1)
}

func generateRandomLocalIP() string {
	return fmt.Sprintf("10.%d.%d.%d",
		rand.Intn(255),
		rand.Intn(255),
		rand.Intn(254)+1)
}

// handleConnect handles connection to a server via a specific service or auto-detect.
// requiredService can be "" for auto-detect, or "ssh", "telnet", "ftp" etc.
func (h *CommandHandler) handleConnect(args []string, requiredService string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Determine command name for usage message
	cmdName := "connect"
	if requiredService != "" {
		cmdName = requiredService
	}

	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: %s <targetIP>", cmdName)}
	}

	targetIP := args[0]

	// Check if server exists - validate before starting progress
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		// Try to get by path (for nested servers)
		server, err = h.serverService.GetServerByPath(targetIP)
		if err != nil {
			return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
		}
	}

	// Build the server path for access check
	var serverPath string
	if h.currentServerPath == "" {
		serverPath = server.IP
	} else {
		serverPath = h.currentServerPath + ".localNetwork." + server.IP
	}

	// Check access using credential service (credentials or backdoor)
	var serviceType string
	var accessMethod string
	var accessUsername string

	// First, check if any service on this server requires no auth (e.g., your own PC)
	var noAuthService *models.Service
	for i := range server.Services {
		svc := &server.Services[i]
		if svc.RequiresAuth != nil && !*svc.RequiresAuth && svc.ServiceGrantsShellAccess() {
			// If a specific service was requested, only use it if it matches
			if requiredService == "" || svc.Name == requiredService {
				noAuthService = svc
				break
			}
		}
	}

	if noAuthService != nil {
		// No authentication required - this is your own server
		serviceType = noAuthService.Name
		accessMethod = "no_auth"
		accessUsername = h.user.Username // Use player's username
	} else if requiredService != "" {
		// User requested a specific service - check if we have access
		accessInfo := h.credentialService.GetAccessInfo(h.user.ID, serverPath, requiredService)
		if !accessInfo.HasAccess {
			// Check if the service exists on the server
			serviceExists := false
			for _, svc := range server.Services {
				if svc.Name == requiredService {
					serviceExists = true
					break
				}
			}
			if !serviceExists {
				return &CommandResult{Error: fmt.Errorf("%s service not available on %s", requiredService, targetIP)}
			}
			return &CommandResult{Error: fmt.Errorf("no access to %s on %s - need credentials or backdoor", requiredService, targetIP)}
		}
		serviceType = requiredService
		accessMethod = accessInfo.AccessMethod
		accessUsername = accessInfo.Username
	} else {
		// Auto-detect: find any accessible service
		hasAccess, method, svc := h.credentialService.CanAccessServer(h.user.ID, serverPath)
		if !hasAccess {
			return &CommandResult{Error: fmt.Errorf("no access to %s - use password_cracker or ssh_exploit first", targetIP)}
		}
		serviceType = svc
		accessMethod = method
		if method == "credentials" {
			if bestCred, err := h.credentialService.GetBestCredentialForService(h.user.ID, serverPath, svc); err == nil && bestCred != nil {
				accessUsername = bestCred.Username
			}
		}
	}

	// Calculate connection time based on user resources
	var duration float64 = 1.0 // default 1 second
	if h.progressService != nil {
		duration = h.progressService.CalculateOperationTime(services.OperationConnect, h.user.Resources)
	}

	// Return a progress operation that will run async
	operationID := fmt.Sprintf("connect-%s-%d", targetIP, time.Now().UnixNano())

	// Capture user info for logging
	userID := h.user.ID
	sourceIP := h.GetEffectiveSourceIP()
	username := h.user.Username
	capturedServiceType := serviceType
	capturedAccessMethod := accessMethod
	capturedAccessUsername := accessUsername
	capturedServerIP := server.IP // Capture server IP for async closure

	// Get the full role info for proper permissions
	var roleInfo *services.ConnectionRole
	if h.roleService != nil {
		roleInfo = h.roleService.GetConnectionRole(userID, serverPath, serviceType, server)
	}
	// Capture role info for async operation
	capturedRoleInfo := roleInfo

	// Build connection message based on access method
	var connectMsg string
	switch accessMethod {
	case "no_auth":
		connectMsg = fmt.Sprintf("Connecting to %s via %s (no auth required)...", targetIP, serviceType)
	case "backdoor":
		if roleInfo != nil && roleInfo.IsRoot {
			connectMsg = fmt.Sprintf("Connecting to %s via %s (backdoor → root)...", targetIP, serviceType)
		} else {
			connectMsg = fmt.Sprintf("Connecting to %s via %s (backdoor)...", targetIP, serviceType)
		}
	default:
		connectMsg = fmt.Sprintf("Authenticating to %s via %s as %s...", targetIP, serviceType, accessUsername)
	}

	// Capture action tracker and handler for async closure
	actionTracker := h.actionTracker
	handler := h
	serverLogService := h.serverLogService

	return &CommandResult{
		StartProgress: &ProgressOperationRequest{
			ID:       operationID,
			Message:  connectMsg,
			Duration: duration,
			Operation: func() *CommandResult {
				// Log the connection with service type
				if serverLogService != nil {
					serverLogService.LogConnect(capturedServerIP, sourceIP, username, &userID, capturedServiceType, true)
				}
				
				// Track server connection for mission objectives
				if actionTracker != nil {
					actionTracker.TrackServerConnect(userID, capturedServerIP, capturedServiceType)
				}

				// Check for auto-completed missions
				var missionCompleted *services.MissionCompletionResult
				if missionService := handler.missionService; missionService != nil {
					missionCompleted = missionService.TryAutoComplete(userID)
				}
				
				// Store role info and service type for the connection
				handler.currentRole = capturedRoleInfo
				handler.SetCurrentServiceType(capturedServiceType)
				
				// Return special marker for shell to handle stack push
				// Format: __CONNECT__<serviceType>:<accessMethod>:<accessUsername>:<isRoot>:<homeDir>:<serverPath>
				isRoot := "0"
				homeDir := "/home/user"
				if capturedRoleInfo != nil {
					if capturedRoleInfo.IsRoot {
						isRoot = "1"
					}
					homeDir = capturedRoleInfo.HomeDir
				}
				result := &CommandResult{
					Output:            fmt.Sprintf("__CONNECT__%s:%s:%s:%s:%s:%s", capturedServiceType, capturedAccessMethod, capturedAccessUsername, isRoot, homeDir, serverPath),
					MissionCompleted: missionCompleted,
				}
				return result
			},
		},
	}
}

func (h *CommandHandler) handleEXIT() *CommandResult {
	if h.currentServerPath == "" {
		// Not in SSH session - return special marker to quit
		return &CommandResult{Output: "__QUIT__"}
	}

	// Log disconnection with service type
	if h.serverLogService != nil && h.user != nil {
		// Extract current server IP from path
		parts := strings.Split(h.currentServerPath, ".")
		serverIP := parts[len(parts)-1]
		
		// The source IP for disconnect is the previous hop (where we came from)
		// If path is "A.localNetwork.B", we're on B and came from A
		// If path is just "A", we came from our local machine (user's IP)
		var sourceIP string
		if len(parts) >= 3 {
			// We have at least one hop: extract the previous server IP
			// Path format: "ip1.localNetwork.ip2.localNetwork.ip3"
			// Parts would be: [ip1, localNetwork, ip2, localNetwork, ip3]
			// Previous hop is parts[len(parts)-3]
			sourceIP = parts[len(parts)-3]
		} else {
			// Direct connection from user's machine
			sourceIP = h.user.IP
		}
		
		h.serverLogService.LogDisconnect(serverIP, sourceIP, h.user.Username, &h.user.ID, h.GetCurrentServiceType())
	}

	// Return special marker for shell to handle stack pop
	return &CommandResult{Output: "__EXIT_CONNECT__"}
}

// isPrivateIP checks if the given IP address is in a private network range
// Note: Duplicated from terminal/renderer.go due to import cycle
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	// Check private IPv4 ranges:
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}
	
	for _, block := range privateBlocks {
		_, ipNet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	
	return false
}

// FormatMissionCompletion formats auto-completed mission rewards for display (exported for shell)
func FormatMissionCompletion(completion *services.MissionCompletionResult) string {
	if completion == nil || completion.Mission == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n" + ui.SuccessStyle.Render("🎉 Mission completed: ") + completion.Mission.Name + "\n")
	rewards := completion.Mission.Rewards
	if rewards.Experience > 0 {
		b.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("+%d XP", rewards.Experience))))
	}
	if rewards.Crypto > 0 {
		b.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("+%.2f cryptocurrency", rewards.Crypto))))
	}
	if len(rewards.Tools) > 0 {
		b.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Tools unlocked: %s", strings.Join(rewards.Tools, ", ")))))
	}
	if len(rewards.ToolUpgrades) > 0 {
		upgrades := []string{}
		for _, u := range rewards.ToolUpgrades {
			upgrades = append(upgrades, fmt.Sprintf("%s +%d %s", u.ToolName, u.Count, u.UpgradeType))
		}
		b.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Tool upgrades: %s", strings.Join(upgrades, ", ")))))
	}
	if len(rewards.Achievements) > 0 {
		b.WriteString(ui.FormatListBullet(ui.SuccessStyleNoBold.Render(fmt.Sprintf("Achievements: %s", strings.Join(rewards.Achievements, ", ")))))
	}
	if len(completion.Mission.Unlocks) > 0 {
		b.WriteString("\n" + ui.FormatKeyValuePair("New missions unlocked:", strings.Join(completion.Mission.Unlocks, ", ")) + "\n")
	}
	return b.String()
}

// appendMissionCompletionIfAny checks for auto-completed missions and appends to output
func (h *CommandHandler) appendMissionCompletionIfAny(output string) string {
	if h.user == nil || h.missionService == nil {
		return output
	}
	if completion := h.missionService.TryAutoComplete(h.user.ID); completion != nil {
		return output + FormatMissionCompletion(completion)
	}
	return output
}

// formatIP formats an IP address for display with color
// Local IPs (private network) are styled in yellow/orange, internet IPs in cyan
// Uses ui.FormatIP to avoid duplication
func formatIP(ip string) string {
	return ui.FormatIP(ip)
}

func parseServerPathParts(path string) []string {
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

func (h *CommandHandler) handleGET(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	var targetIP, toolName string

	if len(args) == 1 {
		// Single arg: get <toolName> - use current server if connected
		if h.currentServerPath == "" {
			return &CommandResult{Error: fmt.Errorf("usage: get <targetIP> <toolName>\n       or connect to a server first and use: get <toolName>")}
		}
		// Extract current server IP from path (last component)
		parts := strings.Split(h.currentServerPath, ".")
		// Handle paths like "server1" or "server1.localNetwork.server2"
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "localNetwork" {
				targetIP = parts[i]
				break
			}
		}
		if targetIP == "" {
			targetIP = h.currentServerPath
		}
		toolName = args[0]
	} else if len(args) == 2 {
		targetIP = args[0]
		toolName = args[1]
	} else {
		return &CommandResult{Error: fmt.Errorf("usage: get <targetIP> <toolName>\n       or when connected: get <toolName>")}
	}

	// Check access: must be connected to this server, or have credentials/backdoor (repo is always allowed)
	alreadyConnected := h.currentServerPath != "" && strings.Contains(h.currentServerPath, targetIP)

	if targetIP != "repo" && !alreadyConnected && h.credentialService != nil {
		serverPath := targetIP
		if h.currentServerPath != "" {
			serverPath = h.currentServerPath + ".localNetwork." + targetIP
		}
		hasAccess, _, _ := h.credentialService.CanAccessServer(h.user.ID, serverPath)
		if !hasAccess && h.exploitationService != nil {
			hasAccess = h.exploitationService.CanAccessServer(h.user.ID, serverPath)
		}
		if !hasAccess {
			server, _ := h.serverService.GetServerByIP(targetIP)
			noAuthRequired := false
			if server != nil {
				for _, svc := range server.Services {
					if svc.RequiresAuth != nil && !*svc.RequiresAuth && svc.ServiceGrantsShellAccess() {
						noAuthRequired = true
						break
					}
				}
			}
			if noAuthRequired {
				return &CommandResult{Error: fmt.Errorf("connect to %s first to download tools (no password needed)", targetIP)}
			}
			return &CommandResult{Error: fmt.Errorf("access required to download tools from %s", targetIP)}
		}
	}

	// Calculate download time based on user resources
	var duration float64 = 2.0 // default 2 seconds
	if h.progressService != nil {
		duration = h.progressService.CalculateOperationTime(services.OperationDownload, h.user.Resources)
	}

	// Return a progress operation that will run async
	operationID := fmt.Sprintf("download-%s-%s-%d", targetIP, toolName, time.Now().UnixNano())
	
	// Capture variables for the closure
	toolService := h.toolService
	userID := h.user.ID
	handler := h
	actionTracker := h.actionTracker
	capturedToolName := toolName
	capturedTargetIP := targetIP
	
	return &CommandResult{
		StartProgress: &ProgressOperationRequest{
			ID:       operationID,
			Message:  fmt.Sprintf("Downloading %s from %s...", toolName, targetIP),
			Duration: duration,
			Operation: func() *CommandResult {
				if err := toolService.DownloadTool(userID, capturedTargetIP, capturedToolName); err != nil {
					return &CommandResult{Error: err}
				}
				
				// Track tool download for mission objectives
				if actionTracker != nil {
					actionTracker.TrackToolDownload(userID, capturedToolName, capturedTargetIP)
				}

				// Check for auto-completed missions
				var missionCompleted *services.MissionCompletionResult
				if missionService := handler.missionService; missionService != nil {
					missionCompleted = missionService.TryAutoComplete(userID)
				}

				// Sync tools to VFS so the new tool appears in help
				handler.SyncUserToolsToVFS()
				
				output := ui.SuccessStyle.Render("✅ Tool ") + ui.AccentBoldStyle.Render(capturedToolName) + ui.SuccessStyle.Render(" downloaded successfully from ") + formatIP(capturedTargetIP) + "\n"
				return &CommandResult{Output: output, MissionCompleted: missionCompleted}
			},
		},
	}
}

func (h *CommandHandler) handleTOOLS() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Get user's tool states (shows versions and patches)
	var toolStates []models.UserToolState
	if err := h.db.Where("user_id = ?", h.user.ID).Preload("Tool").Find(&toolStates).Error; err != nil {
		// Fallback to old method if no tool states
		tools, err := h.toolService.GetUserTools(h.user.ID)
		if err != nil {
			return &CommandResult{Error: err}
		}

		if len(tools) == 0 {
		return &CommandResult{Output: ui.WarningStyle.Render("ℹ️  No tools owned") + "\n"}
		}

		var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Owned tools:", "🛠️"))
		for _, tool := range tools {
		output.WriteString(ui.FormatListBulletWithStyle("• "+tool.Name+": "+tool.Function, ui.AccentStyle))
			if len(tool.Exploits) > 0 {
			output.WriteString("    " + ui.ErrorStyle.Render("⚡ Exploits: "))
				for i, exploit := range tool.Exploits {
					if i > 0 {
						output.WriteString(", ")
					}
				output.WriteString(ui.ErrorStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
				}
				output.WriteString("\n")
			}
		}
		return &CommandResult{Output: output.String()}
	}

	if len(toolStates) == 0 {
		return &CommandResult{Output: ui.WarningStyle.Render("ℹ️  No tools owned") + "\n"}
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Owned tools:", "🛠️"))
	for _, toolState := range toolStates {
		tool := toolState.Tool
		output.WriteString(ui.FormatListBulletWithStyle("• "+tool.Name+": "+tool.Function, ui.AccentStyle))
		output.WriteString("    " + ui.InfoStyle.Render("📦 Version:") + " " + ui.InfoStyle.Render(fmt.Sprintf("%d", toolState.Version)) + "\n")

		// Show upgrade counts if any
		totalUpgrades := toolState.ExploitUpgrades + toolState.CPUUpgrades + toolState.RAMUpgrades + toolState.BandwidthUpgrades
		if totalUpgrades > 0 {
			upgrades := []string{}
			if toolState.ExploitUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d exploit", toolState.ExploitUpgrades))
			}
			if toolState.CPUUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d cpu", toolState.CPUUpgrades))
			}
			if toolState.RAMUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d ram", toolState.RAMUpgrades))
			}
			if toolState.BandwidthUpgrades > 0 {
				upgrades = append(upgrades, fmt.Sprintf("%d bw", toolState.BandwidthUpgrades))
			}
			output.WriteString("    " + ui.AccentBoldStyle.Render("🔧 Upgrades:") + " " + ui.AccentBoldStyle.Render(strings.Join(upgrades, ", ")) + "\n")
		}
		
		if len(toolState.EffectiveExploits) > 0 {
			output.WriteString("    " + ui.ErrorStyle.Render("⚡ Effective Exploits: "))
			for i, exploit := range toolState.EffectiveExploits {
				if i > 0 {
					output.WriteString(", ")
				}
				output.WriteString(ui.ErrorStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
			}
			output.WriteString("\n")
		}
	}

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleEXPLOITED() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Show both old exploited servers AND new credentials/backdoors
	var output strings.Builder
	hasContent := false

	// Show backdoors (direct shell access)
	backdoors, _ := h.credentialService.GetAllBackdoors(h.user.ID)
	if len(backdoors) > 0 {
		hasContent = true
		output.WriteString(ui.HeaderStyle.Render("Backdoor Access (Direct Shell)") + "\n")
		for _, bd := range backdoors {
			output.WriteString(ui.FormatListBullet(
				ui.AccentStyle.Render(bd.ServerPath) + " via " + 
				ui.InfoStyle.Render(bd.ServiceName) + " " +
				ui.SuccessStyle.Render("("+bd.AccessLevel+" access)") + " " +
				ui.DimStyle.Render("["+bd.ExploitType+"]"),
			))
		}
		output.WriteString("\n")
	}

	// Show credentials
	creds, _ := h.credentialService.GetAllCredentials(h.user.ID)
	if len(creds) > 0 {
		hasContent = true
		output.WriteString(ui.HeaderStyle.Render("Credentials (Password Access)") + "\n")
		// Group by server
		serverCreds := make(map[string][]models.DiscoveredCredential)
		for _, c := range creds {
			serverCreds[c.ServerPath] = append(serverCreds[c.ServerPath], c)
		}
		for server, credList := range serverCreds {
			output.WriteString(ui.AccentStyle.Render("  "+server) + "\n")
			for _, c := range credList {
				output.WriteString(fmt.Sprintf("    %s: %s : %s %s\n",
					ui.InfoStyle.Render(c.ServiceName),
					ui.ValueStyle.Render(c.Username),
					ui.WarningStyle.Render(c.Password),
					ui.DimStyle.Render("("+c.Role+")"),
				))
			}
		}
		output.WriteString("\n")
	}

	// Legacy: show old exploited servers (for backward compatibility)
	exploited, err := h.exploitationService.GetExploitedServers(h.user.ID)
	if err == nil && len(exploited) > 0 {
		hasContent = true
		output.WriteString(ui.HeaderStyle.Render("Legacy Exploits") + "\n")
		for _, exp := range exploited {
			output.WriteString(ui.FormatListBullet(ui.AccentStyle.Render(exp.ServerPath) + " (" + ui.InfoStyle.Render(exp.ServiceName) + ")"))
		}
	}

	if !hasContent {
		return &CommandResult{Output: ui.WarningStyle.Render("No access to any servers yet. Use password_cracker or ssh_exploit first.") + "\n"}
	}

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleCREDENTIALS(_ []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("Discovered Credentials") + "\n\n")

	// Get all credentials
	creds, err := h.credentialService.GetAllCredentials(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(creds) == 0 {
		output.WriteString(ui.WarningStyle.Render("No credentials discovered yet.") + "\n")
		output.WriteString(ui.DimStyle.Render("Tip: Use user_enum to find users, then password_cracker to crack passwords.") + "\n")
		return &CommandResult{Output: output.String()}
	}

	// Group by server
	serverCreds := make(map[string][]models.DiscoveredCredential)
	for _, c := range creds {
		serverCreds[c.ServerPath] = append(serverCreds[c.ServerPath], c)
	}

	for server, credList := range serverCreds {
		output.WriteString(ui.InfoStyle.Render("Server: ") + ui.AccentStyle.Render(server) + "\n")
		for _, c := range credList {
			output.WriteString(fmt.Sprintf("  %s  %s : %s  %s  %s\n",
				ui.DimStyle.Render("["+c.ServiceName+"]"),
				ui.ValueStyle.Render(c.Username),
				ui.WarningStyle.Render(c.Password),
				ui.DimStyle.Render("("+c.Role+")"),
				ui.DimStyle.Render("["+string(c.Type)+"]"),
			))
		}
		output.WriteString("\n")
	}

	output.WriteString(ui.DimStyle.Render(fmt.Sprintf("Total: %d credential(s)", len(creds))) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleBACKDOORS() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	var output strings.Builder
	output.WriteString(ui.HeaderStyle.Render("Installed Backdoors") + "\n\n")

	backdoors, err := h.credentialService.GetAllBackdoors(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(backdoors) == 0 {
		output.WriteString(ui.WarningStyle.Render("No backdoors installed yet.") + "\n")
		output.WriteString(ui.DimStyle.Render("Tip: Use ssh_exploit or exploit_kit on servers with RCE vulnerabilities.") + "\n")
		return &CommandResult{Output: output.String()}
	}

	for _, bd := range backdoors {
		output.WriteString(ui.InfoStyle.Render("Server: ") + ui.AccentStyle.Render(bd.ServerPath) + "\n")
		output.WriteString(fmt.Sprintf("  Service: %s\n", ui.ValueStyle.Render(bd.ServiceName)))
		output.WriteString(fmt.Sprintf("  Access:  %s\n", ui.SuccessStyle.Render(bd.AccessLevel)))
		output.WriteString(fmt.Sprintf("  Exploit: %s\n", ui.DimStyle.Render(bd.ExploitType)))
		output.WriteString(fmt.Sprintf("  Tool:    %s\n", ui.DimStyle.Render(bd.ToolUsed)))
		output.WriteString("\n")
	}

	output.WriteString(ui.DimStyle.Render(fmt.Sprintf("Total: %d backdoor(s)", len(backdoors))) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleCRYPTOMINER(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: crypto_miner <targetIP>")}
	}

	serverIP := args[0]

	if err := h.miningService.StartMining(h.user.ID, serverIP); err != nil {
		return &CommandResult{Error: err}
	}

	output := fmt.Sprintf("Mining started on server %s\n", serverIP)
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleSTOPMINING(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: stop_mining <targetIP>")}
	}

	serverIP := args[0]

	if err := h.miningService.StopMining(h.user.ID, serverIP); err != nil {
		return &CommandResult{Error: err}
	}

	output := ui.SuccessStyle.Render("✅ Mining stopped on server ") + formatIP(serverIP) + "\n"
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleMINERS() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	miners, err := h.miningService.GetActiveMiners(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(miners) == 0 {
		return &CommandResult{Output: ui.WarningStyle.Render("ℹ️  No active miners") + "\n"}
	}

	var output strings.Builder
	output.WriteString(ui.WarningStyle.Render("⛏️  Active miners:") + "\n")
	for _, miner := range miners {
		duration := time.Since(miner.StartTime)
		output.WriteString(ui.FormatListBullet(ui.AccentStyle.Render("• Server:") + " " + formatIP(miner.ServerIP) + " " + ui.ValueStyle.Render("(running for "+duration.Round(time.Second).String()+")")))
		output.WriteString("    " + ui.InfoStyle.Render("💻 Resources:") + " ")
		output.WriteString(ui.InfoStyle.Render(fmt.Sprintf("CPU=%.1f", miner.ResourceUsage.CPU)) + ", ")
		output.WriteString(ui.InfoStyle.Render(fmt.Sprintf("Bandwidth=%.1f", miner.ResourceUsage.Bandwidth)) + ", ")
		output.WriteString(ui.InfoStyle.Render(fmt.Sprintf("RAM=%d", miner.ResourceUsage.RAM)) + "\n")
	}

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleWALLET() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Refresh user data
	user, err := h.userService.GetUserByID(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	var output strings.Builder
	output.WriteString(ui.FormatSectionHeader("Wallet Balance:", "💰"))
	output.WriteString("  " + ui.FormatKeyValuePair("₿ Crypto:", fmt.Sprintf("%.2f", user.Wallet.Crypto)) + "\n")
	output.WriteString("  " + ui.FormatKeyValuePair("💾 Data:", fmt.Sprintf("%.2f", user.Wallet.Data)) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleTOUCH(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: touch <filename>")}
	}
	
	if err := h.vfs.CreateFile(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ui.SuccessStyle.Render("📄 Created file: ") + ui.ValueStyle.Render(args[0]) + "\n"}
}

func (h *CommandHandler) handleMKDIR(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: mkdir <dirname>")}
	}
	
	if err := h.vfs.CreateDirectory(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ui.SuccessStyle.Render("📁 Created directory: ") + ui.ListStyle.Render(args[0]) + "\n"}
}

func (h *CommandHandler) handleRM(args []string) *CommandResult {
	if len(args) < 1 {
		return &CommandResult{Error: fmt.Errorf("usage: rm [-r] <filename>")}
	}
	
	recursive := false
	filename := args[0]
	
	if len(args) == 2 && args[0] == "-r" {
		recursive = true
		filename = args[1]
	}
	
	if err := h.vfs.DeleteNode(filename, recursive); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ui.SuccessStyle.Render("🗑️  Deleted: ") + ui.ValueStyle.Render(filename) + "\n"}
}

func (h *CommandHandler) handleDOWNLOAD(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: download <path>\n       When connected to a server, downloads the file to ~/Downloads/")}
	}

	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("download: must be connected to a server first (use ssh, telnet, or connect)")}
	}

	filePath := args[0]
	absPath := filePath
	if !strings.HasPrefix(filePath, "/") {
		currentPath := h.vfs.GetCurrentPath()
		if currentPath == "/" {
			absPath = "/" + filePath
		} else {
			absPath = currentPath + "/" + filePath
		}
	}
	absPath = filepath.Clean(absPath)

	// Handle dynamic log files (same as cat)
	parts := strings.Split(h.currentServerPath, ".")
	serverIP := parts[len(parts)-1]
	var content string
	switch {
	case strings.HasSuffix(absPath, "/var/log/auth.log") || absPath == "/var/log/auth.log":
		content = h.getDynamicAuthLog(serverIP, filePath)
	case strings.HasSuffix(absPath, "/var/log/system.log") || absPath == "/var/log/system.log":
		content = h.getDynamicSystemLog(serverIP, filePath)
	default:
		var err error
		content, err = h.vfs.ReadFileAtPath(filePath)
		if err != nil {
			return &CommandResult{Error: fmt.Errorf("download: %w", err)}
		}
	}

	fileName := filepath.Base(filePath)
	if fileName == "." || fileName == ".." || fileName == "" {
		return &CommandResult{Error: fmt.Errorf("download: invalid file path: %s", filePath)}
	}

	username := "user"
	if h.user != nil && h.user.Username != "" {
		username = h.user.Username
	}
	downloadsDir := "/home/" + username + "/Downloads"

	// Calculate transfer time based on user resources (bandwidth, CPU, RAM)
	var duration float64 = 3.0
	if h.progressService != nil && h.user != nil {
		duration = h.progressService.CalculateOperationTime(services.OperationTransfer, h.user.Resources)
	}

	// Return progress operation - actual write happens after progress bar completes
	operationID := fmt.Sprintf("download-%s-%d", fileName, time.Now().UnixNano())
	capturedContent := content
	capturedFileName := fileName
	capturedDownloadsDir := downloadsDir
	capturedServerIP := serverIP
	homeVFS := h.homeVFS

	return &CommandResult{
		StartProgress: &ProgressOperationRequest{
			ID:       operationID,
			Message:  fmt.Sprintf("Transferring %s from %s...", fileName, formatIP(serverIP)),
			Duration: duration,
			Operation: func() *CommandResult {
				if err := homeVFS.EnsureDirectoryAndCreateFile(capturedDownloadsDir, capturedFileName, capturedContent); err != nil {
					return &CommandResult{Error: fmt.Errorf("download: %w", err)}
				}
				output := ui.SuccessStyle.Render("📥 Downloaded: ") + ui.ValueStyle.Render(capturedFileName) + ui.SuccessStyle.Render(" from ") + formatIP(capturedServerIP) + ui.SuccessStyle.Render(" → ~/Downloads/") + capturedFileName + "\n"
				return &CommandResult{Output: output}
			},
		},
	}
}

func (h *CommandHandler) handleCP(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: cp <src> <dest>")}
	}
	
	if err := h.vfs.CopyNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ui.SuccessStyle.Render("📋 Copied: ") + ui.ValueStyle.Render(args[0]) + " → " + ui.ValueStyle.Render(args[1]) + "\n"}
}

func (h *CommandHandler) handleMV(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: mv <src> <dest>")}
	}
	
	if err := h.vfs.MoveNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ui.SuccessStyle.Render("✂️  Moved: ") + ui.ValueStyle.Render(args[0]) + " → " + ui.ValueStyle.Render(args[1]) + "\n"}
}

func (h *CommandHandler) handleEDIT(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: edit <filename>")}
	}
	
	// Return special marker to enter edit mode
	// The shell will handle this and enter edit mode
	return &CommandResult{Output: fmt.Sprintf("__EDIT_MODE__%s", args[0])}
}

func (h *CommandHandler) handleTUTORIAL(args []string) *CommandResult {
	if h.tutorialService == nil {
		return &CommandResult{Error: fmt.Errorf("tutorial service not available")}
	}

	// Reload tutorials in case they were edited
	if err := h.tutorialService.ReloadTutorials(); err != nil {
		return &CommandResult{Error: fmt.Errorf("failed to reload tutorials: %w", err)}
	}

	// Emoji constants for tutorials
	const (
		emojiTutorial   = "📚"
		emojiBook       = "📖"
		emojiStar       = "⭐"
		emojiRocket     = "🚀"
		emojiLightbulb  = "💡"
		emojiCheck      = "✅"
		emojiArrow      = "➡️"
		emojiSparkles   = "✨"
		emojiGraduation = "🎓"
		emojiSteps      = "👣"
		emojiCode       = "💻"
		emojiTarget     = "🎯"
	)

	// If no args, list all tutorials
	if len(args) == 0 {
		tutorials := h.tutorialService.GetAllTutorials()
		
		var output strings.Builder
		
		// Title with lots of emojis and colors
		output.WriteString(ui.HeaderStyle.Render(emojiSparkles + " " + emojiTutorial + " Available Tutorials " + emojiTutorial + " " + emojiSparkles) + "\n\n")
		
		if len(tutorials) == 0 {
			output.WriteString(ui.WarningStyle.Render(emojiLightbulb+" No tutorials available yet.") + "\n")
			output.WriteString(ui.WarningStyle.Render("Edit tutorials.json to add tutorials.") + "\n")
		} else {
			for i, tutorial := range tutorials {
				// Tutorial ID with emoji and color
				output.WriteString(fmt.Sprintf("  %s %s %s\n", emojiStar, ui.AccentBoldStyle.Render(tutorial.ID), ui.InfoStyle.Bold(true).Render("- "+tutorial.Name)))
				
				// Description with emoji
				output.WriteString(fmt.Sprintf("    %s %s\n", emojiBook, ui.ValueStyle.Render(tutorial.Description)))
				
				if len(tutorial.Prerequisites) > 0 {
					prereqText := strings.Join(tutorial.Prerequisites, ", ")
					output.WriteString(fmt.Sprintf("    %s %s\n", emojiTarget, ui.WarningStyle.Render("Prerequisites: "+prereqText)))
				}
				
				output.WriteString(fmt.Sprintf("    %s %s\n", emojiSteps, ui.SuccessStyle.Render(fmt.Sprintf("Steps: %d", len(tutorial.Steps)))))
				
				if i < len(tutorials)-1 {
					output.WriteString("\n")
				}
			}
			
			output.WriteString("\n")
			output.WriteString(ui.SectionStyle.Render(emojiArrow + " Usage: ") + "tutorial <tutorial_id>\n")
			output.WriteString(ui.AccentBoldStyle.Render(emojiRocket + " Example: ") + "tutorial getting_started\n")
		}
		
		return &CommandResult{Output: output.String()}
	}

	// Get specific tutorial
	tutorialID := args[0]
	tutorial, err := h.tutorialService.GetTutorialByID(tutorialID)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("tutorial not found: %s. Use 'tutorial' to list available tutorials", tutorialID)}
	}

	// Display tutorial with lots of emojis and colors
	var output strings.Builder
	
	// Boxed title with emojis
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.DefaultTheme.Primary)).
		Padding(0, 1).
		Width(50)
	// Note: boxStyle is kept inline as it's a specific layout style, not a reusable semantic style
	
	titleContent := ui.HeaderStyle.Render(emojiGraduation + " Tutorial: " + tutorial.Name + " " + emojiGraduation)
	boxedTitle := boxStyle.Render(titleContent)
	output.WriteString(boxedTitle + "\n\n")
	
	// Description with emoji
	output.WriteString(ui.InfoStyle.Bold(true).Render(emojiBook + " " + tutorial.Description) + "\n\n")
	
	// Prerequisites section
	if len(tutorial.Prerequisites) > 0 {
		output.WriteString(ui.WarningStyle.Render(emojiTarget + " Prerequisites:") + "\n")
		for _, prereq := range tutorial.Prerequisites {
			output.WriteString(fmt.Sprintf("  %s %s\n", emojiCheck, ui.ValueStyle.Render(prereq)))
		}
		output.WriteString("\n")
	}

	// Steps section
	output.WriteString(ui.SuccessStyle.Render(emojiSparkles + " Steps:") + "\n\n")
	
	for i, step := range tutorial.Steps {
		output.WriteString(ui.AccentBoldStyle.Render(fmt.Sprintf("%s Step %d:", emojiSteps, step.ID)))
		output.WriteString(" " + ui.InfoStyle.Bold(true).Render(step.Title) + "\n")
		output.WriteString(ui.ValueStyle.Render(step.Description) + "\n")
		
		if len(step.Commands) > 0 {
			output.WriteString(ui.SectionStyle.Render("  " + emojiCode + " Example commands:") + "\n")
			for _, cmd := range step.Commands {
				output.WriteString(fmt.Sprintf("    $ %s\n", ui.SuccessStyleNoBold.Render(cmd)))
			}
		}
		
		if i < len(tutorial.Steps)-1 {
			output.WriteString("\n")
		}
	}
	
	output.WriteString("\n")
	output.WriteString(ui.GrayStyle.Render(emojiLightbulb+" Tutorial file location: "+h.tutorialService.GetTutorialPath()) + "\n")
	output.WriteString(ui.GrayStyle.Render("Edit this file to modify tutorials.") + "\n")

	return &CommandResult{Output: output.String()}
}

// GetTutorialNames returns tutorial IDs that start with the given prefix (for autocomplete)
func (h *CommandHandler) GetTutorialNames(prefix string) []string {
	if h.tutorialService == nil {
		return []string{}
	}
	
	// Reload tutorials to get latest
	if err := h.tutorialService.ReloadTutorials(); err != nil {
		return []string{}
	}
	
	tutorials := h.tutorialService.GetAllTutorials()
	var matches []string
	for _, tutorial := range tutorials {
		if strings.HasPrefix(tutorial.ID, prefix) {
			matches = append(matches, tutorial.ID)
		}
	}
	return matches
}

// GetUserToolNames returns the names of all tools owned by the current user (for autocomplete)
func (h *CommandHandler) GetUserToolNames() ([]string, error) {
	if h.user == nil || h.toolService == nil {
		return []string{}, nil
	}
	
	tools, err := h.toolService.GetUserTools(h.user.ID)
	if err != nil {
		return []string{}, err
	}
	
	var toolNames []string
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name)
	}
	return toolNames, nil
}

// GetMissionMatches returns mission subcommands and IDs that start with the given prefix (for autocomplete)
func (h *CommandHandler) GetMissionMatches(prefix string) []string {
	if h.missionService == nil || h.user == nil {
		return []string{}
	}
	subcommands := []string{"start", "stop", "status", "list"}
	var matches []string
	for _, sub := range subcommands {
		if strings.HasPrefix(sub, prefix) {
			matches = append(matches, sub)
		}
	}
	available := h.missionService.GetAvailableMissions(h.user.ID, h.user.Level)
	for _, m := range available {
		if strings.HasPrefix(m.ID, prefix) {
			matches = append(matches, m.ID)
		}
	}
	return matches
}

func parseCommand(input string) []string {
	// Simple command parsing - split by spaces
	parts := []string{}
	current := ""
	inQuotes := false

	for _, char := range input {
		if char == '"' || char == '\'' {
			inQuotes = !inQuotes
		} else if char == ' ' && !inQuotes {
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

