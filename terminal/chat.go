package terminal

import (
	"fmt"
	"strings"
	"time"

	"terminal-sh/models"
	"terminal-sh/services"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

// Use time package to avoid unused import error
var _ = time.Time{}

// ChatModel handles the chat UI
type ChatModel struct {
	parent        *ShellModel
	chatService   *services.ChatService
	user          *models.User
	activeRoomID  uuid.UUID
	rooms         []*models.ChatRoom
	roomViewports map[uuid.UUID]*viewport.Model
	roomMessages  map[uuid.UUID][]models.ChatMessage
	tabIndex      int
	textInput     textinput.Model
	width         int
	height        int
	msgChan       chan models.ChatMessage
	sessionID     uuid.UUID
	splitMode     bool
}

// ChatMessageMsg is sent when a new message arrives
type ChatMessageMsg struct {
	Message models.ChatMessage
}

// chatExitMsg signals exiting chat back to shell
type chatExitMsg struct{}

// NewChatModel creates a new chat model
func NewChatModel(parent *ShellModel, chatService *services.ChatService, user *models.User, sessionID uuid.UUID, width, height int, splitMode bool) *ChatModel {
	// Register session with chat service
	msgChan := chatService.RegisterSession(sessionID, user.ID)

	// Get user's rooms
	rooms, _ := chatService.GetRoomsForUser(user.ID)

	// Auto-join #public if not in any rooms
	if len(rooms) == 0 {
		publicRoom, err := chatService.GetRoomByName("#public")
		if err == nil {
			chatService.JoinRoom(publicRoom.ID, user.ID, "")
			rooms, _ = chatService.GetRoomsForUser(user.ID)
		}
	}

	activeRoomID := uuid.Nil
	if len(rooms) > 0 {
		activeRoomID = rooms[0].ID
	}

	// Initialize text input
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type a message or /help for commands..."
	ti.CharLimit = 0
	ti.Width = width
	ti.Focus()

	model := &ChatModel{
		parent:        parent,
		chatService:   chatService,
		user:          user,
		activeRoomID:  activeRoomID,
		rooms:         rooms,
		roomViewports: make(map[uuid.UUID]*viewport.Model),
		roomMessages:  make(map[uuid.UUID][]models.ChatMessage),
		tabIndex:      0,
		textInput:     ti,
		width:         width,
		height:        height,
		msgChan:       msgChan,
		sessionID:     sessionID,
		splitMode:     splitMode,
	}

	// Initialize viewports and load messages for each room
	for _, room := range rooms {
		vp := viewport.New(width, height-3) // Reserve space for tabs and input
		model.roomViewports[room.ID] = &vp

		// Load recent messages
		messages, _ := chatService.GetRecentMessages(room.ID, 100)
		model.roomMessages[room.ID] = messages
		model.updateViewportContent(room.ID)
	}

	return model
}

// Init initializes the chat model
func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		m.textInput.Focus(),
		m.listenForMessages(),
	)
}

// listenForMessages listens to the message channel
func (m *ChatModel) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.msgChan
		if !ok {
			return nil
		}
		return ChatMessageMsg{Message: msg}
	}
}

