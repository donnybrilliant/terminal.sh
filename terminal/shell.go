package terminal

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"terminal-sh/cmd"
	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"
	"terminal-sh/services"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

const (
	gradientFrameCount = 18
	GradientFrameDelay = 75 * time.Millisecond
	gradientWidthCap   = 180
	gradientHeightCap  = 80
)

// ShellModel handles the interactive shell after login
type ShellModel struct {
	db          *database.Database
	userService *services.UserService
	user        *models.User
	vfs         *filesystem.VFS
	handler     *cmd.CommandHandler
	chatService *services.ChatService
	history     []struct {
		command string
		output  string
	}
	textInput    textinput.Model
	textarea     textarea.Model
	showWelcome  bool
	width        int
	height       int
	inputHistory *InputHistory  // Command history for up/down navigation
	editMode     bool           // Whether we're in edit mode
	editFilename string         // File being edited
	shellStack   []ShellContext // Stack for nested SSH sessions
	sessionID    uuid.UUID      // Session ID for chat

	// Welcome gradient animation state
	gradientFrames    []string
	gradientFrameIdx  int
	gradientAnimating bool
	gradientSeed      int64

	// Incremental rendering state
	pendingOutput      string // New output to append (command results)
	pendingClear       bool   // Whether to clear screen
	pendingClearScrollback bool // Whether to clear scrollback (after animation)
	lastPromptLine     string // Last rendered prompt (for detecting changes)
	initialRender      bool   // First render after transition from login
	commandPending     bool   // True when waiting for command result (don't update prompt)
	commandJustDone    bool   // True when command just finished (need to move to next line)
	lastViewContent    string // Last full view content (to detect if only prompt changed)

	// In-app scrollback state
	scrollOffset    int  // Lines scrolled up from bottom (0 = at bottom)
	isScrolledUp    bool // True if user has scrolled up (shows indicator)
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

	// Create VFS and merge user's saved filesystem changes
	var vfs *filesystem.VFS
	if u != nil && u.FileSystem != nil && len(u.FileSystem) > 0 {
		var err error
		vfs, err = filesystem.NewVFSFromMap(username, u.FileSystem)
		if err != nil {
			// If merge fails, fall back to standard VFS
			vfs = filesystem.NewVFS(username)
		}
	} else {
		vfs = filesystem.NewVFS(username)
	}
	
	// Set up save callback for user filesystem
	if u != nil {
		vfs.SetUserID(u.ID.String())
		vfs.SetSaveCallback(func(changes map[string]interface{}) error {
			// Update user's filesystem in database
			u.FileSystem = changes
			return db.Model(u).Update("file_system", changes).Error
		})
	}
	
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
		db:          db,
		userService: userService,
		user:        u,
		vfs:         vfs,
		handler:     handler,
		chatService: chatService,
		history: make([]struct {
			command string
			output  string
		}, 0),
		showWelcome:      true,
		width:            width,
		height:           height,
		inputHistory:     NewInputHistory(100), // Keep last 100 commands
		shellStack:       make([]ShellContext, 0),
		textInput:        ti,
		textarea:         ta,
		initialRender:    true,
		sessionID:        sessionID,
		gradientAnimating: true,
		gradientFrameIdx:  0,
		gradientSeed:      time.Now().UnixNano(),
	}

	// Precompute initial gradient frames
	shellModel.refreshGradientFrames()

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
	cmds := []tea.Cmd{
		func() tea.Msg {
			return WelcomeMsg{}
		},
		tea.WindowSize(),
		m.textInput.Focus(),
	}

	if tick := m.nextGradientTick(); tick != nil {
		cmds = append(cmds, tick)
	}

	return tea.Batch(cmds...)
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
		if m.gradientAnimating {
			m.refreshGradientFrames()
		}
		return m, nil
	case GradientTickMsg:
		if m.gradientAnimating {
			// Advance frame; stop when finished
			if len(m.gradientFrames) == 0 {
				m.refreshGradientFrames()
			}

			if m.gradientFrameIdx+1 < len(m.gradientFrames) {
				m.gradientFrameIdx++
				return m, m.nextGradientTick()
			}

			// Animation complete - clear scrollback to remove animation frames
			m.gradientAnimating = false
			m.showWelcome = true // keep help text, but no ASCII banner
			m.gradientFrames = nil
			m.gradientFrameIdx = 0
			m.pendingClearScrollback = true // Clear scrollback to remove animation
			return m, nil
		}
		return m, nil
	case tea.KeyMsg:
		// Handle edit mode separately
		if m.editMode {
			return m.handleEditModeInput(msg)
		}

		switch msg.String() {
		case "ctrl+q":
			// If in SSH session, exit SSH (same as exit command)
			if m.handler.GetCurrentServerPath() != "" {
				return m.handleExitSSH()
			}
			// In base shell, return to login
			return m.handleLogout()
		case "ctrl+c":
			// Ctrl+C: clear input if no text selected (terminal default behavior)
			// Text selection is handled by terminal emulator, so if we get here,
			// it means no selection - clear the input
			m.textInput.SetValue("")
			m.inputHistory.Reset()
			return m, nil
		case "up":
			// Navigate command history backward
			if cmd, ok := m.inputHistory.Previous(); ok {
				m.textInput.SetValue(cmd)
				m.textInput.CursorEnd()
			}
			return m, nil
		case "down":
			// Navigate command history forward
			if cmd, ok := m.inputHistory.Next(); ok {
				m.textInput.SetValue(cmd)
				m.textInput.CursorEnd()
			} else {
				m.textInput.SetValue("")
			}
			return m, nil
		case "pgup", "shift+up":
			// Scroll up through history
			m.scrollUp(m.height / 2) // Half page at a time
			return m, nil
		case "pgdown", "shift+down":
			// Scroll down through history
			m.scrollDown(m.height / 2) // Half page at a time
			return m, nil
		case "home":
			// Scroll to top of history (only if ctrl is held for home)
			// Regular home goes to start of input line
			if m.scrollOffset > 0 {
				m.scrollToTop()
				return m, nil
			}
		case "end":
			// Scroll to bottom (only if scrolled up)
			if m.scrollOffset > 0 {
				m.scrollToBottom()
				return m, nil
			}
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
				commands := m.getCommandMatches(prefix)
				if completed, ok := CompleteFromList(prefix, commands); ok {
					m.textInput.SetValue(completed)
					m.textInput.CursorEnd()
				}
			} else {
				// Check if we should autocomplete command arguments
				cmd := parts[0]
				prefix := parts[len(parts)-1]
				
				// Handle tutorial name autocomplete
				if cmd == "tutorial" && len(parts) >= 2 {
					tutorialNames := m.getTutorialNames(prefix)
					if completed, ok := CompleteFromList(prefix, tutorialNames); ok {
						current := m.textInput.Value()
						lastSpaceIdx := strings.LastIndex(current, " ")
						if lastSpaceIdx >= 0 {
							m.textInput.SetValue(current[:lastSpaceIdx+1] + completed)
						} else {
							m.textInput.SetValue(completed)
						}
						m.textInput.CursorEnd()
						return m, nil
					}
				}
				
				// Default: Complete file/directory name
				files := m.getFileMatches(prefix)
				if completed, ok := CompleteFromList(prefix, files); ok {
					current := m.textInput.Value()
					lastSpaceIdx := strings.LastIndex(current, " ")
					if lastSpaceIdx >= 0 {
						m.textInput.SetValue(current[:lastSpaceIdx+1] + completed)
					} else {
						m.textInput.SetValue(completed)
					}
					m.textInput.CursorEnd()
				}
			}
			return m, nil
		case "enter":
			line := m.textInput.Value()
			if line == "" {
				// Empty enter - just add a newline effect
				return m, nil
			}
			// Scroll to bottom when executing command
			m.scrollToBottom()
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
			// Add to command history for navigation
			m.inputHistory.Add(line)
			// Don't hide welcome screen yet - wait until command completes
			// This ensures welcome screen stays visible until first command executes
			return m, m.executeCommand(line)
		default:
			// Any typing scrolls to bottom
			if m.scrollOffset > 0 {
				m.scrollToBottom()
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			m.inputHistory.Reset() // Reset history index when typing
			return m, cmd
		}
	case tea.MouseMsg:
		// Only handle mouse wheel scrolling - ignore all other mouse events
		// This allows the terminal emulator to handle text selection (clicks/drags)
		// We check both the button type and action to ensure we only handle wheel events
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.scrollUp(3) // Scroll 3 lines at a time
				return m, nil
			case tea.MouseButtonWheelDown:
				m.scrollDown(3) // Scroll 3 lines at a time
				return m, nil
			}
		}
		// For all other mouse events (clicks, drags, releases, etc.), ignore them
		// This allows the terminal emulator to handle text selection
		// Note: We still return m, nil to acknowledge the message, but we don't
		// process it, which should allow the terminal emulator to handle it
		return m, nil
	case PasteTextMsg:
		// Handle paste: append text to current input value
		currentValue := m.textInput.Value()
		// Filter out control characters, keep only printable ASCII and newlines
		var filteredText strings.Builder
		for _, r := range msg.Text {
			if r >= 32 && r <= 126 {
				// Printable ASCII
				filteredText.WriteRune(r)
			} else if r == '\n' || r == '\r' {
				// Newlines - convert to Enter key press
				// First append what we have so far
				if filteredText.Len() > 0 {
					m.textInput.SetValue(currentValue + filteredText.String())
					currentValue = m.textInput.Value()
					filteredText.Reset()
				}
				// Execute current line if it has content
				if currentValue != "" {
					// Scroll to bottom when executing command
					m.scrollToBottom()
					m.commandPending = true
					// Add command to history immediately
					m.history = append(m.history, struct {
						command string
						output  string
					}{
						command: currentValue,
						output:  "",
					})
					m.inputHistory.Add(currentValue)
					return m, m.executeCommand(currentValue)
				}
				// If empty, just continue (don't execute empty command)
			}
		}
		// Append remaining filtered text
		if filteredText.Len() > 0 {
			m.textInput.SetValue(currentValue + filteredText.String())
			m.textInput.CursorEnd()
		} else if currentValue != "" {
			// Even if no new text, ensure cursor is at end
			m.textInput.CursorEnd()
		}
		m.inputHistory.Reset()
		return m, nil
	case WelcomeMsg:
		m.showWelcome = true
		// Don't set pendingOutput here - initialRender handles it
		return m, nil
	case LogoutMsg:
		// Return to login screen
		return m.handleLogout()
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
					
					// Get server filesystem and create new VFS
					serverVFS, err := m.handler.CreateServerVFS(newServerPath)
					if err != nil {
						output = FormatError(fmt.Errorf("failed to load server filesystem: %w", err))
					} else {
						// Push current context to stack
						m.shellStack = append(m.shellStack, ShellContext{
							serverPath: m.handler.GetCurrentServerPath(),
							vfs:        m.vfs,
							handler:    m.handler,
						})
						
						// Switch to server VFS
						m.vfs = serverVFS
						m.handler.SetVFS(serverVFS)
						
						// Update handler's server path
						m.handler.SetCurrentServerPath(newServerPath)
						
						// Add connection message to history
						parts := strings.Split(newServerPath, ".")
						serverIP := parts[len(parts)-1]
						output = fmt.Sprintf("Connected to %s\n", serverIP)
						output += fmt.Sprintf("Server path: %s\n", newServerPath)
					}
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

			// Hide welcome screen after first command completes (when we get output)
			if len(m.history) == 1 && output != "" {
				m.showWelcome = false
			}

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
			// If in SSH session, exit SSH
			if m.handler.GetCurrentServerPath() != "" {
				return CommandResultMsg{
					Result: &cmd.CommandResult{Output: "__EXIT_SSH__"},
				}
			}
			// In base shell, return to login
			return LogoutMsg{}
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
			ascii := AnimatedWelcome()
			help := strings.TrimSuffix(WelcomeHelpText(m.user, m.db), "\n")

			centered := lipgloss.Place(
				m.width,
				m.height-3, // reserve lines for help + spacer + prompt
				lipgloss.Center,
				lipgloss.Center,
				ascii,
			)

			sb.WriteString(centered)
			sb.WriteString("\n")
			sb.WriteString(help)
			sb.WriteString("\n\n") // spacer line above prompt
			sb.WriteString(promptLine)

			m.lastPromptLine = promptLine
			m.pendingOutput = ""
			return sb.String(), false, false, 1
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

