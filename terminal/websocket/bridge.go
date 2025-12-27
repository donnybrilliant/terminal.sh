package websocket

import (
	"fmt"
	"strings"
	"sync"
	"terminal-sh/database"
	"terminal-sh/services"
	"terminal-sh/terminal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// ANSI escape sequences for terminal control
const (
	// Alternate screen buffer (used for login - no scrollback)
	enterAltScreen = "\x1b[?1049h"
	exitAltScreen  = "\x1b[?1049l"
	
	// Cursor control
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	
	// Screen control
	clearScreen     = "\x1b[2J"          // Clear visible screen
	clearScrollback = "\x1b[3J"          // Clear scrollback buffer
	clearAll        = "\x1b[2J\x1b[3J"   // Clear screen + scrollback
	cursorHome      = "\x1b[H"
	clearLine       = "\x1b[2K"          // Clear entire line
	clearToEOL      = "\x1b[K"           // Clear to end of line
)

// RenderMode determines how content is rendered
type RenderMode int

const (
	RenderModeFullScreen RenderMode = iota // Login: alternate screen, clear each render
	RenderModeIncremental                  // Shell: incremental output, scrollback enabled
)

// BubbleTeaBridge wraps a Bubble Tea model for WebSocket communication
type BubbleTeaBridge struct {
	model       tea.Model
	conn        *websocket.Conn
	db          *database.Database
	userService *services.UserService
	done        chan struct{}
	msgChan     chan tea.Msg
	closeOnce   sync.Once
	width       int
	height      int
	renderMode  RenderMode
	lastView    string
	promptRow   int // Track which row the prompt is on for updates
}

// NewBubbleTeaBridge creates a new bridge between Bubble Tea and WebSocket
func NewBubbleTeaBridge(conn *websocket.Conn, db *database.Database, userService *services.UserService, width, height int) (*BubbleTeaBridge, error) {
	// Ensure reasonable defaults
	if width < 20 {
		width = 80
	}
	if height < 10 {
		height = 24
	}
	
	// Create login model (same as SSH)
	loginModel := terminal.NewLoginModel(db, userService, "", "")
	
	bridge := &BubbleTeaBridge{
		model:       loginModel,
		conn:        conn,
		db:          db,
		userService: userService,
		done:        make(chan struct{}),
		msgChan:     make(chan tea.Msg, 100),
		width:       width,
		height:      height,
		renderMode:  RenderModeFullScreen,
		lastView:    "",
	}
	
	// Initialize model
	initCmd := bridge.model.Init()
	if initCmd != nil {
		go func() {
			if msg := initCmd(); msg != nil {
				bridge.msgChan <- msg
			}
		}()
	}
	
	// Send initial window size
	bridge.msgChan <- tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	}

	// Start goroutine to process messages and send output
	go bridge.processMessages()

	return bridge, nil
}

// isLoginModel checks if current model is the login screen
func (b *BubbleTeaBridge) isLoginModel() bool {
	_, ok := b.model.(*terminal.LoginModel)
	return ok
}

// getShellModel returns the shell model if current model is shell
func (b *BubbleTeaBridge) getShellModel() *terminal.ShellModel {
	shell, ok := b.model.(*terminal.ShellModel)
	if ok {
		return shell
	}
	return nil
}

// prepareFullScreenOutput prepares output for full-screen mode (login)
// Clears screen completely before rendering - used for centered UI
func prepareFullScreenOutput(view string) string {
	var sb strings.Builder
	
	sb.WriteString(clearScreen)
	sb.WriteString(cursorHome)
	sb.WriteString(hideCursor)
	
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		sb.WriteString(line)
		sb.WriteString(clearToEOL)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	return sb.String()
}

// prepareIncrementalOutput prepares output for incremental mode (shell)
// If startRow > 0, positions cursor at that row first
// If content starts with \n, we're appending after a command (move to next line first)
func prepareIncrementalOutput(content string, startRow int) string {
	var sb strings.Builder
	
	// Position cursor at specific row if requested
	if startRow > 0 {
		sb.WriteString(fmt.Sprintf("\x1b[%d;1H", startRow))
	} else if strings.HasPrefix(content, "\n") {
		// Appending after a command - move to end of current line, then next line
		sb.WriteString("\x1b[999C") // Move cursor far right (stops at end of content)
		sb.WriteString("\r\n")      // Move to next line
		content = content[1:]       // Remove the leading \n (we just handled it)
	}
	
	// Convert \n to \r\n for proper terminal rendering
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	sb.WriteString(hideCursor)
	return sb.String()
}

