// Package cmd provides command handlers for terminal commands executed in the game shell.
package cmd

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"
	"terminal-sh/services"
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
		pathStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")) // Blue
		path = pathStyle.Render(path) + "\n"
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
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(titleStyle.Render("Available Commands") + "\n\n")
	
	// Helper function to format list items
	formatListItem := func(text, emoji string) string {
		prefix := "  "
		if emoji != "" {
			prefix = fmt.Sprintf("  %s ", emoji)
		}
		listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
		return listStyle.Render(prefix+text) + "\n"
	}
	
	// System Commands (Filesystem commands from VFS)
	if len(binCommands) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // Blue
			Bold(true)
		output.WriteString(sectionStyle.Render(emojiFolder + " Filesystem:") + "\n")
		
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
			cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green for filesystem commands
			output.WriteString("  " + cmdStyle.Render(cmdPadded) + " - " + desc + "\n")
		}
		output.WriteString("\n")
	}
	
	// Tool Commands (only show tools the user owns)
	if h.user != nil && h.toolService != nil {
		tools, err := h.toolService.GetUserTools(h.user.ID)
		if err == nil && len(tools) > 0 {
			sectionStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("207")). // Pink
				Bold(true)
			output.WriteString(sectionStyle.Render(emojiTools + " Tool Commands:") + "\n")
			for _, tool := range tools {
				// Pad command name to 20 chars for alignment
				cmdPadded := tool.Name
				if len(cmdPadded) < 20 {
					cmdPadded += strings.Repeat(" ", 20-len(cmdPadded))
				}
				// Use color for command name
				cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta for tool commands
				output.WriteString("  " + cmdStyle.Render(cmdPadded) + " - " + tool.Function + "\n")
			}
			output.WriteString("\n")
		}
	}
	
	// User commands
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")). // Green
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiUser + " User:") + "\n")
	output.WriteString(formatListItem("userinfo            - Show user information", ""))
	output.WriteString(formatListItem("whoami              - Display current username", ""))
	output.WriteString(formatListItem("name <newName>      - Change username", ""))
	output.WriteString("\n")
	
	// Network commands
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")). // Cyan
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiNetwork + " Network:") + "\n")
	output.WriteString(formatListItem("ifconfig            - Show network interfaces", ""))
	output.WriteString(formatListItem("scan [targetIP]     - Scan internet or IP", ""))
	output.WriteString(formatListItem("ssh <targetIP>      - Connect to a server", ""))
	output.WriteString(formatListItem("exit                - Disconnect from server", ""))
	output.WriteString(formatListItem("server              - Show current server info", ""))
	output.WriteString(formatListItem("createServer        - Create a new server", ""))
	output.WriteString(formatListItem("createLocalServer   - Create local server", ""))
	output.WriteString("\n")
	
	// Tools/Game commands
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Magenta
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiTool + " Tools:") + "\n")
	output.WriteString(formatListItem("get <targetIP> <tool> - Download tool from server", ""))
	output.WriteString(formatListItem("tools                - List owned tools", ""))
	output.WriteString(formatListItem("exploited            - List exploited servers", ""))
	output.WriteString(formatListItem("wallet               - Show wallet balance", ""))
	output.WriteString("\n")
	
	// Learning
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")). // Cyan
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiTutorial + " Learning:") + "\n")
	output.WriteString(formatListItem("tutorial             - Show available tutorials", ""))
	output.WriteString(formatListItem("tutorial <id>        - Start a tutorial", ""))
	output.WriteString("\n")
	
	// Shopping
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Magenta
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiShop + " Shopping:") + "\n")
	output.WriteString(formatListItem("shop                 - List discovered shops", ""))
	output.WriteString(formatListItem("shop <shopID>        - Browse shop inventory", ""))
	output.WriteString(formatListItem("buy <shopID> <item>  - Purchase item from shop", ""))
	output.WriteString("\n")
	
	// Tool Upgrades
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("207")). // Pink
		Bold(true)
	output.WriteString(sectionStyle.Render(emojiPatch + " Tool Upgrades:") + "\n")
	output.WriteString(formatListItem("patches              - List available patches", ""))
	output.WriteString(formatListItem("patch <name> <tool>  - Apply patch to tool", ""))
	output.WriteString(formatListItem("patch info <name>    - Show patch details", ""))
	output.WriteString("\n")
	
	// System
	sectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")). // Light gray
		Bold(true)
	output.WriteString(sectionStyle.Render("⚙️ System:") + "\n")
	output.WriteString(formatListItem("clear                - Clear the screen", ""))
	output.WriteString(formatListItem("help                 - Show this help message", ""))
	
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
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	return &CommandResult{Output: infoStyle.Render("🔐 Please authenticate via SSH password authentication") + "\n"}
}