// NeedsClearScrollback returns true if scrollback should be cleared (after clear command or animation)
// Calling this resets the flag
func (m *ShellModel) NeedsClearScrollback() bool {
	if m.pendingClear {
		m.pendingClear = false
		return true
	}
	if m.pendingClearScrollback {
		m.pendingClearScrollback = false
		return true
	}
	return false
}

// IsGradientAnimating reports whether the welcome gradient is currently running.
func (m *ShellModel) IsGradientAnimating() bool {
	return m.gradientAnimating
}

// scrollUp scrolls the view up by n lines
func (m *ShellModel) scrollUp(n int) {
	// Calculate total content lines to determine max scroll
	totalLines := m.getTotalContentLines()
	viewportHeight := m.height - 1 // -1 for prompt line
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	
	// Max scroll is total lines minus viewport (can't scroll past the beginning)
	maxScroll := totalLines - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	
	m.scrollOffset += n
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	m.isScrolledUp = m.scrollOffset > 0
}

// scrollDown scrolls the view down by n lines
func (m *ShellModel) scrollDown(n int) {
	m.scrollOffset -= n
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	m.isScrolledUp = m.scrollOffset > 0
}

// scrollToTop scrolls to the top of history
func (m *ShellModel) scrollToTop() {
	totalLines := m.getTotalContentLines()
	viewportHeight := m.height - 1
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	
	maxScroll := totalLines - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollOffset = maxScroll
	m.isScrolledUp = m.scrollOffset > 0
}

