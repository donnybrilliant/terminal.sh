// Package cmd provides command handlers for terminal commands executed in the game shell.
package cmd

import (
	"fmt"
	"math/rand"
	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"
	"terminal-sh/services"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CommandResult represents the result of executing a command, including output, errors, and optional filesystem nodes.
type CommandResult struct {
	Output     string
	Error      error
	Nodes      []*filesystem.Node // For ls command
	LongFormat bool                // For ls -l format
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
	patchService   *services.PatchService
	progressService *services.ProgressService
	chatService    *services.ChatService
	currentServerPath string // Current server path if in SSH mode
	sessionID       *uuid.UUID // Current session ID
	onSSHConnect    func(serverPath string) error // Callback for SSH connection
	onSSHDisconnect func() error // Callback for SSH disconnection
}

// NewCommandHandler creates a new CommandHandler with the provided dependencies.
// Initializes all required services for command execution.
func NewCommandHandler(db *database.Database, vfs *filesystem.VFS, user *models.User, userService *services.UserService, chatService *services.ChatService) *CommandHandler {
	serverService := services.NewServerService(db)
	toolService := services.NewToolService(db, serverService)
	patchService := services.NewPatchService(db, toolService)
	toolService.SetPatchService(patchService) // Link patch service to tool service
	networkService := services.NewNetworkService(serverService)
	shopService := services.NewShopService(db, serverService)
	networkService.SetShopService(shopService) // Link shop service to network service
	shopDiscovery := services.NewShopDiscovery(shopService, serverService, patchService, toolService)
	progressService := services.NewProgressService()
	sessionService := services.NewSessionService(db, serverService)
	exploitationService := services.NewExploitationService(db, toolService, serverService)
	miningService := services.NewMiningService(db, toolService, serverService)
	tutorialService, _ := services.NewTutorialService("") // Initialize tutorial service with default path (data/seed/tutorials.json), ignore error for now
	
	return &CommandHandler{
		db:              db,
		vfs:            vfs,
		user:           user,
		userService:    userService,
		serverService:  serverService,
		networkService: networkService,
		sessionService: sessionService,
		toolService:   toolService,
		exploitationService: exploitationService,
		miningService: miningService,
		tutorialService: tutorialService,
		shopService:   shopService,
		shopDiscovery: shopDiscovery,
		patchService:  patchService,
		progressService: progressService,
		chatService:   chatService,
	}
}

// SetSessionID sets the current session ID for this command handler.
func (h *CommandHandler) SetSessionID(sessionID uuid.UUID) {
	h.sessionID = &sessionID
}

// SetSSHCallbacks sets callbacks for SSH connect and disconnect events.
func (h *CommandHandler) SetSSHCallbacks(onConnect func(serverPath string) error, onDisconnect func() error) {
	h.onSSHConnect = onConnect
	h.onSSHDisconnect = onDisconnect
}

// GetCurrentServerPath returns the current server path (empty if on user's local system).
func (h *CommandHandler) GetCurrentServerPath() string {
	return h.currentServerPath
}

// SetCurrentServerPath sets the current server path (used for restoring from session stack).
func (h *CommandHandler) SetCurrentServerPath(path string) {
	h.currentServerPath = path
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
	case "createServer":
		return h.handleCREATESERVER(args)
	case "createLocalServer":
		return h.handleCREATELOCALSERVER(args)
	case "ssh":
		return h.handleSSH(args)
	case "exit":
		return h.handleEXIT()
	case "get":
		return h.handleGET(args)
	case "tools":
		return h.handleTOOLS()
	case "exploited":
		return h.handleEXPLOITED()
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
	case "stop_mining":
		return h.handleSTOPMINING(args)
	case "miners":
		return h.handleMINERS()
	case "wallet":
		return h.handleWALLET()
	case "password_cracker", "password_sniffer", "ssh_exploit", "user_enum", "lan_sniffer", "rootkit", "exploit_kit", "advanced_exploit_kit", "sql_injector", "xss_exploit", "packet_capture", "packet_decoder":
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
		path += "\n"
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

	content, err := h.vfs.ReadFile(args[0])
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Ensure content ends with newline if not empty
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return &CommandResult{Output: content}
}

func (h *CommandHandler) handleCLEAR() *CommandResult {
	// ANSI escape sequence to clear screen
	return &CommandResult{Output: "\033[2J\033[H"}
}

func (h *CommandHandler) handleHELP() *CommandResult {
	// Read commands from filesystem
	binCommands, usrBinCommands := h.vfs.ListCommands()
	
	var output strings.Builder
	output.WriteString("Available Commands\n\n")
	
	// System Commands
	output.WriteString("System Commands:\n")
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
		output.WriteString(fmt.Sprintf("  %s - %s\n", cmdPadded, desc))
	}
	
	// User Tools (from /usr/bin)
	if len(usrBinCommands) > 0 {
		output.WriteString("\nUser Tools:\n")
		for _, cmd := range usrBinCommands {
			desc, err := h.vfs.GetCommandDescription(cmd)
			if err != nil {
				desc = "No description available"
			}
			// Pad command name to 20 chars for alignment
			cmdPadded := cmd
			if len(cmdPadded) < 20 {
				cmdPadded += strings.Repeat(" ", 20-len(cmdPadded))
			}
			output.WriteString(fmt.Sprintf("  %s - %s\n", cmdPadded, desc))
		}
	}
	
	// Tutorial command
	output.WriteString("\nLearning:\n")
	output.WriteString("  tutorial            - Show available tutorials\n")
	output.WriteString("  tutorial <id>       - Start a tutorial\n")
	
	// Shop commands
	output.WriteString("\nShopping:\n")
	output.WriteString("  shop                - List discovered shops\n")
	output.WriteString("  shop <shopID>       - Browse shop inventory\n")
	output.WriteString("  buy <shopID> <item> - Purchase item from shop\n")
	
	// Patch commands
	output.WriteString("\nTool Upgrades:\n")
	output.WriteString("  patches             - List available patches\n")
	output.WriteString("  patch <name> <tool> - Apply patch to tool\n")
	output.WriteString("  patch info <name>   - Show patch details\n")
	
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
	return &CommandResult{Output: "Please authenticate via SSH password authentication\n"}
}

