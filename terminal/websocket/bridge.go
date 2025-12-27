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
	// Alternate screen buffer (used for login - no scrollback)
	enterAltScreen = "\x1b[?1049h"
	exitAltScreen  = "\x1b[?1049l"
	
	// Cursor control
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	
	// Screen control
	clearScreen = "\x1b[2J"
	cursorHome  = "\x1b[H"
	clearLine   = "\x1b[K"
	clearToEnd  = "\x1b[J"
)

// RenderMode determines how content is rendered
type RenderMode int

const (
	RenderModeFullScreen RenderMode = iota // Login: alternate screen, clear each render
	RenderModeShell                        // Shell: normal buffer, scrollback enabled
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
		sb.WriteString(clearLine)
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	return sb.String()
}

// prepareShellOutput prepares output for shell mode
// Writes content without clearing - enables natural scrolling
// Each render overwrites from home, excess lines scroll into scrollback
func prepareShellOutput(view string, height int) string {
	var sb strings.Builder
	
	// Position at home (no clear - preserves scrollback)
	sb.WriteString(cursorHome)
	sb.WriteString(hideCursor)
	
	lines := strings.Split(view, "\n")
	
	// Write all lines, clearing each line as we go
	for i, line := range lines {
		sb.WriteString(line)
		sb.WriteString(clearLine) // Clear rest of this line
		if i < len(lines)-1 {
			sb.WriteString("\r\n")
		}
	}
	
	// If content is shorter than screen, clear remaining lines
	if len(lines) < height {
		for i := len(lines); i < height; i++ {
			sb.WriteString("\r\n")
			sb.WriteString(clearLine)
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
				b.renderMode = RenderModeShell
				b.lastView = "" // Force redraw
			}
			wasLogin = isLogin
			
			// Get current view
			currentView := b.model.View()
			
			// Skip if unchanged (optimization)
			if currentView == b.lastView {
				continue
			}
			
			// Prepare output based on mode
			var output string
			if b.renderMode == RenderModeFullScreen {
				output = prepareFullScreenOutput(currentView)
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
