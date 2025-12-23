package cmd

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"ssh4xx-go/filesystem"
	"ssh4xx-go/models"
	"ssh4xx-go/services"

	"github.com/google/uuid"
)

type CommandResult struct {
	Output string
	Error  error
	Nodes  []*filesystem.Node // For ls command
}

type CommandHandler struct {
	vfs            *filesystem.VFS
	user           *models.User
	userService    *services.UserService
	serverService  *services.ServerService
	networkService *services.NetworkService
	sessionService *services.SessionService
	toolService    *services.ToolService
	exploitationService *services.ExploitationService
	miningService  *services.MiningService
	currentServerPath string // Current server path if in SSH mode
	sessionID       *uuid.UUID // Current session ID
	onSSHConnect    func(serverPath string) error // Callback for SSH connection
	onSSHDisconnect func() error // Callback for SSH disconnection
}

func NewCommandHandler(vfs *filesystem.VFS, user *models.User, userService *services.UserService) *CommandHandler {
	serverService := services.NewServerService()
	networkService := services.NewNetworkService(serverService)
	sessionService := services.NewSessionService(serverService)
	toolService := services.NewToolService(serverService)
	exploitationService := services.NewExploitationService(toolService, serverService)
	miningService := services.NewMiningService(toolService, serverService)
	return &CommandHandler{
		vfs:            vfs,
		user:           user,
		userService:    userService,
		serverService:  serverService,
		networkService: networkService,
		sessionService: sessionService,
		toolService:   toolService,
		exploitationService: exploitationService,
		miningService: miningService,
	}
}

// SetSessionID sets the current session ID
func (h *CommandHandler) SetSessionID(sessionID uuid.UUID) {
	h.sessionID = &sessionID
}

// SetSSHCallbacks sets callbacks for SSH connect/disconnect
func (h *CommandHandler) SetSSHCallbacks(onConnect func(serverPath string) error, onDisconnect func() error) {
	h.onSSHConnect = onConnect
	h.onSSHDisconnect = onDisconnect
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
		return h.handleLS()
	case "cd":
		return h.handleCD(args)
	case "cat":
		return h.handleCAT(args)
	case "clear":
		return h.handleCLEAR()
	case "help":
		return h.handleHELP()
	case "login":
		return h.handleLOGIN(args)
	case "logout":
		return h.handleLOGOUT()
	case "register":
		return h.handleREGISTER(args)
	case "userinfo":
		return h.handleUSERINFO()
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
	case "crypto_miner":
		return h.handleCRYPTOMINER(args)
	case "stop_mining":
		return h.handleSTOPMINING(args)
	case "miners":
		return h.handleMINERS()
	case "wallet":
		return h.handleWALLET()
	case "password_cracker", "ssh_exploit", "user_enum", "lan_sniffer", "rootkit", "exploit_kit":
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
	return &CommandResult{Output: h.vfs.GetCurrentPath()}
}

func (h *CommandHandler) handleLS() *CommandResult {
	nodes := h.vfs.ListDir()
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

	return &CommandResult{Output: content}
}

func (h *CommandHandler) handleCLEAR() *CommandResult {
	// ANSI escape sequence to clear screen
	return &CommandResult{Output: "\033[2J\033[H"}
}

func (h *CommandHandler) handleHELP() *CommandResult {
	// Return a special marker that the terminal can use to render animated help
	return &CommandResult{Output: "__ANIMATED_HELP__"}
}

func (h *CommandHandler) handleLOGIN(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: login <username> <password>")}
	}
	return &CommandResult{Output: "Please authenticate via SSH password authentication"}
}

func (h *CommandHandler) handleLOGOUT() *CommandResult {
	return &CommandResult{Output: "Logout successful. Goodbye!"}
}