// preparePromptUpdate prepares a prompt-only update at a specific row
func preparePromptUpdate(promptLine string, row int) string {
	var sb strings.Builder
	// Position at the prompt row
	sb.WriteString(fmt.Sprintf("\x1b[%d;1H", row))
	sb.WriteString(clearLine)   // Clear the entire line
	sb.WriteString(promptLine)  // Write new prompt
	sb.WriteString(hideCursor)
	return sb.String()
}

// prepareClearScreen prepares a clear screen with content
// Positions at startRow if > 0
func prepareClearScreen(content string, startRow int) string {
	var sb strings.Builder
	sb.WriteString(clearScreen)
	
	// Position at specific row or home
	if startRow > 0 {
		sb.WriteString(fmt.Sprintf("\x1b[%d;1H", startRow))
	} else {
		sb.WriteString(cursorHome)
	}
	
	// Convert \n to \r\n
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	sb.WriteString(hideCursor)
	return sb.String()
}

// prepareShellOutput prepares shell view for full-screen redraw
// Clears screen, positions content, handles line conversion
func prepareShellOutput(view string, height int) string {
	var sb strings.Builder
	
	// Clear screen and go home
	sb.WriteString(clearScreen)
	sb.WriteString(cursorHome)
	sb.WriteString(hideCursor)
	
	// Convert \n to \r\n for proper terminal rendering
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		sb.WriteString(line)
		sb.WriteString(clearToEOL)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	return sb.String()
}

// prepareShellOutputWithClear prepares shell view and clears scrollback buffer
// Used after 'clear' command to truly clear everything
func prepareShellOutputWithClear(view string, height int) string {
	var sb strings.Builder
	
	// Clear screen AND scrollback buffer, then go home
	sb.WriteString(clearAll)
	sb.WriteString(cursorHome)
	sb.WriteString(hideCursor)
	
	// Convert \n to \r\n for proper terminal rendering
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		sb.WriteString(line)
		sb.WriteString(clearToEOL)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	return sb.String()
}

// processMessages processes Bubble Tea messages and sends View() output over WebSocket
func (b *BubbleTeaBridge) processMessages() {
	defer b.closeDone()

	// Start in alternate screen for login (no scrollback needed)
	initMsg := OutputMessage{
		Type: MessageTypeOutput,
		Data: enterAltScreen + hideCursor,
	}
	if err := b.conn.WriteJSON(initMsg); err != nil {
		return
	}

	// Send initial view
	currentView := b.model.View()
	output := prepareFullScreenOutput(currentView)
	msg := OutputMessage{
		Type: MessageTypeOutput,
		Data: output,
	}
	if err := b.conn.WriteJSON(msg); err != nil {
		return
	}
	b.lastView = currentView
	wasLogin := true

	for {
		select {
		case <-b.done:
			// Clean up on close
			exitMsg := OutputMessage{
				Type: MessageTypeOutput,
				Data: exitAltScreen + showCursor,
			}
			b.conn.WriteJSON(exitMsg)
			return
			
		case teaMsg := <-b.msgChan:
			// Handle resize
			if sizeMsg, ok := teaMsg.(tea.WindowSizeMsg); ok {
				b.width = sizeMsg.Width
				b.height = sizeMsg.Height
			}
			
			// Update model
			var cmd tea.Cmd
			b.model, cmd = b.model.Update(teaMsg)
			
			// Execute any returned command
			if cmd != nil {
				go func(c tea.Cmd) {
					if msg := c(); msg != nil {
						select {
						case b.msgChan <- msg:
						case <-b.done:
							return
						}
					}
				}(cmd)
			}
			
			// Check if we transitioned from login to shell
			isLogin := b.isLoginModel()
			if wasLogin && !isLogin {
				// Transition: exit alternate screen to enable scrollback
				transitionMsg := OutputMessage{
					Type: MessageTypeOutput,
					Data: exitAltScreen + clearScreen + cursorHome,
				}
				if err := b.conn.WriteJSON(transitionMsg); err != nil {
					return
				}
				b.renderMode = RenderModeIncremental
				b.lastView = "" // Force redraw
			}
			wasLogin = isLogin
			
			// Render based on mode
			if b.renderMode == RenderModeFullScreen {
				// Login screen - full screen render
				currentView := b.model.View()
				
				// Skip if unchanged
				if currentView == b.lastView {
					continue
				}
				
				output := prepareFullScreenOutput(currentView)
				msg := OutputMessage{
					Type: MessageTypeOutput,
					Data: output,
				}
				if err := b.conn.WriteJSON(msg); err != nil {
					return
				}
				b.lastView = currentView
			} else {
				// Shell mode - use full screen redraw for reliability
				shell := b.getShellModel()
				
				// Check if we need to clear scrollback (after clear command)
				needsClearScrollback := false
				if shell != nil {
					needsClearScrollback = shell.NeedsClearScrollback()
				}
				
				currentView := b.model.View()
				
				// Skip if unchanged (unless we need to clear scrollback)
				if currentView == b.lastView && !needsClearScrollback {
					continue
				}
				
				var output string
				if needsClearScrollback {
					// Clear everything including scrollback
					output = prepareShellOutputWithClear(currentView, b.height)
				} else {
					output = prepareShellOutput(currentView, b.height)
				}
				
				msg := OutputMessage{
					Type: MessageTypeOutput,
					Data: output,
				}
				if err := b.conn.WriteJSON(msg); err != nil {
					return
				}
				b.lastView = currentView
			}
		}
	}
}