// scrollToBottom scrolls to the bottom (most recent)
func (m *ShellModel) scrollToBottom() {
	m.scrollOffset = 0
	m.isScrolledUp = false
}

// getTotalContentLines calculates the total number of content lines in history
func (m *ShellModel) getTotalContentLines() int {
	total := 0
	
	// Welcome content if shown
	if m.showWelcome && len(m.history) == 0 {
		// ASCII art + help text + spacer
		total += m.height - 1 // Welcome takes full screen minus prompt
	}
	
	// Count history lines
	for _, entry := range m.history {
		total++ // Command line itself
		if entry.output != "" {
			outputLines := strings.Split(strings.TrimSuffix(entry.output, "\n"), "\n")
			total += len(outputLines)
		}
	}
	
	return total
}

// renderScrollIndicator renders a scroll position indicator
func (m *ShellModel) renderScrollIndicator(totalLines, startLine, viewportHeight int) string {
	// Calculate position percentage
	endLine := startLine + viewportHeight
	if endLine > totalLines {
		endLine = totalLines
	}
	
	// Style the indicator
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)
	
	// Show lines range and scroll hint
	indicator := fmt.Sprintf("↑ lines %d-%d of %d (PgUp/PgDn to scroll, End to return)", 
		startLine+1, endLine, totalLines)
	
	return indicatorStyle.Render(indicator)
}