func (h *CommandHandler) handleLOGOUT() *CommandResult {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	return &CommandResult{Output: successStyle.Render("✅ Logout successful. Goodbye!") + "\n"}
}

func (h *CommandHandler) handleREGISTER(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: register <username> <password>")}
	}
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	return &CommandResult{Output: infoStyle.Render("📝 Please register via SSH password authentication (login with new credentials)") + "\n"}
}

func (h *CommandHandler) handleUSERINFO() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	var output strings.Builder
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(headerStyle.Render("👤 User Information") + "\n\n")
	
	// Labels and values
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	
	output.WriteString(labelStyle.Render("Username:") + " " + valueStyle.Render(h.user.Username) + "\n")
	output.WriteString(labelStyle.Render("IP:") + " " + formatIP(h.user.IP) + "\n")
	output.WriteString(labelStyle.Render("Local IP:") + " " + formatIP(h.user.LocalIP) + "\n")
	output.WriteString(labelStyle.Render("MAC:") + " " + valueStyle.Render(h.user.MAC) + "\n")
	output.WriteString(labelStyle.Render("Level:") + " " + valueStyle.Render(fmt.Sprintf("%d", h.user.Level)) + "\n")
	output.WriteString(labelStyle.Render("Experience:") + " " + valueStyle.Render(fmt.Sprintf("%d", h.user.Experience)) + "\n")
	output.WriteString(labelStyle.Render("Resources:") + " " + valueStyle.Render(fmt.Sprintf("CPU=%d, Bandwidth=%.1f, RAM=%d", 
		h.user.Resources.CPU, h.user.Resources.Bandwidth, h.user.Resources.RAM)) + "\n")
	output.WriteString(labelStyle.Render("Wallet:") + " " + valueStyle.Render(fmt.Sprintf("Crypto=%.2f, Data=%.2f", 
		h.user.Wallet.Crypto, h.user.Wallet.Data)) + "\n")
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleINFO() *CommandResult {
	// Display browser/client information (SSH session info)
	var output strings.Builder
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(headerStyle.Render("ℹ️ Client Information") + "\n\n")
	
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	
	output.WriteString(labelStyle.Render("  Connection:") + " " + valueStyle.Render("SSH") + "\n")
	output.WriteString(labelStyle.Render("  Protocol:") + " " + valueStyle.Render("SSH2") + "\n")
	if h.user != nil {
		output.WriteString(labelStyle.Render("  Username:") + " " + valueStyle.Render(h.user.Username) + "\n")
		output.WriteString(labelStyle.Render("  IP Address:") + " " + formatIP(h.user.IP) + "\n")
	}
	if h.sessionID != nil {
		output.WriteString(labelStyle.Render("  Session ID:") + " " + valueStyle.Render(h.sessionID.String()) + "\n")
	}
	output.WriteString(labelStyle.Render("  Terminal:") + " " + valueStyle.Render("ANSI compatible") + "\n")
	
	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleWHOAMI() *CommandResult {
	if h.user == nil {
		guestStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gray
		return &CommandResult{Output: guestStyle.Render("guest") + "\n"}
	}
	usernameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")). // Green
		Bold(true)
	username := usernameStyle.Render(h.user.Username)
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
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Pink
		Bold(true)
	return &CommandResult{Output: successStyle.Render("✅ Username changed to ") + nameStyle.Render(newUsername) + "\n"}
}

func (h *CommandHandler) handleIFCONFIG() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	var output strings.Builder
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(headerStyle.Render("🌐 Network Configuration") + "\n\n")
	
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	
	output.WriteString(labelStyle.Render("IP:") + " " + formatIP(h.user.IP) + "\n")
	output.WriteString(labelStyle.Render("Local IP:") + " " + formatIP(h.user.LocalIP) + "\n")
	output.WriteString(labelStyle.Render("MAC:") + " " + valueStyle.Render(h.user.MAC) + "\n")
	
	return &CommandResult{Output: output.String()}
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
		
		var output strings.Builder
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
		output.WriteString(headerStyle.Render("🔍 Found servers:") + "\n")
		
		listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
		shopStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
		
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = shopStyle.Render(fmt.Sprintf(" [SHOP: %s]", shop.ShopType))
				}
			}
			output.WriteString(listStyle.Render("  - ") + formatIP(server.IP) + " (" + formatIP(server.LocalIP) + ")" + shopIndicator + "\n")
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
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
		output.WriteString(headerStyle.Render("🔍 Connected servers:") + "\n")
		
		listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
		shopStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
		
		for _, server := range servers {
			shopIndicator := ""
			if h.shopService != nil {
				if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
					shopIndicator = shopStyle.Render(fmt.Sprintf(" [SHOP: %s]", shop.ShopType))
				}
			}
			output.WriteString(listStyle.Render("  - ") + formatIP(server.IP) + " (" + formatIP(server.LocalIP) + ")" + shopIndicator + "\n")
		}
		return &CommandResult{Output: output.String()}
	}

	// Scan specific IP
	if len(args) == 1 {
		server, err := h.networkService.ScanIP(args[0])
		if err != nil {
			return &CommandResult{Error: err}
		}
		
		// Format with colors and emojis
		var output strings.Builder
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")). // Magenta
			Bold(true)
		output.WriteString(headerStyle.Render("🔍 Scan Results:") + "\n\n")
		
		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")). // Cyan
			Bold(true)
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Light gray
		
		// IP addresses
		output.WriteString(labelStyle.Render("📍 IP:") + " " + formatIP(server.IP) + "\n")
		output.WriteString(labelStyle.Render("📍 Local IP:") + " " + formatIP(server.LocalIP) + "\n")
		
		// Security level with color
		secLevelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red for high security
		if server.SecurityLevel <= 3 {
			secLevelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green for low security
		} else if server.SecurityLevel <= 6 {
			secLevelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Yellow for medium security
		}
		output.WriteString(labelStyle.Render("🔒 Security Level:") + " " + secLevelStyle.Render(fmt.Sprintf("%d", server.SecurityLevel)) + "\n")
		
		// Resources
		resourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Pink
		output.WriteString(labelStyle.Render("💻 Resources:") + " ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("CPU=%d", server.Resources.CPU)) + ", ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("Bandwidth=%.1f", server.Resources.Bandwidth)) + ", ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("RAM=%d", server.Resources.RAM)) + "\n")
		
		// Wallet
		walletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green
		output.WriteString(labelStyle.Render("💰 Wallet:") + " ")
		output.WriteString(walletStyle.Render(fmt.Sprintf("Crypto=%.2f", server.Wallet.Crypto)) + ", ")
		output.WriteString(walletStyle.Render(fmt.Sprintf("Data=%.2f", server.Wallet.Data)) + "\n")
		
		// Tools
		if len(server.Tools) > 0 {
			toolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
			output.WriteString("\n" + labelStyle.Render("🛠️ Available Tools:") + "\n")
			output.WriteString(valueStyle.Render(fmt.Sprintf("  Use 'get %s <toolName>' to download\n", server.IP)))
			for _, tool := range server.Tools {
				output.WriteString("  " + toolStyle.Render("• " + tool) + "\n")
			}
		}
		
		// Services
		if len(server.Services) > 0 {
			serviceHeaderStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")). // Blue
				Bold(true)
			output.WriteString("\n" + serviceHeaderStyle.Render("🌐 Services:") + "\n")
			for _, service := range server.Services {
				serviceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
				vulnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
				output.WriteString("  " + serviceStyle.Render(fmt.Sprintf("• %s (port %d): %s\n", service.Name, service.Port, service.Description)))
				if service.Vulnerable && len(service.Vulnerabilities) > 0 {
					output.WriteString("    " + vulnStyle.Render("⚠️ Vulnerabilities:") + "\n")
					for _, vuln := range service.Vulnerabilities {
						output.WriteString("      " + vulnStyle.Render(fmt.Sprintf("- %s (level %d)\n", vuln.Type, vuln.Level)))
					}
				}
			}
		}
		
		// Connected IPs
		if len(server.ConnectedIPs) > 0 {
			output.WriteString("\n" + labelStyle.Render("🔗 Connected IPs:") + "\n")
			ipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
			for _, ip := range server.ConnectedIPs {
				output.WriteString("  " + ipStyle.Render("• " + formatIP(ip)) + "\n")
			}
		}
		
		// Shop
		if h.shopService != nil {
			if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
				shopStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // Magenta
				output.WriteString("\n" + shopStyle.Render("🛒 Shop:") + " ")
				output.WriteString(shopStyle.Render(fmt.Sprintf("[%s] %s", shop.ShopType, shop.Name)) + "\n")
				output.WriteString(valueStyle.Render(fmt.Sprintf("  %s\n", shop.Description)))
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
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	output.WriteString(headerStyle.Render("🖥️ Server Information") + "\n\n")
	
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // Light gray
	
	output.WriteString(labelStyle.Render("Server:") + " " + formatIP(server.IP) + "\n")
	output.WriteString(labelStyle.Render("Local IP:") + " " + formatIP(server.LocalIP) + "\n")
	output.WriteString(labelStyle.Render("Security Level:") + " " + valueStyle.Render(fmt.Sprintf("%d", server.SecurityLevel)) + "\n")
	output.WriteString(labelStyle.Render("Resources:") + " " + valueStyle.Render(fmt.Sprintf("CPU=%d, Bandwidth=%.1f, RAM=%d",
		server.Resources.CPU, server.Resources.Bandwidth, server.Resources.RAM)) + "\n")
	output.WriteString(labelStyle.Render("Wallet:") + " " + valueStyle.Render(fmt.Sprintf("Crypto=%.2f, Data=%.2f",
		server.Wallet.Crypto, server.Wallet.Data)) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleCREATESERVER(_ []string) *CommandResult {
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

	var output strings.Builder
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true)
	output.WriteString(successStyle.Render("✅ Server created successfully!") + "\n\n")
	
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	output.WriteString(labelStyle.Render("IP:") + " " + formatIP(server.IP) + "\n")
	output.WriteString(labelStyle.Render("Local IP:") + " " + formatIP(server.LocalIP) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleCREATELOCALSERVER(_ []string) *CommandResult {
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

	var output strings.Builder
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true)
	output.WriteString(successStyle.Render("✅ Local server created successfully!") + "\n\n")
	
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")) // Cyan
	output.WriteString(labelStyle.Render("IP:") + " " + formatIP(server.IP) + "\n")
	output.WriteString(labelStyle.Render("Local IP:") + " " + formatIP(server.LocalIP) + "\n")

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

// formatIP formats an IP address for display with color
// Local IPs (private network) are styled in yellow/orange, internet IPs in cyan
// Note: Can't import terminal package due to import cycle, so this duplicates FormatIP logic
func formatIP(ip string) string {
	var color string
	if isPrivateIP(ip) {
		color = "220" // Yellow/Orange for local IPs
	} else {
		color = "51" // Cyan for internet IPs
	}
	
	ipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return ipStyle.Render(ip)
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

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Pink
		Bold(true)
	output := successStyle.Render("✅ Tool ") + toolStyle.Render(toolName) + successStyle.Render(" downloaded successfully from ") + formatIP(targetIP) + "\n"
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
			infoStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")) // Yellow
			return &CommandResult{Output: infoStyle.Render("ℹ️  No tools owned") + "\n"}
		}

		var output strings.Builder
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")). // Pink
			Bold(true)
		output.WriteString(headerStyle.Render("🛠️  Owned tools:") + "\n")
		toolStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")) // Pink
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Light gray
		exploitStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // Red
		for _, tool := range tools {
			output.WriteString("  " + toolStyle.Render("• "+tool.Name) + ": " + descStyle.Render(tool.Function) + "\n")
			if len(tool.Exploits) > 0 {
				output.WriteString("    " + exploitStyle.Render("⚡ Exploits: "))
				for i, exploit := range tool.Exploits {
					if i > 0 {
						output.WriteString(", ")
					}
					output.WriteString(exploitStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
				}
				output.WriteString("\n")
			}
		}
		return &CommandResult{Output: output.String()}
	}

	if len(toolStates) == 0 {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")) // Yellow
		return &CommandResult{Output: infoStyle.Render("ℹ️  No tools owned") + "\n"}
	}

	var output strings.Builder
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Pink
		Bold(true)
	output.WriteString(headerStyle.Render("🛠️  Owned tools:") + "\n")
	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")) // Pink
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	patchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("207")) // Light pink
	exploitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")) // Red
	for _, toolState := range toolStates {
		tool := toolState.Tool
		output.WriteString("  " + toolStyle.Render("• "+tool.Name) + ": " + descStyle.Render(tool.Function) + "\n")
		output.WriteString("    " + versionStyle.Render("📦 Version:") + " " + versionStyle.Render(fmt.Sprintf("%d", toolState.Version)) + "\n")
		
		if len(toolState.AppliedPatches) > 0 {
			output.WriteString("    " + patchStyle.Render("🔧 Patches:") + " " + patchStyle.Render(strings.Join(toolState.AppliedPatches, ", ")) + "\n")
		}
		
		if len(toolState.EffectiveExploits) > 0 {
			output.WriteString("    " + exploitStyle.Render("⚡ Effective Exploits: "))
			for i, exploit := range toolState.EffectiveExploits {
				if i > 0 {
					output.WriteString(", ")
				}
				output.WriteString(exploitStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
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

	exploited, err := h.exploitationService.GetExploitedServers(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(exploited) == 0 {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")) // Yellow
		return &CommandResult{Output: infoStyle.Render("ℹ️  No exploited servers") + "\n"}
	}

	var output strings.Builder
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")). // Red
		Bold(true)
	output.WriteString(headerStyle.Render("⚡ Exploited servers:") + "\n")
	serverStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")) // Pink
	serviceStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	exploitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")) // Red
	for _, exp := range exploited {
		output.WriteString("  " + serverStyle.Render("• "+exp.ServerPath) + " (" + serviceStyle.Render(exp.ServiceName) + ")\n")
		if len(exp.Exploits) > 0 {
			output.WriteString("    " + exploitStyle.Render("⚡ Exploits: "))
			for i, exploit := range exp.Exploits {
				if i > 0 {
					output.WriteString(", ")
				}
				output.WriteString(exploitStyle.Render(fmt.Sprintf("%s (level %d)", exploit.Type, exploit.Level)))
			}
			output.WriteString("\n")
		}
	}

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

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	output := successStyle.Render("✅ Mining stopped on server ") + formatIP(serverIP) + "\n"
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
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")) // Yellow
		return &CommandResult{Output: infoStyle.Render("ℹ️  No active miners") + "\n"}
	}

	var output strings.Builder
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")). // Yellow
		Bold(true)
	output.WriteString(headerStyle.Render("⛏️  Active miners:") + "\n")
	serverStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")) // Pink
	resourceStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	for _, miner := range miners {
		duration := time.Since(miner.StartTime)
		output.WriteString("  " + serverStyle.Render("• Server:") + " " + formatIP(miner.ServerIP) + " " + timeStyle.Render("(running for "+duration.Round(time.Second).String()+")") + "\n")
		output.WriteString("    " + resourceStyle.Render("💻 Resources:") + " ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("CPU=%.1f", miner.ResourceUsage.CPU)) + ", ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("Bandwidth=%.1f", miner.ResourceUsage.Bandwidth)) + ", ")
		output.WriteString(resourceStyle.Render(fmt.Sprintf("RAM=%d", miner.ResourceUsage.RAM)) + "\n")
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
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")). // Green
		Bold(true)
	output.WriteString(headerStyle.Render("💰 Wallet Balance:") + "\n")
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	cryptoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")) // Yellow
	dataStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")) // Cyan
	output.WriteString("  " + labelStyle.Render("₿ Crypto:") + " " + cryptoStyle.Render(fmt.Sprintf("%.2f", user.Wallet.Crypto)) + "\n")
	output.WriteString("  " + labelStyle.Render("💾 Data:") + " " + dataStyle.Render(fmt.Sprintf("%.2f", user.Wallet.Data)) + "\n")

	return &CommandResult{Output: output.String()}
}

func (h *CommandHandler) handleTOUCH(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: touch <filename>")}
	}
	
	if err := h.vfs.CreateFile(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	return &CommandResult{Output: successStyle.Render("📄 Created file: ") + fileStyle.Render(args[0]) + "\n"}
}

func (h *CommandHandler) handleMKDIR(args []string) *CommandResult {
	if len(args) != 1 {
		return &CommandResult{Error: fmt.Errorf("usage: mkdir <dirname>")}
	}
	
	if err := h.vfs.CreateDirectory(args[0]); err != nil {
		return &CommandResult{Error: err}
	}
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")) // Blue
	return &CommandResult{Output: successStyle.Render("📁 Created directory: ") + dirStyle.Render(args[0]) + "\n"}
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
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	return &CommandResult{Output: successStyle.Render("🗑️  Deleted: ") + fileStyle.Render(filename) + "\n"}
}

func (h *CommandHandler) handleCP(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: cp <src> <dest>")}
	}
	
	if err := h.vfs.CopyNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	return &CommandResult{Output: successStyle.Render("📋 Copied: ") + fileStyle.Render(args[0]) + " → " + fileStyle.Render(args[1]) + "\n"}
}

func (h *CommandHandler) handleMV(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: mv <src> <dest>")}
	}
	
	if err := h.vfs.MoveNode(args[0], args[1]); err != nil {
		return &CommandResult{Error: err}
	}
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Green
	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray
	return &CommandResult{Output: successStyle.Render("✂️  Moved: ") + fileStyle.Render(args[0]) + " → " + fileStyle.Render(args[1]) + "\n"}
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
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")). // Magenta
			Bold(true)
		output.WriteString(titleStyle.Render(emojiSparkles + " " + emojiTutorial + " Available Tutorials " + emojiTutorial + " " + emojiSparkles) + "\n\n")
		
		if len(tutorials) == 0 {
			infoStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")) // Yellow
			output.WriteString(infoStyle.Render(emojiLightbulb + " No tutorials available yet.\n"))
			output.WriteString(infoStyle.Render("Edit tutorials.json to add tutorials.\n"))
		} else {
			for i, tutorial := range tutorials {
				// Tutorial ID with emoji and color
				idStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("213")). // Pink
					Bold(true)
				nameStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("51")). // Cyan
					Bold(true)
				
				output.WriteString(fmt.Sprintf("  %s %s %s\n", emojiStar, idStyle.Render(tutorial.ID), nameStyle.Render("- "+tutorial.Name)))
				
				// Description with emoji
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("252")) // Light gray
				output.WriteString(fmt.Sprintf("    %s %s\n", emojiBook, descStyle.Render(tutorial.Description)))
				
				if len(tutorial.Prerequisites) > 0 {
					prereqStyle := lipgloss.NewStyle().
						Foreground(lipgloss.Color("220")) // Yellow
					prereqText := strings.Join(tutorial.Prerequisites, ", ")
					output.WriteString(fmt.Sprintf("    %s %s\n", emojiTarget, prereqStyle.Render("Prerequisites: "+prereqText)))
				}
				
				stepsStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("46")) // Green
				output.WriteString(fmt.Sprintf("    %s %s\n", emojiSteps, stepsStyle.Render(fmt.Sprintf("Steps: %d", len(tutorial.Steps)))))
				
				if i < len(tutorials)-1 {
					output.WriteString("\n")
				}
			}
			
			output.WriteString("\n")
			usageStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")). // Blue
				Bold(true)
			exampleStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("207")) // Light pink
			output.WriteString(usageStyle.Render(emojiArrow + " Usage: ") + "tutorial <tutorial_id>\n")
			output.WriteString(exampleStyle.Render(emojiRocket + " Example: ") + "tutorial getting_started\n")
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
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")). // Magenta
		Bold(true)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Width(50)
	
	titleContent := titleStyle.Render(emojiGraduation + " Tutorial: " + tutorial.Name + " " + emojiGraduation)
	boxedTitle := boxStyle.Render(titleContent)
	output.WriteString(boxedTitle + "\n\n")
	
	// Description with emoji
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("51")). // Cyan
		Bold(true)
	output.WriteString(descStyle.Render(emojiBook + " " + tutorial.Description) + "\n\n")
	
	// Prerequisites section
	if len(tutorial.Prerequisites) > 0 {
		prereqHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")). // Yellow
			Bold(true)
		output.WriteString(prereqHeaderStyle.Render(emojiTarget + " Prerequisites:") + "\n")
		prereqItemStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Light gray
		for _, prereq := range tutorial.Prerequisites {
			output.WriteString(fmt.Sprintf("  %s %s\n", emojiCheck, prereqItemStyle.Render(prereq)))
		}
		output.WriteString("\n")
	}

	// Steps section
	stepsHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")). // Green
		Bold(true)
	output.WriteString(stepsHeaderStyle.Render(emojiSparkles + " Steps:") + "\n\n")
	
	for i, step := range tutorial.Steps {
		stepNumStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")). // Pink
			Bold(true)
		stepTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")). // Cyan
			Bold(true)
		stepDescStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Light gray
		
		output.WriteString(stepNumStyle.Render(fmt.Sprintf("%s Step %d:", emojiSteps, step.ID)))
		output.WriteString(" " + stepTitleStyle.Render(step.Title) + "\n")
		output.WriteString(stepDescStyle.Render(step.Description) + "\n")
		
		if len(step.Commands) > 0 {
			cmdHeaderStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")). // Blue
				Bold(true)
			output.WriteString(cmdHeaderStyle.Render("  " + emojiCode + " Example commands:") + "\n")
			cmdStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")) // Green
			for _, cmd := range step.Commands {
				output.WriteString(fmt.Sprintf("    $ %s\n", cmdStyle.Render(cmd)))
			}
		}
		
		if i < len(tutorial.Steps)-1 {
			output.WriteString("\n")
		}
	}
	
	output.WriteString("\n")
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // Gray
	output.WriteString(infoStyle.Render(emojiLightbulb + " Tutorial file location: " + h.tutorialService.GetTutorialPath() + "\n"))
	output.WriteString(infoStyle.Render("Edit this file to modify tutorials.\n"))

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

