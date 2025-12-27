package terminal

import (
	"fmt"
	"strings"
	"terminal-sh/cmd"
	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"
	"terminal-sh/services"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// ShellModel handles the interactive shell after login
type ShellModel struct {
	userService *services.UserService
	user        *models.User
	vfs         *filesystem.VFS
	handler     *cmd.CommandHandler
	chatService *services.ChatService
	history     []struct {
		command string
		output  string
	}
	textInput      textinput.Model
	textarea       textarea.Model
	showWelcome    bool
	width          int
	height         int
	commandHistory []string       // All executed commands for navigation
	historyIndex   int            // Current position in history (-1 = new command)
	editMode       bool           // Whether we're in edit mode
	editFilename   string         // File being edited
	shellStack     []ShellContext // Stack for nested SSH sessions
	sessionID      uuid.UUID      // Session ID for chat

	// Incremental rendering state
	pendingOutput   string // New output to append (command results)
	pendingClear    bool   // Whether to clear screen
	lastPromptLine  string // Last rendered prompt (for detecting changes)
	initialRender   bool   // First render after transition from login
	commandPending  bool   // True when waiting for command result (don't update prompt)
	commandJustDone bool   // True when command just finished (need to move to next line)
}

// ShellContext represents a shell session context
type ShellContext struct {
	serverPath string
	vfs        *filesystem.VFS
	handler    *cmd.CommandHandler
}

// NewShellModel creates a new shell model
func NewShellModel(db *database.Database, userService *services.UserService, user interface{}, chatService *services.ChatService) *ShellModel {
	return NewShellModelWithSize(db, userService, user, 80, 24, chatService)
}

// NewShellModelWithSize creates a new shell model with specified window dimensions
func NewShellModelWithSize(db *database.Database, userService *services.UserService, user interface{}, width, height int, chatService *services.ChatService) *ShellModel {
	u, ok := user.(*models.User)
	if !ok {
		// Handle if user is not the right type
		u = nil
	}

	username := "user" // default fallback
	if u != nil && u.Username != "" {
		username = u.Username
	}

	vfs := filesystem.NewVFS(username)
	handler := cmd.NewCommandHandler(db, vfs, u, userService, chatService)

	// Sync user tools to VFS so they appear in help
	if u != nil {
		handler.SyncUserToolsToVFS()
	}

	// Initialize text input component
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = ""
	ti.CharLimit = 0
	ti.Width = width
	ti.Focus()

	// Initialize textarea component for edit mode
	ta := textarea.New()
	ta.Placeholder = "Start typing..."
	ta.CharLimit = 0
	ta.SetWidth(width)
	ta.SetHeight(height - 2) // Reserve space for status line and prompt

	// Generate session ID for chat
	sessionID := uuid.New()

	// Set up SSH callbacks for nested session handling
	shellModel := &ShellModel{
		userService: userService,
		user:        u,
		vfs:         vfs,
		handler:     handler,
		chatService: chatService,
		history: make([]struct {
			command string
			output  string
		}, 0),
		showWelcome:    true,
		width:          width,
		height:         height,
		commandHistory: make([]string, 0),
		historyIndex:   -1,
		shellStack:     make([]ShellContext, 0),
		textInput:      ti,
		textarea:       ta,
		initialRender:  true,
		sessionID:      sessionID,
	}

	// Set SSH callbacks
	handler.SetSSHCallbacks(
		func(serverPath string) error {
			// On SSH connect: push current context to stack
			shellModel.shellStack = append(shellModel.shellStack, ShellContext{
				serverPath: handler.GetCurrentServerPath(),
				vfs:        shellModel.vfs,
				handler:    shellModel.handler,
			})
			return nil
		},
		func() error {
			// On SSH disconnect: pop from stack (handled in exit command)
			return nil
		},
	)

	return shellModel
}

// Init initializes the shell
func (m *ShellModel) Init() tea.Cmd {
	// Send welcome message and request window size
	return tea.Batch(
		func() tea.Msg {
			return WelcomeMsg{}
		},
		tea.WindowSize(),
		m.textInput.Focus(),
	)
}

// Update handles messages
func (m *ShellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.textarea.SetHeight(msg.Height - 2) // Reserve space for status line and prompt
		return m, nil
	case tea.KeyMsg:
		// Handle edit mode separately
		if m.editMode {
			return m.handleEditModeInput(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			// If in SSH session, exit SSH (same as exit command)
			// Otherwise, clear current line
			if m.handler.GetCurrentServerPath() != "" {
				return m.handleExitSSH()
			}
			m.textInput.SetValue("")
			m.historyIndex = -1
			return m, nil
		case "up":
			// Navigate command history backward
			if len(m.commandHistory) > 0 {
				if m.historyIndex == -1 {
					m.historyIndex = len(m.commandHistory) - 1
				} else if m.historyIndex > 0 {
					m.historyIndex--
				}
				m.textInput.SetValue(m.commandHistory[m.historyIndex])
				m.textInput.CursorEnd()
			}
			return m, nil
		case "down":
			// Navigate command history forward
			if m.historyIndex >= 0 {
				m.historyIndex++
				if m.historyIndex >= len(m.commandHistory) {
					m.historyIndex = -1
					m.textInput.SetValue("")
				} else {
					m.textInput.SetValue(m.commandHistory[m.historyIndex])
					m.textInput.CursorEnd()
				}
			}
			return m, nil
		case "tab":
			// Autocomplete
			line := m.textInput.Value()
			parts := strings.Fields(line)

			if len(parts) == 0 || len(parts) == 1 {
				// Complete command name
				prefix := ""
				if len(parts) == 1 {
					prefix = parts[0]
				}
				matches := m.getCommandMatches(prefix)
				if len(matches) == 1 {
					// Single match - complete it
					m.textInput.SetValue(matches[0])
					m.textInput.CursorEnd()
				} else if len(matches) > 1 {
					// Multiple matches - complete common prefix
					commonPrefix := m.findCommonPrefix(matches)
					if commonPrefix != prefix {
						m.textInput.SetValue(commonPrefix)
						m.textInput.CursorEnd()
					}
				}
			} else {
				// Complete file/directory name
				prefix := parts[len(parts)-1]
				matches := m.getFileMatches(prefix)
				if len(matches) == 1 {
					// Single match - complete it
					current := m.textInput.Value()
					// Replace last part with match
					lastSpaceIdx := strings.LastIndex(current, " ")
					if lastSpaceIdx >= 0 {
						m.textInput.SetValue(current[:lastSpaceIdx+1] + matches[0])
					} else {
						m.textInput.SetValue(matches[0])
					}
					m.textInput.CursorEnd()
				} else if len(matches) > 1 {
					// Multiple matches - complete common prefix
					commonPrefix := m.findCommonPrefix(matches)
					if commonPrefix != prefix {
						current := m.textInput.Value()
						lastSpaceIdx := strings.LastIndex(current, " ")
						if lastSpaceIdx >= 0 {
							m.textInput.SetValue(current[:lastSpaceIdx+1] + commonPrefix)
						} else {
							m.textInput.SetValue(commonPrefix)
						}
						m.textInput.CursorEnd()
					}
				}
			}
			return m, nil
		case "enter":
			line := m.textInput.Value()
			if line == "" {
				// Empty enter - just add a newline effect
				return m, nil
			}
			// Don't clear textInput yet - keep command visible until result comes
			// Set flag to prevent prompt updates while command executes
			m.commandPending = true
			// Add command to history immediately
			m.history = append(m.history, struct {
				command string
				output  string
			}{
				command: line,
				output:  "",
			})
			// Add to command history for navigation (if not duplicate of last command)
			if len(m.commandHistory) == 0 || m.commandHistory[len(m.commandHistory)-1] != line {
				m.commandHistory = append(m.commandHistory, line)
			}
			m.historyIndex = -1
			m.showWelcome = false
			return m, m.executeCommand(line)
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			m.historyIndex = -1 // Reset history index when typing
			return m, cmd
		}
	case WelcomeMsg:
		m.showWelcome = true
		// Don't set pendingOutput here - initialRender handles it
		return m, nil
	case CommandResultMsg:
		// Handle clear command - clear history and terminal
		if len(m.history) > 0 {
			lastCommand := m.history[len(m.history)-1].command
			if lastCommand == "clear" {
				// Clear all history
				m.history = make([]struct {
					command string
					output  string
				}, 0)
				m.showWelcome = false
				m.pendingClear = true
				// Clear input state
				m.textInput.SetValue("")
				m.commandPending = false
				m.commandJustDone = false // Not needed since pendingClear handles it
				return m, nil
			}
		}

		// Process command output
		if len(m.history) > 0 {
			lastIdx := len(m.history) - 1
			// Find the last command without output
			for i := len(m.history) - 1; i >= 0; i-- {
				if m.history[i].output == "" {
					lastIdx = i
					break
				}
			}
			// Set output for the last command
			var output string
			if msg.Result.Error != nil {
				output = FormatError(msg.Result.Error)
			} else if msg.Result.Nodes != nil {
				output = FormatDirListWithOptions(msg.Result.Nodes, msg.Result.LongFormat)
			} else if msg.Result.Output != "" {
				// Handle special messages
				if strings.HasPrefix(msg.Result.Output, "__SSH_CONNECT__") {
					// Handle SSH connection - push to stack and update path
					newServerPath := strings.TrimPrefix(msg.Result.Output, "__SSH_CONNECT__")
					// Push current context to stack
					m.shellStack = append(m.shellStack, ShellContext{
						serverPath: m.handler.GetCurrentServerPath(),
						vfs:        m.vfs,
						handler:    m.handler,
					})
					// Update handler's server path
					m.handler.SetCurrentServerPath(newServerPath)
					// Add connection message to history
					parts := strings.Split(newServerPath, ".")
					serverIP := parts[len(parts)-1]
					output = fmt.Sprintf("Connected to %s\n", serverIP)
					output += fmt.Sprintf("Server path: %s\n", newServerPath)
				} else if msg.Result.Output == "__EXIT_SSH__" {
					// Handle exit from SSH session - return immediately
					return m.handleExitSSH()
				} else if msg.Result.Output == "__QUIT__" {
					// Quit the program
					return m, tea.Quit
				} else if msg.Result.Output == "__CHAT_MODE__" {
					// Enter full-screen chat mode
					if m.chatService != nil && m.user != nil {
						chatModel := NewChatModel(m, m.chatService, m.user, m.sessionID, m.width, m.height, false)
						return chatModel, chatModel.Init()
					}
					output = "Chat service not available\n"
				} else if msg.Result.Output == "__CHAT_MODE_SPLIT__" {
					// Enter split-screen chat mode
					if m.chatService != nil && m.user != nil {
						chatModel := NewChatModel(m, m.chatService, m.user, m.sessionID, m.width, m.height, true)
						return chatModel, chatModel.Init()
					}
					output = "Chat service not available\n"
				} else if strings.HasPrefix(msg.Result.Output, "__EDIT_MODE__") {
					filename := strings.TrimPrefix(msg.Result.Output, "__EDIT_MODE__")
					// Enter edit mode
					m.editMode = true
					m.editFilename = filename
					m.textarea.Reset()
					// Try to load existing file content
					if content, err := m.vfs.ReadFile(filename); err == nil {
						m.textarea.SetValue(content)
					}
					m.textInput.Blur()
					m.textarea.Focus()
					output = fmt.Sprintf("Edit mode for %s. Press Ctrl+S to save, Esc/Ctrl+Q to exit.\n", filename)
				} else if msg.Result.Output == "\033[2J\033[H" {
					// Handle clear command ANSI codes - return empty output
					output = ""
				} else {
					output = msg.Result.Output
					// Ensure output ends with newline if not empty
					if !strings.HasSuffix(output, "\n") {
						output += "\n"
					}
				}
			}
			// Set output in history
			m.history[lastIdx].output = output

			// Queue output as pending for incremental render
			// Trim any trailing whitespace/newlines to avoid blank lines
			m.pendingOutput = strings.TrimRight(output, "\n\r ")

			// Mark that command just finished (need to move to next line)
			m.commandJustDone = true
			m.commandPending = false
			m.textInput.SetValue("")
		}
		return m, nil
	}
	return m, nil
}

// executeCommand executes a shell command
func (m *ShellModel) executeCommand(command string) tea.Cmd {
	return func() tea.Msg {
		// Handle special commands
		if command == "quit" || command == "exit" {
			return tea.Quit()
		}

		// Execute command
		result := m.handler.Execute(command)

		return CommandResultMsg{
			Result: result,
		}
	}
}

// GetIncrementalOutput returns content for incremental rendering
// Returns: output to send, whether this is a clear, whether prompt changed, startRow for positioning
func (m *ShellModel) GetIncrementalOutput() (output string, isClear bool, promptOnly bool, startRow int) {
	username := "guest"
	if m.user != nil && m.user.Username != "" {
		username = m.user.Username
	}

	// Handle clear command
	if m.pendingClear {
		m.pendingClear = false
		promptLine := m.getPromptLine(username)
		m.lastPromptLine = promptLine
		// Clear screen, prompt at bottom (row = height)
		return promptLine, true, false, m.height
	}

	// Handle initial render (after login transition)
	if m.initialRender {
		m.initialRender = false
		var sb strings.Builder

		promptLine := m.getPromptLine(username)

		if m.showWelcome {
			welcome := AnimatedWelcome() + "\n" + WelcomeHelpText()
			welcomeLinesList := strings.Split(welcome, "\n")
			welcomeLines := len(welcomeLinesList)

			sb.WriteString(welcome)
			sb.WriteString("\n")
			sb.WriteString(promptLine)

			// Calculate start row to push content to bottom
			// Total lines = welcomeLines + 1 (prompt)
			totalLines := welcomeLines + 1
			startRow := m.height - totalLines + 1
			if startRow < 1 {
				startRow = 1
			}

			m.lastPromptLine = promptLine
			m.pendingOutput = ""
			return sb.String(), false, false, startRow
		} else {
			// No welcome - just prompt at bottom
			m.lastPromptLine = promptLine
			m.pendingOutput = ""
			return promptLine, false, false, m.height
		}
	}

	// Handle command just finished (need to output result + new prompt on next line)
	if m.commandJustDone {
		m.commandJustDone = false
		promptLine := m.getPromptLine(username)
		m.lastPromptLine = promptLine

		// Build output: move to next line, then output (if any), then prompt
		// Use \n prefix to signal bridge to move to next line first
		if m.pendingOutput != "" {
			// Output exists: \n + output + \n + prompt
			result := "\n" + m.pendingOutput + "\n" + promptLine
			m.pendingOutput = ""
			return result, false, false, 0
		} else {
			// No output: just \n + prompt (move to next line, show prompt)
			return "\n" + promptLine, false, false, 0
		}
	}

	// If command is pending (waiting for result), don't update prompt
	if m.commandPending {
		return "", false, false, 0
	}

	// No pending output - check if prompt changed (user typing)
	currentPrompt := m.getPromptLine(username)
	if currentPrompt != m.lastPromptLine {
		m.lastPromptLine = currentPrompt
		// Return prompt with row=-1 to signal "update prompt in place at current scroll position"
		return currentPrompt, false, true, -1
	}

	// Nothing changed
	return "", false, false, 0
}

// getPromptLine returns the current prompt line with input
func (m *ShellModel) getPromptLine(username string) string {
	m.textInput.Prompt = RenderPrompt(username, "terminal.sh", m.vfs.GetCurrentPath())
	return m.textInput.View()
}

// IsEditMode returns whether the shell is in edit mode
func (m *ShellModel) IsEditMode() bool {
	return m.editMode
}

// NeedsClearScrollback returns true if scrollback should be cleared (after clear command)
// Calling this resets the flag
func (m *ShellModel) NeedsClearScrollback() bool {
	if m.pendingClear {
		m.pendingClear = false
		return true
	}
	return false
}

// View renders the shell (used for edit mode and fallback)
func (m *ShellModel) View() string {
	// Get username for prompt
	username := "guest"
	if m.user != nil && m.user.Username != "" {
		username = m.user.Username
	}

	// Ensure we have valid dimensions
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	// Build all content lines
	var contentLines []string

	// Add welcome message as first entry if shown
	if m.showWelcome && len(m.history) == 0 {
		welcome := AnimatedWelcome() + "\n" + WelcomeHelpText()
		welcomeLines := strings.Split(welcome, "\n")
		contentLines = append(contentLines, welcomeLines...)
	}

	// Add all history entries
	for _, entry := range m.history {
		// Command line: prompt + command
		prompt := RenderPrompt(username, "terminal.sh", m.vfs.GetCurrentPath())
		commandLine := prompt + entry.command
		contentLines = append(contentLines, commandLine)

		// Output lines (if any)
		if entry.output != "" {
			outputText := strings.TrimSuffix(entry.output, "\n")
			outputLines := strings.Split(outputText, "\n")
			contentLines = append(contentLines, outputLines...)
		}
	}

	// Build the current prompt line
	m.textInput.Prompt = RenderPrompt(username, "terminal.sh", m.vfs.GetCurrentPath())
	m.textInput.Width = width
	promptLine := m.textInput.View()

	// Calculate available lines for content (minus 1 for prompt)
	availableLines := height - 1
	if availableLines < 1 {
		availableLines = 1
	}

	// Build output
	var output strings.Builder

	if m.editMode {
		// Edit mode rendering - full screen mode
		textareaHeight := height - 2
		if textareaHeight < 3 {
			textareaHeight = 3
		}
		m.textarea.SetHeight(textareaHeight)

		historyAvailableLines := availableLines - textareaHeight - 1

		// Show history (truncated from top if needed)
		if len(contentLines) > historyAvailableLines {
			startIdx := len(contentLines) - historyAvailableLines
			for i := startIdx; i < len(contentLines); i++ {
				output.WriteString(contentLines[i])
				output.WriteString("\n")
			}
		} else {
			// Pad with empty lines first
			paddingLines := historyAvailableLines - len(contentLines)
			for i := 0; i < paddingLines; i++ {
				output.WriteString("\n")
			}
			for _, line := range contentLines {
				output.WriteString(line)
				output.WriteString("\n")
			}
		}

		// Add textarea
		output.WriteString(m.textarea.View())
		output.WriteString("\n")

		// Add status line
		statusLine := fmt.Sprintf("-- Edit mode: %s | Ctrl+S to save, Esc/Ctrl+Q to exit --", m.editFilename)
		output.WriteString(statusLine)
	} else {
		// Normal mode - for full View() (SSH compatibility)
		// When content is short, pad to push prompt to bottom
		if len(contentLines) < availableLines {
			paddingLines := availableLines - len(contentLines)
			for i := 0; i < paddingLines; i++ {
				output.WriteString("\n")
			}
		}

		// Output all content
		for _, line := range contentLines {
			output.WriteString(line)
			output.WriteString("\n")
		}

		// Add prompt at the end
		output.WriteString(promptLine)
	}

	return output.String()
}

// Messages
type WelcomeMsg struct{}

type CommandResultMsg struct {
	Result *cmd.CommandResult
}

// getCommandMatches returns commands that start with the given prefix
func (m *ShellModel) getCommandMatches(prefix string) []string {
	binCommands, usrBinCommands := m.vfs.ListCommands()
	allCommands := append(binCommands, usrBinCommands...)

	var matches []string
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// getFileMatches returns files/directories in current directory that start with prefix
func (m *ShellModel) getFileMatches(prefix string) []string {
	nodes := m.vfs.ListDir()
	var matches []string
	for _, node := range nodes {
		name := node.Name
		if node.IsDir {
			name += "/"
		}
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}

// findCommonPrefix finds the common prefix of a slice of strings
func (m *ShellModel) findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for i := 1; i < len(strs); i++ {
		for len(prefix) > 0 && !strings.HasPrefix(strs[i], prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if len(prefix) == 0 {
			return ""
		}
	}
	return prefix
}

// handleExitSSH handles exiting from an SSH session
func (m *ShellModel) handleExitSSH() (tea.Model, tea.Cmd) {
	if len(m.shellStack) == 0 {
		// No more shells in stack, quit program
		return m, tea.Quit
	}

	// Pop from stack and restore context
	lastIdx := len(m.shellStack) - 1
	context := m.shellStack[lastIdx]
	m.shellStack = m.shellStack[:lastIdx]

	// Restore VFS and handler
	m.vfs = context.vfs
	m.handler = context.handler

	// Update handler's server path
	m.handler.SetCurrentServerPath(context.serverPath)

	// Add exit message as pending output
	m.pendingOutput = "Disconnected\n"
	m.showWelcome = false

	// Clear input state
	m.textInput.SetValue("")
	m.commandPending = false
	m.commandJustDone = true

	return m, nil
}

// handleEditModeInput handles input when in edit mode
func (m *ShellModel) handleEditModeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle special keys for edit mode
	switch msg.String() {
	case "ctrl+s":
		// Save file and exit edit mode
		filename := m.editFilename
		content := m.textarea.Value()
		if err := m.vfs.WriteFile(filename, content); err != nil {
			// Add error to pending output
			m.pendingOutput = FormatError(err)
		} else {
			// Exit edit mode
			m.editMode = false
			m.pendingOutput = fmt.Sprintf("File %s saved.\n", filename)
			m.editFilename = ""
			m.textarea.Reset()
			m.textarea.Blur()
			m.textInput.Focus()
		}
		m.showWelcome = false
		return m, nil
	case "ctrl+c", "esc", "ctrl+q":
		// Exit edit mode without saving
		m.editMode = false
		m.pendingOutput = "Edit mode exited without saving.\n"
		m.editFilename = ""
		m.textarea.Reset()
		m.textarea.Blur()
		m.textInput.Focus()
		m.showWelcome = false
		return m, nil
	default:
		// Delegate all other input to textarea
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}