// GetViewContent returns the view content without the prompt line
// Used to detect if only the prompt changed (user typing)
func (m *ShellModel) GetViewContent() string {
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

	// Calculate available lines for content (minus 1 for prompt)
	availableLines := height - 1
	if availableLines < 1 {
		availableLines = 1
	}

	// Animated gradient welcome fills the screen, then disappears
	if m.gradientAnimating && len(m.history) == 0 {
		frame := m.getCurrentGradientFrame()
		frameLines := strings.Split(frame, "\n")

		// Ensure frame height matches available lines
		if len(frameLines) < availableLines {
			padding := availableLines - len(frameLines)
			for i := 0; i < padding; i++ {
				frameLines = append(frameLines, "")
			}
		} else if len(frameLines) > availableLines {
			frameLines = frameLines[:availableLines]
		}

		var output strings.Builder
		for _, line := range frameLines {
			output.WriteString(line)
			output.WriteString("\n")
		}
		// Note: GetViewContent doesn't include prompt, but View() will add it
		return output.String()
	}

	// Build all content lines (without prompt)
	var contentLines []string

	// Add welcome message as first entry if shown
	if m.showWelcome && len(m.history) == 0 {
		ascii := AnimatedWelcome()
		centered := lipgloss.Place(
			width,
			availableLines-3, // leave room for help + spacer
			lipgloss.Center,
			lipgloss.Center,
			ascii,
		)
		contentLines = append(contentLines, centered)

		// Left-aligned help block directly above the prompt with one spacer line
		helpLines := strings.Split(strings.TrimSuffix(WelcomeHelpText(m.user, m.db), "\n"), "\n")
		for _, line := range helpLines {
			contentLines = append(contentLines, line)
		}
		contentLines = append(contentLines, "") // spacer line between help and prompt
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

	// Build output (without prompt)
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
	} else {
		// Normal mode with scrollback support
		// Calculate viewport window based on scroll offset
		totalLines := len(contentLines)
		viewportHeight := availableLines
		
		// Reserve 1 line for scroll indicator if scrolled up
		if m.isScrolledUp {
			viewportHeight--
		}
		
		// Calculate which lines to show
		// scrollOffset=0 means show the most recent lines (bottom)
		// scrollOffset>0 means we're scrolled up into history
		var startLine, endLine int
		
		if totalLines <= viewportHeight {
			// All content fits - show everything
			startLine = 0
			endLine = totalLines
		} else {
			// Need to window into the content
			// endLine is where we stop (exclusive), startLine is where we start
			endLine = totalLines - m.scrollOffset
			startLine = endLine - viewportHeight
			
			// Clamp to valid range
			if startLine < 0 {
				startLine = 0
			}
			if endLine > totalLines {
				endLine = totalLines
			}
			if endLine < startLine {
				endLine = startLine
			}
		}
		
		// Output the visible window
		for i := startLine; i < endLine; i++ {
			output.WriteString(contentLines[i])
			output.WriteString("\n")
		}
		
		// Add scroll indicator if scrolled up
		if m.isScrolledUp {
			indicator := m.renderScrollIndicator(totalLines, startLine, viewportHeight)
			output.WriteString(indicator)
			output.WriteString("\n")
		}
	}

	return output.String()
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

	// Get content without prompt
	content := m.GetViewContent()
	
	// Calculate available lines for content (minus 1 for prompt)
	contentHeight := m.height
	if contentHeight <= 0 {
		contentHeight = 24
	}
	availableLines := contentHeight - 1
	if availableLines < 1 {
		availableLines = 1
	}
	
	// Count content lines (excluding trailing newlines)
	contentLines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	contentLineCount := len(contentLines)
	if contentLineCount == 0 || (contentLineCount == 1 && contentLines[0] == "") {
		contentLineCount = 0
	}
	
	// Add padding to push prompt to bottom if content is short
	var output strings.Builder
	if contentLineCount < availableLines && !m.editMode {
		paddingLines := availableLines - contentLineCount
		for i := 0; i < paddingLines; i++ {
			output.WriteString("\n")
		}
	}
	
	// Add content
	output.WriteString(content)
	
	// Build the current prompt line
	m.textInput.Prompt = RenderPrompt(username, "terminal.sh", m.vfs.GetCurrentPath())
	m.textInput.Width = width
	promptLine := m.textInput.View()

	// Combine content and prompt
	output.WriteString(promptLine)
	return output.String()
}

