package terminal

import (
	"strings"
	"ssh4xx-go/cmd"
	"ssh4xx-go/filesystem"
	"ssh4xx-go/models"
	"ssh4xx-go/services"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShellModel handles the interactive shell after login
type ShellModel struct {
	userService *services.UserService
	user        *models.User
	vfs         *filesystem.VFS
	handler     *cmd.CommandHandler
	history     []struct {
		command string
		output  string
	}
	currentLine strings.Builder
	showWelcome bool
	width        int
	height       int
}

// NewShellModel creates a new shell model
func NewShellModel(userService *services.UserService, user interface{}) *ShellModel {
	return NewShellModelWithSize(userService, user, 80, 24)
}

// NewShellModelWithSize creates a new shell model with specified window dimensions
func NewShellModelWithSize(userService *services.UserService, user interface{}, width, height int) *ShellModel {
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
	handler := cmd.NewCommandHandler(vfs, u, userService)

	return &ShellModel{
		userService: userService,
		user:        u,
		vfs:         vfs,
		handler:     handler,
		history:     make([]struct{command string; output string}, 0),
		showWelcome: true,
		width:       width,
		height:      height,
	}
}

// Init initializes the shell
func (m *ShellModel) Init() tea.Cmd {
	// Send welcome message and request window size
	return tea.Batch(
		func() tea.Msg {
			return WelcomeMsg{}
		},
		tea.WindowSize(),
	)
}

// Update handles messages
func (m *ShellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Clear current line
			m.currentLine.Reset()
			return m, nil
		case "enter":
			line := m.currentLine.String()
			m.currentLine.Reset()
			if line == "" {
				return m, nil
			}
			// Add command to history immediately
			m.history = append(m.history, struct{command string; output string}{
				command: line,
				output:  "",
			})
			m.showWelcome = false
			return m, m.executeCommand(line)
		case "backspace":
			if m.currentLine.Len() > 0 {
				current := m.currentLine.String()
				m.currentLine.Reset()
				m.currentLine.WriteString(current[:len(current)-1])
			}
			return m, nil
		default:
			// Add character to current line
			if len(msg.Runes) > 0 {
				m.currentLine.WriteRune(msg.Runes[0])
			}
			return m, nil
		}
	case WelcomeMsg:
		m.showWelcome = true
		return m, nil
	case CommandResultMsg:
		// Handle clear command specially - clear history instead of showing ANSI codes
		if len(m.history) > 0 {
			lastCommand := m.history[len(m.history)-1].command
			if lastCommand == "clear" {
				// Clear all history and reset welcome
				m.history = make([]struct{command string; output string}, 0)
				m.showWelcome = false
				return m, nil
			}
		}
		
		// Add command and output to history
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
				output = FormatDirList(msg.Result.Nodes)
			} else if msg.Result.Output != "" {
				if msg.Result.Output == "__ANIMATED_HELP__" {
					output = AnimatedHelp()
				} else if msg.Result.Output == "\033[2J\033[H" {
					// Skip ANSI clear codes - they're handled specially above
					output = ""
				} else {
					output = msg.Result.Output
				}
			}
			// Always set output, even if empty (to mark command as processed)
			m.history[lastIdx].output = output
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