// HandleInput handles input messages from the WebSocket client
func (b *BubbleTeaBridge) HandleInput(msg InputMessage) error {
	keyMsg := convertToKeyMsg(msg)
	
	select {
	case b.msgChan <- keyMsg:
	default:
		// Channel full, drop message
	}
	
	return nil
}

// HandleResize handles resize messages from the WebSocket client
func (b *BubbleTeaBridge) HandleResize(msg ResizeMessage) error {
	select {
	case b.msgChan <- tea.WindowSizeMsg{
		Width:  msg.Width,
		Height: msg.Height,
	}:
	default:
		// Channel full, drop message
	}
	
	return nil
}

// closeDone safely closes the done channel
func (b *BubbleTeaBridge) closeDone() {
	b.closeOnce.Do(func() {
		close(b.done)
	})
}

// Close closes the bridge
func (b *BubbleTeaBridge) Close() {
	b.closeDone()
}

// convertToKeyMsg converts a WebSocket InputMessage to a Bubble Tea KeyMsg
func convertToKeyMsg(msg InputMessage) tea.KeyMsg {
	var keyType tea.KeyType
	var runes []rune
	
	switch msg.Key {
	case "Enter":
		keyType = tea.KeyEnter
	case "Backspace":
		keyType = tea.KeyBackspace
	case "Tab":
		keyType = tea.KeyTab
	case "Space":
		keyType = tea.KeySpace
	case "Up":
		keyType = tea.KeyUp
	case "Down":
		keyType = tea.KeyDown
	case "Left":
		keyType = tea.KeyLeft
	case "Right":
		keyType = tea.KeyRight
	case "Esc", "Escape":
		keyType = tea.KeyEsc
	case "Ctrl+c":
		keyType = tea.KeyCtrlC
	case "Ctrl+s":
		keyType = tea.KeyCtrlS
	case "Ctrl+q":
		keyType = tea.KeyCtrlQ
	case "Ctrl+l":
		keyType = tea.KeyCtrlL
	default:
		if msg.Char != "" {
			runes = []rune(msg.Char)
			keyType = tea.KeyRunes
		} else {
			runes = []rune(msg.Key)
			keyType = tea.KeyRunes
		}
	}
	
	alt := false
	for _, mod := range msg.Modifiers {
		if mod == "Alt" {
			alt = true
		}
	}
	
	return tea.KeyMsg{
		Type:  keyType,
		Runes: runes,
		Alt:   alt,
	}
}
