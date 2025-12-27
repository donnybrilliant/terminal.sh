package websocket

import (
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
	// Alternate screen buffer
	enterAltScreen = "\x1b[?1049h"
	exitAltScreen  = "\x1b[?1049l"
	
	// Cursor control
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	
	// Screen control
	clearScreen = "\x1b[2J"
	cursorHome  = "\x1b[H"
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
	}
	
	// Initialize model
	initCmd := bridge.model.Init()
	if initCmd != nil {
		// Execute init command
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

// prepareOutput converts Bubble Tea View() output for xterm.js
// - Clears the screen completely
// - Converts \n to \r\n for proper line breaks
// - Pads lines to full width to prevent ghosting
func prepareOutput(view string, width, height int) string {
	var sb strings.Builder
	
	// Clear screen and move cursor to home position
	sb.WriteString(clearScreen)
	sb.WriteString(cursorHome)
	sb.WriteString(hideCursor)
	
	// Process the view line by line
	lines := strings.Split(view, "\n")
	
	for i, line := range lines {
		// Write the line
		sb.WriteString(line)
		
		// Clear to end of line (handles any leftover content on this line)
		sb.WriteString("\x1b[K")
		
		// Add carriage return + newline (except for last line)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	return sb.String()
}

// processMessages processes Bubble Tea messages and sends View() output over WebSocket
func (b *BubbleTeaBridge) processMessages() {
	defer b.closeDone()

	lastView := ""
	
	// Send initial setup: enter alternate screen + hide cursor
	initMsg := OutputMessage{
		Type: MessageTypeOutput,
		Data: enterAltScreen + hideCursor,
	}
	if err := b.conn.WriteJSON(initMsg); err != nil {
		return
	}

	// Helper function to send view
	sendView := func(view string) {
		// Only send if view has changed
		if view == lastView {
			return
		}
		
		// Prepare output for xterm
		output := prepareOutput(view, b.width, b.height)
		
		msg := OutputMessage{
			Type: MessageTypeOutput,
			Data: output,
		}
		if err := b.conn.WriteJSON(msg); err != nil {
			return // Connection closed
		}
		lastView = view
	}

	// Send initial view
	currentView := b.model.View()
	sendView(currentView)

	for {
		select {
		case <-b.done:
			// Send exit alternate screen on close
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
			
			// Update model with message
			var cmd tea.Cmd
			b.model, cmd = b.model.Update(teaMsg)
			
			// Execute command if any (commands can return messages)
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
			
			// Always send the current view after any update
			// This is necessary for forms and other interactive components
			currentView := b.model.View()
			
			// Force send on every update (remove duplicate check for now)
			// This ensures the UI stays in sync
			output := prepareOutput(currentView, b.width, b.height)
			msg := OutputMessage{
				Type: MessageTypeOutput,
				Data: output,
			}
			if err := b.conn.WriteJSON(msg); err != nil {
				return // Connection closed
			}
			lastView = currentView
		}
	}
}

// HandleInput handles input messages from the WebSocket client
func (b *BubbleTeaBridge) HandleInput(msg InputMessage) error {
	// Convert WebSocket input message to Bubble Tea KeyMsg
	keyMsg := convertToKeyMsg(msg)
	
	// Send to message channel
	select {
	case b.msgChan <- keyMsg:
	default:
		// Channel full, drop message
	}
	
	return nil
}

// HandleResize handles resize messages from the WebSocket client
func (b *BubbleTeaBridge) HandleResize(msg ResizeMessage) error {
	// Send WindowSizeMsg to message channel
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

// closeDone safely closes the done channel (only once)
func (b *BubbleTeaBridge) closeDone() {
	b.closeOnce.Do(func() {
		close(b.done)
	})
}

// Close closes the bridge and cleans up
func (b *BubbleTeaBridge) Close() {
	b.closeDone()
}

// convertToKeyMsg converts a WebSocket InputMessage to a Bubble Tea KeyMsg
func convertToKeyMsg(msg InputMessage) tea.KeyMsg {
	var keyType tea.KeyType
	var runes []rune
	
	// Handle special keys
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
		// Regular character - use the char field
		if msg.Char != "" {
			runes = []rune(msg.Char)
			keyType = tea.KeyRunes
		} else {
			// Fallback: treat as runes
			runes = []rune(msg.Key)
			keyType = tea.KeyRunes
		}
	}
	
	// Check for Alt modifier (Ctrl is encoded in key type like KeyCtrlC)
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