func (h *CommandHandler) handleREGISTER(args []string) *CommandResult {
	if len(args) != 2 {
		return &CommandResult{Error: fmt.Errorf("usage: register <username> <password>")}
	}
	return &CommandResult{Output: "Please register via SSH password authentication (login with new credentials)"}
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
	output += fmt.Sprintf("Wallet: Crypto=%.2f, Data=%.2f", 
		h.user.Wallet.Crypto, h.user.Wallet.Data)
	
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleWHOAMI() *CommandResult {
	if h.user == nil {
		return &CommandResult{Output: "guest"}
	}
	return &CommandResult{Output: h.user.Username}
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
	return &CommandResult{Output: fmt.Sprintf("Username changed to %s", newUsername)}
}

func (h *CommandHandler) handleIFCONFIG() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}
	
	output := fmt.Sprintf("IP: %s\n", h.user.IP)
	output += fmt.Sprintf("Local IP: %s\n", h.user.LocalIP)
	output += fmt.Sprintf("MAC: %s", h.user.MAC)
	
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
			return &CommandResult{Output: "No servers found"}
		}
		
		output := "Found servers:\n"
		for _, server := range servers {
			output += fmt.Sprintf("  - %s (%s)\n", server.IP, server.LocalIP)
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
			return &CommandResult{Output: "No connected servers found"}
		}
		
		output := "Connected servers:\n"
		for _, server := range servers {
			output += fmt.Sprintf("  - %s (%s)\n", server.IP, server.LocalIP)
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
	output += fmt.Sprintf("Wallet: Crypto=%.2f, Data=%.2f",
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
	output += fmt.Sprintf("Local IP: %s", server.LocalIP)

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
	output += fmt.Sprintf("Local IP: %s", server.LocalIP)

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

	// Update current server path
	h.currentServerPath = newServerPath

	// Call SSH connect callback if set
	if h.onSSHConnect != nil {
		if err := h.onSSHConnect(newServerPath); err != nil {
			return &CommandResult{Error: err}
		}
	}

	output := fmt.Sprintf("Connected to %s\n", server.IP)
	output += fmt.Sprintf("Server path: %s", newServerPath)

	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleEXIT() *CommandResult {
	if h.currentServerPath == "" {
		return &CommandResult{Output: "Not in SSH session"}
	}

	// Parse server path to get parent
	parts := parseServerPathParts(h.currentServerPath)
	
	// Remove "localNetwork" and the last IP
	// Path format: "1.1.1.1.localNetwork.10.0.0.5"
	// We want to go back to "1.1.1.1"
	if len(parts) >= 3 && parts[len(parts)-2] == "localNetwork" {
		// Remove last two parts (localNetwork and IP)
		h.currentServerPath = strings.Join(parts[:len(parts)-2], ".")
	} else if len(parts) == 1 {
		// Top-level server, disconnect completely
		h.currentServerPath = ""
	} else {
		// Fallback: just remove last part
		h.currentServerPath = strings.Join(parts[:len(parts)-1], ".")
	}

	// Call SSH disconnect callback if set
	if h.onSSHDisconnect != nil {
		if err := h.onSSHDisconnect(); err != nil {
			return &CommandResult{Error: err}
		}
	}

	return &CommandResult{Output: "Disconnected"}
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

	if err := h.toolService.DownloadTool(h.user.ID, targetIP, toolName); err != nil {
		return &CommandResult{Error: err}
	}

	output := fmt.Sprintf("Tool %s downloaded successfully from %s", toolName, targetIP)
	return &CommandResult{Output: output}
}

func (h *CommandHandler) handleTOOLS() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	tools, err := h.toolService.GetUserTools(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(tools) == 0 {
		return &CommandResult{Output: "No tools owned"}
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

func (h *CommandHandler) handleEXPLOITED() *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	exploited, err := h.exploitationService.GetExploitedServers(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(exploited) == 0 {
		return &CommandResult{Output: "No exploited servers"}
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

	output := fmt.Sprintf("Mining started on server %s", serverIP)
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

	output := fmt.Sprintf("Mining stopped on server %s", serverIP)
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
		return &CommandResult{Output: "No active miners"}
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
	output += fmt.Sprintf("  Data: %.2f", user.Wallet.Data)

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
	
	// For now, return a message - full edit mode would require more complex state management
	return &CommandResult{Output: fmt.Sprintf("Edit mode for %s. Use ':save' to save, ':exit' to exit.\n(Note: Full edit mode not yet implemented)", args[0])}
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