// Update handles messages
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width

		// Update all viewports
		for roomID := range m.roomViewports {
			vp := m.roomViewports[roomID]
			vp.Width = msg.Width
			vp.Height = msg.Height - 3 // Reserve space for tabs and input
			m.updateViewportContent(roomID)
		}

		return m, nil

	case ChatMessageMsg:
		// New message arrived
		msgRoomID := msg.Message.RoomID

		// Add to messages
		if m.roomMessages[msgRoomID] == nil {
			m.roomMessages[msgRoomID] = make([]models.ChatMessage, 0)
		}
		m.roomMessages[msgRoomID] = append(m.roomMessages[msgRoomID], msg.Message)

		// Trim if too many
		if len(m.roomMessages[msgRoomID]) > 100 {
			m.roomMessages[msgRoomID] = m.roomMessages[msgRoomID][len(m.roomMessages[msgRoomID])-100:]
		}

		// Update viewport if this is the active room
		if msgRoomID == m.activeRoomID {
			m.updateViewportContent(msgRoomID)
		}

		// Continue listening
		cmds = append(cmds, m.listenForMessages())

		return m, tea.Batch(cmds...)

	case chatExitMsg:
		return m.parent, nil

	case tea.KeyMsg:
		// Handle special keys
		switch msg.String() {
		case "ctrl+c", "esc":
			// Exit chat mode
			m.chatService.UnregisterSession(m.sessionID)
			return m.parent, nil

		case "left":
			// Switch to previous tab
			if len(m.rooms) > 0 {
				m.tabIndex--
				if m.tabIndex < 0 {
					m.tabIndex = len(m.rooms) - 1
				}
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "right":
			// Switch to next tab
			if len(m.rooms) > 0 {
				m.tabIndex++
				if m.tabIndex >= len(m.rooms) {
					m.tabIndex = 0
				}
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "tab":
			// Cycle through tabs
			if len(m.rooms) > 0 {
				m.tabIndex = (m.tabIndex + 1) % len(m.rooms)
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Switch to tab by number
			tabNum := int(msg.String()[0] - '1')
			if tabNum >= 0 && tabNum < len(m.rooms) {
				m.tabIndex = tabNum
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "up", "down":
			// Scroll viewport
			if vp, ok := m.roomViewports[m.activeRoomID]; ok {
				var cmd tea.Cmd
				*vp, cmd = vp.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case "enter":
			// Send message or execute command
			input := m.textInput.Value()
			m.textInput.SetValue("")

			if input == "" {
				return m, nil
			}

			// Check if it's a command
			if strings.HasPrefix(input, "/") {
				return m, m.handleCommand(input)
			}

			// Send message to active room
			if m.activeRoomID != uuid.Nil {
				err := m.chatService.SendMessage(m.activeRoomID, m.user.ID, m.user.Username, input)
				if err != nil {
					// Error will be shown in next update
					return m, nil
				}
			}

			return m, nil

		default:
			// Handle text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleCommand processes chat commands
func (m *ChatModel) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/join":
		if len(args) < 1 {
			return nil
		}
		roomName := args[0]
		password := ""
		if len(args) > 1 {
			password = args[1]
		}

		// Get or create room
		room, err := m.chatService.GetRoomByName(roomName)
		if err != nil {
			room, err = m.chatService.CreateRoom(roomName, "public", "", m.user.ID)
			if err != nil {
				return nil
			}
		}

		// Join room
		err = m.chatService.JoinRoom(room.ID, m.user.ID, password)
		if err == nil {
			// Add to rooms list
			m.rooms = append(m.rooms, room)
			m.roomMessages[room.ID] = make([]models.ChatMessage, 0)
			vp := viewport.New(m.width, m.height-3)
			m.roomViewports[room.ID] = &vp

			// Load messages
			messages, _ := m.chatService.GetRecentMessages(room.ID, 100)
			m.roomMessages[room.ID] = messages
			m.updateViewportContent(room.ID)

			// Switch to new room
			m.tabIndex = len(m.rooms) - 1
			m.activeRoomID = room.ID
		}

	case "/leave":
		if len(args) < 1 {
			return nil
		}
		roomName := args[0]

		// Find room
		var roomIndex int = -1
		var roomID uuid.UUID
		for i, room := range m.rooms {
			if room.Name == roomName {
				roomIndex = i
				roomID = room.ID
				break
			}
		}

		if roomIndex >= 0 {
			// Leave room
			m.chatService.LeaveRoom(roomID, m.user.ID)

			// Remove from lists
			m.rooms = append(m.rooms[:roomIndex], m.rooms[roomIndex+1:]...)
			delete(m.roomMessages, roomID)
			delete(m.roomViewports, roomID)

			// Switch to another room if needed
			if len(m.rooms) > 0 {
				if m.tabIndex >= len(m.rooms) {
					m.tabIndex = len(m.rooms) - 1
				}
				m.activeRoomID = m.rooms[m.tabIndex].ID
			} else {
				m.activeRoomID = uuid.Nil
			}
		}

	case "/exit", "/quit":
		// Exit chat mode
		m.chatService.UnregisterSession(m.sessionID)
		return func() tea.Msg {
			return chatExitMsg{}
		}

	case "/help":
		// Show help - add as a system message to current room
		helpText := `Chat Commands:
  /join <room> [password] - Join or create a room
  /leave <room>           - Leave a room
  /rooms                  - List your current rooms
  /who                    - List users in current room
  /exit or /quit          - Exit chat mode
  
Navigation:
  ←/→ or Tab              - Switch between rooms
  ↑/↓                     - Scroll message history
  1-9                     - Jump to room by number
  Esc or Ctrl+C           - Exit chat mode`

		// Add help as a fake message in current room
		if m.activeRoomID != uuid.Nil {
			helpMsg := models.ChatMessage{
				ID:        uuid.New(),
				RoomID:    m.activeRoomID,
				Username:  "system",
				Content:   helpText,
				CreatedAt: time.Now(),
			}
			m.roomMessages[m.activeRoomID] = append(m.roomMessages[m.activeRoomID], helpMsg)
			m.updateViewportContent(m.activeRoomID)
		}

	case "/rooms":
		// List rooms user is in
		var roomList strings.Builder
		roomList.WriteString("Your rooms: ")
		for i, room := range m.rooms {
			if i > 0 {
				roomList.WriteString(", ")
			}
			roomList.WriteString(room.Name)
		}

		if m.activeRoomID != uuid.Nil {
			sysMsg := models.ChatMessage{
				ID:        uuid.New(),
				RoomID:    m.activeRoomID,
				Username:  "system",
				Content:   roomList.String(),
				CreatedAt: time.Now(),
			}
			m.roomMessages[m.activeRoomID] = append(m.roomMessages[m.activeRoomID], sysMsg)
			m.updateViewportContent(m.activeRoomID)
		}

	case "/who":
		// List users in current room
		if m.activeRoomID != uuid.Nil {
			members, err := m.chatService.GetRoomMembers(m.activeRoomID)
			var content string
			if err != nil {
				content = "Error getting room members"
			} else if len(members) == 0 {
				content = "No users in this room"
			} else {
				var names []string
				for _, member := range members {
					names = append(names, member.Username)
				}
				content = "Users in room: " + strings.Join(names, ", ")
			}

			sysMsg := models.ChatMessage{
				ID:        uuid.New(),
				RoomID:    m.activeRoomID,
				Username:  "system",
				Content:   content,
				CreatedAt: time.Now(),
			}
			m.roomMessages[m.activeRoomID] = append(m.roomMessages[m.activeRoomID], sysMsg)
			m.updateViewportContent(m.activeRoomID)
		}
	}

	return nil
}

// updateViewportContent updates the viewport content for a room
func (m *ChatModel) updateViewportContent(roomID uuid.UUID) {
	vp, ok := m.roomViewports[roomID]
	if !ok {
		return
	}

	messages := m.roomMessages[roomID]
	var content strings.Builder

	for _, msg := range messages {
		timestamp := msg.CreatedAt.Format("15:04:05") // Uses time package
		line := fmt.Sprintf("[%s] <%s> %s", timestamp, msg.Username, msg.Content)
		content.WriteString(line)
		content.WriteString("\n")
	}

	vp.SetContent(content.String())
	vp.GotoBottom()
}

// View renders the chat UI
func (m *ChatModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Render tabs
	if len(m.rooms) > 0 {
		tabStyle := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)
		activeTabStyle := tabStyle.Copy().Bold(true).Foreground(lipgloss.Color("205"))
		inactiveTabStyle := tabStyle.Copy().Foreground(lipgloss.Color("240"))

		for i, room := range m.rooms {
			if i == m.tabIndex {
				sb.WriteString(activeTabStyle.Render(room.Name))
			} else {
				sb.WriteString(inactiveTabStyle.Render(room.Name))
			}
		}
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", m.width))
		sb.WriteString("\n")
	}

	// Render viewport
	if vp, ok := m.roomViewports[m.activeRoomID]; ok {
		sb.WriteString(vp.View())
	} else {
		sb.WriteString("No room selected. Use /join <room> to join a room.\n")
	}

	// Render input
	sb.WriteString("\n")
	sb.WriteString(m.textInput.View())

	return sb.String()
}

// Close cleans up the chat model
func (m *ChatModel) Close() {
	m.chatService.UnregisterSession(m.sessionID)
}

// StartMessageLoop starts the message listener and forwards messages to the given channel
// This is used by the WebSocket bridge since it can't run Init() commands properly
func (m *ChatModel) StartMessageLoop(bridgeChan chan tea.Msg) {
	go func() {
		for {
			msg, ok := <-m.msgChan
			if !ok {
				return // Channel closed
			}
			// Forward to bridge channel as ChatMessageMsg
			select {
			case bridgeChan <- ChatMessageMsg{Message: msg}:
			default:
				// Bridge channel full or closed, stop listening
				return
			}
		}
	}()
}