// Messages
type WelcomeMsg struct{}

type CommandResultMsg struct {
	Result *cmd.CommandResult
}

type LogoutMsg struct{}

type GradientTickMsg struct{}

type PasteTextMsg struct {
	Text string
}

// getCommandMatches returns commands that start with the given prefix
func (m *ShellModel) getCommandMatches(prefix string) []string {
	// Built-in commands (from cmd/commands.go)
	// Note: crypto_miner, stop_mining, miners are built-in commands, not tool commands
	builtInCommands := []string{
		"pwd", "ls", "cd", "cat", "clear", "help", "chat", "tutorial",
		"login", "logout", "register", "userinfo", "info", "whoami", "name",
		"ifconfig", "scan", "server", "createServer", "createLocalServer",
		"ssh", "exit", "get", "tools", "exploited", "shop", "buy",
		"patches", "patch", "crypto_miner", "stop_mining", "miners", "wallet",
		"touch", "mkdir", "rm", "cp", "mv", "edit", "vi", "nano",
	}
	
	// Get user's owned tools (only include tool commands the user owns)
	var userToolCommands []string
	if m.user != nil && m.handler != nil {
		// Get user's tools from the handler's tool service
		tools, err := m.handler.GetUserToolNames()
		if err == nil {
			userToolCommands = tools
		}
		// If error, just use empty list (user has no tools or error getting them)
	}
	
	// Get VFS commands (filesystem commands)
	binCommands, usrBinCommands := m.vfs.ListCommands()
	allCommands := append(builtInCommands, userToolCommands...)
	allCommands = append(allCommands, binCommands...)
	allCommands = append(allCommands, usrBinCommands...)

	var matches []string
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// getTutorialNames returns tutorial IDs that start with the given prefix
func (m *ShellModel) getTutorialNames(prefix string) []string {
	if m.handler == nil {
		return []string{}
	}
	
	// Get tutorial service from handler (we need to access it)
	// Since we can't directly access tutorialService from handler, we'll need to
	// reload tutorials each time. This is acceptable since it's only on tab completion.
	// We'll use a helper method on CommandHandler to get tutorial names
	tutorialNames := m.handler.GetTutorialNames(prefix)
	return tutorialNames
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

// handleLogout returns to the login screen
func (m *ShellModel) handleLogout() (tea.Model, tea.Cmd) {
	// Create a new login model with current window size
	loginModel := NewLoginModel(m.db, m.userService, m.chatService, "", "")
	loginModel.width = m.width
	loginModel.height = m.height
	return loginModel, loginModel.Init()
}

// handleExitSSH handles exiting from an SSH session
func (m *ShellModel) handleExitSSH() (tea.Model, tea.Cmd) {
	if len(m.shellStack) == 0 {
		// No more shells in stack, quit program (SSH closes connection)
		return m, tea.Quit
	}

	// Pop from stack and restore context
	lastIdx := len(m.shellStack) - 1
	context := m.shellStack[lastIdx]
	m.shellStack = m.shellStack[:lastIdx]

	// Restore VFS and handler
	m.vfs = context.vfs
	m.handler = context.handler
	
	// Update handler's VFS reference
	m.handler.SetVFS(context.vfs)

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
	case "esc", "ctrl+q":
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

// nextGradientTick schedules the next animation frame if the welcome animation is active.
func (m *ShellModel) nextGradientTick() tea.Cmd {
	if !m.gradientAnimating {
		return nil
	}
	return tea.Tick(GradientFrameDelay, func(time.Time) tea.Msg {
		return GradientTickMsg{}
	})
}

// refreshGradientFrames rebuilds the gradient frames for the current viewport.
func (m *ShellModel) refreshGradientFrames() {
	if !m.gradientAnimating {
		m.gradientFrames = nil
		return
	}

	m.gradientFrames = m.buildGradientFrames()
	if m.gradientFrameIdx >= len(m.gradientFrames) {
		m.gradientFrameIdx = 0
	}
}

// buildGradientFrames produces a set of animated gradient frames that fill the screen.
func (m *ShellModel) buildGradientFrames() []string {
	width := m.width
	height := m.height

	// Ensure reasonable bounds to keep rendering fast
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	if width > gradientWidthCap {
		width = gradientWidthCap
	}
	if height > gradientHeightCap {
		height = gradientHeightCap
	}

	usableHeight := height - 1 // reserve one line for the prompt
	if usableHeight < 4 {
		usableHeight = 4
	}

	// Palette: existing magenta gradient plus monochrome tones
	primary := []string{"205", "213", "207", "219", "218", "212", "205"}
	greys := []string{"232", "235", "237", "240", "244", "248", "252", "255"}
	palette := append(primary, greys...)

	rng := rand.New(rand.NewSource(m.gradientSeed))
	frames := make([]string, gradientFrameCount)

	for f := 0; f < gradientFrameCount; f++ {
		var sb strings.Builder
		phase := rng.Float64() * 6

		for y := 0; y < usableHeight; y++ {
			for x := 0; x < width; x++ {
				// Wave + noise to keep the gradient organic
				wave := math.Sin((float64(x)+phase)/5.0) + math.Cos((float64(y)+float64(f)*1.4)/4.0)
				baseIdx := int(math.Abs(wave) * float64(len(palette)))
				baseIdx = clampInt(baseIdx, 0, len(palette)-1)

				// Occasionally inject bright accent pixels to mimic the ASCII style
				if (x+y+f)%13 == 0 || rng.Float64() > 0.92 {
					accentIdx := int(math.Abs(math.Sin(float64(f)+float64(x)/3+float64(y)/3)) * float64(len(primary)))
					accentIdx = clampInt(accentIdx, 0, len(primary)-1)
					sb.WriteString("\x1b[38;5;")
					sb.WriteString(primary[accentIdx])
					sb.WriteString("m█")
					continue
				}

				sb.WriteString("\x1b[38;5;")
				sb.WriteString(palette[baseIdx%len(palette)])
				sb.WriteString("m█")
			}

			// Reset color at end of each line
			sb.WriteString("\x1b[0m")
			if y < usableHeight-1 {
				sb.WriteString("\n")
			}
		}

		frames[f] = sb.String()
	}

	return frames
}

// getCurrentGradientFrame returns the active frame (building frames on demand).
func (m *ShellModel) getCurrentGradientFrame() string {
	if !m.gradientAnimating {
		return ""
	}
	if len(m.gradientFrames) == 0 {
		m.refreshGradientFrames()
	}
	if len(m.gradientFrames) == 0 {
		return ""
	}
	if m.gradientFrameIdx >= len(m.gradientFrames) {
		m.gradientFrameIdx = 0
	}
	return m.gradientFrames[m.gradientFrameIdx]
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
