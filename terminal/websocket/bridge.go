package websocket

import (
	"fmt"
	"strings"
	"sync"
	"terminal-sh/database"
	"terminal-sh/services"
	"terminal-sh/terminal"
	"time"

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
	clearScreen     = "\x1b[2J"        // Clear visible screen
	clearScrollback = "\x1b[3J"        // Clear scrollback buffer
	clearAll        = "\x1b[2J\x1b[3J" // Clear screen + scrollback
	cursorHome      = "\x1b[H"
	clearLine       = "\x1b[2K" // Clear entire line
	clearToEOL      = "\x1b[K"  // Clear to end of line
)

// RenderMode determines how content is rendered
type RenderMode int

const (
	RenderModeFullScreen  RenderMode = iota // Login: alternate screen, clear each render
	RenderModeIncremental                   // Shell: incremental output, scrollback enabled
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
	lastContent string // Track last content (without prompt) to detect prompt-only changes
	promptRow   int    // Track which row the prompt is on for updates

	gradientTickerRunning bool
}

// NewBubbleTeaBridge creates a new bridge between Bubble Tea and WebSocket
func NewBubbleTeaBridge(conn *websocket.Conn, db *database.Database, userService *services.UserService, chatService *services.ChatService, width, height int) (*BubbleTeaBridge, error) {
	// Ensure reasonable defaults
	if width < 20 {
		width = 80
	}
	if height < 10 {
		height = 24
	}
	
	// Create login model (same as SSH)
	loginModel := terminal.NewLoginModel(db, userService, chatService, "", "")
	
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

// isChatModel checks if current model is the chat screen
func (b *BubbleTeaBridge) isChatModel() bool {
	_, ok := b.model.(*terminal.ChatModel)
	return ok
}

// getChatModel returns the chat model if current model is chat
func (b *BubbleTeaBridge) getChatModel() *terminal.ChatModel {
	chat, ok := b.model.(*terminal.ChatModel)
	if ok {
		return chat
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

// prepareChatOutput clears screen + scrollback for chat rendering to avoid growth
func prepareChatOutput(view string) string {
	var sb strings.Builder

	// Clear screen and scrollback, go home
	sb.WriteString(clearAll)
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

// preparePromptUpdate prepares a prompt-only update
// Uses cursor save/restore to update the prompt line in place without affecting scrollback
func preparePromptUpdate(promptLine string, row int) string {
	var sb strings.Builder
	// Go to beginning of current line and clear it
	sb.WriteString("\r")       // Carriage return - go to column 1
	sb.WriteString(clearLine)  // Clear the entire line
	sb.WriteString(promptLine) // Write new prompt
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
	wasChat := false

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
			
			// Check if we transitioned models
			isLogin := b.isLoginModel()
			isChat := b.isChatModel()

			if wasLogin && !isLogin && !isChat {
				// Transition: login -> shell: exit alternate screen to enable scrollback
				transitionMsg := OutputMessage{
					Type: MessageTypeOutput,
					Data: exitAltScreen + clearScreen + cursorHome,
				}
				if err := b.conn.WriteJSON(transitionMsg); err != nil {
					return
				}
				b.renderMode = RenderModeIncremental
				b.lastView = ""    // Force redraw
				b.lastContent = "" // Reset content tracking
				// Note: lastContent will be set after first render
			}

			if !wasLogin && isLogin {
				// Transition: shell -> login: enter alternate screen for login
				// Stop gradient ticker if running
				b.gradientTickerRunning = false
				
				transitionMsg := OutputMessage{
					Type: MessageTypeOutput,
					Data: enterAltScreen + clearScreen + cursorHome,
				}
				if err := b.conn.WriteJSON(transitionMsg); err != nil {
					return
				}
				b.renderMode = RenderModeFullScreen
				b.lastView = "" // Force redraw
			}

			// Update render mode based on current model
			if isChat {
				b.renderMode = RenderModeFullScreen // Chat uses full screen rendering
				b.lastView = ""                     // Force full redraw to avoid appended output

				// If just entered chat mode, start the message loop
				if !wasChat {
					chatModel := b.getChatModel()
					if chatModel != nil {
						chatModel.StartMessageLoop(b.msgChan)
					}
				}
			}

			// Transition: chat -> shell: reset to incremental mode
			if wasChat && !isChat && !isLogin {
				transitionMsg := OutputMessage{
					Type: MessageTypeOutput,
					Data: clearScreen + cursorHome,
				}
				if err := b.conn.WriteJSON(transitionMsg); err != nil {
					return
				}
				b.renderMode = RenderModeIncremental
				b.lastView = ""    // Force redraw
				b.lastContent = "" // Reset content tracking
			}

			wasLogin = isLogin
			wasChat = isChat
			
			// If we're in shell and the gradient animation is running, start a bridge-side ticker
			if shell := b.getShellModel(); shell != nil {
				if shell.IsGradientAnimating() && !b.gradientTickerRunning {
					b.gradientTickerRunning = true
					go b.runGradientTicker(shell)
				}
			}
			
			// Render based on mode
			if b.renderMode == RenderModeFullScreen {
				// Login or Chat screen - full screen render
				currentView := b.model.View()
				
				// Skip if unchanged
				if currentView == b.lastView {
					continue
				}
				
				isChat := b.isChatModel()
				var output string
				if isChat {
					output = prepareChatOutput(currentView)
				} else {
					output = prepareFullScreenOutput(currentView)
				}
				msg := OutputMessage{
					Type: MessageTypeOutput,
					Data: output,
				}
				if err := b.conn.WriteJSON(msg); err != nil {
					return
				}
				b.lastView = currentView
		} else {
			// Shell mode - always use clearAll to prevent scrollback accumulation
			shell := b.getShellModel()
			if shell == nil {
				continue
			}
			
			// Consume any pending clear flags
			shell.NeedsClearScrollback()
			
			// Get current view
			currentView := b.model.View()
			
			// Skip if unchanged
			if currentView == b.lastView {
				continue
			}
			
			// Always clear screen+scrollback and redraw to prevent line accumulation
			// This sacrifices scrollback history but ensures clean rendering
			var data strings.Builder
			data.WriteString(clearAll)
			data.WriteString(cursorHome)
			data.WriteString(hideCursor)
			
			lines := strings.Split(currentView, "\n")
			for i, line := range lines {
				data.WriteString(line)
				data.WriteString(clearToEOL)
				if i < len(lines)-1 {
					data.WriteString("\r\n")
				}
			}
			
			b.lastView = currentView
			
			msg := OutputMessage{
				Type: MessageTypeOutput,
				Data: data.String(),
			}
			if err := b.conn.WriteJSON(msg); err != nil {
				return
			}
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

// HandleMouse handles mouse messages from the WebSocket client
func (b *BubbleTeaBridge) HandleMouse(msg MouseMessage) error {
	var button tea.MouseButton
	switch msg.Button {
	case "wheelUp":
		button = tea.MouseButtonWheelUp
	case "wheelDown":
		button = tea.MouseButtonWheelDown
	default:
		return nil // Ignore other mouse events for now
	}
	
	mouseMsg := tea.MouseMsg{
		X:      msg.X,
		Y:      msg.Y,
		Button: button,
		Action: tea.MouseActionPress,
	}
	
	select {
	case b.msgChan <- mouseMsg:
	default:
		// Channel full, drop message
	}
	
	return nil
}

// HandlePaste handles paste messages from the WebSocket client
func (b *BubbleTeaBridge) HandlePaste(msg PasteMessage) error {
	// Send paste as a special message type that the shell can handle
	// Use terminal.PasteTextMsg which is defined in the terminal package
	pasteMsg := terminal.PasteTextMsg{Text: msg.Text}
	
	select {
	case b.msgChan <- pasteMsg:
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

// runGradientTicker injects gradient tick messages while the shell welcome animation is active.
// This ensures web sessions (which rely on bridge-driven redraws) animate like SSH.
func (b *BubbleTeaBridge) runGradientTicker(shell *terminal.ShellModel) {
	ticker := time.NewTicker(terminal.GradientFrameDelay)
	defer ticker.Stop()
	defer func() { b.gradientTickerRunning = false }()

	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			// Check if we're still in shell mode
			currentShell := b.getShellModel()
			if currentShell == nil || currentShell != shell {
				// Model changed, stop ticker
				return
			}
			if !shell.IsGradientAnimating() {
				return
			}
			select {
			case b.msgChan <- terminal.GradientTickMsg{}:
			case <-b.done:
				return
			default:
				// Drop tick if channel is full; next tick will try again.
			}
		}
	}
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
		// Ctrl+C is now used for copy in terminals, not for exit
		// We'll handle it as a regular key if needed, but typically it's handled client-side
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