// View renders the shell
func (m *ShellModel) View() string {
	// Get username for prompt
	username := "guest"
	if m.user != nil && m.user.Username != "" {
		username = m.user.Username
	}
	
	// Reserve 1 line for the prompt at the bottom
	// If height is 0 (not yet received), show all history
	availableLines := m.height - 1
	if m.height == 0 || availableLines < 1 {
		// Show all history if height unknown or too small
		availableLines = 1000 // Large number to show all
	}

	var historyLines strings.Builder
	
	// Build all history entries with line counts
	type historyEntry struct {
		text      string
		lineCount int
	}
	
	var entries []historyEntry
	
	// Add welcome message if shown (we'll handle it specially in View)
	if m.showWelcome {
		// Don't add to entries, we'll handle it specially in View
	}
	
	// Build history entries
	for _, entry := range m.history {
		var entryText strings.Builder
		prompt := RenderPrompt(username, "ssh4xx", m.vfs.GetCurrentPath())
		entryText.WriteString(prompt)
		entryText.WriteString(entry.command)
		entryText.WriteString("\n")
		if entry.output != "" {
			entryText.WriteString(entry.output)
			entryText.WriteString("\n")
		}
		text := entryText.String()
		entries = append(entries, historyEntry{
			text:      text,
			lineCount: countLines(text),
		})
	}
	
	// Calculate which entries to show (from the end, fitting in available space)
	usedLines := 0
	startIdx := len(entries)
	for i := len(entries) - 1; i >= 0; i-- {
		if usedLines+entries[i].lineCount <= availableLines {
			usedLines += entries[i].lineCount
			startIdx = i
		} else {
			break
		}
	}
	
	// Build the history section
	for i := startIdx; i < len(entries); i++ {
		historyLines.WriteString(entries[i].text)
	}
	
	historyText := historyLines.String()
	
	// Ensure we have a valid height (fallback to default if not set yet)
	height := m.height
	if height <= 0 {
		height = 24 // Default terminal height
	}
	width := m.width
	if width <= 0 {
		width = 80 // Default terminal width
	}
	
	// If showing welcome message, center only the ASCII art
	if m.showWelcome && len(m.history) == 0 {
		asciiArt := AnimatedWelcome()
		helpText := WelcomeHelpText()
		
		// Center only the ASCII art
		centeredArt := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, asciiArt)
		
		// Calculate where to place help text and prompt (at bottom, left-aligned)
		helpTextLineCount := countLines(helpText)
		promptLineCount := 1
		bottomContentLines := helpTextLineCount + promptLineCount
		bottomStartLine := height - bottomContentLines
		
		// Build final output line by line
		centeredLines := strings.Split(centeredArt, "\n")
		var result strings.Builder
		
		prompt := RenderPrompt(username, "ssh4xx", m.vfs.GetCurrentPath())
		promptLine := prompt + m.currentLine.String() + "_"
		helpLines := strings.Split(strings.TrimSuffix(helpText, "\n"), "\n")
		
		for i := 0; i < height; i++ {
			if i < bottomStartLine {
				// Show centered art lines
				if i < len(centeredLines) {
					result.WriteString(centeredLines[i])
				}
			} else {
				// Show help text or prompt (left-aligned)
				idx := i - bottomStartLine
				if idx < len(helpLines) {
					result.WriteString(helpLines[idx])
				} else if idx == len(helpLines) {
					result.WriteString(promptLine)
				}
			}
			if i < height-1 {
				result.WriteString("\n")
			}
		}
		
		return result.String()
	}
	
	// For normal shell mode, put prompt at absolute bottom
	// Count lines in history (trim trailing newline if present for accurate count)
	historyTextTrimmed := strings.TrimSuffix(historyText, "\n")
	historyLineCount := countLines(historyTextTrimmed)
	
	// Calculate padding needed to push prompt to absolute bottom
	// Total lines in output = padding + history + prompt (1 line)
	// We want the last line to be the prompt
	// So: padding + historyLineCount + 1 = height
	paddingNeeded := height - historyLineCount - 1
	if paddingNeeded < 0 {
		paddingNeeded = 0
	}
	
	// Build the result with padding at the top to push everything to bottom
	var result strings.Builder
	// Add padding newlines at the top
	for i := 0; i < paddingNeeded; i++ {
		result.WriteString("\n")
	}
	// Add history content
	result.WriteString(historyTextTrimmed)
	
	// Current prompt and input (always at absolute bottom)
	prompt := RenderPrompt(username, "ssh4xx", m.vfs.GetCurrentPath())
	result.WriteString(prompt)
	result.WriteString(m.currentLine.String())
	result.WriteString("_") // Cursor

	return result.String()
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, r := range s {
		if r == '\n' {
			count++
		}
	}
	return count
}

// Messages
type WelcomeMsg struct{}

type CommandResultMsg struct {
	Result *cmd.CommandResult
}