func (h *CommandHandler) handleLOGOUT() *CommandResult {
	return &CommandResult{Output: "Logout successful. Goodbye!\n"}
}

func (h *CommandHandler) handleREGISTER(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: register <username> <password>")}
	}
	return &CommandResult{Output: "Please register via SSH password authentication (login with new credentials)\n"}
}

func (h *CommandHandler) handleUSERINFO() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	output := fmt.Sprintf("Username: %s\n", h.user.Username)
	output += fmt.Sprintf("IP: %s\n", h.user.IP)
	output += fmt.Sprintf("Local IP: %s\n", h.user.LocalIP)
	output += fmt.Sprintf("MAC: %s\n", h.user.MAC)
	output += fmt.Sprintf("Level: %d\n", h.user.Level)
	output += fmt.Sprintf("Experience: %d\n", h.user.Experience)
	output += fmt.Sprintf("Resources: CPU=%d, Bandwidth=%.1f, RAM=%d\n", 
		h.user.Resources.CPU, h.user.Resources.Bandwidth, h.user.Resources.RAM)
	output += fmt.Sprintf("Wallet: Crypto=%.2f, Data=%.2f\n", 
		h.user.Wallet.Crypto, h.user.Wallet.Data)
	
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleINFO() *CommandResult {
	// Display browser/client information (SSH session info)
	output := "Client Information:\n"
	output += "  Connection: SSH\n"
	output += "  Protocol: SSH2\n"
	if h.user != nil {
		output += fmt.Sprintf("  Username: %s\n", h.user.Username)
		output += fmt.Sprintf("  IP Address: %s\n", h.user.IP)
	}
	if h.sessionID != nil {
		output += fmt.Sprintf("  Session ID: %s\n", h.sessionID.String())
	}
	output += "  Terminal: ANSI compatible\n"
	
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleWHOAMI() *CommandResult {
	if h.user == nil {
		return &CommandResult{Output: "guest\n"}
	}
	username := h.user.Username
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
	return &CommandResult{Output: fmt.Sprintf("Username changed to %s\n", newUsername)}
}

func (h *CommandHandler) handleIFCONFIG() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	output := fmt.Sprintf("IP: %s\n", h.user.IP)
	output += fmt.Sprintf("Local IP: %s\n", h.user.LocalIP)
	output += fmt.Sprintf("MAC: %s\n", h.user.MAC)
	
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleSCAN(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// If no args, scan internet (top-level servers)
	if len(args) == 0 {
		servers, err := h.networkService.ScanInternet()
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		if len(servers) == 0 {
			return &CommandResult{Output: "No servers found\n"}
		}
		
		output := "Found servers:\n"
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = fmt.Sprintf(" [SHOP: %s]", shop.ShopType)
				}
			}
			output += fmt.Sprintf("  - %s (%s)%s\n", server.IP, server.LocalIP, shopIndicator)
		}
		return &CommandResult{Output: output}
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
		
		output := "Connected servers:\n"
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = fmt.Sprintf(" [SHOP: %s]", shop.ShopType)
				}
			}
			output += fmt.Sprintf("  - %s (%s)%s\n", server.IP, server.LocalIP, shopIndicator)
		}
		return &CommandResult{Output: output}
	}

	// Scan specific IP
	if len(args) == 1 {
		server, err := h.networkService.ScanIP(args[0])
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		output := h.networkService.FormatScanResult(server)
		// Ensure output ends with newline
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		return &CommandResult{Output: output}
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

	output := fmt.Sprintf("Server: %s\n", server.IP)
	output += fmt.Sprintf("Local IP: %s\n", server.LocalIP)
	output += fmt.Sprintf("Security Level: %d\n", server.SecurityLevel)
	output += fmt.Sprintf("Resources: CPU=%d, Bandwidth=%.1f, RAM=%d\n",
		server.Resources.CPU, server.Resources.Bandwidth, server.Resources.RAM)
	output += fmt.Sprintf("Wallet: Crypto=%.2f, Data=%.2f\n",
		server.Wallet.Crypto, server.Wallet.Data)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleCREATESERVER(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	// Generate random IPs
	ip := generateRandomIP()
	localIP := generateRandomLocalIP()

	server, err := h.serverService.CreateServer(ip, localIP)
	if err != nil {
		return &CommandResult{Error: err}
	}

	output := fmt.Sprintf("Server created successfully!\n")
	output += fmt.Sprintf("IP: %s\n", server.IP)
	output += fmt.Sprintf("Local IP: %s\n", server.LocalIP)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleCREATELOCALSERVER(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if h.currentServerPath == "" {
		return &CommandResult{Error: fmt.Errorf("must be connected to a server to create local server")}
	}

	// Get parent server
	parentServer, err := h.serverService.GetServerByPath(h.currentServerPath)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Generate random IPs
	ip := generateRandomIP()
	localIP := generateRandomLocalIP()

	server, err := h.serverService.CreateLocalServer(parentServer.IP, ip, localIP)
	if err != nil {
		return &CommandResult{Error: err}
	}

	output := fmt.Sprintf("Local server created successfully!\n")
	output += fmt.Sprintf("IP: %s\n", server.IP)
	output += fmt.Sprintf("Local IP: %s\n", server.LocalIP)

	return &CommandResult{Output: output}
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

func (h *CommandHandler) handleSSH(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: ssh <targetIP>")}
	}

	targetIP := args[0]

	// Check if server exists
	server, err := h.serverService.GetServerByIP(targetIP)
	if err != nil {
		// Try to get by path (for nested servers)
		server, err = h.serverService.GetServerByPath(targetIP)
		if err != nil {
			return &CommandResult{Error: fmt.Errorf("server not found: %s", targetIP)}
		}
	}

	// Check if server is exploited
	if !h.exploitationService.CanSSHToServer(h.user.ID, server.IP) {
		// Check by path if nested
		if h.currentServerPath != "" {
			fullPath := h.currentServerPath + ".localNetwork." + server.IP
			if !h.exploitationService.CanSSHToServer(h.user.ID, fullPath) {
				return &CommandResult{Error: fmt.Errorf("server %s must be exploited before connecting", targetIP)}
			}
		} else {
			return &CommandResult{Error: fmt.Errorf("server %s must be exploited before connecting", targetIP)}
		}
	}

	// Build new server path
	var newServerPath string
	if h.currentServerPath == "" {
		newServerPath = server.IP
	} else {
		newServerPath = h.currentServerPath + ".localNetwork." + server.IP
	}

	// Show progress bar for SSH connection
	if h.progressService != nil {
		duration := h.progressService.CalculateOperationTime(services.OperationSSH, h.user.Resources)
		durationSeconds := time.Duration(duration * float64(time.Second))
		
		showProgressBar(fmt.Sprintf("Connecting to %s...", targetIP), durationSeconds)
	}

	// Return special marker for shell to handle stack push
	// Shell will push current context, then update the path
	return &CommandResult{Output: fmt.Sprintf("__SSH_CONNECT__%s", newServerPath)}
}

func (h *CommandHandler) handleEXIT() *CommandResult {
	if h.currentServerPath == "" {
		// Not in SSH session - return special marker to quit
		return &CommandResult{Output: "__QUIT__"}
	}

	// Return special marker for shell to handle stack pop
	return &CommandResult{Output: "__EXIT_SSH__"}
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

	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: get <targetIP> <toolName>")}
	}

	targetIP := args[0]
	toolName := args[1]

	// Calculate download time based on user resources
	if h.progressService != nil {
		duration := h.progressService.CalculateOperationTime(services.OperationDownload, h.user.Resources)
		durationSeconds := time.Duration(duration * float64(time.Second))
		
		// Show progress bar
		showProgressBar(fmt.Sprintf("Downloading %s from %s...", toolName, targetIP), durationSeconds)
	}

	if err := h.toolService.DownloadTool(h.user.ID, targetIP, toolName); err != nil {
		return &CommandResult{Error: err}
	}

	// Sync tools to VFS so the new tool appears in help
	h.SyncUserToolsToVFS()

	output := fmt.Sprintf("Tool %s downloaded successfully from %s\n", toolName, targetIP)
	return &CommandResult{Output: output}
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
			return &CommandResult{Output: "No tools owned\n"}
		}

		output := "Owned tools:\n"
		for _, tool := range tools {
			output += fmt.Sprintf("  - %s: %s\n", tool.Name, tool.Function)
			if len(tool.Exploits) > 0 {
				output += "    Exploits: "
				for i, exploit := range tool.Exploits {
					if i > 0 {
						output += ", "
					}
					output += fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)
				}
				output += "\n"
			}
		}
		return &CommandResult{Output: output}
	}

	if len(toolStates) == 0 {
		return &CommandResult{Output: "No tools owned\n"}
	}

	output := "Owned tools:\n"
	for _, toolState := range toolStates {
		tool := toolState.Tool
		output += fmt.Sprintf("  - %s: %s\n", tool.Name, tool.Function)
		output += fmt.Sprintf("    Version: %d\n", toolState.Version)
		
		if len(toolState.AppliedPatches) > 0 {
			output += fmt.Sprintf("    Patches: %s\n", strings.Join(toolState.AppliedPatches, ", "))
		}
		
		if len(toolState.EffectiveExploits) > 0 {
			output += "    Effective Exploits: "
			for i, exploit := range toolState.EffectiveExploits {
				if i > 0 {
					output += ", "
				}
				output += fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)
			}
			output += "\n"
		}
	}

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleEXPLOITED() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	exploited, err := h.exploitationService.GetExploitedServers(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(exploited) == 0 {
		return &CommandResult{Output: "No exploited servers\n"}
	}

	output := "Exploited servers:\n"
	for _, exp := range exploited {
		output += fmt.Sprintf("  - %s (%s)\n", exp.ServerPath, exp.ServiceName)
		if len(exp.Exploits) > 0 {
			output += "    Exploits: "
			for i, exploit := range exp.Exploits {
				if i > 0 {
					output += ", "
				}
				output += fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)
			}
			output += "\n"
		}
	}

	return &CommandResult{Output: output}
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

	output := fmt.Sprintf("Mining stopped on server %s\n", serverIP)
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
		return &CommandResult{Output: "No active miners\n"}
	}

	output := "Active miners:\n"
	for _, miner := range miners {
		duration := time.Since(miner.StartTime)
		output += fmt.Sprintf("  - Server: %s (running for %s)\n", miner.ServerIP, duration.Round(time.Second))
		output += fmt.Sprintf("    Resources: CPU=%.1f, Bandwidth=%.1f, RAM=%d\n",
			miner.ResourceUsage.CPU, miner.ResourceUsage.Bandwidth, miner.ResourceUsage.RAM)
	}

	return &CommandResult{Output: output}
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

	output := fmt.Sprintf("Wallet Balance:\n")
	output += fmt.Sprintf("  Crypto: %.2f\n", user.Wallet.Crypto)
	output += fmt.Sprintf("  Data: %.2f\n", user.Wallet.Data)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleTOUCH(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: touch <filename>")}
	}
	
	if err := h.vfs.CreateFile(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ""}
}

