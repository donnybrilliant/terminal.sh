// Package cmd provides command handlers for terminal commands executed in the game shell.
package cmd

import (
	"fmt"
	"math/rand"
	"net"
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
	// ASCII animation trigger (for ascii -a command)
	StartASCIIAnimation *ASCIIAnimationRequest
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
	patchService   *services.PatchService
	progressService *services.ProgressService
	chatService    *services.ChatService
	missionService *services.MissionService
	achievementService *services.AchievementService
	rewardService  *services.RewardService
	missionGenerator *services.MissionGenerator
	serverGenerator *services.ServerGenerator
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
	achievementService, _ := services.NewAchievementService(db, "") // Initialize achievement service
	rewardService := services.NewRewardService(db, userService, toolService, patchService, achievementService)
	missionService, _ := services.NewMissionService(db, "", rewardService) // Initialize mission service
	
	// Initialize mission generator
	missionGenerator := services.NewMissionGenerator(db, missionService, serverService, toolService, userService)
	missionService.SetMissionGenerator(missionGenerator)
	
	// Initialize server generator
	serverGenerator := services.NewServerGenerator(db, serverService)
	networkService.SetServerGenerator(serverGenerator)
	
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
		missionService: missionService,
		achievementService: achievementService,
		rewardService: rewardService,
		missionGenerator: missionGenerator,
		serverGenerator: serverGenerator,
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
	output.WriteString(formatListItem("ssh <targetIP>      - Connect to a server", ""))
	output.WriteString(formatListItem("exit                - Disconnect from server", ""))
	output.WriteString(formatListItem("server              - Show current server info", ""))
	output.WriteString("\n")
	
	// Tools/Game commands
	output.WriteString(ui.AccentBoldStyle.Render(emojiTool + " Tools:") + "\n")
	output.WriteString(formatListItem("get <targetIP> <tool> - Download tool from server", ""))
	output.WriteString(formatListItem("tools                - List owned tools", ""))
	output.WriteString(formatListItem("exploited            - List exploited servers", ""))
	output.WriteString(formatListItem("wallet               - Show wallet balance", ""))
	output.WriteString("\n")
	
	// Learning
	output.WriteString(ui.InfoStyle.Render(emojiTutorial + " Learning:") + "\n")
	output.WriteString(formatListItem("tutorial             - Show available tutorials", ""))
	output.WriteString(formatListItem("tutorial <id>        - Start a tutorial", ""))
	output.WriteString("\n")
	
	// Story Missions
	output.WriteString(ui.AccentBoldStyle.Render("🎯 Story Missions:") + "\n")
	output.WriteString(formatListItem("mission              - List available missions", ""))
	output.WriteString(formatListItem("mission <id>         - View mission details", ""))
	output.WriteString(formatListItem("mission start <id>   - Start a mission", ""))
	output.WriteString(formatListItem("mission complete <id> - Complete a mission", ""))
	output.WriteString(formatListItem("mission status       - View your mission progress", ""))
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
	output.WriteString("\n")
	
	// System
	output.WriteString(ui.ValueStyle.Render("⚙️ System:") + "\n")
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
	output.WriteString(ui.FormatListBullet("-s, --size <percent> - Size as percentage of viewport: 10-100 (default: 50)"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Size:", ""))
	output.WriteString(ui.FormatListBullet("Size is a percentage (10-100) of the viewport width/height"))
	output.WriteString(ui.FormatListBullet("Larger percentages = bigger ASCII characters (more blocks per character)"))
	output.WriteString(ui.FormatListBullet("Minimum size is enforced to ensure text is readable"))
	output.WriteString(ui.FormatListBullet("Examples: 10 (small), 30 (medium-small), 50 (medium), 70 (large), 100 (extra large)"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Color Palettes:", ""))
	output.WriteString(ui.FormatListBullet("white, orange, green, purple, blue, red, cyan, yellow, pink/magenta"))
	output.WriteString("\n")
	output.WriteString(ui.FormatSectionHeader("Examples:", ""))
	output.WriteString(ui.FormatListBullet("ascii HELLO - Convert \"HELLO\" to ASCII art"))
	output.WriteString(ui.FormatListBullet("ascii WELCOME -a - Create animated welcome with \"WELCOME\" centered"))
	output.WriteString(ui.FormatListBullet("ascii TEST -c green - Use green color palette"))
	output.WriteString(ui.FormatListBullet("ascii \"HELLO WORLD\" -s 80 - Large size (80% of viewport)"))
	output.WriteString(ui.FormatListBullet("ascii TERMINAL -a -c purple -s 60 - Animated with purple palette, 60% size"))
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
		servers, err := h.networkService.ScanInternet()
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
		
		// Tools
		if len(server.Tools) > 0 {
			output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🛠️ Available Tools:") + "\n")
			output.WriteString(ui.ValueStyle.Render(fmt.Sprintf("  Use 'get %s <toolName>' to download\n", server.IP)))
			for _, tool := range server.Tools {
				output.WriteString(ui.FormatListBulletWithStyle("• "+tool, ui.AccentStyle))
			}
		}
		
		// Services
		if len(server.Services) > 0 {
			output.WriteString("\n" + ui.SectionStyle.Render("🌐 Services:") + "\n")
			for _, service := range server.Services {
				output.WriteString("  " + ui.InfoStyle.Render(fmt.Sprintf("• %s (port %d): %s\n", service.Name, service.Port, service.Description)))
				if service.Vulnerable && len(service.Vulnerabilities) > 0 {
					output.WriteString("    " + ui.ErrorStyle.Render("⚠️ Vulnerabilities:") + "\n")
					for _, vuln := range service.Vulnerabilities {
						output.WriteString("      " + ui.ErrorStyle.Render(fmt.Sprintf("- %s (level %d)\n", vuln.Type, vuln.Level)))
					}
				}
			}
		}
		
		// Connected IPs
		if len(server.ConnectedIPs) > 0 {
			output.WriteString("\n" + ui.LabelStyle.Bold(true).Render("🔗 Connected IPs:") + "\n")
			for _, ip := range server.ConnectedIPs {
				output.WriteString(ui.FormatListBulletWithStyle("• "+formatIP(ip), ui.ListStyle))
			}
		}
		
		// Shop
		if h.shopService != nil {
			if shop, err := h.shopService.GetShopByServerIP(server.IP); err == nil {
				output.WriteString("\n" + ui.AccentStyle.Render("🛒 Shop:") + " ")
				output.WriteString(ui.AccentStyle.Render(fmt.Sprintf("[%s] %s", shop.ShopType, shop.Name)) + "\n")
				output.WriteString(ui.ValueStyle.Render(fmt.Sprintf("  %s\n", shop.Description)))
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

	output := ui.SuccessStyle.Render("✅ Tool ") + ui.AccentBoldStyle.Render(toolName) + ui.SuccessStyle.Render(" downloaded successfully from ") + formatIP(targetIP) + "\n"
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
		
		if len(toolState.AppliedPatches) > 0 {
			output.WriteString("    " + ui.AccentBoldStyle.Render("🔧 Patches:") + " " + ui.AccentBoldStyle.Render(strings.Join(toolState.AppliedPatches, ", ")) + "\n")
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

	exploited, err := h.exploitationService.GetExploitedServers(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(exploited) == 0 {
		return &CommandResult{Output: ui.WarningStyle.Render("ℹ️  No exploited servers") + "\n"}
	}

	var output strings.Builder
	output.WriteString(ui.ErrorStyle.Render("⚡ Exploited servers:") + "\n")
	for _, exp := range exploited {
		output.WriteString(ui.FormatListBullet(ui.AccentStyle.Render("• "+exp.ServerPath) + " (" + ui.InfoStyle.Render(exp.ServiceName) + ")"))
		if len(exp.Exploits) > 0 {
			output.WriteString("    " + ui.ErrorStyle.Render("⚡ Exploits: "))
			for i, exploit := range exp.Exploits {
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
			output.WriteString(ui.WarningStyle.Render(emojiLightbulb + " No tutorials available yet.\n"))
			output.WriteString(ui.WarningStyle.Render("Edit tutorials.json to add tutorials.\n"))
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
	output.WriteString(ui.GrayStyle.Render(emojiLightbulb + " Tutorial file location: " + h.tutorialService.GetTutorialPath() + "\n"))
	output.WriteString(ui.GrayStyle.Render("Edit this file to modify tutorials.\n"))

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