func (h *CommandHandler) handleMKDIR(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: mkdir <dirname>")}
	}
	
	if err := h.vfs.CreateDirectory(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ""}
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
	
	return &CommandResult{Output: ""}
}

func (h *CommandHandler) handleCP(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: cp <src> <dest>")}
	}
	
	if err := h.vfs.CopyNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ""}
}

func (h *CommandHandler) handleMV(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: mv <src> <dest>")}
	}
	
	if err := h.vfs.MoveNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	return &CommandResult{Output: ""}
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

	// If no args, list all tutorials
	if len(args) == 0 {
		tutorials := h.tutorialService.GetAllTutorials()
		
		var output strings.Builder
		output.WriteString("Available Tutorials\n\n")
		
		if len(tutorials) == 0 {
			output.WriteString("No tutorials available.\n")
			output.WriteString("Edit tutorials.json to add tutorials.\n")
		} else {
			for _, tutorial := range tutorials {
				output.WriteString(fmt.Sprintf("  %s - %s\n", tutorial.ID, tutorial.Name))
				output.WriteString(fmt.Sprintf("    %s\n", tutorial.Description))
				if len(tutorial.Prerequisites) > 0 {
					output.WriteString(fmt.Sprintf("    Prerequisites: %s\n", strings.Join(tutorial.Prerequisites, ", ")))
				}
				output.WriteString(fmt.Sprintf("    Steps: %d\n\n", len(tutorial.Steps)))
			}
			output.WriteString("Usage: tutorial <tutorial_id>\n")
			output.WriteString("Example: tutorial getting_started\n")
		}
		
		return &CommandResult{Output: output.String()}
	}

	// Get specific tutorial
	tutorialID := args[0]
	tutorial, err := h.tutorialService.GetTutorialByID(tutorialID)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("tutorial not found: %s. Use 'tutorial' to list available tutorials", tutorialID)}
	}

	// Display tutorial
	var output strings.Builder
	output.WriteString(fmt.Sprintf("╔═══════════════════════════════════════╗\n"))
	output.WriteString(fmt.Sprintf("║   Tutorial: %s\n", tutorial.Name))
	output.WriteString(fmt.Sprintf("╚═══════════════════════════════════════╝\n\n"))
	output.WriteString(fmt.Sprintf("%s\n\n", tutorial.Description))
	
	if len(tutorial.Prerequisites) > 0 {
		output.WriteString("Prerequisites:\n")
		for _, prereq := range tutorial.Prerequisites {
			output.WriteString(fmt.Sprintf("  - %s\n", prereq))
		}
		output.WriteString("\n")
	}

	output.WriteString("Steps:\n\n")
	for i, step := range tutorial.Steps {
		output.WriteString(fmt.Sprintf("Step %d: %s\n", step.ID, step.Title))
		output.WriteString(fmt.Sprintf("%s\n", step.Description))
		
		if len(step.Commands) > 0 {
			output.WriteString("Example commands:\n")
			for _, cmd := range step.Commands {
				output.WriteString(fmt.Sprintf("  $ %s\n", cmd))
			}
		}
		
		if i < len(tutorial.Steps)-1 {
			output.WriteString("\n")
		}
	}
	
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("Tutorial file location: %s\n", h.tutorialService.GetTutorialPath()))
	output.WriteString("Edit this file to modify tutorials.\n")

	return &CommandResult{Output: output.String()}
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

